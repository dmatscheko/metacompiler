#!/usr/bin/env bash
#
# probe.sh - the DIFFERENTIAL PROBE, as a command.
#
# The technique this automates found nearly every real defect in this project,
# and it was re-invented by hand every time. Take one program and run it FOUR
# ways:
#
#   1. the interpreter half    languages/<lang>-interpreter.abnf
#   2. llvm.Run                languages/<lang>-to-llvm-ir.abnf   (js_* -> the GO TWIN)
#   3. a native -exe binary    ... -exe                           (js_* -> the C floor
#                                                                  + the MetaJS layer 2)
#   4. the real toolchain      --oracle 'python3 %s'              (if one exists here)
#
# 1 vs 2 is what ./test.sh --cross does. 2 vs 3 IS THE ONLY THING IN THIS PROJECT
# THAT CAN SEE LAYER 2 besides clang-check and native-full - the matrix, --full
# and --cross all run llvm.Run and never touch languages/lib/*-rt.metajs. And 4
# is the only thing that can see the defect class that keeps paying out here:
# BOTH HALVES AGREE AND BOTH ARE WRONG, which byte-identity cannot detect by
# construction.
#
# Usage:
#   tests/probe.sh python p.py
#   tests/probe.sh python p.py --oracle 'python3 %s'
#   tests/probe.sh ruby   p.rb --oracle '/usr/bin/ruby %s'    # 2.6.10: floats only
#   tests/probe.sh java   p.java --oracle 'cd %d && javac Main.java && java Main'
#   tests/probe.sh kotlin p.kt                                # no kotlinc here: 3 ways
#   tests/probe.sh bash   p.sh --oracle 'bash %s'
#
#   --keep DIR    keep the four outputs (default: a temp dir, removed)
#   --legs L      comma-separated subset of interp,run,native,oracle
#   --timeout N   seconds per leg (default 300). A leg that overruns is KILLED
#                 and reported as broken - never diffed.
#   --max-output N  MB a leg may write to stdout or stderr (default 1024). A leg
#                 over it is killed by SIGXFSZ and reported as broken.
#   --cf K=V[,K=V...]  compiler flags for our three legs, passed on as `-cf K=V`
#                 (repeatable). Both grammars must declare each flag, which is
#                 what lets a non-default value be checked against the oracle.
#
# WHEN YOU WRITE THE PROGRAM: read every operand out of an ARRAY. A probe over
# literals tests the grammar's constant folder, not the runtime, and this project
# has been fooled by that. Print one line per row with the inputs on it, so a
# diff names the case.
#
# ORACLES ON THIS MACHINE: go, node, python3 3.14.6, java 24.0.2, swiftc 6.1.2,
# lua 5.5.0, bash 5.3, cc/clang 17. ABSENT: dart, kotlinc, any C# toolchain -
# cite the spec and say so. /usr/bin/ruby is 2.6.10 (2018) and rejects Ruby-3
# syntax; it once produced nine false "hidden defects". Float formatting and
# Float#% are unchanged in 3.x, so FLOAT rows are settleable there and little else.
#
# ---------------------------------------------------------------------------
# A LEG THAT DID NOT RUN IS NOT A LEG THAT AGREED.
#
# Green for a comparison that was never made is the worst outcome here ("A DIFF
# OF TWO RUNS IS NOT A MEASUREMENT UNLESS BOTH RAN TO COMPLETION", manual).
#
# So every leg is in exactly one of two states. It either produced output
# that is compared, or it is BROKEN, named with its reason, and the script exits
# non-zero. A leg is broken when it is requested and:
#
#   - its grammar file does not exist;
#   - the compiler grammar declares no c.exePath, so there is no native leg;
#   - the -exe build failed;
#   - the oracle command does not exist on this machine (exit 127);
#   - it was killed by the timeout, or died on a signal;
#   - the IR interpreter hit its step brake, so the output is TRUNCATED;
#   - it produced NO OUTPUT AT ALL (see the -q/-qq note below).
#
# `tests/probe/selftest.sh` VERIFIES ALL OF THAT BY SABOTAGE, positive case
# included. Run it after touching this file; it takes about twenty seconds.
#
# One language is genuinely unprobeable through stdout and says so rather than
# passing: `calculator`'s result IS its exit code, it has no
# `calculator-interpreter.abnf` (the interpreter grammars are
# calculator-global-stack, -local-stacks, -scientific, -latex), and the probe
# reports both facts instead of comparing two empty files.
# ---------------------------------------------------------------------------
#
# Exit 0 iff every requested leg ran AND every leg that ran agrees.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2
. tests/lib/guard.sh

