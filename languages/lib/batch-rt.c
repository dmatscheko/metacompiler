/*
 * batch-rt.c - the string/arena runtime of languages/batch-to-llvm-ir.abnf.
 *
 * Batch compiles to UNBOXED self-contained IR: every value is a NUL-terminated
 * byte string in a bump arena, and there is no handle runtime under it. That
 * codegen is deliberate and is NOT changing - measured, a self-contained native
 * batch binary runs a 200k-iteration loop in 0.44s / 5.75 MB where Lua through
 * the handle runtime takes 2.41s / 3615 MB. What converges with the rest of the
 * plan (docs/runtime-rework-plan.md) is the RUNTIME LAYER: these 26 helpers used
 * to be hand-written LLVM text inside the grammar, and they are now C compiled by
 * languages/c-to-llvm-ir.abnf.
 *
 * IT IS COMPILED BY languages/c-to-llvm-ir.abnf, NOT BY CLANG.
 * The link input is languages/lib/batch-rt.ll, regenerated with
 *
 *     tests/gen-batch-rt-ll.sh
 *
 * exactly as lib/runtime.ll is by tests/gen-runtime-ll.sh, and for the same
 * reason: llvm.BuildExecutable hands its runtime inputs straight to clang, so
 * passing this .c would let CLANG compile the layer.
 *
 * ABI - it is the one the emitted call sites already used, unchanged:
 *   a string is a char* into the arena or a module string literal, an integer is
 *   a plain int. Our C compiler carries a pointer VALUE as i32* where the batch
 *   emitter writes i8*; the two never meet inside one module (clang links the two
 *   objects, and llvm.Run resolves a declaration against the definition in the
 *   separately parsed runtime module), so the pointer stays a pointer either way.
 *
 * C SUBSET NOTES
 *   - `main` exists only because c-to-llvm-ir.abnf refuses a translation unit
 *     without one. tests/gen-batch-rt-ll.sh strips `define i32 @main` back out of
 *     the module, and checks that @__mec_ginit is empty - which is why NO global
 *     in this file has an initializer: an initialized global emits a store into
 *     __mec_ginit, and nothing would call it once main is gone.
 *   - The arena IS bounds-checked now. This header used to say "There is no
 *     bounds check on the arena, exactly as the hand-emitted IR had none. It is
 *     2 MB and a batch program that outruns it walks off the end" - and that is
 *     precisely what happened: tests/bench/mod.bat runs 40,000 iterations at
 *     ~32 bytes each against a 2 MiB arena, a 1.6x margin, and past the ceiling
 *     the binary exits 139 having printed nothing. See rt_bump and rt_die.
 */

int putchar(int c);
int bat_shift(int from);

/* ---- fail-stop -----------------------------------------------------------
 * The two libc names languages/lib/runtime.c already uses for the same job.
 *
 * NEITHER IS AVAILABLE TO `llvm.Run`, deliberately: abnf/llvmmap.go's
 * libcExterns lists only what the IR interpreter can implement EXACTLY, and it
 * can implement neither. Its comment settles the trade - "Loud-and-absent beats
 * listed-and-wrong" - so a limit crossed under llvm.Run stops the run with
 *
 *     IR interpreter: call to undefined external function @write
 *
 * and a nonzero status, while a clang-built -exe prints the diagnostic on
 * stderr and exits 70. Both are a diagnosis; the SIGSEGV they replaced was not.
 * If you are reading that interpreter message from a batch program, you are out
 * of arena.
 *
 * Neither is reached on any successful path. */
long write(int fd, const char *buf, unsigned long n);
void exit(int code);

/* One byte to fd 2, out of a stack char[1] - the arena is exactly what is gone
 * when this runs. Same shape as runtime.c's o_ch. */
int rt_ech(int c) {
    char b[1];
    b[0] = (char)c;
    write(2, b, (unsigned long)1);
    return 0;
}

int rt_estr(char *s) {
    int i = 0;
    while (s[i] != 0) { rt_ech((int)(unsigned char)s[i]); i = i + 1; }
    return 0;
}

