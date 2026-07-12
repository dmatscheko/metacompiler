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
#   ./test.sh                 run the full matrix (most entries are fast
#                             feature-matrix tests)
#   ./test.sh -f, --filter S  run only entries whose name or args contain S
#                             (case-insensitive), e.g. --filter kotlin
#   ./test.sh -l, --list      list the matrix entries and exit
#   ./test.sh -v, --verbose   print every entry as it runs (default: only failures
#                             and a progress dot per entry)
#   ./test.sh -j, --jobs N    run N entries in parallel (default: CPU count;
#                             1 = sequential)
#   ./test.sh -t, --timeout N per-run timeout in seconds (default 120)
#   ./test.sh --full          run the SECOND test group: the full-syntax ratchet
#                             files (tests/*-test-full.*). For every grammar it
#                             reports the language areas that do not work yet -
#                             the goal is full language support, and this is the
#                             progress report. Informational: always exits 0.
#                             Combine with --filter to probe one language.
#   ./test.sh -h, --help      show this header
#
# Requires: go (to build the compiler). awk and, if present, timeout/gtimeout/perl
# (for the per-run timeout) are used from the base system.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT" || exit 2

FILTER=""; VERBOSE=0; LIST=0; TIMEOUT=120; FULL=0
JOBS="$(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)"
while [ $# -gt 0 ]; do
    case "$1" in
        -f|--filter)  FILTER="${2:-}"; shift ;;
        --filter=*)   FILTER="${1#*=}" ;;
        -v|--verbose) VERBOSE=1 ;;
        -l|--list)    LIST=1 ;;
        -j|--jobs)    JOBS="${2:-1}"; shift ;;
        --jobs=*)     JOBS="${1#*=}" ;;
        -t|--timeout) TIMEOUT="${2:-120}"; shift ;;
        --timeout=*)  TIMEOUT="${1#*=}" ;;
        --full)       FULL=1 ;;
        -h|--help)    sed -n '2,39p' "$0"; exit 0 ;;
        *) echo "test.sh: unknown option '$1' (try --help)" >&2; exit 2 ;;
    esac
    shift
done
case "$JOBS" in ''|*[!0-9]*|0) JOBS=1 ;; esac

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
RESDIR="$(mktemp -d -t mec-results.XXXXXX)"
trap 'rm -rf "$BIN" "$ENTRIES" "$RESDIR"' EXIT

if [ "$LIST" -eq 0 ]; then
    echo "building compiler..."
    go build -o "$BIN" . || { echo "test.sh: build failed" >&2; exit 2; }
fi

# ----------------------------------------------------------------------------
# The second test group: the full-syntax ratchet (./test.sh --full).
#
# Every tests/<lang>-test-full.<ext> walks the WHOLE syntax of its language in
# self-contained SECTION chunks (see tests/js-test-full.js for the anatomy).
# For each grammar of the language the probe runs the file and, whenever the
# grammar chokes, deletes the section around the reported line (plus its
# SECTION-CALL line in main) and retries - so one probe lists EVERY unsupported
# language area, not just the first. When an error carries no usable position,
# each remaining section is run in isolation (everything else deleted) to
# classify it. A file that runs green untouched is reported FULL and cross-
# checked goja vs -frozen; that language is ready to join the default matrix.

# full_error_msg extracts the most telling line of a failed run's output.
full_error_msg() {
    printf '%s\n' "$1" | grep -i 'error\|not implemented' | grep -v '^  ==> Fail' | head -1 | cut -c1-160 | grep . \
        || printf '%s\n' "$1" | grep -v '^  ==> Fail' | grep . | head -1 | cut -c1-160
}

