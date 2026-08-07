# Working on this project: what I wish I had known on day one

Written 2026-08-04 at `ee51718`, after taking all sixteen languages from
"emits IR nothing can link" to "self-contained native binary", building the C floor
and the MetaJS layer 2, adding a garbage collector, and merging the layer-2
duplication twice (the second time correctly).

**Revised the same day, through `1e1330e`.** Since then: a hash index for
`scope_find` (20-42% fewer instructions in thirteen languages), a fifth merge pass
that took the shape residue to two documented declines, kotlin's 1,489-line copy
of `regex.js` retired, three real defects fixed and **two "open defects" in this
very chapter found to have been already fixed** — and, running through all of it,
the discovery that a native binary's instruction count moves 4.8% with nothing but
the LENGTH OF ITS OUTPUT FILENAME, which invalidated every sub-2% single-build
measurement in these documents, including several I had quoted back to people as
facts. That is chapter 4's newest trap and it is the most expensive thing in here.

**Revised 2026-08-07** after ten more todo items closed. Two of them were WRONG
AS WRITTEN (`floExt` is live, not dead code - deleting it as the item implied
would have introduced a defect; and the C# `is` residue was every integral and
floating name, not one), and **seven of the ten turned up a further defect the
item did not name** - five of those live halves divergences `--cross` had never
reached. Chapter 4 gained five traps from that round, including one that no gate
in this project can see.

**Consolidated 2026-08-05.** There used to be four more documents beside this one -
three plan/record files totalling ~14,000 lines plus three concept notes. They were
read in full, their still-actionable content became [todo.md](todo.md) and the
mechanics chapter here, and they were deleted; see chapter 8 for how to read them
in git history. **This
document is now the only manual.** The one other file that is current and NOT
superseded is [abnf-dialect-gotchas.md](abnf-dialect-gotchas.md), the reference for
the ABNF tag dialect, which chapter 4 assumes you have.

How to read this: **chapters 1-3** are what you need to build and test anything.
**Chapter 4** is the traps, and it is the highest value-per-line in the file.
**Chapter 7** is the engine mechanics you would otherwise re-derive at some cost.

**What is still to do is NOT in this file.** It is in
[todo.md](todo.md) next to it, ordered by importance, with every item marked
either verified-this-pass or inherited-and-unchecked. This file explains how the
project works; that one says what is left.

---

# 1. The architecture in one page

A program is compiled by a *grammar*, and there are two grammars per language:
`languages/<lang>-interpreter.abnf` walks a tree, `languages/<lang>-to-llvm-ir.abnf`
emits LLVM IR. The suite compares them constantly, and that comparison is the whole
quality strategy.

The emitted IR comes in two flavours:

- **Self-contained IR** — `c`, `bash`, `batch` and the toys. Every value is an
  unboxed machine word; the runtime is C, compiled by our own `c-to-llvm-ir.abnf`
  into `languages/lib/<lang>-rt.ll`. Fast, small, and it links against libc alone.
- **Handle IR** — the other thirteen. Every value is an `i64` handle and the
  semantics live in `js_*` externs. Those externs are satisfied **two different
  ways**, and this is the single most important fact in the codebase:

```
        mec grammar prog.x            (no -exe)   ->  llvm.Run interprets the module,
                                                       js_* resolve to the GO TWIN
                                                       in abnf/jsrt*.go

        mec grammar prog.x -exe out   (native)    ->  clang links the module with
                                                       lib/runtime.ll  (the C floor)
                                                     + lib/<lang>-rt.ll (MetaJS layer 2)
```

**Two independent implementations of the same semantics.** That is not redundancy to
be eliminated — it is the instrument. Roughly fifteen real defects this session were
found because the two disagreed. Retiring the Go twin was measured and **declined**
(`a3baeee`); the reason is chapter 7.11, and it is stronger than a preference -
`llvm.Run` structurally *cannot* execute the C floor.

The layers, bottom up:

```
libc                     malloc, putchar, write, exit, setjmp/longjmp, pthread
languages/lib/runtime.c  the C FLOOR: ~35 irreducible primitives + the GC +
                         coroutines. Compiled by OUR c-to-llvm-ir.abnf.
languages/lib/*.metajs   LAYER 2: each language's semantics in MetaJS, compiled by
                         OUR metajs-to-llvm-ir.abnf. Shared parts in
                         runtime.metajs / runtime-decimal.metajs / runtime-bignum.metajs
languages/<lang>-to-llvm-ir.abnf   the emitter
```

Every layer is compiled by this repository. `nm -u` on any native binary shows libc
and nothing else.

---

# 2. Using the compiler

```bash
go build -o mec .

./mec languages/python-interpreter.abnf prog.py            # tree-walking half
./mec languages/python-to-llvm-ir.abnf  prog.py            # emit IR, then run it (llvm.Run)
./mec languages/python-to-llvm-ir.abnf  prog.py -qq        # JUST the program's output
./mec languages/python-to-llvm-ir.abnf  prog.py -q -exe a.out && ./a.out   # NATIVE
```

