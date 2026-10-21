#!/usr/bin/env bash
# A long run: 160 extended-glob patterns against 11 subjects through all eight pattern
# expansions, twelve times over - some ten megabytes of strings that nothing keeps,
# while n, tot and line are stored on every pass. A compiled loop hands back what each
# pass allocated, so the string heap stays small however long the run. Checked from
# outside: tests/native-full.sh compares the native binary's stdout with
# tests/bash-heap.expected (real bash 5.3's output). Native only, and named outside
# tests/bash-test-* for that reason: the run is far past the IR interpreter's step brake.
shopt -s extglob
subs=('' a ab abc aab xab bbx abab cab xx 'a*b')
pats=()
for op in '@' '?' '*' '+' '!'; do
  for x in a b ab x '?' '*' '[ab]' c; do
    for y in a b ab x; do pats+=("$op($x|$y)"); done
  done
done
n=0
tot=0
for ((r = 0; r < 12; r++)); do
  for p in "${pats[@]}"; do
    for s in "${subs[@]}"; do
      line="[$s] ${s#$p} ${s##$p} ${s%$p} ${s%%$p} ${s/$p/X} ${s//$p/X} ${s/#$p/X} ${s/%$p/X}"
      n=$((n + 1))
      tot=$((tot + ${#line}))
      (( n % 997 == 0 )) && echo "$n $p $line"
    done
  done
done
echo "lines $n chars $tot"
# Arrays far past any fixed table - 6000 indexed and 3000 associative elements, each read
# back by index or key - and a 200-byte value overwritten on every pass, which its slot
# keeps in one buffer rather than one copy per pass.
big=()
declare -A kv
for ((i = 0; i < 6000; i++)); do big+=("e$i"); done
for ((i = 0; i < 3000; i++)); do kv[k$i]=$i; done
s=0
for ((i = 0; i < 6000; i += 7)); do s=$((s + ${#big[i]} + ${kv[k$((i % 3000))]})); done
long=""
for ((i = 0; i < 2000; i++)); do long="${big[*]:i:40}"; done
echo "arrays ${#big[@]} ${#kv[@]} $s ${#long} ${long:0:24}"
# A 5 KB element overwritten on every pass, in an indexed and an associative array, which
# keeps one buffer per element rather than one copy per pass.
el=(x y)
declare -A em=([k]=v)
for ((i = 0; i < 3000; i++)); do
  el[0]="${big[*]:0:900}$i"
  em[k]="${big[*]:0:900}:$i"
done
echo "elements ${#el[0]} ${el[0]: -6} ${#em[k]} ${em[k]: -6}"
