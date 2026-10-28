#!/usr/bin/env bash
#
# guard-selftest.sh - VERIFY tests/lib/guard.sh BY SABOTAGE.
#
# Each case starts a script that sources guard.sh and leaves something running
# that would fill the disk, then checks that it did not survive.
#
# Usage: tests/lib/guard-selftest.sh
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 2

WORK="$(mktemp -d -t mec-guardself.XXXXXX)"
trap 'pkill -9 -f "$WORK/" 2>/dev/null; rm -rf "$WORK"' EXIT

pass=0; fail=0
ok()  { printf 'ok   %s\n' "$1"; pass=$((pass + 1)); }
bad() { printf 'FAIL %s: %s\n' "$1" "$2"; fail=$((fail + 1)); }

# A writer that never stops. Its path is its identity for pgrep.
cat >"$WORK/flood" <<'EOF'
#!/usr/bin/env bash
while true; do echo xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx; done
EOF
chmod +x "$WORK/flood"

# The host script: sources the guard, removes its temp dir on exit like every
# script here does, and runs the flood two levels down (a subshell starting it),
# so the reap has to walk the tree.
cat >"$WORK/host.sh" <<EOF
#!/usr/bin/env bash
cd "$PWD" || exit 2
. tests/lib/guard.sh
T="\$(mktemp -d)"
trap 'guard_reap; rm -rf "\$T"' EXIT
( "$WORK/flood" >"\$T/out" ) &
wait
EOF
chmod +x "$WORK/host.sh"

# Without job control a background job starts with SIGINT ignored, and a signal
# ignored on entry cannot be trapped, so the INT case would test nothing.
set -m

running() { pgrep -f "bash $WORK/flood" >/dev/null; }

for sig in TERM INT HUP; do
    name="host killed by SIG$sig takes the writer with it"
    "$WORK/host.sh" &
    host=$!
    for _ in $(seq 1 200); do running && break; sleep 0.1; done
    running || { bad "$name" "the writer never started"; kill -9 "$host"; continue; }
    kill -"$sig" "$host"; wait "$host" 2>/dev/null
    sleep 0.3
    if running; then bad "$name" "the writer is still running"; pkill -9 -f "$WORK/flood"
    else ok "$name"; fi
done

# SIGKILL cannot be trapped, so the writer is orphaned: the inherited output cap
# is what must stop it.
name="host SIGKILLed: the orphan stops at the output cap"
MEC_MAX_OUTPUT_MB=1 "$WORK/host.sh" &
host=$!
for _ in $(seq 1 200); do running && break; sleep 0.1; done
kill -9 "$host"; wait "$host" 2>/dev/null
for _ in $(seq 1 50); do running || break; sleep 0.1; done
if running; then bad "$name" "the orphan is still running"; pkill -9 -f "$WORK/flood"
else ok "$name"; fi

name="guard_timeout kills an overrunning command"
( . tests/lib/guard.sh; guard_timeout 1 sleep 30 ); rc=$?
if [ "$rc" = 124 ] || [ "$rc" = 142 ]; then ok "$name"; else bad "$name" "exit $rc"; fi

name="a finished subshell does not reap its siblings"
out="$(bash -c '. tests/lib/guard.sh; sleep 2 & s=$!; ( guard_reap ); kill -0 $s && echo alive; kill $s')"
if [ "$out" = alive ]; then ok "$name"; else bad "$name" "sibling was killed"; fi

echo
printf '%d ok, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