/* A decimal on fd 2 WITHOUT rt_int2str, which allocates. */
int rt_enum(int v) {
    char b[16];
    int a = v < 0 ? 0 - v : v;
    int i = 0;
    int k;
    if (v < 0) { rt_ech(45); }
    if (a == 0) { rt_ech(48); return 0; }
    while (a > 0) { b[i] = (char)(48 + a % 10); i = i + 1; a = a / 10; }
    k = i - 1;
    while (k >= 0) { rt_ech((int)b[k]); k = k - 1; }
    return 0;
}

/* The one exit from a fixed-size table that has run out. 70 is EX_SOFTWARE, and
 * it is deliberately not 139: a compiled batch script that segfaults is
 * indistinguishable from the compiler having emitted a broken binary, and that
 * confusion is the whole reason this function exists. */
void rt_die(char *what, int want, int used, int cap) {
    rt_estr("batch-rt: ");
    rt_estr(what);
    rt_estr(" exhausted: request ");
    rt_enum(want);
    rt_estr(", used ");
    rt_enum(used);
    rt_estr(" of ");
    rt_enum(cap);
    rt_estr(" - the compiled program allocates and never frees ");
    rt_estr("(languages/lib/batch-rt.c). Shorten the run or raise the limit there.\n");
    exit(70);
}

/* ---------------------------------------------------------------- the arena */

static char AR[2097152];
static int APOS;

/* The comparison is `n > 2097152 - p` and not `p + n > 2097152` so that it
 * cannot itself overflow: p is always in [0, 2097152] and n >= 0. */
char *rt_bump(int n) {
    int p = APOS;
    if (n < 0 || n > 2097152 - p) {
        rt_die("string arena", n, p, 2097152);
    }
    APOS = p + n;
    return &AR[p];
}

/* A real "" at a nonzero address: the IR interpreter traps a load at address 0. */
static char EMPTY[1];

/* Call arguments: DEPTH (256) frames of ARGN (10) slots. These are the FIRST
 * globals batch shares with the emitted program - every other byte of runtime
 * state here is private. They are shared because bat_shift, which used to be
 * generated, is not program-coupled at all: it walks two compile-time constants
 * over a fixed-size array, and the only thing keeping it in the grammar was
 * that the array lived on the emitter's side. abnf/llvmlink.go binds a global
 * declaration to its definition by name in both directions, and clang does the
 * same for -exe, so the array can live here and the emitter can declare it. */
char *args[2560];
int frame;

/* ------------------------------------------------------------- byte strings */

int rt_strlen(char *s) {
    int i = 0;
    while (s[i] != 0) { i = i + 1; }
    return i;
}

int rt_streq(char *a, char *b) {
    int i = 0;
    while (1) {
        char ca = a[i];
        char cb = b[i];
        if (ca != cb) { return 0; }
        if (ca == 0) { return 1; }
        i = i + 1;
    }
    return 0;
}

/* lower-case one byte (ASCII only, which is what cmd.exe's comparisons do) */
static char rt_lc(char c) {
    if (c >= 65 && c <= 90) { return c + 32; }
    return c;
}

int rt_streqi(char *a, char *b) {
    int i = 0;
    while (1) {
        char ca = rt_lc(a[i]);
        char cb = rt_lc(b[i]);
        if (ca != cb) { return 0; }
        if (ca == 0) { return 1; }
        i = i + 1;
    }
    return 0;
}

/* Concatenating with "" is the common case and it used to cost a copy: every
 * accumulator in this file starts at EMPTY and grows one rt_strcat at a time,
 * and `set /a` rebuilds a variable's decimal text on every assignment. An arena
 * string is never written after it is built - the only mutation is rt_println
 * into a capture buffer, which comes straight from rt_bump - so handing the
 * other operand back is safe. Nothing in batch compares string pointers by
 * identity (there is no unset marker as in bash-rt.c), so there is no aliasing
 * question to answer here. */
char *rt_strcat(char *a, char *b) {
    int la = rt_strlen(a);
    int lb = rt_strlen(b);
    char *dst;
    int d = 0;
    int i = 0;
    if (lb == 0) { return la == 0 ? EMPTY : a; }
    if (la == 0) { return b; }
    dst = rt_bump(la + lb + 1);
    while (a[i] != 0) { dst[d] = a[i]; d = d + 1; i = i + 1; }
    i = 0;
    while (b[i] != 0) { dst[d] = b[i]; d = d + 1; i = i + 1; }
    dst[d] = 0;
    return dst;
}

