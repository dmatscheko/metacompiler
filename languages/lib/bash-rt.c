/*
 * bash-rt.c - the bash compiler's string runtime, in C.
 *
 * Compiled by languages/c-to-llvm-ir.abnf into the checked-in
 * languages/lib/bash-rt.ll by tests/gen-bash-rt-ll.sh, and linked with the
 * module languages/bash-to-llvm-ir.abnf emits.
 *
 * Every POINTER in a module produced by c-to-llvm-ir.abnf is an `i32*`,
 * whatever it points at, while the bash module's strings are `i8*`. The two
 * are ABI-identical but not type-identical, so the grammar wraps each helper
 * in a tiny `rtw_*` shim that bitcasts.
 */


/* ---- the runtime's shared state ------------------------------------------
 * These globals are the runtime's AND the emitted program's: the program loads
 * last_status, writes gvars[], pushes argv. They are defined here, once, and
 * the grammar DECLARES them by name (`external global`) and the linker binds
 * the two - abnf/llvmlink.go for llvm.Run, clang for -exe.
 */
int last_status;              /* $?                                        */
int exited;                   /* set by `exit`                             */
int exit_code;

/* A real "" at a NONZERO address; the IR interpreter traps loads at address 0,
 * so an empty / unset value must be a genuine pointer to a NUL byte. A SECOND
 * empty string at a different address is the "variable is unset" marker: it
 * reads as "" everywhere, but a pointer comparison against it answers
 * ${v-default} / ${v:+alt}. */
char empty[1];
char unset_marker[1];

/* $0: the program's own path, stored by main's entry block from c.file. It is a
 * global rather than an argument because rt_getvar_byname has to answer ${!r}
 * with r=0, and that lookup happens far from the emitter. */
char *script_name;

/* Variable storage is ONE array of char*, indexed by a compile-time id, so the
 * `local` unwinder can restore a slot chosen at run time. NVARS = 1024. */
char *gvars[1024];
int nvars;

/* Every fixed-size table below calls rt_die when it is full: a refused write
 * would be a silently wrong answer (and an unbalanced rt_cap_begin/rt_cap_end
 * pair would index cap_buf[-1]). */

/* The {name, slot} table: varnames[i] is the source name of gvars[i]. The
 * emitter fills it in main's entry block, in the same loop that marks every
 * slot unset, because only the emitter knows the names. It is what lets
 * rt_getvar_byname / rt_setvar_byname / rt_eval_assign live HERE rather than
 * being generated as a chain over every name the program mentions. */
char *varnames[1024];

/* The `local` save stack: (slot id, previous value) pairs. LSMAX = 512.
 * A BINDING IS NOT JUST A SCALAR: an ARRAY of the same name lives in the flat
 * (name, key, value) store below, and bash restores it on return too, so each
 * entry also records how many of its rows were copied onto the array save
 * stack. ls_acnt[t] is 0 for the usual scalar case and the pop then does
 * nothing beyond clearing whatever the call created. */
int ls_id[512];
char *ls_val[512];
int ls_acnt[512];
int ls_amark[512];   /* the binding's marks (rt_arr_mark 1, rt_arr_dmark 2, rt_arr_smark 4), restored too */
int ls_top;

/* The array save stack the entries above index into. LASMAX = 4096. */
char *las_k[4096];
char *las_v[4096];
int las_top;

/* Output capture for $( ) / ` `: a stack of fixed-size arena buffers.
 * CAPMAX = 16, CAPSZ = 8192. */
char *cap_buf[16];
int cap_len[16];
int cap_sz[16];      /* how big cap_buf[i] is; rt_putc doubles it when it fills */
int cap_depth;

/* Subshell isolation: ( LIST ) snapshots the whole variable array. SSDEPTH = 8. */
char *ss_save[8192];
int ss_depth;

/* ...and the whole array store, so `a=(p q); ( a[0]=Z )` leaves the caller with
 * "p q" (a command substitution takes the same snapshot). The store is a flat
 * triple table (arr_nm / arr_k / arr_v, NARR = 4096) whose entries are arena
 * pointers never rewritten in place, so saving the three pointer arrays and arr_n
 * is a complete copy. It is BSS written only as far as arr_n, so only programs
 * with subshells AND arrays make those pages resident.
 *
 * Attributes (-i/-u/-l/-A) are not here: bash-to-llvm-ir.abnf keeps them in
 * COMPILE-TIME maps and unwinds its attribute log around a subshell body. */
char *ss_anm[32768];
char *ss_ak[32768];
char *ss_av[32768];
int ss_an[8];

/* Standard input for `read`: the buffer and the read position. */
char *stdin_buf;
int stdin_pos;

/* Shell options (set -e / -u / -o pipefail) and the abort flag a set -u failure
 * raises during expansion, plus a [[ ]] that could not be evaluated. */
int opt_errexit;
int opt_nounset;
int opt_pipefail;
int abort_flag;
int cond_err;

/* Two output channels. 0 = standard output (captured by $( ) when one is open),
 * 1 = the standard error channel, 2 = discarded. */
int sink_out;
int sink_err = 1;

/* 1 exactly when this module was linked into a NATIVE executable, stored by the
 * emitted main's entry block (bash-to-llvm-ir.abnf). It is what lets rt_putc
 * write channel 1 to fd 2 without making llvm.Run - which has no fd 2 and no
 * `write` extern - abort on the first stderr byte. See rt_putc. */
int err_native;

/* Positional parameters live in ONE global stack rather than in function
 * arguments, so "$@" can expand to any number of words. NARGV = 4096. */
char *argv[4096];
int argv_top;
int frame_base;
int frame_n;

/* ---- prototypes of every runtime helper (any chunk may call any other) ---- */
char *rt_bump(int n);
int rt_strlen(char *s);
int rt_charlen(char *s);
int rt_charoff(char *s, int k);
int rt_streq(char *a, char *b);
int rt_strcmp(char *a, char *b);
char *rt_strcat(char *a, char *b);
char *rt_int2str(long n);
long rt_str2int(char *s);
long rt_arith(char *s);
int rt_arith_fail(char *expr, char *msg, char *tok);
int rt_class(char *p, int ch);
int rt_cpw(char *s);
int rt_cpat(char *s);
int rt_clsid(char *nm);
int re_ctype(int ctid, int ctch);
int rt_eg(char *egp, char *egs, int egstar);
int rt_glob(char *p, char *s);
char *rt_substr(char *s, int off, int n);
char *rt_csubstr(char *s, int off, int n);
int rt_egclose(char *ecq);
int rt_egalt(char *eaq, int eaa);
int rt_matchlen(char *p, char *s, int i);
int rt_matchend(char *p, char *s);
char *rt_strip(char *s, char *p, int mode);
char *rt_replace(char *s, char *p, char *r, int mode);
char *rt_case(char *s, int mode, char *pat);
char *rt_shquote(char *sq);
char *rt_ansic(char *ac);
int rt_haschar(char *s, int ch);
char *rt_read_line(int u);
char *rt_field(char *l, char *ifs, int idx, int rest);
int rt_nifs(char *l, char *ifs);
char *rt_pad(char *s, int w, int left, int zero);
char *rt_radix(char *s, int base, int upper, int alt);
char *rt_intprec(char *s, int prec);
char *rt_pfq(char *s);
char *rt_unescape(char *s);
int rt_nfields(char *v);
char *rt_getfield(char *v, int k);
char *rt_wordjoin(char *v);
char *rt_uqlist(char *l, char *ifs);
char *rt_splitifs(char *v, char *ifs);
int rt_ifsws(char *ifs, int ch);
char *rt_bnd_acc(char *acc, char *open, char *f);
char *rt_bnd_open(char *open, char *f);
int rt_ifslead(char *v, char *ifs);
int rt_ifstrail(char *v, char *ifs);
char *rt_bnd_acc2(char *acc, char *open, int opened, char *f, int lead, int trail);
char *rt_bnd_open2(char *open, int opened, char *f, int lead, int trail);
int rt_bnd_opened2(int opened, char *f, int lead, int trail);
int rt_arr_find(char *nm, char *k);
int rt_arr_set(char *nm, char *k, char *v);
char *rt_arr_get(char *nm, char *k);
int rt_arr_has(char *nm, char *k);
int rt_arr_del(char *nm, char *k);
int rt_arr_count(char *nm);
char *rt_arr_list(char *nm, char *sep, int star, int keys);
int rt_sc_count(char *nm);
int rt_declp(char *nm, char *flag, int assoc);
char *rt_dqs(char *v);
char *rt_ansiqs(char *v);
char *rt_arrassign(char *nm, int assoc);
int rt_keymeta(char *k);
char *rt_shq(char *v, int isset);
int rt_isset(char *v);
char *rt_declasg(char *nm, char *at, char *v, int isset);
char *rt_declxa(char *nm, char *at, char *v, int isset, int assoc, char *ifs, int star);
char *rt_declpos(char *ifs, char *sep, int star);
char *rt_kvq(char *nm, int assoc, char *v, int isset);
char *rt_kvp(char *nm, char *sep, int star, char *v, int isset);
char *rt_attrf(char *nm, char *at, char *sep, int star);
char *rt_prompt(char *s);
char *rt_ifsfirst(void);
char *rt_sc_get(char *nm, char *k);
int rt_sc_has(char *nm, char *k);
char *rt_sc_list(char *nm, char *sep, int star, int keys);
int rt_isposnm(char *nm);
char *rt_params0(void);
char *rt_pfxnames(char *p, char *sep, int star);
int rt_arr_nextidx(char *nm);
int rt_arr_clear(char *nm);
int rt_arr_mark(char *nm);
int rt_arr_dmark(char *nm);
int rt_arr_dmarked(char *nm);
int rt_arr_smark(char *nm);
int rt_arr_smarked(char *nm);
char *rt_finlist(char *v);
int rt_isfin(char *v);
int rt_arr_append(char *nm, char *f);
char *rt_slicefields(char *f, int off, int n, char *sep, int star);
char *rt_globescape(char *s);
char *rt_catfields(char *a, char *b);
char *rt_mapf(char *v, int op, char *p, char *r, int mode, char *sep, int star);
int rt_argpush(char *v);
int rt_echo(char *fields, int newline, int escapes);
char *rt_param(int n);
char *rt_params(char *sep, int star);
int rt_regex_search(char *pat, char *subj, char *flags, int *caps, int ncaps);
char *rt_regex_error(void);
char *rt_nounset(char *v, char *name, int isarr);
int rt_putc(int c);
int rt_cap_begin(int u);
char *rt_cap_end(int u);
int rt_prints(char *s);
int rt_ss_save(int u);
int rt_ss_restore(int u);
int rt_push_local(int id);
int rt_pop_locals(int d);
int rt_unset_up(int id, int d, int fn);
int rt_fskind(char *p);
char *rt_getvar_byname(char *n);
int rt_setvar_byname(char *n, char *v);
int rt_eval_assign(char *s);
int putchar(int c);

/* ---- fail-stop -----------------------------------------------------------
 * The two libc names languages/lib/runtime.c already uses for the same job.
 *
 * Neither is available to `llvm.Run` (abnf/llvmmap.go's libcExterns lists only
 * what the IR interpreter implements exactly), so a table overflow under
 * llvm.Run stops with "call to undefined external function @write" - if you
 * see that, you are out of arena; see rt_die. Under a -exe binary it prints the
 * diagnostic on stderr and exits 70. Neither is reached on a successful path. */
long write(int fd, const char *buf, unsigned long n);
void exit(int code);

/* One byte to fd 2. A char[1] on the stack rather than a pointer into the
 * arena, because the arena is exactly what is gone when this runs (the same
 * shape as runtime.c's o_ch). */
int rt_ech(int c)
{
    char b[1];
    b[0] = (char)c;
    write(2, b, (unsigned long)1);
    return 0;
}

int rt_estr(char *s)
{
    int i = 0;
    while (s[i] != 0) {
        rt_ech((int)(unsigned char)s[i]);
        i = i + 1;
    }
    return 0;
}

/* A decimal on fd 2 WITHOUT rt_int2str, which allocates. */
int rt_enum(int v)
{
    char b[16];
    int a = v < 0 ? 0 - v : v;
    int i = 0;
    int k;
    if (v < 0) rt_ech(45);
    if (a == 0) { rt_ech(48); return 0; }
    while (a > 0) {
        b[i] = (char)(48 + (a % 10));
        i = i + 1;
        a = a / 10;
    }
    k = i - 1;
    while (k >= 0) {
        rt_ech((int)b[k]);
        k = k - 1;
    }
    return 0;
}

/* The one exit from a fixed-size table that has run out. `what` names the
 * table, so the message says which of the six it was; `want` is the request,
 * `used` and `cap` the state of the table in its own units (bytes for the
 * arena, entries for the rest).
 *
 * 70 is EX_SOFTWARE. It is deliberately not 139: a shell script that segfaults
 * is indistinguishable from the compiler having produced a broken binary, and
 * that confusion is the whole reason this function exists. */
void rt_die(char *what, int want, int used, int cap)
{
    rt_estr("bash-rt: ");
    rt_estr(what);
    rt_estr(" exhausted: request ");
    rt_enum(want);
    rt_estr(", used ");
    rt_enum(used);
    rt_estr(" of ");
    rt_enum(cap);
    rt_estr(" - the compiled program allocates and never frees ");
    rt_estr("(languages/lib/bash-rt.c). Shorten the run or raise the limit there.\n");
    exit(70);
}

/* ---- the string heap ------------------------------------------------------
 * One byte arena plus a bump offset; rt_bump hands out slices and nothing is
 * ever freed (manual 7.8 - the arena is a decision, not an omission).
 *
 * The bounds check is essential: without it a program that outruns 4 MiB
 * writes over the globals after the arena and dies silently with SIGSEGV.
 * The comparison is written `n > 4194304 - p` and not `p + n > 4194304` so it
 * cannot itself overflow: p is always in [0, 4194304] and n >= 0.
 */
char arena[4194304];
int arena_pos;

char *rt_bump(int n) {
    int p = arena_pos;
    if (n < 0 || n > 4194304 - p) {
        rt_die("string arena", n, p, 4194304);
    }
    arena_pos = p + n;
    return &arena[p];
}

/* i32 rt_strlen(i8* s) */
int rt_strlen(char *s) {
    int i = 0;
    while (s[i] != 0) i = i + 1;
    return i;
}

/* rt_charlen: the number of CHARACTERS in s, which is what bash's ${#s} counts -
 * not bytes and not UTF-16 units. A UTF-8 continuation byte is 10xxxxxx, so
 * every byte with (b & 0xC0) != 0x80 starts one character. */
int rt_charlen(char *s) {
    int i = 0, n = 0;
    while (s[i] != 0) {
        int c = (int)(unsigned char)s[i];
        if ((c & 192) != 128) n = n + 1;
        i = i + 1;
    }
    return n;
}

/* rt_charoff: the BYTE offset of character number k, clamped to the length of s. */
int rt_charoff(char *s, int k) {
    int i = 0, n = 0;
    while (s[i] != 0) {
        int c = (int)(unsigned char)s[i];
        if ((c & 192) != 128) {
            if (n >= k) return i;
            n = n + 1;
        }
        i = i + 1;
    }
    return i;
}

/* rt_streq -> 1 if equal, else 0. */
int rt_streq(char *a, char *b) {
    int i = 0;
    while (1) {
        char ca = a[i], cb = b[i];
        if (ca != cb) return 0;
        if (ca == 0) return 1;
        i = i + 1;
    }
    return 0;
}

/* rt_strcmp: -1 / 0 / 1 by unsigned byte order (the shell's < and >). */
int rt_strcmp(char *a, char *b) {
    int i = 0;
    while (1) {
        int ca = (int)(unsigned char)a[i];
        int cb = (int)(unsigned char)b[i];
        if (ca < cb) return -1;
        if (ca > cb) return 1;
        if (ca == 0) return 0;
        i = i + 1;
    }
    return 0;
}

/* rt_strcat: a followed by b, as an arena string.
 *
 * Concatenating with "" is the common case (accumulators start at `empty`), so
 * it hands the other operand back without copying. That is safe because an
 * arena string is never written after it is built.
 *
 * The `la == 0 && lb == 0` arm returns the shared `empty` and NOT `a`, because
 * `a` may be `unset_marker`: that pointer is compared by IDENTITY in rt_nounset
 * to answer ${v-default}, and a concatenation result must never be it. Every
 * other arm returns an operand of nonzero length, which cannot be the marker. */
char *rt_strcat(char *a, char *b) {
    int la = rt_strlen(a);
    int lb = rt_strlen(b);
    char *dst;
    int i = 0, d = 0;
    if (lb == 0) return la == 0 ? &empty[0] : a;
    if (la == 0) return b;
    dst = rt_bump(la + lb + 1);
    while (a[i] != 0) { dst[d] = a[i]; d = d + 1; i = i + 1; }
    i = 0;
    while (b[i] != 0) { dst[d] = b[i]; d = d + 1; i = i + 1; }
    dst[d] = 0;
    return dst;
}

/* rt_int2str: decimal string in the arena (mirrors the interpreter's numToStr).
 * The digits are formed in a stack buffer first so the allocation is exact. A
 * digit is taken from the remainder of the SIGNED value, so LONG_MIN, which has
 * no positive counterpart, prints like every other negative number. */
char *rt_int2str(long n) {
    char tmp[24];
    char *dst;
    if (n == 0) { dst = rt_bump(2); dst[0] = 48; dst[1] = 0; return dst; }
    int neg = n < 0;
    long a = n;
    int t = 23;
    while (a != 0) {
        long q = a / 10;
        long dg = a - q * 10;
        if (dg < 0) dg = 0 - dg;
        tmp[t] = (char)(48 + (int)dg);
        t = t - 1;
        a = q;
    }
    int d = 0;
    dst = rt_bump((23 - t) + (neg ? 1 : 0) + 1);
    if (neg) { dst[0] = 45; d = 1; }
    int j = t + 1;
    while (j <= 23) { dst[d] = tmp[j]; d = d + 1; j = j + 1; }
    dst[d] = 0;
    return dst;
}

/* rt_str2int: leading spaces/tabs, one +/- sign, decimal digits until a
 * non-digit, wrapping modulo 2^64 like every other 64-bit operation here. */
long rt_str2int(char *s) {
    int i = 0, sign = 1;
    long v = 0;
    while (s[i] == 32 || s[i] == 9) i = i + 1;
    if (s[i] == 45 || s[i] == 43) {
        if (s[i] == 45) sign = -1;
        i = i + 1;
    }
    while (s[i] >= 48 && s[i] <= 57) {
        v = v * 10 + (long)((int)s[i] - 48);
        i = i + 1;
    }
    return sign < 0 ? 0 - v : v;
}

/* The fourteen POSIX class names, WHOLE-matched, shared by the glob bracket
 * parser (rt_class) and the ERE bracket parser (re_clsname). 0 = not a class
 * name. */
int rt_clsid(char *nm) {
    int id = 0;
    id = rt_streq(nm, "alpha") != 0 ? 1 : id;
    id = rt_streq(nm, "digit") != 0 ? 2 : id;
    id = rt_streq(nm, "alnum") != 0 ? 3 : id;
    id = rt_streq(nm, "upper") != 0 ? 4 : id;
    id = rt_streq(nm, "lower") != 0 ? 5 : id;
    id = rt_streq(nm, "space") != 0 ? 6 : id;
    id = rt_streq(nm, "blank") != 0 ? 7 : id;
    id = rt_streq(nm, "punct") != 0 ? 8 : id;
    id = rt_streq(nm, "print") != 0 ? 9 : id;
    id = rt_streq(nm, "graph") != 0 ? 10 : id;
    id = rt_streq(nm, "cntrl") != 0 ? 11 : id;
    id = rt_streq(nm, "xdigit") != 0 ? 12 : id;
    id = rt_streq(nm, "word") != 0 ? 13 : id;
    id = rt_streq(nm, "ascii") != 0 ? 14 : id;
    return id;
}

/* rt_class: p points at "[". Returns -1 when the bracket expression is
 * unterminated (the "[" is then a literal), else (bytesConsumed << 1) | matched.
 *
 * "[:name:]" inside the brackets is a POSIX class. An UNKNOWN name is not an
 * error in a glob - bash matches nothing, silently - so it simply contributes
 * no characters. */
/* ---- ONE UTF-8 CODEPOINT ---------------------------------------------------
 * A bracket expression matches one CHARACTER (`[[ ï == [ïn] ]]` is true in
 * bash 5.3), so the glob matcher reads codepoints, not bytes.
 *
 * rt_cpw answers 1 for anything that is not a well-formed sequence, so invalid
 * UTF-8 still advances and compares byte-for-byte. */
int rt_cpw(char *s) {
    int c0 = (int)(unsigned char)s[0];
    int w = 1;
    int i;
    if (c0 >= 240) { w = 4; }
    else if (c0 >= 224) { w = 3; }
    else if (c0 >= 192) { w = 2; }
    i = 1;
    while (i < w) {
        int cc = (int)(unsigned char)s[i];
        if (cc < 128 || cc > 191) return 1;
        i = i + 1;
    }
    return w;
}

int rt_cpat(char *s) {
    int w = rt_cpw(s);
    int c0 = (int)(unsigned char)s[0];
    int v;
    int i;
    if (w == 1) return c0;
    if (w == 2) { v = c0 - 192; }
    else if (w == 3) { v = c0 - 224; }
    else { v = c0 - 240; }
    i = 1;
    while (i < w) {
        v = v * 64 + ((int)(unsigned char)s[i] - 128);
        i = i + 1;
    }
    return v;
}

/* ---- arithmetic -----------------------------------------------------------
 * bash's integers are intmax_t: 64 bits, wrapping on overflow, / and %
 * truncating toward zero. The emitter compiles an expression whose text is
 * known at compile time straight to i64 instructions; what only exists at run
 * time - a variable's value that is not a plain number (`r=a; a=7; $(( r + 1 ))`
 * is 8), an expression with a $-expansion in it (expanded first and then parsed,
 * so `e="1 + 2"; $(( $e * 2 ))` is 5) - goes through rt_arith below.
 *
 * An error (division by 0, a negative exponent, a syntax error, a digit too
 * great for its base) is reported on the standard error channel and raises
 * arith_err; the expression answers 0 and the emitter's check after it abandons
 * the command, as bash does. Twin of arEval and its helpers in
 * bash-interpreter.abnf. */
