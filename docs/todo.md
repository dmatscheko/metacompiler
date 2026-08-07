# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Rebuilt 2026-08-05 at `e284185`; **eleven items closed on 2026-08-06**, **ten on
2026-08-07**, **ten on 2026-08-08**, ten more in `483a7a5`..`af16f71`,
ten more in `305fb9e`..`b82f8ed`, and **ten more in `c4e1942`..`b086d07`**,
with what those agents found on the way folded in.
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
> **On the 2026-08-08 round the items were mostly right and STILL not sufficient.**
> Two were wrong as written — **5.2 said `floExt` was dead code when it is live,
> and deleting it as written would have introduced a defect**; and 1.9 named one
> `is` residue where every integral and floating name was broken. And **seven of
> the ten turned up a further defect the item did not name**, five of them live
> halves divergences `--cross` had never reached: csharp's `1 is char`, kotlin's
> `1L == 1.5`, go's silent `<nil> <nil>` comma-ok, java's anon-class field
> shadowing in the *compiler* half, and js's `next(v)` not reaching a delegate.
> **Treat an item as a place to start probing, never as a specification.**
>
> **On the `c4e1942`..`b086d07` round every item was real, and every one of the
> seven agents found MORE than its item named** — 1.8 named one kotlin defect and
> there were ten; 1.4's `.throw()` came with a `yield`-inside-`catch` defect
> reachable with no `.throw()` at all; 1.1's csharp constructors came with `js_has`
> disagreeing between the engines. Two items were wrong about WHOSE fault it was:
> 1.7 said go's divide-by-zero was unrecoverable in the interpreter when the
> NATIVE binary aborted too, and 1.7's first bullet was marked *deliberate* when
> the deliberate part was the representation, not the answer. **The failure mode
> has moved from "the item is wrong" to "the item is too small."**
>
> **Item NUMBERS are re-assigned every time this file is rebuilt.** Code comments
> of the form `docs/todo.md 1.4` are historical provenance pointing at the
> numbering of the day they were written — do not "fix" them, and do not trust
> them to name today's item.

Baseline every item must preserve (`tests/gates.sh`, ~2.5 min — add `--serial` on a
small machine):

