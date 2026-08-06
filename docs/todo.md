# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Rebuilt 2026-08-05 at `e284185`; **eleven items closed on 2026-08-06**, **ten on
2026-08-07**, **ten on 2026-08-08**, and **ten more in the round `483a7a5`
through `af16f71`**, with what those agents found on the way folded in.
**Chapter 1 is entirely new again**: the old one is closed.

Two conventions, and they are the point of the file:

> **[V]** — a probe or grep was run for this list, on this commit.
> **[U]** — inherited from an older note and NOT re-checked. A lead, not a fact.
>
> **Re-verify before you act.** Of the ten items worked on 2026-08-06, **six of
> ten were wrong**. On 2026-08-07 the rate held. On 2026-08-08 the items were
> finally accurate and the failure mode moved to the WORK instead — two of ten
> agents shipped a NEW halves divergence that their own probes and all seven gates
> called green.
>
> **On the latest round the items were mostly right and STILL not sufficient.**
> Two were wrong as written — **5.2 said `floExt` was dead code when it is live,
> and deleting it as written would have introduced a defect**; and 1.9 named one
> `is` residue where every integral and floating name was broken. And **seven of
> the ten turned up a further defect the item did not name**, five of them live
> halves divergences `--cross` had never reached: csharp's `1 is char`, kotlin's
> `1L == 1.5`, go's silent `<nil> <nil>` comma-ok, java's anon-class field
> shadowing in the *compiler* half, and js's `next(v)` not reaching a delegate.
> **Treat an item as a place to start probing, never as a specification.**
>
> **Item NUMBERS are re-assigned every time this file is rebuilt.** Code comments
> of the form `docs/todo.md 1.4` are historical provenance pointing at the
> numbering of the day they were written — do not "fix" them, and do not trust
> them to name today's item.

Baseline every item must preserve (`tests/gates.sh`, ~2.5 min — add `--serial` on a
small machine):

```
matrix 351/351 · --full 6,873 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16 none held · native-full 15/15 · go test ok
gen-all 15/15 clean · -freeze a fixed point · bench: no row outside its spread
```

**Run `tests/gates.sh --bench` for anything that touches layer 2 or the floor.**
The seven correctness gates cannot see a slowdown — a merge in `120ba0f` cost
python **41%** and java **5.7%** for four commits with every gate green.
`tests/bench.sh` now takes a LOCK (`ed7086f`): its `-exe` directory is fixed so two
checkouts draw from the same paths, which means two concurrent runs used to
overwrite each other's binaries and report a phantom `BUILD FAILED`.

**On a loaded machine pass `--timeout 600` to `./test.sh --full`.** The js
compiler half is ~19 s idle and **82 s at load 16** against a 120 s default. This
fired repeatedly with four agents running; a `RUN-FAILED … exit 124` in a language
your change does not touch is this, not a divergence — re-run before believing it.

---

# 1. Correctness, with an oracle on this machine

These change answers real programs give. Each has a toolchain here that settles it.
**Section 1 was emptied again on 2026-08-07** — all nine items it held are in
chapter 9. What follows is what closing them turned up, so all of it is **[V]** on
`af16f71`.

### 1.1 swift/csharp: only the INITIAL value adopts a declared type **[V]**
`483a7a5` and `1b54644` closed parameters, returns, fields/properties and `[T]`
element types in both languages. **A write after the declaration still does not
adopt**: `var d: Double = 0; d = 3` and `c.D = 3` answer `1`. So do closure
parameter and return types, dictionary/`List<T>`/`Dictionary<K,V>` value types,
tuple element types, and nested `[[Double]]`. This is the one genuinely larger
piece — it needs the declared type remembered per variable and field at every
write site, i.e. a var-type table in both engines of both languages. Both ratchets
carry a block comment listing the open sites so the item is not read as finished.
Oracles: `swiftc`; ECMA-334 10.2.3 for C# (no toolchain).

### 1.2 csharp: an unqualified write to an instance field ABORTS the compiler half **[V]**
```csharp
class B { public double D; public void Set(double d) { D = d; } }
   interpreter 3      compiler/native  ABORT: assignment to undeclared variable: D
```
`this.D = d` works. Pre-existing at `08e8fcb`, a live halves divergence, and
`--cross` has never reached it. This is the highest-value item in the chapter: an
abort on ordinary C#.

### 1.3 swift: a subclass with no `init` gets an empty synthesised memberwise init **[V]**
```swift
class Base { var n: Double; init(_ x: Double) {…}; func f() -> Double { n*2 } }
class Derived: Base {}
Derived(3.0).n  -> nil (swiftc 3.0)      Derived(3.0).f() -> 0.0 (swiftc 6.0)
```
The synthesised init shadows the inherited one. All three engines, pre-existing.
Oracle: `swiftc`.

### 1.4 ruby: a block is bound as a POSITIONAL argument by two parameter shapes **[V]**
`def deflt(x = 5); yield x; end; deflt { |v| v*2 }` → MRI `10`, our interpreter
`NaN`, our compiler **aborts** (`not a number in '*'`). Same for `(*xs)`. The
literal block is bound to the optional/splat parameter instead of staying the
block. Needs parameter binding fixed in `bindParams`/`emitParams` across four
engines. Oracle: `/usr/bin/ruby`.

### 1.5 ruby: `return` inside a block does not return from the enclosing method **[V]**
`def u; A.each { |v| return v*100 if v == 2 }; nil; end` → MRI and our interpreter
`200`, **both compiled halves `nil`**. A live halves divergence, pre-existing, and
`--cross` has never reached it.

### 1.6 ruby: `Hash` has almost no Enumerable surface **[V]**
`map merge dup select sort_by delete any? each_with_object` are all
"unknown Hash method" in every engine. Note for whoever takes it: MRI's
`merge`/`dup` **carry the receiver's default over**, which `f64f86f` just gave
`Hash` — so this item and Hash defaults interact. Blockless `map`/`select` still
crash where MRI answers an Enumerator; that needs the arity model from 1.4, so do
1.4 first.