int arith_err;

int rt_arith_fail(char *expr, char *msg, char *tok)
{
    int saved;
    arith_err = 1;
    saved = sink_out;
    sink_out = sink_err;
    rt_prints("bash: ");
    if (expr[0] != 0) { rt_prints(expr); rt_prints(": "); }
    rt_prints(msg);
    if (tok[0] != 0) { rt_prints(" (error token is \""); rt_prints(tok); rt_prints("\")"); }
    rt_prints("\n");
    sink_out = saved;
    return 0;
}

/* / and %: C's truncating division, plus the two cases the hardware leaves
 * undefined. A zero divisor is bash's "division by 0"; LONG_MIN / -1 wraps back
 * to LONG_MIN and LONG_MIN % -1 is 0, which is what bash 5.3 prints. */
long rt_idiv(long a, long b, int mod)
{
    if (b == 0) {
        rt_arith_fail(empty, "division by 0", empty);
        return 0;
    }
    if (b == -1) return mod != 0 ? 0 : 0 - a;
    return mod != 0 ? a % b : a / b;
}

/* ** by repeated squaring, wrapping like every product. */
long rt_ipow(long b, long e)
{
    long r = 1;
    if (e < 0) {
        rt_arith_fail(empty, "exponent less than 0", empty);
        return 0;
    }
    while (e > 0) {
        if ((e & 1) != 0) r = r * b;
        e = e >> 1;
        if (e > 0) b = b * b;
    }
    return r;
}

/* The evaluator's state. A variable's value is evaluated by a NESTED rt_arith,
 * which saves and restores ar_s / ar_len / ar_p around itself. */
char *ar_s;        /* the expression text                                      */
int ar_len;
int ar_p;          /* the cursor                                               */
int ar_bad;        /* an error was reported: every level unwinds with 0        */
int ar_noeval;     /* > 0 in an operand && / || / ?: skips: no store, no error */
int ar_depth;      /* nesting through variables' values                        */

long ar_comma(void);
long ar_assign(void);
long ar_unary(void);

int ar_at(int d)
{
    int i = ar_p + d;
    if (i < 0 || i >= ar_len) return 0;
    return (int)(unsigned char)ar_s[i];
}

int ar_ws(void)
{
    while (ar_at(0) == 32 || ar_at(0) == 9 || ar_at(0) == 10) ar_p = ar_p + 1;
    return 0;
}

int ar_idch(int c, int first)
{
    if ((c >= 97 && c <= 122) || (c >= 65 && c <= 90) || c == 95) return 1;
    if (first == 0 && c >= 48 && c <= 57) return 1;
    return 0;
}

/* An error at the cursor: bash names the rest of the expression as the token. */
long ar_err(char *msg)
{
    if (ar_bad == 0) {
        ar_bad = 1;
        rt_arith_fail(ar_s, msg, &ar_s[ar_p]);
    }
    return 0;
}

/* A numeric constant: 0x / 0X hexadecimal, a leading 0 octal, BASE#DIGITS with
 * a base from 2 to 64, else decimal. bash scans every letter, digit, @ and _
 * into the constant, so `12abc` is "value too great for base", not 12. The
 * digit alphabet is base dependent above 36: a-z are 10..35, A-Z 36..61, @ 62
 * and _ 63, while at 36 and below the two letter cases coincide. */
long ar_number(void)
{
    long base = 10;
    long v = 0;
    int start = ar_p;
    if (ar_at(0) == 48 && (ar_at(1) == 120 || ar_at(1) == 88)) {
        base = 16;
        ar_p = ar_p + 2;
    } else if (ar_at(0) == 48) {
        base = 8;
    }
    while (1) {
        int c = ar_at(0);
        long d = -1;
        if (c == 35) {
            if (v < 2 || v > 64) {
                ar_p = start;
                return ar_err("invalid arithmetic base");
            }
            base = v;
            v = 0;
            ar_p = ar_p + 1;
            continue;
        }
        if (c >= 48 && c <= 57) d = c - 48;
        else if (c >= 97 && c <= 122) d = c - 87;
        else if (c >= 65 && c <= 90) d = base > 36 ? c - 29 : c - 55;
        else if (c == 64) d = 62;
        else if (c == 95) d = 63;
        else break;
        if (d >= base) {
            ar_p = start;
            return ar_err("value too great for base");
        }
        v = v * base + d;
        ar_p = ar_p + 1;
    }
    return v;
}

/* The name at the cursor, cursor advanced past it. */
char *ar_name(void)
{
    int s = ar_p;
    while (ar_idch(ar_at(0), ar_p == s ? 1 : 0) != 0) ar_p = ar_p + 1;
    return rt_substr(ar_s, s, ar_p - s);
}

/* The key an indexed subscript names: a negative index counts back from one past
 * the highest index (`${a[-1]}` is the last element). */
char *ar_key(char *nm, long k)
{
    if (k < 0) k = (long)rt_arr_nextidx(nm) + k;
    return rt_int2str(k);
}

/* [ EXPR ] after a name, or 0 when the name has no subscript. */
char *ar_subscript(char *nm)
{
    long k;
    if (ar_at(0) != 91) return 0;
    ar_p = ar_p + 1;
    k = ar_comma();
    ar_ws();
    if (ar_at(0) != 93) { ar_err("missing `]'"); return 0; }
    ar_p = ar_p + 1;
    return ar_key(nm, k);
}

/* A variable's value as an operand: empty is 0, anything else is itself an
 * expression (`x=010` is 8, `x="1+2"` is 3, `r=a` is a's value). */
long ar_value(char *nm, char *key)
{
    char *v;
    if (ar_noeval > 0) return 0;
    if (key != 0) v = rt_sc_get(nm, key);
    else if (rt_arr_count(nm) > 0) v = rt_arr_get(nm, "0");
    else v = rt_getvar_byname(nm);
    if (v[0] == 0) return 0;
    if (ar_depth >= 1024) return ar_err("expression recursion level exceeded");
    return rt_arith(v);
}

long ar_store(char *nm, char *key, long v)
{
    if (ar_noeval > 0) return v;
    if (key != 0) rt_arr_set(nm, key, rt_int2str(v));
    else if (rt_arr_count(nm) > 0) rt_arr_set(nm, "0", rt_int2str(v));
    else rt_setvar_byname(nm, rt_int2str(v));
    return v;
}

/* One binary operator, shared by `a OP b` and `a OP= b`. */
long ar_apply(int op, long a, long b)
{
    if (op == 43) return a + b;
    if (op == 45) return a - b;
    if (op == 42) return a * b;
    if (op == 47 || op == 37) {
        if (b == 0) {
            if (ar_noeval > 0) return 0;
            return ar_err("division by 0");
        }
        return rt_idiv(a, b, op == 37 ? 1 : 0);
    }
    if (op == 60) return a << (b & 63);
    if (op == 62) return a >> (b & 63);
    if (op == 38) return a & b;
    if (op == 94) return a ^ b;
    if (op == 124) return a | b;
    return b;
}

long ar_primary(void)
{
    long v;
    int c;
    ar_ws();
    c = ar_at(0);
    if (c == 40) {
        ar_p = ar_p + 1;
        v = ar_comma();
        ar_ws();
        if (ar_at(0) != 41) return ar_err("missing `)'");
        ar_p = ar_p + 1;
        return v;
    }
    if (c >= 48 && c <= 57) return ar_number();
    if (ar_idch(c, 1) != 0) {
        char *nm = ar_name();
        char *key = ar_subscript(nm);
        if (ar_bad != 0) return 0;
        v = ar_value(nm, key);
        ar_ws();
        if ((ar_at(0) == 43 && ar_at(1) == 43) || (ar_at(0) == 45 && ar_at(1) == 45)) {
            ar_store(nm, key, ar_at(0) == 43 ? v + 1 : v - 1);
            ar_p = ar_p + 2;
        }
        return v;
    }
    return ar_err("arithmetic syntax error: operand expected");
}

long ar_unary(void)
{
    int c;
    ar_ws();
    c = ar_at(0);
    if (c == 33 && ar_at(1) != 61) { ar_p = ar_p + 1; return ar_unary() == 0 ? 1 : 0; }
    if (c == 126) { ar_p = ar_p + 1; return ~ar_unary(); }
    if ((c == 43 || c == 45) && ar_at(1) == c) {
        int save = ar_p;
        ar_p = ar_p + 2;
        ar_ws();
        if (ar_idch(ar_at(0), 1) != 0) {
            char *nm = ar_name();
            char *key = ar_subscript(nm);
            long v;
            if (ar_bad != 0) return 0;
            v = ar_value(nm, key);
            return ar_store(nm, key, c == 43 ? v + 1 : v - 1);
        }
        ar_p = save;
    }
    if (c == 45) { ar_p = ar_p + 1; return 0 - ar_unary(); }
    if (c == 43) { ar_p = ar_p + 1; return ar_unary(); }
    return ar_primary();
}

long ar_pow(void)
{
    long b = ar_unary();
    long e;
    ar_ws();
    if (ar_at(0) == 42 && ar_at(1) == 42) {
        ar_p = ar_p + 2;
        e = ar_pow();
        if (e < 0) {
            if (ar_noeval > 0) return 0;
            return ar_err("exponent less than 0");
        }
        return rt_ipow(b, e);
    }
    return b;
}

long ar_mul(void)
{
    long v = ar_pow();
    while (ar_bad == 0) {
        int c;
        ar_ws();
        c = ar_at(0);
        if (!((c == 42 && ar_at(1) != 42 && ar_at(1) != 61) || ((c == 47 || c == 37) && ar_at(1) != 61))) break;
        ar_p = ar_p + 1;
        v = ar_apply(c, v, ar_pow());
    }
    return v;
}

long ar_add(void)
{
    long v = ar_mul();
    while (ar_bad == 0) {
        int c;
        ar_ws();
        c = ar_at(0);
        if (!((c == 43 || c == 45) && ar_at(1) != c && ar_at(1) != 61)) break;
        ar_p = ar_p + 1;
        v = ar_apply(c, v, ar_mul());
    }
    return v;
}

long ar_shift(void)
{
    long v = ar_add();
    while (ar_bad == 0) {
        int c;
        ar_ws();
        c = ar_at(0);
        if (!((c == 60 || c == 62) && ar_at(1) == c && ar_at(2) != 61)) break;
        ar_p = ar_p + 2;
        v = ar_apply(c, v, ar_add());
    }
    return v;
}

long ar_cmp(void)
{
    long v = ar_shift();
    while (ar_bad == 0) {
        int c;
        int eq;
        long r;
        ar_ws();
        c = ar_at(0);
        if (!((c == 60 || c == 62) && ar_at(1) != c)) break;
        eq = ar_at(1) == 61 ? 1 : 0;
        ar_p = ar_p + 1 + eq;
        r = ar_shift();
        if (c == 60) v = (eq != 0 ? v <= r : v < r) ? 1 : 0;
        else v = (eq != 0 ? v >= r : v > r) ? 1 : 0;
    }
    return v;
}

long ar_eq(void)
{
    long v = ar_cmp();
    while (ar_bad == 0) {
        int c;
        long r;
        ar_ws();
        c = ar_at(0);
        if (!((c == 61 || c == 33) && ar_at(1) == 61)) break;
        ar_p = ar_p + 2;
        r = ar_cmp();
        v = ((v == r) == (c == 61)) ? 1 : 0;
    }
    return v;
}

/* & ^ | : one loop each, the operator byte and the two it must not be part of. */
long ar_bit(int lvl)
{
    long v;
    int op = lvl == 0 ? 38 : (lvl == 1 ? 94 : 124);
    v = lvl == 0 ? ar_eq() : ar_bit(lvl - 1);
    while (ar_bad == 0) {
        ar_ws();
        if (!(ar_at(0) == op && ar_at(1) != op && ar_at(1) != 61)) break;
        ar_p = ar_p + 1;
        v = ar_apply(op, v, lvl == 0 ? ar_eq() : ar_bit(lvl - 1));
    }
    return v;
}

/* && and ||: the right operand is parsed either way, but evaluated only when
 * the left one did not already decide the answer. */
long ar_logic(int isOr)
{
    long v = isOr != 0 ? ar_logic(0) : ar_bit(2);
    while (ar_bad == 0) {
        int op = isOr != 0 ? 124 : 38;
        int skip;
        long r;
        ar_ws();
        if (!(ar_at(0) == op && ar_at(1) == op)) break;
        ar_p = ar_p + 2;
        skip = (isOr != 0) ? (v != 0) : (v == 0);
        if (skip != 0) ar_noeval = ar_noeval + 1;
        r = isOr != 0 ? ar_logic(0) : ar_bit(2);
        if (skip != 0) ar_noeval = ar_noeval - 1;
        v = (isOr != 0) ? ((v != 0 || r != 0) ? 1 : 0) : ((v != 0 && r != 0) ? 1 : 0);
    }
    return v;
}

long ar_cond(void)
{
    long c = ar_logic(1);
    long a;
    long b;
    ar_ws();
    if (ar_bad != 0 || ar_at(0) != 63) return c;
    ar_p = ar_p + 1;
    if (c == 0) ar_noeval = ar_noeval + 1;
    a = ar_comma();
    if (c == 0) ar_noeval = ar_noeval - 1;
    ar_ws();
    if (ar_bad != 0) return 0;
    if (ar_at(0) != 58) return ar_err("`:' expected for conditional expression");
    ar_p = ar_p + 1;
    if (c != 0) ar_noeval = ar_noeval + 1;
    b = ar_cond();
    if (c != 0) ar_noeval = ar_noeval - 1;
    return c != 0 ? a : b;
}

/* NAME[SUB] OP= EXPR, recognised by a scan that evaluates nothing, so the
 * subscript's side effects (a[i++]=x) happen exactly once. */
long ar_assign(void)
{
    int save;
    int q;
    int op = 0;
    int oplen = 0;
    ar_ws();
    save = ar_p;
    if (ar_idch(ar_at(0), 1) != 0) {
        int depth = 0;
        q = ar_p;
        while (q < ar_len && ar_idch((int)(unsigned char)ar_s[q], 0) != 0) q = q + 1;
        if (q < ar_len && ar_s[q] == 91) {
            while (q < ar_len) {
                if (ar_s[q] == 91) depth = depth + 1;
                if (ar_s[q] == 93) { depth = depth - 1; if (depth == 0) break; }
                q = q + 1;
            }
            q = q + 1;
        }
        while (q < ar_len && (ar_s[q] == 32 || ar_s[q] == 9 || ar_s[q] == 10)) q = q + 1;
        ar_p = q;
        if (ar_at(0) == 61 && ar_at(1) != 61) { op = 61; oplen = 1; }
        else if (ar_at(1) == 61 && (ar_at(0) == 43 || ar_at(0) == 45 || ar_at(0) == 42 || ar_at(0) == 47
                 || ar_at(0) == 37 || ar_at(0) == 38 || ar_at(0) == 94 || ar_at(0) == 124)) {
            op = ar_at(0);
            oplen = 2;
        } else if ((ar_at(0) == 60 || ar_at(0) == 62) && ar_at(1) == ar_at(0) && ar_at(2) == 61) {
            op = ar_at(0);
            oplen = 3;
        }
        ar_p = save;
    }
    if (op != 0) {
        char *nm = ar_name();
        char *key = ar_subscript(nm);
        long v;
        if (ar_bad != 0) return 0;
        ar_ws();
        ar_p = ar_p + oplen;
        v = ar_assign();
        if (ar_bad != 0) return 0;
        if (op != 61) v = ar_apply(op, ar_value(nm, key), v);
        if (ar_bad != 0) return 0;
        return ar_store(nm, key, v);
    }
    return ar_cond();
}

long ar_comma(void)
{
    long v = ar_assign();
    while (ar_bad == 0) {
        ar_ws();
        if (ar_at(0) != 44) break;
        ar_p = ar_p + 1;
        v = ar_assign();
    }
    return v;
}

/* Evaluate the whole string. An empty expression is 0, as in bash, and a plain
 * decimal number (blanks around, one sign, no leading 0) needs no parse. */
long rt_arith(char *s)
{
    char *saveS = ar_s;
    int saveLen = ar_len;
    int saveP = ar_p;
    long v = 0;
    int i = 0;
    int d0;
    int d1;
    while (s[i] == 32 || s[i] == 9 || s[i] == 10) i = i + 1;
    if (s[i] == 45 || s[i] == 43) i = i + 1;
    d0 = i;
    while (s[i] >= 48 && s[i] <= 57) i = i + 1;
    d1 = i;
    while (s[i] == 32 || s[i] == 9 || s[i] == 10) i = i + 1;
    if (s[i] == 0 && d1 > d0 && (s[d0] != 48 || d1 == d0 + 1)) return rt_str2int(s);
    ar_s = s;
    ar_len = rt_strlen(s);
    ar_p = 0;
    ar_depth = ar_depth + 1;
    ar_ws();
    if (ar_p < ar_len) {
        v = ar_comma();
        ar_ws();
        if (ar_bad == 0 && ar_p < ar_len) {
            if (ar_at(0) == 41) ar_err("arithmetic syntax error: invalid arithmetic operator");
            else ar_err("arithmetic syntax error in expression");
        }
    }
    ar_depth = ar_depth - 1;
    ar_s = saveS;
    ar_len = saveLen;
    ar_p = saveP;
    if (ar_bad != 0) {
        v = 0;
        if (ar_depth == 0) { ar_bad = 0; ar_noeval = 0; }
    }
    return v;
}

int rt_class(char *p, int ch) {
    int j = 1, neg = 0, m = 0, first = 1;
    int c0 = (int)(unsigned char)p[1];
    if (c0 == 33 || c0 == 94) { neg = 1; j = 2; }
    while (1) {
        int cj = (int)(unsigned char)p[j];
        if (cj == 0) return -1;
        if (cj == 93 && first == 0) break;
        first = 0;
        if (cj == 91 && (int)(unsigned char)p[j + 1] == 58) {
            /* [:name:] - find the closing ":]", then look the name up */
            int d = 0;
            int closed = 0;
            while (1) {
                int hc = (int)(unsigned char)p[j + 2 + d];
                if (hc == 0) break;
                if (hc == 58 && (int)(unsigned char)p[j + 3 + d] == 93) { closed = 1; break; }
                d = d + 1;
            }
            if (closed != 0) {
                char *nm = rt_bump(d + 1);
                int t = 0;
                int cid;
                while (t < d) { nm[t] = p[j + 2 + t]; t = t + 1; }
                nm[d] = 0;
                cid = rt_clsid(nm);
                if (cid != 0 && re_ctype(cid, ch) != 0) m = 1;
                j = j + d + 4;
                continue;
            }
            /* no ":]" at all: fall through and treat "[" as an ordinary member */
        }
        /* A member - and a range endpoint - is a CHARACTER, not a byte; `ch` is
         * the subject's codepoint (rt_glob decodes it). See rt_cpw. */
        int mw = rt_cpw(&p[j]);
        int mc = rt_cpat(&p[j]);
        int cn = (int)(unsigned char)p[j + mw];
        int cn2 = 0;
        if (cn == 45) cn2 = (int)(unsigned char)p[j + mw + 1];
        if (cn == 45 && cn2 != 0 && cn2 != 93) {
            int hw = rt_cpw(&p[j + mw + 1]);
            int hc = rt_cpat(&p[j + mw + 1]);
            if (ch >= mc && ch <= hc) m = 1;
            j = j + mw + 1 + hw;
        } else {
            if (ch == mc) m = 1;
            j = j + mw;
        }
    }
    int fin = neg ? 1 - m : m;
    return ((j + 1) * 2) | fin;
}



/* ---- substring helpers for the ${...} family -------------------------------
 * They are placed after rt_glob because every one of them drives the glob
 * matcher over candidate substrings; rt_substr materializes each candidate.
 */

/* char *rt_substr(char *s, int off, int n): n < 0 means "to the end". off and
 * n are clamped. */
char *rt_substr(char *s, int off, int n)
{
    int len;
    int o;
    int rest;
    int nn;
    int i;
    char *dst;

    len = rt_strlen(s);
    o = off < 0 ? 0 : off;
    o = o > len ? len : o;
    rest = len - o;
    nn = n < 0 ? rest : n;
    nn = nn > rest ? rest : nn;
    /* The WHOLE of s is a very common "slice": ${v#} / ${v%} walk it, rt_strip
     * builds it on every trial length, and rt_getfield takes it for a one-field
     * list. Arena strings are immutable (see rt_strcat), so s itself is the
     * answer. The len > 0 guard keeps `unset_marker` from ever being returned as
     * a computed value - rt_nounset compares that pointer by identity. */
    if (o == 0 && nn == len && len > 0) return s;
    dst = rt_bump(nn + 1);
    i = 0;
    while (i < nn) {
        dst[i] = s[o + i];
        i = i + 1;
    }
    dst[nn] = 0;
    return dst;
}

/* char *rt_csubstr(char *s, int off, int n): rt_substr with off and n in
 * CHARACTERS, which is what ${s:off:n} slices in bash. rt_substr itself stays
 * byte-based - every pattern helper drives it over byte positions. */
char *rt_csubstr(char *s, int off, int n)
{
    int bo;
    int bend;
    int byteN;

    bo = rt_charoff(s, off < 0 ? 0 : off);
    /* n < 0 means "to the end"; otherwise the end is off + n characters in. */
    bend = rt_charoff(s, off + (n < 0 ? 0 : n));
    byteN = n < 0 ? -1 : bend - bo;
    byteN = byteN < 0 ? -1 : byteN;
    return rt_substr(s, bo, byteN);
}

/* ---- the extended globs @(a|b) ?(a|b) *(a|b) +(a|b) !(a|b) -----------------
 *
 * Two scanners plus rt_eg. Both scanners count nested parens and step over a
 * backslash escape, so `@(a\)b|c)` splits where bash splits it.
 */

/* int rt_egclose(char *ecq): ecq points at "("; the offset from ecq of the
 * MATCHING ")", or -1. */
