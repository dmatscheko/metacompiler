# What is left: GC, the open defects, and the remaining eleven languages

Companion to [runtime-rework-plan.md](runtime-rework-plan.md), which is the record of
what was BUILT (phases 0-4a, bash/batch convergence, one linker). This document is the
forward plan. HEAD when it was written: `e8bf2c3`.

## Where things stand

```
matrix 325/325 · --full 5,518 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16 - bash, batch, c, lua, metajs all "the clang executable agrees"
```

Five languages produce self-contained native binaries. Two of them (metajs, lua) got
there through the GENERAL architecture - C floor, MetaJS layer 2, language grammar -
which is what makes the remaining eleven a rollout rather than eleven rewrites.

Still on the Go runtime: **29,670 lines across `abnf/jsrt*.go`**, backing eleven
languages.

---

# Part 1 - Memory, which gates everything else

**CLOSED 2026-08-03.** Allocation was down 5.0x since phase 4a (18,072 -> 3,616 bytes
per loop iteration, part 1a below) and still exactly linear, which is a program that runs
for a minute dying whatever the constant is. Part 1b replaced the arena with a mark/sweep
heap and the line is now **flat: 3 MB at 50,000 iterations and 3 MB at 1,000,000**. The
eleven-language rollout no longer multiplies an unbounded-memory runtime by eleven.

The residual, per iteration of `s = s + i % 7`, is live data rather than waste. The
figures 1a was written against, and what it left:

```
                       before 1a          after 1a
scope cells        37   2,368 B     14      896 B
buffers            34   2,368 B     19    1,504 B   scope/array/object storage
number cells       26   1,664 B     14      896 B
array cells        12     768 B      5      320 B   the argument array of every call
object cells        1      64 B      0        0 B
                       7,232 B           3,616 B
```

Two directions. They are not alternatives: the first reduces garbage, the second
reclaims it. **Do them in this order** - every allocation removed by step 1 is one the
collector never has to trace, and step 1 is also where the remaining ~4.8x of TIME is.

## 1a. Stop generating the garbage (emitter, no GC needed) - DONE 2026-08-03

**Allocation per iteration of `s = s + i % 7` is halved: 7,232 -> 3,616 bytes, exactly
-50.0%.** Four emitter changes, no floor change - `lib/runtime.c` and `lib/runtime.ll`
are byte-identical to `f19a8ad`, and `tests/gen-runtime-ll.sh --check` says so.

### The measurement

Counters compiled into a SCRATCH copy of `runtime.c` (`mk_scope`, `mk_arr`,
`mk_num_raw`, `mk_obj`, `buf_new`, `ar_alloc`), differenced between 100,000 and 101,000
iterations so the figure is the STEADY state - at 100 iterations the loop counter is
still sweeping fresh values into the small-integer cache and the number line reads
20.4 instead of 26. `ar_alloc` rounds to 16, so a 56-byte cell costs 64.

```
                     f19a8ad              HEAD+           per-line
scope cells      37   2,368 B      14      896 B          -62%
buffers          34   2,368 B      19    1,504 B          -36%
number cells     26   1,664 B      14      896 B          -46%
array cells      12     768 B       5      320 B          -58%
object cells      1      64 B       0        0 B         -100%
arena calls     110                 52
arena bytes          7,232 B           3,616 B            -50.0%
```

`tests/bench-alloc.sh`, which reads the same number off peak RSS, agrees within 0.5%:

```
                 f19a8ad                       HEAD+
iters=50000      bytes/iter=7283               bytes/iter=3652
iters=100000     bytes/iter=7272               bytes/iter=3641
iters=200000     bytes/iter=7266               bytes/iter=3636
iters=400000     bytes/iter=7263               bytes/iter=3633
```

### Time, from clean archives built and run in their own trees

`git archive f19a8ad | tar x` in one directory, the working tree in the other, `go
build` inside each. Three runs of `/usr/bin/time -p`, user seconds; every program
printed the right answer and exited 0 (a crashing binary looks fast).

```
                            f19a8ad            HEAD+          RSS
lua  s = s + i%7, 2M      4.28 4.38 4.37     2.83 2.89 2.87   13,849 -> 6,925 MB
lua  fib(26)              0.38 0.39 0.39     0.35 0.35 0.36    1,157 ->   929 MB
metajs try/catch, 2M      1.98 2.02 2.01     1.92 1.99 1.96    2,453 -> 1,881 MB
```

The Lua loop is **-34% wall clock and -50% memory**. `fib(26)` is -8%: it is call and
scope bound, and its calls are the ones that really do declare. The try/catch benchmark
is inside the noise on time and -23% on memory - it allocates for reasons this phase did
not touch (a fresh closure frame per try body).

The **Go** half improves too, since it is the same emitted IR: `llvm.Run` on the 200,000
iteration loop went 0.51s / 198 MB -> 0.27s / 34 MB.

### What changed

1. **`makeBlockStmt` no longer opens a scope for a block that binds nothing**
   (`lib/compile-core.js`, so all fifteen compilers). The stated difficulty - the thunk
   is emitted before you know - is solved with a **one-entry phi**: the scope creation
   gets a block of its own that stays empty and unterminated while the body is emitted
   against a phi reading from it, and the phi's incoming value is written afterwards.
   Everything is an APPEND (the header's terminator goes on last, after the
   `js_scope_new`), so no instruction is moved or removed. `phi.Incs` is assignable from
   a tag script under BOTH engines - verified before the change was written.

   Whether the block bound anything is a counter, `declCount`, bumped by every emitted
   call to a scope-writing extern. The set is the scope-taking externs MINUS the eight
   proven read-only (`js_scope_get`, `js_scope_typeof`, `js_kget`, `js_ktget`,
   `js_scope_set` and `js_tset` - which walk the chain and DIE if the name is absent -
   `js_scope_new`, `js_closure`), so a name nobody listed is counted as declaring, which
   is the safe direction. The soundness argument is one line: **a scope nothing was
   bound in is transparent to lookups**, so capturing it and capturing its parent are
   the same thing. `declCount` is restored when a block or a function ends, because what
   they bound went into THEIR scope.

   `metajs-to-llvm-ir.abnf` replaces `callExt` (for `-rt-prims` routing), so it calls
   `noteExt` itself; a grammar that replaces `callExt` and forgets that would lose a
   scope. Grammars that override `makeBlockStmt` (c, csharp, go) keep the old behaviour
   and simply do not get the win.

   Effect on the benchmark: 21 -> 14 scope cells. Deleting block scopes ENTIRELY (wrong,
   but it measures the ceiling) gives 5, so the 9 blocks that remain really do declare.

