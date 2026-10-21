#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 2
# of 4, sections 33 to 46. The split is explained in tests/bash-test-full.sh.
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

# ===== SECTION 33: word-splitting boundaries, read's remainder, $0, and a
#       pattern operator on a subscripted expansion =====
# Every assertion here was settled against bash 5.3.15 before it was written.
s33() {
  # --- a DELIMITER AT THE EDGE of an unquoted expansion separates it from the
  # literal text beside it: `w:[` `a` `b` `]` is four words, not one.
  local x="  a   b  "
  check wb1 "$(printf '<%s>' w:[$x])" '<w:[><a><b><]>'
  check wb2 "$(printf '<%s>' $x)" '<a><b>'
  check wb3 "$(printf '<%s>' pre$x)" '<pre><a><b>'
  check wb4 "$(printf '<%s>' $x post)" '<a><b><post>'
  local y="a " z=" b"
  check wb5 "$(printf '<%s>' $y$z)" '<a><b>'
  check wb6 "$(printf '<%s>' ""$x)" '<><a><b>'
  check wb7 "$(printf '<%s>' $x"")" '<a><b><>'
  local ws33="   "
  check wb8 "$(printf '<%s>' w:[$ws33])" '<w:[><]>'
  local e33=""
  check wb9 "$(printf '<%s>' w:[$e33])" '<w:[]>'
  local h33="a  b"
  check wb10 "$(printf '<%s>' pre$h33 post)" '<prea><b><post>'
  # a LEADING non-whitespace delimiter already makes an empty first field, so it
  # must not separate twice: three words here, not four
  local c33=":a:"
  IFS=: check wb11 "$(IFS=: ; printf '<%s>' w:[$c33])" '<w:[><a><]>'
  IFS=$' \t\n'
  # a QUOTED list inside a word joins at its boundaries and splits nowhere
  local arr33=(one two)
  check wb12 "$(printf '<%s>' x"${arr33[@]}"y)" '<xone><twoy>'
  check wb13 "$(printf '<%s>' x${arr33[@]}y)" '<xone><twoy>'
  set -- p q
  check wb14 "$(printf '<%s>' a"$@"b)" '<ap><qb>'
  check wb15 "$(printf '<%s>' "$@"z)" '<p><qz>'

  # --- printf REUSES ITS FORMAT until the operands run out, and an unquoted
  # operand is field-split first, so the number of passes is a run-time number.
  check pf1 "$(printf '%s|' $x)" 'a|b|'
  check pf2 "$(printf '[%s]' $x)" '[a][b]'
  check pf3 "$(printf '%s|' "$x")" '  a   b  |'
  local y33="p q r"
  check pf4 "$(printf '%s-%s|' $y33)" 'p-q|r-|'
  check pf5 "$(printf 'noconv' $y33)" 'noconv'
  printf -v V33 '%s|' $y33
  check pf6 "$V33" 'p|q|r|'
  check pf7 "$(printf '%s|' $e33)" '|'

  # --- what the LAST name of a `read` gets: the single remaining field, or the
  # RAW remainder less trailing IFS WHITESPACE.
  local a33 b33 c34
  read -r a33 b33 <<< "  x   y   z  "
  check rd1 "[$a33][$b33]" '[x][y   z]'
  read -r a33 <<< "  hd  "
  check rd2 "[$a33]" '[hd]'
  IFS=: read -r a33 b33 <<< "p::q:r:"
  check rd3 "[$a33][$b33]" '[p][:q:r:]'
  IFS=: read -r a33 <<< "p:q:"
  check rd4 "[$a33]" '[p:q:]'
  IFS=: read -r a33 b33 <<< ":x:"
  check rd5 "[$a33][$b33]" '[][x]'
  IFS=: read -r a33 b33 <<< "p:q:"
  check rd6 "[$a33][$b33]" '[p][q]'
  IFS=: read -r a33 b33 <<< ":a:b:"
  check rd7 "[$a33][$b33]" '[][a:b:]'
  IFS=: read -r a33 b33 <<< "a::: "
  check rd8 "[$a33][$b33]" '[a][:: ]'
  read -r a33 b33 c34 <<< "1 2"
  check rd9 "[$a33][$b33][$c34]" '[1][2][]'
  # no names at all: the line unmodified, leading blanks included
  read -r <<< "  hello  world  "
  check rd10 "[$REPLY]" '[  hello  world  ]'
  REPLY=keep
  read -r </dev/null
  check rd11 "[$REPLY]" '[]'

  # --- a PATTERN OPERATOR on a subscripted expansion, applied per ELEMENT
  local A33=(px1 px2 axe)
  check pa1 "$(printf '<%s>' "${A33[@]/x/Y}")" '<pY1><pY2><aYe>'
  check pa2 "${A33[*]/x/Y}" 'pY1 pY2 aYe'
  check pa3 "${A33[1]/x/Y}" 'pY2'
  check pa4 "$(printf '<%s>' "${A33[@]#p}")" '<x1><x2><axe>'
  check pa5 "${A33[1]#p}" 'x2'
  check pa6 "${A33[2]%e}" 'ax'
  check pa7 "${A33[*]##*x}" '1 2 e'
  check pa8 "${A33[*]^^}" 'PX1 PX2 AXE'
  check pa9 "${A33[1]^^}" 'PX2'
  check pa10 "${A33[*]/#p/Q}" 'Qx1 Qx2 axe'
  check pa11 "${A33[*]/%e/Q}" 'px1 px2 axQ'
  check pa12 "[${A33[9]#p}]" '[]'
  # per ELEMENT and only then joined - the join-first answer would be "Qa ba"
  local B33=(aa ba)
  check pa13 "${B33[*]/a/Q}" 'Qa bQ'
  local -A M33=([k1]=pAx)
  check pa14 "${M33[k1],,}" 'pax'
  # ...and the same walk for the @-transforms (no field-list marker bytes may
  # leak into the output)
  local C33=(pa pb)
  check pa15 "$(printf '<%s>' "${C33[@]@Q}")" "<'pa'><'pb'>"
  check pa16 "${C33[*]@U}" 'PA PB'
  check pa17 "${C33[1]@Q}" "'pb'"

  # --- $0 is the SCRIPT, not the engine. The --full runner copies this file to a
  # temporary name, so the assertion is on what $0 must NOT be: the literal
  # "bash" or the GRAMMAR's path.
  case "$0" in
    ""|bash|*.abnf) check zr1 "$0" "the script's own path" ;;
    *) check zr1 ok ok ;;
  esac
}