### 1.7 python: `d[missing]` raises an UNCATCHABLE error in the interpreter **[V]**
`try: d[3] / except KeyError` works in CPython and in both compiled halves; the
interpreter aborts the process. A live halves divergence. Related and separate:
`repr(KeyError)` is wrong in all three engines (`KeyError('3')` where CPython says
`KeyError(3)`) because the exception carries the already-`repr`'d string, which
makes `str(e)` right and `repr(e)` doubly quoted.

### 1.8 goja's shortest-form printing is still trusted by four interpreters **[V]**
`08e8fcb` fixed java and recorded the mechanism: goja's `Number→String` is not
always the shortest round-tripping form (22 of 199,904 doubles; some subnormals
emit a non-digit character), so **any grammar taking digits from `"" + n` has a
live goja-vs-`-frozen` divergence**. Still doing it: **lua** (`luaGStr` picks P
from `luaSigCount("" + n)` — 1 row of 676 wrong against `lua` 5.5.0 *and* a
frozen divergence), **kotlin**, **csharp**, **go**. Their `*-rt.metajs` copies are
safe; it is the interpreter halves. The fix shape is java's: choose P by
round-trip, not by counting the host's digits. Their comments also now assert
something false about `floPrec`'s two implementations, which `08e8fcb` unified.

### 1.9 go: four residues found by the multi-assign probe **[V]**
All confirmed against `go` at `4f77621`:
- **out-of-range index** `xs[7]` prints `<nil>` and exits 0 in the interpreter,
  errors in the compiler, panics in `go`. A halves divergence.
- **a missing map key does not yield a nil map**: `nst["absent"]["k"]` is rejected
  by both halves where Go gives the zero value.
- **`for v, ok := m["a"]; ok; …`** in a for-init: the interpreter says
  `unknown name: v`, the compiler is correct. A halves divergence.
- **nested composite literals** `map[string]map[string]int{"o": {…}}` — the
  interpreter accepts, the compiler says `composite literal element of
  unsupported type`. A halves divergence.

### 1.10 js: `g.return()` at a `yield*` does not close the delegate **[V]**
`af16f71` closed throw-forwarding and `next(v)` forwarding. `return()` remains:
node runs the delegate's `finally`, we do not. It needs the emitted `yield*` loop
to survive the `GEN_EXIT` sentinel `gen_close` throws in — a `finally`-shaped
region in the emitter — and the interpreter's `return()` only sets a done flag and
cannot run a delegate's finalisers at all. A different mechanism from 1.6 and
materially bigger. Oracle: `node`.

Also open and stated in all four grammars' `:description`: **a plain (sync)
generator has no `.throw()`**, so a sync generator delegated to from an async one
takes the no-throw-method path where node forwards. The blocker is that the
compiled generator is the C floor's tag 15 with a host-id dispatched member table,
and `*jsGenerator` in `abnf/jsrt.go` is shared by every language exposing a
generator.

---

# 2. Cross-engine defects and latent traps

Small, cheap, and each one is a divergence waiting for the first program that
reaches it.

### 2.1 Closing an iterator on `throw` is BLOCKED, and the reason is worth reading **[V]**
`ae0c62b` closes on `break`, `return`, `continue L` and a labeled break to an outer
statement. **`throw` was built, passed every probe, and was reverted** — and the
mechanism that killed it rules out the obvious retry.

A per-loop iterator depth stack (js-only, so the fifteen-language `retStmt` risk
never applied) with save/unwind around every `js_try` was byte-identical to node on
nine throw shapes. Then the ratchet failed: **a depth-based stack cannot survive
coroutines, because `for await` suspends INSIDE its own for-of.** A
`try { await Promise.reject(...) } catch` recorded depth 0, suspended, and its catch
later unwound past a different suspended `for await`, whose loop then answered `1`
instead of `123`. Reduced to a two-function repro. A wrong close is a silently wrong
answer, so the whole mechanism came out; `itc20`/`itc24` pin the gap with the reason
beside them. **Any future attempt needs a per-coroutine stack, not a per-program
one.**

### 2.2 `-rdynamic` will break the first Linux native build that uses a generator **[V]**
`gen_create` finds `coro_entry` via `dlsym` with no link flag, which works on
darwin; linux needs `-rdynamic` on the clang line in `abnf/llvmlink.go`. Nothing
here builds on linux to prove it, and there is no `rdynamic` anywhere in the repo.

### 2.3 A Python `set` member that is unhashable **[U]**
Compared with `==` in the compiled halves and `===` in the interpreter
(`a=[1]; b=[1]; {a,b}` is 1 vs 2); CPython raises `TypeError`. Closing it means
changing the shared `dictFind` contract, not Python.

### 2.4 `i in arr` after `delete arr[i]` **[V]**
`true` here, `false` in node. **Declined twice with the same finding**: a real hole
cannot be expressed without changing the shared `*jsArray` in `abnf/jsrt.go`, which
is the array type for *every* language, and a sentinel would need filtering at ~60
read sites where one miss leaks garbage into `join()`. Listed so the third person to
find it stops here.

### 2.5 JS constructs that still abort **[V]** for `super.x`, **[U]** for the rest
`super.x` as a *value* now yields `undefined` rather than aborting. Still aborting:
`super.b = 1` as an assignment target, `new a.b.C()`, nested `new`, `new C` with no
argument list, `class X extends <expression>`, `for (a.b of xs)`, `export`, `with`.
Also `Object.prototype.toString.call(p)` in the compiler halves, and
`typeof instance.method` is `undefined` in both halves (methods live on the
`__class` descriptor, not as own properties). The interpreter additionally lacks a
destructuring `catch` binding — blocked by `excCatch` binding exactly one name.

### 2.6 Interpreter generators replay instead of suspending — js AND python **[V]**
Every `next()` re-runs the body from the top, replaying recorded sends and stopping
at the next yield via a thrown signal. Side effects repeat, it is O(n²), and
`.throw()` and `Symbol.iterator` are missing.

**Python's interpreter half has the same design** (`genStep`), and it is why the
interpreter prints a generator's `finally` on replay's schedule rather than
CPython's — measured identical before and after `d7ef11a`, so it is this item.
`2339678` made it structural for async too: an async body with a side effect
*before* an await repeats it on every resume, documented in all four grammar
headers. **Ordering is right in every engine; only repetition differs.**

