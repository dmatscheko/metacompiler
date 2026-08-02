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

(The figures are the ones current after phase 3; the parenthesised ones are what the
plan was written against.)

- matrix **318/318** (was 308), byte-identical stdout **and stderr** between goja and
  `-frozen`
- `--full` ratchet **5,148 assertions** (was 4,822), no language whose halves disagree
- `--cross` **119/0**
- `tests/clang-check.sh` **16/16** (was 15/15), with bash, batch, c **and metajs** in the
  run-it-natively row
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

## Phase 3a - `char` is one byte - DONE (2026-08-02)

A prerequisite for phase 3 that phase 0 uncovered. The floor is written in C and it is
about **bytes and addresses**: `js_str_mem` takes an `i8*` and a byte length, a string is
a byte array, a handle is a tagged pointer. `languages/c-to-llvm-ir.abnf` gave `char` a
**four-byte cell** - `machBytes()` answered 4 for every integer narrower than a long -
so `"hello"` was emitted as `[6 x i32]` and no byte-oriented code could be expressed at
all. Commit 43cf14e had already measured this and deliberately kept the whole
`str*`/`mem*` family, `atoi`, `atol` and `strdup` OUT of `libcExterns`.

### What changed

`languages/c-to-llvm-ir.abnf`. The model now separates a **cell** from a **register**:

- `ctLTy` spells every integer width honestly - `char`/`_Bool` `i8`, `short` `i16`,
  `int` `i32`, `long` `i64` - and `void` is `i8` so that GCC's `sizeof(void) == 1` and
  the one-byte step of `void *p; p + 1` finally agree with each other.
- `machBytes` is now exactly the byte size of `ctLTy`. The two MUST stay in step: a
  `getelementptr` strides by the LLVM element type while a pointer *difference* divides
  by `machBytes`, so any disagreement makes `p + 1 - p` answer something other than 1.
- A **value** still lives in an `i32`/`i64` register - that is C's own integer promotion
  - so a narrow cell is extended on every load (`loadCt`/`promoteCell`, signed or
  unsigned by the C type) and truncated on every store (`storeTo`). Function
  *signatures* keep passing narrow integers promoted, exactly as before; only the
  parameter's own slot became a cell, and the incoming argument is now converted into
  the declared width there (`f(300)` with `char c` sees 44, as cc does).
- `cellAddr` bitcasts an address whose LLVM type does not match the cell. This is
  **machinery the new model needs, not a defect it fixes**: a pointer *value* is the
  uniform `i32*`, and once `storeTo` truncates, `short *to; *to++ = x` makes llir refuse
  the store (`src=i16; dst=i32*`). At HEAD there was no truncation, the store was `i32`
  into `i32*`, and that construct already answered correctly (4464, same as cc).
- String-literal objects store **converted** elements (`kcell` + `convertNum`), so
  `"\xff"[0]` is the `-1` a signed char holds.

Two further defects are fixed in the same file. Both are **pre-existing and reachable at
HEAD** - they were found while widening `char`, not caused by it, and each produces a
wrong answer *and* a module clang rejects (measured below, section 45 of the ratchet):

