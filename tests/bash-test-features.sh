#!/usr/bin/env bash
# Fast feature-matrix test for the Bash interpreter (bash-interpreter.abnf) and the
# LLVM-IR compiler (bash-to-llvm-ir.abnf). Same protocol as every other
# tests/<lang>-test-features.*: every implemented construct is exercised with the
# SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times, recursion
# stays below depth 6. A failed check prints its id (so a diff pinpoints it) and the
# program ends with `exit $fails`; exit 0 and byte-identical output on all four legs
# (interpreter/compiler x goja/-frozen) mean everything passed.
#
# bash's layer 2 is the C runtime languages/lib/bash-rt.c, and this file is written
# to hit its entry points on purpose: rt_strip / rt_replace / rt_case for
# ${v#p} ${v/p/r} ${v^^}, rt_glob and rt_eg for pattern and extended-glob matching,
# the re_* regex builder for [[ =~ ]], rt_arr_* for both array kinds, rt_field /
# rt_splitifs / rt_wordjoin for IFS work, rt_pad for printf widths, rt_unescape for
# echo -e, rt_ansic for $'...', rt_shquote for ${v@Q}, rt_cap_begin/rt_cap_end for
# command substitution, rt_ss_save/rt_ss_restore for subshells,
# rt_push_local/rt_pop_locals for `local`, rt_getvar_byname/rt_setvar_byname for
# declare and indirect expansion, rt_eval_assign for eval, and rt_fskind for -e/-d/-f.
#
# EVERY ASSERTION BELOW WAS SETTLED BY RUNNING THIS FILE UNDER THE REAL /bin/bash
# (5.3 on the development machine), not derived from documentation, and real bash
# must keep printing the same summary line. Where our engines deliberately diverge
# from bash - the 0x02 first byte, [[:word:]] / [[:ascii:]], `declare -A` with extra
# words, associative-array iteration order (all four listed in
# docs/working-on-this-project.md chapter 4) - nothing is asserted either way here.
#
# Out of scope for the same reasons as the ratchet: external commands of every kind
# (so no cat/ls/sed/grep), pathname expansion against a real directory, process
# substitution, job control, $RANDOM/$SECONDS, and anything that writes to the
# filesystem. Redirection is exercised against /dev/null and through here-documents.

fails=0
checks=0

# cat_stub reads stdin and echoes it, so nothing here needs an external `cat`.
cat_stub() {
  local line first=1 out=""
  while IFS= read -r line; do
    if [ $first -eq 1 ]; then out="$line"; first=0; else out="$out
$line"; fi
  done
  printf '%s' "$out"
}

# check ID GOT WANT : one assertion.
check() {
  checks=$(( checks + 1 ))
  if [ "$2" != "$3" ]; then
    echo "FAIL $1"
    fails=$(( fails + 1 ))
  fi
}

# ----- variables, quoting, escapes -----
v=5
check var1 "$v" 5
check var2 "${v}0" 50
check var3 "value is $v" "value is 5"
check var4 'no $v here' 'no $v here'
check var5 "a\$b" 'a$b'
check var6 "a\\b" 'a\b'
check var7 "say \"hi\"" 'say "hi"'
check var8 a\ b "a b"
check var9 'a'"b"c abc
check var10 'it'\''s' "it's"
unset v
check var11 "$v" ""
e=
check var12 "$e" ""
check var13 "${e-set}" ""
check var14 "${e:-set}" set

# ----- ANSI-C quoting, $'...' : rt_ansic -----
check ans1 $'a\tb' "a	b"
check ans2 $'x\ny' "x
y"
check ans3 $'\\' '\'
check ans4 $'\x41\x42' AB
check ans5 $'\101' A
cr=$'a\rb'
check ans6 "${#cr}" 3
check ans7 $'q\'r' "q'r"
check ans8 $'\x27' "'"

# ----- parameter expansion: defaults, length, indirect -----
name=hello
check par1 "${#name}" 5
check par2 "${#}" 0
unset miss
check par3 "${miss:-fallback}" fallback
check par4 "${name:-fallback}" hello
check par5 "${miss:+alt}" ""
check par6 "${name:+alt}" alt
unset asn
check par7 "${asn:=given}" given
check par8 "$asn" given
tgt=42
ptr=tgt
check par9 "${!ptr}" 42
check par10 "${#miss}" 0

