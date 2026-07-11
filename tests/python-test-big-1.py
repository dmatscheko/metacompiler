# Python subset self test - BIG 1: data structures and sorting algorithms.
#
# Theme: classic algorithms over lists and dict-backed structures - five sorting
# routines cross-checked against one another, binary search, a stack, an in-place
# queue, a singly linked list built from dicts, and matrix arithmetic with nested
# comprehensions. The file runs top to bottom and ends with exit(fails[0]), so the
# run exits 0 exactly when every check passes; the interpreter and the LLVM-IR
# compiler (under both engines) must agree byte for byte.

fails = [0]


def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1


def check_true(name, got):
    if not got:
        print("FAIL", name, "expected a true value")
        fails[0] += 1


# ----- list helpers (list + list does not concatenate in this subset) -----

def concat(a, b):
    out = []
    for x in a:
        out.append(x)
    for x in b:
        out.append(x)
    return out


def copy_list(a):
    out = []
    for x in a:
        out.append(x)
    return out


def render(a):
    return f"{a}"


def is_sorted(a):
    for i in range(len(a) - 1):
        if a[i] > a[i + 1]:
            return False
    return True


# ----- five independent sorts -----

def bubble_sort(src):
    a = copy_list(src)
    n = len(a)
    for i in range(n):
        for j in range(n - 1 - i):
            if a[j] > a[j + 1]:
                a[j], a[j + 1] = a[j + 1], a[j]
    return a


def insertion_sort(src):
    a = copy_list(src)
    for i in range(1, len(a)):
        key = a[i]
        j = i - 1
        while j >= 0 and a[j] > key:
            a[j + 1] = a[j]
            j -= 1
        a[j + 1] = key
    return a


def selection_sort(src):
    a = copy_list(src)
    n = len(a)
    for i in range(n):
        lo = i
        for j in range(i + 1, n):
            if a[j] < a[lo]:
                lo = j
        a[i], a[lo] = a[lo], a[i]
    return a


def quicksort(xs):
    if len(xs) <= 1:
        return copy_list(xs)
    pivot = xs[0]
    rest = xs[1:]
    less = [x for x in rest if x < pivot]
    equal = [x for x in xs if x == pivot]
    greater = [x for x in rest if x > pivot]
    return concat(concat(quicksort(less), equal), quicksort(greater))


def merge(a, b):
    out = []
    i = 0
    j = 0
    while i < len(a) and j < len(b):
        if a[i] <= b[j]:
            out.append(a[i])
            i += 1
        else:
            out.append(b[j])
            j += 1
    while i < len(a):
        out.append(a[i])
        i += 1
    while j < len(b):
        out.append(b[j])
        j += 1
    return out


def merge_sort(xs):
    if len(xs) <= 1:
        return copy_list(xs)
    mid = len(xs) // 2
    left = merge_sort(xs[:mid])
    right = merge_sort(xs[mid:])
    return merge(left, right)


data = [5, 2, 8, 1, 9, 2, 7, 3, 8, 4, 0, 6]
expected = "[0, 1, 2, 2, 3, 4, 5, 6, 7, 8, 8, 9]"

check("bubble sort", render(bubble_sort(data)), expected)
check("insertion sort", render(insertion_sort(data)), expected)
check("selection sort", render(selection_sort(data)), expected)
check("quicksort", render(quicksort(data)), expected)
check("merge sort", render(merge_sort(data)), expected)
check_true("bubble is sorted", is_sorted(bubble_sort(data)))
check_true("original untouched", not is_sorted(data))
check("stable length", len(merge_sort(data)), len(data))

# all five agree with one another
sorts = [bubble_sort(data), insertion_sort(data), selection_sort(data), quicksort(data), merge_sort(data)]
agree = 0
for s in sorts:
    if render(s) == expected:
        agree += 1
check("all sorts agree", agree, 5)

# sorting already-sorted and reverse-sorted inputs
asc = [1, 2, 3, 4, 5]
desc = [5, 4, 3, 2, 1]
check("sort ascending", render(quicksort(asc)), "[1, 2, 3, 4, 5]")
check("sort descending", render(insertion_sort(desc)), "[1, 2, 3, 4, 5]")
check("sort singleton", render(merge_sort([42])), "[42]")
check("sort empty", render(quicksort([])), "[]")


# ----- binary search over a sorted list -----

