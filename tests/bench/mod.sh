#!/usr/bin/env bash
# See mod.kt. bash-rt.c's string arena is uncollected: if this row reads
# RAN AND FAILED (exit 70) the arena is exhausted - lower the count, do not
# delete the row.
s=0
i=0
while [ $i -lt 40000 ]; do
  s=$(( s + i % 7 ))
  i=$(( i + 1 ))
done
echo $s
