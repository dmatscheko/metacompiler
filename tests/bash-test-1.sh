#!/usr/bin/env bash
# Self-checking test for languages/bash-interpreter.abnf.
#
# It exercises every feature of the implemented Bash subset, comparing each computed
# result against its expected value with the check() helper, which prints PASS/FAIL and
# counts failures in the global $fails. The script ends with `exit $fails`, so a fully
# green run exits 0. The output is deterministic and must be byte-identical under goja
# and under -frozen. It stays inside the subset (no pipes, redirection, $(cmd), arrays,
# case or external commands), so real bash produces the same output too.

fails=0
pass=0

# check LABEL GOT WANT : one assertion.
check() {
  if [ "$2" = "$3" ]; then
    echo "PASS: $1"
    pass=$(( pass + 1 ))
  else
    echo "FAIL: $1 -- got [$2] want [$3]"
    fails=$(( fails + 1 ))
  fi
}

echo "=== variables and expansion ==="
a=5
check "assign and \$name" "$a" "5"
check "brace \${name} adjacency" "${a}0" "50"
greeting="value is $a"
check "double-quote interpolation" "$greeting" "value is 5"
literal='no $a or ${a} expansion'
check "single-quote is literal" "$literal" 'no $a or ${a} expansion'
empty=
check "empty assignment" "$empty" ""
b=$a
check "assign from variable" "$b" "5"
path=/usr/local/bin
check "bareword with slashes" "$path" "/usr/local/bin"

echo "=== escapes ==="
check "escaped dollar in dquotes" "price is \$5" "price is \$5"
check "escaped backslash" "back\\slash" "back\\slash"
check "escaped space in bareword" a\ b\ c "a b c"

echo "=== arithmetic \$(( )) ==="
check "add" "$(( 2 + 3 ))" "5"
check "subtract" "$(( 10 - 4 ))" "6"
check "multiply" "$(( 6 * 7 ))" "42"
check "integer divide" "$(( 17 / 5 ))" "3"
check "modulo" "$(( 17 % 5 ))" "2"
check "unary minus" "$(( -5 + 2 ))" "-3"
check "parentheses" "$(( (2 + 3) * 4 ))" "20"
check "nested parentheses" "$(( ((1 + 2) * (3 + 4)) % 6 ))" "3"
check "precedence" "$(( 2 + 3 * 4 - 1 ))" "13"
check "negative modulo" "$(( -7 % 3 ))" "-1"
x=7
y=3
check "arithmetic with bare vars" "$(( x * y - 1 ))" "20"
check "arithmetic with \$vars" "$(( $x + $y ))" "10"

echo "=== arithmetic comparisons (1/0) ==="
check "less-than true" "$(( 3 < 5 ))" "1"
check "less-than false" "$(( 5 < 3 ))" "0"
check "greater-than" "$(( 5 > 3 ))" "1"
check "less-equal (equal)" "$(( 5 <= 5 ))" "1"
check "greater-equal false" "$(( 4 >= 9 ))" "0"
check "equal-equal" "$(( 7 == 7 ))" "1"
check "not-equal" "$(( 7 != 8 ))" "1"

echo "=== if / elif / else ==="
r=none
if [ 1 -eq 1 ]; then r=then; fi
check "if then taken" "$r" "then"
r=none
if [ 1 -eq 2 ]; then r=then; else r=else; fi
check "if else taken" "$r" "else"
grade=75
if [ $grade -ge 90 ]; then
  letter=A
elif [ $grade -ge 80 ]; then
  letter=B
elif [ $grade -ge 70 ]; then
  letter=C
else
  letter=F
fi
check "elif chain" "$letter" "C"

echo "=== test builtin: integer operators ==="
if [ 5 -eq 5 ]; then t=1; else t=0; fi
check "-eq" "$t" "1"
if [ 5 -ne 6 ]; then t=1; else t=0; fi
check "-ne" "$t" "1"
if [ 3 -lt 4 ]; then t=1; else t=0; fi
check "-lt" "$t" "1"
if [ 4 -le 4 ]; then t=1; else t=0; fi
check "-le" "$t" "1"
if [ 9 -gt 2 ]; then t=1; else t=0; fi
check "-gt" "$t" "1"
if [ 9 -ge 9 ]; then t=1; else t=0; fi
check "-ge" "$t" "1"