2. **A numeric literal outside the floor's small-integer cache is boxed once, not once
   per execution** (`lib/compile-core.js`, `emitNum`). Each distinct constant gets a
   memoizing getter function of its own - one global holding the handle, filled on first
   use, handle 0 as the empty marker (never a number, it is `undefined`). A FUNCTION
   rather than an inline branch because `emitNum` answers a value, not a `{b, v}`, and a
   hundred call sites cannot take a block split. `-256..1024` stays a direct `js_num_i`:
   the floor already answers those without allocating.

   This is what the "widen the small-integer cache" item turned into once the
   distribution was actually measured - see below. Effect: 26 -> 14 number cells.

3. **Lua's numeric `for` stopped throwing away half of its work**
   (`lua-to-llvm-ir.abnf`, `makeNumFor`). The loop test was
   `select(stepPos, lua_cmp(<=), lua_cmp(>=))` - and a `select` evaluates both operands,
   so every iteration made two layer-2 calls and discarded one, twice over (the head
   test and the wrap test). Both are branches now, with a phi. The sign of the step is
   also loop INVARIANT (nothing can reassign `$fs`, the name is unspellable in Lua) and
   was being recomputed twice per iteration through `lua_num`; it is hoisted into the
   preheader. Four layer-2 calls per iteration removed out of eleven.

4. **A one-target, one-value assignment skips the positional array**
   (`lua-to-llvm-ir.abnf`, `makeAssign` and `makeLocal`). `lua_first(v)` IS
   `js_get(appendall([], v), 0)` for every `v`, the empty multi-value included - both
   answer nil. The array cost an arena cell and a four-slot buffer inside every loop body
   that assigns.

Cumulative, in order (arena bytes per iteration at i≈100, the figure each step was
measured against): 6,876 -> 4,892 (3) -> 4,475 (4) -> 4,027 (1) -> 3,616 (2).

### The small-integer cache: measured, and NOT widened

The item said to check the distribution before assuming. The 26 number cells of a
baseline iteration are:

```
already in the -256..1024 cache      0.00     (nothing missed it)
integral, |v| <= 65536               1.14
integral, |v| <  2^31                6.00
integral, |v| >= 2^31               18.00     min -2147483649  max 9007199254740992
non-integral                         0.86
```

**Eighteen of the twenty-six were not near the window at all** - they are the guard
constants of layer 2's own fast path, `lv > -2147483649 && lv < 2147483648` and the
`+-9007199254740992` product test in `js_luarith`, re-boxed on every call. No cache of
any plausible width would have caught them; change 2 did (18 -> 6). What a wider cache
could still reach is the 1.14 cells per iteration in the 1,025..65,536 band, and those
are the genuine live values of `i` and `s`: **at most 73 bytes of 3,616, or 2%, in
exchange for a 512 KB table in the floor**. Not worth a floor change, and none was made.

### The item that was NOT done, and why

**Inlining the layer-2 arithmetic fast path into the emitted IR was ranked first and is
not implemented.** The reason is a constraint the ranking did not account for: the
emitted IR runs in TWO worlds - natively against the C floor, where a handle is a
`Cell *`, and under `llvm.Run` against `abnf/jsrt.go`, where it is an index into a Go
table - and they must stay byte-identical. So an inlined fast path may not touch the
representation; it can only be spelled in externs, and it would need `js_typeof`,
`js_seq` and four range compares against `+-2^31` per operand. It is still likely a win
now that change 2 makes those bounds free, but it is a real piece of work in the Lua
emitter rather than the cheap move it looked like, and it should be costed against
1b rather than assumed.

What change 3 shows is that the CALLS were half the problem anyway: four of the eleven
layer-2 calls per iteration were pure waste from a `select` and a loop-invariant read,
and removing them was 29% of the total on its own.

### Gate and verification (at the working tree, `f19a8ad` + this change)

```
./test.sh              matrix 325/325, all green
./test.sh --full       5,504 assertions, 0 languages whose halves disagree
                       (grep 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF': no hits)
./test.sh --cross      119 programs compared, 0 divergent
tests/clang-check.sh   16/16; bash, batch, c, lua, metajs all
                       "ok, and the clang executable agrees"
go test ./abnf/        ok
gen-runtime-ll.sh --check   runtime.ll is up to date (40311 lines)  <- floor untouched
gen-bash-rt-ll.sh --check   up to date        gen-batch-rt-ll.sh --check  up to date
gen-lua-rt-ll.sh  --check   up to date (14314 lines, regenerated)
```

`abnf/jsbootstrap.ll` and `abnf/jsagrammar.go` are **regenerated, not byte-identical** -
and that is correct rather than a miss: the snapshot inlines `lib/compile-core.js`, so a
change to the emitter MUST reach it or the frozen half quietly keeps compiling tag
scripts with the old one. `mec -freeze languages/metajs-to-llvm-ir.abnf` is a fixed
point afterwards (same MD5 on a re-freeze with the rebuilt binary), and every suite
above was re-run against the new snapshot.

One trap found on the way, worth the next person's time: `-rt-lib` renumbers
`funcCount`/`strCount` to 1,000,000 so a library module's symbols cannot collide with
the program module's. The new `jsnum.` / `jsnumg.` globals are module-level symbols too
and needed the same treatment - without it `clang` failed with `duplicate symbol
'_jsnum.1'`, which is a LINK error and therefore loud. Anything else that ever gets a
generated module-level name needs the same line.

### What is left on this line

The 3,616 bytes still split 42% buffers (14 scope buffers of 96 B + 5 array buffers of
32 B), 25% scope cells, 25% number cells, 9% arrays. Every scope that exists now
declares something, so the buffer line and the scope line are the same 14 scopes twice.

**One thing tried and measured at zero, so it is not in the change.** Every MetaJS
function declares `arguments` whether or not the body mentions it, which looks like a
free 96 bytes per frame. It is not: a scope buffer starts at capacity 4 and holds three
parallel arrays, so `js_luarith`'s three parameters plus `arguments` fit in exactly the
same one allocation that three alone would take. Deleting the `arguments` declaration
outright (wrong, but it bounds the win) leaves the benchmark at **3,616.00 bytes per
iteration - identical to the digit**. It would only pay for a function with 4 or more
parameters, where the extra entry forces the doubling to capacity 8.

The honest next targets are therefore the 14 scopes themselves - which means the
argument-array-and-frame cost of a MetaJS CALL, i.e. direct IR calls with i64 parameters
for a known top-level callee instead of `js_arr_new` + `js_call` + a scope of
`js_tdecl`s - and the inlined arithmetic fast path costed above. Both are larger than
anything in this phase.

## 1b. Mark/sweep over the arena - DONE 2026-08-03

**Memory is bounded.** The `s = s + i % 7` Lua loop's resident set no longer depends on
the iteration count at all:

```
                MEC_GC=off (the old arena)        the collector on
iters=50000     rss   228,720,640   4,574 B/it    rss  3,162,112    63 B/it
iters=100000    rss   456,507,392   4,565 B/it    rss  3,194,880    31 B/it
iters=200000    rss   911,949,824   4,559 B/it    rss  3,194,880    15 B/it
iters=400000    rss 1,823,031,296   4,557 B/it    rss  3,178,496     7 B/it
iters=1000000   rss 4,556,144,640   4,556 B/it    rss  3,194,880     3 B/it
```

