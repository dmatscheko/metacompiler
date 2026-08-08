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
rem else. `AR[2097152]` in batch-rt.c is 2 MiB, `rt_bump` has no bounds check
rem (its own header says so) and nothing is freed. This loop costs ~32 bytes of
rem arena per iteration; bisected on this machine the last count that runs is
rem ~65,468 and the first that does not is ~65,625, past which the binary
rem SEGFAULTS (exit 139) printing nothing. 40,000 is only a 1.6x margin - so if
rem this row ever reads RAN AND FAILED, suspect arena exhaustion from a change to
rem batch-rt.c's per-assignment allocation BEFORE suspecting the emitter, and
rem lower the count here rather than deleting the row. (mod.sh had to be cut to
rem 10,000 for exactly this reason; bash's arena is 4 MiB but costs ~202 bytes an
rem iteration.)
set /a s=0
set /a i=0
:loop
if %i% GEQ 40000 goto done
set /a s=s + i %% 7
set /a i=i + 1
goto loop
:done
echo %s%
