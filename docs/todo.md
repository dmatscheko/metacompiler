# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Rebuilt 2026-08-05 at `e284185`, by merging this file's old contents with the
manual's "what is still open" and "what would make this project better" chapters,
and then **re-probing everything**. That mattered more than the merge: of the old
list's largest section — the value model — **almost every item was already fixed**,
and carrying them forward would have sent someone to repair working code. What was
closed is listed at the bottom so nobody re-opens it.

Two conventions, and they are the point of the file:

> **[V]** — a probe or grep was run for this list, on this commit.
> **[U]** — inherited from an older note and NOT re-checked. A lead, not a fact.
>
> **Re-verify before you act.** The lists in this project have gone stale within
> two commits **five** separate times, and twice this week someone was sent to fix
> something that already worked.

Baseline every item must preserve (`tests/gates.sh`, ~2.5 min — the gates run
concurrently since `34814cd`; add `--serial` on a small machine):

```
matrix 351/351 · --full 6,092 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16 none held · native-full 15/15 · go test ok
gen-all 15/15 clean · -freeze a fixed point · bench: no row outside its spread
```

**Run `tests/gates.sh --bench` for anything that touches layer 2 or the floor.**
The seven correctness gates cannot see a slowdown — a merge in `120ba0f` cost
python **41%** and java **5.7%** for four commits with every gate green, because a
41% slowdown is not a wrong answer.

---

# 1. Correctness, with an oracle on this machine

These change answers real programs give. Each has a toolchain here that settles it.

### 1.1 Go untyped constants do not fold at arbitrary precision **[V]**
`fmt.Println(0.1 + 0.2)` prints `0.30000000000000004`; real Go folds constants
exactly and rounds once, printing `0.3`. Needs a constant folder in the front end,
not a value tag. Oracle: `go`.

### 1.2 `float` is modelled as a double in the four JVM-ish languages **[V]**
Java's `1.0f/3.0f` gives `0.3333333333333333` where real Java gives `0.33333334`.
Needs a second box *width*, not a bug fix. Oracle: `java` 24.0.2.

### 1.3 Ruby integer literals lose precision past 2^53 **[V]**
`9007199254740993` reads as `…992`, `0x20000000000001` likewise, and
`12345678901234567890` as `…567000`. Found while verifying `73f1087` — an
80-row-wide MRI difference that turned out to be in the *value* column, not the
formatting. Oracle: `/usr/bin/ruby`.
**Python's is the model, and it is a small job because of that**: `506000a` did
exactly this for python — decide on the TEXT (`pyDecOver53`: strip leading zeros,
compare digit count against `"9007199254740992"`, then lexicographically; never
`parseFloat`, which rounds at exactly the boundary the predicate exists to detect)
and convert radix digits to exact decimal text first, because `js_bigint` reads
decimal only. Ruby also needs arbitrary precision beyond 64 bits, which python got
from `runtime-bignum.metajs`.

### 1.4 Python `repr()` does not escape control characters or backslashes **[V]**
`repr("a\tb")` prints a literal tab where CPython prints `'a\tb'`; the same for
newline, and `repr("back\\slash")` loses the doubling. Found while verifying
`e12ddc1`, confirmed present at `df9199a`, so it is independent of the `str`
method work. Oracle: `python3`. This is what made a 306-row str probe show 36
false differences, so it also costs probe time until it is fixed.

### 1.5 `async`/`await` has no job queue (js, ts) **[V]**
An `async` function compiles to an ordinary function returning its value
directly, so `f().then(...)` dies with *"method call 'then' on a number"*;
`await e` is the identity, `for await` is a plain for-of, `async function*` is a
synchronous generator. Enough for the ratchet, which only defines and type-checks
them.
**The machinery exists**: `abnf/jsrt.go`'s `jsGenerator` runs a compiled body on
its own goroutine with an unbuffered channel handshake and a
`thisStack`/`newTargetStack` switch around every resume — awaits as yields, driven
by a microtask queue drained at the end of `runJSModule`.