# ----- substrings, patterns, case: rt_substr, rt_strip, rt_replace, rt_case -----
p=/usr/local/lib/libfoo.so.1
check sub1 "${p:5}" local/lib/libfoo.so.1
check sub2 "${p:1:3}" usr
check sub3 "${p: -1}" 1
check sub4 "${p:0:0}" ""
check sub5 "${p#*/}" usr/local/lib/libfoo.so.1
check sub6 "${p##*/}" libfoo.so.1
check sub7 "${p%.*}" /usr/local/lib/libfoo.so
check sub8 "${p%%.*}" /usr/local/lib/libfoo
check sub9 "${p#nomatch}" "$p"
s=banana
check sub10 "${s/na/NA}" baNAna
check sub11 "${s//na/NA}" baNANA
check sub12 "${s/#ba/BA}" BAnana
check sub13 "${s/%na/NA}" banaNA
check sub14 "${s//a}" bnn
check sub15 "${s/?a/X}" Xnana
check sub16 "${s^}" Banana
check sub17 "${s^^}" BANANA
u=BANANA
check sub18 "${u,}" bANANA
check sub19 "${u,,}" banana
mx=mIxEd
check sub20 "${mx^^}" MIXED
check sub21 "${mx,,}" mixed
check sub22 "${mx^}" MIxEd
# A case op with a PATTERN operand (${s^^[bn]}) is asserted in tests/bash-test-full.sh.

# ----- ${v@Q} and friends: rt_shquote -----
q="a b"
check opq1 "${q@Q}" "'a b'"
check opq2 "${name@Q}" "'hello'"
check opq3 "${name@U}" HELLO
check opq4 "${u@L}" banana

# ----- arithmetic: every operator, the bases, the update forms -----
check ari1 "$(( 2 + 3 * 4 ))" 14
check ari2 "$(( (2 + 3) * 4 ))" 20
check ari3 "$(( 7 / 2 ))" 3
check ari4 "$(( -7 / 2 ))" -3
check ari5 "$(( 7 % -2 ))" 1
check ari6 "$(( 2 ** 10 ))" 1024
check ari7 "$(( 1 << 4 ))" 16
check ari8 "$(( 255 >> 4 ))" 15
check ari9 "$(( 12 & 10 ))" 8
check ari10 "$(( 12 | 3 ))" 15
check ari11 "$(( 12 ^ 10 ))" 6
check ari12 "$(( ~0 ))" -1
check ari13 "$(( !0 ))" 1
check ari14 "$(( 1 && 0 ))" 0
check ari15 "$(( 1 || 0 ))" 1
check ari16 "$(( 3 > 2 ? 10 : 20 ))" 10
check ari17 "$(( 0x1f + 0 ))" 31
check ari18 "$(( 010 + 0 ))" 8
check ari19 "$(( 2#1011 ))" 11
check ari20 "$(( 1, 2, 3 ))" 3
check ari21 "$(( 3 == 3 ))" 1
check ari22 "$(( 3 != 3 ))" 0
n=5
check ari23 "$(( n++ ))" 5
check ari24 "$n" 6
check ari25 "$(( ++n ))" 7
check ari26 "$(( n-- ))" 7
check ari27 "$(( --n ))" 5
m=10
(( m += 5 ))
check ari28 "$m" 15
(( m *= 2 ))
check ari29 "$m" 30
(( m /= 3 ))
check ari30 "$m" 10
(( m %= 4 ))
check ari31 "$m" 2
let "lk = 3 * 4"
check ari32 "$lk" 12
if (( 1 )); then check ari33 ok ok; else check ari33 no ok; fi
if (( 0 )); then check ari34 no ok; else check ari34 ok ok; fi

# ----- command substitution: rt_cap_begin / rt_cap_end -----
check cmd1 "$(echo hi)" hi
check cmd2 "`echo old`" old
check cmd3 "a$(echo b)c" abc
check cmd4 "$(echo "$(echo deep)")" deep
cs=$(echo 7)
check cmd5 "$cs" 7
co=$(false)
check cmd6 "$?" 1
check cmd7 "$co" ""
check cmd8 "$(( $(echo 3) + 4 ))" 7
check cmd9 "$(echo -n trailing)" trailing
check cmd10 "$(printf 'x\n\n\n')" x

