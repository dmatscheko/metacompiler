/*
 * runtime.c - the C floor of the self-hosted runtime (docs/runtime-rework-plan.md,
 * phase 3). It implements the js_* externs that metajs-to-llvm-ir.abnf declares,
 * so that a MetaJS program compiled to handle IR can be linked into a NATIVE
 * BINARY instead of being interpreted with the Go runtime in abnf/jsrt.go.
 *
 * IT IS COMPILED BY languages/c-to-llvm-ir.abnf, NOT BY CLANG.
 * That is the point of the phase: every layer of the stack is compiled by this
 * repository. The link input is languages/lib/runtime.ll, regenerated with
 *
 *     tests/gen-runtime-ll.sh
 *
 * which runs our own C compiler over this file and keeps the module. clang only
 * ever sees IR that came out of a grammar in languages/.
 *
 * WHAT IT IS NOT
 *   - It is not a MOVING or an incremental collector. Since 2026-08-03 it does
 *     collect: a stop-the-world mark/sweep over the heap, with the C stack
 *     scanned conservatively so no shadow stack has to be emitted (see "the heap
 *     and its GC" below, and docs/runtime-next-plan.md part 1b). Nothing is ever
 *     moved - conservative roots forbid it - so there is no compaction, and a
 *     chunk once malloc'd is never handed back to the OS.
 *   - It is not a full ECMAScript runtime. It implements the MetaJS subset that
 *     metajs-to-llvm-ir.abnf emits, matching abnf/jsrt.go where it can.
 *   - It is not Unicode-complete. Strings carry the UTF-16 code unit view that
 *     abnf/jsrt.go exposes (.length, charCodeAt, charAt, slice, substring,
 *     indexOf, split, and the WTF-8 half of an astral pair), but case mapping
 *     is ASCII-only where the Go twin uses strings.ToUpper.
 *   - Floating point is the SOFT FLOAT that c-to-llvm-ir.abnf emits, which
 *     truncates where IEEE-754 rounds to nearest even. Every exactly
 *     representable value agrees with the Go runtime bit for bit; a
 *     non-representable fraction can be one ulp low (0.1 + 0.2 is
 *     0.29999999999999991 here and 0.30000000000000004 there). Measured, with
 *     the whole table, in the phase 3 section of the plan.
 *
 * VALUE REPRESENTATION
 * A handle is a pointer to a Cell, carried in the i64 the emitted IR passes
 * around, and Cell.tag says what it is. The four singletons are the exception:
 * lib/compile-core.js emits the CONSTANTS 0 undefined, 1 null, 2 false, 3 true
 * straight into the IR, so they are never cells and every accessor answers for
 * them without dereferencing anything (see tag_of / fa).
 *
 * C SUBSET NOTES - constructs deliberately avoided because c-to-llvm-ir.abnf
 * does not handle them (each one measured; see the phase 3 section of the plan):
 *   - a `double` PARAMETER does not compile, and a `double` RETURN VALUE
 *     silently answers the wrong number. Every number therefore travels through
 *     this file as its raw IEEE-754 BIT PATTERN in a long, and a double is
 *     materialised only inside the function that computes with it, through
 *     `union DB`.
 *   - `*p = v` through a `long *` does not compile ("store operands are not
 *     compatible: src=i64; dst=i32*"). `p[0] = v` compiles and is used instead.
 *   - object-like #define is parsed but NOT expanded, so there are no macros.
 */

int putchar(int c);
int puts(const char *s);
long write(int fd, const char *buf, unsigned long n);
void *malloc(unsigned long n);
char *getenv(const char *name);
int setjmp(void *env);
void longjmp(void *env, int v);
void exit(int code);

/* ---- coroutines: the POSIX names the generator primitive stands on --------
 * Declared by hand like everything else here.  pthread_t, pthread_mutex_t,
 * pthread_cond_t and pthread_attr_t are all reached through a POINTER to an
 * OVER-ALLOCATED zeroed buffer that the matching *_init fills in, so this file
 * never needs to know their layout or size - which is what makes the same
 * source work on darwin and on linux.  makecontext/swapcontext were measured
 * first and rejected for exactly that reason: ucontext_t's uc_stack and uc_link
 * have to be written by OFFSET, and the offsets differ per platform.
 *
 * dlopen/dlsym exist for one reason: c-to-llvm-ir.abnf compiles a function NAME
 * used as a value to its funcId (measured - `pthread_create(&t, 0, worker, 0)`
 * emits `inttoptr i32 1`), so there is no way to spell a real code address in
 * this C subset.  dlsym gives one at run time, with no compiler change at all. */
void *dlopen(const char *path, int mode);
void *dlsym(void *h, const char *name);
long pthread_create(void *th, void *attr, void *fn, void *arg);
long pthread_mutex_init(void *m, void *a);
long pthread_cond_init(void *c, void *a);
long pthread_mutex_lock(void *m);
long pthread_mutex_unlock(void *m);
long pthread_cond_wait(void *c, void *m);
long pthread_cond_broadcast(void *c);

/* The compiled program. jsmain(env, args) is the module's entry point and
 * jsdispatch(idx, env, args) is the function table metajs-to-llvm-ir.abnf emits
 * for -exe: js_closure stores a raw function index, and this is how the runtime
 * turns that index back into a call. */
long jsmain(long env, long args);
long jsdispatch(long idx, long env, long args);

/* ---------------------------------------------------------------- doubles */

union DB { double d; long l; };

/* Forward declarations: the C subset needs a prototype before the call. */
long d_trunc(long x);
long d_add(long x, long y);
long d_sub(long x, long y);
long d_mul(long x, long y);
long d_div(long x, long y);
int  d_lt(long x, long y);
int  d_eq(long x, long y);
long d_abs(long x);
long d_mod_go(long x, long y);
int  d_is_nan(long x);
int  d_is_inf(long x);
int  d_is_zero(long x);

long DZERO;     /* bits of 0.0   */
long DONE;      /* bits of 1.0   */
long DNAN;      /* bits of NaN   */
long DINF;      /* bits of +Inf  */
long DNINF;     /* bits of -Inf  */

/* d_from_long / d_to_long / d_trunc are done with BIT ARITHMETIC rather than
 * through the soft float's i2d/d2i. Measured: the emitted i2d loses the low bits
 * above 2^53 (the literal 9007199254740993 came out as 4503599627370496, half
 * its value) and the d2i round trip made a large integral double look
 * non-integral. Bits are exact, and round-to-nearest-even here matches Go's
 * float64(int64). */
long d_from_long(long i) {
	long neg = 0;
	long u;
	long e = 0;
	long t;
	long m;
	long guard;
	long sticky;
	if (i == 0) { return 0; }
	/* INT64_MIN has no positive counterpart - `0 - i` wraps back to itself and
	 * the top-bit scan below then runs zero times and builds nonsense. It is
	 * exactly -2^63, so it is answered directly. Latent until the sized-integer
	 * tag made si_float pass a raw int64 in: found by the 97,664-line
	 * differential probe against abnf/jsrtint.go, which read -1. */
	if (i == (0 - 9223372036854775807 - 1)) { return -4332462841530417152; }   /* -2^63 */
	if (i < 0) { neg = 1; u = 0 - i; } else { u = i; }
	t = u;
	while (t > 1) { t = t >> 1; e = e + 1; }        /* e = index of the top bit */
	if (e <= 52) { m = u << (52 - e); }
	else {
		long sh = e - 52;
		long one = 1;
		m = u >> sh;
		guard = (u >> (sh - 1)) & 1;
		sticky = (sh >= 2) ? ((u & ((one << (sh - 1)) - 1)) != 0) : 0;
		if (guard && (sticky || (m & 1))) {
			m = m + 1;
			if (m > 9007199254740991) { m = m >> 1; e = e + 1; }
		}
	}
	return (neg << 63) | ((e + 1023) << 52) | (m & 4503599627370495);
}
long d_to_long(long x) {
	long e = ((x >> 52) & 2047) - 1023;
	long m;
	long v;
	if (e < 0) { return 0; }
	if (e > 62) { return x < 0 ? (0 - 9223372036854775807 - 1) : 9223372036854775807; }
	m = (x & 4503599627370495) | 4503599627370496;
	if (e >= 52) { v = m << (e - 52); } else { v = m >> (52 - e); }
	return x < 0 ? (0 - v) : v;
}

int d_is_nan(long x) {
	long e = (x >> 52) & 2047;
	long m = x & 4503599627370495;
	return e == 2047 && m != 0;
}
int d_is_inf(long x) {
	long e = (x >> 52) & 2047;
	long m = x & 4503599627370495;
	return e == 2047 && m == 0;
}
int d_is_zero(long x) { return (x & 9223372036854775807) == 0; }
int d_sign(long x)    { return x < 0; }   /* the raw sign bit, -0.0 included */

/* The four arithmetic operators on bit patterns. NaN and infinity are handled
 * here rather than in the soft float, which does not model them. */
long d_add(long x, long y) {
	union DB a; union DB b; union DB r;
	if (d_is_nan(x) || d_is_nan(y)) { return DNAN; }
	if (d_is_inf(x) || d_is_inf(y)) {
		if (d_is_inf(x) && d_is_inf(y) && d_sign(x) != d_sign(y)) { return DNAN; }
		return d_is_inf(x) ? x : y;
	}
	a.l = x; b.l = y; r.d = a.d + b.d; return r.l;
}
long d_neg(long x) { return x ^ (-9223372036854775807 - 1); }
long d_sub(long x, long y) { return d_add(x, d_neg(y)); }
long d_mul(long x, long y) {
	union DB a; union DB b; union DB r;
	if (d_is_nan(x) || d_is_nan(y)) { return DNAN; }
	if (d_is_inf(x) || d_is_inf(y)) {
		if (d_is_zero(x) || d_is_zero(y)) { return DNAN; }
		return (d_sign(x) != d_sign(y)) ? DNINF : DINF;
	}
	a.l = x; b.l = y; r.d = a.d * b.d; return r.l;
}
long d_div(long x, long y) {
	union DB a; union DB b; union DB r;
	if (d_is_nan(x) || d_is_nan(y)) { return DNAN; }
	if (d_is_inf(x) && d_is_inf(y)) { return DNAN; }
	if (d_is_inf(x)) { return (d_sign(x) != d_sign(y)) ? DNINF : DINF; }
	if (d_is_inf(y)) { return (d_sign(x) != d_sign(y)) ? (0 - 9223372036854775807 - 1) : 0; }
	if (d_is_zero(y)) {
		if (d_is_zero(x)) { return DNAN; }
		return (d_sign(x) != d_sign(y)) ? DNINF : DINF;
	}
	a.l = x; b.l = y; r.d = a.d / b.d; return r.l;
}
/* JS % is fmod: the remainder keeps the sign of the dividend. */
/* The truncated remainder. It USED to be x - trunc(x/y)*y, which is right only
 * while the quotient fits 53 bits: `10 % 0.1` answered 0 where the Go runtime
 * answers 0.09999999999999945, and 26 of a 90 case differential probe diverged
 * between llvm.Run and the native binary. fmod is EXACT for every pair of
 * doubles, so this is now a faithful port of Go's own math.Mod - the same
 * algorithm on the same Frexp/Ldexp this file already carries (see the note
 * above d_frexp: "close enough" is not an answer here). The body is defined
 * further down, after d_frexp/d_ldexp. */
long d_mod(long x, long y) { return d_mod_go(x, y); }

/* Ordering. Only ever called with two non-NaN operands. */
int d_lt(long x, long y) { union DB a; union DB b; a.l = x; b.l = y; return a.d < b.d; }
int d_eq(long x, long y) {
	union DB a; union DB b;
	if (d_is_nan(x) || d_is_nan(y)) { return 0; }
	if (d_is_zero(x) && d_is_zero(y)) { return 1; }   /* +0 == -0 */
	a.l = x; b.l = y; return a.d == b.d;
}

/* |x| < 2^63, i.e. the range in which (long)x is defined. */
int d_in_long(long x) {
	union DB a;
	union DB lim;
	if (d_is_nan(x) || d_is_inf(x)) { return 0; }
	a.l = x;
	lim.d = 9223372036854775807.0;
	if (a.d < 0) { a.d = 0 - a.d; }
	return a.d < lim.d;
}

long d_trunc(long x) {
	long e = ((x >> 52) & 2047) - 1023;
	long mask;
	long one = 1;                       /* a LONG one: `1 << 51` in int width is 0 */
	if (d_is_nan(x) || d_is_inf(x)) { return x; }
	if (e < 0) { return x & (0 - 9223372036854775807 - 1); }   /* |x| < 1 -> +-0 */
	if (e >= 52) { return x; }                                  /* already integral */
	mask = (one << (52 - e)) - 1;
	return x & (0 - 1 - mask);
}
long d_floor(long x) {
	long t;
	if (d_is_nan(x) || d_is_inf(x)) { return x; }
	t = d_trunc(x);
	if (d_lt(x, t)) { return d_sub(t, DONE); }
	return t;
}
long d_ceil(long x) {
	long t;
	if (d_is_nan(x) || d_is_inf(x)) { return x; }
	t = d_trunc(x);
	if (d_lt(t, x)) { return d_add(t, DONE); }
	return t;
}
int d_is_integral(long x) {
	if (d_is_nan(x) || d_is_inf(x)) { return 0; }
	return d_eq(x, d_trunc(x));
}
long d_abs(long x) { return x & 9223372036854775807; }

/* ToInt32, exactly like the JS abstract operation: truncate, then wrap modulo
 * 2^32 into the signed range. */
long to_int32(long bits) {
	long t; long v; long e; long m; long sh; long one = 1;
	if (d_is_nan(bits) || d_is_inf(bits)) { return 0; }
	t = d_trunc(bits);
	if (d_in_long(t) == 0) {
		/* |t| >= 2^63, and ToInt32 is a MODULO, not a range test. This used to
		 * answer 0, which is wrong: real JS says `1e20 | 0` is 1661992960 and
		 * `1e19 | 0` is -1981284352. Measured against node; recorded in
		 * docs/runtime-next-plan.md part 2 as 63 lines of the jsJFlo probe,
		 * where the Go twin's `int32(int64(f))` answered -1 on arm64 and this
		 * answered 0 - three implementations, three answers, none of them JS.
		 * abnf/jsrt.go's rt.toInt32 is fixed to match.
		 *
		 * t is an exact integer here, so the low 32 bits are exact too: the
		 * value is m * 2^(e-52) with m the 53 bit significand, so the shift
		 * carries the significand's low bits into place and everything at or
		 * above a shift of 32 is a multiple of 2^32 and contributes nothing.
		 * The significand is masked BEFORE the shift so the product never
		 * leaves 32 bits - `m << 31` would overflow. */
		e = ((t >> 52) & 2047) - 1023;
		sh = e - 52;
		if (sh >= 32) { return 0; }
		m = (t & 4503599627370495) | (one << 52);
		v = (m & ((one << (32 - sh)) - 1)) << sh;
		if (t < 0) { v = 0 - v; }
		v = v & 4294967295;
		if (v >= 2147483648) { v = v - 4294967296; }
		return v;
	}
	v = d_to_long(t);
	v = v & 4294967295;
	if (v >= 2147483648) { v = v - 4294967296; }
	return v;
}
long to_uint32(long bits) {
	long v = to_int32(bits);
	if (v < 0) { v = v + 4294967296; }
	return v;
}

/* -------------------------------------------------- the heap and its GC -- */
/*
 * THIS USED TO BE A BUMP ARENA THAT WAS NEVER FREED. It is now a MARK/SWEEP
 * heap (docs/runtime-next-plan.md, part 1b). The allocation per iteration of a
 * loop had been cut 5x by the emitter work of part 1a and was still exactly
 * LINEAR, because what remains is live data rather than waste: no further
 * reduction fixes a program that runs for a minute.
 *
 * WHY MARK/SWEEP AND NOT REFCOUNTING. A scope holds its parent and a closure
 * holds its defining scope, so a closure defined inside a function CYCLES with
 * the scope that names it - the common case, not an edge case.
 *
 * WHY THE C STACK IS SCANNED CONSERVATIVELY, AND NOT A SHADOW STACK IN THE
 * EMITTER. The plan's sequence said "shadow stack first". A shadow stack was
 * costed and rejected for a reason the plan itself states one paragraph later:
 * the emitted IR runs in TWO worlds - natively a handle is a Cell *, under
 * llvm.Run it is an index into a Go table - and the two must stay
 * byte-identical. Every push/pop a shadow stack emits is IR that the Go half
 * would have to carry and ignore, at every expression temporary, in fifteen
 * emitters. Scanning the real C stack needs NO emitted instruction at all, so
 * the compiler half is untouched by construction, which is the strongest form
 * of the invariant that matters most here ("a GC that changes an answer is
 * worse than no GC").
 *
 * What makes the scan sound is the ordinary ABI argument (this is what Boehm's
 * collector does): a handle live across a call is either spilled to the caller's
 * frame - which is inside the scanned range - or held in a callee-saved
 * register, which the collector captures with a setjmp of its own. Handles are
 * i64 INTEGERS as far as LLVM is concerned, never pointers, so no
 * pointer-provenance optimisation can rewrite one into a form the scan cannot
 * see. A collection can only start inside ar_alloc, so the invariant every
 * allocator here must keep is: between an ar_alloc and the store of its result
 * into a reachable object, do not allocate again. Checked at every site.
 *
 * INTERIOR POINTERS ARE ROOTS. The first version required an exact block
 * address and that was unsound at -O2 - see gc_try. Resolving one is exact and
 * constant time here: a block's address is base + idx * bsize, so the index is a
 * division.
 *
 * LAYOUT: SIZE-CLASS REGIONS WITH THE METADATA IN SIDE BITMAPS, AND NO PER-BLOCK
 * HEADER AT ALL. The first collector (1cd6a41) gave every allocation a 16 byte
 * header holding its size, two flags and a mark epoch, and that header cost
 * +25.4% of allocation - 4,557 bytes per iteration of the Lua benchmark against
 * 3,633 without a collector. It is gone: a CHUNK serves exactly one size class
 * and one kind, so the size and the kind are properties of the chunk, and the
 * "allocated" and "marked" flags are one bit each in two side bitmaps indexed by
 * the block's slot number. A handle IS the block address; there is nothing in
 * front of it.
 *
 *   chunk: [0] next chunk   [1] block size   [2] slot count   [3] slots bumped
 *          [4] alloc bitmap [5] mark bitmap  [6] kind (1 = cells)  [7] unused
 *          then 128 bytes of header padding, and the blocks
 *
 * A block over 1024 bytes gets a chunk of its own (one slot), so the scheme is
 * uniform and the sweep has no special case. Blocks are never moved, never
 * coalesced and never split, so a slot number means the same thing for ever.
 *
 * The mark bitmap is CLEARED at the start of a collection rather than carrying
 * an epoch, which is what the header used to buy: clearing is heap/bsize/8
 * bytes - 6 KB for a 3 MB heap of cells - against 16 bytes on every object.
 */

long AR_CHUNKS;      /* head of the chunk list */
long AR_PART[130];   /* the current bump chunk per partition = class * 2 + kind */
long GC_FREE[130];   /* free lists by partition; the next pointer is in the payload */
long GC_BIG;         /* free blocks over 1024 bytes: [0] next, [1] size, first fit */
long GC_ON;          /* 0 during boot and inside a collection */
long GC_MODE;        /* 0 auto, 1 never, 2 at every allocation, 3 = 2 + poison */
long GC_ALLOCED;
long GC_THRESH;
long GC_LIVE;
long GC_HEAP;
long GC_STACK_BASE;

/* ---- the coroutine registry, which is what the COLLECTOR needs ------------
 * A suspended generator's C stack holds handles that live nowhere else, so it
 * is a root range exactly like the main stack.  Handoff is strictly alternating
 * (one thread runs at a time), so a suspended side has always published its
 * stack pointer and a setjmp of its callee-saved registers BEFORE parking, and
 * the collector reads both without any stop-the-world machinery.
 *
 *   CORO[i]   a live coroutine's control block (see coro_alloc for the layout)
 *   RES_*     the RESUMER's stack while a coroutine runs.  Resumes nest (a
 *             generator may drive another), so this is a LIFO.  RES_JBP/RES_JBD
 *             are the resumer's try-barrier pool and depth, parked with it.
 *
 * Both are GROWN rather than fixed. A fixed CORO[256] was what the proof of
 * concept shipped, and it would have been a hard die() at the 257th live
 * generator - which is not a limit a language can be built on: 10,000 suspended
 * generators is a measured, working workload (~18 KB each, and the thread stack
 * is the whole of it). The old array is not freed, so doubling costs less than
 * one extra copy of the final array in total - the same bargain every other
 * malloc in this file makes. */
long CORO;           /* long *: the live coroutines' control blocks */
long CORO_N;
long CORO_CAP;
long RES_LO;         /* long *, five parallel arrays, one entry per nested resume */
long RES_HI;
long RES_JB;
long RES_JBP;
long RES_JBD;
long RES_N;
long RES_CAP;
long CUR_GEN;        /* the generator whose body is running, for js_yield */
long CORO_ENTRY;     /* &coro_entry, from dlsym; 0 until the first generator */

/* GEN_EXIT is the sentinel gen_close() throws INTO a suspended body so that the
 * `finally` clauses wrapping its yield run before the generator is abandoned -
 * JavaScript's return completion and CPython's GeneratorExit. It is one object,
 * created on the first close and never freed (gc_roots traces it), and no
 * program can name it: nothing in any of the sixteen languages hands out a
 * reference to it, so `pending != GEN_EXIT` in js_try is an exact test. The
 * twin is abnf/jsrt.go's genExit. 0 until the first close, which is why every
 * comparison against it is guarded by GEN_EXIT != 0 - handle 0 is undefined,
 * and `throw undefined` is legal. */
long GEN_EXIT;

long GC_REGS;        /* a jmp_buf used only to spill the callee-saved registers */
long GC_MSTACK;
long GC_MTOP;
long GC_MCAP;
long GC_COUNT;
long PINS[8192];     /* js_gc_pin: module globals that hold a handle for ever */
long PIN_N;

void gc_collect(void);

/* One bit per slot, in a malloc'd side array. */
void bit_set(long bm, long i) {
	char *b = (char *)bm;
	long one = 1;
	b[i >> 3] = (char)(((long)b[i >> 3] & 255) | (one << (i & 7)));
}
void bit_clr(long bm, long i) {
	char *b = (char *)bm;
	long one = 1;
	b[i >> 3] = (char)(((long)b[i >> 3] & 255) & (255 - (one << (i & 7))));
}
int bit_get(long bm, long i) {
	char *b = (char *)bm;
	return (int)(((((long)b[i >> 3]) & 255) >> (i & 7)) & 1);
}
long bm_new(long nblk) {
	long nb = (nblk + 7) / 8 + 1;
	long bm = (long)malloc(nb);
	char *b = (char *)bm;
	long i = 0;
	while (i < nb) { b[i] = 0; i = i + 1; }
	return bm;
}

/* A chunk of `nblk` blocks of `bsize` bytes each, all of one kind. */
long chunk_new(long bsize, long nblk, long kind) {
	long c = (long)malloc(bsize * nblk + 128);
	long *w = (long *)c;
	w[0] = AR_CHUNKS;
	w[1] = bsize;
	w[2] = nblk;
	w[3] = 0;
	w[4] = bm_new(nblk);
	w[5] = bm_new(nblk);
	w[6] = kind;
	w[7] = 0;
	AR_CHUNKS = c;
	GC_HEAP = GC_HEAP + bsize * nblk;
	return c;
}

/* Which chunk holds address v, or 0. The chunk list is walked linearly, as it
 * was before: a heap of a few megabytes is a handful of chunks. */
long chunk_of(long v) {
	long c = AR_CHUNKS;
	while (c != 0) {
		long *w = (long *)c;
		long off = v - c - 128;
		if (off >= 0 && off < w[1] * w[2]) { return c; }
		c = w[0];
	}
	return 0;
}

/* A block of exactly `need` bytes and kind `kind`: the partition's free list,
 * then a bump in the partition's current chunk, then a new chunk. A block over
 * 1024 bytes comes from the big free list or a chunk of its own. */
long ar_block(long need, long kind) {
	long cls = need >> 4;
	long b;
	if (cls > 64) {
		long prev = 0;
		b = GC_BIG;
		while (b != 0) {
			long *w = (long *)b;
			if (w[1] >= need) {
				long c = chunk_of(b);
				long *cw = (long *)c;
				if (prev == 0) { GC_BIG = w[0]; }
				else { long *pw = (long *)prev; pw[0] = w[0]; }
				bit_set(cw[4], 0);
				return b;
			}
			prev = b;
			{ long *nw = (long *)b; b = nw[0]; }
		}
		{
			long c = chunk_new(need, 1, kind);
			long *cw = (long *)c;
			cw[3] = 1;
			bit_set(cw[4], 0);
			return c + 128;
		}
	}
	{
		long part = cls * 2 + kind;
		long cur;
		b = GC_FREE[part];
		if (b != 0) {
			long c = chunk_of(b);
			long *cw = (long *)c;
			{ long *fw = (long *)b; GC_FREE[part] = fw[0]; }
			bit_set(cw[4], (b - c - 128) / need);
			return b;
		}
		cur = AR_PART[part];
		if (cur != 0) {
			long *cw = (long *)cur;
			if (cw[3] < cw[2]) {
				long idx = cw[3];
				cw[3] = idx + 1;
				bit_set(cw[4], idx);
				return cur + 128 + idx * need;
			}
		}
		{
			/* 1 MB of blocks per partition. The pages are only touched as the
			 * bump pointer reaches them, so an unused class costs address space
			 * and no resident memory. */
			long nblk = 1048576 / need;
			long c = chunk_new(need, nblk, kind);
			long *cw = (long *)c;
			AR_PART[part] = c;
			cw[3] = 1;
			bit_set(cw[4], 0);
			return c + 128;
		}
	}
}

/* kind 1 = the payload is a Cell (traced by its tag), 0 = a raw buffer. */
char *ar_alloc_k(long n, long kind) {
	long need = (n + 15) & (0 - 16);
	long b;
	if (need < 16) { need = 16; }
	if (GC_ON != 0) {
		if (GC_MODE == 2) { gc_collect(); }
		else if (GC_MODE != 1 && GC_ALLOCED > GC_THRESH) { gc_collect(); }
	}
	b = ar_block(need, kind);
	GC_ALLOCED = GC_ALLOCED + need;
	return (char *)b;
}
char *ar_alloc(long n) { return ar_alloc_k(n, 0); }

void ar_init(void) {
	long i = 0;
	AR_CHUNKS = 0; GC_BIG = 0; GC_ON = 0;
	GC_ALLOCED = 0; GC_LIVE = 0; GC_HEAP = 0; GC_COUNT = 0; PIN_N = 0;
	/* The same floor a collection leaves behind, so a short program still gets
	 * one collection instead of running the whole matrix without ever entering
	 * the collector. */
	GC_THRESH = 1048576;
	while (i < 130) { GC_FREE[i] = 0; AR_PART[i] = 0; i = i + 1; }
	GC_MSTACK = (long)malloc(65536 * 8);
	GC_MTOP = 0;
	GC_MCAP = 65536;
	GC_REGS = (long)malloc(512);
}

/* ---------------------------------------------------------------- values */

/* tags */
/* 0 undefined, 1 null, 2 boolean, 3 number, 4 string, 5 array, 6 object,
 * 7 closure, 8 host function, 9 bound method, 10 control signal, 11 scope */

struct Cell {
	long tag;
	long a;
	long b;
	long c;
	long d;
	long e;
	long f;
};

/* THE FOUR SINGLETONS ARE PINNED TO 0..3. lib/compile-core.js emits the constant
 * handles 0 undefined, 1 null, 2 false, 3 true directly into the IR (its
 * hUndef/hNull/hFalse/hTrue), so they can never be arena cells - every accessor
 * below answers for them without dereferencing anything. */
long H_UNDEF;
long H_NULL;
long H_FALSE;
long H_TRUE;

long cell_new(long tag) {
	struct Cell *p = (struct Cell *)ar_alloc_k(56, 1);
	p->tag = tag; p->a = 0; p->b = 0; p->c = 0; p->d = 0; p->e = 0; p->f = 0;
	return (long)p;
}
long tag_of(long h) {
	struct Cell *p;
	if (h < 4) { if (h == 0) { return 0; } if (h == 1) { return 1; } return 2; }
	p = (struct Cell *)h;
	return p->tag;
}
long fa(long h) {
	struct Cell *p;
	if (h < 4) { return h == 3 ? 1 : 0; }
	p = (struct Cell *)h;
	return p->a;
}
long fb(long h) { struct Cell *p = (struct Cell *)h; return p->b; }
long fc(long h) { struct Cell *p = (struct Cell *)h; return p->c; }
long fd(long h) { struct Cell *p = (struct Cell *)h; return p->d; }
long fe(long h) { struct Cell *p = (struct Cell *)h; return p->e; }
long ff(long h) { struct Cell *p = (struct Cell *)h; return p->f; }
void sa(long h, long v) { struct Cell *p = (struct Cell *)h; p->a = v; }
void sb(long h, long v) { struct Cell *p = (struct Cell *)h; p->b = v; }
void sc(long h, long v) { struct Cell *p = (struct Cell *)h; p->c = v; }
void sd(long h, long v) { struct Cell *p = (struct Cell *)h; p->d = v; }
void se(long h, long v) { struct Cell *p = (struct Cell *)h; p->e = v; }
void sf(long h, long v) { struct Cell *p = (struct Cell *)h; p->f = v; }

/* --- the collector ------------------------------------------------------ */
/*
 * Marking is PRECISE from a cell (the tag says which fields hold a handle and
 * which hold a raw long - a number's IEEE bits, a closure's function index, a
 * host function's id, a bound method's method id) and CONSERVATIVE from a raw
 * buffer that was reached from the C stack rather than from its owning cell.
 * Both directions can only ever RETAIN too much, never free something live.
 *
 * Every marking step goes through gc_try, which VALIDATES the candidate against
 * the heap before touching it. That is deliberately paid even for a field the
 * tag says is a handle: a `_raw` operator index reaching a traced slot by
 * mistake would otherwise write a mark word into unrelated memory, and a
 * collector whose failure mode is silent corruption is not worth having.
 */

void gc_try(long v, int deep);

void gc_grow(void) {
	long nc = GC_MCAP * 2;
	long ns = (long)malloc(nc * 8);
	long *o = (long *)GC_MSTACK;
	long *n = (long *)ns;
	long i = 0;
	while (i < GC_MTOP) { n[i] = o[i]; i = i + 1; }
	GC_MSTACK = ns;
	GC_MCAP = nc;
}

/* Mark slot `idx` of chunk `c`. The mark stack holds PAIRS - the block address
 * and its chunk - because the chunk is what says how big the block is and
 * whether it is a cell, and finding it again would be a second chunk walk. */
void gc_mark(long c, long idx, int deep) {
	long *cw = (long *)c;
	long *ms;
	if (bit_get(cw[4], idx) == 0) { return; }         /* not allocated: garbage */
	if (bit_get(cw[5], idx) != 0) { return; }         /* already marked */
	bit_set(cw[5], idx);
	GC_LIVE = GC_LIVE + cw[1];
	if (deep == 0) { return; }                        /* no children to look at */
	if (GC_MTOP + 2 > GC_MCAP) { gc_grow(); }
	ms = (long *)GC_MSTACK;
	ms[GC_MTOP] = c + 128 + idx * cw[1];
	ms[GC_MTOP + 1] = c;
	GC_MTOP = GC_MTOP + 2;
}

/* Does v point into a block, and if so, which one?
 *
 * INTERIOR POINTERS ARE ACCEPTED, and that is not conservatism for its own sake:
 * the first version required an EXACT payload address, on the argument that a
 * buffer's owning cell is always reachable too and marks the buffer precisely.
 * That argument is false at -O2, and the whole native lua suite said so. In
 * u_slice the only surviving reference during str_from_units' allocation is
 * `u + b`, an interior pointer into the UTF-16 unit buffer: the compiler had
 * already dropped the string handle `h` after reading fc(h) out of it, so the
 * cell was unreachable, the collector freed it, and the unit buffer went with
 * it. MEC_GC=stress turned that into 9 failures in tests/lua-test-features.lua
 * and 3 in tests/lua-test-complete.lua, all in string.sub. Requiring exact
 * pointers is a standing bet that no optimiser will ever keep only a derived
 * pointer, and that bet does not hold.
 *
 * Resolving one is now EXACT and needs no start bitmap: a chunk holds one size
 * class, so the slot is `(v - base) / bsize` and the block is that slot. */
