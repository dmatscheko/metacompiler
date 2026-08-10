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

> **Treat an item as a place to start probing, never as a specification.** THREE
> rounds running, the list has been wrong more often than right, and ALWAYS in the
> direction of understating the defect.
>
> The eighteenth round is the strongest case yet: of **eleven claims across eight
> items, nine were wrong as written**, and every underlying defect was real and
> larger. One item gave a java error message that belongs to a DIFFERENT RECEIVER
> (`"ab".substring(5)` is `…for length 2`; the recorded `length 6` is
> `"abcdef".substring(7)` — two rows merged in transcription). One said a fix
> "means changing the shared `dictFind` contract"; it does not. One said swift's
> two abort probes "become byte-identical once value semantics land"; they already
> were, and only the stale headers survived. One named three js residues, of which
> the third was not a for-in bug at all — **a function could not carry own
> properties**, which worked under goja and ABORTED under `-frozen`.
>
> The seventeenth and sixteenth had the same shape (see §8 for their verdicts).
>
> **Every time, the defect underneath was real and bigger.** So: **probe the FIX
> as well as the DEFECT**, and re-probe any **[U]** item BEFORE assigning it, not
> during. **Item NUMBERS are re-assigned every rebuild** — a citation of
> `docs/todo.md 1.4` is provenance, not an address.
>
> The corollary that keeps paying: **three agents declined an instruction with a
> measurement and were right every time**, and two partial results that were sent
> back turned up further defects the fix itself had created. Write the brief so
> declining is expected.

Baseline every item must preserve (`tests/gates.sh`, `--serial` on a loaded
machine, `--bench` for anything touching layer 2 or the floor):

```
matrix 385/385 (17 SHOULD ABORT, 15 of them ALL HALVES) · --full 9,468
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

All five were measured by an agent during the eighteenth round, on the tree that
became `3610ad6`, while closing something else. They are findings, not leads.

### 1.1 swift: a closure capturing a `for-in` variable shares ONE binding **[V]**
`for i in 0..<3 { fs.append({ i }) }` then `fs[0]()`, `fs[2]()` gives `3 3` in
both compiled halves and `0 2` in the interpreter; `swiftc` says `0 2`. **A live
halves divergence `--cross` has never reached**, and the interpreter is the
correct side. Pre-existing at `22e7b44`.

### 1.2 ruby: `to_c` is missing in both compiled halves **[V]**
`method call 'to_c' on a number`, while the interpreter answers. A live halves
divergence at base. Adjacent, and agreed-and-wrong rather than divergent:
`"%d" % [nil]` is `"0"` where MRI raises `TypeError: can't convert nil into
Integer`, and `Complex#abs` is `sqrt(re²+im²)` rather than `hypot`, so it
overflows where MRI does not.

### 1.3 ruby: `proc { }` / `lambda { }` with an EMPTY block does not parse **[V]**
Parsed as the bare name plus an empty **Hash literal** in all three engines — the
interpreter says *unknown Hash method: lambda?*, both compiled halves *variable
not defined: proc*. MRI reads a block. Reproduced at `22e7b44`; ruby's SECTION 47
steps around it and says so at the site.

### 1.4 js: `f.constructor` diverges goja vs `-frozen` **[V]**
`function` under goja, `undefined` under `-frozen`, `llvm.Run` and natively.
Pre-existing and identical at `22e7b44`. Answering it needs a `Function` global
the subset does not have, and returning anything else would be wrong rather than
merely absent — so it is **deliberately unasserted**: an assertion here would pin
a divergence. Adjacent: `f.name` is `""` and `f.length` is `0` where node says
`f` and `1`, which needs the emitter to record a closure's declared name and
arity; `Object.getOwnPropertyNames(f)` omits exactly those.

### 1.5 kotlin: `setOf<Int>().first()` says `List is empty.` **[V]**
Kotlin says `Collection is empty.` `kSeqMethod` receives a materialized element
array and the Set-ness is gone by then — the same wall as the `elementAt` note
already at that site. Note the empty-collection MESSAGES are kotlin-stdlib
`_Collections.kt` source, not a measurement: there is no `kotlinc` here and java
cannot answer them. The throwable CLASSES are `java.util`'s and are settled.

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

### 2.4 Ruby and python residue **[V]**
- **A bare `except:` will not catch python's `GeneratorExit`** — costed as 3.3.
- **An abandoned python generator never runs its `finally`** — CPython finalizes
  at collection; we have no finalizer hook and the GC deliberately has none.
- **`d.items()` renders pairs as lists**, the same tuple root cause as 3.1.
- **ruby `Integer#fdiv` on a bignum, and a big meeting a Rational or Complex**,
  promote through the double where MRI is exact — needs a Rational whose parts are
  bigs, a second value model.
