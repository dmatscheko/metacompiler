#!/usr/bin/env bash
#
# bench-alloc.sh - how much memory and time one operation of the self-hosted
# runtime costs, measured rather than guessed.
#
# WHY THIS EXISTS
#
# languages/lib/runtime.c USED TO allocate from a bump arena that was never
# freed, so a long-running program's resident set WAS its allocation total, and
# peak RSS divided by the iteration count was an exact figure for "bytes
# allocated per operation" - no profiler, no sampling.
#
# docs/runtime-next-plan.md part 1b made it a mark/sweep heap, so RSS is now
# BOUNDED and no longer measures allocation. Both numbers still matter and the
# script reports both, from the same binary:
#
#   MEC_GC=off   the collector never runs, which is exactly the old arena, so
#                RSS / iterations is still the allocation rate in bytes per op
#   default      the collector runs, and RSS must FLATTEN as the iteration
#                count grows instead of scaling with it. That is the gate.
#
# Usage: tests/bench-alloc.sh [ITERS]     (default 200000)
#
# It writes its temporary programs and binaries into a temp directory and
# removes them; nothing in the working tree changes.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 2

ITERS="${1:-200000}"
TMP="$(mktemp -d -t mec-bench.XXXXXX)"
BIN="$TMP/mec"
trap 'rm -rf "$TMP"' EXIT

go build -o "$BIN" . || { echo "bench-alloc.sh: build failed" >&2; exit 2; }

# maximum resident set size, in bytes, of a command. BSD `time -l` reports it in
# bytes on darwin and GNU `time -v` in kilobytes, so each is read on its own.
maxrss() {
    if /usr/bin/time -l true >/dev/null 2>&1; then
        /usr/bin/time -l "$@" 2>&1 >/dev/null | awk '/maximum resident set size/ { print $1 }'
    else
        /usr/bin/time -v "$@" 2>&1 >/dev/null | awk '/Maximum resident set size/ { print $NF * 1024 }'
    fi
}

secs() {
    local t0 t1
    t0=$(python3 -c 'import time; print(time.time())')
    "$@" >/dev/null 2>&1
    t1=$(python3 -c 'import time; print(time.time())')
    python3 -c "print('%.2f' % ($t1 - $t0))"
}

echo "=== lua: s = s + i % 7, $ITERS iterations ==============================="
cat > "$TMP/loop.lua" <<EOF
local s = 0
for i = 1, $ITERS do s = s + i % 7 end
print(s)
EOF

"$BIN" languages/lua-to-llvm-ir.abnf "$TMP/loop.lua" -q -exe "$TMP/loop.out" >/dev/null || exit 1
printf '%-34s %8ss %10s MB\n' "native (C floor + layer 2)" \
    "$(secs "$TMP/loop.out")" "$(( $(maxrss "$TMP/loop.out") / 1048576 ))"
printf '%-34s %8ss %10s MB\n' "llvm.Run (the Go runtime)" \
    "$(secs "$BIN" languages/lua-to-llvm-ir.abnf "$TMP/loop.lua" -q)" \
    "$(( $(maxrss "$BIN" languages/lua-to-llvm-ir.abnf "$TMP/loop.lua" -q) / 1048576 ))"
if command -v lua >/dev/null 2>&1; then
    printf '%-34s %8ss %10s MB\n' "real lua ($(lua -v 2>&1 | head -1))" \
        "$(secs lua "$TMP/loop.lua")" "$(( $(maxrss lua "$TMP/loop.lua") / 1048576 ))"
fi

echo
echo "=== MEC_GC=off: no collection, so RSS/iteration IS bytes allocated per op"
echo "=== default:    the collector runs, and RSS must FLATTEN, not scale"
for n in $((ITERS / 4)) $((ITERS / 2)) "$ITERS" $((ITERS * 2)); do
    cat > "$TMP/n.lua" <<EOF
local s = 0
for i = 1, $n do s = s + i % 7 end
print(s)
EOF
    "$BIN" languages/lua-to-llvm-ir.abnf "$TMP/n.lua" -q -exe "$TMP/n.out" >/dev/null || exit 1
    # `env` rather than a VAR=x prefix: maxrss is a shell function, and bash does
    # not scope an assignment prefix to a function call the way it does to a
    # binary - the setting would leak into the next measurement.
    r=$(maxrss env MEC_GC=off "$TMP/n.out")
    g=$(maxrss "$TMP/n.out")
    printf 'iters=%-10s no-gc rss=%-12s bytes/iter=%-8s gc rss=%-11s bytes/iter=%s\n' \
        "$n" "$r" "$((r / n))" "$g" "$((g / n))"
done

echo
echo "=== metajs: try/catch/finally, $ITERS iterations ======================="
cat > "$TMP/try.js" <<EOF
function main() {
	var s = 0
	var i = 0
	while (i < $ITERS) {
		try { if (i % 3 == 0) { throw i } s = s + 1 } catch (e) { s = s + 2 } finally { s = s + 1 }
		i = i + 1
	}
	println(s)
	return 0
}
EOF
"$BIN" languages/metajs-to-llvm-ir.abnf "$TMP/try.js" -q -exe "$TMP/try.out" >/dev/null || exit 1
printf '%-34s %8ss %10s MB\n' "native (C floor)" \
    "$(secs "$TMP/try.out")" "$(( $(maxrss "$TMP/try.out") / 1048576 ))"
