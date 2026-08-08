#!/usr/bin/env bash
#
# native-full.sh - build every self-checking test program as a NATIVE -exe and
# RUN it. Two programs per language: the RATCHET (tests/<lang>-test-full.<ext>)
# and the FEATURE MATRIX (tests/<lang>-test-features.<ext>).
#
# WHY THIS EXISTS, and it is the whole point: `llvm.Run` uses the GO TWIN
# (abnf/jsrt*.go) and never links languages/lib/<lang>-rt.ll, so ./test.sh,
# ./test.sh --full and ./test.sh --cross are ALL BLIND to layer 2. A change to a
# *-rt.metajs file cannot make any of them red.
#
# Measured, so this is not an argument: with a deliberately wrong `dot` knob in
# ruby's rtkFloDigits, `./test.sh --full --filter ruby` reports "279 assertions,
# goja and -frozen byte-identical, 0 languages with halves that disagree" and
# exits green, while the native -exe of the SAME tests/ruby-test-full.rb reports
# 4 failures (n16b n16c n16d n16e).
#
# WHY THE FEATURE FILE IS RUN HERE TOO (todo.md 4.3)
#
# The feature-matrix files ARE in .vscode/launch.json, but their native rows are
# `-exe` with no run, and `mec ... -exe PATH` only BUILDS (manual 7.10). So until
# this script grew the second row, a fix asserted ONLY in a feature-matrix file
# never executed inside a native binary in ANY gate - both layer-2 gates ran the
# ratchet and nothing else. That is not theoretical: a layer-2 `Hash#each` defect
# in ruby-rt.metajs survived all seven gates on the b3eb5c3 round and was found
# only because someone ran the binary by hand.
#
# The alternative - a matrix row that builds AND runs, like `SHOULD ABORT` does -
# was rejected on cost and on blast radius: the matrix runs every entry TWICE
# (goja and -frozen), so it would pay for 28 extra native links in the fast gate
# that people run constantly, and it needs a new marker in test.sh. Here the
# build is already the price of admission and the feature files are the SMALL
# ones - about 40% of the ratchets by line. To keep the bill down the whole
# script now fans out (MEC_JOBS, default: cores) instead of running serially.
#
# The two files are complementary, not redundant: --full's ratchet walks a
# language's whole syntax, the feature matrix is where a freshly fixed defect
# gets its assertion first.
#
# tests/clang-check.sh is the only other gate that links layer 2, and it runs the
# ratchet only.
#
# THE SECOND TABLE: PROGRAMS WITH NO SELF-CHECK LINE (todo.md 4.1)
#
# The four toys - brainfuck, calculator, lisp, tinyc - have feature files and
# `exePath`, and none of them was ever run natively. todo.md 4.1 said that was
# because none of them follows the `"... checks, N failures"` + `exit(fails)`
# protocol. MEASURED, AND TWO OF THE FOUR DO:
#
#   lisp      prints `features: 63 checks, 0 failures` and the value of its last
#             top-level form becomes the exit status (`failures`).
#   tinyc     prints `features: 44 checks, 0 failures` and `return failures` from
#             main becomes the exit status.
#
# So those two just go in ROWS below with everything else, and the item was wrong
# about them. The other two genuinely cannot:
#
#   brainfuck has no arithmetic on strings and no way to count anything - it
#             prints a fixed six-byte banner, `BF8 OK`, and any broken opcode
#             garbles it. Its exit status is always 0.
#   calculator has NO OUTPUT AT ALL: the grammar's whole result is the value of
#             the expression, delivered as the process exit status (mod 256). The
#             feature files are written as a sum of (expr - expected) terms, so
#             the right answer is 0 and any wrong one is not.
#
# EXPECT_ROWS covers those two by comparing what the binary actually produced
# against a checked-in fixture, which is the only thing left when there is no
# count to print. Each row is
#
#     <lang>:<label>:<source file>:<expectation>
#
# where <expectation> is either a fixture path - stdout must be BYTE-IDENTICAL to
# it and the exit status must be 0 - or the word `exit-only`, which says the
# program prints nothing and the exit status IS the assertion. calculator uses
# `exit-only` rather than an empty fixture so that the row states what is actually
# being checked instead of comparing two empty strings and looking like a test.
#
# Regenerate a fixture with `tests/native-full.sh --bless`, and READ THE DIFF
# before you do: blessing a wrong answer is how an expected-output gate dies.
#
# A binary that CRASHES looks fast and prints nothing, so this checks the exit
# status AND the self-check line, and deletes the previous binary first.
#
# HOLDING A ROW: a grammar that links natively but whose feature file does not
# pass natively YET opts out with one comment line - a `//` line inside the
# grammar's :script block, where clang-check's marker also lives -
#
#     // native-full: features-row-held
#
# and the row below says so rather than going quiet: a row left RED destroys its
# own signal, because readers learn to expect the failure and the next genuine
# regression hides behind it. Same mechanism and same reasoning as
# `clang-check: native-row-held` in tests/clang-check.sh. Put it in a COMMENT and
# nowhere else - a stray `; native-full: ...` at grammar top level is not a
# comment, it is a rule, and the grammar then fails to parse at all.
#
# No language holds a row today, and as of 2026-08-08 EVERY language in the table
# has both files: bash and batch were the two with no feature matrix, and both got
# one (tests/bash-test-features.sh, 286 checks against real bash 5.3 as the oracle;
# tests/batch-test-features.cmd, 87 checks, no oracle - see that file's header).
#
# Usage: tests/native-full.sh [lang ...]        (default: every language)
#        MEC_JOBS=4 tests/native-full.sh        cap the fan-out
#        tests/native-full.sh --bless [lang ...]  rewrite the EXPECT_ROWS fixtures
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

