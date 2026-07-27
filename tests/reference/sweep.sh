#!/usr/bin/env bash
#
# sweep.sh - run the third-party reference corpora through our grammars and
# report, per language, how much of the real language we can already parse.
#
# This is the reference-corpus counterpart of `./test.sh --full`: that ratchet
# walks hand-written tests/<lang>-test-full.<ext> files, this one walks the
# language implementers' own test suites in tests/reference/ (fetch them first
# with ./fetch.sh; see README.md for what each corpus is and how it marks the
# files that are SUPPOSED to be rejected).
#
# For every corpus file it runs
#     mec <grammar> <file> -q -warn-unsupported -warn-imports
# and classifies the outcome by what the run printed:
#
#   parsed      no parse error - the file was understood syntactically. It may
#               still have failed later (no main(), runtime error, unsupported
#               construct); that is a different, later question.
#   PARSE-FAIL  "Not everything could be parsed" - a real grammar gap. The
#               position it died at is recorded, and the token there is what
#               the failure histogram clusters on.
#   looped      the run did not finish in -t seconds, but a second pass with the
#               COMPILER grammar proved the file parses: the program itself runs
#               forever (`for(;;);`, `while(1);` - real, deliberate corpus
#               entries). Counted as parsed, since parsing is what we measure.
#   timeout     the run did not finish and the compiler pass did not vindicate
#               it either - a genuine finding about the parser.
#
# Each corpus file carries an expectation, taken from that suite's own
# conventions (test262 fail/, go's // errorcheck first line, swift's
# expected-error markers, dart's [analyzer]/[cfe] markers, kotlin's
# _ERR/Recovery PSI data, OpenJDK's @compile/fail):
#
#   parse       must parse. Anything else is a gap in our grammar.
#   reject      must NOT parse (test262 fail/ - genuine syntax errors).
#   reject?     upstream negative test, but mostly for SEMANTIC errors (type
#               errors, bad references). A syntax-only front end is entitled to
#               parse these, so accepting them is reported, not judged.
#
# It is informational and always exits 0 - like test.sh --full, the number is a
# progress report, not a pass/fail gate.
#
# Usage:
#   ./sweep.sh                    sweep every language that has a corpus
#   ./sweep.sh kotlin python      sweep only these languages
#   ./sweep.sh -n 200 typescript  sample at most 200 files per language (quick look)
#   ./sweep.sh -f 'when'          only files whose path matches (case-insensitive)
#   ./sweep.sh -k 25 kotlin       show the top 25 failure clusters (default 12)
#   ./sweep.sh -l                 list the languages, their grammar and corpus
#   ./sweep.sh -o results.tsv     also write the per-file results (default: .results/)
#   -j N jobs (default: CPU count)   -t SECS per-file timeout (default 10)
#   -c GRAMMAR-KIND  interpreter (default) or to-llvm-ir
#
# Requires: go (to build the compiler), plus the same timeout/awk basics test.sh uses.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$ROOT/../.." && pwd)"

# ---------------------------------------------------------------------------
# The corpus table: one line per language.
#     <lang> <grammar-stem> <corpus-subdir> <file-extension>
# The expectation of each file is decided by expectations_for() below, which is
# where each suite's own negative-test convention is encoded.
# ---------------------------------------------------------------------------
CORPORA="
js         js         js/test262-parser-tests            js
typescript typescript typescript/typescript-conformance  ts
python     python     python/cpython-tests               py
kotlin     kotlin     kotlin/kotlin-psi-testdata         kt
java       java       java/openjdk-javac-tests           java
ruby       ruby       ruby/ruby-spec-language            rb
php        php        php/php-src-tests/extracted        php
csharp     csharp     csharp/mono-mcs-tests              cs
swift      swift      swift/swift-parse-tests            swift
dart       dart       dart/dart-language-tests           dart
go         go         go/go-test-suite                   go
c          c          c/c-testsuite/tests/single-exec    c
lua        lua        lua/lua-tests                      lua
bash       bash       bash/oils-spec/extracted           sh
batch      batch      batch/wine-cmd-tests               cmd
"

JOBS="$(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)"
# The worker re-execs this script and does not parse options, so it picks the
# timeout up from the environment the driver exported.
TIMEOUT="${MEC_SWEEP_TIMEOUT:-10}"
MAX=0; TOPK=12; LIST=0; FILTER=""; OUT=""; KIND="interpreter"
WANT=""