```
matrix 351/351 · --full 7,161 assertions, 0 halves disagree · --cross 119/0
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

These change answers real programs give. Each has a toolchain here that settles
it. **Section 1 was emptied for the sixth time** — all ten items are in chapter 9.
What follows is what closing them turned up, all **[V]** on `b086d07` unless
marked otherwise.

### 1.1 csharp: METHOD-GROUP CONVERSION does not exist anywhere **[V]**
`this.Got`, a bare `Got`, and `K.G` used as delegate values all answer
`undefined` in **both** halves; `Op o = F;` says *unknown name: F*. This is what
still blocks the `E += this.Got` repro that `c4e1942` half-closed — `event` is now
a real modifier and `E += lambda` works, but a *method* cannot become a delegate.
Also here: `this.E()` on a delegate **field** parses as a method call and aborts,
and `.Invoke()` is unsupported. Needs a memoised bound closure per receiver, so
that `E -= s.Got` still removes the same object. Highest-value csharp item.

### 1.2 csharp: method overloading is unimplemented for METHODS **[V]**
`c4e1942` fixed constructors; ordinary methods still keep only the last
declaration, so `K.M(1)`, `K.M(1.5)`, `K.M(1,2)` and `K.M("x")` all answer `4`.
The machinery to copy is right there — `csPickCtor`/`csCtorDispatch` and the
compiler's `js_csis` dispatcher — which is why this is small now and was not
before. **Same shape in swift**: initializer overloads are still picked at WALK
time by labels and arity in the compiler half, so `MB(3.0)` runs `init(_ x: Int)`
where the interpreter and `swiftc` both say `Double`. `js_swfits` is already in
place for it (`a68e16d`). A live halves divergence in swift's case.

### 1.3 csharp: a `long` local write still does not adopt **[V]**
`long x; x = 3; x*1000000*1000000` gives `2112827392`; C# says
`3000000000000`. All three engines agree. `c4e1942` adopted `double`/`float`/
`decimal` local writes and **stopped there on a measurement**: adopting every
local write cost `tests/bench/mod.cs` **+5.3%**, and stayed +5.3% after the
conversion was made allocation-free — so the cost is the layer-2 **call**, not the
work. Closing it needs a plain-number test the emitter can lower **without a
call**. The measurement is at the site.

### 1.4 swift: two adoption sites that need a wrapper, not a walker **[V]**
- **A function-typed slot**: `let f: (Double) -> Double = { x in x/2 }` answers 1
  where `swiftc` says 1.5. Unlike every site `a68e16d` closed, the type is on the
  **slot**, not on the closure, so it needs a synthetic function that adopts each
  argument, calls the original and adopts the result. Both engines can build one
  (`emitOverloadDispatch` is the precedent), so **no new extern**.
- **A NEW dictionary key**: `var d: [String: Double] = [:]; d["b"] = 5` gives 2
  where `swiftc` says 2.5. Existing keys adopt; a new one has no old value to
  read. `declTyReg` could record the structural annotation and `makeAssign` derive
  the element type for a one-level index path.

Also `case is Double:` in a `switch` **does not parse in either half**
(pre-existing, verified at `0b0f417`); `case let z where z is Double` works and
agrees with `swiftc` on all 19 probe rows.

### 1.5 go: five residues, and a nil-pointer dereference is a THREE-WAY split **[V]**
- **`recover()` on a nil-pointer dereference**: `llvm.Run` recovers it, the
  interpreter aborts (`fail` at `interp-core.js:533`/`552` — *inside* the file but
  outside `fail()` itself), and natively the floor's `die("nil dereference…")`
  aborts. Needs a `core` hook at those two lines plus either a floor change or a
  go emitter pre-check.
- **A failed type assertion** to an interface with a missing method does not panic
  in any engine.
- **`fmt.Println(f())` with multiple results** prints `[3 3 3]`; `go` prints
  `3 3 3` (multi-returns travel as an array).
- **`return nil` from a map-typed result** renders `<nil>`; `go` prints `map[]`
  (result values do not adopt their declared type).
- **A func-typed struct field cannot be called**: `h.f(4)` → *unknown method 'f'*.
- **Slicing a nil slice loses its nilness**: after `b086d07`, `var a []int; a == nil`
  is correctly `true` but `e := a[:]; e == nil` is still `false` where `go` says
  `true`. Verified by the coordinator; the empty literal, `make` and `append` cases
  all correctly stay `false`, so the `__nil` mark does not leak — this is the one
  slice expression that should propagate it.

### 1.6 python: `yield from` MATERIALIZES its delegate **[V]**
So a `.throw()` is raised at the `yield from` and never reaches the delegate's own
`except` arms, where CPython forwards it. **Both halves agree, so `--cross` is
blind.** Same root cause as the eager `iterToList`. js/ts forward lazily and now
match node (`b086d07`), which is what makes this one visible as a gap.

### 1.7 python: `raise SomeClass` aborts the COMPILER half **[V]**
A bare class rather than an instance works in the interpreter half and aborts in
the compiler. A live halves divergence, pre-existing and unrelated to generators —
found while matching `.throw(SomeClass)` to each half's own `raise`.

### 1.8 python: two builtins and a printing divergence **[V]**
- **`dict.fromkeys`** — the receiver is the `dict` *type object*, which reaches a
  **different dispatcher in each of the three engines**, and it needs both the
  mcall path and the value path or it is reachable one way and not the other. The
  "NOT the whole of CPython's" comments in all three engines now name it.
- **`vars()` is not bound in any engine** (`x.__dict__` works). One line each.
- **A function PRINTS as its MetaJS source text in the interpreter half** and as
  `[function]` in both compiled halves — a live halves divergence. The compiled
  side's text comes from the shared C floor (`runtime.c:2114/3315`), so converging
  means intercepting in python's own renderer in three engines.

### 1.9 ruby: two call-binding rules that disagree with MRI **[V]**
- **A bare method call does not call a method that declares ANY parameters**:
  `def q(a=1); p 3; end; q` prints nothing in both halves where MRI prints 3.
  `makeVarRef`'s comment says this is deliberate and that the feature-matrix file
  relies on the shape to pass a method around as a value — so closing it means
  finding a different way to express that, not just deleting the guard.
- **An optional parameter takes its default when handed a proc**:
  `def m2(f = 3); m2(some_proc)` uses 3 where MRI binds the proc. Both halves
  agree. Cannot be fixed without a block marker distinct from a positional
  argument — the same line `js_rposbind`'s comment draws.

Also: `Proc#arity` reached via `send(:arity)` answers 0, because the registry
`b086d07` added is armed by the parse-time appearance of the NAME. That is the
price of the gate, and the gate is not optional — see chapter 9.

### 1.10 kotlin: the emitter's `this`-probe is not Kotlin-aware **[V]**
Both are pre-existing (reproduced at `0b0f417`), both live in `rtKtGet`, the
compiler half's hottest name-resolution helper, and both are one shape:
- **`with(m) { entries }`** renders `[(a, 1), (b, 2)]` under `llvm.Run` where the
  interpreter and the native binary say `[a=1, b=2]`. `rtSafeGet(thV, name)` in
  `probeB` calls the shared `js_get`, whose dict arm answers `entries` as a Pair
  list, and it hits before anything Kotlin-specific runs. `m.entries` written out
  is right everywhere.
