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
# _ERR/Recovery PSI data, OpenJDK's @compile/fail and its javac/diags/examples
# '// key: compiler.err.*' lines):
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
            #
            # Everything below that is a PER-FILE correction, and every entry was
            # settled with node v24 parsed at the true Script goal
            # (new vm.Script(src) - `node --check` wraps the file in a CommonJS
            # function and is not a faithful oracle). Nothing here may be added
            # without the same check.

            # (a) The suite's own marker comment, "Shouldn't work, negatives are
            # not allowed", on the unicodeExtendedEscapes ...14 files (one each
            # for Strings, Templates and RegularExpressions). The grep that read
            # it used to run over the WHOLE corpus and caught three more files
            # whose "shouldn't work" is about module resolution and type
            # semantics, not syntax - node/nodeModules1.ts,
            # node/allowJs/nodeModulesAllowJs1.ts ("dynamic import() always uses
            # the esm resolver") and salsa/typeFromPropertyAssignment29.ts
            # ("must be const", "classes already have statics"). They parse, and
            # they are supposed to. Scope the grep to the directory the marker
            # was found in, which is what the rule always meant.
            neg_ts="$TMP/neg.tsmark"
            grep -il "shouldn't work\|should not work" \
                 $(grep '/unicodeExtendedEscapes/' "$all") 2>/dev/null | sort > "$neg_ts" || : > "$neg_ts"

            # (b) Files the directory rule calls 'parse' that are LEXICALLY
            # invalid JavaScript - no TypeScript syntax is involved anywhere near
            # the failure, so node is the whole answer and refusing them is
            # correct. Four clusters, each verified file by file:
            #
            #   templateStringUnterminated1-5 (+ _ES6) and TemplateExpression1
            #       an unterminated template literal; node: "Unexpected end of
            #       input" / "Missing } in template expression".
            #   unicodeExtendedEscapes Strings 07,12,17,19,20,21,22,24,25,
            #       Templates 07,12,17,19, RegularExpressions 07,12
            #       \u{110000}, \u{FFFFFFFF}, \u{r}, \u{}, \u{ , \u{67 - node:
            #       "Undefined Unicode code-point" / "Invalid Unicode escape
            #       sequence" / "Invalid regular expression".
            #   the scanner/ numeric and string literal negatives
            #       1e, 1e+, 01.0, 0x, "\u000G", an unterminated string, a raw
            #       NUL, and a keyword spelled with a \u escape. Five of them
            #       carry the suite's own Sputnik '@negative' tag as well. (The
            #       sixth @negative file, parser/ecmascript5/parserS7.6.1.1_A1.10
            #       .ts, has its offending line COMMENTED OUT and node accepts
            #       it, which is why the tag alone is not the rule.)
            #   parserGreaterThanTokenAmbiguity 2,3,4,7,8,9,12,13,14,17,18,19
            #       the whole point of that directory: '>' '>' separated by a
            #       space, a comment or a newline is NOT '>>', so '1 > > 2' and
            #       '1 >> = 2' are syntax errors. node: "Unexpected token '>'" /
            #       "Unexpected token '='". Files 11, 15, 16 and 20 ('1 >>= 2')
            #       are deliberately NOT here: node calls them "Invalid
            #       left-hand side in assignment", which is an EARLY error, and
            #       this tree's convention (see the js corpus above) is that an
            #       early error must still parse.
            #   MemberFunctionDeclaration8_es6.ts - 'if (a) NOT-SIGN * bar;',
            #       U+00AC where an expression belongs; node: "Invalid or
            #       unexpected token".

            # (c) Files the DIRECTORY rule calls 'reject' that are valid modern
            # JavaScript or TypeScript. parser/ErrorRecovery is a directory of
            # PARSER-recovery tests, and several of its cases recover from a
            # grammar-check error rather than a parse error - the input is
            # perfectly well-formed:
            #
            #   AccessibilityAfterStatic 2,3,4,5,11,14   'static public;',
            #       'static public = 1;', 'static public: number;',
            #       'static public', 'static public() {}', 'static public<T>() {}'
            #       - a member NAMED 'public'. node accepts every JS-expressible
            #       one of them; the ...1, ...7 and ...10 files ('static public
            #       intI ...', two names) stay 'reject', and so does ...6, whose
            #       class brace is never closed.
            #   ArrowFunction4 / parserX_ArrowFunction4   'var v = (a, b) => {};'
            #   parserVariableStatement1-4                'var a,\n b,\n c'
            #       node accepts all six outright.
            #   parserErrorRecovery_IncompleteMemberVariable1   'public con:
            #       "hello";' is an ordinary string-literal type. Its sibling
            #       ...2 carries the actual error ('public con:C "hello";') and
            #       stays 'reject'.
            #   parserCommaInTypeMemberList1 / 2   '{ workItem: any, width:
            #       string }'. A comma in a type member list has been legal
            #       TypeScript for a decade; the corpus itself settles it -
            #       types/conditional/conditionalTypes1.ts:167 writes
            #       'T extends { a: string, b: number }' in a file expected to
            #       parse.
            awk -v marked="$neg_ts" '
                BEGIN { while ((getline l < marked) > 0) mark[l] = 1 }
                {
                  bad = ($0 ~ /\/(ErrorRecovery|SkippedTokens)\//) \
                     || ($0 ~ /\/decorators\/invalid\//) \
                     || ($0 in mark)
                  # (b) lexically invalid, node-verified
                  bad = bad \
                     || ($0 ~ /\/es6\/templates\/templateStringUnterminated[1-5](_ES6)?\.ts$/) \
                     || ($0 ~ /\/es6\/templates\/TemplateExpression1\.ts$/) \
                     || ($0 ~ /unicodeExtendedEscapesInStrings(07|12|17|19|20|21|22|24|25)\.ts$/) \
                     || ($0 ~ /unicodeExtendedEscapesInTemplates(07|12|17|19)\.ts$/) \
                     || ($0 ~ /unicodeExtendedEscapesInRegularExpressions(07|12|17|19)\.ts$/) \
                     || ($0 ~ /\/scanner\/ecmascript[35]\/scanner(ES3)?NumericLiteral[346]\.ts$/) \
                     || ($0 ~ /\/scanner\/ecmascript5\/scannerS7\.4_A2_T2\.ts$/) \
                     || ($0 ~ /\/scanner\/ecmascript5\/scannerS7\.8\.3_A6\.1_T1\.ts$/) \
                     || ($0 ~ /\/scanner\/ecmascript5\/scannerS7\.8\.4_A7\.1_T4\.ts$/) \
                     || ($0 ~ /\/scanner\/ecmascript5\/scannerStringLiterals\.ts$/) \
                     || ($0 ~ /\/scanner\/ecmascript5\/scannerUnexpectedNullCharacter1\.ts$/) \
                     || ($0 ~ /\/scanner\/ecmascript5\/scannerUnicodeEscapeInKeyword[12]\.ts$/) \
                     || ($0 ~ /parserGreaterThanTokenAmbiguity(2|3|4|7|8|9|12|13|14|17|18|19)\.ts$/) \
                     || ($0 ~ /\/MemberFunctionDeclaration8_es6\.ts$/)
                  # (c) valid after all - overrides everything above
                  good = ($0 ~ /parserAccessibilityAfterStatic(2|3|4|5|11|14)\.ts$/) \
                      || ($0 ~ /\/ArrowFunctions\/(parserX_)?ArrowFunction4\.ts$/) \
                      || ($0 ~ /\/parserVariableStatement[1-4]\.ts$/) \
                      || ($0 ~ /parserErrorRecovery_IncompleteMemberVariable1\.ts$/) \
                      || ($0 ~ /parserCommaInTypeMemberList[12]\.ts$/)
                  print ((bad && !good) ? "reject|" : "parse|") $0 }' "$all"
            return ;;
        python)
            # CPython's suite has no negative-test convention to read, so the
            # default here was a blanket "must parse". Two of the thirty files
            # are not valid Python at all - they carry deliberate syntax-error
            # cases inline:
            #     test_listcomps.py   [*i for i in [1, 2, 3]]
            #                         iterable unpacking cannot be used in a
            #                         comprehension
            #     test_patma.py       case +0:
            # Confirmed by compiling each file with the python3 on this machine
            # (3.14): it rejects exactly these two and accepts the other 28. So
            # refusing them is correct and demanding that we parse them made
            # 30/30 unreachable by construction.
            #
            # Re-check with python3 before adding to this list - a file that
            # merely fails for US is a grammar gap, not a corpus bug.
            awk '{ bad = ($0 ~ /\/(test_listcomps|test_patma)\.py$/)
                   print (bad ? "reject|" : "parse|") $0 }' "$all"
            return ;;
        bash)
            # The oils corpus has no negative-test marker to read, and the
            # default here was a blanket "must parse". That was wrong for 73 of
            # the 2562 swept files, and it cost us in both directions.
            #
            # bash itself is the exact oracle: `bash -n` is a parse-only pass, so
            # a file it refuses is one WE must refuse too. Measured 2026-07-27
            # with GNU bash 5.3.15 (aarch64-apple-darwin25), the version this
            # corpus targets:
            #
            #   73 files are rejected by `bash -n`. Of those we already refuse
            #   64 - and every one was being counted as a PARSE-FAIL, i.e. as our
            #   bug, when it was correct behaviour.
            #
            #   The other 9 we WRONGLY ACCEPT, and the old expectation could not
            #   see them at all. They are self-describing:
            #     parse-errors/011-is-its-own-word-needs-a-space.sh
            #     parse-errors/020-misplaced.sh
            #     regex/013-malformed-regex.sh
            #     regex/035-parse-error-with-2-words.sh
            #     var-sub-quote/032-syntax-error-for-single-quote-in-double-quote.sh
            #     dbracket/040-is-syntax-error.sh, 041-is-syntax-error.sh
            #     command-sub-ksh/002, 003 (ksh-only $(...) forms)
            #   Now that they carry 'reject' we are scored on refusing them.
            #
            # After the split: parse 2260/2489 = 90.8%, reject 64/73 = 87.7%.
            # The single blended 88.6% this replaces was neither number.
            #
            # Note the sweep reads oils-spec/extracted/ only. The 264 raw sources
            # under oils-spec/spec/ are never swept - which is correct, they are
            # the INPUT extract-oils.py turned into extracted/ and they mix YSH
            # and OSH syntax with '## stdout:' metadata, so they are not bash
            # programs at all. (`bash -n` refuses 138 of them, for that reason.)
            #
            # CAUTION, learned the hard way on two other corpora the same day: a
            # toolchain is only an oracle at the RIGHT VERSION. `luac` 5.5 and
            # `/usr/bin/ruby` 2.6.10 were tried here too and produced 12 bogus
            # findings between them - 2.6 predates every Ruby 3 form the corpus
            # uses ('{a:}', '(..1)', 'def foo(...)', '**nil'), and luac reports a
            # <const> violation, which is semantic, not syntactic. Re-verify the
            # version before extending this approach to another language.
            if command -v bash >/dev/null 2>&1; then
                while IFS= read -r f; do
                    if bash -n "$f" 2>/dev/null; then printf 'parse|%s\n' "$f"
                    else                              printf 'reject|%s\n' "$f"; fi
                done < "$all"
            else
                echo "  [warn] bash: no bash on PATH, falling back to must-parse" >&2
                sed 's/^/parse|/' "$all"
            fi
            return ;;
        go)
            # The first line of every golang/go test/ file is its directive;
            # "// errorcheck" means the compiler must reject the file.
            tr '\n' '\0' < "$all" \
                | xargs -0 awk 'FNR==1 && /^\/\/ *errorcheck/ { print FILENAME }' \
                | sort > "$neg" ;;
        swift)  grep -rl 'expected-error' "$dir" --include="*.$ext" 2>/dev/null | sort > "$neg" ;;
        dart)   grep -rl -e '\[analyzer\]' -e '\[cfe\]' "$dir" --include="*.$ext" 2>/dev/null | sort > "$neg" ;;
        java)
            # OpenJDK's suite has TWO negative conventions, and only one of them
            # was being read.
            #
            # (1) '@compile/fail' in a jtreg header - the file must not COMPILE.
            #     Mostly semantic (type errors, bad references), so it keeps the
            #     informational 'reject?', exactly like go/swift/dart.
            #
            # (2) javac/diags/examples/ is a directory of DIAGNOSTIC examples,
            #     one minimal file per compiler message, keyed by a
            #         // key: compiler.err.<something>
            #     line instead of by @compile/fail. Many of those messages come
            #     out of the SCANNER or the PARSER - the file is 'int i = =3;',
            #     'if (true) }', 'String s = "abc;' - i.e. input javac itself
            #     cannot parse. Under the old rule all 915 of them were 'parse',
            #     so refusing them counted as OUR bug.
            #
            # The key list below is derived from the corpus, not from our misses.
            # Every one of the 915 example files was run through
            #     javac -XDrawDiagnostics -proc:none -XDshould-stop.ifNoError=PARSE
            # which stops the compile immediately after parsing, so anything it
            # still prints is by construction a scanner/parser diagnostic. A key
            # is listed here only when EVERY example file carrying it produced
            # that same key at parse stage - the probe ran over all files,
            # including the ones we already parse happily, so the rule was free
            # to cost us files, and it did: of the 90 files it moves to 'reject'
            # we were already refusing 43 (previously scored as our failures) and
            # we WRONGLY ACCEPT the other 47, which this expectation now
            # correctly counts against us. Both halves are the point.
            #
            # These carry the HARD 'reject', not 'reject?'. The reasoning is the
            # same as for kotlin's PsiErrorElement and test262's fail/: the
            # signal is exact and purely syntactic - it is the reference parser
            # reporting that it could not parse this text - so we are scored on
            # actually refusing it. '@compile/fail' stays 'reject?' because it
            # says nothing about WHICH phase failed.
            #
            # Deliberately EXCLUDED, and why - do not add them back without
            # re-running the probe:
            #   compiler.err.feature.not.supported.in.source[.plural],
            #   compiler.err.preview.feature.disabled.plural,
            #   compiler.err.option.removed.source/target,
            #   compiler.err.bad.file.name,
            #   compiler.err.implicit.class.should.not.have.package.declaration
            #       javac's parser does emit these, but they are SOURCE-LEVEL
            #       GATES, not syntax: Records.java carries '--release 15' so
            #       that a perfectly well-formed record declaration is refused.
            #       That is a statement about javac's -source flag, not about
            #       the language, and we target modern Java - we must parse them.
            #   compiler.err.invalid.permits.clause
            #       a compound diagnostic: 1 of the 7 files carrying it fails in
            #       the parser ('permits' on a non-sealed class), the other 6
            #       fail in Attr (duplicate type, supertype, type variable...).
            #       The err key alone cannot tell them apart, so the whole key
            #       is out.
            #   compiler.err.expected.module
            #       2 files; ExpectedModule.java ('open class X { }') is a parse
            #       error but ModuleInfoWithoutModule/ModuleInfoWithoutModule.java
            #       is an EMPTY compilation unit that javac parses fine - it only
            #       fails when the harness compiles it as a module-info.
            #   ProcessorWrongType/ProcessorWrongType.java
            #       body is 'clas ProcessorWrongType { }' and javac reports
            #       compiler.err.expected4 at parse stage - but its declared key
            #       is the semantic compiler.err.proc.processor.wrong.type. It
            #       is left as 'parse' on purpose: naming individual files we
            #       happen to refuse is how a rule starts flattering the grammar.
            # Cost of those exclusions: 2 files (ExpectedModule, ProcessorWrongType)
            # that we correctly refuse are still scored as must-parse failures.
            #
            # CAUTION, learned the hard way on other corpora (see bash below): a
            # toolchain is only an oracle at the RIGHT VERSION. This list was
            # built with OpenJDK javac 24.0.2. `/usr/bin/ruby` 2.6 and `luac` 5.5
            # each manufactured false findings when used the same way, because
            # they predated (or postdated) the language their corpus targets.
            # Re-run the probe, do not hand-edit, if javac here ever changes.
            cat > "$TMP/java.diagkeys" <<'JAVA_DIAG_KEYS'
