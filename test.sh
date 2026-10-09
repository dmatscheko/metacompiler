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
#      and -frozen stdout must be identical (the frozen bootstrap must match goja),
#      and so must their stderr - that is where warnings and error messages go.
#   2. correct exit status: the language tests are self-checking (exit 0 == the
#      program's own checks passed). Ordinary entries must exit 0 on both engines;
#      the by-design failures (names containing "FAIL", plus the smaller-match-first
#      and infinite-loop grammar guards) must exit non-zero on both engines.
#   3. expect-nonzero NATIVE rows: `-exe PATH` only builds, so an entry that names
#      SHOULD ABORT is built (which must succeed), then the built BINARY is run and
#      must exit non-zero with the same OUTPUT AND ABORT TEXT as the same program
#      under llvm.Run. That is how the C floor's abort path is reached by a test
#      rather than by hand. An entry that also says ALL HALVES adds the third
#      implementation - the language's own *-interpreter.abnf - so all three must
#      agree on the diagnostic. See abort_norm() for exactly what is normalised
#      (the engine naming itself, and nothing else) and why this is compared
#      engine-against-engine rather than against a checked-in .expected fixture.
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
#   ./test.sh -j, --jobs N    run N entries in parallel (1 = sequential).
#                             Default: 2x CPU count for the MATRIX, 1x for --full
#                             and --cross - measured, see the note at JOBS below.
#   ./test.sh -t, --timeout N per-run timeout in seconds. THE DEFAULT IS PER GROUP:
#                             120 for the matrix and --cross, 900 for --full.
#                             See the note at TIMEOUT below for the measurement.
#   ./test.sh --full          run the SECOND test group: the full-syntax ratchet
#                             files (tests/*-test-full.*). For every grammar it
#                             reports the language areas that do not work yet -
#                             the goal is full language support, and this is the
#                             progress report. A language GAP is informational;
#                             a probe run KILLED by the timeout is NOT, and exits
#                             non-zero (see "KILLED" in the summary).
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

FILTER=""; VERBOSE=0; LIST=0; TIMEOUT=""; TIMEOUT_EXPLICIT=0; FULL=0; CROSS=0
NCPU="$(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)"
JOBS=""; JOBS_EXPLICIT=0
while [ $# -gt 0 ]; do
    case "$1" in
        -f|--filter)  FILTER="${2:-}"; shift ;;
        --filter=*)   FILTER="${1#*=}" ;;
        -v|--verbose) VERBOSE=1 ;;
        -l|--list)    LIST=1 ;;
        -j|--jobs)    JOBS="${2:-1}"; JOBS_EXPLICIT=1; shift ;;
        --jobs=*)     JOBS="${1#*=}"; JOBS_EXPLICIT=1 ;;
        -t|--timeout) TIMEOUT="${2:-120}"; TIMEOUT_EXPLICIT=1; shift ;;
        --timeout=*)  TIMEOUT="${1#*=}"; TIMEOUT_EXPLICIT=1 ;;
        --full)       FULL=1 ;;
        --cross)      CROSS=1 ;;
        -h|--help)    sed -n '2,60p' "$0"; exit 0 ;;
        *) echo "test.sh: unknown option '$1' (try --help)" >&2; exit 2 ;;
    esac
    shift
done
# THE DEFAULT JOB COUNT IS PER GROUP, and the reason is measured rather than
# assumed (docs/todo.md 7.1).
#
# The MATRIX is 351 short, independent subprocesses, most of which spend a large
# part of their life not on a core at all - process start, reading the grammar,
# writing results. At one job per CPU it therefore does NOT saturate the machine:
# `./test.sh` was measured at 35.0 s and 768% CPU against `-j 32`'s 25.4 s at
# 1143%, same result. Re-measured 2026-08-08 as six INTERLEAVED A/B pairs, three
# with the default first and three with -j 32 first so the ordering cannot
# explain it: -j 32 was faster in all six (130.5/82.6, 76.5/60.5, 58.8/54.4,
# 66.3/48.0, 51.4/50.6, 51.6/50.6 s, default/32). Those runs were taken on a
# machine at load 40-70 from other work, which compresses the margin - the win is
# largest when there is idle capacity, which is exactly the case being fixed.
# Twelve runs, six of them oversubscribed: 351/351 every time, no flakiness.
#
# --full and --cross keep one job per CPU. --full is TAIL-BOUND on ~16 ratchet
# files - one of them (js) dominates the whole group, so more jobs cannot help
# (measured 95 s -> 91 s) - and each of its probes holds a whole ratchet file's
# compile in memory, which is the one place oversubscription costs something real.
# --cross measured 12 s either way.
#
# An explicit -j always wins, for every group.
if [ "$JOBS_EXPLICIT" = 0 ]; then
    if [ "$FULL" -eq 1 ] || [ "$CROSS" -eq 1 ]; then JOBS="$NCPU"; else JOBS=$((NCPU * 2)); fi
fi
case "$JOBS" in ''|*[!0-9]*|0) JOBS=1 ;; esac

# THE DEFAULT PER-RUN TIMEOUT IS ALSO PER GROUP, and for the same reason: the
# groups are not the same size. The matrix is 383 short runs and 120 s is many
# times the slowest of them. --full is not: it runs the whole ratchet FILE of a
# language through one grammar, four times per language.
#
# MEASURED on this machine at 1c76ec8, the four legs of the biggest ratchet
# (tests/js-test-full.js), each a SINGLE run and therefore each subject to this
# timeout on its own:
#
#     js-interpreter                110 s
#     js-interpreter  -frozen       423 s     <- the slowest run in the suite
#     js-to-llvm-ir                  42 s
#     js-to-llvm-ir   -frozen       171 s
#
# The `-frozen` interpreter leg is the tail because there the MetaJS engine is
# itself a MetaJS program running on the IR interpreter. tests/gates.sh measured
# the same leg at 456 s idle and 482 s after the fifteenth round's additions, so
# treat ~480 s as the number and 423 s here as the same figure on a quieter
# moment - NOT as a second, smaller measurement.
#
# 900 s is that with ~1.9x of headroom, and it is deliberately the same constant
# gates.sh already passes in as FULL_TIMEOUT: the bare command a human types by
# hand and the gate should not have different brakes, because the whole point of
# docs/todo.md 4.3 is that they DID, and the bare command was the one that
# reported a kill as though it were a defect. If js grows much further the answer
# is to split its ratchet, not to keep raising this.
#
# --cross keeps 120: its slowest program is a feature file, not a ratchet.
if [ "$TIMEOUT_EXPLICIT" = 0 ]; then
    if [ "$FULL" -eq 1 ]; then TIMEOUT=900; else TIMEOUT=120; fi
fi
case "$TIMEOUT" in ''|*[!0-9]*) TIMEOUT=120 ;; esac

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

