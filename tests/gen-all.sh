#!/usr/bin/env bash
#
# gen-all.sh - run every tests/gen-*.sh generator against ONE shared `mec`, in
# parallel, and report one verdict.
#
# The generators' --check is the gate that catches a stale checked-in .ll, so it
# is in the inner loop of any runtime.c / *-rt.metajs change. Building mec once
# (exported as MEC_BIN, which every generator picks up instead of its own go
# build) and fanning the generators out takes it from ~25 s to ~3 s.
#
# MEC_BIN is a trust delegation: the checked-in .ll is compared against whatever
# binary it names. This script always builds a fresh one; do not export a stale
# MEC_BIN into a hand-run generator.
#
# Parallel is safe: each generator reads the grammars and its own source and
# writes only its own mktemp files and (outside --check) its own .ll. No
# generator reads another's output.
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

# This script matches its own glob (tests/gen-*.sh): the loop below skips $0, and a
# nested invocation from anyone else's loop does nothing, so it cannot fork-bomb.
if [ -n "${MEC_GEN_ALL:-}" ]; then
    echo "gen-all.sh: nested invocation (already inside gen-all.sh) - nothing to do"
    exit 0
fi
export MEC_GEN_ALL=1

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SELF="$(basename "${BASH_SOURCE[0]}")"
cd "$ROOT" || exit 2
. tests/lib/guard.sh

CHECK=0
LANGS=()
for a in "$@"; do
    case "$a" in
        --check) CHECK=1 ;;
        -h|--help) sed -n '2,26p' "$0"; exit 0 ;;
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
        [ "$(basename "$g")" = gen-rt-ll.sh ] && continue  # the shared body, needs LANG
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
trap 'guard_reap; rm -rf "$TMP"' EXIT
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