/* rt_sub(s, from, to): the byte range [from, to) as a fresh string. The caller clamps. */
char *rt_sub(char *s, int f, int t) {
    int n = t - f;
    if (n < 0) { n = 0; }
    char *dst = rt_bump(n + 1);
    int i = 0;
    while (i < n) { dst[i] = s[f + i]; i = i + 1; }
    dst[n] = 0;
    return dst;
}

/* rt_lastch(s, ch): index of the LAST occurrence of ch, or -1. */
int rt_lastch(char *s, int c) {
    int i = 0;
    int r = -1;
    while (1) {
        int ch = s[i];
        if (ch == 0) { return r; }
        if (ch == c) { r = i; }
        i = i + 1;
    }
    return r;
}

/* rt_findch(s, ch, from): index of the first occurrence at or after from, or -1. */
int rt_findch(char *s, int c, int f) {
    int i = f;
    while (1) {
        int ch = s[i];
        if (ch == 0) { return -1; }
        if (ch == c) { return i; }
        i = i + 1;
    }
    return -1;
}

/* rt_findstr(s, needle, from): case-insensitive index of needle at or after from, or -1. */
int rt_findstr(char *s, char *nd, int f) {
    int ls = rt_strlen(s);
    int ln = rt_strlen(nd);
    int i = f;
    while (i + ln <= ls) {
        int j = 0;
        int bad = 0;
        while (bad == 0 && j < ln) {
            if (rt_lc(s[i + j]) != rt_lc(nd[j])) { bad = 1; }
            else { j = j + 1; }
        }
        if (bad == 0) { return i; }
        i = i + 1;
    }
    return -1;
}

/* --------------------------------------------------------- the 32-bit ints */

/* The digits are formed in a stack buffer anyway, so the length is known before
 * anything is bumped: this used to take a flat 16 bytes of arena whatever the
 * value, which for a `set /a` counter is three to four times what it needs. */
char *rt_int2str(int n) {
    char tmp[16];
    char *dst;
    if (n == 0) {
        dst = rt_bump(2);
        dst[0] = 48;
        dst[1] = 0;
        return dst;
    }
    int neg = n < 0;
    int av = neg ? -n : n;
    int ti = 15;
    while (av > 0) {
        tmp[ti] = 48 + av % 10;
        ti = ti - 1;
        av = av / 10;
    }
    int di = 0;
    dst = rt_bump((15 - ti) + (neg ? 1 : 0) + 1);
    if (neg) { dst[0] = 45; di = 1; }
    int j = ti + 1;
    while (j <= 15) {
        dst[di] = tmp[j];
        di = di + 1;
        j = j + 1;
    }
    dst[di] = 0;
    return dst;
}

int rt_str2int(char *s) {
    int i = 0;
    int v = 0;
    int sign = 1;
    while (s[i] == 32 || s[i] == 9) { i = i + 1; }
    if (s[i] == 45 || s[i] == 43) {
        if (s[i] == 45) { sign = -1; }
        i = i + 1;
    }
    while (s[i] >= 48 && s[i] <= 57) {
        v = v * 10 + (s[i] - 48);
        i = i + 1;
    }
    return v * sign;
}

/* -------------------------------------------------------- output + capture */

int rt_prints(char *s) {
    int i = 0;
    while (s[i] != 0) {
        putchar((unsigned char)s[i]);
        i = i + 1;
    }
    return 0;
}

/* The capture stack for `for /f ... in ('CMD')`. CAP_SZ is new: the buffers used
 * to be a flat 8192 bytes with NO length check anywhere, so a captured command
 * producing more than that walked out of its buffer and over the rest of the
 * arena - and got the right answer whenever nothing else had been bumped since,
 * which is why no test caught it. bash-rt.c's rt_putc had the identical defect
 * and the identical fix: the buffer GROWS, because truncating would be a
 * regression on exactly the cases that used to work by luck. */
static char *CAP_BUFS[8];
static int CAP_LENS[8];
static int CAP_SZ[8];
static int CAP_SP;

int rt_capstart(void) {
    if (CAP_SP >= 8) {
        rt_die("capture nesting", 1, CAP_SP, 8);
    }
    CAP_BUFS[CAP_SP] = rt_bump(8192);
    CAP_LENS[CAP_SP] = 0;
    CAP_SZ[CAP_SP] = 8192;
    CAP_SP = CAP_SP + 1;
    return 0;
}