# --worker is the internal per-file entry point (re-exec of this script).
if [ "${1:-}" = "--worker" ]; then WORKER=1; shift; else WORKER=0; fi

if [ "$WORKER" -eq 0 ]; then
    while [ $# -gt 0 ]; do
        case "$1" in
            -j|--jobs)    JOBS="${2:-1}"; shift ;;
            -t|--timeout) TIMEOUT="${2:-10}"; shift ;;
            -n|--max)     MAX="${2:-0}"; shift ;;
            -k|--top)     TOPK="${2:-12}"; shift ;;
            -f|--filter)  FILTER="${2:-}"; shift ;;
            -o|--out)     OUT="${2:-}"; shift ;;
            -c|--kind)    KIND="${2:-interpreter}"; shift ;;
            -l|--list)    LIST=1 ;;
            -h|--help)    sed -n '2,50p' "$0"; exit 0 ;;
            -*) echo "sweep.sh: unknown option '$1' (try --help)" >&2; exit 2 ;;
            *)  WANT="$WANT $1" ;;
        esac
        shift
    done
fi
case "$JOBS" in ''|*[!0-9]*|0) JOBS=1 ;; esac

# Same timeout-wrapper ladder as test.sh: a corpus file that sends the PEG
# parser into an endless loop must not wedge the sweep.
if command -v timeout >/dev/null 2>&1;    then RUN() { timeout "$TIMEOUT" "$@"; }
elif command -v gtimeout >/dev/null 2>&1; then RUN() { gtimeout "$TIMEOUT" "$@"; }
elif command -v perl >/dev/null 2>&1;     then RUN() { perl -e 'my $t=shift; alarm $t; exec @ARGV or exit 127' "$TIMEOUT" "$@"; }
else RUN() { "$@"; }
fi

