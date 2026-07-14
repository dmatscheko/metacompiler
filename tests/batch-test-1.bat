@echo off
rem Self-checking test for the Batch interpreter. Every check echoes a PASS line; a
rem failure bumps %fails%. The script ends with `exit /b %fails%`, so a green run is
rem exit code 0 and the output is deterministic (byte-identical under goja and -frozen).
set fails=0

rem ----- variables, echo, string comparison -----
set x=5
if "%x%"=="5" (echo PASS str-eq) else (set /a fails=fails+1)
if "%x%"=="6" (set /a fails=fails+1) else (echo PASS str-ne)

rem ----- set /a arithmetic (+, -, *, /, parentheses) -----
set /a y=x * 2 + 1
if %y% EQU 11 (echo PASS arith-muladd) else (set /a fails=fails+1)
set /a z=(y - 1) / 2
if %z% EQU 5 (echo PASS arith-parendiv) else (set /a fails=fails+1)
set /a neg=0 - 4
if %neg% LSS 0 (echo PASS arith-neg) else (set /a fails=fails+1)

rem ----- numeric comparisons EQU NEQ LSS LEQ GTR GEQ -----
if %y% GEQ 11 (echo PASS geq) else (set /a fails=fails+1)
if %y% GTR 10 (echo PASS gtr) else (set /a fails=fails+1)
if %x% LSS 10 (echo PASS lss) else (set /a fails=fails+1)
if %x% LEQ 5 (echo PASS leq) else (set /a fails=fails+1)
if %x% NEQ 6 (echo PASS neq) else (set /a fails=fails+1)

rem ----- if / else / not / defined -----
if %x% EQU 99 (set /a fails=fails+1) else (echo PASS else-branch)
if defined x (echo PASS defined) else (set /a fails=fails+1)
if not defined nope (echo PASS not-defined) else (set /a fails=fails+1)

rem ----- for loop with accumulation (copy the %%i value into a var first) -----
set /a sum=0
for %%i in (1 2 3 4) do (
    set cur=%%i
    set /a sum=sum+cur
)
if %sum% EQU 10 (echo PASS for-sum) else (set /a fails=fails+1)

rem ----- goto skips over a block -----
goto past
echo SHOULD-NOT-PRINT
set /a fails=fails+1
:past
echo PASS goto

rem ----- call a subroutine (communicates through globals; no %1 params) -----
set /a arg=6
call :double
if %dbl% EQU 12 (echo PASS call) else (set /a fails=fails+1)

echo failures: %fails%
exit /b %fails%

:double
set /a dbl=arg * 2
exit /b 0