- `emitDirectCall` reported a **pointer-returning** function's result as `ctInt`, though
  `retIsPtr` already gives it an `i32*` LLVM result. `s45_at(a, 2) - a` therefore took
  the pointer+integer branch of `emitBinC` with the operands the wrong way round and
  emitted `sext i32* %p to i64` - IR `llvm.Run` accepts and clang refuses ("invalid cast
  opcode").
- `makeCondExpr` left a null-constant arm as an `i32` against a pointer arm, emitting
  `phi i32* [ %p, ... ], [ 0, ... ]` - "integer constant must have integer type".

`languages/c-interpreter.abnf`: `strLitAddr` wrote **raw bytes** into the literal object
instead of converting them to the element type, so `"\xff"[0]` answered 255 here while
the compiler half and cc answered -1. That is now the one-line `convert(ect, ...)` the
array-initializer branch two lines above already did.

The interpreter is **not** byte-packed, and was deliberately left that way: its memory is
one cell per scalar, which is a legitimately different model. It already answered
`sizeof(char) == 1` and `sizeof "hello" == 6` correctly, and no `malloc` was needed for
the new section, so none was added.

### Measurements (real `cc` is the oracle - Apple clang 21.0.0, arm64-darwin)

**How to reproduce these.** Every "HEAD" figure below comes from a clean archive tree
with its own binary, never from the working tree:

```
git archive e81dc94 | (cd /tmp/headco && tar xf -)
(cd /tmp/headco && go build -o /tmp/mec-head .)
/tmp/mec-head /tmp/headco/languages/c-to-llvm-ir.abnf <probe>.c -q
```

Running `mec languages/c-to-llvm-ir.abnf` from the repository root reads the **working
tree**, so a before/after comparison done that way compares the change against itself
and reports "identical". It is worth stating because that is exactly how the first
review of this phase concluded the change was a no-op.

The literal, and the struct:

```
HEAD e81dc94:  @.str.1 = global [6 x i32] zeroinitializer
phase3a:       @.str.1 = global [6 x i8] zeroinitializer

struct S { char a; short b; int c; long d; };
HEAD e81dc94:  %0 = alloca { i32, i32, i32, i64 }
phase3a:       %0 = alloca { i8, i16, i32, i64 }
```

HEAD's own `tests/c-test-full.c` through both grammars, module plus program output:

```
IR+output bytes: HEAD=229251  3a=231290      DIFFERENT
< @.str.1 = global [4 x i32] zeroinitializer
> @.str.1 = global [4 x i8] zeroinitializer
(both still pass it: full: 397 checks, 0 failures)
```

**The reproducing case.** `char` was the one width where `sizeof` and the actual layout
disagreed, which is why `sizeof` alone could never see it - `ctBytes` was honest all
along and only `machBytes`/`ctLTy` were not:

```c
char a[8]; char *p = a;
pn((long)&a[1] - (long)&a[0]);       /* 1 : a char array walks single BYTES */
pn((long)(p + 1) - (long)p);         /* 1 : a char* steps one byte          */
pn((long)&a[8] - (long)&a[0]);       /* 8 : consistent with sizeof a        */
pn((long)sizeof a);                  /* 8                                   */
{ const char *w = "\xff\x7f"; pn(w[0]); pn(w[1]); }   /* -1 127 */
```

```
cc (oracle)     1 1  8 8  -1 127
HEAD e81dc94    4 4 32 8 255 127        <- sizeof says 8, the layout is 32
phase3a         1 1  8 8  -1 127
```

The same three-way run over the two pointer-return defects (ratchet section 45):

```
cc (oracle)     2yn
HEAD e81dc94    <garbage byte>yn   and clang: "integer constant must have integer type"
                                   at  %10 = phi i32* [ %7, %6 ], [ 0, %8 ]
phase3a         2yn                and clang accepts the module
```

HEAD against the new ratchet file - `cc` answers all 444 correctly:

```
cc          full: 444 checks, 0 failures
HEAD        FAIL 4409  FAIL 4410  FAIL 4414  FAIL 4437  FAIL 4501  FAIL 4504  FAIL 4507
phase3a     full: 444 checks, 0 failures
```

The probe from the `libcExterns` comment, and 27 more over the whole family
(`strlen strcmp strncmp strcpy strncpy strcat strncat strchr strrchr strstr memcpy
memmove memset memcmp memchr atoi atol strdup`), our module linked by clang against the
real libc versus `cc` on the same source:

```
cc     5 0 0 -1 1 0 -1 hello 5 world abcd xy123 2 3 4 1 qqqqq abcdef 010123478 -1 0 2 42 -7 1234567890 dup! 4
clang  5 0 0 -1 1 0 -1 hello 5 world abcd xy123 2 3 4 1 qqqqq abcdef 010123478 -1 0 2 42 -7 1234567890 dup! 4
```

Byte-identical. The same family **implemented in C and compiled by this grammar** - which
is what a Go `libcNative` walking the arena would see - agrees under all three engines:

```
cc         5 0 0 -1 0 -1 hello 5 world 2 4 1 qqqqq abcdef -1 0 42 -7 dup! 4 6 1 2 4 1 -1 255 -32768 32768 3 ABC
llvm.Run   5 0 0 -1 0 -1 hello 5 world 2 4 1 qqqqq abcdef -1 0 42 -7 dup! 4 6 1 2 4 1 -1 255 -32768 32768 3 ABC
-exe       5 0 0 -1 0 -1 hello 5 world 2 4 1 qqqqq abcdef -1 0 42 -7 dup! 4 6 1 2 4 1 -1 255 -32768 32768 3 ABC
```

### The arena is not the bug - it never was

Worth recording, because the first reading of the `strlen("hello") == 1` measurement put
the four bytes on the Go side. It is not there. `ma.sizeOfUncached`
(`abnf/llvmmap.go:1942-1959`) derives the whole arena layout from the module's own LLVM
types and is completely honest about width:

```go
case *types.IntType:   return (t.BitSize + 7) / 8
case *types.ArrayType: return t.Len * ma.sizeOf(t.ElemType)
case *types.StructType: /* packed sum of the fields */
```

So the arena did exactly what the module told it to. The module said `[6 x i32]`, the
arena reserved 24 bytes with three padding NULs after each character, and a byte-walking
Go `strlen` correctly reported **1** for what was actually there. Nothing on the Go side
needs changing, and no fix should be dispatched to that layer.

The apparent contradiction - "a `char` array reports `&ca[1] - &ca[0] == 1`, which looks
byte-accurate, yet a byte-walking `strlen` returns 1" - has two separate causes, and
neither is the arena:

- `&ca[1] - &ca[0]` is a C **pointer difference**, which the grammar emits as
  `(ptrtoint b - ptrtoint a) / machBytes(char)`. At HEAD that divided a raw distance of 4
  by 4 and answered 1. The scaling hid the layout; it did not correct it. The unscaled
  form is what exposes it, and it is why the ratchet now asserts the unscaled form.
- `&s.d - &s.a == 7` for `struct S { char a; short b; int c; long d; }` is **phase3a's**
  answer, not HEAD's. Measured three ways (scaled distance, raw distance, `sizeof`):

```
cc (oracle)     8  8 16     <- aligned layout
HEAD e81dc94    3 12 15     <- raw 12 disagrees with sizeof 15
phase3a         7  7 15     <- raw == scaled == the packed sizeof
```

Neither model matches cc's *aligned* struct size, and that is the pre-existing packed
layout choice this phase does not touch. What phase3a fixes is the model disagreeing
with **itself**: at HEAD `sizeof a == 8` while `&a[8] - &a[0] == 32`.

### The libc names that are now safe to add

All 18 pass condition 1 (measured above). All 18 are pure byte/address work over
`ma.mem` plus, for `strdup`, `ma.heapAlloc`, so condition 2 is satisfiable - each has to
be **implemented in `libcNative` in the same change**, never listed for clang alone:

```
strlen  strcmp  strncmp  strcpy  strncpy  strcat  strncat
strchr  strrchr strstr
memcpy  memmove memset  memcmp  memchr
atoi    atol    strdup
```

One caveat, measured, for `strcmp`/`strncmp`/`memcmp`: the **magnitude** of the result is
unspecified by C, and Apple's libc returns the unsigned-byte difference. With runtime
operands `cc` and the real symbol agree exactly:

```
strcmp(x,"abd") -1   strcmp(y,x) 1   strcmp("a","abc") -98   strncmp/memcmp("abcz","abcd",4) 22
```

but with **literal** operands `cc` constant-folds to a normalized sign where the linked
symbol returns the raw difference (`strcmp("\x80","\x01")`: cc 1, libc 127). That is cc
optimizing an unspecified value, not a memory-model defect - but a `libcNative`
implementation must return the unsigned-byte difference, not a normalized -1/0/1, or
`llvm.Run` and `-exe` will disagree on programs that print it.

`puts` stays where it was. Nothing else moves: the varargs `printf` family, `exit`,
`abort`, `qsort` and `bsearch` are still blocked by condition 2 for the reasons
`libcNative` already records.

### Pinned

`tests/c-test-full.c` gained **SECTION 45 (8 checks, 4501-4508)** for the two
pointer-return defects - HEAD fails 4501, 4504 and 4507 - and **SECTION 44 (39 checks,
ids 4401-4439)**: `sizeof(char)`,
`sizeof(short)`, `sizeof "hello" == 6`, single-byte indexing and `char*` stepping
(`(long)(p + 1) - (long)p == 1`), a byte-oriented scan (what `strlen` is), the full
signed and unsigned `char` ranges round-tripping, the `short` wraps, and the string
literal's signed high bytes. `cc` answers every one of them identically.

Deliberately **not** asserted: the raw byte distance between two `short` array elements
(2 in the compiler and cc, 1 in the cell-addressed interpreter). 4435 asserts the
*scaling* instead, which both halves answer. A `char` is the one width where the two
models coincide, and that is what makes the byte assertions sayable at all.

### Known defect, out of scope, measured

`_Bool b; b = 5;` holds **5**, where cc holds 1. Both halves agree with each other, so
neither `--cross` nor the ratchet can see it. It is not a width bug: `_Bool` is modelled
as `ctI(1, true)` in *both* grammars, which is indistinguishable from `unsigned char`, so
the store path cannot tell them apart and normalizing needs a new type-descriptor flag
threaded through `convert`/`emitConv`/`ctEq` in both halves. Only the *initializer* is
normalized today (`makeBoolDecl`). Unchanged by this phase - it behaves exactly as it did
at HEAD.

### Ground truth

matrix **315/315** (308 + phase-1's three `-rt-prims` entries + four added concurrently),
`--full` **4,896 assertions, 0 languages whose halves disagree** (c 397 -> 444),
`--cross` **119/0**, `clang-check` **15/15** with c reporting *"ok, and the clang
executable agrees"* - so the new section is checked in a real clang-built binary, not
only under `llvm.Run`.

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

## Phase 3 - runtime.c, the floor - DONE (2026-08-02)

**A MetaJS program now compiles to a self-contained native binary whose only
undefined symbols are six libc names.** `languages/lib/runtime.c` (2,294 lines)
implements **all 48** externs `metajs-to-llvm-ir.abnf` declares - the 35 of the phase-1
floor plus the 13 that `-rt-prims` can lift into MetaJS - and it is compiled by
`languages/c-to-llvm-ir.abnf`, not by clang.

```
$ nm -u tests/metajs-native-full.out
_exit  _longjmp  _malloc  _putchar  _setjmp  _write
$ nm tests/metajs-native-full.out | grep -c ' T _js_'
52
```

### What was built

- **`languages/lib/runtime.c`** - the floor. Value representation: a handle is a pointer
  to a 7-word arena cell with a tag (number/string/array/object/closure/host function/
  bound method/control signal/scope). Allocation is a bump arena, never freed - the
  accepted limit under Risks. Strings are UTF-8 byte buffers with the **UTF-16 code unit
  view** `abnf/jsrt.go` exposes (`.length`, `charCodeAt`, `charAt`, `slice`,
  `substring`, `indexOf`, `split`), including WTF-8 for a lone surrogate and the seam
  rejoin in `+` that makes `s[0] + s[1] === s` hold for an astral character. The root
  scope is pre-seeded with `println`, `print`, `eprintln`, `printf`, `sprintf`, `sprint`,
  `parseInt`, `parseFloat`, `exit`, `byteLen`, `Math`, `String`, `Array`, `Infinity`,
  `NaN` and `anytype`.
- **`languages/lib/runtime.ll`** (22,350 lines, 199 `define`s, 8 `declare`s) - the
  checked-in output of our own C compiler over that file, regenerated and verified by
  **`tests/gen-runtime-ll.sh`** (`--check` diffs instead of writing). `llvm.BuildExecutable`
  hands its runtime inputs straight to clang, so passing the `.c` would have **clang**
  compile the floor; passing the `.ll` keeps every layer compiled by this repository.
- **`metajs-to-llvm-ir.abnf`** - two additions, both **in the goja-driver tail**, so
  `abnf/jsbootstrap.ll` regenerates **byte-identically** (verified; only
  `abnf/jsagrammar.go` changes, as it embeds the start script):
  - `emitDispatch()` emits `jsdispatch(idx, env, args)`, the function table a linked
    binary needs. `js_closure` stores a raw function *index*, and under `llvm.Run` the Go
    runtime resolves it by looking `jsf_<idx>` up in the module it is attached to - a
    binary has no such reflection. Emitted **only for `-exe`**, so every other run of the
    grammar produces the module it always did, byte for byte.
  - the runtime is `lib/runtime.ll` located **relative to the grammar** (`moduleName()`),
    so the build does not depend on the working directory. An explicit **`-rt` replaces**
    it rather than adding to it, which is what keeps `tests/metajs-link-stubrt.c` able to
    stand in for the whole runtime in the phase-2 link tests - those four matrix entries
    are unchanged.
- **`tests/metajs-test-full.js`** - the MetaJS full-syntax ratchet, 21 sections, **252
  assertions**. It is read three ways: `--full` runs it through both grammars, and
  `tests/clang-check.sh` builds and RUNS it as a native binary.

### Ground truth

```
matrix                318/318   (315 + three -exe entries for the native path)
--full                5,234 assertions, 0 languages whose halves disagree
                        (metajs 252 -> 298, c 444 -> 484: the new ratchet sections)
--cross               119 programs, 0 divergent
clang-check           16/16, and metajs is now in the RUN-IT row:
                        metajs   8809   ok, and the clang executable agrees
jsbootstrap.ll        regenerates byte-identically from source
tests/gen-runtime-ll.sh --check   runtime.ll is up to date (22350 lines)
```

Program output, `llvm.Run` (Go runtime) against the native binary (C floor):

```
metajs-test-full.js        SAME (rc=0)      metajs-test-2.js          SAME (rc=0)
metajs-test-features.js    SAME (rc=0)      metajs-test-try.js        SAME (rc=0)
metajs-test-1.js           SAME (rc=0)      metajs-test-multifile.js  SAME (rc=0)
                                            metajs-link-hello.js      SAME (rc=7)
```

An **841-cell differential probe** - `==` `!=` `===` `!==` `<` `>` `<=` `>=` and `+`
over 29 values (integers, halves, NaN, ±Inf, strings, `" 7 "`, booleans, null, undefined,
arrays, objects, a function) plus `typeof`, `!`, unary `-`, `*1`, 200 shift/mask cases and
`charAt`/`substring`/`slice`/`indexOf`/`charCodeAt` over out-of-range indexes - is
**byte-identical** between the two engines.

Five of the six runtime diagnostics are byte-identical too, and the sixth differs only in
a trace the C floor does not keep:

```
go : js runtime error: call of a non function value: 1 (last member lookups: )
nat: js runtime error: call of a non function value: 1
```

**Performance (Risk 1, measured here rather than at phase 4).** A loop/recursion/array/
string benchmark (3M-iteration arithmetic loop, `fib(26)`, 20k array pushes, 2k string
concatenations):

```
mec compile + link only            0.17s
mec compile + llvm.Run (Go)        1.85s   -> ~1.68s of execution
native binary (C floor)            1.81s
```

Within noise of each other. The C floor is **not** the slowdown the plan feared; the
soft float and the linear `jsdispatch` chain cost about what the Go runtime's handle
table and interface dispatch cost.

### Where the C floor and the Go runtime genuinely differ - measured, not hidden

**Nowhere, as of 2026-08-02.** Every item that stood here has been closed; what follows
is what each of them was and what closed it. The measurements are differential probes,
`llvm.Run` (the Go runtime) against the native binary, and for the C layer the real
`cc` on this machine.

```
probe                                                       cases      divergent
c, random doubles: + - * / and (double)long                 25,680          0
c, ties/cancellation/wide exponents/comparisons             61,843          0
c, infinities, NaNs, subnormals, signed zeros                3,200          0
metajs, literals + - * / sqrt, extreme exponents             55,456         0
metajs, the modelled domain, 500 sqrt and division cases     37,078         0
metajs, labelled operand pairs                               35,594         0
metajs, Infinity/NaN/signed zero over every operator          4,200         0
metajs, Math.pow over 50 bases and 50 exponents               2,500         0
metajs, toUpperCase/toLowerCase, EVERY code point            66,026         0
```

1. **The last bit of an inexact float - FIXED.** `c-to-llvm-ir.abnf` truncated the
   mantissa where IEEE-754 rounds to nearest even, so `0.1 + 0.2` was one ulp low and,
   printed, sat *below* `0.3`. Every helper now computes in an EXTENDED mantissa field -
   the 53 bits shifted left by three, so bits 2..0 are guard/round/sticky - with a
   separate flag for whatever falls below even those, and rounds exactly once at the
   end (`roundPack`). The three pieces that make it correct rather than merely closer:
   the multiply keeps the whole 106-bit product instead of the top 53; the divide takes
   58 quotient bits and treats a non-zero remainder as sticky; and addition and
   subtraction arrange BOTH branches to leave a positive residue below the computed
   value, which is what lets one sticky flag serve a sum and a difference (`a - (b+f)`
   is `(a-b-1) + (1-f)`, `(b+f) - a` is `(b-a) + f`).
2. **`(double)` of an integer above 2^53 - FIXED, and it was a wrong ANSWER.** The old
   `i2d` only ever shifted the magnitude UP, so anything that needed shifting down came
   out halved. It now shifts down into the extended field first, keeping the lost bits
   as sticky, and rounds.
3. **Overflow, underflow, infinities, NaNs, subnormals, signed zero - all now
   modelled.** The exponent used to wrap, so a product that should be `+Inf` came back
   as a finite `1.12e+307` and one that should underflow came back enormous. Overflow
   now saturates and underflow is gradual (`roundPack`); a subnormal operand is
   NORMALIZED in `unpack` (reported raw it overflowed the division accumulator and left
   the product's exponent wrong) which makes every exponent comparison signed; and
   `+-*/` and the comparisons carry the IEEE special-value rules, including a NaN
   coming through with its own payload and being unordered against everything. `x - y`
   became its own helper `__mec_dsub` so a NaN second operand reaches the propagation
   with its sign intact.
4. **`1.0 / 0.0` HUNG THE COMPILER.** The constant folder is the host's exact doubles,
   so it folded that to an infinity, and `dblBits` - which finds an exponent by halving
   until the value drops below 2 - never terminated. Found while porting `math.Pow`,
   which spells its infinities exactly that way.
5. **The number FORMATTER - rewritten exactly.** `runtime.c` generated digits by
   repeatedly multiplying the remainder by ten in floating point, which is right to
   about sixteen digits and therefore wrong in the last place of every seventeen-digit
   number: `10/3` printed `3.3333333333333334`. A double is a 53-bit integer times a
   power of two, so its decimal expansion terminates and every question about it has an
   exact integer answer; `shortest_digits` now works on decimal digit arrays. The
   shortest-form search is an exact integer comparison against the rounding interval
   (`4N` against `4*M*10^(L-k)`, bounded by `2P` and, below a power of two, `P`), and
   the digits are rounded to nearest with TIES TO EVEN, which is what makes
   `1000000000000000.25` render as `...0.2` like Go and not `...0.3`.
6. **The number PARSER - rewritten exactly.** `pow10` built its powers of ten by
   repeated multiplication, one rounding per step, so `1.7976931348623157e308` parsed
   to `+Inf` and `1e100` was a part in 10^16 too large; and the mantissa was
   accumulated into a `long` that stops being exact at 2^53, so
   `parseFloat("0.9999999999999999")` came back as exactly 1. `dec_to_double` now
   reaches the answer without floating point at all: the mantissa is `T * 2^-e`, and
   `2^-e` is `5^e / 10^e` or `2^|e|`, so it is always a decimal bignum divided by a
   power of ten - which for a digit array is reading it from a different offset. The
   fast path (at most 15 digits, |exponent| at most 22) is kept because it is provably
   exact, not merely close.
7. **Integral doubles above 2^53 printed all their digits.** `num_to_str` took the
   exact-integer path for anything inside the `long` range, so `1234567891234567936`
   printed in full where every JS engine prints `1234567891234568000`. The fast path is
   now bounded by 2^53, where an integer IS its own shortest form.
8. **`Math.sqrt` - now correctly rounded, and it no longer diverges.** Newton's
   iteration started from the VALUE ITSELF, which needs one step per power of two:
   eighty were not enough for `sqrt(1e100)`, which answered `8.27e+75`. The start is now
   the exponent halved by a bit-pattern shift, which lands within a few percent. Which
   of the two neighbouring doubles is actually nearest is then decided EXACTLY, by
   comparing squares as whole numbers - a 53-bit mantissa squared is 106 bits, so the
   comparison runs on the same decimal digit arrays the printer uses.
9. **`Math.pow` with a non-integer exponent - now answered.** It used to be NaN.
   Repeated multiplication cannot reach a non-integer exponent, and an approximation of
   exp/log would have disagreed with the Go twin in the last bits, which is the whole
   defect class this floor exists to avoid. `runtime.c` therefore carries a FAITHFUL
   PORT of Go's own `math.Pow`, `math.Exp`, `math.Log`, `math.Frexp`, `math.Ldexp` and
   `math.Modf` - the same constants, the same polynomials, the same order of operations.
   The same sequence of operations gives the same bits now that the arithmetic under it
   is correctly rounded, and 2,500 differential cases say it does.
10. **`toUpperCase`/`toLowerCase` were ASCII-ONLY.** The Go twin is `strings.ToUpper`,
    the full Unicode simple mapping - 2,321 code points on which the two engines
    answered differently. `runtime.c` now carries that mapping as 328 ranges (mode 0 is
    two deltas for a whole range, mode 1 the alternating pairing of the Latin, Greek and
    Cyrillic extension blocks), GENERATED FROM THE GO RUNTIME'S OWN ANSWERS over every
    code point and verified against them: 65,505 BMP and 521 astral rows, zero
    divergent. It maps over code points rather than bytes, and it reproduces one lossy
    Go behaviour deliberately - an UNPAIRED surrogate becomes three U+FFFD, because
    that is what `strings.ToUpper` does to the three invalid UTF-8 bytes it is carried
    in, and the two engines have to agree.
11. **The host global sets of the two MetaJS halves still differ.**
    `metajs-interpreter.abnf` binds exactly `println print printf sprintf eprintln
    parseInt parseFloat Math String exit anytype`; the compiler half
    (`standardJSBindings`) also has `Infinity NaN Array Object byteLen sprint rawSet`.
    The C floor matches the **compiler** half. `tests/metajs-test-full.js` asserts only
    the intersection, because a check on the difference would fail on one half by
    construction. Pre-existing drift, and the ONE item on this list still open. The
    sharp s is the same kind of thing in miniature: `"\u00df".toUpperCase()` is `"SS"`
    in the interpreter half (real JS, a full mapping) and `"\u00df"` in the compiler
    half (Go, a simple mapping), so section 07 asserts around it and says so.

### What now pins all of this

`./test.sh` compares goja against `-frozen` and BOTH of them are the Go runtime, so it
is structurally blind to a wrong answer in the C floor. The two ratchet files carry the
assertions instead, and `tests/clang-check.sh` runs them natively:

- **`tests/metajs-test-full.js` section 22**, 38 assertions on inexact arithmetic,
  `sqrt`, `pow`, accumulation, integer-to-double above 2^53 and the decimal parse, plus
  8 case-mapping assertions in section 07. Against the pre-change native binary, 22 of
  the 28 the section started with FAIL; against the Go runtime and the new native binary
  all of them pass.
- **`tests/c-test-full.c` sections 46 and 47**, 40 assertions, all validated against
  real `cc`. Section 47's operands come out of ARRAYS on purpose: written as literals
  they are folded by the grammar's own constant folder, which uses the host's exact
  doubles and is therefore right whatever the emitted soft float does - a section of
  literals passed at HEAD with the truncating soft float still in place. 19 of its 26
  fail against the pre-change grammar.

### The three pre-existing defects in c-to-llvm-ir.abnf - all FIXED

All three were reproduced from clean archives of `e81dc94` and `43cf14e`, so they
predate this work; all three were invisible to `tests/c-test-full.c`, which never used
the constructs, and all three are now asserted there (section 46) against `cc`.

```c
void ph(double d) { }                 /* a double PARAMETER did not compile:      */
                                      /* getFunc put an i64 in the signature and  */
                                      /* the prologue allocated an i32 slot       */

double gd(long x) { return 1.5; }     /* a double RETURN was silently wrong: only */
                                      /* a 64-bit INTEGER return was recorded, so */
                                      /* the function was declared i32 while      */
                                      /* makeReturn handed it an i64              */

long a; long *p = &a; *p = 42;        /* did not compile; p[0] = 42 did, because  */
                                      /* only the subscript path typed the address*/
```

The fixes: `noteRetCt` and `noteParamCts` record a floating-point type the same way they
record a 64-bit integer one; `kct` gives a floating-point type a 64-bit zero; the
parameter prologue allocates the width the signature promised; `emitCallArgs` converts an
integer argument to a double parameter (C11 6.5.2.2p7); and `storeTo` types the ADDRESS
to the cell (`cellAddr`) and the VALUE to its width - a value of the right C type can
still be sitting in a narrower register than its cell, and `sext i32 to i8` is IR clang
refuses outright.

`languages/lib/runtime.c` no longer needs the workaround the three forced on it, though
it still carries numbers as bit patterns in a `long` where that reads better.

### Exceptions

`js_throw`/`js_try` are `setjmp`/`longjmp` over a stack of jump buffers - the returned
control-signal protocol of `lib/compile-core.js` handles `return`/`break`/`continue`,
but a *throw* still has to unwind C frames, and the emitted IR does not test after a
call. `setjmp` through our own C compiler, linked by clang, was verified in isolation
first (`A7`, same as `cc`). The `finally`-overrides-and-swallows rule of the Go twin is
implemented: a control signal returned by the finally closure clears a pending throw.
A throw with no enclosing `js_try` prints `js runtime error: uncaught exception: <v>` on
stderr and exits 1, which is what the Go side does at the program boundary.

### Not done

- **`-rt-prims` and the C floor cannot be combined in one `-exe` build.** With
  `-rt-prims` the module *defines* the 13 derived primitives and `runtime.ll` defines
  them too, so the link would fail with duplicate symbols. `-rt-prims` is a measurement
  flag with no `-exe` entry in the matrix, so nothing regressed; splitting the derived
  13 into a second `.ll` would fix it when it is wanted.
- **No GC.** The arena leaks by construction. A long-running MetaJS program will grow
  without bound; every test program here allocates a few megabytes at most.
- **`jsdispatch` is a linear compare chain.** O(number of functions) per call. It did not
  show up in the benchmark above, but a program with thousands of functions would want a
  binary search or a real table.

### Original plan text, for reference

- **Value representation:** `i64` handle = tagged pointer. The emitted IR is already
  all-`i64` (`js_str_mem(i8*, i64) -> i64` is the only exception), so nothing in any
  emitter assumes Go.
- **Allocation:** arena bump-allocator, the approach already proven in the bash and
  batch native runtimes. No GC initially - see Risks.
- **Globals:** `println` is not an extern; it resolves through `js_scope_get` against a
  global scope the runtime pre-seeds (`abnf/jsrt.go:8231`). The C floor must seed it too.

**Gate:** `tests/metajs-test-full.*` runs as a native binary and agrees byte-for-byte
with `llvm.Run`. Add metajs to clang-check's run-natively row. **Met.**

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
