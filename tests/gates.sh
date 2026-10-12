#!/usr/bin/env bash
#
# gates.sh - run EVERY gate this project has, in one command, and report one
# verdict.
#
# Green means every gate, including the output greps and the RECORDED NUMBERS
# below (FULL_ASSERTIONS, FULL_GAPS, SHAPE_GROUPS, MATRIX_ENTRIES, LINT_FINDINGS):
# a gate that prints a smaller number without complaining is worse than no gate.
# ./test.sh, --full and --cross run llvm.Run, which resolves js_* to the Go twin,
# so they cannot see layer 2 (languages/lib/*-rt.metajs); only clang-check,
# native-full and a native -exe probe can.
#
# Usage:
#   tests/gates.sh              every gate (~5-6 minutes)
#   tests/gates.sh --quick      skip the two slowest (clang-check, native-full).
#                               NOT sufficient for a layer-2 change - it says so.
#   tests/gates.sh --gen        also check all checked-in .ll files reproduce from
#                               source (MANDATORY after any runtime.c /
#                               *-rt.metajs edit)
#   tests/gates.sh --freeze     also check `-freeze` is a fixed point (MANDATORY
#                               after any metajs-to-llvm-ir.abnf / compile-core.js
#                               edit). Runs LAST and alone: it rewrites
#                               abnf/jsbootstrap.ll and abnf/jsagrammar.go.
#   tests/gates.sh --serial     run the gates one at a time (low-core machine,
#                               clean timings).
#   tests/gates.sh --bench      also check native performance against the recorded
#                               baselines (tests/bench.sh). Slow, so not default.
#   tests/gates.sh --all        --gen --freeze --bench
#
# The gates run concurrently. That is safe because no two write the same path:
# test.sh and clang-check build their own binaries and work in mktemp dirs (the
# matrix's only in-repo writes are the -exe targets in .vscode/launch.json);
# native-full only reads ./mec, which is built once before the fan-out; gen-all
# --check, go test, shape-scan and metajs-lint write nothing.
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
        -h|--help) sed -n "2,36p" "$0"; exit 0 ;;
        *) echo "gates.sh: unknown flag $a" >&2; exit 2 ;;
    esac
done

# ----------------------------------------------------------------------------
# Recorded expectations. A gate that only prints a number is not a gate: a killed
# run, a deleted matrix row or a section that stops parsing all make the suite
# SMALLER while still printing "green". Each constant below turns that into a
# failure. Set them from a COMMITTED tree, never from a working tree with other
# changes in flight; the gate prints the new value when one should be raised.

# Per-run brake for ./test.sh --full. The js ratchet's -frozen interpreter leg is
# the slowest run in the suite and needs headroom under concurrent load; a kill
# is reported as RUN-FAILED, which is red for a reason that is not a defect.
FULL_TIMEOUT=900

# A FLOOR on the total ratchet assertion count (the files legitimately grow).
FULL_ASSERTIONS=12381

# A CEILING on section gaps, summed per half over all languages. A gapped half
# still reports the assertions it ran, so the total barely moves when a section
# stops parsing; this is what notices. Gaps should only ever go down.
FULL_GAPS=0

# The shape ratchet (tools/shape-scan): layer-2 bodies identical modulo names
# across files. BOTH numbers, because a body copied into a third language joins
# an existing group without moving the group count; only the recoverable-lines
# count sees it.
SHAPE_GROUPS=1
SHAPE_LINES=6

# A CEILING on tools/metajs-lint findings (MetaJS constructs that do not mean what
# they mean in JavaScript, e.g. "\f" compiling to "f"). Only the native binary can
# see most of them, so no dynamic gate is a substitute. Raise this only to accept
# a finding deliberately, and say why here.
LINT_FINDINGS=0

# A FLOOR on the matrix entry count (.vscode/launch.json), and on the abort rows
# specifically: they are the only rows that compare abort messages, and a row
# losing its "SHOULD ABORT" or "ALL HALVES" token silently downgrades it.
MATRIX_ENTRIES=402
ABORT_ROWS=19             # rows whose name says SHOULD ABORT
ABORT_ALL_ROWS=18         # of those, the ones that also compare the interpreter
                          # half (ALL HALVES)

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

# --timeout on all three: the gates run concurrently, so every run is a run at
# load and test.sh's 120 s default is too tight.
start matrix ./test.sh --timeout "$FULL_TIMEOUT"
start full   ./test.sh --full  --timeout "$FULL_TIMEOUT"
start cross  ./test.sh --cross --timeout "$FULL_TIMEOUT"
start gotest go test ./abnf/
start shape  go run ./tools/shape-scan -max "$SHAPE_GROUPS" -max-lines "$SHAPE_LINES"
# -selftest asserts every rule still fires on its own construct: a linter whose
# rules stopped matching would report zero findings, like a clean tree.
start lint   go run ./tools/metajs-lint -grammars -q
start lintst go run ./tools/metajs-lint -selftest
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

