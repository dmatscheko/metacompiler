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
# ===== END SECTIONS =====

s176  # SECTION-CALL 176
echo "full: $checks checks, $fails failures"
exit $fails