void gc_try(long v, int deep) {
	long c;
	long *w;
	long off;
	if (v < 4096) { return; }
	c = AR_CHUNKS;
	while (c != 0) {
		w = (long *)c;
		off = v - c - 128;
		if (off >= 0 && off < w[1] * w[2]) {
			long idx = off / w[1];
			if (idx < w[3]) { gc_mark(c, idx, deep); }
			return;
		}
		c = w[0];
	}
}

void gc_trace(long b, long c) {
	long *cw = (long *)c;
	long *w = (long *)b;
	long t;
	if (cw[6] == 0) {
		/* A raw buffer reached from the C stack. Its shape is not known here
		 * (a scope's names/values/type-classes share one block, a string's
		 * unit array holds code points), so every word in it is a candidate. */
		long q = b;
		long end = b + cw[1];
		while (q + 8 <= end) { long *x = (long *)q; gc_try(x[0], 1); q = q + 8; }
		return;
	}
	/* w[0..6] IS the Cell: tag, a, b, c, d, e, f - the block address is the
	 * cell address now that there is no header in front of it. */
	t = w[0];
	if (t == 4)  { gc_try(w[1], 0); gc_try(w[3], 0); return; }  /* bytes, units */
	if (t == 5)  { gc_try(w[1], 1); return; }                   /* array elements */
	if (t == 6)  { gc_try(w[1], 1); gc_try(w[2], 1); return; }  /* keys, values */
	if (t == 7)  { gc_try(w[2], 1); return; }                   /* closure env */
	if (t == 9)  { gc_try(w[1], 1); return; }                   /* bound receiver */
	if (t == 10) { gc_try(w[2], 1); return; }                   /* signal value */
	if (t == 11) { gc_try(w[1], 1); gc_try(w[6], 1); return; }  /* names+vals, parent */
	/* 15 GENERATOR: a fn, b args, d lastValue, f sent/return/thrown value are
	 * handles; c is the RAW control-block pointer and e a raw flag word.  16
	 * generator FUNCTION: a is the closure it wraps. */
	if (t == 15) { gc_try(w[1], 1); gc_try(w[2], 1); gc_try(w[4], 1); gc_try(w[6], 1); return; }
	if (t == 16) { gc_try(w[1], 1); return; }
	/* 3 number, 8 host function, 12 anytype, 13 sized integer, 14 boxed double:
	 * no children - every field of those is a raw long, not a handle. */
}

void gc_drain(void) {
	while (GC_MTOP > 0) {
		long *ms = (long *)GC_MSTACK;
		GC_MTOP = GC_MTOP - 2;
		gc_trace(ms[GC_MTOP], ms[GC_MTOP + 1]);
	}
}

void gc_scan_range(long lo, long hi) {
	long p = (lo + 7) & (0 - 8);
	while (p < hi) {
		long *x = (long *)p;
		gc_try(x[0], 1);
		p = p + 8;
	}
}

/* The marks are CLEARED here rather than carried in a per-block epoch: it is one
 * pass over (slots bumped)/8 bytes per chunk, which is what the 16 byte header
 * used to be paying for. */
void gc_clear_marks(void) {
	long c = AR_CHUNKS;
	while (c != 0) {
		long *cw = (long *)c;
		long nb = (cw[3] + 7) / 8 + 1;
		char *b = (char *)cw[5];
		long i = 0;
		while (i < nb) { b[i] = 0; i = i + 1; }
		c = cw[0];
	}
}

void gc_sweep(void) {
	long c = AR_CHUNKS;
	while (c != 0) {
		long *cw = (long *)c;
		long n = cw[3];
		long bs = cw[1];
		long part = (bs >> 4) * 2 + cw[6];
		long i = 0;
		while (i < n) {
			if (bit_get(cw[4], i) != 0 && bit_get(cw[5], i) == 0) {
				long b = c + 128 + i * bs;
				long *pw = (long *)b;
				bit_clr(cw[4], i);
				if (GC_MODE == 3) {
					/* Poison mode: the block is RETIRED - its allocation bit stays
					 * clear and it goes on no free list, so it is never handed out
					 * again - and its payload is filled with a value that is neither
					 * a tag nor a handle, so a root the mark pass FAILED to reach
					 * shows up as a deterministic wrong answer instead of a rare
					 * use-after-free. */
					long q = 0;
					while (q * 8 < bs) { pw[q] = 195948557; q = q + 1; }
				} else if (bs > 1024) {
					pw[0] = GC_BIG; pw[1] = bs; GC_BIG = b;
				} else {
					pw[0] = GC_FREE[part]; GC_FREE[part] = b;
				}
			}
			i = i + 1;
		}
		c = cw[0];
	}
}

void gc_roots(void);          /* the globals of this file; defined once they exist */

void gc_collect(void) {
	long guard;
	long lo;
	long i;
	if (GC_ON == 0) { return; }
	GC_ON = 0;
	gc_clear_marks();
	GC_MTOP = 0;
	GC_LIVE = 0;
	gc_roots();
	/* The callee-saved registers. A handle live across a call is either spilled
	 * into a frame inside the scanned range or sitting in one of these. */
	setjmp((void *)GC_REGS);
	i = 0;
	while (i < 64) { long *r = (long *)GC_REGS; gc_try(r[i], 1); i = i + 1; }
	/* The C stack, from this frame up to the anchor main() took. The 256 byte
	 * margin below covers the compiler placing `guard` above other slots of this
	 * frame; it reads the red zone, which is mapped, and a false pointer there
	 * costs retention and nothing else. */
	guard = 0;
	lo = ((long)&guard) - 256;
	gc_scan_range(lo, GC_STACK_BASE);
	gc_drain();
	gc_sweep();
	GC_ALLOCED = 0;
	GC_THRESH = GC_LIVE;
	if (GC_THRESH < 1048576) { GC_THRESH = 1048576; }
	GC_COUNT = GC_COUNT + 1;
	GC_ON = 1;
}

/* --- fatal errors ------------------------------------------------------- */

/* A fatal runtime error goes to STDERR, like abnf/jsrt.go's rt.fail, and the
 * wording is that of the Go twin so the two engines report identically. */
void werr(const char *s) {
	long n = 0;
	while (s[n] != 0) { n = n + 1; }
	write(2, s, (unsigned long)n);
}
void die(const char *msg) {
	werr("js runtime error: "); werr(msg); werr("\n");
	exit(1);
}

/* --- numbers ------------------------------------------------------------ */

long num_bits(long h);      /* forward */

/* The small-integer cache, indexed by value + 256. It is shared with js_num_i,
 * which fills the same slots from the raw integer side; see the comment there
 * for why sharing a number cell is safe (a tag-3 cell is written once, in
 * mk_num, and every number comparison in this runtime is by BIT PATTERN). */
long NIC[1281];

/* 0..1024 if these are the bits of a small non-negative integral double, else
 * -1. Spelled out on the bit pattern rather than via d_is_integral/d_to_long
 * because mk_num is on the result path of EVERY arithmetic operation: a
 * non-integer pays one shift, one mask and one compare. +0.0 is bits == 0;
 * -0.0 has the sign bit and is deliberately NOT cached, because it must keep
 * printing as -0. */
long small_int_bits(long bits) {
	long e;
	long m;
	long sh;
	long one = 1;
	if (bits == 0) { return 0; }
	if (bits < 0) { return -1; }                 /* the sign bit */
	e = (bits >> 52) & 2047;
	if (e < 1023 || e > 1033) { return -1; }     /* outside [1, 2048) */
	sh = 1075 - e;
	m = (bits & 4503599627370495) | 4503599627370496;
	if ((m & ((one << sh) - 1)) != 0) { return -1; }
	m = m >> sh;
	if (m > 1024) { return -1; }
	return m;
}

long mk_num_raw(long bits) {
	long h = cell_new(3);
	sa(h, bits);
	return h;
}

long mk_num(long bits) {
	long v = small_int_bits(bits);
	long h;
	if (v < 0) { return mk_num_raw(bits); }
	h = NIC[v + 256];
	if (h != 0) { return h; }
	h = mk_num_raw(bits);
	NIC[v + 256] = h;
	return h;
}
long num_bits(long h) { return fa(h); }

/* --- strings ------------------------------------------------------------ */

/* A string cell holds a byte pointer and a byte length; the buffer always
 * carries a terminating NUL as well, so it can be handed to puts. */
long mk_str(const char *p, long n) {
	long h = cell_new(4);
	char *buf = ar_alloc(n + 1);
	long i = 0;
	while (i < n) { buf[i] = p[i]; i = i + 1; }
	buf[n] = 0;
	sa(h, (long)buf);
	sb(h, n);
	return h;
}
/* js_str_mem is called once per evaluation of every string literal, so the
 * (pointer, length) pair is cached - the same thing abnf/jsrt.go's strMemCache
 * does.
 *
 * THIS USED TO BE DIRECT-MAPPED ON `p >> 3`, AND IT THRASHED. String literals
 * are laid out CONTIGUOUSLY in the module, most of them a handful of bytes
 * long, so `p >> 3` maps every literal inside an eight-byte window onto the
 * same slot: a module with 236 distinct literals collided its way to 60 MISSES
 * PER ITERATION of a `s = s + i %% 7` Lua loop (182.7 calls, 600,133 misses over
 * 10,000 iterations), and each miss is a fresh string cell plus a fresh byte
 * buffer that the arena never reclaims. Open addressing on the low pointer bits
 * with linear probing and NO eviction gives exactly one allocation per distinct
 * literal for the lifetime of the program. The probe is bounded, so a table
 * that somehow fills degrades to the old behaviour instead of looping. */
long SMC_PTR[8192];
long SMC_LEN[8192];
long SMC_VAL[8192];

long str_intern(const char *p, long n) {
	long k = (long)p;
	long slot = (k ^ (k >> 13) ^ (k >> 26)) & 8191;
	long probes = 0;
	long h;
	while (SMC_PTR[slot] != 0 && probes < 64) {
		if (SMC_PTR[slot] == k && SMC_LEN[slot] == n) { return SMC_VAL[slot]; }
		slot = (slot + 1) & 8191;
		probes = probes + 1;
	}
	h = mk_str(p, n);
	if (probes < 64) {
		SMC_PTR[slot] = k;
		SMC_LEN[slot] = n;
		SMC_VAL[slot] = h;
	}
	return h;
}

const char *str_ptr(long h) { return (const char *)fa(h); }
long str_len(long h) { return fb(h); }

long str_slice(long h, long begin, long end);
long mk_cstr(const char *s);
long *buf_new(long cap);

/* ------------------------------------------------------------- UTF-16 ---- */

/* abnf/jsrt.go measures a string in UTF-16 CODE UNITS (.length, charCodeAt,
 * charAt, slice, substring, indexOf, split) while the bytes stay UTF-8 - and
 * WTF-8 for a lone surrogate, which is how a half of an astral pair survives
 * being sliced out and concatenated back. This block is that model.
 *
 * The unit view of a string is computed lazily and memoised in the cell:
 * field e is 0 (not computed), 1 (pure ASCII - units are bytes) or 2 (field c
 * holds the unit buffer), and field d is the unit count. */

long G_CP;
long G_ADV;

void u_decode(const char *p, long n, long i) {
	long c0 = (long)p[i] & 255;
	long c1;
	long c2;
	long c3;
	if (c0 < 128) { G_CP = c0; G_ADV = 1; return; }
	if (c0 >= 240 && i + 3 < n) {
		c1 = (long)p[i + 1] & 63; c2 = (long)p[i + 2] & 63; c3 = (long)p[i + 3] & 63;
		G_CP = ((c0 & 7) << 18) + (c1 << 12) + (c2 << 6) + c3;
		G_ADV = 4;
		return;
	}
	if (c0 >= 224 && i + 2 < n) {
		c1 = (long)p[i + 1] & 63; c2 = (long)p[i + 2] & 63;
		G_CP = ((c0 & 15) << 12) + (c1 << 6) + c2;
		G_ADV = 3;
		return;
	}
	if (c0 >= 192 && i + 1 < n) {
		c1 = (long)p[i + 1] & 63;
		G_CP = ((c0 & 31) << 6) + c1;
		G_ADV = 2;
		return;
	}
	G_CP = 65533;
	G_ADV = 1;
}

/* enc_cp writes one code point as UTF-8 (a surrogate as its 3-byte WTF-8 form,
 * which is what utf8.AppendRune's Go twin appendWTF8 produces) and returns the
 * number of bytes written. */
long enc_cp(char *b, long o, long cp) {
	if (cp < 128) { b[o] = (char)cp; return 1; }
	if (cp < 2048) {
		b[o] = (char)(192 + (cp >> 6));
		b[o + 1] = (char)(128 + (cp & 63));
		return 2;
	}
	if (cp < 65536) {
		b[o] = (char)(224 + (cp >> 12));
		b[o + 1] = (char)(128 + ((cp >> 6) & 63));
		b[o + 2] = (char)(128 + (cp & 63));
		return 3;
	}
	b[o] = (char)(240 + (cp >> 18));
	b[o + 1] = (char)(128 + ((cp >> 12) & 63));
	b[o + 2] = (char)(128 + ((cp >> 6) & 63));
	b[o + 3] = (char)(128 + (cp & 63));
	return 4;
}

int is_high_surr(long c) { return c >= 55296 && c < 56320; }
int is_low_surr(long c)  { return c >= 56320 && c < 57344; }
int is_surr(long c)      { return c >= 55296 && c < 57344; }
long surr_pair(long h, long l) { return 65536 + ((h - 55296) << 10) + (l - 56320); }

/* str_from_units is abnf/jsrt.go's strFromUnits: a high/low pair becomes the
 * astral character, a lone surrogate stays as WTF-8. */
long str_from_units(long *u, long n) {
	char *buf = ar_alloc(n * 4 + 4);
	long o = 0;
	long i = 0;
	long h = cell_new(4);
	while (i < n) {
		long c = u[i];
		if (is_high_surr(c) && i + 1 < n && is_low_surr(u[i + 1])) {
			o = o + enc_cp(buf, o, surr_pair(c, u[i + 1]));
			i = i + 2;
		} else {
			o = o + enc_cp(buf, o, c);
			i = i + 1;
		}
	}
	buf[o] = 0;
	sa(h, (long)buf);
	sb(h, o);
	return h;
}

void units_build(long h) {
	const char *p = str_ptr(h);
	long n = str_len(h);
	long i = 0;
	int ascii = 1;
	long *u;
	long k = 0;
	while (i < n) { if (((long)p[i] & 255) >= 128) { ascii = 0; i = n; } else { i = i + 1; } }
	if (ascii) { sd(h, n); se(h, 1); return; }
	u = buf_new(n + 1);
	i = 0;
	while (i < n) {
		u_decode(p, n, i);
		if (G_CP > 65535) {
			u[k] = 55296 + ((G_CP - 65536) >> 10); k = k + 1;
			u[k] = 56320 + ((G_CP - 65536) & 1023); k = k + 1;
		} else { u[k] = G_CP; k = k + 1; }
		i = i + G_ADV;
	}
	sc(h, (long)u);
	sd(h, k);
	se(h, 2);
}
long str_ulen(long h) { if (fe(h) == 0) { units_build(h); } return fd(h); }
long str_ucode(long h, long i) {
	long *u;
	if (fe(h) == 0) { units_build(h); }
	if (i < 0 || i >= fd(h)) { return -1; }
	if (fe(h) == 1) { return (long)str_ptr(h)[i] & 255; }
	u = (long *)fc(h);
	return u[i];
}
/* The substring between two UNIT indexes. */
long u_slice(long h, long b, long e) {
	long *u;
	if (fe(h) == 0) { units_build(h); }
	if (e < b) { e = b; }
	if (fe(h) == 1) { return str_slice(h, b, e); }
	u = (long *)fc(h);
	return str_from_units(u + b, e - b);
}

/* str_cat is `a + b`: plain concatenation, except that a surrogate pair split
 * across the seam is rejoined into its astral character (Go's strConcat). */
long str_cat(long x, long y) {
	long nx = str_len(x);
	long ny = str_len(y);
	const char *px = str_ptr(x);
	const char *py = str_ptr(y);
	long h = cell_new(4);
	char *buf;
	long i = 0;
	long o = 0;
	long hi = 0;
	long lo = 0;
	if (nx >= 3 && ny >= 3 && ((long)px[nx - 3] & 255) == 237 && ((long)py[0] & 255) == 237) {
		u_decode(px, nx, nx - 3); hi = G_CP;
		u_decode(py, ny, 0);      lo = G_CP;
		if (is_high_surr(hi) && is_low_surr(lo)) {
			buf = ar_alloc(nx + ny + 1);
			while (i < nx - 3) { buf[i] = px[i]; i = i + 1; }
			o = nx - 3;
			o = o + enc_cp(buf, o, surr_pair(hi, lo));
			i = 3;
			while (i < ny) { buf[o] = py[i]; o = o + 1; i = i + 1; }
			buf[o] = 0;
			sa(h, (long)buf);
			sb(h, o);
			return h;
		}
	}
	buf = ar_alloc(nx + ny + 1);
	i = 0;
	while (i < nx) { buf[i] = px[i]; i = i + 1; }
	i = 0;
	while (i < ny) { buf[nx + i] = py[i]; i = i + 1; }
	buf[nx + ny] = 0;
	sa(h, (long)buf);
	sb(h, nx + ny);
	return h;
}

/* wtf8_clean replaces every lone surrogate with U+FFFD, which is what println
 * shows (Go's wtf8Clean) and what charAt answers (gojaCharAt). */
long wtf8_clean(long h) {
	const char *p = str_ptr(h);
	long n = str_len(h);
	long i = 0;
	long o = 0;
	char *buf;
	int dirty = 0;
	while (i < n) { if (((long)p[i] & 255) == 237) { dirty = 1; i = n; } else { i = i + 1; } }
	if (!dirty) { return h; }
	buf = ar_alloc(n + 4);
	i = 0;
	while (i < n) {
		u_decode(p, n, i);
		if (is_surr(G_CP)) { o = o + enc_cp(buf, o, 65533); }
		else { long k = 0; while (k < G_ADV) { buf[o] = p[i + k]; o = o + 1; k = k + 1; } }
		i = i + G_ADV;
	}
	{
		long r = cell_new(4);
		buf[o] = 0;
		sa(r, (long)buf);
		sb(r, o);
		return r;
	}
}

int str_eq(long x, long y) {
	long n;
	/* Identity first. js_str_mem caches one cell per string-literal ADDRESS, so
	 * every occurrence of a name in a module is the same handle - which makes
	 * scope_find and obj_find, the two hottest loops in the runtime, a pointer
	 * compare per entry instead of a byte loop. */
	if (x == y) { return 1; }
	n = str_len(x);
	long i = 0;
	const char *px;
	const char *py;
	if (n != str_len(y)) { return 0; }
	px = str_ptr(x); py = str_ptr(y);
	while (i < n) { if (px[i] != py[i]) { return 0; } i = i + 1; }
	return 1;
}

/* str_eq against a C literal, WITHOUT building a string cell for the literal.
 * method_id_array/method_id_string used to spell every candidate name as
 * `str_eq_c(name, "push")`, which allocated a cell and a byte buffer per
 * CANDIDATE on every member lookup of an array or a string - 20 dead string
 * cells per iteration of the Lua benchmark, and it also defeated str_eq's
 * identity fast path. A NUL inside the JS string is compared honestly: the
 * literal ends there, so the answer is false rather than an over-read. */
int str_eq_c(long x, const char *s) {
	long n = str_len(x);
	long i = 0;
	const char *px = str_ptr(x);
	while (i < n) {
		if (s[i] == 0 || px[i] != s[i]) { return 0; }
		i = i + 1;
	}
	return s[n] == 0;
}

/* -1, 0, 1 by byte order, then by length - Go's string comparison. */
int str_cmp(long x, long y) {
	long nx = str_len(x);
	long ny = str_len(y);
	long n = nx < ny ? nx : ny;
	long i = 0;
	const char *px = str_ptr(x);
	const char *py = str_ptr(y);
	while (i < n) {
		int ca = (int)px[i] & 255;
		int cb = (int)py[i] & 255;
		if (ca != cb) { return ca < cb ? -1 : 1; }
		i = i + 1;
	}
	if (nx == ny) { return 0; }
	return nx < ny ? -1 : 1;
}

long str_slice(long h, long begin, long end) {
	const char *p = str_ptr(h);
	if (end < begin) { end = begin; }
	return mk_str(p + begin, end - begin);
}

/* Every mk_cstr argument in this file is a C STRING LITERAL, so its address is
 * fixed for the life of the process (checked: the only two call sites that pass
 * a variable, seed_host and seed_root, are themselves called with literals from
 * boot). It therefore goes through the same literal cache js_str_mem uses, and
 * `to_string(undefined)`, `typeof x` and the array/string method names stop
 * allocating a cell and a byte buffer per CALL: 19 dead string cells per
 * iteration of the Lua benchmark became 0. */
long mk_cstr(const char *s) {
	long n = 0;
	while (s[n] != 0) { n = n + 1; }
	return str_intern(s, n);
}

/* --- growable handle buffers -------------------------------------------- */

/* A buffer is a raw long* in the heap; the owning cell carries n and cap. It is
 * NOT a Cell, so the collector marks it from the cell that owns it, and scans
 * its words conservatively when it is reached from the C stack instead. */
long *buf_new(long cap) {
	long *p = (long *)ar_alloc(cap * 8);
	long i = 0;
	while (i < cap) { p[i] = 0; i = i + 1; }
	return p;
}
long *buf_grow(long *old, long n, long newcap) {
	long *p = buf_new(newcap);
	long i = 0;
	while (i < n) { p[i] = old[i]; i = i + 1; }
	return p;
}

/* --- arrays ------------------------------------------------------------- */

long mk_arr(void) {
	long h = cell_new(5);
	sa(h, (long)buf_new(4));
	sb(h, 0);
	sc(h, 4);
	return h;
}
long arr_len(long h) { return fb(h); }
long arr_get(long h, long i) {
	long *p = (long *)fa(h);
	if (i < 0 || i >= fb(h)) { return H_UNDEF; }
	return p[i];
}
void arr_set(long h, long i, long v) {
	long *p = (long *)fa(h);
	p[i] = v;
}
void arr_push(long h, long v) {
	long n = fb(h);
	long cap = fc(h);
	long *p = (long *)fa(h);
	if (n >= cap) {
		long nc = cap * 2;
		if (nc < 4) { nc = 4; }
		p = buf_grow(p, n, nc);
		sa(h, (long)p);
		sc(h, nc);
	}
	p[n] = v;
	sb(h, n + 1);
}

/* --- objects ------------------------------------------------------------ */

long mk_obj(void) {
	long h = cell_new(6);
	sa(h, (long)buf_new(4));   /* keys: string handles */
	sb(h, (long)buf_new(4));   /* vals */
	sc(h, 0);
	sd(h, 4);
	return h;
}
long obj_find(long h, long key) {
	long *keys = (long *)fa(h);
	long n = fc(h);
	long i = 0;
	while (i < n) { if (str_eq(keys[i], key)) { return i; } i = i + 1; }
	return -1;
}
long obj_get(long h, long key) {
	long i = obj_find(h, key);
	long *vals;
	if (i < 0) { return H_UNDEF; }
	vals = (long *)fb(h);
	return vals[i];
}
void obj_put(long h, long key, long v) {
	long i = obj_find(h, key);
	long *keys = (long *)fa(h);
	long *vals = (long *)fb(h);
	long n = fc(h);
	long cap = fd(h);
	if (i >= 0) { vals[i] = v; return; }
	if (n >= cap) {
		long nc = cap * 2;
		if (nc < 4) { nc = 4; }
		keys = buf_grow(keys, n, nc);
		vals = buf_grow(vals, n, nc);
		sa(h, (long)keys);
		sb(h, (long)vals);
		sd(h, nc);
	}
	keys[n] = key;
	vals[n] = v;
	sc(h, n + 1);
}

/* ----- Unicode simple case mapping -----
 *
 * toUpperCase/toLowerCase used to be ASCII ONLY, byte by byte, while the Go twin is
 * strings.ToUpper - the full Unicode simple mapping. That is 2,879 code points the
 * two engines answered differently. The table below is the same mapping as ranges:
 * mode 0 gives every code point in [lo,hi] the two deltas as they stand, mode 1 is
 * the alternating upper/lower pairing of the Latin, Greek and Cyrillic extension
 * blocks (Go's own UpperLower case). It was GENERATED FROM THE GO RUNTIME'S OWN
 * ANSWERS over every code point from 32 to 0x10FFFF and verified against them.
 * Only the simple mapping is modelled, which is exactly what the Go twin does: no
 * one-to-many expansion (no sharp s to SS) and no locale. */
long NCASE = 328;
static long CASE_LO[328] = {
	65, 97, 181, 192, 216, 224, 248, 255, 256, 304, 305, 306, 313, 330, 376, 377, 383, 384, 
	385, 386, 390, 391, 393, 395, 398, 399, 400, 401, 403, 404, 405, 406, 407, 408, 410, 412, 
	413, 414, 415, 416, 422, 423, 425, 428, 430, 431, 433, 435, 439, 440, 444, 447, 452, 453, 
	454, 455, 456, 457, 458, 459, 460, 461, 477, 478, 497, 498, 499, 500, 502, 503, 504, 544, 
	546, 570, 571, 573, 574, 575, 577, 579, 580, 581, 582, 592, 593, 594, 595, 596, 598, 601, 
	603, 604, 608, 609, 611, 613, 614, 616, 617, 618, 619, 620, 623, 625, 626, 629, 637, 640, 
	642, 643, 647, 648, 649, 650, 652, 658, 669, 670, 837, 880, 886, 891, 895, 902, 904, 908, 
	910, 913, 931, 940, 941, 945, 962, 963, 972, 973, 975, 976, 977, 981, 982, 983, 984, 1008, 
	1009, 1010, 1011, 1012, 1013, 1015, 1017, 1018, 1021, 1024, 1040, 1072, 1104, 1120, 1162, 
	1216, 1217, 1231, 1232, 1329, 1377, 4256, 4295, 4301, 4304, 4349, 5024, 5104, 5112, 7296, 
	7297, 7298, 7299, 7301, 7302, 7303, 7304, 7312, 7357, 7545, 7549, 7566, 7680, 7835, 7838, 
	7840, 7936, 7944, 7952, 7960, 7968, 7976, 7984, 7992, 8000, 8008, 8017, 8019, 8021, 8023, 
	8025, 8027, 8029, 8031, 8032, 8040, 8048, 8050, 8054, 8056, 8058, 8060, 8064, 8072, 8080, 
	8088, 8096, 8104, 8112, 8115, 8120, 8122, 8124, 8126, 8131, 8136, 8140, 8144, 8152, 8154, 
	8160, 8165, 8168, 8170, 8172, 8179, 8184, 8186, 8188, 8486, 8490, 8491, 8498, 8526, 8544, 
	8560, 8579, 9398, 9424, 11264, 11312, 11360, 11362, 11363, 11364, 11365, 11366, 11367, 
	11373, 11374, 11375, 11376, 11378, 11381, 11390, 11392, 11499, 11506, 11520, 11559, 11565, 
	42560, 42624, 42786, 42802, 42873, 42877, 42878, 42891, 42893, 42896, 42900, 42902, 42922, 
	42923, 42924, 42925, 42926, 42928, 42929, 42930, 42931, 42932, 42948, 42949, 42950, 42951, 
	42960, 42966, 42997, 43859, 43888, 65313, 65345, 66560, 66600, 66736, 66776, 66928, 66940, 
	66956, 66964, 66967, 66979, 66995, 67003, 68736, 68800, 71840, 71872, 93760, 93792, 125184, 
	125218,
};
static long CASE_HI[328] = {
	90, 122, 181, 214, 222, 246, 254, 255, 303, 304, 305, 311, 328, 375, 376, 382, 383, 384, 
	385, 389, 390, 392, 394, 396, 398, 399, 400, 402, 403, 404, 405, 406, 407, 409, 410, 412, 
	413, 414, 415, 421, 422, 424, 425, 429, 430, 432, 434, 438, 439, 441, 445, 447, 452, 453, 
	454, 455, 456, 457, 458, 459, 460, 476, 477, 495, 497, 498, 499, 501, 502, 503, 543, 544, 
	563, 570, 572, 573, 574, 576, 578, 579, 580, 581, 591, 592, 593, 594, 595, 596, 599, 601, 
	603, 604, 608, 609, 611, 613, 614, 616, 617, 618, 619, 620, 623, 625, 626, 629, 637, 640, 
	642, 643, 647, 648, 649, 651, 652, 658, 669, 670, 837, 883, 887, 893, 895, 902, 906, 908, 
	911, 929, 939, 940, 943, 961, 962, 971, 972, 974, 975, 976, 977, 981, 982, 983, 1007, 1008, 
	1009, 1010, 1011, 1012, 1013, 1016, 1017, 1019, 1023, 1039, 1071, 1103, 1119, 1153, 1215, 
	1216, 1230, 1231, 1327, 1366, 1414, 4293, 4295, 4301, 4346, 4351, 5103, 5109, 5117, 7296, 
	7297, 7298, 7300, 7301, 7302, 7303, 7304, 7354, 7359, 7545, 7549, 7566, 7829, 7835, 7838, 
	7935, 7943, 7951, 7957, 7965, 7975, 7983, 7991, 7999, 8005, 8013, 8017, 8019, 8021, 8023, 
	8025, 8027, 8029, 8031, 8039, 8047, 8049, 8053, 8055, 8057, 8059, 8061, 8071, 8079, 8087, 
	8095, 8103, 8111, 8113, 8115, 8121, 8123, 8124, 8126, 8131, 8139, 8140, 8145, 8153, 8155, 
	8161, 8165, 8169, 8171, 8172, 8179, 8185, 8187, 8188, 8486, 8490, 8491, 8498, 8526, 8559, 
	8575, 8580, 9423, 9449, 11311, 11359, 11361, 11362, 11363, 11364, 11365, 11366, 11372, 
	11373, 11374, 11375, 11376, 11379, 11382, 11391, 11491, 11502, 11507, 11557, 11559, 11565, 
	42605, 42651, 42799, 42863, 42876, 42877, 42887, 42892, 42893, 42899, 42900, 42921, 42922, 
	42923, 42924, 42925, 42926, 42928, 42929, 42930, 42931, 42947, 42948, 42949, 42950, 42954, 
	42961, 42969, 42998, 43859, 43967, 65338, 65370, 66599, 66639, 66771, 66811, 66938, 66954, 
	66962, 66965, 66977, 66993, 67001, 67004, 68786, 68850, 71871, 71903, 93791, 93823, 125217, 
	125251,
};
static long CASE_MODE[328] = {
	0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 0, 0, 0, 1, 0, 0, 
	0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 
	0, 1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 
	1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 
	0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 
	1, 1, 0, 0, 0, 1, 1, 1, 1, 1, 0, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 
	1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
};
static long CASE_DU[328] = {
	0, -32, 743, 0, 0, -32, -32, 121, 0, 0, -232, 0, 0, 0, 0, 0, -300, 195, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, 0, 97, 0, 0, 0, 163, 0, 0, 130, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 56, 
	0, -1, -2, 0, -1, -2, 0, -1, -2, 0, -79, 0, 0, -1, -2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10815, 
	0, 0, 0, 0, 0, 10783, 10780, 10782, -210, -206, -205, -202, -203, 42319, -205, 42315, -207, 
	42280, 42308, -209, -211, 42308, 10743, 42305, -211, 10749, -213, -214, 10727, -218, 42307, 
	-218, 42282, -218, -69, -217, -71, -219, 42261, 42258, 84, 0, 0, 130, 0, 0, 0, 0, 0, 0, 0, 
	-38, -37, -32, -31, -32, -64, -63, 0, -62, -57, -47, -54, -8, 0, -86, -80, 7, -116, 0, -96, 
	0, 0, 0, 0, 0, 0, -32, -80, 0, 0, 0, 0, -15, 0, 0, -48, 0, 0, 0, 3008, 3008, 0, 0, -8, 
	-6254, -6253, -6244, -6242, -6243, -6236, -6181, 35266, 0, 0, 35332, 3814, 35384, 0, -59, 
	0, 0, 8, 0, 8, 0, 8, 0, 8, 0, 8, 0, 8, 8, 8, 8, 0, 0, 0, 0, 8, 0, 74, 86, 100, 128, 112, 
	126, 8, 0, 8, 0, 8, 0, 8, 9, 0, 0, 0, -7205, 9, 0, 0, 8, 0, 0, 8, 7, 0, 0, 0, 9, 0, 0, 0, 
	0, 0, 0, 0, -28, 0, -16, 0, 0, -26, 0, -48, 0, 0, 0, 0, -10795, -10792, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, -7264, -7264, -7264, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 48, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -928, -38864, 0, -32, 0, -40, 0, -40, 0, 0, 0, 0, -39, 
	-39, -39, -39, 0, -64, 0, -32, 0, -32, 0, -34,
};
static long CASE_DL[328] = {
	32, 0, 0, 32, 32, 0, 0, 0, 0, -199, 0, 0, 0, 0, -121, 0, 0, 0, 210, 0, 206, 0, 205, 0, 79, 
	202, 203, 0, 205, 207, 0, 211, 209, 0, 0, 211, 213, 0, 214, 0, 218, 0, 218, 0, 218, 0, 217, 
	0, 219, 0, 0, 0, 2, 1, 0, 2, 1, 0, 2, 1, 0, 0, 0, 0, 2, 1, 0, 0, -97, -56, 0, -130, 0, 
	10795, 0, -163, 10792, 0, 0, -195, 69, 71, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 116, 38, 37, 64, 
	63, 32, 32, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -60, 0, 0, -7, 0, -130, 
	80, 32, 0, 0, 0, 0, 15, 0, 0, 0, 48, 0, 7264, 7264, 7264, 0, 0, 38864, 8, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, -3008, -3008, 0, 0, 0, 0, 0, -7615, 0, 0, -8, 0, -8, 0, -8, 0, -8, 0, -8, 0, 0, 0, 
	0, -8, -8, -8, -8, 0, -8, 0, 0, 0, 0, 0, 0, 0, -8, 0, -8, 0, -8, 0, 0, -8, -74, -9, 0, 0, 
	-86, -9, 0, -8, -100, 0, 0, -8, -112, -7, 0, -128, -126, -9, -7517, -8383, -8262, 28, 0, 
	16, 0, 0, 26, 0, 48, 0, 0, -10743, -3814, -10727, 0, 0, 0, -10780, -10749, -10783, -10782, 
	0, 0, -10815, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -35332, 0, 0, -42280, 0, 0, 0, -42308, 
	-42319, -42315, -42305, -42308, -42258, -42282, -42261, 928, 0, -48, -42307, -35384, 0, 0, 
	0, 0, 0, 0, 32, 0, 40, 0, 40, 0, 39, 39, 39, 39, 0, 0, 0, 0, 64, 0, 32, 0, 32, 0, 34, 0,
};

