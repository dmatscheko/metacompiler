/* MetaJS big test 1 - Data structures and algorithms.
 *
 * Exercises the implemented MetaJS subset with real algorithms: five sorts,
 * binary search, number theory (gcd/lcm, sieve of Eratosthenes, fibonacci),
 * and closure- and object-backed data structures (stack, queue, singly linked
 * list, binary search tree, binary min-heap, NxN integer matrices).
 *
 * It is self checking: every result is compared against a precomputed value
 * and main() returns the number of failed checks, so the metacompiler run
 * exits 0 exactly when everything is correct. The same program runs
 * identically under the interpreter and the LLVM-IR compiler, and under both
 * the goja and the frozen engine. **/

var failures = 0;
var checks = 0;

function check(name, got, want) {
    checks = checks + 1;
    if (got !== want) {
        println("FAIL " + name + ": got " + got + " want " + want);
        failures = failures + 1;
    }
}

// Integer arrays are compared by their comma joined text (integers format
// identically everywhere), which keeps the checks free of element loops.
function checkArr(name, gotArr, wantStr) {
    check(name, gotArr.join(","), wantStr);
}

// ----- small array helpers -----

function swap(a, i, j) {
    var tmp = a[i];
    a[i] = a[j];
    a[j] = tmp;
}

function copyArray(a) { return a.slice(0); }

function arraySum(a) {
    var s = 0;
    for (var i = 0; i < a.length; i++) { s += a[i]; }
    return s;
}

function arrayMax(a) {
    var m = a[0];
    for (var i = 1; i < a.length; i++) {
        if (a[i] > m) { m = a[i]; }
    }
    return m;
}

function reverseArray(a) {
    var out = [];
    for (var i = a.length - 1; i >= 0; i--) { out.push(a[i]); }
    return out;
}

function isSorted(a) {
    for (var i = 1; i < a.length; i++) {
        if (a[i - 1] > a[i]) { return false; }
    }
    return true;
}

// ----- sorting algorithms (each returns a fresh sorted array) -----

function bubbleSort(input) {
    var a = copyArray(input);
    var n = a.length;
    for (var i = 0; i < n - 1; i++) {
        var swapped = false;
        for (var j = 0; j < n - 1 - i; j++) {
            if (a[j] > a[j + 1]) {
                swap(a, j, j + 1);
                swapped = true;
            }
        }
        if (!swapped) { break; }
    }
    return a;
}

function insertionSort(input) {
    var a = copyArray(input);
    for (var i = 1; i < a.length; i++) {
        var key = a[i];
        var j = i - 1;
        while (j >= 0 && a[j] > key) {
            a[j + 1] = a[j];
            j--;
        }
        a[j + 1] = key;
    }
    return a;
}

function selectionSort(input) {
    var a = copyArray(input);
    var n = a.length;
    for (var i = 0; i < n - 1; i++) {
        var min = i;
        for (var j = i + 1; j < n; j++) {
            if (a[j] < a[min]) { min = j; }
        }
        if (min != i) { swap(a, i, min); }
    }
    return a;
}

function quickSort(a) {
    if (a.length <= 1) { return copyArray(a); }
    var pivot = a[0];
    var less = [];
    var rest = [];
    for (var i = 1; i < a.length; i++) {
        if (a[i] < pivot) { less.push(a[i]); }
        else { rest.push(a[i]); }
    }
    return quickSort(less).concat([pivot]).concat(quickSort(rest));
}

function merge(x, y) {
    var out = [];
    var i = 0;
    var j = 0;
    while (i < x.length && j < y.length) {
        if (x[i] <= y[j]) { out.push(x[i]); i++; }
        else { out.push(y[j]); j++; }
    }
    while (i < x.length) { out.push(x[i]); i++; }
    while (j < y.length) { out.push(y[j]); j++; }
    return out;
}

function mergeSort(a) {
    if (a.length <= 1) { return copyArray(a); }
    var mid = Math.floor(a.length / 2);
    return merge(mergeSort(a.slice(0, mid)), mergeSort(a.slice(mid)));
}

// ----- searching -----