# ===== SECTION 34: the @-transform family, a command name out of an expansion,
#       and echo's flags out of an expansion =====
# Every assertion here was settled against bash 5.3.15 before it was written.
#
# Two shapes are load-bearing and easy to get wrong when editing this:
#   * `"${a[@]@A}"` is THREE WORDS in bash, so it cannot be the second argument
#     of check - the [*] form is the joined string and printf '<%s>' is the
#     field view.
#   * @A of a LOCAL prints `declare x=...` in bash 5.3 (it names the scope), so
#     the plain-assignment form is asserted on a GLOBAL, exactly as pex10 does.
#
# STILL NOT ASSERTED, and each is measured rather than assumed:
#   * `${v@P}` of a value holding `$name`: bash runs the decoded prompt through
#     a full word expansion and answers the variable's value. The compiler has
#     no expander at run time, so modelling it in one half only would trade a
#     divergence from bash for a divergence between the halves. The BACKSLASH
#     half of that pass is modelled (xe3, xe4).
#   * `\t` in a prompt is the TIME, which is not reproducible.
#   * a computed command name that resolves to `printf`, `[`, `read`, `eval`,
#     `set`, `trap`, `unset`, `declare` or a control-flow builtin: those are
#     compiled from their argument list at compile time, so the compiled half
#     reaches the command-not-found fallback where the interpreter runs them.
#     Recorded at emitComputedCommand in languages/bash-to-llvm-ir.abnf.
#   * `declare -ul x` / `declare -lu x`, which CANCEL in bash (it prints
#     `declare --`) and leave the later attribute on here.
s34helper() { echo "fn:$1"; }
s34() {
  local -a A34=(pa "p b")
  local -A M34=([k]=v)
  S34g="hi there"   # a GLOBAL: @A of a LOCAL prints `declare x=...` in bash 5.3
  declare -i N34=7
  declare -u U34=abc

  check xa1 "${A34[*]@A}" 'declare -a A34=([0]="pa" [1]="p b")'
  check xa2 "$(printf '<%s>' "${A34[@]@A}")" '<declare><-a><A34=([0]="pa" [1]="p b")>'
  check xa3 "${M34[*]@A}" 'declare -A M34=([k]="v" )'
  check xa4 "${A34[1]@A}" "declare -a A34='p b'"
  check xa5 "${S34g@A}" "S34g='hi there'"
  check xa6 "${N34@A}" "declare -i N34='7'"
  check xa7 "${U34@A}" "declare -u U34='ABC'"
  check xa8 "${nosuch34@A}" ""
  check xa9 "${A34[9]@A}" 'declare -a A34'
  # UNQUOTED, the field list is split again: "p b" inside the value too.
  check xa10 "$(printf '<%s>' ${A34[@]@A})" '<declare><-a><A34=([0]="pa"><[1]="p><b")>'
  # bash splits `declare -a A34=` on IFS and keeps the value atomic: one field
  # under IFS=:, two under IFS==, and [*] is the space-joined text under any IFS.
  local ifs34="$IFS"
  IFS=:
  check xa11 "$(printf '<%s>' "${A34[@]@A}")" '<declare -a A34=([0]="pa" [1]="p b")>'
  check xa12 "${A34[*]@A}" 'declare -a A34=([0]="pa" [1]="p b")'
  IFS='='
  check xa13 "$(printf '<%s>' "${A34[@]@A}")" '<declare -a A34><([0]="pa" [1]="p b")>'
  IFS="$ifs34"

  check xb1 "${A34[*]@a}" 'a a'
  check xb2 "${A34@a}" 'a'
  check xb3 "${N34@a}" 'i'
  check xb4 "${S34g@a}" ''
  check xb5 "$(printf '<%s>' "${A34[@]@a}")" '<a><a>'

  check xc1 "${A34[*]@K}" '0 "pa" 1 "p b"'
  check xc2 "$(printf '<%s>' "${A34[@]@k}")" '<0><pa><1><p b>'
  check xc3 "${A34[1]@K}" "'p b'"
  check xc4 "${M34[*]@K}" 'k "v" '

  local E34=""
  check xd1 "${E34@Q}" "''"
  check xd2 "${nosuch34@Q}" ''
  check xd3 "${nosuch34@K}" ''
  check xd4 "${A34[9]@Q}" ''

  local P34='x\ny'
  check xe1 "${P34@P}" "$(printf 'x\ny')"
  local P35='\[foo\]\$'
  check xe2 "${P35@P}" 'foo$'
  local P36='\\$'
  check xe3 "${P36@P}" '$'
  local P37='q\"w'
  check xe4 "${P37@P}" 'q"w'
  local -a PA34=('a\nb' 'c\nd')
  check xe5 "$(printf '<%s>' "${PA34[@]@P}")" "$(printf '<a\nb><c\nd>')"

  set -- one two
  local Z34=0
  check xf1 "${!Z34}" "$0"
  Z34=1;  check xf2 "${!Z34}" one
  Z34='#'; check xf3 "${!Z34}" 2
  Z34='*'; check xf4 "${!Z34}" 'one two'
  Z34=S34g; check xf5 "${!Z34@Q}" "'hi there'"
  Z34=S34g; check xf6 "${!Z34@A}" "S34g='hi there'"

  local C34="echo hi"
  check xg1 "$($C34)" hi
  local G34=""
  check xg2 "$($G34 echo late)" late
  $G34
  check xg3 "$?" 0
  local H34="s34helper zz"
  check xg4 "$($H34)" 'fn:zz'
  local T34=true
  if $T34; then check xg5 ok ok; else check xg5 no ok; fi
  local FA34=(echo aa bb)
  check xg6 "$("${FA34[@]}")" 'aa bb'

  local O34="-n"
  check xh1 "$(echo $O34 x; echo '|')" 'x|'
  local O35="-e"
  check xh2 "$(echo $O35 'a\tb')" "$(printf 'a\tb')"
  local NA34=(-n hi)
  check xh3 "$(echo "${NA34[@]}"; echo '|')" 'hi|'
  check xh4 "$(echo $O34)" ''

  declare -ai AI34=(1 2)
  check xi1 "$(declare -p AI34)" 'declare -ai AI34=([0]="1" [1]="2")'
  check xi2 "${AI34[*]@A}" 'declare -ai AI34=([0]="1" [1]="2")'
  declare -iu IU34=5
  check xi3 "$(declare -p IU34)" 'declare -iu IU34="5"'
  declare -A KQ34=(["k one"]=y)
  check xi4 "$(declare -p KQ34)" 'declare -A KQ34=(["k one"]="y" )'
  check xi5 "${KQ34[*]@A}" 'declare -A KQ34=(["k one"]="y" )'
}

