# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Rebuilt 2026-08-09 at `7af8514`; **~153 items closed since `e284185`**, in fourteen
rounds. The fourteenth closed fourteen items (`3158f1d`..`5dba9b7`) and is the
round that added the abort-text gate, so numbering below has gaps until the next
rebuild.

Two conventions, and they are the point of the file:

> **[V]** — a probe or grep was run for this list, on this commit.
> **[U]** — inherited from an older note and NOT re-checked. A lead, not a fact.

> **Treat an item as a place to start probing, never as a specification.** In the
> thirteenth round all ten were wrong as written, and the defect underneath was
> real and bigger every time. Probe the FIX as well as the DEFECT. **Item NUMBERS
> are re-assigned every rebuild** — a citation of `docs/todo.md 1.4` is provenance.

Baseline every item must preserve (`tests/gates.sh`, ~2.5 min — `--serial` on a
loaded machine, `--bench` for anything touching layer 2 or the floor):

```
matrix 383/383 (16 SHOULD ABORT rows, 10 of them ALL HALVES) · --full 8,733
assertions, 0 halves disagree · --cross 121/0 · clang-check 16/16 none held
go test ok · gen-all 15/15 clean · native-full 20 languages / 37 programs, 0 held
shape-scan 15 groups / 481 recoverable · -freeze a fixed point
bench: 15 rows, no row outside its spread · tests/coro-poc/build.sh byte-identical
```

**Before measuring anything, read the manual's chapter 4 and
`tests/bench/baseline.txt`'s header** — instructions not wall clock, medians over
layout draws, `--paired` inside a row's spread, corroborate with an exact counter.

---

# 1. Correctness, with an oracle on this machine

Chapter 1 was emptied again in the fourteenth round: nine of its eleven items are
closed (`3158f1d`..`5dba9b7`). **Both survivors are [U]** — inherited and not
re-checked — which matters, because in the fourteenth round *every* item an agent
touched was wrong as written and the defect underneath was real and bigger each
time.

### 1.9 python: integer ARITHMETIC does not promote past 2^53 **[U]**
The literals do (`506000a`). `9007199254740992 + 1` is unchanged; 54 of 55
differing probe rows are the `*` column. The site is the plain `l+r`/`l-r`/`l*r`
arms in all three engines — the hot path of every Python program, where a naive
guard measured **+9.5%** — and `*` needs the exact product, the expensive half.

### 1.10 ruby's three integer directives **[U]**
`%d`/`%x`/`%o`/`%b` of a float past 2^53; negatives under `%x`/`%o`/`%b`, which
need MRI's infinite two's-complement `..fb` notation; precision on an integer
directive being ignored. **All three: both halves agree, so `--cross` is blind.**

# 2. Cross-engine defects and latent traps

### 2.2 `with` is the one js construct that cannot be lowered honestly **[V]**
The INTERPRETER can implement it exactly (`scopes.push(o)` — scopes there are
plain objects). Both compiled engines cannot: the floor's scope is a name/value
pair of arrays with a hash index, **not object-backed**, so the only lowering is
snapshot-in/write-back, which cannot see a property added inside the body.
Shipping that would ship a *designed-in* halves divergence. Needs an object-backed
scope in the floor, or nothing.

### 2.3 A boxed value knows its KIND but not its declared class **[V]**
Two boxed values of the same primitive kind and different declared classes cannot
be told apart. Same family as **kotlin's mixed Float/Double set key**, **swift's
`Int` vs `Int64`** (`8 as Int64 is Int` is true where `swiftc` says false) and
**kotlin's empty-range `first`**. All four would likely fall to one mechanism: a
nominal type on the value.

### 2.4 Reading an extension-only member through a PROTOCOL-typed variable **[V]**
swift: `let d: Named = D2(); d.shout` binds to the extension in `swiftc` and to
D2's own member here; through the concrete type every engine agrees. Needs the
expression's **static type** — a propagation pass plus a declared-type table for
locals, loop variables and array elements, touching every member-access site in
both grammars. Comparable to 3.1. Same wall as `Optional(3)` (manual 7.9).

### 2.5 A Python `set` member that is unhashable **[U]**
Compared with `==` in the compiled halves and `===` in the interpreter
(`a=[1]; b=[1]; {a,b}` is 1 vs 2); CPython raises `TypeError`. Closing it means
changing the shared `dictFind` contract, not Python.

### 2.7 Ruby and python residue **[V]**
- **A bare `except:` will not catch python's `GeneratorExit`** — costed as 3.3.
- **An abandoned python generator never runs its `finally`** — CPython finalizes
  at collection; we have no finalizer hook and the GC deliberately has none.
- **`d.items()` renders pairs as lists**, the same tuple root cause as 3.1.
- **ruby `Integer#fdiv` on a bignum, and a big meeting a Rational or Complex**,
  promote through the double where MRI is exact — needs a Rational whose parts are
  bigs, a second value model.
