// Self-checking test for the JavaScript interpreter (js-interpreter.abnf) and the
// LLVM-IR compiler (js-to-llvm-ir.abnf). THEME: data structures + algorithms.
//
// Implements and cross-checks four sorting algorithms, binary search, a stack, a
// queue, a singly linked list (list reversal), a binary search tree (inorder =
// sorted), and an object-backed frequency map iterated with for-in. Every result is
// compared against an independently computed reference; main() returns the number of
// failed checks, so exit code 0 means every check passed. Uses only genuinely
// implemented constructs, so a default (non -warn-unsupported) run of either grammar
// passes, and the compiler emits byte-identical IR under both engines.

var failures = 0;
function check(cond) { if (!cond) { failures = failures + 1; } }

// ----- small array helpers -----
function copyArray(a) {
    var out = [];
    for (var i = 0; i < a.length; i++) { out.push(a[i]); }
    return out;
}
function arraysEqual(a, b) {
    if (a.length !== b.length) { return false; }
    for (var i = 0; i < a.length; i++) { if (a[i] !== b[i]) { return false; } }
    return true;
}
function isSorted(a) {
    for (var i = 1; i < a.length; i++) { if (a[i - 1] > a[i]) { return false; } }
    return true;
}

// ----- four independent sorting algorithms (each returns a fresh array) -----
function bubbleSort(input) {
    var a = copyArray(input);
    var n = a.length;
    for (var i = 0; i < n - 1; i++) {
        var swapped = false;
        for (var j = 0; j < n - 1 - i; j++) {
            if (a[j] > a[j + 1]) {
                var t = a[j];
                a[j] = a[j + 1];
                a[j + 1] = t;
                swapped = true;
            }
        }
        if (!swapped) { break; }   // early exit when already sorted
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

function quickSort(input) {
    var a = copyArray(input);
    quickSortRange(a, 0, a.length - 1);
    return a;
}
function quickSortRange(a, lo, hi) {
    if (lo >= hi) { return; }
    var pivot = a[hi];
    var i = lo - 1;
    for (var j = lo; j < hi; j++) {
        if (a[j] <= pivot) {
            i++;
            var t = a[i]; a[i] = a[j]; a[j] = t;
        }
    }
    var t2 = a[i + 1]; a[i + 1] = a[hi]; a[hi] = t2;
    var p = i + 1;
    quickSortRange(a, lo, p - 1);
    quickSortRange(a, p + 1, hi);
}

function mergeSort(input) {
    if (input.length <= 1) { return copyArray(input); }
    var mid = Math.floor(input.length / 2);
    var left = [];
    var right = [];
    for (var i = 0; i < input.length; i++) {
        if (i < mid) { left.push(input[i]); } else { right.push(input[i]); }
    }
    return merge(mergeSort(left), mergeSort(right));
}
function merge(a, b) {
    var out = [];
    var i = 0;
    var j = 0;
    while (i < a.length && j < b.length) {
        if (a[i] <= b[j]) { out.push(a[i]); i++; }
        else { out.push(b[j]); j++; }
    }
    while (i < a.length) { out.push(a[i]); i++; }
    while (j < b.length) { out.push(b[j]); j++; }
    return out;
}

function testSorting() {
    var data = [5, 2, 9, 1, 5, 6, 3, 8, 7, 0, 4, 2];
    // An independent reference sort (selection sort) to compare against.
    var reference = copyArray(data);
    for (var i = 0; i < reference.length; i++) {
        var minIdx = i;
        for (var j = i + 1; j < reference.length; j++) {
            if (reference[j] < reference[minIdx]) { minIdx = j; }
        }
        var t = reference[i]; reference[i] = reference[minIdx]; reference[minIdx] = t;
    }
    check(isSorted(reference));

    var b = bubbleSort(data);
    var ins = insertionSort(data);
    var q = quickSort(data);
    var m = mergeSort(data);
    check(isSorted(b) && arraysEqual(b, reference));
    check(isSorted(ins) && arraysEqual(ins, reference));
    check(isSorted(q) && arraysEqual(q, reference));
    check(isSorted(m) && arraysEqual(m, reference));
    // The original data must be untouched (all sorts copied it).
    check(data.length === 12 && data[0] === 5 && data[11] === 2);

    // Edge cases.
    check(arraysEqual(quickSort([]), []));
    check(arraysEqual(mergeSort([42]), [42]));
    check(arraysEqual(bubbleSort([3, 3, 3]), [3, 3, 3]));
}

// ----- binary search over a sorted array -----
function binarySearch(sorted, target) {
    var lo = 0;
    var hi = sorted.length - 1;
    while (lo <= hi) {
        var mid = Math.floor((lo + hi) / 2);
        if (sorted[mid] === target) { return mid; }
        if (sorted[mid] < target) { lo = mid + 1; } else { hi = mid - 1; }
    }
    return -1;
}
function testBinarySearch() {
    var s = [1, 3, 5, 7, 9, 11, 13, 15];
    check(binarySearch(s, 1) === 0);
    check(binarySearch(s, 15) === 7);
    check(binarySearch(s, 7) === 3);
    check(binarySearch(s, 8) === -1);
    check(binarySearch(s, 0) === -1);
    check(binarySearch(s, 100) === -1);
    check(binarySearch([], 5) === -1);
    // Every present element must be found at its true index.
    for (var i = 0; i < s.length; i++) { check(binarySearch(s, s[i]) === i); }
}

// ----- a stack, built on a plain array captured in a closure -----
function makeStack() {
    var items = [];
    return { push:    function(v) { items.push(v); },
             pop:     function() { return items.pop(); },
             peek:    function() { return items[items.length - 1]; },
             size:    function() { return items.length; },
             isEmpty: function() { return items.length === 0; } };
}
function testStack() {
    var st = makeStack();
    check(st.isEmpty());
    st.push(10);
    st.push(20);
    st.push(30);
    check(st.size() === 3);
    check(st.peek() === 30);
    check(st.pop() === 30);
    check(st.pop() === 20);
    check(st.size() === 1);
    check(!st.isEmpty());
    check(st.pop() === 10);
    check(st.isEmpty());

    // Use the stack to check balanced brackets.
    check(bracketsBalanced("([]{()})") === true);
    check(bracketsBalanced("([)]") === false);
    check(bracketsBalanced("(((") === false);
    check(bracketsBalanced("") === true);
}
function bracketsBalanced(s) {
    var st = makeStack();
    for (var i = 0; i < s.length; i++) {
        var ch = s.charAt(i);
        if (ch === "(" || ch === "[" || ch === "{") {
            st.push(ch);
        } else {
            if (st.isEmpty()) { return false; }
            var open = st.pop();
            if (ch === ")" && open !== "(") { return false; }
            if (ch === "]" && open !== "[") { return false; }
            if (ch === "}" && open !== "{") { return false; }
        }
    }
    return st.isEmpty();
}

// ----- a queue with head/tail indices, captured in a closure -----
function makeQueue() {
    var items = [];
    var head = 0;
    return { enqueue: function(v) { items.push(v); },
             dequeue: function() { var v = items[head]; head++; return v; },
             count:   function() { return items.length - head; } };
}
function testQueue() {
    var qu = makeQueue();
    qu.enqueue(1);
    qu.enqueue(2);
    qu.enqueue(3);
    check(qu.count() === 3);
    check(qu.dequeue() === 1);
    check(qu.dequeue() === 2);
    qu.enqueue(4);
    check(qu.count() === 2);
    check(qu.dequeue() === 3);
    check(qu.dequeue() === 4);
    check(qu.count() === 0);
}

// ----- a singly linked list, built from {val, next} nodes -----
function listFromArray(a) {
    var head = null;
    for (var i = a.length - 1; i >= 0; i--) { head = { val: a[i], next: head }; }
    return head;
}
function listToArray(head) {
    var out = [];
    var node = head;
    while (node !== null) { out.push(node.val); node = node.next; }
    return out;
}
function listLength(head) {
    var n = 0;
    var node = head;
    while (node !== null) { n++; node = node.next; }
    return n;
}
function listReverse(head) {
    var prev = null;
    var node = head;
    while (node !== null) {
        var nxt = node.next;
        node.next = prev;
        prev = node;
        node = nxt;
    }
    return prev;
}
function testLinkedList() {
    var head = listFromArray([1, 2, 3, 4, 5]);
    check(listLength(head) === 5);
    check(arraysEqual(listToArray(head), [1, 2, 3, 4, 5]));
    var rev = listReverse(head);
    check(arraysEqual(listToArray(rev), [5, 4, 3, 2, 1]));
    check(listLength(rev) === 5);
    check(listToArray(null).length === 0);
    check(listLength(null) === 0);
}

// ----- a binary search tree; inorder traversal yields sorted order -----
function bstInsert(root, value) {
    if (root === null) { return { val: value, left: null, right: null }; }
    if (value < root.val) { root.left = bstInsert(root.left, value); }
    else if (value > root.val) { root.right = bstInsert(root.right, value); }
    return root;                 // duplicates ignored
}
function bstInorder(root, out) {
    if (root === null) { return; }
    bstInorder(root.left, out);
    out.push(root.val);
    bstInorder(root.right, out);
}
function bstContains(root, value) {
    var node = root;
    while (node !== null) {
        if (value === node.val) { return true; }
        node = (value < node.val) ? node.left : node.right;
    }
    return false;
}
function bstHeight(root) {
    if (root === null) { return 0; }
    var lh = bstHeight(root.left);
    var rh = bstHeight(root.right);
    return 1 + (lh > rh ? lh : rh);
}
function testBST() {
    var values = [8, 3, 10, 1, 6, 14, 4, 7, 13];
    var root = null;
    for (var i = 0; i < values.length; i++) { root = bstInsert(root, values[i]); }
    var order = [];
    bstInorder(root, order);
    check(isSorted(order));
    check(order.length === 9);
    check(order[0] === 1 && order[8] === 14);
    for (var j = 0; j < values.length; j++) { check(bstContains(root, values[j])); }
    check(bstContains(root, 99) === false);
    check(bstContains(root, 5) === false);
    check(bstHeight(root) === 4);
    // Inserting duplicates does not change the inorder traversal.
    root = bstInsert(root, 8);
    root = bstInsert(root, 6);
    var order2 = [];
    bstInorder(root, order2);
    check(arraysEqual(order, order2));
}

// ----- an object used as a frequency map, iterated with for-in -----
function wordFrequencies(words) {
    var freq = {};
    for (var i = 0; i < words.length; i++) {
        var w = words[i];
        if (freq[w] === undefined) { freq[w] = 0; }
        freq[w] = freq[w] + 1;
    }
    return freq;
}
function testFrequencyMap() {
    var words = ["a", "b", "a", "c", "b", "a", "d", "b", "a"];
    var freq = wordFrequencies(words);
    check(freq["a"] === 4);
    check(freq["b"] === 3);
    check(freq["c"] === 1);
    check(freq["d"] === 1);

    // for-in visits exactly the four distinct keys, and the counts sum to the input.
    var distinct = 0;
    var total = 0;
    var best = "";
    var bestCount = -1;
    for (var k in freq) {
        distinct++;
        total = total + freq[k];
        if (freq[k] > bestCount) { bestCount = freq[k]; best = k; }
    }
    check(distinct === 4);
    check(total === 9);
    check(best === "a" && bestCount === 4);
}

function main() {
    testSorting();
    testBinarySearch();
    testStack();
    testQueue();
    testLinkedList();
    testBST();
    testFrequencyMap();
    return failures;
}
