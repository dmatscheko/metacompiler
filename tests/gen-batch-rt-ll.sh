#!/usr/bin/env bash
#
# gen-batch-rt-ll.sh - compile languages/lib/batch-rt.c with OUR OWN C compiler and
# keep the module as languages/lib/batch-rt.ll.
#
# WHY THE .ll IS CHECKED IN
#
# Same reason as tests/gen-runtime-ll.sh, which this is copied from:
# llvm.BuildExecutable hands its runtime inputs straight to clang, so passing
# batch-rt.c would have CLANG compile the layer - and the point of the runtime
# rework (docs/runtime-rework-plan.md) is that every layer is compiled by this
# repository. batch-rt.ll is a generated artifact exactly like abnf/jsbootstrap.ll
# and languages/lib/runtime.ll: regenerate it whenever batch-rt.c or
# c-to-llvm-ir.abnf changes, and commit the result.
#
# TWO THINGS THIS DOES THAT gen-runtime-ll.sh DOES NOT
#
#   - It STRIPS `define i32 @main`. runtime.c owns the program entry point of a
#     linked MetaJS binary; batch-rt.c does not - batch-to-llvm-ir.abnf emits the
#     real main. batch-rt.c still has to carry one, because c-to-llvm-ir.abnf
#     refuses a translation unit without a main ("C programs need a main()
#     function"), so the four-line wrapper is cut back out here.
#   - It CHECKS that no @__mec_ginit survives. The C compiler puts the store for
#     every initialized global into that function and calls it from main; with main
#     gone nothing would call it, so an initialized global would silently start life
#     as zero. No global in batch-rt.c has an initializer, and this is what keeps it
#     that way.
#
# The metacompiler prints the module and THEN runs main() with the built-in IR
# interpreter; here that just returns 0. The module is the prefix of the output, so
# it is cut out with the same brace-depth scan tests/clang-check.sh uses.
#
# Usage: tests/gen-batch-rt-ll.sh [--check]
#   --check  regenerate into a temp file and diff, without writing (exit 1 on drift)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

BIN="$(mktemp -t mec-genbrt.XXXXXX)"
OUT="$(mktemp -t mec-batchrt.XXXXXX)"
RAW="$(mktemp -t mec-batchrtraw.XXXXXX)"
trap 'rm -f "$BIN" "$OUT" "$RAW"' EXIT

go build -o "$BIN" . || { echo "gen-batch-rt-ll.sh: build failed" >&2; exit 2; }

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

# drop_main removes the whole `define i32 @main() { ... }` body, and only that one.
drop_main() {
    awk '
        BEGIN { skip = 0 }
        {
            if (skip == 1) { if ($0 ~ /^}/) skip = 0; next }
            if ($0 ~ /^define i32 @main\(\)/) { skip = 1; next }
            print
        }'
}

"$BIN" languages/c-to-llvm-ir.abnf languages/lib/batch-rt.c -q 2>/dev/null | module_only > "$RAW"

if ! grep -q '^define i32 @main()' "$RAW"; then
    echo "gen-batch-rt-ll.sh: no main() in the emitted module - the compile failed" >&2
    "$BIN" languages/c-to-llvm-ir.abnf languages/lib/batch-rt.c -q 2>&1 | tail -5 >&2
    exit 1
fi

drop_main < "$RAW" > "$OUT"

if grep -q '__mec_ginit' "$OUT"; then
    echo "gen-batch-rt-ll.sh: batch-rt.c has an INITIALIZED global - @__mec_ginit is in the" >&2
    echo "module, and with main() stripped nothing would ever call it. Give the global no" >&2
    echo "initializer and assign it from a runtime function instead." >&2
    exit 1
fi

lines="$(wc -l < "$OUT" | tr -d ' ')"
if [ "$lines" -lt 100 ]; then
    echo "gen-batch-rt-ll.sh: only $lines lines of module - the compile failed" >&2
    exit 1
fi

if [ "$CHECK" -eq 1 ]; then
    if diff -q "$OUT" languages/lib/batch-rt.ll >/dev/null 2>&1; then
        echo "batch-rt.ll is up to date ($lines lines)"
        exit 0
    fi
    echo "batch-rt.ll DIFFERS from what batch-rt.c compiles to - run tests/gen-batch-rt-ll.sh" >&2
    exit 1
fi

cp "$OUT" languages/lib/batch-rt.ll
echo "wrote languages/lib/batch-rt.ll ($lines lines)"
