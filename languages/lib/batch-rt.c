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
 *   - There is no bounds check on the arena, exactly as the hand-emitted IR had
 *     none. It is 2 MB and a batch program that outruns it walks off the end.
 */

int putchar(int c);
int bat_shift(int from);

/* ---------------------------------------------------------------- the arena */

static char AR[2097152];
static int APOS;

char *rt_bump(int n) {
    char *p = &AR[APOS];
    APOS = APOS + n;
    return p;
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

char *rt_strcat(char *a, char *b) {
    int la = rt_strlen(a);
    int lb = rt_strlen(b);
    char *dst = rt_bump(la + lb + 1);
    int d = 0;
    int i = 0;
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

char *rt_int2str(int n) {
    char tmp[16];
    char *dst = rt_bump(16);
    if (n == 0) {
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

static char *CAP_BUFS[8];
static int CAP_LENS[8];
static int CAP_SP;

int rt_capstart(void) {
    CAP_BUFS[CAP_SP] = rt_bump(8192);
    CAP_LENS[CAP_SP] = 0;
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
    int i = 0;
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
    return rt_sub(s, ok ? 1 : 0, ok ? n - 1 : n);
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
    base = frame * 10;
    i = from;
    while (i < 9) {
        args[base + i] = args[base + i + 1];
        i = i + 1;
    }
    args[base + 9] = EMPTY;
    return 0;
}
