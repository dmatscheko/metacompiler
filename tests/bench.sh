#!/usr/bin/env bash
#
# bench.sh - the native performance gate, with baselines checked in.
#
# WHY IT MEASURES INSTRUCTIONS AND NOT TIME. Every perf number in this project
# used to be wall clock in a plan document's prose, and wall clock cannot settle
# the questions that get asked here: with other work on the machine, a
# 40,000-iteration kotlin loop varies +-8% between runs of THE SAME BINARY, which
# is larger than most effects being argued about. `/usr/bin/time -l`'s
# "instructions retired" does not care what else is running: re-running ONE
# BINARY reproduces to ~0.02%. So: instructions, always, min of N runs.
#
# AND WHY THAT IS NOT ENOUGH - THE PART THAT COST TWO AGENTS AND ME A WRONG
# ANSWER EACH. A binary's INSTRUCTION COUNT DEPENDS ON ITS CODE LAYOUT, and the
# layout is perturbed by things that change no semantics at all. Measured here on
# ONE unchanged kotlin source, varying nothing but the length of the -exe output
# path:
#
#   17.353  17.494  17.620  17.911  17.915  17.916  17.916  18.018  18.023  18.182  G
#
# A 4.8% spread, visibly bimodal, from a file NAME. The module's own content
# perturbs layout the same way and by the same magnitude, so holding the output
# path fixed does NOT cancel it - changing the module draws a fresh sample
# either way. There is no pairing that cancels layout.
#
# Two consequences, and they are the whole reason this file works the way it does:
#
#   1. A SINGLE BUILD CANNOT MEASURE A SUB-2% EFFECT. Two agents and I each
#      A/B'd one build against one build and got +0.04%, +0.65% and -1.72% for
#      the same change. All three were draws from the distribution above.
#   2. The only sound estimator is a STATISTIC OVER MANY LAYOUT DRAWS. This
#      script builds each program to several different output paths and reports
#      the MEDIAN, with the observed spread next to it so you can see the width
#      of the lottery you are sampling.
#
# So the default tolerance is 2.5%, not 1%: below that, this instrument cannot
# tell you anything, and a gate that fires below its own noise floor is worse
# than no gate. If you need to resolve something smaller, raise --draws until the
# interval is tight enough, and quote the interval and not the point.
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
#   tests/bench.sh --runs N        min of N runs per BUILD (default 3)
#   tests/bench.sh --draws N       N layout draws per program (default 5) - see
#                                  the note below; 1 draw measures nothing
#   tests/bench.sh --tol P         flag a delta beyond P percent (default 2.5)
#
# Exit 0 iff nothing regressed beyond the tolerance. A row that IMPROVES is
# reported and does not fail the gate - but re-record the baseline when you land
# the improvement, or the next person inherits a stale one.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

BASE=tests/bench/baseline.txt
RECORD=0; RUNS=3; DRAWS=5; TOL=2.5; WANT=()
while [ $# -gt 0 ]; do
    case "$1" in
        --record) RECORD=1 ;;
        --runs)   RUNS="$2"; shift ;;
        --draws)  DRAWS="$2"; shift ;;
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

# The draws are taken by building to output paths of DIFFERENT LENGTHS. That is
# the cheapest available handle on layout: it changes no semantics, and the
# spread it produces is the same spread a real module edit produces. The absolute
# directory is FIXED so that two CHECKOUTS draw from the same set of paths and
# their medians are comparable - which is the whole reason it cannot be a mktemp
# directory (see the baseline file's header).
EXEDIR=/tmp/mec-bench
mkdir -p "$EXEDIR"

# ...and because it is fixed, TWO CONCURRENT RUNS WOULD OVERWRITE EACH OTHER'S
# BINARIES. That is not hypothetical: with four agents each running gates.sh, one
# reported `csharp BUILD FAILED` purely from the collision, which reads exactly
# like a real regression. So take a lock for the whole run. `mkdir` is the atomic
# primitive that exists everywhere; flock is not on darwin.
LOCK="$EXEDIR/.lock"
waited=0
until mkdir "$LOCK" 2>/dev/null; do
    if [ "$waited" = 0 ]; then
        echo "bench.sh: another bench run holds $LOCK - waiting (the -exe path must be" >&2
        echo "          identical across runs, so they cannot proceed in parallel)" >&2
    fi
    sleep 5
    waited=$((waited + 5))
    if [ "$waited" -ge 1800 ]; then
        echo "bench.sh: gave up after 30 min. If no bench run is active, remove $LOCK" >&2
        exit 2
    fi
