/*
 * runtime.c - the C floor of the self-hosted runtime. It implements the js_*
 * externs that metajs-to-llvm-ir.abnf declares,
 * so that a MetaJS program compiled to handle IR can be linked into a NATIVE
 * BINARY instead of being interpreted with the Go runtime in abnf/jsrt.go.
 *
 * IT IS COMPILED BY languages/c-to-llvm-ir.abnf, NOT BY CLANG.
 * That is the point: every layer of the stack is compiled by this
 * repository. The link input is languages/lib/runtime.ll, regenerated with
 *
 *     tests/gen-runtime-ll.sh
 *
 * which runs our own C compiler over this file and keeps the module. clang only
 * ever sees IR that came out of a grammar in languages/.
 *
 * WHAT IT IS NOT
 *   - It is not a MOVING or an incremental collector. It is a stop-the-world
 *     mark/sweep over the heap, with the C stack scanned conservatively so no
 *     shadow stack has to be emitted (see "the heap and its GC" below). Nothing
 *     is ever moved - conservative roots forbid it - so there is no compaction,
 *     and a chunk once malloc'd is never handed back to the OS.
 *   - It is not a full ECMAScript runtime. It implements the MetaJS subset that
 *     metajs-to-llvm-ir.abnf emits, matching abnf/jsrt.go where it can.
 *   - It is not Unicode-complete. Strings carry the UTF-16 code unit view that
 *     abnf/jsrt.go exposes (.length, charCodeAt, charAt, slice, substring,
 *     indexOf, split, and the WTF-8 half of an astral pair), and case mapping
 *     is the simple Unicode mapping only (see "Unicode simple case mapping").
 *   - Floating point is the soft float that c-to-llvm-ir.abnf emits, correctly
 *     rounded IEEE-754, so it agrees with the Go runtime bit for bit.
 *
 * VALUE REPRESENTATION
 * A handle is a pointer to a Cell, carried in the i64 the emitted IR passes
 * around, and Cell.tag says what it is. The four singletons are the exception:
 * lib/compile-core.js emits the CONSTANTS 0 undefined, 1 null, 2 false, 3 true
 * straight into the IR, so they are never cells and every accessor answers for
 * them without dereferencing anything (see tag_of / fa).
 *
 * C SUBSET NOTES - constructs deliberately avoided because c-to-llvm-ir.abnf
 * does not handle them:
 *   - Much of this file carries a number as its raw IEEE-754 BIT PATTERN in a
 *     long and materialises a double only inside the function that computes
 *     with it, through `union DB`: an earlier c-to-llvm-ir.abnf could not pass
 *     or return a double. The later ports (pow, the Math members) take and
 *     return doubles directly.
 *   - `*p = v` through a `long *` does not compile ("store operands are not
 *     compatible: src=i64; dst=i32*"). `p[0] = v` compiles and is used instead.
 *   - object-like #define is parsed but NOT expanded, so there are no macros.
 */

int putchar(int c);
int puts(const char *s);
long write(int fd, const char *buf, unsigned long n);
long read(int fd, char *buf, unsigned long n);
int fflush(void *stream);
void *malloc(unsigned long n);
char *getenv(const char *name);
int getentropy(void *buf, unsigned long n);
long time(long *t);
long clock(void);
void *localtime_r(long *t, long *tm);
int setjmp(void *env);
void longjmp(void *env, int v);
void exit(int code);

/* ---- coroutines: the stack switch the generator primitive stands on --------
 * lib/coro-switch.S, the one hand-written file of the floor, for arm64 and
 * x86-64. mec_coro_switch pushes the callee-saved registers, stores the stack
 * pointer in *save and continues on the stack `to` saved; mec_coro_boot does the
 * same save and then calls coro_entry(cb) on the fresh stack whose top is `top`.
 * A coroutine stack is valloc'd with its lowest 16 KB made inaccessible, so an
 * overflow faults as it does on a thread stack. */
void mec_coro_switch(long *save, long to);
void mec_coro_boot(long *save, long top, long cb);
void *valloc(unsigned long n);
int mprotect(void *addr, unsigned long n, int prot);
void free(void *p);

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
 * through the soft float's i2d/d2i: the emitted i2d loses the low bits above
 * 2^53 and the d2i round trip can make a large integral double look
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
	 * exactly -2^63, so it is answered directly (si_float passes a raw int64). */
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
/* The truncated remainder, a faithful port of Go's math.Mod on the same
 * Frexp/Ldexp this file already carries. x - trunc(x/y)*y is NOT enough: it is
 * right only while the quotient fits 53 bits (`10 % 0.1` must be
 * 0.09999999999999945, not 0). The body is defined further down, after
 * d_frexp/d_ldexp. */
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
		/* |t| >= 2^63, and ToInt32 is a MODULO, not a range test: `1e20 | 0` is
		 * 1661992960 and `1e19 | 0` is -1981284352. abnf/jsrt.go's rt.toInt32
		 * matches.
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
 * A MARK/SWEEP heap.
 *
 * WHY MARK/SWEEP AND NOT REFCOUNTING. A scope holds its parent and a closure
 * holds its defining scope, so a closure defined inside a function CYCLES with
 * the scope that names it - the common case, not an edge case.
 *
 * WHY THE C STACK IS SCANNED CONSERVATIVELY, AND NOT A SHADOW STACK IN THE
 * EMITTER. The emitted IR runs in TWO worlds - natively a handle is a Cell *,
 * under llvm.Run it is an index into a Go table - and the two must stay
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
 * INTERIOR POINTERS ARE ROOTS (an exact-address test is unsound at -O2 - see
 * gc_try). Resolving one is exact and
 * constant time here: a block's address is base + idx * bsize, so the index is a
 * division.
 *
 * LAYOUT: SIZE-CLASS REGIONS WITH THE METADATA IN SIDE BITMAPS, AND NO PER-BLOCK
 * HEADER AT ALL (a per-block header was measurably costly). A CHUNK serves
 * exactly one size class and one kind, so the size and the kind are properties
 * of the chunk, and the "allocated" and "marked" flags are one bit each in two
 * side bitmaps indexed by the block's slot number. A handle IS the block
 * address; there is nothing in front of it.
 *
 *   chunk: [0] next chunk   [1] block size   [2] slot count   [3] slots bumped
 *          [4] alloc bitmap [5] mark bitmap  [6] kind (1 = cells)
 *          [7] next chunk of the same partition
 *          then 128 bytes of header padding, and the blocks
 *
 * A block over 1024 bytes gets a chunk of its own (one slot), so the scheme is
 * uniform and the sweep has no special case. Its size is rounded up to one of
 * eight steps per power of two (big_class), and a freed one goes on the free
 * list of its step and kind, so reusing one is a list pop: a first-fit walk of
 * one list is quadratic when a string grows a little at a time, because every
 * block freed before is too small. Blocks are never moved, never
 * coalesced and never split, so a slot number means the same thing for ever.
 *
 * The mark bitmap is CLEARED at the start of a collection rather than carrying
 * an epoch: clearing is heap/bsize/8 bytes - 6 KB for a 3 MB heap of cells.
 *
 * THERE ARE NO FREE LISTS for the small classes. After the mark, the set of
 * allocated blocks IS the set of marked ones, so the sweep swaps the two bitmap
 * pointers instead of visiting a slot, and the allocator hands out the next
 * clear alloc bit at or after a per-partition cursor, walking the partition's
 * chunks in order (the cursors rewind at every collection). A collection costs
 * the live set plus a few words per chunk, not the heap; an allocation is a
 * bitmap probe in address order rather than a chunk lookup per free-list pop.
 */

long AR_CHUNKS;      /* head of the chunk list */
long AR_PART[130];   /* the chunk a partition (class * 2 + kind) allocates from */
long AR_CUR[130];    /* the slot in AR_PART to resume the search at */
long AR_PHEAD[130];  /* the partition's first chunk; [7] links the rest */
long GC_BIG[880];    /* free blocks over 1024 bytes by big_class * 2 + kind: [0] next */
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
 * (one side runs at a time), so a suspended side has always published its
 * stack pointer and a setjmp of its callee-saved registers BEFORE parking, and
 * the collector reads both without any stop-the-world machinery.
 *
 *   CORO[i]   a live coroutine's control block (see coro_alloc for the layout)
 *   RES_*     the RESUMER's stack while a coroutine runs.  Resumes nest (a
 *             generator may drive another), so this is a LIFO.  RES_JBP/RES_JBD
 *             are the resumer's try-barrier pool and depth, parked with it.
 *
 * Both are GROWN rather than fixed: 10,000 suspended generators is a working
 * workload (each owns a 2 MB coroutine stack, of which it touches a few pages). The old array
 * is not freed, so doubling costs less than one extra copy of the final array in
 * total - the same bargain every other malloc in this file makes. */
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
long IN_BUF;         /* char *: standard input's buffer, for readIn */
long IN_POS;
long IN_LEN;

/* GEN_EXIT is the sentinel gen_close() throws INTO a suspended body so that the
 * `finally` clauses wrapping its yield run before the generator is abandoned -
 * JavaScript's return completion (Python's close() is a throw instead). It is one object,
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

/* ---- the PERMANENT region -------------------------------------------------
 * Objects that live for the whole run - everything runtime.c's boot() seeds, a
 * layer-2 library's scope, functions and constants (allocated between
 * js_gc_perm(1) and js_gc_perm(0) in its jsrtlib_boot), the interned string
 * literals and the small-integer cache - are allocated here instead of in the
 * chunks. Marking them every collection made the collector's cost grow with the
 * size of the language's runtime library although a program uses little of it.
 *
 * The region is ONE malloc: cells bump up from PERM_LO, raw buffers bump down
 * from PERM_HI, and it is full when the two meet - an allocation then falls
 * back to the ordinary heap, which is always sound. Nothing in it is ever
 * freed, marked or swept, and gc_try rejects an address in it with two
 * compares. Permanent objects are still MUTABLE (a library's module variables,
 * a table filled lazily, a scope that grew into a fresh buffer), so the region
 * is a ROOT: every collection traces each permanent cell by its tag and scans
 * the raw half conservatively. That costs a few compares per word, against a
 * chunk lookup, a mark and a sweep slot per object before.
 *
 * PERM_ON counts the open js_gc_perm brackets; non-zero routes ar_alloc_k here. */
long PERM_LO;
long PERM_HI;
long PERM_CTOP;
long PERM_RBOT;
long PERM_ON;
long SMC_YOUNG;      /* cached string literals NOT in the region: 0 skips the SMC root loop */
long NIC_YOUNG;      /* likewise for the small-integer cache */

/* ----- the three RECOVERABLE-ERROR HOOKS -----------------------------------
 *
 * scope_get, get_member and set_member are shared by all sixteen languages. By
 * default a miss is an uncatchable abort: `x` with no binding and `null.x` stop the
 * program dead where node throws a ReferenceError / a TypeError and CPython a
 * NameError. Making the abort a throw UNCONDITIONALLY is wrong - kotlin, swift,
 * java and metajs mean "this program is broken" by it, and the three
 * SHOULD-ABORT matrix rows are how the floor's abort path is reached by a test
 * at all - so the choice is the LANGUAGE's, made once at boot.
 *
 * The mechanism is a registered closure per site, and it costs nothing on the
 * hot path: the test is on the MISS path only, and a language that registers
 * nothing gets the abort. The closure ANSWERS the value to throw
 * rather than throwing it itself, which is what makes the hook non-recursive -
 * the slot is cleared for the duration of the call, so a handler that itself
 * reads a name nowhere bound aborts with the original diagnostic instead of
 * recursing forever, and it is restored before the throw (a throw longjmps, so
 * a restore placed after it would never run).
 *
 * The twins are abnf/jsrt.go's rt.missVar / rt.missMem, and the registering
 * externs are js_rt_missvar / js_rt_missmem below. */
long RT_MISS_VAR;    /* closure(name)      -> the value to throw, or 0 = abort */
long RT_MISS_MEM;    /* closure(key, obj)  -> the value to throw, or 0 = abort */
long RT_MISS_SET;    /* closure(key, obj)  -> the value to throw, or 0 = abort */
long RT_MISS_FN;     /* closure(callee, args) -> the call's result, or 0 = abort */
long RT_GEN_BUSY;    /* closure()          -> the value to throw, or 0 = abort (gen_busy) */
long RNG_STATE;      /* Math.random's xorshift64 state, 0 until the first call */
long FN_SRC_TEXT;    /* a js program's function source text (js_jssrc), or 0 */
long FN_SRC_TAB;     /* its span table; 0 while none is registered */
long FN_SRC_SE;      /* the table parsed: from at 2*idx, to at 2*idx+1, -1 = none */
long FN_SRC_N;       /* how many indexes FN_SRC_SE covers */

void gc_collect(void);

