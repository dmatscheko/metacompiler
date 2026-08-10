# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Completed work lives in git, not here. [§8](#8-settled--do-not-re-open) keeps
one line per settled question — the VERDICT and its REASON, nothing else, because
that is the only part that stops someone re-opening it.

Two conventions, and they are the point of the file:

> **[V]** — a probe or grep was run for this list, on this commit.
> **[U]** — inherited from an older note and NOT re-checked. A lead, not a fact.

> **Treat an item as a place to start probing, never as a specification.** Two
> rounds running, the list has been wrong more often than right, and ALWAYS in the
> direction of understating the defect.
>
> The seventeenth round: of ten items, **four were already fixed or had the wrong
> premise** (python's integer arithmetic past 2^53 — 70 probe rows agree with
> CPython; ruby's `%g`; python's `&`, its own example row included; java's
> divide-by-zero message, already asserted) and **three prescribed a FIX that was
> wrong** (see §8). One named two defects and there were **six**, three of them
> live halves divergences. One warned that a change would break every nested
> class name; measured, it does not.
>
> The sixteenth was the same shape: three items wrong as written, two prescribed
> fixes wrong — dart's "erase type parameters" (Dart reifies), C#'s "the message
> applies to every derived type" (fixes one name, breaks twenty-six) — and one
> asserting real TypeScript accepts a form it rejects.
>
> **Every time, the defect underneath was real and bigger.** So: **probe the FIX
> as well as the DEFECT**, and re-probe any **[U]** item BEFORE assigning it, not
> during. **Item NUMBERS are re-assigned every rebuild** — a citation of
> `docs/todo.md 1.4` is provenance, not an address.

Baseline every item must preserve (`tests/gates.sh`, `--serial` on a loaded
machine, `--bench` for anything touching layer 2 or the floor):

```
matrix 383/383 (16 SHOULD ABORT, 14 of them ALL HALVES) · --full 9,219
assertions, 0 halves disagree · --cross 121/0 · clang-check 16/16 none held
go test ok · gen-all 15/15 clean · native-full 20 languages, 0 failures
shape-scan 15 groups / 481 recoverable · metajs-lint 0 findings, self-test ok
-freeze a fixed point · bench: 15 rows, no row outside its spread
tests/coro-poc/build.sh byte-identical
```

**Before measuring anything, read the manual's chapter 4 and
`tests/bench/baseline.txt`'s header** — instructions not wall clock, medians over
layout draws, `--paired`, corroborate with an exact counter.

---

# 1. Correctness, with an oracle on this machine

### 1.1 swift: `var b = a` SHARES the array **[V]**
`var a=[1,2,3]; var b=a; b.append(9); print(a)` is `[1,2,3,9]` here and `[1,2,3]`
under `swiftc`. Copy-on-assign is missing for **every** array, dict, tuple and
struct binding, in all three engines — Swift's value semantics are the language's
defining property, so this is the largest remaining swift gap. Adjacent and much
smaller: `String(s.sorted())` gives `["e","h",…]` where swiftc gives `ehllo`,
because `String(x)` and `String(describing: x)` are one conversion here.

### 1.2 kotlin: the remaining `fail()` sites in the index family **[V]**
Each agreed-and-wrong, each with its java 24.0.2 answer already measured:
`removeAt(9)`/`set(9,x)` want `IndexOutOfBoundsException: Index 9 out of bounds
for length 2` (the same helper the kotlin index work added); `add(9,x)` wants the DIFFERENT format
`Index: 9, Size: 2`; `"ab".substring(5)` answers `""` where java raises
`StringIndexOutOfBoundsException: Range [5, 2) out of bounds for length 6`;
`listOf<Int>().first()` aborts where Kotlin throws `NoSuchElementException`.

### 1.3 ruby: `format` residue, and both are ARGUMENT-list rules **[V]**
"Too few arguments" is not raised anywhere (`"%d" % []` is `"0"`); MRI raises.
This is a separate decline from 1.3's format-STRING work and converging it means
raising in all three engines. Also `Rational#round(n)` / `Complex` go through
`toF` and answer a Float where MRI answers a Rational for `n > 0`.

# 2. Cross-engine defects and latent traps

