#!/usr/bin/env bash
# A loop inside a loop, twenty thousand inner passes, each holding a regex match's 512
# bytes of capture slots. Every stack slot of a compiled function lives in its entry
# block; one made where it is used would be a fresh alloca on every pass and overflow
# the native stack long before the end. Checked from outside: tests/native-full.sh
# compares the native binary's stdout with tests/bash-stack.expected (real bash 5.3's
# output). Native only, and named outside tests/bash-test-* for that reason: twenty
# thousand regex matches are past the IR interpreter's step brake. Then subjects of
# 70,000 bytes: the regex VM's backtracking lives on a stack of its own, and its step
# limit (per start position) scales with the subject, so linear work - and a
# search whose many starts each do linear work - is never cut off.
n=0
m=0
while [ $n -lt 20 ]; do
  k=0
  while [ $k -lt 1000 ]; do
    [[ ab =~ x ]] || m=$k
    k=$(( k + 1 ))
  done
  n=$(( n + 1 ))
done
echo "done $n $k $m"
s=ab
while (( ${#s} < 70000 )); do s=$s$s; done
s=${s:0:70000}
[[ $s =~ (a|b)(a|b)+ ]]; echo "$? ${#BASH_REMATCH[0]} ${BASH_REMATCH[2]}"
[[ $s =~ ^([ab]b?)+$ ]]; echo "$? ${#BASH_REMATCH[0]} ${BASH_REMATCH[1]}"
[[ ${s}x =~ (ab)+x ]]; echo "$? ${#BASH_REMATCH[0]}"
t=${s:0:1000}
[[ ${t}x =~ (ab)+c|x ]]; echo "$? ${BASH_REMATCH[0]}"
