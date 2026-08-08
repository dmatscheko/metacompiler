# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Rebuilt 2026-08-05 at `e284185`; **eighty-one items closed since**, in eight
rounds. Chapter 1 has been emptied and refilled eight times; chapter 9 keeps only
what those rounds taught.

Two conventions, and they are the point of the file:

> **[V]** — a probe or grep was run for this list, on this commit.
> **[U]** — inherited from an older note and NOT re-checked. A lead, not a fact.

> ## Re-verify before you act — and probe the FIX as well as the DEFECT
>
> Across eight rounds the failure mode kept moving, and every stage of it is still
> possible:
>
> - **The item is wrong.** Six of ten on the first round. Three items later were
>   wrong about *which engine* was broken; two named a defect already fixed.
> - **The item is too small** — now the common case. On three rounds nearly every
>   agent found MORE than its item named: one named a single kotlin defect where
>   there were ten, another one `yield from` consequence where there were five.
>   Five of those extras were live halves divergences `--cross` had never reached.
> - **The item's PRESCRIPTION is wrong while its diagnosis is right.** Four items
>   in one round told you how to fix them and were wrong: one regressed two shapes
>   when implemented as written, one gave a false reason, one prescribed a design
>   the value model cannot express, and one declared a bullet impossible that the
>   emitter does trivially.
> - **The work is wrong and every gate says green.** Two agents shipped a NEW
>   halves divergence that their own probes and all seven gates called green.
>
> **Treat an item as a place to start probing, never as a specification.**
>
> **Item NUMBERS are re-assigned every rebuild.** Code comments of the form
> `docs/todo.md 1.4` are provenance pointing at the numbering of the day they were
> written — do not "fix" them, and do not trust them to name today's item.

Baseline every item must preserve (`tests/gates.sh`, ~2.5 min — add `--serial` on a
small machine):

```
matrix 351/351 · --full 7,509 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16 none held · native-full 15/15 · go test ok
gen-all 15/15 clean · -freeze a fixed point · bench: no row outside its spread
```

**Run `tests/gates.sh --bench` for anything that touches layer 2 or the floor.**
The seven correctness gates cannot see a slowdown — a merge in `120ba0f` cost
python **41%** and java **5.7%** for four commits with every gate green.
`tests/bench.sh` takes a LOCK (`ed7086f`): its `-exe` directory is fixed so two
checkouts draw from the same paths, so two concurrent runs used to overwrite each
other's binaries and report a phantom `BUILD FAILED`.

**THE MEDIAN IS THE WRONG STATISTIC WHEN THE SAMPLE IS BIMODAL, AND THESE SAMPLES
ARE BIMODAL.** Layout lands in one of two clusters, so with draws split between
them the median is decided by which side the middle element falls on — it can jump
the whole gap while the code is unchanged. `tests/bench.sh` prints the MEAN beside
the median, says `BIMODAL` when it detects the shape, and has **`--paired
OTHERTREE`**: it builds both checkouts to the same paths and reports a SIGN TEST
over matched pairs. **Reach for `--paired` whenever a delta is smaller than the
row's own spread.** It settled two questions in `d5eb318` in OPPOSITE directions —
swift's `Array.insert` read +2.73% on 21-draw medians and is 12-of-14 pairs
identical (not a cost; the emitted module is md5-identical), while python's builtin
table read +2.67% and is genuinely slower in 11 of 12 pairs. **Always corroborate a
paired verdict with an exact counter** (module bytes, `MEC_GC_STATS`, binary size);
counters are exempt from the lottery and a paired verdict is still a timing.

**FIVE DRAWS IS NOT ENOUGH FOR ANY ROW, AND NINE IS NOT ENOUGH FOR RUBY.** Two
phantom regressions were chased to ground: python read **+0.35% against an
artificially tight 0.24% spread** at 5 draws and **+0.13% against 4.13%** at 9;
ruby read **+2.23%** at 9 draws and **+0.06%** at 15 AND at 21, two independent
paired runs. Both were draw-set artifacts of the bimodal layout distribution
(manual chapter 4). **Raise `--draws` until two paired runs agree, and quote the
spread beside the median.**