/* The mapped code point, or c itself. Binary search over the ranges. */
long case_map(long c, int up) {
	long lo = 0;
	long hi = NCASE - 1;
	long mid;
	while (lo <= hi) {
		mid = (lo + hi) / 2;
		if (c < CASE_LO[mid]) { hi = mid - 1; }
		else if (c > CASE_HI[mid]) { lo = mid + 1; }
		else {
			if (CASE_MODE[mid] != 0) {
				/* even offset from lo is the upper of a pair, odd is the lower */
				if ((c - CASE_LO[mid]) % 2 == 0) { return up ? c : c + 1; }
				return up ? c - 1 : c;
			}
			return c + (up ? CASE_DU[mid] : CASE_DL[mid]);
		}
	}
	return c;
}

/* ------------------------------------------------ number <-> string ------ */

/* pow10 returns 10^n as bit pattern, for -308 <= n <= 308. */
long pow10(long n) {
	union DB r;
	union DB t;
	long neg = 0;
	long i = 0;
	if (n < 0) { neg = 1; n = 0 - n; }
	r.d = 1.0;
	t.d = 10.0;
	while (i < n) { r.d = r.d * t.d; i = i + 1; }
	if (neg) { r.d = 1.0 / r.d; }
	return r.l;
}

/* ----- exact decimal arithmetic, for the digits of a double -----
 *
 * A double is a 53-bit integer times a power of two, so its value has a FINITE
 * decimal expansion, and every question about it - what its digits are, and how
 * few of them read back as itself - has an exact integer answer. These helpers
 * work on decimal digit arrays (ONE DIGIT PER BYTE, least significant first),
 * which is the representation that makes those digits fall straight out.
 *
 * The float-arithmetic version this replaces generated digits by repeatedly
 * multiplying the remainder by ten, which is right to about sixteen digits and
 * therefore WRONG IN THE LAST PLACE of every seventeen-digit number: 10.0/3.0
 * printed as 3.3333333333333334 where the value is 3.3333333333333335, and the
 * shortest-form search it drove could not find the sixteen-digit form of
 * 0.9999999999999999 at all. Nothing here touches a double. */

/* a = a * k, in place; k must be small enough that 9*k plus a carry stays inside
 * a long, which every caller here satisfies (the largest k is a 53-bit
 * mantissa). Returns the new length. */
long dec_mul(char *a, long n, long k) {
	long i = 0;
	long carry = 0;
	long t;
	while (i < n || carry > 0) {
		t = carry;
		if (i < n) { t = t + (long)a[i] * k; }
		a[i] = (char)(t % 10);
		carry = t / 10;
		i = i + 1;
	}
	return i;
}
/* The length with leading (that is, top-end) zero digits dropped. */
long dec_norm(const char *a, long n) {
	while (n > 0 && a[n - 1] == 0) { n = n - 1; }
	return n;
}
long dec_cmp(const char *a, long na, const char *b, long nb) {
	long i;
	if (na != nb) { return na > nb ? 1 : -1; }
	i = na - 1;
	while (i >= 0) {
		if (a[i] != b[i]) { return a[i] > b[i] ? 1 : -1; }
		i = i - 1;
	}
	return 0;
}
/* d = |a - b| */
long dec_absdiff(const char *a, long na, const char *b, long nb, char *d) {
	const char *x;
	const char *y;
	long nx;
	long ny;
	long i = 0;
	long borrow = 0;
	long t;
	if (dec_cmp(a, na, b, nb) >= 0) { x = a; nx = na; y = b; ny = nb; }
	else { x = b; nx = nb; y = a; ny = na; }
	while (i < nx) {
		t = (long)x[i] - borrow;
		if (i < ny) { t = t - (long)y[i]; }
		if (t < 0) { t = t + 10; borrow = 1; } else { borrow = 0; }
		d[i] = (char)t;
		i = i + 1;
	}
	return dec_norm(d, nx);
}
/* The scratch is file scope rather than stack: 5^1074 is 751 digits, and the
 * runtime is single threaded and never re-enters this. */
char G_DP[1024];
char G_DN[1024];
char G_DV[1024];
char G_DHI[1024];
char G_DLO[1024];
char G_DC[1024];
char G_DD[1024];

long g_ndig;

/* shortest_digits fills digs with the SHORTEST decimal digit string that reads
 * back as |bits| (seventeen digits always do), sets g_ndig to how many, and
 * returns the decimal exponent of the first digit. bits must be finite and
 * non-zero - it is the shared core of both renderings below.
 *
 * Write the value as m * 2^e with m a whole 53-bit mantissa. Let P be 2^e when
 * e >= 0 and 5^(-e) when e < 0; then N = m * P is a whole number and the value
 * is exactly N * 10^p, with p zero or e. A candidate of k digits reads back as
 * this double exactly when it lands inside the rounding interval, whose two
 * halves are 2^(e-1) wide - or half that below a power of two. Scaling
 * everything by four keeps those halves whole too, so the whole decision is an
 * integer comparison: 2P and (below a power of two) P are the bounds, 4N is the
 * value, and 4*M*10^(L-k) is the candidate. */
/* g_mindig is the MINIMUM number of digits the search may answer with. It is 1
 * for every caller but java's Double.toString, which never uses fewer than two
 * significant digits - and when the shortest form has one, the second digit is
 * not a zero but the closest two digit decimal to the actual value. Starting the
 * search at k = 2 IS that decimal, because the candidate for k digits is the
 * exact value rounded to k digits, nearest with ties to even. See jvm_flo_str. */
long g_mindig = 1;

/* g_fixdig, when > 0, turns the search off entirely: the answer is the EXACT
 * value rounded to that many significant digits, nearest with ties to even,
 * whether or not it reads back. That is a different question from the one the
 * rest of this function asks, and floPrec (host id 70) is the caller that wants
 * it - see docs/todo.md 2.8.
 *
 * WHY THE DIFFERENCE MATTERED. g_mindig = n only starts the SEARCH at k = n, so
 * a value whose shortest round-tripping form is longer than n came back with the
 * shortest form instead: floPrec(1/3, 1) answered "3333333333333333e-1" here and
 * "3e-1" in floPrecStr (abnf/commonscript.go), the Go/goja/frozen twin - a
 * latent halves divergence for every layer-2 caller, invisible only because the
 * one live caller (lua's %g, luaGStr) always asks for an n at least as large as
 * the shortest form, where the two answers coincide. Exactly-n is the contract
 * that is kept, because it is the one a %e / %g formatter needs and the one a
 * script CANNOT build for itself: "at least n" is derivable from exactly-n plus
 * the host's shortest form ("" + v), and exactly-n is not derivable from "at
 * least n" at all. python's pyPctSig / pyBPctSig exist only because of this, and
 * languages/python-interpreter.abnf records the symptom it cost: %e of
 * 1234.5678 printed 1.234567e+03 natively, a TRUNCATION of the shortest eight
 * digit form rather than the correctly rounded seven digit one.
 *
 * k is clamped to the exact expansion's own length: asking for more digits than
 * the value has can only add trailing zeros, and every caller strips those. */
long g_fixdig = 0;

long shortest_digits(long bits, char *digs) {
	long E;
	long F;
	long m;
	long e;
	long p;
	long i;
	long k;
	long t;
	long nP;
	long nN;
	long nV;
	long nHI;
	long nLO;
	long nC;
	long nD;
	long L;
	long M;
	long best = 0;
	long bestM = 0;
	long bestK = 0;
	long cmp;
	long lim;
	long round;
	long cut;
	long any;
	long inc;
	char rev[24];
	long dm;
	long e10;
	bits = d_abs(bits);
	E = (bits >> 52) & 2047;
	F = bits & 4503599627370495L;
	if (E == 0) { m = F; e = -1074; } else { m = F + 4503599627370496L; e = E - 1075; }
	nP = 1;
	G_DP[0] = 1;
	if (e >= 0) {
		p = 0;
		i = 0;
		while (i < e) { nP = dec_mul(G_DP, nP, 2); i = i + 1; }
	} else {
		p = e;
		i = 0;
		while (i < 0 - e) { nP = dec_mul(G_DP, nP, 5); i = i + 1; }
	}
	i = 0;
	while (i < nP) { G_DN[i] = G_DP[i]; G_DHI[i] = G_DP[i]; G_DLO[i] = G_DP[i]; i = i + 1; }
	nN = dec_mul(G_DN, nP, m);
	L = nN;
	i = 0;
	while (i < nN) { G_DV[i] = G_DN[i]; i = i + 1; }
	nV = dec_mul(G_DV, nN, 4);
	nHI = dec_mul(G_DHI, nP, 2);
	/* Directly below a power of two the gap down to the previous double is half
	 * the gap up, so the interval is not symmetric. */
	nLO = nP;
	if (m != 4503599627370496L || E <= 1) { nLO = dec_mul(G_DLO, nP, 2); }
	lim = L < 17 ? L : 17;
	/* The fixed-precision answer: one candidate, no round-trip test. This is the
	 * body of the search's inner round == 0 pass, lifted out. */
	if (g_fixdig > 0) {
		k = g_fixdig;
		if (k > lim) { k = lim; }
		if (k < 1) { k = 1; }
		M = 0;
		i = 0;
		while (i < k) { M = M * 10 + (long)G_DN[L - 1 - i]; i = i + 1; }
		cut = L > k ? (long)G_DN[L - 1 - k] : 0;
		any = 0;
		i = L - k - 2;
		while (i >= 0) { if (G_DN[i] != 0) { any = 1; i = 0; } i = i - 1; }
		if (cut > 5 || (cut == 5 && (any != 0 || (M & 1) != 0))) { M = M + 1; }
		best = 1;
		bestM = M;
		bestK = k;
	}
	k = g_mindig < 1 ? 1 : g_mindig;
	if (k > lim) { k = lim; }
	while (k <= lim && best == 0) {
		round = 0;
		while (round < 2 && best == 0) {
			M = 0;
			i = 0;
			while (i < k) { M = M * 10 + (long)G_DN[L - 1 - i]; i = i + 1; }
			/* Round the exact digits to k, NEAREST WITH TIES TO EVEN - the same
			 * rule the Go and JS shortest-form printers use, and the reason
			 * 1000000000000000.25 renders as ...0.2 and not ...0.3. Where two
			 * candidates of the same length both read back, the second pass
			 * offers the other one. */
			cut = L > k ? (long)G_DN[L - 1 - k] : 0;
			any = 0;
			i = L - k - 2;
			while (i >= 0) { if (G_DN[i] != 0) { any = 1; i = 0; } i = i - 1; }
			inc = 0;
			if (cut > 5 || (cut == 5 && (any != 0 || (M & 1) != 0))) { inc = 1; }
			if (round == 1) { inc = 1 - inc; }
			if (round == 1 && L <= k) { round = 2; }
			else {
				M = M + inc;
				nC = 0;
				while (nC < L - k) { G_DC[nC] = 0; nC = nC + 1; }
				t = M * 4;
				while (t > 0) { G_DC[nC] = (char)(t % 10); t = t / 10; nC = nC + 1; }
				nC = dec_norm(G_DC, nC);
				cmp = dec_cmp(G_DC, nC, G_DV, nV);
				if (cmp == 0) { best = 1; }
				else {
					nD = dec_absdiff(G_DC, nC, G_DV, nV, G_DD);
					if (cmp > 0) { t = dec_cmp(G_DD, nD, G_DHI, nHI); }
					else { t = dec_cmp(G_DD, nD, G_DLO, nLO); }
					/* A candidate exactly on the boundary reads back as this
					 * double only when this double is the even one. */
					if (t < 0 || (t == 0 && (m & 1) == 0)) { best = 1; }
				}
				if (best) { bestM = M; bestK = k; }
				round = round + 1;
			}
		}
		k = k + 1;
	}
	if (best == 0) {
		bestK = lim;
		bestM = 0;
		i = 0;
		while (i < bestK) { bestM = bestM * 10 + (long)G_DN[L - 1 - i]; i = i + 1; }
		if (L > bestK && G_DN[L - 1 - bestK] >= 5) { bestM = bestM + 1; }
	}
	t = bestM;
	dm = 0;
	while (t > 0) { rev[dm] = (char)(48 + (t % 10)); t = t / 10; dm = dm + 1; }
	i = 0;
	while (i < dm) { digs[i] = rev[dm - 1 - i]; i = i + 1; }
	e10 = L - bestK + p + dm - 1;
	/* A trailing zero is dropped, but never below g_mindig digits: a forced
	 * two digit form is exactly "1.0E20" / "4.9E-324", and stripping the zero
	 * would put the first one back to "1E20". */
	while (dm > 1 && dm > g_mindig && digs[dm - 1] == 48) { dm = dm - 1; }
	g_ndig = dm;
	return e10;
}

/* int_digits writes the decimal digits of |bits| (which must be an integral
 * value inside the long range) into out and returns how many bytes it wrote. */
long int_digits(long bits, char *out) {
	long v = d_to_long(d_abs(bits));
	char rev[24];
	long n = 0;
	long o = 0;
	long i;
	if (v == 0) { rev[0] = 48; n = 1; }
	while (v > 0) { rev[n] = (char)(48 + (v % 10)); v = v / 10; n = n + 1; }
	i = n - 1;
	while (i >= 0) { out[o] = rev[i]; o = o + 1; i = i - 1; }
	return o;
}

/* jsNumString: the JS rendering of a number, matching abnf/jsrt.go's
 * jsNumString for every value whose decimal form this can reproduce. */
long num_to_str(long bits) {
	char digs[24];
	char out[64];
	long e10;
	long best;
	long i;
	long o = 0;
	long neg;
	if (d_is_nan(bits)) { return mk_cstr("NaN"); }
	if (d_is_inf(bits)) { return d_sign(bits) ? mk_cstr("-Infinity") : mk_cstr("Infinity"); }
	if (d_is_zero(bits)) { return mk_cstr("0"); }
	neg = d_sign(bits);
	/* The exact integer path: no decimal search at all, and it is the path the
	 * whole integer arithmetic of a program takes. Only BELOW 2^53, where the
	 * integer is its own shortest form: 1234567891234567936 is a double whose
	 * shortest form is 1234567891234568000, and printing all of its digits is
	 * not what any JS engine does. */
	if (d_is_integral(bits) && d_in_long(bits) &&
	    d_to_long(d_abs(bits)) < 9007199254740992L) {
		if (neg) { out[o] = 45; o = o + 1; }
		o = o + int_digits(bits, out + o);
		return mk_str(out, o);
	}
	e10 = shortest_digits(bits, digs);
	best = g_ndig;
	if (neg) { out[o] = 45; o = o + 1; }
	if (e10 >= -6 && e10 < 21) {
		if (e10 >= 0) {
			i = 0;
			while (i <= e10) {
				out[o] = i < best ? digs[i] : 48;
				o = o + 1;
				i = i + 1;
			}
			if (best > e10 + 1) {
				out[o] = 46; o = o + 1;
				i = e10 + 1;
				while (i < best) { out[o] = digs[i]; o = o + 1; i = i + 1; }
			}
		} else {
			out[o] = 48; o = o + 1;
			out[o] = 46; o = o + 1;
			i = -1;
			while (i > e10) { out[o] = 48; o = o + 1; i = i - 1; }
			i = 0;
			while (i < best) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		}
	} else {
		long ex = e10;
		long en;
		char erev[8];
		out[o] = digs[0]; o = o + 1;
		if (best > 1) {
			out[o] = 46; o = o + 1;
			i = 1;
			while (i < best) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		}
		out[o] = 101; o = o + 1;             /* 'e' */
		if (ex < 0) { out[o] = 45; o = o + 1; ex = 0 - ex; } else { out[o] = 43; o = o + 1; }
		en = 0;
		if (ex == 0) { erev[0] = 48; en = 1; }
		while (ex > 0) { erev[en] = (char)(48 + (ex % 10)); ex = ex / 10; en = en + 1; }
		i = en - 1;
		while (i >= 0) { out[o] = erev[i]; o = o + 1; i = i - 1; }
	}
	return mk_str(out, o);
}

int is_space(int c) { return c == 32 || c == 9 || c == 10 || c == 13 || c == 11 || c == 12; }

/* ----- exact decimal -> double -----
 *
 * The double nearest N * 10^e10, where N is the whole number whose digits are
 * dig[0..nd-1], LEAST SIGNIFICANT FIRST. No float arithmetic takes part in the
 * decision, which is the point: multiplying by a power of ten built up by
 * repeated multiplication rounds once per step, so "1.7976931348623157e308"
 * came out as +Inf and "1e100" one part in 10^16 too large.
 *
 * The mantissa m of the answer is T * 2^(-e) for the right binary exponent e,
 * and THAT is reachable with decimal arithmetic alone: 2^(-e) is 5^e / 10^e
 * when e >= 0, and 2^(-e) when e is negative, so m is always a decimal bignum
 * divided by a power of ten - and dividing a decimal digit array by a power of
 * ten is reading it from a different offset. e is found by guessing (log2(10)
 * is 217706/65536 to a part in a million) and then stepping. */
char G_DY[2400];
char G_DT[2400];
long dec_to_double(const char *dig, long nd, long e10, long neg) {
	long p;
	long L;
	long e;
	long q;
	long shift = 0;
	long nY = 0;
	long i;
	long t;
	long m = 0;
	long dm;
	long any;
	long tries;
	long bits;
	nd = dec_norm(dig, nd);
	if (nd == 0) { return neg ? (0 - 9223372036854775807L - 1L) : 0L; }
	/* N with any POSITIVE decimal exponent absorbed as trailing zeros, so the
	 * remaining exponent p is never above zero. */
	p = e10;
	i = 0;
	if (p > 0) {
		while (i < p) { G_DT[i] = 0; i = i + 1; }
		t = 0;
		while (t < nd) { G_DT[i] = dig[t]; i = i + 1; t = t + 1; }
		L = i;
		p = 0;
	} else {
		while (i < nd) { G_DT[i] = dig[i]; i = i + 1; }
		L = nd;
	}
	q = L - 1 + p;
	t = q * 217706;
	e = t >= 0 ? t / 65536 : 0 - ((0 - t + 65535) / 65536);
	e = e - 52;
	tries = 0;
	while (tries < 24) {
		if (e < -1074) { e = -1074; }
		i = 0;
		while (i < L) { G_DY[i] = G_DT[i]; i = i + 1; }
		nY = L;
		if (e >= 0) {
			i = 0;
			while (i < e) { nY = dec_mul(G_DY, nY, 5); i = i + 1; }
			shift = e - p;
		} else {
			i = 0;
			while (i < 0 - e) { nY = dec_mul(G_DY, nY, 2); i = i + 1; }
			shift = 0 - p;
		}
		nY = dec_norm(G_DY, nY);
		/* Everything below the cut is read again for the rounding, so it has to
		 * be zero rather than whatever an earlier attempt left there. */
		i = nY;
		while (i < shift) { G_DY[i] = 0; i = i + 1; }
		dm = nY - shift;
		if (dm < 0) { dm = 0; }
		if (dm > 17) {
			e = e + ((dm - 16) * 217706) / 65536 + 1;
			tries = tries + 1;
		} else {
			m = 0;
			i = nY - 1;
			while (i >= shift) { m = m * 10 + (long)G_DY[i]; i = i - 1; }
			if (m >= 9007199254740992L) { e = e + 1; tries = tries + 1; }
			else if (m < 4503599627370496L && e > -1074) {
				if (dm < 15) { e = e - ((16 - dm) * 217706) / 65536 - 1; }
				else { e = e - 1; }
				tries = tries + 1;
			} else { tries = 24; }
		}
	}
	/* The digits below the cut decide the rounding: nearest, ties to even. */
	if (shift > 0) {
		t = shift - 1 < nY ? (long)G_DY[shift - 1] : 0;
		any = 0;
		i = shift - 2;
		while (i >= 0) { if (G_DY[i] != 0) { any = 1; i = 0; } i = i - 1; }
		if (t > 5 || (t == 5 && (any != 0 || (m & 1) != 0))) { m = m + 1; }
	}
	if (m >= 9007199254740992L) { m = m / 2; e = e + 1; }
	if (e > 971) { return neg ? DNINF : DINF; }
	if (m < 4503599627370496L) { bits = m; }
	else { bits = ((e + 1075) << 52) | (m - 4503599627370496L); }
	if (neg) { bits = bits | (0 - 9223372036854775807L - 1L); }
	return bits;
}
char G_DIG[820];

/* str_to_num is Go's strconv.ParseFloat on the TRIMMED string: an empty string
 * is 0 and anything that does not parse whole is NaN (so "0x10" is NaN here,
 * exactly as in the Go runtime). */
long str_to_num(long h) {
	const char *s = str_ptr(h);
	long n = str_len(h);
	long i = 0;
	long j = n;
	long neg = 0;
	long digits = 0;
	long mant = 0;
	long mdig = 0;
	long exp10 = 0;
	long esign = 1;
	long eval = 0;
	union DB r;
	union DB p;
	while (i < j && is_space((int)s[i] & 255)) { i = i + 1; }
	while (j > i && is_space((int)s[j - 1] & 255)) { j = j - 1; }
	if (i == j) { return DZERO; }
	if (s[i] == 43) { i = i + 1; } else if (s[i] == 45) { neg = 1; i = i + 1; }
	if (j - i == 3 && s[i] == 78 && s[i+1] == 97 && s[i+2] == 78) { return DNAN; }
	if (j - i == 8 && s[i] == 73 && s[i+1] == 110 && s[i+2] == 102) {
		return neg ? DNINF : DINF;
	}
	if (j - i == 3 && s[i] == 73 && s[i+1] == 110 && s[i+2] == 102) {
		return neg ? DNINF : DINF;
	}
	/* The significant digits are collected WHOLE, most significant first, so the
	 * conversion below can be exact; exp10 is kept so that the value is exactly
	 * those digits read as an integer, times 10^exp10. */
	while (i < j && s[i] >= 48 && s[i] <= 57) {
		long dv = s[i] - 48;
		if (mdig == 0 && dv == 0) { }               /* a leading zero is nothing */
		else if (mdig < 800) { G_DIG[mdig] = (char)dv; mdig = mdig + 1; }
		else { exp10 = exp10 + 1; }
		if (mdig < 16) { mant = mant * 10 + dv; }
		digits = digits + 1;
		i = i + 1;
	}
	if (i < j && s[i] == 46) {
		i = i + 1;
		while (i < j && s[i] >= 48 && s[i] <= 57) {
			long dv = s[i] - 48;
			if (mdig == 0 && dv == 0) { exp10 = exp10 - 1; }
			else if (mdig < 800) { G_DIG[mdig] = (char)dv; mdig = mdig + 1; exp10 = exp10 - 1; }
			if (mdig < 16) { mant = mant * 10 + dv; }
			digits = digits + 1;
			i = i + 1;
		}
	}
	if (digits == 0) { return DNAN; }
	if (i < j && (s[i] == 101 || s[i] == 69)) {
		long edig = 0;
		i = i + 1;
		if (i < j && s[i] == 43) { i = i + 1; }
		else if (i < j && s[i] == 45) { esign = -1; i = i + 1; }
		while (i < j && s[i] >= 48 && s[i] <= 57) {
			eval = eval * 10 + (s[i] - 48);
			if (eval > 100000) { eval = 100000; }
			edig = edig + 1;
			i = i + 1;
		}
		if (edig == 0) { return DNAN; }
		exp10 = exp10 + esign * eval;
	}
	if (i != j) { return DNAN; }
	if (mdig == 0) { return neg ? (0 - 9223372036854775807L - 1L) : DZERO; }
	/* The magnitude is 10^(mdig-1+exp10); only the far extremes are decided here,
	 * so that the conversion below never has to hold an absurd number of digits. */
	if (mdig - 1 + exp10 > 400) { return neg ? DNINF : DINF; }
	if (mdig - 1 + exp10 < -400) { return neg ? (0 - 9223372036854775807L - 1L) : DZERO; }
	/* The fast path, and it is EXACT rather than merely close: a mantissa of at
	 * most fifteen digits is a whole double, 10^k for |k| <= 22 is a whole
	 * double, and a single multiply or divide rounds once, correctly. Anything
	 * else goes the long way round. */
	if (mdig <= 15 && exp10 >= -22 && exp10 <= 22) {
		r.l = d_from_long(mant);
		if (exp10 < 0) { p.l = pow10(0 - exp10); r.d = r.d / p.d; }
		else { p.l = pow10(exp10); r.d = r.d * p.d; }
		if (neg) { r.d = 0.0 - r.d; }
		return r.l;
	}
	/* dec_to_double wants the digits least significant first. */
	{
		char rev[820];
		long k = 0;
		while (k < mdig) { rev[k] = G_DIG[mdig - 1 - k]; k = k + 1; }
		return dec_to_double(rev, mdig, exp10, neg);
	}
}

/* -------------------------------------------------- the value operations - */

long to_string(long h);
long to_number(long h);
int truthy(long h);
long js_call(long callee, long self, long args);
long type_of(long h);

/* The sized integer (tag 13, abnf/jsrtint.go). Declared here because the five
 * value operations below have to know about it and it is defined after them. */
long si_val(long h);
long si_float(long h);
long si_str(long h);
int  si_eq(long a, long b);

/* The boxed double (tag 14, abnf/jsrtjvm.go's jsJFlo). Declared here for the
 * same reason: the value operations below have to know about it. */
long jf_text(long h);
int  jf_num_eq(long f, long other);

long to_number(long h) {
	long t = tag_of(h);
	if (t == 3) { return num_bits(h); }
	if (t == 13) { return si_float(h); }
	if (t == 14) { return fa(h); }        /* jsrt.go toNumber: `case jsJFlo: t.f` */
	if (t == 0) { return DNAN; }
	if (t == 1) { return DZERO; }
	if (t == 2) { return fa(h) ? DONE : DZERO; }
	if (t == 4) { return str_to_num(h); }
	return DNAN;
}

int truthy(long h) {
	long t = tag_of(h);
	if (t == 0 || t == 1) { return 0; }
	if (t == 2) { return (int)fa(h); }
	if (t == 3) { long b = num_bits(h); return !d_is_zero(b) && !d_is_nan(b); }
	if (t == 13) { return fa(h) != 0; }   /* jsrt.go truthy: `case jsGInt: t.v != 0` */
	/* jsrt.go truthy: `case jsJFlo: t.f != 0 && t.f == t.f` - a NaN box is falsy
	 * and so is -0.0, exactly as for a plain number. */
	if (t == 14) { long b = fa(h); return !d_is_zero(b) && !d_is_nan(b); }
	if (t == 4) { return str_len(h) > 0; }
	return 1;
}

long to_string(long h) {
	long t = tag_of(h);
	if (t == 0) { return mk_cstr("undefined"); }
	if (t == 1) { return mk_cstr("null"); }
	if (t == 2) { return fa(h) ? mk_cstr("true") : mk_cstr("false"); }
	if (t == 3) { return num_to_str(num_bits(h)); }
	if (t == 13) { return si_str(h); }    /* jsrt.go toString: `case jsGInt: giStr(t)` */
	if (t == 14) { return jf_text(h); }   /* jsrt.go toString: `case jsJFlo: jvmFloText(t)` */
	if (t == 4) { return h; }
	if (t == 5) {
		long out = mk_cstr("");
		long n = arr_len(h);
		long i = 0;
		while (i < n) {
			long e = arr_get(h, i);
			if (i > 0) { out = str_cat(out, mk_cstr(",")); }
			if (tag_of(e) != 0 && tag_of(e) != 1) { out = str_cat(out, to_string(e)); }
			i = i + 1;
		}
		return out;
	}
	if (t == 6) { return mk_cstr("[object Object]"); }
	if (t == 7 || t == 8 || t == 9 || t == 16) { return mk_cstr("[function]"); }   /* tag 16 too: the Go twin's js_genfn result is a *hostFunc */
	if (t == 11) { return mk_cstr("[scope]"); }
	return mk_cstr("[value]");
}

long type_of(long h) {
	long t = tag_of(h);
	if (t == 0) { return mk_cstr("undefined"); }
	if (t == 2) { return mk_cstr("boolean"); }
	/* A SIZED INTEGER IS A NUMBER. This is the whole point of the tag: the Go
	 * twin answers "number" for jsGInt (abnf/jsrt.go typeOf), and an object box
	 * in layer 2 could only ever have answered "object". */
	if (t == 3 || t == 13 || t == 14) { return mk_cstr("number"); }
	if (t == 4) { return mk_cstr("string"); }
	/* Tag 16 is a GENERATOR FUNCTION and it is callable: is_callable() below
	 * answers true for it, the Go twin's js_genfn returns a *hostFunc whose
	 * typeOf is "function" (abnf/jsrt.go), and a caller that asks
	 * `typeof m == "function"` before calling a member must get the same
	 * answer in both halves. */
	if (t == 7 || t == 8 || t == 9 || t == 16) { return mk_cstr("function"); }
	return mk_cstr("object");
}

/* The type CLASS used by the MetaJS type pin: 0 = none (undefined / null),
 * else a small code per typeof class. */
long type_class(long h) {
	long t = tag_of(h);
	if (t == 0 || t == 1) { return 0; }
	if (t == 2) { return 2; }
	if (t == 3 || t == 13 || t == 14) { return 3; }   /* a sized integer and a boxed double pin as "number" */
	if (t == 4) { return 4; }
	if (t == 7 || t == 8 || t == 9 || t == 16) { return 6; }   /* a generator function pins as "function", like type_of */
	return 5;
}
const char *class_name(long tc) {
	if (tc == 2) { return "boolean"; }
	if (tc == 3) { return "number"; }
	if (tc == 4) { return "string"; }
	if (tc == 6) { return "function"; }
	return "object";
}

