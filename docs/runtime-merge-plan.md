# Merging the runtime: a shared MetaJS layer, and a smaller C floor

Third plan in the series, after [runtime-rework-plan.md](runtime-rework-plan.md) (what
was built) and [runtime-next-plan.md](runtime-next-plan.md) (GC, the floor primitives,
and all sixteen language migrations). HEAD when written: `c2e1811`.

## The goal, and the goal that is NOT here

Two things, in order:

1. **Merge what eleven layer-2 files implement separately** into a shared
   `languages/lib/runtime.metajs`.
2. **Move out of `languages/lib/runtime.c` whatever does not need to be there**, using
   "is it hot?" as the criterion rather than "is it possible?".

**The real prize is neither.** It is the DIVERGENCES the merge exposes: nine
implementations of `js_jadd` that are structurally identical and differ in their float
path is a question nobody has asked, and merging is what forces the answer.

**Explicitly NOT a goal: deleting `abnf/jsrt*.go`.** See "Why this makes the Go twin more
valuable, not less" at the end. That was measured and declined in
`runtime-next-plan.md` Part 4, and this refactor strengthens the case rather than
weakening it.

## Where things stand

```
matrix 325/325 · --full 5,686 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16, all sixteen "ok, and the clang executable agrees", none held
```

Hand-written source today:

```
C floor        languages/lib/runtime.c              5,061 lines, 301 bodies, 86 js_* externs
per-language C bash-rt.c 2,696 + batch-rt.c 490     3,186
MetaJS layer 2 11 files                            32,154
Go twin (KEPT) abnf/jsrt*.go                       30,010
```

---

# Part A - the shared MetaJS layer

## The measurement that motivates it

**29 extern names have between 2 and 9 implementations**, costing **1,427 lines across
the eleven files**:

```
js_jadd        9   csharp dart go java kotlin php python ruby swift
js_is_type     8   csharp dart go java kotlin python ruby swift
js_pyset       6   csharp dart go python ruby swift
js_mcall       6   csharp dart java lua php swift
js_dict_new    6   csharp dart go python ruby swift
js_supercall   5   csharp java js kotlin ruby
js_pylen       5   csharp dart python ruby swift
js_pyget       5   csharp dart python ruby swift
js_char        4 · js_char_code 3 · 20 more at 2
```

per file: csharp 262 · python 208 · java 176 · swift 174 · dart 152 · kotlin 130 ·
go 99 · ruby 99 · js 67 · php 56 · lua 4.

**And they are not trivially the same.** `js_jadd`, java against swift:

```js
// java                                    // swift
if (floIs(l) || floIs(r))                  if (floIs(l) || floIs(r))
    return jvmArith("+", l, r)                 return flo(swNum(l) + swNum(r), floStyle(...))
```

Same function, same shape, **different float path**. Whether they agree on every input is
unknown. That is the shape of the whole exercise.

**Name collision UNDERSTATES the duplication.** 1,665 function names appear in exactly
one file, but internal helpers are language-prefixed (`ktNum`, `phGenCall`, `dt_kget`,
`sw_kget`) so identical logic never collides by name. `sw_kget`/`sw_kset`/`sw_safeget`
transferred **verbatim** from swift to dart under new names — that duplication is
invisible to this measurement and must be found by reading.

## The mechanism already exists

Ruby proved a layer-2 file may `import` a shared MetaJS source: `import "./regex.js"`
compiles the 1,559-line engine straight into `ruby-rt.ll`, because
`metajs-to-llvm-ir.abnf` resolves relative imports and `-rt-lib`'s lazy boot runs them
into the same pinned scope. Regex cost **~460 lines of glue instead of ~2,000**. No new
feature is needed for Part A.

## Sequencing - one family at a time, never a big bang

Sixteen languages are green. A single merge commit touching eleven layer-2 files
produces a red suite nobody can bisect. Do this instead, and **verify all sixteen after
each step**:

1. **`js_jadd` first** (9 implementations, the widest, and the one with a known
   divergence in its float path). It is the pilot: it proves the mechanism, and it is
   small enough that a divergence found is a divergence understood.
2. `js_is_type` (8), then `js_pyset`/`js_dict_new`/`js_mcall` (6 each).
3. The `js_py*` family as a block — dart wrote it as language-neutral and ruby reused
   it, so it is already half-merged in spirit.
4. The 2-implementation tail, cheapest last.
5. **Then the invisible duplication**: read for the transferred-verbatim helpers
   (`*_kget`/`*_kset`/`*_safeget` in swift/dart/kotlin is the known one) and merge those.

For each family, in order:

- **Diff the implementations first, and write down every difference before merging.**
  That list IS the deliverable — a difference that turns out to be intentional is a
  language knob; one that turns out to be accidental is a defect that was invisible.
- Decide knob-or-defect **against `abnf/jsrt*.go`**, which is the independent reading.
  Where the twin and a layer-2 file disagree, the twin is the oracle unless the real
  language toolchain says otherwise.
- Merge into `languages/lib/runtime.metajs`, parameterised by the smallest knob set that
  covers the intentional differences. `lib/interp-core.js`'s `core` object is the model:
  thirteen knobs cover fifteen interpreters, each documented with the language that
  forced it.
- Re-run the full suite plus **the per-language differential probe** for every affected
  language. The probe is what catches a merge that changes an answer.

## Gate for Part A

