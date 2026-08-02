# Self-hosted runtime: the plan

## The goal

Every compiled language produces a **self-contained native binary**, and the semantics of
each language exist **once** instead of twice.

Today twelve of the fifteen compilers emit *handle IR* - LLVM that calls `js_*` externs
implemented in Go (`abnf/jsrt*.go`, 29,657 lines, 570 distinct externs). clang has
nothing to link those against, so `-exe` is available only to bash, batch and c. Worse,
those Go functions are *twins* of JavaScript already in `languages/lib/`: the interpreter
implements each language's semantics in JS, the compiler implements the same semantics
again in Go, and they are kept in sync by hand. `interp-core.js:287` says so outright:

> Must match the Go twin exactly (isTypeName in abnf/jsrt.go, extern js_is_type)

`./test.sh` cannot see drift between the twins, because both halves of the matrix run the
*same* copy and agree with each other by construction.

## The target architecture

| layer | contents | written in | compiled by |
|---|---|---|---|
| 3 | per-language knobs (13 of them today) | MetaJS | the language grammar |
| 2 | shared semantics + regex | MetaJS | `metajs-to-llvm-ir.abnf` |
| 1 | 35 primitives (measured, phase 1) | C | `c-to-llvm-ir.abnf` |

Every layer compiled by this repo. No hand-written `.ll`, no Go in the produced binary.

### Evidence this is reachable (measured 2026-08-02, not assumed)

- `metajs-to-llvm-ir.abnf` compiles `languages/lib/interp-core.js` unchanged into
  **13,555 lines of IR, 148 functions, and clang accepts the module.** `regex.js`
  compiles too. Layer 2 is not a rewrite; it largely exists.
- `c-to-llvm-ir.abnf` handles structs, pointer members, `sizeof`, heap allocation and
  **function pointers, including a function pointer in a struct field called through it**
  - which is exactly the closure representation `js_closure`/`js_call` need. A probe
  printed `468` identically under `llvm.Run` and as a clang-built binary.
- Exceptions need no `setjmp`: `compile-core.js:202` already implements them as a
  **returned control signal** (`js_ctl_*` + `excDispatch`), not stack unwinding. That is
  ordinary C.
- The MetaJS self-hosting surface is only **48 externs** (measured; the "43" this line
  used to claim missed the `js_ctl_*` family, which `lib/compile-core.js` emits) - generic
  and language-neutral - against 55-101 per language for the other compilers.

### Why layer 1 cannot also be MetaJS

Bootstrap circularity, not portability. Every MetaJS number compiles to `js_num_i`,
every call to `js_call`. `js_call` written in MetaJS compiles into something that calls
`js_call`. There is no bottom. Architecture is a minor concern: LLVM IR plus clang
covers targets, and the OS-specific surface is roughly *allocate, write bytes, exit*.

C is not sacred. What layer 1 needs is any language we can compile that has **unboxed
integers and raw memory**. C is the one we already have a working native compiler for.

## Invariants - none of these may regress at any checkpoint

- matrix **308/308**, byte-identical stdout **and stderr** between goja and `-frozen`
- `--full` ratchet **4,822 assertions**, no language whose halves disagree
- `--cross` **119/0**
- `tests/clang-check.sh` **15/15**
- **Never a green suite over a binary nobody executed.** Any language that becomes
  self-contained must move from clang-check's `ok (module only)` row into the
  run-it-natively row in the same change.

---

## Phase 0 - unblock (prerequisite, already chipped)

Two live defects block a C runtime outright.

1. **`(long)ptr` emits `sext` instead of `ptrtoint`.** clang rejects the module. The
   whole handle ABI is `i64`, so a runtime in C converts pointers to integers
   constantly. Invisible to `./test.sh` (`llvm.Run` resolves by handle, not by type) and
   to clang-check (no `*-test-full.c` casts a pointer to an integer).
