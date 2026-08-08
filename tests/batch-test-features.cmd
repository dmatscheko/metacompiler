@echo off
rem Fast feature-matrix test for the Batch interpreter (batch-interpreter.abnf) and
rem the LLVM-IR compiler (batch-to-llvm-ir.abnf). Same protocol as every other
rem tests/<lang>-test-features.*: every implemented construct is exercised with the
rem SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times,
rem recursion stays below depth 6. A failed check prints its id and the file ends
rem with 'exit /b %fails%'; exit 0 and byte-identical output on all four legs
rem (interpreter/compiler x goja/-frozen) mean everything passed.
rem
rem WHY THIS FILE EXISTS (docs/todo.md 4.1)
rem
rem batch's layer 2 is the C runtime languages/lib/batch-rt.c, and until this file
rem existed the ONLY program that ever exercised it was tests/batch-test-full.cmd -
rem the same single-gate hole item 4.1 names for bash. batch was also missing from
rem tests/native-full.sh's ROWS entirely; both are fixed together. The assertions
rem below are aimed at the runtime's entry points on purpose: rt_subst and
rem rt_substr for %v:a=b% and %v:~n,m%, rt_mods for the %~dpnx modifiers, rt_tokb
rem and rt_isdelim for 'for /f' tokenising, rt_lineat and rt_nlines for its
rem multi-line string sets, rt_streqi for case-insensitive 'if ==' and 'if /i',
rem rt_fskind for 'if exist', rt_stripq for %~1, rt_expand for delayed expansion,
rem rt_capstart/rt_capend for 'for /f' over a command, and bat_shift for 'shift'.
rem
rem GROUND TRUTH, and this is the honest limitation: there is no cmd.exe on the
rem development machine, so - exactly as tests/batch-test-full.cmd says in its own
rem header - none of these assertions could be executed against the real shell.
rem Nothing NEW is claimed here: every form below is one the ratchet already
rem settled from the cmd.exe help texts and the wine-cmd-tests corpus under
rem tests/reference/batch, condensed into the small file where a freshly fixed
rem defect can get its assertion cheaply. Anything the ratchet left out for being
rem ambiguous is left out here too.
rem
rem Out of scope, same reasons as the ratchet: external programs, creating or
rem reading files, directory enumeration (so no 'for /r' and no 'for /d'), the
rem %~a %~t %~z modifiers (they stat a file), interactive input, %RANDOM% and the
rem date/time variables, codepages and the registry.

set /a fails=0
set /a checks=0
goto main

rem check: compares %got% against %want% and reports %cid%, through variables
rem rather than through %1..%3, so the helper itself needs only set, if and call.
:check
set /a checks=checks+1
if not "%got%"=="%want%" echo FAIL %cid%
if not "%got%"=="%want%" set /a fails=fails+1
exit /b 0

rem ----- echo and quoting -----
:fe01
set cid=ech1
set got=plain
set want=plain
call :check
set cid=ech2
set got=a b
set want=a b
call :check
set cid=ech3
set got="quoted"
set want="quoted"
call :check
exit /b 0

rem ----- set and set /a -----
:fe02
set x=5
set cid=set1
set got=%x%
set want=5
call :check
set /a y=x*2+1
set cid=set2
set got=%y%
set want=11
call :check
set /a y=(2+3)*4
set cid=set3
set got=%y%
set want=20
call :check
set /a y=17/5
set cid=set4
set got=%y%
set want=3
call :check
set /a "y=17 %% 5"
set cid=set5
set got=%y%
set want=2
call :check
set /a y=-7/2
set cid=set6
set got=%y%
set want=-3
call :check
set /a "y=1 << 4"
set cid=set7
set got=%y%
set want=16
call :check
set /a "y=255 >> 4"
set cid=set8
set got=%y%
set want=15
call :check
set /a "y=12 & 10"
set cid=set9
set got=%y%
set want=8
call :check
set /a "y=12 | 3"
set cid=set10
set got=%y%
set want=15
call :check
set /a "y=12 ^ 10"
set cid=set11
set got=%y%
set want=6
call :check
set /a y=~0
set cid=set12
set got=%y%
set want=-1
call :check
set /a y=0x10
set cid=set13
set got=%y%
set want=16
call :check
set /a y=010
set cid=set14
set got=%y%
set want=8
call :check
set /a y=3
set /a y+=4
set cid=set15
set got=%y%
set want=7
call :check
set /a y*=3
set cid=set16
set got=%y%
set want=21
call :check
set /a y-=1
set cid=set17
set got=%y%
set want=20
call :check
set /a y=1,y=y+5
set cid=set18
set got=%y%
set want=6
call :check
set cid=set19
set undefinedvar=
set got=[%undefinedvar%]
set want=[]
call :check
exit /b 0

