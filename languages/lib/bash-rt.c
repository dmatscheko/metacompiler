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
/* The empty field list (see rt_fput); every other list lives in the arena's list region. */
char list_nil[2] = {2, 0};

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
 * A BINDING IS NOT JUST A SCALAR: an ARRAY of the same name lives in the array
 * store below, and bash restores it on return too, so each entry also records
 * how many of its rows were copied onto the array save stack. ls_acnt[t] is 0
 * for the usual scalar case and the pop then does nothing beyond clearing
 * whatever the call created. */
int ls_id[512];
char *ls_val[512];
int ls_acnt[512];
int ls_amark[512];   /* the record's marks (1 2 4, see rt_arr_mark), and 8 when it was not sorted */
int ls_top;

/* The array save stack the entries above index into, grown by doubling. */
char **las_k;
char **las_v;
int las_cap;
int las_top;

/* Output capture for $( ) / ` `: one arena buffer per nesting depth, allocated
 * the first time that depth is reached and REUSED after (rt_cap_end hands back a
 * copy), so a loop of command substitutions costs only its results. CAPMAX =
 * 4096 depths, CAPSZ = 8192 bytes to start. */
char *cap_buf[4096];
int cap_len[4096];
int cap_sz[4096];    /* how big cap_buf[i] is; rt_putc doubles it when it fills */
int cap_nul[4096];   /* a NUL byte was dropped at this depth (rt_cap_end warns) */
int cap_depth;

/* Subshell isolation: ( LIST ) and $( ) snapshot the whole variable array and
 * the whole array store, so `a=(p q); ( a[0]=Z )` leaves the caller with "p q".
 * A row's key and value are heap strings never rewritten in place, so copying the
 * pointers is a complete snapshot. Each depth keeps its own malloc'd blocks -
 * ss_blk[d]: nvars values, then a key and a value per row; ss_rec[d]: rows,
 * marks and sortedness of each of the ss_an[d] records - made the first time the
 * depth is reached and reused after, grown when the store has outgrown them. A
 * record made inside the subshell is emptied on the way out. So the nesting is
 * bounded by SSMAX = 4096, recursion through $( ) included.
 *
 * Attributes (-i/-u/-l/-A) are not here: bash-to-llvm-ir.abnf keeps them in
 * COMPILE-TIME maps and unwinds its attribute log around a subshell body. */
char **ss_blk[4096];
long ss_cap[4096];
int *ss_rec[4096];
int ss_rcap[4096];
int ss_an[4096];
int ss_depth;

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
 * arguments, so "$@" can expand to any number of words. rt_argpush grows it. */
char **argv;
int argv_cap;
int argv_top;
int frame_base;
int frame_n;

/* ---- prototypes of every runtime helper (any chunk may call any other) ---- */
char *rt_bump(int n);
char *rt_lbump(int n);
int rt_islist(char *v);
char *rt_lcat(char *a, char *b);
int rt_strlen(char *s);
int rt_charlen(char *s);
int rt_charoff(char *s, int k);
int rt_streq(char *a, char *b);
int rt_strcmp(char *a, char *b);
char *rt_strcat(char *a, char *b);
char *rt_int2str(long n);
long rt_str2int(char *s);
long rt_arith(char *s);
int rt_isassoc(char *nm);
char *rt_arr_key(char *nm, char *sub);
int rt_arith_fail(char *expr, char *msg, char *tok);
int rt_class(char *p, int ch);
int rt_cpw(char *s);
int rt_cpat(char *s);
int rt_gch(char *s);
int rt_uctype(int ctid, int ctch);
int rt_clsid(char *nm);
int re_ctype(int ctid, int ctch);
int rt_eg(char *egp, char *egs, int egstar);
int rt_glob(char *p, char *s);
char *rt_substr(char *s, int off, int n);
char *rt_csubstr(char *s, int off, int n);
int rt_egclose(char *ecq);
int rt_egalt(char *eaq, int eaa);
int rt_matchlen(char *p, char *s, int len, int i, int pl);
int rt_matchend(char *p, char *s);
char *rt_strip(char *s, char *p, int mode);
char *rt_replace(char *s, char *p, char *r, int mode);
char *rt_case(char *s, int mode, char *pat);
char *rt_shquote(char *sq);
char *rt_ansic(char *ac);
int rt_utf8put(char *d, long cp);
char *rt_unraw(char *s);
int rt_haschar(char *s, int ch);
char *rt_read_line(int u);
char *rt_field(char *l, char *ifs, int idx, int rest);
int rt_nifs(char *l, char *ifs);
char *rt_radix(char *s, int base, int upper, int alt);
char *rt_pfq(char *s);
int rt_nfields(char *v);
char *rt_getfield(char *v, int k);
char *rt_ljoin(char *l, char *sep);
char *rt_fput(char *acc, char *f);
char *rt_fadd(char *acc, char *f);
int rt_fpos(char *v);
int rt_fnext(char *v, int p);
char *rt_fat(char *v, int p);
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
int rt_arr_set(char *nm, char *k, char *v);
char *rt_arr_get(char *nm, char *k);
int rt_arr_has(char *nm, char *k);
int rt_arr_del(char *nm, char *k);
int rt_arr_count(char *nm);
int rt_arr_any(char *nm);
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
long rt_arr_nextidx(char *nm);
int rt_arr_iset(char *nm, char *k, char *v);
char *rt_arr_ikey(char *nm, long k);
int rt_arr_clear(char *nm);
int rt_arr_adopt(char *nm, int id);
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
char *rt_cap_end(int cmdsub);
int rt_prints(char *s);
int rt_ss_save(int u);
int rt_ss_restore(int u);
int rt_push_local(int id);
int rt_pop_locals(int d);
int rt_unset_up(int id, int d, int fn);
int rt_fskind(char *p);
char *rt_getvar_byname(char *n);
char *rt_getvar_ind(char *n);
int rt_varslot(char *n);
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
int fflush(void *f);

/* One byte to fd 2. A char[1] on the stack rather than a pointer into the
 * arena, because the arena is exactly what is gone when this runs (the same
 * shape as runtime.c's o_ch). Standard output is putchar's stdio buffer, so it
 * is flushed first (fflush(NULL): every stream, no FILE* needed): bash writes
 * each command's output as it runs, and `cmd 2>&1` interleaves the two in
 * program order. */
int rt_ech(int c)
{
    char b[1];
    b[0] = (char)c;
    fflush(0);
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
    rt_estr(" - a fixed limit of languages/lib/bash-rt.c. ");
    rt_estr("Shorten the run or raise the limit there.\n");
    exit(70);
}

/* ---- the string heap ------------------------------------------------------
 * Two heaps, each a chain of malloc'd chunks: heap 0 holds strings, heap 1
 * FIELD LISTS. A value IS a list exactly when it lies in a heap-1 chunk or is
 * list_nil (rt_islist); the 0x02 a list starts with is therefore not what
 * makes it one, and a plain string may start with that byte too: `w=$'\x02x'`
 * is one field holding three characters.
 *
 * A heap position is VIRTUAL: chunk k covers [h_cvs[k], h_cvs[k] + h_ccap[k])
 * of one growing line, so positions compare across chunks. A request the
 * current chunk cannot hold moves to the next chunk - the one after it if that
 * is big enough, else a fresh one - and leaves the old chunk's tail unused, so
 * every string stays contiguous.
 *
 * RECLAIMING. There is no collector. The emitted program takes a MARK (both
 * positions) before a loop and before each top-level line, and rt_release
 * goes back to it ahead of every further pass, at the loop's exit and at the
 * line's end; a chunk it leaves is reused when the heap grows again. That is
 * sound because a string kept beyond that point is either older than the mark
 * or written into a store that outlives it - a variable, an array, a
 * positional parameter, a capture buffer, a subshell snapshot - and every such
 * write raises the FLOOR (rt_pin): a string release never goes below it. A
 * list is never stored (each store takes a joined string or one field), so
 * its release has no floor. A scalar assignment also moves its value down
 * onto the mark (rt_keep), so the garbage its own expansion made goes back
 * too. A helper whose allocations are all scratch for one call (rt_eg,
 * rt_globat, rt_patany) goes back to its entry position with rt_hset.
 */
long h_pos[2];        /* the next free position                            */
long h_cv[2];         /* the current chunk's first position ...            */
long h_cend[2];       /* ... and the one past its end                       */
long h_adj[2];        /* the address of position p is h_adj + p             */
int h_ci[2];          /* its index                                          */
int h_nch[2];         /* chunks allocated; those past h_ci hold nothing live */
char *h_chunk[128];   /* chunk k of heap h is entry h * 64 + k              */
long h_cvs[128];
long h_ccap[128];
long a_floor;
char *fin_last;
/* The list rt_fput or rt_fadd made last and its length in bytes: the one list
 * rt_fadd may extend in place (a release of heap 1 forgets it). */
char *fa_last;
int fa_len;
void *malloc(unsigned long n);
void free(void *p);
unsigned long strlen(const char *s);
void *memmove(void *d, const void *s, unsigned long n);

/* Make chunk k of heap h current. */
void rt_hcur(int h, int k) {
    int at = h * 64 + k;
    h_ci[h] = k;
    h_adj[h] = (long)h_chunk[at] - h_cvs[at];
    h_cv[h] = h_cvs[at];
    h_cend[h] = h_cvs[at] + h_ccap[at];
}

/* rt_hbump's slow path: n bytes at the start of the next chunk. A spare chunk
 * too small for them is freed with every one after it (none holds anything)
 * and replaced. Chunk k is 1 MiB << k, up to 1 GiB, or the request. */
char *rt_hgrow(int h, int n) {
    int k = h_nch[h] == 0 ? 0 : h_ci[h] + 1;
    int at = h * 64 + k;
    long cv = h_nch[h] == 0 ? 0 : h_cend[h];
    if (n < 0 || k >= 64) {
        rt_die(h == 0 ? "string heap" : "field-list heap", n, k, 64);
    }
    if (k >= h_nch[h] || h_ccap[at] <= (long)n) {
        long sz = (long)1048576 << (k < 10 ? k : 10);
        char *c;
        int j = k;
        while (j < h_nch[h]) {
            free(h_chunk[h * 64 + j]);
            j = j + 1;
        }
        if (sz <= (long)n) sz = (long)n + 1;
        c = (char *)malloc((unsigned long)sz);
        if (c == 0) {
            rt_die(h == 0 ? "string heap" : "field-list heap", n, k, 64);
        }
        h_chunk[at] = c;
        h_cvs[at] = cv;
        h_ccap[at] = sz;
        h_nch[h] = k + 1;
    }
    rt_hcur(h, k);
    h_pos[h] = cv + n;
    return h_chunk[at];
}

/* `>=`, not `>`: before the first chunk the bounds are all 0 and even a
 * request of 0 must make one. */
char *rt_hbump(int h, int n) {
    long p = h_pos[h];
    if (n < 0 || (long)n >= h_cend[h] - p) {
        return rt_hgrow(h, n);
    }
    h_pos[h] = p + n;
    return (char *)(h_adj[h] + p);
}

/* rt_hbump of heap 0, spelled out: it is the runtime's hottest call. */
char *rt_bump(int n) {
    long p = h_pos[0];
    if (n < 0 || (long)n >= h_cend[0] - p) {
        return rt_hgrow(0, n);
    }
    h_pos[0] = p + n;
    return (char *)(h_adj[0] + p);
}

char *rt_lbump(int n) {
    return rt_hbump(1, n);
}

/* The index of the chunk of heap h that holds position p, which is not past
 * h_pos[h]. */
int rt_hchunk(int h, long p) {
    int k = h_ci[h];
    while (k > 0 && h_cvs[h * 64 + k] > p) k = k - 1;
    return k;
}

/* Heap h back to position p, which is not past h_pos[h]: the chunk holding p
 * becomes current again. */
void rt_hset(int h, long p) {
    if (p < h_cv[h]) rt_hcur(h, rt_hchunk(h, p));
    h_pos[h] = p;
}

int rt_islist(char *v) {
    int k;
    if (v[0] != 2) return 0;
    if (v == &list_nil[0]) return 1;
    k = 0;
    while (k < h_nch[1]) {
        if (v >= h_chunk[64 + k] && v < h_chunk[64 + k] + h_ccap[64 + k]) return 1;
        k = k + 1;
    }
    return 0;
}

/* The virtual position of string v, or -1 for one in no live string chunk (a
 * literal, a static). */
long rt_vpos(char *v) {
    int k = 0;
    while (k <= h_ci[0] && k < h_nch[0]) {
        if (v >= h_chunk[k] && v < h_chunk[k] + h_ccap[k]) return h_cvs[k] + (v - h_chunk[k]);
        k = k + 1;
    }
    return -1;
}

/* Where a string release to mark m lands: m, or the floor above it. */
long rt_alo(long m) {
    return m > a_floor ? m : a_floor;
}

/* The release forgets rt_finlist's list and rt_fadd's: each is known by its
 * address, which may now be handed out again. */
int rt_release(long m, long lm) {
    rt_hset(0, rt_alo(m));
    rt_hset(1, lm);
    fin_last = 0;
    fa_last = 0;
    return 0;
}

/* Every string allocated so far may be reachable from a long-lived store. */
int rt_pin(void) {
    a_floor = h_pos[0];
    return 0;
}

/* A scalar assignment's value v, with everything allocated since mark (m, lm)
 * given back. A v below where the release lands (older, a literal, or stored
 * by the expansion itself) stays put, and so does a list. A v above it that is
 * shorter than the garbage below it moves down - to the release point, or to
 * the start of the first later chunk with room, at the latest its own - so
 * the copy frees at least what it costs; a longer v stays, and with it
 * everything allocated so far. strlen and memmove are libc: the IR
 * interpreter runs them as one step each (abnf/llvmmap.go, libcExterns). */
char *rt_keep(long m, long lm, char *v) {
    long lo = rt_alo(m);
    long vp;
    long n;
    int k;
    if (rt_islist(v) != 0) {
        rt_hset(0, lo);
        return v;
    }
    rt_hset(1, lm);
    fin_last = 0;
    fa_last = 0;
    vp = rt_vpos(v);
    if (vp < lo) {
        rt_hset(0, lo);
        return v;
    }
    n = (long)strlen(v);
    if (n >= vp - lo) return v;
    k = rt_hchunk(0, lo);
    while (lo + n + 1 > h_cvs[k] + h_ccap[k]) {
        k = k + 1;
        lo = h_cvs[k];
    }
    v = (char *)memmove(h_chunk[k] + (lo - h_cvs[k]), v, (unsigned long)(n + 1));
    rt_hset(0, lo + n + 1);
    return v;
}

/* The list a followed by the raw bytes b (more fields, already encoded). */
char *rt_lcat(char *a, char *b) {
    int la = rt_strlen(a);
    int lb = rt_strlen(b);
    char *dst = rt_lbump(la + lb + 1);
    int i = 0;
    while (i < la) { dst[i] = a[i]; i = i + 1; }
    i = 0;
    while (i < lb) { dst[la + i] = b[i]; i = i + 1; }
    dst[la + lb] = 0;
    return dst;
}

/* The same, 8-byte aligned, for a block of pointers. A chunk starts aligned,
 * and the block starts n + 7 bytes below h_pos in the current chunk. */
char *rt_bump8(int n) {
    long q;
    rt_bump(n + 7);
    q = h_pos[0] - n - 7;
    return (char *)(h_adj[0] + q + (8 - (q - h_cv[0]) % 8) % 8);
}

/* i32 rt_strlen(i8* s) */
int rt_strlen(char *s) {
    int i = 0;
    while (s[i] != 0) i = i + 1;
    return i;
}

/* rt_charlen: the number of CHARACTERS in s, which is what bash's ${#s} counts -
 * not bytes and not UTF-16 units. A byte that is not part of a well-formed
 * UTF-8 sequence is a character of its own (rt_cpw). */
int rt_charlen(char *s) {
    int i = 0, n = 0;
    while (s[i] != 0) {
        i = i + rt_cpw(s + i);
        n = n + 1;
    }
    return n;
}

