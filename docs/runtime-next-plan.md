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

**Layer 2 cannot create a new primitive TYPE.** The Go twin carries an integer outside
2^53 as `jsGInt`, for which `js_typeof` answers `"number"`; layer 2's best is an object
box, and `js_typeof` answers `"object"`. Lua survived on ordering luck (`lua_str` asks
`js_luisnum` before `js_typeof`). **`jsGInt` is already shared by Go, Java, Kotlin and
C#**, so the next language will not be lucky.

The honest fix is a **language-neutral sized-integer tag in the C floor** - not a
per-language hack, and it must land before java/csharp/go/kotlin. The alternative
(change the emitter to stop asking `js_typeof` about a number) is worth costing too.
Do not paper over it.

**`c-interpreter.abnf` has no standard library.** `callFunction` knows `putchar` and
`getchar` and nothing else. This already cost a measured 51-assertion ratchet section
for the 18 libc names, which had to be reverted: the compiler half passed everywhere
(`cc`, `llvm.Run`, native, all printing `full: 495 checks, 0 failures`) but the
interpreter half stopped at the first call, and `--full` records a half's count only
when it has zero gaps - so one red section cost 444 assertions and the
halves-agree invariant. Adding libc to the interpreter unblocks that section and is
likely needed for the rollout anyway.

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

1. **The sized-integer floor tag (Part 2) lands before java, csharp, go and kotlin** -
   they are the `jsGInt` users.
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