Read the right-hand column as a constant, not as a rate: **3.1 MB flat**, against a line
that was still perfectly straight after part 1a had cut its slope by 5x. Same binary in
both columns - `MEC_GC=off` is the switch, so this is one measurement, not two builds.

**Refcounting is the wrong shape here and this is settled, not a preference.** A scope
holds its parent, and a closure holds its defining scope - so a closure defined inside a
function cycles with the scope that names it, and that is the *common* case, not an edge
case. Refcounting would leak exactly the programs people write.

### The shadow stack was costed and NOT built, and that is the main design decision

The plan said "shadow stack first". It is not there, and the reason is a constraint the
plan states one paragraph further down: **the emitted IR runs in two worlds** - natively
a handle is a `Cell *`, under `llvm.Run` it is an index into a Go table - and the two
must stay byte-identical. Every push and pop a shadow stack emits is IR that the Go half
has to carry and ignore, at every expression temporary, in fifteen emitters, forever.

**Scanning the real C stack conservatively needs no emitted instruction at all**, so the
compiler half is untouched *by construction* - which is the strongest available form of
the invariant that matters most here ("a GC that changes an answer is worse than no
GC"). The soundness argument is the ordinary ABI one, the same one Boehm's collector
rests on: a handle live across a call is either spilled into the caller's frame, which
is inside the scanned range, or held in a callee-saved register, which the collector
captures with a `setjmp` of its own. Handles are i64 INTEGERS as far as LLVM is
concerned, never pointers, so no pointer-provenance optimisation can rewrite one into a
form the scan cannot see.

The only IR that changed is **one extern, `js_gc_pin`, at three call sites**, and it is
the IDENTITY in both halves. It exists because the emitted module has globals of its own
that hold a handle for the life of the program and that the C floor cannot see:
`lib/compile-core.js`'s `jsnumg.N` (part 1a's memoised literal boxing) and
`metajs-to-llvm-ir.abnf`'s `jsrtlib_env` / `jsrtlib_f_*` (a `-rt-lib` library's scope and
its resolved functions). `abnf/jsrt.go` binds it to `func(a) { return a[0] }`.

### What was built (`languages/lib/runtime.c`)

The bump arena is now a mark/sweep heap. Every allocation is a **block**: a 16 byte
header (`size`, then flags = free bit, cell bit, and the **mark epoch** in bits 8 and up)
and a 16 byte aligned payload. A cell's handle is its payload address, so nothing outside
the allocator ever sees a header. A chunk is a malloc'd slab carrying a **start bitmap**,
one bit per 16 bytes, set when a block is carved; blocks are never moved, never
coalesced and never split after the fact, so a bit once set stays true and the bitmap
never has to be rebuilt. Freed blocks go on **segregated free lists** by size class
(`size/16`, classes 2..64) plus a first-fit list for anything larger. The mark epoch is
why a collection never walks the heap to *clear* marks - it just bumps a counter.

Roots are: the file's own globals (`G_ROOT`, `THROWN`, `RETSLOT`, `NIC`, `SMC_VAL`), the
`jmp_buf` pool `JB`, the pinned module globals, the callee-saved registers, and the C
stack from the collector's own frame up to an anchor `main` records. Marking is
**precise from a cell** (the tag says which of `a..f` is a handle and which is a raw long
- a number's IEEE bits, a closure's function index, a host id, a bound method's method
id) and **conservative from a raw buffer** reached from the stack, whose shape is not
knowable there. Every marking step goes through the validating `gc_try`, even for a field
the tag calls a handle: a `_raw` operator index reaching a traced slot by mistake would
otherwise write a mark word into unrelated memory, and a collector whose failure mode is
silent corruption is not worth having.

**The trigger is an explicit, testable knob**, which is what made the two defects below
findable:

```
MEC_GC=off      never collect - this IS the old arena, so RSS/iterations is still
                exactly the allocation rate (that is the left column above)
(default)       collect when the bytes allocated since the last collection exceed
                the live set, floor 1 MB
MEC_GC=stress   collect at EVERY allocation
MEC_GC=poison   collect on the ordinary threshold, but RETIRE every swept block and
                fill it with a non-value instead of reusing it, so a root the mark
                pass missed is a wrong answer rather than memory that happens to
                still hold the right bytes
MEC_GC_STATS=1  report collections / live / heap / pinned on stderr at exit
```

`poison` is deliberately *not* `stress` as well: retiring means the heap only grows, and
collecting per allocation on top of that is quadratic - it did not finish on the Lua
ratchet in eight minutes, against 1.1 s for the form that ships.

### TWO defects the knob found, one of them a design error

**1. Requiring an EXACT payload address is unsound at -O2. This is the important one.**
The first version accepted a conservative root only when the word was exactly a block's
payload address, on the argument that a buffer's owning cell is always reachable too and
marks the buffer precisely. **That argument is false, and the native Lua suite said so.**
In `u_slice` the only surviving reference while `str_from_units` allocates is `u + b`, an
INTERIOR pointer into the UTF-16 unit buffer: clang had already dropped the string handle
`h` after reading `fc(h)` out of it, so the cell was unreachable, the collector freed it,
and the unit buffer went with it.

```
MEC_GC=stress, exact-pointers-only collector, clean build:
  tests/lua-test-features.lua   142 checks, 9 failures   (all string.sub)
  tests/lua-test-complete.lua                3 failures
  MEC_GC=off / auto / poison on the same binary:         0 failures
```

Interior pointers are accepted now, resolved through the start bitmap by scanning back to
the nearest set bit at or below the offset (with the exact-payload case checked first,
because that is what almost every candidate is). The general lesson is worth carrying:
**requiring exact pointers is a standing bet that no optimiser will ever keep only a
derived pointer, and that bet does not hold.** `auto` mode never hit it - only
`MEC_GC=stress` did, which is the argument for having the knob at all.

**2. A chunk's tail was abandoned when the bump pointer moved to a new chunk**, so one
large allocation could lose up to its own size of a perfectly good chunk, every time -
a leak again by another name. The tail is carved as a free block instead.

### The price, stated rather than buried: +25% allocation

The 16 byte header per block is not free. Per iteration of the Lua loop, with the
collector off so the figure is comparable to part 1a's:

```
                       8c396c4      HEAD
bytes/iter (400,000)     3,633      4,557      +25.4%
```

That is the header, essentially exactly: part 1a counted 52 allocations an iteration, and
52 x 16 = 832, so 3,633 + 832 = 4,465 against 4,557 measured - the remaining 92 being the
rounding of a 56 byte cell to 64 and then, with a header in front of it, to 80. It buys a
walkable heap, which is what a sweep needs.

**The obvious next lever is that cells are all one size**: a cell-only region with the
size implicit and the mark bits in a side array would give the header back, and it is a
larger change than this one. Not done, and recorded here rather than assumed away.

