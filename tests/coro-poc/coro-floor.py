#!/usr/bin/env python3
"""coro-floor.py - apply the COROUTINE proof-of-concept to a COPY of the C floor.

It never touches languages/lib/runtime.c: it reads that file (or, with --head,
`git show HEAD:languages/lib/runtime.c`) and writes the patched text to stdout or
to the path given with -o.  `tests/coro-poc/build.sh` drives it; the unified diff
it produces is what docs/runtime-next-plan.md quotes for the floor's owner.

Every edit is anchored on an exact line of runtime.c, and a missing anchor is a
hard error rather than a silent no-op - the whole point is that the patch cannot
quietly apply to nothing.
"""
import subprocess
import sys

# --------------------------------------------------------------------------
# 1. libc prototypes.  Threads and dl are declared by hand, exactly as putchar /
#    malloc / setjmp already are: runtime.c has no #include and c-to-llvm-ir.abnf
#    has no headers.  Every argument is a pointer or a long, so NO STRUCT LAYOUT
#    is needed anywhere - that is what rules ucontext out (see the plan).
P_PROTO_ANCHOR = "void exit(int code);"
P_PROTO = """void exit(int code);

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
long pthread_cond_broadcast(void *c);"""

# --------------------------------------------------------------------------
# 2. The registry globals, next to GC_STACK_BASE because gc_roots reads them.
P_GLOB_ANCHOR = "long GC_STACK_BASE;"
P_GLOB = """long GC_STACK_BASE;

/* ---- the coroutine registry, which is what the COLLECTOR needs ------------
 * A suspended generator's C stack holds handles that live nowhere else, so it
 * is a root range exactly like the main stack.  Handoff is strictly alternating
 * (one thread runs at a time), so a suspended side has always published its
 * stack pointer and a setjmp of its callee-saved registers BEFORE parking, and
 * the collector reads both without any stop-the-world machinery.
 *
 *   CORO[i]   a live coroutine's control block (see coro_alloc for the layout)
 *   RES_*     the RESUMER's stack while a coroutine runs.  Resumes nest (a
 *             generator may drive another), so this is a LIFO. */
long CORO[256];
long CORO_N;
long RES_LO[64];
long RES_HI[64];
long RES_JB[64];
long RES_N;
long CUR_GEN;          /* the generator whose body is running, for js_yield */
long CORO_ENTRY;       /* &coro_entry, from dlsym; 0 until the first generator */"""

# --------------------------------------------------------------------------
# 3. gc_trace: the two new tags.  The cell's fields are at a different WORD
#    OFFSET depending on whether a block still carries the 16 byte header, so
#    the anchor is picked from the tag-11 line that is already there rather than
#    hard-coded - a wrong offset here would be a collector that writes mark words
#    into unrelated memory, which is the one failure mode runtime.c's gc_try was
#    written to make impossible.
P_TRACE_VARIANTS = [
    # working tree 2026-08-03: no header, so the cell IS the block (a = w[1]).
    ("	if (t == 11) { gc_try(w[1], 1); gc_try(w[6], 1); return; }  /* names+vals, parent */", 1),
    # 1cd6a41: a 16 byte header in front, so a = w[3].
    ("	if (t == 11) { gc_try(w[3], 1); gc_try(w[8], 1); return; }  /* names+vals, parent */", 3),
]


def trace_edit(text):
    for anchor, base in P_TRACE_VARIANTS:
        if text.count(anchor) == 1:
            return anchor, anchor + """
	/* 15 GENERATOR: a fn, b args, d lastValue, f sent/return value are handles;
	 * c is the RAW control-block pointer and e a raw flag word.  16 generator
	 * FUNCTION: a is the closure it wraps. */
	if (t == 15) { gc_try(w[%d], 1); gc_try(w[%d], 1); gc_try(w[%d], 1); gc_try(w[%d], 1); return; }
	if (t == 16) { gc_try(w[%d], 1); return; }""" % (
                base, base + 1, base + 3, base + 5, base)
    return None, None

# --------------------------------------------------------------------------
# 4. gc_roots: scan every suspended stack.
P_ROOTS_ANCHOR = """	while (i < JB_DEPTH) { if (JB[i] != 0) { gc_scan_range(JB[i], JB[i] + 512); } i = i + 1; }
}"""
P_ROOTS = """	while (i < JB_DEPTH) { if (JB[i] != 0) { gc_scan_range(JB[i], JB[i] + 512); } i = i + 1; }
	/* Every SUSPENDED coroutine stack, and the registers it parked with.  The
	 * generator cell itself is a root while its coroutine exists: the body's
	 * frames refer to it, and a generator the program has dropped still owns a
	 * parked thread - the same cost abnf/jsrt.go's parked goroutine has. */
	i = 0;
	while (i < CORO_N) {
		long *cb = (long *)CORO[i];
		gc_try(cb[7], 1);
		if (cb[1] == 1) {
			gc_scan_range(cb[4], cb[5]);
			gc_scan_range(cb[6], cb[6] + 512);
		}
		i = i + 1;
	}
	/* and the RESUMER stacks, parked while a coroutine runs. */
	i = 0;
	while (i < RES_N) {
		gc_scan_range(RES_LO[i], RES_HI[i]);
		gc_scan_range(RES_JB[i], RES_JB[i] + 512);
		i = i + 1;
	}
	gc_try(CUR_GEN, 1);
}"""

