# Outstanding work

**This file is the to-do list. [working-on-this-project.md](working-on-this-project.md)
is the manual** — architecture, how to build and test, the traps, and the engine
mechanics. Read the manual first; nothing here explains how anything works.

Rebuilt 2026-08-05 at `e284185`; **~119 items closed since**, in eleven rounds.
Chapter 1 has been emptied and refilled every round; chapter 9 keeps only what
those rounds taught.

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
> - **The item is too small** — now the common case, and on the ninth round it was
>   NINE items of ten. An extension did not lose a type's static overloads, it
>   REPLACED all of them including methods and subscripts; one csharp item named
>   one language of two; one ruby bullet named one String method of seventeen; go's
>   four residues came with six more on the same code path; kotlin's one range
>   defect came with eight. Many of those extras were live halves divergences
>   `--cross` had never reached.
> - **The item's PRESCRIPTION is wrong while its diagnosis is right.** Four items
>   in one round told you how to fix them and were wrong: one regressed two shapes
>   when implemented as written, one gave a false reason, one prescribed a design
>   the value model cannot express, and one declared a bullet impossible that the
>   emitter does trivially.
> - **The work is wrong and every gate says green.** Two agents shipped a NEW
>   halves divergence that their own probes and all seven gates called green.
> - **The item's own EXAMPLE does not reproduce.** New on the eleventh round, and
>   twice: swift's nested-type example passes at base (the labels have to
>   collide), and 3.2's stated blocker — the SHOULD-ABORT rows pinning
>   `js_scope_get` — does not exist at all; every one of the 30 `variable not
>   defined` references in the tree is a comment. **Reproduce the item before you
>   fix it**, and if it does not reproduce, the defect is usually still there in a
>   shape the item did not find.
>
> **Treat an item as a place to start probing, never as a specification.**
>
> **Item NUMBERS are re-assigned every rebuild.** Code comments of the form
> `docs/todo.md 1.4` are provenance pointing at the numbering of the day they were
> written — do not "fix" them, and do not trust them to name today's item.

Baseline every item must preserve (`tests/gates.sh`, ~2.5 min — add `--serial` on a
small machine):