### 1.6 An iterator's `return()` is never called on early exit (js, ts, python) **[V]**
Fallout from `fe9fa61`/`e12ddc1`, which made `for`-of lazy. `break`, `throw` or a
`return` out of the loop body now leaves the generator suspended instead of closing
it: after `for (x of g) { break }` a second loop over `g` **resumes** here where
node closes it, and a `finally` around a `yield` does not run. Confirmed against
`node`.
**It cannot be done in the emitter alone**, which is why it was left: the floor's
generator cell exposes `next` *only* (`runtime.c`, tag 15), so a guarded
`it.return()` would work under `llvm.Run` and be a no-op natively — a halves
divergence bought for no behaviour, since the Go twin's `finish()` does not run the
body's `finally` either. Needs `runtime.c` + `abnf/jsrt.go` + layer 2 in ONE commit.
Same family: **`g.close()` answers a `{value, done}` record** in the Go twin and
layer 2 where the interpreter and CPython answer `None`, and nothing tests it.

---

# 2. Cross-engine defects and latent traps

Small, cheap, and each one is a divergence waiting for the first program that
reaches it.

### 2.1 Java's `record` equality is wrong for three more component types **[V]**
`emitRecordEquals` (`java-to-llvm-ir.abnf`), measured against `javac`/`java`
24.0.2 while fixing the char case in `304a234`, and documented in a comment at the
site. Each needs a NEW EXTERN, which is why none was half-fixed:
- a `double`/`float` component uses `===` where a record uses
  `Double.compare(a,b) == 0`, so **NaN records compare unequal in all three of our
  engines** (oracle: equal) and **+0.0/−0.0 records compare equal in both compiler
  engines** (oracle: unequal);
- a **reference** component compares by identity where a record uses
  `Objects.equals` — `record Line(Dot a, Dot b)`, a shape already in
  `tests/java-test-full.java`, and any object with a user-declared `equals`,
  compare unequal in both compiler engines. The interpreter's `jValEq` recurses and
  gets nested records right, so this is an interp-vs-run divergence too;
- the interpreter's `jValEq` returns **false** for two equal `double` components
  (`RD(1.0).equals(RD(1.0))`).

`csharp-to-llvm-ir.abnf`'s `$valeq` — a recursive IR function emitted once per
record-declaring program — is the shape that answers all three.

### 2.2 `-rdynamic` will break the first Linux native build that uses a generator **[V]**
`gen_create` finds `coro_entry` via `dlsym` with no link flag, which works on
darwin; linux needs `-rdynamic` on the clang line in `abnf/llvmlink.go`. Nothing
here builds on linux to prove it, and there is no `rdynamic` anywhere in the repo.

### 2.3 A Python `set` member that is unhashable **[U]**
Compared with `==` in the compiled halves and `===` in the interpreter
(`a=[1]; b=[1]; {a,b}` is 1 vs 2); CPython raises `TypeError`. Closing it means
changing the shared `dictFind` contract, not Python.

### 2.4 Python's dict does not unify `True` and `1` **[U]**
`d[True]="t"; d[1]="one"` is ONE entry in CPython and TWO here, because
`strict_eq`/`rt.strictEq` make a bool and a number different types. Documented at
`runtime.metajs:583`, `python-rt.metajs:213`, `abnf/jsrt.go:2428`. Both halves
agree, so `--cross` is blind.

### 2.5 `i in arr` after `delete arr[i]` **[V]**
`true` here, `false` in node. **Declined twice with the same finding**: a real
hole cannot be expressed without changing the shared `*jsArray` in
`abnf/jsrt.go`, which is the array type for *every* language in the tree, and a
sentinel would need filtering at ~60 read sites where one miss leaks garbage into
`join()`. Listed so the third person to find it stops here.

### 2.6 JS constructs that still abort **[V]** for `super.x`, **[U]** for the rest
`super.x` as a *value* now yields `undefined` rather than aborting. Still
aborting: `super.b = 1` as an assignment target, `new a.b.C()`, nested `new`,
`new C` with no argument list, `class X extends <expression>`, `for (a.b of xs)`,
`export`, `with`. The interpreter additionally lacks a destructuring `catch`
binding — blocked by `excCatch` in `interp-core.js` binding exactly one name,
though the `DPattern`/`bindPattern` machinery already exists.

### 2.7 Interpreter generators replay instead of suspending — js AND python **[V]**
Every `next()` re-runs the body from the top, replaying recorded sends and
stopping at the next yield via a thrown signal. Side effects repeat, it is O(n²),
`try`/`finally` interacts badly with the signal, and `.return()`, `.throw()` and
`Symbol.iterator` are missing.

**Python's interpreter half has the same design** (`genStep`), confirmed while
making `for`-of lazy in `e12ddc1`: a `print` after the first `yield` fires again on
the third step, so the interpreter diverges from CPython *and* from our own two
compiled halves on side-effect ORDER — a probe shows 17 lines where the compiled
halves and CPython agree on 11. Nothing in the suite reaches it, which is why
`--cross` is green.