- **Unqualified `size` on an EXTENSION RECEIVER** fails in the compiler:
  `val List<Int>.mid get() = this[size / 2]` → *unknown name: size* (the
  interpreter answers 2). The narrow fix is to route a receiver with no `__class`
  to the `js_ktfget` arm — but that arm *aborts* for a class-literal receiver, so
  it needs its own probe round.

Related and separately recorded at `k2Line`: `println(f())` for a Unit-returning
`f` prints `kotlin.Unit` under `llvm.Run` and a **blank line** natively. MetaJS
has no `arguments`, so layer 2 cannot tell `println()` from `println(Unit)`.

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

### 2.6 The BigInt mix `TypeError` is raised but not catchable **[V]**
`1n + 1` correctly reports *"Cannot mix BigInt and other types"*, but it aborts
rather than being caught by a JS `try`/`catch`. Same family as item 3.2.

### 2.7 go: imported packages share the main file's globals **[V]**
With `mconst.A = 0.1` and a main-file `const A = 5`, `mconst.Sum()` returned 11 at
base. `08e8d3f` fixed it for CONSTANTS (each file gets its own scope stack) but the
underlying runtime global-namespace sharing is still there for vars and funcs. Also:
an imported package's exported CONSTS are not put on the package object (`mconst.S`
is `<nil>`); `var a [size]int` with a named const size gives `len(a) == 1` in the
interpreter; and the compiler half cannot parse `const a, b = 0.1, 0.2`.

### 2.8 Ruby and python residue **[V]**
- **A bare `except:` will not catch python's `GeneratorExit`** — `d7ef11a` routes
  the close sentinel past every catch arm so the two engines agree; CPython lets
  `except:` catch it. Documented at the `js_try` site. **Now costed as item 3.4**,
  after `b086d07` landed `.throw()` and looked at doing `close()` the same way:
  three walls, and the first is that the guard is shared with fifteen languages.
- **An abandoned python generator never runs its `finally`** — CPython finalizes at
  collection; we have no finalizer hook and the GC deliberately has none.
- **`d.items()` renders pairs as lists**, the same tuple root cause as 3.1.
- **`.send()` works on an iterator object** where CPython raises `AttributeError`.
- **ruby `Integer#fdiv` on a bignum, and a big meeting a Rational or Complex**,
  promote through the double where MRI is exact. That needs a Rational whose parts
  are bigs — a second value model, deliberately not started.

### 2.9 Six dead helpers in `python-rt.metajs` **[V]**
`pyFStrictEq pyFTruthy pyFClampSub pyFToInt pyFCall1 pyFWrap32`, plus `pyFBigEq`/
`pyFBigIsZero`: ports of the arms `6eec533` deleted, with one comment describing
callers that no longer exist. Left rather than risk a late unverified edit;
deleting them shrinks layer 2 and is a clean follow-up.

---

### 2.10 A boxed value knows its KIND but not its declared class **[V]**
Two boxed values of the same primitive kind and different declared classes cannot
be told apart — this value model has one representation per kind, not per box.
Surfaced by `44dbd04` while routing java's `equals`, and stated in both
`:description` blocks rather than guessed at. Same family as **kotlin's mixed
Float/Double set key** (closed in `1b54644`) and as **swift's `Int` vs `Int64`**,
which normalise to one representation so `8 as Int64 is Int` is true where
`swiftc` says false (recorded at the site in `a68e16d`). All three would likely be
closed by one mechanism: a nominal type on the value.

### 2.11 A `getattr()`-bound python builtin cannot carry keyword arguments **[V]**
`getattr(xs,"sort")(reverse=True)` differs from `xs.sort(reverse=True)`: a
signature-less builtin receives its keyword dict as a trailing POSITIONAL, and the
bound wrapper cannot tell that from a real argument. Documented at all three sites
and in both grammar headers by `51436f9`. Related and deliberate: `sorted()` still
refuses `key=`/`reverse=` loudly, because they are keyword-only in CPython and a
signature-less builtin here is positional-only — `list.sort` takes both only
because a METHOD call carries its `kw` dict.

### 2.12 Small unbound builtins met in passing **[V]**
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