2. **`stubUndefined` nulls `malloc`.** `libcExterns` (`abnf/llvmmap.go:1004`) is only
   `{putchar, getchar, puts, abs}`; everything else declared-but-undefined gets a
   zero/null-returning body. A C program that allocates **builds successfully and
   segfaults**, while the identical module handed straight to clang prints the right
   answer. Same failure class as bash's `rt_regex_search`.

**Gate:** a C program using `malloc` and function pointers builds via `-exe` and runs
correctly, and `tests/c-test-full.c` covers both (pointer round-tripped through a long;
heap-allocated struct written and read back).

## Phase 1 - measure the floor - DONE (2026-08-02)

**The floor is 35, not ~15. 13 of the 48 primitives leave.**

Built: `languages/lib/rt-prims.metajs` (the derived primitives, in MetaJS) plus the
`-rt-prims` flag, which makes `metajs-to-llvm-ir.abnf` compile that file into the module
it is building and route each operator to the compiled MetaJS body instead of declaring
the Go external. What the emitted module still `declare`s is then the floor, by
construction - no inspection, no judgement call:

```
$ mec languages/metajs-to-llvm-ir.abnf tests/metajs-test-features.js -q -rt-prims | grep -c '^declare'
35
```

**Correction to the numbers above:** the surface is **48**, not 43. The 43 came from
grepping the grammar file; five more (`js_ctl_break`, `js_ctl_continue`, `js_ctl_kind`,
`js_ctl_return`, `js_ctl_value`) are emitted by `lib/compile-core.js`, and `js_srcpos`
is a 49th under `-trace`.

### The measured floor (35)

| primitive | why it cannot leave C |
|---|---|
| `js_num_i` | turns a raw i64 literal into a handle - the only way a constant enters |
| `js_str_mem` | takes an `i8*` and a byte length; the one non-`i64` signature |
| `js_obj_new`, `js_arr_new` | allocation |
| `js_get`, `js_set` | the member load/store |
| `js_call` | an indirect call through a handle |
| `js_closure` | its first argument is a raw function index, not a handle |
| `js_arg` | its second argument is a raw index, and it binds the parameters of *every* compiled function, including any MetaJS primitive |
| `js_arr_push` | it *is* the calling convention: entering a MetaJS primitive means building its argument array with it |
| `js_scope_new`, `js_scope_get`, `js_tdecl`, `js_tset` | every function prologue and every name read; a MetaJS implementation's own locals go through them |
| `js_truthy` | returns an unboxed 0/1 that the emitted `icmp` compares, and every `if` in an implementation calls it |
| `js_getret`, `js_setret` | a single global mutable cell; MetaJS has no way to name one (its own globals live in a scope object, which is a floor primitive) |
| `js_add`, `js_mul`, `js_div`, `js_mod` | unbox, and unboxing is a load |
| `js_lt` | the one relational comparison that has to unbox |
| `js_seq` | the one equality that has to read a type tag |
| `js_band`, `js_bor`, `js_bxor`, `js_shl`, `js_shr` | ToInt32 is an unboxing load |
| `js_throw`, `js_try` | the panic/recover pair itself |
| `js_ctl_return`, `js_ctl_break`, `js_ctl_continue`, `js_ctl_value` | build and read the control-signal value that `js_try` recognises **by Go type**; they move only when `js_try` does |
| `js_ctl_kind` | returns an unboxed 0..3 that the emitted `icmp` compares |

### Derived in MetaJS (13)

`js_eq`, `js_ne`, `js_sne`, `js_not`, `js_gt`, `js_le`, `js_ge`, `js_neg`, `js_sub`,
`js_tonum`, `js_num_str`, `js_ushr`, `js_typeof`.

Three coercions replace every type test, so no tag read is needed: `x === x + ""` is the
string test, `x === x * 1` the number test, `x === x` the NaN test.

### Where the hypothesis was wrong

**Wrong in the optimistic direction - these do NOT leave the floor:**

