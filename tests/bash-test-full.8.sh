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
# ===== SECTION 160: a declaration's option word with a letter it does not take =====
# A leading word starting with - (declare, typeset and local: or +) is an option word
# whatever follows the sign: its first letter the builtin does not take is reported with the
# word's sign ("+=: invalid option" for +x=3, "--" for --x), the usage line follows, and the
# builtin does nothing (status 2), written or computed alike. export's and readonly's +X=V is
# an operand, which is no name. Every expected value is bash 5.3's.
s160() {
  local e
  s160s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$( (export +x=3 s160a=1; echo "a $?"; declare -p s160a; declare +x=3 s160b=1; echo "b $?"; declare -p s160b) 2>&1)
  check ivo1 "$(s160s "$e")" 'export: `+x=3'"'"': not a valid identifier;a 1;declare -x s160a="1";declare: +=: invalid option;declare: usage: declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...];b 2;declare: s160b: not found;'
  e=$( (declare --x s160c; echo "c $?"; typeset -i5 s160c; echo "d $?"; readonly -r=1 s160c; echo "e $?"; declare -p s160c) 2>&1)
  check ivo2 "$(s160s "$e")" 'declare: --: invalid option;declare: usage: declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...];c 2;typeset: -5: invalid option;typeset: usage: typeset [-aAfFgiIlnrtux] name[=value] ... or typeset -p [-aAfFilnrtux] [name ...];d 2;readonly: -r: invalid option;readonly: usage: readonly [-aAf] [name[=value] ...] or readonly -p;e 2;declare: s160c: not found;'
  s160f() { local +i=2 s160d=1; echo "f $?"; local -p; }
  e=$( (local -x=1; echo "g $?"; s160f) 2>&1)
  check ivo3 "$(s160s "$e")" 'local: -=: invalid option;local: usage: local [option] name[=value] ...;g 2;local: +=: invalid option;local: usage: local [option] name[=value] ...;f 2;'
  e=$( (v='+x=3'; declare $v s160e; echo "h $?"; export $v s160e=2; echo "i $?"; v='-y=3 s160g'; declare -x $v; echo "j $?"; declare '-%' s160h; echo "k $?"; declare -p s160e s160g s160h) 2>&1)
  check ivo4 "$(s160s "$e")" 'declare: +=: invalid option;declare: usage: declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...];h 2;export: `+x=3'"'"': not a valid identifier;i 1;declare: -y: invalid option;declare: usage: declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...];j 2;declare: -%: invalid option;declare: usage: declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...];k 2;declare -x s160e="2";declare: s160g: not found;declare: s160h: not found;'
}
# ===== SECTION 161: a callee through its caller's local nameref =====
# A name is looked up through the calling functions, so a function called while its caller's
# `local -n w=x` is in effect reads and stores x through w: assignments, +=, arithmetic,
# expansions with operators, read, printf -v, unset, a for loop (which retargets w, and stops
# at a word that names no variable) and a hidden NAME[SUB] local the caller made; a local of
# its own hides it. Every expected value is bash 5.3's.
s161() {
  local e
  s161s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s161f() { local -n w=$1; shift; "$@"; }
  s161a() { w=8; w+=1; (( w += 1 )); echo "$w ${#w} ${w:-d} ${w/8/Y} $(( w + 1 ))"; }
  s161b() { read -r w <<< "rd"; echo "r=$w"; printf -v w '%s' pf; echo "p=$w"; }
  s161c() { unset w; echo "u=${w-gone}"; }
  s161d() { for w in 1 2; do :; done; echo "l=$w"; }
  s161e() { local w=own; echo "own=$w"; }
  s161g() { w=(p q); w+=(z); echo "${w[*]} ${#w[@]}"; }
  e=$( (s161x=1; s161f s161x s161a; echo "x=$s161x"; s161f s161x s161b; echo "x=$s161x"; s161f s161x s161c; echo "x=${s161x-gone}") 2>&1)
  check cnr1 "$(s161s "$e")" '82 2 82 Y2 83;x=82;r=rd;p=pf;x=pf;u=gone;x=gone;'
  e=$( (s161x=xv; s161f s161x s161d; echo "x=$s161x"; s161f s161x s161e; echo "x=$s161x"; unset s161x; s161f s161x s161g; echo "x=${s161x[*]}") 2>&1)
  check cnr2 "$(s161s "$e")" '`1'"'"': not a valid identifier;l=xv;x=xv;own=own;x=xv;p q z 3;x=p q z;'
  s161h() { local -n w=$1; declare w; w=5; s161i; echo "h: $w ${s161y[*]}"; }
  s161i() { echo "i: $w"; w=7; w+=1; (( w += 1 )); echo "i2: $w"; }
  e=$( (s161y=(a b c); s161h 's161y[1]'; echo "${s161y[*]}"; s161f 's161y[2]' s161i; echo "${s161y[*]}") 2>&1)
  check cnr3 "$(s161s "$e")" 'i: 5;i2: 72;h: 72 a b c;a b c;i: c;i2: 72;a b 72;'
}
# ===== SECTION 162: hidden NAME[SUB] array locals through a nameref =====
s162() {
  local e
  s162s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s162f() { local -n w=$1; declare -a w; w[2]=x; declare -p 's162a[1]'; w=5; declare -p 's162a[1]'; echo "[${w[2]}] [$w ${w[*]}] [${w[0]}] [${#w[@]}] [${!w[@]}]"; w+=(y); unset 'w[0]'; declare -p 's162a[1]'; }
  e=$( (s162a=(a b c); s162f 's162a[1]'; declare -p s162a) 2>&1)
  check hna1 "$(s162s "$e")" 'declare -a s162a[1]=([2]="x");declare -a s162a[1]=([0]="5" [2]="x");[x] [5 5 x] [5] [2] [0 2];declare -a s162a[1]=([2]="x" [3]="y");declare -a s162a=([0]="a" [1]="b" [2]="c");'
  s162g() { local -n w=$1; declare -A w=([k]=v [j]=u); echo "${w[k]} ${#w[@]}"; local -a w; echo "st=$?"; declare -ai w=2+3; echo "st=$?"; declare -p 's162a[1]'; }
  s162h() { local -n w=$1; declare -ai w=2+3; declare -a w+=4; declare -ar w; declare -p "s162a[1]"; s162i; echo "st=$?"; declare -p 's162a[1]'; }
  s162i() { w[3]=q; echo "i ${w[*]}"; }
  e=$( (s162a=(a b c); s162g 's162a[1]'; s162h 's162a[1]'; declare -p s162a) 2>&1)
  check hna2 "$(s162s "$e")" 'v 2;local: w: cannot convert associative to indexed array;st=1;declare: w: cannot convert associative to indexed array;st=1;declare -A s162a[1]=([k]="v" [j]="u" );declare -air s162a[1]=([0]="9");w: readonly variable;'
  s162j() { local -n w=$1; declare -ax w=(1 2); local -a w; w+=(m); declare -p 's162a[1]'; for e in "${w[@]}"; do echo "e=$e"; done; unset w; echo "[${w[1]}]"; }
  e=$( (s162a=(a b c); s162j 's162a[1]'; declare -p s162a) 2>&1)
  check hna3 "$(s162s "$e")" 'declare -ax s162a[1]=([0]="1" [1]="2" [2]="m");e=1;e=2;e=m;[];declare -a s162a=([0]="a" [1]="b" [2]="c");'
}
# ===== SECTION 163: parameter operators on an array name; set -u; circular array namerefs =====
s163() {
  local e
  s163s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; case $l in *"not a valid identifier") l="${l%%\`*}NVI" ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  local x=(p q) y=([1]=a) x2=("" b) n=() s163r
  local -A z=([k]=v) z0=([0]=v)
  local -n w=x
  check pao1 "${x:-d} ${x-e} ${x:+f} ${x+g} ${x?i} ${w:-d} ${w-e} ${w:+f} ${w+g}" 'p p f g p p p f g'
  check pao2 "[${y-e}] [${y+g}] [${y:-d}] [${y:+f}] [${z-e}] [${z+g}] [${z0-e}] [${z0+g}] [${n-e}] [${n+g}]" '[e] [] [d] [] [e] [] [v] [g] [e] []'
  check pao3 "[${x2:-d}] [${x2-e}] [${x2:+f}] [${x2+g}] [${y@Q}] [${y@A}] [${#y}]" '[d] [] [] [g] [] [declare -a y] [0]'
  s163r=y
  check pao4 "[${!s163r-e}] [${!s163r+g}] [${y:=zz}] [${y[*]}]" '[e] [] [zz] [zz a]'
  e=$( (y=([1]=a); set -u; echo "a ${y-dd} ${y:-ee} ${y+ff} ${zz-dd} ${zz+ff} ${n-dd} ${3:-p}"; echo "b $x $z0"; (echo "c $y") || echo "st=$?"; (echo "d $n") || echo "st=$?"; (echo "e $z") || echo "st=$?"; (echo "f ${#y}") || echo "st=$?"; (echo "g ${y?gone}") || echo "st=$?") 2>&1)
  check pao5 "$(s163s "$e")" 'a dd ee  dd  dd p;b p v;y: unbound variable;st=1;n: unbound variable;st=1;z: unbound variable;st=1;y: unbound variable;st=1;y: gone;st=1;'
  s163a() { local -n w=w; declare -a w=$1; echo "s=$?"; declare -p w; }
  s163b() { local -n w=w; local -A w=$1; echo "s=$?"; declare -p w; }
  s163c() { local -n w=w; declare -ai w+=$1; echo "s=$?"; declare -p w; }
  s163d() { local -n w=w; declare -Au w+=x; echo "s=$?"; declare -p w; }
  e=$( (s163a x; s163a 3; s163b x; s163b 3) 2>&1)
  check pao6 "$(s163s "$e")" 'local: warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;declare: NVI;s=0;declare -an w=();local: warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;declare: `3'"'"': invalid variable name for name reference;s=1;declare -an w=();local: warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;s=0;declare -An w=([0]="x" );local: warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;local: `3'"'"': invalid variable name for name reference;s=1;declare -An w=();'
  e=$( (s163c x; s163c 3; s163d) 2>/dev/null)
  check pao7 "$(s163s "$e")" 's=0;declare -ain w=();s=1;declare -an w=();s=0;declare -Anu w=([0]="X" );'
}
# ===== SECTION 164: array declarations through computed namerefs and to array namerefs =====
s164() {
  local e
  s164s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; case $l in *"circular name reference") continue ;; *"not a valid identifier") l="${l%%\`*}NVI" ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s164a() { local -n w=$1; declare -a w=x; echo "s=$?"; declare -p w s164g; }
  s164b() { local -n w=$1; declare -A w=x; echo "s=$?"; declare -p s164h; }
  s164c() { local s164v=5; local -n w=$1; declare -ai w=(1+2 3); echo "s=$?"; declare -p s164v; }
  e=$( (s164a s164g; declare -p s164g; s164b s164h; s164c s164v) 2>&1)
  check dna1 "$(s164s "$e")" 's=0;declare -n w="s164g";declare -a s164g=([0]="x");declare: s164g: not found;s=0;declare -A s164h=([0]="x" );s=0;declare -ai s164v=([0]="3" [1]="3");'
  s164d() { local -n w=w; declare -a w=(); declare -a w=3; echo "s=$?"; declare -a w=; echo "s=$?"; local -a w[1]=3+; echo "s=$?"; declare -p w; declare -A w; echo "s=$?"; declare -p w; }
  s164e() { local -n w=w; declare -a w=x w=3; echo "s=$?"; declare -p w; }
  e=$( (s164d; s164e) 2>&1)
  check dna2 "$(s164s "$e")" 'declare: `3'"'"': invalid variable name for name reference;s=1;declare: `'"'"': invalid variable name for name reference;s=1;local: `3+'"'"': invalid variable name for name reference;s=1;declare -an w=();declare: w: cannot convert indexed to associative array;s=1;declare -an w=();declare: NVI;declare: `3'"'"': invalid variable name for name reference;s=1;declare -an w=();'
  s164f() { local -n w=w; declare -A w=x; declare -A w=3; echo "s=$?"; declare w=y; echo "s=$?"; declare -p w; declare -Ai w+=z; echo "s=$?"; declare -A w[k]=y; echo "s=$?"; declare -p w; declare -a w=y; echo "s=$?"; declare -Ar w=q; declare -p w; }
  e=$( (s164f) 2>&1)
  check dna3 "$(s164s "$e")" 'declare: `3'"'"': invalid variable name for name reference;s=1;s=0;declare -An w=([0]="y" );s=0;s=0;declare -Ain w=([0]="0" );declare: w: cannot convert associative to indexed array;s=1;declare -Ainr w=([0]="0" );'
}
# ===== SECTION 165: a function's declaration through its own nameref =====
# A function's local, declare or typeset through its own nameref to a local of the same
# function acts on that local, and a refusal names the nameref. To any other variable it
# makes a new local of the target's name, which may hide a caller's local, a readonly one
# included, but not a readonly global: that is refused naming the target. Every expected
# value is bash 5.3's.
s165() {
  local e
  s165s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s165a() { local -A g=([k]=v); local -n w=g; declare -a w=x; echo "s=$?"; local -a w=(x); echo "s=$?"; declare -p g; }
  s165b() { local -r g1=1; local -n w=g1; declare w=2; echo "s=$? $g1"; local w=3; echo "s=$?"; typeset w=4; echo "s=$?"; }
  e=$( (s165a; s165b) 2>&1)
  check nlt1 "$(s165s "$e")" 'declare: w: cannot convert associative to indexed array;s=1;w: cannot convert associative to indexed array;local: w: cannot convert associative to indexed array;s=1;declare -A g=([k]="v" );declare: w: readonly variable;s=1 1;local: w: readonly variable;s=1;typeset: w: readonly variable;s=1;'
  s165c() { local -r s165l=1; local -A s165m=([k]=v); s165d; echo "c=$s165l ${s165m[k]}"; }
  s165d() { local -n w=s165l v=s165m; declare w=2; echo "s=$? $w"; declare -p s165l; declare -r v; declare -a v=x; echo "s=$?"; declare -p s165m; }
  s165f() { local -n w=s165G; declare w=2; echo "s=$? $s165G"; local w; echo "s=$?"; }
  s165g() { local -r s165x=1; s165h; echo "g=$s165x"; }
  s165h() { local s165x=2; echo "s=$? $s165x"; declare s165y; declare -r s165x; declare -p s165x; }
  e=$( (s165c; readonly s165G=1; s165f; s165g) 2>&1)
  check nlt2 "$(s165s "$e")" 's=0 2;declare -- s165l="2";declare: v: readonly variable;s=1;declare -ar s165m;c=1 v;declare: s165G: readonly variable;s=1 1;local: s165G: readonly variable;s=1;s=0 2;declare -r s165x="2";g=1;'
}
# ===== SECTION 166: a circular nameref that is an associative array =====
# A function's circular nameref given -A and a value names no variable bash can follow:
# its value and elements read as unset, every store to it or through it is dropped with
# status 0 (a default included, which still expands), read and mapfile consume their
# records, test -v finds nothing, ${!REF} is an invalid indirect expansion and `unset`
# unsets the nameref itself. export and readonly of it change nothing, except that
# readonly makes the variable bash finds instead readonly, after which both refuse as a
# readonly variable named "" (twice, status 1, with a value). One that is an indexed
# array is never set for -v and its ${!REF} is empty. Every expected value is bash 5.3's.
s166() {
  local e
  s166s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; case $l in *"circular name reference") continue ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s166a() { local -n w=w; declare -A w=x; w=([k]=3); echo "s=$?"; w[j]=4; w=y; w+=z; w+=([m]=1); w[j]+=5; (( w[k] = 7 )); let 'w[q]=1'; echo "s=$?"; declare -p w; }
  s166b() { local -n w=w; declare -A w=x; echo "<$w> <${w[0]}> <${w[k]:=v}> <${w:=v}> <${w-u}> <${w[k]-u}> <${w[*]-u}> <${#w[@]}> <${!w[@]}> <$(( w + 1 ))>"; [[ -v w ]]; echo "v=$?"; [[ -v w[0] ]]; echo "v=$?"; declare -p w; }
  e=$( (s166a; s166b) 2>&1)
  check anr1 "$(s166s "$e")" 's=0;s=0;declare -An w=([0]="x" );<> <> <v> <v> <u> <u> <u> <0> <> <1>;v=1;v=1;declare -An w=([0]="x" );'
  s166c() { local -n w=w; declare -A w=x; { mapfile -n1 w; echo "s=$?"; read -a w; echo "s=$?"; read w; echo "s=$?"; printf -v 'w[k]' %s p; echo "s=$?"; read x; echo "x=$x"; } <<< $'a\nb\nc\nd'; for w in p q; do echo "<$w>"; done; declare -p w; unset 'w[0]'; declare -p w; unset w; echo "s=$?"; declare -p w; w=5; declare -p w; }
  s166d() { local -n w=w; declare -A w=x; echo "<${!w}>"; echo no; }
  s166e() { local -n w=w; declare -A w=x; echo "<${w[@]:=z}>"; echo no; }
  s166f() { local -n w=w; declare w=(x y); [[ -v w ]]; echo "v=$?"; test -v 'w[0]'; echo "v=$?"; echo "<${!w}>"; }
  e=$( (s166c) 2>&1; (s166d) 2>&1; echo "s=$?"; (s166e) 2>&1; echo "s=$?"; (s166f) 2>&1)
  check anr2 "$(s166s "$e")" 's=0;s=0;s=0;s=0;x=d;<>;<>;declare -An w=([0]="x" );declare -An w=([0]="x" );s=0;declare -- w;declare -- w="5";w: invalid indirect expansion;s=1;w[@]: bad array subscript;s=2;v=1;v=1;<>;'
  s166g() { local -n w=w; declare -A w=x; export w=1; echo "s=$?"; export w; echo "s=$?"; readonly -a w; echo "s=$?"; declare -p w; readonly w=2; echo "s=$?"; readonly w; echo "s=$?"; export w=3; echo "s=$?"; export -n w; echo "s=$?"; s166h; declare -p w; }
  s166h() { export w; echo "h=$?"; }
  s166i() { local -n w=w; declare -A w=x; readonly w; echo "s=$?"; unset w; readonly w; echo "s=$?"; declare -p w; }
  e=$( (s166g; s166i) 2>&1)
  check anr3 "$(s166s "$e")" 's=0;s=0;s=0;declare -An w=([0]="x" );: readonly variable;: readonly variable;s=1;: readonly variable;s=0;: readonly variable;: readonly variable;s=1;s=0;: readonly variable;h=0;declare -An w=([0]="x" );s=0;s=0;declare -r w;'
}
# ===== SECTION 167: unset of a computed name =====
# `unset $v` acts on the name the word expands to as `unset NAME` does: a local of the
# running function stays declared, a caller's local is dropped, a readonly variable is
# refused, and a nameref unsets the variable it names and stays a nameref - a local of
# the function it names stays declared, a caller's is dropped. Every expected value is
# bash 5.3's.
s167() {
  local e
  s167s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s167a() { local x=1 v=x; unset $v; declare -p x; local -n r=s167q; v=r; s167q=5; unset $v; declare -p r; echo "<${s167q-U}>"; }
  s167b() { v=s167z; unset $v; echo "in:${s167z-U}"; }
  s167c() { local s167z=7; s167b; echo "out:${s167z-U}"; }
  s167d() { local -n rr=tt; local tt=4; unset rr; declare -p tt; tt=5; v=rr; unset $v; declare -p tt rr; }
  s167e() { v=rr; unset $v; }
  s167f() { local s167y=3; local -n rr=s167y; s167e; declare -p s167y; }
  e=$( (s167a; s167z=G; s167c; echo "g:$s167z"; readonly s167r=1; v=s167r; unset $v; echo "s=$?"; s167d; s167y=G; s167f) 2>&1)
  check cun1 "$(s167s "$e")" 'declare -- x;declare -n r="s167q";<U>;in:G;out:G;g:G;unset: s167r: cannot unset: readonly variable;s=1;declare -- tt;declare -- tt;declare -n rr="tt";declare -- s167y="G";'
}
# ===== SECTION 168: a default through a nameref to an element =====
# ${REF[SUB]=WORD} through a nameref whose value is NAME[SUB] reads the never-set variable
# that subscript applies to, and then expands WORD and refuses the store as one to the
# name NAME[SUB] (after the running function's name when no command has run in it yet),
# which abandons the line with status 2. Every expected value is bash 5.3's.
s168() {
  local e
  s168s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s168a() { echo "${s168r[0]=v}"; echo no; }
  s168v=(p q)
  declare -n s168r='s168v[1]'
  e=$( (echo "<${s168r[0]-u}> <${s168r:=w}> <${s168r[0]:+x}>"; echo "${s168r[0]:=$(echo hi >&2; echo k)}"; echo no) 2>&1; echo "s=$?"; (s168a; echo no) 2>&1; echo "s=$?"; (echo "${s168r[@]:=v}"; echo no) 2>&1; echo "s=$?")
  check nde1 "$(s168s "$e")" '<u> <q> <>;hi;`s168v[1]'"'"': not a valid identifier;s=2;s168a: `s168v[1]'"'"': not a valid identifier;s=2;s168r[@]: bad array subscript;s=2;'
  unset -n s168r
}
# ===== SECTION 169: unset -n of a name that is no nameref; unset of a caller's prefix binding =====
# `unset -n NAME` unsets only a nameref: a variable, an array, a function or NAME[SUB] stays,
# a readonly one is refused. A function's `unset` of the prefix binding its caller's call
# gave it uncovers the global binding. Every expected value is bash 5.3's.
s169() {
  local e
  s169s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s169f() { local y=1; unset -n y; declare -p y; }
  s169g() { echo g; }
  s169i() { unset s169x; declare -p s169x; }
  s169o() { s169x=1 s169i; echo "o:${s169x-U}"; }
  s169j() { local nm=s169z; unset "$nm"; declare -p s169z; }
  s169p() { s169z=1 s169j; echo "p:${s169z-U}"; }
  e=$( (s169a=1; s169b=(1 2 3); s169t=v; declare -n s169r=s169t; readonly s169q=1
        unset -n s169a 's169b[1]' s169b s169g; echo "a:$? ${s169a-U} ${s169b[*]}"; s169g; s169f
        unset -n s169r; echo "r:${s169r-U} ${s169t-U}"; unset -n s169q; echo "q:$?"
        v=s169a; unset -n "$v"; echo "c:${s169a-U}"
        s169x=G; s169o; s169z=G; s169p) 2>&1)
  check unn1 "$(s169s "$e")" 'a:0 1 1 2 3;g;declare -- y="1";r:U v;unset: s169q: cannot unset: readonly variable;q:1;c:1;declare -- s169x="G";o:G;declare -- s169z="G";p:G;'
}
# ===== SECTION 170: a default for a positional or special parameter =====
# ${N=WORD} and ${N:=WORD} on an unset (or empty) positional or special parameter - $1,
# $10, $@, $* - are refused before WORD expands ("$1: cannot assign in this way"), through
# an indirection after it ("1: invalid variable name"); either abandons the line with status
# 1. A set one ($#, $?, $-, $0) expands as it is. A store through a nameref that is refused
# - readonly, a bad subscript - abandons the line with status 2. Every expected value is
# bash 5.3's.
s170() {
  local e
  s170s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s170f() { echo "${2=y}"; echo no; }
  s170g() { : ${1:=${s170z:=7}}; echo no; }
  e=$( (s170g; echo no) 2>&1; echo "a:$? ${s170z-U}"
       (s170f; echo no) 2>&1; echo "b:$?"
       (s170f p; echo no) 2>&1; echo "c:$?"
       (s170f p q; echo "d:$?") 2>&1)
  check noa1 "$(s170s "$e")" '$1: cannot assign in this way;a:1 U;$2: cannot assign in this way;b:1;$2: cannot assign in this way;c:1;q;no;d:0;'
  s170h() { echo "${@:=x}"; echo no; }
  s170i() { echo "${*=x}"; echo no; }
  s170j() { echo "${10:=x}"; echo no; }
  s170k() { echo "<${#:=x}> <${?:=x}> <${-:+set}> <${0:+set}>"; }
  e=$( (s170h; echo no) 2>&1; echo "h:$?"; (s170i; echo no) 2>&1; echo "i:$?"; (s170j 1; echo no) 2>&1; echo "j:$?"; s170h p; s170k)
  check noa2 "$(s170s "$e")" '$@: cannot assign in this way;h:1;$*: cannot assign in this way;i:1;$10: cannot assign in this way;j:1;p;no;<0> <0> <set> <set>;'
  s170m() { local r=$1; : ${!r:=${s170y:=7}}; echo no; }
  s170n() { local r='@'; echo "${!r=x}"; echo no; }
  e=$( (s170m 2; echo no) 2>&1; echo "m:$?"; (s170n; echo no) 2>&1; echo "n:$?")
  check noa3 "$(s170s "$e")" '2: invalid variable name;m:1;@: invalid variable name;n:1;'
  e=$( (readonly s170t; declare -n s170r=s170t; : ${s170r:=5}; echo no) 2>&1; echo "s=$?"
       (declare -n s170q='s170a[@]'; : ${s170q:=5}; echo no) 2>&1; echo "s=$?")
  check noa4 "$(s170s "$e")" 's170t: readonly variable;s=2;s170a[@]: bad array subscript;s=2;'
}
# ===== SECTION 171: readonly through a nameref that is an array =====
# A function's circular nameref given an associative array names no variable bash can
# follow: readonly of it makes the variable bash finds instead - named "" - readonly, and
# every later scalar store through it is refused as that variable's. An assignment or a
# ${REF:=WORD} abandons the line (status 1, 2), a builtin's store or an arithmetic one
# fails it (status 1; read's non-last name 2); an element, an array literal, mapfile,
# read -a and a prefix binding still store nothing, status 0. export and readonly of one
# that is an indexed array change nothing, status 0. Every expected value is bash 5.3's.
s171() {
  local e
  s171s() { local l; while IFS= read -r l; do case $l in *"warning: w: circular name reference") ;; *": line "*) l=${l#*: line }; l=${l#*: }; printf '%s;' "$l" ;; *) printf '%s;' "$l" ;; esac; done <<< "$1"; }
  s171a() { local -n w=w; declare -A w=x; readonly w; echo "r:$?"; w=3; echo no; }
  s171b() { local -n w=w; declare -A w=x; readonly w; w+=3; echo no; }
  s171c() { local -n w=w; declare -A w=x; readonly w; : ${w:=6}; echo no; }
  e=$( (s171a; echo no) 2>&1; echo "a:$?"; (s171b; echo no) 2>&1; echo "b:$?"; (s171c; echo no) 2>&1; echo "c:$?")
  check vro1 "$(s171s "$e")" 'r:0;: readonly variable;a:1;: readonly variable;b:1;: readonly variable;c:2;'
  s171d() {
    local -n w=w; declare -A w=x; readonly w
    read w <<< "r"; echo "d:$?"; read w s171u <<< "r s"; echo "d2:$? ${s171u-U}"
    printf -v w x; echo "e:$?"; (( w = 4 )); echo "f:$?"; let w=5 s171v=1; echo "g:$? ${s171v-U}"
    for w in 1 2; do echo in; done; echo "h:$?"
    w[k]=1; echo "i:$?"; w=(a); echo "j:$?"; mapfile w <<< "m"; echo "k:$?"; read -a w <<< "q"; echo "l:$?"
    w=3 eval 'echo "m:$?"'; readonly w; echo "n:$?"
    : $(( w = 2 )); echo no
  }
  e=$( (s171d; echo no) 2>&1; echo "o:$?")
  check vro2 "$(s171s "$e")" ': readonly variable;d:1;: readonly variable;d2:2 U;: readonly variable;e:1;: readonly variable;f:1;: readonly variable;g:1 U;: readonly variable;h:1;i:0;j:0;k:0;l:0;m:0;: readonly variable;n:0;: readonly variable;o:1;'
  s171x() { local -n w=w; declare -a w=x 2>/dev/null; export w=1; echo "x:$?"; declare -p w; export w; echo "x2:$?"; declare -p w; }
  s171y() { local -n w=w; declare -a w=x 2>/dev/null; readonly w 2>/dev/null; echo "y:$?"; declare -p w; }
  e=$( (s171x; s171y) 2>&1)
  check vro3 "$(s171s "$e")" 'x:0;declare -an w=();x2:0;declare -an w=();y:0;declare -an w=();'
}
# ===== SECTION 172: $!, and a line an expansion error abandons =====
# $! is unset until the first `cmd &`: ${!-u} is the word, ${!:=x} is refused ("$!: cannot
# assign in this way", status 1), and under set -u $! is "$!: unbound variable" and ${!} is
# "!: unbound variable", while ${#!} is 0. Each `cmd &` then sets it to a new value, which
# a job in a subshell does not change for the shell and the job itself does not see. After
# an expansion error the rest of the command is not expanded: a later ${NAME=WORD} stores
# nothing and is not refused again, a later ${NAME?WORD} raises nothing. A function's
# readonly of its circular nameref that is an indexed array changes nothing after it
# returns. Every expected value is bash 5.3's.
s172() {
  local e
  s172s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$(echo "[$!] ${!-u} <${!+set}> ${!:-e} ${#!}"; (: ${!:=x}; echo no) 2>&1; echo "a:$?"
      (set -u; echo "${#!} ${!-d}"; echo "$!"; echo no) 2>&1; echo "b:$?"; (set -u; echo "${!}"; echo no) 2>&1; echo "c:$?")
  check bgp1 "$(s172s "$e")" '[] u <> e 0;$!: cannot assign in this way;a:1;0 d;$!: unbound variable;b:1;!: unbound variable;c:1;'
  e=$( (: &); echo "d:${!+set}"; s172f() { : & }; s172f; s172p=$!; echo "e:${!+set}"
       s172j=$({ [ "$s172p" = "$!" ] && echo same; } &); echo "f:$s172j"
       : & [ "$s172p" != "$!" ] && echo g:new; ( : & ); [ "$s172p" != "$!" ] && echo h:kept)
  check bgp2 "$e" 'd:
e:set
f:same
g:new
h:kept'
  e=$( (readonly s172r; : ${s172r:=1} ${s172r:=2} ${s172z:=3} ${s172u:?no} $(echo sub >&2); echo no) 2>&1; echo "i:$? ${s172z-U}"
       (unset s172t; readonly s172t; declare -n s172q=s172t; : ${s172q:=5} "${s172q:=6}"; echo no) 2>&1; echo "j:$?"
       (: $((1/0)) ${s172y:=6}; echo no) 2>&1; echo "k:$? ${s172y-U}")
  check ast1 "$(s172s "$e")" 's172r: readonly variable;i:2 U;s172t: readonly variable;j:2;1/0: division by 0 (error token is "0");k:1 U;'
  s172a() { local -n w=w; declare -a w=(p q); readonly w; echo "r:$?"; }
  s172b() { local -n w=w; echo "l:$?"; }
  e=$(w=; unset w; s172a 2>/dev/null; declare -p w 2>&1; s172b 2>/dev/null; w=5; echo "w:$w")
  check vrol "$(s172s "$e")" 'r:0;declare: w: not found;l:0;w:5;'
}
# ===== SECTION 173: the variable named "" =====
# A function's nameref that is an associative array names the variable "" of the call that
# binds it, which all such namerefs of that call share: a scalar store through one (=, +=,
# read, printf -v, ((..)), let, ${w:=..}, for) stores there, a callee's store too, an array
# store does not. readonly or export of one gives "" the attribute and unsets it; a store
# through another nameref to a readonly "" is then refused. `declare -p ""` prints it, a
# return drops it ("not found", status 1), and `local -p ""` never finds it. Every expected
# value is bash 5.3's.
s173() {
  local e
  s173a() { local -n w=w; declare -A w; w=3; declare -p ""; local -n v=v; declare -A v; v+=4; declare -p ""; echo "w=[$w]"; }
  e=$(s173a 2>/dev/null; declare -p "" 2>/dev/null; echo "t:$?")
  check vv1 "$e" 'declare -- ="3"
declare -- ="34"
w=[]
t:1'
  s173b() { local -n w=w; declare -A w; read w <<< "rd"; declare -p ""; printf -v w '%s' 'a"b$c'; declare -p ""; (( w = 9 )); declare -p ""
            let w=10; declare -p ""; : ${w:=d}; declare -p ""; for w in p q; do :; done; declare -p ""; w=(a b); w[1]=x; declare -p ""; }
  e=$(s173b 2>/dev/null)
  check vv2 "$e" 'declare -- ="rd"
declare -- ="a\"b\$c"
declare -- ="9"
declare -- ="10"
declare -- ="d"
declare -- ="q"
declare -- ="q"'
  s173c() { local -n w=w; declare -A w; w=1; s173d; declare -p ""; ( w=7; declare -p "" ); declare -p ""; readonly w; declare -p ""
            local -n v=v; declare -A v; v=2; echo "v:$?"; }
  s173d() { w=8; }
  e=$(s173c 2>/dev/null; echo "r:$?")
  check vv3 "$e" 'declare -- ="8"
declare -- ="7"
declare -- ="8"
declare -r '
  s173e() { local -n w=w; declare -A w; w=1; export w; declare -p ""; w=4; declare -p ""; local -p "" 2>/dev/null; echo "l:$?"; }
  e=$(s173e 2>/dev/null)
  check vv4 "${e//$'\n'/;}" 'declare -x ;declare -x ="4";l:1'
}
# ===== END SECTIONS =====

s154  # SECTION-CALL 154
s155  # SECTION-CALL 155
s156  # SECTION-CALL 156
s157  # SECTION-CALL 157
s158  # SECTION-CALL 158
s159  # SECTION-CALL 159
s160  # SECTION-CALL 160
s161  # SECTION-CALL 161
s162  # SECTION-CALL 162
s163  # SECTION-CALL 163
s164  # SECTION-CALL 164
s165  # SECTION-CALL 165
s166  # SECTION-CALL 166
s167  # SECTION-CALL 167
s168  # SECTION-CALL 168
s169  # SECTION-CALL 169
s170  # SECTION-CALL 170
s171  # SECTION-CALL 171
s172  # SECTION-CALL 172
s173  # SECTION-CALL 173
echo "full: $checks checks, $fails failures"
exit $fails
