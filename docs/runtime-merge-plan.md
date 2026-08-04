# Merging the runtime: a shared MetaJS layer, and a smaller C floor

Third plan in the series, after [runtime-rework-plan.md](runtime-rework-plan.md) (what
was built) and [runtime-next-plan.md](runtime-next-plan.md) (GC, the floor primitives,
and all sixteen language migrations). HEAD when written: `c2e1811`.

## The goal, and the goal that is NOT here

Two things, in order:

1. **Merge what eleven layer-2 files implement separately** into a shared
   `languages/lib/runtime.metajs`.
2. **Move out of `languages/lib/runtime.c` whatever does not need to be there**, using
   "is it hot?" as the criterion rather than "is it possible?".

**The real prize is neither.** It is the DIVERGENCES the merge exposes: nine
implementations of `js_jadd` that are structurally identical and differ in their float
path is a question nobody has asked, and merging is what forces the answer.

**Explicitly NOT a goal: deleting `abnf/jsrt*.go`.** See "Why this makes the Go twin more
valuable, not less" at the end. That was measured and declined in
`runtime-next-plan.md` Part 4, and this refactor strengthens the case rather than
weakening it.

## Where things stand

```
matrix 325/325 · --full 5,686 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16, all sixteen "ok, and the clang executable agrees", none held
```

Hand-written source today:

```
C floor        languages/lib/runtime.c              5,061 lines, 301 bodies, 86 js_* externs
per-language C bash-rt.c 2,696 + batch-rt.c 490     3,186
MetaJS layer 2 11 files                            32,154
Go twin (KEPT) abnf/jsrt*.go                       30,010
```

---

# Part A - the shared MetaJS layer

## The measurement that motivates it

**29 extern names have between 2 and 9 implementations**, costing **1,427 lines across
the eleven files**:

```
js_jadd        9   csharp dart go java kotlin php python ruby swift
js_is_type     8   csharp dart go java kotlin python ruby swift
js_pyset       6   csharp dart go python ruby swift
js_mcall       6   csharp dart java lua php swift
js_dict_new    6   csharp dart go python ruby swift
js_supercall   5   csharp java js kotlin ruby
js_pylen       5   csharp dart python ruby swift
js_pyget       5   csharp dart python ruby swift
js_char        4 · js_char_code 3 · 20 more at 2
```

per file: csharp 262 · python 208 · java 176 · swift 174 · dart 152 · kotlin 130 ·
go 99 · ruby 99 · js 67 · php 56 · lua 4.

**And they are not trivially the same.** `js_jadd`, java against swift:

```js
// java                                    // swift
if (floIs(l) || floIs(r))                  if (floIs(l) || floIs(r))
    return jvmArith("+", l, r)                 return flo(swNum(l) + swNum(r), floStyle(...))
```

Same function, same shape, **different float path**. Whether they agree on every input is
unknown. That is the shape of the whole exercise.

**Name collision UNDERSTATES the duplication.** 1,665 function names appear in exactly
one file, but internal helpers are language-prefixed (`ktNum`, `phGenCall`, `dt_kget`,
`sw_kget`) so identical logic never collides by name. `sw_kget`/`sw_kset`/`sw_safeget`
transferred **verbatim** from swift to dart under new names — that duplication is
invisible to this measurement and must be found by reading.

## The mechanism already exists

Ruby proved a layer-2 file may `import` a shared MetaJS source: `import "./regex.js"`
compiles the 1,559-line engine straight into `ruby-rt.ll`, because
`metajs-to-llvm-ir.abnf` resolves relative imports and `-rt-lib`'s lazy boot runs them
into the same pinned scope. Regex cost **~460 lines of glue instead of ~2,000**. No new
feature is needed for Part A.

## Sequencing - one family at a time, never a big bang

Sixteen languages are green. A single merge commit touching eleven layer-2 files
produces a red suite nobody can bisect. Do this instead, and **verify all sixteen after
each step**:

1. **`js_jadd` first** (9 implementations, the widest, and the one with a known
   divergence in its float path). It is the pilot: it proves the mechanism, and it is
   small enough that a divergence found is a divergence understood.
