#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language).
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

# ===== SECTION 01: baseline =====
# Condensed re-assertion of the feature-matrix basics this file builds on.
s01() {
  a=5
  check bas1 "$a" 5
  check bas2 "${a}0" 50
  greeting="value is $a"
  check bas3 "$greeting" "value is 5"
  check bas4 'no $a here' 'no $a here'
  check bas5 "$(( 2 + 3 * 4 ))" 14
  local r=none
  if [ 1 -eq 1 ]; then r=then; else r=else; fi
  check bas6 "$r" then
  local acc=""
  for w in x y z; do acc="$acc$w"; done
  check bas7 "$acc" xyz
  local i=0 sum=0
  while [ $i -lt 5 ]; do sum=$(( sum + i )); i=$(( i + 1 )); done
  check bas8 "$sum" 10
  true
  check bas9 "$?" 0
  false
  check bas10 "$?" 1
}

# ===== SECTION 02: quoting and escaping =====
s02() {
  local v=world
  check quo1 "hello $v" "hello world"
  check quo2 'hello $v' 'hello $v'
  check quo3 "a\$b" 'a$b'
  check quo4 "a\\b" 'a\b'
  check quo5 "say \"hi\"" 'say "hi"'
  check quo6 a\ b "a b"
  # adjacent quoted and bare segments concatenate into ONE word
  check quo7 'a'"b"c "abc"
  # a single quote cannot be escaped inside single quotes; this is the idiom
  check quo8 'it'\''s' "it's"
  # $'...' is the ANSI-C quoting form: escapes are interpreted
  check quo9 $'a\tb' "a	b"
  check quo10 $'x\ny' "x
y"
  # a backslash-newline inside an unquoted word is a line continuation
  check quo11 ab\
cd "abcd"
  # inside double quotes a backslash is literal unless it precedes $ ` " \ or newline
  check quo12 "a\qb" 'a\qb'
}

# ===== SECTION 03: parameter expansion: defaults and length =====
s03() {
  local set_v=hello unset_v=
  unset unset_v
  check par1 "${set_v}" hello
  check par2 "${#set_v}" 5
  check par3 "${unset_v:-fallback}" fallback
  check par4 "${set_v:-fallback}" hello
  local empty=""
  check par5 "${empty:-fallback}" fallback
  check par6 "${empty-fallback}" ""
  check par7 "${unset_v:+alt}" ""
  check par8 "${set_v:+alt}" alt
  local assigned=
  unset assigned
  check par9 "${assigned:=given}" given
  check par10 "$assigned" given
  # ${!name} is indirect expansion
  local target=42 pointer=target
  check par11 "${!pointer}" 42
}

# ===== SECTION 04: parameter expansion: substrings, patterns, case =====
s04() {
  local p=/usr/local/lib/libfoo.so.1
  check sub1 "${p:5}" local/lib/libfoo.so.1
  check sub2 "${p:1:3}" usr
  check sub3 "${p: -1}" 1
  check sub4 "${p#*/}" usr/local/lib/libfoo.so.1
  check sub5 "${p##*/}" libfoo.so.1
  check sub6 "${p%.*}" /usr/local/lib/libfoo.so
  check sub7 "${p%%.*}" /usr/local/lib/libfoo
  local s=banana
  check sub8 "${s/na/NA}" baNAna
  check sub9 "${s//na/NA}" baNANA
  check sub10 "${s/#ba/BA}" BAnana
  check sub11 "${s/%na/NA}" banaNA
  check sub12 "${s^}" Banana
  check sub13 "${s^^}" BANANA
  local u=BANANA
  check sub14 "${u,}" bANANA
  check sub15 "${u,,}" banana
}

# ===== SECTION 05: command substitution =====
s05() {
  check cmd1 "$(echo hi)" hi
  check cmd2 "`echo old`" old
  check cmd3 "a$(echo b)c" abc
  # nesting, and the trailing newline is stripped
  check cmd4 "$(echo "$(echo deep)")" deep
  local n
  n=$(echo 7)
  check cmd5 "$n" 7
  # a command substitution's own exit status reaches $?
  local out
  out=$(false)
  check cmd6 "$?" 1
  check cmd7 "$out" ""
  # arithmetic inside command substitution and vice versa
  check cmd8 "$(( $(echo 3) + 4 ))" 7
}