- **The whole scope family stays.** "A scope is just an object with a parent link" is
  true of the *representation* and irrelevant to the *bootstrap*: a MetaJS `js_scope_get`
  is a MetaJS function, and its own parameters are bound with `js_tdecl` into a scope made
  by `js_scope_new` and read back with `js_scope_get`. Routed anyway, the module carries
  both a `declare` and a `define` of the name, and the run dies:
  `js runtime error: call of a non function value: undefined (last member lookups: main (*abnf.jsScope))`.
  `registerPrim` now warns when a primitive is used by its own implementation, so this
  can no longer pass silently.
- **`js_truthy` stays**, for two reasons: it returns an *unboxed* 0/1 (the emitter writes
  `icmp ne js_truthy(v), 0`, and a MetaJS body returns the boolean *handle* 2 or 3, both
  non-zero), and every branch inside its own implementation calls it. Routed anyway,
  every condition became true: `FAIL if-elseif-else: got bigbigbig want bigmidsmall`,
  then the step-limit brake fired on the first `while`.
- **`js_arg` and `js_arr_push` stay** - they are the calling convention, so calling the
  MetaJS version needs them first.
- **`js_getret`/`js_setret` stay.** Not because of the statement form (a `for`-init
  clause emits no `js_setret`), but because they need one global cell and MetaJS cannot
  name one without the scope primitives.
- **`js_try`/`js_throw` stay.** The returned-signal protocol is real and is what makes a
  C floor possible - but it lives in the *emitter* (`compile-core.js` `excDispatch`), not
  above the primitives: `js_try` still has to run a closure under a recover and recognise
  a `jsCtl` by its Go type.

**Wrong in the pessimistic direction - these DO leave the floor:**

- **`js_sub`** - `a - b` is `a * 1 + b * ~0`. `js_add` already unboxes; subtraction adds
  no new load. (The `-1` is written `~0` on purpose: a literal `-1` is unary minus.)
- **`js_eq`** - only *one* equality primitive has to read a type tag. Loose equality is a
  coercion cascade on top of `===`, and the Go twin (`looseEq`) already answers false for
  primitive-against-object instead of running ToPrimitive.
- **`js_typeof`** - reachable from `===` against the four singleton handles plus the two
  coercions; functions are separated from arrays by rendering as `[function]` with no
  `length`.
- **`js_ushr`** - `>>` plus a mask, both already present.

### Ground truth

- matrix **311/311** (308 + three new `-rt-prims` entries in `launch.json`), byte-identical
  stdout and stderr goja vs `-frozen`; the emitted IR itself is byte-identical between the
  engines with `-rt-prims` on (9,586 lines for the features test).
- `--full` **4,849 assertions, 0 languages whose halves disagree**; `--cross` **119/0**;
  `clang-check` **15/15**, and `clang -S -x ir` also accepts the `-rt-prims` module.
- A 625-cell differential probe (`==` `!=` `===` `!==` `<` `>` `<=` `>=` `-` over 25
  values incl. NaN, ±Inf, ±0, strings, arrays, objects, a closure; plus `typeof`, `!`,
  unary `-`, `*1`, and 63 shift cases) is byte-identical with and without `-rt-prims`.

**Cost:** +1,235 IR lines on the features test (8,351 -> 9,586); no measurable compile-time
difference. `-rt-prims` is opt-in and off for tag scripts (the frozen tag-script runtime's
`c` has no `parse`), so the bootstrap does not pay for the measurement.

## Phase 2 - the linking feature - DONE (2026-08-02)

`buildExecutable` wrote one temp `.ll` and ran `clang -Wno-override-module -o out tmp.ll`.
It now takes a third argument - the grammar's runtime - and links it, plus `-L`/`-l`:

- `llvm.BuildExecutable(m, c.exePath, c.runtime)`. The runtime list is any mix of
  `.c`/`.ll`/`.o`/`.a`; the grammar's own list and the new **`-rt FILE`** flag (which a
  grammar reads as `c.runtime`) are merged, deduplicated, in that order - so a grammar
  supplying its own runtime and a user adding an object file compose instead of
  overriding. `-rt` works even for a grammar that passes nothing, because
  `buildExecutable` merges it itself.