**On a loaded machine pass `--timeout 600` to `./test.sh --full`.** The js compiler
half is ~19 s idle and **82 s at load 16** against a 120 s default. This fires
repeatedly with several agents running; a `RUN-FAILED … exit 124` in a language
your change does not touch is this, not a divergence — re-run before believing it.

---

# 1. Correctness, with an oracle on this machine

These change answers real programs give. Each has a toolchain here that settles
it. **Section 1 was emptied for the eighth time** — all ten items are in
chapter 9. What follows is what closing them turned up, all **[V]** on `468f2eb`
unless marked otherwise.

**The failure mode moved again, and this time it was the PRESCRIPTIONS.** Every
item's *defect* was real except where noted, but four items were wrong about the
**fix**, and that is a new place to be careful:

- **1.1 csharp told you how to close the operator bullet, and doing that
  regressed two shapes.** "Decline the user operator when no candidate accepts
  the operands" using the signature machinery — tested per parameter with
  `csIsType`, a null operand stopped selecting `operator ==(V,V)` (ECMA-334
  10.2.7 makes null convertible to any reference type, and `is` says false) and
  an implicit numeric conversion stopped selecting a `double` parameter.
- **1.5 ruby gave a REASON that was false.** "Binding one needs a variadic
  closure and MetaJS has no `arguments`" — MetaJS *has* `arguments`;
  `metajs-to-llvm-ir.abnf:650` binds it in every compiled function and
  `lib/go-rt.metajs:1709` already builds a bound receiver with it.
- **1.6 kotlin prescribed a DESIGN that is not available.** "Carry the receiver's
  origin on the value" — `setMember` rejects every non-numeric key on a
  `*jsArray`, in the twin and in the floor, so a materialized progression cannot
  be marked at all. The *syntactic position* decides instead.
- **1.2 swift declared one bullet impossible and it was not.** "Needs layer 2 to
  build a closure inside `js_swadoptdeep` and cannot" — true of layer 2, and
  irrelevant: the emitter knows every function-type leaf at compile time and can
  pass in a map of makers.

And **2.5 was 7 of 11 bullets already fixed**, including one whose stated blocker
("`excCatch` binds exactly one name") was simply false.

**So: probe the fix as well as the defect.** An item's diagnosis and its
prescription are two different claims and this round the prescriptions were less
reliable than the diagnoses.

### 1.1 No `Error`/`TypeError`/`ReferenceError` GLOBALS exist in any engine **[V]**
`new TypeError("x")` is *variable not defined* in all six js/ts engines. So the
BigInt mix error closed in `bed05be` is caught as a **string**: `e.message` and
`e instanceof TypeError` are unavailable, matching the precedent already set by
the iterator-throw `TypeError`. This is the ceiling on every "make X catchable"
item, including 3.2, and it is one value-model decision away from being fixed
properly (a real exception class hierarchy for js, which python and ruby already
have).

### 1.2 csharp: `undefined + int` is a live run-vs-native divergence **[V]**
`NaN` in the interpreter and `llvm.Run`, **`1` natively**: `rtjNum` in
`languages/lib/runtime-jvm.metajs` answers 0 for undefined where the twin's
`toNumber` answers NaN. Repro: `dynamic d = obj; d.Missing + 1`. **Deliberately
not fixed in `1260832`** because that file is shared with java and the correct C#
answer is a `RuntimeBinderException` neither engine can raise — so closing it
means deciding whether java should move too, or splitting the body.

### 1.3 csharp: a `(object)a == null` guard inside a user `operator ==` recurses forever **[V]**
Stack overflow, reproduced identically at `d473319`, so pre-existing. There are no
static types here to make the cast suppress the operator, which is exactly what
C# uses it for — this is the idiomatic null-guard, so it is likely to be met
again.

### 1.4 An extension's static overloads still lose the type's own **[V]**
`emitExtMembers` restarts the static-overload counter at 0, so
`extension S { static func f(_:Int) }` on a type already declaring `static func f`
drops the type's. swift, pre-existing "last wins" for that shape and NOT widened
by `fc635bb`, which fixed the non-extension case. Closing it needs the same
`initIdxs` treatment initializers already get.

