#!/usr/bin/env bash
#
# clang-check.sh - hand every -to-llvm-ir grammar's emitted module to clang.
#
# THE BLIND SPOT THIS CLOSES
#
# ./test.sh runs every entry twice, goja and -frozen, and demands byte-identical
# stdout. That is a strong invariant and there is a whole class of defect it
# cannot see: IR that OUR llvm.Run accepts and clang rejects. llvm.Run resolves
# branches by handle rather than by name, so an LLVM-invalid module - the classic
# case is a block label colliding with a parameter name in the same function -
# runs correctly under both engines, prints the right answer, and reports FULL.
# Both engines agree, so byte-identity is satisfied by construction. See the
# "parameter-name/block-label namespace collision" entry in
# docs/abnf-dialect-gotchas.md, which records exactly that bug being found in
# batch-to-llvm-ir.abnf long after both engines were green.
#
# WHY THIS IS NOT ./test.sh -exe
#
# -exe PATH hands the module to clang AND LINKS IT into a native executable, so
# it only works for a grammar whose IR is self-contained. Only three of the
# fifteen languages emit that: bash, batch and c (plus the toys - calculator,
# brainfuck, lisp, tinyc). The other twelve emit HANDLE IR that calls js_* externs
# implemented in Go (abnf/jsrt.go), which clang has nothing to link against, so
# -exe is not available to them and never will be. As of 2026-07-27 the matrix
# invoked -exe zero times, so in practice NO language was clang-checked at all.
#
# The fix is to stop at compile: `clang -S -x ir` parses, verifies and codegens
# the module without linking, so undefined externs are simply declarations and
# stay undefined. That gets every one of the fifteen languages checked, which is
# the whole point - the collision class lives in the emitter, and eleven of the
# twelve emitters it can afflict had no check on them whatsoever.
#
# The input is each language's own tests/<lang>-test-full.<ext>, because that
# file is the one input designed to walk the WHOLE syntax of its language, so it
# drives the largest fraction of each emitter.
#
# Informational: prints a table and exits non-zero only if clang rejects a module.
#
# Usage:
#   tests/clang-check.sh              check every language
#   tests/clang-check.sh kotlin js    check only the named ones
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

command -v clang >/dev/null 2>&1 || { echo "clang-check.sh: no clang on PATH" >&2; exit 2; }

BIN="$(mktemp -t mec-clang.XXXXXX)"
WORK="$(mktemp -d -t mec-clangwork.XXXXXX)"
trap 'rm -rf "$BIN" "$WORK"' EXIT

echo "building compiler..."
go build -o "$BIN" . || { echo "clang-check.sh: build failed" >&2; exit 2; }

# A compiler grammar prints its MODULE and then the program's own stdout under
# -q (the module is a compiler's default output, not noise). Keep everything up
# to and including the last line that is "}" or begins with @ / declare / define,
# which is where the module ends and the program's output begins.
module_only() {
    awk 'BEGIN { n = 0 } { l[n++] = $0 }
         END { last = -1
               for (i = 0; i < n; i++)
                   if (l[i] == "}" || l[i] ~ /^declare / || l[i] ~ /^@/ || l[i] ~ /^define /) last = i
               for (i = 0; i <= last; i++) print l[i] }'
}

printf '\n%-12s %8s  %s\n' LANGUAGE LINES VERDICT
printf -- '------------------------------------------------------------\n'

rc=0 n=0 bad=0
for f in tests/*-test-full.*; do
    base="$(basename "$f")"; lang="${base%%-test-full.*}"
    if [ "$#" -gt 0 ]; then
        want=0
        for a in "$@"; do [ "$a" = "$lang" ] && want=1; done
        [ "$want" -eq 1 ] || continue
    fi
    g="languages/$lang-to-llvm-ir.abnf"
    [ -f "$g" ] || { printf '%-12s %8s  (no compiler grammar)\n' "$lang" -; continue; }
    n=$((n + 1))
    ll="$WORK/$lang.ll"
    "$BIN" "$g" "$f" -q 2>/dev/null | module_only > "$ll"
    lines="$(wc -l < "$ll" | tr -d ' ')"
    if [ "$lines" -lt 5 ]; then
        printf '%-12s %8s  no module emitted - check this, it should not happen\n' "$lang" "$lines"
        bad=$((bad + 1)); rc=1; continue
    fi
    if clang -S -x ir "$ll" -o /dev/null 2>"$ll.err"; then
        printf '%-12s %8s  ok\n' "$lang" "$lines"
    else
        printf '%-12s %8s  CLANG REJECTS\n' "$lang" "$lines"
        sed -n '1,6p' "$ll.err" | sed 's/^/                       /'
        bad=$((bad + 1)); rc=1
    fi
done

printf -- '------------------------------------------------------------\n'
if [ "$bad" -eq 0 ]; then
    printf '%d modules, all accepted by clang\n\n' "$n"
else
    printf '%d modules, %d REJECTED by clang\n\n' "$n" "$bad"
    printf 'A rejected module still runs correctly under llvm.Run and still reports\n'
    printf 'FULL, because llvm.Run resolves by handle, not by name. Fix the emitter.\n\n'
fi
exit "$rc"
