#!/usr/bin/env bash
#
# gen-bash-rt-ll.sh - compile languages/lib/bash-rt.c with OUR OWN C compiler and
# keep the module as languages/lib/bash-rt.ll.
#
# WHY THE .ll IS CHECKED IN
#
# languages/bash-to-llvm-ir.abnf splices this module into the module it emits
# (llvm.SpliceIR), and llvm.BuildExecutable would hand a runtime input straight
# to clang - so shipping the .c would have CLANG compile bash's runtime. The
# whole point of docs/runtime-rework-plan.md is that every layer is compiled by
# this repository, so the runtime is compiled by languages/c-to-llvm-ir.abnf
# here and what runs is IR that came out of a grammar in languages/. bash-rt.ll
# is a generated artifact exactly like abnf/jsbootstrap.ll: regenerate it
# whenever bash-rt.c or c-to-llvm-ir.abnf changes, and commit the result.
#
# The metacompiler prints the module and THEN runs main() with the built-in IR
# interpreter; bash-rt.c has no main of its own (the bash module supplies it), so
# that run reports a missing entry point and is ignored. The module is the prefix
# of the output, so it is cut out with the same brace-depth scan
# tests/clang-check.sh uses.
#
# Usage: tests/gen-bash-rt-ll.sh [--check]
#   --check  regenerate into a temp file and diff, without writing (exit 1 on drift)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

BIN="$(mktemp -t mec-genbashrt.XXXXXX)"
OUT="$(mktemp -t mec-bashrt.XXXXXX)"
trap 'rm -f "$BIN" "$OUT"' EXIT

go build -o "$BIN" . || { echo "gen-bash-rt-ll.sh: build failed" >&2; exit 2; }

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

# drop_main removes the placeholder `int main()` of bash-rt.c: the emitted bash
# module defines its own main, and two of them cannot live in one module.
drop_main() {
    awk '
        BEGIN { skip = 0 }
        {
            if (skip) { if ($0 ~ /^}/) skip = 0; next }
            if ($0 ~ /^define [^@]*@main\(/) { skip = 1; next }
            print
        }'
}

"$BIN" languages/c-to-llvm-ir.abnf languages/lib/bash-rt.c -q 2>/dev/null | module_only | drop_main > "$OUT"

lines="$(wc -l < "$OUT" | tr -d ' ')"
if [ "$lines" -lt 100 ]; then
    echo "gen-bash-rt-ll.sh: only $lines lines of module - the compile failed" >&2
    "$BIN" languages/c-to-llvm-ir.abnf languages/lib/bash-rt.c -q 2>&1 | tail -5 >&2
    exit 1
fi

if [ "$CHECK" -eq 1 ]; then
    if diff -q "$OUT" languages/lib/bash-rt.ll >/dev/null 2>&1; then
        echo "bash-rt.ll is up to date ($lines lines)"
        exit 0
    fi
    echo "bash-rt.ll DIFFERS from what runtime.c compiles to - run tests/gen-bash-rt-ll.sh" >&2
    exit 1
fi

cp "$OUT" languages/lib/bash-rt.ll
echo "wrote languages/lib/bash-rt.ll ($lines lines)"