# ---------------------------------------------------------------------------
# Worker: run every task in one chunk file and append its result rows.
# argv: <chunk-file> <result-file>
# task line: <lang>|<expect>|<file>       ('|' keeps paths, which may contain
#                                          neither '|' nor whitespace, intact)
# row:  <lang> TAB <expect> TAB <result> TAB <file> TAB <pos> TAB <token>
#
# One process per chunk rather than one per file: mec dominates the cost, and a
# pool of $JOBS long-lived shells avoids both the re-exec per file and xargs
# (whose BSD -I caps a replacement at 255 bytes - shorter than many corpus
# paths - and aborts the whole run when one exceeds it).
# ---------------------------------------------------------------------------
run_one() {
    lang="${1%%|*}"; rest="${1#*|}"
    expect="${rest%%|*}"; file="${rest#*|}"
    grammar="$MEC_SWEEP_GRAMMAR_DIR/$(eval "echo \$MEC_SWEEP_GRAMMAR_$lang")"

    # C is a two-stage language, and this repo models that: c-preprocessor.abnf
    # feeds the C grammar through -pipe, and both stages are first-class matrix
    # entries. Running a .c file straight at the C grammar therefore measures a
    # stage no real toolchain uses alone - cc always preprocesses, and a file
    # holding `#define BAR 0` then `int x = BAR;` is not meaningfully parseable
    # without it. So sweep C the way the matrix does.
    #
    # Measured before adopting: of 30 direct parse failures 3 are recovered, and
    # of 60 files that already parsed 0 break. (An agent had estimated ~9
    # recovered; the measurement says 3. The point is the honest pipeline, not
    # the three files.) Macro EXPANSION lives in the preprocessor and must not
    # be copied into the C grammars.
    pre=""
    if [ "$lang" = "c" ]; then
        pre="$MEC_SWEEP_GRAMMAR_DIR/languages/c-preprocessor.abnf"
        [ -f "$pre" ] || pre=""
    fi

    # Cap the captured output (a corpus program may print without bound) while
    # still learning the run's real exit status: the producer writes it to a
    # side file. If head closed the pipe first the producer dies by SIGPIPE and
    # leaves no status - but a run that printed 64K plainly got past parsing.
    rcf="$MEC_SWEEP_RESDIR/$$.rc"
    rm -f "$rcf"
    if [ -n "$pre" ]; then
        out="$( { RUN "$MEC_SWEEP_BIN" "$pre" "$file" -pipe "$grammar" -q -warn-unsupported -warn-imports 2>&1
                  echo $? > "$rcf"; } | head -c 65536 )"
    else
        out="$( { RUN "$MEC_SWEEP_BIN" "$grammar" "$file" -q -warn-unsupported -warn-imports 2>&1
                  echo $? > "$rcf"; } | head -c 65536 )"
    fi
    rc="$(cat "$rcf" 2>/dev/null || echo 0)"
    case "$rc" in ''|*[!0-9]*) rc=0 ;; esac

    result=parsed; pos=""; tok=""
    case "$out" in
        *"Not everything could be parsed"*)
            result=PARSE-FAIL
            # "Last good parse position: <file>:<line>:<col>" - the token sitting
            # at that spot is the construct our grammar does not know.
            pos="$(printf '%s' "$out" | sed -n 's/.*Last good parse position: \(.*\)/\1/p' | head -1)"
            line="${pos%:*}"; line="${line##*:}"
            col="${pos##*:}"
            case "$line$col" in
                ''|*[!0-9]*) ;;
                *) tok="$(awk -v ln="$line" -v cl="$col" '
                        # The parse died at line:col. Report the first real
                        # token from there on - if the rest of that line is
                        # blank the offending construct starts on a later line,
                        # so keep looking rather than reporting "end of line".
                        FNR < ln { next }
                        {
                            s = (FNR == ln) ? substr($0, cl) : $0
                            sub(/^[ \t]+/, "", s)
                            if (s == "") next
                            if (match(s, /^[A-Za-z_$][A-Za-z_0-9$]*/))      t = substr(s, 1, RLENGTH)
                            else if (match(s, /^[^A-Za-z_0-9 \t]+/))        t = substr(s, 1, RLENGTH)
                            else                                            t = substr(s, 1, 8)
                            print t; exit
                        }' "$file" 2>/dev/null | head -1)" ;;
            esac
            ;;
    esac
    # 124 = GNU timeout; the perl fallback dies on SIGALRM (142).
    if [ "$rc" -eq 124 ] || [ "$rc" -eq 142 ]; then
        result=timeout
        # A timeout is ambiguous: either the PARSER went pathological (a real
        # finding) or the file parsed fine and the PROGRAM runs forever -
        # `for(;;);` and `while(1);` are legitimate, deliberate corpus entries,
        # and an interpreter grammar cannot be step-capped because execution is
        # the annotation scripts running, not the IR interpreter.
        #
        # The compiler grammar settles it. It is a static pass: it parses, emits
        # IR, and only then runs that IR under -max-steps, so a runaway program
        # stops at the step limit instead of hanging. If the compiler gets far
        # enough to emit IR or trip the step limit, the PARSE was fine and the
        # interpreter timeout was execution, not a grammar gap - which is what
        # this sweep measures. Only timeouts pay for this second pass.
        cgram="$(printf '%s' "$grammar" | sed 's/-interpreter\.abnf$/-to-llvm-ir.abnf/')"
        if [ "$cgram" != "$grammar" ] && [ -f "$cgram" ]; then
            if [ -n "$pre" ]; then
                cout="$(RUN "$MEC_SWEEP_BIN" "$pre" "$file" -pipe "$cgram" -q \
                            -warn-unsupported -warn-imports -max-steps 2000000 2>&1 | head -c 4096)"
            else
                cout="$(RUN "$MEC_SWEEP_BIN" "$cgram" "$file" -q -warn-unsupported \
                            -warn-imports -max-steps 2000000 2>&1 | head -c 4096)"
            fi
            case "$cout" in
                *"Not everything could be parsed"*) ;;   # genuinely unparseable
                *"step limit exceeded"*|*"@str."*|*"define "*|*"declare "*)
                    result=looped ;;
            esac
        fi
    fi

    printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$lang" "$expect" "$result" "$file" "$pos" "$tok" \
        >> "$OUTFILE"
}

if [ "$WORKER" -eq 1 ]; then
    CHUNK="$1"; OUTFILE="$2"
    : > "$OUTFILE"
    n=0
    while IFS= read -r task; do
        [ -z "$task" ] && continue
        run_one "$task"
        n=$((n + 1))
        [ $((n % 50)) -eq 0 ] && printf '.' >&2
    done < "$CHUNK"
    exit 0
fi

# ---------------------------------------------------------------------------
# Driver
# ---------------------------------------------------------------------------
selected() { # $1 = lang
    [ -z "$WANT" ] && return 0
    local w
    for w in $WANT; do [ "$w" = "$1" ] && return 0; done
    return 1
}