- **`-L DIR` / `-l NAME`** reach clang verbatim (write them separated: `-l m`, not `-lm`,
  because `-lb`/`-lf` are existing parser flags).
- **Stubbing is now conditional, and that is the important half.** With nothing to link,
  a declared-but-undefined symbol can only be a language-level name with no symbol
  anywhere (the lisp form referencing a never-defined name inside a short-circuit), so
  `stubUndefined` links a zero body and warns - unchanged. As soon as the build links a
  runtime (`c.runtime`/`-rt`, or any `-l`), the same symbol is a genuine link error:
  nothing is stubbed, and an unresolved name is reported on stderr *by name* with a
  non-zero exit. The report is built from the MODULE (sorted, filtered by what the linker
  mentions), never from raw linker text, which carries temp file names the matrix would
  see as noise.

`metajs-to-llvm-ir.abnf` grew the `-exe` branch (in the goja-driver tail, so
`jsbootstrap.ll` regenerates byte-identically; `jsagrammar.go` does change, as it embeds
the whole start script).

**Gate:** the plan's stated gate needs the phase-3 C floor, so it was proven with an
explicit STAND-IN instead: `tests/metajs-link-stubrt.c` defines by hand the ten `js_*`
the smallest MetaJS program declares, plus a `main` calling `jsmain`.

```
$ mec languages/metajs-to-llvm-ir.abnf tests/metajs-link-hello.js -q \
      -exe tests/metajs-link.out -rt tests/metajs-link-stubrt.c
js compiler: wrote executable tests/metajs-link.out          # exit 0, and it runs
$ nm tests/metajs-link.out | grep -c ' T _js_'
11                                                            # every extern resolved

$ mec ... tests/metajs-link-missing.js -exe ... -rt tests/metajs-link-stubrt.c
error: 1 unresolved symbol(s), and this build links a runtime, so they are NOT stubbed:
error:     js_add                                             # exit 1, nothing written

$ mec languages/lisp-to-llvm-ir.abnf tests/lisp-test-features.txt -q -exe tests/lisp-link.out
warning: no definition for boom; linking a stub that returns zero ...   # exit 0
$ tests/lisp-link.out
features: 63 checks, 0 failures                               # the stub case still works
```

`-l`/`-L` against a real library (not in the matrix - it depends on what the machine has):
a C program calling `zlibVersion()` links only with `-l z`, and `-L <dir> -l mecdemo`
against a hand-built `libmecdemo.a` printed `42`. A bogus `-l` fails with
`ld: library 'mec_no_such_lib' not found`, which is the matrix entry that pins the
passthrough permanently.

Four `launch.json` entries were added (matrix 311 -> 315): the metajs link, the
unresolved-symbol failure, the lisp stub-and-warn build, and the bogus `-l`.

## Phase 3 - runtime.c, the floor

Implement the measured floor from phase 1 in C, compiled by `c-to-llvm-ir`.

- **Value representation:** `i64` handle = tagged pointer. The emitted IR is already
  all-`i64` (`js_str_mem(i8*, i64) -> i64` is the only exception), so nothing in any
  emitter assumes Go.
- **Allocation:** arena bump-allocator, the approach already proven in the bash and
  batch native runtimes. No GC initially - see Risks.
- **Globals:** `println` is not an extern; it resolves through `js_scope_get` against a
  global scope the runtime pre-seeds (`abnf/jsrt.go:8231`). The C floor must seed it too.

**Gate:** `tests/metajs-test-full.*` runs as a native binary and agrees byte-for-byte
with `llvm.Run`. Add metajs to clang-check's run-natively row.

## Phase 4 - layer 2, one pilot language

Compile `interp-core.js` (proven) and export its functions under the extern names the
emitters call. Do **one** language first.