# <lang>:<extension of tests/<lang>-test-{full,features}.*>[:<kinds this language has>]
#
# The optional third field names the kinds that exist, so a language with only one
# of the two does not print a "no such file" row for the other one for ever. It is
# a statement about the repository, not an opt-out: adding the missing file is a
# one-word edit here, and leaving the field off means "both", which is still the
# default and still the thing that complains when a file goes missing.
#
#   batch    has a ratchet and `exePath` and had NEVER been in this table, though
#            tests/clang-check.sh does native-run it - todo.md 4.1's "undocumented
#            asymmetry". It is here now, with a feature file of its own written the
#            same round for the same reason bash got one: batch's layer 2 is also a
#            C runtime (languages/lib/batch-rt.c) and also had exactly one program
#            on it.
#   lisp     and tinyc are toys with a feature file and no ratchet; both follow the
#   tinyc    self-check protocol exactly (see the second-table comment above).
ROWS="bash:sh batch:cmd c:c csharp:cs dart:dart go:go java:java js:js kotlin:kt
      lua:lua metajs:js php:php python:py ruby:rb swift:swift typescript:ts
      lisp:txt:features tinyc:txt:features"

# <lang>:<label>:<source file>:<fixture path | exit-only>   - see the comment above
EXPECT_ROWS="brainfuck:features:tests/brainfuck-test-features.txt:tests/brainfuck-features.expected
             calculator:features:tests/calculator-test-features.txt:exit-only
             calculator:compiler:tests/calculator-compiler-features.txt:exit-only"

JOBS="${MEC_JOBS:-$( (sysctl -n hw.ncpu || nproc || echo 4) 2>/dev/null )}"
case "$JOBS" in ''|*[!0-9]*) JOBS=4 ;; esac
[ "$JOBS" -lt 1 ] && JOBS=1

BLESS=0
ARGS=""
for a in "$@"; do
    case "$a" in
        --bless) BLESS=1 ;;
        -h|--help) sed -n '/^# Usage:/,/^set -u$/p' "$0" | sed '$d; s/^#\{1\} \{0,1\}//'; exit 0 ;;
        -*) echo "native-full.sh: unknown flag $a" >&2; exit 2 ;;
        *) ARGS="$ARGS $a" ;;
    esac
done
WANT="${ARGS# }"

