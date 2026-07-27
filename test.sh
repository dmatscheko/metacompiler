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
#   ./test.sh --cross         run the THIRD test group: diff each language's two
#                             halves against each other. The default matrix runs
#                             every entry twice (goja and -frozen) and demands
#                             byte-identical output - but that compares each
#                             engine against ITSELF and is blind to the
#                             interpreter and the compiler of one language giving
#                             different answers. This group is the check for
#                             that. Informational: always exits 0.
#                             Combine with --filter to probe one language.
#   ./test.sh -h, --help      show this header
#
# Requires: go (to build the compiler). awk and, if present, timeout/gtimeout/perl
# (for the per-run timeout) are used from the base system.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT" || exit 2

FILTER=""; VERBOSE=0; LIST=0; TIMEOUT=120; FULL=0; CROSS=0
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
        --cross)      CROSS=1 ;;
        -h|--help)    sed -n '2,48p' "$0"; exit 0 ;;
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
        # Same vacuity trap as in full_probe: a section that "passes" without the
        # summary line never ran at all, so treat a missing summary as a failure.
        if [ "$rc" -eq 0 ] && ! printf '%s\n' "$out" | grep -Eq '^full: [0-9]+ checks, [0-9]+ failures$'; then
            rc=1
            out="VACUOUS: exits 0 but never printed the summary line - main() is not being run"
        fi
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
                # A zero exit is NOT enough. Every *-test-full.* file ends main()
                # with the summary line 'full: <checks> checks, <failures> failures'
                # (see the conventions in tests/js-test-full.js). If a grammar runs
                # the file top to bottom without ever entering main() - which is
                # exactly what the python grammar did until 2026-07-27 - it exits 0
                # having executed NOTHING, and this probe used to report that as
                # full support. Demand the summary line, so an empty ratchet is
                # loud instead of green.
                if ! printf '%s\n' "$out" | grep -Eq '^full: [0-9]+ checks, [0-9]+ failures$'; then
                    printf '    VACUOUS - exits 0 but never printed the summary line: main() is not being run\n'
                    break
                fi
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

