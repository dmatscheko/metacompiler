#!/usr/bin/env bash
#
# gen-csharp-rt-ll.sh - compile languages/lib/csharp-rt.metajs with OUR OWN MetaJS
# compiler and keep the module as languages/lib/csharp-rt.ll.
#
# This is layer 2 of docs/runtime-next-plan.md part 3 for C#: C#'s own
# semantics, written in MetaJS, compiled by languages/metajs-to-llvm-ir.abnf with
# -rt-lib into a module that csharp-to-llvm-ir.abnf links NEXT TO lib/runtime.ll (the
# C floor) when it builds an -exe. Exactly the same arrangement, and the same
# reason, as tests/gen-runtime-ll.sh: llvm.BuildExecutable hands its runtime
# inputs straight to clang, so what clang sees has to be IR that came out of a
# grammar in languages/.
#
# csharp-rt.ll is a generated artifact like abnf/jsbootstrap.ll and
# languages/lib/runtime.ll: regenerate it whenever csharp-rt.metajs or
# metajs-to-llvm-ir.abnf changes, and commit the result.
#
# Under -rt-lib the compiler PRINTS the module and exits (there is no program to
# run), so - unlike the C floor's generator - no output has to be cut off the
# tail. The brace-depth scan is kept anyway, so a future warning on stdout cannot
# silently end up inside the .ll.
#
# Usage: tests/gen-csharp-rt-ll.sh [--check]
#   --check  regenerate into a temp file and diff, without writing (exit 1 on drift)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

BIN="$(mktemp -t mec-gencs.XXXXXX)"
OUT="$(mktemp -t mec-csrt.XXXXXX)"
trap 'rm -f "$BIN" "$OUT"' EXIT

go build -o "$BIN" . || { echo "gen-csharp-rt-ll.sh: build failed" >&2; exit 2; }

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

"$BIN" languages/metajs-to-llvm-ir.abnf languages/lib/csharp-rt.metajs -q -rt-lib \
    | module_only > "$OUT" || { echo "gen-csharp-rt-ll.sh: compile failed" >&2; exit 2; }

lines="$(wc -l < "$OUT" | tr -d ' ')"
[ "$lines" -gt 100 ] || { echo "gen-csharp-rt-ll.sh: only $lines lines - the compile must have failed" >&2; exit 2; }

if [ "$CHECK" -eq 1 ]; then
    if diff -q "$OUT" languages/lib/csharp-rt.ll >/dev/null 2>&1; then
        echo "csharp-rt.ll is up to date ($lines lines)"
        exit 0
    fi
    echo "csharp-rt.ll is STALE - rerun tests/gen-csharp-rt-ll.sh" >&2
    diff "$OUT" languages/lib/csharp-rt.ll | head -20 >&2
    exit 1
fi

cp "$OUT" languages/lib/csharp-rt.ll
echo "wrote languages/lib/csharp-rt.ll ($lines lines)"