int rt_egclose(char *ecq)
{
    int i;
    int d;
    int c;

    i = 0;
    d = 0;
    while (1) {
        c = (int)(unsigned char)ecq[i];
        if (c == 0) return -1;
        if (c == 92 && (int)(unsigned char)ecq[i + 1] != 0) {
            i = i + 2;
            continue;
        }
        if (c == 40) {
            d = d + 1;
        } else if (c == 41) {
            d = d - 1;
            if (d == 0) return i;
        }
        i = i + 1;
    }
}

/* int rt_egalt(char *eaq, int eaa): eaq points at "(" and eaa is the offset of
 * an alternative's first byte; the offset of its terminator - a "|" at depth 0,
 * or the group's own ")". */
int rt_egalt(char *eaq, int eaa)
{
    int i;
    int d;
    int c;

    i = eaa;
    d = 0;
    while (1) {
        c = (int)(unsigned char)eaq[i];
        if (c == 0) return i;
        if (c == 92 && (int)(unsigned char)eaq[i + 1] != 0) {
            i = i + 2;
            continue;
        }
        if (c == 40) {
            d = d + 1;
        } else if (c == 41) {
            if (d == 0) return i;
            d = d - 1;
        } else if (c == 124 && d == 0) {
            return i;
        }
        i = i + 1;
    }
}
/* ---- glob (pathname-style) matching: * ? [...] and a backslash escape ------
 * The same matcher backs case and [[ == ]], exactly like the shell. Both
 * functions walk raw char* pointers; "*" backtracks by CALLING rt_glob again,
 * so the call stack is the backtracking stack.
 */

/* rt_glob and rt_eg are mutually recursive; this prototype is the forward
 * declaration, with the POINTER parameters the definition has
 * (c-to-llvm-ir.abnf's noteParamCts; pinned by SECTION 48 of
 * tests/c-test-full.c).
 */
int rt_eg(char *p, char *s, int star);

/* int rt_glob(char *p, char *s): 1 when the whole subject matches the whole
 * pattern. */
int rt_glob(char *p, char *s)
{
    char *pp;
    char *sp;
    char *pcur;
    char *sk;
    char *pafter;
    char *scur;
    char *scur2;
    int pc;
    int isPfx;
    int nextc;
    int sat;
    int pac;
    int rec;
    int kc;
    int sc;
    int cr;
    int matched;
    int off;
    int nc;
    int isEsc;
    int scw;
    int pcw;

    pp = p;
    sp = s;
    while (1) {
        pcur = pp;
        pc = (int)(unsigned char)pcur[0];
        if (pc == 0) {
            sat = (int)(unsigned char)sp[0];
            if (sat == 0) return 1;
            return 0;
        }
        /* An extglob head has to be recognised BEFORE the plain "*" and "?"
         * arms; all five prefixes are only that when "(" is the next byte. */
        isPfx = (pc == 64) || (pc == 63) || (pc == 42) || (pc == 43) || (pc == 33);
        nextc = (int)(unsigned char)pcur[1];
        if (isPfx && nextc == 40) return rt_eg(pcur, sp, 0);

        if (pc == 42) {
            /* skip a run of "*" */
            while (1) {
                sk = pp;
                if ((int)(unsigned char)sk[0] != 42) break;
                pp = sk + 1;
            }
            pafter = pp;
            pac = (int)(unsigned char)pafter[0];
            if (pac == 0) return 1;
            while (1) {
                scur = sp;
                rec = rt_glob(pafter, scur);
                if (rec != 0) return 1;
                kc = (int)(unsigned char)scur[0];
                if (kc == 0) return 0;
                sp = scur + rt_cpw(scur);
            }
        }

        scur2 = sp;
        if ((int)(unsigned char)scur2[0] == 0) return 0;
        /* "?" matches one CHARACTER and a bracket expression holds characters,
         * so the subject is decoded once here and every arm below steps by scw. */
        scw = rt_cpw(scur2);
        sc = rt_cpat(scur2);

        if (pc == 63) {
            pp = pcur + 1;
            sp = scur2 + scw;
            continue;
        }
        if (pc == 91) {
            cr = rt_class(pcur, sc);
            if (cr >= 0) {
                matched = cr & 1;
                off = cr / 2;
                if (matched == 0) return 0;
                pp = pcur + off;
                sp = scur2 + scw;
                continue;
            }
            /* cr < 0: not a bracket expression after all - fall through to
             * the literal arm. */
        } else {
            nc = (int)(unsigned char)pcur[1];
            isEsc = (pc == 92) && (nc != 0);
            if (isEsc) {
                pcw = rt_cpw(pcur + 1);
                if (rt_cpat(pcur + 1) != sc) return 0;
                pp = pcur + 1 + pcw;
                sp = scur2 + scw;
                continue;
            }
        }
        /* literal */
        pcw = rt_cpw(pcur);
        if (rt_cpat(pcur) != sc) return 0;
        pp = pcur + pcw;
        sp = scur2 + scw;
    }
}

/* int rt_eg(char *egp, char *egs, int egstar): egp points at the prefix
 * character of an extglob and egs at the subject; 1 when egp (group AND
 * everything after it) matches all of egs.
 *
 * Each alternative is tried against every PREFIX of the subject and the rest of
 * the pattern against what is left - the same "split and recurse" shape rt_glob
 * uses for "*", with the call stack as the backtracking stack. `egstar` is the
 * +(...) continuation: after one mandatory occurrence the group carries on with
 * *(...) semantics, and because a repeat must consume at least one byte the
 * recursion is finite.
 *   @(A|B)  exactly one alternative        !(A|B)  a prefix NO alternative matches
 *   ?(A|B)  zero or one                    *(A|B)  zero or more
 *   +(A|B)  one or more
 */
int rt_eg(char *egp, char *egs, int egstar)
{
    int kind;
    char *open;
    int clo;
    char *rest;
    int slen;
    int isBang;
    int isAt;
    int isQ;
    int zeroOk;
    int one1;
    int minK;
    int a;
    int aend;
    char *alt;
    int k;
    char *pre;
    char *tail;
    char *t3;
    int rr;
    int ba;
    int be;
    char *balt;
    char *bpre;
    int any;
    char *btail;
    int g;

    kind = egstar != 0 ? 42 : (int)(unsigned char)egp[0];
    open = egp + 1;
    clo = rt_egclose(open);
    if (clo < 0) return 0;
    rest = egp + (clo + 2);
    slen = rt_strlen(egs);
    isBang = kind == 33;
    isAt = kind == 64;
    isQ = kind == 63;
    zeroOk = isQ || (kind == 42);
    one1 = isAt || isQ;              /* exactly one occurrence, so k may be 0 */

    if (isBang) {
        /* ! : a prefix that NO alternative matches, with the rest matching
         * what is left. */
        k = 0;
        while (k < slen + 1) {
            any = 0;
            ba = 1;
            while (ba < clo) {
                be = rt_egalt(open, ba);
                balt = rt_substr(open + ba, 0, be - ba);
                bpre = rt_substr(egs, 0, k);
                any = rt_glob(balt, bpre) != 0 ? 1 : any;
                if ((int)(unsigned char)open[be] == 41) break;
                ba = be + 1;
            }
            btail = egs + k;
            /* rt_glob runs even when `any` is already set. */
            g = rt_glob(rest, btail);
            if (any == 0 && g != 0) return 1;
            k = k + 1;
        }
        return 0;
    }

    /* @ ? * + : zero occurrences first (? and *), then one alternative at a
     * time. */
    if (zeroOk) {
        if (rt_glob(rest, egs) != 0) return 1;
    }
    minK = one1 ? 0 : 1;
    a = 1;
    while (a < clo) {
        aend = rt_egalt(open, a);
        alt = rt_substr(open + a, 0, aend - a);
        k = minK;
        while (k < slen + 1) {
            pre = rt_substr(egs, 0, k);
            if (rt_glob(alt, pre) != 0) {
                if (one1) {
                    tail = egs + k;
                    if (rt_glob(rest, tail) != 0) return 1;
                } else {
                    /* a * / + repeat must consume something, or the recursion
                     * would not terminate. */
                    t3 = egs + k;
                    rr = rt_eg(egp, t3, 1);
                    if (k > 0 && rr != 0) return 1;
                }
            }
            k = k + 1;
        }
        if ((int)(unsigned char)open[aend] == 41) break;
        a = aend + 1;
    }
    return 0;
}

/* int rt_matchlen(char *p, char *s, int i): end index of the LONGEST match of
 * p starting at i, or -1. */
int rt_matchlen(char *p, char *s, int i)
{
    int j;
    char *cand;

    j = rt_strlen(s);
    while (1) {
        if (j < i) return -1;
        cand = rt_substr(s, i, j - i);
        if (rt_glob(p, cand) != 0) return j;
        j = j - 1;
    }
}

/* int rt_matchend(char *p, char *s): start index of the LONGEST match of p
 * that reaches the end of s, or -1. */
int rt_matchend(char *p, char *s)
{
    int len;
    int i;
    char *cand;

    len = rt_strlen(s);
    i = 0;
    while (1) {
        if (i > len) return -1;
        cand = rt_substr(s, i, -1);
        if (rt_glob(p, cand) != 0) return i;
        i = i + 1;
    }
}

/* ---- ${v#p} / ${v/p/r} / case conversion / quoting / read ----------------- */

/* The grammar loads `read_eof` by name for the `read` builtin's status. */
int read_eof;

/* rt_strip: ${v#p} / ${v##p} / ${v%p} / ${v%%p}; mode 0 = #, 1 = ##, 2 = %, 3 = %%.
 * A trial length t walks 0..len; k is t for the ascending modes (# and %%) and
 * len - t for the descending ones (## and %), so the first hit is the shortest
 * or the longest match as required. */
char *rt_strip(char *s, char *p, int mode) {
    int len = rt_strlen(s);
    int isPre = (mode == 0) || (mode == 1);
    int asc = (mode == 0) || (mode == 3);
    int t = 0;
    while (t <= len) {
        int k = asc ? t : len - t;
        char *pre = rt_substr(s, 0, k);
        char *suf = rt_substr(s, k, -1);
        char *cand = isPre ? pre : suf;
        if (rt_glob(p, cand) != 0) {
            char *rpre = rt_substr(s, k, -1);
            char *rsuf = rt_substr(s, 0, k);
            return isPre ? rpre : rsuf;
        }
        t = t + 1;
    }
    return s;
}

/* rt_replace: ${v/p/r} and friends; mode 0 = first, 1 = all, 2 = anchored at the
 * start (#), 3 = anchored at the end (%). */
char *rt_replace(char *s, char *p, char *r, int mode) {
    int len = rt_strlen(s);
    char *out = empty;
    int i = 0;
    int done = 0;
    if (mode >= 2) {
        if (mode == 2) {
            int jj = rt_matchlen(p, s, 0);
            if (jj < 0) return s;
            return rt_strcat(r, rt_substr(s, jj, -1));
        }
        int ii = rt_matchend(p, s);
        if (ii < 0) return s;
        return rt_strcat(rt_substr(s, 0, ii), r);
    }
    while (1) {
        if (i > len || done != 0) return out;
        int j = rt_matchlen(p, s, i);
        if (j > i) {
            /* a real (non empty) match: append the replacement and jump past it */
            out = rt_strcat(out, r);
            i = j;
            if (mode == 0) {
                out = rt_strcat(out, rt_substr(s, j, -1));
                done = 1;
            }
        } else {
            /* no match here: carry one character across */
            out = rt_strcat(out, rt_substr(s, i, 1));
            i = i + 1;
        }
    }
    return out;
}

#include "case-simple.h"

/* rt_case: ${v^} / ${v^^} / ${v,} / ${v,,}; mode 0 = ^, 1 = ^^, 2 = ",", 3 = ",,".
 *
 * It walks CODEPOINTS, not bytes, and maps each one as towupper / towlower do: the
 * Unicode SIMPLE mapping (case-simple.h), one character for one, so "ß" stays and
 * "İ" lowers to "i". A mapped character can be one byte longer in UTF-8
 * (U+0250 -> U+2C6F), hence the doubled buffer. */
char *rt_case(char *s, int mode, char *pat) {
    int len = rt_strlen(s);
    char *dst = rt_bump(len + len + 1);
    int up = mode < 2;
    int all = (mode == 1) || (mode == 3);
    int i = 0;
    int o = 0;
    int first = 1;
    int nopat = pat[0] == 0;
    char one[8];
    while (i < len) {
        int c0 = (int)(unsigned char)s[i];
        int touch = all || (first != 0);
        int w = 1;
        int t = 0;
        long cp = c0;
        if (c0 >= 240) { w = 4; cp = c0 & 7; }
        else if (c0 >= 224) { w = 3; cp = c0 & 15; }
        else if (c0 >= 192) { w = 2; cp = c0 & 31; }
        if (i + w > len) { w = len - i; touch = 0; }
        /* THE PATTERN OPERAND. ${v^^[bn]} converts only the characters the glob
         * matches (bash 5.3: banana -> BaNaNa). The pattern is matched against
         * ONE character at a time, so the subject is the current
         * codepoint's bytes in a stack buffer - never the arena, or a long value
         * would allocate one temporary per character. An ABSENT pattern is ""
         * and means "every character"; rt_glob("", "a") is false, so the empty
         * case is tested for rather than handed to the matcher. */
        while (t < w) {
            one[t] = s[i + t];
            if (t > 0) { cp = (cp << 6) | ((int)(unsigned char)s[i + t] & 63); }
            t = t + 1;
        }
        one[w] = 0;
        if (touch != 0 && nopat == 0) { touch = rt_glob(pat, one) != 0; }
        if (touch != 0 && (c0 < 128 || c0 >= 192)) { cp = case_map(cp, up); }
        else { cp = -1; }
        if (cp < 0) {
            t = 0;
            while (t < w) {
                dst[o + t] = one[t];
                t = t + 1;
            }
            o = o + w;
        } else if (cp < 128) {
            dst[o] = (char)cp;
            o = o + 1;
        } else if (cp < 2048) {
            dst[o] = (char)(192 | (cp >> 6));
            dst[o + 1] = (char)(128 | (cp & 63));
            o = o + 2;
        } else if (cp < 65536) {
            dst[o] = (char)(224 | (cp >> 12));
            dst[o + 1] = (char)(128 | ((cp >> 6) & 63));
            dst[o + 2] = (char)(128 | (cp & 63));
            o = o + 3;
        } else {
            dst[o] = (char)(240 | (cp >> 18));
            dst[o + 1] = (char)(128 | ((cp >> 12) & 63));
            dst[o + 2] = (char)(128 | ((cp >> 6) & 63));
            dst[o + 3] = (char)(128 | (cp & 63));
            o = o + 4;
        }
        i = i + w;
        first = 0;
    }
    dst[o] = 0;
    return dst;
}

/* rt_shquote: the ${v@Q} / ${v@K} / ${v@k} form - sq quoted so the shell would
 * read it back unchanged. Single quotes normally, with an embedded ' spelled
 * '\'', and the $'...' form as soon as the value holds a control character. */
char *rt_shquote(char *sq) {
    int len = rt_strlen(sq);
    /* 4 bytes per input byte is the worst case ('\'' and \xNN alike), plus $'' and NUL. */
    char *dst = rt_bump(len * 4 + 8);
    int o = 0;
    int i = 0;
    int ctl = 0;
    /* pass 1: is there a control character? */
    while (i < len) {
        int c0 = (int)(unsigned char)sq[i];
        int isCtl = (c0 < 32) || (c0 == 127);
        ctl = isCtl ? 1 : ctl;
        i = i + 1;
    }
    if (ctl != 0) {
        dst[o] = (char)36; o = o + 1;
        dst[o] = (char)39; o = o + 1;
        int k = 0;
        while (k < len) {
            int c0 = (int)(unsigned char)sq[k];
            if (c0 == 39) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)39; o = o + 1;
            } else if (c0 == 92) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)92; o = o + 1;
            } else if (c0 == 7) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)97; o = o + 1;
            } else if (c0 == 8) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)98; o = o + 1;
            } else if (c0 == 9) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)116; o = o + 1;
            } else if (c0 == 10) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)110; o = o + 1;
            } else if (c0 == 11) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)118; o = o + 1;
            } else if (c0 == 12) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)102; o = o + 1;
            } else if (c0 == 13) {
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)114; o = o + 1;
            } else if ((c0 < 32) || (c0 == 127)) {
                /* bash 5.3.15 writes OCTAL here - ${v@Q} of $'\001' is $'\001',
                 * not $'\x01'. */
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)(48 + (c0 / 64)); o = o + 1;
                dst[o] = (char)(48 + ((c0 / 8) % 8)); o = o + 1;
                dst[o] = (char)(48 + (c0 % 8)); o = o + 1;
            } else {
                dst[o] = (char)c0; o = o + 1;
            }
            k = k + 1;
        }
        dst[o] = (char)39; o = o + 1;
    } else {
        dst[o] = (char)39; o = o + 1;
        int j = 0;
        while (j < len) {
            int c0 = (int)(unsigned char)sq[j];
            if (c0 == 39) {
                dst[o] = (char)39; o = o + 1;
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)39; o = o + 1;
                dst[o] = (char)39; o = o + 1;
            } else {
                dst[o] = (char)c0; o = o + 1;
            }
            j = j + 1;
        }
        dst[o] = (char)39; o = o + 1;
    }
    dst[o] = 0;
    return dst;
}

/* the value of one hex digit, or -1. */
int rt_hexval(int hc) {
    if (hc >= 48 && hc <= 57) return hc - 48;
    if (hc >= 97 && hc <= 102) return hc - 87;
    if (hc >= 65 && hc <= 70) return hc - 55;
    return -1;
}

/* rt_ansic: the ${v@E} form - the $'...' escapes decoded at RUN time.
 *
 * \NNN (one to THREE octal digits, total, so $'\0101' is \b then "1") and \xHH
 * (one or two hex digits) are decoded, so ${v@E} inverts ${v@Q}. A \x with no
 * hex digit after it stays literal, as bash leaves it. */
char *rt_ansic(char *ac) {
    int len = rt_strlen(ac);
    char *dst = rt_bump(len + 1);
    int i = 0;
    int o = 0;
    while (i < len) {
        int c0 = (int)(unsigned char)ac[i];
        int c1 = (int)(unsigned char)ac[(i + 1 < len) ? i + 1 : i];
        if ((c0 == 92) && (i + 1 < len) && (c1 >= 48) && (c1 <= 55)) {
            int v = 0;
            int n = 0;
            i = i + 1;
            while (n < 3 && i < len) {
                int d = (int)(unsigned char)ac[i];
                if (d < 48 || d > 55) break;
                v = v * 8 + (d - 48);
                i = i + 1;
                n = n + 1;
            }
            dst[o] = (char)(v % 256);
            o = o + 1;
        } else if ((c0 == 92) && (i + 1 < len) && (c1 == 120) &&
                   (rt_hexval((int)(unsigned char)ac[(i + 2 < len) ? i + 2 : i]) >= 0) &&
                   (i + 2 < len)) {
            int v = 0;
            int n = 0;
            i = i + 2;
            while (n < 2 && i < len) {
                int d = rt_hexval((int)(unsigned char)ac[i]);
                if (d < 0) break;
                v = v * 16 + d;
                i = i + 1;
                n = n + 1;
            }
            dst[o] = (char)v;
            o = o + 1;
        } else if ((c0 == 92) && (i + 1 < len)) {
            int mapped = c1;
            if (c1 == 97) mapped = 7;
            if (c1 == 98) mapped = 8;
            if (c1 == 116) mapped = 9;
            if (c1 == 110) mapped = 10;
            if (c1 == 118) mapped = 11;
            if (c1 == 102) mapped = 12;
            if (c1 == 114) mapped = 13;
            if (c1 == 101) mapped = 27;
            if (c1 == 69) mapped = 27;
            dst[o] = (char)mapped;
            o = o + 1;
            i = i + 2;
        } else {
            dst[o] = (char)c0;
            o = o + 1;
            i = i + 1;
        }
    }
    dst[o] = 0;
    return dst;
}

/* rt_haschar: is ch one of the characters of s? */
int rt_haschar(char *s, int ch) {
    int i = 0;
    while (1) {
        int c0 = (int)(unsigned char)s[i];
        if (c0 == 0) return 0;
        if (c0 == ch) return 1;
        i = i + 1;
    }
    return 0;
}

/* rt_read_line: the next line of stdin_buf (without its newline), advancing
 * stdin_pos. Returns a null-marked empty string and leaves the position at the
 * end when there is nothing left; read_eof answers whether that happened. */
char *rt_read_line(int u) {
    char *buf = stdin_buf;
    int pos = stdin_pos;
    int len = rt_strlen(buf);
    read_eof = 0;
    if (pos >= len) {
        read_eof = 1;
        return empty;
    }
    int k = pos;
    while (1) {
        int atEnd = k >= len;
        char ch = buf[k];
        int isNL = ch == 10;
        if (atEnd || isNL) break;
        k = k + 1;
    }
    char *line = rt_substr(buf, pos, k - pos);
    stdin_pos = (k < len) ? k + 1 : k;
    return line;
}

/* ---- fields, padding, escapes, field lists -------------------------------- */

/* rt_nifs: how many fields `l` splits into under the POSIX delimiter rule (the
 * one rt_splitifs walks and bash-interpreter.abnf's splitIFS mirrors). Only
 * rt_field needs it, to tell "one field left" from "several". */
int rt_nifs(char *l, char *ifs) {
    int len = rt_strlen(l);
    int i = 0;
    int n = 0;
    while (i < len && rt_ifsws(ifs, (int)(unsigned char)l[i]) != 0) { i = i + 1; }
    while (i < len) {
        while (i < len && rt_haschar(ifs, (int)(unsigned char)l[i]) == 0) { i = i + 1; }
        n = n + 1;
        while (i < len && rt_ifsws(ifs, (int)(unsigned char)l[i]) != 0) { i = i + 1; }
        if (i < len && rt_haschar(ifs, (int)(unsigned char)l[i]) != 0) {
            i = i + 1;
            while (i < len && rt_ifsws(ifs, (int)(unsigned char)l[i]) != 0) { i = i + 1; }
        }
    }
    return n;
}

