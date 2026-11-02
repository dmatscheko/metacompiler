#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 6
# of 7, sections 124 to 134. The split is explained in tests/bash-test-full.sh.
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

# ===== SECTION 124: prefix and suffix removal with a star =====
# ${v#p} ${v##p} ${v%p} ${v%%p} for patterns holding a "*": the shortest and the longest
# match, with "?", bracket expressions, escapes and several stars, on a subject longer
# than the pattern many times over. Every expected value is bash 5.3's.
s124() {
  local r p s='/usr/lib/a.b: line 12: x.tar.gz: y' l=''
  local -a ps=('*: line *: ' '*/' '*.' '*.*' '*[ab]' '*[!.]' '*\:' '*?' '*??' '**' '*' '*x*' '*b*' '*z' '*/*/' '?*:' '*[[:digit:]] ')
  for p in "${ps[@]}"; do r="$r|${s#$p}|${s##$p}|${s%$p}|${s%%$p}"; done
  check sst1 "$r" '|x.tar.gz: y|y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|usr/lib/a.b: line 12: x.tar.gz: y|a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|b: line 12: x.tar.gz: y|gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|b: line 12: x.tar.gz: y||/usr/lib/a.b: line 12: x.tar||/a.b: line 12: x.tar.gz: y|r.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|usr/lib/a.b: line 12: x.tar.gz: y||/usr/lib/a.b: line 12: x.tar.gz: || line 12: x.tar.gz: y| y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|usr/lib/a.b: line 12: x.tar.gz: y||/usr/lib/a.b: line 12: x.tar.gz: ||sr/lib/a.b: line 12: x.tar.gz: y||/usr/lib/a.b: line 12: x.tar.gz:||/usr/lib/a.b: line 12: x.tar.gz: y||/usr/lib/a.b: line 12: x.tar.gz: y||/usr/lib/a.b: line 12: x.tar.gz: y||/usr/lib/a.b: line 12: x.tar.gz: y||.tar.gz: y||/usr/lib/a.b: line 12: ||/a.b: line 12: x.tar.gz: y||/usr/lib/a.||: y|: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|lib/a.b: line 12: x.tar.gz: y|a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y| line 12: x.tar.gz: y| y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y|/usr/lib/a.b: line 12: x.tar.gz: y'
  for p in a b c d e f g h; do l="$l$p$p$p$p$p$p$p$p$p$p:"; done
  check sst2 "${l#*h:}|${l##*:}|${l%:*}|${l%%:*}|${l#*d?d}|${l%d*}|${l%%d*}" '||aaaaaaaaaa:bbbbbbbbbb:cccccccccc:dddddddddd:eeeeeeeeee:ffffffffff:gggggggggg:hhhhhhhhhh|aaaaaaaaaa|ddddddd:eeeeeeeeee:ffffffffff:gggggggggg:hhhhhhhhhh:|aaaaaaaaaa:bbbbbbbbbb:cccccccccc:ddddddddd|aaaaaaaaaa:bbbbbbbbbb:cccccccccc:'
}
# ===== SECTION 125: expansion errors in a case word, a case pattern and [[ ]] =====
# An expansion error in the word or a pattern of a case abandons the command, so a
# subshell holding it ends with status 1; a bad substitution names the whole word. The
# patterns are expanded only up to the first that matches, and ;& runs the next body
# without expanding its patterns. In [[ ]] an operand's expansion error abandons the
# command too, while its own arithmetic only fails it; && and || expand only what they
# need. Every expected value is bash 5.3's.
s125() {
  local e f w nl=$'\n' x=abc
  e=$( (case a${#x-a}b in x) ;; esac; echo no) 2>&1; echo "st=$?"); w="$0: line $LINENO: a\${#x-a}b: bad substitution${nl}st=1"
  f=$( (case x in a${#x-a}b) echo m ;; x) echo x ;; esac) 2>&1; echo "st=$?"); e="$e$nl$f"; w="$w$nl$0: line $LINENO: a\${#x-a}b: bad substitution${nl}st=1"
  f=$( (case x in x|a${#x-a}b) echo hit ;; esac) 2>&1; echo "st=$?"); e="$e$nl$f"; w="$w${nl}hit${nl}st=0"
  f=$( (case a${s125u?unset}b in x) ;; esac) 2>&1; echo "st=$?"); e="$e$nl$f"; w="$w$nl$0: line $LINENO: s125u: unset${nl}st=1"
  f=$( (case x in x) echo one ;& y$(echo side >&2)) echo two ;; esac; case x in y|x|$(echo lazy >&2)) echo three ;; esac) 2>&1); e="$e$nl$f"; w="$w${nl}one${nl}two${nl}three"
  check scw1 "$e" "$w"
  e=$( ([[ x == a${#x-a}b ]]; echo no) 2>&1; echo "st=$?"); w="$0: line $LINENO: a\${#x-a}b: bad substitution${nl}st=1"
  f=$( ([[ -n $((1/0)) ]]; echo no) 2>&1; echo "st=$?"); e="$e$nl$f"; w="$w$nl$0: line $LINENO: 1/0: division by 0 (error token is \"0\")${nl}st=1"
  f=$( ([[ 1/0 -eq 1 ]]; echo "in=$?") 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: [[: 1/0: division by 0 (error token is \"0\")${nl}in=1"
  f=$( ([[ x == x || a${#x-a}b == y ]]; echo "a=$?"; [[ x == y && $(echo no >&2) == y ]]; echo "b=$?") 2>&1); e="$e$nl$f"; w="$w${nl}a=0${nl}b=1"
  check scw2 "$e" "$w"
}
# ===== SECTION 126: circular namerefs: how often bash warns =====
# Through a circular chain of namerefs the count and the keys (${#REF[@]}, ${!REF[@]})
# warn once and an element or the elements twice; an @-transform reads an empty value
# after one warning. Inside a function a nameref to itself is declared with two warnings.
# Every expected value is bash 5.3's.
s126() {
  local e f l w nl=$'\n' c=': warning: p: circular name reference'
  e=$( (declare -n p=q q=p; echo "${#p[@]}|${#p[*]}|${!p[@]}|${#p[1]}") 2>&1); l=$LINENO
  w="$0: line $l$c$nl$0: line $l$c$nl$0: line $l$c$nl$0: line $l$c${nl}0|0||0"
  f=$( (declare -n p=q q=p; echo "${p[@]}|${p@Q}|${p@U}") 2>&1); e="$e$nl$f"; l=$LINENO
  w="$w$nl$0: line $l$c$nl$0: line $l$c$nl$0: line $l$c$nl$0: line $l$c$nl||"
  check snc1 "$e" "$w"
  s126f() { local -n l=l; echo "${#l[@]}"; declare -p l; }; l=$LINENO
  e=$( (s126f) 2>&1)
  check snc2 "$e" "$0: line $l: local: warning: l: circular name reference
$0: line $l: warning: l: circular name reference
$0: line $l: warning: l: circular name reference
0
declare -n l=\"l\""
}
# ===== SECTION 127: errors of an arithmetic text's expansions; function attributes =====
# An error raised while an arithmetic text's expansions are expanded is the word's: it
# carries no "((: " prefix and abandons the line, where an error of the evaluation only
# fails (( )). A bad substitution there names the arithmetic text: what (( )) or $(( ))
# enclose, a for (( )) part from its first token, an offset from the ":" on. Through a
# circular nameref @a and @A warn three times, four with a subscript, and a chain that
# closes on a local reads, stores and unsets the global variable of that name. A byte that
# starts no well-formed UTF-8 sequence is a character of its own to a pattern and to case mapping.
# declare -ft gives a function the trace attribute, which declare -F and -pf show. Every
# expected value is bash 5.3's.
s127() {
  local e f l w nl=$'\n' x=abc s=hello b='bad substitution'
  e=$( ( (( $((1/0)) )); echo no) 2>&1); w="$0: line $LINENO: 1/0: division by 0 (error token is \"0\")"
  f=$( (if (( 1 + $((2/0)) )); then echo t; else echo e; fi; echo no) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: 2/0: division by 0 (error token is \"0\")"
  f=$( (for (( i=$((3/0)); i<1; i++ )); do :; done; echo no) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: 3/0: division by 0 (error token is \"0\")"
  f=$( (echo ${s:1+$((4/0))}; echo no) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: 4/0: division by 0 (error token is \"0\")"
  f=$( ( (( 1/0 + $x )); echo "st=$?") 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: ((: 1/0 + abc : division by 0 (error token is \"0 + abc \")${nl}st=1"
  check sxe1 "$e" "$w"
  e=$( ( (( 1+${#x-a}  )); echo no) 2>&1); w="$0: line $LINENO:  1+\${#x-a}  : $b"
  f=$( (echo "a $((${#x-a}+1)) b") 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: \${#x-a}+1: $b"
  f=$( (echo $[ ${#x-a} ]) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO:  \${#x-a} : $b"
  f=$( (for ((  i=${#x-a} ; i<1; i++ )); do :; done) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: i=\${#x-a} : $b"
  f=$( (echo ${s:  ${#x-a}  }) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO:   \${#x-a}  : $b"
  f=$( (echo "${s:1: 1+${#x-a}}") 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO:  1+\${#x-a}: $b"
  f=$( ( (( $'\t'${#x-a} )) ) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO:  '"$'\t'"'\${#x-a} : $b"
  f=$( ([[ 1 -eq "${#x-a}" ]]) 2>&1); e="$e$nl$f"; w="$w$nl$0: line $LINENO: \${#x-a}: $b"
  check sxe2 "$e" "$w"
  local c=': warning: q: circular name reference'
  e=$( (declare -n q=r r=q; echo "[${q@a}]") 2>&1); l=$LINENO
  w="$0: line $l$c$nl$0: line $l$c$nl$0: line $l$c$nl[]"
  f=$( (declare -n q=r r=q; echo "[${q[@]@A}]") 2>&1); e="$e$nl$f"; l=$LINENO
  w="$w$nl$0: line $l$c$nl$0: line $l$c$nl$0: line $l$c$nl$0: line $l$c$nl[]"
  f=$( (s127n() { local -n n; n=n; echo "[$?]"; }; s127n) 2>&1); e="$e$nl$f"; w="$w$nl[0]"
  check sxe3 "$e" "$w"
  s127r() { local -n s127v=s127v; echo "${#s127v[@]} $s127v"; s127v=X; s127v+=Z; printf -v s127v %s "$s127v"P; echo "$s127v"
    for s127v in F1 F2; do :; done; unset s127v; echo "[${s127v-u}]"; }
  s127v=G; e=$(s127r 2>/dev/null; echo "$s127v"); f=$(s127r 2>&1 >/dev/null); w=${f//[!$nl]/}; f=${f//maximum nameref depth/@}; f=${f//[!@]/}
  check snl1 "$e ${#w} ${#f}" $'1 G\nXZP\n[u]\nF2 12 5'
  local v=$'a\x80b' v2=$'ab\x80' v3=$'\xc3b' v4=$'\xc0\xafa'
  check sxb1 "${v#?}|${v##??}|${v%?}|${v//?/-}|${v2%??}|${v2#??}|${v3^^}|${v4^^}|${v2^^}" $'\x80b|b|a\x80|---|a|\x80|\xc3B|\xc0\xafA|AB\x80'
  e=$(s127f() { :; }; s127g() { :; }
    declare -ft s127f s127g nosuch; echo "st=$?"; declare -f +t s127g; declare -pF s127f s127g
    f=$(declare -f s127f); echo "${f##*$nl}"; f=$(declare -pf s127f); echo "${f##*$nl}"
    typeset -tF s127g; echo "st=$?"
    unset -f s127f; s127f() { :; }; declare -pF s127f s127g; declare -pF nosuch 2>/dev/null; echo "st=$?"
    (declare +t -f s127g); declare -F s127g; declare -pF s127g)
  check sft1 "$e" 'st=1
declare -ft s127f
declare -f s127g
}
declare -ft s127f
st=0
declare -f s127f
declare -ft s127g
st=1
s127g
declare -ft s127g'
}
# ===== SECTION 128: circular namerefs in stores and arithmetic; case toggling =====
# Through a nameref chain that closes on a local, an element store, an array literal and an
# arithmetic store land on the global variable of that name and leave the nameref; an
# arithmetic store looks the name up once more than an assignment (one more warning), `let`
# once more again for each NAME[SUB], and outside a function an arithmetic scalar store
# through a circular chain stores nothing. An arithmetic element store makes a set scalar
# element 0. ${v~} and ${v~~} toggle the case of the first / every ASCII letter, a pattern
# picking the characters. -n takes -i and the case attributes away, and fails with -i.
# declare -ft with no name prints only the traced functions, whatever the others hold.
# Every expected value is bash 5.3's.
s128() {
  local e f c='circular name reference' m='maximum nameref depth'
  # s128c TEXT WORD: how many lines TEXT has, and how many of them hold WORD.
  s128c() { local IFS=$'\n' l n=0 wd=$2; set -- $1; for l; do [[ $l == *"$wd"* ]] && n=$(( n + 1 )); done; echo "$# $n"; }
  s128a=(G0 G1)
  s128f() { local -n s128a=s128a; s128a[1]=Q; s128a[3]+=R; echo "${s128a[3]} ${#s128a[@]}"; s128a=(n "${s128a[1]}"); s128a+=(o); declare -p s128a; }
  e=$(s128f 2>/dev/null; declare -p s128a); f=$(s128f 2>&1 >/dev/null)
  check sgc1 "$e $(s128c "$f" "$c")" 'R 3
declare -n s128a="s128a"
declare -a s128a=([0]="n" [1]="Q" [2]="o") 13 13'
  s128g() { local -n s128s=s128s; (( s128s = 5 )); let s128s+=1; (( s128s[2] = 4 )); let 's128s[3]=s128s[2]+1'; local x='s128s[0]++'; let x; echo $(( s128s[3] )) $(( s128s )); }
  s128s=1; e=$(s128g 2>/dev/null; declare -p s128s); f=$(s128g 2>&1 >/dev/null)
  check sgc2 "$e $(s128c "$f" "$m")" '5 7
declare -a s128s=([0]="7" [2]="4" [3]="5") 26 2'
  s128k() { local -n s128d=s128d; echo "${s128d:=DEF} ${s128d=X} ${s128d:-Y}"; }
  unset s128d; e=$(s128k 2>/dev/null; echo "$s128d"); f=$(s128k 2>&1 >/dev/null)
  check sgc3 "$e $(s128c "$f" "$m")" 'DEF DEF DEF
DEF 6 1'
  e=$(declare -n s128q=s128r s128r=s128q; (( s128q = 5 )); echo "st=$?"; (( s128q[1] = 6 )); declare -p s128q; declare -n s128t=s128u s128u=s128t; let 's128t[2]=7'; declare -p s128t)
  f=$( (declare -n s128q=s128r s128r=s128q; (( s128q = 5 )); (( s128q[1] = 6 )); declare -n s128t=s128u s128u=s128t; let 's128t[2]=7') 2>&1)
  check sgc4 "$e $(s128c "$f" "$c")" 'st=0
declare -n s128q="s128r"
declare -n s128t="s128u" 9 8'
  e=$(x=7; (( x[2]=4 )); y=8; let 'y[1]=5'; z=9; s='z[3]=1'; (( $s )); declare -p x y z)
  check sae1 "$e" 'declare -a x=([0]="7" [2]="4")
declare -a y=([0]="8" [1]="5")
declare -a z=([0]="9" [3]="1")'
  local v=hELLo a=(abC Def) u=$'\xc3\xa9Ab' t=aB-1
  check stc1 "${v~} ${v~~} ${v~~[lLb]} ${v~[a-z]} ${a[*]~} ${a[1]~~} ${u~} ${u~~} ${t~~?}" $'HELLo HellO hEllo HELLo AbC def dEF \xc3\xa9Ab \xc3\xa9aB Ab-1'
  e=$(s128m=3; s128n() { local -i n; local -n n=s128m; declare -p n; local -u o=q; local -n o=s128m; declare -p o; local -n -i p=s128m; echo "st=$?"; n=4+1; declare -p s128m; }; s128n)
  check snd1 "$e" 'declare -n n="s128m"
declare -n o="s128m"
st=1
declare -- s128m="4+1"'
  e=$(s128p() { echo "$(case a in a) echo x;; esac)"; }; s128o() { :; }; declare -ft s128o; declare -ft; echo "st=$?")
  check sdf1 "$e" $'s128o () \n{ \n    :\n}\ndeclare -ft s128o\nst=0'
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
# ===== END SECTIONS =====

s124  # SECTION-CALL 124
s125  # SECTION-CALL 125
s126  # SECTION-CALL 126
s127  # SECTION-CALL 127
s128  # SECTION-CALL 128
s129  # SECTION-CALL 129
s130  # SECTION-CALL 130
s131  # SECTION-CALL 131
s132  # SECTION-CALL 132
s133  # SECTION-CALL 133
s134  # SECTION-CALL 134
echo "full: $checks checks, $fails failures"
exit $fails