### 2.1 `with` is the one js construct that cannot be lowered honestly **[V]**
The INTERPRETER can implement it exactly (`scopes.push(o)` — scopes there are
plain objects). Both compiled engines cannot: the floor's scope is a name/value
pair of arrays with a hash index, **not object-backed**, so the only lowering is
snapshot-in/write-back, which cannot see a property added inside the body.
Shipping that would ship a *designed-in* halves divergence. Needs an object-backed
scope in the floor, or nothing.

### 2.2 A boxed value knows its KIND but not its declared class **[V]**
Two boxed values of the same primitive kind and different declared classes cannot
be told apart. Same family as **kotlin's mixed Float/Double set key**, **swift's
`Int` vs `Int64`** (`8 as Int64 is Int` is true where `swiftc` says false) and
**kotlin's empty-range `first`**. All four would likely fall to one mechanism: a
nominal type on the value.

### 2.3 Reading an extension-only member through a PROTOCOL-typed variable **[V]**
swift: `let d: Named = D2(); d.shout` binds to the extension in `swiftc` and to
D2's own member here; through the concrete type every engine agrees. Needs the
expression's **static type** — a propagation pass plus a declared-type table for
locals, loop variables and array elements, touching every member-access site in
both grammars. Comparable to 3.1. Same wall as `Optional(3)` (manual 7.9).

### 2.4 A Python `set` member that is unhashable **[U]**
Compared with `==` in the compiled halves and `===` in the interpreter
(`a=[1]; b=[1]; {a,b}` is 1 vs 2); CPython raises `TypeError`. Closing it means
changing the shared `dictFind` contract, not Python.

### 2.5 Ruby and python residue **[V]**
- **A bare `except:` will not catch python's `GeneratorExit`** — costed as 3.3.
- **An abandoned python generator never runs its `finally`** — CPython finalizes
  at collection; we have no finalizer hook and the GC deliberately has none.
- **`d.items()` renders pairs as lists**, the same tuple root cause as 3.1.
- **ruby `Integer#fdiv` on a bignum, and a big meeting a Rational or Complex**,
  promote through the double where MRI is exact — needs a Rational whose parts are
  bigs, a second value model.
- **ruby `equal?`** is missing in all four engines, and it is manual 7.6's
  `is`/`strictEq` under-read: MRI has `1.equal?(1)` true and `"a".equal?("a")`
  false, and a bare `===` diverges between the interpreter and layer 2.
- **ruby `singleton_class`** cannot be matched — MRI's `to_s` embeds the address.
- **ruby `respond_to?` on non-objects** needs a universal-method table; the
  module-object unqualified-name rule is a heuristic whose exact fix is a
  `js_rselfcall(scope, name, args)` extern, ~3 engines × 25 lines.
- **`method(:x).class` is `Proc`** where MRI says `Method`.

### 2.6 php's undefined `$variable` — the blocker is in the floor **[V]**
PHP 8 warns and evaluates to **null**, and the program continues; `b56c2b0` made
every *throwing* case catchable but left this one aborting. **The floor's hook can
only answer a value to THROW, never a value to RETURN.** Three ways out, none
cheap: (a) a `runtime.c` hook, ~15 lines plus its Go twin and php's registration —
and a floor change must land with its layer-2 deletions in ONE sequenced commit;
(b) a `js_scope_has`-guarded read, which **doubles the chain walk on php's hottest
path**; (c) emitter pre-declaration of every `$name` a body mentions, ~30 lines ×
2 grammars with `isset`/`compact`/`global` regression risk. Written at
`js_phmissname`.

