/*
 * bash-rt.c - the bash compiler's string runtime, in C.
 *
 * languages/bash-to-llvm-ir.abnf used to hand-emit every one of these helpers as
 * LLVM IR built with the llvm.ir.* builder API - roughly 3,200 lines of MetaJS
 * whose only product was IR text. This file is the same runtime written once, in
 * C, and compiled by languages/c-to-llvm-ir.abnf into the checked-in
 * languages/lib/bash-rt.ll by tests/gen-bash-rt-ll.sh. See
 * docs/runtime-rework-plan.md.
 *
 * TWO THINGS THE PORT HAS TO RESPECT, both measured rather than assumed:
 *
 * 1. Every POINTER in a module produced by c-to-llvm-ir.abnf is an `i32*`,
 *    whatever it points at, while the bash module's strings are `i8*`. The two
 *    are ABI-identical but not type-identical, so the grammar wraps each helper
 *    in a tiny `rtw_*` shim that bitcasts. Call sites are unchanged.
 * 2. The globals below are the runtime's own state AND the emitted program's:
 *    the program loads last_status, writes gvars[], pushes argv. They are
 *    defined HERE and reached from the grammar by name, so there is exactly one
 *    definition of each.
 */


/* ---- the runtime's shared state ------------------------------------------
 * These globals are the runtime's AND the emitted program's: the program loads
 * last_status, writes gvars[], pushes argv. They are defined here, once, and
 * the grammar reaches them by name through llvm.SpliceGlobal.
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

/* Variable storage is ONE array of char*, indexed by a compile-time id, so the
 * `local` unwinder can restore a slot chosen at run time. NVARS = 1024. */
char *gvars[1024];
int nvars;

/* The `local` save stack: (slot id, previous value) pairs. LSMAX = 512. */
int ls_id[512];
char *ls_val[512];
int ls_top;

/* Output capture for $( ) / ` `: a stack of fixed-size arena buffers.
 * CAPMAX = 16, CAPSZ = 8192. */
char *cap_buf[16];
int cap_len[16];
int cap_depth;

/* Subshell isolation: ( LIST ) snapshots the whole variable array. SSDEPTH = 8. */
char *ss_save[8192];
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
char *rt_int2str(int n);
int rt_str2int(char *s);
int rt_class(char *p, int ch);
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
char *rt_case(char *s, int mode);
char *rt_shquote(char *sq);
char *rt_ansic(char *ac);
int rt_haschar(char *s, int ch);
char *rt_read_line(int u);
char *rt_field(char *l, char *ifs, int idx, int rest);
char *rt_pad(char *s, int w, int left, int zero);
char *rt_unescape(char *s);
int rt_nfields(char *v);
char *rt_getfield(char *v, int k);
char *rt_wordjoin(char *v);
char *rt_splitifs(char *v, char *ifs);
char *rt_bnd_acc(char *acc, char *open, char *f);
char *rt_bnd_open(char *open, char *f);
int rt_arr_find(char *nm, char *k);
int rt_arr_set(char *nm, char *k, char *v);
char *rt_arr_get(char *nm, char *k);
int rt_arr_has(char *nm, char *k);
int rt_arr_del(char *nm, char *k);
int rt_arr_count(char *nm);
char *rt_arr_list(char *nm, char *sep, int star, int keys);
int rt_arr_nextidx(char *nm);
int rt_arr_clear(char *nm);
int rt_arr_append(char *nm, char *f);
char *rt_slicefields(char *f, int off, int n, char *sep, int star);
char *rt_globescape(char *s);
char *rt_catfields(char *a, char *b);
int rt_argpush(char *v);
char *rt_param(int n);
char *rt_params(char *sep, int star);
int rt_regex_search(char *pat, char *subj, char *flags, int *caps, int ncaps);
char *rt_regex_error(void);
char *rt_nounset(char *v);
int rt_putc(int c);
int rt_cap_begin(int u);
char *rt_cap_end(int u);
int rt_prints(char *s);
int rt_ss_save(int u);
int rt_ss_restore(int u);
int rt_push_local(int id);
int rt_pop_locals(int d);
int rt_fskind(char *p);
int putchar(int c);

/* ---- the string heap ------------------------------------------------------
 * One byte arena plus a bump offset; rt_bump hands out slices and nothing is
 * ever freed. Same 4 MB the hand-emitted runtime reserved.
 */
