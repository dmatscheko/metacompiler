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
# -q (the module is a compiler's default output, not noise), so the module has to
# be cut out of the combined stream.
#
# Do NOT do this by finding the LAST line that looks like the end of IR. That is
# the obvious approach and it is wrong, because the PROGRAM's output can contain
# such a line: php's var_dump prints a bare "}" to close an array, which pushes
# the cut past the real end and feeds program text to clang. It reports
#     error: expected top-level entity ... int(9223372036854775807)
# which reads exactly like an emitter bug and is not one.
#
# Scan forward instead, tracking brace depth. The module is a prefix: at depth 0
# only @globals, declare, define and blanks belong to it, and the first line at
# depth 0 that is none of those is where the program's output starts.
module_only() {
    awk '
        BEGIN { depth = 0 }
        {
            if (depth > 0) {                       # inside a define body
                print
                if ($0 ~ /^}/) depth = 0
                next
            }
            if ($0 ~ /^define /) { print; depth = 1; if ($0 ~ /}[ \t]*$/) depth = 0; next }
            if ($0 ~ /^@/ || $0 ~ /^declare / || $0 ~ /^[ \t]*$/ ||
                $0 ~ /^source_filename/ || $0 ~ /^target / || $0 ~ /^;/ ||
                $0 ~ /^!/ || $0 ~ /^%[A-Za-z_.$]/ || $0 ~ /^attributes /) { print; next }
            exit                                   # first non-module line: stop
        }'
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
        # Parsing is not the whole story. `clang -S -x ir` proves the module is
        # WELL-FORMED; it says nothing about whether the compiled program
        # BEHAVES the way it does under our own llvm.Run. Those can differ - on
        # 2026-07-27 the bash ratchet passed 289/289 under llvm.Run and failed 7
        # assertions in the clang-built executable, all in the regex path, a
        # defect that had been sitting behind a green matrix and a green
        # clang-check.
        #
        # For the three languages that emit self-contained IR (bash, batch, c -
        # plus the toys) we can go further and RUN it, because the ratchet file
        # is self-checking: it exits non-zero and prints its own failure count.
        # Do that. For a handle-IR language there is nothing to link, so the
        # module check is all there is and the row says so.
        # A grammar can HOLD its native row while still implementing -exe. The two
        # questions are genuinely different: `exePath` answers "does this grammar
        # link natively at all" (main.go greps for the same literal, and refuses
        # -exe without it, so a silent no-op cannot look like a native build),
        # while this row answers "does its whole ratchet PASS natively yet".
        #
        # php was the first language where those diverged, and is also the proof
        # that holding the row is the right move rather than a euphemism for
        # failure. It linked, and 221 of its 306 ratchet assertions agreed
        # byte-for-byte with llvm.Run, but the rest needed two primitives the C
        # floor did not have (object-key enumeration and coroutines - WALL 2 and
        # WALL 1 of docs/runtime-next-plan.md part 3). Leaving the row RED would
        # have destroyed its own signal: readers learn to expect the failure and
        # the next genuine regression hides behind it. So the row was held, the
        # two primitives were built, and php now passes 306/306 with the marker
        # deleted - which is all that turning a held row back on ever takes.
        #
        # No language holds the row today. The mechanism stays because the next
        # migration will need it: js, typescript and python are still to come.
        #
        # A grammar opts out with one comment line, `clang-check: native-row-held`,
        # and the row below says so rather than going quiet. Deleting that line from
        # the grammar is the whole of turning the row on.
        if grep -q 'clang-check: native-row-held' "languages/$lang-to-llvm-ir.abnf" 2>/dev/null; then
            printf '%-12s %8s  ok (module only - links natively, row HELD: see the grammar)\n' "$lang" "$lines"
            continue
        fi
        if grep -q 'exePath' "languages/$lang-to-llvm-ir.abnf" 2>/dev/null; then
            exe="$WORK/$lang.exe"
            if ! "$BIN" "$g" "$f" -q -exe "$exe" >/dev/null 2>"$ll.exe.err"; then
                printf '%-12s %8s  ok (module), BUT -exe FAILED TO BUILD\n' "$lang" "$lines"
                sed -n '1,4p' "$ll.exe.err" | sed 's/^/                       /'
                bad=$((bad + 1)); rc=1; continue
            fi
            out="$("$exe" 2>&1)"; erc=$?
            if [ "$erc" -eq 0 ]; then
                printf '%-12s %8s  ok, and the clang executable agrees\n' "$lang" "$lines"
            else
                printf '%-12s %8s  ok (module), BUT THE NATIVE RUN DISAGREES\n' "$lang" "$lines"
                printf '%s\n' "$out" | grep -E '^(FAIL|full:)' | head -8 | sed 's/^/                       /'
                printf '                       (llvm.Run passes this file; clang does not - the class -exe exists for)\n'
                bad=$((bad + 1)); rc=1
            fi
            continue
        fi
        printf '%-12s %8s  ok (module only - handle IR, nothing to link)\n' "$lang" "$lines"
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
    printf '%d modules, %d with a finding\n\n' "$n" "$bad"
    printf 'Both findings above are invisible to ./test.sh, which compares each engine\n'
    printf 'against ITSELF and is satisfied by any module llvm.Run can execute:\n'
    printf '  - CLANG REJECTS      the module is not well-formed LLVM (a block label\n'
    printf '                       colliding with a parameter name is the classic case).\n'
    printf '                       llvm.Run resolves by handle, not by name, so it runs\n'
    printf '                       anyway and the ratchet still reports FULL.\n'
    printf '  - NATIVE RUN DISAGREES  the module is well-formed and clang builds it, but\n'
    printf '                       the executable does not behave as it does under\n'
    printf '                       llvm.Run. Only the three self-contained-IR languages\n'
    printf '                       (bash, batch, c) can be checked this far at all.\n\n'
fi
exit "$rc"