# 1. The matrix. Its own summary line is trustworthy about PASS/FAIL - but not
#    about SIZE, so the recorded counts above are checked here too.
m=$(grep -E '^matrix: ' "$log/matrix" | tail -1)
me=$(printf '%s\n' "$m" | sed -n 's/^matrix: \([0-9][0-9]*\) entries run.*/\1/p')
ar=$(grep -c '"name": ".*SHOULD ABORT' .vscode/launch.json)
aa=$(grep -c '"name": ".*SHOULD ABORT.*ALL HALVES' .vscode/launch.json)
if ! grep -q '^all green' "$log/matrix"; then
    fail matrix "${m:-NO SUMMARY} - see $log/matrix"; cp "$log/matrix" /tmp/gates-matrix.log
elif [ -z "$me" ]; then
    fail matrix "all green, but NO ENTRY COUNT in the summary - the run did not finish"
elif [ "$me" -lt "$MATRIX_ENTRIES" ]; then
    fail matrix "$m - SHORT: $((MATRIX_ENTRIES - me)) rows FEWER than the recorded $MATRIX_ENTRIES. A smaller suite is not a green one."
elif [ "$ar" -lt "$ABORT_ROWS" ] || [ "$aa" -lt "$ABORT_ALL_ROWS" ]; then
    fail matrix "$m - ABORT ROWS SHRANK: $ar SHOULD ABORT ($ABORT_ROWS recorded), $aa of them ALL HALVES ($ABORT_ALL_ROWS recorded). Those rows are the only abort-text check in the suite."
elif [ "$me" -gt "$MATRIX_ENTRIES" ] || [ "$ar" -gt "$ABORT_ROWS" ] || [ "$aa" -gt "$ABORT_ALL_ROWS" ]; then
    pass matrix "$m; abort rows $ar/$aa (over the recorded $MATRIX_ENTRIES entries, $ABORT_ROWS/$ABORT_ALL_ROWS abort rows - RAISE them in tests/gates.sh)"
else
    pass matrix "$m; abort rows $ar SHOULD ABORT, $aa of them ALL HALVES"
fi