def binary_search(a, target):
    lo = 0
    hi = len(a) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if a[mid] == target:
            return mid
        if a[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1


sorted_data = merge_sort(data)
check("bsearch found 0", binary_search(sorted_data, 0), 0)
check("bsearch found 9", binary_search(sorted_data, 9), 11)
check("bsearch missing", binary_search(sorted_data, 100), -1)
check("bsearch missing low", binary_search(sorted_data, -5), -1)
found_all = 0
for v in [0, 1, 3, 4, 5, 6, 7, 9]:
    if binary_search(sorted_data, v) >= 0:
        found_all += 1
check("bsearch finds present", found_all, 8)


# ----- a stack (LIFO) on a plain list -----

stack = []
for v in [10, 20, 30]:
    stack.append(v)
check("stack size", len(stack), 3)
check("stack top", stack[-1], 30)
check("stack pop", stack.pop(), 30)
check("stack pop 2", stack.pop(), 20)
check("stack size after", len(stack), 1)


# ----- an in-place FIFO queue (mutates the backing list) -----

def enqueue(q, v):
    q.append(v)


def dequeue(q):
    front = q[0]
    for i in range(len(q) - 1):
        q[i] = q[i + 1]
    q.pop()
    return front


queue = []
for v in [1, 2, 3, 4]:
    enqueue(queue, v)
check("queue len", len(queue), 4)
check("dequeue 1", dequeue(queue), 1)
check("dequeue 2", dequeue(queue), 2)
enqueue(queue, 5)
check("queue order", render(queue), "[3, 4, 5]")
check("dequeue 3", dequeue(queue), 3)


# ----- a singly linked list built from dicts -----

def ll_push(head, v):
    return {"val": v, "next": head}


def ll_length(head):
    n = 0
    node = head
    while node != None:
        n += 1
        node = node["next"]
    return n


def ll_sum(head):
    total = 0
    node = head
    while node != None:
        total += node["val"]
        node = node["next"]
    return total


def ll_to_list(head):
    out = []
    node = head
    while node != None:
        out.append(node["val"])
        node = node["next"]
    return out


def ll_reverse(head):
    prev = None
    node = head
    while node != None:
        nxt = node["next"]
        node["next"] = prev
        prev = node
        node = nxt
    return prev


lst = None
for v in [1, 2, 3, 4, 5]:
    lst = ll_push(lst, v)
check("ll length", ll_length(lst), 5)
check("ll head", lst["val"], 5)
check("ll sum", ll_sum(lst), 15)
check("ll to list", render(ll_to_list(lst)), "[5, 4, 3, 2, 1]")
lst = ll_reverse(lst)
check("ll reversed", render(ll_to_list(lst)), "[1, 2, 3, 4, 5]")
check("ll length after reverse", ll_length(lst), 5)


# ----- matrices as nested lists with fresh rows via comprehension -----

def make_matrix(rows, cols, fill):
    return [[fill for c in range(cols)] for r in range(rows)]


def identity(n):
    m = make_matrix(n, n, 0)
    for i in range(n):
        m[i][i] = 1
    return m


def mat_mul(a, b):
    n = len(a)
    m = len(b[0])
    k = len(b)
    out = make_matrix(n, m, 0)
    for i in range(n):
        for j in range(m):
            acc = 0
            for t in range(k):
                acc += a[i][t] * b[t][j]
            out[i][j] = acc
    return out


def transpose(a):
    rows = len(a)
    cols = len(a[0])
    out = make_matrix(cols, rows, 0)
    for i in range(rows):
        for j in range(cols):
            out[j][i] = a[i][j]
    return out


def trace(a):
    s = 0
    for i in range(len(a)):
        s += a[i][i]
    return s


grid = make_matrix(2, 3, 0)
grid[0][0] = 7
check("fresh rows", grid[1][0], 0)
check("grid set", grid[0][0], 7)

ident = identity(3)
check("identity render", render(ident), "[[1, 0, 0], [0, 1, 0], [0, 0, 1]]")
check("identity trace", trace(ident), 3)

A = [[1, 2, 3], [4, 5, 6]]
B = [[7, 8], [9, 10], [11, 12]]
prod = mat_mul(A, B)
check("mat mul render", render(prod), "[[58, 64], [139, 154]]")
check("mat mul cell", prod[1][1], 154)

# multiplying by identity returns the same matrix
sq = [[2, 0, 1], [3, 5, 4], [6, 7, 8]]
check("mul by identity", render(mat_mul(sq, identity(3))), render(sq))
check("trace square", trace(sq), 15)

T = transpose(A)
check("transpose render", render(T), "[[1, 4], [2, 5], [3, 6]]")
check("transpose twice", render(transpose(T)), render(A))


check("no failures", fails[0], 0)
if fails[0] == 0:
    print("Python big-1 (data structures and sorting) self test passed")
exit(fails[0])