char *rt_capend(void) {
    CAP_SP = CAP_SP - 1;
    char *buf = CAP_BUFS[CAP_SP];
    buf[CAP_LENS[CAP_SP]] = 0;
    return buf;
}

int rt_println(char *s) {
    if (CAP_SP == 0) {
        rt_prints(s);
        putchar(10);
        return 0;
    }
    int top = CAP_SP - 1;
    char *buf = CAP_BUFS[top];
    int len = CAP_LENS[top];
    int need = len + rt_strlen(s) + 2;   /* the line, its newline, and a NUL */
    int i = 0;
    if (need > CAP_SZ[top]) {
        int nsz = CAP_SZ[top];
        char *nb;
        int t = 0;
        while (nsz < need) { nsz = nsz * 2; }
        nb = rt_bump(nsz);
        while (t < len) { nb[t] = buf[t]; t = t + 1; }
        CAP_BUFS[top] = nb;
        CAP_SZ[top] = nsz;
        buf = nb;
    }
    while (s[i] != 0) {
        buf[len + i] = s[i];
        i = i + 1;
    }
    buf[len + i] = 10;
    CAP_LENS[top] = len + i + 1;
    return 0;
}

/* ----------------------------------------------------- the %VAR% operators */

char *rt_stripq(char *s) {
    int n = rt_strlen(s);
    int big = n >= 2;
    int lastIdx = big ? n - 1 : 0;
    int ok = big && s[0] == 34 && s[lastIdx] == 34;
    /* The unquoted case is the whole string, and an arena string is immutable,
     * so it needs no copy. rt_mods calls this on every ~ modifier and rt_fskind
     * on every `if exist`. */
    if (ok == 0) { return s; }
    return rt_sub(s, 1, n - 1);
}

char *rt_substr(char *s, int st0, int ln, int hl) {
    int n = rt_strlen(s);
    int st = st0 < 0 ? n + st0 : st0;
    if (st < 0) { st = 0; }
    if (st > n) { st = n; }
    int end = hl != 0 ? (ln < 0 ? n + ln : st + ln) : n;
    if (end < st) { end = st; }
    if (end > n) { end = n; }
    return rt_sub(s, st, end);
}

char *rt_subst(char *s, char *fr, char *to, int star) {
    int lf = rt_strlen(fr);
    int ls = rt_strlen(s);
    if (lf == 0) { return s; }
    if (star != 0) {
        /* *from=to : everything up to and including the match is replaced */
        int sp0 = rt_findstr(s, fr, 0);
        if (sp0 < 0) { return s; }
        return rt_strcat(to, rt_sub(s, sp0 + lf, ls));
    }
    /* from=to : every occurrence */
    char *out = EMPTY;
    int i = 0;
    while (1) {
        int p = rt_findstr(s, fr, i);
        if (p < 0) { return rt_strcat(out, rt_sub(s, i, ls)); }
        out = rt_strcat(rt_strcat(out, rt_sub(s, i, p)), to);
        i = p + lf;
    }
    return out;
}

/* rt_mods(s, mask): the ~ modifiers as pure string arithmetic.
   mask bits: 1 = drive, 2 = path, 4 = name, 8 = extension (~f is 1|2|4|8). */
char *rt_mods(char *s, int mask) {
    char *q = rt_stripq(s);
    int n = rt_strlen(q);
    int colonIdx = n >= 2 ? 1 : 0;
    int hasDrv = n >= 2 && q[colonIdx] == 58;
    int dEnd = hasDrv ? 2 : 0;
    char *drive = rt_sub(q, 0, dEnd);
    char *rest = rt_sub(q, dEnd, n);
    int rn = rt_strlen(rest);
    int c1 = rt_lastch(rest, 92);
    int c2 = rt_lastch(rest, 47);
    int cut = c1 > c2 ? c1 : c2;
    char *path = rt_sub(rest, 0, cut + 1);
    char *base = rt_sub(rest, cut + 1, rn);
    int bn = rt_strlen(base);
    int dot = rt_lastch(base, 46);
    int split = dot > 0 ? dot : bn;
    char *nm = rt_sub(base, 0, split);
    char *ext = rt_sub(base, split, bn);
    char *out = rt_strcat((mask & 1) != 0 ? drive : EMPTY, (mask & 2) != 0 ? path : EMPTY);
    out = rt_strcat(out, (mask & 4) != 0 ? nm : EMPTY);
    out = rt_strcat(out, (mask & 8) != 0 ? ext : EMPTY);
    return out;
}