/* rt_field: field `idx` of line `l` under the POSIX delimiter rule; with
 * rest != 0 what the LAST name of a `read` gets, which is not the same thing.
 *
 * Empty fields count (`IFS=: read -r a b <<< ":x:"` leaves a empty and gives b
 * "x"). Per bash 5.3.15, the remainder has three arms:
 *
 *   no field there      ""
 *   EXACTLY ONE left    that field - no delimiters, no whitespace, so
 *                       `IFS=: read -r a b <<< "p:q:"` gives b "q"
 *   TWO OR MORE left    the RAW text from that field's start to the end of the
 *                       line, less trailing IFS WHITESPACE only, so
 *                       `IFS=: read -r a b <<< ":a:b:"` gives b "a:b:" and
 *                       `read -r a b <<< "  x   y   z  "` gives b "y   z"
 *
 * bash-interpreter.abnf's readRest is the same three arms. */
char *rt_field(char *l, char *ifs, int idx, int rest) {
    int len = rt_strlen(l);
    int i = 0;      /* scan index          */
    int n = 0;      /* field number        */
    int s = 0;      /* current field start */
    int e = 0;      /* current field end   */
    int t = 0;
    int left = 0;
    if (rest != 0) left = rt_nifs(l, ifs) - idx;
    while (i < len && rt_ifsws(ifs, (int)(unsigned char)l[i]) != 0) { i = i + 1; }
    while (i < len) {
        s = i;
        while (i < len && rt_haschar(ifs, (int)(unsigned char)l[i]) == 0) { i = i + 1; }
        e = i;
        if (n == idx) {
            if (rest == 0) return rt_substr(l, s, e - s);
            if (left <= 1) return rt_substr(l, s, e - s);
            t = len;
            while (t > s) {
                if (rt_ifsws(ifs, (int)(unsigned char)l[t - 1]) == 0) break;
                t = t - 1;
            }
            return rt_substr(l, s, t - s);
        }
        n = n + 1;
        while (i < len && rt_ifsws(ifs, (int)(unsigned char)l[i]) != 0) { i = i + 1; }
        if (i < len && rt_haschar(ifs, (int)(unsigned char)l[i]) != 0) {
            i = i + 1;
            while (i < len && rt_ifsws(ifs, (int)(unsigned char)l[i]) != 0) { i = i + 1; }
        }
    }
    return &empty[0];
}

/* rt_pad: printf's field padding. */
char *rt_pad(char *s, int w, int left, int zero) {
    char *v = s;
    while (rt_strlen(v) < w) {
        char *padCh;
        char *a;
        char *b;
        if (zero != 0) padCh = "0"; else padCh = " ";
        a = rt_strcat(v, " ");
        b = rt_strcat(padCh, v);
        if (left != 0) v = a; else v = b;
    }
    return v;
}

/* rt_radix: printf's %x / %X / %o / %u - the operand as an UNSIGNED 64-bit
 * value in `base`, with the # prefix when `alt` and the value is not zero.
 *
 * The C subset has no unsigned 64-bit type, so the value is carried as four
 * 16-bit limbs and divided long-hand: with r < base <= 16, r * 65536 + limb
 * stays below 2^20. */
char *rt_radix(char *s, int base, int upper, int alt) {
    long n = rt_str2int(s);
    int lb[4];
    char tmp[72];
    int t = 71;
    char *dst;
    int d;
    int j;
    int k;
    int pre = 0;
    int nz = 0;
    k = 0;
    while (k < 4) {
        lb[3 - k] = (int)((n >> (16 * k)) & 65535);
        if (lb[3 - k] != 0) nz = 1;
        k = k + 1;
    }
    if (nz == 0) {
        dst = rt_bump(2);
        dst[0] = 48;
        dst[1] = 0;
        return dst;
    }
    while (nz != 0) {
        int r = 0;
        int dg;
        nz = 0;
        k = 0;
        while (k < 4) {
            int cur = r * 65536 + lb[k];
            lb[k] = cur / base;
            r = cur % base;
            if (lb[k] != 0) nz = 1;
            k = k + 1;
        }
        dg = r;
        if (dg < 10) tmp[t] = (char)(48 + dg);
        else if (upper != 0) tmp[t] = (char)(55 + dg);
        else tmp[t] = (char)(87 + dg);
        t = t - 1;
    }
    if (alt != 0 && base == 8) pre = 1;
    if (alt != 0 && base == 16) pre = 2;
    dst = rt_bump(pre + (71 - t) + 1);
    d = 0;
    if (pre == 1) { dst[0] = 48; d = 1; }
    if (pre == 2) {
        dst[0] = 48;
        if (upper != 0) dst[1] = 88; else dst[1] = 120;
        d = 2;
    }
    j = t + 1;
    while (j <= 71) { dst[d] = tmp[j]; d = d + 1; j = j + 1; }
    dst[d] = 0;
    return dst;
}

/* rt_intprec: printf's PRECISION on an integer - a minimum DIGIT count, which a
 * "-" sign is not part of. `%.3d` of 7 is 007 and `%.0d` of 0 is empty. */
char *rt_intprec(char *s, int prec) {
    int neg = s[0] == 45;
    int off = 0;
    int len;
    int pad;
    char *dst;
    int d;
    int i;
    if (neg != 0) off = 1;
    len = rt_strlen(s) - off;
    if (prec == 0 && len == 1 && s[off] == 48) len = 0;
    pad = prec - len;
    if (pad < 0) pad = 0;
    dst = rt_bump(off + pad + len + 1);
    d = 0;
    if (neg != 0) { dst[0] = 45; d = 1; }
    i = 0;
    while (i < pad) { dst[d] = 48; d = d + 1; i = i + 1; }
    i = 0;
    while (i < len) { dst[d] = s[off + i]; d = d + 1; i = i + 1; }
    dst[d] = 0;
    return dst;
}

/* rt_pfq: printf %q. This is bash's BACKSLASH form, NOT ${v@Q}'s single-quoted
 * one - `printf %q "a b"` is `a\ b` while `${v@Q}` is `'a b'`. An empty operand
 * is '' and a value holding a control character falls back to rt_shquote, which
 * already emits the $'...' spelling. Mirrors printfQuote in
 * languages/bash-interpreter.abnf. */
char *rt_pfq(char *s) {
    int len = rt_strlen(s);
    int i;
    int ctl = 0;
    char *dst;
    int o;
    if (len == 0) {
        dst = rt_bump(3);
        dst[0] = 39;
        dst[1] = 39;
        dst[2] = 0;
        return dst;
    }
    i = 0;
    while (i < len) {
        int c = (int)(unsigned char)s[i];
        if (c < 32 || c == 127) ctl = 1;
        i = i + 1;
    }
    if (ctl != 0) return rt_shquote(s);
    dst = rt_bump(2 * len + 1);
    o = 0;
    i = 0;
    while (i < len) {
        int c = (int)(unsigned char)s[i];
        int safe = 0;
        if (c >= 48 && c <= 57) safe = 1;
        if (c >= 65 && c <= 90) safe = 1;
        if (c >= 97 && c <= 122) safe = 1;
        if (c == 37 || c == 43 || c == 44 || c == 45 || c == 46 || c == 47) safe = 1;
        if (c == 58 || c == 61 || c == 64 || c == 95) safe = 1;
        if (c > 127) safe = 1;
        if (safe == 0) { dst[o] = 92; o = o + 1; }
        dst[o] = s[i];
        o = o + 1;
        i = i + 1;
    }
    dst[o] = 0;
    return dst;
}

/* rt_unescape: echo -e / printf backslash escapes. */
char *rt_unescape(char *s) {
    int len = rt_strlen(s);
    char *dst = rt_bump(len + 1);
    int i = 0, o = 0;
    while (1) {
        int c0 = (int)(unsigned char)s[i];
        int isEsc = (c0 == 92) && (i + 1 < len);
        int c;
        if (c0 == 0) break;
        if (isEsc) {
            int n1 = (int)(unsigned char)s[i + 1];
            c = n1;
            if (n1 == 110) c = 10;   /* n */
            if (n1 == 116) c = 9;    /* t */
            if (n1 == 114) c = 13;   /* r */
            if (n1 == 97) c = 7;     /* a */
            if (n1 == 98) c = 8;     /* b */
            if (n1 == 102) c = 12;   /* f */
            if (n1 == 118) c = 11;   /* v */
            if (n1 == 101) c = 27;   /* e */
            if (n1 == 48) c = 0;     /* 0 */
            i = i + 2;
        } else {
            c = c0;
            i = i + 1;
        }
        dst[o] = (char)c;
        o = o + 1;
    }
    dst[o] = 0;
    return dst;
}

/* ---- field lists ----------------------------------------------------------
 * A plain string is ONE field; a string that starts with "\x02" is a list, each
 * field followed by "\x01". */

int rt_nfields(char *v) {
    int c0 = (int)(unsigned char)v[0];
    int i = 1, n = 0;
    int len = rt_strlen(v);
    if (c0 != 2) return 1;
    while (i < len) {
        int ch = (int)(unsigned char)v[i];
        if (ch == 1) n = n + 1;
        i = i + 1;
    }
    return n;
}

char *rt_getfield(char *v, int k) {
    int c0 = (int)(unsigned char)v[0];
    int i = 1, n = 0, s = 1;
    int len = rt_strlen(v);
    if (c0 != 2) {
        if (k == 0) return v;
        return &empty[0];
    }
    while (i < len) {
        int ch = (int)(unsigned char)v[i];
        if (ch == 1) {
            if (n == k) return rt_substr(v, s, i - s);
            n = n + 1;
            s = i + 1;
        }
        i = i + 1;
    }
    return &empty[0];
}

/* AN UNQUOTED FIELD LIST ("$@", ${a[@]} and their transforms outside quotes) IS
 * SPLIT AGAIN, AS ONE VALUE: bash 5.3 joins the fields with the first IFS
 * character and splits the result like any other expansion, so
 * `set -- "a b" c; printf '<%s>' $@` is three fields, and under IFS=: the list
 * ("a" "" "b") is a, "", b. A null IFS cannot join anything: each non-empty
 * field then stands alone and the empty ones vanish. Answers the joined string
 * for the caller to split, a field list for a null IFS, and a plain string
 * unchanged. bash-interpreter.abnf's uqList is the same rule. */
char *rt_uqlist(char *l, char *ifs) {
    int len = rt_strlen(l);
    int sep = (int)(unsigned char)ifs[0];
    int i = 1;
    int o = 0;
    char *dst;
    if ((int)(unsigned char)l[0] != 2 || rt_isfin(l) != 0) return l;
    if (len < 2) return sep == 0 ? "\x02" : &empty[0];
    dst = rt_bump(len + 1);
    if (sep == 0) {
        dst[0] = 2;
        o = 1;
        while (i < len) {
            int ch = (int)(unsigned char)l[i];
            int prev = (int)(unsigned char)dst[o - 1];
            if (ch != 1 || (prev != 1 && prev != 2)) { dst[o] = l[i]; o = o + 1; }
            i = i + 1;
        }
    } else {
        while (i < len) {
            if ((int)(unsigned char)l[i] != 1) { dst[o] = l[i]; o = o + 1; }
            else if (i < len - 1) { dst[o] = ifs[0]; o = o + 1; }
            i = i + 1;
        }
    }
    dst[o] = 0;
    return dst;
}

/* rt_wordjoin: a field list flattened back to one string, fields space-joined. */
char *rt_wordjoin(char *v) {
    int c0 = (int)(unsigned char)v[0];
    int n = rt_nfields(v);
    char *op = &empty[0];
    int i = 0;
    if (c0 != 2) return v;
    while (i < n) {
        char *cur = op;
        char *withsp = rt_strcat(cur, " ");
        char *fld;
        if (i > 0) cur = withsp;
        fld = rt_getfield(v, i);
        op = rt_strcat(cur, fld);
        i = i + 1;
    }
    return op;
}

/* One IFS WHITESPACE character: space, tab or newline AND a member of IFS. See
 * rt_splitifs. */
int rt_ifsws(char *ifs, int ch) {
    if (ch != 32 && ch != 9 && ch != 10) return 0;
    return rt_haschar(ifs, ch) != 0 ? 1 : 0;
}

char *rt_splitifs(char *v, char *ifs) {
    int len;
    int q = 0;
    int nf = 0;
    int tot = 0;
    int i;
    int o;
    int pass;
    char *dst;
    dst = &empty[0];
    /* A field list is split as the one value rt_uqlist makes of it; under a null
     * IFS that is a list again, already split. */
    if ((int)(unsigned char)v[0] == 2) {
        v = rt_uqlist(v, ifs);
        if ((int)(unsigned char)v[0] == 2) return v;
    }
    len = rt_strlen(v);
    /* THE POSIX DELIMITER RULE: bash 5.3.15 splits "a:b::c" under IFS=: into
     * FOUR fields. A run of IFS whitespace is one delimiter and is ignored at both ends of
     * the value; any OTHER member of IFS delimits exactly one field by itself,
     * so two in a row leave an empty field between them. One delimiter is
     * therefore "optional IFS whitespace, at most one non-whitespace IFS
     * character, optional IFS whitespace", which is why a trailing ":" adds no
     * field while a leading one adds an empty first field.
     *
     * languages/bash-interpreter.abnf's splitIFS is the same walk.
     *
     * TWO PASSES over one walk (this is the single largest per-word cost in
     * the runtime): pass 0 sizes the buffer, pass 1 fills it, so it stays
     * linear rather than quadratic.
     *
     * The scan below is a fast path ahead of both, kept on purpose because it
     * is measurably faster for the common separator-free value: no IFS
     * character anywhere means one field spanning the whole value, and a plain
     * string IS a one-field list in this protocol. An empty value has NO fields
     * (the "\x02" literal below), and a value already carrying the 0x02 marker
     * must not be returned as itself or it would be re-read as a list. */
    if (len > 0 && (int)(unsigned char)v[0] != 2) {
        while (q < len && rt_haschar(ifs, (int)(unsigned char)v[q]) == 0) { q = q + 1; }
        if (q == len) return v;
    }
    pass = 0;
    while (pass < 2) {
        i = 0;
        nf = 0;
        tot = 0;
        o = 1;
        while (i < len && rt_ifsws(ifs, (int)(unsigned char)v[i]) != 0) { i = i + 1; }
        while (i < len) {
            while (i < len && rt_haschar(ifs, (int)(unsigned char)v[i]) == 0) {
                if (pass != 0) { dst[o] = v[i]; o = o + 1; }
                tot = tot + 1;
                i = i + 1;
            }
            nf = nf + 1;
            if (pass != 0) { dst[o] = 1; o = o + 1; }
            while (i < len && rt_ifsws(ifs, (int)(unsigned char)v[i]) != 0) { i = i + 1; }
            if (i < len && rt_haschar(ifs, (int)(unsigned char)v[i]) != 0) {
                i = i + 1;
                while (i < len && rt_ifsws(ifs, (int)(unsigned char)v[i]) != 0) { i = i + 1; }
            }
        }
        if (pass == 0) {
            /* no fields at all: the empty list is a module literal. */
            if (nf == 0) return "\x02";
            /* EXACTLY ONE FIELD AND NO SEPARATOR BYTE AT ALL - by far the
             * commonest case, and it needs no allocation, because a plain string
             * IS a one-field list in this protocol: rt_nfields answers 1 for it,
             * rt_getfield(v, 0) answers v, rt_wordjoin returns it unchanged, and
             * rt_bnd_acc/rt_bnd_open both go through rt_nfields. The only value
             * that must not take this path is one whose first byte is already the
             * 0x02 list marker, which would then be re-read as a list. */
            if (nf == 1 && tot == len && (int)(unsigned char)v[0] != 2) return v;
            dst = rt_bump(1 + tot + nf + 1);
            dst[0] = 2;
        } else {
            dst[o] = 0;
        }
        pass = pass + 1;
    }
    return dst;
}

/* Appending a field list to a word under construction:
 *   n == 0   nothing changes
 *   n == 1   open = open + f[0]
 *   n >= 2   acc = acc + (open + f[0]) + f[1] .. f[n-2], each closed; open = f[n-1]
 */
char *rt_bnd_acc(char *acc, char *open, char *f) {
    int n = rt_nfields(f);
    char *op = acc;
    int i = 1;
    if (n < 2) return op;
    op = rt_strcat(rt_strcat(rt_strcat(acc, open), rt_getfield(f, 0)), "\x01");
    while (i < n - 1) {
        op = rt_strcat(rt_strcat(op, rt_getfield(f, i)), "\x01");
        i = i + 1;
    }
    return op;
}

char *rt_bnd_open(char *open, char *f) {
    int n = rt_nfields(f);
    if (n == 0) return open;
    if (n == 1) return rt_strcat(open, rt_getfield(f, 0));
    return rt_getfield(f, n - 1);
}

/* THE SAME APPEND, WITH THE TWO BOUNDARY DELIMITERS OF AN UNQUOTED EXPANSION.
 *
 * `x="  a   b  "; echo w:[$x]` is four words in bash 5.3.15 - `w:[`, `a`, `b`,
 * `]`. rt_splitifs throws the leading and trailing IFS whitespace away, which
 * is right for the fields but loses the boundary against the literal text.
 *
 * `opened` is that state: whether anything has gone into `open` since
 * the last field was closed. A boundary delimiter closes `open` IF one is open
 * and does nothing otherwise, which is what keeps `echo $x` two words and
 * `$x$y` (x="a ", y=" b") two rather than three. rt_ifslead / rt_ifstrail
 * answer the two flags from the RAW value, and only IFS WHITESPACE leads: a
 * leading non-whitespace delimiter already produces an empty first field.
 *
 * Three entry points over one walk, because C returns one value and the
 * caller needs three.
 * bash-interpreter.abnf's mkWord is the same state machine. */
int rt_ifslead(char *v, char *ifs) {
    if (v[0] == 0) return 0;
    return rt_ifsws(ifs, (int)(unsigned char)v[0]);
}

int rt_ifstrail(char *v, char *ifs) {
    int len = rt_strlen(v);
    if (len == 0) return 0;
    return rt_haschar(ifs, (int)(unsigned char)v[len - 1]) != 0 ? 1 : 0;
}

char *rt_bnd_acc2(char *acc, char *open, int opened, char *f, int lead, int trail) {
    int n = rt_nfields(f);
    char *op = acc;
    char *cur = open;
    int isopen = opened;
    int i = 1;
    if (lead != 0 && isopen != 0) {
        op = rt_strcat(rt_strcat(op, cur), "\001");
        cur = &empty[0];
        isopen = 0;
    }
    if (n >= 2) {
        op = rt_strcat(rt_strcat(rt_strcat(op, cur), rt_getfield(f, 0)), "\001");
        while (i < n - 1) {
            op = rt_strcat(rt_strcat(op, rt_getfield(f, i)), "\001");
            i = i + 1;
        }
        cur = rt_getfield(f, n - 1);
        isopen = 1;
    } else if (n == 1) {
        cur = rt_strcat(cur, rt_getfield(f, 0));
        isopen = 1;
    }
    if (trail != 0 && isopen != 0) op = rt_strcat(rt_strcat(op, cur), "\001");
    return op;
}

char *rt_bnd_open2(char *open, int opened, char *f, int lead, int trail) {
    int n = rt_nfields(f);
    char *cur = open;
    int isopen = opened;
    if (lead != 0 && isopen != 0) { cur = &empty[0]; isopen = 0; }
    if (n >= 2) { cur = rt_getfield(f, n - 1); isopen = 1; }
    else if (n == 1) { cur = rt_strcat(cur, rt_getfield(f, 0)); isopen = 1; }
    if (trail != 0 && isopen != 0) cur = &empty[0];
    return cur;
}

int rt_bnd_opened2(int opened, char *f, int lead, int trail) {
    int n = rt_nfields(f);
    int isopen = opened;
    if (lead != 0) isopen = 0;
    if (n >= 1) isopen = 1;
    if (trail != 0) isopen = 0;
    return isopen;
}

/* ---- arrays ---------------------------------------------------------------
 * One flat store of (arrayName, key, value) triples with a linear scan. Keys
 * are strings for both kinds of array; entries stay in INSERTION order, which
 * is also the order ${!a[*]} and ${a[@]} report. NARR = 4096.
 */
char *arr_nm[4096];   /* NEW GLOBAL */
char *arr_k[4096];    /* NEW GLOBAL */
char *arr_v[4096];    /* NEW GLOBAL */
int arr_n;            /* NEW GLOBAL */

/* i32 rt_arr_find(i8* nm, i8* k): slot index, or -1. */
int rt_arr_find(char *nm, char *k) {
    int n = arr_n;
    int i = 0;
    while (i < n) {
        int sameNm = rt_streq(nm, arr_nm[i]) != 0;
        int sameK = rt_streq(k, arr_k[i]) != 0;
        if (sameNm && sameK) return i;
        i = i + 1;
    }
    return -1;
}

/* i32 rt_arr_set(i8* nm, i8* k, i8* v) */
int rt_arr_set(char *nm, char *k, char *v) {
    int at = rt_arr_find(nm, k);
    if (at >= 0) {
        arr_v[at] = v;
        return 0;
    }
    int n = arr_n;
    /* ARRMAX = 4096 distinct (array, key) pairs. */
    if (n >= 4096) {
        rt_die("array store", 1, n, 4096);
    }
    arr_nm[n] = nm;
    arr_k[n] = k;
    arr_v[n] = v;
    arr_n = n + 1;
    return 0;
}

/* i8* rt_arr_get(i8* nm, i8* k): the value, or "" when absent. */
char *rt_arr_get(char *nm, char *k) {
    int at = rt_arr_find(nm, k);
    if (at >= 0) return arr_v[at];
    return empty;
}

/* i32 rt_arr_has(i8* nm, i8* k) */
int rt_arr_has(char *nm, char *k) {
    int at = rt_arr_find(nm, k);
    return at >= 0;
}

/* i32 rt_arr_del(i8* nm, i8* k): remove one entry, shifting the tail of the
 * store down over the hole; the array stays set (rt_arr_mark). */