# ===== SECTION 35: the operator forms of $@ and $*, ${!prefix@}, and an empty "$@" =====
# $@ and $* with an operator are the positional parameters as a LIST, like
# ${a[@]} / ${a[*]}: "${@:2}" is the parameters from $2 (the offset counts from $0),
# a pattern operator applies to each parameter before the join. The word of
# ${x:-w} keeps its fields, and under a null IFS an unquoted $* in it stays fields.
# "$@" with no parameters is no word at all. Every expected value is bash 5.3's.
s35() {
  set -- "a b" c "" d
  check pp1 "$(printf '<%s>' "${@:2}")" '<c><><d>'
  check pp2 "$(printf '<%s>' "${*:2}"),$(printf '<%s>' "${@:1:2}")" '<c  d>,<a b><c>'
  check pp3 "$(printf '<%s>' "${@@Q}")" "<'a b'><'c'><''><'d'>"
  check pp4 "${*@Q}" "'a b' 'c' '' 'd'"
  check pp5 "$(printf '<%s>' "${@:-x}")" '<a b><c><><d>'
  check pp6 "${*/c/C}|${*#c}" 'a b C  d|a b   d'
  check pp7 "$(printf '<%s>' "${@#a}" "${@%b}")" '< b><c><><d><a ><c><><d>'
  check pp8 "$(printf '<%s>' "${@: -1}" "${@: -9}")" '<d>'
  local Zq35=1 Zw35=2
  declare -A Zh35=([k]=1)
  check pp9 "$(printf '<%s>' "${!Zq35@}" "${!Zw*}")" '<Zq35><Zw35>'
  check pp10 "$(printf '<%s>' "${!Zh*}")" '<Zh35>'
  local x35=""
  check pp11 "$(printf '<%s>' "${x35:-$@}")" '<a b><c><><d>'
  check pp12 "$(IFS=; printf '<%s>' ${x35:-$*} ${x35:-pre$*post})" '<a b><c><d><prea b><c><dpost>'
  check pp13 "$(IFS=; printf '<%s>' "${x35:-$*}")" '<a bcd>'
  local A35=(p q r)
  check pp14 "$(printf '<%s>' "${A35[@]: -5}" x "${x35: -3}")" '<x><>'
  set --
  check pp15 "$(printf '<%s>' x "$@" y "${@:1}" "${@+set}" "${*-un}")" '<x><y><un>'
}

# ===== SECTION 36: @A/@a/@K/@k on $@ and $*, a ${x-w} word's own quoting, `e=()` =====
# ${@@A} is the `set -- 'a' 'b c'` that would recreate the parameters, a FIELD LIST
# split like ${a[@]@A}'s `declare -a a=`; @K and @k quote each parameter as @Q
# does and @a is each one's (empty) attributes. The word of ${x:-w} is expanded with
# its OWN quoting when the ${...} stands unquoted: its quoted parts are not split
# and the fields it makes are not split again. An array assigned `()` is set.
# Every expected value is bash 5.3's.
s36() {
  set -- a "b c"
  check xa1 "$(printf '<%s>' "${@@A}")" "<set><--><'a'><'b c'>"
  check xa2 "$(printf '<%s>' "${*@A}" ${*@A})" "<set -- 'a' 'b c'><set><--><'a'><'b><c'>"
  check xa3 "$(IFS=:; printf '<%s>' "${*@A}" ${@@A})" "<set -- 'a':'b c'><set -- 'a'><'b c'>"
  check xa4 "$(printf '<%s>' "${@@K}" "${@@k}" "${*@K}")" "<'a'><'b c'><'a'><'b c'><'a' 'b c'>"
  check xa5 "$(printf '<%s>' "${@@a}" "${*@a}")" '<><>< >'
  check xa6 "$(set --; printf '<%s>' x "${@@A}" "${*@A}" "${@@K}" y)" '<x><><y>'
  set -- "a b" c "" d
  local x36="" y36="p q"
  check dw1 "$(printf '<%s>' ${x36:-"$@"})" '<a b><c><><d>'
  check dw2 "$(IFS=; printf '<%s>' ${x36:-"$@"} ${x36:-$@})" '<a b><c><><d><a b><c><d>'
  check dw3 "$(printf '<%s>' ${x36:-"$y36"} ${x36:-$y36} w${x36:-"$@"}v)" '<p q><p><q><wa b><c><><dv>'
  check dw4 "$(printf '<%s>' ${y36:+"$@"} ${x36:-a" "b} ${x36:-a b})" '<a b><c><><d><a b><a><b>'
  check dw5 "$(f36() { echo $#; }; f36 ${x36:-""}; f36 $x36"$@"$x36)" '1
4'
  local e36a=() e36d=(1)
  declare -a e36b
  unset 'e36d[0]'
  check ea1 "${!e36*}" 'e36a e36d'
}


# ===== SECTION 37: 64-bit arithmetic =====
# bash's integers are intmax_t: 64 bits, wrapping, with a shift count taken modulo 64
# (the shift instruction's own rule on arm64 and x86-64). Operands come out of an array so
# no grammar can fold them. Every expected value is bash 5.3's.
s37() {
  local v=(10 3000000000 65536 -2147483648 2147483647 32 33 -1 -9223372036854775808)
  local t=${v[0]} big=${v[1]} h=${v[2]} m=${v[3]} x=${v[4]} z=${v[8]}
  check w64a "$(( t ** t )) $(( big + 0 )) $(( h * h ))" "10000000000 3000000000 4294967296"
  check w64b "[$(( m ))] [$(( x + 1 ))] [$(( 1 << 31 ))] [$(( m * 2 ))]" "[-2147483648] [2147483648] [2147483648] [-4294967296]"
  local s r=""
  for s in ${v[5]} ${v[6]} ${v[7]}; do r="$r $(( 1 << s )) $(( -8 >> s ))"; done
  check w64c "$r" " 4294967296 -1 8589934592 -1 -9223372036854775808 -1"
  local w="-9223372036854775808 7766279631452241919 -9223372036854775808 0 9223372036854775807"
  check w64d "$(( 9223372036854775807 + 1 )) $(( 99999999999999999999 )) $(( z / -1 )) $(( z % -1 )) $(( z - 1 ))" "$w"
  w="1099511627776 ffffffffffffffff 1777777777777777777777 18446744073709551615 B2D05E00"
  check w64e "$(printf '%d %x %o %u %X' $(( 1 << 40 )) -1 -1 -1 "$big")" "$w"
  w="-6289078614652622815 -9223372036854775808 0 -8446744073709551616"
  check w64f "$(( 3 ** 40 )) $(( 2 ** 63 )) $(( 2 ** 64 )) $(( t ** 19 ))" "$w"
  w="-3000000001 33558016 3000000001 -3000000001 -1"
  check w64g "$(( ~big )) $(( big & 0xF0F0F0F0F )) $(( big | 1 )) $(( big ^ -1 )) $(( 0xffffffffffffffff ))" "$w"
  local y=12345678901
  w="37037036703 1763668414 901 -1763668414 -901"
  check w64h "$(( y * 3 )) $(( y / 7 )) $(( y % 1000 )) $(( -y / 7 )) $(( -y % 1000 ))" "$w"
  let q=big*2
  declare -i di=$big
  di+=$big
  local i o=""
  for (( i = 4294967294; i < 4294967297; i++ )); do o="$o $i"; done
  check w64i "$q $di$o" "6000000000 6000000000 4294967294 4294967295 4294967296"
  check w64j "$(( big > 1 )) $(( -big < 1 )) $(( big == 3000000000 )) $(( 64#zzzzzzzzzzzz ))" "1 1 1 4099276460824344803"
  [ "$big" -gt 2147483647 ]; local rt=$?
  check w64k "$rt" 0
}

# ===== SECTION 38: what arithmetic evaluates, and when =====
# && and || skip an operand the left side decided, and ?: evaluates one arm. A variable's
# value is itself an expression (x=010 is 8, r=a reads a), and an expression holding an
# expansion is expanded FIRST and parsed after, so `$e * 2` with e="1 + 2" is 1 + 2 * 2.
# Every expected value is bash 5.3's.
s38() {
  local z=0 k=1
  check sc1 "$(( z && (z = 9) )) z=$z" "0 z=0"
  check sc2 "$(( 1 || (z = 9) )) z=$z" "1 z=0"
  check sc3 "$(( k == 1 ? k++ : k-- )) k=$k" "1 k=2"
  k=0
  check sc4 "$(( k == 1 ? k++ : k-- )) k=$k" "0 k=-1"
  check sc5 "$(( 0 && 1/0 )) $(( 1 || 1/0 )) $(( 0 ? 1/0 : 5 )) $(( 1 ? 2 : 3 , 4 ))" "0 1 5 4"
  local r=a a=7 e="1 + 2"
  check ev1 "$(( r + 1 ))" 8
  check ev2 "$(( e * 2 )) $(( $e * 2 )) $(( ${e} * 2 ))" "6 5 5"
  (( $e * 2 == 5 )); check ev3 "$?" 0
  local x=010 y="1+2" w="2*3"
  (( y++ )); (( w += 1 ))
  check ev4 "$(( x )) $y $w" "8 4 7"
  local v1=5 v2=v1 v3=v2 asg="t38=4"
  check ev5 "$(( "3" + 4 )) $(( $(echo 3) * 2 )) $(( v3 * 2 )) $(( asg )) $t38" "7 6 10 4 4"
  local n=5
  check ev6 "$(( n$((1+1)) )) $(( 1 + (2 ? 3 : 4) * 2 ))" "0 7"
}

# ===== SECTION 39: indexed subscripts are arithmetic; sparse arrays =====
# An indexed array's subscript is an arithmetic expression, evaluated once, and a negative
# one counts back from one past the highest index; NAME[SUB] is an operand and an assignment
# target inside arithmetic too. An indexed array lists in INDEX order, an unindexed element
# of a literal takes the index after the element before it, and ${a[@]:off:len} counts
# indices. Every expected value is bash 5.3's.
s39() {
  local a=(a0 a1 a2 a3 a4) i=2
  check ix1 "${a[i]} ${a[i+1]} ${a[$i+1]}" "a2 a3 a3"
  a[i+2]=W
  check ix2 "${a[*]}" "a0 a1 a2 a3 W"
  local nums=(3 4 5) m=(1 2 3) p=1
  (( m[p]++ ))
  check ix3 "sum=$(( nums[0] + nums[2] )) ${m[*]}" "sum=8 1 3 3"
  local b=(p q r s)
  check ix4 "[${b[-1]}] [${b[-2]}]" "[s] [r]"
  b[-1]=S
  check ix5 "${!b[*]} = ${b[*]}" "0 1 2 3 = p q r S"
  local c=(x y)
  c[i++]=Z
  check ix6 "i=$i ${!c[*]} ${c[*]} [${c[i--]}] i=$i" "i=3 0 1 2 x y Z [] i=2"
  local -A h
  h[foo]=3
  h[i+1]=7
  check ix7 "$(( h[foo] + 1 )) ${!h[*]}" "4 foo i+1"
  local f=(5 6 7) g=(5 6 7) k=(9 8 7) q=() s=(1 2 3) x=5
  unset 'f[-1]'
  [[ -v g[-1] ]]; local vr=$?
  (( q[3] = 4, q[1] += 2 ))
  check ix8 "${!f[*]} $vr ${#k[-1]} ${k[*]: -1} ${!q[*]} ${q[*]}" "0 1 0 1 7 1 3 2 4"
  check ix9 "$(( s[1] * s[2] )) $(( s )) $(( x[0] + 1 ))" "6 1 6"
  local e=() sp=()
  e[5]=x
  e[2]=y
  check sp1 "${!e[*]} = ${e[*]}" "2 5 = y x"
  local l1=([3]=c [1]=a b) l2=([2]=x [2]=y z) l3=(1 2 3)
  l3+=([7]=q r)
  check sp2 "${!l1[*]} = ${l1[*]} / ${!l2[*]} = ${l2[*]} / ${!l3[*]} = ${l3[*]}" "1 2 3 = a b c / 2 3 = y z / 0 1 2 7 8 = 1 2 3 q r"
  sp[2]=x
  sp[5]=y
  sp[9]=z
  check sp3 "[${sp[*]:3}] [${sp[*]:1:1}] [${sp[*]: -4}] [${sp[*]:5:2}]" "[y z] [x] [z] [y z]"
  check sp7 "$(printf '<%s>' "${sp[@]:5}" "${sp[@]:3:1}")" "<y><z><y>"
  check sp4 "[${sp[*]:0:2}] [${sp[*]:9}] [${sp[*]:10}] [${sp[*]: -1}] [${sp[*]: -20}]" "[x y] [z] [] [z] []"
  local l4=(a b c) l5=() l6=() l7=(x y z)
  l4=([5]=x "${l4[@]}")
  l5[10]=a
  l5[3]=b
  l5+=(c)
  i=1
  l6=([010]=x [0x10]=y [1+1]=z [i+3]=w)
  unset 'l7[1]'
  l7+=(w)
  check sp5 "${!l4[*]} = ${l4[*]} / ${!l5[*]} = ${l5[*]}" "5 6 7 8 = x a b c / 3 10 11 = b a c"
  check sp6 "${!l6[*]} = ${l6[*]} / ${!l7[*]} = ${l7[*]}" "2 4 8 16 = z w x y / 0 2 3 = x z w"
}

# ===== SECTION 40: expansion errors =====
# ${v:?w} on an unset value ends the (sub)shell with status 1; an arithmetic or substring
# error in an EXPANSION abandons the command line - in a subshell, the subshell - with status
# 1, while (( )), let and for (( )) only fail. Each case runs in a command substitution so
# the abandoned line is its own. Every expected value is bash 5.3's.
s40() {
  local e40="" r
  r=$( (: ${u40:?custom}) 2>/dev/null; echo "a$?"; (: ${u40?}) 2>/dev/null; echo "b$?"; (: ${e40:?}) 2>/dev/null; echo "c$?"; (: ${e40?}) 2>/dev/null; echo "d$?" )
  check ex1 "$r" "a1
b1
c1
d0"
  r=$( { : ${u40:?gone}; echo never; } 2>/dev/null )
  check ex2 "$?[$r]" "1[]"
  r=$( { d=0; f40() { echo "r=$(( 7 / d ))"; echo after; }; f40; echo "st=$?"; } 2>/dev/null )
  check ex3 "$?[$r]" "1[]"
  r=$( { : $(( 2 ** -1 )); echo never; } 2>/dev/null )
  check ex4 "$?[$r]" "1[]"
  r=$( { s=abc; t=${s:1:-3}; echo never; } 2>/dev/null )
  check ex5 "$?[$r]" "1[]"
  r=$( { s=abc; echo "[${s: -5:-1}] [${s:5:-1}] [${s:2:-1}]"; } 2>/dev/null )
  check ex6 "$?$r" "0[] [] []"
  r=$( { a=(); a[-1]=x; echo never; } 2>/dev/null )
  check ex7 "$?[$r]" "1[]"
  r=$( { d=0; (( 5 / d )); echo -n "a$?"; let x=1/d; echo -n "b$?"; for (( i = 0; i < 1/d; i++ )); do :; done; echo -n "c$?"; (( 1 + )); echo -n "d$?"; } 2>/dev/null )
  check ex8 "$r" "a1b1c1d1"
  r=$( { n=08; echo "$(( 1 + 1 ))"; x=$(( n )); echo never; } 2>/dev/null )
  check ex9 "$?[$r]" "1[2]"
}

# ===== SECTION 41: [[ A -eq B ]] operands are arithmetic =====
# Inside [[ ]] the numeric tests evaluate both sides as arithmetic expressions (octal,
# names, operators included); test / [ still compare decimal integers. bash 5.3's answers.
s41() {
  local x=4 y="1 + 2" i=0 d=0 r=""
  [[ 2+3 -eq 5 ]]; r="$r$?"
  [[ 010 -eq 8 ]]; r="$r$?"
  [[ x -eq 4 ]]; r="$r$?"
  [[ $y -eq 3 ]]; r="$r$?"
  [[ "2 * 3" -eq 6 ]]; r="$r$?"
  check nt1 "$r" "00000"
  r=""
  [[ 1 -ne 2 && x+1 -gt x && 3 -lt 0x10 && 1 -le 1 && 3000000000 -ge 2147483648 ]]; r="$r$?"
  [ 010 -eq 8 ]; r="$r$?"
  [[ 1 -eq 1/d ]] 2>/dev/null; r="$r$?"
  [[ i++ -eq 0 ]]; r="$r$?$i"
  check nt2 "$r" "01101"
}

# ===== SECTION 42: return / exit modulo 256, case fall-through and status =====
# `return N` and `exit N` are N modulo 256; ";&" falls into the next clause's body without
# testing it, ";;&" tests the next clause; a case whose matched body is empty, or that
# matches nothing, answers 0. Every expected value is bash 5.3's.
s42() {
  local r="" s
  f42a() { return 300; }
  f42b() { return -1; }
  f42c() { return 4294967297; }
  f42a; r="$r $?"
  f42b; r="$r $?"
  f42c; r="$r $?"
  (exit 300); r="$r $?"
  (exit -2); r="$r $?"
  check rt1 "$r" " 44 255 1 44 254"
  r=""
  for s in a b c; do
    case "$s" in a) r="${r}A;" ;& b) r="${r}B;" ;; c) r="${r}C;" ;;& *) r="${r}S;" ;; esac
  done
  check cs1 "$r" "A;B;B;C;S;"
  r=""
  case a in a) r="${r}1" ;& b) r="${r}2" ;& c) r="${r}3" ;; d) r="${r}4" ;; esac
  case b in a) r="${r}5" ;;& b) r="${r}6" ;;& *) r="${r}7" ;;& c) r="${r}8" ;; esac
  check cs2 "$r" "12367"
  r=""
  false; case x in y) ;; esac; r="$r$?"
  false; case x in x) ;; esac; r="$r$?"
  case x in x) false ;; esac; r="$r$?"
  case x in x) false ;& y) ;; esac; r="$r$?"
  case x in x) ;& y) false ;; esac; r="$r$?"
  check cs3 "$r" "00101"
}