# ----------------------------------------------------------------------------
# The third test group: cross-half agreement (./test.sh --cross).
#
# The default matrix runs every entry twice, goja and -frozen, and demands
# byte-identical stdout. That is a strong invariant and it is blind to one whole
# class of defect: the INTERPRETER grammar and the COMPILER grammar of the same
# language giving DIFFERENT answers for the same program. Both halves are
# self-consistent across both script hosts, both report FULL, and the matrix is
# green while nil.to_s is "" in one and "null" in the other. Nothing flags that.
#
# So run every test program through both halves of its language and diff. A
# difference is not automatically a bug - the real language decides which half
# is right, and this group cannot know that - so it reports rather than judges,
# and always exits 0. Its job is to make an invisible class visible.
#
# cross_strip removes a -to-llvm-ir grammar's emitted module, leaving just the
# program's own output. -q prints the module AND the output for a compiler (the
# module is its default output, not noise), and -qq is not a uniform substitute:
# a grammar running through the JS runtime still prints program output under it,
# one running through llvm.Run does not. So strip by structure: drop everything
# up to and including the last line that is "}" or begins with @ / declare /
# define. On interpreter output nothing matches and the text passes through.
# It also drops the harness's own trailers, which name the half that ran and so
# differ by construction ("ruby interpreter: program finished" vs "ruby compiler:
# jsmain() returned 0"), and blank padding. An abort message is one of those: the
# interpreter half's "<lang> interpreter error: ..." was already dropped, so the
# compiled half's "IR interpreter: ..." is dropped with it - otherwise the two
# halves aborting on the SAME cause read as a divergence purely because only one
# of the two spellings survived. An abort is still visible either way, as the
# output that stops. What survives is the program's own
# output plus any warning text - and a warning present in one half and not the
# other IS a real finding, because it means the halves implement different
# subsets, so those are deliberately kept.
#
# Warnings need one extra step to survive that. A COMPILER emits them while it
# is building the module, so they land BEFORE the IR and the structural strip
# above would delete them wholesale - which made all thirteen languages look as
# though only their interpreter half warned, when in fact both halves warn
# identically and the harness was eating one copy. So lift every "warning:" line
# out first and re-emit it ahead of the stripped body. Relative order between a
# warning and the program output is not comparable across the halves anyway (one
# warns during compilation, the other during the run), but PRESENCE is exactly
# what this group wants to see.
cross_strip() {
    # A NUL byte reaches here from grammars that embed one in a string constant;
    # command substitution drops it anyway and warns about it on stderr, so drop
    # it explicitly and identically for both halves.
    local all; all="$(LC_ALL=C tr -d '\000')"
    printf '%s\n' "$all" | grep -E '^warning: ' || true
    printf '%s\n' "$all" | grep -vE '^warning: ' \
    | awk 'BEGIN { n = 0 } { l[n++] = $0 }
         END { last = -1
               for (i = 0; i < n; i++)
                   if (l[i] == "}" || l[i] ~ /^declare / || l[i] ~ /^@/ || l[i] ~ /^define /) last = i
               for (i = last + 1; i < n; i++) print l[i] }' \
    | grep -Ev '^[a-z#+-]+ (interpreter|compiler): |^  ==> Fail$|^[a-z#+-]+ (interpreter|compiler) error: ' \
    | grep -Ev '^(js runtime error: |jsmain\(\) returned )' \
    | grep -Ev '^[a-z0-9-]+: (exit status|main\(\) returned|program value is) [0-9-]+$' \
    | grep -Ev '^--- Executed with the built-in IR interpreter, output:$' \
    | grep -Ev '^IR interpreter: ' \
    | awk 'NF { blank = 0; for (i = 1; i <= held; i++) print ""; held = 0; print; seen = 1; next }
           seen { held++ }'
}

# cross_one diffs the two halves of one language on one program, writing its
# verdict to $4.
cross_one() {
    local L="$1" P="$2" R="$3"
    local gi="languages/$L-interpreter.abnf" gc="languages/$L-to-llvm-ir.abnf"
    local oi oc ri rc

    # -i tests/imports is what the matrix gives every *-test-multifile entry, and
    # without it those programs run in their DEGENERATE mode: the import does not
    # resolve, -warn-imports skips it, and each half then fails its own way on the
    # missing symbol. That compares the recovery paths instead of the feature, so
    # pass the same include root here. It is inert for every other program.
    oi="$(RUN "$BIN" -i tests/imports "$gi" "$P" -q -warn-unsupported -warn-imports 2>&1 | cross_strip)"; ri=$?
    oc="$(RUN "$BIN" -i tests/imports "$gc" "$P" -q -warn-unsupported -warn-imports 2>&1 | cross_strip)"; rc=$?

    if [ "$oi" = "$oc" ]; then
        printf 'agree\n' > "$R"; return
    fi
    # Both halves refusing the program is agreement, not divergence: a construct
    # neither implements is a gap the ratchet already measures, not a wrong
    # answer. Only report when at least one half produced real output.
    if [ -z "$oi" ] && [ -z "$oc" ]; then printf 'agree\n' > "$R"; return; fi
    printf '%s\n' "$oi" > "$R.i"; printf '%s\n' "$oc" > "$R.c"
    local d kind
    d="$(diff "$R.i" "$R.c" 2>/dev/null)"
    # Separate a warning-only difference from a real one. Under
    # -warn-unsupported the two halves can warn at different points for the same
    # construct - which is worth seeing, but it is NOT the two of them computing
    # different answers, and reporting both the same way makes the important
    # kind easy to misread. (It misread once during development: a warning-only
    # row looked like the compiler implementing ||= that the interpreter
    # lacked, when in fact both refuse it identically.)
    if printf '%s\n' "$d" | grep -E '^[<>]' | grep -qvE '^[<>] *(warning:|$)'; then
        kind="DIFFER"
    else
        kind="differ (warnings only)"
    fi
    { printf '%s  %s  %s\n' "$kind" "$L" "$P"
      printf '%s\n' "$d" | head -12 | sed 's/^/          /'
    } > "$R"
    rm -f "$R.i" "$R.c"
}

if [ "$CROSS" -eq 1 ]; then
    PAIRS=""
    for gi in languages/*-interpreter.abnf; do
        [ -f "$gi" ] || continue
        L="$(basename "$gi" -interpreter.abnf)"
        [ -f "languages/$L-to-llvm-ir.abnf" ] || continue
        for P in tests/"$L"-test-*; do
            [ -f "$P" ] || continue
            case "$P" in *-test-full.*) continue ;; esac   # the ratchet's own group
            hay="$L $P"
            if [ -n "$FILTER" ] && ! printf '%s' "$hay" | grep -qiF -- "$FILTER"; then continue; fi
            PAIRS="$PAIRS $L|$P"
        done
    done
    if [ -z "$PAIRS" ]; then echo "test.sh --cross: nothing matched"; exit 0; fi
    if [ "$LIST" -eq 1 ]; then printf '%s\n' $PAIRS | tr '|' ' '; exit 0; fi

    SEM="$RESDIR/xsem"; mkfifo "$SEM"; exec 5<>"$SEM"; rm -f "$SEM"
    i=0; while [ "$i" -lt "$JOBS" ]; do printf '.' >&5; i=$((i + 1)); done

    n=0; XREPORTS=""
    for pair in $PAIRS; do
        L="${pair%%|*}"; P="${pair#*|}"
        n=$((n + 1)); r="$RESDIR/x.$(printf '%04d' "$n")"
        XREPORTS="$XREPORTS $r"
        IFS= read -r -n 1 -u 5 _t
        { cross_one "$L" "$P" "$r"; printf '.' >&5; printf '.' >&2; } &
    done
    wait
    echo
    echo "cross-half agreement - the interpreter and the compiler of each language,"
    echo "run on the same program and diffed. The matrix cannot see this class:"
    echo "it compares each engine against ITSELF, never the two halves."
    echo
    nd=0; nw=0
    for r in $XREPORTS; do
        [ -f "$r" ] || continue
        if [ "$(head -1 "$r")" = "agree" ]; then continue; fi
        case "$(head -1 "$r")" in "DIFFER "*) nd=$((nd + 1)) ;; *) nw=$((nw + 1)) ;; esac
        cat "$r"
    done
    echo
    printf '%d programs compared, %d divergent, %d differing only in warnings\n' "$n" "$nd" "$nw"
    echo
    echo "  A divergence is not automatically a bug - the real language decides which"
    echo "  half is right. Settle each against the real toolchain, then fix the wrong"
    echo "  half and pin it with a ratchet assertion."
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
