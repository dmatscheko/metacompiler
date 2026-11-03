#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 5
# of 9, sections 90 to 123. SECTION 96's grown table alone takes ~20M of the step
# brake. The split is explained in tests/bash-test-full.sh.
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

# ===== SECTION 90: computed operands of declare and read =====
# An operand of a declaration that is quoted or comes out of an expansion is re-read as the
# declaration its fields spell: NAME[SUB]=VALUE stores an element (a refused subscript fails
# the builtin with status 1), the value lands under the variable's attributes, a readonly
# variable refuses it, and an attribute option applies to a quoted operand as to a written
# one. A `read` whose operands come out of an expansion stores the same way; -a leaves its
# array set even when the line is empty, and a store by name into an array is to element 0.
# Every expected value is bash 5.3's.
s90() {
  local r s n a
  a=(1 2 3); s='a[-1]=R'; declare "$s"; check cdo1 "${a[*]}" "1 2 R"
  s='a[1]+=Z'; declare "$s" 's90c=2'; check cdo2 "$?|${a[*]}|$s90c" "0|1 2Z R|2"
  s='a[-8]=Q'; r=$(declare "$s" 2>&1); check cdo3 "$?|$r" "1|$0: line $LINENO: a[-8]: bad array subscript"
  declare -i s90i; s='s90i=2*21'; declare "$s"; check cdo4 "$s90i" 42
  s90g() { local s='s90i=3*3'; local "$s"; echo "$s90i"; }
  check cdo5 "$(s90g)|$s90i" "3*3|42"
  readonly s90ro=1
  s='s90ro=2'; r=$(declare "$s" 2>&1); check cdo6 "$?|$r|$s90ro" "1|$0: line $LINENO: declare: s90ro: readonly variable|1"
  declare -a 's90e[2]=x'; check cdo7 "$(declare -p s90e)" 'declare -a s90e=([2]="x")'
  declare -i 's90n=1+2' s90m; check cdo8 "$s90n" 3
  r=$(declare 'x y' 's90z=1' 2>&1); check cdo9 "$?|$r" "1|$0: line $LINENO: declare: \`x y': not a valid identifier"
  n=s90v; read "$n" <<< 'p q'; check crd1 "$s90v" "p q"
  n='s90a s90b'; read $n <<< ' 1  2  3 '; check crd2 "$s90a|$s90b" "1|2  3"
  declare -i s90k; n=s90k; read "$n" <<< '6*7'; check crd3 "$s90k" 42
  n=s90ro; read s90x "$n" <<< 'q r' 2>/dev/null; check crd4 "$?|$s90x|$s90ro" "1|q|1"
  n='s90el[2]'; read "$n" <<< elem; check crd5 "$(declare -p s90el)" 'declare -a s90el=([2]="elem")'
  n=s90arr; read -a "$n" <<< 'e f'; check crd6 "${s90arr[*]}" "e f"
  s90w=(1 2 3); n=s90w; read "$n" <<< new; check crd7 "${s90w[*]}" "new 2 3"
  printf -v "$n" %s pv; check crd8 "${s90w[*]}" "pv 2 3"
  read -a s90y <<< ''; check crd9 "$(declare -p s90y)" 'declare -a s90y=()'
}
# ===== SECTION 91: a function's own redirections, and loop bodies in braces =====
# Redirections written after a function's body belong to the body: they apply at every call
# and `declare -f` prints them after it. A for, select or for (( )) loop may have its body in
# braces, which `declare -f` prints as do ... done. Every expected value is bash 5.3's.
s91() {
  local r
  s91a() { echo a; } >/dev/null; check fr1 "[$(s91a)]" "[]"
  s91b() { read x; echo "got $x"; } <<< hi; check fr2 "$(s91b; s91b)" "got hi
got hi"
  s91c() if true; then echo b; fi >/dev/null; check fr3 "[$(s91c)]" "[]"
  r=$(declare -f s91b s91c); check fr4 "$r" "s91b () 
{ 
    read x;
    echo \"got \$x\"
} <<< hi
s91c () 
{ 
    if true; then
        echo b;
    fi > /dev/null
}"
  s91d() { for x in 1 2; { echo "$x"; }; }
  s91e() for ((i=0; i<2; i++)) { echo $i; }
  s91f() { select s in p q; { break; }; }
  check fr5 "$(s91d; s91e)" "1
2
0
1"
  r=$(declare -f s91d s91e s91f); check fr6 "$r" "s91d () 
{ 
    for x in 1 2;
    do
        echo \"\$x\";
    done
}
s91e () 
{ 
    for ((i=0; i<2; i++))
    do
        echo \$i;
    done
}
s91f () 
{ 
    select s in p q;
    do
        break;
    done
}"
}
# ===== SECTION 92: read's backslashes, empty subscripts, NAME[SUB] alone, -a with a value =====
# Without -r, read takes a backslash as quoting the next byte and drops it, joins a line ended
# by a backslash with the next, and drops one at the end of the input; an escaped separator
# stays in its field. The last name takes the rest of the record less its trailing
# separators, escaped or not. A name that is no identifier fails read (status 1) and
# printf -v (status 2); read reports the first before it reads. An empty subscript `b[]` is
# a bad array subscript. declare, typeset and local read NAME[SUB] alone as `-a NAME`;
# export and readonly refuse it. Under -a, NAME=VALUE and NAME+=VALUE store into element 0,
# and a local starts from no value, whatever the name held outside. Every expected value is
# bash 5.3's.
s92() {
  local r n x y a b l
  read x <<< 'a\b\\c'; check rbs1 "$x" 'ab\c'
  read -r x <<< 'a\b'; check rbs2 "$x" 'a\b'
  read a b <<< 'p\ q r'; check rbs3 "$a|$b" "p q|r"
  r=$(printf 'ab\\' | { read x; echo "$?|$x"; }); check rbs4 "$r" "1|ab"
  r=$(printf 'a\\\nb\n' | { read x; echo "$x"; }); check rbs5 "$r" "ab"
  read a b <<< 'x y\  '; check rbs6 "$a|$b" "x|y "
  read a b <<< 'x y z\ '; check rbs7 "$a|$b" "x|y z"
  read -rd , x <<< 'p\,q'; check rbs8 "$x" 'p\'
  r=$(printf 'a\nb' | { read -d '' x; echo "$?|$x"; }); check rbs9 "$r" "1|a
b"
  r=$(printf 'abc' | { read x; echo "$?|$x"; }); check rbs10 "$r" "1|abc"
  r=$(printf 'c\nd\n' | while read l; do echo "[$l]"; done); check rbs11 "$r" "[c]
[d]"
  r=$(read 1x <<< q 2>&1); check rnm1 "$?|$r" "1|$0: line $LINENO: read: \`1x': not a valid identifier"
  r=$({ read 1x; read y; echo "$y"; } <<< L 2>/dev/null); check rnm2 "$r" "L"
  r=$(read y 'z z' <<< 'a b' 2>&1; echo "$?|$y"); check rnm3 "$r" "$0: line $LINENO: read: \`z z': not a valid identifier
1|a"
  r=$(printf -v 1x %s q 2>&1); check rnm4 "$?|$r" "2|$0: line $LINENO: printf: \`1x': not a valid identifier"
  r=$(printf -v 'b[' %s q 2>&1); check rnm5 "$?|$r" "2|$0: line $LINENO: printf: \`b[': not a valid identifier"
  b=(1 2)
  r=$(exec 2>&1; b[]=x; echo "$?"); check esb1 "$r" "$0: line $LINENO: b[]: bad array subscript"
  g92() { local b[]=y; }
  r=$(g92 2>&1; echo "$?"); check esb2 "${r#*: line *: }" "b[]: bad array subscript
1"
  r=$(printf -v 'b[]' %s z 2>&1; echo "$?|${b[*]}"); check esb3 "$r" "$0: line $LINENO: printf: \`b[]': not a valid identifier
2|1 2"
  r=$(read 'b[]' <<< w 2>&1; echo "$?|${b[*]}"); check esb4 "$r" "$0: line $LINENO: read: \`b[]': not a valid identifier
1|1 2"
  declare s92d[5]; check dsb1 "$(declare -p s92d)" "declare -a s92d"
  declare 's92e[1]'; check dsb2 "$(declare -p s92e)" "declare -a s92e"
  n='s92f[x y]'; declare "$n"; check dsb3 "$(declare -p s92f)" "declare -a s92f"
  g92() { local s=1 t; local s[2] t[1]; declare -p s t; }
  check dsb4 "$(g92)" "declare -a s=()
declare -a t"
  r=$(readonly 's92g[1]' 2>&1); check dsb5 "$?|$r" "1|$0: line $LINENO: readonly: \`s92g[1]': not a valid identifier"
  r=$(export s92h[1] 2>&1); check dsb6 "$?|$r" "1|$0: line $LINENO: export: \`s92h[1]': not a valid identifier"
  r=$(declare 's92i[]' 2>&1); check dsb7 "$?|$r" "1|$0: line $LINENO: declare: \`s92i[]': not a valid identifier"
  declare -a s92x=5; check dav1 "$(declare -p s92x)" 'declare -a s92x=([0]="5")'
  s92w=(1 2); declare -a s92w=7; check dav2 "$(declare -p s92w)" 'declare -a s92w=([0]="7")'
  s92y=3; declare -a s92y+=4; check dav3 "$(declare -p s92y)" 'declare -a s92y=([0]="4")'
  g92() { local s=1; local -a s; declare -p s; local u; local -a u; declare -p u; }
  check dav4 "$(g92)" "declare -a s=()
declare -a u"
  s92s=3; g92() { local s92s+=4; local -i s92t+=5; declare -p s92s s92t; }
  check dav5 "$(g92)" 'declare -- s92s="4"
declare -i s92t="5"'
  g92() { local s92s+=4; echo "$s92s"; }; check dav6 "$(s92s=5 g92)" 54
  n=s92c; declare -a "$n"; check dac1 "$(declare -p s92c)" "declare -a s92c"
  declare -a "$n=5"; check dac2 "$(declare -p s92c)" 'declare -a s92c=([0]="5")'
  s92v=(1 2); n=s92v; declare -a "$n+=8"; check dac3 "$(declare -p s92v)" 'declare -a s92v=([0]="8")'
  g92() { local s=1; local -a "$1"; declare -p s; }; check dac4 "$(g92 s)" "declare -a s=()"
}
# ===== SECTION 93: prefix exports, read -n/-N, empty and @ subscripts, nameref values, +a =====
# A prefix binding is exported while its command runs, and a local made from it keeps the
# export. read -n stops after COUNT characters, -N reads exactly COUNT and splits nothing.
# An empty subscript is a no-op for unset, a bad substitution in ${ }, a bad array subscript
# in a literal (quoted as declare spells it) and its own diagnostics in arithmetic; a blank
# one is element 0. A nameref value must name a variable. +a / +A cannot take an array of
# that kind apart. read and printf -v store an associative array's @ and * as keys, which
# declare -p quotes. Every expected value is bash 5.3's.
s93() {
  local r x n b
  b=(1 2)
  g93() { local s+=4; declare -p s; }; check pxl1 "$(s=5 g93)" 'declare -x s="54"'
  g93() { local s; unset s; declare -p s; }; check pxl2 "$(s=5 g93)" 'declare -x s'
  g93() { declare -p s93p; }; check pxl3 "$(s93p=5 g93)" 'declare -x s93p="5"'
  read -n 2 x <<< abcd; check rdn1 "$x" "ab"
  read -N 3 x <<< 'a bcd'; check rdn2 "$x" "a b"
  r=$(printf 'a\nbc' | { read -N 3 x; echo "$?[$x]"; }); check rdn3 "$r" "0[a
b]"
  r=$(printf 'xy\nz' | { read -N 5 x; echo "$?[$x]"; }); check rdn4 "$r" "1[xy