[ $# -ge 2 ] || { sed -n '2,50p' "$0"; exit 2; }
LANG_NAME="$1"; PROG="$2"; shift 2
ORACLE=""; KEEP=""; LEGS="interp,run,native,oracle"; TIMEOUT=300; MAX_OUT=1024; CF=()
while [ $# -gt 0 ]; do
    case "$1" in
        --oracle)  ORACLE="$2"; shift ;;
        --keep)    KEEP="$2"; shift ;;
        --legs)    LEGS="$2"; shift ;;
        --timeout) TIMEOUT="$2"; shift ;;
        --max-output) MAX_OUT="$2"; shift ;;
        --cf)      CF+=(-cf "$2"); shift ;;
        *) echo "probe.sh: unknown flag $1" >&2; exit 2 ;;
    esac
    shift
done
[ -f "$PROG" ] || { echo "probe.sh: no such program: $PROG" >&2; exit 2; }

# A MISSPELLED LEG NAME MUST NOT QUIETLY REMOVE A LEG (`--legs nativ` would
# otherwise run three legs and say nothing).
for l in ${LEGS//,/ }; do
    case "$l" in
        interp|run|native|oracle) ;;
        *) echo "probe.sh: --legs: unknown leg '$l' (want interp,run,native,oracle)" >&2; exit 2 ;;
    esac
done
want() { case ",$LEGS," in *",$1,"*) return 0 ;; *) return 1 ;; esac; }

case "$TIMEOUT" in ''|*[!0-9]*) echo "probe.sh: --timeout wants seconds" >&2; exit 2 ;; esac
case "$MAX_OUT" in ''|*[!0-9]*|0) echo "probe.sh: --max-output wants megabytes" >&2; exit 2 ;; esac

[ -x ./mec ] || go build -o mec . || exit 2

# A LEG MUST NOT OUTLIVE THE PROBE (tests/lib/guard.sh says why): every leg and
# anything it started is killed before $work goes.
work=""
cleanup() {
    guard_reap
    [ -z "$KEEP" ] && [ -n "$work" ] && rm -rf "$work"
}
trap cleanup EXIT

if [ -n "$KEEP" ]; then mkdir -p "$KEEP"; work="$KEEP"
else work=$(mktemp -d); fi

INTERP="languages/$LANG_NAME-interpreter.abnf"
COMPILE="languages/$LANG_NAME-to-llvm-ir.abnf"

