// Self-checking TypeScript test #big-1: DATA STRUCTURES + ALGORITHMS.
//
// Themes: sorting (insertion / bubble / quick / merge), binary search, a generic
// Stack and Queue, a singly linked list (append / get / reverse / toArray), a binary
// search tree (insert / contains / in-order traversal), and a small min-heap. Every
// result is compared to a known-good value; main() returns the number of failed checks,
// so exit code 0 means all engines (goja and -frozen) and both the interpreter and the
// compiler agree.
//
// The full static type layer - interfaces, generic <T> signatures, an enum, ': Type'
// annotations, 'as' casts - is erased at run time; what executes is plain JavaScript.

let failures: number = 0;

function check(cond: boolean, _label: string): void {
    if (!cond) { failures = failures + 1; }
}

// A three-way comparison result, erased at run time to its integer constants
// (bare members auto-increment from 0: Less=0, Equal=1, Greater=2).
enum Ordering {
    Less,
    Equal,
    Greater,
}

// ---- small array helpers (no Array.prototype extras are used) ----

function arraysEqual(a: number[], b: number[]): boolean {
    if (a.length !== b.length) { return false; }
    for (let i: number = 0; i < a.length; i++) {
        if (a[i] !== b[i]) { return false; }
    }
    return true;
}

function copyArray(a: number[]): number[] {
    const out: number[] = [];
    for (let i: number = 0; i < a.length; i++) { out.push(a[i]); }
    return out;
}

function isSorted(a: number[]): boolean {
    for (let i: number = 1; i < a.length; i++) {
        if (a[i - 1] > a[i]) { return false; }
    }
    return true;
}

function compareNumbers(a: number, b: number): Ordering {
    if (a < b) { return Ordering.Less; }
    if (a > b) { return Ordering.Greater; }
    return Ordering.Equal;
}

// ---- sorting algorithms (each returns a new, sorted array) ----

function insertionSort(input: number[]): number[] {
    const a: number[] = copyArray(input);
    for (let i: number = 1; i < a.length; i++) {
        const key: number = a[i];
        let j: number = i - 1;
        while (j >= 0 && a[j] > key) {
            a[j + 1] = a[j];
            j--;
        }
        a[j + 1] = key;
    }
    return a;
}

function bubbleSort(input: number[]): number[] {
    const a: number[] = copyArray(input);
    let swapped: boolean = true;
    let n: number = a.length;
    while (swapped) {
        swapped = false;
        for (let i: number = 1; i < n; i++) {
            if (a[i - 1] > a[i]) {
                const t: number = a[i - 1];
                a[i - 1] = a[i];
                a[i] = t;
                swapped = true;
            }
        }
        n = n - 1;
    }
    return a;
}

function quickSort(input: number[]): number[] {
    if (input.length <= 1) { return copyArray(input); }
    const pivot: number = input[Math.floor(input.length / 2)];
    const less: number[] = [];
    const equal: number[] = [];
    const greater: number[] = [];
    for (let i: number = 0; i < input.length; i++) {
        const v: number = input[i];
        const c: Ordering = compareNumbers(v, pivot);
        if (c === Ordering.Less) { less.push(v); }
        else if (c === Ordering.Greater) { greater.push(v); }
        else { equal.push(v); }
    }
    return quickSort(less).concat(equal).concat(quickSort(greater));
}

function merge(a: number[], b: number[]): number[] {
    const out: number[] = [];
    let i: number = 0;
    let j: number = 0;
    while (i < a.length && j < b.length) {
        if (a[i] <= b[j]) { out.push(a[i]); i++; }
        else { out.push(b[j]); j++; }
    }
    while (i < a.length) { out.push(a[i]); i++; }
    while (j < b.length) { out.push(b[j]); j++; }
    return out;
}

