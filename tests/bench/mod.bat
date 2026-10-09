@echo off
rem See mod.kt. Batch has no while loop: a label and a guarded goto. batch-rt.c's
rem arena is uncollected: exit 70 means it is exhausted - lower the count, do not
rem delete the row.
set /a s=0
set /a i=0
:loop
if %i% GEQ 40000 goto done
set /a s=s + i %% 7
set /a i=i + 1
goto loop
:done
echo %s%
