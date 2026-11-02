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

# ===== SECTION 130: syntax errors of test and [ =====
# test picks its rule by the number of arguments (up to four, and so does a parenthesised
# group of up to four); a longer list is parsed as an expression. An argument left over is
# "too many arguments" (or "syntax error: `-X' unexpected"), a lone word before another is
# "unary operator expected", and the first error, an operand's "integer expected" included,
# is the only one reported; the status is 2. Every expected value is bash 5.3's.
s130() {
  local e
  s130m() { local m=${1#*: line }; printf '%s;' "${m#*: }"; }
  e=$(s130m "$(test 1 -eq 1 x 2>&1; echo "|$?")"; s130m "$(test x -eq 2>&1; echo "|$?")"; s130m "$([ 1 -eq 1 x ] 2>&1; echo "|$?")")
  check tse1 "$e" $'test: too many arguments\n|2;test: x: unary operator expected\n|2;[: too many arguments\n|2;'
  e=$(s130m "$(test 1 -eq 1 -x 2>&1; echo "|$?")"; s130m "$(test '(' x y z 2>&1; echo "|$?")"; s130m "$(test x == y -a 2>&1; echo "|$?")"; s130m "$(test x -eq 1 y 2>&1; echo "|$?")"; s130m "$([ x 2>&1; echo "|$?")")
  check tse2 "$e" $'test: syntax error: `-x\' unexpected\n|2;test: `)\' expected, found y\n|2;test: argument expected\n|2;test: x: integer expected\n|2;[: missing `]\'\n|2;'
  e=$({ test -n -a -z; echo $?; test x -a '(' '' ')'; echo $?; test ! '(' ! '' ')'; echo $?; test -t -a -n x; echo $?; test -n x -a -q; echo $?; test -a / -a '(' -n x ')'; echo $?; } 2>&1)
  check tse3 "$e" $'0\n0\n1\n1\n0\n0'
}

# ===== SECTION 131: declarations of a nameref to itself =====
# `local -n NAME=NAME` warns before any other check, a readonly NAME's refusal included. A
# declaration without -n of a function's own nameref whose chain is circular looks it up
# twice, each a warning, and then gives the value to the nameref itself, which must name a
# variable. Every expected value is bash 5.3's.
s131() {
  local e
  s131s() { local l; while IFS= read -r l; do l=${l#*: line }; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$(declare -gi s131w=1; s131m() { local -n s131w=s131w; declare s131w=9+1; echo "s=$?"; }; s131m 2>&1; echo "w=$s131w")
  check nrc1 "$(s131s "$e")" "local: warning: s131w: circular name reference;warning: s131w: circular name reference;warning: s131w: circular name reference;warning: s131w: circular name reference;declare: \`9+1': invalid variable name for name reference;s=1;w=1;"
  e=$(s131p() { local -n s131v=s131v; declare s131v; declare s131v=s131x; echo "s=$?"; declare -p s131v; }; s131p 2>&1)
  check nrc2 "$(s131s "$e")" 'local: warning: s131v: circular name reference;warning: s131v: circular name reference;warning: s131v: circular name reference;warning: s131v: circular name reference;warning: s131v: circular name reference;warning: s131v: circular name reference;s=0;declare -n s131v="s131x";'
  e=$(declare -gr s131r=1; s131f() { local -n s131r=s131r; echo "s=$?"; }; s131f 2>&1; s131g() { local s131y=s131y; local -n s131y; echo "s=$?"; }; s131g 2>&1)
  check nrc3 "$(s131s "$e")" 'local: warning: s131r: circular name reference;local: s131r: readonly variable;s=1;s=0;'
}

# ===== SECTION 132: test with operators known at run time; test -o and -R =====
# test parses the words an expansion gives, so an operator can come out of a variable or
# out of "$@", and an unquoted word can give no field or several. -o asks for a set -o
# option, -R whether a name is itself a nameref that has a value. Every expected value is
# bash 5.3's.
s132() {
  local e op n x y s132v=1
  op=-eq; e=$(test 1 $op 2; echo -n $?; op=-lt; test 1 $op 2; echo -n $?; op='='; [ a $op a ]; echo -n $?)
  n='!'; e="$e$([ $n a = b ]; echo -n $?; [ "$n" ]; echo -n $?; set -- '(' a ')'; test "$@"; echo -n $?)"
  check trt1 "$e" 100000
  e=$(set -- 1 -eq 2; test "$@"; echo -n $?; set -- 1 -eq 1; [ "$@" ]; echo -n $?; set --; test "$@"; echo -n $?; set -- a = a -a b = c; test "$@"; echo -n $?)
  x=; y='a b'; e="$e$([ $x ]; echo -n $?; [ $y ] 2>/dev/null; echo -n $?; [ -n $x ]; echo -n $?; IFS=:; x='1:-lt:2'; test $x; echo -n $?)"
  check trt2 "$e" 10111200
  s132s() { local l; while IFS= read -r l; do l=${l#*: line }; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$(set -- 1 -eq x; test "$@" 2>&1; echo "|$?"; set -- 1 2 3 4 5; test "$@" 2>&1; echo "|$?"; x=; [ $x = a ] 2>&1; echo "|$?")
  check trt3 "$(s132s "$e")" 'test: x: integer expected;|2;test: too many arguments;|2;[: =: unary operator expected;|2;'
  e=$(test -o braceexpand; echo -n $?; [[ -o hashall ]]; echo -n $?; test -o posix; echo -n $?; [ -o nonsense ]; echo -n $?; set -o pipefail; [[ -o pipefail ]]; echo -n $?; set -e; test -o errexit; echo -n $?)
  check tro1 "$e" 001100
  e=$(declare -n s132r=s132v s132u; s132f() { local s132r=1; [[ -R s132r ]]; echo -n $?; }; test -R s132r; echo -n $?; [[ -R s132u ]]; echo -n $?; test -R s132v; echo -n $?; s132f; s132u=s132v; [[ -R s132u ]]; echo -n $?)
  check tro2 "$e" 01110
}

# ===== SECTION 133: nameref declarations refused or landing on the nameref =====
# -n of an operand written NAME[SUB] is refused before any other check, a readonly NAME's
# included. A declaration with attributes, += or an array literal of a function's own
# circular nameref gives them to the nameref itself. Every expected value is bash 5.3's.
s133() {
  local e
  s133s() { local l; while IFS= read -r l; do l=${l#*: line }; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$(declare -r s133r=1; s133f() { local -n 's133z[1]=s133z'; echo "s=$?"; local -n 's133r[1]=s133r'; echo "s=$?"; declare -n s133a[2] s133b=x; echo "s=$?"; declare -p s133b; }; s133f 2>&1)
  check nrs1 "$(s133s "$e")" 'local: s133z[1]: reference variable cannot be an array;s=1;local: s133r[1]: reference variable cannot be an array;s=1;declare: s133a[2]: reference variable cannot be an array;s=1;declare -n s133b="x";'
  e=$(s133f() { local -n s133w=s133w; declare -i s133w=x; echo -n "$? "; declare -p s133w; }; s133f 2>/dev/null)
  e="$e|$(s133f() { local -n s133w=s133w; declare -r s133w=x; declare -p s133w; local -n s133v=s133v; declare -xu s133v+=y; declare -p s133v; }; s133f 2>/dev/null)"
  check nrs2 "$e" $'1 declare -in s133w="s133w"|declare -nr s133w="x"\ndeclare -nxu s133v="S133VY"'
  e=$(s133f() { local -n s133w=s133w; declare s133w=(x y); echo "s=$?"; declare -p s133w; local -n s133v=s133v; declare -r s133v=9; echo "s=$?"; declare -p s133v; }; s133f 2>&1)
  check nrs3 "$(s133s "$e")" 'local: warning: s133w: circular name reference;warning: s133w: circular name reference;warning: s133w: circular name reference;warning: s133w: circular name reference;warning: s133w: circular name reference;warning: s133w: circular name reference;s=0;declare -an s133w=([0]="x" [1]="y");local: warning: s133v: circular name reference;warning: s133v: circular name reference;warning: s133v: circular name reference;warning: s133v: circular name reference;declare: `9'"'"': invalid variable name for name reference;s=1;declare -n s133v="s133v";'
}
# ===== SECTION 134: readonly and export of NAME[SUB], stores through namerefs, test -nt -ot -ef =====
# export and readonly take no NAME[SUB], with a value or without, and readonly's -n takes
# its attribute away as export's does. declare -g gives -x to the global a local or a
# circular nameref hides. A scalar store to a circular nameref that holds an array changes
# nothing; a store through a readonly nameref lands on its target. -nt, -ot and -ef compare
# files, here ones that exist against ones that do not. Every expected value is bash 5.3's.
s134() {
  local e o p
  s134s() { local l; while IFS= read -r l; do l=${l#*: line }; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$({ readonly s134a[1]=b; echo "s=$?"; readonly -a s134b[1]=c; echo "s=$?"; export -n s134b[1]=c; echo "s=$?"; export s134c=1 's134d[k]=v'; echo "s=$?"; declare -p s134a s134b s134c s134d; } 2>&1)
  check rox1 "$(s134s "$e")" 'readonly: `s134a[1]'"'"': not a valid identifier;s=1;readonly: `s134b[1]'"'"': not a valid identifier;s=1;export: `s134b[1]'"'"': not a valid identifier;s=1;export: `s134d[k]'"'"': not a valid identifier;s=1;declare: s134a: not found;declare: s134b: not found;declare -x s134c="1";declare: s134d: not found;'
  e=$(readonly -n s134e=x; readonly -n s134f; declare -p s134e s134f 2>&1; s134e=y; echo "$s134e")
  check rox2 "$(s134s "$e")" 'declare -- s134e="x";declare: s134f: not found;y;'
  e=$(s134g() { local -n s134w=s134w; declare -g -x s134w=q; local s134v=1; declare -g -x s134v=r; }; s134g 2>/dev/null; declare -p s134w s134v)
  check rox3 "$e" $'declare -x s134w="q"\ndeclare -x s134v="r"'
  e=$(s134g() { local -n s134w=s134w; declare s134w=(x y); s134w=z; s134w+=r; declare -p s134w; }; s134g 2>/dev/null)
  check rox4 "$e" 'declare -an s134w=([0]="x" [1]="y")'
  e=$(declare -nr s134r=s134x; s134r=y; echo -n "$s134x "; s134g() { local -nr s134q=s134y; s134q=z; (( s134q += 1 )); echo -n "$s134y "; local -n s134w=s134w; declare -r s134w=s134z; s134w=v; echo -n "$s134z"; }; s134g 2>/dev/null)
  check rox5 "$e" 'y 1 v'
  e=$(s134g() { local -n s134w=s134w; declare -g -r s134w=1; s134w=2; echo no; }; s134g 2>&1; echo no)
  check rox6 "$(s134s "$e")" 'local: warning: s134w: circular name reference;warning: s134w: circular name reference;warning: s134w: maximum nameref depth (8) exceeded;s134w: readonly variable;'
  e=$(declare -n 's134h[x y]' 2>&1; echo "s=$?"; s134g() { local -n 's134i[a b]'; }; s134g 2>&1)
  check rox7 "$(s134s "$e")" 'declare: s134h[x y]: reference variable cannot be an array;s=1;local: s134i[a b]: reference variable cannot be an array;'
  o=-nt; p=/s134-none
  e=$([ /dev/null -nt "$p" ]; echo -n $?; [ "$p" -nt /dev/null ]; echo -n $?; test "$p" -ot / ; echo -n $?; [[ / -ot $p ]]; echo -n $?; [ / $o "$p" ]; echo -n $?; [ "$p" $o "$p" ]; echo -n $?)
  e="$e $([ /usr -ef /usr ]; echo -n $?; [[ /dev/null -ef /dev/null ]]; echo -n $?; test / -ef /dev; echo -n $?; o=-ef; [ "$p" $o "$p" ]; echo -n $?; [ /usr -nt /usr ]; echo -n $?; [[ / -ot / ]]; echo -n $?)"
  check rox8 "$e" '010101 001111'
}
# ===== SECTION 135: list values for array members, stores to circular array namerefs =====
# NAME[SUB]=(list) is refused and abandons the line, after a declaration has made NAME an
# array. export and readonly of a function's circular local nameref act on the global. A
# store to a circular nameref that holds an array leaves the array as it was; its
# declaration's own list still lands. Every expected value is bash 5.3's, less the byte
# bash prints between the quotes.
s135() {
  local e
  s135s() { local l; while IFS= read -r l; do l=${l#*: line }; l=${l//$'\001'/}; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$( { s135a[1]=(x y); echo no; } 2>&1; echo "s=$?"; declare -p s135a 2>&1)
  check alm1 "$(s135s "$e")" 's135a[1]: cannot assign list to array member;'
  e=$(s135f() { local s135b[2]=(p); echo no; }; s135f 2>&1; echo "s=$?")
  check alm2 "$(s135s "$e")" 's135b[2]: cannot assign list to array member;'
  e=$(declare -A s135c; { declare -A s135c[k]=(v w); echo no; } 2>&1; declare -p s135c)
  check alm3 "$(s135s "$e")" 's135c[k]: cannot assign list to array member;'
  e=$(s135f() { export s135d[1]=(p q); echo no; }; s135f 2>&1; declare -p s135d 2>&1)
  check alm4 "$(s135s "$e")" 's135d[1]: cannot assign list to array member;'
  e=$(s135g() { local -n s135w=s135w; export s135w=q; declare -p s135w; local -n s135v=s135v; readonly s135v; export -n s135v; }; s135g 2>&1; declare -p s135w)
  check alm5 "$(s135s "$e")" 'local: warning: s135w: circular name reference;warning: s135w: circular name reference;warning: s135w: maximum nameref depth (8) exceeded;warning: s135w: circular name reference;declare -n s135w="s135w";local: warning: s135v: circular name reference;warning: s135v: circular name reference;warning: s135v: circular name reference;warning: s135v: maximum nameref depth (8) exceeded;warning: s135v: circular name reference;declare -x s135w="q";'
  e=$(s135g() { local -n s135u=s135u; readonly s135u=(x); }; s135g 2>&1; declare -p s135u)
  check alm6 "$(s135s "$e")" 'local: warning: s135u: circular name reference;warning: s135u: circular name reference;warning: s135u: circular name reference;warning: s135u: circular name reference;warning: s135u: circular name reference;declare -ar s135u=([0]="x");'
  e=$(readonly s135z=1; export s135z=2 2>/dev/null; declare -p s135z)
  check alm7 "$e" 'declare -rx s135z="1"'
  e=$(s135g() { local -n s135t=s135t; declare s135t=(x y); s135t[1]=q; echo no; }; s135g 2>&1; echo "s=$?")
  check alm8 "$(s135s "$e")" 'local: warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;`'"'"': not a valid identifier;'
  e=$(s135g() { local -n s135q=s135q; declare s135q=(x y); (( s135q[1]=7 )); echo "s=$?"; let 's135q[0]=4'; echo "s=$?"; declare s135q[1]=z; echo "s=$?"; unset 's135q[1]' s135q; echo "s=$?"; read -a s135q <<< 'a b'; echo "s=$?"; declare s135q+=(z); declare -p s135q; }; s135g 2>/dev/null)
  check alm9 "$e" $'s=0\ns=0\ns=1\ns=0\ns=1\ndeclare -an s135q=([0]="x" [1]="y" [2]="z")'
}
# ===== END SECTIONS =====

s129  # SECTION-CALL 129
s130  # SECTION-CALL 130
s131  # SECTION-CALL 131
s132  # SECTION-CALL 132
s133  # SECTION-CALL 133
s134  # SECTION-CALL 134
s135  # SECTION-CALL 135
echo "full: $checks checks, $fails failures"
exit $fails