**If genuine suspension is unreachable** — the `-frozen` engine exposes no
goroutines and both engines must agree byte-for-byte — **documenting the limitation
in each grammar's `:description` is a valid closure**, and `d7ef11a`/`2339678`/
`ae0c62b` have already done it for their own corners.

### 2.7 The BigInt mix `TypeError` is raised but not catchable **[V]**
`1n + 1` correctly reports *"Cannot mix BigInt and other types"*, but it aborts
rather than being caught by a JS `try`/`catch`. Same family as item 3.2.

### 2.8 go: imported packages share the main file's globals **[V]**
With `mconst.A = 0.1` and a main-file `const A = 5`, `mconst.Sum()` returned 11 at
base. `08e8d3f` fixed it for CONSTANTS (each file gets its own scope stack) but the
underlying runtime global-namespace sharing is still there for vars and funcs. Also:
an imported package's exported CONSTS are not put on the package object (`mconst.S`
is `<nil>`); `var a [size]int` with a named const size gives `len(a) == 1` in the
interpreter; and the compiler half cannot parse `const a, b = 0.1, 0.2`.

### 2.9 Ruby and python residue **[V]**
- **A bare `except:` will not catch python's `GeneratorExit`** — `d7ef11a` routes
  the close sentinel past every catch arm so the two engines agree; CPython lets
  `except:` catch it. Documented at the `js_try` site.
- **An abandoned python generator never runs its `finally`** — CPython finalizes at
  collection; we have no finalizer hook and the GC deliberately has none.
- **`d.items()` renders pairs as lists**, the same tuple root cause as 3.1.
- **`.send()` works on an iterator object** where CPython raises `AttributeError`.
- **ruby `Integer#fdiv` on a bignum, and a big meeting a Rational or Complex**,
  promote through the double where MRI is exact. That needs a Rational whose parts
  are bigs — a second value model, deliberately not started.

### 2.10 Six dead helpers in `python-rt.metajs` **[V]**
`pyFStrictEq pyFTruthy pyFClampSub pyFToInt pyFCall1 pyFWrap32`, plus `pyFBigEq`/
`pyFBigIsZero`: ports of the arms `6eec533` deleted, with one comment describing
callers that no longer exist. Left rather than risk a late unverified edit;
deleting them shrinks layer 2 and is a clean follow-up.

---

### 2.11 A boxed value knows its KIND but not its declared class **[V]**
Two boxed values of the same primitive kind and different declared classes cannot
be told apart — this value model has one representation per kind, not per box.
Surfaced by `44dbd04` while routing java's `equals`, and stated in both
`:description` blocks rather than guessed at. Same family as 1.8 (kotlin's mixed
Float/Double set key), and the two would likely be closed by one mechanism.

### 2.12 A `getattr()`-bound python builtin cannot carry keyword arguments **[V]**
`getattr(xs,"sort")(reverse=True)` differs from `xs.sort(reverse=True)`: a
signature-less builtin receives its keyword dict as a trailing POSITIONAL, and the
bound wrapper cannot tell that from a real argument. Documented at all three sites
and in both grammar headers by `51436f9`. Related and deliberate: `sorted()` still
refuses `key=`/`reverse=` loudly, because they are keyword-only in CPython and a
signature-less builtin here is positional-only — `list.sort` takes both only
because a METHOD call carries its `kw` dict.

### 2.13 Small unbound builtins met in passing **[V]**
java: `Integer.toHexString`, `Double.NaN`, `Boolean.valueOf`. swift:
`Float.greatestFiniteMagnitude` / `.leastNonzeroMagnitude` (neither does `Double`).
Each is a one-line binding; they are grouped because none is worth its own item.

# 3. Costed designs, deliberately not started

Each of these has been sized. None is a small change, and the sizing is the
expensive part that has already been paid for.

### 3.1 A real tuple type for Python — the blocker is `runtime.c:3088` **[V]**
`set_member` accepts a named key only on tag 6; on tag 5 (array) it demands a
numeric index and otherwise dies. So the cheap fix (mark the array with a `__tup`
property) is **not representable in the engine the native binaries run on**, and a
tuple must become a tag-6 BOX — the float box's ~640 lines again, zero in the C
floor.
**It is atomic**: any subset is a `--cross` divergence. Sites to classify:
`python-interpreter.abnf` 17 `Array.isArray` + 49 tuple refs, `python-rt.metajs`
42 `pyIsArr`, `python-to-llvm-ir.abnf` 54 tuple refs, `abnf/jsrt.go` 241
`*jsArray`.
What it unblocks: the `%`-formatting argument-arity heuristic disappears,
`BaseException.__str__` stops hand-building `('a', 1)` at its site, and
hashability becomes a real rule (today a list is a valid dict key).
**Copy the python FLOAT box, not the tag-14 floor box** — manual §7.6 says why,
and its two surprises (key aliasing needs two hops; `is` under-reads, so the box
must be a pointer) are the reusable half.

### 3.2 A catchable `NameError` under the compiler **[U]**
The site is `js_scope_get`, shared by all sixteen languages and pinned by
clang-check and the SHOULD-ABORT rows.

### 3.3 Part B: moving floor bodies into layer 2 has no landed move **[U]**
The floor→layer-2 upcall costs **120 ns** and is affordable; a nine-iteration
binary search over BOXED ARRAYS costs **3.2 µs, 26× the whole call**. `case_map`
moved whole, passed every gate, and was reverted at 15× / 135× / 2.0×.
Two mechanisms to carry: the 2.0× on a loop that never uppercases was **pinned
interned constants** (`lib/compile-core.js:254` pins every distinct numeric
constant outside `[-256, 1024]`; a lazy `rtCaseInit()` removes it), and the IR
bill is per-language — −5,079 lines of `runtime.ll` bought **+12,400 lines in each
of thirteen `<lang>-rt.ll`, i.e. +156,000 checked-in IR lines to delete 104 lines
of C**.
Candidates with corpus call counts: `fmt_apply` 6, `fmt_sprint` 0, `fmt_top` 53,
`fmt_val` 0, `js_num_str` 306, the `dec_*` family (`dec_mul` 35,491 — handle-level
but NOT cold). **`d_pow` and its family cannot move at all** (manual §7.7).
**The direction actually worth taking is not another leaf: it is the VARIADIC
PROLOGUE**, where `sample` on the spread-heaviest Go program is dominated by
`scope_get` and `ar_block` repacking args, still MetaJS in both builds.