### 3.4 A catchable `GeneratorExit` — three walls, in cost order **[V]**
`b086d07` landed `.throw()` and **declined `close()` deliberately**, which is the
useful half of the answer:
- `js_try`'s `pending != GEN_EXIT` guard is **shared with fifteen languages**,
  where a close IS a return completion and must not be catchable. Making GEN_EXIT
  catchable globally is wrong.
- `GEN_EXIT` is a bare floor object with **no Python class**, so `except Exception`
  (which must NOT catch it) and bare `except:` (which must) cannot be told apart.
  It needs to be a real `GeneratorExit` whose `__super` is `BaseException` — which
  only python's layer 2 can build.
- So the floor needs `gen_close(g, sentinel)` with the language supplying the
  value, and python's `g.close()` reaches `gen_close` through the tag-15 `return`
  member, which has nowhere to pass one. **A new extern → three engines (manual
  §7.2) → plus `coro-poc`**, plus `gen_close` learning CPython's
  `RuntimeError("generator ignored GeneratorExit")`.
This supersedes chapter 2.8's first bullet, which recorded the *current* behaviour
as deliberate. It still is, until all three walls come down together.

### 3.5 `Symbol.iterator` is a VALUE-MODEL change, not a member-table entry **[V]**
There are **no symbols in the value model at all**, and `for..of`, `yield*` and
`js_iterable` all test structurally for a callable `next` in three engines. Adding
them means a new tag in `runtime.c`, a Go type in `jsrt.go`, `typeof`/`strict_eq`/
`keysOf`/dict-key arms in all three engines, a well-known-symbol registry, and
computed-key support in the class and object-literal emitters of js/ts.
Comparable in size to the tuple item (3.1). It buys one thing the structural test
does not already give: **a user class becoming iterable**.

### 3.6 Coroutines: option A (CPS in the emitter) is the eventual answer **[U]**
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

### 7.7 SEVEN languages have no bench row at all **[V]**
`tests/bench/` has **no swift, dart, php, js, typescript, bash, batch or metajs
program**, so seven languages' layer 2 has no performance gate whatsoever.

Two measured cases of what that costs. `673a8b1` cost ~20% on a `yield*`-heavy
workload (5.35 s → 6.40 s on 300,000 steps) with **no gate able to catch it**. And
in `a68e16d` a swift change cost **+10.3%** in collections (`charAt` allocates a
one-character string in the floor); the agent found it BY HAND because no bench row
loads `swift-rt.ll`, and a `mod.swift` would have caught it automatically.

Both agents measured and disclosed; the next one might not. Add the programs in
the shape of the others and record their rows **by hand** — `--record` rewrites
every row and deletes the header, which is where all the reasoning lives (see
`tests/bench/mod.go` for how the go row was added).

### 7.8 Wire the shape scan into CI with a ratchet **[V]**
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

## The ten closed in `c4e1942`..`b086d07` — ALL of chapter 1 again

`+123 assertions` (7,038 → 7,161), four commits. Seven agents, and **every one of
them found more than its item named**.

- **1.1 + 1.3(csharp) + 1.10** — `c4e1942`. `CtorInit` PARSED `: base(...)` and
  threw the arguments away, so the base class' *parameterless* constructor always
  ran and `new C(3).D` printed **nothing**. A class also kept ONE `ctorInfo`, so
  the last declaration won — `new Ctr(4).D` was 4 where C# says 9. The compiler's
  dispatcher lowers over **`js_csis`, the extern its own `is` already uses**, so
  that half needed no new extern. Found: **`js_has` disagreed between the engines
  for C#** — layer 2 walked the `__class` chain for `__get_X`, the twin answers
  only for a `*jsAccessor`, which a C# property never is.