# THE TWO HALVES NEED DIFFERENT QUIET FLAGS, and the asymmetry is not arbitrary.
#
# `-q` suppresses the compiler's own stage chatter; `-qq` additionally suppresses
# the grammar's DEFAULT OUTPUT - the text the grammar itself emits.
#
#   interpreter half: the program's output IS the grammar's default output, so
#                     -qq erases everything and -q is the flag you want.
#   compiler half:    the grammar's default output is the LLVM MODULE, and for a
#                     HANDLE-IR language the program's prints come from llvm.Run
#                     on a side channel. So -qq drops the module and keeps
#                     exactly the program - with no parsing at all.
#
# Self-contained-IR languages (bash, batch, c and the toys) emit IR that calls libc
# directly; there is no js_* side channel, llvm.Run collects the program's output and
# the grammar writes it with llvm.WriteOut, which -qq keeps. The module itself is
# still the grammar's default output, so the probe below takes -q and cuts it off.
#
# So the flavour decides the flag:
#
#   handle IR                  -qq
#   self-contained IR          -q, then CUT THE MODULE off the front.
#
# and the flavour is read off the grammar rather than a hard-coded name list:
# a handle-IR grammar links the C floor, `libPath("runtime.ll")`, and a
# self-contained one does not. (c.exePath cannot be used for this: every
# compiler grammar declares it.) The classification is checked, not trusted:
# if it is wrong in either direction the leg comes out empty or full of IR, and
# an empty leg is a loud failure.
#
# Cutting the module is the case the manual warns about: do NOT look for the
# last line that looks like IR, because var_dump prints a bare `}` and PHP will
# fool you. Scan FORWARD tracking brace depth - clang-check.sh's module_only()
# is the reference implementation, and module_lines() below is its complement.
handle_ir() { grep -q 'libPath("runtime.ll")' "$COMPILE" 2>/dev/null; }

# How many leading lines of $1 are the emitted module. The module is a prefix: at
# depth 0 only @globals, declare, define and blanks belong to it, and the first
# line at depth 0 that is none of those starts the program's output.
#
# awk TRUNCATES A STRING AT A NUL (docs chapter 7.10: a probe printing a control
# character reported a phantom defect this exact way), so the count is computed
# on a NUL-free copy and applied to the raw bytes with `tail -n +K`.
#
# THE `declare` ARM REQUIRES AN `@`, and that is not decoration. Every LLVM
# `declare` names a symbol (`declare i64 @js_add(i64, i64)`); a BASH program
# printing `declare -a a=([0]="pa")` does not, and must not be eaten as module.
module_lines() {
    tr -d '\000' <"$1" | awk '
        BEGIN { depth = 0; n = 0 }
        {
            if (depth > 0) { n = NR; if ($0 ~ /^}/) depth = 0; next }
            if ($0 ~ /^define /) { n = NR; depth = 1; if ($0 ~ /}[ \t]*$/) depth = 0; next }
            if ($0 ~ /^@/ || $0 ~ /^declare [^\n]*@/ || $0 ~ /^[ \t]*$/ ||
                $0 ~ /^source_filename/ || $0 ~ /^target / || $0 ~ /^;/ ||
                $0 ~ /^!/ || $0 ~ /^%[A-Za-z_.$]/ || $0 ~ /^attributes /) { n = NR; next }
            exit
        }
        END { print n + 0 }'
}

# Under -q an engine still signs off on stdout. That is the ENGINE talking, not
# the program, and no real toolchain says it - so it has to go before comparing
# against an oracle, or a leg reads one line longer than the others and looks like
# a halves divergence when it is not.
#
# ENUMERATING THE WORDINGS DOES NOT WORK: there are many (`program ran to
# completion`, `program finished`, `program value is`, `main() returned N`,
# `phpmain() returned N`, `wrote executable ...`) and a missed one shows up as a
# fake divergence.
#
# So match the SHAPE: a line beginning `<lang> interpreter:` or `<lang> compiler:`
# at column 0. Every grammar's sign-off has that prefix and nothing else on any
# leg does. A program that deliberately prints such a line is the only false
# positive available, and it would have to start the line with its own language's
# name to collide.
#
# The self-contained grammars have a second shape: bash and batch sign off as
# `bash-to-llvm-ir: exit status 0`, and lisp and tinyc as `lisp: program value
# is 0` / `tinyc: main() returned 0` - the GRAMMAR FILE's own basename, or the
# bare language name, with no `interpreter`/`compiler` word at all. That is
# still not an enumeration of WORDINGS: it is the three names this invocation
# was given (the language and its two grammar basenames), so it stays exact
# whatever the sign-off goes on to say.
GNAMES="$(basename "$COMPILE" .abnf)|$(basename "$INTERP" .abnf)|$LANG_NAME"
#
# `-a` is load-bearing and it is the awk-truncates-at-NUL trap wearing a
# different hat: a probe whose output contains a NUL byte (brainfuck's cell dump
# does) makes grep decide the stream is BINARY, print `Binary file (standard
# input) matches`, and discard every real line - so two legs would compare equal.
strip_engine() {
    # A sign-off is normally its own line. But it is written to the same stdout
    # the program writes, so when the program's LAST output has no trailing
    # newline - `echo -n x`, which is ordinary in bash and batch - the tag is
    # GLUED onto it: `xbash-to-llvm-ir: exit status 0`. No line-anchored filter
    # can see that. Strip the glued form first, then the plain one. The glued
    # form needs program text before the tag: blanking a sign-off that stands on
    # its own line would leave an empty line the grep below cannot drop.
    sed -E "s/^(.+)($GNAMES): (exit status [0-9]+|wrote executable .*)\$/\\1/" \
  | grep -a -v -E "^([a-z0-9#+-]+ (interpreter|compiler)|$GNAMES): "
}