int rt_arr_del(char *nm, char *k) {
    int at = rt_arr_find(nm, k);
    int i = at;
    if (at >= 0) {
        while (1) {
            int n = arr_n;
            if (i < n - 1) {
                int j = i + 1;
                arr_nm[i] = arr_nm[j];
                arr_k[i] = arr_k[j];
                arr_v[i] = arr_v[j];
                i = j;
            } else {
                arr_n = arr_n - 1;
                break;
            }
        }
        rt_arr_mark(nm);
    }
    return 0;
}

/* i32 rt_arr_count(i8* nm) */
int rt_arr_count(char *nm) {
    int n = arr_n;
    int i = 0, c = 0;
    while (i < n) {
        int same = rt_streq(nm, arr_nm[i]) != 0;
        c = same ? c + 1 : c;
        i = i + 1;
    }
    return c;
}

/* i8* rt_arr_list(i8* nm, i8* sep, i32 star, i32 keys): the keys or the values
 * of an array, as a field list (star == 0) or joined with sep (star != 0). */
char *rt_arr_list(char *nm, char *sep, int star, int keys) {
    int n = arr_n;
    int isStar = star != 0;
    char *out = isStar ? empty : "\002";
    int i = 0, c = 0;
    while (i < n) {
        int same = rt_streq(nm, arr_nm[i]) != 0;
        if (same) {
            char *kv = keys != 0 ? arr_k[i] : arr_v[i];
            char *cur = out;
            int needSep = isStar && c > 0;
            char *withSep = rt_strcat(cur, sep);
            if (needSep) cur = withSep;
            cur = rt_strcat(cur, kv);
            char *withMark = rt_strcat(cur, "\001");
            if (!isStar) cur = withMark;
            out = cur;
            c = c + 1;
        }
        i = i + 1;
    }
    return out;
}

/* i32 rt_arr_nextidx(i8* nm): one past the largest numeric key. */
int rt_arr_nextidx(char *nm) {
    int n = arr_n;
    int i = 0, mx = 0;
    while (i < n) {
        int same = rt_streq(nm, arr_nm[i]) != 0;
        int kn = rt_str2int(arr_k[i]);
        int bigger = same && kn >= mx;
        mx = bigger ? kn + 1 : mx;
        i = i + 1;
    }
    return mx;
}

/* The FINAL fields of an unquoted ${v-w} / ${v+w}: the word was split by its own
 * quoting, so the word around the ${...} takes the list as it is instead of
 * splitting it again (bash 5.3: `set -- "a b" ""; printf '<%s>' ${x:-"$@"}` is
 * <a b><>). The list is a fresh copy and is known by its ADDRESS, the last one
 * made, which nothing else can hand on; rt_uqlist returns it untouched. Twin of
 * dfltWord's {fin} in bash-interpreter.abnf. */
char *fin_last;
char *rt_finlist(char *v) {
    if ((int)(unsigned char)v[0] == 2) fin_last = rt_strcat(v, empty);
    else fin_last = rt_strcat(rt_strcat("\002", v), "\001");
    return fin_last;
}

int rt_isfin(char *v) {
    return v == fin_last ? 1 : 0;
}

/* AN ARRAY ASSIGNED A LITERAL IS SET EVEN WITH NO ELEMENT: bash 5.3 lists `e=()`
 * in ${!e*} and not `declare -a e`. A row store keeps nothing for an empty array,
 * so the literal leaves a mark - the row ("\003NAME", "") - which rt_arr_clear
 * drops with the rows and the subshell snapshot copies with them. Unsetting an
 * element leaves the same mark: `b[0]=1; unset 'b[0]'` is the set, empty
 * `declare -a b=()`. Twin of the `lit` flag of an array in bash-interpreter.abnf. */
int rt_arr_mark(char *nm) {
    return rt_arr_set(rt_strcat("\003", nm), empty, empty);
}

/* A DECLARED-ONLY ARRAY (`declare -a e` with no value) is not set, but
 * `declare -p e` prints `declare -a e`: the row ("\003NAME", "d") records the
 * declaration. Twin of the `decl` flag of an array in bash-interpreter.abnf. */
int rt_arr_dmark(char *nm) {
    return rt_arr_set(rt_strcat("\003", nm), "d", empty);
}

/* A SCALAR DECLARED WITHOUT A VALUE (`declare x`, or a local unset in the
 * function that declared it) is not set, but `declare -p x` prints
 * `declare -- x`: the row ("\003NAME", "s"). Twin of dclVars in
 * bash-interpreter.abnf. */
int rt_arr_smark(char *nm) {
    return rt_arr_set(rt_strcat("\003", nm), "s", empty);
}

/* Is the row name rn NAME's mark? */
int rt_ismark(char *rn, char *nm) {
    int i = 0;
    if ((int)(unsigned char)rn[0] != 3) return 0;
    while (nm[i] != 0 && rn[i + 1] == nm[i]) i = i + 1;
    return nm[i] == 0 && rn[i + 1] == 0;
}

/* Does NAME carry the mark with key k ("" set, "d" declared)? */
int rt_arr_haskmark(char *nm, char *k) {
    int i = 0;
    while (i < arr_n) {
        if (rt_ismark(arr_nm[i], nm) != 0 && rt_streq(arr_k[i], k) != 0) return 1;
        i = i + 1;
    }
    return 0;
}

int rt_arr_marked(char *nm) {
    return rt_arr_haskmark(nm, empty);
}

int rt_arr_dmarked(char *nm) {
    return rt_arr_haskmark(nm, "d");
}

int rt_arr_smarked(char *nm) {
    return rt_arr_haskmark(nm, "s");
}

/* i32 rt_arr_clear(i8* nm): drop every entry of one array, and its mark. */
int rt_arr_clear(char *nm) {
    int i = 0, o = 0;
    while (i < arr_n) {
        int same = rt_streq(nm, arr_nm[i]) != 0 || rt_ismark(arr_nm[i], nm) != 0;
        if (!same) {
            arr_nm[o] = arr_nm[i];
            arr_k[o] = arr_k[i];
            arr_v[o] = arr_v[i];
            o = o + 1;
        }
        i = i + 1;
    }
    arr_n = o;
    return 0;
}

/* "@" and "*" name the POSITIONAL PARAMETERS at the four rt_sc_ sites below,
 * keyed from 1: the emitter routes the operator forms of $@ and $* (${@:2},
 * ${*@Q}, a pattern replacement) through the subscripted-array path. Twin of
 * isPosName in languages/bash-interpreter.abnf. */
int rt_isposnm(char *nm) {
    if (rt_streq(nm, "@") != 0) return 1;
    if (rt_streq(nm, "*") != 0) return 1;
    return 0;
}

/* ---- a SET SCALAR IS A ONE-ELEMENT ARRAY ----------------------------------
 * bash 5.3 answers Q for "${x[@]}" with x=Q, Q for "${x[0]}", 1 for ${#x[@]}
 * and 0 for ${!x[@]}: a plain scalar occupies index 0 of an array of one.
 *
 * WHY FOUR SEPARATE NAMES AND NOT A CHANGE TO rt_arr_count. rt_arr_count IS the test
 * `is this name an array` everywhere else in the emitted program - varString
 * picks between rt_arr_get(name, "0") and the scalar slot with it, `[[ -v x ]]`
 * asks it, and every assignment path asks it. Making it answer 1 for a scalar
 * would make all of those read a plain variable as an array, which is a much
 * larger claim than the one bash makes. So the fallback lives in four functions
 * that ONLY the ${name[...]} expansion sites call, exactly as
 * languages/bash-interpreter.abnf keeps scKeyList/scValList/scGet/scHas beside
 * arrKeyList/arrValList/arrGet/arrHas rather than inside them.
 *
 * The scalar is found BY NAME with rt_getvar_byname, which is the same lookup
 * the emitted `${!ref}` uses, so no new plumbing crosses the boundary: the
 * emitter passes the name it already passes to rt_arr_get. */
int rt_sc_count(char *nm) {
    char *v;
    int c;
    if (rt_isposnm(nm) != 0) return frame_n;
    c = rt_arr_count(nm);
    if (c > 0) return c;
    v = rt_getvar_byname(nm);
    return v != &unset_marker[0] ? 1 : 0;
}

char *rt_sc_get(char *nm, char *k) {
    char *v;
    if (rt_isposnm(nm) != 0) return rt_param(rt_str2int(k));
    if (rt_arr_count(nm) > 0) return rt_arr_get(nm, k);
    v = rt_getvar_byname(nm);
    if (v != &unset_marker[0] && rt_streq(k, "0") != 0) return v;
    return &empty[0];
}

int rt_sc_has(char *nm, char *k) {
    char *v;
    int pk;
    if (rt_isposnm(nm) != 0) {
        pk = rt_str2int(k);
        if (pk >= 1 && pk <= frame_n) return 1;
        return 0;
    }
    if (rt_arr_count(nm) > 0) return rt_arr_has(nm, k);
    v = rt_getvar_byname(nm);
    if (v != &unset_marker[0] && rt_streq(k, "0") != 0) return 1;
    return 0;
}

/* The one-element form of rt_arr_list: "\x02" v "\x01" as a field list, or the
 * bare value when `star` asked for the joined form. `keys` makes the single
 * key "0" the payload, which is what ${!x[@]} reports. */
char *rt_sc_list(char *nm, char *sep, int star, int keys) {
    char *v;
    if (rt_isposnm(nm) != 0) return rt_params(sep, star);
    if (rt_arr_count(nm) > 0) return rt_arr_list(nm, sep, star, keys);
    v = rt_getvar_byname(nm);
    if (v == &unset_marker[0]) return star != 0 ? &empty[0] : "\002";
    if (keys != 0) v = "0";
    if (star != 0) return v;
    return rt_strcat(rt_strcat("\002", v), "\001");
}

/* The positional parameters as a field list that starts at $0, which is what
 * ${@:off:len} counts from ("${@:1}" is $1 on). */
char *rt_params0(void) {
    return rt_catfields(rt_strcat(rt_strcat("\002", script_name), "\001"), rt_params(empty, 0));
}

/* ${!p*} / ${!p@}: the names of the SET variables starting with p, sorted - a
 * scalar that is not the unset marker, or an array with an element or a literal
 * (rt_arr_mark; arrays live in arr_nm, not among the scalar slots). Joined with sep when star is set, a
 * field list otherwise. Twin of varNamesWithPrefix in
 * languages/bash-interpreter.abnf. */
char *rt_pfxnames(char *p, char *sep, int star) {
    char *nm[1024];
    int n = 0;
    int pl = rt_strlen(p);
    int total = nvars + arr_n;
    int i = 0;
    int j;
    int dup;
    char *v;
    char *out;
    char *t;
    while (i < total) {
        v = i < nvars ? varnames[i] : arr_nm[i - nvars];
        if ((int)(unsigned char)v[0] == 3) v = rt_substr(v, 1, rt_strlen(v) - 1);
        int ok = rt_strlen(v) >= pl && rt_streq(rt_substr(v, 0, pl), p) != 0;
        if (ok && rt_getvar_byname(v) == &unset_marker[0] && rt_arr_count(v) == 0
            && rt_arr_marked(v) == 0) ok = 0;
        dup = 0;
        j = 0;
        while (ok && j < n) {
            if (rt_streq(nm[j], v) != 0) dup = 1;
            j = j + 1;
        }
        if (ok && dup == 0 && n < 1024) {
            j = n - 1;
            while (j >= 0 && rt_strcmp(nm[j], v) > 0) {
                nm[j + 1] = nm[j];
                j = j - 1;
            }
            nm[j + 1] = v;
            n = n + 1;
        }
        i = i + 1;
    }
    out = star != 0 ? empty : "\002";
    i = 0;
    while (i < n) {
        t = out;
        if (star != 0 && i > 0) t = rt_strcat(t, sep);
        t = rt_strcat(t, nm[i]);
        if (star == 0) t = rt_strcat(t, "\001");
        out = t;
        i = i + 1;
    }
    return out;
}

/* ---- declare -p ------------------------------------------------------------
 * `declare -p NAME` reproduces the declaration bash would need to recreate the
 * variable. Per bash 5.3:
 *
 *     declare -- x="hello"    declare -i n="7"    declare -a a=([0]="p" [1]="q")
 *     declare -A m=([k]="v" )      <- the trailing space is bash's, indexed has none
 *     bash: declare: nosuchvar: not found          (fd 2, status 1)
 *
 * The ATTRIBUTE LETTER comes from the emitter, because -i/-u/-l live in
 * compile-time maps in languages/bash-to-llvm-ir.abnf; array-ness and the value
 * are asked here. languages/bash-interpreter.abnf's declLine is the same text,
 * character for character, from its own tables.
 *
 * IS IT AN ARRAY: at least one element, or set empty (rt_arr_mark) - then
 * `declare -a a=()`. One only declared (rt_arr_dmark) is `declare -a a`. The
 * interpreter's declIsArr / declOnly ask the same two questions. */
/* The $'...' form, with bash's escape table - the same one shellQuote uses in
 * languages/bash-interpreter.abnf.
 *
 * rt_ansiqs and rt_dqs build a string rather than print, so `declare -p` and
 * ${a[@]@A} share one renderer. */
char *rt_ansiqs(char *v) {
    int n = rt_strlen(v);
    char *d = rt_bump(n * 4 + 8);
    int o = 0;
    int i = 0;
    d[o] = (char)36; o = o + 1;
    d[o] = (char)39; o = o + 1;
    while (i < n) {
        int c = (int)(unsigned char)v[i];
        int e = 0;
        if (c == 39) e = 39;
        else if (c == 92) e = 92;
        else if (c == 7) e = 97;
        else if (c == 8) e = 98;
        else if (c == 9) e = 116;
        else if (c == 10) e = 110;
        else if (c == 11) e = 118;
        else if (c == 12) e = 102;
        else if (c == 13) e = 114;
        else if (c == 27) e = 69;
        if (e != 0) {
            d[o] = (char)92; o = o + 1;
            d[o] = (char)e; o = o + 1;
        } else if (c < 32 || c == 127) {
            d[o] = (char)92; o = o + 1;
            d[o] = (char)(48 + (c / 64)); o = o + 1;
            d[o] = (char)(48 + ((c / 8) % 8)); o = o + 1;
            d[o] = (char)(48 + (c % 8)); o = o + 1;
        } else {
            d[o] = (char)c; o = o + 1;
        }
        i = i + 1;
    }
    d[o] = (char)39; o = o + 1;
    d[o] = 0;
    return d;
}

/* "..." unless the value holds a control character. A byte >= 128 is not one:
 * in a UTF-8 locale bash prints the accented byte intact. */
char *rt_dqs(char *v) {
    int n = rt_strlen(v);
    int ctl = 0;
    int i = 0;
    char *d;
    int o = 0;
    while (i < n) {
        int c = (int)(unsigned char)v[i];
        if (c < 32 || c == 127) { ctl = 1; }
        i = i + 1;
    }
    if (ctl != 0) return rt_ansiqs(v);
    d = rt_bump(n * 2 + 4);
    d[o] = (char)34; o = o + 1;
    i = 0;
    while (i < n) {
        int c = (int)(unsigned char)v[i];
        if (c == 34 || c == 92 || c == 36 || c == 96) { d[o] = (char)92; o = o + 1; }
        d[o] = (char)c; o = o + 1;
        i = i + 1;
    }
    d[o] = (char)34; o = o + 1;
    d[o] = 0;
    return d;
}

/* Does this subscript need quoting inside `[...]`? bash quotes an associative
 * key only when the shell would need it - `[ab_c]`, `[k]` and `[5x]` stay bare,
 * `["k one"]` and `["q\$x"]` do not - and the test is sh_contains_shell_metas
 * from bash's own lib/sh. */
int rt_keymeta(char *k) {
    int i = 0;
    while (k[i] != 0) {
        int c = (int)(unsigned char)k[i];
        if (c == 32 || c == 9 || c == 10) return 1;
        if (c == 39 || c == 34 || c == 92) return 1;
        if (c == 124 || c == 38 || c == 59) return 1;
        if (c == 40 || c == 41 || c == 60 || c == 62) return 1;
        if (c == 33 || c == 123 || c == 125) return 1;
        if (c == 42 || c == 91 || c == 63 || c == 93) return 1;
        if (c == 94 || c == 36 || c == 96 || c == 44) return 1;
        if (c == 126 && i == 0) return 1;
        if (c == 35 && i == 0) return 1;
        i = i + 1;
    }
    return 0;
}

/* `NAME=([k]="v" ...)`: the third FIELD of `declare -p` and of ${a[@]@A}. */
char *rt_arrassign(char *nm, int assoc) {
    char *out = rt_strcat(nm, "=(");
    int c = 0;
    int i = 0;
    while (i < arr_n) {
        if (rt_streq(nm, arr_nm[i]) != 0) {
            if (c > 0) out = rt_strcat(out, " ");
            out = rt_strcat(out, "[");
            out = rt_strcat(out, rt_keymeta(arr_k[i]) != 0 ? rt_dqs(arr_k[i]) : arr_k[i]);
            out = rt_strcat(out, "]=");
            out = rt_strcat(out, rt_dqs(arr_v[i]));
            c = c + 1;
        }
        i = i + 1;
    }
    if (assoc != 0 && c > 0) out = rt_strcat(out, " ");
    return rt_strcat(out, ")");
}

/* ---- the ${v@A} / ${v@K} / ${v@k} / ${v@a} family ------------------------ */
/* Every rule below was measured against GNU bash 5.3.15 and is written into
 * languages/bash-interpreter.abnf character for character; see xformOne there. */

/* Is this the unset marker? ${nosuch@Q} is EMPTY in bash while ${empty@Q} is
 * ''. */
int rt_isset(char *v) { return (v == &unset_marker[0]) ? 0 : 1; }

char *rt_shq(char *v, int isset) {
    if (isset == 0) return empty;
    return rt_shquote(v);
}

/* ${v@A} of a scalar or of ONE element: the assignment that would recreate it,
 * with the `declare -FLAGS ` prefix bash writes when the name has an attribute.
 * An element that is not set on a name that IS prints the declaration alone -
 * `a=(pa pb); echo "${a[9]@A}"` is `declare -a a`. */
char *rt_declasg(char *nm, char *at, char *v, int isset) {
    char *base;
    if (at[0] == 0) {
        if (isset == 0) return empty;
        return rt_strcat(rt_strcat(nm, "="), rt_shquote(v));
    }
    base = rt_strcat(rt_strcat("declare -", at), " ");
    base = rt_strcat(base, nm);
    if (isset == 0) return base;
    return rt_strcat(rt_strcat(base, "="), rt_shquote(v));
}

/* ${a[@]@A}: the whole `declare -p` rendering as a FIELD LIST. bash splits the
 * text `declare -a a=` on IFS (a null IFS counting as the default) and keeps the
 * parenthesised value atomic, glued to the last field unless that text ends in a
 * delimiter: `printf '<%s>' "${a[@]@A}"` prints three items under the default
 * IFS, one under IFS=: and two under IFS==. The [*] form is the one string
 * `declare -a a=(...)` whatever IFS holds. A name that is not an array falls back
 * to the scalar form, which is what bash does too. Twin of the @A arm in
 * bash-interpreter.abnf. */
char *rt_declxa(char *nm, char *at, char *v, int isset, int assoc, char *ifs, int star) {
    int isarr = (rt_arr_count(nm) > 0 || assoc != 0) ? 1 : 0;
    char *flag;
    char *asg;
    char *val;
    char *txt;
    char *fs;
    char *out;
    int n;
    if (isarr == 0) return rt_declasg(nm, at, v, isset);
    /* `at` already carries the array letter in front of the scalar ones, so
     * `declare -ai x` answers `-ai`; the fallback is for the case where the
     * emitter could not decide array-ness at all. */
    flag = at[0] == 0 ? (assoc != 0 ? "-A" : "-a") : rt_strcat("-", at);
    asg = rt_arrassign(nm, assoc);
    if (star != 0) {
        out = rt_strcat("declare ", flag);
        out = rt_strcat(out, " ");
        return rt_strcat(out, asg);
    }
    if (ifs[0] == 0) {
        out = rt_strcat("\002declare\001", flag);
        out = rt_strcat(out, "\001");
        out = rt_strcat(out, asg);
        return rt_strcat(out, "\001");
    }
    val = rt_substr(asg, rt_strlen(nm) + 1, rt_strlen(asg) - rt_strlen(nm) - 1);
    txt = rt_strcat(rt_strcat(rt_strcat("declare ", flag), " "), rt_strcat(nm, "="));
    fs = rt_splitifs(txt, ifs);
    if ((int)(unsigned char)fs[0] != 2) return rt_strcat(fs, val);
    n = rt_strlen(fs);
    if (n < 2 || rt_ifstrail(txt, ifs) != 0) return rt_strcat(rt_strcat(fs, val), "\001");
    return rt_strcat(rt_strcat(rt_substr(fs, 0, n - 1), val), "\001");
}

/* ${@@A} / ${*@A}: the command that would recreate the positional parameters,
 * `set -- 'a' 'b c'` (bash 5.3), in the shape rt_declxa gives an array: the [*]
 * form is the one string, the values joined by sep (IFS's first character); the
 * [@] form is a FIELD LIST, the text `set -- ` split on IFS (a null IFS counting
 * as the default) and glued to the first value unless it ends in a delimiter,
 * then one field per further value. With no parameters [@] is no field and [*]
 * the empty string. Twin of posDeclA in bash-interpreter.abnf. */
char *rt_declpos(char *ifs, char *sep, int star) {
    int cnt = frame_n;
    int i = 0;
    char *out;
    char *fs;
    int n;
    if (star != 0) {
        if (cnt == 0) return empty;
        out = "set -- ";
        while (i < cnt) {
            if (i > 0) out = rt_strcat(out, sep);
            out = rt_strcat(out, rt_shquote(argv[frame_base + i]));
            i = i + 1;
        }
        return out;
    }
    if (cnt == 0) return "\002";
    if (ifs[0] == 0) {
        out = "\002set\001--\001";
    } else {
        fs = rt_splitifs("set -- ", ifs);
        n = rt_strlen(fs);
        if ((int)(unsigned char)fs[0] != 2) {
            out = rt_strcat(rt_strcat("\002", fs), rt_shquote(argv[frame_base]));
            out = rt_strcat(out, "\001");
            i = 1;
        } else if (n < 2 || rt_ifstrail("set -- ", ifs) != 0) {
            out = fs;
        } else {
            out = rt_strcat(rt_substr(fs, 0, n - 1), rt_shquote(argv[frame_base]));
            out = rt_strcat(out, "\001");
            i = 1;
        }
    }
    while (i < cnt) {
        out = rt_strcat(rt_strcat(out, rt_shquote(argv[frame_base + i])), "\001");
        i = i + 1;
    }
    return out;
}

