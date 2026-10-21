#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 3
# of 4, sections 47 to 60 and 62 on; a new section goes here. The split is explained in
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

# ===== SECTION 47: command substitution depth =====
# Command substitutions nest as deep as the recursion asks: each depth's capture buffer and
# variable snapshot are allocated once and reused. Every expected value is bash 5.3's.
s47() {
  d47() { if [ "$1" -le 0 ]; then echo 0; return; fi; echo $(( 1 + $(d47 $(( $1 - 1 ))) )); }
  check cd1 "$(d47 6) $(d47 8) $(d47 20)" "6 8 20"
  f47() { if [ "$1" -lt 2 ]; then echo "$1"; return; fi; echo $(( $(f47 $(( $1 - 1 ))) + $(f47 $(( $1 - 2 ))) )); }
  check cd2 "$(f47 9)" 34
}

# ===== SECTION 48: break / continue levels, a computed unset =====
# `break N` / `continue N` past the enclosing loops act on the outermost; outside a loop (a
# called function's body included) they only complain; a level below 1 leaves every loop with
# status 1; the level may be computed. unset of a name built by expansion. Every expected value
# is bash 5.3's.
s48() {
  local r="" i j x
  for i in 1 2; do for j in 1 2; do break 5; done; r="${r}never"; done
  r="$r rc=$?"
  for i in 1 2; do for j in 1 2; do r="$r $i$j"; continue 9; done; r="${r}never"; done
  check bk1 "$r" " rc=0 11 21"
  r=""
  f48() { break; }
  for i in 1 2 3; do r="$r f$i"; f48 2>/dev/null; done
  for i in 1 2; do x=2; for j in 1 2; do r="$r $i$j"; break $x; done; done
  for i in 1 2; do break 0 2>/dev/null; r="$r z$i"; done
  check bk2 "$r rc=$?" " f1 f2 f3 11 rc=1"
  local d=(5 6 7)
  i=1
  unset "d[$((i+1))]"
  check us1 "${!d[*]}" "0 1"
}

# ===== SECTION 49: line continuation, loop status, redirected assignments, echo -e =====
# A backslash-newline between two words is no word at all. A loop's status is its last body
# command's, 0 when the body never ran or was left by break / continue. An assignment can carry
# a redirection, which does not apply to its command substitution. echo -e reads octal only as
# \0NNN, keeps \" as written, accepts combined flags, and its \c ends all output. Two
# command substitutions in a row each keep their own text. Every expected value is bash 5.3's.
s49() {
  s49f() { echo "$#:[$1][$2][$3]"; }
  check lc1 "$(s49f a \
 b)" "2:[a][b][]"
  check lc2 "$(s49f a\
b c)" "2:[ab][c][]"
  local r="" i
  while (( 0 )); do :; done; r="$r $?"
  while true; do false; break; done; r="$r $?"
  false; until true; do :; done; r="$r $?"
  for i in 1 2; do false; continue; done; r="$r $?"
  false; for ((i = 0; i < 0; i++)); do :; done; r="$r $?"
  for ((i = 0; i < 2; i++)); do false; done; r="$r $?"
  false; for i in; do :; done; r="$r $?"
  false; if false; then :; fi; r="$r $?"
  check lst1 "$r" " 0 0 0 0 0 1 0 0"
  local a b
  a=$(echo x) 2>/dev/null
  b=$(echo y; false) >/dev/null
  check asr1 "$a $b $?" "x y 1"
  a=$(echo p)
  b=$(echo q)
  check csa1 "$a$b" "pq"
  check ece1 "$(echo -e 'a\101b|a\0101b|q\"q|h\x41\xZ')" 'a\101b|aAb|q\"q|hA\xZ'
  check ece2 "$(echo -e 'x\cy' z; echo -n w)" "xw"
  check ece3 "$(echo -en 'p\tq'; echo -nE '\t|')" "p	q\\t|"
  check ece4 "$(printf '%b' 'q\"q|\0101\101')" 'q\"q|AA'
}

# ===== SECTION 50: $'...' in ${...} words, ${!r} of a list, textual let / for (( )), bytes =====
# ${x@Q} spells ESC \E. A $'...' part of a ${...} word - default, pattern, replacement - is
# ANSI-C quoted, inside "..." too, and its "&" is literal. ${!r} with r='arr[@]' is one word
# per element. let "..." and the parts of for (( ; ; )) are expanded as TEXT before they are
# evaluated, and a part may be empty. \xHH / \NNN make BYTES: %c and %.Ns cut by byte, a
# lone byte counts as one character, and bytes that spell a character are that character.
# Every expected value is bash 5.3's.
s50() {
  local x=$'a\eb' k=hello u
  check aq1 "${x@Q}" "\$'a\\Eb'"
  check aq2 "${k/l/$'\x26'}|${k//l/$'\x26'}|${k#$'\x68'}|${u:-$'\x41b'}|${u-$'q\tq'}" "he&lo|he&&o|ello|Ab|q	q"
  local arr=(p 'q r' s) r='arr[@]' o
  o=$(printf '<%s>' "${!r}" ${!r})
  check ind1 "$o" "<p><q r><s><p><q><r><s>"
  local e='1 + 2' y i n=0
  let "y = $e * 2"
  for (( i = $e * 0; i < $e * 2; i += 3 )); do n=$((n + i)); done
  check txa1 "$y $n" "5 5"
  i=0
  for (( ; i < $e; )); do i=$((i + 1)); done
  for (( ; ; )); do break; done
  check txa2 "$i" "3"
  local z=$'\303\251' b=$'\xc3' c d
  c=$(printf '%c' é)
  d=$(printf '[%.1s|%.3s|%3.1s]' é aéb é)
  check byt1 "${#z} ${#b} ${#c} $z" "1 1 1 é"
  check byt2 "$c|$d" "$b|[$b|aé|  $b]"
  check byt3 "$(printf '%s' "$b"$'\xa9')" "é"
}

