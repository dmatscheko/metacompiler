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

Baseline every item must preserve (`tests/gates.sh`, ~5 min):

```
matrix 329/329 · --full 6,003 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16 none held · native-full 15/15 · go test ok
gen-all 15/15 clean · -freeze a fixed point
```

---

# 1. Correctness, with an oracle on this machine

These change answers real programs give. Each has a toolchain here that settles it.

### 1.1 Python integer arithmetic does not promote past 2^53 **[V]**
`9007199254740992 + 1` is unchanged. Of 55 differing rows in a 626-literal probe
against CPython 3.14.6, **54 are the `*` column** and 1 is `+`; literals, types,
`&`, `>>` and `abs` are all exact. The site is the plain `l+r`/`l-r`/`l*r` arms in
all three engines — **the hot path of every Python program**, where a naive guard
measured **+9.5%** — and `*` needs the exact product, which is the expensive half.
The integer *literal* half is done (`506000a`); this is the arithmetic half.

### 1.2 Go untyped constants do not fold at arbitrary precision **[V]**
`fmt.Println(0.1 + 0.2)` prints `0.30000000000000004`; real Go folds constants
exactly and rounds once, printing `0.3`. Needs a constant folder in the front end,
not a value tag. Oracle: `go`.

### 1.3 `float` is modelled as a double in the four JVM-ish languages **[V]**
Java's `1.0f/3.0f` gives `0.3333333333333333` where real Java gives `0.33333334`.
Needs a second box *width*, not a bug fix. Oracle: `java` 24.0.2.

### 1.4 Ruby's three integer directives **[V]**
All three are shared by **both halves**, so `--cross` is blind to them by
construction. Oracle: `/usr/bin/ruby` 2.6.10 — integer formatting is unchanged in
3.x, so this old Ruby settles them.
- `%d`/`%x`/`%o`/`%b` of a float past 2^53 print the double's shortest form or
  saturate at int64; MRI prints the exact integer.
- Negatives under `%x`/`%o`/`%b` need MRI's infinite two's-complement notation
  (`"%x" % -5` is `..fb`, `%o` is `..73`, `%b` is `..1011`).
- A precision on an integer directive is ignored (`"%.0x" % 0` is `""` in MRI).

The `#` flag and the base prefixes landed in `1eb31e2` and are the model to
follow: `rbBasePrefix` / `basePrefix` / `rubyBasePrefix`, changed together in
`abnf/jsrt.go`'s `rubyFormat`, `lib/ruby-rt.metajs`'s `rbFormat`, and
`ruby-interpreter.abnf`'s `fmtOne`.

### 1.5 Python has no `str` method library at all **[V]**
`"abc".upper()` fails identically in both halves with `unknown String method:
upper`. A missing *feature*, not a defect. Three files:
`python-interpreter.abnf`, `python-to-llvm-ir.abnf`, `lib/python-rt.metajs`.

### 1.6 `async`/`await` has no job queue (js, ts) **[V]**
An `async` function compiles to an ordinary function returning its value
directly, so `f().then(...)` dies with *"method call 'then' on a number"*;
`await e` is the identity, `for await` is a plain for-of, `async function*` is a
synchronous generator. Enough for the ratchet, which only defines and type-checks
them.
**The machinery exists**: `abnf/jsrt.go`'s `jsGenerator` runs a compiled body on
its own goroutine with an unbuffered channel handshake and a
`thisStack`/`newTargetStack` switch around every resume — awaits as yields, driven
by a microtask queue drained at the end of `runJSModule`.

### 1.7 `for`-of drains eagerly in js, ts and python, so an infinite generator hangs **[V]**
`js_iterable` materialises the sequence (`js-to-llvm-ir.abnf:3672` and `:4223`,
whose own comment says "materializes"; python via `js_pyiter`).
**C# is the counter-example and owes layer 2 zero lines** — its emitter already
drives `js_get(g,"next")` lazily in IR. So **this is an EMITTER change (a
`next()`-until-`done` loop), not a layer-2 one**, and the coroutine primitive it
needs already exists. ~30 lines for js/ts, ~50 for python (`send(v)` = `next(v)`,
`StopIteration` carrying the return value, `close()` = drop the reference).

---

# 2. Cross-engine defects and latent traps

Small, cheap, and each one is a divergence waiting for the first program that
reaches it.

### 2.1 `js_cscmp` does not special-case NaN **[U]**
`lib/csharp-rt.metajs:1011` is a bare delegation to `rtjRel`. The recorded fix is
one line — `if (c == 2) { return false }` on the compare result. Until it lands
the two C# engines disagree on a NaN comparison and the ratchet assertion cannot
be added. No C# toolchain here; cite ECMA-334.

