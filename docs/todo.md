# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Completed work lives in git, not here. [§8](#8-settled--do-not-re-open) keeps one
line per settled question — the VERDICT and its REASON, nothing else, because the
reason is the only part that stops someone re-opening it.

Two conventions, and they are the point of the file:

> **[V]** — a probe or grep was run for this item, on this commit.
> **[U]** — inherited from an older note and NOT re-checked. A lead, not a fact.

> **Treat an item as a place to start probing, never as a specification.** Items
> have repeatedly understated their defect or named the wrong fix. Probe the FIX as
> well as the DEFECT, re-probe a **[U]** item before assigning it, and expect a
> brief's instruction to be declined with a measurement. Item numbers change as
> items close, so never cite them from code.

The baseline every change must preserve is whatever `tests/gates.sh` records
(its constants: FULL_ASSERTIONS, MATRIX_ENTRIES, SHAPE_GROUPS, ...); run it with
`--gen --freeze`, and with `--bench` for anything touching layer 2 or the floor.

**Before measuring anything, read the manual's chapter 4 and
`tests/bench/baseline.txt`'s header** — instructions not wall clock, medians over
layout draws, `--paired`, corroborate with an exact counter.

**Do not raise `FULL_ASSERTIONS` from a working tree.** The gate prints
"+N, RAISE FULL_ASSERTIONS" and that message is the ratchet doing its job; the
floor is raised afterwards, once, from a COMMITTED tree.

---

# 1. Correctness, with an oracle on this machine

Every item here was measured while closing something else, so they are findings
rather than leads — but the header's warning still applies: **re-probe before
assigning**, at the magnitude the item names.

### 1.1 python: `%d`/`{:d}` on a non-integer operand is a THREE-WAY divergence **[V]**
`"%d" % 1e300` gives `1e+300` natively, `9223372036854775807` in the Go twin, and
CPython the exact 301-digit integer. `"%d" % None` gives `NaN` natively, `0` in the
twin, and CPython a `TypeError`. Same for list/tuple/dict/set/str operands: **192
differing rows, both halves wrong and differently wrong.** `python3` settles every
row. Found while merging `rtTag`; probe recipe in that commit (`cd75454`).

### 1.2 js/ts: identifier `\u` escapes are never resolved in the INTERPRETER halves **[V]**
`var a = 1; print(a)` prints `1` under the compiler and throws
`ReferenceError: a is not defined` under the interpreter, in both js and typescript.
A live halves divergence, pre-existing. The compiler has `idName`; the interpreters
have nothing. It is ~30 `up.in` sites per interpreter grammar and `up.in` is matched
TEXT, so no tag inside `Id` can fix it centrally — which is why `js-test-full.js`
SECTION 52 deliberately asserts no escaped spelling. `node` settles it.

### 1.3 js: spreading a generator answers an empty array **[V]**
`[...g()]` gives length 0 where node gives 2, in both halves, pre-existing. The
spread path (`evalArgs`/`spreadInto`) admits arrays and strings only. It is why
`fns09` drives its generator with `next()`.

### 1.4 ruby: `String#to_i` is a three-way divergence **[V]**
`"0xffffffffffffffffff".to_i` → interpreter `4722366482869645213696`, both compiled
halves `0`, MRI `0`; with an explicit `.to_i(16)` → interpreter the same big value,
compiled halves `0`, MRI `4722366482869645213695`. So the interpreter honours a `0x`
prefix that MRI's base-10 `to_i` ignores, AND the compiled halves ignore an explicit
base. `rubySmBasePrefix` also carries the FMA contraction of manual chapter 4 —
fixing only that swaps one wrong answer for another without closing the divergence.

### 1.5 ruby: `print`/`puts` emit earlier arguments before a later one raises **[V]**
`print "A","B",boom,"C"` prints `AB` in the compiled halves and nothing in the
interpreter; `puts "D", boom` prints `D` in all our legs. MRI evaluates every
argument first and prints nothing. A live halves divergence for `print`,
agreed-and-wrong for `puts`.