# --------------------------------------------------------------------------
# 5. is_callable knows tag 16.
P_CALLABLE_ANCHOR = "int is_callable(long h) { long t = tag_of(h); return t == 7 || t == 8 || t == 9; }"
P_CALLABLE = "int is_callable(long h) { long t = tag_of(h); return t == 7 || t == 8 || t == 9 || t == 16; }"

# --------------------------------------------------------------------------
# 6. get_member: a generator's `next`.
P_MEMBER_ANCHOR = "	if (t == 6) { return obj_get(obj, to_string(key)); }"
P_MEMBER = """	if (t == 6) { return obj_get(obj, to_string(key)); }
	/* A generator's iterator protocol.  The PoC implements `next` only; current
	 * / key / valid / send / getReturn are the same three lines each over the
	 * same handshake (see the cost table in docs/runtime-next-plan.md). */
	if (t == 15 && tag_of(key) == 4 && str_eq_c(key, "next")) { return mk_bound(obj, 60); }"""

# --------------------------------------------------------------------------
# 7. js_call routes a generator FUNCTION, and the coroutine block itself goes in
#    right after js_call (it needs jsdispatch, mk_obj, obj_put, die).
P_CALL_ANCHOR = """long js_call(long callee, long self, long args) {
	long t = tag_of(callee);
	if (t == 7) { return jsdispatch(fa(callee), fb(callee), args); }
	if (t == 8) { return host_call(fa(callee), self, args); }
	if (t == 9) { return builtin_method(fa(callee), fb(callee), args); }
	die2("call of a non function value: ", callee);
	return H_UNDEF;
}"""
P_CALL = """long gen_create(long gf, long args);

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
 * abnf/jsrt.go's jsGenerator and the closure js_genfn wraps.  The Go half runs
 * the body on a goroutine and hands over on two unbuffered channels; this half
 * runs it on a pthread and hands over on one mutex + condition variable.  Both
 * are STRICTLY ALTERNATING - at any moment exactly one side runs - so neither
 * needs a lock around the runtime's own state.
 *
 * WHY A THREAD AND NOT A SECOND STACK.  Measured, not assumed:
 *   - makecontext/swapcontext work on this machine (macOS 15, arm64) but are
 *     deprecated there, and every field this file would have to write into a
 *     ucontext_t (uc_stack.ss_sp, ss_size, uc_link) has a PLATFORM-DEPENDENT
 *     OFFSET.  runtime.c has no headers, so it would have to hard-code them.
 *   - a pthread is reached entirely through opaque pointers plus *_init calls,
 *     so this source is layout-free.
 * The price is a parked thread per live generator, which is exactly the price
 * the Go half already pays in a parked goroutine.
 *
 * THE CONTROL BLOCK is a long[16] rather than a struct, so no new type crosses
 * the C subset, and it is malloc'd rather than allocated on the heap this file
 * collects - the collector must be able to read it while sweeping.  It holds NO
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
 *
 * THE GENERATOR CELL (tag 15):
 *   a  the closure of the body        b  the argument array
 *   c  the control block, RAW         d  the last yielded value
 *   e  raw: 1 once the body returned  f  the sent value / the return value
 */

long coro_alloc(long g) {
	long *cb = (long *)malloc(8 * 16);
	long i = 0;
	while (i < 16) { cb[i] = 0; i = i + 1; }
	cb[2] = (long)malloc(256);
	cb[3] = (long)malloc(256);
	cb[6] = (long)malloc(512);
	i = 0;
	while (i < 64) { long *jb = (long *)cb[6]; jb[i] = 0; i = i + 1; }
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
 * instead of running the body.  The twin of abnf/jsrt.go's js_genfn. */
long js_genfn(long body) { long h = cell_new(16); sa(h, body); return h; }

/* The thread entry.  Its ADDRESS is taken with dlsym, never as a C value. */
void *coro_entry(void *arg) {
	long *cb = (long *)arg;
	long anchor = 0;
	long g;
	long ret;
	cb[5] = ((long)&anchor) + 256;
	pthread_mutex_lock((void *)cb[2]);
	while (cb[0] == 0) { pthread_cond_wait((void *)cb[3], (void *)cb[2]); }
	pthread_mutex_unlock((void *)cb[2]);
	/* The running side always owns GC_STACK_BASE, so a collection triggered by
	 * an allocation inside the body scans THIS stack and not main's. */
	GC_STACK_BASE = cb[5];
	g = cb[7];
	CUR_GEN = g;
	cb[1] = 3;
	ret = js_call(fa(g), H_UNDEF, fb(g));
	sf(g, ret);
	se(g, 1);
	sd(g, H_UNDEF);
	cb[1] = 2;
	pthread_mutex_lock((void *)cb[2]);
	cb[0] = 0;
	pthread_cond_broadcast((void *)cb[3]);
	pthread_mutex_unlock((void *)cb[2]);
	return 0;
}

/* Resume a generator until its next yield or its return.  Answers nothing: the
 * result is in the cell (d = yielded, e = done, f = returned). */
void gen_resume(long g, long sent) {
	long *cb;
	long anchor = 0;
	long save_gen;
	if (fe(g) != 0) { return; }
	if (fc(g) == 0) {
		long blk = coro_alloc(g);
		long th = 0;
		if (CORO_N >= 256) { die("too many live generators"); }
		sc(g, blk);
		CORO[CORO_N] = blk;
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
	if (RES_N >= 64) { die("generators nested too deeply"); }
	if (RES_JB[RES_N] == 0) { RES_JB[RES_N] = (long)malloc(512); }
	RES_LO[RES_N] = ((long)&anchor) - 256;
	RES_HI[RES_N] = GC_STACK_BASE;
	setjmp((void *)RES_JB[RES_N]);
	RES_N = RES_N + 1;
	save_gen = CUR_GEN;
	pthread_mutex_lock((void *)cb[2]);
	cb[0] = 1;
	pthread_cond_broadcast((void *)cb[3]);
	while (cb[0] == 1) { pthread_cond_wait((void *)cb[3], (void *)cb[2]); }
	pthread_mutex_unlock((void *)cb[2]);
	RES_N = RES_N - 1;
	GC_STACK_BASE = RES_HI[RES_N];
	CUR_GEN = save_gen;
}

/* js_yield: suspend the body, hand v to the pending next() and answer the value
 * the FOLLOWING next(x) sends in.  The twin of abnf/jsrt.go's js_yield. */
long js_yield(long v) {
	long g = CUR_GEN;
	long *cb;
	long anchor = 0;
	if (g == 0) { die("yield outside of a generator"); }
	cb = (long *)fc(g);
	sd(g, v);
	cb[4] = ((long)&anchor) - 256;
	setjmp((void *)cb[6]);
	cb[1] = 1;
	pthread_mutex_lock((void *)cb[2]);
	cb[0] = 0;
	pthread_cond_broadcast((void *)cb[3]);
	while (cb[0] == 0) { pthread_cond_wait((void *)cb[3], (void *)cb[2]); }
	pthread_mutex_unlock((void *)cb[2]);
	cb[1] = 3;
	GC_STACK_BASE = cb[5];
	CUR_GEN = g;
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
		gen_resume(g, arg_at(args, 0));
		if (fe(g) != 0) { val = ff(g); done = 1; } else { val = fd(g); done = 0; }
	}
	res = mk_obj();
	obj_put(res, mk_cstr("value"), val);
	obj_put(res, mk_cstr("done"), done != 0 ? H_TRUE : H_FALSE);
	return res;
}"""