# ----- the test builtin, and rt_fskind for the file operators -----
if [ 5 -eq 5 ]; then check tst1 ok ok; else check tst1 no ok; fi
if [ 5 -ne 6 ]; then check tst2 ok ok; else check tst2 no ok; fi
if [ 3 -lt 4 ]; then check tst3 ok ok; else check tst3 no ok; fi
if [ 4 -le 4 ]; then check tst4 ok ok; else check tst4 no ok; fi
if [ 9 -gt 2 ]; then check tst5 ok ok; else check tst5 no ok; fi
if [ 9 -ge 9 ]; then check tst6 ok ok; else check tst6 no ok; fi
if [ abc = abc ]; then check tst7 ok ok; else check tst7 no ok; fi
if [ abc != xyz ]; then check tst8 ok ok; else check tst8 no ok; fi
if [ -z "" ]; then check tst9 ok ok; else check tst9 no ok; fi
if [ -n x ]; then check tst10 ok ok; else check tst10 no ok; fi
if [ ! -z x ]; then check tst11 ok ok; else check tst11 no ok; fi
if [ 1 -eq 1 -a 2 -eq 2 ]; then check tst12 ok ok; else check tst12 no ok; fi
if [ 1 -eq 2 -o 2 -eq 2 ]; then check tst13 ok ok; else check tst13 no ok; fi
if [ -e /dev/null ]; then check tst14 ok ok; else check tst14 no ok; fi
if [ -d /dev ]; then check tst15 ok ok; else check tst15 no ok; fi
# rt_fskind is a hard-coded path model (no stat() in the C floor), so `-f` on a real
# regular file is false here; -e and -d are asserted only on paths the model and the
# real filesystem agree about.
if [ -e /tmp ]; then check tst16 ok ok; else check tst16 no ok; fi
if [ -e /no/such/path/here ]; then check tst17 no ok; else check tst17 ok ok; fi
if test 1 -eq 1; then check tst18 ok ok; else check tst18 no ok; fi

# ----- [[ ]] : string comparison, globs (rt_glob), extglob (rt_eg) -----
str=hello
if [[ $str == hello ]]; then check dbr1 ok ok; else check dbr1 no ok; fi
if [[ $str != world ]]; then check dbr2 ok ok; else check dbr2 no ok; fi
if [[ $str == h*o ]]; then check dbr3 ok ok; else check dbr3 no ok; fi
if [[ $str == "h*o" ]]; then check dbr4 no ok; else check dbr4 ok ok; fi
if [[ $str == h?llo ]]; then check dbr5 ok ok; else check dbr5 no ok; fi
if [[ $str == [gh]ello ]]; then check dbr6 ok ok; else check dbr6 no ok; fi
if [[ $str == [!xy]ello ]]; then check dbr7 ok ok; else check dbr7 no ok; fi
if [[ abc < abd ]]; then check dbr8 ok ok; else check dbr8 no ok; fi
if [[ abd > abc ]]; then check dbr9 ok ok; else check dbr9 no ok; fi
if [[ -n $str && -z "" ]]; then check dbr10 ok ok; else check dbr10 no ok; fi
if [[ -z $str || -n $str ]]; then check dbr11 ok ok; else check dbr11 no ok; fi
if [[ ! -z $str ]]; then check dbr12 ok ok; else check dbr12 no ok; fi
if [[ ( 1 -eq 1 ) && ( 2 -eq 2 ) ]]; then check dbr13 ok ok; else check dbr13 no ok; fi
if [[ abc == @(abc|def) ]]; then check dbr14 ok ok; else check dbr14 no ok; fi
if [[ xy == ?(x)y ]]; then check dbr15 ok ok; else check dbr15 no ok; fi
if [[ y == ?(x)y ]]; then check dbr16 ok ok; else check dbr16 no ok; fi
if [[ xxxy == *(x)y ]]; then check dbr17 ok ok; else check dbr17 no ok; fi
if [[ y == +(x)y ]]; then check dbr18 no ok; else check dbr18 ok ok; fi
if [[ zy == !(x)y ]]; then check dbr19 ok ok; else check dbr19 no ok; fi

