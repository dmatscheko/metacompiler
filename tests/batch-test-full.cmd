@echo off
rem Full-syntax test: Windows Batch (cmd.exe command language).
rem
rem This file belongs to the SECOND test group (./test.sh --full): it is NOT part
rem of the default matrix. The goal of the metacompiler is to support the full
rem languages; this file is the ratchet that measures how far the batch grammars
rem are. It walks the practical cmd.exe batch syntax, one self-contained SECTION
rem per language area. The --full runner runs the file, and whenever a grammar
rem aborts it removes the section around the error and retries - so the report
rem lists every unsupported section, not just the first.
rem
rem Conventions (shared by every *-test-full.* file):
rem   - prologue (before the first SECTION marker): the check helper only
rem   - each section: 'rem ===== SECTION <nn>: <name> =====' followed by one
rem     :sNN subroutine, self-contained, ending in 'exit /b 0'
rem   - the tail (:main, after END SECTIONS) calls each section via a line
rem     tagged 'SECTION-CALL <nn>' - written as trailing words on the call,
rem     which cmd passes as arguments the subroutine never reads - and prints
rem     the summary line
rem     'full: <checks> checks, <failures> failures'
rem   - the file ends with 'exit /b %fails%' (exit 0 == full support)
rem
rem GROUND TRUTH. There is no cmd.exe on the development machine, so unlike the
rem bash ratchet NONE of these assertions could be executed. Every one of them is
rem derived from the documented behaviour of cmd.exe (the 'cmd /?', 'set /?',
rem 'for /?', 'if /?' and 'call /?' help texts, which are the normative
rem description of the language) and from the behaviour the wine-cmd-tests corpus
rem under tests/reference/batch encodes. Where a form is genuinely ambiguous in
rem the documentation it was left OUT rather than guessed at; the report that
rem accompanies this file lists the areas that could not be settled.
rem
rem Deliberately out of scope (not syntax, or unrunnable in this harness):
rem external programs of every kind (findstr, more, xcopy, ...), so no pipeline
rem carries real data; creating a file, so nothing is asserted about reading one
rem back and 'if exist' is only asserted for paths every Windows installation
rem has; enumerating a directory, which rules out 'for /r' and 'for /d'; the
rem %~a / %~t / %~z modifiers, which stat a file (the pure string modifiers
rem %~f %~d %~p %~n %~x %~nx ARE covered); interactive input, so 'set /p' is
rem only exercised for the no-input case and 'pause' / 'choice' not at all;
rem date/time/%RANDOM%; codepages and unicode; and the registry.
rem
rem Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
rem code), organized after the cmd.exe command reference.

set /a fails=0
set /a checks=0
goto main

rem check: compares %got% against %want% and reports %cid%. It communicates
rem through variables rather than through %1..%3 so that the prologue itself
rem needs nothing beyond 'set', 'if' and 'call' - argument passing is a language
rem feature of its own, and section 12 is where it is measured.
:check
set /a checks=checks+1
if not "%got%"=="%want%" echo FAIL %cid%
if not "%got%"=="%want%" set /a fails=fails+1
exit /b 0

rem ===== SECTION 01: baseline =====
rem Condensed re-assertion of the basics this file builds on.
:s01
set x=5
set cid=bas1
set got=%x%
set want=5
call :check
set /a y=x * 2 + 1
set cid=bas2
set got=%y%
set want=11
call :check
set cid=bas3
set got=no
if %y% GEQ 11 set got=yes
set want=yes
call :check
set cid=bas4
set got=no
if defined x set got=yes
set want=yes
call :check
set /a sum=0
for %%i in (1 2 3 4) do (
    set cur=%%i
    call :s01add
)
set cid=bas5
set got=%sum%
set want=10
call :check
goto s01done
echo SHOULD-NOT-PRINT
:s01done
set cid=bas6
set got=jumped
set want=jumped
call :check
exit /b 0
:s01add
set /a sum=sum+cur
exit /b 0