### 2.7 lua has no `error`/`pcall` at all **[V]**
Without them "catchable" is vacuous, and the compiled halves' `member 'q' of
undefined` cannot be brought onto the interpreter's `attempt to index a nil value`
— routing through the miss hook would just produce a third wording. A real fix is
a throw/catch surface: 2 grammars + layer 2 + twin.

### 2.8 a generator body's `this` is wrong in the NATIVE build only **[V]**
`function* g(){ yield this }` then `g().next().value` gives the GENERATOR OBJECT
natively and `undefined` in the other two legs. `js_this` is one global slot in
layer 2 and the emitter arms it with the receiver of the `.next()` call. `--cross`
cannot see it (it compares interp vs run). Closing it needs a **per-coroutine
`this`** in the C floor's control block — manual 7.8's highest-risk area.
Deliberately unasserted in both ratchets: `e4c8659` made 2 of 3 legs correct, which
WIDENS the gap, so an assertion would turn `clang-check` and `native-full` red.

### 2.9 kotlin: a user throwable `is Exception` is FALSE in both compiled halves **[V]**
`class Top(m: String) : Exception(m)`, then `Top("x") is Exception`: **true** in
the interpreter, **false** under `llvm.Run` and natively. A live halves divergence
that `--cross` is blind to, found while closing 2.9 and pre-existing. The compiler
half deliberately never links a user class's `__super` to a builtin throwable and
marks the instance `__kthrow` instead (`emitSuperCtor`). The fix is a
`js_ktexccls(name)` extern answering the `ktExcHierarchy` descriptor plus a
`__super` link in `emitSuperLinks`, which would also retire the `__kthrow` marker
and `js_ktexcmatch`'s "intermediate roots accept a non-throwable" hack. Adjacent:
**`Outer.Mid.Deep(...)` — three levels of nesting — does not resolve in the
compiler half at all** (`method call 'Deep' on a function`).

### 2.10 js/ts: three for-in and private-name residues, all agreed-and-wrong **[V]**
Each survives 2.4 and each is convergent across all four engines, so `--cross` is
blind to all three. **Deleting a key during `for (k in o)`** yields `abc` where
node yields `ab` — the interpreter fix is two lines but the compiler fix is a
per-iteration `js_jshas` guard inside `emitForOf`'s block structure, so it is
both-or-neither and a one-sided fix would CREATE a divergence. **The same private
name in a subclass** (`class A { #v } class B extends A { #v }`) is one slot here
and two in node; closing it needs a per-class-BODY key, not a per-name one.
**`for (k in someFunction)`** lists nothing where node lists own properties.

# 3. Costed designs, deliberately not started

Each has been sized. None is small, and the sizing is the expensive part.

### 3.1 A real tuple type for Python — the blocker is `runtime.c:3088` **[V]**
`set_member` accepts a named key only on tag 6; on tag 5 (array) it demands a
numeric index and otherwise dies. So the cheap fix (mark the array with `__tup`)
is **not representable in the engine the native binaries run on**, and a tuple
must become a tag-6 BOX — the float box's ~640 lines again, zero in the C floor.
**It is atomic**: any subset is a `--cross` divergence. Sites:
`python-interpreter.abnf` 17 `Array.isArray` + 49 tuple refs, `python-rt.metajs`
42 `pyIsArr`, `python-to-llvm-ir.abnf` 54 tuple refs, `abnf/jsrt.go` 241
`*jsArray`. Unblocks the `%`-formatting arity heuristic, `BaseException.__str__`,
and hashability as a real rule. **Copy the python FLOAT box, not the tag-14 floor
box** — manual 7.6 says why, and its two surprises are the reusable half.

### 3.2 Part B: moving floor bodies into layer 2 has no landed move **[U]**
The floor→layer-2 upcall costs **120 ns** and is affordable; a nine-iteration
binary search over BOXED ARRAYS costs **3.2 µs, 26× the whole call**. `case_map`
moved whole, passed every gate, and was reverted at 15× / 135× / 2.0×. Two
mechanisms to carry: the 2.0× on a loop that never uppercases was **pinned
interned constants**, and the IR bill is per-language — −5,079 lines of
`runtime.ll` bought **+156,000 checked-in IR lines to delete 104 lines of C**.
`d_pow` and its family **cannot move at all** (manual 7.7). **The direction worth
taking is not another leaf: it is the VARIADIC PROLOGUE**, where `sample` on the
spread-heaviest Go program is dominated by `scope_get` and `ar_block` repacking
args.

### 3.3 A catchable `GeneratorExit` — three walls, in cost order **[V]**
`b086d07` landed `.throw()` and **declined `close()` deliberately**:
`js_try`'s `pending != GEN_EXIT` guard is **shared with fifteen languages**, where
a close IS a return completion and must not be catchable; `GEN_EXIT` is a bare
floor object with **no Python class**, so `except Exception` (which must not catch
it) and bare `except:` (which must) cannot be told apart; and the floor needs
`gen_close(g, sentinel)` with the language supplying the value, reached through the
tag-15 `return` member, which has nowhere to pass one. A new extern → three
engines → plus `coro-poc`.

### 3.4 `Symbol.iterator` is a VALUE-MODEL change **[V]**
There are **no symbols in the value model at all**, and `for..of`, `yield*` and
`js_iterable` all test structurally for a callable `next` in three engines. Needs
a new tag in `runtime.c`, a Go type, `typeof`/`strict_eq`/`keysOf`/dict-key arms
in all three engines, a well-known-symbol registry, and computed-key support in
the class and object-literal emitters. Comparable to 3.1. Buys one thing the
structural test does not: **a user class becoming iterable**.

### 3.5 Coroutines: option A (CPS in the emitter) is the eventual answer **[U]**
What shipped is B (pthread + condvar): **4.1 µs per `next()`** natively against
**0.75 µs** under `llvm.Run`, ~**18.8 KB per live suspended generator**, and an
abandoned generator parks its thread forever. **B1 (`swapcontext`) is dead**:
deprecated on darwin, and `runtime.c` has no `#include`, so `ucontext_t` fields
would be hard-coded byte offsets — a floor that guesses a struct layout is not a
floor. Nothing is wasted by having done B: the extern surface is identical.

# 5. Duplication and dead code

`go run ./tools/shape-scan` is the measure.

> **⚠ A ZERO-KNOB DIFFERENCE LIST IS NECESSARY AND NOT SUFFICIENT — SIZE THE CALL
> SITE, NOT THE BODY.** The fifth pass merged on an empty difference list alone
> and shipped a regression that stood for four commits: `pyIsObj` → `rtIsObj` cost
> **python +41%**, `rbIsObj` **ruby +11%**, `rtjIsIntegral` **java +5.7%** —
> because each is a predicate asked at ~15–30 sites inside an arithmetic path, and
> **in MetaJS a call is a scope frame**. Sharing stayed free for kotlin's `ktIsInt`
> at the *same* body (−0.64%). Identical bodies, different call sites, and only the
> call sites decide. `8de27d7` is the worked example of doing it right: every merge
> there was already a call to a same-cost body in the same pinned scope, and the
> one that was not — folding `runtime-dartswift` — cost python **+1.46%** on its own
> and had to be paid back by the merges. **Measure with `--bench --paired` before
> and after.**

### 5.1 The ruby-side pairs of the `-min 40` tier **[V]**
`pyBRepeat`/`rbRepeat`, `pyIsNone`/`rbIsNil`, `pyTag`/`rbTag` — all three need the
ruby side, and `ruby-rt.metajs` belonged to another agent during `8de27d7`, where a
one-sided merge would have been incoherent. Both languages already import
`runtime.metajs`, so these cost no new import. The tier stands at **47 groups /
750 recoverable** at `-min 40`; the gate's 140-char threshold is unchanged at 15 /
481. Not free, and manual 7.13 says why: `js-rt.metajs` imports `regex.js` only and
`lua-rt.metajs` imports nothing, so anything reaching into those two costs a first
import plus seven hooks.

### 5.2 `sw_kget`/`sw_kset`/`sw_safeget` are duplicated across three GRAMMARS **[V]**
`swift-to-llvm-ir.abnf:2495,2531`, `dart:2437,2451`, `kotlin:2555` (which says
"transferred verbatim" at the site). They are **emitter closures that build basic
blocks**, so `runtime.metajs` cannot hold them; merging means a shared ABNF module
imported by three grammars.

### 5.3 The other five lowered probes can follow kotlin's **[V]**
`a6df3a8` moved kotlin's ten scope helpers into layer 2 and deleted the abort that
stood for the whole ceiling. The same conversion is open for php `isset`, python's
`py_setvar` residue (a containment walk is `scopeHas` + `scopeParent`, ~15 lines
each), swift/dart `sw_kget`/`sw_kset` (~40 each, the same body twice) and ruby
`defined?` (~30). **Each needs the same judgement kotlin's did: keep the hot
local-hit arm inline and route only the MISS paths**, or you pay a scope frame on
the hottest name resolution in the language.

# 6. Tooling and developer experience

### 6.1 `stmtPos` is wrong in python and ABSENT in ruby and php **[V]**
Found by building `-why`, and it breaks **four** consumers at once — `-trace`,
`-cfgraph`, `-callgraph` and `-why` all read the same `js_srcpos` markers.
Measured: a python file whose statements are on lines 1, 2, 3 reports them as
**2, 3, 4**, the last pointing past EOF; kotlin is correct on the same probe.
The cause is that a tag's `up.pos` is the position AFTER the production matched
and python's `SmallLine` ends in `NL`. **It is not the one-liner it looks like**:
moving `stmtPos` down to the alternatives was tried and reverted, because
`PosSmallLine` still consumes the `NL`, and a compound statement's tag ends after
its whole suite — so `If` would mark the END of its block. It wants a rule that
ends at the statement's last token, for both shapes. Separately,
`grep -L stmtPos languages/*-to-llvm-ir.abnf` shows **ruby and php never apply it
at all** — the only two handle-IR languages missing it, so they get no line
attribution in any of the four tools. `stmtPos` returns its argument unchanged
unless `c.tracing` is on, so nothing here can affect ordinary emission.

### 6.2 Two swift abort rows can be promoted to ALL HALVES **[V]**
Closing 1.1 made `tests/swift-abort-test.swift` and
`tests/swift-abort-test-throw.swift` byte-identical on all five legs, and their
own header still says neither is green on all three implementations. The abort
rows are the ONLY thing in the suite that can see an abort message, so promoting
them is worth real care: the first row is currently an interpreter-only
`SHOULD FAIL` and would have to become a `SHOULD ABORT, ALL HALVES` `-exe` row,
which changes what it tests. Needs `.vscode/launch.json` plus `ABORT_ALL_ROWS`
(14) and possibly `ABORT_ROWS` (16) in `tests/gates.sh`, and the stale header
comment in both files retired.

### 6.3 Three claims in `docs/abnf-dialect-gotchas.md` are STALE **[V]**
`string.split`, `array.join` and `array.length =` are all listed as absent from
the frozen engine and all three WORK — measured in both script hosts and in a
native MetaJS binary. `tools/metajs-lint` deliberately does not carry them as
rules (a linter that fires on working code gets switched off), so the document is
now the only place the wrong claim survives. Confirmed still absent: `splice`,
`lastIndexOf`, `toPrecision`, `for...in`; and `indexOf(x, from)` **accepts and
silently ignores** its from-index.

### 7.1 JS reject side — three coherent groups
**Assignment targets (8 files)**: a form that is not a LeftHandSideExpression used
as one — `() => {}()`, `({a:this}=0)`, `switch (c) { default: default: }`.

> **A warning that was paid for once.** The neighbouring `for(([a]) of 0)`,
> `for((1 + 1) in list)`, `1++` and `1--` files look like the same group and are
> **not**. Guards refusing a parenthesized non-target for-head and a numeric
> postfix operand won 7 reject files and cost **8 must-parse** files — `0++`,
> `for(([0]) in 0)`, `for((0) of 0)` under `early/`, which are early errors that
> must PARSE. Reverted. **Do not reintroduce without measuring both rows.**

**Goal symbol, Script vs Module (17 files)**: a `:script` guard cannot see the
file name — `abnf/parserscript.go` exposes only `getSrc`/`setSrc`/`getSdx`/
`setSdx`/`peek`. A Script may not contain `import`/`export` at all (6 files); a
Module is always strict, reserves `await`, and forbids Annex B `<!--` comments
(8 `*.module.js` files). Needs a filename or module-goal boolean exposed to guards
in **both** engines. **Note what `2a59980` settled**: these grammars implement the
**Script** goal, evidenced by `HtmlCloseAtStart` accepting Annex B comments, sloppy
mode by default, and top-level `return` being a syntax error.

**Unicode identifiers (3 files)**: `IdUni`/`IdUniPart` approximate
ID_Start/ID_Continue with a few wide ranges, so `var 🀂` is wrongly accepted. Real
tables would also serve Java, C#, Kotlin, Swift and Python, so they belong in
`languages/lib/ident.abnf` as an **additive** change — do not alter the existing
ASCII `Id` or `KwEnd`. A compressed range list checked by a `:script` guard is
probably cheaper than thousands of ABNF ranges.

### 7.2 Other corpora
- **batch 0 of 2** — the ratchet is FULL at 24 sections, but neither wine file
  parses. No statistical weight; still a stark gap.
- **swift 80%** — an 18-file lexical tail: suppressed conformances, `@isolated(any)`
  in a type, `borrowing` as a bare type name, raw multi-line strings with 3+ pound
  levels, parameter packs, form feed and NBSP. **For Swift, lexical work buys
  corpus and semantic work buys ratchet — they are nearly disjoint.**
- **dart 96.6%** — ~53 of the remaining 117 are multitests whose surviving `//#`
  variants the files themselves label "syntax error".
- **python 92.9% and ruby 98.1%** are at their hard caps.
- **go / php / csharp / java / c** have ordinary long tails.

---

# 8. Settled — do not re-open

**~194 items closed in seventeen rounds** (`e284185`..HEAD). Verdicts only, and
only the REASON, because the reason is the only part that stops a re-open. No
dates, no order, no commit archaeology — the detail is in git and the durable
lessons are in the manual (ch. 4 and 7).

**Measured, then declined.** `i in arr` after `delete` · iterator close on
`throw` (a depth stack cannot survive `for await`) · a catchable unknown-name
error in java/go/swift/csharp/metajs (COMPILE-time in each real toolchain) ·
kotlin `:script` → `!"token"` · `case_map` into layer 2 (135×) · retiring the Go
twin (manual 7.11) · dart `x as T` as a throwing cast · an mtime skip for the
generators (a `git checkout` would make a stale `.ll` report CLEAN) · matching
swiftc's **exit 133** on a trap (its stdout discard is libc block buffering, not a
language property — the same binary on a terminal prints everything; and our
interpreter legs run *inside* `mec`, so `raise()` would kill the metacompiler and
at most 1 of 3 legs could match, widening the halves gap).