long mk_bool(int b) { return b ? 3 : 2; }

int strict_eq(long a, long b) {
	long ta = tag_of(a);
	long tb = tag_of(b);
	/* A boxed double compares by VALUE too, and jsrt.go strictEq checks it
	 * BEFORE the sized integer - so the order of these four lines is the Go
	 * twin's order, not a preference. jvmNumEq accepts only another box or a
	 * plain number, so a float box against a SIZED INTEGER is false in both
	 * halves (Go's jvmNumEq has no jsGInt case). */
	if (ta == 14) { return jf_num_eq(fa(a), b); }
	if (tb == 14) { return jf_num_eq(fa(b), a); }
	/* A sized integer compares by VALUE, never by identity - jsrt.go strictEq
	 * puts the two jsGInt arms exactly here, before the type switch, so
	 * int8(1) === 1 holds and two boxes of the same integer are equal. A box
	 * against a non-number is false. */
	if (ta == 13) { return (tb == 13 || tb == 3) && si_eq(a, b); }
	if (tb == 13) { return ta == 3 && si_eq(a, b); }
	if (ta == 0) { return tb == 0; }
	if (ta == 1) { return tb == 1; }
	if (ta == 2) { return tb == 2 && fa(a) == fa(b); }
	if (ta == 3) { return tb == 3 && d_eq(num_bits(a), num_bits(b)); }
	if (ta == 4) { return tb == 4 && str_eq(a, b); }
	return a == b;                        /* identity for everything else */
}

int is_undef_or_null(long h) { long t = tag_of(h); return t == 0 || t == 1; }

int loose_eq(long a, long b) {
	long ta;
	long tb;
	if (is_undef_or_null(a) && is_undef_or_null(b)) { return 1; }
	if (is_undef_or_null(a) || is_undef_or_null(b)) { return 0; }
	if (tag_of(a) == 2) { return loose_eq(mk_num(to_number(a)), b); }
	if (tag_of(b) == 2) { return loose_eq(a, mk_num(to_number(b))); }
	ta = tag_of(a); tb = tag_of(b);
	if (ta == 3 && tb == 3) { return d_eq(num_bits(a), num_bits(b)); }
	if (ta == 4 && tb == 4) { return str_eq(a, b); }
	if (ta == 3 && tb == 4) { return d_eq(num_bits(a), str_to_num(b)); }
	if (ta == 4 && tb == 3) { return d_eq(str_to_num(a), num_bits(b)); }
	if ((ta == 3 || ta == 4) != (tb == 3 || tb == 4)) { return 0; }
	return strict_eq(a, b);
}

/* -1, 0, 1, or 2 for "either side is NaN" - the marker abnf/jsrt.go uses. */
int js_compare(long a, long b) {
	long an;
	long bn;
	if (tag_of(a) == 4 && tag_of(b) == 4) { return str_cmp(a, b); }
	an = to_number(a);
	bn = to_number(b);
	if (d_is_nan(an) || d_is_nan(bn)) { return 2; }
	if (d_lt(an, bn)) { return -1; }
	if (d_lt(bn, an)) { return 1; }
	return 0;
}

long js_add_v(long a, long b) {
	long ta = tag_of(a);
	long tb = tag_of(b);
	if (ta == 4 || tb == 4) { return str_cat(to_string(a), to_string(b)); }
	if (ta == 5 || tb == 5 || ta == 6 || tb == 6) { return str_cat(to_string(a), to_string(b)); }
	return mk_num(d_add(to_number(a), to_number(b)));
}

/* -------------------------------------------------------- sized integers - */
/*
 * TAG 13 - a SIZED INTEGER. This is a REAL PRIMITIVE TYPE of the C floor and
 * the twin of abnf/jsrtint.go's jsGInt, which is the authoritative spec: every
 * function below is named after the giXxx it implements and matches it.
 *
 * WHY IT IS HERE AND NOT IN LAYER 2
 * A MetaJS value cannot be a new primitive type. The best layer 2 can build is
 * an object box, and js_typeof answers "object" for that where the Go twin
 * answers "number" - a divergence between the two halves of every language that
 * carries an integer past 2^53. It is LANGUAGE NEUTRAL: jsGInt already backs
 * Go, Java, Kotlin and C#, and Lua's integer box (languages/lib/lua-rt.metajs)
 * is built on it too.
 *
 *   cell.a = the 64 bit value, ALREADY truncated to the width: sign extended
 *            for a signed width, zero extended into the low w bits for an
 *            unsigned one - except that a uint64 keeps the raw bit pattern and
 *            is READ as unsigned (si_u / si_float / si_str)
 *   cell.b = the width in bits: 8, 16, 32 or 64
 *   cell.c = 1 when the type is unsigned
 *
 * THE INVARIANT, identical in both halves (jsrtint.go's header):
 *   a plain number (tag 3)  ==  a signed 64 bit integer exact in a double
 *   a tag 13 cell           ==  every other integer: a sized or unsigned type,
 *                               or a 64 bit value outside +-2^53
 * si_norm is the ONE place it is applied.
 */

long si_make(long v, long w, long u) {
	long h = cell_new(13);
	sa(h, v); sb(h, w); sc(h, u);
	return h;
}

/* giTrunc: wrap a value into w bits with u's signedness, which is what every
 * arithmetic operator does on overflow in Go, Java, Kotlin and C#. */
long si_trunc(long v, long w, long u) {
	if (w == 8)  { v = v & 255;        if (!u && v >= 128) { v = v - 256; }               return v; }
	if (w == 16) { v = v & 65535;      if (!u && v >= 32768) { v = v - 65536; }           return v; }
	if (w == 32) { v = v & 4294967295; if (!u && v >= 2147483648) { v = v - 4294967296; } return v; }
	return v;
}

/* giU: the value read as unsigned at width w. */
unsigned long si_u(long v, long w) {
	if (w == 8)  { return (unsigned long)(v & 255); }
	if (w == 16) { return (unsigned long)(v & 65535); }
	if (w == 32) { return (unsigned long)(v & 4294967295); }
	return (unsigned long)v;
}

/* float64(uint64(v)), rounded to nearest even. d_from_long already does the
 * signed reading with bit arithmetic (the soft float's i2d loses the low bits
 * above 2^53); this is the same routine with the top bit read as a value rather
 * than as a sign, which only matters for a uint64 at or above 2^63. */
long d_from_ulong(long i) {
	long m;
	long guard;
	long sticky;
	long e = 63;
	long one = 1;
	if (i >= 0) { return d_from_long(i); }
	m = (long)(((unsigned long)i) >> 11);
	guard = (long)((((unsigned long)i) >> 10) & 1);
	sticky = ((i & ((one << 10) - 1)) != 0);
	if (guard && (sticky || (m & 1))) {
		m = m + 1;
		if (m > 9007199254740991) { m = m >> 1; e = e + 1; }
	}
	return ((e + 1023) << 52) | (m & 4503599627370495);
}

/* giFromFloat: truncate toward zero and wrap modulo 2^64, which is what a
 * conversion from a floating point value to a 64 bit integer type does.
 *
 * NOT EXACTLY MATCHABLE, and the mismatch is in the Go twin rather than here:
 * giFromFloat's last line is `int64(m)` on a value that is NaN when the operand
 * is an infinity, and Go leaves that conversion implementation defined. On this
 * machine (darwin/arm64) it answers 0, measured; on amd64 the same expression
 * answers -9223372036854775808. 0 is what is implemented here, so the two
 * halves agree on arm64 and both would differ from a Go twin built for amd64.
 * The same architecture dependence bit the float %% of phase 4a. */
long si_from_float(long b) {
	long m;
	long lim = 1086;                 /* 2^63  */
	long two63;
	long two64;
	if (d_is_nan(b)) { return 0; }
	if (d_is_inf(b)) { return 0; }   /* see the note above */
	two63 = lim << 52;
	two64 = (lim + 1) << 52;
	if (!d_lt(b, two63) || d_lt(b, two63 | (0 - 9223372036854775807 - 1))) {
		m = d_mod_go(d_trunc(b), two64);
		if (!d_lt(m, two63)) { m = d_sub(m, two64); }
		else if (d_lt(m, two63 | (0 - 9223372036854775807 - 1))) { m = d_add(m, two64); }
		if (d_is_nan(m)) { return 0; }
		return d_to_long(m);
	}
	return d_to_long(d_trunc(b));
}

/* giVal: the 64 bit value of any integral operand. A plain number truncates
 * toward zero; anything else goes through to_number first. */
long si_val(long h) {
	long t = tag_of(h);
	if (t == 13) { return fa(h); }
	if (t == 3)  { return si_from_float(num_bits(h)); }
	return si_from_float(to_number(h));
}

/* giFloat: the floating point reading, as BITS. A uint64 needs its own path -
 * int64(-1) read as a uint64 is 1.8446744073709552e19, not -1. */
long si_float(long h) {
	if (tag_of(h) == 13) {
		if (fc(h) && fb(h) == 64) { return d_from_ulong(fa(h)); }
		return d_from_long(fa(h));
	}
	return to_number(h);
}

/* The decimal text of a 64 bit magnitude. */
long si_digits(unsigned long v, int neg) {
	char out[24];
	long o = 24;
	if (v == 0) { o = 23; out[23] = 48; }
	while (v > 0) { o = o - 1; out[o] = (char)(48 + (long)(v % 10)); v = v / 10; }
	if (neg) { o = o - 1; out[o] = 45; }
	return mk_str(out + o, 24 - o);
}

/* giStr: the decimal text of a box - the one place the unsigned reading
 * matters for output. */
long si_str(long h) {
	long v = fa(h);
	if (fc(h)) { return si_digits(si_u(v, fb(h)), 0); }
	/* 0 - INT64_MIN wraps back to INT64_MIN, whose UNSIGNED reading is the
	 * magnitude 9223372036854775808 - so the negation is done unsigned. */
	if (v < 0) { return si_digits((unsigned long)(0 - v), 1); }
	return si_digits((unsigned long)v, 0);
}

/* giNorm: a plain number when the value is a signed 64 bit one a double holds
 * exactly, a box otherwise. EVERYTHING that produces an integer goes through
 * it, so the invariant above holds by construction. */
long si_norm(long v, long w, long u) {
	v = si_trunc(v, w, u);
	if (w == 64 && !u && v <= 9007199254740992 && v >= (0 - 9007199254740992)) {
		return mk_num(d_from_long(v));
	}
	return si_make(v, w, u);
}

/* giWidthOf: the result type of a binary operation. At most one operand is a
 * box in a well typed program and it decides; where both are, the LEFT wins. */
long si_width_of(long l, long r) {
	if (tag_of(l) == 13) { return fb(l); }
	if (tag_of(r) == 13) { return fb(r); }
	return 64;
}
long si_uns_of(long l, long r) {
	if (tag_of(l) == 13) { return fc(l); }
	if (tag_of(r) == 13) { return fc(r); }
	return 0;
}

/* giIsNumeric: there is no jsChar in the C floor, so a number is a plain
 * number, a sized integer (tag 13) or a boxed double (tag 14). */
int si_numeric(long h) { long t = tag_of(h); return t == 3 || t == 13 || t == 14; }

/* giEq for two integral operands: they compare by VALUE whatever their widths. */
int si_eq(long a, long b) { return si_val(a) == si_val(b); }

/* giCmp: -1, 0 or 1, unsigned when the result type is unsigned - so uint64max
 * is greater than 0 rather than -1 less than 0. */
int si_cmp(long a, long b) {
	long w = si_width_of(a, b);
	long x = si_val(a);
	long y = si_val(b);
	if (si_uns_of(a, b)) {
		unsigned long ux = si_u(x, w);
		unsigned long uy = si_u(y, w);
		if (ux < uy) { return -1; }
		if (ux > uy) { return 1; }
		return 0;
	}
	if (x < y) { return -1; }
	if (x > y) { return 1; }
	return 0;
}

/* giArith: one binary arithmetic or bitwise operator, evaluated at the result
 * type's full width and wrapped to it. `op` is a string handle, because the
 * emitters pass the operator as a compile time constant string - one extern per
 * operator would have been eleven externs. */
long si_apply(long code, long l, long r) {
	long w = si_width_of(l, r);
	long u = si_uns_of(l, r);
	/* A SHIFT is the one operator whose result type is the LEFT operand's ALONE;
	 * the count is a separate operand with a type of its own. Go says so outright
	 * ("the result type of a shift is the type of the left operand"), and so do
	 * JLS 15.19, ECMA-334 12.11, Kotlin's shl/shr/ushr (the count is always an
	 * Int) and Swift's `>> <RHS: BinaryInteger>` (the result is Self).
	 *
	 * MEASURED, 2026-08-04: this is a guard, not a repair. A fatal probe placed
	 * here and at the other three sites (giArith in abnf/jsrtint.go, szArithSlow
	 * in lib/interp-core.js, siOp in metajs-interpreter.abnf) fired ZERO times
	 * across matrix 325/325, --full 5,800 assertions, --cross 119 programs and
	 * clang-check 16/16: every language's layer 2 already hands the count over as
	 * a plain number or a 64-bit signed box (jvShift masks to & 31 / & 63,
	 * k1Shift to k1I32(r) & mask, csShift through csShiftCount, js_swshift
	 * through sintConv(n, 64, false), dart's ints are all 64-bit signed), and Go's
	 * emitter wraps the count in js_gival. Real `java` 24 and `swiftc` 6.1.2
	 * confirmed both halves already right on the shapes those two languages can
	 * actually express (int >> byte, long >> short, int << long, int >>> char,
	 * Int >> UInt8, Int64 >> UInt16, Int << UInt32, UInt8 >> Int, Int8 >> UInt64).
	 * Kotlin, C# and Dart cannot express a non-int count at all; no toolchain for
	 * those three is installed here, so that rests on the specifications named
	 * above. */
	long a = si_val(l);
	long b = si_val(r);
	long x = 0;
	long s;
	if (code == 9 || code == 10) { w = si_width_of(l, l); u = si_uns_of(l, l); }
	if (code == 0) { x = a + b; }
	else if (code == 1) { x = a - b; }
	else if (code == 2) { x = a * b; }
	else if (code == 3) {
		if (b == 0) { die("integer divide by zero"); }
		if (u) { x = (long)(si_u(a, w) / si_u(b, w)); }
		else if (a == (0 - 9223372036854775807 - 1) && b == -1) { x = a; }
		else { x = a / b; }
	}
	else if (code == 4) {
		if (b == 0) { die("integer divide by zero"); }
		if (u) { x = (long)(si_u(a, w) % si_u(b, w)); }
		else if (a == (0 - 9223372036854775807 - 1) && b == -1) { x = 0; }
		else { x = a % b; }
	}
	else if (code == 5) { x = a & b; }
	else if (code == 6) { x = a | b; }
	else if (code == 7) { x = a ^ b; }
	else if (code == 8) { x = a & (0 - b - 1); }   /* &^, i.e. a & ~b */
	else if (code == 9) {
		if (si_width_of(l, l) != w || si_uns_of(l, l) != u) { die("PROBE-C <<"); }
		/* Go's rule, and unlike C not undefined: a count at or above the width
		 * shifts everything out. A negative count does too. */
		s = b;
		if (s < 0 || s >= w) { x = 0; } else { x = a << s; }
	}
	else if (code == 10) {
		if (si_width_of(l, l) != w || si_uns_of(l, l) != u) { die("PROBE-C >>"); }
		s = b;
		if (s < 0) { x = 0; }
		else if (u) {
			if (s >= w) { x = 0; } else { x = (long)(si_u(a, w) >> s); }
		}
		else {
			if (s >= w) { s = w - 1; }
			x = a >> s;
		}
	}
	return si_norm(x, w, u);
}

long si_arith(long op, long l, long r) {
	if (str_eq_c(op, "+"))  { return si_apply(0, l, r); }
	if (str_eq_c(op, "-"))  { return si_apply(1, l, r); }
	if (str_eq_c(op, "*"))  { return si_apply(2, l, r); }
	if (str_eq_c(op, "/"))  { return si_apply(3, l, r); }
	if (str_eq_c(op, "%"))  { return si_apply(4, l, r); }
	if (str_eq_c(op, "&"))  { return si_apply(5, l, r); }
	if (str_eq_c(op, "|"))  { return si_apply(6, l, r); }
	if (str_eq_c(op, "^"))  { return si_apply(7, l, r); }
	if (str_eq_c(op, "&^")) { return si_apply(8, l, r); }
	if (str_eq_c(op, "<<")) { return si_apply(9, l, r); }
	if (str_eq_c(op, ">>")) { return si_apply(10, l, r); }
	die("js_giarith: unknown operator");
	return H_UNDEF;
}

/* ---------------------------------------------------------- boxed doubles - */
/*
 * TAG 14 - a BOXED DOUBLE, the twin of abnf/jsrtjvm.go's jsJFlo, which is the
 * authoritative spec: every function below is named after the jvmXxx it
 * implements and matches it.
 *
 * WHY IT IS HERE AND NOT IN LAYER 2, and why it is a SECOND primitive rather
 * than a case of tag 13. A statically typed language has to tell `1.0 / 3.0`
 * from `1 / 3`, which are the same two operands in a value model where every
 * number is one double: Java answers 0.3333333333333333 for the first and 0 for
 * the second. So floatness goes ON the value. Tag 13 cannot carry it - its
 * payload is an integer and si_norm's whole job is to UNBOX a value a double
 * holds exactly, which is the opposite of what a double needs. jsrtint.go says
 * the same thing from the other side: giVal reads a jsJFlo by truncating it.
 *
 *   cell.a = the double's BITS (the floor's doubles are bit patterns in a long)
 *   cell.b = the print STYLE, because that is the only thing the statically
 *            typed languages differ in and it is what jvmFloText switches on:
 *              0 floJava  1.0 -> "1.0"   1e20 -> "1.0E20"  inf -> "Infinity"
 *              1 floGo    1.0 -> "1"     1e20 -> "1e+20"   inf -> "+Inf"
 *              2 floCS    1.0 -> "1"     1e20 -> "1E+20"   inf -> "Infinity"
 *
 * THE INVARIANT, identical in both halves (jsrtint.go's header):
 *   a plain number (tag 3)  ==  an INTEGRAL type (int / long / short / byte)
 *   a tag 14 cell           ==  a double / float
 * and every operator that meets one evaluates in floating point and answers
 * one (JLS 5.6.2, binary numeric promotion). There is no normalization step:
 * unlike si_norm, a double NEVER unboxes, because 1.0 and 1 must stay
 * distinguishable - that is the entire point of the type.
 */

long go_float_str(long bits);            /* defined with the other renderings */

long jf_make(long bits, long sty) {
	long h = cell_new(14);
	sa(h, bits); sb(h, sty);
	return h;
}

/* jvmStyleOf: the style an operation's result inherits - the one of whichever
 * operand is a float, the LEFT one when both are. */
long jf_style_of(long l, long r) {
	if (tag_of(l) == 14) { return fb(l); }
	if (tag_of(r) == 14) { return fb(r); }
	return 0;
}

/* jvmFloStr: java.lang.Double.toString. Always a decimal point, the shortest
 * digit run that round-trips, plain decimal for 1e-3 <= |d| < 1e7 and
 * computerized scientific notation ("1.0E20") outside that range. The two
 * bounds are compared as DOUBLES, the way Go's `a >= 1e-3 && a < 1e7` does.
 *
 * MEASURED, not assumed: testing the decimal exponent instead (e10 >= -3 &&
 * e10 < 7) is INDISTINGUISHABLE - 0 of 51 ratchet assertions and 0 of 10,149
 * probe lines, over 24 mantissas x 61 exponents x both signs. The shortest form
 * of a double at or above 1e-3 always has e10 >= -3, so the two formulations
 * agree everywhere the probe could reach. The value test is kept because it is
 * what the Go source says, not because it was shown to matter. */
long JF_1EM3 = 4562254508917369340;      /* the bits of 1e-3 */
long JF_1EM5 = 4532020583610935537;      /* the bits of 1e-5 */

long jvm_flo_str(long bits) {
	char digs[24];
	char out[64];
	long e10;
	long nd;
	long o = 0;
	long i;
	long a;
	if (d_is_nan(bits)) { return mk_cstr("NaN"); }
	if (d_is_inf(bits)) { return d_sign(bits) ? mk_cstr("-Infinity") : mk_cstr("Infinity"); }
	if (d_is_zero(bits)) { return d_sign(bits) ? mk_cstr("-0.0") : mk_cstr("0.0"); }
	a = d_abs(bits);
	/* TWO significant digits at least, and where the shortest form has one the
	 * second is the closest two digit decimal to the value rather than a zero.
	 * It shows only among the SUBNORMALS, where the gap between neighbours is
	 * comparable to the value itself: Double.MIN_VALUE is 4.9406564584124654E
	 * -324, which java renders "4.9E-324" and the shortest-plus-".0" rule
	 * renders "5.0E-324". Measured against real java 24.0.2; it was 254 lines
	 * of the 17,674 line java probe (docs/runtime-next-plan.md part 3). Every
	 * NORMAL double is within about 1e-16 of its one digit form, a thousandth
	 * of a two digit step, so the two rules agree everywhere else. */
	g_mindig = 2;
	e10 = shortest_digits(a, digs);
	g_mindig = 1;
	nd = g_ndig;
	if (d_sign(bits)) { out[o] = 45; o = o + 1; }
	if (!d_lt(a, JF_1EM3) && d_lt(a, d_from_long(10000000))) {
		/* The PLAIN form prints the number, not the two digit significand:
		 * 0.001 is "0.001" and not "0.0010", while 100.0 still gets its forced
		 * ".0" from the branch below. So the padding zero is dropped again
		 * here - the minimum is about the SIGNIFICAND, and only the scientific
		 * form shows one. */
		while (nd > 1 && digs[nd - 1] == 48) { nd = nd - 1; }
		if (e10 >= 0) {
			i = 0;
			while (i <= e10) { out[o] = i < nd ? digs[i] : 48; o = o + 1; i = i + 1; }
			out[o] = 46; o = o + 1;
			if (nd > e10 + 1) {
				i = e10 + 1;
				while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
			} else { out[o] = 48; o = o + 1; }   /* the forced ".0" */
		} else {
			out[o] = 48; o = o + 1;
			out[o] = 46; o = o + 1;
			i = -1;
			while (i > e10) { out[o] = 48; o = o + 1; i = i - 1; }
			i = 0;
			while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		}
	} else {
		/* "1.0E20" / "1.5E-8": a mantissa that always has a point, and an
		 * exponent with no leading zeros, no '+' and no minimum width. */
		long ex = e10;
		long en;
		char er[8];
		out[o] = digs[0]; o = o + 1;
		out[o] = 46; o = o + 1;
		if (nd > 1) {
			i = 1;
			while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		} else { out[o] = 48; o = o + 1; }
		out[o] = 69; o = o + 1;               /* 'E' */
		if (ex < 0) { out[o] = 45; o = o + 1; ex = 0 - ex; }
		en = 0;
		if (ex == 0) { er[0] = 48; en = 1; }
		while (ex > 0) { er[en] = (char)(48 + (ex % 10)); ex = ex / 10; en = en + 1; }
		i = en - 1;
		while (i >= 0) { out[o] = er[i]; o = o + 1; i = i - 1; }
	}
	return mk_str(out, o);
}

/* csFloStr: C#'s double.ToString() under the invariant culture - the shortest
 * round-tripping digits, no trailing ".0" for an integral value, and scientific
 * notation with a capital E and a two-digit minimum exponent outside
 * [1e-5, 1e15). */
long cs_flo_str(long bits) {
	char digs[24];
	char out[64];
	long e10;
	long nd;
	long o = 0;
	long i;
	long a;
	if (d_is_nan(bits)) { return mk_cstr("NaN"); }
	if (d_is_inf(bits)) { return d_sign(bits) ? mk_cstr("-Infinity") : mk_cstr("Infinity"); }
	if (d_is_zero(bits)) { return d_sign(bits) ? mk_cstr("-0") : mk_cstr("0"); }
	a = d_abs(bits);
	e10 = shortest_digits(a, digs);
	nd = g_ndig;
	if (d_sign(bits)) { out[o] = 45; o = o + 1; }
	if (!d_lt(a, JF_1EM5) && d_lt(a, d_from_long(1000000000000000))) {
		if (e10 >= 0) {
			i = 0;
			while (i <= e10) { out[o] = i < nd ? digs[i] : 48; o = o + 1; i = i + 1; }
			if (nd > e10 + 1) {
				out[o] = 46; o = o + 1;
				i = e10 + 1;
				while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
			}
		} else {
			out[o] = 48; o = o + 1;
			out[o] = 46; o = o + 1;
			i = -1;
			while (i > e10) { out[o] = 48; o = o + 1; i = i - 1; }
			i = 0;
			while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		}
	} else {
		long ex = e10;
		out[o] = digs[0]; o = o + 1;
		if (nd > 1) {
			out[o] = 46; o = o + 1;
			i = 1;
			while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		}
		out[o] = 69; o = o + 1;               /* 'E' */
		if (ex < 0) { out[o] = 45; o = o + 1; ex = 0 - ex; } else { out[o] = 43; o = o + 1; }
		if (ex < 10) { out[o] = 48; o = o + 1; out[o] = (char)(48 + ex); o = o + 1; }
		else {
			char er[8];
			long en = 0;
			while (ex > 0) { er[en] = (char)(48 + (ex % 10)); ex = ex / 10; en = en + 1; }
			i = en - 1;
			while (i >= 0) { out[o] = er[i]; o = o + 1; i = i - 1; }
		}
	}
	return mk_str(out, o);
}

/* jvmFloText: a boxed float rendered in ITS OWN language's style. */
long jf_text(long h) {
	long sty = fb(h);
	if (sty == 1) { return go_float_str(fa(h)); }
	if (sty == 2) { return cs_flo_str(fa(h)); }
	return jvm_flo_str(fa(h));
}

/* jvmNumEq: Java's == with a double on one side. The other side has to be a
 * number too - another box or a plain number (there is no jsChar here) - and
 * then the comparison is numeric, so 1.0 == 1 is true and NaN == NaN is
 * false. Anything else, a SIZED INTEGER included, is false. */
int jf_num_eq(long f, long other) {
	long t = tag_of(other);
	if (t == 14) { return d_eq(f, fa(other)); }
	if (t == 3)  { return d_eq(f, num_bits(other)); }
	return 0;
}

/* jvmArith: one binary arithmetic operator. A double on either side makes the
 * whole operation floating point and its result a double; two integral operands
 * keep the 32 bit wrap the compiler emits for them. The operator arrives as the
 * same 0..4 index si_apply uses for its first five, so the two boxes take one
 * number rather than an operator string across the boundary. */
long jf_arith(long code, long l, long r) {
	long a = to_number(l);
	long b = to_number(r);
	long x = 0;                    /* +0.0: jvmArith's `var x float64` with no
	                                * arm taken, so an operator index the table
	                                * does not have answers 0 in both halves */
	if (code == 0) { x = d_add(a, b); }
	else if (code == 1) { x = d_sub(a, b); }
	else if (code == 2) { x = d_mul(a, b); }
	else if (code == 3) { x = d_div(a, b); }
	else if (code == 4) { x = d_mod_go(a, b); }
	if (tag_of(l) == 14 || tag_of(r) == 14) { return jf_make(x, jf_style_of(l, r)); }
	return mk_num(d_from_long(to_int32(x)));
}

/* jvmMinMax: the operand this picked, but the DOUBLE overload of
 * Math.max/min is selected as soon as one side is a double, so the answer is a
 * double then - Math.max(1.5, 2) is 2.0. */
long jf_minmax(long l, long r, int take_l) {
	long v = take_l ? l : r;
	if (tag_of(l) == 14 || tag_of(r) == 14) {
		if (tag_of(v) == 14) { return v; }
		return jf_make(to_number(v), jf_style_of(l, r));
	}
	return v;
}

/* jvmTakeL: the OPERAND CHOICE of Math.max / Math.min, and the two cases a
 * plain `>` / `<` cannot see. java.lang.Math documents both for the double
 * overloads and java 24.0.2 confirms each: Math.min(-0.0, 0.0) is -0.0 and
 * Math.max(-0.0, 0.0) is 0.0 ("if one argument is positive zero and the other
 * negative zero, the result of min is negative zero", and of max positive
 * zero), and Math.min(NaN, 1.0) is NaN ("if either value is NaN, then the
 * result is NaN"). Both comparisons are FALSE, so a bare `<` took the right
 * operand and answered 0.0 and 1.0. Everything else is the ordinary
 * comparison, a tie still going to the right operand. */
int jf_take_l(long a, long b, int want_max) {
	if (d_is_nan(a)) { return 1; }             /* the left one is the NaN */
	if (d_is_nan(b)) { return 0; }             /* the right one is */
	if (d_is_zero(a) && d_is_zero(b)) {
		/* Both zero and only the sign bit tells them apart: min wants the
		 * negative one, max the positive one. */
		return want_max ? d_sign(b) : d_sign(a);
	}
	return want_max ? d_lt(b, a) : d_lt(a, b);
}

/* jvmMathObject's abs: a double keeps its box (and its style), everything else
 * wraps to 32 bits the way the compiler's integer arithmetic does. */
long jf_abs(long v) {
	if (tag_of(v) == 14) { return jf_make(d_abs(fa(v)), fb(v)); }
	/* A SIGNED 64 bit box is negated IN THE BOX. The fallthrough's to_int32
	 * truncates a long, so jf_abs(3000000000L) answered -1294967296 and
	 * jf_abs(Long.MAX_VALUE) answered 0. Two's-complement negation maps
	 * Long.MIN_VALUE to itself, which is the answer JLS 15.15.4 asks for, and
	 * an UNSIGNED box is a magnitude already. This is the third engine of the
	 * primitive: abnf/jsrtjvm.go's floAbs and languages/metajs-interpreter.abnf's
	 * "floAbs" carry the same three lines, and si_norm is the shape the floor's
	 * own negation uses (see js_gineg). */
	if (tag_of(v) == 13 && fb(v) == 64) {
		if (fc(v)) { return v; }
		if (fa(v) < 0) { return si_norm(0 - fa(v), 64, 0); }
		return v;
	}
	return mk_num(d_from_long(to_int32(d_abs(to_number(v)))));
}

/* ---------------------------------------------------------------- scopes - */

/* A string's CONTENT hash, memoised in the cell's field f.
 *
 * A string uses a = bytes, b = byte length, c = the UTF-16 unit buffer, d = the
 * unit count and e = the unit kind; f is unused, and gc_trace's tag-4 arm only
 * ever looks at a and c, so a raw long parked in f is invisible to the
 * collector. Memoising matters because the caller is scope_find: a name is
 * hashed once per string CELL for the life of the program, and every module-scope
 * lookup after that is a load.
 *
 * It has to be a CONTENT hash, not the handle. js_str_mem interns one cell per
 * literal ADDRESS, so within one module every occurrence of a name is the same
 * handle - but a native build links TWO modules (the program and lib/<lang>-rt.ll),
 * and the same name spelled in both has two different cells. str_eq compares
 * bytes for exactly that reason, and so must this.
 *
 * FNV-1a, kept inside 32 bits so the value does not depend on how the multiply
 * overflows, and forced non-zero so that 0 can stay the "not computed" marker. */
long str_hash(long h) {
	long v = ff(h);
	const char *p;
	long n;
	long i = 0;
	if (v != 0) { return v; }
	p = str_ptr(h);
	n = str_len(h);
	v = 2166136261;
	while (i < n) {
		v = ((v ^ ((long)p[i] & 255)) * 16777619) & 4294967295;
		i = i + 1;
	}
	if (v == 0) { v = 1; }
	sf(h, v);
	return v;
}

/* A scope cell: a = names buffer (string handles), b = values buffer,
 * c = type-class buffer, d = count, e = capacity, f = parent handle (0 at the
 * root). Handle 0 as a scope argument means the ROOT scope, matching
 * abnf/jsrt.go's scopeOf. */
long G_ROOT;

