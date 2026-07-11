#!/usr/bin/env bash
#
# test.sh - run the whole metacompiler test matrix.
#
# The test suite IS .vscode/launch.json: every configuration with an "args" list is
# one grammar/input run. This script runs each of them twice - once with the default
# goja tag engine and once goja-free (with -frozen appended) - and checks the two
# invariants the project guarantees:
#
#   1. byte-identical output: for every run whose args contain -q (quiet), the goja
#      and -frozen stdout must be identical (the frozen bootstrap must match goja).
#   2. correct exit status: the language tests are self-checking (exit 0 == the
#      program's own checks passed). Ordinary entries must exit 0 on both engines;
#      the by-design failures (names containing "FAIL", plus the smaller-match-first
#      and infinite-loop grammar guards) must exit non-zero on both engines.
#
# It exits 0 iff the whole matrix is green.
#
# Usage:
#   ./test.sh                 run the full matrix (several minutes: the brainfuck
#                             big tests are 50k-172k opcodes)
#   ./test.sh -k, --quick     skip the slow brainfuck-test-big-* entries
#   ./test.sh -f, --filter S  run only entries whose name or args contain S
#                             (case-insensitive), e.g. --filter kotlin
#   ./test.sh -l, --list      list the matrix entries and exit
#   ./test.sh -v, --verbose   print every entry as it runs (default: only failures
#                             and a progress counter)
#   ./test.sh -t, --timeout N per-run timeout in seconds (default 120)
#   ./test.sh -h, --help      show this header
#
# Requires: go (to build the compiler). awk and, if present, timeout/gtimeout/perl
# (for the per-run timeout) are used from the base system.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT" || exit 2

QUICK=0; FILTER=""; VERBOSE=0; LIST=0; TIMEOUT=120
while [ $# -gt 0 ]; do
    case "$1" in
        -k|--quick)   QUICK=1 ;;
        -f|--filter)  FILTER="${2:-}"; shift ;;
        --filter=*)   FILTER="${1#*=}" ;;
        -v|--verbose) VERBOSE=1 ;;
        -l|--list)    LIST=1 ;;
        -t|--timeout) TIMEOUT="${2:-120}"; shift ;;
        --timeout=*)  TIMEOUT="${1#*=}" ;;
        -h|--help)    sed -n '2,40p' "$0"; exit 0 ;;
        *) echo "test.sh: unknown option '$1' (try --help)" >&2; exit 2 ;;
    esac
    shift
done

command -v go >/dev/null 2>&1 || { echo "test.sh: 'go' not found on PATH" >&2; exit 2; }

# Pick a per-run timeout wrapper from whatever the system offers, so a regression
# that hangs cannot wedge the whole run. Falls back to running without a timeout.
if command -v timeout >/dev/null 2>&1;  then RUN() { timeout "$TIMEOUT" "$@"; }
elif command -v gtimeout >/dev/null 2>&1; then RUN() { gtimeout "$TIMEOUT" "$@"; }
elif command -v perl >/dev/null 2>&1;   then RUN() { perl -e 'my $t=shift; alarm $t; exec @ARGV or exit 127' "$TIMEOUT" "$@"; }
else RUN() { "$@"; }
fi

BIN="$(mktemp -t mec-test.XXXXXX)"
ENTRIES="$(mktemp -t mec-entries.XXXXXX)"
OG="$(mktemp -t mec-og.XXXXXX)"; OF="$(mktemp -t mec-of.XXXXXX)"
EG="$(mktemp -t mec-eg.XXXXXX)"; EF="$(mktemp -t mec-ef.XXXXXX)"
trap 'rm -f "$BIN" "$ENTRIES" "$OG" "$OF" "$EG" "$EF"' EXIT

if [ "$LIST" -eq 0 ]; then
    echo "building compiler..."
    go build -o "$BIN" . || { echo "test.sh: build failed" >&2; exit 2; }
fi