echo "=== test builtin: string operators ==="
if [ "abc" = "abc" ]; then t=1; else t=0; fi
check "string =" "$t" "1"
if [ "abc" != "xyz" ]; then t=1; else t=0; fi
check "string !=" "$t" "1"
if [ -z "" ]; then t=1; else t=0; fi
check "-z on empty" "$t" "1"
if [ -n "x" ]; then t=1; else t=0; fi
check "-n on nonempty" "$t" "1"
if [ -z "x" ]; then t=1; else t=0; fi
check "-z on nonempty is false" "$t" "0"

echo "=== && and || ==="
flag=untouched
true && flag=ran
check "&& runs on success" "$flag" "ran"
flag=untouched
false && flag=ran
check "&& skips on failure" "$flag" "untouched"
flag=untouched
false || flag=ran
check "|| runs on failure" "$flag" "ran"
flag=untouched
true || flag=ran
check "|| skips on success" "$flag" "untouched"

echo "=== builtins true / false / : and \$? ==="
true
check "true sets \$? to 0" "$?" "0"
false
check "false sets \$? to 1" "$?" "1"
:
check "colon sets \$? to 0" "$?" "0"

echo "=== for loops ==="
acc=""
for w in x y z; do
  acc="$acc$w"
done
check "for over word list" "$acc" "xyz"
total=0
for k in 1 2 3 4 5; do
  total=$(( total + k ))
done
check "for summing numbers" "$total" "15"
count=0
for item in one; do
  count=$(( count + 1 ))
done
check "for single element" "$count" "1"

echo "=== while loops ==="
sum=0
i=1
while [ $i -le 5 ]; do
  sum=$(( sum + i ))
  i=$(( i + 1 ))
done
check "while sum 1..5" "$sum" "15"
countdown=""
n=3
while [ $n -gt 0 ]; do
  countdown="$countdown$n"
  n=$(( n - 1 ))
done
check "while countdown" "$countdown" "321"

echo "=== functions, positional params, \$# ==="
add() {
  gresult=$(( $1 + $2 ))
}
add 12 30
check "function args \$1 \$2" "$gresult" "42"
nargs() {
  gcount=$#
}
nargs a b c d
check "function \$# arg count" "$gcount" "4"
nargs
check "function \$# with no args" "$gcount" "0"
concat3() {
  gcat="$1-$2-$3"
}
concat3 red green blue
check "function string args" "$gcat" "red-green-blue"

echo "=== return and \$? from functions ==="
ispositive() {
  if [ $1 -gt 0 ]; then
    return 0
  fi
  return 1
}
ispositive 5
check "return 0 for positive" "$?" "0"
ispositive -3
check "return 1 for negative" "$?" "1"

echo "=== recursion (positional-param frames) ==="
fresult=0
factorial() {
  if [ $1 -le 1 ]; then
    fresult=1
  else
    factorial $(( $1 - 1 ))
    fresult=$(( $1 * fresult ))
  fi
}
factorial 5
check "factorial 5" "$fresult" "120"
factorial 6
check "factorial 6" "$fresult" "720"

# Iterative Fibonacci (the subset has no `local`, so a two-branch recursive fib cannot
# hold an intermediate across both calls -- real bash has the same limitation).
fa=0
fb=1
fi=0
while [ $fi -lt 10 ]; do
  fnext=$(( fa + fb ))
  fa=$fb
  fb=$fnext
  fi=$(( fi + 1 ))
done
check "iterative fibonacci 10" "$fa" "55"

echo "=== combined: FizzBuzz-style loop ==="
out=""
c=1
while [ $c -le 15 ]; do
  if [ $(( c % 15 )) -eq 0 ]; then
    out="${out}FB "
  elif [ $(( c % 3 )) -eq 0 ]; then
    out="${out}F "
  elif [ $(( c % 5 )) -eq 0 ]; then
    out="${out}B "
  else
    out="${out}${c} "
  fi
  c=$(( c + 1 ))
done
check "fizzbuzz 1..15" "$out" "1 2 F 4 B F 7 8 F B 11 F 13 14 FB "

echo "=== echo formatting (deterministic output) ==="
echo one   two     three
echo -n "joined"
echo -n " with"
echo " no gaps"
echo "trailing newline follows"

echo "=== summary ==="
echo "passed: $pass"
echo "failed: $fails"
if [ $fails -eq 0 ]; then
  echo "ALL TESTS PASSED"
else
  echo "SOME TESTS FAILED"
fi
exit $fails