**If genuine suspension is unreachable here** — the
`-frozen` engine exposes no goroutines and both engines must agree byte-for-byte —
**then document the replay limitation in the grammar's `:description`** rather
than leaving it implicit. That is a valid closure for this item.

### 2.8 The BigInt mix `TypeError` is raised but not catchable **[V]**
`1n + 1` correctly reports *"Cannot mix BigInt and other types"*, but it aborts
rather than being caught by a JS `try`/`catch`. Same family as item 3.2.

### 2.9 `ord` and `chr` are missing from the Python INTERPRETER **[V]**
They exist in both compiled halves and are absent from
`python-interpreter.abnf`'s `hostGlobals`, so any program that uses them is an
interp-vs-run divergence. Found because a probe written in ordinary Python died on
`ord`. Two entries in one table.

---

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

### 4.2 `--full` has a rare unreproduced false positive on `js` **[U]**
One run in four reported `js MISMATCH: 354 FROZEN-DIFF`; eight direct re-runs
hashed identically (1,012,563 bytes, exit 0) and the `timeout 120` hypothesis was
measured and rejected (19.0 s wall under 12-way load). Mechanism unidentified;
most likely the process being killed, since a truncated work file is exactly what
`cmp -s` reports. **Re-run before believing a lone `FROZEN-DIFF` on `js`.**

### 4.3 Two `die("PROBE-C <<")` probes are wired into the shipping floor **[V]**
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

### 5.2 Four foreign method arms live in python's dispatcher **[V]**
`python-rt.metajs:5426` `isEmpty`, `:5447` `removeLast`, `:5487` `sumOf`, `:5497`
`forEach`. Python can utter none of them, so `x.sumOf()` succeeds here and raises
`AttributeError` in CPython — a fidelity bug, not just dead code. (`isEmpty` and
`removeLast` are real Swift and Dart methods and must stay in **those** files.)

### 5.3 `floExt` in `go-to-llvm-ir.abnf` is unreachable **[V]**
`emitBinNum` consults `giBase` first and `giBase` holds every key `floExt` has, so
`js_jfsub/js_jfmul/js_jfdiv/js_jfmod` always leave through `js_giarith`. Defined
at `:1356`, read at `:1339` and `:1505`.

### 5.4 The four `runtime-*.metajs` splits can be folded back **[V]**
`runtime-decimal` (188 lines), `runtime-bignum` (263), `runtime-jvm` (159),
`runtime-dartswift` (50) were split off **only** to keep the per-declaration
live-body tax off non-callers. The `scope_find` hash index removed that tax, so
they can go back into `runtime.metajs` whenever that is the tidier shape — a pure
tidy, no measurable cost either way. **This one really is free, and 5.1's warning
does not apply to it**: folding a module back moves DECLARATIONS, which the hash
index made free, and adds no call depth. It is the *delegation* that costs, not the
location.

### 5.5 `sw_kget`/`sw_kset`/`sw_safeget` are duplicated across three GRAMMARS **[V]**
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

# 9. Closed since the last version of this file — do not re-open

Every one of these was listed as outstanding and is **fixed**. Re-probed on
2026-08-05 unless marked otherwise; the probes are one-liners if you doubt it.

**The whole integer-width section.** Swift `Int.max` and `1 << 62`; Lua
`math.maxinteger` and `9223372036854775807 + 1` wrapping to
`-9223372036854775808`; PHP `echo 1234567890123456789` exact; C `long`
arithmetic (`(int)(5000000000L/1000000)` is 5000, matching `cc`); Dart
`9223372036854775807 + 1`; **Java and Kotlin** `int` overflow and `(byte)200`.

**Dart's int/double distinction.** `print(1.0)` prints `1.0`, `1 / 1` is `1.0`,
`1 is double` is `false`, `1 == 1.0` and `identical(1.0, 1.0)` are `true`.

**Go widths on struct fields.** `var s S; s.b = 255; s.b++` gives `0`, matching
real Go.

**JS BigInt.** It has real precision now — `10n ** 30n` is exact to all 31 digits
and `9007199254740993n` round-trips. (The mix `TypeError` is raised with the right
message but is not catchable — that residue is item 2.11.)

**The two `--cross` rows.** Kotlin nested type declarations and assignment to a
computed target both work; the JS `debugger` statement is a no-op as node has it.

**Kotlin `String.uppercase()`/`lowercase()`** in the compiler.

