#!/usr/bin/env bash
#
# gates.sh - run EVERY gate this project has, in one command, and report one
# verdict.
#
# Why this exists: the gates are eight separate commands with three different
# reporting conventions, and two of them are only correct when you also grep
# their output. Getting that wrong is not hypothetical - a `-frozen`-only
# divergence never reached `--full`'s summary line at all until 52dff5e, and the
# default suite is STRUCTURALLY BLIND to every languages/lib/*-rt.metajs change
# because ./test.sh, --full and --cross all run llvm.Run, which resolves js_* to
# the Go twin and never touches layer 2. Only clang-check, native-full and a
# native -exe probe can see layer 2 at all.
#
# So the rule this script encodes: green means all eight, including the greps AND
# the two RECORDED NUMBERS (FULL_ASSERTIONS, SHAPE_GROUPS) below - a gate that
# prints a smaller number without complaining is worse than no gate, and this one
# used to do exactly that.
#
# Usage:
#   tests/gates.sh              every gate (~5-6 minutes)
#   tests/gates.sh --quick      skip the two slowest (clang-check, native-full).
#                               NOT sufficient for a layer-2 change - it says so.
#   tests/gates.sh --gen        also re-check all fifteen checked-in .ll files
#                               reproduce from source (MANDATORY after any
#                               runtime.c / *-rt.metajs edit; via tests/gen-all.sh,
#                               which shares one binary and runs them in parallel)
#   tests/gates.sh --freeze     also check `-freeze` is a fixed point (MANDATORY
#                               after any metajs-to-llvm-ir.abnf / compile-core.js
#                               edit). Always runs LAST and alone - it REWRITES
#                               abnf/jsbootstrap.ll and abnf/jsagrammar.go, which
#                               every other gate reads.
#   tests/gates.sh --serial     run the gates one at a time (the old behaviour).
#                               Use on a low-core machine, or to get clean timings.
#   tests/gates.sh --bench      also check native PERFORMANCE against the recorded
#                               baselines (tests/bench.sh). Not on by default
#                               because it is slow - but see below for why you
#                               should use it before believing a refactor is free.
#
# THE GATES RUN CONCURRENTLY, and the reason it is safe is worth stating because a
# false green here is the worst outcome this repo has available:
#
#   ./test.sh          builds its OWN binary (mktemp) and keeps results in mktemp.
#                      Its only writes inside the repo are the `-exe PATH` targets
#                      named in .vscode/launch.json (tests/*.out), and only the
#                      MATRIX run uses those - --full and --cross do not.
#   clang-check.sh     its own mktemp binary, its own mktemp work dir.
#   native-full.sh     mktemp output dir; READS ./mec, never writes it.
#   gen-all.sh         its own temp binary; --check writes nothing.
#   go test            writes nothing.
#   shape-scan         reads languages/lib/*.metajs; writes nothing.
#
# So no two concurrent gates write the same path, and ./mec is written exactly
# once, before the fan-out. The VERDICT LOGIC BELOW IS UNCHANGED - only when each
# command runs is different, and each verdict still reads that gate's own captured
# output.
#
# Exit 0 iff every gate that ran passed.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

QUICK=0; GEN=0; FREEZE=0; SERIAL=0; BENCH=0
for a in "$@"; do
    case "$a" in
        --quick)  QUICK=1 ;;
        --gen)    GEN=1 ;;
        --freeze) FREEZE=1 ;;
        --serial) SERIAL=1 ;;
        --bench)  BENCH=1 ;;
        --all)    GEN=1; FREEZE=1; BENCH=1 ;;
        -h|--help) sed -n "2,38p" "$0"; exit 0 ;;
        *) echo "gates.sh: unknown flag $a" >&2; exit 2 ;;
    esac
done

