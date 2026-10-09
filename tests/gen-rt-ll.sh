#!/usr/bin/env bash
#
# gen-rt-ll.sh LANG [--check] - compile languages/lib/LANG-rt.metajs (a language's
# layer 2, its own semantics written in MetaJS) with OUR OWN MetaJS compiler and
# keep the module as languages/lib/LANG-rt.ll. LANG-to-llvm-ir.abnf links it next
# to lib/runtime.ll (the C floor) when it builds an -exe; llvm.BuildExecutable
# hands it straight to clang, so it has to be IR that came out of a grammar in
# languages/.
#
# LANG-rt.ll is a generated artifact: regenerate it whenever LANG-rt.metajs,
# lib/runtime.metajs or metajs-to-llvm-ir.abnf changes, and commit the result.
# For LANG=metajs the grammar that compiles the file is also the one that links
# its output, so a change to metajs-to-llvm-ir.abnf alone can make it stale.
#
# Under -rt-lib the compiler prints the module and exits; module_only still cuts
# at the end of the module so a stray line on stdout cannot end up in the .ll.
#
#   --check  regenerate into a temp file and diff, without writing (exit 1 on drift)
#
# MEC_BIN (set by tests/gen-all.sh) names a prebuilt mec to use instead of a
# private go build; unset, empty or not executable means build one.
set -u

LANG_="${1:-}"
[ -n "$LANG_" ] || { echo "usage: tests/gen-rt-ll.sh LANG [--check]" >&2; exit 2; }
NAME="gen-$LANG_-rt-ll.sh"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MEC_BIN_ABS=""
if [ -n "${MEC_BIN:-}" ]; then
    case "$MEC_BIN" in
        /*) [ -x "$MEC_BIN" ] && MEC_BIN_ABS="$MEC_BIN" ;;
        *)  [ -x "$PWD/$MEC_BIN" ] && MEC_BIN_ABS="$PWD/$MEC_BIN" ;;
    esac
fi

cd "$ROOT" || exit 2
SRC="languages/lib/$LANG_-rt.metajs"
DST="languages/lib/$LANG_-rt.ll"
[ -f "$SRC" ] || { echo "$NAME: no $SRC" >&2; exit 2; }

CHECK=0
[ "${2:-}" = "--check" ] && CHECK=1

OUT="$(mktemp -t "mec-$LANG_-rt.XXXXXX")"
if [ -n "$MEC_BIN_ABS" ]; then
    BIN="$MEC_BIN_ABS"
    trap 'rm -f "$OUT"' EXIT
else
    BIN="$(mktemp -t "mec-gen-$LANG_.XXXXXX")"
    trap 'rm -f "$BIN" "$OUT"' EXIT
    go build -o "$BIN" . || { echo "$NAME: build failed" >&2; exit 2; }
fi

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

"$BIN" languages/metajs-to-llvm-ir.abnf "$SRC" -q -rt-lib \
    | module_only > "$OUT" || { echo "$NAME: compile failed" >&2; exit 2; }

lines="$(wc -l < "$OUT" | tr -d ' ')"
[ "$lines" -gt 100 ] || { echo "$NAME: only $lines lines - the compile must have failed" >&2; exit 2; }

if [ "$CHECK" -eq 1 ]; then
    if diff -q "$OUT" "$DST" >/dev/null 2>&1; then
        echo "$LANG_-rt.ll is up to date ($lines lines)"
        exit 0
    fi
    echo "$LANG_-rt.ll is STALE - rerun tests/$NAME" >&2
    diff "$OUT" "$DST" | head -20 >&2
    exit 1
fi

cp "$OUT" "$DST"
echo "wrote $DST ($lines lines)"
