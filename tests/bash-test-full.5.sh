#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 5
# of 5, sections 90 on; a new section goes here. SECTION 96's grown table alone takes
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
echo "full: $checks checks, $fails failures"
exit $fails
