#!/usr/bin/env bash
# Fast feature-matrix test for the Bash interpreter (bash-interpreter.abnf) and the
# LLVM-IR compiler (bash-to-llvm-ir.abnf). Same protocol as every other
# tests/<lang>-test-features.*: every implemented construct is exercised with the
# SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times, recursion
# stays below depth 6. A failed check prints its id (so a diff pinpoints it) and the
# program ends with `exit $fails`; exit 0 and byte-identical output on all four legs
# (interpreter/compiler x goja/-frozen) mean everything passed.
#
# WHY THIS FILE EXISTS AND WHAT IT IS AIMED AT (docs/todo.md 4.1)
#
# bash was the ONLY native-full language with no feature file, and bash's layer 2 is
# not MetaJS at all - it is the C runtime languages/lib/bash-rt.c, ~2,950 lines,
# which only tests/bash-test-full.sh ever exercised. Both gates that can see layer 2
# (tests/clang-check.sh and tests/native-full.sh) ran that one file, so every
# bash-rt.c entry point had exactly ONE gate on it and a freshly fixed defect had
# nowhere cheap to get its assertion. This file is the second one, and it is written
# to hit the runtime's entry points on purpose: rt_strip / rt_replace / rt_case for
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
# NOT asserted: ${s^^[bn]} (a case op with a PATTERN operand). Real bash 5.3 says
# BaNaNa; both our engines ignore the pattern and answer BANANA. Recorded in the
# report for docs/todo.md - it is a bash-rt.c / grammar change, not a test change.

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
# NOT asserted: -f. languages/lib/bash-rt.c's rt_fskind is a hard-coded path
# whitelist (there is no stat() in the C floor), so `-f /dev/null` is TRUE here and
# false in real bash, and `-f` on any real regular file is false here. -e and -d are
# asserted only on paths the whitelist and the real filesystem agree about.
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
# NOT asserted: $(( 10#$num )). Real bash 5.3 answers 123 for 10#00123; both our
# engines abort with `command not found: 10#00123` - the base#digits form is only
# recognised for a LITERAL base prefix, not one built by expansion. Reported.
check str7 "$(( 2#1010 + 1 ))" 11
check str8 "$(( -0 ))" 0

# ----- done -----
echo "features: $checks checks, $fails failures"
exit $fails