- **ruby `singleton_class`** cannot be matched — MRI's `to_s` embeds the address.
  The `NameError` wording for a missing method is unmatchable for the same reason.
- **ruby `respond_to?` on non-objects** needs a universal-method table; the
  module-object unqualified-name rule is a heuristic whose exact fix is a
  `js_rselfcall(scope, name, args)` extern, ~3 engines × 25 lines.
- **`method(:x)` on five names that take a BLOCK or are lowered at compile time**
  — `lambda`, `proc`, `loop`, `raise`, `require`/`exit`/`gets` — still answers
  `NameError` where MRI answers a Method. An OMISSION identical in all four legs,
  not a wrong answer, and each has a reason recorded at the site.
- **`Float#to_r` is `(1.5/1)`** where MRI says `(3/2)`, and `Complex` answers
  where MRI raises (`to_i`/`to_f`/`to_r` → RangeError,
  `round`/`floor`/`ceil`/`positive?` → NoMethodError; `conjugate`/`rect`/`polar`
  absent). Both need the bignum-Rational value model this section already names.

### 2.5 php's undefined `$variable` — the blocker is in the floor **[V]**
PHP 8 warns and evaluates to **null**, and the program continues; `b56c2b0` made
every *throwing* case catchable but left this one aborting. **The floor's hook can
only answer a value to THROW, never a value to RETURN.** Three ways out, none
cheap: (a) a `runtime.c` hook, ~15 lines plus its Go twin and php's registration —
and a floor change must land with its layer-2 deletions in ONE sequenced commit;
(b) a `js_scope_has`-guarded read, which **doubles the chain walk on php's hottest
path**; (c) emitter pre-declaration of every `$name` a body mentions, ~30 lines ×
2 grammars with `isset`/`compact`/`global` regression risk. Written at
`js_phmissname`.

### 2.6 lua has no `error`/`pcall` at all **[V]**
Without them "catchable" is vacuous, and the compiled halves' `member 'q' of
undefined` cannot be brought onto the interpreter's `attempt to index a nil value`
— routing through the miss hook would just produce a third wording. A real fix is
a throw/catch surface: 2 grammars + layer 2 + twin.

### 2.7 a generator body's `this` is wrong in the NATIVE build only **[V]**
`function* g(){ yield this }` then `g().next().value` gives the GENERATOR OBJECT
natively and `undefined` in the other two legs. `js_this` is one global slot in
layer 2 and the emitter arms it with the receiver of the `.next()` call. `--cross`
cannot see it (it compares interp vs run). Closing it needs a **per-coroutine
`this`** in the C floor's control block — manual 7.8's highest-risk area.
Deliberately unasserted in both ratchets: `e4c8659` made 2 of 3 legs correct, which
WIDENS the gap, so an assertion would turn `clang-check` and `native-full` red.

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

### 6.2 The js ratchet is now the slowest thing in `--full`, at 70% of its brake **[V]**
`tests/js-test-full.js`'s `-frozen` interpreter leg measures **627 s wall / 504 s
user idle** against `FULL_TIMEOUT=900`, up from 410–440 s at the sixteenth round.
Under concurrent gate load it has been KILLED outright, which reads as
`RUN-FAILED` — a red gate for a reason that is not a defect, the exact class that
timeout exists to prevent. `tests/gates.sh` already records the verdict: **if js
grows much further the answer is to SPLIT its ratchet, not to keep raising the
brake.** Splitting costs `.vscode/launch.json` rows, the `--full` per-language
accounting and `native-full`; raising the brake costs nothing and buys nothing.

### 6.3 A tuple-keyed python dict is O(n·m) **[V]**
Structural array comparison (the eighteenth round's fix for unhashable keys) has
no index behind it: layer 2's `pyDictFind` cannot hash an array and the
interpreter's index cannot represent one, so every stored key is compared
element-wise. It was O(n) before only because it never found anything. **Hashing
tuples needs the real tuple type (3.1)**, so this is a consequence of that item
rather than an independent one. Measured cost on a dict-saturated microbenchmark:
+0.77% against that row's own 1.13% spread, i.e. inside the noise.

### 6.4 kotlin carries an unattributed +2.28% **[V]**
`tests/bench.sh --draws 9` puts kotlin at +2.28% median against its own 1.85%
spread (gate tolerance 2.5%, so it passes); a paired sign test over 16 draws says
14 slower, ~+1.6%. **Five controls say what it is NOT**: not layer-2 growth
(appending ~800 dead `.ll` lines reproduces it 6/6), not the two new hot-path
arms, not the four new helpers, not a scope-name hash collision, and not executed
code anyone can point at — `mod.kt` has no member read, no indexed write and no
collection call, its emitted module is **md5-identical** to base, and
`MEC_GC_STATS` reports the same collections and heap. That points at module
LAYOUT, which is manual chapter 4's effect, and it is written up at the site.

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

**~194 items closed** (`e284185`..HEAD). **One section, grouped by VERDICT and by
nothing else** — no dates, no rounds, no order, no commit archaeology. The
verdict plus its REASON is the whole content, because the reason is the only part
that stops someone re-opening it; the detail is in git and the durable lessons are
in the manual (ch. 4 and 7).

**DECLINED after measurement.** `i in arr` after `delete` · iterator close on
`throw` (a depth stack cannot survive `for await`) · a catchable unknown-name
error in java/go/swift/csharp/metajs (COMPILE-time in each real toolchain) ·
kotlin `:script` → `!"token"` · `case_map` into layer 2 (135×) · retiring the Go
twin (manual 7.11) · dart `x as T` as a throwing cast · an mtime skip for the
generators (a `git checkout` would make a stale `.ll` report CLEAN) · matching
swiftc's **exit 133** on a trap (its stdout discard is libc block buffering, not a
language property, and our interpreter legs run *inside* `mec`, so `raise()` would
kill the metacompiler and at most 1 of 3 legs could match — widening the gap).

**The ITEM was wrong** — already fixed, or a false premise, oracle in hand.
python integer arithmetic past 2^53 (already exact, 70 rows) · python's `&`
int32-signed (already correct, its own example row included) · ruby `%g` (already
implemented — the real gap was the `#` flag) · java's divide-by-zero message
(already right and already asserted) · `(super as any)[k]` (real `tsc` rejects it,
TS1034) · dart type-parameter erasure (Dart reifies) · kotlin's nested throwable
name "changes every `Name@1`" (it does not — `Name@1` reads `__name`, only the
throwable path reads `__bname`) · swift's two abort probes "become byte-identical
once value semantics land" (they already were; only the stale headers survived).