- **ruby `equal?`** is missing in all four engines, and it is manual 7.6's
  `is`/`strictEq` under-read: MRI has `1.equal?(1)` true and `"a".equal?("a")`
  false, and a bare `===` diverges between the interpreter and layer 2. Needs its
  own oracle pass.
- **ruby `singleton_class`** cannot be matched — MRI's `to_s` embeds the object
  address.
- **ruby `respond_to?` on non-objects** (`"a".respond_to?(:upcase)`) needs a
  universal-method table rather than dispatcher arms; and the module-object
  unqualified-name rule is a heuristic whose exact fix is a `js_rselfcall(scope,
  name, args)` extern, ~3 engines × 25 lines.
- **`method(:x).class` is `Proc`** where MRI says `Method`.

### 2.8 A `getattr()`-bound python builtin cannot carry keyword arguments **[V]**
`getattr(xs,"sort")(reverse=True)` differs from `xs.sort(reverse=True)`: a
signature-less builtin receives its keyword dict as a trailing POSITIONAL.
Documented at all three sites. Related and deliberate: `sorted()` refuses
`key=`/`reverse=` loudly, because they are keyword-only in CPython.

### 2.9 php's undefined `$variable` — the blocker is in the floor **[V]**
PHP 8 warns and evaluates to **null**, and the program continues; `b56c2b0` made
every *throwing* case catchable but left this one aborting. **The floor's hook can
only answer a value to THROW, never a value to RETURN.** Three ways out, none
cheap: (a) a `runtime.c` hook, ~15 lines plus its Go twin and php's registration —
and a floor change must land with its layer-2 deletions in ONE sequenced commit;
(b) a `js_scope_has`-guarded read, which **doubles the chain walk on php's hottest
path**; (c) emitter pre-declaration of every `$name` a body mentions, ~30 lines ×
2 grammars with `isset`/`compact`/`global` regression risk. The reason is written
at `js_phmissname`.

