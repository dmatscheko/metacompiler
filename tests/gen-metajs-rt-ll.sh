#!/usr/bin/env bash
#
# gen-metajs-rt-ll.sh - compile languages/lib/metajs-rt.metajs with OUR OWN MetaJS
# compiler and keep the module as languages/lib/metajs-rt.ll.
#
# This is MetaJS'S OWN layer 2, the sixteenth and last - the door that
# docs/runtime-next-plan.md ("Part B, move 1: `case_map` CANNOT MOVE") named as the
# unblock for Part B. metajs-to-llvm-ir.abnf used to link lib/runtime.ll and nothing
# else, so a floor body that moved into lib/runtime.metajs reached fifteen languages'
# -exe builds and vanished from MetaJS's. Now metajs-to-llvm-ir.abnf links
# lib/metajs-rt.ll next to lib/runtime.ll like every other grammar does, and the
# compiler that builds it is itself. Exactly the same arrangement, and the same
# reason, as tests/gen-runtime-ll.sh: llvm.BuildExecutable hands its runtime
# inputs straight to clang, so what clang sees has to be IR that came out of a
# grammar in languages/.
#
# NOTE THE BOOTSTRAP CIRCLE, which is unique to this one generator: the grammar that
# compiles this file is the grammar that LINKS its output. A change to
# metajs-to-llvm-ir.abnf can therefore require a rerun of this script even when
# metajs-rt.metajs did not change, and `--check` is the thing that says so.
#
# metajs-rt.ll is a generated artifact like abnf/jsbootstrap.ll and
# languages/lib/runtime.ll: regenerate it whenever metajs-rt.metajs or
# metajs-to-llvm-ir.abnf changes, and commit the result.
#
# Under -rt-lib the compiler PRINTS the module and exits (there is no program to
# run), so - unlike the C floor's generator - no output has to be cut off the
# tail. The brace-depth scan is kept anyway, so a future warning on stdout cannot
# silently end up inside the .ll.
#
# Usage: tests/gen-metajs-rt-ll.sh [--check]
#   --check  regenerate into a temp file and diff, without writing (exit 1 on drift)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

BIN="$(mktemp -t mec-genmjs.XXXXXX)"
OUT="$(mktemp -t mec-mjsrt.XXXXXX)"
trap 'rm -f "$BIN" "$OUT"' EXIT

go build -o "$BIN" . || { echo "gen-metajs-rt-ll.sh: build failed" >&2; exit 2; }

module_only() {
    awk '
        BEGIN { depth = 0 }
        {
            if (depth > 0) {
                print
                if ($0 ~ /^}/) depth = 0
                next
            }
            if ($0 ~ /^define /) { print; depth = 1; if ($0 ~ /}[ \t]*$/) depth = 0; next }
            if ($0 ~ /^@/ || $0 ~ /^declare / || $0 ~ /^[ \t]*$/ ||
                $0 ~ /^source_filename/ || $0 ~ /^target / || $0 ~ /^;/ ||
                $0 ~ /^!/ || $0 ~ /^%[A-Za-z_.$]/ || $0 ~ /^attributes /) { print; next }
            exit
        }'
}

"$BIN" languages/metajs-to-llvm-ir.abnf languages/lib/metajs-rt.metajs -q -rt-lib \
    | module_only > "$OUT" || { echo "gen-metajs-rt-ll.sh: compile failed" >&2; exit 2; }

lines="$(wc -l < "$OUT" | tr -d ' ')"
[ "$lines" -gt 100 ] || { echo "gen-metajs-rt-ll.sh: only $lines lines - the compile must have failed" >&2; exit 2; }

if [ "$CHECK" -eq 1 ]; then
    if diff -q "$OUT" languages/lib/metajs-rt.ll >/dev/null 2>&1; then
        echo "metajs-rt.ll is up to date ($lines lines)"
        exit 0
    fi
    echo "metajs-rt.ll is STALE - rerun tests/gen-metajs-rt-ll.sh" >&2
    diff "$OUT" languages/lib/metajs-rt.ll | head -20 >&2
    exit 1
fi

cp "$OUT" languages/lib/metajs-rt.ll
echo "wrote languages/lib/metajs-rt.ll ($lines lines)"