rem ===== SECTION 02: echo forms and quoting =====
:s02
rem 'echo.' and 'echo(' both print an empty line; only their acceptance is
rem measured here, since the output itself is not captured.
echo.
echo(
set cid=ech1
set got=ok
set want=ok
call :check
rem A caret escapes the next character outside quotes.
set caret=a^&b
set cid=ech2
set got=%caret%
set want=a^&b
call :check
rem Inside double quotes the caret is literal.
set quoted="a^b"
set cid=ech3
set got=%quoted%
set want="a^b"
call :check
rem set "name=value" is the form that does not capture a trailing space.
set "trimmed=value"
set cid=ech4
set got=%trimmed%
set want=value
call :check
rem A percent sign is written %% in a batch file.
set pct=100%%
set cid=ech5
set got=%pct%
set want=100%
call :check
exit /b 0

rem ===== SECTION 03: set, set /a and set /p =====
:s03
set plain=hello
set cid=set1
set got=%plain%
set want=hello
call :check
rem Assigning nothing removes the variable.
set plain=
set cid=set2
set got=no
if not defined plain set got=yes
set want=yes
call :check
rem An undefined variable expands to nothing at top level.
set cid=set3
set got=[%plain%]
set want=[]
call :check
rem set /a assigns and evaluates.
set /a n=7
set cid=set4
set got=%n%
set want=7
call :check
rem set /p with redirected empty input leaves the variable unchanged.
set keep=before
set /p keep=prompt: < nul
set cid=set5
set got=%keep%
set want=before
call :check
exit /b 0

rem ===== SECTION 04: set /a operators =====
:s04
set /a a1=17 / 5
set cid=ari1
set got=%a1%
set want=3
call :check
set /a "a2=17 %% 5"
set cid=ari2
set got=%a2%
set want=2
call :check
set /a a3=-17 / 5
set cid=ari3
set got=%a3%
set want=-3
call :check
set /a a4=(2 + 3) * 4
set cid=ari4
set got=%a4%
set want=20
call :check
set /a "a5=1 << 4"
set cid=ari5
set got=%a5%
set want=16
call :check
set /a "a6=255 >> 4"
set cid=ari6
set got=%a6%
set want=15
call :check
set /a "a7=12 & 10"
set cid=ari7
set got=%a7%
set want=8
call :check
set /a "a8=12 | 3"
set cid=ari8
set got=%a8%
set want=15
call :check
set /a "a9=12 ^ 10"
set cid=ari9
set got=%a9%
set want=6
call :check
set /a a10=~0
set cid=ari10
set got=%a10%
set want=-1
call :check
set /a a11=0x10
set cid=ari11
set got=%a11%
set want=16
call :check
set /a a12=010
set cid=ari12
set got=%a12%
set want=8
call :check
set /a a13=5
set /a a13+=3
set cid=ari13
set got=%a13%
set want=8
call :check
set /a a14=6
set /a a14*=7
set cid=ari14
set got=%a14%
set want=42
call :check
rem A comma sequences several assignments; the variable keeps the last value.
set /a a15=1,a15=2,a15=3
set cid=ari15
set got=%a15%
set want=3
call :check
rem set /a reads a bare name as a variable.
set /a base=10
set /a derived=base * 3
set cid=ari16
set got=%derived%
set want=30
call :check
exit /b 0

rem ===== SECTION 05: string substitution and substrings =====
:s05
set pth=C:\dir\sub\file.txt
set cid=str1
set got=%pth:~0,2%
set want=C:
call :check
set cid=str2
set got=%pth:~3,3%
set want=dir
call :check
set cid=str3
set got=%pth:~-3%
set want=txt
call :check
set cid=str4
set got=%pth:~-8,4%
set want=file
call :check
set cid=str5
set got=%pth:~3%
set want=dir\sub\file.txt
call :check
set word=banana
set cid=str6
set got=%word:na=NA%
set want=baNANA
call :check
rem The match is case-insensitive.
set mixed=Hello World
set cid=str7
set got=%mixed:WORLD=there%
set want=Hello there
call :check
rem Replacing with nothing deletes.
set cid=str8
set got=%word:na=%
set want=ba
call :check
rem A leading * replaces everything up to and including the match.
set cid=str9
set got=%pth:*sub\=%
set want=file.txt
call :check
exit /b 0

rem ===== SECTION 06: delayed expansion =====
:s06
rem Everything delayed-expansion happens inside the setlocal; the result leaves
rem it through 'endlocal & set NAME=%inner%', whose right side is expanded while
rem the local environment is still in force. The check counters therefore stay
rem outside, where endlocal cannot discard them.
setlocal enabledelayedexpansion
set counter=0
for %%i in (1 2 3) do set /a counter=!counter! + 1
rem Inside a block %var% is the value from BEFORE the block, !var! is current.
set v=old
set snapshot=
set live=
for %%i in (1) do (
    set v=new
    set snapshot=%v%
    set live=!v!
)
rem Delayed expansion also drives the substring operator.
set text=abcdef
set out=!counter!/!snapshot!/!live!/!text:~2,3!
endlocal & set dlyout=%out%
set cid=dly1
set got=%dlyout%
set want=3/old/new/cde
call :check
exit /b 0

rem ===== SECTION 07: setlocal and endlocal =====
:s07
set outerv=outside
setlocal
set outerv=inside
set innerv=%outerv%
endlocal & set locseen=%innerv%
set cid=loc1
set got=%locseen%
set want=inside
call :check
set cid=loc2
set got=%outerv%
set want=outside
call :check
rem A variable created inside setlocal does not survive endlocal.
setlocal
set createdv=temp
endlocal
set cid=loc3
set got=no
if not defined createdv set got=yes
set want=yes
call :check
exit /b 0

rem ===== SECTION 08: if forms =====
:s08
set v=5
set cid=if1
set got=no
if "%v%"=="5" set got=yes
set want=yes
call :check
set cid=if2
set got=no
if %v% EQU 5 set got=yes
set want=yes
call :check
set cid=if3
set got=no
if %v% NEQ 6 set got=yes
set want=yes
call :check
set cid=if4
set got=no
if %v% LSS 10 if %v% GTR 1 set got=yes
set want=yes
call :check
set cid=if5
if %v% EQU 99 (set got=then) else (set got=else)
set want=else
call :check
set cid=if6
set got=no
if not %v% EQU 99 set got=yes
set want=yes
call :check
rem /i makes a string comparison case-insensitive.
set name=Alpha
set cid=if7
set got=no
if /i "%name%"=="ALPHA" set got=yes
set want=yes
call :check
set cid=if8
set got=no
if "%name%"=="ALPHA" (set got=bad) else (set got=yes)
set want=yes
call :check
rem A multi-line parenthesised body.
set cid=if9
set got=no
if %v% EQU 5 (
    set got=yes
)
set want=yes
call :check
rem if exist against a device name that always exists.
set cid=if10
set got=no
if exist nul set got=yes
set want=yes
call :check
exit /b 0

rem ===== SECTION 09: for /l =====
:s09
set /a total=0
for /l %%i in (1,1,5) do (
    set cur=%%i
    call :s09add
)
set cid=forl1
set got=%total%
set want=15
call :check
set /a total=0
for /l %%i in (10,-2,4) do (
    set cur=%%i
    call :s09add
)
set cid=forl2
set got=%total%
set want=28
call :check
rem A step that never reaches the end runs zero times.
set /a total=0
for /l %%i in (5,1,1) do (
    set cur=%%i
    call :s09add
)
set cid=forl3
set got=%total%
set want=0
call :check
exit /b 0
:s09add
set /a total=total+cur
exit /b 0

rem ===== SECTION 10: for /f over strings =====
:s10
rem Quoted text is parsed directly; the default delimiters are space and tab.
for /f %%a in ("one two three") do set first=%%a
set cid=forf1
set got=%first%
set want=one
call :check
rem tokens= picks fields, and the letters run on from the loop variable.
for /f "tokens=1,3" %%a in ("alpha beta gamma") do (
    set t1=%%a
    set t3=%%b
)
set cid=forf2
set got=%t1%
set want=alpha
call :check
set cid=forf3
set got=%t3%
set want=gamma
call :check
rem delims= changes the separator set.
for /f "tokens=2 delims=," %%a in ("a,b,c") do set mid=%%a
set cid=forf4
set got=%mid%
set want=b
call :check
rem tokens=* takes the whole remainder.
for /f "tokens=*" %%a in ("  leading spaces here") do set whole=%%a
set cid=forf5
set got=%whole%
set want=leading spaces here
call :check
rem tokens=2* takes one field and then the rest.
for /f "tokens=1*" %%a in ("head the rest of it") do (
    set h=%%a
    set r=%%b
)
set cid=forf6
set got=%h%
set want=head
call :check
set cid=forf7
set got=%r%
set want=the rest of it
call :check
rem eol= sets the comment character, so a matching line is skipped entirely.
set skipped=untouched
for /f "eol=;" %%a in (";commented") do set skipped=%%a
set cid=forf8
set got=%skipped%
set want=untouched
call :check
rem A single-quoted set is a COMMAND whose output is parsed.
for /f %%a in ('echo piped') do set fromcmd=%%a
set cid=forf9
set got=%fromcmd%
set want=piped
call :check
exit /b 0

rem ===== SECTION 11: plain for and its variable modifiers =====
:s11
set acc=
for %%i in (a b c) do call :s11cat %%i
set cid=forp1
set got=%acc%
set want=abc
call :check
rem The ~ modifiers are pure string operations on the loop variable.
for %%i in ("C:\dir\sub\file.txt") do (
    set fname=%%~ni
    set fext=%%~xi
    set fdrv=%%~di
    set fpath=%%~pi
    set fnx=%%~nxi
    set fnoq=%%~i
)
set cid=forp2
set got=%fname%
set want=file
call :check
set cid=forp3
set got=%fext%
set want=.txt
call :check
set cid=forp4
set got=%fdrv%
set want=C:
call :check
set cid=forp5
set got=%fpath%
set want=\dir\sub\
call :check
set cid=forp6
set got=%fnx%
set want=file.txt
call :check
rem %%~i on its own strips surrounding quotes.
set cid=forp7
set got=%fnoq%
set want=C:\dir\sub\file.txt
call :check
exit /b 0
:s11cat
set acc=%acc%%1
exit /b 0

rem ===== SECTION 12: call, arguments and parameter modifiers =====
:s12
call :s12sum 3 4
set cid=cal1
set got=%res%
set want=7
call :check
call :s12first alpha beta
set cid=cal2
set got=%res%
set want=alpha
call :check
rem %~1 strips surrounding double quotes.
call :s12first "quoted value"
set cid=cal3
set got=%res%
set want=quoted value
call :check
rem %* is the whole argument tail.
call :s12all one two three
set cid=cal4
set got=%res%
set want=one two three
call :check
rem shift moves the arguments down.
call :s12shifted a b c
set cid=cal5
set got=%res%
set want=b
call :check
rem %~n1 and friends work on arguments too.
call :s12name "C:\dir\report.log"
set cid=cal6
set got=%res%
set want=report
call :check
rem An argument that was never passed expands to nothing.
call :s12ninth
set cid=cal7
set got=[%res%]
set want=[]
call :check
exit /b 0
:s12sum
set /a res=%1 + %2
exit /b 0
:s12first
set res=%~1
exit /b 0
:s12all
set res=%*
exit /b 0
:s12shifted
shift
set res=%~1
exit /b 0
:s12name
set res=%~n1
exit /b 0
:s12ninth
set res=%9
exit /b 0

rem ===== SECTION 13: goto, labels and :eof =====
:s13
set trail=
goto :s13a
set trail=%trail%X
:s13a
set trail=%trail%A
goto s13b
set trail=%trail%Y
:s13b
set trail=%trail%B
set cid=goto1
set got=%trail%
set want=AB
call :check
rem goto :eof returns from the current call.
call :s13sub
set cid=goto2
set got=%res%
set want=early
call :check
rem A label may be written with leading whitespace and a trailing comment.
goto s13c
   :s13c
set cid=goto3
set got=reached
set want=reached
call :check
exit /b 0
:s13sub
set res=early
goto :eof
set res=late
exit /b 0

rem ===== SECTION 14: errorlevel =====
:s14
call :s14code 0
set cid=err1
set got=%errorlevel%
set want=0
call :check
call :s14code 3
set cid=err2
set got=%errorlevel%
set want=3
call :check
rem 'if errorlevel n' is true when the code is n OR GREATER.
call :s14code 3
set cid=err3
set got=no
if errorlevel 3 set got=yes
set want=yes
call :check
call :s14code 3
set cid=err4
set got=no
if errorlevel 4 set got=bad
if not errorlevel 4 set got=yes
set want=yes
call :check
rem A successful internal command clears it.
call :s14code 5
ver > nul
set cid=err5
set got=%errorlevel%
set want=0
call :check
exit /b 0
:s14code
exit /b %1

rem ===== SECTION 15: command separators and conditional execution =====
:s15
rem & runs both commands regardless of status.
set a=
set b=
set a=1& set b=2
set cid=sep1
set got=%a%%b%
set want=12
call :check
rem && runs the second only after success.
set flag=untouched
call :s15ok && set flag=ran
set cid=sep2
set got=%flag%
set want=ran
call :check
set flag=untouched
call :s15bad && set flag=ran
set cid=sep3
set got=%flag%
set want=untouched
call :check
rem || runs the second only after failure.
set flag=untouched
call :s15bad || set flag=ran
set cid=sep4
set got=%flag%
set want=ran
call :check
set flag=untouched
call :s15ok || set flag=ran
set cid=sep5
set got=%flag%
set want=untouched
call :check
exit /b 0
:s15ok
exit /b 0
:s15bad
exit /b 1

rem ===== SECTION 16: redirection =====
:s16
rem Output to nul is discarded and the command still succeeds.
echo discarded > nul
set cid=red1
set got=%errorlevel%
set want=0
call :check
rem Appending is >> and also succeeds against nul.
echo appended >> nul
set cid=red2
set got=%errorlevel%
set want=0
call :check
rem 2> redirects only the error stream.
echo kept 2> nul
set cid=red3
set got=%errorlevel%
set want=0
call :check
rem 2>&1 merges the error stream into the output stream.
echo merged > nul 2>&1
set cid=red4
set got=%errorlevel%
set want=0
call :check
rem < redirects input; reading nothing leaves set /p alone (see section 03).
set indata=kept
set /p indata= < nul
set cid=red5
set got=%indata%
set want=kept
call :check
rem A redirection may be attached to a parenthesised block.
(
    echo one
    echo two
) > nul
set cid=red6
set got=%errorlevel%
set want=0
call :check
exit /b 0

rem ===== SECTION 17: nested blocks and multi-line structures =====
:s17
set /a grid=0
for /l %%i in (1,1,3) do (
    for /l %%j in (1,1,2) do (
        set /a grid=grid+1
    )
)
set cid=nst1
set got=%grid%
set want=6
call :check
set outcome=
if 1 EQU 1 (
    if 2 EQU 2 (
        set outcome=both
    ) else (
        set outcome=inner-else
    )
) else (
    set outcome=outer-else
)
set cid=nst2
set got=%outcome%
set want=both
call :check
rem A for body may hold several statements separated by newlines.
set /a acc=0
for %%i in (5 10) do (
    set cur=%%i
    call :s17add
    call :s17add
)
set cid=nst3
set got=%acc%
set want=30
call :check
exit /b 0
:s17add
set /a acc=acc+cur
exit /b 0

rem ===== SECTION 18: rem, :: comments and line continuation =====
:s18
rem A rem line is ignored.
rem set poisoned=yes
set cid=cmt1
set got=no
if not defined poisoned set got=yes
set want=yes
call :check
:: A :: line is a label that acts as a comment.
:: set alsopoisoned=yes
set cid=cmt2
set got=no
if not defined alsopoisoned set got=yes
set want=yes
call :check
rem A caret at the end of a line continues it.
set joined=abc^
def
set cid=cmt3
set got=%joined%
set want=abcdef
call :check
exit /b 0

rem ===== SECTION 19: variable expansion corner cases =====
:s19
rem An expansion inside a quoted string still happens.
set inner=VALUE
set cid=exp1
set got="wrapped %inner%"
set want="wrapped VALUE"
call :check
rem Two expansions run together.
set p1=ab
set p2=cd
set cid=exp2
set got=%p1%%p2%
set want=abcd
call :check
rem An expansion may build a name that another expansion then uses, via call.
set target=found
set indirect=target
set cid=exp3
call set got=%%%indirect%%%
set want=found
call :check
rem Variable names are case-insensitive.
set MixedCase=here
set cid=exp4
set got=%mixedcase%
set want=here
call :check
rem An expansion of an undefined name inside a quoted comparison is empty.
set cid=exp5
set got=no
if "%definitely_not_set%"=="" set got=yes
set want=yes
call :check
exit /b 0

rem ===== SECTION 20: exit codes =====
:s20
rem exit /b returns from the script or subroutine with a code.
call :s20code 0
set cid=xit1
set got=%errorlevel%
set want=0
call :check
call :s20code 7
set cid=xit2
set got=%errorlevel%
set want=7
call :check
rem exit /b with no code leaves the previous errorlevel in place.
call :s20code 4
call :s20bare
set cid=xit3
set got=%errorlevel%
set want=4
call :check
exit /b 0
:s20code
exit /b %1
:s20bare
exit /b

rem ===== SECTION 21: for-variable letters that are also modifier letters =====
rem Every ~ modifier is a letter and so is every for-variable, so %%~ff and %%~pd are
rem ambiguous to a greedy scan. This is the one area of the file whose expected values are
rem not inferred from the help text but taken from a suite that was validated against
rem Windows: tests/reference/batch/wine-cmd-tests has `for %%f in ("C D" E) do echo %%~ff`
rem and `for %%d in ("I J" K) do echo %%~pd`, and its .exp file expects the drive/path of
rem the LOOP VARIABLE. So the last letter of the run is the variable.
:s21
for %%f in ("C:\dir\sub\file.txt") do set r=%%~ff
set cid=mod1
set got=%r%
set want=C:\dir\sub\file.txt
call :check
for %%d in ("C:\dir\sub\file.txt") do set r=%%~pd
set cid=mod2
set got=%r%
set want=\dir\sub\
call :check
for %%n in ("C:\dir\sub\file.txt") do set r=%%~dn
set cid=mod3
set got=%r%
set want=C:
call :check
rem A combination still ends at the variable.
for %%x in ("C:\dir\sub\file.txt") do set r=%%~dpx
set cid=mod4
set got=%r%
set want=C:\dir\sub\
call :check
exit /b 0

rem ===== SECTION 22: parentheses as ordinary text =====
rem Outside a block a ( or ) in a value is just a character; inside one, a ) closes the
rem block, which is why a value that opens a paren has to close it again.
:s22
set p1=(a)
set cid=par1
set got=%p1%
set want=(a)
call :check
set p2=Done (ok)
set cid=par2
set got=%p2%
set want=Done (ok)
call :check
set p3=a(b)c(d)
set cid=par3
set got=%p3%
set want=a(b)c(d)
call :check
rem A balanced pair inside a parenthesised block does not end the block.
if 1 EQU 1 (
    set p4=in(block)
)
set cid=par4
set got=%p4%
set want=in(block)
call :check
exit /b 0