/* How many extra words of hash table a scope of capacity `cap` carries, 0 for
 * none: an open-addressed table of 2*cap slots, each holding an entry's index
 * plus one, so a 0 means empty.
 *
 * SMALL SCOPES ARE DELIBERATELY NOT INDEXED. A call or block scope holds a
 * handful of names, is allocated by the million, and its whole name array is one
 * or two cache lines - a scan of it is already as fast as a hash probe. What the
 * index is for is the MODULE scope: hundreds of entries, searched by every
 * global reference in the program, and it stops growing after boot.
 *
 * cap is always 0 or a power of two >= 4 (scope_put doubles from 4), so the slot
 * count is a power of two and the mask below is exact. Two slots per capacity
 * word keeps the load factor at or below 1/2, which is also what guarantees the
 * probe loop terminates: there is always an empty slot.
 *
 * A FATTER SLOT WAS TRIED AND IS WORSE. Carrying the name handle beside the
 * index makes a hit one memory access instead of two (the slot, then
 * names[idx]), which looked like the reason a small module lost. Measured: lua
 * +6.1% against +3.9%, kotlin -40.4% against -42.1%. */
long scope_hwords(long cap) {
	if (cap < 32) { return 0; }
	return cap * 2;
}

/* Put entry `idx` into the table that follows the three cap-sized thirds of the
 * block. */
void scope_hput(long *names, long cap, long idx) {
	long *tab = names + cap * 3;
	long m = cap * 2 - 1;
	long slot = str_hash(names[idx]) & m;
	while (tab[slot] != 0) { slot = (slot + 1) & m; }
	tab[slot] = idx + 1;
}

/* Capacity 4, not 8. A scope is allocated on EVERY function call, so its size is
 * the per-call cost - and it was the growth rate of a long-running program too
 * back when nothing was freed: 8 cost 248 bytes a call, 4 costs 152. Almost every scope
 * holds a handful of names (parameters plus a few locals) and scope_put doubles
 * when it does not. Measured on a 2M-iteration Lua loop: the resident set and the
 * system time both fell by about a third. */
/* A scope now starts EMPTY - capacity 0, no buffers - and scope_put allocates
 * the first time a name is declared in it.
 *
 * The reason is measured. lib/compile-core.js's makeBlockStmt opens a scope for
 * every BLOCK, not only for every call, so a `s = s + i %% 7` Lua loop through
 * layer 2 allocated 33 scopes per iteration against 9 calls: roughly three
 * quarters of them are block scopes that never declare a single name, and each
 * was paying for three four-slot buffers it never wrote to. An empty scope costs
 * one 56-byte cell and one arena call instead of four calls and 152 bytes.
 *
 * The three buffers are also ONE allocation rather than three: names, values and
 * type classes are the three consecutive `cap`-sized thirds of a single block,
 * so a scope that does declare costs two arena calls, not four. */
long mk_scope(long parent) {
	long h = cell_new(11);
	sa(h, 0);
	sb(h, 0);
	sc(h, 0);
	sd(h, 0);
	se(h, 0);
	sf(h, parent);
	return h;
}
long scope_of(long h) { if (h == 0) { return G_ROOT; } return h; }

/* THE HOTTEST LOOP IN THE RUNTIME. Every global reference in every one of the
 * sixteen languages ends here, and until the hash arm it was a LINEAR SCAN of
 * the module scope: 461 entries for a kotlin program, walked by every `+`.
 * Measured on kotlin (40,000 x `s = s + i %% 7`, native -exe, instructions
 * retired) the whole collector was 5.09 G of 30.5 G and this scan was worth more
 * - and it made the POSITION of a declaration matter, which is what
 * docs/runtime-merge-plan.md's "live-body tax" turned out to be.
 *
 * The answer is unchanged in every case: the table indexes the same array and
 * the winner is still the FIRST matching entry, because scope_put never inserts
 * a duplicate (it assigns instead) and nothing ever removes an entry, so at most
 * one index in the table can match. js_scope_get / scopeHas / js_scope_has
 * therefore see exactly what they saw before.
 *
 * The arm is chosen on n, which the linear scan needs anyway, and NOT on the
 * capacity: reading fe(s) here as well is another load on the path every small
 * scope takes, and it measured (lua +2.5% for that alone). n <= cap and cap is a
 * power of two, so n >= 32 guarantees cap >= 32 and therefore that scope_hwords
 * gave this scope a table.
 *
 * THE ARM IS NOT FREE FOR A LANGUAGE THAT CANNOT USE IT, and the reason is
 * inlining. At HEAD this function is small enough that clang inlines it into
 * scope_get, scope_put and js_tdecl; a second arm makes it too big, and lua -
 * whose 84-declaration module scope is the smallest of the sixteen and whose
 * benchmark is dominated by SMALL scopes - pays +3.9%. Isolated: a bare
 * `if (n >= 1000000) return <call>` that never fires already costs it +2.8%, so
 * it is the shape of the function and not the hashing. Splitting the arm into
 * its own function was measured and is WORSE (+4.7%), as is carrying the name
 * handle in the slot (+6.1%). It is kept because the other side of the trade is
 * -41% on python, ruby and kotlin. */
long scope_find(long s, long name) {
	long *names = (long *)fa(s);
	long n = fd(s);
	long i = 0;
	if (n >= 32) {
		long cap = fe(s);
		long *tab = names + cap * 3;
		long m = cap * 2 - 1;
		long slot = str_hash(name) & m;
		while (tab[slot] != 0) {
			long idx = tab[slot] - 1;
			if (str_eq(names[idx], name)) { return idx; }
			slot = (slot + 1) & m;
		}
		return -1;
	}
	while (i < n) { if (str_eq(names[i], name)) { return i; } i = i + 1; }
	return -1;
}
void scope_put(long s, long name, long v) {
	long i = scope_find(s, name);
	long *names = (long *)fa(s);
	long *vals = (long *)fb(s);
	long *tcs = (long *)fc(s);
	long n = fd(s);
	long cap = fe(s);
	if (i >= 0) { vals[i] = v; return; }
	if (n >= cap) {
		long nc = cap * 2;
		long hc;
		long *base;
		long k = 0;
		if (nc < 4) { nc = 4; }
		hc = scope_hwords(nc);
		/* One allocation for all four arrays. buf_new zeroes, so the table
		 * arrives empty; nothing else may allocate before the store below. */
		base = buf_new(nc * 3 + hc);
		while (k < n) {
			base[k] = names[k];
			base[nc + k] = vals[k];
			base[nc + nc + k] = tcs[k];
			k = k + 1;
		}
		names = base;
		vals = base + nc;
		tcs = base + nc + nc;
		sa(s, (long)names);
		sb(s, (long)vals);
		sc(s, (long)tcs);
		se(s, nc);
		cap = nc;
		/* The table is rebuilt from scratch on every growth - the slot of an
		 * entry depends on the mask, so the old one cannot be copied. It is
		 * amortised: doubling makes this O(n) total over the scope's life. */
		if (hc != 0) { k = 0; while (k < n) { scope_hput(names, cap, k); k = k + 1; } }
	}
	names[n] = name;
	vals[n] = v;
	tcs[n] = 0;
	sd(s, n + 1);
	/* Spelled out rather than through scope_hwords: this runs on EVERY
	 * declaration in every scope in the program, and the overwhelming majority
	 * of them are small ones that have no table. */
	if (cap >= 32) { scope_hput(names, cap, n); }
}
void scope_pin(long s, long i, long tc) { long *tcs = (long *)fc(s); tcs[i] = tc; }
long scope_tc(long s, long i) { long *tcs = (long *)fc(s); return tcs[i]; }

void wstr(long h) { write(2, str_ptr(h), (unsigned long)str_len(h)); }

void die2(const char *msg, long h) {
	werr("js runtime error: "); werr(msg); wstr(to_string(h)); werr("\n");
	exit(1);
}

/* "member 'k' of undefined" and friends: a prefix, the KEY, a middle and the
 * OBJECT, which is the shape rt.fail uses for the member diagnostics. */
void die3(const char *pre, long key, const char *mid, long obj) {
	werr("js runtime error: "); werr(pre); wstr(to_string(key));
	werr(mid); wstr(to_string(obj)); werr("\n");
	exit(1);
}

long scope_get(long s, long name) {
	long cur = s;
	while (cur != 0) {
		long i = scope_find(cur, name);
		long *vals;
		if (i >= 0) { vals = (long *)fb(cur); return vals[i]; }
		cur = ff(cur);
	}
	die2("variable not defined: ", name);
	return H_UNDEF;
}

/* host_scope: the scope argument of the scope* HOST BUILTINS (ids 65..68).
 *
 * It is deliberately NOT scope_of(). scope_of maps the handle 0 to G_ROOT,
 * because an emitter writes 0 for "the global scope"; a host builtin never sees
 * that handle - its arguments are VALUES, so a caller writing scopeNew(0) hands
 * over a tag 3 number cell, not the handle 0. Anything that is not a tag 11
 * cell is an error here, which is what the two twins (abnf/jsrtint.go's
 * scopeBindings and metajs-interpreter.abnf's scHas) do as well; only scopeNew
 * takes undefined, for "no parent". The one thing a layer-2 file DOES receive
 * is an emitter's real scope handle, and that is a tag 11 cell and arrives
 * here unchanged. */
long host_scope(long h) {
	if (tag_of(h) == 11) { return h; }
	die2("not a scope: ", type_of(h));
	return 0;
}

/* ---------------------------------------------------------- host builtins */

/* Host function ids. A host function cell carries the id in field a. */
long mk_host(long id) { long h = cell_new(8); sa(h, id); return h; }

/* --------------------------------------------------------- bound methods - */

long mk_bound(long recv, long mid) { long h = cell_new(9); sa(h, recv); sb(h, mid); return h; }

/* Method ids, grouped by receiver:
 *   1..9   array   push pop shift unshift reverse slice indexOf join concat
 *   20..29 string  charCodeAt charAt indexOf replace slice substring split
 *                  toUpperCase toLowerCase trim
 *   40..41 function apply call
 */
long method_id_array(long name) {
	if (str_eq_c(name, "push"))    { return 1; }
	if (str_eq_c(name, "pop"))     { return 2; }
	if (str_eq_c(name, "shift"))   { return 3; }
	if (str_eq_c(name, "unshift")) { return 4; }
	if (str_eq_c(name, "reverse")) { return 5; }
	if (str_eq_c(name, "slice"))   { return 6; }
	if (str_eq_c(name, "indexOf")) { return 7; }
	if (str_eq_c(name, "join"))    { return 8; }
	if (str_eq_c(name, "concat"))  { return 9; }
	return 0;
}
long method_id_string(long name) {
	if (str_eq_c(name, "charCodeAt"))  { return 20; }
	if (str_eq_c(name, "charAt"))      { return 21; }
	if (str_eq_c(name, "indexOf"))     { return 22; }
	if (str_eq_c(name, "replace"))     { return 23; }
	if (str_eq_c(name, "slice"))       { return 24; }
	if (str_eq_c(name, "substring"))   { return 25; }
	if (str_eq_c(name, "split"))       { return 26; }
	if (str_eq_c(name, "toUpperCase")) { return 27; }
	if (str_eq_c(name, "toLowerCase")) { return 28; }
	if (str_eq_c(name, "trim"))        { return 29; }
	return 0;
}

/* ------------------------------------------------------- member access --- */

/* maybeNumeric, matching abnf/jsrt.go: a non-string key is always an index
 * candidate, and a string one is only when it could parse as a number. */
int maybe_numeric(long key) {
	int c;
	if (tag_of(key) != 4) { return 1; }
	if (str_len(key) == 0) { return 1; }
	c = (int)str_ptr(key)[0] & 255;
	return c <= 32 || (c >= 48 && c <= 57) || c == 43 || c == 45 || c == 46 ||
	       c == 73 || c == 105 || c == 78 || c == 110;
}

/* An index that is a non-negative integer, or -1. */
long index_of_key(long key) {
	long n = to_number(key);
	if (d_is_nan(n) || d_is_inf(n)) { return -1; }
	if (!d_is_integral(n)) { return -1; }
	if (!d_in_long(n)) { return -1; }
	{ long i = d_to_long(n); if (i < 0) { return -1; } return i; }
}

int is_callable(long h) { long t = tag_of(h); return t == 7 || t == 8 || t == 9 || t == 16; }

long get_member(long obj, long key) {
	long t = tag_of(obj);
	long name;
	long idx;
	if (t == 0 || t == 1) { die3("member '", key, "' of ", obj); }
	if (tag_of(key) == 4 && is_callable(obj)) {
		if (str_eq_c(key, "apply")) { return mk_bound(obj, 40); }
		if (str_eq_c(key, "call"))  { return mk_bound(obj, 41); }
	}
	if (t == 6) { return obj_get(obj, to_string(key)); }
	/* A generator's iterator protocol. The floor owns `next` only; current /
	 * key / valid / send / getReturn are layer 2 over it (see the cost table in
	 * docs/runtime-next-plan.md). */
	if (t == 15 && tag_of(key) == 4 && str_eq_c(key, "next")) { return mk_bound(obj, 60); }
	/* g.return(v): the CLOSING half of the iterator protocol. It is answered
	 * here and not left to layer 2 because a generator is a floor cell - nothing
	 * can be stored on one - so python-rt.metajs used to keep a closed-ness side
	 * table that the native for-loop never consulted. See gen_close. */
	if (t == 15 && tag_of(key) == 4 && str_eq_c(key, "return")) { return mk_bound(obj, 61); }
	if (t == 5) {
		if (tag_of(key) == 4) {
			long mid;
			if (str_eq_c(key, "length")) { return mk_num(d_from_long(arr_len(obj))); }
			mid = method_id_array(key);
			if (mid != 0) { return mk_bound(obj, mid); }
		}
		if (!maybe_numeric(key)) { return H_UNDEF; }
		idx = index_of_key(key);
		if (idx >= 0 && idx < arr_len(obj)) { return arr_get(obj, idx); }
		return H_UNDEF;
	}
	if (t == 4) {
		if (tag_of(key) == 4) {
			long mid;
			if (str_eq_c(key, "length")) { return mk_num(d_from_long(str_ulen(obj))); }
			mid = method_id_string(key);
			if (mid != 0) { return mk_bound(obj, mid); }
		}
		if (!maybe_numeric(key)) { return H_UNDEF; }
		idx = index_of_key(key);
		if (idx >= 0 && idx < str_ulen(obj)) { return u_slice(obj, idx, idx + 1); }
		return H_UNDEF;
	}
	name = to_string(key);
	return H_UNDEF;
}

void set_member(long obj, long key, long val) {
	long t = tag_of(obj);
	long idx;
	if (t == 0 || t == 1) { die3("member assignment '", key, "' on ", obj); }
	if (t == 6) { obj_put(obj, to_string(key), val); return; }
	if (t == 5) {
		if (tag_of(key) == 4 && str_eq_c(key, "length")) {
			long n = to_number(val);
			long ln;
			if (!d_is_integral(n) || d_sign(n)) { die2("invalid array length ", val); }
			ln = d_to_long(n);
			while (arr_len(obj) > ln) { sb(obj, arr_len(obj) - 1); }
			while (arr_len(obj) < ln) { arr_push(obj, H_UNDEF); }
			return;
		}
		if (!maybe_numeric(key)) { die2("invalid array index ", key); }
		idx = index_of_key(key);
		if (idx < 0) { die2("invalid array index ", key); }
		while (arr_len(obj) <= idx) { arr_push(obj, H_UNDEF); }
		arr_set(obj, idx, val);
		return;
	}
	die3("member assignment '", key, "' on ", obj);
}

/* ------------------------------------------------------------- printing - */

long OUTFD;   /* 1 for println/print, 2 for eprintln */

long from_char_code(long args);
long arg_at(long args, long i);
long arg_num(long args, long i);

void o_ch(int c) { if (OUTFD == 1) { putchar(c); } else { char b[1]; b[0] = (char)c; write(2, b, 1); } }
void o_str(long s) {
	long n = str_len(s);
	const char *p = str_ptr(s);
	long i = 0;
	if (OUTFD == 1) { while (i < n) { putchar((int)p[i]); i = i + 1; } }
	else { write(2, p, (unsigned long)n); }
}
void o_cstr(const char *s) { long i = 0; while (s[i] != 0) { o_ch((int)s[i]); i = i + 1; } }

/* Go's %v of a float64, which is what println feeds fmt for a number that is
 * NOT an integer inside the int64 range: shortest digits, and the exponent form
 * when the decimal exponent is below -4 or at least 6. */
long go_float_str(long bits) {
	char digs[24];
	char out[64];
	long e10;
	long nd;
	long o = 0;
	long i;
	if (d_is_nan(bits)) { return mk_cstr("NaN"); }
	if (d_is_inf(bits)) { return d_sign(bits) ? mk_cstr("-Inf") : mk_cstr("+Inf"); }
	if (d_is_zero(bits)) { return d_sign(bits) ? mk_cstr("-0") : mk_cstr("0"); }
	e10 = shortest_digits(bits, digs);
	nd = g_ndig;
	if (d_sign(bits)) { out[o] = 45; o = o + 1; }
	if (e10 < -4 || e10 >= 6) {
		long ex = e10;
		out[o] = digs[0]; o = o + 1;
		if (nd > 1) {
			out[o] = 46; o = o + 1;
			i = 1;
			while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		}
		out[o] = 101; o = o + 1;
		if (ex < 0) { out[o] = 45; o = o + 1; ex = 0 - ex; } else { out[o] = 43; o = o + 1; }
		if (ex < 10) { out[o] = 48; o = o + 1; out[o] = (char)(48 + ex); o = o + 1; }
		else {
			char er[8];
			long en = 0;
			while (ex > 0) { er[en] = (char)(48 + (ex % 10)); ex = ex / 10; en = en + 1; }
			i = en - 1;
			while (i >= 0) { out[o] = er[i]; o = o + 1; i = i - 1; }
		}
	} else if (e10 >= 0) {
		i = 0;
		while (i <= e10) { out[o] = i < nd ? digs[i] : 48; o = o + 1; i = i + 1; }
		if (nd > e10 + 1) {
			out[o] = 46; o = o + 1;
			i = e10 + 1;
			while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		}
	} else {
		out[o] = 48; o = o + 1;
		out[o] = 46; o = o + 1;
		i = -1;
		while (i > e10) { out[o] = 48; o = o + 1; i = i - 1; }
		i = 0;
		while (i < nd) { out[o] = digs[i]; o = o + 1; i = i + 1; }
	}
	return mk_str(out, o);
}

/* fmt_val is Go's fmt %v of what toGoNatural makes of a value - which is what
 * println / print write, and it is NOT the same as to_string: undefined renders
 * as <nil>, an array as [a b c], an object as map[k:v] with SORTED keys, an
 * integral number as its digits and any other number through Go's float
 * formatting. */
long fmt_val(long h) {
	long t = tag_of(h);
	if (t == 0 || t == 1) { return mk_cstr("<nil>"); }
	if (t == 2) { return fa(h) ? mk_cstr("true") : mk_cstr("false"); }
	if (t == 4) { return h; }
	if (t == 3) {
		long b = num_bits(h);
		if (d_is_integral(b) && d_in_long(b) && !(d_is_zero(b) && d_sign(b))) {
			char out[32];
			long o = 0;
			if (d_sign(b)) { out[o] = 45; o = o + 1; }
			o = o + int_digits(b, out + o);
			return mk_str(out, o);
		}
		return go_float_str(b);
	}
	/* A sized integer reaches toGoNatural as an int64 / uint64, so %v is its
	 * digits (abnf/jsrt.go toGoNatural, `case jsGInt`). */
	if (t == 13) { return si_str(h); }
	/* A boxed double reaches toGoNatural as its own language's float text
	 * (abnf/jsrt.go toGoNatural, `case jsJFlo: jvmFloText(t)`), so print and
	 * println show "1.0" for a Java double and "1" for a Go one. */
	if (t == 14) { return jf_text(h); }
	if (t == 5) {
		long n = arr_len(h);
		long i = 0;
		long out = mk_cstr("[");
		while (i < n) {
			if (i > 0) { out = str_cat(out, mk_cstr(" ")); }
			out = str_cat(out, fmt_val(arr_get(h, i)));
			i = i + 1;
		}
		return str_cat(out, mk_cstr("]"));
	}
	if (t == 6) {
		/* Go prints a map with its keys SORTED, so this has to as well. */
		long *keys = (long *)fa(h);
		long *vals = (long *)fb(h);
		long n = fc(h);
		long *ord = buf_new(n + 1);
		long i = 0;
		long j;
		long out = mk_cstr("map[");
		while (i < n) { ord[i] = i; i = i + 1; }
		i = 1;
		while (i < n) {
			long v = ord[i];
			j = i - 1;
			while (j >= 0 && str_cmp(keys[ord[j]], keys[v]) > 0) { ord[j + 1] = ord[j]; j = j - 1; }
			ord[j + 1] = v;
			i = i + 1;
		}
		i = 0;
		while (i < n) {
			if (i > 0) { out = str_cat(out, mk_cstr(" ")); }
			out = str_cat(out, keys[ord[i]]);
			out = str_cat(out, mk_cstr(":"));
			out = str_cat(out, fmt_val(vals[ord[i]]));
			i = i + 1;
		}
		return str_cat(out, mk_cstr("]"));
	}
	return mk_cstr("[function]");
}

/* fmt_top is fmt_val at the TOP level of a print: Go's printArgs runs every
 * string operand through wtf8Clean, so a lone surrogate shows as U+FFFD. */
long fmt_top(long h) { if (tag_of(h) == 4) { return wtf8_clean(h); } return fmt_val(h); }

void print_val(long h) { o_str(fmt_top(h)); }

/* The digits of an integral value, for %d. */
long int_str(long bits) {
	char out[32];
	long o = 0;
	if (d_is_nan(bits) || d_is_inf(bits) || !d_in_long(bits)) { return fmt_val(mk_num(bits)); }
	bits = d_trunc(bits);
	if (d_sign(bits)) { out[o] = 45; o = o + 1; }
	o = o + int_digits(bits, out + o);
	return mk_str(out, o);
}

/* fmt_apply is the printf/sprintf verb subset this runtime supports:
 * %d %s %c %v %f %% . Width, precision and the exotic verbs are NOT supported -
 * a MetaJS program that needs them gets the verb back verbatim, which is a
 * visible wrong answer rather than a silent one. */
long fmt_apply(long f, long args, long firstArg) {
	const char *p = str_ptr(f);
	long n = str_len(f);
	long i = 0;
	long ai = firstArg;
	long out = mk_cstr("");
	char lit[2];
	while (i < n) {
		int ch = (int)p[i] & 255;
		if (ch != 37) { lit[0] = (char)ch; out = str_cat(out, mk_str(lit, 1)); i = i + 1; }
		else if (i + 1 < n) {
			int v = (int)p[i + 1] & 255;
			i = i + 2;
			if (v == 37) { out = str_cat(out, mk_cstr("%")); }
			else if (v == 100) { out = str_cat(out, int_str(arg_num(args, ai))); ai = ai + 1; }
			else if (v == 115) { out = str_cat(out, fmt_top(arg_at(args, ai))); ai = ai + 1; }
			else if (v == 118) { out = str_cat(out, fmt_top(arg_at(args, ai))); ai = ai + 1; }
			else if (v == 102) { out = str_cat(out, num_to_str(arg_num(args, ai))); ai = ai + 1; }
			else if (v == 99) {
				long a = mk_arr();
				arr_push(a, arg_at(args, ai));
				out = str_cat(out, from_char_code(a));
				ai = ai + 1;
			} else {
				lit[0] = 37; out = str_cat(out, mk_str(lit, 1));
				lit[0] = (char)v; out = str_cat(out, mk_str(lit, 1));
			}
		} else { lit[0] = (char)ch; out = str_cat(out, mk_str(lit, 1)); i = i + 1; }
	}
	return out;
}

/* Go's fmt.Sprint: a space goes between two operands when NEITHER is a string. */
long fmt_sprint(long args) {
	long out = mk_cstr("");
	long i = 0;
	while (i < arr_len(args)) {
		if (i > 0 && tag_of(arr_get(args, i - 1)) != 4 && tag_of(arr_get(args, i)) != 4) {
			out = str_cat(out, mk_cstr(" "));
		}
		out = str_cat(out, fmt_top(arr_get(args, i)));
		i = i + 1;
	}
	return out;
}

/* ---------------------------------------------------------- control flow - */

long mk_ctl(long kind, long value) { long h = cell_new(10); sa(h, kind); sb(h, value); return h; }

/* One jmp_buf per DEPTH, allocated once and reused. It used to be a fresh
 * malloc(512) per js_try - three of them for a try/catch/finally - and nothing
 * ever frees it, so a `try { throw } catch {} finally {}` in a loop leaked about
 * 1.5 KB per iteration: 200,000 of them cost 499 MB, of which this was 261 MB.
 * A buffer at depth d is only live while JB_DEPTH > d, and js_try restores
 * JB_DEPTH before it returns, so two try statements at the same depth are
 * sequential and can share the buffer. THE SETJMP MUST STILL BE DONE ONCE PER
 * ENTRY - only the storage is reused.
 *
 * THE POOL IS PER STACK, not per process. A generator body runs on its own C
 * stack, so a setjmp it takes is only valid on that stack and a longjmp to a
 * buffer the RESUMER filled in would jump into another thread's frame - the
 * undefined behaviour docs/runtime-next-plan.md named as the gate on this
 * primitive. JBP therefore points at the pool of the stack that is RUNNING:
 * main's for main, and cb[9] for a coroutine, swapped by whichever side gains
 * control (exactly as GC_STACK_BASE is). JB_CAP is that pool's length. */
long JBP;            /* long *: the running stack's jmp_buf pool */
long JB_CAP;         /* its length in slots */
long JB_MAIN;        /* main's own pool, so the swap has something to go back to */
long JB_DEPTH;

long jb_at(long d) {
	long *p = (long *)JBP;
	if (d >= JB_CAP) { die("try nested too deeply"); }
	if (p[d] == 0) { p[d] = (long)malloc(512); }
	return p[d];
}

/* A jmp_buf pool as a ROOT RANGE: the saved registers in a buffer can be the
 * only copy of a handle between a setjmp and the longjmp that restores them. */
void gc_scan_jb(long pool, long depth) {
	long *p = (long *)pool;
	long i = 0;
	if (pool == 0) { return; }
	while (i < depth) { if (p[i] != 0) { gc_scan_range(p[i], p[i] + 512); } i = i + 1; }
}

long THROWN;

long js_throw(long v);

/* ------------------------------------------------------------- the calls - */

long arg_at(long args, long i) { if (i < arr_len(args)) { return arr_get(args, i); } return H_UNDEF; }
long arg_num(long args, long i) { if (i < arr_len(args)) { return to_number(arr_get(args, i)); } return DNAN; }
long arg_str(long args, long i) { if (i < arr_len(args)) { return to_string(arr_get(args, i)); } return mk_cstr(""); }

long host_call(long id, long self, long args);
long builtin_method(long recv, long mid, long args);

long gen_create(long gf, long args);

long js_call(long callee, long self, long args) {
	long t = tag_of(callee);
	if (t == 7) { return jsdispatch(fa(callee), fb(callee), args); }
	if (t == 8) { return host_call(fa(callee), self, args); }
	if (t == 9) { return builtin_method(fa(callee), fb(callee), args); }
	/* A generator FUNCTION: calling it runs NOTHING and answers a generator. */
	if (t == 16) { return gen_create(callee, args); }
	die2("call of a non function value: ", callee);
	return H_UNDEF;
}

/* ------------------------------------------------------------- coroutines - */
/*
 * TAG 15 is a GENERATOR and TAG 16 a generator FUNCTION, the floor's twins of
 * abnf/jsrt.go's jsGenerator and the closure js_genfn wraps. The Go half runs
 * the body on a goroutine and hands over on two unbuffered channels; this half
 * runs it on a pthread and hands over on one mutex + condition variable. Both
 * are STRICTLY ALTERNATING - at any moment exactly one side runs - so neither
 * needs a lock around the runtime's own state.
 *
 * WHY A THREAD AND NOT A SECOND STACK. Measured, not assumed:
 *   - makecontext/swapcontext work on this machine (macOS 15, arm64) but are
 *     deprecated there, and every field this file would have to write into a
 *     ucontext_t (uc_stack.ss_sp, ss_size, uc_link) has a PLATFORM-DEPENDENT
 *     OFFSET. runtime.c has no headers, so it would have to hard-code them.
 *   - a pthread is reached entirely through opaque pointers plus *_init calls,
 *     so this source is layout-free.
 * The price is a parked thread per live generator - exactly the price the Go
 * half already pays in a parked goroutine - and about 4.5 us a resume against
 * the Go half's 0.75 us, all of it the condvar's two kernel round trips.
 *
 * THE CONTROL BLOCK is a long[16] rather than a struct, so no new type crosses
 * the C subset, and it is malloc'd rather than allocated on the heap this file
 * collects - the collector must be able to read it while sweeping. It holds NO
 * handle except cb[7]; every value that travels between the two sides lives in
 * the generator CELL, which is traced.
 *
 *   cb[0] turn: 1 = the coroutine runs, 0 = the resumer runs
 *   cb[1] state: 0 not started, 1 SUSPENDED (its stack is a root), 2 done,
 *                3 running
 *   cb[2] pthread_mutex_t buffer     cb[3] pthread_cond_t buffer
 *   cb[4] the suspended stack's low water mark
 *   cb[5] the coroutine stack's base, published by the thread itself
 *   cb[6] a jmp_buf buffer: the callee-saved registers it parked with
 *   cb[7] the generator cell
 *   cb[8] the pthread_t
 *   cb[9] its OWN jmp_buf pool (a setjmp is only valid on the stack that took
 *         it), cb[10] the depth it parked at, cb[12] the pool's length
 *   cb[11] 1 if the body left by a THROW rather than a return
 *
 * THE GENERATOR CELL (tag 15):
 *   a  the closure of the body        b  the argument array
 *   c  the control block, RAW         d  the last yielded value
 *   e  raw: 1 once the body finished  f  the sent value / the return value /
 *                                        the thrown value
 */

/* Copy a long array into a bigger zeroed one. The old block is not freed: this
 * file never frees a malloc (the jmp_buf pools do not either), and doubling
 * makes the total abandoned smaller than the array that survives. */
long coro_grow(long arr, long cap, long newcap) {
	long *n = (long *)malloc(8 * newcap);
	long *o = (long *)arr;
	long i = 0;
	while (i < newcap) { n[i] = 0; i = i + 1; }
	i = 0;
	while (i < cap) { n[i] = o[i]; i = i + 1; }
	return (long)n;
}

long coro_alloc(long g) {
	long *cb = (long *)malloc(8 * 16);
	long i = 0;
	while (i < 16) { cb[i] = 0; i = i + 1; }
	cb[2] = (long)malloc(256);
	cb[3] = (long)malloc(256);
	cb[6] = (long)malloc(512);
	i = 0;
	while (i < 64) { long *jb = (long *)cb[6]; jb[i] = 0; i = i + 1; }
	cb[9] = (long)malloc(8 * 128);
	cb[12] = 128;
	i = 0;
	while (i < 128) { long *jp = (long *)cb[9]; jp[i] = 0; i = i + 1; }
	pthread_mutex_init((void *)cb[2], 0);
	pthread_cond_init((void *)cb[3], 0);
	cb[7] = g;
	return (long)cb;
}

long gen_create(long gf, long args) {
	long h = cell_new(15);
	sa(h, fa(gf));
	sb(h, args);
	sc(h, 0);
	sd(h, H_UNDEF);
	se(h, 0);
	sf(h, H_UNDEF);
	return h;
}

/* js_genfn: wrap a compiled closure so that CALLING it creates a generator
 * instead of running the body. The twin of abnf/jsrt.go's js_genfn. */
long js_genfn(long body) { long h = cell_new(16); sa(h, body); return h; }

