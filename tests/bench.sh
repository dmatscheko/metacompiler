#!/usr/bin/env bash
#
# bench.sh - the native performance gate, with baselines checked in.
#
# WHY IT MEASURES INSTRUCTIONS AND NOT TIME. Every perf number in this project
# used to be wall clock in a plan document's prose, and wall clock cannot settle
# the questions that get asked here: with other work on the machine, a
# 40,000-iteration kotlin loop varies +-8% between runs of THE SAME BINARY, which
# is larger than most effects being argued about. `/usr/bin/time -l`'s
# "instructions retired" reproduces to ~0.02% across rebuilds and does not care
# what else is running. So: instructions, always, min of N runs.
#
# The programs in tests/bench/ are deliberately the same loop in every language
# (`s = s + i % 7`, 40,000 iterations, no float, no regexp, no allocation in the
# body) - that is the harness docs/runtime-merge-plan.md's measurements use, so
# the numbers here are comparable to the ones written up there. `mod.c` is the
# CONTROL: it compiles to self-contained IR with no js_* extern at all, so a
# floor or layer-2 change that moves it is measuring the machine, not the change.
#
# Usage:
#   tests/bench.sh                 run every program, diff against the baseline
#   tests/bench.sh kotlin python   run only these
#   tests/bench.sh --record        overwrite the baseline with today's numbers
#   tests/bench.sh --runs N        min of N runs per program (default 3)
#   tests/bench.sh --tol P         flag a delta beyond P percent (default 1.0)
#
# Exit 0 iff nothing regressed beyond the tolerance. A row that IMPROVES is
# reported and does not fail the gate - but re-record the baseline when you land
# the improvement, or the next person inherits a stale one.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

BASE=tests/bench/baseline.txt
RECORD=0; RUNS=3; TOL=1.0; WANT=()
while [ $# -gt 0 ]; do
    case "$1" in
        --record) RECORD=1 ;;
        --runs)   RUNS="$2"; shift ;;
        --tol)    TOL="$2"; shift ;;
        -h|--help) sed -n '2,28p' "$0"; exit 0 ;;
        -*) echo "bench.sh: unknown flag $1" >&2; exit 2 ;;
        *) WANT+=("$1") ;;
    esac
    shift
done

[ -x ./mec ] || go build -o mec . || exit 2

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# ext -> language, i.e. which grammar compiles it.
lang_of() {
    case "$1" in
        kt) echo kotlin ;; py) echo python ;; lua) echo lua ;; c) echo c ;;
        java) echo java ;; rb) echo ruby ;; kts) echo kotlin ;;
        js) echo js ;; ts) echo typescript ;; go) echo go ;; php) echo php ;;
        cs) echo csharp ;; swift) echo swift ;; dart) echo dart ;;
        *) echo "" ;;
    esac
}

# instructions_of BINARY -> the minimum over $RUNS runs, or "" if it never ran.
instructions_of() {
    local best="" n
    for _ in $(seq "$RUNS"); do
        n=$(/usr/bin/time -l "$1" 2>&1 >/dev/null | awk '/instructions retired/ {print $1}')
        [ -n "$n" ] || continue
        if [ -z "$best" ] || [ "$n" -lt "$best" ]; then best="$n"; fi
    done
    echo "$best"
}

declare -a NAMES VALUES
rc=0
printf '%-10s %16s %16s %9s\n' language instructions baseline delta

for f in tests/bench/mod.*; do
    [ -f "$f" ] || continue
    ext="${f##*.}"
    lang=$(lang_of "$ext")
    [ -n "$lang" ] || { echo "bench.sh: no grammar known for .$ext ($f)" >&2; continue; }
    if [ ${#WANT[@]} -gt 0 ]; then
        hit=0; for w in "${WANT[@]}"; do [ "$w" = "$lang" ] && hit=1; done
        [ "$hit" = 1 ] || continue
    fi

    exe="$work/$lang.exe"
    if ! ./mec "languages/$lang-to-llvm-ir.abnf" "$f" -q -exe "$exe" >/dev/null 2>"$work/$lang.err"; then
        printf '%-10s %16s\n' "$lang" "BUILD FAILED"
        sed -n '1,3p' "$work/$lang.err" | sed 's/^/           /'
        rc=1; continue
    fi
    # A crashing binary looks FAST: /usr/bin/time reports happily on a process
    # that died. Check the exit code before believing any number below it.
    "$exe" >"$work/$lang.out" 2>&1; erc=$?
    if [ "$erc" != 0 ]; then
        printf '%-10s %16s (exit %d) - the timing of a crash is not a measurement\n' \
            "$lang" "RAN AND FAILED" "$erc"
        rc=1; continue
    fi

    n=$(instructions_of "$exe")
    if [ -z "$n" ]; then
        printf '%-10s %16s - /usr/bin/time -l reported no counter\n' "$lang" "NO COUNTER"
        rc=1; continue
    fi
    NAMES+=("$lang"); VALUES+=("$n")

    old=$(awk -v l="$lang" '$1 == l {print $2}' "$BASE" 2>/dev/null)
    if [ -z "$old" ]; then
        printf '%-10s %16s %16s %9s\n' "$lang" "$n" "-" "new"
    else
        d=$(awk -v a="$n" -v b="$old" 'BEGIN {printf "%+.2f", (a-b)*100.0/b}')
        printf '%-10s %16s %16s %8s%%\n' "$lang" "$n" "$old" "$d"
        over=$(awk -v d="$d" -v t="$TOL" 'BEGIN {print (d > t) ? 1 : 0}')
        [ "$over" = 1 ] && rc=1
    fi
done

if [ "$RECORD" = 1 ]; then
    {
        echo "# tests/bench.sh baselines: instructions retired, min of $RUNS runs,"
        echo "# native -exe binaries built from tests/bench/mod.*. Re-record with"
        echo "# tests/bench.sh --record when you land an intended change, and say"
        echo "# in the commit message which rows moved and why."
        echo "# recorded at commit $(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
        for i in "${!NAMES[@]}"; do printf '%-10s %s\n' "${NAMES[$i]}" "${VALUES[$i]}"; done
    } > "$BASE"
    echo
    echo "baseline written to $BASE"
    exit 0
fi

echo
if [ "$rc" = 0 ]; then echo "no regression beyond ${TOL}%"; else echo "REGRESSED (or failed) - see the rows above"; fi
exit $rc