# one_program LANG KIND FILE - build it, run it, and write two files:
#   $OUT/$LANG.$KIND.row   the table line (plus any FAIL detail lines)
#   $OUT/$LANG.$KIND.rc    0 or 1
one_program() {
    L="$1"; kind="$2"; f="$3"
    tag="$L/$kind"
    bin="$OUT/$L.$kind.out"
    rm -f "$bin"
    if ! ./mec "languages/$L-to-llvm-ir.abnf" "$f" -q -exe "$bin" \
            >/dev/null 2>"$OUT/$L.$kind.err"; then
        { printf "%-20s %-6s %s\n" "$tag" "-" "BUILD FAILED"
          tail -3 "$OUT/$L.$kind.err" | sed 's/^/    /'; } > "$OUT/$L.$kind.row"
        echo 1 > "$OUT/$L.$kind.rc"; return
    fi
    if [ ! -x "$bin" ]; then
        printf "%-20s %-6s %s\n" "$tag" "-" "no binary written" > "$OUT/$L.$kind.row"
        echo 1 > "$OUT/$L.$kind.rc"; return
    fi
    "$bin" > "$OUT/$L.$kind.txt" 2>&1
    x=$?
    line="$(grep -E 'checks, [0-9]+ failure' "$OUT/$L.$kind.txt" | tail -1)"
    [ -n "$line" ] || line="NO SELF-CHECK LINE - the binary printed nothing usable"
    { printf "%-20s %-6s %s\n" "$tag" "$x" "$line"
      grep -E '^FAIL' "$OUT/$L.$kind.txt" | sed 's/^/    /'; } > "$OUT/$L.$kind.row"
    case "$x:$line" in
        0:*", 0 failures"*) echo 0 > "$OUT/$L.$kind.rc" ;;
        *)                  echo 1 > "$OUT/$L.$kind.rc" ;;
    esac
}

# one_expect LANG LABEL FILE EXPECTATION - the EXPECT_ROWS variant: no self-check
# line to grep, so the whole binary's stdout is compared against a fixture (or, for
# `exit-only`, the exit status alone is the verdict and any output at all is a
# finding). Writes the same two files as one_program.
one_expect() {
    L="$1"; kind="$2"; f="$3"; want="$4"
    tag="$L/$kind"
    bin="$OUT/$L.$kind.out"
    rm -f "$bin"
    if ! ./mec "languages/$L-to-llvm-ir.abnf" "$f" -q -exe "$bin" \
            >/dev/null 2>"$OUT/$L.$kind.err"; then
        { printf "%-20s %-6s %s\n" "$tag" "-" "BUILD FAILED"
          tail -3 "$OUT/$L.$kind.err" | sed 's/^/    /'; } > "$OUT/$L.$kind.row"
        echo 1 > "$OUT/$L.$kind.rc"; return
    fi
    "$bin" > "$OUT/$L.$kind.txt" 2>&1
    x=$?
    if [ "$want" = exit-only ]; then
        sz=$(wc -c < "$OUT/$L.$kind.txt" | tr -d ' ')
        if [ "$x" = 0 ] && [ "$sz" = 0 ]; then
            printf "%-20s %-6s %s\n" "$tag" "$x" \
                   "exit-only: 0, and it printed nothing (the exit status IS the result)" \
                   > "$OUT/$L.$kind.row"
            echo 0 > "$OUT/$L.$kind.rc"
        else
            { printf "%-20s %-6s %s\n" "$tag" "$x" \
                     "FAIL exit-only: wanted exit 0 and no output, got exit $x and $sz bytes"
              head -3 "$OUT/$L.$kind.txt" | sed 's/^/    /'; } > "$OUT/$L.$kind.row"
            echo 1 > "$OUT/$L.$kind.rc"
        fi
        return
    fi
    if [ "$BLESS" = 1 ]; then
        cp "$OUT/$L.$kind.txt" "$want"
        printf "%-20s %-6s %s\n" "$tag" "$x" "BLESSED $want ($(wc -c < "$want" | tr -d ' ') bytes) - READ THE DIFF" \
               > "$OUT/$L.$kind.row"
        echo 0 > "$OUT/$L.$kind.rc"; return
    fi
    if [ ! -f "$want" ]; then
        printf "%-20s %-6s %s\n" "$tag" "$x" "no fixture $want - run tests/native-full.sh --bless" \
               > "$OUT/$L.$kind.row"
        echo 1 > "$OUT/$L.$kind.rc"; return
    fi
    if [ "$x" = 0 ] && cmp -s "$OUT/$L.$kind.txt" "$want"; then
        printf "%-20s %-6s %s\n" "$tag" "$x" \
               "output matches $want byte-for-byte ($(wc -c < "$want" | tr -d ' ') bytes)" \
               > "$OUT/$L.$kind.row"
        echo 0 > "$OUT/$L.$kind.rc"
    else
        { printf "%-20s %-6s %s\n" "$tag" "$x" "FAIL: output differs from $want (or exit != 0)"
          diff "$want" "$OUT/$L.$kind.txt" | head -6 | sed 's/^/    /'; } > "$OUT/$L.$kind.row"
        echo 1 > "$OUT/$L.$kind.rc"
    fi
}