char arena[4194304];
int arena_pos;

char *rt_bump(int n) {
    char *p = &arena[arena_pos];
    arena_pos = arena_pos + n;
    return p;
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

/* rt_strcat: fresh arena string = a followed by b. */
char *rt_strcat(char *a, char *b) {
    int la = rt_strlen(a);
    int lb = rt_strlen(b);
    char *dst = rt_bump(la + lb + 1);
    int i = 0, d = 0;
    while (a[i] != 0) { dst[d] = a[i]; d = d + 1; i = i + 1; }
    i = 0;
    while (b[i] != 0) { dst[d] = b[i]; d = d + 1; i = i + 1; }
    dst[d] = 0;
    return dst;
}

/* rt_int2str: decimal string in the arena (mirrors the interpreter's numToStr). */
char *rt_int2str(int n) {
    char tmp[16];
    char *dst = rt_bump(16);
    if (n == 0) { dst[0] = 48; dst[1] = 0; return dst; }
    int neg = n < 0;
    int a = neg ? 0 - n : n;
    int t = 15;
    while (a > 0) {
        tmp[t] = (char)(48 + (a % 10));
        t = t - 1;
        a = a / 10;
    }
    int d = 0;
    if (neg) { dst[0] = 45; d = 1; }
    int j = t + 1;
    while (j <= 15) { dst[d] = tmp[j]; d = d + 1; j = j + 1; }
    dst[d] = 0;
    return dst;
}

/* rt_str2int: leading spaces/tabs, one +/- sign, digits until a non-digit. */
int rt_str2int(char *s) {
    int i = 0, v = 0, sign = 1;
    while (s[i] == 32 || s[i] == 9) i = i + 1;
    if (s[i] == 45 || s[i] == 43) {
        if (s[i] == 45) sign = -1;
        i = i + 1;
    }
    while (s[i] >= 48 && s[i] <= 57) {
        v = v * 10 + ((int)s[i] - 48);
        i = i + 1;
    }
    return v * sign;
}

/* rt_class: p points at "[". Returns -1 when the bracket expression is
 * unterminated (the "[" is then a literal), else (bytesConsumed << 1) | matched. */
int rt_class(char *p, int ch) {
    int j = 1, neg = 0, m = 0, first = 1;
    int c0 = (int)(unsigned char)p[1];
    if (c0 == 33 || c0 == 94) { neg = 1; j = 2; }
    while (1) {
        int cj = (int)(unsigned char)p[j];
        if (cj == 0) return -1;
        if (cj == 93 && first == 0) break;
        first = 0;
        int cn = (int)(unsigned char)p[j + 1];
        int cn2 = 0;
        if (cn == 45) cn2 = (int)(unsigned char)p[j + 2];
        if (cn == 45 && cn2 != 0 && cn2 != 93) {
            if (ch >= cj && ch <= cn2) m = 1;
            j = j + 3;
        } else {
            if (ch == cj) m = 1;
            j = j + 1;
        }
    }
    int fin = neg ? 1 - m : m;
    return ((j + 1) * 2) | fin;
}


/* ================= frag-glob.c ================= */

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
    /* n < 0 means "to the end"; otherwise the end is off + n characters in.
     * BOTH rt_charoff calls happen unconditionally, as in the IR. */
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
 * rt_eg is mutually recursive with rt_glob; the prelude's prototype is the
 * forward declaration the IR expressed by declaring the function early.
 */

/* THE ONE PLACE THE C SUBSET COULD NOT EXPRESS THE IR DIRECTLY.
 * rt_glob and rt_eg are mutually recursive, so one of them is necessarily
 * CALLED before it is defined - and languages/c-to-llvm-ir.abnf builds a
 * not-yet-defined callee with non-pointer parameters (getFuncByArity pushes
 * {ptr: false} for every argument), so the later definition then tries to store
 * an i32 parameter into an i32** slot and the compile fails with
 *   store operands are not compatible: src=i32; dst=i32**
 * Minimal repro, nothing to do with bash:
 *   int g(char *a);
 *   int f(char *a) { return g(a); }
 *   int g(char *a) { return a[0]; }
 * The forward call therefore goes through INTEGER parameters, which the default
 * spec already matches, and rt_eg_fwd converts them back. See
 * docs/runtime-rework-plan.md; the fix belongs in c-to-llvm-ir.abnf and is not
 * made here.
 */
int rt_eg_fwd(long egp, long egs, int egstar);

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
        if (isPfx && nextc == 40) return rt_eg_fwd((long)(void *)pcur, (long)(void *)sp, 0);

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
                sp = scur + 1;
            }
        }

        scur2 = sp;
        sc = (int)(unsigned char)scur2[0];
        if (sc == 0) return 0;

        if (pc == 63) {
            pp = pcur + 1;
            sp = scur2 + 1;
            continue;
        }
        if (pc == 91) {
            cr = rt_class(pcur, sc);
            if (cr >= 0) {
                matched = cr & 1;
                off = cr / 2;
                if (matched == 0) return 0;
                pp = pcur + off;
                sp = scur2 + 1;
                continue;
            }
            /* cr < 0: not a bracket expression after all - fall through to
             * the literal arm, exactly as the IR does. */
        } else {
            nc = (int)(unsigned char)pcur[1];
            isEsc = (pc == 92) && (nc != 0);
            if (isEsc) {
                if (nc != sc) return 0;
                pp = pcur + 2;
                sp = scur2 + 1;
                continue;
            }
        }
        /* literal */
        if (pc != sc) return 0;
        pp = pcur + 1;
        sp = scur2 + 1;
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
            /* the IR evaluates BOTH operands of the And, so rt_glob runs even
             * when `any` is already set. */
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
                     * would not terminate. The IR computes the And's second
                     * operand - this call - unconditionally. */
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