function binarySearch(a, target) {
    var lo = 0;
    var hi = a.length - 1;
    while (lo <= hi) {
        var mid = Math.floor((lo + hi) / 2);
        if (a[mid] == target) { return mid; }
        if (a[mid] < target) { lo = mid + 1; }
        else { hi = mid - 1; }
    }
    return -1;
}

// ----- number theory -----

function gcd(x, y) {
    while (y != 0) {
        var t = y;
        y = x % y;
        x = t;
    }
    return x;
}

function lcm(x, y) { return x / gcd(x, y) * y; }

function sieve(n) {
    var composite = [];
    for (var i = 0; i <= n; i++) { composite.push(false); }
    var primes = [];
    for (var p = 2; p <= n; p++) {
        if (!composite[p]) {
            primes.push(p);
            for (var m = p * p; m <= n; m += p) { composite[m] = true; }
        }
    }
    return primes;
}

function fib(n) {
    var a = 0;
    var b = 1;
    for (var i = 0; i < n; i++) {
        var t = a + b;
        a = b;
        b = t;
    }
    return a;
}

// ----- closure backed stack and queue -----

function makeStack() {
    var items = [];
    return {
        push: function(x) { items.push(x); return items.length; },
        pop: function() { return items.pop(); },
        peek: function() { return items[items.length - 1]; },
        size: function() { return items.length; },
        isEmpty: function() { return items.length == 0; }
    };
}

function makeQueue() {
    var items = [];
    return {
        enqueue: function(x) { items.push(x); return items.length; },
        dequeue: function() { return items.shift(); },
        size: function() { return items.length; }
    };
}

// ----- singly linked list (plain objects) -----

function cons(value, next) { return {value: value, next: next}; }

function listFromArray(a) {
    var head = null;
    for (var i = a.length - 1; i >= 0; i--) { head = cons(a[i], head); }
    return head;
}

function listLength(node) {
    var count = 0;
    var cur = node;
    while (cur != null) {
        count++;
        cur = cur.next;
    }
    return count;
}

function listSum(node) {
    var s = 0;
    var cur = node;
    while (cur != null) {
        s += cur.value;
        cur = cur.next;
    }
    return s;
}

function reverseList(node) {
    var prev = null;
    var cur = node;
    while (cur != null) {
        var nxt = cur.next;
        cur.next = prev;
        prev = cur;
        cur = nxt;
    }
    return prev;
}

function listToArray(node) {
    var out = [];
    var cur = node;
    while (cur != null) {
        out.push(cur.value);
        cur = cur.next;
    }
    return out;
}

// ----- binary search tree -----

function bstInsert(root, value) {
    if (root == null) { return {value: value, left: null, right: null}; }
    if (value < root.value) { root.left = bstInsert(root.left, value); }
    else if (value > root.value) { root.right = bstInsert(root.right, value); }
    return root;
}

function bstFromArray(a) {
    var root = null;
    for (var i = 0; i < a.length; i++) { root = bstInsert(root, a[i]); }
    return root;
}

function bstContains(root, value) {
    var cur = root;
    while (cur != null) {
        if (value == cur.value) { return true; }
        if (value < cur.value) { cur = cur.left; }
        else { cur = cur.right; }
    }
    return false;
}

function bstInorder(root, out) {
    if (root == null) { return; }
    bstInorder(root.left, out);
    out.push(root.value);
    bstInorder(root.right, out);
}

function bstHeight(root) {
    if (root == null) { return 0; }
    var lh = bstHeight(root.left);
    var rh = bstHeight(root.right);
    return 1 + (lh > rh ? lh : rh);
}

// ----- binary min-heap over an array -----

function heapPush(heap, value) {
    heap.push(value);
    var i = heap.length - 1;
    while (i > 0) {
        var parent = Math.floor((i - 1) / 2);
        if (heap[parent] <= heap[i]) { break; }
        swap(heap, parent, i);
        i = parent;
    }
}