# ----- [[ =~ ]] : the re_* regex builder -----
if [[ abc123 =~ ^[a-z]+[0-9]+$ ]]; then check rgx1 ok ok; else check rgx1 no ok; fi
if [[ abc =~ ^[0-9]+$ ]]; then check rgx2 no ok; else check rgx2 ok ok; fi
if [[ 2026-08-08 =~ ^([0-9]{4})-([0-9]{2})-([0-9]{2})$ ]]; then
  check rgx3 "${BASH_REMATCH[0]}" 2026-08-08
  check rgx4 "${BASH_REMATCH[1]}" 2026
  check rgx5 "${BASH_REMATCH[3]}" 08
else
  check rgx3 no ok; check rgx4 no ok; check rgx5 no ok
fi
if [[ foobar =~ o+b ]]; then check rgx6 ok ok; else check rgx6 no ok; fi
if [[ foobar =~ x?bar ]]; then check rgx7 ok ok; else check rgx7 no ok; fi
if [[ ab =~ a|z ]]; then check rgx8 ok ok; else check rgx8 no ok; fi
if [[ "a.c" =~ a\.c ]]; then check rgx9 ok ok; else check rgx9 no ok; fi
if [[ abc =~ [[:alpha:]]+ ]]; then check rgx10 ok ok; else check rgx10 no ok; fi
if [[ 42 =~ [[:digit:]][[:digit:]] ]]; then check rgx11 ok ok; else check rgx11 no ok; fi
if [[ aXc =~ a[^0-9]c ]]; then check rgx12 ok ok; else check rgx12 no ok; fi

# ----- case -----
casev() {
  case "$1" in
    apple)   echo fruit ;;
    b*)      echo bee ;;
    x|y|z)   echo letter ;;
    [0-9])   echo digit ;;
    *)       echo other ;;
  esac
}
check cas1 "$(casev apple)" fruit
check cas2 "$(casev banana)" bee
check cas3 "$(casev y)" letter
check cas4 "$(casev 7)" digit
check cas5 "$(casev qqq)" other

# ----- indexed arrays: rt_arr_* -----
arr=(alpha beta gamma)
check arr1 "${arr[0]}" alpha
check arr2 "${arr[2]}" gamma
check arr3 "${#arr[@]}" 3
check arr4 "${#arr[1]}" 4
check arr5 "${arr[*]}" "alpha beta gamma"
arr[1]=BETA
check arr6 "${arr[1]}" BETA
arr+=(delta)
check arr7 "${#arr[@]}" 4
check arr8 "${arr[3]}" delta
check arr9 "${arr[*]:1:2}" "BETA gamma"
check arr10 "${!arr[*]}" "0 1 2 3"
check arr11 "${arr[9]}" ""
unset 'arr[0]'
check arr12 "${#arr[@]}" 3
joined=""
for e in "${arr[@]}"; do joined="$joined:$e"; done
check arr13 "$joined" ":BETA:gamma:delta"
empty_arr=()
check arr14 "${#empty_arr[@]}" 0
two=("a b" "c d")
check arr15 "${#two[@]}" 2
check arr16 "${two[0]}" "a b"
sparse=()
sparse[5]=five
check arr17 "${#sparse[@]}" 1
check arr18 "${!sparse[*]}" 5

# ----- associative arrays (NOT iteration order: that is a documented divergence) -----
declare -A ages
ages[ann]=30
ages[bob]=41
check asc1 "${ages[ann]}" 30
check asc2 "${ages[bob]}" 41
check asc3 "${#ages[@]}" 2
check asc4 "${ages[nobody]}" ""
if [[ -v ages[ann] ]]; then check asc5 ok ok; else check asc5 no ok; fi
if [[ -v ages[nobody] ]]; then check asc6 no ok; else check asc6 ok ok; fi
ages[ann]=31
check asc7 "${ages[ann]}" 31
unset 'ages[bob]'
check asc8 "${#ages[@]}" 1
declare -A colors=([red]=ff0000 [green]=00ff00)
check asc9 "${colors[red]}" ff0000
check asc10 "${colors[green]}" 00ff00

# ----- brace expansion -----
check brc1 "$(echo a{1,2,3}b)" "a1b a2b a3b"
check brc2 "$(echo {1..5})" "1 2 3 4 5"
check brc3 "$(echo {5..1})" "5 4 3 2 1"
check brc4 "$(echo {a..e})" "a b c d e"
check brc5 "$(echo {1..9..3})" "1 4 7"
check brc6 "$(echo pre{x,y}post)" "prexpost preypost"
check brc7 "$(echo {a,b}{1,2})" "a1 a2 b1 b2"
check brc8 "$(echo {single})" "{single}"