### 3.4 Coroutines: option A (CPS in the emitter) is the eventual answer **[U]**
What shipped is B (pthread + condvar): **4.1 µs per `next()`** natively against
**0.75 µs** under `llvm.Run`, ~**18.8 KB per live suspended generator** (10,000 →
187.8 MB RSS), and an abandoned generator parks its thread forever.
**B1 (`swapcontext`) would be 6.3× faster and is dead**: deprecated on darwin, and
`runtime.c` has no `#include`, so `ucontext_t` fields would be hard-coded byte
offsets — a floor that guesses a struct layout is not a floor. Nothing is wasted
by having done B: the extern surface is identical.

---

# 4. Test coverage that is missing

The suite's blind spots, in the order they are likely to hurt.

### 4.1 The two MetaJS halves bind different host globals **[V]**
`metajs-interpreter.abnf` binds eleven; the compiler half's `standardJSBindings`
(and the C floor, which matches the *compiler*) also bind `Infinity NaN Array
Object byteLen sprint rawSet`. `tests/metajs-test-full.js` asserts only the
intersection, so nothing can see it. Same family: `"ß".toUpperCase()` is `"SS"` in
the interpreter (real JS) and `"ß"` in the compiler and floor (Go's simple
mapping).

### 4.2 Two `die("PROBE-C <<")` probes are wired into the shipping floor **[V]**
`runtime.c:2410` and `:2417`. They are the fatal probes that measured the
shift-result-type claim, which fired **zero** times. Document them at the site or
remove them; they currently read as leftover scaffolding.

---

# 5. Duplication and dead code

All small. All verified. The shape scan is the measure — `go run ./tools/shape-scan`.

### 5.1 The `-min 40` shape tier is untouched: 15 groups, 111 recoverable lines **[V]**

> **⚠ READ THIS BEFORE MERGING ANYTHING HERE. A ZERO-KNOB DIFFERENCE LIST IS
> NECESSARY AND NOT SUFFICIENT — SIZE THE CALL SITE, NOT THE BODY.** The fifth
> pass (`120ba0f`) merged bodies on the strength of an empty difference list alone
> and shipped a regression that stood for four commits: `pyIsObj` →
> `rtIsObjNotArr` → `rtIsObj` cost **python +41%**, `rbIsObj` cost **ruby +11%**,
> and `rtjIsIntegral` cost **java +5.7%** — because each is a predicate asked at
> ~15–30 sites inside an arithmetic or attribute path, and in MetaJS **a call is a
> scope frame**. All three were reverted to self-contained bodies in `a8e6aa2` /
> `34814cd` with the measurements at the sites.
> Sharing stayed free for kotlin's `ktIsInt` at the *same* body (−0.64%, inside a
> 3.05% spread), which is exactly the point: identical bodies, different call
> sites, and only the call sites decide. **Measure with `tests/gates.sh --bench`
> before and after — the seven correctness gates cannot see a slowdown.**

`-min 60` is finished at 2 groups, both deliberate declines (manual §7.12).
Note that the 4-line `*Arg` bodies below are the *same shape* as the ones that
regressed, so the intra-file merges are the safe half of this item and the
cross-language ones need the measurement.
**The most useful part is that much of this tier is INTRA-file duplication,
mergeable with no import and no shared-module decision**: `k1Arg`/`k2Arg`/`k3Arg`
(kotlin :1061, :2175, :2890), `jvArg`/`jxArg` (js :287, :3157),
`pyEArgAt`/`pyFArgAt` (python :5087, :5276) — one 4-line body, 8 copies; plus
`k1Integral`/`k1Numeric`/`k3Integral` and two nullish pairs. ~32 lines recoverable
inside single files.
Cross-language ones whose home is **already imported**, so they are free today:
`pyBRepeat`/`rbRepeat`, `pyIsNone`/`rbIsNil`, `pyTag`/`rbTag`, `csMaxI64`/`jlMaxL`.
Not free, and why, is manual §7.13: anything involving `js-rt.metajs` (imports
`regex.js` only) or `lua-rt.metajs` (imports nothing).

### 5.2 ~~`floExt` is unreachable~~ — **THE ITEM WAS WRONG. CLOSED `4f77621`.**
`floExt` is **live**: `emitTypedBin` reaches it whenever a STATIC float type is
known, so `float64(9) / x` emits `js_jfdiv` while `a[0]-a[1]` does not. Only the
`emitBinNum` **read** was dead, because `giBase` holds every key `floExt` has and
is consulted first. **Deleting the table, as this item implied, would have
introduced a defect.** Proved by construction and by fatal probe at both sites
(the dead one fired 0 times over the whole corpus, the live one fired); the two
dead lines are gone and both proofs are recorded at the site. Kept here as the
standing example of why this file says re-verify before you act.

### 5.3 The four `runtime-*.metajs` splits can be folded back **[V]**
`runtime-decimal` (188 lines), `runtime-bignum` (263), `runtime-jvm` (159),
`runtime-dartswift` (50) were split off **only** to keep the per-declaration
live-body tax off non-callers. The `scope_find` hash index removed that tax, so
they can go back into `runtime.metajs` whenever that is the tidier shape — a pure
tidy, no measurable cost either way. **This one really is free, and 5.1's warning
does not apply to it**: folding a module back moves DECLARATIONS, which the hash
index made free, and adds no call depth. It is the *delegation* that costs, not the
location.

### 5.4 `sw_kget`/`sw_kset`/`sw_safeget` are duplicated across three GRAMMARS **[V]**
`swift-to-llvm-ir.abnf:2495,2531`, `dart:2437,2451`, `kotlin:2555` (which says
"transferred verbatim" at the site). They are **emitter closures that build basic
blocks**, so `runtime.metajs` cannot hold them; merging means a shared ABNF module
imported by three grammars. Never started.