# ===== SECTION 06: arithmetic =====
s06() {
  check ari1 "$(( 7 / 2 ))" 3
  check ari2 "$(( -7 / 2 ))" -3
  check ari3 "$(( 7 % -2 ))" 1
  check ari4 "$(( 2 ** 10 ))" 1024
  check ari5 "$(( 1 << 4 ))" 16
  check ari6 "$(( 255 >> 4 ))" 15
  check ari7 "$(( 12 & 10 ))" 8
  check ari8 "$(( 12 | 3 ))" 15
  check ari9 "$(( 12 ^ 10 ))" 6
  check ari10 "$(( ~0 ))" -1
  check ari11 "$(( !0 ))" 1
  check ari12 "$(( 1 && 0 ))" 0
  check ari13 "$(( 1 || 0 ))" 1
  check ari14 "$(( 3 > 2 ? 10 : 20 ))" 10
  check ari15 "$(( 0x10 + 0 ))" 16
  check ari16 "$(( 010 + 0 ))" 8
  check ari17 "$(( 2#1011 ))" 11
  local i=5
  check ari18 "$(( i++ ))" 5
  check ari19 "$i" 6
  check ari20 "$(( ++i ))" 7
  local j=10
  (( j += 5 ))
  check ari21 "$j" 15
  (( j *= 2 ))
  check ari22 "$j" 30
  # (( )) as a command: status 0 when the value is non-zero
  if (( 1 )); then check ari23 yes yes; else check ari23 no yes; fi
  if (( 0 )); then check ari24 no ok; else check ari24 ok ok; fi
  # let is the same evaluator
  let "k = 3 * 4"
  check ari25 "$k" 12
  check ari26 "$(( 1, 2, 3 ))" 3
}

# ===== SECTION 07: indexed arrays =====
s07() {
  local -a arr=(alpha beta gamma)
  check arr1 "${arr[0]}" alpha
  check arr2 "${arr[2]}" gamma
  check arr3 "${#arr[@]}" 3
  check arr4 "${#arr[1]}" 4
  local all="${arr[@]}"
  check arr5 "$all" "alpha beta gamma"
  check arr6 "${arr[*]}" "alpha beta gamma"
  arr[1]=BETA
  check arr7 "${arr[1]}" BETA
  arr+=(delta)
  check arr8 "${#arr[@]}" 4
  check arr9 "${arr[3]}" delta
  check arr10 "${arr[*]:1:2}" "BETA gamma"
  check arr11 "${!arr[*]}" "0 1 2 3"
  local -a sparse=()
  sparse[5]=five
  check arr12 "${#sparse[@]}" 1
  check arr13 "${!sparse[*]}" 5
  unset 'arr[0]'
  check arr14 "${#arr[@]}" 3
  local joined=""
  for e in "${arr[@]}"; do joined="$joined:$e"; done
  check arr15 "$joined" ":BETA:gamma:delta"
  # an unquoted ${a[@]} splits per element, "${a[*]}" joins with the first IFS char
  local -a two=("a b" "c d")
  check arr16 "${#two[@]}" 2
  check arr17 "${two[0]}" "a b"
}

# ===== SECTION 08: associative arrays =====
s08() {
  local -A ages
  ages[ann]=30
  ages[bob]=41
  check asc1 "${ages[ann]}" 30
  check asc2 "${ages[bob]}" 41
  check asc3 "${#ages[@]}" 2
  local -A colors=([red]=ff0000 [green]=00ff00)
  check asc4 "${colors[red]}" ff0000
  check asc5 "${colors[green]}" 00ff00
  # a missing key expands to the empty string
  check asc6 "${colors[blue]}" ""
  # -v tests whether a key is set
  if [[ -v colors[red] ]]; then check asc7 yes yes; else check asc7 no yes; fi
  if [[ -v colors[blue] ]]; then check asc8 no ok; else check asc8 ok ok; fi
  colors[red]=cc0000
  check asc9 "${colors[red]}" cc0000
  unset 'colors[red]'
  check asc10 "${#colors[@]}" 1
}