done
trap 'rm -rf "$work" "$LOCK"' EXIT

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
printf '%-10s %16s %16s %9s  %s\n' language median baseline delta 'spread over draws'

for f in tests/bench/mod.*; do
    [ -f "$f" ] || continue
    ext="${f##*.}"
    lang=$(lang_of "$ext")
    [ -n "$lang" ] || { echo "bench.sh: no grammar known for .$ext ($f)" >&2; continue; }
    if [ ${#WANT[@]} -gt 0 ]; then
        hit=0; for w in "${WANT[@]}"; do [ "$w" = "$lang" ] && hit=1; done
        [ "$hit" = 1 ] || continue
    fi

    # One draw per output-path length. Each is a separate build of the SAME
    # source and a legitimate sample of the layout distribution.
    : > "$work/$lang.draws"
    bad_draw=0
    d=1
    while [ "$d" -le "$DRAWS" ]; do
        pad=$(printf 'x%.0s' $(seq 1 "$d"))
        exe="$EXEDIR/$lang$pad"
        if ! ./mec "languages/$lang-to-llvm-ir.abnf" "$f" -q -exe "$exe" >/dev/null 2>"$work/$lang.err"; then
            printf '%-10s %16s\n' "$lang" "BUILD FAILED"
            sed -n '1,3p' "$work/$lang.err" | sed 's/^/           /'
            bad_draw=1; break
        fi
        # A crashing binary looks FAST: /usr/bin/time reports happily on a
        # process that died. Check the exit code before believing any number.
        "$exe" >"$work/$lang.out" 2>&1; erc=$?
        if [ "$erc" != 0 ]; then
            printf '%-10s %16s (exit %d) - the timing of a crash is not a measurement\n' \
                "$lang" "RAN AND FAILED" "$erc"
            bad_draw=1; break
        fi
        n=$(instructions_of "$exe")
        if [ -z "$n" ]; then
            printf '%-10s %16s - /usr/bin/time -l reported no counter\n' "$lang" "NO COUNTER"
            bad_draw=1; break
        fi
        echo "$n" >> "$work/$lang.draws"
        d=$((d + 1))
    done
    if [ "$bad_draw" = 1 ]; then rc=1; continue; fi

    # The MEDIAN over draws, not the min: the min chases the luckiest layout and
    # is exactly as unstable as a single build.
    read -r n lo hi < <(sort -n "$work/$lang.draws" | awk '
        {v[NR] = $1}
        END {
            m = (NR % 2) ? v[(NR+1)/2] : int((v[NR/2] + v[NR/2+1]) / 2)
            printf "%d %d %d\n", m, v[1], v[NR]
        }')
    spread=$(awk -v a="$lo" -v b="$hi" 'BEGIN {printf "%.2f", (b-a)*100.0/a}')
    NAMES+=("$lang"); VALUES+=("$n")

    old=$(awk -v l="$lang" '$1 == l {print $2}' "$BASE" 2>/dev/null)
    if [ -z "$old" ]; then
        printf '%-10s %16s %16s %9s  %s%% over %d draws\n' "$lang" "$n" "-" "new" "$spread" "$DRAWS"
    else
        delta=$(awk -v a="$n" -v b="$old" 'BEGIN {printf "%+.2f", (a-b)*100.0/b}')
        printf '%-10s %16s %16s %8s%%  %s%% over %d draws\n' "$lang" "$n" "$old" "$delta" "$spread" "$DRAWS"
        # A delta smaller than THIS PROGRAM'S OWN layout spread is not evidence
        # of anything, whichever way it points and however big it looks. It must
        # not fail the gate: java has been seen at +3.20% with a 3.21% spread,
        # which is a coin landing heads, not a regression. Failing on it would
        # train everyone to ignore this gate, which is the worst outcome.
        noise=$(awk -v d="$delta" -v s="$spread" 'BEGIN {d = (d < 0) ? -d : d; print (d < s) ? 1 : 0}')
        over=$(awk -v d="$delta" -v t="$TOL" 'BEGIN {print (d > t) ? 1 : 0}')
        if [ "$noise" = 1 ]; then
            printf '           (delta is inside the layout spread - not evidence either way)\n'
        elif [ "$over" = 1 ]; then
            rc=1
        fi
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
