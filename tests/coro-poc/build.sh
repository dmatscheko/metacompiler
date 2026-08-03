#!/usr/bin/env bash
#
# build.sh - the COROUTINE proof of concept
# (docs/runtime-next-plan.md, "Coroutines - the fifth-language wall").
#
# It runs tests/coro-poc/gen.ll in BOTH engines and diffs the two outputs:
#
#   llvm.Run   go test ./abnf/ -run TestCoroPoC   (abnf/coropoc_test.go)
#   native     a clang link of gen.ll with a PATCHED COPY of the C floor
#
# languages/lib/runtime.c is NEVER written to: coro-floor.py reads it and writes
# the patched text into a temp directory.  --diff prints the unified patch, which
# is what the floor's owner would apply.
#
# Usage:
#   tests/coro-poc/build.sh            build, run both engines, diff
#   tests/coro-poc/build.sh --diff     print the floor patch and exit
#   tests/coro-poc/build.sh --gc       also run all four collector modes
#   tests/coro-poc/build.sh --break    the discriminating-power table: break one
#                                      GC root at a time and see whether the PoC
#                                      notices
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT" || exit 2
HERE="tests/coro-poc"

WORK="$(mktemp -d -t mec-coro.XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

MODE="${1:-run}"

if [ "$MODE" = "--diff" ]; then
    python3 "$HERE/coro-floor.py" -o "$WORK/runtime-coro.c" || exit 2
    diff -u languages/lib/runtime.c "$WORK/runtime-coro.c"
    exit 0
fi

cat > "$WORK/modonly.awk" <<'AWK'
BEGIN { depth = 0 }
{
    if (depth > 0) { print; if ($0 ~ /^}/) depth = 0; next }
    if ($0 ~ /^define /) { print; depth = 1; if ($0 ~ /}[ \t]*$/) depth = 0; next }
    if ($0 ~ /^@/ || $0 ~ /^declare / || $0 ~ /^[ \t]*$/ ||
        $0 ~ /^source_filename/ || $0 ~ /^target / || $0 ~ /^;/ ||
        $0 ~ /^!/ || $0 ~ /^%[A-Za-z_.$]/ || $0 ~ /^attributes /) { print; next }
    exit
}
AWK

go build -o "$WORK/mec" . || { echo "build.sh: go build failed" >&2; exit 2; }

# --- the C floor, patched into a COPY and compiled by OUR OWN C compiler ----
build_floor() {   # $1 = output .ll, $2 = optional perl -0 mutation of the C
    python3 "$HERE/coro-floor.py" -o "$WORK/f.c" || return 2
    if [ -n "${2:-}" ]; then perl -0pi -e "$2" "$WORK/f.c" || return 2; fi
    "$WORK/mec" languages/c-to-llvm-ir.abnf "$WORK/f.c" 2>/dev/null \
        | awk -f "$WORK/modonly.awk" > "$1"
    [ -s "$1" ]
}

build_floor "$WORK/runtime-coro.ll" "" || { echo "build.sh: floor compile failed" >&2; exit 2; }
clang -O2 -w -o "$WORK/coro-poc.out" "$HERE/gen.ll" "$WORK/runtime-coro.ll" \
    || { echo "build.sh: clang link failed" >&2; exit 2; }

"$WORK/coro-poc.out" > "$WORK/native.out" 2>&1
NRC=$?
go test ./abnf/ -run TestCoroPoC -v 2>&1 \
    | sed -n '/^=== RUN/,/^--- /p' | grep -v '^=== RUN\|^--- \|^#' > "$WORK/gorun.out"

echo "=== llvm.Run (abnf/jsrt.go: a goroutine and two channels) ==="
cat "$WORK/gorun.out"
echo "=== native (the C floor: a pthread and one condition variable), rc=$NRC ==="
cat "$WORK/native.out"
echo "==="
if cmp -s "$WORK/gorun.out" "$WORK/native.out" && [ "$NRC" = "0" ]; then
    echo "coro PoC: BYTE-IDENTICAL in both engines ($(wc -l < "$WORK/native.out" | tr -d ' ') lines)"
