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

Verified at `1e1330e`. The authoritative per-item write-ups are in
`runtime-next-plan.md`'s "WHAT IS STILL OPEN" and `runtime-merge-plan.md`.

**Defects, with oracles available:**

1. ~~**ruby-rt's `%g` is unimplemented.**~~ **CLOSED, and the item was wrong.**
   `%g` landed in `2d3e6f5`, one commit after this document's verification point,
   and a 1,393-row four-way probe found every non-`#` row already matching MRI.
   What *was* missing was the **`#` (alternate) flag**, in all three engines — and
   the two compiled halves failed it worse than the interpreter: they treated `#`
   as the *conversion*, emitted a literal `#`, left the argument unconsumed and
   **shifted every later directive**. Fixed in `1eb31e2`, along with `"%d" % -1.5`
   (the interpreter floored, MRI truncates).
2. ~~**Python's `&` is int32-signed.**~~ **CLOSED as a NULL RESULT — it was
   already fixed.** 50 probe rows over `& | ^ ~ << >>` across sign and word
   boundaries, *including this item's own `-1 & 0xffffffff` example*, agree with
   CPython 3.14.6 on all four legs at `e1307f3`. The item was stale, not open.
   What *was* real and is now fixed (`506000a`): integer **literals** past 2^53,
   where the boxing predicate called `parseFloat` — which rounds at exactly the
   boundary it exists to detect — and radix literals, which were never considered
   for boxing at all.

   **Both of these are the same lesson, and it is the most reliable one in this
   file: RE-GREP BEFORE ACTING ON ANY LINE OF A PLAN LIST.** Two of the two
   "defects with oracles available" listed here were not defects. The list has now
   gone stale within two commits five times.

**Merge residue:**

3. ~~14 shape groups~~ ~~8 groups / ~84 recoverable~~ — **DOWN TO 2 GROUPS / 9
   LINES** (`120ba0f`), and both survivors are deliberate declines, not residue.
   The fifth pass retired 205 lines once the `scope_find` hash index made the
   declaration tax zero and every tax-based decline re-openable: seven zero-knob
   groups into `runtime.metajs` (`rtCopyArr` had **seven** copies, two of them in
   kotlin 993 lines apart) and four into `regex.js`.

   **The finding worth carrying: the regex divergence was between the ENGINES,
   not among the copies.** `abnf/jsrtregex.go` already had *one* shared
   `rxCache`/`rxGet`/`rxObjRe`/`rxMatchOf` for all four languages while layer 2
   had four of each — so layer 2 was the wrong side of the comparison the whole
   time, and the shared bodies now carry the Go twin's own names so the two files
   line up when diffed.

   The two declines: java/kotlin `js_jband`, whose single difference sits on a
   path **neither emitter can produce**; and `luChar`/`js_char`, because `lua-rt`
   imports *nothing* and `runtime.metajs` would hand it definitions it does not
   want. Dart's `dtJsCompare` remains genuinely blocked (a third coercion on a
   path only invalid Dart reaches, and no `dart` toolchain here).
