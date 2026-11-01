#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 5
# of 5, sections 95 on; a new section goes here. SECTION 96's grown table alone takes
# ~20M of the step brake. The split is explained in tests/bash-test-full.sh.
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

# ===== SECTION 95: expanded subscripts, associative order =====
# Only a written @ or * names a whole array: a subscript that expands to one is an element's
# key, and an arithmetic error on an indexed array. An associative array lists its keys in
# bash's hash order, through deletions, appends and subshells (SECTION 96 grows the table).
# Every expected value is bash 5.3's.
s95() {
  local r x y
  declare -A s95m=([a]=1 [b]=2 ["@"]=at ["*"]=st); x='@'; y='*'
  check xsb1 "${s95m[$x]}|${s95m[$y]}|${s95m["@"]}|${s95m[\*]}|${#s95m[$x]}" 'at|st|at|st|2'
  check xsb2 "${s95m[$x]:-d}|${s95m[$x]/a/A}|${s95m[$x]@Q}" "at|At|'at'"
  check xsb3 "${#s95m[@]} ${#s95m[*]}" '4 4'
  s95b=(10 20)
  r=$( (echo "${s95b[$x]}"; echo "st=$?") 2>&1); check xsb4 "$r" "$0: line $LINENO: @: arithmetic syntax error: operand expected (error token is \"@\")"
  r=$( (echo "[${s95b[$y]}]") 2>&1); check xsb5 "$r" "$0: line $LINENO: *: arithmetic syntax error: operand expected (error token is \"*\")"
  declare -A s95o=([one]=1 [two]=2 [three]=3 [four]=4 [five]=5 [six]=6)
  check aso1 "${!s95o[*]}" 'four two three six one five'
  check aso2 "${s95o[*]}" '4 2 3 6 1 5'
  unset 's95o[two]'; s95o[two]=7; s95o+=([seven]=8 [one]=9)
  check aso3 "$(declare -p s95o)" 'declare -A s95o=([four]="4" [seven]="8" [two]="7" [three]="3" [six]="6" [one]="9" [five]="5" )'
  r=$( s95o[eight]=0; echo "${!s95o[*]}" ); check aso4 "$r" 'four seven two three six one five eight'
  g95() { local -A s95l=([p]=1 [q]=2 [r]=3 [k10]=4 [k11]=5); echo "${!s95l[*]}"; }
  check aso5 "$(g95)" 'r q p k11 k10'
}
# ===== SECTION 96: a grown associative table =====
# bash's associative table quadruples its buckets once it holds twice as many elements,
# and lists the elements in the grown table's order from then on. Every expected value
# is bash 5.3's.
s96() {
  local r x
  declare -A s96g
  for ((x = 0; x < 2049; x++)); do s96g[k$x]=$x; done
  r="${!s96g[*]}"; check asg1 "${r:0:35} ${#s96g[@]} ${s96g[k1234]}" 'k1698 k1699 k1696 k1697 k1694 k1695 2049 1234'
  unset 's96g[k1698]'; s96g[k1698]=x
  r="${!s96g[*]}"; check asg2 "${r:0:29}" 'k1698 k1699 k1696 k1697 k1694'
}
# ===== SECTION 97: bare words of an associative literal =====
# An associative literal whose first element is a bare word is key value pairs: a word is
# not field-split, and a later [k]=v is the word "[k]=v". Once the first element is a
# [k]=v, a bare word is "must use subscript", quoted as written (in single quotes under a
# declaration builtin), and the line is abandoned (a subshell, whole). Every expected
# value is bash 5.3's.
s97() {
  local r w='p q r s' l
  declare -A s97a=($w); check alw1 "$(declare -p s97a)" 'declare -A s97a=(["p q r s"]="" )'
  declare -A s97b=($w x); check alw2 "$(declare -p s97b)" 'declare -A s97b=(["p q r s"]="x" )'
  declare -A s97c=(a b [x]=1 c); check alw3 "$(declare -p s97c)" 'declare -A s97c=([a]="b" ["[x]=1"]="c" )'
  r=$( (declare -A s97d=([x]=0 [y]=2 a [z]=3); echo "no") 2>&1; echo "st=$?"); l=$LINENO
  check alw4 "$r" "$0: line $l: s97d: 'a': must use subscript when assigning associative array
st=1"
  r=$( (declare -A s97e; s97e=([x]=0 a 1) ; echo "no"; declare -p s97e) 2>&1); l=$LINENO
  check alw5 "$r" "$0: line $l: s97e: a: must use subscript when assigning associative array"
  r=$( (declare -A s97e; s97e=([x]=0 a 1)
        declare -p s97e) 2>&1); l=$LINENO
  check alw6 "$r" "$0: line $l: s97e: a: must use subscript when assigning associative array"
  r=$( (declare -A s97f=([x]=0 'a b' 1)) 2>&1); l=$LINENO
  check alw7 "$r" "$0: line $l: s97f: ''\\''a b'\\''': must use subscript when assigning associative array"
  declare -A s97g; s97g[x]=1; s97g+=(a); check alw8 "$(declare -p s97g)" 'declare -A s97g=([x]="1" [a]="" )'
}
# ===== SECTION 98: an associative key that expands to nothing =====
# Storing under it is "NAME[SUB]: bad array subscript" quoting the subscript as written,
# which abandons the line (a builtin's store fails instead); reading it is "NAME: bad
# array subscript" and no element, and the line goes on. Every expected value is bash
# 5.3's.
s98() {
  local r l k=''
  declare -A s98m=([a]=1)
  r=$( (s98m[$k]=v; echo "no") 2>&1; echo "st=$?"); l=$LINENO
  check aek1 "$r" "$0: line $l: s98m[\$k]: bad array subscript
st=1"
  r=$( (s98m[$k]+=v; echo "no") 2>&1); check aek2 "$r" "$0: line $LINENO: s98m[\$k]: bad array subscript"
  r=$( (echo "[${s98m[$k]}] st=$?"; echo "on") 2>&1); l=$LINENO
  check aek3 "$r" "$0: line $l: s98m: bad array subscript
[] st=0
on"
  r=$( (x=${s98m[$k]}; echo "st=$? [$x]") 2>&1); check aek4 "$r" "$0: line $LINENO: s98m: bad array subscript
st=0 []"
  r=$( (echo "[${s98m[""]}]") 2>&1); check aek5 "$r" "$0: line $LINENO: s98m: bad array subscript
[]"
  check aek6 "$(declare -p s98m)" 'declare -A s98m=([a]="1" )'
}
# ===== SECTION 99: prefix assignments before a declaration builtin =====
# The binding is an exported scalar: for the builtin only it hides the name's array and
# attributes, and a builtin that keeps it (export, readonly, -x, -r, a local) keeps it
# exported; export -n does not keep it, and +x keeps no export. Every expected value is
# bash 5.3's.
s99() {
  local r
  r=$(declare -u s99u=x; s99u=b declare -p s99u; declare -p s99u); check pfd1 "$r" 'declare -x s99u="b"
declare -u s99u="X"'
  r=$(s99a=(1 2); s99a=5 declare -p s99a; s99a=5 declare -g s99a=7; declare -p s99a); check pfd2 "$r" 'declare -x s99a="5"
declare -a s99a=([0]="7" [1]="2")'
  r=$(declare -A s99m=([k]=v); s99m=5 declare -p s99m; declare -p s99m); check pfd3 "$r" 'declare -x s99m="5"
declare -A s99m=([k]="v" )'
  r=$(s99n=2+3 declare -p s99n); check pfd4 "$r" 'declare -x s99n="2+3"'
  r=$(s99w=1; s99w=5 declare -r s99w; declare -p s99w); check pfd5 "$r" 'declare -rx s99w="5"'
  r=$(s99y=1; s99y=5 export -n s99y; declare -p s99y); check pfd6 "$r" 'declare -- s99y="1"'
  r=$(s99z=1; s99z=5 declare +x s99z; declare -p s99z); check pfd7 "$r" 'declare -- s99z="5"'
  r=$(s99p=(1 2); s99p=5 export s99p; declare -p s99p); check pfd8 "$r" 'declare -ax s99p=([0]="5" [1]="2")'
  g99() { local s99s=1; s99s=6 declare s99s; declare -p s99s; }
  check pfd9 "$(g99)" 'declare -x s99s="6"'
}
# ===== END SECTIONS =====

s95   # SECTION-CALL 95
s96   # SECTION-CALL 96
s97   # SECTION-CALL 97
s98   # SECTION-CALL 98
s99   # SECTION-CALL 99
echo "full: $checks checks, $fails failures"
exit $fails
