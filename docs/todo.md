# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Rebuilt 2026-08-10 at `8de27d7`, after the fifteenth round. Completed work lives in
git, not here — see [§9](#9-settled--do-not-re-open) for the one-paragraph version.

Two conventions, and they are the point of the file:

> **[V]** — a probe or grep was run for this list, on this commit.
> **[U]** — inherited from an older note and NOT re-checked. A lead, not a fact.

> **Treat an item as a place to start probing, never as a specification.** In the
> fifteenth round **nine of ten items were wrong as written**: two were already
> closed (one 123 commits earlier), two understated the defect, one named the wrong
> symptom, one had a mistaken cost estimate, and one carried a suggested fix that
> would have made two languages worse. **Every time, the defect underneath was real
> and bigger.** Probe the FIX as well as the DEFECT. **Item NUMBERS are re-assigned
> every rebuild** — a citation of `docs/todo.md 1.4` is provenance.

Baseline every item must preserve (`tests/gates.sh`, ~2.5 min — `--serial` on a
loaded machine, `--bench` for anything touching layer 2 or the floor):

```
matrix 383/383 (16 SHOULD ABORT rows, 14 of them ALL HALVES) · --full 8,927
assertions, 0 halves disagree · --cross 121/0 · clang-check 16/16 none held
go test ok · gen-all 15/15 clean · native-full 20 languages, 0 failures
shape-scan 15 groups / 481 recoverable · -freeze a fixed point
bench: 15 rows, no row outside its spread · tests/coro-poc/build.sh byte-identical
```

**Before measuring anything, read the manual's chapter 4 and
`tests/bench/baseline.txt`'s header** — instructions not wall clock, medians over
layout draws, `--paired`, corroborate with an exact counter.

---

# 1. Correctness, with an oracle on this machine

### 1.1 python: integer ARITHMETIC does not promote past 2^53 **[U]**
The literals do (`506000a`). `9007199254740992 + 1` is unchanged; 54 of 55
differing probe rows are the `*` column. The site is the plain `l+r`/`l-r`/`l*r`
arms in all three engines — the hot path of every Python program, where a naive
guard measured **+9.5%** — and `*` needs the exact product, the expensive half.

### 1.2 kotlin's abort text is agreed-and-wrong against the JVM **[V]**
All three engines now say `abort probe` where Kotlin/JVM says
`Exception in thread "main" java.lang.RuntimeException: abort probe`. This is
exactly the class `800ef57` closed for java/csharp/go, and kotlin was not in that
item. No `kotlinc` here, but kotlin targets the JVM and `java` 24.0.2 settles the
format. Same for **swift's exit code and stream**: we exit 1 on stderr where
`swiftc` traps with SIGTRAP (133) — converging needs the floor to raise a signal.

### 1.3 C# `new Exception()` gives Message = the class name **[V]**
.NET's parameterless constructor generates
`"Exception of type 'System.Exception' was thrown."`. Three engines, one of them
`abnf/jsrtcsharp.go`. Spec-cited only — there is no dotnet on this machine.

### 1.4 swift's `integer divide by zero` is agreed and wrong **[V]**
`swiftc` says `Fatal error: Division by zero`. Agreed in all three engines, so
`--cross` is blind. **The message is SHARED across languages**, so converging it
is a cross-language decision and not a swift edit — that is why `d3ee2b2` left it.

### 1.5 ruby: `%a`/`%A`, the argument selectors, and two float gaps **[V]**
After `d28d8ca` closed `%B`/`%u`/`%c`/`%p` and character-counted widths, what is
left is: **`%a`/`%A` hex-float**, missing in all three (needs a `frexp`-then-×16
digit loop written three times — layer 2 has no access to a double's bits);
**the argument selectors** `%1$s`, `%<name>s`, `%{name}` and `*`, which DIVERGE
between the halves (`"%1$s-%2$s" % ["a","b"]` is `a- b` in the interpreter and
`$s- $s` in the compiled halves) and need argument consumption restructured in all
three; **`Float::INFINITY` is `uninitialized constant`** in the interpreter while
`1.0/0.0` works; and **`(1.0/0.0).to_i` answers `Infinity`** where MRI raises
`FloatDomainError` — one item with the formatter, not two, and `9102c91`
deliberately preserved the existing answer rather than inventing one.

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

### 2.4 dart: a type PARAMETER or `typedef` name in a pattern does not match **[V]**
`switch(v){ T t => … }` inside `class Box<T>` answers `no`, and `f is IntFn` is
false. Pre-existing in `dartIsType` and measured at base in BOTH halves — typed
patterns now inherit it, where before `b0eb4c4` they matched by accident. Needs
the emitter and interpreter to know which names are type parameters and erase them
to `dynamic`. Adjacent, both left deliberately: **`x as T` is an identity cast**
that never throws (changing it changes every `as` in the language), and
**`o as Shape is Shape` does not parse** (`TypeTestExpr` allows one postfix).

### 2.5 A Python `set` member that is unhashable **[U]**
Compared with `==` in the compiled halves and `===` in the interpreter
(`a=[1]; b=[1]; {a,b}` is 1 vs 2); CPython raises `TypeError`. Closing it means
changing the shared `dictFind` contract, not Python.

### 2.6 Ruby and python residue **[V]**
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

### 2.7 php's undefined `$variable` — the blocker is in the floor **[V]**
PHP 8 warns and evaluates to **null**, and the program continues; `b56c2b0` made
every *throwing* case catchable but left this one aborting. **The floor's hook can
only answer a value to THROW, never a value to RETURN.** Three ways out, none
cheap: (a) a `runtime.c` hook, ~15 lines plus its Go twin and php's registration —
and a floor change must land with its layer-2 deletions in ONE sequenced commit;
(b) a `js_scope_has`-guarded read, which **doubles the chain walk on php's hottest
path**; (c) emitter pre-declaration of every `$name` a body mentions, ~30 lines ×
2 grammars with `isset`/`compact`/`global` regression risk. Written at
`js_phmissname`.

### 2.8 lua has no `error`/`pcall` at all **[V]**
Without them "catchable" is vacuous, and the compiled halves' `member 'q' of
undefined` cannot be brought onto the interpreter's `attempt to index a nil value`
— routing through the miss hook would just produce a third wording. A real fix is
a throw/catch surface: 2 grammars + layer 2 + twin.

### 2.9 a generator body's `this` is wrong in the NATIVE build only **[V]**
`function* g(){ yield this }` then `g().next().value` gives the GENERATOR OBJECT
natively and `undefined` in the other two legs. `js_this` is one global slot in
layer 2 and the emitter arms it with the receiver of the `.next()` call. `--cross`
cannot see it (it compares interp vs run). Closing it needs a **per-coroutine
`this`** in the C floor's control block — manual 7.8's highest-risk area.
Deliberately unasserted in both ratchets: `e4c8659` made 2 of 3 legs correct, which
WIDENS the gap, so an assertion would turn `clang-check` and `native-full` red.

### 2.10 js: three object-model holes left after `bd80f09` **[V]**
- **`Object.keys(<class object>)` is wrong in BOTH halves, differently.** node says
  `["sf"]` (static fields; methods are non-enumerable). Interpreter `[]`, compiler
  `m,ionly,g,iv`. Needs a descriptor arm in `js_keys`. Nothing asserts it.
- **`e.constructor` is `undefined`** where node hands back the class.
- **`(super.m).call(x)` uses the read-time receiver**, not `x` — consistent with
  the engine's existing bound-method shim, divergent from node.
- **`(super as any)[k]` does not parse in either typescript grammar** (a bare
  `super` is not a Primary, so the cast has nothing to apply to). Real TS accepts it.

### 2.11 swift and php/kotlin residue from the abort round **[V]**
- **A top-level `defer` is unsupported in swift's interpreter half**, and the
  HOST error it dies with is itself a goja-vs-`-frozen` divergence
  (`TypeError: Not a function` vs `call of a non function value`). The compiled
  half handles it, so this is also a live halves divergence.
- **`js_kindex` diverges for swift**: the Go twin says
  `string index %d out of range for %q` and answers a `jsChar`; `swift-rt.metajs`
  says `string index out of range` and answers a String.
- **Two unimplemented swift Array members disagree in their REFUSAL text**
  (`popLast()`, `Array(s)`) — swiftc supports both.
- **kotlin `data class` thrown**: `llvm.Run` prints `P(x=1)`, the native binary
  `[object Object]` — the floor's `to_string` flattens every tag-6 object.
  Unreachable from valid Kotlin, and it lives in `jsrt.go` + `runtime.c`.
- **php `throw 1.5`**: `[object Object]` in the interpreter (its float is a
  `{__flo}` box), `1.5` in both compiled halves. Unreachable from valid PHP.

### 2.12 bash expansion residue, all pre-existing **[V]**
`${a[@]@A}` needs the whole `declare -p` rendering (interp `a='pa' a='pb'`, the
compiled halves still leak `a=$'\002pa\001pb\001'`); **`${a[@]@a}`** is a halves
divergence (interp `a a`, compiled `a`); **`${!z}` with `z=0`** is another
(`rt_getvar_byname` cannot reach positional parameters); **a command NAME from an
expansion is not field-split** in either half (`cmd="echo hi"; $cmd` dies); and
**`echo`'s flags are read only from compile-time constants** in the compiler, so
`opts="-n"; echo $opts x` prints the flag as an argument.

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

# 4. Test coverage and harness

### 4.1 `Math.random` cannot be pinned until it is fixed **[V]**
`typeof Math.random` is `function` under goja and `undefined` under `-frozen` in
the grammar host. An assertion on it is red in one engine **by construction**, so
it would read as `FROZEN-DIFF` rather than as a pin. It needs a fix in `abnf/`
first — bind it identically in all three engines, or drop it from goja — and only
then an assertion. Same class as the `Math` *members* divergence `1367c23`
recorded and left.

### 4.2 Sections after `===== END SECTIONS =====` are invisible to the gap probe **[V]**
`full_probe`'s awk `exit`s at that marker, so python's sections 37–41, ruby's
40–43 and kotlin's 76–81 can never be isolated or deleted-and-retried: if one
breaks, the probe reports "error at line N outside the sections" and gives up.
Either move the marker to the end of each file or teach the awk to ignore it.

### 4.3 swift's compiler half lacks `uppercased()`/`lowercased()` **[V]**
The interpreter has both. A live halves CAPABILITY gap that `--cross` does not see
because no test program calls it — and the reason swift got no case-mapping pins
in `e1816b3` while kotlin, ruby and python did.

### 4.4 `--full`'s total still includes a killed language's surviving half **[V]**
`e1816b3` made a kill loud and made it exit 1, but the printed total still counts
the half that did finish. Annotating it was chosen over zeroing it, because
zeroing would make the number *more* wrong; recorded in case the other choice is
preferred later.

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

### 6.1 Per-language coverage of the floor **[U]**
The instrumentation that found 24 unreached floor bodies was written once and
thrown away. Keeping it as `tools/floor-coverage` would make "is this body dead or
untested?" answerable at any time — **it is the question that gates 3.2**, because
a body no test reaches is a body no move can be validated against.

### 6.2 A `--why` flag for the emitters **[U]**
Understanding which extern an emitter chooses for an operator means reading
thousands of lines of grammar, repeatedly. A flag printing the extern chosen per
AST node would make layer-2 work dramatically faster.

### 6.3 Make MetaJS's dialect gaps loud **[U]**
No exponent literal, no `toPrecision`, typed locals, ASI differences, **and
`keysOf` is not bound for grammar scripts at all** (only for layer-2 MetaJS —
found by running it, goja answers `ReferenceError`). Each is discovered by a
confusing failure. A `-verify` pass over `lib/*.metajs` that names them would pay
for itself immediately. `docs/abnf-dialect-gotchas.md` lists them; nothing
enforces them.

# 7. Reference-corpus gaps

Recorded 2026-07-27 and **not re-measured since — treat every number as [U]**.
Corpus: lua 100%, js 98.9, ruby 98.1, dart 96.6, typescript 93.7, python 92.9,
java 92.1, kotlin 91.0, bash 88.6, c 87.3, csharp 84.6, php 83.1, go 81.1,
swift 80.0, batch 0/2.

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

# 9. Settled — do not re-open

**~163 items closed in fifteen rounds** (`e284185`..`8de27d7`). Detail is in git
history; the durable lessons are in the manual's chapters 4 and 7. Only verdicts
live here, because the *reason* is the only part that stops a re-open.

**Declined on a measurement, not a preference.** `i in arr` after `delete` (needs
the shared `*jsArray` changed for every language, declined twice) · iterator close
on `throw` (built, matched node, reverted — a depth stack cannot survive
`for await`; `itc20`/`itc24` pin it) · a catchable unknown-name error in
java/go/swift/csharp/metajs (COMPILE-time in each real toolchain, so nothing runs
to catch) · kotlin's `:script` guards → `!"token"` (equivalent by intention only,
and a mismatch silently changes what parses) · `case_map` into layer 2 (reverted at
135×) · retiring the Go twin (`llvm.Run` structurally cannot execute the C floor —
manual 7.11).

**Not worth doing, examined and dropped.** The `.expected` fixture path is
exercised by one program each (brainfuck, calculator) — no action until a third
exists · a `getattr()`-bound python builtin cannot carry keyword arguments: it is
documented at all three sites and its sibling refusal is deliberate, so it belongs
in the manual's permanent-divergence list · `glibc < 2.34`'s `-ldl -lpthread`
(untestable on darwin) · `mod.py` headroom (a size class, not a threshold).

**Moved, not deleted.** The scope-API note, the per-language recorded divergences
and the byte-vs-character `printf` rule now live in the manual (7.3, 7.9, 7.7),
which is where they belong.