function mergeSort(input: number[]): number[] {
    if (input.length <= 1) { return copyArray(input); }
    const mid: number = Math.floor(input.length / 2);
    const left: number[] = [];
    const right: number[] = [];
    for (let i: number = 0; i < input.length; i++) {
        if (i < mid) { left.push(input[i]); } else { right.push(input[i]); }
    }
    return merge(mergeSort(left), mergeSort(right));
}

// Binary search over a sorted array; returns the index or -1.
function binarySearch(a: number[], target: number): number {
    let lo: number = 0;
    let hi: number = a.length - 1;
    while (lo <= hi) {
        const mid: number = lo + Math.floor((hi - lo) / 2);
        if (a[mid] === target) { return mid; }
        if (a[mid] < target) { lo = mid + 1; } else { hi = mid - 1; }
    }
    return -1;
}

function testSorting(): void {
    const data: number[] = [5, 2, 9, 1, 5, 6, 3, 8, 7, 4, 0, 5];
    const expected: number[] = [0, 1, 2, 3, 4, 5, 5, 5, 6, 7, 8, 9];

    const bySorts: number[][] = [
        insertionSort(data),
        bubbleSort(data),
        quickSort(data),
        mergeSort(data),
    ];
    for (let i: number = 0; i < bySorts.length; i++) {
        check(isSorted(bySorts[i]), "sorted-" + i);
        check(arraysEqual(bySorts[i], expected), "matches-expected-" + i);
        check(bySorts[i].length === data.length, "length-preserved-" + i);
    }
    // The original array must be untouched (the sorts copy).
    check(data[0] === 5 && data.length === 12, "input-unmodified");

    // Every sort must agree with every other.
    for (let i: number = 1; i < bySorts.length; i++) {
        check(arraysEqual(bySorts[0], bySorts[i]), "sorts-agree-" + i);
    }

    // An already-sorted and a reversed input.
    check(arraysEqual(insertionSort([1, 2, 3]), [1, 2, 3]), "already-sorted");
    check(arraysEqual(mergeSort([3, 2, 1]), [1, 2, 3]), "reversed");
    check(arraysEqual(quickSort([]), []), "empty-sort");
    check(arraysEqual(bubbleSort([42]), [42]), "single-sort");
}

function testBinarySearch(): void {
    const sorted: number[] = [1, 3, 5, 7, 9, 11, 13, 15];
    check(binarySearch(sorted, 1) === 0, "bs-first");
    check(binarySearch(sorted, 15) === 7, "bs-last");
    check(binarySearch(sorted, 9) === 4, "bs-mid");
    check(binarySearch(sorted, 8) === -1, "bs-missing");
    check(binarySearch(sorted, 0) === -1, "bs-below");
    check(binarySearch(sorted, 100) === -1, "bs-above");
    check(binarySearch([], 5) === -1, "bs-empty");
}

// ---- generic Stack<T> (LIFO) over an internal array ----
class Stack<T> {
    private items: T[] = [];

    push(x: T): void { this.items.push(x); }
    pop(): T { return this.items.pop(); }
    peek(): T { return this.items[this.items.length - 1]; }
    size(): number { return this.items.length; }
    isEmpty(): boolean { return this.items.length === 0; }
}

// ---- generic Queue<T> (FIFO) over an internal array ----
class Queue<T> {
    private items: T[] = [];

    enqueue(x: T): void { this.items.push(x); }
    dequeue(): T { return this.items.shift(); }
    size(): number { return this.items.length; }
    isEmpty(): boolean { return this.items.length === 0; }
}