if [ "$LIST" -eq 1 ]; then
    printf '%-11s %-34s %-36s %s\n' LANG GRAMMAR CORPUS STATUS
    while read -r lang stem dir ext; do
        [ -z "$lang" ] && continue
        g="languages/$stem-$KIND.abnf"
        [ -f "$REPO/$g" ] || g="$g (MISSING)"
        if [ -d "$ROOT/$dir" ]; then st="$(find "$ROOT/$dir" -name "*.$ext" | wc -l | tr -d ' ') files"
        else st="not fetched"; fi
        printf '%-11s %-34s %-36s %s\n' "$lang" "$g" "$dir" "$st"
    done <<< "$CORPORA"
    exit 0
fi

command -v go >/dev/null 2>&1 || { echo "sweep.sh: 'go' not found on PATH" >&2; exit 2; }

BIN="$(mktemp -t mec-sweep.XXXXXX)"
TASKS="$(mktemp -t mec-tasks.XXXXXX)"
RESDIR="$(mktemp -d -t mec-sweepres.XXXXXX)"
TMP="$(mktemp -d -t mec-sweeptmp.XXXXXX)"
trap 'rm -rf "$BIN" "$TASKS" "$RESDIR" "$TMP"' EXIT

echo "building compiler..."
(cd "$REPO" && go build -o "$BIN" .) || { echo "sweep.sh: build failed" >&2; exit 2; }

# expectations_for <lang> <corpus-dir> <ext>
# Emits "<expect>\t<file>" for every corpus file, encoding each suite's own
# convention for which files are supposed to be rejected. See README.md.
expectations_for() {
    local lang="$1" dir="$ROOT/$2" ext="$3"
    local all="$TMP/all.$lang" neg="$TMP/neg.$lang"
    find "$dir" -name "*.$ext" -type f | sort > "$all"
    [ -n "$FILTER" ] && { grep -i -- "$FILTER" "$all" > "$all.f" || true; mv "$all.f" "$all"; }
    : > "$neg"

    case "$lang" in
        js)
            # test262-parser-tests: pass/ and early/ must parse (an early error
            # is a static-semantics rule, not a grammar rule); fail/ must not.
            #
            # Two fail/ files are STALE, not gaps. This corpus is a frozen
            # snapshot that predates ES2022 class fields, so it still files
            #     (class {a})     98204d734f8c72b3.js
            #     (class {a=0})   ef81b93cf9bdb4ec.js
            # as syntax errors. They are valid modern JavaScript and accepting
            # them is correct. Verified against node v24 parsed with the true
            # Script goal symbol (new vm.Script(src), not `node --check`, whose
            # CommonJS wrapper legalises top-level `return` and would have
            # excused 13 more files that are genuinely our bugs). Every other
            # fail/ file we accept is a real grammar gap - do not grow this list
            # without the same check.
            awk -v d="$dir" '
                { stale = ($0 ~ /(98204d734f8c72b3|ef81b93cf9bdb4ec)\.js$/)
                  print ($0 ~ d "/fail/" && !stale ? "reject|" : "parse|") $0 }' "$all"
            return ;;
        typescript)
            # The conformance suite has no per-file error baseline, but it does
            # segregate its deliberately-malformed inputs by DIRECTORY, and those
            # are syntax errors rather than semantic ones: parser/ErrorRecovery
            # ('return {' unterminated at top level, '().toString()'),
            # parser/SkippedTokens, and decorators/invalid ('@dec type T = ...',
            # which the TS grammar itself refuses). Demanding that we parse them
            # was wrong - refusing them is the correct behaviour, so they carry
            # the hard 'reject' expectation and we are scored on refusing them.
            awk '{ print ($0 ~ /\/(ErrorRecovery|SkippedTokens)\// || $0 ~ /\/decorators\/invalid\// \
                          ? "reject|" : "parse|") $0 }' "$all"
            return ;;
        go)
            # The first line of every golang/go test/ file is its directive;
            # "// errorcheck" means the compiler must reject the file.
            tr '\n' '\0' < "$all" \
                | xargs -0 awk 'FNR==1 && /^\/\/ *errorcheck/ { print FILENAME }' \
                | sort > "$neg" ;;
        swift)  grep -rl 'expected-error' "$dir" --include="*.$ext" 2>/dev/null | sort > "$neg" ;;
        dart)   grep -rl -e '\[analyzer\]' -e '\[cfe\]' "$dir" --include="*.$ext" 2>/dev/null | sort > "$neg" ;;
        java)   grep -rl '@compile/fail' "$dir" --include="*.$ext" 2>/dev/null | sort > "$neg" ;;
        kotlin)
            # Kotlin ships the EXPECTED PSI TREE next to every input, as a .txt
            # companion, and that is the exact signal: a PsiErrorElement in the
            # tree is the Kotlin parser itself recording a SYNTAX error, so the
            # file must not parse. Anything else must parse.
            #
            # This is deliberately NOT keyed on the '// COMPILATION_ERRORS'
            # marker, which was tried first and is wrong: of the 450 files
            # carrying it, 174 have a perfectly clean PSI tree - their errors
            # are SEMANTIC (type errors, unresolved references), which a
            # syntax-only front end is required to parse. Keying on the marker
            # excused us from 174 files we ought to handle.
            #
            # Because the signal is exact, kotlin gets the HARD 'reject'
            # expectation (like test262's fail/), not the informational
            # 'reject?': we are scored on actually refusing these.
            # Two companion FORMATS, and only one of them can hold a
            # PsiErrorElement. testData/psi/ ships a parse TREE; testData/lexer/
            # ships a TOKEN DUMP, where the lexer records a rejected character
            # as BAD_CHARACTER (and an unterminated string as DANGLING_NEWLINE)
            # and the word PsiErrorElement never appears. Testing only for
            # PsiErrorElement therefore demanded that we parse every lexer file,
            # including the ones Kotlin itself refuses - which is how the
            # vertical tab, NEL, LS and PS files came to look like grammar gaps
            # when in fact kotlinc rejects them and so should we. (Only the form
            # feed is real Kotlin whitespace: pageBreak.txt absorbs it into
            # WHITE_SPACE. Both VT and FF are legal INSIDE a string literal.)
            while IFS= read -r f; do
                t="${f%.$ext}.txt"
                if [ -f "$t" ] && grep -qE 'PsiErrorElement|BAD_CHARACTER|DANGLING_NEWLINE' "$t"; then
                    printf 'reject|%s\n' "$f"
                else
                    printf 'parse|%s\n' "$f"
                fi
            done < "$all"
            return ;;
    esac

    if [ -s "$neg" ]; then
        comm -12 "$all" "$neg" | sed 's/^/reject?|/'
        comm -23 "$all" "$neg" | sed 's/^/parse|/'
    else
        sed 's/^/parse|/' "$all"
    fi
}