/* rt_charoff: the BYTE offset of character number k, clamped to the length of s. */
int rt_charoff(char *s, int k) {
    int i = 0, n = 0;
    while (s[i] != 0) {
        if (n >= k) return i;
        n = n + 1;
        i = i + rt_cpw(s + i);
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

/* rt_sput: s copied to d and terminated; answers where the next part goes. A
 * word of many parts is one rt_bump of their summed lengths and an rt_sput per
 * part (bash-to-llvm-ir.abnf's mkWord), where a chain of rt_strcat would copy
 * the growing prefix once per part. */
char *rt_sput(char *d, char *s) {
    int n = (int)strlen(s);
    memmove(d, s, (unsigned long)(n + 1));
    return d + n;
}

/* ---- append buffers --------------------------------------------------------
 * `s+=x` in a loop would pin every intermediate value of s in the arena, which
 * is quadratic in the final length. Instead slot id's value lives in its own
 * malloc'd buffer (ab_buf[id], ab_len bytes used of ab_cap), and rt_sapp
 * appends in place, doubling the buffer when it is full, as long as nothing
 * but slot id can see the buffer: an array element or a positional parameter
 * that took a pointer into it marks it shared (rt_abheld), and another
 * variable, a `local` save or standard input holding one is found by the scan
 * in rt_abmine. A subshell snapshots the variables, so a buffer is only
 * written at the subshell depth that made it. A buffer that stops being its
 * slot's value is freed when the next one replaces it, if nothing holds it. */
char *ab_buf[1024];
int ab_len[1024];
int ab_cap[1024];
int ab_shared[1024];
int ab_depth[1024];
int ab_ids[1024];     /* the slots that have a buffer, ab_n of them */
int ab_n;

/* Does slot id's buffer hold the address v? */
int rt_abin(int id, char *v) {
    char *b = ab_buf[id];
    return b != 0 && v >= b && v < b + ab_cap[id] ? 1 : 0;
}

/* A store outside the variables is taking v: a buffer holding it is shared. */
int rt_abheld(char *v) {
    int i = 0;
    while (i < ab_n) {
        if (rt_abin(ab_ids[i], v) != 0) ab_shared[ab_ids[i]] = 1;
        i = i + 1;
    }
    return 0;
}

/* 1 when slot id's buffer may be written or freed: nothing but slot id holds it. */
int rt_abmine(int id) {
    int j = 0;
    if (ab_buf[id] == 0 || ab_shared[id] != 0 || ab_depth[id] != ss_depth) return 0;
    if (rt_abin(id, stdin_buf) != 0) return 0;
    while (j < nvars) {
        if (j != id && rt_abin(id, gvars[j]) != 0) return 0;
        j = j + 1;
    }
    j = 0;
    while (j < ls_top) {
        if (rt_abin(id, ls_val[j]) != 0) return 0;
        j = j + 1;
    }
    return 1;
}

/* The value o + x for slot id, whose value is o (`s+=x`). */
char *rt_sapp(int id, char *o, char *x) {
    int lx = (int)strlen(x);
    int lo;
    int cap;
    char *nb;
    int mine;
    if (lx == 0) return rt_strcat(o, x);
    mine = rt_abmine(id);
    if (o == ab_buf[id] && mine != 0 && ab_len[id] + lx < ab_cap[id]) {
        memmove(o + ab_len[id], x, (unsigned long)(lx + 1));
        ab_len[id] = ab_len[id] + lx;
        return o;
    }
    lo = o == ab_buf[id] ? ab_len[id] : (int)strlen(o);
    cap = 2 * (lo + lx) + 16;
    nb = (char *)malloc((unsigned long)cap);
    if (nb == 0) rt_die("append buffer", cap, lo, cap);
    memmove(nb, o, (unsigned long)lo);
    memmove(nb + lo, x, (unsigned long)(lx + 1));
    if (ab_buf[id] == 0) {
        ab_ids[ab_n] = id;
        ab_n = ab_n + 1;
    } else if (mine != 0) {
        free(ab_buf[id]);
    }
    ab_buf[id] = nb;
    ab_len[id] = lo + lx;
    ab_cap[id] = cap;
    ab_shared[id] = 0;
    ab_depth[id] = ss_depth;
    return nb;
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
 * rt_cpw answers 1 for anything that is not a well-formed sequence - a stray
 * byte, an overlong form, a surrogate, a code point past U+10FFFF - so invalid
 * UTF-8 still advances and compares byte-for-byte. Twin of bytesStr in
 * bash-interpreter.abnf. */
int rt_cpw(char *s) {
    int c0 = (int)(unsigned char)s[0];
    int c1;
    int w = 1;
    int i;
    /* Below 0xC2 no sequence starts: ASCII, a stray continuation byte or an
     * overlong lead. The glob matcher asks this per character, so it is first. */
    if (c0 < 194) return 1;
    c1 = (int)(unsigned char)s[1];
    if (c0 >= 194 && c0 <= 223) { w = 2; }
    else if (c0 >= 224 && c0 <= 239) { w = 3; }
    else if (c0 >= 240 && c0 <= 244) { w = 4; }
    if ((c0 == 224 && c1 < 160) || (c0 == 237 && c1 > 159) || (c0 == 240 && c1 < 144) || (c0 == 244 && c1 > 143)) return 1;
    i = 1;
    while (i < w) {
        int cc = (int)(unsigned char)s[i];
        if (cc < 128 || cc > 191) return 1;
        i = i + 1;
    }
    return w;
}

int rt_cpat(char *s) {
    int c0 = (int)(unsigned char)s[0];
    int w;
    int v;
    int i;
    if (c0 < 194) return c0;
    w = rt_cpw(s);
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

/* The character at s as the glob matcher compares it: rt_cpat, except that a
 * byte from 0x80 up that starts no well-formed character is U+DC00 plus the
 * byte, as the interpreter's bytesStr reads it. So a raw 0xE9 is not "é": it
 * equals only a raw 0xE9 in the pattern and is in no POSIX class. */
int rt_gch(char *s) {
    int c = rt_cpat(s);
    if (c >= 128 && rt_cpw(s) == 1) return 56320 + c;
    return c;
}

/* ${v?word} / ${v:?word} on an unset (or, with the colon, empty) value: the word
 * - or bash's own text when it is empty - on the standard error channel, and the
 * abort flag, which ends the (sub)shell with status 1. */
char *rt_pcheck(char *v, int bad, char *name, char *msg)
{
    int saved;
    if (bad != 0) {
        saved = sink_out;
        sink_out = sink_err;
        rt_prints("bash: ");
        rt_prints(name);
        rt_prints(": ");
        rt_prints(msg);
        rt_prints("\n");
        sink_out = saved;
        abort_flag = 1;
    }
    return v;
}

/* ${v:off:len} with a negative len that ends before off: bash's "substring
 * expression < 0", which abandons the command like an arithmetic error. */
int rt_suberr(int bad, int len)
{
    if (bad != 0) rt_arith_fail(rt_int2str((long)len), "substring expression < 0", empty);
    return 0;
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
 * arith_err; the expression answers 0. A command that evaluates arithmetic itself
 * ((( )), let, for (( )), [[ -eq ]]) then fails with status 1; an EXPANSION
 * error abandons the command line it is part of (emitAbortCheck), as bash does.
 * Only the first error is reported, and nothing is stored after it (ar_store,
 * the emitter's arLhsSet): bash abandons the whole expression.
 * Twin of arEval and its helpers in bash-interpreter.abnf. */
int arith_err;

/* What an arithmetic error is reported for, stored by the emitter around (( )), let,
 * [[ ]], for (( )) and ${v:off} ("((: ", "let: ", "[[: ", "v: "). Twin of arPfx in
 * bash-interpreter.abnf. */
char *ar_pfx_s;
char *ar_pfx(void)
{
    return ar_pfx_s != 0 ? ar_pfx_s : &empty[0];
}

int rt_arith_fail(char *expr, char *msg, char *tok)
{
    int saved;
    if (arith_err != 0) return 0;
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
long rt_idiv(long a, long b, int mod, char *expr, char *tok)
{
    if (b == 0) {
        rt_arith_fail(expr[0] != 0 ? rt_strcat(ar_pfx(), expr) : expr, "division by 0", tok);
        return 0;
    }
    if (b == -1) return mod != 0 ? 0 : 0 - a;
    return mod != 0 ? a % b : a / b;
}

/* ** by repeated squaring, wrapping like every product. */
long rt_ipow(long b, long e, char *expr, char *tok)
{
    long r = 1;
    if (e < 0) {
        rt_arith_fail(expr[0] != 0 ? rt_strcat(ar_pfx(), expr) : expr, "exponent less than 0", tok);
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
long ar_errtok(char *msg, char *tok);
char *ar_latok(int p);

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
    return ar_errtok(msg, ar_latok(ar_p));
}

/* bash names the expression without its leading blanks, after the command's prefix. */
long ar_errtok(char *msg, char *tok)
{
    int i = 0;
    if (ar_bad == 0) {
        ar_bad = 1;
        while (ar_s[i] == 32 || ar_s[i] == 9 || ar_s[i] == 10) i = i + 1;
        rt_arith_fail(rt_strcat(ar_pfx(), &ar_s[i]), msg, tok);
    }
    return 0;
}

/* The error token of ** and /= %=: the text after the right operand, or the operand's
 * last token when nothing follows. Twin of arLaTok in bash-interpreter.abnf. */
char *ar_latok(int p)
{
    int q = p;
    int e;
    while (ar_s[q] == 32 || ar_s[q] == 9 || ar_s[q] == 10) q = q + 1;
    if (ar_s[q] != 0) return &ar_s[q];
    q = p;
    while (q > 0 && (ar_s[q - 1] == 32 || ar_s[q - 1] == 9 || ar_s[q - 1] == 10)) q = q - 1;
    e = q;
    while (q > 0 && ar_idch((int)(unsigned char)ar_s[q - 1], 0) != 0) q = q - 1;
    if (q == e && q > 0) q = q - 1;
    return &ar_s[q];
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

/* [ EXPR ] after a name, or 0 when the name has no subscript. An associative
 * array's subscript is the literal text up to the "]". */
char *ar_subscript(char *nm)
{
    long k;
    if (ar_at(0) != 91) return 0;
    ar_p = ar_p + 1;
    if (rt_isassoc(nm) != 0) {
        int s0 = ar_p;
        while (ar_p < ar_len && ar_at(0) != 93) ar_p = ar_p + 1;
        if (ar_at(0) != 93) { ar_err("missing `]'"); return 0; }
        ar_p = ar_p + 1;
        return rt_substr(ar_s, s0, ar_p - s0 - 1);
    }
    k = ar_comma();
    ar_ws();
    if (ar_at(0) != 93) { ar_err("missing `]'"); return 0; }
    ar_p = ar_p + 1;
    return rt_arr_ikey(nm, k);
}

/* A variable's value as an operand: empty is 0, anything else is itself an
 * expression (`x=010` is 8, `x="1+2"` is 3, `r=a` is a's value). */
long ar_value(char *nm, char *key)
{
    char *v;
    if (ar_noeval > 0) return 0;
    if (key != 0) v = rt_sc_get(nm, key);
    else if (rt_arr_any(nm) != 0) v = rt_arr_get(nm, "0");
    else v = rt_getvar_byname(nm);
    if (v[0] == 0) return 0;
    if (ar_depth >= 1024) return ar_err("expression recursion level exceeded");
    return rt_arith(v);
}

long ar_store(char *nm, char *key, long v)
{
    if (ar_noeval > 0 || arith_err != 0) return v;
    if (key != 0) rt_arr_iset(nm, key, rt_int2str(v));
    else if (rt_arr_any(nm) != 0) rt_arr_iset(nm, "0", rt_int2str(v));
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
        return rt_idiv(a, b, op == 37 ? 1 : 0, empty, empty);
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
            return ar_errtok("exponent less than 0", ar_latok(ar_p));
        }
        return rt_ipow(b, e, empty, empty);
    }
    return b;
}

long ar_mul(void)
{
    long v = ar_pow();
    long rv;
    int rs;
    while (ar_bad == 0) {
        int c;
        ar_ws();
        c = ar_at(0);
        if (!((c == 42 && ar_at(1) != 42 && ar_at(1) != 61) || ((c == 47 || c == 37) && ar_at(1) != 61))) break;
        ar_p = ar_p + 1;
        ar_ws();
        rs = ar_p;
        rv = ar_pow();
        if (c != 42 && rv == 0 && ar_noeval == 0) return ar_errtok("division by 0", &ar_s[rs]);
        v = ar_apply(c, v, rv);
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
        if ((op == 47 || op == 37) && v == 0 && ar_noeval == 0) return ar_errtok("division by 0", ar_latok(ar_p));
        if (op != 61) v = ar_apply(op, ar_value(nm, key), v);
        if (ar_bad != 0) return 0;
        return ar_store(nm, key, v);
    }
    return ar_cond();
}

/* An assignment operator where no name stood before it (`8 /= 0`). Twin of arAsgOpHere. */
int ar_asgop_here(void)
{
    int c0 = ar_at(0);
    int c1 = ar_at(1);
    if (c0 == 61) return c1 != 61 ? 1 : 0;
    if (c1 == 61) return (c0 == 43 || c0 == 45 || c0 == 42 || c0 == 47 || c0 == 37 || c0 == 38 || c0 == 94 || c0 == 124) ? 1 : 0;
    return ((c0 == 60 || c0 == 62) && c1 == c0 && ar_at(2) == 61) ? 1 : 0;
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
 * decimal number (blanks around, one sign, no leading 0) needs no parse: it is
 * read in the same pass that recognises it, since every variable an arithmetic
 * expression names comes through here. */
long rt_arith(char *s)
{
    char *saveS;
    int saveLen;
    int saveP;
    long v = 0;
    int i = 0;
    int neg = 0;
    int d0;
    while (s[i] == 32 || s[i] == 9 || s[i] == 10) i = i + 1;
    if (s[i] == 45) neg = 1;
    if (s[i] == 45 || s[i] == 43) i = i + 1;
    d0 = i;
    while (s[i] >= 48 && s[i] <= 57) {
        v = v * 10 + (long)((int)s[i] - 48);
        i = i + 1;
    }
    if (i > d0 && (s[d0] != 48 || i == d0 + 1)) {
        while (s[i] == 32 || s[i] == 9 || s[i] == 10) i = i + 1;
        if (s[i] == 0) return neg != 0 ? 0 - v : v;
    }
    v = 0;
    saveS = ar_s;
    saveLen = ar_len;
    saveP = ar_p;
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
            else if (ar_asgop_here() != 0) ar_err("attempted assignment to non-variable");
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
                if (cid != 0 && (ch < 128 ? re_ctype(cid, ch) : rt_uctype(cid, ch)) != 0) m = 1;
                j = j + d + 4;
                continue;
            }
            /* no ":]" at all: fall through and treat "[" as an ordinary member */
        }
        /* A member - and a range endpoint - is a CHARACTER, not a byte; `ch` is
         * the subject's character as rt_gch reads it (rt_glob decodes it). */
        int mw = rt_cpw(&p[j]);
        int mc = rt_gch(&p[j]);
        int cn = (int)(unsigned char)p[j + mw];
        int cn2 = 0;
        if (cn == 45) cn2 = (int)(unsigned char)p[j + mw + 1];
        if (cn == 45 && cn2 != 0 && cn2 != 93) {
            int hw = rt_cpw(&p[j + mw + 1]);
            int hc = rt_gch(&p[j + mw + 1]);
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
 *
 * The extended globs @( ) ?( ) *( ) +( ) !( ) are patterns only under
 * `shopt -s extglob` (opt_extglob) - and always on the right hand side of
 * [[ == ]], which matches through rt_glob_x (eg_force). Otherwise "@(a|X)" is
 * text: bash 5.3 leaves aXa alone for ${s#@(a|X)}.
 */
int opt_extglob;
int eg_force;
/* eg_inalt counts the alternatives being matched: at the end of the subject
 * a star behaves differently inside one (rt_glob). Twin of egInAlt in
 * bash-interpreter.abnf. */
int eg_inalt;

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
    char *scur;
    char *scur2;
    int pc;
    int isPfx;
    int nextc;
    int sat;
    int eg;
    int wc;
    int we;
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
        if (isPfx && nextc == 40 && (opt_extglob != 0 || eg_force != 0)) return rt_eg(pcur, sp, 0);

        /* "*", as bash's GMATCH does it (lib/glob/sm_loop.c). The run of "*"
         * and "?" after it goes first: a "?" takes one character, and a ?(..)
         * or *(..) group is tried right there and then dropped, so *?(a) and
         * **(a) both act as "*". What follows is tried at every start BEFORE
         * the end of the subject, never at the end itself - except that at the
         * end a !(..) in the top-level pattern matches. Twin of globAt in
         * bash-interpreter.abnf. */
        if (pc == 42) {
            eg = opt_extglob != 0 || eg_force != 0;
            pp = pcur + 1;
            while (1) {
                wc = (int)(unsigned char)pp[0];
                if (wc == 0) return 1;
                we = -1;
                if (eg && (wc == 63 || wc == 42) && (int)(unsigned char)pp[1] == 40) we = rt_egclose(pp + 1);
                if (we >= 0) {
                    if (wc == 63) {
                        if (rt_eg(pp, sp, 0) != 0) return 1;
                    } else {
                        scur = sp;
                        while ((int)(unsigned char)scur[0] != 0) {
                            if (rt_eg(pp, scur, 0) != 0) return 1;
                            scur = scur + rt_cpw(scur);
                        }
                    }
                    pp = pp + we + 2;
                } else if (wc == 63) {
                    if ((int)(unsigned char)sp[0] == 0) return 0;
                    sp = sp + rt_cpw(sp);
                    pp = pp + 1;
                } else if (wc == 42) {
                    pp = pp + 1;
                } else {
                    break;
                }
            }
            if ((int)(unsigned char)sp[0] == 0) {
                return eg_inalt == 0 && eg && wc == 33 && (int)(unsigned char)pp[1] == 40;
            }
            while ((int)(unsigned char)sp[0] != 0) {
                if (rt_glob(pp, sp) != 0) return 1;
                sp = sp + rt_cpw(sp);
            }
            return 0;
        }

        scur2 = sp;
        if ((int)(unsigned char)scur2[0] == 0) return 0;
        /* "?" matches one CHARACTER and a bracket expression holds characters,
         * so the subject is decoded once here and every arm below steps by scw. */
        scw = rt_cpw(scur2);
        sc = rt_gch(scur2);

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
                if (rt_gch(pcur + 1) != sc) return 0;
                pp = pcur + 1 + pcw;
                sp = scur2 + scw;
                continue;
            }
        }
        /* literal */
        pcw = rt_cpw(pcur);
        if (rt_gch(pcur) != sc) return 0;
        pp = pcur + pcw;
        sp = scur2 + scw;
    }
}

/* int rt_egone(char *alt, char *s): 1 when the alternative alt matches all of s. */
int rt_egone(char *alt, char *s)
{
    int r;
    eg_inalt = eg_inalt + 1;
    r = rt_glob(alt, s);
    eg_inalt = eg_inalt - 1;
    return r;
}

/* int rt_eg(char *egp, char *egs, int egstar): egp points at the prefix
 * character of an extglob and egs at the subject; 1 when egp (group AND
 * everything after it) matches all of egs.
 *
 * Each alternative is tried against every PREFIX of the subject, a whole
 * number of characters long, and the rest of the pattern against what is left
 * - the same "split and recurse" shape rt_glob uses for "*", with the call
 * stack as the backtracking stack. `egstar` is the *(...) continuation of a
 * repeat. An occurrence may be empty ([[ "" == +(|a) ]] is true), but only a
 * non-empty one repeats, so the recursion is finite.
 *   @(A|B)  exactly one alternative        !(A|B)  a prefix NO alternative matches
 *   ?(A|B)  zero or one                    *(A|B)  zero or more
 *   +(A|B)  one or more
 * Nothing it allocates outlives it, so every return gives the arena back.
 * Twin of egMatch in bash-interpreter.abnf.
 */
int rt_eg(char *egp, char *egs, int egstar)
{
    int kind;
    char *open;
    int clo;
    char *rest;
    int slen;
    int a;
    int aend;
    int k;
    int any;
    char *pre;
    char *alt;
    long mark = h_pos[0];

    kind = egstar != 0 ? 42 : (int)(unsigned char)egp[0];
    open = egp + 1;
    clo = rt_egclose(open);
    if (clo < 0) return 0;
    rest = egp + (clo + 2);
    slen = rt_strlen(egs);

    if (kind == 33) {
        k = 0;
        while (k <= slen) {
            pre = rt_substr(egs, 0, k);
            any = 0;
            a = 1;
            while (any == 0 && a <= clo) {
                aend = rt_egalt(open, a);
                any = rt_egone(rt_substr(open + a, 0, aend - a), pre);
                if ((int)(unsigned char)open[aend] == 41) break;
                a = aend + 1;
            }
            if (any == 0 && rt_glob(rest, egs + k) != 0) { rt_hset(0, mark); return 1; }
            k = k + (k < slen ? rt_cpw(egs + k) : 1);
        }
        rt_hset(0, mark);
        return 0;
    }

    if ((kind == 63 || kind == 42) && rt_glob(rest, egs) != 0) { rt_hset(0, mark); return 1; }
    a = 1;
    while (a <= clo) {
        aend = rt_egalt(open, a);
        alt = rt_substr(open + a, 0, aend - a);
        k = 0;
        while (k <= slen) {
            if (rt_egone(alt, rt_substr(egs, 0, k)) != 0) {
                if (rt_glob(rest, egs + k) != 0) { rt_hset(0, mark); return 1; }
                if ((kind == 42 || kind == 43) && k > 0 && rt_eg(egp, egs + k, 1) != 0) { rt_hset(0, mark); return 1; }
            }
            k = k + (k < slen ? rt_cpw(egs + k) : 1);
        }
        if ((int)(unsigned char)open[aend] == 41) break;
        a = aend + 1;
    }
    rt_hset(0, mark);
    return 0;
}

/* [[ == ]]: a match with the extended globs on, whatever shopt says. */
int rt_glob_x(char *p, char *s)
{
    int saved = eg_force;
    int r;
    eg_force = 1;
    r = rt_glob(p, s);
    eg_force = saved;
    return r;
}

/* 1 when byte offset k of s is a CHARACTER boundary: a pattern operator never
 * cuts a UTF-8 sequence (bash 5.3: v=hé; ${v%?} is h, not h\xC3). */
int rt_isbound(char *s, int k)
{
    if (k <= 0 || s[k] == 0) return 1;
    return ((int)(unsigned char)s[k] & 192) != 128;
}

/* int rt_patlen(char *p): the number of CHARACTERS every match of p spans, or
 * -1 when "*" or an extglob lets it vary. Every other element matches exactly
 * one character, so a search only has to try the candidates of that length -
 * bash narrows its own search the same way (umatchlen). Twin of patFixedLen in
 * bash-interpreter.abnf. */
int rt_patlen(char *p)
{
    int n = 0;
    int i = 0;
    while (p[i] != 0) {
        int pc = (int)(unsigned char)p[i];
        int nc = (int)(unsigned char)p[i + 1];
        if (pc == 42) return -1;
        if ((pc == 64 || pc == 63 || pc == 43 || pc == 33) && nc == 40 && (opt_extglob != 0 || eg_force != 0)) return -1;
        int cr = pc == 91 ? rt_class(&p[i], 0) : -1;
        if (cr >= 0) i = i + cr / 2;
        else if (pc == 92 && nc != 0) i = i + 1 + rt_cpw(&p[i + 1]);
        else i = i + rt_cpw(&p[i]);
        n = n + 1;
    }
    return n;
}

/* char *rt_slice(char *s, int i, int n): the n bytes of s at offset i, which
 * must lie inside s. rt_substr measures all of s first, which a loop over the
 * positions of a long subject cannot afford. */
char *rt_slice(char *s, int i, int n)
{
    char *d = rt_bump(n + 1);
    int k = 0;
    while (k < n) { d[k] = s[i + k]; k = k + 1; }
    d[n] = 0;
    return d;
}

/* int rt_globat(char *p, char *s, int i, int n): rt_glob of p against the n
 * bytes of s at offset i. The candidate (and whatever rt_class takes) goes back
 * to the arena afterwards: a search tries many candidates and keeps none. */
int rt_globat(char *p, char *s, int i, int n)
{
    long mark = h_pos[0];
    int r = rt_glob(p, rt_slice(s, i, n));
    rt_hset(0, mark);
    return r;
}

/* int rt_matchlen(char *p, char *s, int len, int i, int pl): end index of the
 * LONGEST match of p starting at i, or -1; len is rt_strlen(s) and pl is
 * rt_patlen(p). rt_glob steps the subject by rt_cpw, so a match of pl
 * characters can only end pl such steps after i: that is the one candidate. */
int rt_matchlen(char *p, char *s, int len, int i, int pl)
{
    int j = len;
    int lo = i;
    int n = 0;
    if (pl >= 0) {
        j = i;
        while (n < pl && j < len) {
            j = j + rt_cpw(&s[j]);
            n = n + 1;
        }
        if (n < pl) return -1;
        lo = j;
    }
    while (j >= lo) {
        if (rt_isbound(s, j) != 0 && rt_globat(p, s, i, j - i) != 0) return j;
        j = j - 1;
    }
    return -1;
}

/* int rt_matchend(char *p, char *s): start index of the LONGEST match of p
 * that reaches the end of s, or -1. */
int rt_matchend(char *p, char *s)
{
    int len = rt_strlen(s);
    int pl = rt_patlen(p);
    int i = 0;
    int hi = len;
    if (pl >= 0) {
        i = len - 4 * pl > 0 ? len - 4 * pl : 0;
        hi = len - pl;
    }
    while (i <= hi) {
        if (rt_isbound(s, i) != 0 && rt_globat(p, s, i, len - i) != 0) return i;
        i = i + 1;
    }
    return -1;
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
    int pl = rt_patlen(p);
    int t = 0;
    while (t <= len) {
        int k = asc ? t : len - t;
        int clen = isPre ? k : len - k;
        int fits = pl < 0 || (clen >= pl && clen <= 4 * pl);
        if (fits && rt_isbound(s, k) != 0 && rt_globat(p, s, isPre ? 0 : k, clen) != 0) {
            return isPre ? rt_substr(s, k, -1) : rt_substr(s, 0, k);
        }
        t = t + 1;
    }
    return s;
}

/* The REPLACEMENT of ${v/p/r} with the matched text m substituted for every
 * unquoted "&" (bash 5.2+'s patsub_replacement, on by default): k=hello;
 * ${k//l/&&} is hellllo. A quoted part of the word reaches here with its "&"
 * and "\" escaped by a backslash (rt_repescape), an expansion's value raw; either
 * way "\&" and "\\" are a literal & and \, and a backslash before anything else
 * is itself. Twin of repSubst in bash-interpreter.abnf. */
char *rt_repsub(char *r, char *m)
{
    int i = 0;
    int special = 0;
    char *out;
    while (r[i] != 0) {
        if (r[i] == 38 || r[i] == 92) special = 1;
        i = i + 1;
    }
    if (special == 0) return r;
    out = empty;
    i = 0;
    while (r[i] != 0) {
        int c = (int)(unsigned char)r[i];
        int n = (int)(unsigned char)r[i + 1];
        if (c == 92 && (n == 38 || n == 92)) {
            out = rt_strcat(out, rt_substr(r, i + 1, 1));
            i = i + 2;
        } else if (c == 38) {
            out = rt_strcat(out, m);
            i = i + 1;
        } else {
            out = rt_strcat(out, rt_substr(r, i, 1));
            i = i + 1;
        }
    }
    return out;
}

/* A quoted part of a ${v/p/r} REPLACEMENT: its "&" and "\" are literal. */
char *rt_repescape(char *s) {
    int len = rt_strlen(s);
    char *dst = rt_bump(len * 2 + 1);
    int i = 0, o = 0;
    while (i < len) {
        int ch = (int)(unsigned char)s[i];
        if (ch == 38 || ch == 92) {
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

/* rt_replace collects its result in rp_buf, which doubles when full: appending
 * to an arena string with rt_strcat copies all of it on every step, quadratic
 * in time and arena over a long subject. */
char *rp_buf;
int rp_len;
int rp_cap;

void rp_add(char *b, int n) {
    int i = 0;
    if (rp_len + n + 1 > rp_cap) {
        char *nb;
        rp_cap = (rp_len + n + 1) * 2;
        nb = rt_bump(rp_cap);
        while (i < rp_len) { nb[i] = rp_buf[i]; i = i + 1; }
        rp_buf = nb;
    }
    i = 0;
    while (i < n) { rp_buf[rp_len + i] = b[i]; i = i + 1; }
    rp_len = rp_len + n;
    rp_buf[rp_len] = 0;
}

/* int rt_patany(char *p, char *s, int mode, int wide): bash's match_pattern
 * (subst.c) first rules out a subject that p cannot match anywhere, with one
 * match of p widened by stars: a "*" in front unless p starts with a plain "*",
 * and one behind unless it ends with an unescaped "*". When subject and pattern
 * are both ASCII (match_upattern, wide == 0) the anchored side of mode 2 (#) and
 * 3 (%) gets no star. A star never tries the empty tail, so v=xy;
 * ${v/%@(|a)/Z} is xy. Every search of rt_replace, the ones behind a
 * replacement too, passes this first. Twin of patAnywhere in
 * bash-interpreter.abnf. */
int rt_patany(char *p, char *s, int mode, int wide)
{
    int n = rt_strlen(p);
    long mark = h_pos[0];
    int lead = (int)(unsigned char)p[0] == 42 && !(opt_extglob != 0 && (int)(unsigned char)p[1] == 40);
    int bs = 0;
    int r;
    char *np = p;
    if (lead && (int)(unsigned char)p[n - 1] == 42) return rt_glob(p, s);
    if (!lead && (wide != 0 || mode != 2)) np = rt_strcat("*", p);
    if (wide != 0 || mode != 3) {
        while (bs < n - 1 && (int)(unsigned char)p[n - 2 - bs] == 92) bs = bs + 1;
        if ((int)(unsigned char)p[n - 1] != 42 || bs % 2 == 1) np = rt_strcat(np, "*");
    }
    r = rt_glob(np, s);
    rt_hset(0, mark);
    return r;
}

/* The index of the last byte of s from 0x80 up, or -1: rt_patany's `wide`
 * for the subject from offset i is rt_lasthigh(s) >= i. */
int rt_lasthigh(char *s)
{
    int i = 0;
    int h = -1;
    while (s[i] != 0) {
        if ((int)(unsigned char)s[i] >= 128) h = i;
        i = i + 1;
    }
    return h;
}

/* rt_replace: ${v/p/r} and friends; mode 0 = first, 1 = all, 2 = anchored at the
 * start (#), 3 = anchored at the end (%). An UNSET value is left alone. A null
 * pattern replaces nothing unless anchored: ${v/#/x} prefixes x. Otherwise this
 * is bash's pat_subst: the first start with a match wins, with its longest match,
 * an EMPTY one included, and after an empty match the character there is copied
 * and the search goes on behind it (v=abac; ${v//*(a)/Z} is ZZbZZc). bash tries
 * a pattern at the END of the subject only when it starts with "*", which for
 * modes 0, 1 and 2 decides an empty subject: e=""; ${e//?(a)/Z} is "" while a
 * lone star pattern matches it. Twin of replacePat in bash-interpreter.abnf. */
char *rt_replace(char *s, char *p, char *r, int mode) {
    int len = rt_strlen(s);
    int pl = rt_patlen(p);
    int from = 0;
    int pw;
    int sw;
    if (s == &unset_marker[0]) return s;
    if (p[0] == 0 && mode < 2) return s;
    pw = rt_lasthigh(p) >= 0;
    sw = rt_lasthigh(s);
    if (p[0] != 0 && rt_patany(p, s, mode, pw || sw >= 0) == 0) return s;
    if (len == 0 && p[0] != 0 && p[0] != 42 && mode < 3) return s;
    if (mode >= 2) {
        if (mode == 2) {
            int jj = rt_matchlen(p, s, len, 0, pl);
            if (jj < 0) return s;
            return rt_strcat(rt_repsub(r, rt_substr(s, 0, jj)), rt_substr(s, jj, -1));
        }
        int ii = rt_matchend(p, s);
        if (ii < 0) return s;
        return rt_strcat(rt_substr(s, 0, ii), rt_repsub(r, rt_substr(s, ii, -1)));
    }
    if (len == 0) {
        if (rt_glob(p, s) != 0) return rt_repsub(r, s);
        return s;
    }
    rp_buf = empty;
    rp_len = 0;
    rp_cap = 0;
    while (from < len) {
        int i = from;
        int j = -1;
        char *rep;
        if (from > 0 && rt_patany(p, &s[from], mode, pw || sw >= from) == 0) break;
        while (j < 0 && i < len) {
            j = rt_matchlen(p, s, len, i, pl);
            if (j < 0) i = i + rt_cpw(&s[i]);
        }
        if (j < 0) break;
        rep = rt_repsub(r, rt_slice(s, i, j - i));
        rp_add(&s[from], i - from);
        rp_add(rep, rt_strlen(rep));
        from = j;
        if (mode == 0) break;
        if (i == j) {
            /* an empty match: copy the character it sits on, then search behind it */
            int w = rt_cpw(&s[i]);
            rp_add(&s[i], w);
            from = i + w;
        }
    }
    rp_add(&s[from], len - from);
    return rp_len == 0 ? empty : rp_buf;
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

/* The class of a character from U+0080 up in the macOS en_US.UTF-8 locale, which
 * is what bash 5.3 asks iswctype for a POSIX bracket class in a glob and
 * iswprint for the $'...' quoting below:
 *     0 unassigned or a surrogate  1 upper  2 lower  3 other alpha
 *     4 digit (and xdigit)  5 punct  6 space  7 cntrl  8 other printable
 * where 1-6 and 8 are print, 1-5 and 8 graph, and nothing is blank. A raw byte
 * is U+DC80..U+DCFF (rt_gch), so it is in no class. uclass_runs0/1 hold the runs
 * from U+0080 up, each a class digit and then its length in base 26 (leading
 * digits a-z, the last A-Z), split in two because the C compiler cannot index
 * one string literal past byte 4095. Twin of uclass in
 * languages/bash-interpreter.abnf, which holds the same string and says where
 * it came from. */
char *uclass_runs0 =
    "7bG6B5B8F5B8C3B5B8B7B8H2B5C8C3B5B8D5B1X8B1H2Y8B2I1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "C1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2C1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1C2B1B2B1B2D1C2B1B2B1C2B1D2C1E2B1C2"
    "B1D2D1C2B1C2B1B2B1B2B1C2B1B2C1B2B1C2B1D2B1B2B1C2C3B1B2D3E1B3B2B1B3B2B1B3B2B1B2B1"
    "B2B1B2B1B2B1B2B1B2B1B2B1B2C1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2C1B3B2B1B2B1D2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2H1C2B1C2C1B2B1E2B1B2B1B2B1B2B1B2cR3B2bB3S8E3M8O3"
    "F8H3B8B3B8eZ1B2B1B2B3B8B1B2B0C3B2D5B1B0E8C1B5B1D0B1B0B1C2B1R0B1J2bJ1B2C1D2D1B2B1"
    "B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2F1B2B8B1B2B1C2C1bZ2bW1B2B1B2B1B2B1B2B1"
    "B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B8I1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1C2B1B2B1B2B1B2B1B2B1B2B1B2C1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B0B1bM0C3B5G2bP5C0C8"
    "D0B8bT5B8B5B8C5B8C5B8B0I3bB0E3E5C0L7G8D5C8B5C8N5B7B5D3bR8V4K5E3C8B3dV5B3B8H7B8H3"
    "C8H3C4K3D8C3B5O0B7B3B8B3bE8bB0C3dL8L3B0O4K3bH8J3C8B5D3B0C8D3W8E3B8J3B8D3B8F0C5P0"
    "B3Z8D0C5B0B3L0F3Y8B3G0B7C0G8I3bQ8Y7B8bH3cC8D3B8S3B8H3K8C5C4K5B3Q8D0B3I0C3C0C3W0B"
    "3H0B3B0D3E0C8B3B8H0C8C0C8D3B0I8B0E3C0B3D8C0C4K3C8K3B5B8B0C8D0B3G0E3C0C3W0B3H0B3C"
    "0B3C0B3C0C8B0B8F0E8C0C8D0D8B0H3E0B3B0H4K8C3D8B5B0K8D0B3J0B3D0B3W0B3H0B3C0B3F0C8B"
    "3B8I0B8D0B8D0C3B0P3C8C0C4K5B8B0H3B8G0B8D0B3I0C3C0C3W0B3H0B3C0B3F0C8B3B8H0C8C0C8D"
    "0H8D0E3C0B3D8C0C4K8B3B8G0K8B3B0B3G0D3D0B3E0D3C0B3B0B3C0D3C0D3D0D3M0E8F0D8D0B8E0C"
    "3B0G8B0O4K8L0F8F3I0B3D0B3X0B3Q0C8B3B8H0B8D0B8E0H8C0B3D0C3B0C3C8C0C4K0H5B8I3B8D5B"
    "3I0B3D0B3X0B3K0B3F0C8B3B8H0B8D0B8E0H8C0G3C0B3C8C0C4K0B3C8B0M8E3J0B3D0B3bP8C3B8H0"
    "B8D0B8E3B8B0E3D8I3D8C0C4K8K3G0B8D0B3S0D3Y0B3J0B3B0C3H0D8B0E8G0B8B0B8I0G4K0C8C5B0"
    "M3bW8B3C8H0E8B3H8I5B4K5C0bL3C0B3B0B3F0B3Y0B3B0B3K8B3C8J3B0C3F0B3B0B8H0B4K0C3E0bG"
    "3B8D5P8B5B8L4K8Q5E8C3I0B3bK0E8U5B8C3F8L0B8bK0B8P0B8C5F8E5C0bL3bR8U3B4K5G3G8E3E8D"
    "3B8D3C8H3D8E3N8M3B8B4K8G1bM0B1B0F1B0C2bR5B3B2D3mR0B3E0C3H0B3B0B3E0C3bP0B3E0C3bH0"
    "B3E0C3H0B3B0B3E0C3P0B3cF0B3E0C3cP0C8D5J8U0D3Q8K0G1dI0C2G0C5B3xW8B5B3R6B3bA5C0D3c"
    "X5D8D3I0H3S8E0J3T8D5C0J3S8C0M3N0B3D0B8C0M3cA8bG5D3B5D8B3B8B0C4K0G8K0G5L8D7B8B4K0"
    "G3dL0H3F8C3bI8B3B0F3cS0K3bF0B8M0E8M0E8B0D5C4K3bE0C3F0L3bS0E3bA0G4K8B0D8bI3X8F0C5"
    "C3cB8K0B8bD0C8B4K0G4K0G5H3B5G0C8bF0bX8F3bV8R3I0D4K5H8bC5C0B8D3bE8N3C4K3bS8O0I5E3"
    "bK8U0D5F4K0D3D4K3bK5C2J0H1bR0C1D5I0I8D5B8V3E8B3G8B3C8D3B0F2bS3cL2N3B2bI3bL8cM1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2J1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2J1I2G0C1G0C2I1I2I1I2G0C1G0C2I0B1B0B1B0B1B0B1B2I1I2O0C2I3I2I3I2I3I2F0"
    "B2C1E3B8B2B8D2D0B2C1E3B8D2E0C2C1E0B8D2I1F8D0C2D0B2C1E3B8C0B6L7F5Y6C7F6B5U8B5N8B5"
    "M6B7F0B7K8B3B0C8J5C3B8N5C0B3N0D8bH0P8bH0P8C1B8E1B8C2B1D2C1D2B8B1B8D1F8G1B8B1B8B1"
    "B8B1E8B2B1E2B3E2B8C2C1C8F1B2E8E2B8cA1B2B8H0E8oM5E8bD5C8jS0Z8L0V8bdW5O8dB5C8bF5K8"
    "pN5W8cL5E8bG5C8oK0C8bG0B8eB1bW2bW1B2B1D2C1B2B1B2B1B2B1E2B1B2C1B2G3C1D2B1B2B1B2B1"
    "B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1";
char *uclass_runs1 =
    "B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1"
    "B2B1B2B1B2B1B2B1B2B1B2B1B2C8G1B2B1B2B8D1B2B0F5E8B5C2bM0B2B0F2B0C3cE0H3B5B0O8B3X0"
    "J3H0B3H0B3H0B3H0B3H0B3H0B3H0B3H0B8bG5bV3B5bG8C5M0bI8bA0B8dL0M8iG0bA8M0E6B5D8B3C8"
    "B5K8C5M8Q5B3F8F3C5B8C0B3dI0C8E3D5B3dM5B3E0F3bR0B3dQ0B8Q3bG8bK0M3Q8bF0B8sM3jtO8cM"
    "3bguF0D8cD0J3bU5C3kJ5D3Q4K3C0U1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B"
    "2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B3B8E5B8K5B3B1B2B1B2B1B2B1B2B1B2B1B2B1B"
    "2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B3C8C3cS8M5G0I8X3J8C1B2B1B2B1B2B1B2B1B2B1B2B1B2D1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2"
    "B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B3B2I1B2B1B2B1C2B1B2B1B2B1B2B1B2B3B8C1B2"
    "B1B2B3B1B2B1B2D1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1B2B1F2B1F2B1B2B1B2B1B2B1B2B1"
    "B2B1B2B1B2B1E2B1B2B0F1B2B0B2B0B2B1B2B1B2B0Y3D1B2B3D2B3H8B3D8B3E8B3X8K0D8K0G3cA5E"
    "0I8C3bY8S0I5C4K0G8S3G5D3B5B3C8B4K3bC8I5C3X8N0L5B3bD0D8E3bV8O5N0B3B4K0E5C3F8B3K4K"
    "3F0B3bP8O0J3D8B3I8C0C4K0C5E3X8D3B8D3bY8B3B8D3C8C3F8C3B8B3B0Y3D5C3L8F5C3D8C0K3G0C"
    "3G0C3G0J3H0B3H0B2bR8B3E2J3B8C0E2dC3bJ8I5B8C0C4K0G3qnS0M3X0E3bX0daY8jmE3oC0C3eC0b"
    "M2H0M2F0F3B8B3K8B3N0B3F0B3B0B3C0B3C0B3eE8R0Q3nZ5C8Q3cM0C3cC0H8B0bG3M8U5K0G8Q5bJ0"
    "B5O8B5B8D0B5B8B5C0E3F0B3fF0E5D8B5G8B5E4K5C8D5C1bA5D8B5B8B2bA5B8B5B8B5H3dL0D3G0C3"
    "G0C3G0C3D0D8H0B8H0K7D8C0C3M0B3bA0B3T0B3C0B3P0C3O0bI3eT0F5D0E8bT0D8dK0B8N0D8B0bV8"
    "bU0fA3bD0D3bX0P8bC0E3bG8E0J3U8B3I8B0F3bM8F0F3bE0B5B3bK0E3I5B8F0bQ1bO2bO3dA0C4K0G"
    "1bK0E2bK0E3bO0I3cA0L5B1L0B1P0B1H0B1C0B2L0B2P0B2H0B2C0cP3lZ0J3W0K3I0Y3G0B3bQ0B3J0"
    "cR3G0C3B0B3bS0B3C0D3B0C3X0B5B8I3X8J3bF0I8J0bW3T0B3C0F8F3W8G0D5B3bA0F5B0cM3cE0E8C"
    "3C8Q0C8bU3B8D0B8C0F8E3E0B3D0B3bD0C8D0E8K0H5J0H3bD8C5B3bD8D0bG3I8B3bC8C0E8F5H0J3c"
    "C0D5H3W0C8I3T0F8I3S0H5E0M8H0dC3cV0cD1bZ0N2bZ0H8G3bK8E0I4K0lI8bF0B3bQ0B8C5B0C3C0c"
    "X8D3bD8K3B0I3W8P5F0W3S8E5E0bM3V8H0U3X0J8D3cB8P5H0E8U4K8B3C8C3B0J8E3bT8L5C7B5E8B0"
    "K7B0C3Z0H4K0G8D3bK8O0B4K5E3B8C3B0I3bJ8B5C3B0J8D3bW8O3E5E8E5B8C4K3B5B3B5D0B8U0L3S"
    "0B3Z8M5G8B3C8B0cK3H0B3B0B3E0B3P0B3K5B0G3bV8M0F4K0G8E0B3I0C3C0C3W0B3H0B3C0B3F0B8C"
    "3B8H0C8C0C8D0C3B0G8B0F3F8C0C8H0D8F0fJ3cB8S3E5F4K5C0B5B8B3D0bE3bW8U3C5B3B0I4K0gK3"
    "bV8H0C8J5X3E8C0bI3bW8R5D3B0L4K0G5N0T3bR8N3B5B0G4K0cC3bB0C8P0E4K8C5D8B3H0hD3bS8P5"
    "B0dW1bG2bG4K8J0M3I0C3B0C3I0B3C0B3Y8G0B8C0C8E3B8B3B8C5D0J4K0cS3I0C3bN8H0C8H3B5B3B"
    "8B0bB3B8K3bO8H3B8E5I8B0I3B8L3bU8Q5D3B5F0N3cV0H5K0jM3J0B3bL8I0B8I3B5F0K4K8T0D5C3b"
    "E0C8W0B8O0cV3H0B3C0B3bM8G0D8B0B8C0B8H3B8B0I4K0G3G0B3C0B3bG8F0B8C0B8F3B0H4K0lY3T8"
    "E5C0H8C3B8B3N0B3bI8H0D8F5N4K0dI3B0P8bY0N5B3bjM0dY8eH0B5F0L3hO0dxK3dT5C0N3bpG7Q8B"
    "3G8P0fyG3wL0muB3vX0H3bF0B4K0E5C3dB0B4K0G3bE0C8F5B0K3bW8H5F8E3E5B8B0K4K0B8H0B3V0F"
    "3T0baM1bG2bG8X5E0dX3cX0E8B3B8cD0H8E3N0cM3C5B3B8B0L8C0O3jcA0I3bvQ0bQ3J0nfR3E0B3H0"
    "B3C0B3lF0P3B0bD3D0C3B0O3E0I3pG0dkU3eD0F3N0D3J0H3K0C8D5B7E0gyU8bU0C8X0J8eM0cI8jM0"
    "K8bN0C8cW7I8eI0V8cS0eS8U0M8U0M8dJ0J8Z0fF1bA2bA1bA2H0B2S1bA2bA1B0B1C0C1B0C1C0C1E0"
    "B1I2E0B2B0B2H0B2L1bA2bA1C0B1E0C1I0B1H0B2bA1C0B1E0B1F0B1B0D1H0B2bA1bA2bA1bA2bA1bA"
    "2bA1bA2bA1bA2bA1bA2bC0C1Z8B2Z8B2G1Z8B2Z8B2G1Z8B2Z8B2G1Z8B2Z8B2G1Z8B2Z8B2G1B2B0C4"
    "bY8yX5F0P8F0B8P0bqM2K3B2U0G2G0iF8H0B8R0C8H0B8C0B8F0F3cK0bH8B0eI3bT0D8H3H0C4K0E3B"
    "8B0mI3bE8B0R3bS8E4K0F8B0rW3bC8E4K0bcO3H0B3E0B3C0B3P0B3hP0C8Q0bP1bI2bI8H3B0E4K0E5"
    "C0beF8cQ0cY8cJ0hM3E0B3bB0B3C0B3B0C3B0B3K0B3E0B3B0B3B0G3B0E3B0B3B0B3B0B3D0B3C0B3B"
    "0C3B0B3B0B3B0B3B0B3B0B3C0B3B0C3E0B3H0B3E0B3E0B3B0B3K0B3R0F3D0B3F0B3R0cA8C0kK8bS0"
    "E8dW0M8P0C8P0B8P0B8bL0K8gS0cE8bD0N8bS0E8J0H8C0O8G0fY8blW0E8R0D8N0D8eP0E8dR0G8M0E"
    "8B0P8M0E8cE0I8K0G8bO0I8bE0C8C0dA8nC0M8O0C8N0D8J0H8bU0B8H0I8O0E8J0H8J0H8fR0B8cD0b"
    "L4K0bnQ3clfC0bG3gdU0G3iO0C3inQ0O3lbL0epJ3uW0cfY3hhZ0F3gfG0bomxL7B0bE7dS0eY8jG0ds"
    "fO8dsyO0C8dsyO0C";
int uclass_cuts[3370];   /* the first code point past each run */
int uclass_cls[3370];
int uclass_n = 0;
int rt_uclass(int cp) {
    int lo = 0;
    int hi;
    if (uclass_n == 0) {
        int p = 128;
        int n = 0;
        int k = 0;
        int r = 0;
        while (r < 2) {
            char *row = r == 0 ? uclass_runs0 : uclass_runs1;
            int i = 0;
            while (row[i] != 0) {
                int d = (int)(unsigned char)row[i];
                if (d < 65) k = d - 48;
                else if (d >= 97) n = n * 26 + d - 97;
                else {
                    p = p + n * 26 + d - 65;
                    uclass_cuts[uclass_n] = p;
                    uclass_cls[uclass_n] = k;
                    uclass_n = uclass_n + 1;
                    n = 0;
                }
                i = i + 1;
            }
            r = r + 1;
        }
    }
    /* the first run that ends past cp */
    hi = uclass_n;
    while (lo < hi) {
        int mid = (lo + hi) / 2;
        if (uclass_cuts[mid] <= cp) lo = mid + 1; else hi = mid;
    }
    return lo < uclass_n ? uclass_cls[lo] : 0;
}
int rt_wprint(int cp) {
    int k = rt_uclass(cp);
    return (k != 0 && k != 7) ? 1 : 0;
}

/* re_ctype's answer for a character from U+0080 up: an alphabetic character is
 * alnum and word, a digit of any script is also xdigit, and none is blank or
 * ascii. Twin of uCtype in languages/bash-interpreter.abnf. */
int rt_uctype(int ctid, int ctch) {
    int k = rt_uclass(ctch);
    if (ctid == 1) return k >= 1 && k <= 3;
    if (ctid == 2 || ctid == 12) return k == 4;
    if (ctid == 3 || ctid == 13) return k >= 1 && k <= 4;
    if (ctid == 4) return k == 1;
    if (ctid == 5) return k == 2;
    if (ctid == 6) return k == 6;
    if (ctid == 8) return k == 5;
    if (ctid == 9) return k != 0 && k != 7;
    if (ctid == 10) return k != 0 && k != 7 && k != 6;
    if (ctid == 11) return k == 7;
    return 0;
}

/* Does quoting v need the $'...' form: a control character, a byte >= 0x80
 * that does not start a well-formed UTF-8 character (rt_cpw answers 1), or a
 * character rt_wprint rejects. Twin of needsAnsi in bash-interpreter.abnf. */
int rt_needansi(char *v) {
    int i = 0;
    while (v[i] != 0) {
        int c = (int)(unsigned char)v[i];
        int w;
        if (c < 32 || c == 127) return 1;
        w = 1;
        if (c >= 128) {
            w = rt_cpw(v + i);
            if (w == 1 || rt_wprint(rt_cpat(v + i)) == 0) return 1;
        }
        i = i + w;
    }
    return 0;
}

/* rt_shquote: the ${v@Q} / ${v@K} / ${v@k} form - sq quoted so the shell would
 * read it back unchanged. Single quotes normally, with an embedded ' spelled
 * '\'', and the $'...' form (rt_ansiqs) when rt_needansi. */
char *rt_shquote(char *sq) {
    int len = rt_strlen(sq);
    char *dst;
    int o = 0;
    int i = 0;
    if (rt_needansi(sq) != 0) return rt_ansiqs(sq);
    /* 4 bytes per input byte is the worst case ('\''), plus the quotes and NUL. */
    dst = rt_bump(len * 4 + 3);
    dst[o] = (char)39; o = o + 1;
    while (i < len) {
        int c0 = (int)(unsigned char)sq[i];
        if (c0 == 39) {
            dst[o] = (char)39; o = o + 1;
            dst[o] = (char)92; o = o + 1;
            dst[o] = (char)39; o = o + 1;
            dst[o] = (char)39; o = o + 1;
        } else {
            dst[o] = (char)c0; o = o + 1;
        }
        i = i + 1;
    }
    dst[o] = (char)39; o = o + 1;
    dst[o] = 0;
    return dst;
}

/* A $'...' constant holding raw bytes: the emitter interns each raw byte B as the
 * WTF-8 of U+DC00+B (ED B2 80 .. ED B3 BF, never valid UTF-8), and this turns every
 * such triple back into B. See ansiWord in bash-to-llvm-ir.abnf. */
char *rt_unraw(char *s) {
    int len = rt_strlen(s);
    char *dst = rt_bump(len + 1);
    int i = 0;
    int o = 0;
    while (i < len) {
        int c0 = (int)(unsigned char)s[i];
        int c1 = (int)(unsigned char)s[i + 1];
        if (c0 == 237 && (c1 == 178 || c1 == 179) && i + 2 < len) {
            dst[o] = (char)(128 + (c1 - 178) * 64 + ((int)(unsigned char)s[i + 2] - 128));
            i = i + 3;
        } else {
            dst[o] = (char)c0;
            i = i + 1;
        }
        o = o + 1;
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
 * (one or two hex digits) are decoded, so ${v@E} inverts ${v@Q}, and \uHHHH /
 * \UHHHHHHHH are the code point's UTF-8. A \x, \u or \U with no hex digit after
 * it stays literal, as bash leaves it. */
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
        } else if ((c0 == 92) && (i + 1 < len) && (c1 == 117 || c1 == 85) &&
                   (i + 2 < len) && (rt_hexval((int)(unsigned char)ac[i + 2]) >= 0)) {
            long v = 0;
            int n = 0;
            int maxd = (c1 == 117) ? 4 : 8;
            i = i + 2;
            while (n < maxd && i < len) {
                int d = rt_hexval((int)(unsigned char)ac[i]);
                if (d < 0) break;
                v = v * 16 + d;
                i = i + 1;
                n = n + 1;
            }
            o = o + rt_utf8put(dst + o, v);
        } else if ((c0 == 92) && (c1 == 99) && (i + 2 < len)) {
            /* \cX: the control character of X's first byte ("?" is DEL, else
             * toupper(X) & 0x1f); a multibyte X's other bytes follow as they are.
             * Twin of ansiC in bash-interpreter.abnf. */
            int x = (int)(unsigned char)ac[i + 2];
            dst[o] = (char)((x == 63) ? 127 : (x & 31));
            o = o + 1;
            i = i + 3;
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
            /* any other escape stays as written, backslash included */
            if (mapped == c1 && c1 != 92 && c1 != 39 && c1 != 34 && c1 != 63) {
                dst[o] = (char)92;
                o = o + 1;
            }
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

/* rt_pfq: printf %q. This is bash's BACKSLASH form, NOT ${v@Q}'s single-quoted
 * one - `printf %q "a b"` is `a\ b` while `${v@Q}` is `'a b'`. An empty operand
 * is '' and a value that rt_needansi falls back to rt_shquote, which already
 * emits the $'...' spelling. Mirrors printfQuote in
 * languages/bash-interpreter.abnf. */
char *rt_pfq(char *s) {
    int len = rt_strlen(s);
    int i;
    char *dst;
    int o;
    if (len == 0) {
        dst = rt_bump(3);
        dst[0] = 39;
        dst[1] = 39;
        dst[2] = 0;
        return dst;
    }
    if (rt_needansi(s) != 0) return rt_shquote(s);
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

/* ---- printf -----------------------------------------------------------------
 * bash's printf FORMAT [ARG...], read at run time: the format may come out of an
 * expansion, and `%*d` takes its width from an operand. The format is reused
 * until the operands run out; a conversion past the last operand reads "" or 0.
 *
 * Measured against bash 5.3: the flags - + space # 0, width and precision (a
 * literal or *; a negative * width left-justifies, a negative * precision is
 * none), the length modifiers h l L j z t (ignored), and d i o u x X c s b q.
 * Width and precision count BYTES. A numeric operand is read like strtoimax with
 * base 0 (0x1f is 31, 010 is 8) - strtoumax for o u x X - or 'C / "C for a
 * character's code; an operand that is not entirely a number is reported and
 * its leading number used, with status 1. An invalid conversion character ends
 * the output there, status 1.
 * %e %f %g %a are not implemented: the operand is printed as it is (bash-rt.c
 * has no double).
 *
 * Escapes: the FORMAT knows \a \b \e \E \f \n \r \t \v \\ \" \' \?, \NNN octal
 * (one to three digits, \0 included), \xHH, \uHHHH and \UHHHHHHHH, and keeps
 * anything else - \c included - as written. A %b operand knows the same, reads
 * octal as \0NNN or \NNN, and its \c ends ALL output.
 *
 * A NUL byte is written to the output like any other, but a string here cannot
 * hold one, so the text printf and echo -e build is ENCODED: byte 1 escapes the
 * byte after it, "\x01" "0" for NUL and "\x01\x01" for a 1 byte itself. pf_vlen
 * and pf_vcut count such a pair as the one byte it stands for; rt_pf_out writes
 * the text decoded, rt_pf_value is the value printf -v stores (it ends at a NUL).
 * Twin of printfBuiltin in bash-interpreter.abnf. */
int printf_rc;
int pf_stop;      /* a %b operand's \c: no further output                 */
char *pf_e;       /* the text pf_escape appends to, a field list of pieces */
int pf_ai;        /* the next operand                                      */
int pf_ap;        /* where it starts in the operand list (rt_fpos)         */
int pf_have;      /* the operand pf_arg answered existed                   */

/* One byte as encoded text (see above). */
char *pf_chr(int c)
{
    char *d;
    if (c == 0 || c == 1) {
        d = rt_bump(3);
        d[0] = 1;
        d[1] = (char)(c == 0 ? 48 : 1);
        d[2] = 0;
        return d;
    }
    d = rt_bump(2);
    d[0] = (char)c;
    d[1] = 0;
    return d;
}

/* A %s operand as encoded text: only its 1 bytes need escaping. */
char *pf_enc(char *s)
{
    char *d;
    int i = 0, n = 0, o = 0;
    while (s[i] != 0) {
        if (s[i] == 1) n = n + 1;
        i = i + 1;
    }
    if (n == 0) return s;
    d = rt_bump(i + n + 1);
    i = 0;
    while (s[i] != 0) {
        if (s[i] == 1) { d[o] = 1; o = o + 1; }
        d[o] = s[i];
        o = o + 1;
        i = i + 1;
    }
    d[o] = 0;
    return d;
}

/* Writes the UTF-8 pattern of code point cp to d and answers its length. bash
 * writes a value that is no character - a surrogate, or past U+10FFFF - in the
 * same pattern, up to six bytes for 2^31 - 1, and nothing from 2^31 on.
 * Twin of pfChr in bash-interpreter.abnf and ucsChr in bash-to-llvm-ir.abnf. */
int rt_utf8put(char *d, long cp)
{
    int n = 2;
    int k;
    if (cp < 128) {
        d[0] = (char)cp;
        return 1;
    }
    if (cp >= 2147483648) return 0;
    while (n < 6 && cp >= (1L << (5 * n + 1))) n = n + 1;
    for (k = n - 1; k > 0; k = k - 1) {
        d[k] = (char)(128 + cp % 64);
        cp = cp / 64;
    }
    d[0] = (char)(((65280 >> n) & 255) + cp);
    return n;
}

/* The UTF-8 encoding of a code point (\u / \U). */
char *pf_utf8(long cp)
{
    char *d;
    if (cp < 128) return pf_chr((int)cp);
    d = rt_bump(7);
    d[rt_utf8put(d, cp)] = 0;
    return d;
}

int pf_hexval(int c)
{
    if (c >= 48 && c <= 57) return c - 48;
    if (c >= 97 && c <= 102) return c - 87;
    if (c >= 65 && c <= 70) return c - 55;
    return -1;
}

/* The escape at s[i] (a backslash), appended to pf_e; answers the bytes it used.
 * pctb is 0 for the format, 1 for a %b operand, 2 for `echo -e` (pf_bstr). */
int pf_escape(char *s, int i, int pctb)
{
    int c = (int)(unsigned char)s[i + 1];
    int k = 0;
    long v = 0;
    int n;
    int maxd;
    char *t = 0;
    if (c == 0) {
        pf_e = rt_fadd(pf_e, "\\");
        return 1;
    }
    if (c == 97) t = pf_chr(7);
    if (c == 98) t = pf_chr(8);
    if (c == 101 || c == 69) t = pf_chr(27);
    if (c == 102) t = pf_chr(12);
    if (c == 110) t = pf_chr(10);
    if (c == 114) t = pf_chr(13);
    if (c == 116) t = pf_chr(9);
    if (c == 118) t = pf_chr(11);
    if (c == 92) t = "\\";
    if (pctb == 0) {
        if (c == 34) t = "\"";
        if (c == 39) t = "'";
        if (c == 63) t = "?";
    }
    if (t != 0) {
        pf_e = rt_fadd(pf_e, t);
        return 2;
    }
    if (c >= 48 && c <= 55 && (pctb != 2 || c == 48)) {
        /* the format: one to three digits; %b: \0 and then up to three, or \NNN;
         * echo -e: only \0 and then up to three */
        n = i + 1;
        maxd = 3;
        if (pctb != 0 && c == 48) n = i + 2;
        while (k < maxd && s[n + k] >= 48 && s[n + k] <= 55) {
            v = v * 8 + (long)(s[n + k] - 48);
            k = k + 1;
        }
        pf_e = rt_fadd(pf_e, pf_chr((int)(v & 255)));
        return n + k - i;
    }
    if (c == 120 || c == 117 || c == 85) {
        maxd = 8;
        if (c == 120) maxd = 2;
        if (c == 117) maxd = 4;
        while (k < maxd && pf_hexval((int)(unsigned char)s[i + 2 + k]) >= 0) {
            v = v * 16 + (long)pf_hexval((int)(unsigned char)s[i + 2 + k]);
            k = k + 1;
        }
        if (k == 0) {
            pf_e = rt_fadd(pf_e, rt_substr(s, i, 2));
            return 2;
        }
        if (c == 120) pf_e = rt_fadd(pf_e, pf_chr((int)v));
        else pf_e = rt_fadd(pf_e, pf_utf8(v));
        return 2 + k;
    }
    if (c == 99 && pctb != 0) {
        pf_stop = 1;
        return 2;
    }
    pf_e = rt_fadd(pf_e, rt_substr(s, i, 2));
    return 2;
}

/* A %b operand (mode 1) or an `echo -e` word (mode 2) with its escapes applied
 * (pf_stop at \c). */
char *pf_bstr(char *s, int mode)
{
    char *saved = pf_e;
    char *sfa = fa_last;
    int sfl = fa_len;
    long lpos = h_pos[1];
    char *r;
    int i = 0;
    pf_e = list_nil;
    while (s[i] != 0 && pf_stop == 0) {
        if (s[i] == 92) {
            i = i + pf_escape(s, i, mode);
        } else {
            pf_e = rt_fadd(pf_e, pf_chr((int)(unsigned char)s[i]));
            i = i + 1;
        }
    }
    /* The pieces are garbage once joined; dropping them leaves the caller's list
     * the last one made, so its rt_fadd goes on in place. */
    r = rt_ljoin(pf_e, empty);
    rt_hset(1, lpos);
    fa_last = sfa;
    fa_len = sfl;
    pf_e = saved;
    return r;
}

int pf_bad(char *a, char *msg)
{
    int saved = sink_out;
    sink_out = sink_err;
    rt_prints("bash: printf: ");
    rt_prints(a);
    rt_prints(": ");
    rt_prints(msg);
    rt_prints("\n");
    sink_out = saved;
    printf_rc = 1;
    return 0;
}

/* Would the digits s[ds..de) in `base` exceed the largest magnitude the
 * conversion holds - 2^63 - 1, 2^63 for a negative signed operand, 2^64 - 1 for
 * an unsigned one (o u x X read like strtoumax, so 0xffffffffffffffff is fine)?
 * Compared as text against that limit, with the leading zeros dropped. */
int pf_over(char *s, int ds, int de, long base, int neg, int uns)
{
    char *lim;
    int n;
    int ln;
    int k;
    while (ds < de && s[ds] == 48) ds = ds + 1;
    n = de - ds;
    if (base == 10) lim = uns != 0 ? "18446744073709551615" : (neg != 0 ? "9223372036854775808" : "9223372036854775807");
    else if (base == 16) lim = uns != 0 ? "ffffffffffffffff" : (neg != 0 ? "8000000000000000" : "7fffffffffffffff");
    else lim = uns != 0 ? "1777777777777777777777" : (neg != 0 ? "1000000000000000000000" : "777777777777777777777");
    ln = rt_strlen(lim);
    if (n != ln) return n > ln ? 1 : 0;
    k = 0;
    while (k < n) {
        int c = (int)(unsigned char)s[ds + k];
        if (c >= 65 && c <= 70) c = c + 32;
        if (c != (int)(unsigned char)lim[k]) return c > (int)(unsigned char)lim[k] ? 1 : 0;
        k = k + 1;
    }
    return 0;
}

/* A numeric operand (see the head comment); uns for o u x X. */
long pf_num(char *a, int uns)
{
    int i = 0;
    int neg = 0;
    long base = 10;
    long v = 0;
    int ds;
    long d;
    if (a[0] == 39 || a[0] == 34) {
        if (a[1] == 0) return 0;
        return (long)rt_cpat(&a[1]);
    }
    while (a[i] == 32 || a[i] == 9 || a[i] == 10) i = i + 1;
    if (a[i] == 45 || a[i] == 43) {
        neg = a[i] == 45;
        i = i + 1;
    }
    if (a[i] == 48 && (a[i + 1] == 120 || a[i + 1] == 88) && pf_hexval((int)(unsigned char)a[i + 2]) >= 0) {
        base = 16;
        i = i + 2;
    } else if (a[i] == 48) {
        base = 8;
    }
    ds = i;
    while (1) {
        d = (long)pf_hexval((int)(unsigned char)a[i]);
        if (d < 0 || d >= base) break;
        v = v * base + d;
        i = i + 1;
    }
    if (pf_over(a, ds, i, base, neg, uns) != 0) {
        pf_bad(a, "Result too large");
        if (uns != 0) return -1;
        if (neg != 0) return 0 - 9223372036854775807L - 1;
        return 9223372036854775807L;
    }
    if (neg != 0) v = 0 - v;
    if (i == ds || a[i] != 0) {
        if (a[i] == 120 || a[i] == 88) pf_bad(a, "invalid hexadecimal number");
        else if (base == 8 && a[i] >= 48 && a[i] <= 57) pf_bad(a, "invalid octal number");
        else pf_bad(a, "invalid number");
    }
    return v;
}

/* Pad s to width w: spaces on the left, on the right (left), or zeros after the
 * first `keep` bytes (a sign or a 0x prefix). */
/* The number of bytes encoded text stands for, and its first n of them. */
int pf_vlen(char *s)
{
    int i = 0;
    int n = 0;
    while (s[i] != 0) {
        if (s[i] == 1 && s[i + 1] != 0) i = i + 1;
        i = i + 1;
        n = n + 1;
    }
    return n;
}

char *pf_vcut(char *s, int n)
{
    int i = 0;
    int k = 0;
    while (s[i] != 0 && k < n) {
        if (s[i] == 1 && s[i + 1] != 0) i = i + 1;
        i = i + 1;
        k = k + 1;
    }
    return rt_substr(s, 0, i);
}

/* Encoded text written out, and the value it is up to its first NUL. */
int rt_pf_out(char *s)
{
    int i = 0;
    while (s[i] != 0) {
        if (s[i] == 1 && s[i + 1] != 0) {
            i = i + 1;
            rt_putc(s[i] == 48 ? 0 : 1);
        } else {
            rt_putc((int)(unsigned char)s[i]);
        }
        i = i + 1;
    }
    return 0;
}

char *rt_pf_value(char *s)
{
    char *d = rt_bump(rt_strlen(s) + 1);
    int i = 0;
    int o = 0;
    while (s[i] != 0) {
        if (s[i] == 1 && s[i + 1] != 0) {
            i = i + 1;
            if (s[i] == 48) break;
        }
        d[o] = s[i];
        o = o + 1;
        i = i + 1;
    }
    d[o] = 0;
    return d;
}

char *pf_pad(char *s, int w, int left, int zero, int keep)
{
    int len = pf_vlen(s);
    int n = w > len ? w - len : 0;
    char *fill = n > 0 ? rt_bump(n + 1) : &empty[0];
    int i = 0;
    while (i < n) {
        fill[i] = zero != 0 ? 48 : 32;
        i = i + 1;
    }
    if (n > 0) fill[n] = 0;
    if (left != 0) return rt_strcat(s, fill);
    if (zero != 0) return rt_strcat(rt_strcat(rt_substr(s, 0, keep), fill), rt_substr(s, keep, -1));
    return rt_strcat(fill, s);
}

/* One integer conversion: conv is d i o u x or X. */
char *pf_int(long v, int conv, int left, int plus, int space, int alt, int zero, int w, int prec)
{
    char *dg;
    char *pre = empty;
    int len;
    int base = 16;
    if (conv == 100 || conv == 105) {
        dg = rt_int2str(v);
        if (dg[0] == 45) {
            dg = rt_substr(dg, 1, -1);
            pre = "-";
        } else if (plus != 0) {
            pre = "+";
        } else if (space != 0) {
            pre = " ";
        }
    } else {
        if (conv == 111) base = 8;
        if (conv == 117) base = 10;
        dg = rt_radix(rt_int2str(v), base, conv == 88 ? 1 : 0, 0);
    }
    if (prec == 0 && v == 0) dg = empty;
    len = rt_strlen(dg);
    while (len < prec) {
        dg = rt_strcat("0", dg);
        len = len + 1;
    }
    if (conv == 111 && alt != 0 && dg[0] != 48) dg = rt_strcat("0", dg);
    if ((conv == 120 || conv == 88) && alt != 0 && v != 0) {
        if (conv == 88) pre = "0X";
        else pre = "0x";
    }
    return pf_pad(rt_strcat(pre, dg), w, left, (zero != 0 && prec < 0 && left == 0) ? 1 : 0, rt_strlen(pre));
}

/* The next operand of the field list args (nargs of them), "" past the end. */
char *pf_arg(char *args, int nargs)
{
    int k = pf_ai;
    char *a;
    pf_ai = k + 1;
    pf_have = k < nargs ? 1 : 0;
    if (k >= nargs) return empty;
    a = rt_fat(args, pf_ap);
    pf_ap = rt_fnext(args, pf_ap);
    return a;
}

/* The conversion at fmt[i] (just past the "%"), appended to pf_e; answers the index
 * after it, or -1 for an invalid conversion character. */
int pf_conv(char *fmt, int i, char *args, int nargs)
{
    int left = 0, plus = 0, space = 0, alt = 0, zero = 0;
    int w = 0, prec = -1, cv, saved;
    long nv;
    char *a;
    char *piece;
    int start = i - 1;
    while (fmt[i] == 45 || fmt[i] == 43 || fmt[i] == 32 || fmt[i] == 35 || fmt[i] == 48) {
        if (fmt[i] == 45) left = 1;
        if (fmt[i] == 43) plus = 1;
        if (fmt[i] == 32) space = 1;
        if (fmt[i] == 35) alt = 1;
        if (fmt[i] == 48) zero = 1;
        i = i + 1;
    }
    if (fmt[i] == 42) {
        nv = pf_num(pf_arg(args, nargs), 0);
        if (nv < 0) {
            left = 1;
            nv = 0 - nv;
        }
        w = (int)nv;
        i = i + 1;
    } else {
        while (fmt[i] >= 48 && fmt[i] <= 57) {
            w = w * 10 + (fmt[i] - 48);
            i = i + 1;
        }
    }
    if (fmt[i] == 46) {
        i = i + 1;
        prec = 0;
        if (fmt[i] == 42) {
            nv = pf_num(pf_arg(args, nargs), 0);
            prec = nv < 0 ? -1 : (int)nv;
            i = i + 1;
        } else {
            while (fmt[i] >= 48 && fmt[i] <= 57) {
                prec = prec * 10 + (fmt[i] - 48);
                i = i + 1;
            }
        }
    }
    while (fmt[i] != 0 && rt_haschar("hlLjzt", (int)(unsigned char)fmt[i]) != 0) i = i + 1;
    cv = (int)(unsigned char)fmt[i];
    if (cv == 0 || rt_haschar("diouxXcsbqeEfFgGaA", cv) == 0) {
        saved = sink_out;
        sink_out = sink_err;
        rt_prints("bash: printf: `");
        if (cv == 0) rt_prints(rt_substr(fmt, start, -1));
        else rt_prints(rt_substr(fmt, i, 1));
        rt_prints("': invalid format character\n");
        sink_out = saved;
        printf_rc = 1;
        return -1;
    }
    i = i + 1;
    a = pf_arg(args, nargs);
    if (rt_haschar("diouxX", cv) != 0) {
        nv = 0;
        if (pf_have != 0) nv = pf_num(a, (cv == 100 || cv == 105) ? 0 : 1);
        piece = pf_int(nv, cv, left, plus, space, alt, zero, w, prec);
    } else {
        piece = pf_enc(a);
        if (cv == 99) piece = a[0] != 0 ? pf_chr((int)(unsigned char)a[0]) : pf_chr(0);
        if (cv == 98) piece = pf_bstr(a, 1);
        if (cv == 113) piece = rt_pfq(a);
        if (prec >= 0 && cv != 99) piece = pf_vcut(piece, prec);
        piece = pf_pad(piece, w, left, zero, 0);
    }
    pf_e = rt_fadd(pf_e, piece);
    return i;
}

/* The whole command's output; printf_rc is its status. */
char *rt_printf(char *fmt, char *args)
{
    int nargs = rt_nfields(args);
    int pass = 0;
    int used = 0;
    int i;
    int c;
    pf_e = list_nil;
    pf_stop = 0;
    pf_ai = 0;
    pf_ap = rt_fpos(args);
    printf_rc = 0;
    while (pass == 0 || (used != 0 && pf_ai < nargs && pf_stop == 0)) {
        int before = pf_ai;
        pass = pass + 1;
        i = 0;
        while (fmt[i] != 0 && pf_stop == 0) {
            c = (int)(unsigned char)fmt[i];
            if (c == 92) {
                i = i + pf_escape(fmt, i, 0);
            } else if (c == 37 && fmt[i + 1] == 37) {
                pf_e = rt_fadd(pf_e, "%");
                i = i + 2;
            } else if (c == 37) {
                i = pf_conv(fmt, i + 1, args, nargs);
                if (i < 0) return rt_ljoin(pf_e, empty);
            } else {
                pf_e = rt_fadd(pf_e, pf_chr(c));
                i = i + 1;
            }
        }
        used = pf_ai > before ? 1 : 0;
    }
    return rt_ljoin(pf_e, empty);
}

/* ---- field lists ----------------------------------------------------------
 * A plain string is ONE field; a list (rt_islist) is "\x02" followed by each
 * field and a "\x01" after it. Inside a field a 0x02 escapes the byte after it,
 * so a field can hold 0x01 and 0x02 themselves (`x=$'a\x01b'; printf '%s' "$x"`
 * is one operand): rt_fput writes a field so, rt_getfield reads it back. Twin of
 * encFields / decFields in bash-interpreter.abnf. */

/* The field f, escaped, and its terminator, written at dst + d; answers where
 * the list's NUL went. lf and ne are f's length and its count of 0x01 / 0x02. */
int rt_fwrite(char *dst, int d, char *f, int lf, int ne) {
    int i = 0;
    if (ne == 0) {
        memmove(dst + d, f, (unsigned long)lf);
        d = d + lf;
    } else {
        while (i < lf) {
            if (f[i] == 1 || f[i] == 2) { dst[d] = 2; d = d + 1; }
            dst[d] = f[i];
            d = d + 1;
            i = i + 1;
        }
    }
    dst[d] = 1;
    dst[d + 1] = 0;
    return d + 1;
}

/* The count of bytes of f that rt_fwrite escapes. */
int rt_fesc(char *f) {
    int i = 0, ne = 0;
    while (f[i] != 0) {
        if (f[i] == 1 || f[i] == 2) ne = ne + 1;
        i = i + 1;
    }
    return ne;
}

/* The list acc followed by the field f, escaped, and its terminator. acc is
 * left as it was. */
char *rt_fput(char *acc, char *f) {
    int la = (int)strlen(acc);
    int lf = (int)strlen(f);
    int ne = rt_fesc(f);
    char *dst = rt_lbump(la + lf + ne + 2);
    memmove(dst, acc, (unsigned long)la);
    fa_len = rt_fwrite(dst, la, f, lf, ne);
    fa_last = dst;
    return dst;
}

/* 1 when the list acc can grow by n bytes in place: it is the list made last,
 * nothing has been allocated in heap 1 since, and its chunk has the room. */
int rt_ftop(char *acc, int n) {
    if (acc != fa_last || h_adj[1] + h_pos[1] != (long)(acc + fa_len + 1)) return 0;
    return (long)n < h_cend[1] - h_pos[1] ? 1 : 0;
}

/* rt_fput for a caller that OWNS acc and never reads it again (a builder's
 * accumulator): f is written onto acc's end in place when rt_ftop allows, so a
 * loop of them is linear where a loop of rt_fput copies the whole list every
 * time. */
char *rt_fadd(char *acc, char *f) {
    int lf = (int)strlen(f);
    int ne = rt_fesc(f);
    if (rt_ftop(acc, lf + ne + 1) == 0) return rt_fput(acc, f);
    h_pos[1] = h_pos[1] + lf + ne + 1;
    fa_len = rt_fwrite(acc, fa_len, f, lf, ne);
    return acc;
}

/* WALKING A LIST: rt_fpos(v) is where its first field starts (-1: there is
 * none), rt_fnext(v, p) where the field after the one at p starts, rt_fat(v, p)
 * the field at p. A plain string is one field, at 0; a list's fields start at 1
 * or later. Each step reads only its own field, so a walk is linear where
 * rt_getfield(v, i) for every i is quadratic. */
int rt_fpos(char *v) {
    if (rt_islist(v) == 0) return 0;
    return v[1] == 0 ? -1 : 1;
}

int rt_fnext(char *v, int p) {
    int i = p;
    if (p == 0) return -1;
    while (v[i] != 0 && v[i] != 1) {
        if (v[i] == 2 && v[i + 1] != 0) i = i + 1;
        i = i + 1;
    }
    if (v[i] == 0) return -1;
    i = i + 1;
    return v[i] == 0 ? -1 : i;
}

char *rt_fat(char *v, int p) {
    int i = p, ne = 0, o = 0;
    char *dst;
    if (p == 0) return v;
    while (v[i] != 0 && v[i] != 1) {
        if (v[i] == 2 && v[i + 1] != 0) { i = i + 1; ne = ne + 1; }
        i = i + 1;
    }
    dst = rt_bump(i - p - ne + 1);
    if (ne == 0) {
        memmove(dst, v + p, (unsigned long)(i - p));
        dst[i - p] = 0;
        return dst;
    }
    while (p < i) {
        if (v[p] == 2) p = p + 1;
        dst[o] = v[p];
        o = o + 1;
        p = p + 1;
    }
    dst[o] = 0;
    return dst;
}

/* The fields of the list l joined with sep into one string, in one
 * allocation. A plain l is its own one field. */
char *rt_ljoin(char *l, char *sep) {
    int ls;
    int n = 0, nf = 0, i = 1, o = 0;
    char *dst;
    if (rt_islist(l) == 0) return l;
    ls = (int)strlen(sep);
    while (l[i] != 0) {
        if (l[i] == 2 && l[i + 1] != 0) i = i + 1;
        else if (l[i] == 1) nf = nf + 1;
        n = n + 1;
        i = i + 1;
    }
    if (nf == 0) return &empty[0];
    dst = rt_bump(n - nf + (nf - 1) * ls + 1);
    i = 1;
    while (l[i] != 0) {
        if (l[i] == 2 && l[i + 1] != 0) {
            i = i + 1;
            dst[o] = l[i];
            o = o + 1;
        } else if (l[i] == 1) {
            if (l[i + 1] != 0) {
                memmove(dst + o, sep, (unsigned long)ls);
                o = o + ls;
            }
        } else {
            dst[o] = l[i];
            o = o + 1;
        }
        i = i + 1;
    }
    dst[o] = 0;
    return dst;
}

int rt_nfields(char *v) {
    int i = 1, n = 0;
    if (rt_islist(v) == 0) return 1;
    while (v[i] != 0) {
        if (v[i] == 2 && v[i + 1] != 0) i = i + 1;
        else if (v[i] == 1) n = n + 1;
        i = i + 1;
    }
    return n;
}

/* Field k of a list, its escapes removed. */
char *rt_getfield(char *v, int k) {
    int p = k < 0 ? -1 : rt_fpos(v);
    while (k > 0 && p >= 0) {
        p = rt_fnext(v, p);
        k = k - 1;
    }
    return p < 0 ? &empty[0] : rt_fat(v, p);
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
    if (rt_islist(l) == 0 || rt_isfin(l) != 0) return l;
    if (len < 2) return sep == 0 ? &list_nil[0] : &empty[0];
    if (sep == 0) {
        dst = rt_lbump(len + 1);
        /* the list as it is, minus the terminator of each empty field; fs is
         * where the field being copied starts. */
        int fs = 1;
        dst[0] = 2;
        o = 1;
        while (i < len) {
            if (l[i] == 2 && i + 1 < len) {
                dst[o] = 2;
                dst[o + 1] = l[i + 1];
                o = o + 2;
                i = i + 1;
            } else if (l[i] != 1 || o > fs) {
                dst[o] = l[i];
                o = o + 1;
                if (l[i] == 1) fs = o;
            }
            i = i + 1;
        }
    } else {
        dst = rt_bump(len + 1);
        while (i < len) {
            if (l[i] == 2 && i + 1 < len) { i = i + 1; dst[o] = l[i]; o = o + 1; }
            else if (l[i] != 1) { dst[o] = l[i]; o = o + 1; }
            else if (i < len - 1) { dst[o] = ifs[0]; o = o + 1; }
            i = i + 1;
        }
    }
    dst[o] = 0;
    return dst;
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
    int ne = 0;
    int i;
    int o;
    int pass;
    char *dst;
    dst = &empty[0];
    /* A field list is split as the one value rt_uqlist makes of it; under a null
     * IFS that is a list again, already split. */
    if (rt_islist(v) != 0) {
        v = rt_uqlist(v, ifs);
        if (rt_islist(v) != 0) return v;
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
     * (list_nil below). */
    if (len > 0) {
        while (q < len && rt_haschar(ifs, (int)(unsigned char)v[q]) == 0) { q = q + 1; }
        if (q == len) return v;
    }
    pass = 0;
    while (pass < 2) {
        i = 0;
        nf = 0;
        tot = 0;
        ne = 0;
        o = 1;
        while (i < len && rt_ifsws(ifs, (int)(unsigned char)v[i]) != 0) { i = i + 1; }
        while (i < len) {
            while (i < len && rt_haschar(ifs, (int)(unsigned char)v[i]) == 0) {
                if (v[i] == 1 || v[i] == 2) {
                    ne = ne + 1;
                    if (pass != 0) { dst[o] = 2; o = o + 1; }
                }
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
            if (nf == 0) return &list_nil[0];
            /* EXACTLY ONE FIELD AND NO SEPARATOR BYTE AT ALL - by far the
             * commonest case, and it needs no allocation, because a plain string
             * IS a one-field list in this protocol: rt_nfields answers 1 for it,
             * rt_getfield(v, 0) answers v, rt_ljoin returns it unchanged, and
             * rt_bnd_acc/rt_bnd_open both go through rt_nfields. */
            if (nf == 1 && tot == len) return v;
            dst = rt_lbump(1 + tot + ne + nf + 1);
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
 * acc is the word's own accumulator, never read again once passed in, so it
 * grows with rt_fadd (and so does rt_bnd_acc2's).
 */
char *rt_bnd_acc(char *acc, char *open, char *f) {
    int n = rt_nfields(f);
    char *op = acc;
    int i = 1;
    int p = rt_fpos(f);
    if (n < 2) return op;
    op = rt_fadd(acc, rt_strcat(open, rt_fat(f, p)));
    while (i < n - 1) {
        p = rt_fnext(f, p);
        op = rt_fadd(op, rt_fat(f, p));
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
    int p = rt_fpos(f);
    if (lead != 0 && isopen != 0) {
        op = rt_fadd(op, cur);
        cur = &empty[0];
        isopen = 0;
    }
    if (n >= 2) {
        op = rt_fadd(op, rt_strcat(cur, rt_fat(f, p)));
        while (i < n) {
            p = rt_fnext(f, p);
            cur = rt_fat(f, p);
            if (i < n - 1) op = rt_fadd(op, cur);
            i = i + 1;
        }
        isopen = 1;
    } else if (n == 1) {
        cur = rt_strcat(cur, rt_fat(f, p));
        isopen = 1;
    }
    if (trail != 0 && isopen != 0) op = rt_fadd(op, cur);
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
 * One RECORD per array name, found through an open-addressing table of the
 * names (rt_nhash, linear probing, doubled at half full), so a lookup costs the
 * name and not the size of every other array. Record r keeps its rows in two
 * malloc'd vectors, ar_k[r] / ar_v[r] (ar_n[r] used of ar_cap[r]), in the order
 * ${!a[*]} and ${a[@]} report: index order for an indexed array (rt_arr_iset),
 * insertion order for an associative one (rt_arr_set). ar_srt[r] says every key
 * is a canonical decimal (rt_iscanon) and they ascend, which lets a lookup
 * bisect; any other record of 16 rows or more is looked up through ar_kh[r], an
 * open-addressing table of row + 1 by key, built on the first lookup, extended by
 * an append and dropped by anything that moves rows. ar_mk[r] holds the marks
 * (rt_arr_mark 1, rt_arr_dmark 2, rt_arr_smark 4). A record is never removed:
 * one with no row and no mark is an absent array. The name is copied; keys and
 * values are heap strings, held by rt_pin like any other stored value.
 */
char **ar_nm;
char ***ar_k;
char ***ar_v;
int *ar_n;
int *ar_cap;
int *ar_mk;
int *ar_srt;
int **ar_kh;
int *ar_khsz;
int ar_cnt;
int ar_rcap;
int *ar_h;            /* record + 1, 0 = empty */
int ar_hsz;           /* a power of two, or 0 before the first record */
long arr_rows;        /* the rows of every record: 0 answers rt_arr_any at once */

void *realloc(void *p, unsigned long n);

/* djb2 over the name's bytes; the caller masks it to its table. */
int rt_nhash(char *n)
{
    int h = 5381;
    int i = 0;
    while (n[i] != 0) {
        h = (h * 33) ^ (int)(unsigned char)n[i];
        i = i + 1;
    }
    return h;
}

/* A block of n bytes from malloc, or the fail-stop. */
char *rt_xalloc(char *what, long n)
{
    char *p = (char *)malloc((unsigned long)n);
    if (p == 0) rt_die(what, (int)n, 0, 0);
    return p;
}

/* The same block grown to n bytes, its contents kept. */
char *rt_xgrow(char *what, char *p, long n)
{
    char *q = (char *)realloc(p, (unsigned long)n);
    if (q == 0) rt_die(what, (int)n, 0, 0);
    return q;
}

/* The record of array nm, or -1. */
int rt_ar_find(char *nm)
{
    int m;
    int h;
    int e;
    if (ar_cnt == 0) return -1;
    m = ar_hsz - 1;
    h = rt_nhash(nm) & m;
    e = ar_h[h];
    while (e != 0) {
        if (rt_streq(nm, ar_nm[e - 1]) != 0) return e - 1;
        h = (h + 1) & m;
        e = ar_h[h];
    }
    return -1;
}

void rt_ar_hput(int r)
{
    int m = ar_hsz - 1;
    int h = rt_nhash(ar_nm[r]) & m;
    while (ar_h[h] != 0) h = (h + 1) & m;
    ar_h[h] = r + 1;
}

/* The record of array nm, made empty when there is none. */
int rt_ar_rec(char *nm)
{
    int r = rt_ar_find(nm);
    int i;
    int len;
    if (r >= 0) return r;
    if (ar_cnt == ar_rcap) {
        ar_rcap = ar_rcap == 0 ? 64 : ar_rcap * 2;
        ar_nm = (char **)rt_xgrow("array records", (char *)ar_nm, (long)ar_rcap * 8);
        ar_k = (char ***)rt_xgrow("array records", (char *)ar_k, (long)ar_rcap * 8);
        ar_v = (char ***)rt_xgrow("array records", (char *)ar_v, (long)ar_rcap * 8);
        ar_n = (int *)rt_xgrow("array records", (char *)ar_n, (long)ar_rcap * 4);
        ar_cap = (int *)rt_xgrow("array records", (char *)ar_cap, (long)ar_rcap * 4);
        ar_mk = (int *)rt_xgrow("array records", (char *)ar_mk, (long)ar_rcap * 4);
        ar_srt = (int *)rt_xgrow("array records", (char *)ar_srt, (long)ar_rcap * 4);
        ar_kh = (int **)rt_xgrow("array records", (char *)ar_kh, (long)ar_rcap * 8);
        ar_khsz = (int *)rt_xgrow("array records", (char *)ar_khsz, (long)ar_rcap * 4);
    }
    if (2 * (ar_cnt + 1) > ar_hsz) {
        if (ar_h != 0) free(ar_h);
        ar_hsz = ar_hsz == 0 ? 128 : ar_hsz * 2;
        ar_h = (int *)rt_xalloc("array names", (long)ar_hsz * 4);
        i = 0;
        while (i < ar_hsz) {
            ar_h[i] = 0;
            i = i + 1;
        }
        i = 0;
        while (i < ar_cnt) {
            rt_ar_hput(i);
            i = i + 1;
        }
    }
    r = ar_cnt;
    len = (int)strlen(nm);
    ar_nm[r] = rt_xalloc("array names", (long)len + 1);
    memmove(ar_nm[r], nm, (unsigned long)(len + 1));
    ar_k[r] = 0;
    ar_v[r] = 0;
    ar_n[r] = 0;
    ar_cap[r] = 0;
    ar_mk[r] = 0;
    ar_srt[r] = 1;
    ar_kh[r] = 0;
    ar_khsz[r] = 0;
    ar_cnt = r + 1;
    rt_ar_hput(r);
    return r;
}

/* Room in record r for n rows. */
void rt_ar_room(int r, int n)
{
    int cap = ar_cap[r];
    if (n <= cap) return;
    cap = cap == 0 ? 8 : cap * 2;
    if (cap < n) cap = n;
    ar_k[r] = (char **)rt_xgrow("array rows", (char *)ar_k[r], (long)cap * 8);
    ar_v[r] = (char **)rt_xgrow("array rows", (char *)ar_v[r], (long)cap * 8);
    ar_cap[r] = cap;
}

/* Is k a decimal as rt_int2str writes it: 0, or an optional minus and a nonzero
 * digit, at most 18 digits so it cannot overflow? */
int rt_iscanon(char *k)
{
    int i = k[0] == 45 ? 1 : 0;
    int d = i;
    if (k[i] == 48) return k[i + 1] == 0 && i == 0 ? 1 : 0;
    while (k[i] >= 48 && k[i] <= 57) i = i + 1;
    return k[i] == 0 && i > d && i - d <= 18 ? 1 : 0;
}

/* Record r's key table: row i entered, or dropped. */
void rt_ar_khput(int r, int i)
{
    int m = ar_khsz[r] - 1;
    int h = rt_nhash(ar_k[r][i]) & m;
    while (ar_kh[r][h] != 0) h = (h + 1) & m;
    ar_kh[r][h] = i + 1;
}

void rt_ar_khdrop(int r)
{
    if (ar_kh[r] != 0) free(ar_kh[r]);
    ar_kh[r] = 0;
}

/* The row of key k in record r, or -1. A sorted record holds canonical keys
 * only, so another key is not there, and a canonical one is bisected for. */
int rt_ar_pos(int r, char *k)
{
    int n = ar_n[r];
    char **ks = ar_k[r];
    int i;
    int m;
    if (ar_srt[r] != 0) {
        int lo = 0;
        int hi = n - 1;
        long kv;
        if (rt_iscanon(k) == 0) return -1;
        kv = rt_str2int(k);
        while (lo <= hi) {
            int mid = (lo + hi) / 2;
            long km = rt_str2int(ks[mid]);
            if (km == kv) return mid;
            if (km < kv) lo = mid + 1;
            else hi = mid - 1;
        }
        return -1;
    }
    if (n >= 16) {
        if (ar_kh[r] == 0) {
            m = 32;
            while (m < 4 * n) m = m * 2;
            ar_kh[r] = (int *)rt_xalloc("array keys", (long)m * 4);
            ar_khsz[r] = m;
            i = 0;
            while (i < m) {
                ar_kh[r][i] = 0;
                i = i + 1;
            }
            i = 0;
            while (i < n) {
                rt_ar_khput(r, i);
                i = i + 1;
            }
        }
        m = ar_khsz[r] - 1;
        i = rt_nhash(k) & m;
        while (ar_kh[r][i] != 0) {
            if (rt_streq(k, ks[ar_kh[r][i] - 1]) != 0) return ar_kh[r][i] - 1;
            i = (i + 1) & m;
        }
        return -1;
    }
    i = 0;
    while (i < n) {
        if (rt_streq(k, ks[i]) != 0) return i;
        i = i + 1;
    }
    return -1;
}

/* Insert (k, v) as row `at` of record r. */
void rt_ar_ins(int r, int at, char *k, char *v)
{
    int n = ar_n[r];
    rt_ar_room(r, n + 1);
    if (at < n) {
        memmove(ar_k[r] + at + 1, ar_k[r] + at, (unsigned long)(n - at) * 8);
        memmove(ar_v[r] + at + 1, ar_v[r] + at, (unsigned long)(n - at) * 8);
    }
    ar_k[r][at] = k;
    ar_v[r][at] = v;
    ar_n[r] = n + 1;
    arr_rows = arr_rows + 1;
    if (ar_kh[r] != 0) {
        if (at < n || 2 * (n + 1) > ar_khsz[r]) rt_ar_khdrop(r);
        else rt_ar_khput(r, n);
    }
}

/* i32 rt_arr_set(i8* nm, i8* k, i8* v): update the row of k, or append one. */
int rt_arr_set(char *nm, char *k, char *v) {
    int r = rt_ar_rec(nm);
    int at = rt_ar_pos(r, k);
    int n;
    rt_pin();
    rt_abheld(k);
    rt_abheld(v);
    if (at >= 0) {
        ar_v[r][at] = v;
        return 0;
    }
    n = ar_n[r];
    if (ar_srt[r] != 0 && (rt_iscanon(k) == 0 || (n > 0 && rt_str2int(ar_k[r][n - 1]) >= rt_str2int(k))))
        ar_srt[r] = 0;
    rt_ar_ins(r, n, k, v);
    return 0;
}

/* AN INDEXED ARRAY KEEPS ITS ROWS IN INDEX ORDER, the order bash lists ${!a[*]}
 * and ${a[@]} in whatever order the elements were assigned (a[5]=x; a[2]=y lists
 * 2 5): a new row goes in front of the first row of the same array with a greater
 * index, and a key is matched by its numeric value. An associative array keeps
 * insertion order (manual 7.9), so its writes stay on rt_arr_set. A write to
 * bad_key is dropped. */
char bad_key[1];
int rt_arr_iset(char *nm, char *k, char *v) {
    int r;
    int n;
    int pos;
    int i;
    long kv;
    char **ks;
    rt_abheld(v);
    if (k == &bad_key[0]) {
        arith_err = 1;
        return 0;
    }
    rt_pin();
    r = rt_ar_rec(nm);
    n = ar_n[r];
    ks = ar_k[r];
    kv = rt_str2int(k);
    if (ar_srt[r] != 0) {
        int lo = 0;
        int hi = n - 1;
        if (rt_iscanon(k) == 0) k = rt_int2str(kv);
        /* appending in ascending order - the common case - is one compare */
        if (n > 0 && rt_str2int(ks[n - 1]) < kv) lo = n;
        while (lo <= hi) {
            int mid = (lo + hi) / 2;
            long km = rt_str2int(ks[mid]);
            if (km == kv) {
                ar_v[r][mid] = v;
                return 0;
            }
            if (km < kv) lo = mid + 1;
            else hi = mid - 1;
        }
        rt_ar_ins(r, lo, k, v);
        return 0;
    }
    /* An unsorted record (an associative write came first): scan from the end
     * for the row to update or the place to insert. */
    pos = 0;
    i = n - 1;
    while (i >= 0) {
        long ki = rt_str2int(ks[i]);
        if (ki == kv) {
            ar_v[r][i] = v;
            return 0;
        }
        if (ki < kv) {
            pos = i + 1;
            break;
        }
        i = i - 1;
    }
    rt_ar_ins(r, pos, k, v);
    return 0;
}

/* The key an INDEXED subscript's value names: a negative index counts back from
 * one past the highest one (${a[-1]} is the last element; a set scalar is
 * element 0). One still negative after that is bash's "bad array subscript":
 * reported, and answered with bad_key, which no element has and rt_arr_iset
 * refuses to write. Twin of subKey in bash-interpreter.abnf. */
char *rt_arr_ikey(char *nm, long k) {
    int saved;
    if (k < 0) k = (rt_arr_any(nm) != 0 ? rt_arr_nextidx(nm) : (long)rt_sc_count(nm)) + k;
    if (k < 0) {
        saved = sink_out;
        sink_out = sink_err;
        rt_prints("bash: ");
        rt_prints(nm);
        rt_prints(": bad array subscript\n");
        sink_out = saved;
        return &bad_key[0];
    }
    return rt_int2str(k);
}

/* The names declared associative anywhere in the program, registered by main's
 * entry block from the emitter's whole-program list: a subscript of one of them is
 * a literal key, not arithmetic, wherever it is met at run time. */
char *assoc_nm[256];
int assoc_n;
int rt_assoc_add(char *nm) {
    if (assoc_n >= 256) {
        rt_die("associative-array names", 1, assoc_n, 256);
    }
    assoc_nm[assoc_n] = nm;
    assoc_n = assoc_n + 1;
    return 0;
}
int rt_isassoc(char *nm) {
    int i = 0;
    while (i < assoc_n) {
        if (rt_streq(nm, assoc_nm[i]) != 0) return 1;
        i = i + 1;
    }
    return 0;
}

/* The key a subscript's TEXT names: an associative array's is the text itself, an
 * indexed array's the value of the text as an arithmetic expression; "@" and "*"
 * name the whole array and pass through. Twin of subKey in bash-interpreter.abnf. */
char *rt_arr_key(char *nm, char *sub) {
    if (rt_streq(sub, "@") != 0 || rt_streq(sub, "*") != 0) return sub;
    if (rt_isassoc(nm) != 0) return sub;
    return rt_arr_ikey(nm, rt_arith(sub));
}

/* One [k]=v of an INDEXED array literal. The answer is the index a following bare
 * word takes, one past k - b=([3]=c [1]=a b) puts b at 2 - or `next` unchanged
 * when k was a bad subscript. */
long rt_arr_litkey(char *nm, char *k, char *v, long next) {
    char *key = rt_arr_key(nm, k);
    if (key == &bad_key[0]) return next;
    rt_arr_iset(nm, key, v);
    return rt_str2int(key) + 1;
}

/* The fields of a bare word of an INDEXED array literal, stored from index next
 * on; the answer is the index after the last one. */
/* The fields `let` word arguments expanded to, each its own expression, in order; the
 * first error stops the command. With `need`, the list is every argument of the command
 * and no field at all is bash's "expression expected".
 * Twin of the letws arm of makeLetCmd in bash-interpreter.abnf. */
long rt_letfields(char *list, int need)
{
    int p = rt_fpos(list);
    long v = 0;
    if (p < 0 && need != 0) rt_arith_fail("let", "expression expected", empty);
    while (p >= 0 && arith_err == 0) {
        v = rt_arith(rt_fat(list, p));
        p = rt_fnext(list, p);
    }
    return v;
}

long rt_arr_litwords(char *nm, char *f, long next) {
    int p = rt_fpos(f);
    while (p >= 0) {
        rt_arr_iset(nm, rt_int2str(next), rt_fat(f, p));
        next = next + 1;
        p = rt_fnext(f, p);
    }
    return next;
}

/* i8* rt_arr_get(i8* nm, i8* k): the value, or "" when absent. */
char *rt_arr_get(char *nm, char *k) {
    int r = rt_ar_find(nm);
    int at;
    if (r < 0) return empty;
    at = rt_ar_pos(r, k);
    if (at >= 0) return ar_v[r][at];
    return empty;
}

/* i32 rt_arr_has(i8* nm, i8* k) */
int rt_arr_has(char *nm, char *k) {
    int r = rt_ar_find(nm);
    if (r < 0) return 0;
    return rt_ar_pos(r, k) >= 0 ? 1 : 0;
}

/* i32 rt_arr_del(i8* nm, i8* k): remove one row; the array stays set
 * (rt_arr_mark). On a name with no row, key 0 is the scalar, which is unset:
 * `p=ab; unset 'p[0]'` leaves p unset in bash 5.3. */
int rt_arr_del(char *nm, char *k) {
    int r = rt_ar_find(nm);
    int at;
    int n;
    if (r < 0 || ar_n[r] == 0) {
        if (rt_streq(k, "0") != 0) rt_setvar_byname(nm, &unset_marker[0]);
        return 0;
    }
    at = rt_ar_pos(r, k);
    if (at < 0) return 0;
    n = ar_n[r] - 1;
    if (at < n) {
        memmove(ar_k[r] + at, ar_k[r] + at + 1, (unsigned long)(n - at) * 8);
        memmove(ar_v[r] + at, ar_v[r] + at + 1, (unsigned long)(n - at) * 8);
    }
    ar_n[r] = n;
    arr_rows = arr_rows - 1;
    ar_mk[r] = ar_mk[r] | 1;
    rt_ar_khdrop(r);
    return 0;
}

/* 1 when the array has a row at all: the test every variable read makes. */
int rt_arr_any(char *nm) {
    int r;
    if (arr_rows == 0) return 0;
    r = rt_ar_find(nm);
    return r >= 0 && ar_n[r] > 0 ? 1 : 0;
}

/* i32 rt_arr_count(i8* nm) */
int rt_arr_count(char *nm) {
    int r = rt_ar_find(nm);
    return r >= 0 ? ar_n[r] : 0;
}

/* i8* rt_arr_list(i8* nm, i8* sep, i32 star, i32 keys): the keys or the values
 * of an array, as a field list (star == 0) or joined with sep (star != 0). */
char *rt_arr_list(char *nm, char *sep, int star, int keys) {
    int r = rt_ar_find(nm);
    char *out = list_nil;
    int i = 0;
    int n = r >= 0 ? ar_n[r] : 0;
    while (i < n) {
        out = rt_fadd(out, keys != 0 ? ar_k[r][i] : ar_v[r][i]);
        i = i + 1;
    }
    return star != 0 ? rt_ljoin(out, sep) : out;
}

/* rt_arr_nextidx(nm): one past the largest non-negative numeric key, 0 for
 * none - a sorted record's last row. */
long rt_arr_nextidx(char *nm) {
    int r = rt_ar_find(nm);
    int i = 0;
    int n;
    long mx = 0;
    if (r < 0) return 0;
    n = ar_n[r];
    if (ar_srt[r] != 0) {
        if (n == 0) return 0;
        mx = rt_str2int(ar_k[r][n - 1]);
        return mx >= 0 ? mx + 1 : 0;
    }
    while (i < n) {
        long kn = rt_str2int(ar_k[r][i]);
        mx = kn >= mx ? kn + 1 : mx;
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
char *rt_finlist(char *v) {
    if (rt_islist(v) != 0) fin_last = rt_lcat(v, empty);
    else fin_last = rt_fput(list_nil, v);
    return fin_last;
}

int rt_isfin(char *v) {
    return v == fin_last ? 1 : 0;
}

/* AN ARRAY ASSIGNED A LITERAL IS SET EVEN WITH NO ELEMENT: bash 5.3 lists `e=()`
 * in ${!e*} and not `declare -a e`. A record with no row is otherwise absent, so
 * the literal leaves mark 1, which rt_arr_clear drops with the rows and the
 * subshell snapshot copies with them. Unsetting an element leaves the same mark:
 * `b[0]=1; unset 'b[0]'` is the set, empty `declare -a b=()`. Twin of the `lit`
 * flag of an array in bash-interpreter.abnf. */
int rt_arr_mark(char *nm) {
    int r = rt_ar_rec(nm);
    ar_mk[r] = ar_mk[r] | 1;
    return 0;
}

/* A DECLARED-ONLY ARRAY (`declare -a e` with no value) is not set, but
 * `declare -p e` prints `declare -a e`: mark 2 records the declaration. Twin of
 * the `decl` flag of an array in bash-interpreter.abnf. */
int rt_arr_dmark(char *nm) {
    int r = rt_ar_rec(nm);
    ar_mk[r] = ar_mk[r] | 2;
    return 0;
}

/* A SCALAR DECLARED WITHOUT A VALUE (`declare x`, or a local unset in the
 * function that declared it) is not set, but `declare -p x` prints
 * `declare -- x`: mark 4. Twin of dclVars in bash-interpreter.abnf. */
int rt_arr_smark(char *nm) {
    int r = rt_ar_rec(nm);
    ar_mk[r] = ar_mk[r] | 4;
    return 0;
}

/* NAME's marks, 0 for a name without a record. */
int rt_arr_marks(char *nm) {
    int r = rt_ar_find(nm);
    return r >= 0 ? ar_mk[r] : 0;
}

int rt_arr_marked(char *nm) {
    return rt_arr_marks(nm) & 1;
}

int rt_arr_dmarked(char *nm) {
    return (rt_arr_marks(nm) & 2) != 0 ? 1 : 0;
}

int rt_arr_smarked(char *nm) {
    return (rt_arr_marks(nm) & 4) != 0 ? 1 : 0;
}

/* i32 rt_arr_clear(i8* nm): drop every row of one array, and its marks. */
int rt_arr_clear(char *nm) {
    int r = rt_ar_find(nm);
    if (r < 0) return 0;
    arr_rows = arr_rows - ar_n[r];
    ar_n[r] = 0;
    ar_mk[r] = 0;
    ar_srt[r] = 1;
    rt_ar_khdrop(r);
    return 0;
}

/* A SET SCALAR BECOMES ELEMENT 0 of the array an element assignment or an
 * appending literal makes of its name: bash 5.3 answers `abcd x` for
 * t=abcd; t[1]=x; echo "${t[*]}". Slot id is left unset, so the name reads
 * through the array from then on. Twin of arrAdopt in bash-interpreter.abnf. */
int rt_arr_adopt(char *nm, int id) {
    char *v = gvars[id];
    if (v == &unset_marker[0] || rt_arr_any(nm) != 0) return 0;
    rt_arr_iset(nm, "0", v);
    gvars[id] = &unset_marker[0];
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
 * WHY FOUR SEPARATE NAMES AND NOT A CHANGE TO rt_arr_any. rt_arr_any IS the test
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
    if (rt_arr_any(nm) != 0) return rt_arr_get(nm, k);
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
    if (rt_arr_any(nm) != 0) return rt_arr_has(nm, k);
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
    if (rt_arr_any(nm) != 0) return rt_arr_list(nm, sep, star, keys);
    v = rt_getvar_byname(nm);
    if (v == &unset_marker[0]) return star != 0 ? &empty[0] : &list_nil[0];
    if (keys != 0) v = "0";
    if (star != 0) return v;
    return rt_fput(list_nil, v);
}

/* The positional parameters as a field list that starts at $0, which is what
 * ${@:off:len} counts from ("${@:1}" is $1 on). */
char *rt_params0(void) {
    return rt_catfields(rt_fput(list_nil, script_name), rt_params(empty, 0));
}

/* ${!p*} / ${!p@}: the names of the SET variables starting with p, sorted - a
 * scalar that is not the unset marker, or an array with an element or a literal
 * (rt_arr_mark; arrays are records, not scalar slots). Joined with sep when star is set, a
 * field list otherwise. Twin of varNamesWithPrefix in
 * languages/bash-interpreter.abnf. */
char *rt_pfxnames(char *p, char *sep, int star) {
    char *nm[1024];
    int n = 0;
    int pl = rt_strlen(p);
    int total = nvars + ar_cnt;
    int i = 0;
    int j;
    int dup;
    char *v;
    char *out;
    while (i < total) {
        v = i < nvars ? varnames[i] : ar_nm[i - nvars];
        int ok = rt_strlen(v) >= pl && rt_streq(rt_substr(v, 0, pl), p) != 0;
        if (ok && rt_getvar_byname(v) == &unset_marker[0] && rt_arr_any(v) == 0
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
    out = list_nil;
    i = 0;
    while (i < n) {
        out = rt_fadd(out, nm[i]);
        i = i + 1;
    }
    return star != 0 ? rt_ljoin(out, sep) : out;
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
/* The $'...' form, with bash's escape table - the same one ansiQuote uses in
 * languages/bash-interpreter.abnf. A byte that does not start a well-formed
 * UTF-8 character is \NNN, and so is each byte of a character rt_wprint
 * rejects; any other well-formed one is copied whole.
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
        int w = 1;
        if (c >= 128) w = rt_cpw(v + i);
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
        } else if (c < 32 || c == 127 || (c >= 128 && (w == 1 || rt_wprint(rt_cpat(v + i)) == 0))) {
            /* the byte, or every byte of a character rt_wprint rejects, as \NNN */
            int k = 0;
            while (k < w) {
                int b = (int)(unsigned char)v[i + k];
                d[o] = (char)92; o = o + 1;
                d[o] = (char)(48 + (b / 64)); o = o + 1;
                d[o] = (char)(48 + ((b / 8) % 8)); o = o + 1;
                d[o] = (char)(48 + (b % 8)); o = o + 1;
                k = k + 1;
            }
        } else {
            /* a byte, or a whole well-formed character, as itself */
            int k = 0;
            while (k < w) { d[o] = v[i + k]; o = o + 1; k = k + 1; }
        }
        i = i + w;
    }
    d[o] = (char)39; o = o + 1;
    d[o] = 0;
    return d;
}

/* "..." unless rt_needansi. A well-formed UTF-8 character does not need it: in
 * a UTF-8 locale bash prints the accented character intact. */
char *rt_dqs(char *v) {
    int n = rt_strlen(v);
    int i = 0;
    char *d;
    int o = 0;
    if (rt_needansi(v) != 0) return rt_ansiqs(v);
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
    int r = rt_ar_find(nm);
    int n = r >= 0 ? ar_n[r] : 0;
    int c = 0;
    while (c < n) {
        char *k = ar_k[r][c];
        if (c > 0) out = rt_strcat(out, " ");
        out = rt_strcat(out, "[");
        out = rt_strcat(out, rt_keymeta(k) != 0 ? rt_dqs(k) : k);
        out = rt_strcat(out, "]=");
        out = rt_strcat(out, rt_dqs(ar_v[r][c]));
        c = c + 1;
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
    int isarr = (rt_arr_any(nm) != 0 || assoc != 0) ? 1 : 0;
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
    if (ifs[0] == 0) return rt_fput(rt_fput(rt_fput(list_nil, "declare"), flag), asg);
    val = rt_substr(asg, rt_strlen(nm) + 1, rt_strlen(asg) - rt_strlen(nm) - 1);
    txt = rt_strcat(rt_strcat(rt_strcat("declare ", flag), " "), rt_strcat(nm, "="));
    fs = rt_splitifs(txt, ifs);
    if (rt_islist(fs) == 0) return rt_strcat(fs, val);
    n = rt_strlen(fs);
    if (n < 2 || rt_ifstrail(txt, ifs) != 0) return rt_fput(fs, val);
    /* glued onto the last field: drop its terminator and append */
    return rt_fput(rt_substr(fs, 0, n - 1), val);
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
        out = list_nil;
        while (i < cnt) {
            out = rt_fadd(out, rt_shquote(argv[frame_base + i]));
            i = i + 1;
        }
        return rt_strcat("set -- ", rt_ljoin(out, sep));
    }
    if (cnt == 0) return list_nil;
    if (ifs[0] == 0) {
        out = rt_fput(rt_fput(list_nil, "set"), "--");
    } else {
        fs = rt_splitifs("set -- ", ifs);
        n = rt_strlen(fs);
        if (rt_islist(fs) == 0) {
            out = rt_fput(list_nil, rt_strcat(fs, rt_shquote(argv[frame_base])));
            i = 1;
        } else if (n < 2 || rt_ifstrail("set -- ", ifs) != 0) {
            out = fs;
        } else {
            out = rt_fput(rt_substr(fs, 0, n - 1), rt_shquote(argv[frame_base]));
            i = 1;
        }
    }
    while (i < cnt) {
        out = rt_fadd(out, rt_shquote(argv[frame_base + i]));
        i = i + 1;
    }
    return out;
}

/* ${a[@]@K}: `key "value"` pairs as ONE field, with the trailing space bash
 * writes for the associative form only. */
char *rt_kvq(char *nm, int assoc, char *v, int isset) {
    char *out = list_nil;
    int r = rt_ar_find(nm);
    int n = r >= 0 ? ar_n[r] : 0;
    int c = 0;
    if (n == 0 && assoc == 0) return rt_shq(v, isset);
    while (c < n) {
        char *k = ar_k[r][c];
        out = rt_fadd(out, rt_keymeta(k) != 0 ? rt_dqs(k) : k);
        out = rt_fadd(out, rt_dqs(ar_v[r][c]));
        c = c + 1;
    }
    if (assoc != 0 && c > 0) out = rt_fadd(out, empty);
    return rt_ljoin(out, " ");
}

/* ${a[@]@k}: the same pairs, unquoted, as separate FIELDS. */
char *rt_kvp(char *nm, char *sep, int star, char *v, int isset) {
    char *out = list_nil;
    int r = rt_ar_find(nm);
    int n = r >= 0 ? ar_n[r] : 0;
    int i = 0;
    if (n == 0) return rt_shq(v, isset);
    while (i < n) {
        out = rt_fadd(rt_fadd(out, ar_k[r][i]), ar_v[r][i]);
        i = i + 1;
    }
    return star != 0 ? rt_ljoin(out, sep) : out;
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
    out = list_nil;
    while (i < n) {
        out = rt_fadd(out, at);
        i = i + 1;
    }
    return star != 0 ? rt_ljoin(out, sep) : out;
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
    int p = rt_fpos(f);
    while (p >= 0) {
        rt_arr_set(nm, rt_int2str(rt_arr_nextidx(nm)), rt_fat(f, p));
        p = rt_fnext(f, p);
    }
    return 0;
}

/* ${a[@]:off:len} on an INDEXED array counts INDICES, not positions: the slice
 * starts at the first element whose index is >= off (a negative off is relative
 * to one past the highest index) and takes len elements, all of them when haslen
 * is 0. bash 5.3 with sp[2]=x sp[5]=y sp[9]=z: ${sp[*]:3} is "y z", ${sp[*]: -4}
 * is "z". A name with no rows (a scalar, one element at index 0) is sliced by
 * position, which for it is the same thing. A negative len is bash's "substring
 * expression < 0". */
char *rt_arr_slice(char *nm, long off, long len, int haslen, char *sep, int star) {
    int r = rt_ar_find(nm);
    int n = r >= 0 ? ar_n[r] : 0;
    int isStar = star != 0;
    char *out = isStar ? empty : list_nil;
    int i = 0;
    long taken = 0;
    if (haslen != 0 && len < 0) {
        rt_arith_fail(rt_int2str(len), "substring expression < 0", empty);
        return out;
    }
    if (rt_arr_any(nm) == 0) {
        char *all = rt_sc_list(nm, sep, 0, 0);
        int cnt = rt_nfields(all);
        if (off < 0) off = off + cnt;
        if (off < 0 || off > cnt) return out;
        if (haslen == 0 || len > cnt) len = cnt;
        return rt_slicefields(all, (int)off, (int)len, sep, star);
    }
    if (off < 0) {
        off = rt_arr_nextidx(nm) + off;
        if (off < 0) return out;
    }
    out = list_nil;
    while (i < n) {
        if (rt_str2int(ar_k[r][i]) >= off && (haslen == 0 || taken < len)) {
            out = rt_fadd(out, ar_v[r][i]);
            taken = taken + 1;
        }
        i = i + 1;
    }
    return isStar ? rt_ljoin(out, sep) : out;
}

/* i8* rt_slicefields(i8* f, i32 off, i32 n, i8* sep, i32 star) */
char *rt_slicefields(char *f, int off, int n, char *sep, int star) {
    char *out = list_nil;
    int p = off < 0 ? -1 : rt_fpos(f);
    int taken = 0;
    while (off > 0 && p >= 0) {
        p = rt_fnext(f, p);
        off = off - 1;
    }
    while (p >= 0 && taken < n) {
        out = rt_fadd(out, rt_fat(f, p));
        p = rt_fnext(f, p);
        taken = taken + 1;
    }
    return star != 0 ? rt_ljoin(out, sep) : out;
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

/* i8* rt_rxescape(i8* s): make every ERE metacharacter of s literal, which is
 * what a QUOTED part of a [[ =~ ]] regex is ([[ abc =~ "a.c" ]] is false). */
char *rt_rxescape(char *s) {
    int len = rt_strlen(s);
    char *dst = rt_bump(len * 2 + 1);
    int i = 0, o = 0;
    while (i < len) {
        int ch = (int)(unsigned char)s[i];
        int meta = ch == 92 || ch == 46 || ch == 91 || ch == 93 || ch == 40 || ch == 41 || ch == 42
                || ch == 43 || ch == 63 || ch == 123 || ch == 125 || ch == 124 || ch == 94 || ch == 36;
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
    int at = rt_fpos(v);
    char *out = list_nil;
    while (at >= 0) {
        char *f = rt_fat(v, at);
        char *g;
        if (op == 0) g = rt_strip(f, p, mode);
        else if (op == 1) g = rt_replace(f, p, r, mode);
        else if (op == 3) g = rt_shquote(f);
        else if (op == 4) g = rt_ansic(f);
        else if (op == 5) g = rt_prompt(f);
        else g = rt_case(f, mode, p);
        out = rt_fadd(out, g);
        at = rt_fnext(v, at);
    }
    return star != 0 ? rt_ljoin(out, sep) : out;
}

/* i8* rt_catfields(i8* a, i8* b): the list a followed by b's fields, a plain b
 * being one field. a is the caller's accumulator (a command's argument list),
 * so it grows in place like rt_fadd's. */
char *rt_catfields(char *a, char *b) {
    int lb;
    if (rt_islist(b) == 0) return rt_fadd(a, b);
    lb = (int)strlen(b) - 1;
    if (rt_ftop(a, lb) != 0) {
        h_pos[1] = h_pos[1] + lb;
        memmove(a + fa_len, b + 1, (unsigned long)(lb + 1));
        fa_len = fa_len + lb;
        return a;
    }
    a = rt_lcat(a, &b[1]);
    fa_last = a;
    fa_len = (int)strlen(a);
    return a;
}

/* `echo` with the FLAG SCAN AT RUN TIME, so `opts="-n"; echo $opts x` treats
 * the expanded -n as a flag, as bash does. `newline` and
 * `escapes` carry in whatever the constant prefix already settled, and the scan
 * continues over the fields the expansions produced - which is the same order
 * languages/bash-interpreter.abnf's echoCmd walks. */
/* An echo option word is "-" and one or more of n e E (`-en` = `-e -n`): 1 when f is one.
 * Twin of echoFlags in languages/bash-interpreter.abnf. */
int echo_flags(char *f) {
    int j = 1;
    if (f[0] != 45 || f[1] == 0) return 0;
    while (f[j] != 0) {
        if (f[j] != 110 && f[j] != 101 && f[j] != 69) return 0;
        j = j + 1;
    }
    return 1;
}

int rt_echo(char *fields, int newline, int escapes) {
    int p = rt_fpos(fields);
    int flagging = 1;
    int first = 1;
    pf_stop = 0;
    while (p >= 0) {
        char *f = rt_fat(fields, p);
        p = rt_fnext(fields, p);
        if (flagging != 0) {
            if (echo_flags(f) != 0) {
                int j = 1;
                while (f[j] != 0) {
                    if (f[j] == 110) newline = 0;
                    else escapes = f[j] == 101 ? 1 : 0;
                    j = j + 1;
                }
                continue;
            }
            flagging = 0;
        }
        if (first == 0) rt_prints(" ");
        if (escapes != 0) rt_pf_out(pf_bstr(f, 2));
        else rt_prints(f);
        if (pf_stop != 0) return 0;     /* \c: no further output, not even the newline */
        first = 0;
    }
    if (newline != 0) rt_putc(10);
    return 0;
}

/* i32 rt_argpush(i8* v): append every FIELD of v to the argv stack, doubling
 * the stack when it is full. */
int rt_argpush(char *v) {
    int p = rt_fpos(v);
    char **na;
    while (p >= 0) {
        int t = argv_top;
        if (t >= argv_cap) {
            argv_cap = argv_cap == 0 ? 4096 : argv_cap * 2;
            na = (char **)malloc((unsigned long)argv_cap * 8);
            if (na == 0) rt_die("positional-parameter stack", 1, t, argv_cap);
            if (t > 0) memmove(na, argv, (unsigned long)t * 8);
            if (argv != 0) free(argv);
            argv = na;
        }
        argv[t] = rt_fat(v, p);
        rt_abheld(argv[t]);
        argv_top = t + 1;
        rt_pin();
        p = rt_fnext(v, p);
    }
    return 0;
}

/* i8* rt_param(i32 n): the n-th positional parameter ("" past the end). */
/* A positional parameter past $# is UNSET, so ${2-word} and ${1+word} can tell
 * it from a set, empty one. */
char *rt_param(int n) {
    int cnt = frame_n;
    if (n >= 1 && n <= cnt) {
        int idx = frame_base + (n - 1);
        return argv[idx];
    }
    return &unset_marker[0];
}

/* i8* rt_params(i8* sep, i32 star): "$@" when sep is the field marker, "$*"
 * when it is a separator. */
char *rt_params(char *sep, int star) {
    int cnt = frame_n;
    int base = frame_base;
    char *out = list_nil;
    int i = 0;
    while (i < cnt) {
        out = rt_fadd(out, argv[base + i]);
        i = i + 1;
    }
    return star != 0 ? rt_ljoin(out, sep) : out;
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
                rt_pin();
                cap_sz[d - 1] = nsz;
                buf = nb;
            }
            /* A command substitution drops a NUL byte, as bash does. */
            if (c != 0) {
                buf[l] = (char)c;
                cap_len[d - 1] = l + 1;
            } else {
                cap_nul[d - 1] = 1;
            }
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
    /* CAPMAX. Dying, not refusing the push: the matching rt_cap_end would
     * still decrement cap_depth and index cap_buf[-1]. */
    if (d >= 4096) {
        rt_die("command-substitution nesting", 1, d, 4096);
    }
    if (cap_sz[d] == 0) {
        buf = rt_bump(8192);
        cap_buf[d] = buf;
        rt_pin();
        cap_sz[d] = 8192;
    }
    cap_len[d] = 0;
    cap_nul[d] = 0;
    cap_depth = d + 1;
    return 0;
}

/* Close the innermost buffer and answer a COPY of it (the buffer is reused) with
 * its trailing newlines dropped. Copied here, not by rt_substr: that answers its
 * argument itself for a whole-string slice, and this buffer is not immutable.
 * cmdsub is 1 for a command substitution, which warns about a dropped NUL as
 * bash 5.3 does; a pipeline stage passes 0. Twin of mkCmdSub in
 * bash-interpreter.abnf. */
char *rt_cap_end(int cmdsub)
{
    int d2;
    char *buf;
    char *dst;
    int l;
    int i;
    int saved;
    d2 = cap_depth - 1;
    cap_depth = d2;
    if (cmdsub != 0 && cap_nul[d2] != 0) {
        saved = sink_out;
        sink_out = sink_err;
        rt_prints("bash: warning: command substitution: ignored null byte in input\n");
        sink_out = saved;
    }
    buf = cap_buf[d2];
    l = cap_len[d2];
    while (l > 0) {
        if (buf[l - 1] != (char)10) {
            break;
        }
        l = l - 1;
    }
    dst = rt_bump(l + 1);
    i = 0;
    while (i < l) {
        dst[i] = buf[i];
        i = i + 1;
    }
    dst[l] = (char)0;
    return dst;
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
    int i;
    int r;
    long p;
    char **blk;
    int *rec;
    d = ss_depth;
    if (d >= 4096) {
        rt_die("subshell nesting", 1, d, 4096);
    }
    if (ss_cap[d] < nvars + 2 * arr_rows || ss_blk[d] == 0) {
        ss_cap[d] = nvars + 4 * arr_rows + 64;
        if (ss_blk[d] != 0) free(ss_blk[d]);
        ss_blk[d] = (char **)rt_xalloc("subshell snapshot", ss_cap[d] * 8);
    }
    if (ss_rcap[d] < ar_cnt || ss_rec[d] == 0) {
        ss_rcap[d] = 2 * ar_cnt + 16;
        if (ss_rec[d] != 0) free(ss_rec[d]);
        ss_rec[d] = (int *)rt_xalloc("subshell snapshot", (long)ss_rcap[d] * 12);
    }
    blk = ss_blk[d];
    rec = ss_rec[d];
    i = 0;
    while (i < nvars) {
        blk[i] = gvars[i];
        i = i + 1;
    }
    p = nvars;
    r = 0;
    while (r < ar_cnt) {
        int n = ar_n[r];
        rec[3 * r] = n;
        rec[3 * r + 1] = ar_mk[r];
        rec[3 * r + 2] = ar_srt[r];
        if (n > 0) {
            memmove(blk + p, ar_k[r], (unsigned long)n * 8);
            memmove(blk + p + n, ar_v[r], (unsigned long)n * 8);
        }
        p = p + 2 * n;
        r = r + 1;
    }
    ss_an[d] = ar_cnt;
    ss_depth = d + 1;
    return 0;
}

/* Pop a whole variable snapshot. */
int rt_ss_restore(int u)
{
    int d;
    int i;
    int r;
    long p;
    char **blk;
    int *rec;
    d = ss_depth - 1;
    ss_depth = d;
    blk = ss_blk[d];
    rec = ss_rec[d];
    i = 0;
    while (i < nvars) {
        gvars[i] = blk[i];
        i = i + 1;
    }
    p = nvars;
    arr_rows = 0;
    r = 0;
    while (r < ar_cnt) {
        int n = 0;
        ar_mk[r] = 0;
        ar_srt[r] = 1;
        if (r < ss_an[d]) {
            n = rec[3 * r];
            ar_mk[r] = rec[3 * r + 1];
            ar_srt[r] = rec[3 * r + 2];
            rt_ar_room(r, n);
            if (n > 0) {
                memmove(ar_k[r], blk + p, (unsigned long)n * 8);
                memmove(ar_v[r], blk + p + n, (unsigned long)n * 8);
            }
            p = p + 2 * n;
        }
        ar_n[r] = n;
        arr_rows = arr_rows + n;
        rt_ar_khdrop(r);
        r = r + 1;
    }
    return 0;
}

/* Remember gvars[id]'s current value on the local save stack. */
int rt_push_local(int id)
{
    int t;
    int r;
    int cnt = 0;
    t = ls_top;
    /* LSMAX. Refusing the push would leak the "local" into the caller. */
    if (t >= 512) {
        rt_die("local save stack", 1, t, 512);
    }
    ls_id[t] = id;
    ls_val[t] = gvars[id];
    /* Copy this name's array rows aside: none when the name has no record,
     * which is why `local` in a hot recursive function does not pay for this. */
    ls_amark[t] = 0;
    r = rt_ar_find(varnames[id]);
    if (r >= 0) {
        cnt = ar_n[r];
        ls_amark[t] = ar_mk[r] | (ar_srt[r] != 0 ? 0 : 8);
        if (las_top + cnt > las_cap) {
            las_cap = 2 * (las_top + cnt) + 64;
            las_k = (char **)rt_xgrow("local array save stack", (char *)las_k, (long)las_cap * 8);
            las_v = (char **)rt_xgrow("local array save stack", (char *)las_v, (long)las_cap * 8);
        }
        if (cnt > 0) {
            memmove(las_k + las_top, ar_k[r], (unsigned long)cnt * 8);
            memmove(las_v + las_top, ar_v[r], (unsigned long)cnt * 8);
        }
        las_top = las_top + cnt;
    }
    ls_acnt[t] = cnt;
    ls_top = t + 1;
    return 0;
}

/* Put entry t's saved binding back; its array rows start at las index `rows`. */
int rt_ls_restore(int t, int rows)
{
    int cnt = ls_acnt[t];
    int r;
    char *nm = varnames[ls_id[t]];
    gvars[ls_id[t]] = ls_val[t];
    if (cnt == 0 && ls_amark[t] == 0) {
        rt_arr_clear(nm);
        return 0;
    }
    r = rt_ar_rec(nm);
    rt_ar_room(r, cnt);
    if (cnt > 0) {
        memmove(ar_k[r], las_k + rows, (unsigned long)cnt * 8);
        memmove(ar_v[r], las_v + rows, (unsigned long)cnt * 8);
    }
    arr_rows = arr_rows - ar_n[r] + cnt;
    ar_n[r] = cnt;
    ar_mk[r] = ls_amark[t] & 7;
    ar_srt[r] = (ls_amark[t] & 8) != 0 ? 0 : 1;
    rt_ar_khdrop(r);
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

/* The slot of the variable called n, or -1. Every by-name access - ${!r}, a name
 * in a run-time arithmetic string, declare - goes through here, so the names are
 * indexed once, the first time, in an open-addressing table (rt_nhash, linear
 * probing; nvars is final by then, main's entry block having filled varnames). */
int vh_tab[4096];     /* slot + 1, 0 = empty */
int vh_built;

int rt_vhash(char *n)
{
    return rt_nhash(n) & 4095;
}

int rt_varslot(char *n)
{
    int h;
    int k;
    if (vh_built == 0) {
        k = 0;
        while (k < nvars) {
            h = rt_vhash(varnames[k]);
            while (vh_tab[h] != 0) h = (h + 1) & 4095;
            vh_tab[h] = k + 1;
            k = k + 1;
        }
        vh_built = 1;
    }
    h = rt_vhash(n);
    while (vh_tab[h] != 0) {
        k = vh_tab[h] - 1;
        if (rt_streq(n, varnames[k]) != 0) return k;
        h = (h + 1) & 4095;
    }
    return -1;
}

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

/* The bare ${!r}: rt_getvar_byname, except that a name for a whole list - r='arr[@]'
 * or r='@' - answers a FIELD list, one field per element, as "${arr[@]}" does. Twin of
 * indVal in bash-interpreter.abnf. */
char *rt_getvar_ind(char *n)
{
    int len = rt_strlen(n);
    if (rt_streq(n, "@") != 0) return rt_params(rt_ifsfirst(), 0);
    if (len > 3 && n[len - 1] == 93 && n[len - 2] == 64 && n[len - 3] == 91)
        return rt_sc_list(rt_substr(n, 0, len - 3), rt_ifsfirst(), 0, 0);
    return rt_getvar_byname(n);
}

char *rt_getvar_byname(char *n)
{
    char *sel;
    int i;
    int c0;
    int allDigits;
    int len = rt_strlen(n);
    /* NAME[SUB]: ${!r} with r='arr[1]' reads the element, and r='arr[@]' the joined
     * list, exactly as ${arr[1]} / ${arr[@]} would. Twin of the subscript arm of
     * getVarVal in bash-interpreter.abnf. */
    if (len > 2 && n[len - 1] == 93) {
        i = 0;
        while (i < len && n[i] != 91) i = i + 1;
        if (i > 0 && i < len - 1) {
            char *nm = rt_substr(n, 0, i);
            char *sub = rt_substr(n, i + 1, len - i - 2);
            char *k;
            if (rt_streq(sub, "@") != 0 || rt_streq(sub, "*") != 0) return rt_sc_list(nm, rt_ifsfirst(), 1, 0);
            k = rt_arr_key(nm, sub);
            if (rt_sc_has(nm, k) == 0) return &unset_marker[0];
            return rt_sc_get(nm, k);
        }
    }
    /* THE SPECIAL PARAMETERS ARE NOT IN varnames[]: $0..$9, $#, $@, $* and $?
     * have no slot (`z=0; echo "${!z}"` is the script path in bash).
     * languages/bash-interpreter.abnf's getVarVal answers the same set. */
    if (rt_streq(n, "?") != 0) return rt_int2str(last_status);
    if (rt_streq(n, "#") != 0) return rt_int2str(frame_n);
    /* $@ and $* both come through joined here; the bare ${!r} asks rt_getvar_ind,
     * which hands "$@" back as a field list. */
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
    i = rt_varslot(n);
    if (i >= 0) sel = gvars[i];
    /* An array name reads its element 0, as $arr does. */
    if (sel == &unset_marker[0] && rt_arr_any(n) != 0) return rt_arr_get(n, "0");
    return sel;
}

/* The same lookup, writing instead of reading. A name the program never
 * mentions has no slot and is silently dropped. */
int rt_setvar_byname(char *n, char *v)
{
    int i = rt_varslot(n);
    if (i >= 0) gvars[i] = v;
    rt_pin();
    return 0;
}

/* `unset WORD` where the word came out of an expansion: NAME[SUB] removes the
 * element (the usual subscript rules), NAME the variable and its array. */
int rt_unset_name(char *n)
{
    int len = rt_strlen(n);
    int i = 0;
    if (len > 2 && n[len - 1] == 93) {
        while (i < len && n[i] != 91) i = i + 1;
        if (i > 0 && i < len - 1) {
            char *nm = rt_substr(n, 0, i);
            rt_arr_del(nm, rt_arr_key(nm, rt_substr(n, i + 1, len - i - 2)));
            return 0;
        }
    }
    rt_setvar_byname(n, &unset_marker[0]);
    rt_arr_clear(n);
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
 * Design: Thompson-style codegen plus a backtracking VM.
 *   * re_alt/re_cat/re_rep/re_atom parse the ERE and emit 4-word instructions
 *     into re_prog. Recursion is the C call stack, as rt_glob already does it.
 *   * re_run walks that program over an explicit backtrack stack (re_bx,
 *     re_by), so a long subject cannot exhaust the C stack. Leftmost-first,
 *     greedy.
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
 *   RE_BT  1048576  backtrack-stack entries          -> re_bx, re_by
 *   RE_STEPS 400000 VM steps per rt_regex_search call, on top of four per
 *            (instruction, subject position) pair: linear work never
 *            reaches it, the revisiting of a pathological pattern does
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
long re_steps;                /* NEW GLOBAL: VM steps used                  */
long re_cap;                  /* NEW GLOBAL: the step cap of one search     */
/* The backtrack stack, RE_BT entries: an alternative's pc and sp, or a
 * slot's undo (-1 - slot, its old value). */
int re_bx[1048576];           /* NEW GLOBAL                                 */
int re_by[1048576];           /* NEW GLOBAL                                 */
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

/* the backtracking VM. 1 match, 0 fail, -2 step cap (or a backtrack stack
 * deeper than RE_BT). A SPLIT pushes the alternative not taken, a SAVE or
 * CLEAR the old value of each slot it writes; a failure pops back to the
 * latest alternative, putting the slots back on the way. */
int re_run(int rpc, int rsp)
{
    int pc;
    int sp;
    long st;
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
    int top;
    int fail;
    int i;
    int bx;

    pc = rpc;
    sp = rsp;
    top = 0;
    slen = re_slen;
    subj = re_subj;
    while (1) {
        st = re_steps;
        if (st >= re_cap) {
            return -2;
        }
        re_steps = st + 1;
        op = re_get(pc, 0);
        vx = re_get(pc, 1);
        vy = re_get(pc, 2);
        atEnd = (sp >= slen);
        pc1 = pc + 1;
        sp1 = sp + 1;
        fail = 0;

        /* Every byte read is fenced by sp < slen, so no path loads the NUL
         * terminator or anything after it. */
        if (op == 1) {                        /* OP_CHAR */
            if (atEnd) {
                fail = 1;
            } else {
                cb = re_fold1((int)(unsigned char)subj[sp]);
                if (cb != vx) {
                    fail = 1;
                } else {
                    pc = pc1;
                    sp = sp1;
                }
            }
        } else if (op == 2) {                 /* OP_ANY */
            if (atEnd) {
                fail = 1;
            } else {
                pc = pc1;
                sp = sp1;
            }
        } else if (op == 3) {                 /* OP_CLASS */
            if (atEnd) {
                fail = 1;
            } else {
                lb = re_fold1((int)(unsigned char)subj[sp]);
                byt = (int)(unsigned char)re_cls[vx * 32 + lb / 8];
                got = (byt >> (lb % 8)) & 1;
                if (got == 0) {
                    fail = 1;
                } else {
                    pc = pc1;
                    sp = sp1;
                }
            }
        } else if (op == 4) {                 /* OP_SPLIT */
            if (top == 1048576) {
                return -2;
            }
            re_bx[top] = vy;
            re_by[top] = sp;
            top = top + 1;
            pc = vx;
        } else if (op == 5) {                 /* OP_JMP */
            pc = vx;
        } else if (op == 6) {                 /* OP_SAVE */
            /* A capture (or a loop's empty-iteration mark) is UNDONE when the
             * branch that set it fails, so a dead alternative cannot leave an
             * offset behind. */
            if (top == 1048576) {
                return -2;
            }
            re_bx[top] = -1 - vx;
            re_by[top] = re_slot[vx];
            top = top + 1;
            re_slot[vx] = sp;
            pc = pc1;
        } else if (op == 7) {                 /* OP_MATCH */
            return 1;
        } else if (op == 8) {                 /* OP_BOL */
            if (sp != 0) {
                fail = 1;
            } else {
                pc = pc1;
            }
        } else if (op == 9) {                 /* OP_EOL */
            if (sp != slen) {
                fail = 1;
            } else {
                pc = pc1;
            }
        } else if (op == 10) {                /* OP_PROG */
            if (sp == re_slot[vx]) {
                fail = 1;
            } else {
                pc = pc1;
            }
        } else if (op == 11) {                /* OP_CLEAR */
            /* OP_SAVE's undo, once per slot of the range. */
            if (top + (vy - vx + 1) > 1048576) {
                return -2;
            }
            for (i = vx; i < vy + 1; i++) {
                re_bx[top] = -1 - i;
                re_by[top] = re_slot[i];
                top = top + 1;
                re_slot[i] = -1;
            }
            pc = pc1;
        } else {
            fail = 1;
        }

        if (fail != 0) {
            while (1) {
                if (top == 0) {
                    return 0;
                }
                top = top - 1;
                bx = re_bx[top];
                if (bx >= 0) {
                    pc = bx;
                    sp = re_by[top];
                    break;
                }
                re_slot[-1 - bx] = re_by[top];
            }
        }
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
    re_cap = 400000 + (long)(slen + 1) * (long)re_pc * 4;
    re_steps = 0;
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
