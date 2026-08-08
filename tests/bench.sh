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
# A 4.8% spread, VISIBLY BIMODAL, from a file NAME. The module's own content
# perturbs layout the same way and by the same magnitude, so holding the output
# path fixed does NOT cancel it - changing the module draws a fresh sample either
# way, and no pairing makes layout go away. What --paired below buys is a SIGN
# TEST over matched builds, which is a different and weaker claim than
# cancellation, and it is the strongest one available here.
#
# Note "bimodal" specifically: these values cluster around ~17.9 and ~18.02, they
# do not spread evenly. That is why the MEDIAN can be the wrong statistic - with
# draws split between clusters it is decided by which side the middle element
# lands on, and it can jump the whole gap while the code is unchanged. The rows
# below therefore print the mean too, and say so when a sample looks bimodal.
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
#   tests/bench.sh --paired DIR    compare against another CHECKOUT draw by draw,
#                                  building both to the SAME path so the layout term
#                                  cancels. This is the instrument to reach for when a
#                                  delta is smaller than the row's own spread - it has
#                                  settled such questions in BOTH directions.
#
# Exit 0 iff nothing regressed beyond the tolerance. A row that IMPROVES is
# reported and does not fail the gate - but re-record the baseline when you land
# the improvement, or the next person inherits a stale one.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

BASE=tests/bench/baseline.txt
RECORD=0; RUNS=3; DRAWS=5; TOL=2.5; PAIRED=""; WANT=()
while [ $# -gt 0 ]; do
    case "$1" in
        --record) RECORD=1 ;;
        --runs)   RUNS="$2"; shift ;;
        --draws)  DRAWS="$2"; shift ;;
        --paired) PAIRED="$2"; shift ;;
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