4. ~~**kotlin carries the whole of `regex.js` verbatim**~~ — **CLOSED,
   2026-08-04. `kotlin-rt.metajs` line 7275 is now `import "./regex.js"` and the
   1,489-line copy is deleted.** Declined at ~~+18-23%~~ then ~~+10.7%~~;
   re-measured after the `scope_find` hash index and shipped at a cost that is
   **indistinguishable from zero**, on a *correctness* argument, not a
   performance one:

   ```
                            old (verbatim)      new (import)     delta
   40,000 x s = s + i % 7   mean 17.989 G       mean 17.976 G    -0.07%  (n=20)
                            95% CI on the delta [-0.60%, +0.38%]
   1,000 x regexp ops       mean 15.052 G       mean 15.093 G    +0.28%  (n=14)
                            95% CI on the delta [-0.60%, +1.17%]
   println("hi")            med  27.076 M       med  27.048 M    -0.10%  (n=10)
   ```

   The cost is six bodies kotlin cannot reach (`rxMatchAt` `rxTest`
   `rxGroupCount` `rxNameAt` `rxReplace` `rxSplit`) against a scope that is now
   hash-indexed — six extra `js_tdecl` calls at module init. No
   `runtime-regex.metajs` was created: it would drop those six, which are worth
   nothing measurable, and re-introduce a second regex text to hand-synchronise.
   Full write-up, the probe, and the four-way `jxGetProg`/`pyERxGet`/`rbRxGet`/
   `k5Get` memo now worth retiring: `runtime-merge-plan.md`, "THE THIRD ASKING".

   **⚠ THE HARNESS DEFECT THIS FOUND, which invalidates a lot of prose in these
   plan documents.** A single-build A/B on a native binary **cannot resolve
   anything below about 2%.** Take one byte-identical `.ll`, one source, one
   tree, and vary nothing but the **length of the `-exe` output filename**:

   ```
   17.756 17.779 17.910 17.912 17.912 17.914 17.916 17.920 17.937 17.943
   18.017 18.018 18.021 18.023 18.029 18.029 18.031 18.035 18.071 18.600  G
   ```

   **A 4.8% spread, bimodal, from the name of the output file.** It is not the
   run and it is not codegen: six rebuilds of one source to one path give
   `md5`-identical binaries and 0.05% spread. The path string lands in the binary
   and moves the code layout — and so does the module's own content, identically.
   **Therefore holding the `-exe` path fixed across variants cancels NOTHING**
   (I recorded that as the fix; it is not). There is no pairing that cancels
   layout. The only sound estimator is a **mean over ≥15 layout draws with a
   confidence interval** — vary the output filename length, it is free. Two
   careful people got +0.04% and +0.65% on these same two sources before doing
   this; both were single draws.

   Also: `gen-*-rt-ll.sh` runs its own `go build`, so another agent's in-flight
   `abnf/*.go` edit breaks your measurement mid-sweep. Measure in a
   `git archive <base> | tar x` tree, per §4.

   The declaration-index curve that governed every merge decision in this project
   is flat, which is why the above reversed:

   ```
   regex block at HEAD placement      before 30.970 G      after 18.023 G
   relocated to declaration index 0          33.399 G  +7.8%      17.923 G  -0.6%
   import runtime.metajs APPENDED            44.131 G +42.5%      18.130 G  +0.6%
   ```

   **The rule is: there is no rule. A shared layer-2 body costs a language that
   cannot call it 0%, at either end of the module.** Size it however you like.
   Two consequences: the `runtime-decimal` / `runtime-bignum` / `runtime-jvm` /
   `runtime-dartswift` splits exist for a reason that no longer holds, and every
   small pair the shape pass declined on this tax is worth re-opening — the
   regex import above is the first of them to be re-opened, and it went the other
   way. **The remaining ones have not been.**

   **Note what `import` APPENDING measured before the index landed: +42.5%, a
   LOSS.** What a declaration's index costs is not how many declarations precede
   a body — it is where the HOT ones sit, and `runtime.metajs` holds `js_jadd`.

   **Measure with `/usr/bin/time -l` instructions retired, not wall clock.** With
   agents running, a 40k-iteration loop varies +-8% between runs of the same binary;
   instructions reproduce to 0.02% **for one fixed binary** — which is exactly why
   the layout lottery above went unnoticed for so long. Reproducibility per binary
   is not reproducibility per build, and the rows just above (`-0.6%`, `+0.6%`)
   are single builds and should be read as **0 +- 2%**.

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
8. **Python integer ARITHMETIC does not promote past 2^53** (the literals now do).
   `9007199254740992 + 1` is unchanged; 54 of 55 differing probe rows against
   CPython are the `*` column. The site is the plain `l+r`/`l-r`/`l*r` arms in all
   three engines — the hot path of every Python program, where a naive guard
   measured +9.5% — and `*` needs the exact product, which is the expensive half.
9. **Ruby's integer directives**, all three of which BOTH HALVES AGREE ON, so
   `--cross` is blind by construction: `%d`/`%x`/`%o`/`%b` of a float past 2^53;
   negatives under `%x`/`%o`/`%b`, which need MRI's infinite two's-complement
   `..fb` notation; and precision on an integer directive being ignored.
10. **`gates.sh` runs its seven gates serially at 315% CPU.** Running them
   concurrently was measured at **291 s → 150 s with all seven verdicts
   byte-identical** — but it was not shipped, correctly: the matrix entries in
   `.vscode/launch.json` write **fixed paths inside the repo**, so proving no two
   concurrent gates collide needs an audit of three more scripts. One green run is
   not proof, and a false green in `gates.sh` is the worst outcome available here.
   The conditions to discharge it are recorded.

**Known differences from real toolchains, measured and left:** struct alignment is
packed (`{char;short;int;long}` is 15 bytes, `cc` says 16); `_Bool b = 5` holds 5;
identical string literals are not merged (unspecified in C); lua's `-0.0` unary minus
in the compiler half; a tuple prints as a list; ~9 latent bash ERE defects carried
verbatim from the hand-emitted engine.