rem ===== SECTION 23: for /f command chains and absent file sets =====
:s23
rem The single-quoted set is a command line, so it can chain with &.
for /f %%a in ('echo one& echo two') do set last=%%a
set cid=ffc1
set got=%last%
set want=two
call :check
rem Each line drives one iteration.
set /a rounds=0
for /f %%a in ('echo alpha& echo beta& echo gamma') do set /a rounds=rounds+1
set cid=ffc2
set got=%rounds%
set want=3
call :check
rem A bare word is a FILE set. This one does not exist, so the body never runs and the
rem variable keeps its value. (cmd.exe also reports the missing file; that goes to the
rem error stream, which this file does not capture.)
set untouched=kept
for /f %%a in (no-such-file-9k3.txt) do set untouched=%%a
set cid=ffc3
set got=%untouched%
set want=kept
call :check
exit /b 0

rem ===== SECTION 24: if exist =====
rem Only paths that every Windows installation has are asserted here - there is no way to
rem create a file from this harness, and a path that happens to exist on the machine that
rem runs it would not be a property of the language.
:s24
set cid=exi1
set got=no
if exist nul set got=yes
set want=yes
call :check
set cid=exi2
set got=no
if exist C:\Windows set got=yes
set want=yes
call :check
rem A trailing separator names the same directory.
set cid=exi3
set got=no
if exist C:\Windows\ set got=yes
set want=yes
call :check
set cid=exi4
set got=no
if not exist C:\no-such-directory-9k3 set got=yes
set want=yes
call :check
exit /b 0