/* One bit per slot, in a malloc'd side array. */
void bit_set(long bm, long i) {
	char *b = (char *)bm;
	long one = 1;
	b[i >> 3] = (char)(((long)b[i >> 3] & 255) | (one << (i & 7)));
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

/* ---- THE CHUNK INDEX ------------------------------------------------------
 *
 * A sorted array of (payloadLo, payloadHi, chunk) triples, binary searched.
 * It exists because "which chunk holds this address" is THE HOTTEST QUESTION IN
 * THE COLLECTOR: gc_try below asks it for EVERY WORD of the conservatively
 * scanned C stack, of every raw buffer, and of every field of every traced
 * cell - and the overwhelming majority of those are not pointers at all. A
 * linear walk of the chunk list makes the collector O(chunks) per candidate
 * word, and chunks are not few: ar_block below takes 1 MB PER PARTITION, of
 * which there are up to 130.
 *
 * WHY AN ARRAY AND NOT A HASH. Chunk payloads are DISJOINT ADDRESS RANGES, not
 * keys - a candidate word points into the middle of one, never at its base - so
 * a hash of base addresses cannot answer the question at all without also
 * knowing the alignment, and chunk bases come from malloc and are not aligned to
 * anything useful. A sorted range array answers it in log2(chunks) with no
 * assumptions. Insertion is O(chunks) at CHUNK CREATION, which happens once per
 * megabyte of heap.
 *
 * The old index array is not freed when the array doubles: there are at most
 * log2(chunks) of them, the largest is 24 bytes per chunk, and this is a
 * malloc'd side table the collector must be able to read at any moment - the
 * same argument the mark bitmaps are allocated under. */
long AR_IX;          /* malloc'd triples lo,hi,chunk - sorted ascending by lo */
long AR_IX_N;
long AR_IX_CAP;
long AR_LO;          /* the lowest payload address of any chunk, and one past the */
long AR_HI;          /* highest: gc_try rejects anything outside without a search */

void ar_ix_add(long c, long lo, long hi) {
	long *a;
	long i;
	if (AR_IX_N >= AR_IX_CAP) {
		long ncap = (AR_IX_CAP == 0) ? 64 : AR_IX_CAP * 2;
		long na = (long)malloc(ncap * 24);
		long *dst = (long *)na;
		long k = 0;
		if (AR_IX_N > 0) {
			long *src = (long *)AR_IX;
			while (k < AR_IX_N * 3) { dst[k] = src[k]; k = k + 1; }
		}
		AR_IX = na;
		AR_IX_CAP = ncap;
	}
	a = (long *)AR_IX;
	i = AR_IX_N;
	while (i > 0 && a[(i - 1) * 3] > lo) {
		a[i * 3] = a[(i - 1) * 3];
		a[i * 3 + 1] = a[(i - 1) * 3 + 1];
		a[i * 3 + 2] = a[(i - 1) * 3 + 2];
		i = i - 1;
	}
	a[i * 3] = lo;
	a[i * 3 + 1] = hi;
	a[i * 3 + 2] = c;
	AR_IX_N = AR_IX_N + 1;
	if (lo < AR_LO) { AR_LO = lo; }
	if (hi > AR_HI) { AR_HI = hi; }
}

/* The chunk whose PAYLOAD contains v, or 0. The ranges are disjoint (each is one
 * malloc), so the answer is the last range whose lo is <= v, if v is inside it.
 * An address in a chunk's 128 byte HEADER is inside no payload and answers 0. */
long ar_ix_find(long v) {
	long *a;
	long lo = 0;
	long hi = AR_IX_N;
	if (AR_IX_N == 0) { return 0; }
	a = (long *)AR_IX;
	while (lo < hi) {
		long mid = (lo + hi) / 2;
		if (a[mid * 3] <= v) { lo = mid + 1; } else { hi = mid; }
	}
	if (lo == 0) { return 0; }
	lo = lo - 1;
	if (v < a[lo * 3 + 1]) { return a[lo * 3 + 2]; }
	return 0;
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
	/* The SWEEP walks the linked list (it wants every chunk, in any order); the
	 * address LOOKUP uses the index. */
	ar_ix_add(c, c + 128, c + 128 + bsize * nblk);
	return c;
}

/* Which chunk holds address v, or 0. */
long chunk_of(long v) { return ar_ix_find(v); }

/* The step of a block over 1024 bytes: for 2^k < need <= 2^(k+1) the size is
 * rounded up to a multiple of 2^(k-3), at most 12.5% more, and the class is
 * (k - 10) * 8 + the step's number. The rounded size is left in BIG_SIZE. */
long BIG_SIZE;
long big_class(long need) {
	long k = 10;
	long step;
	while ((2048 << (k - 10)) < need) { k = k + 1; }
	step = 128 << (k - 10);
	BIG_SIZE = ((need + step - 1) / step) * step;
	return (k - 10) * 8 + BIG_SIZE / step - 9;
}

/* A block of exactly `need` bytes and kind `kind`: the first clear alloc bit
 * at or after the partition's cursor, in its chunks in list order, then a new
 * chunk at the end of the list. A whole byte of set bits is skipped at once. A
 * block over 1024 bytes comes from its step's free list or a chunk of its own. */
long ar_block(long need, long kind) {
	long cls = need >> 4;
	long b;
	if (cls > 64) {
		long ix = big_class(need) * 2 + kind;
		b = GC_BIG[ix];
		if (b != 0) {
			long *w = (long *)b;
			long *cw = (long *)chunk_of(b);
			GC_BIG[ix] = w[0];
			bit_set(cw[4], 0);
			return b;
		}
		{
			long c = chunk_new(BIG_SIZE, 1, kind);
			long *cw = (long *)c;
			cw[3] = 1;
			bit_set(cw[4], 0);
			return c + 128;
		}
	}
	{
		long part = cls * 2 + kind;
		long cur = AR_PART[part];
		long one = 1;
		while (cur != 0) {
			long *cw = (long *)cur;
			char *bm = (char *)cw[4];
			long n = cw[2];
			long i = AR_CUR[part];
			while (i < n) {
				long by = (long)bm[i >> 3] & 255;
				if (by == 255) { i = (i | 7) + 1; }
				else if (((by >> (i & 7)) & 1) != 0) { i = i + 1; }
				else {
					bm[i >> 3] = (char)(by | (one << (i & 7)));
					if (i >= cw[3]) { cw[3] = i + 1; }
					AR_CUR[part] = i + 1;
					return cur + 128 + i * need;
				}
			}
			AR_CUR[part] = n;
			if (cw[7] == 0) { break; }
			cur = cw[7];
			AR_PART[part] = cur;
			AR_CUR[part] = 0;
		}
		{
			/* 1 MB of blocks per partition. The pages are only touched as the
			 * bump pointer reaches them, so an unused class costs address space
			 * and no resident memory. */
			long nblk = 1048576 / need;
			long c = chunk_new(need, nblk, kind);
			long *cw = (long *)c;
			if (cur == 0) { AR_PHEAD[part] = c; }
			else { long *pw = (long *)cur; pw[7] = c; }
			AR_PART[part] = c;
			AR_CUR[part] = 1;
			cw[3] = 1;
			bit_set(cw[4], 0);
			return c + 128;
		}
	}
}

/* A block of the permanent region, or 0 when it is full. Every cell is 64 bytes
 * (cell_new is the only kind-1 allocation), which is what lets gc_roots walk
 * the cell half by stride. */
long perm_block(long need, long kind) {
	long b;
	if (kind == 1) {
		if (need != 64 || PERM_CTOP + 64 > PERM_RBOT) { return 0; }
		b = PERM_CTOP;
		PERM_CTOP = b + 64;
		return b;
	}
	if (PERM_RBOT - need < PERM_CTOP) { return 0; }
	PERM_RBOT = PERM_RBOT - need;
	return PERM_RBOT;
}
int is_perm(long h) { return h >= PERM_LO && h < PERM_HI; }
long perm_used(void) { return (PERM_CTOP - PERM_LO) + (PERM_HI - PERM_RBOT); }

/* kind 1 = the payload is a Cell (traced by its tag), 0 = a raw buffer. */
char *ar_alloc_k(long n, long kind) {
	long need = (n + 15) & (0 - 16);
	long b;
	if (need < 16) { need = 16; }
	if (PERM_ON != 0) {
		b = perm_block(need, kind);
		if (b != 0) { return (char *)b; }
	}
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
	AR_CHUNKS = 0; GC_ON = 0;
	AR_IX = 0; AR_IX_N = 0; AR_IX_CAP = 0;
	AR_LO = 9223372036854775807; AR_HI = 0;
	GC_ALLOCED = 0; GC_LIVE = 0; GC_HEAP = 0; GC_COUNT = 0; PIN_N = 0;
	/* A quarter of the floor a collection leaves behind, so a short program
	 * still gets one collection instead of running the whole matrix without
	 * ever entering the collector. */
	GC_THRESH = 1048576;
	while (i < 130) { AR_PART[i] = 0; AR_CUR[i] = 0; AR_PHEAD[i] = 0; i = i + 1; }
	i = 0;
	while (i < 880) { GC_BIG[i] = 0; i = i + 1; }
	GC_MSTACK = (long)malloc(65536 * 8);
	GC_MTOP = 0;
	GC_MCAP = 65536;
	GC_REGS = (long)malloc(512);
	/* Address space only: the pages are touched as the region fills. */
	PERM_LO = (long)malloc(8388608);
	PERM_HI = PERM_LO + 8388608;
	PERM_CTOP = PERM_LO;
	PERM_RBOT = PERM_HI;
	PERM_ON = 0;
	SMC_YOUNG = 0;
	NIC_YOUNG = 0;
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
 * at -O2 a derived pointer can be the only surviving reference. In u_slice, during
 * str_from_units' allocation, the only live reference may be `u + b`, an interior
 * pointer into the UTF-16 unit buffer, after the compiler has dropped the string
 * handle `h`. An exact-address test would free the cell and its unit buffer with
 * it (MEC_GC=stress shows it in string.sub).
 *
 * Resolving one is EXACT and needs no start bitmap: a chunk holds one size
 * class, so the slot is `(v - base) / bsize` and the block is that slot.
 *
 * THIS IS THE HOTTEST FUNCTION IN THE COLLECTOR: it is asked about every word of
 * the conservatively scanned C stack and of every raw buffer, and most of those
 * are not pointers. The chunk lookup is ar_ix_find's binary search, so the
 * cheap rejections come first: a small integer, the bytes of a string, a raw
 * double all fall outside [AR_LO, AR_HI), and a permanent object is never
 * marked (the region is traced as a root instead). */
void gc_try_chunk(long v, int deep) {
	long c = ar_ix_find(v);
	long *w;
	if (c == 0) { return; }
	w = (long *)c;
	{
		long idx = (v - c - 128) / w[1];
		if (idx < w[3]) { gc_mark(c, idx, deep); }
	}
}
void gc_try(long v, int deep) {
	if (v < AR_LO || v >= AR_HI) { return; }
	if (v >= PERM_LO && v < PERM_HI) { return; }
	gc_try_chunk(v, deep);
}

/* The children of the cell at b. w[0..6] IS the Cell: tag, a, b, c, d, e, f -
 * the block address is the cell address (there is no header in front of it). */
void gc_trace_cell(long b) {
	long *w = (long *)b;
	long t = w[0];
	if (t == 4)  { gc_try(w[1], 0); gc_try(w[3], 0); return; }  /* bytes, units */
	if (t == 5)  { gc_try(w[1], 1); gc_try(w[5], 1); return; }  /* elements, named properties */
	if (t == 6)  { gc_try(w[1], 1); gc_try(w[2], 1); return; }  /* keys, values */
	/* 7 8 9 16 are the function kinds: e holds the name js_jsfnmeta set. */
	if (t == 7)  { gc_try(w[2], 1); gc_try(w[5], 1); return; }  /* closure env, name */
	if (t == 8)  { gc_try(w[5], 1); return; }                   /* name */
	if (t == 9)  { gc_try(w[1], 1); gc_try(w[5], 1); return; }  /* bound receiver, name */
	if (t == 10) { gc_try(w[2], 1); return; }                   /* signal value */
	if (t == 11) { gc_try(w[1], 1); gc_try(w[6], 1); return; }  /* names+vals, parent */
	/* 15 GENERATOR: a fn, b args, d lastValue, f sent/return/thrown value are
	 * handles; c is the RAW control-block pointer and e a raw flag word.  16
	 * generator FUNCTION: a is the closure it wraps. */
	if (t == 15) { gc_try(w[1], 1); gc_try(w[2], 1); gc_try(w[4], 1); gc_try(w[6], 1); return; }
	if (t == 16) { gc_try(w[1], 1); gc_try(w[5], 1); return; }
	/* 3 number, 12 anytype, 13 sized integer, 14 boxed double:
	 * no children - every field of those is a raw long, not a handle. */
}

void gc_trace(long b, long c) {
	long *cw = (long *)c;
	if (cw[6] == 0) {
		/* A raw buffer reached from the C stack. Its shape is not known here
		 * (a scope's names/values/type-classes share one block, a string's
		 * unit array holds code points), so every word in it is a candidate. */
		long q = b;
		long end = b + cw[1];
		while (q + 8 <= end) { long *x = (long *)q; gc_try(x[0], 1); q = q + 8; }
		return;
	}
	gc_trace_cell(b);
}

void gc_drain(void) {
	while (GC_MTOP > 0) {
		long *ms = (long *)GC_MSTACK;
		GC_MTOP = GC_MTOP - 2;
		gc_trace(ms[GC_MTOP], ms[GC_MTOP + 1]);
	}
}

/* gc_try's two rejections are spelled out here because most words of a scanned
 * range are not heap pointers, and clang does not inline gc_try. */
void gc_scan_range(long lo, long hi) {
	long p = (lo + 7) & (0 - 8);
	while (p < hi) {
		long *x = (long *)p;
		long v = x[0];
		if (v >= AR_LO && v < AR_HI && (v < PERM_LO || v >= PERM_HI)) { gc_try_chunk(v, 1); }
		p = p + 8;
	}
}

/* The marks are CLEARED here rather than carried in a per-block epoch: it is one
 * pass over (slots bumped)/8 bytes per chunk. */
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

/* Allocated := marked, by swapping the two bitmaps; the old alloc bitmap is the
 * next collection's (cleared) mark bitmap. Only a block over 1024 bytes and
 * poison mode need to see each freed slot. */
void gc_sweep(void) {
	long c = AR_CHUNKS;
	while (c != 0) {
		long *cw = (long *)c;
		long n = cw[3];
		long bs = cw[1];
		long i = 0;
		if (GC_MODE != 3 && bs <= 1024) { n = 0; }
		while (i < n) {
			if (bit_get(cw[4], i) != 0 && bit_get(cw[5], i) == 0) {
				long b = c + 128 + i * bs;
				long *pw = (long *)b;
				if (GC_MODE == 3) {
					/* Poison mode: the block is RETIRED - its allocation bit is
					 * clear and the cursors never rewind in this mode, so it is
					 * never handed out again - and its payload is filled with a
					 * value that is neither a tag nor a handle, so a root the mark
					 * pass FAILED to reach shows up as a deterministic wrong answer
					 * instead of a rare use-after-free. */
					long q = 0;
					while (q * 8 < bs) { pw[q] = 195948557; q = q + 1; }
				} else {
					long ix = big_class(bs) * 2 + cw[6];
					pw[0] = GC_BIG[ix]; GC_BIG[ix] = b;
				}
			}
			i = i + 1;
		}
		{ long t = cw[4]; cw[4] = cw[5]; cw[5] = t; }
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
	if (GC_MODE != 3) {
		i = 0;
		while (i < 130) { AR_PART[i] = AR_PHEAD[i]; AR_CUR[i] = 0; i = i + 1; }
	}
	GC_ALLOCED = 0;
	/* The permanent region is live by definition; counting it keeps the
	 * threshold what it would be had those objects been marked. */
	GC_LIVE = GC_LIVE + perm_used();
	/* The next collection comes after as many bytes as are live now, but not
	 * before 4 MB: below that a collection's fixed cost (the roots, the
	 * permanent region, the bitmaps) outweighs what it frees, and a program
	 * with a small live set would collect every few hundred calls. */
	GC_THRESH = GC_LIVE;
	if (GC_THRESH < 4194304) { GC_THRESH = 4194304; }
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

/* Fill slot v of the small-integer cache. A cached cell lives as long as the
 * cache, so it is allocated permanent. */
long nic_fill(long v, long bits) {
	long h;
	PERM_ON = PERM_ON + 1;
	h = mk_num_raw(bits);
	PERM_ON = PERM_ON - 1;
	if (!is_perm(h)) { NIC_YOUNG = NIC_YOUNG + 1; }
	NIC[v + 256] = h;
	return h;
}

long mk_num(long bits) {
	long v = small_int_bits(bits);
	long h;
	if (v < 0) { return mk_num_raw(bits); }
	h = NIC[v + 256];
	if (h != 0) { return h; }
	return nic_fill(v, bits);
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
 * does. Open addressing with linear probing and NO eviction gives exactly one
 * allocation per distinct literal for the lifetime of the program. The probe is
 * bounded, so a table that somehow fills degrades to allocating instead of
 * looping.
 *
 * THE HOME SLOT IS A MULTIPLICATIVE HASH of the address (bits 19..31 of
 * k * 2654435761, Knuth's 2^32/phi). Literals sit contiguously in each module,
 * a few bytes apart, so a hash that keeps the low address bits maps a whole
 * module onto one band of slots, and the bands of runtime.c, layer 2 and the
 * program overlap wherever their addresses happen to fall: a hot literal then
 * probes dozens of slots on every call, and how many depends on the link
 * layout.
 *
 * A HIT AFTER PROBING TAKES OVER ITS HOME SLOT, so a literal a loop evaluates
 * is found by str_intern's one compare from its second lookup on, whatever was
 * inserted first; only two hot literals with the same home keep trading the
 * slot. That is sound for linear probing without deletion: the entry it
 * displaces lands further along the same unbroken run it was in, so its own
 * probe from its home still reaches it. */
long SMC_PTR[8192];
long SMC_LEN[8192];
long SMC_VAL[8192];

/* The lookup when the literal is not in its home slot: the probe, the takeover
 * of the home slot, the insertion. Kept out of str_intern so the hit at home
 * is a leaf with no frame to build. */
long smc_miss(const char *p, long n, long home) {
	long k = (long)p;
	long slot = home;
	long probes = 0;
	long h;
	while (SMC_PTR[slot] != 0 && probes < 64) {
		if (SMC_PTR[slot] == k && SMC_LEN[slot] == n) {
			h = SMC_VAL[slot];
			if (probes != 0) {
				SMC_PTR[slot] = SMC_PTR[home];
				SMC_LEN[slot] = SMC_LEN[home];
				SMC_VAL[slot] = SMC_VAL[home];
				SMC_PTR[home] = k;
				SMC_LEN[home] = n;
				SMC_VAL[home] = h;
			}
			return h;
		}
		slot = (slot + 1) & 8191;
		probes = probes + 1;
	}
	if (probes >= 64) { return mk_str(p, n); }
	/* A cached literal lives as long as the cache: permanent. */
	PERM_ON = PERM_ON + 1;
	h = mk_str(p, n);
	PERM_ON = PERM_ON - 1;
	if (!is_perm(h)) { SMC_YOUNG = SMC_YOUNG + 1; }
	SMC_PTR[slot] = k;
	SMC_LEN[slot] = n;
	SMC_VAL[slot] = h;
	return h;
}

long str_intern(const char *p, long n) {
	long home = (((long)p * 2654435761) >> 19) & 8191;
	if (SMC_PTR[home] == (long)p && SMC_LEN[home] == n) { return SMC_VAL[home]; }
	return smc_miss(p, n, home);
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

/* RT_RAW_STR is set (js_rt_rawstr) by a language whose strings are BYTES, go:
 * three bytes that spell a surrogate in UTF-8 are bytes like any others there,
 * so str_cat does not rejoin two of them and print writes them as they are. */
long RT_RAW_STR;
long js_rt_rawstr(void) { RT_RAW_STR = 1; return 0; }

/* str_cat is `a + b`: plain concatenation, except that a surrogate pair split
 * across the seam is rejoined into its astral character (Go's strConcat), unless
 * the strings are bytes (RT_RAW_STR). */
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
	if (nx >= 3 && ny >= 3 && ((long)px[nx - 3] & 255) == 237 && ((long)py[0] & 255) == 237 && RT_RAW_STR == 0) {
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
 * shows (Go's wtf8Clean); a byte string (RT_RAW_STR) is printed as it is. */
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

/* str_eq against a C literal, WITHOUT building a string cell for the literal:
 * method_id_array/method_id_string test every candidate name this way on every
 * member lookup, so allocating here would be a dead cell per candidate. A NUL
 * inside the JS string is compared honestly: the literal ends there, so the
 * answer is false rather than an over-read. */
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
 * fixed for the life of the process (the only two call sites that pass a
 * variable, seed_host and seed_root, are themselves called with literals from
 * boot). It therefore goes through the same literal cache js_str_mem uses, so
 * `to_string(undefined)`, `typeof x` and the array/string method names do not
 * allocate per CALL. */
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

#include "case-simple.h"

/* ----- full case mapping: the twin of abnf/casemap.go -----
 *
 * case_str(s, mode) maps a string's case. mode is a bit set: 1 upper (else
 * lower), 2 FULL - SpecialCasing.txt's unconditional one-to-many mappings
 * ("ß" -> "SS", "İ" -> "i̇") - and 4 Final_Sigma: a capital sigma lowercases to
 * "ς" when a cased letter precedes it and none follows, case-ignorable
 * characters skipped both ways; 8 maps every character to its titlecase
 * instead, 16 to its full case folding (with 2: "ß" -> "ss", "İ" -> "i̇").
 * toUpperCase is 3 and toLowerCase 6, as in
 * ECMAScript; unicodeCase(s, mode) (host id 97) hands any mode to layer 2. The
 * tables are Unicode 16, generated from CPython 3.14's str.upper/str.lower.
 * CASED_*: cased letters that are not also case-ignorable, the only ones the
 * Final_Sigma context can stop at. CASEIGN_*: Case_Ignorable. UPFULL_* and
 * TIFULL_*: the uppercase and titlecase mappings longer than one character,
 * FOLD_*: the case foldings that differ from the lowercase, three slots each,
 * 0-padded. */
long NCASED = 152;
long NCASEIGN = 452;
long NUPFULL = 102;
long NTIFULL = 48;
long NFOLD = 297;
static long CASED_LO[152] = {
	65, 97, 170, 181, 186, 192, 216, 248, 444, 452, 661, 880,
	886, 891, 895, 902, 904, 908, 910, 931, 1015, 1162, 1329, 1376,
	4256, 4295, 4301, 4304, 4349, 5024, 5112, 7296, 7312, 7357, 7424, 7531,
	7545, 7680, 7960, 7968, 8008, 8016, 8025, 8027, 8029, 8031, 8064, 8118,
	8126, 8130, 8134, 8144, 8150, 8160, 8178, 8182, 8450, 8455, 8458, 8469,
	8473, 8484, 8486, 8488, 8490, 8495, 8505, 8508, 8517, 8526, 8544, 8579,
	9398, 11264, 11390, 11499, 11506, 11520, 11559, 11565, 42560, 42624, 42786, 42865,
	42891, 42896, 42960, 42963, 42965, 42997, 43002, 43824, 43872, 43888, 64256, 64275,
	65313, 65345, 66560, 66736, 66776, 66928, 66940, 66956, 66964, 66967, 66979, 66995,
	67003, 68736, 68800, 68944, 68976, 71840, 93760, 119808, 119894, 119966, 119970, 119973,
	119977, 119982, 119995, 119997, 120005, 120071, 120077, 120086, 120094, 120123, 120128, 120134,
	120138, 120146, 120488, 120514, 120540, 120572, 120598, 120630, 120656, 120688, 120714, 120746,
	120772, 122624, 122635, 122661, 125184, 127280, 127312, 127344,
};
static long CASED_HI[152] = {
	90, 122, 170, 181, 186, 214, 246, 442, 447, 659, 687, 883,
	887, 893, 895, 902, 906, 908, 929, 1013, 1153, 1327, 1366, 1416,
	4293, 4295, 4301, 4346, 4351, 5109, 5117, 7306, 7354, 7359, 7467, 7543,
	7578, 7957, 7965, 8005, 8013, 8023, 8025, 8027, 8029, 8061, 8116, 8124,
	8126, 8132, 8140, 8147, 8155, 8172, 8180, 8188, 8450, 8455, 8467, 8469,
	8477, 8484, 8486, 8488, 8493, 8500, 8505, 8511, 8521, 8526, 8575, 8580,
	9449, 11387, 11492, 11502, 11507, 11557, 11559, 11565, 42605, 42651, 42863, 42887,
	42894, 42957, 42961, 42963, 42972, 42998, 43002, 43866, 43880, 43967, 64262, 64279,
	65338, 65370, 66639, 66771, 66811, 66938, 66954, 66962, 66965, 66977, 66993, 67001,
	67004, 68786, 68850, 68965, 68997, 71903, 93823, 119892, 119964, 119967, 119970, 119974,
	119980, 119993, 119995, 120003, 120069, 120074, 120084, 120092, 120121, 120126, 120132, 120134,
	120144, 120485, 120512, 120538, 120570, 120596, 120628, 120654, 120686, 120712, 120744, 120770,
	120779, 122633, 122654, 122666, 125251, 127305, 127337, 127369,
};
static long CASEIGN_LO[452] = {
	39, 46, 58, 94, 96, 168, 173, 175, 180, 183, 688, 884,
	890, 900, 903, 1155, 1369, 1375, 1425, 1471, 1473, 1476, 1479, 1524,
	1536, 1552, 1564, 1600, 1611, 1648, 1750, 1759, 1770, 1807, 1809, 1840,
	1958, 2027, 2042, 2045, 2070, 2137, 2184, 2192, 2199, 2249, 2362, 2364,
	2369, 2381, 2385, 2402, 2417, 2433, 2492, 2497, 2509, 2530, 2558, 2561,
	2620, 2625, 2631, 2635, 2641, 2672, 2677, 2689, 2748, 2753, 2759, 2765,
	2786, 2810, 2817, 2876, 2879, 2881, 2893, 2901, 2914, 2946, 3008, 3021,
	3072, 3076, 3132, 3134, 3142, 3146, 3157, 3170, 3201, 3260, 3263, 3270,
	3276, 3298, 3328, 3387, 3393, 3405, 3426, 3457, 3530, 3538, 3542, 3633,
	3636, 3654, 3761, 3764, 3782, 3784, 3864, 3893, 3895, 3897, 3953, 3968,
	3974, 3981, 3993, 4038, 4141, 4146, 4153, 4157, 4184, 4190, 4209, 4226,
	4229, 4237, 4253, 4348, 4957, 5906, 5938, 5970, 6002, 6068, 6071, 6086,
	6089, 6103, 6109, 6155, 6211, 6277, 6313, 6432, 6439, 6450, 6457, 6679,
	6683, 6742, 6744, 6752, 6754, 6757, 6771, 6783, 6823, 6832, 6912, 6964,
	6966, 6972, 6978, 7019, 7040, 7074, 7080, 7083, 7142, 7144, 7149, 7151,
	7212, 7222, 7288, 7376, 7380, 7394, 7405, 7412, 7416, 7468, 7544, 7579,
	8125, 8127, 8141, 8157, 8173, 8189, 8203, 8216, 8228, 8231, 8234, 8288,
	8294, 8305, 8319, 8336, 8400, 11388, 11503, 11631, 11647, 11744, 11823, 12293,
	12330, 12337, 12347, 12441, 12540, 40981, 42232, 42508, 42607, 42612, 42623, 42652,
	42736, 42752, 42864, 42888, 42994, 43000, 43010, 43014, 43019, 43045, 43052, 43204,
	43232, 43263, 43302, 43335, 43392, 43443, 43446, 43452, 43471, 43493, 43561, 43569,
	43573, 43587, 43596, 43632, 43644, 43696, 43698, 43703, 43710, 43713, 43741, 43756,
	43763, 43766, 43867, 43881, 44005, 44008, 44013, 64286, 64434, 65024, 65043, 65056,
	65106, 65109, 65279, 65287, 65294, 65306, 65342, 65344, 65392, 65438, 65507, 65529,
	66045, 66272, 66422, 67456, 67463, 67506, 68097, 68101, 68108, 68152, 68159, 68325,
	68900, 68942, 68969, 68975, 69291, 69372, 69446, 69506, 69633, 69688, 69744, 69747,
	69759, 69811, 69817, 69821, 69826, 69837, 69888, 69927, 69933, 70003, 70016, 70070,
	70089, 70095, 70191, 70196, 70198, 70206, 70209, 70367, 70371, 70400, 70459, 70464,
	70502, 70512, 70587, 70606, 70608, 70610, 70625, 70712, 70722, 70726, 70750, 70835,
	70842, 70847, 70850, 71090, 71100, 71103, 71132, 71219, 71229, 71231, 71339, 71341,
	71344, 71351, 71453, 71455, 71458, 71463, 71727, 71737, 71995, 71998, 72003, 72148,
	72154, 72160, 72193, 72243, 72251, 72263, 72273, 72281, 72330, 72344, 72752, 72760,
	72767, 72850, 72874, 72882, 72885, 73009, 73018, 73020, 73023, 73031, 73104, 73109,
	73111, 73459, 73472, 73526, 73536, 73538, 73562, 78896, 78919, 90398, 90413, 92912,
	92976, 92992, 93504, 93547, 94031, 94095, 94176, 94179, 110576, 110581, 110589, 113821,
	113824, 118528, 118576, 119143, 119155, 119173, 119210, 119362, 121344, 121403, 121461, 121476,
	121499, 121505, 122880, 122888, 122907, 122915, 122918, 122928, 123023, 123184, 123566, 123628,
	124139, 124398, 125136, 125252, 127995, 917505, 917536, 917760,
};
static long CASEIGN_HI[452] = {
	39, 46, 58, 94, 96, 168, 173, 175, 180, 184, 879, 885,
	890, 901, 903, 1161, 1369, 1375, 1469, 1471, 1474, 1477, 1479, 1524,
	1541, 1562, 1564, 1600, 1631, 1648, 1757, 1768, 1773, 1807, 1809, 1866,
	1968, 2037, 2042, 2045, 2093, 2139, 2184, 2193, 2207, 2306, 2362, 2364,
	2376, 2381, 2391, 2403, 2417, 2433, 2492, 2500, 2509, 2531, 2558, 2562,
	2620, 2626, 2632, 2637, 2641, 2673, 2677, 2690, 2748, 2757, 2760, 2765,
	2787, 2815, 2817, 2876, 2879, 2884, 2893, 2902, 2915, 2946, 3008, 3021,
	3072, 3076, 3132, 3136, 3144, 3149, 3158, 3171, 3201, 3260, 3263, 3270,
	3277, 3299, 3329, 3388, 3396, 3405, 3427, 3457, 3530, 3540, 3542, 3633,
	3642, 3662, 3761, 3772, 3782, 3790, 3865, 3893, 3895, 3897, 3966, 3972,
	3975, 3991, 4028, 4038, 4144, 4151, 4154, 4158, 4185, 4192, 4212, 4226,
	4230, 4237, 4253, 4348, 4959, 5908, 5939, 5971, 6003, 6069, 6077, 6086,
	6099, 6103, 6109, 6159, 6211, 6278, 6313, 6434, 6440, 6450, 6459, 6680,
	6683, 6742, 6750, 6752, 6754, 6764, 6780, 6783, 6823, 6862, 6915, 6964,
	6970, 6972, 6978, 7027, 7041, 7077, 7081, 7085, 7142, 7145, 7149, 7153,
	7219, 7223, 7293, 7378, 7392, 7400, 7405, 7412, 7417, 7530, 7544, 7679,
	8125, 8129, 8143, 8159, 8175, 8190, 8207, 8217, 8228, 8231, 8238, 8292,
	8303, 8305, 8319, 8348, 8432, 11389, 11505, 11631, 11647, 11775, 11823, 12293,
	12333, 12341, 12347, 12446, 12542, 40981, 42237, 42508, 42610, 42621, 42623, 42655,
	42737, 42785, 42864, 42890, 42996, 43001, 43010, 43014, 43019, 43046, 43052, 43205,
	43249, 43263, 43309, 43345, 43394, 43443, 43449, 43453, 43471, 43494, 43566, 43570,
	43574, 43587, 43596, 43632, 43644, 43696, 43700, 43704, 43711, 43713, 43741, 43757,
	43764, 43766, 43871, 43883, 44005, 44008, 44013, 64286, 64450, 65039, 65043, 65071,
	65106, 65109, 65279, 65287, 65294, 65306, 65342, 65344, 65392, 65439, 65507, 65531,
	66045, 66272, 66426, 67461, 67504, 67514, 68099, 68102, 68111, 68154, 68159, 68326,
	68903, 68942, 68973, 68975, 69292, 69375, 69456, 69509, 69633, 69702, 69744, 69748,
	69761, 69814, 69818, 69821, 69826, 69837, 69890, 69931, 69940, 70003, 70017, 70078,
	70092, 70095, 70193, 70196, 70199, 70206, 70209, 70367, 70378, 70401, 70460, 70464,
	70508, 70516, 70592, 70606, 70608, 70610, 70626, 70719, 70724, 70726, 70750, 70840,
	70842, 70848, 70851, 71093, 71101, 71104, 71133, 71226, 71229, 71232, 71339, 71341,
	71349, 71351, 71453, 71455, 71461, 71467, 71735, 71738, 71996, 71998, 72003, 72151,
	72155, 72160, 72202, 72248, 72254, 72263, 72278, 72283, 72342, 72345, 72758, 72765,
	72767, 72871, 72880, 72883, 72886, 73014, 73018, 73021, 73029, 73031, 73105, 73109,
	73111, 73460, 73473, 73530, 73536, 73538, 73562, 78912, 78933, 90409, 90415, 92916,
	92982, 92995, 93506, 93548, 94031, 94111, 94177, 94180, 110579, 110587, 110590, 113822,
	113827, 118573, 118598, 119145, 119170, 119179, 119213, 119364, 121398, 121452, 121461, 121476,
	121503, 121519, 122886, 122904, 122913, 122916, 122922, 122989, 123023, 123197, 123566, 123631,
	124143, 124399, 125142, 125259, 127999, 917505, 917631, 917999,
};
static long UPFULL_CP[102] = {
	223, 329, 496, 912, 944, 1415, 7830, 7831, 7832, 7833, 7834, 8016,
	8018, 8020, 8022, 8064, 8065, 8066, 8067, 8068, 8069, 8070, 8071, 8072,
	8073, 8074, 8075, 8076, 8077, 8078, 8079, 8080, 8081, 8082, 8083, 8084,
	8085, 8086, 8087, 8088, 8089, 8090, 8091, 8092, 8093, 8094, 8095, 8096,
	8097, 8098, 8099, 8100, 8101, 8102, 8103, 8104, 8105, 8106, 8107, 8108,
	8109, 8110, 8111, 8114, 8115, 8116, 8118, 8119, 8124, 8130, 8131, 8132,
	8134, 8135, 8140, 8146, 8147, 8150, 8151, 8162, 8163, 8164, 8166, 8167,
	8178, 8179, 8180, 8182, 8183, 8188, 64256, 64257, 64258, 64259, 64260, 64261,
	64262, 64275, 64276, 64277, 64278, 64279,
};
static long UPFULL_TO[306] = {
	83, 83, 0, 700, 78, 0, 74, 780, 0, 921, 776, 769,
	933, 776, 769, 1333, 1362, 0, 72, 817, 0, 84, 776, 0,
	87, 778, 0, 89, 778, 0, 65, 702, 0, 933, 787, 0,
	933, 787, 768, 933, 787, 769, 933, 787, 834, 7944, 921, 0,
	7945, 921, 0, 7946, 921, 0, 7947, 921, 0, 7948, 921, 0,
	7949, 921, 0, 7950, 921, 0, 7951, 921, 0, 7944, 921, 0,
	7945, 921, 0, 7946, 921, 0, 7947, 921, 0, 7948, 921, 0,
	7949, 921, 0, 7950, 921, 0, 7951, 921, 0, 7976, 921, 0,
	7977, 921, 0, 7978, 921, 0, 7979, 921, 0, 7980, 921, 0,
	7981, 921, 0, 7982, 921, 0, 7983, 921, 0, 7976, 921, 0,
	7977, 921, 0, 7978, 921, 0, 7979, 921, 0, 7980, 921, 0,
	7981, 921, 0, 7982, 921, 0, 7983, 921, 0, 8040, 921, 0,
	8041, 921, 0, 8042, 921, 0, 8043, 921, 0, 8044, 921, 0,
	8045, 921, 0, 8046, 921, 0, 8047, 921, 0, 8040, 921, 0,
	8041, 921, 0, 8042, 921, 0, 8043, 921, 0, 8044, 921, 0,
	8045, 921, 0, 8046, 921, 0, 8047, 921, 0, 8122, 921, 0,
	913, 921, 0, 902, 921, 0, 913, 834, 0, 913, 834, 921,
	913, 921, 0, 8138, 921, 0, 919, 921, 0, 905, 921, 0,
	919, 834, 0, 919, 834, 921, 919, 921, 0, 921, 776, 768,
	921, 776, 769, 921, 834, 0, 921, 776, 834, 933, 776, 768,
	933, 776, 769, 929, 787, 0, 933, 834, 0, 933, 776, 834,
	8186, 921, 0, 937, 921, 0, 911, 921, 0, 937, 834, 0,
	937, 834, 921, 937, 921, 0, 70, 70, 0, 70, 73, 0,
	70, 76, 0, 70, 70, 73, 70, 70, 76, 83, 84, 0,
	83, 84, 0, 1348, 1350, 0, 1348, 1333, 0, 1348, 1339, 0,
	1358, 1350, 0, 1348, 1341, 0,
};

/* c lies in one of the CASED / CASEIGN ranges. */
long case_cased(long c) {
	long a = 0;
	long b = NCASED - 1;
	long m;
	while (a <= b) {
		m = (a + b) / 2;
		if (c < CASED_LO[m]) { b = m - 1; }
		else if (c > CASED_HI[m]) { a = m + 1; }
		else { return 1; }
	}
	return 0;
}
long case_ignorable(long c) {
	long a = 0;
	long b = NCASEIGN - 1;
	long m;
	while (a <= b) {
		m = (a + b) / 2;
		if (c < CASEIGN_LO[m]) { b = m - 1; }
		else if (c > CASEIGN_HI[m]) { a = m + 1; }
		else { return 1; }
	}
	return 0;
}

/* The code point starting at unit i (a surrogate pair joined, a lone surrogate
 * itself), and how many units it takes. */
long case_cp_at(long s, long i, long n) {
	long cu = str_ucode(s, i);
	long t;
	if (cu >= 55296 && cu <= 56319 && i + 1 < n) {
		t = str_ucode(s, i + 1);
		if (t >= 56320 && t <= 57343) { return 65536 + ((cu - 55296) << 10) + (t - 56320); }
	}
	return cu;
}
/* The code point ENDING just before unit i, and how many units it takes. */
long case_cp_before(long s, long i) {
	long cu = str_ucode(s, i - 1);
	long h;
	if (cu >= 56320 && cu <= 57343 && i >= 2) {
		h = str_ucode(s, i - 2);
		if (h >= 55296 && h <= 56319) { return 65536 + ((h - 55296) << 10) + (cu - 56320); }
	}
	return cu;
}
long case_units(long cp) { return cp > 65535 ? 2 : 1; }

/* The capital sigma at unit i ends a word. Twin of caseFinalSigma. */
long case_final_sigma(long s, long i, long n) {
	long j = i;
	long c = -1;
	while (j > 0) {
		c = case_cp_before(s, j);
		j = j - case_units(c);
		if (!case_ignorable(c)) { break; }
		c = -1;
	}
	if (c < 0 || !case_cased(c)) { return 0; }
	j = i + 1;
	while (j < n) {
		c = case_cp_at(s, j, n);
		if (!case_ignorable(c)) { return !case_cased(c); }
		j = j + case_units(c);
	}
	return 1;
}

static long TIFULL_CP[48] = {
	223, 329, 496, 912, 944, 1415, 7830, 7831, 7832, 7833, 7834, 8016,
	8018, 8020, 8022, 8114, 8116, 8118, 8119, 8130, 8132, 8134, 8135, 8146,
	8147, 8150, 8151, 8162, 8163, 8164, 8166, 8167, 8178, 8180, 8182, 8183,
	64256, 64257, 64258, 64259, 64260, 64261, 64262, 64275, 64276, 64277, 64278, 64279,
};
static long TIFULL_TO[144] = {
	83, 115, 0, 700, 78, 0, 74, 780, 0, 921, 776, 769,
	933, 776, 769, 1333, 1410, 0, 72, 817, 0, 84, 776, 0,
	87, 778, 0, 89, 778, 0, 65, 702, 0, 933, 787, 0,
	933, 787, 768, 933, 787, 769, 933, 787, 834, 8122, 837, 0,
	902, 837, 0, 913, 834, 0, 913, 834, 837, 8138, 837, 0,
	905, 837, 0, 919, 834, 0, 919, 834, 837, 921, 776, 768,
	921, 776, 769, 921, 834, 0, 921, 776, 834, 933, 776, 768,
	933, 776, 769, 929, 787, 0, 933, 834, 0, 933, 776, 834,
	8186, 837, 0, 911, 837, 0, 937, 834, 0, 937, 834, 837,
	70, 102, 0, 70, 105, 0, 70, 108, 0, 70, 102, 105,
	70, 102, 108, 83, 116, 0, 83, 116, 0, 1348, 1398, 0,
	1348, 1381, 0, 1348, 1387, 0, 1358, 1398, 0, 1348, 1389, 0,
};

/* The index of cp in UPFULL_CP, or -1. */
long case_upfull(long cp) {
	long a = 0;
	long b = NUPFULL - 1;
	long m;
	while (a <= b) {
		m = (a + b) / 2;
		if (cp < UPFULL_CP[m]) { b = m - 1; }
		else if (cp > UPFULL_CP[m]) { a = m + 1; }
		else { return m; }
	}
	return -1;
}

static long FOLD_CP[297] = {
	181, 223, 329, 383, 496, 837, 912, 944, 962, 976, 977, 981,
	982, 1008, 1009, 1013, 1415, 5024, 5025, 5026, 5027, 5028, 5029, 5030,
	5031, 5032, 5033, 5034, 5035, 5036, 5037, 5038, 5039, 5040, 5041, 5042,
	5043, 5044, 5045, 5046, 5047, 5048, 5049, 5050, 5051, 5052, 5053, 5054,
	5055, 5056, 5057, 5058, 5059, 5060, 5061, 5062, 5063, 5064, 5065, 5066,
	5067, 5068, 5069, 5070, 5071, 5072, 5073, 5074, 5075, 5076, 5077, 5078,
	5079, 5080, 5081, 5082, 5083, 5084, 5085, 5086, 5087, 5088, 5089, 5090,
	5091, 5092, 5093, 5094, 5095, 5096, 5097, 5098, 5099, 5100, 5101, 5102,
	5103, 5104, 5105, 5106, 5107, 5108, 5109, 5112, 5113, 5114, 5115, 5116,
	5117, 7296, 7297, 7298, 7299, 7300, 7301, 7302, 7303, 7304, 7830, 7831,
	7832, 7833, 7834, 7835, 7838, 8016, 8018, 8020, 8022, 8064, 8065, 8066,
	8067, 8068, 8069, 8070, 8071, 8072, 8073, 8074, 8075, 8076, 8077, 8078,
	8079, 8080, 8081, 8082, 8083, 8084, 8085, 8086, 8087, 8088, 8089, 8090,
	8091, 8092, 8093, 8094, 8095, 8096, 8097, 8098, 8099, 8100, 8101, 8102,
	8103, 8104, 8105, 8106, 8107, 8108, 8109, 8110, 8111, 8114, 8115, 8116,
	8118, 8119, 8124, 8126, 8130, 8131, 8132, 8134, 8135, 8140, 8146, 8147,
	8150, 8151, 8162, 8163, 8164, 8166, 8167, 8178, 8179, 8180, 8182, 8183,
	8188, 43888, 43889, 43890, 43891, 43892, 43893, 43894, 43895, 43896, 43897, 43898,
	43899, 43900, 43901, 43902, 43903, 43904, 43905, 43906, 43907, 43908, 43909, 43910,
	43911, 43912, 43913, 43914, 43915, 43916, 43917, 43918, 43919, 43920, 43921, 43922,
	43923, 43924, 43925, 43926, 43927, 43928, 43929, 43930, 43931, 43932, 43933, 43934,
	43935, 43936, 43937, 43938, 43939, 43940, 43941, 43942, 43943, 43944, 43945, 43946,
	43947, 43948, 43949, 43950, 43951, 43952, 43953, 43954, 43955, 43956, 43957, 43958,
	43959, 43960, 43961, 43962, 43963, 43964, 43965, 43966, 43967, 64256, 64257, 64258,
	64259, 64260, 64261, 64262, 64275, 64276, 64277, 64278, 64279,
};
static long FOLD_TO[891] = {
	956, 0, 0, 115, 115, 0, 700, 110, 0, 115, 0, 0,
	106, 780, 0, 953, 0, 0, 953, 776, 769, 965, 776, 769,
	963, 0, 0, 946, 0, 0, 952, 0, 0, 966, 0, 0,
	960, 0, 0, 954, 0, 0, 961, 0, 0, 949, 0, 0,
	1381, 1410, 0, 5024, 0, 0, 5025, 0, 0, 5026, 0, 0,
	5027, 0, 0, 5028, 0, 0, 5029, 0, 0, 5030, 0, 0,
	5031, 0, 0, 5032, 0, 0, 5033, 0, 0, 5034, 0, 0,
	5035, 0, 0, 5036, 0, 0, 5037, 0, 0, 5038, 0, 0,
	5039, 0, 0, 5040, 0, 0, 5041, 0, 0, 5042, 0, 0,
	5043, 0, 0, 5044, 0, 0, 5045, 0, 0, 5046, 0, 0,
	5047, 0, 0, 5048, 0, 0, 5049, 0, 0, 5050, 0, 0,
	5051, 0, 0, 5052, 0, 0, 5053, 0, 0, 5054, 0, 0,
	5055, 0, 0, 5056, 0, 0, 5057, 0, 0, 5058, 0, 0,
	5059, 0, 0, 5060, 0, 0, 5061, 0, 0, 5062, 0, 0,
	5063, 0, 0, 5064, 0, 0, 5065, 0, 0, 5066, 0, 0,
	5067, 0, 0, 5068, 0, 0, 5069, 0, 0, 5070, 0, 0,
	5071, 0, 0, 5072, 0, 0, 5073, 0, 0, 5074, 0, 0,
	5075, 0, 0, 5076, 0, 0, 5077, 0, 0, 5078, 0, 0,
	5079, 0, 0, 5080, 0, 0, 5081, 0, 0, 5082, 0, 0,
	5083, 0, 0, 5084, 0, 0, 5085, 0, 0, 5086, 0, 0,
	5087, 0, 0, 5088, 0, 0, 5089, 0, 0, 5090, 0, 0,
	5091, 0, 0, 5092, 0, 0, 5093, 0, 0, 5094, 0, 0,
	5095, 0, 0, 5096, 0, 0, 5097, 0, 0, 5098, 0, 0,
	5099, 0, 0, 5100, 0, 0, 5101, 0, 0, 5102, 0, 0,
	5103, 0, 0, 5104, 0, 0, 5105, 0, 0, 5106, 0, 0,
	5107, 0, 0, 5108, 0, 0, 5109, 0, 0, 5104, 0, 0,
	5105, 0, 0, 5106, 0, 0, 5107, 0, 0, 5108, 0, 0,
	5109, 0, 0, 1074, 0, 0, 1076, 0, 0, 1086, 0, 0,
	1089, 0, 0, 1090, 0, 0, 1090, 0, 0, 1098, 0, 0,
	1123, 0, 0, 42571, 0, 0, 104, 817, 0, 116, 776, 0,
	119, 778, 0, 121, 778, 0, 97, 702, 0, 7777, 0, 0,
	115, 115, 0, 965, 787, 0, 965, 787, 768, 965, 787, 769,
	965, 787, 834, 7936, 953, 0, 7937, 953, 0, 7938, 953, 0,
	7939, 953, 0, 7940, 953, 0, 7941, 953, 0, 7942, 953, 0,
	7943, 953, 0, 7936, 953, 0, 7937, 953, 0, 7938, 953, 0,
	7939, 953, 0, 7940, 953, 0, 7941, 953, 0, 7942, 953, 0,
	7943, 953, 0, 7968, 953, 0, 7969, 953, 0, 7970, 953, 0,
	7971, 953, 0, 7972, 953, 0, 7973, 953, 0, 7974, 953, 0,
	7975, 953, 0, 7968, 953, 0, 7969, 953, 0, 7970, 953, 0,
	7971, 953, 0, 7972, 953, 0, 7973, 953, 0, 7974, 953, 0,
	7975, 953, 0, 8032, 953, 0, 8033, 953, 0, 8034, 953, 0,
	8035, 953, 0, 8036, 953, 0, 8037, 953, 0, 8038, 953, 0,
	8039, 953, 0, 8032, 953, 0, 8033, 953, 0, 8034, 953, 0,
	8035, 953, 0, 8036, 953, 0, 8037, 953, 0, 8038, 953, 0,
	8039, 953, 0, 8048, 953, 0, 945, 953, 0, 940, 953, 0,
	945, 834, 0, 945, 834, 953, 945, 953, 0, 953, 0, 0,
	8052, 953, 0, 951, 953, 0, 942, 953, 0, 951, 834, 0,
	951, 834, 953, 951, 953, 0, 953, 776, 768, 953, 776, 769,
	953, 834, 0, 953, 776, 834, 965, 776, 768, 965, 776, 769,
	961, 787, 0, 965, 834, 0, 965, 776, 834, 8060, 953, 0,
	969, 953, 0, 974, 953, 0, 969, 834, 0, 969, 834, 953,
	969, 953, 0, 5024, 0, 0, 5025, 0, 0, 5026, 0, 0,
	5027, 0, 0, 5028, 0, 0, 5029, 0, 0, 5030, 0, 0,
	5031, 0, 0, 5032, 0, 0, 5033, 0, 0, 5034, 0, 0,
	5035, 0, 0, 5036, 0, 0, 5037, 0, 0, 5038, 0, 0,
	5039, 0, 0, 5040, 0, 0, 5041, 0, 0, 5042, 0, 0,
	5043, 0, 0, 5044, 0, 0, 5045, 0, 0, 5046, 0, 0,
	5047, 0, 0, 5048, 0, 0, 5049, 0, 0, 5050, 0, 0,
	5051, 0, 0, 5052, 0, 0, 5053, 0, 0, 5054, 0, 0,
	5055, 0, 0, 5056, 0, 0, 5057, 0, 0, 5058, 0, 0,
	5059, 0, 0, 5060, 0, 0, 5061, 0, 0, 5062, 0, 0,
	5063, 0, 0, 5064, 0, 0, 5065, 0, 0, 5066, 0, 0,
	5067, 0, 0, 5068, 0, 0, 5069, 0, 0, 5070, 0, 0,
	5071, 0, 0, 5072, 0, 0, 5073, 0, 0, 5074, 0, 0,
	5075, 0, 0, 5076, 0, 0, 5077, 0, 0, 5078, 0, 0,
	5079, 0, 0, 5080, 0, 0, 5081, 0, 0, 5082, 0, 0,
	5083, 0, 0, 5084, 0, 0, 5085, 0, 0, 5086, 0, 0,
	5087, 0, 0, 5088, 0, 0, 5089, 0, 0, 5090, 0, 0,
	5091, 0, 0, 5092, 0, 0, 5093, 0, 0, 5094, 0, 0,
	5095, 0, 0, 5096, 0, 0, 5097, 0, 0, 5098, 0, 0,
	5099, 0, 0, 5100, 0, 0, 5101, 0, 0, 5102, 0, 0,
	5103, 0, 0, 102, 102, 0, 102, 105, 0, 102, 108, 0,
	102, 102, 105, 102, 102, 108, 115, 116, 0, 115, 116, 0,
	1396, 1398, 0, 1396, 1381, 0, 1396, 1387, 0, 1406, 1398, 0,
	1396, 1389, 0,
};

/* The index of cp in FOLD_CP, or -1. */
long case_fold(long cp) {
	long a = 0;
	long b = NFOLD - 1;
	long m;
	while (a <= b) {
		m = (a + b) / 2;
		if (cp < FOLD_CP[m]) { b = m - 1; }
		else if (cp > FOLD_CP[m]) { a = m + 1; }
		else { return m; }
	}
	return -1;
}

/* The index of cp in TIFULL_CP, or -1. */
long case_tifull(long cp) {
	long a = 0;
	long b = NTIFULL - 1;
	long m;
	while (a <= b) {
		m = (a + b) / 2;
		if (cp < TIFULL_CP[m]) { b = m - 1; }
		else if (cp > TIFULL_CP[m]) { a = m + 1; }
		else { return m; }
	}
	return -1;
}

/* The simple titlecase: the uppercase except for the Latin digraphs (whose
 * titlecase is their middle form) and Georgian Mkhedruli (no titlecase), as
 * Go's unicode.ToTitle. */
long case_title(long cp) {
	if (cp >= 452 && cp <= 460) { return cp - (cp - 452) % 3 + 1; }
	if (cp >= 497 && cp <= 499) { return 498; }
	if ((cp >= 4304 && cp <= 4346) || (cp >= 4349 && cp <= 4351)) { return cp; }
	return case_map(cp, 1);
}

long case_str(long recv, long mode) {
	/* Over CODE POINTS: the mapping is Unicode-wide and can change a
	 * character's length. A full mapping writes at most three BMP units for
	 * one unit, so the buffer is sized for that. */
	long un = str_ulen(recv);
	long *u = buf_new(3 * un + 1);
	long up = mode & 1;
	long ti = mode & 8;
	long k = 0;
	long i = 0;
	long cp;
	long adv;
	long x;
	long t;
	while (i < un) {
		cp = case_cp_at(recv, i, un);
		adv = case_units(cp);
		if (cp < 128) {
			if ((up || ti) && cp >= 97 && cp <= 122) { cp = cp - 32; }
			else if (!up && !ti && cp >= 65 && cp <= 90) { cp = cp + 32; }
			u[k] = cp;
			k = k + 1;
			i = i + 1;
			continue;
		}
		if (cp >= 55296 && cp <= 57343) {
			/* An unpaired surrogate has no case: it passes through, as in node. */
			u[k] = cp;
			k = k + 1;
			i = i + 1;
			continue;
		}
		x = (mode & 16) ? case_fold(cp) : -1;
		if (x >= 0) {
			t = 0;
			while (t < 3 && FOLD_TO[3 * x + t] != 0) {
				u[k] = FOLD_TO[3 * x + t];
				k = k + 1;
				t = t + 1;
			}
			i = i + adv;
			continue;
		}
		x = (ti && (mode & 2)) ? case_tifull(cp) : -1;
		if (x >= 0) {
			t = 0;
			while (t < 3 && TIFULL_TO[3 * x + t] != 0) {
				u[k] = TIFULL_TO[3 * x + t];
				k = k + 1;
				t = t + 1;
			}
			i = i + adv;
			continue;
		}
		x = (up && !ti && (mode & 2)) ? case_upfull(cp) : -1;
		if (x >= 0) {
			t = 0;
			while (t < 3 && UPFULL_TO[3 * x + t] != 0) {
				u[k] = UPFULL_TO[3 * x + t];
				k = k + 1;
				t = t + 1;
			}
			i = i + adv;
			continue;
		}
		if (!up && !ti && cp == 304 && (mode & 2)) {
			u[k] = 105; k = k + 1;
			u[k] = 775; k = k + 1;
			i = i + adv;
			continue;
		}
		if (ti) { cp = case_title(cp); }
		else if (!up && cp == 931 && (mode & 4) && case_final_sigma(recv, i, un)) { cp = 962; }
		else { cp = case_map(cp, up); }
		if (cp > 65535) {
			u[k] = 55296 + ((cp - 65536) >> 10); k = k + 1;
			u[k] = 56320 + ((cp - 65536) & 1023); k = k + 1;
		} else { u[k] = cp; k = k + 1; }
		i = i + adv;
	}
	return str_from_units(u, k);
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
 * Generating digits by repeatedly multiplying a float remainder by ten is right
 * to only about sixteen digits, so it is WRONG IN THE LAST PLACE of every
 * seventeen-digit number (10.0/3.0 is 3.3333333333333335, not ...334). Nothing
 * here touches a double. */

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
 * it. g_mindig = n would only start the SEARCH at k = n, so a value whose
 * shortest form is longer than n would come back with the shortest form
 * (floPrec(1/3, 1) must be "3e-1", as floPrecStr in abnf/commonscript.go - the
 * Go/goja/frozen twin - answers). Exactly-n is the contract a %e / %g formatter
 * needs and the one a script CANNOT build for itself: "at least n" is derivable
 * from exactly-n plus the host's shortest form ("" + v), not the other way round.
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
 * repeated multiplication rounds once per step, which would turn
 * "1.7976931348623157e308" into +Inf and "1e100" one part in 10^16 too large.
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

long digit_value(int c);

/* Does s[i..j) spell the lower-case word w, ignoring ASCII case? */
int word_ci(const char *s, long i, long j, const char *w) {
	long k = 0;
	while (i + k < j && w[k] != 0 && (((int)s[i + k] & 255) | 32) == (int)w[k]) { k = k + 1; }
	return i + k == j && w[k] == 0;
}

/* Go's underscoreOK over s[i..j): an underscore only between two digits, where a
 * base prefix counts as a digit and a hex number's digits include a-f. */
int under_ok(const char *s, long i, long j) {
	long saw = 94;
	long hex = 0;
	if (i < j && (s[i] == 43 || s[i] == 45)) { i = i + 1; }
	if (j - i >= 2 && s[i] == 48 && (((int)s[i + 1] & 255) | 32) == 120) { i = i + 2; saw = 48; hex = 1; }
	while (i < j) {
		long c = (long)s[i] & 255;
		if ((c >= 48 && c <= 57) || (hex != 0 && (c | 32) >= 97 && (c | 32) <= 102)) { saw = 48; }
		else if (c == 95) {
			if (saw != 48) { return 0; }
			saw = 95;
		} else {
			if (saw == 95) { return 0; }
			saw = 33;
		}
		i = i + 1;
	}
	return saw != 95;
}

/* The double nearest m * 2^e2 (0 <= m < 2^61), where sticky says nonzero bits
 * were already dropped below m: nearest, ties to even. */
long bin_to_double(long m, long e2, long sticky, long neg) {
	long e;
	long sh;
	long q;
	long rem;
	long half;
	long sbit = neg ? (0 - 9223372036854775807L - 1L) : 0L;
	if (m == 0) { return sbit; }
	while (m < 1152921504606846976L) { m = m * 2; e2 = e2 - 1; }
	/* m is in [2^60, 2^61), so the leading bit's exponent is e2 + 60. */
	e = e2 + 60;
	if (e > 1023) { return neg ? DNINF : DINF; }
	sh = 8;
	if (e < -1022) { sh = 8 + (-1022 - e); }
	if (sh > 61) { return sbit; }
	q = m >> sh;
	rem = m & ((1L << sh) - 1);
	half = 1L << (sh - 1);
	if (rem > half || (rem == half && (sticky != 0 || (q & 1) != 0))) { q = q + 1; }
	if (e < -1022) { return q | sbit; }
	if (q == 9007199254740992L) { q = q / 2; e = e + 1; }
	if (e > 1023) { return neg ? DNINF : DINF; }
	return (((e + 1023) << 52) | (q - 4503599627370496L)) | sbit;
}

/* Go's hexadecimal float, after the "0x": hex digits with an optional point, then
 * a binary exponent that is NOT optional. Sixty bits of mantissa are kept and the
 * rest only decide the rounding. */
long hex_to_num(const char *s, long i, long j, long neg) {
	long m = 0;
	long e2 = 0;
	long sticky = 0;
	long digits = 0;
	long dot = 0;
	long esign = 1;
	long eval = 0;
	long edig = 0;
	long more = 1;
	while (i < j && more != 0) {
		long c = (long)s[i] & 255;
		long d = digit_value((int)c);
		if (c == 46 && dot == 0) { dot = 1; i = i + 1; }
		else if (c == 95) { i = i + 1; }
		else if (d >= 0 && d < 16) {
			digits = digits + 1;
			if (m < 72057594037927936L) {
				m = m * 16 + d;
				if (dot != 0) { e2 = e2 - 4; }
			} else {
				if (d != 0) { sticky = 1; }
				if (dot == 0) { e2 = e2 + 4; }
			}
			i = i + 1;
		} else { more = 0; }
	}
	if (digits == 0 || i == j || (s[i] != 112 && s[i] != 80)) { return DNAN; }
	i = i + 1;
	if (i < j && s[i] == 43) { i = i + 1; }
	else if (i < j && s[i] == 45) { esign = -1; i = i + 1; }
	while (i < j && ((s[i] >= 48 && s[i] <= 57) || s[i] == 95)) {
		if (s[i] != 95) {
			eval = eval * 10 + (s[i] - 48);
			if (eval > 100000) { eval = 100000; }
			edig = edig + 1;
		}
		i = i + 1;
	}
	if (edig == 0 || i != j) { return DNAN; }
	return bin_to_double(m, e2 + esign * eval, sticky, neg);
}

/* str_to_num is Go's strconv.ParseFloat on the TRIMMED string: an empty string
 * is 0 and anything that does not parse whole is NaN. That takes in "inf" and
 * "infinity" in any case and with a sign, "nan" in any case without one, a
 * hexadecimal float with its p exponent ("0x10" alone is NaN), and underscores
 * between digits; out of range is +-Inf rather than an error. Twin of
 * rt.toNumber in abnf/jsrt.go. */
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
	long k;
	long us = 0;
	union DB r;
	union DB p;
	while (i < j && is_space((int)s[i] & 255)) { i = i + 1; }
	while (j > i && is_space((int)s[j - 1] & 255)) { j = j - 1; }
	if (i == j) { return DZERO; }
	k = i;
	if (s[i] == 43) { i = i + 1; } else if (s[i] == 45) { neg = 1; i = i + 1; }
	if (i == k && word_ci(s, i, j, "nan")) { return DNAN; }
	if (word_ci(s, i, j, "inf") || word_ci(s, i, j, "infinity")) { return neg ? DNINF : DINF; }
	if (j - i > 2 && s[i] == 48 && (((int)s[i + 1] & 255) | 32) == 120) {
		if (!under_ok(s, k, j)) { return DNAN; }
		return hex_to_num(s, i + 2, j, neg);
	}
	/* The significant digits are collected WHOLE, most significant first, so the
	 * conversion below can be exact; exp10 is kept so that the value is exactly
	 * those digits read as an integer, times 10^exp10. */
	while (i < j && ((s[i] >= 48 && s[i] <= 57) || s[i] == 95)) {
		long dv = s[i] - 48;
		if (s[i] == 95) { us = 1; }
		else {
			if (mdig == 0 && dv == 0) { }               /* a leading zero is nothing */
			else if (mdig < 800) { G_DIG[mdig] = (char)dv; mdig = mdig + 1; }
			else { exp10 = exp10 + 1; }
			if (mdig < 16) { mant = mant * 10 + dv; }
			digits = digits + 1;
		}
		i = i + 1;
	}
	if (i < j && s[i] == 46) {
		i = i + 1;
		while (i < j && ((s[i] >= 48 && s[i] <= 57) || s[i] == 95)) {
			long dv = s[i] - 48;
			if (s[i] == 95) { us = 1; }
			else {
				if (mdig == 0 && dv == 0) { exp10 = exp10 - 1; }
				else if (mdig < 800) { G_DIG[mdig] = (char)dv; mdig = mdig + 1; exp10 = exp10 - 1; }
				if (mdig < 16) { mant = mant * 10 + dv; }
				digits = digits + 1;
			}
			i = i + 1;
		}
	}
	if (digits == 0) { return DNAN; }
	if (i < j && (s[i] == 101 || s[i] == 69)) {
		long edig = 0;
		i = i + 1;
		if (i < j && s[i] == 43) { i = i + 1; }
		else if (i < j && s[i] == 45) { esign = -1; i = i + 1; }
		while (i < j && ((s[i] >= 48 && s[i] <= 57) || s[i] == 95)) {
			if (s[i] == 95) { us = 1; }
			else {
				eval = eval * 10 + (s[i] - 48);
				if (eval > 100000) { eval = 100000; }
				edig = edig + 1;
			}
			i = i + 1;
		}
		if (edig == 0) { return DNAN; }
		exp10 = exp10 + esign * eval;
	}
	if (i != j) { return DNAN; }
	if (us != 0 && !under_ok(s, k, j)) { return DNAN; }
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
long fn_text(long h);
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
	if (t == 7 || t == 8 || t == 9 || t == 16) {   /* tag 16 too: the Go twin's js_genfn result is a *hostFunc */
		if (FN_SRC_TAB != 0) { return fn_text(h); }
		return mk_cstr("[function]");
	}
	if (t == 11) { return mk_cstr("[scope]"); }
	return mk_cstr("[value]");
}

/* The six answers of type_of, made once by boot(): `typeof` is on the hot path
 * of every layer 2, and mk_cstr measures and looks up its literal each time. */
long TY_UNDEF;
long TY_BOOL;
long TY_NUM;
long TY_STR;
long TY_FN;
long TY_OBJ;
long type_of(long h) {
	long t = tag_of(h);
	if (t == 0) { return TY_UNDEF; }
	if (t == 2) { return TY_BOOL; }
	/* A SIZED INTEGER IS A NUMBER. This is the whole point of the tag: the Go
	 * twin answers "number" for jsGInt (abnf/jsrt.go typeOf), and an object box
	 * in layer 2 could only ever have answered "object". */
	if (t == 3 || t == 13 || t == 14) { return TY_NUM; }
	if (t == 4) { return TY_STR; }
	/* Tag 16 is a GENERATOR FUNCTION and it is callable: is_callable() below
	 * answers true for it, the Go twin's js_genfn returns a *hostFunc whose
	 * typeOf is "function" (abnf/jsrt.go), and a caller that asks
	 * `typeof m == "function"` before calling a member must get the same
	 * answer in both halves. */
	if (t == 7 || t == 8 || t == 9 || t == 16) { return TY_FN; }
	return TY_OBJ;
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
 * machine (darwin/arm64) it answers 0; on amd64 the same expression answers
 * -9223372036854775808. 0 is what is implemented here, so the two halves agree
 * on arm64 and both would differ from a Go twin built for amd64. */
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

/* An integer divide by zero dies, except in Go, where it is a run-time PANIC:
 * deferred calls still run and js_gotop reports it with go's wording and exit
 * status. go-to-llvm-ir.abnf sets the flag (js_gidivthrow) in a program whose
 * divisions carry no js_gidivz pre-check, which is exactly a program that never
 * calls recover() - so nothing can observe that the thrown value is go's text
 * rather than a runtime.Error. */
long js_throw(long v);
long SI_DIVZ_THROWS = 0;
long js_gidivthrow(void) { SI_DIVZ_THROWS = 1; return H_UNDEF; }
void si_divzero(void) {
	if (SI_DIVZ_THROWS) { js_throw(mk_cstr("runtime error: integer divide by zero")); }
	die("integer divide by zero");
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
	 * The override on the line below is UNCONDITIONAL, so the shift arms compute
	 * at the left operand's own width and signedness whatever type the count
	 * arrived with. In practice every language's layer 2 already hands the count
	 * over as a plain number or a 64-bit signed box, so this is a backstop.
	 * Twins: giArith in abnf/jsrtint.go, szArithSlow in lib/interp-core.js, siOp
	 * in metajs-interpreter.abnf. */
	long a = si_val(l);
	long b = si_val(r);
	long x = 0;
	long s;
	if (code == 9 || code == 10) { w = si_width_of(l, l); u = si_uns_of(l, l); }
	if (code == 0) { x = a + b; }
	else if (code == 1) { x = a - b; }
	else if (code == 2) { x = a * b; }
	else if (code == 3) {
		if (b == 0) { si_divzero(); }
		if (u) { x = (long)(si_u(a, w) / si_u(b, w)); }
		else if (a == (0 - 9223372036854775807 - 1) && b == -1) { x = a; }
		else { x = a / b; }
	}
	else if (code == 4) {
		if (b == 0) { si_divzero(); }
		if (u) { x = (long)(si_u(a, w) % si_u(b, w)); }
		else if (a == (0 - 9223372036854775807 - 1) && b == -1) { x = 0; }
		else { x = a % b; }
	}
	else if (code == 5) { x = a & b; }
	else if (code == 6) { x = a | b; }
	else if (code == 7) { x = a ^ b; }
	else if (code == 8) { x = a & (0 - b - 1); }   /* &^, i.e. a & ~b */
	else if (code == 9) {
		/* Go's rule, and unlike C not undefined: a count at or above the width
		 * shifts everything out. A negative count does too. */
		s = b;
		if (s < 0 || s >= w) { x = 0; } else { x = a << s; }
	}
	else if (code == 10) {
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
 * Testing the decimal exponent instead (e10 >= -3 && e10 < 7) gives the same
 * answers - the shortest form of a double at or above 1e-3 always has
 * e10 >= -3 - but the value test is what the Go source says. */
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
	 * renders "5.0E-324" (java 24.0.2). Every NORMAL double is within about
	 * 1e-16 of its one digit form, a thousandth of a two digit step, so the two
	 * rules agree everywhere else. */
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
 * result is NaN"). Both comparisons are FALSE, so a bare `<` would take the
 * right operand and answer 0.0 and 1.0. Everything else is the ordinary
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
	/* A SIGNED 64 bit box is negated IN THE BOX: the fallthrough's to_int32
	 * would truncate a long. Two's-complement negation maps
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
 * NOT A FATTER SLOT: carrying the name handle beside the index would make a hit
 * one memory access instead of two (the slot, then names[idx]), but it doubles
 * the table and measured slower overall. */
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

/* A scope starts EMPTY - capacity 0, no buffers - and scope_put allocates
 * the first time a name is declared in it (growing 4, 8, 16, ...).
 *
 * lib/compile-core.js's makeBlockStmt opens a scope for every BLOCK, not only
 * for every call, and most block scopes never declare a single name. An empty
 * scope costs one 56-byte cell and one arena call.
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
 * sixteen languages ends here. A module scope has hundreds of entries (461 for a
 * kotlin program), so a scope of n >= 32 is searched through its hash table;
 * a linear scan would also make the POSITION of a declaration matter.
 *
 * Both arms give the same answer: the table indexes the same array and the
 * winner is the FIRST matching entry, because scope_put never inserts a
 * duplicate (it assigns instead) and nothing ever removes an entry, so at most
 * one index in the table can match.
 *
 * The arm is chosen on n, which the linear scan needs anyway, and NOT on the
 * capacity: reading fe(s) here as well is another load on the path every small
 * scope takes, and it is measurable. n <= cap and cap is a
 * power of two, so n >= 32 guarantees cap >= 32 and therefore that scope_hwords
 * gave this scope a table.
 *
 * THE HASH ARM IS OUT OF LINE, in scope_find_hashed below, on purpose: written
 * inline it makes this function too big for clang to keep inlining into
 * scope_get, scope_put and js_tdecl, and then every SMALL scope pays for an arm
 * it never takes (measurable on lua, whose benchmark is dominated by small
 * scopes). Keep this function small. */
long scope_find_hashed(long *names, long cap, long name) {
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
long scope_find(long s, long name) {
	long *names = (long *)fa(s);
	long n = fd(s);
	long i = 0;
	if (n >= 32) { return scope_find_hashed(names, fe(s), name); }
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

long js_throw(long v);

/* Hand `name` (and, for the member form, the object) to the language's hook and
 * throw what it answers. Clearing the slot for the duration is the recursion
 * guard described where RT_MISS_VAR is declared. Answering 0 means "abort after
 * all", so a hook can decline one case and keep the diagnostic. */
long rt_miss_var(long name) {
	long h = RT_MISS_VAR;
	long args;
	long v;
	if (h == 0) { return 0; }
	RT_MISS_VAR = 0;
	args = mk_arr();
	arr_push(args, name);
	v = js_call(h, H_UNDEF, args);
	RT_MISS_VAR = h;
	return v;
}

long rt_miss_mem(long key, long obj) {
	long h = RT_MISS_MEM;
	long args;
	long v;
	if (h == 0) { return 0; }
	RT_MISS_MEM = 0;
	args = mk_arr();
	arr_push(args, key);
	arr_push(args, obj);
	v = js_call(h, H_UNDEF, args);
	RT_MISS_MEM = h;
	return v;
}

/* The WRITE sibling. It is a hook of its own rather than a flag on the read one
 * because the two messages differ: node says "Cannot read properties of null
 * (reading 'x')" and "Cannot set properties of null (setting 'x')". */
long rt_miss_set(long key, long obj) {
	long h = RT_MISS_SET;
	long args;
	long v;
	if (h == 0) { return 0; }
	RT_MISS_SET = 0;
	args = mk_arr();
	arr_push(args, key);
	arr_push(args, obj);
	v = js_call(h, H_UNDEF, args);
	RT_MISS_SET = h;
	return v;
}

/* The CALL sibling: a call of a value that is no function. Unlike the others the
 * closure takes the callee AND the argument array and its answer IS the call's
 * result: lua calls a table through its __call there and raises "attempt to call
 * a nil value" itself. It stays registered while it runs, because the __call it
 * makes runs program code. */
long rt_miss_fn(long callee, long args) {
	long a = mk_arr();
	arr_push(a, callee);
	arr_push(a, args);
	return js_call(RT_MISS_FN, H_UNDEF, a);
}

long scope_get(long s, long name) {
	long cur = s;
	long exc;
	while (cur != 0) {
		long i = scope_find(cur, name);
		long *vals;
		if (i >= 0) { vals = (long *)fb(cur); return vals[i]; }
		cur = ff(cur);
	}
	exc = rt_miss_var(name);
	if (exc != 0) { js_throw(exc); }
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
 *   1..10  array   push pop shift unshift reverse slice indexOf join concat sort
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
	if (str_eq_c(name, "sort"))    { return 10; }
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

/* The array (or string) index a member key names, or -1 for a property name:
 * a number that is a non-negative integer, or a string spelling one
 * canonically below 2^32 - 1 (the ECMAScript array index, so "01", "1.0",
 * "+1", " 1", "-0" and "" are names). Twin of rt.arrIndexOf in abnf/jsrt.go. */
long index_of_key(long key) {
	long n;
	if (tag_of(key) == 4) {
		long len = str_len(key);
		const char *p = str_ptr(key);
		long v = 0;
		long i = 0;
		if (len == 0 || len > 10 || (p[0] == 48 && len > 1)) { return -1; }
		while (i < len) {
			int c = (int)p[i] & 255;
			if (c < 48 || c > 57) { return -1; }
			v = v * 10 + (c - 48);
			i = i + 1;
		}
		if (v >= 4294967295) { return -1; }
		return v;
	}
	n = to_number(key);
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
	long exc;
	if (t == 0 || t == 1) {
		exc = rt_miss_mem(key, obj);
		if (exc != 0) { js_throw(exc); }
		die3("member '", key, "' of ", obj);
	}
	if (tag_of(key) == 4 && is_callable(obj)) {
		if (str_eq_c(key, "apply")) { return mk_bound(obj, 40); }
		if (str_eq_c(key, "call"))  { return mk_bound(obj, 41); }
	}
	if (t == 6) { return obj_get(obj, to_string(key)); }
	/* A generator's iterator protocol. The floor owns `next` only; current /
	 * key / valid / send / getReturn are layer 2 over it. */
	if (t == 15 && tag_of(key) == 4 && str_eq_c(key, "next")) { return mk_bound(obj, 60); }
	/* g.return(v): the CLOSING half of the iterator protocol. It is answered
	 * here and not left to layer 2 because a generator is a floor cell - nothing
	 * can be stored on one - and a layer-2 closed-ness side table would not be
	 * consulted by the native for-loop. See gen_close. */
	if (t == 15 && tag_of(key) == 4 && str_eq_c(key, "return")) { return mk_bound(obj, 61); }
	/* g.throw(v): raise v AT the yield the body is parked at, so the body's own
	 * try/catch and finally clauses see it. Like `return` it belongs on the floor
	 * and not in layer 2, because the record travels on the RESUME VALUE - the
	 * cell's f slot, which only the yield this generator is parked at ever reads.
	 * That is what makes it per-coroutine by construction rather than by care
	 * (a per-PROGRAM depth stack does not work). See gen_throw. */
	if (t == 15 && tag_of(key) == 4 && str_eq_c(key, "throw")) { return mk_bound(obj, 62); }
	if (t == 5) {
		if (tag_of(key) == 4) {
			long mid;
			if (str_eq_c(key, "length")) { return mk_num(d_from_long(arr_len(obj))); }
			mid = method_id_array(key);
			if (mid != 0) { return mk_bound(obj, mid); }
		}
		idx = index_of_key(key);
		if (idx >= 0) { return arr_get(obj, idx); }
		/* A key that is no index names a property in field e (set_member). */
		if (fe(obj) != H_UNDEF) { return obj_get(fe(obj), to_string(key)); }
		return H_UNDEF;
	}
	if (t == 4) {
		if (tag_of(key) == 4) {
			long mid;
			if (str_eq_c(key, "length")) { return mk_num(d_from_long(str_ulen(obj))); }
			mid = method_id_string(key);
			if (mid != 0) { return mk_bound(obj, mid); }
		}
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
	long exc;
	if (t == 0 || t == 1) {
		exc = rt_miss_set(key, obj);
		if (exc != 0) { js_throw(exc); }
		die3("member assignment '", key, "' on ", obj);
	}
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
		idx = index_of_key(key);
		/* A key that is no index is a NAMED property: it lives in an object
		 * kept in field e (arrProps), made at the first such write. */
		if (idx < 0) {
			if (fe(obj) == H_UNDEF) { se(obj, mk_obj()); }
			obj_put(fe(obj), to_string(key), val);
			return;
		}
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
long fmt_top(long h) {
	if (tag_of(h) == 4) { return RT_RAW_STR != 0 ? h : wtf8_clean(h); }
	return fmt_val(h);
}

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

/* One jmp_buf per DEPTH, allocated once and reused: this file never frees a
 * malloc, so a fresh buffer per js_try would leak in every loop with a try.
 * A buffer at depth d is only live while JB_DEPTH > d, and js_try restores
 * JB_DEPTH before it returns, so two try statements at the same depth are
 * sequential and can share the buffer. THE SETJMP MUST STILL BE DONE ONCE PER
 * ENTRY - only the storage is reused.
 *
 * THE POOL IS PER STACK, not per process. A generator body runs on its own C
 * stack, so a setjmp it takes is only valid on that stack and a longjmp to a
 * buffer the RESUMER filled in would jump into another thread's frame - the
 * undefined behaviour. JBP therefore points at the pool of the stack that is RUNNING:
 * main's for main, and cb[9] for a coroutine, swapped by whichever side gains
 * control (exactly as GC_STACK_BASE is). JB_CAP is that pool's length. */
long JBP;            /* long *: the running stack's jmp_buf pool */
long JB_CAP;         /* its length in slots */
long JB_MAIN;        /* main's own pool, so the swap has something to go back to */
long JB_DEPTH;

/* The pool doubles when a try nests past its length (every go function body is
 * one, so a recursion is as deep as its pool). The buffers themselves do not
 * move, only the array of pointers to them, and the running stack's owner -
 * main's JB_MAIN or the running generator's cb[9]/cb[12] - takes the new one. */
long coro_grow(long arr, long cap, long newcap);
long jb_at(long d) {
	long *p;
	if (d >= JB_CAP) {
		long ncap = JB_CAP * 2;
		long np = coro_grow(JBP, JB_CAP, ncap);
		if (JBP == JB_MAIN) { JB_MAIN = np; }
		else { long *cb = (long *)fc(CUR_GEN); cb[9] = np; cb[12] = ncap; }
		JBP = np;
		JB_CAP = ncap;
	}
	p = (long *)JBP;
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
/* An argument as a whole number: truncated, a NaN 0 (the Go twin's jsToInt). */
long arg_idx(long args, long i) { long v = arg_num(args, i); return d_is_nan(v) ? 0 : d_to_long(v); }
long arg_str(long args, long i) { if (i < arr_len(args)) { return to_string(arr_get(args, i)); } return mk_cstr(""); }

long host_call(long id, long self, long args);
long builtin_method(long recv, long mid, long args);
long js_jsfnmeta(long f, long nm, long ln);
long fn_meta_get(long f, long key);
long fn_src_idx(long f);
int fn_kind(long h);

long gen_create(long gf, long args);

/* ---- THE CALL DEPTH ------------------------------------------------------
 * CALL_DEPTH counts the calls of the PROGRAM's own closures that are running:
 * a closure of the emitted module, whose function index is below the 1,000,000
 * a -rt-lib library is numbered from, so layer 2's own calls are not counted
 * and the count is the one the Go twin keeps (its runtime is Go, not closures).
 * A try clause is run uncounted (call_clause), so a frame is a function.
 *
 * Past CALL_MAX the language's js_rt_deep hook decides: a closure answering
 * the value to throw (python's RecursionError, java's StackOverflowError), or,
 * with a non-zero status, a message printed on stderr and an exit with that
 * status (go, swift and csharp end the process). The counter is restored by
 * every catch (js_try, since a throw skips the decrements) and around every
 * generator resume. The twin is abnf/jsrt.go's deepN. */
long CALL_DEPTH;
long CALL_MAX;
long RT_DEEP;
long RT_DEEP_EXIT;
long RT_DEEP_BUSY;   /* 1 while the hook runs */

void call_too_deep(void) {
	long max = CALL_MAX;
	long v;
	if (RT_DEEP_EXIT != 0) {
		fflush(0);
		OUTFD = 2;
		o_str(to_string(RT_DEEP));
		exit((int)RT_DEEP_EXIT);
	}
	/* Room for the hook and the constructor of the error it answers; a hook
	 * that recurses past even that aborts. */
	if (RT_DEEP_BUSY != 0) { die("stack overflow inside the stack overflow hook"); }
	RT_DEEP_BUSY = 1;
	CALL_MAX = max + 64;
	v = js_call(RT_DEEP, H_UNDEF, mk_arr());
	CALL_MAX = max;
	RT_DEEP_BUSY = 0;
	js_throw(v);
}

long js_rt_deep(long limit, long hook, long status) {
	CALL_MAX = d_to_long(to_number(limit));
	RT_DEEP = hook;
	RT_DEEP_EXIT = d_to_long(to_number(status));
	return 0;
}

/* A call the emitter makes directly rather than through js_call (MetaJS's
 * direct calls), counted around the call; a throw skips js_deep_out and js_try
 * restores the count. Twins in abnf/jsrt.go. */
long js_deep_in(void) {
	CALL_DEPTH = CALL_DEPTH + 1;
	if (CALL_DEPTH > CALL_MAX) { call_too_deep(); }
	return 0;
}
long js_deep_out(void) {
	CALL_DEPTH = CALL_DEPTH - 1;
	return 0;
}

long js_call(long callee, long self, long args) {
	long t = tag_of(callee);
	if (t == 7) {
		long r;
		if (fa(callee) >= 1000000) { return jsdispatch(fa(callee), fb(callee), args); }
		CALL_DEPTH = CALL_DEPTH + 1;
		if (CALL_DEPTH > CALL_MAX) { call_too_deep(); }
		r = jsdispatch(fa(callee), fb(callee), args);
		CALL_DEPTH = CALL_DEPTH - 1;
		return r;
	}
	if (t == 8) { return host_call(fa(callee), self, args); }
	if (t == 9) { return builtin_method(fa(callee), fb(callee), args); }
	/* A generator FUNCTION: calling it runs NOTHING and answers a generator. */
	if (t == 16) { return gen_create(callee, args); }
	if (RT_MISS_FN != 0) { return rt_miss_fn(callee, args); }
	die2("call of a non function value: ", callee);
	return H_UNDEF;
}

/* js_call0..js_call3(callee, arguments...): a call with no receiver and that
 * many arguments, which metajs-to-llvm-ir emits instead of building the array
 * itself. A host builtin only ever READS its argument array (host_call keeps
 * neither the array nor its buffer), so it gets one on the C stack, where the
 * collector's stack scan sees the three handles; any other callee may keep its
 * array (`arguments`, a generator) and gets a heap one. Twins: the same four
 * externs in abnf/jsrt.go. */
long js_call_k(long callee, long n, long a0, long a1, long a2) {
	long args;
	if (tag_of(callee) == 8) {
		long buf[3];
		long cell[7];
		buf[0] = a0; buf[1] = a1; buf[2] = a2;
		cell[0] = 5; cell[1] = (long)buf; cell[2] = n; cell[3] = 3;
		cell[4] = 0; cell[5] = 0; cell[6] = 0;
		return host_call(fa(callee), H_UNDEF, (long)cell);
	}
	args = mk_arr();
	if (n > 0) { arr_push(args, a0); }
	if (n > 1) { arr_push(args, a1); }
	if (n > 2) { arr_push(args, a2); }
	return js_call(callee, H_UNDEF, args);
}
long js_call0(long f) { return js_call_k(f, 0, 0, 0, 0); }
long js_call1(long f, long a) { return js_call_k(f, 1, a, 0, 0); }
long js_call2(long f, long a, long b) { return js_call_k(f, 2, a, b, 0); }
long js_call3(long f, long a, long b, long c) { return js_call_k(f, 3, a, b, c); }

/* ------------------------------------------------------------- coroutines - */
/*
 * TAG 15 is a GENERATOR and TAG 16 a generator FUNCTION, the floor's twins of
 * abnf/jsrt.go's jsGenerator and the closure js_genfn wraps. The Go half runs
 * the body on a goroutine and hands over on two unbuffered channels; this half
 * runs it on a STACK OF ITS OWN and hands over by switching stacks
 * (lib/coro-switch.S). Both are STRICTLY ALTERNATING - at any moment exactly one
 * side runs - so neither needs a lock around the runtime's own state.
 *
 * makecontext/swapcontext are not used: ucontext_t's uc_stack and uc_link have
 * to be written by OFFSET, the offsets differ per platform, and runtime.c has no
 * headers. The switch is a few instructions per architecture instead, and a
 * resume costs a function call rather than a thread handoff.
 *
 * THE CONTROL BLOCK is a long[16] rather than a struct, so no new type crosses
 * the C subset, and it is malloc'd rather than allocated on the heap this file
 * collects - the collector must be able to read it while sweeping. It holds NO
 * handle except cb[7]; every value that travels between the two sides lives in
 * the generator CELL, which is traced.
 *
 *   cb[1] state: 0 not started, 1 SUSPENDED (its stack is a root), 2 done,
 *                3 running
 *   cb[2] the coroutine's saved stack pointer while the resumer runs
 *   cb[3] the resumer's saved stack pointer while the coroutine runs
 *   cb[4] the suspended stack's low water mark
 *   cb[5] the coroutine stack's top (its base, for the collector)
 *   cb[6] a jmp_buf buffer: the callee-saved registers it parked with
 *   cb[7] the generator cell
 *   cb[8] the coroutine stack's valloc'd block, freed once the body is done
 *   cb[9] its OWN jmp_buf pool (a setjmp is only valid on the stack that took
 *         it), cb[10] the depth it parked at, cb[12] the pool's length
 *   cb[11] 1 if the body left by a THROW rather than a return
 *   cb[13] 1 if this resume is a CLOSE: js_yield raises GEN_EXIT (gen_close)
 *   cb[14] 1 if this resume is a THROW-IN: js_yield raises the cell's f slot
 *          (gen_throw). Both live in the CONTROL BLOCK, one per generator, which
 *          is what makes them per-coroutine rather than per-program.
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

/* A body's stack is 2 MB: room for the call limit a language registers with
 * js_rt_deep (10,000 lua frames take about 1 MB), since a recursion inside a
 * generator counts toward the same limit as one outside. */
long coro_alloc(long g) {
	long *cb = (long *)malloc(8 * 16);
	long i = 0;
	while (i < 16) { cb[i] = 0; i = i + 1; }
	cb[8] = (long)valloc(2097152);
	if (cb[8] == 0) { die("coroutines: out of memory for a generator's stack"); }
	mprotect((void *)cb[8], 16384, 0);
	cb[5] = cb[8] + 2097152;
	cb[6] = (long)malloc(512);
	i = 0;
	while (i < 64) { long *jb = (long *)cb[6]; jb[i] = 0; i = i + 1; }
	cb[9] = (long)malloc(8 * 128);
	cb[12] = 128;
	i = 0;
	while (i < 128) { long *jp = (long *)cb[9]; jp[i] = 0; i = i + 1; }
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

/* The body's entry, called by mec_coro_boot on the coroutine's own stack. It
 * never returns: there is no frame under it, so a finished body switches back to
 * its resumer for the last time instead. */
void coro_entry(long arg) {
	long *cb = (long *)arg;
	long g;
	long ret;
	long barrier;
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
	 * setjmp'd - i.e. into a frame on another stack, which only appears to work
	 * while the resumer stays parked and produces a WRONG ANSWER as soon as a
	 * collection moves the timing. The body therefore
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
	mec_coro_switch(&cb[2], cb[3]);
}

/* A resume of a generator whose body is RUNNING (cb[1] == 3) - the body stepping
 * itself, directly or through another generator - would switch to the stack it
 * is already on. It raises what the language's RT_GEN_BUSY hook answers
 * (CPython's ValueError) instead, or aborts, and the close or throw-in that
 * asked for the resume is dropped. The twin is abnf/jsrt.go's jsGenerator.enter. */
void gen_busy(long *cb) {
	cb[13] = 0;
	cb[14] = 0;
	if (RT_GEN_BUSY == 0) { die("generator already running"); }
	js_throw(js_call(RT_GEN_BUSY, H_UNDEF, mk_arr()));
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
	long save_calls;
	long threw;
	if (fe(g) != 0) { return; }
	if (fc(g) == 0) {
		long blk = coro_alloc(g);
		if (CORO_N >= CORO_CAP) {
			long nc = CORO_CAP == 0 ? 64 : CORO_CAP * 2;
			CORO = coro_grow(CORO, CORO_CAP, nc);
			CORO_CAP = nc;
		}
		sc(g, blk);
		{ long *cr = (long *)CORO; cr[CORO_N] = blk; }
		CORO_N = CORO_N + 1;
	}
	cb = (long *)fc(g);
	if (cb[1] == 3) { gen_busy(cb); }
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
	/* The body counts its calls on top of the resumer's, and whatever it left
	 * running when it parked is not the resumer's. */
	save_calls = CALL_DEPTH;
	if (cb[1] == 0) { mec_coro_boot(&cb[3], cb[5], (long)cb); }
	else { mec_coro_switch(&cb[3], cb[2]); }
	RES_N = RES_N - 1;
	/* A finished body never runs again, so its stack goes back to the C heap -
	 * readable again first, since free may touch the guard page's neighbour. */
	if (cb[1] == 2 && cb[8] != 0) {
		mprotect((void *)cb[8], 16384, 3);
		free((void *)cb[8]);
		cb[8] = 0;
	}
	{ long *hi = (long *)RES_HI; GC_STACK_BASE = hi[RES_N]; }
	CUR_GEN = save_gen;
	JBP = save_jbp;
	JB_CAP = save_cap;
	JB_DEPTH = save_depth;
	CALL_DEPTH = save_calls;
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
	mec_coro_switch(&cb[2], cb[3]);
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
	/* gen_throw set cb[14] before resuming: this yield does not answer either,
	 * it raises the value the resumer put in the cell's f slot - an ORDINARY
	 * throw, so the body's own catch arms may take it, unlike GEN_EXIT. */
	if (cb[14] != 0) {
		cb[14] = 0;
		js_throw(ff(g));
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

/* g.throw(v): raise v at the yield the body is parked at. The twin of
 * abnf/jsrt.go's (*jsGenerator).throwInto.
 *
 * Three shapes, and both node and CPython 3.14 agree on all three:
 *   - SUSPENDED: the value is raised inside the body. A body that catches it and
 *     yields again answers {value, false}; one that catches and returns answers
 *     {value: ret, true}; one that does not catch propagates out of throw()
 *     itself, which gen_resume already does for any body that leaves by a throw.
 *   - NOT STARTED: nothing runs - there is no suspension point and therefore no
 *     `finally` to unwind - the generator is marked done and v propagates.
 *   - ALREADY DONE: v propagates and nothing else changes; the return value the
 *     cell still holds is left alone, since getReturn reads it. */
long gen_throw(long g, long args) {
	long v = arg_at(args, 0);
	long *cb;
	long val;
	long done;
	long res;
	if (fe(g) != 0) { js_throw(v); return H_UNDEF; }
	if (fc(g) == 0) {
		se(g, 1);
		sd(g, H_UNDEF);
		sf(g, H_UNDEF);
		js_throw(v);
		return H_UNDEF;
	}
	cb = (long *)fc(g);
	cb[14] = 1;
	gen_resume(g, v, 0);
	if (fe(g) != 0) { val = ff(g); done = 1; } else { val = fd(g); done = 0; }
	res = mk_obj();
	obj_put(res, mk_cstr("value"), val);
	obj_put(res, mk_cstr("done"), done != 0 ? H_TRUE : H_FALSE);
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

/* Array.prototype.sort (ECMA-262 23.1.3.30): a STABLE top-down merge sort, in
 * place. undefined sorts last and never reaches the comparator; without one the
 * elements compare as strings, as `<` compares two strings. Twin of arraySort in
 * abnf/jsrt.go, which runs the same merges so a comparator with side effects sees
 * the same calls in both engines. */
int sort_before(long cmp, long a, long b) {
	long pair;
	if (a == H_UNDEF) { return 0; }
	if (b == H_UNDEF) { return 1; }
	if (cmp == H_UNDEF) { return js_compare(to_string(a), to_string(b)) == -1; }
	pair = mk_arr();
	arr_push(pair, a);
	arr_push(pair, b);
	return d_lt(to_number(js_call(cmp, H_UNDEF, pair)), d_from_long(0));
}
void sort_merge(long cmp, long a, long tmp, long lo, long hi) {
	long mid;
	long i;
	long j;
	long k;
	if (hi - lo < 2) { return; }
	mid = lo + (hi - lo) / 2;
	sort_merge(cmp, a, tmp, lo, mid);
	sort_merge(cmp, a, tmp, mid, hi);
	i = lo; j = mid; k = lo;
	while (i < mid && j < hi) {
		if (sort_before(cmp, arr_get(a, j), arr_get(a, i))) { arr_set(tmp, k, arr_get(a, j)); j = j + 1; }
		else { arr_set(tmp, k, arr_get(a, i)); i = i + 1; }
		k = k + 1;
	}
	while (i < mid) { arr_set(tmp, k, arr_get(a, i)); i = i + 1; k = k + 1; }
	while (j < hi) { arr_set(tmp, k, arr_get(a, j)); j = j + 1; k = k + 1; }
	k = lo;
	while (k < hi) { arr_set(a, k, arr_get(tmp, k)); k = k + 1; }
}

long builtin_method(long recv, long mid, long args) {
	long t = tag_of(recv);
	if (mid == 60) { return gen_next(recv, args); }   /* generator.next(v) */
	if (mid == 61) { return gen_return(recv, args); } /* generator.return(v) */
	if (mid == 62) { return gen_throw(recv, args); }  /* generator.throw(v) */
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
			if (arr_len(args) > 1) {          /* relative from-index, like slice */
				long b = arg_num(args, 1);
				i = clamp_index(d_is_nan(b) ? 0 : d_to_long(d_trunc(b)), n);
			}
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
		if (mid == 10) {                                      /* sort, in place */
			long tmp = mk_arr();
			long i = 0;
			while (i < arr_len(recv)) { arr_push(tmp, H_UNDEF); i = i + 1; }
			sort_merge(arg_at(args, 0), recv, tmp, 0, arr_len(recv));
			return recv;
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
			return u_slice(recv, i, i + 1);
		}
		if (mid == 22) {                                      /* indexOf, in UNITS */
			long from = arr_len(args) > 1 ? clamp_sub(arg_num(args, 1), n) : 0;
			long hay = from > 0 ? u_slice(recv, from, n) : recv;
			long at = str_index_of(hay, arg_str(args, 0));
			if (at < 0) { return mk_num(d_from_long(-1)); }
			if (at > 0) { at = str_ulen(str_slice(hay, 0, at)); }
			return mk_num(d_from_long(from + at));
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
		if (mid == 27) { return case_str(recv, 3); }          /* toUpperCase */
		if (mid == 28) { return case_str(recv, 6); }          /* toLowerCase */
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
 *      30 String.fromCharCode  31 Array.isArray  32 byteLen  97 unicodeCase
 *      101 byteAt  102 byteSub  103 bytesStr
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
	 * Newton converges in half a dozen steps. Starting from the VALUE ITSELF would
	 * need one step per power of two. */
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

/* ----- exp, log, pow -----
 *
 * A FAITHFUL PORT of Go's own math.Exp, math.Log, math.Frexp, math.Ldexp and
 * math.Modf - the same constants, the same polynomials, the same order of
 * operations. The emitted arithmetic is correctly rounded IEEE-754, so the same
 * sequence of operations gives the same bits as the Go twin. Any other
 * approximation of exp/log would disagree with it in the last bits, and "close
 * enough" is not an answer here. pow is not Go's: d_pow rounds correctly, as
 * crPow in abnf/jsrtpow.go does. */

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
	 * would lose the sign (`Math.pow(-0.5, 2147483647)` is -0 in node and in
	 * the Go twin). The infinity below is
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
	 * -0.0 - where `0.0 - 0.0` is +0.0 and loses it (Java prints "-0.0"). */
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
/* A correctly rounded pow for a positive finite x and a finite non-zero y: the
 * double nearest x^y, ties to even. Twin of crPow in abnf/jsrtpow.go, which
 * explains the method; the arithmetic here is the same operation for operation.
 * Double-double values travel as o[0] + o[1]. */
void dd_two_sum(double a, double b, double *o) {
	double s = a + b;
	double bb = s - a;
	o[0] = s;
	o[1] = (a - (s - bb)) + (b - bb);
}
void dd_fast_two_sum(double a, double b, double *o) {
	double s = a + b;
	o[0] = s;
	o[1] = b - (s - a);
}
void dd_two_prod(double a, double b, double *o) {
	double p = a * b;
	double t = 134217729.0 * a;
	double ah = t - (t - a);
	double al = a - ah;
	double bh;
	double bl;
	t = 134217729.0 * b;
	bh = t - (t - b);
	bl = b - bh;
	o[0] = p;
	o[1] = ((ah * bh - p) + ah * bl + al * bh) + al * bl;
}
void dd_mul(double ah, double al, double bh, double bl, double *o) {
	double t[2];
	dd_two_prod(ah, bh, t);
	dd_fast_two_sum(t[0], t[1] + (ah * bl + al * bh), o);
}
void dd_add(double ah, double al, double bh, double bl, double *o) {
	double t[2];
	dd_two_sum(ah, bh, t);
	dd_fast_two_sum(t[0], t[1] + (al + bl), o);
}
/* exp(ah+al) = 2^k * (o[0]+o[1]); the answer is k. */
long d_pow_exp_dd(double ah, double al, double *o) {
	union DB u;
	double kd; double q; double s[2]; double t[2]; double v[2];
	long i;
	u.d = ah * 1.4426950408889634 + 0.5;
	u.l = d_floor(u.l);
	kd = u.d;
	dd_two_prod(kd, 1.9082149292705877e-10, t);
	dd_add(ah - kd * 6.93147180369123816490e-01, 0.0, 0.0 - t[0], 0.0 - t[1], s);
	dd_fast_two_sum(s[0], s[1] + (al - kd * 1.1612227229362532e-26), s);
	s[0] = s[0] * 0.0625;
	s[1] = s[1] * 0.0625;
	q = 1.6059043836821613e-10;
	q = 2.08767569878681e-09 + s[0] * q;
	q = 2.505210838544172e-08 + s[0] * q;
	q = 2.755731922398589e-07 + s[0] * q;
	q = 2.7557319223985893e-06 + s[0] * q;
	t[0] = q;
	t[1] = 0.0;
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(2.48015873015873e-05, 2.1511947866775882e-23, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(0.0001984126984126984, 1.7209558293420705e-22, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(0.001388888888888889, -5.300543954373577e-20, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(0.008333333333333333, 1.1564823173178714e-19, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(0.041666666666666664, 2.3129646346357427e-18, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(0.16666666666666666, 9.25185853854297e-18, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(0.5, 0.0, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t); dd_add(1.0, 0.0, t[0], t[1], t);
	dd_mul(s[0], s[1], t[0], t[1], t);
	for (i = 0; i < 4; i = i + 1) {
		dd_add(2.0, 0.0, t[0], t[1], v);
		dd_mul(t[0], t[1], v[0], v[1], t);
	}
	dd_fast_two_sum(1.0, t[0], v);
	dd_fast_two_sum(v[0], v[1] + t[1], o);
	return (long)kd;
}
void d_pow_log_dd(double x, double *o) {
	double l0 = d_log(x);
	double v[2];
	double c;
	long k = d_pow_exp_dd(l0, 0.0, v);
	c = ((d_ldexp(x, 0 - k) - v[0]) - v[1]) / v[0];
	c = c - c * c * 0.5;
	dd_fast_two_sum(l0, c, o);
}
/* x^y, rounded once, when it is a double or a midpoint between two. Twin of
 * powExact in abnf/jsrtpow.go. */
int d_pow_exact(double x, double y, double *o) {
	union DB u;
	double fr;
	long ke[1];
	long mx; long ex; long p; long ey; long i; long r; long m; long top; long t;
	long s; long n; long rest; long half;
	fr = d_frexp(x, ke);
	mx = (long)d_ldexp(fr, 53);
	ex = ke[0] - 53;
	while ((mx & 1) == 0) { mx = mx >> 1; ex = ex + 1; }
	fr = d_frexp(y, ke);
	p = (long)d_ldexp(fr, 53);
	ey = ke[0] - 53;
	while ((p & 1) == 0) { p = p >> 1; ey = ey + 1; }
	for (i = ey; i < 0 && mx != 1; i = i + 1) {
		r = (long)d_sqrt_d((double)mx);
		while (r * r > mx) { r = r - 1; }
		while ((r + 1) * (r + 1) <= mx) { r = r + 1; }
		if (r * r != mx) { return 0; }
		mx = r;
	}
	if (ey >= 0) {
		if (ey > 8 || p > 1099511627776L || p < -1099511627776L) { return 0; }
		p = p << ey;
		ex = ex * p;
	} else {
		if (ex % (1L << (0 - ey)) != 0) { return 0; }
		ex = (ex >> (0 - ey)) * p;
	}
	m = 1;
	if (mx != 1) {
		if (p < 0) { return 0; }
		for (i = 0; i < p; i = i + 1) {
			if (m > 18014398509481984L / mx) { return 0; }
			m = m * mx;
		}
		if (m >= 18014398509481984L) { return 0; }
	}
	top = ex;
	for (t = m; t > 1; t = t >> 1) { top = top + 1; }
	if (top >= -1022) {
		u.l = d_from_long(m);
		o[0] = d_ldexp(u.d, ex);
		return 1;
	}
	s = -1074 - ex;
	if (s > 60) { o[0] = 0.0; return 1; }
	n = m >> s;
	rest = m - (n << s);
	half = 1L << (s - 1);
	if (rest > half || (rest == half && (n & 1) == 1)) { n = n + 1; }
	o[0] = d_ldexp((double)n, -1074);
	return 1;
}
double d_pow_pos(double x, double y) {
	double l[2]; double ph[2]; double v[2]; double ex[1];
	double err; double c; double q; double s; double j; double rem; double up; double dn; double hi;
	long k;
	if (y == 2.0) { return x * x; }
	if (y == -1.0) { return 1.0 / x; }
	if (y == 0.5) { return d_sqrt_d(x); }
	d_pow_log_dd(x, l);
	dd_two_prod(y, l[0], ph);
	if (ph[0] > 710.0) { return 1.0 / 0.0; }
	if (ph[0] < -746.0) { return 0.0; }
	dd_fast_two_sum(ph[0], ph[1] + y * l[1], ph);
	k = d_pow_exp_dd(ph[0], ph[1], v);
	err = v[0] * 3.2311742677852644e-27;
	c = d_ldexp(1.0, -1022 - k);
	if (v[0] >= c) {
		hi = v[0] + (v[1] + err);
		if (hi == v[0] + (v[1] - err)) { return d_ldexp(hi, k); }
		if (d_pow_exact(x, y, ex)) { return ex[0]; }
		return d_ldexp(v[0] + v[1], k);
	}
	q = d_ldexp(1.0, -1074 - k);
	s = c + v[0];
	j = s - c;
	rem = v[0] - j;
	up = rem + (v[1] + err);
	dn = rem + (v[1] - err);
	if (dn > q * 0.5) {
		j = j + q;
	} else if (up < 0.0 - q * 0.5) {
		j = j - q;
	} else if (up >= q * 0.5 || dn <= 0.0 - q * 0.5) {
		if (d_pow_exact(x, y, ex)) { return ex[0]; }
		if (rem + v[1] > q * 0.5) { j = j + q; }
		else if (rem + v[1] < 0.0 - q * 0.5) { j = j - q; }
	}
	return d_ldexp(d_ldexp(j, k + 600), -600);
}
long d_pow(long xb, long yb) {
	union DB ux;
	union DB uy;
	union DB ur;
	double x; double y;
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
	if (x < 0.0) {
		if (d_modf_int(y) != y) { return DNAN; }
		ur.d = d_pow_pos(0.0 - x, y);
		if (d_odd_int(y)) { return d_neg(ur.l); }
		return ur.l;
	}
	ur.d = d_pow_pos(x, y);
	return ur.l;
}

/* ----- the transcendentals: sin cos tan asin acos atan sinh cosh tanh
 * asinh acosh atanh expm1 log1p log2 log10 cbrt atan2 hypot -----
 *
 * WHY THEY ARE PORTS. The native binary and llvm.Run have to print the SAME
 * DIGITS for every Math member abnf/jsrt.go's standardJSBindings offers, so these
 * take the road d_pow / d_exp / d_log took: a FAITHFUL PORT of Go's own math
 * package - the same constants, the same polynomials, the same order of
 * operations. An implementation that is merely accurate is a run-vs-native
 * divergence one ulp wide. Every function below is $GOROOT/src/math/<name>.go
 * line for line.
 *
 * WHAT THE PORT CANNOT REACH, AND WHY llvm.fma.f64 IS NOT THE ANSWER. A small
 * fraction of arguments still differ from llvm.Run by ONE ULP, in five
 * expressions: `(x - y*PI4A)` and its two continuations in sin/cos/tan, tan's
 * `z + z*q`, xatan's `x*z + x`, and log2's `Log(frac)*(1/Ln2) + exp`. (asin's
 * `1 - x*x` cancels, so its error is not one ulp: d_one_minus_sq computes it
 * exactly.) THE GO COMPILER CONTRACTS `a*b + c` INTO AN FMA ON
 * arm64 and the C here emits a separate fmul and fadd. Emitting an FMA here to
 * match is declined:
 *
 *   1. IT WOULD BE WRONG ON amd64. Go contracts on arm64/ppc64/s390x/riscv64/
 *      loong64 and NOT on amd64/386, so the TWIN's digits are already
 *      architecture-dependent. There is no spelling that matches both.
 *   2. IT IS A PER-EXPRESSION AUDIT, NOT A FLAG. Each site would have to be
 *      identified against Go's arm64 SSA output by hand and re-audited on every
 *      Go release - the fusion decisions belong to the Go compiler and are
 *      specified nowhere.
 *   3. THIS FILE IS NOT COMPILED BY CLANG. c-to-llvm-ir.abnf has no intrinsic
 *      syntax, so reaching llvm.fma.f64 means either changing that grammar
 *      (shared with bash, batch and c) or declaring libm's fma() - which breaks
 *      the standing property that `nm -u` on a native binary shows libc alone.
 *
 * Making the TWIN unfused instead (explicit float64() conversions) only MOVES
 * the divergence: standardJSBindings is also the frozen grammar host, so it
 * would diverge from goja's Math in every interpreter half.
 *
 * The one that is not a straight transliteration is trigReduce's 128-bit
 * arithmetic: Go writes it with math/bits, and the C subset here has no 128-bit
 * type, so u_mul64 / u_add64 / u_lzc64 do it in 32-bit halves. mPi4 is a
 * FUNCTION rather than a table because runtime.c has no other global array
 * initialiser and a global initialiser needs __mec_ginit to have run, which the
 * floor's link has no call site for. */

/* Logical (unsigned) right shift by 0..64. Go defines a shift at or past the
 * width as zero; LLVM's lshr is poison there, and trigReduce shifts by
 * `64 - bitshift` with bitshift possibly 0. */
long u_shr64(long v, long n) {
	unsigned long u;
	if (n <= 0) { return v; }
	if (n >= 64) { return 0; }
	u = (unsigned long)v;
	return (long)(u >> n);
}
int u_lt64(long a, long b) { return (unsigned long)a < (unsigned long)b; }
/* The 128-bit product of two unsigned 64-bit values: the high half is returned,
 * the low half is written to lo[0]. bits.Mul64. */
long u_mul64(long a, long b, long *lo) {
	long a0 = a & 4294967295L;
	long a1 = u_shr64(a, 32);
	long b0 = b & 4294967295L;
	long b1 = u_shr64(b, 32);
	long p00 = a0 * b0;
	long p01 = a0 * b1;
	long p10 = a1 * b0;
	long mid = u_shr64(p00, 32) + (p01 & 4294967295L) + (p10 & 4294967295L);
	lo[0] = (p00 & 4294967295L) | (mid << 32);
	return a1 * b1 + u_shr64(p01, 32) + u_shr64(p10, 32) + u_shr64(mid, 32);
}
/* bits.Add64: the sum, with the carry out written to cout[0]. */
long u_add64(long a, long b, long cin, long *cout) {
	long s = a + b;
	long c = u_lt64(s, a) ? 1 : 0;
	long s2 = s + cin;
	if (u_lt64(s2, s)) { c = 1; }
	cout[0] = c;
	return s2;
}
long u_lzc64(long v) {
	unsigned long u = (unsigned long)v;
	long n = 0;
	if (v == 0) { return 64; }
	while ((u >> 63) == 0UL) { u = u << 1; n = n + 1; }
	return n;
}

double d_fabs(double x) { union DB u; u.d = x; u.l = u.l & 9223372036854775807L; return u.d; }
double d_copysign(double x, double y) {
	union DB a;
	union DB b;
	a.d = x;
	b.d = y;
	a.l = (a.l & 9223372036854775807L) | (b.l & (0 - 9223372036854775807L - 1L));
	return a.d;
}
double d_nan_d(void) { union DB u; u.l = DNAN; return u.d; }
int d_isnan_d(double x) { union DB u; u.d = x; return d_is_nan(u.l); }
int d_isinf_d(double x) { union DB u; u.d = x; return d_is_inf(u.l); }
double d_bits_to_d(long b) { union DB u; u.l = b; return u.d; }
long d_d_to_bits(double x) { union DB u; u.d = x; return u.l; }

/* mPi4: the binary digits of 4/pi, Go's trig_reduce.go table. Read three
 * consecutive entries per reduction, so the linear chain costs nothing that
 * matters on a path taken only for |x| >= 2^29. */
long m_pi4(long i) {
	if (i == 0) { return 1L; }
	if (i == 1) { return 5040379952546458195L; } /* 0x45f306dc9c882a53 */
	if (i == 2) { return -554312585572664447L; } /* 0xf84eafa3ea69bb81 */
	if (i == 3) { return -5276763892624187261L; } /* 0xb6c52b3278872083 */
	if (i == 4) { return -242412250269775165L; } /* 0xfca2c757bd778ac3 */
	if (i == 5) { return 7946843935494350272L; } /* 0x6e48dc74849ba5c0 */
	if (i == 6) { return 905889640498799673L; } /* 0x0c925dd413a32439 */
	if (i == 7) { return -271387810574676355L; } /* 0xfc3bd63962534e7d */
	if (i == 8) { return -3385462365541201655L; } /* 0xd1046bea5d768909 */
	if (i == 9) { return -3226582509936509822L; } /* 0xd338e04d68befc82 */
	if (i == 10) { return 8296664548579374057L; } /* 0x7323ac7306a673e9 */
	if (i == 11) { return 4109744767560208502L; } /* 0x3908bf177bf25076 */
	if (i == 12) { return 4607516669194743839L; } /* 0x3ff12fffbc0b301f */
	if (i == 13) { return -2423460969061230018L; } /* 0xde5e2316b414da3e */
	if (i == 14) { return -2707510419574615186L; } /* 0xda6cfd9e4f96136e */
	if (i == 15) { return -7022098299024845734L; } /* 0x9e8c7ecd3cbfd45a */
	if (i == 16) { return -1562901285013036298L; } /* 0xea4f758fd7cbe2f6 */
	if (i == 17) { return 8795094592981902804L; } /* 0x7a0e73ef14a525d4 */
	if (i == 18) { return -2884908082597807600L; } /* 0xd7f6bf623f1aba10 */
	return -6051042886442100905L;        /* 0xac06608df8f6d757 */
}

/* Payne-Hanek reduction, trig_reduce.go. j is written to jout[0]. */
double d_trig_reduce(double x, long *jout) {
	long ix;
	long exp;
	long digit;
	long bitshift;
	long z0; long z1; long z2;
	long z2hi; long z1hi; long z1lo; long z0lo;
	long lo; long hi; long c[1]; long dummy[1];
	long j; long lz; long e;
	double z;
	if (x < 0.7853981633974483) { jout[0] = 0; return x; }
	ix = d_d_to_bits(x);
	exp = ((u_shr64(ix, 52)) & 2047) - 1023 - 52;
	ix = ix & 4503599627370495L;
	ix = ix | 4503599627370496L;
	digit = (exp + 61) / 64;
	bitshift = (exp + 61) - digit * 64;
	z0 = (m_pi4(digit) << bitshift) | u_shr64(m_pi4(digit + 1), 64 - bitshift);
	z1 = (m_pi4(digit + 1) << bitshift) | u_shr64(m_pi4(digit + 2), 64 - bitshift);
	z2 = (m_pi4(digit + 2) << bitshift) | u_shr64(m_pi4(digit + 3), 64 - bitshift);
	z2hi = u_mul64(z2, ix, dummy);
	z1hi = u_mul64(z1, ix, dummy);
	z1lo = dummy[0];
	z0lo = z0 * ix;
	lo = u_add64(z1lo, z2hi, 0, c);
	hi = u_add64(z0lo, z1hi, c[0], dummy);
	j = u_shr64(hi, 61);
	hi = (hi << 3) | u_shr64(lo, 61);
	lz = u_lzc64(hi);
	e = 1023 - (lz + 1);
	hi = (hi << (lz + 1)) | u_shr64(lo, 64 - (lz + 1));
	hi = u_shr64(hi, 64 - 52);
	hi = hi | (e << 52);
	z = d_bits_to_d(hi);
	if ((j & 1) == 1) { j = j + 1; j = j & 7; z = z - 1.0; }
	jout[0] = j;
	return z * 0.7853981633974483;
}

/* sin.c's coefficients, Go's _sin and _cos arrays. */
double d_sin(double x) {
	double y; double z; double zz;
	long j;
	long jo[1];
	int sign = 0;
	if (x == 0.0 || d_isnan_d(x)) { return x; }
	if (d_isinf_d(x)) { return d_nan_d(); }
	if (x < 0.0) { x = 0.0 - x; sign = 1; }
	if (x >= 536870912.0) {
		z = d_trig_reduce(x, jo);
		j = jo[0];
		y = 0.0;
	} else {
		j = (long)(x * 1.2732395447351628);
		y = (double)j;
		if ((j & 1) == 1) { j = j + 1; y = y + 1.0; }
		j = j & 7;
		z = ((x - y * 7.85398125648498535156e-1) - y * 3.77489470793079817668e-8)
		    - y * 2.69515142907905952645e-15;
	}
	if (j > 3) { sign = !sign; j = j - 4; }
	zz = z * z;
	if (j == 1 || j == 2) {
		y = 1.0 - 0.5 * zz + zz * zz * ((((((-1.13585365213876817300e-11 * zz) + 2.08757008419747316778e-9) * zz
		    + -2.75573141792967388112e-7) * zz + 2.48015872888517045348e-5) * zz
		    + -1.38888888888730564116e-3) * zz + 4.16666666666665929218e-2);
	} else {
		y = z + z * zz * ((((((1.58962301576546568060e-10 * zz) + -2.50507477628578072866e-8) * zz
		    + 2.75573136213857245213e-6) * zz + -1.98412698295895385996e-4) * zz
		    + 8.33333333332211858878e-3) * zz + -1.66666666666666307295e-1);
	}
	if (sign) { y = 0.0 - y; }
	return y;
}
double d_cos(double x) {
	double y; double z; double zz;
	long j;
	long jo[1];
	int sign = 0;
	if (d_isnan_d(x) || d_isinf_d(x)) { return d_nan_d(); }
	x = d_fabs(x);
	if (x >= 536870912.0) {
		z = d_trig_reduce(x, jo);
		j = jo[0];
		y = 0.0;
	} else {
		j = (long)(x * 1.2732395447351628);
		y = (double)j;
		if ((j & 1) == 1) { j = j + 1; y = y + 1.0; }
		j = j & 7;
		z = ((x - y * 7.85398125648498535156e-1) - y * 3.77489470793079817668e-8)
		    - y * 2.69515142907905952645e-15;
	}
	if (j > 3) { j = j - 4; sign = !sign; }
	if (j > 1) { sign = !sign; }
	zz = z * z;
	if (j == 1 || j == 2) {
		y = z + z * zz * ((((((1.58962301576546568060e-10 * zz) + -2.50507477628578072866e-8) * zz
		    + 2.75573136213857245213e-6) * zz + -1.98412698295895385996e-4) * zz
		    + 8.33333333332211858878e-3) * zz + -1.66666666666666307295e-1);
	} else {
		y = 1.0 - 0.5 * zz + zz * zz * ((((((-1.13585365213876817300e-11 * zz) + 2.08757008419747316778e-9) * zz
		    + -2.75573141792967388112e-7) * zz + 2.48015872888517045348e-5) * zz
		    + -1.38888888888730564116e-3) * zz + 4.16666666666665929218e-2);
	}
	if (sign) { y = 0.0 - y; }
	return y;
}
double d_tan(double x) {
	double y; double z; double zz;
	long j;
	long jo[1];
	int sign = 0;
	if (x == 0.0 || d_isnan_d(x)) { return x; }
	if (d_isinf_d(x)) { return d_nan_d(); }
	if (x < 0.0) { x = 0.0 - x; sign = 1; }
	if (x >= 536870912.0) {
		z = d_trig_reduce(x, jo);
		j = jo[0];
		y = 0.0;
	} else {
		j = (long)(x * 1.2732395447351628);
		y = (double)j;
		if ((j & 1) == 1) { j = j + 1; y = y + 1.0; }
		z = ((x - y * 7.85398125648498535156e-1) - y * 3.77489470793079817668e-8)
		    - y * 2.69515142907905952645e-15;
	}
	zz = z * z;
	if (zz > 1e-14) {
		y = z + z * (zz * (((-1.30936939181383777646e4 * zz) + 1.15351664838587416140e6) * zz
		    + -1.79565251976484877988e7)
		    / ((((zz + 1.36812963470692954678e4) * zz + -1.32089234440210967447e6) * zz
		    + 2.50083801823357915839e7) * zz + -5.38695755929454629881e7));
	} else {
		y = z;
	}
	if ((j & 2) == 2) { y = -1.0 / y; }
	if (sign) { y = 0.0 - y; }
	return y;
}

/* atan.go: xatan / satan / atan. */
double d_xatan(double x) {
	double z = x * x;
	z = z * ((((-8.750608600031904122785e-01 * z + -1.615753718733365076637e+01) * z
	    + -7.500855792314704667340e+01) * z + -1.228866684490136173410e+02) * z
	    + -6.485021904942025371773e+01)
	    / (((((z + 2.485846490142306297962e+01) * z + 1.650270098316988542046e+02) * z
	    + 4.328810604912902668951e+02) * z + 4.853903996359136964868e+02) * z
	    + 1.945506571482613964425e+02);
	z = x * z + x;
	return z;
}
double d_satan(double x) {
	if (x <= 0.66) { return d_xatan(x); }
	if (x > 2.41421356237309504880) {
		return 1.5707963267948966 - d_xatan(1.0 / x) + 6.123233995736765886130e-17;
	}
	return 0.7853981633974483 + d_xatan((x - 1.0) / (x + 1.0)) + 0.5 * 6.123233995736765886130e-17;
}
double d_atan(double x) {
	if (x == 0.0) { return x; }
	if (x > 0.0) { return d_satan(x); }
	return 0.0 - d_satan(0.0 - x);
}
/* 1 - x*x rounded ONCE, as Go's fused `1 - x*x` in asin.go is on arm64. The
 * separate fmul cancels near |x| = 1 (asin off by hundreds of ulps); Dekker's
 * split gives x*x = p + e exactly and a two-sum gives 1 - p = s + t exactly. */
double d_one_minus_sq(double x) {
	double c = 134217729.0 * x;
	double hi = c - (c - x);
	double lo = x - hi;
	double p = x * x;
	double e = ((hi * hi - p) + 2.0 * (hi * lo)) + lo * lo;
	double s = 1.0 - p;
	double bb = s - 1.0;
	double t = (1.0 - (s - bb)) + ((0.0 - p) - bb);
	return s + (t - e);
}
double d_asin(double x) {
	double temp;
	int sign = 0;
	if (x == 0.0) { return x; }
	if (x < 0.0) { x = 0.0 - x; sign = 1; }
	if (x > 1.0) { return d_nan_d(); }
	temp = d_sqrt_d(d_one_minus_sq(x));
	if (x > 0.7) { temp = 1.5707963267948966 - d_satan(temp / x); }
	else { temp = d_satan(x / temp); }
	if (sign) { temp = 0.0 - temp; }
	return temp;
}
double d_acos(double x) { return 1.5707963267948966 - d_asin(x); }
double d_atan2(double y, double x) {
	double q;
	if (d_isnan_d(y) || d_isnan_d(x)) { return d_nan_d(); }
	if (y == 0.0) {
		if (x >= 0.0 && !d_sign(d_d_to_bits(x))) { return d_copysign(0.0, y); }
		return d_copysign(3.141592653589793, y);
	}
	if (x == 0.0) { return d_copysign(1.5707963267948966, y); }
	if (d_isinf_d(x)) {
		if (!d_sign(d_d_to_bits(x))) {
			if (d_isinf_d(y)) { return d_copysign(0.7853981633974483, y); }
			return d_copysign(0.0, y);
		}
		if (d_isinf_d(y)) { return d_copysign(2.356194490192345, y); }
		return d_copysign(3.141592653589793, y);
	}
	if (d_isinf_d(y)) { return d_copysign(1.5707963267948966, y); }
	q = d_atan(y / x);
	if (x < 0.0) {
		if (q <= 0.0) { return q + 3.141592653589793; }
		return q - 3.141592653589793;
	}
	return q;
}

/* sinh.go / tanh.go. */
double d_sinh(double x) {
	double temp; double sq; double ex;
	int sign = 0;
	if (x < 0.0) { x = 0.0 - x; sign = 1; }
	if (x > 21.0) { temp = d_exp(x) * 0.5; }
	else if (x > 0.5) { ex = d_exp(x); temp = (ex - 1.0 / ex) * 0.5; }
	else {
		sq = x * x;
		temp = (((-0.2630563213397497062819489e+2 * sq + -0.2894211355989563807284660366e+4) * sq
		    + -0.8991272022039509355398013511e+5) * sq + -0.6307673640497716991184787251e+6) * x;
		temp = temp / (((sq + -0.173678953558233699533450911e+3) * sq
		    + 0.1521517378790019070696485176e+5) * sq + -0.6307673640497716991212077277e+6);
	}
	if (sign) { temp = 0.0 - temp; }
	return temp;
}
double d_cosh(double x) {
	double ex;
	x = d_fabs(x);
	if (x > 21.0) { return d_exp(x) * 0.5; }
	ex = d_exp(x);
	return (ex + 1.0 / ex) * 0.5;
}
double d_tanh(double x) {
	double z = d_fabs(x);
	double s;
	if (z > 0.5 * 8.8029691931113054295988e+01) {
		if (x < 0.0) { return -1.0; }
		return 1.0;
	}
	if (z >= 0.625) {
		s = d_exp(2.0 * z);
		z = 1.0 - 2.0 / (s + 1.0);
		if (x < 0.0) { z = 0.0 - z; }
		return z;
	}
	if (x == 0.0) { return x; }
	s = x * x;
	return x + x * s * ((-9.64399179425052238628e-1 * s + -9.92877231001918586564e1) * s
	    + -1.61468768441708447952e3)
	    / (((s + 1.12811678491632931402e2) * s + 2.23548839060100448583e3) * s
	    + 4.84406305325125486048e3);
}

/* log1p.go, then the three inverse hyperbolics that are written on top of it. */
double d_log1p(double x) {
	double absx; double f; double c; double u; double hfsq; double s; double R; double z;
	long iu;
	long k;
	if (x < -1.0 || d_isnan_d(x)) { return d_nan_d(); }
	if (x == -1.0) { return -1.0 / 0.0; }
	if (d_isinf_d(x) && !d_sign(d_d_to_bits(x))) { return x; }
	absx = d_fabs(x);
	f = 0.0;
	iu = 0;
	c = 0.0;
	k = 1;
	if (absx < 4.142135623730950488017e-01) {
		if (absx < 1.862645149230957e-09) {                /* 2**-29 */
			if (absx < 5.551115123125783e-17) { return x; }  /* 2**-54 */
			return x - x * x * 0.5;
		}
		if (x > -2.928932188134524755992e-01) { k = 0; f = x; iu = 1; }
	}
	if (k != 0) {
		if (absx < 9007199254740992.0) {
			u = 1.0 + x;
			iu = d_d_to_bits(u);
			k = (u_shr64(iu, 52)) - 1023;
			if (k > 0) { c = 1.0 - (u - x); } else { c = x - (u - 1.0); }
			c = c / u;
		} else {
			u = x;
			iu = d_d_to_bits(u);
			k = (u_shr64(iu, 52)) - 1023;
			c = 0.0;
		}
		iu = iu & 4503599627370495L;
		if (u_lt64(iu, 1865452045155277L)) {           /* 0x0006a09e667f3bcd */
			u = d_bits_to_d(iu | 4607182418800017408L);   /* | 0x3ff0000000000000 */
		} else {
			k = k + 1;
			u = d_bits_to_d(iu | 4602678819172646912L);   /* | 0x3fe0000000000000 */
			iu = u_shr64(4503599627370496L - iu, 2);
		}
		f = u - 1.0;
	}
	hfsq = 0.5 * f * f;
	if (iu == 0) {
		if (f == 0.0) {
			if (k == 0) { return 0.0; }
			c = c + (double)k * 1.90821492927058770002e-10;
			return (double)k * 6.93147180369123816490e-01 + c;
		}
		R = hfsq * (1.0 - 0.66666666666666666 * f);
		if (k == 0) { return f - R; }
		return (double)k * 6.93147180369123816490e-01
		    - ((R - ((double)k * 1.90821492927058770002e-10 + c)) - f);
	}
	s = f / (2.0 + f);
	z = s * s;
	R = z * (6.666666666666735130e-01 + z * (3.999999999940941908e-01
	    + z * (2.857142874366239149e-01 + z * (2.222219843214978396e-01
	    + z * (1.818357216161805012e-01 + z * (1.531383769920937332e-01
	    + z * 1.479819860511658591e-01))))));
	if (k == 0) { return f - (hfsq - s * (hfsq + R)); }
	return (double)k * 6.93147180369123816490e-01
	    - ((hfsq - (s * (hfsq + R) + ((double)k * 1.90821492927058770002e-10 + c))) - f);
}
double d_asinh(double x) {
	double temp;
	int sign = 0;
	if (d_isnan_d(x) || d_isinf_d(x)) { return x; }
	if (x < 0.0) { x = 0.0 - x; sign = 1; }
	if (x > 268435456.0) { temp = d_log(x) + 6.93147180559945286227e-01; }
	else if (x > 2.0) { temp = d_log(2.0 * x + 1.0 / (d_sqrt_d(x * x + 1.0) + x)); }
	else if (x < 3.725290298461914e-09) { temp = x; }
	else { temp = d_log1p(x + x * x / (1.0 + d_sqrt_d(1.0 + x * x))); }
	if (sign) { temp = 0.0 - temp; }
	return temp;
}
double d_acosh(double x) {
	double t;
	if (x < 1.0 || d_isnan_d(x)) { return d_nan_d(); }
	if (x == 1.0) { return 0.0; }
	if (x >= 268435456.0) { return d_log(x) + 0.6931471805599453; }
	if (x > 2.0) { return d_log(2.0 * x - 1.0 / (x + d_sqrt_d(x * x - 1.0))); }
	t = x - 1.0;
	return d_log1p(t + d_sqrt_d(2.0 * t + t * t));
}
double d_atanh(double x) {
	double temp;
	int sign = 0;
	if (x < -1.0 || x > 1.0 || d_isnan_d(x)) { return d_nan_d(); }
	if (x == 1.0) { return 1.0 / 0.0; }
	if (x == -1.0) { return -1.0 / 0.0; }
	if (x < 0.0) { x = 0.0 - x; sign = 1; }
	if (x < 3.725290298461914e-09) { temp = x; }
	else if (x < 0.5) { temp = x + x; temp = 0.5 * d_log1p(temp + temp * x / (1.0 - x)); }
	else { temp = 0.5 * d_log1p((x + x) / (1.0 - x)); }
	if (sign) { temp = 0.0 - temp; }
	return temp;
}

/* expm1.go. The three `Float64frombits(Float64bits(y) + k<<52)` steps are exponent
 * arithmetic on the bit pattern, which is why this cannot be written in MetaJS
 * either (manual 7.7). */
double d_expm1(double x) {
	double absx; double c; double hi; double lo; double t; double hfx; double hxs;
	double r1; double e; double y;
	long k;
	int sign = 0;
	if (d_isnan_d(x)) { return x; }
	if (d_isinf_d(x)) { if (d_sign(d_d_to_bits(x))) { return -1.0; } return x; }
	absx = x;
	if (x < 0.0) { absx = 0.0 - absx; sign = 1; }
	if (absx >= 3.88162421113569373274e+01) {
		if (sign) { return -1.0; }
		if (absx >= 7.09782712893383973096e+02) { return 1.0 / 0.0; }
	}
	c = 0.0;
	k = 0;
	if (absx > 3.46573590279972654709e-01) {
		if (absx < 1.03972077083991796413e+00) {
			if (!sign) { hi = x - 6.93147180369123816490e-01; lo = 1.90821492927058770002e-10; k = 1; }
			else { hi = x + 6.93147180369123816490e-01; lo = 0.0 - 1.90821492927058770002e-10; k = -1; }
		} else {
			if (!sign) { k = (long)(1.44269504088896338700e+00 * x + 0.5); }
			else { k = (long)(1.44269504088896338700e+00 * x - 0.5); }
			t = (double)k;
			hi = x - t * 6.93147180369123816490e-01;
			lo = t * 1.90821492927058770002e-10;
		}
		x = hi - lo;
		c = (hi - x) - lo;
	} else if (absx < 5.551115123125783e-17) {              /* 2**-54 */
		return x;
	}
	hfx = 0.5 * x;
	hxs = x * hfx;
	r1 = 1.0 + hxs * (-3.33333333333331316428e-02 + hxs * (1.58730158725481460165e-03
	    + hxs * (-7.93650757867487942473e-05 + hxs * (4.00821782732936239552e-06
	    + hxs * -2.01099218183624371326e-07))));
	t = 3.0 - r1 * hfx;
	e = hxs * ((r1 - t) / (6.0 - x * t));
	if (k == 0) { return x - (x * e - hxs); }
	e = x * (e - c) - c;
	e = e - hxs;
	if (k == -1) { return 0.5 * (x - e) - 0.5; }
	if (k == 1) {
		if (x < -0.25) { return -2.0 * (e - (x + 0.5)); }
		return 1.0 + 2.0 * (x - e);
	}
	if (k <= -2 || k > 56) {
		y = 1.0 - (e - x);
		y = d_bits_to_d(d_d_to_bits(y) + (k << 52));
		return y - 1.0;
	}
	if (k < 20) {
		t = d_bits_to_d(4607182418800017408L - u_shr64(9007199254740992L, k));  /* 1-2**-k */
		y = t - (e - x);
		y = d_bits_to_d(d_d_to_bits(y) + (k << 52));
		return y;
	}
	t = d_bits_to_d((1023 - k) << 52);                       /* 2**-k */
	y = x - (e + t);
	y = y + 1.0;
	y = d_bits_to_d(d_d_to_bits(y) + (k << 52));
	return y;
}

/* log10.go: Log10 and Log2. Log2's frac == 0.5 arm is what makes Math.log2 of a
 * power of two EXACT, which the plain Log(x)/Ln2 is not. */
double d_log10(double x) { return d_log(x) * 0.4342944819032518; }
double d_log2(double x) {
	double frac;
	long e[1];
	frac = d_frexp(x, e);
	if (frac == 0.5) { return (double)(e[0] - 1); }
	return d_log(frac) * 1.4426950408889634 + (double)e[0];
}

/* cbrt.go. */
double d_cbrt(double x) {
	double t; double r; double s; double w;
	int sign = 0;
	if (x == 0.0 || d_isnan_d(x) || d_isinf_d(x)) { return x; }
	if (x < 0.0) { x = 0.0 - x; sign = 1; }
	t = d_bits_to_d((long)((unsigned long)d_d_to_bits(x) / 3UL) + 3071306043645493248L); /* B1<<32 */
	if (x < 2.22507385850720138309e-308) {
		t = 18014398509481984.0;                              /* 2**54 */
		t = t * x;
		t = d_bits_to_d((long)((unsigned long)d_d_to_bits(t) / 3UL) + 2990241250352824320L); /* B2<<32 */
	}
	r = t * t / x;
	s = 5.42857142857142815906e-01 + r * t;
	t = t * (3.57142857142857150787e-01 + 1.60714285714285720630e+00
	    / (s + 1.41428571428571436819e+00 + -7.05306122448979611050e-01 / s));
	t = d_bits_to_d((d_d_to_bits(t) & (68719476732L << 28)) + 1073741824L);
	s = t * t;
	r = x / s;
	w = t + t;
	r = (r - t) / (w + r);
	t = t + t * r;
	if (sign) { t = 0.0 - t; }
	return t;
}

/* Math.fround: the double nearest x that a float32 can hold. `float` is ACCEPTED
 * by c-to-llvm-ir.abnf and silently means `double` - `(float)0.1` came back as
 * plain 0.1 - so the narrowing is done on the bit pattern here.
 * Round to nearest, ties to even, over the 24-bit significand, with the float32
 * subnormal range and the overflow to infinity both explicit. */
double d_fround(double x) {
	union DB u;
	long b; long sgn; long e; long m; long full; long sh; long q; long r; long half; long E;
	u.d = x;
	b = u.l;
	sgn = b & (0 - 9223372036854775807L - 1L);
	e = (b >> 52) & 2047;
	m = b & 4503599627370495L;
	if (e == 2047) { return x; }                        /* NaN and +-Inf pass through */
	if (e == 0) { u.l = sgn; return u.d; }              /* +-0 and every double subnormal */
	full = m | 4503599627370496L;                       /* the 53-bit significand */
	E = e - 1023;
	if (E > 127) { u.l = sgn | DINF; return u.d; }
	sh = (E >= -126) ? 29 : 29 + (-126 - E);
	if (sh > 62) { u.l = sgn; return u.d; }
	q = full >> sh;
	r = full - (q << sh);
	half = 1L << (sh - 1);
	if (r > half || (r == half && (q & 1) == 1)) { q = q + 1; }
	if (E >= -126) {
		if (E == 127 && q == 16777216L) { u.l = sgn | DINF; return u.d; }
		u.d = d_ldexp((double)q, E - 23);
	} else {
		u.d = d_ldexp((double)q, -149);
	}
	u.l = u.l | sgn;
	return u.d;
}
/* Math.clz32: the leading zero count of ToUint32(x); 0 has all 32. */
long d_clz32(long v) {
	long u = v & 4294967295L;
	long n = 0;
	if (u == 0) { return 32; }
	while ((u & 2147483648L) == 0) { u = u << 1; n = n + 1; }
	return n;
}

/* Math.hypot is VARIADIC in JavaScript, so it cannot be Go's two-argument
 * math.Hypot; the loop lives in host_call below and its twin is jsrt.go's
 * mathObj "hypot". The naive sum of squares makes Math.hypot(1e300, 1e300)
 * Infinity where node answers 1.41...e+300 - ECMA-262 asks implementations to
 * avoid exactly that overflow - so both engines scale by the largest magnitude
 * first, which for TWO arguments is bit for bit Go's own math.Hypot
 * (p*Sqrt(1+(q/p)^2), since (p/p)^2 is exactly 1). */

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

/* Every number in this file travels as its IEEE-754 bit pattern in a long (see
 * the C SUBSET NOTES at the top), so the transcendental host arms need the two
 * conversions in one place rather than a `union DB` per arm. */
long mk_dnum(double v) { union DB u; u.d = v; return mk_num(u.l); }
double arg_d(long args, long i) { union DB u; u.l = arg_num(args, i); return u.d; }

long host_call(long id, long self, long args) {
	long n = arr_len(args);
	long i = 0;
	/* A switch rather than a chain of ifs: clang folds the case compares into a
	 * jump table, so a builtin costs the same whatever its id. Every case
	 * returns or exits; none falls through. */
	switch (id) {
	case 1: case 2: case 3: {
		OUTFD = (id == 3) ? 2 : 1;
		while (i < n) { if (i > 0) { o_ch(32); } print_val(arr_get(args, i)); i = i + 1; }
		if (id != 2) { o_ch(10); }
		OUTFD = 1;
		return H_UNDEF;
	}
	case 4: {
		long r = arg_num(args, 1);
		return mk_num(js_parse_int(arg_str(args, 0), d_is_nan(r) ? 0 : d_to_long(d_trunc(r))));
	}
	case 5: { return mk_num(js_parse_float(arg_str(args, 0))); }
	case 6: {
		long v = arg_num(args, 0);
		exit((int)(d_is_nan(v) ? 0 : to_int32(v)));
	}
	case 10: { return mk_num(d_floor(arg_num(args, 0))); }
	case 11: { return mk_num(d_abs(arg_num(args, 0))); }
	case 12: case 13: {
		long acc = (id == 12) ? DNINF : DINF;
		while (i < n) {
			long v = to_number(arr_get(args, i));
			if (d_is_nan(v) || d_is_nan(acc)) { acc = DNAN; }
			/* Two zeros compare equal; max prefers +0 and min -0 (the sign bit
			 * is the sign of the raw long). */
			else if (d_is_zero(v) && d_is_zero(acc)) {
				if (id == 12 ? acc < 0 : v < 0) { acc = v; }
			}
			else if (id == 12) { if (d_lt(acc, v)) { acc = v; } }
			else { if (d_lt(v, acc)) { acc = v; } }
			i = i + 1;
		}
		return mk_num(acc);
	}
	case 14: {
		long a = to_int32(arg_num(args, 0));
		long b = to_int32(arg_num(args, 1));
		long p = a * b;
		p = p & 4294967295;
		if (p >= 2147483648) { p = p - 4294967296; }
		return mk_num(d_from_long(p));
	}
	case 15: { return mk_num(d_ceil(arg_num(args, 0))); }
	case 16: { return mk_num(d_trunc(arg_num(args, 0))); }
	/* Math.round: jsMathRound in abnf/jsrt.go, step for step. floor(x + 0.5)
	 * is wrong twice: the addition rounds 0.49999999999999994 up to 1, and a
	 * negative operand that rounds to zero must give -0. */
	case 17: {
		long v = arg_num(args, 0);
		long r;
		union DB h;
		union DB one;
		if (d_is_nan(v) || d_is_inf(v) || d_is_zero(v)) { return mk_num(v); }
		h.d = 0.5;
		one.d = 1.0;
		r = d_floor(v);
		if (!d_lt(d_sub(v, r), h.l)) { r = d_add(r, one.l); }
		if (d_is_zero(r) && v < 0) { r = 1; r = r << 63; }
		return mk_num(r);
	}
	case 18: {
		long v = arg_num(args, 0);
		if (d_is_nan(v) || d_is_zero(v)) { return mk_num(v); }
		return mk_num(d_sign(v) ? d_neg(DONE) : DONE);
	}
	case 19: { return mk_num(d_sqrt(arg_num(args, 0))); }
	case 20: { return mk_num(d_pow(arg_num(args, 0), arg_num(args, 1))); }
	case 30: { return from_char_code(args); }
	case 31: { return mk_bool(tag_of(arg_at(args, 0)) == 5); }
	case 32: { return mk_num(d_from_long(str_len(arg_str(args, 0)))); }
	/* fail(msg): a RUNTIME ERROR, not an exception. The Go twin's rt.fail
	 * (abnf/jsrt.go) panics with exactly this text, so the two engines report an
	 * impossible operand the same way; throwing instead would prefix the message
	 * with "uncaught exception: ". Layer 2 of every language needs it - Lua's
	 * "attempt to perform arithmetic on a nil value" arrives here - and
	 * languages/lib/interp-core.js already calls a fail() of its own. */
	case 36: {
		werr("js runtime error: ");
		wstr(to_string(arg_at(args, 0)));
		werr("\n");
		exit(1);
	}
	case 33: {                                        /* sprintf */
		if (n == 0) { return mk_cstr(""); }
		return fmt_apply(arg_str(args, 0), args, 1);
	}
	case 34: {                                        /* printf */
		if (n == 0) { return H_UNDEF; }
		OUTFD = 1;
		o_str(fmt_apply(arg_str(args, 0), args, 1));
		return H_UNDEF;
	}
	case 35: { return fmt_sprint(args); }             /* sprint */
	/* ----- the SIZED INTEGER, for layer 2.
	 * A MetaJS runtime library cannot make a new primitive type, so the floor
	 * hands it the constructor and the three readers. The value travels as two
	 * 32 bit halves because a MetaJS number is a double and cannot carry 64 bits
	 * exactly - the same {h, l} pair languages/lib/interp-core.js already uses.
	 *   sint(hi, lo, bits, unsigned)  build one, through si_norm: the result is a
	 *                                 PLAIN NUMBER when the invariant says so
	 *   sintIs(v)                     is this a sized integer (tag 13)
	 *   sintHi(v) / sintLo(v)         the two halves back out, unsigned
	 *   sintWidth(v) / sintUns(v)     the declared width and signedness */
	case 40: {
		long hi = d_to_long(d_trunc(arg_num(args, 0))) & 4294967295;
		long lo = d_to_long(d_trunc(arg_num(args, 1))) & 4294967295;
		long w = n > 2 ? d_to_long(d_trunc(arg_num(args, 2))) : 64;
		long u = n > 3 ? truthy(arg_at(args, 3)) : 0;
		return si_norm((hi << 32) | lo, w, u);
	}
	case 41: { return mk_bool(tag_of(arg_at(args, 0)) == 13); }
	case 42: { return mk_num(d_from_long((long)(si_u(si_val(arg_at(args, 0)), 64) >> 32))); }
	case 43: { return mk_num(d_from_long(si_val(arg_at(args, 0)) & 4294967295)); }
	case 44: { long v = arg_at(args, 0); return mk_num(d_from_long(si_width_of(v, v))); }
	case 45: { long v = arg_at(args, 0); return mk_bool(si_uns_of(v, v) != 0); }
	/*   sintOp(code, l, r)            one binary operator, by index:
	 *                                 0 + 1 - 2 * 3 / 4 % 5 & 6 | 7 ^ 8 &^ 9 << 10 >>
	 *   sintCmp(l, r)                 -1 / 0 / 1, unsigned where the type is
	 *   sintConv(v, bits, unsigned)   an explicit conversion (js_giconv)
	 *   sintStr(v) / sintNum(v)       the decimal text and the double reading */
	case 46: { return si_apply(d_to_long(d_trunc(arg_num(args, 0))), arg_at(args, 1), arg_at(args, 2)); }
	case 47: { return mk_num(d_from_long(si_cmp(arg_at(args, 0), arg_at(args, 1)))); }
	case 48: {
		return si_norm(si_val(arg_at(args, 0)), d_to_long(d_trunc(arg_num(args, 1))), truthy(arg_at(args, 2)));
	}
	case 49: { long v = arg_at(args, 0); if (tag_of(v) == 13) { return si_str(v); } return to_string(v); }
	case 50: { return mk_num(si_float(arg_at(args, 0))); }
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
	case 51: {
		long sty = n > 1 ? d_to_long(d_trunc(arg_num(args, 1))) : 0;
		return jf_make(to_number(arg_at(args, 0)), sty);
	}
	case 52: { return mk_bool(tag_of(arg_at(args, 0)) == 14); }
	case 53: { return mk_num(to_number(arg_at(args, 0))); }
	case 54: { long v = arg_at(args, 0); return mk_num(d_from_long(jf_style_of(v, v))); }
	case 55: { long v = arg_at(args, 0); if (tag_of(v) == 14) { return jf_text(v); } return to_string(v); }
	case 56: { return jf_arith(d_to_long(d_trunc(arg_num(args, 0))), arg_at(args, 1), arg_at(args, 2)); }
	case 57: {
		long l = arg_at(args, 0);
		long r = arg_at(args, 1);
		if (tag_of(l) == 14) { return mk_bool(jf_num_eq(fa(l), r)); }
		if (tag_of(r) == 14) { return mk_bool(jf_num_eq(fa(r), l)); }
		return mk_bool(strict_eq(l, r));
	}
	case 58: case 59: {
		long l = arg_at(args, 0);
		long r = arg_at(args, 1);
		long ln = to_number(l);
		long rn = to_number(r);
		/* jvmMathObject's max/min, through jf_take_l - which is `>` / `<` plus
		 * the signed-zero and NaN cases those two cannot see. */
		return jf_minmax(l, r, jf_take_l(ln, rn, id == 58));
	}
	case 60: { return jf_abs(arg_at(args, 0)); }
	/* keysOf(o): an object's OWN keys, in insertion order. It is js_keys, and
	 * it is here for the reason the sint* and flo* families are: an EXTERN is
	 * only callable from an emitter, so a layer-2 file could not enumerate an
	 * object at all - MetaJS has neither for..in nor Object.keys. Every language
	 * whose runtime has to render, compare or copy an object needs it (PHP's
	 * var_dump, `==` on objects, the (array) cast and clone are the first four).
	 * A dict box ({__dict,keys,vals}) yields its key array; a plain object
	 * yields its string keys, skipping the internal __-prefixed slots. */
	case 61: { return js_keys(arg_at(args, 0)); }
	/* sintRaw(hi, lo, bits, unsigned): sint() WITHOUT si_norm - always a tag 13
	 * cell, never a plain number.
	 *
	 * WHY IT EXISTS. si_norm's whole job is to UNBOX a signed 64 bit value a
	 * double holds exactly, and that is precisely what a statically typed
	 * language's 64 bit signed type must NOT do: Java's `1000000L * 1000000L` is
	 * 1000000000000 while `1000000 * 1000000` is -727379968, so a small long has
	 * to stay a long. Every other sint* builtin goes through si_norm.
	 *
	 * si_trunc IS still applied, so the cell keeps the invariant its header
	 * states (`the value already truncated to the width and sign extended`) -
	 * a caller asking for 8 bits gets a well formed int8 box, not 300 labelled
	 * as one. The ONLY thing dropped is si_norm's unboxing arm. */
	case 62: {
		long hi = d_to_long(d_trunc(arg_num(args, 0))) & 4294967295;
		long lo = d_to_long(d_trunc(arg_num(args, 1))) & 4294967295;
		long w = n > 2 ? d_to_long(d_trunc(arg_num(args, 2))) : 64;
		long u = n > 3 ? truthy(arg_at(args, 3)) : 0;
		return si_make(si_trunc((hi << 32) | lo, w, u), w, u);
	}
	/* delKey(o, key): remove an own property, key-order entry and all. It is
	 * js_del, and it is here for the reason keysOf is: an EXTERN is only callable
	 * from an emitter, and MetaJS has no `delete`, so a layer-2 file could not
	 * remove a key AT ALL. Every
	 * language whose runtime models `delete`, `unset`, `remove` or a map erase
	 * needs it. Answers true whether or not the key was there, like the twin. */
	case 63: { return js_del(arg_at(args, 0), arg_at(args, 1)); }
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
	 * MetaJS has no way to open a cell - short of lowering every such probe
	 * into each language's EMITTER, which is IR-building duplicated per language
	 * for one language-neutral question.
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
	case 64: {
		long p = arg_at(args, 0);
		return mk_scope(is_undef_or_null(p) ? 0 : host_scope(p));
	}
	case 65: {
		long p = ff(host_scope(arg_at(args, 0)));
		if (p == 0) { return H_UNDEF; }
		return p;
	}
	case 66: { return scope_get(host_scope(arg_at(args, 0)), to_string(arg_at(args, 1))); }
	case 67: {
		return mk_bool(scope_find(host_scope(arg_at(args, 0)), to_string(arg_at(args, 1))) >= 0);
	}
	case 68: {
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
	 * such name (without this, a layer 2 can only guess - exclude every other
	 * box and test whether v["next"] is callable).
	 *
	 * A generator is an OBJECT to typeof in all three halves, so this is a
	 * refinement of typeof and never contradicts it. */
	case 69: { return mk_bool(tag_of(arg_at(args, 0)) == 15); }
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
	 * its comment for why g_mindig = n, which only starts the search there, is
	 * a different function.
	 * The trailing zeros come off here, because the CALLER decides how many
	 * digits to show.
	 *
	 * The twins are floPrecStr in abnf/commonscript.go (goja and the frozen
	 * grammar host) and the floPrec binding of languages/metajs-interpreter.abnf. */
	case 70: {
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
	/* THE MATH TRANSCENDENTALS LIVE AT THE COLD END OF THIS CHAIN, ON PURPOSE.
	 * host_call dispatches on a LINEAR if-chain, so arms placed next to their
	 * neighbours (after id 20) would put extra integer compares in front of every
	 * sint / flo / scope call - measurable on java, which is nothing but boxed
	 * doubles and sized integers. Ids 71..93 are the coldest names in the
	 * runtime; the hot ones keep their position. Whether a new arm costs an
	 * existing id a compare can be checked by diffing host_call's `icmp eq i64`
	 * sites in languages/lib/runtime.ll.
	 *
	 * Seeding the Math members puts a few kilobytes on the live set for good
	 * (cells plus key strings held by the root scope). Making the seeding lazy
	 * would need a property-miss hook the floor does not have.
	 */
	/* The transcendental members of Math, host ids 71..91 - the set
	 * standardJSBindings in abnf/jsrt.go offers. Each body is Go's own math
	 * package, ported above. */
	case 71: { return mk_dnum(d_sin(arg_d(args, 0))); }
	case 72: { return mk_dnum(d_cos(arg_d(args, 0))); }
	case 73: { return mk_dnum(d_tan(arg_d(args, 0))); }
	case 74: { return mk_dnum(d_asin(arg_d(args, 0))); }
	case 75: { return mk_dnum(d_acos(arg_d(args, 0))); }
	case 76: { return mk_dnum(d_atan(arg_d(args, 0))); }
	case 77: { return mk_dnum(d_sinh(arg_d(args, 0))); }
	case 78: { return mk_dnum(d_cosh(arg_d(args, 0))); }
	case 79: { return mk_dnum(d_tanh(arg_d(args, 0))); }
	case 80: { return mk_dnum(d_asinh(arg_d(args, 0))); }
	case 81: { return mk_dnum(d_acosh(arg_d(args, 0))); }
	case 82: { return mk_dnum(d_atanh(arg_d(args, 0))); }
	case 83: { return mk_dnum(d_exp(arg_d(args, 0))); }
	case 84: { return mk_dnum(d_expm1(arg_d(args, 0))); }
	case 85: { return mk_dnum(d_log(arg_d(args, 0))); }
	case 86: { return mk_dnum(d_log1p(arg_d(args, 0))); }
	case 87: { return mk_dnum(d_log2(arg_d(args, 0))); }
	case 88: { return mk_dnum(d_log10(arg_d(args, 0))); }
	case 89: { return mk_dnum(d_cbrt(arg_d(args, 0))); }
	case 90: { return mk_dnum(d_atan2(arg_d(args, 0), arg_d(args, 1))); }
	case 91: {
		/* Math.hypot, variadic and scaled - see the Math.hypot note above.
		 * ECMA-262 orders the special cases: an infinity wins over a NaN. */
		double mx = 0.0;
		double sum = 0.0;
		double t;
		int sawinf = 0;
		int sawnan = 0;
		while (i < n) {
			t = arg_d(args, i);
			if (d_isinf_d(t)) { sawinf = 1; }
			if (d_isnan_d(t)) { sawnan = 1; }
			t = d_fabs(t);
			if (t > mx) { mx = t; }
			i = i + 1;
		}
		if (sawinf) { return mk_dnum(1.0 / 0.0); }
		if (sawnan) { return mk_dnum(d_nan_d()); }
		if (mx == 0.0) { return mk_dnum(0.0); }
		i = 0;
		while (i < n) { t = arg_d(args, i) / mx; sum = sum + t * t; i = i + 1; }
		return mk_dnum(mx * d_sqrt_d(sum));
	}
	/* clz32 and fround: the Go twin's (abnf/jsrt.go standardJSBindings), which
	 * the goja grammar host also uses (installGojaMathSurface). */
	case 92: { return mk_num(d_from_long(d_clz32(to_int32(arg_num(args, 0))))); }
	case 93: { return mk_dnum(d_fround(arg_d(args, 0))); }
	/* fnMeta(f, key) / fnMetaSet(f, name, length): see js_jsfnmeta. */
	case 94: { return fn_meta_get(arg_at(args, 0), arg_str(args, 1)); }
	case 95: {
		/* A fourth argument is fn_meta_get's "src": where the source text is. */
		if (arr_len(args) > 3 && tag_of(arg_at(args, 3)) == 3 && fn_kind(arg_at(args, 0))) {
			sd(arg_at(args, 0), d_to_long(d_trunc(arg_num(args, 3))));
		}
		return js_jsfnmeta(arg_at(args, 0), arg_str(args, 1), mk_num(arg_num(args, 2)));
	}
	/* Math.random: xorshift64, seeded once from getentropy (time() if that
	 * fails). The top 52 bits of the state become the mantissa of a double in
	 * [1, 2), and subtracting 1 gives [0, 1). The Go twin uses math/rand. */
	case 96: {
		long x = RNG_STATE;
		union DB one;
		if (x == 0) {
			if (getentropy(&x, 8) != 0 || x == 0) { x = time(0) | 1; }
		}
		x = x ^ (x << 13);
		x = x ^ ((x >> 7) & 144115188075855871);
		x = x ^ (x << 17);
		RNG_STATE = x;
		one.d = 1.0;
		return mk_num(d_sub(((x >> 12) & 4503599627370495) | one.l, one.l));
	}
	/* unicodeCase(s, mode): see case_str. */
	case 97: { return case_str(arg_str(args, 0), d_to_long(arg_num(args, 1))); }
	/* writeBytes(s): the low byte of each UTF-16 unit of s, to standard output -
	 * the raw output of a language whose strings hold one byte per unit (lua).
	 * Twin of lowBytes in abnf/jsrt.go. */
	case 98: {
		long s = arg_str(args, 0);
		const char *p = str_ptr(s);
		long len = str_len(s);
		while (i < len) {
			long cp;
			u_decode(p, len, i);
			i = i + G_ADV;
			cp = G_CP;
			if (cp >= 65536) {
				putchar((int)((55296 + ((cp - 65536) >> 10)) & 255));
				cp = 56320 + ((cp - 65536) & 1023);
			}
			putchar((int)(cp & 255));
		}
		return H_UNDEF;
	}
	/* arrKind(a) / arrKindSet(a, k): the small kind number a language keeps on
	 * an array cell, in field d (the collector never reads it; mk_arr zeroes
	 * it, so a copy starts at 0). Twin of jsArray.kind in abnf/jsrt.go. */
	case 99: {
		long a = arg_at(args, 0);
		if (tag_of(a) != 5) { return mk_num(d_from_long(0)); }
		return mk_num(d_from_long(fd(a)));
	}
	case 100: {
		long a = arg_at(args, 0);
		if (tag_of(a) == 5) { sd(a, d_to_long(arg_num(args, 1))); }
		return a;
	}
	/* arrProps(a) / arrPropsSet(a, o): the object holding the NAMED (non-index)
	 * properties a language keeps on an array, in field e - undefined until set,
	 * since mk_arr leaves e at handle 0, and not carried over to a copy. Twin of
	 * jsArray.props in abnf/jsrt.go. */
	case 106: {
		long a = arg_at(args, 0);
		if (tag_of(a) != 5) { return H_UNDEF; }
		return fe(a);
	}
	case 107: {
		long a = arg_at(args, 0);
		if (tag_of(a) == 5) { se(a, arg_at(args, 1)); }
		return a;
	}
	/* byteAt(s, i) / byteSub(s, lo, hi) / bytesStr(a): a string as the BYTES a
	 * cell holds (UTF-8, WTF-8 for a lone surrogate), for a language whose
	 * strings are byte strings (go). byteAt is the i-th byte, -1 outside s;
	 * byteSub the bytes [lo, hi) clamped to s, kept as they are even where they
	 * cut a character; bytesStr the string whose bytes are a's numbers. Twin of
	 * byteBindings in abnf/jsrtint.go. */
	case 101: {
		long s = arg_str(args, 0);
		long k = arg_idx(args, 1);
		if (k < 0 || k >= str_len(s)) { return mk_num(d_from_long(-1)); }
		return mk_num(d_from_long((long)str_ptr(s)[k] & 255));
	}
	case 102: {
		long s = arg_str(args, 0);
		long len = str_len(s);
		long lo = arg_idx(args, 1);
		long hi = arg_idx(args, 2);
		if (hi > len) { hi = len; }
		if (lo < 0) { lo = 0; }
		if (hi < lo) { hi = lo; }
		return mk_str(str_ptr(s) + lo, hi - lo);
	}
	case 103: {
		long a = arg_at(args, 0);
		long len = tag_of(a) == 5 ? arr_len(a) : 0;
		char *buf = ar_alloc(len + 1);
		long h;
		while (i < len) {
			long v = to_number(arr_get(a, i));
			buf[i] = (char)((d_is_nan(v) ? 0 : d_to_long(v)) & 255);
			i = i + 1;
		}
		buf[len] = 0;
		h = cell_new(4);
		sa(h, (long)buf);
		sb(h, len);
		return h;
	}
	/* osQuery(kind, arg): what an os library needs from the machine - 0 the
	 * time in seconds, 1 the processor time in seconds (clock() counts
	 * microseconds on darwin and glibc), 2 the local zone's offset east of UTC
	 * in seconds at time arg, 3 its abbreviation, 4 whether daylight saving time
	 * is in effect then, 5 the environment variable named arg, undefined when
	 * unset. localtime_r's struct tm is read as longs: tm_isdst is the low half
	 * of word 4, tm_gmtoff word 5 and tm_zone word 6, the layout of darwin and
	 * glibc on a 64-bit machine. Twin of osBindings in abnf/jsrtint.go. */
	case 104: {
		long kind = d_to_long(arg_num(args, 0));
		long tm[8];
		long t;
		if (kind == 0) { return mk_num(d_from_long(time(0))); }
		if (kind == 1) { return mk_dnum((double)clock() / 1000000.0); }
		if (kind == 5) {
			long s = arg_str(args, 1);
			long len = str_len(s);
			char *buf = ar_alloc(len + 1);
			char *e;
			while (i < len) { buf[i] = str_ptr(s)[i]; i = i + 1; }
			buf[len] = 0;
			e = getenv(buf);
			if (e == 0) { return H_UNDEF; }
			return mk_cstr(e);
		}
		t = d_to_long(arg_num(args, 1));
		if (localtime_r(&t, tm) == 0) {
			if (kind == 3) { return mk_cstr("UTC"); }
			if (kind == 4) { return mk_bool(0); }
			return mk_num(d_from_long(0));
		}
		if (kind == 2) { return mk_num(d_from_long(tm[5])); }
		if (kind == 3) { return mk_cstr((const char *)tm[6]); }
		return mk_bool((tm[4] & 4294967295) != 0);
	}
	/* readIn(kind): standard input, read through one buffer for the whole run -
	 * kind 0 the next line with its "\n" ("" at the end of the input), 1
	 * everything left. Standard output is flushed first, so a prompt shows
	 * before the read waits. Twin of readIn in abnf/jsrtint.go. */
	case 105: {
		long kind = d_to_long(arg_num(args, 0));
		long cap = 64;
		long done = 0;
		char *out = ar_alloc(cap);
		char *buf;
		fflush(0);
		if (IN_BUF == 0) { IN_BUF = (long)malloc(4096); }
		buf = (char *)IN_BUF;
		while (done == 0) {
			if (IN_POS >= IN_LEN) {
				IN_LEN = read(0, buf, 4096);
				IN_POS = 0;
				if (IN_LEN <= 0) { IN_LEN = 0; done = 1; }
			}
			while (done == 0 && IN_POS < IN_LEN) {
				char c = buf[IN_POS];
				IN_POS = IN_POS + 1;
				if (i == cap) {
					char *grown = ar_alloc(cap * 2);
					long k = 0;
					while (k < i) { grown[k] = out[k]; k = k + 1; }
					out = grown;
					cap = cap * 2;
				}
				out[i] = c;
				i = i + 1;
				if (kind == 0 && c == 10) { done = 1; }
			}
		}
		return mk_str(out, i);
	}
	}
	die("unknown host function");
	return H_UNDEF;
}

/* ---------------------------------------------------- exceptions --------- */

/* THE TEXT OF AN EXCEPTION NOBODY CAUGHT. to_string of an object is
 * "[object Object]" - the correct answer for String(o) and a useless one here,
 * so an uncaught `new TypeError("boom")` must print "TypeError: boom", as
 * llvm.Run does (its jsvString consults the value's own toString).
 *
 * Two shapes are recognised, and they are the two the exception hierarchies in
 * this project actually use - this is deliberately NOT a toString dispatch,
 * because the js Error's toString lives on the CLASS descriptor and calling it
 * from here would mean a __class walk plus a re-entrant js_call inside the
 * abort path:
 *
 *   {name, message}   the js/ts Error instance (js_jserrinit writes both as OWN
 *                     properties) -> "TypeError: boom", node's own first line.
 *   {__class, args}   the python / ruby style instance -> str(e), which is what
 *                     the interpreter halves already print.
 *
 * Anything else keeps to_string exactly. The twin is abnf/jsrt.go's excText. */
long exc_text(long v) {
	long nm;
	long msg;
	long args;
	if (tag_of(v) != 6) { return to_string(v); }
	nm = obj_get(v, mk_cstr("name"));
	msg = obj_get(v, mk_cstr("message"));
	if (tag_of(msg) == 4) {
		if (tag_of(nm) == 4 && str_len(nm) > 0) {
			if (str_len(msg) == 0) { return nm; }
			return str_cat(str_cat(nm, mk_cstr(": ")), msg);
		}
		return msg;
	}
	args = obj_get(v, mk_cstr("args"));
	if (tag_of(args) == 5 && arr_len(args) > 0) { return to_string(arr_get(args, 0)); }
	return to_string(v);
}

long js_throw(long v) {
	THROWN = v;
	if (JB_DEPTH <= 0) {
		long s = exc_text(v);
		OUTFD = 2;
		o_cstr("js runtime error: uncaught exception: ");
		o_str(s);
		o_ch(10);
		exit(1);
	}
	{ long *p = (long *)JBP; longjmp((void *)p[JB_DEPTH - 1], 1); }
	return H_UNDEF;
}

/* A try clause: a program closure runs without counting a call (see CALL_DEPTH). */
long call_clause(long c, long args) {
	if (tag_of(c) == 7) { return jsdispatch(fa(c), fb(c), args); }
	return js_call(c, H_UNDEF, args);
}

long js_try(long tryC, long catchC, long finC) {
	long result = H_UNDEF;
	long pending = H_UNDEF;
	int havePending = 0;              /* a separate flag: `throw undefined` is legal */
	long mydepth = JB_DEPTH;
	long calls = CALL_DEPTH;
	long buf;
	int hasCatch = is_callable(catchC);
	int hasFin = is_callable(finC);
	buf = jb_at(JB_DEPTH);
	JB_DEPTH = JB_DEPTH + 1;
	if (setjmp((void *)buf) == 0) {
		result = call_clause(tryC, mk_arr());
		JB_DEPTH = mydepth;
	} else {
		JB_DEPTH = mydepth;
		CALL_DEPTH = calls;
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
			result = call_clause(catchC, a);
			JB_DEPTH = mydepth;
		} else {
			JB_DEPTH = mydepth;
			CALL_DEPTH = calls;
			pending = THROWN;
			havePending = 1;
		}
	}
	if (hasFin) {
		long fres = H_UNDEF;
		buf = jb_at(JB_DEPTH);
			JB_DEPTH = JB_DEPTH + 1;
		if (setjmp((void *)buf) == 0) {
			fres = call_clause(finC, mk_arr());
			JB_DEPTH = mydepth;
			/* A return/break/continue in finally overrides the try/catch
			 * completion AND swallows a pending throw, exactly like in JS. */
			if (tag_of(fres) == 10) { havePending = 0; result = fres; }
		} else {
			JB_DEPTH = mydepth;
			CALL_DEPTH = calls;
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
	/* The recoverable-error hooks: an emitted closure the module registered once
	 * at boot and never mentions again, so nothing else keeps it alive. */
	gc_try(RT_MISS_VAR, 1);
	gc_try(RT_MISS_MEM, 1);
	gc_try(RT_MISS_SET, 1);
	gc_try(RT_MISS_FN, 1);
	gc_try(RT_GEN_BUSY, 1);
	gc_try(RT_DEEP, 1);
	gc_try(FN_SRC_TEXT, 1);
	gc_try(FN_SRC_TAB, 1);
	gc_try(RETSLOT, 1);
	gc_try(GEN_EXIT, 1);
	if (NIC_YOUNG != 0) { while (i < 1281) { gc_try(NIC[i], 1); i = i + 1; } }
	i = 0;
	if (SMC_YOUNG != 0) { while (i < 8192) { gc_try(SMC_VAL[i], 1); i = i + 1; } }
	i = 0;
	while (i < PIN_N) { gc_try(PINS[i], 1); i = i + 1; }
	/* The permanent region (see PERM_LO): each cell by its tag, and the raw
	 * half word by word, as gc_trace scans a raw buffer. */
	i = PERM_LO;
	while (i < PERM_CTOP) { gc_trace_cell(i); i = i + 64; }
	gc_scan_range(PERM_RBOT, PERM_HI);
	i = 0;
	gc_scan_jb(JBP, JB_DEPTH);
	/* Every SUSPENDED coroutine stack, the registers it parked with, and the
	 * try barriers it parked at. The generator cell itself is a root while its
	 * coroutine exists: the body's frames refer to it, and a generator the
	 * program has dropped still owns its parked stack - the same cost
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
	/* A permanent object is never collected, so it needs no slot. */
	if (is_perm(v)) { return v; }
	if (PIN_N >= 8192) { die("too many pinned globals"); }
	PINS[PIN_N] = v;
	PIN_N = PIN_N + 1;
	return v;
}

/* js_gc_perm(1) opens and js_gc_perm(0) closes a bracket inside which every
 * allocation is permanent (see PERM_LO). A -rt-lib library's jsrtlib_boot
 * brackets its top level with it. No-op in the Go half (abnf/jsrt.go). */
long js_gc_perm(long on) {
	if (on != 0) { PERM_ON = PERM_ON + 1; } else { PERM_ON = PERM_ON - 1; }
	return 0;
}

/* The recoverable-error hooks. An emitter that wants a name with no binding
 * (or a member read on null/undefined) to be a CATCHABLE exception registers a
 * closure here in its boot block; one that wants the abort registers nothing.
 * See the note at RT_MISS_VAR for why the closure answers the value instead of
 * throwing it. The twins are abnf/jsrt.go's js_rt_missvar / js_rt_missmem. */
long js_rt_missvar(long clo) { RT_MISS_VAR = clo; return 0; }
long js_rt_missmem(long clo) { RT_MISS_MEM = clo; return 0; }
long js_rt_missset(long clo) { RT_MISS_SET = clo; return 0; }
long js_rt_missfn(long clo) { RT_MISS_FN = clo; return 0; }
long js_rt_genbusy(long clo) { RT_GEN_BUSY = clo; return 0; }

/* js_str_mem is the string-literal cache; str_intern, next to mk_str above, is
 * the whole of it. */
long js_str_mem(const char *p, long n) { return str_intern(p, n); }

/* Small-integer cache. js_num_i is how EVERY numeric literal in the emitted IR
 * becomes a value, and layer 2 is full of them - operator indices, 0, 1, 2, the
 * bit widths - so a fresh 56-byte cell per evaluation would be a large share of
 * all allocation. A number cell is written once, in mk_num, and never mutated
 * (no other sa/sb/... touches a tag-3
 * cell), and every number comparison in the runtime is by BIT PATTERN, never by
 * handle identity - so one shared cell per small integer is indistinguishable
 * from a fresh one. -256..1024 covers the literals a compiled module emits. */
long js_num_i(long v) {
	long h;
	if (v >= -256 && v <= 1024) {
		h = NIC[v + 256];
		if (h != 0) { return h; }
		return nic_fill(v, d_from_long(v));
	}
	return mk_num(d_from_long(v));
}
long js_num_str(long h) { return mk_num(str_to_num(h)); }
long js_obj_new(void)   { return mk_obj(); }
long js_arr_new(void)   { return mk_arr(); }
long js_arr_push(long a, long v) { arr_push(a, v); return 0; }
long js_arg(long args, long i)   { if (i < arr_len(args)) { return arr_get(args, i); } return H_UNDEF; }
/* A variadic parameter (Go's ...T, Python's *args, Kotlin's vararg): the call
 * arguments from index `from` (a raw integer) on, as a new array. Twins of
 * js_gorest and js_pyrest in abnf/jsrt.go. */
long js_gorest(long args, long from) {
	long out = mk_arr();
	long n;
	long i = from;
	if (tag_of(args) != 5) { return out; }
	n = arr_len(args);
	while (i < n) { arr_push(out, arr_get(args, i)); i = i + 1; }
	return out;
}
long js_pyrest(long args, long from) { return js_gorest(args, from); }

long js_get(long o, long k)          { return get_member(o, k); }
long js_set(long o, long k, long v)  { set_member(o, k, v); return 0; }

long js_closure(long idx, long env) { long h = cell_new(7); sa(h, idx); sb(h, env); return h; }

/* A function's `name` and `length`, kept ON the function cell: e is the name
 * string handle (0 = never set, read as ""), f the length as a raw long. Fields
 * e and f are unused by every function kind (7 closure, 8 host function,
 * 9 bound method, 16 generator function), so setting them is O(1) and the pair
 * dies with the function. Any other value is passed through untouched.
 * js_jsfnmeta is what the js/ts emitters wrap a closure in at its creation
 * site; layer 2 reaches the same storage through fnMeta / fnMetaSet (host ids
 * 94/95). Twin: jsvFnMetaSet in abnf/jsrtjsprint.go. */
int fn_kind(long h) {
	long t = tag_of(h);
	return t == 7 || t == 8 || t == 9 || t == 16;
}
long js_jsfnmeta(long f, long nm, long ln) {
	if (fn_kind(f)) { se(f, nm); sf(f, d_to_long(d_trunc(to_number(ln)))); }
	return f;
}
long fn_meta_get(long f, long key) {
	if (!fn_kind(f)) { return H_UNDEF; }
	if (str_eq_c(key, "name")) { if (fe(f) == 0) { return mk_cstr(""); } return fe(f); }
	if (str_eq_c(key, "length")) { return mk_num(d_from_long(ff(f))); }
	/* One above the function index whose source text f shows, 0 for none; undefined
	 * while no program registered any (fn_src_idx). */
	if (str_eq_c(key, "src")) { if (FN_SRC_TAB == 0) { return H_UNDEF; } return mk_num(d_from_long(fn_src_idx(f) + 1)); }
	return H_UNDEF;
}

/* The source text of a js program's functions, which String(f) answers (ES2023
 * 20.2.3.5). The module registers it once (js_jssrc): the text of its files and a
 * table of "idx,from,to;" entries behind the highest index and a ';', from/to in
 * UTF-16 units of that text, one entry per function index whose definition has a
 * span. It is parsed on the first String(f). A closure shows its own index (fa); a
 * value that is no plain closure was linked to one by js_jsfnsrc, which keeps it
 * one above in the otherwise unused field d. Only a js module registers a table, so
 * every other language keeps "[function]". Twins: jsvFnText and the two externs in
 * abnf/jsrtjsprint.go. */
long js_jssrc(long text, long tab) { FN_SRC_TEXT = text; FN_SRC_TAB = tab; FN_SRC_SE = 0; return H_UNDEF; }
long js_jsfnsrc(long f, long idx) { if (fn_kind(f)) { sd(f, d_to_long(d_trunc(to_number(idx))) + 1); } return f; }
void fn_src_parse(void) {
	const char *p = str_ptr(FN_SRC_TAB);
	long n = str_len(FN_SRC_TAB);
	long i = 0;
	long v = 0;
	long k = 0;
	long f[3];
	long *se;
	while (i < n && p[i] != 59) { v = v * 10 + ((int)p[i] & 255) - 48; i = i + 1; }
	FN_SRC_N = v + 1;
	FN_SRC_SE = (long)malloc(FN_SRC_N * 16);
	se = (long *)FN_SRC_SE;
	k = 0;
	while (k < FN_SRC_N * 2) { se[k] = -1; k = k + 1; }
	i = i + 1;
	v = 0;
	k = 0;
	while (i < n) {
		int c = (int)p[i] & 255;
		if (c == 44 || c == 59) {
			f[k] = v;
			k = k + 1;
			v = 0;
			if (c == 59) {
				if (k == 3 && f[0] >= 0 && f[0] < FN_SRC_N) { se[2 * f[0]] = f[1]; se[2 * f[0] + 1] = f[2]; }
				k = 0;
			}
		} else { v = v * 10 + c - 48; }
		i = i + 1;
	}
}
/* The function index whose source text f shows, or -1. */
long fn_src_idx(long f) {
	long t = tag_of(f);
	long idx = -1;
	long *se;
	if (fd(f) != 0) { idx = fd(f) - 1; }
	else if (t == 7) { idx = fa(f); }
	else if (t == 16) { return fn_src_idx(fa(f)); }
	if (FN_SRC_SE == 0) { fn_src_parse(); }
	se = (long *)FN_SRC_SE;
	if (idx < 0 || idx >= FN_SRC_N) { return -1; }
	if (se[2 * idx] < 0) { return -1; }
	return idx;
}
/* String(f): the source text of its definition, or V8's native code for a function
 * that has none - a builtin, a bound function. */
long fn_text(long h) {
	long idx = fn_src_idx(h);
	long nm = fe(h);
	long *se;
	if (idx >= 0) { se = (long *)FN_SRC_SE; return u_slice(FN_SRC_TEXT, se[2 * idx], se[2 * idx + 1]); }
	if (nm == 0) { nm = mk_cstr(""); }
	if (str_len(nm) >= 6) { if (str_eq_c(str_slice(nm, 0, 6), "bound ")) { nm = mk_cstr(""); } }
	return str_cat(str_cat(mk_cstr("function "), nm), mk_cstr("() { [native code] }"));
}

long js_scope_new(long parent) { return mk_scope(scope_of(parent)); }
long js_scope_get(long s, long name) { return scope_get(scope_of(s), name); }

/* js_scope_cget(cache, env, name bytes, name length): scope_get(env, name) for
 * a name the calling MetaJS function does not declare, so that none of its own
 * scopes can hold it and the answer depends on its closure scope `env` alone.
 * The name is made a string only on a miss. metajs-to-llvm-ir
 * emits one four-word cache per name: c[0] the env it was filled for, c[1] the
 * scope holding the name, c[2] the entry's index there and c[3] env's entry
 * count when the holder is env's PARENT.
 *
 * A hit is exact because entries are never removed or reordered: a holder that
 * IS env keeps the entry at that index, and a holder one level up still wins
 * while env has gained no name since. Only PERMANENT scopes are cached - a
 * layer-2 library's module scope and the root - because a collected scope's
 * address can come back as a different scope, and a permanent one never dies. */
long js_scope_cget(long *c, long env, const char *np, long nn) {
	long s = scope_of(env);
	long name;
	long p;
	long i;
	long *vals;
	if (c[0] == s) {
		if (c[1] == s || fd(s) == c[3]) { vals = (long *)fb(c[1]); return vals[c[2]]; }
	}
	name = str_intern(np, nn);
	if (is_perm(s)) {
		i = scope_find(s, name);
		if (i >= 0) { c[0] = s; c[1] = s; c[2] = i; vals = (long *)fb(s); return vals[i]; }
		p = ff(s);
		if (p != 0 && is_perm(p)) {
			i = scope_find(p, name);
			if (i >= 0) { c[0] = s; c[1] = p; c[2] = i; c[3] = fd(s); vals = (long *)fb(p); return vals[i]; }
		}
	}
	return scope_get(s, name);
}

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

/* js_scope_set_strict: the same store in STRICT JavaScript, where a name that no
 * scope binds is not created but is the ReferenceError a read of it raises - the
 * language's missing-variable hook answers it, as for scope_get. The twin is
 * abnf/jsrt.go's scopeSetStrict. */
long js_scope_set_strict(long s, long name, long v) {
	long cur = scope_of(s);
	long exc;
	while (cur != 0) {
		long i = scope_find(cur, name);
		if (i >= 0) { long *vals = (long *)fb(cur); vals[i] = v; return 0; }
		cur = ff(cur);
	}
	exc = rt_miss_var(name);
	if (exc != 0) { js_throw(exc); }
	die2("assignment to undeclared variable: ", name);
	return 0;
}

/* Is `name` bound in THIS scope - no chain walk. It is the EXTERN twin of the
 * scopeHas BUILTIN (host 67 below, and jsrtint.go's scBindings): a builtin is
 * callable from MetaJS, an extern from emitted IR.
 *
 * js_scope_typeof is the chain walk and cannot express it - it answers
 * "undefined" for an absent name and for a slot holding undefined alike, and it
 * finds a name an ENCLOSING scope binds. */
long js_scope_has(long s, long name) {
	return mk_bool(scope_find(scope_of(s), name) >= 0);
}

/* The value `name` holds in THIS scope, or undefined when this scope does not
 * bind it - no chain walk and no abort. It is the read that pairs with
 * js_scope_has, for a language whose variables are local to one frame (php):
 * one lookup where js_scope_has plus js_scope_get would be two. The twin is
 * abnf/jsrt.go's js_scope_own. */
long js_scope_own(long s, long name) {
	long cur = scope_of(s);
	long i = scope_find(cur, name);
	long *vals;
	if (i < 0) { return H_UNDEF; }
	vals = (long *)fb(cur);
	return vals[i];
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
 * table is quadratic in the number of deletions and never freed, and it
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
 * The jsJFlo arms:
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
/* There is deliberately NO js_gicmp / js_gieq / js_ginot / js_ginum / js_gistr /
 * js_giis: no emitter needs them. languages/go-to-llvm-ir.abnf sends `==` and
 * `!=` to the shared js_seq (strict_eq's tag-13 arm at the top of this file does
 * the width-exact compare), the ordered comparisons to
 * js_gilt/js_gile/js_gigt/js_gige, `^x` to emitBin("^") + js_band at the
 * operand's width, fmt.Sprint to the shared to_string, and float64(x) to
 * js_giconv. If a language ever DOES need one, write it then, with the ratchet
 * assertion that reaches it. */
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
 * There is no js_jfsub / js_jfmul / js_jfmod / js_jfneg / js_jfint: NOTHING
 * EMITS THEM. Java routes -, *, / and % through js_jvarith, unary minus through
 * js_jvneg and (int) through js_jvint / makePrimCast's width-exact narrowing;
 * Go's float operators leave through js_giarith, whose tag-14 arm calls
 * jf_arith. js_jfdiv IS emitted. */
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
 * THE GENERAL RULE: every extern a layer-2 file DEFINES is a name this file may
 * not also define - both modules are handed to clang, so two definitions are
 * `ld: duplicate symbol` (which the metacompiler reports as "1 unresolved
 * symbol(s)", see abnf/linkdiag_test.go). Before moving any js_* body in here,
 * run `grep -l 'function js_<name>' languages/lib/*.metajs` and delete the
 * layer-2 copy (and regenerate its .ll) in the same commit. */
long js_bytelen(long v) { return mk_num(d_from_long(str_len(to_string(v)))); }

/* js_gospread: Go's f(xs...) and C#'s params-array call. Flatten the LAST element
 * of the argument array IN PLACE, so the callee sees the slice's elements as
 * separate arguments; a nil/undefined tail contributes nothing.
 *
 * It is here and not in the shared MetaJS layer because it has ONE
 * specification across Go and C# (the Go twin's "js_gospread" in abnf/jsrt.go
 * is the authority). Its neighbours js_has and js_goslice must STAY per
 * language, because Go counts a string in BYTES and C# in UTF-16 code units.
 *
 * It does NOT upcall, and may not: tests/coro-poc/gen.ll links the floor with
 * NO layer 2 at all. go-rt.metajs and csharp-rt.metajs must not define it (see
 * the rule at js_bytelen above). */
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
/* n is the RAW index of the goto's label (compile-core's ctlLabels), not a handle. */
long js_ctl_goto(long n)   { return mk_ctl(4 + n, H_UNDEF); }
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
	CALL_MAX = 9223372036854775807;
	JB_MAIN = (long)malloc(8 * 512);
	JB_CAP = 512;
	jbi = 0;
	while (jbi < 512) { long *jp = (long *)JB_MAIN; jp[jbi] = 0; jbi = jbi + 1; }
	JBP = JB_MAIN;
	ar_init();
	/* Everything boot() seeds lives for the whole run. */
	PERM_ON = 1;
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
	TY_UNDEF = mk_cstr("undefined");
	TY_BOOL = mk_cstr("boolean");
	TY_NUM = mk_cstr("number");
	TY_STR = mk_cstr("string");
	TY_FN = mk_cstr("function");
	TY_OBJ = mk_cstr("object");

	seed_root("println", mk_host(1));
	seed_root("print", mk_host(2));
	seed_root("eprintln", mk_host(3));
	seed_root("parseInt", mk_host(4));
	seed_root("parseFloat", mk_host(5));
	seed_root("exit", mk_host(6));
	seed_root("byteLen", mk_host(32));
	seed_root("unicodeCase", mk_host(97));
	seed_root("writeBytes", mk_host(98));
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
	seed_root("fnMeta", mk_host(94));
	seed_root("fnMetaSet", mk_host(95));
	seed_root("arrKind", mk_host(99));
	seed_root("arrKindSet", mk_host(100));
	seed_root("arrProps", mk_host(106));
	seed_root("arrPropsSet", mk_host(107));
	seed_root("byteAt", mk_host(101));
	seed_root("byteSub", mk_host(102));
	seed_root("bytesStr", mk_host(103));
	seed_root("osQuery", mk_host(104));
	seed_root("readIn", mk_host(105));
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
	/* THE MEMBERS OF Math ARE PINNED IN THREE PLACES: this list,
	 * standardJSBindings in abnf/jsrt.go (the Go twin, llvm.Run) and the grammar
	 * host's own Math that languages/metajs-interpreter.abnf binds.
	 * abnf/hostglobals_test.go's TestMathMembersAgree reads all three out of
	 * their own sources and fails if one of them gains or loses a member. */
	seed_host(math, "sin", 71);
	seed_host(math, "cos", 72);
	seed_host(math, "tan", 73);
	seed_host(math, "asin", 74);
	seed_host(math, "acos", 75);
	seed_host(math, "atan", 76);
	seed_host(math, "sinh", 77);
	seed_host(math, "cosh", 78);
	seed_host(math, "tanh", 79);
	seed_host(math, "asinh", 80);
	seed_host(math, "acosh", 81);
	seed_host(math, "atanh", 82);
	seed_host(math, "exp", 83);
	seed_host(math, "expm1", 84);
	seed_host(math, "log", 85);
	seed_host(math, "log1p", 86);
	seed_host(math, "log2", 87);
	seed_host(math, "log10", 88);
	seed_host(math, "cbrt", 89);
	seed_host(math, "atan2", 90);
	seed_host(math, "hypot", 91);
	seed_host(math, "clz32", 92);
	seed_host(math, "fround", 93);
	seed_host(math, "random", 96);
	{
		union DB p;
		p.d = 3.141592653589793;
		obj_put(math, mk_cstr("PI"), mk_num(p.l));
		p.d = 2.718281828459045;
		obj_put(math, mk_cstr("E"), mk_num(p.l));
		p.d = 0.6931471805599453;
		obj_put(math, mk_cstr("LN2"), mk_num(p.l));
		p.d = 2.302585092994046;
		obj_put(math, mk_cstr("LN10"), mk_num(p.l));
		p.d = 1.4426950408889634;
		obj_put(math, mk_cstr("LOG2E"), mk_num(p.l));
		p.d = 0.4342944819032518;
		obj_put(math, mk_cstr("LOG10E"), mk_num(p.l));
		p.d = 1.4142135623730951;
		obj_put(math, mk_cstr("SQRT2"), mk_num(p.l));
		p.d = 0.7071067811865476;
		obj_put(math, mk_cstr("SQRT1_2"), mk_num(p.l));
	}
	seed_root("Math", math);

	strobj = mk_obj();
	seed_host(strobj, "fromCharCode", 30);
	seed_root("String", strobj);

	arrobj = mk_obj();
	seed_host(arrobj, "isArray", 31);
	seed_root("Array", arrobj);

	/* The collector's trigger, made explicit and testable rather than implicit.
	 *   MEC_GC=off      never collect (for measurement)
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
	PERM_ON = 0;
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
		long s = exc_text(THROWN);
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
		o_str(to_string(mk_num(d_from_long(GC_HEAP + perm_used()))));
		o_cstr(" perm=");
		o_str(to_string(mk_num(d_from_long(perm_used()))));
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

/* js_realpath(p): the canonical absolute path of an existing file, with ".",
 * ".." and symbolic links resolved against the working directory, or false
 * when it does not resolve. An empty path is the working directory; a path
 * holding a NUL byte resolves to nothing. access(2) runs first because the
 * kernel's lookup demands a DIRECTORY before every further component and a
 * trailing slash, which macOS's realpath(3) does not ("file/" and "file/.."
 * resolve there). The answer is copied with mk_str, never str_intern: the
 * buffer realpath(3) returns is freed here, and str_intern caches by POINTER.
 * Twin of realPath in abnf/jsrtphp.go. */
int access(const char *path, int mode);
char *realpath(const char *path, char *resolved);
long js_realpath(long p) {
	long s = to_string(p);
	long n = str_len(s);
	const char *src = str_ptr(s);
	char *buf = (char *)malloc(n + 2);
	char *r = 0;
	long m = 0;
	long h;
	long i = 0;
	while (i < n) {
		if (src[i] == 0) { free(buf); return H_FALSE; }
		buf[i] = src[i];
		i = i + 1;
	}
	buf[n] = 0;
	if (n == 0) { buf[0] = 46; buf[1] = 0; }
	if (access(buf, 0) == 0) { r = realpath(buf, 0); }
	free(buf);
	if (r == 0) { return H_FALSE; }
	while (r[m] != 0) { m = m + 1; }
	h = mk_str(r, m);
	free(r);
	return h;
}