# ===== SECTION 09: brace and tilde expansion =====
s09() {
  check brc1 "$(echo a{1,2,3}b)" "a1b a2b a3b"
  check brc2 "$(echo {1..5})" "1 2 3 4 5"
  check brc3 "$(echo {5..1})" "5 4 3 2 1"
  check brc4 "$(echo {a..e})" "a b c d e"
  check brc5 "$(echo {1..9..3})" "1 4 7"
  check brc6 "$(echo pre{x,y}post)" "prexpost preypost"
  check brc7 "$(echo {a,b}{1,2})" "a1 a2 b1 b2"
  # a brace with no comma or range is NOT an expansion
  check brc8 "$(echo {single})" "{single}"
  # tilde expands to $HOME; the value itself is environment dependent, so only
  # the shape is asserted
  local home_expanded=~
  if [ -n "$home_expanded" ]; then check brc9 yes yes; else check brc9 no yes; fi
  if [ "$home_expanded" = "~" ]; then check brc10 no ok; else check brc10 ok ok; fi
}

# ===== SECTION 10: word splitting and IFS =====
s10() {
  local packed="a b c"
  local -a parts=($packed)
  check wsp1 "${#parts[@]}" 3
  check wsp2 "${parts[1]}" b
  # quoting suppresses splitting
  local -a whole=("$packed")
  check wsp3 "${#whole[@]}" 1
  local saved_ifs="$IFS"
  IFS=:
  local csv="x:y:z"
  local -a fields=($csv)
  check wsp4 "${#fields[@]}" 3
  check wsp5 "${fields[2]}" z
  # "$*" joins with the first character of IFS
  set -- one two
  check wsp6 "$*" "one:two"
  IFS="$saved_ifs"
  set --
  # an unquoted empty expansion disappears entirely
  local empty=""
  local -a none=($empty)
  check wsp7 "${#none[@]}" 0
}

# ===== SECTION 11: [[ ]] conditional expressions =====
s11() {
  local s=hello n=42
  if [[ $s == hello ]]; then check dbr1 yes yes; else check dbr1 no yes; fi
  if [[ $s == h* ]]; then check dbr2 yes yes; else check dbr2 no yes; fi
  if [[ $s == "h*" ]]; then check dbr3 no ok; else check dbr3 ok ok; fi
  if [[ $s != world ]]; then check dbr4 yes yes; else check dbr4 no yes; fi
  if [[ $s =~ ^h.*o$ ]]; then check dbr5 yes yes; else check dbr5 no yes; fi
  if [[ abc123 =~ ([a-z]+)([0-9]+) ]]; then
    check dbr6 "${BASH_REMATCH[1]}" abc
    check dbr7 "${BASH_REMATCH[2]}" 123
  else
    check dbr6 no yes
    check dbr7 no yes
  fi
  if [[ $n -eq 42 && $s == hello ]]; then check dbr8 yes yes; else check dbr8 no yes; fi
  if [[ $n -lt 10 || $s == hello ]]; then check dbr9 yes yes; else check dbr9 no yes; fi
  if [[ ! $s == world ]]; then check dbr10 yes yes; else check dbr10 no yes; fi
  if [[ -z "" && -n x ]]; then check dbr11 yes yes; else check dbr11 no yes; fi
  # < and > compare lexically inside [[ ]] (no redirection there)
  if [[ abc < abd ]]; then check dbr12 yes yes; else check dbr12 no yes; fi
  if [[ ( $n == 42 ) ]]; then check dbr13 yes yes; else check dbr13 no yes; fi
  # an unquoted expansion is not word-split inside [[ ]]
  local spaced="a b"
  if [[ $spaced == "a b" ]]; then check dbr14 yes yes; else check dbr14 no yes; fi
  # BASH_REMATCH has one entry per capture PAIR, the whole match included
  if [[ abc123 =~ ^([a-z]+)([0-9]+)$ ]]; then check dbr15 "${#BASH_REMATCH[@]}" 3; else check dbr15 no yes; fi
  # a group that did not participate is still counted, and expands to nothing
  if [[ xy =~ (a)?(x) ]]; then check dbr16 "${#BASH_REMATCH[@]}-[${BASH_REMATCH[1]}]" "3-[]"; else check dbr16 no yes; fi
  # a failed match is status 1 and leaves BASH_REMATCH empty
  [[ hello =~ ^z ]]
  check dbr17 "$?" 1
  check dbr18 "${#BASH_REMATCH[@]}" 0
  # a pattern that does not compile is status 2, and the shell carries on
  local bad_re="("
  [[ x =~ $bad_re ]] 2>/dev/null
  check dbr19 "$?" 2
  # ^ and $ anchor the whole subject (no multiline), and matching is case sensitive
  [[ ABC =~ ^abc$ ]]
  check dbr20 "$?" 1
  # backtracking into an alternation, and a match that must reach the end
  if [[ foobar =~ ^(foo|foobar)bar$ ]]; then check dbr21 "${BASH_REMATCH[1]}" foo; else check dbr21 no yes; fi
}

