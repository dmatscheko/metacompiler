#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 3
# of 3, sections 47 on; a new section goes here. The split is explained in
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
echo "full: $checks checks, $fails failures"
exit $fails