**The ITEM was wrong, oracle in hand.** `(super as any)[k]` — real `tsc` rejects
it (TS1034) · dart type-parameter erasure — Dart reifies · the C# generated
message on every derived type — fixes one name, breaks twenty-six · **python
integer arithmetic past 2^53 — already exact** (70 probe rows over 2^53, 2^60 and
negatives agree with CPython on all four legs) · **ruby `%g` — already
implemented** (the real gap was the `#` flag) · **python's `&` int32-signed —
already correct**, its own example row included · **java's divide-by-zero message
— already right** (`/ by zero`, catchable, already asserted) · **kotlin's nested
throwable name would "change every `Name@1`" — it does not** (`Name@1` reads
`__name`; only the throwable path reads `__bname`).

**The prescribed FIX was wrong.** ruby's malformed formats: "emit % and rescan"
would have converged all three engines on behaviour MRI rejects, turning a halves
divergence into agreed-and-wrong · ruby's `round(n)` via `rtExactDec`: the true
expansion of 2.675 is 2.67 and MRI answers 2.68, because `round_half_up` carries
a midpoint correction · swift's abort message via a floor change: Go already
guards in front of the shared layer, so `runtime.c` needed no edit at all.

**Examined and dropped.** The `.expected` fixture path · `getattr()`-bound python
builtins with kwargs · `glibc < 2.34`'s `-ldl -lpthread` · `mod.py` headroom.

**Closed this round, and what each one actually was.** ruby `Float#round/floor/
ceil(n)` — MRI's own numeric.c algorithms, including the `int_round_zero_p` cliff
that overrides the arithmetic · ruby malformed formats — now RAISE with MRI's
messages, and the blast radius of raising turned out to be zero · kotlin
out-of-range indexing — six defects, not two, and the bounds check measured FREE
because it folds into work the read already does · js/ts private names and
`for..in` — one root cause (the interpreters filed `#x` unmangled), and the
compilers were wrong in a second place nobody had checked · swift Array/String
members — nineteen of them, plus six adjacent defects, and SECTION 36 runs under
real `swiftc` · swift's abort message — Go already guards in front of the shared
layer, so the floor was never touched · `--full`'s total silently discarding a
gapped half's assertions · `tools/floor-coverage` (330 of 345 floor bodies
reached) · `tools/metajs-lint`, which found two live defects on its first run,
both wrong answers in shipped native binaries that no dynamic gate could see ·
`-why`, which was justified by answering three questions BY HAND first and
getting two of them wrong.

**Moved to the manual.** The scope API (7.3) · per-language divergences (7.9) ·
the byte-vs-character `printf` rule (7.7).