z]"
  read -n 0 x <<< abc; check rdn5 "$?[$x]" "0[]"
  r=$(read -n q x <<< abc 2>&1); check rdn6 "$?|$r" "1|$0: line $LINENO: read: q: invalid number"
  r=$(read -n <<< abc 2>&1); check rdn7 "$?|${r%%usage:*}" "2|$0: line $LINENO: read: -n: option requires an argument
read: "
  unset 'b[]'; check esu1 "$?|${b[*]}" "0|1 2"
  r=$( (echo "${b[]}") 2>&1); check esu2 "$?|$r" "1|$0: line $LINENO: \${b[]}: bad substitution"
  r=$(exec 2>&1; b+=([]=v); echo "$?"); check esu3 "$r" "$0: line $LINENO: []=v: bad array subscript"
  r=$(exec 2>&1; declare -A s93n=([]=v); echo "$?"); check esu4 "$r" "$0: line $LINENO: ['']='v': bad array subscript"
  declare -A s93m=([k]=); check esu5 "$(declare -p s93m)" 'declare -A s93m=([k]="" )'
  r=$(let 'b[]=3' 2>&1; echo "$?"); check esa1 "$r" "$0: line $LINENO: let: \`b[]': not a valid identifier
0"
  r=$(let 'x=b[]' 2>&1; echo "$?|$x"); check esa2 "$r" "$0: line $LINENO: b[]: bad array subscript
$0: line $LINENO: b[]: bad array subscript
1|0"
  r=$(for (( b[]=0; 0; )); do :; done 2>&1; echo "$?"); check esa3 "$r" "$0: line $LINENO: ((: \`b[]': not a valid identifier