# full_section_table prints one TAB-separated record per SECTION of $1:
#
#     id <TAB> name <TAB> first line <TAB> last line
#
# IT IS THE ONE PLACE THAT DECIDES WHERE A SECTION ENDS, and it exists because
# the previous answer - "the section region ends at `===== END SECTIONS =====`" -
# had silently stopped being true in ELEVEN of the sixteen ratchet files, and a
# twelfth never had the marker at all (docs/todo.md 4.2, which named three of the
# eleven). Both awks below used to `exit` at that marker, so every
# section after it was invisible: it could not be isolated, it could not be
# deleted-and-retried, and a grammar choking inside one reported
#
#     error at line N outside the sections (prologue/main must stay minimal)
#
# and gave up on the whole file - one broken section hiding every gap behind it.
# Measured at f1910a9, sections living AFTER the marker: go 18, typescript 13,
# js 12, python 11, bash 9, java 6, ruby 4, csharp 3, dart 3, kotlin 3, lua 1 =
# 83, plus the last section of tests/metajs-test-full.js, which has no marker at
# all and so ran off the end of the scan. 84 of 627 sections, unreachable.
#
# WHY THIS RATHER THAN MOVING THE MARKER IN EACH FILE. Moving it is the smaller
# diff and it is the fragile one: the marker is a hand-maintained line in sixteen
# files that different people append sections to, and it has already drifted in
# eleven of them and gone missing in a twelfth. Nothing checks it, because it
# IS the check. The boundary is derivable from structure that cannot drift - a
# section header, and the SECTION-CALL driver that every one of these files must
# have to run at all - so it is derived here instead, and the marker becomes one
# optional hint among three rather than the only one. Ruby settles the argument
# on its own: its driver is SPLIT, calls 01-38, then sections 40-43, then calls
# 39-43, so there is no single position a marker could be moved to.
#
# The rule, in priority order. A section starts at its header and ends at the
# line before the first ANCHOR after it:
#
#   * the next SECTION header               - it is not part of any section
#   * an `===== END SECTIONS =====` marker   - same, and still honoured where it
#                                             is placed correctly
#   * the start of a DRIVER BLOCK - the first line of a run of `SECTION-CALL <nn>`
#     lines - backed up over the lines that OPEN it, to the nearest blank line.
#     That back-up is what keeps `function main() {` (and C#'s `static int Main()`
#     plus its brace, and PHP's two `global` lines) out of the deletion; it never
#     crosses a blank line, a header or a marker, and it is only ever applied to
#     a driver anchor, because a header and a marker are lines that belong to
#     nobody while a driver block has scaffolding above it.
#   * nothing - end of file. That is the right answer for java, whose main() sits
#     in the PROLOGUE and whose last section runs to the last line of the file.
#
# The name is extracted exactly as before, so multi-line section headers (bash 33,
# csharp 33) keep reading the way they always did.
#
# CHECKED RATHER THAN ARGUED, over all sixteen files: every one of the 543
# sections the old scan could see keeps BYTE-IDENTICAL bounds here, the 84 it
# could not see are now bounded, no two ranges overlap, no range runs past the
# end of its file, and no range contains a `SECTION-CALL <nn>` line - which is
# the property that says a deletion cannot eat the driver.
full_section_table() {
    awk '
        { line[NR] = $0 }
        /===== SECTION [0-9]+:/ {
            s = $0; sub(/.*SECTION /, "", s)
            i = s; sub(/:.*/, "", i)
            nm = s; sub(/^[0-9]*: */, "", nm); sub(/ =====.*/, "", nm)
            nsec++; sid[nsec] = i; snm[nsec] = nm; sst[nsec] = NR
            kind[NR] = "header"; prevcall = 0; next
        }
        /===== END SECTIONS/ { kind[NR] = "marker"; prevcall = 0; next }
        {
            iscall = ($0 ~ /SECTION-CALL [0-9]/)
            if (iscall && !prevcall) kind[NR] = "driver"
            prevcall = iscall
        }
        END {
            for (k = 1; k <= nsec; k++) {
                a = 0
                for (j = sst[k] + 1; j <= NR; j++) if (kind[j] != "") { a = j; break }
                if (a == 0) {
                    e = NR
                } else if (kind[a] == "driver") {
                    e = a - 1
                    while (e > sst[k] && line[e] !~ /^[ \t]*$/) e--
                    if (e <= sst[k]) e = a - 1
                } else {
                    e = a - 1
                }
                printf "%s\t%s\t%d\t%d\n", sid[k], snm[k], sst[k], e
            }
        }
    ' "$1"
}

# full_error_msg extracts the most telling line of a failed run's output.
full_error_msg() {
    printf '%s\n' "$1" | grep -i 'error\|not implemented' | grep -v '^  ==> Fail' | head -1 | cut -c1-160 | grep . \
        || printf '%s\n' "$1" | grep -v '^  ==> Fail' | grep . | head -1 | cut -c1-160
}