---

# 6. Simplifications now unblocked

Both of these were blocked by something that no longer blocks them.

### 6.1 Kotlin's nine scope helpers can come back out of the emitter **[V]**
`js_ktget`, `js_kset`, `js_ktvarset`, `js_ktmextget/set/call`, `js_ktouter`,
`js_ktrefbase`, `js_ktdthis`, `js_ktbareref` were lowered into ~250 lines of IR
(`kotlin-to-llvm-ir.abnf:2605+`) **only** because the floor had no scope-parent
accessor. That blocker is gone —
`scopeNew/scopeParent/scopeGet/scopeHas/scopeDecl` are host ids 64..68 in all
three engines — and **no emitter uses the new scope API yet**. The six lowered
probes (ruby `defined?`, php `isset`, swift/dart, kotlin's nine, python's
`py_setvar` residue) are all conversion targets.

### 6.2 Kotlin's three `:script` peek guards can become native `!"token"` **[V]**
`NotArrow`, `NotParen`, `NotColon` (`kotlin-interpreter.abnf:1278`ff, byte-
identical in the compiler half) are hand-written parse-time scripts that skip
whitespace via `c.peek` and return an impossible token. Since the dialect gained
`!'token'` negative lookahead (`abnf-of-abnf.abnf:61`, `NotToken`) all three are
pure "next token isn't X".
**`SameLine` must stay a `:script`** — native lookahead skips the very newlines it
inspects.
**Risk, stated by the original analysis**: it changes the *shared* grammar and
depends on `!"token"` matching the guards' whitespace-skipping exactly; a mismatch
silently changes what the grammar accepts, so it needs its own validated pass.

---

# 7. Tooling and developer experience

Ordered by how much time each would save. The first one is measured and ready.

### 7.1 Should `test.sh` default to 2× ncpu jobs? **[V]**
`test.sh` defaults `JOBS` to `sysctl -n hw.ncpu`, which does **not** saturate the
machine: `./test.sh` 35.0 s at 768% CPU, `-j 32` **25.4 s at 1143%**, same
329/329. The other two groups do not benefit and should keep the default
(`--full` 95 s → 91 s, tail-bound on ~16 ratchet files; `--cross` 12 s both ways).
Decide whether the matrix's default becomes `2 * ncpu` or whether `gates.sh`
passes `-j` for the matrix only, and confirm on a second machine that
oversubscription introduces no flakiness or memory pressure — the entries are
independent subprocesses, so the mechanism is the one already used at 16-way, but
this was a single measurement.

### 7.2 Re-measure lua's hash-index buy-backs with draws **[V]**
The `scope_find` hash index costs lua **~+4%** — real, re-measured twice (it
exceeds both rows' spreads, `c` as a control moved +0.02%, and a 5× longer loop
gives the same percentage, so it is steady-state loop cost). Its two proposed
remedies were measured at **+2.8% / +4.7% / +6.1% as SINGLE BUILDS**, against a
row whose own 13-draw range is **3.3%** — so **"both ways of buying it back are
worse" is not established.** Rebuild the two variants (out-of-line arm,
name-in-slot) and measure with `tests/bench.sh --draws 9` or more, reporting
median, range and overlap for lua **and** for kotlin/python. The open question is
whether lua's ~4% can be bought back without giving up the −19.5%..−42.0% the
other nine languages get.
**One datum from the 2026-08-06 baseline re-record**: lua came in at +2.04%
against its own 2.05% spread with NO lua change anywhere in the round — recorded
in `tests/bench/baseline.txt` rather than smoothed. lua is the noisiest row
relative to its own size, so this item needs more draws than most.

### 7.3 Per-language coverage of the floor **[U]**
The instrumentation that found 24 unreached floor bodies was written once and
thrown away. Keeping it as `tools/floor-coverage` would make "is this body dead or
untested?" answerable at any time — **it is the question that gates Part B (3.3)**,
because a body no test reaches is a body no move can be validated against.

### 7.4 A `--why` flag for the emitters **[U]**
Understanding which extern an emitter chooses for an operator means reading
thousands of lines of grammar, repeatedly. A flag that prints the extern chosen
per AST node would make layer-2 work dramatically faster.

### 7.5 Make MetaJS's dialect gaps loud **[U]**
No exponent literal, no `toPrecision`, typed locals, ASI differences: each is
discovered by a confusing failure. A `-verify` pass over `lib/*.metajs` that names
them would pay for itself immediately. (`docs/abnf-dialect-gotchas.md` lists them;
nothing enforces them.)

### 7.6 Make the frozen-snapshot rule mechanical **[U]**
"Did you `-freeze` after touching the emitter?" is a question a script should ask.
`tests/gates.sh --freeze` checks it on demand; a pre-commit hook would remove the
class entirely.

### 7.7 Wire the shape scan into CI with a ratchet **[V]**
`go run ./tools/shape-scan -max 2` fails above two groups. Nothing runs it
automatically.

---

# 8. Reference-corpus gaps

Recorded 2026-07-27 and **not re-measured since — treat every number as [U]**.
Corpus: lua 100%, js 98.9, ruby 98.1, dart 96.6, typescript 93.7, python 92.9,
java 92.1, kotlin 91.0, bash 88.6, c 87.3, csharp 84.6, php 83.1, go 81.1,
swift 80.0, batch 0/2.

### 8.1 JS reject side — three coherent groups
**Assignment targets (8 files)**: a form that is not a LeftHandSideExpression used
as one — `() => {}()`, `({a:this}=0)`, `switch (c) { default: default: }`, a
statement inside a template substitution.

> **A warning that was paid for once.** The neighbouring `for(([a]) of 0)`,
> `for((1 + 1) in list)`, `1++` and `1--` files look like the same group and are
> **not**. Guards refusing a parenthesized non-target for-head and a numeric
> postfix operand won 7 reject files and cost **8 must-parse** files — `0++`,
> `for(([0]) in 0)`, `for((0) of 0)` under `early/`, which are early errors that
> must PARSE. Reverted. **Do not reintroduce without measuring both rows.**