function heapPop(heap) {
    var n = heap.length;
    if (n == 0) { return -1; }
    var top = heap[0];
    var last = heap.pop();
    if (heap.length > 0) {
        heap[0] = last;
        var m = heap.length;
        var i = 0;
        while (true) {
            var l = 2 * i + 1;
            var r = 2 * i + 2;
            var smallest = i;
            if (l < m && heap[l] < heap[smallest]) { smallest = l; }
            if (r < m && heap[r] < heap[smallest]) { smallest = r; }
            if (smallest == i) { break; }
            swap(heap, i, smallest);
            i = smallest;
        }
    }
    return top;
}

function heapSort(input) {
    var heap = [];
    for (var i = 0; i < input.length; i++) { heapPush(heap, input[i]); }
    var out = [];
    while (heap.length > 0) { out.push(heapPop(heap)); }
    return out;
}

// ----- NxN integer matrices as arrays of arrays -----

function identity(n) {
    var m = [];
    for (var i = 0; i < n; i++) {
        var row = [];
        for (var j = 0; j < n; j++) { row.push(i == j ? 1 : 0); }
        m.push(row);
    }
    return m;
}

function matMul(a, b, n) {
    var c = [];
    for (var i = 0; i < n; i++) {
        var row = [];
        for (var j = 0; j < n; j++) {
            var sum = 0;
            for (var k = 0; k < n; k++) { sum += a[i][k] * b[k][j]; }
            row.push(sum);
        }
        c.push(row);
    }
    return c;
}

function transpose(a, n) {
    var t = [];
    for (var i = 0; i < n; i++) {
        var row = [];
        for (var j = 0; j < n; j++) { row.push(a[j][i]); }
        t.push(row);
    }
    return t;
}

function flatten(m) {
    var out = [];
    for (var i = 0; i < m.length; i++) {
        for (var j = 0; j < m[i].length; j++) { out.push(m[i][j]); }
    }
    return out;
}