### Discriminating power of the new ratchet sections, measured

`tests/metajs-test-full.js` SECTION 25 (7 assertions, metajs 397 -> 404) and
`tests/lua-test-full.lua` SECTION 25 (7 assertions, lua 310 -> 317) are the shapes that
show a collector freeing live data: data held across a call that allocates heavily, a
closure keeping the scope that defines it, a value thrown across twenty frames, a
repeatedly reallocated array and object, a mutated upvalue, and - the one the C stack
scan exists for - an argument temporary live in a machine register while a LATER argument
runs a few thousand allocations.

They discriminate **only in the native binary**: the other two engines are collected by
Go and by the host JS engine, so there they are ordinary correctness checks. Breaking the
collector one way at a time, in a clean copy of the tree, both ratchets built with `-exe`
and run under `MEC_GC=stress` (ABORT = the binary died or the run never reached its own
summary; rc 139 is a segfault). Re-measured against the collector as it SHIPS, after the
interior-pointer fix above - the earlier run against the exact-pointers-only version gave
the same table but for two rows where lua segfaulted instead of dying on its own check:

```
                                              metajs (404)        lua (317)
the C stack not scanned                    ABORT (segfault)   ABORT
the callee-saved registers not scanned     ABORT              ABORT
js_gc_pin's roots not scanned              ABORT              ABORT
a scope's PARENT not traced                ABORT (segfault)   ABORT
a closure's ENV not traced                 ABORT (segfault)   ABORT
NIC / SMC_VAL (the two caches) not roots   ABORT              ABORT
an object's VALUE buffer not traced        ABORT              ABORT (segfault)
an array's element buffer not traced       ABORT (segfault)   ABORT
a raw buffer's contents not scanned        ABORT              ABORT
the JB jmp_buf pool not scanned            ABORT               0 of 317
a string's UTF-16 UNIT buffer not traced     15 of 404         0 of 317
G_ROOT not an explicit root                   0 of 404         0 of 317
THROWN not an explicit root                   0 of 404         0 of 317
```

**The last two are honest zeros and they are stated, not hidden.** Neither is dead code:
`G_ROOT` survives every current program because `scope_of` reads it constantly, so it is
always live in some frame the conservative scan sees, and `THROWN` because `js_try`'s C
frame holds the same handle across the whole unwind. Both are kept because a root set
that depends on a register allocator's habits is not a root set. The `JB` row is the
mirror image and is why it is there at all: Lua cannot tell (its ratchet has no
exceptions - `error`/`pcall` are out of scope for that file), and MetaJS dies outright.

### Every existing suite, and the native GC sweep

Beyond the standing suites, every Lua and MetaJS program in `tests/` was built as a
NATIVE binary and run in all four collector modes, with stdout, stderr and exit status
compared byte for byte against `MEC_GC=off`:

```
native GC sweep: 16 programs built, 16 identical across off/auto/stress/poison, 0 divergent
```

and both full ratchets, whose four modes agree byte for byte on stdout, stderr and exit
status (`MEC_GC_STATS=1` in the middle column, so "it ran" is a measurement):

```
                        off        auto      stress    poison    collections (auto)
tests/metajs-test-full 0.43s      0.01s      4.41s     0.01s      7
tests/lua-test-full    0.38s      0.05s     76.39s     0.94s
```

The `off` column being SLOWER than `auto` on both is not noise and is worth keeping:
with nothing reclaimed these two short files touch 8 MB and 5 MB of fresh pages, and the
page faults cost more than the collector does. `stress` is a collection per allocation
(129,072 of them for the MetaJS ratchet) and is a debugging mode, not a configuration.

### Time, from clean archives built and run in their own trees

`8c396c4` and this tree, each `go build` inside its own directory. Three runs of
`/usr/bin/time -p`; every program printed the right answer and exited 0.

```
                             user seconds              wall seconds       max RSS
                          8c396c4      HEAD        8c396c4     HEAD
lua  s = s + i%7, 2M    2.84 2.84 2.85  3.36 3.33 3.32   3.12  3.31   6,925 ->    3 MB
lua  fib(26)            0.38 0.36 0.35  0.41 0.42 0.41   0.39  0.42     928 ->    2 MB
metajs try/catch, 2M    1.99 1.98 1.96  2.12 2.11 2.10   2.48  2.56   1,880 ->    2 MB
```

**+17% / +14% / +7% of user time, against a resident set 2,300x / 460x / 940x smaller.**
Wall clock moves much less (+6% / +8% / +3%) because the arena's own page faults were
paying part of the difference already. The collector's work is the sweep, which walks
every block of the heap, plus the +25% of allocation the headers cost.

**The trigger threshold was tuned and the measurement is recorded rather than the
conclusion alone.** A collection is due when the bytes allocated since the last one
exceed the live set, with a floor; on the 2M Lua loop:

```
floor  1 MB   3.36 3.33 3.32 user    3 MB RSS    <- shipped
floor  4 MB   3.16 3.20 3.20 user    6 MB RSS
floor 16 MB   3.22 3.20 3.18 user   18 MB RSS
```

4 MB is 5% faster for twice the resident set, and 16 MB buys nothing beyond it. **1 MB
ships anyway**, and the reason is test coverage rather than memory: at 1 MB every short
program in the matrix actually enters the collector (the metajs ratchet collects 7 times
instead of 4), and 5% of a benchmark is worth less than that.

### Gate and verification

```
./test.sh              matrix 325/325, all green
./test.sh --full       5,518 assertions (5,504 + 7 metajs + 7 lua),
                       0 languages whose halves disagree
                       (grep 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF': no hits)
./test.sh --cross      119 programs compared, 0 divergent
tests/clang-check.sh   16/16; bash, batch, c, lua, metajs all
                       "ok, and the clang executable agrees"
go test ./abnf/        ok
gen-runtime-ll.sh   --check   up to date (42506 lines, regenerated)
gen-lua-rt-ll.sh    --check   up to date (14360 lines, regenerated)
gen-bash-rt-ll.sh   --check   up to date        gen-batch-rt-ll.sh --check  up to date
mec -freeze languages/metajs-to-llvm-ir.abnf    fixed point (same MD5 on a re-freeze
                       with the rebuilt binary); regenerated because the snapshot
                       INLINES lib/compile-core.js, which grew the js_gc_pin call
```

### What this does NOT do

- **It never moves an object**, and it cannot: conservative roots forbid it. So there is
  no compaction, and fragmentation is whatever the segregated free lists leave. Not a
  problem for this workload - every size in the benchmark is a cell or a small buffer -
  but it is a real limit for a program that alternates sizes.
- **It never returns memory to the OS.** A chunk, once malloc'd, is kept.
- **It is not incremental**: a collection is a stop-the-world mark and a full sweep. The
  sweep is what the 5% threshold measurement above is really measuring.
