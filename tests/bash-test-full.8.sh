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
# ===== END SECTIONS =====

s154  # SECTION-CALL 154
s155  # SECTION-CALL 155
s156  # SECTION-CALL 156
echo "full: $checks checks, $fails failures"
exit $fails
