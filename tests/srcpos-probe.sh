#!/usr/bin/env bash
#
# srcpos-probe.sh - does every compiler grammar attribute a statement to the
# line the statement is ON?
#
# WHY THIS EXISTS. The compilers emit a js_srcpos marker per statement
# (lib/compile-core.js stmtPos, active under -trace / -cfgraph / -callgraph /
# -why), and FOUR tools read those markers: the event stream's "line" field, the
# CFG's per-block line ranges, the call graph's definition lines, and -why's
# per-statement attribution. Nothing in the matrix, --full or --cross looks at
# any of them, so a wrong marker is invisible to every other gate. The two
# defect shapes it catches: an off-by-one (statements on lines 1, 2, 3
# reported as 2, 3, 4, the last pointing PAST EOF), and a grammar that emits
# no markers at all.
#
# WHAT IT ASSERTS, and why exactly this much. Every case below is three
# CONSECUTIVE SIMPLE STATEMENTS at known lines, and the two properties checked
# are those two shapes:
#
#   1. each of those three lines is reported by a statement marker, and
#   2. NO marker points past the end of the file.
#
# Extra markers are allowed and only listed under -v. That is deliberate: a tag's
# up.pos is the position AFTER its node matched, so kotlin, java, go, dart and
# the rest attribute a COMPOUND statement (and a top-level declaration) to the
# line where its body ENDS, while python, ruby and php - which capture the
# statement's FIRST token - attribute it to its header. Both are defensible,
# they differ per language today, and this probe is not the place to settle it.
#
# A language in the list below that emits NO markers at all is a FAILURE: the
# list is exactly the handle-IR compilers, all of which are supposed to have
# them. (The self-contained-IR ones - c, bash, batch, the toys - are not listed.)
#
# Usage: tests/srcpos-probe.sh [-v] [LANG ...]

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

VERBOSE=0
WANT=()
for a in "$@"; do
    case "$a" in
        -v|--verbose) VERBOSE=1 ;;
        -h|--help) sed -n "2,34p" "$0"; exit 0 ;;
        *) WANT+=("$a") ;;
    esac
done

if [ ! -x ./mec ]; then
    go build -o mec . || exit 2
fi

WORK=$(mktemp -d) || exit 2
trap 'rm -rf "$WORK"' EXIT

# LANG:EXT:the lines the three statements are on. The source is in src_for().
CASES=(
    "python:py:1 2 3"
    "ruby:rb:1 2 3"
    "php:php:2 3 4"
    "js:js:1 2 3"
    "typescript:ts:1 2 3"
    "metajs:metajs:1 2 3"
    "lua:lua:1 2 3"
    "kotlin:kt:2 3 4"
    "java:java:3 4 5"
    "csharp:cs:4 5 6"
    "swift:swift:1 2 3"
    "dart:dart:2 3 4"
    "go:go:4 5 6"
)

# The js/typescript/metajs engines run the module and then ENTER main, so the
# probe declares one - after the three statements, which keeps their lines at
# the top of the file.
src_for() {
    case "$1" in
    python) printf 'x = 1\ny = 2\nz = 3\n' ;;
    ruby)   printf 'x = 1\ny = 2\nz = 3\n' ;;
    php)    printf '<?php\n$x = 1;\n$y = 2;\n$z = 3;\n' ;;
    js|typescript) printf 'let x = 1\nlet y = 2\nlet z = 3\nfunction main() { if (x + y + z > 0) { return 0 } return 1 }\n' ;;
    metajs) printf 'var x = 1\nvar y = 2\nvar z = 3\nfunction main() { if (x + y + z > 0) { return 0 } return 1 }\n' ;;
    lua)    printf 'local x = 1\nlocal y = 2\nlocal z = 3\n' ;;
    kotlin) printf 'fun main() {\n    var x = 1\n    var y = 2\n    var z = 3\n}\n' ;;
    java)   printf 'public class Main {\n  public static void main(String[] a) {\n    int x = 1;\n    int y = 2;\n    int z = 3;\n  }\n}\n' ;;
    csharp) printf 'using System;\nclass Program {\n  static void Main() {\n    int x = 1;\n    int y = 2;\n    int z = 3;\n  }\n}\n' ;;
    swift)  printf 'var x = 1\nvar y = 2\nvar z = 3\n' ;;
    dart)   printf 'void main() {\n  var x = 1;\n  var y = 2;\n  var z = 3;\n}\n' ;;
    go)     printf 'package main\n\nfunc main() {\n\tx := 1\n\ty := 2\n\tz := 3\n\t_, _, _ = x, y, z\n}\n' ;;
    esac
}

pass=0; fail=0; skip=0
for c in "${CASES[@]}"; do
    lang=${c%%:*}; rest=${c#*:}; ext=${rest%%:*}; want=${rest#*:}
    if [ ${#WANT[@]} -gt 0 ]; then
        keep=0
        for w in "${WANT[@]}"; do [ "$w" = "$lang" ] && keep=1; done
        [ $keep = 1 ] || continue
    fi
    g="languages/$lang-to-llvm-ir.abnf"
    if [ ! -f "$g" ]; then echo "SKIP  $lang (no compiler grammar)"; skip=$((skip+1)); continue; fi
    src="$WORK/p.$ext"
    src_for "$lang" > "$src"
    nlines=$(wc -l < "$src" | tr -d ' ')
    tr="$WORK/$lang.trace"
    out=$(./mec "$g" "$src" -qq -trace "$tr" 2>&1)
    rc=$?
    if [ $rc != 0 ]; then
        echo "FAIL  $lang: the probe program did not compile/run (exit $rc)"
        [ $VERBOSE = 1 ] && echo "$out"
        fail=$((fail+1)); continue
    fi
    got=$(grep '"ev":"stmt"' "$tr" 2>/dev/null \
          | sed -E 's/.*"line":([0-9]+).*/\1/' | sort -n -u | paste -sd' ' -)
    bad=""
    if [ -z "$got" ]; then
        bad="no js_srcpos markers at all - the grammar does not apply a statement position"
    else
        for w in $want; do
            case " $got " in *" $w "*) ;; *) bad="$bad statement line $w is not reported;" ;; esac
        done
        for l in $got; do
            [ "$l" -gt "$nlines" ] && bad="$bad marker at line $l is past EOF ($nlines lines);"
        done
    fi
    if [ -z "$bad" ]; then
        pass=$((pass+1))
        [ $VERBOSE = 1 ] && echo "ok    $lang: statements on [$want], reported [$got]"
    else
        echo "FAIL  $lang: statements are on [$want], reported [$got]:$bad"
        fail=$((fail+1))
    fi
done

echo "----------------------------------------------------------------"
echo "srcpos: $pass correct, $fail wrong, $skip skipped"
[ $fail = 0 ] || exit 1
exit 0