: > "$TASKS"
LANGS=""
while read -r lang stem dir ext; do
    [ -z "$lang" ] && continue
    selected "$lang" || continue
    grammar="languages/$stem-$KIND.abnf"
    if [ ! -f "$REPO/$grammar" ]; then
        echo "  [skip] $lang: no $grammar"; continue
    fi
    if [ ! -d "$ROOT/$dir" ]; then
        echo "  [skip] $lang: corpus $dir not fetched (./fetch.sh $lang)"; continue
    fi
    n_before="$(wc -l < "$TASKS" | tr -d ' ')"
    expectations_for "$lang" "$dir" "$ext" \
        | { [ "$MAX" -gt 0 ] && head -n "$MAX" || cat; } \
        | sed "s@^@$lang|@" >> "$TASKS"
    n_after="$(wc -l < "$TASKS" | tr -d ' ')"
    if [ "$n_after" -eq "$n_before" ]; then
        echo "  [skip] $lang: no files matched"; continue
    fi
    eval "export MEC_SWEEP_GRAMMAR_$lang=\"\$grammar\""
    LANGS="$LANGS $lang"
done <<< "$CORPORA"

TOTAL="$(wc -l < "$TASKS" | tr -d ' ')"
[ "$TOTAL" -eq 0 ] && { echo "sweep.sh: nothing to sweep"; exit 0; }

export MEC_SWEEP_BIN="$BIN" MEC_SWEEP_RESDIR="$RESDIR" MEC_SWEEP_GRAMMAR_DIR="$REPO"
export MEC_SWEEP_TIMEOUT="$TIMEOUT"
echo "sweeping $TOTAL files across$LANGS ($JOBS jobs, ${TIMEOUT}s timeout)..."

cd "$REPO" || exit 2
# Deal the tasks round-robin into $JOBS chunks (round-robin, not contiguous, so
# one language's slow files cannot all land on the same worker), then run one
# worker per chunk and wait for the pool.
awk -v n="$JOBS" -v d="$TMP" '{ print > (d "/chunk." (NR % n)) }' "$TASKS"
for i in $(seq 0 $((JOBS - 1))); do
    [ -f "$TMP/chunk.$i" ] || continue
    "$ROOT/sweep.sh" --worker "$TMP/chunk.$i" "$RESDIR/$i.tsv" &