# 2. The ratchet. Informational by design - it ALWAYS exits 0 - so the summary
#    line is not a verdict and the grep is the actual gate.
bad=$(grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF|RUN-FAILED' "$log/full")
n=$(grep -E 'assertions in total' "$log/full" | tail -1)
# A half that recorded NO ASSERTION COUNT AT ALL (test.sh's NO-COUNT: line).
nocount=$(sed -n 's/^  NO-COUNT: [0-9]* language(s) had a half that recorded NO ASSERTION COUNT AT ALL://p' "$log/full")
# Backstop: a count column of '-' means a path out of full_probe wrote no flag.
blank=$(grep -E '^  [a-z0-9]+ +-$' "$log/full" | awk '{print $1}' | tr '\n' ' ')
# Section gaps, against the recorded ceiling. No GAPS: line at all means zero.
gaps=$(sed -n 's/^  GAPS: \([0-9][0-9]*\) section-gap(s).*/\1/p' "$log/full" | tail -1)
[ -z "$gaps" ] && gaps=0
# The total itself, as a number, against what it is supposed to be.
tot=$(printf '%s\n' "$n" | sed -n 's/^ *\([0-9][0-9]*\) assertions in total.*/\1/p')
if [ -n "$nocount" ]; then
    fail --full "${n:-ran} -- NO ASSERTION COUNT AT ALL from a half of:$nocount (its assertions are missing from the total)"
elif [ -n "$bad" ]; then
    fail --full "${n:-ran} -- $(echo "$bad" | head -3)"
elif [ "$gaps" -gt "$FULL_GAPS" ]; then
    fail --full "${n:-ran} -- $gaps SECTION GAP(S), above the recorded ceiling of $FULL_GAPS. A section of a ratchet file stopped running; the total barely moves, so this line is what sees it. See the GAPS: block in the output."
    cp "$log/full" /tmp/gates-full.log
elif [ -n "$blank" ]; then
    fail --full "${n:-ran} -- RECORDED NOTHING for: $blank (a count of '-' should now be unreachable: a path out of full_probe is not writing a flag)"
elif [ -z "$tot" ]; then
    fail --full "NO ASSERTION TOTAL in the output at all - the run did not finish"
elif [ "$tot" -lt "$FULL_ASSERTIONS" ]; then
    fail --full "SHORT READ: $tot assertions, $((FULL_ASSERTIONS - tot)) BELOW the recorded $FULL_ASSERTIONS."
    printf '  %-24s %s\n' "" "A SMALLER NUMBER IS A FAILURE, not a smaller test run. Usually killed runs"
    printf '  %-24s %s\n' "" "(raise --timeout / re-run when quiet); otherwise assertions stopped executing."
    cp "$log/full" /tmp/gates-full.log
elif [ "$gaps" -lt "$FULL_GAPS" ]; then
    pass --full "${n:-ran} ($gaps section gaps, BELOW the recorded $FULL_GAPS - LOWER FULL_GAPS in tests/gates.sh)"
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

# 5. The layer-2 shape ratchet: the only gate that can see a body copy-pasted
#    into a second language.
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

# 5b. The MetaJS dialect gaps. Static, so it sees defects that show up only in
#     the native binary.
lrc=$(rc_of lintst)
ls_=$(grep -E '^metajs-lint: [0-9]+ finding' "$log/lint" | tail -1)
lf=$(printf '%s\n' "$ls_" | sed -n 's/^metajs-lint: \([0-9][0-9]*\) finding.*/\1/p')
[ -z "$lf" ] && lf=$(grep -cE '^  [a-z-]+ +languages/' "$log/lint")
if [ "$lrc" != 0 ]; then
    fail metajs-lint "SELF-TEST FAILED - a rule stopped detecting its own construct, so 0 findings would mean nothing: $(tail -3 "$log/lintst" | tr '\n' ' ')"
elif [ "$(rc_of lint)" != 0 ] && [ -z "$lf" ]; then
    fail metajs-lint "did not run - $(tail -2 "$log/lint" | tr '\n' ' ')"
elif [ "$lf" -gt "$LINT_FINDINGS" ]; then
    fail metajs-lint "$lf dialect-gap findings, ABOVE the recorded $LINT_FINDINGS - a NEW one; copied to /tmp/gates-lint.log"
    cp "$log/lint" /tmp/gates-lint.log
elif [ "$lf" -lt "$LINT_FINDINGS" ]; then
    pass metajs-lint "$lf findings (BELOW the recorded $LINT_FINDINGS - LOWER LINT_FINDINGS in tests/gates.sh), self-test ok"
else
    pass metajs-lint "$lf known findings, self-test ok"
fi

if [ "$QUICK" = 0 ]; then
    # 6 and 7 are THE ONLY GATES THAT SEE LAYER 2.
    # CLANG_MODULES is a FLOOR, like FULL_ASSERTIONS: it may grow, never shrink.
    CLANG_MODULES=20
    crc=$(rc_of clang)
    held=$(grep -c 'row HELD' "$log/clang")
    ok=$(grep -c 'the clang executable agrees' "$log/clang")
    if [ "$ok" -gt "$CLANG_MODULES" ]; then
        extra="  (+$((ok - CLANG_MODULES)), RAISE CLANG_MODULES in tests/gates.sh)"
    else extra=""; fi
    if [ "$crc" = 0 ] && [ "$ok" -ge "$CLANG_MODULES" ] && [ "$held" = 0 ]; then
        pass clang-check "$ok modules agreeing, none held$extra"
    else fail clang-check "$ok agreeing of $CLANG_MODULES expected, $held held, rc=$crc - copied to /tmp/gates-clang.log"; cp "$log/clang" /tmp/gates-clang.log; fi

    nf=$(grep -E 'languages, every native binary|native binaries' "$log/native" | tail -1)
    if grep -q 'every native binary exits 0 with 0 failures' "$log/native"; then pass native-full "$nf"
    else fail native-full "${nf:-see $log/native}"; cp "$log/native" /tmp/gates-native.log; fi
else
    printf '  %-24s %s\n' "clang-check" "SKIPPED (--quick)"
    printf '  %-24s %s\n' "native-full" "SKIPPED (--quick)"
    echo "  NOTE: --quick cannot see languages/lib/*-rt.metajs. Do not call a layer-2 change green on it."
fi

if [ "$GEN" = 1 ]; then
    # Its exit status is the verdict; the summary line is the last line.
    grc=$(rc_of gen)
    gs=$(grep -E '^gen-all: ' "$log/gen" | tail -1)
    if [ "$grc" = 0 ]; then pass "gen --check" "${gs:-ok}"
    else fail "gen --check" "${gs:-no summary} - copied to /tmp/gates-gen.log"; cp "$log/gen" /tmp/gates-gen.log; fi
fi

if [ "$BENCH" = 1 ]; then
    # Every other gate checks correctness; a slowdown is not a wrong answer.
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
