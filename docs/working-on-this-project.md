# Working on this project: what I wish I had known on day one

Written 2026-08-04 at `ee51718`, after taking all sixteen languages from
"emits IR nothing can link" to "self-contained native binary", building the C floor
and the MetaJS layer 2, adding a garbage collector, and merging the layer-2
duplication twice (the second time correctly).

The three plan documents next to this one are the *record* of that work:
[runtime-rework-plan.md](runtime-rework-plan.md) (what was built),
[runtime-next-plan.md](runtime-next-plan.md) (GC, floor primitives, the sixteen
migrations, and the consolidated open list), [runtime-merge-plan.md](runtime-merge-plan.md)
(the shared MetaJS layer). **This document is the manual.** Read it first.

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
(`a3baeee`); see "Why the Go runtime stays" in `runtime-next-plan.md` Part 4.

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
./mec languages/python-to-llvm-ir.abnf  prog.py -q         # -q: program output only
./mec languages/python-to-llvm-ir.abnf  prog.py -q -exe a.out && ./a.out   # NATIVE
```

Flags you will actually use (full list in `main.go`'s header):

| flag | what it does |
|---|---|
| `-q` / `-qq` | program output only / errors only. **A compiler grammar prints its module first**, so without `-q` you get thousands of lines of IR before the program's output. |
| `-exe PATH` | link a native executable. Refuses loudly if the grammar has no `c.exePath`. |
| `-rt FILE` | link an extra `.c`/`.ll`/`.o`/`.a`. Declaring a runtime also makes an unresolved symbol a **hard error** instead of a zero-returning stub. |
| `-frozen` | run the tag scripts on the frozen MetaJS bootstrap instead of goja. **The matrix runs everything both ways and demands byte-identical output.** |
| `-i DIR` | add an include root for imports. |
| `-warn-imports` / `-warn-unsupported` | warn instead of aborting — how you get a call graph out of a partially-understood language. |
| `-max-steps N` | the IR interpreter's endless-loop brake (default 1e8). A big benchmark hits it. |
| `-freeze F` | regenerate the frozen bootstrap snapshot. Needed after **any** change to `metajs-to-llvm-ir.abnf` or `lib/compile-core.js`. |

**Reading a compiler grammar's output**: the module is a *prefix* of stdout, and the
program's own output follows. To separate them, scan forward tracking brace depth —
do **not** look for the last line that looks like IR, because `var_dump` prints a
bare `}` and PHP will fool you. `tests/clang-check.sh`'s `module_only()` is the
reference implementation.

---

# 3. Testing

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
tests/gen-*.sh --check       # fifteen generators: does the checked-in .ll still
                             # reproduce from its source? MANDATORY after any
                             # runtime.c / *-rt.metajs change.
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

**A crashing binary looks fast.** `/usr/bin/time` reports happily on a process that
died. I quoted bash at "0.44 s / 5.75 MB" as an argument against an architecture
change; it was segfaulting on every run and printing nothing. **Check the exit code
and the expected output**, not just the timing.

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

# 7. What is still open

Verified at `ee51718`. The authoritative per-item write-ups are in
`runtime-next-plan.md`'s "WHAT IS STILL OPEN" and `runtime-merge-plan.md`.

**Defects, with oracles available:**

1. **ruby-rt's `%g` is unimplemented.** Pre-existing; `ruby` 2.6.10 settles it.
2. **Python's `&` is int32-signed.** `-1 & 0xffffffff` gives `-1`; CPython says
   `4294967295`. Both halves agree, so `--cross` is blind. `python3` settles it.

**Merge residue:**

3. ~~14 shape groups, ~131 recoverable lines.~~ **Down to 8 groups / ~84
   recoverable** (`2d3e6f5`), and seven of the eight need a file another agent held.
   Dart's `dtJsCompare` is genuinely blocked (a third coercion on a path only invalid
   Dart reaches, no toolchain). `luChar`/`js_char` cannot share a name —
   `runtime.metajs` already exports `js_char` for a *different contract* — and both
   sites now cross-reference each other.
4. **kotlin carries the whole of `regex.js` verbatim** — 51 of 57 functions
   byte-identical, 1,217 lines, zero divergences. ~~+18-23%~~ ~~+10.7%~~ — and
   ~~the tax tracks top-level DECLARATIONS~~ **THE TAX IS GONE.** `scope_find`
   (`runtime.c`) has a hash index since 2026-08-04, and the declaration-index
   curve that governed every merge decision in this project is now flat:

   ```
   regex block at HEAD placement      before 30.970 G      after 18.023 G
   relocated to declaration index 0          33.399 G  +7.8%      17.923 G  -0.6%
   import runtime.metajs APPENDED            44.131 G +42.5%      18.130 G  +0.6%
   ```

   **The rule is: there is no rule. A shared layer-2 body costs a language that
   cannot call it 0%, at either end of the module.** Size it however you like.
   Two consequences: the `runtime-decimal` / `runtime-bignum` / `runtime-jvm` /
   `runtime-dartswift` splits exist for a reason that no longer holds, and every
   small pair the shape pass declined on this tax is worth re-opening. The
   `runtime-regex.metajs` split is a live question again for the first time
   (~1.1% for the six bodies kotlin omits, and nothing for placement).

   **Note what `import` APPENDING measured before the index landed: +42.5%, a
   LOSS.** What a declaration's index costs is not how many declarations precede
   a body — it is where the HOT ones sit, and `runtime.metajs` holds `js_jadd`.

   **Measure with `/usr/bin/time -l` instructions retired, not wall clock.** With
   agents running, a 40k-iteration loop varies +-8% between runs of the same binary;
   instructions reproduce to 0.02%.

**Recorded, larger, and deliberately not started:**

5. **A real tuple type for Python.** `set_member` rejects a named key on tag-5, so a
   tuple must become a box — the float box's ~640 lines again. 10/10 probe rows wrong
   *identically* in both halves, so it is a CPython gap, not a divergence.
6. **A catchable `NameError` under the compiler.** The site is `js_scope_get`, shared
   by all sixteen languages and pinned by clang-check and the SHOULD-ABORT rows.
7. **Part B has no landed move.** The floor→layer-2 upcall costs **120 ns** and is
   affordable; MetaJS **data-structure** work costs **3.2 µs** and is not. Every
   remaining candidate (`fmt_*`, `js_num_str`, the `dec_*` family) is reached by
   every print. Each needs its own benchmark before being written.

**Known differences from real toolchains, measured and left:** struct alignment is
packed (`{char;short;int;long}` is 15 bytes, `cc` says 16); `_Bool b = 5` holds 5;
identical string literals are not merged (unspecified in C); lua's `-0.0` unary minus
in the compiler half; a tuple prints as a list; ~9 latent bash ERE defects carried
verbatim from the hand-emitted engine.

---

# 8. What would make this project better

Ordered by how much time each would have saved me.

**1. A layer-2 test gate that runs by default.** `tests/native-full.sh` exists now,
but it is not part of `./test.sh`. Until it is, the default suite is structurally
blind to every `*-rt.metajs` change, and a newcomer will not know that. Fold it in as
a fourth group, or make `./test.sh` refuse to report green without it.

**2. `./test.sh --probe`.** The differential probe found nearly every real defect and
is re-invented by hand each time. A group that, per language, generates operand
matrices from a small spec, runs them under `llvm.Run` and native and the real
toolchain, and diffs — would turn the most productive technique in the project into
a command.

**3. A shape-scan lint.** The duplication that mattered was invisible to every
name-based view for months. `tools/shape-scan` (normalise identifiers to positional
tokens, group by body shape, report cross-language groups) is ~40 lines and would
have caught python and ruby writing the same decimal bignum the day it happened.
Wire it into CI with a ratchet on the group count.

**4. Make the frozen-snapshot rule mechanical.** "Did you `-freeze` after touching
the emitter?" is a question a script should ask. A pre-commit check that regenerates
and diffs would remove a whole class of near-misses.

**5. `-O2` was missing from `buildExecutable` for the entire project's life.** Adding
it made every native binary 2.2× faster. **Look for more of those**: the build path
had never been profiled, and the second-biggest win (the literal cache thrashing) was
the same shape — nobody had measured, so nobody knew.

**6. A benchmark suite with recorded baselines.** `tests/bench-alloc.sh` and friends
exist but the numbers live in prose in the plan documents. Machine-readable
baselines, checked in, would make "is this a regression?" a command instead of an
archaeology exercise.

**7. Make MetaJS's dialect gaps loud.** No exponent literal, no `toPrecision`, typed
locals, ASI differences: each is discovered by a confusing failure. A `-verify` pass
over `lib/*.metajs` that names them would pay for itself immediately.

**0. ~~`scope_find` is the single hottest line in the runtime~~ — DONE, 2026-08-04.**
It has a hash index now (an open-addressed `index + 1` table in a fourth part of the
scope's single buffer block, built at 32 entries and up, keyed on a content hash
memoised in the string cell's unused field `f`). Nine languages run **20–42% fewer
instructions**: python −42.0, kotlin −40.9, ruby −39.4, swift −33.8, php −32.4,
go −30.1, java −29.7, dart −28.7, typescript −28.5, js −26.5, csharp −19.5. `c` is
the control at −0.1%.

**lua is +4.1% and metajs +0.4%, and that is not the hashing.** Their module scopes
are too small to be indexed at all; the cost is that a second arm makes `scope_find`
too big for clang to inline into `scope_get`/`scope_put`/`js_tdecl`. A branch that
can never fire already costs lua +2.8%. Two ways of buying it back were measured and
are worse (out-of-line arm +4.7%, name-in-slot +6.1%); both are recorded at the site.

The *other* half of this item — **making `import` APPEND** — was executed and is a
**LOSS of +42.5%**, reverted, and written up at the `ImportStmt` rule in
`metajs-to-llvm-ir.abnf` so it is not attempted a third time.

**What is now the hottest line, nobody has looked.** `sample` on the lua and kotlin
native binaries is the tool; `ar_block`, `js_str_mem` and `obj_find` are the names
that came up while this was being measured.

**8. Per-language coverage of the floor.** The instrumentation that found 24
unreached floor bodies was written once and thrown away. Keeping it as
`tools/floor-coverage` would make "is this body dead or untested?" answerable at any
time — it is the question that governs Part B.

**9. A `--why` flag for the emitters.** Understanding which extern an emitter chooses
for an operator meant reading thousands of lines of grammar repeatedly. A flag that
prints the extern chosen per AST node would make layer-2 work dramatically faster.

**10. Faster iteration.** A full sweep is ~5 minutes; `kotlin-rt.ll` alone is 120k
lines to regenerate. Incremental `.ll` regeneration keyed on source hash, and a
`--changed-only` mode for the matrix, would shorten the loop that every one of these
tasks runs dozens of times.