# ===== SECTION 12: the test builtin =====
s12() {
  if [ 5 -eq 5 ] && [ 5 -ne 6 ] && [ 3 -lt 4 ] && [ 4 -le 4 ]; then
    check tst1 yes yes
  else
    check tst1 no yes
  fi
  if [ 9 -gt 2 ] && [ 9 -ge 9 ]; then check tst2 yes yes; else check tst2 no yes; fi
  if [ abc = abc ] && [ abc != xyz ]; then check tst3 yes yes; else check tst3 no yes; fi
  if [ -z "" ] && [ -n x ]; then check tst4 yes yes; else check tst4 no yes; fi
  if [ ! -z x ]; then check tst5 yes yes; else check tst5 no yes; fi
  if [ 1 -eq 1 -a 2 -eq 2 ]; then check tst6 yes yes; else check tst6 no yes; fi
  if [ 1 -eq 2 -o 2 -eq 2 ]; then check tst7 yes yes; else check tst7 no yes; fi
  if [ \( 1 -eq 1 \) ]; then check tst8 yes yes; else check tst8 no yes; fi
  # 'test' is the same builtin spelled without brackets
  if test 1 -eq 1; then check tst9 yes yes; else check tst9 no yes; fi
  # a bare one-argument test is true when the argument is non-empty
  if [ x ]; then check tst10 yes yes; else check tst10 no yes; fi
  if [ "" ]; then check tst11 no ok; else check tst11 ok ok; fi
  # /dev/null exists and is not a regular file
  if [ -e /dev/null ]; then check tst12 yes yes; else check tst12 no yes; fi
  if [ -d /dev/null ]; then check tst13 no ok; else check tst13 ok ok; fi
}

# ===== SECTION 13: case =====
s13() {
  s13kind() {
    case "$1" in
      "") echo empty ;;
      [0-9]) echo digit ;;
      [0-9]*) echo number ;;
      a|b|c) echo abc ;;
      *.txt) echo text ;;
      ?) echo single ;;
      *) echo other ;;
    esac
  }
  check cas1 "$(s13kind '')" empty
  check cas2 "$(s13kind 7)" digit
  check cas3 "$(s13kind 42x)" number
  check cas4 "$(s13kind b)" abc
  check cas5 "$(s13kind notes.txt)" text
  check cas6 "$(s13kind z)" single
  check cas7 "$(s13kind hello)" other
  # ;;& falls through to test the remaining patterns
  local acc=""
  case abc in
    a*) acc="${acc}1" ;;&
    *c) acc="${acc}2" ;;
    *) acc="${acc}3" ;;
  esac
  check cas8 "$acc" 12
  # a case with a bracket negation
  case q in
    [!a-p]) check cas9 yes yes ;;
    *) check cas9 no yes ;;
  esac
}

