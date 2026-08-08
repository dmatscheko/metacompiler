@echo off
rem See mod.kt. Same loop - `s = s + i %% 7`, 40,000 iterations, no float, no
rem allocation of a data structure in the body - so the shape matches the other
rem programs. Two things about the row are NOT the same as theirs.
rem
rem 1. WHAT THIS ROW MEASURES IS NOT WHAT THE HANDLE-IR ROWS MEASURE. batch
rem compiles to SELF-CONTAINED IR against languages/lib/batch-rt.c - unboxed
rem machine words, a bump arena, no js_* extern and no MetaJS layer 2 (manual
rem chapter 1) - so it belongs beside mod.c and mod.sh, never beside kotlin or
rem python. A layer-2 change cannot move this row. It is here because batch-rt.c
rem had no performance gate at all and its arena is uncollected (manual 7.8).
rem
rem 2. THE SHAPE DIFFERENCES, both forced by the language:
rem   - Batch has no while loop. The nearest idiomatic thing is a label plus a
rem     guarded `goto`, which the compiler lowers to basic blocks - the same
rem     control flow the other programs' `while` gets.
rem   - Every variable is a STRING. `set /a` parses the decimal text and writes
rem     it back, so like mod.sh this loop exercises the runtime's int/string
rem     conversion where mod.c does not.
rem
rem THE ARENA CEILING, measured, because a crash here will look like something
rem else. `AR[2097152]` in batch-rt.c is 2 MiB and nothing is ever freed, so the
rem question "how many iterations fit" has an exact answer. Two things changed
rem since this row was added and both are measured:
rem
rem   * rt_bump IS BOUNDS-CHECKED NOW. Running out of arena prints
rem       batch-rt: string arena exhausted: request 16, used 2097152 of 2097152 ...
rem     on stderr and exits 70. It used to exit 139 printing nothing, which is
rem     the manual chapter 4 trap ("a crashing binary looks fast"): the ceiling
rem     was 1.6x away and the next person to add a few bytes an iteration would
rem     have got a segfault with no diagnosis in a checked-in bench row.
rem
rem   * THE LOOP COSTS 13.1 BYTES AN ITERATION, DOWN FROM 31.95. Measured as the
rem     slope of maximum RSS against iteration count (the arena is BSS, so only
rem     the pages actually bumped become resident, which makes RSS exact here):
rem     10,000 and 50,000 iterations, 1,802,240 -> 3,080,192 bytes before,
rem     1,572,864 -> 2,097,152 after. Three changes to batch-rt.c did it, none of
rem     them to this file: rt_strcat no longer copies when an operand is empty,
rem     rt_int2str bumps the digits it needs instead of a flat 16, and rt_stripq
rem     returns an unquoted string unchanged instead of copying it.
rem
rem So the ceiling is now 160,351 iterations - bisected on this machine, 160,351
rem runs and 160,497 does not - and the standard 40,000 sits under it at a 4.0x
rem margin, up from 1.6x. If this row ever reads RAN AND FAILED, read the exit
rem code: 70 is arena exhaustion from a change to batch-rt.c's per-assignment
rem allocation, and it is to be suspected BEFORE the emitter. Lower the count
rem here rather than deleting the row.
set /a s=0
set /a i=0
:loop
if %i% GEQ 40000 goto done
set /a s=s + i %% 7
set /a i=i + 1
goto loop
:done
echo %s%
