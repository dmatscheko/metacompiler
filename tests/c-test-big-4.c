/* C subset big test 4 -- string and number processing over char arrays.
 * The subset has no string literals, so every buffer is built character by
 * character from char literals and terminated with a 0. Theme: text routines
 * (length, copy, compare, reverse, palindrome, Caesar cipher, run-length
 * coding, word/vowel stats), number<->text conversion (itoa/atoi and arbitrary
 * base conversion, checked by round-tripping), and a recursive-descent
 * arithmetic evaluator (+ - * / and parentheses with correct precedence) that
 * walks a char buffer through a Parser struct. Every result is checked;
 * main() returns the failure count, so the run exits 0 on success. Identical
 * under both C engines. **/

int putchar(int c);

int nfail = 0;

int check(int got, int want) {
    if (got != want) {
        nfail++;
        putchar('F');
        putchar('0' + (nfail % 10));
        putchar('\n');
    }
    return got == want;
}

int sgn(int x) { return x < 0 ? -1 : (x > 0 ? 1 : 0); }

/* ---- basic string routines (char arrays as int arrays) ---- */

int str_len(int *s) {
    int n = 0;
    while (s[n] != 0) { n++; }
    return n;
}
int str_copy(int *dst, int *src) {
    int i = 0;
    while (src[i] != 0) { dst[i] = src[i]; i++; }
    dst[i] = 0;
    return i;
}
int str_cmp(int *a, int *b) {
    int i = 0;
    while (a[i] != 0 && a[i] == b[i]) { i++; }
    return a[i] - b[i];
}
int str_eq(int *a, int *b) { return str_cmp(a, b) == 0; }
int str_reverse(int *s) {
    int i = 0;
    int j = str_len(s) - 1;
    while (i < j) {
        int t = s[i];
        s[i] = s[j];
        s[j] = t;
        i++;
        j--;
    }
    return 0;
}
int is_palindrome(int *s) {
    int i = 0;
    int j = str_len(s) - 1;
    while (i < j) {
        if (s[i] != s[j]) { return 0; }
        i++;
        j--;
    }
    return 1;
}

/* ---- number <-> text ---- */

int reverse_buf(int *buf, int len) {        /* reverse buf[0, len) in place */
    int a = 0;
    int b = len - 1;
    while (a < b) {
        int t = buf[a];
        buf[a] = buf[b];
        buf[b] = t;
        a++;
        b--;
    }
    return 0;
}
int int_to_str(int n, int *buf) {           /* returns the length written */
    int i = 0;
    int neg = 0;
    if (n < 0) { neg = 1; n = -n; }
    if (n == 0) { buf[i] = '0'; i++; }
    while (n > 0) {
        buf[i] = '0' + (n % 10);
        i++;
        n /= 10;
    }
    if (neg) { buf[i] = '-'; i++; }
    buf[i] = 0;
    reverse_buf(buf, i);
    return i;
}
int str_to_int(int *s) {
    int i = 0;
    int sign = 1;
    if (s[0] == '-') { sign = -1; i = 1; }
    else if (s[0] == '+') { i = 1; }
    int v = 0;
    while (s[i] >= '0' && s[i] <= '9') {
        v = v * 10 + (s[i] - '0');
        i++;
    }
    return sign * v;
}

int digit_char(int d) { return d < 10 ? '0' + d : 'a' + (d - 10); }
int char_digit(int ch) {
    if (ch >= '0' && ch <= '9') { return ch - '0'; }
    if (ch >= 'a' && ch <= 'f') { return ch - 'a' + 10; }
    return -1;
}
int to_base(int n, int base, int *buf) {    /* n >= 0, base 2..16 */
    int i = 0;
    if (n == 0) { buf[i] = '0'; i++; }
    while (n > 0) {
        buf[i] = digit_char(n % base);
        i++;
        n /= base;
    }
    buf[i] = 0;
    reverse_buf(buf, i);
    return i;
}
int from_base(int *buf, int base) {
    int v = 0;
    int i = 0;
    while (buf[i] != 0) {
        v = v * base + char_digit(buf[i]);
        i++;
    }
    return v;
}