- matrix **325/325**, `--full` **0 languages whose halves disagree**, `--cross` **119/0**,
  clang-check **16/16 all agreeing, none held**, `go test ./abnf/` ok, every
  `gen-*.sh --check` clean, `-freeze` a fixed point.
- Every affected language's differential probe (`llvm.Run` vs native) still byte-identical.
- **A written list of every difference found, each classified knob or defect**, with the
  defects fixed and pinned by a ratchet assertion whose discriminating power is measured.

---

# Part B - a smaller C floor

## What is actually irreducible

Phase 1 of `runtime-next-plan.md` measured the floor at **35 primitives** — the ones that
cannot be written in MetaJS because MetaJS compiles to them (allocation, member
load/store, indirect call, unboxed arithmetic, scopes, the control-signal family).
`runtime.c` today has **301 bodies and 86 `js_*` externs**. The difference is not
necessity; it is expedience, and most of it arrived because a language needed an answer
quickly.

**Cold, fiddly, correctness-heavy — the candidates:**

```
d_pow  d_exp  d_log  d_sqrt  d_frexp  d_ldexp  d_modf_int  d_mod  d_mod_go
dec_absdiff  dec_cmp  dec_mul  dec_norm  dec_to_double        (the decimal bignum)
fmt_apply  fmt_sprint  fmt_top  fmt_val  js_num_str           (number formatting)
the Unicode simple-case tables (328 ranges)
```

These are the parts that have produced the most defects per line this session — the
signed-zero trap alone bit `d_pow`, `d_mod_go`, swift's `rounded()`, java, and dart's
`math.Mod`. In MetaJS they would be easier to read, easier to test, and shared.

## The criterion, and the counter-pressure

**Move it only if it is cold.** Layer 2 runs roughly **5.5x slower** than the Go path and
allocates into a collected arena; `bench-alloc.sh` is at **3,711 B/iter** with the
collector off and RSS flat at ~3.3 MB. Anything on the string or arithmetic hot path
stays in C.

So the order is: **measure first, move second.** Instrument to find which floor bodies
are actually hot on `tests/bench-alloc.sh`, `lua-bench-alloc.lua` and
`metajs-bench-try.js`, publish the list, and move only from the cold end. Report
bytes/iteration and wall time before and after **every** move; revert anything that
costs more than a few percent and record the measurement at the site.

## What can never move

Documented in `runtime-next-plan.md` Part 4 and unchanged by this work: the 35
irreducible primitives, `setjmp`/`longjmp`, the pthread coroutine block, the conservative
GC (it scans a C stack), and the libc surface. Also note the floor's strings are
**length-carrying, not NUL-terminated** (`struct Cell` + `js_str_mem(i8*, i64)`), verified
against node — do not "fix" that into C-string semantics.

## Gate for Part B

Same suite gates as Part A, plus: `tests/coro-poc/build.sh` with `--gc` and `--break`
unchanged, memory still bounded (rss flat at 250k/1M/2M), and a before/after table for
every body moved.

---

# Why this makes the Go twin MORE valuable, not less

The tempting conclusion — "once layer 2 is merged we have something like the old Go
runtime, so delete `abnf/jsrt*.go`" — is exactly backwards, and it is worth writing down
because it will be proposed again.

**Today there are eleven independent readings** of the shared semantics: eleven files,
written by eleven separate processes from `abnf/jsrt*.go`. That independence is why the
`js_jadd` divergence above exists to be found.

**After a successful merge there is one.** The merge improves consistency by destroying
independence — that is what merging means. The Go twin then becomes the *only* remaining
artifact written from a separate reading, and the only thing that can still disagree with
`runtime.metajs`.

And the structural blockers do not move:

- `llvm.Run` **cannot execute the C floor** — `setjmp`/`longjmp`, `pthread_create`+`dlsym`,
  and a collector that scans `[sp, GC_STACK_BASE)` of a C stack that does not exist there.
  Part 4 demonstrated a live array being collected. A merged layer 2 still bottoms out on
  C, so it still cannot back the interpreter.
- `abnf/jsrt.go` is the engine `-frozen` itself runs on (`newJSRT` at
  `frozen.go:220/530/813`).
- `jsrtint.go`/`jsrtjvm.go` carry the `sint*`/`flo*`/`keysOf`/`delKey` bindings that
  **layer 2 itself calls**.

**The twin is the oracle for this entire plan.** Every merge step is validated by diffing
against it. Removing it would remove the instrument.

---

# Rules (unchanged, and every one was earned)

- Never `git stash` / `checkout` / `reset` / `clean` — repo-wide, destroys concurrent work.
  Use `git show HEAD:<file>`.
- Agents never commit; the coordinator verifies and commits, by explicit path.
- **A change that fixes N and breaks M>N is a LOSS**: revert it and record the measurement
  at the site.
- **`mec languages/x.abnf prog` reads the grammar from that path AT RUN TIME.** To compare
  versions, `git archive <rev> | tar x -C <dir>`, `go build` INSIDE, run there — otherwise
  you compare a change against itself.
- **A crashing binary looks fast.** Check the exit code AND the expected output, and delete
  stale binaries before re-verifying.
- **`./test.sh --full`'s summary is not sufficient alone** — also
  `grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF'`.
- Brace value-less early returns (frozen-only ASI trap, `abnf-dialect-gotchas.md:539`).
- **Write layer 2 against the Go CODE, not the Go COMMENTS.**
- **Measure a new assertion's discriminating power** against a clean archive. Assertions
  that pass either way are decoration; say so when they are.
- Verify a commit from a clean checkout, not the working tree.