# ---------------------------------------------------------------------------
# --paired OTHERTREE: the instrument that settles a sub-spread question.
#
# It is a SIGN TEST over matched builds, and it is not magic: pairing does NOT
# cancel layout, because a content change perturbs layout too. What it does is
# hold the PATH constant and give you N matched comparisons instead of two summary
# statistics, so the question becomes "does the change win consistently?" rather
# than "which cluster did each median land in?". A consistent sign across most
# pairs is evidence; a mix is not, however large the mean gap looks.
#
# It has settled two questions that medians got wrong in OPPOSITE directions:
# swift's Array.insert read +2.73% on 21-draw medians and is 12-of-14 pairs
# IDENTICAL - a bimodal median artifact, corroborated by an md5-identical emitted
# module - while python's builtin table read +2.67% on medians and is genuinely
# higher in 11 of 12 pairs, corroborated by MEC_GC_STATS showing the heap step a
# whole 2 MiB arena doubling.
#
# ALWAYS corroborate a paired verdict with an exact counter (module bytes,
# MEC_GC_STATS, binary size). Counters are exempt from the lottery; timings are
# not, and this is still a timing.
#
# The other tree must be a full checkout that builds - `git archive <rev> | tar x`
# then `go build` INSIDE it (manual chapter 4: mec reads its grammar from the path
# you give it, so an old binary run here silently uses THIS tree's grammar).
# ---------------------------------------------------------------------------
if [ -n "$PAIRED" ]; then
    [ -x "$PAIRED/mec" ] || { echo "bench.sh: $PAIRED/mec is not built - run 'go build -o mec .' inside it" >&2; exit 2; }
    mkdir -p "$EXEDIR"
    printf '%-10s %5s %8s %8s %8s   %s\n' language pairs 'this' 'other' 'mean' 'verdict'
    prc=0
    for f in tests/bench/mod.*; do
        [ -f "$f" ] || continue
        lang=$(lang_of "${f##*.}"); [ -n "$lang" ] || continue
        if [ ${#WANT[@]} -gt 0 ]; then
            hit=0; for w in "${WANT[@]}"; do [ "$w" = "$lang" ] && hit=1; done
            [ "$hit" = 1 ] || continue
        fi
        [ -f "$PAIRED/$f" ] || { printf '%-10s %s\n' "$lang" "SKIPPED - $PAIRED has no $f"; continue; }
        : > "$work/$lang.pairs"; ok=1
        d=1
        while [ "$d" -le "$DRAWS" ]; do
            pad=$(printf 'x%.0s' $(seq 1 "$d"))
            exe="$EXEDIR/$lang$pad"
            ./mec "languages/$lang-to-llvm-ir.abnf" "$f" -q -exe "$exe" >/dev/null 2>&1 || { ok=0; break; }
            a=$(instructions_of "$exe")
            ( cd "$PAIRED" && ./mec "languages/$lang-to-llvm-ir.abnf" "$f" -q -exe "$exe" >/dev/null 2>&1 ) || { ok=0; break; }
            b=$(instructions_of "$exe")
            [ -n "$a" ] && [ -n "$b" ] || { ok=0; break; }
            echo "$a $b" >> "$work/$lang.pairs"
            d=$((d + 1))
        done
        [ "$ok" = 1 ] || { printf '%-10s %s\n' "$lang" "FAILED to build or measure a pair"; prc=1; continue; }
        read -r np wins losses ties ma mb pct < <(awk '
            {n++; sa += $1; sb += $2; if ($1 > $2) w++; else if ($1 < $2) l++; else t++}
            END { printf "%d %d %d %d %d %d %.2f\n", n, w+0, l+0, t+0, sa/n, sb/n, (sa-sb)*100.0/sb }' "$work/$lang.pairs")
        verdict="no difference"
        sig=$(awk -v w="$wins" -v l="$losses" -v n="$np" 'BEGIN { print (w >= 0.8*n) ? 1 : ((l >= 0.8*n) ? -1 : 0) }')
        [ "$sig" = "1" ]  && verdict="SLOWER in ${wins}/${np} pairs"
        [ "$sig" = "-1" ] && verdict="faster in ${losses}/${np} pairs"
        printf '%-10s %5s %8s %8s %+7s%%   %s (ties %s)\n' "$lang" "$np" "$ma" "$mb" "$pct" "$verdict" "$ties"
    done
    exit $prc
fi

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
    #
    # BUT THE MEDIAN IS THE WRONG STATISTIC WHEN THE SAMPLE IS BIMODAL, which on
    # this target it often is - layout tends to land in one of two clusters, not
    # on a continuum. With half the draws in each cluster the median is decided by
    # which side the middle element falls on, so it can jump the whole gap between
    # clusters while nothing about the code changed. That is not hypothetical: a
    # swift change measured +2.73% on 21-draw medians and 0% paired, and the
    # emitted module was md5-identical. So the mean is reported too, and a bimodal
    # sample is called out - on one, believe the MEAN and then go and settle it
    # with --paired.
    read -r n lo hi mean gap gaplo < <(sort -n "$work/$lang.draws" | awk '
        {v[NR] = $1; s += $1}
        END {
            m = (NR % 2) ? v[(NR+1)/2] : int((v[NR/2] + v[NR/2+1]) / 2)
            bg = 0; bi = 1
            for (i = 1; i < NR; i++) if (v[i+1] - v[i] > bg) { bg = v[i+1] - v[i]; bi = i }
            printf "%d %d %d %d %d %d\n", m, v[1], v[NR], int(s/NR), bg, bi
        }')
    spread=$(awk -v a="$lo" -v b="$hi" 'BEGIN {printf "%.2f", (b-a)*100.0/a}')
    # Bimodal iff the largest gap between neighbouring draws is most of the total
    # spread AND it splits the sample rather than shaving off one outlier.
    bimodal=$(awk -v g="$gap" -v lo="$lo" -v hi="$hi" -v i="$gaplo" -v n="$DRAWS" \
        'BEGIN { print (hi > lo && g > 0.4 * (hi - lo) && i >= n/4 && i <= 3*n/4) ? 1 : 0 }')
    NAMES+=("$lang"); VALUES+=("$n")

    old=$(awk -v l="$lang" '$1 == l {print $2}' "$BASE" 2>/dev/null)
    if [ -z "$old" ]; then
        printf '%-10s %16s %16s %9s  %s%% over %d draws\n' "$lang" "$n" "-" "new" "$spread" "$DRAWS"
    else
        delta=$(awk -v a="$n" -v b="$old" 'BEGIN {printf "%+.2f", (a-b)*100.0/b}')
        mdelta=$(awk -v a="$mean" -v b="$old" 'BEGIN {printf "%+.2f", (a-b)*100.0/b}')
        printf '%-10s %16s %16s %8s%%  %s%% over %d draws\n' "$lang" "$n" "$old" "$delta" "$spread" "$DRAWS"
        printf '           mean %s (%s%%)%s\n' "$mean" "$mdelta" \
            "$([ "$bimodal" = 1 ] && echo '  <- BIMODAL sample: the median is unreliable here, read the mean and settle it with --paired')"
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
