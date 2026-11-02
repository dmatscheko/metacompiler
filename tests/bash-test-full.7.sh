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
# ===== END SECTIONS =====

s135  # SECTION-CALL 135
s136  # SECTION-CALL 136
s137  # SECTION-CALL 137
echo "full: $checks checks, $fails failures"
exit $fails