function testStackQueue(): void {
    const s: Stack<number> = new Stack<number>();
    check(s.isEmpty(), "stack-empty");
    s.push(1); s.push(2); s.push(3);
    check(s.size() === 3, "stack-size");
    check(s.peek() === 3, "stack-peek");
    check(s.pop() === 3, "stack-pop3");
    check(s.pop() === 2, "stack-pop2");
    check(s.size() === 1, "stack-size1");
    check(!s.isEmpty(), "stack-nonempty");

    // Reverse a sequence by pushing then popping.
    const src: number[] = [10, 20, 30, 40];
    const st: Stack<number> = new Stack<number>();
    for (let i: number = 0; i < src.length; i++) { st.push(src[i]); }
    const rev: number[] = [];
    while (!st.isEmpty()) { rev.push(st.pop()); }
    check(arraysEqual(rev, [40, 30, 20, 10]), "stack-reverse");

    const q: Queue<number> = new Queue<number>();
    q.enqueue(1); q.enqueue(2); q.enqueue(3);
    check(q.size() === 3, "queue-size");
    check(q.dequeue() === 1, "queue-deq1");
    check(q.dequeue() === 2, "queue-deq2");
    q.enqueue(4);
    check(q.dequeue() === 3, "queue-deq3");
    check(q.dequeue() === 4, "queue-deq4");
    check(q.isEmpty(), "queue-empty");
}

// ---- singly linked list ----
class ListNode {
    value: number;
    next: ListNode;

    constructor(value: number) {
        this.value = value;
        this.next = null;
    }
}

class LinkedList {
    private head: ListNode = null;
    private count: number = 0;

    append(value: number): void {
        const node: ListNode = new ListNode(value);
        if (this.head === null) {
            this.head = node;
        } else {
            let cur: ListNode = this.head;
            while (cur.next !== null) { cur = cur.next; }
            cur.next = node;
        }
        this.count = this.count + 1;
    }

    length(): number { return this.count; }

    get(index: number): number {
        let cur: ListNode = this.head;
        let i: number = 0;
        while (cur !== null && i < index) { cur = cur.next; i++; }
        if (cur === null) { return -1; }
        return cur.value;
    }

    toArray(): number[] {
        const out: number[] = [];
        let cur: ListNode = this.head;
        while (cur !== null) { out.push(cur.value); cur = cur.next; }
        return out;
    }

    // Reverse the list in place (pointer surgery).
    reverse(): void {
        let prev: ListNode = null;
        let cur: ListNode = this.head;
        while (cur !== null) {
            const nxt: ListNode = cur.next;
            cur.next = prev;
            prev = cur;
            cur = nxt;
        }
        this.head = prev;
    }

    sum(): number {
        let total: number = 0;
        let cur: ListNode = this.head;
        while (cur !== null) { total = total + cur.value; cur = cur.next; }
        return total;
    }
}

function testLinkedList(): void {
    const list: LinkedList = new LinkedList();
    check(list.length() === 0, "list-empty-len");
    for (let i: number = 1; i <= 5; i++) { list.append(i * 10); }
    check(list.length() === 5, "list-len");
    check(arraysEqual(list.toArray(), [10, 20, 30, 40, 50]), "list-toarray");
    check(list.get(0) === 10, "list-get0");
    check(list.get(4) === 50, "list-get4");
    check(list.get(9) === -1, "list-get-oob");
    check(list.sum() === 150, "list-sum");
    list.reverse();
    check(arraysEqual(list.toArray(), [50, 40, 30, 20, 10]), "list-reversed");
    check(list.get(0) === 50, "list-get0-rev");
    check(list.sum() === 150, "list-sum-after-reverse");
}

// ---- binary search tree ----
class TreeNode {
    value: number;
    left: TreeNode;
    right: TreeNode;

    constructor(value: number) {
        this.value = value;
        this.left = null;
        this.right = null;
    }
}

class BST {
    private root: TreeNode = null;
    private size: number = 0;

    insert(value: number): void {
        this.root = this.insertInto(this.root, value);
    }

    // Recursive insert that returns the (possibly new) subtree root.
    private insertInto(node: TreeNode, value: number): TreeNode {
        if (node === null) {
            this.size = this.size + 1;
            return new TreeNode(value);
        }
        if (value < node.value) { node.left = this.insertInto(node.left, value); }
        else if (value > node.value) { node.right = this.insertInto(node.right, value); }
        // equal -> ignore (set semantics)
        return node;
    }

