/* Test file for the C99 grammar (tests/c99-parser.abnf).
 * It deliberately contains the constructs that are hard to parse. **/

#include <stdio.h>
#include <stdlib.h>
#define MAX(a, b) ((a) > (b) ? (a) : (b))
#define LONG_MACRO(x) do { \
        (void)(x);         \
    } while (0)

typedef unsigned long size_type;
typedef struct Node Node;

struct Node {
    int value;
    unsigned flags : 4;
    unsigned : 4;
    struct Node *next;
    union {
        double d;
        char bytes[8];
    } payload;
};

enum Color { RED, GREEN = 2, BLUE, };

static const char *messages[] = {
    "hello \"world\"\n",
    "tab\there, hex \x41, octal \101, unicode ä",
    "adjacent " "strings " "concatenate",
};

static double doubled = 1.0;   /* identifier starts with keyword "double" */
static int interned = 2;       /* identifier starts with keyword "int" */
int format = 3;                /* identifier starts with keyword "for" */

typedef int (*BinaryOp)(int, int);

static int add(int a, int b) { return a + b; }

int apply(BinaryOp op, int a, int b)
{
    return op(a, b);
}

size_type count_nodes(const Node *head)
{
    size_type n = 0;
    for (const Node *p = head; p != NULL; p = p->next) {
        n++;
    }
    return n;
}

int main(int argc, char **argv)
{
    Node *list = NULL;          /* typedef-name led pointer declaration */
    BinaryOp fn = &add;         /* address of function */
    size_type total = 0;
    int matrix[2][3] = { { 1, 2, 3 }, { 4, 5, 6 } };
    struct Node n1 = { .value = 41, .flags = 3, .payload = { .d = 0.5 } };
    int sparse[8] = { [0] = 1, [7] = 2 };
    long big = 0x7fffffffL;
    unsigned u = 42u;
    float f = 1.5e-3f;
    double hexfloat = 0x1.8p3;
    char c = '\n';
    char wide_ish = 'x';

    (void)argv;
    free(malloc(16));           /* call with call argument, must stay a statement */

    if (argc > 1 && interned != 0) {
        total += (size_type)argc;      /* cast to typedef name */
    } else if (!format) {
        total = MAX(u, 7);
    }

    while (total < 10) {
        total <<= 1;
        total |= 1;
    }

    do {
        total--;
    } while (total > 12);

    switch (argc) {
    case 1:
        total += sizeof(int) * 2;
        break;
    case 2:
        total += sizeof total;
        break;
    default:
        total = doubled > 0.5 ? total : 0;
        break;
    }

    for (u = 0; u < 3; u++) {
        matrix[1][u] += apply(fn, (int)u, sparse[u]);
    }

    n1.next = list;
    list = &n1;

    if (count_nodes(list) != 1) {
        goto fail;
    }

    printf("%s %ld %f %f %c%c\n", messages[0], big, f, hexfloat, c, wide_ish);
    printf("colors: %d %d %d\n", RED, GREEN, BLUE);
    return 0;

fail:
    return 1;
}