# full_isolate classifies each remaining section of $2 by running it alone
# (prologue + that section + main with only its call). Prints one line per
# still-failing section.
full_isolate() {
    local G="$1" work="$2" iso="$2.iso"
    local id ids name rc out
    ids="$(awk '/===== SECTION [0-9]+:/ { line = $0; sub(/.*SECTION /, "", line); sub(/:.*/, "", line); print line }' "$work")"
    for id in $ids; do
        awk -v keep="$id" '
            /===== SECTION [0-9]+:/ { line = $0; sub(/.*SECTION /, "", line); cur = line; sub(/:.*/, "", cur); insec = 1 }
            /===== END SECTIONS/ { insec = 0 }
            {
                if (insec && cur != keep) next
                if ($0 ~ /SECTION-CALL/ && $0 !~ ("SECTION-CALL " keep)) next
                print
            }
        ' "$work" > "$iso"
        out="$(RUN "$BIN" "$G" "$iso" -q 2>&1)"; rc=$?
        if [ "$rc" -ne 0 ]; then
            name="$(awk -v want="$id" '/===== SECTION [0-9]+:/ {
                line = $0; sub(/.*SECTION /, "", line); id = line; sub(/:.*/, "", id)
                if (id == want) { nm = line; sub(/^[0-9]*: */, "", nm); sub(/ =====.*/, "", nm); print nm; exit }
            }' "$work")"
            printf '    - %s %s: %s\n' "$id" "$name" "$(full_error_msg "$out")"
        fi
    done
    rm -f "$iso"
}

# full_probe writes the gap report for one grammar/test-file pair to $3.
full_probe() {
    local G="$1" F="$2" R="$3"
    local work="$R.work"
    cp "$F" "$work"
    local gaps=0 iter=0 rc out ln reason impl info id name start end
    {
        printf '%s:\n' "$(basename "$G" .abnf)"
        while :; do
            iter=$((iter + 1))
            if [ "$iter" -gt 60 ]; then printf '    (stopped after 60 rounds)\n'; break; fi
            out="$(RUN "$BIN" "$G" "$work" -q 2>&1)"; rc=$?
            if [ "$rc" -eq 0 ]; then
                if [ "$gaps" -eq 0 ]; then
                    RUN "$BIN" "$G" "$F" -q          > "$work.g" 2>/dev/null
                    RUN "$BIN" "$G" "$F" -q -frozen  > "$work.f" 2>/dev/null
                    if cmp -s "$work.g" "$work.f"; then
                        printf '    FULL - every section parses and passes, goja and -frozen byte-identical\n'
                    else
                        printf '    FULL under goja, BUT -frozen fails or differs - inspect before celebrating\n'
                    fi
                else
                    printf '    (the remaining sections pass)\n'
                fi
                break
            fi
            # Where did it choke? Parse failures, "not implemented" aborts and
            # generic "line N" messages carry a position; everything else goes
            # through the per-section isolation fallback.
            ln="$(printf '%s\n' "$out" | sed -n 's/.*Last good parse position was ln \([0-9][0-9]*\),.*/\1/p' | head -1)"
            reason="does not parse"
            if [ -z "$ln" ]; then
                impl="$(printf '%s\n' "$out" | grep 'not implemented' | head -1)"
                if [ -n "$impl" ]; then
                    ln="$(printf '%s\n' "$impl" | sed -n 's/.*:\([0-9][0-9]*\)).*/\1/p')"
                    reason="$(printf '%s\n' "$impl" | sed -n 's/.*error: \(.*\) not implemented.*/not implemented: \1/p')"
                    [ -z "$reason" ] && reason="not implemented"
                fi
            fi
            if [ -z "$ln" ]; then
                ln="$(printf '%s\n' "$out" | sed -n 's/.* line \([0-9][0-9]*\).*/\1/p' | head -1)"
                [ -n "$ln" ] && reason="error"
            fi
            if [ -z "$ln" ]; then
                printf '    error without a usable position: %s\n' "$(full_error_msg "$out")"
                full_isolate "$G" "$work"
                break
            fi
            info="$(awk -v L="$ln" '
                /===== SECTION [0-9]+:/ {
                    if (insec && L >= start && L < NR) { printf "%s\t%s\t%d\t%d\n", id, name, start, NR - 1; exit }
                    line = $0
                    sub(/.*SECTION /, "", line); id = line; sub(/:.*/, "", id)
                    name = line; sub(/^[0-9]*: */, "", name); sub(/ =====.*/, "", name)
                    start = NR; insec = 1
                    next
                }
                /===== END SECTIONS/ {
                    if (insec && L >= start && L < NR) { printf "%s\t%s\t%d\t%d\n", id, name, start, NR - 1 }
                    exit
                }
            ' "$work")"
            if [ -z "$info" ]; then
                printf '    error at line %s outside the sections (prologue/main must stay minimal): %s\n' "$ln" "$(full_error_msg "$out")"
                break
            fi
            id="$(printf '%s' "$info" | cut -f1)"
            name="$(printf '%s' "$info" | cut -f2)"
            start="$(printf '%s' "$info" | cut -f3)"
            end="$(printf '%s' "$info" | cut -f4)"
            printf '    - %s %s: %s\n' "$id" "$name" "$reason"
            gaps=$((gaps + 1))
            sed "${start},${end}d" "$work" > "$work.n" && mv "$work.n" "$work"
            grep -v "SECTION-CALL $id" "$work" > "$work.n" && mv "$work.n" "$work"
        done
    } > "$R"
    rm -f "$work" "$work.n" "$work.iso" "$work.g" "$work.f"
}