/* ---- Caesar cipher (in place, letters wrap, other chars pass through) ---- */

int caesar(int *s, int shift) {
    int i = 0;
    int m = ((shift % 26) + 26) % 26;       /* normalize the shift to 0..25 */
    while (s[i] != 0) {
        int ch = s[i];
        if (ch >= 'a' && ch <= 'z') { s[i] = 'a' + (ch - 'a' + m) % 26; }
        else if (ch >= 'A' && ch <= 'Z') { s[i] = 'A' + (ch - 'A' + m) % 26; }
        i++;
    }
    return 0;
}

/* ---- run-length coding (single-digit runs) ---- */

int rle_encode(int *src, int *dst) {
    int i = 0;
    int o = 0;
    while (src[i] != 0) {
        int ch = src[i];
        int count = 1;
        while (src[i + count] == ch && count < 9) { count++; }
        dst[o] = ch;
        o++;
        dst[o] = '0' + count;
        o++;
        i += count;
    }
    dst[o] = 0;
    return o;
}
int rle_decode(int *src, int *dst) {
    int i = 0;
    int o = 0;
    while (src[i] != 0) {
        int ch = src[i];
        int count = src[i + 1] - '0';
        int k;
        for (k = 0; k < count; k++) { dst[o] = ch; o++; }
        i += 2;
    }
    dst[o] = 0;
    return o;
}

/* ---- text statistics ---- */

int is_vowel(int ch) {
    return ch == 'a' || ch == 'e' || ch == 'i' || ch == 'o' || ch == 'u'
        || ch == 'A' || ch == 'E' || ch == 'I' || ch == 'O' || ch == 'U';
}
int count_vowels(int *s) {
    int n = 0;
    int i = 0;
    while (s[i] != 0) {
        if (is_vowel(s[i])) { n++; }
        i++;
    }
    return n;
}
int count_words(int *s) {                   /* runs of non-space characters */
    int n = 0;
    int i = 0;
    int in_word = 0;
    while (s[i] != 0) {
        if (s[i] == ' ') { in_word = 0; }
        else if (!in_word) { n++; in_word = 1; }
        i++;
    }
    return n;
}
int count_upper(int *s) {
    int n = 0;
    int i = 0;
    while (s[i] != 0) {
        if (s[i] >= 'A' && s[i] <= 'Z') { n++; }
        i++;
    }
    return n;
}

/* ---- recursive-descent arithmetic evaluator ----
 * expr   = term { ('+' | '-') term }
 * term   = factor { ('*' | '/') factor }
 * factor = number | '(' expr ')'
 * The three routines are mutually recursive (factor calls expr for a
 * parenthesized group), so the shared cursor lives in globals rather than a
 * passed pointer: that keeps the parse functions parameterless and free of the
 * forward-reference cycle. g_src points at the buffer, g_pos is the position.
 */

int *g_src;
int g_pos;

int pk(void) { return g_src[g_pos]; }               /* peek the current char */
int adv(void) { int c = g_src[g_pos]; g_pos++; return c; }  /* consume and return it */

int parse_expr(void);                       /* forward declaration */

int parse_factor(void) {
    int ch = pk();
    if (ch == '(') {
        adv();                               /* consume '(' */
        int v = parse_expr();
        adv();                               /* consume ')' */
        return v;
    }
    int v = 0;
    while (pk() >= '0' && pk() <= '9') {
        v = v * 10 + (adv() - '0');
    }
    return v;
}
int parse_term(void) {
    int v = parse_factor();
    while (1) {
        int ch = pk();
        if (ch == '*') { adv(); v = v * parse_factor(); }
        else if (ch == '/') { adv(); v = v / parse_factor(); }
        else { break; }
    }
    return v;
}
int parse_expr(void) {
    int v = parse_term();
    while (1) {
        int ch = pk();
        if (ch == '+') { adv(); v = v + parse_term(); }
        else if (ch == '-') { adv(); v = v - parse_term(); }
        else { break; }
    }
    return v;
}
int eval(int *s) {
    g_src = s;
    g_pos = 0;
    return parse_expr();
}