# --- fan out -----------------------------------------------------------------
langs=""
running=0
for row in $ROWS; do
    L="${row%%:*}"
    rest="${row#*:}"
    e="${rest%%:*}"
    case "$rest" in *:*) kinds="${rest#*:}" ;; *) kinds="full features" ;; esac
    if [ -n "$WANT" ]; then
        case " $WANT " in *" $L "*) ;; *) continue;; esac
    fi
    langs="$langs $L:$e:$(echo $kinds | tr ' ' ,)"
    for kind in $kinds; do
        f="tests/$L-test-$kind.$e"
        if [ ! -f "$f" ]; then
            printf "%-20s %-6s %s\n" "$L/$kind" "-" "no $f" > "$OUT/$L.$kind.row"
            echo skip > "$OUT/$L.$kind.rc"
            continue
        fi
        if [ "$kind" = features ] &&
           grep -q 'native-full: features-row-held' "languages/$L-to-llvm-ir.abnf" 2>/dev/null; then
            printf "%-20s %-6s %s\n" "$L/$kind" "-" \
                   "row HELD by the grammar (see languages/$L-to-llvm-ir.abnf)" \
                   > "$OUT/$L.$kind.row"
            echo skip > "$OUT/$L.$kind.rc"
            continue
        fi
        if [ "$running" -ge "$JOBS" ]; then wait -n 2>/dev/null || wait; running=$((running - 1)); fi
        one_program "$L" "$kind" "$f" &
        running=$((running + 1))
    done
done

# the second table: the toys, checked against a fixture or by exit status alone
expects=""
for row in $EXPECT_ROWS; do
    L="${row%%:*}"
    rest="${row#*:}"
    kind="${rest%%:*}"
    rest="${rest#*:}"
    f="${rest%%:*}"
    want="${rest#*:}"
    if [ -n "$WANT" ]; then
        case " $WANT " in *" $L "*) ;; *) continue;; esac
    fi
    expects="$expects $L:$kind"
    if [ ! -f "$f" ]; then
        printf "%-20s %-6s %s\n" "$L/$kind" "-" "no $f" > "$OUT/$L.$kind.row"
        echo skip > "$OUT/$L.$kind.rc"
        continue
    fi
    if [ "$running" -ge "$JOBS" ]; then wait -n 2>/dev/null || wait; running=$((running - 1)); fi
    one_expect "$L" "$kind" "$f" "$want" &
    running=$((running + 1))
done
wait

# --- report, in a fixed order regardless of completion order ------------------
rc=0
langs_run=0
progs=0
held=0
printf "%-20s %-6s %s\n" PROGRAM EXIT RESULT
printf -- "------------------------------------------------------------------\n"
tally() {   # tally LANG KIND
    [ -f "$OUT/$1.$2.row" ] || return 0
    cat "$OUT/$1.$2.row"
    case "$(cat "$OUT/$1.$2.rc")" in
        0)    progs=$((progs + 1)) ;;
        skip) grep -q 'row HELD' "$OUT/$1.$2.row" && held=$((held + 1)) ;;
        *)    progs=$((progs + 1)); rc=1 ;;
    esac
}
for triple in $langs; do
    L="${triple%%:*}"
    kinds="$(echo "${triple##*:}" | tr , ' ')"
    langs_run=$((langs_run + 1))
    for kind in $kinds; do tally "$L" "$kind"; done
done
seen=""
for pair in $expects; do
    L="${pair%%:*}"
    # calculator has two rows; a language is counted once
    case " $seen " in *" $L "*) ;; *) seen="$seen $L"; langs_run=$((langs_run + 1)) ;; esac
    tally "$L" "${pair##*:}"
done
printf -- "------------------------------------------------------------------\n"
printf "%d native programs run (ratchet + feature matrix + fixtures), %d rows held, %d jobs\n" \
       "$progs" "$held" "$JOBS"
if [ "$rc" = 0 ]; then
    # gates.sh greps this exact phrase - do not reword it.
    echo "$langs_run languages, every native binary exits 0 with 0 failures"
else
    echo "FAILURES - see above"
fi
exit $rc