function main() {
    var data = [5, 2, 9, 1, 5, 6, 3, 8, 7, 4, 0];
    var sortedStr = "0,1,2,3,4,5,5,6,7,8,9";

    // ----- sorts all agree with each other and are actually sorted -----
    checkArr("bubble sort", bubbleSort(data), sortedStr);
    checkArr("insertion sort", insertionSort(data), sortedStr);
    checkArr("selection sort", selectionSort(data), sortedStr);
    checkArr("quick sort", quickSort(data), sortedStr);
    checkArr("merge sort", mergeSort(data), sortedStr);
    checkArr("heap sort", heapSort(data), sortedStr);
    check("input untouched", data.join(","), "5,2,9,1,5,6,3,8,7,4,0");
    check("already sorted stays", isSorted(bubbleSort(data)), true);
    check("empty sort", mergeSort([]).length, 0);
    check("single sort", quickSort([42])[0], 42);
    check("reverse then sort", insertionSort([9, 8, 7, 6, 5]).join(","), "5,6,7,8,9");

    // ----- array helpers -----
    check("array sum", arraySum(data), 50);
    check("array max", arrayMax(data), 9);
    checkArr("reverse array", reverseArray([1, 2, 3, 4]), "4,3,2,1");

    // ----- binary search over the sorted array -----
    var sorted = mergeSort(data);
    check("bsearch found first", binarySearch(sorted, 0), 0);
    check("bsearch found last", binarySearch(sorted, 9), 10);
    check("bsearch middle", binarySearch(sorted, 6), 7);
    check("bsearch missing", binarySearch(sorted, 100), -1);
    check("bsearch below", binarySearch(sorted, -5), -1);
    check("bsearch each present", binarySearch([2, 4, 6, 8, 10], 8), 3);

    // ----- number theory -----
    check("gcd coprime", gcd(17, 5), 1);
    check("gcd common", gcd(48, 36), 12);
    check("gcd zero", gcd(7, 0), 7);
    check("lcm", lcm(4, 6), 12);
    check("lcm coprime", lcm(7, 5), 35);
    checkArr("sieve 30", sieve(30), "2,3,5,7,11,13,17,19,23,29");
    check("prime count to 100", sieve(100).length, 25);
    check("largest prime under 50", arrayMax(sieve(50)), 47);
    check("fib 10", fib(10), 55);
    check("fib 20", fib(20), 6765);
    check("fib 0", fib(0), 0);
    check("fib 1", fib(1), 1);

    // ----- stack (LIFO) -----
    var st = makeStack();
    check("stack empty", st.isEmpty(), true);
    st.push(10);
    st.push(20);
    check("stack push returns size", st.push(30), 3);
    check("stack peek", st.peek(), 30);
    check("stack size", st.size(), 3);
    check("stack pop 1", st.pop(), 30);
    check("stack pop 2", st.pop(), 20);
    check("stack not empty", st.isEmpty(), false);
    check("stack pop 3", st.pop(), 10);
    check("stack empty again", st.isEmpty(), true);

    // ----- queue (FIFO) -----
    var q = makeQueue();
    q.enqueue(1);
    q.enqueue(2);
    q.enqueue(3);
    check("queue size", q.size(), 3);
    check("queue dequeue 1", q.dequeue(), 1);
    check("queue dequeue 2", q.dequeue(), 2);
    q.enqueue(4);
    check("queue dequeue 3", q.dequeue(), 3);
    check("queue dequeue 4", q.dequeue(), 4);
    check("queue drained", q.size(), 0);

    // ----- two stacks are independent (closure state) -----
    var sa = makeStack();
    var sb = makeStack();
    sa.push(1);
    sa.push(2);
    sb.push(9);
    check("independent a", sa.size(), 2);
    check("independent b", sb.size(), 1);
    check("independent b top", sb.peek(), 9);

    // ----- linked list -----
    var list = listFromArray([10, 20, 30, 40]);
    check("list length", listLength(list), 4);
    check("list sum", listSum(list), 100);
    check("list head", list.value, 10);
    check("list second", list.next.value, 20);
    checkArr("list to array", listToArray(list), "10,20,30,40");
    var rev = reverseList(list);
    checkArr("list reversed", listToArray(rev), "40,30,20,10");
    check("empty list length", listLength(null), 0);
    check("empty list sum", listSum(null), 0);

    // ----- binary search tree: inorder walk is sorted -----
    var tree = bstFromArray([5, 3, 8, 1, 4, 7, 9, 2, 6]);
    var walk = [];
    bstInorder(tree, walk);
    checkArr("bst inorder sorted", walk, "1,2,3,4,5,6,7,8,9");
    check("bst contains yes", bstContains(tree, 7), true);
    check("bst contains no", bstContains(tree, 100), false);
    check("bst root", tree.value, 5);
    check("bst height", bstHeight(tree), 4);
    var deg = bstFromArray([1, 2, 3, 4, 5]);
    check("degenerate tree height", bstHeight(deg), 5);

    // ----- min-heap ordering -----
    var h = [];
    heapPush(h, 5);
    heapPush(h, 3);
    heapPush(h, 8);
    heapPush(h, 1);
    heapPush(h, 9);
    heapPush(h, 2);
    check("heap min at top", h[0], 1);
    check("heap pop 1", heapPop(h), 1);
    check("heap pop 2", heapPop(h), 2);
    check("heap pop 3", heapPop(h), 3);
    check("heap new min", h[0], 5);
    checkArr("heap sort ties", heapSort([4, 4, 1, 1, 3, 2]), "1,1,2,3,4,4");

    // ----- matrices -----
    var a = [[1, 2], [3, 4]];
    var b = [[5, 6], [7, 8]];
    checkArr("matrix multiply", flatten(matMul(a, b, 2)), "19,22,43,50");
    checkArr("identity multiply", flatten(matMul(a, identity(2), 2)), "1,2,3,4");
    checkArr("transpose", flatten(transpose(a, 2)), "1,3,2,4");
    var i3 = identity(3);
    checkArr("identity 3", flatten(i3), "1,0,0,0,1,0,0,0,1");
    var m3 = [[1, 2, 3], [4, 5, 6], [7, 8, 9]];
    checkArr("3x3 by identity", flatten(matMul(m3, identity(3), 3)), "1,2,3,4,5,6,7,8,9");
    checkArr("3x3 squared", flatten(matMul(m3, m3, 3)), "30,36,42,66,81,96,102,126,150");

    printf("%c%c %d checks\n", 79, 75, checks);
    if (failures == 0) { println("MetaJS big test 1 (data structures) passed"); }
    return failures;
}
