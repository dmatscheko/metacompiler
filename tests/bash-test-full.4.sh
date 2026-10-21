#!/usr/bin/env bash
# Full-syntax test: Bash (the practical POSIX-plus-bash shell language) - part 4
# of 4, section 61, whose sizes alone take ~65M of the step brake. The split is
# explained in tests/bash-test-full.sh.
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

# ===== SECTION 61: long lists, long words, long accumulations =====
# Each of these is linear in its length in bash, and has to be in a compiled program
# too: a variable appended to in a loop, a loop over a long list, thousands of
# positional parameters, "$@" / "$*" and printf over them, a command of thousands of
# words, a quoted word of hundreds of parts and an array expanded whole. The sizes fit the IR interpreter's step brake only
# when every one of them is linear. A compiled `s+=x` appends in place, so the last
# assertion pins what must not see that. Every expected value is bash 5.3's.
s61() {
  local s="" w n=0 i t u a
  for ((i = 0; i < 1000; i++)); do s+="w$((i % 10)) "; done
  for w in $s; do n=$((n + 1)); done
  check ln1 "${#s} $n $w" "3000 1000 w9"
  set -- $s $s $s $s $s
  t="$*"
  check ln2 "$# $1 ${4500} ${@: -2:1} ${#t} [${t:0:6}]" "5000 w0 w9 w8 14999 [w0 w1 ]"
  printf -v u '%*s' 200
  w="$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u.$u."
  check ln3 "${#w} [${w:199:3}]" "60300 [ . ]"
  a=($s)
  t="${a[*]}"
  set -- "${a[@]}"
  check ln4 "${#a[@]} ${#t} $# ${a[999]} ${1000}" "1000 2999 1000 w9 w9"
  printf -v u '%s,' "$@"
  printf -v w '%b' "\\x41$s\\x42"
  printf -v t '%s,' {1..2000}
  check ln5 "${#u} [${u:0:6}] ${#w} [${w:0:4}] [${w: -3}] ${#t} ${t: -5}" "3000 [w0,w1,] 3002 [Aw0 ] [9 B] 8893 2000,"
  # An appended-to variable shares nothing with the copies taken of it: another
  # variable, an array element, a suffix of it, a positional parameter, a subshell,
  # a function's local of the same name, itself and a here-document.
  s=ab; s+=cd; t=$s; s+=ef
  w="$t $s"
  s+=gh; a=("$s" "${s#a}"); s+=ij
  w+=" ${a[*]} $s"
  set -- "$s"; s+=kl
  w+=" $1 $s"
  u=$(s+=sub; echo "$s"); s+=mn
  w+=" $u $s"
  s61l; s+=op
  w+=" $s"
  s+=$s; s+=${s:2:2}
  w+=" $s"
  while read -r u; do s+=W; w+=" $u"; done <<EOF
$s
EOF
  check ln6 "$w $s" "abcd abcdef abcdefgh bcdefgh abcdefghij abcdefghij abcdefghijkl abcdefghijklsub abcdefghijklmn inLM abcdefghijklmnop abcdefghijklmnopabcdefghijklmnopcd abcdefghijklmnopabcdefghijklmnopcd abcdefghijklmnopabcdefghijklmnopcdW"
}
s61l() { local s=in; s+=L; s+=M; w+=" $s"; }
# ===== END SECTIONS =====

s61   # SECTION-CALL 61
echo "full: $checks checks, $fails failures"
exit $fails
