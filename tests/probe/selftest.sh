#!/usr/bin/env bash
#
# selftest.sh - VERIFY tests/probe.sh BY SABOTAGE.
#
# probe.sh is a gate, and this project's standard for a gate is that it must be
# shown to FAIL on a broken input, not merely to pass on a good one - above all,
# it must never report "all N legs agree" over EMPTY legs.
#
# So the cases below are the negative half, and tests/probe/good.sh is the
# positive one. A gate that only goes red proves as little as one that only goes
# green.
#
# Usage: tests/probe/selftest.sh          (needs bash 5.x on PATH, as an oracle)
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 2

PROBE=tests/probe.sh
WORK="$(mktemp -d -t mec-probeself.XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

pass=0; fail=0

# check NAME EXPECTED_RC EXPECTED_SUBSTRING -- command...
check() {
    local name="$1" want_rc="$2" want_txt="$3"; shift 4
    local out rc
    out="$("$@" 2>&1)"; rc=$?
    if [ "$rc" != "$want_rc" ]; then
        printf 'FAIL %-38s exit %s, wanted %s\n' "$name" "$rc" "$want_rc"
        printf '%s\n' "$out" | sed 's/^/       /' | head -12
        fail=$((fail + 1)); return
    fi
    if ! printf '%s\n' "$out" | grep -qF "$want_txt"; then
        printf 'FAIL %-38s exit %s ok, but no %s\n' "$name" "$rc" "\"$want_txt\""
        printf '%s\n' "$out" | sed 's/^/       /' | head -12
        fail=$((fail + 1)); return
    fi
    printf 'ok   %-38s exit %s\n' "$name" "$rc"
    pass=$((pass + 1))
}

[ -x ./mec ] || go build -o mec . || exit 2

echo "== the positive case: it must still go GREEN on a real agreement"
# Four legs, and the fourth is real bash 5.3. If this row goes red the rest of
# the file is measuring nothing.
check "bash 4-way vs real bash" 0 "all 4 legs agree" -- \
    "$PROBE" bash tests/probe/good.sh --oracle 'bash %s'

echo
echo "== the defect this file exists for: an EMPTY leg is not an agreeing leg"
check "all legs silent (js-test-1)" 2 "produced NO OUTPUT" -- \
    "$PROBE" js tests/js-test-1.js
check "calculator (no stdout at all)" 2 "THERE WAS NO COMPARISON" -- \
    "$PROBE" calculator tests/calculator-test-1.txt

# The sharpest one: sabotage the IR-flavour classifier so bash is treated as a
# handle-IR language (its run leg then comes out empty), and check that the
# three surviving legs agreeing PERFECTLY is still not a pass.
#
# The copy has to live in tests/ rather than in $WORK, because probe.sh finds
# the repo root as `dirname $0`/..  - a copy anywhere else would cd somewhere
# else and fail for the wrong reason.
SAB=tests/.probe-selftest-sabotage.sh
trap 'rm -rf "$WORK" "$SAB"' EXIT
sed 's/^handle_ir() {.*$/handle_ir() { return 0; }/' "$PROBE" >"$SAB"
chmod +x "$SAB"
grep -q '^handle_ir() { return 0; }$' "$SAB" || {
    echo "FAIL sabotage did not apply - handle_ir() was renamed?"; fail=$((fail + 1)); }
check "sabotaged flavour test, 3 legs agree" 1 "NOT a four-way comparison" -- \
    "$SAB" bash tests/probe/good.sh --oracle 'bash %s'

echo
echo "== the other ways a leg silently degrades"
check "oracle not installed here" 1 "NOT INSTALLED" -- \
    "$PROBE" bash tests/probe/good.sh --oracle 'no-such-toolchain-here %s'
check "grammar does not exist" 2 "no languages/calculator-interpreter.abnf" -- \
    "$PROBE" calculator tests/calculator-test-1.txt
check "misspelled --legs drops a leg" 2 "unknown leg" -- \
    "$PROBE" bash tests/probe/good.sh --legs interp,run,nativ

# A leg that prints some of its rows and is then killed: "a truncated leg is not
# a measurement".
cat >"$WORK/hang.sh" <<'EOF'
V=(1 2 3 4 5)
for i in "${!V[@]}"; do echo "row ${V[$i]}"; done
while true; do :; done
EOF
check "prints, then hangs (timeout 5s)" 2 "a truncated leg is not a measurement" -- \
    "$PROBE" bash "$WORK/hang.sh" --timeout 5
# The same input also trips the IR interpreter's step brake, which truncates the
# run leg without any timeout being involved.
check "step brake truncates the run leg" 2 "hit the IR step brake" -- \
    "$PROBE" bash "$WORK/hang.sh" --timeout 5

echo
printf -- '----------------------------------------------------------\n'
printf '%d ok, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
