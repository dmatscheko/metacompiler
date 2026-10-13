#!/usr/bin/env bash
# A loop inside a loop, twenty thousand inner passes, each holding a regex match's 512
# bytes of capture slots. Every stack slot of a compiled function lives in its entry
# block; one made where it is used would be a fresh alloca on every pass and overflow
# the native stack long before the end. Checked from outside: tests/native-full.sh
# compares the native binary's stdout with tests/bash-stack.expected (real bash 5.3's
# output). Native only, and named outside tests/bash-test-* for that reason: twenty
# thousand regex matches are past the IR interpreter's step brake.
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
