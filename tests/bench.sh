#!/usr/bin/env bash
#
# bench.sh - the native performance gate, with baselines checked in.
#
# It measures INSTRUCTIONS RETIRED (min of N runs), not wall clock: re-running one
# binary reproduces to ~0.02% regardless of machine load.
#
# But a binary's instruction count depends on its CODE LAYOUT, which moves for
# reasons that change no semantics - even the length of the -exe output path
# (one unchanged kotlin source spread 4.8%, bimodally, over ten path lengths).
# Changing the module perturbs layout the same way. Hence:
#   - a single build cannot measure a sub-2% effect;
#   - each program is built to several output paths ("layout draws") and the row
#     reports the MEDIAN with its spread, plus the MEAN, and flags a bimodal
#     sample (there the median can jump between clusters; read the mean and
#     settle it with --paired);
#   - the default tolerance is 2.5%. To resolve less, raise --draws and quote the
#     interval, not the point.
#
# The programs in tests/bench/mod.* are the same loop in every language
# (`s = s + i % 7`, no float, no regexp, no allocation in the body). c, bash and
# batch compile to self-contained IR against a C runtime and are the CONTROLS: a
# floor or layer-2 change that moves them is measuring the machine. The other
# rows link lib/runtime.ll plus lib/<lang>-rt.ll. Never compare a row of one
# family against the other. mod.sh runs fewer iterations because bash's arena is
# uncollected. typescript has no program: it links the same layer 2 as js.
#
# Every handle-IR row also prints `heap` and `collections` from MEC_GC_STATS.
# They are exact counters, exempt from the layout lottery. The heap grows in
# whole ~1 MiB chunks the first time a size class is used, which costs nothing
# measurable; a changed collection count means the row executes different work.
#
# Usage:
#   tests/bench.sh                 run every program, diff against the baseline
#   tests/bench.sh kotlin python   run only these
#   tests/bench.sh --record        overwrite the measured rows of the baseline
#                                  (header and unmeasured rows are kept)
#   tests/bench.sh --runs N        min of N runs per BUILD (default 3)
#   tests/bench.sh --draws N       N layout draws per program (default 5)
#   tests/bench.sh --tol P         flag a delta beyond P percent (default 2.5)
#   tests/bench.sh --paired DIR    compare against another CHECKOUT draw by draw,
#                                  both built to the SAME path - a sign test over
#                                  matched builds, not a cancellation of layout
#                                  (see the note at the --paired block)
#
# Exit 0 iff nothing regressed beyond the tolerance. An improvement does not fail
# the gate, but re-record the baseline when you land it.

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
        -h|--help) sed -n '2,47p' "$0"; exit 0 ;;
        -*) echo "bench.sh: unknown flag $1" >&2; exit 2 ;;
        *) WANT+=("$1") ;;
    esac
    shift
done

[ -x ./mec ] || go build -o mec . || exit 2


work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Draws build to output paths of different lengths (a semantics-free handle on
# layout). The directory is FIXED so two checkouts draw from the same paths and
# their medians compare - so it cannot be a mktemp directory, and concurrent runs
# must be serialised by a lock (mkdir is atomic; flock is not on darwin).
EXEDIR=/tmp/mec-bench
mkdir -p "$EXEDIR"

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
#
# .metajs is not a real MetaJS extension - every MetaJS file in the tree is a
# .js - but mod.js already claims .js for the js grammar, and one extension can
# only name one grammar here. tests/bench/mod.metajs says so at the top.
lang_of() {
    case "$1" in
        kt) echo kotlin ;; py) echo python ;; lua) echo lua ;; c) echo c ;;
        java) echo java ;; rb) echo ruby ;; kts) echo kotlin ;;
        js) echo js ;; ts) echo typescript ;; go) echo go ;; php) echo php ;;
        cs) echo csharp ;; swift) echo swift ;; dart) echo dart ;;
        metajs) echo metajs ;; sh) echo bash ;; bat) echo batch ;;
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
# --paired OTHERTREE: a SIGN TEST over matched builds. Pairing holds the output
# path constant; it does NOT cancel layout, because a content change perturbs
# layout too. A consistent sign across most pairs is evidence, a mix is not.
# Corroborate with an exact counter (module bytes, MEC_GC_STATS, binary size).
#
# A unanimous sign is still NOT evidence for a change to a SHARED layer-2 file:
# adding a name to languages/lib/runtime.metajs renumbers the @str.N intern table
# of every importing module, a perturbation identical in every pair. Measure
# against a dead-code matched control (the same insertion, uncalled), not base.
#
# The other tree must be a full checkout that builds: `git archive <rev> | tar x`
# then `go build` INSIDE it (mec reads its grammar from the path you give it).
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

