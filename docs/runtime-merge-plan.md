# Merging the runtime: a shared MetaJS layer, and a smaller C floor

> **STATUS 2026-08-04, FOURTH PASS.** The third pass's residue - 14 shape groups,
> 259 lines - was re-judged on its DIFFERENCE LISTS rather than on lines saved,
> and is now **8 groups, 154 lines, 84 recoverable**; seven of the eight need a
> file another agent owned that round. Two two-language modules landed
> (`runtime-jvm.metajs`, 2 knobs; `runtime-dartswift.metajs`, 0 knobs), the
> live-body tax was measured ON ITS OWN and turns out to have two components -
> the live walk tracks CODE (~7 pinned objects and ~+1.3% per 100 lines), heap
> and collection count track DATA - so it is an argument about hundreds of lines
> and not tens, and the merge found **`long == double` answering FALSE in both
> COMPILER engines in BOTH java and csharp** while both interpreter halves were
> right. See "Part A, FOURTH pass" below.
>
> **STATUS 2026-08-04, THIRD PASS. Part A was declared done at ~0.5% unified and
> that was WRONG** - every earlier pass worked from the extern-name table, which the
> plan itself warned UNDERSTATES the duplication. A shape scan (normalise every
> identifier to a positional token, group by the result) found 24 cross-language
> groups the name view cannot see, 776 lines, and the largest was python and ruby
> having independently written the same decimal bignum. See
> "Part A, THIRD pass - the shape scan, and the tax that reframes the whole
> exercise" below. It merged 393 code lines away, found and fixed a NaN
> relational defect in JAVA that was wrong in BOTH engines in DIFFERENT
> directions, and produced the constraint that governs every future merge:
> **a layer-2 body is PERMANENTLY LIVE, so a shared body taxes every importer that
> cannot call it** (+7-24% wall, measured three times). That is why
> `lib/runtime.metajs` is now three files, and why kotlin's 1,217-line verbatim
> copy of `regex.js` was merged, measured at +18-23%, and REVERTED.
>
> **The rest of Part A is DONE. Part B is OPEN AND UNBLOCKED, and its first
> candidate has been TRIED AND REJECTED ON MEASUREMENT.** The structural blocker is
> gone - `lib/metajs-rt.metajs` and the fifteenth generator landed in `33923d4`, so
> MetaJS has a layer 2 like the other fifteen. Move 1 (`case_map`) was then executed
> whole, passed every gate, and was **reverted**: 15x on a 1M-`toUpperCase` loop, 135x
> on a mixed-script one, 2x on a lua loop that never uppercases at all, and +156,000
> lines of checked-in IR to delete 104 lines of C. The decomposition is the useful
> part - the floor -> layer-2 CALL costs **120 ns**, the MetaJS binary search costs
> **3.2 us** - so what does not move is DATA-STRUCTURE work, not upcalls. See
> "Move 1, `case_map` - EXECUTED IN FULL, MEASURED, AND REVERTED" below.
> The single consolidated
> list of everything still open across all three plan documents is at the end of
> [runtime-next-plan.md](runtime-next-plan.md), under "WHAT IS STILL OPEN".

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

> **2026-08-04: the prize was real, but this example was not.** The merge DID expose
> defects (the C# NaN sentinel, `rtWrap32`'s NaN/Inf guard, 862 lines of unreachable
> code) - and `js_jadd`'s "known divergence in the float path" turned out to be **two
> spellings of one expression**: `jvmArith` is `floOp(0, ...)`, which is
> `jsrtjvm.go:176`'s `rt.toNumber(l)+rt.toNumber(r)`, i.e. the same thing swift spells
> directly. The nine-way family reduced to two knobs. See "The plan's headline example
> was wrong, and the method is what found it". Keep the goal; do not go looking for
> this particular divergence.

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

**Re-measured 2026-08-04 at `c1bc760`** (the table above is the snapshot this plan
was written against, kept so the deltas are readable):