# --------------------------------------------------------------------------
# 8. builtin_method: mid 60.
P_BM_ANCHOR = """long builtin_method(long recv, long mid, long args) {
	long t = tag_of(recv);"""
P_BM = """long builtin_method(long recv, long mid, long args) {
	long t = tag_of(recv);
	if (mid == 60) { return gen_next(recv, args); }"""

EDITS = [
    ("libc prototypes", P_PROTO_ANCHOR, P_PROTO),
    ("registry globals", P_GLOB_ANCHOR, P_GLOB),
    ("gc_roots coroutine stacks", P_ROOTS_ANCHOR, P_ROOTS),
    ("is_callable tag 16", P_CALLABLE_ANCHOR, P_CALLABLE),
    ("get_member generator.next", P_MEMBER_ANCHOR, P_MEMBER),
    ("js_call + the coroutine block", P_CALL_ANCHOR, P_CALL),
    ("builtin_method mid 60", P_BM_ANCHOR, P_BM),
]


def main() -> int:
    args = sys.argv[1:]
    out = None
    src_path = "languages/lib/runtime.c"
    from_head = False
    i = 0
    while i < len(args):
        if args[i] == "-o":
            i += 1
            out = args[i]
        elif args[i] == "--head":
            from_head = True
        else:
            src_path = args[i]
        i += 1

    if from_head:
        text = subprocess.check_output(
            ["git", "show", "HEAD:" + src_path], text=True)
    else:
        with open(src_path, encoding="utf-8") as f:
            text = f.read()

    anchor, repl = trace_edit(text)
    if anchor is None:
        sys.stderr.write("coro-floor.py: no gc_trace anchor matched - the cell "
                         "field offsets changed, re-derive them\n")
        return 2
    text = text.replace(anchor, repl, 1)

    for name, anchor, repl in EDITS:
        if text.count(anchor) != 1:
            sys.stderr.write(
                "coro-floor.py: anchor for %r matched %d times, expected 1\n"
                % (name, text.count(anchor)))
            return 2
        text = text.replace(anchor, repl, 1)

    if out:
        with open(out, "w", encoding="utf-8") as f:
            f.write(text)
    else:
        sys.stdout.write(text)
    return 0


if __name__ == "__main__":
    sys.exit(main())