# ----- word splitting and IFS: rt_splitifs, rt_wordjoin, rt_getfield -----
packed="a b c"
parts=($packed)
check wsp1 "${#parts[@]}" 3
check wsp2 "${parts[1]}" b
whole=("$packed")
check wsp3 "${#whole[@]}" 1
saved_ifs="$IFS"
IFS=:
fields=("x:y:z")
fields=($(echo "x:y:z"))
check wsp4 "${#fields[@]}" 3
check wsp5 "${fields[2]}" z
set -- one two
check wsp6 "$*" "one:two"
IFS="$saved_ifs"
set --
check wsp7 "$#" 0
nothing=""
none=($nothing)
check wsp8 "${#none[@]}" 0
sq="  padded  "
trimmed=($sq)
check wsp9 "${#trimmed[@]}" 1
check wsp10 "${trimmed[0]}" padded

# ----- loops: 0, 1, 3 and 4 iterations -----
acc=""
for w in x y z; do acc="$acc$w"; done
check lop1 "$acc" xyz
acc=""
for w in solo; do acc="$acc$w"; done
check lop2 "$acc" solo
acc=""
for w in ""; do acc="${acc}."; done
check lop3 "$acc" .
sum=0
for k in 1 2 3 4; do sum=$(( sum + k )); done
check lop4 "$sum" 10
sum=0
i=0
while [ $i -lt 4 ]; do sum=$(( sum + i )); i=$(( i + 1 )); done
check lop5 "$sum" 6
sum=0
i=0
until [ $i -ge 3 ]; do sum=$(( sum + 1 )); i=$(( i + 1 )); done
check lop6 "$sum" 3
sum=0
for (( c = 0; c < 4; c++ )); do sum=$(( sum + c )); done
check lop7 "$sum" 6
acc=""
for w in a b c d; do
  if [ "$w" = b ]; then continue; fi
  if [ "$w" = d ]; then break; fi
  acc="$acc$w"
done
check lop8 "$acc" ac
acc=""
for o in 1 2; do
  for inner in a b; do
    if [ "$inner" = b ]; then continue 2; fi
    acc="$acc$o$inner"
  done
done
check lop9 "$acc" 1a2a
never=0
for w in ; do never=1; done
check lop10 "$never" 0
sum=0
while false; do sum=1; done
check lop11 "$sum" 0
acc=""
for e in "${arr[@]}"; do acc="$acc|$e"; done
check lop12 "$acc" "|BETA|gamma|delta"

# ----- functions, local, return, recursion (below depth 6) -----
addtwo() { gsum=$(( $1 + $2 )); }
addtwo 12 30
check fun1 "$gsum" 42
retval() { return 3; }
retval
check fun2 "$?" 3
scoped() { local hidden=inner; echo "$hidden"; }
outer=untouched
hidden=outer_value
check fun3 "$(scoped)" inner
check fun4 "$hidden" outer_value
fact() {
  if [ "$1" -le 1 ]; then echo 1; return; fi
  local sub
  sub=$(fact $(( $1 - 1 )))
  echo $(( $1 * sub ))
}
check fun5 "$(fact 5)" 120
check fun6 "$(fact 1)" 1
function kwform() { echo kw; }
check fun7 "$(kwform)" kw
nested() { inner_fn() { echo deep; }; inner_fn; }
check fun8 "$(nested)" deep
echoall() { echo "$#:$1:$2"; }
check fun9 "$(echoall a b)" "2:a:b"
check fun10 "$(echoall)" "0::"

