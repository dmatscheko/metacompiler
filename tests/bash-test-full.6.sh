#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 6
# of 6, sections 129 on; a new section goes here. The split is explained in
# tests/bash-test-full.sh.
#
# This file belongs to the SECOND test group (./test.sh --full): it is NOT part
# of the default matrix. The goal of the metacompiler is to support the full
# languages; this file is the ratchet that measures how far the bash grammars
# are. It walks the whole practical Bash syntax, one self-contained SECTION per
# language area. The --full runner runs the file, and whenever a grammar aborts
# it removes the section around the error and retries - so the report lists
# every unsupported section, not just the first.
#
# Conventions (shared by every *-test-full.* file):
#   - prologue (before the first SECTION marker): the check helper only
#   - each section: '# ===== SECTION <nn>: <name> =====' at top level, holding
#     one sNN function (plus any sNN-prefixed helpers), self-contained, with no
#     references to other sections
#   - the tail calls each section via a line tagged 'SECTION-CALL <nn>' and
#     prints the summary line 'full: <checks> checks, <failures> failures'
#   - the file ends with 'exit $fails' (exit 0 == full support, verified)
#
# Every assertion in this file was settled by running it under the real
# /bin/bash (5.x) on the development machine, not derived from documentation.
#
# Deliberately out of scope (not syntax, or unrunnable in this harness):
# external commands of every kind - so no cat/ls/sed/grep/awk anywhere - which
# in turn rules out pathname expansion against a real directory (glob PATTERNS
# are covered instead, by case and by [[ == ]], which is the same matcher),
# process substitution, job control, coprocesses, 'wait', signal delivery
# beyond the EXIT trap, $RANDOM / $SECONDS / time, terminal-dependent behaviour,
# locale-dependent collation, and 'read' from a terminal. Nothing writes to the
# filesystem: redirection is exercised against /dev/null and through here-docs
# and here-strings, which need no writable file.
#
# Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
# code), organized after the GNU Bash reference manual with the ANTLR
# grammars-v4 shell grammars as a coverage checklist.

fails=0
checks=0

# check ID GOT WANT : one assertion.
check() {
  checks=$(( checks + 1 ))
  if [ "$2" != "$3" ]; then
    echo "FAIL $1"
    fails=$(( fails + 1 ))
  fi
}

# ===== SECTION 129: integer operands of test; circular namerefs keep attributes; type =====
# test and [ take a plain decimal integer for -eq ... -ge (white space around it and one sign
# allowed, no arithmetic, nothing beyond 64 bits): anything else is "integer expected", the
# first one reported, and the status is 2 whatever -a, -o and ! make of the rest; [[ ]]
# evaluates arithmetic instead. A store through a nameref chain that closes on a local lands
# on the global variable under that variable's attributes. `type` of a computed name prints
# a function while another function's body holds a case pattern.
# Every expected value is bash 5.3's.
s129() {
  local e f v
  e=$(for v in x 'x+0' 'a[1]' '' 0x10 '1 2' - 99999999999999999999 -9223372036854775809 ' 12 ' $'\t-12\n' +12 012 -9223372036854775808; do test "$v" -eq 12 2>/dev/null; printf '%s ' $?; done)
  check tie1 "$e" '2 2 2 2 2 2 2 2 2 0 1 0 0 1 '
  e=$({ [ 1 -eq 1 -o x -eq 1 ]; echo $?; [ ! x -eq 1 ]; echo $?; [ 1 -lt 2 -a 3 -gt 2 ]; echo $?; if [ 2 -ge y ]; then echo t; else echo "f $?"; fi; [[ x -eq 0 ]]; echo $?; } 2>/dev/null)
  check tie2 "$e" $'2\n2\n0\nf 2\n0'
  f=$([ 'x+0' -eq 3 ] 2>&1); f=${f#*: line }; e=$(test a -lt b 2>&1); e=${e#*: line }
  check tie3 "${f#*: }|${e#*: }" '[: x+0: integer expected|test: a: integer expected'
  e=$(declare -gi s129l=3; s129f() { local -n s129l=s129l; s129l=7; s129l+=8; }; s129f 2>/dev/null; declare -gu s129u=a; s129h() { local -n s129u=s129u; s129u=bc; s129u+=de; }; s129h 2>/dev/null; echo "$s129l $s129u")
  check snc1 "$e" '15 BCDE'
  e=$(declare -gi s129a=1 s129b=2; s129g() { local -n s129a=s129b s129b=s129a; s129a=3+3; s129b=4+4; }; s129g 2>/dev/null; declare -gi s129z=1; s129k() { local -n s129z=s129z; read s129z <<< "2+3"; printf -v s129z '%s' "4*2"; }; s129k 2>/dev/null; echo "$s129a $s129b $s129z")
  check snc2 "$e" '6 8 8'
  e=$(s129p() { echo "$(case a in a) echo x;; esac)"; }; s129q() { :; }; v=s129q; type $v; type -t s129p; command -V $v)
  check stl1 "$e" $'s129q is a function\ns129q () \n{ \n    :\n}\nfunction\ns129q is a function\ns129q () \n{ \n    :\n}'
}
# ===== END SECTIONS =====

s129  # SECTION-CALL 129
echo "full: $checks checks, $fails failures"
exit $fails