**Goal symbol, Script vs Module (17 files)**: a `:script` guard cannot see the
file name — `abnf/parserscript.go` exposes only `getSrc`/`setSrc`/`getSdx`/
`setSdx`/`peek`, and `abnf/frozen.go` mirrors it. A Script may not contain
`import`/`export` at all (6 files); a Module is always strict, reserves `await`,
and forbids the Annex B `<!--` comments (8 `*.module.js` files). Needs a filename
or module-goal boolean exposed to guards in **both** engines, since the matrix
demands byte-identical output.

**Unicode identifiers (3 files)**: `IdUni`/`IdUniPart` approximate
ID_Start/ID_Continue with a few wide ranges, so `var 🀂` is wrongly accepted. Real
tables would also serve Java, C#, Kotlin, Swift and Python, so they belong in the
shared `languages/lib/ident.abnf` as an **additive** change — do not alter the
existing ASCII `Id` or `KwEnd`, which many grammars depend on. A compressed range
list checked by a `:script` guard is probably cheaper than thousands of ABNF
ranges.

### 8.2 Other corpora
- **batch 0 of 2** — the ratchet is FULL at 24 sections and 129 assertions, but
  neither wine file parses. No statistical weight; still a stark gap.
- **swift 80%** — an 18-file lexical tail: suppressed conformances in unusual
  positions, `@isolated(any)` in a type, `borrowing` as a bare type name, raw
  multi-line strings with 3+ pound levels, parameter packs (`each T`), form feed
  and NBSP. **For Swift, lexical work buys corpus and semantic work buys ratchet
  — they are nearly disjoint.**
- **dart 96.6%** — ~53 of the remaining 117 are multitests whose surviving `//#`
  variants the files themselves label "syntax error"; largely unreachable without
  multitest expansion.
- **python 92.9% and ruby 98.1%** are at their hard caps (files CPython itself
  rejects; non-UTF-8 fixtures).
- **go / php / csharp / java / c** have ordinary long tails.

---

# 9. Closed — do not re-open

## The ten closed in `483a7a5`..`af16f71` — ALL of chapter 1, plus 2.8 and 5.2

`+156 assertions` (6,717 → 6,873), six commits, `483a7a5`..`af16f71`.

- **1.1 swift + csharp, declared-type adoption** — `483a7a5`, `1b54644`. The item
  read as a Float nicety; it was giving **Double** wrong answers (`g(3)` → `1`)
  and sized-integer ones (`w(250)` → `260`). Four sites each, decided at PARSE
  time. swift also fixed an abort found on the way: `return (1, 2)` parses as the
  identifier `return` applied to a tuple.
- **1.2 + 1.4 ruby** — `f64f86f`. `yield` inside `begin`/`rescue` aborted the
  compiler half; the block is now published per method into its own scope and read
  through the scope chain. Hash defaults are honoured at the op-assign and append
  reads, which both bypassed `Hash#[]`.
- **1.3 python dict/set surface** — `f64f86f`. Eleven methods, not ten. Found:
  `{1,2} <= {2,3}` was **true** (two set objects fell into the host's relational
  operator, which renders both as `[object Object]`), and `getattr(d,"items")()`
  aborted in both compiled halves because `items` existed only as an emitter
  lowering.
- **1.5 + 2.8 java** — `08e8fcb`. The anon-class local, plus the mirror-image
  defect in the *compiler* half. `floPrec` now means exactly-*n* in both engines.
  The two printing defects had one root cause: **goja's `Number→String` is not
  always the shortest round-tripping form**, which was also a live
  goja-vs-`-frozen` divergence no matrix entry reached.
- **1.6 js/ts `yield*` throw-forwarding** — `af16f71`, in all four grammars, with
  `next(v)` forwarding fixed on the same line. The new state is per-generator-
  instance by construction, which is what 2.1's reverted mechanism lacked.
- **1.7 + 5.2 go** — `4f77621`. A PEG ordering hazard, not a go quirk: three
  two-name comma-ok rules are tried before `ShortDecl` and each matched the first
  element of a multi-assign. The probe found three more, including
  `v, ok := box.mp["k"]` answering **`<nil> <nil>` silently in both halves**.
- **1.8 kotlin mixed-width collection keys** — `1b54644`, fixed in the LANGUAGE
  layer so `strict_eq` stays byte-identical to the floor's. Found `1L == 1.5`
  true in the interpreter, which truncated the double into the integer's width.
- **1.9 csharp `is`** — `1b54644`. Every integral and floating name, not one.
  `decimal` was **declined on merit** (binary64 arithmetic under a
  `System.Decimal` label is more misleading than an honest double box) and
  `csFloStr`'s `1e-5` bound left as **not established** — ECMA-334 does not
  specify `ToString()` text at all.

**The near-miss worth more than any of the fixes**: an unprefixed `numText` added
to `lib/compile-core.js` silently replaced python's own, diverging python's halves
by 13,087 lines from a change about java floats. **No gate caught it** — only a
base-module diff did. Now in the manual, chapter 4.


## The ten closed on 2026-08-08 — ALL of chapter 1, plus 2.4 and 2.9

`+295 assertions` (6,422 → 6,717). Every item had a real oracle on this machine and
every one of them was **accurate as written** — the first round where the list was
not the problem.

- **1.1 `float` as a real binary32, the last three languages** — swift `f91058a`,
  go `5be2946`, csharp `418ab6f`, finishing what `8f43e84` (java) and `e5e68b5`
  (kotlin) began. Each needed its own renderer, exactly as the item predicted, and
  the shared mechanism proved out: go generalised `abnf/jsrtjvm.go` to ask the width
  through `jvmIs32` and the promotion DIRECTION through `jvmWidens32` (java and C#
  widen per JLS 5.6.2 / ECMA-334 12.4.7.3; Go's spec forbids the mixed pair), and
  csharp then cost exactly the predicted one byte plus one arm in each. **The item
  said `floCSF = 4`; it is 5, because `floGoF` took 4 first.** swift needed none of
  it — its Float rides style byte 3 with `jvmFround` reused, so the file was
  untouched. **The floor was measured to be the wrong home** and stayed that way.
- **1.2 / 1.3 python `hasattr`/`getattr` and the `list` surface** — `51436f9`. A
  string's methods are not in `pyMethodCall` at all (`pystrmethod.go` installs them
  by wrapping the extern), so a bound method must re-enter through the extern.
  Found on the way: **`pop(i) ignored its index`** in all three engines, and
  `hasattr(x,"length")` was True in the interpreter only.