### 2.10 lua has no `error`/`pcall` at all **[V]**
Without them "catchable" is vacuous, and the compiled halves' `member 'q' of
undefined` cannot be brought onto the interpreter's `attempt to index a nil value`
— routing through the miss hook would just produce a third wording. A real fix is
a throw/catch surface: 2 grammars + layer 2 + twin.

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
`*jsArray`. Unblocks: the `%`-formatting arity heuristic, `BaseException.__str__`,
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
floor object with **no Python class**, so `except Exception` (which must not
catch it) and bare `except:` (which must) cannot be told apart; and the floor
needs `gen_close(g, sentinel)` with the language supplying the value, reached
through the tag-15 `return` member, which has nowhere to pass one. A new extern →
three engines → plus `coro-poc`.

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

# 4. Test coverage that is missing

### 4.2 The goja full-case-mapping fix is pinned in metajs only **[V]**
`1367c23` routed goja's `toUpperCase`/`toLowerCase` through the same
`strings.ToUpper` the frozen engine uses, closing the divergence in **every**
language's interpreter half at once. Only `tests/metajs-test-full.js` asserts it.
Related and harmless-but-invisible: `Math.random` is `function` under goja and
`undefined` under `-frozen` in the grammar host.

### 4.3 `./test.sh --full` is misleading by hand **[V]**
js reads `MISMATCH: 576 RUN-FAILED` under the default 120 s timeout — the
`-frozen` js interpreter leg simply overruns. Pre-existing, and `gates.sh` is
green only because it passes `--timeout 600` itself. Either raise the default or
make the bare command say why.

### 4.4 The `.expected` fixture path is used by ONE program **[V]**
Only brainfuck uses a fixture and only calculator uses `exit-only`. The mechanism
is barely exercised — if a third fixture is added, check `--bless` still does the
right thing.

# 5. Duplication and dead code

`go run ./tools/shape-scan` is the measure.

### 5.1 The `-min 40` shape tier: 15 groups, 111 recoverable lines **[V]**

> **⚠ A ZERO-KNOB DIFFERENCE LIST IS NECESSARY AND NOT SUFFICIENT — SIZE THE CALL
> SITE, NOT THE BODY.** The fifth pass merged on an empty difference list alone
> and shipped a regression that stood for four commits: `pyIsObj` → `rtIsObj` cost
> **python +41%**, `rbIsObj` **ruby +11%**, `rtjIsIntegral` **java +5.7%** —
> because each is a predicate asked at ~15–30 sites inside an arithmetic path, and
> **in MetaJS a call is a scope frame**. Sharing stayed free for kotlin's
> `ktIsInt` at the *same* body (−0.64%). Identical bodies, different call sites,
> and only the call sites decide. **Measure with `--bench` before and after.**

The safe half is **INTRA-file** duplication, mergeable with no import and no
shared-module decision: `k1Arg`/`k2Arg`/`k3Arg`, `jvArg`/`jxArg`,
`pyEArgAt`/`pyFArgAt` — one 4-line body, 8 copies — plus
`k1Integral`/`k1Numeric`/`k3Integral` and two nullish pairs. ~32 lines.
Cross-language ones whose home is already imported, so they are free today:
`pyBRepeat`/`rbRepeat`, `pyIsNone`/`rbIsNil`, `pyTag`/`rbTag`, `csMaxI64`/`jlMaxL`.
Not free, and why, is manual 7.13: `js-rt.metajs` imports `regex.js` only and
`lua-rt.metajs` imports nothing.

### 5.3 `sw_kget`/`sw_kset`/`sw_safeget` are duplicated across three GRAMMARS **[V]**
`swift-to-llvm-ir.abnf:2495,2531`, `dart:2437,2451`, `kotlin:2555` (which says
"transferred verbatim" at the site). They are **emitter closures that build basic
blocks**, so `runtime.metajs` cannot hold them; merging means a shared ABNF module
imported by three grammars.

### 5.4 The other five lowered probes can follow kotlin's **[V]**
`a6df3a8` moved kotlin's ten scope helpers into layer 2 and deleted the abort that
stood for the whole ceiling. The same conversion is open for php `isset`, python's
`py_setvar` residue (a containment walk is `scopeHas` + `scopeParent`, ~15 lines
each), swift/dart `sw_kget`/`sw_kset` (~40 each, the same body twice) and ruby
`defined?` (~30). **Each needs the same judgement kotlin's did: keep the hot
local-hit arm inline and route only the MISS paths**, or you pay a scope frame on
the hottest name resolution in the language.

# 7. Tooling and developer experience

### 7.3 Per-language coverage of the floor **[U]**
The instrumentation that found 24 unreached floor bodies was written once and
thrown away. Keeping it as `tools/floor-coverage` would make "is this body dead or
untested?" answerable at any time — **it is the question that gates 3.2**, because
a body no test reaches is a body no move can be validated against.

### 7.4 A `--why` flag for the emitters **[U]**
Understanding which extern an emitter chooses for an operator means reading
thousands of lines of grammar, repeatedly. A flag printing the extern chosen per
AST node would make layer-2 work dramatically faster.

### 7.5 Make MetaJS's dialect gaps loud **[U]**
No exponent literal, no `toPrecision`, typed locals, ASI differences, **and
`keysOf` is not bound for grammar scripts at all** (only for layer-2 MetaJS —
found by running it, goja answers `ReferenceError`). Each is discovered by a
confusing failure. A `-verify` pass over `lib/*.metajs` that names them would pay
for itself immediately. `docs/abnf-dialect-gotchas.md` lists them; nothing
enforces them.

# 8. Reference-corpus gaps

Recorded 2026-07-27 and **not re-measured since — treat every number as [U]**.
Corpus: lua 100%, js 98.9, ruby 98.1, dart 96.6, typescript 93.7, python 92.9,
java 92.1, kotlin 91.0, bash 88.6, c 87.3, csharp 84.6, php 83.1, go 81.1,
swift 80.0, batch 0/2.

### 8.1 JS reject side — three coherent groups
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
mode by default, and top-level `return` being a syntax error (which rules out
CommonJS, since node legalises it in the wrapper).

**Unicode identifiers (3 files)**: `IdUni`/`IdUniPart` approximate
ID_Start/ID_Continue with a few wide ranges, so `var 🀂` is wrongly accepted. Real
tables would also serve Java, C#, Kotlin, Swift and Python, so they belong in
`languages/lib/ident.abnf` as an **additive** change — do not alter the existing
ASCII `Id` or `KwEnd`. A compressed range list checked by a `:script` guard is
probably cheaper than thousands of ABNF ranges.

### 8.2 Other corpora
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

# 9. Settled — do not re-open

**~153 items closed in fourteen rounds** (`e284185`..`5dba9b7`); detail in git
history, lessons in the manual's chapter 4. Only verdicts stay — the reason is the
only part that stops a re-open. **`i in arr` after `delete`**: needs the shared
`*jsArray` changed for every language, declined twice. **Iterator close on
`throw`**: built, matched node, reverted — a depth stack cannot survive `for
await`; needs a per-coroutine one, `itc20`/`itc24` pin it. **Catchable
unknown-name error in java/go/swift/csharp/metajs**: COMPILE-time in each real
toolchain, so nothing runs to catch. **kotlin's `:script` guards → `!"token"`**:
equivalent by intention only, and a mismatch silently changes what parses.
**glibc < 2.34's `-ldl -lpthread`**: untestable on darwin. **`mod.py` headroom**:
a size class, not a threshold. The scope API note and the per-language recorded
divergences moved to the manual (7.3, 7.9), which is where they belong.