2. `js_is_type` (8), then `js_pyset`/`js_dict_new`/`js_mcall` (6 each).
3. The `js_py*` family as a block — dart wrote it as language-neutral and ruby reused
   it, so it is already half-merged in spirit.
4. The 2-implementation tail, cheapest last.
5. **Then the invisible duplication**: read for the transferred-verbatim helpers
   (`*_kget`/`*_kset`/`*_safeget` in swift/dart/kotlin is the known one) and merge those.

For each family, in order:

- **Diff the implementations first, and write down every difference before merging.**
  That list IS the deliverable — a difference that turns out to be intentional is a
  language knob; one that turns out to be accidental is a defect that was invisible.
- Decide knob-or-defect **against `abnf/jsrt*.go`**, which is the independent reading.
  Where the twin and a layer-2 file disagree, the twin is the oracle unless the real
  language toolchain says otherwise.
- Merge into `languages/lib/runtime.metajs`, parameterised by the smallest knob set that
  covers the intentional differences. `lib/interp-core.js`'s `core` object is the model:
  thirteen knobs cover fifteen interpreters, each documented with the language that
  forced it.
- Re-run the full suite plus **the per-language differential probe** for every affected
  language. The probe is what catches a merge that changes an answer.

## Gate for Part A

- matrix **325/325**, `--full` **0 languages whose halves disagree**, `--cross` **119/0**,
  clang-check **16/16 all agreeing, none held**, `go test ./abnf/` ok, every
  `gen-*.sh --check` clean, `-freeze` a fixed point.
- Every affected language's differential probe (`llvm.Run` vs native) still byte-identical.
- **A written list of every difference found, each classified knob or defect**, with the
  defects fixed and pinned by a ratchet assertion whose discriminating power is measured.

## Part A - what has been done

The shared file is **`languages/lib/runtime.metajs`**, imported by nine layer-2
files. Three families are merged, two emitter bugs are fixed, and every gate is
green.

### The two emitter bugs, both fixed in `metajs-to-llvm-ir.abnf`

- **`core.importFile` appended `".js"` to any path not already ending in it**, so
  `import "./runtime.metajs"` searched for `runtime.metajs.js`. It now appends
  only when the path has NO extension at all, scanned backwards to the first `.`
  or `/` by hand (no `lastIndexOf`, no `break` in the tag script). Measured: the
  probe `import "./mjslib"` + `import "./other.metajs"` resolves under BOTH goja
  and `-frozen` after, and at HEAD fails with
  `unresolved import './other.metajs'`. `importFile` is above the
  `// ===== goja driver =====` marker, so this needed a re-freeze;
  `abnf/jsagrammar.go` and `abnf/jsbootstrap.ll` are regenerated and `-freeze` is
  a verified fixed point.
- **`-rt-lib` read its export surface from the MAIN file's source text only**
  (`c.readFile(c.file)`), so a `function js_*` in an IMPORTED file compiled, ran
  correctly under `llvm.Run`, and was **undefined at link time**. `./test.sh` and
  `--cross` are both blind to it - neither builds an `-exe` - and
  `tests/clang-check.sh` was the only gate that reported it, nine times at once.
  The scan now covers every imported file, deduplicated by symbol so a name
  declared in both cannot emit two definitions. **This removed the nine one-line
  forwarders and, more importantly, the trap for every family merged from here
  on.**

### Merged

`js_jadd` (9 languages), `js_is_type` (6), `js_dict_new` (6), plus `rtIsInt32`,
`rtWrap32` and `rtIsPlainNum` underneath them.

```
                    HEAD    now   delta        (code lines, comments excluded)
go                  1224   1120    -104
java                 819    754     -65
csharp               955    891     -64
kotlin              7416   7366     -50
swift                953    906     -47
python              3447   3410     -37
ruby                2057   2020     -37
dart                1197   1182     -15
php                 1703   1690     -13
runtime.metajs         0     68     +68
                                   ----
                                   -364     68 shared lines replace 364
```

`runtime.metajs` is 334 lines, of which **266 are the difference list**.

