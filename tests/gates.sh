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
# So the rule this script encodes: green means EVERY gate, including the greps AND
# the RECORDED NUMBERS below (FULL_ASSERTIONS, FULL_GAPS, SHAPE_GROUPS,
# MATRIX_ENTRIES, LINT_FINDINGS) - a gate that prints a smaller number without
# complaining is worse than no gate, and this one used to do exactly that.
#
# 2026-08-10 (docs/todo.md 6.1): two of the greps below were only correct BY
# OVERLAP. A ratchet half that recorded no assertion count printed `-`, which the
# `blank=` grep caught when BOTH halves did it and which the MISMATCH grep caught
# when only one did (`MISMATCH: - 384`). Neither grep was aimed at the class and
# neither said what had happened. test.sh now names it - a `NO-COUNT:` line - and
# the check below reads that line. The same change makes a gapped half report the
# assertions it really ran instead of nothing, which is why FULL_GAPS exists: the
# total can no longer be the thing that notices a section falling out.
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
#   metajs-lint        reads languages/lib/* and languages/*.abnf; writes nothing.
#                      Runs twice: once over the corpus, once as -selftest.
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
        -h|--help) sed -n "2,48p" "$0"; exit 0 ;;
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
#
# 2026-08-10: 600 -> 900. Measured, not guessed: the js ratchet's `-frozen`
# interpreter leg takes 456 s AT BASE on this machine and 482 s after the
# fifteenth round's additions - 80% of a 600 s budget, before any concurrent
# load. Under five agents it was killed outright in four separate gate runs, and
# every one of those printed `js MISMATCH: RUN-FAILED`, i.e. a RED GATE FOR A
# REASON THAT IS NOT A DEFECT. That is the same class this constant exists to
# prevent, one level up: a brake tight enough to fire on load teaches people to
# ignore the gate. If js grows much further the answer is to split its ratchet,
# not to keep raising this.
FULL_TIMEOUT=900
#
# SET IT FROM A COMMITTED TREE, NOT A WORKING ONE. On 2026-08-08 this was
# deliberately left at the parent's 7625 while the tree measured 7720, because
# other agents' ratchet additions were in flight and uncommitted - a floor must not
# be set from a tree that may not be the one that gets committed. The gate says
# "+N, RAISE FULL_ASSERTIONS" until someone does, and that message IS the ratchet
# doing its job rather than a nag.
#
# 2026-08-09: raised to 8123 AFTER the ten-item round was committed
# (d945bb9..d0f4db3, +357). Six agents contributed to that number and each one
# correctly refused to raise it themselves for the reason above.
FULL_ASSERTIONS=9219      # SEVENTEENTH round, +149 over the sixteenth's 9070:
                          # ruby +47 (Float/Integer round/floor/ceil with a digit
                          # count, and malformed format strings raising MRI's own
                          # ArgumentError messages), swift +44 (nineteen Array and
                          # String members, ranges as values, and one abort
                          # wording), js +20 and typescript +20 (private names
                          # mangled identically in all four engines, a total
                          # js_jskeys, and the ES2022 brand check), kotlin +18
                          # (out-of-range indexing throwing the JVM's exceptions,
                          # and nested throwable names).
                          #
                          # SIX AGENTS RAN IN THIS ROUND and every one of them saw
                          # "+N, RAISE FULL_ASSERTIONS" and left it alone, which is
                          # the twenty-sixth through thirty-first in a row. Raised
                          # here afterwards from a COMMITTED tree (d5360cc),
                          # measured by ONE whole-tree sweep of all six changes
                          # together rather than by adding up the isolated runs:
                          # 9219 is exactly 9070 + 47 + 44 + 20 + 20 + 18, which is
                          # the evidence that no change interfered with another.
                          # Each was also verified alone, in a tree built from a
                          # clean archive with only that agent's files copied in.

# SECTION GAPS in the ratchet, summed per half over all sixteen languages.
#
# docs/todo.md 6.1. Until 2026-08-10 a half with ANY gap recorded NO COUNT: the
# per-language line read `-`, the language contributed ZERO to the total, and one
# unparsable line in tests/lua-test-full.lua took 384 assertions off the number
# this project is gated on. MEASURED at db6bf43, both shapes:
#
#   gap in BOTH halves       `lua          -`            total 0, "0 halves disagree"
#   gap in ONE half          `lua MISMATCH: - 384`       total 0
#
# The first was caught by gates.sh only because a count of `-` matched the
# `blank=` grep; the second only because `- 384` happens to print the word
# MISMATCH. Two greps overlapping is not a design. test.sh now records the count,
# the verdict and the gap count as SEPARATE fields, so a gapped half contributes
# the assertions it really ran (375, above) and says PARTIAL - which means the
# TOTAL no longer moves much when a section stops parsing, and the total can
# therefore no longer be the thing that notices. This constant is what notices.
#
# A CEILING, not a floor, and the opposite direction from FULL_ASSERTIONS: gaps
# should only ever go down. 0 is not aspirational - all sixteen languages run
# their whole ratchet today, on both halves.
FULL_GAPS=0

SHAPE_GROUPS=15           # recorded 2026-08-08 at 6f60337
SHAPE_LINES=481           # recoverable lines, same commit

