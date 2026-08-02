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
 *   - It is not a garbage collector. Memory comes from a bump arena and is never
 *     freed; the process exits and the OS reclaims it. Accepted limit, recorded
 *     in the plan under Risks.
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
int setjmp(void *env);
void longjmp(void *env, int v);
void exit(int code);

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
long d_mod(long x, long y) {
	long q; long t;
	if (d_is_nan(x) || d_is_nan(y) || d_is_inf(x) || d_is_zero(y)) { return DNAN; }
	if (d_is_inf(y)) { return x; }
	if (d_is_zero(x)) { return x; }
	q = d_div(x, y);
	t = d_trunc(q);
	return d_sub(x, d_mul(t, y));
}

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
	long t; long v;
	if (d_is_nan(bits) || d_is_inf(bits)) { return 0; }
	t = d_trunc(bits);
	if (d_in_long(t) == 0) { return 0; }
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

/* ------------------------------------------------------------- the arena */

char *ar_base;
long ar_off;
long ar_cap;

void ar_init(void) { ar_cap = 1048576; ar_base = (char *)malloc(1048576); ar_off = 0; }

char *ar_alloc(long n) {
	char *p;
	n = (n + 15) & (0 - 16);          /* 16 byte aligned, like malloc */
	if (ar_off + n > ar_cap) {
		long want = 1048576;
		while (want < n) { want = want * 2; }
		ar_base = (char *)malloc(want);
		ar_cap = want;
		ar_off = 0;
	}
	p = ar_base + ar_off;
	ar_off = ar_off + n;
	return p;
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
	struct Cell *p = (struct Cell *)ar_alloc(56);
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

/* --- fatal errors ------------------------------------------------------- */

void wr(const char *s) {
	long i = 0;
	while (s[i] != 0) { putchar((int)s[i]); i = i + 1; }
}
void wrn(const char *s, long n) {
	long i = 0;
	while (i < n) { putchar((int)s[i]); i = i + 1; }
}
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

long mk_num(long bits) {
	long h = cell_new(3);
	sa(h, bits);
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
	long n = str_len(x);
	long i = 0;
	const char *px;
	const char *py;
	if (n != str_len(y)) { return 0; }
	px = str_ptr(x); py = str_ptr(y);
	while (i < n) { if (px[i] != py[i]) { return 0; } i = i + 1; }
	return 1;
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

long mk_cstr(const char *s) {
	long n = 0;
	while (s[n] != 0) { n = n + 1; }
	return mk_str(s, n);
}

/* --- growable handle buffers -------------------------------------------- */

/* A buffer is a raw long* in the arena; the owning cell carries n and cap. */
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

/* Digits of |x| into buf (17 of them) plus the decimal exponent of the first
 * digit, returned. x must be finite and non-zero. */
long g_ndig;
long dtoa_digits(long x, char *buf) {
	union DB m;
	union DB ten;
	long e10 = 0;
	long i = 0;
	long dg;
	x = d_abs(x);
	m.l = x;
	ten.d = 10.0;
	while (m.d >= 10.0) { m.d = m.d / ten.d; e10 = e10 + 1; }
	while (m.d < 1.0)   { m.d = m.d * ten.d; e10 = e10 - 1; }
	while (i < 17) {
		dg = (long)m.d;
		if (dg > 9) { dg = 9; }
		if (dg < 0) { dg = 0; }
		buf[i] = (char)(48 + dg);
		m.d = (m.d - (double)dg) * ten.d;
		i = i + 1;
	}
	g_ndig = 17;
	return e10;
}

/* Round the 17 digits in buf to k digits (in place); returns 1 when the
 * rounding carried past the first digit, which shifts the exponent. */
int dtoa_round(char *buf, long k) {
	long i;
	int carry;
	if (k >= 17) { return 0; }
	carry = buf[k] >= 53;      /* '5' */
	i = k - 1;
	while (carry && i >= 0) {
		if (buf[i] == 57) { buf[i] = 48; i = i - 1; }
		else { buf[i] = (char)(buf[i] + 1); carry = 0; }
	}
	i = k;
	while (i < 17) { buf[i] = 48; i = i + 1; }
	if (carry) {
		i = 16;
		while (i > 0) { buf[i] = buf[i - 1]; i = i - 1; }
		buf[0] = 49;           /* '1' */
		return 1;
	}
	return 0;
}

/* Rebuild the double from k digits at exponent e10. */
long dtoa_value(const char *buf, long k, long e10) {
	long mant = 0;
	long i = 0;
	union DB r;
	union DB p;
	long e = e10 - k + 1;
	while (i < k) { mant = mant * 10 + (buf[i] - 48); i = i + 1; }
	r.l = d_from_long(mant);
	/* DIVIDE by a positive power of ten rather than multiplying by its
	 * reciprocal: 35 / 10 is exactly 3.5, while 35 * (1/10) is not. */
	if (e < 0) { p.l = pow10(0 - e); r.d = r.d / p.d; }
	else { p.l = pow10(e); r.d = r.d * p.d; }
	return r.l;
}

/* shortest_digits fills digs with the SHORTEST decimal digit string that
 * reproduces |bits| exactly (17 digits always do), sets g_ndig to how many, and
 * returns the decimal exponent of the first digit. bits must be finite and
 * non-zero - it is the shared core of both renderings below. */
long shortest_digits(long bits, char *digs) {
	long e10 = dtoa_digits(bits, digs);
	long best = 17;
	long k = 1;
	long i;
	while (k <= 17) {
		char tryb[24];
		long te;
		i = 0;
		while (i < 17) { tryb[i] = digs[i]; i = i + 1; }
		te = e10;
		if (dtoa_round(tryb, k)) { te = te + 1; }
		if (dtoa_value(tryb, k, te) == d_abs(bits)) {
			i = 0;
			while (i < 17) { digs[i] = tryb[i]; i = i + 1; }
			e10 = te;
			best = k;
			k = 18;
		} else {
			k = k + 1;
		}
	}
	while (best > 1 && digs[best - 1] == 48) { best = best - 1; }
	g_ndig = best;
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
	 * whole integer arithmetic of a program takes. */
	if (d_is_integral(bits) && d_in_long(bits)) {
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
	while (i < j && s[i] >= 48 && s[i] <= 57) {
		if (mdig < 18) { mant = mant * 10 + (s[i] - 48); mdig = mdig + 1; }
		else { exp10 = exp10 + 1; }
		digits = digits + 1;
		i = i + 1;
	}
	if (i < j && s[i] == 46) {
		i = i + 1;
		while (i < j && s[i] >= 48 && s[i] <= 57) {
			if (mdig < 18) { mant = mant * 10 + (s[i] - 48); mdig = mdig + 1; exp10 = exp10 - 1; }
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
	if (exp10 > 308) { return neg ? DNINF : DINF; }
	if (exp10 < -340) { return neg ? (0 - 9223372036854775807 - 1) : 0; }
	r.l = d_from_long(mant);
	if (exp10 < 0) { p.l = pow10(0 - exp10); r.d = r.d / p.d; }
	else { p.l = pow10(exp10); r.d = r.d * p.d; }
	if (neg) { r.d = 0.0 - r.d; }
	return r.l;
}

/* -------------------------------------------------- the value operations - */

long to_string(long h);
long to_number(long h);
int truthy(long h);
long js_call(long callee, long self, long args);
long type_of(long h);

long to_number(long h) {
	long t = tag_of(h);
	if (t == 3) { return num_bits(h); }
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
	if (t == 4) { return str_len(h) > 0; }
	return 1;
}

long to_string(long h) {
	long t = tag_of(h);
	if (t == 0) { return mk_cstr("undefined"); }
	if (t == 1) { return mk_cstr("null"); }
	if (t == 2) { return fa(h) ? mk_cstr("true") : mk_cstr("false"); }
	if (t == 3) { return num_to_str(num_bits(h)); }
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
	if (t == 7 || t == 8 || t == 9) { return mk_cstr("[function]"); }
	if (t == 11) { return mk_cstr("[scope]"); }
	return mk_cstr("[value]");
}

long type_of(long h) {
	long t = tag_of(h);
	if (t == 0) { return mk_cstr("undefined"); }
	if (t == 2) { return mk_cstr("boolean"); }
	if (t == 3) { return mk_cstr("number"); }
	if (t == 4) { return mk_cstr("string"); }
	if (t == 7 || t == 8 || t == 9) { return mk_cstr("function"); }
	return mk_cstr("object");
}

/* The type CLASS used by the MetaJS type pin: 0 = none (undefined / null),
 * else a small code per typeof class. */
long type_class(long h) {
	long t = tag_of(h);
	if (t == 0 || t == 1) { return 0; }
	if (t == 2) { return 2; }
	if (t == 3) { return 3; }
	if (t == 4) { return 4; }
	if (t == 7 || t == 8 || t == 9) { return 6; }
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

/* ---------------------------------------------------------------- scopes - */

/* A scope cell: a = names buffer (string handles), b = values buffer,
 * c = type-class buffer, d = count, e = capacity, f = parent handle (0 at the
 * root). Handle 0 as a scope argument means the ROOT scope, matching
 * abnf/jsrt.go's scopeOf. */
long G_ROOT;

long mk_scope(long parent) {
	long h = cell_new(11);
	sa(h, (long)buf_new(8));
	sb(h, (long)buf_new(8));
	sc(h, (long)buf_new(8));
	sd(h, 0);
	se(h, 8);
	sf(h, parent);
	return h;
}
long scope_of(long h) { if (h == 0) { return G_ROOT; } return h; }

long scope_find(long s, long name) {
	long *names = (long *)fa(s);
	long n = fd(s);
	long i = 0;
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
		if (nc < 8) { nc = 8; }
		names = buf_grow(names, n, nc);
		vals = buf_grow(vals, n, nc);
		tcs = buf_grow(tcs, n, nc);
		sa(s, (long)names);
		sb(s, (long)vals);
		sc(s, (long)tcs);
		se(s, nc);
	}
	names[n] = name;
	vals[n] = v;
	tcs[n] = 0;
	sd(s, n + 1);
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
	if (str_eq(name, mk_cstr("push")))    { return 1; }
	if (str_eq(name, mk_cstr("pop")))     { return 2; }
	if (str_eq(name, mk_cstr("shift")))   { return 3; }
	if (str_eq(name, mk_cstr("unshift"))) { return 4; }
	if (str_eq(name, mk_cstr("reverse"))) { return 5; }
	if (str_eq(name, mk_cstr("slice")))   { return 6; }
	if (str_eq(name, mk_cstr("indexOf"))) { return 7; }
	if (str_eq(name, mk_cstr("join")))    { return 8; }
	if (str_eq(name, mk_cstr("concat")))  { return 9; }
	return 0;
}
long method_id_string(long name) {
	if (str_eq(name, mk_cstr("charCodeAt")))  { return 20; }
	if (str_eq(name, mk_cstr("charAt")))      { return 21; }
	if (str_eq(name, mk_cstr("indexOf")))     { return 22; }
	if (str_eq(name, mk_cstr("replace")))     { return 23; }
	if (str_eq(name, mk_cstr("slice")))       { return 24; }
	if (str_eq(name, mk_cstr("substring")))   { return 25; }
	if (str_eq(name, mk_cstr("split")))       { return 26; }
	if (str_eq(name, mk_cstr("toUpperCase"))) { return 27; }
	if (str_eq(name, mk_cstr("toLowerCase"))) { return 28; }
	if (str_eq(name, mk_cstr("trim")))        { return 29; }
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

int is_callable(long h) { long t = tag_of(h); return t == 7 || t == 8 || t == 9; }

long get_member(long obj, long key) {
	long t = tag_of(obj);
	long name;
	long idx;
	if (t == 0 || t == 1) { die3("member '", key, "' of ", obj); }
	if (tag_of(key) == 4 && is_callable(obj)) {
		if (str_eq(key, mk_cstr("apply"))) { return mk_bound(obj, 40); }
		if (str_eq(key, mk_cstr("call")))  { return mk_bound(obj, 41); }
	}
	if (t == 6) { return obj_get(obj, to_string(key)); }
	if (t == 5) {
		if (tag_of(key) == 4) {
			long mid;
			if (str_eq(key, mk_cstr("length"))) { return mk_num(d_from_long(arr_len(obj))); }
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
			if (str_eq(key, mk_cstr("length"))) { return mk_num(d_from_long(str_ulen(obj))); }
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
		if (tag_of(key) == 4 && str_eq(key, mk_cstr("length"))) {
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

long JB[512];
long JB_DEPTH;
long THROWN;

long js_throw(long v);

/* ------------------------------------------------------------- the calls - */

long arg_at(long args, long i) { if (i < arr_len(args)) { return arr_get(args, i); } return H_UNDEF; }
long arg_num(long args, long i) { if (i < arr_len(args)) { return to_number(arr_get(args, i)); } return DNAN; }
long arg_str(long args, long i) { if (i < arr_len(args)) { return to_string(arr_get(args, i)); } return mk_cstr(""); }

long host_call(long id, long self, long args);
long builtin_method(long recv, long mid, long args);

long js_call(long callee, long self, long args) {
	long t = tag_of(callee);
	if (t == 7) { return jsdispatch(fa(callee), fb(callee), args); }
	if (t == 8) { return host_call(fa(callee), self, args); }
	if (t == 9) { return builtin_method(fa(callee), fb(callee), args); }
	die2("call of a non function value: ", callee);
	return H_UNDEF;
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
			long bn = str_len(recv);
			char *buf = ar_alloc(bn + 1);
			const char *p = str_ptr(recv);
			long i = 0;
			n = bn;
			while (i < n) {
				int c = (int)p[i] & 255;
				if (mid == 27 && c >= 97 && c <= 122) { c = c - 32; }
				if (mid == 28 && c >= 65 && c <= 90) { c = c + 32; }
				buf[i] = (char)c;
				i = i + 1;
			}
			buf[n] = 0;
			return mk_str(buf, n);
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

long d_sqrt(long x) {
	union DB a;
	union DB g;
	union DB two;
	long i = 0;
	if (d_is_nan(x) || d_sign(x)) { if (d_is_zero(x)) { return x; } return DNAN; }
	if (d_is_inf(x) || d_is_zero(x)) { return x; }
	a.l = x;
	g.d = a.d;
	two.d = 2.0;
	while (i < 80) { g.d = (g.d + a.d / g.d) / two.d; i = i + 1; }
	return g.l;
}
/* Integer exponents only - the cases a MetaJS program actually uses. */
long d_pow(long x, long y) {
	union DB r;
	union DB b;
	long n;
	long neg = 0;
	long i = 0;
	if (d_is_nan(y)) { return DNAN; }
	if (d_is_zero(y)) { return DONE; }
	if (!d_is_integral(y) || !d_in_long(y)) {
		if (d_is_nan(x) || d_sign(x)) { return DNAN; }
		return DNAN;                       /* not modelled; see the header */
	}
	n = d_to_long(y);
	if (n < 0) { neg = 1; n = 0 - n; }
	b.l = x;
	r.d = 1.0;
	while (i < n) { r.d = r.d * b.d; i = i + 1; }
	if (neg) { r.d = 1.0 / r.d; }
	return r.l;
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
	longjmp((void *)JB[JB_DEPTH - 1], 1);
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
	buf = (long)malloc(512);
	JB[JB_DEPTH] = buf;
	JB_DEPTH = JB_DEPTH + 1;
	if (setjmp((void *)buf) == 0) {
		result = js_call(tryC, H_UNDEF, mk_arr());
		JB_DEPTH = mydepth;
	} else {
		JB_DEPTH = mydepth;
		pending = THROWN;
		havePending = 1;
	}
	if (havePending && hasCatch) {
		long a = mk_arr();
		arr_push(a, pending);
		havePending = 0;
		buf = (long)malloc(512);
		JB[JB_DEPTH] = buf;
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
		buf = (long)malloc(512);
		JB[JB_DEPTH] = buf;
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

/* js_str_mem is called once per evaluation of every string literal, so the
 * (pointer, length) pair is cached - the same thing abnf/jsrt.go's strMemCache
 * does. Direct-mapped, and a miss just re-creates the string. */
long SMC_PTR[1024];
long SMC_LEN[1024];
long SMC_VAL[1024];

long js_str_mem(const char *p, long n) {
	long slot = ((long)p >> 3) & 1023;
	long h;
	if (SMC_PTR[slot] == (long)p && SMC_LEN[slot] == n) { return SMC_VAL[slot]; }
	h = mk_str(p, n);
	SMC_PTR[slot] = (long)p;
	SMC_LEN[slot] = n;
	SMC_VAL[slot] = h;
	return h;
}

long js_num_i(long v)   { return mk_num(d_from_long(v)); }
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
}

int main(void) {
	long r;
	long buf;
	boot();
	buf = (long)malloc(512);
	JB[0] = buf;
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
	{
		long n = to_number(r);
		if (d_is_nan(n)) { return 0; }
		return (int)to_int32(n);
	}
}