/* There is NO filesystem behind a compiled run, so `if exist` answers from the same
   small MODEL of the standard paths the interpreter uses - the shape
   bash-to-llvm-ir.abnf established for its -e/-f/-d predicates. 0 = absent, 1 =
   exists and is not a directory, 2 = directory. Both spellings of a directory (with
   and without the trailing separator) are in the table, which is cheaper than
   stripping. Written as a compare chain rather than a table because a table of
   string pointers would need an initializer, and an initialized global emits a store
   into __mec_ginit - see the header. */
int rt_fskind(char *s) {
    char *q = rt_stripq(s);
    if (rt_streqi(q, "nul")) { return 1; }
    if (rt_streqi(q, "con")) { return 1; }
    if (rt_streqi(q, "prn")) { return 1; }
    if (rt_streqi(q, "aux")) { return 1; }
    if (rt_streqi(q, "com1")) { return 1; }
    if (rt_streqi(q, "com2")) { return 1; }
    if (rt_streqi(q, "com3")) { return 1; }
    if (rt_streqi(q, "com4")) { return 1; }
    if (rt_streqi(q, "com5")) { return 1; }
    if (rt_streqi(q, "com6")) { return 1; }
    if (rt_streqi(q, "com7")) { return 1; }
    if (rt_streqi(q, "com8")) { return 1; }
    if (rt_streqi(q, "com9")) { return 1; }
    if (rt_streqi(q, "lpt1")) { return 1; }
    if (rt_streqi(q, "lpt2")) { return 1; }
    if (rt_streqi(q, "lpt3")) { return 1; }
    if (rt_streqi(q, "lpt4")) { return 1; }
    if (rt_streqi(q, "lpt5")) { return 1; }
    if (rt_streqi(q, "lpt6")) { return 1; }
    if (rt_streqi(q, "lpt7")) { return 1; }
    if (rt_streqi(q, "lpt8")) { return 1; }
    if (rt_streqi(q, "lpt9")) { return 1; }
    if (rt_streqi(q, "c:")) { return 2; }
    if (rt_streqi(q, "c:\\")) { return 2; }
    if (rt_streqi(q, "c:\\windows")) { return 2; }
    if (rt_streqi(q, "c:\\windows\\")) { return 2; }
    if (rt_streqi(q, "c:\\windows\\system32")) { return 2; }
    if (rt_streqi(q, "c:\\windows\\system32\\")) { return 2; }
    if (rt_streqi(q, "c:\\users")) { return 2; }
    if (rt_streqi(q, "c:\\users\\")) { return 2; }
    if (rt_streqi(q, "c:\\program files")) { return 2; }
    if (rt_streqi(q, "c:\\program files\\")) { return 2; }
    if (rt_streqi(q, "c:\\programdata")) { return 2; }
    if (rt_streqi(q, "c:\\programdata\\")) { return 2; }
    return 0;
}

/* -------------------------------------------- for /f: line/token boundaries */

int rt_isdelim(int c, char *d) {
    int i = 0;
    while (1) {
        int ch = d[i];
        if (ch == 0) { return 0; }
        if (ch == c) { return 1; }
        i = i + 1;
    }
    return 0;
}

/* rt_tokb(s, delims, idx, which): boundary of the idx-th (1-based) token.
     which 0 = its start, 1 = its end, 2 = the start of the remainder after it.
   Answers -1 when there is no such token. The terminating NUL is never a delimiter,
   so both scans stop at the end of the string without a separate length check. */
int rt_tokb(char *s, char *d, int idx, int which) {
    int i = 0;
    int k = 0;
    int st = 0;
    while (1) {
        while (rt_isdelim(s[i], d) != 0) { i = i + 1; }
        if (s[i] == 0) { return -1; }
        st = i;
        while (s[i] != 0 && rt_isdelim(s[i], d) == 0) { i = i + 1; }
        k = k + 1;
        if (k == idx) {
            if (which == 2) {
                while (rt_isdelim(s[i], d) != 0) { i = i + 1; }
                return i;
            }
            return which == 0 ? st : i;
        }
    }
    return -1;
}