# ===== SECTION 43: ${N-w} / ${N+w} on positional parameters, ${!ref} to an element =====
# A positional parameter past $# is UNSET to the - and + operators; an indirection may land
# on NAME[SUB] (the element, or the joined list for *), and on an array name (element 0).
# A subscript of an associative array met at run time is literal. bash 5.3's answers.
s43() {
  f43() { echo "[${2-unset}] [${3:-dflt}] [${1+set}] [${1:+set2}] [${0+zero}] [${10-ten}]"; }
  check pp1 "$(f43) $(f43 one)" "[unset] [dflt] [] [] [zero] [ten] [unset] [dflt] [set] [set2] [zero] [ten]"
  g43() { echo "[${2-u}] [${2:-e}] [${2+s}] [${2:+t}] [${4-u}]"; }
  check pp2 "$(g43 a "" c)" "[] [e] [s] [] [u]"
  local arr=(x y z) r1='arr[1]' r2=arr r3='arr[*]' r4='arr[-1]' r5='arr[i+1]' r7='arr[7]' i=0
  check ir1 "[${!r1}] [${!r2}] [${!r3}] [${!r4}] [${!r5}]" "[y] [x] [x y z] [z] [y]"
  check ir2 "[${!r7-none}] [${!r1-none}] [${!r7:+p}] [${!r1:+p}]" "[none] [y] [] [p]"
  local -A m43=([foo]=F [bar]=7)
  local r6='m43[foo]' r8='m43[bar]' e43='m43[bar] * 2'
  check ir3 "[${!r6}] $(( ${!r8} + 1 )) $(( $e43 ))" "[F] 8 14"
}