### 1.5 go: four residues, each with `go` as the oracle **[V]**
- an **iota-group constant as an array length** (`const (R = iota; G; B); var a [B]int`
  is 0 where go says 2) in all three engines — `cBindSpec` has no resolved value
  for a spec with no `=`, so it needs the iota re-emission machinery;
- a **struct-valued map key** (`Named{Pair{1,2}: "p"}`) misses on lookup, because
  `dictFind` compares by identity; base cannot even parse the literal;
- the **compiler's package object is a snapshot**, so a later write to a package
  var is invisible through the qualifier while the interpreter (which binds the
  scope) is live. Making the compiler live needs the package object to *be* a
  scope, and a scope handle is not something `js_get` can read a member off;
- **struct type names are one flat table** across packages (`structs` /
  `structFields`), so two packages declaring the same type name collide.

### 1.6 ruby: `Kernel#lambda { |x| }.call` is not arity-checked **[V]**
`->(x){}.call` is, after `310e6b6`. The block's shape reaches `lambdaWrap` /
`js_rlambda` only through the arity registry, which is gated on the program naming
`arity` — and using that gate for *correctness* would make behaviour depend on it.
Closing it means carrying the block's `kinds` from the literal site into
`Kernel#lambda` in both engines and wrapping the closure: ~60–80 lines over four
files, and it changes the identity of a `Kernel#lambda` result.

### 1.7 ruby: an uncaught exception reports `[object Object]` **[V]**
In both compiled halves; the interpreter prints the message. Pre-existing at
`d473319`, in shared machinery, and reachable by any uncaught `raise` — so it is
the first thing a user sees when their program is wrong.

### 1.8 kotlin: an empty range's `first` is a live halves divergence **[V]**
`(5..1).first` is **5** in the interpreter and **`kotlin.Unit`** in both compiled
halves. Not fixed in `deddac4` and deliberately not asserted, because no single
expected value holds for both halves: it is unrecoverable from an empty
materialized list without changing how a progression is represented. Same family
as 2.4's "the value model has one representation per kind".

### 1.9 python: `len.__name__` answers nothing **[V]**
The builtin-render table added in `468f2eb` deliberately does not feed `__name__`,
because doing so would desynchronise the def-name table that answers a *user*
function's `__name__`. One line if the two tables are reconciled.

### 1.10 Small gaps met in passing, each cheap **[V]**
ruby `String#start_with?` is unimplemented in both compiled halves (the
interpreter has it) and `s[0, 6]` answers one character instead of the substring
in all three; kotlin's unqualified `toString()` on a `Pair`/`Map.Entry` misses in
all three, and `mr.range.toString()` gives `[object Object]`. Also java
`Integer.toHexString`, `Double.NaN`, `Boolean.valueOf`; swift
`Float.greatestFiniteMagnitude` / `.leastNonzeroMagnitude`. Each is a one-line
binding or close to it; they are grouped because none is worth its own item.

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

### 2.3 `with` is the one js construct that cannot be lowered honestly **[V]**
~170 lines, and size is not the blocker. The INTERPRETER can implement it exactly
(`scopes.push(o)` — scopes there are plain objects). Both compiled engines cannot:
the floor's scope is a name/value pair of arrays with a hash index
(`runtime.c` `scope_find`), **not object-backed**, so the only available lowering
is snapshot-in/write-back, which cannot see a property added inside the body.
Shipping that would ship a *designed-in* halves divergence, which is worse than
the gap. `tests/js-test-full.js`'s own header already lists `with` as out of
scope. Recorded so the next person stops here; it needs an object-backed scope in
the floor, or nothing.

### 2.4 A boxed value knows its KIND but not its declared class **[V]**
Two boxed values of the same primitive kind and different declared classes cannot
be told apart — this value model has one representation per kind, not per box.
Surfaced by `44dbd04` while routing java's `equals`, and stated in both
`:description` blocks rather than guessed at. Same family as **kotlin's mixed
Float/Double set key** (closed in `1b54644`), **swift's `Int` vs `Int64`** (which
normalise to one representation so `8 as Int64 is Int` is true where `swiftc` says
false, recorded at the site in `a68e16d`), and **kotlin's empty-range `first`**
(item 1.8). All four would likely be closed by one mechanism: a nominal type on
the value.