# ===== SECTION 51: numbers read by arithmetic and test =====
# A variable holding a number may be padded with blanks, newline included, on either side.
# A literal divisor divides like any other, a negative left side truncating toward zero.
# test reads its integer operands in decimal, one sign allowed. An assignment of one
# expansion keeps its value whole. Every expected value is bash 5.3's.
s51() {
  local x=$'\n5' y=$'\t-7 \n' n=-7 m=17
  check num1 "$(( x )) $(( x + 1 )) $(( y * 2 ))" "5 6 -14"
  (( m %= 5 ))
  check num2 "$(( n % 3 )) $(( n / 2 )) $(( n % 7 )) $(( 7 / 1 )) $(( n / 7 )) $m" "-1 -3 0 7 -1 2"
  local t=""
  [ -3 -lt +2 ] && t="${t}a"
  [ 40000 -gt 39999 ] && t="${t}b"
  [ 012 -eq 12 ] && t="${t}c"
  [ -0 -eq 0 ] && t="${t}d"
  [ 5 -lt -5 ] || t="${t}e"
  check num3 "$t" "abcde"
  set -- p 'q  r'
  local a1 a2 a3
  a1=$*
  a2=$@
  a3=$(( 2 * 3 ))$#
  check num4 "$a1|$a2|$a3" "p q  r|p q  r|62"
}

# ===== SECTION 52: let words and redirections, NUL bytes, loops inside loops =====
# let's arguments are words: redirections stand among them, an unquoted expansion is
# field-split into several arguments, every word is expanded before any is evaluated, and
# the first error stops the command. A NUL byte that printf or echo -e makes is written
# out, dropped by a command substitution, and ends a printf -v value and a $'...' string.
# Every expected value is bash 5.3's.
s52() {
  local e='1 + 2' x y z w p q r s v
  let x=1 2>/dev/null
  check let1 "$? $x" "0 1"
  let y=$e*2 z=7 2>/dev/null
  check let2 "$? $y $z" "1 1 "
  let w=4 >/dev/null p=$w+1 2>&1
  check let3 "$? $w $p" "0 4 1"
  let q=1 + r=2 2>/dev/null
  check let4 "$? $q $r" "1 1 "
  local f='s=3 v=0'
  let $f
  check let5 "$? $s $v" "1 3 0"
  local u=
  let $u 2>/dev/null
  check let6 "$?" "1"
  let $u s=9 || s=bad
  check let7 "$s" "9"
  local n1 n2 n3 n4 n5
  printf -v n1 'e\0f'
  printf -v n2 '[%3c]' ''
  printf -v n3 '%b|' 'z\0z'
  { n4=$(printf 'a\0b'; echo -e 'c\0d'); } 2>/dev/null
  n5=$'g\0h'x$'\x00'y
  check nul1 "$n1|$n2|$n3|$n4|$n5" "e|[  |z|abcd|gxy"
  local i j t=""
  for i in 1 2 3; do
    j=0
    while [ $j -lt 3 ]; do
      for k in a b; do
        [[ $k$j =~ ^b ]] && continue
        t="$t$i$j$k"
      done
      j=$(( j + 1 ))
    done
  done
  check lpl1 "$t $?" "10a11a12a20a21a22a30a31a32a 0"
}