    contains(value: number): boolean {
        let cur: TreeNode = this.root;
        while (cur !== null) {
            if (value === cur.value) { return true; }
            if (value < cur.value) { cur = cur.left; } else { cur = cur.right; }
        }
        return false;
    }

    count(): number { return this.size; }

    // In-order traversal collecting values into the accumulator array.
    private inOrderInto(node: TreeNode, acc: number[]): void {
        if (node === null) { return; }
        this.inOrderInto(node.left, acc);
        acc.push(node.value);
        this.inOrderInto(node.right, acc);
    }

    inOrder(): number[] {
        const acc: number[] = [];
        this.inOrderInto(this.root, acc);
        return acc;
    }

    height(): number {
        return this.heightOf(this.root);
    }

    private heightOf(node: TreeNode): number {
        if (node === null) { return 0; }
        return 1 + Math.max(this.heightOf(node.left), this.heightOf(node.right));
    }
}

function testBST(): void {
    const tree: BST = new BST();
    const values: number[] = [8, 3, 10, 1, 6, 14, 4, 7, 13];
    for (let i: number = 0; i < values.length; i++) { tree.insert(values[i]); }
    // Inserting a duplicate must not grow the tree.
    tree.insert(8);
    tree.insert(6);
    check(tree.count() === 9, "bst-count");
    // In-order of a BST is the sorted unique set.
    check(arraysEqual(tree.inOrder(), [1, 3, 4, 6, 7, 8, 10, 13, 14]), "bst-inorder-sorted");
    check(tree.contains(7), "bst-contains-7");
    check(tree.contains(14), "bst-contains-14");
    check(!tree.contains(2), "bst-missing-2");
    check(!tree.contains(100), "bst-missing-100");
    // The height of this particular tree is 4 (8 -> 3 -> 6 -> 7 or ... -> 4).
    check(tree.height() === 4, "bst-height");
}

// ---- a binary min-heap over an array ----
class MinHeap {
    private data: number[] = [];

    size(): number { return this.data.length; }

    push(value: number): void {
        this.data.push(value);
        let i: number = this.data.length - 1;
        while (i > 0) {
            const parent: number = Math.floor((i - 1) / 2);
            if (this.data[parent] <= this.data[i]) { break; }
            this.swap(parent, i);
            i = parent;
        }
    }

    pop(): number {
        const n: number = this.data.length;
        if (n === 0) { return -1; }
        const top: number = this.data[0];
        const last: number = this.data.pop();
        if (n > 1) {
            this.data[0] = last;
            this.siftDown(0);
        }
        return top;
    }

    private siftDown(start: number): void {
        let i: number = start;
        const n: number = this.data.length;
        while (true) {
            const l: number = 2 * i + 1;
            const r: number = 2 * i + 2;
            let smallest: number = i;
            if (l < n && this.data[l] < this.data[smallest]) { smallest = l; }
            if (r < n && this.data[r] < this.data[smallest]) { smallest = r; }
            if (smallest === i) { break; }
            this.swap(smallest, i);
            i = smallest;
        }
    }

    private swap(a: number, b: number): void {
        const t: number = this.data[a];
        this.data[a] = this.data[b];
        this.data[b] = t;
    }
}

function testMinHeap(): void {
    const heap: MinHeap = new MinHeap();
    const input: number[] = [5, 3, 8, 1, 9, 2, 7, 4, 6, 0];
    for (let i: number = 0; i < input.length; i++) { heap.push(input[i]); }
    check(heap.size() === 10, "heap-size");
    // Popping a min-heap yields the values in ascending order (heapsort).
    const out: number[] = [];
    while (heap.size() > 0) { out.push(heap.pop()); }
    check(arraysEqual(out, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]), "heap-sorted-out");
    check(heap.pop() === -1, "heap-empty-pop");
}

function main(): number {
    testSorting();
    testBinarySearch();
    testStackQueue();
    testLinkedList();
    testBST();
    testMinHeap();
    return failures;
}
