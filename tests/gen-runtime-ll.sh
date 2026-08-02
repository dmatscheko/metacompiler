#!/usr/bin/env bash
#
# gen-runtime-ll.sh - compile languages/lib/runtime.c with OUR OWN C compiler and
# keep the module as languages/lib/runtime.ll.
#
# WHY THE .ll IS CHECKED IN
#
# llvm.BuildExecutable hands its runtime inputs straight to clang, so passing
# runtime.c would have CLANG compile the C floor - and the whole point of phase 3
# of docs/runtime-rework-plan.md is that every layer is compiled by this
# repository. So the C floor is compiled by languages/c-to-llvm-ir.abnf here, and
# what clang links is IR that came out of a grammar in languages/. runtime.ll is
# a generated artifact exactly like abnf/jsbootstrap.ll: regenerate it whenever
# runtime.c or c-to-llvm-ir.abnf changes, and commit the result.
#
# The metacompiler prints the module and THEN runs main() with the built-in IR
# interpreter; that run cannot work here (runtime.c calls jsmain/jsdispatch,
# which only exist in the MetaJS module it is linked with, and setjmp, which the
# IR interpreter has no external for). The module is the prefix of the output, so
# it is cut out with the same brace-depth scan tests/clang-check.sh uses.
#
# Usage: tests/gen-runtime-ll.sh [--check]
#   --check  regenerate into a temp file and diff, without writing (exit 1 on drift)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

BIN="$(mktemp -t mec-genrt.XXXXXX)"
OUT="$(mktemp -t mec-runtime.XXXXXX)"
trap 'rm -f "$BIN" "$OUT"' EXIT

go build -o "$BIN" . || { echo "gen-runtime-ll.sh: build failed" >&2; exit 2; }

module_only() {
    awk '
        BEGIN { depth = 0 }
        {
            if (depth > 0) { print; if ($0 ~ /^}/) depth = 0; next }
            if ($0 ~ /^define /) { print; depth = 1; if ($0 ~ /}[ \t]*$/) depth = 0; next }
            if ($0 ~ /^@/ || $0 ~ /^declare / || $0 ~ /^[ \t]*$/ ||
                $0 ~ /^source_filename/ || $0 ~ /^target / || $0 ~ /^;/ ||
                $0 ~ /^!/ || $0 ~ /^%[A-Za-z_.$]/ || $0 ~ /^attributes /) { print; next }
            exit
        }'
}

"$BIN" languages/c-to-llvm-ir.abnf languages/lib/runtime.c -q 2>/dev/null | module_only > "$OUT"

lines="$(wc -l < "$OUT" | tr -d ' ')"
if [ "$lines" -lt 100 ]; then
    echo "gen-runtime-ll.sh: only $lines lines of module - the compile failed" >&2
    "$BIN" languages/c-to-llvm-ir.abnf languages/lib/runtime.c -q 2>&1 | tail -5 >&2
    exit 1
fi

if [ "$CHECK" -eq 1 ]; then
    if diff -q "$OUT" languages/lib/runtime.ll >/dev/null 2>&1; then
        echo "runtime.ll is up to date ($lines lines)"
        exit 0
    fi
    echo "runtime.ll DIFFERS from what runtime.c compiles to - run tests/gen-runtime-ll.sh" >&2
    exit 1
fi

cp "$OUT" languages/lib/runtime.ll
echo "wrote languages/lib/runtime.ll ($lines lines)"