# ===== SECTION 44: quoted pattern characters are literal =====
# In a case pattern, the pattern operand of ${v#p} ${v%p} ${v/p/r} ${v^^p} and the regex of
# [[ =~ ]], a quoted or backslash-escaped part matches itself; an unquoted expansion stays a
# pattern. Every expected value is bash 5.3's.
s44() {
  local s r="" p='?' q='*' rx='b+'
  for s in x '*' a.c abc; do
    case $s in '*') r="${r}star " ;; *) r="${r}other " ;; esac
    case $s in "a.c"|\*) r="${r}lit " ;; *) r="${r}no " ;; esac
  done
  check qp1 "$r" "other no star lit other lit other no "
  s='*'
  check qp2 "[${s#\*}] [${s%"$p"}] [${s%$p}] [${s/'*'/R}] [${s//"*"/R}] [${s#"$q"}] [${s#$q}]" "[] [*] [] [R] [R] [] [*]"
  s=x
  check qp3 "[${s/'*'/R}] [${s/"?"/R}] [${s/?/R}] [${s//[x]/R}] [${s//"[x]"/R}]" "[x] [x] [R] [R] [x]"
  s='a*b*c'
  check qp4 "[${s//\*/-}] [${s%%"*"*}] [${s#*"*"}]" "[a-b-c] [a] [b*c]"
  s=abc
  check qp5 "[${s^^"b"}] [${s^^'[ab]'}] [${s^^[ab]}]" "[aBc] [abc] [ABc]"
  r=""
  [[ abc =~ "a.c" ]]; r="$r$?"
  [[ abc =~ a"."c ]]; r="$r$?"
  [[ a.c =~ a"."c ]]; r="$r$?"
  [[ abc =~ a\.c ]]; r="$r$?"
  [[ 'a$' =~ a'$' ]]; r="$r$?"
  [[ abbb =~ a$rx ]]; r="$r$?"
  [[ abbb =~ a"$rx" ]]; r="$r$?"
  [[ 'ab+' =~ a"$rx" ]]; r="$r$?"
  check qp6 "$r" "11010010"
}