/* The thread entry. Its ADDRESS is taken with dlsym, never as a C value. */
void *coro_entry(void *arg) {
	long *cb = (long *)arg;
	long anchor = 0;
	long g;
	long ret;
	long barrier;
	cb[5] = ((long)&anchor) + 256;
	pthread_mutex_lock((void *)cb[2]);
	while (cb[0] == 0) { pthread_cond_wait((void *)cb[3], (void *)cb[2]); }
	pthread_mutex_unlock((void *)cb[2]);
	/* The running side always owns GC_STACK_BASE and JBP, so a collection
	 * triggered by an allocation inside the body scans THIS stack and not
	 * main's, and a try in the body uses THIS stack's barriers. */
	GC_STACK_BASE = cb[5];
	JBP = cb[9];
	JB_CAP = cb[12];
	JB_DEPTH = 0;
	g = cb[7];
	CUR_GEN = g;
	cb[1] = 3;
	/* THE THROW BARRIER, and the reason this primitive is safe to build a
	 * language on. js_throw longjmps to the innermost barrier; without one here
	 * a throw inside a generator body would jump to a jmp_buf the RESUMER
	 * setjmp'd - i.e. into another thread's frame, which only appears to work
	 * while the resumer stays parked and produces a WRONG ANSWER as soon as a
	 * collection moves the timing (MEC_GC=stress caught it). The body therefore
	 * unwinds only to here; the value travels back in the cell, and gen_resume
	 * re-raises it on the resumer's own stack. That is exactly the three steps
	 * abnf/jsrt.go takes with recover() and a re-panic. */
	barrier = jb_at(0);
	JB_DEPTH = 1;
	ret = H_UNDEF;
	if (setjmp((void *)barrier) == 0) {
		ret = js_call(fa(g), H_UNDEF, fb(g));
		cb = (long *)fc(CUR_GEN);
		sf(CUR_GEN, ret);
		cb[11] = 0;
	} else {
		cb = (long *)fc(CUR_GEN);
		sf(CUR_GEN, THROWN);
		cb[11] = 1;
	}
	g = cb[7];
	JB_DEPTH = 0;
	se(g, 1);
	sd(g, H_UNDEF);
	cb[1] = 2;
	pthread_mutex_lock((void *)cb[2]);
	cb[0] = 0;
	pthread_cond_broadcast((void *)cb[3]);
	pthread_mutex_unlock((void *)cb[2]);
	return 0;
}

/* Resume a generator until its next yield or its return. Answers nothing: the
 * result is in the cell (d = yielded, e = done, f = returned or thrown), and a
 * body that threw has its value RE-RAISED here, on the resumer's stack. */
void gen_resume(long g, long sent, int closing) {
	long *cb;
	long anchor = 0;
	long save_gen;
	long save_jbp;
	long save_cap;
	long save_depth;
	long threw;
	if (fe(g) != 0) { return; }
	if (fc(g) == 0) {
		long blk = coro_alloc(g);
		long th = 0;
		if (CORO_N >= CORO_CAP) {
			long nc = CORO_CAP == 0 ? 64 : CORO_CAP * 2;
			CORO = coro_grow(CORO, CORO_CAP, nc);
			CORO_CAP = nc;
		}
		sc(g, blk);
		{ long *cr = (long *)CORO; cr[CORO_N] = blk; }
		CORO_N = CORO_N + 1;
		if (CORO_ENTRY == 0) {
			void *hh = dlopen(0, 2);
			CORO_ENTRY = (long)dlsym(hh, "coro_entry");
			if (CORO_ENTRY == 0) { die("coroutines: dlsym of coro_entry failed"); }
		}
		pthread_create(&th, 0, (void *)CORO_ENTRY, (void *)blk);
		{ long *b2 = (long *)blk; b2[8] = th; }
	}
	cb = (long *)fc(g);
	sf(g, sent);
	if (RES_N >= RES_CAP) {
		long nr = RES_CAP == 0 ? 64 : RES_CAP * 2;
		RES_LO  = coro_grow(RES_LO,  RES_CAP, nr);
		RES_HI  = coro_grow(RES_HI,  RES_CAP, nr);
		RES_JB  = coro_grow(RES_JB,  RES_CAP, nr);
		RES_JBP = coro_grow(RES_JBP, RES_CAP, nr);
		RES_JBD = coro_grow(RES_JBD, RES_CAP, nr);
		RES_CAP = nr;
	}
	{
		long *lo = (long *)RES_LO; long *hi = (long *)RES_HI;
		long *jb = (long *)RES_JB; long *jp = (long *)RES_JBP; long *jd = (long *)RES_JBD;
		if (jb[RES_N] == 0) { jb[RES_N] = (long)malloc(512); }
		lo[RES_N] = ((long)&anchor) - 256;
		hi[RES_N] = GC_STACK_BASE;
		jp[RES_N] = JBP;
		jd[RES_N] = JB_DEPTH;
		setjmp((void *)jb[RES_N]);
	}
	RES_N = RES_N + 1;
	save_gen = CUR_GEN;
	save_jbp = JBP;
	save_cap = JB_CAP;
	save_depth = JB_DEPTH;
	pthread_mutex_lock((void *)cb[2]);
	cb[0] = 1;
	pthread_cond_broadcast((void *)cb[3]);
	while (cb[0] == 1) { pthread_cond_wait((void *)cb[3], (void *)cb[2]); }
	pthread_mutex_unlock((void *)cb[2]);
	RES_N = RES_N - 1;
	{ long *hi = (long *)RES_HI; GC_STACK_BASE = hi[RES_N]; }
	CUR_GEN = save_gen;
	JBP = save_jbp;
	JB_CAP = save_cap;
	JB_DEPTH = save_depth;
	threw = cb[11];
	if (threw != 0) {
		cb[11] = 0;
		/* A body being CLOSED leaves by the GEN_EXIT sentinel js_yield threw
		 * into it; that is the close completing, not an exception, so it stops
		 * here. Anything else a closing body throws - from its own finally, say
		 * - propagates exactly as a normal resume's would. */
		if (closing != 0 && GEN_EXIT != 0 && ff(g) == GEN_EXIT) {
			sf(g, H_UNDEF);
		} else {
			js_throw(ff(g));
		}
	}
}

/* gen_close(g): abandon a generator, running the `finally` clauses that wrap its
 * suspended yield. The twin of abnf/jsrt.go's (*jsGenerator).finish.
 *
 * A generator that never started, or that already finished, is marked done and
 * nothing runs - a body that has not begun has no finally to run, which is what
 * both node and CPython do. A SUSPENDED body is resumed with cb[13] set, and
 * js_yield turns that into a js_throw of GEN_EXIT on the coroutine's own stack,
 * so the unwinding is the ordinary one: every js_try barrier between the yield
 * and the body's entry runs its finally clause, and coro_entry's barrier catches
 * what is left. */
void gen_close(long g) {
	long *cb;
	if (fe(g) != 0) { return; }
	if (fc(g) == 0) {
		se(g, 1);
		sd(g, H_UNDEF);
		sf(g, H_UNDEF);
		return;
	}
	if (GEN_EXIT == 0) {
		GEN_EXIT = mk_obj();
		obj_put(GEN_EXIT, mk_cstr("__genexit"), H_TRUE);
	}
	cb = (long *)fc(g);
	cb[13] = 1;
	gen_resume(g, H_UNDEF, 1);
	/* A body that swallowed the sentinel and yielded again is abandoned where
	 * it stands rather than driven in a loop: node answers a TypeError there
	 * and CPython a RuntimeError, and neither is expressible in the floor. */
	se(g, 1);
	sd(g, H_UNDEF);
}

/* js_yield: suspend the body, hand v to the pending next() and answer the value
 * the FOLLOWING next(x) sends in. The twin of abnf/jsrt.go's js_yield. */
long js_yield(long v) {
	long g = CUR_GEN;
	long *cb;
	long anchor = 0;
	if (g == 0) { die("yield outside of a generator"); }
	cb = (long *)fc(g);
	sd(g, v);
	cb[4] = ((long)&anchor) - 256;
	cb[10] = JB_DEPTH;
	setjmp((void *)cb[6]);
	cb[1] = 1;
	pthread_mutex_lock((void *)cb[2]);
	cb[0] = 0;
	pthread_cond_broadcast((void *)cb[3]);
	while (cb[0] == 0) { pthread_cond_wait((void *)cb[3], (void *)cb[2]); }
	pthread_mutex_unlock((void *)cb[2]);
	cb[1] = 3;
	GC_STACK_BASE = cb[5];
	JBP = cb[9];
	JB_CAP = cb[12];
	JB_DEPTH = cb[10];
	CUR_GEN = g;
	/* gen_close set cb[13] before resuming: this yield does not answer, it
	 * THROWS, so the body unwinds through its own finally clauses on its own
	 * stack and its own barrier pool. */
	if (cb[13] != 0) {
		cb[13] = 0;
		js_throw(GEN_EXIT);
	}
	return ff(g);
}

/* g.next(v) -> {value, done}, byte for byte what abnf/jsrt.go's step() builds. */
long gen_next(long g, long args) {
	long val;
	long done;
	long res;
	if (fe(g) != 0) {
		val = H_UNDEF;
		done = 1;
	} else {
		gen_resume(g, arg_at(args, 0), 0);
		if (fe(g) != 0) { val = ff(g); done = 1; } else { val = fd(g); done = 0; }
	}
	res = mk_obj();
	obj_put(res, mk_cstr("value"), val);
	obj_put(res, mk_cstr("done"), done != 0 ? H_TRUE : H_FALSE);
	return res;
}

/* g.return(v) -> {value: v, done: true}, having closed the body. The twin of
 * abnf/jsrt.go's (*jsGenerator).finish, and the same record it builds. */
long gen_return(long g, long args) {
	long v = arg_at(args, 0);
	long res;
	gen_close(g);
	sf(g, v);
	res = mk_obj();
	obj_put(res, mk_cstr("value"), v);
	obj_put(res, mk_cstr("done"), H_TRUE);
	return res;
}

/* clamp_index / slice_range / substring_range mirror abnf/jsrt.go exactly. */
long clamp_index(long i, long length) {
	if (i < 0) { i = i + length; }
	if (i < 0) { return 0; }
	if (i > length) { return length; }
	return i;
}
long G_BEGIN;
long G_END;
void slice_range(long length, long args) {
	G_BEGIN = 0;
	G_END = length;
	if (arr_len(args) > 0) {
		long n = to_number(arr_get(args, 0));
		G_BEGIN = clamp_index(d_is_nan(n) ? 0 : d_to_long(d_trunc(n)), length);
	}
	if (arr_len(args) > 1 && tag_of(arr_get(args, 1)) != 0) {
		long n = to_number(arr_get(args, 1));
		G_END = clamp_index(d_is_nan(n) ? 0 : d_to_long(d_trunc(n)), length);
	}
	if (G_END < G_BEGIN) { G_END = G_BEGIN; }
}
long clamp_sub(long bits, long length) {
	long v;
	if (d_is_nan(bits) || d_sign(bits)) { return 0; }
	if (d_is_inf(bits)) { return length; }
	v = d_to_long(d_trunc(bits));
	if (v > length) { return length; }
	return v;
}
void substring_range(long length, long args) {
	G_BEGIN = clamp_sub(arg_num(args, 0), length);
	G_END = length;
	if (arr_len(args) > 1 && tag_of(arr_get(args, 1)) != 0) {
		G_END = clamp_sub(to_number(arr_get(args, 1)), length);
	}
	if (G_BEGIN > G_END) { long t = G_BEGIN; G_BEGIN = G_END; G_END = t; }
}

long str_index_of(long h, long sub) {
	long n = str_len(h);
	long m = str_len(sub);
	const char *p = str_ptr(h);
	const char *q = str_ptr(sub);
	long i = 0;
	if (m == 0) { return 0; }
	while (i + m <= n) {
		long j = 0;
		while (j < m && p[i + j] == q[j]) { j = j + 1; }
		if (j == m) { return i; }
		i = i + 1;
	}
	return -1;
}

long builtin_method(long recv, long mid, long args) {
	long t = tag_of(recv);
	if (mid == 60) { return gen_next(recv, args); }   /* generator.next(v) */
	if (mid == 61) { return gen_return(recv, args); } /* generator.return(v) */
	if (mid == 40) {          /* fn.apply(this, argsArray) */
		long a = arg_at(args, 1);
		if (tag_of(a) != 5) { a = mk_arr(); }
		return js_call(recv, arg_at(args, 0), a);
	}
	if (mid == 41) {          /* fn.call(this, ...) */
		long a = mk_arr();
		long i = 1;
		while (i < arr_len(args)) { arr_push(a, arr_get(args, i)); i = i + 1; }
		return js_call(recv, arg_at(args, 0), a);
	}
	if (t == 5) {
		if (mid == 1) {                                       /* push */
			long i = 0;
			while (i < arr_len(args)) { arr_push(recv, arr_get(args, i)); i = i + 1; }
			return mk_num(d_from_long(arr_len(recv)));
		}
		if (mid == 2) {                                       /* pop */
			long n = arr_len(recv);
			long v;
			if (n == 0) { return H_UNDEF; }
			v = arr_get(recv, n - 1);
			sb(recv, n - 1);
			return v;
		}
		if (mid == 3) {                                       /* shift */
			long n = arr_len(recv);
			long v;
			long i = 0;
			if (n == 0) { return H_UNDEF; }
			v = arr_get(recv, 0);
			while (i + 1 < n) { arr_set(recv, i, arr_get(recv, i + 1)); i = i + 1; }
			sb(recv, n - 1);
			return v;
		}
		if (mid == 4) {                                       /* unshift */
			long m = arr_len(args);
			long n = arr_len(recv);
			long out = mk_arr();
			long i = 0;
			while (i < m) { arr_push(out, arr_get(args, i)); i = i + 1; }
			i = 0;
			while (i < n) { arr_push(out, arr_get(recv, i)); i = i + 1; }
			sa(recv, fa(out)); sb(recv, fb(out)); sc(recv, fc(out));
			return mk_num(d_from_long(arr_len(recv)));
		}
		if (mid == 5) {                                       /* reverse, in place */
			long i = 0;
			long j = arr_len(recv) - 1;
			while (i < j) {
				long v = arr_get(recv, i);
				arr_set(recv, i, arr_get(recv, j));
				arr_set(recv, j, v);
				i = i + 1; j = j - 1;
			}
			return recv;
		}
		if (mid == 6) {                                       /* slice */
			long out = mk_arr();
			long i;
			slice_range(arr_len(recv), args);
			i = G_BEGIN;
			while (i < G_END) { arr_push(out, arr_get(recv, i)); i = i + 1; }
			return out;
		}
		if (mid == 7) {                                       /* indexOf */
			long i = 0;
			long n = arr_len(recv);
			long want = arg_at(args, 0);
			while (i < n) { if (strict_eq(arr_get(recv, i), want)) { return mk_num(d_from_long(i)); } i = i + 1; }
			return mk_num(d_from_long(-1));
		}
		if (mid == 8) {                                       /* join */
			long sep = arr_len(args) > 0 ? arg_str(args, 0) : mk_cstr(",");
			long out = mk_cstr("");
			long n = arr_len(recv);
			long i = 0;
			while (i < n) {
				long e = arr_get(recv, i);
				if (i > 0) { out = str_cat(out, sep); }
				if (!is_undef_or_null(e)) { out = str_cat(out, to_string(e)); }
				i = i + 1;
			}
			return out;
		}
		if (mid == 9) {                                       /* concat */
			long out = mk_arr();
			long i = 0;
			long n = arr_len(recv);
			while (i < n) { arr_push(out, arr_get(recv, i)); i = i + 1; }
			i = 0;
			while (i < arr_len(args)) {
				long a = arr_get(args, i);
				if (tag_of(a) == 5) {
					long j = 0;
					while (j < arr_len(a)) { arr_push(out, arr_get(a, j)); j = j + 1; }
				} else { arr_push(out, a); }
				i = i + 1;
			}
			return out;
		}
	}
	if (t == 4) {
		long n = str_ulen(recv);
		if (mid == 20) {                                      /* charCodeAt */
			long b = arg_num(args, 0);
			long i = d_is_nan(b) ? 0 : d_to_long(d_trunc(b));
			long code = str_ucode(recv, i);
			if (code < 0) { return mk_num(DNAN); }
			return mk_num(d_from_long(code));
		}
		if (mid == 21) {                                      /* charAt */
			long b = arg_num(args, 0);
			long i = d_is_nan(b) ? 0 : d_to_long(d_trunc(b));
			if (i < 0 || i >= n) { return mk_cstr(""); }
			/* goja answers U+FFFD for a lone surrogate here (gojaCharAt). */
			return wtf8_clean(u_slice(recv, i, i + 1));
		}
		if (mid == 22) {                                      /* indexOf, in UNITS */
			long at = str_index_of(recv, arg_str(args, 0));
			if (at <= 0) { return mk_num(d_from_long(at)); }
			return mk_num(d_from_long(str_ulen(str_slice(recv, 0, at))));
		}
		if (mid == 23) {                                      /* replace, first only */
			long pat = arg_str(args, 0);
			long rep = arg_str(args, 1);
			long at = str_index_of(recv, pat);
			if (at < 0) { return recv; }
			return str_cat(str_cat(str_slice(recv, 0, at),
			                       rep),
			               str_slice(recv, at + str_len(pat), n));
		}
		if (mid == 24) { slice_range(n, args); return u_slice(recv, G_BEGIN, G_END); }
		if (mid == 25) { substring_range(n, args); return u_slice(recv, G_BEGIN, G_END); }
		if (mid == 26) {                                      /* split */
			long sep = arg_str(args, 0);
			long out = mk_arr();
			long m = str_len(sep);
			long start = 0;
			if (m == 0) {
				long i = 0;
				while (i < n) { arr_push(out, u_slice(recv, i, i + 1)); i = i + 1; }
				return out;
			}
			while (1) {
				long rest = str_slice(recv, start, str_len(recv));
				long at = str_index_of(rest, sep);
				if (at < 0) { arr_push(out, rest); return out; }
				arr_push(out, str_slice(rest, 0, at));
				start = start + at + m;
			}
		}
		if (mid == 27 || mid == 28) {                         /* toUpperCase/LowerCase */
			/* Over CODE POINTS, not bytes: the mapping is Unicode-wide, and it can
			 * change a character's UTF-8 length in both directions. The units array
			 * is UTF-16, so an astral character arrives as a surrogate pair and is
			 * rejoined before mapping and split again after. */
			long un = str_ulen(recv);
			long *u = buf_new(3 * un + 1);
			long k = 0;
			long i = 0;
			while (i < un) {
				long cu = str_ucode(recv, i);
				long cp = cu;
				long adv = 1;
				if (cu >= 55296 && cu <= 56319 && i + 1 < un) {
					long t = str_ucode(recv, i + 1);
					if (t >= 56320 && t <= 57343) {
						cp = 65536 + ((cu - 55296) << 10) + (t - 56320);
						adv = 2;
					}
				}
				if (adv == 1 && cu >= 55296 && cu <= 57343) {
					/* An UNPAIRED surrogate is not valid UTF-8, and the Go twin is
					 * strings.ToUpper, which replaces each of the three bytes it is
					 * carried in with U+FFFD. Lossy, and matched here on purpose:
					 * the two engines have to give the same answer. */
					u[k] = 65533; u[k + 1] = 65533; u[k + 2] = 65533;
					k = k + 3;
					i = i + 1;
					continue;
				}
				cp = case_map(cp, mid == 27);
				if (cp > 65535) {
					u[k] = 55296 + ((cp - 65536) >> 10); k = k + 1;
					u[k] = 56320 + ((cp - 65536) & 1023); k = k + 1;
				} else { u[k] = cp; k = k + 1; }
				i = i + adv;
			}
			return str_from_units(u, k);
		}
		if (mid == 29) {                                      /* trim */
			const char *p = str_ptr(recv);
			long i = 0;
			long j = str_len(recv);
			while (i < j && is_space((int)p[i] & 255)) { i = i + 1; }
			while (j > i && is_space((int)p[j - 1] & 255)) { j = j - 1; }
			return str_slice(recv, i, j);
		}
	}
	die("unknown builtin method");
	return H_UNDEF;
}

/* ------------------------------------------------------- host functions - */

/* ids: 1 println  2 print  3 eprintln  4 parseInt  5 parseFloat  6 exit
 *      10 floor 11 abs 12 max 13 min 14 imul 15 ceil 16 trunc 17 round
 *      18 sign  19 sqrt 20 pow
 *      30 String.fromCharCode  31 Array.isArray  32 byteLen
 */

/* Compare (M * 2^E)^2 with (m2 * 2^e2), exactly. A 53-bit mantissa squared is
 * 106 bits, which no long holds, so the comparison is done on decimal digit
 * arrays - the same exact arithmetic the printer uses. */
long sq_cmp(long M, long E, long m2, long e2) {
	long a = 2 * E;
	long s = a < e2 ? a : e2;
	long na = 0;
	long nb = 0;
	long i = 0;
	long t = M;
	while (t > 0) { G_DC[na] = (char)(t % 10); t = t / 10; na = na + 1; }
	na = dec_mul(G_DC, na, M);
	while (i < a - s) { na = dec_mul(G_DC, na, 2); i = i + 1; }
	t = m2;
	while (t > 0) { G_DD[nb] = (char)(t % 10); t = t / 10; nb = nb + 1; }
	i = 0;
	while (i < e2 - s) { nb = dec_mul(G_DD, nb, 2); i = i + 1; }
	return dec_cmp(G_DC, na, G_DD, nb);
}
/* (bits)^2 against m2 * 2^e2. */
long sq_cmp_bits(long bits, long m2, long e2) {
	long E = (bits >> 52) & 2047;
	long M = bits & 4503599627370495L;
	long e = -1074;
	if (E != 0) { M = M + 4503599627370496L; e = E - 1075; }
	return sq_cmp(M, e, m2, e2);
}
/* Correctly rounded square root. Newton's iteration in soft float lands within
 * an ulp or two of the true root; WHICH neighbour is nearest is then decided
 * exactly, by comparing squares as whole numbers. Stepping by one ulp is a step
 * of one on the bit pattern, which carries across the exponent by itself. */
long d_sqrt(long x) {
	union DB a;
	union DB g;
	union DB two;
	long i = 0;
	long Ex;
	long mx;
	long ex;
	long steps;
	long c;
	long M;
	long sc;
	if (d_is_nan(x) || d_sign(x)) { if (d_is_zero(x)) { return x; } return DNAN; }
	if (d_is_inf(x) || d_is_zero(x)) { return x; }
	/* A subnormal is scaled up by 2^106 first (exactly) and the root scaled back
	 * down by 2^53, so the iteration below always starts from a normal value. */
	sc = 0;
	if (((x >> 52) & 2047) == 0) { union DB u; u.l = x; u.d = u.d * 81129638414606681695789005144064.0; x = u.l; sc = 1; }
	a.l = x;
	/* HALVE THE EXPONENT for the starting guess: shifting the bit pattern right by
	 * one and adding back half the bias lands within a few percent of the root, so
	 * Newton converges in half a dozen steps. Starting from the VALUE ITSELF, as this
	 * did, needs one step per power of two - eighty were not enough for sqrt(1e100),
	 * which came out 8.27e+75. */
	g.l = (x >> 1) + 2305702271725338624L;      /* 1023 << 51 */
	two.d = 2.0;
	while (i < 30) { g.d = (g.d + a.d / g.d) / two.d; i = i + 1; }
	Ex = (x >> 52) & 2047;
	mx = x & 4503599627370495L;
	ex = -1074;
	if (Ex != 0) { mx = mx + 4503599627370496L; ex = Ex - 1075; }
	steps = 0;
	while (steps < 16 && sq_cmp_bits(g.l, mx, ex) > 0) { g.l = g.l - 1; steps = steps + 1; }
	steps = 0;
	while (steps < 16 && sq_cmp_bits(g.l + 1, mx, ex) <= 0) { g.l = g.l + 1; steps = steps + 1; }
	/* g*g <= x < (g+1ulp)^2 now, so the root is between g and the next double.
	 * The midpoint is (2M+1) * 2^(e-1); comparing ITS square with x says which
	 * side the root falls on. An exact hit is impossible - the square of an odd
	 * number needs 107 bits - but the tie is broken to even anyway. */
	Ex = (g.l >> 52) & 2047;
	M = g.l & 4503599627370495L;
	ex = -1074;
	if (Ex != 0) { M = M + 4503599627370496L; ex = Ex - 1075; }
	mx = x & 4503599627370495L;
	i = -1074;
	if (((x >> 52) & 2047) != 0) { mx = mx + 4503599627370496L; i = ((x >> 52) & 2047) - 1075; }
	c = sq_cmp(2 * M + 1, ex - 1, mx, i);
	if (c < 0 || (c == 0 && (M & 1) != 0)) { g.l = g.l + 1; }
	if (sc) { union DB u; u.l = g.l; u.d = u.d / 9007199254740992.0; return u.l; }
	return g.l;
}
double d_sqrt_d(double v) { union DB u; u.d = v; u.l = d_sqrt(u.l); return u.d; }

/* ----- pow -----
 *
 * A NON-INTEGER exponent used to answer NaN: repeated multiplication cannot reach
 * one, and any approximation of exp/log would have disagreed with the Go twin in the
 * last bits, which is the divergence this whole floor exists to avoid. What follows
 * is therefore a FAITHFUL PORT of Go's own math.Pow, math.Exp, math.Log, math.Frexp,
 * math.Ldexp and math.Modf - the same constants, the same polynomials, the same order
 * of operations. Now that the emitted arithmetic is correctly rounded IEEE-754, the
 * same sequence of operations gives the same bits, and it does: verified against the
 * Go runtime over a differential probe of every shape of argument. Ported rather than
 * invented precisely because "close enough" is not an answer here. */

double d_frexp(double f, long *eout) {
	union DB u;
	long e;
	long adj = 0;
	if (f == 0.0) { eout[0] = 0; return f; }
	u.d = f;
	if (d_is_inf(u.l) || d_is_nan(u.l)) { eout[0] = 0; return f; }
	if (u.d < 0.0 ? (0.0 - u.d) < 2.2250738585072014e-308 : u.d < 2.2250738585072014e-308) {
		u.d = u.d * 4503599627370496.0;      /* 2^52 */
		adj = -52;
	}
	e = ((u.l >> 52) & 2047) - 1023 + 1 + adj;
	u.l = (u.l & (0 - 9218868437227405313L)) | (1022L << 52);
	eout[0] = e;
	return u.d;
}
double d_ldexp(double frac, long exp) {
	union DB u;
	double m = 1.0;
	if (frac == 0.0) { return frac; }
	u.d = frac;
	if (d_is_inf(u.l) || d_is_nan(u.l)) { return frac; }
	if (u.d < 0.0 ? (0.0 - u.d) < 2.2250738585072014e-308 : u.d < 2.2250738585072014e-308) {
		u.d = u.d * 4503599627370496.0;
		exp = exp - 52;
	}
	exp = exp + ((u.l >> 52) & 2047) - 1023;
	/* UNDERFLOW KEEPS THE SIGN. Spelled with the sign BIT rather than as the
	 * literal `-0.0`, because unary minus on a double is emitted as `0.0 - x` by
	 * languages/c-to-llvm-ir.abnf and `0.0 - 0.0` is +0.0 - so the literal
	 * silently lost the sign here and `Math.pow(-0.5, 2147483647)` was +0
	 * natively against -0 in node and in the Go twin. The infinity below is
	 * unaffected: `-1.0 / 0.0` computes the sign rather than writing it. */
	if (exp < -1075) {
		union DB z;
		z.l = frac < 0.0 ? (0 - 9223372036854775807L - 1L) : 0L;
		return z.d;
	}
	if (exp > 1023) { return frac < 0.0 ? -1.0 / 0.0 : 1.0 / 0.0; }
	if (exp < -1022) { exp = exp + 53; m = 1.0 / 9007199254740992.0; }
	u.l = (u.l & (0 - 9218868437227405313L)) | ((exp + 1023) << 52);
	return m * u.d;
}
/* Go's math.Mod, ported exactly. The loop subtracts |y| scaled to the same
 * binade as the running remainder, so every subtraction is exact and the whole
 * result is - which is what x - trunc(x/y)*y could not be. */
long d_mod_go(long x, long y) {
	union DB ux;
	union DB uy;
	union DB ur;
	double ya;
	double yfr;
	double rfr;
	double r;
	long e[1];
	long yexp;
	long rexp;
	int neg;
	if (d_is_nan(x) || d_is_nan(y) || d_is_inf(x) || d_is_zero(y)) { return DNAN; }
	if (d_is_inf(y)) { return x; }
	if (d_is_zero(x)) { return x; }
	ux.l = x;
	uy.l = y;
	ya = uy.d < 0.0 ? 0.0 - uy.d : uy.d;
	neg = ux.d < 0.0;
	r = neg ? 0.0 - ux.d : ux.d;
	yfr = d_frexp(ya, e);
	yexp = e[0];
	while (r >= ya) {
		rfr = d_frexp(r, e);
		rexp = e[0];
		if (rfr < yfr) { rexp = rexp - 1; }
		r = r - d_ldexp(ya, rexp - yexp);
	}
	ur.d = r;
	/* The sign is put back by FLIPPING THE BIT, not by `0.0 - r`: Go's math.Mod
	 * negates x itself, so a zero remainder keeps x's sign and Mod(-1, 1) is
	 * -0.0 - where `0.0 - 0.0` is +0.0 and loses it. Invisible until the boxed
	 * double arrived, because the JS rendering of -0.0 is "0" while Java's is
	 * "-0.0"; found by the 10,149 line jsJFlo probe against jsrtjvm.go, 18 lines
	 * of which read "0.0" here and "-0.0" in the Go twin. */
	if (neg) { return d_neg(ur.l); }
	return ur.l;
}

/* The integer part; the caller takes the fraction as f - that. */
double d_modf_int(double f) {
	union DB u;
	long e;
	if (f < 1.0) {
		if (f < 0.0) { return 0.0 - d_modf_int(0.0 - f); }
		return 0.0;
	}
	u.d = f;
	e = ((u.l >> 52) & 2047) - 1023;
	if (e < 52) { u.l = u.l & (0 - (1L << (52 - e))); }
	return u.d;
}
double d_log(double x) {
	union DB u;
	double f1; double f; double k; double s; double s2; double s4;
	double t1; double t2; double R; double hfsq;
	long ki;
	long ke[1];
	u.d = x;
	if (d_is_nan(u.l) || (d_is_inf(u.l) && !d_sign(u.l))) { return x; }
	if (x < 0.0) { u.l = DNAN; return u.d; }
	if (x == 0.0) { return -1.0 / 0.0; }
	f1 = d_frexp(x, ke);
	ki = ke[0];
	if (f1 < 0.7071067811865476) { f1 = f1 * 2.0; ki = ki - 1; }
	f = f1 - 1.0;
	k = (double)ki;
	s = f / (2.0 + f);
	s2 = s * s;
	s4 = s2 * s2;
	t1 = s2 * (6.666666666666735130e-01 + s4 * (2.857142874366239149e-01 +
	     s4 * (1.818357216161805012e-01 + s4 * 1.479819860511658591e-01)));
	t2 = s4 * (3.999999999940941908e-01 + s4 * (2.222219843214978396e-01 +
	     s4 * 1.531383769920937332e-01));
	R = t1 + t2;
	hfsq = 0.5 * f * f;
	return k * 6.93147180369123816490e-01 -
	       ((hfsq - (s * (hfsq + R) + k * 1.90821492927058770002e-10)) - f);
}
double d_exp(double x) {
	union DB u;
	double hi; double lo; double r; double t; double c; double y;
	long k;
	u.d = x;
	if (d_is_nan(u.l)) { return x; }
	if (d_is_inf(u.l)) { return d_sign(u.l) ? 0.0 : x; }
	if (x > 7.09782712893383973096e+02) { return 1.0 / 0.0; }
	if (x < -7.45133219101941108420e+02) { return 0.0; }
	if (x > -3.725290298461914e-09 && x < 3.725290298461914e-09) { return 1.0 + x; }
	k = 0;
	if (x < 0.0) { k = (long)(1.44269504088896338700e+00 * x - 0.5); }
	if (x > 0.0) { k = (long)(1.44269504088896338700e+00 * x + 0.5); }
	hi = x - (double)k * 6.93147180369123816490e-01;
	lo = (double)k * 1.90821492927058770002e-10;
	r = hi - lo;
	t = r * r;
	c = r - t * (1.66666666666666657415e-01 + t * (-2.77777777770155933842e-03 +
	    t * (6.61375632143793436117e-05 + t * (-1.65339022054652515390e-06 +
	    t * 4.13813679705723846039e-08))));
	y = 1.0 - ((lo - (r * c) / (2.0 - c)) - hi);
	return d_ldexp(y, k);
}
int d_odd_int(double y) {
	double yi;
	if (y < 0.0) { y = 0.0 - y; }
	if (y >= 9007199254740992.0) { return 0; }
	yi = d_modf_int(y);
	if (y - yi != 0.0) { return 0; }
	return ((long)yi & 1) == 1;
}
long d_pow(long xb, long yb) {
	union DB ux;
	union DB uy;
	union DB ur;
	double x; double y; double yi; double yf; double a1; double x1;
	long ae; long xe; long i; long ke[1];
	int ysign;
	ux.l = xb;
	uy.l = yb;
	x = ux.d;
	y = uy.d;
	if (d_is_zero(yb) || x == 1.0) { return DONE; }
	if (y == 1.0) { return xb; }
	if (d_is_nan(xb) || d_is_nan(yb)) { return DNAN; }
	if (d_is_zero(xb)) {
		if (y < 0.0) {
			if (d_sign(xb) && d_odd_int(y)) { return DNINF; }
			return DINF;
		}
		if (d_sign(xb) && d_odd_int(y)) { return xb; }
		return DZERO;
	}
	if (d_is_inf(yb)) {
		double ax = x < 0.0 ? 0.0 - x : x;
		if (x == -1.0) { return DONE; }
		if ((ax < 1.0) == (!d_sign(yb))) { return DZERO; }
		return DINF;
	}
	if (d_is_inf(xb)) {
		if (d_sign(xb)) { ur.d = 1.0 / x; return d_pow(ur.l, uy.l ^ (0 - 9223372036854775807L - 1L)); }
		if (y < 0.0) { return DZERO; }
		return DINF;
	}
	if (y == 0.5) { return d_sqrt(xb); }
	if (y == -0.5) { ur.d = 1.0 / d_sqrt_d(x); return ur.l; }
	ysign = y < 0.0;
	{ double ay = ysign ? 0.0 - y : y;
	  yi = d_modf_int(ay);
	  yf = ay - yi; }
	if (yf != 0.0 && x < 0.0) { return DNAN; }
	if (yi >= 9223372036854775808.0) {
		double ax = x < 0.0 ? 0.0 - x : x;
		if (x == -1.0) { return DONE; }
		if ((ax < 1.0) == (!ysign)) { return DZERO; }
		return DINF;
	}
	a1 = 1.0;
	ae = 0;
	if (yf != 0.0) {
		if (yf > 0.5) { yf = yf - 1.0; yi = yi + 1.0; }
		a1 = d_exp(yf * d_log(x));
	}
	x1 = d_frexp(x, ke);
	xe = ke[0];
	i = (long)yi;
	while (i != 0) {
		if (xe < -4096 || 4096 < xe) {
			/* catastrophic under/overflow: the exponent alone decides */
			ae = ae + xe;
			i = 0;
		} else {
			if ((i & 1) == 1) { a1 = a1 * x1; ae = ae + xe; }
			x1 = x1 * x1;
			xe = xe << 1;
			if (x1 < 0.5) { x1 = x1 + x1; xe = xe - 1; }
			i = i >> 1;
		}
	}
	if (ysign) { a1 = 1.0 / a1; ae = 0 - ae; }
	ur.d = d_ldexp(a1, ae);
	return ur.l;
}