# ----------------------------------------------------------------------------
# The two recorded expectations, and why a gate that only prints a number is not
# a gate.
#
# THE RATCHET UNDER-REPORTS SILENTLY WHEN A RUN IS KILLED. ./test.sh --full
# defaults to a 120 s per-run timeout; the js compiler half takes ~19 s idle and
# 82 s AT LOAD 16. With several agents on the machine this script has been seen
# reporting the ratchet 437 ASSERTIONS BELOW BASELINE WITH NO FAILURE ANYWHERE -
# the js and kotlin probes had been killed, both halves of each recorded no count
# at all, the halves therefore AGREED, and the summary faithfully printed a
# smaller number as though it were the answer.
#
# Two changes close it. FULL_TIMEOUT raises the brake so a loaded machine does
# not trip it, and FULL_ASSERTIONS says what the answer is supposed to be, so a
# short read is a FAILURE rather than a smaller number. (test.sh now also emits
# RUN-FAILED when a probe run is killed, which the grep below catches - but that
# is the mechanism, and this is the invariant. Both, because the class of bug
# being defended against is "the gate said nothing".)
#
# FULL_ASSERTIONS is a FLOOR, not an equality: the ratchet files legitimately
# grow. Growing it is a one-line edit and the script tells you the new number.
FULL_TIMEOUT=600
#
# 2026-08-08: this working tree measures 7720, but the number recorded here is
# the 6f60337 baseline of 7625 ON PURPOSE - other agents' ratchet additions were
# in flight and uncommitted when it was measured, and a floor must not be set from
# a tree that may not be the one that gets committed. The gate says "+95, RAISE
# FULL_ASSERTIONS" until someone does; that message IS the ratchet doing its job.
FULL_ASSERTIONS=7766      # recorded 2026-08-08 after the ten-item round

# The shape ratchet: layer-2 bodies in languages/lib/*.metajs that are identical
# modulo names, ACROSS FILES, at the default -min 140. It is the only measure of
# the duplication the extern-NAME view cannot see (manual 7.12), and until now
# nothing ran it. A NEW cross-language copy-paste raises this and fails the gate.
#
# Note this is not "2". docs/todo.md 7.8 says `-max 2` and that number predates
# a69a4fd, which fixed shape-scan's brace counting - it had been running one-line
# function bodies on into the next function and mis-grouping the result. It sees
# 2,191 top-level bodies today where it saw 1,713, and the honest current count is
# 15 groups / 820 lines occupied / 481 recoverable. Lowering it is item 5.1's job.
#
# BOTH numbers, because the group count alone has a hole: a body copied into a
# THIRD language joins an existing group and the count does not move. Measured -
# pasting pyABigMul into lua-rt.metajs left it at 15 while recoverable went
# 481 -> 511.
SHAPE_GROUPS=15           # recorded 2026-08-08 at 6f60337
SHAPE_LINES=481           # recoverable lines, same commit

FAILED=()
pass() { printf '  %-24s %s\n' "$1" "$2"; }
fail() { printf '  %-24s %s\n' "$1" "$2"; FAILED+=("$1"); }

log=$(mktemp -d)
trap 'rm -rf "$log"' EXIT

echo "== building"
if ! go build -o mec . 2>"$log/build"; then
    cat "$log/build"; echo "gates.sh: BUILD FAILED"; exit 1
fi

# start KEY CMD... - run a gate, capturing its output and exit status under $log.
# Concurrent unless --serial. Every verdict below reads these two files and
# nothing else, so the two modes cannot disagree.
start() {
    key="$1"; shift
    if [ "$SERIAL" = 1 ]; then
        "$@" >"$log/$key" 2>&1; echo $? >"$log/$key.rc"
    else
        ( "$@" >"$log/$key" 2>&1; echo $? >"$log/$key.rc" ) &
    fi
}
rc_of() { cat "$log/$1.rc" 2>/dev/null || echo 127; }

echo "== gates$([ "$SERIAL" = 1 ] && echo ' (serial)')"

# --timeout on all three: this script runs six gates CONCURRENTLY, so every run
# here is by construction a run at load, and test.sh's 120 s default is not sized
# for that (js's compiler half: ~19 s idle, 82 s at load 16). The matrix and
# --cross fail LOUDLY on a killed run - they compare exit codes between two legs -
# so for them this only removes false reds; --full is the one where a kill used to
# be silent, and it is the reason the number below is checked as well.
start matrix ./test.sh --timeout "$FULL_TIMEOUT"
start full   ./test.sh --full  --timeout "$FULL_TIMEOUT"
start cross  ./test.sh --cross --timeout "$FULL_TIMEOUT"
start gotest go test ./abnf/
start shape  go run ./tools/shape-scan -max "$SHAPE_GROUPS" -max-lines "$SHAPE_LINES"
if [ "$QUICK" = 0 ]; then
    start clang  tests/clang-check.sh
    start native tests/native-full.sh