# ===== SECTION 14: loops =====
s14() {
  local acc=""
  local i
  for (( i = 0; i < 4; i++ )); do acc="$acc$i"; done
  check lop1 "$acc" 0123
  acc=""
  local n=3
  until [ $n -eq 0 ]; do acc="$acc$n"; n=$(( n - 1 )); done
  check lop2 "$acc" 321
  acc=""
  for w in a b c d; do
    if [ "$w" = b ]; then continue; fi
    if [ "$w" = d ]; then break; fi
    acc="$acc$w"
  done
  check lop3 "$acc" ac
  # break N leaves both loops
  acc=""
  local x y
  for x in 1 2; do
    for y in a b; do
      acc="$acc$x$y"
      if [ "$y" = a ]; then break 2; fi
    done
  done
  check lop4 "$acc" 1a
  # a for loop with no 'in' iterates over the positional parameters
  s14args() {
    local out=""
    for a; do out="$out$a"; done
    echo "$out"
  }
  check lop5 "$(s14args p q r)" pqr
  # a while loop reading from a here-doc
  acc=""
  local line
  while read -r line; do acc="$acc[$line]"; done <<EOF
one
two
EOF
  check lop6 "$acc" "[one][two]"
}

# ===== SECTION 15: functions =====
s15() {
  s15add() { echo $(( $1 + $2 )); }
  check fun1 "$(s15add 2 3)" 5
  # the 'function' keyword form, with and without ()
  function s15twice { echo $(( $1 * 2 )); }
  check fun2 "$(s15twice 21)" 42
  # 'local' really is local
  s15outer() {
    local v=inner
    echo "$v"
  }
  v=outer
  check fun3 "$(s15outer)" inner
  check fun4 "$v" outer
  # return sets $?
  s15ok() { return 0; }
  s15bad() { return 3; }
  s15ok
  check fun5 "$?" 0
  s15bad
  check fun6 "$?" 3
  # recursion
  s15fact() {
    if [ "$1" -le 1 ]; then echo 1; return; fi
    local sub
    sub=$(s15fact $(( $1 - 1 )))
    echo $(( $1 * sub ))
  }
  check fun7 "$(s15fact 5)" 120
  # $FUNCNAME names the running function
  s15who() { echo "$FUNCNAME"; }
  check fun8 "$(s15who)" s15who
  # a function can be called with more arguments than it names
  s15count() { echo $#; }
  check fun9 "$(s15count a b c d e)" 5
}

# ===== SECTION 16: positional parameters =====
s16() {
  s16show() {
    echo "$#|$1|$2|$*"
  }
  check pos1 "$(s16show a b)" "2|a|b|a b"
  s16shift() {
    shift
    echo "$1"
  }
  check pos2 "$(s16shift a b c)" b
  s16shift2() {
    shift 2
    echo "$#:$1"
  }
  check pos3 "$(s16shift2 a b c d)" "2:c"
  # "$@" keeps the arguments separate, "$*" joins them
  s16at() {
    local -a copy=("$@")
    echo "${#copy[@]}"
  }
  check pos4 "$(s16at 'a b' c)" 2
  s16star() {
    local joined="$*"
    echo "$joined"
  }
  check pos5 "$(s16star 'a b' c)" "a b c"
  # set -- rebinds the positional parameters
  set -- x y z
  check pos6 "$#" 3
  check pos7 "$2" y
  set --
  check pos8 "$#" 0
  # ${10} and beyond need braces
  s16tenth() { echo "${10}"; }
  check pos9 "$(s16tenth 1 2 3 4 5 6 7 8 9 TEN)" TEN
}

# ===== SECTION 17: pipelines and lists =====
s17() {
  # a pipeline's status is the LAST command's
  s17false() { return 1; }
  s17true() { return 0; }
  s17false | s17true
  check pip1 "$?" 0
  s17true | s17false
  check pip2 "$?" 1
  # ! negates a pipeline's status
  ! s17false
  check pip3 "$?" 0
  # data really flows through the pipe
  check pip4 "$(echo hello | while read -r l; do echo "[$l]"; done)" "[hello]"
  check pip5 "$(printf 'a\nb\n' | while read -r l; do printf '%s.' "$l"; done)" "a.b."
  # PIPESTATUS records every stage
  s17false | s17true | s17false
  local -a ps=("${PIPESTATUS[@]}")
  check pip6 "${ps[0]}" 1
  check pip7 "${ps[1]}" 0
  check pip8 "${ps[2]}" 1
  # && and || short-circuit on status, and bind left to right
  local flag=untouched
  s17false && flag=ran
  check pip9 "$flag" untouched
  s17false || flag=ran
  check pip10 "$flag" ran
  # a ; list runs both regardless
  local a=0 b=0
  a=1; b=2
  check pip11 "$a$b" 12
}

# ===== SECTION 18: redirection and here-documents =====
s18() {
  # stdout to /dev/null is silent, and the command still succeeds
  echo swallowed > /dev/null
  check red1 "$?" 0
  # 2>&1 merges stderr into stdout, so the substitution captures it
  s18err() { echo oops >&2; }
  check red2 "$(s18err 2>&1)" oops
  check red3 "$(s18err 2>/dev/null)" ""
  # a here-document, with and without expansion
  local v=VAL
  check red4 "$(cat_stub <<EOF
plain $v
EOF
)" "plain VAL"
  check red5 "$(cat_stub <<'EOF'
raw $v
EOF
)" 'raw $v'
  # <<- strips leading TABS from the body and the delimiter
  check red6 "$(cat_stub <<-EOF
	indented
	EOF
)" "indented"
  # a here-string
  local got
  read -r got <<< "here string"
  check red7 "$got" "here string"
  # input redirection from /dev/null gives EOF at once
  local none=unset
  read -r none < /dev/null
  check red8 "$?" 1
  # a redirection on a compound command applies to the whole thing
  local acc=""
  while read -r l; do acc="$acc$l"; done < /dev/null
  check red9 "$acc" ""
}
# cat_stub reads stdin and echoes it, so section 18 needs no external cat.
cat_stub() {
  local line first=1 out=""
  while IFS= read -r line; do
    if [ $first -eq 1 ]; then out="$line"; first=0; else out="$out
$line"; fi
  done
  printf '%s' "$out"
}

