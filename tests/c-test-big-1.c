/* C subset big test 1 -- data structures and algorithms.
 * Theme: classic in-place algorithms over int arrays plus abstract data types
 * (stack, circular queue, binary min-heap) built as structs holding an int*
 * into a backing array, and a singly linked list built from a node pool with
 * real pointer reversal. Every result is checked against a known-good value;
 * main() returns the number of failed checks, so the run exits 0 on success.
 * Runs identically under c-interpreter.abnf and c-to-llvm-ir.abnf. **/

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

/* ---- low level helpers ---- */

int swap(int *x, int *y) {          /* pointer swap */
    int t = *x;
    *x = *y;
    *y = t;
    return 0;
}

int aswap(int *a, int i, int j) {   /* swap two array cells through a pointer */
    int t = a[i];
    a[i] = a[j];
    a[j] = t;
    return 0;
}

int is_sorted(int *a, int n) {
    int i;
    for (i = 1; i < n; i++) {
        if (a[i - 1] > a[i]) { return 0; }
    }
    return 1;
}

int array_eq(int *a, int *b, int n) {
    int i;
    for (i = 0; i < n; i++) {
        if (a[i] != b[i]) { return 0; }
    }
    return 1;
}

int array_sum(int *a, int n) {
    int s = 0;
    int i;
    for (i = 0; i < n; i++) { s += a[i]; }
    return s;
}

/* ---- in-place sorts ---- */

int bubble_sort(int *a, int n) {
    int i;
    int j;
    for (i = 0; i < n - 1; i++) {
        int swapped = 0;
        for (j = 0; j < n - 1 - i; j++) {
            if (a[j] > a[j + 1]) { aswap(a, j, j + 1); swapped = 1; }
        }
        if (!swapped) { break; }        /* early exit when already sorted */
    }
    return 0;
}

int selection_sort(int *a, int n) {
    int i;
    int j;
    for (i = 0; i < n - 1; i++) {
        int m = i;
        for (j = i + 1; j < n; j++) {
            if (a[j] < a[m]) { m = j; }
        }
        if (m != i) { aswap(a, i, m); }
    }
    return 0;
}

int insertion_sort(int *a, int n) {
    int i;
    for (i = 1; i < n; i++) {
        int key = a[i];
        int j = i - 1;
        while (j >= 0 && a[j] > key) {
            a[j + 1] = a[j];
            j--;
        }
        a[j + 1] = key;
    }
    return 0;
}

int partition(int *a, int lo, int hi) {     /* Lomuto partition */
    int pivot = a[hi];
    int i = lo - 1;
    int j;
    for (j = lo; j < hi; j++) {
        if (a[j] <= pivot) {
            i++;
            aswap(a, i, j);
        }
    }
    aswap(a, i + 1, hi);
    return i + 1;
}

int quicksort(int *a, int lo, int hi) {     /* recursive */
    if (lo < hi) {
        int p = partition(a, lo, hi);
        quicksort(a, lo, p - 1);
        quicksort(a, p + 1, hi);
    }
    return 0;
}

/* ---- searching ---- */

int binary_search(int *a, int n, int key) { /* iterative, a is sorted */
    int lo = 0;
    int hi = n - 1;
    while (lo <= hi) {
        int mid = lo + (hi - lo) / 2;
        if (a[mid] == key) { return mid; }
        if (a[mid] < key) { lo = mid + 1; }
        else { hi = mid - 1; }
    }
    return -1;
}

int bsearch_rec(int *a, int lo, int hi, int key) {
    if (lo > hi) { return -1; }
    int mid = lo + (hi - lo) / 2;
    if (a[mid] == key) { return mid; }
    if (a[mid] < key) { return bsearch_rec(a, mid + 1, hi, key); }
    return bsearch_rec(a, lo, mid - 1, key);
}

/* ---- array transforms ---- */

int reverse_array(int *a, int n) {          /* two indices walking inward */
    int lo = 0;
    int hi = n - 1;
    while (lo < hi) {
        aswap(a, lo, hi);
        lo++;
        hi--;
    }
    return 0;
}

int rotate_left(int *a, int n, int k) {     /* rotate via triple reverse */
    k = k % n;
    reverse_array(a, k);
    reverse_array(a + k, n - k);
    reverse_array(a, n);
    return 0;
}

int merge_sorted(int *a, int na, int *b, int nb, int *out) {
    int i = 0;
    int j = 0;
    int o = 0;
    while (i < na && j < nb) {
        if (a[i] <= b[j]) { out[o++] = a[i++]; }
        else { out[o++] = b[j++]; }
    }
    while (i < na) { out[o++] = a[i++]; }
    while (j < nb) { out[o++] = b[j++]; }
    return o;
}