### 2.2 `floAbs` still carries the 32-bit wrap that `Math.abs` was fixed for **[U]**
`abnf/jsrtjvm.go:391` — non-`jsJFlo` operands go through
`float64(rt.toInt32(math.Abs(...)))`. Latent: it bites the day a long reaches
layer 2's float primitive.

### 2.3 Java's generated `record` equality compares components with `js_seq` **[U]**
`java-to-llvm-ir.abnf:3923`. The Go twin's char is a comparable struct; layer 2's
is a `{__char}` box, so `record R(char c)` compares by **identity** natively. No
test declares a char record component, so it is unmeasured rather than
known-wrong. One-line fix: `js_jchareq`, which both emitters already use
elsewhere. A sibling in C# (a two-char list pattern with a rest,
`csharp-to-llvm-ir.abnf` ~2470) was reported but the line has moved.

### 2.4 The FMA divergence is closed in layer 2 and still open in the Go twin **[U]**
Go fuses `x - math.Floor(x/y)*y` into an FMSUB on arm64; clang does not, so the
halves diverged once `q*y` left the exactly-representable range
(`9007199254740991 % -3`). Real ruby says the fused answer is correct, so it was
layer 2's defect, and `rbFmaSub` (Dekker two-product/two-sum) now reproduces it.
**The twin is exact only by accident of fusion** — on a non-FMA target the halves
diverge again. The fix is three `math.FMA` lines at `abnf/jsrt.go:3458` and
`:3673`.

### 2.5 `-rdynamic` will break the first Linux native build that uses a generator **[V]**
`gen_create` finds `coro_entry` via `dlsym` with no link flag, which works on
darwin; linux needs `-rdynamic` on the clang line in `abnf/llvmlink.go`. Nothing
here builds on linux to prove it, and there is no `rdynamic` anywhere in the repo.

### 2.6 A Python `set` member that is unhashable **[U]**
Compared with `==` in the compiled halves and `===` in the interpreter
(`a=[1]; b=[1]; {a,b}` is 1 vs 2); CPython raises `TypeError`. Closing it means
changing the shared `dictFind` contract, not Python.

### 2.7 Python's dict does not unify `True` and `1` **[U]**
`d[True]="t"; d[1]="one"` is ONE entry in CPython and TWO here, because
`strict_eq`/`rt.strictEq` make a bool and a number different types. Documented at
`runtime.metajs:583`, `python-rt.metajs:213`, `abnf/jsrt.go:2428`. Both halves
agree, so `--cross` is blind.

### 2.8 `i in arr` after `delete arr[i]` **[V]**
`true` here, `false` in node. **Declined twice with the same finding**: a real
hole cannot be expressed without changing the shared `*jsArray` in
`abnf/jsrt.go`, which is the array type for *every* language in the tree, and a
sentinel would need filtering at ~60 read sites where one miss leaks garbage into
`join()`. Listed so the third person to find it stops here.

### 2.9 JS constructs that still abort **[V]** for `super.x`, **[U]** for the rest
`super.x` as a *value* now yields `undefined` rather than aborting. Still
aborting: `super.b = 1` as an assignment target, `new a.b.C()`, nested `new`,
`new C` with no argument list, `class X extends <expression>`, `for (a.b of xs)`,
`export`, `with`. The interpreter additionally lacks a destructuring `catch`
binding — blocked by `excCatch` in `interp-core.js` binding exactly one name,
though the `DPattern`/`bindPattern` machinery already exists.

### 2.10 JS interpreter generators replay instead of suspending **[U]**
Every `next()` re-runs the body from the top, replaying recorded sends and
stopping at the next yield via a thrown signal. Side effects repeat, it is O(n²),
`try`/`finally` interacts badly with the signal, and `.return()`, `.throw()` and
`Symbol.iterator` are missing. **If genuine suspension is unreachable here** — the
`-frozen` engine exposes no goroutines and both engines must agree byte-for-byte —
**then document the replay limitation in the grammar's `:description`** rather
than leaving it implicit. That is a valid closure for this item.

### 2.11 The BigInt mix `TypeError` is raised but not catchable **[V]**
`1n + 1` correctly reports *"Cannot mix BigInt and other types"*, but it aborts
rather than being caught by a JS `try`/`catch`. Same family as item 3.2.

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

### 4.1 No native (`-exe`) matrix rows for any of the thirteen handle-IR languages **[V]**
`.vscode/launch.json`'s 32 `-exe` rows are metajs, lua and lisp only. Every
migration wrote out the exact args it wanted (php 3, swift 2, java 3, csharp 3,
dart 2, go 3, ruby 3, js 2, ts 2, python 3) and all passed at the time; none were
added because the file belonged to the coordinator. Partly mitigated —
`clang-check.sh`/`native-full.sh` already build the ratchets natively — so what is
actually missing is native coverage of the **features** and **multifile**
programs.

