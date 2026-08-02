# What is left: GC, the open defects, and the remaining eleven languages

Companion to [runtime-rework-plan.md](runtime-rework-plan.md), which is the record of
what was BUILT (phases 0-4a, bash/batch convergence, one linker). This document is the
forward plan. HEAD when it was written: `e8bf2c3`.

## Where things stand

```
matrix 325/325 · --full 5,307 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16 - bash, batch, c, lua, metajs all "the clang executable agrees"
```

Five languages produce self-contained native binaries. Two of them (metajs, lua) got
there through the GENERAL architecture - C floor, MetaJS layer 2, language grammar -
which is what makes the remaining eleven a rollout rather than eleven rewrites.

Still on the Go runtime: **29,670 lines across `abnf/jsrt*.go`**, backing eleven
languages.

---

# Part 1 - Memory, which gates everything else

Allocation is down 2.6x (18,072 -> 7,006 bytes per loop iteration) but **still exactly
linear**. A program that runs for a minute cannot be fixed by allocating less. Rolling
out eleven more languages on top of an unbounded-memory runtime multiplies the problem
by eleven, so this comes first.

The residual, per iteration of `s = s + i % 7`, is live data rather than waste:

```
33 scope cells      1,848 B   one per call AND one per block (makeBlockStmt)
20 scope buffers    1,920 B   the scopes that do declare something
26 number cells     1,456 B   intermediates outside the small-integer window
12 array cells +    1,056 B   the argument array of every call
 1 object cell        120 B
```

Two directions. They are not alternatives: the first reduces garbage, the second
reclaims it. **Do them in this order** - every allocation removed by step 1 is one the
collector never has to trace, and step 1 is also where the remaining ~4.8x of TIME is.

## 1a. Stop generating the garbage (emitter, no GC needed)

- **Inline the layer-2 fast paths into the IR.** Three of the five lines above exist
  because a Lua `+` is a MetaJS *call*: it costs a scope, an argument array and its
  intermediates. Phase 4 measured an inlined plain-number fast path as the single
  biggest win of that phase (497s -> 29.5s). Doing it in the emitter removes the call
  and everything under it in one move.
- **`makeBlockStmt` opens a scope per BLOCK**, 33 against 9 actual calls - three
  quarters never declare anything. Fixable in `lib/compile-core.js` alone. The
  difficulty is that the thunk is already emitted by the time you know whether the
  block declares anything; solving that probably means a two-pass emit or a
  patch-after-the-fact.
- **Widen the small-integer cache** and check whether the 26 number cells are mostly
  just outside its window before assuming they are not.

Gate: re-run `tests/bench-alloc.sh` and report bytes/iteration. Target the scope and
array lines specifically; if a change does not move them, it did not work.

## 1b. Mark/sweep over the arena

**Refcounting is the wrong shape here and this is settled, not a preference.** A scope
holds its parent, and a closure holds its defining scope - so a closure defined inside a
function cycles with the scope that names it, and that is the *common* case, not an edge
case. Refcounting would leak exactly the programs people write.

Mark/sweep is cheap here because the hard parts are already true:

- **Roots are enumerable**: `G_ROOT`, `NIC`, `SMC_VAL`, `THROWN`, `RETSLOT`, and layer
  2's scope global.
- **Cells are uniform 56 bytes**, so an arena chunk is a walkable array and the mark bit
  fits in `Cell.tag`.
- **The one real cost is a shadow stack** in the emitter, so that handles live in C
  frames are roots. This is the piece to design first - get it wrong and the collector
  frees live data, which is the worst failure mode available.

Sequence it as: shadow stack first, verified by a mark-only pass that proves it can
reach everything live (mark, then assert no live handle is unmarked); sweep second.

Gate: the benchmarks run in **bounded** memory, and every existing suite stays green
byte-for-byte. A GC that changes an answer is worse than no GC.

---

# Part 2 - The open defects

Ordered by whether they block the rollout.

## Blocking the rollout

~~**Layer 2 cannot create a new primitive TYPE.**~~ **DONE 2026-08-02 - tag 13.**
`languages/lib/runtime.c` now has a real sized-integer tag, the twin of
`abnf/jsrtint.go`'s `jsGInt`, which stays the authoritative specification: `js_typeof`
answers `"number"` for it in **both** halves. It is language neutral by construction -
the whole `js_gi*` extern family (`js_giarith js_gicmp js_gieq js_giconv js_gineg
js_ginot js_ginum js_gistr js_giis js_gival js_gilt js_gile js_gigt js_gige js_giadd
js_gilit`) is implemented, so go/java/kotlin/csharp can be linked natively without
touching the floor again.