compiler.err.annotation.missing.element.value
compiler.err.array.and.receiver
compiler.err.array.dimension.missing
compiler.err.assert.as.identifier
compiler.err.bad.initializer
compiler.err.cannot.create.array.with.diamond
compiler.err.cannot.create.array.with.type.arguments
compiler.err.catch.without.try
compiler.err.class.method.or.field.expected
compiler.err.class.not.allowed
compiler.err.default.label.not.allowed
compiler.err.dot.class.expected
compiler.err.else.without.if
compiler.err.empty.char.lit
compiler.err.enum.as.identifier
compiler.err.enum.cant.be.generic
compiler.err.enum.constant.expected
compiler.err.enum.constant.not.expected
compiler.err.expected
compiler.err.expected.module.or.open
compiler.err.expected.str
compiler.err.expected2
compiler.err.expected3
compiler.err.expected4
compiler.err.extraneous.semicolon
compiler.err.finally.without.try
compiler.err.fp.number.too.large
compiler.err.fp.number.too.small
compiler.err.guard.not.allowed
compiler.err.illegal.array.creation.both.dimension.and.initialization
compiler.err.illegal.char
compiler.err.illegal.digit.in.binary.literal
compiler.err.illegal.digit.in.octal.literal
compiler.err.illegal.dot
compiler.err.illegal.esc.char
compiler.err.illegal.line.end.in.char.lit
compiler.err.illegal.nonascii.digit
compiler.err.illegal.start.of.expr
compiler.err.illegal.start.of.stmt
compiler.err.illegal.start.of.type
compiler.err.illegal.text.block.open
compiler.err.illegal.underscore
compiler.err.illegal.unicode.esc
compiler.err.initializer.not.allowed
compiler.err.instance.initializer.not.allowed.in.records
compiler.err.int.number.too.large
compiler.err.invalid.binary.number
compiler.err.invalid.hex.number
compiler.err.invalid.lambda.parameter.declaration
compiler.err.invalid.meth.decl.ret.type.req
compiler.err.invalid.module.directive
compiler.err.invalid.yield
compiler.err.local.enum
compiler.err.malformed.fp.lit
compiler.err.no.annotations.on.dot.class
compiler.err.not.stmt
compiler.err.orphaned
compiler.err.premature.eof
compiler.err.record.cannot.declare.instance.fields
compiler.err.record.cant.declare.field.modifiers
compiler.err.record.component.and.old.array.syntax
compiler.err.record.patterns.annotations.not.allowed
compiler.err.repeated.modifier
compiler.err.restricted.type.not.allowed
compiler.err.restricted.type.not.allowed.array
compiler.err.restricted.type.not.allowed.compound
compiler.err.restricted.type.not.allowed.here
compiler.err.sealed.or.non.sealed.local.classes.not.allowed
compiler.err.statement.not.expected
compiler.err.switch.case.unexpected.statement
compiler.err.this.as.identifier
compiler.err.try.with.resources.expr.needs.var
compiler.err.try.without.catch.finally.or.resource.decls
compiler.err.unclosed.char.lit
compiler.err.unclosed.comment
compiler.err.unclosed.str.lit
compiler.err.unclosed.text.block
compiler.err.underscore.as.identifier
compiler.err.use.of.underscore.not.allowed
compiler.err.use.of.underscore.not.allowed.non.variable
compiler.err.use.of.underscore.not.allowed.with.brackets
compiler.err.varargs.and.old.array.syntax
compiler.err.varargs.and.receiver
compiler.err.varargs.must.be.last
compiler.err.variable.not.allowed
compiler.err.wrong.receiver
JAVA_DIAG_KEYS
            # A key line may be indented (TextBlock*.java write ' // key: ...'),
            # and must match the WHOLE key - 'compiler.err.expected' must not
            # swallow 'compiler.err.expected.module'.
            sed 's/\./\\./g
                 s@^@^[[:space:]]*//[[:space:]]*key:[[:space:]]*@
                 s@$@[[:space:]]*$@' "$TMP/java.diagkeys" > "$TMP/java.diagre"
            : > "$TMP/java.hard"
            if [ -d "$dir/javac/diags/examples" ]; then
                grep -rlE -f "$TMP/java.diagre" "$dir/javac/diags/examples" \
                     --include="*.$ext" 2>/dev/null | sort > "$TMP/java.hard" || : > "$TMP/java.hard"
            fi
            grep -rl '@compile/fail' "$dir" --include="*.$ext" 2>/dev/null | sort > "$neg"
            awk -v hard="$TMP/java.hard" -v soft="$neg" '
                BEGIN { while ((getline l < hard) > 0) h[l] = 1
                        while ((getline l < soft) > 0) s[l] = 1 }
                { print ($0 in h ? "reject|" : ($0 in s ? "reject?|" : "parse|")) $0 }' "$all"
            return ;;
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