### 4.2 The two MetaJS halves bind different host globals **[V]**
`metajs-interpreter.abnf` binds eleven; the compiler half's `standardJSBindings`
(and the C floor, which matches the *compiler*) also bind `Infinity NaN Array
Object byteLen sprint rawSet`. `tests/metajs-test-full.js` asserts only the
intersection, so nothing can see it. Same family: `"ß".toUpperCase()` is `"SS"` in
the interpreter (real JS) and `"ß"` in the compiler and floor (Go's simple
mapping).

### 4.3 `--full` has a rare unreproduced false positive on `js` **[U]**
One run in four reported `js MISMATCH: 354 FROZEN-DIFF`; eight direct re-runs
hashed identically (1,012,563 bytes, exit 0) and the `timeout 120` hypothesis was
measured and rejected (19.0 s wall under 12-way load). Mechanism unidentified;
most likely the process being killed, since a truncated work file is exactly what
`cmp -s` reports. **Re-run before believing a lone `FROZEN-DIFF` on `js`.**

### 4.4 Two `die("PROBE-C <<")` probes are wired into the shipping floor **[V]**
`runtime.c:2410` and `:2417`. They are the fatal probes that measured the
shift-result-type claim, which fired **zero** times. Document them at the site or
remove them; they currently read as leftover scaffolding.

---

# 5. Duplication and dead code

All small. All verified. The shape scan is the measure — `go run ./tools/shape-scan`.

### 5.1 The `-min 40` shape tier is untouched: 15 groups, 111 recoverable lines **[V]**
`-min 60` is finished at 2 groups, both deliberate declines (manual §7.12).
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
tidy, no measurable cost either way.

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

Ordered by how much time each would save. The first two are measured and ready.

### 7.1 Overlap the seven gates in `gates.sh` — measured at 291 s → 150 s **[V]**
`tests/gates.sh` runs its gates strictly serially at 315% average CPU on a 16-core
machine. Launched concurrently with a single wait: **150 s, every verdict
byte-identical**. ~1.94× on the whole verification loop, in a file the harness
already owns.
**Not shipped, and the hazard must be discharged first**: the matrix entries in
`.vscode/launch.json` write **fixed paths inside the repo** (`-exe
tests/metajs-native-fail.out`, `…-undeclared.out`, `…-prims.out`,
`tests/metajs-link.out`). Verify no gate other than `./test.sh` writes into
`tests/`, and that no two concurrent gates can write the same path. `native-full`
uses `./mec` at the repo root, which `gates.sh` builds — confirm nothing rewrites
it mid-run. **`--freeze` writes `abnf/jsbootstrap.ll` and `jsagrammar.go`, which
every other gate reads, and must stay strictly serial after the wait.** Consider a
`--serial` escape hatch for low-core machines. **A false green here is
catastrophic**, so the verdict logic must not change at all — only when each
command runs.

### 7.2 Should `test.sh` default to 2× ncpu jobs? **[V]**
`test.sh` defaults `JOBS` to `sysctl -n hw.ncpu`, which does **not** saturate the
machine: `./test.sh` 35.0 s at 768% CPU, `-j 32` **25.4 s at 1143%**, same
329/329. The other two groups do not benefit and should keep the default
(`--full` 95 s → 91 s, tail-bound on ~16 ratchet files; `--cross` 12 s both ways).
Decide whether the matrix's default becomes `2 * ncpu` or whether `gates.sh`
passes `-j` for the matrix only, and confirm on a second machine that
oversubscription introduces no flakiness or memory pressure — the entries are
independent subprocesses, so the mechanism is the one already used at 16-way, but
this was a single measurement.

### 7.3 Re-measure lua's hash-index buy-backs with draws **[V]**
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

### 7.4 Per-language coverage of the floor **[U]**
The instrumentation that found 24 unreached floor bodies was written once and
thrown away. Keeping it as `tools/floor-coverage` would make "is this body dead or
untested?" answerable at any time — **it is the question that gates Part B (3.3)**,
because a body no test reaches is a body no move can be validated against.

### 7.5 A `--why` flag for the emitters **[U]**
Understanding which extern an emitter chooses for an operator means reading
thousands of lines of grammar, repeatedly. A flag that prints the extern chosen
per AST node would make layer-2 work dramatically faster.

### 7.6 Make MetaJS's dialect gaps loud **[U]**
No exponent literal, no `toPrecision`, typed locals, ASI differences: each is
discovered by a confusing failure. A `-verify` pass over `lib/*.metajs` that names
them would pay for itself immediately. (`docs/abnf-dialect-gotchas.md` lists them;
nothing enforces them.)

### 7.7 Make the frozen-snapshot rule mechanical **[U]**
"Did you `-freeze` after touching the emitter?" is a question a script should ask.
`tests/gates.sh --freeze` checks it on demand; a pre-commit hook would remove the
class entirely.

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