/* ---- stack (struct with int* into a backing array) ---- */

struct Stack { int *data; int top; int cap; };

int stack_init(struct Stack *s, int *buf, int cap) {
    s->data = buf;
    s->top = 0;
    s->cap = cap;
    return 0;
}
int stack_push(struct Stack *s, int v) {
    if (s->top >= s->cap) { return 0; }
    s->data[s->top] = v;
    s->top++;
    return 1;
}
int stack_pop(struct Stack *s) {
    s->top--;
    return s->data[s->top];
}
int stack_empty(struct Stack *s) { return s->top == 0; }

/* ---- circular queue ---- */

struct Queue { int *data; int head; int count; int cap; };

int queue_init(struct Queue *q, int *buf, int cap) {
    q->data = buf;
    q->head = 0;
    q->count = 0;
    q->cap = cap;
    return 0;
}
int queue_push(struct Queue *q, int v) {
    if (q->count >= q->cap) { return 0; }
    int idx = (q->head + q->count) % q->cap;
    q->data[idx] = v;
    q->count++;
    return 1;
}
int queue_pop(struct Queue *q) {
    int v = q->data[q->head];
    q->head = (q->head + 1) % q->cap;
    q->count--;
    return v;
}

/* ---- binary min-heap ---- */

struct Heap { int *data; int size; };

int heap_push(struct Heap *h, int v) {
    int i = h->size;
    h->data[i] = v;
    h->size++;
    while (i > 0) {
        int parent = (i - 1) / 2;
        if (h->data[parent] <= h->data[i]) { break; }
        aswap(h->data, parent, i);
        i = parent;
    }
    return 0;
}
int heap_pop_min(struct Heap *h) {
    int top = h->data[0];
    h->size--;
    h->data[0] = h->data[h->size];
    int i = 0;
    while (1) {
        int l = 2 * i + 1;
        int r = 2 * i + 2;
        int smallest = i;
        if (l < h->size && h->data[l] < h->data[smallest]) { smallest = l; }
        if (r < h->size && h->data[r] < h->data[smallest]) { smallest = r; }
        if (smallest == i) { break; }
        aswap(h->data, i, smallest);
        i = smallest;
    }
    return top;
}

/* ---- singly linked list from a node pool, with pointer reversal ---- */

struct Node { int val; struct Node *next; };
struct List { struct Node *head; };

struct Node pool[64];
int pool_top = 0;

int new_node(int v) {           /* returns the pool index of a fresh node */
    int idx = pool_top;
    pool[idx].val = v;
    pool[idx].next = 0;
    pool_top++;
    return idx;
}
int list_prepend(struct List *lst, int v) {
    int idx = new_node(v);
    pool[idx].next = lst->head;
    lst->head = &pool[idx];
    return 0;
}
int list_length(struct List *lst) {
    int n = 0;
    struct Node *cur = lst->head;
    while (cur != 0) { n++; cur = cur->next; }
    return n;
}
int list_sum(struct List *lst) {
    int s = 0;
    struct Node *cur = lst->head;
    while (cur != 0) { s += cur->val; cur = cur->next; }
    return s;
}
int list_reverse(struct List *lst) {
    struct Node *prev = 0;
    struct Node *cur = lst->head;
    while (cur != 0) {
        struct Node *nxt = cur->next;
        cur->next = prev;
        prev = cur;
        cur = nxt;
    }
    lst->head = prev;
    return 0;
}
int list_nth(struct List *lst, int k) {     /* value at position k */
    struct Node *cur = lst->head;
    while (k > 0 && cur != 0) { cur = cur->next; k--; }
    return cur->val;
}