int main(void) {
    /* --- basic string ops: build "racecar" and "hello" --- */
    int rc[8];
    rc[0] = 'r'; rc[1] = 'a'; rc[2] = 'c'; rc[3] = 'e';
    rc[4] = 'c'; rc[5] = 'a'; rc[6] = 'r'; rc[7] = 0;
    check(str_len(rc), 7);
    check(is_palindrome(rc), 1);

    int hello[8];
    hello[0] = 'h'; hello[1] = 'e'; hello[2] = 'l'; hello[3] = 'l';
    hello[4] = 'o'; hello[5] = 0;
    check(str_len(hello), 5);
    check(is_palindrome(hello), 0);

    int copy[8];
    check(str_copy(copy, hello), 5);
    check(str_eq(copy, hello), 1);
    str_reverse(copy);                          /* "olleh" */
    check(copy[0], 'o');
    check(copy[4], 'h');
    check(str_eq(copy, hello), 0);
    str_reverse(copy);                          /* back to "hello" */
    check(str_eq(copy, hello), 1);

    /* string comparison: "apple" vs "apply" */
    int apple[8];
    int apply[8];
    apple[0] = 'a'; apple[1] = 'p'; apple[2] = 'p'; apple[3] = 'l'; apple[4] = 'e'; apple[5] = 0;
    apply[0] = 'a'; apply[1] = 'p'; apply[2] = 'p'; apply[3] = 'l'; apply[4] = 'y'; apply[5] = 0;
    check(sgn(str_cmp(apple, apply)), -1);      /* 'e' < 'y' */
    check(sgn(str_cmp(apply, apple)), 1);
    check(str_cmp(apple, apple), 0);

    /* --- itoa / atoi round trips --- */
    int nbuf[16];
    check(int_to_str(0, nbuf), 1);
    check(nbuf[0], '0');
    check(int_to_str(12345, nbuf), 5);
    check(str_to_int(nbuf), 12345);
    check(int_to_str(-678, nbuf), 4);
    check(nbuf[0], '-');
    check(str_to_int(nbuf), -678);

    int values[6];
    values[0] = 0; values[1] = 7; values[2] = -42;
    values[3] = 1000; values[4] = -999999; values[5] = 2147483; /* stays well inside int */
    int vi;
    int roundtrip_ok = 1;
    for (vi = 0; vi < 6; vi++) {
        int_to_str(values[vi], nbuf);
        if (str_to_int(nbuf) != values[vi]) { roundtrip_ok = 0; }
    }
    check(roundtrip_ok, 1);

    /* --- base conversion --- */
    int bbuf[40];
    check(to_base(255, 16, bbuf), 2);
    check(bbuf[0], 'f');
    check(bbuf[1], 'f');
    check(from_base(bbuf, 16), 255);

    check(to_base(10, 2, bbuf), 4);             /* "1010" */
    check(bbuf[0], '1');
    check(bbuf[1], '0');
    check(bbuf[2], '1');
    check(bbuf[3], '0');
    check(from_base(bbuf, 2), 10);

    to_base(0, 2, bbuf);
    check(bbuf[0], '0');
    check(from_base(bbuf, 2), 0);

    /* decimal -> base 7 -> back, over a range */
    int base_ok = 1;
    int bn;
    for (bn = 0; bn <= 200; bn++) {
        to_base(bn, 7, bbuf);
        if (from_base(bbuf, 7) != bn) { base_ok = 0; }
    }
    check(base_ok, 1);

    /* --- Caesar cipher round trip and a known shift --- */
    int msg[16];
    msg[0] = 'A'; msg[1] = 'b'; msg[2] = 'c'; msg[3] = 'X'; msg[4] = 'y'; msg[5] = 'z'; msg[6] = 0;
    int orig[16];
    str_copy(orig, msg);
    caesar(msg, 3);                             /* A->D, b->e, c->f, X->A, y->b, z->c */
    check(msg[0], 'D');
    check(msg[1], 'e');
    check(msg[3], 'A');                         /* wraps around */
    check(msg[5], 'c');
    caesar(msg, -3);                            /* decode restores the original */
    check(str_eq(msg, orig), 1);
    caesar(msg, 29);                            /* 29 == shift of 3 (mod 26) */
    check(msg[0], 'D');
    caesar(msg, 23);                            /* 3 + 23 == 26 -> identity */
    check(str_eq(msg, orig), 1);

    /* --- run-length coding: "aaabbc" --- */
    int raw[16];
    raw[0] = 'a'; raw[1] = 'a'; raw[2] = 'a'; raw[3] = 'b'; raw[4] = 'b'; raw[5] = 'c'; raw[6] = 0;
    int enc[16];
    int dec[16];
    check(rle_encode(raw, enc), 6);             /* "a3b2c1" */
    check(enc[0], 'a');
    check(enc[1], '3');
    check(enc[2], 'b');
    check(enc[3], '2');
    check(enc[4], 'c');
    check(enc[5], '1');
    rle_decode(enc, dec);
    check(str_eq(dec, raw), 1);                 /* decode reverses encode */

    /* --- text statistics: "Hi There Bob" --- */
    int sentence[24];
    sentence[0] = 'H'; sentence[1] = 'i'; sentence[2] = ' ';
    sentence[3] = 'T'; sentence[4] = 'h'; sentence[5] = 'e'; sentence[6] = 'r'; sentence[7] = 'e'; sentence[8] = ' ';
    sentence[9] = 'B'; sentence[10] = 'o'; sentence[11] = 'b'; sentence[12] = 0;
    check(count_words(sentence), 3);
    check(count_upper(sentence), 3);            /* H, T, B */
    check(count_vowels(sentence), 4);           /* i, e, e, o */
    check(str_len(sentence), 12);

    /* --- recursive-descent evaluator --- */
    int e1[16];
    e1[0] = '2'; e1[1] = '+'; e1[2] = '3'; e1[3] = '*'; e1[4] = '4'; e1[5] = 0;
    check(eval(e1), 14);                        /* precedence: 2 + (3*4) */

    int e2[16];
    e2[0] = '('; e2[1] = '2'; e2[2] = '+'; e2[3] = '3'; e2[4] = ')'; e2[5] = '*'; e2[6] = '4'; e2[7] = 0;
    check(eval(e2), 20);                        /* parentheses override precedence */

    int e3[16];
    e3[0] = '1'; e3[1] = '0'; e3[2] = '+'; e3[3] = '2'; e3[4] = '0';
    e3[5] = '/'; e3[6] = '5'; e3[7] = '-'; e3[8] = '3'; e3[9] = 0;
    check(eval(e3), 11);                        /* 10 + (20/5) - 3 */

    int e4[16];
    e4[0] = '1'; e4[1] = '0'; e4[2] = '0'; e4[3] = '-'; e4[4] = '5';
    e4[5] = '0'; e4[6] = '-'; e4[7] = '2'; e4[8] = '5'; e4[9] = 0;
    check(eval(e4), 25);                        /* left associative subtraction */

    int e5[20];
    e5[0] = '('; e5[1] = '('; e5[2] = '1'; e5[3] = '+'; e5[4] = '2'; e5[5] = ')';
    e5[6] = '*'; e5[7] = '('; e5[8] = '3'; e5[9] = '+'; e5[10] = '4'; e5[11] = ')'; e5[12] = ')'; e5[13] = 0;
    check(eval(e5), 21);                        /* (1+2) * (3+4) */

    int e6[24];                                 /* multi-digit and nesting */
    e6[0] = '2'; e6[1] = '*'; e6[2] = '('; e6[3] = '1'; e6[4] = '0'; e6[5] = '+';
    e6[6] = '5'; e6[7] = ')'; e6[8] = '-'; e6[9] = '1'; e6[10] = '2'; e6[11] = '/';
    e6[12] = '4'; e6[13] = 0;
    check(eval(e6), 27);                        /* 2*(15) - (12/4) = 30 - 3 */

    if (nfail == 0) {
        putchar('O'); putchar('K'); putchar('\n');
    }
    return nfail;
}