```
matrix 357/357 · --full 8,123 assertions, 0 halves disagree · --cross 121/0
clang-check 16/16 none held · go test ok · gen-all 15/15 clean
native-full 20 languages / 37 native programs (ratchet AND feature matrix), 0 held
shape-scan 15 groups / 481 recoverable · -freeze a fixed point
bench: 15 rows, no row outside its spread · tests/coro-poc/build.sh byte-identical
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

**`tests/gates.sh` now passes `--timeout 600` itself, and a short read is a
FAILURE.** It used to be silent: if the FIRST probe run is killed, `full_probe`
returns without writing the count, both halves record `-`, the two halves
therefore AGREE, and the summary prints a smaller number as though it were the
answer — observed at **437 assertions below baseline with no failure reported
anywhere**. `test.sh` now emits `RUN-FAILED` on a killed first run, and `gates.sh`
checks the total against a recorded FLOOR (`FULL_ASSERTIONS`) and names any
language whose count is `-`. **Running `./test.sh --full` by hand still needs
`--timeout 600` on a loaded machine** — the js compiler half is ~19 s idle and
82 s at load 16 against a 120 s default.

---

# 1. Correctness, with an oracle on this machine

These change answers real programs give. Each has a toolchain here that settles
it. **Section 1 was emptied for the tenth time** — all eight items, plus 2.2 and
4.1, are closed in `d945bb9..d0f4db3`. What follows is what closing them turned
up, all **[V]** on `5cb4d9f`.

**The failure mode this round: the item was too small EVERY TIME, and twice its
own example did not reproduce.** swift's nested-type example passes at base (the
labels have to COLLIDE), and 3.2's stated blocker — that the SHOULD-ABORT rows pin
`js_scope_get` — turned out not to exist: all 30 `variable not defined` references
in the tree are comments and no test asserts on that abort. One item named 1 defect
of 9, one named 6 String methods of 78, one named 3 `fail` sites of 4. Five items
turned up live halves divergences `--cross` had never reached.

### 1.1 C# typed catch does not exist in EITHER half — and probably not in five more **[V]**
`catch (FormatException e) {…} catch (InvalidOperationException e) {…}` runs the
*FormatException* body on an `InvalidOperationException`, in both engines, so it is
a spec divergence rather than a halves one. **This is why the new
`IndexOutOfRangeException` / `ArgumentOutOfRangeException` / `KeyNotFoundException`
/ `OverflowException` are only distinguishable by `.Message`.** The blocker is
structural: BCL exceptions are `{__excname, Message}` plain objects with no
`__class`/`__super`, so a type test has nothing to walk. The fix is real
descriptors with a parent chain (the shape java's `emitBuiltinExc` already has),
a `js_csexcmatch`, and the `jCatchDispatch` built for java in `d945bb9`, which
ports almost verbatim — ~200 lines across four files.
**The same gap likely exists in kotlin, swift, dart, php and python**: every
compiler half whose `makeTry` keeps only `items[i].catchbody` when
`catchT == undefined`. Worth one sweep across all of them, and that sweep is the
item, not the C# half alone.

### 1.2 A nested swift type shadows a same-named TOP-LEVEL type **[V]**
```swift
struct Inner { func who() -> String { return "top" } }
struct A { struct Inner { func who() -> String { return "nested" } } }
print(Inner().who(), A.Inner().who())   // ours "nested nested", swiftc "top nested"
```
Wrong in all three engines. The compiler half can be fixed alone (it resolves bare
type names statically through `typeKeyOf`), but **the interpreter cannot follow** —
it resolves a bare type name at *run* time through the scope chain with no static
type context, so removing the bare binding breaks its superclass and protocol
lookups. Doing only the compiler half ships a NEW halves divergence, which is why
`69d8e17` stopped here. The interpreter needs a qualified `__qname` on each
descriptor plus resolution outward through `ownerStack`.

### 1.3 `obj.is_a?(SomeModule)` after `include M` — a live halves divergence **[V]**
False in both ruby compiled halves, true in the interpreter. The compiler flattens
`include` at build time and never records the module on the class object; the fix
is a `$mix$<Name>` marker in `makeClassStmt` checked by `rubyIsA`/`rbIsA`.
Pre-existing, verified against base.

### 1.4 Calling a non-function is uncatchable, and the halves word it differently **[V]**
`xs[1]()` with `undefined` in the slot: the js interpreter says *call of a non
function value (method '1')*, `llvm.Run` says *unknown list method '1'*. node says
`xs[1] is not a function`, which needs the source expression text no engine keeps.
The site (`js_call`'s `die2` / `rt.call`) is used by every language's layer-2
internals, so making it throw risks masking engine bugs — a fourth
`js_rt_miss*`-style hook would work, and `d0f4db3` established that pattern.

### 1.5 A non-empty bounded ruby Range is not a value **[V]**
`p (1..3)` prints `[1, 2, 3]`, and `.begin`/`.end`/`.exclude_end?` abort with
*unknown Array method*. `d0f4db3`'s predecessor fixed the SLICING half without a
representation change (10 of 10 rows now match MRI against 5 of 10), but the range
as a VALUE needs `{__rrange}` for *all* bounded ranges plus the whole consumption
surface — for-in, each/map/sum/step/splat/===/inspect — in the emitter and both
compiled runtimes.

### 1.6 python's static binding analysis is absent **[V]**
`def h(): print(w); w = 1` raises `UnboundLocalError` in CPython and `NameError`
in all three engines here, because the name is a local by CPython's *static*
analysis, which no engine performs. Both halves agree, so `--cross` is blind.
Related and cheaper: `type(re.compile(…)).__name__` is `'object'` where CPython
says `'Pattern'` — not fixed because CPython's *AttributeError* for the same value
says `'re.Pattern'`, a different string, so a one-line arm trades one wrong answer
for another.

### 1.7 python's missing module and descriptor surface **[V]**
`import math` binds nothing (`math` is undefined in all three engines; `re` is the
only module with a bound object), `open` likewise; `staticmethod` and
`classmethod` are undefined; `list.count` as an unbound descriptor raises where
CPython answers `<method 'count' of 'list' objects>` (~40 lines per engine on the
`pyBoundInfo` record `9a64f07` added); and `len.__self__` needs a `builtins`
module object.

### 1.8 bash: three divergences from real bash 5.3, all found by the new feature file **[V]**
`${s^^[bn]}` ignores the pattern operand entirely (bash `BaNaNa`, ours `BANANA`);
**`-f` is a hard-coded path whitelist** because `rt_fskind` has no `stat()`, so
`[ -f /dev/null ]` is true here and false in bash, and `-f` on any real regular
file is false here and true in bash; and `$(( 10#$num ))` dies with *command not
found* because `base#digits` is only recognised when the prefix is not built by
expansion. Also **bash drops empty fields under a non-whitespace IFS** —
`IFS=:` on `a:b::c` gives 4 fields in bash and 3 here — which needs the POSIX
delimiter rule in `rt_splitifs` and touches every unquoted expansion, so it wants
its own ratchet section.

### 1.9 Compiled bash discards stderr entirely **[V]**
`rt_putc` drops every channel but 0, so `set -u` failures, bad regexes and `[[ ]]`
errors print NOTHING under `-exe` where real bash prints to fd 2. `rt_estr` in
`bash-rt.c` is now a working fd-2 writer the grammar could reuse (`4a2facd`).

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

### 2.2 The emitters write `frame`/`nvars`/`gvars` with no depth check **[V]**
`4a2facd` bounds-checked all nine ALLOCATION paths in `bash-rt.c`/`batch-rt.c`
(the `rt_limit` flag guarding four of them was written at four sites and read
nowhere), but the EMITTED code can still walk off two fixed arrays that no runtime
can defend: `batch-to-llvm-ir.abnf`'s `makeCall` increments `frame` past
`DEPTH=256` into `args[2560]`, and bash's emitter fills `gvars[1024]`/
`varnames[1024]` with no cap. `bat_shift` now refuses to compound the batch one.
Related and smaller: `%VAR:~N,M%` in batch is wrong for N past ~1184
(`%big:~11834,6%` is empty on an 11,840-byte value while `%big:~-6%` is right) —
likely the emitter's offset handling, not `rt_substr`; and whether `exit /b`
implicitly pops an open `setlocal` is unsettled here with no cmd.exe to ask.

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

### 3.2 A catchable `NameError` under the compiler — **NOW UNBLOCKED** **[V]**
The site is `js_scope_get`, shared by all sixteen languages and pinned by
clang-check and the SHOULD-ABORT rows. **The ceiling that blocked it is gone**:
`4d54d60` gave js/ts a real Error class hierarchy, and python and ruby already had
one, so a raised value can now BE a `ReferenceError`/`NameError` rather than a
string. What remains is deciding which `fail` sites become throws in each
language, and doing it without moving the SHOULD-ABORT rows. See 1.8.

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

### 4.1 The `.expected` fixture path is used by ONE program **[V]**
`c11e9c5` took `native-full.sh` from 15 languages / 29 native programs to
**20 / 37** and added the expected-output mode the toys needed. Only brainfuck
uses a fixture and only calculator uses `exit-only`; lisp and tinyc turned out to
already follow the `"…checks, N failures"` protocol. The mechanism is therefore
barely exercised — if a third fixture is ever added, check `--bless` still does
the right thing, and note the sabotage recipe that proved the whole thing works is
in that commit message.

### 4.2 The goja full-case-mapping fix is pinned in metajs only **[V]**
`1367c23` routed goja's `toUpperCase`/`toLowerCase` through the same
`strings.ToUpper` the frozen engine uses, which closed the divergence in **every**
language's interpreter half at once (`ruby#upcase`, `python .upper()`, …). Only
`tests/metajs-test-full.js` asserts it, so nothing would catch a regression in the
others. Related and harmless-but-invisible: `Math.random` is `function` under goja
and `undefined` under `-frozen` in the grammar host.

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

**`-min 60` is NOT "finished at 2 groups" — that number predates `a69a4fd`**,
which fixed shape-scan's brace counting (one-line bodies had been running on into
the next function). It is **34** today, and the default `-min 140` tier is **15
groups / 820 lines occupied / 481 recoverable** over 2,200 bodies — the numbers
`tests/gates.sh` now ratchets. Lowering them is this item's job.
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

### 5.2 The four `runtime-*.metajs` splits can be folded back **[V]**
`runtime-decimal` (188 lines), `runtime-bignum` (263), `runtime-jvm` (159),
`runtime-dartswift` (50) were split off **only** to keep the per-declaration
live-body tax off non-callers. The `scope_find` hash index removed that tax, so
they can go back into `runtime.metajs` whenever that is the tidier shape — a pure
tidy, no measurable cost either way. **This one really is free, and 5.1's warning
does not apply to it**: folding a module back moves DECLARATIONS, which the hash
index made free, and adds no call depth. It is the *delegation* that costs, not the
location.

### 5.3 `sw_kget`/`sw_kset`/`sw_safeget` are duplicated across three GRAMMARS **[V]**
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

### 7.1 Re-measure lua's hash-index buy-backs with draws **[V]**
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

### 7.2 Per-language coverage of the floor **[U]**
The instrumentation that found 24 unreached floor bodies was written once and
thrown away. Keeping it as `tools/floor-coverage` would make "is this body dead or
untested?" answerable at any time — **it is the question that gates Part B (3.3)**,
because a body no test reaches is a body no move can be validated against.

### 7.3 A `--why` flag for the emitters **[U]**
Understanding which extern an emitter chooses for an operator means reading
thousands of lines of grammar, repeatedly. A flag that prints the extern chosen
per AST node would make layer-2 work dramatically faster.

### 7.4 Make MetaJS's dialect gaps loud **[U]**
No exponent literal, no `toPrecision`, typed locals, ASI differences: each is
discovered by a confusing failure. A `-verify` pass over `lib/*.metajs` that names
them would pay for itself immediately. (`docs/abnf-dialect-gotchas.md` lists them;
nothing enforces them.)

### 7.5 On glibc < 2.34, the native link needs `-ldl -lpthread` **[U]**
`dlopen`/`dlsym` live in `libdl` and `pthread_create` in `libpthread` there, so
such a system needs both beside the new `-rdynamic`. Recorded in the code comment
rather than guessed at, and untestable here — this machine is darwin.

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

**~109 items over ten rounds** (`e284185`..`4d54d60`), ratchet to 7,766
assertions: the value model and numbers, coroutines and iterators, the six object
models, the python/ruby/kotlin library surfaces, and the tooling (`gates.sh`,
`probe.sh`, `bench.sh`, `shape-scan`, `gen-all.sh`, `-O2` — 2.2× on every native
binary — and a `scope_find` hash index worth 20–42%). Order and dates no longer
matter; per-item detail is in git history and in the commit messages, which are
written to be read.

## The lessons worth carrying — each paid for once

- **The instrument itself was wrong twice**: a single build cannot measure below
  ~2% (the layout lottery), and the MEDIAN is the wrong statistic on a bimodal
  sample. Both fixes are in `bench.sh`; the rules are at the top of this file.
- **Whenever a gate reports "nothing", ask what it would do if the thing WERE
  there** — three instruments were wrong that way, including a recorded fatal
  probe that was a tautology and could not fire.
- **Four regressions were invisible to every correctness gate**, one of them
  python +41% for four commits. That is why `--bench` exists and why 5.1 warns.
- **Deleting "dead" code needs a fatal probe that fires zero times, not an
  argument** — the one item that skipped it was wrong, and deleting as written
  would have introduced a defect.
- **Both halves agreeing is not evidence**; the largest class of defect closed
  here is "both halves agree and both are wrong", invisible to `--cross`. Two
  defects were found only by running a binary BY HAND, and two test files had been
  relying on defects — when a fix makes a test fail, establish which of the two is
  wrong before touching either.