### The plan's headline example was wrong, and the method is what found it

`js_jadd`'s "known divergence in the float path" - java's `jvmArith("+", l, r)`
against swift's `flo(swNum(l) + swNum(r), floStyle(...))` - **is not a
divergence**. `jvmArith` is `floOp(0, ...)`, `floOp` is `jsrtjvm.go:285` calling
`rt.jvmArith`, and `rt.jvmArith` (`jsrtjvm.go:176`) is
`jsJFlo{f: rt.toNumber(l)+rt.toNumber(r), sty: jvmStyleOf(l, r)}` when either
side is a box: the same expression spelled through the floor. Likewise
`goStyleOf` and `k1StyleOf` differ only in a final arm the caller has already
excluded. The whole nine-way family reduced to **two knobs**, `js_is_type` to
**two**, and `js_dict_new` to **none - six byte-identical copies of one line**,
which is worth stating because the 29-name table counts implementations, not
disagreements.

### Defects

- **C# NaN sentinel - fixed, measured, pinned.** `rt.jsCompare` answers the
  sentinel `2` for a NaN operand and `js_cscmp` compared it as an ordering, so
  `>` and `>=` came out TRUE where ECMA-334 12.12.1 makes all four false. Fixed
  in `csharp-rt.metajs`; `abnf/jsrtcsharp.go:477` took the same fix. Measured
  from a clean `git archive HEAD` tree with the new assertion in place: the
  compiler half gave `FAIL NaN >` and `FAIL NaN >=` while the interpreter half
  passed - **the two halves already disagreed at HEAD and `--cross` never probed
  it.** Pinned by four assertions in `tests/csharp-test-1.cs`; **two of the four
  discriminate**, and the other two are said at the site to pass either way.
- **`rtWrap32` NaN/Inf guard.** `rt.toInt32` (`jsrt.go:8502`) answers 0; csharp
  and java guarded, dart/swift/go/php did not and would answer NaN. Fixed by
  merging, **zero discriminating power**, and the reason is itself a finding -
  see below.
- **`guard < 64` on the `__class`/`__super` walk.** csharp/java/kotlin/swift cap
  it; python/ruby and the oracle (`jsrt.go:8472`) do not. Kept the cap for all
  six: the oracle HANGS on a cyclic `__super`, and no language here can express a
  65-deep hierarchy. Recorded, not fixed.

### Dead code that arrived by copying

The only call site that could reach `rtWrap32`'s NaN guard in dart and swift is
`sumOf` - and **`sumOf` is a KOTLIN method name**. `dart-to-llvm-ir.abnf`,
`dart-interpreter.abnf`, `swift-to-llvm-ir.abnf` and `swift-interpreter.abnf`
mention it zero times. It was transferred verbatim out of `kotlin-rt.metajs` and
is unreachable in both files. Exactly the invisible duplication the plan
predicted, invisible to any extern table, and still present.

### Go's float rendering moved to the floor

`go-rt.metajs` carried `goFloStr` / `goSignBit` / `goDigits` / `goLayout` /
`goSci` - 110 lines - under a comment saying the floor's `floStr` could not be
used because `jvmFloText` routes style 1 to Java's layout. **That comment was
stale when it was written**: `jf_text` (`runtime.c`) sends style 1 to
`go_float_str` and `jvmFloText` (`jsrtjvm.go:65`) sends `floGo` to `goFloStr`,
landed in `d30629f`. Verified before deleting with a 47-value probe (0, the
infinities, -0.0, 1e-5 and 1e-4 either side of the %e/%f switch, 1e6/1e7, 1e20,
5e-324, MaxFloat64, 1.0/3.0) under real go 1.26.5, the go interpreter, the go
compiler, and - the one that actually links `go-rt.ll` - a **native `-exe`**.
Native == `llvm.Run` == real go on 46 of 47; line 47 is `-0.0`, where Go folds
the untyped constant to `0` in the front end, a pre-existing difference unrelated
to rendering.

### `js_bytelen` - was verified as a pair here, and has since LANDED as one

