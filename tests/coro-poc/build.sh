#!/usr/bin/env bash
#
# build.sh - the COROUTINE proof of concept
# (docs/runtime-next-plan.md, "Coroutines - the fifth-language wall").
#
# It runs tests/coro-poc/gen.ll in BOTH engines and diffs the two outputs:
#
#   llvm.Run   go test ./abnf/ -run TestCoroPoC   (abnf/coropoc_test.go)
#   native     a clang link of gen.ll with a COPY of the C floor
#
# THE PATCH HAS LANDED.  coro-floor.py used to patch a COPY of runtime.c because
# the floor had no coroutine; the floor owns one now, so coro-floor.py only makes
# the copy (and fails loudly if the coroutine block ever leaves the floor), and
# --diff prints nothing.  The copy still matters: --break mutates it, and the
# real languages/lib/runtime.c is never written to by this script.
#
# Usage:
#   tests/coro-poc/build.sh            build, run both engines, diff
#   tests/coro-poc/build.sh --diff     the floor patch (empty since it landed)
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
    echo "(empty: the coroutine primitive is IN languages/lib/runtime.c)" >&2
    python3 "$HERE/coro-floor.py" --check || exit 2
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

# --- the C floor, copied and compiled by OUR OWN C compiler ----------------
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
        's/\t\tgc_scan_range\(lo\[i\], hi\[i\]\);/\t\t;/'
    brk "the RESUMER registers not scanned" \
        's/\t\tgc_scan_range\(jb\[i\], jb\[i\] \+ 512\);/\t\t;/'
    brk "the generator cell not a root" \
        's/\t\tgc_try\(cb\[7\], 1\);/\t\t;/'
    brk "tag 15 children not traced" \
        's/if \(t == 15\) \{ gc_try\(w\[1\], 1\); gc_try\(w\[2\], 1\); gc_try\(w\[4\], 1\); gc_try\(w\[6\], 1\); return; \}/if (t == 15) { return; }/'
    brk "tag 16 (the wrapped closure) not traced" \
        's/if \(t == 16\) \{ gc_try\(w\[1\], 1\); return; \}/if (t == 16) { return; }/'
    brk "GC_STACK_BASE not switched on resume" \
        's/\tGC_STACK_BASE = cb\[5\];\n\tJBP = cb\[9\];\n\tJB_CAP = cb\[12\];\n\tJB_DEPTH = 0;/\tJBP = cb[9];\n\tJB_CAP = cb[12];\n\tJB_DEPTH = 0;/'
    brk "GC_STACK_BASE not switched back on yield" \
        's/\tGC_STACK_BASE = cb\[5\];\n\tJBP = cb\[9\];\n\tJB_CAP = cb\[12\];\n\tJB_DEPTH = cb\[10\];/\tJBP = cb[9];\n\tJB_CAP = cb[12];\n\tJB_DEPTH = cb[10];/'
    brk "CUR_GEN not a root" \
        's/\tgc_try\(CUR_GEN, 1\);/\t;/'
    # --- the throw gate: the four things that make a throw across a yield safe -
    brk "a suspended coroutine's try barriers not scanned" \
        's/\t\t\tgc_scan_jb\(cb\[9\], cb\[10\]\);/\t\t\t;/'
    brk "the parked resumer's try barriers not scanned" \
        's/\t\tgc_scan_jb\(jp\[i\], jd\[i\]\);/\t\t;/'
    brk "ONE GLOBAL jmp_buf pool (the pre-fix floor)" \
        's/\tcb\[9\] = \(long\)malloc\(8 \* 128\);\n\tcb\[12\] = 128;\n\ti = 0;\n\twhile \(i < 128\) \{ long \*jp = \(long \*\)cb\[9\]; jp\[i\] = 0; i = i \+ 1; \}/\tcb[9] = JB_MAIN;\n\tcb[12] = 512;/'
    brk "the body's throw not re-raised on the resumer" \
        's/\t\tjs_throw\(ff\(g\)\);/\t\t;/'
fi