/* UTF-8 encode one code unit, the strFromUnits of abnf/jsrt.go for everything
 * that is not half of a surrogate pair. */
long from_char_code(long args) {
	long n = arr_len(args);
	long *u = buf_new(n + 1);
	long i = 0;
	while (i < n) {
		long b = to_number(arr_get(args, i));
		long v = d_is_nan(b) ? 0 : d_to_long(d_trunc(b));
		u[i] = v & 65535;                    /* ECMA ToUint16 */
		i = i + 1;
	}
	return str_from_units(u, n);
}

long digit_value(int c) {
	if (c >= 48 && c <= 57) { return c - 48; }
	if (c >= 97 && c <= 122) { return c - 97 + 10; }
	if (c >= 65 && c <= 90) { return c - 65 + 10; }
	return -1;
}

long js_parse_int(long h, long radix) {
	const char *s = str_ptr(h);
	long n = str_len(h);
	long i = 0;
	long j = n;
	union DB val;
	union DB rd;
	long digits = 0;
	long sign = 1;
	while (i < j && is_space((int)s[i] & 255)) { i = i + 1; }
	while (j > i && is_space((int)s[j - 1] & 255)) { j = j - 1; }
	if (i < j && s[i] == 45) { sign = -1; i = i + 1; }
	else if (i < j && s[i] == 43) { i = i + 1; }
	if (radix == 0) {
		radix = 10;
		if (i + 1 < j && s[i] == 48 && (s[i + 1] == 120 || s[i + 1] == 88)) { radix = 16; i = i + 2; }
	} else if (radix == 16 && i + 1 < j && s[i] == 48 && (s[i + 1] == 120 || s[i + 1] == 88)) {
		i = i + 2;
	}
	if (radix < 2 || radix > 36) { return DNAN; }
	val.d = 0.0;
	rd.d = (double)radix;
	while (i < j) {
		long d = digit_value((int)s[i] & 255);
		if (d < 0 || d >= radix) { i = j; }
		else { val.d = val.d * rd.d + (double)d; digits = digits + 1; i = i + 1; }
	}
	if (digits == 0) { return DNAN; }
	if (sign < 0) { val.d = 0.0 - val.d; }
	return val.l;
}

/* parseFloat: the longest numeric PREFIX. */
long js_parse_float(long h) {
	const char *s = str_ptr(h);
	long n = str_len(h);
	long i = 0;
	long start;
	long end;
	long seen = 0;
	while (i < n && is_space((int)s[i] & 255)) { i = i + 1; }
	start = i;
	if (i < n && (s[i] == 43 || s[i] == 45)) { i = i + 1; }
	while (i < n && s[i] >= 48 && s[i] <= 57) { i = i + 1; seen = 1; }
	if (i < n && s[i] == 46) {
		i = i + 1;
		while (i < n && s[i] >= 48 && s[i] <= 57) { i = i + 1; seen = 1; }
	}
	if (seen == 0) { return DNAN; }
	end = i;
	if (i < n && (s[i] == 101 || s[i] == 69)) {
		long k = i + 1;
		long ed = 0;
		if (k < n && (s[k] == 43 || s[k] == 45)) { k = k + 1; }
		while (k < n && s[k] >= 48 && s[k] <= 57) { k = k + 1; ed = 1; }
		if (ed) { end = k; }
	}
	return str_to_num(str_slice(h, start, end));
}

long js_keys(long o);          /* the keysOf builtin below is exactly this */
long js_del(long o, long key); /* and delKey is exactly this */

long host_call(long id, long self, long args) {
	long n = arr_len(args);
	long i = 0;
	if (id == 1 || id == 2 || id == 3) {
		OUTFD = (id == 3) ? 2 : 1;
		while (i < n) { if (i > 0) { o_ch(32); } print_val(arr_get(args, i)); i = i + 1; }
		if (id != 2) { o_ch(10); }
		OUTFD = 1;
		return H_UNDEF;
	}
	if (id == 4) {
		long r = arg_num(args, 1);
		return mk_num(js_parse_int(arg_str(args, 0), d_is_nan(r) ? 0 : d_to_long(d_trunc(r))));
	}
	if (id == 5) { return mk_num(js_parse_float(arg_str(args, 0))); }
	if (id == 6) {
		long v = arg_num(args, 0);
		exit((int)(d_is_nan(v) ? 0 : to_int32(v)));
	}
	if (id == 10) { return mk_num(d_floor(arg_num(args, 0))); }
	if (id == 11) { return mk_num(d_abs(arg_num(args, 0))); }
	if (id == 12 || id == 13) {
		long acc = (id == 12) ? DNINF : DINF;
		while (i < n) {
			long v = to_number(arr_get(args, i));
			if (d_is_nan(v) || d_is_nan(acc)) { acc = DNAN; }
			else if (id == 12) { if (d_lt(acc, v)) { acc = v; } }
			else { if (d_lt(v, acc)) { acc = v; } }
			i = i + 1;
		}
		return mk_num(acc);
	}
	if (id == 14) {
		long a = to_int32(arg_num(args, 0));
		long b = to_int32(arg_num(args, 1));
		long p = a * b;
		p = p & 4294967295;
		if (p >= 2147483648) { p = p - 4294967296; }
		return mk_num(d_from_long(p));
	}
	if (id == 15) { return mk_num(d_ceil(arg_num(args, 0))); }
	if (id == 16) { return mk_num(d_trunc(arg_num(args, 0))); }
	if (id == 17) {
		long v = arg_num(args, 0);
		union DB h;
		if (d_is_nan(v) || d_is_inf(v)) { return mk_num(v); }
		h.d = 0.5;
		return mk_num(d_floor(d_add(v, h.l)));
	}
	if (id == 18) {
		long v = arg_num(args, 0);
		if (d_is_nan(v) || d_is_zero(v)) { return mk_num(v); }
		return mk_num(d_sign(v) ? d_neg(DONE) : DONE);
	}
	if (id == 19) { return mk_num(d_sqrt(arg_num(args, 0))); }
	if (id == 20) { return mk_num(d_pow(arg_num(args, 0), arg_num(args, 1))); }
	if (id == 30) { return from_char_code(args); }
	if (id == 31) { return mk_bool(tag_of(arg_at(args, 0)) == 5); }
	if (id == 32) { return mk_num(d_from_long(str_len(arg_str(args, 0)))); }
	/* fail(msg): a RUNTIME ERROR, not an exception. The Go twin's rt.fail
	 * (abnf/jsrt.go) panics with exactly this text, so the two engines report an
	 * impossible operand the same way; throwing instead would prefix the message
	 * with "uncaught exception: ". Layer 2 of every language needs it - Lua's
	 * "attempt to perform arithmetic on a nil value" arrives here - and
	 * languages/lib/interp-core.js already calls a fail() of its own. */
	if (id == 36) {
		werr("js runtime error: ");
		wstr(to_string(arg_at(args, 0)));
		werr("\n");
		exit(1);
	}
	if (id == 33) {                                        /* sprintf */
		if (n == 0) { return mk_cstr(""); }
		return fmt_apply(arg_str(args, 0), args, 1);
	}
	if (id == 34) {                                        /* printf */
		if (n == 0) { return H_UNDEF; }
		OUTFD = 1;
		o_str(fmt_apply(arg_str(args, 0), args, 1));
		return H_UNDEF;
	}
	if (id == 35) { return fmt_sprint(args); }             /* sprint */
	/* ----- the SIZED INTEGER, for layer 2 (docs/runtime-next-plan.md part 2).
	 * A MetaJS runtime library cannot make a new primitive type, so the floor
	 * hands it the constructor and the three readers. The value travels as two
	 * 32 bit halves because a MetaJS number is a double and cannot carry 64 bits
	 * exactly - the same {h, l} pair languages/lib/interp-core.js already uses.
	 *   sint(hi, lo, bits, unsigned)  build one, through si_norm: the result is a
	 *                                 PLAIN NUMBER when the invariant says so
	 *   sintIs(v)                     is this a sized integer (tag 13)
	 *   sintHi(v) / sintLo(v)         the two halves back out, unsigned
	 *   sintWidth(v) / sintUns(v)     the declared width and signedness */
	if (id == 40) {
		long hi = d_to_long(d_trunc(arg_num(args, 0))) & 4294967295;
		long lo = d_to_long(d_trunc(arg_num(args, 1))) & 4294967295;
		long w = n > 2 ? d_to_long(d_trunc(arg_num(args, 2))) : 64;
		long u = n > 3 ? truthy(arg_at(args, 3)) : 0;
		return si_norm((hi << 32) | lo, w, u);
	}
	if (id == 41) { return mk_bool(tag_of(arg_at(args, 0)) == 13); }
	if (id == 42) { return mk_num(d_from_long((long)(si_u(si_val(arg_at(args, 0)), 64) >> 32))); }
	if (id == 43) { return mk_num(d_from_long(si_val(arg_at(args, 0)) & 4294967295)); }
	if (id == 44) { long v = arg_at(args, 0); return mk_num(d_from_long(si_width_of(v, v))); }
	if (id == 45) { long v = arg_at(args, 0); return mk_bool(si_uns_of(v, v) != 0); }
	/*   sintOp(code, l, r)            one binary operator, by index:
	 *                                 0 + 1 - 2 * 3 / 4 % 5 & 6 | 7 ^ 8 &^ 9 << 10 >>
	 *   sintCmp(l, r)                 -1 / 0 / 1, unsigned where the type is
	 *   sintConv(v, bits, unsigned)   an explicit conversion (js_giconv)
	 *   sintStr(v) / sintNum(v)       the decimal text and the double reading */
	if (id == 46) { return si_apply(d_to_long(d_trunc(arg_num(args, 0))), arg_at(args, 1), arg_at(args, 2)); }
	if (id == 47) { return mk_num(d_from_long(si_cmp(arg_at(args, 0), arg_at(args, 1)))); }
	if (id == 48) {
		return si_norm(si_val(arg_at(args, 0)), d_to_long(d_trunc(arg_num(args, 1))), truthy(arg_at(args, 2)));
	}
	if (id == 49) { long v = arg_at(args, 0); if (tag_of(v) == 13) { return si_str(v); } return to_string(v); }
	if (id == 50) { return mk_num(si_float(arg_at(args, 0))); }
	/* ----- the BOXED DOUBLE, for layer 2 (tag 14, abnf/jsrtjvm.go). The same
	 * shape as the sint* family above: layer 2 cannot make a primitive type, so
	 * the floor hands it the constructor and the readers. A double travels as
	 * ONE MetaJS number, unlike a sized integer's two halves - a double is
	 * exactly what a MetaJS number already is.
	 *   flo(v, style)        build one: style 0 java (default), 1 go, 2 c#
	 *   floIs(v)             is this a boxed double (jvmIsFlo)
	 *   floNum(v)            the double back out (to_number)
	 *   floStyle(v)          the style, 0 for anything that is not a box
	 *   floStr(v)            the language's own float text (jvmFloText)
	 *   floOp(code, l, r)    one binary operator, 0 + 1 - 2 * 3 / 4 % (jvmArith)
	 *   floEq(l, r)          strict_eq's two tag-14 arms (jvmNumEq)
	 *   floMax / floMin      Math.max / Math.min (jvmMinMax)
	 *   floAbs(v)            Math.abs */
	if (id == 51) {
		long sty = n > 1 ? d_to_long(d_trunc(arg_num(args, 1))) : 0;
		return jf_make(to_number(arg_at(args, 0)), sty);
	}
	if (id == 52) { return mk_bool(tag_of(arg_at(args, 0)) == 14); }
	if (id == 53) { return mk_num(to_number(arg_at(args, 0))); }
	if (id == 54) { long v = arg_at(args, 0); return mk_num(d_from_long(jf_style_of(v, v))); }
	if (id == 55) { long v = arg_at(args, 0); if (tag_of(v) == 14) { return jf_text(v); } return to_string(v); }
	if (id == 56) { return jf_arith(d_to_long(d_trunc(arg_num(args, 0))), arg_at(args, 1), arg_at(args, 2)); }
	if (id == 57) {
		long l = arg_at(args, 0);
		long r = arg_at(args, 1);
		if (tag_of(l) == 14) { return mk_bool(jf_num_eq(fa(l), r)); }
		if (tag_of(r) == 14) { return mk_bool(jf_num_eq(fa(r), l)); }
		return mk_bool(strict_eq(l, r));
	}
	if (id == 58 || id == 59) {
		long l = arg_at(args, 0);
		long r = arg_at(args, 1);
		long ln = to_number(l);
		long rn = to_number(r);
		/* jvmMathObject's max/min, through jf_take_l - which is `>` / `<` plus
		 * the signed-zero and NaN cases those two cannot see. */
		return jf_minmax(l, r, jf_take_l(ln, rn, id == 58));
	}
	if (id == 60) { return jf_abs(arg_at(args, 0)); }
	/* keysOf(o): an object's OWN keys, in insertion order. It is js_keys, and
	 * it is here for the reason the sint* and flo* families are: an EXTERN is
	 * only callable from an emitter, so a layer-2 file could not enumerate an
	 * object at all - MetaJS has neither for..in nor Object.keys. Every language
	 * whose runtime has to render, compare or copy an object needs it (PHP's
	 * var_dump, `==` on objects, the (array) cast and clone are the first four).
	 * A dict box ({__dict,keys,vals}) yields its key array; a plain object
	 * yields its string keys, skipping the internal __-prefixed slots. */
	if (id == 61) { return js_keys(arg_at(args, 0)); }
	/* sintRaw(hi, lo, bits, unsigned): sint() WITHOUT si_norm - always a tag 13
	 * cell, never a plain number.
	 *
	 * WHY IT EXISTS (docs/runtime-next-plan.md part 3, the java migration).
	 * si_norm's whole job is to UNBOX a signed 64 bit value a double holds
	 * exactly, and that is precisely what a statically typed language's 64 bit
	 * signed type must NOT do: Java's `1000000L * 1000000L` is 1000000000000
	 * while `1000000 * 1000000` is -727379968, so a small long has to stay a
	 * long. Every one of the eleven sint* builtins goes through si_norm, so
	 * layer 2 had no constructor for a small BOXED value at all. Java worked
	 * around it by marking the box UNSIGNED - the one shape si_norm will not
	 * unbox - and then re-supplying six signed operations (/ % >>, compare,
	 * decimal text, double reading) in about 90 lines of java-rt.metajs. That
	 * workaround is owed once per language, and csharp, go, kotlin, swift and
	 * dart all have a 64 bit signed type.
	 *
	 * si_trunc IS still applied, so the cell keeps the invariant its header
	 * states (`the value already truncated to the width and sign extended`) -
	 * a caller asking for 8 bits gets a well formed int8 box, not 300 labelled
	 * as one. The ONLY thing dropped is si_norm's unboxing arm. */
	if (id == 62) {
		long hi = d_to_long(d_trunc(arg_num(args, 0))) & 4294967295;
		long lo = d_to_long(d_trunc(arg_num(args, 1))) & 4294967295;
		long w = n > 2 ? d_to_long(d_trunc(arg_num(args, 2))) : 64;
		long u = n > 3 ? truthy(arg_at(args, 3)) : 0;
		return si_make(si_trunc((hi << 32) | lo, w, u), w, u);
	}
	/* delKey(o, key): remove an own property, key-order entry and all. It is
	 * js_del, and it is here for the reason keysOf is: an EXTERN is only callable
	 * from an emitter, and MetaJS has no `delete`, so a layer-2 file could not
	 * remove a key AT ALL - it had to blank the slot and keep a side table. Every
	 * language whose runtime models `delete`, `unset`, `remove` or a map erase
	 * needs it. Answers true whether or not the key was there, like the twin. */
	if (id == 63) { return js_del(arg_at(args, 0), arg_at(args, 1)); }
	/* ----- the SCOPE, for layer 2 (tag 11) -----
	 *
	 *   scopeNew(parent)      a scope whose parent is `parent`, or NO parent
	 *                         when the argument is undefined / missing
	 *   scopeParent(s)        the enclosing scope, undefined at the top
	 *   scopeGet(s, name)     the value, walking the CHAIN (js_scope_get);
	 *                         aborts with "variable not defined" if nowhere
	 *   scopeHas(s, name)     is the name bound in THIS scope - no chain walk
	 *   scopeDecl(s, name, v) bind here, overwriting a binding of this scope
	 *
	 * WHY THEY ARE HERE. A scope is the floor's private storage: an emitter
	 * hands a layer-2 function a scope handle (js_pyset_var's `s`, every
	 * `defined?`/`isset`/`global` probe) and layer 2 could do NOTHING with it -
	 * MetaJS has no way to open a cell. Six languages answered that by lowering
	 * the probe into their EMITTER instead (docs/runtime-next-plan.md: swift,
	 * dart, go, ruby, kotlin's nine helpers, python's six), which is IR-building
	 * duplicated per language for what is one language-neutral question.
	 * scopeHas is deliberately the OWN-scope test and not a chain walk, because
	 * that is the one thing js_scope_typeof cannot express (it answers
	 * "undefined" for an absent name and for a slot holding undefined alike)
	 * and the one python asked for; the chain walk is scopeParent in a loop.
	 * There is no scopeSet: `walk with scopeHas/scopeParent, then scopeDecl` is
	 * it, and js_scope_set stays the emitter's.
	 *
	 * NOTE the two deliberate asymmetries with js_scope_new / js_scope_get:
	 * an ABSENT parent is undefined and not G_ROOT (a builtin's arguments are
	 * values, so the handle 0 never arrives, and a chain that reached the host
	 * globals would have no twin in the interpreter half, whose host globals
	 * are not a scope), and the scope argument is TYPE CHECKED (host_scope).
	 * The twins are abnf/jsrtint.go's scopeBindings and
	 * metajs-interpreter.abnf's scNew/scParent/scGet/scHas/scDecl. */
	if (id == 64) {
		long p = arg_at(args, 0);
		return mk_scope(is_undef_or_null(p) ? 0 : host_scope(p));
	}
	if (id == 65) {
		long p = ff(host_scope(arg_at(args, 0)));
		if (p == 0) { return H_UNDEF; }
		return p;
	}
	if (id == 66) { return scope_get(host_scope(arg_at(args, 0)), to_string(arg_at(args, 1))); }
	if (id == 67) {
		return mk_bool(scope_find(host_scope(arg_at(args, 0)), to_string(arg_at(args, 1))) >= 0);
	}
	if (id == 68) {
		scope_put(host_scope(arg_at(args, 0)), to_string(arg_at(args, 1)), arg_at(args, 2));
		return H_UNDEF;
	}
	/* isGenerator(v): is this value a GENERATOR (tag 15), the cell js_gen_create
	 * makes and js_gen_next drives.
	 *
	 * WHY A PREDICATE AND NOT A GENERAL js_tag(v). A numeric tag cannot be the
	 * same in the three engines: js_genfn is a tag 16 CELL here and a *hostFunc
	 * in abnf/jsrt.go (so it would be 16 in one half and 8 in the other), and
	 * metajs-interpreter.abnf has no tag numbering at all - its values are the
	 * host engine's, and a closure, a host function and a bound method are one
	 * JS function there. Every OTHER distinction a numeric tag would carry is
	 * already answered by a name layer 2 has: typeof (undefined / boolean /
	 * number / string / function / object), sintIs, floIs, and `typeof
	 * v.length == "number"` for an array. A generator is the one shape with no
	 * such name, and php-rt.metajs's phIsGen is the evidence: it excludes
	 * __dict / __refcell / __isclass / __class / length one by one and then
	 * tests whether v["next"] is callable, which its own report calls "the one
	 * guess in this port".
	 *
	 * A generator is an OBJECT to typeof in all three halves, so this is a
	 * refinement of typeof and never contradicts it. */
	if (id == 69) { return mk_bool(tag_of(arg_at(args, 0)) == 15); }
	/* floPrec(v, n): the correctly rounded n SIGNIFICANT DIGIT decimal of |v|,
	 * written DIGITS "e" EXP - the digits with no point and the trailing zeros
	 * stripped, and EXP the decimal exponent of the FIRST digit. It is the digit
	 * string of C's "%.*e", and it is the one thing a shortest-form printer
	 * cannot answer: a %g formatter has to be able to ask for a FIXED number of
	 * digits. Lua's tostring is exactly that caller - it tries "%.15g" and, if
	 * that does not read back, "%.17g" - and neither JS-dialect half has
	 * toPrecision, so without this the script halves could only guess.
	 *
	 * shortest_digits already IS the algorithm: its candidate for k digits is
	 * the exact value rounded to k, nearest with ties to even, which is what
	 * "%.ke" writes. g_fixdig = n takes THAT candidate and nothing else - see
	 * its comment for why g_mindig = n, which only started the search there, was
	 * a different function and disagreed with the Go twin (docs/todo.md 2.8).
	 * The trailing zeros come off here, because the CALLER decides how many
	 * digits to show.
	 *
	 * The twins are floPrecStr in abnf/commonscript.go (goja and the frozen
	 * grammar host) and the floPrec binding of languages/metajs-interpreter.abnf. */
	if (id == 70) {
		long v = arg_num(args, 0);
		long nd = arg_num(args, 1);
		char digs[32];
		char out[64];
		long e10;
		long nn;
		long o = 0;
		long i = 0;
		long saved;
		if (d_is_nan(v) || d_is_inf(v) || d_is_zero(v)) { return mk_cstr("0e0"); }
		nd = d_is_nan(nd) ? 1 : d_to_long(d_trunc(nd));
		if (nd < 1) { nd = 1; }
		if (nd > 17) { nd = 17; }
		saved = g_fixdig;
		g_fixdig = nd;
		e10 = shortest_digits(v, digs);
		nn = g_ndig;
		g_fixdig = saved;
		while (nn > 1 && digs[nn - 1] == 48) { nn = nn - 1; }
		while (i < nn) { out[o] = digs[i]; o = o + 1; i = i + 1; }
		out[o] = 101; o = o + 1;                         /* 'e' */
		if (e10 < 0) { out[o] = 45; o = o + 1; e10 = 0 - e10; }
		o = o + int_digits(d_from_long(e10), out + o);
		return mk_str(out, o);
	}
	die("unknown host function");
	return H_UNDEF;
}

/* ---------------------------------------------------- exceptions --------- */

long js_throw(long v) {
	THROWN = v;
	if (JB_DEPTH <= 0) {
		long s = to_string(v);
		OUTFD = 2;
		o_cstr("js runtime error: uncaught exception: ");
		o_str(s);
		o_ch(10);
		exit(1);
	}
	{ long *p = (long *)JBP; longjmp((void *)p[JB_DEPTH - 1], 1); }
	return H_UNDEF;
}

long js_try(long tryC, long catchC, long finC) {
	long result = H_UNDEF;
	long pending = H_UNDEF;
	int havePending = 0;              /* a separate flag: `throw undefined` is legal */
	long mydepth = JB_DEPTH;
	long buf;
	int hasCatch = is_callable(catchC);
	int hasFin = is_callable(finC);
	buf = jb_at(JB_DEPTH);
	JB_DEPTH = JB_DEPTH + 1;
	if (setjmp((void *)buf) == 0) {
		result = js_call(tryC, H_UNDEF, mk_arr());
		JB_DEPTH = mydepth;
	} else {
		JB_DEPTH = mydepth;
		pending = THROWN;
		havePending = 1;
	}
	/* A catch clause does NOT see the generator-close sentinel: GEN_EXIT is a
	 * RETURN COMPLETION, which JavaScript routes past every catch and through
	 * every finally, and abnf/jsrt.go gets the same answer for free (its catch
	 * arm tests for *jsThrown and the close panic is not one). The cost of the
	 * divergence it buys against CPython - where a bare `except:` DOES catch
	 * GeneratorExit - is one unusual program shape, against a floor that would
	 * otherwise let a `catch` swallow a close and leave the body suspended
	 * inside its own catch clause. */
	if (havePending && hasCatch && (GEN_EXIT == 0 || pending != GEN_EXIT)) {
		long a = mk_arr();
		arr_push(a, pending);
		havePending = 0;
		buf = jb_at(JB_DEPTH);
			JB_DEPTH = JB_DEPTH + 1;
		if (setjmp((void *)buf) == 0) {
			result = js_call(catchC, H_UNDEF, a);
			JB_DEPTH = mydepth;
		} else {
			JB_DEPTH = mydepth;
			pending = THROWN;
			havePending = 1;
		}
	}
	if (hasFin) {
		long fres = H_UNDEF;
		buf = jb_at(JB_DEPTH);
			JB_DEPTH = JB_DEPTH + 1;
		if (setjmp((void *)buf) == 0) {
			fres = js_call(finC, H_UNDEF, mk_arr());
			JB_DEPTH = mydepth;
			/* A return/break/continue in finally overrides the try/catch
			 * completion AND swallows a pending throw, exactly like in JS. */
			if (tag_of(fres) == 10) { havePending = 0; result = fres; }
		} else {
			JB_DEPTH = mydepth;
			pending = THROWN;
			havePending = 1;
		}
	}
	if (havePending) { js_throw(pending); }
	return result;
}

/* ------------------------------------------------------ the js_* externs - */

long RETSLOT;

/* The PRECISE half of the root set: every global of this file that can hold a
 * handle. The C stack, the callee-saved registers and the pinned module globals
 * are the other three; together they are the whole of it.
 *
 * SMC_VAL is the string-literal cache and NIC the small-integer cache: neither
 * is a weak table, so both keep their cells alive for the life of the process -
 * which is what makes one shared cell per literal sound in the first place.
 * JB holds jmp_bufs, whose saved registers can be the only copy of a handle
 * between a setjmp and the longjmp that restores them, so they are scanned too;
 * that costs a retention at worst. */
void gc_roots(void) {
	long i = 0;
	gc_try(G_ROOT, 1);
	gc_try(THROWN, 1);
	gc_try(RETSLOT, 1);
	gc_try(GEN_EXIT, 1);
	while (i < 1281) { gc_try(NIC[i], 1); i = i + 1; }
	i = 0;
	while (i < 8192) { gc_try(SMC_VAL[i], 1); i = i + 1; }
	i = 0;
	while (i < PIN_N) { gc_try(PINS[i], 1); i = i + 1; }
	i = 0;
	gc_scan_jb(JBP, JB_DEPTH);
	/* Every SUSPENDED coroutine stack, the registers it parked with, and the
	 * try barriers it parked at. The generator cell itself is a root while its
	 * coroutine exists: the body's frames refer to it, and a generator the
	 * program has dropped still owns a parked thread - the same cost
	 * abnf/jsrt.go's parked goroutine has. */
	i = 0;
	while (i < CORO_N) {
		long *cr = (long *)CORO;
		long *cb = (long *)cr[i];
		gc_try(cb[7], 1);
		if (cb[1] == 1) {
			gc_scan_range(cb[4], cb[5]);
			gc_scan_range(cb[6], cb[6] + 512);
			gc_scan_jb(cb[9], cb[10]);
		}
		i = i + 1;
	}
	/* and the RESUMER stacks, parked while a coroutine runs. */
	i = 0;
	while (i < RES_N) {
		long *lo = (long *)RES_LO; long *hi = (long *)RES_HI;
		long *jb = (long *)RES_JB; long *jp = (long *)RES_JBP; long *jd = (long *)RES_JBD;
		gc_scan_range(lo[i], hi[i]);
		gc_scan_range(jb[i], jb[i] + 512);
		gc_scan_jb(jp[i], jd[i]);
		i = i + 1;
	}
	gc_try(CUR_GEN, 1);
}

/* js_gc_pin answers its argument unchanged, and is IDENTITY in the Go half
 * (abnf/jsrt.go). It exists because the emitted module has globals of its own
 * that hold a handle for the life of the program and that this file cannot see:
 * lib/compile-core.js's jsnumg.N (the memoised boxing of a numeric literal) and
 * metajs-to-llvm-ir.abnf's jsrtlib_env / jsrtlib_f_* (a -rt-lib library's scope
 * and its resolved functions). Registering them here is three call sites and one
 * extern; conservatively scanning the data segment instead is not portable, and
 * emitting a shadow stack for them would be IR the Go half has to carry. */
long js_gc_pin(long v) {
	if (PIN_N >= 8192) { die("too many pinned globals"); }
	PINS[PIN_N] = v;
	PIN_N = PIN_N + 1;
	return v;
}

/* js_str_mem is the string-literal cache; str_intern, next to mk_str above, is
 * the whole of it. */
long js_str_mem(const char *p, long n) { return str_intern(p, n); }

/* Small-integer cache. js_num_i is how EVERY numeric literal in the emitted IR
 * becomes a value, and layer 2 is full of them - operator indices, 0, 1, 2, the
 * bit widths - so it ran 58.7 times per iteration of the Lua benchmark, each one
 * a fresh 56-byte cell the arena never reclaims. A number cell is written once,
 * in mk_num, and never mutated (checked: no other sa/sb/... touches a tag-3
 * cell), and every number comparison in the runtime is by BIT PATTERN, never by
 * handle identity - so one shared cell per small integer is indistinguishable
 * from a fresh one. -256..1024 covers the literals a compiled module emits. */
long js_num_i(long v) {
	long h;
	if (v >= -256 && v <= 1024) {
		h = NIC[v + 256];
		if (h != 0) { return h; }
		h = mk_num(d_from_long(v));
		NIC[v + 256] = h;
		return h;
	}
	return mk_num(d_from_long(v));
}
long js_num_str(long h) { return mk_num(str_to_num(h)); }
long js_obj_new(void)   { return mk_obj(); }
long js_arr_new(void)   { return mk_arr(); }
long js_arr_push(long a, long v) { arr_push(a, v); return 0; }
long js_arg(long args, long i)   { if (i < arr_len(args)) { return arr_get(args, i); } return H_UNDEF; }