rem ----- substitution and substrings: rt_subst, rt_substr -----
:fe03
set p=C:\dir\sub\file.txt
set cid=sst1
set got=%p:\=/%
set want=C:/dir/sub/file.txt
call :check
set cid=sst2
set got=%p:.txt=.log%
set want=C:\dir\sub\file.log
call :check
set cid=sst3
set got=%p:nomatch=X%
set want=C:\dir\sub\file.txt
call :check
set s=abcdefgh
set cid=sst4
set got=%s:~0,3%
set want=abc
call :check
set cid=sst5
set got=%s:~3%
set want=defgh
call :check
set cid=sst6
set got=%s:~-2%
set want=gh
call :check
set cid=sst7
set got=%s:~2,-2%
set want=cdef
call :check
set cid=sst8
set got=%s:~0,0%
set want=
call :check
exit /b 0

rem ----- if forms, including /i and numeric comparison: rt_streqi -----
:fe04
set cid=if1
set got=no
if "a"=="a" set got=yes
set want=yes
call :check
set cid=if2
set got=no
if not "a"=="b" set got=yes
set want=yes
call :check
set cid=if3
set got=no
if /i "ABC"=="abc" set got=yes
set want=yes
call :check
set cid=if4
set got=no
if 5 EQU 5 set got=yes
set want=yes
call :check
set cid=if5
set got=no
if 4 LSS 5 set got=yes
set want=yes
call :check
set cid=if6
set got=no
if 6 GTR 5 set got=yes
set want=yes
call :check
set cid=if7
set got=no
if 5 GEQ 5 set got=yes
set want=yes
call :check
set cid=if8
set got=no
if 5 LEQ 5 set got=yes
set want=yes
call :check
set cid=if9
set got=no
if 5 NEQ 6 set got=yes
set want=yes
call :check
set defined_one=x
set cid=if10
set got=no
if defined defined_one set got=yes
set want=yes
call :check
set cid=if11
set got=no
if not defined never_defined_here set got=yes
set want=yes
call :check
set cid=if12
set got=neither
if "1"=="2" (set got=then) else (set got=else)
set want=else
call :check
set cid=if13
set got=neither
if "1"=="1" (set got=then) else (set got=else)
set want=then
call :check
exit /b 0

rem ----- if exist: rt_fskind -----
:fe05
set cid=ex1
set got=no
if exist nul set got=yes
set want=yes
call :check
set cid=ex2
set got=no
if not exist C:\no\such\path\here set got=yes
set want=yes
call :check
exit /b 0

rem ----- for /l : 0, 1, 3 and 4 iterations -----
:fe06
set /a acc=0
for /l %%i in (1,1,4) do set /a acc=acc+%%i
set cid=forl1
set got=%acc%
set want=10
call :check
set /a acc=0
for /l %%i in (1,1,1) do set /a acc=acc+1
set cid=forl2
set got=%acc%
set want=1
call :check
set /a acc=0
for /l %%i in (5,1,1) do set /a acc=acc+1
set cid=forl3
set got=%acc%
set want=0
call :check
set /a acc=0
for /l %%i in (10,-3,1) do set /a acc=acc+1
set cid=forl4
set got=%acc%
set want=4
call :check
exit /b 0