- **1.4 ruby's Array surface** — `3267449`, and the second pass is the one to read.
  Sorting nested arrays aborted the interpreter while both compiled halves answered;
  copying their arm over revealed **two bugs they both already had and agreed on**.
- **1.5 / 1.6 / 2.9 java** — `44dbd04`. Default `toString`/`equals` on `js_jhash`'s
  model; `__outer` hops for inner and anonymous classes; JLS 6.4.2, a field obscures
  a type name. Exposed a live divergence: `((Object)1).equals(1L)` was **true** in
  both compiler engines.
- **1.7 go `-z` on a complex** — `5be2946`. Fixing it exposed a divergence in the
  complex PRODUCT, whose cause was that the interpreter's float `*` loses the zero
  sign; fixed with four IEEE-by-construction primitives rather than by patching the
  multiply, **because goja and the frozen engine are two implementations and only
  the multiply happened to differ today**. Also folded Go's rule that an untyped
  constant HAS NO SIGNED ZERO.
- **1.8 js `yield*` in async generators and `ag.throw()`** — `fb71770`. The residue
  was worse than "synchronous": an async generator's `next()` answers a promise, so
  the unawaited drive never terminated at all. The throw-in record travels **on the
  resume value**, which is what makes it per-coroutine by construction — the thing
  2.1 records as mandatory.
- **2.4 kotlin `Float.hashCode()` and `is`** — `30cd9f6`. Needed a binary32 BIT
  EXTRACTION where `e5e68b5` had written only the rounding. The call site worth
  naming is `ktElemHash`, which every collection fold reaches — fixing only the
  receiver method would have left `Arrays.asList(1.5f).hashCode()` wrong.

**The signed-zero trap appeared three more times** (swift, csharp, go's interpreter
`*`), taking it to **nine sightings**. `-x`, never `0 - x`.

## The ten closed on 2026-08-07

- **1.1 java unqualified field access** — `77eb804`. The item UNDERSTATED it:
  unqualified STATIC access and unqualified WRITES were broken too. The compiler
  half branches at runtime on `js_scope_typeof` so a local still wins; the
  interpreter hangs it on `core.varMiss`, consulted only after the whole scope
  chain misses, so the hit path costs nothing. **70 of 70 new assertions fail at
  the parent**, in both halves.
- **1.2 `float` as binary32, kotlin** — `e5e68b5`. Kotlin/JVM's `Float` IS java's
  by specification, so `java` is a real oracle for every row; 11,343 probe lines
  went 4,111/4,030/4,030 wrong → 0/0/0. **The floor was measured to be the wrong
  home for the width** and the reason is recorded at the site.
- **1.3 go named constants** — `08e8d3f`. A parse-time scope stack whose asymmetry
  is the design: *poison leaking outward is a decline; a const leaking outward is a
  wrong answer.* Also folded rune and complex constants. **The `convByPos`
  cross-file collision turned out to be real, not theoretical** — brute-forcing pad
  lengths made a division answer 3.5 in both halves at base.
- **1.4 python builtins** — `6eec533`. 24 added exact against CPython, plus
  `dict.pop`/`set.add` — which were missing from all THREE engines, not the
  interpreter as the item said. `input`/`format`/`frozenset`/`hash`/`id` declined
  with the reason at each binding site. Two live halves divergences found by the
  sweep and fixed (`isinstance(1, object)`, `set("aab")`).
- **1.5 `list.count`** — `6eec533`. Both compiled halves treated the argument as a
  KOTLIN PREDICATE. `str.count` was already correct including the non-overlapping
  rule — a null result, now a control.
- **1.6 ruby `Float#to_s`** — `4731755`. The rule was settled empirically on 3,000
  random doubles: `decpt < -3 || (decpt > 15 && decpt >= digits.length)` reproduces
  all 3,000; `decpt > 15` alone misses 297. **`rtFloStr` needed no change**, so the
  nine languages importing it were untouched.
- **1.7 `for await` and `async function*`** — `ae0c62b`. **The "genuinely blocked"
  verdict was false.** `js_yield` already is one suspension channel; it needed a
  tag saying what is travelling on it. Ordering byte-identical to node in three
  engines. Also fixed a halves divergence nobody had listed: `await` in an async
  CLASS METHOD was the identity in the compiler halves.