- **The other native runtimes are untouched.** `bash-rt.c`, `batch-rt.c` and the C
  interpreter's memory have their own arenas and no collector; only `runtime.c`, which
  is what MetaJS and Lua (and the eleven languages after them) stand on, got one.

---

# Part 2 - The open defects

Ordered by whether they block the rollout.

## Blocking the rollout

~~**Layer 2 cannot create a new primitive TYPE.**~~ **DONE 2026-08-02 - tag 13.**
`languages/lib/runtime.c` now has a real sized-integer tag, the twin of
`abnf/jsrtint.go`'s `jsGInt`, which stays the authoritative specification: `js_typeof`
answers `"number"` for it in **both** halves. It is language neutral by construction -
the whole `js_gi*` extern family (`js_giarith js_gicmp js_gieq js_giconv js_gineg
js_ginot js_ginum js_gistr js_giis js_gival js_gilt js_gile js_gigt js_gige js_giadd
js_gilit`) is implemented, so go/java/kotlin/csharp can be linked natively without
touching the floor again.

```
cell.a  the 64 bit value, already truncated to the width and sign extended
cell.b  the width: 8, 16, 32 or 64        cell.c  1 when unsigned
```

**Ground truth.** A 97,664-line differential probe - `giNorm` over 8 width/signedness
combinations x 32 seeds, then 11 operators x 32 x 32 per combination, plus `giCmp`,
`giEq`, unary minus, `^x`, `giFloat`, and `giConv` between every pair of widths - is
**byte-identical** between the C floor and a Go oracle compiled straight out of
`jsrtint.go`. A second 240-case probe covers the float -> sized-integer conversion,
`+-Inf` and NaN included, and also agrees exactly.

**Two places it cannot match, stated rather than approximated:**

- `giFromFloat` ends in `int64(m)` on a value that is NaN when the operand is an
  infinity, and Go leaves that conversion *implementation defined*. **Re-measured
  2026-08-03 on both architectures**, by building the expression out of `jsrtint.go`
  twice: `GOARCH=arm64` answers `0`, `GOARCH=amd64` (same machine, under Rosetta)
  answers `-9223372036854775808`. There are now THREE implementations, not two, and
  the odd one out is the Go twin: the C floor's `si_from_float` returns `0` from an
  explicit `if (d_is_inf(b))`, and the interpreter half's `i64FromNum` reaches `0`
  *structurally* - `Inf - Inf` is NaN and `NaN >>> 0` is `0` - so both are
  architecture independent by construction and only the Go twin moves. Measured
  across all five engines here (interpreter goja, interpreter `-frozen`, compiler
  goja, compiler `-frozen`, native): `sintConv(1/0, 64, 0)` is `0` in every one.
  **SECTION 24 deliberately asserts nothing about an infinity**, so the ratchet
  itself stays architecture independent. Same dependence as the float `%` of
  phase 4a.
- `js_giarith` and `js_giadd` in `jsrtint.go` route a `jsJFlo` (the boxed double of
  `jsrtjvm.go`) to `jvmArith` before reaching `giArith`. `jsJFlo` is a *second*
  primitive type the floor does not have, so those arms are absent. Nothing linked
  natively creates one today, and **the first of java/kotlin/csharp to migrate will
  need `jsJFlo` given exactly the treatment `jsGInt` just got.** Checked against the
  `sint*` work of 2026-08-03 rather than assumed: `sintOp` calls `rt.giArith`
  **directly**, not `js_giarith`, so it never reaches the `jvmArith` detour, and no
  `sint*` binding in either half has a float-box arm to be wrong about. Nothing here
  quietly assumes a `jsJFlo` exists or that it does not.

**The alternative was costed and rejected.** "Stop asking `js_typeof` about a number"
is not one change: it is a rule every layer-2 file of all eleven languages has to keep
at every site, forever - `lua-rt.metajs` alone has 22 `typeof` sites. It cannot be
checked mechanically, the Go twin answers `"number"` at every one of them, and a slip
is invisible unless a test happens to print the value. The tag is one place, and it
makes the two halves structurally identical instead of accidentally identical.

**Lua is converted.** `lua-rt.metajs` no longer builds `{__li: {h, l}}`; `luNorm` is
`sint(p.h, p.l, 64, 0)`, which *is* `giNorm`. The ordering dependence is gone,
demonstrated by making the same one-line deletion in both trees - remove the
sized-integer arm from `js_lustr`, so `typeof` decides:

```
                       660c47a                      HEAD
math.maxinteger        9223372036854776000          9223372036854775807
math.mininteger        -9223372036854776000         -9223372036854775808
9007199254740993       9007199254740992             9007199254740993
```

The 13,306-line Lua differential probe is byte-identical between `llvm.Run` and the
native binary, **and** byte-identical to 660c47a's native output - the conversion
changed no answer. Against the installed `lua 5.5` it differs on 2,087 lines, the same
2,087 as 660c47a, all in the number formatter.

**Cost, measured** (native, user time, 660c47a -> HEAD): 2M-iteration `s = s + i % 7`
**4.00 -> 4.16s**, `fib(24)` **0.16 -> 0.16s**, 200k big-integer `(t + big) % p`
**2.30 -> 2.47s**. Two guards had to be written carefully to get there and the
intermediate numbers are recorded at their sites: a magnitude guard on both operands of
`js_lucmp` cost **11.9s** on the first benchmark and a `luPlain()` call **7.6s**,
against 4.16s for the form that ships - `float64` is monotone, so a *strict* double
comparison is already right for a sized integer and only a TIE needs the exact path.

**What a layer-2 file has to do to use the tag** - the whole interface, with
`lua-rt.metajs` as the worked example:

```js
var b = sint(hi, lo, 64, 0)   // build one: two unsigned 32-bit halves, width,
                              // unsignedness. Answers a PLAIN NUMBER when the value
                              // is exact in a double - that is giNorm, and the
                              // invariant is applied here and nowhere else.
sintIs(v)                     // js_giis: is this a sized integer
sintHi(v) / sintLo(v)         // the two halves back out, unsigned
sintWidth(v) / sintUns(v)     // the declared width and signedness
sintOp(code, l, r)            // one binary operator, 0..10 = + - * / % & | ^ &^ << >>
sintCmp(l, r) / sintConv(v, bits, uns) / sintStr(v) / sintNum(v)
```

and then the one rule the tag imposes: **`typeof v == "number"` is TRUE for a sized
integer**, so every site that meant "an ordinary double" has to say so.
`lua-rt.metajs` spells that `luPlain(v)` and uses a cheaper equivalent in the three hot
paths. In exchange three sites that used to be load-bearing became redundant - measured
by deleting each and re-running the ratchet: the sized-integer arms of `js_lustr`,
`js_lukey` and `js_lunum` can all go without one assertion failing, because the floor
now renders, keys and coerces a sized integer correctly on its own. They are kept
because they say what they mean.