### 2.5 A Python `set` member that is unhashable **[U]**
Compared with `==` in the compiled halves and `===` in the interpreter
(`a=[1]; b=[1]; {a,b}` is 1 vs 2); CPython raises `TypeError`. Closing it means
changing the shared `dictFind` contract, not Python.

### 2.6 `i in arr` after `delete arr[i]` **[V]**
`true` here, `false` in node. **Declined twice with the same finding**: a real hole
cannot be expressed without changing the shared `*jsArray` in `abnf/jsrt.go`, which
is the array type for *every* language, and a sentinel would need filtering at ~60
read sites where one miss leaks garbage into `join()`. Listed so the third person to
find it stops here.

### 2.7 Ruby and python residue **[V]**
- **A bare `except:` will not catch python's `GeneratorExit`** — `d7ef11a` routes
  the close sentinel past every catch arm so the two engines agree; CPython lets
  `except:` catch it. Documented at the `js_try` site. **Costed as item 3.4.**
- **An abandoned python generator never runs its `finally`** — CPython finalizes at
  collection; we have no finalizer hook and the GC deliberately has none.
- **`d.items()` renders pairs as lists**, the same tuple root cause as 3.1.
- **ruby `Integer#fdiv` on a bignum, and a big meeting a Rational or Complex**,
  promote through the double where MRI is exact. That needs a Rational whose parts
  are bigs — a second value model, deliberately not started.
- **`method(:x).class` is `Proc`** where MRI says `Method`, and a class-object
  `self` (`def self.x`) still cannot be bound (`310e6b6`).

### 2.8 A `getattr()`-bound python builtin cannot carry keyword arguments **[V]**
`getattr(xs,"sort")(reverse=True)` differs from `xs.sort(reverse=True)`: a
signature-less builtin receives its keyword dict as a trailing POSITIONAL, and the
bound wrapper cannot tell that from a real argument. Documented at all three sites
and in both grammar headers by `51436f9`. Related and deliberate: `sorted()` still
refuses `key=`/`reverse=` loudly, because they are keyword-only in CPython and a
signature-less builtin here is positional-only — `list.sort` takes both only
because a METHOD call carries its `kw` dict.

### 2.9 `tests/bench/mod.py` sits essentially on an ARENA THRESHOLD **[V]**
It will tax the next person who adds a kilobyte to python's layer 2 by ~2.4%, for
free, and it will not be their change. Measured in `468f2eb`: the builtin-render
table added 3,184 bytes of live data and the heap stepped 9,530,224 → 11,627,376,
**exactly one 2 MiB arena doubling**, which the sweep then walks 3,574 times —
with `collections=3574` IDENTICAL before and after, so nothing new is executed.
The control is decisive: one *unused* 1 KB string global added to
`python-rt.metajs` at `d473319`, and nothing else, moves the row **+2.40%**.
Either give `mod.py` some headroom (and re-record), or record the arena occupancy
beside the row so the step is predictable. Do not "fix" it by shrinking layer 2.

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

### 4.3 A FEATURE-MATRIX assertion is never RUN natively — only built **[V]**
The matrix's native rows are `-exe` with no run, and `mec … -exe PATH` only BUILDS
(manual 7.10). So a fix asserted **only** in `tests/<lang>-test-features.<ext>`
never executes in a native binary in any gate: `clang-check` and `native-full`
both run the **ratchet** file, `tests/<lang>-test-full.<ext>`.

**This is not theoretical.** On the `b3eb5c3` round a layer-2 `Hash#each` defect —
in `ruby-rt.metajs`, i.e. the half only a native run can see — survived every one
of the seven gates and was found because the agent ran the binary by hand. It was
closed only after a ratchet SECTION was added.

