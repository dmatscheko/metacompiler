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
# No language holds a row today: all fourteen feature files that exist build and
# pass natively (measured 2026-08-08, 2,719 further assertions). bash is the only
# language here with no feature file at all, and its row says that too.
#
# Usage: tests/native-full.sh [lang ...]        (default: every language)
#        MEC_JOBS=4 tests/native-full.sh        cap the fan-out
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

# <lang>:<extension of tests/<lang>-test-{full,features}.*>
ROWS="bash:sh c:c csharp:cs dart:dart go:go java:java js:js kotlin:kt lua:lua
      metajs:js php:php python:py ruby:rb swift:swift typescript:ts"

JOBS="${MEC_JOBS:-$( (sysctl -n hw.ncpu || nproc || echo 4) 2>/dev/null )}"
case "$JOBS" in ''|*[!0-9]*) JOBS=4 ;; esac
[ "$JOBS" -lt 1 ] && JOBS=1

WANT="${*:-}"

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

# --- fan out -----------------------------------------------------------------
langs=""
running=0
for row in $ROWS; do
    L="${row%%:*}"
    e="${row##*:}"
    if [ -n "$WANT" ]; then
        case " $WANT " in *" $L "*) ;; *) continue;; esac
    fi
    langs="$langs $L:$e"
    for kind in full features; do
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
wait

# --- report, in a fixed order regardless of completion order ------------------
rc=0
langs_run=0
progs=0
held=0
printf "%-20s %-6s %s\n" PROGRAM EXIT RESULT
printf -- "------------------------------------------------------------------\n"
for pair in $langs; do
    L="${pair%%:*}"
    langs_run=$((langs_run + 1))
    for kind in full features; do
        [ -f "$OUT/$L.$kind.row" ] || continue
        cat "$OUT/$L.$kind.row"
        case "$(cat "$OUT/$L.$kind.rc")" in
            0)    progs=$((progs + 1)) ;;
            skip) grep -q 'row HELD' "$OUT/$L.$kind.row" && held=$((held + 1)) ;;
            *)    progs=$((progs + 1)); rc=1 ;;
        esac
    done
done
printf -- "------------------------------------------------------------------\n"
printf "%d native programs run (ratchet + feature matrix), %d rows held, %d jobs\n" \
       "$progs" "$held" "$JOBS"
if [ "$rc" = 0 ]; then
    # gates.sh greps this exact phrase - do not reword it.
    echo "$langs_run languages, every native binary exits 0 with 0 failures"
else
    echo "FAILURES - see above"
fi
exit $rc