### 1.6 ruby: `defined?` kind classification, four residues **[V]**
Kernel names (`p`, `sprintf`, `format`, `lambda`, `proc`, `rand`) answer nil or
`"local-variable"` where MRI and the interpreter say `"method"` — blocked on a fourth
copy of the host-globals list, or an extern-signature change in `abnf/jsrt.go`.
`defined?(printf)` reads `"local-variable"` because the chain's root is the MetaJS
host scope carrying `print`/`printf`/`sprintf`, the collision `js_rmethodval` already
documents. `defined?(@nope)` answers `"instance-variable"` in all four legs where MRI
says nil. And `defined?(loc)` at TOP LEVEL answers `"method"` in the interpreter — its
`globalScope` is `scopes[0]`, so the local walk skips it — where MRI and both compiled
halves say `"local-variable"`.

### 1.7 kotlin: `Array`, `Sequence` and a Map's `values` still say `List is empty.` **[V]**
`arrayOf<Int>().first()` should say `Array is empty.`, `asSequence().first()`
`Sequence is empty.`, `map.values.first()` `Collection is empty.` **An Array, a
Sequence, a Map's `values` view and a List are the same host array in this value
model**, so no receiver survives to carry a kind — see 2.2. The `values` row is
deliberately NOT asserted rather than pinned to a known-wrong answer. Strings are
spec-derived from `_Strings.kt`/`_Collections.kt`; there is no `kotlinc` here.
Also absent: `reduceOrNull`/`reduceRightOrNull`.

### 1.8 swift and dart: extensions on a builtin receiver **[V]**
swift `extension String { var lenPlus: Int { return count + 1 } }` fails in all THREE
engines (`unknown name: count`) where real Swift accepts it — `getMember` does not
know Swift's `count`, so it needs the interpreter, the twin and layer 2 together.
dart `extension IX on int { bool evenq() => isEven; }` still fails: `getMember` knows
no Dart `int` members, so it needs a member table in `abnf/jsrt.go` and in layer 2.
**Do NOT route either through `js_kget`**: its string surface is `length` plus ten
JAVASCRIPT method names, and `809d470` records what happened when dart tried —
`"abc".slice(1)` started working, which real Dart rejects.

### 1.9 php: variable scope leaks into every function, in both halves **[V]**
`$g=5; function f(){ echo $g; }` prints `5`; a closure sees `$g` with no `use`. PHP
has no lexical variable scope, so this is not a corner. A function's body scope
chains to the module scope and variables are ordinary `$`-prefixed bindings in it.
Both halves agree, so `--cross` is blind and no oracle here can see it. A fix needs a
VARIABLE BOUNDARY (`global`/`use`/`$GLOBALS`/`static` are the four documented ways
through) in both grammars, on the hottest path in the language — so the boundary test
must be ONE `js_scope_has` on ONE frame, never a walk. **Entangled with 2.5 in one
direction: closing the leak turns every accidental global read into 2.5's abort, so
2.5 must land first.**