**So: a layer-2 or floor fix needs a ratchet section, not just a feature-matrix
assertion.** Two ways to close the hole properly, neither started: give the
feature-matrix files native rows that RUN (the `SHOULD ABORT` rows already prove
the mechanism exists), or make `native-full.sh` run both files per language.

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
> **The discipline this item created has now paid out once.** `468f2eb` deleted
> eight python helpers claimed dead by the old item 2.9 — but only after a fatal
> probe at the top of each body fired **0 times** across every python program
> built NATIVELY and run. Two neighbours in the same family, `pyFTypeOf` and
> `pyFArgAt`, are live and were kept. Do this every time; the cost is one
> regenerate-and-run.

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

Eight rounds of ten, `e284185`..`10c6e58`, taking the ratchet to **7,509
assertions**. Per-item detail is in this file's git history and in the commit
messages, which are written to be read. What is kept here is only the part that
should change how the NEXT item is worked.

## What was closed, by area

- **Value model and numbers** — `float` as a real binary32 in java, kotlin, swift,
  go and csharp, each with its own renderer; Go untyped and named constants (a
  parse-time bignum/rational folder); Ruby's arbitrary-precision Integer (~2,300
  lines, and ruby ended **11.8% FASTER**); Ruby `Float#to_s`; python `repr()`;
  java `record` equality and `hashCode`; kotlin `Float.hashCode()`; the
  signed-zero family, sighted **nine** times.
- **Coroutines and iterators** — async/await, `for await`, `async function*`,
  `ag.throw()`, `.throw()` on plain generators, `yield*` throw-forwarding and
  delegate `next(v)`, iterator close on `break`/`return`/labeled break, python
  `yield from` (five consequences where the item named one), `g.return()`.
- **Object models** — java unqualified field access; csharp constructors,
  `: base(...)`, overloaded operators, `is`, inherited statics and delegate
  fields; swift initializer/method/subscript overload dispatch and `as` as a real
  coercion; python `super()` over the instance MRO; go interface method sets,
  embedded-struct promotion, multi-value expansion; ruby blocks-as-arguments,
  non-local `return`, `method(:x)`, `send`/`__send__`/`public_send`.
- **Library surfaces** — python builtins and the dict/set/list/str surfaces; ruby
  Array and Hash; kotlin index properties and collection receivers; twelve foreign
  method names denied by RECEIVER TYPE rather than by name.
- **Infrastructure** — `tests/gates.sh`, `tests/probe.sh`, `tests/bench.sh`,
  `tools/shape-scan`, `tests/gen-all.sh` (25.3 s → 3.3 s), native `-exe` matrix
  rows, overlapping the gates (5:07 → 2:26), **`-O2` missing from
  `buildExecutable` for the project's entire life** (2.2× on every native binary),
  and a `scope_find` hash index (13 languages, 20–42% fewer instructions).

## The lessons worth carrying

The mechanics are in the manual, chapters 4 and 7. These are the ones about how to
WORK, and every one was paid for.

- **The instrument itself was wrong twice.** A single build cannot measure below
  ~2% (the layout lottery), and then the MEDIAN turned out to be the wrong
  statistic on a bimodal sample. Both fixes are in `tests/bench.sh` — `--draws`,
  `--paired`, the printed mean — and the rule is at the top of this file.
- **Four regressions were invisible to every correctness gate.** Three were caught
  because someone measured: swift's `charAt` (+10.3% collections), go's
  unconditional `js_gidivz` (**+30.58%**), ruby's `Proc#arity` registry (181 KB →
  **5.1 MB** live). The fourth was not caught for four commits — the fifth merge
  pass cost python **41%**, ruby 11% and java 5.7% by turning three hot predicates
  into one-line delegations. That is why `--bench` exists and why 5.1 carries its
  warning.
- **Two defects were found only by running a binary BY HAND**, because a
  feature-matrix native row only builds. That hole is item 4.3, still open.
- **Deleting "dead" code needs a fatal probe that fires zero times**, not an
  argument. Eight python helpers went that way; two neighbours in the same family
  proved live and were kept. The one item that skipped the step (`floExt`) was
  wrong, and deleting as written would have introduced a defect.
- **Both halves agreeing is not evidence.** The largest class of defect closed
  across these rounds is "both halves agree and both are wrong" — `--cross` cannot
  see it by construction; only an external oracle can.
- **Two test files had been relying on defects.** When a fix makes a test fail,
  establish which of the two is wrong before touching either.