fi
[ "$GEN" = 1 ] && start gen tests/gen-all.sh --check
# NOT concurrent with the others, and not by default. bench.sh measures INSTRUCTIONS
# RETIRED, which does not care about machine load - but it builds native binaries to
# fixed paths, and its draws must not contend with six other gates for cores while
# doing it. It runs after the wait, below.
wait

# 1. The matrix. Its own summary line is trustworthy.
m=$(grep -E '^matrix: ' "$log/matrix" | tail -1)
if grep -q '^all green' "$log/matrix"; then pass matrix "${m:-ok}"; else fail matrix "${m:-NO SUMMARY} - see $log/matrix"; cp "$log/matrix" /tmp/gates-matrix.log; fi

# 2. The ratchet. Informational by design - it ALWAYS exits 0 - so the summary
#    line is not a verdict and the grep is the actual gate.
bad=$(grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF|RUN-FAILED' "$log/full")
n=$(grep -E 'assertions in total' "$log/full" | tail -1)
# A language whose count is '-' recorded NOTHING - it never reached the summary
# line. Two halves that both do this AGREE, so it produces no MISMATCH; it is
# invisible except in the total. Name it.
blank=$(grep -E '^  [a-z0-9]+ +-$' "$log/full" | awk '{print $1}' | tr '\n' ' ')
# The total itself, as a number, against what it is supposed to be.
tot=$(printf '%s\n' "$n" | sed -n 's/^ *\([0-9][0-9]*\) assertions in total.*/\1/p')
if [ -n "$bad" ]; then
    fail --full "${n:-ran} -- $(echo "$bad" | head -3)"
elif [ -n "$blank" ]; then
    fail --full "${n:-ran} -- RECORDED NOTHING for: $blank (both halves silent: not a MISMATCH, so only the total moved)"
elif [ -z "$tot" ]; then
    fail --full "NO ASSERTION TOTAL in the output at all - the run did not finish"
elif [ "$tot" -lt "$FULL_ASSERTIONS" ]; then
    fail --full "SHORT READ: $tot assertions, $((FULL_ASSERTIONS - tot)) BELOW the recorded $FULL_ASSERTIONS."
    printf '  %-24s %s\n' "" "A SMALLER NUMBER IS A FAILURE, not a smaller test run. Usually killed runs"
    printf '  %-24s %s\n' "" "(raise --timeout / re-run when quiet); otherwise assertions stopped executing."
    cp "$log/full" /tmp/gates-full.log
elif [ "$tot" -gt "$FULL_ASSERTIONS" ]; then
    pass --full "$tot assertions (+$((tot - FULL_ASSERTIONS)) over the recorded $FULL_ASSERTIONS - RAISE FULL_ASSERTIONS in tests/gates.sh), 0 halves disagree"
else
    pass --full "${n:-ran}"
fi

# 3. The two halves of each language, diffed against each other.
c=$(grep -E 'programs compared' "$log/cross" | tail -1)
case "$c" in
    *", 0 divergent"*) pass --cross "$c" ;;
    *) fail --cross "${c:-no summary}" ;;
esac

# 4. Go-side unit tests.
if [ "$(rc_of gotest)" = 0 ]; then pass "go test" "$(tail -1 "$log/gotest")"
else fail "go test" "$(tail -3 "$log/gotest" | tr '\n' ' ')"; fi

# 5. The layer-2 shape ratchet (docs/todo.md 7.8). Cheap, and the only gate that
#    can see a body copy-pasted into a second language.
srcv=$(rc_of shape)
ss=$(grep -E 'cross-language groups' "$log/shape" | tail -1)
sg=$(printf '%s\n' "$ss" | sed -n 's/^\([0-9][0-9]*\) cross-language groups.*/\1/p')
sl=$(printf '%s\n' "$ss" | sed -n 's/.*, \([0-9][0-9]*\) recoverable.*/\1/p')
if [ "$srcv" != 0 ]; then
    fail shape-scan "${ss:-no summary} - ABOVE the recorded $SHAPE_GROUPS groups / $SHAPE_LINES recoverable lines. A new cross-language duplicate; copied to /tmp/gates-shape.log"
    cp "$log/shape" /tmp/gates-shape.log