# full_isolate classifies each remaining section of $2 by running it alone
# (prologue + that section + main with only its call). Prints one line per
# still-failing section.
full_isolate() {
    local G="$1" work="$2" iso="$2.iso" tbl="$2.tbl"
    local id ids name rc out
    full_section_table "$work" > "$tbl"
    ids="$(cut -f1 "$tbl")"
    for id in $ids; do
        # Keep the skeleton (anything in no section's range) plus the one section
        # being classified. The ranges come from full_section_table, so a section
        # after the END SECTIONS marker is isolated like any other - before
        # docs/todo.md 4.2 every such section was treated as skeleton and was
        # therefore present in EVERY isolation run, which is how one unparsable
        # late section could fail all of them.
        awk -v keep="$id" -v tbl="$tbl" '
            BEGIN {
                while ((getline t < tbl) > 0) {
                    split(t, f, "\t")
                    for (i = f[3]; i <= f[4]; i++) owner[i] = f[1]
                }
                close(tbl)
            }
            {
                if (owner[NR] != "" && owner[NR] != keep) next
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
            name="$(awk -F'\t' -v want="$id" '$1 == want { print $2; exit }' "$tbl")"
            printf '    - %s %s: %s\n' "$id" "$name" "$(full_error_msg "$out")"
        fi
    done
    rm -f "$iso" "$tbl"
}

# full_probe writes the gap report for one grammar/test-file pair to $3.
full_probe() {
    local G="$1" F="$2" R="$3"
    local work="$R.work"
    cp "$F" "$work"
    local gaps=0 iter=0 rc out ln reason impl info id name start end
    # THREE facts come out of this probe, and until 2026-08-10 they were crammed
    # into ONE file ($R.checks), which is docs/todo.md 6.1:
    #
    #   $R.checks  the assertion COUNT, or '-' when nothing was measured
    #   $R.flag    WHAT HAPPENED: FULL | PARTIAL | FROZEN-DIFF | RUN-FAILED |
    #              VACUOUS | UNMEASURED
    #   $R.gaps    how many SECTIONS had to be deleted to get there
    #
    # Overloading one file meant every non-FULL outcome DESTROYED the count. A
    # half with one gap wrote no count at all, a FROZEN-DIFF half overwrote its
    # count with the word, and both then contributed ZERO to the ratchet total -
    # a language that ran 375 of its 384 assertions was reported as '-' and
    # subtracted 384 from the number this project is gated on. Measured at
    # db6bf43 by inserting one unparsable line into tests/lua-test-full.lua:
    # `lua -` and `0 assertions in total; 0 languages with halves that disagree`.
    # Splitting them means the count survives every outcome that produced one.
    echo -          > "$R.checks"
    echo UNMEASURED > "$R.flag"
    echo 0          > "$R.gaps"
    {
        # The FILE is named beside the grammar: a language whose ratchet is split runs
        # the same grammar pair twice, and two identical headers in the report would be
        # unreadable (docs/todo.md 6.2).
        printf '%s (%s):\n' "$(basename "$G" .abnf)" "$(basename "$F")"
        while :; do
            iter=$((iter + 1))
            if [ "$iter" -gt 60 ]; then printf '    (stopped after 60 rounds)\n'; break; fi
            out="$(RUN "$BIN" "$G" "$work" -q 2>&1)"; rc=$?
            # A run KILLED BY THE PER-RUN TIMEOUT is not a language gap, and until
            # this arm existed it was reported as one - SILENTLY, which is the worst
            # available outcome here.
            #
            # The path: the kill leaves $out empty, so no 'Last good parse position',
            # no 'not implemented' and no 'line N' can be extracted, and the probe
            # fell through to full_isolate / 'error without a usable position' and
            # broke out of the loop WITHOUT EVER WRITING $R.checks. The count then
            # reads '-'. If both halves of a language were killed - which is the
            # normal case, because whatever is slow is slow in both - the two halves
            # AGREE on '-', the CHECK-COUNT report prints no MISMATCH, no
            # RUN-FAILED/FROZEN-DIFF line exists anywhere in the output for
            # tests/gates.sh to grep, and the only trace is that the total assertion
            # count is smaller. Observed for real: gates.sh reported the ratchet 437
            # assertions below baseline with no failure reported anywhere.
            #
            # 124 is timeout(1)/gtimeout(1); 142 is the perl-alarm fallback (SIGALRM);
            # 137/143 are a KILL/TERM from anything else.
            case "$rc" in
                124|137|142|143)
                    printf '    RUN-FAILED - the probe run was KILLED (exit %s) after the %ss --timeout.\n' "$rc" "$TIMEOUT"
                    printf '                 This is NOT a language gap and NOT a frozen divergence: nothing\n'
                    printf '                 was measured. Re-run on a quieter machine, or raise --timeout.\n'
                    echo RUN-FAILED > "$R.flag"
                    break ;;
            esac
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
                    echo VACUOUS > "$R.flag"
                    break
                fi
                # THE COUNT IS RECORDED WHETHER OR NOT SECTIONS HAD TO BE REMOVED
                # (docs/todo.md 6.1). It used to be recorded only in the gaps == 0
                # arm, so a half with a single gap printed "(the remaining sections
                # pass)" - true, and it never said how many, and the summary read
                # '-'. The rest of the file is a REAL MEASUREMENT of the rest of
                # the file; refusing to report it does not make the ratchet safer,
                # it makes the number this project gates on drop by a whole
                # language while the report says only that some sections passed.
                #
                # The count and the gap count are both reported; the total adds the
                # count, the summary names the gaps, and tests/gates.sh has a
                # recorded FULL_GAPS ceiling so a NEW gap is still red.
                {
                    # Record how many assertions actually ran. "FULL" only says the
                    # file got to the end; it cannot see a section whose checks
                    # stopped executing (an early return, a swallowed exception, a
                    # loop that runs zero times), which silently lowers the count
                    # while staying green. The two halves of a language must agree
                    # on it - see the CHECK-COUNT report at the end of this group.
                    printf '%s\n' "$out" | sed -n 's/^full: \([0-9]*\) checks.*/\1/p' | head -1 > "$R.checks"
                    printf '%s\n' "$gaps" > "$R.gaps"
                    # stderr is compared too, not discarded: warnings live there
                    # (they used to pollute stdout), so a ratchet file that starts
                    # warning under one engine only would otherwise pass unseen.
                    # Every ratchet file runs with an EMPTY stderr today, on both
                    # engines and both halves - measured, all 15 languages.
                    # $work, NOT $F: with gaps == 0 it is a byte copy of $F, and
                    # with gaps > 0 it is the file MINUS the sections that do not
                    # parse - which is the only version of the file this half can
                    # run at all. Comparing goja against -frozen on $F when the
                    # gaps > 0 path had reached here would compare two failures.
                    RUN "$BIN" "$G" "$work" -q          > "$work.g" 2>"$work.ge"; grc=$?
                    RUN "$BIN" "$G" "$work" -q -frozen  > "$work.f" 2>"$work.fe"; frc=$?
                    # A run killed by the timeout leaves a truncated output file,
                    # which cmp would report as a divergence. Only a kill is
                    # RUN-FAILED; any other non-zero exit is a real
                    # goja-vs-frozen divergence and falls through to the else arm.
                    killed=0
                    for rc in "$grc" "$frc"; do
                        case "$rc" in 124|137|142|143) killed=1 ;; esac
                    done
                    if [ "$killed" -eq 1 ]; then
                        printf '    RUN-FAILED - goja exit %s, -frozen exit %s: killed by the %ss --timeout;\n' \
                            "$grc" "$frc" "$TIMEOUT"
                        printf '                 this is NOT a frozen divergence - re-run, or raise --timeout\n'
                        echo - > "$R.checks"; echo RUN-FAILED > "$R.flag"
                    elif [ "$grc" -eq 0 ] && [ "$frc" -eq 0 ] && cmp -s "$work.g" "$work.f" && cmp -s "$work.ge" "$work.fe"; then
                        if [ "$gaps" -eq 0 ]; then
                            echo FULL > "$R.flag"
                            printf '    FULL - %s assertions, goja and -frozen byte-identical\n' "$(cat "$R.checks")"
                        else
                            echo PARTIAL > "$R.flag"
                            printf '    PARTIAL - %s assertions from the sections that DO run (%s gap(s) above), goja and -frozen byte-identical\n' \
                                "$(cat "$R.checks")" "$gaps"
                        fi
                    else
                        printf '    %s under goja, BUT -frozen fails or differs - inspect before celebrating\n' \
                            "$([ "$gaps" -eq 0 ] && echo FULL || echo PARTIAL)"
                        # And make the SUMMARY say so. The count above was recorded
                        # from the goja run, so without this the end-of-group report
                        # compares two goja numbers, finds them equal, and prints "0
                        # languages with halves that disagree" while this very line
                        # says otherwise. Found 2026-08-02 by an agent whose frozen
                        # half was silently losing every store (ASI: a bare `return`
                        # followed by an expression on the next line) - 17 of 59 new
                        # assertions red in ONE half, invisible to the summary that
                        # everyone actually quotes.
                        # The COUNT IS KEPT, and only the flag says FROZEN-DIFF.
                        # It used to overwrite $R.checks, which meant a language
                        # whose halves differ under -frozen ALSO silently
                        # subtracted its whole ratchet from the total - the same
                        # defect as 6.1, one arm over. The flag is what the
                        # summary and tests/gates.sh read.
                        echo FROZEN-DIFF > "$R.flag"
                    fi
                }
                break
            fi
            # Where did it choke? Parse failures, "not implemented" aborts and
            # generic "line N" messages carry a position; everything else goes
            # through the per-section isolation fallback.
            #
            # TWO SPELLINGS, and the first one has not existed for a long time.
            # abnf/parser.go:1288 says
            #
            #     Not everything could be parsed. Last good parse position: <file>:<ln>:<col>
            #
            # and the pattern here only knew the older `was ln N,` wording, so it
            # matched NOTHING: every parse failure in every ratchet fell through to
            # the isolation fallback and was reported as "error without a usable
            # position", which is the slowest path and the least informative one.
            # Found while probing docs/todo.md 4.2 - a deliberately broken section
            # in lua was reported as 25 VACUOUS sections plus one real gap, and the
            # dead sed is half of why. Both spellings are read now; the old one
            # costs a line and cannot fire wrongly.
            ln="$(printf '%s\n' "$out" | sed -n 's/.*Last good parse position was ln \([0-9][0-9]*\),.*/\1/p' | head -1)"
            if [ -z "$ln" ]; then
                ln="$(printf '%s\n' "$out" \
                      | sed -n 's/.*Last good parse position: .*:\([0-9][0-9]*\):[0-9][0-9]*.*/\1/p' | head -1)"
            fi
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
            info="$(full_section_table "$work" \
                    | awk -F'\t' -v L="$ln" '$3 <= L && L <= $4 { print; exit }')"
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
    rm -f "$work" "$work.n" "$work.iso" "$work.g" "$work.f" "$work.ge" "$work.fe"
}