# One more piece of engine chatter, and it only exists between the module and
# the program: brainfuck's compiler grammar announces the handover with
#     --- Executed with the built-in IR interpreter, output:
# There is no `<name>: ` in it, so the shape above cannot reach it, and leaving
# it in makes every brainfuck probe permanently red - which destroys the row's
# own signal (clang-check.sh's held-row note is the same argument).
#
# The narrowest rule that removes it: drop AT MOST ONE line, only if it is the
# FIRST line after the module in the RUN leg, and only if it is a dashed banner
# ending in a colon. A program line would have to be in exactly that position
# and exactly that shape to be eaten. If a second grammar ever grows a banner of
# a different shape, the run leg reads one line long and the probe says so
# loudly - which is the whole point of the rest of this file.
#
# Tested with head/tr/grep rather than awk, because the line after it may well
# be binary: awk truncates a string at a NUL and brainfuck prints NULs.
is_handover_banner() {   # is_handover_banner FILE LINENO
    tail -n "+$2" "$1" | head -1 | tr -d '\000' | grep -qE '^-{2,}[ \t].*:$'
}

declare -a RAN=()
declare -a BROKEN=()

reject() {   # reject NAME REASON
    printf '  %-8s BROKEN: %s\n' "$1" "$2"
    BROKEN+=("$1: $2")
}

# Run one leg under a watchdog. stdout -> NAME.raw, stderr -> NAME.err, and the
# exit status -> NAME.rc. A leg that overruns is killed and flagged, because a
# killed leg that leaves a short file behind is exactly how a truncation gets
# diffed as if it were an answer.
#
# The cleanup trap cannot run when the probe itself is SIGKILLed, so the leg
# also carries its own limits: an output cap (SIGXFSZ) and a CPU-time cap
# (SIGXCPU). The CPU cap is four times the wall-clock timeout, because the Go
# engine legs legitimately use more than one core; it only has to stop an
# orphan, not race the watchdog.
run_leg() {
    local name="$1"; shift
    rm -f "$work/$name.timeout"
    ( ulimit -S -f $((MAX_OUT * 1024)) -t $((TIMEOUT * 4)); exec "$@" ) >"$work/$name.raw" 2>"$work/$name.err" &
    local pid=$!
    { sleep "$TIMEOUT"; if kill -0 "$pid" 2>/dev/null; then : >"$work/$name.timeout"; kill -9 "$pid" 2>/dev/null; fi; } >/dev/null 2>&1 &
    local wd=$!
    wait "$pid" 2>/dev/null; local rc=$?
    kill "$wd" 2>/dev/null; wait "$wd" 2>/dev/null
    printf '%s\n' "$rc" >"$work/$name.rc"
}