Flags you will actually use (full list in `main.go`'s header):

| flag | what it does |
|---|---|
| `-q` / `-qq` | suppress the stage chatter / **also suppress the grammar's DEFAULT OUTPUT**. See the note below — which one you want depends on which half you are running, and it is not obvious. |
| `-exe PATH` | link a native executable. Refuses loudly if the grammar has no `c.exePath`. |
| `-rt FILE` | link an extra `.c`/`.ll`/`.o`/`.a`. Declaring a runtime also makes an unresolved symbol a **hard error** instead of a zero-returning stub. |
| `-frozen` | run the tag scripts on the frozen MetaJS bootstrap instead of goja. **The matrix runs everything both ways and demands byte-identical output.** |
| `-i DIR` | add an include root for imports. |
| `-warn-imports` / `-warn-unsupported` | warn instead of aborting — how you get a call graph out of a partially-understood language. |
| `-max-steps N` | the IR interpreter's endless-loop brake (default 1e8). A big benchmark hits it. |
| `-freeze F` | regenerate the frozen bootstrap snapshot. Needed after **any** change to `metajs-to-llvm-ir.abnf` or `lib/compile-core.js`. |

**`-q` versus `-qq`, and why the two halves want different ones.** The module a
compiler grammar prints is not noise that `-q` failed to strip — **emitting IR is
that grammar's output**. `-qq` is what suppresses a grammar's own default output,
and the two halves therefore behave oppositely:

| | what `-q` gives you | what `-qq` gives you |
|---|---|---|
| `<lang>-interpreter.abnf` | the program's output (its output IS the default output) | **nothing** |
| `<lang>-to-llvm-ir.abnf` | the module, then the program's output | **exactly the program** — the module is the default output, and the program's prints arrive from `llvm.Run` on a side channel |

So for a differential probe you want `-q` on the interpreter half and `-qq` on the
compiler half, and then no parsing at all. `tests/probe.sh` does this.

Both engines also sign off on stdout — `<lang> interpreter: program ran to
completion`, and **three other wordings** (`program finished`, `program value
is`, and the `compiler:` form). That is the engine talking, and no real toolchain
says it, so it has to go before any comparison against an oracle means anything.
Matching only the first wording once made ruby's interpreter leg read one line
longer than the other three and look like a halves divergence.

**If you do need the module and the program separated without `-qq`**: scan
forward tracking brace depth. Do **not** look for the last line that looks like
IR, because `var_dump` prints a bare `}` and PHP will fool you.
`tests/clang-check.sh`'s `module_only()` is the reference implementation.

---

# 3. Testing

**Start here: `tests/gates.sh` runs every gate below in one command** and reports
one verdict, including the two greps that two of them are only correct *with*.
`--quick` skips `clang-check` and `native-full`, and says so loudly — those are
the only two gates that can see layer 2. `--gen` adds the generator check,
`--freeze` adds the snapshot fixed-point check. Use the individual commands when
you are iterating on one thing; use `gates.sh` before you believe you are done.

Four groups. They are not interchangeable and each is blind to something.

```bash
./test.sh                    # the MATRIX: .vscode/launch.json IS the test suite.
                             # Every entry runs twice, goja and -frozen, and the two
                             # must be byte-identical on stdout AND stderr.
./test.sh --full             # the RATCHET: tests/<lang>-test-full.<ext>, one file per
                             # language walking its whole syntax. Informational.
./test.sh --cross            # each language's TWO HALVES diffed against each other.
tests/clang-check.sh         # hand every emitted module to clang; for the languages
                             # that link, BUILD AND RUN the ratchet natively.
tests/native-full.sh         # build+run all fifteen ratchets natively, reporting the
                             # failure COUNT (clang-check reports a verdict).
go test ./abnf/              # Go-side unit tests.
tests/gen-all.sh --check     # the fifteen generators, in parallel over one shared
                             # binary (25s -> 3s): does each checked-in .ll still
                             # reproduce from its source? MANDATORY after any
                             # runtime.c / *-rt.metajs change. The individual
                             # tests/gen-<lang>-rt-ll.sh still work standalone.
tests/probe.sh L f [--oracle C]   # ONE program, FOUR ways, diffed - see below
tests/bench.sh [--draws N]   # native perf against checked-in baselines, in
                             # instructions retired, as a MEDIAN OVER LAYOUT DRAWS
go run ./tools/shape-scan    # layer-2 bodies that are identical modulo names
```

**What each group cannot see** — learn this or you will ship a defect:

- **The matrix compares each engine against ITSELF.** goja vs `-frozen` is one
  implementation under two script hosts. It cannot see a wrong answer both halves agree on.
- **`--full`'s summary line is not sufficient.** Always also
  `grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF'` over the full output. Until
  `52dff5e` a `-frozen`-only divergence never reached the summary at all.
- **`--cross` only sees what a test program exercises.** Several defects sat in
  code no test reached.
- **`./test.sh`, `--full` and `--cross` CANNOT SEE LAYER 2 AT ALL.** `llvm.Run` uses
  the Go twin. Proved by sabotage: dropping the forced `.0` from the shared
  formatter leaves `--full` green at 279 assertions while the native binary fails 7.
  **A layer-2 change is only tested by `clang-check.sh` / `native-full.sh` / a
  native `-exe` differential probe.**
- **`clang-check` checks the exit code**, so a ratchet that does not *assert* the
  behaviour will not catch it either.

## The differential probe — the technique that found most real defects

Write a program that prints a few thousand lines of results, run it under
`llvm.Run` and as a native `-exe` binary, and `diff`. Then run it under the **real
toolchain** and diff again. Almost every genuine defect this session came from one
of these, not from the suite.

Read every operand out of an array so the grammar's constant folder cannot evaluate
it at compile time — a probe of literals tests the folder, not the runtime.

---

# 4. The traps, each of which cost real time

**A rebuilt binary does not give you the old grammar.** `mec languages/x.abnf prog`
reads the grammar **from that path at run time**; the Go binary is only the engine.
Building an old commit's binary and running it from the repo root silently uses the
*working tree's* grammar. This made me compare a change against itself and tell an
agent its correct 172-line fix was a no-op. To compare versions:
`git archive <rev> | tar x -C /tmp/x`, `go build` **inside** `/tmp/x`, run it there.

**A DIFF OF TWO RUNS IS NOT A MEASUREMENT UNLESS BOTH RAN TO COMPLETION.** I
probed 60,000 doubles for a goja-vs-`-frozen` divergence and got ~61,000 differing
rows in *both* the base and the fixed tree. The `-frozen` leg had **truncated at
1,446 lines** against the default `-max-steps`; the diff was faithfully reporting
the truncation. Record the exit code AND the line count of every leg and check
them against the number of probe values before diffing. This is the same failure
that once made the suite report a phantom `FROZEN-DIFF` for a whole round.

**A crashing binary looks fast.** `/usr/bin/time` reports happily on a process that
died. I quoted bash at "0.44 s / 5.75 MB" as an argument against an architecture
change; it was segfaulting on every run and printing nothing. **Check the exit code
and the expected output**, not just the timing.

**A SINGLE BUILD CANNOT MEASURE ANYTHING BELOW ~2%, and this is the trap that
cost the most.** A native binary's instruction count depends on its CODE LAYOUT,
and layout moves for reasons that change no semantics whatsoever. One unchanged
kotlin source, varying nothing but the **length of the `-exe` output filename**:

```
17.353  17.494  17.620  17.911  17.915  17.916  17.916  18.018  18.023  18.182  G
```

**4.8%, bimodal, from a file name.** It is not the run (re-running one binary
reproduces to 0.02%) and not codegen (rebuilding one source to one path is
`md5`-identical). The module's own content perturbs layout the same way and by the
same magnitude — so **holding the output path fixed cancels nothing**, and there
is no pairing that does. Three separate attempts to measure ONE change this way
returned **+0.04%, +0.65% and −1.72%**.

Use `tests/bench.sh --draws N`: it builds each program several ways, reports the
MEDIAN with the observed spread beside it, and **ignores any delta smaller than
that row's own spread** — java has been seen at +3.20% against a 3.21% spread.
Per-row floors: `c` 0.35%, kotlin 1.05%, python 1.31%, lua 2.04%, java 3.21%,
ruby 4.14%.

**Counters are exempt and are exact**: allocation bytes, `MEC_GC_STATS`, binary
size, `sample` frame counts. Measure those once. It is instruction counts and
wall clock that need draws — and the corollary nobody expects: **agreement is not
confirmation either.** Two builds landing on the same number inside a
percent-wide lottery is a coin landing heads, and several conclusions in the plan
documents were drawn from exactly that.

**An unprefixed helper in `lib/compile-core.js` can SILENTLY REPLACE a grammar's
own.** `python-to-llvm-ir.abnf` defines `numText`; a change to the java float
literal added a `numText` to the shared core, and python's module grew two
big-integer literals and its goja and `-frozen` halves diverged by **13,087
lines** — from a commit with nothing about numbers in it. **No gate caught it.**
Only diffing the emitted module against a clean `git archive` of the base did.
Prefix anything you add to a shared core (`coreNum*`), and after touching
`compile-core.js` diff the emitted modules of every language, not just yours.

**PEG ordered choice never comes back, so a two-name form can eat a multi-assign's
first element.** `MapOk`, `TypeOk`/`AssertOk` and `RecvOk` are tried before
`ShortDecl` in go's `SimpleStmt`; each matched the *first element* of
`a, b := xs[i], ys[j]` and the statement was then rejected at the comma rather
than re-read. The fix is a `!","` guard on the narrower rules. The general lesson
is that **a rule which can match a PREFIX of a later alternative must exclude the
continuation explicitly** — the grammar cannot discover it by backtracking.

**goja's `Number→String` is not always the shortest round-tripping form.**
Measured: **22 of 199,904 random doubles (~1 in 9,000)**, and at some
subnormal boundaries it emits a non-digit character outright
(`-;.627494252092118e-309`). The frozen host is exact, so **any grammar that
takes a double's digits from `"" + n` has a live goja-vs-`-frozen` divergence**
waiting for the first value that reaches it. Choose the digit count by
round-trip (`floPrec(a, n)` for n = 1..17, stop when `parseFloat` agrees), never
by counting the host's own printed digits. java is fixed; **lua, kotlin, csharp
and go's interpreter halves still do this** (todo.md).

**A RANDOM float probe has near-zero discriminating power; targeted values have
all of it.** A 1,450-line random probe agreed with the oracle at base while 85
hand-picked values gave 170 differing lines and an outright abort. Sample the
boundaries — 17-significant-digit values, subnormals, `MIN_VALUE`, the exponent
thresholds — not the space.

**Only the NATIVE leg sees some IR errors.** Building a phi inline inside a
`js_scope_set` argument list compiled, verified and ran correctly under
`llvm.Run`, then failed `clang` with *PHI nodes not grouped at top of basic
block* — because `emitStr` emits an instruction into the block before the phi is
constructed. `llvm.Run` is more permissive than clang; a module that runs is not
a module that links.

**A fix spanning engines lands with its assertions in ONE commit.** I split a pair —
the twin half went in without its layer-2 line — and HEAD's native binary failed two
assertions while `llvm.Run` passed. Backing out only the twin made it *worse* (both
halves wrong, assertions committed, suite red). It is three things: twin, layer 2,
assertions. Often it is **five** — several defects this session turned out to live in
the Go twin, layer 2, the `*-interpreter.abnf`, the C floor **and**
`metajs-interpreter.abnf`. Grep for every implementation before you start.

**MetaJS is typed.** A reassigned local keeps its first type, so a Go-style
`x = normalize(x)` dies with *"variable 'x' has type boolean and cannot hold a
number"* — and in the interpreter **only under `-frozen`**, so `--full` reads
`FROZEN-DIFF` while goja is green. Use out-of-line helpers.

**`x * 0` does not give a signed zero** in the ABNF tag dialect: the engine keeps
integral values as integers, so `-4 * 0` is `+0`. Divide a zero by −1 instead.
Likewise **unary minus must be `-x`, never `0 - x`** — that loses `-0.0`'s sign, and
it has bitten `d_pow`, `d_mod_go`, Swift's `rounded()`, Java, Dart's `math.Mod` and
Ruby's interpreter.

**MetaJS has no exponent literal.** `1e308` parses as the identifier `e308`. Build
an infinity as `1e307 * 10`.

**Brace value-less early returns.** `if (bad) return` followed by an expression on
the next line parses as `return (expr)` — real JS applies ASI, the frozen parser does
not. It silently lost *every store* in one half for an entire agent-round. See
`docs/abnf-dialect-gotchas.md:539`.

**Write layer 2 against the Go CODE, not the Go COMMENTS.** `jsrtdart.go`'s comment
says two Lists concatenate; `rt.jsAdd` renders both sides and concatenates the text.
Layer 2 was written to the comment.

**Regenerate the frozen snapshot, and check it is a fixed point.** Any change to
`metajs-to-llvm-ir.abnf` or `lib/compile-core.js` (which the snapshot inlines) needs
`mec -freeze languages/metajs-to-llvm-ir.abnf`, and a second `-freeze` must produce
a byte-identical file.

**Verify a commit from a clean checkout.** Committing by explicit path narrows what
gets swept in; it does not stop a needed file being left *out*. Swift once shipped
without the Go file defining its externs and a fresh checkout failed three matrix
entries. `git archive HEAD | tar x -C /tmp/x` and run the suite there.

---

# 5. The rules that made the work reliable

- **A change that fixes N and breaks M>N is a LOSS.** Revert it and record the
  measurement at the site so nobody re-derives it. This is why `case_map` and
  kotlin's `regex.js` import are documented rather than shipped.
- **Measure before implementing.** `jsdispatch` was struck because LLVM already emits
  jump tables and the apparent O(n) was the *collector*. `case_map` passed every gate
  and was reverted at 135× on a real workload.
- **Measure a new assertion's discriminating power.** Run it against a clean archive
  of the parent commit: how many fail? An assertion that passes either way is
  decoration, and saying so is worth more than pretending otherwise.
- **Verify claims rather than inheriting them.** The shift-result-type item, the
  `js_jadd` "divergence", the `floStr` complaint, the lua 15/16/17 rule and the
  "five halves" counts were all wrong as written — including several I wrote myself.
- **Null results are results.** `js_dict_new` was six byte-identical copies with zero
  knobs; the shift-result-type guard fires zero times. Both are worth recording.
- **The defect class that keeps paying out is "both halves agree and both are wrong."**
  Byte-identity cannot see it by construction. Only an external oracle can.

## Oracles on this machine

Installed and usable: `go`, `node`, `python3` 3.14.6, `java` 24.0.2,
`swiftc` 6.1.2, `lua` 5.5.0, `bash` 5.3, `cc`/`clang` 17.
**Absent**: `dart`, `kotlinc`, any C# toolchain — cite the language spec and say so.
`/usr/bin/ruby` is **2.6.10 (2018)** and rejects every Ruby-3 form the corpus uses;
it once produced nine false "hidden defects". Its `Float#to_s` and `Float#%` are
unchanged in 3.x, so *float* rows are settleable there.

`node --check` wraps a file in a CommonJS function and legalises top-level `return`;
use `new vm.Script(src)` for the Script goal.

---

# 6. Working with agents on this

Everything above is why the agent briefs in this repo look the way they do. What
worked:

- **Agents never commit.** The coordinator verifies and commits by explicit path.
  Durable state lives in git, never in an agent's context — so an agent can die,
  be compressed, or run dry at no cost.
- **Every agent writes findings into the plan documents**, which get committed with
  the code. Knowledge outlives the context that produced it.
- **Verify in isolation.** `git archive <base> | tar x -C /tmp/x`, copy in only that
  agent's files, build inside, run the suite. That is how you tell one agent's
  regression from another's in-flight edit.
- **Disjoint file sets, stated explicitly**, and a named owner for every shared file
  (`runtime.c`, `abnf/jsrt.go`, `metajs-to-llvm-ir.abnf`, `.vscode/launch.json`).
- **Wait on the subagent, never poll a file's mtime.** One agent burned hours in
  `ls -l` loops waiting for a child that had already finished and reported elsewhere.
- **The plan list goes stale within two commits.** It happened three times. An item's
  verdict is only as good as the commit it names — **re-grep the implementation
  before acting on any line of it.**

---

# 7. Engine mechanics you would otherwise re-derive

Everything here was paid for once. It is the part of the deleted plan documents
worth keeping, and most of it is invisible from the code alone.

## 7.1 The `__raw` rule — the defect that compiles, links, and passes everything

`compile-core`'s `truthy()` reads a `callExt` result as a **raw integer**
(`NewICmp(IPredNE, …, handle(0))`). A layer-2 shim that returns MetaJS `false`
returns handle **2**, and `2 != 0` — so **every condition in the program is
taken.** It compiles, it links, and nothing in the suite looks at it. A `__raw`
suffix on the layer-2 function name fixes it.

You need it exactly when the grammar **overrides `core.truthyExt`**, so **grep the
ASSIGNMENT, not the read, and grep inside the `-exe` arm** — js/ts are the first
language where the answer differs between engines (`llvm.Run` keeps the floor's
`js_truthy`; the native build overrides it for BigInt). Tally: lua, php, python
need it; swift, java, csharp, dart, go, ruby, kotlin do not.

The general form is broader than truthiness: **any emitter consuming a `callExt`
result as a number** needs the check, and `grep NewICmp` over the grammar finds
them. `js_ctl_kind` is the second instance — it answers a raw 0/1/2/3 (0 not a
signal, 1 return, 2 break, 3 continue), so the test must be `IPredEQ …, 1`,
because `truthy()` silently accepts break and continue too. Ruby goes furthest and
overrides `truthy()` itself with `IPredUGE, v, handle(3)`: handles 0/1/2 are
undefined/null/false and 3 is true, so `>= 3` *is* Ruby truthiness with no call at
all — sound natively only because `runtime.c` seeds the same four handles.

## 7.2 Adding or moving anything at the engine boundary

- **A host global means THREE engines**: `seed_root(...)` in
  `languages/lib/runtime.c`, the bindings map in `abnf/jsrtint.go`/`jsrtjvm.go`,
  and `hostGlobals` in `languages/metajs-interpreter.abnf`. A layer-2 file written
  against the floor alone links natively and dies under `llvm.Run` with
  *variable not defined: sint*.
- **Two `function js_*` of the same name in one layer-2 file** → `error:
  redefinition of global '@jsrtlib_f_js_<name>'`. A duplicate symbol *used to* be
  reported as an unresolved one, which is why **a floor addition and its layer-2
  deletions must land in ONE commit**. `grep -l 'function js_<name>'
  languages/lib/*.metajs` before moving any body into `runtime.c`.
- **`-rt-lib` renumbers `funcCount`/`strCount` to 1,000,000** — every new
  generated module-level name owes the same line, or clang says
  `duplicate symbol '_jsnum.1'`.
- **Every floor → layer-2 upcall touches FIVE files, not four.**
  `tests/coro-poc/build.sh` links `tests/coro-poc/gen.ll` against the floor with
  **no layer 2 at all**, so a new upcall breaks it with `Undefined symbols:
  _js_case_map`. The matrix, `--cross` and even `clang-check` are all blind to it.
  A floor body calling into layer 2 also needs `long to_number(long h);` and
  `long mk_bool(int b);` declared, or the floor emits an **i32** return and the
  module fails to verify.
- **`metajs-rt.ll` can go stale on its own** — uniquely, the grammar that compiles
  `lib/metajs-rt.metajs` is the grammar that links its output.

## 7.3 The private-representation probe — three answers, one safety property

When the Go twin represents something as a Go TYPE that layer 2 can only
represent as an object:

- **(a) lower a SCOPE probe into the emitter** (php, swift, dart, ruby —
  `js_scope_typeof`/`js_scope_get`). `sw_kget`/`sw_kset`/`sw_safeget` were
  transferred verbatim to dart and kotlin, so **look the extern up in
  `abnf/jsrt.go` first — if it is `js_kget`/`js_kset`, the lowering already
  exists.**
- **(b) lower a CONTROL-SIGNAL probe** (go, over tag-10 `js_ctl_kind`).
- **(c) when the floor keeps no such state at all, layer 2 keeps it and the
  EMITTER hands it over** — `js_this`/`js_newtarget` read the Go runtime's own
  call stack and `js_call` drops `self`, so the emitter emits
  `js_jssetthis(receiver)` before every call site it owns. No restore is needed:
  `this` is read once per function at entry.

The rule for all four JS walls (`this`, the accessor box, `typeof`, the BigInt
operators) is **route the OPERATOR through layer 2, in the emitter, under `-exe`
ONLY** — no floor change, no Go twin change. **Then diff the emitted non-native
module against a clean `git archive` of the base commit**: one `git archive` plus
one `diff` turns "will the matrix survive this" into a measurement.

Two ceilings: **there is no scope-PARENT accessor reachable from an emitter** (why
kotlin's chain walks stayed — though see §7.27, the floor has one now), and
**`js_scope_typeof` cannot distinguish "holds undefined" from "absent"**, which
became a hard failure in python (`declCompNames` declares comprehension targets
holding undefined, so `[i*i for i in range(4)]` answered `[NaN,NaN,NaN,NaN]`).

## 7.4 Grep the interpreter before porting a Go twin

**Every language already has a complete MetaJS implementation of its own semantics
in `languages/<lang>-interpreter.abnf`'s start script** — in the same frozen
subset a layer-2 file is written in (no `for..in`, no `split`, no exponent
literals: exactly what a fresh translation gets wrong), kept in agreement with the
Go twin by the matrix. js lifted `jsStr`/`jsToPrim`/`jsLooseEq`/`jsBuiltin`/
`jsStringMethod`/the whole BigInt family this way. Only what touches the VALUE
MODEL cannot be lifted.

Related: **a layer-2 file may `import "./regex.js"`** — `metajs-to-llvm-ir.abnf`
resolves `import "./x"` relative to the file, and `-rt-lib`'s lazy boot runs the
imported top-level declarations into the same pinned scope. Ruby's regex glue cost
~460 lines instead of ~2,000; python's ~660 instead of ~2,200; js's 548 instead of
~2,700.

Two cheap greps before costing anything: `grep -c 'js_genfn\|js_yield'
languages/<lang>-to-llvm-ir.abnf` before costing a coroutine (dart, go, csharp,
kotlin, ruby all answer 0 — their `sync*`/`suspend`/iterators are lowered by the
emitter), and `grep -n 'math.Floor(.*)\*' abnf/jsrt*.go` before assuming an FMA
divergence.

## 7.5 The value model: sized integers, and where each language unboxes

**Ask WHERE a language's unboxing boundary sits, not whether it has a 64-bit
type.** `si_norm` unboxes any signed 64-bit value a double holds exactly. Go and
Dart want that (plain `sintOp`, zero wrapper lines). **Kotlin says 32** — a `Long`
is a box at every magnitude, because an unboxed small Long makes
`1000000L*1000000L` a 32-bit multiply answering `-727379968`. Java says nowhere.
Ruby and Python say "no meaning" (Ruby's Integer is a plain double and its Float a
`{__flo}` box, so `7/2==3` and `7.0/2==3.5` both hold).

`sintRaw(hi, lo, bits, uns)` answers all three non-Go cases. **The invariant that
goes with it is SILENT when missed**: `si_apply` ends in `si_norm`, so `2L + 3L`
comes back as a plain number — as an *int* to everything downstream. Every
long-producing `sintOp` must go through a re-box (`sintRaw(sintHi(t), sintLo(t),
64, false)`, spelled `csNorm` in C#, `jlOp`/`jlShr` in java). Mutation-proved: not
re-boxing fails 7 assertions.

## 7.6 The float box — the precedent to copy, and its two surprises

Two float mechanisms exist. The **tag-14 floor box** (java/kotlin/go/csharp;
`runtime.c:2449-2532`, host ids 51-60) is the **wrong** vehicle for a dynamically
typed language: its rule is "either operand is a box ⇒ result is a box", but
Python's `/` must make a float from two ints, and it drags a fourth print style
into the floor, `jsrtjvm.go` and `metajs-interpreter.abnf`. The right vehicle is
the **plain-object box** (ruby `{__flo}`, php `{__pf}`, dart `{__flo}`) — a Go
twin type per language, **zero lines in the C floor**. Estimate was 700–950 lines;
actual ~640 across four files.

Two risks behaved surprisingly, and they are the reusable half:

- **Dict/set key aliasing must be tried TWICE.** A box's alias is the plain
  number, and the number's alias is the bool, so `{True:"t"}[1.0]` is two hops.
  The reverse direction is unreachable by alias, so all three engines grew a
  key-equality hook on the fallback SCAN.
- **`is`/`strictEq` UNDER-reads.** `1.0 is 1.0` is true in CPython while
  `float('nan') is float('nan')` is false — which no VALUE struct can express.
  Hence the box is a POINTER and all three engines run
  `(a === b) || (both floats && doubles ===)`. Found by a ratchet assertion, not
  by reasoning.

## 7.7 Numbers, formatting, and what cannot move out of C

**`d_pow`/`d_exp`/`d_log`/`d_sqrt`/`d_frexp`/`d_ldexp`/`d_modf_int` cannot move
into MetaJS, and the argument is circular rather than a preference**: they are not
helpers on top of doubles, they ARE the double — each reads or writes the IEEE-754
layout through `union DB`. MetaJS has no union and no access to a double's bits,
and its numbers *are* these doubles, so a MetaJS `pow` written on `*` and `/`
lowers to `__mec_dmul`/`__mec_ddiv` in the same file. Same for `d_mod`,
`d_mod_go`, `d_is_nan/inf/zero`, `d_sign`, `d_neg`, `d_abs`, `d_from_long`,
`d_to_long`, `d_trunc`, `d_floor`. Separately, **strike `d_frexp`/`d_ldexp`/
`d_mod` from any candidate list on hotness**: `metajs-bench-try.js` does `i % 3`
once per iteration and calls them 143,792 / 123,793 / 20,000 times.

**The exact float formatter, and why the obvious one is wrong twice.**
`strconv.FormatFloat` is EXACT and rounds **half to even**.
`Math.floor(a*10^p + 0.5)` is wrong both because it rounds half away from zero
*and* because it cannot reach digits past 2^53. The replacement is the exact
decimal expansion of the double (`m*2^e`, or `m*5^-e` with the point moved) on a
little-endian base-10000 bignum plus a half-to-even rounder — it exists,
language-neutral, as `rtExactDec`/`rtRoundHalfEven`/`rtFixed` in `runtime.metajs`.
**Half-up-vs-half-even has been a live divergence three separate times and
`--cross` was silent every time.** Also: **`printf` width and precision count
BYTES, not characters** — `"%10s" % "café"` gets five spaces; `byteLen` is the
floor's answer.

**Only the digit machinery is shared; the renderers are per-language on purpose.**
CPython: `str == repr`, exponential iff `decpt <= -4 || decpt > 16`, **no** forced
`.0` in exponential form, exponent zero-padded to ≥2 digits, lowercase
`nan`/`inf`. Ruby: `decpt < -3 || decpt > 15`, forces `.0`, `NaN`/`Infinity`,
byte-padded. Java: **value**-window `1e-3 <= |d| < 1e7`, forces `.0`, upper `E`,
**two** minimum significant digits — and the forced second digit belongs to the
*significand*, computed against the ACTUAL value (`4.9E-324`, not `5.0E-324`).
Dart: ECMAScript's.

**lua's `%g` window is 15 then 17, never 16**, and `floPrec`'s contract is "AT
LEAST n digits". lua 5.5.0 tries `"%.15g"` and, if it does not read back,
`"%.17g"`; a 15/16/17 model differs on 10 of 91 probe values. The `%g` threshold
is `exp < -4 || exp >= P` with **P the precision actually used**. The trap on top:
`floPrec(v, n)` answers the *shortest round-tripping* form when n is smaller than
that, so `%e` of `1234.5678` came out `1.234567e+03`. Every existing caller asks
n ≥ shortest, where the contracts coincide.

## 7.8 The garbage collector — decisions, not omissions

It **never moves an object** (conservative roots forbid it), **never returns
memory to the OS**, is stop-the-world, and **only `runtime.c` has one** —
`bash-rt.c`, `batch-rt.c` and the C interpreter keep uncollected arenas.

- **Requiring exact payload addresses is unsound at `-O2`.** In `u_slice` the only
  surviving reference while `str_from_units` allocated was `u + b`, an interior
  pointer, and clang had already dropped the owning handle — 9 `string.sub`
  failures under `MEC_GC=stress`, while `off`/`auto`/`poison` saw nothing.
  Interior pointers resolve by `idx = (v - base) / bsize`.
- **Any new emitted module-level global holding a handle must be pinned with
  `js_gc_pin`** — the emitted module has globals the floor cannot see. `js_gc_pin`
  is the identity in both halves.
- **`MEC_GC` is the only reason those defects were findable**: `off`, default
  (collect when bytes-since-last > live set, floor 1 MB), `stress`, `poison`
  (retire and fill every swept block, so a missed root is a wrong answer rather
  than lucky bytes), plus `MEC_GC_STATS=1`. `poison` is deliberately **not** also
  `stress` — that combination is quadratic. **`stress` is a correctness tool, not
  a benchmark mode**: it collects at every allocation, so size the program in
  hundreds of iterations, not tens of thousands.
- **A shadow stack was costed and declined** (IR the Go half carries and ignores,
  at every temporary, in fifteen emitters, forever), and **refcounting is settled
  as wrong here**, not a preference: a scope holds its parent and a closure holds
  its defining scope, so a closure defined inside a function cycles with the scope
  that names it — the common case.
- **The coroutine GC contract**: the running side owns `GC_STACK_BASE`; a registry
  of suspended stacks publishes `sp` plus a `setjmp` of callee-saved registers
  before parking; no handle lives in the malloc'd control block; nothing moves.
  **The failure modes are SILENT** — a resumer stack not scanned prints 1 instead
  of 1234; `GC_STACK_BASE` not switched on resume prints `<nil>` instead of 42. A
  single global `jmp_buf` pool is a SIGSEGV.

## 7.9 Behaviours that are the specification, not slips — do not "fix" them

- **`box == 5` is false while `box === 5` is true.** `loose_eq` sees a box as
  neither number nor string; `strict_eq` has an explicit tag-13/14 arm. The C
  floor and the Go twin do this independently and identically.
- **`===` in layer 2 is NOT JavaScript identity.** It lowers to the floor's
  `strict_eq`, whose first four lines compare a boxed double and a sized integer
  BY VALUE, in `rt.strictEq`'s own arm order — and `rt.strictEq` is what
  `rt.dictFind` finds keys with. So a bare `===` key scan **is** the oracle. The
  one exception is a Char box: `{__char}` is an ordinary object to `strict_eq`, so
  a `Dictionary<char,V>` misses every key without an unboxing arm.
- **Truthiness of tag 13 and tag 14 is not converted in the interpreter half** —
  `if (flo(0))` is false natively and true in the interpreter. Deliberately
  unasserted.
- **A JS BigInt is INTERNED and never freed, and that is load-bearing**: the
  floor's `strict_eq` has no BigInt case and falls to identity, so without one
  canonical object per value `10n === 10n` is false at every unrouted site. Ruby's
  Symbols are interned for the same reason.
- **`keysOf` hides `__`-prefixed slots** and answers own keys in insertion order
  (maintained by hand in `makeObject`/`makeAssign`/`makeIncDec`/`makePreIncDec` —
  any new object-write site must maintain it), but the Go twin's `clone` walks
  `o.keys` raw and copies `__class` by accident. **Every language that clones
  through `keysOf` must ask which hidden slots the copy must keep.**
- **`giFromFloat` of an infinity is architecture-dependent in the Go twin only**
  (`int64(NaN)` is 0 on arm64, `math.MinInt64` on amd64), which is why SECTION 24
  asserts nothing about an infinity.
- **The `guard < 64` cap on the `__class`/`__super` walk** is kept for all six
  languages even though the oracle has no cap, **because the oracle HANGS on a
  cyclic `__super`** and no language here can express a 65-deep hierarchy.

**Deliberate non-convergences with real toolchains, measured and LEFT.** Treat
these as the specification of this project, not as a backlog — each was probed and
a decision was taken:

- **C**: struct alignment is packed (`{char;short;int;long}` is 15 bytes where
  `cc` says 16); `_Bool b = 5` holds 5; identical string literals are not merged
  (unspecified in C); `c-interpreter.abnf` deliberately lacks the varargs `printf`
  family, `exit`/`abort`, `qsort`/`bsearch` and `malloc`/`free`.
- **Bash**: a string whose first byte is `0x02` is read as the in-band field-LIST
  marker, so `${#v}` of a lone `$'\x02'` is 0 where bash says 1 (fixing it means
  re-plumbing the field encoding in both grammars); both our engines accept
  `[[:word:]]` and `[[:ascii:]]` which bash rejects, and converging means changing
  `regex.js`, shared by ten languages; bash *errors* on
  `declare -A f=([k]=v one two)` where we accept; bash's associative-array
  iteration is hash order (unspecified) against our insertion order; and `rt_eg`'s
  `*`/`+` arm evaluates itself at `k == 0` and discards the result, because the IR
  computes both operands of an `And` and the arena offsets are observable through
  pointer identity.
- **Lua**: `pairs` order is unspecified in Lua itself, so three ratchet assertions
  cannot match any particular answer; `-0.0` unary minus differs in the compiler
  half.
- **Swift**: `Optional(3)` needs an Optional box in the value model (`Int? = 3`
  *is* `3`), so recovering it needs the declared type at the print site, i.e. a
  type checker; class instances print a bare type name where real Swift
  module-qualifies (`sp2.C`), and there is no module concept here.
- **Go**: `&P{1}` prints `{1}` — in both halves a struct value *is* its own
  reference, so `P{1}` and `&P{1}` are the same object and `fmt` distinguishes
  them by static type. A real pointer cell pulls in auto-deref, method dispatch,
  assignment targets and equality.
- **PHP**: `Generator::send()` works in the compiler and cannot work in the
  interpreter without a suspendable body — a model difference, stated in both
  `:description` blocks.
- **C#**: no dotnet on this machine, so the spec-cited values are the least
  confident thing here; the shakiest is that `System.Array` does not *override*
  `ToString`, which is what makes `int[]` render `System.Int32[]`.
- **Dart**: `Map`/`Set`/`Record` `toString` is spec-derived rather than
  corpus-derived, and `Record.toString` is explicitly unspecified in `dart:core`.
- **Python**: a tuple prints as a list (see todo.md 3.1 for why a real tuple type
  is atomic and ~640 lines).

Per-language divergences from real toolchains, recorded and permanent: Go byte-
slicing a rune in half cannot be represented at all (the floor's strings are
UTF-16); Go counts `len([]byte(s))` in RUNES and prints `%q` as UTF-8 bytes. An
astral character prints as two U+FFFD in swift. JS: `(-0.4).toFixed(0)` is `"0"`,
`(0.1).toString(3)` emits 52 fraction digits, the `d` flag builds no `m.indices`.
Python: integral floats render as int, `&|^` are ToInt32, `inf`/`nan` spell
`Infinity`/`NaN`, no `ZeroDivisionError`, the format mini-language is a subset,
and **`js_pylen` counts CODE POINTS while `js_pyget`'s string indexing counts
UTF-16 CODE UNITS** — an `abnf/jsrt.go` inconsistency reproduced rather than
repaired. `abnf/jsrtphp.go` misreads `"1e400"` as 1 where both our halves answer
INF (the PHP spec agrees with us) — sole cause of 431 hunks in a 40,948-line
probe, and permanent, since retiring the Go twin was declined.
`c-interpreter.abnf` deliberately lacks the varargs `printf` family,
`exit`/`abort`, `qsort`/`bsearch` and `malloc`/`free`.

## 7.10 Dialect and tooling snags that cost an afternoon each

- **`(255).toString(16)` works under goja and ABORTS under `-frozen`** ("call of a
  non function value"). Use an explicit digit loop.
- **`0 - 0.0` is `+0.0`** and the tag engine keeps integral values as integers, so
  the working spelling for a negative zero is **`0 / (0 - 1)`**. Same root bug in
  the floor's `d_ldexp`, because `c-to-llvm-ir.abnf` emits unary minus on a double
  as `0.0 - x` — write the sign BIT.
- **`sum()` needs `var t = anytype`** in the interpreter: MetaJS is typed and an
  accumulator that starts as the number `0` cannot later hold a box.
- **awk truncates a string at a NUL**, so a probe printing a control character
  reports a phantom defect — both the comparison script and `strip.awk` cut the
  module off stdout with awk. Compute the cut on a NUL-free copy and apply it to
  the raw bytes with `tail -n +K`.
- **`mec … -exe PATH` only BUILDS** (links, prints the path, exits 0), so a bare
  `-exe` matrix row asks "does this link" and nothing more. `SHOULD ABORT` means
  build-then-run-and-require-nonzero. The word "fail" is avoided because any test
  name containing FAIL means "the metacompiler must exit non-zero" — the opposite.
- **Assertions must be written at the width where the rule can bite.** `>>`
  clamping, `<<` at count ≥ width and unsigned division all measured **0 of 4**
  discriminating power as first written, because every operand was 8-bit.
- **`select` in an emitter evaluates BOTH operands.** lua's numeric `for` test
  made two layer-2 calls per iteration and discarded one; fixing that plus
  hoisting a loop-invariant sign was 4 of 11 layer-2 calls per iteration, **29% of
  the whole allocation saving**.
- **`jsdispatch` is not a linear chain in practice** — LLVM lowers it to jump
  tables, and callee index 0 vs 799 in an 800-function module is 0.89 s vs 0.88 s
  over 3M indirect calls. The growth with module size was the COLLECTOR.
- **Widening the small-integer cache is not worth it**: 18 of 26 number cells per
  iteration are layer 2's own guard constants, and a wider cache reaches at most
  73 bytes of 3,616 for a 512 KB table.

## 7.11 Why `llvm.Run` can never run layer 2

The link is fine, and `getenv`/`write`/`exit` are 9-line stubs. `setjmp`/`longjmp`
is ~100 lines and every `throw` in all sixteen languages goes through it;
`pthread_create` + `dlopen`/`dlsym` are worse, since `dlsym` must answer a real
code address where the interpreter's function-as-value is a funcId in an i32.

**The unfixable part is the collector: it has nothing to scan.** `gc_collect`
walks `[sp, GC_STACK_BASE)`, but under `llvm.Run` an SSA value lives in `fr.regs`,
a Go slice outside the interpreted address space, and an `alloca` is bump-
allocated upward and never freed. Measured silently wrong: a probe returns 9
normally and **0 under `PROBE_GC=s`** — an array collected while live.

The only timeable alternative (build with `-exe` inside `./test.sh`) costs 2.6×,
reaches only the compiler halves, makes clang a hard dependency, and breaks the
tool's primary invocation. **This is why the Go twin stays**, and if it were ever
retired, 12,713 lines could not go under any answer — `abnf/jsrt.go` (9,837) **is
the engine `-frozen` itself runs on**, and `jsrtint.go` (682) holds floor host ids
61/62/63, so **deleting it breaks the NATIVE path, not the Go one**.

## 7.12 Architecture decisions taken AGAINST, so nobody re-litigates

- **No `lib/kotlin-common.js`, and no `lib/js-core.js` for the js/ts/metajs
  family.** Helpers byte-identical across a language's two grammars
  (`splitMembers`, `kCharCode`, the `retLabels` stack) or across the three
  js-family grammars (`makeAnd`, `makeOr`, `makeCond`, `makeSwitch`, …) are
  **deliberately left duplicated**: no language pair here has a `*-common.js`,
  each interpreter/compiler pair is self-contained beyond the generic
  `interp-core.js`/`compile-core.js`, and a family-only shared file would trade
  harmless duplication for an inconsistency with fifteen other languages. Most of
  those names also exist with **different** bodies in c/csharp/java/go/swift, so
  core is not a legal home either — and the goja-hoist-vs-frozen-textual-position
  rule would make the two engines pick opposite winners.
- **`makeThrow` stays per-language**: eight compilers share its shape, but
  **dart's is a different function** — Dart's `throw` is an *expression*, so it
  returns `{b, v: hUndef}` instead of terminating the block with `deadBlock()`.
- **`makeTry`, `makeReturn`, `makeAssign`, `makeArray`, `mcall`, `makeIncDec` look
  shared and are not** — they encode each language's return-signal protocol,
  control-signal wiring, handle constants and method dispatch. Do not
  "consolidate" them.
- **`js_has` and `js_goslice` stay split permanently**: Go counts a string in
  BYTES, C# in UTF-16 code units. Two specifications, not a dedup debt.
- **A shared `lib/str-rt.c` between `bash-rt.c` and `batch-rt.c`** was never built
  and should not be: the saving is source lines only, across two runtimes whose
  ABIs and arenas differ.
- **The duplicated-extern-NAME view must not be used to plan a merge.** It
  understates (internal helpers are language-prefixed, so identical logic never
  collides by name — the shape scan found 24 groups / 776 lines invisible to it)
  **and** overstates (of 24 `js_*` names defined in more than one file, only two
  have bodies that group at any threshold; `js_mcall`'s six copies are 45, 1, 73,
  4, 15 and 51 lines). `tools/shape-scan` is the measure.

## 7.14 A suspension signal is a host THROW, and every host `try` it passes runs

The interpreter halves implement generators by **replay**: `next()` re-runs the
body from the top, replays recorded sends, and stops at the next yield by
**throwing** a marker (`{__genYield}` in python, the same shape in js). That
marker travels through the *host* engine's exception machinery — so **every
`try`/`finally` and every `with` in the interpreted program's own machinery ran
on the way past.**

The visible consequence, identical in python and js and found in both only by
probing rather than by reading:

```
CPython/node   n1 a      n2 b       close fin
base           n1 a fin  n2 a b fin close        <- finally at the FIRST next()
```

A generator's `finally` fired at the first `next()` — *before* the real toolchain
runs it at all — and again at every step, and never at the close. Both items
claimed "ordering is right in every engine; only repetition differs". **That was
false in both.** The fix is to let the suspension marker through untouched
(`isGenSuspend` / `jsExcTry`) and to close deliberately, by putting an **exit
record on the resume value** of the yield the generator is parked at — the same
"the record travels on the resume value" pattern that makes throw-forwarding
per-coroutine.

Two things follow that are worth knowing before touching this area:

- **`interp-core.js`'s `excTry` is shared with six languages.** Overriding it by
  name is §7.12's include trap; js/ts took a language-LOCAL copy instead.
- **`.throw()` on a plain generator is blocked in every language**, and the
  blocker is one table: the floor's tag-15 member table offers `next` (host id
  60) and `return` (61) and nothing else (`runtime.c:3111/3116`), and
  `*jsGenerator` in `abnf/jsrt.go` is shared by every language that exposes a
  generator. The interpreter machinery exists in python and js; shipping it there
  alone would be a new halves divergence, which is why it is not shipped.

## 7.15 The value model already carries the type — you rarely need a type table

Swift's write-site adoption looked like it needed a per-variable declared-type
table in both engines. It did not: floatness (`{__flo}` / `__f32`) and integer
width (`{__sz}`) live **on the value**, and the language is statically typed, so a
write can adopt *the type of what the slot already holds*. One extern
(`js_swadoptlike`) in three engines closed every write site the item listed.
csharp reached the same conclusion from the other direction — its unqualified-write
branch has to identify the member and its declaring class anyway, so adoption came
free inside it. **Ask what the existing value already knows before designing a
side table.**

## 7.13 Who imports what

```
runtime.metajs (804)      <- csharp go dart java php kotlin ruby python swift metajs-rt
runtime-decimal (188)     <- python ruby swift
runtime-bignum  (263)     <- python ruby
runtime-jvm     (159)     <- csharp java
runtime-dartswift (50)    <- dart swift
regex.js       (~1,265)   <- js python ruby kotlin
lua-rt.metajs             <- imports NOTHING
js-rt.metajs              <- imports regex.js ONLY
```

`runtime.metajs` requires seven `rtk*` hooks from each importer (`rtkStr`
`rtkNum` `rtkIsChar` `rtkIsArr` `rtkCall` `rtkIsDict` `rtkDictFind`), plus
`rtkFloDigits` for `runtime-decimal`. **That hook contract, not size, is what
makes a first import expensive** — and it is the real reason the `luChar` and
`jvCtorDescOf` merges are declined.

**A `regex.js` change costs an edit in more than one place**: `abnf/jsrtregex.go`
is a line-by-line port of it (the Go twin for the compiler grammars) and must be
hand-synchronised, while the four importers and the interpreters pick the change
up automatically. The `rxGet` cache key keeps a `$` prefix on purpose — it is what
keeps a pattern spelling `"toString"` out of the intern table.

---

# 8. Where the old plan documents went

Until 2026-08-05 there were four more documents here: `runtime-rework-plan.md`
(the original C-floor/layer-2 plan), `runtime-next-plan.md` (10,095 lines: the GC,
the floor primitives, the sixteen language migrations), `runtime-merge-plan.md`
(five passes of de-duplicating layer 2), and three `*-concept-consolidation.md`
files. They were **removed deliberately**: they were the *record* of finished
work, they had gone stale within two commits five separate times, and having five
overlapping sources meant nobody could tell which one was current.

Everything still actionable from them is in [todo.md](todo.md); the durable
mechanics are in chapter 7 above. Everything else is in git history, in full:

```bash
git log --oneline --diff-filter=D -- docs/          # the commit that removed them
git show 8316a41:docs/runtime-next-plan.md          # read one, unchanged
git show 8316a41:docs/runtime-merge-plan.md
git show 8316a41:docs/runtime-rework-plan.md
```

**Code comments still cite them by name** — 158 references across ~70 files, of
the form *"per docs/runtime-next-plan.md part 3"*. Those are **provenance**, and
they were left as written rather than repointed at this file: they name a specific
part of a specific document, and rewriting them to point here would turn an
accurate historical citation into a false one. Use the `git show` above.

`docs/abnf-dialect-gotchas.md` is NOT one of the removed documents. It is current,
it is the reference for the ABNF tag dialect, and chapter 4 assumes you have it.
