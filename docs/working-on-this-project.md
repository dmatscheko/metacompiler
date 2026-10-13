# Working on this project: what I wish I had known on day one

This is the only manual. The one other current reference is
[abnf-dialect-gotchas.md](abnf-dialect-gotchas.md), for the ABNF tag dialect, which
chapter 4 assumes you have. What is still to do is in [todo.md](todo.md), not here.

How to read this: **chapters 1-3** are what you need to build and test anything.
**Chapter 4** is the traps, and it is the highest value-per-line in the file.
**Chapter 7** is the engine mechanics you would otherwise re-derive at some cost.

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
libc                     malloc, putchar, write, exit, setjmp/longjmp, valloc
languages/lib/runtime.c  the C FLOOR: ~35 irreducible primitives + the GC +
                         coroutines. Compiled by OUR c-to-llvm-ir.abnf.
languages/lib/*.metajs   LAYER 2: each language's semantics in MetaJS, compiled by
                         OUR metajs-to-llvm-ir.abnf. Shared parts in
                         runtime.metajs
languages/<lang>-to-llvm-ir.abnf   the emitter
```

Every layer is compiled by this repository, except `languages/lib/coro-switch.S`:
two assembly routines (arm64 and x86-64) that switch a generator's stack, which
the C subset cannot spell. `nm -u` on any native binary shows libc and nothing
else.

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

**Install the frozen-snapshot hook once, and stop thinking about the rule**:
`ln -s ../../tools/pre-commit .git/hooks/pre-commit` refuses a commit that stages
`metajs-to-llvm-ir.abnf` or `lib/compile-core.js` without the regenerated
snapshot. `git commit --no-verify` bypasses it; `tests/gates.sh --freeze` remains
the fixed-point check.

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

**An extern is only callable FROM AN EMITTED MODULE, never from layer 2.** A
`lib/*-rt.metajs` file calling `js_keys(o)` links, passes clang, and then dies
natively with *ReferenceError: js_keys is not defined*. The same function is
reachable from layer 2 under its HOST-BUILTIN name (`keysOf`, host id 61). Nothing
in the matrix, `--full` or `--cross` can see this - only a native `-exe` probe can.

**`up.in` arrives with whitespace ALREADY STRIPPED.** `<T extends num>` reads as
`<Textendsnum>`, so a scan cannot separate a type parameter's name from its bound
and a structured parse is the only option. Cost one agent a wrong first
implementation of dart's type parameters. `c.srcSlice(from, to)` reads the source as
written back (docs/abnf-dialect-gotchas.md, "`up.in` hands over the matched TOKENS").

**`hasOwn` exists in `interp-core.js` and NOT in `compile-core.js`.** In an emitter,
test `map["$k"] == undefined` instead. The two cores are not the same surface, and
the difference is only discovered by an emitter that works in one half.

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

**goja is a patched copy in `third_party/goja`, because its `Number→String` was
wrong.** Upstream goja's shortest-digits fallback (the Rhino-derived `ftoa`
behind Grisu) is not always shortest or even round-trip - about 1 double in
11,000, and at some subnormals a non-digit character (`F.250738585070127e-309`).
Every grammar that takes a double's digits from `"" + n` inherits that as a
goja-vs-`-frozen` divergence, so the one patch (`ftoa/ftostr.go`: shortest
digits from `strconv`) fixes them all, and
`TestNumberToStringAgreesAcrossHosts` pins goja against the frozen host's
`jsNumString` on ~430,000 boundary doubles. Do not "upgrade" goja through
`go get`: the `replace` in `go.mod` is what keeps the patch.

**A RANDOM float probe has near-zero discriminating power; targeted values have
all of it.** A 1,450-line random probe agreed with the oracle at base while 85
hand-picked values gave 170 differing lines and an outright abort. Sample the
boundaries — 17-significant-digit values, subnormals, `MIN_VALUE`, the exponent
thresholds — not the space.

**A value pushed onto the ABNF tag stack is DEEP-COPIED, and the symptom is that
there is no symptom.** Handing a walk-time record up to an enclosing declaration
as `push({nested: pop(), nrec: lastDeclRec})` delivers a *clone*: re-keying moved
the clone while the declaration's own emit closure still read the original, and
the fix simply, silently did not work — while a subset of probes passed for an
unrelated reason. It took an instrumented emit to see it. **Pass an INDEX into a
script-side array, not the record.** (`swift-to-llvm-ir.abnf`, `declRecs`.)

**`rxExtraExterns` fills ONE extern map for EVERY language.** Wrapping a *shared*
extern in a language's own twin file therefore leaks across languages: overriding
`m["js_mcall"]` in `jsrtjava.go` made a **C# `Substring`** raise java's
*"Range [10, 6) out of bounds for length 6"*. Observed, not theorised. The correct
pattern is a new language-specific name (`js_jvsub`, `js_csmcall`, `js_ktsmcall`).

**`host_call` is a `switch`, and it must stay one.** As a linear `if`-chain,
twenty-three arms added in the middle cost java **+7.08%** — every
`sint`/`flo`/`scope` call paid the extra compares. Our C compiler lowers a
`switch` to a compare chain that clang folds into a jump table (an indirect `br` in
`otool -tV` of a native binary), so a new host id costs nothing
wherever it goes. Every case returns; a case that falls out would run the next one.

**Go's arm64 backend contracts `a*b+c` into FMA inside `math`, and our C does
not.** That is the whole of the residual one-ulp difference between `llvm.Run` and
a native binary on transcendentals — 21 of 6,734 probe rows. Go does *not*
contract on amd64, so the Go twin's digits are already architecture-dependent and
no spelling matches both. Note also that **assigning the product to a local does
not stop Go fusing it; only an explicit `float64()` conversion does.**

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
an infinity as `Math.pow(2, 1024)`, as `pyMInf` does - under goja `1e307 * 10` is
the finite 1e308.

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

**`tar x` OVERWRITES, so re-extracting an archive over a verification tree
silently reverts the files you copied in.** The symptom is that a change you are
verifying "does not take effect", or that two gate runs disagree for no reason;
an agent lost two full gate runs to it and reasonably but wrongly concluded that
`test.sh` was reverting files. It is not — **no script in `tests/` runs `git` at
all**. Either extract into a fresh directory every time (`rm -rf` first), or
re-copy your files after any re-extraction, and `md5` the files you care about
before and after a long run if you are unsure.

**A LANGUAGE'S GO TWIN IS NOT CONFINED TO `abnf/jsrt<lang>.go`.** Isolating one
language by copying its grammars, its `lib/<lang>-rt.metajs` and
`abnf/jsrt<lang>.go` is *not enough*: ruby's `Rational`/`Complex`/`Method` work
put 429 lines into the shared `abnf/jsrt.go`. Verifying without it showed a large
fake halves divergence — the interpreter half had the fix and the `llvm.Run` half
did not, because the twin it resolves `js_*` against was the base one. Copy the
shared twin too, and remember that doing so drags in every *other* language's
in-flight edits to that file, so a single-language isolation is only clean when no
other agent owns `jsrt.go` that round.

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

**And a real `tsc`, which nobody knew was here until 2026-08-10.** VS Code ships
TypeScript 6.0.3 at
`/Applications/Visual Studio Code.app/Contents/Resources/app/extensions/node_modules/typescript/lib/typescript.js`.
Drive it from node - `ts.createSourceFile(name, src, ts.ScriptTarget.Latest, true)`
and read `sf.parseDiagnostics` - and you get real TS error codes. It immediately
paid for itself: todo item 2.10 asserted that real TS accepts `(super as any)[k]`
and that our grammars were wrong to reject it. tsc rejects it too, with
**TS1034 "'super' must be followed by an argument list or member access"**, so the
item's fix would have taught two grammars to accept invalid TypeScript.
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
- **Findings go into [todo.md](todo.md) or this manual**, committed with the code -
  not into code comments. Knowledge outlives the context that produced it.
- **Verify in isolation.** `git archive <base> | tar x -C /tmp/x`, copy in only that
  agent's files, build inside, run the suite. That is how you tell one agent's
  regression from another's in-flight edit.
- **Disjoint file sets, stated explicitly**, and a named owner for every shared file
  (`runtime.c`, `abnf/jsrt.go`, `metajs-to-llvm-ir.abnf`, `.vscode/launch.json`).
- **Wait on the subagent, never poll a file's mtime.** One agent burned hours in
  `ls -l` loops waiting for a child that had already finished and reported elsewhere.
- **The to-do list goes stale within two commits.** An item's verdict is only as
  good as the commit it names — **re-grep the implementation before acting on any
  line of it.**
- **Code comments explain the code, not its history.** No round numbers, commit
  hashes, todo item numbers or "used to" narratives in code: todo.md is renumbered
  as items close, so a `todo.md N.M` citation in code soon points at the wrong item.

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
them. `js_ctl_kind` is the second instance — it answers a raw kind (0 not a
signal, 1 return, 2 break, 3 continue, 4 + n a goto to compile-core's label n), so
the test must be `IPredEQ …, 1`, because `truthy()` silently accepts every other
signal too. Ruby goes furthest and
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
kotlin's chain walks stayed — though see §7.27, the floor has one now; the scope
API `scopeNew`/`scopeParent`/`scopeGet`/`scopeHas`/`scopeDecl`, host ids 64..68,
is a MetaJS HOST BUILTIN reachable only from a `lib/*-rt.metajs` file, and emitters
have shipped against it through layer 2 since ruby's `018c693` and kotlin's
`a6df3a8`), and
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
`d_to_long`, `d_trunc`, `d_floor`.

**But do NOT read that as "hex-float formatting needs the floor".** Ruby's `%a`
was written entirely in MetaJS in 2026-08-10 with no bit access at all: scaling a
double by a power of two is EXACT provided you never overflow and never enter the
subnormals, and both directions can be driven so they do not (divide only while
`x >= p`, multiply only while `x*p < 2`). `d_frexp` is not reachable from MetaJS
anyway - it takes an out-pointer. 6,624 probe rows, exact to `5e-324`, and the same
arithmetic in all three engines so they agree by construction.

Separately, **strike `d_frexp`/`d_ldexp`/
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
`--cross` was silent every time.** Also: **whether `printf` width and precision count BYTES or CHARACTERS is
PER-LANGUAGE, and this file asserted the wrong one for Ruby until 2026-08-10.**
It said "count BYTES, not characters — `"%10s" % "café"` gets five spaces;
`byteLen` is the floor's answer". Measured against `/usr/bin/ruby`: **MRI pads to
six spaces** (result `length` 10, `bytesize` 11), because Ruby counts CHARACTERS.
All three engines had been written to `abnf/jsrt.go`'s `len(body)`, and
`jsrt.go` had never been checked against MRI — so the wrong rule was in the code,
in the twin, and in this manual at once. Ruby also counts an astral character as
one, i.e. code points and not UTF-16 units. `byteLen` remains the right answer
where the language really is byte-counted (C, and PHP's `printf`); **check the
oracle per language rather than inheriting this sentence.**

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
- **Objects that live for the whole run are allocated in a PERMANENT region**
  (`PERM_LO` in `runtime.c`): what `boot()` seeds, a layer-2 library's top level
  (`jsrtlib_boot` brackets it with `js_gc_perm(1)`/`js_gc_perm(0)`), the
  string-literal and small-integer caches. They are never marked or swept; the
  region is a root, traced by tag and scanned conservatively each collection,
  because those objects stay mutable. Marked like the rest of the heap, the whole
  runtime library is re-traced by every collection, and a language pays for each
  layer-2 function it never calls (csharp: ~600 instructions per function per
  collection, against ~200 for a permanent one).
- **Any new emitted module-level global holding a handle must be pinned with
  `js_gc_pin`** — the emitted module has globals the floor cannot see. `js_gc_pin`
  is the identity in both halves.
- **`MEC_GC` is the only reason those defects were findable**: `off`, default
  (collect when bytes-since-last > live set, floor 4 MB, the first collection
  at 1 MB), `stress`, `poison` (retire and fill every swept block, so a missed
  root is a wrong answer rather than lucky bytes), plus `MEC_GC_STATS=1`. `poison` is deliberately **not** also
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
- **Python**: a tuple is an array carrying kind 16 (`arrKind` / `arrKindSet`,
  `jsArray.kind`, field d of the floor's array cell), which every engine already
  kept; bytes and bytearray carry 17 and 18 the same way, and their str-like
  methods run the str ones over a transcoding that keeps them ASCII-only
  (`pyBytesToS`, `abnf/jsrtpybytes.go`). `hash()` of a str is a fixed polynomial over its code units, where
  CPython's is SipHash under a per-process random key, so no value of it is
  reproducible anyway; only `hash("") == 0` and equal-strings-hash-equal hold.

**Recorded at their sites in the same way, moved here from todo.md 2.11 when that
list was compressed.** ~~**C#** `e.ToString()` prints `[object Object]`~~ -
**CLOSED 2026-08-10.** It answers `System.Exception: m` now, and for an exception
that has not been thrown .NET's `_stackTraceString` is null and `FullName: Message`
is the ENTIRE answer, so that case is byte-exact. What is still absent is the
`at Program.Main()` frames (no call-site line table in any engine) and inner
exceptions (no inner slot in the value model); **both omissions can only shorten
the answer, neither makes the produced part disagree.** **Ruby**
`("a".."e").size` is 5 where MRI says nil — the emitter lowers every bare `.size`
to `js_pylen`, which is also for-in's trip count (implemented, measured, reverted).
**Python** `repr(math)` names a machine-specific path (`.so` or `.py`), where
ours is `<module 'math' (built-in)>`; a name a runtime module does not provide
(`from typing import List`) is left unbound rather than an ImportError, because
the module's surface is a subset of CPython's. **Swift** `x is Inner` cannot tell two
same-named nested types apart.
**Dart** `catch (e, s)` drops the stack-trace binder — real frames are not
reproducible here and `StackTrace.empty` would be a divergence of a different
shape; an uncaught `__dexc` in the compiler half prints `bad` rather than
`FormatException: bad` (the handler is `runJSModule` in the shared `abnf/jsrt.go`).
**Bash** the compiler's attribute scope is LEXICAL where bash's is dynamic;
`readonly` is not enforced; `%e %f %g %a` need a double in `bash-rt.c`, which has
none.

Per-language divergences from real toolchains, recorded and permanent: Go byte-
slicing a rune in half cannot be represented at all (the floor's strings are
UTF-16); Go counts `len([]byte(s))` in RUNES. An
~~astral character prints as two U+FFFD in swift~~ - **NO LONGER TRUE for string
literals, 2026-08-10**: `swUnescape` was copying a literal one code unit at a time
and handing the host a LONE SURROGATE, which UTF-8 cannot encode. All four legs
print the character now. JS: `(0.1).toString(3)` emits 52 fraction digits, the `d` flag builds no `m.indices`.
Python: integral floats render as int, `&|^` are ToInt32, `inf`/`nan` spell
`Infinity`/`NaN`, the format mini-language is a subset.
A Python str is indexed, sliced, iterated and searched by CODE POINT over the
floor's UTF-16 strings: python emits `js_pyidx` for a subscript because the
shared `js_pyget` must stay UTF-16 for dart and csharp, and layer 2 converts
positions with an ASCII fast path (`byteLen(s) == s.length`), so a non-ASCII
string pays a linear scan per position. `abnf/jsrtphp.go` misreads `"1e400"` as 1 where both our halves answer
INF (the PHP spec agrees with us) — sole cause of 431 hunks in a 40,948-line
probe, and permanent, since retiring the Go twin was declined.
A PHP byte that is not part of a character (chr(200), `"\xC8"`, one byte of
"é") is held as the lone surrogate U+DC00 + byte, so strlen, offsets, ord and
concatenation are exact, but `echo` prints it as U+FFFD where PHP writes the raw
byte: every engine's print turns a lone surrogate into U+FFFD, and a raw-byte
write would be a change to the shared print path.
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
is ~100 lines and every `throw` in all sixteen languages goes through it; the
generators' stack switch is assembly (`lib/coro-switch.S`), which an IR
interpreter has no way to run at all.

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
- **`.throw()` on a plain generator was blocked in every language by one table**,
  and the fix is the same pattern: the floor's tag-15 member table now offers a
  third member (`throw`, host id 62 -> `gen_throw`), the flag lives in the
  generator's own CONTROL BLOCK (`cb[14]`, beside `cb[13]`'s close) and the value
  in the cell's `f` slot, so it is per-coroutine by construction. The Go twin's
  half is `throwInto` plus a `*genThrowSignal` on the resume channel. At a
  `yield*` the value is FORWARDED to the delegate's `throw()` in all four engines,
  which in the emitters meant giving the yield* barrier a CATCH that turns the
  raised value back into the `{__athrow, v}` record the loop head already
  dispatched on for `ag.throw()` - the async machinery was 90% of the work
  already. **`tests/coro-poc/coro-floor.py` is the gate that catches this**, and
  not only through an undefined symbol: its `--check` needles are matched with
  `count() != 1`, so a NEW occurrence of one reads as the piece having
  DISAPPEARED. `gen_throw`'s `js_throw(ff(g))` at two tabs made `gen_resume`'s at
  three stop being unique.
- **A `yield` inside a `catch`/`except` ARM ran the enclosing `finally` at the
  suspension**, in the js, typescript and python interpreter halves, and it is
  reachable with no `.throw()` at all: `try { throw 1 } catch (e) { yield 99 }
  finally { … }`. `jsExcTry`/`makeTry` only tested for the suspension signal in
  the catch clause of the *try* body, so one thrown from inside the *handler* was
  invisible to it. This is the same defect §7.14 fixed one arm further out, it
  survived that round, and it was a live halves divergence: 2 of 4 legs.

## 7.15 The value model already carries the type — you rarely need a type table

Swift's write-site adoption looked like it needed a per-variable declared-type
table in both engines. Mostly it did not: floatness (`{__flo}` / `__f32`) and
integer width (`{__sz}`) live **on the value**, so a write to a slot whose declared
or inferred type is **numeric** can adopt *the type of what the slot already
holds* - one extern (`js_swadoptlike`) in three engines. The rule is only that
narrow: an **existential** slot (`Any`, a protocol, `any P`, an Optional of one)
holds whatever it was given, so `var a: Any = 1.5; a = 2` holds the Int 2. The
write site therefore still asks the declaration (`swKeepMode`: the local's
annotation, else the property's, by name and per type when names collide) and
skips adoption for an existential one. csharp reached the same conclusion from
the other direction — its unqualified-write branch has to identify the member and
its declaring class anyway, so adoption came free inside it. **Ask what the
existing value already knows before designing a side table - and what the
declaration allows it to become.**

## 7.16 What a layer-2 call costs, and what the MetaJS compiler removes from it

Every layer 2 is MetaJS compiled by `metajs-to-llvm-ir.abnf`, so its cost per
call is that compiler's. Three things keep it down, and all three are exact rather
than heuristic (the start script's comments at `wFrames`, `gCachePtr` and `dRec`
have the rules):

- **A parameter or `var` that is never assigned and never read from a nested
  function is an SSA value**, not a scope slot, and a function that binds nothing
  else gets no scope of its own. Assigning a parameter, capturing it in a closure
  or a `try` body, or naming `arguments` puts it back in the scope.
- **A name the function does not declare is read through `js_scope_cget`**, a
  per-name cache in the module that the floor fills only for PERMANENT scopes (a
  layer 2's module scope and the root), so a helper or host builtin costs a few
  loads instead of a hashed walk.
- **A call of a top-level function** compares the looked-up closure with the one
  its declaration published and, when it is the same, calls its body directly
  with the arguments in registers. A plain call of up to three arguments
  otherwise goes through `js_call0..3`, which hands a host builtin its arguments
  in a C-stack array.

They took java's `s = s + i % 7` from ~150,000 to ~34,000 instructions an
iteration. Under `-trace`, `-cfgraph`, `-callgraph` and `-why` all three are off:
those read variable and call NAMES off the module.

## 7.13 Who imports what

```
runtime.metajs            <- csharp go dart java php kotlin ruby python swift metajs-rt lua
                             (lua for the hook-free exact decimal only)
regex.js                  <- js python ruby kotlin java dart
int64.js                  <- interp-core.js, bash interpreter (the exact {h, l} 64-bit
                             pair; bash's arithmetic is intmax_t)
lua-rt.metajs             <- imports lua-strlib.js and runtime.metajs ONLY
lua-strlib.js             <- lua-rt.metajs, lua interpreter (Lua patterns and
                             string.format; Go twins: abnf/jsrtluastr.go,
                             abnf/jsrtluafmt.go)
js-rt.metajs              <- imports regex.js, json.js and js-num.js ONLY
json.js                   <- js-rt.metajs, js and typescript interpreters
                             (Go twin: abnf/jsrtjson.go)
js-num.js                 <- js-rt.metajs, js and typescript interpreters
                             (StringToNumber and parseFloat; Go twin:
                             jsStrToNum / jsParseFloatJS in abnf/jsrtjsbig.go)
go-fmt.js                 <- go-rt.metajs, go interpreter (fmt.Sprintf; Go twin:
                             gopFormat in abnf/jsrtgolang.go)
fdlibm.js                 <- java-rt.metajs, java interpreter, kotlin-rt.metajs,
                             kotlin interpreter (Go twin: abnf/jsrtfdlibm.go)
kotlin-math.js            <- kotlin-rt.metajs, kotlin interpreter (kotlin.math
                             and java.lang.Math on doubles; Go twin:
                             abnf/jsrtktmath.go)
python-num.js             <- python-rt.metajs, python interpreter (int's and
                             float's own methods at the bit and double level;
                             Go twin: abnf/jsrtpynum.go)
python-suggest.js         <- python-rt.metajs, python interpreter (the "Did you
                             mean" of an unexpected keyword argument; Go twin:
                             abnf/jsrtpysuggest.go)
python-dir.js             <- python-rt.metajs, python interpreter (the fixed
                             names dir() of a user class answers; Go twin:
                             pyDirObject in abnf/jsrt.go)
python-printable.js       <- python-rt.metajs, python interpreter (Unicode
                             printability for repr() and str.isprintable(), and
                             XID_Start/XID_Continue for str.isidentifier(); Go
                             twin: abnf/pyprintable.go)
java-chars.js             <- java-rt.metajs, java interpreter, csharp-rt.metajs,
                             csharp interpreter (Go twin: abnf/jsrtjavachars.go)
java-strings.js           <- java-rt.metajs, java interpreter (String's methods
                             and String.format; Go twin: abnf/jsrtjavastr.go)
csharp-strings.js         <- csharp-rt.metajs, csharp interpreter (System.String
                             and System.Char; Go twin: abnf/jsrtcsstr.go)
csharp-collections.js     <- csharp-rt.metajs, csharp interpreter (List<T> and
                             Dictionary<TKey,TValue>; Go twin: abnf/jsrtcscoll.go)
csharp-linq.js            <- csharp-rt.metajs, csharp interpreter (System.Linq's
                             Enumerable operators; Go twin: abnf/jsrtcslinq.go)
csharp-math.js            <- csharp-rt.metajs, csharp interpreter (System.Math's
                             static members; Go twin: abnf/jsrtcsmath.go)
utf16-strings.js          <- java-rt.metajs, csharp-rt.metajs, java and csharp
                             interpreters (case mapping and literal search over
                             UTF-16 units; Go twin: abnf/jsrtjavastr.go)
ruby-respond.js           <- ruby-rt.metajs, ruby interpreter (respond_to? on a
                             builtin; Go twin: rubyRespondNames in
                             abnf/jsrtruby.go, kept equal by a go test)
ruby-num.js               <- ruby-rt.metajs, ruby interpreter (String#to_f and
                             Float(String), MRI's rb_cstr_to_dbl; Go twin:
                             rubyStrToDbl in abnf/jsrt.go)
hex-float.js              <- python-rt.metajs, swift-rt.metajs, ruby-rt.metajs,
                             dart-rt.metajs and the python, swift, ruby, dart and
                             metajs interpreters (a hexadecimal float numeral,
                             correctly rounded; the Go twins use
                             strconv.ParseFloat)
php-strings.js            <- php-rt.metajs, both php grammars (substr, strpos and
                             the other byte-string natives, and the compiler's
                             literal check; Go twin: abnf/jsrtphpstr.go)
```

Grammar-level includes shared by one language's grammars only: `go-constfold.js`
(go), `java-decltypes.js` (both java grammars: declared types for assignment
conversion and static overload selection), `java-util.java` (both java grammars:
java.util's list types WRITTEN IN JAVA and compiled with a program that imports
them, so one implementation serves all three engines), `java-lang.java` (the
same for java.lang's StringBuilder, compiled when a program creates one),
`java-map.java` (the same for java.util's Map and HashMap, in java's own
iteration order), `java-stream.java` (the same for java.util.stream, compiled at
a java.util.stream import or a call of a method named stream),
`php-stdlib.php` (both php grammars: PHP's array and string library and SPL WRITTEN IN
PHP, compiled with a program that names one of its functions or classes),
`python-copy.py` and `python-dataclasses.py` (both python grammars: the copy and
dataclasses modules WRITTEN IN PYTHON, compiled with a program that imports
them),
`kotlin-decltypes.js` (both kotlin grammars: the static types an integer literal
adopts - type arguments, declared and inferred names, class properties - read at
walk time, and which `==` has an integer literal operand and so compares across
the integer types),
`go-strings.go` (both go grammars: the strings package WRITTEN IN GO, compiled with
a program that imports "strings"; goLibFile in `go-constfold.js` names it),
`csharp-overload.js` (both csharp grammars: the better function member among
applicable overloads), `csharp-default.js` (both csharp grammars: the target type
of the bare `default` literal), `swift-dotwant.js` (both swift grammars: the expected
type of an implicit member inside an array, ternary, closure, call argument,
== operand or collection-method argument),
`js-callee.js` (all four
js/ts grammars: V8's re-printing of a callee in "is not a function"). Each is ONE implementation of a feature, not a home for
helpers that happen to be duplicated (7.12 still holds for those).

`lib/js-builtins.js` is different in kind: it is JAVASCRIPT, not MetaJS - Set, Map,
Array.from, Array.of and Boolean written in the language and compiled INTO a program
by the two js grammars (`jsbLoad`, via `load()`) when the program names Set, Map,
Array or Boolean. One source compiled by the program's own grammar is what makes all
four legs agree.

There is one shared layer-2 module. A layer-2 call is a scope lookup by NAME, so a
body in `runtime.metajs` that a language never calls costs that language nothing at
run time (its module only carries the name as a string constant).

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

`docs/abnf-dialect-gotchas.md` is NOT one of the removed documents. It is current,
it is the reference for the ABNF tag dialect, and chapter 4 assumes you have it.