if [ "$FULL" -eq 1 ]; then
    FILES="$(ls tests/*-test-full.* 2>/dev/null)"
    if [ -z "$FILES" ]; then echo "test.sh --full: no tests/*-test-full.* files found" >&2; exit 2; fi
    if [ "$LIST" -eq 1 ]; then printf '%s\n' $FILES; exit 0; fi

    SEM="$RESDIR/fsem"
    mkfifo "$SEM"; exec 4<>"$SEM"; rm -f "$SEM"
    i=0; while [ "$i" -lt "$JOBS" ]; do printf '.' >&4; i=$((i + 1)); done

    # A LANGUAGE MAY HAVE ITS RATCHET IN MORE THAN ONE FILE, and the two names it
    # needs are then different things (docs/todo.md 6.2):
    #
    #   lang  picks the GRAMMAR PAIR, and comes from the name up to '-test-full.'
    #   unit  is the ACCOUNTING KEY of the check-count agreement below, and is one
    #         per FILE
    #
    # tests/js-test-full.js and tests/js-test-full.2.js are both js, so both run
    # through languages/js-{interpreter,to-llvm-ir}.abnf - but their assertion counts
    # are of DIFFERENT FILES and must not be compared with each other. The agreement
    # rule is "the two halves of one language run the same file, so they must agree";
    # what that is really about is one FILE through two grammars, which is what `unit`
    # keys. Grouping both files under "js" would have reported the split as a
    # MISMATCH - the most serious word this group prints - for no defect at all.
    #
    # The suffix a second file carries is the part between '-test-full.' and the
    # extension, so js-test-full.2.js is unit "js2". It stays [a-z0-9]+ on purpose:
    # tests/gates.sh detects a half that recorded no count with a
    # `^  [a-z0-9]+ +-$` grep over the per-unit lines below, and a label with a dash
    # or a dot in it would slip past that detector.
    n=0; REPORTS=""
    for f in $FILES; do
        base="$(basename "$f")"; lang="${base%%-test-full.*}"
        unit="$lang"
        sfx="${base#*-test-full.}"           # '<part>.<ext>' for a split file, '<ext>' otherwise
        case "$sfx" in *.*) unit="$lang${sfx%%.*}" ;; esac
        for G in "languages/$lang-interpreter.abnf" "languages/$lang-to-llvm-ir.abnf"; do
            hay="$lang $unit $G $f"
            if [ -n "$FILTER" ] && ! printf '%s' "$hay" | grep -qiF -- "$FILTER"; then continue; fi
            n=$((n + 1))
            r="$RESDIR/full.$(printf '%03d' "$n")"
            REPORTS="$REPORTS $r"
            printf '%s\n' "$unit" > "$r.lang"
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

    # CHECK-COUNT agreement. A grammar half that reaches the end of the ratchet
    # file reports FULL, and that is the headline - but it is possible to be FULL
    # while running FEWER assertions than the other half of the same language,
    # which means one half is quietly skipping work the other does. The ratchet
    # file is one file, so the two halves must produce the SAME count; anything
    # else is a defect in whichever half runs fewer. This is cheap and
    # self-maintaining: it needs no expected number checked into the repo.
    echo
    echo "assertion counts (both halves run the SAME FILE, so they must agree)"
    # A KILLED RUN IS NOT A MISMATCH, and calling it one is docs/todo.md 4.3.
    # MISMATCH is the most serious word this group prints - it means the two
    # halves of a language computed different numbers of assertions from one
    # file - and until now a run the harness KILLED was reported in exactly those
    # words. Four gate runs in one round printed a red line for a timeout rather
    # than for a defect, which is how a suite teaches people to ignore it. So the
    # two are separated here: a kill says it was killed, says what the brake was,
    # and says how to raise it.
    #
    # It stays a FAILURE, though, and in three independent ways, because the
    # exact hole this project has already fallen into is the opposite one (see
    # the FULL_ASSERTIONS comment in tests/gates.sh): killed probes recorded no
    # count, both halves therefore "agreed", and the summary printed 437
    # assertions below baseline as though that were the answer.
    #   1. the per-language line still carries the literal token RUN-FAILED,
    #      which is what tests/gates.sh greps for;
    #   2. the total line SAYS SO ON ITSELF, and is followed by an explicit
    #      statement naming what each killed language did and did not contribute;
    #   3. this group exits NON-ZERO when anything was killed - the only case in
    #      which it does. A language GAP remains informational, exit 0.
    #
    # WHAT A KILLED LANGUAGE CONTRIBUTES TO THE TOTAL (docs/todo.md 4.4). If one
    # half was killed and the other finished, the survivor's count IS added, and
    # that is a deliberate choice re-argued rather than inherited from e1816b3:
    # zeroing the language would move the total FURTHER from the answer - by that
    # language's whole ratchet, typically hundreds of assertions - and the number
    # people read next to it is FULL_ASSERTIONS, a floor. A total short by 1,500
    # because a run was killed and a total short by 1,500 because someone deleted
    # a section look identical, and only one of them is a defect. Keeping the
    # survivor's count means the residual delta still measures assertion LOSS.
    #
    # WHAT ACTUALLY CHANGES, then, is the READING of it, not the arithmetic:
    # the total line was UNMARKED. It read exactly like a clean one, so the number
    # could be quoted - or written into FULL_ASSERTIONS - with the disclaimer
    # three lines below it left behind. The line now carries the disclaimer
    # itself, and the banner names the surviving half's number instead of leaving
    # the reader to work out which number is the suspect one. The leading
    # "<N> assertions in total" spelling is preserved for tests/gates.sh's sed:
    # it must still SEE the number to report it short.
    #
    # One SUSPECTED half of the defect turned out not to be live, and saying so is
    # worth more than pretending it was. The contribution is v[0], the first half
    # in the sorted list, which looks order-dependent - kill the interpreter and
    # get 0, kill the compiler and get the full count. It is not: MEASURED at
    # f1910a9 by injecting rc=124 for one grammar, a one-half kill on lua gave
    # `384 assertions in total` BOTH ways, because the feeding pipeline sorts
    # WHOLE LINES and "384" sorts before "RUN-FAILED". The asymmetry is latent,
    # not live; it is spelled out at the site below rather than left resting on
    # the ASCII order of a token name.
    awk_out="$RESDIR/full.summary"
    for r in $REPORTS; do
        [ -f "$r.lang" ] || continue
        printf '%s\t%s\t%s\t%s\n' "$(cat "$r.lang")" \
            "$(cat "$r.checks" 2>/dev/null || echo -)" \
            "$(cat "$r.flag"   2>/dev/null || echo UNMEASURED)" \
            "$(cat "$r.gaps"   2>/dev/null || echo 0)"
    done | sort | awk -F'\t' -v tmo="$TIMEOUT" '
        { if ($1 != last) { if (last != "") emit(); last = $1; n = 0 }
          v[n] = $2; f[n] = $3; g[n] = $4 + 0; n++ }
        # THE COUNT AND THE VERDICT ARE NOW SEPARATE FIELDS (docs/todo.md 6.1).
        # v[] is the assertion count or "-"; f[] is FULL/PARTIAL/FROZEN-DIFF/
        # RUN-FAILED/VACUOUS/UNMEASURED; g[] is the number of sections that had to
        # be deleted before the half would run. Previously ONE field carried all
        # three, so every non-FULL outcome erased the count and the language
        # contributed 0 - the total silently stopped being a measurement, which is
        # the whole reason this item exists.
        function emit(   i, bad, killed, nocount, got, gs, ng) {
            bad = 0; killed = 0; nocount = 0; got = ""; gs = 0; ng = 0
            for (i = 1; i < n; i++) if (v[i] != v[0]) bad = 1
            for (i = 0; i < n; i++) {
                # FROZEN-DIFF: this half is FULL/PARTIAL under goja but differs
                # under -frozen. Flag it even when BOTH halves carry it, because
                # two halves failing the same way is not agreement - it is two
                # defects.
                if (f[i] == "FROZEN-DIFF") bad = 1
                # RUN-FAILED is the harness saying it stopped the run. Not a
                # disagreement; reported separately.
                if (f[i] == "RUN-FAILED") killed = 1
                # NO COUNT AT ALL. Before this field existed the only trace of
                # this was a "-" in the count column, which was caught by
                # tests/gates.sh only BY OVERLAP: both halves silent read as
                # agreement and produced no MISMATCH, and one half silent printed
                # "MISMATCH: - 384" and was caught by the OTHER grep. It is a
                # named category now and gates.sh looks for it by name.
                if (v[i] !~ /^[0-9]+$/) nocount = 1
                gs += g[i]; if (g[i] > 0) ng++
            }
            if (n < 2)        printf "  %-12s %s (only one half ran - filtered?)\n", last, v[0]
            else if (killed) { printf "  %-12s KILLED (RUN-FAILED):", last
                               for (i = 0; i < n; i++) printf " %s", (f[i] == "RUN-FAILED" ? "RUN-FAILED" : v[i])
                               printf " - NOT a halves divergence, nothing was measured\n"
                               nkilled++; killednames = killednames " " last }
            else if (nocount) { printf "  %-12s NO-COUNT:", last
                                for (i = 0; i < n; i++) printf " %s", (v[i] ~ /^[0-9]+$/ ? v[i] : f[i])
                                printf " - a half recorded NO ASSERTION COUNT; whatever it runs is MISSING from the total\n"
                                nnocount++; nocountnames = nocountnames " " last }
            else if (bad)   { printf "  %-12s MISMATCH:", last
                              for (i = 0; i < n; i++) printf " %s%s", v[i], (f[i] == "FROZEN-DIFF" ? "(FROZEN-DIFF)" : "")
                              printf "\n"; mismatch++ }
            else if (gs > 0) { printf "  %-12s %s  PARTIAL: %d section gap(s) summed over %d half/halves - the count is of the REST of the file\n", last, v[0], gs, ng
                               ngappy++; gapnames = gapnames sprintf(" %s(%d)", last, gs); totgaps += gs }
            else            printf "  %-12s %s\n", last, v[0]
            # WHICH value goes into the total: the FIRST NUMERIC one, in every
            # case. This used to be v[0] except when killed, which was right only
            # by an ASCII accident the old comment here spelled out at length -
            # the feeding pipeline sorts WHOLE LINES, so "lua<TAB>384" sorted
            # before "lua<TAB>RUN-FAILED" because "3" < "R". Renaming a token
            # would have moved the total. Scanning for the first numeric is the
            # same answer for every case that has one and does not depend on the
            # sort order at all.
            for (i = 0; i < n; i++) if (v[i] ~ /^[0-9]+$/ && got == "") got = v[i]
            if (got != "") total += got
            if (killed) {
                if (got != "") survived = survived sprintf("\n  %s contributed %s, from the ONE half that finished; the other measured nothing.", last, got)
                else           survived = survived sprintf("\n  %s contributed NOTHING - both halves were killed.", last)
            }
            if (!killed && nocount) {
                if (got != "") missing = missing sprintf("\n  %s contributed %s, from the half that DID record a count; the other recorded none.", last, got)
                else           missing = missing sprintf("\n  %s contributed NOTHING - NO half recorded a count.", last)
            }
        }
        END { if (last != "") emit()
              # KEEP THE LEADING "<N> assertions in total" SPELLING: tests/gates.sh
              # reads the number out of it with a sed on "^ *N assertions in total"
              # and must keep seeing it, so that a short read is reported as short
              # rather than as "no total at all". Everything after the number is
              # free text, which is where the kill disclaimer goes - ON THE LINE,
              # so the number cannot be quoted without it (docs/todo.md 4.4).
              #
              # THE TOTAL SAYS WHAT KIND OF NUMBER IT IS. Three shapes, and only
              # the last one is a clean measurement:
              #   killed   - a run was stopped; nothing was measured for it
              #   no-count - a half produced no count; its work is missing
              #   gaps     - sections were deleted to make the file run; the
              #              count is of what remains and is HONEST, just smaller
              # A gap is the case docs/todo.md 6.1 is about: it used to make the
              # WHOLE language contribute 0 and print "-", so one unparsable line
              # in lua took 384 assertions off the total and the summary still
              # read "0 languages with halves that disagree".
              if (nkilled > 0)
                  printf "\n  %d assertions in total - NOT A MEASUREMENT, %d language(s) killed, see below; %d languages with halves that disagree\n", total, nkilled, mismatch
              else if (nnocount > 0)
                  printf "\n  %d assertions in total - INCOMPLETE, %d language(s) recorded no count, see NO-COUNT below; %d languages with halves that disagree\n", total, nnocount, mismatch
              else if (totgaps > 0)
                  printf "\n  %d assertions in total, from %d language(s) with %d SECTION GAP(S) skipped, see GAPS below; %d languages with halves that disagree\n", total, ngappy, totgaps, mismatch
              else
                  printf "\n  %d assertions in total; %d languages with halves that disagree\n", total, mismatch
              if (nnocount > 0) {
                  printf "\n  NO-COUNT: %d language(s) had a half that recorded NO ASSERTION COUNT AT ALL:%s\n", nnocount, nocountnames
                  printf "  That half did not report a `full: N checks` line, so the total above is missing\n"
                  printf "  whatever it runs. It is NOT a gap (a gap still measures the rest of the file)\n"
                  printf "  and NOT a kill (that is reported separately). Usual causes: VACUOUS - main() is\n"
                  printf "  not being run; UNMEASURED - the probe hit an error it could not attribute to a\n"
                  printf "  section, or the grammar is missing. tests/gates.sh greps for the NO-COUNT: line\n"
                  printf "  above BY NAME - it is the designed detector, not the two greps that used to\n"
                  printf "  catch this class only by overlapping each other.\n"
                  printf "  What the affected languages put into the total above:%s\n", missing
              }
              if (totgaps > 0) {
                  printf "\n  GAPS: %d section-gap(s), summed per half, across %d language(s), deleted before the file would run:%s\n", totgaps, ngappy, gapnames
                  printf "  Each of those languages still CONTRIBUTED its remaining assertions to the total\n"
                  printf "  above - that is the fix for docs/todo.md 6.1, and it is why the total falls by\n"
                  printf "  the size of the deleted sections rather than by a whole language. The gap list\n"
                  printf "  per grammar is in the per-language report above. tests/gates.sh has a recorded\n"
                  printf "  FULL_GAPS ceiling, so a NEW gap is red even though the total barely moves.\n"
              }
              if (nkilled > 0) {
                  printf "\n  %d LANGUAGE(S) WERE KILLED BY THE %s s --timeout:%s\n", nkilled, tmo, killednames
                  printf "  A killed run records NO COUNT, so THE TOTAL ABOVE IS NOT A MEASUREMENT - it\n"
                  printf "  is a smaller number with holes in it, and that is the exact failure mode the\n"
                  printf "  FULL_ASSERTIONS floor in tests/gates.sh exists to catch. Nothing diverged and\n"
                  printf "  nothing ran. Re-run on a quieter machine, or:  ./test.sh --full --timeout %d\n", tmo * 2
                  # Name the suspect number instead of leaving it to be worked out.
                  # A half that finished IS counted (the alternative moves the total
                  # further from the answer, by a whole ratchet file) - so
                  # say which languages those are and how much each of them put in.
                  printf "  What the killed languages put into the total above:%s\n", survived
              }
            }' | tee "$awk_out"
    # exit 1 iff something was killed OR a half recorded no count at all.
    # Language GAPS stay informational (exit 0) - that is what this group is for,
    # and the gap is reported, counted and ratcheted by tests/gates.sh's FULL_GAPS
    # instead. A NO-COUNT half is different in kind: it is a half of the suite
    # that produced no measurement, which is the failure mode this whole group's
    # comments are written against, so it is a non-zero exit as well as a named
    # line. The verdict is read back out of the text that was just printed rather
    # than carried in a variable, because the awk above runs in a pipeline and its
    # subshell cannot set one.
    if grep -q 'WERE KILLED BY THE' "$awk_out" 2>/dev/null; then exit 1; fi
    if grep -q '^  NO-COUNT: ' "$awk_out" 2>/dev/null; then exit 1; fi
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