```
cell.a  the 64 bit value, already truncated to the width and sign extended
cell.b  the width: 8, 16, 32 or 64        cell.c  1 when unsigned
```

**Ground truth.** A 97,664-line differential probe - `giNorm` over 8 width/signedness
combinations x 32 seeds, then 11 operators x 32 x 32 per combination, plus `giCmp`,
`giEq`, unary minus, `^x`, `giFloat`, and `giConv` between every pair of widths - is
**byte-identical** between the C floor and a Go oracle compiled straight out of
`jsrtint.go`. A second 240-case probe covers the float -> sized-integer conversion,
`+-Inf` and NaN included, and also agrees exactly.

**Two places it cannot match, stated rather than approximated:**

- `giFromFloat` ends in `int64(m)` on a value that is NaN when the operand is an
  infinity, and Go leaves that conversion *implementation defined*. Measured here
  (darwin/arm64) it is `0`; on amd64 the same expression is `-9223372036854775808`.
  The floor implements `0`, so the two halves agree on arm64 and **both would differ
  from a Go twin built for amd64**. Same architecture dependence as the float `%` of
  phase 4a.
- `js_giarith` and `js_giadd` in `jsrtint.go` route a `jsJFlo` (the boxed double of
  `jsrtjvm.go`) to `jvmArith` before reaching `giArith`. `jsJFlo` is a *second*
  primitive type the floor does not have, so those arms are absent. Nothing linked
  natively creates one today, and **the first of java/kotlin/csharp to migrate will
  need `jsJFlo` given exactly the treatment `jsGInt` just got.**

**The alternative was costed and rejected.** "Stop asking `js_typeof` about a number"
is not one change: it is a rule every layer-2 file of all eleven languages has to keep
at every site, forever - `lua-rt.metajs` alone has 22 `typeof` sites. It cannot be
checked mechanically, the Go twin answers `"number"` at every one of them, and a slip
is invisible unless a test happens to print the value. The tag is one place, and it
makes the two halves structurally identical instead of accidentally identical.

**Lua is converted.** `lua-rt.metajs` no longer builds `{__li: {h, l}}`; `luNorm` is
`sint(p.h, p.l, 64, 0)`, which *is* `giNorm`. The ordering dependence is gone,
demonstrated by making the same one-line deletion in both trees - remove the
sized-integer arm from `js_lustr`, so `typeof` decides:

```
                       660c47a                      HEAD
math.maxinteger        9223372036854776000          9223372036854775807
math.mininteger        -9223372036854776000         -9223372036854775808
9007199254740993       9007199254740992             9007199254740993
```

The 13,306-line Lua differential probe is byte-identical between `llvm.Run` and the
native binary, **and** byte-identical to 660c47a's native output - the conversion
changed no answer. Against the installed `lua 5.5` it differs on 2,087 lines, the same
2,087 as 660c47a, all in the number formatter.

**Cost, measured** (native, user time, 660c47a -> HEAD): 2M-iteration `s = s + i % 7`
**4.00 -> 4.16s**, `fib(24)` **0.16 -> 0.16s**, 200k big-integer `(t + big) % p`
**2.30 -> 2.47s**. Two guards had to be written carefully to get there and the
intermediate numbers are recorded at their sites: a magnitude guard on both operands of
`js_lucmp` cost **11.9s** on the first benchmark and a `luPlain()` call **7.6s**,
against 4.16s for the form that ships - `float64` is monotone, so a *strict* double
comparison is already right for a sized integer and only a TIE needs the exact path.

**What a layer-2 file has to do to use the tag** - the whole interface, with
`lua-rt.metajs` as the worked example:

```js
var b = sint(hi, lo, 64, 0)   // build one: two unsigned 32-bit halves, width,
                              // unsignedness. Answers a PLAIN NUMBER when the value
                              // is exact in a double - that is giNorm, and the
                              // invariant is applied here and nowhere else.
sintIs(v)                     // js_giis: is this a sized integer
sintHi(v) / sintLo(v)         // the two halves back out, unsigned
sintWidth(v) / sintUns(v)     // the declared width and signedness
sintOp(code, l, r)            // one binary operator, 0..10 = + - * / % & | ^ &^ << >>
sintCmp(l, r) / sintConv(v, bits, uns) / sintStr(v) / sintNum(v)
```