done
wait
echo

RESULTS="$TMP/results.tsv"
# Each worker writes its own file (no interleaved appends), so there can be tens
# of thousands of them - collect with find, not a glob that would blow ARG_MAX.
find "$RESDIR" -name '*.tsv' -exec cat {} + 2>/dev/null | sort > "$RESULTS"
if [ -n "$OUT" ]; then cp "$RESULTS" "$OUT"; SAVED="$OUT"
else mkdir -p "$ROOT/.results" && cp "$RESULTS" "$ROOT/.results/sweep.tsv"; SAVED="$ROOT/.results/sweep.tsv"; fi

# ---------------------------------------------------------------------------
# Scoreboard
# ---------------------------------------------------------------------------
echo
awk -F'\t' '
{
    lang = $1; expect = $2; result = $3
    langs[lang] = 1
    n[lang, expect]++
    got[lang, expect, result]++
}
function pct(a, b) { return b > 0 ? sprintf("%5.1f%%", 100 * a / b) : "    -" }
END {
    printf "%-11s %-9s %7s  %-16s %-14s %6s %s\n", "LANGUAGE", "EXPECT", "FILES", "PARSED", "PARSE-FAIL", "LOOPED", "TIMEOUT"
    printf "%s\n", "----------------------------------------------------------------------------------------"
    for (lang in langs) order[++k] = lang
    # tiny insertion sort - awk has no portable sort of array values
    for (i = 2; i <= k; i++) { v = order[i]; j = i - 1
        while (j >= 1 && order[j] > v) { order[j+1] = order[j]; j-- } order[j+1] = v }
    for (i = 1; i <= k; i++) {
        lang = order[i]
        for (ei = 1; ei <= 3; ei++) {
            expect = (ei == 1 ? "parse" : (ei == 2 ? "reject" : "reject?"))
            tot = n[lang, expect]; if (tot == 0) continue
            lp = got[lang, expect, "looped"]+0
            # A "looped" file parsed - the compiler pass proved it - so it
            # counts as PARSED. It is still shown separately, because a corpus
            # full of runaway programs is worth knowing about.
            ok = got[lang, expect, "parsed"]+0 + lp
            pf = got[lang, expect, "PARSE-FAIL"]+0
            to = got[lang, expect, "timeout"]+0
            printf "%-11s %-9s %7d  %6d %s   %6d %s  %6d %5d\n", \
                   (ei == 1 ? lang : ""), expect, tot, ok, pct(ok, tot), pf, pct(pf, tot), lp, to
        }
    }
    print ""
    print "  parse   = must parse; PARSED% is the score to ratchet upward."
    print "  reject  = genuine syntax errors; here PARSE-FAIL% is the score (we correctly refused)."
    print "  reject? = upstream negative tests that are mostly SEMANTIC (type/reference errors);"
    print "            a syntax-only front end may legitimately parse them - informational only."
    print "  LOOPED  = parsed, but the program itself never terminates (for(;;); and friends)."
    print "            Counted inside PARSED: the grammar understood the file, which is what"
    print "            this sweep measures. TIMEOUT is what is left - a real parser finding."
}' "$RESULTS"

# Failure clusters: the token sitting where the parse died, biggest first. This
# is the actionable half of the report - each cluster names a construct the
# grammar does not know yet.
echo
echo "Top parse-failure clusters (token at the position the parse died, expect=parse only):"
for lang in $LANGS; do
    clusters="$(awk -F'\t' -v l="$lang" \
        '$1 == l && $2 == "parse" && $3 == "PARSE-FAIL" && $6 != "" { print $6 }' "$RESULTS" \
        | sort | uniq -c | sort -rn | head -n "$TOPK")"
    [ -z "$clusters" ] && continue
    printf '\n  %s:\n' "$lang"
    printf '%s\n' "$clusters" | awk '{ n = $1; $1 = ""; sub(/^ /, ""); printf "    %6d  %s\n", n, $0 }'
done

echo
echo "per-file results: $SAVED"
echo "  e.g. awk -F'\\t' '\$1==\"kotlin\" && \$3==\"PARSE-FAIL\"' $SAVED | head"
exit 0
