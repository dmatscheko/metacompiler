#!/usr/bin/env bash
# See mod.kt. bash-rt.c keeps every value a pass stores (s and i), so its
# string heap grows with the count - keep it modest, and if this row reads
# RAN AND FAILED (exit 70) lower the count, do not delete the row.
s=0
i=0
while [ $i -lt 40000 ]; do
  s=$(( s + i % 7 ))
  i=$(( i + 1 ))
done
echo $s
