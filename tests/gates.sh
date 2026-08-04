#!/usr/bin/env bash
#
# gates.sh - run EVERY gate this project has, in one command, and report one
# verdict.
#
# Why this exists: the gates are seven separate commands with three different
# reporting conventions, and two of them are only correct when you also grep
# their output. Getting that wrong is not hypothetical - a `-frozen`-only
# divergence never reached `--full`'s summary line at all until 52dff5e, and the
# default suite is STRUCTURALLY BLIND to every languages/lib/*-rt.metajs change
# because ./test.sh, --full and --cross all run llvm.Run, which resolves js_* to
# the Go twin and never touches layer 2. Only clang-check, native-full and a
# native -exe probe can see layer 2 at all.
#
# So the rule this script encodes: green means all seven, including the greps.
#
# Usage:
#   tests/gates.sh              every gate (~5-6 minutes)
#   tests/gates.sh --quick      skip the two slowest (clang-check, native-full).
#                               NOT sufficient for a layer-2 change - it says so.
#   tests/gates.sh --gen        also re-check all fifteen checked-in .ll files
#                               reproduce from source (MANDATORY after any
#                               runtime.c / *-rt.metajs edit; via tests/gen-all.sh,
#                               which shares one binary and runs them in parallel)
#   tests/gates.sh --freeze     also check `-freeze` is a fixed point (MANDATORY
#                               after any metajs-to-llvm-ir.abnf / compile-core.js
#                               edit)
#
# Exit 0 iff every gate that ran passed.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

QUICK=0; GEN=0; FREEZE=0
for a in "$@"; do
    case "$a" in
        --quick)  QUICK=1 ;;
        --gen)    GEN=1 ;;
        --freeze) FREEZE=1 ;;
        --all)    GEN=1; FREEZE=1 ;;
        -h|--help) sed -n '2,30p' "$0"; exit 0 ;;
        *) echo "gates.sh: unknown flag $a" >&2; exit 2 ;;
    esac
done

FAILED=()
pass() { printf '  %-24s %s\n' "$1" "$2"; }
fail() { printf '  %-24s %s\n' "$1" "$2"; FAILED+=("$1"); }

log=$(mktemp -d)
trap 'rm -rf "$log"' EXIT

echo "== building"
if ! go build -o mec . 2>"$log/build"; then
    cat "$log/build"; echo "gates.sh: BUILD FAILED"; exit 1
fi

echo "== gates"

# 1. The matrix. Its own summary line is trustworthy.
./test.sh >"$log/matrix" 2>&1
m=$(grep -E '^matrix: ' "$log/matrix" | tail -1)
if grep -q '^all green' "$log/matrix"; then pass matrix "${m:-ok}"; else fail matrix "${m:-NO SUMMARY} - see $log/matrix"; cp "$log/matrix" /tmp/gates-matrix.log; fi

# 2. The ratchet. Informational by design - it ALWAYS exits 0 - so the summary
#    line is not a verdict and the grep is the actual gate.
./test.sh --full >"$log/full" 2>&1
bad=$(grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF' "$log/full")
n=$(grep -E 'assertions in total' "$log/full" | tail -1)
if [ -z "$bad" ]; then pass --full "${n:-ran}"; else fail --full "${n:-ran} -- $(echo "$bad" | head -3)"; fi

# 3. The two halves of each language, diffed against each other.
./test.sh --cross >"$log/cross" 2>&1
c=$(grep -E 'programs compared' "$log/cross" | tail -1)
case "$c" in
    *", 0 divergent"*) pass --cross "$c" ;;
    *) fail --cross "${c:-no summary}" ;;
esac

# 4. Go-side unit tests.
if go test ./abnf/ >"$log/gotest" 2>&1; then pass "go test" "$(tail -1 "$log/gotest")"
else fail "go test" "$(tail -3 "$log/gotest" | tr '\n' ' ')"; fi

if [ "$QUICK" = 0 ]; then
    # 5 and 6 are THE ONLY GATES THAT SEE LAYER 2.
    tests/clang-check.sh >"$log/clang" 2>&1; crc=$?
    held=$(grep -c 'row HELD' "$log/clang")
    ok=$(grep -c 'the clang executable agrees' "$log/clang")
    if [ "$crc" = 0 ] && [ "$ok" -ge 16 ] && [ "$held" = 0 ]; then pass clang-check "$ok/16 agreeing, none held"
    else fail clang-check "$ok/16 agreeing, $held held, rc=$crc - copied to /tmp/gates-clang.log"; cp "$log/clang" /tmp/gates-clang.log; fi

    tests/native-full.sh >"$log/native" 2>&1
    nf=$(grep -E 'languages, every native binary|native binaries' "$log/native" | tail -1)
    if grep -q 'every native binary exits 0 with 0 failures' "$log/native"; then pass native-full "$nf"
    else fail native-full "${nf:-see $log/native}"; cp "$log/native" /tmp/gates-native.log; fi
else
    printf '  %-24s %s\n' "clang-check" "SKIPPED (--quick)"
    printf '  %-24s %s\n' "native-full" "SKIPPED (--quick)"
    echo "  NOTE: --quick cannot see languages/lib/*-rt.metajs. Do not call a layer-2 change green on it."
fi

if [ "$GEN" = 1 ]; then
    # tests/gen-all.sh builds mec once, exports MEC_BIN so the fifteen generators
    # skip their own go build, and fans them out. Same fifteen checks, ~6x faster.
    # Its exit status is the verdict; the summary line is the last line.
    tests/gen-all.sh --check >"$log/gen" 2>&1; grc=$?
    gs=$(grep -E '^gen-all: ' "$log/gen" | tail -1)
    if [ "$grc" = 0 ]; then pass "gen --check" "${gs:-ok}"
    else fail "gen --check" "${gs:-no summary} - copied to /tmp/gates-gen.log"; cp "$log/gen" /tmp/gates-gen.log; fi
fi

if [ "$FREEZE" = 1 ]; then
    # Both artefacts: -freeze writes the IR snapshot AND the frozen grammar.
    sum() { cat abnf/jsbootstrap.ll abnf/jsagrammar.go | (md5 -q 2>/dev/null || md5sum | cut -d' ' -f1); }
    before=$(sum)
    ./mec -freeze languages/metajs-to-llvm-ir.abnf >/dev/null 2>&1
    after=$(sum)
    if [ "$before" = "$after" ]; then pass "-freeze" "fixed point (identical)"
    else fail "-freeze" "SNAPSHOT WAS STALE - it changed; re-run to confirm the new one is a fixed point"; fi
fi

echo
if [ ${#FAILED[@]} -eq 0 ]; then
    echo "ALL GATES GREEN$([ "$QUICK" = 1 ] && echo " (layer 2 NOT covered: --quick)")"
    exit 0
fi
echo "FAILED: ${FAILED[*]}"
exit 1
