#!/usr/bin/env bash
# A bash program on which all four probe legs genuinely agree, including the
# real toolchain (bash 5.3 on this machine). It is the POSITIVE half of
# tests/probe/selftest.sh: a gate that only ever goes red proves nothing, so the
# sabotage cases need a case that must stay green beside them.
#
# Every operand is read out of an ARRAY, per the manual: a probe over literals
# tests the grammar's constant folder, not the runtime.
#
# Deliberately absent, because our bash and real bash disagree on them and this
# file's job is to be the control, not the finding:
#   printf %x / %o        our compiler half prints decimal; our interpreter half
#                         gets %x right and %o wrong, so this is both a halves
#                         divergence and a both-halves-wrong.
#   array elements holding leading/trailing spaces or a tab
#   the exit status of a trailing while loop
A=(0 1 2 3 7 12 25 100 -1 -7 -100 999 1000 65535)
B=(1 2 3 5 7 11 13 4 6 8 9 10 12 14)
S=(abc x "hello world" "a.b.c" ABC aXbXc 123 z mississippi end)

for i in "${!A[@]}"; do
  a=${A[$i]}; b=${B[$i]}
  echo "arith $a $b : $((a+b)) $((a-b)) $((a*b)) $((a/b)) $((a%b)) $((a<<2)) $((a>>1))"
  echo "cmp   $a $b : $(( a<b )) $(( a<=b )) $(( a==b )) $(( a>b )) $(( a!=b ))"
  echo "logic $a $b : $((a&b)) $((a|b)) $((a^b)) $((a && b)) $((a || b))"
done

for i in "${!S[@]}"; do
  s=${S[$i]}
  echo "str   [$s] len=${#s} up=${s^^} lo=${s,,}"
  echo "sub   [$s] 0:3=[${s:0:3}] 2=[${s:2}] repl=[${s//a/A}] pre=[${s#a}] suf=[${s%c}]"
  case "$s" in
    a*) echo "case  [$s] starts-a" ;;
    *c) echo "case  [$s] ends-c" ;;
    *)  echo "case  [$s] other" ;;
  esac
done

f() { local acc=0; local k; for k in "$@"; do acc=$((acc + k)); done; echo "$acc"; }
echo "fn    $(f "${A[@]}") $(f "${B[@]}")"

j=""
for i in "${!B[@]}"; do j="$j:${B[$i]}"; done
echo "join  $j"

t=0; for ((i = 0; i < 20; i++)); do t=$((t + i * i)); done
echo "loop  $t"

i=0
while [ $i -lt 5 ]; do echo "while $i $((i * i))"; i=$((i + 1)); done
exit 0