else
    echo "coro PoC: DIVERGENT"
    diff "$WORK/gorun.out" "$WORK/native.out"
    exit 1
fi

if [ "$MODE" = "--gc" ] || [ "$MODE" = "--break" ]; then
    echo
    echo "--- the four collector modes, same binary ---"
    for m in off auto stress poison; do
        MEC_GC="$m" MEC_GC_STATS=1 "$WORK/coro-poc.out" > "$WORK/m.out" 2> "$WORK/m.err"
        rc=$?
        if cmp -s "$WORK/native.out" "$WORK/m.out"; then v=SAME; else v=DIVERGENT; fi
        printf '%-8s rc=%-3s %-10s %s\n' "$m" "$rc" "$v" "$(cat "$WORK/m.err")"
    done
fi

if [ "$MODE" = "--break" ]; then
    echo
    echo "--- discriminating power: one GC root broken at a time, MEC_GC=stress ---"
    brk() {
        if build_floor "$WORK/v.ll" "$2" \
           && clang -O2 -w -o "$WORK/v.out" "$HERE/gen.ll" "$WORK/v.ll" 2>/dev/null; then
            MEC_GC=stress "$WORK/v.out" > "$WORK/v.txt" 2>&1
            rc=$?
            if cmp -s "$WORK/native.out" "$WORK/v.txt"; then
                printf '%-46s rc=%-3s %s\n' "$1" "$rc" "0 of $(wc -l < "$WORK/native.out" | tr -d ' ')"
            else
                printf '%-46s rc=%-3s %s\n' "$1" "$rc" \
                    "DIFFERS on $(diff "$WORK/native.out" "$WORK/v.txt" | grep -c '^<') lines"
            fi
        else
            printf '%-46s BUILD FAILED\n' "$1"
        fi
    }
    brk "a suspended coroutine stack not scanned" \
        's/if \(cb\[1\] == 1\) \{\n\t\t\tgc_scan_range\(cb\[4\], cb\[5\]\);/if (0) {\n\t\t\tgc_scan_range(cb[4], cb[5]);/'
    brk "the parked registers not scanned" \
        's/gc_scan_range\(cb\[6\], cb\[6\] \+ 512\);/;/'
    brk "the RESUMER stacks not scanned" \
        's/\t\tgc_scan_range\(RES_LO\[i\], RES_HI\[i\]\);/\t\t;/'
    brk "the RESUMER registers not scanned" \
        's/\t\tgc_scan_range\(RES_JB\[i\], RES_JB\[i\] \+ 512\);/\t\t;/'
    brk "the generator cell not a root" \
        's/\t\tgc_try\(cb\[7\], 1\);/\t\t;/'
    brk "tag 15 children not traced" \
        's/if \(t == 15\) \{ gc_try\(w\[1\], 1\); gc_try\(w\[2\], 1\); gc_try\(w\[4\], 1\); gc_try\(w\[6\], 1\); return; \}/if (t == 15) { return; }/'
    brk "tag 16 (the wrapped closure) not traced" \
        's/if \(t == 16\) \{ gc_try\(w\[1\], 1\); return; \}/if (t == 16) { return; }/'
    brk "GC_STACK_BASE not switched on resume" \
        's/\tGC_STACK_BASE = cb\[5\];\n\tg = cb\[7\];/\tg = cb[7];/'
    brk "GC_STACK_BASE not switched back on yield" \
        's/\tGC_STACK_BASE = cb\[5\];\n\tCUR_GEN = g;/\tCUR_GEN = g;/'
    brk "CUR_GEN not a root" \
        's/\tgc_try\(CUR_GEN, 1\);/\t;/'
fi
