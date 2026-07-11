/* C subset big test 3 -- structs and pointer graphs (C's "objects").
 * Theme: modeling entities with structs and mutating them through pointers.
 * Integer 2D geometry (points, rectangles, a polygon area by the shoelace
 * formula), a rational-number ADT that normalizes itself via gcd, a matrix ADT
 * backed by an int* with 2D index math, and a binary search tree built from a
 * node pool with real left/right pointer links (insert, search, in-order walk,
 * height). Every result is checked; main() returns the failure count, so the
 * run exits 0 on success. Identical under both C engines. **/

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

int iabs(int x) { return x < 0 ? -x : x; }
int imax(int a, int b) { return a > b ? a : b; }
int imin(int a, int b) { return a < b ? a : b; }
int igcd(int a, int b) {
    a = iabs(a);
    b = iabs(b);
    while (b != 0) {
        int t = a % b;
        a = b;
        b = t;
    }
    return a;
}

/* ---- 2D points as little value objects reached through pointers ---- */

struct Point { int x; int y; };

int pt_set(struct Point *p, int x, int y) { p->x = x; p->y = y; return 0; }
int pt_dot(struct Point *a, struct Point *b) { return a->x * b->x + a->y * b->y; }
int pt_cross(struct Point *a, struct Point *b) { return a->x * b->y - a->y * b->x; }
int pt_manhattan(struct Point *a, struct Point *b) {
    return iabs(a->x - b->x) + iabs(a->y - b->y);
}
int pt_eq(struct Point *a, struct Point *b) { return a->x == b->x && a->y == b->y; }

/* twice the signed polygon area over an array of points (shoelace) */
int shoelace2(struct Point *pts, int n) {
    int s = 0;
    int i;
    for (i = 0; i < n; i++) {
        int j = (i + 1) % n;
        s += pts[i].x * pts[j].y - pts[j].x * pts[i].y;
    }
    return s;
}

/* ---- rectangles with nested struct-value fields ---- */

struct Rect { struct Point lo; struct Point hi; };

int rect_area(struct Rect *r) {
    return (r->hi.x - r->lo.x) * (r->hi.y - r->lo.y);
}
int rect_contains(struct Rect *r, int x, int y) {
    return x >= r->lo.x && x <= r->hi.x && y >= r->lo.y && y <= r->hi.y;
}
int span_overlap(int alo, int ahi, int blo, int bhi) {
    int lo = imax(alo, blo);
    int hi = imin(ahi, bhi);
    return hi > lo ? hi - lo : 0;
}
int rect_intersect_area(struct Rect *a, struct Rect *b) {
    return span_overlap(a->lo.x, a->hi.x, b->lo.x, b->hi.x)
         * span_overlap(a->lo.y, a->hi.y, b->lo.y, b->hi.y);
}

/* ---- rational numbers that normalize themselves ---- */

struct Frac { int num; int den; };

int frac_set(struct Frac *f, int num, int den) { f->num = num; f->den = den; return 0; }
int frac_norm(struct Frac *f) {
    if (f->den < 0) { f->num = -f->num; f->den = -f->den; }
    int g = igcd(f->num, f->den);
    if (g != 0) { f->num = f->num / g; f->den = f->den / g; }
    return 0;
}
int frac_add(struct Frac *a, struct Frac *b, struct Frac *out) {
    out->num = a->num * b->den + b->num * a->den;
    out->den = a->den * b->den;
    frac_norm(out);
    return 0;
}
int frac_mul(struct Frac *a, struct Frac *b, struct Frac *out) {
    out->num = a->num * b->num;
    out->den = a->den * b->den;
    frac_norm(out);
    return 0;
}
int frac_eq(struct Frac *a, struct Frac *b) {       /* cross-multiply */
    return a->num * b->den == b->num * a->den;
}

/* ---- matrix ADT backed by an int* with row-major 2D indexing ---- */

struct Matrix { int *cells; int rows; int cols; };

int mat_init(struct Matrix *m, int *buf, int rows, int cols) {
    m->cells = buf;
    m->rows = rows;
    m->cols = cols;
    return 0;
}
int mat_get(struct Matrix *m, int r, int c) { return m->cells[r * m->cols + c]; }
int mat_set(struct Matrix *m, int r, int c, int v) { m->cells[r * m->cols + c] = v; return 0; }
int mat_identity(struct Matrix *m) {
    int i;
    int j;
    for (i = 0; i < m->rows; i++) {
        for (j = 0; j < m->cols; j++) {
            mat_set(m, i, j, i == j ? 1 : 0);
        }
    }
    return 0;
}
int mat_mul(struct Matrix *a, struct Matrix *b, struct Matrix *out) {
    int i;
    int j;
    int k;
    for (i = 0; i < a->rows; i++) {
        for (j = 0; j < b->cols; j++) {
            int s = 0;
            for (k = 0; k < a->cols; k++) {
                s += mat_get(a, i, k) * mat_get(b, k, j);
            }
            mat_set(out, i, j, s);
        }
    }
    return 0;
}
int mat_trace(struct Matrix *m) {
    int s = 0;
    int i;
    for (i = 0; i < m->rows; i++) { s += mat_get(m, i, i); }
    return s;
}
int mat_eq(struct Matrix *a, struct Matrix *b) {
    if (a->rows != b->rows || a->cols != b->cols) { return 0; }
    int i;
    int j;
    for (i = 0; i < a->rows; i++) {
        for (j = 0; j < a->cols; j++) {
            if (mat_get(a, i, j) != mat_get(b, i, j)) { return 0; }
        }
    }
    return 1;
}