# ----- positional parameters, shift, $@ vs $* -----
posargs() {
  local out="$#"
  shift
  out="$out/$1"
  out="$out/$*"
  echo "$out"
}
check pos1 "$(posargs a b c)" "3/b/b c"
countargs() { echo $#; }
check pos2 "$(countargs 1 2 3 4)" 4
check pos3 "$(countargs)" 0
atstar() {
  local n=0
  for a in "$@"; do n=$(( n + 1 )); done
  echo $n
}
check pos4 "$(atstar "a b" c)" 2
set -- p q r
check pos5 "$#" 3
check pos6 "$1$2$3" pqr
shift 2
check pos7 "$1" r
set --
check pos8 "$#" 0

# ----- exit status, lists, pipelines, ! -----
true
check sts1 "$?" 0
false
check sts2 "$?" 1
:
check sts3 "$?" 0
flag=untouched
true && flag=ran
check sts4 "$flag" ran
flag=untouched
false && flag=ran
check sts5 "$flag" untouched
flag=untouched
false || flag=ran
check sts6 "$flag" ran
if ! false; then check sts7 ok ok; else check sts7 no ok; fi
check sts8 "$(echo a; echo b)" "a
b"
check sts9 "$(echo pipe | cat_stub)" pipe
(exit 4)
check sts10 "$?" 4
check sts11 "$(true; echo $?)" 0

# ----- subshells and command groups: rt_ss_save / rt_ss_restore -----
outerv=parent
( outerv=child )
check sub_1 "$outerv" parent
check sub_2 "$( outerv=child; echo "$outerv" )" child
{ groupv=set; }
check sub_3 "$groupv" set
check sub_4 "$( { echo one; echo two; } )" "one
two"

# ----- redirection and here-documents -----
echo discarded > /dev/null
check red1 "$?" 0
check red2 "$(cat_stub <<'EOF'
literal $novar
EOF
)" 'literal $novar'
check red3 "$(cat_stub <<EOF
expanded $name
EOF
)" "expanded hello"
check red4 "$(cat_stub <<-EOF
	tab-stripped
	EOF
)" "tab-stripped"
check red5 "$( { echo err >&2; } 2>/dev/null; echo out )" out
red_none=unset
read -r red_none < /dev/null
check red6 "$?" 1

# ----- printf (rt_pad) and echo -e (rt_unescape) -----
check prf1 "$(printf '%s' abc)" abc
check prf2 "$(printf '%d' 42)" 42
check prf3 "$(printf '[%5s]' ab)" "[   ab]"
check prf4 "$(printf '[%-5s]' ab)" "[ab   ]"
check prf5 "$(printf '[%05d]' 42)" "[00042]"
check prf6 "$(printf '%s-%s' a b)" a-b
prf_tab="$(printf 'x\ty')"
check prf7 "${#prf_tab}" 3
check prf8 "$(printf '%c' abc)" a
check prf9 "$(printf '%%')" '%'
check prf10 "$(echo -e 'a\tb')" "a	b"
check prf11 "$(echo -e 'a\\b')" 'a\b'
check prf12 "$(echo -n no-newline)" no-newline
check prf13 "$(echo one   two)" "one two"
check prf14 "$(printf '%s\n' a b c)" "a
b
c"

# ----- read: rt_read_line and rt_field -----
readline() {
  local l
  read -r l <<< "one two three"
  echo "$l"
}
check rdl1 "$(readline)" "one two three"
readfields() {
  local a b c
  read -r a b c <<< "one two three"
  echo "$a|$b|$c"
}
check rdl2 "$(readfields)" "one|two|three"
readrest() {
  local a rest
  read -r a rest <<< "head the whole tail"
  echo "$rest"
}
check rdl3 "$(readrest)" "the whole tail"

# ----- declare, eval, unset: rt_getvar_byname / rt_setvar_byname / rt_eval_assign -----
declare dv=declared
check dcl1 "$dv" declared
declare -i iv=7
iv=iv+1
check dcl2 "$iv" 8
eval 'ev=evaluated'
check dcl3 "$ev" evaluated
eval "check dcl4 \"\$ev\" evaluated"
vname=dv
check dcl5 "${!vname}" declared
eval "$vname=rewritten"
check dcl6 "$dv" rewritten
unset dv
check dcl7 "${dv-gone}" gone
check dcl8 "$(x=inline; echo $x)" inline

# ----- the SCOPE of a declaration: rt_push_local saves the attributes and the
# array too, not only the scalar -----
dsi=1
dsfn1() { declare -i dsi; dsi=2+3; check dsc1 "$dsi" 5; }
dsfn1
check dsc2 "$dsi" 1
dsi="3+4"
check dsc3 "$dsi" "3+4"
dsfn2() { local -i dsj=3; dsj+=4; check dsc4 "$dsj" 7; }
dsfn2
dsj="1+1"
check dsc5 "$dsj" "1+1"
dsfn3() { declare -g dsg=fromfn; export dse=ev; }
dsfn3
check dsc6 "$dsg" fromfn
check dsc7 "$dse" ev
dsa=(g0 g1)
dsfn4() { local -a dsa=(l0); check dsc8 "${dsa[0]}/${#dsa[@]}" "l0/1"; }
dsfn4
check dsc9 "${dsa[0]}-${dsa[1]}-${#dsa[@]}" "g0-g1-2"
declare -A dsm=([gk]=gv)
dsfn5() { local -A dsm; dsm[lk]=lv; check dsc10 "${dsm[lk]}${dsm[gk]}" lv; }
dsfn5
check dsc11 "${dsm[gk]}${dsm[lk]}" gv
declare -u dsu=aa
check dsc12 "$dsu" AA
dsu=bb
check dsc13 "$dsu" BB
check dsc14 "${dsu@a}" u
check dsc15 "${dsa@a}" a
unset dsu
dsu=cc
check dsc16 "$dsu" cc

