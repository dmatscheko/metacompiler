#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 7
# of 7, sections 135 on; a new section goes here. The split is explained in
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

# ===== SECTION 135: list values for array members, stores to circular array namerefs =====
# NAME[SUB]=(list) is refused and abandons the line, after a declaration has made NAME an
# array. export and readonly of a function's circular local nameref act on the global. A
# store to a circular nameref that holds an array leaves the array as it was; its
# declaration's own list still lands. Every expected value is bash 5.3's, less the byte
# bash prints between the quotes.
s135() {
  local e
  s135s() { local l; while IFS= read -r l; do l=${l#*: line }; l=${l//$'\001'/}; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$( { s135a[1]=(x y); echo no; } 2>&1; echo "s=$?"; declare -p s135a 2>&1)
  check alm1 "$(s135s "$e")" 's135a[1]: cannot assign list to array member;'
  e=$(s135f() { local s135b[2]=(p); echo no; }; s135f 2>&1; echo "s=$?")
  check alm2 "$(s135s "$e")" 's135b[2]: cannot assign list to array member;'
  e=$(declare -A s135c; { declare -A s135c[k]=(v w); echo no; } 2>&1; declare -p s135c)
  check alm3 "$(s135s "$e")" 's135c[k]: cannot assign list to array member;'
  e=$(s135f() { export s135d[1]=(p q); echo no; }; s135f 2>&1; declare -p s135d 2>&1)
  check alm4 "$(s135s "$e")" 's135d[1]: cannot assign list to array member;'
  e=$(s135g() { local -n s135w=s135w; export s135w=q; declare -p s135w; local -n s135v=s135v; readonly s135v; export -n s135v; }; s135g 2>&1; declare -p s135w)
  check alm5 "$(s135s "$e")" 'local: warning: s135w: circular name reference;warning: s135w: circular name reference;warning: s135w: maximum nameref depth (8) exceeded;warning: s135w: circular name reference;declare -n s135w="s135w";local: warning: s135v: circular name reference;warning: s135v: circular name reference;warning: s135v: circular name reference;warning: s135v: maximum nameref depth (8) exceeded;warning: s135v: circular name reference;declare -x s135w="q";'
  e=$(s135g() { local -n s135u=s135u; readonly s135u=(x); }; s135g 2>&1; declare -p s135u)
  check alm6 "$(s135s "$e")" 'local: warning: s135u: circular name reference;warning: s135u: circular name reference;warning: s135u: circular name reference;warning: s135u: circular name reference;warning: s135u: circular name reference;declare -ar s135u=([0]="x");'
  e=$(readonly s135z=1; export s135z=2 2>/dev/null; declare -p s135z)
  check alm7 "$e" 'declare -rx s135z="1"'
  e=$(s135g() { local -n s135t=s135t; declare s135t=(x y); s135t[1]=q; echo no; }; s135g 2>&1; echo "s=$?")
  check alm8 "$(s135s "$e")" 'local: warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;warning: s135t: circular name reference;`'"'"': not a valid identifier;'
  e=$(s135g() { local -n s135q=s135q; declare s135q=(x y); (( s135q[1]=7 )); echo "s=$?"; let 's135q[0]=4'; echo "s=$?"; declare s135q[1]=z; echo "s=$?"; unset 's135q[1]' s135q; echo "s=$?"; read -a s135q <<< 'a b'; echo "s=$?"; declare s135q+=(z); declare -p s135q; }; s135g 2>/dev/null)
  check alm9 "$e" $'s=0\ns=0\ns=1\ns=0\ns=1\ndeclare -an s135q=([0]="x" [1]="y" [2]="z")'
}
# ===== SECTION 136: declaration operands with brackets in the subscript =====
# The subscript of a declaration operand that is one word ends at the "]" that closes its
# "[", brackets inside it counted. Every expected value is bash 5.3's.
s136() {
  local e v
  s136s() { local l; while IFS= read -r l; do l=${l#*: line }; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$( { declare -n 's136a[c6[x y]]'; echo "s=$?"; declare 's136a[a[1]]=x'; echo "s=$?"; declare -p s136a; } 2>&1)
  check dsb1 "$(s136s "$e")" 'declare: s136a[c6[x y]]: reference variable cannot be an array;s=1;s=0;declare -a s136a=([0]="x");'
  e=$(s136f() { local -n 's136b[c6[x y]]'; echo "s=$?"; local 's136c[b[2]]=y'; declare -p s136c; }; s136f 2>&1)
  check dsb2 "$(s136s "$e")" 'local: s136b[c6[x y]]: reference variable cannot be an array;s=1;declare -a s136c=([0]="y");'
  e=$( { declare -a 's136d[a[1]]=(p q)'; declare 's136e[a[1]'; echo "s=$?"; declare 's136e[a]b]=x'; echo "s=$?"; declare -p s136d; } 2>&1)
  check dsb3 "$(s136s "$e")" 'declare: `s136e[a[1]'"'"': not a valid identifier;s=1;declare: `s136e[a]b]=x'"'"': not a valid identifier;s=1;declare -a s136d=([0]="p" [1]="q");'
  v='s136g[x[1]]'
  e=$( { declare "$v"; declare -p s136g; v='s136h[a[1]]+=x'; declare "$v"; declare -p s136h; export "$v"; echo "s=$?"; } 2>&1)
  check dsb4 "$(s136s "$e")" 'declare -a s136g;declare -a s136h=([0]="x");export: `s136h[a[1]]'"'"': not a valid identifier;s=1;'
}
# ===== SECTION 137: stores to a readonly nameref with no value =====
# A nameref with no value takes a stored value as its own, which must name a variable:
# "not a valid identifier" comes before the nameref's readonly, and arithmetic reports it
# and goes on. An element of one names no variable. Every expected value is bash 5.3's.
s137() {
  local e
  s137s() { local l; while IFS= read -r l; do l=${l#*: line }; printf '%s;' "${l#*: }"; done <<< "$1"; }
  e=$( { declare -nr s137a; s137a=5; echo no; } 2>&1; echo "s=$?")
  e="$e"$'\n'"$(declare -nr s137a; { s137a=abc; echo no; } 2>&1; echo "s=$?")"
  e="$e"$'\n'"$(declare -nr s137a; { s137a+=5; echo no; } 2>&1; echo "s=$?"; declare -p s137a)"
  check rnv1 "$(s137s "$e")" '`5'"'"': not a valid identifier;s137a: readonly variable;`5'"'"': not a valid identifier;'
  e=$( { declare -nr s137b; (( s137b = 3 )); echo "s=$?"; (( s137b = 0 )); echo "s=$?"; let s137b=4; echo "s=$?"; echo "$(( s137b = 2 ))"; declare -p s137b; } 2>&1)
  check rnv2 "$(s137s "$e")" '((: `3'"'"': not a valid identifier;s=0;((: `0'"'"': not a valid identifier;s=1;let: `4'"'"': not a valid identifier;s=0;`2'"'"': not a valid identifier;2;declare -nr s137b;'
  e=$( { declare -nr s137c; read s137c <<< 7; echo "s=$?"; printf -v s137c %s 8; echo "s=$?"; read s137c <<< abc; echo "s=$?"; } 2>&1)
  check rnv3 "$(s137s "$e")" 'read: `7'"'"': not a valid identifier;s=1;printf: `8'"'"': not a valid identifier;s=1;s137c: readonly variable;s=1;'
  e=$(declare -nr s137d; { s137d[1]=3; echo no; } 2>&1; declare -p s137d)
  e="$e"$'\n'"$(declare -n s137e; { (( s137e[1] = 4 )); echo "s=$?"; } 2>&1; { s137e[1]=3; echo no; } 2>&1; declare -p s137e)"
  check rnv4 "$(s137s "$e")" '`'"'"': not a valid identifier;((: `'"'"': not a valid identifier;s=0;`'"'"': not a valid identifier;'
  e=$(s137f() { local -nr s137g; s137g=5; echo no; }; s137f 2>&1; echo "s=$?")
  check rnv5 "$(s137s "$e")" '`5'"'"': not a valid identifier;'
}
# ===== SECTION 138: list values in declarations, assignment prefixes and failed stores =====
# NAME[SUB]=(list) as a declaration operand first declares NAME, and a declaration that
# fails then reports bare at top level, where it abandons the line, and twice in a function,
# which goes on. A quoted (list) is compound only for an array declaration and draws a
# deprecation warning on a subscript. A prefix may assign a list. A refused ${NAME:=WORD}
# store gives status 2 in ( ) and 1 in $( ). Removing a pattern * L or L * costs time
# linear in the value, which a value of thousands of characters needs. Every expected
# value is bash 5.3's.
s138() {
  local e
  s138s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$(s138a=(a b); s138f() { declare -gA s138a[k]=(x); echo "s=$?"; }; s138f 2>&1; declare -p s138a)
  check dlm1 "$(s138s "$e")" 's138f: s138a: cannot convert indexed to associative array;declare: s138a: cannot convert indexed to associative array;s=1;declare -a s138a=([0]="a" [1]="b");'
  e=$(s138f() { local s138b=(a); local -A s138b[k]=(x); echo "s=$?"; declare -p s138b; }; s138f 2>&1)
  check dlm2 "$(s138s "$e")" 's138b: cannot convert indexed to associative array;local: s138b: cannot convert indexed to associative array;s=1;declare -a s138b=([0]="a");'
  e="$(readonly s138c=1; s138g() { declare -gA s138c[1]=(x); echo "s=$?"; }; s138g 2>&1)"
  check dlm3 "$(s138s "$e")" 's138g: s138c: readonly variable;declare: s138c: readonly variable;s=1;'
  e=$( { s138x=1 s138e[2]=(y) echo hi; } 2>&1; echo "s=$?"; s138h() { echo "h[$s138e]"; declare -p s138e; }; s138e=(1 "a  b" [3]=q) s138h)
  check pfl1 "$(s138s "$e")" '`s138e[2]'"'"': not a valid identifier;hi;s=0;h[(1 a  b [3]=q)];declare -x s138e="(1 a  b [3]=q)";'
  local v='(x y)'
  e=$(declare -g 's138i[1]=(x)' 2>&1; declare -a s138j=$v; declare -A s138k=$v; declare s138l=$v; declare -ai s138m=$v; declare -p s138i s138j s138k s138l s138m)
  check qcv1 "$(s138s "$e")" 'warning: s138i[1]=(x): quoted compound array assignment deprecated;declare -a s138i=([1]="(x)");declare -a s138j=([0]="x" [1]="y");declare -A s138k=([x]="y" );declare -- s138l="(x y)";declare -ai s138m=([0]="0" [1]="0");'
  e=$(s138n=(1 2); declare -g s138n='(x y)'; s138o=(1); export s138o='(x y)'; s138f() { local -a l='(p q)'; local m='(r)'; declare -p l m; }; s138f; declare -p s138n s138o)
  check qcv2 "$(s138s "$e")" 'declare -a l=([0]="p" [1]="q");declare -- m="(r)";declare -a s138n=([0]="x" [1]="y");declare -ax s138o=([0]="(x y)");'
  e=$(readonly s138p; ( : "${s138p:=9}"; echo no ) 2>&1; echo "s=$?"; { r=$( : "${s138p:=9}"; echo no ); } 2>&1; echo "s=$?")
  e="$e"$'\n'"$(s138q=(1); readonly s138q; ( : "${s138q[3]:=5}"; echo no ) 2>&1; echo "s=$?")"
  check rst1 "$(s138s "$e")" 's138p: readonly variable;s=2;s138p: readonly variable;s=1;s138q: readonly variable;s=2;'
  local s=': line 1: ' i
  for i in {1..9}; do s=$s$s; done
  local t=${s#*: line } u=${s%: line*} v=${s#*Z} w=${s%%Z*}
  check prm1 "${#s}/${#t}/${s##*: line }/${#u}/${s%%: *}/${s##*1: }/${#v}/${#w}" "5120/5113/1: /5110///5120/5120"
}
# ===== SECTION 139: reads through a nameref that is an array =====
# A function's circular nameref given an array literal names no variable: a read of its
# value is refused, which abandons the line with status 1, while its elements, their
# count, its length and arithmetic read it as unset. bash names bytes of its array in
# the message, which these checks leave out. Every expected value is bash 5.3's.
s139() {
  local e
  s139s() { local l; while IFS= read -r l; do case $l in *circular*) ;; *"invalid variable name for name reference") printf 'inv;' ;; *) printf '%s;' "$l" ;; esac; done <<< "$1"; }
  s139a() { local -n w=w; declare w=(x y) 2>/dev/null; echo "[$w]"; }
  s139b() { local -n w=w; declare w=(x y) 2>/dev/null; echo "[${w:-q}]"; }
  s139c() { local -n w=w; declare w=(x y) 2>/dev/null; local x=$w; echo "[$x]"; }
  s139d() { local -n w=w; declare w=(x y) 2>/dev/null; echo "[${w[1]}|${w[@]}|${#w}|${#w[@]}|$(( w + 1 ))]"; }
  s139e() { local -n w=w; declare w=(x y) 2>/dev/null; echo "[${w:?no}]"; }
  s139g() { local -n w=w; declare w=(x y) 2>/dev/null; echo "[${w+set}]"; }
  e=$( (s139a; echo "s=$?") 2>&1; echo "s=$?"; (s139b; echo "s=$?") 2>&1; echo "s=$?"; (s139c; echo "s=$?") 2>&1; echo "s=$?")
  check nra1 "$(s139s "$e")" 'inv;s=1;inv;s=1;inv;s=1;'
  e=$( (s139d; echo "s=$?") 2>&1; echo "s=$?"; (s139e; echo "s=$?") 2>&1; echo "s=$?"; (s139g; echo "s=$?") 2>&1; echo "s=$?")
  check nra2 "$(s139s "$e")" '[||0|0|1];s=0;s=0;inv;s=1;inv;s=1;'
}
# ===== SECTION 140: array literals for a circular nameref, again =====
# A function's circular nameref that an array literal made an array stays a nameref, and
# the next literal declaration replaces its elements. Every expected value is bash 5.3's.
s140() {
  local e
  s140f() { local -n w=w; local w=(q); local w=(z); declare -p w; }
  s140g() { local -n w=w; declare -a w=(q r); local w+=(s); declare -p w; }
  e=$(s140f 2>/dev/null; s140g 2>/dev/null)
  check cna1 "$e" 'declare -an w=([0]="z")'$'\n''declare -an w=([0]="q" [1]="r" [2]="s")'
}
# ===== SECTION 141: NAME[SUB] declarations and array reads on a global nameref =====
# Outside a function's locals, NAME[SUB]=VALUE of a nameref makes NAME itself the array,
# warning that the nameref attribute goes; -g without a value acts on the target. read -a
# and mapfile store through a nameref to its target or, when it has no value, to it, with
# the same warning. Every expected value is bash 5.3's.
s141() {
  local e
  s141s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$(declare -gn s141a=s141t; s141t=4; declare -g s141a[1]=5 2>&1; declare -p s141a s141t)
  check nsd1 "$(s141s "$e")" 'warning: s141a: removing nameref attribute;declare -a s141a=([1]="5");declare -- s141t="4";'
  e=$(declare -gn s141b; declare -ga s141b[1]=3 2>&1; declare -gn s141c=s141u; declare -g s141c[1]; declare -p s141b s141c s141u)
  check nsd2 "$(s141s "$e")" 'warning: s141b: removing nameref attribute;declare -a s141b=([1]="3");declare -n s141c="s141u";declare -a s141u;'
  e=$(declare -gn s141d=s141v; s141v=1; declare -g s141d[1]; declare -p s141d s141v)
  check nsd3 "$(s141s "$e")" 'declare -n s141d="s141v";declare -a s141v=([0]="1");'
  e=$(declare -gn s141e; read -a s141e <<< 'x y' 2>&1; declare -gn s141f=s141w; read -a s141f <<< 'p q'; declare -p s141e s141f s141w)
  check nsd4 "$(s141s "$e")" 'warning: s141e: removing nameref attribute;declare -a s141e=([0]="x" [1]="y");declare -n s141f="s141w";declare -a s141w=([0]="p" [1]="q");'
  e=$(declare -gn s141g; mapfile s141g <<< 'x' 2>&1; declare -p s141g)
  check nsd5 "$(s141s "$e")" 'warning: s141g: removing nameref attribute;declare -a s141g=([0]=$'"'"'x\n'"'"');'
}
# ===== SECTION 142: the command name before a declaration's expansion errors =====
# An array literal that a declaration refuses while its words expand (an indexed array made
# associative, a readonly name given the -a or -A option) is reported first with bash's
# current command name: the function just called, "[[" or "((" after one of those, nothing
# after a simple command. Then declare reports it again. Without such an option a readonly
# literal abandons the line with no declare report. Every expected value is bash 5.3's.
s142() {
  local e
  s142s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$(s142ib=(1); s142f() { declare -gA s142ib=(x); }; s142f 2>&1; echo "s $?")
  check tcn1 "$(s142s "$e")" 's142f: s142ib: cannot convert indexed to associative array;declare: s142ib: cannot convert indexed to associative array;s 1;'
  e=$(s142ic=(1); s142g() { true; declare -gA s142ic=(x); }; s142g 2>&1; echo "s $?")
  check tcn2 "$(s142s "$e")" 's142ic: cannot convert indexed to associative array;declare: s142ic: cannot convert indexed to associative array;s 1;'
  e=$(s142id=(1); s142h() { echo "$(declare -gA s142id=(x))"; }; s142h 2>&1)
  check tcn3 "$(s142s "$e")" 's142h: s142id: cannot convert indexed to associative array;declare: s142id: cannot convert indexed to associative array;'
  e=$(s142ie=(1); s142j() { [[ -n x ]]; declare -gA s142ie=(x); }; s142j 2>&1)
  check tcn4 "$(s142s "$e")" '[[: s142ie: cannot convert indexed to associative array;declare: s142ie: cannot convert indexed to associative array;'
  e=$(s142k() { declare -a s142c=(1); echo "k $?"; }; readonly s142c=1; s142k 2>&1; echo "s $?")
  check tcn5 "$(s142s "$e")" 's142k: s142c: readonly variable;declare: s142c: readonly variable;k 1;s 0;'
  e=$(s142m() { declare -g s142d=(1); echo "m $?"; }; readonly s142d=1; s142m 2>&1; echo "s $?")
  check tcn6 "$(s142s "$e")" 's142d: readonly variable;'
}
# ===== SECTION 143: declarations of a function's nameref that loops only at run time =====
# A local nameref whose value makes it refer to itself is circular only once it runs. An
# array literal declaration then makes it an array that keeps the nameref attribute; one that
# is also readonly abandons the line after the warnings. NAME[SUB]=VALUE of a nameref with
# no value makes it an empty array and refuses VALUE as a name. Every expected value is
# bash 5.3's.
s143() {
  local e
  s143s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s143f() { local -n w=$1; local w=(q); declare -p w; }
  e=$(s143f w 2>/dev/null)
  check rcn1 "$e" 'declare -an w=([0]="q")'
  s143g() { local -nr w=w; local w=(q); echo "g $?"; }
  e=$(s143g 2>&1; echo "s $?")
  check rcn2 "$(s143s "$e")" 'local: warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;'
  s143h() { local -n q7; declare q7[1]=5; echo "h $?"; declare -p q7; }
  e=$(s143h 2>&1)
  check rcn3 "$(s143s "$e")" 'declare: `5'"'"': invalid variable name for name reference;h 1;declare -an q7;'
}
# ===== SECTION 144: an extglob prefix removal of every line of a subshell's errors =====
# ${l##*: line +([0-9]): } drops the file and line prefix of each error line that a
# subshell's command substitution caught, many times over. Every expected value is bash 5.3's.
s144() {
  local e o
  shopt -s extglob
  s144s() { local l; while IFS= read -r l; do printf '%s;' "${l##*: line +([0-9]): }"; done <<< "$1"; }
  for o in 1 2 3 4 5 6 7 8 9 10 11 12; do
    e=$( ( s144a=(1); declare -gA s144a=(x); readonly s144r=1; s144r=2; echo "c $o" ) 2>&1 )
    e=$(s144s "$e")
  done
  check egs1 "$e" 's144a: cannot convert indexed to associative array;declare: s144a: cannot convert indexed to associative array;s144r: readonly variable;'
  check egs2 "$(s144s 'x: line 3: y: line 45: z')" 'z;'
  local l='/a/b.sh: line 12: x: y: line 3: z'
  check egs3 "${l%: line +([0-9]): *}|${l%%: line +([0-9]): *}|${l%%line @(1|3)*}|${l##*@(x|y)\: }" '/a/b.sh: line 12: x: y|/a/b.sh|/a/b.sh: |line 3: z'
  check egs4 "${l#?(/)a}|${l##*[0-9]: }" '/b.sh: line 12: x: y: line 3: z|z'
  shopt -u extglob
}
# ===== SECTION 145: the expansion-time reports of a declaration come first =====
# A declaration's words expand before the builtin runs, and before its redirections apply:
# every literal's "NAME: ..." comes before the builtin's "declare: NAME: ...", and a
# redirection hides only the builtin's. A NAME[SUB]=(list) operand takes its turn among the
# others. Every expected value is bash 5.3's.
s145() {
  local e
  s145s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$(s145a=(1); s145f() { declare -gA s145a=(x) s145a=(y); echo "f $?"; }; s145f 2>&1)
  check dxo1 "$(s145s "$e")" 's145f: s145a: cannot convert indexed to associative array;s145f: s145a: cannot convert indexed to associative array;declare: s145a: cannot convert indexed to associative array;declare: s145a: cannot convert indexed to associative array;f 1;'
  e=$(s145b=(1); s145g() { declare -gA s145b=(x) 2>/dev/null; echo "g $?"; }; s145g 2>&1)
  check dxo2 "$(s145s "$e")" 's145g: s145b: cannot convert indexed to associative array;g 1;'
  e=$(readonly s145r=1; s145h() { declare -a s145r=(1) s145v=$(echo v) s145r=(2); echo "h $?"; }; s145h 2>&1)
  check dxo3 "$(s145s "$e")" 's145h: s145r: readonly variable;s145h: s145r: readonly variable;declare: s145r: readonly variable;declare: s145r: readonly variable;h 1;'
  e=$(s145c=(1); s145q=(1); s145k() { declare -gA s145c=(x) s145q[1]=(a b) s145z=(y); echo "k $?"; declare -p s145z; }; s145k 2>&1)
  check dxo4 "$(s145s "$e")" 's145k: s145c: cannot convert indexed to associative array;s145k: s145q: cannot convert indexed to associative array;declare: s145c: cannot convert indexed to associative array;declare: s145q: cannot convert indexed to associative array;k 1;declare -A s145z=([y]="" );'
  e=$( (s145d=(1); s145m() { declare -gA s145d=(x) s145n[1]=(a b) s145d=(y); echo "m $?"; }; s145m) 2>&1; echo "s $?")
  check dxo5 "$(s145s "$e")" 's145m: s145d: cannot convert indexed to associative array;s145n[1]: cannot assign list to array member;s 1;'
}
# ===== SECTION 146: -a or -A of a function's readonly local =====
# The declaration is refused, status 1, and the local scalar becomes an empty array of that
# kind all the same, keeping its attributes; one that is an array already stays as it is.
# Every expected value is bash 5.3's.
s146() {
  local e
  s146s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s146f() { local -r y=1; local -a y; echo "a $?"; declare -p y; }
  e=$(s146f 2>/dev/null)
  check rla1 "${e//$'\n'/;}" 'a 1;declare -ar y=()'
  s146g() { local -ri y=1; declare -A y=x; echo "b $?"; declare -p y; }
  e=$(s146g 2>&1)
  check rla2 "$(s146s "$e")" 'declare: y: readonly variable;b 1;declare -Air y=();'
  s146h() { local -r y; local -a y; local -nr w=w; local -a w; declare -p y w; } 2>/dev/null
  check rla3 "$(s146h)" 'declare -ar y
declare -anr w=()'
  s146k() { local -ra y=(1); local -A y; declare -p y; } 2>/dev/null
  check rla4 "$(s146k)" 'declare -ar y=([0]="1")'
}
# ===== SECTION 147: declarations of a function's circular nameref =====
# readonly of a function's nameref that refers to itself acts on the global variable, and a
# literal declared through the nameref then meets that readonly global: after the lookups'
# warnings "NAME: readonly variable", and the line is abandoned. A value that is NAME itself
# warns once more, stored or not. A value the nameref refuses applies no attribute. Every
# expected value is bash 5.3's.
s147() {
  local e
  s147s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$( (s147f() { local -n w=w; readonly w; echo "a $?"; local w=(q); echo "b $?"; }; s147f) 2>&1; echo "s $?")
  check crg1 "$(s147s "$e")" 'local: warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;warning: w: maximum nameref depth (8) exceeded;a 0;warning: w: circular name reference;warning: w: circular name reference;warning: w: circular name reference;w: readonly variable;s 1;'
  e=$(s147g() { local -n v=v; local v=v; echo "c $?"; declare -p v; }; s147g 2>&1)
  check crg2 "$(s147s "$e")" 'local: warning: v: circular name reference;warning: v: circular name reference;warning: v: circular name reference;warning: v: circular name reference;warning: v: circular name reference;c 0;declare -n v="v";'
  e=$(s147h() { local -n v=v; local -i v=v; echo "d $?"; }; s147h 2>&1)
  check crg3 "$(s147s "$e")" 'local: warning: v: circular name reference;warning: v: circular name reference;warning: v: circular name reference;warning: v: circular name reference;warning: v: circular name reference;d 1;'
  s147k() { local -n u=$1; local -i u=3; echo "e $?"; declare -p u; }
  e=$(s147k u 2>/dev/null)
  check crg4 "${e//$'\n'/;}" 'e 1;declare -n u="u"'
  s147m() { local -n u=$1; local -i u=u; echo "f $?"; declare -p u; }
  e=$(s147m u 2>/dev/null)
  check crg5 "${e//$'\n'/;}" 'f 1;declare -in u="u"'
}
# ===== SECTION 148: declare -g NAME[SUB]=VALUE of a function's nameref with no value =====
# The global NAME becomes an array (a set scalar its element 0), but the element store looks
# NAME up as usual, finds the nameref and is refused as a store to the name "": status 1.
# Without a value it is only declared. Every expected value is bash 5.3's.
s148() {
  local e
  s148s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$(s148f() { local -n q7; declare -g q7[1]=5; echo "a $?"; declare -p q7; }; s148f 2>&1; declare -p q7)
  check gvn1 "$(s148s "$e")" 'declare: `'"'"': not a valid identifier;a 1;declare -n q7;declare -a q7=();'
  e=$(r5=1; s148g() { local -n r5; declare -g r5[2]+=x; echo "b $?"; }; s148g 2>&1; declare -p r5)
  check gvn2 "$(s148s "$e")" 'declare: `'"'"': not a valid identifier;b 1;declare -a r5=([0]="1");'
  e=$(s148h() { local -n q8; declare -g q8[1]; echo "c $?"; declare -p q8; }; s148h 2>&1)
  check gvn3 "$(s148s "$e")" 'c 0;declare -n q8;'
}
# ===== SECTION 149: a name made readonly by code further down =====
# A function that runs after a later line made a name readonly meets it as bash does: its
# `local NAME` is refused and makes no local, so every later declaration, assignment or
# arithmetic store of NAME in the call is refused too, and declare -p shows the global. A
# local made before the readonly is not. Every expected value is bash 5.3's.
s149() {
  local e
  s149s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$(
    s149f() { local s149d; echo "a $?"; declare -a s149d=(1); echo "b $?"; local s149d; echo "c $?"; declare -p s149d; }
    s149g() { s149c=2; echo "d $?"; }
    s149h() { local s149e; s149e=3; echo "e $?"; }
    s149k() { local s149e; (( s149e = 4 )); echo "f $?"; }
    s149m() { local s149m=1; declare -g -r s149m=9; s149m=2; echo "g $s149m"; }
    readonly s149d=1 s149c=1 s149e=1
    s149f 2>&1; (s149g) 2>&1; echo "h $?"; (s149h) 2>&1; echo "i $?"; (s149k) 2>&1; echo "l $?"; s149m; echo "j $s149m"
  )
  check rol1 "$(s149s "$e")" 'local: s149d: readonly variable;a 1;s149d: readonly variable;declare: s149d: readonly variable;b 1;local: s149d: readonly variable;c 1;declare -r s149d="1";s149c: readonly variable;h 1;local: s149e: readonly variable;s149e: readonly variable;i 1;local: s149e: readonly variable;s149e: readonly variable;f 1;l 0;g 2;j 9;'
  e=$( (for i in 1 2; do s149x=$i; echo "x=$s149x"; readonly s149x; done) 2>&1; echo "k $?")
  check rol2 "$(s149s "$e")" 'x=1;s149x: readonly variable;k 1;'
}
# ===== SECTION 150: line numbers of a function's commands inside $( ) =====
# bash runs a $( ) from its printed text, where each command a function body separates with
# ; stands on a line of its own, a subshell's too, and the commands after a body closed on
# its line follow its `}`. An error message keeps the same line. The line prefixes stay in
# these checks. Every expected value is bash 5.3's.
s150() {
  local e l
  l=$LINENO; e=$(s150f() { echo $LINENO; echo $LINENO; (echo $LINENO; echo "x;" $LINENO); echo ${x:-;} $LINENO; }; s150f); check fln1 "${e//$'\n'/,}" "$((l+2)),$((l+3)),$((l+4)),x; $((l+5)),; $((l+6))"
  l=$LINENO; e=$(s150g() { echo $LINENO; echo $LINENO; }; echo $LINENO; s150g
echo $LINENO); check fln2 "${e//$'\n'/,}" "$((l+5)),$((l+3)),$((l+4)),$((l+6))"
  l=$LINENO; e=$(s150h() {
echo $LINENO; echo $LINENO
echo $LINENO
}
s150h); check fln3 "${e//$'\n'/,}" "$((l+6)),$((l+7)),$((l+9))"
  l=$LINENO; e=$(s150k() { readonly s150r=1; s150r=2; echo no; }
s150k 2>&1); check fln4 "${e#*: line }" "$((l+4)): s150r: readonly variable"
}
# ===== END SECTIONS =====

s135  # SECTION-CALL 135
s136  # SECTION-CALL 136
s137  # SECTION-CALL 137
s138  # SECTION-CALL 138
s139  # SECTION-CALL 139
s140  # SECTION-CALL 140
s141  # SECTION-CALL 141
s142  # SECTION-CALL 142
s143  # SECTION-CALL 143
s144  # SECTION-CALL 144
s145  # SECTION-CALL 145
s146  # SECTION-CALL 146
s147  # SECTION-CALL 147
s148  # SECTION-CALL 148
s149  # SECTION-CALL 149
s150  # SECTION-CALL 150
echo "full: $checks checks, $fails failures"
exit $fails