```
C floor        languages/lib/runtime.c              5,204 lines   (+143: the scope API,
                                                                   isGenerator, js_bytelen)
MetaJS layer 2 languages/lib/*.metajs              31,477         (-677: Part A's merges
                                                                   and 862 lines of dead code)
Go twin (KEPT) abnf/jsrt*.go                       30,248
clang-check    16/16 "ok, and the clang executable agrees", no row held
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

1. **`js_jadd` first** (9 implementations, the widest, and the one with a ~~known
   divergence in its float path~~ **- SUPERSEDED 2026-08-04: there was no divergence,
   see the note at the top and "The plan's headline example was wrong"**). It was the
   pilot anyway, and it proved the mechanism.
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
- ~~**Java's `jl*` collapse, and it is not the ~90-line freebie it looks like.**~~
  **DONE 2026-08-04 (`9689f81`), and the cost analysis was exactly right.** As
  written: marking a long box unsigned is what defeats `si_norm`, and `sintOp`
  opcode 10 (`>>`) branches on that flag: with an honest `sintRaw` signed box `>>`
  becomes arithmetic - which is what `jlSar` was for - but Java's `>>>`, today
  `sintOp(10)` direct, then needs a helper of its own. The collapse trades
  `jlSar` for a `jlShr` rather than removing it, across ~30 call sites.

  That is what shipped. `languages/lib/java-rt.metajs` now builds every long with
  `sintRaw` (lines 119-122, 139, 154), carries **nine** `jl*` helpers, and says so
  at line 77: "`>>>` now costs `jlShr` - `jlSar` traded for `jlShr`, one helper".
  The floor half (`sintRaw`, host id 62) is `runtime.c:5099` +
  `abnf/jsrtint.go:389`. See `runtime-next-plan.md`, "THE COLLAPSE ONTO `sintRaw`".

## Part A, THIRD pass - the shape scan, and the tax that reframes the whole exercise

**Part A was declared done at ~0.5% unified and that was wrong**, for the reason
the plan itself predicted and nobody acted on: every earlier pass worked from the
EXTERN-NAME table, and internal helpers are language-prefixed so identical logic
never collides by name. The fix is mechanical - **normalise each function body
(strip comments and whitespace, replace every identifier with a positional token
`v0`, `v1`, ... in first-appearance order) and group by the resulting string.**
Over `languages/lib/*-rt.metajs`, bodies above ~60 normalised characters:

```
                                groups   lines occupied   recoverable
before this pass                    24              776           421
after                               14              259           131
```

### Merged, largest cluster first

**1. The decimal bignum (python + ruby + swift), 267 lines, ZERO KNOBS.** python
and ruby each carried a complete exact-decimal float formatter, reached through
`pyBFloStr` and `rbFloStr`, and swift carried the splitter as well. The extern
view could never see it. `python-rt.metajs:2214` SAID SO at the site ("Copied
verbatim from languages/lib/ruby-rt.metajs ... renamed to the pyB prefix"). The
difference list is EMPTY for nine of the ten functions and the tenth differed
only in the NAME of its digit alphabet (`PYB_DIGITS` vs `RB_DIGITS`, the same 36
characters). `rtSplit10` `rtZeros` `rtDigitAt` `rtBigFromNum` `rtBigMulSmall`
`rtBigStr` `rtExactDec` `rtRoundHalfEven` `rtFixed` `rtBaseStr`.

Two knobs were found ABOVE the bignum, in the float rendering, and both are real:

- `rtSciText(digits, e10, dot)` - the exponential half. `dot` forces a fraction
  digit onto a one-digit mantissa, which is Ruby's rule (`1.0e+20`) and not
  Python's or Swift's (`1e+20`).
- the BOUNDARY between exponential and plain form, which stays in the three files
  as `rtkFloDigits`: Python `decpt <= -4 || decpt > 16`, Ruby (numeric.c
  flo_to_s) `decpt < -3 || decpt > DBL_DIG`, Swift on the VALUE rather than the
  exponent (SwiftDtoa goes scientific above 2^53). **The two LOW bounds are the
  same predicate spelled twice** - `decpt <= -4` and `decpt < -3` both mean
  `e10 <= -5` - so only the high bound really differs, and it differs three ways.
- `rtFloStr(x, nanS, infS, ninfS)` - the non-finite spellings are PARAMETERS, not
  hooks: python/swift `nan`/`inf`/`-inf`, ruby `NaN`/`Infinity`/`-Infinity`.

Verified three ways per language, not by the suite (see the tax section: the
suite cannot see layer 2): swift native `-exe` == `llvm.Run` == **real swiftc
6.1.2 on all 34 probe values**; ruby and python native == `llvm.Run` == HEAD
byte-for-byte, and against real `ruby 2.6.10` / `python3 3.14.6` on everything
except two PRE-EXISTING gaps confirmed at HEAD (`%g` unimplemented in ruby-rt;
Python's `&` is int32-signed, so `-1 & 0xffffffff` is `-1` where CPython says
`4294967295`).

**2. The relational comparison - and a DEFECT in java, in BOTH engines, with two
DIFFERENT wrong answers.** `jpCompare` (java), `csCompare` (csharp), `swLoose`
(swift) and `dtJsCompare` (dart) are four copies of `rt.jsCompare`
(`abnf/jsrt.go:1530`), which answers the SENTINEL `2` when either operand
coerces to NaN and says at the site that it means "every relation is false".
csharp carries the sentinel (Part A fixed it); **java did not, and neither engine
was right**:

```
                             NaN<1  NaN>1  NaN<=1  NaN>=1
sentinel 2, read as ordering  false  TRUE   false   TRUE     <- the Go twin
0 ("equal"), old layer 2      false  false  TRUE    TRUE     <- native -exe
JLS 15.20.1 / java 24         false  false  false   false
```

Measured on a 14-row probe against **java 24**: the Go twin wrong on 4 rows,
layer 2 wrong on 6, **and the two halves disagreeing with each other on 6**. The
java INTERPRETER half was right all along (it uses the host's own `<` on
doubles), so `--cross` would have reported this the day a NaN relation was
asserted - and nothing in 5,950 assertions ever asserted one. Fixed in ONE change
across both engines: `rtCompare` in `lib/runtime.metajs` carries the sentinel,
`js_jvcmp` guards it, and `abnf/jsrtjava.go:467` takes the same three lines -
literally the fix `jsrtcsharp.go:477` already had. **Pinned by four assertions
(`flt10a`-`flt10d`) in `tests/java-test-full.java`; all four DISCRIMINATE** -
from a clean `git archive HEAD` tree they fail in the Go twin AND in the native
`-exe` and pass in the interpreter. Real `javac`/`java` 24 runs the ratchet file:
288 checks, 0 failures.

**Dart DECLINED, with the measurement.** `dtJsCompare` coerces with `dtNum`
(string -> `parseInt`, `true` -> 1, else 0), which is neither `rtkNum`
(= `dtNumOf`, NaN for a non-number) nor the oracle's `rt.toNumber`. Merging would
swap one non-oracle coercion for another on a path that `js_dartlt`/`le`/`gt`/`ge`
only reach for NON-number operands - i.e. for programs Dart rejects - and no dart
toolchain exists on this machine to settle it. It shares the NaN-through-0 shape
and is recorded, not fixed.

**3. Three zero-knob merges onto bodies that were ALREADY shared.** `csharp` and
`java` each called `rtFindMethod` in one place and still carried a *second*
private copy of the same eleven-line `__class`/`__super` walk (`cpFindToString`,
`jpFindMember`); `dart`, `kotlin` and `swift` each carried a private copy of
`rtIsDictObj`. Both are pure deletions at zero cost - the importer already links
the shared body. Plus `rtSubstring` (`csSubstring` == `jpSubstring`, byte for
byte).

### THE FINDING THAT REFRAMES PART A: layer 2 is PERMANENTLY LIVE

**A shared body taxes every file that imports it, whether or not that language
can call a single line of it.** Every function in a `<lang>-rt.ll` is a pinned
closure the conservative collector walks on every collection. Measured three
times this pass, each on a program that cannot reach the merged code:

```
kotlin, 400k x `s = s + i % 7`, native -exe, MEC_GC_STATS
  HEAD                                    21.6-22.5 s   live 222,096   heap  9.47 MB
  + the decimal bignum in runtime.metajs  23.5-25.1 s   live 224,400   heap 10.51 MB   +7..11%
  + it split into its own module          21.0-21.5 s   live 222,432   heap  9.47 MB   none

swift, 400k x `s = s + i % 7`
  HEAD                                    3.13-3.16 s   live 103,536   heap  9.44 MB
  + the bignum it never calls             3.70-3.94 s   live 105,072   heap 10.49 MB   +17..24%
  + the bignum split out again            3.10-3.35 s   live 104,016   heap  9.44 MB   none
```

So `lib/runtime.metajs` is now **three files**: `runtime.metajs` (imported by
ten), `runtime-decimal.metajs` (the float rendering - python, ruby, swift) and
`runtime-bignum.metajs` (the exact decimal and `FormatInt` - python, ruby). This
is the same lesson the `case_map` revert recorded from the other direction, and
it is a STANDING CONSTRAINT on every future merge: **a body shared by 2 of 10
importers is a tax on 8 unless it gets its own module.** That, and not arithmetic
on lines saved, is why the small two-language pairs below are declined.

### A LOSS, executed whole and REVERTED: kotlin's verbatim copy of regex.js

`kotlin-rt.metajs` contains **the whole of `lib/regex.js` pasted in** - the shape
scan says **51 of its 57 functions are byte-identical, 1,217 lines, zero
divergences**, the single largest duplication in layer 2, and the file admits it
at the site. Replacing it with `import "./regex.js"` was done in full and it
WORKS: kotlin matrix 47/47, native `kotlin-test-full` 979 checks 0 failures,
source 9,372 -> 7,901 lines. It was reverted on the measurement above: **+18 to
+23% on a kotlin loop with no regexp in it**, because the import restores 20
function bodies kotlin had deleted as unreachable and every one of them is
permanently live. The measurement is recorded at the site in
`kotlin-rt.metajs`, along with the instruction to keep the two texts in step by
hand and the scan that checks it.

### THE GATE NOBODY HAD: `./test.sh` CANNOT SEE LAYER 2

`llvm.Run` uses the Go twin and never links `*-rt.ll`, so the matrix, `--full`
and `--cross` are all blind to `lib/*-rt.metajs`. Measured: with a deliberate
wrong `dot` knob in ruby's `rtkFloDigits`, `./test.sh --full --filter ruby`
reports **279 assertions, halves agree, green** - while a native `-exe` of the
same `tests/ruby-test-full.rb` reports **4 failures** (n16b, n16c, n16d, n16e).
`tests/clang-check.sh` is the only committed gate that links layer 2 and it runs
one small program per language. Every group in this pass was therefore verified
by building `tests/<lang>-test-full.<ext>` as a native `-exe` and running it, for
all fifteen languages that have one; the counts come out exactly equal to
`--full`'s (python 375, ruby 279, swift 211, java 288, csharp 254, dart 232,
kotlin 979, go 319, php 306, lua 330, js 378, metajs 557, typescript 300, c 577,
bash 430), 0 failures. **That harness belongs in `tests/`; it is the missing
gate.**

### Declined, with the reason - and the 21 externs re-judged

The prompt for this pass asked for the still-duplicated externs to be re-judged
"on the difference list rather than the arithmetic". The answer changed, but not
in the direction expected: **the tax measurement is a stronger argument against
them than the lines-saved heuristic ever was.** A 10-line body shared by 2 of 10
importers costs 8 languages live bytes on every collection to save 10 lines, and
a private module per pair is not a structure anyone can maintain.

- `csNum`/`jvNum` (2 hooks: the char box and the string coercion - csharp
  `parseFloat`, java `jpStrNum`), `csStrictEq`/`jvStrictEq` (2 hooks),
  `csFloArith`/`jvmArith` (1 hook, and the operand mappers genuinely differ - see
  js_jadd note 3), `js_cscmp`/`js_jvcmp` (2 hooks). Four pairs, ~35 lines
  recoverable, all csharp+java only.
- `dtStrLen`/`k2RuneLen`, `dtJavaStr`/`swJavaStr`, `dtIsArr`/`swIsArr`,
  `pyIsObj`/`rbIsObj`, `luChar`/`js_char`, java/kotlin `js_jband`.
- The regex glue: `jxGetProg`/`k5Get`/`pyERxGet`/`rbRxGet` are FOUR byte-identical
  eight-line memos differing only in the cache variable. Their natural home is
  `lib/regex.js` - which js, python and ruby already import at zero tax - but
  `abnf/jsrtregex.go` is "a line-by-line port of this file" and kotlin holds a
  verbatim copy, so one 24-line saving costs an edit in three places that must
  stay in step. Declined on that, not on the arithmetic.
- `k5MatchOf`/`rbRxMatchOf` + `k5ObjRe`/`rbRxObjRe` + `ktRxIsRegex`/`rbRxIsRegex`
  (kotlin+ruby, 62 lines, ~35 recoverable) is the best remaining candidate and
  wants a two-language module of its own. Left open with the difference list
  taken: the knobs are the numeric coercion (`ktNum` vs `rbToF`, i.e. `rtkNum`)
  and the nullish test (`ktIsNullish` vs `rbIsNil`).

### Where things stand after this pass

```
                    HEAD    now   delta   (code lines, comments and blanks excluded)
python-rt           3892   3644    -248
ruby-rt             2067   1818    -249
swift-rt             854    765     -89
csharp-rt            810    772     -38
java-rt              692    659     -33
dart-rt             1145   1139      -6
kotlin-rt           6750   6744      -6
runtime.metajs       145    167     +22
runtime-decimal        0     74     +74
runtime-bignum         0    180    +180
                                   -----
                                    -393     426 shared lines replace 819
```

Gates, all re-run after every group: matrix **329/329** - `--full` **5,954
assertions** (5,950 + the four java NaN ratchets), **0 languages whose halves
disagree**, no `BUT -frozen`/`VACUOUS`/`MISMATCH`/`FROZEN-DIFF` - `--cross`
**119/0** - clang-check **16/16, all sixteen "ok, and the clang executable
agrees", none held** - `go test ./abnf/` ok - all fifteen `gen-*.sh --check`
clean - `-freeze` a fixed point (`jsagrammar.go` and `jsbootstrap.ll` byte-identical
after re-running it) - `MEC_GC=off` **3,711 B/iter** at 400k with gc RSS flat at
3.34 MB across 50k/100k/200k/400k - and the fifteen native `-exe` full-test runs
above.

---

## Part A, FOURTH pass - the residue re-judged on its DIFFERENCE LISTS, the tax
## measured on its own, and a cross-engine defect in TWO languages (2026-08-04)

The third pass left **14 shape groups, 259 lines, 131 recoverable** and declined
them on the tax argument. This pass re-judged each group on its DIFFERENCE LIST
and merged what the lists justified.

```
                                groups   lines occupied   recoverable
after the third pass                14              259           131
after this pass                      8              154            84
```

(The scan is the third pass's, reproduced exactly: normalise every identifier to
a positional token, group by the resulting string, keep cross-language groups
whose normalised body is at least **140 characters**. That threshold reproduces
14/259/131 at `702dc68` to the line. At a permissive 60 characters the same scan
reads 42/638/381 before and 35/508/319 after, and the extra groups it sees at
that threshold are named below.)

### THE TAX HAS TWO COMPONENTS, and only one of them tracks code

The third pass declined every small pair on the live-body tax, having measured it
**once, on a 267-line body with data in it**. Measured on its own, with dead
code-only bodies (no module-level data, no string literals) appended to
`swift-rt.metajs` and never called - swift, 400,000 x `s = s + i % 7`, native
`-exe`, `MEC_GC_STATS`, min of three runs:

```
added lines   live objects        heap        collections   wall
      0       104,016             9,442,688   4,097         3.40 s
    182       105,168  (+1,152)   9,442,688   4,097         3.52 s   +3.5%
    720       109,072  (+5,056)   9,442,688   4,097         3.71 s   +9.1%
```

So the **live walk tracks CODE**, linearly, at ~7 pinned objects and ~+1.3% wall
per 100 lines; **heap and collection COUNT track DATA** and did not move at all.
The decimal bignum's +17-24% was both together (it moved heap 9.44 -> 10.49 MB
and collections 3,841 -> 4,097 as well as live +1,536).

**That changes the verdict on the small pairs.** A ten-line body shared by 2 of
10 importers costs the other eight ~70 live objects each, ~0.13% - under the
noise floor of a single timing run. The tax is an argument about *hundreds* of
lines; using it against *tens* was the same inversion, one heuristic later, that
made the first Part A pass stop at 0.5% unified. What it IS an argument for is
the split module, which costs the non-importers exactly zero - so both merges
below are new two-language files rather than additions to `runtime.metajs`.

It also answers, negatively, the open question the manual records about kotlin's
verbatim `regex.js`: **the tax tracks code, so 1,217 lines of mostly-code predicts
+15% and the measurement was +18-23%.** A `runtime-regex.metajs` split does not
make that free for kotlin; only not importing it does.

> **The prediction in that last paragraph is DEAD, and so is its reason.** The
> import shipped on 2026-08-04 at a cost indistinguishable from zero; see "THE
> THIRD ASKING". The paragraph below is preserved as the record of what was
> believed at the time.
>
> **The prediction in that last paragraph is right; its reason is not.** The
> section below re-measured it on the actual regex body, in instructions retired
> rather than wall clock, and the code-vs-declaration confound is disentangled by
> a 2x2. The verdict on kotlin's copy is unchanged - the split is still declined -
> and the coefficient to use is per DECLARATION, not per line.

### THE TAX IS DECLARATIONS, NOT DATA AND NOT CODE - and it is not the collector

**Wall clock cannot settle this question on this machine.** Two agents were
running beside this one; a 40,000-iteration kotlin loop varies **+-8% between
runs of the same binary**, which is larger than every effect being argued about.
`/usr/bin/time -l` reports **instructions retired**, which reproduces to **0.02%**
across rebuilds, and every number below is that. Harness: kotlin,
`40,000 x s = s + i % 7` (no regexp, no float), native `-exe`, base **30.506 G
instructions**, `MEC_GC_STATS` quoted where it moves.

**1. The one measurement that has to be made first: relocate, changing nothing.**
Take `kotlin-rt.metajs`, leave every byte of it alone, and move the 1,489-line
verbatim `regex.js` block to a different point in the file. Same text, same 461
top-level declarations, same emitted `.ll` line count, `MEC_GC_STATS` identical to
the byte (collections 1,880, live 222,080, heap 9,465,728) in every row:

```
regex block at declaration index    0      50     150     250   ~330 (HEAD)
instructions                    33.401  31.818  30.521  30.366  30.506 G
                                 +9.3%   +4.3%   -0.1%   -0.6%     --
```

**A +9.3% swing from moving text.** That number is the whole of what the third
pass attributed to live data, because `import` PREPENDS an imported file's
top-level declarations to `jsrun` (`metajs-to-llvm-ir.abnf` ~590 / ~1011), so
`import "./regex.js"` *is* the index-0 row.

**2. The regex import, decomposed.** `impe` imports an engine-only `regex.js`
with exactly the declarations kotlin's copy already has; `imp` imports the whole
file, six declarations more (`rxMatchAt` `rxTest` `rxGroupCount` `rxNameAt`
`rxReplace` `rxSplit`).

```
                          instructions   vs HEAD   live      heap
HEAD, verbatim copy         30.506 G       --      222,080   9,465,728
relocate the copy to 0      33.401 G      +9.3%    222,080   9,465,728
import engine-only (impe)   33.401 G      +9.5%    222,080   9,465,728
import "./regex.js" (imp)   33.760 G     +10.7%    222,752   9,465,728
```

`impe` == `move` to three decimal places. **The import mechanism itself is free**
(importing a two-line module: +0.2%). The 1,217 duplicated lines are worth
**+1.1%**, and the other +9.5% is placement, which no split can move.

**3. IT IS NOT THE COLLECTOR.** Subtract a `MEC_GC=off` run from each:

```
                     total      MEC_GC=off   collector's share
HEAD                 30.508 G     25.415 G      5.092 G
relocated            33.401 G     28.309 G      5.093 G
import engine-only   33.401 G     28.338 G      5.063 G
import regex.js      33.760 G     28.689 G      5.071 G
```

The collector's share **does not move**. Every one of these differences is
MUTATOR work, in `runtime.c`'s `scope_find` (2755) - a **linear scan** of the
module scope's name array, over 461 entries for kotlin. A declaration costs twice
over: it delays every lookup that resolves after it, and it lengthens every
lookup that MISSES the module scope and falls through to `G_ROOT`. That is why
prepending costs about twice what appending costs, and why the collector is
innocent.

**4. The 2x2 that separates lines from declarations.** The swift filler above
varied line count and function count TOGETHER, which is the confound. Hold one
fixed and move the other (all appended, so placement is constant):

```
                    decls  code lines    live      instructions   vs HEAD
HEAD                   0        0      222,080      30.506 G        --
A                      4      728      222,464      31.091 G       +1.9%
B                     40      760      240,672      32.722 G       +7.3%
C                     40       80      240,672      32.161 G       +5.4%
tiny5                  5        5      222,608      31.128 G       +2.0%
fat5                   5    1,220      222,608      31.122 G       +2.0%
```

- **`tiny5` vs `fat5`: 244x the code, byte-identical `live`, and the same cost to
  0.01%.** Code size does not enter the live set at all.
- **A vs B: the same ~730 lines, 4 declarations against 40 - +1.9% against
  +7.3%.** Declarations, not lines.
- **B vs C: the same 40 declarations, 760 lines against 80 - identical `live`,
  +1.8% apart.** So there IS a code-size component, but it is ~0.2% per 100 lines
  and it is NOT in `live` (i-cache and code layout, not the marker), an order of
  magnitude below the +1.3%/100 lines the swift filler suggested.

The swift rows are consistent with this once read the other way: `+1,152` and
`+5,056` live BYTES (`GC_LIVE` is bytes, not objects - `runtime.c:674`) over
filler whose function count grew with its line count, which is ~120 bytes per
DECLARATION, the same figure kotlin gives. Part of kotlin's jump at 40 extra
declarations is the scope array's capacity doubling (512 -> 1024 slots x 3 arrays
x 8 bytes = 12,288 bytes in one step), which is a step, not a slope.

### THE RULE, so the next person does not re-measure

> **THIS RULE IS DEAD. `scope_find` now has a hash index (2026-08-04) and the
> declaration tax it measures is GONE — every coefficient below is 0 to within
> the noise floor. It is kept because the reasoning that produced it is right
> and because the section after next re-measures each row. Do not size a shared
> body in declarations any more; size it in whatever you like, because a
> layer-2 body a language cannot call now costs that language nothing.**

**A shared layer-2 body taxes a non-calling importer by roughly**

```
    0.13%  per top-level declaration it adds          (it lengthens every miss)
  + 0.11%  per top-level declaration, if PLACED FIRST (it delays every hit)
  = 0.24%  per top-level declaration for an IMPORTED body, because import prepends
  + 0.2%   per 100 lines of code, from layout - not from the collector
  + 0        per byte of live data beyond the declaration's own cell
```

on a kotlin-sized module (461 declarations) with a hot loop that calls layer 2.
The model closes on the case it was derived from: regex.js is 86 declarations, of
which kotlin already has 80, so 86 x 0.11 (placement) + 6 x 0.24 (new) = **9.9%**
against **10.7%** measured.

**The threshold, stated as the deliverable asked.** Size a shared body in
DECLARATIONS, never in lines:

- **Up to ~10 declarations (~2.4% imported): merge it.** That is the decimal
  bignum and `runtime-decimal.metajs` together - 14 declarations, measured at
  **+2.2%** on kotlin (30.506 -> 31.193 G), not the +7-11% the third pass
  recorded. The splits stay, because they cost nothing and are the right shape,
  but the number to quote is the small one, and the small pairs the third pass
  declined on this tax were declined on a number that was too big.
- **Beyond ~40 declarations (~10%): do not import it into a language that cannot
  call it**, whatever it saves in source. There is no line count at which a body
  becomes too big; there is a declaration count.

### kotlin's `regex.js`: the SECOND revert, and what would change the answer

> **OVERTAKEN. Item 2 below landed, and the import was measured a THIRD time and
> SHIPPED - `kotlin-rt.metajs` imports `regex.js` today and the verbatim copy is
> gone. Read "THE THIRD ASKING" instead; it also shows that this section's
> `impe` experiment, and every other single-build difference below ~2% in this
> file, is inside a build-layout lottery and cannot be read as a measurement.
> This section is kept because its decomposition of the cost was correct in the
> era when the cost was large enough to see.**

The split was measured and **declined again**. A `runtime-regex.metajs` holding
the shared engine - the exact experiment, `impe` above, an engine-only module
carrying precisely the declarations kotlin's copy carries - costs kotlin
**+9.5%** against the verbatim copy it would replace. It is not a wash: today's
copy sits at declaration index ~330, and an import lands at index 0. Only the six
bodies kotlin omits, **+1.1%**, are on the table, and 1,217 lines of duplication
is not worth 1.1% of every kotlin program.

**What would have to change**, and neither is in this file:

1. **`import` appending rather than prepending.** An import placed at declaration
   index 150 or beyond measures at or below the noise floor (-0.1%, -0.6%). If
   `metajs-to-llvm-ir.abnf` emitted an imported module's top-level declarations
   AFTER the importing file's own, kotlin's `import "./regex.js"` would cost
   ~+1.1% instead of ~+10.7%, these 1,217 lines could be deleted, and the
   `runtime-decimal` / `runtime-bignum` / `runtime-jvm` / `runtime-dartswift`
   splits would stop costing their non-callers anything at all. It is one
   ordering decision in the stash-and-prepend path (~590 / ~1011).
2. **A hash or sorted index on `scope_find`.** The whole of this tax is a linear
   scan over a 461-entry array that never changes after boot. It is also worth
   more than this item: the collector is only 17% of these runs (5.09 G of
   30.5 G), and the mutator's biggest single line is that scan.

Until one of those lands, the two texts stay in step BY HAND; the shape scan in
`kotlin-rt.metajs` at the copy is how to check that a change to `lib/regex.js`
arrived here too.

### BOTH OF THOSE WERE TRIED. ITEM 2 LANDED AND ENDS THE TAX; ITEM 1 IS A LOSS

Same harness throughout: kotlin, `40,000 x s = s + i % 7`, native `-exe`,
`/usr/bin/time -l` **instructions retired**, min of three, and every comparison
against a clean `git archive 85f3409` build in its own tree (the manual's §4 trap:
`mec` reads the grammar from the path you give it, so an old binary run in this
repo uses THIS repo's grammar).

#### Item 1, `import` APPENDING: measured, and it is a LOSS of +42.5%

The prediction above is wrong, and the reason is worth more than the change would
have been. **What a declaration's index costs is not how many declarations
precede a body - it is where the HOT declarations sit.** `kotlin-rt.metajs`
imports `runtime.metajs`, whose 19 declarations include **`js_jadd`, kotlin's
`+`**. Prepended they sit at index 0..18 of a 461-entry module scope and every
`+` in the program finds one almost immediately; appended they sit at 442..460
and every `+` scans the whole array to get there.

```
                                  instructions   vs prepend   live      heap
import PREPENDED (HEAD)             30.970 G         --       222,080   9,465,728
import APPENDED                     44.131 G       +42.5%     222,080   9,465,728
```

`MEC_GC_STATS` is **byte-identical** in both rows, so all of it is mutator work
in `scope_find`, exactly as the section above says - the model of WHICH work was
just backwards. The flip is also semantically observable (MetaJS does not hoist,
so a main-file top-level statement calling an imported function would now run
before its declaration), which was the risk the deliverable asked about; it is
moot, because the ordering is reverted. `metajs-to-llvm-ir.abnf`'s `ImportStmt`
rule now records this so nobody "fixes" the order again.

#### Item 2, the hash index on `scope_find`: LANDED, and it is 20-42% off nine languages

`runtime.c`'s scope cell already packs names, values and type classes into the
three thirds of ONE block. It now carries a fourth part: an open-addressed table
of `2*cap` slots holding `index + 1`, built only for a scope of 32 entries or
more, rebuilt on every capacity doubling, and reachable from exactly where the
names buffer already was - so the conservative collector needs no change and
`MEC_GC=stress` is clean. The name's hash is FNV-1a over its BYTES (two modules
are linked in a native build, so the same name is two different interned cells),
memoised in the string cell's field `f`, which nothing else uses and `gc_trace`'s
tag-4 arm never reads.

Semantics do not move: `scope_put` never inserts a duplicate and nothing ever
removes an entry, so at most one index can match and the answer is the same one
the scan gave. `js_scope_get` / `scopeHas` / `js_scope_has` are unchanged.

The same loop in each language, native `-exe`, against the same clean base:

```
  python      133.928 G -> 77.683 G   -42.0%      lua      7.982 G -> 8.308 G  +4.1%
  kotlin       30.502 G -> 18.013 G   -40.9%      metajs   1.198 G -> 1.203 G  +0.4%
  ruby         25.675 G -> 15.549 G   -39.4%      c       11.04  M -> 11.03 M  -0.1%
  swift        15.062 G ->  9.965 G   -33.8%
  php          30.111 G -> 20.366 G   -32.4%
  go            2.990 G ->  2.090 G   -30.1%
  java         36.562 G -> 25.702 G   -29.7%
  typescript    4.577 G ->  3.274 G   -28.5%
  dart         13.271 G ->  9.466 G   -28.7%
  js            4.577 G ->  3.363 G   -26.5%
  csharp       63.628 G -> 51.190 G   -19.5%
```

`c` is self-contained IR with no scopes at all and is the control.

**lua is a real regression and it is NOT the hashing.** lua's 84-declaration
module scope is the smallest of the sixteen and its benchmark is dominated by
SMALL scopes. Isolated by building a runtime whose index can never fire: a bare
`if (n >= 1000000) return <call>` added to `scope_find` already costs lua
**+2.8%**, because at HEAD that function is small enough for clang to inline into
`scope_get`, `scope_put` and `js_tdecl`, and a second arm makes it too big.
`sample` on a 4M-iteration lua loop puts 33.5% of the profile in the scope family
against 23% before, with the extra samples on the LINEAR path. Two attempts to
buy it back were measured and are worse, and both are recorded at the site:
splitting the arm into its own function costs lua **+4.7%**, and carrying the
name handle beside the index in a two-word slot costs **+6.1%** (and kotlin
-40.4% instead of -42.1%). It is kept at +4.1% for lua and +0.4% for metajs
against -20 to -42% for the other nine.

#### The declaration-index curve, RE-MEASURED - the tax is gone

The experiment the rule was built on, repeated on the shipped runtime: relocate
kotlin's 1,489-line verbatim `regex.js` block, changing nothing else.

```
                                    before (85f3409)      after the hash index
regex block at HEAD placement          30.970 G                18.023 G
regex block at declaration index 0     33.399 G   +7.8%        17.923 G   -0.6%
import runtime.metajs APPENDED         44.131 G  +42.5%        18.130 G   +0.6%
```

**Placement is now +-0.6%, which is the noise floor of the harness.** Every
coefficient in "THE RULE" is 0:

```
    0%  per top-level declaration it adds
  + 0%  per top-level declaration if PLACED FIRST
  + 0%  for an IMPORTED body, whichever end it lands at
  + the same ~0.2% per 100 lines of layout, which was never the collector either
```

**What follows for the merge decisions this document made on the old arithmetic.**

- The `runtime-decimal` / `runtime-bignum` / `runtime-jvm` / `runtime-dartswift`
  splits cost their non-callers **nothing**. They can be merged back into
  `runtime.metajs` whenever that is the tidier shape; the reason they were split
  no longer exists.
- **The small pairs the third pass declined on the live-body tax were declined on
  a number that is now zero.** Every one of them is worth re-opening.
- **kotlin's verbatim `regex.js` copy: the case for a `runtime-regex.metajs`
  split is now open, and this is the third time of asking.** The +9.5% that
  declined it was placement, and placement is free; what is left is the +1.1% the
  six extra bodies cost, measured before the index and worth re-measuring after
  it. 1,217 lines of hand-synchronised duplication against ~1% is a judgement
  call rather than an obvious no - which is a different answer from the one this
  document has given twice. NOT executed here: it is a `kotlin-rt.metajs` change
  and this round owned `runtime.c`.
  **Re-measured and EXECUTED the next round - see the section below.**

The `scope_find` scan is no longer the mutator's biggest line. Whatever is
now, nobody has looked - `sample` on the lua and kotlin binaries is how to find
out, and `ar_block`, `js_str_mem` and `obj_find` are the names that came up.

### THE THIRD ASKING: the split is EXECUTED, and its cost is INDISTINGUISHABLE
### FROM ZERO - plus the harness defect that invalidates every +-1% row in this
### document (2026-08-04)

Two reverts, and now the third measurement retires both. `kotlin-rt.metajs`
line 7275 is `import "./regex.js"`; the 1,489-line verbatim copy is deleted.
**No `runtime-regex.metajs` was created**, and the reason is below.

#### READ THIS FIRST: single-build A/B COMPARISON DOES NOT WORK ON THIS TARGET

This section originally reported **+0.04%**. The coordinator, verifying, measured
**+0.65%** on the same two sources with a harness that looked *more* careful than
mine. Neither number was wrong and neither was reproducible, because **both were
single draws from a build-layout lottery about four percent wide.**

The proof, in three steps:

**1. The build is deterministic and the run is quiet.** Same source, same output
path, six rebuilds: the binary is `md5`-identical every time, and instructions
retired span 18.018-18.028 G - **0.05%**. So neither codegen nor machine noise is
the variance.

**2. Changing only the OUTPUT FILENAME moves it by percent.** Take one
byte-identical `kotlin-rt.ll`, one source, one tree, and vary nothing but the
`-exe` path's *length*. Twenty draws of the verbatim-copy build:

```
  17.756 17.779 17.910 17.912 17.912 17.914 17.916 17.920 17.937 17.943
  18.017 18.018 18.021 18.023 18.029 18.029 18.031 18.035 18.071 18.600  G
```

**A 4.8% spread from the name of the output file**, clearly bimodal around
~17.92 and ~18.03. The path string lands in the binary and moves the code layout.

**3. Therefore "same exe path for both variants" cancels NOTHING.** This is the
trap, and it is the one I fell into and then wrote into this document as the fix.
Layout is perturbed by the path string *and* by the module's own content, in the
same way and by the same magnitude. Holding the path fixed while changing the
module still draws a fresh sample. **There is no pairing that cancels it.** The
only sound estimator is a mean over many layout draws.

Twenty draws per variant, one run each, non-regex loop; fourteen draws per
variant on the regexp loop; ten on a near-empty program:

```
                          old (verbatim)        new (import)        delta
40,000 x s = s + i % 7    mean 17.989 G         mean 17.976 G      -0.07%
  (n=20/20)               range 17.756-18.600   range 17.523-18.212
                          95% CI on the delta [-0.60%, +0.38%]   (bootstrap)

1,000 x regexp ops        mean 15.052 G         mean 15.093 G      +0.28%
  (n=14/14)               range 14.799-15.312   range 14.695-15.558
                          95% CI on the delta [-0.60%, +1.17%]

println("hi")             med  27.076 M         med  27.048 M      -0.10%
  (n=10/10, module init dominates; MEC_GC_STATS byte-identical, pinned=13)
```

**Both confidence intervals straddle zero.** The distributions overlap almost
completely. The change is not measurable on this machine by this harness, in
either direction, on either kind of program.

**Every headline this section first carried was an artifact, and all three are
withdrawn**: the +0.04%, the -1.72% "straight win" on the calling path, and the
`impe`-is-slower inversion. So was the coordinator's +0.65%. What survives is
that the true cost is *smaller than a 4% measurement window*, which the mechanism
independently predicts: the import adds six top-level declarations to a
461-entry module scope that **has had a hash index since e1307f3**, so lookups
are O(1) and the only real work added is six extra `js_tdecl` calls at module
init - hundreds of instructions against 1.8e10. The near-empty-program row, where
module init is most of the runtime, is where that would show, and it does not.

**The correction this forces on the rest of this document.** The post-index curve
two sections up reads `-0.6%` and `+0.6%`; those are single draws and should be
read as **0 +- 2%**. Nothing in this file that rests on a single-build difference
below ~2% means anything. **Any future layer-2 sizing decision must average over
at least ~15 layout draws** (vary the `-exe` filename length; it is free) and
quote a confidence interval, or it is not a measurement.

#### The decision, re-argued on the honest numbers

The performance argument is now *empty in both directions*, so the split has to
stand or fall on correctness - and that is where it was always strongest:

- **1,489 lines that had to be kept in step BY HAND**, against a `regex.js` that
  three other languages already import and that `abnf/jsrtregex.go` line-by-line
  ports. Three texts, one of them maintained by hand-diffing. The shape scan that
  found this had to be *run* to notice; nothing else could.
- The standing precedent declined at **+1.1%**, but that +1.1% was a REAL cost
  with a mechanism: `scope_find` was a linear scan, and six extra declarations
  genuinely lengthened every lookup that missed the module scope. **That
  mechanism no longer exists.** Declining now would be declining at 0.
- The only thing kotlin gains is six bodies its grammar cannot reach
  (`rxMatchAt`, `rxTest`, `rxGroupCount`, `rxNameAt`, `rxReplace`, `rxSplit`).

**And it is why no `runtime-regex.metajs` exists.** Such a module would carry
exactly the copy's declarations and drop those six - but since six declarations
are worth nothing measurable, it buys nothing, while re-introducing a second
regex text to keep in step. It was built and measured (`impe`) before this was
understood; the measurement said "slower", which was noise, and the right reason
to reject it is that it has no purpose.

If the reviewer's judgement is that an unmeasurable-but-possibly-positive cost on
every kotlin program is not worth 1,489 lines of de-duplication, **that is a
defensible call and the working tree is the place to make it** - the change is one
`git checkout` of two files. What is not defensible is either of the two numbers
this section first quoted.

#### The 2.2% spread on the coordinator's old rows

Asked about, and **I cannot reproduce it as a rebuild effect.** Six rebuilds of
one source to one path give `md5`-identical binaries and 0.05% spread, so two
builds with identical inputs cannot differ by 2.2%. Two explanations fit: the two
"old" builds did not have identical inputs (a different `-exe` path between them
puts them 2-4% apart, as above), or machine interference during those runs. It is
worth `md5`-ing the two binaries to tell which - but either way 2.2% is exactly
the width of the layout lottery, and it is more evidence for it, not against.

#### What it costs, and what it was tested with

kotlin now carries six bodies its grammar cannot reach: `rxMatchAt`, `rxTest`,
`rxGroupCount`, `rxNameAt`, `rxReplace`, `rxSplit`. (The deleted copy's own header said
"SIX of the entry points - rxMatchAt, rxMatchWhole, rxTest, rxReplace, rxSplit,
rxGroupCount, rxNameAt", which names **seven** and includes `rxMatchWhole`, a
body that was present in both texts. The list was wrong; the count was right.)

**How it was tested, since §3 of the manual says the matrix cannot see layer 2.**
A kotlin regexp differential probe - 45 patterns x 37 texts plus named groups,
group templates, replacement lambdas and the five `RegexOption`s, every operand
read out of an array so the folder cannot fold it, 1,694 lines of output covering
matching, groups, named groups, replace, split, anchors, classes, quantifiers,
alternation, backreferences, lookahead and the lazy forms. Three ways, all
**byte-identical**: `llvm.Run` (the Go twin) against the native `-exe` after the
split, and the native `-exe` *before* the split against the native `-exe` after
it. The last of those is the one that matters - it is the direct statement that
this deletion changed no answer. The probe also confirmed the engine's own
limits, which are unchanged: lookbehind, POSIX bracket expressions and inline
`(?i)` groups are rejected at compile time with their own messages, so those
patterns are not in it.

**No `kotlinc` on this machine** (manual §5). The ground truth for the behaviour
being preserved is `java.util.regex`, which Kotlin's `Regex` is specified to
delegate to - but the split needed no appeal to it, because the before/after
native outputs are byte-identical and `tests/kotlin-test-full.kt`'s 45 `re*`
assertions (`re01`..`re44`) run natively in `tests/native-full.sh` at 979 checks,
0 failures.

**Left for another round, because those files were owned elsewhere this round.**
All four regexp languages now import `regex.js`, and all four then declare the
same eight-line memo over it, differing only in the name of the cache object:

```
    js-rt.metajs:3067      jxGetProg / jxCache
    python-rt.metajs:4723  pyERxGet  / pyERxCache
    ruby-rt.metajs:1501    rbRxGet   / rbRxCache
    kotlin-rt.metajs:7272  k5Get     / k5Cache
```

Every language maps its own flag letters onto the neutral set *before* calling,
and `rxCompile` is deterministic and language-neutral, so the four caches do not
need to be separate at all: one `rxGet(pattern, flags)` over one `rxCache` in
`regex.js` retires all four (a native binary is one language, so a shared cache
cannot mix dialects). 32 lines to 8, four call-sites to update, and after this
section it costs nothing.

### THE DEFECT: `long == double` was false in BOTH COMPILER ENGINES, in TWO
### languages, while both INTERPRETER halves were right

Merging java's and csharp's strict equality is what found it.

```
                       long == double   double == long   long != double
JLS 15.21.1 / real java 24.0.2   true         true            false
ECMA-334 12.12.9 (C#, no toolchain)  true     true            false
llvm.Run (the Go twin)          FALSE        FALSE            TRUE
native -exe (layer 2)           FALSE        FALSE            TRUE
both interpreter halves         true         true            false
```

`floEq` is the floor's `jf_num_eq` and the twin's `jvmNumEq`, and NEITHER has a
tag 13 arm - a boxed double against a sized integer answered false. A comment at
**three** sites called that deliberate ("matched rather than smoothed"), which is
how it survived; what it actually matched was the other engine's copy of the same
mistake. `--cross` could not report it because nothing in 5,954 assertions had
ever compared a long to a double.

Both halves of `==` promote first, so the fix is in the LANGUAGE layer and not in
the primitive:

- `lib/runtime-jvm.metajs`'s `rtjStrictEq` reads both operands with `rtjNum` when
  a sized integer meets a boxed double - the same reading `rtjArith` already
  gives that very pair of operands.
- `abnf/jsrtjvm.go`'s `jvmNumEq` gains the `jsGInt` arm (with the unsigned-64
  path `giFloat` documents). `giEq` in `jsrtint.go:283` already read the pair
  this way and `rt.strictEq` short-circuited past it.
- `floEq`'s host binding in `jsrtjvm.go` now rejects a `jsGInt` EXPLICITLY, so
  the primitive stays byte-identical to the floor's `jf_num_eq`, which is
  untouched. `tests/metajs-test-full.js`'s `flo38` pins that contract and says
  which layer promotes.

**Pinned by six assertions, and all six DISCRIMINATE**: `flt10e`/`flt10f`/`flt10g`
in `tests/java-test-full.java` and `flt11a`/`flt11b`/`flt11c` in
`tests/csharp-test-full.cs` fail in the Go twin AND in the native `-exe` from a
clean `git archive 702dc68` tree, and pass in both interpreters. Real
`javac`/`java` 24.0.2 runs the java ratchet: **291 checks, 0 failures**. A 259-row
java differential probe (doubles x doubles, longs x doubles, ints x chars, chars
x doubles, read out of arrays so the folder cannot fold them) is now byte-identical
across `llvm.Run`, the native binary, the interpreter and real java; before the
fix the two compiler engines were wrong on 2 rows. dart and go were probed for the
same shape and are correct.

### Merged 1: `lib/runtime-jvm.metajs` - csharp + java, TWO knobs, six bodies

The difference list, taken before merging:

```
csNum / jvNum              TWO differences, BOTH IDENTITY. csIsChar and jvIsChar
                           are each rtIsCharBox(v); jpStrNum's whole body is
                           `var t = parseFloat(s); return t`, i.e. csharp's
                           parseFloat. The third pass recorded this pair as
                           "2 hooks"; the code has ZERO. jpStrNum is deleted.
csIsIntegral/jvIsIntegral  ZERO differences, same reason.
js_arr_new_n               ZERO differences once csNum == jvNum.
csStrictEq / jvStrictEq    ONE: csCmp vs jvCmp.
js_cscmp / js_jvcmp        ONE: the same csCmp vs jvCmp.
csFloArith / jvmArith      ONE: csFloOperand vs jvmOperand.
```

Two knobs, and both are REAL. `rtjkCmp` - csCmp promotes with `csPromote` and
compares two `csBox` cells with `sintCmp`, while jvCmp switches on `jvW` (JLS
5.6.2) and compares two plain int32s below 64 bits: two integer models, not one
spelled twice. `rtjkFloOperand` - csharp converts only the char box and lets a
tag 13 cell through (a C# long is an honest signed cell), java sends everything
non-float through `jvNum` (its long box is marked unsigned and the floor's
`to_number` would read it as a magnitude).

**Why merge fifty lines at all**: because `js_cscmp` and `js_jvcmp` are one body,
and this is the SECOND defect the pair has shipped at that one site. The NaN
sentinel was fixed in the C# copy in Part A and java's was still wrong months
later, in both engines, in two different directions; the sized-integer equality
above was wrong in both languages at once. One body cannot do that a third time.
The line count is not the argument and never was.

### Merged 2: `lib/runtime-dartswift.metajs` - dart + swift, ZERO knobs

```
dtIsArr / swIsArr        ONE apparent difference, dtIsObj vs swIsObj, and BOTH
                         are rtIsObj(v). ZERO knobs.
dtJavaStr / swJavaStr    ZERO differences, byte for byte.
```

Fourteen lines, both lists empty. php's `phIsList` is the same discrimination
plus a `{__refcell}` rejection - a third specification, not a knob - and stays
where it is. go's `goJavaStr` and js's `jxToStr` share the shape only at the
permissive threshold and are two-line delegations to a different renderer.

### Declined, each on a property of the code

- **`luChar` (lua) / `js_char` (swift)** - byte-identical, and still declined:
  the shared body could not be called `js_char`, because `lib/runtime.metajs`
  already exports that name for a DIFFERENT contract (the `{__char}` box, where
  swift's answers a one-character STRING), and `lua-rt.metajs` imports nothing at
  all today. Five lines is not worth lua's first import. **Both sites now carry a
  cross-reference** naming the other copy, which is the part that actually
  addresses the risk.
- **Dart's `dtJsCompare`** - unchanged from the third pass and still a genuine
  block: a third coercion (`dtNum`) on a path `js_dartlt`/`le`/`gt`/`ge` reach
  only for NON-number operands, i.e. for programs Dart rejects, and no `dart`
  toolchain exists here to settle which coercion is right.
- **The 4-way regex memo** (`jxGetProg` / `k5Get` / `pyERxGet` / `rbRxGet`, 24
  recoverable) and the kotlin+ruby regex trio - unchanged, and this round they
  are also **another agent's files**. Reported, not edited. Their home is
  `lib/regex.js`, and the three-place edit (`regex.js`, `abnf/jsrtregex.go`'s
  line-by-line port, kotlin's verbatim copy) is the standing cost.
- **`dtStrLen` / `k2RuneLen`**, **`pyIsObj` / `rbIsObj`**, **java/kotlin
  `js_jband`** - each needs a file another agent owns this round.

### The 21 duplicated extern NAMES, re-judged - and the name view retired

24 `js_*` names are defined in more than one layer-2 file. **Only two of them
(`js_jband`, `js_char`) have bodies that the shape scan groups at all**, at any
threshold. The other 22 are name collisions over genuinely different code, and
`js_pylen` is the clearest illustration: its five copies are 7, 9, 11, 16 and 7
lines, and python's begins with a `__len__` dunder lookup that no other language
has. `js_mcall`'s six are 45, 1, 73, 4, 15 and 51 lines.

**So the extern-name table should not be used to plan merges again.** It
understated the duplication (which is what the third pass found) AND it
overstates it (which is what this one found): it is measuring names, and the
thing being merged is bodies. The shape scan is the measure; the residual above
is the number.

### Where things stand after this pass

```
                        3rd pass    now   delta   (code lines, comments/blanks excluded)
csharp-rt                    772    728     -44
java-rt                      659    611     -48
dart-rt                     1139   1126     -13
swift-rt                     765    752     -13
runtime-jvm.metajs             0     51     +51
runtime-dartswift.metajs       0     14     +14
                                          -----
                                            -53     65 shared lines replace 118
```

Gates, all re-run in a tree built from `git archive 702dc68` plus only this
pass's files: matrix **329/329** - `--full` **5,960 assertions** (5,954 + the six
new ratchets), **0 languages whose halves disagree**, no
`BUT -frozen`/`VACUOUS`/`MISMATCH`/`FROZEN-DIFF` - `--cross` **119/0** -
clang-check **16/16, all sixteen "ok, and the clang executable agrees", none
held** - `tests/native-full.sh` **15/15, every binary exit 0 with 0 failures** -
`go test ./abnf/` ok - all fifteen `gen-*.sh --check` clean - `-freeze` a fixed
point - `MEC_GC=off` **3,711 B/iter** at 400k with gc RSS flat at 3.34 MB. The
dart and swift native ratchet outputs are byte-identical to `702dc68`'s; csharp's
and java's differ only in the six new assertions.

---

# Part B - a smaller C floor

## THE DOOR IS OPEN: MetaJS has a layer 2 (2026-08-04)

`runtime-next-plan.md` ("Part B, move 1: `case_map` CANNOT MOVE") ended Part B before it
started: **every** remaining candidate is reached through the C floor's host-builtin
dispatch, that dispatch **is MetaJS's standard library**, and
`languages/metajs-to-llvm-ir.abnf` linked `rts = [runtimePath()]` — the C floor and
nothing else. A body that left `runtime.c` for `lib/runtime.metajs` arrived in fifteen
languages' `-exe` builds and **vanished from MetaJS's**. Part B's movable set was empty.

It is not empty now. **`languages/lib/metajs-rt.metajs` exists** and
`metajs-to-llvm-ir.abnf` links `[libPath("runtime.ll"), libPath("metajs-rt.ll")]`, exactly
as the other fifteen grammars do. Three pieces, all of them the shape the other fifteen
already had:

- `lib/metajs-rt.metajs` — deliberately THIN: `import "./runtime.metajs"` and no export of
  its own. MetaJS's semantics are the generic `js_*` externs and the floor answers all of
  them; what the import buys is the DOOR, and `-rt-lib`'s export scan already reads
  imported files (the `js_jadd` link bug that widened it).
- `tests/gen-metajs-rt-ll.sh`, the fifteenth generator, with the mandatory `--check`. It
  carries a warning the other fourteen do not need: **the grammar that compiles this file
  is the grammar that links its output**, so a change to `metajs-to-llvm-ir.abnf` can make
  `metajs-rt.ll` stale on its own.
- `emitDispatch(dname, chainExt)` — js-to-llvm-ir.abnf's chained function table, verbatim.
  It is asked for **only when the default runtime list is linked**: with an explicit `-rt`
  (`tests/metajs-link-stubrt.c`) there is no layer-2 module, and that build keeps the
  single-table module it always had, byte for byte. The ext declaration rides in a
  one-element array rather than in a local reassigned from `undefined`, because MetaJS is
  typed and `-frozen` is where such a rewrite dies while goja stays green.

**Cost of the door alone**, everything else unchanged — `tests/metajs-test-full.js -exe`
**474K → 513K** (+8%), `metajs-bench-try` 20k iterations **36.5 → 37.5 ms/process** and a
warm hello-world **1.9 → 1.9 ms** (`jsrtlib_boot` is lazy and no MetaJS program calls a
layer-2 extern yet, so the runtime cost is zero by construction), `MEC_GC=off`
**3,711 B/iter unchanged**, RSS flat at 3,309,568. `-freeze` is a fixed point
(`jsagrammar.go` carries the new driver; `jsbootstrap.ll` byte-identical). Suite: matrix
**325/325**, `--full` **5,834 assertions, 0 languages whose halves disagree** and no
`BUT -frozen`/`VACUOUS`/`MISMATCH`/`FROZEN-DIFF`, `--cross` **119/0**, clang-check
**16/16 all agreeing**, all fifteen `gen-*.sh --check` clean.

## Move 1, `case_map` — EXECUTED IN FULL, MEASURED, AND **REVERTED**. It is a LOSS,
## and the number that says so also unblocks the rest of Part B (2026-08-04)

The move was landed whole this time — all four files together, plus a fifth nobody
had named — every gate in the Part B list was met, and it was then **reverted**,
because the two benchmarks that had never been pointed at it say it costs between
**65x and 130x** on any program that uppercases text. The plan's own criterion is
"revert anything that costs more than a few percent and record the measurement at
the site". This is that record.

### What was built, and it all worked

`runtime.c` 5,238 → 5,134 (−104: five `static long[328]` tables, the search,
`NCASE`) · `runtime.ll` 48,487 → **43,408** (−5,079) · `runtime.metajs` 605 → 750
(the 328 ranges + a 20-line binary search) · `js-rt.metajs` and `lua-rt.metajs`
+1 import each. The floor half was the three lines the previous write-up predicted,
including the two prototypes (`long to_number(long h);` and `long mk_bool(int b);`)
without which the floor emits an i32 return and the module fails to verify with
`store operands are not compatible: src=i32; dst=i64*`.

**Every gate passed.** matrix **325/325** · `--full` **5,834 assertions, 0
languages whose halves disagree**, no `BUT -frozen`/`VACUOUS`/`MISMATCH`/
`FROZEN-DIFF` · `--cross` **119/0** · clang-check **16/16, all sixteen agreeing,
none held** · `go test ./abnf/` ok · all fifteen `gen-*.sh --check` clean ·
`-freeze` a fixed point · `MEC_GC=off` **3,709–3,713 B/iter** with gc RSS flat at
3.37 MB over 250k/500k/1M/2M · `coro-poc/build.sh` `--gc` byte-identical in all
four collector modes and `--break` line-for-line identical to `33923d4`.
`tests/metajs-test-full.js -exe` **513K → 589K**, still 557 checks, 0 failures,
with `toUpperCase`/`toLowerCase` running a MetaJS binary search called from the C
floor. A 21-code-point probe (Latin, Greek, Cyrillic, İ/ı, ǅ/ǈ, ẞ, ᾈ, Ⅰ/ⅰ,
Deseret) gave byte-identical answers from `llvm.Run` and the native binary.

### A FIFTH file, which the four-file finding did not reach

`tests/coro-poc/build.sh` links `tests/coro-poc/gen.ll` against the C floor **with
no layer 2 at all** — gen.ll *is* the module — so the upcall broke it with
`Undefined symbols: _js_case_map, referenced from _case_map`. `./test.sh`,
`--cross` and even `clang-check.sh` are all blind to that; only the coro PoC sees
it. It is fixable in one stanza (a `define i64 @js_case_map` identity stub in
gen.ll, which is exactly what `tests/metajs-link-stubrt.c` does for the link test),
but **every future floor → layer-2 upcall touches five files, not four.**

### The measurement that killed it

The step-0 instrumentation had said `case_map` was cold: 114 corpus calls, zero on
`bench-alloc.sh` and zero on `metajs-bench-try.js`. That is a statement about the
BENCHMARKS, not about the function. Two programs that do what `case_map` is for:

| program (native `-exe`, best of 3) | floor (`33923d4`) | layer 2 | factor |
|---|---|---|---|
| metajs: 1M × `"a".toUpperCase()` | **0.23 s** | 3.46 s | **15x** |
| metajs: 200k × up+low of a 17-char mixed-script string (6.8M mappings) | **0.24 s** | 32.40 s | **135x** |
| lua: 400k `s = s + i % 7` (never uppercases) | 0.93 s | 1.82 s | **2.0x** |

The lua row is the one that shows the second, independent cost. It has nothing to
do with calling `case_map` — that program never does. Five top-level array
literals are ~1,640 boxed number cells **plus 321 PINNED interned constants**
(`lib/compile-core.js:254` pins every distinct numeric constant outside
[-256, 1024] into a module global that `js_gc_pin` roots for ever), all
permanently live, and the collector walks them on every one of the run's 1,378
collections: live bytes 93,760 → 147,632, `pinned` 13 → 334, heap 9.44 → 12.62 MB.
Putting the five literals behind a lazy `rtCaseInit()` removes that row completely
(0.88 s, live 97,312, pinned 13, heap unchanged) and does nothing for the other
two.

**THE GENERAL FINDING, and it is the useful part: the floor → layer-2 CALL is
CHEAP; the MetaJS BODY is not.** Decomposed with an identity upcall — the same
four-file move, same prototypes, same imports, but with
`function js_case_map(c, up) { if (up) { return c } return c }` as the whole of
layer 2's half:

| 1M `"a".toUpperCase()` | s | delta / call |
|---|---|---|
| C floor: static tables + binary search | 0.23 | — |
| upcall + IDENTITY layer-2 body | **0.36** | **+0.12 µs** |
| upcall + the real MetaJS binary search | 3.46 | +3.23 µs |

So the shim, `jsrtlib_boot`, `js_arr_new`, `js_call` and the `jsdispatch` compare
chain together cost **120 ns**, which is affordable. The nine iterations of a
binary search over BOXED ARRAYS cost **3.1 µs — 26x the entire call** — about
345 ns per loop iteration, because every `t[mid]` is a handle op and every
comparison is a boxed compare into a collected arena.

### What this means for the rest of Part B

The criterion in "The criterion, and the counter-pressure" below is now sharper
and can be applied before writing any code:

- **Data-structure work does not move.** A body whose cost is indexing a table or
  walking an array pays ~345 ns per element touched, against a few ns in C, and
  its table pays a permanent GC tax in all thirteen layer-2 modules. `case_map`
  and the Unicode ranges are the worked example; they are hereby on the
  can-never-move list next to the `d_*` family.
- **The IR bill is the other half of it, and it is per-language.** −5,079 lines of
  `runtime.ll` bought +12,400 lines in EACH of the thirteen `<lang>-rt.ll`
  (metajs-rt 3,048 → 12,622, js-rt 59,285 → 71,798, lua-rt 14,115 → 26,621,
  kotlin-rt 112,366 → 121,993, …). Net **+156,000 lines of checked-in IR to delete
  104 lines of C.** Any candidate carrying a table pays this shape.
- **A small, arithmetic, genuinely cold body could still move**, because 120 ns of
  call overhead is not what stops it. That is the honest remaining opening, and it
  is narrower than the candidate list below implies: `fmt_apply` / `fmt_sprint` /
  `fmt_top` / `fmt_val` and `js_num_str` are reached by **every print**, so each of
  them needs a print-loop benchmark of its own before, not after.

**Part B has still moved no body, and this is now a measured verdict rather than a
structural block.** The door (`lib/metajs-rt.metajs`, the fifteenth generator) is
open and correct and cost 474K → 513K on `metajs-test-full -exe` with zero runtime
cost; what is missing is a candidate whose MetaJS body is cheap enough.

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

## Part B - was BLOCKED; the block is GONE and the answer changed (2026-08-04)

> **STRUCK, and kept for the argument.** The door landed in `33923d4`
> (`lib/metajs-rt.metajs` + `tests/gen-metajs-rt-ll.sh` + `metajs-to-llvm-ir.abnf`
> linking two runtime inputs), so "MetaJS has no layer 2" is no longer true and the
> movable set is no longer empty for structural reasons. `case_map` was then moved
> for real and REVERTED on measurement - see "Move 1, `case_map` - EXECUTED IN FULL,
> MEASURED, AND REVERTED" above. The sentence below that has NOT aged is
> "**it cannot move**": the conclusion survived, with a completely different reason
> and a number attached.

Step 0 was done: the floor was instrumented and the hot/cold split MEASURED, which
corrected the candidate list above in one place and put the whole `d_*` family on a
can-never-move list. `case_map` came out as the best first move - 114 corpus calls,
zero on both benches, pure table lookup.

**It cannot move, and neither can anything else on the list.**
`languages/lib/runtime.metajs` is layer 2, and MetaJS itself has no layer 2:
`metajs-to-llvm-ir.abnf` links exactly one runtime input (`rts = [runtimePath()]`), so
a body that leaves the floor for `runtime.metajs` disappears from MetaJS's own native
build - and every remaining candidate is reached through the host-builtin dispatch that
IS MetaJS's standard library. **Part B's movable set is currently empty.**

The unblock is a `languages/lib/metajs-rt.metajs` plus a fifteenth `tests/gen-*.sh`.
**Re-checked 2026-08-04: neither exists at `c1bc760`, and both are UNTRACKED in the
working tree** - the door is being built. Opening it does not by itself close this:
the finding is that the movable set is empty, and that closes when a body actually
moves, with the before/after table the gate above asks for. Full write-up, with the
per-body call counts: `runtime-next-plan.md`, "Part B, move 1: `case_map` CANNOT
MOVE" and "What would unblock it, and who owns it".

---

# Why this makes the Go twin MORE valuable, not less

The tempting conclusion — "once layer 2 is merged we have something like the old Go
runtime, so delete `abnf/jsrt*.go`" — is exactly backwards, and it is worth writing down
because it will be proposed again.

**Today there are eleven independent readings** of the shared semantics: eleven files,
written by eleven separate processes from `abnf/jsrt*.go`. That independence is why the
defects above exist to be found - the C# NaN sentinel, which the two halves already
disagreed on at HEAD and which `--cross` never probed, is the concrete case. (The
`js_jadd` "divergence" this sentence originally cited was not one; the argument does not
depend on it.)

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