and then the one rule the tag imposes: **`typeof v == "number"` is TRUE for a sized
integer**, so every site that meant "an ordinary double" has to say so.
`lua-rt.metajs` spells that `luPlain(v)` and uses a cheaper equivalent in the three hot
paths. In exchange three sites that used to be load-bearing became redundant - measured
by deleting each and re-running the ratchet: the sized-integer arms of `js_lustr`,
`js_lukey` and `js_lunum` can all go without one assertion failing, because the floor
now renders, keys and coerces a sized integer correctly on its own. They are kept
because they say what they mean.

**Ratchet**: `tests/lua-test-full.lua` SECTION 24, **67 assertions**, every one also
checked against the installed `lua 5.5`. Its **discriminating power against a clean
archive of 660c47a is 0 of 67**, in all three engines - and that is the finding, not a
gap: no Lua-level program can tell the object box from the tag, which is "Lua survived
by luck of ordering" restated as a measurement. What it does discriminate is the
conversion's own failure modes; mutating one layer-2 site at a time:

```
js_lucmp: non-strict '<=', so a box tie answers wrong        2 fail
js_lucmp: the tie arm treats NaN as equal                    2 fail
js_lucmp: no exact fall-through at all (660c47a semantics)   3 fail
js_lutype: typeof instead of luPlain                         3 fail
luNorm: box every value instead of only outside 2^53        15 fail
js_luisint / js_lustr / js_lukey / js_lunum arms removed      0 fail  (redundant, above)
```

**One latent floor defect found on the way, and fixed.** `d_from_long(INT64_MIN)` built
nonsense - `0 - i` wraps back to `i`, so the top-bit scan ran zero times. Unreachable
until `si_float` passed a raw `int64` in; the probe read `-1` where Go reads
`-9223372036854775808`.

**The one gate item NOT met.** The sized-integer semantics could not be pinned in
`tests/metajs-test-full.js`. That file is run by **both** MetaJS halves, and
`metajs-interpreter.abnf` binds its host globals to **goja natives** (`hostGlobals`),
so a MetaJS *source* program cannot reach a `jsGInt` there at all; a section using
`sint(...)` would be a red section and would cost the whole half's 326 assertions -
precisely the 51-assertion trap recorded under `c-interpreter` below. The fix is a
handful of lines in `metajs-interpreter.abnf`: a goja-`BigInt` implementation of the
six `sint*` globals, added to `hostGlobals`. It was outside the file set for this
change and is the natural first step of the java migration.

~~**`c-interpreter.abnf` has no standard library.**~~ **DONE 2026-08-02.** `libcCall` in
`languages/c-interpreter.abnf` now implements the 18 byte/address names (`strlen strcmp
strncmp strcpy strncpy strcat strncat strchr strrchr strstr memcpy memmove memset memcmp
memchr atoi atol strdup`) over the interpreter's own memory, and the reverted ratchet
section is back as SECTION 50 of `tests/c-test-full.c`. **c: 505 -> 564 assertions**,
both halves FULL and byte-identical goja/`-frozen`, `--full` total 5,307 -> 5,366 with
**0 halves disagreeing**; matrix 325/325, `--cross` 119/0, clang-check still
`c ... ok, and the clang executable agrees`, `go test ./abnf/` ok.

What made it shareable: the interpreter stays **one cell per scalar** (it was NOT made
byte-packed - that is the model, and 93c814f preserved it deliberately), and
`sizeof(char) == 1` in both models, so over CHARACTER data a cell *is* a byte and a
length in bytes *is* a length in cells. SECTION 50 therefore stays on char storage;
`memcpy` over an `int` array would be n/4 cells here and n bytes there, and is left out
rather than asserted wrongly. Only the SIGN of `strcmp`/`strncmp`/`memcmp` is asserted -
C leaves the magnitude unspecified - but the interpreter returns the same **unsigned byte
difference** `libcNative`'s `byteDiff` does, verified: `cc -O0`, `cc -O2`, the
interpreter (goja and `-frozen`), `llvm.Run` and the clang-built `-exe` binary all print
`22 / -22 / 127 / 127 / -127 / 98 / 98` for the same runtime-operand probe.

Two things found on the way, both written down at the site:

- A **bare `return` followed by an expression on the next line** is a goja-vs-frozen
  trap: real JS applies ASI, the frozen MetaJS parser does not, so `if (bad) return`
  newline `mem[a] = v` parsed as `return (mem[a] = v)` and **every store was silently
  lost under `-frozen` while reads were unaffected** - 17 of 59 new assertions red in
  one half only. Guard positively. Noted at `cPut`.