### 1.10 php: `empty($undef)` aborts, and `$u[0] = 1` does not vivify **[V]**
`empty($undef)` / `empty($undef[0])` abort in both halves where PHP 8 answers true
and diagnoses nothing — and because `empty` takes an arbitrary EXPRESSION, the
non-aborting read must be selected at PARSE time (`KwEmpty "(" TgtOnly ")"` before
the expression alternative) in both grammars. Separately `$u[0] = 1` on an undeclared
variable aborts where PHP creates the array: `emitRefPathV`'s
`viv && ref.path.length > 1` guard skips vivification for a single-step target.

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
- **`Complex` answers where MRI raises** (`to_i`/`to_f`/`to_r` → RangeError,
  `round`/`floor`/`ceil`/`positive?` → NoMethodError; `conjugate`/`rect`/`polar`
  absent) — needs the bignum-Rational value model this section already names.
  (`Float#to_r` and `Complex#abs` were closed at `44bdde7`; `abs` needed
  power-of-two scaling, because the textbook `hi*sqrt(1+t²)` is WORSE in the
  ordinary range and Apple's own libm `hypot` is not correctly rounded either.)
- **Complex's parts are raw doubles** in all three engines, so no engine can tell
  `Complex(0,0)` from `Complex(0.0,0.0)`: `Complex(0,5).abs` is `5.0` where MRI says
  the Integer `5`, `5.0.to_c` is `(5+0i)` not `(5.0+0i)`. Same class as the above.
- **`Kernel#Integer(str)` is imprecise past 2^53** — now agreed across all three
  legs, so `--cross` is blind. `bigFromDec`/`rbABigFromStr`/`big.Int` all exist;
  layer 2 has no radix→decimal helper, so it is ~3 × 25 lines plus assertions.
- **`1.5r` does not parse** (`RatLit` takes only an integer mantissa), and
  `p two 1, 2` — a nested parenthesis-less call in an argument — does not parse.

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

# 4. Open defects with an oracle answer, found in review

Each was probed against the real toolchain (or, where none exists here, the
language spec, marked so). None is fixed yet.

### 4.1 python: integer division by zero never raises **[V]**
`1 // 0` prints `Infinity` and `x % 0` prints `NaN` in all three engines; CPython
raises `ZeroDivisionError`. Only the exception class exists.

### 4.2 python: `str.title()` treats `ß` as uncased **[V]**
`"straße".title()` is `StraßE` in all engines (CPython `Straße`): the case-fallback
length guard makes `ß` end a word.

### 4.3 python: smaller gaps **[V]**
`dict.fromkeys`; set `issuperset`/`isdisjoint`/`intersection`/`difference` (and
`hasattr` says False for them). `except E as w` does not unbind `w`. An uncaught
KeyError's repr differs between halves (`[1]` vs `1`). `__mro__` lacks `object`.
`None.k = v` is not catchable. Natively `float(2**80)` is `nan` and
`2**80 == float(2**80)` is False (CPython `1.2089258196146292e+24`, True).

### 4.4 java: assignment conversion only at declarations **[V]**
A declaration and a field initializer convert (`double d = b` widens a float,
`static float f = 16777217` rounds), but parameters, returns, plain reassignment,
field/array stores and array initializers do not, in all engines: `q(2)` for
`void q(double)` prints `2` (java `2.0`), `double r(){return 3;}` prints `3`,
`long t; t = 2; t*1000000*1000000*1000` wraps at 32 bits. Needs the declared type
threaded to those sites in both grammars.

### 4.5 java: defaults and constants **[V]**
Uninitialised `double`/`long`/`float`/`byte` fields read `null` (java `0.0`/`0`).
`Double.POSITIVE_INFINITY`/`NEGATIVE_INFINITY` are `null` in the interpreter and
`NaN` compiled (a halves divergence); `Float.NaN` and `Math.sqrt` are unknown in the
interpreter. `Integer.parseInt`'s NumberFormatException is not catchable in the
compiled halves.

### 4.6 go: the built-in `error` interface is not modelled **[V]**
`x.(error)` is false even for a type with `Error()`, `case error:` never matches,
and a recovered runtime panic is a plain string (so `r.(string)` succeeds where go
fails). Natively, an UNRECOVERED integer divide by zero still dies through the C
floor (`js runtime error: ...`, exit 1; go: `panic: runtime error: ...`, exit 2).
`a.(int)` on a string does not panic. fmt: composites are not walked per element
in bad-verb notes (`[%!s(int=1) %!s(int=2)]`), int64/uint64 print as int/uint, and
`%f %g %e %x %o %b` are unimplemented.

### 4.7 ruby **[V]**
`{}.fetch(:a)`'s KeyError is not catchable by `rescue KeyError`/`IndexError`.
`NotImplementedError` sits under StandardError (MRI: ScriptError). The compiled
halves lack `IOError`/`LocalJumpError`. The interpreter lacks `Float#div`,
`divmod`, `rationalize`. `"\u{48 49}"` gives `H 49}` (MRI `HI`).
`p (2.0**55).to_r` parses as `(p ...).to_r`. The interpreter answers `"method"` for
`defined?(x)` of a top-level local (MRI and the compiled halves: `local-variable`).

### 4.8 js/ts **[V]**
`Math.pow(1, NaN)` and `Math.pow(±1, ±Infinity)` give 1 (node NaN), probably `**`
too. `JSON.stringify` is missing everywhere. A derived class whose `super()` returns
an object does not rebind `this`. On an Error, `hasOwnProperty("name")` is true and
`getOwnPropertyNames` gives `[]` (node: false, `[stack, message]`). A call of a
non-function names the callee without node's spacing (`o.a+1 is not a function`).

### 4.9 swift **[V]**
`d[k] ?? "x"` and `if let v = d[k]` on a missing key trap (swiftc: the default).
`x as? Double ?? -1` does not parse. `Dictionary.first` answers a key, not a tuple.
`var x: Double? = nil; x = 3` prints `3` (swiftc `3.0`). A typealias to `Any`, an
observed/computed `Any` property, and `for var`/`if var`/closure-parameter locals
still adopt the old value's type on a write.

### 4.10 C# (spec only - no dotnet here) **[U]**
Overload resolution takes the first declared applicable candidate, not the most
specific (`O(object)` declared before `O(string)` wins for `"s"`). Derived field
initializers run after the base constructor (C#: before). `flt36`/`flt37` assert
`Single` exponent forms whose threshold is unverified here.

### 4.11 dart (spec only - no dart here) **[U]**
The features check `tdef-legacy-form` asserts `int Function(int)` is a
`dynamic Function(bool)`; parameters are contravariant, so Dart says false.
Chained `x as T is T` may be a syntax error in real Dart.

### 4.12 bash, lua **[V]**
bash: unquoted `${a[@]@A}` is 4 words, ours 3. lua: a subnormal float key read back
through `pairs()` compares unequal in the compiled halves.

### 4.13 MetaJS and the C subset **[V]**
The MetaJS lexer reads `"\f"`/`"\v"` as the letters (`"\f".charCodeAt(0)` is 102);
metajs-lint guards the known sites but the lexer is the root. `-frozen`'s
`indexOf(x, from)` ignores `from` (`"aXaXa".indexOf("a", 2)`: goja 2, frozen 0).
The C subset's `(float)x` is a no-op.

### 4.14 Replay-based generators repeat side effects in the interpreters **[V]**
js, ts and python interpreter halves drive generators and async functions by
replay, so effects before a `yield`/`await` run again (js `take(inf(), 3)` gives
`t0 t2 t5`). The compiled halves are correct.

### 4.15 Case mapping is no longer pinned goja-vs-frozen **[V]**
The python/ruby/kotlin ratchet rows that compared goja's and `-frozen`'s case
mapping asserted single-character answers the real languages contradict
(`"ß".upper()` is `SS`) and were removed. Pin the agreement again with
oracle-correct (full-mapping) expectations or a Go test.

# 5. Duplication and dead code

`go run ./tools/shape-scan` is the measure; `tests/gates.sh` records the gate-tier
(140 chars) numbers. The permissive tier (`-min 40`) shows what is left below it.

> **⚠ A ZERO-KNOB DIFFERENCE LIST IS NECESSARY AND NOT SUFFICIENT — SIZE THE CALL
> SITE, NOT THE BODY.** The fifth pass merged on an empty difference list alone and
> shipped a regression that stood for four commits: `pyIsObj` → `rtIsObj` cost
> **python +41%**, `rbIsObj` **ruby +11%**, `rtjIsIntegral` **java +5.7%** — because
> each is a predicate asked at ~15–30 sites inside an arithmetic path, and **in
> MetaJS a call is a scope frame**. `8de27d7` is the worked example of doing it
> right, and `cd75454` is the sharpest: the same merge DELEGATING measured
> **+10.60%, slower 9/9**, and INLINED measured **−13.90%, faster 9/9**. Identical
> bodies, different call sites, and only the call sites decide.

> **⚠ AND THE INSTRUMENT HAS A BLIND SPOT ON EXACTLY THIS KIND OF CHANGE.** Adding a
> name to `runtime.metajs` renumbers the `@str.N` intern table of every importing
> module (148 source lines → 7,705 insertions in `python-rt.ll`), and `--paired`
> cancels the output PATH and nothing else — so that shift is identical in every pair
> and reads as a **unanimous sign**. A nine-line uncalled body measured "+0.95%,
> slower in 8 of 9". **Measure against a DEAD-CODE MATCHED CONTROL.** Full note in
> `tests/bench.sh`'s header.

### 5.1 The frame-elimination class, and it is the biggest measured win available **[V]**
`rtTag`'s **−13.9% on python** is ONE scope frame removed from ONE predicate. The same
`!pyIsObj(v)`-then-field-test shape sits at `pyIsFlo`, `pyIsNumeric`, `pyInstClassOf`,
`rbUserObj` and `rbIsClassObj`, all on arithmetic paths. Each needs its own call-site
sizing and its own measurement against a matched control, which is why `cd75454` did
not fold them in.

### 5.2 `rtIsNil` could serve three more languages **[V]**
`jvUndefOrNull` (js-rt), `ktIsNullish` (kotlin-rt) and `phIsNull` (php-rt) are the
same four lines with the two tests swapped — a second shape-scan group. kotlin and php
already import `runtime.metajs`; **js imports only `regex.js`, so it pays a first
import plus seven `rtk*` hooks** (manual 7.13), which is the real cost and not the
line count.

# 6. Tooling and developer experience

### 6.1 The `-frozen` host needs 11.9 GB for one js ratchet part **[V]**
`tests/js-test-full.js` part 1 under the `-frozen` interpreter needs **11,907 MB
where goja needs 99 MB** for the same file, and typescript's *larger* ratchet needs
2,398 MB — so it is js-path-specific, not file size. Fifty sections cost 57 s of CPU
parsed one at a time and 356 s in one file. **First suspect: `chainItems`, which
retains every folded member chain for the whole run.** This is what made the ratchet
split (`ed9f71a`) necessary: at 23.5 GB the machine pages while other legs run, and
that is how the probe was being KILLED — a red gate for a reason that is not a defect.

### 6.2 `-trace` output is not reproducible run to run **[V]**
32 lines of a 170-line python trace carry raw Go heap addresses, because a scope
object gets rendered into a `decl`/`read` `val`. Pre-existing, and it means **no gate
can ever compare a trace file byte-for-byte.** The site is `abnf/trace.go`'s value
rendering, not a grammar. Same for ruby and php.

### 6.3 php aborts on a top-level `label:`/`goto` or a bare `{ … }` block **[V]**
`TypeError: Not a function` from `compile-core.js:617` (`makeSeq`) via `buildMain`:
the top-level list is built by `makeSeq`, which — unlike php's own `makeBody` — does
not recognise the `{lblstmt}` / `{body}` records those statements push. The same
shapes work INSIDE a function.

### 6.4 `-callgraph` names a function by the minimum marker in its body **[V]**
So `def f` on line 8 reads `L9` in every language. Cosmetic, and now visible in three
more languages because `932b3e2` gave ruby and php line attribution at all.

# 7. Corpus recognition

### 7.1 JS reject side — what is left after two passes
`026d5cd` took the unicode group (+3 reject, −0 must-parse) and `53102a4` the goal
symbol (+9, −0). Reject stands at **703 / 727**, must-parse at **2623 / 2653**.

> **A warning that was paid for TWICE.** Guards refusing a parenthesized non-target
> for-head and a numeric postfix operand won 7 reject files and cost **8 must-parse**
> — `0++`, `for(([0]) in 0)`, `for((0) of 0)` under `early/`, which are early errors
> that must PARSE. And "a Script may not contain `import`/`export`" won 6 and cost
> **3**: `js-test-full.js` (the ratchet itself), `js-test-recognize.js` and
> `js-test-multifile.js`. **Always measure BOTH rows.**

**Assignment targets (8 files)**: a form that is not a LeftHandSideExpression used as
one — `() => {}()`, `({a:this}=0)`, `switch (c) { default: default: }`.

**`import`/`export` in a Script (6 files)**: measured, and the fix is not in a
grammar — it needs the three ratchet files renamed to `.mjs`, i.e.
`.vscode/launch.json` and `test.sh`.

**`export default` (2 files)**: `ExportDefaultExpr` takes `[ ";" ]` rather than
`StmtEnd`, so `export default 3 4` parses as two statements. `StmtEnd` there costs two
must-parse files, because an anonymous `export default class {}` is a DECLARATION in
the spec and ends at its own brace. The real fix is anonymous class/function
declaration forms in the `ExportDefaultDecl` branch.

**Unicode identifiers — CLOSED for js/ts, and NOT transferable.** `026d5cd` has the
real ID_Start/ID_Continue tables in `languages/lib/ident-unicode.abnf`. Five languages
were checked against a toolchain or a spec and each needs its OWN list: swift
(`swiftc` disagrees in both directions), java (JDK 24 disagrees three ways), kotlin,
csharp, and python (XID_*, and **python3 is Unicode 16.0 while node 24 is 17.0 —
they disagree on 4,647 code points for XID_Start alone**, so the tables must come
from python3). Also `\u{0000000061}` (>8 hex digits) is accepted by node and rejected
here, an 8-digit cap inherited from the replaced guards.

### 7.2 Other corpora
- **batch 0 of 2** — the ratchet is FULL at 24 sections, but neither wine file parses.
- **swift 80%** — an 18-file lexical tail: suppressed conformances, `@isolated(any)`
  in a type, `borrowing` as a bare type name, raw multi-line strings with 3+ pound
  levels, parameter packs, form feed and NBSP. **For Swift, lexical work buys corpus
  and semantic work buys ratchet — they are nearly disjoint.**
- **dart 96.6%** — ~53 of the remaining 117 are multitests whose surviving `//#`
  variants the files themselves label "syntax error".
- **python 92.9% and ruby 98.1%** are at their hard caps.
- **go** — one file fewer, deliberately: `fixedbugs/issue24801.go` is comment-only
  with no `package` clause and real `gofmt -e` rejects it, so `53102a4`'s parser fix
  made that row truthful rather than worse.
- **php / csharp / java / c** have ordinary long tails.

# 8. Settled — do not re-open

**~196 items, one section, grouped by VERDICT and by nothing else** — no dates, no
rounds, no order, no commit archaeology, because none of it changes what to do
next. Each line is a verdict plus the shortest reason that stops someone
re-opening it. The detail is in git; the durable lessons are in the manual
(ch. 4 and 7).

**DECLINED after measurement — each is a number the next person should not have to
re-derive.** php's `isset`/`$this`
scope probe into layer 2 (**+0.77%, slower in 9 of 9 paired draws**, ~490 instructions
per probe call: unlike kotlin's, php's probe has NO miss arm layer 2 answers better,
so it is all hot arm) · swift's and dart's `sw_kget`/`sw_kset` into layer 2
(**+81.19% and +68.97%, 9 of 9** — `mod.swift`, a bare `while` over two locals, emits
4 `sw_kget` + 2 `sw_kset` PER ITERATION) · python's `py_setvar` containment walk (the
premise does not survive: the emitter already knows those names statically via
`pyInnerNames`, and the floor already owns the name so layer 2 cannot re-use it) ·
a shared `.abnf` module for the three emitter closures (**impossible, not merely
unwanted: an `:include`d file's `:startScript` is SILENTLY DISCARDED**, since
`GetStartScript` takes the first and includes are appended after the host's — there is
a warning for it now) · `rtTag` in its DELEGATING spelling (**+10.60%, slower 9/9** —
`120ba0f`'s regression re-run; the INLINED spelling is −13.90% and shipped) ·
`import`/`export` refused in a Script (+6 reject files, **−3 must-parse**, including
the js ratchet itself) · the goal symbol for TypeScript (the corpus has ZERO
`.mts`/`.module.ts` files and `tsc` decides module-ness from CONTENT) · ID_Start
tables for swift/java/kotlin/csharp/python (**each language's identifier set really is
different, and two toolchains proved it in both directions**) · swift's `stride(...)`
(absent from the subset, and its `for-in` is a shape already asserted) · making a
named function expression's inner name immutable (needs a const flag on the floor's
scope, or a scope-walking extern on every plain-name assignment, measured at
+21%..+43% elsewhere on that path).

**DECLINED after measurement.** `i in arr` after `delete` · iterator close on
`throw` (a depth stack cannot survive `for await`) · a catchable unknown-name error
in java/go/swift/csharp/metajs (COMPILE-time in each real toolchain) · kotlin
`:script` → `!"token"` · `case_map` into layer 2 (135×) · retiring the Go twin
(`llvm.Run` structurally cannot execute the C floor, manual 7.11) · dart `x as T`
as a throwing cast · an mtime skip for the generators (a `git checkout` would make
a stale `.ll` report CLEAN) · swiftc's **exit 133** on a trap (libc buffering, not
a language property, and our interpreter legs run *inside* `mec`, so at most 1 of 3
legs could match — widening the gap) · kotlin's unattributed **+2.28%** (five
controls exclude every attributable cause; it is module LAYOUT, ch. 4's 4.8%
effect, and it passes the gate) · a tuple-keyed python dict being **O(n·m)** (a
consequence of the tuple type, not independent work; +0.77% inside a 1.13% spread).