# ===== SECTION 19: subshells and command groups =====
s19() {
  local v=outer
  # a subshell gets a copy: assignments inside do not escape
  ( v=inner )
  check sub1 "$v" outer
  # a brace group runs in the current shell: assignments DO escape
  { v=grouped; }
  check sub2 "$v" grouped
  # a subshell's exit status is the last command's
  ( true )
  check sub3 "$?" 0
  ( false )
  check sub4 "$?" 1
  # exit inside a subshell leaves only the subshell
  ( exit 7 )
  check sub5 "$?" 7
  # a group can be redirected as a unit
  check sub6 "$( { echo a; echo b; } )" "a
b"
  # $BASHPID differs inside a subshell; $$ does not
  local outer_pid=$$
  local inner_pid
  inner_pid=$( echo $$ )
  check sub7 "$inner_pid" "$outer_pid"
  # a subshell inherits the variables it does not change
  check sub8 "$( echo "$v" )" grouped
}

# ===== SECTION 20: exit status, set -e and traps =====
s20() {
  # set -e makes the shell leave on the first failing command
  local out
  out=$( set -e; false; echo reached )
  check err1 "$out" ""
  out=$( set -e; true; echo reached )
  check err2 "$out" reached
  # a failing command in a condition does NOT trip set -e
  out=$( set -e; if false; then :; fi; echo reached )
  check err3 "$out" reached
  # set +e turns it back off
  out=$( set -e; set +e; false; echo reached )
  check err4 "$out" reached
  # an EXIT trap runs when the (sub)shell ends
  check err5 "$( trap 'echo bye' EXIT; echo hi )" "hi
bye"
  # set -u makes an unset variable an error
  out=$( { set -u; echo "${undefined_on_purpose}"; echo reached; } 2>/dev/null )
  check err6 "$out" ""
  # set -o pipefail makes a pipeline report the first failure
  local st
  st=$( set -o pipefail; false | true; echo $? )
  check err7 "$st" 1
  st=$( false | true; echo $? )
  check err8 "$st" 0
}