/* ${a[@]@K}: `key "value"` pairs as ONE field, with the trailing space bash
 * writes for the associative form only. */
char *rt_kvq(char *nm, int assoc, char *v, int isset) {
    char *out = empty;
    int c = 0;
    int i = 0;
    if (rt_arr_count(nm) == 0 && assoc == 0) return rt_shq(v, isset);
    while (i < arr_n) {
        if (rt_streq(nm, arr_nm[i]) != 0) {
            if (c > 0) out = rt_strcat(out, " ");
            out = rt_strcat(out, rt_keymeta(arr_k[i]) != 0 ? rt_dqs(arr_k[i]) : arr_k[i]);
            out = rt_strcat(out, " ");
            out = rt_strcat(out, rt_dqs(arr_v[i]));
            c = c + 1;
        }
        i = i + 1;
    }
    if (assoc != 0 && c > 0) out = rt_strcat(out, " ");
    return out;
}

/* ${a[@]@k}: the same pairs, unquoted, as separate FIELDS. */
char *rt_kvp(char *nm, char *sep, int star, char *v, int isset) {
    char *out = star != 0 ? empty : "\002";
    int c = 0;
    int i = 0;
    if (rt_arr_count(nm) == 0) return rt_shq(v, isset);
    while (i < arr_n) {
        if (rt_streq(nm, arr_nm[i]) != 0) {
            if (star != 0) {
                if (c > 0) out = rt_strcat(out, sep);
                out = rt_strcat(out, arr_k[i]);
                out = rt_strcat(out, sep);
                out = rt_strcat(out, arr_v[i]);
            } else {
                out = rt_strcat(out, arr_k[i]);
                out = rt_strcat(out, "\001");
                out = rt_strcat(out, arr_v[i]);
                out = rt_strcat(out, "\001");
            }
            c = c + 1;
        }
        i = i + 1;
    }
    return out;
}

/* ${a[@]@a}: the attribute letters, ONCE PER ELEMENT - `a=(pa pb)` answers
 * `a a`, and ${@@a} once per positional parameter (none for none). */
char *rt_attrf(char *nm, char *at, char *sep, int star) {
    int n = rt_arr_count(nm);
    char *out;
    int i = 0;
    if (rt_isposnm(nm) != 0) {
        n = frame_n;
    } else if (n == 0) {
        n = 1;
    }
    out = star != 0 ? empty : "\002";
    while (i < n) {
        if (star != 0) {
            if (i > 0) out = rt_strcat(out, sep);
            out = rt_strcat(out, at);
        } else {
            out = rt_strcat(out, at);
            out = rt_strcat(out, "\001");
        }
        i = i + 1;
    }
    return out;
}

/* ${v@P}: PS1-style prompt expansion. This is promptExpand from
 * languages/bash-interpreter.abnf, escape for escape: the ones that need a
 * process behind them (\u \h \s \j \! \#) answer the same documented fixed
 * values, and the second pass is the double-quoted backslash removal bash runs
 * over the decoded prompt (`promptvars`), which is what turns \\$ into $. */
char *rt_prompt(char *s) {
    int n = rt_strlen(s);
    char *u0 = rt_getvar_byname("USER");
    char *p0 = rt_getvar_byname("PWD");
    int ex;
    char *d;
    int o = 0;
    int i = 0;
    char *dq;
    int m;
    int j;
    if (u0 == &unset_marker[0]) u0 = empty;
    if (p0 == &unset_marker[0]) p0 = empty;
    /* Every escape expands to at most one of \u, \w or "localhost", so this
     * bound holds however the escapes are arranged. */
    ex = rt_strlen(u0) + rt_strlen(p0) + 16;
    d = rt_bump(n * (ex + 4) + 64);
    while (i < n) {
        int c = (int)(unsigned char)s[i];
        int e;
        if (c != 92) { d[o] = (char)c; o = o + 1; i = i + 1; continue; }
        i = i + 1;
        if (i >= n) { d[o] = (char)92; o = o + 1; i = i + 1; continue; }
        e = (int)(unsigned char)s[i];
        i = i + 1;
        if (e == 97) { d[o] = (char)7; o = o + 1; }
        else if (e == 101) { d[o] = (char)27; o = o + 1; }
        else if (e == 110) { d[o] = (char)10; o = o + 1; }
        else if (e == 114) { d[o] = (char)13; o = o + 1; }
        else if (e == 92) { d[o] = (char)92; o = o + 1; }
        else if (e == 91 || e == 93) { o = o + 0; }
        else if (e == 36) { d[o] = (char)36; o = o + 1; }
        else if (e == 117) {
            int k = 0;
            while (u0[k] != 0) { d[o] = u0[k]; o = o + 1; k = k + 1; }
        }
        else if (e == 104 || e == 72) {
            char *hs = "localhost";
            int k = 0;
            while (hs[k] != 0) { d[o] = hs[k]; o = o + 1; k = k + 1; }
        }
        else if (e == 119 || e == 87) {
            int k = 0;
            int st = 0;
            if (e == 87) {
                int pl = rt_strlen(p0);
                int last = -1;
                k = 0;
                while (k < pl) { if ((int)(unsigned char)p0[k] == 47) last = k; k = k + 1; }
                if (last >= 0 && last != pl - 1) st = last + 1;
                k = st;
            }
            while (p0[k] != 0) { d[o] = p0[k]; o = o + 1; k = k + 1; }
        }
        else if (e == 115) {
            char *bs = "bash";
            int k = 0;
            while (bs[k] != 0) { d[o] = bs[k]; o = o + 1; k = k + 1; }
        }
        else if (e == 106 || e == 33 || e == 35) { d[o] = (char)48; o = o + 1; }
        else if (e >= 48 && e <= 55) {
            int val = e - 48;
            int cnt = 1;
            while (cnt < 3 && i < n) {
                int dch = (int)(unsigned char)s[i];
                if (dch < 48 || dch > 55) break;
                val = val * 8 + (dch - 48);
                i = i + 1;
                cnt = cnt + 1;
            }
            d[o] = (char)val; o = o + 1;
        }
        else { d[o] = (char)92; o = o + 1; d[o] = (char)e; o = o + 1; }
    }
    d[o] = 0;
    /* pass two: the backslash removal of a double-quoted expansion. */
    m = o;
    dq = rt_bump(m + 2);
    j = 0;
    o = 0;
    while (j < m) {
        int c = (int)(unsigned char)d[j];
        if (c == 92 && j + 1 < m) {
            int nx = (int)(unsigned char)d[j + 1];
            if (nx == 36 || nx == 96 || nx == 34 || nx == 92 || nx == 10) {
                dq[o] = (char)nx; o = o + 1; j = j + 2; continue;
            }
        }
        dq[o] = (char)c; o = o + 1; j = j + 1;
    }
    dq[o] = 0;
    return dq;
}

int rt_declp(char *nm, char *flag, int assoc) {
    int n = rt_arr_count(nm);
    int isarr = (n > 0 || rt_arr_marked(nm) != 0) ? 1 : 0;
    char *v;
    int saved;
    if (isarr == 0) {
        v = rt_getvar_byname(nm);
        /* Declared with -a/-A: with no value `declare -a e`; a plain assignment
         * since then is element 0, which this store keeps in the scalar slot. */
        if (rt_arr_dmarked(nm) != 0) {
            rt_prints("declare -");
            rt_prints(assoc != 0 ? "A" : "a");
            rt_prints(flag);
            rt_prints(" ");
            rt_prints(nm);
            if (v != &unset_marker[0]) {
                rt_prints("=([0]=");
                rt_prints(rt_dqs(v));
                rt_prints(assoc != 0 ? " )" : ")");
            }
            rt_prints("\n");
            return 0;
        }
        if (v == &unset_marker[0] && rt_arr_smarked(nm) != 0) {
            rt_prints("declare -");
            rt_prints(flag[0] == 0 ? "-" : flag);
            rt_prints(" ");
            rt_prints(nm);
            rt_prints("\n");
            return 0;
        }
        if (v == &unset_marker[0]) {
            /* On the standard error CHANNEL, so a 2>/dev/null silences it exactly
             * as the shell does - the same rule rt_nounset follows. */
            saved = sink_out;
            sink_out = sink_err;
            rt_prints("bash: declare: ");
            rt_prints(nm);
            rt_prints(": not found\n");
            sink_out = saved;
            return 1;
        }
        rt_prints("declare -");
        /* `flag` is the SCALAR attribute letters, "" for a name that has none -
         * where bash writes `declare -- x`. */
        rt_prints(flag[0] == 0 ? "-" : flag);
        rt_prints(" ");
        rt_prints(nm);
        rt_prints("=");
        rt_prints(rt_dqs(v));
        rt_prints("\n");
        return 0;
    }
    rt_prints("declare -");
    rt_prints(assoc != 0 ? "A" : "a");
    rt_prints(flag);
    rt_prints(" ");
    rt_prints(rt_arrassign(nm, assoc));
    rt_prints("\n");
    return 0;
}

/* i32 rt_arr_append(i8* nm, i8* fields): append every field with the next
 * numeric index. */
int rt_arr_append(char *nm, char *f) {
    int n = rt_nfields(f);
    int i = 0;
    while (i < n) {
        int idx = rt_arr_nextidx(nm);
        char *ks = rt_int2str(idx);
        char *vs = rt_getfield(f, i);
        rt_arr_set(nm, ks, vs);
        i = i + 1;
    }
    return 0;
}

/* i8* rt_slicefields(i8* f, i32 off, i32 n, i8* sep, i32 star) */
char *rt_slicefields(char *f, int off, int n, char *sep, int star) {
    int cnt = rt_nfields(f);
    int isStar = star != 0;
    char *out = isStar ? empty : "\002";
    int i = off, taken = 0;
    while (i < cnt && taken < n) {
        char *cur = out;
        int needSep = isStar && taken > 0;
        char *withSep = rt_strcat(cur, sep);
        if (needSep) cur = withSep;
        char *fld = rt_getfield(f, i);
        cur = rt_strcat(cur, fld);
        char *withMark = rt_strcat(cur, "\001");
        if (!isStar) cur = withMark;
        out = cur;
        i = i + 1;
        taken = taken + 1;
    }
    return out;
}

/* i8* rt_globescape(i8* s): make every glob metacharacter of s literal, which
 * is what a QUOTED part of a [[ ]] pattern is ( [[ $s == "h*" ]] does not
 * glob ). */
char *rt_globescape(char *s) {
    int len = rt_strlen(s);
    char *dst = rt_bump(len * 2 + 1);
    int i = 0, o = 0;
    while (i < len) {
        int ch = (int)(unsigned char)s[i];
        int meta = ch == 42 || ch == 63 || ch == 91 || ch == 93 || ch == 92;
        if (meta) {
            dst[o] = 92;
            o = o + 1;
        }
        dst[o] = (char)ch;
        o = o + 1;
        i = i + 1;
    }
    dst[o] = 0;
    return dst;
}

/* i8* rt_mapf(i8* v, i32 op, i8* p, i8* r, i32 mode, i8* sep, i32 star): one
 * pattern operator applied to EVERY FIELD of a field list.
 *
 * op 0 = rt_strip, 1 = rt_replace, 2 = rt_case, 3 = rt_shquote, 4 = rt_ansic,
 * and `mode` is that function's own mode. The @-transforms must be
 * element-wise too, or they would see the 0x02 / 0x01 marker bytes as data.
 * With star != 0 the results are joined with `sep` (${a[*]/x/Y});
 * otherwise they come back as a field list (${a[@]/x/Y}).
 *
 * THE MAP HAS TO BE PER ELEMENT, not on the joined string: bash 5.3.15 answers
 * "Qa bQ" for `b=(aa ba); echo "${b[*]/a/Q}"` where replacing once in "aa ba"
 * would answer "Qa ba". languages/bash-interpreter.abnf's applyPatOp is the
 * same three arms, driven by the same walk. */
char *rt_mapf(char *v, int op, char *p, char *r, int mode, char *sep, int star) {
    int n = rt_nfields(v);
    int i = 0;
    char *out;
    if (star != 0) out = &empty[0]; else out = "\002";
    while (i < n) {
        char *f = rt_getfield(v, i);
        char *g;
        if (op == 0) g = rt_strip(f, p, mode);
        else if (op == 1) g = rt_replace(f, p, r, mode);
        else if (op == 3) g = rt_shquote(f);
        else if (op == 4) g = rt_ansic(f);
        else if (op == 5) g = rt_prompt(f);
        else g = rt_case(f, mode, p);
        if (star != 0) {
            if (i > 0) out = rt_strcat(out, sep);
            out = rt_strcat(out, g);
        } else {
            out = rt_strcat(rt_strcat(out, g), "\001");
        }
        i = i + 1;
    }
    return out;
}

/* i8* rt_catfields(i8* a, i8* b): concatenate two field lists. */
char *rt_catfields(char *a, char *b) {
    int n = rt_nfields(b);
    char *out = a;
    int i = 0;
    while (i < n) {
        char *fld = rt_getfield(b, i);
        char *t = rt_strcat(out, fld);
        out = rt_strcat(t, "\001");
        i = i + 1;
    }
    return out;
}

/* `echo` with the FLAG SCAN AT RUN TIME, so `opts="-n"; echo $opts x` treats
 * the expanded -n as a flag, as bash does. `newline` and
 * `escapes` carry in whatever the constant prefix already settled, and the scan
 * continues over the fields the expansions produced - which is the same order
 * languages/bash-interpreter.abnf's echoCmd walks. */
int rt_echo(char *fields, int newline, int escapes) {
    int n = rt_nfields(fields);
    int i = 0;
    int flagging = 1;
    int first = 1;
    while (i < n) {
        char *f = rt_getfield(fields, i);
        if (flagging != 0) {
            if (rt_streq(f, "-n") != 0) { newline = 0; i = i + 1; continue; }
            if (rt_streq(f, "-e") != 0) { escapes = 1; i = i + 1; continue; }
            if (rt_streq(f, "-E") != 0) { escapes = 0; i = i + 1; continue; }
            flagging = 0;
        }
        if (first == 0) rt_prints(" ");
        rt_prints(escapes != 0 ? rt_unescape(f) : f);
        first = 0;
        i = i + 1;
    }
    if (newline != 0) rt_putc(10);
    return 0;
}

/* i32 rt_argpush(i8* v): append every FIELD of v to the argv stack. */
int rt_argpush(char *v) {
    int n = rt_nfields(v);
    int i = 0;
    while (i < n) {
        int t = argv_top;
        if (t >= 4096) {           /* ARGVMAX; there was no test here either */
            rt_die("positional-parameter stack", 1, t, 4096);
        }
        argv[t] = rt_getfield(v, i);
        argv_top = t + 1;
        i = i + 1;
    }
    return 0;
}

/* i8* rt_param(i32 n): the n-th positional parameter ("" past the end). */
char *rt_param(int n) {
    int cnt = frame_n;
    if (n >= 1 && n <= cnt) {
        int idx = frame_base + (n - 1);
        return argv[idx];
    }
    return empty;
}

/* i8* rt_params(i8* sep, i32 star): "$@" when sep is the field marker, "$*"
 * when it is a separator. */
char *rt_params(char *sep, int star) {
    int cnt = frame_n;
    int base = frame_base;
    int isStar = star != 0;
    char *out = isStar ? empty : "\002";
    int i = 0;
    while (i < cnt) {
        char *v = argv[base + i];
        char *cur = out;
        int needSep = isStar && i > 0;
        char *withSep = rt_strcat(cur, sep);
        if (needSep) cur = withSep;
        cur = rt_strcat(cur, v);
        char *withMark = rt_strcat(cur, "\001");
        if (!isStar) cur = withMark;
        out = cur;
        i = i + 1;
    }
    return out;
}

/* ---- output, capture stack, subshell snapshots, local save stack --------- */

/* Under set -u, reading an unset variable raises the abort flag. */
/* `name` is the source name of the variable being read, for the diagnostic:
 * bash 5.3 prints "bash: line N: nope: unbound variable" on fd 2. The line
 * number is not recoverable here - the emitted program keeps no line table - so the text
 * is the rest of bash's message, and languages/bash-interpreter.abnf writes the
 * same string, character for character, from getVarVal.
 *
 * `isarr` is 1 when the caller's OTHER select arm is the live one (an array name
 * expands to element 0). The emitter cannot branch there - varString has to hand
 * its value back in the block it was given - and an LLVM select evaluates both
 * operands, so without this flag the dead arm raised the abort. */
char *rt_nounset(char *v, char *name, int isarr)
{
    char *um;
    int saved;
    um = &unset_marker[0];
    if (isarr == 0 && opt_nounset != 0 && v == um) {
        abort_flag = 1;
        /* On the standard error CHANNEL, not straight to fd 2: a `2>/dev/null`
         * around the failing command must silence it exactly as it does in the
         * shell, and rt_putc is what honours that. */
        saved = sink_out;
        sink_out = sink_err;
        rt_prints("bash: ");
        rt_prints(name);
        rt_prints(": unbound variable\n");
        sink_out = saved;
    }
    return v;
}

/* One output byte - into the innermost capture buffer if a command substitution
 * is running, otherwise to stdout. Channel 1 is the standard error channel and
 * channel 2 is the bit bucket.
 *
 * Channel 1 goes to fd 2 one byte at a time through rt_ech, so a redirection
 * that reroutes the channel still decides where the text goes; $( ) capture is
 * channel 0 only, exactly like the shell.
 *
 * WHY IT IS GATED ON err_native. `write` is deliberately absent from
 * abnf/llvmmap.go's libcExterns, and llvm.Run has no standard error at all (its
 * RunResult carries only Out, filled by putchar), so under llvm.Run channel 1
 * is dropped. err_native is 1 ONLY in a native executable (bash-to-llvm-ir.abnf,
 * `c.exePath`), so the write is unreachable in the engine that cannot perform
 * it. This is the same ceiling as manual chapter 7.11.
 *
 * The INTERPRETER half writes channel 1 through the tag host's eprintln, so the
 * two halves a user actually runs agree; see bash-interpreter.abnf writeChan. */
int rt_putc(int c)
{
    int chan;
    int d;
    char *buf;
    int l;
    chan = sink_out;
    if (chan == 1) {
        if (err_native != 0) {
            rt_ech(c);
        }
        return 0;
    }
    if (chan == 0) {
        d = cap_depth;
        if (d > 0) {
            buf = cap_buf[d - 1];
            l = cap_len[d - 1];
            /* A full buffer GROWS - a fresh, doubled block out of the arena
             * and a copy - rather than truncating the substitution. */
            if (l + 1 >= cap_sz[d - 1]) {
                int nsz;
                char *nb;
                int t;
                nsz = cap_sz[d - 1] * 2;
                nb = rt_bump(nsz);
                t = 0;
                while (t < l) {
                    nb[t] = buf[t];
                    t = t + 1;
                }
                cap_buf[d - 1] = nb;
                cap_sz[d - 1] = nsz;
                buf = nb;
            }
            buf[l] = (char)c;
            cap_len[d - 1] = l + 1;
        } else {
            putchar(c);
        }
    }
    return 0;
}

/* Open a fresh capture buffer. */
int rt_cap_begin(int u)
{
    int d;
    char *buf;
    d = cap_depth;
    /* CAPMAX = 16. Dying, not refusing the push: the matching rt_cap_end
     * would still decrement cap_depth and index cap_buf[-1]. */
    if (d >= 16) {
        rt_die("command-substitution nesting", 1, d, 16);
    }
    buf = rt_bump(8192);
    cap_buf[d] = buf;
    cap_sz[d] = 8192;
    cap_len[d] = 0;
    cap_depth = d + 1;
    return 0;
}

/* Close the innermost buffer, drop its trailing newlines, NUL-terminate it. */
char *rt_cap_end(int u)
{
    int d2;
    char *buf;
    int l;
    d2 = cap_depth - 1;
    cap_depth = d2;
    buf = cap_buf[d2];
    l = cap_len[d2];
    while (l > 0) {
        if (buf[l - 1] != (char)10) {
            break;
        }
        l = l - 1;
    }
    buf[l] = (char)0;
    return buf;
}

/* Emit each byte until NUL (no trailing newline). */
int rt_prints(char *s)
{
    int i;
    i = 0;
    while (s[i] != (char)0) {
        rt_putc((int)(unsigned char)s[i]);
        i = i + 1;
    }
    return 0;
}

/* Push a whole variable snapshot. */
int rt_ss_save(int u)
{
    int d;
    int base;
    int abase;
    int i;
    d = ss_depth;
    /* SSDEPTH = 8: ss_save is 8 * NVARS pointers. */
    if (d >= 8) {
        rt_die("subshell nesting", 1, d, 8);
    }
    base = d * 1024;
    i = 0;
    while (i < nvars) {
        ss_save[base + i] = gvars[i];
        i = i + 1;
    }
    /* NARR = 4096 entries per depth, the same ceiling rt_arr_set enforces. */
    abase = d * 4096;
    i = 0;
    while (i < arr_n) {
        ss_anm[abase + i] = arr_nm[i];
        ss_ak[abase + i] = arr_k[i];
        ss_av[abase + i] = arr_v[i];
        i = i + 1;
    }
    ss_an[d] = arr_n;
    ss_depth = d + 1;
    return 0;
}

/* Pop a whole variable snapshot. */
int rt_ss_restore(int u)
{
    int d;
    int base;
    int abase;
    int n;
    int i;
    d = ss_depth - 1;
    ss_depth = d;
    base = d * 1024;
    i = 0;
    while (i < nvars) {
        gvars[i] = ss_save[base + i];
        i = i + 1;
    }
    abase = d * 4096;
    n = ss_an[d];
    i = 0;
    while (i < n) {
        arr_nm[i] = ss_anm[abase + i];
        arr_k[i] = ss_ak[abase + i];
        arr_v[i] = ss_av[abase + i];
        i = i + 1;
    }
    arr_n = n;
    return 0;
}