rem ----- plain for over a word list, and the variable modifiers: rt_mods -----
:fe07
rem %acc% would expand ONCE before the loop, so the accumulator goes through a
rem subroutine - the same shape the ratchet uses.
set acc=
for %%w in (x y z) do call :fe07cat %%w
set cid=forw1
set got=%acc%
set want=xyz
call :check
set /a n=0
for %%w in (only) do set /a n=n+1
set cid=forw2
set got=%n%
set want=1
call :check
for %%f in (C:\dir\sub\file.txt) do set nm=%%~nf
set cid=mod1
set got=%nm%
set want=file
call :check
for %%f in (C:\dir\sub\file.txt) do set xt=%%~xf
set cid=mod2
set got=%xt%
set want=.txt
call :check
for %%f in (C:\dir\sub\file.txt) do set dr=%%~df
set cid=mod3
set got=%dr%
set want=C:
call :check
for %%f in (C:\dir\sub\file.txt) do set pa=%%~pf
set cid=mod4
set got=%pa%
set want=\dir\sub\
call :check
for %%f in (C:\dir\sub\file.txt) do set nx=%%~nxf
set cid=mod5
set got=%nx%
set want=file.txt
call :check
for %%f in ("quoted.txt") do set uq=%%~f
set cid=mod6
set got=%uq%
set want=quoted.txt
call :check
exit /b 0
:fe07cat
set acc=%acc%%1
exit /b 0

rem ----- for /f over strings: rt_tokb, rt_isdelim, rt_lineat, rt_nlines -----
:fe08
for /f "tokens=1" %%a in ("one two three") do set t1=%%a
set cid=forf1
set got=%t1%
set want=one
call :check
for /f "tokens=2" %%a in ("one two three") do set t2=%%a
set cid=forf2
set got=%t2%
set want=two
call :check
for /f "tokens=1,3" %%a in ("one two three") do set t3=%%a-%%b
set cid=forf3
set got=%t3%
set want=one-three
call :check
for /f "tokens=1*" %%a in ("head the whole rest") do set t4=%%b
set cid=forf4
set got=%t4%
set want=the whole rest
call :check
for /f "delims=," %%a in ("a,b,c") do set t5=%%a
set cid=forf5
set got=%t5%
set want=a
call :check
for /f "tokens=2 delims=," %%a in ("a,b,c") do set t6=%%a
set cid=forf6
set got=%t6%
set want=b
call :check
set /a nl=0
for /f "tokens=*" %%a in ("single") do set /a nl=nl+1
set cid=forf7
set got=%nl%
set want=1
call :check
exit /b 0

rem ----- delayed expansion: rt_expand -----
rem
rem NOTE, and it cost three silent assertions to find: `endlocal` restores the
rem WHOLE environment, so a `set /a checks=checks+1` executed between setlocal and
rem endlocal is thrown away with everything else - the check runs, prints its FAIL,
rem and then vanishes from the count and from %fails%, which is a failure the exit
rem status cannot see. So nothing calls :check inside a setlocal region here.
rem The probe does its own comparing and reports through its exit code, and the
rem assertion is made outside. Each probe calls `endlocal` EXPLICITLY on both
rem paths: whether `exit /b` pops an open setlocal by itself is exactly the kind of
rem thing no oracle on this machine can settle (our engines leave it open; the
rem cmd.exe help text says the changes are discarded when the batch file ends),
rem so nothing here depends on it either way.
:fe09
call :fe09probe
set cid=dly1
set got=%errorlevel%
set want=0
call :check
exit /b 0

:fe09probe
setlocal enabledelayedexpansion
set /a acc=0
for /l %%i in (1,1,3) do (
  set /a acc=!acc!+%%i
)
if not "!acc!"=="6" goto fe09bad
set dv=start
set dv=changed
if not "!dv!"=="changed" goto fe09bad
endlocal
exit /b 0
:fe09bad
endlocal
exit /b 1