**Ratchet**: `tests/lua-test-full.lua` SECTION 24, **67 assertions**, every one also
checked against the installed `lua 5.5`. Its **discriminating power against a clean
archive of 660c47a is 0 of 67**, in all three engines - and that is the finding, not a
gap: no Lua-level program can tell the object box from the tag, which is "Lua survived
by luck of ordering" restated as a measurement. What it does discriminate is the
conversion's own failure modes; mutating one layer-2 site at a time:

```
js_lucmp: non-strict '<=', so a box tie answers wrong        2 fail
js_lucmp: the tie arm treats NaN as equal                    2 fail
js_lucmp: no exact fall-through at all (660c47a semantics)   3 fail
js_lutype: typeof instead of luPlain                         3 fail
luNorm: box every value instead of only outside 2^53        15 fail
js_luisint / js_lustr / js_lukey / js_lunum arms removed      0 fail  (redundant, above)
```

**One latent floor defect found on the way, and fixed.** `d_from_long(INT64_MIN)` built
nonsense - `0 - i` wraps back to `i`, so the top-bit scan ran zero times. Unreachable
until `si_float` passed a raw `int64` in; the probe read `-1` where Go reads
`-9223372036854775808`.

~~**The one gate item NOT met.** The sized-integer semantics could not be pinned in
`tests/metajs-test-full.js`.~~ **DONE 2026-08-03 - SECTION 24, 71 assertions,
metajs 326 -> 397.** Both halves run it and agree byte for byte, stdout and stderr,
under goja and `-frozen`; the clang-built native binary agrees too. `--full` is
**5,504 assertions, 0 languages whose halves disagree**; matrix 325/325, `--cross`
119/0, clang-check 16/16 (`metajs ... ok, and the clang executable agrees`),
`go test ./abnf/` ok.

**It was not the handful of lines this paragraph predicted, and the reason is a
THIRD implementation nobody had counted.** `sint` did not exist in the Go twin
either: `abnf/jsrt.go`'s `standardJSBindings` never bound the eleven names, so

```
$ mec languages/metajs-to-llvm-ir.abnf prog.js
js runtime error: variable not defined: sint
```

The C floor seeds them (`seed_root("sint", mk_host(40))` and the ten after it) and
`llvm.Run` did not, so the *compiler* half was only half converted: a layer-2 file
written against the floor could be linked natively but not run by `llvm.Run`. Lua
did not notice because `lua-to-llvm-ir.abnf` emits `js_lu*` externs under
`llvm.Run` and only reaches `lua-rt.metajs` through `lua-rt.ll` in the `-exe` path.
So the work is in three files, one per engine:

- **`abnf/jsrtint.go`** - `giBindings`, the eleven `sint*` globals over `jsGInt`,
  plus `programJSBindings` (= `standardJSBindings` + those), which `runJSModule`
  now uses. Deliberately NOT `standardJSBindings` itself: that one also backs
  `frozenBaseBindings`, and the grammar-script hosts (goja's in
  `commonscript.go` and the frozen one) must keep binding the same names or a tag
  script can come to depend on one engine.
- **`languages/metajs-interpreter.abnf`** - the interpreter half. **`BigInt` is
  the wrong vehicle**, which is worth writing down: the grammar's `:script` runs
  under goja *and* under the frozen MetaJS engine, and the frozen one has no
  `BigInt`. The right one was already in the tree - `lib/interp-core.js`'s
  `{__sz}` box over `{h, l}` 32-bit pairs, itself validated against `go run` over
  156,490 vectors - so the box is that and the new code is only the floor's
  `si_*` transliterated onto it.
- **`tests/metajs-test-full.js`** - SECTION 24.

**The price, and it is the interesting part.** A sized integer answering
`typeof == "number"` is not a property of a value in this half - values here ARE
the host engine's - so it is a property of the *interpreter*, and every place a
program can tell an object from a number had to be routed through the floor's
answer. There are five, and four are converted:

```
typeof v         siTypeof, wired into makeTypeof
the pinned type  siTypeof, wired into typeClass   <- var v = box; v = 7 is legal
an operator      siBinOp: to_number / to_string / the strict_eq, loose_eq and
                 js_compare arms of runtime.c, dispatched from binOp behind a
                 typeof guard (measured: no cost, 2.43s -> 2.41s on a 400k loop)
println / print  the digit text, as fmt.Fprintln of the int64 gives there
TRUTHINESS       NOT converted - see below
```