**The prescribed FIX was wrong.** The C# generated message on every derived type —
fixes one name, breaks twenty-six · ruby's malformed formats via "emit % and
rescan" — would have converged all three engines on behaviour MRI rejects, turning
a halves divergence into agreed-and-wrong · ruby's `round(n)` via `rtExactDec` —
the true expansion of 2.675 is 2.67 and MRI answers 2.68, because `round_half_up`
carries a midpoint correction · swift's abort message via a floor change — Go
already guards in front of the shared layer, so `runtime.c` needed no edit ·
python's unhashable-key check by refusing `list` — a tuple IS a list here, so it
would abort `{(1,2),(3,4)}` and `d[i,j]`; structural array comparison instead.

**FIXED, and what each one actually turned out to be** — kept only where the
finding is bigger than the fix. ruby `Float#round/floor/ceil(n)` — MRI's own
numeric.c algorithms, including the `int_round_zero_p` cliff that overrides the
arithmetic · ruby malformed formats — now RAISE with MRI's messages, and the blast
radius of raising was zero · kotlin out-of-range indexing — six defects, not two,
and the bounds check measured FREE because it folds into work the read already
does · js/ts private names and `for..in` — one root cause (the interpreters filed
`#x` unmangled) and a second place in the compilers nobody had checked · swift
Array/String members — nineteen, plus six adjacent, SECTION 36 running under real
`swiftc` · `--full`'s total silently discarding a gapped half's assertions ·
`tools/floor-coverage` (330 of 345 floor bodies reached) · `tools/metajs-lint`,
which found two live defects on its first run, both wrong answers in shipped
native binaries no dynamic gate could see · `-why`, justified by answering three
questions BY HAND first and getting two of them wrong.

**Settled by the eighteenth round.** swift value semantics — copy at STORE, not
on read, which is complete by construction and is why an in-place subscript write
still mutates · swift's two abort probes — already byte-identical, so they got the
full row pair instead, proved by sabotage · kotlin user throwables — link
`__super` to the builtin hierarchy; the "changes every user class" worry is false
because the arm only fires when the superclass has no descriptor in this compile ·
kotlin `js_ktexcmatch`'s non-throwable hack — NOT deleted, it self-retires behind
its own `!isThrowable` guard · python unhashable keys — `dict`/`set` refused,
arrays compared STRUCTURALLY, because a tuple IS a list here · ruby `equal?` — not
a bare `===`, since the twin's `jsRat`/`jsCpx` are comparable Go structs and `==`
would have been a designed-in halves divergence · ruby kernel `method(:x)` — a hit
in the PARENTLESS scope is a host builtin, a `def` always lands in the module
scope, which has a parent · js function properties — the side-table read must not
be spelled `fp[k]`, which walks goja's own `Object.prototype`.

**Dropped, or moved out of this file.** The `.expected` fixture path ·
`getattr()`-bound python builtins with kwargs · `glibc < 2.34`'s `-ldl -lpthread`
· `mod.py` headroom. Moved to the manual: the scope API (7.3), per-language
divergences (7.9), the byte-vs-character `printf` rule (7.7).