if [ "$FULL" -eq 1 ]; then
    FILES="$(ls tests/*-test-full.* 2>/dev/null)"
    if [ -z "$FILES" ]; then echo "test.sh --full: no tests/*-test-full.* files found" >&2; exit 2; fi
    if [ "$LIST" -eq 1 ]; then printf '%s\n' $FILES; exit 0; fi

    SEM="$RESDIR/fsem"
    mkfifo "$SEM"; exec 4<>"$SEM"; rm -f "$SEM"
    i=0; while [ "$i" -lt "$JOBS" ]; do printf '.' >&4; i=$((i + 1)); done

    n=0; REPORTS=""
    for f in $FILES; do
        base="$(basename "$f")"; lang="${base%%-test-full.*}"
        for G in "languages/$lang-interpreter.abnf" "languages/$lang-to-llvm-ir.abnf"; do
            hay="$lang $G $f"
            if [ -n "$FILTER" ] && ! printf '%s' "$hay" | grep -qiF -- "$FILTER"; then continue; fi
            n=$((n + 1))
            r="$RESDIR/full.$(printf '%03d' "$n")"
            REPORTS="$REPORTS $r"
            if [ ! -f "$G" ]; then
                printf '%s:\n    (grammar not found, skipped)\n' "$(basename "$G" .abnf)" > "$r"
                continue
            fi
            IFS= read -r -n 1 -u 4 _t
            { full_probe "$G" "$f" "$r"; printf '.' >&4; } &
        done
    done
    wait
    echo "full-syntax ratchet - unsupported language areas per grammar"
    echo "(the second test group: run on demand to see what full language support"
    echo " still needs; the default matrix is unaffected by these files)"
    if [ -z "$REPORTS" ]; then echo; echo "nothing matched the filter"; exit 0; fi
    for r in $REPORTS; do echo; cat "$r"; done
    exit 0
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

n_entries=$(wc -l < "$ENTRIES" | tr -d ' ')