# ===== SECTION 53: an arithmetic error abandons the expression, control bytes as data =====
# The first arithmetic error ends the whole expression: nothing after it is evaluated or
# stored, and an assignment whose value fails to expand is not made (the command is
# abandoned, which the EXIT trap of a command substitution observes). Bytes 0x01 and 0x02
# are ordinary data inside a field, and a command substitution that drops a NUL byte warns.
# Every expected value is bash 5.3's.
s53() {
  local z=0 v w w2 u=5 c=1 g=3 r='q1 = 6' q1 a=(1 2 3) n1 n2 o
  (( v = 1/z )) 2>/dev/null
  check aer1 "$? ${v:-unset}" "1 unset"
  (( w = 2, w2 = 1/z )) 2>/dev/null
  check aer2 "${w:-unset} ${w2:-unset}" "2 unset"
  (( u = 7 + 1/z, c++ )) 2>/dev/null
  check aer3 "$u $c" "5 1"
  (( c++, 1/z, c++ )) 2>/dev/null
  check aer4 "$c" "2"
  (( g = 1 ? 2/z : 4 )) 2>/dev/null
  check aer5 "$g" "3"
  (( 1/z, $r )) 2>/dev/null
  check aer6 "${q1:-unset}" "unset"
  (( a[1] = 1/z )) 2>/dev/null
  check aer7 "${a[*]}" "1 2 3"
  let "n1 = 4" "n2 = 1/z" 2>/dev/null
  check aer8 "$? $n1 ${n2:-unset}" "1 4 unset"
  { o=$( trap 'echo "k=$k"' EXIT; k=4; k=$(( 1/z )) ); } 2>/dev/null
  check aer9 "$? $o" "1 k=4"
  { o=$( trap 'echo "${#b[@]} ${b[*]}"' EXIT; b=(5 6); b=(1 $(( 2/z )) 3) ); } 2>/dev/null
  check aer10 "$o" "2 5 6"
  { o=$( trap 'echo "${b[*]}"' EXIT; b=(5 6); b[1]=$(( 2/z )) ); } 2>/dev/null
  check aer11 "$o" "5 6"
  { o=$( trap 'echo "$k"' EXIT; k=4; k+=$(( 2/z )) ); } 2>/dev/null
  check aer12 "$o" "4"
  { o=$( trap 'echo "${d-unset}"' EXIT; declare d=$(( 1/z )) ); } 2>/dev/null
  check aer13 "$? $o" "1 unset"
  o=$( { (( 1/z + 2/z )); } 2>&1 )
  o=${o//division by 0/@}
  check aer14 "${o//[^@]/}" "@"
  local x=$'a\x01b' y=$'a\x02b' p=$'p\x01q\x02r' e
  check cbf1 "$(printf '<%s>' "$x" "$y")" "<$x><$y>"
  local l=("$x" "$y" "$p")
  check cbf2 "${#l[@]} ${#l[0]} ${#l[2]} $(printf '<%s>' "${l[@]}")" "3 3 5 <$x><$y><$p>"
  local qx=$'Q\x01b' qy=$'Q\x02b'
  check cbf3 "$(printf '<%s>' "${l[@]/a/Q}" "${l[@]:1:1}")" "<$qx><$qy><$p><$y>"
  s53f() { printf '%s:' "$#" "$@"; }
  check cbf4 "$(s53f "$x" "$p")" "2:$x:$p:"
  check cbf5 "$(s53f $x "${x}z")" "2:$x:${x}z:"
  o=
  for e in "$x" $y; do o="$o[$e]"; done
  check cbf6 "$o" "[$x][$y]"
  check cbf7 "$(IFS=$'\x01'; printf '<%s>' $x)" "<a><b>"
  check cbf8 "$(printf '%c|%b|%.2s' "$x" "$x" "$x")" "a|$x|a${x:1:1}"
  o=$( { e=$(printf 'n\0m'); echo "$e"; } 2>&1 )
  check nul2 "${o##*$'\n'}|${o//*: warning: command substitution: ignored null byte in input$'\n'*/W}" "nm|W"
}

# ===== SECTION 54: several assignments in one command, nested $(( )), expansion before redirection =====
# `a=1 b=$a` runs its assignments left to right. A $(( )) nested in an arithmetic context is
# arithmetic, not `$( ( ) )`. A command's words are expanded BEFORE its redirections apply, and
# a redirection is undone by every way out of its command: return, break, an expansion error.
# Every expected value is bash 5.3's.
s54() {
  local a b c d s=abcdef o i z=0 l1 l2
  a=1 b=2; check mas1 "$a $b" "1 2"
  c=$'x' d=$c; check mas2 "$c $d" "x x"
  false; a=$? b=$?; check mas3 "$a $b" "1 1"
  a=$(false) b=1; check mas4 "$?" 1
  a=(1 2) b[3]=x c+=z 2>/dev/null; check mas5 "${a[*]} ${b[3]} $c" "1 2 x xz"
  check nar1 "${s:1:$(( 1+1 ))} ${s:$((2)):2}" "bc cd"
  check nar2 "$(( $(( 2*3 )) + 1 ))" 7
  o=$( { echo "$(echo hi >&2)" 2>/dev/null; } 2>&1 )
  check xbr1 "$o" hi
  s54f() { printf '%s' "$1"; }
  o=$( { s54f "$(echo e2 >&2)" 2>/dev/null; } 2>&1 )
  check xbr2 "$o" e2
  s54r() { { echo in; return 3; } >/dev/null; }
  o=$(s54r; echo "after $?")
  check xbr3 "$o" "after 3"
  o=$(for i in 1 2; do { echo x; break; } >/dev/null; done; echo after)
  check xbr4 "$o" after
  o=$( ( echo "$(( 3/z ))" >/dev/null ) 2>/dev/null; echo after)
  check xbr5 "$o" after
  o=$(printf 'l1\nl2\n' | { read l1 2>/dev/null; read l2; echo "$l1 $l2"; })
  check xbr6 "$o" "l1 l2"
}

# ===== SECTION 55: a value whose first byte is 0x02 =====
# 0x02 is ordinary data at the START of a value too: a plain string is one field whatever
# its first byte. Every expected value is bash 5.3's.
s55() {
  local w=$'\x02lead' y=$'a\x02b' s=$'\x02' o e a
  check stx1 "$(printf '<%s>' "$w" ${w})" "<$w><$w>"
  check stx2 "${y:1}" "${s}b"
  check stx3 "${#s} ${#w}" "1 5"
  check stx4 "$(printf '<%s>' $w "$w"x x"$w")" "<$w><${w}x><x$w>"
  a=("$w" "$s" b)
  check stx5 "${#a[@]} ${#a[0]} $(printf '<%s>' "${a[@]}")" "3 5 <$w><$s><b>"
  s55f() { printf '%s:' "$#" "$@"; }
  check stx6 "$(s55f "$w") $(s55f $w) $(s55f "$s")" "1:$w: 1:$w: 1:$s:"
  set -- "$w" z
  check stx7 "$(printf '<%s>' "$@" "$1")" "<$w><z><$w>"
  o=
  for e in "$w" $w; do o="$o[$e]"; done
  check stx8 "$o" "[$w][$w]"
  o=$(printf '%s' "$w")
  check stx9 "${#o} ${w/l/L} ${s:-dflt}" "5 ${s}Lead $s"
  e="${w}${w}"; o=${w:0:3}
  check stx10 "${#e} ${#o}" "10 3"
}

# ===== SECTION 56: the text of an arithmetic error =====
# bash names the command ("((: ", "let: ", "[[: ", "v: " in ${v:off}), the expression without
# its leading blanks, and an error token: for / and % the text from the divisor on, for **
# and /= the token after the right operand or else its last one. The location prefix is
# dropped before comparing (bash writes "FILE: line N: ", these engines "bash: ").
# Every expected value is bash 5.3's.
s56m() { local m=${1#bash: }; printf '%s' "${m#*line [0-9]*: }"; }
s56() {
  local z=0 v=1 x=3 e='4/z' s=abc o d h
  o=$( { (( v = 1/z )); } 2>&1 ); check aem1 "$(s56m "$o")" '((: v = 1/z : division by 0 (error token is "z ")'
  o=$( { echo $(( 10 / (3-3) + 1 )); } 2>&1 ); check aem2 "$(s56m "$o")" '10 / (3-3) + 1 : division by 0 (error token is "(3-3) + 1 ")'
  o=$( { let 'q=1/z'; } 2>&1 ); check aem3 "$(s56m "$o")" 'let: q=1/z: division by 0 (error token is "z")'
  o=$( { echo $(( 2 ** -1 + 5 )); } 2>&1 ); check aem4 "$(s56m "$o")" '2 ** -1 + 5 : exponent less than 0 (error token is "+ 5 ")'
  o=$( { (( v /= z, 1 )); } 2>&1 ); check aem5 "$(s56m "$o")" '((: v /= z, 1 : division by 0 (error token is ", 1 ")'
  o=$( { echo $(( e + 1 )); } 2>&1 ); check aem6 "$(s56m "$o")" '4/z: division by 0 (error token is "z")'
  o=$( { [[ 4 -lt $x/z ]]; } 2>&1 ); check aem7 "$(s56m "$o")" '[[: 3/z: division by 0 (error token is "z")'
  o=$( { echo ${s: 1/z }; } 2>&1 ); check aem8 "$(s56m "$o")" 's: 1/z : division by 0 (error token is "z ")'
  o=$( { echo $(( 8 /= 0 )); } 2>&1 ); check aem9 "$(s56m "$o")" '8 /= 0 : attempted assignment to non-variable (error token is "/= 0 ")'
  o=$( { echo $(( 1 + )); } 2>&1 ); check aem10 "$(s56m "$o")" '1 + : arithmetic syntax error: operand expected (error token is "+ ")'
  o=$( { declare -i d; d=5/0; } 2>&1 ); check aem11 "$(s56m "$o")" '5/0: division by 0 (error token is "0")'
  o=$( { read -r h <<S56
a $(( 7/z ))
S56
  } 2>&1 ); check aem12 "$(s56m "$o")" '7/z : division by 0 (error token is "z ")'
  o=$( { for (( d = 2 % z; d < 1; d++ )); do :; done; } 2>&1 ); check aem13 "$(s56m "$o")" '((: d = 2 % z: division by 0 (error token is "z")'
  o=$( { echo x <<< "$(( 7/z ))"; } 2>/dev/null ); check aem14 "$? [$o]" "1 []"
  o=$( { { echo y; } <<< "$(( 7/z ))"; } 2>/dev/null
        echo "st $?" ); check aem15 "$o" ""
}

# ===== SECTION 57: $[ ] arithmetic =====
# $[ EXPR ] is the obsolete spelling of $(( EXPR )), still expanded (bash 5.3).
s57() {
  local x=5 a=(10 20 30) e='1 + 2'
  check lar1 "$[ 3 + 4 ]" 7
  check lar2 "$[x*2]:$[ (x+1) % 4 ]" "10:2"
  check lar3 "$[ a[1] + a[x-3] ]" 50
  check lar4 "$[ $e * 2 ]" 5
  check lar5 "$(( $[2+3] * 2 ))" 10
  check lar6 "q$[1+1]q" q2q
  check lar7 '$[1+1]' '$[1+1]'
}

# ===== SECTION 58: \u / \U code points =====
# \uHHHH and \UHHHHHHHH write a code point's UTF-8 (in a UTF-8 locale) - a surrogate
# or a value past U+10FFFF too, in the same pattern up to six bytes, and nothing from
# 2^31 on; a byte that is no character counts as one in ${#v}. Every expected value is
# bash 5.3's under LC_ALL=en_US.UTF-8.
s58() {
  local v
  v=$'\uDFFF'; check uesc1 "${#v}" 3
  check uesc2 "$v" $'\xed\xbf\xbf'
  check uesc3 $'\u00e9\U0001F600\u41' "é😀A"
  check uesc4 $'\U00110000' $'\xf4\x90\x80\x80'
  v=$'x\U7FFFFFFFy'; check uesc5 "${#v}" 8
  check uesc6 $'a\U80000000b' ab
  check uesc7 $'a\u0b' $'a\x0b'
  check uesc8 $'a\U00000000b' a
  check uesc9 $'a\uxb' 'a\uxb'
  check uesc10 "$(printf '\uD800|\U7FFFFFFF')" $'\xed\xa0\x80|\xfd\xbf\xbf\xbf\xbf\xbf'
  v='a\uDFFFb\U1F600\ux'; check uesc11 "${v@E}" $'a\xed\xbf\xbfb\U1F600\\ux'
  check uesc12 "$(echo -e '\U04000000')" $'\xfc\x84\x80\x80\x80\x80'
}

# ===== SECTION 59: extended globs behind a star =====
# bash's star (GMATCH in lib/glob/sm_loop.c) tries what follows it only at starts BEFORE
# the end of the subject, so an @( ) behind it never matches an empty tail - while a
# top-level !( ) there always does. A ?( ) or *( ) right behind the star is tried in place
# and dropped (**(a) acts as *). One occurrence of +( ) may be empty, and an empty last
# alternative counts. ${v/%p/r} first matches *p against an ASCII subject (match_upattern),
# so the same star decides it; a non-ASCII one gets *p* (match_wpattern). Every expected
# value is bash 5.3's under LC_ALL=en_US.UTF-8.
s59() {
  local r="" s p v=xy w=xyé e="" c=""
  for s in "" a b xy ab; do
    for p in '*@(|a)' '*@(a|)' '*!(a)' '**(a)' '*?(b)' '*?@(|a)' 'x*@(|a)' '+(|a)b' '@(*!(a))' '@()' '*!(a)b'; do
      if [[ $s == $p ]]; then r="${r}1"; else r="${r}0"; fi
    done
    r="$r "
  done
  check egs1 "$r" "00111000011 11011000000 00111001101 00111000100 00111001101 "
  shopt -s extglob
  check egs2 "[${v/%@(|a)/Z}] [${w/%@(|a)/Z}] [${v/#@(|a)/Z}] [${v//@(|a)/Z}] [${v/%*(a)/Z}] [${v/%\*/Z}] [${e/%@(|a)/Z}] [${v%@(|a)}]" "[xy] [xyéZ] [Zxy] [ZxZy] [xyZ] [xy] [] [xy]"
  p='*@(|a)'; case xy in $p) c=1 ;; *) c=2 ;; esac
  p='*!(a)'; case "" in $p) c="${c}3" ;; *) c="${c}4" ;; esac
  p='+(|a)b'; case b in $p) c="${c}5" ;; *) c="${c}6" ;; esac
  shopt -u extglob
  check egs3 "$c" 235
  # A group's prefix is a whole number of characters: é is never split into two bytes.
  r=""
  [[ é == !(|é)? ]] && r="${r}1" || r="${r}0"
  [[ é == @(?)? ]] && r="${r}1" || r="${r}0"
  [[ éa == @(?)?a ]] && r="${r}1" || r="${r}0"
  check egs4 "$r" 000
}