int main(void) {
    int i;

    /* --- sorting --- */
    int base[8];
    int seed = 5;
    for (i = 0; i < 8; i++) { seed = (seed * 7 + 3) % 97; base[i] = seed; }

    int s1[8];
    for (i = 0; i < 8; i++) { s1[i] = base[i]; }
    bubble_sort(s1, 8);
    check(is_sorted(s1, 8), 1);

    int s2[8];
    for (i = 0; i < 8; i++) { s2[i] = base[i]; }
    selection_sort(s2, 8);
    check(is_sorted(s2, 8), 1);
    check(array_eq(s1, s2, 8), 1);

    int s3[8];
    for (i = 0; i < 8; i++) { s3[i] = base[i]; }
    insertion_sort(s3, 8);
    check(array_eq(s1, s3, 8), 1);

    int s4[8];
    for (i = 0; i < 8; i++) { s4[i] = base[i]; }
    quicksort(s4, 0, 7);
    check(array_eq(s1, s4, 8), 1);
    check(array_sum(s4, 8), array_sum(base, 8));    /* sorting preserves the sum */

    /* --- searching in the sorted array --- */
    check(binary_search(s1, 8, s1[0]), 0);
    check(binary_search(s1, 8, s1[7]), 7);
    check(binary_search(s1, 8, s1[4]), 4);
    check(binary_search(s1, 8, -1), -1);
    check(bsearch_rec(s1, 0, 7, s1[3]), 3);
    check(bsearch_rec(s1, 0, 7, 1000), -1);

    /* --- reverse / rotate --- */
    int r[6];
    for (i = 0; i < 6; i++) { r[i] = i + 1; }   /* 1 2 3 4 5 6 */
    reverse_array(r, 6);
    check(r[0], 6);
    check(r[5], 1);
    reverse_array(r, 6);                          /* back to 1..6 */
    rotate_left(r, 6, 2);                         /* 3 4 5 6 1 2 */
    check(r[0], 3);
    check(r[4], 1);
    check(r[5], 2);
    check(array_sum(r, 6), 21);

    /* --- merge two sorted runs --- */
    int ma[3];
    int mb[4];
    ma[0] = 1; ma[1] = 4; ma[2] = 9;
    mb[0] = 2; mb[1] = 3; mb[2] = 5; mb[3] = 10;
    int mo[7];
    check(merge_sorted(ma, 3, mb, 4, mo), 7);
    check(is_sorted(mo, 7), 1);
    check(mo[0], 1);
    check(mo[6], 10);

    /* --- stack --- */
    int sbuf[16];
    struct Stack st;
    stack_init(&st, sbuf, 16);
    check(stack_empty(&st), 1);
    for (i = 1; i <= 5; i++) { stack_push(&st, i * 10); }
    check(stack_empty(&st), 0);
    check(stack_pop(&st), 50);              /* LIFO */
    check(stack_pop(&st), 40);
    stack_push(&st, 99);
    check(stack_pop(&st), 99);
    check(stack_pop(&st), 30);

    /* --- circular queue: wrap around by pushing past a pop --- */
    int qbuf[4];
    struct Queue q;
    queue_init(&q, qbuf, 4);
    queue_push(&q, 1);
    queue_push(&q, 2);
    queue_push(&q, 3);
    check(queue_pop(&q), 1);                 /* FIFO */
    check(queue_pop(&q), 2);
    queue_push(&q, 4);
    queue_push(&q, 5);                        /* wraps into a freed slot */
    check(queue_push(&q, 6), 1);             /* now count == cap */
    check(queue_push(&q, 7), 0);             /* full: rejected */
    check(queue_pop(&q), 3);
    check(queue_pop(&q), 4);
    check(queue_pop(&q), 5);
    check(queue_pop(&q), 6);

    /* --- min-heap: pushing an unsorted stream then popping yields sorted --- */
    int hbuf[16];
    struct Heap h;
    h.data = hbuf;
    h.size = 0;
    int feed[7];
    feed[0] = 7; feed[1] = 3; feed[2] = 9; feed[3] = 1; feed[4] = 5; feed[5] = 8; feed[6] = 2;
    for (i = 0; i < 7; i++) { heap_push(&h, feed[i]); }
    check(h.size, 7);
    int prev = -1;
    int ok = 1;
    for (i = 0; i < 7; i++) {
        int m = heap_pop_min(&h);
        if (m < prev) { ok = 0; }
        prev = m;
    }
    check(ok, 1);                            /* popped in nondecreasing order */
    check(prev, 9);                          /* last (largest) */
    check(h.size, 0);

    /* --- linked list with pointer reversal --- */
    struct List lst;
    lst.head = 0;
    for (i = 1; i <= 5; i++) { list_prepend(&lst, i); }   /* head..tail = 5 4 3 2 1 */
    check(list_length(&lst), 5);
    check(list_sum(&lst), 15);
    check(list_nth(&lst, 0), 5);
    check(list_nth(&lst, 4), 1);
    list_reverse(&lst);                        /* now 1 2 3 4 5 */
    check(list_nth(&lst, 0), 1);
    check(list_nth(&lst, 4), 5);
    check(list_sum(&lst), 15);                 /* reversal preserves the sum */

    if (nfail == 0) {
        putchar('O'); putchar('K'); putchar('\n');
    }
    return nfail;
}