# MetaJS DIALECT-GAP findings (docs/todo.md 6.4), a CEILING.
#
# docs/abnf-dialect-gotchas.md lists a dozen ways MetaJS is not JavaScript and
# NOTHING ENFORCED ANY OF THEM until tools/metajs-lint. Every gap in that list is
# discovered the same way: a confusing failure under -frozen only, or - for the
# `\f` one - in the NATIVE BINARY ONLY, where the matrix, --full and --cross are
# all structurally blind. That is the exact profile of a defect that needs a
# static check, because no dynamic gate here can have one.
#
# ALL THREE WERE REAL AND ALL THREE ARE NOW FIXED. Both were confirmed by running
# them, and both are kept here because the class will recur:
#
#   languages/lib/ruby-rt.metajs  rbIsSpaceCh tested c == "\f", which compiles to
#       c == "f", so the NATIVE ruby binary stripped the letter f:
#       "ffoo barf".strip was "oo bar" natively and "ffoo barf" in the
#       interpreter and in real Ruby. The vertical tab in the SAME EXPRESSION was
#       already spelled \u000b, which is how the broken one survived.
#   languages/lib/regex.js:277    rxIsMetaCh's meta set contained \f and \v, which
#       become the letters f and v. Measured: kotlin
#       Regex(Regex.escape("f")+Regex.escape("v")+Regex.escape("x")) matching
#       "fvx" was true under goja, FALSE under -frozen and FALSE in the native
#       binary, where real Kotlin says true. abnf/jsrtregex.go is a line-by-line
#       port of that file and is written in Go, where \f and \v are real - so the
#       TWIN was right and only the MetaJS side was wrong, which is exactly why no
#       dynamic gate could see it.
#
# Raise this ONLY to record a finding you have decided to accept, and say why
# here. It must never go up silently.
LINT_FINDINGS=0

# The matrix ENTRY COUNT, for the same reason FULL_ASSERTIONS exists: the matrix
# is .vscode/launch.json, so a row that is deleted, renamed out of its mode, or
# commented out makes the suite SMALLER and still prints "all green". That is the
# failure mode this whole file is written against, and until now the matrix was
# the one gate with no recorded number at all.
#
# It matters more since the abort-text rows landed (docs/todo.md 4.1), because
# those rows are the ONLY thing in the suite that can see an abort message: a
# "SHOULD ABORT" row silently losing its "ALL HALVES" token, or an -exe row
# losing the "SHOULD ABORT" words, downgrades a real comparison to "does this
# link" without failing anything.
#
# A FLOOR, not an equality - rows are added often, and the gate says so.
MATRIX_ENTRIES=383        # 2026-08-09: 357 + 26 abort-text rows (13 languages x
                          # {interpreter SHOULD FAIL, native SHOULD ABORT}); the
                          # three MetaJS SHOULD ABORT rows were promoted in place.
                          #
                          # THE TREE IS AT 385 as this is written (docs/todo.md 6.2
                          # gave swift's two abort probe FILES the full pair of rows
                          # each, +2), and this floor is deliberately left at 383 for
                          # the reason FULL_ASSERTIONS states above: a floor is set
                          # from a COMMITTED tree, by whoever commits, not from a
                          # working one with other agents' rows in flight. The gate
                          # says "RAISE them" until then, and that message IS the
                          # ratchet working.
# And the abort rows specifically, since the count above cannot tell WHICH 383.
ABORT_ROWS=17             # rows whose name says SHOULD ABORT
ABORT_ALL_ROWS=15         # of those, the ones that also compare the interpreter
                          # half (ALL HALVES). Raising this is how a language
                          # closes its interpreter-vs-compiler abort divergence.
                          # 2026-08-10: 16/14 -> 17/15, docs/todo.md 6.2. Swift's
                          # array-index probe carried ONLY an interpreter SHOULD
                          # FAIL row, because its two compiler halves used to
                          # disagree ("uncaught exception: list index out of range"
                          # under llvm.Run against "list index out of range"
                          # natively); it has the native ALL HALVES row now, and the
                          # throw probe gained the interpreter row it was missing,
                          # so BOTH files carry the full pair and nothing was
                          # traded away. Closing docs/todo.md 1.1 was not what made
                          # this possible - measured at 22e7b44, all five legs of
                          # both files already agreed, so the two file headers were
                          # the last place the old divergence survived.

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
# Two runs, and the second one is the point. -selftest asserts every rule still
# fires on its own construct: a linter whose rules have stopped matching reports
# ZERO FINDINGS and is indistinguishable from a clean tree, which is the same
# false-green shape every recorded number in this file exists to prevent. Both
# are sub-second after the first go build.
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
# A half that recorded NO ASSERTION COUNT AT ALL. test.sh names this case
# explicitly now - `NO-COUNT: <n> language(s) had a half that recorded NO
# ASSERTION COUNT AT ALL: <names>` - and THIS GREP IS THE DESIGNED DETECTOR.
# Before it, the class was caught only by two greps overlapping: both halves
# silent printed a count of `-` and matched the `blank=` line below, one half
# silent printed `MISMATCH: - 384` and matched the `bad=` line above. Neither
# was aimed at it, and neither says what happened.
nocount=$(sed -n 's/^  NO-COUNT: [0-9]* language(s) had a half that recorded NO ASSERTION COUNT AT ALL://p' "$log/full")
# The old detector, kept as a backstop rather than as the reason: a language
# whose count column is literally '-' should now be impossible (test.sh writes a
# named flag for every exit path out of the probe), so if this ever fires again
# it means a NEW path out of full_probe forgot to record one.
blank=$(grep -E '^  [a-z0-9]+ +-$' "$log/full" | awk '{print $1}' | tr '\n' ' ')
# Section gaps, against the recorded ceiling. The total can no longer be relied
# on to notice these - that is the POINT of the 6.1 fix, since a gapped half now
# contributes its real count - so they are checked on their own line. No GAPS:
# line at all means zero.
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

# 5b. The MetaJS dialect gaps (docs/todo.md 6.4). Static, so unlike every other
#     gate here it can see the `\f` class at all - that one is wrong in the
#     native binary only, and no dynamic gate in this repository reaches it.
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