# ===== SECTION 45: ${v/p/r} - "&", an empty subject, extglob, characters =====
# An unquoted "&" in the replacement is the matched text (patsub_replacement, on by default
# in bash 5.2+); a quoted or escaped one is literal. An empty value is a subject, an unset
# one is not. The extended globs are patterns only under `shopt -s extglob` (and always in
# [[ == ]]). A pattern never splits a UTF-8 character. Every expected value is bash 5.3's.
s45() {
  local k=hello r='<&>' e="" s=aXa v=hé w=éàü
  check rp1 "[${k/l/<&>}] [${k//l/&&}] [${k/l/\&}] [${k/l/"&"}] [${k/l/'&'}] [${k/l*/(&)}]" "[he<l>lo] [hellllo] [he&lo] [he&lo] [he&lo] [he(llo)]"
  check rp2 "[${k/l/$r}] [${k/l/"$r"}] [${k//[el]/[&]}] [${k/#h/&&}] [${k/%o/&!}]" "[he<l>lo] [he<&>lo] [h[e][l][l]o] [hhello] [hello!]"
  r='a\&b'
  check rp3 "[${k/l/$r}] [${k/l/\\}] [${k/l/a\b}] [${k/l/"\&"}] [${k/l/\\&}]" "[hea&blo] [he\lo] [heablo] [he\&lo] [he\llo]"
  check rp4 "[${e/*/R}] [${e//*/R}] [${e/#/R}] [${e/%/R}] [${e/x/R}] [${nosuch45/*/R}]" "[R] [R] [R] [R] [] []"
  check eg1 "[${s#@(a|X)}] [${s//@(a|X)/-}] [${s/+(a)/-}]" "[aXa] [aXa] [aXa]"
  shopt -s extglob
  shopt -q extglob; local q45=$?
  check eg2 "[${s#@(a|X)}] [${s//@(a|X)/-}] [${s/+(a)/-}] $q45" "[Xa] [---] [-Xa] 0"
  shopt -u extglob
  [[ aXa == @(a|X)Xa ]]; local d45=$?
  shopt -q extglob
  check eg3 "[${s#@(a|X)}] $d45 $?" "[aXa] 0 1"
  # An EMPTY match counts: after one the character it sits on is copied and the search goes
  # on behind it. At the end of the subject only a pattern starting with "*" is tried, so an
  # empty subject is matched by *(a) but not by ?(a) or !(b).
  local g45=abac x45=xy a45=(xy ab "")
  shopt -s extglob
  check eg4 "[${g45//*(a)/Z}] [${x45//*(a)/Z}] [${x45/?(a)/Z}] [${e//!(b)/Z}] [${e//?(a)/Z}] [${e/#?(a)/Z}] [${e//*(a)/Z}] [${g45//*(a)/<&>}] [${a45[*]//*(a)/Z}]" "[ZZbZZc] [ZxZy] [Zxy] [] [] [] [Z] [<a><>b<a><>c] [ZxZy ZZb Z]"
  shopt -u extglob
  check u81 "[${v%?}] [${v#?}] [${v%%?}] [${v/%?/Z}] [${v/#?/Z}]" "[h] [é] [h] [hZ] [Zé]"
  check u82 "[${w%?}] [${w%??}] [${w#?}] [${w/?/-}] [${w//?/-}]" "[éà] [é] [àü] [-àü] [---]"
  # An astral character (a UTF-16 surrogate pair) is ONE character to every pattern element.
  local y="😀b😀c" z=a😀 m='[🎉-😀]' c45=""
  check u83 "[${y#?}] [${y#??}] [${y%?}] [${y%%?*}] [${y/?/x}] [${y//?/x}] [${y/%?/Z}] [${z#a?}] [${z/#*?/Z}]" "[b😀c] [😀c] [😀b😀] [] [xb😀c] [xxxx] [😀b😀Z] [] [Z]"
  case 😀 in ?) c45=1 ;; ??) c45=2 ;; *) c45=0 ;; esac
  case a😀 in a$m) c45="$c45 r" ;; esac
  case a😀 in a[!x]) c45="$c45 n" ;; esac
  case 😀x in \😀?) c45="$c45 e" ;; esac
  [[ 😀😀 == ?? ]] && c45="$c45 d"
  [[ 😀😀b == +(😀)? ]] && c45="$c45 g"
  [[ 😀😀b == !(b)b ]] && c45="$c45 x"
  check u84 "$c45 [${y^^[😀b]}] [${y//[😀]/-}]" "1 r n e d g x [😀B😀c] [-b-c]"
  # A POSIX class holds characters of every script, as the locale's iswctype classifies them:
  # a digit of any script is also xdigit, a titlecase letter is neither upper nor lower,
  # U+3000 is space but not blank, a combining mark is graph but not punct. A byte that
  # starts no character is in no class.
  local x85="é😀" y85="a١²中 ¡　€ÀÿƀǅＡ" r85=$'a\xe9b' c85=""
  check u85 "[${x85#[[:alpha:]]}] [${x85%[[:print:]]}] [${y85//[[:alpha:]]/A}] [${y85//[[:digit:]]/D}] [${y85//[[:upper:]]/U}] [${y85//[[:lower:]]/L}]" "[😀] [é] [A١²A ¡　€AAAAA] [aD²中 ¡　€ÀÿƀǅＡ] [a١²中 ¡　€UÿƀǅU] [L١²中 ¡　€ÀLLǅＡ]"
  case $'́' in [[:punct:]]) c85=p ;; [[:graph:]]) c85=g ;; esac
  [[ ǅ == [[:upper:]] || ǅ == [[:lower:]] ]] || c85="$c85 t"
  [[ $'\u0085' == [[:cntrl:]] ]] && c85="$c85 c"
  [[ $' ' == [[:blank:]] ]] || c85="$c85 b"
  [[ $'\U000e0001' == [[:cntrl:]] ]] && c85="$c85 f"
  [[ $'\U0010fffd' == [[:print:]] ]] && c85="$c85 u"
  [[ $'\xe9' == [[:alpha:]] ]] || c85="$c85 r"
  check u86 "[${y85//[[:punct:]]/P}] [${y85//[[:space:]]/S}] [${y85//[[:xdigit:]]/X}] [${y85//[[:word:]]/W}] [${y85//[![:graph:]]/N}] [${r85//[![:alpha:]]/N}] $c85" "[a١²中 P　€ÀÿƀǅＡ] [a١²中S¡S€ÀÿƀǅＡ] [XX²中 ¡　€ÀÿƀǅＡ] [WW²W ¡　€WWWWW] [a١²中N¡N€ÀÿƀǅＡ] [aNb] g t c b f u r"
  # A long subject: padding and every pattern operator stay linear in its length.
  local L45 Z45 T45 a b c d
  printf -v L45 '%*s' 5000
  printf -v Z45 '%05000d' 7
  T45=${L45// /ab}
  check rp5 "${#L45} ${#T45} ${T45:9996} ${#Z45} ${Z45//0/}" "5000 10000 abab 5000 7"
  L45=${L45}é
  a=${L45/%?/E} b=${L45#?} c=${L45%" é"} d=${Z45/#0/x}
  check rp6 "${#a} [${a: -2}] ${#b} ${#c} ${d:0:2} ${#d}" "5001 [ E] 5000 4999 x0 5000"
}

# ===== SECTION 46: printf flags, operands and escapes =====
# The + space # 0 - flags with widths and precisions (also from * operands), numeric operands
# read like strtoimax / strtoumax with base 0 or as 'C character codes (an invalid one is
# reported, its leading number used, status 1), escapes in the FORMAT, and %b's own escapes
# with \c ending all output. Every expected value is bash 5.3's.
s46() {
  local r x f
  r=$(printf '[%05d] [%#05x] [%+d] [% d] [%-+4d] [%d] [%d]' -7 255 5 5 3 "'A" "'a")
  check pf1 "$r" "[-0007] [0x0ff] [+5] [ 5] [+3  ] [65] [97]"
  r=$(printf '<%*d><%.*s><%-*s|>' 5 42 2 abcdef 4 x; printf -- '[%s]' dd)
  check pf2 "$r" "<   42><ab><x   |>[dd]"
  x=$(printf '\101\x41\0101|\q|\"|\\|\c|☺|after')
  check pf3 "${x@Q}" "\$'AA\\b1|\\\\q|\"|\\\\|\\\\c|☺|after'"
  r=$(printf '[%b] [%b] [%b] [%b]' '\0101' '\x41' '\101' 'a\qb'; printf '[%b]' 'x\cy' 'z'; printf '|')
  check pf4 "$r" '[A] [A] [A] [a\qb][x|'
  r=$(printf '[%d]' abc 2>/dev/null; echo "rc=$?"; printf '[%d]' "1+2" 2>/dev/null; echo "rc=$?")
  check pf5 "$r" "[0]rc=1
[1]rc=1"
  r=$(printf '[%d] [%d] [%d] [%i] [%x] [%o] [%u] [%X]' 0x1f 010 -0x10 " 12" 0x1f 010 -1 255)
  check pf6 "$r" "[31] [8] [-16] [12] [1f] [10] [18446744073709551615] [FF]"
  r=$(printf '[%5d] [%-5d] [%05.3d] [%.0d] [%.3d] [%#o] [%#x] [%#X] [%#.0o] [%+.0d]' 42 42 7 0 -7 8 0 255 0 0)
  check pf7 "$r" "[   42] [42   ] [  007] [] [-007] [010] [0] [0XFF] [0] [+]"
  r=$(printf '[% 5d] [%+5d] [%+05d] [% 05d] [%-05d] [%.3x] [%#.3o] [%8.3x]' 3 3 3 3 3 10 8 255)
  check pf8 "$r" "[    3] [   +3] [+0003] [ 0003] [3    ] [00a] [010] [     0ff]"
  r=$(printf '[%d]' 99999999999999999999 2>/dev/null; echo "rc=$?"; printf '[%x]' 0xffffffffffffffff; echo "rc=$?")
  check pf9 "$r" "[9223372036854775807]rc=1
[ffffffffffffffff]rc=0"
  r=$(printf '[%y]' 1 2>/dev/null; echo "rc=$?"; printf '[%q] [%q] [%ld] [%hd]' "a b" "" 5 6)
  check pf10 "$r" "[rc=1
[a\\ b] [''] [5] [6]"
  f='%s=%d;'
  r=$(printf "$f" a 1 b 2; printf '%.*d|%*s|' -1 5 -4 ab)
  check pf11 "$r" "a=1;b=2;5|ab  |"
}
# ===== END SECTIONS =====

s33   # SECTION-CALL 33
s34   # SECTION-CALL 34
s35   # SECTION-CALL 35
s36   # SECTION-CALL 36
s37   # SECTION-CALL 37
s38   # SECTION-CALL 38
s39   # SECTION-CALL 39
s40   # SECTION-CALL 40
s41   # SECTION-CALL 41
s42   # SECTION-CALL 42
s43   # SECTION-CALL 43
s44   # SECTION-CALL 44
s45   # SECTION-CALL 45
s46   # SECTION-CALL 46
echo "full: $checks checks, $fails failures"
exit $fails