rem ===== SECTION 25: the glued echo separator and the echo on/off boundary =====
rem cmd treats a "." ":" or "/" GLUED to the echo keyword as a separator, not as
rem text: `echo.word` prints "word", `echo.` prints an empty line. With a space
rem in between the punctuation is ordinary text again. And `echo on` / `echo off`
rem only toggle echoing when that is the whole line - `echo off now` prints
rem "off now". There is no cmd.exe and no wine on this machine, so the ground
rem truth for every value below is wine's own expectation file, which the harness
rem in tests/reference/batch/wine-cmd-tests/tests/batch.c compares cmd against:
rem   test_builtins.cmd.exp:  echo.word -> word     echo .word -> .word
rem                           echo:word -> word     echo :word -> :word
rem                           echo/word -> word     echo /word -> /word
rem                           echo off now -> off now
:s25
for /f "delims=" %%v in ('echo.word') do set got=%%v
set cid=eds1
set want=word
call :check
for /f "delims=" %%v in ('echo .word') do set got=%%v
set cid=eds2
set want=.word
call :check
for /f "delims=" %%v in ('echo:word') do set got=%%v
set cid=eds3
set want=word
call :check
for /f "delims=" %%v in ('echo :word') do set got=%%v
set cid=eds4
set want=:word
call :check
for /f "delims=" %%v in ('echo/word') do set got=%%v
set cid=eds5
set want=word
call :check
for /f "delims=" %%v in ('echo /word') do set got=%%v
set cid=eds6
set want=/word
call :check
rem `echo on` / `echo off` are only the toggle when nothing follows them.
for /f "delims=" %%v in ('echo off now') do set got=%%v
set cid=eds7
set want=off now
call :check
for /f "delims=" %%v in ('echo on x') do set got=%%v
set cid=eds8
set want=on x
call :check
rem The glued forms with nothing after the separator print an empty line, which
rem makes the for /f body run zero times - so `got` keeps its previous value.
set got=untouched
for /f "delims=" %%v in ('echo.') do set got=%%v
set cid=eds9
set want=untouched
call :check
set got=untouched
for /f "delims=" %%v in ('echo:') do set got=%%v
set cid=eds10
set want=untouched
call :check
rem And the statement form prints the same text to stdout, which the goja and
rem -frozen runs must agree on byte for byte.
echo.here
echo.
echo:here
echo/here
goto :eof

