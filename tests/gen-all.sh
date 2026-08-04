#!/usr/bin/env bash
#
# gen-all.sh - run every tests/gen-*.sh generator against ONE shared `mec`, in
# parallel, and report one verdict.
#
# WHY THIS EXISTS
#
# `tests/gen-*.sh --check` is the gate that catches a stale checked-in .ll, so it
# is mandatory after any runtime.c / *-rt.metajs change - which means it is in the
# inner loop, run dozens of times per task. Serially it took ~25 s on this
# machine. Two things were paid for that nobody needed:
#
#   1. Fifteen `go build`s of the same unchanged source. Measured: only the FIRST
#      is expensive (~2.4 s cold); the other fourteen hit Go's build cache and
#      cost ~0.07 s each. So this is ~1 s of the 25 s on a warm cache and ~3.5 s
#      on a cold one - real, but not the main event.
#   2. Serialisation. That IS the main event. The fifteen compiles are 0.8-2.7 s
#      each and 20.6 s in total, on a 16-core machine, one at a time.
#
# So: build once, export MEC_BIN (each generator picks it up and skips its own
# build; see the MEC_BIN_ABS block in any of them), and fan out. Measured on this
# 16-core machine, warm cache, clean tree: 25.3 s serial -> 3.3 s. Verified equal:
# a parallel write and a serial write produce byte-identical .ll for all fifteen,
# and a deliberately corrupted .ll still FAILS --check here and in each script.
#
# CAVEAT ON MEC_BIN. It is a trust delegation: whatever binary it names is what
# the checked-in .ll is compared against. This script always builds a fresh one,
# so that is sound here. Exporting a STALE MEC_BIN into a hand-run generator
# would check the .ll against an old compiler - so don't. A wrong binary fails
# loudly rather than silently passing (the >100-line floor in every generator
# catches it), but an *old* mec would not.
#
# WHY PARALLEL IS SAFE HERE. Each generator reads the grammars and its own one
# source file, writes its own private mktemp files, and - only outside --check -
# writes its own distinct languages/lib/<lang>.ll. No generator reads another
# generator's output: the thirteen handle-IR ones compile a .metajs with
# `-rt-lib`, which emits a module and links nothing, and the three C-floor ones
# (runtime, bash, batch) compile a .c the same way. There is no shared mutable
# state and no shared temp path. This was also checked empirically: a parallel
# run and a serial run produce byte-identical .ll files for all fifteen.
#
# Usage:
#   tests/gen-all.sh              regenerate and WRITE all fifteen .ll files
#   tests/gen-all.sh --check      verify each checked-in .ll still reproduces
#   tests/gen-all.sh [--check] ruby python ...
#                                 only those languages ("runtime" for the C floor)
#   MEC_JOBS=4 tests/gen-all.sh   cap the fan-out (default: number of cores)
#
# Exit 0 iff every generator that ran passed.
set -u

# THIS SCRIPT MATCHES ITS OWN GLOB. `for g in tests/gen-*.sh` is the idiom the
# manual documents and gates.sh used to use, and it now picks up gen-all.sh too -
# which without these two guards is a fork bomb (it did fork-bomb this machine
# once while this was being written). Guard one: the glob below skips $0. Guard
# two, for anyone else's loop: a nested invocation does nothing and succeeds.
if [ -n "${MEC_GEN_ALL:-}" ]; then
    echo "gen-all.sh: nested invocation (already inside gen-all.sh) - nothing to do"
    exit 0
fi
export MEC_GEN_ALL=1

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SELF="$(basename "${BASH_SOURCE[0]}")"
cd "$ROOT" || exit 2

CHECK=0
LANGS=()
for a in "$@"; do
    case "$a" in
        --check) CHECK=1 ;;
        -h|--help) sed -n '2,40p' "$0"; exit 0 ;;
        -*) echo "gen-all.sh: unknown flag $a" >&2; exit 2 ;;
        *) LANGS+=("$a") ;;
    esac
done

# Which generators. `runtime` is tests/gen-runtime-ll.sh; everything else is
# tests/gen-<lang>-rt-ll.sh.
GENS=()
if [ "${#LANGS[@]}" -eq 0 ]; then
    for g in tests/gen-*.sh; do
        [ -f "$g" ] || continue
        [ "$(basename "$g")" = "$SELF" ] && continue   # never recurse into myself
        GENS+=("$g")
    done
else
    for l in "${LANGS[@]}"; do
        if [ -f "tests/gen-$l-rt-ll.sh" ]; then GENS+=("tests/gen-$l-rt-ll.sh")
        elif [ -f "tests/gen-$l-ll.sh" ]; then GENS+=("tests/gen-$l-ll.sh")
        else echo "gen-all.sh: no generator for '$l'" >&2; exit 2; fi
    done
fi
if [ "${#GENS[@]}" -eq 0 ]; then echo "gen-all.sh: nothing to run" >&2; exit 2; fi

# One build for all of them.
TMP="$(mktemp -d -t mec-genall.XXXXXX)"
trap 'rm -rf "$TMP"' EXIT
if ! go build -o "$TMP/mec" . 2>"$TMP/build.log"; then
    cat "$TMP/build.log" >&2
    echo "gen-all.sh: BUILD FAILED" >&2
    exit 2
fi
export MEC_BIN="$TMP/mec"

JOBS="${MEC_JOBS:-$( (sysctl -n hw.ncpu || nproc || echo 4) 2>/dev/null )}"
case "$JOBS" in ''|*[!0-9]*) JOBS=4 ;; esac
[ "$JOBS" -lt 1 ] && JOBS=1

start=$(date +%s)
running=0
for g in "${GENS[@]}"; do
    b="$(basename "$g" .sh)"
    if [ "$CHECK" -eq 1 ]; then
        ( "$g" --check >"$TMP/$b.out" 2>&1; echo $? >"$TMP/$b.rc" ) &
    else
        ( "$g" >"$TMP/$b.out" 2>&1; echo $? >"$TMP/$b.rc" ) &
    fi
    running=$((running + 1))
    if [ "$running" -ge "$JOBS" ]; then wait -n 2>/dev/null || wait; running=$((running - 1)); fi
done
wait
end=$(date +%s)

failed=()
for g in "${GENS[@]}"; do
    b="$(basename "$g" .sh)"
    rc="$(cat "$TMP/$b.rc" 2>/dev/null || echo 99)"
    line="$(tail -1 "$TMP/$b.out" 2>/dev/null)"
    if [ "$rc" = 0 ]; then
        printf '  %-24s ok    %s\n' "$b" "$line"
    else
        printf '  %-24s FAIL  rc=%s %s\n' "$b" "$rc" "$line"
        failed+=("$b")
        sed 's/^/      | /' "$TMP/$b.out" 2>/dev/null | head -12
    fi
done

n="${#GENS[@]}"
if [ "${#failed[@]}" -eq 0 ]; then
    echo "gen-all: all $n clean ($((end - start))s, $JOBS jobs)"
    exit 0
fi
echo "gen-all: ${#failed[@]}/$n FAILED: ${failed[*]} ($((end - start))s, $JOBS jobs)"
exit 1