declare -a NAMES VALUES HEAPS COLLS
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
        # The arena counters are exact and layout-independent: one draw answers
        # them. The self-contained rows have no collector and record "-".
        if [ "$d" = 1 ]; then
            gcline=$(MEC_GC_STATS=1 "$exe" 2>&1 >/dev/null | sed -n 's/^gc: //p')
            heap=$(printf '%s\n' "$gcline" | tr ' ' '\n' | sed -n 's/^heap=//p')
            coll=$(printf '%s\n' "$gcline" | tr ' ' '\n' | sed -n 's/^collections=//p')
            [ -n "$heap" ] || heap="-"
            [ -n "$coll" ] || coll="-"
        fi
        echo "$n" >> "$work/$lang.draws"
        d=$((d + 1))
    done
    if [ "$bad_draw" = 1 ]; then rc=1; continue; fi

    # The MEDIAN over draws, not the min (the min chases the luckiest layout).
    # The mean is reported too, for bimodal samples.
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
    NAMES+=("$lang"); VALUES+=("$n"); HEAPS+=("$heap"); COLLS+=("$coll")

    old=$(awk -v l="$lang" '$1 == l {print $2}' "$BASE" 2>/dev/null)
    if [ -z "$old" ]; then
        printf '%-10s %16s %16s %9s  %s%% over %d draws\n' "$lang" "$n" "-" "new" "$spread" "$DRAWS"
    else
        delta=$(awk -v a="$n" -v b="$old" 'BEGIN {printf "%+.2f", (a-b)*100.0/b}')
        mdelta=$(awk -v a="$mean" -v b="$old" 'BEGIN {printf "%+.2f", (a-b)*100.0/b}')
        printf '%-10s %16s %16s %8s%%  %s%% over %d draws\n' "$lang" "$n" "$old" "$delta" "$spread" "$DRAWS"
        printf '           mean %s (%s%%)%s\n' "$mean" "$mdelta" \
            "$([ "$bimodal" = 1 ] && echo '  <- BIMODAL sample: the median is unreliable here, read the mean and settle it with --paired')"
        # A delta inside this program's own layout spread is not evidence either
        # way and must not fail the gate.
        noise=$(awk -v d="$delta" -v s="$spread" 'BEGIN {d = (d < 0) ? -d : d; print (d < s) ? 1 : 0}')
        over=$(awk -v d="$delta" -v t="$TOL" 'BEGIN {print (d > t) ? 1 : 0}')
        if [ "$noise" = 1 ]; then
            printf '           (delta is inside the layout spread - not evidence either way)\n'
        elif [ "$over" = 1 ]; then
            rc=1
        fi
    fi

    # The arena line never fails the gate; it attributes a heap step.
    oheap=$(awk -v l="$lang" '$1 == l {print $3}' "$BASE" 2>/dev/null)
    ocoll=$(awk -v l="$lang" '$1 == l {print $4}' "$BASE" 2>/dev/null)
    if [ "$heap" = "-" ]; then
        printf '           arena  no collector in this row (self-contained IR against a C runtime)\n'
    else
        printf '           arena  heap=%s collections=%s' "$heap" "$coll"
        if [ -n "$oheap" ] && [ "$oheap" != "-" ] && [ "$oheap" != "$heap" ]; then
            read -r hd chunks rest < <(awk -v a="$heap" -v b="$oheap" \
                'BEGIN { d = a - b; c = d / 1048576; printf "%+d %+d %d\n", d, (c < 0 ? -int(-c) : int(c)), (d % 1048576 + 1048576) % 1048576 }')
            printf '  <- STEPPED %s bytes from the recorded %s\n' "$hd" "$oheap"
            if [ "$chunks" != "+0" ] && [ "$chunks" != "-0" ]; then
                printf '                  = %s arena chunk(s) of 1 MiB' "$chunks"
                [ "$rest" != 0 ] && printf ' plus %s bytes of big blocks' "$rest"
                printf ' (a size class used for the first time; costs nothing measurable)\n'
            else
                printf '                  under one chunk, so this is big-block traffic (>1024 bytes a\n'
                printf '                  block), which costs exactly its own bytes and no chunk.\n'
            fi
        else
            printf '\n'
        fi
        if [ -n "$ocoll" ] && [ "$ocoll" != "-" ] && [ "$ocoll" != "$coll" ]; then
            printf '                  collections %s -> %s: this row now EXECUTES different work,\n' "$ocoll" "$coll"
            printf '                  which a pure layer-2 size change never does.\n'
        fi
    fi
done

if [ "$RECORD" = 1 ]; then
    # Carries every `#` header line through (except the previous provenance
    # line) and keeps the rows this run did not measure.
    {
        if grep -q '^#' "$BASE" 2>/dev/null; then
            grep '^#' "$BASE" | grep -v '^# recorded at commit'
        else
            echo "# tests/bench.sh baselines. Columns: language, instructions retired"
            echo "# (MEDIAN over layout draws, min of $RUNS runs each), then the two EXACT"
            echo "# arena counters heap and collections (MEC_GC_STATS)."
            echo "# Native -exe binaries built from tests/bench/mod.*. Re-record with"
            echo "# tests/bench.sh --record when you land an intended change, and say"
            echo "# in the commit message which rows moved and why."
        fi
        echo "# recorded at commit $(git rev-parse --short HEAD 2>/dev/null || echo unknown), $DRAWS draws, min of $RUNS runs."
        {
            for i in "${!NAMES[@]}"; do
                printf '%-10s %-16s %-10s %s\n' \
                    "${NAMES[$i]}" "${VALUES[$i]}" "${HEAPS[$i]}" "${COLLS[$i]}"
            done
            # Rows this run did not measure survive untouched.
            awk -v keep=" $(printf '%s ' "${NAMES[@]}")" '
                /^#/ || /^[ \t]*$/ { next }
                index(keep, " " $1 " ") { next }
                { print }' "$BASE" 2>/dev/null
        } | sort
    } > "$work/base.new"
    mv "$work/base.new" "$BASE"
    echo
    echo "baseline written to $BASE (header preserved, unmeasured rows kept)"
    exit 0
fi

echo
if [ "$rc" = 0 ]; then echo "no regression beyond ${TOL}%"; else echo "REGRESSED (or failed) - see the rows above"; fi
exit $rc