/* ---- binary search tree from a node pool with real pointer links ---- */

struct TNode { int key; struct TNode *left; struct TNode *right; };
struct Tree { struct TNode *root; };

struct TNode tpool[64];
int tpool_top = 0;

int tree_new_node(int key) {
    int idx = tpool_top;
    tpool_top++;
    tpool[idx].key = key;
    tpool[idx].left = 0;
    tpool[idx].right = 0;
    return idx;
}
int tree_insert(struct Tree *t, int key) {
    int idx = tree_new_node(key);
    if (t->root == 0) { t->root = &tpool[idx]; return 0; }
    struct TNode *cur = t->root;
    while (1) {
        if (key < cur->key) {
            if (cur->left == 0) { cur->left = &tpool[idx]; return 0; }
            cur = cur->left;
        } else {
            if (cur->right == 0) { cur->right = &tpool[idx]; return 0; }
            cur = cur->right;
        }
    }
}
int tree_search(struct Tree *t, int key) {
    struct TNode *cur = t->root;
    while (cur != 0) {
        if (key == cur->key) { return 1; }
        if (key < cur->key) { cur = cur->left; }
        else { cur = cur->right; }
    }
    return 0;
}
int tree_min(struct TNode *node) {          /* leftmost key */
    while (node->left != 0) { node = node->left; }
    return node->key;
}
int tree_max(struct TNode *node) {
    while (node->right != 0) { node = node->right; }
    return node->key;
}
int tree_height(struct TNode *node) {
    if (node == 0) { return 0; }
    return 1 + imax(tree_height(node->left), tree_height(node->right));
}
int tree_count(struct TNode *node) {
    if (node == 0) { return 0; }
    return 1 + tree_count(node->left) + tree_count(node->right);
}
int inorder(struct TNode *node, int *out, int idx) {    /* fills out[], returns next index */
    if (node == 0) { return idx; }
    idx = inorder(node->left, out, idx);
    out[idx] = node->key;
    idx++;
    idx = inorder(node->right, out, idx);
    return idx;
}

int is_sorted(int *a, int n) {
    int i;
    for (i = 1; i < n; i++) {
        if (a[i - 1] > a[i]) { return 0; }
    }
    return 1;
}

