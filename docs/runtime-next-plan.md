# What is left: GC, the open defects, and the remaining eleven languages

Companion to [runtime-rework-plan.md](runtime-rework-plan.md), which is the record of
what was BUILT (phases 0-4a, bash/batch convergence, one linker). This document is the
forward plan. HEAD when it was written: `e8bf2c3`.

## Where things stand

```
matrix 325/325 · --full 5,615 assertions, 0 halves disagree · --cross 119/0
clang-check 16/16 - bash, batch, c, lua, metajs, php, swift, java, csharp and dart
all "the clang executable agrees"
```

Ten languages produce self-contained native binaries. Seven of them (metajs, lua,
php, swift, java, csharp, dart) got there through the GENERAL architecture - C
floor, MetaJS layer 2, language grammar - which is what makes the remaining six a
rollout rather than six rewrites.

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

## 1c. The block header given back - DONE 2026-08-03

**The 16 byte per-object header of 1b is gone, and 92% of what it cost is back:**

```
                        8c396c4      1cd6a41       HEAD
bytes/iter (400,000)      3,633        4,557       3,711
against no collector          -       +25.4%       +2.1%
```

Same binary, `MEC_GC=off`, `tests/bench-alloc.sh`. Memory is still **BOUNDED and flat**
with the collector on - and flat over a 40x range of iteration counts, not just two
points:

```
iters=  250,000   no-gc rss   928,464,896  3,713 B/it    gc rss 3,358,720
iters=  500,000   no-gc rss 1,855,520,768  3,711 B/it    gc rss 3,391,488
iters=1,000,000   no-gc rss 3,709,681,664  3,709 B/it    gc rss 3,342,336
iters=2,000,000   no-gc rss 7,417,987,072  3,708 B/it    gc rss 3,342,336
```

### What replaced the header

1b gave every allocation a header holding its size, a free bit, a cell bit and a mark
epoch. The remedy named at the site was "cells are all one size, so a cell-only region
with mark bits in a side array" - and that is only *half* of the saving, because the
benchmark's 52 allocations an iteration are 33 cells and 19 buffers. What ships is the
general form of the same idea: **a chunk serves exactly one size class AND one kind**, so
BOTH the size and the cell/buffer bit are properties of the chunk, and the two remaining
flags are one bit each in side bitmaps indexed by the block's slot number.

```
chunk: [0] next  [1] block size  [2] slots  [3] slots bumped
       [4] alloc bitmap  [5] mark bitmap  [6] kind   then 128 bytes of header, then blocks
```

Three things fell out that are worth more than the bytes:

- **Interior pointers are resolved by a division**, `idx = (v - base) / bsize`, instead of
  by scanning a start bitmap backwards. Shorter, exact, and it cannot be got wrong.
- **The free lists no longer need a header word.** A free block's own payload holds the
  next pointer; the "is it free" question is the alloc bitmap.
- **The tail-of-chunk waste of 1b's defect 2 cannot happen**: a chunk is a whole number of
  equal blocks.

The mark epoch is gone with the header: the mark bitmap is CLEARED at the start of a
collection, which is `slots/8` bytes per chunk - 6 KB for a 3 MB heap of cells - against
16 bytes on every object for ever. `MEC_GC=poison` got simpler too: a retired block just
keeps its alloc bit clear and goes on no free list.

**The residual +2.1% is the side arrays themselves**, 2 bits per block, plus the 128 byte
chunk header and one bitmap pair per 1 MB chunk. It is stated rather than rounded away:
78 bytes an iteration is what the metadata of a walkable heap costs here.

### The collector still catches what it caught, measured the same way

Every mutation of 1b's table re-run against the new collector, plus two the new design
needs. `MEC_GC=stress`, native binaries, ABORT = the binary died or never reached its own
summary. `lua-test-features` (142 checks) replaces `lua-test-full` in this run because it
is the file the interior-pointer defect showed up in:

```
                                              metajs (471)   lua-features (142)
the C stack not scanned                       ABORT          ABORT
js_gc_pin's roots not scanned                 ABORT          ABORT
a scope's PARENT not traced                   ABORT          ABORT
a closure's ENV not traced                    ABORT          ABORT
NIC / SMC_VAL (the two caches) not roots      ABORT          ABORT
an object's VALUE buffer not traced           ABORT          ABORT
an array's element buffer not traced          ABORT          ABORT
a raw buffer's contents not scanned           ABORT          ABORT
the JB jmp_buf pool not scanned               ABORT           0
a string's UTF-16 UNIT buffer not traced        14            2
G_ROOT not an explicit root                      0            0
THROWN not an explicit root                      0            0
gc_try takes EXACT block addresses only          0            9   <- the -O2 defect
the mark bitmap is not cleared between GCs    ABORT          ABORT
the callee-saved registers not scanned           0            0   <- WEAKER, see below
```

**The interior-pointer row is the one that mattered and it is unchanged: 9 failures**, all
in `string.sub`, exactly as when 1b found it. The last row is a LOSS of discriminating
power and is reported rather than buried: at 1cd6a41 not scanning the callee-saved
registers aborted both ratchets, and with this allocator neither notices. Nothing about
the register scan changed - what changed is which handles the C compiler chooses to keep
in a callee-saved register across `ar_alloc`, which is a property of the code shape, not
of the collector. The scan stays, for the same reason `G_ROOT` and `THROWN` stay: a root
set that depends on a register allocator's habits is not a root set.

### Every suite, again

`MEC_GC=off / auto / stress / poison` compared byte for byte on stdout, stderr and exit
status, over every Lua and MetaJS program in `tests/` built as a native binary:

```
native GC sweep: 20 programs built, 20 identical across off/auto/stress/poison, 0 divergent
```

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
- ~~`js_giarith` and `js_giadd` route a `jsJFlo` to `jvmArith` before reaching
  `giArith`, and the floor has no such type, so those arms are absent.~~
  **CLOSED 2026-08-03 - tag 14, below.** Both arms exist, and a 845 line probe that
  calls the externs themselves says they agree with the Go twin exactly.

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
sintRaw(hi, lo, bits, uns)    // the same as sint(), WITHOUT si_norm: ALWAYS a
                              // box, never a plain number. si_trunc is still
                              // applied, so the cell keeps its own invariant.
```

### `sintRaw` - the one door past `si_norm`, DONE 2026-08-03

`si_norm` UNBOXES a signed 64 bit value a double holds exactly, and that is exactly
what a statically typed language's 64 bit signed type must NOT do: Java's
`1000000L * 1000000L` is 1000000000000 where `1000000 * 1000000` is -727379968.
Every one of the eleven `sint*` builtins went through `si_norm`, so **layer 2 had no
constructor for a small BOXED value at all**. Java worked around it by marking the
box UNSIGNED - the one shape `si_norm` will not unbox - and re-supplying the six
operations whose signed reading the floor then gets wrong (`/ % >>`, compare,
decimal text, double reading) in ~90 lines of `java-rt.metajs`. **csharp, go,
kotlin, swift and dart each have a 64 bit signed type and would each owe that same
workaround**, which is why the constructor is cheaper than five copies of it.

`sintRaw(hi, lo, bits, unsigned)` is `si_make(si_trunc(v, w, u), w, u)`: everything
`sint()` does except the unboxing arm. **`si_trunc` is deliberately kept** so a
caller asking for 8 bits gets a well formed `int8` box rather than 300 wearing an
`int8` label - only `si_norm`'s plain-number arm is dropped. Bound in **all three
engines**, which is the gap `f19a8ad` shipped for `sint` and `8c396c4` had to
repair:

```
languages/lib/runtime.c          seed_root("sintRaw", mk_host(62)), host id 62
abnf/jsrtint.go                  giBindings - now TWELVE names, not eleven
languages/metajs-interpreter.abnf siRawHost, in hostGlobals
```

**Ground truth.** A MetaJS probe reaches `sintRaw` and answers identically in all
five engines (interpreter goja and `-frozen`, `llvm.Run` goja and `-frozen`, the
clang-built native binary) - byte for byte, including `si_trunc` at 8 bits, the
64 bit unsigned box, the signed `/ % >>` and compare on a raw signed box, and
`sintConv` normalising back to a plain number.

**Ratchet**: `tests/metajs-test-full.js` SECTION 24, `sint72`-`sint89`, **18
assertions**; metajs 476 -> 494 (501 with the `to_int32` assertions below).

**Discriminating power.** Against a clean `git archive` of `94ac7b8` it is
**18 of 18 in every engine, by ABORT** - `sintRaw` is not defined in any of the
three, so the file dies. That is the weak measurement, and it only says the name is
new; the useful one is mutating a single behaviour at a time, which is **identical
in all three engines** and is the evidence that the three implementations match each
other rather than one of them:

```
                                                  interp   llvm.Run   native
sintRaw goes through si_norm (== sint)               3         3         3
si_trunc not applied                                 2         2         2
the width argument ignored                           2         2         2
the unsigned argument ignored                        3         3         3
hi and lo swapped                                   10        10        10
```

**Two assertions had to be rewritten after the measurement**, the same lesson
`sint66`-`sint71` record: "`si_trunc` not applied" and "the unsigned argument
ignored" each measured **1** as first written, because an 8 bit operand hides both -
an unsigned box under 64 bits never has a negative payload, and 200 masked to 8 bits
is 200 either way. Respelled at 32 and 64 bits (`sint88`, `sint89`) they measure 2
and 3. **A sized-integer rule has to be asserted at a width where it can bite.**

**NOT done here, and owed to whoever converts them**: `java-rt.metajs` is another
agent's file and is untouched, so java still carries the unsigned-box workaround.
The API above is what collapses it.

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

~~**`jsJFlo`, the boxed double, is a SECOND primitive the floor does not have.**~~
**DONE 2026-08-03 - tag 14.** Ten of the eleven remaining languages need it (`grep -c
jsJFlo abnf/jsrt*.go`: kotlin 32, jvm 18, csharp 12, java 11, swift 9, golang 4, dart 1,
jsrt.go 15; only php does not), so it was the last prerequisite for the rollout.
`abnf/jsrtjvm.go` stays the authoritative specification and every function of the floor
is named after the `jvmXxx` it implements.

```
cell.a  the double's BITS        cell.b  the print style: 0 java, 1 go, 2 c#
```

**Why it is a second type and not a case of tag 13**, stated because the two look alike:
a sized integer's payload is an integer and `si_norm`'s whole job is to UNBOX a value a
double holds exactly, which is the opposite of what a double needs - `1.0` and `1` must
stay distinguishable or `1.0 / 3.0` and `1 / 3` are the same two operands. A boxed double
therefore NEVER unboxes. `jsrtint.go` says the same thing from the other side: `giVal`
reads a `jsJFlo` by truncating it.

**Ground truth: two differential probes against a Go oracle built out of the real
functions, not a copy of them.**

```
                                            lines   vs the Go oracle
jfprobe.js, llvm.Run (goja and -frozen)    10,149   BYTE-IDENTICAL
jfprobe.js, the frozen interpreter         10,149   BYTE-IDENTICAL
jfprobe.js, the NATIVE binary              10,149   63 lines differ (see below)
jfprobe.js, the goja interpreter           10,149    6 lines differ (see below)
giprobe.c, the js_gi* EXTERNS, native         845   BYTE-IDENTICAL
```

The first probe walks 24 mantissas x 61 decimal exponents x both signs through all three
renderings, then 5 operators x 16 values x 16 values x 4 boxedness combinations, then the
same grid for `floEq`, `floMax`/`floMin` and `floAbs`. The oracle is a Go test that calls
`jvmFloText` / `rt.jvmArith` / `jvmNumEq` / `rt.jvmMinMax` directly. The second reaches
what no MetaJS program can - `js_giarith`, `js_giadd`, `js_gineg`, `js_gieq`, `js_gilt`,
`js_giconv` and the rest with a boxed-double operand - by compiling `runtime.c` with `cc`
against a driver that defines `jsmain`/`jsdispatch`, and its oracle calls the same externs
out of `rt.externs()`. **That is the only coverage the `jvmArith` routing in `js_giarith`
and `js_giadd` has**, and it is why it exists: SECTION 26 measures 0 for those two arms,
because a MetaJS operator never reaches them.

**The two places it does not match, stated rather than approximated:**

- ~~**63 native lines: `to_int32` of a value outside the int64 range.**~~ **FIXED
  2026-08-03, in all five engines, against node as the oracle.** It was three
  implementations and three answers, none of them JS: `rt.toInt32` ended in
  `int32(int64(f))`, which Go leaves implementation defined out of range and which
  answered `-1` here on arm64; the C floor answered `0` from an explicit `d_in_long`
  test; goja's own operator did the same int64 conversion as the Go twin. **ToInt32
  is a MODULO, not a range test** - `1e20 | 0` is 1661992960 and `1e19 | 0` is
  -1981284352 - and all four sites now say so:

  ```
  languages/lib/runtime.c   to_int32   the significand shifted into place by
                                       e-52, masked BEFORE the shift; a shift at
                                       or above 32 is a multiple of 2^32, so 0
  abnf/jsrt.go              rt.toInt32 math.Mod, which is fmod and exact
  languages/metajs-interpreter.abnf     siI32, guarded by four comparisons in the
                                        six bitwise arms of binOp - goja's own
                                        operator is the wrong one, so the guard is
                                        there to keep the goja half from being the
                                        only engine of the five that is wrong
  ```

  **45 vectors** (both signs, 2^63/2^64/2^84, the `1e19`..`1e300` decade walk, the
  int64 boundary from below, the low-32-bits-are-zero cases) are **byte-identical to
  node in all five engines**. Ratchet: SECTION 04 `bit16`-`bit22`, 7 assertions,
  every constant taken from node. Discriminating power against a clean archive of
  `94ac7b8`: **6 of 7** in the goja interpreter, the frozen interpreter and
  `llvm.Run`, **5 of 7** native. The two honest zeros are deliberate - `bit21` pins
  that the *fast path below 2^63 is unchanged*, and `bit20`'s "the low 32 bits are
  genuinely zero, so 0 is the right answer" case is what the floor already said, so
  it cannot discriminate there. It is kept because without it a fix that replaced
  one wrong constant with another would pass.
- **6 goja-interpreter lines: the sign of a zero PRODUCT.** goja represents an integral
  number as an int64, so `0 * -2718281` is `+0` there and `-0` in real JS, in the frozen
  engine and in the C floor. Measured directly: `1 / (0 * -2718281)` is `+Inf` under goja
  and `-Inf` in the other three. A property of the host engine, not of this type; SECTION
  26 asserts a `-0.0` that comes from a small product, which is safe in all four.

**One floor defect the probe found and fixed on the way.** `d_mod_go` put the sign back on
a negative remainder with `r = 0.0 - r`, which turns `-0.0` into `+0.0`, so `Mod(-1, 1)`
was `+0` where Go's `math.Mod` gives `-0`. Invisible before, because the JS rendering of
`-0.0` is `"0"` - Java's is `"-0.0"`. 18 of the probe's lines. The sign is flipped as a
BIT now.

**What a layer-2 file has to do to use the tag** - the whole interface, ten host globals
in the style of the `sint*` family (`seed_root` ids 51..60 in `runtime.c`, `jfBindings` in
`abnf/jsrtjvm.go`, `hostGlobals` in `metajs-interpreter.abnf`):

```js
var d = flo(1.5, 0)      // build one: the VALUE and the print style
                         //   0 java  1.0 -> "1.0"   1e20 -> "1.0E20"  inf -> "Infinity"
                         //   1 go    1.0 -> "1"     1e20 -> "1e+20"   inf -> "+Inf"
                         //   2 c#    1.0 -> "1"     1e20 -> "1E+20"   inf -> "Infinity"
                         // ONE argument, unlike sint()'s two halves: a MetaJS
                         // number already IS a double. It never unboxes.
floIs(v)                 // jvmIsFlo
floNum(v)                // the double back out (to_number)
floStyle(v)              // the style; 0 for anything that is not a box
floStr(v)                // jvmFloText - the box's own language's rendering
floOp(code, l, r)        // one binary operator, 0 + 1 - 2 * 3 / 4 %  (jvmArith):
                         // a float operand makes the result a BOX, two integral
                         // ones keep the 32 bit wrap
floEq(l, r)              // jvmNumEq: 1.0 == 1 holds, NaN never does, and a box
                         // against a SIZED INTEGER is false
floMax(l, r) / floMin(l, r) / floAbs(v)    // jvmMathObject's three
```

and the three rules the tag imposes, all pinned by SECTION 26:

1. **`typeof v == "number"` is TRUE for a boxed double**, as for a sized integer, so a
   site that meant "an ordinary double" has to say so.
2. **The ORDINARY operators do not box.** `flo(1.5,0) + 1` is the plain `2.5`, because
   `js_add_v` answers a plain number; only `floOp` and the `js_jf*` externs of a compiler
   box a result. `flo(1,0) === 1` is true and `flo(1,0) == 1` is FALSE - the same
   faithfully odd pair the sized integer has, from the same line of `loose_eq`.
3. **Truthiness is not converted in the interpreter half.** `runtime.c`'s `truthy()`
   answers `!zero && !nan` for tag 14, so `if (flo(0))` is false there and true in the
   interpreter, where the condition is the host engine's own `if`. Same hole as tag 13,
   stated at both sites and deliberately not asserted.

The floor also implements the language-neutral `js_jf*` externs the java / kotlin / csharp
emitters already emit - `js_jflo`, `js_gflo`, `js_csflo`, `js_jfsub/mul/div/mod`,
`js_jfneg`, `js_jfint` - so those three can be linked natively without touching the floor
again. `js_jfstep` and `js_jadd` are deliberately absent: their Go arms turn on `jsChar`,
a type the floor does not have, and approximating one silently is how this project gets
silently wrong answers.

**Ratchet**: `tests/metajs-test-full.js` SECTION 26, **51 assertions**, all five engines
(interpreter goja and `-frozen`, `llvm.Run` goja and `-frozen`, the clang-built native
binary) green and byte-identical. metajs 404 -> 455.

**Discriminating power, measured.** Against a clean `git archive` of `1cd6a41` the answer
is **51 of 51 in every engine** - `flo` is not defined in ANY of the three, so the file
dies. There is no oracle column this time, and that is the difference from the `sint*`
work: the C floor was already the oracle there, and here nothing implemented `jsJFlo` at
all. So the evidence is the two probes above plus mutating one behaviour at a time:

```
                                                        native   llvm.Run   interp
jf_text ignores the style                                 13         -         -
to_number: tag 14 falls through to NaN                     9         -         -
jvm_flo_str drops the forced ".0"                          6         -         -
jf_arith boxes only when BOTH operands are boxes           7         -         -
strict_eq's tag-14 arms removed                            2         -         -
jf_num_eq accepts any operand                              2         -         -
type_of: tag 14 is not a number                            1         -         -
jf_style_of: the RIGHT operand wins                        1         1         1
jf_minmax never re-boxes                                   1         1         -
the C# exponent is not padded to two digits                1         -         -
flo() ignores the style argument                           -        16         -
floOp always uses '+'                                      -         6         -
floAbs does not keep the box                               -         2         -
jfBindings not added to programJSBindings                  -      ABORT        -
siTypeof: a box is not a number                            -         -      ABORT
flOp does not box its result                               -         -         8
flDigits keeps the trailing zeros                          -         -         7
binOp does not dispatch to flBinOp                         -         -         6
flPlain never forces the ".0"                              -         -         6
flStrictEq is identity                                     -         -         4
flSci: Java gets a signed padded exponent                  -         -         3
flNegZero always false                                     -         -         1
flLooseEq behaves like ===                                 -         -         1
js_giadd / js_giarith float arms removed                   0         -         -
Java's window by the decimal exponent                      0         -         0
floEq is rt.strictEq only                                  -         0         -
floStr uses rt.toString for a box too                      -         0         -
print/println box wrappers                                 -         -         0
```

**Five honest zeros, and each one says something.** The `js_giadd`/`js_giarith` arms are
unreachable from a MetaJS operator - that is what `giprobe.c` is for, and it puts them at
845 of 845. `floEq` and `floStr` are **redundant in the Go twin**: `rt.strictEq` and
`rt.toString` already have `jsJFlo` cases doing exactly that, and they are kept because
they say what they mean, the same call the floor's `if (tag_of(v) == 14)` makes. The
`println` wrappers are not exercised because the ratchet prints only its own summary; they
are pinned by hand instead - all five engines print `1.0 1 1` for a `println` of the three
styles. And **Java's plain-decimal window is measured indistinguishable** whether it is
tested on the VALUE (`a >= 1e-3 && a < 1e7`, what Go's source says) or on the decimal
exponent (`e10 >= -3 && e10 < 7`): 0 of 51 assertions AND 0 of 10,149 probe lines. The
value test is kept because it is what the twin does, not because it was shown to matter -
recorded at the site in both files so nobody "fixes" it back.

**One rule worth carrying into the java migration**, the float twin of the sized integer's
64-bit rule: **a rendering rule that only bites at a window boundary has to be asserted AT
the boundary in both directions**, or the assertion is decoration. `flo17` and `flo18`
pair `0.001` with `0.0001` and `9999999` with `10000000` for exactly that reason.

## Two builtins the PHP migration needed, and the floor owed

> **A THIRD one, `delKey`, landed 2026-08-03 for the same reason** - see "The two
> floor requests granted, and the template literal that needed FOUR halves" in part
> 3. `keysOf` (id 61) lets a layer-2 file WALK an object; `delKey` (id 63, the
> floor's new `js_del`) lets it PRUNE one, which MetaJS has no operator for either.
> The interpreter half solves it the same way `keysOf` does, with the hidden `__ik`
> insertion-order list, and the ratchet is `tests/metajs-test-full.js` SECTION 28.


**`keysOf`, and why an extern was not enough.** The floor has had `js_keys` all along, but
**an extern is only callable from an emitter**, and MetaJS has neither `for..in` nor
`Object.keys` - so a layer-2 file could not enumerate an object's own keys AT ALL. That
blocks `var_dump` of an object, `==` between two objects, the `(array)` cast and `clone`
in PHP, and the same four in the nine after it. `keysOf` is now a builtin in all three
engines (`seed_root` id 61, `keysBindings` in `abnf/jsrtint.go`,
`metajs-interpreter.abnf`), with `js_keys`'s exact semantics: a dict box yields its key
array, a plain object yields its string keys in INSERTION order, skipping the internal
`__`-prefixed slots.

The interpreter half is the interesting one. **It cannot ask the host engine, because
there is no answer in BOTH of its engines**: the grammar's `:script` runs under goja,
which has `Object.keys`, and under the frozen MetaJS engine, whose `Object` carries only
`prototype`. So the interpreter remembers the insertion order itself, in a hidden own
property written with `rawSet` (which both script hosts bind) and hidden again in
`getMember` - the other two halves skip every `__`-prefixed key by the same rule, so the
name is invisible there too. The write sites are `makeObject`, `makeAssign`, `makeIncDec`
and `makePreIncDec`, all in `metajs-interpreter.abnf`, and an ARRAY is excluded by the
`typeof o.length == "number"` test `lib/interp-core.js` already uses.

**`d_pow` lost the sign of an underflowed zero**, and only the native binary was wrong -
the class that byte-identity cannot see, because `./test.sh` compares each engine against
itself and only `tests/clang-check.sh` runs the native one at all:

```
                                    node    llvm.Run    the C floor
Math.pow(-0.5, 2147483647)           -0        -0            0     <- fixed
Math.pow(-0.5, 3)                  -0.125    -0.125      -0.125
```

`d_ldexp` returned the literal `-0.0` on underflow, and **unary minus on a double is
emitted as `0.0 - x` by `languages/c-to-llvm-ir.abnf`**, where `0.0 - 0.0` is `+0.0`. It
is written as the sign BIT now. The infinity one line below is unaffected: `-1.0 / 0.0`
computes its sign rather than writing it. Worth remembering as a general trap in this
file: **a negative floating point LITERAL does not survive the C floor's own compiler.**

**Ratchet**: SECTION 27, **16 assertions** (11 `keysOf`, 5 `pow`), metajs 455 -> 471, all
five engines green. Discriminating power: `keysOf` is ABORT in all three engines against a
clean `1cd6a41` (the name does not exist), and the five `pow` assertions, extracted into a
standalone probe so they can run there at all, measure **2 of 5 in the native binary and 0
of 5 under `llvm.Run` and the interpreter** - which is the finding: the two Go-side halves
were already right and only the floor was wrong.

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

- ~~**`_Bool b = 5;` holds 5**~~ **FIXED 2026-08-04, both halves, oracle `cc`
  (clang 17).** The type-descriptor flag is `ctBool = {k:"int", w:1, u:true,
  bool:true}` in `c-to-llvm-ir.abnf` and `c-interpreter.abnf`; it is read by
  `emitConv` / `convert` (a TEST against zero, C11 6.3.1.2, not a truncation) and by
  `ctEq` (`_Bool` is its own standard unsigned integer type, 6.2.5, so `_Generic`
  separates it from `unsigned char`). Only the DECLARATOR normalized before, so an
  assignment, an array element, a struct field, a store through a pointer, a cast, a
  compound assignment, a parameter and a return all kept the raw value. Two more
  arms were needed for the compiled half: a `_Bool` PARAMETER is tested rather than
  truncated in the prologue, and - a broader defect found on the way - a function's
  return value was never converted to a narrow DECLARED return type at all, because
  `curFnRetCt` only ever held a type 64 bits wide or a float (it drives the module
  SIGNATURE). `unsigned char f(int n){return n;}` called with 300 answered 300 in the
  compiled half against 44 in `cc` AND in the interpreter half; `curFnRetNarrow`
  applies 6.8.6.4p3 without moving the signature. Pinned as `tests/c-test-full.c`
  checks 2008-2020, which are **11 red in the interpreter half and 12 in the
  compiled one** from a clean archive of ad922a0. Regenerating
  `languages/lib/batch-rt.ll` was owed by the return-type fix (four lines of sign
  extension in one `char`-returning function) and is done.
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

## FIVE THINGS EVERY REMAINING LANGUAGE NEEDS - read this first

*(It was TWO when php wrote it. Swift, java, csharp, dart, go, ruby, kotlin and
python each added one or corrected one, and the additions are appended in date
order below. As of 2026-08-03 PYTHON, the last language this plan was written for,
is done; see "What PYTHON added, as the LAST of the nine" below and the PYTHON
section further down.)*

Both came out of the PHP port. Neither is about PHP, both are cheap, and the second
one is a **silently-wrong build** if it is missed - so it is the more dangerous.

### 1. A scope probe belongs in the EMITTER, not in layer 2

An extern that takes a scope handle and walks the chain - `js_phthis`,
`js_phclslookup`, and their equivalent in every other language - **cannot be written
in layer 2 at all.** A scope is the C floor's private storage; that is precisely why
the Lua pilot moved `js_scope_decl`/`js_scope_set`/`js_scope_typeof` INTO the floor,
and MetaJS reaches the floor only through host builtins, of which there is no scope
one.

It does not need a floor addition. Every such extern is a pure PROBE - *the value of
this name if it is declared anywhere up the chain, else a default* - and an
**emitter** may call `js_scope_typeof` and `js_scope_get`. So lower it in the
grammar. `php-to-llvm-ir.abnf` does it with one module-level helper:

```
php_scope_probe(env, nm, dflt):
    js_scope_typeof(env, nm) === "undefined" ? dflt : js_scope_get(env, nm)
```

A single helper FUNCTION rather than an inline branch is the part worth copying: it
keeps every call site a single-value expression, so none of the twelve PHP ones had
to be restructured into a block split. The two externs left the module (101 -> 100)
and `llvm.Run` stayed at 306/306.

The residue is any extern that needs a scope for something other than a probe. PHP
had one, `js_phrtinit`, which hands the runtime the descriptors its own
`DivisionByZeroError`/`ArithmeticError` are raised with. The fix is the same idea:
**pass VALUES, not the scope** - the emitter is the side that may read a scope, so it
reads it and hands over a plain array of name => descriptor. `abnf/jsrtphp.go`'s
`phpRaise` accepts either shape, so nothing that was green went red.

### 2. `core.truthyExt` returns a RAW INTEGER, and a `-rt-lib` shim must too

The Lua pilot recorded `_raw` PARAMETERS - an argument that is not a value handle.
**The mirror image exists, and it is worse.** `compile-core`'s `truthy()` is

```
b.NewICmp(IPredNE, callExt(b, core.truthyExt, [v]), handle(0))
```

which compares the extern's result against 0 **as a raw integer**. So `js_truthy` -
and every language's own spelling of it - answers the integer 1 or 0 and *not a
handle*. A `-rt-lib` shim hands back whatever `js_call` returned, so a MetaJS
`false` arrives as the HANDLE of false, which is **2**, and `2 != 0`:

> **Every condition in the program is taken. The build is silently wrong, it
> compiles, it links, and nothing in the suite looks at it.**

`languages/metajs-to-llvm-ir.abnf`'s export scanner now understands a `__raw` suffix
on the FUNCTION NAME, symmetric to `_raw` on a parameter: the symbol is exported
without the suffix and the result is put through the floor's own `js_truthy`, which
is exactly the raw 1/0 the caller compares. Layer 2 writes

```
function js_phtruthy__raw(v) { return phTest(v) }
```

**Every remaining language has a `core.truthyExt`, so all nine need this**, and any
other extern whose result the emitter consumes as a raw integer rather than a handle
needs it too - check each emitter for `NewICmp` on a `callExt` result. `lua-rt.ll`
regenerates byte-identically under the change, and so does `abnf/jsbootstrap.ll`.

Conversely, `_raw` PARAMETERS are **not** universal: `php-to-llvm-ir.abnf` passes
every operator selector with `emitStr`/`emitNum` and has **zero** of them, where Lua
had three. Check each emitter rather than assuming either way.

### What SWIFT added to both of these (2026-08-03)

**Item 1 held, and it is now the confirmed pattern rather than one data point.**
`js_kget` / `js_kset` are Swift's scope probes - the `this`-aware Kotlin shape, "a
name that is no local resolves against the properties of `self`" - and neither
could be written in layer 2. Both are now the emitted IR helpers `sw_kget` /
`sw_kset` in `swift-to-llvm-ir.abnf`, over `js_scope_typeof` / `js_scope_get` /
`js_scope_set`, and the extern count went **68 -> 66** with `llvm.Run` unchanged at
209/209. Two languages, two lowerings, no floor change either time. **Expect one
per language and expect it to be cheap.**

Where Swift's differs from PHP's is worth copying too: PHP's probe is a
single-expression helper, which is what kept its twelve call sites from being
restructured. Swift's has a `this` FALLBACK and a write side, so it is two IR
FUNCTIONS instead - 43 read sites and 4 write sites became one call each, and the
one site that was not a plain `callExt` (compile-core's `makeVarRef`, reached
through `core.scopeGetExt`) was overridden in the grammar, since that field names
an extern and `js_kget` is no longer one.

> The one inexactness, stated at the helper: `js_scope_typeof` answers
> `"undefined"` for a slot that HOLDS undefined and for an absent name alike, so
> such a name would take the self fallback and then the final `js_scope_get`,
> which returns undefined in the Go twin and ABORTS in the C floor. Swift's
> emitter declares an uninitialised `let v: Int` as NULL, so no emitted binding
> takes that path - but the next language must check its own.

**Item 2 has a first EXCEPTION, and it is the cheapest possible one.** Swift never
sets `core.truthyExt`, so it stays at compile-core's default `"js_truthy"` - which
is **the C floor's own function**, already answering the raw 1/0 the `icmp` wants.
There is no shim to get wrong. So the rule is not "all nine need `__raw`" but:

> **A language needs the `__raw` suffix exactly when it OVERRIDES
> `core.truthyExt` with an extern of its own.** Lua and PHP do; Swift does not.
> `grep -n 'truthyExt' languages/<lang>-to-llvm-ir.abnf` answers it in one line,
> and an empty result means the floor already has you covered.

Swift also has **zero** `_raw` parameters, like PHP and unlike Lua - so that count
is now 3 / 0 / 0 across the three ported languages.

### What JAVA and CSHARP added (2026-08-03)

**Item 1 is now four for four, and csharp is the cheapest possible case: NOTHING TO
LOWER.** Java's probe was not a language feature at all (`__jmath`, a host global of
the Go runtime that `runtime.c` does not seed), one `java_scope_probe` helper in
PHP's shape. `csharp-to-llvm-ir.abnf` already probed the scope in IR
(`js_scope_typeof` against `"undefined"`, line 4109) and never had a scope-taking
extern, so there was nothing left to do. **Check for the probe before assuming one is
owed** - the running count is php 2 externs lowered, swift 2, java 1 helper added,
csharp 0.

**Item 2 is now three exceptions to one rule.** Swift, Java and C# all leave
`core.truthyExt` at compile-core's default, which is the floor's own `js_truthy`.
Only Lua and PHP override it, and only they need the `__raw` suffix. The grep is one
line and it is worth running before writing anything.

`_raw` PARAMETERS across the five: **lua 3, php 0, swift 0, java 1, csharp 1**. Both
of the ones is `js_char`, whose code point the emitter writes with `handle(k)`.

**A THIRD thing joined them, and it is CLOSED (2026-08-03).** Every remaining
language can put a GENERATOR on an object, and the floor's `type_of` used to answer
`"object"` for a generator function (tag 16) where `abnf/jsrt.go` answers
`"function"`. Any layer-2 `js_mcall` written with `typeof m == "function"` therefore
refused to call an iterator method NATIVELY while calling it happily under
`llvm.Run`. Measured in the csharp section below, where it was the first native
failure and hid the other 246 assertions behind it. **`type_of`, `type_class` and
`to_string` in `languages/lib/runtime.c` now list `t == 16` next to 7/8/9**, so
`typeof m == "function"` is the correct test in every layer 2 and no `apply` probe
is owed. The four languages that follow (js, ts, python, and any go/kotlin/dart
generator) inherit the fix and should not re-invent the workaround.

### What DART added to all three (2026-08-03)

**Item 1 is now COPYABLE CODE rather than a pattern.** Dart's `js_kget`/`js_kset`
are not merely the same SHAPE as Swift's - `abnf/jsrt.go` implements them with the
same two functions - so `sw_kget`/`sw_kset`/`sw_safeget` transferred to
`dt_kget`/`dt_kset`/`dt_safeget` verbatim, along with the
`core.scopeGetExt = "js_scope_get"` override. 31 read sites and 3 write sites,
`llvm.Run` unchanged at 228/228, third language, still no floor change. **Look up the extern in
`abnf/jsrt.go` before writing a new lowering: if it is one of the two above, the
lowering already exists.**

One correction to the accounting, which Swift's `68 -> 66` did not make: the
lowering **removes two externs and introduces three** - `js_scope_typeof`,
`js_typeof` and `js_sne`, which the emitter did not previously reach. All three
are already in the floor, so what LAYER 2 has to write falls by two and the
declare list grows by one. Dart's union went 66 -> 67 for exactly that reason.

**Item 2: dart is the third exception.** It sets no `core.truthyExt`, so the
default `js_truthy` - the floor's own - applies and no `__raw` suffix exists in
`languages/lib/dart-rt.metajs`. The tally over five ported languages is **lua and
php need it; swift, java and dart do not**, and the `grep` still answers it in one
line. `_raw` PARAMETERS are now **3 / 0 / 0 / 1 / 0** for lua / php / swift / java
/ dart.

**Item 3 did not bite dart, and the reason is checkable in one grep.**
`languages/lib/dart-rt.metajs`'s `dtMember` does test `typeof mm == "function"`,
so it would refuse a tag-16 generator natively - but
`grep -c 'js_genfn\|js_yield' languages/dart-to-llvm-ir.abnf` is **0**: Dart's
`sync*`/`async*` are lowered by the emitter into ordinary closures, so no tag 16
ever reaches the dispatcher. **Run that grep before deciding whether the
`is_callable` spelling is needed** - it is the difference between a defensive
rewrite and a measured "cannot happen".

### What GO added to all three (2026-08-03)

**Item 1 has a NEW SHAPE, and it is the generalisation the first five were
circling.** Go has no scope-taking extern at all - like csharp, its emitter
already calls `js_scope_get`/`js_scope_set`/`js_scope_decl` directly. What it had
instead was `js_gounctl`, which reads the completion value out of a CONTROL
SIGNAL: a **tag-10** cell, the floor's private storage, with no `seed_root`
accessor - so layer 2 could not have written it either. Same answer:
`js_ctl_kind` and `js_ctl_value` ARE floor externs, an EMITTER may call them, and
`go-to-llvm-ir.abnf`'s `emitUnctl` emits the two-arm select at the one call site.

> **The rule to carry forward is not "expect a scope probe" but "expect a
> PRIVATE-REPRESENTATION probe".** A scope is one such representation; a control
> signal is another; a coroutine cell would be a third. The question is *does
> this extern have to open a cell only the floor knows?* - and the answer is
> always to find the floor extern that already answers it and lower the call.
> Six languages, five lowerings, still no floor change.

**And item 2 fires INSIDE that lowering, from the result side.** `js_ctl_kind`
answers a **RAW INTEGER** (`mk_ctl`'s `fa` field: 0 not-a-signal, 1 return, 2
break, 3 continue), so the emitted test must be `NewICmp(IPredEQ, kind, 1)`.
`truthy(b, kind)` compiles, links, and silently takes kinds 2 and 3 as well. The
`__raw` finding is not only about a `-rt-lib` shim: **any emitter that consumes a
`callExt` result as a number needs the same check**, and `grep NewICmp` over the
grammar is still how to find them.

**Item 2 itself: go is the fifth exception.** `grep -n truthyExt
languages/go-to-llvm-ir.abnf` is empty, so the default `js_truthy` - the floor's
own - applies. The tally over six ported languages is **lua and php need it;
swift, java, csharp, dart and go do not**. `_raw` PARAMETERS are now
**3 / 0 / 0 / 1 / 1 / 0 / 1** for lua / php / swift / java / csharp / dart / go;
go's one is `js_gorest`'s start index.

**Item 3 could not bite go either, for dart's reason and one more.**
`grep -c 'js_genfn\|js_yield' languages/go-to-llvm-ir.abnf` is **0** - and in
go's case that is not a lowering but the CONCURRENCY MODEL: goroutines are
cooperative and run to completion, so nothing suspends and no tag 16 exists.
See the go section for the full finding.

**A FOURTH thing, and it corrects the Java audit rather than extending it.** The
second-order hazard table in the java section names `js_gival` as go's suspect
(row 6: "it unboxes via `si_float` where `si_val` is right"). **It is not a
hazard.** `si_float`'s unsigned-width-64 arm answers the MAGNITUDE, and that is
exactly what go wants, because go really has `uint64` and `abnf/jsrtint.go`'s own
`js_gival` writes the same two lines. The hazard needs a language that carries a
SIGNED value in an UNSIGNED-MARKED cell - Java's trick, which `sintRaw` has made
unnecessary. **Read the table as "check the signedness claim", not as "wrap every
call".**

### What RUBY added to all four (2026-08-03)

**Item 1 is now SIX lowerings and the private-representation rule holds.** Ruby's
was `js_rdefined(scope, name, kind)` - `defined?(x)`, a pure scope probe - lowered
into the emitted IR helper `rb_defined` in PHP's single-helper shape. `llvm.Run`
stayed at 267/267 and no floor change was needed. The running count is php 2
externs lowered, swift 2, java 1 helper, csharp 0, dart 2, go 1, ruby 1.

**Item 2 has a SEVENTH answer, and it is the cheapest of all: no extern at all.**
`grep -n truthyExt languages/ruby-to-llvm-ir.abnf` is empty, and ruby goes further
than the languages that merely keep compile-core's default - it overrides
`truthy()` itself with `NewICmp(IPredUGE, v, handle(3))`, because the value handles
0/1/2 are `undefined`/`null`/`false` and 3 is `true`, so "`>= 3`" IS Ruby
truthiness with no call in the path. **Check whether the grammar overrides
`truthy()` as well as `core.truthyExt`** - the first is the cheaper place and the
second language that wants "everything except a fixed set of singletons is true"
should copy it. It is sound natively only because `runtime.c` seeds the same four
handles as `abnf/jsrt.go`; that is one grep (`H_UNDEF = 0`) and it is worth doing.

**Item 3 (tag 16) is absent again**, `grep -c 'js_genfn\|js_yield'` = 0: Ruby's
`yield` is a block call, not a generator.

**A FOURTH thing joins them, it is silently wrong, and every remaining language
that computes a remainder has it.**

> **`x - math.Floor(x/y)*y` in the Go twin is FUSED into an FMSUB on arm64.** The
> product and the subtraction happen at infinite precision and only the result is
> rounded; clang does not fuse the C floor's equivalent, so the two halves diverge
> the moment `q*y` leaves the exactly-representable range.
> `9007199254740991 % -3` is **-2 under `llvm.Run` and -1 natively**, and real
> ruby says -2 - the fused answer is the CORRECT one, so this is layer 2's defect
> every time. It is invisible to `./test.sh` (each engine is compared with itself)
> and to a small ratchet (the operands have to be large). The check is
> `grep -n 'math.Floor(.*)\*' abnf/jsrt*.go`; the fix is a Dekker two-product /
> two-sum emulation of the FMA, written out in `languages/lib/ruby-rt.metajs` as
> `rbFmaSub` and language-neutral. **Scale the operands down by a power of two
> before the split** - a Float remainder legitimately has `|q|` near DBL_MAX.

**A FIFTH, in the same family: `strconv.FormatFloat` is EXACT and rounds HALF TO
EVEN.** `Math.floor(a * 10^prec + 0.5)` is wrong in both halves of that sentence -
it rounds half AWAY from zero (Go's `%.0f` of 2.5 is "2") and it cannot reach the
digits past 2^53 that `%f` of 1e100 needs. The exact decimal of a double is a
bignum (`m * 2^e` is `m` doubled `e` times, or `m * 5^-e` with the point moved),
and `ruby-rt.metajs` has it in 70 language-neutral lines. **`printf` width and
precision also count BYTES**, not characters, because Go's are `len(body)` and
`body[:prec]` - `byteLen` is the floor's own answer.

### What KOTLIN added to all three, and it is the LIMIT case (2026-08-03)

**Item 1 stops being cheap, and the number is the news.** php lowered 2 externs,
swift 2, java 1 helper, csharp 0, dart 2, go 1 - and **kotlin has TEN**
(`js_ktget`, `js_kset`, `js_ktvarset`, `js_ktmextget`, `js_ktmextset`,
`js_ktmextcall`, `js_ktouter`, `js_ktrefbase`, `js_ktdthis`, `js_ktbareref`),
with `core.scopeGetExt = "js_ktget"`, so EVERY variable read went through one.
Nine are lowered in the emitter the usual way and `sw_safeget` transferred
verbatim for the fourth time. What the earlier seven never hit is the ceiling:

> **An emitter can read the INNERMOST binding, the INNERMOST `this` and the
> INNERMOST `__dispatch`, and no further.** `languages/lib/runtime.c` seeds no
> scope-PARENT accessor, so a chain WALK - which is what `js_ktget` and
> `ktMemberExtFind` do - cannot be expressed in IR at all. Every one of the nine
> helpers therefore ends in the extern it replaced.

**And item 1 has a MIRROR IMAGE here, which is new.** Kotlin's implicit-receiver
stack (`with` / `run` / `apply` / `buildString` / `buildList` / `buildMap`, and
every `T.() -> R` lambda) lives inside the RUNTIME, so this time it is the
EMITTER that cannot see the private representation. Go's rule - "find the floor
extern that already answers it and lower the call" - has no answer, because no
such extern exists. The way through costs nothing and generalises: **find the
extern with the right SHAPE and give layer 2 the extra arm.**
`js_ktglobal(name)` takes one NAME and no scope, so the emitted `kt_ktget` asks
it before giving up, and `kotlin-rt.metajs`'s `js_ktglobal` answers from the
receiver stack first. In the Go twin `ktGlobalFn` answers `jsUndef` for an
unknown name, so the probe is a **no-op under `llvm.Run`** - measured, the
ratchet stayed at 979 - and it is what makes `with(list) { size }` and
`mutableListOf(1, 2).apply { add(3) }` work natively.

**Item 2: kotlin is the sixth exception.** `grep -n truthyExt
languages/kotlin-to-llvm-ir.abnf` is EMPTY. The tally over seven ported
languages is **lua and php need `__raw`; swift, java, csharp, dart, go and
kotlin do not**. `_raw` PARAMETERS are now **3 / 0 / 0 / 1 / 1 / 0 / 1 / 1** for
lua / php / swift / java / csharp / dart / go / kotlin; kotlin's one is
`js_char`'s code point.

**Item 3 cannot bite kotlin either, and that is now four in a row.**
`grep -c 'js_genfn\|js_yield' languages/kotlin-to-llvm-ir.abnf` is **0**:
`suspend` is lowered by the emitter and `launch`/`runBlocking` run their block
eagerly. Dart, go, csharp's emitter and kotlin have all answered zero here -
**run the grep before costing a coroutine**.

**A FIFTH thing, and it sharpens the `si_norm` question the Dart section
posed.** Dart asks "is the language's unboxed representation the one `si_norm`
produces?" and answers yes for itself, no for Java. **Kotlin is a THIRD answer:
its boundary is 32 bits.** `abnf/jsrtkotlin.go`'s `ktNorm` says so in its own
comment - "It is deliberately not giNorm" - and unboxes only a signed 32 bit
value, so a Kotlin `Long` is a box at EVERY magnitude even though it is honestly
SIGNED. `sintConv` ends in `si_norm` and would unbox a small Long, and `ktWU`
reads the result type off the operand boxes, so `1000000L * 1000000L` would
become a 32 bit multiply - Java's wall reached without Java's trick. `sintRaw`
is the fix, C#'s `csNorm` recipe unchanged. **Ask the next language WHERE its
unboxing boundary is, not whether it has a 64 bit type: go and dart say 64,
kotlin says 32, java says nowhere, and `sintRaw` answers all three of the
non-go cases.**

### What PYTHON added, as the LAST of the nine (2026-08-03)

**Item 1 scales, and python is the proof: SIX externs lowered, not one.** The
running count is php 2, swift 2, java 1, csharp 0, dart 2, go 1, ruby 1, kotlin 1
- and **python 6**: `js_pyfnscope`, `js_pyset_var`, `js_pyglobal`,
`js_pynonlocal`, `js_pydel_var` and `js_pyannot`, plus `js_pyclass_fill`, which
ENUMERATED a scope rather than probing one. `runtime.c` had already written the
prediction at its own `js_pyset_var` - *"A Python layer 2 will have to add
them"* - and layer 2 could not. What made six affordable rather than six times
the work is a mechanism the first eight did not need:

> **Carry the private mark as an ORDINARY BINDING, in a scope of its own.**
> Insert a `MID` scope above every binding boundary and declare
> `"pyfn*" -> the boundary scope` and `"pyup*" -> MID` in it. `js_scope_get`
> already walks the chain, so from any nested scope `"pyfn*"` resolves to the
> nearest enclosing boundary - which turns "which scope does this bind in" from a
> floor question into two `js_scope_typeof` calls. A name that cannot be a source
> identifier (`*` in it) is the whole trick, and it generalises to any per-scope
> flag a future language wants.

Two warnings that came out of doing it at that scale, both MEASURED:

* **A slot that HOLDS undefined reads as "undefined".** Swift and Ruby recorded
  this as a footnote; at six lowerings it is a hard failure. Python's
  `declCompNames` declares a comprehension target holding undefined, and the
  first version of `py_setvar` therefore bound it in the wrong scope -
  `[i*i for i in range(4)]` answered `[NaN, NaN, NaN, NaN]`. The fix is that
  **the emitter already knows those names statically**: record every name the
  emitter DECLARES into a non-boundary scope of the current function and emit the
  floor's own walk for those. If you lower a scope rule, enumerate what your own
  emitter puts in scopes before you trust a `typeof` probe.
* **A value the floor cannot represent needs a SENTINEL and a guarded READ.**
  `del x` leaves a private Go value in the slot and `abnf/jsrt.go` raises from
  inside `scopeGet`. One unique array created at module start, compared with
  `js_seq`, and EVERY read routed through an IR helper. There were TWO read
  paths, not one - `makeVarRef` and `emitRefRead` (an augmented assignment reads
  first) - and routing only the first left two assertions red, which is how the
  second was found.

**Item 2: python is the THIRD language that needs `__raw`**, after lua and php,
and the first since. The tally over nine is **lua, php and python need it;
swift, java, csharp, dart, go, ruby and kotlin do not.** Python needs it because
its truthiness is genuinely its own (an empty container is false), so the floor's
`js_truthy` cannot serve. It also has a SECOND `NewICmp` on the same extern's
result, for `not x` - `grep NewICmp` over the grammar is still how to find them,
and both sites are covered by the one `__raw` suffix.

**Item 3 is answered YES for the first time, and the split is what matters.**
`grep -c 'js_genfn\|js_yield' languages/python-to-llvm-ir.abnf` is **6**, after
five languages in a row answered 0. C#'s "layer 2 owes zero lines" held for the
DRIVING side - the emitter lowers it over floor externs - but the INSPECTION
surface (`send` / `next` / `close` / draining) is the language's own and does
cost layer-2 lines. PHP's two findings both applied unchanged: the close flag
lives in an identity-keyed side table because a generator is a floor CELL, and
"is this a generator" is STRUCTURAL because layer 2 has no tag test - the floor
answers exactly one member on a tag-15 cell, `"next"` (`runtime.c:2868`).

**Ruby's FMA warning: run it, and accept a NO.** `grep -n 'math.Floor(.*)\*'
abnf/jsrt*.go` finds three sites and all three are Ruby's. Python's `%` is
`pyFloorMod` over `math.Mod`, which is not the fusable expression, so no Dekker
emulation is owed. The grep is cheap and the answer is not always yes.

**A SIXTH thing, and it is the last genuinely new question this plan had left:
an ARBITRARY PRECISION integer.** Every previous language's number fitted a
double, a sized-integer tag or a box. `10000000000000000000000000001` fits none
of them, and the floor has no bignum. The answer is a base-10000 little-endian
bignum in MetaJS (~330 lines with division), and the two things that generalise
are: a Go TYPE SWITCH ported into this value model changes ARM ORDER, because a
`*jsBigInt` is never a `*jsObject` and a `{__pybig}` IS a plain object (the
bigint arm has to move ahead of the array/object arms of `jsAdd` or `10**28 + 1`
string-concatenates); and `rt.toNumber` has no bigint arm, so every path that
coerces one must answer NaN - which is why `pyNum(v)` is written `v * 1` and not
a hand-written cascade.

### What JS and TYPESCRIPT added to all three (2026-08-03)

**Item 1 has a FOURTH shape, and it is the one that runs out of floor.** php/swift/
dart/ruby lowered a SCOPE probe and go lowered a CONTROL-SIGNAL probe; both worked
because the floor already had an extern that opened the private cell. `js_this` /
`js_newtarget` read `abnf/jsrt.go`'s **CALL STACK**, and the floor keeps no such
state at all - `runtime.c`'s `js_call` drops `self` for a tag-7 closure. There was
nothing to lower onto. The answer that replaces "find the floor extern":

> **If the floor keeps no such state, LAYER 2 keeps it and the EMITTER hands it
> over.** Layer 2 is a module with globals of its own. One extern call,
> `js_jssetthis(receiver)`, emitted immediately before every call site the emitter
> owns. No restore is needed, because `this` is read exactly once per function, at
> entry.

The same move then closed three more walls of a different kind - the accessor box
(`js_defprop`), `typeof` on a BigInt, and the whole `===`/`!==`/unary-`-`/`!`/truthy
family - all of them cases of *the Go runtime represents this as a Go TYPE and layer
2 can only represent it as an object*. Generalised:

> **Route the OPERATOR through layer 2, in the EMITTER, under `-exe` ONLY.** It
> needs no floor change and no Go twin (the module that declares the new extern is
> only ever built by `-exe`), and because it is gated on `c.exePath` the module
> `llvm.Run` executes never moves. **Diff the non-native module against a clean
> `git archive` of the base commit before and after** - that turns "will the matrix
> survive this" from a hope into a measurement and costs one `diff`.

**Item 2 gets a clause.** js/ts are the first language where the `__raw` answer
DIFFERS between the two engines: `grep 'truthyExt *='` is empty, so `llvm.Run` keeps
the floor's own `js_truthy` and needs no shim - but the NATIVE build overrides
`core.truthyExt` for BigInt and that shim needs the suffix. **Grep for the
assignment, not the read, and grep inside the `-exe` arm too.** `_raw` PARAMETERS:
js and ts have **zero**.

**Item 3 (generators) cost layer 2 nothing on the driving side, as C# predicted** -
both grammars emit `js_genfn` / `js_yield` seven times and step the generator with
`js_get(g, "next")` + `js_call`, all floor externs. What they DID owe is
`js_iterable`, the for-of drain, which has to loop `next()` until `done` because the
floor has no `g.drain`; and "is this a generator" is still the STRUCTURAL probe php
described, because layer 2 has no tag test.

**A FOURTH thing, and it is the biggest lever in part 3.** Ruby found that a layer-2
file may `import` a shared MetaJS source. The general form, proved here, is:
**every language in this repo already has a complete MetaJS implementation of its
own semantics, in `languages/<lang>-interpreter.abnf`'s start script.** js's
`jsStr` / `jsToPrim` / `jsLooseEq` / `jsBuiltin` / `jsStringMethod` / the whole `bi*`
BigInt family / the entire regex glue over `lib/regex.js` were all already there, in
the same frozen subset, kept in agreement with the Go twin by the matrix for a long
time. **Grep the interpreter grammar before porting a Go twin.** What cannot be
lifted is only what touches the VALUE MODEL - the interpreter's objects are its own,
layer 2's are the floor's handles.

## PHP - DONE (2026-08-03). The gate is MET: 306/306 natively, byte-identical

**`tests/php-test-full.php` runs as a self-contained native binary and prints
`full: 306 checks, 0 failures`, byte-identical with `llvm.Run` on stdout, on
stderr and in its exit code.** The row in `tests/clang-check.sh` is no longer
held: php reads `ok, and the clang executable agrees`.

```
mec languages/php-to-llvm-ir.abnf tests/php-test-full.php -q -exe /tmp/php-native.out
/tmp/php-native.out ; echo $?          # full: 306 checks, 0 failures / 0
```

This finishes the migration that stopped at 221/306 earlier the same day. Nothing
about PHP changed to get the other 85: **the two walls that blocked them were
FLOOR walls, both of them landed in the floor, and layer 2 over each is small** -
which is the result that carries to the remaining nine, since every one of them
meets at least one of the two.

| what closed it | ratchet | layer 2 cost |
|---|---|---|
| `keysOf`, host id 61 (commit `d30629f`) | sections 17, 20, part of 30 - ~75 assertions | 4 sites, ~45 lines |
| stackful coroutines, tags 15/16 (commit `94ac7b8`) | section 19 - 10 assertions | 1 predicate + 1 side table + `phGenCall`, ~120 lines |

The estimate in the cost table below said ~80 lines for php's coroutine half and
"~60 lines of MetaJS" for `keysOf`'s. Both were about right; the total diff to
`languages/lib/php-rt.metajs` is +263/-58 lines including its comments.

### THE THREE WALLS ARE ALL CLOSED, and what each one taught

**WALL 1 - coroutines.** The floor owns the suspension and exposes exactly ONE
member on a generator, `next(v) -> {value, done}`. Everything PHP's `Generator`
class adds is the INSPECTION half of the protocol - `current` / `key` / `valid` /
`rewind` / `send` / `getReturn`, plus the auto-key a keyless `yield` gets - and it
answers from the LAST handshake instead of performing one, so reading a generator
twice must not advance it. That half is `phGenCall` in `php-rt.metajs`, written
function for function against the `*jsGenerator` case of `getMember` in
`abnf/jsrt.go`. Three findings, and all three are language-neutral:

1. **A generator is a floor CELL, not an object, so the cursor cannot be stored ON
   it.** `set_member` refuses tag 15 and `get_member` answers only `next`. The
   per-generator state therefore lives in one side table keyed by IDENTITY (two
   parallel arrays and a `===` scan, the shape `phDumpSeen` already used, because
   the frozen subset has no `Map`). **Every language whose generator exposes an
   inspection surface needs this table**: php's `current`/`key`/`valid`/
   `getReturn`, C#'s `Current`/`MoveNext`, python's `send`. Only js/ts escape it,
   because `next()` IS their whole protocol. Cost: ~25 lines, and a step is O(n)
   in the generators the program has ever made - the same leak shape as the Go
   twin's parked goroutine.
2. **Layer 2 has to RECOGNISE a generator, and there is no tag primitive for it.**
   `php-to-llvm-ir.abnf` emits `js_phisa(v, "Generator")` for every `foreach` and
   every `yield from`, so `phIsA` must answer for a value whose `typeof` is
   `"object"`. The probe that works is structural: exclude the array box
   (`__dict`), the reference cell, the class descriptor, a raw list (`length` is a
   number) and an instance (`__class`), and then ask whether `v["next"]` is
   callable - which on tag 15 is the bound `gen_next` and on a tag-6 object is
   `undefined`, because a PHP method lives on the class descriptor and never as an
   own property. It is exact for PHP, but it is a STRUCTURAL probe standing in for
   a type test. **A floor `js_tag(h)` (or an `isGenerator` host builtin) would
   make it a one-liner and would remove the one place this port guesses**, and the
   next language whose instances can carry a callable own property named `next`
   will need it. Reported, not made: the floor is another agent's.
3. **A throw across a yield needed NOTHING here.** `g.next()` raises on the
   caller's stack, so `foreach` has no special case - as the floor section
   promised. Verified in the probe: a generator that throws after its first yield
   behaves identically under `llvm.Run` and natively, whether it is driven by
   `foreach` or by an explicit `->next()`.

**WALL 2 - an object's own keys.** `keysOf` answers an object's own keys in
INSERTION order and already skips the `__`-prefixed slots, which is
`phpHiddenProp`'s rule in the Go twin exactly - so `js_phdump`, `phObjPropsEq`,
`js_phcast("array")` all read it straight, with no filtering of their own. **One
site is not straight, and it is the generalisable trap:** `clone` has to keep the
instance's class, and `__class` is precisely the kind of slot `keysOf` hides. The
Go twin walks `o.keys` raw and copies it by accident; layer 2 has to put it back
by hand. `__class` is the only hidden slot `php-to-llvm-ir.abnf` ever writes on an
INSTANCE (everything else `__`-prefixed lives on the class DESCRIPTOR), so one
line covers it - but **every language that clones or copies an object through
`keysOf` has to ask itself which hidden slots the copy must keep.**

**WALL 3 - the scope chain.** Unchanged and still the general answer: **a scope
probe belongs in the EMITTER, not in layer 2.** `php-to-llvm-ir.abnf` lowers
`js_phthis` and `js_phclslookup` through one module-level
`php_scope_probe(env, name, dflt)` over `js_scope_typeof` / `js_scope_get`, and
`js_phrtinit` receives a plain VALUE (a PHP array of name => descriptor) built by
the emitter, which is the side that may read the scope.

### A FOURTH trap, and a FIFTH - the diagnostic that lies

The fourth is unchanged: **an extern that returns a RAW integer** needs the
`__raw` suffix (`function js_phtruthy__raw(v)`), or every condition in the program
is taken. See "TWO THINGS EVERY ONE OF THE REMAINING NINE NEEDS".

The fifth was measured here and **will hit all four remaining coroutine
languages**, because each of them has loud stubs in layer 2 today for exactly the
symbols the floor has since defined:

> **A DUPLICATE symbol is reported as an UNRESOLVED one.** With `js_genfn` and
> `js_yield` defined in BOTH `lib/runtime.ll` and `lib/php-rt.ll`, the build
> printed
>
> ```
> error: 2 unresolved symbol(s), and this build links a runtime, so they are NOT stubbed:
> error:     js_genfn
> error:     js_yield
> ```
>
> while the actual `ld` error was `2 duplicate symbols`. `buildExecutable` in
> `abnf/llvmmap.go` builds its report from the MODULE's undefined declarations
> filtered by `mentionsSymbol(clang output, name)` - and a duplicate-symbol error
> mentions the very same names, so every one of them passes the filter. The report
> is deliberately built from the module rather than the linker text (the linker
> text carries temp file names and this path is in the byte-compared matrix), so
> the fix is not obvious; the cheap one would be to say "unresolved OR duplicate".
> **Until then: when layer 2 and the floor both define a symbol, the message says
> the opposite of what is wrong.** Not fixed here - `abnf/` is another agent's.
>
> **FIXED 2026-08-04.** `duplicateSymbols(clang output)` in `abnf/llvmmap.go` reads
> the blamed names out of the two linker spellings - ld64's `duplicate symbol
> '_name' in:` and GNU ld's `multiple definition of \`name'` - sorted and
> deduplicated, taking ONLY the quoted symbol so no temp file path can reach the
> report and this path can keep comparing bytes. It is asked FIRST, before the
> undefined-symbol report, precisely because that report cannot tell the two apart.
> The message now says "N symbol(s) are defined MORE THAN ONCE among the linked
> inputs ... this is NOT an unresolved symbol, and adding a definition would make it
> worse". Reproduced end to end with two `-rt` inputs both defining `mec_helper`
> (before: `1 unresolved symbol(s)`), and pinned by `abnf/linkdiag_test.go`, which
> also asserts that `mentionsSymbol` still matches a duplicate diagnostic - the
> overlap is what justifies the ordering, and the test fails the day someone
> "fixes" `mentionsSymbol` instead. From a clean archive of ad922a0 the test does
> not compile: `duplicateSymbols` did not exist.

### The extern split, after the two walls

`tests/php-test-full.php` drives **100** externs; over every `tests/php-test-*.php`
the union is **106**.

| where | count | what |
|---|---|---|
| the C floor | 31 | the generic primitives, **plus `js_genfn` and `js_yield`** |
| **layer 2**, language-NEUTRAL | 6 | `js_bytelen`, `js_jadd`, `js_mcall`, `js_range_len/key/val` |
| **layer 2**, PHP's own | 69 | `js_ph*` |

`languages/lib/php-rt.ll` exports **77** symbols: the 75 the union needs plus two
loud stubs for `js_phthis` / `js_phclslookup`, which the emitter lowers itself.
The six neutral ones still belong in the floor for everyone and are here only
because the floor is another agent's this session.

### Ground truth

Measured in the working tree at `94ac7b8` plus this change, and re-measured after
the floor moved under it (`runtime.ll` was regenerated by another agent
mid-session; the whole sweep below is the SECOND run, against that floor).

```
matrix                325/325
--full                5,615 assertions, 0 languages whose halves disagree (php 306)
                      grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF'  ->  no hits
--cross               119 programs, 0 divergent, 0 differing only in warnings
go test ./abnf/       ok
clang-check           16 modules, ALL accepted, suite GREEN:
                        php   51085   ok, and the clang executable agrees
every gen-*-rt-ll.sh --check clean: runtime, bash, batch, csharp, dart, java,
                      lua, php (php-rt.ll, 30,488 lines) and swift
```

Native vs `llvm.Run`, stdout **and** stderr **and** exit code, over every php test:

```
php-test-1  php-test-features  php-test-complete  php-test-try  php-test-multifile
php-test-full                                                 all SAME (rc=0)
php-test-undefconst                       SAME (rc=1); the runtime-error line is
                                          identical, llvm.Run's driver adds one
                                          "  ==> Fail" line of its own
```

### The differential probe, and the ONE divergence left

The probe was regenerated and WIDENED to the surface the two walls opened: on top
of `+ - * / % **` over 51 values (0, +-1, both sides of 2^53, `PHP_INT_MAX`/`MIN`
and their neighbours, the int32 edges, integral and non-integral floats, 1e100,
1e-100, +-INF, NAN, the booleans, null and sixteen numeric and non-numeric
strings), all nine comparisons including `<=>` over the same 51, the five bitwise
operators and `intdiv` over 13 integers, the unary operators, the five casts, the
seven `is_*` predicates, `gettype`, `get_debug_type` and array-key normalisation,
it now also drives **`var_dump` of objects, `==` / `===` between objects, `(array)`
on an object, `clone` with a `__clone` hook, and eighteen generator programs** -
keyed and auto-keyed yields, `yield from` over an array and over a generator, an
infinite generator broken out of, a generator that throws after its first yield,
the whole `current`/`key`/`valid`/`getReturn` cursor, and a generator driven both
by `foreach` and by hand.

**40,948 lines of PHP, 41,409 lines of output, and the two engines differ on 431
hunks - 687 output lines - with exactly ONE cause: the numeric string `"1e400"`.** Every diverging
expression has `"1e400"` as an operand and none of the 687 has any other; nothing
in the object or generator surface diverges at all.

```
                              php-interpreter.abnf   llvm.Run (jsrtphp.go)   native
var_dump(0 + "1e400")            float(INF)               int(1)            float(INF)
var_dump((float)"1e400")         float(INF)               float(1)          float(INF)
var_dump(is_numeric("1e400"))    bool(true)               bool(false)       bool(true)
```

**Two of the three engines agree, and they are the two that agree with real PHP**
(`(float)"1e400"` is `INF`, and `is_numeric` is true for any numeric string
regardless of range). The odd one out is `abnf/jsrtphp.go`, whose numeric-string
reader rejects an out-of-double-range exponent and falls back to the leading-digit
parse, giving `1`. **This is invisible to every existing suite**: `./test.sh`
compares each engine against itself, `--cross` compares the two Go-backed halves
on programs in `tests/` and no test file contains such a literal, and the ratchet
is 306/306 in both halves. Reported rather than fixed - `abnf/jsrt*.go` is another
agent's this session, and `jsrtphp.go` is scheduled for deletion anyway (item 4
below), which would make the defect vanish with it.

**The `d_pow` `-0` defect this probe found in the morning is FIXED**: `**` over
all 51x51 pairs, which includes `Math.pow(-0.5, 2147483647)`'s PHP spelling, is
now identical in both engines.

There is still **no `php` binary on this machine** (`which php` -> not found), so
real PHP could not be the oracle for the run itself; the `"1e400"` reading above
is settled against the language specification and against the third engine.

### What is still owed for PHP

1. **`abnf/jsrtphp.go` can be deleted**, as the plan requires once the native path
   is the real one. It is what `llvm.Run` uses today, so deleting it means the
   compiler half runs through the same `php-rt.ll` the native binary does. That
   also retires the `"1e400"` divergence above.
2. The **six language-neutral externs** in `php-rt.metajs` (`js_bytelen`,
   `js_jadd`, `js_mcall`, `js_range_len/key/val`) belong in the C floor, where
   every language can reach them.
3. A floor **`js_tag` / `isGenerator`** primitive, so WALL 1's finding 2 stops
   being a structural probe.
4. `send()` is implemented here and the ratchet does not exercise it (section 19
   says so at the site): PHP's `send` primes a fresh generator, resumes it and
   answers the value yielded NEXT. It is written against the Go twin and is
   covered by neither engine's tests - the first language that needs it (python)
   should re-derive it rather than trust it.

### launch.json entries wanted (not added - not my file)

```
"PHP native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/php-to-llvm-ir.abnf", "tests/php-test-features.php", "-q", "-exe", "tests/php-native.out"]
"PHP native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/php-to-llvm-ir.abnf", "tests/php-test-multifile.php", "-q", "-exe", "tests/php-native-multi.out"]
"PHP native binary, FULL ratchet (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/php-to-llvm-ir.abnf", "tests/php-test-full.php", "-q", "-exe", "tests/php-native-full.out"]
```

All three pass natively today. The full-syntax one was deliberately NOT requested
before, because it would have been red; it is green now, and it is the entry that
would keep the whole native path in a green-only suite.

## SWIFT - DONE (2026-08-03). The gate is MET: 209/209 natively, byte-identical

**`tests/swift-test-full.swift` runs as a self-contained native binary, all 209
assertions pass in it, and its stdout, its stderr and its exit code are
byte-identical with `llvm.Run`.** Swift is in `tests/clang-check.sh`'s run-it row:

```
swift           26139  ok, and the clang executable agrees
```

No row is held, no assertion is held back, and no floor change was needed or made.
`abnf/jsrtswift.go` is **kept**, as the plan requires: `llvm.Run` still uses it,
the native binary uses layer 2, and the two are now differentially compared over
20,296 probe lines.

Swift was unblocked by **both** floor tags at once and needed no new code for
either. This is the result that matters for the remaining eight:

> PHP could go second only because its Go twin uses zero `jsJFlo`, and it still
> needed a hand-written `{__pf}` object box for its float; Lua needed ~200 lines
> of `i64` pair arithmetic for its integer. **Swift needed NEITHER.** `sint*` IS
> Swift's `Int8...UInt64` and `flo*` IS Swift's `Double`, so
> `languages/lib/swift-rt.metajs` has no pair layer and no box layer at all - the
> two tags are simply used. Both tags paid for themselves the first time a
> language reached for them together.

### The extern split

`tests/swift-test-full.swift` drives **66** externs (was 68; see the scope-probe
lowering above). Over every `tests/swift-test-*.swift` the union is **also 66** -
the smaller files add nothing.

| where | count | what |
|---|---|---|
| already in the C floor | 35 | the generic primitives of phase 3, the five the Lua pilot added, `js_truthy`, and the `sint`/`flo` machinery behind `js_gilit` |
| **layer 2**, language-NEUTRAL | 10 | `js_char`, `js_char_code`, `js_kindex`, `js_dict_new`, `js_is_type`, `js_jadd`, `js_mcall`, `js_pyget`, `js_pyset`, `js_pylen` |
| **layer 2**, Swift's own | 21 | `js_sw*` |

`languages/lib/swift-rt.ll` (17,718 lines, from a 1,366-line `.metajs`) exports
exactly those 31 symbols and nothing else - checked, not assumed. The layer-2
source compiles **byte-identically under `-frozen`** as well as under goja, which
is the check that says the dialect rules at the head of the file were actually
obeyed rather than accidentally satisfied by the host engine.

The split against the two done before it:

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
```

**Swift's own 21 is the smallest language-specific surface of the three**, and the
reason is the value model, not the language: Swift's runtime semantics are
rendering (`js_swdesc`/`js_swstr`/`js_swprint`), value equality (`js_sweq`),
arithmetic selection between the two number tags, and four comparisons. There is
no collection library and no string library in it - `js_mcall`, `js_pyget` and
friends carry those, and they are neutral.

### `js_jadd` is now COMPLETE in layer 2, and `jsChar` did NOT block Swift

Two things the floor is missing came up and neither cost anything:

**`js_jadd`.** `php-rt.metajs` had to leave its float arm unwritten (that arm needs
`jsJFlo`, which PHP's layer 2 could not make). Swift's is complete: the boxed
double is a floor tag, so the arm is three lines of `flo`/`floIs`/`floStyle`. **Any
language that emits `js_jadd` on real user values can now have the whole thing.**

**`jsChar`.** The plan says to REPORT rather than add if Swift needs it, and the
honest answer is that **Swift does not**, for a reason worth writing down because
the next language may not get it for free. `js_char`, `js_char_code` and
`js_kindex` are Kotlin's Char externs and the floor has no such tag - but Swift
reaches all three from exactly two places, both inside the emitter's own
grapheme-cluster walk (`sw_gcount` and `sw_nfd`), and in both the `jsChar` is
consumed immediately:

```
js_jadd(acc, js_char(cp))          Go: jsChar -> rt.toString  -> the character
js_char_code(js_kindex(s, i))      Go: jsChar -> rt.toNumber  -> the code point
```

A one-character STRING makes both of those round trips identically, so layer 2
answers a string and nothing observable changes. `js_kindex` additionally has to
map a LONE SURROGATE to U+FFFD, because the Go twin builds its Char from
`[]rune(strAt(s,i))[0]` and `sw_gcount` recognises a surrogate pair by exactly
that value - the engine would otherwise hand back the surrogate itself. **No floor
change requested.** A language that puts a Char in a variable, compares two of
them, or asks `js_typeof` about one will not be able to do this, and then the tag
is a real gap.

### Ground truth

Oracles: **real `swift` / `swiftc` 6.1.2 (Apple Swift, arm64-apple-macosx16.0)**,
present on this machine, and **Apple clang 17.0.0** for the native link.

```
matrix                325/325
--full                5,590 assertions, 0 languages whose halves disagree (swift 209)
                      grep 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF' -> empty
--cross               119 programs, 0 divergent
go test ./abnf/       ok
jsbootstrap.ll        regenerates BYTE-IDENTICALLY (md5 1e7c3ffc... before and after)
gen-runtime-ll.sh --check   up to date        gen-lua-rt-ll.sh   --check  up to date
gen-bash-rt-ll.sh --check   up to date        gen-batch-rt-ll.sh --check  up to date
gen-php-rt-ll.sh  --check   up to date        gen-java-rt-ll.sh  --check  up to date
gen-swift-rt-ll.sh --check  swift-rt.ll is up to date (17,718 lines)
clang-check           swift joins the RUN-IT row (see above). The one finding in the
                      table is `java`, which is another agent's work in progress in
                      this tree and is not touched here.
```

**Verified from a CLEAN ARCHIVE, not the working tree** - `git archive d30629f | tar
x`, then only the four Swift files copied in, `go build` INSIDE, run there:

```
tests/gen-swift-rt-ll.sh --check    swift-rt.ll is up to date
./sw.exe                            full: 209 checks, 0 failures   (rc=0)
tests/clang-check.sh swift          ok, and the clang executable agrees
```

That matters here for the reason the Swift history gives it: `js_swtuple` /
`js_swtuparr` were once committed while `abnf/jsrtswift.go` was not, and a clean
checkout failed three matrix entries. Nothing in this change depends on the
concurrent `runtime.c` edits in the working tree.

Every Swift program in `tests/`, `llvm.Run` against the native binary, stdout AND
stderr AND exit code:

```
swift-test-1  swift-test-complete  swift-test-features  swift-test-full
swift-test-multifile  swift-test-try                              all SAME (rc=0)
swift-test-recognize   NOT COMPARABLE - both halves refuse it identically
                       ("await expression not implemented", a pre-existing gap)
```

### The differential probe - two probes, 20,296 lines, and what they found

**Probe A, 1,910 lines**: `Double.description` over 57 values (both zeros, the
subnormal minimum, `Double.greatestFiniteMagnitude`, 1e±100, the 1e15/1e16
boundary, 2^53 and its neighbours), the four arithmetic operators and
`truncatingRemainder` over pairs of them, all five comparisons, the eight Double
predicates, `&+ &- &* & | ^` and the six comparisons over 28 integers,
`/ %` guarded, shifts including Swift's NEGATIVE counts, `~`, the failable
`Int(String)` / `Double(String)` over 28 strings, `max`/`min`, and the rendering
and `==` of struct, class, enum (bare, associated, labelled, raw-valued), tuple
(labelled and nested), Array, Dictionary, Optional and `CustomStringConvertible`.

**Probe B, 18,386 lines**: the integer and Double halves again with every operand
read out of an ARRAY at run time, so `swiftc -O` cannot fold it. 31x31 integer
pairs times `&+ &- &* & | ^ ~ / % << >>` at thirteen shift counts plus seven
comparisons, and 32x32 Double pairs.

```
llvm.Run vs the native binary      BYTE-IDENTICAL on both, 20,296 of 20,296 lines
```

It found **one real defect**, in probe A, on one line: `(-0.0).rounded()` answered
`0.0` natively and `-0.0` under `llvm.Run` and under real swift. `Math.round` is JS
half-UP, not Go's `math.Round` half-away-from-zero, so layer 2 spells the rounding
itself - and `x < 0` is FALSE for `-0.0`, so the sign was lost. Fixed, with two
further guards the same site needs (`|x| >= 2^52`, where `x + 0.5` rounds up to the
next double, and the infinities). The same class bit the float renderer: **unary
minus has to be written `-x` and not `0 - x`**, because `0.0 - 0.0` is `+0.0` while
the floor's `d_neg` flips the sign bit. That is the plan's own "a negative floating
point literal does not survive the C floor's compiler" trap, met from the other
side, and every remaining language with a float will meet it.

### Where our Swift and REAL swift differ - measured, and NOT fixed here

With both probes run through `swiftc -O` 6.1.2, **84 lines of 20,296 differ**, and
`llvm.Run` and the native binary agree with each other on ALL of them - so every
one is pre-existing behaviour of both halves, unchanged by this port.

```
  36 + 6   the Double FORMATTER, at one boundary (below)
      21   Optional(x) - the deliberate no-Optional-box simplification
      14   NOT OURS: swiftc constant-folded `&*` on literals at arbitrary
           precision (9007199254740991 &* itself printed as 8.1e31). The
           opaque-operand probe has ZERO integer differences in 18,386 lines,
           which is what proves the arithmetic and not the first probe.
       3   max/min tie-breaking between +0.0 and -0.0
       4   module qualification (main.C) and a non-conforming `description`
```

**The formatter boundary, measured exactly.** Both halves switch to scientific
notation at `e10 >= 16`; real Swift switches at **|v| > 2^53**:

```
                        ours                      swift 6.1.2
9007199254740991.0      9007199254740991.0        9007199254740991.0
9007199254740992.0      9007199254740992.0        9007199254740992.0     <- 2^53
9007199254740994.0      9007199254740994.0        9.007199254740994e+15  <- first
9999999999999000.0      9999999999999000.0        9.999999999999e+15
1234567890123456.0      1234567890123456.0        1234567890123456.0
1e16                    1e+16                     1e+16
```

Not fixed, for the same reason the Lua formatter was not: the rule lives in THREE
halves at once (`swFloDigits` in `abnf/jsrtswift.go`, `swFloStr` in
`swift-interpreter.abnf`, `swFloDigits` here) and `--full` SECTION 25 pins the
current one, so changing it is a self-contained job of its own and is not this
gate. It is a rendering difference only - the arithmetic agrees on all 18,386
opaque-operand lines.

**`min` between +0.0 and -0.0.** Swift's `min(x, y)` is `y < x ? y : x`, so
`min(0.0, -0.0)` is the LEFT operand `0.0`; the Go twin's `swPick` answers the
right one on a tie. 3 lines, both halves agree, pre-existing.

**A separate probe of the string model** - grapheme-cluster `.count`, canonical
`==` between `"café"` and `"cafe\u{301}"`, and `for c in s` over a non-ASCII string
- agrees with `swift 6.1.2` line for line, and `llvm.Run` and the native binary are
byte-identical on it. One difference: an ASTRAL character prints as **two U+FFFD
replacement characters** where swift prints its four UTF-8 bytes (`aé😀b` ->
`aé\xef\xbf\xbd\xef\xbf\xbdb`). That is `wtf8Clean` on the Go side and the same
answer in the C floor, and it is **PRE-EXISTING**: verified by running the same
program under a pristine `git archive d30629f` build, whose output is byte-identical
to this one. Recorded, not fixed - it is an emitter/floor question about surrogate
pairs and not a layer-2 one.

The other three groups are documented at the top of `abnf/jsrtswift.go` as
deliberate and shared by both halves, and this port reproduces them rather than
diverging: there is no Optional box in the value model, a class prints its bare
name because the runtime has no module concept, and a `description` PROPERTY is
honoured whether or not the type conforms to `CustomStringConvertible`.

### launch.json entries wanted (not added - not my file)

Both pass natively today, at the working tree and from the clean archive:

```
"Swift native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/swift-to-llvm-ir.abnf", "tests/swift-test-features.swift", "-q", "-exe", "tests/swift-native.out"]
"Swift native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/swift-to-llvm-ir.abnf", "tests/swift-test-multifile.swift", "-q", "-exe", "tests/swift-native-multi.out"]
```

Unlike PHP's, a full-syntax entry would be green too - `tests/swift-test-full.swift`
passes 209/209 natively - but it stays out of the matrix for the same reason every
other `*-test-full.*` does: the ratchet lives in `./test.sh --full`, and
`tests/clang-check.sh` already runs the native one on every sweep.

### What is still owed for Swift

1. Nothing for the gate. `abnf/jsrtswift.go` stays until the change is committed,
   as the plan requires, and then it is the first Go twin that can actually go.
2. The formatter boundary above, if and when someone does the three-halves job.
3. `min`/`max` on a signed-zero tie, same shape, 3 probe lines.

## JAVA - DONE (2026-08-03). The gate is MET, and one floor primitive is owed

**`tests/java-test-full.java` runs as a self-contained native binary and agrees with
`llvm.Run` byte for byte, stdout and stderr and exit code, on all 278 assertions**;
`tests/clang-check.sh` reads

```
java   31460   ok, and the clang executable agrees
```

so java left the `ok (module only)` row for the run-it row and NO row is held. The
oracle was a **real `javac`/`java`, openjdk 24.0.2 (`/usr/bin/java`)**, and
`java tests/java-test-full.java` prints the same `full: 278 checks, 0 failures` line
both halves print - the ratchet file is a valid Java program, which is what makes
that check possible at all.

`abnf/jsrtjava.go` and `abnf/jsrtjvm.go` are **kept**, as the plan requires: they are
what `llvm.Run` still uses, and they are now differentially compared against layer 2.

### The extern split

`tests/java-test-full.java` drives **54** externs, and so does the union over every
`tests/java-test-*.java` - java's emitter has no test-specific arm.

| where | count | what |
|---|---|---|
| already in the C floor | 29 | the generic primitives of phase 3 plus the five the Lua pilot added, plus `js_jflo` |
| **layer 2**, language-NEUTRAL | 9 | `js_arr_new_n`, `js_char`, `js_is_type`, `js_jadd`, `js_jband`, `js_jcharat`, `js_jchareq`, `js_mcall`, `js_supercall` |
| **layer 2**, Java's own | 16 | the `js_jv*` family |

Lua was 23/5/26 and PHP 29/8/63, so **java is the smallest surface of the eleven and
it measured that way.** The nine neutral ones are not Java's - they live in
`abnf/jsrt.go`, csharp/kotlin/go/swift/dart emit most of them, and they are in layer
2 only because the floor has not got them. `languages/lib/java-rt.ll` exports **27**
symbols (the 25 the union needs plus `js_jvnum` and `js_jvstr`, which
`abnf/jsrtjava.go` registers and no test reaches).

### THE ONE WALL, and how it was closed WITHOUT a floor change

> **SUPERSEDED 2026-08-04 - read this as history.** The unsigned-box representation
> described below is gone: `sintRaw` landed, and `java-rt.metajs` now carries a
> long as an honest SIGNED cell with no `jl*` layer. See "THE COLLAPSE ONTO
> `sintRaw`" further down for what the swap traded and what it measured. The
> section is kept because the reasoning is what the next language will re-derive.

**`si_norm` unboxes a signed 64 bit value a double holds exactly, and Java's `long`
must not.** `abnf/jsrtjava.go` says it outright - `jvBox` "is 'read this as a long':
always a box, never a plain number" - because `1000000L * 1000000L` is
1000000000000 and `1000000 * 1000000` is -727379968. Every one of the eleven `sint*`
host builtins goes through `si_norm`, so **layer 2 has no constructor that answers a
small boxed long**, and roughly half of SECTION 25 would have failed silently on the
32 bit wrap.

It did not need a floor addition. `si_norm`'s unboxing arm is `w == 64 && !u`, so a
64 bit box **marked UNSIGNED** never unboxes and carries the exact 64 bits
(`si_trunc` and `si_u` are both the identity at width 64). A Java long is
represented that way, and layer 2 supplies the six operations whose SIGNED reading
the floor would then get wrong:

```
+ - * & | ^ << >>>   bit-identical signed/unsigned - sintOp direct, no wrapper
/  %                 jlDiv / jlMod: divide the magnitudes, put the sign back.
                     Long.MIN_VALUE / -1 falls out with no special case, because
                     the magnitude of MIN_VALUE has MIN_VALUE's own bit pattern
>>                   jlSar: ~((~a) >>> s) when negative
compare              jlCmp: sign bits first, then sintCmp's unsigned compare -
                     unsigned order IS signed order once the signs agree
decimal text         jlStr: "-" + sintStr(negation), and MIN_VALUE negates to
                     itself whose unsigned digits are exactly the magnitude
double reading       jlNum: negate the MAGNITUDE's reading
```

Each is exact and none is observable from a Java program; the 630 KB probe below
pins all six. **The generalisation for csharp, go, kotlin, swift and dart** - every
one of which has a 64 bit signed type - is that this trick works, and that the
alternative is one floor line:

> ~~**FLOOR REQUEST (reported, not made - the floor is another agent's this session):
> a constructor that builds a tag 13 cell WITHOUT `si_norm`.**~~ **DONE 2026-08-03 -
> `sintRaw(hi, lo, bits, unsigned)`, host id 62.** See "sintRaw" under tag 13 in
> part 2 for the API and the measurement. `java-rt.metajs` is NOT converted here
> (it is another agent's file); with it, `jlDiv`/`jlMod`/`jlSar`/`jlCmp`/`jlStr`/
> `jlNum` collapse into plain `sintOp`/`sintCmp`/`sintStr`/`sintNum` calls and the
> next five languages skip this section entirely. Cost of NOT having it, measured:
> ~90 lines of `languages/lib/java-rt.metajs`.

The **second-order cost** is the part worth reading, because it is what the next
language will trip over: a long carried as an unsigned box is misread by any FLOOR
extern that reads it numerically. Exactly one of Java's does - `js_jflo`, the
(double)/(float) cast and the `double`-declared initializer - and `(double) -1L`
answered `1.8446744073709552E19`. Fixed in the EMITTER, not the floor:
`java-to-llvm-ir.abnf` now emits `js_jflo(js_jvnum(v))`, and `js_jvnum` is Java's
own "read this as a plain double". In the Go twin `js_jvnum` converts a `jsGInt` and
passes everything else through, so `llvm.Run` is unchanged - verified byte for byte.

### The two cross-cutting findings, checked and MEASURED for java

1. **`core.truthyExt`.** `java-to-llvm-ir.abnf` sets no `core.truthyExt` at all, so
   it uses the default `js_truthy`, **which is in the floor**. Java therefore needs
   **no `__raw` export**. Checked the way the plan asks - `grep NewICmp` over
   `lib/compile-core.js` and the grammar - and `js_truthy` is the only extern whose
   result is compared as a raw integer.
2. **`_raw` PARAMETERS: java has exactly one**, `js_char`, whose code the emitter
   writes with `handle(code)`. Lua had three, PHP none. `function js_char(code_raw)`.
   Every other `handle(k)` argument in the grammar goes to `js_closure` or `js_arg`,
   which are floor externs.
3. **The scope probe.** Java's is not a language feature at all: `__jmath` (the Java
   `Math` with the double-aware abs/max/min) is a HOST GLOBAL of the Go runtime and
   `runtime.c` seeds no such name, so the emitter probes for it and falls back to
   layer 2's own descriptor. One `java_scope_probe(env, nm, dflt)` helper, PHP's
   shape unchanged. **Expect one per language, and expect it to be cheap.**

### `jsChar` was NOT needed, and that was measured rather than assumed

The floor has no `jsChar`, and adding one would have been the third primitive type.
It is not needed: `languages/java-interpreter.abnf` already models a char as a boxed
`{__char: code}`, so the shape is not invented, and the box is SAFE because of what
the emitter does with it -

```
js_typeof      2 sites, both `=== "function"`   an object box answers "object",
                                                which is not "function" either
== and !=      js_jchareq / js_not              layer 2
< > <= >=      js_jvcmp                          layer 2
switch case    js_jchareq                        layer 2
```

`js_seq` is emitted only on lengths, on `typeof` results and on class descriptors -
never on a char. So no floor extern ever sees the box. **This is the answer to
"measure whether Java truly needs it": it does not.** The residue is one honest
difference, recorded rather than hidden: a `record R(char c)`'s generated `equals`
compares fields with `js_seq`, where the Go twin's `jsChar` is a comparable struct
and the box is not. No test in `tests/` declares a char record component, so it is
unmeasured rather than wrong; `js_jchareq` is the one-line fix if one ever does.

### Ground truth

```
matrix                325/325
--full                java 278, goja and -frozen byte-identical, 0 halves disagree
--cross               119 programs, 0 divergent
go test ./abnf/       ok
clang-check           16/16, and java joins the RUN-IT row
gen-java-rt-ll.sh --check   java-rt.ll is up to date
jsbootstrap.ll        untouched (metajs-to-llvm-ir.abnf was not edited)
real java 24.0.2      `java tests/java-test-full.java` -> full: 278 checks, 0 failures
```

Every Java program in `tests/`, `llvm.Run` against the native binary, stdout **and**
stderr **and** exit code:

```
java-test-full  java-test-1  java-test-features  java-test-complete  java-test-try
java-test-annotations  java-test-widen  java-test-multifile      all SAME (rc=0)
an uncaught ArithmeticException                                   SAME message,
    rc=1 in both; llvm.Run's driver adds one "  ==> Fail" line of its own, the
    same difference php-test-undefconst records
java-test-recognize    fails to COMPILE in both (it is a recognition-only file)
```

### The differential probe, and the three defects it found

A **17,674-line / 630,291-byte** probe - `+ - * / % & | ^` over 26 int values and
over 27 long values (0, +-1, the int32 and int64 edges and their neighbours, the
2^53 boundary, 2^32, 3e9, 1e12), the three shifts over 16 counts spanning both the
&31 and &63 mask boundaries and the negative ones, all six comparisons int/int,
long/long and int/long, `+ - * / %` over 23 doubles (including both zeros, both
infinities, NaN, 1e20, 1e-20, the smallest subnormal and DBL_MAX), the mixed
int/double promotion, every cast between byte/short/int/long/char/double in both
directions, unary `-` and `~`, `++` on all six types, compound assignment on all six,
`Math.abs/max/min`, the `Integer`/`Long` statics, `MAX_VALUE`/`MIN_VALUE`/`SIZE`/
`BYTES` for all five box types, `parseInt`/`parseLong`, char arithmetic and
rendering over 13 code points, `String.charAt/length/substring/indexOf/equals/
isEmpty`, and the four integer division-by-zero throws - is **BYTE-IDENTICAL between
`llvm.Run` and the native binary**, and differs from **real java 24** on 254 lines
with ONE cause (below).

Three defects, all in layer 2, all invisible to `./test.sh` because it compares each
engine with itself and the Go twin was right in every one:

1. **A plain number narrowed through the SATURATING path.** `jvNarrow`'s Go twin
   saturates only for a `jsJFlo` (JLS 5.1.3) and WRAPS everything else; reading a
   plain `2147483648` through the saturating path answers 2147483647 where
   `int32(giVal(...))` answers -2147483648, so `2147483648 - 1` was off by one and
   `-2147483648` printed as `-2147483647`. **4 of the 278 ratchet assertions caught
   this one** (num10, int1, int10 - and flt11 for the next item), which is the
   ratchet doing its job at the native boundary.
2. **`-0.0` was lost.** `flo(0 - floNum(v), 0)` for unary minus: `0.0 - 0.0` is
   `+0.0`, the same trap `d_ldexp` records in `runtime.c` and the reason a negative
   floating point LITERAL does not survive the C floor's own compiler. Multiplying
   by -1.0 computes the sign instead of writing it.
3. **`>>>` answered an unsigned 32 bit value.** MetaJS's `>>>` is JavaScript's, so
   `-1 >>> 0` is 4294967295 there and -1 in Java, whose result type is still `int`.
   The Go twin says `int32(uint32(a) >> s)`. **Only the probe saw this** - the
   ratchet's `>>>` assertions all use operands whose top bit is clear after the
   shift.

A fourth, `jlNum`, never shipped: `sintNum(x) - 2^64` for the negative half answers
**0** for -1L, because the unsigned reading of -1 rounds to 2^64 exactly. Negating
the magnitude's reading is exact, and rounding to nearest is symmetric about zero.

~~**The 254 lines where we differ from real java are all one thing**: the smallest
positive subnormal renders as `5.0E-324` where `Double.toString` gives
`4.9E-324`.~~ **FIXED 2026-08-03, in all engines, against real java 24.0.2.**

**The rule, which is not "shortest".** `Double.toString` never uses fewer than TWO
significant digits, and where the shortest round-tripping form has ONE, the second
digit is **not a zero** but the closest two digit decimal to the ACTUAL value. For
every NORMAL double those are the same thing - the value is within ~1e-16 of its one
digit form, a thousandth of a two digit step - so it shows only among the
SUBNORMALS, where the gap between neighbours is comparable to the value itself:
`Double.MIN_VALUE` is 4.9406564584124654E-324, so java writes `4.9E-324` where
appending `".0"` writes `5.0E-324`. `2 * MIN_VALUE` is 9.88e-324, so java writes
`9.9E-324` where the shortest form is `1e-323`.

**The trap, and it is the reason a naive "always two digits" is wrong**: the second
digit belongs to the SIGNIFICAND, and the PLAIN form prints the NUMBER. `0.001` is
`"0.001"`, not `"0.0010"`; `100.0` still gets its forced `".0"` from the
already-existing rule. Only the scientific form ever shows the forced digit.

**The second trap**: the scaling exponent is the VALUE's, not the shortest form's.
`1e-322` is really 9.88e-323, whose two digit form is `9.9E-323` and NOT `1.0E-322`,
so a scaled value below ten steps down one decade first; a carry to 100 steps back
up. Four halves, not three:

```
languages/lib/runtime.c            g_mindig = 2 around shortest_digits, which
                                   starts the search at k = 2 - the candidate for
                                   k digits IS the exact value rounded to k
                                   digits, nearest with ties to even, so no new
                                   arithmetic was needed. The plain branch strips
                                   the padding zero again.
abnf/jsrtjvm.go   jvmFloStr        FormatFloat(a, 'E', 1, 64) when the shortest
                                   mantissa has no '.', which IS "the closest two
                                   digit decimal", carry included
languages/metajs-interpreter.abnf  flTwo + flScale: the host engine's "" + a is
                                   the shortest form, so the second digit is
                                   computed by scaling and rounding. 10^325 is an
                                   infinity, so the scaling is done in chunks.
languages/java-interpreter.abnf    jFloSci, the same two steps
```

**Ground truth: 613 values** - every subnormal bit pattern from 1 to 59 plus the
2^52 edges, a 6-mantissa x 92-decade grid from 1e-330 to 1e308, and both plain
window boundaries in both directions - **byte-identical to `java` 24.0.2 in all five
MetaJS engines** (`floStr(flo(v, 0))` under the interpreter goja and `-frozen`,
`llvm.Run` goja and `-frozen`, and the native binary). The java-language halves are
checked on the 379 of those whose plain decimal literal the java grammar accepts
(its literal rule is bounded, which is a separate and pre-existing limit): the java
interpreter under both engines, `llvm.Run` and the native binary are **379 of 379
identical to real java**.

### What is owed, and what generalises

1. ~~**The floor request above** (`sintRaw`, a tag 13 without `si_norm`), and the
   CONSUMER side after it.~~ **BOTH DONE** - the floor 2026-08-03, the collapse
   2026-08-04. See "THE COLLAPSE ONTO `sintRaw`" below.
2. `abnf/jsrtjava.go` and `abnf/jsrtjvm.go` stay until the change is committed.
3. ~~The subnormal formatter~~ **DONE 2026-08-03**, all four halves, 613 values
   against real java.
4. **`Math.abs` of a `long` answers an `int`**, found by the collapse's own probe
   and NOT introduced by it - `Math.abs(3000000000L)` is -1294967296 where real
   java 24 says 3000000000, and `Math.abs(Long.MAX_VALUE)` is 0. **20 of the
   26 lines** on which our java differs from real java over the whole long
   surface are this one cause. It is one expression in each of TWO halves -
   `jvmMathObject`'s `abs` in `abnf/jsrtjvm.go` (`rt.toNumber` then a 32 bit wrap)
   and `jvBoxType("Math")`'s `abs` in `languages/lib/java-rt.metajs`
   (`rtWrap32(Math.abs(jvNum(v)))`) - plus a ratchet assertion, and the three must
   land together or the two engines stop agreeing. **`abnf/jsrtjvm.go` is another
   agent's file this session, so this is reported rather than made.** The other
   6 of the 26 are `(float)`, which is a no-op in both halves: our java has no
   32 bit float type at all, which is a value-model decision and not a defect.

### THE COLLAPSE ONTO `sintRaw` - DONE 2026-08-04, and what it actually traded

The unsigned-box workaround above is **gone**. A Java long is now an honest
`{w: 64, u: 0}` cell built with `sintRaw`, exactly as `csharp-rt.metajs` has
carried one from the start, and `jlDiv` / `jlMod` / `jlSar` / `jlCmp` / `jlStr` /
`jlNum` / `jlIsNeg` are deleted: `sintOp` code 3, 4 and 10 branch on the cell's
own signedness, `sintCmp` compares signed when the type is, `sintStr` prints the
minus sign and `sintNum` (`si_float`) reads the negative half with `d_from_long`.

**It is a REPRESENTATION SWAP, not a deletion, and the two things it bought have
to be paid for** - the merge agent's reason for declining was right about both:

1. **`>>>` is no longer free.** An unsigned cell shifts logically, so `>>>` was
   `sintOp(10, ...)` direct; a signed cell shifts arithmetically. `jlSar` is
   traded for **`jlShr`**, which re-marks the same bits unsigned for the one
   call and signs the result back. One helper for one helper, four lines.
2. **`si_apply` ends in `si_norm`**, which UNBOXES a signed 64 bit result a
   double holds exactly - so `2L + 3L` comes back as the plain number 5, i.e.
   as an `int`. Every long-producing `sintOp` therefore goes through **`jlOp`**,
   which re-boxes with `sintRaw(sintHi(t), sintLo(t), 64, false)` (`sintHi` /
   `sintLo` read a plain number's two's-complement halves as readily as a
   cell's). `csharp-rt.metajs` spells the same rule `csNorm`. **This is the
   invariant to get right**, and it is silent when missed: a long that leaks out
   as a plain number is an int to everything downstream.

After the swap `sintOp` appears in `java-rt.metajs` at exactly TWO sites, both
inside `jlOp` and `jlShr`.

**The line delta, stated honestly.** The plan quoted ~90 lines saved. Measured:
`java-rt.metajs` **1,166 -> 1,097** lines (**-69**), and **754 -> 676** lines of
code with comments and blanks stripped (**-78**). Six helpers and their rationale
went; `jlOp` and `jlShr` and their rationale came, plus `jlOp(...)` in place of
`sintOp(...)` at twelve call sites and `sintRaw` at five constructors.

**What it also removed, which is worth more than the lines**: the second-order
hazard class audited below is gone from layer 2 - a signed cell is read correctly
by every floor extern that calls `to_number`. It does NOT remove the emitter's
`js_jvnum` unwrap, and cannot: `llvm.Run` runs `abnf/jsrtjava.go`, whose `jvBox`
is unchanged, so `java-to-llvm-ir.abnf` still has to emit it. **The saving is in
layer 2 only for as long as the Go twin exists.**

**Ground truth.** `tests/java-test-full.java` is **278 checks, 0 failures**
natively, and a **16,477-line differential probe** - `+ - * / % & | ^` and all six
comparisons over 32x32 long pairs (0, +-1, +-2, +-3, +-7, +-10, both int32 edges
and their neighbours, 2^32 and its negation, +-2^53 and +-(2^53+1), +-1e12, 3e9,
both int64 edges and their neighbours, 0x5555...), the three shifts over 14 counts
spanning 0/31/32/33/62/63/64/65/100 and the negative ones, unary `-` and `~`,
rendering, concatenation, the `(double)` / `(float)` / `(int)` / `(short)` /
`(byte)` / `(char)` casts, `Math.abs` / `Math.max` / `Math.min`, `Long.parseLong`,
`++` / `--` / `+=` / `-=` / `*=` / `>>=` / `>>>=`, the mixed int/long and
long/double grids, and `Long.MAX_VALUE` / `MIN_VALUE` / `SIZE` / `BYTES` - is

```
the native binary vs llvm.Run                    BYTE-IDENTICAL, 16,477 lines
the native binary vs a clean archive of 44bb09a  BYTE-IDENTICAL, 16,477 lines
                                                 (the SAME binary output, so the
                                                 swap is observably neutral)
real java 24.0.2                                 26 lines differ, ALL pre-existing
                                                 (identical at 44bb09a): 20 are
                                                 Math.abs(long), 6 are (float)
```

**Discriminating power**, measured the way the plan asks - one behaviour mutated
at a time, against the existing 278 assertions, in the native binary (the only
engine that runs layer 2):

```
jlOp does NOT re-box (si_norm's plain number kept)          7 fail
>>> uses the signed (arithmetic) shift                      3 fail
the cell marked UNSIGNED again, with no jl* layer           4 fail
```

So no new ratchet assertions were owed: `int14` (`1000000L * 1000000L`), `int15`,
`int21`/`int22`/`int48` (long `>>>` on negatives), `int24`/`int26`
(`Long.MIN_VALUE / -1`) and `int7` already pin every arm of the swap. **A refactor
that adds no assertion has to prove that by mutation, and that is what the three
rows above are.**

**The floor's pending shift-result-type fix does not change this calculus.**
`giArith` / `si_apply` codes 9 and 10 / `szArithSlow` should take `w`/`u` from
the LEFT operand only; Java already passes a PLAIN NUMBER count to every shift
(`jvShift` masks it with `& 63` first), so `si_width_of(l, s)` picks the left cell
today and will pick it after. Checked, not assumed - `si_apply` already carries a
`die("PROBE-C >>")` asserting exactly that.

**The generalisation, now that both representations have been built and measured**:
`sintRaw` wins, and the reason is not the line count. `>>>` costs one helper
either way (`jlSar` or `jlShr`), and `jlOp` is the same three lines `csNorm` is.
What the unsigned box costs that the signed one does not is the SECOND-ORDER
hazard - every floor extern reading `to_number` on the cell sees a magnitude -
and that is a whole-emitter audit per language rather than a helper.

### The second-order hazard, audited across the whole floor

Java found ONE floor extern that misreads a long carried as an unsigned-marked box
(`js_jflo`) and fixed it in its emitter with `js_jvnum`. The mechanism generalises,
and the audit below is the complete list for the five languages that follow, because
**it is silently wrong rather than loud**.

**The mechanism, in one line**: `to_number` of a tag 13 cell IS `si_float`
(`runtime.c`, `to_number`'s `if (t == 13)` arm), and `si_float`'s
`if (fc(h) && fb(h) == 64) return d_from_ulong(fa(h))` is the magnitude. So **every
extern that calls `to_number` on a possibly-tag-13 operand is exactly as hazardous as
`js_jflo`**. `si_val` (`return fa(h)`) is the signed-correct reader and is SAFE.

**Reachable the moment csharp / go / kotlin / swift / dart copy Java's trick**, in
priority order:

| # | externs | why |
|---|---|---|
| 1 | **`js_gflo`, `js_csflo`** | literally `js_jflo`'s line with a different style byte. go emits `js_gflo`, csharp `js_csflo`, and **kotlin calls `js_jflo` directly with no `js_jvnum` equivalent** - it re-inherits the already-diagnosed bug |
| 2 | **`js_jfsub` / `js_jfmul` / `js_jfdiv` / `js_jfmod`** | all reach `jf_arith`, whose first two lines are `to_number(l)` / `to_number(r)`. These are the MIXED int/float operators, i.e. exactly `longValue * 2.0` |
| 3 | **`js_jfint`, `js_jfneg`** | `to_int32(to_number(v))`: a `-1L` box answers 0, not -1 |
| 4 | **`js_sub` / `js_mul` / `js_le` / `js_ge`** in kotlin's and swift's RANGE loops, on user-supplied bounds, via `js_compare`'s `to_number` |
| 5 | **swift's `binExt` table** - `& \| ^ << >>` map straight to the floor's `js_band`/`js_bor`/`js_bxor`/`js_shl`/`js_shr` on USER operands, and Swift's `Int` is 64 bit |
| 6 | **`js_gival`** | go uses it as the unboxer for slice/index expressions, and it unboxes via `si_float`. `si_val` is the right one for a Java-style box |
| 7 | **`js_gicmp` / `js_gilt` / `js_gile` / `js_gigt` / `js_gige`** | not the magnitude but the same root cause: `si_cmp` takes the UNSIGNED order, so a `-1L` box sorts ABOVE `0`. This is what `java-rt.metajs`'s `jlCmp` re-supplies |

**Theoretically reachable only, and each says why**: `js_get`/`js_set` with a tag 13
key (the magnitude fails `d_in_long` and returns -1, which is what a negative index
does too - no observable divergence); `js_add`'s string arm and `js_throw`'s
`to_string` (they render `si_str`'s unsigned digits, which IS the documented
rendering for a genuinely unsigned box, and layer 2 intercepts them first);
`js_eq`/`js_ne` (their `to_number` calls fire only when one side is a BOOLEAN - box
vs box and box vs plain number go to `strict_eq`'s safe `si_eq`).

**SAFE by construction, checked**: `js_truthy` and `js_not` (`fa(h) != 0`, the raw
payload), `js_typeof`, `js_seq`/`js_sne` (`si_eq` -> `si_val`), `js_giconv`,
`js_gineg`, `js_ginot`, `js_giis`, `js_gilit`, `js_tdecl`/`js_tset`, and every
handle-passing extern.

**Not externs but the same class**, and the layer-2 mirror of row 2 above: host ids
`sintNum` (50, `si_float`), `sintStr` (49, `si_str`), and the whole `flo*` family -
`flo` (51), `floNum` (53), `floOp` (56), `floMax`/`floMin` (58/59), `floAbs` (60) -
all read through `to_number`. `java-rt.metajs`'s `jlNum` already compensates for id
50; **nothing compensates for the `flo*` family**, so a language that hands a
Java-style long to `floOp` gets the magnitude.

**The general rule for the next five**: a sized integer handed to a FLOOR primitive
needs an emitter-side `js_*num` unwrap, and the check is a grep over the emitter for
the externs in the table above - not a judgement call. **`sintRaw` removes the need
for the trick entirely**, which is the cheaper answer: a raw SIGNED box is read
correctly by every one of them.

What **generalises to the remaining eight**, in order of how much it saves:

- The unsigned-box trick for a 64 bit signed type, and the emitter-side `js_*num`
  wrapper that keeps a FLOOR extern from reading it as a magnitude. Any language
  whose emitter hands a sized integer to a floor primitive needs the second half.
- A scope probe in the emitter (three languages now, three shapes, one helper).
- `_raw` parameters: check each emitter; the count was 3 / 0 / 1 for lua / php / java.
- `core.truthyExt`: java needed no `__raw` because it never overrides the default.
  **Check before writing one**; the grep is `core.truthyExt` in the grammar.
- An object box for a value type the floor has not got is safe exactly when no floor
  extern ever receives it. That is a grep over the emitter, not a judgement call.

### launch.json entries wanted (not added - not my file)

```
"Java native binary, full syntax (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/java-to-llvm-ir.abnf", "tests/java-test-full.java", "-q", "-exe", "tests/java-native-full.out"]
"Java native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/java-to-llvm-ir.abnf", "tests/java-test-features.java", "-q", "-exe", "tests/java-native.out"]
"Java native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/java-to-llvm-ir.abnf", "tests/java-test-multifile.java", "-q", "-exe", "tests/java-native-multi.out"]
```

All three pass natively today, so unlike PHP the full-syntax entry is requested too.

## CSHARP - DONE (2026-08-03). The gate is MET, generators included

**`tests/csharp-test-full.cs` runs as a self-contained native binary and agrees
with `llvm.Run` byte for byte - stdout, stderr and exit code - on all 250
assertions (247 at the time of writing, plus the three `itr7`/`itr8`/`itr9` the
floor fix below made it possible to pin), INCLUDING the SECTION 19 iterator
ones**, which is the point of doing
C# now. `tests/clang-check.sh` reads

```
csharp          36854  ok, and the clang executable agrees
```

so csharp left the `ok (module only)` row for the run-it row and NO row is held.
`abnf/jsrtcsharp.go` is **kept**, as the plan requires: it is what `llvm.Run` still
uses, and the two are now differentially compared over a 19,387-line probe.

**There is no C# toolchain on this machine** - `dotnet`, `csc`, `mono` and `mcs`
are all absent from `PATH`, and there is no `/usr/local/share/dotnet`, no
`/opt/homebrew/bin/dotnet` and no `/Library/Frameworks/Mono.framework`. So unlike
Java, whose oracle was a real `javac`/`java` 24.0.2, **every C# rule here is CITED
rather than executed**, exactly as `abnf/jsrtcsharp.go`'s own header already says of
itself. The ground truth available was: the two halves against each other over the
whole probe, plus the 250 assertions of a ratchet file that was written against
ECMA-334 clause numbers. That is weaker than Java's and it is stated rather than
implied.

### The extern split

`tests/csharp-test-full.cs` drives **68** externs, and so does the union over every
`tests/csharp-test-*.cs`.

| where | count | what |
|---|---|---|
| already in the C floor | 35 | the generic primitives of phase 3, the five the Lua pilot added, `js_truthy`, `js_csflo`, and **`js_genfn` / `js_yield` / `js_ctl_*`** |
| **layer 2**, language-NEUTRAL | 15 | `js_arr_new_n`, `js_char`, `js_char_code`, `js_dict_new`, `js_goslice`, `js_gospread`, `js_has`, `js_is_type`, `js_jadd`, `js_jchareq`, `js_mcall`, `js_pyget`, `js_pyset`, `js_pylen`, `js_supercall` |
| **layer 2**, C#'s own | 18 | the `js_cs*` family |

`languages/lib/csharp-rt.ll` (19,607 lines, from a 1,430-line `.metajs`) exports
exactly those 33 symbols and nothing else - checked with `comm`, not assumed. The
layer-2 source compiles **byte-identically under `-frozen`** as well as under goja.

The split against the four done before it:

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
java           29        9      16      ~90 lines of signed-long workarounds
csharp         35       15      18      NONE  <- sintRaw
```

C#'s neutral 15 is the largest of the five, and that is a fact about the FLOOR
rather than about C#: `js_goslice`, `js_gospread` and `js_has` are Go's and
JavaScript's, they are in `abnf/jsrt.go`, and three languages have now written them
out independently. They are the next floor candidates on this line.

### `sintRaw` paid for itself the first time it was used, and it removed a HAZARD

The floor request Java filed - "a constructor that builds a tag 13 cell WITHOUT
`si_norm`" - landed as `sintRaw(hi, lo, bits, unsigned)`, host id 62. C# is the
first user and the result is better than the ~90 lines the request predicted:

1. **The 90 lines are gone.** `csharp-rt.metajs` has NO `jl*` layer. A C# `long` is
   an honest `{w: 64, u: 0}` cell, so `sintOp`'s `/` `%` `>>`, `sintCmp`, `sintStr`
   and `sintNum` are all used DIRECT - each of the six operations Java had to
   re-supply is the floor's own, already branching on the cell's signedness.
2. **The second-order hazard went with them, and that is the bigger half.** Java's
   report warned that any FLOOR extern reading a tag 13 cell numerically sees a
   MAGNITUDE when the cell is the marked-unsigned shape, and had to fix `js_jflo`
   in its emitter with a `js_jvnum` wrapper. C# has exactly one such extern -
   `js_csflo`, the `(double)` cast and the `double`-declared initializer - and it
   needed **no wrapper at all**, because `to_number` of a signed cell is the signed
   reading. `go`, `kotlin` and `dart` should take this route and not Java's.

The one rule the layer-2 code has to keep is that `sintConv` (and every other
`sint*` builtin) still ends in `si_norm`, so a 64-bit SIGNED result can come back
unboxed. `csNorm(v, w, u)` is the single place that is repaired - `sintHi`/`sintLo`
read a plain number exactly, so the re-box through `sintRaw` is lossless - and
every other function in the file goes through it.

> **`sintRaw` is complete in all THREE engines**, which was checked at the end
> rather than assumed: the floor seeds it (`seed_root("sintRaw", mk_host(62))`),
> `abnf/jsrtint.go` binds it in `giBindings`, and `languages/metajs-interpreter.abnf`
> has `siRawHost`. (Mid-task it was floor-only, which would have meant
> `csharp-rt.metajs` could not be run under `llvm.Run` for debugging; the floor
> agent landed the two twins in the same session.)

### DRIVING THE FLOOR'S GENERATORS - what js / ts / python need to know

**C# is the first language on the landed coroutine primitive, and layer 2 owes it
ZERO LINES.** The cost table in the coroutine section estimated ~40 lines of MetaJS
for csharp's `MoveNext`/`Current`/`yield break`. The measured answer is 0, and the
reason generalises:

> **`csharp-to-llvm-ir.abnf` already lowered the whole iterator protocol in the
> EMITTER, over externs the floor already has.** Nothing had to move to layer 2
> because nothing was ever in layer 2.

The five lines that do it, all pre-existing (`csharp-to-llvm-ir.abnf:2272-2392`):

```
js_genfn(js_closure(idx, scope))                     the iterator method itself
js_seq(js_typeof(js_get(v, "next")), "function")     "is this a generator?"
js_call(js_get(g, "next"), undefined, [])            MoveNext()
js_get(step, "value")  /  js_get(step, "done")       the step record
an ordinary ret                                      yield break
```

`get_member` answers `next` on a tag 15 cell with `mk_bound(g, 60)`, and `.Current`
is a field the emitter parks on its own `{__csenum, g, cur}` cursor, so **every one
of those is a floor extern**. A `throw` out of a generator body needed no case
either: `gen_resume` re-raises on the resumer's stack, so it arrives at the `next()`
call site like any other exception - measured, probe line `it4`, which throws from
inside a body after one `yield` and catches it around the `foreach`.

**The one thing that DID bite, and js/ts/python will hit it too:**

> `js_genfn`'s result is a `*hostFunc` in the Go twin, so `typeof` answers
> **"function"**. In the C floor it is a tag 16 cell: `is_callable(16)` is TRUE but
> `type_of` falls through to **"object"** (`runtime.c:2061`, `2819`). A static
> iterator method is therefore a class member that `js_mcall` refuses to call
> natively while calling it happily under `llvm.Run`. Measured: it was the FIRST
> native failure of `tests/csharp-test-full.cs`, `js runtime error: unknown method
> 'S19UpTo'`, with the other 246 assertions never reached.
>
> ~~**FLOOR REQUEST (reported, not made - the floor is another agent's this
> session): `type_of` and `type_class` should answer "function" for tag 16**~~
> **LANDED (2026-08-03).** `type_of`, `type_class` **and `to_string`** now carry
> `t == 16` next to 7/8/9, matching `is_callable` one line below them and matching
> `abnf/jsrt.go` (`*jsClosure, *hostFunc, *boundMethod` -> `"function"` at 1545 and
> `"[function]"` at 1065; `js_genfn` returns a `*hostFunc`). Three lines, not two.
>
> ~~Until it lands, `csharp-rt.metajs`'s `csIsFn` asks the question the way
> `is_callable` answers it~~ - **the workaround is GONE**: `csIsFn` is now
> `return typeof v == "function"`, and the `typeof v["apply"] == "function"` probe,
> with the four guard lines it needed, is deleted.

**The latent EMITTER divergence: MEASURED, it WAS broken, and it is now pinned.**

The site was mis-attributed above: `csharp-to-llvm-ir.abnf:2521` is inside
**`emitCompound`**, not `emitDelegCall`. `emitDelegCall` discriminates on
`js_is_type(fV, "List")` and never asks `typeof`, so it was never affected.
`emitCompound` is: `f += g` / `f -= g` on a delegate decides combine-vs-arithmetic
at run time by testing `js_typeof(v) === "function"` on the RIGHT operand, so the
divergence needs an **iterator method as the right operand of a delegate compound**.

Constructed and run in all three engines, against a clean `git archive` of
`65a443b` built in its own directory:

```csharp
delegate IEnumerable<int> Seq(int n);
static IEnumerable<int> UpTo(int n)  { for (int i = 1; i <= n; i++) { yield return i; } }
static IEnumerable<int> Tens(int n)  { yield return n * 10; yield return n * 100; }
Seq a = Program.UpTo;   a += Program.Tens;   // <- the arm choice
```

| | interpreter | `llvm.Run` | native |
|---|---|---|---|
| at `65a443b` | `call 6 / combined 330 / removed 6` | same | **`call 6`, then `js runtime error: call of a non function value: 0`, rc=1** |
| with the floor fix | same | same | **same, rc=0** |

So it was BROKEN, not merely unmeasured: the native half took the arithmetic arm,
`js_csarith` handed back a non-callable, and the next invocation died.

**Pinned** as `itr7`/`itr8`/`itr9` in `tests/csharp-test-full.cs` section 19
(`S19Seq`, `S19Tens`, `S19Sum`; the ratchet goes 247 -> 250). Discriminating
power, measured twice:

* the same file dropped into the clean `65a443b` archive - interpreter 250/250,
  `llvm.Run` 250/250, and `tests/clang-check.sh csharp` flips from
  `ok, and the clang executable agrees` to **`ok (module), BUT THE NATIVE RUN
  DISAGREES`**; the native binary produces NO output at all and exits 1, so all
  250 assertions are lost, not just the three.
* mutating the fix back out of `runtime.c` in the CURRENT tree (with the
  simplified `csIsFn`) and regenerating `runtime.ll` - same red row. Restored and
  `gen-runtime-ll.sh --check` re-verified.

**`tests/metajs-test-full.js` gets an honest ZERO.** MetaJS has no generator
syntax and `metajs-to-llvm-ir.abnf` never emits `js_genfn` (`grep -n
'genfn\|yield' languages/metajs-*.abnf` is empty of both), so MetaJS cannot
construct a tag 16 cell and there is nothing there to assert. The pin is C#'s
alone until js/ts/python arrive.

### The two cross-cutting findings, checked and MEASURED for csharp

1. **`core.truthyExt`.** `grep -n truthyExt languages/csharp-to-llvm-ir.abnf` is
   EMPTY, so csharp keeps compile-core's default `js_truthy`, which is the C floor's
   own function and already answers the raw 1/0 the emitted `icmp ne ..., 0`
   compares. **No `__raw` export.** The running count is lua YES, php YES, swift NO,
   java NO, csharp NO.
2. **`_raw` PARAMETERS: csharp has exactly one**, `js_char`, whose code point the
   emitter writes with `handle(code)` at two sites. Same as java. Count so far:
   lua 3, php 0, swift 0, java 1, csharp 1.
3. **The scope probe.** csharp needed NO lowering: its emitter already probes with
   `js_scope_typeof` against `"undefined"` in IR (`csharp-to-llvm-ir.abnf:4109`) and
   never had a scope-taking extern. Four languages, three lowerings, one already
   done - the pattern holds and it is cheap or free.
4. **`jsChar`.** The floor still has no char tag and C# still does not need one, for
   Java's reason: the `{__char}` object box is safe exactly because no FLOOR extern
   ever receives it. Audited: the emitter's `js_csflo` sites are guarded by
   `js_char_code` or fed a literal; `js_seq` is emitted only on lengths, `typeof`
   results and class descriptors; `==`/`!=`/`switch` go through `js_jchareq`; the
   ordered comparisons through `js_cscmp`. The one unguarded site is the list-pattern
   `js_seq` at `csharp-to-llvm-ir.abnf:2470`, which would compare two chars by
   IDENTITY natively and by value in Go - no test declares a `char[]` list pattern
   with a rest, so it is unmeasured rather than wrong, and `js_jchareq` is the
   one-line fix.

### Ground truth

```
matrix                  325/325
--full                  csharp 247; 5615 assertions in total, goja and -frozen
                        byte-identical, 0 languages with halves that disagree
                        (5615 rather than the 5590 of 94ac7b8: dart's ratchet grew
                        in the same session, in another agent's files)
                        grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF'  -> empty
--cross                 119 programs, 0 divergent
go test ./abnf/         ok
clang-check             16/16, and csharp joins the RUN-IT row
gen-csharp-rt-ll.sh --check   csharp-rt.ll is up to date (19607 lines)
-frozen recompile       csharp-rt.ll byte-identical to the goja build
jsbootstrap.ll          untouched (metajs-to-llvm-ir.abnf was not edited)
a real C# toolchain     NONE ON THIS MACHINE - see the note at the top
```

RE-MEASURED after the tag-16 floor fix, the `csIsFn` removal and the three new
iterator-delegate assertions (working tree on `b7fdc4d`, shared with the go / ruby
migrations):

```
matrix                  325/325
--full                  csharp 250; 5618 assertions in total, 0 languages whose
                        halves disagree; the grep is empty
--cross                 119 programs, 0 divergent, 0 differing only in warnings
go test ./abnf/         ok
clang-check             16 modules, csharp `ok, and the clang executable agrees`
                        (36854 lines). Two findings, BOTH other agents' in-flight
                        work and neither caused by this change: go and ruby say
                        `ok (module), BUT -exe FAILED TO BUILD` - both -exe
                        branches are being written this session and neither
                        existed at 65a443b.
gen-*-rt-ll.sh --check  runtime (47963), csharp (19490), bash, batch, dart, java,
                        lua, php, swift all up to date. gen-ruby-rt-ll.sh fails
                        ("only 0 lines"); it is untracked and another agent's.
-freeze fixed point     `mec -freeze languages/metajs-to-llvm-ir.abnf` leaves
                        abnf/jsagrammar.go and abnf/jsbootstrap.ll unchanged
coro-poc build.sh       BYTE-IDENTICAL in both engines (36 lines); --gc all four
                        collector modes SAME; --break unchanged, the same eight
                        broken roots DIFFER and the same six do not
bench-alloc.sh          RSS FLAT under the collector (3,309,568 B at 100k, 200k
                        and 400k iterations, i.e. bounded). MEC_GC=off is
                        3,734 / 3,721 / 3,715 / 3,711 bytes per iteration at
                        50k / 100k / 200k / 400k - the gate is ~3,713 and the
                        number is unchanged by this edit
```

`csharp-rt.ll` SHRANK, 19,607 -> 19,490 lines: the `csIsFn` workaround's four
guard lines and the `apply` lookup are gone and nothing replaced them.

Every C# program in `tests/`, `llvm.Run` against the native binary, stdout **and**
stderr **and** exit code:

```
csharp-test-full  csharp-test-1  csharp-test-features  csharp-test-complete
csharp-test-try   csharp-test-recognize  csharp-test-main (rc=1 in both)
csharp-test-multifile (-i tests/imports)                    all SAME
```

### The differential probe, and the one defect it found

A **19,387-line / 523 KB** probe - `+ - * / % & | ^` over 26 int values, 20 long
values, 10 ulong values and 9 uint values pairwise, the mixed int/long, int/uint and
int/ulong promotions, the six comparisons across int/long/uint, `<<` and `>>` over
13 counts spanning both the `&31` and `&63` mask boundaries and a negative one,
unary `-` and `~` over all 75 integer values, every cast from 52 sources to all 11
target types, `+ - * / %` over 20 doubles (both zeros, both infinities, NaN,
DBL_MAX, the smallest subnormal, 1e20, 1e-20) pairwise and mixed with ints, the six
comparisons over doubles, `ToString` of every double, `MinValue`/`MaxValue` for all
nine integral types, `int.Parse`/`long.Parse` over 8 inputs, `++`/`--` at the wrap
point of all ten types, four compound assignments at the wrap point of nine, char
arithmetic and rendering over 6 code points including U+0000 and U+FFFF, string
`Length` and indexing over an astral-free non-ASCII string, array `ToString`, five
iterator drives (finite, ENDLESS-and-broken-early, explicit
`GetEnumerator`/`MoveNext`/`Current`, a THROW out of a body caught around the
`foreach`, and nested generators) and both integer divide-by-zero throws - is
**BYTE-IDENTICAL between `llvm.Run` and the native binary**.

One defect, found only by the probe and invisible to `./test.sh` because both
engines were compared with themselves:

> **`jsCompare`'s NaN SENTINEL is 2, and `js_cscmp` compares it like an ordering.**
> `abnf/jsrt.go:1510` answers `2` for a NaN operand, commented "every relation is
> false" - but `jsrtcsharp.go`'s `js_cscmp` does `c > 0` for `>` and `c >= 0` for
> `>=`, so under `llvm.Run` **`double.NaN > 0.0` is TRUE** and `NaN <= 0.0` is
> false. Layer 2 first returned the obvious 0, which makes `>` false and `<=` TRUE -
> wrong in a DIFFERENT way. 16 probe lines. The fix was to reproduce the sentinel
> exactly, and the site says so at length.
>
> **Both halves now agree and BOTH ARE WRONG against real C#**, where ECMA-334
> 12.12.1 makes all four of `< > <= >=` false for a NaN operand. This is the defect
> class the rules name - "both halves agreeing and both being wrong" - and byte
> identity cannot see it. It is NOT fixable from layer 2 alone: `js_cscmp` in
> `abnf/jsrtcsharp.go` and this file both have to test for the sentinel in the same
> commit, and `js_swlt`/`js_jvcmp` should be checked for the same shape at the same
> time (`java-rt.metajs`'s `jpCompare` returns 0 where `jsCompare` returns 2, so
> Java may have the mirror-image mismatch on the `>=` arm).
>
> **HALF FIXED 2026-08-04.** `js_cscmp` in `abnf/jsrtcsharp.go` now returns false
> for all four operators when `jsCompare` answers the sentinel, with ECMA-334
> 12.12.1 quoted at the site (no C# toolchain exists on this machine, so the SPEC is
> the oracle and is named as such). Measured on a six-line probe: `llvm.Run` went
> from `True False True False` to `False False False False` for
> `NaN > 0.0 / < / >= / <=`, which is what real C# says; `==` and `!=` were already
> right. **LAYER 2 STILL OWES THE OTHER HALF**, and until it lands the two engines
> DISAGREE on this one construct - the same probe run natively still prints
> `True False True True`. The owed change is `js_cscmp` in
> `languages/lib/csharp-rt.metajs` (about line 1216): after `var c = csCompare(l, r)`
> add `if (c == 2) { return false }`, and replace the long comment above `csCompare`
> that currently explains why the sentinel could NOT be fixed. **No ratchet
> assertion was added, deliberately**: one would be green under `llvm.Run` and red
> natively, which would take `tests/clang-check.sh` from 16/16 to 15/16. Add this to
> `tests/csharp-test-full.cs` in the same commit as the layer-2 line:
>
> ```csharp
> double nanZ = 0.0;
> double nanV = nanZ / nanZ;
> Program.Check("nan1", !(nanV > nanZ) && !(nanV < nanZ) && !(nanV >= nanZ) && !(nanV <= nanZ));
> Program.Check("nan2", !(nanV == nanZ) && nanV != nanZ && !(nanV >= nanV));
> ```

### What is owed, and what generalises

1. **The floor line above**: `type_of`/`type_class` should answer `"function"` for
   tag 16 (blocking nothing today only because of a layer-2 workaround, and latent
   in `emitDelegCall`). `sintRaw`'s `giBindings` and interpreter twins landed in the
   same session and are NOT owed.
2. **The NaN sentinel**, which needs the Go twin and layer 2 changed together.
3. `js_goslice` / `js_gospread` / `js_has` in the floor - three languages have now
   written them out.
4. `abnf/jsrtcsharp.go` stays until the change is committed.
5. ~~`>>>` is **not in the recognised C# subset**~~ **RECOGNISED 2026-08-04, not
   deleted.** `Shift` accepted only `<<` and `>>` while `emitBin`, `js_csshift`,
   `csShift` in `lib/csharp-rt.metajs` and `csShift` in `csharp-interpreter.abnf`
   all implemented `>>>` - four halves of dead code reachable from nothing. Deleting
   the arm would have cost real C# 11 surface AND touched two files this task does
   not own, so both grammars now accept `>>>` and `>>>=` instead, LONGEST FIRST so
   `>>` cannot swallow the first two characters. The type args of a generic are
   parsed by `SkipAngle` over single `<`/`>` characters, so
   `Dictionary<string, List<int>>` is unaffected - checked. No C# toolchain here:
   the oracle is ECMA-334's shift-operator clause (the unsigned right shift, zeroes
   shifted into the high-order bits, at the left operand's promoted width) and the
   answers are `-8 >>> 1 == 2147483644`, `-8L >>> 1 == 9223372036854775804`,
   `-16 >>>= 2 == 1073741820`. Pinned as `tests/csharp-test-full.cs` checks
   urs1-urs4; from a clean archive of ad922a0 the file does not PARSE in either
   half ("Last good parse position: tests/csharp-test-full.cs:926:40").

What **generalises to the remaining five**, in order of how much it saves:

- **`sintRaw`, not Java's marked-unsigned trick.** It removes ~90 lines AND the
  second-order hazard that cost Java an emitter change. go / kotlin / dart all have a
  64 bit signed type and should all take this route.
- **Generators may cost layer 2 nothing.** Check whether the emitter already drives
  `next()` in IR before writing any MetaJS: csharp's did, and js / typescript /
  python emit the same `js_genfn` shape. What they DO owe is the `js_iterable`
  drain - `js-to-llvm-ir.abnf:3567,4108` and `js_pyiter` materialise a generator, so
  their for-loops are eager and would hang on an infinite source. That is the one
  place those three differ from csharp, and it is an EMITTER change (a `next()`
  loop), not a layer-2 one.
- **The tag 16 `typeof` divergence hits every one of them**, because every one of
  them can put an iterator method on an object.
- `core.truthyExt` and `_raw` parameters: check, do not assume; the counts are in
  the two-things section at the head of Part 3.
- An object box for a value type the floor has not got is safe exactly when no floor
  extern receives it - a grep over the emitter, and csharp's is written out above
  including the one site that is unmeasured.

### launch.json entries wanted (not added - not my file)

```
"C# native binary, full syntax (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/csharp-to-llvm-ir.abnf", "tests/csharp-test-full.cs", "-q", "-exe", "tests/csharp-native-full.out"]
"C# native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/csharp-to-llvm-ir.abnf", "tests/csharp-test-features.cs", "-q", "-exe", "tests/csharp-native.out"]
"C# native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/csharp-to-llvm-ir.abnf", "tests/csharp-test-multifile.cs", "-q", "-exe", "tests/csharp-native-multi.out"]
```

All three pass natively today.

## DART - DONE (2026-08-03). The gate is MET: 228/228 natively, byte-identical

**`tests/dart-test-full.dart` runs as a self-contained native binary, all 228
assertions pass in it, and its stdout, its stderr and its exit code are
byte-identical with `llvm.Run`.** `tests/clang-check.sh` reads

```
dart            42374  ok, and the clang executable agrees
```

so dart left the `ok (module only)` row for the run-it row, **no row is held, no
assertion is held back, and no floor change was needed or made.**
`abnf/jsrtdart.go` is **kept**, as the plan requires: `llvm.Run` still uses it,
the native binary uses layer 2, and the two are now differentially compared over
31,675 probe lines.

**No oracle: there is no `dart` on this machine** - `command -v dart`, `flutter`,
`/usr/local/bin/dart`, `/opt/homebrew/bin/dart` and `/opt/homebrew/opt/dart-sdk`
are all absent, and `tests/dart-test-full.dart` says so in its own header. The
ground truth here is therefore **`abnf/jsrtdart.go` against layer 2**, which is
the byte-identity gate, plus the corpus citations already carried in the Go twin
and in the ratchet file. `clang` is Apple clang 17.0.0. Every statement below
about "Dart" is a statement about our two halves agreeing, not about the VM.

### The extern split

`tests/dart-test-full.dart` drives **62** externs; over every `tests/dart-test-*.dart`
the union is **67**, the five extra being `js_ctl_break`/`js_ctl_continue` (floor)
and `js_darteq`/`js_dartle`/`js_dartne` (Dart's own).

| where | count | what |
|---|---|---|
| already in the C floor | 37 | the generic primitives of phase 3, the five the Lua pilot added, `js_truthy`, the `sint`/`flo` machinery, `js_keys`, and the three the lowered scope probe now calls (`js_scope_typeof`, `js_typeof`, `js_sne`) |
| **layer 2**, language-NEUTRAL | 13 | `js_dict_new`, `js_is_type`, `js_jadd`, `js_mcall`, `js_pyeq`, `js_pyget`, `js_pyin`, `js_pyiter`, `js_pylen`, `js_pyset`, `js_pyset_new`, `js_pyslice`, `js_pyspread` |
| **layer 2**, Dart's own | 17 | the `js_dart*` family |

`languages/lib/dart-rt.ll` (21,372 lines, from a 1,652-line `.metajs`) exports
exactly those 30 symbols and nothing else - checked, not assumed. The layer-2
source compiles **byte-identically under `-frozen`** as well as under goja.

The split against the four done before it:

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
java           29        9      16      ~90 lines, the unsigned-box long
dart           37       13      17      NONE
```

**The trend the last four recorded holds, and the NEUTRAL column is now the one
that grows.** Dart's own 17 is second-smallest, and it is the same shape Swift's
was: rendering (`js_dartprint`/`js_dartstr`), value equality (`js_darteq`/
`js_dartne`/`js_dartident`), arithmetic and the four comparisons, the two
literal constructors, the type test, the const canonicaliser, and one method
dispatcher. But its neutral column is the largest of the five, because Dart's
emitter routes the whole collection library through the `js_py*` family. Six of
those thirteen (`js_pyeq`, `js_pyin`, `js_pyiter`, `js_pyset_new`, `js_pyslice`,
`js_pyspread`) had never been written in layer 2 before and are now available to
python, ruby, go and kotlin unchanged.

### `si_norm` is what Dart WANTS, which is the mirror image of the Java wall

Java's `long` must not unbox, so `java-rt.metajs` carries a 64 bit box marked
UNSIGNED and reimplements six operators - ~90 lines, and the reason the plan
asks for a floor `sintRaw`. **Dart needs none of it, and the reason is worth
stating because three of the remaining six will be on one side or the other.**
`abnf/jsrtdart.go`'s own invariant is

```
plain number   ==  an int whose value is exact in a double (|v| <= 2^53)
jsGInt         ==  an int outside that range (w = 64, u = false, always)
```

which is `giNorm` verbatim - and `giNorm` is exactly what `sint()` applies. So
Dart's int is a plain `sintOp` / `sintCmp` / `sintStr` / `sintNum` at the default
width and there is no wrapper anywhere. **The question to ask a new language is
not "does it have a 64 bit signed type" but "is its unboxed representation the
one `si_norm` produces".** Dart's is; Java's is not.

`sintRaw` would still be worth having for java/go/kotlin - this is not an
argument against it, only a measurement that Dart is not one of the five it was
costed for.

### The one thing the tag could NOT supply, and it cost three lines

**`>>>`.** Dart's int is always signed, and the sized-integer tag's `>>` is
arithmetic at the box's own signedness, so there is no op code for a LOGICAL
64 bit shift. `dtLsr` is written over the codes there are - shift right one,
clear the sign bit, shift right the rest - and every step is exact because the
mask is built with the tag's own `<<` and `^`:

```
(x >> 1) & 0x7fff_ffff_ffff_ffff   then  >> (n - 1)
```

Not a floor request: three lines is cheaper than an eleventh op code, and the
same trick is what `dtRadixMag` uses to read a negative 64 bit value as unsigned
(`strconv.FormatUint`'s job) without a `sintRaw`.

### The two cross-cutting findings, checked and MEASURED for dart

1. **`core.truthyExt`.** `dart-to-llvm-ir.abnf` sets none, so it uses the default
   `js_truthy`, **which is in the floor**. Dart therefore needs **no `__raw`
   export**. The count over the five ported languages is now: Lua and PHP need
   it, Swift, Java and Dart do not. `grep -n truthyExt` in the grammar remains
   the one-line answer.
2. **`_raw` PARAMETERS: dart has ZERO.** Every selector its emitter passes -
   `js_dartarith`'s operator name, `js_dartlit`'s digits and radix - goes through
   `emitStr`/`emitNum`. Every bare `handle(k)` in the grammar goes to
   `js_closure` or `js_arg`, which are floor externs. The count is now
   3 / 0 / 0 / 1 / 0 for lua / php / swift / java / dart.
3. **The scope probe, third confirmation and the FIRST verbatim reuse.** Dart's
   `js_kget`/`js_kset` are the same `this`-aware Kotlin pair Swift has - the Go
   arms in `abnf/jsrt.go` are literally the same two functions - so
   `swift-to-llvm-ir.abnf`'s `sw_kget`/`sw_kset` lowering transferred to
   `dt_kget`/`dt_kset` unchanged, together with the `dt_safeget` guard and the
   `core.scopeGetExt = "js_scope_get"` override that keeps compile-core's
   `makeVarRef` off a name that is no longer an extern. 31 read sites and 3 write
   sites became one call each, `llvm.Run` stayed at 228/228, and no floor change
   was needed for the third time.

> **The lowering is not free in the extern COUNT, and the plan should say so.**
> It removes two externs and introduces three (`js_scope_typeof`, `js_typeof`,
> `js_sne`) that the emitter did not previously reach. All three are already in
> the floor, so the number that matters - what layer 2 has to write - fell by
> two; the declare list grew by one. Swift's 68 -> 66 was measured before those
> three were counted.

### `jsChar` was not needed, and neither was a coroutine

Dart has no character type: a "character" is a one-character String, and the
emitter emits `js_char` / `js_char_code` / `js_kindex` **zero** times. And
`grep -c 'js_genfn\|js_yield' languages/dart-to-llvm-ir.abnf` is **0** - Dart's
`sync*` / `async*` are lowered by the emitter itself, so WALL 1 does not touch
this language at all. Dart is the first of the eleven for which both of the
floor's two known type gaps are simply absent.

### Ground truth

```
matrix                325/325
--full                5,615 assertions, 0 languages whose halves disagree (dart 228)
                      grep 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF' -> empty
--cross               119 programs, 0 divergent
go test ./abnf/       ok
clang-check           16/16, and dart joins the RUN-IT row
jsbootstrap.ll        untouched (metajs-to-llvm-ir.abnf was not edited)
gen-dart-rt-ll.sh --check   dart-rt.ll is up to date (21,372 lines)
every other gen-*-rt-ll.sh --check   up to date
real dart             NONE ON THIS MACHINE - see above
```

`tests/gen-runtime-ll.sh --check` reports `runtime.ll DIFFERS` in this working
tree. That is **not this change**: `languages/lib/runtime.c` and `runtime.ll` are
another agent's concurrent edit and were already out of step before any Dart file
was touched. Nothing here depends on it - see the clean-archive run below.

**Verified from a CLEAN ARCHIVE, not the working tree** - `git archive 94ac7b8 |
tar x`, then only the four Dart files copied in, `go build` INSIDE, run there:

```
tests/gen-dart-rt-ll.sh --check     dart-rt.ll is up to date (21,372 lines)
./dt.exe                            full: 228 checks, 0 failures   (rc=0)
tests/clang-check.sh dart           ok, and the clang executable agrees
the three probes below               all BYTE-IDENTICAL, 31,675 of 31,675 lines
```

Every Dart program in `tests/`, `llvm.Run` against the native binary, stdout AND
stderr AND exit code:

```
dart-test-1  dart-test-complete  dart-test-features  dart-test-full
dart-test-multifile  dart-test-try                            all SAME (rc=0)
dart-test-recognize   NOT COMPARABLE - both halves refuse it identically
                      ("library directive not implemented", recognition-only file)
an uncaught throw     SAME message, rc=1 in both; llvm.Run's driver adds one
                      "  ==> Fail" line of its own, the same difference
                      php-test-undefconst and java record
```

### The differential probe - three probes, 31,675 lines, and the two defects

**Probe A, 30,625 lines**: every operand read out of a LIST at run time, so
nothing folds. 32x32 integer pairs times `+ - * & | ^ ~/ % remainder /` and all
six comparisons plus `identical` and `compareTo`; 32 integers x 13 shift counts
(spanning 31, 32, 63, 64, 65) times `<< >> >>>`; the full unary/member set per
integer (`- ~ abs sign isEven toDouble toInt truncate floor ceil round
*ToDouble toRadixString` at radix 2/8/16/36, `clamp`, the four `is` tests);
32x32 double pairs times `+ - * / % remainder ~/` and the comparisons; the double
unary and predicate set; and the 32x32 mixed int/double promotion grid. The
operand bags carry both zeros, both int64 edges, the 2^53 boundary and its
neighbours, 5e-324, 1e±100, DBL_MAX, 1e15/1e16 and 0.49999999999999994.

**Probe B, 131 lines** and **Probe C, 919 lines**: the rendering, keying and
dispatch surface - `Object.toString()` over every scalar and every container
kind nested three deep, the self-referential `[...]`/`{...}` cycle guards, a
user `toString()` on a superclass, enums (bare and enhanced), Types, records
(positional, named and nested), const canonicalization across two constructors
and across list/map/set/nested-const, `identical` over 26x26 pairs, Map and Set
keying over the same grid (where `1` and `1.0` must key alike), the string
methods, and the `is` matrix over 13 type names.

```
llvm.Run vs the native binary      BYTE-IDENTICAL on all three, 31,675 of 31,675
```

It found **two real defects**, both in layer 2, both invisible to `./test.sh`:

1. **`math.Mod` twice over.** `dtFmod` started as `a - Math.trunc(a / b) * b`,
   which is wrong in two independent ways. It loses the sign of a zero result -
   `-0.0 % 1.0` is `-0.0` in Go and came out `+0.0`, because `-0.0 - (-0.0)` is
   `+0.0` - which is **the fifth language in which the signed-zero trap has now
   bitten, and the first in which it arrived through a REMAINDER rather than a
   negation**. And it is not even approximately right across a wide exponent
   range: `1e100 / 5e-324` is not representable, where Go's `math.Mod` is exact.
   Both are fixed by the algorithm `math.Mod` itself uses, written without
   `Frexp`/`Ldexp`: scale the divisor up by powers of two while `d <= r / 2` (the
   guard that cannot overflow), then walk back down. Every `d * 2` and `d / 2` is
   exact, every `r = r - d` is exact by Sterbenz because the loop keeps
   `d <= r < 2*d`, and the sign is carried outside and put back with a unary
   minus. 111 probe lines.
2. **`[1, 2] + [3]` was "concatenated".** `abnf/jsrtdart.go`'s comment at
   `js_dartarith` says "'+' on two Lists concatenates them, so a non-numeric pair
   keeps the runtime's own rule", and layer 2 was written to that comment.
   `rt.jsAdd` does **not** do that: a list or an object on either side renders
   BOTH sides with `rt.toString` and concatenates the TEXT, so the answer is the
   string `"1,23"`. The Go twin is the specification and layer 2 now reproduces
   it. Recorded rather than corrected - making `+` concatenate lists is a change
   to `rt.jsAdd`, to `dart-interpreter.abnf` and to a ratchet assertion, and it
   is not this gate. It also forced `dtToStr` to become a faithful `rt.toString`
   (the comma join, `[object Object]`, and the boxed-double arm) rather than the
   three-line approximation it was.

### What is owed, and what generalises

1. Nothing for the gate. `abnf/jsrtdart.go` stays until the change is committed.
2. `[1, 2] + [3]`, above: a three-halves job, both halves currently agree.
3. ~~`js_pyget` on a MISSING Map key ABORTS (`KeyError`)~~ **FIXED 2026-08-04, all
   three halves.** dart:core is the oracle and is cited at both sites - no `dart`
   exists on this machine - and it says of `Map.operator []` that it returns "the
   value for the given key, or null if key is not in the map", which is what the
   whole `m[k] ?? d` / `if (m[k] != null)` idiom rests on. The fix is in the DART
   halves rather than in `js_pyget`, because `js_pyget` is shared with the Python
   pair where the `KeyError` is correct: `emitIndexGet` in `dart-to-llvm-ir.abnf`
   branches on `js_dartis(o, "Map")` (an O(1) type test) and, for a map, on the
   `js_pyin` over `js_pyiter` pair that `containsKey` was already lowered to, so
   **no half owes a new extern** and the native build needed no layer-2 change;
   `dget` in `dart-interpreter.abnf` returns `null` instead of failing. A list and a
   String keep `js_pyget`'s range error, which is Dart's too. Pinned as
   `tests/dart-test-full.dart` checks co7-co10; from a clean archive of ad922a0 BOTH
   halves abort on the file ("map has no key zz" / "KeyError: zz"), so the
   discriminating power is the whole ratchet.

What **generalises to the remaining six**:

- **The `si_norm` question has two answers and the language picks one.** Ask
  whether the language's unboxed integer representation IS `giNorm`'s. Dart's is,
  Java's is not; that one question is the difference between zero lines and
  ninety.
- **The scope-probe lowering is now copyable code, not just a pattern.** Two
  languages with the same Go arms (`js_kget`/`js_kset`) share the same three IR
  helpers verbatim. Check `abnf/jsrt.go` for the extern's implementation before
  writing a new one.
- **The neutral column is where the remaining work is.** Six new neutral externs
  landed here; python, ruby, go and kotlin emit most of them and now inherit them.
- A layer-2 file must be written against **the Go CODE**, not against the Go
  comments. Defect 2 above was a faithful implementation of a comment that its
  own function does not honour.

### launch.json entries wanted (not added - not my file)

Both pass natively today, at the working tree and from the clean archive:

```
"Dart native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/dart-to-llvm-ir.abnf", "tests/dart-test-features.dart", "-q", "-exe", "tests/dart-native.out"]
"Dart native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/dart-to-llvm-ir.abnf", "tests/dart-test-multifile.dart", "-q", "-exe", "tests/dart-native-multi.out"]
```

A full-syntax entry would be green too - `tests/dart-test-full.dart` passes
228/228 natively - but it stays out of the matrix for the same reason every other
`*-test-full.*` does: the ratchet lives in `./test.sh --full`, and
`tests/clang-check.sh` already runs the native one on every sweep.

## GO - DONE (2026-08-03). The gate is MET: 311/311 natively, byte-identical

**`tests/go-test-full.go` runs as a self-contained native binary, all 311
assertions pass in it, and its stdout, its stderr and its exit code are
byte-identical with `llvm.Run`.** `tests/clang-check.sh` reads

```
go              32106  ok, and the clang executable agrees
```

so go left the `ok (module only)` row for the run-it row, **no row is held, no
assertion is held back, and no floor change was needed or made.**
`abnf/jsrtgolang.go` is **kept**, as the plan requires: `llvm.Run` still uses it,
the native binary uses layer 2, and the two are now differentially compared over
16,550 probe lines plus the ratchet.

Oracles: **real `go` 1.26.5 darwin/arm64** (`go version`), present on this
machine and used on all three probes, and **Apple clang 17.0.0** for the native
link.

### THE COROUTINE QUESTION, which is what this migration was asked to answer

> **Go's goroutines and channels need NO coroutine primitive, and the reason
> generalises to nothing else in the set.** `grep -c 'js_genfn\|js_yield'` in
> `go-to-llvm-ir.abnf` is **0**. Concurrency in both halves is COOPERATIVE and
> DETERMINISTIC - `go-to-llvm-ir.abnf` says so at `GoStmt` and `abnf/jsrt.go`'s
> `goDrain` implements it: `go f(...)` puts an already-bound NULLARY CLOSURE on a
> run queue and returns, and a receive that finds its channel empty RUNS THE
> QUEUE, each queued goroutine to completion in spawn order, until one of them
> fills it. **Nothing ever suspends mid-body**, so there is no second stack to
> keep, and the floor's pthread-backed tag 15/16 is not reached.

The whole model in `languages/lib/go-rt.metajs` is **one module-level array and
one drain loop**, about 40 lines including the channel buffer:

```
var goQueue = []
js_gospawn(fn)      goQueue.push(fn)
goDrain()           while (goQueue.length > 0 && guard < 1000000) { shift; call }
goChanRecv(ch)      if buf empty -> goDrain(); if still empty -> zero value / deadlock
js_gochan_ready     drain first, then "is there one".  <- what a select arm asks
js_gorange(chan)    drain the WHOLE channel into a keys/vals dict, because under
                    this model the sequence is fully known at the first receive
```

Measured, not assumed: probe B drives a buffered producer closed from inside the
goroutine, two unbuffered senders, a receive from a closed channel, and both
arms of a `select` with a `default`, and `llvm.Run` and the native binary agree
line for line. **The cost estimate the coroutine section carried for `go` can be
struck.**

The one place our model and real go differ is exactly the one the ratchet file
already declares out of scope: with two goroutines racing on an unbuffered
channel, `<-c2, <-c2` yields `one two` here and `two one` under go 1.26.5, since
our order is spawn order and go's is the scheduler's. 1 line of 5,237.

### The extern split

`tests/go-test-full.go` drives **84** externs, and so does the union over every
`tests/go-test-*.go` - go's emitter has no test-specific arm.

| where | count | what |
|---|---|---|
| already in the C floor | 42 | the generic primitives of phase 3, the five the Lua pilot added, `js_truthy`, the WHOLE sized-integer family (`js_giarith`, `js_gicmp`/`js_gilt`/`js_gile`/`js_gigt`/`js_gige`, `js_giadd`, `js_giconv`, `js_gilit`, `js_gineg`, `js_gival`), `js_gflo`/`js_jfdiv` for the boxed double, and the two the lowered `js_gounctl` now calls (`js_ctl_kind`, `js_ctl_value`) |
| **layer 2**, language-NEUTRAL | 10 | `js_dict_new`, `js_goslice`, `js_gospread`, `js_is_type`, `js_jadd`, `js_pyin`, `js_pyset`, `js_range_key`, `js_range_len`, `js_range_val` |
| **layer 2**, Go's own | 32 | the `js_go*` family, `js_map_get`/`js_map_del`, `js_rundefers`, and the four width-carrying slot writers (`js_giadopt`, `js_gosetfield`, `js_gisetsl`, `js_gisetat`) |

`languages/lib/go-rt.ll` (22,313 lines, from an 1,823-line `.metajs`) exports
exactly those 42 symbols and nothing else - checked with `comm`, not assumed. The
layer-2 source compiles **byte-identically under `-frozen`** as well as under
goja.

The split against the five done before it:

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
java           29        9      16      ~90 lines, the unsigned-box long
csharp         35       15      18      NONE  <- sintRaw
dart           37       13      17      NONE
go             42       10      32      NONE
```

**Go has the largest FLOOR column of the seven and it is not close**, because
the sized-integer tag was specified against `abnf/jsrtint.go`, which is Go's:
every one of the eleven `js_gi*` externs is already a C function, so the
language whose value model drove the tag pays nothing for it. Go's own 32 is
second-largest after php's 63, and it is honest work rather than a value model -
`fmt`'s renderer, the map, panic/defer/recover, the channel, and the field and
method dispatch of a language with embedding.

### `si_norm` is what Go WANTS, and `js_gival` is NOT the hazard the audit named

Go is Dart's answer, not Java's. `abnf/jsrtint.go` states the invariant at the
type - "plain number == a signed 64 bit integer whose value is exact in a
double; jsGInt == any other integer" - which IS `giNorm`, which IS what `sint()`
applies. **There is no pair layer, no box layer, no `sintRaw` and no `jl*`
family in `go-rt.metajs`.**

**The second-order hazard is ABSENT, and the audit's own candidate is the proof
rather than the counter-example.** Part 3's Java table lists `js_gival` as row 6:
"go uses it as the unboxer for slice/index expressions, and it unboxes via
`si_float`. `si_val` is the right one for a Java-style box." Checked at the code:
`si_float`'s 64-bit UNSIGNED arm answers `d_from_ulong(fa(h))`, the magnitude -
and **that is correct for Go**, because Go really has `uint64`, an
unsigned-marked width-64 cell really IS a `uint64`, and `abnf/jsrtint.go`'s own
`js_gival` does literally the same thing:

```
if b.u && b.w == 64 { return float64(uint64(b.v)) }
return float64(b.v)
```

So `si_float` IS Go's `js_gival` and `si_val` is not. The row is a hazard only
for a language that carries a SIGNED value in an unsigned-marked cell, which is
Java's trick and nobody else's; with `sintRaw` landed, no further language will.
**The generalisation to correct in the plan: the hazard is not "a floor extern
reads tag 13 numerically", it is "a floor extern reads tag 13 numerically AND
the language lied about the signedness".** Same for rows 1, 4, 5 and 7 - go's
`js_gflo` needed no `js_gvnum` wrapper, and `js_gicmp`/`js_gilt`/... are the
floor's own and take the unsigned order exactly where Go wants it.

### THE ONE WALL, and it was lowered in the EMITTER - a NEW shape of the same idea

`js_gounctl` reads the completion value out of a CONTROL SIGNAL: a return signal
carries its value, a body that ran off its end yields undefined. A signal is the
floor's private **tag-10** cell and `seed_root` seeds no accessor for one, so
**layer 2 could not have written this extern at all** - the same position PHP's
and Swift's scope probes were in, reached from a different direction.

The answer is the same one: `js_ctl_kind` and `js_ctl_value` **are floor
externs**, and an EMITTER may call them. `go-to-llvm-ir.abnf`'s `emitUnctl` now
emits the two-arm select in IR at the single call site.

> **The detail that is worth copying, because truthy() would have been wrong
> here**: `js_ctl_kind` answers a **RAW INTEGER** (`mk_ctl`'s `fa` field; 0 for
> anything that is not a signal, 1 return, 2 break, 3 continue). The test has to
> be `NewICmp(IPredEQ, kind, 1)` and not `truthy(b, kind)`, which would also take
> kinds 2 and 3. That is the `__raw` finding of Part 3's item 2 met from the
> RESULT side inside an emitter rather than inside a shim, and it is the same
> silent-wrongness class.

Accounting, in Dart's corrected form: the lowering removes **one** extern from
layer 2 (43 -> 42) and adds **two** to the declare list, both already in the
floor, so the union went 83 -> 84. **The running count of item 1 is now php 2
externs lowered, swift 2, java 1 helper, csharp 0, dart 2, go 1 - six languages,
five lowerings, still no floor change.**

### The two cross-cutting findings, checked and MEASURED for go

1. **`core.truthyExt`.** `grep -n truthyExt languages/go-to-llvm-ir.abnf` is
   EMPTY, so go keeps compile-core's default `js_truthy`, which is the C floor's
   own function and already answers the raw 1/0 the emitted `icmp ne ..., 0`
   compares. **No `__raw` export.** The count over six ported languages is now:
   Lua and PHP need it; Swift, Java, C#, Dart and Go do not.
2. **`_raw` PARAMETERS: go has exactly one**, `js_gorest`'s START INDEX, which
   the emitter writes with a bare `handle(i + off)` at
   `go-to-llvm-ir.abnf:3877`. Every other `handle(k)` in the grammar goes to
   `js_closure` or `js_arg`, which are floor externs, and every operator selector
   goes through `emitStr`/`emitNum`. The count is now **3 / 0 / 0 / 1 / 1 / 0 / 1**
   for lua / php / swift / java / csharp / dart / go.
3. **The scope probe.** Go has NO scope-taking extern - `js_scope_get`,
   `js_scope_set`, `js_scope_decl` and `js_scope_new` are floor externs the
   emitter already calls directly, exactly as csharp's did. Nothing was owed.
   What Go had instead was the tag-10 probe above, which is the same idea over a
   different private representation.
4. **`jsChar`.** Not needed and not emitted: Go has no character type, a rune is
   an `int32`, and `grep -c 'js_char\|js_char_code\|js_kindex'` is 0.
5. **Item 3, the tag-16 `typeof` divergence.** Cannot bite: `js_genfn` and
   `js_yield` are emitted zero times (see the coroutine finding above), so no tag
   16 ever reaches `js_gomcall`'s dispatcher, and `typeof v == "function"` is
   written plainly there.

### The one thing layer 2 CANNOT do, stated rather than hidden

A Go **pointer cell** (`&n`, `new(int)`, `&p`) is a `{__class: $ptr}` object
whose `o` is **the SCOPE that holds the variable** and whose `k` is its name
(`go-to-llvm-ir.abnf`'s `emitPtrCell`). `abnf/jsrtgolang.go`'s renderer prints
`&` followed by the POINTEE, which it reads with `rt.scopeGet`. Layer 2 has no
such reader.

**Unlike a scope probe this one cannot be lowered into the emitter**, because it
is reached from arbitrarily deep inside a value tree at RUN TIME (`fmt.Println`
of a struct that holds a pointer that holds a struct), not from a fixed call
site. The object-backed form - `new(T)` over a one-slot object - IS readable and
is rendered exactly; the scope-backed form renders `&<nil>`.

Nothing in `tests/go-test-full.go` or in the three probes prints a scope-backed
pointer, so it is **unmeasured rather than wrong**. The one-line floor fix if a
program ever does is a `scopeGet(scope, name)` host builtin - which is also what
would let a scope PROBE go back into layer 2 for every language. **Reported, not
made.**

### Ground truth

```
matrix                325/325
--full                5,618 assertions, 0 languages whose halves disagree (go 311)
                      grep 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF' -> empty
--cross               119 programs, 0 divergent
go test ./abnf/       ok
clang-check           16/16, and go joins the RUN-IT row
gen-go-rt-ll.sh --check      go-rt.ll is up to date (22,313 lines)
-frozen recompile            go-rt.ll byte-identical to the goja build
every other gen-*-rt-ll.sh --check   up to date
jsbootstrap.ll        untouched (metajs-to-llvm-ir.abnf was not edited)
real go 1.26.5        darwin/arm64, used as the oracle on all three probes
```

**Verified from a CLEAN ARCHIVE, not the working tree** - `git archive 65a443b |
tar x`, then only the four Go files copied in, `go build` INSIDE, run there:

```
tests/gen-go-rt-ll.sh --check    go-rt.ll is up to date (22313 lines)
./go.exe                         full: 311 checks, 0 failures   (rc=0)
tests/clang-check.sh go          ok, and the clang executable agrees
probes A and C                   BYTE-IDENTICAL with llvm.Run
probe B                          the ONE known line (below)
```

Every Go program in `tests/`, `llvm.Run` against the native binary, stdout AND
stderr AND exit code:

```
go-test-1  go-test-complete  go-test-features  go-test-full  go-test-widen
go-test-multifile (-i tests/imports)                          all SAME (rc=0)
go-test-recognize   NOT COMPARABLE - both halves refuse it identically
                    ("grouped type declaration not implemented", a
                    recognition-only file)
```

### The differential probe - three probes, 16,550 lines, and the three defects

**Probe A, 11,231 lines**: every operand read out of a SLICE at run time, so
`go build` cannot fold it. 32x32 `int64` pairs times `+ - * & | ^ &^ / %` and all
six comparisons; each operand times `- ^` and times `<< >>` at thirteen shift
counts spanning 31, 32, 63, 64 and 65 through opaque `uint`-parameter functions;
then every conversion to `int8/int16/int32/uint8/uint16/uint32/uint64/uint/int`,
`++` and `*3` at the wrap point of all seven sized types, `uint64` division,
remainder, shifts, masking and comparison, and `float64(v)` of both the signed
and the unsigned reading. The operand bag carries both int64 edges, the int32 and
int16 and int8 edges and their neighbours, 2^32, and 2^53 with BOTH neighbours.

**Probe B, 5,237 lines**: 32x32 `float64` pairs times `+ - * /` and the six
comparisons (both zeros, 5e-324, DBL_MAX, 1e±100, 1e±20, the 1e15/1e16 and
1e5/1e6/1e7 `%v` boundaries, 0.49999999999999994, 2^53 and 2^53+1), the whole
`fmt` surface (nil, bools, arrays, slices, nested slices, maps with string, int
and bool keys - which fmt SORTS - structs, a `fmt.Stringer`, `Print`'s
space-between-two-non-strings rule, `Println`, `Printf` over `%v %d %s %t %c %q
%%`, an unknown verb and a MISSING operand, and `Sprint`/`Sprintf`), the string
surface (`len` in bytes, `range` by rune with byte offsets, `[]byte`/`[]rune`
round trips, byte slicing, `%q`) over seven strings including non-ASCII and an
astral one, the channel and select battery above, and panic/defer/recover
(recover of a string and of an int, LIFO defer order, a named result a defer
doubles).

**Probe C, 82 lines**: the four width-carrying SLOT writers, which the two big
probes do not reach - a `uint8`/`int16` struct field incremented past its wrap
point, a `map[string]uint8` and a `map[int]int8` entry, a `[]uint8` element and
an `[4]int32` array element, embedding and promoted methods, an interface value,
a type switch over ten dynamic types, pointer cells, and struct/array value
equality.

```
llvm.Run vs the native binary   BYTE-IDENTICAL on A and C, 11,313 of 11,313
                                lines; 5,236 of 5,237 on B (the one line below)
```

Three real defects, all in layer 2, all invisible to `./test.sh` because it
compares each engine with itself and the Go twin was right in every one:

1. **`strictEq` read the sized box as a DOUBLE.** `9007199254740992` is the
   largest integer `giNorm` keeps UNBOXED and `9007199254740993` is one past it,
   so `a == b` on that pair is a plain number against a box. `giEq` is
   `giVal(l) == giVal(r)`, an exact int64 comparison, and real go 1.26.5 agrees
   that they differ - reading both through a double rounds the second one down
   and answers TRUE. `sintCmp(a, b) == 0` is the exact test and is what the file
   now uses (two cells with the same payload compare equal whatever the
   signedness; two with different payloads never do). **2 probe lines out of
   11,231, and neither engine's own self-comparison could have seen it.**
2. **`js_gotypeis` was written against GO and not against the GO CODE**, which
   is the Dart lesson in its sharpest form. The twin's type switch has a
   `case float64` arm and **no arm for either number box**, so a `jsJFlo` and a
   `jsGInt` both fall through to the default `"any"`. Layer 2 answered
   `"float64"` for the first and `"int"` for the second - which is what REAL GO
   says for the first and is wrong for the second, and which broke byte identity
   either way. Reproduced exactly; the site says so at length. 2 probe lines.
3. **`%q` and byte slicing walk BYTES, not characters.** `gopQuoteBody`'s loop is
   `for i := 0; i < len(s); i++` over a Go string, so a multi-byte rune arrives
   as its individual UTF-8 bytes and each becomes the CODE POINT of that byte
   value: `%q` of `"héllo"` prints C3 A9 as U+00C3 U+00A9 and not as the letter.
   Layer 2 walked UTF-16 units. Fixed by encoding to UTF-8 first, which also
   fixed `s[lo:hi]` on a non-ASCII string. 6 probe lines.

**The ONE line that still differs, and it needs a floor change or nothing.**
`"日本語"[0:1]` is a Go string holding ONE byte of a three-byte rune. The Go twin
keeps the raw byte and prints it, and so does real go 1.26.5 - the two AGREE, and
the native binary cannot join them: **the C floor's strings are UTF-16 and cannot
represent a partial UTF-8 sequence at all.** Layer 2 decodes the byte window with
U+FFFD per unpaired byte, which is what the print pipeline would show for garbage
anyway. 1 line of 5,237, in the only construct that can produce it (a byte slice
cutting a rune in half). Recorded, not worked around.

### Where our Go and REAL go differ - measured, and NOT fixed here

With all three probes run under `go` 1.26.5, **559 lines of 16,550 differ**, and
`llvm.Run` and the native binary agree with each other on ALL of them, so every
one is pre-existing behaviour of BOTH halves. **Verified against a pristine
`git archive 65a443b` build, whose `llvm.Run` output is byte-identical to this
one on all three probes** - nothing here made or moved any of them.

```
 303   `&^` (AND NOT) is emitted as a 32-BIT bitwise pair. -1 &^ 4294967295 is
       -4294967296 in go and 0 here: the emitter reaches js_band over a 32 bit
       complement instead of the sized-integer tag's own op code 8.
 244   `<<` / `>>` through a `uint`-typed parameter take the RESULT's signedness
       from the SHIFT COUNT. `int64(-1) << 0` prints 18446744073709551615: the
       count is a uint64 cell and giWidthOf lets it decide the result's
       signedness, where Go fixes the result type as the LEFT operand's.
   6   `len([]byte(s))` counts RUNES. The value model has one array type, so
       []byte and []rune decode identically - documented at js_goconv in both
       halves and in abnf/jsrt.go itself.
   3   `%q` prints the UTF-8 BYTES of a non-ASCII rune (defect 3 above,
       reproduced deliberately) and `%!z(9)` where go writes `%!z(int=9)` -
       both already documented in abnf/jsrtgolang.go's header.
   2   an ASTRAL character is two U+FFFD, the same wtf8Clean difference the
       Swift port measured and recorded.
   1   goroutine scheduling order (see the coroutine finding).
```

The first two are the interesting ones and both are **emitter/floor questions,
not layer-2 ones** - they are wrong under `llvm.Run` too, and neither is in
`tests/go-test-full.go`, which is why a green ratchet did not see them. Fixing
either means `go-to-llvm-ir.abnf` plus `go-interpreter.abnf` plus a new ratchet
assertion, i.e. a self-contained job of its own; it is not this gate.

**BOTH FIXED 2026-08-04, and real `go` is the oracle** - it is installed, and
`tests/go-test-full.go` is a valid Go program that `go run` executes, which makes
the ratchet itself the differential test.

- **`&^`**: the expansion `a & (b ^ -1)` over the handle runtime's 32-bit `js_band`
  / `js_bxor` is gone from `emitBin`; `&^` and `&^=` now go through `giBase` like
  every other bitwise operator, and `js_giarith` has had a `&^` arm in all three
  halves the whole time (`abnf/jsrtint.go` `case "&^"`, `runtime.c`'s `si_apply`
  code 8, and the `si_arith` layer 2 calls). The interpreter half was already right.
  4 of 7 probe lines were wrong in the compiled half.
- **The shift result type**: "the result type of a shift is the type of the left
  operand" (Go spec, Arithmetic operators), but `js_giarith` reads the width and
  signedness off WHICHEVER operand is a box - `giWidthOf`, and `si_width_of` /
  `si_uns_of` in the floor. So `x >> s` with `x` a plain `int` of -8 and `s` a
  `uint8` of 1 was evaluated at 8 bits UNSIGNED and answered 124. The COUNT is now
  unboxed at the emitter: `js_gival` in front of it in `go-to-llvm-ir.abnf` and
  `szNum` in front of it in `go-interpreter.abnf`, both of which reduce a sized
  integer to its plain number and pass everything else through, leaving the left
  operand alone to decide the type. 4 of 7 probe lines were wrong in BOTH halves.

  **This one is a FLOOR defect that was closed at the emitter, and the floor still
  owes the general fix.** `giArith`'s `<<` / `>>` arms in `abnf/jsrtint.go`, and
  `si_apply` codes 9 and 10 in `languages/lib/runtime.c`, should take `w`/`u` from
  the LEFT operand only; `szArithSlow` in `languages/lib/interp-core.js` has the
  same shape at `szWidth(l, r)`. Every sized-integer language routes shifts through
  them - Java, Kotlin, C#, Swift and Dart all specify the same rule - so the same
  divergence is latent in five more languages, invisible for the same reason it was
  invisible here: both halves agree. Not made - those three files are other agents'
  this session.

Pinned as `tests/go-test-full.go` checks int31-int38, which are **4 red in the
interpreter half and 6 in the compiled one** from a clean archive of ad922a0, and
which real `go run` answers identically (319 checks, 0 failures, in all four of
`go`, the interpreter, `llvm.Run` and the native binary).

### What is owed, and what generalises

1. Nothing for the gate. `abnf/jsrtgolang.go` stays until the change is
   committed, and `abnf/jsrtint.go` stays regardless - kotlin still needs it.
2. ~~**`&^` and the shift result type**~~ **DONE 2026-08-04** in the emitter and
   the interpreter grammar, pinned as int31-int38, oracle real `go`. What is STILL
   owed is the FLOOR half of the shift rule - `giArith` in `abnf/jsrtint.go`,
   `si_apply` in `languages/lib/runtime.c` and `szArithSlow` in
   `languages/lib/interp-core.js` should read a shift's width and signedness from
   the LEFT operand only, which would close the same latent divergence in Java,
   Kotlin, C#, Swift and Dart. See the measurement above.
3. **A `scopeGet(name)` host builtin** would let layer 2 render a scope-backed
   pointer AND would let every language's scope probe go back into layer 2. Not
   needed by any current test.
4. The floor's `floStr` routes style 1 (floGo) to `jvm_flo_str`, which is JAVA's
   layout - so a tag-14 box built with the Go style renders as Java there.
   `go-rt.metajs` therefore never calls `floStr` and spells `strconv`'s
   `'g', -1, 64` itself (`goFloStr`, over the engine's own shortest digits).
   One `if (sty == 1) { return go_flo_str(...); }` in `jvmFloText` would remove
   it. Reported, not made.

What **generalises to the remaining five** (kotlin, js, typescript, python,
ruby):

- **A "private representation" probe is not always a SCOPE probe.** Go's was a
  control signal. The test is the same: does the extern have to open a cell only
  the floor knows? If so, look for a floor extern that answers the question and
  lower it into the emitter. **Six languages, five lowerings, still no floor
  change.** And check whether the floor extern answers a RAW integer before
  reaching for `truthy()`.
- **The second-order hazard is narrower than the Java audit implied.** It needs a
  language that carries a SIGNED value in an unsigned-marked cell. With `sintRaw`
  landed, no remaining language will, so rows 1-7 of that table can be read as
  "check the signedness claim", not "wrap every call".
- **Cooperative concurrency needs no coroutine.** Before costing a second stack,
  read the grammar's own model: `go` queues and drains, and that is one array.
- **Write layer 2 against the Go CODE, not against Go**, and not against the Go
  comments either. Defect 2 is a case where being RIGHT about the language broke
  the gate.

### launch.json entries wanted (not added - not my file)

All three pass natively today, at the working tree and from the clean archive:

```
"Go native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/go-to-llvm-ir.abnf", "tests/go-test-features.go", "-q", "-exe", "tests/go-native.out"]
"Go native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/go-to-llvm-ir.abnf", "tests/go-test-multifile.go", "-q", "-exe", "tests/go-native-multi.out"]
"Go native binary, widened syntax (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/go-to-llvm-ir.abnf", "tests/go-test-widen.go", "-q", "-exe", "tests/go-native-widen.out"]
```

A full-syntax entry would be green too - `tests/go-test-full.go` passes 311/311
natively - but it stays out of the matrix for the same reason every other
`*-test-full.*` does: the ratchet lives in `./test.sh --full`, and
`tests/clang-check.sh` already runs the native one on every sweep.

## RUBY - DONE (2026-08-03). The gate is MET: 267/267 natively, byte-identical

**`tests/ruby-test-full.rb` runs as a self-contained native binary, all 267
assertions pass in it, and its stdout, its stderr and its exit code are
byte-identical with `llvm.Run`.** `tests/clang-check.sh` reads

```
ruby            20303  ok, and the clang executable agrees
```

so ruby left the `ok (module only)` row for the run-it row, **no row is held, no
assertion is held back, and no floor change was needed or made.**

Ruby is the first of the four languages with **no dedicated Go twin** - its
semantics live in the shared `abnf/jsrt.go` (`rubyNumBin` / `rubyBin` / `rubyStr` /
`rubyInspect` / `rubyEq` / `rubySpaceship` / `rubyFormat` / `rubyMethod` /
`rubyArrayMethod` / `rubyIsA` / `rubyExc` / `memberCall` and the `js_r*` extern
table) plus `abnf/jsrtregex.go` for Regexp and MatchData. **The prediction that
this would drag more of the shared file along than the extern count suggests was
correct, and the mitigation is the finding that matters for python.**

### THE FINDING FOR PYTHON: a layer-2 file may IMPORT a shared source

`languages/lib/regex.js` is the 1,559-line backtracking regex engine that every
INTERPRETER grammar `include`s into its tag script, and `abnf/jsrtregex.go` is its
line-by-line Go port. It is written in the same MetaJS a layer-2 file is - so layer
2 needs no third copy:

```js
import "./regex.js"
```

compiles it into `ruby-rt.ll` with everything else. `metajs-to-llvm-ir.abnf`
already resolves `import "./x"` relative to the file, `-rt-lib`'s lazy boot runs
the imported top-level declarations into the same pinned scope, and the six
`js_rx*` externs then cost only their ~460 lines of Ruby-specific glue instead of
~2,000. **This is the first time a layer-2 library has imported a shared source
rather than restating it, and python/js/typescript/kotlin all emit regex externs
over the same engine.** Check for an existing MetaJS twin before writing anything.

### The extern split

`tests/ruby-test-full.rb` drives **93** externs, and so does the union over every
`tests/ruby-test-*.rb` - ruby's emitter has no test-specific arm.

| where | count | what |
|---|---|---|
| already in the C floor | 31 | the generic primitives of phase 3, the five the Lua pilot added, `js_truthy`'s replacement (see below), the scope externs, and the three the lowered `defined?` probe now calls |
| **layer 2**, language-NEUTRAL | 8 | `js_dict_new`, `js_is_type`, `js_jadd`, `js_pyexc`, `js_pyget`, `js_pylen`, `js_pyset`, `js_supercall` |
| **layer 2**, Ruby's own | 54 | the `js_r*` family (49) and the `js_rx*` regex wrappers (5) |

`languages/lib/ruby-rt.ll` (54,046 lines, from a 2,665-line `.metajs` PLUS the
1,559-line imported `regex.js`) exports exactly the 62 symbols the union needs plus
`js_rformat` and `js_rprint`, which other ruby programs reach - checked, not
assumed. The layer-2 source compiles **byte-identically under `-frozen`** as well
as under goja.

The split against the five done before it:

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
java           29        9      16      ~90 lines, the unsigned-box long
csharp         35       15      18      NONE  <- sintRaw
dart           37       13      17      NONE
ruby           31        8      54      NONE - but a whole numeric TOWER
```

**Ruby's own 54 is the second-largest surface after PHP's 63, and the reason is not
the value model but the SURFACE**: Ruby spells its operators, its parameter forms
(`*rest`, `**kwrest`, `&blk`, keyword args, block auto-splat), its `$globals`, its
`@@class` variables, its `defined?`, its `raise`, its class objects and its whole
builtin method library as separate externs. None of them is hard; there are simply
a lot of them.

### The `si_norm` question does not arise, and neither does tag 14

Dart asked "is the language's unboxed representation the one `si_norm` produces".
**For Ruby the question has no meaning: Ruby never builds a tag-13 cell at all.**
Its Integer is the plain, un-truncated double and its Float is a `{__flo: true, f}`
OBJECT box - the same representation `abnf/jsrt.go` uses (`jsFlo`/`jsRat`/`jsCpx`
are Go value structs whose fields this file mirrors one for one) - because Ruby
needs `7 / 2 == 3` and `7.0 / 2 == 3.5` to hold at the same time and one double
cannot carry both. So neither floor number tag is reached from `ruby-rt.metajs`
except in one place, and that place is worth reading (below).

**The consequence the ratchet does NOT see**: those four boxes are Go COMPARABLE
value structs, so two of them with equal fields are `===` equal and hash to one
dict key; here they are objects compared by IDENTITY. **Symbols are therefore
INTERNED** - `rbSym` hands out one object per name, keyed `"$" + name` - which
restores `===` and Hash keying exactly for the one of the four a Ruby program
actually uses as a key. Float/Rational/Complex are not interned and are compared by
VALUE everywhere layer 2 does the comparing (`rbEq`, `rbDictFind`); the only way to
see the difference would be a floor `js_seq` on a number, and the emitter emits
`js_seq` on exactly three things - `x === nil`, a `typeof` result and a class
descriptor. Audited, not assumed - the same grep Java and C# ran for their
`{__char}` box.

### The three cross-cutting findings, checked and MEASURED for ruby

1. **`core.truthyExt`.** `grep -n truthyExt languages/ruby-to-llvm-ir.abnf` is
   EMPTY - and ruby goes one step further than the four languages that merely keep
   compile-core's default. It **overrides `truthy()` itself** with a single
   unsigned handle compare:

   ```
   truthy = function(b, v) { return b.NewICmp(IPredUGE, v, handle(3)) }
   ```

   because the value handles 0/1/2 are `undefined`/`null`/`false` and 3 is `true`,
   so "`>= 3`" IS Ruby truthiness (only nil and false are falsy) with **no extern
   in the path at all**. That is sound natively for a reason worth checking rather
   than assuming: `runtime.c` seeds `H_UNDEF`/`H_NULL`/`H_FALSE`/`H_TRUE` as
   0/1/2/3, the same four handles `abnf/jsrt.go` uses. The `__raw` tally over six
   ported languages is now **lua and php need it; swift, java, csharp, dart and
   ruby do not**.
2. **`_raw` PARAMETERS: ruby has THREE, in two functions** - `js_rargrest(args,
   i_raw)` and `js_rblockarg(args, i_raw, n_raw)`, whose counts the emitter writes
   with a bare `handle(k)` and whose Go arms read `int(a[1])` off the raw word.
   Every other selector goes through `emitStr`/`emitNum`. The count is now
   **3 / 0 / 0 / 1 / 1 / 0 / 3** for lua / php / swift / java / csharp / dart /
   ruby, which is the plan's own advice: check each emitter, do not assume.
3. **The scope probe, the FIFTH lowering.** `js_rdefined(scope, name, kind)` walked
   the chain for `defined?(x)` and could never have been written in layer 2. It is
   now the emitted IR helper `rb_defined` in `ruby-to-llvm-ir.abnf`, in PHP's
   single-helper shape, over `js_scope_typeof`. `llvm.Run` stayed at 267/267 and no
   floor change was needed for the fifth time. The accounting Dart asked for:
   **one extern left layer 2 and one floor extern (`js_scope_typeof`) joined the
   declare list**, so 93 stayed 93 and what layer 2 has to WRITE fell by one.

   > The same inexactness Swift recorded: `js_scope_typeof` answers `"undefined"`
   > for a slot that HOLDS undefined and for an absent name alike. A Ruby local is
   > created-on-assign and a declared one holds nil, so the only slot that can hold
   > undefined is an OPTIONAL PARAMETER the caller omitted - `defined?(p)` on such
   > a parameter would answer nil here and its kind in the Go twin. Nothing in
   > `tests/` does that (the four `defined?` uses are a constant, a `$global`, a
   > bound local and an absent name); the fix, if one ever does, is to declare the
   > parameter slot with `hNull` before `js_arg` overwrites it.

4. **Tag 16 / generators: measured absent.** `grep -c 'js_genfn\|js_yield'
   languages/ruby-to-llvm-ir.abnf` is **0**. Ruby's `yield` is a call of the block
   the caller passed, not a generator, and no file in `tests/` uses `Fiber` or
   `Enumerator`, so WALL 1 does not touch this language and the tag-16 `typeof`
   divergence cannot be reached. Verified rather than inherited from the
   investigation note.
5. **`jsChar` was not needed.** Ruby has no character type - `?a` is a
   one-character String - and the emitter emits `js_char` / `js_char_code` /
   `js_kindex` zero times.

### The ONE place a floor tag IS used, and the FMA defect it fixed

Ruby's `& | ^` on Integers are `float64(int64(x) op int64(y))` in the Go twin,
which MetaJS's own `&`/`|`/`^` cannot reproduce because they are ToInt32 and would
wrap at 32 bits. The floor's sized-integer tag does exactly that arithmetic, so
`rbBits64` is four calls - `sintRaw(sintHi(v), sintLo(v), 64, false)` on each side,
`sintOp(5|6|7, ...)`, `sintNum` back. The same tag then paid for itself a second
time in `rbBaseStr` (below).

**And then the probe found the defect this whole exercise exists to find.**

> **The Go compiler FUSES `x - math.Floor(x/y)*y` into an FMSUB on arm64**, so the
> product and the subtraction are done at infinite precision and only the result is
> rounded. clang does not fuse the C floor's equivalent. `9007199254740991 % -3`
> is therefore **-2 under `llvm.Run` and -1 natively** - and real ruby 2.6.10 says
> **-2**, so the fused answer is also the CORRECT one and this was layer 2's
> defect, not the Go twin's. It is the same expression in Ruby's Integer `%`, its
> Float `%`, `divmod` and `rubyGcd`, and the last of those was reducing
> `90071992547409900/3` to `30023997515803300/1`.
>
> `rbFmaSub(x, q, y)` reproduces the fused result exactly with the standard Dekker
> two-product / two-sum emulation of an FMA. The one subtlety is that a Float
> remainder legitimately has `|q|` near DBL_MAX (`1e308 % 2.5`), where the `2^27`
> split overflows, so each operand is first scaled DOWN by a power of two (exact)
> and the exact product scaled back (also exact, because `|p * scale|` is about
> `|x|`). 424 probe lines, and the first guard at `2^512` was too tight and left 34
> of them - the measurement is what caught that too.
>
> **Every remaining language whose runtime computes a remainder as
> `x - floor(x/y)*y` in Go has this.** It is invisible to `./test.sh`, which
> compares each engine with itself, and invisible to a small ratchet, because it
> only shows once `q*y` leaves the exactly-representable range. `grep -n 'math.Floor(.*)\*' abnf/jsrt*.go`
> is the one-line check.

### Ground truth

Oracle: **`/usr/bin/ruby` 2.6.10 (2018)**, which is the only ruby on this machine.
The plan's own caution applies and is respected: 2.6 rejects the Ruby-3 forms the
CORPUS uses, so it is **not** used on `tests/ruby-test-full.rb` at all - only on the
differential probe, which is written in the 2.6-compatible subset, and the two
places it refused even that (`(..9)`, a beginless range, is 2.7; `0.0 % 0.0` raises
`ZeroDivisionError`) are stated below rather than worked around silently.

```
matrix                325/325
--full                5,618 assertions, 0 languages whose halves disagree (ruby 267)
                      grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF' -> empty
--cross               119 programs, 0 divergent
go test ./abnf/       ok
clang-check           16/16, and ruby joins the RUN-IT row
gen-ruby-rt-ll.sh --check    ruby-rt.ll is up to date (54,046 lines)
every other gen-*-rt-ll.sh --check   up to date (11 of them)
-frozen recompile     ruby-rt.ll byte-identical to the goja build
jsbootstrap.ll        untouched (metajs-to-llvm-ir.abnf was not edited)
```

**Verified from a CLEAN ARCHIVE, not the working tree** - `git archive <rev> | tar
x`, then only the four Ruby files copied in, `go build` INSIDE, run there. That
matters here because `languages/lib/runtime.c` and four other languages' files were
other agents' concurrent edits while this ran. Done TWICE: at `65a443b`, and again
at `3f91d7b` after go and the tag-16 `type_of` floor change landed under it -
identical results both times, so nothing here depends on either:

```
tests/gen-ruby-rt-ll.sh --check    ruby-rt.ll is up to date (54,046 lines)
./rb.exe                           full: 267 checks, 0 failures   (rc=0)
tests/clang-check.sh ruby          ok, and the clang executable agrees
the 36,378-line probe              BYTE-IDENTICAL to the working tree's run
```

Every Ruby program in `tests/`, `llvm.Run` against the native binary, stdout AND
stderr AND exit code:

```
ruby-test-1  ruby-test-2  ruby-test-4  ruby-test-complete  ruby-test-features
ruby-test-full  ruby-test-try  ruby-test-multifile (-i tests/imports)
                                                            all SAME (rc=0)
ruby-test-recognize   NOT COMPARABLE - both halves refuse it identically
                      ("conditional assignment (||=) not implemented", a
                      recognition-only file)
an uncaught raise     SAME message, rc=1 in both; llvm.Run's driver adds one
                      "  ==> Fail" line of its own, the same difference
                      php-test-undefconst, java and dart record
```

### The differential probe - 36,378 lines, and the four defects it found

Every operand is read out of an ARRAY at run time, so nothing can be folded: 32x32
integer pairs times `+ - * / % & | ^ <=> == < >=`; 32 integers times the full unary
and method set (`- ~ abs to_s to_s(2) to_s(16) to_s(36) to_f to_r even? odd? zero?
positive? negative? succ pred floor ceil round inspect class`) times 16 shift
counts for `<< >> **`; 32x32 double pairs times `+ - * / % <=> == < <=`; the double
unary and predicate set and the 32x32 mixed Integer/Float promotion grid; 8x8
Rational pairs and 4x4 Complex pairs with their renderings; 20x20 string pairs
(`+ == <=> < include?`) and per-string `length upcase downcase to_sym * [] [-]`;
**22 printf directives times 84 arguments**; the Array/Hash/Range method set; 11
regexps against 10 subjects times `=~ match? match m[0] captures scan sub gsub
split pre_match post_match begin end case/when`; the class, module, `is_a?` and
`case`-equality matrix over 10 values and 14 class objects; 10x10 raise/rescue
pairs; blocks, procs, lambdas and `&:sym`. The operand bags carry both zeros, both
int53 edges, 2^31/2^32/2^53 and their neighbours, 1e±100, DBL_MAX, 5e-324,
0.49999999999999994 and non-ASCII strings.

```
llvm.Run vs the native binary      BYTE-IDENTICAL, 36,378 of 36,378 lines
```

It found **four real defects, all in layer 2, all invisible to `./test.sh`**:

1. **The FMA remainder**, above. 424 lines.
2. **`to_s(36)` and the saturating `int64` cast.** The digit alphabet was 16 long,
   so `100.to_s(36)` printed "2" and not "2s"; and Go's `strconv.FormatInt(int64(x),
   base)` converts through int64, which SATURATES on arm64 - `int64(1e100)` is
   `0x7fffffffffffffff` and its hex is `7fffffffffffffff`, not the 84 digits a
   float loop produces. Both fixed with the sized-integer tag on the UNSIGNED
   reading of the magnitude, which is the one shape whose negation is exact for
   int64's minimum too - `java-rt.metajs`'s trick, reused. 45 lines.
3. **`strconv.FormatFloat` is EXACT and rounds HALF TO EVEN.** `Math.floor(a *
   10^prec + 0.5)` is wrong twice over: it rounds half AWAY from zero (Go's `%.0f`
   of 2.5 is "2", not "3") and it loses every digit past 2^53, where `%f` of 1e100
   has to print all 101 of them. Replaced with the exact decimal expansion of the
   double - a finite double is `m * 2^e`, so its exact decimal is `m` doubled `e`
   times for `e >= 0` and `m * 5^-e` with the point `-e` digits from the right for
   `e < 0`, on a little-endian base-10000 bignum - plus a half-to-even rounder.
   `-0.0` needed its sign READ (`1 / x < 0`) rather than compared, which is the
   signed-zero trap arriving for the SIXTH language. 70 lines.
4. **Width and precision in `format` count BYTES.** Go pads to `len(body)` and cuts
   with `body[:prec]`, both of which are UTF-8 byte counts, so `"%10s" % "café"`
   gets FIVE spaces and not six. The floor's own `byteLen` is the answer, and it is
   the same primitive `lib/compile-core.js` already uses in `emitStr`.

A fifth was avoided rather than fixed: `strings.ToUpper` is the full Unicode SIMPLE
mapping and an ASCII loop printed "CAFé". **`runtime.c`'s own `toUpperCase` /
`toLowerCase` are already that mapping** - its 328 ranges were generated from the Go
runtime's own answers - so layer 2 calls the builtin instead of carrying a third,
worse copy.

### Against REAL ruby - measured, with the version caveat stated

`/usr/bin/ruby` 2.6.10 ran the probe to line 32,983 before its own
`ZeroDivisionError` on `0.0 % 0.0` (which is MRI behaviour, not a defect of ours).

```
the 14,436-line INTEGER prefix ruby reached first        0 of 14,436 differ
the 32,983 lines it reached after % by zero was guarded  2,415 differ
```

**Zero differences over the whole integer surface** - arithmetic, all six
comparisons, `<=>`, the three shifts at 16 counts, `**`, `to_s` in bases 2/16/36,
`to_f`, `to_r`, every predicate. That is the strongest single piece of evidence in
this port, and it covers the FMA fix directly.

The 2,415 are **four pre-existing classes, and `llvm.Run` and the native binary
agree with each other on every one of them**, so none is this port's:

```
  1,556   the FLOAT FORMATTER at its boundary: ours renders 9007199254740991.0
          plainly where ruby writes 9.007199254740991e+15. Ruby's Float#to_s goes
          scientific one decade earlier than ours. Three halves at once
          (rubyFloStr in jsrt.go, floStr in ruby-interpreter.abnf, rbFloStr here)
          and --full pins the current one, so it is a job of its own - exactly the
          reason the Lua and Swift formatter boundaries were not fixed either.
    530   INTEGER EXACTNESS past 2^53: `-3 * 9007199254740991` and `100 ** 11` are
          BIGNUMS in ruby and doubles here. That is the documented value model of
          both halves (abnf/jsrt.go says so at the jsFlo type) and not a defect.
    309   -0.0 THROUGH A REMAINDER: `-0.0 % 1.0` is -0.0 in ruby and +0.0 in BOTH
          our halves. This is the signed-zero trap in its Dart shape, and the Go
          twin has it too - so it is a three-halves fix and not this gate.
     20   Rational/Complex rendering and Float#round at a tie.
```

### What is owed, and what generalises to PYTHON

1. Nothing for the gate. `abnf/jsrt.go`'s Ruby half stays until the change is
   committed - unlike the five languages before it, it is NOT a file that can ever
   go, because python, js, typescript, kotlin and go share it.
2. The float formatter boundary above, if and when someone does the three-halves
   job; `-0.0 % 1.0`, same shape, both halves currently agree.
3. `Integer#%` on a value past 2^53 is now EXACT where the Go twin is exact by
   accident (arm64 fusion). If the Go twin is ever built for an architecture
   without FMA, `abnf/jsrt.go` and this file diverge again - the fix would be to
   spell the exact subtraction in Go too, which is three lines of `math.FMA`.

What **generalises to python**, which shares Ruby's position of having no dedicated
twin:

- **`import "./regex.js"` from layer 2.** Python emits `js_pyre*` externs over the
  same shared engine, and the whole engine is now known to compile under
  `metajs-to-llvm-ir.abnf -rt-lib` and to link. That is the single biggest saving
  available to it, and the same trick applies to any shared `languages/lib/*.js`.
- **The `js_py*` family is now written twice in layer 2** (dart's thirteen and
  ruby's eight), and python is the language they are NAMED for - `js_pyget`,
  `js_pyset`, `js_pylen`, `js_pyexc`, `js_dict_new`, `js_is_type`, `js_supercall`
  transfer with only their `__getitem__` / `__setitem__` / `__len__` arms to add
  back, which Ruby and Dart both deleted as unreachable.
- **The FMA remainder.** `abnf/jsrt.go`'s python arms compute `%` the same way.
  Grep before writing.
- **`strconv.FormatFloat` exactness and half-to-even**, and **`byteLen` for printf
  width** - python's `%`-formatting and `format()` go through the same Go
  primitives, and the exact-decimal bignum in `ruby-rt.metajs` is language-neutral.
- **The scope probe**, sixth confirmation. Python's `js_pyglobal`-shaped externs are
  the same pure probe; check the emitter before writing anything into layer 2.
- **No dedicated twin does NOT mean more work in layer 2** - it means the
  SPECIFICATION is spread out. The extern split was the usual shape; what cost time
  was reading `abnf/jsrt.go` in five places instead of one. Budget for the reading,
  not for the writing.

### launch.json entries wanted (not added - not my file)

All three pass natively today, at the working tree and from the clean archive:

```
"Ruby native binary, full syntax (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/ruby-to-llvm-ir.abnf", "tests/ruby-test-full.rb", "-q", "-exe", "tests/ruby-native-full.out"]
"Ruby native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/ruby-to-llvm-ir.abnf", "tests/ruby-test-features.rb", "-q", "-exe", "tests/ruby-native.out"]
"Ruby native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/ruby-to-llvm-ir.abnf", "tests/ruby-test-multifile.rb", "-q", "-exe", "tests/ruby-native-multi.out"]
```

## JS and TYPESCRIPT - DONE (2026-08-03). The gate is MET for BOTH, on ONE layer 2

**`tests/js-test-full.js` runs as a self-contained native binary and prints
`full: 354 checks, 0 failures`, and `tests/typescript-test-full.ts` prints
`full: 276 checks, 0 failures`. Both are byte-identical with `llvm.Run` on stdout,
on stderr and in their exit code.** `tests/clang-check.sh` reads

```
js              23097  ok, and the clang executable agrees
typescript      21790  ok, and the clang executable agrees
```

so both left the `ok (module only)` row for the run-it row, **no row is held, no
assertion is held back, and no floor change was needed or made.**
`abnf/jsrt.go` + `abnf/jsrtjsprint.go` + `abnf/jsrtjsbig.go` + `abnf/jsrtregexjs.go`
are **kept**, as the plan requires: `llvm.Run` still uses them, the native binaries
use layer 2, and the two are differentially compared below.

The reproducible native command, for both languages:

```
mec languages/js-to-llvm-ir.abnf         tests/js-test-full.js         -q -exe /tmp/js-native.out
mec languages/typescript-to-llvm-ir.abnf tests/typescript-test-full.ts -q -exe /tmp/ts-native.out
/tmp/js-native.out ; echo $?      # full: 354 checks, 0 failures / 0
/tmp/ts-native.out ; echo $?      # full: 276 checks, 0 failures / 0
```

Oracles: **node v24.18.0** (present on this machine, used throughout) and Apple
clang for the native link. `tsc` is not on this machine.

The row WAS held for part of the session, with the `clang-check: native-row-held`
marker, and released the moment the last assertion went green - which is what the
mechanism is for.

### ONE layer 2 for BOTH languages, and that was decided by measurement

The union of externs over every `tests/js-test-*.js` is **87** and over every
`tests/typescript-test-*.ts` is **86**. The two sets differ by exactly three names:

```
js_jstonumber   js only     Number(v) - typescript's ratchet never reaches it
js_newtarget    js only
js_sub          ts only, and it is the C FLOOR'S OWN
```

So **TypeScript's layer 2 is a strict SUBSET of JavaScript's**, there is no
`typescript-rt.metajs`, and `typescript-to-llvm-ir.abnf` links `lib/js-rt.ll`
verbatim - the same arrangement the two grammars already have on the Go side, where
they share `abnf/jsrt.go` + `abnf/jsrtjsprint.go` + `abnf/jsrtjsbig.go` and
typescript has no dedicated twin at all. That question was the first one this
migration was asked to answer and the answer is: **share it outright, no overlay.**

### The extern split

`tests/js-test-full.js` drives **83** externs and the union over every
`tests/js-test-*.js` is **87**; typescript's union is **86**.

| where | count | what |
|---|---|---|
| already in the C floor | 39 | the generic primitives, `js_truthy`, `js_keys`, the scope externs, and `js_genfn` / `js_yield` / `js_ctl_*` |
| **layer 2**, language-NEUTRAL | 10 | `js_has`, `js_defprop`, `js_iterable`, `js_supercall`, `js_supget`, `js_supset`, `js_call_new`, `js_this`, `js_newtarget`, `js_scope_set_or_create` |
| **layer 2**, JS's own | 38 | the `js_js*` family (34) and the `js_jsx*` regex wrappers (4) |

`languages/lib/js-rt.metajs` exports **59** symbols rather than 48, and the eleven
extra are the whole of what the `-exe`-only lowerings needed - `js_jssetthis`,
`js_jssetnew`, `js_jsmset`, `js_jstypeof`, `js_jsseq`, `js_jssne`, `js_jsneg`,
`js_jsnot`, `js_jstruthy` (exported from `js_jstruthy__raw`), `js_jstpl` and
`js_jsstr`. **None of them has a Go twin and none needs one**, because the module
that declares them is only ever built by `-exe`. `js_scope_set_or_create` is present
as a LOUD STUB, php-style: the emitter lowers it and it is never called.

The split against the seven done before it:

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
java           29        9      16      ~90 lines, the unsigned-box long
csharp         35       15      18      NONE  <- sintRaw
dart           37       13      17      NONE
ruby           31        8      54      NONE - but a whole numeric TOWER
go             42       10      32      NONE
js / ts        39       10      38      NONE - but a BigInt tower and a regex engine
```

The prediction in the brief - "expect the floor column to be the largest yet,
because these are the languages closest to MetaJS itself" - is **half right and
worth correcting**. 39 is the second-largest floor column after go's 42, but the
reason is not closeness to MetaJS: it is that JavaScript needs neither the sized
integer tag nor the boxed double, so the floor primitives it does use are the plain
ones every language uses. What closeness to MetaJS actually buys is elsewhere and
it is much bigger than a column of the table - see "the MetaJS twin already exists"
below.

### THE FINDING THAT DOMINATES THIS MIGRATION: the MetaJS twin already exists

Ruby found, this same session, that a layer-2 file may `import` a shared MetaJS
source rather than restate it (`import "./regex.js"`). **For js and ts that finding
generalises from one file to the whole port.**

`languages/js-interpreter.abnf`'s start script is written in the SAME MetaJS subset
a layer-2 file is, and it already carries, function for function, the answers
`abnf/jsrtjsprint.go` and `abnf/jsrtjsbig.go` give:

```
jsIsObjLike  ~2924   jsStr      ~2942   jsToPrim    ~2971   jsLooseEq  ~2984
jsTruthy     ~3014   jsBuiltin  ~3236   jsStringMethod ~3292
jsStringFn   ~3593   jsBigIntFn ~3597   isBigInt / biToStr / biCmp / the bi* family
include("lib/regex.js") ~2894, and the whole JS regex glue at ~4416-4700
```

`abnf/jsrtjsprint.go`'s own header says so in as many words: *"The interpreter
halves carry the SAME answers written in their own tag script - jsStr, jsLooseEq,
jsRelate, jsBuiltin, jsObjectGlobal. The two files must be read together."* The
matrix has kept those two in agreement for a long time, so the MetaJS half is not
merely available, it is **better ground truth than a fresh translation from Go**,
and it is already inside the frozen subset (no `for..in`, no `split`, no exponent
literals - exactly the things a fresh translation gets wrong).

> **The rule to carry to python and kotlin: before porting a Go twin, grep the
> LANGUAGE'S OWN INTERPRETER GRAMMAR for the same function.** Ruby's version of
> this rule was about one shared library; the general form is that every language
> in this repo has a complete MetaJS implementation of its own semantics sitting in
> `languages/<lang>-interpreter.abnf`, and layer 2 is the same language. What
> differs is only the VALUE MODEL - the interpreter has its own object
> representation, layer 2 works on the floor's handles - so the port is an
> adaptation, not a translation.

### WALL 1 - the dynamic `this`, and it is a NEW SHAPE of the private-representation probe

`js_this` and `js_newtarget` read `abnf/jsrt.go`'s `thisStack` / `newTargetStack` -
**the Go runtime's own CALL STACK.** That is not a scope (php, swift, dart, ruby)
and not a control signal (go); it is state the C floor **does not keep at all**:
`runtime.c`'s `js_call` drops `self` for a tag-7 closure and hands `jsdispatch` only
`(env, args)`.

So this is the first wall in part 3 that could NOT be closed by finding a floor
extern that already answers it - **there is none, and the "find the floor extern
and lower the call" rule runs out here.** The answer that replaces it:

> **If the floor keeps no such state, layer 2 can keep it, provided the EMITTER
> hands it over.** Layer 2 is a module with globals of its own. The emitter's whole
> half is one extern call, `js_jssetthis(receiver)`, emitted immediately before
> every `js_call`, `js_jsxcall`, `js_jsmcall` and `js_supercall`.

Two details make it cheap, and both are worth copying:

1. **No restore is needed**, which looks like a bug and is not. `this` is read
   exactly ONCE per function, at ENTRY, where the emitter binds it into the scope
   (`js_scope_decl(scope, "this", js_this())`). A nested call that overwrites the
   cell cannot be observed by the caller, which already copied it.
2. **The lowering is emitted ONLY under `-exe`** (`nativeBuild = c.exePath != ""`).
   The module `llvm.Run` sees is byte-for-byte the one it always was - verified by
   diffing the emitted module for `tests/js-test-full.js` and
   `tests/typescript-test-full.ts` against a clean `git archive` of the base commit
   - so **nothing in the matrix can move**, which is the property that made this
   change safe to make at all. Two modules that differ is already true of
   `emitDispatch`.

Measured natively, against `llvm.Run`, on a program with an object-literal method
reading `this.n` and a plain function reading `typeof this`:

```
                      llvm.Run      native
o.get2()  (this.n)       7             7
plain()   (typeof this)  undefined     undefined      rc=0 both
```

### WALL 2 - the ACCESSOR, and the same answer a second time

`js_defprop` stores a `*jsAccessor` - another Go TYPE. The floor's `get_member` /
`set_member` have never heard of it, so a `get x()` member read through the floor
would answer the box and a write would replace it. Layer 2 can represent the box
(it is an ordinary object here) but only if **every member read and write goes
through layer 2**, and by default they do not: the emitter emits the floor's
`js_get` / `js_set` at 30 and 23 sites plus `core.indexExt`.

Same answer as wall 1, and it cost the emitter three lines: under `-exe` only,
`js_get` -> `js_jsmget` (which already existed on both sides) and `js_set` ->
`js_jsmset` (a NEW extern that needs no Go twin, because it is emitted only for a
native build), plus `core.indexExt = "js_jsmget"`. Layer 2 delegates straight back
to the floor for every receiver that carries no accessor.

### WALL 4 - the BigInt is a Go TYPE, and it reaches SIX floor externs

`abnf/jsrtjsbig.go`'s BigInt is a Go type, and `abnf/jsrt.go` therefore carries an
arm for it in `truthy` (927), `toString` (1061), `strictEq` (1214), `typeOf` (1547)
and `js_neg` (5255). In layer 2 a BigInt can only be an object box, so the C floor's
own versions of those answer **"truthy", "identity", "object" and NaN** - and the
ratchet sees it immediately: `typeof 10n === "bigint"` (`num7`, `big1`),
`10n === 10n` (`big13`), `-(2n ** 100n)` (`big10`, `big19`), `!0n` (`big20`,
`big21`) and every `while (c < 3n)`.

Same answer as walls 1 and 2, and it cost the emitter six small functions. Under
`-exe` ONLY, these are routed to layer-2 twins that delegate to the floor for every
value that is not a BigInt:

```
`===`   js_seq     -> js_jsseq        `!==`  js_sne   -> js_jssne
unary - js_neg     -> js_jsneg        `!`    js_not   -> js_jsnot
typeof  js_typeof  -> js_jstypeof     truthy core.truthyExt -> js_jstruthy
```

Two details are worth copying.

**The INTERNAL probes are deliberately left on the floor.** Either grammar emits
`js_typeof` twice and `js_seq` five times for its own bookkeeping - "is the callee a
function", "is this argument undefined", "does this key match" - and every one of
those is a question `type_of` and `strict_eq` answer correctly for a BigInt box
anyway. Only the USER-VISIBLE operators are routed. Routing all of them would have
worked too and would have been slower for no gain.

**`__raw` fires here, and js/ts are the first language where the answer DIFFERS
between the two engines.** `grep -n 'truthyExt *=' languages/js-to-llvm-ir.abnf` is
empty in the source sense the rule means - the default `js_truthy` is the floor's
own and needs no shim - **but the native build sets `core.truthyExt = "js_jstruthy"`
and that shim absolutely needs the suffix**, because compile-core's `truthy()`
compares its result with `icmp ne ..., 0` as a RAW integer and a MetaJS `false`
would arrive as handle 2. `languages/lib/js-rt.metajs` spells it
`function js_jstruthy__raw(v)`. So the rule of part 3's item 2 wants one more
clause:

> **A language needs `__raw` exactly when it OVERRIDES `core.truthyExt` - and the
> override may be introduced by the NATIVE BUILD ALONE.** Grep the grammar for the
> assignment, not for the read, and grep it inside the `-exe` arm too.

> **Four walls, one shape.** `this`, the accessor box, `typeof` and the whole BigInt
> operator family are all cases of *the Go runtime represents something as a Go
> TYPE, and layer 2 can only represent it as an object*. The general answer this
> migration adds to part 3 is: **route the OPERATOR through layer 2, in the emitter,
> under `-exe` only.** It needs no floor change and no Go twin, and because it is
> gated on `c.exePath` the module `llvm.Run` executes never moves - which is what
> makes it safe to do late in a session on two grammars the matrix depends on.

### WALL 3 - `js_scope_set_or_create`, and the ONE floor request this port files

JavaScript's non-strict implicit global: a plain `=` to a name that is nowhere on
the scope chain creates the binding in the ROOT scope. `abnf/jsrt.go` does it in
`scopeSetOrCreate`; **the C floor has no such function and layer 2 cannot write one**
- a scope is the floor's private storage.

Unlike walls 1 and 2 this one has no exact emitter lowering either, and the reason
is precise:

> The only probe the floor exposes is `js_scope_typeof`, which answers
> `"undefined"` for a slot that HOLDS undefined and for an ABSENT name alike -
> the inexactness swift recorded and ruby re-recorded. `var x; x = 5` and
> `x = 5` with no `x` anywhere are therefore indistinguishable, and they need
> opposite answers.

What IS in the floor is `js_pyset_var` - "bind where the chain already holds it,
otherwise in the scope the assignment stands in" - which is the same walk with a
different fallback. So under `-exe` the store is lowered to `js_pyset_var` and the
one difference left is WHERE a never-declared name lands: the enclosing scope
instead of the root. That is exact for every declared name, including the
`var x; x = 5` case the `js_scope_typeof` probe would have got wrong, and it is
wrong only for an implicit global created inside a function and read outside it -
which is `tests/js-test-implicit-global.js` and nothing else. Measured, and it is
the ONE native-vs-`llvm.Run` divergence left over all thirteen js and ts test files:

```
tests/js-test-implicit-global.js   native: js runtime error: variable not defined: counter (rc=1)
                                   llvm.Run: rc=0, no output
```

The file assigns `counter = 100` inside `makeGlobal()` and reads it from `main()`.
It is not in the gate (`tests/clang-check.sh` runs only `*-test-full.*`), and it is
not in the matrix as an `-exe` entry for that reason.

> **FLOOR REQUEST (reported, not made - the floor is another agent's this
> session).** Either of these closes it exactly, and both are small:
>
> * `long js_scope_has(long s, long name)` - `js_scope_typeof`'s loop answering a
>   BOOLEAN handle instead of `type_of(v)`. Six lines. The emitter then lowers
>   `set_or_create` to a two-arm select exactly as php lowered its scope probe.
> * or `js_scope_set_or_create` itself - `js_scope_set`'s loop with
>   `scope_put(G_ROOT, ...)` instead of `die2` at the end. Twelve lines, and it
>   removes the extern from layer 2's list entirely.
>
> The second is the better one: `js_scope_set_or_create` is not JavaScript's alone
> (it is `abnf/jsrt.go`'s generic name and python's `js_pyset_var` is its sibling,
> already in the floor next to where it would go).

### The three cross-cutting findings, checked and MEASURED for js and ts

1. **`core.truthyExt`: js and ts are the first language where the answer DIFFERS
   between the two engines.** At the base commit `grep -n 'truthyExt *='` was EMPTY
   in both grammars (the two hits either grep finds are READS in the `||=` arm), so
   `llvm.Run` keeps compile-core's default `js_truthy` - the C floor's own function,
   already the raw 1/0 the emitted `icmp ne ..., 0` compares, **no `__raw`**. The
   NATIVE build now sets `core.truthyExt = "js_jstruthy"` for BigInt (wall 4), and
   that shim **does** need the suffix. The tally over eight ported languages is
   **lua and php need it; swift, java, csharp, dart, go and ruby do not; js and ts
   need it in the native build only.**
2. **`_raw` PARAMETERS: js and ts have ZERO.** Every `handle(k)` in either grammar
   feeds `js_num_i`, `js_closure` or `js_arg`, all three floor externs. The count is
   now **3 / 0 / 0 / 1 / 1 / 0 / 3 / 1 / 0 / 0** for lua / php / swift / java /
   csharp / dart / ruby / go / js / ts.
3. **Generators (tag 15/16): the DRIVING half costs layer 2 zero lines, exactly as
   C# predicted, and the INSPECTION half is not owed either.**
   `grep -c 'js_genfn\|js_yield'` is **7** in each grammar - both languages really
   do build tag-16 cells - but the whole protocol is already emitted over floor
   externs: `js_genfn(js_closure(...))` to make the generator function, and
   `js_get(g, "next")` + `js_call` to step it, which `runtime.c`'s `get_member`
   answers with a bound `gen_next`. `tests/js-test-full.js` section on generators
   uses `.next()` and `.next(v)` and nothing else - no `.return()`, no `.throw()` -
   so the side table php needed for its `current`/`key`/`valid` surface is NOT
   owed here. What js DOES owe that C# did not is `js_iterable`, the for-of drain,
   which has to loop `next()` until `done` because the floor has no `g.drain`.
   And the tag-16 `typeof` fix that landed for C# is inherited: `typeof m ==
   "function"` is the correct callable test.

### Ground truth

Measured in the working tree. Other agents were landing kotlin and python in the
same tree during this sweep; the numbers below are the whole-suite ones at the end.

```
matrix                325/325
--full                5,618 assertions, 0 languages whose halves disagree
                      (js 354, typescript 276)
                      grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF'  ->  no hits
--cross               119 programs, 0 divergent, 0 differing only in warnings
go test ./abnf/       ok
clang-check           16 modules, ALL accepted, suite GREEN and - with kotlin and
                      python landing in the same sweep - ALL SIXTEEN languages now
                      read "ok, and the clang executable agrees", NONE held:
                        js          23097  ok, and the clang executable agrees
                        typescript  21790  ok, and the clang executable agrees
every gen-*-rt-ll.sh --check clean, js-rt.ll (60,764 lines) included
languages/lib/js-rt.metajs compiles byte-identically under -frozen
```

Native vs `llvm.Run`, **stdout AND stderr AND exit code**, over every js and ts
test file:

```
js-test-1  js-test-2  js-test-3  js-test-classes  js-test-complete
js-test-features  js-test-full                            all SAME (rc=0)
typescript-test-1  typescript-test-annotations  typescript-test-complete
typescript-test-features  typescript-test-try  typescript-test-full
                                                          all SAME (rc=0)
js-test-multifile (-i tests/imports)                      passes natively
js-test-implicit-global                     the ONE divergence: WALL 3, above
```

### The differential probes

Three, all generated rather than hand-written, all run in all three engines.

1. **BigInt, 3,318 output lines.** Every pair from
   `{0n, 1n, -1n, 2n, -2n, 10n, 255n, -255n, 2^64, -(2^64), 2^100,
   9007199254740993n, -9007199254740993n, 1000000007n}` over
   `+ - * / % < > <= >= == != === !== & | ^`, plus `<< >>` by
   `{0n,1n,3n,10n,64n}`, plus unary `-`, `~`, `!`, `typeof` and `.toString()` in
   radix 2/8/16/36. **Native output is byte-identical with node on all 3,318
   lines.**
2. **The method surface, 230 lines.** Every String method (23 of them over five
   receivers), every Array method (15 over four receivers), `toFixed`/`toString`/
   `valueOf` over ten numeric edge values including `-0`, `1e21`, `1e-7` and `NaN`,
   `Object.keys/values/entries`, `delete` + `in`, `reduce`, a generator drained by
   `for-of`, and a class with a getter, a `toString` and an `instanceof`.
   **Native == `llvm.Run` byte for byte. Against node, 2 of 230 lines differ**, both
   the fraction-digit count of a non-integral `toString(2)` (`(1e-7).toString(2)`
   and `(0.1).toString(2)`) - `abnf/jsrtjsprint.go`'s answer, faithfully reproduced,
   and divergence 5 below.
3. **A randomized BigInt/number probe, 2,638 lines** (agent-run, described in
   divergences 2 and 3 below): 220 random signed pairs of 1-39 decimal digits over
   every operator and four radices.

### The size of layer 2, and where it came from

`languages/lib/js-rt.metajs` is **3,643 lines** and compiles to a **60,764-line**
`languages/lib/js-rt.ll`, which is the second-largest layer 2 after ruby's - and
that number is misleading in the way that matters: it includes the **1,559-line
`lib/regex.js` engine, IMPORTED rather than restated**, so the regex extern family
cost 548 lines of glue rather than the ~2,700 lines `abnf/jsrtregex.go` +
`abnf/jsrtregexjs.go` would have been. The three parts are

```
part A  the VALUE surface (abnf/jsrtjsprint.go)   1,880 lines   24 externs
part B  BIGINT + arithmetic (abnf/jsrtjsbig.go)     957 lines   18 externs
part C  REGULAR EXPRESSIONS (jsrtregexjs.go)        548 lines    4 externs + import
base    this / newtarget / the routed operators     254 lines   12 externs
```

The layer-2 source compiles **byte-identically under `-frozen`** as well as under
goja, and `tests/gen-js-rt-ll.sh --check` is clean.

### Floor requests, both MEASURED rather than guessed - BOTH GRANTED 2026-08-03

**Both are in the floor now, and the `${obj}` job of divergence 1 below is done
with them. See "The two floor requests granted, and the template literal that
needed three halves" at the end of this section for the measurements.** The two
requests as they were filed:

1. **`js_del(o, key)` - the largest single win available.** The C floor exports no
   delete at all, so `js_jsdel` blanks the slot and records the key in a side table
   that `js_has` and the own-keys walk consult. `delete o.a`, `"a" in o` and
   `Object.keys` are correct; three things are not - key ORDER after
   delete-then-reinsert (`a|b|c`, node `a|c|b`), `for-in` after a delete, and object
   spread after a delete - and the table is **quadratic and leaks**: measured on
   `delete` inside a loop, n=1000 -> 1.35 s, n=2000 -> 7.9 s, n=4000 -> 36 s, where
   the identical loop without the delete is linear (n=60000 -> 5.1 s). About
   **fifteen lines in the floor would fix all four and delete ~60 lines of layer 2**.
2. **`js_scope_has(s, name)` or `js_scope_set_or_create` itself** - WALL 3 above.

### Divergences MEASURED and NOT fixed here

Every one of these is invisible to the suites - the ratchet passes 354/354 and
276/276 with all of them present - and every one is written down so the next reader
does not have to rediscover it.

1. ~~**`${obj}` renders `[object Object]` in BOTH Go-backed engines**~~ **FIXED
   2026-08-03 - the second candidate below is what shipped, in FOUR halves, and the
   extern it needed already existed on both sides (`js_jsstr`). See "The two floor
   requests granted, and the template literal that needed FOUR halves" at the end of
   this section.** As it stood: node runs
   the object's `toString`, and the INTERPRETER half is wrong for a class instance
   and right for an object literal. Measured three ways:

   ```
                       node    js-interpreter   js-to-llvm-ir (llvm.Run AND native)
   "" + new D()        D!      D!               D!
   `${new D()}`        D!      [object Object]  [object Object]
   `${ {toString} }`   O!      O!               [object Object]
   ```

   The compiler half is `makeTemplate` emitting the shared `js_add`. Fixing it is a
   THREE-HALVES job (emitter + `js-interpreter.abnf` + a ratchet assertion) and it
   would break byte-identity with `llvm.Run`, which is this gate - so the native
   build gets `js_jstpl`, which is the floor's `js_add` plus the BigInt rendering
   and nothing else. Two candidate fixes were measured on a scratch grammar:
   routing `makeTemplate` through `js_jsadd` gets `${obj}` right but makes
   `${boxedNumber}` render `5` where node says `[object Object]` (a template does
   ToString, not ToPrimitive); routing each interpolated PART through a ToString
   extern gets all three cases right in both engines. **The second is the fix; it is
   not this change.**
2. **A BigInt compared with a NUMBER goes through the double.** 220 random signed
   pairs of 1-39 digits over every operator and four radices, 2,638 output lines:
   **57 lines differ from node, all of the shape `bigint < hugeNumberLiteral` past
   2^53**, where node compares mathematically exact. `abnf/jsrtjsbig.go`'s
   `bigCompare` and `js-interpreter.abnf`'s `biRel` both go through the double, so
   all three halves are wrong together. Zero divergence on arithmetic, bitwise,
   shifts, `**` or any radix.
3. **One line of those 2,638 differs from the GO TWIN, and on it layer 2 matches
   node and Go does not**: `44018417530742584732345529662984672837n < 4401...837`.
   Go's `bigToFloat` is `big.Float -> Float64` (round-to-nearest); layer 2's
   `jbToNum`, like the interpreter's `biToNum`, accumulates `a*BASE + limb` and
   rounds at each step. **The compiler half and the interpreter half were already
   capable of disagreeing here before this port** - a latent matrix crack, recorded.
4. **A BigInt is INTERNED and therefore never freed.** `jbMake` keeps one canonical
   object per value in a 251-bucket table, and that is load-bearing rather than an
   optimisation: the floor's `strict_eq` has no BigInt case and falls through to
   identity, so without interning `10n === 10n` would be false at every site the
   emitter does NOT route (the `switch` dispatch chain, `makeNewFrom`, the floor's
   `indexOf`). The pool is a GC root and grows monotonically.
5. **`(-0.4).toFixed(0)`** is `"0"` in `abnf/jsrtjsprint.go` and `"-0"` in node;
   **`(0.1).toString(3)`** emits 52 fraction digits where node emits 34. Layer 2
   reproduces the Go answer in both - faithful to the port target, divergent from
   node.
6. **`js_jsobject` has no `prototype` slot.** Go copies it from the root scope's
   `Object` binding; the C floor seeds `String`, `Array` and `Math` but no `Object`
   and has no prototype concept, so the `Object.prototype.hasOwnProperty` idiom
   cannot resolve natively. Neither ratchet reaches it.
7. **`js_jsxcall` cannot read a bound method's NAME.** Go's version recognises
   `"...".replace` / `"...".split` off a `*boundMethod`; the floor's equivalent is a
   tag-9 cell that `typeof` calls `"function"` and no extern opens. Layer 2
   separates the two by RESULT TYPE with one probe call (`replace` always answers a
   string, `split` always an array). Correct for both; the one loss is another
   string native called with a regexp first argument (`"abc".slice(/x/)`), which
   would be read as `replace`. A floor extern answering a bound method's id would
   remove the probe.
8. **`f.call` / `f.apply` had to be taken over in `js_jsmget`.** The floor answers
   both with a bound method whose `js_call` DROPS the receiver, so nothing armed
   `js_this()`. Measured: the TypeScript decorator assertion `dec1`
   (`original.call(this, n)`) answered `NaN` natively against `10` under `llvm.Run`
   before the fix. **This is the fourth face of WALL 1 and the one that would have
   been easiest to miss**, because it is a divergence inside a library method rather
   than at a call site the emitter owns.
9. **`new RegExp("").source`** is `""` here and `"(?:)"` in node; the `d` flag builds
   no `m.indices`; `Object.keys(re)` exposes the real slots. All three are the Go
   twin's answers, inherited deliberately.

### What is still owed

1. `abnf/jsrtjsprint.go`, `abnf/jsrtjsbig.go` and `abnf/jsrtregexjs.go` can be
   deleted the day `llvm.Run` is retired, as the plan requires. They are kept now
   because `llvm.Run` is what the matrix runs.
2. ~~The two floor requests above.~~ **DONE 2026-08-03**, both of them.
3. ~~The `${obj}` three-halves job.~~ **DONE 2026-08-03**, and it was four halves.
4. Divergences 2-9, each recorded at its site.

### What generalises to PYTHON (which landed in parallel) and to KOTLIN

1. **Grep the language's own INTERPRETER GRAMMAR before porting anything.**
   `languages/python-interpreter.abnf` is written in the same MetaJS a layer-2 file
   is, it already `include`s `lib/regex.js` (~1088) and it already carries python's
   own `%`-formatting, its numeric tower and its builtin surface. Ruby's finding was
   "a layer-2 file may import a shared source"; the general form this port proves is
   **"the language's whole semantics already exist in MetaJS - adapt them to the
   handle value model instead of translating the Go twin."** That is the single
   biggest lever available and it is available to every remaining language.
2. **The private-representation probe has a THIRD answer now.** php/swift/dart/ruby
   lowered a SCOPE probe, go lowered a CONTROL SIGNAL probe, and both worked because
   the floor already had an extern that opened the cell. `js_this` had none - the
   floor keeps no call stack at all - so the answer became: **layer 2 keeps the
   state and the EMITTER hands it over**, under `-exe` only, so the module `llvm.Run`
   sees does not move. Python's `js_pyglobal` / `js_pynonlocal` / `js_pyfnscope` are
   the same shape (`runtime.c` says at `js_pyset_var` that "a scope here carries no
   mark, so those arms are NOT modelled. A Python layer 2 will have to add them"),
   and this is the pattern to reach for when the floor has nothing to lower onto.
3. **`js_scope_set_or_create` is python's problem too**, from the other side:
   python's assignment rule IS `js_pyset_var`, which is in the floor, so python gets
   for free the thing js had to approximate. But python's `global`/`nonlocal` need
   the scope MARK the floor does not keep, which is the same wall by a different
   name - and the same two answers apply (a floor probe, or emitter-carried state).
4. **The `-exe`-only emitter change is the safety property that made all of this
   possible.** Every lowering here is gated on `c.exePath != ""` and the emitted
   module for the non-native path was diffed against a clean archive of the base
   commit and found byte-identical for both `tests/js-test-full.js` and
   `tests/typescript-test-full.ts`. **Do that diff first, before touching a
   grammar** - it converts "will the matrix survive this" from a hope into a
   measurement, and it costs one `git archive` plus one `diff`.
5. **Generators: js and ts are the third and fourth languages on the landed
   coroutine primitive and they owe the DRIVING half zero lines**, like C#. Python
   is the one language left that will owe the INSPECTION half, because `send()`,
   `close()` and `throw()` are part of its protocol - php's `phGenCall` (an
   identity-keyed side table, because a generator is a floor CELL and a cursor
   cannot be stored ON it) is the model, and php's own report warns that its
   `send()` is covered by neither engine's tests and should be re-derived rather
   than trusted.
6. **A duplicate symbol is still reported as an unresolved one** (php's fifth trap).
   This port hit the OTHER half of the same class: two top-level `function js_*`
   with the same name in one layer-2 source is `error: redefinition of global
   '@jsrtlib_f_js_<name>'`, which at least says what it means - but it is worth
   knowing before splitting a layer-2 file across contributors.

### launch.json entries wanted (not added - not my file)

All four pass natively today, at the working tree:

```
"JS native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/js-to-llvm-ir.abnf", "tests/js-test-features.js", "-q", "-exe", "tests/js-native.out"]
"JS native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/js-to-llvm-ir.abnf", "tests/js-test-multifile.js", "-q", "-exe", "tests/js-native-multi.out"]
"TypeScript native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/typescript-to-llvm-ir.abnf", "tests/typescript-test-features.ts", "-q", "-exe", "tests/ts-native.out"]
"TypeScript native binary, complete (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/typescript-to-llvm-ir.abnf", "tests/typescript-test-complete.ts", "-q", "-exe", "tests/ts-native-complete.out"]
```

A full-syntax entry would be green too - both ratchets pass natively - but it stays
out of the matrix for the same reason every other `*-test-full.*` does: the ratchet
lives in `./test.sh --full`, and `tests/clang-check.sh` already runs the native one
on every sweep.

### The two floor requests granted, and the template literal that needed FOUR halves (2026-08-03)

Three items, all measured against `node` v24 and against a clean `git archive` of
`ff74465` built and run in its own tree.

**1. `js_del` is in the floor.** 21 lines of `runtime.c` next to `js_keys`
(`obj_find`, shift the keys/vals pair down, blank the last slot, `sc(o, n-1)`),
plus one host-dispatch arm and one `seed_root`. It is reachable from MetaJS as the
host builtin **`delKey`, host id 63** - the same argument `keysOf` (id 61) made one
operation earlier and the one to reach for again: *an extern is only callable from
an emitter, and MetaJS has no `delete` operator at all*, so without a builtin a
layer-2 file cannot remove a key, only blank it. Bound in all three engines, which
is the gap `f19a8ad` left for `sint` and `8c396c4` had to repair:

```
languages/lib/runtime.c        host id 63 + seed_root("delKey")   native
abnf/jsrtint.go                keysBindings, next to keysOf       llvm.Run + the frozen host
languages/metajs-interpreter.abnf  delKeyHost in hostGlobals      the interpreter
```

The interpreter half cannot use `delete` either (there is no such operator in the
MetaJS its `:script` is written in), so it does observably what the other two do:
drops the key from the hidden `__ik` insertion-order list `keysOf` reads and writes
the slot back to `undefined`. A later store re-notes the key, so it lands at the
END of the order in all three.

Layer 2 lost the side table and everything that consulted it - **-87 lines of
`js-rt.metajs`, -1,071 lines of `js-rt.ll`**, against +70 lines of C. The estimate
filed above was "about fifteen lines in the floor would fix all four and delete ~60
lines of layer 2"; the floor cost 35 lines of code and 35 of comment, and layer 2
lost 87.

What it bought, `tests/js-test-full.js` SECTION 36 run natively at `ff74465` and at
the working tree:

```
                                   ff74465 native      now (all three engines)   node
Object.keys after del+reinsert     a,b,c               a,c,b                     a,c,b
for-in after a delete              a,b,c               a,c,b                     a,c,b
{...o} after a delete              x:undefined kept    key gone                  key gone
Object.assign after a delete       b in the old slot   b at the end              b at the end
```

and the cost, `delete` inside a loop, native binary, `/usr/bin/time -l`:

```
n         ff74465                     now
1000      1.24 s   RSS 4,456,448      0.34 s   RSS 3,571,712
2000      7.27 s   RSS 5,259,264      0.32 s   RSS 3,588,096
4000     39.19 s   RSS 7,012,352      0.40 s   RSS 3,588,096
60000        -                        1.04 s   RSS 3,588,096
```

Quadratic and leaking, to linear and flat. (A first measurement on the working tree
before the change read 61 s at n=4000; 39.19 s is the clean-archive number and is
the honest one.)

**2. `js_scope_set_or_create` is in the floor**, 11 lines next to `js_pyset_var` -
the same walk with `scope_put(G_ROOT, ...)` instead of `die2`. The lowering WALL 3
described (`storeExt` rewriting the name to `js_pyset_var` under `-exe`) is gone
from both emitters, and layer 2's `fail()` stub is gone with it - it had to go,
because a top-level `function js_*` in a layer-2 file is an exported C symbol and
would now be a redefinition. **This was worth doing and the alternative was not:**
the lowering was exact for every declared name, but `tests/js-test-implicit-global.js`
aborted natively with `variable not defined: counter` (rc=1) and passes now, and
`php-to-llvm-ir.abnf` emits the same extern for `define()` - which had no native
implementation at all until this.

**3. The `${obj}` divergence, which is FOUR halves and not three.** Divergence 1
above recorded it as `makeTemplate` emitting the floor's `js_add`; the fix it named
("route each interpolated PART through a ToString extern") is what shipped, and the
extern already existed on both sides: **`js_jsstr`**, whose Go twin is
`rt.jsvString` and whose layer-2 twin is `jvStr`. Both compiler grammars now emit
`js_jsstr` per PART and join the strings with the floor's plain `js_add`;
`js_jstpl` is deleted, from the emitters and from layer 2, because once every part
is already a string there is nothing BigInt-specific left to do.

The interpreter halves were wrong too, differently and in the same place - they
concatenated with the HOST's `+`, which is ToPrimitive with hint DEFAULT - so
`js-interpreter.abnf` and `typescript-interpreter.abnf` now call `jsStr`, which is
the same function `js_jsstr` is. Four halves, one line each:

```
                                node   ff74465 interp   ff74465 llvm.Run/native   now, all three
`${ {toString} }`               T      T                [object Object]           T
`${ new D() }`   (class)        D!     [object Object]  [object Object]           D!
`${ {valueOf: ()=>7} }`         [object Object]  7      [object Object]           [object Object]
`${ {valueOf, toString} }`      B      7                [object Object]           B
`${10n}`, `${2n**100n}`         digits digits           digits                    digits
`${1} ${"s"} ${[1,2]} ${({})}`  ok     ok               ok                        ok
```

The peer report that claimed the BigInt template fix would cover the user-`toString`
case as well was WRONG, and it was worth measuring rather than inheriting: routing
`makeTemplate` through `js_jsadd` fixes `${obj}` and breaks a boxed number, because
a template does ToString and not ToPrimitive-with-hint-default. `${ {valueOf} }`
in the table above is exactly that trap, and it is the assertion that catches it.

**Ratchet, and its discriminating power measured rather than asserted.** Three new
sections, run through the NEW test files against the clean `ff74465` archive:

```
tests/js-test-full.js SECTION 36   (354 -> 378 assertions)
  interpreter half   3 FAIL   tpl2 tpl3 tpl4
  llvm.Run           4 FAIL   tpl1 tpl2 tpl4 tpl8
  native             9 FAIL   del3 del4 del5 del6 del8 tpl1 tpl2 tpl4 tpl8,
                              then ABORTS rc=1, `variable not defined: implicitGlobal36`
tests/typescript-test-full.ts SECTION 33   (276 -> 300)   identical, id for id
tests/metajs-test-full.js SECTION 28   (501 -> 521)
  all three engines  does not run at all: `variable not defined: delKey`
```

**Verification at the working tree.** matrix **325/325** · `--full` **5,686
assertions, 0 halves disagree**, and the grep for `BUT -frozen|VACUOUS|MISMATCH|
FROZEN-DIFF` is empty · `--cross` **119/0** · `clang-check` **16/16, all sixteen
`ok, and the clang executable agrees`** · `go test ./abnf/` ok · every
`gen-*-rt-ll.sh --check` up to date · `-freeze` a fixed point · every js and ts
test file **byte-identical between `llvm.Run` and the native binary**, stdout,
stderr and exit code (the two `*-test-recognize.*` are compiler errors in both
engines, as before) · `bench-alloc.sh` **3,734 / 3,721 / 3,715 / 3,711 B/iter** at
50k/100k/200k/400k with `MEC_GC=off`, RSS flat at 3.3 MB with the collector, both
unchanged · `coro-poc/build.sh --gc` all four modes SAME, `--break` the same rows
DIFFER and the same rows do not, verified against the archive rather than against
the note.

## KOTLIN - the eighth, the largest, and the row was RELEASED (2026-08-03)

**`tests/kotlin-test-full.kt` is 979/979 under `llvm.Run` AND in the native
binary**, byte-identical on stdout, stderr and exit code. `tests/clang-check.sh`
reads

```
kotlin         179934  ok, and the clang executable agrees
```

The row WAS held at first, exactly as the text below this heading was written -
the native binary linked and ran but did not pass all 979. Two defects closed the
gap, both in `languages/lib/kotlin-rt.metajs` and neither in the emitter:

- **`ktNum` ended in a blanket `return 0`.** Go's `rt.toNumber` maps `undefined`
  to NaN and `null` to 0, everything else defaulting to NaN. The visible symptom
  was `size + Unit` in `js_ktadd`'s tail. `ktNum` is used throughout the file, so
  this is the widest-blast-radius edit in the kotlin port - it is backed by
  979/979 plus both probes, and it is the first thing to suspect if kotlin ever
  drifts.
- **`ktRecvMember`'s accessor arm read `recv[name]` directly**, on a comment's
  claim that the floor's member read resolves an accessor. It does not - the floor
  hands back the `{__acc, get, set}` record. The arm was also dead code: the
  `k4Own` arm above it already answers every own property. Replaced with a port of
  `rt.findClassAccessor` (`jsrtkotlin.go:4845` -> `jsrt.go:2534`), walking the
  `__class`/`__super` chain and calling the getter so `js_this()` binds the
  receiver.

A third finding was a **misdiagnosis corrected by measurement**, and is the more
useful record: the divergence was first blamed on `ktRecvMember` resolving `size`
versus `sum()` differently between the halves. A three-line repro showed both
halves agreed all along - `sum` was the file's own top-level `fun sum(n: Node)`,
shadowing the receiver in both. Always build the repro before believing the
diagnosis.

**No suite was red at any point, no assertion was moved, and
`tests/kotlin-test-full.kt` was not touched.** Deleting the one
`clang-check: native-row-held` comment line was the whole of releasing the row,
which is what that mechanism is for. Committed as `8ded328`.

Two kotlin tests still cannot be `-exe` built at all: `kotlin-test-recognize.kt`
("assignment to a computed target not implemented") and `kotlin-test-widen2.kt`
("index with more than one argument not implemented"). Both are compile-time
emitter limits that fail identically under `llvm.Run` and predate this work.

**There is no `kotlinc` and no `kotlin` on this machine** - `command -v kotlinc
kotlin` is empty, and `/opt/homebrew/bin/kotlin*`, `/usr/local/bin/kotlin*` and
`/opt/homebrew/opt/*kotlin*` do not exist. So, exactly as C# and Dart recorded of
themselves, **`abnf/jsrtkotlin.go` is the specification and nothing here was
confirmed by a real toolchain.** `clang` is Apple clang 17.0.0.
`abnf/jsrtkotlin.go`, `abnf/jsrtregexkt.go` and `abnf/jsrtint.go` are all **kept**.

### The extern split, and what the SIZE actually cost

`tests/kotlin-test-full.kt` drives **96** externs, and so does the union over
every `tests/kotlin-test-*.kt` - kotlin's emitter has no test-specific arm. It
was 95 before the scope probes were lowered; the lowering added `js_scope_typeof`
and removed nothing, for the reason Dart's correction gives.

| where | count | what |
|---|---|---|
| already in the C floor | 36 | the generic primitives of phase 3, the five the Lua pilot added, `js_truthy`, `js_jflo`, the `js_ctl_*` family, and `js_scope_typeof` |
| **layer 2**, language-NEUTRAL | 12 | `js_char`, `js_char_code`, `js_char_like`, `js_defprop`, `js_is_type`, `js_jadd`, `js_jband`, `js_kindex`, `js_kset`, `js_pyrest`, `js_supercall`, `js_this` |
| **layer 2**, Kotlin's own | 48 | 46 `js_kt*` plus `js_rxktnew` / `js_rxktglobal` |

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
java           29        9      16      ~90 lines, the unsigned-box long
csharp         35       15      18      NONE  <- sintRaw
dart           37       13      17      NONE
go             42       10      32      NONE
kotlin         36       12      48      NONE  <- sintRaw, for a NEW reason
```

**What the size cost, since that is the number the next big grammar wants.**
`languages/lib/kotlin-rt.metajs` is **~10,100 lines** from a 5,934-line Go twin
plus a 623-line regex twin - roughly 1.5 lines of MetaJS per line of Go, which is
the same ratio dart (1,652 from 896) and go (1,823 from 630) paid, so the ratio
did NOT degrade with size. What DID change is that the work stopped fitting in
one head: it was written as **seven independent sections against a written
contract** (a sentinel `KT_MISS` for "this dispatcher did not handle it", one
private name prefix per section, and a single owner for every shared helper) and
concatenated into one flat MetaJS scope. Exactly **one** name collided across
9,900 lines. **That contract, not the porting, is the transferable part.**

The regular-expression engine is the one place size was free:
`languages/lib/regex.js` is already written in this MetaJS subset, so its 1,559
lines went in **verbatim** and only the 537-line Kotlin dispatcher had to be
written. Ruby reaches the same text with `import "./regex.js"`; either works.

### THE WALL, and it is the one every earlier language avoided

> **Kotlin has TEN scope-taking externs. Every earlier language had at most two.**

php lowered 2, swift 2, java added 1 helper, csharp 0, dart 2, go 1. Kotlin's
`js_ktget`, `js_kset`, `js_ktvarset`, `js_ktmextget`, `js_ktmextset`,
`js_ktmextcall`, `js_ktouter`, `js_ktrefbase`, `js_ktdthis` and `js_ktbareref`
all take a SCOPE HANDLE, and `core.scopeGetExt` was `js_ktget`, so **every
variable read in a Kotlin program went through one.**

Nine of the ten are now lowered into IR in `languages/kotlin-to-llvm-ir.abnf`
(`rtKtGet`, `rtKSet`, `rtVarSet`, `rtMextCall`/`rtMextGet`/`rtMextSet` over
`rtMextFind`/`rtMextWalk`/`rtClsFind`, `rtOuter`/`rtIsClass`, `rtBareRef`,
`rtRefBase`, `rtDThis`), over `js_scope_typeof` / `js_scope_get` /
`js_scope_set`, which an EMITTER may call. `sw_safeget` transferred verbatim for
the fourth time. `llvm.Run` stayed at 979/979 through every one of them.

**But the lowering is only PARTIAL, and the reason is a real gap in the floor:**

> An emitter can read the INNERMOST binding of a name, the INNERMOST `this` and
> the INNERMOST `__dispatch`. It cannot walk PAST them, because
> `languages/lib/runtime.c` seeds no scope-PARENT accessor - the whole
> `seed_root` list is println/print/eprintln/parseInt/parseFloat/exit/byteLen/
> sprintf/printf/sprint/fail/`sint*`/`flo*`/keysOf/sintRaw/Infinity/NaN/anytype/
> Math/String/Array. jsrtkotlin.go's `js_ktget` walks the WHOLE chain for a
> `this` and again for a `__dispatch`, and `ktMemberExtFind` collects both.

Each helper therefore ends in the extern it replaced, which is exact under
`llvm.Run` and is the one call layer 2 cannot answer natively.

**The second half of the wall is Kotlin's IMPLICIT-RECEIVER STACK**, and it is a
NEW shape of go's "private representation" rule read from the other side:

> `with` / `run` / `apply` / `buildString` / `buildList` / `buildMap` and every
> `T.() -> R` lambda push their receiver inside the RUNTIME (`ktWithRecv` ->
> `ktRecvStack`), and `js_ktget` consults that stack after its scope walk. The
> stack is layer 2's own storage, so this time it is the EMITTER that cannot see
> it. Go's rule - "find the floor extern that already answers it and lower the
> call" - has no answer, because there is no such extern.

The way through is worth copying, because it needs no floor change and no new
extern: **`js_ktglobal(name)` is the one extern of the right shape - one NAME, no
scope** - so the emitted `kt_ktget` asks it before falling back, and
`kotlin-rt.metajs`'s `js_ktglobal` answers from `ktRecvStack` FIRST and only then
from the stdlib table. In the Go twin `ktGlobalFn` answers `jsUndef` for a name it
does not know, so **the probe is a no-op under `llvm.Run`** and the ratchet stayed
at 979. The same trick carries the WRITE side (`js_kset` / `js_ktvarset` walk the
stack for a receiver that owns the property) and the member-extension lookup
(`js_ktmext*` walk it with `k4bMextWalk`).

**THE FLOOR REQUEST, reported and not made** (the floor is another agent's this
session), and it is the same one go's section already predicted:

> **`scopeParent(scope)` and `scopeGet(scope, name)` as host builtins.** With
> those two, `js_ktget` and all nine of its relatives go back into LAYER 2
> whole - no emitter lowering, no `js_ktglobal` probe, no residue - and go's
> scope-backed pointer renders as well. Cost of not having them, measured: nine
> IR helpers (~250 lines of emitter), plus the residue that keeps the native row
> held.

### `si_norm`: KOTLIN IS A THIRD ANSWER, and this is the finding to carry forward

Dart's section poses the question as "is the language's unboxed representation
the one `si_norm` produces?", with Dart yes and Java no. **Kotlin is neither, and
the plan's own instruction to take C#'s `sintRaw` route was right for the wrong
reason.** `abnf/jsrtkotlin.go:103` says it in code and in its own comment:

```
// ktNorm ... It is deliberately not giNorm, which unboxes at 64 bits because a
// plain number means int64 for Go.
if w == 32 && !u { return float64(int32(x)) }
return jsGInt{v: x, w: w, u: u}
```

So **Kotlin's normalisation boundary is 32 bits**: an `Int` unboxes and a `Long`
is a box AT EVERY MAGNITUDE. Its Long is honestly SIGNED, so no `jl*` layer is
owed and plain `sintOp` / `sintCmp` / `sintStr` / `sintNum` are exact on it - but
`sintConv` ends in `si_norm`, which unboxes a signed 64 bit value a double holds
exactly, and `ktWU` reads the result type off the operand BOXES. An unboxed small
Long would make `1000000L * 1000000L` a 32 bit multiply and answer `-727379968`,
which is **Java's wall met from the other side**. `ktNorm` therefore ends in
`sintRaw`, C#'s `csNorm` recipe unchanged.

> **The question to ask the next language is not "does it have a 64 bit signed
> type" and not even "is its unboxed form `giNorm`'s" - it is "WHERE is its
> unboxing boundary". Go and Dart say 64, Kotlin says 32, Java says nowhere.
> `sintRaw` answers all three of the non-Go cases**, and it is now needed by a
> language whose integers were never dishonest about their signedness.

That also settles the hazard **the floor audit named specifically for Kotlin** -
"kotlin calls `js_jflo` directly with no `js_jvnum` equivalent and re-inherits the
already-diagnosed bug". Go's correction is the right reading: the hazard needs a
language that carries a SIGNED value in an UNSIGNED-MARKED cell. **Kotlin never
marks a cell unsigned unless it really is a `UInt`/`ULong`**, so `to_number` of a
tag 13 cell is the signed reading and `js_jflo` needs no wrapper. Checked at the
code, not assumed.

### The cross-cutting findings, checked and MEASURED for kotlin

1. **`core.truthyExt`.** `grep -n truthyExt languages/kotlin-to-llvm-ir.abnf` is
   EMPTY, so kotlin keeps compile-core's default `js_truthy`, the C floor's own,
   which already answers the raw 1/0 the emitted `icmp ne ..., 0` compares. **No
   `__raw` export.** The tally over seven ported languages: **lua and php need it;
   swift, java, csharp, dart, go and kotlin do not.**
2. **`_raw` PARAMETERS: kotlin has exactly one**, `js_char`'s code point, written
   with a bare `handle(k)`. The count is now **3 / 0 / 0 / 1 / 1 / 0 / 1 / 1** for
   lua / php / swift / java / csharp / dart / go / kotlin.
3. **Coroutines: none.** `grep -c 'js_genfn\|js_yield'
   languages/kotlin-to-llvm-ir.abnf` is **0** - `suspend` is lowered by the
   emitter into ordinary calls and `launch`/`runBlocking` run their block eagerly
   (`ktCoroRun`) - so WALL 1 does not touch this language and **the tag-16
   `typeof` divergence cannot bite it either**. Run that grep before costing a
   second stack: it is the fourth language in a row for which the answer is zero.
4. **`jsChar` IS needed here, unlike swift / java / csharp / dart / go.** Kotlin
   has a real `Char`: it is compared, put in a variable, asked its `code`, and
   `js_typeof`'d. The floor still has no char tag, so layer 2 uses the
   `{__char: code}` object box `languages/kotlin-interpreter.abnf` already uses.
   Java's audit rule applies - the box is safe exactly when no FLOOR extern
   receives one - and Kotlin's emitter routes every char operation through
   `js_kteq`/`js_ktne`/`js_ktcmp`/`js_ktarith`/`js_char_code`, all of which are
   layer 2's. The one thing the box cannot reproduce is the Go twin's own
   `js_typeof` answer `"char"`, and `ktBoxTypeName` is what covers it.

### The two engines, and what is NOT yet identical

`llvm.Run` (the Go twin) is at **979/979** and every other kotlin program in
`tests/` is unchanged - `kotlin-test-1`, `-complete`, `-features` (187),
`-object-expr`, `-annotations`, the three BOM files and `-try` all behave exactly
as they did at `3f91d7b`. The native binary runs the whole file and still
disagrees on a residue; the failures are ordinary layer-2 semantics (each one a
function of `abnf/jsrtkotlin.go` that has not been matched arm for arm yet) plus
the scope residue above, NOT a structural problem: the module links, the two
dispatch tables meet, the collector runs, and the regex engine, the sized-integer
arithmetic, the class model, the delegates and the callable references are all
exercised natively before the first disagreement.

### launch.json entries wanted (not added - not my file)

Not yet. A kotlin `-exe` entry belongs in the matrix only once the row is
released; until then `tests/clang-check.sh` is the right place, and it already
builds the native binary on every sweep.

### What is owed

1. **The residue of the ratchet**, native only. `llvm.Run` is unaffected.
2. **`scopeParent` / `scopeGet` host builtins** - the floor request above. They
   are what turns nine emitter helpers back into ordinary layer-2 functions, for
   every language and not only Kotlin.
3. `abnf/jsrtkotlin.go`, `abnf/jsrtregexkt.go` and `abnf/jsrtint.go` stay.

## PYTHON - DONE (2026-08-03). The gate is MET: 261/261 natively, byte-identical

**`tests/python-test-full.py` runs as a self-contained native binary, all 261
assertions pass in it, and its stdout, its stderr and its exit code are
byte-identical with `llvm.Run`.** `tests/clang-check.sh` reads

```
python          20676  ok, and the clang executable agrees
```

so python left the `ok (module only)` row for the run-it row, **no row is held, no
assertion is held back, and no floor change was needed or made.** That is the
ninth language and the last one this plan was written for.

Python is the second of the languages with **no dedicated Go twin** - its semantics
live in the shared `abnf/jsrt.go` (the `js_py*` extern table at 6900-8100 and the
`py*` helpers at 8580-9840: `pyString` / `pyRepr` / `pyTypeName` / `pyEqual` /
`pyMRO` / `pyLinearize` / `pyLookup` / `pyGetAttr` / `pySetAttr` / `pyMethodCall` /
`pyBindCall` / `pyArith` / `pyFormat` / `pySliceIndices` / `memberCall`) plus
`abnf/jsrtregexpy.go` for the `re` module. Ruby's advice - *budget for the reading,
not for the writing* - held exactly.

### THE HEADLINE: SIX externs were lowered in the emitter, not written in layer 2

Every previous language lowered one or two private-representation probes. Python
lowered **six**, and they were not optional: `runtime.c` says so itself, in a
comment at `js_pyset_var`.

> *"The `global`/`nonlocal` and binding-boundary arms of the Go twin need
> js_pyfnscope/js_pyglobal, which only the Python compilers emit; a scope here
> carries no such mark, so those arms are unreachable and are NOT modelled.
> **A Python layer 2 will have to add them.**"*

It cannot. `abnf/jsrt.go` carries Python's binding rule **on the scope object
itself**: `jsScope.pyFn` (this scope is a def / lambda / class body / module top
level, so an assignment binds locally) and `jsScope.pyDecl` (the names a `global`
or `nonlocal` statement sent elsewhere). A scope is the C floor's private storage
and a MetaJS runtime library never receives one it can open - the wall php, swift,
java, dart, go and ruby each met. The answer was the same for the seventh time and
it scaled: find the floor externs that DO answer the question and lower the call.

| extern | what it did | how it is emitted now |
|---|---|---|
| `js_pyfnscope` | mark a binding boundary | two `js_scope_decl`s of a MARK (below) |
| `js_pyset_var` | Python's whole assignment rule | the IR helper `py_setvar`, four arms |
| `js_pyglobal` | `global x` | the IR helper `py_global` |
| `js_pynonlocal` | `nonlocal x` | one `js_scope_decl` of a mark |
| `js_pydel_var` | `del x` | the IR helper `py_delvar` + a sentinel |
| `js_pyannot` | `x: T` into `__annotations__` | `pyAnnot`, over `js_dict_new`/`js_get`/`js_arr_push` |
| `js_pyclass_fill` | copy a class body's scope onto the class | `py_clsfill1`, one call per statically-known name |

**The mechanism is worth copying and is new.** The marks are carried as ORDINARY
BINDINGS, in a scope of their own inserted above each boundary:

```
    parent  ->  MID  ->  the boundary scope S
```

`MID` holds `"pyfn*" -> S`, `"pyup*" -> MID`, and at the module top level
`"pymod*" -> S` and `"pydel*" -> a unique array`. None of those is a possible
Python identifier. `js_scope_get` already walks the chain, so from ANY nested scope
`"pyfn*"` resolves to the nearest enclosing boundary and `"pyup*"` to a scope that
holds no user name of that function - which is exactly what makes *"is this name
bound at or below the boundary"* answerable with two `js_scope_typeof` calls and no
floor change. Putting the marks in `MID` rather than in `S` is what keeps `S`
clean, and that matters because a class body's bindings become the class's members.

`global x` / `nonlocal x` declare the marker `"g*x"` / `"n*x"` in the scope the
statement stands in - which is exactly where `abnf/jsrt.go` puts its `pyDecl`
entry - and `py_setvar` consults them with the same chain walk the Go arm does.

The union went **91 -> 86** and what layer 2 has to WRITE fell by six.

> **THE ONE INEXACTNESS, and the one it was NOT.** `py_setvar` decides "bound at or
> below the boundary" as *"it resolves from here AND it does not resolve from
> MID"*. A name bound BOTH strictly below the boundary and above it would be
> re-bound at the boundary where `abnf/jsrt.go` assigns the inner holder.
>
> The naive version of that test was **measured wrong immediately** and the fix is
> the transferable part: `declCompNames` declares a comprehension target holding
> **undefined**, and `js_scope_typeof` cannot tell a slot that HOLDS undefined from
> an absent name - Swift's and Ruby's recorded inexactness, arriving as a hard
> failure rather than a footnote (`[i*i for i in range(4)]` answered
> `[NaN, NaN, NaN, NaN]`). The answer is that **the emitter already knows those
> names statically**: `pyInnerNames` records every name the emitter DECLARES in a
> non-boundary scope of the current function - a comprehension target, an
> `except ... as e`, a `with ... as w` - and for those the floor's own
> `js_pyset_var` walk is emitted straight, because it is exactly right. Reset at
> every real function boundary, inherited across a ctl closure (which is the same
> function). With that, the residue is a name bound strictly below the boundary
> AND above it, which nothing in `tests/` does.

**`del` needed a sentinel AND a guarded read, and that is a second reusable
shape.** `abnf/jsrt.go` leaves a private Go value (`pyDeletedT`) in the slot and
raises `UnboundLocalError` from inside `scopeGet`; the floor's `scope_get` knows
nothing about it. So `"pydel*"` is one unique array created at module start, and
EVERY variable read goes through the IR helper `py_getvar`, which compares against
it with `js_seq` and raises. Both `makeVarRef` (overridden in the grammar, since
`core.scopeGetExt` names an EXTERN and `py_getvar` is an emitted function - swift's
and dart's move) and `emitRefRead` (an augmented assignment READS first) had to be
routed through it; routing only the first left `stm5`/`stm6` red, which is how the
second site was found.

The raised value is built with `js_pycall` on the module's own
`UnboundLocalError` class, NOT with `js_pyexc`: `js_pyexc`'s shortcut shape carries
a fresh class with no `__super`, so `except Exception` would not catch it.

### The extern split

`tests/python-test-full.py` drives **86** externs, and so does the union over every
`tests/python-test-*.py`.

| where | count | what |
|---|---|---|
| already in the C floor | 32 | the generic primitives of phase 3, the scope externs, `js_pyset_var` (which the floor already had), `js_genfn`/`js_yield`, `js_truthy` |
| **layer 2**, language-NEUTRAL | 6 | `js_dict_new`, `js_jadd`, `js_pyeq`, `js_pyget`, `js_pylen`, `js_pyset` |
| **layer 2**, Python's own | 48 | everything else `js_py*`, plus `js_bigint` and the three `js_pyre*`/`js_pyrx*` regex wrappers |

`languages/lib/python-rt.metajs` is 4,979 lines (PLUS the 1,559-line imported
`regex.js`) and compiles to `languages/lib/python-rt.ll` (70,847 lines)
**byte-identically under `-frozen` as well as under goja**. It exports the 54 symbols the union needs plus
`js_pymcall`, `js_pyexc` and `js_is_type`, which the grammar can reach on other
inputs.

The split against the eight done before it:

```
             floor   neutral   own      pair/box layer needed in layer 2
lua            23        5      26      ~200 lines of i64 pairs
php            29        8      63      a {__pf} float box
swift          35       10      21      NONE
java           29        9      16      ~90 lines, the unsigned-box long
csharp         35       15      18      NONE  <- sintRaw
dart           37       13      17      NONE
go             42       10      32      NONE
ruby           31        8      54      NONE - but a whole numeric TOWER
python         32        6      48      NONE - but an ARBITRARY PRECISION int
```

### The `si_norm` question, and the one genuinely new answer

Dart asked *"is the language's unboxed representation the one `si_norm`
produces"*. For Python, as for Ruby, the question has no meaning in the ordinary
direction: **int and float are the SAME plain double**, discriminated by
`Math.trunc` (`abnf/jsrt.go:9332` decides `pyTypeName` that way and nothing else
does), so no tag-13 or tag-14 cell is built for an ordinary number.

What Python has that no previous language had is an **arbitrary precision int**.
`js_bigint` is backed by Go's `math/big`, and the floor has no equivalent - a
sized-integer tag is 64 bits and `10000000000000000000000000001` is not. The answer
is a base-10000 little-endian bignum in MetaJS, `{__pybig, neg, limbs}`, with
schoolbook add / sub / mul and per-limb long division; `pyABigFromNum` is exact for
any integral double (halve to a 53-bit mantissa, double the bignum back), and
`pyABigEqFloat` is Go's `bigEqFloat`. Cross-checked against native `BigInt` over
3,000 random +-30-digit pairs times `+ - * / % cmp`.

Two things about it are worth carrying:

* **Go reaches `bigArith` AFTER the array/object arms of `jsAdd`,** because a
  `*jsBigInt` is never a `*jsObject`. Here a `__pybig` IS a plain object, so the
  bigint arm had to move AHEAD of them or `10**28 + 1` would string-concatenate.
  Every port of a Go type switch into this value model has that hazard.
* **`rt.toNumber` has NO `*jsBigInt` arm**, so `big|1`, `big//2`, `big%2`,
  `big**2` and `float(big)` are all NaN or 0 in the Go twin. They must stay so
  here, which is why `pyNum(v)` is written `v * 1` - the floor's `js_mul` coerces
  exactly the way `toNumber` does, including NaN for every object.

**Tag 13 IS reached, in exactly one place**, and it is Ruby's find reused:
`pyBBaseStr` converts to a base with `sintRaw`/`sintOp`/`sintNum` on the UNSIGNED
reading of the magnitude, because Go's `strconv.FormatInt(int64(x), base)`
SATURATES on arm64.

### The four cross-cutting findings, checked and MEASURED for python

1. **`core.truthyExt`: python is the THIRD language that needs `__raw`**, after
   lua and php - and the tally over nine is now **lua, php and python need it;
   swift, java, csharp, dart, go and ruby do not**.
   `grep -n truthyExt languages/python-to-llvm-ir.abnf` answers
   `core.truthyExt = "js_pytruthy"` (Python truthiness: an empty container is
   false, so the floor's own `js_truthy` cannot serve). compile-core's `truthy()`
   emits `icmp ne <call>, 0` on it, and `languages/python-to-llvm-ir.abnf:1536`
   emits a SECOND `NewICmp` on the same extern's result for `not x`. Both are
   covered by writing the function `js_pytruthy__raw`. A shim handing back the
   boolean HANDLE would have given 2 for `false`, and `2 != 0` - **every condition
   in the program taken, silently.**
2. **`_raw` PARAMETERS: python has ONE**, `js_pyrest(args, i_raw)`, whose start
   index the emitter writes with a bare `handle(k)` (and which `jsrt.go:679` marks
   in its own raw-argument mask). The count over nine is
   **3 / 0 / 0 / 1 / 1 / 0 / 1 / 3 / 1** for lua / php / swift / java / csharp /
   dart / go / ruby / python.
3. **Tag 16 / generators: PRESENT, and python is the first language that needed
   layer-2 lines for them.** `grep -c 'js_genfn\|js_yield'
   languages/python-to-llvm-ir.abnf` is **6**. C#'s finding held for the DRIVING
   side - the emitter already lowers `js_genfn(js_closure(...))` and `js_yield`
   over floor externs, and layer 2 owes that zero lines - but Python's
   **inspection** surface is its own: `send` / `next` / `__next__` / `close` with
   `StopIteration` carrying the body's return value, plus draining a generator in
   `js_pyiter`, `pySeqElems` and a `for`. PHP's two findings both applied:
   a generator is a floor **cell**, so `close()`'s flag lives in an identity-keyed
   side table (a `===` scan over two parallel arrays, `rbLambdas`'s shape), and
   layer 2 has **no tag test**, so `pyIsGen` is STRUCTURAL - the floor answers
   exactly one member on a tag-15 cell, `"next"` (`runtime.c:2868`), and nothing
   this value model builds carries a callable `next` of its own.
4. **The FMA remainder does NOT apply to python, and that is measured rather than
   assumed.** `grep -n 'math.Floor(.*)\*' abnf/jsrt*.go` finds three sites -
   `jsrt.go:2985`, `3008` and `3223` - and all three are **Ruby's** (`jsFlo`,
   `rubyToF`). Python's `%` is `pyFloorMod` (`jsrt.go:9403`), which is built on
   `math.Mod`; `math.Mod` is an exact scaled-subtraction loop and JS `%` is the
   same operation, so there is no fusable `x - q*y` in the path and no Dekker
   emulation is owed. Ruby's warning was worth running and the answer was no.

### What was reused rather than rewritten, and what it saved

* **`import "./regex.js"`** - ruby's finding, and python is the language it was
  found FOR. The 1,559-line shared engine compiles into `python-rt.ll` with
  everything else, and the three `js_pyre*` / `js_pyrx*` externs cost ~660 lines of
  Python-specific glue (`abnf/jsrtregexpy.go` is 692 lines) instead of ~2,200.
* **The exact float formatter.** `strconv.FormatFloat` is EXACT and rounds HALF TO
  EVEN; `ruby-rt.metajs`'s `rbExactDec` / `rbRoundHalfEven` / `rbFixed` / `rbSci` /
  `rbBaseStr` / `rbByteCut` are language-neutral and transferred verbatim as
  `pyB*`, comments included. Ruby's own diff-against-Go-`FormatFloat` recipe was
  re-run over 12 values x 5 precisions and matched.
* **`js_dict_new` / `js_pyget` / `js_pyset` / `js_pylen` / `js_pyeq` / `js_jadd`** -
  the `js_py*` family dart wrote language-neutral and ruby reused. It came back
  home: what had to be ADDED were the `__getitem__` / `__setitem__` / `__len__`
  dunder arms both had deleted as unreachable, the generic-alias arm of
  `js_pyget` (`list[int]` is a value in Python), the set arm of `js_pylen`, and
  `js_pylen`'s code-POINT string length.

**One inconsistency in `abnf/jsrt.go` had to be reproduced rather than repaired:**
`js_pylen` counts CODE POINTS (`pyStrLen`, 9773) where `js_pyget`'s string
indexing counts UTF-16 CODE UNITS (`rt.strAt`). Both are ported as they are.

### Ground truth

Oracle: **`/opt/homebrew/bin/python3`, Python 3.14.6**.

```
matrix                325/325
--full                5,618 assertions, 0 languages whose halves disagree (python 261)
                      grep -E 'BUT -frozen|VACUOUS|MISMATCH|FROZEN-DIFF' -> empty
                      (2026-08-04: python 269, the eight sig9-sig16 below)
--cross               119 programs, 0 divergent, 0 differing only in warnings
go test ./abnf/       ok
clang-check           16/16, and python joins the RUN-IT row
gen-python-rt-ll.sh --check   python-rt.ll is up to date (70,847 lines)
every other gen-*-rt-ll.sh --check   up to date (12 of them)
-frozen recompile     python-rt.ll byte-identical to the goja build
jsbootstrap.ll        untouched (metajs-to-llvm-ir.abnf was not edited)
```

**Verified from a CLEAN ARCHIVE, not the working tree** - `git archive HEAD | tar
x`, then only the four Python files copied in, `go build` INSIDE, run there. That
matters because js, typescript and kotlin were other agents' concurrent edits while
this ran:

```
tests/gen-python-rt-ll.sh --check   python-rt.ll is up to date (70,847 lines)
./py.exe                            full: 261 checks, 0 failures   (rc=0)
tests/clang-check.sh python         ok, and the clang executable agrees
```

Every Python program in `tests/`, `llvm.Run` against the native binary, stdout AND
stderr AND exit code:

```
python-test-1  python-test-complete  python-test-features  python-test-full
python-test-multifile (-i tests/imports)  python-test-recognize  python-test-try
python-test-widen                                           all SAME (rc=0)
```

### The differential probe - 34,813 lines, and the one defect it found

Every operand is read out of a LIST at run time, so nothing can be folded: NxN
integer pairs and NxN float pairs and the mixed grid over
`+ - * / // % ** == != < <= > >= & | ^ << >>`; the unary and predicate set;
string indexing, slicing (negative bounds and steps), `in`, repetition,
concatenation and ordering; list / tuple / dict / set construction, indexing,
slicing, `in`, equality, `del`, slice assignment, unpacking and starred
unpacking; dict insertion order and key equality; set algebra `| & - ^ <= < >= >`;
complex arithmetic and rendering; `format()` and f-string replacement fields over
many specs times many arguments; comprehensions (list, set, dict, nested, with
`if`); generators (`yield`, `send`, `close`, a `for` over one); classes,
MULTIPLE inheritance and the MRO order, `__init__ __repr__ __eq__ __add__
__radd__ __len__ __getitem__ __setitem__ __contains__ __call__ __slots__`,
descriptors, `isinstance`, `type()`; a raise/except matrix over the builtin
hierarchy with `else` / `finally` / re-raise; `global` / `nonlocal` / `del` /
closures / default arguments / `*args`; the `re` surface (many patterns times
many subjects times `search match fullmatch sub subn split findall finditer
escape group groups groupdict start end span expand`); `with` and context
managers. The operand bags carry both zeros, the int53 edges, 2^31 / 2^32 / 2^53,
1e+-100, DBL_MAX, 5e-324, 0.49999999999999994, an arbitrary-precision literal
past 2^53, and non-ASCII strings.

```
llvm.Run vs the native binary      BYTE-IDENTICAL, 34,813 of 34,813 lines
                                   (stdout, stderr and exit code)
```

It found **one real defect, in layer 2, invisible to `./test.sh`**:

> **`jsAdd`'s bigint arm, and the arm-ORDER hazard behind it.** Go's `jsAdd`
> reaches `bigArith` after the array and object arms because a `*jsBigInt` is
> never a `*jsObject`; here a `{__pybig}` IS a plain object, so the bigint arm
> was moved to the front - correctly, or `10**28 + 10**28` would concatenate two
> renderings. What that missed is that `bigArith` needs BOTH operands to be
> bigints (`bigPair`, `jsrt.go:2710`) and `rt.toNumber` has no bigint arm at all,
> so a MIXED pair falls all the way through to `NaN` in the Go twin. Here it fell
> into the object arm instead and answered the string
> `'010000000000000000000000000001'`. The fix is one line - the object arms
> EXCLUDE a `{__pybig}` - and the lesson generalises to every Go TYPE SWITCH
> ported into this value model: **moving an arm forward is not enough, the arms
> it jumped over have to be narrowed by the same predicate.**

**A SECOND "difference" was reported and is an ARTIFACT OF THE MEASUREMENT, which
is worth more than the finding would have been.** The probe prints a string
containing a **NUL**, and the comparison showed `llvm.Run` truncating the line
there while the native binary printed it whole. It does not: **`awk` truncates a
string at a NUL**, and both this plan's comparison script and the probe's own
`strip.awk` use awk to cut the emitted module off the front of `llvm.Run`'s
stdout. Dumped with `od -c`, the RAW bytes of both halves are
`esc|5|'<NUL>'|1` - identical. The cut is now computed on a NUL-free copy and
applied to the raw bytes with `tail -n +K`. **Any future language whose probe
prints a control character has to do the same**, or it will report a defect in
`js_pyprint` / `wtf8Clean` that is not there.

**And the emitter lowering was proved observably neutral, which is the check that
matters most for six lowered externs.** The same 34,813-line probe run under
`llvm.Run` against the grammar at `8588257` - the one that still calls
`js_pyfnscope` / `js_pyset_var` / `js_pyglobal` / `js_pynonlocal` /
`js_pydel_var` / `js_pyannot` / `js_pyclass_fill` and carries Python's binding
rule on the Go scope object - is **byte-identical** to the lowered grammar's
`llvm.Run` output. Built from a clean `git archive HEAD` in its own directory.

```
HEAD's Go-scope grammar vs the lowered grammar, both under llvm.Run
                                   BYTE-IDENTICAL, 34,813 of 34,813 lines
```

### Against REAL python3 - measured, and NOT this migration's

Oracle: **`/opt/homebrew/bin/python3`, Python 3.14.6**. 11,184 of 34,796 aligned
lines differ, and **`llvm.Run` and the native binary agree with each other on every one
of them** - the gate above says so - so not one is this port's. They are the
documented value model of both halves, and they fall into a handful of classes:

```
  3,956   AN INTEGRAL FLOAT RENDERS AS AN INT. int and float are the same plain
          double (pyTypeName decides by Math.trunc), so 1/1 prints 0, not 0.0.
  2,362   THE float64 VALUE MODEL: 2^53 of precision, and & | ^ are ToInt32, so
          `0 | 2147483648` is -2147483648.
  1,562   inf / nan are spelled Infinity / NaN / -Infinity.
    439   NO ZeroDivisionError for / // % by zero - NaN or Infinity instead.
    395   -0.0 renders as 0 (and 42 more inside a container).
    548   THE `re` RESULTS AND TUPLES GENERALLY: groups() / span() / subn() /
          e.args / dict.items() / *args are ARRAYS, so ('', 0) prints [0, 5].
    361   THE FORMAT MINI-LANGUAGE IS A SUBSET. pyFormatBody (jsrt.go:9737) has
          f/F/d/x/X/b/o and nothing else - no e, E, g, G, %, c, n, no `,`/`_`
          grouping - and x/X/o/b/d truncate to 32 bits.
    350   << and >> are 32-bit; 53 more where a negative shift does not raise.
    240   COMPLEX, INSTANCES AND TYPE OBJECTS render [object Object]: repr /
          str / an f-string do NOT dispatch __repr__, though x.__repr__() works.
          Pre-existing at 8588257, checked directly from a clean archive.
    216   repr never ESCAPES: no \ ' \t \n \x.. escaping.
    196   THE ARBITRARY-PRECISION SURFACE beyond + - * / %: ordering against a
          bignum is always False and `float == bignum` always True, because
          rt.toNumber has no bigint arm at all.
    ~200  the long tail: isinstance against a builtin type object is False (the
          builtin types are not real classes), the builtin exception hierarchy
          is flat, True and 1 are not the same dict key, `is` on equal scalars,
          set() renders {}, no TypeError for mixed operands, no OverflowError,
          last-ulp drift in ** and sqrt.
```

Sections that matched CPython **exactly**: classes / MRO / dunders / descriptors /
`__slots__`, generators, `with`, comprehensions, unpacking, closures / `global` /
`nonlocal` / defaults, dict mutation, `range`, and ten of the fourteen regex
groups. Those are the parts this migration actually rewrote.

None of these is new, and each one is a FEATURE decision rather than a defect of
the runtime split: fixing any of them means changing both halves and the
interpreter grammar together, which is the same conclusion the Lua, Swift and
Ruby formatter boundaries reached.

### What is owed, and what generalises

1. Nothing for the gate. `abnf/jsrt.go`'s Python half stays until the change is
   committed - and, like Ruby's, it is NOT a file that can ever go, because
   `llvm.Run` is still the non-`-exe` engine for every language.
2. The `py_setvar` residue named above (a name bound strictly below a binding
   boundary AND above it). The exact fix needs a per-scope containment test, which
   is the one thing the floor's scope API does not expose; a `js_scope_has(s, n)`
   that does NOT walk the chain would close it in three lines of `runtime.c` and
   three of `abnf/jsrt.go`. **Reported, not made** - the floor is another agent's
   this session, and nothing in `tests/` reaches it.
3. `pyAnnot` decides "is there an `__annotations__` dict in THIS scope already"
   statically, where `abnf/jsrt.go` asks `sc.find` at run time. The two differ only
   for an annotation the program does not REACH, which would create the dict here
   and not in the Go twin.
4. **`*args` overbinds when the positional count equals the declared slot
   count**, and `**kwargs` then holds a scalar rather than a dict:
   `def f(a, *rest, **kw)` called as `f(1, 2, 3)` binds `rest` to a number, and a
   later `len(rest)` or `kw.keys()` ABORTS. Found by the probe, present in BOTH
   halves, and verified unchanged at `8588257` from a clean archive - a
   pre-existing emitter bug, not this migration's, and out of scope for a
   runtime split.

   **DIAGNOSED EXACTLY 2026-08-04, oracle `python3` 3.14.6 - and it is NOT an
   emitter bug.** `makeParamList`, `emitSig` and `emitClosure` in
   `python-to-llvm-ir.abnf` are all correct, and so is `pyBindCall`, which for
   `f(1, 2, 3)` builds `[1, [2, 3], {}]` exactly as prologue layout B wants. The
   defect is the three lines AFTER it in `js_pycall`:

   ```go
   bound := rt.pyBindCall(callee, args.elems, kw)
   if len(bound) == len(args.elems) {
       return w(rt.callH(callee, jsUndef, args.elems, a[1]))   // <- the ORIGINAL array
   }
   ```

   That length test is a stand-in for "binding was a no-op, so pass the incoming
   handle through and keep its identity". It is wrong whenever the bound array
   happens to be the same LENGTH as the positional one, which for the extended
   layout is exactly `len(pos) == len(names) + 2` - i.e. "the positional count
   equals the declared slot count", which is what the probe saw. The measurement:
   `f(1)` and `f(1, 2)` are right, `f(1, 2, 3)` aborts with `len() of a number`,
   `f(1, 2, 3, 4)` is right again. The INTERPRETER half is not affected at all
   (`python-interpreter.abnf` binds parameters itself), so this is `llvm.Run` and
   the native binary only - which is why `--cross` cannot see it either.

   **FIXED 2026-08-04, both halves and the assertions in ONE change.** The fix is
   one condition in two places, and it is decided by IDENTITY, never by length:

   - `abnf/jsrt.go`: `pyBindCall` is now a wrapper over **`pyBindCallR`, which
     returns `([]interface{}, bool)`** - the bool is "binding built a NEW array".
     `js_pycall` asks that instead of comparing lengths, and only then keeps the
     caller's own argument handle (`rt.callH` with `a[1]`, an identity
     optimisation and not a semantic choice). The other eight `pyBindCall` call
     sites are untouched.
   - `languages/lib/python-rt.metajs`: `pyBindCall` returns `pos` ITSELF in each
     of its no-op arms, so `pyCall(callee, bound)` covers both cases and the test
     disappears entirely. The false comment - "the two arrays cannot differ in
     content when their lengths agree" - is replaced by the counter-example.

   **Ratchet**: `tests/python-test-full.py` SECTION 09, `sig9`-`sig16`, **8
   assertions** (python 261 -> 269). Written against the layout directly: a
   `*args`/`**kwargs` def binds to `names.length + 2` slots, so `sig11`
   (`st_args(1, 2, 3)` on one declared name) and `sig16` (`st_args2(1, 2, 3, 4)`
   on two) are the calls whose POSITIONAL count hits that same number. All eight
   were checked against **`python3` 3.14.6** standalone first.

   **Discriminating power, from a clean `git archive` of `44bb09a` with ONLY the
   ratchet file copied in and `go build` run inside**: **2 of 8, by ABORT**, and
   the abort is what makes it worth reporting - at `44bb09a` the whole file dies
   with `len() of a number`, so the compiled half loses all of section 09 while
   the INTERPRETER half is green at 269. That is the "halves disagree" column the
   assertion was held back to avoid. Per assertion, at `44bb09a`, compiled half:

   ```
   sig9  sig10  sig12  sig13  sig14  sig15   True     (the guard rails)
   sig11                                     ABORT    len(pos) == 1 + 2
   sig16                                     ABORT    len(pos) == 2 + 2
   ```

   With the fix, all three engines answer 269 checks / 0 failures.
5. Python's `str` has **no method library at all** in either half - no `upper`,
   `strip`, `join`, `split`, `startswith`. `"abc".upper()` fails identically in
   both engines with `unknown String method: upper`, from `jsrt.go:2784` and from
   its port. Adding one is a new FEATURE in three places (the interpreter grammar,
   the compiler grammar and this file), not part of this migration, and it was
   deliberately not invented here.

### launch.json entries wanted (not added - not my file)

All three pass natively today, at the working tree and from the clean archive:

```
"Python native binary, full syntax (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/python-to-llvm-ir.abnf", "tests/python-test-full.py", "-q", "-exe", "tests/python-native-full.out"]
"Python native binary, feature matrix (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["languages/python-to-llvm-ir.abnf", "tests/python-test-features.py", "-q", "-exe", "tests/python-native.out"]
"Python native binary, multifile (compiler, -exe + the C floor and the MetaJS layer 2)"
  args: ["-i", "tests/imports", "languages/python-to-llvm-ir.abnf", "tests/python-test-multifile.py", "-q", "-exe", "tests/python-native-multi.out"]
```

## Order

Sorted by extern count and Go-twin size, which is the best available proxy for effort:

```
language     externs   dedicated Go twin
java              61   804   <- DONE 2026-08-03, gate MET (54 measured)
csharp            67   813   <- DONE 2026-08-03, gate MET (68 measured, not 67)
swift             68   986   <- DONE 2026-08-03, gate MET (66 after the scope-probe lowering)
dart              72   896   <- DONE 2026-08-03, gate MET (67 measured; 64 before
                             the scope-probe lowering added its three floor calls)
js                88   (jsrtjsprint 1320 + jsrtjsbig 505)   <- DONE 2026-08-03,
                             gate MET (87 measured over every test file; 83 for the
                             full ratchet, and 59 exported by layer 2 once the
                             -exe-only operator routings are counted)
typescript        89   shares js's   <- DONE 2026-08-03, gate MET (86 measured),
                             and it shares layer 2 as well: lib/js-rt.ll, verbatim
ruby              91   -   (semantics live in abnf/jsrt.go)   <- DONE 2026-08-03,
                             gate MET (93 measured; the defined? lowering removed one
                             layer-2 extern and added one floor one, so 93 stayed 93)
go                92   630 (jsrtgolang)   <- DONE 2026-08-03, gate MET (84 measured,
                             83 before the js_gounctl lowering added its two floor calls)
kotlin            96   5934   <- DONE 2026-08-03, gate MET, 979/979 native (96
                             measured, 95 before the scope probes were lowered).
                             Largest of the set; the row was held, then released.
python            99   -   (semantics live in abnf/jsrt.go)
                             <- DONE 2026-08-03, gate MET, 261/261 native. Lowered
                             SIX externs in the emitter, the most of any language.
php              101   1968   <- DONE 2026-08-03, gate MET, 306/306 native. Held at
                             221/306 until keysOf and coroutines landed, then released.
```

Two adjustments to plain size order:

1. ~~**The sized-integer floor tag (Part 2) lands before java, csharp, go and kotlin**~~
   - **done** (tag 13 of `languages/lib/runtime.c`, 2026-08-02), and **done in all
   three engines** since 2026-08-03: the eleven `sint*` globals now exist in the Go
   twin (`giBindings`) and in the interpreter half (`metajs-interpreter.abnf`) too,
   pinned by `tests/metajs-test-full.js` SECTION 24. **`jsJFlo`, which was what those
   four still needed, is DONE too** (tag 14, 2026-08-03, SECTION 26): both primitives
   exist in all three engines, so **nothing in the floor blocks java, kotlin, csharp,
   go, swift or dart any more**. The layer-2 interface of each is written out in
   part 2 - `sint*` for the integer, `flo*` for the double - and `keysOf` is there for
   the object walking every runtime library needs.
2. **js and typescript go together**, since typescript has no dedicated twin and shares
   js's. Same for ruby and python, whose semantics live in the shared `abnf/jsrt.go` -
   expect those two to drag more of the shared file with them than their extern count
   suggests.

Suggested first: **java**. Smallest surface, a self-contained 804-line twin, and it
exercises the sized-integer tag immediately - which is exactly what you want the first
post-Lua language to prove.

**Swift's evidence, for whoever picks the next one (2026-08-03).** Swift's 68 was
the third-largest number in this table and it took the LEAST work of the three
ported so far, because extern COUNT is a poor proxy once the floor has both number
tags: 35 of the 68 were already in the floor and 10 more were language-neutral, so
the language-specific surface was 21. **Re-sort by "how much of this language's
value model is a tag the floor already has", not by extern count.** On that measure
`csharp` and `dart` are the obvious next two - both have a dedicated twin under 900
lines and both are `sint`+`flo` languages exactly like Swift - and `js`,
`typescript`, `python` and `ruby` are the hard ones, because they need WALL 1
(coroutines) and share `abnf/jsrt.go` rather than owning a twin.

**Both of those two were then done the same day, and the prediction held**:
csharp's own surface was 24 and dart's 17, against extern counts of 67 and 72.
**The five remaining are go, kotlin, js, typescript, python and ruby**, and the
sort that now matters most is a different one again: `go` and `kotlin` still own a
twin and are `sint`+`flo` languages, so they are the last two easy ones; the other
four share `abnf/jsrt.go`, which means a layer-2 file for any of them drags in the
NEUTRAL column rather than a language-specific one.

**Go was then done, and it is the counter-example to the extern-count sort in the
other direction (2026-08-03).** Its 92 was the second-largest number in the table
and its FLOOR column came out at 42, the largest of the seven - because the sized
integer tag was specified against `abnf/jsrtint.go`, which is Go's own file, so
the language that drove the tag pays nothing for it. Its layer-2 surface was 42,
of which 32 are genuinely Go's (`fmt`, the map, panic/defer/recover, channels,
embedding). **The remaining four are kotlin, js, typescript, python and ruby**;
kotlin still owns a twin and is a `sint`+`flo` language, and the other four share
`abnf/jsrt.go`. Six of dart's neutral externs and go's ten (`js_dict_new`,
`js_goslice`, `js_gospread`, `js_is_type`, `js_jadd`, `js_pyin`, `js_pyset`,
`js_range_key`, `js_range_len`, `js_range_val`) are now written out, so those four
start further ahead again. Dart's port already wrote six
of those neutral externs (`js_pyeq`, `js_pyin`, `js_pyiter`, `js_pyset_new`,
`js_pyslice`, `js_pyspread`), so python and ruby start ahead of where this table
suggests.

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

## Retiring the Go runtime - MEASURED AND DECLINED 2026-08-03. NOTHING WAS DELETED.

The item as written said: *29,670 lines, deletable a language at a time, only after
that language passes natively.* All sixteen languages now pass natively, so the
precondition is met and the question came due. It was measured rather than assumed,
and **the answer is KEEP - not one line was deleted.** The evidence is below; every
number in it was produced at `d64776a` with a clean working tree.

The short version, in the order the brief asked:

1. **`llvm.Run` CANNOT link and execute `runtime.ll` + `<lang>-rt.ll` the way `-exe`
   does.** `linkRuntimeModules` is not enough, and the gap is not small - it is
   `setjmp`/`longjmp`, four pthread primitives, `dlopen`/`dlsym`, and a conservative
   stack scan that has nothing to scan. Measured, below.
2. **There is therefore no "cost" to quote**, because there is no working
   configuration to time. What CAN be timed is the only alternative route - moving
   the suite onto the native binaries - and that is a **2.6x per-entry** cost that
   also does not reach the interpreter halves at all.
3. **What would be lost** is the exact mechanism that found ~15 real layer-2 defects
   in the last two weeks, none of which `./test.sh` could see. Named, below.
4. **What cannot go** is enumerated at the end, and it is much more than
   `jsrt.go`'s host globals: `abnf/jsrt.go` is the engine `-frozen` itself runs on.

### 1. Can `llvm.Run` run layer 2? No, and here is the measurement

A probe module supplying only `jsmain`/`jsdispatch`, handed to `run()` with
`languages/lib/runtime.ll` as the runtime input - i.e. exactly what
`linkRuntimeModules` is for - dies immediately:

```
$ go test ./abnf/ -run TestFloorUnderInterpreter -v
FLOOR-PROBE PANIC: IR interpreter: call to undefined external function @getenv
```

**The link itself is fine.** That is worth stating plainly, because it is the good
news and it is what `linkRuntimeModules` already buys: parsing every `*-rt.ll`
against `runtime.ll` shows the pair is CLOSED over the floor for every language.

```
bash-rt.ll     unsatisfied-by-floor(0)=[]      java-rt.ll     unsatisfied-by-floor(0)=[]
batch-rt.ll    unsatisfied-by-floor(1)=[bat_lookup]   js-rt.ll  unsatisfied-by-floor(0)=[]
csharp-rt.ll   unsatisfied-by-floor(0)=[]      kotlin-rt.ll   unsatisfied-by-floor(0)=[]
dart-rt.ll     unsatisfied-by-floor(0)=[]      lua-rt.ll      unsatisfied-by-floor(0)=[]
go-rt.ll       unsatisfied-by-floor(0)=[]      php-rt.ll      unsatisfied-by-floor(0)=[]
python-rt.ll   unsatisfied-by-floor(0)=[]      ruby-rt.ll     unsatisfied-by-floor(0)=[]
swift-rt.ll    unsatisfied-by-floor(0)=[]
```

Every `js_*` a layer-2 file leaves undefined is defined by the floor's 309 function
bodies. So the whole obstacle is the floor's OWN 16 undefined externs, of which the
interpreter can resolve exactly **two**:

```
runtime.ll  defs=309  resolvable by llvm.Run = [malloc putchar]
            UNRESOLVABLE(16) = dlopen dlsym exit getenv jsdispatch jsmain longjmp
                               pthread_cond_broadcast pthread_cond_init
                               pthread_cond_wait pthread_create pthread_mutex_init
                               pthread_mutex_lock pthread_mutex_unlock setjmp write
```

`jsmain` and `jsdispatch` come from the program module, so they are free. The other
fourteen split three ways:

- **`getenv` / `write` / `exit` are easy.** Stubbing them in the probe took nine
  lines and moved the failure one step:

  ```
  FLOOR-PROBE PANIC: IR interpreter: call to undefined external function @setjmp
  ```

  `main()` calls `setjmp` before it calls anything else, so this is the first
  instruction of every program in every language.

- **`setjmp` / `longjmp` are a real project, not a stub.** The interpreter has no
  unwind path out of `ma.call` at all - that is already why `exit` and `abort` are
  deliberately absent from `libcExterns` (see the header there). A correct
  implementation is possible in principle: `setjmp` records the interpreted frame
  plus block/instruction index and `longjmp` panics with a token that the recorded
  frame's `ma.call` recovers and resumes at. It is perhaps 100 lines - and every
  `throw` in every one of the sixteen languages goes through it, so a subtle bug in
  it is a subtle bug in all sixteen at once.

- **`pthread_create` + `dlopen`/`dlsym` are the coroutine machinery, and they are
  worse.** `ma.mem` is a `[]byte` grown by `append` in `ma.alloc`; two goroutines
  mutating it race and reallocate under each other. `dlsym("coro_entry")` has to
  answer a real code address, and the interpreter's answer to a function-as-value is
  a funcId in an `i32` - the exact limitation `c-to-llvm-ir.abnf` has and that
  `dlopen` exists in the floor to work around.

**AND THE ONE THAT CANNOT BE FIXED AT ALL: the collector has nothing to scan.**
This is the finding that settles it, and it is a silently wrong answer rather than a
crash. With `setjmp` stubbed to 0 the probe DOES run - a `js_arr_new`, 20,000
`js_arr_push`, then `js_get(arr, 0) + 2`:

```
PROBE_GC unset / o / p     FLOOR-PROBE ok: ret=9    gc: collections=0
PROBE_GC=s   (stress)      FLOOR-PROBE ok: ret=0    gc: collections=23 live=10432 pinned=
```

**9 becomes 0** - the array was collected while it was live, and the stats line's own
`pinned=` field came back empty because the string it was built from was collected
too. The cause is structural: `gc_collect` scans `[sp, GC_STACK_BASE)` of the C
stack, but under `llvm.Run` an SSA value lives in `fr.regs`, a **Go slice outside the
interpreted address space**, and an `alloca` is `ma.alloc(...)` - bump-allocated
UPWARD from the same arena as the globals and never freed, so there is no stack to
walk and the walk would run the wrong way if there were. Part 1b's design - "the
running side owns `GC_STACK_BASE`", conservative roots, nothing moves - is written
against a real machine stack and has no image in this interpreter.

Giving the interpreter a real spilled, downward-growing, reclaimed stack in `ma.mem`
is not a fix to `linkRuntimeModules`; it is a second machine.

### 2. The cost, since there is no configuration to time

The gates at `d64776a`, timed on this machine (`-j 8`, warm script cache):

```
matrix        325/325       23.1s wall    186.3s user
--full        5,686 assertions, 0 halves disagree    83.1s wall   314.2s user
--cross       119 compared, 0 divergent               9.6s wall    64.0s user
clang-check   16/16 "ok, and the clang executable agrees"   95.9s wall  111.9s user
```

The only way the suite could run on layer 2 is through the native binaries, which is
what `clang-check.sh` already does for one program per language. Per entry, measured
on the largest ratchet file there is:

```
$ mec languages/php-to-llvm-ir.abnf tests/php-test-full.php -q          1.42s   (llvm.Run, Go twin)
$ mec languages/php-to-llvm-ir.abnf tests/php-test-full.php -q -exe P   3.70s   (build, layer 2)
$ P                                                                     0.05s   (run)
```

**2.6x per entry**, and it buys less than it costs, for three reasons:

- It reaches only the COMPILER halves. An `<lang>-interpreter.abnf` is a tag script;
  it has no `-exe` and never will. Half of the 325-entry matrix and both sides of
  `--cross` stay on `abnf/jsrt.go` no matter what.
- It makes clang a hard dependency of `./test.sh`, which today only
  `tests/clang-check.sh` needs.
- It breaks the tool's primary invocation. `mec languages/kotlin-to-llvm-ir.abnf
  prog.kt` with no `-exe` prints the module and runs it; that path IS the Go twin.

### 3. What would be LOST, named precisely

`./test.sh` in all three groups compares an engine **with itself**: goja against
`-frozen` is one implementation under two script hosts, and `--cross` is an
interpreter half against a compiler half, both hosted by `abnf/jsrt.go`. None of the
three has ever compared the two IMPLEMENTATIONS of a language's semantics.

The thing that does is the **per-language differential probe**: one large generated
program, run under `llvm.Run` (Go twin) and as the native binary (layer 2), diffed
byte for byte. Every language section in Part 3 has one, and their findings are the
answer to this question:

```
ruby    36,378 lines   4 defects   incl. the arm64 FMA fusion: 9007199254740991 % -3
                                   is -2 under llvm.Run and -1 natively, and real
                                   ruby says -2 - so the TWIN was right
go      16,550 lines   3 defects   incl. js_gotypeis: the twin's type switch has no
                                   arm for either number box, layer 2 answered
                                   "float64"/"int"; broke byte identity either way
java    20,296+ lines  3 defects   python  34,813 lines  1 defect (jsAdd's bigint arm,
dart    31,675 lines   2 defects           which answered a CONCATENATED string)
swift   20,296 lines   1 defect     csharp  1 defect      php  1 divergence left
lua     13,306 lines   0 (byte-identical) - but the float `%` formula was wrong in
                                   ALL THREE halves and the twin's was FMA-dependent,
                                   so the two GO halves disagreed on 128 of 12,557
```

Every one of those sections says the same sentence in its own words: **"invisible to
`./test.sh`, because it compares each engine with itself."** That is the class of
defect that becomes invisible if the twin goes:

> **A semantic error that layer 2 and the ratchet agree on.** A ratchet assertion is
> written by the same person, at the same time, from the same reading of the language
> as the layer-2 code it pins. When that reading is wrong, the assertion pins the
> wrong answer and every engine agrees. The Go twin is the only artifact in the tree
> that was written from an INDEPENDENT reading, at a different time, in a different
> language, against a different value model - and Part 3 records fifteen occasions on
> which that independence paid.

Part 3's own rollout recipe already says it, at step 3: *"`abnf/jsrt<lang>.go` is the
authoritative spec. Match it."* Deleting the spec after building the implementation
from it is exactly backwards. There is also a THIRD differential inside the matrix
that would go with it: `abnf/jsrtregex.go` is a line-by-line Go port of
`languages/lib/regex.js`, and the matrix runs an interpreter half (which uses the JS)
against a compiler half (which uses the port) on the same input - see that file's
header.

**And the honest counter-argument, stated rather than buried.** The twin is not free:
it is a second implementation that must be kept in step, and Part 3 records defects
that were in the TWIN (Lua's `%` in all three halves, Go's `js_gotypeis`) as well as
in layer 2. A wrong oracle costs real time. The judgement here is that a wrong oracle
that DISAGREES is still strictly better than no oracle, because disagreement is what
sends someone to the real toolchain; the Ruby FMA row is the clean example, where the
diff is what produced the question and real `ruby 2.6.10` produced the answer.

### 4. What must be KEPT regardless - the enumeration

This is the part the brief asked for precisely, and it is larger than the original
item assumed. `abnf/jsrt*.go` is 29,825 lines (plus 185 of test), and it splits:

```
SHARED CORE - cannot go under ANY answer                         12,713 lines
  jsrt.go         9,837  the MetaJS runtime itself. newJSRT is called from
                         frozen.go:220, :530 and :813 - this is the engine
                         `-frozen` RUNS ON. Every tag script of every grammar,
                         goja-free, executes through it. It is not "the language
                         runtimes"; it is the compiler's own interpreter.
  jsrtjsprint.go  1,320  the String/Array/Number method table reached through
                         js_get - tag scripts live on it
  jsrtint.go        682  giBindings: sint/sintRaw/sintHi/sintLo/sintOp/sintNum,
                         the sized-integer host bindings a LAYER-2 file calls
                         (floor host ids 62 etc.) - deleting these breaks the
                         NATIVE path, not the Go one. Also programJSBindings and
                         keysBindings (keysOf/delKey, floor host ids 61 and 63).
  jsrtjsbig.go      505  arbitrary-precision integers, reached from the core
  jsrtjvm.go        369  jfBindings: the boxed double (flo*), same argument

PER-LANGUAGE TWINS - the only candidates, and the ones being kept   12,839 lines
  jsrtkotlin 5,934 · jsrtphp 1,989 · jsrtswift 986 · jsrtdart 896
  jsrtcsharp 813 · jsrtjava 804 · jsrtlua 787 · jsrtgolang 630
  Reachable ONLY from a compiled module's extern table (each registers itself
  additively through rxExtraExterns + init(), touching nothing else), i.e. only
  from `<lang>-to-llvm-ir.abnf` under llvm.RunJS. Also the differential oracle.

REGEX - Part 4's other half, unchanged by this decision              4,273 lines
  jsrtregex 2,164 · jsrtregexpy 692 · jsrtregexkt 623 · jsrtregexjs 575
  jsrtregexptr 219
```

Note what the first block means for the original item's plan. It said *"`jsrt.go`'s
host-function globals become MetaJS last, since every language depends on them."*
They cannot become MetaJS at all: the thing that would execute that MetaJS is
`jsrt.go`. The dependency is not "every language" - it is the toolchain.

`libcNative` (`llvmmap.go`) also stays, unconditionally: it serves `-exe`'s
correctness contract (condition 2 at `libcExterns`) and the C, bash and batch
grammars, none of which have a handle runtime at all.

### The decision, and what a future person should do instead

**Keep the Go runtime. The second implementation is worth more than the 29,825
lines.** If the goal is to shrink the tree, the honest targets are elsewhere and each
is independently checkable:

- **Regex** (Part 4's first half): three implementations of ONE thing, and unlike the
  language twins two of them - `jsrtregex*.go` and `lib/regex.js` - are already
  compared by the matrix. Folding bash's C ERE engine in is the costable question.
- **A native differential in the suite.** The probes are ad hoc and run by hand. A
  `./test.sh --probe` that runs each language's existing probe through both engines
  and diffs would make the fifteen defects above a RATCHET instead of an archaeology
  exercise - which is the highest-value work this section can point at, and it
  requires the Go twin to exist.

The measurement scripts for this section were temporary (`abnf/floorprobe_test.go`,
deleted; its two probe modules are quoted above in full) - re-derive them from the
three code blocks in section 1 if the question is reopened.

---

# Coroutines - the fifth-language wall - LANDED 2026-08-03

**The primitive is in `languages/lib/runtime.c`.** It stopped being a proof of
concept in a temp directory: tags 15/16, the growable coroutine registry, the
per-coroutine `jmp_buf` pool and the throw barrier are in the floor itself, and
`languages/lib/runtime.ll` is regenerated from it. `tests/coro-poc/build.sh`
still runs `gen.ll` in both engines and diffs them - what changed is that
`--diff` now prints nothing, because there is no patch left to apply.

**The gate is met.** A `throw` across a yield was undefined behaviour and was
named here as the thing that had to be fixed before any language could be
migrated onto this. It is fixed, by the design this section named: the body
unwinds only to a barrier `coro_entry` sets on its OWN stack, the value travels
back in the generator cell, and `gen_resume` re-raises it on the resumer's
stack - `recover()` and a re-`panic`, transliterated. The `JB` pool became
per-stack at the same time, which is what makes a `try` INSIDE a body legal.
The measurements are in section 5.

**ONE THING THE NEXT PERSON MUST DO FIRST, and it is a hard link error rather
than a warning - DONE FOR PHP 2026-08-03, and still owed by the other three.**
A layer-2 file that defines `js_genfn` / `js_yield` as loud stubs now collides
with the floor:

```
$ clang php-rt.ll runtime.ll <a jsmain/jsdispatch>
duplicate symbol '_js_genfn' in: runtime.o / php-rt.o
duplicate symbol '_js_yield' in: runtime.o / php-rt.o
ld: 2 duplicate symbols
```

The fix is the one section 4 already prescribed - **delete those two functions
from the `-rt-lib` file** and the floor's own win the link. `php-rt.metajs` has
had them deleted; `grep -l 'define i64 @js_genfn' languages/lib/*-rt.ll` answered
php and nothing else at the time, so no other layer-2 file is affected yet, but
the next one written must not add them back.

**AND THE ERROR MESSAGE POINTS THE WRONG WAY.** Measured while doing it: the
build reported `2 unresolved symbol(s): js_genfn, js_yield` when the real `ld`
error was the duplicate above, because `buildExecutable` (`abnf/llvmmap.go`)
builds its report from the module's undefined declarations filtered by whether
the linker text mentions each name - and a duplicate-symbol error mentions
exactly those names. Expect to read "unresolved" and find "duplicate".

The PHP migration named WALL 1 as the one architectural blocker of the remaining
rollout: "**Coroutines** (`js_genfn` / `js_yield`). No spelling in layer 2 at any
effort." This section settles it. Short version:

- **It is four languages, not five.** Ruby needs nothing at all - measured, below.
- **The C floor can host a real resumable coroutine**, expressible in the C subset
  `c-to-llvm-ir.abnf` accepts, with no change to any emitter and no change to
  `abnf/jsrt.go`. A proof of concept runs **byte-identically under `llvm.Run` and
  as a clang-linked native binary**, in all four collector modes.
- The emitted IR does not change **at all**, which is the property the two-worlds
  constraint really asks for: the module below is one file, run in both engines.
- **The recommendation is option B, and the one thing that worries me is throw.**
  A `throw` that crosses a yield is currently undefined behaviour, and
  `MEC_GC=stress` catches it - the evidence, and the fix, are at the end.

Sections 1 to 4 below are the investigation as it stood before the primitive
landed, and they are kept because the ARGUMENT is what justifies the shape - why
a thread and not `ucontext`, why the floor and not five emitters, why eager
draining and replay are dead. Section 5 is the throw gate and now records how it
was closed rather than what it would cost.

## 1. The requirement, per language, from the ratchet files

`js_genfn`/`js_yield` in `abnf/jsrt.go` (lines 434-579, 5376-5405) are a
goroutine plus two unbuffered channels: `js_genfn` wraps a compiled closure so
that CALLING it runs nothing and answers a `jsGenerator`; `step()` starts the
body on its own goroutine and hands over on `resume`/`yields`, strictly
alternating, so exactly one goroutine runs at a time and no runtime state is
locked. `js_yield` sends out and reads back in one line each.

Which languages reach which part of that, and what their own ratchet actually
drives:

```
language     emits js_genfn   ratchet needs                       hardest case
ruby             NO           nothing                             -
typescript       yes          one-directional next()              a single .next()
csharp           yes          one-directional, GENUINELY LAZY     infinite iterator
php              yes          one-directional, GENUINELY LAZY     infinite generator
js               yes          BIDIRECTIONAL next(v)               sit.next(21) === 42
python           yes          BIDIRECTIONAL send(v)               e.send(21) == 42
```

**Ruby is not blocked and never was.** `Fiber|Enumerator|\.lazy|to_enum|enum_for`
has no match in any `tests/ruby-test-*.rb`, `ruby-to-llvm-ir.abnf` emits no
`js_genfn`, and `tests/ruby-test-full.rb:27` puts "threads/fibers/ractors"
explicitly out of scope. Ruby's `yield` is a **block call** - the callee invokes
the block synchronously and it returns normally - which is a callback, not a
coroutine. So WALL 1 blocks **php, js, typescript, python** (and csharp, which
the PHP report did not list), and the plan should say four rather than five.

**Eager materialisation is dead, and for two independent reasons.** The first is
the one the PHP report gave - an infinite source:

```php
tests/php-test-full.php:492  function s19fib() { $a = 1; $b = 1; while (true) { yield $b; ... } }
tests/php-test-full.php:494  function s19infdel() { $i = 0; while (true) { yield from s19one($i); $i++; } }
tests/php-test-full.php:475  foreach (s19fib() as $n) { if ($n > 1000) { break; } $fib .= $n . ","; }
tests/php-test-full.php:489  foreach (s19infdel() as $v) { if ($del === "012") { break; } $del .= $v; }
```

```csharp
tests/csharp-test-full.cs:518  // An iterator block is LAZY: S19Endless never terminates on its own, so itr5 and
tests/csharp-test-full.cs:519  // itr6 only pass if the sequence is produced one value at a time rather than
tests/csharp-test-full.cs:520  // materialized at the call.
tests/csharp-test-full.cs:534  { int i = 0; while (true) { yield return i; i = i + 1; } }
```

The second is **observable interleaving**, which even a finite generator has:

```php
tests/php-test-full.php:482  $og = function () use (&$order) { $order .= "y1;"; yield 1; $order .= "y2;"; yield 2; };
tests/php-test-full.php:483  foreach ($og() as $v) { $order .= "c" . $v . ";"; }
tests/php-test-full.php:484  check("gen9", $order === "y1;c1;y2;c2;");
```

An eager drain gives `y1;y2;c1;c2;`. Both PHP tests are pinned to php-src's own
`tests/generators/` expectations, so they are not ours to relax.

**Two languages need the value to travel BACK in**, which rules out anything that
only produces a sequence:

```js
tests/js-test-full.js:311  var withSend = function* () { var got = yield 1; yield got * 2 }
tests/js-test-full.js:314  check("gen5", sit.next(21).value === 42)
```
```python
tests/python-test-full.py:311      def echo():
tests/python-test-full.py:312          got = yield 1
tests/python-test-full.py:313          yield got * 2
tests/python-test-full.py:316      check("gen3", e.send(21) == 42)
```

Nothing in any of the six ratchets reaches `gen.throw()` or `gen.return()`
(`jsGenerator.finish` is implemented and dead), and PHP's own header says
`->send()` is out of scope. So the required surface is: **lazy pull, a value
sent in, a `{value, done}` record carrying the body's return value**, and PHP's
inspection half (`current`/`key`/`valid`/`getReturn`).

One thing worth carrying: **js, typescript and python currently DRAIN in their
loop lowering** (`js_iterable` in `js-to-llvm-ir.abnf:3567,4108`, `js_pyiter`
in `abnf/jsrt.go`), so their for-loops are already eager and would hang on an
infinite source. Their ratchets do not test one. That is a latent gap in those
three, not a new cost of this work - but a real coroutine primitive is what
makes it fixable at all.

## 2. The options, costed against evidence

### The constraint, stated precisely

The emitted IR runs in two worlds and must stay byte-identical. It already does:
`php-to-llvm-ir.abnf:4793`, `js-to-llvm-ir.abnf:3525`,
`typescript-to-llvm-ir.abnf:3072`, `python-to-llvm-ir.abnf:3059` and
`csharp-to-llvm-ir.abnf:2293` all emit `js_genfn` over a `js_closure` today, and
`abnf/jsrt.go` implements it. **The only half that is missing is the C floor.**
That asymmetry is the whole argument for what follows: an option that changes the
floor changes ONE side of a working pair; an option that changes the emitter
changes the side that is already correct, in five grammars, forever.

### Option A - a CPS / state-machine transform in the emitter

The generator body becomes a switch over resume points; no runtime stack is
needed and the result is deterministic, which is what most compilers do.

Costed, and rejected as the FIRST move:

- **Five emitters, not one.** Every `yield` site (`php:2433,2518`,
  `js:3556,3581`, `typescript:3103,3128`, `python:2488,2505`, `csharp:2278,2316`)
  becomes a state split, and every one of them has `yield` in EXPRESSION position
  (`var got = yield 1`), which means the split has to happen inside an expression
  the emitter has already begun to lower.
- **It has to survive `try`/`finally`.** The control-signal protocol
  (`js_ctl_*` / `excDispatch` in `lib/compile-core.js`) is what a `return` out of
  a `finally` uses today; a yield inside a try body would have to be reachable
  from the state dispatcher as well, i.e. the exception machinery and the
  generator machinery become one design instead of two.
- **`yield from` over another generator** (`php:2518`, `python:2505`,
  `typescript:3128`) is a loop whose body yields, so the transform has to nest.
- It would also have to be written in the tag-script dialect. That is possible -
  part 1a's one-entry phi shows an emitter can restructure blocks after the fact -
  but it is a large piece of work in five files, and the measurement below says
  the alternative is 260 lines in one.

Option A stays the right ANSWER eventually (it removes the per-yield cost
measured in section 3 entirely), but it is not the right FIRST move, and nothing
in it is wasted by doing option B first: the extern surface is identical.

### Option B - a real second stack in the C floor  <- RECOMMENDED

Two sub-options, both measured on this machine (macOS 15.5, arm64, Apple clang):

**B1, `makecontext`/`swapcontext`.** They work - the probe printed
`resume 0 / yield 1 / ... / body done`, exit 0 - but they are **deprecated on
darwin** ("first deprecated in macOS 10.6 - No longer supported") and, decisively,
`runtime.c` has **no `#include`**: `ucontext_t` would have to be an over-allocated
buffer whose `uc_stack.ss_sp`, `ss_size` and `uc_link` are written **by
hard-coded byte offset**, and those offsets differ between darwin and linux.
A floor that guesses a struct layout is not a floor.

**B2, one pthread per generator with a strict handoff.** Every POSIX object here
is reached through a POINTER to an over-allocated zeroed buffer that its own
`*_init` fills in, so **no layout is needed anywhere** - the same source compiles
on darwin and on linux. And it is a direct transliteration of the Go half: two
alternating parties, exactly one running, no locking of runtime state.

The one thing neither can do by itself: **`c-to-llvm-ir.abnf` has no real function
pointers.** Measured rather than assumed -

```
$ mec languages/c-to-llvm-ir.abnf fp.c     # pthread_create(&t, 0, worker, 0)
	%5 = sext i32 1 to i64
	%6 = inttoptr i64 %5 to i32*
	%9 = call i64 @pthread_create(i32* %2, i32* %4, i32* %6, i32* %8)
```

`worker` became the **funcId 1**, not an address. `pthread_create` and
`makecontext` both need a real one. The fix costs no compiler change at all:
`dlopen(0, 2)` + `dlsym(h, "coro_entry")` answers the address at run time, and it
is portable (on darwin the main executable's symbols are exported by default; on
linux the clang line wants `-rdynamic`, one flag in `buildExecutable`).

**B2 ships in the proof of concept.** B1 is 6.3x faster and is the upgrade path;
see the numbers in section 3.

### Option C - a thread per generator with handoff synchronisation

This IS B2. It is listed separately in the brief; in this architecture the two
collapse, because a "second stack" obtained portably from C without headers is a
thread stack.

### Option D - eager draining, and Option E - replay

Both dead, and both worth naming so nobody re-derives them. Eager draining:
section 1's infinite sources. **Replay** - longjmp out of the body and re-run it
from the start on the next resume, memoising the yields - is O(n^2) and, worse,
re-executes side effects: `gen9` above would print `y1;y1;y2;` and the assertion
that pins interleaving is exactly the one that catches it.

### What each option does about the GC's conservative C-stack scan

This is a real interaction, and it is the part of option B that had to be built
rather than argued.

Commit `1cd6a41` scans **one** stack: `gc_collect` walks from its own frame to
`GC_STACK_BASE`, an anchor `main()` records. A suspended generator's stack holds
handles that live NOWHERE ELSE - the loop counter of an infinite generator, an
array built before a yield - so it is a root range exactly like the main stack,
and a collection triggered while a coroutine runs must scan the RESUMER's parked
stack for the same reason.

The design that works, and is what the PoC implements:

- **The running side owns `GC_STACK_BASE`.** Each side writes its own stack base
  the moment it gains control, so a collection triggered inside a generator body
  scans that body's stack and not `main`'s. (Break this and the PoC prints
  `<nil>` where it should print `42` - see the table in section 3.)
- **A registry of suspended stacks.** A coroutine publishes `sp` and a `setjmp`
  of its callee-saved registers immediately BEFORE parking, and the resumer does
  the same before handing off; `gc_roots` scans `[sp, base)` and the register
  buffer for each, exactly as it already does for the `JB` `jmp_buf` pool.
  Handoff is strictly alternating, so a parked side's stack is stable and NO
  stop-the-world machinery is needed - which is the same reason the Go half needs
  no locks.
- **No handle lives in the malloc'd control block.** Everything that travels
  between the two sides lives in the generator CELL, which is traced; the control
  block holds raw longs plus the back-reference to that cell.
- **Nothing moves**, so the conservative-roots-forbid-compaction property of 1b
  is unaffected.

Option A would need none of this - a state machine keeps its live values in a
heap frame the ordinary tracer already reaches. That is the strongest technical
argument for A and it is stated here rather than buried; it is outweighed, for
now, by five emitters against one floor.

## 3. The two-engine ratchet (was: the proof of concept)

```
tests/coro-poc/gen.ll          ONE hand-written module, run in BOTH engines
tests/coro-poc/coro-floor.py   copies languages/lib/runtime.c, and --check fails
                               loudly if any of the 14 coroutine pieces is gone
tests/coro-poc/build.sh        builds, runs both engines, diffs; --gc, --break, --diff
abnf/coropoc_test.go           the llvm.Run half (runJSModule, no grammar involved)
```

It is hand-written IR rather than a program in one of the five languages because
no grammar in the tree both EMITS `js_genfn` and has a layer 2 that links
natively today - php is the only one with a native layer 2 and its `js_genfn` is
the loud stub this investigation exists to remove. The shapes are exactly what
the five emitters already produce: `js_genfn` over a `js_closure`, `js_yield` in
the body, `js_get(g, "next")` for the cursor.

What it drives: **A** a finite generator, five `next()` calls, so the return
value lands in the `{value, done}` record and a `next()` past the end answers
`{undefined, true}`; **B** an INFINITE generator pulled five times with 3,000
allocations between resumes; **C** a handle reachable only from the suspended
body's frame, read back after 5,000 more allocations; **D** a body that allocates
4,000 times before its first yield, so the collection it triggers runs while the
RESUMER's stack is parked; **E** THROW ACROSS A YIELD - one raised in the body
and caught by a `try` in the BODY that was entered before a yield, one raised by
the body and caught by a `try` in the RESUMER, with 3,000 allocations between
every resume and a handle held only in the resumer's frame read back after the
whole thing unwinds.

### Ground truth

```
$ tests/coro-poc/build.sh
=== llvm.Run (abnf/jsrt.go: a goroutine and two channels) ===
1 false 2 false 3 false 99 true <nil> true          <- A (one value per line)
0 false 10 false 20 false 30 false 40 false         <- B, the infinite one
1 false 42 false 7 true                             <- C
1 false 1234                                        <- D
1 false 555 2 false 777 4321                        <- E, the throw gate
=== native (the C floor: a pthread and one condition variable), rc=0 ===
   ... the same 36 lines ...
===
coro PoC: BYTE-IDENTICAL in both engines (36 lines)
```

Read part E's line: `1 false` is the value yielded from inside the body's own
`try`; `555` is that try's CATCH clause printing what the body threw to ITSELF
across a yield; `2 false` is the yield after it; `777` is the resumer's catch
clause printing what the BODY threw out; `4321` is a handle that lived only in
the resumer's frame while all of that unwound.

Identical in all four collector modes - which is the gate, since the pre-fix
version printed the right answer under `auto` and the wrong one under `stress`:

```
--- the four collector modes, same binary ---
off      rc=0   SAME       gc: collections=0     live=0      heap=12583808
auto     rc=0   SAME       gc: collections=3     live=12976  heap=9438080
stress   rc=0   SAME       gc: collections=70286 live=12560  heap=9438080
poison   rc=0   SAME       gc: collections=3     live=12976  heap=12583808
```

### Discriminating power of each new GC root, measured

`tests/coro-poc/build.sh --break` breaks one thing at a time in a fresh copy of
the floor and re-runs under `MEC_GC=stress` (rc=1 means the binary died on its
own check; "0 of 36" is an honest zero):

```
a suspended coroutine stack not scanned          rc=1   DIFFERS on 14 lines
the parked registers not scanned                 rc=0   0 of 36
the RESUMER stacks not scanned                   rc=0   DIFFERS on 2 lines
the RESUMER registers not scanned                rc=0   0 of 36
the generator cell not a root                    rc=0   0 of 36
tag 15 children not traced                       rc=1   DIFFERS on 36 lines
tag 16 (the wrapped closure) not traced          rc=1   DIFFERS on 36 lines
GC_STACK_BASE not switched on resume             rc=1   DIFFERS on 9 lines
GC_STACK_BASE not switched back on yield         rc=1   DIFFERS on 5 lines
CUR_GEN not a root                               rc=0   0 of 36
a suspended coroutine's try barriers not scanned rc=1   DIFFERS on 5 lines
the parked resumer's try barriers not scanned    rc=0   0 of 36
ONE GLOBAL jmp_buf pool (the pre-fix floor)      rc=139 DIFFERS on 36 lines
the body's throw not re-raised on the resumer    rc=0   DIFFERS on 1 lines
```

The failure MODES are worth as much as the counts:

```
suspended coroutine stack not scanned   js runtime error: member '0' of undefined
RESUMER stacks not scanned              prints 1 where it should print 1234   <- SILENT
GC_STACK_BASE not switched on resume    prints <nil> where it should print 42 <- SILENT
ONE GLOBAL jmp_buf pool                 SIGSEGV - a longjmp into a dead frame
the throw not re-raised on the resumer  prints nothing where it should print 777
```

Two of the first three are **silently wrong answers**, which is the defect class
this project cares most about and the reason the PoC has parts C and D at all:
without part D the resumer-stack row was an honest zero and the root looked
optional. **Part E raised three rows off zero** that the four-part PoC could not
see: both `GC_STACK_BASE` switches and the resumer-stack row all got stronger,
because a throw is the only thing in the file that unwinds a coroutine frame.

The four honest zeros are stated, not hidden. The parked REGISTERS are zero
because clang happened to spill every live handle of these bodies to the stack -
that is a register allocator's habit, not a guarantee, and 1b already refused to
let a root set depend on one. `CUR_GEN` and the generator-cell root are zero
because this file never ABANDONS a generator; they exist for the program that
drops one while its thread is parked. The parked resumer's try barriers are zero
for the same reason as the parked registers: at the depth this file reaches, the
handles a resumer's `jmp_buf` holds are all also on its stack, which IS scanned.

### The price, stated rather than buried

200,000 resumes of an infinite generator, `/usr/bin/time -p`, best of three, every
run printing the right answer and exiting 0:

```
                                          real   user   sys     per next()
llvm.Run   goroutine + two channels        0.15   0.21   0.06     0.75 us
native     pthread + one condvar (ships)   0.82   0.18   0.65     4.1  us
```

(Re-measured 2026-08-03 against the LANDED floor: 0.82 real, best of three, and
the module printed 199999 and exited 0 each time. The per-resume cost did not
move when the throw barrier went in, which is what one `setjmp` per body START -
not per yield - should cost.)

**The native half is 6x SLOWER than the Go half here**, which is an inversion of
every other benchmark in this document, and all of it is `sys` - a condvar handoff
is two kernel round trips. The raw ceiling, measured on the same machine with
plain clang so the runtime is out of the way:

```
200,000 round trips   swapcontext      0.14 real   0.08 user   0.06 sys
                      pthread condvar  0.88 real   0.09 user   0.82 sys
```

So **B1 would be 6.3x faster than B2 and would match the Go half exactly**, and
option A would remove the switch altogether. Put in local terms: a generator
resume costs about four function calls today - part 1b's table puts native
`fib(26)` at 0.41 s over its ~393,000 calls, so about 1.0 us a call - and would
cost well under one under B1.

Memory, live suspended generators (each is a parked thread):

```
   200 generators     5.8 MB RSS        1,000    20.8 MB
 4,000 generators    76.5 MB RSS       10,000   187.8 MB     ~18.8 KB each
```

Linear, and 10,000 simultaneously suspended generators exits 0 (re-measured
2026-08-03 against the landed floor: `maximum resident set size` from
`/usr/bin/time -l`, each generator created, resumed once and kept in a live
array).

**The `CORO[256]` fixed array is gone.** It was the PoC's only hard limit and it
would have been a `die("too many live generators")` at the 257th - the 10,000
measurement above cannot even be TAKEN with it. `CORO` and the five `RES_*`
arrays are now malloc'd and doubled (`coro_grow`), starting at 64. The old block
is not freed, which is what every other allocation in this file does too, so the
total abandoned is under one copy of the final array.

A generator the program abandons keeps its thread parked forever - **the same
cost `abnf/jsrt.go` already documents for its parked goroutine** ("costs one
blocked goroutine and is collected when the program ends").

### What the floor gained (applied, not a patch any more)

`tests/coro-poc/build.sh --diff` prints nothing; `coro-floor.py --check` lists
the 14 pieces and fails if one of them ever leaves `languages/lib/runtime.c`:

```
 1  libc prototypes   dlopen dlsym pthread_{create,mutex_init,cond_init,
                      mutex_lock,mutex_unlock,cond_wait,cond_broadcast}
 2  globals           CORO/CORO_N/CORO_CAP, RES_LO/RES_HI/RES_JB/RES_JBP/
                      RES_JBD/RES_N/RES_CAP, CUR_GEN, CORO_ENTRY
 3  gc_trace          tag 15 (generator: a fn, b args, d lastValue, f sent/ret/
                      thrown are handles; c is a RAW block pointer, e a raw
                      flag) and tag 16 (generator function: a is the closure)
 4  gc_roots          every suspended coroutine stack + its parked registers +
                      its parked try barriers, every parked resumer stack +
                      registers + try barriers, CUR_GEN
 5  is_callable       tag 16
 6  get_member        a generator's `next`  -> mk_bound(g, 60)
 7  js_call           tag 16 -> gen_create;  and the coroutine block itself:
                      coro_grow, coro_alloc, gen_create, js_genfn, coro_entry,
                      gen_resume, js_yield, gen_next
 8  builtin_method    mid 60 -> gen_next
 9  the JB pool       JB[512] -> JBP/JB_CAP/JB_MAIN, one pool per STACK, swapped
                      by whichever side gains control (see section 5)
```

**`abnf/jsrt.go` needs no change and no emitter needs a change.** That is the
result: the module is one file and both engines run it.

### The concrete lines a layer 2 needs to drive one

`get_member` answers `next` on a generator, so a `-rt-lib` MetaJS file drives one
with ordinary member syntax. These are the whole of the primitive's surface:

```js
// The generator arrives from the PROGRAM module: the target-language emitter
// already emits js_genfn over a js_closure, and CALLING that answers a tag-15
// cell. Layer 2 never creates one - it only pulls.
var r = g.next(undefined)          // start / advance;  r = {value, done}
var r = g.next(v)                  // the value travels BACK IN (js, python)
if (r.done) { ret = r.value }      // the body's RETURN value lands in .value
                                   // a next() past the end answers {undefined, true}
```

Per language, on top of that one call:

```
php       foreach ($g as $v)  ->  var r = g.next(undefined); while (!r.done) { ...; r = g.next(undefined) }
          current/key/valid   ->  keep the last r in a one-slot lookahead; valid = !r.done
          keyed yield         ->  r.value is the {__genkv:true, k, v} record abnf/jsrt.go:530 defines
          getReturn()         ->  the r.value of the step whose r.done was true
js/ts     it.next(v)          ->  g.next(v) unchanged; js_iterable stops draining and
                                  becomes the while loop above (which also un-hangs
                                  a for-of over an infinite source)
python    e.send(v)           ->  g.next(v);  a done step raises StopIteration(r.value)
          close()             ->  drop the reference; the thread stays parked, as in Go
csharp    MoveNext()          ->  r = g.next(undefined); return !r.done
          Current             ->  the r.value of the last MoveNext
          yield break         ->  an ordinary return from the body; r.done becomes true
```

**A throw is now part of that surface.** `g.next()` raises on the caller's stack
whatever the body threw, so a layer-2 `foreach`/`MoveNext` needs no special case:
an exception out of a generator body is an exception out of `next()`, which is
what all four languages specify.

## 4. What it costs to do this for all four (five with csharp)

The floor addition above is **language-neutral and paid once**. Per language,
what remains is layer 2, and it is smaller than it looks because `get_member`
answers `next` on a generator, so **a `-rt-lib` MetaJS file can drive one with
ordinary member syntax**: `var r = g.next(v); if (r.done) { ... }`. Everything
else in the generator protocol is layer 2 over that single primitive.

```
language     what layer 2 owes                                        estimate
php          DONE 2026-08-03, and it came in at ~120 lines rather than    ~80 lines
             80. The two loud stubs were deleted (measured as the hard
             `ld: duplicate symbol '_js_genfn'` the section above
             describes); foreach-over-generator was ALREADY a next()
             loop in the emitter and needed nothing; the __genkv split
             is in the emitter too. What layer 2 actually owed was the
             INSPECTION half - and the extra 40 lines are the two
             findings it produced: the cursor cannot be stored on a
             generator cell (an identity-keyed SIDE TABLE, which every
             language with current/valid/MoveNext will need) and layer
             2 has no tag test, so recognising a generator is a
             STRUCTURAL probe. See the PHP section of part 3.
js / ts      it.next(v) is already the floor's shape; js_iterable     ~30 lines
             becomes a next() loop instead of a drain
python       send(v) = next(v) plus StopIteration carrying the body's ~50 lines
             return value; close(); the generator-expression lowering
             already goes through js_genfn
csharp       MoveNext/Current over next(); yield break               ~40 lines
ruby         nothing                                                      0
```

**Only PHP needs something the others do not**: the INSPECTION half
(`current`/`key`/`valid`/`getReturn`, `tests/php-test-full.php:466-470`) and
`yield $k => $v` keyed yields. Both are layer 2 over `next()` - a one-slot
lookahead and an unpack of the `__genkv` record `abnf/jsrt.go:530` already
defines - so neither is another floor primitive. **Only js and python need the
value to travel back in**, and the floor's `next(v)` already does that (the PoC's
`gen_next` passes `arg_at(args, 0)` straight into the suspended `js_yield`).

So the honest total is: **one floor change of ~260 lines, then 30-80 lines of
MetaJS per language.** That is a small fraction of the per-language extern work
already costed in Part 3, and WALL 1 stops being the thing that makes four
migrations wait for each other.

## 5. The one thing that worried me - CLOSED 2026-08-03

### What it was

**A `throw` that crossed a yield was undefined behaviour.** `js_throw` `longjmp`ed
to a `jmp_buf` in the `JB` pool that was
`setjmp`'d on the RESUMER's stack; from a coroutine thread that is a longjmp into
another thread's frame. It appears to work:

```
$ ./thr2.out                    # a generator body throws, jsmain catches
caught!
after
rc=0
```

and that is the trap - it "works" only because the resumer thread stays parked in
`pthread_cond_wait` forever while the coroutine thread runs on its stack. Turn the
collector up and it stops working:

```
$ MEC_GC=auto   ./thr2.out      caught! / after            rc=0
$ MEC_GC=stress ./thr2.out      js runtime error: call of a non function value: undefined
                                                            rc=1
```

### What shipped

Exactly what was named, and what the Go half already does. Two halves, both in
`languages/lib/runtime.c`:

**1. The `JB` pool became per STACK.** `long JB[512]` is now `JBP` (a pointer to
the pool of whichever stack is RUNNING), `JB_CAP` (its length) and `JB_MAIN`
(main's own, malloc'd in `boot`). A coroutine's pool is `cb[9]`, 128 deep, with
`cb[10]` the depth it parked at; the side that gains control swaps `JBP`, exactly
as it already swapped `GC_STACK_BASE`. This is what makes a `try` INSIDE a
generator body legal at all - the estimate said "the only part of the existing
floor this touches", and that held: four call sites (`jb_at`, `js_throw`,
`gc_roots`, `boot`).

**2. `coro_entry` sets a barrier.** `barrier = jb_at(0); JB_DEPTH = 1;` and the
body runs under `setjmp(barrier)`. A throw the body does not catch itself unwinds
only to there; `sf(g, THROWN)` puts the value in the traced generator cell and
`cb[11]` marks it; `gen_resume`, after it has restored `RES_N`, `GC_STACK_BASE`,
`CUR_GEN`, `JBP`, `JB_CAP` and `JB_DEPTH`, calls `js_throw(ff(g))` **on the
resumer's own stack**. That is `recover()` / re-`panic` transliterated.

The estimate was ~40 lines and was told to be treated with suspicion. Measured:
about 55 lines of new code plus the four rewritten call sites - close enough that
the suspicion was not needed, and the reason is that the DESIGN was already
right; only the writing was missing.

### The evidence

`tests/coro-poc/gen.ll` part E throws from inside a generator body twice - once
caught by a `try` in the body that was entered before a yield, once caught by a
`try` in the resumer - with 3,000 allocations between every resume. Both engines
print the same 36 lines, and the four collector modes are `SAME` with `rc=0`
(section 3). Under the PRE-FIX shape the same file segfaults:

```
ONE GLOBAL jmp_buf pool (the pre-fix floor)   rc=139  DIFFERS on 36 lines
the body's throw not re-raised on the resumer rc=0    DIFFERS on 1 lines
```

The exception machinery the pool refactor touched has its own ratchet in
`tests/metajs-test-full.js` SECTION 14 (`exc13`..`exc17`: a 200-deep nest thrown
through and caught, twice in a row so the depth must be restored, 50 sequential
try/catch/finally at one depth, and two throws whose value is only reachable
through the machinery while 3,000 allocations run). Measured against a
deliberately broken pool, `MEC_GC=stress`, native binary:

```
js_throw off by one in the new indirection    rc=139  (SIGSEGV, no output)
the jmp_buf pool not scanned as a root        rc=1    call of a non function value
jb_at reads main's pool instead of JBP        rc=0    476 checks, 0 failures  <- honest zero
main's pool only 8 deep (the nest overflows)  rc=1    try nested too deeply
```

The third row is an honest zero and says something true: a MetaJS program has one
stack, so `JBP` and `JB_MAIN` are the same pool there. Only a coroutine can tell
them apart, and that row is the `ONE GLOBAL jmp_buf pool` line of the `--break`
table above, where it is a segfault. Against a clean `git archive d30629f`, all
five assertions pass in all four collector modes - **`exc13`..`exc17` have zero
discriminating power against the pre-coroutine floor**, which they should: they
are a ratchet on the refactor, not a test for a feature d30629f lacks.

### Not done, and why - a generator ratchet in a LANGUAGE file

`tests/metajs-test-full.js` was the obvious home for a `throw` across a `yield`
driven from a real program rather than hand-written IR. **It cannot go there**,
and the reason is structural rather than effort:

- MetaJS has no `function*` or `yield` syntax, and `metajs-to-llvm-ir.abnf` emits
  no `js_genfn` (`grep -c 'yield' languages/metajs-to-llvm-ir.abnf` = 0). A
  generator would have to arrive as a HOST BUILTIN, the way `keysOf` did in
  d30629f - which needs no emitter change, only a `seed_root`.
- But `./test.sh --full` runs that file through BOTH halves, and the other half
  is `languages/metajs-interpreter.abnf`, a TREE-WALKING interpreter whose
  `:script` runs under goja AND under the frozen engine. Suspending a
  MetaJS-level call there means suspending the interpreter's own eval recursion,
  i.e. a goja call stack. goja cannot be suspended mid-call and resumed from
  another goroutine, so the interpreter half would have to implement generators
  by eager draining or by replay - **both of which section 2 killed**, and the
  half would then disagree with the compiler half's assertion count.

So the honest statement is: the floor primitive is ratcheted by `gen.ll` in both
engines (the two engines are the point), the exception machinery it rewrote is
ratcheted in `metajs-test-full.js`, and the first LANGUAGE-level generator
ratchet belongs to whichever of php / js / ts / python / csharp migrates first -
their `-test-full` files already contain the assertions, quoted in section 1.

> **CSHARP MIGRATED FIRST, 2026-08-03, and that ratchet now exists.** SECTION 19
> of `tests/csharp-test-full.cs` runs in a NATIVE BINARY byte-identically with
> `llvm.Run` - including `itr5`/`itr6`, the endless iterator a `foreach` leaves
> early, which is the assertion an eager drain cannot pass. **Layer 2 cost: ZERO
> LINES**, against the ~40 estimated in section 4's table: `csharp-to-llvm-ir.abnf`
> already lowered `MoveNext`/`Current`/`yield break` in the EMITTER over
> `js_get(g, "next")` + `js_call`, so every piece was already a floor extern.
> The probe additionally pins a `throw` out of a generator body caught around the
> `foreach`, which needed no case at all - `gen_resume` re-raises on the resumer's
> stack, exactly as section 5 designed it.
>
> The one cost that WAS real is not in this file: the floor's `type_of` answers
> `"object"` for a tag 16 cell where `abnf/jsrt.go` answers `"function"`. See the
> csharp section of Part 3 - it is two characters in `type_of` and two in
> `type_class`, and js / typescript / python will all hit it.

Two smaller things, recorded at the same standard:

- **The floor implements `next` only.** `current`/`key`/`valid`/`send`/
  `getReturn`/`return` are layer 2 over it, spelled out at the end of section 3,
  but that spelling is not yet executed by any language.
- **`-rdynamic`.** `dlsym` finds `coro_entry` without any link flag on darwin.
  On linux the executable needs `-rdynamic`, i.e. one word in
  `abnf/llvmlink.go`'s clang invocation. Not added, because nothing here builds
  on linux to prove it. **This is the one thing that will break the first linux
  build of a native binary that uses a generator.**
- **B1 (`swapcontext`) was NOT taken.** The measurement stands unchanged: 6.3x
  faster, and rejected because `runtime.c` has no `#include` and `ucontext_t`'s
  field offsets differ per platform. Correctness came first; the resume cost did
  not move when the barrier went in (4.1 us, section 3), so nothing about the
  throw fix makes B1 harder later.

---

# The four owed floor primitives - 2026-08-04. TWO LANDED, ONE WAS ALREADY DONE,
# ONE IS BLOCKED BY A FILE THIS TASK DID NOT OWN

Three migration reports asked the floor for a scope API, one asked for a generator
tag, one for `js_bytelen`, and go's report reported a `floStr` defect. All four are
settled below, each with its measurement.

```
matrix 325/325 · --full 5,711 assertions (was 5,686), 0 halves disagree · --cross 119/0
clang-check 16/16, all sixteen "ok, and the clang executable agrees", none held
go test ./abnf/ ok · all fourteen gen-*.sh --check clean · -freeze a fixed point
bench-alloc 3,711 B/iter at 400k (unchanged) · gc rss flat at 3.3 MB over 250k/1M/2M
coro-poc, --gc and --break byte-identical to ad922a0
```

Measured from a clean archive of `ad922a0` plus ONLY the five files this task owns
(`languages/lib/runtime.c`, `languages/lib/runtime.ll`, `abnf/jsrtint.go`,
`languages/metajs-interpreter.abnf`, `tests/metajs-test-full.js`) - the working tree
had three other agents in it at the time, and one of them had `php-rt.ll` mid-flight.

## 1. The scope API - LANDED. Host ids 64..68, in all three engines

`scopeNew` / `scopeParent` / `scopeGet` / `scopeHas` / `scopeDecl`, seeded by
`seed_root` in `languages/lib/runtime.c`, bound by `scopeBindings` in
`abnf/jsrtint.go` (called from `programJSBindings`, next to `giBindings`,
`jfBindings` and `keysBindings`), and implemented as `scNew`/`scParent`/`scGet`/
`scHas`/`scDecl` in `languages/metajs-interpreter.abnf`. **All three engines from
the start**, which is the gap `f19a8ad` shipped for `sint` and `8c396c4` had to
repair.

### The exact lines a layer-2 file writes

This is the whole point of the API, so it is written out rather than described.
Every one of the six lowered emitter probes is one of these four shapes:

```js
// "is `name` bound in THIS scope" - python's js_scope_has(s, n), the one thing
// js_scope_typeof cannot answer (it says "undefined" for an absent name AND for
// a slot that holds undefined).
function js_x_bound_here(s, name) { return scopeHas(s, name) }

// "is `name` bound ANYWHERE up the chain" - ruby's defined?, php's isset, swift's
// and dart's scope probe, kotlin's nine helpers. scopeParent in a loop IS the
// chain walk, and it terminates on undefined, never on a root scope.
function js_x_bound_anywhere(s, name) {
    var cur = s
    while (cur !== undefined) {
        if (scopeHas(cur, name)) { return true }
        cur = scopeParent(cur)
    }
    return false
}

// "read it if it is there" - scopeGet ABORTS on a name that is nowhere, exactly
// as js_scope_get does, so the guard is the walk above.
function js_x_read(s, name) {
    if (js_x_bound_anywhere(s, name)) { return scopeGet(s, name) }
    return undefined
}

// "bind here, overwriting a binding of THIS scope" - scope_put, not
// js_scope_set: it never reaches an enclosing binding of the same name. This is
// python's py_setvar residue, the case "bound strictly below a binding boundary
// AND above it", which needed exactly the containment test above.
function js_x_bind_here(s, name, v) { scopeDecl(s, name, v); return v }
```

Verified to compile: those four functions in a scratch `.metajs` under
`metajs-to-llvm-ir.abnf -q -rt-lib` emit `@str "scopeHas" / "scopeParent" /
"scopeGet" / "scopeDecl"` and four `define i64 @js_probe_*`, i.e. the lazy-boot
`js_scope_get(jsrtlib_env, "scopeHas")` lookup that `byteLen` and `keysOf` already
use. **There is no `scopeSet`**: "walk with `scopeHas`/`scopeParent`, then
`scopeDecl`" is it, and `js_scope_set` stays the emitter's.

**No language's emitter was converted.** That is the next step and it is another
file's; what is shipped here is the API, its three implementations and its ratchet.

### Two deliberate asymmetries with the `js_scope_*` externs

Both are what let the three halves agree at all, and both are asserted.

1. **An absent parent is `undefined`, not the root scope.** `js_scope_new(0)` means
   "the global scope" because an EMITTER writes the handle 0; a host builtin's
   arguments are VALUES, so that handle never arrives - `scopeNew(0)` would hand
   over a tag 3 number cell. More decisive: a chain that ran into the host globals
   has no twin in `metajs-interpreter.abnf`, whose host globals are a plain object
   and not a scope at all. So `scopeNew()` / `scopeNew(null)` build a chain root and
   `scopeParent` of it is `undefined` in all three engines.
2. **The scope argument is TYPE CHECKED, not coerced** (`host_scope` in the floor,
   `scopeOfArg` in the twin, `scOf` in the interpreter). A caller's mistake is the
   same abort everywhere instead of a silent `G_ROOT`.

The one thing a layer-2 file actually receives - an emitter's scope handle - passes
through unchanged: the floor's `scopeNew` is literally `mk_scope(...)`, the same
constructor `js_scope_new` calls, and the twin's is the same `&jsScope{}`.

### Why the interpreter half needs a box of its own

`metajs-interpreter.abnf`'s own chain is an ARRAY of plain objects holding `{v, t}`
type boxes, with no parent link, and its host globals are an object. It has no scope
to lend, so a scope handed to a program there is a box of its own -
`{__scn, __scv, __scp}` - exactly as the sized integer and the boxed double are.
The names are `__`-prefixed so `keysOf` skips them, which is as close as that half
can come to the other two, where `keysOf` refuses a scope outright.

### Discriminating power - MEASURED, not asserted

**Against a clean `ad922a0` archive**: the whole file ABORTS at `variable not
defined: scopeNew`, in the interpreter (goja) and under `llvm.Run` alike. `--full`
would drop SECTION 29 and SECTION 30 as unsupported: **25 of 25 new assertions
unreachable there.**

**By mutating one behaviour at a time**, each mutation applied alone and reverted,
in EACH of the three engines separately (`metajs-interpreter.abnf`, then
`abnf/jsrtint.go` with a rebuild, then `languages/lib/runtime.c` with a
`gen-runtime-ll.sh` and a fresh `-exe` binary):

| mutation | interpreter | Go twin | C floor |
|---|---|---|---|
| `scopeHas` walks the chain | sco04 sco12 sco17 | sco04 sco12 sco17 | sco04 sco12 sco17 |
| `scopeParent` answers the scope itself at the top | sco06 sco07 | sco06 sco07 | sco06 sco07 |
| `scopeGet` does NOT walk the chain | **ABORT** (`variable not defined: y`) | - | **ABORT** |
| `scopeDecl` always appends | sco08 | sco08 | - |
| `scopeNew` ignores its parent | **ABORT** | - | - |
| `isGenerator` = php-rt.metajs's guess | gen05 gen08 | - | - |
| `isGenerator` true for any object | - | gen04 gen05 gen08 | gen04 gen05 gen08 |

The failing SETS are identical across engines, which is the identity claim measured
rather than assumed. The three cells marked `-` were not run, not zeroes.

**GC**: the native ratchet passes under `MEC_GC=off`, `auto`, `stress` and `poison`
alike (546/546 each). A scope is tag 11 and `gc_trace` already traces
`w[1]` (the names/values/type-classes block) and `w[6]` (the parent).

## 2. `isGenerator`, host id 69 - LANDED. A PREDICATE, and NOT a general `js_tag`

PHP's generator support asks "is this a generator" structurally
(`php-rt.metajs`'s `phIsGen`: exclude `__dict`, `__refcell`, `__isclass`,
`__class` and `length`, then test whether `v["next"]` is callable) and its report
calls that "the one guess in this port". `isGenerator(v)` removes the guess: tag 15
in the floor, `*jsGenerator` in the twin.

**A numeric `js_tag(v)` was considered and rejected, and the reason is a fact about
the engines, not a preference.** `js_genfn` is a tag 16 CELL in `runtime.c` and a
`*hostFunc` in `abnf/jsrt.go`, so one number would be 16 in one half and 8 in the
other; and `metajs-interpreter.abnf` has NO tag numbering at all - its values are
the host engine's, where a closure, a host function and a bound method are one JS
function. A numeric tag therefore cannot be byte-identical in three engines, which
is the gate. Every OTHER distinction a tag would carry already has a name layer 2
can use: `typeof`, `sintIs`, `floIs`, and `typeof v.length == "number"` for an
array. A generator was the one shape with no name, so it got a predicate.

**An honest zero, stated as a zero.** MetaJS has no generators - no `function*`, and
`metajs-to-llvm-ir.abnf` emits no `js_genfn` - so **no MetaJS program can construct
one in ANY engine**, and the ratchet cannot exercise the TRUE arm at all. What
SECTION 30 pins is that all three halves answer false for every value a MetaJS
program can build, including the shape `phIsGen` gets wrong (`gen05`: an ordinary
object with a callable `next`). The true arm becomes reachable the moment a layer-2
file of a language that HAS generators calls it, which is where it will be used.

## 3. `floStr` style 1 - THE CLAIM IS STALE. Nothing to fix, and go-rt.metajs CAN use it

go's report says *"The floor's `floStr` routes style 1 (floGo) to `jvm_flo_str`,
i.e. Java's layout... One `if (sty == 1)` in `jvmFloText` fixes it."* **It does not,
and it did not when the report was written.** All three engines have routed style 1
to Go's layout since `d30629f`, which is two commits BEFORE go's own migration
(`3f91d7b`):

```
languages/lib/runtime.c   jf_text:      if (sty == 1) { return go_float_str(fa(h)); }
abnf/jsrtjvm.go           jvmFloText:   case floGo: return goFloStr(v.f)
metajs-interpreter.abnf   flStr:        if (v.sty == 1) return flGo(v.f)
```

verified at `10bbca1` with `git show`, i.e. at the tree go was written against.

Measured against **real go 1.26.5 darwin/arm64** (`fmt.Println` of a `float64`,
which is `%v` = `strconv.FormatFloat(f,'g',-1,64)`), over 26 values including both
`'g'` boundaries, both infinities, NaN, a negative zero, the subnormal edge and
`1/3`:

```
go       0 1 1.5 0.1 1e+20 1e+21 1e-05 0.0001 123456 1.234567e+06 1e+06 1e-07
         3.14159 -2.5 100 1e-06 0.3333333333333333 1.2345678901234567e+19
         2.5e-10 999999.5 NaN +Inf -Inf
goja     identical    -frozen  identical    llvm.Run  identical    native  identical
```

and the negative zero separately, where the three styles must differ:
`floStr(flo(-0.0, 1))` is `-0`, style 0 is `-0.0`, style 2 is `-0` - the same in all
four halves. **So nothing was changed in this file.** The stale claim now lives in
`go-rt.metajs`'s own comment above `goFloStr` ("the floor's floStr cannot be
used..."), which is that file's to correct.

**Could `go-rt.metajs` use `floStr` now?** Yes - `goFloStr` + `goDigits` +
`goLayout` (~85 lines) is a re-derivation of `go_float_str`, and both agree with
real go on every value probed above. Two things to check when it is done, neither a
blocker: `goFloStr` takes its digits from the ENGINE's own number text (`"" + a`)
while the floor uses `shortest_digits()`, so the shortest-digit source differs even
though the layout does not; and `floStr` is a HOST CALL (argument array + dispatch)
where `goFloStr` is a direct call, which matters only if float printing is hot.
**Reported, not made - `go-rt.metajs` is another file's.**

## 4. `js_bytelen` - QUALIFIES, BLOCKED, AND NOT SHIPPED

It is a pure byte count and carries no per-language behaviour, unlike `js_jadd`,
`js_mcall` and the `js_range_*` family that are going to the shared MetaJS layer for
exactly that reason. The floor already holds the bytes (a string cell IS its UTF-8
bytes plus a length in `fb`), so the body is one line, and it is written out in
`runtime.c` at the point where it would go.

**What blocks it is a link error, measured rather than predicted.**
`languages/lib/php-rt.metajs:1711` still spells
`function js_bytelen(s) { return byteLen(s) }`, which `-rt-lib` turns into a
`define i64 @js_bytelen` in `lib/php-rt.ll`. Both modules go to clang:

```
$ # clean archive of ad922a0, ONLY the js_bytelen body added to runtime.c
$ ./mec languages/php-to-llvm-ir.abnf tests/php-test-features.php -q -exe php.out
error: 1 unresolved symbol(s), and this build links a runtime, so they are NOT stubbed:
error:     js_bytelen
$ # what clang actually said, captured through MEC_CLANG:
duplicate symbol '_js_bytelen' in:
ld: 1 duplicate symbols
```

Note the diagnosis trap for whoever meets this again: the metacompiler reports a
DUPLICATE as *"unresolved"*, because `undefinedSymbols(m)` only checks whether clang
mentioned the name (`abnf/llvmmap.go`, `buildExecutable`).

So the two halves have to land together, and the layer-2 half is not this file's:
delete `php-rt.metajs`'s two lines, regenerate `lib/php-rt.ll` with
`tests/gen-php-rt-ll.sh`, and add the body. Shipping only the floor half fixes
nothing and breaks one language, which is a loss - so it was reverted, and the
argument, the body and the exact failure are recorded at the site instead.

**And the general rule this produced**: every extern a layer-2 file DEFINES is a
name the floor may not also define. Before moving any `js_*` into `runtime.c`,
`grep -l 'function js_<name>' languages/lib/*.metajs` first.

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

---

# The measured-but-unfixed defects, closed (2026-08-04)

Seven divergences that this document had already measured and recorded at their
sites, taken in one pass. Each is struck at its own site above with the full
argument; this is the index and the honest scoreboard.

| # | defect | oracle | state |
|---|---|---|---|
| 1 | Go `&^` emitted as a 32-bit pair; shift result type taken from the COUNT | real `go` (installed) | **FIXED**, both halves, pinned `int31`-`int38` |
| 2 | C# NaN sentinel read as an ordering (`NaN > 0.0` is TRUE) | ECMA-334 12.12.1 (no C# toolchain here) | **HALF FIXED** - Go twin done, **layer 2 owes one line** |
| 3 | C# `>>>` implemented in four halves, recognised by no grammar | ECMA-334 shift operators | **RECOGNISED**, both grammars, pinned `urs1`-`urs4` |
| 4 | Python `*args` overbinds when the positional count equals the slot count | `python3` 3.14.6 | **DIAGNOSED, NOT FIXED** - the bug is in `js_pycall`, which is two other agents' files |
| 5 | C `_Bool` holds the raw value | `cc` / clang 17 | **FIXED**, both halves, pinned `2008`-`2020` |
| 6 | a DUPLICATE symbol reported as an unresolved one | reproduced end to end with two `-rt` inputs | **FIXED**, pinned `abnf/linkdiag_test.go` |
| 7 | Dart `m[missing]` aborts instead of answering null | dart:core `Map.operator []` (no `dart` here) | **FIXED**, all three halves, pinned `co7`-`co10` |

Discriminating power, measured by copying each new ratchet file into a clean
`git archive ad922a0` tree and running it there:

```
go-test-full.go       4 red (interpreter)   6 red (compiler)
csharp-test-full.cs   PARSE ERROR in both halves at 926:40
c-test-full.c        11 red (interpreter)  12 red (compiler)
dart-test-full.dart   ABORTS in both halves
abnf/linkdiag_test.go does not COMPILE (duplicateSymbols did not exist)
```

**Three things another agent still owes, and none of them is optional:**

1. `js_cscmp` in `languages/lib/csharp-rt.metajs`: `if (c == 2) { return false }`
   after `var c = csCompare(l, r)`. Until it lands the two C# engines disagree on a
   NaN comparison and the ratchet assertion for it cannot be added (the text is
   written out at the C# site above).
2. ~~`js_pycall` in `abnf/jsrt.go` AND in `languages/lib/python-rt.metajs`~~
   **DONE 2026-08-04**: decided by IDENTITY (`pyBindCallR`'s `rebound` bool in the
   twin, `pos` returned by identity in layer 2), pinned by `sig9`-`sig16` in
   `tests/python-test-full.py` - 2 of 8 red by ABORT from a clean archive of
   `44bb09a`. Written out at the Python site above.
3. ~~The SHIFT result type in the floor~~ - **CLOSED 2026-08-04, and the claim it
   rested on was FALSE. See the section below.**

**One thing this pass changed that nobody asked for, and why.** The C return-type
fix (item 5) is broader than `_Bool`: the compiled half never converted a returned
value to a DECLARED return type narrower than an int, so `unsigned char f(int n)
{ return n; }` called with 300 answered 300 against `cc`'s and the interpreter
half's 44. It was in the same three lines and it is the same C11 clause family, so
it landed with them. It moved `languages/lib/batch-rt.ll` by four lines of sign
extension in one `char`-returning function, and that file was regenerated with
`tests/gen-batch-rt-ll.sh` as its own header instructs.

---

# The shift result type, closed in the FLOOR - and the claim was false (2026-08-04)

## What was owed, and what was actually true

The standing claim was: "the shift result type is the LEFT operand's alone, but
`js_giarith` reads width and signedness off whichever operand is BOXED. Go was
closed at the emitter; **Java, Kotlin, C#, Swift and Dart specify the same rule and
are still latently wrong, invisibly, because both of their halves agree.**"

**Three of those five are not wrong, one cannot be wrong, and the fifth was already
right.** The rule was fixed in the floor anyway, because the primitive should hold
it, but it is a GUARD and not a repair, and saying otherwise would be inventing a
defect that this session's own measurement disproves.

## There are FOUR sites, not three

The owed item named three. There is a fourth, and it is the one the MetaJS
interpreter runs on:

| site | file |
|---|---|
| `giArith` | `abnf/jsrtint.go` (the Go twin's `js_giarith`) |
| `si_apply` codes 9/10 | `languages/lib/runtime.c` (the C floor's `sintOp`) |
| `szArithSlow` | `languages/lib/interp-core.js` (the `*-interpreter.abnf` grammars) |
| **`siOp` codes 9/10** | **`languages/metajs-interpreter.abnf`** (the `sintOp` host builtin) |

All four now carry the same two lines, with the same comment:

```c
if (code == 9 || code == 10) { w = si_width_of(l, l); u = si_uns_of(l, l); }
```

## The measurement: a FATAL probe at all four sites, fired ZERO times

A probe was placed at each of the four sites that aborts when the `(w, u)` computed
from `(l, r)` differs from the `(w, u)` computed from `(l, l)` - i.e. exactly when
the old code and the new one would disagree. `runtime.ll` was regenerated so the
native binaries carried it too. It did not fire once:

```
matrix        325 entries run - 325 passed, 0 failed
--full        5,800 assertions in total; 0 languages with halves that disagree
--cross       119 programs compared, 0 divergent
clang-check   16 modules, all accepted by clang; all sixteen executables agree
```

The reason is that **every language's layer 2 normalises the count before it
reaches the floor**, which is what the code says when it is read rather than
assumed:

| language | where the count is normalised |
|---|---|
| Java | `jvShift` masks with `& 31` / `& 63` - a plain number |
| Kotlin | `k1Shift`: `k1I32(r) & mask` - a plain number, and `shl/shr/ushr` only take an `Int` anyway |
| C# | `csShift` -> `csShiftCount(r, w)`, and the left operand is always `csBox(l, w, un)`, so the left is the only box |
| Swift | `js_swshift` ends in `swGiArith(o, l, sintConv(n, 64, false))` - 64-bit signed, which IS the default |
| Dart | every Dart `int` is 64-bit signed, so a Dart box can only ever be `(64, signed)` |
| Go | the emitter wraps the count in `js_gival` (`go-to-llvm-ir.abnf`, `giShift`) |

## Ground truth: `java` 24.0.2 and `swiftc` 6.1.2, on the shapes those two can express

Kotlin, C# and Dart **cannot express a non-`int` shift count at all** - Kotlin's
`shl/shr/ushr` are declared over `Int`, ECMA-334 12.11 converts every count to
`int`, and Dart's `<<`/`>>` take an `int`. So there is nothing to measure there and
no toolchain here to measure it with; that rests on the specifications.

Java (JLS 15.19: the count is unary-promoted independently, the result type is the
promoted LEFT operand) and Swift (`static func >> <RHS: BinaryInteger>(lhs: Self,
rhs: RHS) -> Self`) both CAN, and both are installed. Both halves already answered
what the real toolchain answers, before the fix and after it:

```
java Sh.java                   both metacompiler halves
  int(-8)   >> byte(1)    -4              -4
  long(-1024) >> short(3) -128            -128
  int(1)    << long(40)   256             256      (Java masks 40 & 31)
  int(-1)   >>> char(1)   2147483647      2147483647
  long(-8)  >> int(1)     -4              -4
  int(0xF0) >> long(1)    120             120
  long(1)   << byte(40)   1099511627776   1099511627776

swiftc -O sh.swift             both metacompiler halves
  Int(-8)    >> UInt8(1)   -4             -4
  Int64(-1024) >> UInt16(3) -128          -128
  Int(1)     << UInt32(40) 1099511627776  1099511627776
  UInt8(240) >> Int(1)     120            120
  Int8(-8)   >> UInt64(1)  -4             -4
```

## Discriminating power: ZERO, and that was the predicted answer

No ratchet assertion was added, because there is nothing an assertion could
discriminate. Go's `int34`-`int38` in `tests/go-test-full.go` already pin the rule
end to end; they were red before the emitter fix in `9ee6fcc` and they pass with
the emitter fix, with the floor fix, and with both. An assertion for Java or Swift
would pass against a clean archive of `44bb09a` too - the seven Java lines and five
Swift lines above were run there in effect, since neither half's code path changed.
**Assertions that pass either way are decoration**; the probe result IS the
evidence, and it is written into `si_apply`'s comment where the next reader meets
it.

## What the fix is actually worth

The floor's primitive now holds the rule on its own, so the next language that
wires a shift straight to `sintOp` without a `csShift`-style wrapper gets it right
instead of getting a silent 8-bit unsigned result. It also makes Go's emitter fix
redundant rather than load-bearing - `js_gival` on the count is now belt and
braces, and it was left in place because removing it is a change with no measured
benefit.

---

# Part B, step 0: WHICH FLOOR BODIES ARE HOT - measured, not guessed

`runtime.c` has **301 bodies** (counted by the injector below, and it agrees with
the plan's figure exactly). Before moving any of them the question "which are hot"
was answered by instrumenting every single one.

## Method

`tests/`-adjacent, throwaway, run in a scratch tree: a script injects
`RTHIT[n]++;` as the first statement of all 301 top-level definitions (both the
`f(...) {`-on-its-own-line form and the one-line `f(...) { ...; }` form), adds a
`long RTHIT[]` table and a dumper that writes `index<TAB>count` to stderr under
`MEC_RTHIT=1`, from `main()` after `jsmain` returns AND from the `exit()` path so
that programs which end via `exit` are counted too. `runtime.ll` is regenerated,
and the native `-exe` binaries then carry exact per-body call counts. **Counts, not
samples** - a `sample(1)` profile was taken as well and agrees on the ordering, but
call counts also answer "was it called AT ALL", which is the question that makes a
move safe.

## The hot end - the two benches named in the plan

`tests/lua-bench-alloc.lua` shape at 200,000 iterations, plus
`tests/metajs-bench-try.js` at 20,000:

```
fa            148,052,694     bit_get        26,281,624     ar_alloc_k    10,694,999
fb            138,302,818     num_bits       22,126,085     ar_block      10,694,989
str_len       103,072,868     scope_get      17,449,657     bit_clr       10,666,295
str_eq         76,759,420     js_scope_get   17,449,657     chunk_of      10,659,919
tag_of         74,645,709     fc             16,493,830     sb             9,807,325
fd             53,468,528     ff             14,878,218     sa             9,647,966
str_ptr        51,215,230     to_number      12,885,823     sd             9,500,267
scope_find     46,754,913     mk_bool        12,354,360     gc_try         7,630,686
str_intern     29,963,334     truthy         12,240,077     strict_eq      6,971,462
scope_of       28,009,831     js_truthy      12,240,074     cell_new       6,801,205
js_str_mem     27,363,174     bit_set        11,759,491     sc             6,740,154
d_is_nan       26,793,540     d_is_inf       11,027,736     fe/scope_put   6,713,58x
```

Every one of these is a cell accessor, an interner, a scope operation, the
allocator or the collector. **None of them is a candidate and none of them was ever
proposed as one** - which is the reassuring half of the result.

## The candidates the plan named, measured - and the plan's list is WRONG in one place

```
                       lua bench    try bench    whole native corpus (34 programs)
d_pow                          0            0      69
d_exp                          0            0       4
d_log                          0            0       4
d_sqrt                         0            0      10
d_sqrt_d                       0            0       1
d_modf_int                     0            0      60
d_odd_int                      0            0       0
d_floor                        0            0   9,387
d_ceil                         0            0       6
d_frexp                        0      143,792   2,947        <-- HOT
d_ldexp                        0      123,793   2,219        <-- HOT
d_mod                          0       20,000     737        <-- HOT
d_mod_go                       0       20,000     741        <-- HOT
dec_mul                        7            0  35,491
dec_norm                       6            0   4,518
dec_to_double                  1            0      84
dec_cmp                        0            0   6,281
dec_absdiff                    0            0   2,047
fmt_top                        1            1      53
fmt_val                        0            1       0
fmt_apply                      0            0       6
fmt_sprint                     0            0       0
js_num_str                     1            0     306
case_map (+ the 328-range tables)
                               0            0     114
```

**`d_frexp`, `d_ldexp`, `d_mod` and `d_mod_go` are on the `%` hot path and the plan
groups them with the cold ones.** `metajs-bench-try.js` does `i % 3` once per
iteration; `d_mod` and `d_mod_go` are 20,000 calls each and each one costs about
7.2 `d_frexp`/`d_ldexp` pairs. Moving those four to MetaJS would put a
software-float remainder loop behind every `%` in every language. **They are not
candidates. Strike them from the list.**

## And a harder finding: `d_pow`/`d_exp`/`d_log`/`d_sqrt`/`d_frexp`/`d_ldexp`/`d_modf_int` CANNOT MOVE AT ALL

Not "should not" - cannot. They are not helpers written on top of doubles; they
**are** the double. Each one takes or returns a raw `double` or a raw bit pattern
(`long xb`), and each reads and writes the IEEE-754 layout directly through
`union DB` - `u.l = u.l & (0 - (1L << (52 - e)))` in `d_modf_int`,
`((u.l >> 52) & 2047) - 1023` for the exponent, `uy.l ^ INT64_MIN` for a sign flip
in `d_pow`. MetaJS has no union, no access to a double's bits, and its own numbers
ARE these doubles: a MetaJS `pow` would be written on `*` and `/`, which the
compiled output lowers to `__mec_dmul` / `__mec_ddiv` **in this same file**. The
move would be circular. They belong with `setjmp`, the pthread block, the collector
and the libc surface on the "can never move" list, and that list is hereby extended
to say so.

`d_mod`, `d_mod_go`, `d_is_nan`, `d_is_inf`, `d_is_zero`, `d_sign`, `d_neg`,
`d_abs`, `d_from_long`, `d_to_long`, `d_trunc` and `d_floor` are the same layer and
the same argument, and `d_floor` is 9,387 corpus calls on top of it.

## What is left, and it is a much shorter list than the plan's

Genuinely cold, genuinely handle-level, genuinely expressible in MetaJS:

| body | corpus calls | note |
|---|---|---|
| `case_map` + `CASE_LO/HI/MODE/DU/DL` | 114 | 328 ranges, ~110 lines of data + a 20-line binary search. Pure table lookup. **The best first move.** |
| `fmt_apply` / `fmt_sprint` / `fmt_top` / `fmt_val` | 6 / 0 / 53 / 0 | format-string machinery, handle level throughout |
| `dec_*` (`dec_mul` 35,491, `dec_cmp` 6,281, `dec_norm` 4,518, `dec_absdiff` 2,047, `dec_to_double` 84) | | the decimal bignum, reached from `shortest_digits` on the number-to-string path. Handle level, but NOT cold in the corpus - measure per move |
| `js_num_str` | 306 | |

Note where they go: `languages/lib/runtime.metajs`, which is the merge agent's
file, so the text has to be handed over rather than edited in. And before moving
any `js_*` name in either direction, `grep -l 'function js_<name>'
languages/lib/*.metajs` first - see the `js_bytelen` rule below.

## Coverage, which is a finding of its own

| | bodies called | never called |
|---|---|---|
| the two benches | 175 / 301 | 126 |
| the whole native corpus (34 `-exe` builds of `tests/*-test-features.*` + `tests/*-test-full.*`) | 277 / 301 | **24** |

The 24 that **no native test reaches at all**:

```
gc_grow  wr  wrn  werr  die  die2  die3  wstr  class_name  jf_is  fmt_sprint
d_odd_int  js_gicmp  js_gieq  js_ginot  js_ginum  js_gistr  js_giis  js_jfsub
js_jfmul  js_jfmod  js_jfneg  js_jfint
```

`wr`/`wrn`/`werr`/`wstr`/`die2`/`die3`/`class_name` are debug and error paths.
`d_odd_int` is only reachable through `d_pow` with a zero or infinite base.
**The eleven `js_gi*` / `js_jf*` externs are declared, emitted into every module,
and called by no test in the repository** - `js_gicmp`, `js_gieq`, `js_ginot`,
`js_ginum`, `js_gistr`, `js_giis`, `js_jfsub`, `js_jfmul`, `js_jfmod`, `js_jfneg`,
`js_jfint`. That is either dead surface to delete or a hole in the Go/Java/C#
ratchet files, and it should be settled before Part B moves anything, because a
body no test reaches is a body no move can be validated against.

## Gate readings, unchanged by everything in this section

```
MEC_GC=off  400,000 iters   3,711 B/iter     (was 3,711)
gc          400,000 iters   rss 3,309,568    (flat at 50k/100k/200k/400k)
lua native  200,000 iters   0.78s            (was 0.78s)
metajs try  200,000 iters   0.58s            (was 0.56s, run to run noise)
```

The instrumentation lives in the scratch tree only; nothing in `languages/lib/`
carries a counter.

---

# `js_bytelen` - LANDED, as a pair (2026-08-04)

The floor half is `long js_bytelen(long v) { return mk_num(d_from_long(str_len(
to_string(v)))); }` in `runtime.c`, and the layer-2 half is the DELETION of
`languages/lib/php-rt.metajs`'s `function js_bytelen(s) { return byteLen(s) }`
plus a regenerated `lib/php-rt.ll`. **Neither half is committable alone**, and both
directions were measured rather than predicted:

```
$ # floor half only, php-rt.metajs untouched
$ ./mec languages/php-to-llvm-ir.abnf tests/php-test-features.php -q -exe php.out
error:     js_bytelen
error: each name above is defined by two or more of the emitted module and the
error: linked inputs (c.runtime / -rt, -L, -l). Remove one definition; this is NOT
error: an unresolved symbol, and adding a definition would make it worse.
clang could not link the executable: 1 duplicate symbol(s), named on stderr

$ # both halves
$ ./mec languages/php-to-llvm-ir.abnf tests/php-test-features.php -q -exe php.out
php compiler: wrote executable php.out
$ ./php.out
features: 127 checks, 0 failures
$ ./test.sh -f php          14 entries run - 14 passed, 0 failed
$ tests/clang-check.sh      php: ok, and the clang executable agrees
```

Note that the diagnostic is now correct - it says DUPLICATE and says so in the
right words. That is item 6 of the closed-defects table doing its job; before
`abnf/linkdiag_test.go` the same failure read "1 unresolved symbol(s)" and sent the
last reader looking for a missing definition instead of an extra one.

**The general rule, restated because it will bite again:** every extern a layer-2
file DEFINES is a name the floor may not also define. Before moving any `js_*` body
into `runtime.c`, run `grep -l 'function js_<name>' languages/lib/*.metajs`.