# Validate one leg's raw output and, if it is sound, turn it into NAME.out and
# enrol it for comparison. Anything else is BROKEN and never diffed.
accept_leg() {   # accept_leg NAME [module]
    local name="$1" cut="${2:-}"
    local rc; rc="$(cat "$work/$name.rc" 2>/dev/null || echo '?')"

    if [ -f "$work/$name.timeout" ]; then
        reject "$name" "KILLED after ${TIMEOUT}s - a truncated leg is not a measurement"
        return 1
    fi
    # A Swift trap is SIGTRAP by definition (swiftc's binary dies with 133), so
    # all four legs exit 133 on purpose and the status is compared like any other.
    if [ "$rc" = $((128 + $(kill -l XFSZ))) ]; then
        reject "$name" "wrote more than ${MAX_OUT} MB and was killed - raise --max-output"
        return 1
    fi
    if [ "$rc" -ge 128 ] 2>/dev/null && ! { [ "$LANG_NAME" = swift ] && [ "$rc" = 133 ]; }; then
        reject "$name" "died on signal $((rc - 128)) (exit $rc) - a crashing leg is not a measurement"
        return 1
    fi
    if [ "$cut" = module ]; then
        local k; k="$(module_lines "$work/$name.raw")"
        is_handover_banner "$work/$name.raw" "$((k + 1))" && k=$((k + 1))
        tail -n "+$((k + 1))" "$work/$name.raw" | strip_engine >"$work/$name.out"
    else
        strip_engine <"$work/$name.raw" >"$work/$name.out"
    fi

    # The IR interpreter's endless-loop brake stops mid-program and leaves a
    # PARTIAL file. Only the program's output and stderr are searched: the module
    # itself may carry the same words as a string constant (bash's =~ does).
    if grep -q 'IR interpreter: step limit exceeded' "$work/$name.err" "$work/$name.out" 2>/dev/null; then
        reject "$name" "hit the IR step brake - output is TRUNCATED (raise -max-steps)"
        return 1
    fi

    local lines; lines="$(wc -l <"$work/$name.out" | tr -d ' ')"
    if [ "$lines" -eq 0 ]; then
        reject "$name" "produced NO OUTPUT (exit $rc) - nothing to compare, so this is not agreement"
        [ -s "$work/$name.err" ] && sed -n '1,3p' "$work/$name.err" | sed 's/^/           /'
        return 1
    fi

    printf '  %-8s %6s lines, exit %s\n' "$name" "$lines" "$rc"
    RAN+=("$name")
    return 0
}

echo "probe: $LANG_NAME $PROG"

# ---- leg 1: the interpreter half -----------------------------------------
if want interp; then
    if [ ! -f "$INTERP" ]; then
        reject interp "no $INTERP (name the real grammar, or --legs without interp)"
    else
        run_leg interp ./mec "$INTERP" "$PROG" -q ${CF[@]+"${CF[@]}"}
        accept_leg interp
    fi
fi

# ---- leg 2: llvm.Run over the emitted module ------------------------------
if want run; then
    if [ ! -f "$COMPILE" ]; then
        reject run "no $COMPILE"
    elif handle_ir; then
        run_leg run ./mec "$COMPILE" "$PROG" -qq ${CF[@]+"${CF[@]}"}
        accept_leg run
    else
        # Self-contained IR: -qq would erase the module AND the program. Take
        # -q and cut the module off the front instead.
        run_leg run ./mec "$COMPILE" "$PROG" -q ${CF[@]+"${CF[@]}"}
        accept_leg run module
    fi
fi

# ---- leg 3: the native -exe binary ----------------------------------------
if want native; then
    if [ ! -f "$COMPILE" ]; then
        reject native "no $COMPILE"
    elif ! grep -q 'exePath' "$COMPILE" 2>/dev/null; then
        # main.go greps for the same literal and refuses -exe without it, so
        # this is the grammar saying it has no native path - say so, do not
        # quietly drop to three legs.
        reject native "$COMPILE declares no c.exePath - no native leg exists"
    elif ! ./mec "$COMPILE" "$PROG" -q ${CF[@]+"${CF[@]}"} -exe "$work/a.out" >/dev/null 2>"$work/build.err"; then
        reject native "the -exe BUILD FAILED"
        sed -n '1,3p' "$work/build.err" | sed 's/^/           /'
    else
        run_leg native "$work/a.out"
        accept_leg native
    fi