0"
  check esa4 "$(( b[ ] + 5 ))" 6
  r=$(declare -n s93r='b[]' 2>&1); check nrv1 "$?|$r" "1|$0: line $LINENO: declare: \`b[]': invalid variable name for name reference"
  r=$(declare -n s93r='' 2>&1); check nrv2 "$?|$r" "1|$0: line $LINENO: declare: \`': not a valid identifier"
  n='1x'; r=$(declare -n s93r="$n" 2>&1; declare -p s93r 2>/dev/null); check nrv3 "$?|$r" "1|$0: line $LINENO: declare: \`1x': invalid variable name for name reference"
  declare -a s93a=(1 2); r=$(declare +a s93a 2>&1); check pla1 "$?|$r" "1|$0: line $LINENO: declare: s93a: cannot destroy array variables in this way"
  declare -A s93b=([k]=1); r=$(declare +A s93b 2>&1); check pla2 "$?|$r" "1|$0: line $LINENO: declare: s93b: cannot destroy array variables in this way"
  declare +a s93b; check pla3 "$?" 0
  declare -A s93c; printf -v 's93c[@]' %s q; check aks1 "$(declare -p s93c)" 'declare -A s93c=(["@"]="q" )'
  declare -A s93d; read 's93d[*]' <<< z; check aks2 "$(declare -p s93d)" 'declare -A s93d=(["*"]="z" )'
  declare -A s93e; n='s93e[@]'; printf -v "$n" %s q2; check aks3 "$(declare -p s93e)" 'declare -A s93e=(["@"]="q2" )'
  declare -A s93f; n='s93f[*]'; read "$n" <<< z2; check aks4 "$(declare -p s93f)" 'declare -A s93f=(["*"]="z2" )'
}
# ===== SECTION 94: misplaced assignments, nameref stores, declare prefixes, +a of empty arrays =====
# A name an `=` follows is not evaluated before the misplaced assignment is reported. A store
# landing on a nameref itself must name a variable: an assignment is abandoned, read / printf
# -v / a for loop fail, a declaration fails and (outside a local) takes the nameref away. A
# declaration's value reports its arithmetic errors under the keyword, its subscript not.
# A subshell in a function redeclares the function's locals. +a / +A refuse an empty array
# too. Every expected value is bash 5.3's.
s94() {
  local r x l
  s94b=(1 2 3)
  x='1+'; r=$( (echo $(( 1 && x=4 ))) 2>&1); check aen1 "$r" "$0: line $LINENO: 1 && x=4 : attempted assignment to non-variable (error token is \"=4 \")"
  r=$( (echo $(( 1 && s94b[] = 4 ))) 2>&1); check aen2 "$r" "$0: line $LINENO: 1 && s94b[] = 4 : attempted assignment to non-variable (error token is \"= 4 \")"
  r=$( (declare -n s94r; s94r='s94b[]'; echo "st=$?") 2>&1); check nrs1 "$r" "$0: line $LINENO: \`s94b[]': not a valid identifier"
  r=$( (declare -n s94r; s94r=''; echo "st=$?") 2>&1); check nrs2 "$r" "$0: line $LINENO: \`': not a valid identifier"
  r=$( (declare -n s94r; s94r='s94b[1]'; declare -p s94r) 2>&1); check nrs3 "$r" 'declare -n s94r="s94b[1]"'
  r=$( (declare -n s94r; read s94r <<< '1x'; echo "st=$?"; declare -p s94r) 2>&1); l=$LINENO
  check nrs4 "$r" "$0: line $l: read: \`1x': not a valid identifier
st=1
declare -n s94r"
  r=$( (declare -n s94r; printf -v s94r %s 'b[]'; echo "st=$?") 2>&1); check nrs5 "$r" "$0: line $LINENO: printf: \`b[]': not a valid identifier
st=1"
  r=$( (declare -n s94r; for s94r in s94a '1y' s94c; do :; done; echo "st=$?"; declare -p s94r) 2>&1); l=$LINENO
  check nrs6 "$r" "$0: line $l: \`1y': not a valid identifier
st=1
declare -n s94r=\"s94a\""
  r=$( (declare -n s94r; for s94r in s94a s94c; do :; done; declare -p s94r) 2>&1); check nrs7 "$r" 'declare -n s94r="s94c"'
  g94() { declare -n s94r; declare s94r='1x'; echo "st=$?"; declare -p s94r; }; l=$LINENO
  r=$(g94 2>&1); check nrs8 "$r" "$0: line $l: declare: \`1x': invalid variable name for name reference
st=1
declare -n s94r"
  r=$( (declare -gn s94g; declare -g s94g='1x'; echo "st=$?"; declare -p s94g) 2>&1); l=$LINENO
  check nrsa "$r" "$0: line $l: declare: \`1x': not a valid identifier
st=1
$0: line $l: declare: s94g: not found"
  g94() { declare -n s94r; local s94r='1x'; echo "in $?"; declare -p s94r; }; l=$LINENO
  r=$(g94 2>&1); check nrs9 "$r" "$0: line $l: local: \`1x': invalid variable name for name reference
in 1
declare -n s94r"
  g94() { local s94l=5; ( local s94l; echo "[$s94l]"; declare s94j=2; declare -p s94j ); declare -p s94j 2>/dev/null; }
  check lsub1 "$(g94)" '[5]
declare -- s94j="2"'
  r=$( (declare -i s94i=1x; echo "st=$?") 2>&1); check dpf1 "$r" "$0: line $LINENO: declare: 1x: value too great for base (error token is \"1x\")"
  g94() { local -i s94i=1/0; echo in; }; l=$LINENO
  r=$(g94 2>&1); check dpf2 "$r" "$0: line $l: local: 1/0: division by 0 (error token is \"0\")"
  r=$( (declare s94i[1/0]=3; echo "st=$?") 2>&1); check dpf3 "$r" "$0: line $LINENO: 1/0: division by 0 (error token is \"0\")"
  r=$( (declare -a s94e; declare +a s94e; echo "st=$?"; declare -p s94e) 2>&1); l=$LINENO
  check pla4 "$r" "$0: line $l: declare: s94e: cannot destroy array variables in this way
st=1
declare -a s94e"
  r=$( (s94e=(1); unset 's94e[0]'; declare -g +a s94e; echo "st=$?") 2>&1); l=$LINENO
  check pla5 "$r" "$0: line $l: declare: s94e: cannot destroy array variables in this way
st=1"
  r=$( (declare -A s94m; declare +A s94m; echo "st=$?") 2>&1); l=$LINENO
  check pla6 "$r" "$0: line $l: declare: s94m: cannot destroy array variables in this way
st=1"
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
# ===== SECTION 100: prefix assignments before eval =====
# The string runs under the bindings as a function body does: each is an exported scalar
# hiding the name's array and attributes, what the string assigns to it is undone
# afterwards, `unset` in the string brings back the binding it hid for good, and the
# string's own declarations outlive the bindings. Every expected value is bash 5.3's.
s100() {
  local r
  r=$(s100d=(1 2); s100d=7 eval 's100d[1]=x; declare -p s100d'; declare -p s100d); check pev1 "$r" 'declare -ax s100d=([0]="7" [1]="x")
declare -a s100d=([0]="1" [1]="2")'
  r=$(declare -i s100c=1; s100c=2+3 eval 's100c=4+4; declare -p s100c'; declare -p s100c); check pev2 "$r" 'declare -x s100c="4+4"
declare -i s100c="1"'
  r=$(declare -u s100u=x; s100u=b builtin eval 'declare -p s100u'); check pev3 "$r" 'declare -x s100u="b"'
  r=$(s100d=(1 2); s100d=7 command eval 'echo "${s100d[1]-none} ${#s100d[@]}"'); check pev4 "$r" 'none 1'
  r=$(declare -A s100m=([k]=v); s100m=5 eval 'declare -p s100m'); check pev5 "$r" 'declare -x s100m="5"'
  r=$(s100c=1; s100c=2 eval 'unset s100c; echo "[${s100c-unset}]"; s100c=5'; echo "$s100c"); check pev6 "$r" '[1]
5'
  g100() { local s100c=7; s100c=3 eval 'unset s100c; echo "[${s100c-unset}]"'; echo "[$s100c]"; }
  check pev7 "$(g100)" '[7]
[7]'
  h100() { s100c=3 eval 'declare s100w=8'; echo "[$s100w]"; }
  check pev8 "$(h100)" '[8]'
}
# ===== SECTION 101: options a declaration builtin does not take =====
# export and readonly take -aAfnp, declare, typeset and local -aAfFgiIlnprtux; any other
# letter is "KW: -X: invalid option" and the builtin's usage line, status 2, and nothing is
# declared. Every expected value is bash 5.3's.
s101() {
  local r l
  r=$( (export -i s101a=3; echo "st=$? [${s101a-unset}]") 2>&1); l=$LINENO
  check dbo1 "$r" "$0: line $l: export: -i: invalid option
export: usage: export [-fn] [name[=value] ...] or export -p [-f]
st=2 [unset]"
  r=$( (readonly -x s101b=1; echo "st=$? [${s101b-unset}]") 2>&1); l=$LINENO
  check dbo2 "$r" "$0: line $l: readonly: -x: invalid option
readonly: usage: readonly [-aAf] [name[=value] ...] or readonly -p
st=2 [unset]"
  r=$( (declare -i -Z s101c=1; echo "st=$? [${s101c-unset}]") 2>&1); l=$LINENO
  check dbo3 "$r" "$0: line $l: declare: -Z: invalid option
declare: usage: declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...]
st=2 [unset]"
  g101() { local -Z s101d=1; echo "st=$? [${s101d-unset}]"; }
  r=$(g101 2>&1); l=$LINENO
  check dbo4 "$r" "$0: line $((l - 1)): local: -Z: invalid option
local: usage: local [option] name[=value] ...
st=2 [unset]"
  r=$( (export -n -r s101e=1 2>/dev/null; echo "st=$? [${s101e-unset}]") 2>&1); check dbo5 "$r" 'st=2 [unset]'
  r=$( (typeset -Z s101f; echo "st=$?") 2>&1); l=$LINENO
  check dbo6 "$r" "$0: line $l: typeset: -Z: invalid option
typeset: usage: typeset [-aAfFgiIlnrtux] name[=value] ... or typeset -p [-aAfFilnrtux] [name ...]
st=2"
}
# ===== SECTION 102: the subscript of a NAME[SUB] operand is a word =====
# unset, printf -v, read, test -v, a nameref's target and ${!r} expand SUB before it names
# an element; an associative SUB that expands to nothing is "[SUB]: bad array subscript"
# with SUB as written. Every expected value is bash 5.3's.
s102() {
  local r l
  r=$(declare -A m=([a]=1 [b]=2 ['$k']=3 [q]=4); k=b; unset 'm[$k]'; echo "$? ${m[b]-none} ${#m[@]} ${m['$k']}"); check usx1 "$r" '0 none 3 3'
  r=$( (declare -A m=([a]=1); k=''; unset 'm[$k]'; echo "st=$? ${!m[*]}") 2>&1); l=$LINENO
  check usx2 "$r" "$0: line $l: unset: [\$k]: bad array subscript
st=1 a"
  r=$( (declare -A m=([a]=1 [q]=2); unset 'm[]'; echo "st=$?"; unset 'm["q"]'; echo "st=$? ${!m[*]}") 2>&1); check usx3 "$r" 'st=0
st=0 a'
  r=$(declare -a ia=(0 1 2 3); i=2; unset 'ia[$i]'; unset 'ia[""]'; echo "${!ia[*]}"); check usx4 "$r" '1 3'
  r=$(declare -A m; k=b; printf -v 'm[$k]' '%s' hi; printf -v 'm["z"]' '%s' zz; echo "${m[b]} ${m[z]}"); check usx5 "$r" 'hi zz'
  r=$( (declare -A m; k=''; printf -v 'm[$k]' '%s' hi; echo "st=$?") 2>&1); l=$LINENO
  check usx6 "$r" "$0: line $l: m[\$k]: bad array subscript
st=1"
  r=$(declare -A m; k=w; read 'm[$k]' <<< rr; echo "${m[w]-none} ${m['$k']-none}"); check usx7 "$r" 'rr none'
  r=$(declare -A m=([b]=7); k=b; declare -n nr='m[$k]'; s='m[$k]'; echo "$nr ${!s}"); check usx8 "$r" '7 7'
  r=$(declare -A m=([b]=7); k=b; x=b; [[ -v 'm[$k]' ]]; a=$?; [[ -v m[$x] ]]; echo "$a $?"); check usx9 "$r" '0 0'
  r=$(declare -a ia=(5 6); j=1; [[ -v 'ia[$j]' ]]; a=$?; j=4; [[ -v 'ia[$j]' ]]; echo "$a $?"); check usx10 "$r" '0 1'
  r=$( (declare -A m=([a]=1); k=''; echo "L=${#m[$k]}"; echo after) 2>&1); l=$LINENO
  check usx11 "$r" "$0: line $l: [\$k]: bad array subscript"
}
# ===== SECTION 103: a declaration builtin quotes the subscript it was handed =====
# declare, local and typeset see NAME[SUB]=v after expansion, so a bad subscript is quoted
# expanded ("m[]", "ia[1-20]"); an assignment quotes it as written. Every expected value is
# bash 5.3's.
s103() {
  local r l
  r=$( (declare -A m; k=''; declare m[$k]=v; echo "st=$?"; m[$k]=v; echo no) 2>&1); l=$LINENO
  check dsl1 "$r" "$0: line $l: m[]: bad array subscript
st=1
$0: line $l: m[\$k]: bad array subscript"
  g103() { local -A m; local m["$k"]+=v; echo "st=$?"; }
  r=$(k=''; g103 2>&1); l=$LINENO
  check dsl2 "$r" "$0: line $((l - 1)): m[]: bad array subscript
st=1"
  r=$( (declare -a ia=(1); i=1; typeset ia[$i-20]=x; echo "st=$?") 2>&1); l=$LINENO
  check dsl3 "$r" "$0: line $l: ia[1-20]: bad array subscript
st=1"
}
# ===== SECTION 104: a declaration of a nameref acts on its target =====
# Without -n, declare, typeset, local, export and readonly of a nameref give its target the
# attributes and the value, through a chain too. In a function (as here) the declaration
# makes a local of the target, so `declare -r r` leaves an unset readonly t. Every expected
# value is bash 5.3's.
s104() {
  local r l
  r=$( (declare -n s104r=s104t; s104t=1; declare -r s104r; declare -p s104r s104t; s104t=2; echo no) 2>&1); l=$LINENO
  check dnt1 "$r" "declare -n s104r=\"s104t\"
declare -r s104t
$0: line $l: s104t: readonly variable"
  r=$(declare -n s104a=s104u; s104u=ab; declare -u s104a; typeset -i s104a; declare -p s104a s104u); check dnt2 "$r" 'declare -n s104a="s104u"
declare -iu s104u'
  r=$(declare -n s104a=s104u; s104u=ab; declare -u s104a; s104u=cd; declare -p s104u); check dnt5 "$r" 'declare -u s104u="CD"'
  r=$(declare -n s104a=s104u; declare -a s104a; export s104a; declare -p s104u); check dnt3 "$r" 'declare -ax s104u'
  r=$(declare -n s104c=s104d; declare -n s104d=s104e; s104e=0; readonly s104c=2; declare -p s104c s104d s104e); check dnt4 "$r" 'declare -n s104c="s104d"
declare -n s104d="s104e"
declare -r s104e="2"'
  h104() { local -n s104k=s104t; local -i s104k=1+1; declare -p s104t; }
  r=$(s104t=x; h104; declare -p s104t); check dnt6 "$r" 'declare -i s104t="2"
declare -- s104t="x"'
}
# ===== SECTION 105: $'...' in a subscript; keys declare -p prints $'...' =====
# A subscript written in a word is a word, so $'...' in it is ANSI-C quoting, in a store,
# an associative literal and ${a[SUB]}. A key holding a control character or a byte that is
# no character is printed in the $'...' form, as a value is. Every expected value is bash
# 5.3's.
s105() {
  local r
  r=$(declare -A x; x[$'\xff']=1; k=$'\xff'; echo "${x[$k]-none} ${x[$'\xff']-none}"; declare -p x); check ans1 "$r" "1 1
declare -A x=([\$'\\377']=\"1\" )"
  r=$(declare -A z; z[$'a\tb']=3; z+=([$'c\nd']=4); declare -p z); check ans2 "$r" "declare -A z=([\$'a\\tb']=\"3\" [\$'c\\nd']=\"4\" )"
  r=$(declare -A q=([$'\xfe']=1 [$'q\'r']=2); declare -p q); check ans3 "$r" "declare -A q=([\"q'r\"]=\"2\" [\$'\\376']=\"1\" )"
  r=$(declare -A z; k=$'a\tb'; z[$k]=xy; declare z[$'e\tf']=7; k=$'e\tf'; echo "${#z[$'a\tb']} ${z[$k]}"); check ans4 "$r" '2 7'
  r=$(declare -A w; w[$'\x01']=1; declare -p w); check ans5 "$r" "declare -A w=([\$'\\001']=\"1\" )"
}
# ===== SECTION 106: -a and -A of export and readonly; a computed array literal =====
# export's and readonly's -a / -A leave a bare name what it was (an array literal or a value
# still makes one). A computed operand NAME=(...) under -a is an array literal: its words are
# split at blanks, quotes removed and "$NAME" expanded; in a function (as here) += starts
# from the declaration's new local. Every expected value is bash 5.3's.
s106() {
  local r t
  r=$(export -a s106a; declare -p s106a); check xab1 "$r" 'declare -x s106a'
  r=$(s106f=5; readonly -A s106f; declare -p s106f); check xab2 "$r" 'declare -r s106f="5"'
  r=$(s106g=(1); readonly -A s106g; declare -p s106g); check xab3 "$r" 'declare -ar s106g=([0]="1")'
  r=$(n=s106r; declare -a "$n=(1 2)"; declare -p s106r); check dlit1 "$r" 'declare -a s106r=([0]="1" [1]="2")'
  t='s106e=([3]=c [1]=a b "x y" '"'"'q r'"'"')'
  r=$(declare -a "$t"; declare -p s106e); check dlit2 "$r" 'declare -a s106e=([1]="a" [2]="b" [3]="x y" [4]="q r")'
  r=$(s=s106n; t='s106x=(1 2)'; export -a "$s" "$t"; declare -p s106n s106x); check dlit3 "$r" 'declare -x s106n
declare -ax s106x=([0]="1" [1]="2")'
  r=$(s106p=(9 8); x=V; t='s106p+=(7 "a\"b" "$x" [7]="k$x")'; declare -a "$t"; declare -p s106p); check dlit4 "$r" 'declare -a s106p=([0]="7" [1]="a\"b" [2]="V" [7]="kV")'
  r=$(declare -i s106i; t='s106i=(1+1 [4]=2+4)'; declare -a "$t"; declare -p s106i); check dlit5 "$r" 'declare -ai s106i=([0]="2" [4]="6")'
}
# ===== SECTION 107: export and readonly take no +X =====
# Their options end at the first word not starting with "-": +x is an operand, which is no
# name ("not a valid identifier", status 1), and the rest are still declared. Every expected
# value is bash 5.3's.
s107() {
  local r
  r=$(export +x s107a=1 2>/dev/null; echo "st=$?"; declare -p s107a); check xpl1 "$r" 'st=1
declare -x s107a="1"'
  r=$(export +n s107b=2 2>/dev/null; echo "st=$?"; declare -p s107b); check xpl2 "$r" 'st=1
declare -x s107b="2"'
  r=$(readonly +r s107c=3 2>/dev/null; echo "st=$?"; declare -p s107c); check xpl3 "$r" 'st=1
declare -r s107c="3"'
  r=$(export -p +x >/dev/null 2>&1; echo "st=$?"); check xpl4 "$r" 'st=1'
  r=$(export s107d=4; declare -g +x s107d; echo "st=$?"; declare -p s107d); check xpl5 "$r" 'st=0
declare -- s107d="4"'
}
# ===== SECTION 108: an array's kind belongs to its binding =====
# Associative or indexed is a property of the variable in effect: a local hides a global's
# kind, unset and a subshell end it, a computed `declare -A` names its array when it runs,
# and -A of an indexed array (or -a of an associative one) is refused. Every expected
# value is bash 5.3's.
s108() {
  local r
  r=$(declare -A s108m=([a]=1); f() { local s108m; s108m[x]=1; s108m[y+1]=2; declare -p s108m; }; f; declare -p s108m); check akd1 "$r" 'declare -a s108m=([0]="1" [1]="2")
declare -A s108m=([a]="1" )'
  r=$(declare -A s108g=([a]=1); g() { local s108g[@]=z 2>/dev/null; echo "st=$? ${#s108g[@]}"; s108g[1+1]=q; echo "${!s108g[@]}"; }; g); check akd2 "$r" 'st=1 0
2'
  r=$(s=s108h; declare -A "$s"; s108h[k]=v; s108h[1+1]=w; echo "${s108h[k]} ${!s108h[@]}"); check akd3 "$r" 'v k 1+1'
  r=$(t=s108i; declare -A "$t=([p]=1 [q]=2)"; s108i[r+1]=3; declare -p s108i); check akd4 "$r" 'declare -A s108i=([q]="2" [p]="1" [r+1]="3" )'
  r=$(declare -A s108u=([p]=1); unset s108u; s108u[p+1]=2; declare -p s108u); check akd5 "$r" 'declare -a s108u=([1]="2")'
  r=$( (declare -A s108s; s108s[a+b]=1); s108s[a+b]=1; declare -p s108s); check akd6 "$r" 'declare -a s108s=([0]="1")'
  r=$(s108c=(1 2); declare -gA s108c 2>/dev/null; echo "st=$?"; declare -p s108c); check akd7 "$r" 'st=1
declare -a s108c=([0]="1" [1]="2")'
  r=$(declare -gA s108d=([x]=1); declare -ga s108d 2>/dev/null; echo "st=$?"; declare -p s108d); check akd8 "$r" 'st=1
declare -A s108d=([x]="1" )'
  r=$(s108e=6; declare -gA s108e; s108e[k]=1; declare -p s108e); check akd9 "$r" 'declare -A s108e=([0]="6" [k]="1" )'
  r=$(declare -A s108f; f() { local s108f; s108f[1]=x; echo "${s108f@a}"; }; f; echo "${s108f@a}"); check akd10 "$r" 'a
A'
  r=$(declare -A s108k=([a]=1); f() { local s108k=(x y z); echo "${s108k[@]:1}"; s108k=([1+1]=w); echo "${!s108k[@]}"; }; f; echo "${!s108k[@]}"); check akd11 "$r" 'y z
2
a'
  r=$(s=s108l; t='s108l=(one two three)'; declare -A "$s" "$t"; declare -p s108l); check akd12 "$r" 'declare -A s108l=([three]="" [one]="two" )'
  r=$(declare -A s108o; f() { s108o[a+b]=1; }; f; g() { local -A s108p; s108p[c+d]=2; declare -p s108p; }; g; declare -p s108o); check akd13 "$r" 'declare -A s108p=([c+d]="2" )
declare -A s108o=([a+b]="1" )'
  r=$(n=s108n; declare -a "$n=(2*3 a?b)"; declare -p s108n); check dlit6 "$r" 'declare -a s108n=([0]="2*3" [1]="a?b")'
}
# ===== SECTION 109: a quoted part of an indexed subscript stays quoted =====
# In NAME[SUB]=v the SUB of an indexed array is arithmetic text that keeps a quoted part's
# quotes ($'..' re-quoted, \x its backslash), so bash 5.3 refuses it as an arithmetic
# syntax error; a declaration's operand, an array literal and an associative key read SUB
# as a word. Every expected value is bash 5.3's.
s109() {
  local r
  r=$(s109a=(a b c); (s109a[$'1']=5) 2>/dev/null; echo "st=$?"; (s109a['1']=5) 2>/dev/null; echo "st=$?"; (s109a[\1]=5) 2>/dev/null; echo "st=$?"; k=1; (s109a[$k'']=5) 2>/dev/null; echo "st=$?"; (s109a[1+$'2']=5) 2>/dev/null; echo "st=$?"; s109a["1"]=6; echo "${s109a[*]}"); check aqs1 "$r" 'st=1
st=1
st=1
st=1
st=1
a 6 c'
  r=$(s109b=(a b c); declare -g s109b[$'1']=D; s109b+=([$'2']=E); declare -A s109m; s109m[$'k k']=1; s109m['q']=2; echo "${s109b[*]} ${s109m[k k]} ${s109m[q]}"); check aqs2 "$r" 'a D E 1 2'
  r=$(f() { local s109c[\0]=F; echo "${s109c[*]}"; }; f; (s109u[$'0']=1) 2>/dev/null; echo "st=$?"); check aqs3 "$r" 'F
st=1'
}
# ===== SECTION 110: a quoted part of an indexed subscript stays quoted on a read =====
# Reading NAME[SUB] of an indexed array evaluates SUB as its own arithmetic expression,
# with a quoted part's quotes kept, so ${a['1']} and $(( a['1'] )) are arithmetic syntax
# errors; an error in a subscript abandons the line even under (( )), let and [[ ]]. In a
# function an array literal for an array of the other kind is reported twice and fails.
# Every expected value is bash 5.3's.
s110h() { case $1 in *"$2"*) echo 1 ;; *) echo 0 ;; esac; }
s110() {
  local r e
  r=$(s110x=; echo A $s110x B; echo A $s110x; echo $s110x B); check ecf1 "$r" 'A B
A
B'
  r=$(s110a=(a b c); (echo "${s110a['1']}"; echo no) 2>/dev/null; echo "st=$?"; (echo ${s110a[$'2']}) 2>/dev/null; echo "st=$?"; (echo $(( s110a['1'] ))) 2>/dev/null; echo "st=$?"; (echo ${#s110a[\1]}) 2>/dev/null; echo "st=$?"; echo "${s110a["1"]} ${s110a[$'2']}"); check qrd1 "$r" 'st=1
st=1
st=1
st=1
b c'
  r=$(s110a=(a b c); f() { (( s110a[1+] )); echo no; }; (f; echo no) 2>/dev/null; echo "st=$?"; (let "y=s110a[1+]" || echo no; echo no) 2>/dev/null; echo "st=$?"; ([[ s110a[1+] -eq 0 ]]; echo no) 2>/dev/null; echo "st=$?"; (( 0 && s110a[1+] )); echo "st=$?"; echo "$(( 0 ? s110a[1+] : 2 ))"); check qrd2 "$r" 'st=1
st=1
st=1
st=1
2'
  r=$(e=$( (( 1 ? : 2 )) 2>&1); s110h "$e" ": expression expected"; e=$( (( 1 2 )) 2>&1); s110h "$e" "syntax error in expression"; e=$( (( 1 @ )) 2>&1); s110h "$e" "invalid arithmetic operator"); check qrd3 "$r" '1
1
1'
  r=$(s110n=(0 1 2 3); s110j=(2 1); echo "${s110n[s110j[0]]} ${s110n[ s110n[s110j[1]] ]} $(( s110n[s110j[0]] + 1 ))"; s110n[s110j[1]]=x; unset 's110n[s110j[0]]'; echo "${s110n[*]}"); check qrd4 "$r" '2 1 3
0 x 3'
  r=$(s110a=(a b c); (unset "s110a['1']"; echo no) 2>/dev/null; echo "st=$?"; (test -v "s110a['1']"; echo no) 2>/dev/null; echo "st=$?"; [[ -v "s110a['1']" ]]; echo "st=$?"; e=$( (read 's110a[$'"'"'0'"'"']' <<< v; echo no) 2>&1); s110h "$e" "error token is \"\$'0'\""); check qrd5 "$r" 'st=1
st=1
st=0
1'
  r=$(s110a=(a b c); exec 2>/dev/null; (unset "s110a['1']"); echo "st=$?"; (test -v "s110a[\1]"); echo "st=$?"); check qrd7 "$r" 'st=1
st=1'
  r=$(e=$( (s110b=(1); s110b[1'']=x) 2>&1); s110h "$e" "invalid arithmetic operator (error token is \"''\")"); check qrd6 "$r" '1'
  r=$(s110c=(1); f() { local s110d=(1) s110v=0; declare -A s110d=([y]=2) s110v=9; echo "st=$? $s110v"; declare -p s110d; }; f 2>/dev/null; g() { declare -gA s110c=([x]=1) s110w=5; echo "st=$? $s110w"; }; e=$(g 2>&1); s110h "$e" "g: s110c: cannot convert indexed to associative array"; s110h "$e" "declare: s110c: cannot convert"; echo "${e##*$'\n'}"); check acv1 "$r" 'st=1 9
declare -a s110d=([0]="1")
1
1
st=1 5'
}
# ===== SECTION 111: -v of a whole array; a nameref operand of -v =====
# [[ -v NAME[@] ]] (and [*]) asks whether NAME has any element unless NAME is associative,
# where @ and * are plain keys; a plain NAME of an array asks for element 0; the operand is
# read through its nameref chain, and a chain ending at an element is no variable. Every
# expected value is bash 5.3's.
s111() {
  local r
  r=$(s111a=(a b c); s111s=([3]=x); s111e=(); declare -a s111d; s111c=''; declare -A s111m=([k]=1) s111n=([@]=1)
    for n in s111a s111s s111e s111d s111c s111m s111n s111u; do [[ -v "$n[@]" ]]; echo -n "$? "; [[ -v $n[*] ]]; echo -n "$? "; test -v "$n[@]"; echo -n "$?|"; done
    [[ -v s111a[@] ]]; echo "$?"); check vat1 "$r" '0 0 0|0 0 0|1 1 1|1 1 1|0 0 0|1 1 1|0 1 0|1 1 1|0'
  r=$(s111s=([3]=x); s111t=([0]=z); declare -A s111m=([k]=1) s111z=([0]=1); s111x=5; s111x[3]=y
    for n in s111s s111t s111m s111z s111x; do [[ -v $n ]]; echo -n "$? "; done; unset 's111x[0]'; [[ -v s111x ]]; echo -n "$? "; test -v s111s; echo "$?"); check vat2 "$r" '1 0 1 0 0 1 1'
  r=$(s111s=([3]=x); s111t=([0]=z); s111a=(p q); declare -n s111r=s111s s111q=s111t s111er='s111a[1]'
    for n in s111r s111q 's111r[3]' 's111r[@]' s111er; do [[ -v $n ]]; echo -n "$? "; done
    f() { local -a la=([4]=z); local -n lr=la; [[ -v lr ]]; echo -n "$? "; [[ -v lr[4] ]]; echo -n "$? "; [[ -v "lr[*]" ]]; echo "$?"; }; f); check vat3 "$r" '1 0 0 0 1 1 0 0'
}
# ===== SECTION 112: an option word after an operand =====
# After an operand, -A is no option to the builtin ("not a valid identifier", status 1), but
# bash still marks every array literal after an unquoted word starting with - for the kind
# its letters name (A associative, else a indexed) and g for the global, as the words are
# expanded. Every expected value is bash 5.3's.
s112h() { case $1 in *"$2"*) echo 1 ;; *) echo 0 ;; esac; }
s112() {
  local r e
  r=$(declare s112w=7 -A s112e=([z]=3) s112s=5 2>/dev/null; echo "st=$? $s112w"; declare -p s112e s112s; declare s112f=([z]=3) "-A" -i s112g=([y]=1+1) 2>/dev/null; declare -p s112f s112g); check dla1 "$r" 'st=1 7
declare -A s112e=([z]="3" )
declare -- s112s="5"
declare -a s112f=([0]="3")
declare -a s112g=([0]="1+1")'
  r=$(f() { declare s112a=1 -g s112b=([a]=1) 2>/dev/null; declare -p s112b; }; f; declare -p s112b; declare -A s112m=([k]=v); declare -a s112x=(1) -A s112m=([j]=2) 2>/dev/null; echo "st=$?"; declare -p s112m); check dla2 "$r" 'declare -- s112b
declare -a s112b=([0]="1")
st=1
declare -A s112m=([j]="2" )'
  r=$(g() { local s112l=(1); declare x -A s112l=([k]=v); declare -p s112l; }; e=$(g 2>&1); s112h "$e" "s112l: cannot convert indexed to associative array"; s112h "$e" "declare: s112l: cannot"; s112h "$e" "declare -a s112l=([0]=\"1\")"; s112i=(1 2); e=$( (declare x -A s112i=([k]=v); echo no) 2>&1); s112h "$e" "s112i: cannot convert"; s112h "$e" "no"); check dla3 "$r" '1
0
1
0
1'
}
# ===== SECTION 113: the subscript of a computed declaration operand =====
# A NAME[SUB]=v operand that came out of an expansion has its SUB read as an assignment
# statement reads it - an indexed array's as arithmetic text, quotes and all, an associative
# one's as a word - where a written operand's SUB is expanded first. A declaration makes NAME
# an indexed array before it reads SUB, so a refused SUB leaves one: set and empty, or here,
# a function's local, unset. Every expected value is bash 5.3's.
s113() {
  local r
  r=$(s113a=(a b); (declare 's113a[\1]=w'; echo no) 2>/dev/null; echo "st=$?"; (declare "s113a['1']=w"; echo no) 2>/dev/null; echo "st=$?"; declare 's113a["1"]=x' s113a[\0]=y; echo "st=$?"; v='s113a[\1]=z'; (declare "$v"; echo no) 2>/dev/null; echo "st=$?"; declare -p s113a); check dsq1 "$r" 'st=1
st=1
st=0
st=1
declare -a s113a=([0]="y" [1]="x")'
  r=$(declare -A s113m; declare "s113m['k']=1" 's113m[\x]=2'; v='s113m["y"]=3'; declare "$v"; declare -p s113m); check dsq2 "$r" 'declare -A s113m=([y]="3" [x]="2" [k]="1" )'
  r=$(declare 's113b[-5]=w' 2>/dev/null; echo "st=$?"; declare -p s113b; s113f=5; declare 's113f[-5]=w' 2>/dev/null; declare -p s113f; declare -i s113c[-5]=w 2>/dev/null; declare -p s113c; declare -A s113n; declare 's113n[]=w' 2>/dev/null; declare -p s113n); check dsq3 "$r" 'st=1
declare -a s113b
declare -a s113f
declare -ai s113c
declare -A s113n'
}
# ===== SECTION 114: a quoted ${v:off:len} =====
# The offset and the length are arithmetic text, read as $(( )) reads one: a double quote is
# removed, a single quote and a backslash stay and are a syntax error. Every expected value
# is bash 5.3's.
s114h() { case $1 in *"$2"*) echo 1 ;; *) echo 0 ;; esac; }
s114() {
  local r e
  r=$(s114a=(a b c); s114s=hello; (echo "${s114a[@]:'1'}"; echo no) 2>/dev/null; echo "st=$?"; (echo ${s114s:1:'2'}; echo no) 2>/dev/null; echo "st=$?"; echo "${s114s:"1"} ${s114a[@]:"1":1} ${s114s: 1 :"2" }"); check aoq1 "$r" 'st=1
st=1
ello b el'
  r=$(s114s=hello; e=$( (echo "${s114s:1'+'1}") 2>&1); s114h "$e" "s114s: 1'+'1: arithmetic syntax error: invalid arithmetic operator (error token is \"'+'1\")"; e=$( (echo ${s114s:\1}) 2>&1); s114h "$e" "s114s: \\1: arithmetic syntax error: operand expected"); check aoq2 "$r" '1
1'
}
# ===== SECTION 115: computed option words of declare =====
# An option that came out of an expansion is an option still: -A, -a, -g, -x act at run time,
# also after an operand or a computed "--". It does not mark an array literal, which is
# assigned indexed first and then refused, while a written -A marks one even after "--".
# -a with -A is an invalid option, status 2. A nameref passes -A on to its target. Every
# expected value is bash 5.3's.
s115() {
  local r
  r=$(s115o=-A; declare $s115o s115x; s115x[k]=v; echo "1: ${s115x[k]} ${!s115x[@]}"; declare $s115o s115y=([p]=1 [q]=2) 2>/dev/null; echo "rc=$?"; declare -p s115y; s115w=-a; declare $s115w s115v=5; s115v[3]=7; declare -p s115v; declare s115q1=1 $s115o s115q2 2>/dev/null; echo "rc=$?"; declare -p s115q2); check dco1 "$r" '1: v k
rc=1
declare -a s115y=([0]="2")
declare -a s115v=([0]="5" [3]="7")
rc=1
declare -- s115q2'
  r=$(s115o=-A; s115f() { local og=-Ag; declare $og s115g; s115g[x]=1; declare $s115o s115l; s115l[m]=n; declare -p s115l; }; s115f; declare -p s115g); check dco2 "$r" 'declare -A s115l=([m]="n" )
declare -A s115g=([x]="1" )'
  r=$(declare -- -A s115e=(a b) 2>/dev/null; echo "rc=$?"; declare -p s115e; (declare -a -A s115t) 2>/dev/null; echo "rc=$?"; (declare -aA s115t) 2>/dev/null; echo "rc=$?"); check dco3 "$r" 'rc=1
declare -A s115e=([a]="b" )
rc=2
rc=2'
  r=$(declare -n s115r=s115tg; declare -A s115r=([x]=1 [y]=2); declare -p s115tg); check dco4 "$r" 'declare -A s115tg=([y]="2" [x]="1" )'
}
# ===== SECTION 116: local -p =====
# local with no name lists the call's own locals, an unset one as `declare -- NAME`; a name
# that is no local of the call is "not found", status 1. Every expected value is bash 5.3's.
s116h() { case $1 in *"$2"*) echo 1 ;; *) echo 0 ;; esac; }
s116() {
  local r
  r=$(s116g=1; s116f() { local s116b s116a=1; local -p; local -p s116a; local -p s116g 2>/dev/null; echo "rc=$?"; }; s116f); check lcp1 "$r" 'declare -- s116a="1"
declare -- s116b
declare -- s116a="1"
rc=1'
  r=$(s116f() { local e; e=$(local -p s116zz 2>&1); s116h "$e" "local: s116zz: not found"; e=$(typeset -p s116zz 2>&1); s116h "$e" "typeset: s116zz: not found"; }; s116f); check lcp2 "$r" '1
1'
}
# ===== SECTION 117: quoting inside arithmetic text =====
# In a ${v:off:len} in double quotes, $'...' and $"..." are their text. An arithmetic text
# holding a "$" or a "`" is expanded first, and a subscript's single quotes are removed then,
# their text unexpanded; without one they stay and are a syntax error. In double quotes, a
# ${NAME[...} whose "[" is never closed is reported with its whole word. Every expected
# value is bash 5.3's.
s117h() { case $1 in *"$2"*) echo 1 ;; *) echo 0 ;; esac; }
s117() {
  local r
  r=$(s117s=hello; echo "${s117s:$'1'} ${s117s:$"2"} ${s117s:1:$'3'}"; echo $"a b"c); check xq1 "$r" 'ello llo ell
a bc'
  r=$(s117iv=(10 20 30); x=1; echo $(( s117iv['1'] + $x )) $(( s117iv['1'"$x"] )) $[ s117iv['2'] + ${x} ]; for (( s117k=s117iv['1']+$x; s117k<0; s117k++ )); do :; done; echo "k=$s117k"); check xq2 "$r" '21 0 31
k=21'
  r=$(s117iv=(10 20 30); s117ia=(a b c); x=1; e=$( (echo $(( s117iv['$x'] ))) 2>&1); s117h "$e" '$x: arithmetic syntax error: operand expected (error token is "$x")'; e=$( (echo $(( s117iv['1'] ))) 2>&1); s117h "$e" "'1': arithmetic syntax error: operand expected (error token is \"'1'\")"; e=$( (echo "${s117ia['$x']}") 2>&1); s117h "$e" "'1': arithmetic syntax error: operand expected (error token is \"'1'\")"); check xq3 "$r" '1
1
1'
  r=$(declare -A s117m; e=$( (echo x"${s117m[[]-v}"y) 2>&1); s117h "$e" 'bad substitution: no closing `}'"' in x\"\${s117m[[]-v}\"y"; e=$( (echo ${s117m[[]-v}) 2>&1); s117h "$e" '${s117m[[]-v}: bad substitution'); check xq4 "$r" '1
1'
}
# ===== SECTION 118: arrays through a nameref =====
# NAME[SUB], NAME=(...), ${NAME[...]}, ${!NAME[@]}, ${#NAME[@]}, an arithmetic NAME[SUB], read
# and printf -v into NAME[SUB] and unset 'NAME[SUB]' act on the array a nameref names. A
# declaration refusing to convert or destroy that array, or to give a readonly one a value,
# names the nameref; a store or an unset refused by a readonly target names the target and
# abandons the line, or fails the builtin. Every expected value is bash 5.3's.
s118h() { case $1 in *"$2"*) echo 1 ;; *) echo 0 ;; esac; }
s118() {
  local r
  r=$(declare -A s118t; declare -n s118r=s118t; s118r[q]=2; s118r[z]=3; k=("${!s118r[@]}"); echo "${s118r[q]} ${s118t[z]} ${#k[@]} ${#s118r[@]}"; s118a=(1 2 3); declare -n s118s=s118a; s118s[5]=9; s118s+=(7); echo "${s118s[@]} ${#s118s[@]} ${!s118s[@]} ${s118s[1]} ${s118s[@]:1:2}"; unset 's118r[q]'; echo "${!s118t[@]}"; s118r=([k]=v); echo "${!s118t[@]}=${s118t[k]}"); check nra1 "$r" '2 3 2 2
1 2 3 9 7 5 0 1 2 5 6 2 2 3
z
k=v'
  r=$(s118f() { local -n p=$1; p=(x y); p[4]=t; echo "${!p[@]} ${p[-1]}"; }; s118f s118g; echo "${s118g[@]}"; declare -iu s118u=(); declare -n s118v=s118u; s118v[1]=3+4; s118v+=(2*3); echo "${s118u[@]}"; b=5; s118c=(b); declare -n s118w=s118c; echo "${!s118w[0]:-x} ${!s118w[1]-y}"); check nra2 "$r" '0 1 4 t
x y t
7 6
5 y'
  r=$(s118a=(10 20 30); declare -n s118r=s118a; echo $(( s118r[1] + 1 )); (( s118r[2] += 5 )); let 's118r[0]=s118r[0]*2'; z='s118r[0]+1'; echo "${s118a[*]} $(( z ))"; declare -A s118m=([k]=3); declare -n s118q=s118m; (( s118q[k]++ )); echo "${s118m[k]}"; read -r 's118r[4]' <<< hey; printf -v 's118r[5]' %s pv; echo "${s118a[4]} ${s118a[5]}"); check nra3 "$r" '21
20 20 35 21
4
hey pv'
  r=$(s118b=(1); declare -gn s118n=s118b; e=$(declare -gA s118n 2>&1); s118h "$e" "declare: s118n: cannot convert indexed to associative array"; e=$(declare -g +a s118n 2>&1); s118h "$e" "declare: s118n: cannot destroy array variables in this way"; readonly s118o=5; declare -gn s118p=s118o; e=$(declare -g s118p=8 2>&1); s118h "$e" "declare: s118p: readonly variable"; e=$( (s118p=6; echo no) 2>&1); s118h "$e" "s118o: readonly variable"; e=$( (read s118p <<< 7; echo "rc=$?") 2>&1); s118h "$e" "rc=1"; e=$(unset s118p 2>&1); s118h "$e" "unset: s118o: cannot unset: readonly variable"; echo "$s118o"); check nra4 "$r" '1
1
1
1
1
1
5'
}
# ===== SECTION 119: export -p NAME, and an expansion error ends the command =====
# export and readonly with -p and a name give the name the attribute and print nothing. After
# an expansion error (a bad substitution, an arithmetic error, ${v?}) the rest of the command
# is not expanded: only the first error is reported and a later command substitution does not
# run. In double quotes a ${NAME:$'...'} whose text starts with -, =, ? or + is read as that
# text written there. Every expected value is bash 5.3's.
s119c() { case $1 in *"$2"*"$2"*) echo 2 ;; *"$2"*) echo 1 ;; *) echo 0 ;; esac; }
s119() {
  local r e
  r=$(export -p s119a; echo "rc=$?"; readonly -p s119b; export -p s119c=5; readonly -p s119d=x s119e; declare -p s119a s119b s119c s119d s119e); check xpp1 "$r" 'rc=0
declare -x s119a
declare -r s119b
declare -x s119c="5"
declare -r s119d="x"
declare -r s119e'
  e=$( (echo ${#s119x-a} ${#s119y-b}; echo no) 2>&1); r=$(s119c "$e" "bad substitution"; s119c "$e" "no")
  e=$( (echo $((1/0)) $(echo x$((6*7)) >&2) $((2/0))) 2>&1); r="$r
$(s119c "$e" "division by 0"; s119c "$e" "x42")"
  e=$( (s119f() { echo "in ${#s119x-a} $(echo x$((6*7)) >&2)"; echo cont; }; s119f; echo after) 2>&1); r="$r
$(s119c "$e" "bad substitution"; s119c "$e" "x42"; s119c "$e" "cont"; s119c "$e" "after")"
  e=$( (echo ${s119u?gone} $(echo x$((6*7)) >&2); echo no) 2>&1); r="$r
$(s119c "$e" "s119u: gone"; s119c "$e" "x42"; s119c "$e" "no")"
  check xpp2 "$r" '1
0
1
0
1
0
0
0
1
0
0'
  r=$(s=hello; echo "${s:$'-1'}" "${s:$'+1'}" "${s:$'-'}" "${s:$'?e'}" "${s:$'1'}" "${s:$"-1"}" "${s119u:$'-dflt'}" "${s:$'1':$'2'}"); check xpp3 "$r" 'hello 1 hello hello ello o dflt el'
}
# ===== SECTION 120: the subscript of an arithmetic text read at run time =====
# A variable's value or a let argument used as arithmetic expands an indexed subscript as in
# double quotes, where a ' is a character. In a $(( )) text that holds an expansion, what an
# expansion or a double-quoted part puts into a subscript is not expanded again, and a ' it
# holds is shown with a backslash. An error inside a subscript names the subscript text with
# its trailing blanks, and a "no closing }" message shows a $'...' decoded. Every expected
# value is bash 5.3's.
s120m() { local e=${1#*line }; e=${e#*: }; printf '%s\n' "$e"; }
s120() {
  local r e s120v=(10 20 30) x=1 z
  r=$(z='s120v[$x]'; echo $(( z )); z="s120v[\$x+1]"; echo $(( z )); z="s120v[\"\$x\"]"; echo $(( z )); let 'z=s120v[$x]'; echo "$z")
  check sbq1 "$r" '20
30
20
20'
  e=$( (z="s120v['\$x']"; echo $(( z ))) 2>&1); r=$(s120m "$e")
  e=$( (let 's120w=s120v['\''$x'\'']') 2>&1); r="$r
$(s120m "$e")"
  e=$( (echo $(( s120v[ '1' ] ))) 2>&1); r="$r
$(s120m "$e")"
  check sbq2 "$r" "'1': arithmetic syntax error: operand expected (error token is \"'1'\")
'1': arithmetic syntax error: operand expected (error token is \"'1'\")
'1' : arithmetic syntax error: operand expected (error token is \"'1' \")"
  e=$( (echo $(( s120v["'1'"] + $x ))) 2>&1); r=$(s120m "$e")
  e=$( (d='$x'; echo $(( s120v[$d] ))) 2>&1); r="$r
$(s120m "$e")"
  r="$r
$(echo $(( s120v[ "1" ] + $x )))"
  e=$( (echo $(( s120v["'1'"] ))) 2>&1); r="$r
$(s120m "$e")"
  check sbq3 "$r" "\\'1\\': arithmetic syntax error: operand expected (error token is \"\\'1\\'\")
\$x: arithmetic syntax error: operand expected (error token is \"\$x\")
21
'1': arithmetic syntax error: operand expected (error token is \"'1'\")"
  local -A s120a
  e=$( (echo x"${s120a[$'x'[]-v}"y) 2>&1); r=$(s120m "$e")
  check sbq4 "$r" "bad substitution: no closing \`}' in x\"\${s120a[x[]-v}\"y"
}
# ===== SECTION 121: readonly attributes, and namerefs meeting arrays =====
# A readonly variable refuses -a, -A, -i, -l, -u, -n and their + forms (none of the line's
# attributes apply), and allows -x. -n meets an array: a global declaration of a non-array
# with -a drops the -n; an array variable, an array value or -a on a local is refused
# (the value still assigned, the local made an empty array). A set variable made a nameref
# must already name one. An element or literal store through a circular nameref warns and
# lands on the nameref itself, now a plain array; through a nameref to NAME[SUB] it is not
# a valid identifier, and a subscripted read finds no variable. Every expected value is
# bash 5.3's.
s121() {
  local r e l nl=$'\n'
  r=$( (readonly s121z=1
    declare -gi s121z; echo "i=$?"; declare -gu s121z; echo "u=$?"; declare -ga s121z; echo "a=$?"
    declare -gA s121z; echo "A=$?"; declare -g +i s121z; echo "pi=$?"; declare -gix s121z; echo "ix=$?"
    declare -gx s121z; echo "x=$?"; declare -p s121z) 2>/dev/null)
  check sro1 "$r" 'i=1
u=1
a=1
A=1
pi=1
ix=1
x=0
declare -rx s121z="1"'
  e=$( (readonly s121y=1 s121u=abc; declare -gl s121y; declare -gn s121y; declare -gn s121u) 2>&1); l=$LINENO
  check sro2 "$e" "$0: line $l: declare: s121y: readonly variable
$0: line $l: declare: \`1': invalid variable name for name reference
$0: line $l: declare: s121u: readonly variable"
  r=$( (declare -gan s121a; echo "1=$?"; declare -p s121a
    s121b=v; declare -gan s121b; echo "2=$?"; declare -p s121b
    s121c=(1); declare -gn s121c; echo "3=$?"; declare -p s121c
    declare -gn s121d=(p q); echo "4=$?"; declare -p s121d
    local -an s121e=y; echo "5=$?"; declare -p s121e
    s121f=1; declare -gn s121f; echo "6=$?"; declare -p s121f
    declare -gn s121g; declare -ga s121g; echo "7=$?"; declare -p s121g) 2>/dev/null)
  check snr1 "$r" '1=0
declare -a s121a
2=0
declare -a s121b=([0]="v")
3=1
declare -a s121c=([0]="1")
4=1
declare -a s121d=([0]="p" [1]="q")
5=1
declare -a s121e
6=1
declare -- s121f="1"
7=0
declare -a s121g'
  e=$( (s121c=(1); declare -gn s121c; local -An s121h) 2>&1); l=$LINENO
  check snr2 "$e" "$0: line $l: declare: s121c: reference variable cannot be an array
$0: line $l: local: s121h: reference variable cannot be an array"
  r=$( (declare -gn s121p=s121q s121q=s121p; s121p[1]=4; echo "rc=$?"; declare -p s121p; s121p=5; declare -p s121p
    declare -gn s121r=s121s s121s=s121r; s121r=(x y); echo "l=$?"; declare -p s121r s121s) 2>/dev/null)
  check scr1 "$r" 'rc=0
declare -a s121p=([1]="4")
declare -a s121p=([0]="5" [1]="4")
l=0
declare -a s121r=([0]="x" [1]="y")
declare -n s121s="s121r"'
  e=$( (declare -gn s121p=s121q s121q=s121p; s121p[1]=4) 2>&1); l=$LINENO
  check scr2 "$e" "$0: line $l: warning: s121p: circular name reference
$0: line $l: warning: s121p: circular name reference"
  r=$( (s121v=(1 2 3); declare -gn s121w=s121v[1]; echo "${s121w[0]}|$s121w|${s121w[@]}|${#s121w[@]}|${s121w[1]:-d}") 2>/dev/null)
  check sne1 "$r" '|2||0|d'
  e=$( (s121v=(1 2 3); declare -gn s121w=s121v[1]; s121w[0]=9; echo no) 2>&1); l=$LINENO
  r="$0: line $l: \`s121v[1]': not a valid identifier"
  e="$e$nl$( (s121v=(1 2 3); declare -gn s121w=s121v[1]; s121w=(7 8); echo no) 2>&1)"; l=$LINENO
  r="$r$nl$0: line $l: \`s121v[1]': not a valid identifier"
  e="$e$nl$( (s121v=(1 2 3); declare -gn s121w=s121v[1]; s121w+=(z); echo no) 2>&1)"; l=$LINENO
  r="$r$nl$0: line $l: \`s121v[1]': not a valid identifier"
  check sne2 "$e" "$r"
}
# ===== SECTION 122: the word a bad substitution names; brackets in arithmetic subscripts =====
# "bad substitution" names the double-quoted string the ${...} is in, else its whole word,
# with a $( ) in it printed again and an unquoted $'...' single-quoted. A [ or ] an
# expansion puts into an arithmetic subscript neither nests nor ends it, and an associative
# key keeps it. bash reads the token after an operand before it evaluates the operand, so
# a stray "]]" is reported first. A local of an earlier $( ) is none in the function after
# it. Every expected value is bash 5.3's.
s122() {
  local r e f w nl=$'\n' x=abc t='arithmetic syntax error: invalid arithmetic operator (error token is'
  local -A m
  e=$( (echo "pre ${#x-a} post") 2>&1); w="$0: line $LINENO: pre \${#x-a} post: bad substitution"
  f=$( (echo x${#x-a}y) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: x\${#x-a}y: bad substitution"
  f=$( (echo x${m[$'x'[]-v}y) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: x\${m['x'[]-v}y: bad substitution"
  f=$( (echo "a $(echo  hi   there) ${#x-a} b") 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: a \$(echo hi there) \${#x-a} b: bad substitution"
  f=$( (echo pre"in ${#x-a}"post) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: in \${#x-a}: bad substitution"
  f=$( (echo "a\"b ${#x-a}") 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: a\"b \${#x-a}: bad substitution"
  f=$( (y=p${#x-a}q) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: p\${#x-a}q: bad substitution"
  f=$( (echo "<$(echo x${#x-a}y)>") 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: x\${#x-a}y: bad substitution$nl<>"
  check sbw1 "$e" "$w"
  local iv=(10 20 30) g='1]' h='[' p='1+'
  local -A m2=([1]=5 ['1]']=7)
  e=$( (echo $(( iv[$g+1] ))) 2>&1); w="$0: line $LINENO: 1]+1: $t \"]+1\")"
  f=$( (echo $(( iv[1 $h] ))) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: 1 [: $t \"[\")"
  f=$( (echo $(( iv[ '1' ] ]] ))) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: iv[ '1' ] ]] : $t \"]] \")"
  f=$( (echo $(( p ]] ))) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: p ]] : $t \"]] \")"
  f=$( (echo $(( iv[ $g ] ]] ))) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: iv[ 1\\] ] ]] : $t \"]] \")"
  check sab1 "$e" "$w"
  check sab2 "$(( m2[$g] + 0 ))" 7
  r=$(declare -gi s122g; declare -gu s122u
    s122f() {
      : $(local s122g)
      local s122g
      s122g=1+1
      : $(local s122u)
      local s122u=abc
      : $(local s122q=1)
      s122q=2
      echo "$s122g $s122u"
    }
    s122f
    echo "$s122q")
  check slc1 "$r" '1+1 abc
2'
}
# ===== SECTION 123: the capcase and trace attributes =====
# declare -c keeps every later value of a name capitalized: the first character upper, the
# rest lower, for an element, an append, `read`, `printf -v` and a nameref's store alike. A
# name holds one of -u, -l and -c: each turns the others off, two at once leave none, and +X
# wins over -X. -t (trace) changes no value; both show in declare -p and ${v@a}, in bash's
# order a i n r t x c l u. A readonly name refuses -c but takes -t. Every expected value is
# bash 5.3's.
s123() {
  local r
  r=$(declare -c a=hELLO; echo "$a"; a="wORLD foo"; echo "$a"; a+=bAR; declare -p a
    declare -c g="" h=1aBC i=" aB"; declare -p g h i
    declare -cu b=xyZ; declare -cl c=XyZ; declare -c +c d=aB; declare +c -c e=aB; declare -p b c d e
    declare -u f=abc; declare -c f; f=qWE; declare -p f; declare -l f; f=QwE; declare -p f
    declare -ac n=(hELLO wORLD); n+=(eF); n[5]=zZ; declare -p n
    declare -A m; declare -c m; m[k]=vAL; m[k]+=xX; declare -p m
    local -a k=( xX yY ); declare -c k; k[2]=zZ; declare -p k
    declare -n r=a; r=nEW; declare -p a
    read -r a <<< "bAR baZ"; declare -p a; printf -v a '%s' yY; declare -p a
    for a in oNe tWO; do :; done; declare -p a
    declare -ic q=3+4; declare -p q)
  check scc1 "$r" 'Hello
World foo
declare -c a="World foobar"
declare -c g=""
declare -c h="1abc"
declare -c i=" ab"
declare -- b="xyZ"
declare -- c="XyZ"
declare -- d="aB"
declare -- e="aB"
declare -c f="Qwe"
declare -l f="qwe"
declare -ac n=([0]="Hello" [1]="World" [2]="Ef" [5]="Zz")
declare -Ac m=([k]="Valxx" )
declare -ac k=([0]="xX" [1]="yY" [2]="Zz")
declare -c a="New"
declare -c a="Bar baz"
declare -c a="Yy"
declare -c a="Two"
declare -ic q="7"'
  r=$(declare -t t=5; declare -p t; declare -tx x=1; declare -p x; declare +t t; declare -p t
    declare -tcxir b=1; declare -p b; echo "${b@a}"; declare -ta d=(1); declare -p d
    declare -tc y; echo "${y@a}|"; declare -t v=1; echo "${v@A}"
    s123f() { local -c l=mIXED; declare -p l; local -t lt=1; declare -p lt; declare -t t; echo "${t@a}"; }
    s123f; declare -p t
    declare -t u=1; unset u; u=2; declare -p u
    declare -t w; ( declare +t w; declare -p w ); declare -p w)
  check sct1 "$r" 'declare -t t="5"
declare -tx x="1"
declare -- t="5"
declare -irtxc b="1"
irtxc
declare -at d=([0]="1")
tc|
declare -t v='"'"'1'"'"'
declare -c l="Mixed"
declare -t lt="1"
t
declare -- t="5"
declare -- u="2"
declare -- w
declare -t w'
  r=$( (declare -r o=1; declare -c o; echo "c=$?"; declare +c o; echo "c=$?"; declare -t o; echo "t=$?"; declare -p o) 2>&1); local ln=$LINENO
  check scr1 "$r" "$0: line $ln: declare: o: readonly variable
c=1
$0: line $ln: declare: o: readonly variable
c=1
t=0
declare -rt o=\"1\""
}
# ===== END SECTIONS =====

s90   # SECTION-CALL 90
s91   # SECTION-CALL 91
s92   # SECTION-CALL 92
s93   # SECTION-CALL 93
s94   # SECTION-CALL 94
s95   # SECTION-CALL 95
s96   # SECTION-CALL 96
s97   # SECTION-CALL 97
s98   # SECTION-CALL 98
s99   # SECTION-CALL 99
s100  # SECTION-CALL 100
s101  # SECTION-CALL 101
s102  # SECTION-CALL 102
s103  # SECTION-CALL 103
s104  # SECTION-CALL 104
s105  # SECTION-CALL 105
s106  # SECTION-CALL 106
s107  # SECTION-CALL 107
s108  # SECTION-CALL 108
s109  # SECTION-CALL 109
s110  # SECTION-CALL 110
s111  # SECTION-CALL 111
s112  # SECTION-CALL 112
s113  # SECTION-CALL 113
s114  # SECTION-CALL 114
s115  # SECTION-CALL 115
s116  # SECTION-CALL 116
s117  # SECTION-CALL 117
s118  # SECTION-CALL 118
s119  # SECTION-CALL 119
s120  # SECTION-CALL 120
s121  # SECTION-CALL 121
s122  # SECTION-CALL 122
s123  # SECTION-CALL 123
echo "full: $checks checks, $fails failures"
exit $fails