# ===== SECTION 21: printf, read and echo =====
s21() {
  check prf1 "$(printf 'plain')" plain
  check prf2 "$(printf '%s-%s' a b)" a-b
  check prf3 "$(printf '%d' 42)" 42
  check prf4 "$(printf '%5s|' hi)" "   hi|"
  check prf5 "$(printf '%-5s|' hi)" "hi   |"
  check prf6 "$(printf '%03d' 7)" 007
  check prf7 "$(printf '%s\n' one two)" "one
two"
  check prf8 "$(printf '%%')" "%"
  # printf -v assigns instead of printing
  local dest
  printf -v dest '%s=%s' k v
  check prf9 "$dest" "k=v"
  # read splits on IFS and the last name takes the rest
  local a b rest
  read -r a b rest <<< "one two three four"
  check prf10 "$a" one
  check prf11 "$b" two
  check prf12 "$rest" "three four"
  # read -a fills an array
  local -a words
  read -r -a words <<< "x y z"
  check prf13 "${#words[@]}" 3
  check prf14 "${words[2]}" z
  # echo -n suppresses the newline, echo -e interprets escapes
  check prf15 "$(echo -n abc)" abc
  check prf16 "$(echo -e 'a\tb')" "a	b"
}

# ===== SECTION 22: declarations, eval and misc builtins =====
s22() {
  # declare/typeset with attributes
  declare -i num=10
  num=num+5
  check dec1 "$num" 15
  declare -r frozen=fixed
  check dec2 "$frozen" fixed
  # -u/-l force case on assignment
  declare -u shout=quiet
  check dec3 "$shout" QUIET
  declare -l whisper=LOUD
  check dec4 "$whisper" loud
  # export marks a variable for the environment; the value stays readable
  export exported=here
  check dec5 "$exported" here
  check dec6 "$( echo "$exported" )" here
  # unset removes a variable
  local gone=x
  unset gone
  check dec7 "${gone-absent}" absent
  # eval re-parses its argument
  local name=dyn value=42
  eval "$name=$value"
  check dec8 "$dyn" 42
  check dec9 "$(eval echo 'a b')" "a b"
  # ${var@Q}-style operators and command grouping in an assignment
  local literal='a b'
  check dec10 "${#literal}" 3
  # a command can be prefixed with a temporary assignment
  s22show() { echo "$TMPVAR"; }
  check dec11 "$(TMPVAR=once s22show)" once
  check dec12 "${TMPVAR-unset}" unset
  # : is the null command and always succeeds
  :
  check dec13 "$?" 0
  # shift/getopts-free option walking with a while/case
  local -a argv=(-a -b file)
  local opts=""
  local arg
  for arg in "${argv[@]}"; do
    case "$arg" in
      -*) opts="$opts$arg" ;;
      *) break ;;
    esac
  done
  check dec14 "$opts" "-a-b"
}