fi

# ---- leg 4: the real toolchain --------------------------------------------
if want oracle && [ -n "$ORACLE" ]; then
    # %s -> the program, %d -> its directory. A toolchain that insists on a
    # particular filename (javac) wants the %d form and a `cd`.
    cmd="${ORACLE//%s/$PROG}"; cmd="${cmd//%d/$(dirname "$PROG")}"
    run_leg oracle bash -c "$cmd"
    orc="$(cat "$work/oracle.rc")"
    if [ "$orc" = 127 ]; then
        # "command not found" is the one oracle failure that reads like a result.
        # dart, kotlinc and dotnet are absent from this machine on purpose.
        reject oracle "the oracle command is NOT INSTALLED here (exit 127): $cmd"
    else
        accept_leg oracle && printf '           [%s]\n' "$cmd"
    fi
fi

# ---- the verdict -----------------------------------------------------------
rc=0
[ ${#BROKEN[@]} -eq 0 ] || rc=1

if [ ${#RAN[@]} -lt 2 ]; then
    echo
    echo "probe: fewer than two legs produced output - THERE WAS NO COMPARISON."
    [ ${#BROKEN[@]} -eq 0 ] || for b in "${BROKEN[@]}"; do echo "  $b"; done
    [ -n "$KEEP" ] && echo "outputs kept in $KEEP"
    exit 2
fi

echo
ref="${RAN[0]}"
for leg in "${RAN[@]:1}"; do
    if diff -q "$work/$ref.out" "$work/$leg.out" >/dev/null 2>&1; then
        printf '  %-8s == %-8s identical\n' "$ref" "$leg"
    else
        n=$(diff "$work/$ref.out" "$work/$leg.out" | grep -c '^[<>]')
        printf '  %-8s != %-8s %s differing lines:\n' "$ref" "$leg" "$n"
        diff "$work/$ref.out" "$work/$leg.out" | head -20 | sed 's/^/           /'
        rc=1
    fi
done

# The exit status is part of the answer, not decoration. `/usr/bin/time` reports
# happily on a process that died, and a leg that printed the right lines and
# then exited 1 has not agreed with one that exited 0.
refrc="$(cat "$work/$ref.rc")"
for leg in "${RAN[@]:1}"; do
    legrc="$(cat "$work/$leg.rc")"
    if [ "$legrc" != "$refrc" ]; then
        printf '  %-8s != %-8s EXIT STATUS %s vs %s\n' "$ref" "$leg" "$refrc" "$legrc"
        rc=1
    fi
done

echo
if [ ${#BROKEN[@]} -ne 0 ]; then
    echo "LEGS DID NOT RUN - this probe is NOT a four-way comparison:"
    for b in "${BROKEN[@]}"; do echo "  $b"; done
    echo "  (${#RAN[@]} legs were compared: ${RAN[*]})"
    echo
fi

if [ "$rc" = 0 ]; then
    echo "all ${#RAN[@]} legs agree (${RAN[*]})"
    if ! want oracle || [ -z "$ORACLE" ]; then
        echo "NOTE: no oracle leg. Agreement between our own engines cannot see the"
        echo "      defect class that has paid out most here - both halves wrong together."
    fi
else
    echo "LEGS DISAGREE OR DID NOT RUN - and which pair disagrees tells you where to look:"
    echo "  interp vs run     ... the grammars, or the Go twin"
    echo "  run vs native     ... layer 2 (languages/lib/*-rt.metajs) or the C floor"
    echo "  ours vs oracle    ... a real specification gap, in however many engines agree"
fi
[ -n "$KEEP" ] && echo "outputs kept in $KEEP"
exit $rc