---

# 8. What would make this project better

Ordered by how much time each would have saved me.

**1. ~~A layer-2 test gate that runs by default.~~ — DONE, 2026-08-04.**
`tests/gates.sh` runs all seven gates in one command and cannot report green
without `clang-check` and `native-full`, the only two that see layer 2. `--quick`
skips them and says in as many words that a layer-2 change must not be called
green on it. It also encodes the two greps that two gates are only correct with —
`--full` exits 0 BY DESIGN, so its summary line was never a verdict.

**2. ~~`./test.sh --probe`.~~ — DONE, 2026-08-04, as `tests/probe.sh`.** One
program, four ways — the interpreter half, `llvm.Run`, a native `-exe` binary, and
a real toolchain — diffed, and it reports WHICH PAIR disagrees, because that names
the layer: interp-vs-run is the grammars or the twin, run-vs-native is layer 2 or
the floor, ours-vs-oracle is a specification gap in however many engines agree.
It deliberately does NOT generate the programs from a spec: the per-language
program is the irreducible part, and the plumbing was what got re-invented.
Two worked templates in `tests/probe/`.

**3. ~~A shape-scan lint.~~ — DONE, 2026-08-04, as `tools/shape-scan`.** It
found on its own, at the permissive threshold, the groups the merge plan had only
ever found by hand — plus `rtCopyArr`'s seven copies. `-max N` is the CI ratchet;
the count is **2** at `-min 60` today and both are documented declines.
It also caught a defect in itself, which is the argument for building it: it
reported a group of three "identical" bodies of 6, 35 and 41 lines, which cannot
be true. Finding a function's end by scanning for a `}` in column 0 breaks on the
one-line form, so every one-liner swallowed everything up to the next column-0
brace. Brace-counting sees 1,713 bodies where that saw 1,409.

**4. Make the frozen-snapshot rule mechanical.** "Did you `-freeze` after touching
the emitter?" is a question a script should ask. A pre-commit check that regenerates
and diffs would remove a whole class of near-misses.

**5. `-O2` was missing from `buildExecutable` for the entire project's life.** Adding
it made every native binary 2.2× faster. **Look for more of those**: the build path
had never been profiled, and the second-biggest win (the literal cache thrashing) was
the same shape — nobody had measured, so nobody knew.

**6. ~~A benchmark suite with recorded baselines.~~ — DONE, 2026-08-04, as
`tests/bench.sh` + `tests/bench/baseline.txt`.** The same loop in six languages,
instructions retired, medians over layout draws, with each row's own noise floor
recorded so "is this a regression?" is a command. Building it is what exposed the
layout lottery in §4 — and the first version of it had that very bug, comparing
builds at `mktemp` paths that differed every run. **Every perf number in the plan
documents was then audited against the finding, and nothing in the 0.1-2% band
survived anywhere in either file.**

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

**lua is ~+4% and metajs +0.4%, and that is not the hashing.** (The lua figure
was re-measured twice after the layout lottery was understood — 13 draws a side
and 9 draws a side. The two runs agree the effect is REAL: it exceeds both rows'
spreads, `c` as a control moved +0.02%, and a 5x longer loop gives the same
percentage, so it is steady-state loop cost. They disagree 2% on the magnitude,
and the lower figure reproduced three times, so **the originally recorded +4.1%
stands.** `metajs +0.4%` is a single build and is not evidence in either
direction. The two "ways of buying it back, both worse" rows are single builds
against a 3.3%-wide row and are NOT established.) Their module scopes
are too small to be indexed at all; the cost is that a second arm makes `scope_find`
too big for clang to inline into `scope_get`/`scope_put`/`js_tdecl`. A branch that
can never fire already costs lua +2.8%. Two ways of buying it back were measured and
are worse (out-of-line arm +4.7%, name-in-slot +6.1%); both are recorded at the site.

The *other* half of this item — **making `import` APPEND** — was executed and is a
**LOSS of +42.5%**, reverted, and written up at the `ImportStmt` rule in
`metajs-to-llvm-ir.abnf` so it is not attempted a third time.

**The first thing the index paid for**: kotlin's 1,489-line verbatim `regex.js`
copy, declined twice on the tax this item removed, is now `import "./regex.js"`
at **+0.04%** (§7 item 4). The other merges the tax declined have not been
re-opened yet.

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