**Pilot: Lua** - smallest surface on both axes:

```
lua        externs=55   go twin=774 lines
java       externs=61   go twin=804
csharp     externs=67   go twin=813
swift      externs=68   go twin=986
dart       externs=72   go twin=896
js/ts/py/ruby/go  externs=88-99   (twins spread across jsrt.go and friends)
php        externs=101  go twin=1968
kotlin     externs=96   go twin=5934
```

The work is *not* writing a runtime - it is exposing layer 2 under the extern names, and
reconciling the JS and Go twins wherever they have drifted. Every reconciliation is a
latent bug found.

**Gate:** lua produces a native binary; the lua ratchet is byte-identical across
`llvm.Run`, the native binary, goja and `-frozen`. Only then delete `abnf/jsrtlua.go`.

## Phase 5 - roll out

Repeat phase 4 per language in the size order above, easiest first, kotlin last. One
language per change. **Never delete a Go twin before its language passes natively.**

## Phase 6 - regex

`languages/lib/regex.js` already exists and already compiles, and replaces roughly 4,273
lines of Go (`jsrtregex.go`, `jsrtregexjs.go`, `jsrtregexkt.go`, `jsrtregexpy.go`,
`jsrtregexptr.go`). Preserve the dialect modes - `e` POSIX ERE, `j` JS repeat, `p` POSIX
classes. **Those are by design, not bugs:** `^(a*)*$` capturing `"aaa"` is JS+POSIX while
Python, Java, Perl and Ruby capture `""`. Unifying them would break three languages.

## Phase 7 - retire the Go runtime

29,657 lines deleted. `jsrt.go`'s host-function globals become MetaJS.

---

## Risks, honestly

1. **Performance.** A MetaJS-compiled runtime bottoming out in 15 C primitives will be
   markedly slower than hand-written Go. Measure at the phase 4 gate, not at the end.
   Fallback if it is unacceptable: keep the Go runtime for `llvm.Run` (the fast inner
   loop for tests) and use the self-hosted one only for `-exe`. That reintroduces a twin,
   so treat it as a defeat, not a design.
2. **`llvm.Run` also loses its Go runtime - and this is mostly good.** Once the runtime
   is linked IR, `js_kget` stops being an extern and becomes a defined function
   `llvm.Run` simply executes, bottoming out at the C floor and then at `malloc`/`putchar`
   which `resolveExtern` already handles. End-to-end self-consistent, and it means the
   interpreted and compiled halves finally run *the same runtime* - which is the twin
   problem dissolving rather than moving. It is also the main slowdown.
3. **No GC.** An arena leaks. Fine for compile-and-run tests, not for long-running
   programs. Decide explicitly whether to ship refcounting later or accept the limit.
4. **The `-frozen` subset binds layer 2.** The MetaJS runtime must itself run under the
   frozen bootstrap: no `for...in`, no `splice`, `array.length` not assignable, a
   reassigned parameter *or local* keeps its first type, no `split`/`join`, no exponent
   literals in tag scripts. `interp-core.js` already satisfies all of this - do not
   regress it.
5. **Drift during migration.** Both runtimes exist for as long as the rollout takes.
   Every phase-4/5 change must keep the un-migrated languages on Go.

## Rules for anyone executing this

- Never `git stash/checkout/reset/clean` - it is repo-wide and destroys concurrent work.
  Use `git show HEAD:<file>` for a clean copy.
- Do not tune a test to flatter the runtime. A red section is information.
- **A change that fixes N cases and breaks M>N is a LOSS**: revert it and record the
  measurement at the site so nobody re-derives it.
- Verify a commit from a clean checkout (`git archive HEAD | (cd /tmp/x && tar xf -)`),
  not from the working tree - `swift-to-llvm-ir.abnf` once shipped without the
  `abnf/jsrtswift.go` defining the externs it called, and the working tree hid it.
- Ground truth is mandatory. Paste the output.