- **1.8 iterator close on `return` / outer labeled break** — `ae0c62b`, with no
  runtime stack (the emitter keeps the enclosing loops' SSA handles). Emitted
  modules for programs without those shapes are byte-identical to base. The `throw`
  case is item 2.1.
- **2.1 `record.hashCode()`** — `77eb804`. JLS 8.10.3 leaves the combination
  unspecified, so OpenJDK's `h*31` is pinned MEASURED against java 24.0.2, while
  every component type's own hash is exact and reproduced bit for bit.
- **2.4 twelve foreign method names on python receivers** — `6eec533`. Denial keyed
  by RECEIVER TYPE, not name — a flat name switch would have broken `d.get(k, d)`
  and `s.add(x)`, which are real Python. All three engines now raise a CATCHABLE
  `AttributeError`, which is what makes it assertable at all.
- **2.9 / 2.11 ruby half-gaps and value model** — `4731755`. Three of five bullets
  were wrong as written; `printf` was not one wrong answer but three, including a
  live run-vs-native divergence. **And `[1.0] == [1.0]` was FALSE IN THE NATIVE
  BINARY ONLY** — `rbPyEqual` ended in `===` and a Float box is an object in layer
  2 where the twin's `jsFlo` is a value struct. Invisible to the matrix, `--full`
  and `--cross` by construction. MRI's `==` vs `eql?` distinction now exists.

## The eleven closed on 2026-08-06

Old numbering, since that is what other notes cite. **Three of them were never
defects**, which is why this file leads with "re-verify before you act".

- **1.1 Go untyped constants** — `c240f92`. A front-end folder, ~730 lines
  byte-identical between the two halves: base-1e7 bignum plus an exact rational,
  evaluated at parse time, keyed by source position. 328 of 4,212 arithmetic rows
  and **232 of 6,936 COMPARISON rows** wrong against `go` → 0. Comparisons matter
  because Go compares constants exactly too: `2.0/3.0 != 0.6666666666666666` is
  *true*. Named constants remain (item 1.3).
- **1.2 `float` as a real binary32** — `8f43e84`, **java only**; the item said four
  languages and it is five (item 1.2 above). 630 lines wrong against `java` → 0 in
  both compiler engines. The first shape cost **+5.32% against a 4.96% spread** and
  was fixed before landing.
- **1.3 Ruby's arbitrary-precision Integer** — `a3164ff`. The item was mis-scoped:
  it read "literals lose precision" and pointed at `runtime-bignum.metajs`, which
  is float *formatting* and has no integer arithmetic at all. The real item was
  "Ruby has no Bignum" (`2**53+1` was wrong with no literal involved), ~2,300 lines
  across four engines. **A first agent correctly refused to start and mapped it
  instead** — that map is what made the second run land. Ruby ended **11.8%
  FASTER**: the two `typeof`s that detect a bignum also answer `rbToF` and
  `rbNumRank` without calling them, removing four MetaJS calls per arithmetic step
  the *old* code was already paying.
- **1.4 Python `repr()`** — `555af82`. CPython's `unicode_repr`, including the
  quote-selection rule the item did not mention. 21 of 28 assertions fail at the
  parent in each of three engines. Nine unassigned (`Cn`) code points still print
  raw; full `Cn` printability needs the Unicode database.
- **1.5 async/await** — `2339678`. A promise is a plain object with hidden slots, so
  **`runtime.c` was never touched**; `await` compiles to `js_yield` and the EMITTER
  builds the coroutine, which is what makes one algorithm work in both engines.
  Ordering byte-identical to node in `llvm.Run`, native and typescript.
- **1.6 Iterator `return()` on early exit** — `d7ef11a`. The floor learned to throw
  a pinned `GEN_EXIT` sentinel *into* a suspended body so `finally` unwinds; the Go
  twin's `finish()` never ran `finally` either. **The item was right for js/ts and
  wrong for python** — CPython does not close on `break`, and our compiler halves
  already matched. It also uncovered a blocker: `emitClosure` reset `sawYield` for a
  try/finally body closure, so a python generator with a `finally` around its yield
  **could not compile at all**.
- **2.1 Java `record` equality** — `70b3b46`. One extern, three engines; Java needs
  no recursive helper because `Objects.equals` dispatches. The item understated two
  bullets (the interpreter was wrong for *every* box type and for any user-declared
  `equals`, making it an oracle divergence in all three engines) and missed a third:
  **`r.equals(null)` aborted the whole program**.
- **2.4 Python dict `True`/`1`** — **already fixed**, and already asserted at
  `bki3`–`bki16`. `c762428` fixes only the comment that still said otherwise —
  which was the active hazard, since it named the wrong file for the fix.
- **2.9 `ord`/`chr`** — `555af82`. The item said "missing from the interpreter";
  they were missing from **all three engines**. The sweep it asked for found seven
  more, six fixed — `class C(object)` did not compile before it.
- **4.2 The `js FROZEN-DIFF` flake** — `53caf6d`, and it was never a divergence.
  `full_probe` compared the goja and `-frozen` legs **without checking either exit
  code**; a run killed by the 120 s timeout leaves a truncated file, and `cmp -s`
  faithfully reports "differs". The js compiler half is ~19 s idle and **82 s at
  load 16**. It now reports `RUN-FAILED … NOT a frozen divergence`.
- **5.2 Foreign method arms in python's dispatcher** — `555af82`. The framing was
  wrong in a way that would have caused a regression: the arms are layer 2's port
  of `rt.memberCall`, **the table nine languages share**, so deleting python's
  copies alone would have split native from `llvm.Run`. Fixed as a python-specific
  deny plus the deletions. Twelve more names remain (item 2.4).

## Earlier

Every one of these was listed as outstanding and is **fixed**.

**The whole integer-width section** (swift, lua, php, C, dart, java, kotlin).
**Dart's int/double distinction.** **Go widths on struct fields.** **JS BigInt
precision.** **The two `--cross` rows.** **Kotlin `String.uppercase()`.**
**Ruby `%g`** — it landed one commit after the note claiming it was missing.
**Python's `&` being int32-signed** — never broken at the commit that recorded it.
**Python integer literals past 2^53**, decimal and radix. **kotlin's verbatim
1,489-line copy of `regex.js`** and the 4-way regex memo. **The `-min 60` shape
residue** — 14 groups down to 2 deliberate declines. **`scope_find` was the hottest
line in the runtime** — a hash index gave 13 languages 20–42% fewer instructions.
**A layer-2 test gate, a probe command, a shape lint, benchmark baselines** —
`tests/gates.sh`, `tests/probe.sh`, `tools/shape-scan`, `tests/bench.sh`.
**The generator sweep** 25.3 s → 3.3 s, and **`-O2` missing from `buildExecutable`**
for the project's entire life (2.2× on every native binary).
**Python integer arithmetic past 2^53**, **Ruby's three integer directives**,
**Python's `str` method library**, **`for`-of draining eagerly**, **`js_cscmp`
NaN**, **`floAbs`'s 32-bit wrap**, **Java `record` char equality**, **the FMA
divergence**, **native `-exe` matrix rows**, **overlapping the gates** (5:07 → 2:26).

**And a regression found and fixed that was on no list**: the fifth merge pass
(`120ba0f`) cost python 41%, ruby 11% and java 5.7% by turning three hot predicates
into one-line delegations. Reverted in `a8e6aa2`/`34814cd`; `tests/gates.sh --bench`
exists because nothing ran `tests/bench.sh` for four commits.
