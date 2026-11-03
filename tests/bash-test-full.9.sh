#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 9
# of 9, sections 176 on; a new section goes here. The split is explained in
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

# ===== SECTION 176: declare -f / -F with array literals and values =====
# A -f / -F query assigns a NAME=(...) operand as the words expand, under the options that
# shape the value (-a/-A/-g/-c/-i/-l/-u), then looks for a function NAME. A NAME=VALUE
# operand is "cannot use `-f' to make functions", ending the builtin, except under -p,
# which looks the word up. With operands and no -p, -n/-i/-A/-a are invalid options, status
# 1. Every expected value is bash 5.3's.
s176() {
  local e
  s176s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  s176g() { echo g; }
  s176h() { echo h; }
  s176a() { local v=7; declare -f q=(1 2); echo "s=$?"; declare -p q; declare -F r=1 s176g; echo "s=$?"
            declare -F s176g r+=1 s176h; echo "s=$?"; declare -f s176g q2=(1) >/dev/null; echo "s=$?"; declare -p q2
            typeset -F t='(1)'; echo "s=$?"; declare -p t; local -f l=(3 4); echo "s=$?"; declare -p l
            declare -ft r[1]=1 s176g; echo "s=$?"; declare -Fp r[$v]=x$v q3=(5) s176h; echo "s=$?"; declare -p q3
            declare -fp r[1]; echo "s=$?"; declare -F x=$(echo side >&2); echo "s=$?"; }
  e=$(s176a 2>&1)
  check fq1 "$(s176s "$e")" 's=1;declare -a q=([0]="1" [1]="2");declare: cannot use `-f'"'"' to make functions;s=1;s176g;declare: cannot use `-f'"'"' to make functions;s=1;s=1;declare -a q2=([0]="1");typeset: cannot use `-f'"'"' to make functions;s=1;declare: t: not found;s=1;declare -a l=([0]="3" [1]="4");declare: cannot use `-f'"'"' to make functions;s=1;declare: r[7]=x7: not found;declare: q3: not found;declare -f s176h;s=1;declare -a q3=([0]="5");declare: r[1]: not found;s=1;side;declare: cannot use `-f'"'"' to make functions;s=1;'
  s176b() { declare -f -i n=(1+1); echo "s=$?"; declare -p n; declare -fu u=(ab); echo "s=$?"; declare -p u
            local -fA m=([a]=1); echo "s=$?"; declare -p m; declare -fni s176g; echo "s=$?"; declare -fAa s176g; echo "s=$?"
            declare -fpa s176g >/dev/null; echo "s=$?"; declare -fia >/dev/null; echo "s=$?"; }
  e=$(s176b 2>&1)
  check fq2 "$(s176s "$e")" 'declare: -i: invalid option;s=1;declare -ai n=([0]="2");s=1;declare -au u=([0]="AB");local: -A: invalid option;s=1;declare -A m=([a]="1" );declare: -n: invalid option;s=1;declare: -A: invalid option;s=1;s=0;s=0;'
}
# ===== SECTION 177: function attributes: declare -f with -x/-r/-c/-l/-u and +ATTR, export -f, readonly -f =====
# A function keeps the attributes r, t, x, c, l and u, printed in that order after `declare
# -f`. With operands and no -p, -f / -F set each attribute given and clear each +ATTR, a
# name that is no function failing silently; any +ATTR, +a and +i too, makes it set rather
# than print, and -c, -l and -u each clear the other two. With no operand an attribute picks
# the functions holding any of them. export -f sets x (-n clears it), readonly -f sets r; for
# both a name that is no function is "not a function". A readonly function is not
# redefined ("NAME: readonly function", inside a function body on the line where the body
# of its last definition starts), unset or given +r, each status 1. A redefinition keeps
# the attributes. Every expected value is bash 5.3's.
s177() {
  local e
  s177s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$( { s177g() { :; }; s177h() { :; }; s177k() { :; }
       declare -f +a s177g; echo "a=$?"; declare -F +i s177g; echo "i=$?"; declare -fx s177g s177h; echo "x=$?"
       declare -ft s177g; declare -fx s177n; echo "n=$?"; declare -Fx; declare -Ftr; declare -f +x s177h; declare -Fp s177h
       declare -fl s177k; declare -Fp s177k; declare -fu s177k; declare -Fp s177k; declare -fcu s177k; declare -Fp s177k
       declare -f -x +x s177k; declare -Fp s177k; declare -fpx s177k; echo "p=$?"; declare -f +g s177k; declare -Fp s177k
       s177h() { echo 2; }; declare -Fx; s177g() { echo 3; }; declare -F s177g; declare -Fp s177g; } 2>&1 )
  check fa1 "$(s177s "$e")" 'a=0;i=0;x=0;n=1;declare -ftx s177g;declare -fx s177h;declare -ftx s177g;declare -f s177h;declare -fl s177k;declare -fu s177k;declare -f s177k;declare -f s177k;s177k () ;{ ;    :;};p=0;s177k () ;{ ;    :;};declare -f s177k;declare -ftx s177g;s177g;declare -ftx s177g;'
  e=$( { s177g() { echo g; }; declare -fr s177g; echo "r=$?"
       s177g() {
         echo h
       }
       echo "d=$? $(s177g)"; ( s177g() { :; }; echo "s=$?" ); declare -f +r s177g; echo "pr=$?"; declare -f +x s177g; echo "px=$?"
       unset -f s177g; echo "u=$?"; unset s177g; echo "u=$?"; declare -Fp s177g; declare -Fr; declare -f -r
       f() { s177g() { :; }; echo "in=$?"; }; f; } 2>&1 )
  check fr1 "$(s177s "$e")" "r=0;s177g: readonly function;d=1 g;s177g: readonly function;s=1;declare: s177g: readonly function;pr=1;px=0;unset: s177g: cannot unset: readonly function;u=1;unset: s177g: cannot unset: readonly function;u=1;declare -fr s177g;declare -fr s177g;s177g () ;{ ;    echo g;};declare -fr s177g;s177g: readonly function;in=1;"
  e=$( { s177g() { :; }; s177h() { :; }; x=1
       export -f s177g; echo "e=$?"; export -f s177n s177h x s177g=1; echo "e=$?"; declare -Fp s177g s177h; export -f; export -pf
       export -nf s177g; declare -Fp s177g; export -fn s177n; echo "n=$?"; readonly -f s177h; echo "r=$?"; readonly -nf s177h; declare -Fp s177h
       readonly -f s177n; echo "r=$?"; readonly -pf; readonly -af s177g; declare -Fp s177g; readonly -fa; echo "ra=$?"; } 2>&1 )
  check fx1 "$(s177s "$e")" 'e=0;export: s177n: not a function;export: x: not a function;export: s177g=1: not a function;e=1;declare -fx s177g;declare -fx s177h;s177g () ;{ ;    :;};declare -fx s177g;s177h () ;{ ;    :;};declare -fx s177h;s177g () ;{ ;    :;};declare -fx s177g;s177h () ;{ ;    :;};declare -fx s177h;declare -f s177g;export: s177n: not a function;n=1;r=0;declare -frx s177h;readonly: s177n: not a function;r=1;s177h () ;{ ;    :;};declare -frx s177h;declare -fr s177g;ra=0;'
  s177r() { declare -fr s177g; local l0=$LINENO
            s177g() {
              :
            }
            function   s177g
            {
              :
            }; echo "$l0"; }
  e=$(s177g() { :; }; s177r 2>&1)
  local l=${e##*$'\n'} o= x
  while IFS= read -r x; do o+="${x#*: line };"; done <<< "$e"
  check fr2 "$o" "$((l + 5)): s177g: readonly function;$((l + 5)): s177g: readonly function;$l;"
}
# ===== SECTION 178: declare -f / -F operands, +p/+f/+F, and where a readonly function is reported =====
# In a -f / -F query every word after the options names a function, whatever it spells: an
# option word or a word that is no identifier after an operand is a name looked up like any
# other (silently missing, "not found" under -p, "not a function" for export -f and readonly
# -f). +p prints as -p does; +f and +F change nothing, so a name is only checked (status 1
# for no function). A readonly function's redefinition inside a `for NAME` or `case` is
# reported on that command's first line, and outside every function on the last line of its
# command line (here, an eval string's). Every expected value is bash 5.3's.
s178() {
  local e
  s178s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$( { s178g() { :; }; s178h() { :; }; function s178a-b { :; }
       declare -F s178g +r s178h; echo "s=$?"; declare -Fp s178g -x s178h; echo "s=$?"; declare -F s178g -- s178h; echo "s=$?"
       declare -Fp s178g 'x y' "-z"; echo "s=$?"; declare -F s178a-b; echo "s=$?"; declare -Fp -- -x s178a-b; echo "s=$?"
       export -f s178g -y; echo "s=$?"; readonly -f s178h +q; echo "s=$?"; declare -Fp s178h; } 2>&1 )
  check fo1 "$(s178s "$e")" 's178g;s178h;s=1;declare -f s178g;declare: -x: not found;declare -f s178h;s=1;s178g;s178h;s=1;declare -f s178g;declare: x y: not found;declare: -z: not found;s=1;s178a-b;s=0;declare: -x: not found;declare -f s178a-b;s=1;export: -y: not a function;s=1;readonly: +q: not a function;s=1;declare -fr s178h;'
  e=$( { s178g() { :; }; declare -fx s178g
       declare -F +p s178g; echo "s=$?"; declare -f +p s178g; echo "s=$?"; declare -F +p s178n; echo "s=$?"
       declare -F +f s178g; echo "s=$?"; declare -F +f s178n; echo "s=$?"; declare -f +F s178g; echo "s=$?"
       declare -F +F +p s178g; echo "s=$?"; declare -F +x +p s178g; echo "s=$?"; declare -F +f s178g y=1; echo "s=$?"; } 2>&1 )
  check fo2 "$(s178s "$e")" 'declare -fx s178g;s=0;s178g () ;{ ;    :;};declare -fx s178g;s=0;declare: s178n: not found;s=1;s=0;s=1;s=0;declare -fx s178g;s=0;declare -fx s178g;s=0;declare: cannot use `-f'"'"' to make functions;s=1;'
  s178r() { s178g() { :; }; declare -fr s178g; local l0=$LINENO
            eval 'if true; then
s178g() { :; }
:
fi'
            eval '{
s178g() { :; }
} ; :
:'
            for i in 1; do
              :
              s178g() { :; }
            done
            case a in
              a)
                s178g() { :; } ;;
            esac
            echo "$l0"; }
  e=$(s178r 2>&1)
  local l=${e##*$'\n'} o= x
  while IFS= read -r x; do o+="${x#*: line };"; done <<< "$e"
  check fo3 "$o" "$((l + 4)): s178g: readonly function;$((l + 7)): s178g: readonly function;$((l + 9)): s178g: readonly function;$((l + 13)): s178g: readonly function;$l;"
}
# ===== SECTION 179: the operands of a -f / -F query spelled from fields, and "--" before them =====
# A declaration whose first operand is a quoted option word reads its words as fields: under
# -f / -F -p each is looked up in operand order, so a "not found" sits between the functions
# printed (a quote, an empty field and a "--" after the options spell no other word). A
# written "--" ends the options for a computed field after it too, which is then an operand.
# Every expected value is bash 5.3's.
s179() {
  local e
  s179s() { local l; while IFS= read -r l; do case $l in *": line "*) l=${l#*: line }; l=${l#*: } ;; esac; printf '%s;' "$l"; done <<< "$1"; }
  e=$( { s179g() { :; }; s179h() { :; }
       declare -F '-p' s179g 'x y' s179h; echo "s=$?"; declare -F '-p' s179g "it's" '' s179h; echo "s=$?"
       typeset -F '-p' s179g 'a b' s179h >/dev/null; echo "s=$?"; declare -F '-p' -- '-x' s179g; echo "s=$?"; } 2>&1 )
  check fp1 "$(s179s "$e")" "declare -f s179g;declare: x y: not found;declare -f s179h;s=1;declare -f s179g;declare: it's: not found;declare: : not found;declare -f s179h;s=1;typeset: a b: not found;s=1;declare: -x: not found;declare -f s179g;s=1;"
  e=$( { v=-x; declare -- "$v" s179q=1; echo "s=$?"; declare -p s179q; v=-p; declare -- "$v"; echo "s=$?"; } 2>&1 )
  check fp2 "$(s179s "$e")" "declare: \`-x': not a valid identifier;s=1;declare -- s179q=\"1\";declare: \`-p': not a valid identifier;s=1;"
}
# ===== END SECTIONS =====

s176  # SECTION-CALL 176
s177  # SECTION-CALL 177
s178  # SECTION-CALL 178
s179  # SECTION-CALL 179
echo "full: $checks checks, $fails failures"
exit $fails