# Extract the matrix from the JSONC launch.json. Every entry is a "name": line
# followed by a single "program": ... "args": [ ... ] line; emit one tab-separated
# record per entry: name <TAB> arg1 <TAB> arg2 ... The "-----" separators, the
# -freeze entry (mutates the snapshot) and entries without args are dropped.
awk '
    /"name":/ {
        n = $0; sub(/.*"name":[[:space:]]*"/, "", n); sub(/".*/, "", n); name = n
    }
    /"args":/ {
        a = $0; sub(/.*"args":[[:space:]]*\[/, "", a); sub(/\].*/, "", a)
        gsub(/"[[:space:]]*,[[:space:]]*"/, "\t", a)   # arg boundaries -> tab
        sub(/^[[:space:]]*"/, "", a); sub(/"[[:space:]]*$/, "", a)
        if (a ~ /-----/ || a ~ /-freeze/) next
        print name "\t" a
    }
' .vscode/launch.json > "$ENTRIES"

if [ "$LIST" -eq 1 ]; then
    nl -ba "$ENTRIES" | sed 's/\t/  |  /g'
    exit 0
fi

pass=0; fail=0; skipped=0; total=0; idx=0
n_entries=$(wc -l < "$ENTRIES" | tr -d ' ')
declare -a FAILURES=()

while IFS=$'\t' read -r name rest; do
    [ -z "${name:-}" ] && continue
    IFS=$'\t' read -r -a args <<< "$rest"
    idx=$((idx + 1))

    hay="$name ${args[*]}"
    if [ -n "$FILTER" ] && ! printf '%s' "$hay" | grep -qiF -- "$FILTER"; then
        continue
    fi
    if [ "$QUICK" -eq 1 ] && printf '%s' "${args[*]}" | grep -q 'brainfuck-test-big'; then
        skipped=$((skipped + 1)); continue
    fi
    total=$((total + 1))

    has_q=0; for a in "${args[@]}"; do [ "$a" = "-q" ] && has_q=1; done
    # Expected-to-fail: author-declared (name mentions FAIL) or the two grammar
    # guards that fail by design (their names do not carry FAIL).
    should_fail=0
    case "$name" in *[Ff][Aa][Ii][Ll]*) should_fail=1 ;; esac
    case " ${args[*]} " in *smaller-match-first*|*infinite-loop*) should_fail=1 ;; esac

    RUN "$BIN" "${args[@]}"          >"$OG" 2>"$EG"; rc_g=$?
    RUN "$BIN" "${args[@]}" -frozen  >"$OF" 2>"$EF"; rc_f=$?

    problems=()
    if [ "$should_fail" -eq 1 ]; then
        [ "$rc_g" -eq 0 ] && problems+=("goja exited 0 but should fail")
        [ "$rc_f" -eq 0 ] && problems+=("frozen exited 0 but should fail")
    else
        [ "$rc_g" -ne 0 ] && problems+=("goja exit $rc_g")
        [ "$rc_f" -ne 0 ] && problems+=("frozen exit $rc_f")
    fi
    if [ "$has_q" -eq 1 ] && ! cmp -s "$OG" "$OF"; then
        problems+=("goja vs frozen -q output differ")
    fi

    if [ "${#problems[@]}" -eq 0 ]; then
        pass=$((pass + 1))
        [ "$VERBOSE" -eq 1 ] && printf '  ok   [%d/%d] %s\n' "$idx" "$n_entries" "$name"
    else
        fail=$((fail + 1))
        printf 'FAIL   %s\n' "$name"
        for p in "${problems[@]}"; do printf '         - %s\n' "$p"; done
        # Show a little context from stderr to aid debugging.
        err="$(head -3 "$EG" | sed 's/^/         goja: /')"
        [ -n "$err" ] && printf '%s\n' "$err"
        FAILURES+=("$name")
    fi

    if [ "$VERBOSE" -eq 0 ] && [ $((total % 20)) -eq 0 ]; then
        printf '  ... %d run, %d passed, %d failed\n' "$total" "$pass" "$fail"
    fi
done < "$ENTRIES"

echo
echo "----------------------------------------------------------------"
printf 'matrix: %d entries run' "$total"
[ "$skipped" -gt 0 ] && printf ' (%d slow entries skipped)' "$skipped"
printf ' - %d passed, %d failed\n' "$pass" "$fail"

if [ "$fail" -ne 0 ]; then
    echo "FAILURES:"
    for f in "${FAILURES[@]}"; do echo "  - $f"; done
    exit 1
fi
echo "all green"
exit 0