# run_entry runs one matrix entry (both engines), compares, and leaves its
# verdict in $RESDIR: "<idx>.ok" (empty) or "<idx>.fail" (the printable failure
# block). Entries are independent, so any number of them may run in parallel;
# each uses its own output files under $RESDIR.
run_entry() {
    local idx="$1" name="$2"; shift 2
    local args=("$@")
    local og="$RESDIR/$idx.og" of="$RESDIR/$idx.of" eg="$RESDIR/$idx.eg" ef="$RESDIR/$idx.ef"

    local has_q=0 a
    for a in "${args[@]}"; do [ "$a" = "-q" ] && has_q=1; done
    # Expected-to-fail: author-declared (name mentions FAIL) or the two grammar
    # guards that fail by design (their names do not carry FAIL).
    local should_fail=0
    case "$name" in *[Ff][Aa][Ii][Ll]*) should_fail=1 ;; esac
    case " ${args[*]} " in *smaller-match-first*|*infinite-loop*) should_fail=1 ;; esac

    local rc_g rc_f
    RUN "$BIN" "${args[@]}"          >"$og" 2>"$eg"; rc_g=$?
    RUN "$BIN" "${args[@]}" -frozen  >"$of" 2>"$ef"; rc_f=$?

    local problems=()
    if [ "$should_fail" -eq 1 ]; then
        [ "$rc_g" -eq 0 ] && problems+=("goja exited 0 but should fail")
        [ "$rc_f" -eq 0 ] && problems+=("frozen exited 0 but should fail")
    else
        [ "$rc_g" -ne 0 ] && problems+=("goja exit $rc_g")
        [ "$rc_f" -ne 0 ] && problems+=("frozen exit $rc_f")
    fi
    if [ "$has_q" -eq 1 ] && ! cmp -s "$og" "$of"; then
        problems+=("goja vs frozen -q output differ")
    fi

    if [ "${#problems[@]}" -eq 0 ]; then
        : > "$RESDIR/$idx.ok"
        [ "$VERBOSE" -eq 1 ] && printf '  ok   [%d/%d] %s\n' "$idx" "$n_entries" "$name"
    else
        # Compose the whole failure block first and print it with one write, so
        # parallel workers cannot interleave mid-block.
        local block p
        block="$(printf 'FAIL   %s\n' "$name"
                 for p in "${problems[@]}"; do printf '         - %s\n' "$p"; done
                 head -3 "$eg" | sed 's/^/         goja: /')"
        printf '%s\n' "$block" > "$RESDIR/$idx.fail"
        printf '%s\n' "$block"
    fi
    rm -f "$og" "$of" "$eg" "$ef"
    [ "$VERBOSE" -eq 0 ] && printf '.' >&2
    return 0
}

# A FIFO token semaphore caps the pool at $JOBS parallel entries (works on the
# stock macOS bash 3.2: no wait -n). Each worker takes a token before it starts
# and puts one back when done.
SEM="$RESDIR/sem"
mkfifo "$SEM"
exec 3<>"$SEM"
rm -f "$SEM"
i=0; while [ "$i" -lt "$JOBS" ]; do printf '.' >&3; i=$((i + 1)); done

total=0; idx=0
while IFS=$'\t' read -r name rest; do
    [ -z "${name:-}" ] && continue
    IFS=$'\t' read -r -a args <<< "$rest"
    idx=$((idx + 1))

    hay="$name ${args[*]}"
    if [ -n "$FILTER" ] && ! printf '%s' "$hay" | grep -qiF -- "$FILTER"; then
        continue
    fi
    total=$((total + 1))

    IFS= read -r -n 1 -u 3 _token
    { run_entry "$idx" "$name" "${args[@]}"; printf '.' >&3; } &
done < "$ENTRIES"
wait

pass=$(ls "$RESDIR" | grep -c '\.ok$')
fail=$(ls "$RESDIR" | grep -c '\.fail$')

echo
echo "----------------------------------------------------------------"
printf 'matrix: %d entries run - %d passed, %d failed\n' "$total" "$pass" "$fail"

if [ "$fail" -ne 0 ]; then
    echo "FAILURES:"
    for f in $(ls "$RESDIR" | grep '\.fail$' | sort -n); do
        head -1 "$RESDIR/$f" | sed 's/^FAIL   /  - /'
    done
    exit 1
fi
echo "all green"
exit 0