**Ruby `%g`** — it landed in `2d3e6f5`, one commit after the note claiming it was
missing. A 1,393-row probe found every non-`#` row already matching MRI. The real
gap was the `#` flag, fixed in `1eb31e2`.

**Python's `&` being int32-signed** — never broken at the commit that recorded it;
50 rows over `& | ^ ~ << >>` including the note's own `-1 & 0xffffffff` example
agree with CPython.

**Python integer literals past 2^53**, decimal and radix (`506000a`).

**kotlin's verbatim 1,489-line copy of `regex.js`** (`9c7ac05`), and **the 4-way
regex memo** `jxGetProg`/`k5Get`/`pyERxGet`/`rbRxGet` (`120ba0f`).

**The `-min 60` shape residue** — 14 groups down to 2 deliberate declines
(`120ba0f`).

**`scope_find` was the hottest line in the runtime** — it has a hash index
(`e1307f3`): 13 languages 20–42% fewer instructions.

**A layer-2 test gate, a probe command, a shape lint, and benchmark baselines** —
`tests/gates.sh`, `tests/probe.sh`, `tools/shape-scan`, `tests/bench.sh`.

**The generator sweep** — 25.3 s → 3.3 s (`1a03d08`), and `-O2` was missing from
`buildExecutable` for the project's entire life (2.2× on every native binary).

## The ten closed on 2026-08-05

Old numbering, since that is what other notes will cite. **Two of them were never
defects**, which is the reason this file leads with "re-verify before you act".

- **1.1 Python integer arithmetic past 2^53** — `e12ddc1`. 266 of 1,452 probe rows
  differed from CPython; now 0. The guard is two comparisons and is sound in both
  directions, so it costs nothing (−1.09% over 9 draws, inside the spread).
- **1.4 Ruby's three integer directives** — `73f1087`. 1,363 wrong rows against MRI
  → 0. **One of the three bullets was wrong**: MRI does *not* ignore a precision on
  an integer directive. Two further defects surfaced with it, including a live
  halves divergence (the interpreter floored where both compiled halves truncated).
- **1.5 Python's `str` method library** — `e12ddc1`. 41 methods, 45,064 probe rows
  against `python3`. Needed a FOURTH file (`abnf/pystrmethod.go`) because the item
  said "three files" and the compiler half resolves `js_*` to the Go twin.
- **1.7 `for`-of drains eagerly** — `fe9fa61` (js/ts) and `e12ddc1` (python).
  An infinite generator was not a program you could write; side-effect order now
  matches `node`/CPython byte for byte. Residue is item 1.6 above.
- **2.1 `js_cscmp` NaN** — **already fixed** at `runtime-jvm.metajs:145` and in the
  twin, with an ECMA-334 citation, and the site's own comment said so. Assertions
  added in `304a234`; their discriminating power is **0 of 4**.
- **2.2 `floAbs`'s 32-bit wrap** — `304a234`, and it was in **three** engines, not
  the one recorded. The metajs interpreter was worse than latent: it answered **0**
  for every sized-integer operand. Nothing reached the arm from any language, which
  is why it survived; `flo51`–`flo53` reach it now and 3 of 3 fail at the parent.
- **2.3 Java `record` char equality** — `304a234`. Compared by identity natively
  where real java says equal. The C# sibling was already fixed. Three further
  component-type defects are item 2.1 above.
- **2.4 The FMA divergence in the Go twin** — `73f1087`. Two fusable expressions,
  not three. No answer changes on arm64 and that is not claimed — the point is that
  it no longer depends on codegen, pinned by `abnf/jsrtfma_test.go`. A live third
  engine the item did not mention (`ruby-interpreter.abnf`'s `imod`) was 16 rows
  adrift.
- **4.1 Native `-exe` matrix rows** — `34814cd`. 22 rows, features and multifile
  across 11 languages, each checked to build and exit 0 first. Matrix 329 → 351.
- **7.1 Overlap the gates** — `34814cd`. 5:07 → 2:26 with every verdict identical;
  the hazards the item listed were discharged one by one and written into the
  script's header. `--serial` restores the old behaviour.

**And a regression this round found and fixed, which was NOT on any list**: the
fifth merge pass (`120ba0f`) cost python 41%, ruby 11% and java 5.7% by turning
three hot predicates into one-line delegations. Reverted in `a8e6aa2` / `34814cd`.
`tests/gates.sh --bench` exists now because nothing ran `tests/bench.sh` for four
commits — see the warning on item 5.1.