# ----- string helpers exercised through the shell surface -----
check str1 "${#name}" 5
concat="$name world"
check str2 "$concat" "hello world"
check str3 "${concat// /_}" hello_world
check str4 "$(( ${#concat} ))" 11
rep=""
for k in 1 2 3; do rep="$rep$name"; done
check str5 "${#rep}" 15
check str6 "${rep:5:5}" hello
# $(( 10#$num )) (a base prefix on an expanded operand) is asserted in
# tests/bash-test-full.sh.
# ----- printf conversions: rt_radix / rt_intprec / rt_pfq -----
check pfc1 "$(printf '%x|%X|%o' 12 255 8)" "c|FF|10"
check pfc2 "$(printf '%o|%u' 4095 42)" "7777|42"
check pfc3 "$(printf '%#x|%#o|%#x' 255 8 0)" "0xff|010|0"
check pfc4 "$(printf '[%.3d][%.5x][%.0d]' 7 255 0)" "[007][000ff][]"
check pfc5 "$(printf '[%5x][%-5x][%05x]' 255 255 255)" "[   ff][ff   ][000ff]"
check pfc6 "$(printf '[%.2s]' abcdef)" "[ab]"
check pfc7 "$(printf '%b' 'a\tb')" "$(printf 'a\tb')"
check pfc8 "$(printf '%q|%q' 'a b' '')" "a\\ b|''"
pfv=""
printf -v pfv '%04x' 255
check pfc9 "$pfv" 00ff
pfsp="  12"
check pfc10 "$(printf '%d' "$pfsp")" 12
check pfc11 "$(( pfsp + 1 ))" 13

# IFS is an ordinary SET variable, in both halves.
check ifsv1 "${#IFS}" 3
check ifsv2 "${IFS:0:1}" " "
check ifsv3 "${IFS-unset}" "$IFS"
check str7 "$(( 2#1010 + 1 ))" 11
check str8 "$(( -0 ))" 0

# ----- an assignment's RHS is NOT field-split; a scalar IS a one-element array;
# ----- an element takes a default; a subshell isolates arrays and attributes;
# ----- declare -p prints -----
# Every line here was settled against bash 5.3. tests/bash-test-full.sh SECTION 32
# has the wide version and lists what is deliberately not asserted.
nsp="  a   b  "
nsy=$nsp
check nsp1 "[$nsy]" "[  a   b  ]"
nsr=$( echo "  cs  " )
check nsp2 "[$nsr]" "[  cs  ]"
nsa=("  p q  " r)
nsi=0
nsv=${nsa[$nsi]}
check nsp3 "[$nsv]" "[  p q  ]"
nsb=([0]=$nsp)
check nsp4 "[${nsb[0]}]" "[  a   b  ]"
nsc=($nsp)
check nsp5 "${#nsc[@]}" 2
scq=Q
check sca1 "${scq[@]}|${scq[0]}|${#scq[@]}|${!scq[@]}" "Q|Q|1|0"
sce=(p q)
check scd1 "${sce[9]-dflt}|${sce[0]-dflt}|${sce[9]+plus}|${sce[0]+plus}" "dflt|p||plus"
declare -A scf=([k]=v)
check scd2 "${scf[zz]-dd}|${scf[k]-dd}" "dd|v"
scg=()
check scd3 "${scg[@]-D}" D
ssi=(p q)
( ssi[0]=Z; ssi+=(r) )
check ssa1 "[${ssi[*]}]" "[p q]"
declare -A ssj=([k]=v)
( ssj[k]=W; ssj[n]=X )
check ssa2 "[${ssj[k]}][${ssj[n]-unset}]" "[v][unset]"
ssk=(1 2)
ssl=$( ssk[0]=9; echo "${ssk[0]}" )
check ssa3 "$ssl ${ssk[0]}" "9 1"
ssn=5
( declare -i ssn; ssn=2+3 )
ssn=2+3
check ssa4 "$ssn" "2+3"
dpp=hello
check dpa1 "$(declare -p dpp)" 'declare -- dpp="hello"'
declare -i dpq=7
check dpa2 "$(declare -p dpq)" 'declare -i dpq="7"'
dpr=(p q)
check dpa3 "$(declare -p dpr)" 'declare -a dpr=([0]="p" [1]="q")'
declare -A dps=([k]=v)
check dpa4 "$(declare -p dps)" 'declare -A dps=([k]="v" )'
dpt=$'a\tb'
check dpa5 "$(declare -p dpt)" "declare -- dpt=\$'a\\tb'"
declare -p nosuchname 2>/dev/null
check dpa6 "$?" 1
declare -ai dpu=(1 2)
check dpa7 "$(declare -p dpu)" 'declare -ai dpu=([0]="1" [1]="2")'
declare -A dpv=(["k one"]=y)
check dpa8 "$(declare -p dpv)" 'declare -A dpv=(["k one"]="y" )'