# ===== SECTION 60: what a loop keeps while its scratch is reused =====
# A compiled loop gives back what each pass allocated, and a scalar assignment moves its
# value down over its own scratch. What a pass stores - a scalar, a copy of the loop
# variable, an array element, a positional parameter, a capture - has to survive every
# later pass. (The long run itself is tests/bash-native-heap.sh.) Every expected value
# is bash 5.3's.
s60() {
  local w ws=(alpha beta gamma) prev="" cur="" al="" acc="" i r=""
  local -a kept=()
  for w in "${ws[@]}" "${ws[@]}"; do
    prev=$cur
    al=$w
    cur=${w//a/A}
    kept+=("${w:1:2}${#acc}")
    acc+="${w%a}."
    : "$w$w$w$w"
  done
  check hr1 "$prev $cur $al ${kept[*]} $acc" "betA gAmmA gamma lp0 et5 am9 lp14 et19 am23 alph.bet.gamm.alph.bet.gamm."
  set --
  i=0
  while [ $i -lt 4 ]; do
    set -- "$@" "${ws[i % 3]^^}$i"
    i=$((i + 1))
  done
  r=$(for w in "$@"; do printf '%s,' "${w,,}"; : "$w$w$w"; done)
  check hr2 "$# $* $r" "4 ALPHA0 BETA1 GAMMA2 ALPHA3 alpha0,beta1,gamma2,alpha3,"
  # --- a long value a plain assignment stores is kept in its slot's own buffer, which
  # the next one rewrites in place: whatever else took the old value keeps it
  local L=0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz big cp out k=0 sub fr
  local -a held=()
  s60f() { local big="${L}in$1"; out="$out<${big:72}|$2>"; big="${L}in2$1"; out="$out<${big:72}>"; }
  s60g() { big="${L}g${1:72}"; out="$out[${1:72}|${big:72}]"; }
  out=""
  set --
  while [ $k -lt 4 ]; do
    big="${L}a$k"
    cp=$big
    held[k]=$big
    set -- "$big" "$@"
    big="${L}b$k"
    s60f "$k" "${big:72}"
    s60g "$big"
    sub=$(big="${L}s$k"; fr="${L}f$k"; fr="${fr}F"; echo "${big:72}${fr:72}")
    ( big="${L}t$k" )
    out="$out{${cp:72}|${big:72}|$sub|${fr:72}}"
    fr="${L}o$k"
    while read -r line; do
      big="${L}h$k"
      out="$out(${line:72})"
    done <<EOF
$big
${big}2
EOF
    k=$((k + 1))
  done
  check hr3 "$out" "<in0|b0><in20>[b0|gb0]{a0|gb0|s0f0F|}(gb0)(gb02)<in1|b1><in21>[b1|gb1]{a1|gb1|s1f1F|o0}(gb1)(gb12)<in2|b2><in22>[b2|gb2]{a2|gb2|s2f2F|o1}(gb2)(gb22)<in3|b3><in23>[b3|gb3]{a3|gb3|s3f3F|o2}(gb3)(gb32)"
  check hr4 "${held[*]:0:4}" "${L}a0 ${L}a1 ${L}a2 ${L}a3"
  check hr5 "$# ${1:72} ${2:72} ${4:72} ${big:72} ${cp:72}" "4 a3 a2 a0 h3 a3"
}
# ===== SECTION 62: a scalar is element 0 of the array its name becomes =====
# An element assignment, an appending literal or `declare -a` on a SET scalar keeps
# its value as element 0; a scalar assignment to an array name writes element 0, and
# `+=` appends to its text. Every expected value is bash 5.3's.
s62() {
  local t=abcd u=ab v=ab w=ab y=ab z=ab k=ab e= p=ab g
  t[1]=x
  check sa1 "${t[*]}|${#t[@]}|${!t[*]}|$t" "abcd x|2|0 1|abcd"
  u[1]=x
  v[0]=x
  check sa2 "${u[*]}|${v[*]}|${#v[@]}" "ab x|x|1"
  w[0]+=x
  y+=(x y)
  check sa3 "${w[*]}|${y[*]}|${#y[@]}" "abx|ab x y|3"
  z[3]=q
  k[-1]=K
  e[1]=x
  check sa4 "${!z[*]}|${z[*]}|$k|${#e[@]}|[${e[0]}]" "0 3|ab q|K|2|[]"
  s62f() { local q=lo; q[2]=m; echo "${q[*]}"; }
  local q=out
  check sa5 "$(s62f)|$q" "lo m|out"
  s62g() { local -a q; q[1]=z; echo "${q[*]}"; }
  check sa6 "$(s62g)" "z"
  local h=ab
  ( h[1]=c; echo "${h[*]}" ) > /dev/null
  check sa7 "${h[*]}|$(h[1]=c; echo "${h[*]}")" "ab|ab c"
  unset 'p[0]'
  check sa8 "[${p-unset}]|${#p[@]}" "[unset]|0"
  local r=ab
  r[1]=c
  r=zz
  g[2]=x
  g=G
  check sa9 "${r[*]}|${g[*]}|${!g[*]}" "zz c|G x|0 2"
  local f=(ab c)
  f+=d
  check sa10 "${f[*]}" "abd c"
  d62=ab
  declare -ga d62
  check sa11 "${d62[*]}|${#d62[@]}" "ab|1"
  local o=ab
  o=(x)
  unset 'o[0]'
  check sa12 "[${o-unset}]" "[unset]"
  # --- each array is found by its own name; a large one bisects or hashes its keys
  local -a big=()
  local -A am=()
  local i=0 n=0 s=0
  while [ $i -lt 300 ]; do
    big+=("v$i")
    am[k$i]=$i
    i=$((i + 1))
  done
  i=0
  while [ $i -lt 300 ]; do
    s=$((s + ${am[k$i]}))
    [ "${big[i]}" = "v$i" ] && n=$((n + 1))
    i=$((i + 3))
  done
  check sa13 "${#big[@]} ${#am[@]} ${big[-1]} ${am[k299]} $n $s" "300 300 v299 299 100 14850"
  big[5000]=far
  big[4000]=mid
  unset 'big[0]' 'am[k7]'
  check sa14 "${#big[@]} ${big[*]: -1001} ${big[299]} ${#am[@]} [${am[k7]-gone}]" "301 mid far v299 299 [gone]"
  s62h() { local big=(one); big[1]=two; s62r="${big[*]}"; }
  s62h
  check sa15 "$s62r ${#big[@]} ${big[4000]}" "one two 301 mid"
  # --- an unset element is gone, and a later write brings it back
  local -a ua=(x y)
  local -A um=([p]=1)
  unset 'ua[1]' 'um[p]'
  check sa16 "[${ua[1]-gone}][${um[p]-gone}]" "[gone][gone]"
  ua[1]=z
  um[p]=2
  check sa17 "${ua[*]} ${#ua[@]} ${um[p]} ${#um[@]}" "x z 2 2 1"
}
# ===== SECTION 63: mapfile, element attributes, unset 'p[1]', ${!a[k]} =====
# mapfile / readarray store the records of standard input in an indexed array; -i, -u and
# -l apply to every element an array stores as to a scalar; `unset 'p[1]'` on a scalar is
# an error; ${!a[k]} and ${!a[@] OP} are indirections through the element(s), and one that
# lands on no name is an expansion error. Every expected value is bash 5.3's.
s63m() { local m=${1#bash: }; m=${m#*line [0-9]*: }; printf '%s' "${m%%$'\n'*}"; }
s63c() { r63="$r63<$1:$2>"; }
s63() {
  local -a m=() n=(1 2 3 4)
  local o r x
  mapfile -t m <<< "a b
c

d"
  check mf1 "${#m[@]}|${m[*]}|[${m[2]}]" "4|a b c  d|[]"
  mapfile m <<< "x"
  check mf2 "${#m[@]} ${#m[0]}" "1 2"
  mapfile -t -O 2 -n 1 n <<< "X
Y"
  check mf3 "${n[*]}" "1 2 X 4"
  readarray -t -s 1 -d : m <<< "p:q:r"
  check mf4 "${#m[@]}|${m[0]}|${m[1]}" $'2|q|r\n'
  r63=""
  mapfile -tc 2 -C s63c m <<< "a
b
c
d
e"
  check mf5 "$r63 ${m[*]}" "<1:b><3:d> a b c d e"
  x=5
  mapfile -t -O 1 x <<< "w"
  check mf6 "${x[*]}|${#x[@]}" "5 w|2"
  x=5
  mapfile -t x < /dev/null
  check mf7 "${#x[@]}|${x-unset}" "0|unset"
  mapfile -t <<< "M"
  check mf8 "${MAPFILE[*]}" "M"
  o=$( { mapfile -t -n x m <<< "q"; } 2>&1 ); r=$?
  check mf9 "$r $(s63m "$o")" "1 mapfile: x: invalid line count"
  o=$( { readarray -q; } 2>&1 ); r=$?
  check mf10 "$r $(s63m "$o")" "2 readarray: -q: invalid option"
  o=$( { mapfile -t 1x <<< "q"; } 2>&1 ); r=$?
  check mf11 "$r $(s63m "$o")" "1 mapfile: \`1x': not a valid identifier"
  # --- -i, -u and -l on the elements of an array
  local -i ni=5
  ni[1]=3+4
  ni[2]+=2
  ni+=(2*3)
  local -l lo=ABC
  lo[1]=XY
  lo+=(Mn)
  local -u up=(ab)
  up[0]+=cd
  up[2]=ef
  check ea1 "${ni[*]}|${lo[*]}|${up[*]}" "5 7 2 6|abc xy mn|ABCD EF"
  local -ai ai=(1+1 2*3)
  ai=7*2
  ai+=1
  local -Ai am=([k]=1+1)
  am[j]=2*4
  am+=([h]=3+3)
  check ea2 "${ai[*]}|${am[k]} ${am[j]} ${am[h]}" "15 6|2 8 6"
  local -ai ar
  read -a ar <<< "3+3 4"
  local -ua mu
  mapfile -t mu <<< "x
y"
  check ea3 "${ar[*]}|${mu[*]}|${up[3]:=gh} ${up[*]}" "6 4|X Y|GH ABCD EF GH"
  o=$( local -ai e=(1 2); { e=(5 x+ 7); echo never; } 2>/dev/null ); r=$?
  check ea4 "$r[$o]" "1[]"
  # --- unset of an element of a scalar
  local p=ab q=cd w
  unset 'p[1]' 2>/dev/null; r=$?
  unset 'q[0]' 'nosuch[3]'
  check us1 "$r|$p|$?|${q-unset}" "1|ab|0|unset"
  unset 'p[@]' 2>/dev/null; r=$?
  unset 'w[2]' 2>/dev/null
  check us2 "$r|$?|$p" "1|1|ab"
  o=$( { unset 'p[-1]'; } 2>&1 )
  check us3 "$(s63m "$o")" "unset: p: not an array variable"
  # --- indirection through an element
  local hv=hello wv=world
  local -a ia=(hv wv) one=(hv)
  check in1 "${!ia[0]}|${!ia[1]^^}|${!ia[1]:1:3}|${!one[@]:1}|${!one[*]#h}|${!ia[*]}" "hello|WORLD|orl|ello|ello|0 1"
  check in2 "${!ia[5]:-dflt}|${!ia[7]-d2}" "dflt|d2"
  o=$( { echo "${!ia[@]:0:3}"; echo never; } 2>&1 ); r=$?
  check in3 "$r $(s63m "$o")" "1 hv wv: invalid variable name"
  o=$( { unset nsr; echo "${!nsr}"; echo never; } 2>&1 ); r=$?
  check in4 "$r $(s63m "$o")" "1 nsr: invalid indirect expansion"
  o=$( { x=''; echo "${!x:-q}"; echo never; } 2>&1 ); r=$?
  check in5 "$r $(s63m "$o")" "1 : invalid variable name"
}
# ===== SECTION 64: what holds a long array element that is overwritten =====
# An element assignment keeps a long value in the element's own buffer, which the next one
# rewrites in place: a copy taken of it - by a variable, a positional parameter, another
# element, an expansion still being built, a here-string, a caller's array a `local` hides,
# a subshell - keeps the old value, and an element unset, emptied or replaced by arithmetic
# reads back right. Every expected value is bash 5.3's.
s64f() { local -a a; a[0]="${L64}f$1"; out="$out<${a[0]:72}>"; a[0]="${L64}g$1"; out="$out<${a[0]:72}>"; }
s64g() { a[0]="${L64}h${1:72}"; out="$out[${1:72}|${a[0]:72}]"; }
s64() {
  L64=0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz
  local -a a=(x y) b=() c=()
  local -A m64=([k]=v)
  local out="" k=0 cp line sub
  set --
  while [ $k -lt 4 ]; do
    a[0]="${L64}a$k"
    m64[k]="${L64}m$k"
    cp=${a[0]}
    b[k]=${a[0]}
    c=("${a[@]}")
    set -- "${a[0]}" "$@"
    a[0]="${L64}b$k"
    m64[k]+="+"
    s64f "$k"
    s64g "${a[0]}"
    sub=$(a[0]="${L64}s$k"; echo "${a[0]:72}")
    ( a[0]="${L64}t$k" )
    line="${a[0]:72}$(( a[1] = k ))${a[0]:72}"
    out="$out{${cp:72}|${b[k]:72}|${c[0]:72}|$sub|$line|${m64[k]:72}}"
    while read -r line; do
      a[0]="${L64}r$k"
      out="$out(${line:72})"
    done <<< "${a[0]}"
    k=$((k + 1))
  done
  check eb1 "$out" "<f0><g0>[b0|hb0]{a0|a0|a0|s0|hb00hb0|m0+}(hb0)<f1><g1>[b1|hb1]{a1|a1|a1|s1|hb11hb1|m1+}(hb1)<f2><g2>[b2|hb2]{a2|a2|a2|s2|hb22hb2|m2+}(hb2)<f3><g3>[b3|hb3]{a3|a3|a3|s3|hb33hb3|m3+}(hb3)"
  check eb2 "$# ${1:72} ${4:72} ${b[*]:0:4}" "4 a3 a0 ${L64}a0 ${L64}a1 ${L64}a2 ${L64}a3"
  a[0]="${L64}u"
  (( a[0] = 7 ))
  a[2]="${L64}v"
  unset 'a[2]'
  a[3]="${L64}w"
  a=("${a[3]}" "${a[@]}")
  a[0]="${L64}x"
  check eb3 "${#a[@]} ${a[0]:72} ${a[1]} ${a[2]} ${a[3]:72}" "4 x 7 3 w"
  a[0]="${L64}o"
  ( unset 'a[0]'; a[5]="${L64}z"; a[6]="${L64}y" )
  sub=$( (( a[0] = 1 )); a[1]="${L64}q"; a=(); a[2]="${L64}p"; echo "${a[2]:72}" )
  check eb4 "${a[0]:72} $sub ${#a[@]}" "o p 4"
}
# ===== SECTION 65: every pipeline stage is a subshell =====
# Without lastpipe each stage of a pipeline runs in a subshell, the last one too: what it
# assigns, reads or maps is gone afterwards, and exit / return / set -e end only the stage.
# A ( ) starts outside every loop; a simple command run as a stage and a $( ) still see the
# loops around them, and break / continue there end only the subshell. Every expected value
# is bash 5.3's.
s65r() { local v=1; v=2 | true; echo in | return 7; echo "$v $?"; (return 3); echo "r$?"; }
s65() {
  local x=1 out="" i
  echo a | read x
  x=2 | true
  printf 'p\nq\n' | mapfile -t s65m
  echo z | { read s65n; s65o=1; }
  check pl1 "$x ${#s65m[@]} ${s65n-unset} ${s65o-unset}" "1 0 unset unset"
  local -a a=(1)
  a[1]=2 | true
  local -A s65h=([k]=v)
  s65h[j]=w | true
  check pl2 "${#a[@]} ${#s65h[@]}" "1 1"
  exit 3 | true
  out="$?"
  echo z | exit 4
  out="$out $? $(s65r)"
  check pl3 "$out" "0 4 1 7
r3"
  out=$( set -e; { false; echo no; } | true; echo y1; f() { false; echo no; }; f | true; echo y2; true | false; echo no )
  check pl4 "$? $out" "1 y1
y2"
  out=""
  for i in 1 2; do out="$out $( (break; echo "b$i") 2>/dev/null )"; done
  for i in 1 2; do true | break; out="$out $i${PIPESTATUS[1]}"; done
  for i in 1 2; do out="$out $(true | { continue; echo "c$i"; } 2>/dev/null)"; done
  for i in 1 2; do v=$(continue; echo no); out="$out [$v$?]"; done
  check pl5 "$out" " b1 b2 10 20 c1 c2 [0] [0]"
}
# ===== SECTION 66: ${!REF OP} acts on the target as if it were written there =====
# An indirection with an operator applies it to what REF names: @, * and NAME[@] / NAME[*]
# are lists, sliced and mapped element by element, and NAME[SUB] is that element. A scalar's
# NAME[@] is the substring of its value. A ${!REF:?} error names "!REF". Every expected value
# is bash 5.3's (with the default IFS).
s66p() { printf '<%s>' "$@"; }
s66m() { local m=${1#bash: }; m=${m#*line [0-9]*: }; printf '%s' "${m%%$'\n'*}"; }
s66() {
  local -a a=(aa bb 'c c' dd) e=()
  local s=hello x o r
  set -- p q 'r s'
  x='@'
  check ix1 "$(s66p "${!x:1}" / "${!x: -1}" / "${!x:2:1}" / "${!x#?}" / "${!x^^}" / ${!x:1})" \
    "<p><q><r s></><r s></><q></><><>< s></><P><Q><R S></><p><q><r><s>"
  check ix2 "$(s66p "${!x@Q}" "${!x:-d}")|$(set --; s66p "${!x:-d}" "${!x:+alt}")" "<'p'><'q'><'r s'><p><q><r s>|<d>"
  x='a[@]'
  check ix3 "$(s66p "${!x:1:2}" / "${!x/?/Z}" / "${!x%?}" / "${!x: -1}")" "<bb><c c></><Za><Zb><Z c><Zd></><a><b><c ><d></><dd>"
  x='a[*]'
  check ix4 "$(s66p "${!x:1}" "${!x//[ac]/Y}")" "<bb c c dd><YY bb Y Y dd>"
  x='a[2]'
  check ix5 "$(s66p "${!x:1}" "${!x^}" "${!x@Q}" "${!x:-d}")" "< c><C c><'c c'><c c>"
  x='s[@]'
  check ix6 "$(s66p "${!x:1:-1}" / "${s[@]:1}" / "${s[@]:9}" / "${s[*]:9}" / "${s[@]: -2}")" "<ell></><ello></></><></><lo>"
  x='e[@]'
  check ix7 "$(s66p "${!x:-d}" "${!x-u}")|$(s66p "${!x:+alt}")" "<d><u>|<>"
  o=$( { echo "${!x:?gone}"; echo never; } 2>&1 ); r=$?
  check ix8 "$r $(s66m "$o")" "1 !x: gone"
  o=$( { x=nosuch; echo "${!x:?}"; echo never; } 2>&1 ); r=$?
  check ix9 "$r $(s66m "$o")" "1 !x: parameter null or not set"
  o=$( { echo "${e[@]:?empty}"; echo never; } 2>&1 ); r=$?
  check ix10 "$r $(s66m "$o")" "1 e[@]: empty"
}
# ===== SECTION 67: every store honours the attributes; a declaration expands first =====
# read, printf -v, a for loop's name and ${v:=w} store a value the way NAME=v does: -i
# evaluates it (an error abandons the line and stores nothing), -u / -l map its case, and an
# array name takes it as element 0. A declaration expands its words - and assigns an array
# literal - before its redirections apply and before it stores anything. Every expected
# value is bash 5.3's.
s67m() { local m=${1#bash: }; m=${m#*line [0-9]*: }; printf '%s' "${m%%$'\n'*}"; }
s67() {
  local -i c=5 c2 f p fi n=4 r1=1 r2=2
  local -u u g q
  local -l h
  local -au d=(ab)
  local acc="" o
  read c <<< "1+2"
  read c2 <<< "c+4"
  read u <<< "ab cd"
  read d <<< xy
  check at1 "$c $c2 $u ${d[0]} ${#d[@]}" "3 7 AB CD XY 1"
  check at2 "${f:=2+3} $f ${g:=xy} $g ${h:=AbC} $h" "5 5 XY XY abc abc"
  printf -v p '%s' 6*7
  printf -v q '%s' hi
  for fi in 1+1 3*3; do acc="$acc $fi"; done
  check at3 "$p $q$acc" "42 HI 2 9"
  o=$( { read r1 r2 <<< "2*3 4+4 5"; echo never; } 2>&1 )
  check at4 "$(s67m "$o")" 'read: 4+4 5: arithmetic syntax error in expression (error token is "5")'
  o=$( { n=1+; echo never; } 2>&1 )
  check at5 "$(s67m "$o")" '1+: arithmetic syntax error: operand expected (error token is "+")'
  s67x=0
  local s67x=1 y6=$s67x
  local -a c6=(7) d6=${c6[0]}
  check at6 "$y6 $d6" "0 7"
  o=$( { local -ai k6=(1 x+ 3) 2>/dev/null; echo never; } 2>&1 )
  check at7 "$(s67m "$o")" 'x+: arithmetic syntax error: operand expected (error token is "+")'
  o=$( { local -i n6=1+ 2>/dev/null; echo never; } 2>&1 )
  check at8 "[$o]" "[]"
  o=$( { local v6=$((2+)) 2>/dev/null; echo never; } 2>&1 )
  check at9 "$(s67m "$o")" '2+: arithmetic syntax error: operand expected (error token is "+")'
}
# ===== SECTION 68: what set -e looks at; a subshell's own EXIT trap =====
# set -e ends the shell on a failing simple command, ( ), pipeline or function call - and on
# the LAST command of an && / || list - but not on a ! pipeline, a compound command whose
# failure was exempt inside it, or anything run as a condition (a function called from one
# included); a command substitution does not inherit it. A subshell starts with no EXIT
# trap, and the status it leaves with is the one its own trap started with, unless the trap
# exits. Every expected value is bash 5.3's.
s68f() { false; echo inf; }
s68g() { false && true; }
s68() {
  local o r=""
  o=$( set -e; false && true; ! true; if true; then false && true; fi; { false && true; }
       for i in 1; do false && true; done; case x in x) false && true;; esac; while false; do :; done
       s68f || true; x=$(false; echo hi); echo "$x survived" )
  check ee1 "$o" "inf
hi survived"
  ( set -e; true && false; echo no ); r="$r $?"
  ( set -e; ! false && false; echo no ); r="$r $?"
  ( set -e; s68g; echo no ); r="$r $?"
  ( set -e; (( 0 )); echo no ); r="$r $?"
  ( set -e; ( false && true ); echo no ); r="$r $?"
  ( set -e; echo x | { false && true; }; echo no ); r="$r $?"
  check ee2 "$r" " 1 1 1 1 1 1"
  o=$( trap 'echo outer' EXIT; x=$( exit 4 ); y=$( echo in ); echo "$x|$y" )
  check tr1 "$o" "|in
outer"
  o=$( ( trap 'false' EXIT; true ); echo "$?"; ( trap 'echo "[$?]"; exit 3' EXIT; false ); echo "$?" )
  check tr2 "$o" "0
[1]
3"
  o=$( r=$( trap 'echo T' EXIT; false ); echo "$? $r" )
  check tr3 "$o" "1 T"
}
# ===== SECTION 69: a second `local` of a name in the same call =====
# Only the first `local` of a name in a call starts it unset and plain; a second one keeps
# its value, array and attributes, and adds its own - except that a scalar declared an array
# again becomes an empty one. The first at RUN time counts: one in a loop or behind an `if`
# that did not run is not a second. Every expected value is bash 5.3's.
s69g=G
s69f() {
  local a=5; local a
  local b=6; local -i b; b=1+1
  local -i c=3; local c; c=2+2
  local -a d=(1 2); local d
  local -u h=ab; local h; h=cd
  local x=5; local -a x
  local out="$a $b $c ${d[*]} $h [${x[0]} ${#x[@]}]" i v
  for i in 1 2; do local v; out="$out [$v]"; v=$i; done
  if false; then local s69g=1; fi; local s69g; out="$out [$s69g]"
  echo "$out"
}
s69() {
  check lo1 "$(s69f)" "5 2 4 1 2 CD [ 0] [] [1] []"
  check lo2 "$s69g" "G"
}
# ===== END SECTIONS =====

s47   # SECTION-CALL 47
s48   # SECTION-CALL 48
s49   # SECTION-CALL 49
s50   # SECTION-CALL 50
s51   # SECTION-CALL 51
s52   # SECTION-CALL 52
s53   # SECTION-CALL 53
s54   # SECTION-CALL 54
s55   # SECTION-CALL 55
s56   # SECTION-CALL 56
s57   # SECTION-CALL 57
s58   # SECTION-CALL 58
s59   # SECTION-CALL 59
s60   # SECTION-CALL 60
s62   # SECTION-CALL 62
s63   # SECTION-CALL 63
s64   # SECTION-CALL 64
s65   # SECTION-CALL 65
s66   # SECTION-CALL 66
s67   # SECTION-CALL 67
s68   # SECTION-CALL 68
s69   # SECTION-CALL 69
echo "full: $checks checks, $fails failures"
exit $fails