- **`test.sh --full` counted `c` at 564 anyway** while `c-interpreter` was reporting
  `FULL under goja, BUT -frozen fails or differs`. The per-language assertion count and
  the halves-agree verdict are read from the goja run alone, so a frozen-only divergence
  does not reach the summary. Not fixed here (out of the files for this task) but it is a
  real hole in the invariant: the summary is not sufficient, read the per-half lines.

Still deliberately absent from BOTH halves, unchanged and loud rather than wrong: the
varargs `printf` family (neither interpreter has a varargs ABI), `exit`/`abort` (no
unwind path out of a call), `qsort`/`bsearch` (a function pointer is a `funcId`, not a
callable address). The interpreter also still has no `malloc`/`free`; `strdup` uses the
interpreter's own bump `alloc`, and nothing frees it.

## Silently wrong, both halves agreeing (the class byte-identity cannot see)

- **`_Bool b = 5;` holds 5**; `cc` holds 1. Both halves agree, so `--cross` and the
  matrix are blind. `_Bool` is `ctI(1,true)` in both grammars, indistinguishable from
  `unsigned char`, so normalizing needs a new type-descriptor flag threaded through
  `convert`/`emitConv`/`ctEq` in both halves.
- **Struct layout is packed, with no alignment padding.** `{char a; short b; int c;
  long d;}` is 15 bytes here and 16 in `cc`; `&s.d - &s.a` is 7 against 8. Both halves
  agree with each other and neither agrees with `cc`.
- **~9 latent defects in the bash ERE engine**, faithfully carried over from the
  hand-emitted version during the port and deliberately not fixed there: POSIX class
  names matched by prefix, no bound checks anywhere, `@E` not inverting `@Q`. They are
  listed in the BASH convergence section of the companion document.

## Known differences, measured, lower priority

- **Our Lua vs real lua: 3,822 probe lines, all in the number FORMATTER**, not the
  arithmetic - `1e+100.0`, `-0.0` printing as `0.0`, shortest-form against `%.17g`.
  Pre-existing in both halves.
- **The two MetaJS halves bind different host globals** (`metajs-interpreter.abnf`
  lacks `Infinity NaN Array Object byteLen sprint rawSet`), so the ratchet asserts only
  the intersection. `"ß".toUpperCase()` is the same thing in miniature: `"SS"` in the
  interpreter (real JS, full mapping) and `"ß"` in the compiler (Go, simple mapping).
- **MetaJS has no exponent literals.** `1e15` fails to parse, which is a language gap
  rather than a runtime one, and it bites anyone writing float tests.
- **Identical string literals are not merged**, where `cc` merges them. Unspecified in
  C (6.4.5p7), so not a bug - but it is observable, and worth a note so nobody
  "fixes" it into a difference that matters.

## Helpers that could not move to C

- **bash: 3** (`rt_setvar_byname`, `rt_getvar_byname`, `rt_eval_assign`) - chains over
  every variable name the program mentions, generated after the walk. Movable only if
  the emitter also emits a `{name, slot}` table.
- **batch: none. CLOSED 2026-08-02.** `rt_expand` was blocked only by the
  declared-only-function pointer-parameter defect, and `e8bf2c3` fixed it - a probe
  against clean archives of `e8bf2c3^` and `660c47a` shows `declare i32* @ext1(i32 %0)`
  becoming `declare i32* @ext1(i32* %0)`. Batch is 26/26 in C. The five functions the
  grammar still generates (`bat_setlocal`, `bat_endlocal`, `bat_shift`, `bat_lookup`,
  `main`) all walk per-program slot globals, so they are program-coupled by
  construction rather than by a defect - there is nothing left to move.

---

# Part 3 - The remaining eleven languages

## Order

Sorted by extern count and Go-twin size, which is the best available proxy for effort:

```
language     externs   dedicated Go twin
java              61   804
csharp            67   813
swift             68   986
dart              72   896
js                88   (jsrtjsprint 1320 + jsrtjsbig 505)
typescript        89   shares js's
ruby              91   -   (semantics live in abnf/jsrt.go)
go                92   630 (jsrtgolang)
kotlin            96   5934
python            99   -   (semantics live in abnf/jsrt.go)
php              101   1968
```

Two adjustments to plain size order:

1. ~~**The sized-integer floor tag (Part 2) lands before java, csharp, go and kotlin**~~
   - **done** (tag 13 of `languages/lib/runtime.c`, 2026-08-02). What is left for those
   four is `jsJFlo`, the boxed double of `abnf/jsrtjvm.go`, which is the same problem
   one type over; and the six `sint*` globals in `metajs-interpreter.abnf`, without
   which the MetaJS ratchet cannot pin either of them.
2. **js and typescript go together**, since typescript has no dedicated twin and shares
   js's. Same for ruby and python, whose semantics live in the shared `abnf/jsrt.go` -
   expect those two to drag more of the shared file with them than their extern count
   suggests.

Suggested first: **java**. Smallest surface, a self-contained 804-line twin, and it
exercises the sized-integer tag immediately - which is exactly what you want the first
post-Lua language to prove.

## Per language, the shape (from the Lua pilot)

1. Split the extern list three ways: already in the C floor / language-NEUTRAL and
   belongs in the floor / language-specific and belongs in layer 2. Lua's split was
   23 / 5 / 26, and those 5 neutral ones (`js_scope_decl`, `js_scope_set`,
   `js_scope_typeof`, `js_pyset_var`, `js_keys`) are already there for everyone.
2. Write `languages/lib/<lang>-rt.metajs`, compiled by `metajs-to-llvm-ir.abnf` via a
   `tests/gen-<lang>-rt-ll.sh` (`--check` mode mandatory), linked with `-rt-lib`.
3. `abnf/jsrt<lang>.go` is the authoritative spec. Match it. Where you cannot, say so
   explicitly - approximating silently is how this project gets silently-wrong answers.
4. **Gate**: `tests/<lang>-test-full.<ext>` runs as a native binary and agrees
   byte-for-byte with `llvm.Run`, and the language moves out of clang-check's
   `ok (module only)` row into `ok, and the clang executable agrees`. Never a green
   suite over a binary nobody executed.
5. **Do not delete the Go twin** until the language passes natively and the change is
   committed.

## Traps the pilot found, which will recur

- **`_raw` parameters.** An emitter's `handle(k)` operator index is not a value handle,
  and 0..3 are the handles of `undefined/null/false/true`. This produced a silently
  wrong first build (`print(1+1)` -> `0`). Most emitters use the same trick.
- Expect each language to surface real defects in its own semantics. Lua's float `%`
  was the wrong formula in **all three halves**, and the Go twin's was
  arm64-FMA-dependent so the two Go halves disagreed on 128 of 12,557 probe lines. Use
  the real toolchain as oracle, per [oracle discipline](abnf-dialect-gotchas.md) and the
  rules at the end of the companion document.

---

# Part 4 - Regex, and then retiring Go

## Three implementations, which is two too many

```
Go    4,273 lines   abnf/jsrtregex*.go (5 files)
JS    1,559 lines   languages/lib/regex.js
C        37 funcs   re_* in languages/lib/bash-rt.c   (POSIX ERE, for bash)
```

`regex.js` already compiles through the MetaJS compiler, so the layer-2 path is open.
**Preserve the dialect modes** - `e` POSIX ERE, `j` JS repeat, `p` POSIX classes. They
are by design: `^(a*)*$` capturing `"aaa"` is JS+POSIX while Python, Java, Perl and Ruby
capture `""`, and unifying them would break three languages. Whether bash's C engine
folds into this or stays is an open question worth costing rather than assuming - it is
the only one that is already native.

## Retiring the Go runtime

29,670 lines, deletable a language at a time, only after that language passes natively.
`jsrt.go`'s host-function globals (the pre-seeded `println` and friends) become MetaJS
last, since every language depends on them.

---

# Rules (unchanged, and they earned their place this session)

- Never `git stash` / `checkout` / `reset` / `clean` - repo-wide, destroys concurrent
  work. Use `git show HEAD:<file>`.
- A change that fixes N and breaks M>N is a **LOSS**: revert it and record the
  measurement at the site.
- Ground truth is mandatory; paste the output.
- **`mec languages/x.abnf prog` reads the grammar from that path AT RUN TIME.** Building
  an old commit's binary and running it from the repo root uses the WORKING TREE's
  grammar and compares a change against itself. `cd` into each tree.
- **A crashing binary looks fast.** Check the exit code AND the expected output;
  `/usr/bin/time` reports happily on a process that died.
- Verify a commit from a clean checkout (`git archive HEAD | tar x -C /tmp/x`), not from
  the working tree.
- The defect class that keeps surfacing is **both halves agreeing and both being
  wrong**. Byte-identity cannot see it by construction. Every instance this session was
  caught by an external oracle - `cc`, real `lua`, Go's `math`.