int rt_eg_fwd(long egp, long egs, int egstar)
{
    return rt_eg((char *)(void *)egp, (char *)(void *)egs, egstar);
}
/* ================= frag-strip.c ================= */
/* ---- chunk: rt_strip .. rt_read_line -------------------------------------- */

/* NEW GLOBAL: the hand-emitted IR defines `read_eof` right in front of
 * rt_read_line (m.NewGlobalDef("read_eof", zero)); the grammar loads it by name
 * for the `read` builtin's status, so it has to live in the C file too. */
int read_eof;

/* rt_strip: ${v#p} / ${v##p} / ${v%p} / ${v%%p}; mode 0 = #, 1 = ##, 2 = %, 3 = %%.
 * A trial length t walks 0..len; k is t for the ascending modes (# and %%) and
 * len - t for the descending ones (## and %), so the first hit is the shortest
 * or the longest match as required. BOTH substrings are built every round,
 * exactly as the IR's select does, because the arena offsets are observable. */
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

/* rt_case: ${v^} / ${v^^} / ${v,} / ${v,,}; mode 0 = ^, 1 = ^^, 2 = ",", 3 = ",,". */
char *rt_case(char *s, int mode) {
    int len = rt_strlen(s);
    char *dst = rt_bump(len + 1);
    int up = mode < 2;
    int all = (mode == 1) || (mode == 3);
    int i = 0;
    while (i < len) {
        int c0 = (int)(unsigned char)s[i];
        int isLow = (c0 >= 97) && (c0 <= 122);
        int isUp = (c0 >= 65) && (c0 <= 90);
        int raised = isLow ? c0 - 32 : c0;
        int lowered = isUp ? c0 + 32 : c0;
        int mapped = up ? raised : lowered;
        int touch = all || (i == 0);
        dst[i] = (char)(touch ? mapped : c0);
        i = i + 1;
    }
    dst[len] = 0;
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
                int hi = c0 / 16;
                int lo = c0 % 16;
                dst[o] = (char)92; o = o + 1;
                dst[o] = (char)120; o = o + 1;
                dst[o] = (char)(hi < 10 ? hi + 48 : hi + 87); o = o + 1;
                dst[o] = (char)(lo < 10 ? lo + 48 : lo + 87); o = o + 1;
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

/* rt_ansic: the ${v@E} form - the $'...' escapes decoded at RUN time. */
char *rt_ansic(char *ac) {
    int len = rt_strlen(ac);
    char *dst = rt_bump(len + 1);
    int i = 0;
    int o = 0;
    while (i < len) {
        int c0 = (int)(unsigned char)ac[i];
        int c1 = (int)(unsigned char)ac[(i + 1 < len) ? i + 1 : i];
        if ((c0 == 92) && (i + 1 < len)) {
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

/* ================= frag-field.c ================= */
/* ---- fields, padding, escapes, field lists -------------------------------- */

/* rt_field: field `idx` of line `l` split on the characters of `ifs` (empty
 * fields dropped); with rest != 0 the whole remainder from that field on, which
 * is what the LAST name of a `read` gets. */
char *rt_field(char *l, char *ifs, int idx, int rest) {
    int len = rt_strlen(l);
    int i = 0;      /* scan index        */
    int n = 0;      /* field number      */
    int s = 0;      /* current field start */
    int in = 0;     /* inside a field?   */
    while (i < len) {
        int cch = (int)(unsigned char)l[i];
        if (rt_haschar(ifs, cch) != 0) {
            /* a separator closes the field we were in */
            if (in != 0) {
                if (n == idx && rest == 0) return rt_substr(l, s, i - s);
                n = n + 1;
                in = 0;
            }
        } else {
            /* a field character */
            if (in == 0) {
                in = 1;
                s = i;
                /* the wanted field starts here and the caller asked for the
                 * whole remainder */
                if (n == idx && rest != 0) return rt_substr(l, i, -1);
            }
        }
        i = i + 1;
    }
    /* end of line: the last field may still be open */
    if (in != 0 && n == idx) return rt_substr(l, s, -1);
    return &empty[0];
}

/* rt_pad: printf's field padding. Both arms of the IR select are evaluated, so
 * BOTH rt_strcat calls run on every iteration and in this order. */
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

/* rt_wordjoin: a field list flattened back to one string, fields space-joined.
 * The separator rt_strcat is inside an IR select, so it runs on EVERY iteration
 * (including i == 0) and its allocation is part of the arena order. */
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

/* rt_splitifs: an unquoted expansion split into a field list. */
char *rt_splitifs(char *v, char *ifs) {
    int len = rt_strlen(v);
    char *op = "\x02";
    int i = 0, s = 0, in = 0;
    while (i < len) {
        int ch = (int)(unsigned char)v[i];
        if (rt_haschar(ifs, ch) != 0) {
            /* close the field that ends here */
            if (in != 0) {
                op = rt_strcat(rt_strcat(op, rt_substr(v, s, i - s)), "\x01");
                in = 0;
            }
        } else {
            if (in == 0) {
                in = 1;
                s = i;
            }
        }
        i = i + 1;
    }
    if (in != 0) op = rt_strcat(rt_strcat(op, rt_substr(v, s, len - s)), "\x01");
    return op;
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

/* ================= frag-arr.c ================= */
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
 * store down over the hole. */
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
            /* both arms of the IR select are evaluated */
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

/* i32 rt_arr_clear(i8* nm): drop every entry of one array. */
int rt_arr_clear(char *nm) {
    int i = 0, o = 0;
    while (i < arr_n) {
        int same = rt_streq(nm, arr_nm[i]) != 0;
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

/* i32 rt_argpush(i8* v): append every FIELD of v to the argv stack. */
int rt_argpush(char *v) {
    int n = rt_nfields(v);
    int i = 0;
    while (i < n) {
        int t = argv_top;
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

/* ================= frag-io.c ================= */
/* ---- output, capture stack, subshell snapshots, local save stack --------- */

/* Under set -u, reading an unset variable raises the abort flag. */
char *rt_nounset(char *v)
{
    char *um;
    um = &unset_marker[0];
    if (opt_nounset != 0 && v == um) {
        abort_flag = 1;
    }
    return v;
}

/* One output byte - into the innermost capture buffer if a command substitution
 * is running, otherwise to stdout. Channels other than 0 are dropped. */
int rt_putc(int c)
{
    int chan;
    int d;
    char *buf;
    int l;
    chan = sink_out;
    if (chan == 0) {
        d = cap_depth;
        if (d > 0) {
            buf = cap_buf[d - 1];
            l = cap_len[d - 1];
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
    buf = rt_bump(8192);
    cap_buf[d] = buf;
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
    int i;
    d = ss_depth;
    base = d * 1024;
    i = 0;
    while (i < nvars) {
        ss_save[base + i] = gvars[i];
        i = i + 1;
    }
    ss_depth = d + 1;
    return 0;
}

/* Pop a whole variable snapshot. */
int rt_ss_restore(int u)
{
    int d;
    int base;
    int i;
    d = ss_depth - 1;
    ss_depth = d;
    base = d * 1024;
    i = 0;
    while (i < nvars) {
        gvars[i] = ss_save[base + i];
        i = i + 1;
    }
    return 0;
}

/* Remember gvars[id]'s current value on the local save stack. */
int rt_push_local(int id)
{
    int t;
    t = ls_top;
    ls_id[t] = id;
    ls_val[t] = gvars[id];
    ls_top = t + 1;
    return 0;
}

/* Restore every entry above `d` (innermost first). */
int rt_pop_locals(int d)
{
    int t;
    t = ls_top;
    while (t > d) {
        t = t - 1;
        ls_top = t;
        gvars[ls_id[t]] = ls_val[t];
        t = ls_top;
    }
    return 0;
}

/* There is no real filesystem behind a compiled run, so the file predicates go
 * through a model of the standard paths: 1 = file, 2 = directory, 0 = absent.
 * Every comparison is performed, in order; the last hit wins. */
int rt_fskind(char *p)
{
    int k;
    k = 0;
    k = rt_streq(p, "/dev/null") != 0 ? 1 : k;
    k = rt_streq(p, "/dev/zero") != 0 ? 1 : k;
    k = rt_streq(p, "/dev/tty") != 0 ? 1 : k;
    k = rt_streq(p, "/dev/stdin") != 0 ? 1 : k;
    k = rt_streq(p, "/dev/stdout") != 0 ? 1 : k;
    k = rt_streq(p, "/dev/stderr") != 0 ? 1 : k;
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


/* ================= frag-regex.c ================= */
/* ==== [[ str =~ ere ]] - the POSIX ERE engine =============================
 *
 * Ported byte-for-byte from the hand-emitted IR in languages/bash-to-llvm-ir.abnf
 * (the block that began at "---- [[ str =~ ere ]] ----").
 *
 * Contract, unchanged:
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
 *   RE_PROG  4096   instruction slots, 4 words each  -> re_prog[16384]
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

int re_prog[16384];           /* NEW GLOBAL: RE_PROG * 4 words              */
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
/* re_bad is declared EARLY in the grammar (reForL's guard reads it), but it is
 * one global like any other. */
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
    return re_prog[gpc * 4 + gk];
}

int re_set(int spc, int sk, int sv)
{
    re_prog[spc * 4 + sk] = sv;
    return 0;
}

/* append one instruction, answer its pc. An overrun raises re_bad rather than
 * writing past re_prog - the arena would have absorbed that write. */
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
 * closing ":]", or 0 when the name is unknown or the "[:" never closes. */
int re_clsname(void)
{
    int c0;
    int c1;
    int c2;
    int id;
    int d;
    int hc;
    c0 = re_at(0);
    c1 = re_at(1);
    c2 = re_at(2);
    id = 0;
    id = (c0 == 97 && c1 == 108 && c2 == 112) ? 1 : id;   /* alpha  */
    id = (c0 == 97 && c1 == 108 && c2 == 110) ? 3 : id;   /* alnum  */
    id = (c0 == 97 && c1 == 115) ? 14 : id;               /* ascii  */
    id = (c0 == 98) ? 7 : id;                             /* blank  */
    id = (c0 == 99) ? 11 : id;                            /* cntrl  */
    id = (c0 == 100) ? 2 : id;                            /* digit  */
    id = (c0 == 103) ? 10 : id;                           /* graph  */
    id = (c0 == 108) ? 5 : id;                            /* lower  */
    id = (c0 == 112 && c1 == 114) ? 9 : id;               /* print  */
    id = (c0 == 112 && c1 == 117) ? 8 : id;               /* punct  */
    id = (c0 == 115) ? 6 : id;                            /* space  */
    id = (c0 == 117) ? 4 : id;                            /* upper  */
    id = (c0 == 119) ? 13 : id;                           /* word   */
    id = (c0 == 120) ? 12 : id;                           /* xdigit */
    d = 0;
    while (1) {
        hc = re_at(d);
        if (hc == 0) {
            return 0;
        }
        if (hc == 58 && re_at(d + 1) == 93) {
            re_pos = re_pos + (d + 2);
            return id;
        }
        d = d + 1;
    }
    return 0;
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
        /* Indexed by re_bad. The text only ever reaches the stderr CHANNEL,
         * whose bytes this runtime drops, but a wrong code would still be a
         * wrong status, so they are real. */
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