# strip_module drops a compiler grammar's emitted LLVM module from the FRONT of a
# run's stdout, leaving the program's own output. It is shared by cross_strip
# (group three) and abort_norm (the SHOULD ABORT rows of the matrix) - one
# implementation, so the two groups cannot disagree about where a module ends.
strip_module() {
    awk '
        # Drop the emitted module, keep the program output that follows it.
        #
        # This used to search for the LAST line that looked like the end of IR
        # and cut there. That is wrong, and it silently truncated the very
        # output this group exists to compare: the PROGRAM can print such a
        # line - php var_dump closes an array with a bare "}" - so the cut
        # jumped PAST real output. Worse, it did so ASYMMETRICALLY, because the
        # interpreter half has no module at all: one half lost the lines before
        # its last "}" and the other lost module-plus-output, so a genuine
        # divergence could cancel out into a clean diff.
        #
        # Scan forward with brace depth instead. The module is a prefix: at
        # depth 0 only @globals, declare, define and blanks belong to it, and
        # the first depth-0 line that is none of those begins the output. On
        # interpreter output nothing matches at line 1, so it all passes
        # through, which is what makes the two halves comparable.
        BEGIN { depth = 0; done = 0 }
        done { print; next }
        {
            if (depth > 0) { if ($0 ~ /^}/) depth = 0; next }
            if ($0 ~ /^define /) { depth = 1; if ($0 ~ /}[ \t]*$/) depth = 0; next }
            if ($0 ~ /^@/ || $0 ~ /^declare / || $0 ~ /^source_filename/ ||
                $0 ~ /^target / || $0 ~ /^;/ || $0 ~ /^!/ ||
                $0 ~ /^%[A-Za-z_.$]/ || $0 ~ /^attributes /) { next }
            if ($0 ~ /^[ \t]*$/ && !seenout) { next }
            done = 1; seenout = 1; print
        }'
}