`runtime.c` documents that adding the body collides with
`php-rt.metajs`'s `function js_bytelen(s) { return byteLen(s) }`. Deleting the
layer-2 line alone leaves PHP's native `-exe` unresolvable, which is a loss, so
it is NOT deleted here. Instead the pair was **verified end to end** in a scratch
tree carrying both halves: PHP matrix 14/14, `tests/php-test-features.php` native
`-exe` 127 checks 0 failures, `llvm.Run` half agreeing, clang-check php "ok, and
the clang executable agrees". The two edits, to be made together:

```
languages/lib/php-rt.metajs   delete `function js_bytelen(s) { return byteLen(s) }`
                              (and its two comment lines), then tests/gen-php-rt-ll.sh
languages/lib/runtime.c       long js_bytelen(long v) { return mk_num(d_from_long(str_len(to_string(v)))); }
```

## Part A, second pass - the remaining families, and 862 lines of dead code

```
                  HEAD    now   delta   (code lines, comments and blanks excluded)
kotlin            7366   6750    -616
java               754    678     -76
csharp             891    827     -64
swift              906    856     -50
dart              1182   1138     -44
go                1120   1080     -40
ruby              2020   1987     -33
php               1690   1661     -29
js                2685   2664     -21
lua                679    666     -13
runtime.metajs      68    145     +77
                                 -----
                                  -909
```

`runtime.metajs` is 605 lines, of which **460 are the difference list**.

### Merged this pass

`js_pyset` (5 of 6), `js_supercall` (4 of 5), `js_char` (3 of 4),
`js_char_code` (2 of 3), and beneath them the helpers the extern table could
never see: `rtIsObj`, `rtPrepend`, `rtCharBox`, `rtIsCharBox`, `rtIsDictObj`,
`rtDictFind`, `rtFindMethod`.

### `js_pyset` - both open questions settled, and both against the machine

- **Key equality was NOT a disagreement, and the first pass's reading was
  wrong.** `===` in layer 2 is not JavaScript identity: it lowers to the floor's
  `strict_eq` (`runtime.c:2112`), whose first four lines compare a **boxed
  double (tag 14)** and a **sized integer (tag 13)** BY VALUE, in `rt.strictEq`'s
  own arm order, and `rt.strictEq` (`jsrt.go:1164`) is what `rt.dictFind`
  (`jsrt.go:2348`) finds keys with. So dart's, ruby's and swift's bare `===`
  scan IS the oracle, and `goKeyEq` -> `goStrictEq` is that same function
  respelled by hand. Measured, because a reading is not a run: a Swift probe
  keyed by `Int` 10^12 (a tag 13 box), a rebuilt `String`, and a `Double` gave
  **identical output under real swiftc 6.1.2, the swift interpreter and a native
  `-exe`**. The one real difference is csharp's `csValueEq`, whose first arm
  unboxes a `{__char}`: a Char box is an ordinary object to `strict_eq`, so a
  `Dictionary<char, V>` would miss every key without it, and the oracle has the
  arm too (`jsrt.go:1167`). That is `rtkDictFind`, and it is a knob about
  **which languages have a Char**, not about what equal keys are.
- **The index coercion IS a divergence, and merging fixed it.** The oracle is
  `int(rt.toNumber(...))`, which truncates; csharp and swift did not truncate.
  The merged body does. **Discriminating power: zero, and that is the finding** -
  ECMA-334 12.8.11.3 converts an array-access index to int/uint/long/ulong and
  Swift's `Array.subscript(index: Int)` takes an `Int`, so neither language can
  express a fractional index in any program either grammar accepts. (No C# or
  Dart toolchain exists on this machine; the C# claim rests on the standard, the
  Swift one on the standard AND on swiftc 6.1.2.)
- **A Python finding the probe threw off, not fixed and not this function's.**
  `d[True] = "t"; d[1] = "one"` is ONE entry under CPython 3.13 and **TWO in both
  metacompiler halves**, because `strict_eq` - and `rt.strictEq` with it - make a
  bool and a number different types. Both halves agree, so `--cross` is blind to
  it. It is an unimplemented Python semantic in the value model, under
  `abnf/jsrt.go`, not a merge defect.