- **1.2 + 1.3(swift)** — `a68e16d`. `true is Bool` was false in EVERY language
  (`rtIsType`'s boolean arm spells the Java/Kotlin name `Boolean`), and the fix is
  swift-LOCAL: `interp-core.js` untouched. Adoption became one structural walker
  instead of six special cases. The agent **introduced and then caught a
  run-vs-native divergence with its own native probe leg**, and removed a **+10.3%**
  regression before landing (`charAt` ALLOCATES in the floor; the extern is now
  chosen at emit time).
- **1.8 kotlin** — `e60e3d0`. The item named one defect; `js_ktfget` answered
  `undefined` for **every** index property and the sweep found **ten**, including
  `indices` on a String and a StringBuilder missing in BOTH halves,
  `IntProgression.last` returning the bound rather than the last element, and a Set
  not being a builtin receiver in ANY engine.
- **1.4 + 1.5 + 1.6 + 1.7 + 1.9** — `b086d07`, one commit because all four
  workstreams edit `abnf/jsrt.go` and one edits `runtime.c`.
  - `.throw()`: the tag-15 table gained a third member; the flag lives in the
    generator's own control block and the value rides the resume slot, so it is
    **per-coroutine by construction** — the property item 2.1 records as
    mandatory. Found: **a `yield` inside a `catch` ARM ran the enclosing `finally`
    at the suspension** in three interpreter halves, reachable with **no
    `.throw()` at all**; and `StopIteration`'s args were wrong in all three engines
    *and disagreed with each other*.
  - ruby: the item proposed three accessors and that is **not sufficient** — an
    OPTIONAL parameter also mis-binds, needing `spare = npos - required`, which no
    per-parameter accessor can compute. Found: `\a \b \f \v \e`, octal `\nnn`
    and `\u{...}` **were never decoded in any engine**, because all three called
    the shared JavaScript `unescapeJs` and therefore agreed.
  - python: `super()` reproduces CPython's mechanism (a `__class__` cell + the
    frame's first positional) rather than approximating it, and scans the
    **instance's** MRO — the parent pointer is unsound for diamonds. Found: the
    interpreter resolved members **depth-first** where both compiled halves used C3.
  - go: **the item was wrong about who was broken** (the native binary aborted
    too), and bullet 1's "deliberate" marking was about the representation, not the
    answer. `fail()` is untouched — go uses it for non-runtime conditions that must
    not become recoverable panics.

**Three regressions were measured and removed or gated BEFORE landing**, none of
which any correctness gate can see: swift's `charAt` (+10.3% collections), go's
unconditional `js_gidivz` (**+30.58%** — a layer-2 call where `js_giarith` is a C
floor one, so it is not one more call of the same kind; now gated on
`usesRecover`), and ruby's `Proc#arity` registry (181 KB → **5.1 MB** live, since
it retains every closure with its whole scope chain; now armed only when the
program names `arity`).

**And a gate that failed for the right reason and the wrong cause.** `--bench` went
red on csharp at +3.93% over 5 draws. At 9 draws it is +1.98% — and the CLEAN BASE
TREE reads +2.02% against the same baseline, so the change is **−0.04% against its
own parent** and it was the recorded baseline that had drifted. Re-recorded in
`22b08de`. **Five draws is not always enough to separate a row from its spread.**

## The ten closed in `305fb9e`..`b82f8ed` — ALL of chapter 1 again, plus 2.6

`+165 assertions` (6,873 → 7,038), five commits.

- **1.2 + 1.1(csharp) + 1.8(csharp)** — `305fb9e`. The item named one aborting
  shape; there were five, and an accessor-property read was *worse* than an abort
  (the compiler answered nothing). Found: **a subclass declaring any ordinary
  method silently lost every inherited field** — `BaseSpec` pushes
  `{ext: "<name>"}` and `makeMethod` pushes `ext: <bool>`, and `ext != undefined`
  matched both.
- **1.7 + 2.6(python)** — `c2b4071`. **The item was wrong about who was broken**:
  all three engines aborted, not just the interpreter, and the compiled halves'
  exception hierarchy was **flat** (`except LookupError` missed `KeyError`).
- **1.4/1.5/1.6 ruby** — `c2b4071`. Blocks bound as positional arguments,
  non-local `return` from a block, and the `Hash` Enumerable surface with defaults
  carried by `merge`/`dup`.
- **1.9 + 1.8(go)** — `c2b4071`. Four residues plus five more, including **both
  halves reading past a slice's length into its capacity** and **layer 2's `fail`
  not being recoverable while the Go twin's is** — a recovered panic continued
  under `llvm.Run` and aborted the native binary.
- **1.3 + 1.1(swift)** — `8ff0999`. A subclass inherits its initializers; the write
  sites needed no type table. Found: the interpreter's `overloadFits` rejected
  **every `Double` overload** because the shared `rtIsType` answers false for a
  `{__flo}` box.
- **1.10 + 2.6(js)** — `673a8b1`. `g.return()` closes the delegate in both halves;
  and js had python's ordering defect too.
- **1.8(lua, kotlin)** — `b82f8ed`. lua had a real wrong answer against `lua`
  5.5.0, not just a latent divergence. Found: kotlin's interpreter
  `Double.toString` lacked java's two-significant-digit minimum.

**Both 2.6 items claimed "ordering is right in every engine; only repetition
differs". Both were false** — a generator's `finally` fired at the first `next()`,
because the suspension marker is a host throw that every host `try` intercepted.
Manual §7.14.

**Priced and disclosed rather than discovered later**: js's `g.return()` fix costs
a `js_try` per delegated element, ~20% on 300k `yield*` steps. **No bench row
covers js or ts** — see 7.8.


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