rem ----- setlocal / endlocal scoping -----
:fe10
set scoped=outer
call :fe10probe
set cid=loc1
set got=%errorlevel%
set want=0
call :check
set cid=loc2
set got=%scoped%
set want=outer
call :check
exit /b 0

:fe10probe
setlocal
set scoped=inner
if not "%scoped%"=="inner" goto fe10bad
endlocal
exit /b 0
:fe10bad
endlocal
exit /b 1

rem ----- call, arguments and %~ modifiers on parameters: rt_stripq, bat_shift -----
:fe11
call :fe11helper alpha beta
set cid=arg1
set got=%harg1%
set want=alpha
call :check
set cid=arg2
set got=%harg2%
set want=beta
call :check
call :fe11quoted "spaced value"
set cid=arg3
set got=%hq%
set want=spaced value
call :check
call :fe11shift a b c
set cid=arg4
set got=%hshift%
set want=b
call :check
call :fe11count
set cid=arg5
set got=%hempty%
set want=[]
call :check
exit /b 0

:fe11helper
set harg1=%1
set harg2=%2
exit /b 0

:fe11quoted
set hq=%~1
exit /b 0

:fe11shift
shift
set hshift=%1
exit /b 0

:fe11count
set hempty=[%1]
exit /b 0

rem ----- errorlevel and exit codes -----
:fe12
call :fe12rc 0
set cid=erl1
set got=%errorlevel%
set want=0
call :check
call :fe12rc 3
set cid=erl2
set got=%errorlevel%
set want=3
call :check
rem `call :check` ends in `exit /b 0`, which clears errorlevel, so each of these
rem re-establishes the code it is about to test.
call :fe12rc 3
set cid=erl3
set got=no
if errorlevel 3 set got=yes
set want=yes
call :check
call :fe12rc 3
set cid=erl4
set got=no
if errorlevel 4 set got=bad
if not errorlevel 4 set got=yes
set want=yes
call :check
call :fe12rc 0
exit /b 0

:fe12rc
exit /b %1

rem ----- goto, labels and :eof -----
:fe13
set gpath=none
goto fe13target
set gpath=fellthrough
:fe13target
set cid=goto1
set got=%gpath%
set want=none
call :check
call :fe13eof
set cid=goto2
set got=%geof%
set want=reached
call :check
exit /b 0

:fe13eof
set geof=reached
goto :eof

rem ----- conditional execution and separators -----
:fe14
set sep=untouched
call :fe12rc 0 && set sep=ran
set cid=sep1
set got=%sep%
set want=ran
call :check
set sep=untouched
call :fe12rc 1 && set sep=ran
set cid=sep2
set got=%sep%
set want=untouched
call :check
set sep=untouched
call :fe12rc 1 || set sep=ran
set cid=sep3
set got=%sep%
set want=ran
call :check
set a1=1& set a2=2
set cid=sep4
set got=%a1%%a2%
set want=12
call :check
call :fe12rc 0
exit /b 0

rem ----- redirection to NUL, and nested blocks -----
:fe15
echo swallowed> NUL
set cid=red1
set got=%errorlevel%
set want=0
call :check
set /a bacc=0
if "1"=="1" (
  for /l %%i in (1,1,3) do (
    set /a bacc=bacc+1
  )
)
set cid=blk1
set got=%bacc%
set want=3
call :check
exit /b 0

rem ----- comments and expansion corners -----
:fe16
rem this is a comment and must not run
:: and so is this
set cid=cmt1
set got=fine
set want=fine
call :check
set withspace=a b
set cid=exp1
set got=[%withspace%]
set want=[a b]
call :check
set pct=100%%
set cid=exp2
set got=%pct%
set want=100%
call :check
exit /b 0

:main
call :fe01
call :fe02
call :fe03
call :fe04
call :fe05
call :fe06
call :fe07
call :fe08
call :fe09
call :fe10
call :fe11
call :fe12
call :fe13
call :fe14
call :fe15
call :fe16
echo features: %checks% checks, %fails% failures
exit /b %fails%
