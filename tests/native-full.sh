#!/usr/bin/env bash
#
# native-full.sh - build every tests/<lang>-test-full.<ext> as a NATIVE -exe and
# run it.
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
# tests/clang-check.sh is the only other gate that links layer 2, and it runs one
# small program per language. This runs the ratchet files - the same assertions
# --full counts, through the code that actually ships in a binary.
#
# A binary that CRASHES looks fast and prints nothing, so this checks the exit
# status AND the self-check line, and deletes the previous binary first.
#
# Usage: tests/native-full.sh [lang ...]        (default: every language)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

# <lang>:<extension of tests/<lang>-test-full.*>
ROWS="bash:sh c:c csharp:cs dart:dart go:go java:java js:js kotlin:kt lua:lua
      metajs:js php:php python:py ruby:rb swift:swift typescript:ts"

WANT="${*:-}"
rc=0
run=0
printf "%-12s %-6s %s\n" LANGUAGE EXIT RESULT
printf -- "------------------------------------------------------------\n"
for row in $ROWS; do
    L="${row%%:*}"
    e="${row##*:}"
    if [ -n "$WANT" ]; then
        case " $WANT " in *" $L "*) ;; *) continue;; esac
    fi
    f="tests/$L-test-full.$e"
    [ -f "$f" ] || { printf "%-12s %-6s %s\n" "$L" "-" "no $f"; continue; }
    run=$((run + 1))
    bin="$OUT/$L.out"
    rm -f "$bin"
    if ! ./mec "languages/$L-to-llvm-ir.abnf" "$f" -q -exe "$bin" >/dev/null 2>"$OUT/$L.err"; then
        printf "%-12s %-6s %s\n" "$L" "-" "BUILD FAILED"
        tail -3 "$OUT/$L.err"
        rc=1
        continue
    fi
    if [ ! -x "$bin" ]; then
        printf "%-12s %-6s %s\n" "$L" "-" "no binary written"
        rc=1
        continue
    fi
    "$bin" > "$OUT/$L.txt" 2>&1
    x=$?
    line="$(grep -E 'checks, [0-9]+ failure' "$OUT/$L.txt" | tail -1)"
    [ -n "$line" ] || line="NO SELF-CHECK LINE - the binary printed nothing usable"
    printf "%-12s %-6s %s\n" "$L" "$x" "$line"
    grep -E '^FAIL' "$OUT/$L.txt" | sed 's/^/    /'
    case "$x:$line" in
        0:*", 0 failures"*) ;;
        *) rc=1;;
    esac
done
printf -- "------------------------------------------------------------\n"
if [ "$rc" = 0 ]; then
    echo "$run languages, every native binary exits 0 with 0 failures"
else
    echo "FAILURES - see above"
fi
exit $rc