long js_get(long o, long k)          { return get_member(o, k); }
long js_set(long o, long k, long v)  { set_member(o, k, v); return 0; }

long js_closure(long idx, long env) { long h = cell_new(7); sa(h, idx); sb(h, env); return h; }

long js_scope_new(long parent) { return mk_scope(scope_of(parent)); }
long js_scope_get(long s, long name) { return scope_get(scope_of(s), name); }

/* ----- the scope and object primitives the OTHER language emitters call -----
 *
 * MetaJS pins a variable's type, so its declaration/assignment pair is
 * js_tdecl/js_tset above. Every other compiler grammar emits the UNPINNED pair
 * js_scope_decl/js_scope_set, plus js_scope_typeof (a name that need not be
 * declared) and js_pyset_var (assign up the chain, else declare here). These are
 * the same walks js_tdecl/js_tset/js_scope_get already do, without the type
 * check - they belong to the floor because a scope's storage is private to it,
 * and they are LANGUAGE NEUTRAL, so layer 2 of any language finds them here.
 * The twins are abnf/jsrt.go: js_scope_decl, js_scope_set (scopeSet),
 * js_scope_typeof and js_pyset_var. */
long js_scope_decl(long s, long name, long v) { scope_put(scope_of(s), name, v); return 0; }

long js_scope_set(long s, long name, long v) {
	long cur = scope_of(s);
	while (cur != 0) {
		long i = scope_find(cur, name);
		if (i >= 0) { long *vals = (long *)fb(cur); vals[i] = v; return 0; }
		cur = ff(cur);
	}
	die2("assignment to undeclared variable: ", name);
	return 0;
}

/* js_scope_set_or_create: JavaScript's non-strict implicit global. The name binds
 * where the chain already holds it, and otherwise in the ROOT scope - NOT in the
 * scope the assignment stands in, which is what js_pyset_var below does.
 *
 * The two differ in exactly one case and it is the one that has no other answer:
 * `function f() { counter = 100 } f(); print(counter)`. js_scope_typeof cannot be
 * used to build this in an emitter, because it answers "undefined" for a slot that
 * HOLDS undefined and for an ABSENT name alike, and `var x; x = 5` and a bare
 * `x = 5` need opposite answers. So the walk belongs here, next to its sibling.
 * The twin is abnf/jsrt.go's scopeSetOrCreate. */
long js_scope_set_or_create(long s, long name, long v) {
	long cur = scope_of(s);
	while (cur != 0) {
		long i = scope_find(cur, name);
		if (i >= 0) { long *vals = (long *)fb(cur); vals[i] = v; return 0; }
		cur = ff(cur);
	}
	scope_put(G_ROOT, name, v);
	return 0;
}

/* Is `name` bound in THIS scope - no chain walk. It is the EXTERN twin of the
 * scopeHas BUILTIN (host 67 below, and jsrtint.go's scBindings), which layer 2
 * has had since d9014d7 and an EMITTER could not reach: a builtin is callable
 * from MetaJS, an extern from emitted IR, and the own-scope question was only
 * ever answerable on one side.
 *
 * js_scope_typeof is the chain walk and cannot express it - it answers
 * "undefined" for an absent name and for a slot holding undefined alike, and it
 * finds a name an ENCLOSING scope binds. See abnf/jsrt.go's twin for the class
 * fill that used it as an own-scope test and copied a module name onto a class. */
long js_scope_has(long s, long name) {
	return mk_bool(scope_find(scope_of(s), name) >= 0);
}

long js_scope_typeof(long s, long name) {
	long cur = scope_of(s);
	while (cur != 0) {
		long i = scope_find(cur, name);
		if (i >= 0) { long *vals = (long *)fb(cur); return type_of(vals[i]); }
		cur = ff(cur);
	}
	return mk_cstr("undefined");
}

/* Python's assignment rule, which the Lua compiler reuses for a bare `x = v`:
 * the name binds where the chain already holds it, and otherwise in the scope
 * the assignment stands in. The `global`/`nonlocal` and binding-boundary arms of
 * the Go twin need js_pyfnscope/js_pyglobal, which only the Python compilers
 * emit; a scope here carries no such mark, so those arms are unreachable and are
 * NOT modelled. A Python layer 2 will have to add them. */
long js_pyset_var(long s, long name, long v) {
	long cur = scope_of(s);
	while (cur != 0) {
		long i = scope_find(cur, name);
		if (i >= 0) { long *vals = (long *)fb(cur); vals[i] = v; return 0; }
		cur = ff(cur);
	}
	scope_put(scope_of(s), name, v);
	return 0;
}

/* js_del(o, key): REMOVE an own property, key-order entry and all. Like the
 * `delete` operator it answers true even when there was nothing to remove, and
 * anything that is not an object is a no-op. The twin is abnf/jsrt.go's js_del.
 *
 * Why it is in the floor rather than in a layer 2: obj_put/obj_get are all a
 * layer-2 file can reach, so without this a runtime has to BLANK the slot and
 * remember the key in a side table that every own-keys walk then consults. That
 * table is quadratic in the number of deletions and never freed - JavaScript's
 * measured 61 s for 4000 delete-in-a-loop iterations against 0.03 s here - and it
 * still cannot repair key ORDER after a delete-then-reinsert, for-in, or object
 * spread, because those walk the floor's own key array.
 *
 * The last slot is BLANKED as well as unlinked: the keys/vals buffers are traced
 * conservatively (gc_trace's raw-buffer arm reads every word of the block, not
 * just the first fc(o)), so a stale handle left past the end would keep its
 * object alive for as long as the owner lives. */
long js_del(long o, long key) {
	long k;
	long i;
	long n;
	long *keys;
	long *vals;
	if (tag_of(o) != 6) { return H_TRUE; }
	k = to_string(key);
	i = obj_find(o, k);
	if (i < 0) { return H_TRUE; }
	keys = (long *)fa(o);
	vals = (long *)fb(o);
	n = fc(o);
	while (i + 1 < n) {
		keys[i] = keys[i + 1];
		vals[i] = vals[i + 1];
		i = i + 1;
	}
	keys[n - 1] = 0;
	vals[n - 1] = 0;
	sc(o, n - 1);
	return H_TRUE;
}

/* An object's own keys in INSERTION ORDER (obj_put appends), skipping the
 * internal __-prefixed slots, exactly like the Go twin. A dict box
 * ({__dict,keys,vals}) yields its key array; nothing the Lua compiler builds is
 * one, but the twin answers it and so does this. */
long js_keys(long o) {
	long out;
	long n;
	long i;
	if (tag_of(o) != 6) { die2("js_keys: not an object (got ", o); }
	if (truthy(obj_get(o, mk_cstr("__dict")))) {
		long ks = obj_get(o, mk_cstr("keys"));
		out = mk_arr();
		i = 0;
		while (i < arr_len(ks)) { arr_push(out, arr_get(ks, i)); i = i + 1; }
		return out;
	}
	out = mk_arr();
	n = fc(o);
	i = 0;
	while (i < n) {
		long *keys = (long *)fa(o);
		long k = keys[i];
		i = i + 1;
		if (str_len(k) >= 2) {
			const char *p = str_ptr(k);
			if (p[0] == '_' && p[1] == '_') { continue; }
		}
		arr_push(out, k);
	}
	return out;
}

long js_tdecl(long s, long name, long v) {
	long sco = scope_of(s);
	long i;
	if (tag_of(v) == 12) {                 /* var x = anytype */
		scope_put(sco, name, H_UNDEF);
		i = scope_find(sco, name);
		scope_pin(sco, i, 1);
		return 0;
	}
	scope_put(sco, name, v);
	i = scope_find(sco, name);
	scope_pin(sco, i, type_class(v));
	return 0;
}

long js_tset(long s, long name, long v) {
	long cur = scope_of(s);
	if (tag_of(v) == 12) { die("MetaJS: anytype can only initialize a declaration"); }
	while (cur != 0) {
		long i = scope_find(cur, name);
		if (i >= 0) {
			long tc = type_class(v);
			if (tc != 0) {
				long old = scope_tc(cur, i);
				if (old == 1) { /* anytype: never checked, never pinned */ }
				else if (old != 0 && old != tc) {
					OUTFD = 2;
					o_cstr("js runtime error: MetaJS: variable '");
					o_str(name);
					o_cstr("' has type ");
					o_cstr(class_name(old));
					o_cstr(" and cannot hold a ");
					o_cstr(class_name(tc));
					o_ch(10);
					exit(1);
				} else { scope_pin(cur, i, tc); }
			}
			scope_put(cur, name, v);
			return 0;
		}
		cur = ff(cur);
	}
	die2("assignment to undeclared variable: ", name);
	return 0;
}

long js_truthy(long h) { return truthy(h) ? 1 : 0; }
long js_not(long h)    { return mk_bool(!truthy(h)); }
long js_typeof(long h) { return type_of(h); }
long js_tonum(long h)  { return mk_num(to_number(h)); }
long js_neg(long h)    { return mk_num(d_neg(to_number(h))); }

long js_add(long a, long b) { return js_add_v(a, b); }
long js_sub(long a, long b) { return mk_num(d_sub(to_number(a), to_number(b))); }
long js_mul(long a, long b) { return mk_num(d_mul(to_number(a), to_number(b))); }
long js_div(long a, long b) { return mk_num(d_div(to_number(a), to_number(b))); }
long js_mod(long a, long b) { return mk_num(d_mod(to_number(a), to_number(b))); }

long js_eq(long a, long b)  { return mk_bool(loose_eq(a, b)); }
long js_ne(long a, long b)  { return mk_bool(!loose_eq(a, b)); }
long js_seq(long a, long b) { return mk_bool(strict_eq(a, b)); }
long js_sne(long a, long b) { return mk_bool(!strict_eq(a, b)); }
long js_lt(long a, long b)  { return mk_bool(js_compare(a, b) == -1); }
long js_gt(long a, long b)  { return mk_bool(js_compare(a, b) == 1); }
long js_le(long a, long b)  { long c = js_compare(a, b); return mk_bool(c == -1 || c == 0); }
long js_ge(long a, long b)  { long c = js_compare(a, b); return mk_bool(c == 1 || c == 0); }

/* ---- the sized-integer externs (abnf/jsrtint.go's init(), function for
 * function). They are what languages/go-to-llvm-ir.abnf already emits, and what
 * java / kotlin / csharp will emit; the C floor implements them so those four
 * can be linked natively.
 *
 * THE jsJFlo ARMS ARE HERE NOW (they were the one stated gap of f19a8ad):
 * jsrtint.go's js_giarith and js_giadd route a boxed double to jvmArith before
 * reaching giArith, and tag 14 is that box. giIsIntegral deliberately does NOT
 * include it - si_integral is still {3, 13} - so a float operand keeps
 * js_compare in the four ordered comparisons instead of being truncated into
 * si_cmp. */

int si_integral(long h) { long t = tag_of(h); return t == 3 || t == 13; }

/* jsrtint.go's js_giarith: a float operand takes the FLOAT path. Only '- * / %'
 * are listed there, deliberately - '+' arrives at js_giadd instead, because in
 * Go it also concatenates strings - so a '+' that reached HERE with a float
 * operand falls through to si_arith exactly as it does in the Go twin. */
long js_giarith(long op, long l, long r) {
	if (tag_of(l) == 14 || tag_of(r) == 14) {
		if (str_eq_c(op, "-")) { return jf_arith(1, l, r); }
		if (str_eq_c(op, "*")) { return jf_arith(2, l, r); }
		if (str_eq_c(op, "/")) { return jf_arith(3, l, r); }
		if (str_eq_c(op, "%")) { return jf_arith(4, l, r); }
	}
	return si_arith(op, l, r);
}
long js_giconv(long v, long bits, long uns) {
	return si_norm(si_val(v), d_to_long(d_trunc(to_number(bits))), truthy(uns));
}
/* -x keeps the operand's type and negates at its own width, so -int8(-128)
 * wraps back to -128. */
long js_gineg(long v) {
	/* A double negates as a double, so -0.0 and -1.0/0.0 stay real. */
	if (tag_of(v) == 14) { return jf_make(d_neg(fa(v)), fb(v)); }
	return si_norm(0 - si_val(v), si_width_of(v, v), si_uns_of(v, v));
}
/* WHY THERE IS NO js_gicmp / js_gieq / js_ginot / js_ginum / js_gistr / js_giis
 * HERE, and why adding one back would be a mistake. All six were written for a
 * sized-integer surface no emitter ever grew: languages/go-to-llvm-ir.abnf sends
 * `==` and `!=` to the shared js_seq (strict_eq's tag-13 arm at the top of this
 * file does the width-exact compare), the ordered comparisons to
 * js_gilt/js_gile/js_gigt/js_gige, `^x` to emitBin("^") + js_band at the
 * operand's width, fmt.Sprint to the shared to_string, and float64(x) to
 * js_giconv. Every one of those answers was checked against real `go run`
 * (uint64max == uint64max, 9007199254740993 == 9007199254740992, ^uint8(0xF0),
 * ^uint64(0), fmt.Sprint(uint64max), float64(uint64max), NaN != NaN) in
 * llvm.Run AND in a native -exe build, and all three agree. The bodies were
 * dead surface - zero calls in the emitted IR of all 34 corpus programs, zero
 * declares, zero internal callers - and the Part B coverage instrumentation
 * found them by never reaching them. Deleted 2026-08-04; see the "settling the
 * 24" section of docs/runtime-next-plan.md. If a language ever DOES need one,
 * write it then, with the ratchet assertion that reaches it. */
/* The UNBOXER: a sized integer becomes its plain number, everything else passes
 * through untouched. */
long js_gival(long v) { if (tag_of(v) == 13) { return mk_num(si_float(v)); } return v; }
/* The four ordered comparisons. A box has to go through si_cmp rather than the
 * shared js_compare, because a uint64 above 2^63 reads as a NEGATIVE int64
 * there, so uint64max < 0 would hold. */
long si_rel(long l, long r) {
	if (tag_of(l) == 13 || tag_of(r) == 13) {
		if (si_integral(l) && si_integral(r)) { return si_cmp(l, r); }
	}
	return js_compare(l, r);
}
long js_gilt(long l, long r) { return mk_bool(si_rel(l, r) < 0); }
long js_gile(long l, long r) { long c = si_rel(l, r); return mk_bool(c == -1 || c == 0); }
long js_gigt(long l, long r) { long c = si_rel(l, r); return mk_bool(c == 1); }
long js_gige(long l, long r) { long c = si_rel(l, r); return mk_bool(c == 1 || c == 0); }
/* '+' is the one arithmetic operator that is not only arithmetic: in Go it also
 * concatenates strings, and only when BOTH sides are strings. */
long js_giadd(long l, long r) {
	if (tag_of(l) == 4 && tag_of(r) == 4) { return str_cat(l, r); }
	/* A float operand adds as a float and keeps its box - checked BEFORE the
	 * numeric test, as in jsrtint.go, so `1.5 + "x"` is not a concatenation. */
	if (tag_of(l) == 14 || tag_of(r) == 14) { return jf_arith(0, l, r); }
	if (!si_numeric(l) || !si_numeric(r)) { return js_add_v(l, r); }
	return si_apply(0, l, r);
}

/* ---- the boxed-double externs of abnf/jsrt.go's big extern table (js_jflo,
 * js_gflo, js_csflo, js_jfdiv), which are what languages/java-to-llvm-ir.abnf,
 * kotlin- and csharp- already emit. They are the LANGUAGE NEUTRAL ones:
 * js_jfstep and js_jadd are left out because their Go arms turn on jsChar, a
 * type the floor does not have, and approximating them silently is how this
 * project gets silently wrong answers.
 *
 * js_jfsub / js_jfmul / js_jfmod / js_jfneg / js_jfint used to sit here too and
 * are gone for the same reason the six js_gi* names above are: NOTHING EMITS
 * THEM. Java routes -, *, / and % through js_jvarith, unary minus through
 * js_jvneg and (int) through js_jvint / makePrimCast's width-exact narrowing;
 * Go's own floExt map still names js_jfsub/js_jfmul/js_jfmod but is unreachable,
 * because emitBinNum consults giBase FIRST and giBase already holds every key
 * floExt has, so those ops leave through js_giarith - whose tag-14 arm calls the
 * same jf_arith these wrappers did. js_jfdiv is the one survivor and it IS
 * emitted (one declare in go-test-full's module). Checked against real `go run`
 * and real `java` on 2.5-1.5, 2.5*1.5, 2.5/1.5, 2.5%1.5, -2.5, 1.0/0.0, -1.0/0.0,
 * 0.0/0.0, -0.0, float32 arithmetic, compound -=/*=//=/%=, (int)3.9, (int)-3.9,
 * (int)3000000000L and -5L: llvm.Run and a native -exe both agree with the real
 * toolchain, with these bodies deleted. See docs/runtime-next-plan.md. */
long js_jflo(long v)  { return jf_make(to_number(v), 0); }
long js_gflo(long v)  { return jf_make(to_number(v), 1); }
long js_csflo(long v) { return jf_make(to_number(v), 2); }
long js_jfdiv(long l, long r) { return jf_arith(3, l, r); }
/* A literal a double cannot hold exactly: the emitter passes its DIGITS,
 * because emitNum would already have rounded 9223372036854775807 to
 * 9223372036854776000 on the way into the module. (text, radix, bits,
 * unsigned); an out-of-range value WRAPS, which is what a Go constant
 * conversion does. Accumulated by hand rather than through str_to_num: the full
 * 64 bit range has to round trip and an overflowing literal has to wrap. */
long js_gilit(long text, long radix, long bits, long uns) {
	long rx = d_to_long(d_trunc(to_number(radix)));
	long s = to_string(text);
	const char *p = (const char *)str_ptr(s);
	long n = str_len(s);
	long i = 0;
	long neg = 0;
	unsigned long acc = 0;
	long v;
	if (n > 0 && p[0] == 45) { neg = 1; i = 1; }
	while (i < n) {
		long c = (long)p[i];
		long d = -1;
		if (c >= 48 && c <= 57) { d = c - 48; }
		else if (c >= 97 && c <= 102) { d = c - 97 + 10; }
		else if (c >= 65 && c <= 70) { d = c - 65 + 10; }
		if (d >= 0 && d < rx) { acc = acc * (unsigned long)rx + (unsigned long)d; }
		i = i + 1;
	}
	v = (long)acc;
	if (neg) { v = 0 - v; }
	return si_norm(v, d_to_long(d_trunc(to_number(bits))), truthy(uns));
}

long js_band(long a, long b) { return mk_num(d_from_long(to_int32(to_number(a)) & to_int32(to_number(b)))); }
long js_bor(long a, long b)  { return mk_num(d_from_long(to_int32(to_number(a)) | to_int32(to_number(b)))); }
long js_bxor(long a, long b) { return mk_num(d_from_long(to_int32(to_number(a)) ^ to_int32(to_number(b)))); }
long js_shl(long a, long b) {
	long v = to_int32(to_number(a));
	long s = to_uint32(to_number(b)) & 31;
	long r = (v << s) & 4294967295;
	if (r >= 2147483648) { r = r - 4294967296; }
	return mk_num(d_from_long(r));
}
long js_shr(long a, long b) {
	long v = to_int32(to_number(a));
	long s = to_uint32(to_number(b)) & 31;
	return mk_num(d_from_long(v >> s));
}
long js_ushr(long a, long b) {
	long v = to_uint32(to_number(a));
	long s = to_uint32(to_number(b)) & 31;
	return mk_num(d_from_long(v >> s));
}

/* js_bytelen: the UTF-8 BYTE length of a value's string form - to_string first,
 * matching abnf/jsrt.go's `len(rt.toString(u(a[0])))`, and the same number the
 * byteLen host builtin (id 32) already answers. It has NO per-language behaviour,
 * unlike js_jadd / js_mcall / the js_range_* family that went to the shared MetaJS
 * layer precisely because each language means something different by them, and
 * this file already holds the bytes (a string cell IS its UTF-8 bytes plus a
 * length in fb), so the body is one line.
 *
 * THIS HALF IS NOT COMMITTABLE ALONE, AND THAT IS A MEASUREMENT, NOT A JUDGEMENT.
 * languages/lib/php-rt.metajs also spells `function js_bytelen(s) { return
 * byteLen(s) }`, which -rt-lib turns into a `define i64 @js_bytelen` in
 * lib/php-rt.ll. Both modules are handed to clang, so with both halves present
 * every PHP native build fails:
 *
 *     duplicate symbol '_js_bytelen' in: ... ld: 1 duplicate symbols
 *
 * (measured 2026-08-04 from a clean archive of ad922a0 with only this body added:
 * tests/php-test-features.php, -exe, MEC_CLANG capturing clang's own output; the
 * metacompiler reports it as "1 unresolved symbol(s)" because undefinedSymbols()
 * only checks whether clang MENTIONED the name - see abnf/linkdiag_test.go.)
 *
 * So this line and the DELETION of php-rt.metajs's two lines plus a regenerated
 * lib/php-rt.ll (tests/gen-php-rt-ll.sh) are ONE commit. Either half alone is a
 * loss: this half alone breaks every PHP native build, and the deletion alone
 * leaves js_bytelen unresolvable.
 *
 * AND THE GENERAL RULE THIS PRODUCED: every extern a layer-2 file DEFINES is a
 * name this file may not also define. Before moving any js_* body in here, run
 * `grep -l 'function js_<name>' languages/lib/*.metajs` first. */
long js_bytelen(long v) { return mk_num(d_from_long(str_len(to_string(v)))); }

/* js_gospread: Go's f(xs...) and C#'s params-array call. Flatten the LAST element
 * of the argument array IN PLACE, so the callee sees the slice's elements as
 * separate arguments; a nil/undefined tail contributes nothing.
 *
 * THE SECOND FLOOR MOVE, AND THE FIRST DOWNWARD ONE. It is here and not in the
 * shared MetaJS layer because - unlike js_has and js_goslice, its two neighbours
 * in that plan item - it carried ONE specification, not two: go-rt.metajs and
 * csharp-rt.metajs spelled the same body character for character in intent (the
 * Go twin's "js_gospread" in abnf/jsrt.go is the third copy and the authority).
 * js_has and js_goslice must STAY split, because Go counts a string in BYTES and
 * C# in UTF-16 code units - that split landed in 33923d4 and is not a debt.
 *
 * MEASURED, because "downward dedup is free" is a prediction and predictions are
 * not measurements (docs/runtime-merge-plan.md Part B, the case_map result: an
 * upcall costs 120 ns, MetaJS data-structure work costs 3.2 us). This body IS
 * MetaJS data-structure work moving the other way, so it should be a gain, and it
 * is a small one. Probe: 4,000 calls that each spread a 500-element slice into a
 * variadic callee - 2,000,000 elements moved - native -exe from
 * go-to-llvm-ir.abnf, six runs interleaved before/after so drift cannot favour
 * either:
 *
 *     layer 2   27.3 / 28.0 / 27.9 s        floor   26.1 / 26.7 / 27.3 s
 *
 * every pair the same way round: about 1.0 s, i.e. 500 ns per element for the
 * MetaJS body. The floor body does not appear AT ALL in 3,878 `sample` frames of
 * the after binary, so its own share is under 35 ms for the same 2,000,000
 * elements - at least 30x, and the case_map ratio is confirmed with its sign
 * flipped.
 *
 * IT IS ONLY 3.7% OF THAT PROGRAM, AND THAT IS THE MORE USEFUL NUMBER. `sample`
 * says the spread-heaviest Go program one can write is dominated by `scope_get`
 * and `ar_block` - the VARIADIC CALLEE's own prologue, which packs the 500
 * arguments back into a slice and is still layer-2 MetaJS in both builds. The
 * caller-side splice was never the expensive half. A future move should aim
 * there, not here.
 *
 * It does NOT upcall, so the five-file rule the
 * case_map probe found (js-rt.metajs and lua-rt.metajs must import runtime.metajs,
 * and tests/coro-poc/gen.ll links the floor with NO layer 2 at all) does not bite
 * here - but tests/coro-poc/gen.ll is exactly why the body may not call back out.
 *
 * The deletions in go-rt.metajs and csharp-rt.metajs are part of THIS change: two
 * definitions of the name is `ld: duplicate symbol '_js_gospread'`, the general
 * rule js_bytelen produced eleven lines above. */
long js_gospread(long args) {
	long n;
	long m;
	long last;
	long i;
	if (tag_of(args) != 5) { return 0; }
	n = arr_len(args);
	if (n == 0) { return 0; }
	last = arr_get(args, n - 1);
	sb(args, n - 1);
	if (tag_of(last) == 5) {
		m = arr_len(last);
		i = 0;
		while (i < m) { arr_push(args, arr_get(last, i)); i = i + 1; }
		return 0;
	}
	if (is_undef_or_null(last)) { return 0; }
	arr_push(args, last);
	return 0;
}

long js_setret(long v) { RETSLOT = v; return 0; }
long js_getret(void)   { return RETSLOT; }

long js_ctl_return(long v) { return mk_ctl(1, v); }
long js_ctl_break(void)    { return mk_ctl(2, H_UNDEF); }
long js_ctl_continue(void) { return mk_ctl(3, H_UNDEF); }
long js_ctl_kind(long h)   { if (tag_of(h) == 10) { return fa(h); } return 0; }
long js_ctl_value(long h)  { if (tag_of(h) == 10) { return fb(h); } return H_UNDEF; }

/* ------------------------------------------------------------- startup --- */

void seed_host(long obj, const char *name, long id) { obj_put(obj, mk_cstr(name), mk_host(id)); }
void seed_root(const char *name, long v) { scope_put(G_ROOT, mk_cstr(name), v); }

void boot(void) {
	union DB u;
	long math;
	long strobj;
	long arrobj;
	long jbi;
	/* main's own try-barrier pool. It is malloc'd rather than a global array
	 * because JBP has to be able to point at EITHER it or a coroutine's, and a
	 * pointer to one thing is simpler than an index into two. */
	JB_MAIN = (long)malloc(8 * 512);
	JB_CAP = 512;
	jbi = 0;
	while (jbi < 512) { long *jp = (long *)JB_MAIN; jp[jbi] = 0; jbi = jbi + 1; }
	JBP = JB_MAIN;
	ar_init();
	u.d = 0.0; DZERO = u.l;
	u.d = 1.0; DONE = u.l;
	DINF = 9218868437227405312;
	DNINF = DINF ^ (0 - 9223372036854775807 - 1);
	DNAN = DINF + 1;
	H_UNDEF = 0;
	H_NULL = 1;
	H_FALSE = 2;
	H_TRUE = 3;
	RETSLOT = H_UNDEF;
	JB_DEPTH = 0;
	THROWN = 0;
	OUTFD = 1;
	G_ROOT = mk_scope(0);

	seed_root("println", mk_host(1));
	seed_root("print", mk_host(2));
	seed_root("eprintln", mk_host(3));
	seed_root("parseInt", mk_host(4));
	seed_root("parseFloat", mk_host(5));
	seed_root("exit", mk_host(6));
	seed_root("byteLen", mk_host(32));
	seed_root("sprintf", mk_host(33));
	seed_root("printf", mk_host(34));
	seed_root("sprint", mk_host(35));
	seed_root("fail", mk_host(36));
	seed_root("sint", mk_host(40));
	seed_root("sintIs", mk_host(41));
	seed_root("sintHi", mk_host(42));
	seed_root("sintLo", mk_host(43));
	seed_root("sintWidth", mk_host(44));
	seed_root("sintUns", mk_host(45));
	seed_root("sintOp", mk_host(46));
	seed_root("sintCmp", mk_host(47));
	seed_root("sintConv", mk_host(48));
	seed_root("sintStr", mk_host(49));
	seed_root("sintNum", mk_host(50));
	seed_root("flo", mk_host(51));
	seed_root("floIs", mk_host(52));
	seed_root("floNum", mk_host(53));
	seed_root("floStyle", mk_host(54));
	seed_root("floStr", mk_host(55));
	seed_root("floOp", mk_host(56));
	seed_root("floEq", mk_host(57));
	seed_root("floMax", mk_host(58));
	seed_root("floMin", mk_host(59));
	seed_root("floAbs", mk_host(60));
	seed_root("keysOf", mk_host(61));
	seed_root("sintRaw", mk_host(62));
	seed_root("delKey", mk_host(63));
	seed_root("scopeNew", mk_host(64));
	seed_root("scopeParent", mk_host(65));
	seed_root("scopeGet", mk_host(66));
	seed_root("scopeHas", mk_host(67));
	seed_root("scopeDecl", mk_host(68));
	seed_root("isGenerator", mk_host(69));
	seed_root("floPrec", mk_host(70));
	seed_root("Infinity", mk_num(DINF));
	seed_root("NaN", mk_num(DNAN));
	seed_root("anytype", cell_new(12));

	math = mk_obj();
	seed_host(math, "floor", 10);
	seed_host(math, "abs", 11);
	seed_host(math, "max", 12);
	seed_host(math, "min", 13);
	seed_host(math, "imul", 14);
	seed_host(math, "ceil", 15);
	seed_host(math, "trunc", 16);
	seed_host(math, "round", 17);
	seed_host(math, "sign", 18);
	seed_host(math, "sqrt", 19);
	seed_host(math, "pow", 20);
	{
		union DB p;
		p.d = 3.141592653589793;
		obj_put(math, mk_cstr("PI"), mk_num(p.l));
		p.d = 2.718281828459045;
		obj_put(math, mk_cstr("E"), mk_num(p.l));
	}
	seed_root("Math", math);

	strobj = mk_obj();
	seed_host(strobj, "fromCharCode", 30);
	seed_root("String", strobj);

	arrobj = mk_obj();
	seed_host(arrobj, "isArray", 31);
	seed_root("Array", arrobj);

	/* The collector's trigger, made explicit and testable rather than implicit.
	 *   MEC_GC=off      never collect (the pre-1b behaviour, for measurement)
	 *   MEC_GC=stress   collect at EVERY allocation - turns a rare use-after-free
	 *                   into a deterministic one
	 *   MEC_GC=poison   collect on the ordinary threshold, but RETIRE every swept
	 *                   block and fill it with a non-value instead of reusing it,
	 *                   so a root the mark pass missed shows up as a wrong answer
	 *                   rather than as memory that happens to still hold the right
	 *                   bytes. (Deliberately NOT stress as well: retiring means the
	 *                   heap only grows, and collecting per allocation on top of
	 *                   that is quadratic - it did not finish on the Lua ratchet.) */
	{
		char *e = getenv("MEC_GC");
		if (e != 0) {
			if (e[0] == 111) { GC_MODE = 1; }     /* o */
			if (e[0] == 115) { GC_MODE = 2; }     /* s */
			if (e[0] == 112) { GC_MODE = 3; }     /* p */
		}
	}
	GC_ON = 1;
}

int main(void) {
	long r;
	long buf;
	long anchor = 0;
	/* The far end of the C stack scan. The margin covers main's other locals,
	 * whichever slots the compiler gave them. */
	GC_STACK_BASE = ((long)&anchor) + 256;
	boot();
	buf = jb_at(0);
	JB_DEPTH = 1;
	if (setjmp((void *)buf) != 0) {
		long s = to_string(THROWN);
		OUTFD = 2;
		o_cstr("js runtime error: uncaught exception: ");
		o_str(s);
		o_ch(10);
		OUTFD = 1;
		exit(1);
	}
	r = jsmain(0, 0);
	JB_DEPTH = 0;
	/* MEC_GC_STATS=1 reports the collector on stderr, so "it ran" is a
	 * measurement rather than an assumption. Off by default: every suite here
	 * compares stderr byte for byte. */
	if (getenv("MEC_GC_STATS") != 0) {
		OUTFD = 2;
		o_cstr("gc: collections=");
		o_str(to_string(mk_num(d_from_long(GC_COUNT))));
		o_cstr(" live=");
		o_str(to_string(mk_num(d_from_long(GC_LIVE))));
		o_cstr(" heap=");
		o_str(to_string(mk_num(d_from_long(GC_HEAP))));
		o_cstr(" pinned=");
		o_str(to_string(mk_num(d_from_long(PIN_N))));
		o_ch(10);
		OUTFD = 1;
	}
	{
		long n = to_number(r);
		if (d_is_nan(n)) { return 0; }
		return (int)to_int32(n);
	}
}
