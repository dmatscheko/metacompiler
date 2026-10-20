@echo off
rem Output captures nested twelve deep: every level of the recursion is a call
rem redirected to nul, which keeps a capture open for the whole callee, and the
rem innermost level adds a for /f command capture on top. Checked from outside:
rem tests/native-full.sh runs the native binary with 2>&1 and compares with
rem tests/batch-nest.expected (the interpreter half's output; no cmd.exe here),
rem so a capture-stack limit's diagnostic or a lost line both show.
echo before
set inner=none
call :r 12
echo after
echo inner=%inner%
goto :eof

:r
echo hidden %1
if %1 GTR 1 goto more
for /f %%a in ('echo deep') do set inner=%%a
goto :eof
:more
set /a n=%1-1
call :r %n% > nul
goto :eof
