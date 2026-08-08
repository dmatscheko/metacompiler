#!/usr/bin/env bash
# See mod.kt. Same loop - `s = s + i % 7`, 40,000 iterations, no float, no
# regexp, no allocation of a data structure in the body.
#
# 1. WHAT THIS ROW MEASURES IS NOT WHAT THE HANDLE-IR ROWS MEASURE. bash compiles
# to SELF-CONTAINED IR against languages/lib/bash-rt.c: unboxed machine words, a
# bump-allocated arena, no js_* extern and no MetaJS layer 2 at all (manual
# chapter 1). So this row belongs with mod.c and mod.bat, the other two
# self-contained programs, and NOT beside kotlin or python - a layer-2 change
# cannot move it, and its absolute number is small for that reason. It is here
# because bash-rt.c had no performance gate of any kind, and it is one of the two
# runtimes with an UNCOLLECTED arena (manual 7.8) - exactly the shape an
# instruction count catches.
#
# 2. THE ARENA CEILING, MEASURED, AND WHY THE COUNT IS 40,000 AGAIN.
#
# `arena[4194304]` in bash-rt.c is 4 MiB and nothing is ever freed, so the
# question "how many iterations fit" has an exact answer and it used to be a bad
# one. When this row was first added the loop cost ~202 bytes of arena per
# iteration, the ceiling was ~20,750, and the 40,000-iteration version of this
# file EXITED 139 HAVING PRINTED NOTHING - `/usr/bin/time` reporting happily on a
# process that died, the manual chapter 4 trap live in the first row measured.
# The file was cut to 10,000 to get under it.
#
# Two things changed and both are measured:
#
#   * rt_bump IS BOUNDS-CHECKED NOW. Running out of arena prints
#         bash-rt: string arena exhausted: request 6, used 4194300 of 4194304 ...
#     on stderr and exits 70. The silent 139 is gone, so a crash in this row can
#     no longer be misread as anything else, and bench.sh's exit-code check
#     surfaces the reason rather than just the code.
#
#   * THE LOOP COSTS 69.6 BYTES AN ITERATION, DOWN FROM 204.8. Measured as the
#     slope of maximum-RSS against iteration count (the arena is BSS, so only the
#     pages actually bumped become resident, which makes RSS an exact
#     instrument here): 2,000 and 14,000 iterations, 1,769,472 -> 3,063,808 bytes
#     before, 1,703,936 -> 2,539,520 after. Three changes to bash-rt.c did it,
#     none of them to this file: rt_strcat and rt_substr no longer copy when one
#     operand is empty or the slice is the whole string, rt_int2str bumps the
#     digits it needs instead of a flat 16, and rt_splitifs sizes its output in
#     one pass instead of building it with three rt_strcats per field (and
#     returns the value untouched when it is a single separator-free field, which
#     a plain string already is in that protocol).
#
# So the ceiling is now 55,898 iterations - bisected on this machine, 55,898
# runs and 56,015 does not - and 40,000 sits under it at a 1.40x margin. That is
# thinner than the 2x this file used to keep, and it is deliberate: the reason a
# thin margin was dangerous was that crossing it was SILENT, and it no longer is.
# Read a nonzero exit from this row as arena exhaustion first, and the message on
# stderr will say so outright.
#
# IF YOU RAISE bash's PER-ITERATION ARENA COST, this row crosses the ceiling
# before it gets 40% slower, and it will say `RAN AND FAILED (exit 70)`. Lower
# the count here and re-record, or take the cost back out; do not delete the row.
s=0
i=0
while [ $i -lt 40000 ]; do
  s=$(( s + i % 7 ))
  i=$(( i + 1 ))
done
echo $s