cross_strip() {
    # A NUL byte reaches here from grammars that embed one in a string constant;
    # command substitution drops it anyway and warns about it on stderr, so drop
    # it explicitly and identically for both halves.
    local all; all="$(LC_ALL=C tr -d '\000')"
    printf '%s\n' "$all" | grep -E '^warning: ' || true
    printf '%s\n' "$all" | grep -vE '^warning: ' \
    | strip_module \
    | grep -Ev '^[a-z#+-]+ (interpreter|compiler): |^  ==> Fail$|^[a-z#+-]+ (interpreter|compiler) error: ' \
    | grep -Ev '^(js runtime error: |jsmain\(\) returned )' \
    | grep -Ev '^[a-z0-9-]+: (exit status|main\(\) returned|program value is) [0-9-]+$' \
    | grep -Ev '^--- Executed with the built-in IR interpreter, output:$' \
    | grep -Ev '^IR interpreter: ' \
    | awk 'NF { blank = 0; for (i = 1; i <= held; i++) print ""; held = 0; print; seen = 1; next }
           seen { held++ }'
}

# ----------------------------------------------------------------------------
# abort_norm - the normaliser behind the SHOULD ABORT rows (docs/todo.md 4.1).
#
# WHY THIS EXISTS. Until now nothing in the suite could ratchet an ABORT TEXT.
# The matrix compares each engine against ITSELF (goja vs -frozen), so a
# wrong-but-consistent message passes; --cross deliberately DELETES abort lines
# before diffing (see cross_strip's comment) so the two halves aborting for the
# same cause do not read as a divergence; and an -exe row only BUILDS. That is
# how b56c2b0 came to find five halves divergences in abort messages by hand,
# one of them llvm.Run leaking a Go type name ((*abnf.jsArray)) into
# user-visible output.
#
# The property that was missing is ENGINE AGAINST ENGINE on the diagnostic
# itself, and it is deliberately NOT a checked-in .expected fixture: a stderr
# fixture has to be blessed every time a message legitimately changes, whereas
# "the three implementations of this language must say the same thing" is
# self-maintaining and is exactly the invariant that was broken.
#
# WHAT IS NORMALISED, and nothing else. An over-normalised comparison that can
# never fail is worse than no gate at all, so this list is short and every entry
# is the ENGINE IDENTIFYING ITSELF, never part of the program's diagnostic:
#
#   * the emitted module (strip_module) - only a compiler leg has one, and it is
#     that grammar's output, not the program's.
#   * llvm.Run's own "  ==> Fail" verdict line and its "--- Executed with the
#     built-in IR interpreter" banner - the METACOMPILER saying the module
#     returned non-zero, which a native binary has no analogue for.
#   * the harness sign-offs: "<lang> interpreter: program ran to completion" and
#     its three sibling wordings, "jsmain() returned N", "<name>: exit status N".
#   * the leading engine tag ONLY - "js runtime error: " and
#     "<lang> interpreter error: " / "<lang> compiler error: " - keeping the
#     whole diagnostic that follows it. This is the one substitution that could
#     hide something, and it cannot hide a message, only the name of the engine
#     that printed it.
#
# NOT normalised, on purpose: file names, line numbers, quoted values, class
# names, exception spelling. Every leg is handed the SAME source path, so a
# ruby diagnostic naming "tests/ruby-abort-test.rb:6" is comparable as-is - and
# the moment two engines disagree about a name or a number, that IS the finding.
#
# stdout and stderr are concatenated (stdout first) rather than compared
# separately, because WHICH STREAM the diagnostic uses is itself a difference
# between the legs: the interpreter halves print it on stdout, llvm.Run and the
# native binaries on stderr. Concatenating per leg is deterministic and keeps
# the program's own pre-abort output in the comparison too.
#
#   abort_norm <stdout-file> <stderr-file>
abort_norm() {
    cat "$1" "$2" \
    | LC_ALL=C tr -d '\000' \
    | strip_module \
    | grep -Ev '^  ==> Fail$' \
    | grep -Ev '^--- Executed with the built-in IR interpreter, output:$' \
    | grep -Ev '^[a-z#+._-]+ (interpreter|compiler): ' \
    | grep -Ev '^jsmain\(\) returned -?[0-9]+$' \
    | grep -Ev '^[A-Za-z0-9#+._-]+: (exit status|main\(\) returned|program value is) -?[0-9]+$' \
    | sed -E 's/^js runtime error: //; s/^[a-z#+._-]+ (interpreter|compiler) error: //'
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
    # THE SAME KILLED-RUN HAZARD AS THE RATCHET, and it was live here too
    # (docs/todo.md 4.3 asked; this is what was found). This used to be
    #     oi="$(RUN ... | cross_strip)"; ri=$?
    # which records the exit status of CROSS_STRIP, not of the run - so it was
    # always 0 and was never read again. A leg killed by the per-run timeout
    # therefore arrived as TRUNCATED output and was compared as though it were an
    # answer: one leg killed reads as DIFFER (a defect that is not there), and
    # BOTH legs killed - the normal case, because whatever is slow is slow in both
    # halves - reads as "agree" and the group prints "0 divergent" while having
    # measured nothing at all. That last one is silent, which is worse.
    #
    # So capture the RUN's own status, from a file rather than a pipeline, and
    # report a kill as its own kind. 124 is timeout(1)/gtimeout(1), 142 is the
    # perl-alarm fallback, 137/143 are KILL/TERM from anything else. Only those
    # four codes: a program that merely EXITS non-zero is normal here (several
    # test programs abort on purpose) and must keep being compared.
    RUN "$BIN" -i tests/imports "$gi" "$P" -q -warn-unsupported -warn-imports >"$R.iraw" 2>&1; ri=$?
    RUN "$BIN" -i tests/imports "$gc" "$P" -q -warn-unsupported -warn-imports >"$R.craw" 2>&1; rc=$?
    oi="$(cross_strip < "$R.iraw")"
    oc="$(cross_strip < "$R.craw")"
    rm -f "$R.iraw" "$R.craw"
    case "$ri:$rc" in
        124:*|137:*|142:*|143:*|*:124|*:137|*:142|*:143)
            printf 'KILLED  %s  %s\n' "$L" "$P" > "$R"
            printf '          a leg was KILLED (interpreter exit %s, compiler exit %s) after the %ss\n' \
                "$ri" "$rc" "$TIMEOUT" >> "$R"
            printf '          --timeout. NOTHING WAS COMPARED - this is not a divergence and not an\n' >> "$R"
            printf '          agreement. Re-run on a quieter machine, or raise --timeout.\n' >> "$R"
            return ;;
    esac

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
    nd=0; nw=0; nk=0
    for r in $XREPORTS; do
        [ -f "$r" ] || continue
        if [ "$(head -1 "$r")" = "agree" ]; then continue; fi
        case "$(head -1 "$r")" in
            "DIFFER "*) nd=$((nd + 1)) ;;
            "KILLED "*) nk=$((nk + 1)) ;;
            *)          nw=$((nw + 1)) ;;
        esac
        cat "$r"
    done
    echo
    printf '%d programs compared, %d divergent, %d differing only in warnings\n' "$((n - nk))" "$nd" "$nw"
    # A killed pair is subtracted from the compared count above rather than
    # folded into it, and it is reported on its own line, because "compared" has
    # to mean compared. tests/gates.sh keys off ", 0 divergent" in the line
    # above; the non-zero exit below is what makes a kill reach a caller that
    # only looks at status.
    if [ "$nk" -gt 0 ]; then
        echo
        printf '%d PAIR(S) WERE KILLED BY THE %ss --timeout and were NOT COMPARED. The counts\n' "$nk" "$TIMEOUT"
        printf 'above are therefore over a SMALLER SET, which is not a result. Re-run when the\n'
        printf 'machine is quiet, or:  ./test.sh --cross --timeout %d\n' "$((TIMEOUT * 2))"
    fi
    echo
    echo "  A divergence is not automatically a bug - the real language decides which"
    echo "  half is right. Settle each against the real toolchain, then fix the wrong"
    echo "  half and pin it with a ratchet assertion."
    [ "$nk" -gt 0 ] && exit 1
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

    # EXPECT-NONZERO NATIVE ROWS. `mec ... -exe PATH` only BUILDS: it links the
    # binary, prints the path and exits 0, so every -exe entry above asks "does
    # this link" and nothing else. The three tests/metajs-fail-test*.js programs
    # are the ones whose whole point is the ABORT, and their native halves were
    # therefore not run by anything - which is exactly why the C floor's abort
    # path (werr, die, die2, die3, wstr, o_cstr, class_name) read as untested
    # while the coverage sweep called it "proved by probe".
    #
    # An entry whose name says SHOULD ABORT declares: the BUILD must succeed
    # (rc 0, so it is NOT a "SHOULD FAIL" entry and the word "fail" must stay out
    # of its name, or the rule above would demand the opposite), and then the
    # built BINARY must be run and must exit non-zero.
    #
    # And the binary's OUTPUT is compared against the SAME program under
    # llvm.Run - the Go twin, the same entry minus `-exe PATH`. That is the check
    # with the teeth: it pins the floor's diagnostic wording to the Go runtime's,
    # in a place where the two implementations are completely separate code. A
    # bare "it exited non-zero" would pass on a segfault.
    #
    # A name that ALSO says ALL HALVES adds the third implementation: the same
    # program under languages/<lang>-interpreter.abnf. Three independent
    # implementations of one language's abort text, compared against each other
    # (see abort_norm above for exactly what is normalised and what is not).
    # It is a separate token because it is not universally green yet - several
    # languages have a live interpreter-vs-compiler divergence in their abort
    # text, and those rows carry the two-leg form until their owner closes it.
    # Promoting a row is a one-word edit in .vscode/launch.json.
    local abort_exe=0 abort_all=0 exe="" i
    case "$name" in *"SHOULD ABORT"*) abort_exe=1 ;; esac
    case "$name" in *"ALL HALVES"*) abort_all=1 ;; esac
    local twin=() interp=()
    if [ "$abort_exe" -eq 1 ]; then
        i=0
        while [ "$i" -lt "${#args[@]}" ]; do
            if [ "${args[$i]}" = "-exe" ]; then exe="${args[$((i + 1))]}"; i=$((i + 2)); continue; fi
            twin+=("${args[$i]}"); i=$((i + 1))
        done
        # The interpreter half of the same language, if this row names a
        # compiler grammar and that half exists.
        interp=("${twin[@]}")
        interp[0]="${interp[0]/-to-llvm-ir.abnf/-interpreter.abnf}"
        [ "${interp[0]}" = "${twin[0]}" ] && interp=()
        [ "${#interp[@]}" -gt 0 ] && [ ! -f "${interp[0]}" ] && interp=()
    fi
    local ng="$RESDIR/$idx.ng" nf="$RESDIR/$idx.nf" nt="$RESDIR/$idx.nt" ni="$RESDIR/$idx.ni"
    local rc_ng=0 rc_nf=0 rc_ni=0

    local rc_g rc_f
    RUN "$BIN" "${args[@]}"          >"$og" 2>"$eg"; rc_g=$?
    [ "$abort_exe" -eq 1 ] && [ -x "$exe" ] && { RUN "$exe" >"$ng.o" 2>"$ng"; rc_ng=$?; }
    RUN "$BIN" "${args[@]}" -frozen  >"$of" 2>"$ef"; rc_f=$?
    [ "$abort_exe" -eq 1 ] && [ -x "$exe" ] && { RUN "$exe" >"$nf.o" 2>"$nf"; rc_nf=$?; }
    [ "$abort_exe" -eq 1 ] && RUN "$BIN" "${twin[@]}" >"$nt.o" 2>"$nt"
    if [ "$abort_exe" -eq 1 ] && [ "$abort_all" -eq 1 ] && [ "${#interp[@]}" -gt 0 ]; then
        RUN "$BIN" "${interp[@]}" >"$ni.o" 2>"$ni"; rc_ni=$?
    fi

    # A KILLED RUN IS NEVER A PASS, whatever the row expects (docs/todo.md 4.3).
    # This is checked FIRST and outside the should_fail branch because of the
    # trap: the by-design-failing rows - every name containing FAIL, and the
    # SHOULD ABORT rows below - assert only that the exit status is NON-ZERO, and
    # timeout(1) kills with 124, which is non-zero. So a row whose whole purpose
    # is to fail would have gone GREEN on a machine too loaded to run it, and the
    # matrix would have reported a pass for a run that never finished. The four
    # codes are timeout/gtimeout's 124, the perl-alarm fallback's 142, and
    # KILL/TERM's 137/143.
    local problems=()
    killed_rc() { case "$1" in 124|137|142|143) return 0 ;; *) return 1 ;; esac; }
    killed_rc "$rc_g" && problems+=("goja was KILLED (exit $rc_g) after the ${TIMEOUT}s --timeout - nothing was measured, raise --timeout")
    killed_rc "$rc_f" && problems+=("frozen was KILLED (exit $rc_f) after the ${TIMEOUT}s --timeout - nothing was measured, raise --timeout")
    if [ "$should_fail" -eq 1 ]; then
        [ "$rc_g" -eq 0 ] && problems+=("goja exited 0 but should fail")
        [ "$rc_f" -eq 0 ] && problems+=("frozen exited 0 but should fail")
    else
        { [ "$rc_g" -ne 0 ] && ! killed_rc "$rc_g"; } && problems+=("goja exit $rc_g")
        { [ "$rc_f" -ne 0 ] && ! killed_rc "$rc_f"; } && problems+=("frozen exit $rc_f")
    fi
    if [ "$abort_exe" -eq 1 ]; then
        if [ -z "$exe" ]; then
            problems+=("SHOULD ABORT entry with no -exe PATH in its args")
        elif [ ! -x "$exe" ]; then
            problems+=("-exe produced no executable at $exe")
        else
            [ "$rc_ng" -eq 0 ] && problems+=("the goja-built native binary exited 0 but should abort")
            [ "$rc_nf" -eq 0 ] && problems+=("the frozen-built native binary exited 0 but should abort")
            # ...and the same trap one level down: "must exit non-zero" is
            # satisfied by the timeout killing it.
            killed_rc "$rc_ng" && problems+=("the goja-built native binary was KILLED (exit $rc_ng) after the ${TIMEOUT}s --timeout, it did not abort")
            killed_rc "$rc_nf" && problems+=("the frozen-built native binary was KILLED (exit $rc_nf) after the ${TIMEOUT}s --timeout, it did not abort")
            killed_rc "$rc_ni" && problems+=("the interpreter half was KILLED (exit $rc_ni) after the ${TIMEOUT}s --timeout, it did not abort")
            cmp -s "$ng" "$nf" || problems+=("goja-built vs frozen-built native stderr differ")
            cmp -s "$ng.o" "$nf.o" || problems+=("goja-built vs frozen-built native stdout differ")
            # The ABORT TEXT COMPARISON (docs/todo.md 4.1). abort_norm strips the
            # emitted module and each engine's own self-identification and
            # nothing else, so what is compared here IS the program's diagnostic,
            # and it is compared between implementations that share no code.
            abort_norm "$ng.o" "$ng" > "$ng.n"
            abort_norm "$nt.o" "$nt" > "$nt.n"
            if ! cmp -s "$ng.n" "$nt.n"; then
                problems+=("ABORT TEXT: the native binary and llvm.Run disagree
             native:   $(head -3 "$ng.n" | tr '\n' '|')
             llvm.Run: $(head -3 "$nt.n" | tr '\n' '|')")
            fi
            if [ "$abort_all" -eq 1 ]; then
                if [ "${#interp[@]}" -eq 0 ]; then
                    problems+=("ALL HALVES row, but no interpreter grammar for it")
                else
                    [ "$rc_ni" -eq 0 ] && problems+=("the interpreter half exited 0 but should abort")
                    abort_norm "$ni.o" "$ni" > "$ni.n"
                    if ! cmp -s "$ni.n" "$ng.n"; then
                        problems+=("ABORT TEXT: the interpreter half and the native binary disagree
             interp:   $(head -3 "$ni.n" | tr '\n' '|')
             native:   $(head -3 "$ng.n" | tr '\n' '|')")
                    fi
                fi
            fi
        fi
    fi
    if [ "$has_q" -eq 1 ] && ! cmp -s "$og" "$of"; then
        problems+=("goja vs frozen -q output differ")
    fi
    # STDERR is compared on exactly the same entries as stdout. Two reasons:
    #
    #   1. warnings live here now (they used to be printed to stdout, where they
    #      polluted a compiler's emitted module and a program's output). Without
    #      this line, moving them off stdout would silently DROP the 35 entries
    #      that pass -warn-imports / -warn-unsupported out of the byte-identity
    #      check - a coverage loss disguised as a clean change.
    #   2. it closes an older gap: an ERROR message that differs between the two
    #      engines was invisible to this matrix.
    #
    # Adding it found two entries (Kotlin delegation SHOULD FAIL, PHP undefined
    # constant SHOULD FAIL) whose stderr had diverged all along: a panic from the
    # RUNNING PROGRAM was re-panicked by the frozen engine wrapped in the whole
    # startScript source (~4800 lines), where goja printed two lines. That is
    # fixed at the source (jsProgramPanic in abnf/jsrt.go), not excluded here -
    # and the fix also retired the documented exception, the MetaJS type
    # violation entry, which had to run without -q for exactly that reason and
    # now carries -q like everything else.
    #
    # Entries that still differ on stderr between the engines are the trace and
    # timing ones (-vv/-vv2 tag traces, -speed) - all of them without -q, so they
    # are excluded here exactly as they are for stdout.
    if [ "$has_q" -eq 1 ] && ! cmp -s "$eg" "$ef"; then
        problems+=("goja vs frozen -q stderr differ")
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
                 head -3 "$eg" | sed 's/^/         goja: /'
                 # Only when the two stderrs differ is the frozen side worth
                 # showing - otherwise it is the same three lines twice.
                 cmp -s "$eg" "$ef" || head -3 "$ef" | sed 's/^/         frozen: /')"
        printf '%s\n' "$block" > "$RESDIR/$idx.fail"
        printf '%s\n' "$block"
    fi
    rm -f "$og" "$of" "$eg" "$ef" \
          "$ng" "$ng.o" "$ng.n" "$nf" "$nf.o" "$nt" "$nt.o" "$nt.n" "$ni" "$ni.o" "$ni.n"
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
