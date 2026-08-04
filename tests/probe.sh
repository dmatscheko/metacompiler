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
#
#   --keep DIR   keep the four outputs (default: a temp dir, removed)
#   --legs L     comma-separated subset of interp,run,native,oracle
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
# Exit 0 iff every leg that ran agrees.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

[ $# -ge 2 ] || { sed -n '2,40p' "$0"; exit 2; }
LANG_NAME="$1"; PROG="$2"; shift 2
ORACLE=""; KEEP=""; LEGS="interp,run,native,oracle"
while [ $# -gt 0 ]; do
    case "$1" in
        --oracle) ORACLE="$2"; shift ;;
        --keep)   KEEP="$2"; shift ;;
        --legs)   LEGS="$2"; shift ;;
        *) echo "probe.sh: unknown flag $1" >&2; exit 2 ;;
    esac
    shift
done
[ -f "$PROG" ] || { echo "probe.sh: no such program: $PROG" >&2; exit 2; }

want() { case ",$LEGS," in *",$1,"*) return 0 ;; *) return 1 ;; esac; }

[ -x ./mec ] || go build -o mec . || exit 2

if [ -n "$KEEP" ]; then mkdir -p "$KEEP"; work="$KEEP"
else work=$(mktemp -d); trap 'rm -rf "$work"' EXIT; fi

INTERP="languages/$LANG_NAME-interpreter.abnf"
COMPILE="languages/$LANG_NAME-to-llvm-ir.abnf"

# `-q` does NOT get you the program's output on its own. A compiler grammar still
# prints its whole MODULE first (clang-check.sh passes -q and strips the module
# anyway - that is why). Separating them by eye is a trap: do not look for "the
# last line that looks like IR", because var_dump prints a bare `}` and PHP will
# fool you. Track brace depth forward. This is clang-check.sh's module_only()
# inverted - it keeps what that one drops.
strip_module() {
    awk '
        BEGIN { depth = 0; done = 0 }
        {
            if (done) { print; next }
            if (depth > 0) { if ($0 ~ /^}/) depth = 0; next }
            if ($0 ~ /^define /) { depth = 1; if ($0 ~ /}[ \t]*$/) depth = 0; next }
            if ($0 ~ /^@/ || $0 ~ /^declare / || $0 ~ /^[ \t]*$/ ||
                $0 ~ /^source_filename/ || $0 ~ /^target / || $0 ~ /^;/ ||
                $0 ~ /^!/ || $0 ~ /^%[A-Za-z_.$]/ || $0 ~ /^attributes /) { next }
            done = 1; print
        }'
}

# Both our engines sign off with `<lang> interpreter:` / `<lang> compiler:
# program ran to completion`. That is the ENGINE talking, not the program, and no
# other leg has it - dropping it is what makes our legs comparable to a real
# toolchain's.
strip_engine() { grep -v -E '(interpreter|compiler): program ran to completion$'; }

declare -a RAN
report() { printf '  %-8s %6s lines, exit %s\n' "$1" "$(wc -l <"$work/$1.out" 2>/dev/null | tr -d ' ')" "$2"; }

echo "probe: $LANG_NAME $PROG"

if want interp && [ -f "$INTERP" ]; then
    ./mec "$INTERP" "$PROG" -q 2>"$work/interp.err" | strip_engine >"$work/interp.out"; r=${PIPESTATUS[0]}
    report interp "$r"; RAN+=(interp)
fi

if want run && [ -f "$COMPILE" ]; then
    ./mec "$COMPILE" "$PROG" -q 2>"$work/run.err" | strip_module | strip_engine >"$work/run.out"; r=${PIPESTATUS[0]}
    report run "$r"; RAN+=(run)
fi

if want native && [ -f "$COMPILE" ] && grep -q 'exePath' "$COMPILE" 2>/dev/null; then
    if ./mec "$COMPILE" "$PROG" -q -exe "$work/a.out" >/dev/null 2>"$work/native.err"; then
        "$work/a.out" >"$work/native.out" 2>>"$work/native.err"; r=$?
        report native "$r"; RAN+=(native)
    else
        echo "  native   BUILD FAILED:"; sed -n '1,3p' "$work/native.err" | sed 's/^/           /'
    fi
fi

if want oracle && [ -n "$ORACLE" ]; then
    # %s -> the program, %d -> its directory. A toolchain that insists on a
    # particular filename (javac) wants the %d form and a `cd`.
    cmd="${ORACLE//%s/$PROG}"; cmd="${cmd//%d/$(dirname "$PROG")}"
    eval "$cmd" >"$work/oracle.out" 2>"$work/oracle.err"; r=$?
    report oracle "$r"; printf '           [%s]\n' "$cmd"
    RAN+=(oracle)
fi

[ ${#RAN[@]} -ge 2 ] || { echo "probe: fewer than two legs ran - nothing to compare"; exit 2; }

echo
ref="${RAN[0]}"
rc=0
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

echo
if [ "$rc" = 0 ]; then
    echo "all ${#RAN[@]} legs agree (${RAN[*]})"
    if ! want oracle || [ -z "$ORACLE" ]; then
        echo "NOTE: no oracle leg. Agreement between our own engines cannot see the"
        echo "      defect class that has paid out most here - both halves wrong together."
    fi
else
    echo "LEGS DISAGREE - and which pair disagrees tells you where to look:"
    echo "  interp vs run     ... the grammars, or the Go twin"
    echo "  run vs native     ... layer 2 (languages/lib/*-rt.metajs) or the C floor"
    echo "  ours vs oracle    ... a real specification gap, in however many engines agree"
fi
[ -n "$KEEP" ] && echo "outputs kept in $KEEP"
exit $rc