# ----- the @-transform family, a computed command name, echo's flags -----
# tests/bash-test-full.sh SECTION 34 is the full set and carries the reasoning;
# these are the rows that must stay green in the MATRIX and in the native binary,
# which the ratchet file does not reach.
xa=(pa "p b")
declare -A xm=([k]=v)
xs="hi there"
check xtf1 "${xa[*]@A}" 'declare -a xa=([0]="pa" [1]="p b")'
check xtf2 "$(printf '<%s>' "${xa[@]@A}")" '<declare><-a><xa=([0]="pa" [1]="p b")>'
check xtf3 "${xm[*]@A}" 'declare -A xm=([k]="v" )'
check xtf4 "${xs@A}" "xs='hi there'"
check xtf5 "${xa[*]@a}" 'a a'
check xtf6 "${xa[*]@K}" '0 "pa" 1 "p b"'
check xtf7 "$(printf '<%s>' "${xa[@]@k}")" '<0><pa><1><p b>'
check xtf8 "${nosuchname@Q}" ''
check xtf9 "${nosuchname@A}" ''
xp='\[foo\]\$'
check xtf10 "${xp@P}" 'foo$'
# ${!r} reaches the SPECIAL parameters, not just the named slots
set -- one two
xz=0
check xtf11 "${!xz}" "$0"
xz='#'
check xtf12 "${!xz}" 2
# a command NAME out of an expansion is field-split like any other word
xc="echo hi"
check xtf13 "$($xc)" hi
xe2=""
check xtf14 "$($xe2 echo late)" late
xfn2() { echo "fn:$1"; }
xg2="xfn2 zz"
check xtf15 "$($xg2)" 'fn:zz'
# ...and echo reads its flags from whatever the expansion produced
xo="-n"
check xtf16 "$(echo $xo x; echo '|')" 'x|'
xna=(-n hi)
check xtf17 "$(echo "${xna[@]}"; echo '|')" 'hi|'

# ----- an expansion error abandons the rest of its command LINE with status 1 -----
xd0=0
xab=0
{ xab=1; echo "r=$(( 7 / xd0 ))"; xab=2; } 2>/dev/null; xab=3
check eab1 "$?/$xab" "1/1"
xabf() { xab=4; echo "r=$(( 7 / xd0 ))"; xab=5; }
{ xabf; xab=6; } 2>/dev/null
check eab2 "$?/$xab" "1/4"
xs0=abc
{ xab=7; : "${xs0:1:-3}"; xab=8; } 2>/dev/null
check eab3 "$?/$xab" "1/7"
# ...while a command that evaluates arithmetic itself only fails
{ (( 5 / xd0 )); xab="$?"; let "xq = 1 / xd0"; xab="$xab$?"; } 2>/dev/null
check eab4 "$?/$xab" "0/11"

# ----- a long run of command substitutions costs only their results -----
xcs=0
for (( xci = 0; xci < 600; xci++ )); do xcv=$(echo "$xci"); xcs=$(( xcs + xcv )); done
check cs600 "$xcs" 179700

# ----- done -----
echo "features: $checks checks, $fails failures"
exit $fails