### `js_mcall` does NOT merge, and what merged instead

Six implementations, **four calling conventions and five method tables**: csharp,
java and swift each open with a String arm over a different method set and then a
container arm over a different container model; php has neither and opens on
`phIsGen`; dart is one line into `dtMember`; lua binds the receiver as `self`
instead of prepending it. Parameterising that costs more knobs than lines.

What they DO share is the `__class`/`__super` walk that resolves a name to a
callable - **six copies of the same eleven lines**, counting `js_supercall`'s
four. That is now `rtFindMethod`, and `js_supercall` is three lines on top of it.
The 29-name table counts implementations; this row's duplication was somewhere
else entirely.

### The `sumOf` class, hunted systematically

Method-name arms in a layer-2 dispatcher, checked against whether the language
can even utter the name. **Eleven arms deleted:**

```
dart-rt      sumOf   filter  size            Kotlin's collection API. Dart has
                                             where / length, and no sumOf at all.
swift-rt     sumOf                           Swift has reduce.
ruby-rt      magnitude                       Swift's Int.magnitude, in Ruby's
                                             Numeric dispatcher next to abs.
csharp-rt    toUpperCase toLowerCase trim     the Java/JS spellings, next to the
             isEmpty includes containsKey     live ToUpper/ToLower/Trim. csMName
                                             (csharp-to-llvm-ir.abnf:3412)
                                             TRANSLATES Contains -> contains and
                                             rewrites ContainsKey into keys +
                                             contains, so none of the six can be
                                             emitted at all.
```

Four more are python's (`sumOf`, `forEach`, `isEmpty`, `removeLast`) and are
listed for `python-rt.metajs`'s owner rather than cut here.

The heuristic over-reports and the judgement is the work: `isEmpty` and
`removeLast` are REAL Swift and Dart methods and stay; `toLocaleString` is real
JavaScript; kotlin's `alnum`/`punct`/`xdigit` are POSIX classes inside the regex
engine, reachable from any user pattern. A name absent from the grammar is not
evidence, because the grammar passes the user program's own spelling through -
the question is whether the LANGUAGE has the method.

### And a bigger class the hunt found: 862 lines of unreachable functions