`truthy()` answers `fa(h) != 0` for tag 13, so **`if (sint(0, 0, 8, 0))` is false
in the other half and true here**: the condition is the host engine's own `if`,
and the statement builders live in `lib/interp-core.js`. Stated at the site and
deliberately not asserted. Two smaller ones for the same reason: `x++` on a box
(`makeIncDec`'s `v - 0`), and `printf`/`sprintf`, which are left unwrapped on
purpose because their `%d` takes the Go `int64` in the other half and would print
`%!d(string=...)` if this one passed the digits.

One faithfully odd thing SECTION 24 pins rather than smooths: `box == 5` is
**false** while `box === 5` is **true**. `loose_eq`'s
`(ta == 3 || ta == 4) != (tb == 3 || tb == 4)` line sees a box as neither a number
nor a string, while `strict_eq` has an explicit tag-13 arm that compares by value.
The C floor and the Go twin do this independently and identically, so it is the
specification, not a slip.

**Discriminating power, measured.** Against a clean `git archive` of f19a8ad:

```
interpreter half   71 of 71   `sint` is not defined - the section cannot run
llvm.Run half      71 of 71   same
NATIVE binary       0 of 71   the C floor already implemented all of it
```

That 0 is the useful number, not a gap: the native column is the *oracle*. The
binary built by f19a8ad's own untouched grammar and `runtime.c` runs all 71 new
assertions green, which is what says the two implementations written here match
the one that already existed rather than matching each other. Mutating one
behaviour at a time in the new code (ABORT = the half dies outright and costs its
whole 397, the trap this section had to avoid):

```
                                                        interp   llvm.Run
siNorm boxes every value instead of only outside 2^53      4         4
sint()'s width argument ignored                           13        13
sint()'s unsigned argument ignored                         -        15
hi and lo swapped                                         31         -
siTypeof answers typeof, not "number"                   1 + ABORT    -
typeClass pins typeof, not siTypeof                          ABORT   -
binOp does not dispatch to siBinOp                       8 + ABORT   -
applyOp does not go through binOp (s += box)                 ABORT   -
siStrictEq is identity                                     4         -
loose == behaves like ===                                  1         -
si_trunc always sign-extends                               5         -
sintConv ignores signedness                                3         3
siCmp always signed / the RIGHT operand wins the type    2 / 1       -
siStr uses the signed reading for an unsigned box          3         -
siNum reads a uint64 signed                                1         -
signed >> does not clamp the count to w-1                  1         -
<< ignores the count >= width rule                         1         -
division always signed / unsigned >> reads signed        2 / 1       -
sintHi reads the value signed                              -         1
the + string arm of siBinOp                                1         -
sintStr's giStr arm / sintNum's giFloat arm                -         0
print/println box wrappers; siLooseEq's two-box arm;
  siStr's signed-vs-unsigned split below 64 bits           0         -
```

The last three rows are the honest zeros. `sintStr`'s `giStr` arm and `sintNum`'s
`giFloat` arm are **redundant in the Go twin** - `rt.toString` and `rt.toNumber`
already have `jsGInt` cases that do exactly that - and are kept only because they
say what they mean, the same call the floor's `if (tag_of(v) == 13)` makes. The
`println` wrappers are not exercised because the ratchet prints nothing but its
own summary; they are pinned by hand instead (all five engines print
`18446744073709551615` for a `println` of a `uint64` box) and left in because the
alternative is `[object Object]` the first time anyone debugs with one. `siStr`'s
unsigned reading below 64 bits cannot differ at all - the payload of an unsigned
box under 64 bits is never negative.

**Four assertions had to be rewritten after the measurement**, which is the whole
argument for doing it. `>>` clamping to `w-1`, `<<` obeying count >= width,
unsigned division and the unsigned `>>` all measured **0 of 4** as first written,
because every one used an 8-bit operand and the truncation to the width hides the
difference: an 8-bit value shifted right by anything at or above 8 is `0` or `-1`
whether the count is clamped or taken modulo 64. Respelled at 64 bits
(`sint66`-`sint70`) they discriminate 1, 1, 2 and 1. **The rule is worth carrying
into the java migration: a sized-integer rule that only bites at the type's own
width has to be asserted at 64 bits, or the assertion is decoration.**

**One thing seen on the way that is NOT this change, recorded because a red line
is information.** One `./test.sh --full` run in four reported
`js-to-llvm-ir: FULL under goja, BUT -frozen fails or differs` and
`js MISMATCH: 354 FROZEN-DIFF`. It did not reproduce: the other three `--full`
runs were clean, and `js-to-llvm-ir.abnf` on `tests/js-test-full.js` hashes
**identically over eight direct runs, five goja and three `-frozen`** (same
`579ed36f...`, 1,012,563 bytes, empty stderr, exit 0 every time). The js grammar
is untouched here and the only vector this change even has into it is eleven more
entries in the map `newJSRT` seeds the root scope from. The `timeout 120`
hypothesis was measured and rejected: the entry takes **19.0s wall under a 12-way
parallel load**, nowhere near the limit. So the mechanism is still unidentified -
most likely the process being killed, since a truncated `$work.f` is exactly what
`cmp -s` would report. **`--full` should be treated as having a rare false
positive on this line until someone catches it**; re-run before believing a lone
`FROZEN-DIFF` on `js`.

~~**`c-interpreter.abnf` has no standard library.**~~ **DONE 2026-08-02.** `libcCall` in
`languages/c-interpreter.abnf` now implements the 18 byte/address names (`strlen strcmp
strncmp strcpy strncpy strcat strncat strchr strrchr strstr memcpy memmove memset memcmp
memchr atoi atol strdup`) over the interpreter's own memory, and the reverted ratchet
section is back as SECTION 50 of `tests/c-test-full.c`. **c: 505 -> 564 assertions**,
both halves FULL and byte-identical goja/`-frozen`, `--full` total 5,307 -> 5,366 with
**0 halves disagreeing**; matrix 325/325, `--cross` 119/0, clang-check still
`c ... ok, and the clang executable agrees`, `go test ./abnf/` ok.

What made it shareable: the interpreter stays **one cell per scalar** (it was NOT made
byte-packed - that is the model, and 93c814f preserved it deliberately), and
`sizeof(char) == 1` in both models, so over CHARACTER data a cell *is* a byte and a
length in bytes *is* a length in cells. SECTION 50 therefore stays on char storage;
`memcpy` over an `int` array would be n/4 cells here and n bytes there, and is left out
rather than asserted wrongly. Only the SIGN of `strcmp`/`strncmp`/`memcmp` is asserted -
C leaves the magnitude unspecified - but the interpreter returns the same **unsigned byte
difference** `libcNative`'s `byteDiff` does, verified: `cc -O0`, `cc -O2`, the
interpreter (goja and `-frozen`), `llvm.Run` and the clang-built `-exe` binary all print
`22 / -22 / 127 / 127 / -127 / 98 / 98` for the same runtime-operand probe.

Two things found on the way, both written down at the site:

- A **bare `return` followed by an expression on the next line** is a goja-vs-frozen
  trap: real JS applies ASI, the frozen MetaJS parser does not, so `if (bad) return`
  newline `mem[a] = v` parsed as `return (mem[a] = v)` and **every store was silently
  lost under `-frozen` while reads were unaffected** - 17 of 59 new assertions red in
  one half only. Guard positively. Noted at `cPut`.
- **`test.sh --full` counted `c` at 564 anyway** while `c-interpreter` was reporting
  `FULL under goja, BUT -frozen fails or differs`. The per-language assertion count and
  the halves-agree verdict are read from the goja run alone, so a frozen-only divergence
  does not reach the summary. Not fixed here (out of the files for this task) but it is a
  real hole in the invariant: the summary is not sufficient, read the per-half lines.

Still deliberately absent from BOTH halves, unchanged and loud rather than wrong: the
varargs `printf` family (neither interpreter has a varargs ABI), `exit`/`abort` (no
unwind path out of a call), `qsort`/`bsearch` (a function pointer is a `funcId`, not a
callable address). The interpreter also still has no `malloc`/`free`; `strdup` uses the
interpreter's own bump `alloc`, and nothing frees it.

## Silently wrong, both halves agreeing (the class byte-identity cannot see)

- **`_Bool b = 5;` holds 5**; `cc` holds 1. Both halves agree, so `--cross` and the
  matrix are blind. `_Bool` is `ctI(1,true)` in both grammars, indistinguishable from
  `unsigned char`, so normalizing needs a new type-descriptor flag threaded through
  `convert`/`emitConv`/`ctEq` in both halves.
- **Struct layout is packed, with no alignment padding.** `{char a; short b; int c;
  long d;}` is 15 bytes here and 16 in `cc`; `&s.d - &s.a` is 7 against 8. Both halves
  agree with each other and neither agrees with `cc`.
- **~9 latent defects in the bash ERE engine**, faithfully carried over from the
  hand-emitted version during the port and deliberately not fixed there: POSIX class
  names matched by prefix, no bound checks anywhere, `@E` not inverting `@Q`. They are
  listed in the BASH convergence section of the companion document.

## Known differences, measured, lower priority

- **Our Lua vs real lua: 3,822 probe lines, all in the number FORMATTER**, not the
  arithmetic - `1e+100.0`, `-0.0` printing as `0.0`, shortest-form against `%.17g`.
  Pre-existing in both halves.
- **The two MetaJS halves bind different host globals** (`metajs-interpreter.abnf`
  lacks `Infinity NaN Array Object byteLen sprint rawSet`), so the ratchet asserts only
  the intersection. `"ß".toUpperCase()` is the same thing in miniature: `"SS"` in the
  interpreter (real JS, full mapping) and `"ß"` in the compiler (Go, simple mapping).
- **MetaJS has no exponent literals.** `1e15` fails to parse, which is a language gap
  rather than a runtime one, and it bites anyone writing float tests.
- **Identical string literals are not merged**, where `cc` merges them. Unspecified in
  C (6.4.5p7), so not a bug - but it is observable, and worth a note so nobody
  "fixes" it into a difference that matters.

## Helpers that could not move to C

- **bash: 3** (`rt_setvar_byname`, `rt_getvar_byname`, `rt_eval_assign`) - chains over
  every variable name the program mentions, generated after the walk. Movable only if
  the emitter also emits a `{name, slot}` table.
- **batch: none. CLOSED 2026-08-02.** `rt_expand` was blocked only by the
  declared-only-function pointer-parameter defect, and `e8bf2c3` fixed it - a probe
  against clean archives of `e8bf2c3^` and `660c47a` shows `declare i32* @ext1(i32 %0)`
  becoming `declare i32* @ext1(i32* %0)`. Batch is 26/26 in C. The five functions the
  grammar still generates (`bat_setlocal`, `bat_endlocal`, `bat_shift`, `bat_lookup`,
  `main`) all walk per-program slot globals, so they are program-coupled by
  construction rather than by a defect - there is nothing left to move.

---

# Part 3 - The remaining eleven languages

## Order

Sorted by extern count and Go-twin size, which is the best available proxy for effort:

```
language     externs   dedicated Go twin
java              61   804
csharp            67   813
swift             68   986
dart              72   896
js                88   (jsrtjsprint 1320 + jsrtjsbig 505)
typescript        89   shares js's
ruby              91   -   (semantics live in abnf/jsrt.go)
go                92   630 (jsrtgolang)
kotlin            96   5934
python            99   -   (semantics live in abnf/jsrt.go)
php              101   1968
```

Two adjustments to plain size order:

1. ~~**The sized-integer floor tag (Part 2) lands before java, csharp, go and kotlin**~~
   - **done** (tag 13 of `languages/lib/runtime.c`, 2026-08-02), and **done in all
   three engines** since 2026-08-03: the eleven `sint*` globals now exist in the Go
   twin (`giBindings`) and in the interpreter half (`metajs-interpreter.abnf`) too,
   pinned by `tests/metajs-test-full.js` SECTION 24. What is left for those four is
   `jsJFlo`, the boxed double of `abnf/jsrtjvm.go` - the same problem one type over,
   and now with a worked three-engine example to copy.
2. **js and typescript go together**, since typescript has no dedicated twin and shares
   js's. Same for ruby and python, whose semantics live in the shared `abnf/jsrt.go` -
   expect those two to drag more of the shared file with them than their extern count
   suggests.

Suggested first: **java**. Smallest surface, a self-contained 804-line twin, and it
exercises the sized-integer tag immediately - which is exactly what you want the first
post-Lua language to prove.

## Per language, the shape (from the Lua pilot)

1. Split the extern list three ways: already in the C floor / language-NEUTRAL and
   belongs in the floor / language-specific and belongs in layer 2. Lua's split was
   23 / 5 / 26, and those 5 neutral ones (`js_scope_decl`, `js_scope_set`,
   `js_scope_typeof`, `js_pyset_var`, `js_keys`) are already there for everyone.
2. Write `languages/lib/<lang>-rt.metajs`, compiled by `metajs-to-llvm-ir.abnf` via a
   `tests/gen-<lang>-rt-ll.sh` (`--check` mode mandatory), linked with `-rt-lib`.
3. `abnf/jsrt<lang>.go` is the authoritative spec. Match it. Where you cannot, say so
   explicitly - approximating silently is how this project gets silently-wrong answers.
4. **Gate**: `tests/<lang>-test-full.<ext>` runs as a native binary and agrees
   byte-for-byte with `llvm.Run`, and the language moves out of clang-check's
   `ok (module only)` row into `ok, and the clang executable agrees`. Never a green
   suite over a binary nobody executed.
5. **Do not delete the Go twin** until the language passes natively and the change is
   committed.

## Traps the pilot found, which will recur

- **`_raw` parameters.** An emitter's `handle(k)` operator index is not a value handle,
  and 0..3 are the handles of `undefined/null/false/true`. This produced a silently
  wrong first build (`print(1+1)` -> `0`). Most emitters use the same trick.
- Expect each language to surface real defects in its own semantics. Lua's float `%`
  was the wrong formula in **all three halves**, and the Go twin's was
  arm64-FMA-dependent so the two Go halves disagreed on 128 of 12,557 probe lines. Use
  the real toolchain as oracle, per [oracle discipline](abnf-dialect-gotchas.md) and the
  rules at the end of the companion document.

---

# Part 4 - Regex, and then retiring Go

## Three implementations, which is two too many

```
Go    4,273 lines   abnf/jsrtregex*.go (5 files)
JS    1,559 lines   languages/lib/regex.js
C        37 funcs   re_* in languages/lib/bash-rt.c   (POSIX ERE, for bash)
```

`regex.js` already compiles through the MetaJS compiler, so the layer-2 path is open.
**Preserve the dialect modes** - `e` POSIX ERE, `j` JS repeat, `p` POSIX classes. They
are by design: `^(a*)*$` capturing `"aaa"` is JS+POSIX while Python, Java, Perl and Ruby
capture `""`, and unifying them would break three languages. Whether bash's C engine
folds into this or stays is an open question worth costing rather than assuming - it is
the only one that is already native.

## Retiring the Go runtime

29,670 lines, deletable a language at a time, only after that language passes natively.
`jsrt.go`'s host-function globals (the pre-seeded `println` and friends) become MetaJS
last, since every language depends on them.

---

# Rules (unchanged, and they earned their place this session)

- Never `git stash` / `checkout` / `reset` / `clean` - repo-wide, destroys concurrent
  work. Use `git show HEAD:<file>`.
- A change that fixes N and breaks M>N is a **LOSS**: revert it and record the
  measurement at the site.
- Ground truth is mandatory; paste the output.
- **`mec languages/x.abnf prog` reads the grammar from that path AT RUN TIME.** Building
  an old commit's binary and running it from the repo root uses the WORKING TREE's
  grammar and compares a change against itself. `cd` into each tree.
- **A crashing binary looks fast.** Check the exit code AND the expected output;
  `/usr/bin/time` reports happily on a process that died.
- Verify a commit from a clean checkout (`git archive HEAD | tar x -C /tmp/x`), not from
  the working tree.
- The defect class that keeps surfacing is **both halves agreeing and both being
  wrong**. Byte-identity cannot see it by construction. Every instance this session was
  caught by an external oracle - `cc`, real `lua`, Go's `math`.