/* Remember gvars[id]'s current value on the local save stack. */
int rt_push_local(int id)
{
    int t;
    t = ls_top;
    /* LSMAX. Refusing the push would leak the "local" into the caller. */
    if (t >= 512) {
        rt_die("local save stack", 1, t, 512);
    }
    ls_id[t] = id;
    ls_val[t] = gvars[id];
    /* Copy this name's array rows aside. The scan is skipped entirely when the
     * program has no array live, which is the common case and is why `local`
     * in a hot recursive function does not pay for this. */
    int cnt = 0;
    ls_amark[t] = 0;
    if (arr_n > 0) {
        char *nm = varnames[id];
        ls_amark[t] = rt_arr_marked(nm) + 2 * rt_arr_dmarked(nm) + 4 * rt_arr_smarked(nm);
        int i = 0;
        while (i < arr_n) {
            if (rt_streq(nm, arr_nm[i]) != 0) {
                if (las_top >= 4096) {
                    rt_die("local array save stack", 1, las_top, 4096);
                }
                las_k[las_top] = arr_k[i];
                las_v[las_top] = arr_v[i];
                las_top = las_top + 1;
                cnt = cnt + 1;
            }
            i = i + 1;
        }
    }
    ls_acnt[t] = cnt;
    ls_top = t + 1;
    return 0;
}

/* Put entry t's saved binding back; its array rows start at las index `rows`. */
int rt_ls_restore(int t, int rows)
{
    gvars[ls_id[t]] = ls_val[t];
    int cnt = ls_acnt[t];
    if (arr_n > 0 || cnt > 0) {
        char *nm = varnames[ls_id[t]];
        rt_arr_clear(nm);
        int i = rows;
        while (i < rows + cnt) {
            rt_arr_set(nm, las_k[i], las_v[i]);
            i = i + 1;
        }
        if ((ls_amark[t] & 1) != 0) rt_arr_mark(nm);
        if ((ls_amark[t] & 2) != 0) rt_arr_dmark(nm);
        if ((ls_amark[t] & 4) != 0) rt_arr_smark(nm);
    }
    return 0;
}

/* Restore every entry above `d` (innermost first). An id of -1 is an entry
 * rt_unset_up already retired: it only releases its array rows. */
int rt_pop_locals(int d)
{
    int t;
    t = ls_top;
    while (t > d) {
        t = t - 1;
        ls_top = t;
        int cnt = ls_acnt[t];
        if (ls_id[t] >= 0) rt_ls_restore(t, las_top - cnt);
        las_top = las_top - cnt;
        t = ls_top;
    }
    return 0;
}

/* `unset` of a name that is a local of a CALLER: bash drops that caller's
 * binding, so the next one outward is visible again. `d` is the current call's
 * save-stack depth at entry and every call's first entry is its FUNCNAME (slot
 * `fn`), which is how the frames below d are told apart. The innermost frame
 * holding `id` loses it as a whole: its LOWEST entry (the value before the
 * first `local`) is restored and every entry of `id` in it is retired, so the
 * frame's return restores nothing over later writes. Answers 1 when a frame
 * held `id`. Twin of unsetCallerLocal in bash-interpreter.abnf. */
int rt_unset_up(int id, int d, int fn)
{
    int top = d;
    int low = -1;
    int t = d - 1;
    while (t >= 0) {
        if (ls_id[t] == id) low = t;
        if (ls_id[t] == fn) {
            if (low >= 0) {
                int rows = las_top;
                int i = ls_top - 1;
                while (i >= low) {
                    rows = rows - ls_acnt[i];
                    i = i - 1;
                }
                rt_ls_restore(low, rows);
                i = t;
                while (i < top) {
                    if (ls_id[i] == id) ls_id[i] = -1;
                    i = i + 1;
                }
                return 1;
            }
            top = t;
        }
        t = t - 1;
    }
    return 0;
}

/* ---- name -> slot ---------------------------------------------------------- */
/* Ordinary loops over the varnames[] {name, slot} table, so the emitted module
 * does not grow with the number of distinct variables. */

/* ${!name}: the value of the variable whose NAME is `n`, or the unset marker.
 * Names are distinct, so at most one slot can match. */
/* "$*" joins with the FIRST character of IFS (a space when IFS is unset). The
 * emitter has ifsFirstValue for the same thing; this is the copy the runtime's
 * own ${!r} path needs, and IFS is an ordinary named slot so the lookup below
 * cannot recurse into a special. */
char *rt_ifsfirst(void)
{
    char *v = rt_getvar_byname("IFS");
    if (v == &unset_marker[0]) return " ";
    return rt_substr(v, 0, 1);
}

char *rt_getvar_byname(char *n)
{
    char *sel;
    int i;
    int c0;
    int allDigits;
    /* THE SPECIAL PARAMETERS ARE NOT IN varnames[]: $0..$9, $#, $@, $* and $?
     * have no slot (`z=0; echo "${!z}"` is the script path in bash).
     * languages/bash-interpreter.abnf's getVarVal answers the same set. */
    if (rt_streq(n, "?") != 0) return rt_int2str(last_status);
    if (rt_streq(n, "#") != 0) return rt_int2str(frame_n);
    /* ${!r} answers ONE string - the indirection path has no way to hand a field
     * list back - so both $@ and $* come through as the joined form, which is
     * what `"${!r}"` prints in bash either way. */
    if (rt_streq(n, "*") != 0 || rt_streq(n, "@") != 0) return rt_params(rt_ifsfirst(), 1);
    if (rt_streq(n, "$") != 0) return "1000";
    if (rt_streq(n, "!") != 0) return "0";
    if (rt_streq(n, "-") != 0) return "hB";
    c0 = rt_strlen(n);
    allDigits = c0 > 0 ? 1 : 0;
    i = 0;
    while (i < c0) {
        int ch = (int)(unsigned char)n[i];
        if (ch < 48 || ch > 57) allDigits = 0;
        i = i + 1;
    }
    if (allDigits != 0) {
        int k = rt_str2int(n);
        if (k == 0) return script_name;
        return rt_param(k);
    }
    sel = &unset_marker[0];
    i = 0;
    while (i < nvars) {
        if (rt_streq(n, varnames[i]) != 0) {
            sel = gvars[i];
        }
        i = i + 1;
    }
    return sel;
}

/* The same lookup, writing instead of reading. A name the program never
 * mentions has no slot and is silently dropped. */
int rt_setvar_byname(char *n, char *v)
{
    int i;
    i = 0;
    while (i < nvars) {
        if (rt_streq(n, varnames[i]) != 0) {
            gvars[i] = v;
        }
        i = i + 1;
    }
    return 0;
}

/* `NAME=VALUE` as one string (what `export` / `declare` / `eval` hand over):
 * split at the FIRST '=' and assign. No '=' at all assigns nothing. */
int rt_eval_assign(char *s)
{
    int len;
    int i;
    len = rt_strlen(s);
    i = 0;
    while (i < len) {
        if (s[i] == 61) {
            rt_setvar_byname(rt_substr(s, 0, i), rt_substr(s, i + 1, -1));
            return 0;
        }
        i = i + 1;
    }
    return 0;
}

/* There is no real filesystem behind a compiled run, so the file predicates go
 * through a model of the standard paths: 0 = absent, 1 = regular file,
 * 2 = directory, 3 = character device.
 * Every comparison is performed, in order; the last hit wins. */
int rt_fskind(char *p)
{
    int k;
    k = 0;
    /* CHARACTER DEVICES, not regular files: `[ -f /dev/null ]` is false in
     * bash 5.3. Why this is a model and not a stat() is written out at fsKind
     * in languages/bash-interpreter.abnf. */
    k = rt_streq(p, "/dev/null") != 0 ? 3 : k;
    k = rt_streq(p, "/dev/zero") != 0 ? 3 : k;
    k = rt_streq(p, "/dev/tty") != 0 ? 3 : k;
    k = rt_streq(p, "/dev/stdin") != 0 ? 3 : k;
    k = rt_streq(p, "/dev/stdout") != 0 ? 3 : k;
    k = rt_streq(p, "/dev/stderr") != 0 ? 3 : k;
    k = rt_streq(p, "/") != 0 ? 2 : k;
    k = rt_streq(p, "/dev") != 0 ? 2 : k;
    k = rt_streq(p, "/tmp") != 0 ? 2 : k;
    k = rt_streq(p, "/usr") != 0 ? 2 : k;
    k = rt_streq(p, "/etc") != 0 ? 2 : k;
    k = rt_streq(p, "/var") != 0 ? 2 : k;
    k = rt_streq(p, "/bin") != 0 ? 2 : k;
    k = rt_streq(p, "/usr/bin") != 0 ? 2 : k;
    return k;
}


/* ==== [[ str =~ ere ]] - the POSIX ERE engine =============================
 *
 * Contract:
 *   rt_regex_search > 0  matched; the number of capture PAIRS (${#BASH_REMATCH[@]})
 *                  == 0  no match
 *                  == -1 the pattern did not compile; rt_regex_error() has the text
 *                  == -2 the engine's step cap was hit on a pathological pattern
 * caps is caller-allocated, ncaps SLOTS (ncaps/2 pairs), pre-filled with -1 so a
 * group that did not participate reads -1/-1. Offsets are BYTES into the subject.
 *
 * Design: Thompson-style codegen plus a recursive backtracking VM.
 *   * re_alt/re_cat/re_rep/re_atom parse the ERE and emit 4-word instructions
 *     into re_prog. Recursion is the C call stack, as rt_glob already does it.
 *   * re_run walks that program, recursing at SPLIT. Leftmost-first, greedy.
 *   * a * or + loop is fenced by SAVE/PROG on a scratch slot, so an iteration
 *     that consumed nothing cannot repeat.
 *
 * Sizes (the grammar's RE_* constants), written as literals because the C
 * subset has no #define:
 *   RE_PROG  4096   instruction slots, 3 words each  -> re_prog[12288]
 *   RE_NCLS    64   bracket bitmaps, 32 bytes each   -> re_cls[2048]
 *   RE_NCAP   128   capture slots (64 pairs)
 *   RE_NMARK   64   empty-iteration guard slots
 *   RE_SLOTS  192   = RE_NCAP + RE_NMARK             -> re_slot[192]
 *   RE_STEPS 400000 total VM steps per rt_regex_search call
 *   RE_CSTK  8192   the OP_CLEAR undo stack          -> re_cstk[8192]
 *
 * Opcodes, as literals:
 *   1 CHAR   2 ANY   3 CLASS   4 SPLIT   5 JMP   6 SAVE
 *   7 MATCH  8 BOL   9 EOL    10 PROG   11 CLEAR
 * OP_CLEAR x,y sets capture slots x..y to -1; x > y is a legal no-op.
 */

/* ---- engine state -------------------------------------------------------- */

int re_prog[12288];           /* NEW GLOBAL: RE_PROG * 3 words              */
unsigned char re_cls[2048];   /* NEW GLOBAL: RE_NCLS * 32 bitmap bytes      */
int re_slot[192];             /* NEW GLOBAL: RE_SLOTS capture / mark slots  */
int re_pc;                    /* NEW GLOBAL: next instruction to emit       */
char *re_pat;                 /* NEW GLOBAL: the pattern being compiled     */
int re_pos;                   /* NEW GLOBAL: the pattern cursor             */
int re_ng;                    /* NEW GLOBAL: groups seen so far             */
int re_ncls;                  /* NEW GLOBAL: bitmaps allocated              */
int re_nmark;                 /* NEW GLOBAL: guard slots handed out         */
int re_depth;                 /* NEW GLOBAL: open-group depth               */
char *re_err;                 /* NEW GLOBAL: last compile error text        */
char *re_subj;                /* NEW GLOBAL: the subject being searched     */
int re_slen;                  /* NEW GLOBAL: its length in bytes            */
int re_steps;                 /* NEW GLOBAL: VM steps used                  */
/* The undo stack for OP_CLEAR. A cleared slot has to come BACK when the
 * iteration that cleared it fails, exactly as OP_SAVE puts its old offset back -
 * but a CLEAR writes a whole range, too big for the re_run frame (one frame per
 * backtracking choice). So the old values go on a stack of their own, and an
 * overrun fails the attempt the way the step cap does. */
int re_cstk[8192];            /* NEW GLOBAL                                 */
int re_ctop;                  /* NEW GLOBAL                                 */
int re_fold;                  /* NEW GLOBAL: the "i" flag                   */
int re_bad;                   /* NEW GLOBAL: compile error code, 0 = fine   */

/* The mutually recursive halves, declared before any body needs them. */
int re_alt(void);
int re_cat(void);
int re_rep(void);
int re_atom(void);
int re_run(int rpc, int rsp);

/* ---- program store ------------------------------------------------------- */

/* word k of instruction pc */
int re_get(int gpc, int gk)
{
    return re_prog[gpc * 3 + gk];
}

int re_set(int spc, int sk, int sv)
{
    re_prog[spc * 3 + sk] = sv;
    return 0;
}

/* append one instruction, answer its pc. An overrun raises re_bad rather than
 * writing past re_prog. An instruction is THREE words (opcode, x, y). */
int re_emit(int eop, int ex, int ey)
{
    int pc;
    pc = re_pc;
    if (pc >= 4096) {
        re_bad = 4;               /* Regular expression too big */
        return 0;
    }
    re_set(pc, 0, eop);
    re_set(pc, 1, ex);
    re_set(pc, 2, ey);
    re_pc = pc + 1;
    return pc;
}

/* the next empty-iteration guard slot. Slots wrap rather than run out, so a
 * pattern with more than RE_NMARK loops degrades to sharing a guard instead of
 * indexing off the end of re_slot. */
int re_mark(void)
{
    int n;
    n = re_nmark;
    re_nmark = n + 1;
    return 128 + n % 64;
}

/* ---- pattern cursor ------------------------------------------------------ */

/* the pattern byte d ahead of the cursor, 0 at or past the end. The scan STOPS
 * at the NUL, so a two-character lookahead at the last character never reads the
 * byte after the terminator. */
int re_at(int atd)
{
    char *p;
    int i;
    int k;
    int c;
    p = re_pat;
    i = re_pos;
    k = 0;
    c = 0;
    while (1) {
        c = (int)(unsigned char)p[i];
        if (!(c != 0 && k < atd)) {
            break;
        }
        i = i + 1;
        k = k + 1;
    }
    return c;
}

/* the decimal number at the cursor, clamped, cursor advanced past it. */
int re_num(void)
{
    int v;
    int c;
    v = 0;
    while (1) {
        c = re_at(0);
        if (!(c >= 48 && c <= 57)) {
            break;
        }
        v = (v < 256) ? (v * 10 + (c - 48)) : 256;
        re_pos = re_pos + 1;
    }
    return v;
}

/* 1 when the cursor sits on a quantifier. bash 5.3 rejects a SECOND one outright
 * - "a**", "a*?", "a+*", "a{2}{3}" and "a*{2}" are all status 2 - so this is the
 * guard, not a fold. (A "{" that is not followed by a digit is a literal brace
 * and therefore not a quantifier: "a{b" matches.) */
int re_isq(void)
{
    int c;
    int d;
    int simple;
    int iv;
    c = re_at(0);
    d = re_at(1);
    simple = (c == 42) | (c == 43) | (c == 63);
    iv = (c == 123) & (d >= 48 && d <= 57);
    return (simple | iv) ? 1 : 0;
}

/* c lowercased when the i flag is on, else c unchanged. */
int re_fold1(int fdc)
{
    int isU;
    isU = (fdc >= 65 && fdc <= 90);
    return (re_fold != 0 && isU) ? (fdc + 32) : fdc;
}

/* ---- bracket-expression bitmaps ------------------------------------------ */

int re_bit1(int b1k, int b1c)
{
    int idx;
    idx = b1k * 32 + b1c / 8;
    re_cls[idx] = re_cls[idx] | (unsigned char)(1 << (b1c % 8));
    return 0;
}

/* Under the i flag a letter joins the set in BOTH cases, which is why the VM
 * only has to fold the subject byte. */
int re_setbit(int sbk, int sbc)
{
    int isU;
    int isL;
    re_bit1(sbk, sbc);
    isU = (sbc >= 65 && sbc <= 90);
    isL = (sbc >= 97 && sbc <= 122);
    if (re_fold != 0 && (isU | isL)) {
        re_bit1(sbk, isU ? (sbc + 32) : (sbc - 32));
    }
    return 0;
}

int re_clrcls(int cck)
{
    int i;
    int base;
    base = cck * 32;
    for (i = 0; i < 32; i++) {
        re_cls[base + i] = 0;
    }
    return 0;
}

int re_negcls(int nck)
{
    int i;
    int base;
    base = nck * 32;
    for (i = 0; i < 32; i++) {
        re_cls[base + i] = re_cls[base + i] ^ 255;
    }
    return 0;
}

/* membership of the POSIX character classes, branch-free. */
int re_ctype(int ctid, int ctch)
{
    int upper;
    int lower;
    int digit;
    int alpha;
    int alnum;
    int graph;
    int r;
    upper = (ctch >= 65 && ctch <= 90) ? 1 : 0;
    lower = (ctch >= 97 && ctch <= 122) ? 1 : 0;
    digit = (ctch >= 48 && ctch <= 57) ? 1 : 0;
    alpha = upper | lower;
    alnum = alpha | digit;
    graph = (ctch >= 33 && ctch <= 126) ? 1 : 0;
    r = 0;
    r = (ctid == 1) ? alpha : r;
    r = (ctid == 2) ? digit : r;
    r = (ctid == 3) ? alnum : r;
    r = (ctid == 4) ? upper : r;
    r = (ctid == 5) ? lower : r;
    /* space */
    r = (ctid == 6) ? (((ctch == 32) ? 1 : 0) | ((ctch >= 9 && ctch <= 13) ? 1 : 0)) : r;
    /* blank */
    r = (ctid == 7) ? (((ctch == 32) ? 1 : 0) | ((ctch == 9) ? 1 : 0)) : r;
    /* punct */
    r = (ctid == 8) ? (graph & (1 - alnum)) : r;
    /* print */
    r = (ctid == 9) ? ((ctch >= 32 && ctch <= 126) ? 1 : 0) : r;
    r = (ctid == 10) ? graph : r;
    /* cntrl */
    r = (ctid == 11) ? (((ctch >= 0 && ctch <= 31) ? 1 : 0) | ((ctch == 127) ? 1 : 0)) : r;
    /* xdigit */
    r = (ctid == 12) ? (digit | (((ctch >= 97 && ctch <= 102) ? 1 : 0)
                                 | ((ctch >= 65 && ctch <= 70) ? 1 : 0))) : r;
    /* word */
    r = (ctid == 13) ? (alnum | ((ctch == 95) ? 1 : 0)) : r;
    /* ascii */
    r = (ctid == 14) ? ((ctch >= 0 && ctch <= 127) ? 1 : 0) : r;
    return r;
}

/* the cursor sits just after "[:". Answers the re_ctype id and steps past the
 * closing ":]", or 0 when the name is unknown or the "[:" never closes.
 *
 * The name is matched WHOLE: bash 5.3.15 rejects [[:bogus:]] and
 * [[:lowercase:]] with "invalid character class" and status 2, so an unknown
 * name returns 0, which re_bracket turns into re_bad = 9. */
int re_clsname(void)
{
    int d;
    int hc;
    char *nm;
    int id;
    /* the name runs up to ":]"; an unterminated "[:" is not a name at all. */
    d = 0;
    while (1) {
        hc = re_at(d);
        if (hc == 0) {
            return 0;
        }
        if (hc == 58 && re_at(d + 1) == 93) {
            break;
        }
        d = d + 1;
    }
    nm = rt_bump(d + 1);
    hc = 0;
    while (hc < d) {
        nm[hc] = (char)re_at(hc);
        hc = hc + 1;
    }
    nm[d] = 0;
    id = rt_clsid(nm);
    if (id == 0) {
        return 0;             /* re_bracket raises re_bad = 9 */
    }
    re_pos = re_pos + (d + 2);
    return id;
}

/* parse [...] at the cursor into a bitmap and emit CLASS.
 * Settled against bash 5.3: a leading "]" is a literal, "\" inside a bracket is
 * a LITERAL backslash (so [a\-b] is {a} plus the range \..b, and does NOT match
 * "-"), and a reversed range like [z-a] is a compile error. */
int re_bracket(void)
{
    int k;
    int c0;
    int isNeg;
    int neg;
    int first;
    int c;
    int cid;
    int j;
    int jto;
    int pd0;
    int pd1;
    int nxt;
    int nxt2;
    int nxt3;
    k = re_ncls;
    if (k >= 64) {
        re_bad = 4;               /* Regular expression too big */
        return 0;
    }
    re_ncls = k + 1;
    re_clrcls(k);
    re_pos = re_pos + 1;
    c0 = re_at(0);
    isNeg = (c0 == 94);
    neg = isNeg ? 1 : 0;
    re_pos = re_pos + (isNeg ? 1 : 0);
    first = 1;
    while (1) {
        c = re_at(0);
        if (c == 0) {
            re_bad = 2;           /* Unmatched [, [^, [:, [. or [= */
            return 0;
        }
        if (c == 93 && first == 0) {
            break;                /* close */
        }
        first = 0;
        if (c == 91 && re_at(1) == 58) {
            /* a POSIX class name */
            re_pos = re_pos + 2;
            cid = re_clsname();
            if (cid == 0) {
                re_bad = 9;       /* Invalid character class name */
                return 0;
            }
            for (j = 0; j < 256; j++) {
                if (re_ctype(cid, j) != 0) {
                    re_setbit(k, j);
                }
            }
            /* A POSIX class is a SET, so it cannot be an endpoint of a range:
             * bash reports "invalid character range" for [[:alpha:]-z] and for
             * [a-[:digit:]]. A trailing "-]" is still an ordinary hyphen, so the
             * test is the same one the range below uses. */
            pd0 = re_at(0);
            pd1 = re_at(1);
            if (pd0 == 45 && (pd1 != 93 && pd1 != 0)) {
                re_bad = 7;       /* Invalid range end */
                return 0;
            }
            continue;
        }
        /* an ordinary item */
        re_pos = re_pos + 1;
        nxt = re_at(0);
        nxt2 = re_at(1);
        nxt3 = re_at(2);
        if (nxt == 45 && (nxt2 != 93 && nxt2 != 0)) {
            if (nxt2 == 91 && nxt3 == 58) {
                re_bad = 7;
                return 0;
            }
            re_pos = re_pos + 2;
            if (c > nxt2) {
                re_bad = 7;
                return 0;
            }
            jto = nxt2 + 1;
            for (j = c; j < jto; j++) {
                re_setbit(k, j);
            }
            continue;
        }
        re_setbit(k, c);
    }
    /* close */
    re_pos = re_pos + 1;
    if (neg != 0) {
        re_negcls(k);
    }
    re_emit(3, k, 0);             /* OP_CLASS */
    return 0;
}