int main(void) {
    int i;

    /* --- points --- */
    struct Point a;
    struct Point b;
    pt_set(&a, 3, 4);
    pt_set(&b, 1, 2);
    check(pt_dot(&a, &b), 3 * 1 + 4 * 2);           /* 11 */
    check(pt_cross(&a, &b), 3 * 2 - 4 * 1);         /* 2 */
    check(pt_manhattan(&a, &b), 2 + 2);             /* 4 */
    check(pt_eq(&a, &b), 0);
    struct Point c;
    pt_set(&c, 3, 4);
    check(pt_eq(&a, &c), 1);

    /* --- polygon area via shoelace over a global struct array --- */
    struct Point square[4];
    pt_set(&square[0], 0, 0);
    pt_set(&square[1], 4, 0);
    pt_set(&square[2], 4, 3);
    pt_set(&square[3], 0, 3);
    check(shoelace2(square, 4), 24);                /* 2 * area, area = 12 */

    struct Point tri[3];
    pt_set(&tri[0], 0, 0);
    pt_set(&tri[1], 4, 0);
    pt_set(&tri[2], 0, 6);
    check(shoelace2(tri, 3), 24);                   /* area 12 */

    /* --- rectangles with nested struct values --- */
    struct Rect r1;
    r1.lo.x = 0; r1.lo.y = 0;
    r1.hi.x = 4; r1.hi.y = 5;
    check(rect_area(&r1), 20);
    check(rect_contains(&r1, 2, 3), 1);
    check(rect_contains(&r1, 5, 3), 0);
    check(rect_contains(&r1, 0, 0), 1);             /* on the boundary */

    struct Rect r2;
    r2.lo.x = 2; r2.lo.y = 1;
    r2.hi.x = 6; r2.hi.y = 4;
    check(rect_intersect_area(&r1, &r2), 6);        /* x-overlap 2, y-overlap 3 */
    struct Rect r3;
    r3.lo.x = 10; r3.lo.y = 10;
    r3.hi.x = 12; r3.hi.y = 12;
    check(rect_intersect_area(&r1, &r3), 0);        /* disjoint */

    /* --- rational arithmetic --- */
    struct Frac half;
    struct Frac third;
    frac_set(&half, 1, 2);
    frac_set(&third, 1, 3);
    struct Frac sum;
    frac_add(&half, &third, &sum);
    check(sum.num, 5);
    check(sum.den, 6);                              /* 1/2 + 1/3 = 5/6 */

    struct Frac prod;
    frac_mul(&half, &third, &prod);
    check(prod.num, 1);
    check(prod.den, 6);                             /* 1/2 * 1/3 = 1/6 */

    struct Frac two_quarters;
    frac_set(&two_quarters, 2, 4);
    frac_norm(&two_quarters);
    check(two_quarters.num, 1);
    check(two_quarters.den, 2);                     /* normalizes to 1/2 */
    check(frac_eq(&two_quarters, &half), 1);

    struct Frac neg;
    frac_set(&neg, 3, -6);                          /* sign moves to the numerator */
    frac_norm(&neg);
    check(neg.num, -1);
    check(neg.den, 2);

    struct Frac wsum;                               /* 1/2 + 1/2 = 1/1 */
    frac_add(&half, &half, &wsum);
    check(wsum.num, 1);
    check(wsum.den, 1);

    /* --- matrices --- */
    int abuf[4];
    int bbuf[4];
    int cbuf[4];
    struct Matrix ma;
    struct Matrix mb;
    struct Matrix mc;
    mat_init(&ma, abuf, 2, 2);
    mat_init(&mb, bbuf, 2, 2);
    mat_init(&mc, cbuf, 2, 2);
    /* ma = [[1,2],[3,4]] */
    mat_set(&ma, 0, 0, 1); mat_set(&ma, 0, 1, 2);
    mat_set(&ma, 1, 0, 3); mat_set(&ma, 1, 1, 4);
    check(mat_get(&ma, 1, 0), 3);
    check(mat_trace(&ma), 5);

    mat_identity(&mb);
    check(mat_get(&mb, 0, 0), 1);
    check(mat_get(&mb, 0, 1), 0);
    mat_mul(&ma, &mb, &mc);                         /* A * I == A */
    check(mat_eq(&mc, &ma), 1);

    /* mb = [[0,1],[1,0]] swap columns when multiplied on the right */
    mat_set(&mb, 0, 0, 0); mat_set(&mb, 0, 1, 1);
    mat_set(&mb, 1, 0, 1); mat_set(&mb, 1, 1, 0);
    mat_mul(&ma, &mb, &mc);                         /* [[2,1],[4,3]] */
    check(mat_get(&mc, 0, 0), 2);
    check(mat_get(&mc, 0, 1), 1);
    check(mat_get(&mc, 1, 0), 4);
    check(mat_get(&mc, 1, 1), 3);

    /* a 3x3 times identity, checked via trace preservation */
    int dbuf[9];
    int ibuf[9];
    int obuf[9];
    struct Matrix md;
    struct Matrix mi;
    struct Matrix mo;
    mat_init(&md, dbuf, 3, 3);
    mat_init(&mi, ibuf, 3, 3);
    mat_init(&mo, obuf, 3, 3);
    int r;
    int col;
    int fillv = 1;
    for (r = 0; r < 3; r++) {
        for (col = 0; col < 3; col++) {
            mat_set(&md, r, col, fillv);
            fillv++;
        }
    }
    mat_identity(&mi);
    mat_mul(&md, &mi, &mo);
    check(mat_eq(&mo, &md), 1);
    check(mat_trace(&md), 1 + 5 + 9);               /* diagonal 1,5,9 */

    /* --- binary search tree --- */
    struct Tree tree;
    tree.root = 0;
    int keys[9];
    keys[0] = 50; keys[1] = 30; keys[2] = 70; keys[3] = 20; keys[4] = 40;
    keys[5] = 60; keys[6] = 80; keys[7] = 35; keys[8] = 65;
    for (i = 0; i < 9; i++) { tree_insert(&tree, keys[i]); }

    check(tree_count(tree.root), 9);
    check(tree_search(&tree, 35), 1);
    check(tree_search(&tree, 65), 1);
    check(tree_search(&tree, 99), 0);
    check(tree_search(&tree, 50), 1);               /* the root */
    check(tree_min(tree.root), 20);
    check(tree_max(tree.root), 80);
    check(tree_height(tree.root), 4);               /* levels: 50 / 30,70 / 20,40,60,80 / 35,65 */

    int walk[9];
    int filled = inorder(tree.root, walk, 0);
    check(filled, 9);
    check(is_sorted(walk, 9), 1);                   /* in-order of a BST is sorted */
    check(walk[0], 20);
    check(walk[8], 80);
    check(walk[4], 50);                             /* median in sorted order */

    if (nfail == 0) {
        putchar('O'); putchar('K'); putchar('\n');
    }
    return nfail;
}