elif [ -n "$sg" ] && { [ "$sg" -lt "$SHAPE_GROUPS" ] || { [ -n "$sl" ] && [ "$sl" -lt "$SHAPE_LINES" ]; }; }; then
    pass shape-scan "$ss (BELOW the recorded $SHAPE_GROUPS/$SHAPE_LINES - lower SHAPE_GROUPS/SHAPE_LINES in tests/gates.sh)"
else
    pass shape-scan "${ss:-ok}"
fi

if [ "$QUICK" = 0 ]; then
    # 6 and 7 are THE ONLY GATES THAT SEE LAYER 2.
    crc=$(rc_of clang)
    held=$(grep -c 'row HELD' "$log/clang")
    ok=$(grep -c 'the clang executable agrees' "$log/clang")
    if [ "$crc" = 0 ] && [ "$ok" -ge 16 ] && [ "$held" = 0 ]; then pass clang-check "$ok/16 agreeing, none held"
    else fail clang-check "$ok/16 agreeing, $held held, rc=$crc - copied to /tmp/gates-clang.log"; cp "$log/clang" /tmp/gates-clang.log; fi

    nf=$(grep -E 'languages, every native binary|native binaries' "$log/native" | tail -1)
    if grep -q 'every native binary exits 0 with 0 failures' "$log/native"; then pass native-full "$nf"
    else fail native-full "${nf:-see $log/native}"; cp "$log/native" /tmp/gates-native.log; fi
else
    printf '  %-24s %s\n' "clang-check" "SKIPPED (--quick)"
    printf '  %-24s %s\n' "native-full" "SKIPPED (--quick)"
    echo "  NOTE: --quick cannot see languages/lib/*-rt.metajs. Do not call a layer-2 change green on it."
fi

if [ "$GEN" = 1 ]; then
    # tests/gen-all.sh builds mec once, exports MEC_BIN so the fifteen generators
    # skip their own go build, and fans them out. Same fifteen checks, ~6x faster.
    # Its exit status is the verdict; the summary line is the last line.
    grc=$(rc_of gen)
    gs=$(grep -E '^gen-all: ' "$log/gen" | tail -1)
    if [ "$grc" = 0 ]; then pass "gen --check" "${gs:-ok}"
    else fail "gen --check" "${gs:-no summary} - copied to /tmp/gates-gen.log"; cp "$log/gen" /tmp/gates-gen.log; fi
fi

if [ "$BENCH" = 1 ]; then
    # THE GATE THAT WOULD HAVE CAUGHT a8e6aa2's PARENT. Every one of the eight gates
    # above checks CORRECTNESS, and all of them were green while a merge in 120ba0f
    # cost python 41% and ruby 11% for four commits - because a 41% slowdown is not
    # a wrong answer. bench.sh existed and nothing ran it. This is that gap.
    tests/bench.sh --draws 5 >"$log/bench" 2>&1; brc=$?
    bs=$(grep -E 'no regression|REGRESSED' "$log/bench" | tail -1)
    if [ "$brc" = 0 ]; then pass bench "${bs:-ok}"
    else fail bench "${bs:-see the rows} - copied to /tmp/gates-bench.log"; cp "$log/bench" /tmp/gates-bench.log; fi
fi

if [ "$FREEZE" = 1 ]; then
    # STRICTLY SERIAL, and after the wait above: -freeze REWRITES the two files
    # every other gate reads, so it cannot overlap any of them.
    # Both artefacts: -freeze writes the IR snapshot AND the frozen grammar.
    sum() { cat abnf/jsbootstrap.ll abnf/jsagrammar.go | (md5 -q 2>/dev/null || md5sum | cut -d' ' -f1); }
    before=$(sum)
    ./mec -freeze languages/metajs-to-llvm-ir.abnf >/dev/null 2>&1
    after=$(sum)
    if [ "$before" = "$after" ]; then pass "-freeze" "fixed point (identical)"
    else fail "-freeze" "SNAPSHOT WAS STALE - it changed; re-run to confirm the new one is a fixed point"; fi
fi

echo
if [ ${#FAILED[@]} -eq 0 ]; then
    echo "ALL GATES GREEN$([ "$QUICK" = 1 ] && echo " (layer 2 NOT covered: --quick)")"
    exit 0
fi
echo "FAILED: ${FAILED[*]}"
exit 1