/* append a copy of instructions [from,to), relocating the jump targets that
 * point INSIDE the fragment (including its one-past-the-end label). A fragment
 * is self-contained by construction, so no other target can occur. SAVE and PROG
 * operands are slot ids, not pcs, and are deliberately left alone - copies of
 * one loop body share a guard slot, which is correct because MARK saves and
 * restores it. */
int re_copy(int cpfrom, int cpto)
{
    int delta;
    int i;
    int op;
    int vx;
    int vy;
    int isJ;
    int xin;
    int yin;
    int nx;
    int ny;
    delta = re_pc - cpfrom;
    for (i = cpfrom; i < cpto && re_bad == 0; i++) {
        op = re_get(i, 0);
        vx = re_get(i, 1);
        vy = re_get(i, 2);
        isJ = (op == 5) | (op == 4);                  /* JMP or SPLIT */
        xin = (vx >= cpfrom) & (vx <= cpto);
        yin = (vy >= cpfrom) & (vy <= cpto);
        nx = (isJ & xin) ? (vx + delta) : vx;
        ny = ((op == 4) & yin) ? (vy + delta) : vy;
        re_emit(op, nx, ny);
    }
    return 0;
}

/* one ERE atom. */
int re_atom(void)
{
    int c;
    int gn;
    int fits;
    int ec;
    c = re_at(0);

    if (c == 40) {                /* ( */
        /* A group numbers itself on its OPENING paren, which is what bash
         * reports, so the index is taken before re_alt can nest another one.
         * Past RE_NCAP/2 groups the brackets are dropped but the group is still
         * COUNTED, matching the call site's "true pair count even when it
         * overflows caps". */
        re_pos = re_pos + 1;
        gn = re_ng + 1;
        re_ng = gn;
        fits = (gn * 2 + 1) < 128;
        if (fits) {
            re_emit(6, gn * 2, 0);            /* OP_SAVE */
        }
        re_depth = re_depth + 1;
        re_alt();
        re_depth = re_depth - 1;
        if (re_bad != 0) {
            return 0;
        }
        if (re_at(0) != 41) {
            re_bad = 1;                       /* Unmatched ( or \( */
            return 0;
        }
        re_pos = re_pos + 1;
        if (fits) {
            re_emit(6, gn * 2 + 1, 0);
        }
        return 0;
    }
    if (c == 46) {                /* . */
        re_pos = re_pos + 1;
        re_emit(2, 0, 0);                     /* OP_ANY */
        return 0;
    }
    /* ^ and $ are anchors ANYWHERE in an ERE, not only at the ends: bash 5.3
     * says [[ "a^b" =~ a^b ]] is false. */
    if (c == 94) {                /* ^ */
        re_pos = re_pos + 1;
        re_emit(8, 0, 0);                     /* OP_BOL */
        return 0;
    }
    if (c == 36) {                /* $ */
        re_pos = re_pos + 1;
        re_emit(9, 0, 0);                     /* OP_EOL */
        return 0;
    }
    if (c == 91) {                /* [ */
        re_bracket();
        return 0;
    }
    if (c == 92) {                /* backslash */
        ec = re_at(1);
        if (ec == 0) {
            re_bad = 6;                       /* Trailing backslash */
            return 0;
        }
        re_pos = re_pos + 2;
        re_emit(1, re_fold1(ec), 0);          /* OP_CHAR */
        return 0;
    }
    if (c == 42 || c == 43 || c == 63) {      /* * + ? with nothing to repeat */
        re_bad = 5;               /* Invalid preceding regular expression */
        return 0;
    }
    re_pos = re_pos + 1;
    re_emit(1, re_fold1(c), 0);
    return 0;
}

/* an atom plus its quantifier. TWO slots are reserved ahead of the atom - one
 * for the quantifier's SPLIT/JMP and one for the loop's empty-iteration MARK -
 * so neither has to be inserted in front of code that is already emitted. */
int re_rep(void)
{
    int i;
    int j;
    int end;
    int empty_ok;
    int inf;
    int any;
    int m;
    int n;
    int hole1;
    int hole2;
    int hole3;
    int g0;
    int g1;
    int astart;
    int aend;
    int hasG;
    int hiSlot;
    int qc;
    int qd;
    int sc;
    int gotq;
    int pt;
    int ps;
    int st;
    int mv;
    int nv;
    int mv2;
    int it0;
    int it1;
    int isp;
    int imk;
    int s;
    int cs;
    int endv;
    int pv;

    empty_ok = 0;
    inf = 0;
    any = 0;
    end = -1;
    hole1 = re_emit(5, 0, 0);                 /* OP_JMP hole */
    hole2 = re_emit(5, 0, 0);
    /* A THIRD reserved slot, and it is deliberately INSIDE the atom's fragment:
     * it carries the per-iteration capture reset, so every copy re_copy makes
     * for an interval carries its own. Which groups to clear is only known once
     * the atom has been parsed, hence the hole. */
    g0 = re_ng;
    hole3 = re_emit(5, 0, 0);
    astart = hole3;
    re_atom();
    aend = re_pc;
    g1 = re_ng;
    /* The captures inside a repeated atom are cleared at the start of every
     * iteration, so a group that took part in an earlier iteration but not the
     * last one reads as unset: [[ ab =~ ^((a)|b*)*$ ]] leaves BASH_REMATCH[2]
     * empty. The group numbers inside the atom are contiguous, so one range
     * covers them; with no groups at all the slot degrades to a jump onto the
     * atom. */
    hasG = (g1 > g0);
    re_set(hole3, 0, hasG ? 11 : 5);          /* OP_CLEAR or OP_JMP */
    re_set(hole3, 1, hasG ? ((g0 + 1) * 2) : (hole3 + 1));
    /* Groups past the 64th get no SAVE either (the call site's caps array ends),
     * so the range stops at the last real capture slot and never reaches the
     * marks that share re_slot with them. */
    hiSlot = g1 * 2 + 1;
    re_set(hole3, 2, (hiSlot < 128) ? hiSlot : 127);
    if (re_bad != 0) {
        return 0;
    }

    /* "{" opens an interval only when a DIGIT follows; bash 5.3 takes "a{b" and
     * "a{" as a literal brace but rejects "a{1" outright. */
    qc = re_at(0);
    qd = re_at(1);
    if (!(qc == 123 && (qd >= 48 && qd <= 57))) {
        /* ---- simple quantifiers. Exactly ONE is allowed; a second is an error. */
        sc = re_at(0);
        gotq = 0;
        if (sc == 42) {                       /* * */
            re_pos = re_pos + 1;
            empty_ok = 1;
            inf = 1;
            any = 1;
            gotq = 1;
        } else if (sc == 43) {                /* + */
            re_pos = re_pos + 1;
            inf = 1;
            any = 1;
            gotq = 1;
        } else if (sc == 63) {                /* ? */
            re_pos = re_pos + 1;
            empty_ok = 1;
            any = 1;
            gotq = 1;
        }
        if (gotq) {
            if (re_isq() != 0) {
                re_bad = 5;
                return 0;
            }
        }
        if (any == 0) {
            re_set(hole1, 1, hole2);
            re_set(hole2, 1, astart);
            return 0;
        }
        if (inf == 0) {                       /* ? */
            re_set(hole1, 0, 4);              /* OP_SPLIT */
            re_set(hole1, 1, hole2);
            re_set(hole1, 2, re_pc);
            re_set(hole2, 1, astart);
            return 0;
        }
        if (empty_ok == 0) {                  /* + */
            pt = re_mark();
            re_set(hole1, 1, hole2);
            re_set(hole2, 0, 6);              /* OP_SAVE */
            re_set(hole2, 1, pt);
            re_emit(10, pt, 0);               /* OP_PROG */
            ps = re_emit(4, hole2, 0);        /* OP_SPLIT */
            re_set(ps, 2, re_pc);
            return 0;
        }
        /* * */
        st = re_mark();
        re_set(hole2, 0, 6);
        re_set(hole2, 1, st);
        re_emit(10, st, 0);
        re_emit(5, hole1, 0);
        re_set(hole1, 0, 4);
        re_set(hole1, 1, hole2);
        re_set(hole1, 2, re_pc);
        return 0;
    }

    /* ---- {m}, {m,}, {m,n} */
    re_pos = re_pos + 1;
    mv = re_num();
    m = mv;
    if (re_at(0) == 44) {                     /* , */
        re_pos = re_pos + 1;
        if (re_at(0) == 125) {                /* } */
            n = -1;
        } else {
            n = re_num();
        }
    } else {
        n = mv;
    }
    if (re_at(0) != 125) {
        re_bad = 3;                           /* Invalid content of \{\} */
        return 0;
    }
    re_pos = re_pos + 1;
    nv = n;
    mv2 = m;
    if (nv >= 0 && nv < mv2) {
        re_bad = 3;
        return 0;
    }
    /* "a{2}{3}" is status 2 in bash 5.3, exactly like "a**". */
    if (re_isq() != 0) {
        re_bad = 5;
        return 0;
    }
    if (nv == 0 && mv2 == 0) {
        /* {0}: the atom never runs at all - jump straight past it. */
        re_set(hole1, 1, aend);
        return 0;
    }
    if (nv < 0) {
        /* {m,}: m mandatory copies, then a starred one. */
        if (mv2 == 0) {
            it0 = re_mark();
            re_set(hole2, 0, 6);
            re_set(hole2, 1, it0);
            re_emit(10, it0, 0);
            re_emit(5, hole1, 0);
            re_set(hole1, 0, 4);
            re_set(hole1, 1, hole2);
            re_set(hole1, 2, re_pc);
            return 0;
        }
        re_set(hole1, 1, hole2);
        re_set(hole2, 1, astart);
        for (i = 1; i < mv2 && re_bad == 0; i++) {
            re_copy(astart, aend);
        }
        isp = re_emit(4, 0, 0);
        it1 = re_mark();
        imk = re_emit(6, it1, 0);
        re_copy(astart, aend);
        re_emit(10, it1, 0);
        re_emit(5, isp, 0);
        re_set(isp, 1, imk);
        re_set(isp, 2, re_pc);
        return 0;
    }

    /* {m,n}: n-m of the copies are optional. Each optional SPLIT parks its END
     * target in word 2 as a chain link and the walk at the bottom fills them
     * all in. */
    if (mv2 == 0) {
        re_set(hole1, 0, 4);
        re_set(hole1, 1, hole2);
        re_set(hole1, 2, end);
        end = hole1;
        re_set(hole2, 1, astart);
        for (i = 1; i < nv && re_bad == 0; i++) {
            s = re_emit(4, 0, end);
            end = s;
            cs = re_pc;
            re_copy(astart, aend);
            re_set(s, 1, cs);
        }
    } else {
        re_set(hole1, 1, hole2);
        re_set(hole2, 1, astart);
        for (i = 1; i < mv2 && re_bad == 0; i++) {
            re_copy(astart, aend);
        }
        for (j = mv2; j < nv && re_bad == 0; j++) {
            s = re_emit(4, 0, end);
            end = s;
            cs = re_pc;
            re_copy(astart, aend);
            re_set(s, 1, cs);
        }
    }
    endv = re_pc;
    while (end >= 0) {
        pv = end;
        end = re_get(pv, 2);
        re_set(pv, 2, endv);
    }
    return 0;
}

/* a run of quantified atoms; answers how many there were, so re_alt can reject
 * an EMPTY branch - bash 5.3 rejects "(a|)", "|a" and "" alike, status 2. */
int re_cat(void)
{
    int n;
    int c;
    int isRp;
    int stop;
    n = 0;
    while (1) {
        c = re_at(0);
        /* A ")" only closes a group when one is open; at the top level bash
         * takes it as a literal - [[ "a)b" =~ ")" ]] matches. */
        isRp = (c == 41) & (re_depth > 0);
        stop = (c == 0) | (c == 124) | isRp;
        if (stop | (re_bad != 0)) {
            break;
        }
        re_rep();
        n = n + 1;
    }
    return n;
}

/* branch | branch | ... */
int re_alt(void)
{
    int pend;
    /* An empty BRANCH is not decided here: whether it is an error depends on how
     * many branches there turn out to be. empty_seen remembers that one was seen
     * and nalt counts them; the verdict is at the end. */
    int empty_seen;
    int nalt;
    int sp;
    int body;
    int n;
    int jj;
    int endv;
    int pv;
    pend = -1;
    empty_seen = 0;
    nalt = 0;
    sp = 0;
    body = 0;
    while (1) {
        nalt = nalt + 1;
        sp = re_emit(5, 0, 0);                /* OP_JMP hole */
        body = re_pc;
        n = re_cat();
        if (n == 0 && re_bad == 0) {
            empty_seen = 1;
        }
        if (re_bad != 0) {
            break;
        }
        if (re_at(0) != 124) {                /* | */
            break;
        }
        re_pos = re_pos + 1;
        jj = re_emit(5, 0, pend);
        pend = jj;
        re_set(sp, 0, 4);                     /* OP_SPLIT */
        re_set(sp, 1, body);
        re_set(sp, 2, re_pc);
    }
    re_set(sp, 1, body);
    endv = re_pc;
    while (pend >= 0) {
        pv = pend;
        pend = re_get(pv, 2);
        re_set(pv, 1, endv);
    }
    /* A POSIX ERE branch may not be empty: bash rejects "", "a|", "|a", "(a|)"
     * and "(|a)" alike with "empty (sub)expression". The one exception is a
     * group whose WHOLE body is empty - "()", "(())" and "(a)()" match the empty
     * string in bash 5.3 - so the emptiness is only an error for a branch of a
     * real alternation, or for the top level of the pattern. */
    if ((empty_seen != 0) & ((nalt > 1) | (re_depth == 0)) & (re_bad == 0)) {
        re_bad = 8;                           /* empty (sub)expression */
    }
    return 0;
}

/* the backtracking VM. 1 match, 0 fail, -2 step cap. */
int re_run(int rpc, int rsp)
{
    int pc;
    int sp;
    int st;
    int op;
    int vx;
    int vy;
    int slen;
    char *subj;
    int atEnd;
    int pc1;
    int sp1;
    int cb;
    int lb;
    int byt;
    int got;
    int sr;
    int old;
    int vr;
    int cbase;
    int cn;
    int cr;
    int i;

    pc = rpc;
    sp = rsp;
    while (1) {
        st = re_steps;
        if (st >= 400000) {
            return -2;
        }
        re_steps = st + 1;
        op = re_get(pc, 0);
        vx = re_get(pc, 1);
        vy = re_get(pc, 2);
        slen = re_slen;
        subj = re_subj;
        atEnd = (sp >= slen);
        pc1 = pc + 1;
        sp1 = sp + 1;

        /* Every byte read is fenced by sp < slen, so no path loads the NUL
         * terminator or anything after it. */
        if (op == 1) {                        /* OP_CHAR */
            if (atEnd) {
                return 0;
            }
            cb = re_fold1((int)(unsigned char)subj[sp]);
            if (cb != vx) {
                return 0;
            }
            pc = pc1;
            sp = sp1;
            continue;
        }
        if (op == 2) {                        /* OP_ANY */
            if (atEnd) {
                return 0;
            }
            pc = pc1;
            sp = sp1;
            continue;
        }
        if (op == 3) {                        /* OP_CLASS */
            if (atEnd) {
                return 0;
            }
            lb = re_fold1((int)(unsigned char)subj[sp]);
            byt = (int)(unsigned char)re_cls[vx * 32 + lb / 8];
            got = (byt >> (lb % 8)) & 1;
            if (got == 0) {
                return 0;
            }
            pc = pc1;
            sp = sp1;
            continue;
        }
        if (op == 4) {                        /* OP_SPLIT */
            /* The C call stack IS the backtracking stack. */
            sr = re_run(vx, sp);
            if (sr != 0) {
                return sr;
            }
            pc = vy;
            continue;
        }
        if (op == 5) {                        /* OP_JMP */
            pc = vx;
            continue;
        }
        if (op == 6) {                        /* OP_SAVE */
            /* A capture (or a loop's empty-iteration mark) is UNDONE when the
             * branch that set it fails, so a dead alternative cannot leave an
             * offset behind. */
            old = re_slot[vx];
            re_slot[vx] = sp;
            vr = re_run(pc1, sp);
            if (vr == 0) {
                re_slot[vx] = old;
            }
            return vr;
        }
        if (op == 7) {                        /* OP_MATCH */
            return 1;
        }
        if (op == 8) {                        /* OP_BOL */
            if (sp != 0) {
                return 0;
            }
            pc = pc1;
            continue;
        }
        if (op == 9) {                        /* OP_EOL */
            if (sp != slen) {
                return 0;
            }
            pc = pc1;
            continue;
        }
        if (op == 10) {                       /* OP_PROG */
            if (sp == re_slot[vx]) {
                return 0;
            }
            pc = pc1;
            continue;
        }
        if (op == 11) {                       /* OP_CLEAR */
            /* The same write / recurse / put-it-back discipline as OP_SAVE, over
             * a RANGE of slots. The old values go on re_cstk, and an overrun of
             * that stack fails the attempt rather than writing past it. */
            cbase = re_ctop;
            cn = vy - vx + 1;
            if (cbase + cn > 8192) {
                return 0;
            }
            for (i = vx; i < vy + 1; i++) {
                re_cstk[cbase + (i - vx)] = re_slot[i];
                re_slot[i] = -1;
            }
            re_ctop = cbase + cn;
            cr = re_run(pc1, sp);
            re_ctop = cbase;
            if (cr == 0) {
                for (i = vx; i < vy + 1; i++) {
                    re_slot[i] = re_cstk[cbase + (i - vx)];
                }
            }
            return cr;
        }
        return 0;
    }
    return 0;
}

/* ---- the two entry points ------------------------------------------------ */

int rt_regex_search(char *pat, char *subj, char *flags, int *caps, int ncaps)
{
    int i;
    int j;
    int k;
    int start;
    int hasCaps;
    int fc;
    int bad;
    char *msg;
    int slen;
    int pairs;
    int r;
    int room;
    int lastp;
    int slots;

    /* Pre-fill with -1 so no caller can read a stale offset, whatever it does
     * with the answer. */
    hasCaps = (caps != 0);
    if (hasCaps) {
        for (i = 0; i < ncaps; i++) {
            caps[i] = -1;
        }
    }

    /* flags: only "i" is meaningful to bash's =~ (shopt -s nocasematch); a null
     * pointer is "no flags" and must not be dereferenced. */
    re_fold = 0;
    if (flags != 0) {
        j = 0;
        while (1) {
            fc = (int)(unsigned char)flags[j];
            if (fc == 0) {
                break;
            }
            if (fc == 105) {                  /* i */
                re_fold = 1;
            }
            j = j + 1;
        }
    }

    /* ---- compile */
    if (pat == 0) {
        re_pat = empty;
    } else {
        re_pat = pat;
    }
    re_pos = 0;
    re_pc = 0;
    re_ng = 0;
    re_ncls = 0;
    re_nmark = 0;
    re_bad = 0;
    re_depth = 0;
    re_emit(6, 0, 0);                         /* OP_SAVE 0 */
    re_alt();
    if (re_bad != 0) {
        bad = re_bad;
        /* Indexed by re_bad. */
        msg = empty;
        if (bad == 1) { msg = "Unmatched ( or \\("; }
        if (bad == 2) { msg = "Unmatched [, [^, [:, [., or [="; }
        if (bad == 3) { msg = "Invalid content of \\{\\}"; }
        if (bad == 4) { msg = "Regular expression too big"; }
        if (bad == 5) { msg = "Invalid preceding regular expression"; }
        if (bad == 6) { msg = "Trailing backslash"; }
        if (bad == 7) { msg = "Invalid range end"; }
        if (bad == 8) { msg = "empty (sub)expression"; }
        if (bad == 9) { msg = "Invalid character class name"; }
        re_err = msg;
        return -1;
    }
    re_err = empty;
    re_emit(6, 1, 0);                         /* OP_SAVE 1 */
    re_emit(7, 0, 0);                         /* OP_MATCH */
    slen = rt_strlen(subj);
    re_subj = subj;
    re_slen = slen;
    re_steps = 0;
    re_ctop = 0;
    pairs = re_ng + 1;

    /* ---- search: leftmost wins, so the start position is tried in order. */
    start = 0;
    while (start <= slen) {
        for (k = 0; k < 192; k++) {
            re_slot[k] = -1;
        }
        r = re_run(0, start);
        if (r == -2) {
            return -2;
        }
        if (r == 1) {
            /* The TRUE pair count is answered even when it exceeds the caps
             * array, so ${#BASH_REMATCH[@]} stays exact and the entries past it
             * read as empty. */
            room = ncaps / 2;
            lastp = (pairs < room) ? pairs : room;
            slots = hasCaps ? (lastp * 2) : 0;
            for (j = 0; j < slots; j++) {
                caps[j] = re_slot[j];
            }
            return pairs;
        }
        start = start + 1;
    }
    return 0;
}

char *rt_regex_error(void)
{
    char *v;
    v = re_err;
    /* Never a null: the caller concatenates this straight into a message, and
     * reading byte 0 of address 0 is the documented trap our machine forgives
     * and a real process does not. */
    if (v == 0) {
        return empty;
    }
    return v;
}

/* languages/c-to-llvm-ir.abnf requires a main(); the bash module supplies the
 * real one, so this placeholder is stripped by tests/gen-bash-rt-ll.sh. */
int main() { return 0; }
