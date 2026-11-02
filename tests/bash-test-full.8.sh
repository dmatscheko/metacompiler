#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 8
# of 8, sections 154 on; a new section goes here. The split is explained in
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

# ===== SECTION 154: readonly through a function's computed or circular nameref =====
# A for loop's nameref control variable is the nameref's own store: only its own readonly
# refuses it, except through a circular chain to a readonly global, which warns and refuses.
# A prefix assignment, read -a, mapfile and eval through a nameref to a readonly variable are
# refused; readonly and export of a computed nameref go to its target; declare of a value
# through one is refused naming the target, status 1. Every expected value is bash 5.3's.
s154() {
  local e
  s154s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; case $l in "warning: w: circular name reference") l=c ;; "warning: w: maximum nameref depth (8) exceeded") l=m ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s154a() { local -n w=w; readonly w; echo "-"; for w in p; do echo in; done; echo "y $?"; }
  s154b() { local -n w=$1; readonly w; echo "-"; for w in p; do echo "in $s154z"; done; echo "y $?"; }
  s154c() { local -nr w=s154z; for w in p; do echo "in $w"; done; echo "k $?"; declare -p w; }
  e=$( (s154a) 2>&1; (s154b s154z) 2>&1; (s154c) 2>&1)
  check cnr1 "$(s154s "$e")" 'local: warning: w: circular name reference;c;c;m;-;m;w: readonly variable;y 1;-;in ;y 0;w: readonly variable;k 1;declare -nr w="s154z";'
  s154d() { local -n w=$1; readonly w; echo "-"; w=q echo x; echo "y $?"; }
  s154f() { local -n w=$1; readonly w; echo "-"; read -a w <<< "p q"; echo "y $?"; }
  s154g() { local -n w=$1; readonly w; echo "-"; mapfile w <<< "p"; echo "y $?"; }
  s154h() { local -n w=$1; readonly w; echo "-"; eval 'w=q'; echo "y $?"; }
  e=$( (s154d s154z) 2>&1; (s154f s154z) 2>&1; (s154g s154z) 2>&1; (s154h s154z) 2>&1; echo "s $?")
  check cnr2 "$(s154s "$e")" '-;w: readonly variable;x;y 0;-;w: readonly variable;y 1;-;w: readonly variable;y 1;-;s154z: readonly variable;s 1;'
  s154k() { local -n w=$1; readonly w; export w; declare -p w s154y; }
  s154m() { local -n w=$1; declare w=q; echo "y $?"; }
  e=$( (s154y=1; s154k s154y) 2>&1; (readonly s154x=1; s154m s154x; echo "z $?") 2>&1)
  check cnr3 "$(s154s "$e")" 'declare -n w="s154y";declare -rx s154y="1";declare: s154x: readonly variable;y 1;z 0;'
}
# ===== SECTION 155: ${w[SUB]:=WORD} and element stores through a circular nameref =====
# Through a function's nameref that refers to itself, the global variable takes the element,
# adopting its scalar value as element 0, with the lookups' warnings; a readonly global refuses
# it. ${w[@]:=WORD} of a blank indexed array is a bad subscript that abandons the line. read
# of NAME[SUB] stores the same way. Every expected value is bash 5.3's.
s155() {
  local e
  s155s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; case $l in "warning: w: circular name reference") l=c ;; "warning: w: maximum nameref depth (8) exceeded") l=m ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s155a() { local -n w=w; : ${w[1]:=q}; echo "y $?"; }
  s155b() { local -n w=w; readonly w; : ${w[1]:=q}; echo "y $?"; }
  e=$( (w=g; s155a; declare -p w) 2>&1; (w=g; s155b; echo no) 2>&1; echo "s $?")
  check cnd1 "$(s155s "$e")" 'local: warning: w: circular name reference;c;c;c;c;c;y 0;declare -a w=([0]="g" [1]="q");local: warning: w: circular name reference;c;c;c;c;c;c;w: readonly variable;s 2;'
  s155c() { local -n w=w; : ${w[@]:=q}; echo "y $?"; }
  s155d() { local -n w=$1; : ${w[@]:=q}; echo "y $?"; }
  e=$( (s155c) 2>&1; echo "s $?"; (s155d s155z) 2>&1; echo "t $?")
  check cnd2 "$(s155s "$e")" 'local: warning: w: circular name reference;c;c;c;c;w[@]: bad array subscript;s 2;w[@]: bad array subscript;t 2;'
  s155e() { local -n w=w; w[1]=q; }
  s155f() { local -n w=w; read 'w[2]' <<< z; }
  e=$( (w=g; s155e; declare -p w) 2>/dev/null; (w=g; s155f; declare -p w) 2>/dev/null)
  check cnd3 "${e//$'\n'/;}" 'declare -a w=([0]="g" [1]="q");declare -a w=([0]="g" [2]="z")'
}
# ===== SECTION 156: prefix, array and localizing stores through a nameref =====
# A prefix assignment through a circular nameref binds a plain variable for the command and
# leaves the nameref as it was; an array or element store to a readonly target names the
# nameref; a function's localizing declaration through its nameref to NAME[SUB] does nothing,
# status 0. Every expected value is bash 5.3's.
s156() {
  local e
  s156s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; case $l in "warning: w: circular name reference") l=c ;; "warning: w: maximum nameref depth (8) exceeded") l=m ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s156a() { local -n w=w; w=q echo x; echo "x $w"; declare -p w; }
  e=$( (s156a) 2>&1)
  check cnp1 "$(s156s "$e")" 'local: warning: w: circular name reference;c;c;x;c;x ;declare -n w="w";'
  s156b() { local -n w=s156r; w=(p); echo "y $?"; }
  s156c() { local -n w=s156r; w[1]=p; echo "y $?"; }
  e=$( (readonly s156r=1; s156b) 2>&1; (readonly s156r=1; s156c) 2>&1)
  check cnp2 "$(s156s "$e")" 'w: readonly variable;w: readonly variable;'
  s156d() { local -n w=$1; local -i w=3; echo "a $?"; declare -p w; declare -p s156q; }
  s156e() { local -n w=$1; local w=3; echo "b $?"; declare -p s156q; }
  s156f() { local -n w=s156q[2]; local -i w=3; echo "c $?"; declare -p s156q; }
  e=$( (s156q=(1); s156d 's156q[2]'; s156e 's156q[2]'; s156f) 2>&1)
  check cnp3 "$(s156s "$e")" 'a 0;declare -n w="s156q[2]";declare -a s156q=([0]="1");b 0;declare -a s156q=([0]="1");c 0;declare -a s156q=([0]="1");'
}
# ===== SECTION 157: declarations, readonly and run-time stores through a computed nameref =====
# A readonly given through a function's computed nameref holds for stores by the variable's
# own name and for eval's and arithmetic's run-time stores; export and readonly refuse a
# nameref to NAME[SUB] as an identifier and store through it, other declarations outside a
# function give their attributes to the array; a function's `local -n w=''` still makes w a
# plain local. Every expected value is bash 5.3's.
s157() {
  local e
  s157s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; case $l in "warning: w: circular name reference") l=c ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s157a() { local -n w=$1; readonly w; }
  e=$( (s157x=4; s157a s157x; s157x=5; echo "a $? $s157x") 2>&1; (s157x=4; s157a s157x; (( s157x = 6 )); echo "b $? $s157x") 2>&1)
  check crn1 "$(s157s "$e")" 's157x: readonly variable;s157x: readonly variable;b 1 4;'
  s157b() { local -n w=w; w=q declare -p w; }
  s157c() { local -n w=$1; w=q declare -p w; declare -p w; }
  e=$( (s157b) 2>&1; s157q=(1 2); s157c 's157q[1]' 2>&1)
  check crn2 "$(s157s "$e")" 'local: warning: w: circular name reference;c;c;declare -x w="q";declare -x w="q";declare -n w="s157q[1]";'
  s157d() { local -n w=$1; declare -r w; declare -p w; }
  s157e() { local -n w=$1; readonly w; export w=5; echo "d $?"; declare -p w s157q; }
  s157f() { local -n w=$1; declare -rx w=3; declare -p s157y; }
  e=$( (s157q=(1 2 3); s157d 's157q[2]'; s157e 's157q[2]') 2>&1; (s157y=1; s157f s157y; declare -p s157y) 2>&1)
  check crn3 "$(s157s "$e")" 'declare -n w="s157q[2]";readonly: `s157q[2]'"'"': not a valid identifier;export: `s157q[2]'"'"': not a valid identifier;d 0;declare -n w="s157q[2]";declare -a s157q=([0]="1" [1]="2" [2]="5");declare -rx s157y="3";declare -- s157y="1";'
  e=$( (s157q=(1 2 3); declare -g -n r='s157q[2]'; export r=5; declare -g -i r=2+3; declare -g -u r; declare -p s157q r) 2>&1; (s157q=(1 2 3); declare -g -n r='s157q[2]'; declare -g -r r=4; echo "e $?"; declare -p s157q) 2>&1)
  check crn4 "$(s157s "$e")" 'export: `s157q[2]'"'"': not a valid identifier;declare -aiu s157q=([0]="1" [1]="2" [2]="5");declare -n r="s157q[2]";s157q: readonly variable;e 0;declare -ar s157q=([0]="1" [1]="2" [2]="3");'
  e=$( (s='s157y'; declare -g -n r=$s; declare -g -i r=2+3; readonly r; declare -p r s157y) 2>&1)
  check crn5 "$(s157s "$e")" 'declare -n r="s157y";declare -ir s157y="5";'
  s157i() { local -n w=$1; declare -r w; declare -p w s157y; }
  e=$( (s157y=1; s157i s157y) 2>&1)
  check crn8 "$(s157s "$e")" 'declare -n w="s157y";declare -r s157y;'
  s157j() { local -n w=$1; eval "$2"; echo "f $?"; }
  s157g() { local -n w=$1; (( $2 )); echo "g $? $s157z"; }
  e=$( (readonly s157z=1; s157j s157z 'w=8') 2>&1; (readonly s157z=1; s157g s157z 'w = 9') 2>&1; (s157a s157y; s='s157y=7'; eval "$s"; echo no) 2>&1)
  check crn6 "$(s157s "$e")" 's157z: readonly variable;s157z: readonly variable;g 1 1;s157y: readonly variable;'
  s157h() { local -n w=''; echo "h $?"; declare -p w; w=5; }
  e=$( (s157h; echo "${w-unset}") 2>&1)
  check crn7 "$(s157s "$e")" 'local: `'"'"': not a valid identifier;h 1;declare -- w;unset;'
  local -r s157r=1
  local s157o= s157t= s157u=
  s157k() { local -n w=$1; eval "$2"; s157o="$s157o k $?"; }
  s157k s157r 'w=8' 2>/dev/null; s157o="$s157o l $?"
  e=$'s157t=1\ns157r=2\ns157u=3'; eval "$e" 2>/dev/null; s157o="$s157o m $? $s157t $s157u"
  e='s157r=2'; eval "$e" 2>/dev/null || s157o="$s157o n $?"
  check crn9 "$s157o" ' k 1 l 0 m 0 1 3 n 1'
}
# ===== SECTION 158: stores refused by a readonly known only when they run, prefix declarations =====
# read, declare, local, unset, printf -v, for and a prefix assignment each refuse a variable
# made readonly through a function's nameref, or by a function defined after them; a bare
# `local` of a function's own readonly local is no error. Before a declaration builtin, a
# refused readonly prefix name leaves only its own operand as written, and through a nameref
# the binding exports the variable it names, while export and readonly drop the nameref
# operand. Every expected value is bash 5.3's.
s158() {
  local e
  s158s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s158a() { local -n w=$1; readonly w; }
  e=$( (s158x=4; s158a s158x; read s158x <<< 9; echo "a $? $s158x"; declare s158x=5; echo "b $? $s158x"
        s158b() { local s158x=6; echo "c $? $s158x"; }; s158b; unset s158x; echo "d $? $s158x"
        printf -v s158x %s 7; echo "e $? $s158x"; for s158x in 1; do echo no; done; echo "f $? $s158x"
        s158x=9 true; echo "g $? $s158x") 2>&1)
  check cro1 "$(s158s "$e")" 's158x: readonly variable;a 1 4;declare: s158x: readonly variable;b 1 4;local: s158x: readonly variable;c 1 4;unset: s158x: cannot unset: readonly variable;d 1 4;s158x: readonly variable;e 1 4;s158x: readonly variable;f 1 4;s158x: readonly variable;g 0 4;'
  s158c() { read s158y <<< 9; echo "h $? $s158y"; read -a s158y <<< 8; echo "i $? $s158y"; unset s158y; echo "j $? $s158y"; printf -v 's158y[1]' %s 7; echo "k $?"; }
  s158d() { for s158y in 1; do echo no; done; echo "l $? $s158y"; s158y=9 true; echo "m $? $s158y"; }
  e=$( (s158y=4; readonly s158y; s158c; s158d) 2>&1)
  check cro2 "$(s158s "$e")" 's158y: readonly variable;h 1 4;s158y: readonly variable;i 1 4;unset: s158y: cannot unset: readonly variable;j 1 4;s158y: readonly variable;k 1;s158y: readonly variable;l 1 4;s158y: readonly variable;m 0 4;'
  s158e() { local v=1; s158f; local v; echo "n $? $v"; declare v; echo "o $? $v"; }
  s158f() { readonly v; }
  e=$( (s158e) 2>&1)
  check cro3 "$(s158s "$e")" 'n 0 1;o 0 1;'
  s158g() { s158z=8 t=3 local t; echo "p $? [$t]"; s158z=8 t=5 declare t; echo "q $? [$t]"; s158z=9 u=2 local s158z u; echo "r $? [$u]"; }
  e=$( (s158z=4; readonly s158z; s158g; s158z=9 t=1 export s158z t; echo "s $? [$t]"; declare -p s158z t) 2>&1)
  check cro4 "$(s158s "$e")" 's158z: readonly variable;p 0 [3];s158z: readonly variable;q 0 [5];s158z: readonly variable;local: s158z: readonly variable;r 1 [2];s158z: readonly variable;s 0 [1];declare -rx s158z="4";declare -x t="1";'
  s158h() { local -n w=s158q; w=k export w; echo "t $?"; declare -p s158q w; }
  s158i() { local -n w=s158q; w=k readonly w; echo "u $?"; declare -p s158q; }
  s158j() { local -n w=s158q; w=k declare w; echo "v $?"; declare -p s158q w; }
  s158k() { local -n w=s158q; w=k local -r w; echo "w $?"; declare -p s158q; }
  e=$( (s158q=1; s158h; s158i; s158j; s158k; declare -n g=s158q; g=k declare -x g; declare -p s158q g) 2>&1)
  check cro5 "$(s158s "$e")" 't 0;declare -- s158q="1";declare -n w="s158q";u 0;declare -- s158q="1";v 0;declare -x s158q="k";declare -n w="s158q";w 0;declare -rx s158q="k";declare -x s158q="k";declare -n g="s158q";'
}
# ===== SECTION 159: hidden NAME[SUB] locals, export through a prefix-bound nameref =====
# A function's declare, local or typeset through its nameref whose chain ends at NAME[SUB]
# makes a local literally named NAME[SUB]: every later lookup through a chain ending there
# finds it in place of the element (reads, stores, +=, arithmetic, -i -u -x -r, unset,
# local -p, ${REF@a}, declare -p NAME[SUB]), while ${!r} still reads the element. export and
# readonly NAME=VALUE through a prefix-bound nameref store through a function's local
# nameref with no attribute and change nothing through a global one; a prefix name readonly
# only at run time keeps its array before eval. Every expected value is bash 5.3's.
s159() {
  local e
  s159s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  local -a s159a=(a b c d)
  s159f() { local -n w=$1; declare w; w=5; echo "${s159a[2]} $w"; local -p; w+=x; local r='s159a[2]'; echo "$w ${!r} ${#w} ${w:1} ${w@a}"; [[ -v w ]] && echo set; unset w; echo "[$w] [${w-u}]"; w=4; local -n v=$1; echo "$v"; declare -p 's159a[2]'; }
  e=$(s159f 's159a[2]'; echo "${s159a[*]}")
  check chl1 "$(s159s "$e")" 'c 5;declare -- s159a[2]="5";declare -n w="s159a[2]";5x c 2 x ;set;[] [u];4;declare -- s159a[2]="4";a b c d;'
  s159g() { local -n w=$1; declare -i w; w=2*3; echo "$w ${s159a[1]}"; w+=1; (( w += 2 )); echo "$w $(( w * 2 ))"; local -u w=ab; echo "$w ${w@a}"; }
  e=$(s159g 's159a[1]'; echo "${s159a[*]}")
  check chl2 "$(s159s "$e")" '6 b;9 18;0 iu;a b c d;'
  s159h() { local -n w=$1; declare -r w; export w=5; echo "a $?"; w=9; echo "b $?"; }
  s159i() { local -n w=$1; typeset -x w=4; local -p; echo "${w@A} ${w@a}"; readonly w; unset w; echo "c $?"; }
  e=$( (s159h 's159a[0]'; echo "d $?") 2>&1; s159i 's159a[3]' 2>&1; echo "${s159a[*]}")
  check chl3 "$(s159s "$e")" 's159a[0]: readonly variable;a 1;s159a[0]: readonly variable;declare -x s159a[3]="4";declare -n w="s159a[3]";declare -x s159a[3]='"'4'"' x;unset: s159a[3]: cannot unset: readonly variable;c 1;a b c d;'
  s159j() { local -n w=s159x; w=k export w=7 s159y=1; echo "e $?"; declare -p s159x; }
  s159k() { local s159x=0; local -n w=s159x; w=k readonly w+=8; echo "f $?"; declare -p s159x; }
  e=$( (s159x=1; s159j; declare -p s159x; s159k; declare -n g=s159x; g=k export g=9; echo "g $?"; declare -p s159x
        declare -gn s159n=s159x; s159n=k export s159n=3; echo "h $?"; declare -p s159x) 2>&1)
  check chp1 "$(s159s "$e")" 'e 0;declare -- s159x="7";declare -- s159x="7";f 0;declare -- s159x="08";g 0;declare -- s159x="9";h 0;declare -- s159x="9";'
  s159l() { local -n w=$1; readonly w; }
  e=$( (s159z=(1 2 3); s159l s159z; s159z=5 eval 'declare -p s159z'; echo "h $?"; declare -p s159z) 2>&1)
  check chp2 "$(s159s "$e")" 's159z: readonly variable;declare -ar s159z=([0]="1" [1]="2" [2]="3");h 0;declare -ar s159z=([0]="1" [1]="2" [2]="3");'
}
# ===== END SECTIONS =====

s154  # SECTION-CALL 154
s155  # SECTION-CALL 155
s156  # SECTION-CALL 156
s157  # SECTION-CALL 157
s158  # SECTION-CALL 158
s159  # SECTION-CALL 159
echo "full: $checks checks, $fails failures"
exit $fails