A call-graph reachability pass over each layer-2 compile unit (roots: every
top-level `function js_*`, which is the entire `-rt-lib` export surface, plus the
file's top level) found **60 functions no extern of their own file can reach**:

```
kotlin  36 fns  741 lines   ktFormat 108, ktTypeObj 89, ktParseNum 45, k1Sci 42,
                            ktFieldTy 22, ktFixed 21, rxSplit 21, ktLateinitCheck
                            20, k1Radix 19, rxReplace 19, ktAdoptTy 17,
                            ktSimpleName 17, k1PadNum 17, + 23 more
go       7 fns   41 lines   goClassName goIntOf goIsBox goIsFlo goIsPlain
                            goIsSlice goKeyEq
js       3 fns   30 lines   jbMcall jvInternalKey jvTodo
php      4 fns   22 lines   phIntMax phIsClass phTrunc phTwo63
lua      2 fns   17 lines   i64Fits53 i64ToNumS
csharp   1 fn     9 lines   csI
ruby     1 fn     2 lines   rbIsClass
python  13 fns  132 lines   NOT CUT - python-rt.metajs is another agent's file
```

Kotlin dominates because the file is **six independent readings stacked in one
file** (`k0` / `k1` / `k2` / `k3` / `k4` / `kt` sections): `ktFormat` and
`k2Format` are the same Go function ported twice, and so are `ktParseNum`/
`k2ParseNum`, `ktFixed`/`k2Fixed`, `ktFieldTy`/`k4bFieldTy`, `ktAdoptTy`/
`k4bAdoptTy`, `ktSetFrom`/`k4SetFrom`, `ktDelegBox`/`k4DelegBox`,
`ktIsCollection`/`k3IsCollection`. The dead half of each pair is what went. The
analysis over-approximates reachability (an identifier anywhere in a live body
counts, including inside a property access), so everything it calls dead is dead.

### `js_bytelen` - the layer-2 half is DELETED

`php-rt.metajs`'s `function js_bytelen(s) { return byteLen(s) }` is gone, and
`runtime.c` carries the body. Verified as a pair: clang-check php **"ok, and the
clang executable agrees"**, PHP matrix 14/14, and `tests/php-test-features.php`
**127 checks 0 failures byte-identical between `llvm.Run` and a native `-exe`**.

### `*_kget` / `*_kset` / `*_safeget` - the plan mislocated them

They are **not layer-2 functions at all**. `sw_kget` / `sw_safeget` /
`dt_kget` / `dt_safeget` are IR helper functions EMITTED by
`swift-to-llvm-ir.abnf:2495,2531` and `dart-to-llvm-ir.abnf:2437,2451` through
`rtFunc`, with kotlin's copy at `kotlin-to-llvm-ir.abnf:2555` saying "transferred
verbatim" at the site. `runtime.metajs` cannot hold them: they are emitter
closures that build basic blocks, not MetaJS runtime bodies. Merging them means a
shared ABNF module for the three grammars, which is real work in three files this
pass did not own.

### Not merged, with the difference list taken

- **`js_pylen` (5) and `js_pyget` (5).** Structurally the same, and the knob
  count is what stops it: `js_pyget` alone needs the dict test, the dict find,
  the key RENDERER for the KeyError message (`cpStr`/`dtStr`/`pyStr`/`rbStr`/
  `swDesc` - five different functions), the string length and the string index
  (dart routes both through `dtStrLen`/`dtStrAt`), plus python's `__getitem__`
  dunder and generic-alias arms and a message that appends the index in three
  languages and not in the other two. Six knobs for a twenty-line body across
  five languages, and every one of the six is a genuine language difference
  rather than a spelling. `js_pylen` is the same shape with dart's and swift's
  set arm added.
- **The 2-implementation tail, all seventeen rows.** Each pair costs two to four
  knobs to save ~10 lines across two languages, and the pairs are not variants of
  one function: `js_range_len` is `goNum` vs `phArrParts`; `js_pyin` is `===` vs
  `goStrictEq` and `dtToStr` vs `goStrOf`; `js_pyiter` is dart's `dtStrAt` loop
  vs python's generator drain; `js_pyrest` FAILS on a non-array in kotlin and
  answers `[]` in python; `js_this` reads a different global in each of its two.
  The one genuinely shared piece across the tail - the `__class`/`__super` walk -
  is already `rtFindMethod`.
- **Java's `jl*` collapse, and it is not the ~90-line freebie it looks like.**
  Marking a long box unsigned is what defeats `si_norm`, and `sintOp` opcode 10
  (`>>`) branches on that flag: with an honest `sintRaw` signed box `>>` becomes
  arithmetic - which is what `jlSar` was for - but Java's `>>>`, today
  `sintOp(10)` direct, then needs a helper of its own. The collapse trades
  `jlSar` for a `jlShr` rather than removing it, across ~30 call sites.

---

# Part B - a smaller C floor

## What is actually irreducible

Phase 1 of `runtime-next-plan.md` measured the floor at **35 primitives** — the ones that
cannot be written in MetaJS because MetaJS compiles to them (allocation, member
load/store, indirect call, unboxed arithmetic, scopes, the control-signal family).
`runtime.c` today has **301 bodies and 86 `js_*` externs**. The difference is not
necessity; it is expedience, and most of it arrived because a language needed an answer
quickly.

**Cold, fiddly, correctness-heavy — the candidates:**

```
d_pow  d_exp  d_log  d_sqrt  d_frexp  d_ldexp  d_modf_int  d_mod  d_mod_go
dec_absdiff  dec_cmp  dec_mul  dec_norm  dec_to_double        (the decimal bignum)
fmt_apply  fmt_sprint  fmt_top  fmt_val  js_num_str           (number formatting)
the Unicode simple-case tables (328 ranges)
```

These are the parts that have produced the most defects per line this session — the
signed-zero trap alone bit `d_pow`, `d_mod_go`, swift's `rounded()`, java, and dart's
`math.Mod`. In MetaJS they would be easier to read, easier to test, and shared.

## The criterion, and the counter-pressure

**Move it only if it is cold.** Layer 2 runs roughly **5.5x slower** than the Go path and
allocates into a collected arena; `bench-alloc.sh` is at **3,711 B/iter** with the
collector off and RSS flat at ~3.3 MB. Anything on the string or arithmetic hot path
stays in C.

So the order is: **measure first, move second.** Instrument to find which floor bodies
are actually hot on `tests/bench-alloc.sh`, `lua-bench-alloc.lua` and
`metajs-bench-try.js`, publish the list, and move only from the cold end. Report
bytes/iteration and wall time before and after **every** move; revert anything that
costs more than a few percent and record the measurement at the site.

## What can never move

Documented in `runtime-next-plan.md` Part 4 and unchanged by this work: the 35
irreducible primitives, `setjmp`/`longjmp`, the pthread coroutine block, the conservative
GC (it scans a C stack), and the libc surface. Also note the floor's strings are
**length-carrying, not NUL-terminated** (`struct Cell` + `js_str_mem(i8*, i64)`), verified
against node — do not "fix" that into C-string semantics.

## Gate for Part B

Same suite gates as Part A, plus: `tests/coro-poc/build.sh` with `--gc` and `--break`
unchanged, memory still bounded (rss flat at 250k/1M/2M), and a before/after table for
every body moved.

---

# Why this makes the Go twin MORE valuable, not less

The tempting conclusion — "once layer 2 is merged we have something like the old Go
runtime, so delete `abnf/jsrt*.go`" — is exactly backwards, and it is worth writing down
because it will be proposed again.

**Today there are eleven independent readings** of the shared semantics: eleven files,
written by eleven separate processes from `abnf/jsrt*.go`. That independence is why the
`js_jadd` divergence above exists to be found.

**After a successful merge there is one.** The merge improves consistency by destroying
independence — that is what merging means. The Go twin then becomes the *only* remaining
artifact written from a separate reading, and the only thing that can still disagree with
`runtime.metajs`.

And the structural blockers do not move:

- `llvm.Run` **cannot execute the C floor** — `setjmp`/`longjmp`, `pthread_create`+`dlsym`,
  and a collector that scans `[sp, GC_STACK_BASE)` of a C stack that does not exist there.
  Part 4 demonstrated a live array being collected. A merged layer 2 still bottoms out on
  C, so it still cannot back the interpreter.
- `abnf/jsrt.go` is the engine `-frozen` itself runs on (`newJSRT` at
  `frozen.go:220/530/813`).
- `jsrtint.go`/`jsrtjvm.go` carry the `sint*`/`flo*`/`keysOf`/`delKey` bindings that
  **layer 2 itself calls**.

**The twin is the oracle for this entire plan.** Every merge step is validated by diffing
against it. Removing it would remove the instrument.

---

# Rules (unchanged, and every one was earned)

- Never `git stash` / `checkout` / `reset` / `clean` — repo-wide, destroys concurrent work.
  Use `git show HEAD:<file>`.
- Agents never commit; the coordinator verifies and commits, by explicit path.
- **A change that fixes N and breaks M>N is a LOSS**: revert it and record the measurement
  at the site.
- **`mec languages/x.abnf prog` reads the grammar from that path AT RUN TIME.** To compare
  versions, `git archive <rev> | tar x -C <dir>`, `go build` INSIDE, run there — otherwise
  you compare a change against itself.
- **A crashing binary looks fast.** Check the exit code AND the expected output, and delete
  stale binaries before re-verifying.
- **`./test.sh --full`'s summary is not sufficient alone** — also
  `grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF'`.
- Brace value-less early returns (frozen-only ASI trap, `abnf-dialect-gotchas.md:539`).
- **Write layer 2 against the Go CODE, not the Go COMMENTS.**
- **Measure a new assertion's discriminating power** against a clean archive. Assertions
  that pass either way are decoration; say so when they are.
- Verify a commit from a clean checkout, not the working tree.