rem ===== SECTION 26: a captured command larger than the capture buffer =====
rem languages/lib/batch-rt.c's rt_capstart bumps an 8192-byte block and rt_println
rem used to copy into it with NO length check of any kind, so a captured command
rem producing more than that walked out of the buffer and over the rest of the
rem arena. It got the right answer whenever nothing else had been bumped since,
rem which is exactly why no test caught it - the same defect, and now the same
rem fix, as bash-rt.c's rt_putc: the buffer GROWS by doubling, because truncating
rem would be a regression on the cases that used to work by luck.
rem
rem MEASURED discriminating power: 16 lines of 592 bytes is 9,488 bytes, 1.16
rem buffers. Against a clean `git archive` build of the parent this section
rem reports capb1 = 15 rather than 16 - the sixteenth line lands on top of the
rem loop's own state - and capb2 is wrong with it. Both pass here.
:s26
set s26u=abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789
set s26h=%s26u%%s26u%%s26u%%s26u%%s26u%%s26u%%s26u%%s26u%
set /a s26n=0
set s26last=none
for /f "delims=" %%v in ('echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%& echo %s26h%') do (
  set /a s26n=s26n+1
  set s26last=%%v
)
set cid=capb1
set got=%s26n%
set want=16
call :check
set cid=capb2
set got=%s26last%
set want=%s26h%
call :check
rem A SHORTER capture after the grown one: the stack must hand out a fresh
rem 8192-byte block rather than still pointing at the grown one.
for /f "delims=" %%v in ('echo tiny') do set s26t=%%v
set cid=capb3
set got=%s26t%
set want=tiny
call :check
exit /b 0

rem ===== END SECTIONS =====

:main
call :s01 SECTION-CALL 01
call :s02 SECTION-CALL 02
call :s03 SECTION-CALL 03
call :s04 SECTION-CALL 04
call :s05 SECTION-CALL 05
call :s06 SECTION-CALL 06
call :s07 SECTION-CALL 07
call :s08 SECTION-CALL 08
call :s09 SECTION-CALL 09
call :s10 SECTION-CALL 10
call :s11 SECTION-CALL 11
call :s12 SECTION-CALL 12
call :s13 SECTION-CALL 13
call :s14 SECTION-CALL 14
call :s15 SECTION-CALL 15
call :s16 SECTION-CALL 16
call :s17 SECTION-CALL 17
call :s18 SECTION-CALL 18
call :s19 SECTION-CALL 19
call :s20 SECTION-CALL 20
call :s21 SECTION-CALL 21
call :s22 SECTION-CALL 22
call :s23 SECTION-CALL 23
call :s24 SECTION-CALL 24
call :s25 SECTION-CALL 25
call :s26 SECTION-CALL 26
echo full: %checks% checks, %fails% failures
exit /b %fails%
