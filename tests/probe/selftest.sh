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

# The sharpest ones sabotage a copy of probe.sh itself. The copy has to live in
# tests/ rather than in $WORK, because probe.sh finds the repo root as
# `dirname $0`/.. - a copy anywhere else would cd somewhere else and fail for
# the wrong reason.
SAB=tests/.probe-selftest-sabotage.sh
trap 'rm -rf "$WORK" "$SAB"' EXIT

# sabotage NAME SED_EXPR WANT - write $SAB with the edit applied and check that
# it changed exactly WANT lines: an edit that no longer matches (probe.sh was
# reworded) would leave an intact copy that goes green for the wrong reason.
sabotage() {
    sed "$2" "$PROBE" >"$SAB"; chmod +x "$SAB"
    local n; n=$(diff "$PROBE" "$SAB" | grep -c '^>')
    [ "$n" = "$3" ] && return 0
    printf 'FAIL %-38s changed %s lines of %s, wanted %s - reworded?\n' "$1" "$n" "$PROBE" "$3"
    fail=$((fail + 1)); return 1
}

# An EMPTY run leg: both run_leg run commands become `true`. The three surviving
# legs agree PERFECTLY, and that is still not a pass.
sabotage "sabotage: empty run leg" 's/run_leg run \.\/mec /run_leg run true /' 2 &&
check "run leg emptied, 3 legs agree" 1 "run: produced NO OUTPUT" -- \
    "$SAB" bash tests/probe/good.sh --oracle 'bash %s'

# A CORRUPTED run leg: module_lines() cuts nothing, so bash's run leg keeps the
# emitted module in front of the program's output - what a wrong IR-flavour
# classification produces. It must read as a disagreement, not as a pass.
sabotage "sabotage: module left uncut" 's/print n + 0 }/print 0 }/' 1 &&
check "run leg keeps the module" 1 "interp   != run" -- \
    "$SAB" bash tests/probe/good.sh --oracle 'bash %s'

echo
echo "== the other ways a leg silently degrades"
check "oracle not installed here" 1 "NOT INSTALLED" -- \
    "$PROBE" bash tests/probe/good.sh --oracle 'no-such-toolchain-here %s'
check "grammar does not exist" 2 "no languages/calculator-interpreter.abnf" -- \
    "$PROBE" calculator tests/calculator-test-1.txt
check "misspelled --legs drops a leg" 2 "unknown leg" -- \
    "$PROBE" bash tests/probe/good.sh --legs interp,run,nativ
# --cf must reach all three of our legs: an undeclared flag breaks every one.
check "--cf reaches the native build" 2 "native   BROKEN: the -exe BUILD FAILED" -- \
    "$PROBE" bash tests/probe/good.sh --cf nope=1 --legs interp,run,native
check "--cf reaches the interpreter leg" 2 "interp   BROKEN: produced NO OUTPUT (exit 2)" -- \
    "$PROBE" bash tests/probe/good.sh --cf nope=1 --legs interp,run,native

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
# run leg without any timeout being involved. The run leg alone, under a timeout
# it never reaches (the brake trips in about a second), so a loaded machine
# cannot turn the brake into a kill.
check "step brake truncates the run leg" 2 "hit the IR step brake" -- \
    "$PROBE" bash "$WORK/hang.sh" --legs run --timeout 120

# A native leg that prints forever must be stopped by the output cap, not by
# the disk filling up.
cat >"$WORK/flood.sh" <<'EOF'
V=(x)
while true; do echo "row ${V[0]}"; done
EOF
check "prints forever (cap 1 MB)" 2 "was killed - raise --max-output" -- \
    "$PROBE" bash "$WORK/flood.sh" --legs native --max-output 1 --timeout 120

# A probe that is terminated mid-leg must take the leg with it. --keep gives the
# binary a known path to look for afterwards. The match is exact (-x) because
# the clang command that builds it names the same path.
"$PROBE" bash "$WORK/hang.sh" --legs native --timeout 120 --keep "$WORK/orphan" >/dev/null 2>&1 &
probe_pid=$!
started=""
for _ in $(seq 1 300); do pgrep -fx "$WORK/orphan/a.out" >/dev/null && { started=1; break; }; sleep 0.2; done
kill -TERM "$probe_pid"; wait "$probe_pid" 2>/dev/null
sleep 0.5
if [ -z "$started" ]; then
    printf 'FAIL %-38s the native leg never started\n' "terminated probe kills its leg"
    fail=$((fail + 1))
elif pgrep -fx "$WORK/orphan/a.out" >/dev/null; then
    printf 'FAIL %-38s leg still running after the probe died\n' "terminated probe kills its leg"
    pkill -9 -fx "$WORK/orphan/a.out"
    fail=$((fail + 1))
else
    printf 'ok   %-38s\n' "terminated probe kills its leg"
    pass=$((pass + 1))
fi

echo
printf -- '----------------------------------------------------------\n'
printf '%d ok, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