**The ITEM was wrong** — already fixed, or a false premise, oracle in hand.
python integer arithmetic past 2^53 · python's `&` int32-signed (both already
correct, each including the item's own example row) · ruby `%g` (already
implemented; the real gap was the `#` flag) · java's divide-by-zero message
(already right AND already asserted) · `(super as any)[k]` (real `tsc` rejects it,
TS1034) · dart type-parameter erasure (Dart reifies) · kotlin's nested throwable
name "changes every `Name@1`" (only the throwable path reads `__bname`) · swift's
two abort probes "become byte-identical once value semantics land" (they already
were; only the stale headers survived).

**The prescribed FIX was wrong.** C#'s generated message on every derived type
(fixes one name, breaks twenty-six) · ruby's malformed formats via "emit % and
rescan" (would have converged all three engines on behaviour MRI rejects, turning a
halves divergence into agreed-and-wrong) · ruby's `round(n)` via `rtExactDec` (the
true expansion of 2.675 is 2.67 and MRI says 2.68 — `round_half_up` carries a
midpoint correction) · swift's abort message via a floor change (Go already guards
in front of the shared layer) · python's unhashable-key check by refusing `list` (a
tuple IS a list here, so it would abort `{(1,2),(3,4)}` and `d[i,j]`).

**FIXED.** ruby: `Float#round/floor/ceil(n)` (MRI's own numeric.c, including the
`int_round_zero_p` cliff), malformed formats now RAISE, `equal?` (not a bare `===`
— the twin's `jsRat`/`jsCpx` are comparable Go structs, which would have been a
designed-in halves divergence), kernel `method(:x)` (a hit in the PARENTLESS scope
is a host builtin) · kotlin: out-of-range indexing (six defects, not two; the
bounds check is FREE because it folds into the read), user throwables via `__super`,
`js_ktexcmatch`'s hack self-retires behind its own guard · js/ts: private names and
`for..in` (one root cause, plus a second site in the compilers nobody had checked),
function own-properties (the side-table read must not be spelled `fp[k]`, which
walks goja's `Object.prototype`) · swift: nineteen Array/String members plus six
adjacent under real `swiftc`, and value semantics as copy-at-STORE (complete by
construction, which is why an in-place subscript write still mutates) · python:
unhashable keys — `dict`/`set` refused, arrays compared STRUCTURALLY · `--full`'s
total silently discarding a gapped half's assertions · `tools/floor-coverage`
(330/345 floor bodies reached) · `tools/metajs-lint` (two live defects on its first
run, both wrong answers in shipped native binaries that no dynamic gate could see)
· `-why`.

**Dropped, or moved out of this file.** The `.expected` fixture path ·
`getattr()`-bound python builtins with kwargs · `glibc < 2.34`'s `-ldl -lpthread` ·
`mod.py` headroom. Moved to the manual: the scope API (7.3), per-language
divergences (7.9), the byte-vs-character `printf` rule (7.7).
