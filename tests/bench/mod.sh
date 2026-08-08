#!/usr/bin/env bash
# See mod.kt. Same loop - `s = s + i % 7`, no float, no regexp, no allocation of
# a data structure in the body - but at 10,000 iterations and NOT 40,000. Both
# differences are deliberate and are explained below; read them before comparing
# this number to anything.
#
# 1. WHAT THIS ROW MEASURES IS NOT WHAT THE HANDLE-IR ROWS MEASURE. bash compiles
# to SELF-CONTAINED IR against languages/lib/bash-rt.c: unboxed machine words, a
# bump-allocated arena, no js_* extern and no MetaJS layer 2 at all (manual
# chapter 1). So this row belongs with mod.c, the self-contained control, and NOT
# beside kotlin or python - a layer-2 change cannot move it, and its absolute
# number is small for that reason. It is here because bash-rt.c had no
# performance gate of any kind, and it is one of the two runtimes with an
# UNCOLLECTED arena (manual 7.8) - exactly the shape an instruction count catches.
#
# 2. WHY 10,000 AND NOT 40,000, WHICH IS A FINDING AND NOT A PREFERENCE.
# `arena[4194304]` in bash-rt.c is 4 MiB, `rt_bump` has no bounds check, and
# nothing is ever freed. This loop costs ~202 bytes of arena per iteration -
# `$(( ))` re-parses the accumulator's decimal text and rt_int2str writes a fresh
# arena string for every assignment, because bash has no typed integer variable.
# Bisected on this machine: the last iteration count that runs is ~20,750 and the
# first that does not is ~20,781, and past it the binary SEGFAULTS (exit 139)
# with no diagnostic at all - it just stops printing. The 40,000-iteration
# version of this file dies that way. 10,000 leaves a 2x margin.
#
# That margin is itself part of the gate: if someone raises bash's per-iteration
# arena cost by 2x, this row does not get slower, it CRASHES, and bench.sh checks
# the exit code precisely because "a crashing binary looks fast" (manual chapter
# 4). Read a crash here as arena exhaustion first.
s=0
i=0
while [ $i -lt 10000 ]; do
  s=$(( s + i % 7 ))
  i=$(( i + 1 ))
done
echo $s