int rt_nlines(char *s) {
    int i = 0;
    int n = 0;
    int pend = 0;
    while (s[i] != 0) {
        if (s[i] == 10) { n = n + 1; pend = 0; }
        else { pend = 1; }
        i = i + 1;
    }
    return n + pend;
}

/* rt_lineat(s, idx): the idx-th newline-separated line, with a trailing CR removed. */
char *rt_lineat(char *s, int idx) {
    int n = rt_strlen(s);
    int i = 0;
    int k = 0;
    int st = 0;
    int en = -1;
    while (i < n) {
        if (s[i] == 10) {
            if (k == idx) { en = i; i = n; }
            else {
                k = k + 1;
                st = i + 1;
                i = i + 1;
            }
        } else { i = i + 1; }
    }
    if (en < 0) {
        if (k != idx) { return EMPTY; }
        en = n;
    }
    int nonEmpty = en > st;
    int lastI = nonEmpty ? en - 1 : st;
    int isCr = nonEmpty && s[lastI] == 13;
    return rt_sub(s, st, isCr ? en - 1 : en);
}

/* ------------------------------------------------- the second expansion pass */

/* bat_lookup is GENERATED per program by batch-to-llvm-ir.abnf, from the variable
   table of the batch script being compiled - it is the one thing in this file that
   calls BACK into the program module. That is why rt_expand was the one helper of
   the 26 that could not move when the other 25 did: c-to-llvm-ir.abnf used to emit
   a declared-only function's POINTER PARAMETER as a plain i32
   (`declare i32* @bat_lookup(i32 %0)`), which in a native binary truncates a 64-bit
   pointer to 32 bits. Commit e8bf2c3 fixed that - funcParamPtrs now records
   pointer-ness for a declared-only function and emitCallArgs converts - and it is
   pinned by SECTION 48 of tests/c-test-full.c, so the prototype below survives:

       declare i32* @bat_lookup(i32* %0)

   The call is resolved across modules by NAME, in both directions: clang links the
   two objects, and llvm.Run binds a block-less function object to the definition
   carrying its name (abnf/llvmlink.go). */
char *bat_lookup(char *nm);

/* rt_expand(s): one more %VAR% pass over an already-expanded string, which is what
   `call set NAME=%%%indirect%%%` needs. A %..% pair with at least one byte between
   the percents is looked up; anything else is copied through literally. */
char *rt_expand(char *s) {
    int n = rt_strlen(s);
    char *out = EMPTY;
    int i = 0;
    while (i < n) {
        int j = s[i] == 37 ? rt_findch(s, 37, i + 1) : -1;
        if (j > i + 1) {
            out = rt_strcat(out, bat_lookup(rt_sub(s, i + 1, j)));
            i = j + 1;
        } else {
            out = rt_strcat(out, rt_sub(s, i, i + 1));
            i = i + 1;
        }
    }
    return out;
}

/* c-to-llvm-ir.abnf refuses a translation unit without a main; the batch program
   module supplies the real one, and tests/gen-batch-rt-ll.sh strips this one out
   of the emitted module. */
int main(void) { return 0; }

/* `shift /n`: slide args[from..ARGN-2] of the CURRENT frame down by one and
 * blank the last slot. ARGN is 10 and the frame stride is ARGN, both compile-
 * time constants, so nothing here depends on the program being compiled - which
 * is why this moved out of batch-to-llvm-ir.abnf. */
int bat_shift(int from)
{
    int base;
    int i;
    /* args is DEPTH (256) frames of ARGN (10), and `frame` is written by the
     * EMITTED program, not here - batch-to-llvm-ir.abnf's makeCall increments it
     * with no depth check, so a batch script that recurses 256 deep indexes past
     * args. This runtime cannot stop that store, but it can refuse to compound
     * it, and the diagnostic names the real cause. */
    if (frame < 0 || frame >= 256) {
        rt_die("call nesting", 1, frame, 256);
    }
    base = frame * 10;
    i = from;
    while (i < 9) {
        args[base + i] = args[base + i + 1];
        i = i + 1;
    }
    args[base + 9] = EMPTY;
    return 0;
}