# ===== SECTION 23: interpreter/compiler agreement ratchet =====
# Character-counting parameter expansion. ${#s} and ${s:o:n} count CHARACTERS in
# bash; this file's grammars used to disagree three ways on non-ASCII text - real
# bash 5.3 said 3 for "a😀b", bash-interpreter.abnf said 4 (UTF-16 code units)
# and bash-to-llvm-ir.abnf said 6 (bytes). ${a[i]:o:n} sliced the element LIST in
# the compiler instead of the element. Every value below is what
# /opt/homebrew/bin/bash 5.3 prints (in a UTF-8 locale).
s23() {
  local s="a😀b"
  check uni1 "${#s}" 3
  check uni2 "${s:1:1}" "😀"
  check uni3 "${s:0:2}" "a😀"
  check uni4 "${s:2}" "b"
  check uni5 "${s: -1}" "b"
  check uni6 "${s:1:-1}" "😀"
  local u="ünïcödé"
  check uni7 "${#u}" 7
  check uni8 "${u:2:3}" "ïcö"
  check uni9 "${u:0:1}" "ü"
  local e=""
  check uni10 "${#e}" 0
  check uni11 "${s:9}" ""
  check uni12 "${s:0:0}" ""
  local arr=(one "thr😀e")
  check uni13 "${#arr[1]}" 5
  check uni14 "${arr[1]:1:2}" "hr"
  check uni15 "${#arr[@]}" 2
}
# ===== SECTION 24: background lists, word "#", and word-shaped function names =====
# Three constructs the grammars used to refuse or mis-read. Every expected value
# below was produced by running the same code under GNU bash 5.3.15 first.
#
#   - `LIST &` : "&" both marks a list asynchronous AND separates statements, so
#     `a & b`, `a &` and `a &<newline>b` are all one list. LIMIT (both halves):
#     there is no second process here, so the statement runs to completion in a
#     subshell and reports status 0 - `cmd & wait` sees exactly what bash sees,
#     concurrency does not, so nothing below depends on overlap.
#   - "#" only opens a comment at the START of a word: bash prints a#b for
#     `echo a#b`, and 64#a is an arithmetic base-N literal, not a comment.
#   - a function NAME is a word, not an identifier: show-len, ble/foo, py-repr.
s24-hyphen() { echo "hy-$1"; }
s24/slash() { echo "sl-$1"; }
s24sub() ( echo "in-sub"; )
s24() {
  # "#" inside a word is literal
  check bg1 "$(echo a#b)" "a#b"
  local hv=y#z
  check bg2 "$hv" "y#z"
  check bg3 "$(( 64#a ))" 10
  check bg4 "$(( 2#1011 ))" 11

  # word-shaped function names, and a subshell function body
  check bg5 "$(s24-hyphen q)" "hy-q"
  check bg6 "$(s24/slash r)" "sl-r"
  check bg7 "$(s24sub)" "in-sub"

  # `&` as a statement terminator and as a separator
  local out
  out="$( { echo one & wait; } )"
  check bg8 "$out" "one"
  # "&" as a SEPARATOR between two statements. Only the count is asserted: real
  # bash runs the two concurrently, so their order is not defined - what is being
  # ratcheted here is that `a & b` is one statement list, not a syntax error.
  out="$( { echo two & echo three; wait; } )"
  check bg9 "${#out}" 9
  # a group and a subshell may be backgrounded
  out="$( { { echo grp; } & wait; } )"
  check bg10 "$out" "grp"
  out="$( { ( echo sub ) & wait; } )"
  check bg11 "$out" "sub"
  # inside a loop body and a case body
  out="$( { for i in 1 2; do echo "i=$i" & done; wait; } )"
  check bg12 "$out" "i=1
i=2"
  out="$( { case x in x) echo c1 & ;; esac; wait; } )"
  check bg13 "$out" "c1"
  # a backgrounded statement runs in a SUBSHELL: the parent keeps its value
  local v=1
  v=2 &
  wait
  check bg14 "$v" 1
  # `wait` with nothing to wait for succeeds
  wait
  check bg15 $? 0
}
# ===== END SECTIONS =====

s01   # SECTION-CALL 01
s02   # SECTION-CALL 02
s03   # SECTION-CALL 03
s04   # SECTION-CALL 04
s05   # SECTION-CALL 05
s06   # SECTION-CALL 06
s07   # SECTION-CALL 07
s08   # SECTION-CALL 08
s09   # SECTION-CALL 09
s10   # SECTION-CALL 10
s11   # SECTION-CALL 11
s12   # SECTION-CALL 12
s13   # SECTION-CALL 13
s14   # SECTION-CALL 14
s15   # SECTION-CALL 15
s16   # SECTION-CALL 16
s17   # SECTION-CALL 17
s18   # SECTION-CALL 18
s19   # SECTION-CALL 19
s20   # SECTION-CALL 20
s21   # SECTION-CALL 21
s22   # SECTION-CALL 22
s23   # SECTION-CALL 23
s24   # SECTION-CALL 24
echo "full: $checks checks, $fails failures"
exit $fails
