<?php
// PHP subset BIG test 1 -- Data structures & algorithms.
//
// Theme: classic sorting and searching algorithms plus hand-built container
// types (Stack, Queue, singly linked list) and a dictionary histogram. Every
// result is compared to an expected value; the program counts failures in
// $fails and ends with exit($fails), so a metacompiler run exits 0 exactly when
// every check passes. The interpreter (php-interpreter.abnf) and the LLVM-IR
// compiler (php-to-llvm-ir.abnf) run the same file and must agree byte for byte.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// ----- array helpers -----
// Arrays cannot be compared with === (that would compare identity), so equality
// is checked element by element.
function arrEq($a, $b) {
    if (count($a) !== count($b)) {
        return false;
    }
    $n = count($a);
    for ($i = 0; $i < $n; $i++) {
        if ($a[$i] !== $b[$i]) {
            return false;
        }
    }
    return true;
}

function checkArr($name, $got, $want) {
    global $fails;
    if (!arrEq($got, $want)) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

function copyArr($a) {
    $out = [];
    foreach ($a as $x) {
        $out[] = $x;
    }
    return $out;
}

function isSorted($a) {
    $n = count($a);
    for ($i = 1; $i < $n; $i++) {
        if ($a[$i - 1] > $a[$i]) {
            return false;
        }
    }
    return true;
}

function arraySum($a) {
    $s = 0;
    foreach ($a as $x) {
        $s += $x;
    }
    return $s;
}

function arrayMin($a) {
    $m = $a[0];
    $n = count($a);
    for ($i = 1; $i < $n; $i++) {
        if ($a[$i] < $m) {
            $m = $a[$i];
        }
    }
    return $m;
}

function arrayMax($a) {
    $m = $a[0];
    $n = count($a);
    for ($i = 1; $i < $n; $i++) {
        if ($a[$i] > $m) {
            $m = $a[$i];
        }
    }
    return $m;
}

function arrayReverse($a) {
    $out = [];
    $n = count($a);
    for ($i = $n - 1; $i >= 0; $i--) {
        $out[] = $a[$i];
    }
    return $out;
}

// ----- sorting algorithms (each returns a fresh, sorted array) -----
function bubbleSort($a) {
    $arr = copyArr($a);
    $n = count($arr);
    for ($i = 0; $i < $n - 1; $i++) {
        for ($j = 0; $j < $n - 1 - $i; $j++) {
            if ($arr[$j] > $arr[$j + 1]) {
                $tmp = $arr[$j];
                $arr[$j] = $arr[$j + 1];
                $arr[$j + 1] = $tmp;
            }
        }
    }
    return $arr;
}

function insertionSort($a) {
    $arr = copyArr($a);
    $n = count($arr);
    for ($i = 1; $i < $n; $i++) {
        $key = $arr[$i];
        $j = $i - 1;
        while ($j >= 0 && $arr[$j] > $key) {
            $arr[$j + 1] = $arr[$j];
            $j--;
        }
        $arr[$j + 1] = $key;
    }
    return $arr;
}

function selectionSort($a) {
    $arr = copyArr($a);
    $n = count($arr);
    for ($i = 0; $i < $n - 1; $i++) {
        $sel = $i;
        for ($j = $i + 1; $j < $n; $j++) {
            if ($arr[$j] < $arr[$sel]) {
                $sel = $j;
            }
        }
        if ($sel !== $i) {
            $tmp = $arr[$i];
            $arr[$i] = $arr[$sel];
            $arr[$sel] = $tmp;
        }
    }
    return $arr;
}

function quickSort($a) {
    $n = count($a);
    if ($n <= 1) {
        return copyArr($a);
    }
    $pivot = $a[0];
    $less = [];
    $more = [];
    for ($i = 1; $i < $n; $i++) {
        if ($a[$i] < $pivot) {
            $less[] = $a[$i];
        } else {
            $more[] = $a[$i];
        }
    }
    $out = [];
    foreach (quickSort($less) as $x) {
        $out[] = $x;
    }
    $out[] = $pivot;
    foreach (quickSort($more) as $x) {
        $out[] = $x;
    }
    return $out;
}

function merge($l, $r) {
    $out = [];
    $i = 0;
    $j = 0;
    $nl = count($l);
    $nr = count($r);
    while ($i < $nl && $j < $nr) {
        if ($l[$i] <= $r[$j]) {
            $out[] = $l[$i];
            $i++;
        } else {
            $out[] = $r[$j];
            $j++;
        }
    }
    while ($i < $nl) {
        $out[] = $l[$i];
        $i++;
    }
    while ($j < $nr) {
        $out[] = $r[$j];
        $j++;
    }
    return $out;
}

function mergeSort($a) {
    $n = count($a);
    if ($n <= 1) {
        return copyArr($a);
    }
    $mid = intdiv($n, 2);
    $left = [];
    $right = [];
    for ($i = 0; $i < $mid; $i++) {
        $left[] = $a[$i];
    }
    for ($i = $mid; $i < $n; $i++) {
        $right[] = $a[$i];
    }
    return merge(mergeSort($left), mergeSort($right));
}

// Binary search over a sorted array; returns the index or -1.
function binarySearch($a, $target) {
    $lo = 0;
    $hi = count($a) - 1;
    while ($lo <= $hi) {
        $mid = intdiv($lo + $hi, 2);
        if ($a[$mid] === $target) {
            return $mid;
        }
        if ($a[$mid] < $target) {
            $lo = $mid + 1;
        } else {
            $hi = $mid - 1;
        }
    }
    return -1;
}

// ----- exercise the sorts -----
$data = [5, 2, 8, 1, 9, 3, 7, 4, 6, 0];
$expected = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];

checkArr("bubble", bubbleSort($data), $expected);
checkArr("insertion", insertionSort($data), $expected);
checkArr("selection", selectionSort($data), $expected);
checkArr("quick", quickSort($data), $expected);
checkArr("merge", mergeSort($data), $expected);

// The original array must be untouched (the sorts copy first).
checkArr("original intact", $data, [5, 2, 8, 1, 9, 3, 7, 4, 6, 0]);

// A trickier input with duplicates and negatives.
$dupes = [3, -1, 3, 0, -1, 5, 3, -2, 5];
$dupesSorted = [-2, -1, -1, 0, 3, 3, 3, 5, 5];
checkArr("bubble dupes", bubbleSort($dupes), $dupesSorted);
checkArr("insertion dupes", insertionSort($dupes), $dupesSorted);
checkArr("quick dupes", quickSort($dupes), $dupesSorted);
checkArr("merge dupes", mergeSort($dupes), $dupesSorted);

// All five sorts must agree with each other on a third input.
$rnd = [42, 7, 42, 13, 99, 1, 56, 7, 23, 8, 8, 100];
$b1 = bubbleSort($rnd);
check("sorts agree 1", arrEq($b1, insertionSort($rnd)), true);
check("sorts agree 2", arrEq($b1, selectionSort($rnd)), true);
check("sorts agree 3", arrEq($b1, quickSort($rnd)), true);
check("sorts agree 4", arrEq($b1, mergeSort($rnd)), true);
check("result sorted", isSorted($b1), true);
check("count preserved", count($b1), 12);
check("sum preserved", arraySum($b1), arraySum($rnd));

// ----- reductions -----
check("sum", arraySum($expected), 45);
check("min", arrayMin($dupes), -2);
check("max", arrayMax($dupes), 5);
checkArr("reverse", arrayReverse([1, 2, 3, 4]), [4, 3, 2, 1]);

// ----- binary search -----
$sorted = mergeSort($rnd);
check("find 42", $sorted[binarySearch($sorted, 42)], 42);
check("find 100", $sorted[binarySearch($sorted, 100)], 100);
check("find 1", $sorted[binarySearch($sorted, 1)], 1);
check("find missing", binarySearch($sorted, 1000), -1);
check("find missing neg", binarySearch($sorted, -5), -1);
$seq = [10, 20, 30, 40, 50, 60, 70];
check("bsearch 30", binarySearch($seq, 30), 2);
check("bsearch 70", binarySearch($seq, 70), 6);
check("bsearch 10", binarySearch($seq, 10), 0);
check("bsearch gap", binarySearch($seq, 35), -1);

// ----- Stack (LIFO): append to grow, rebuild without the last item to pop -----
class Stack {
    public $items;
    public function __construct() {
        $this->items = [];
    }
    public function push($v) {
        $this->items[] = $v;
        return $this;
    }
    public function pop() {
        $n = count($this->items);
        $last = $this->items[$n - 1];
        $rest = [];
        for ($i = 0; $i < $n - 1; $i++) {
            $rest[] = $this->items[$i];
        }
        $this->items = $rest;
        return $last;
    }
    public function peek() {
        return $this->items[count($this->items) - 1];
    }
    public function size() {
        return count($this->items);
    }
    public function isEmpty() {
        return count($this->items) === 0;
    }
}

$st = new Stack();
$st->push(1)->push(2)->push(3);
check("stack size", $st->size(), 3);
check("stack peek", $st->peek(), 3);
check("stack pop a", $st->pop(), 3);
check("stack pop b", $st->pop(), 2);
$st->push(42)->push(7);
check("stack size 2", $st->size(), 3);
check("stack pop c", $st->pop(), 7);
check("stack pop d", $st->pop(), 42);
check("stack pop e", $st->pop(), 1);
check("stack empty", $st->isEmpty(), true);

// Reverse a sequence by pushing then popping.
$src = [1, 2, 3, 4, 5];
$rev = new Stack();
foreach ($src as $v) {
    $rev->push($v);
}
$reversed = [];
while (!$rev->isEmpty()) {
    $reversed[] = $rev->pop();
}
checkArr("stack reverse", $reversed, [5, 4, 3, 2, 1]);

// ----- Queue (FIFO): append to grow, advance a head index to dequeue -----
class Queue {
    public $items;
    public $head;
    public function __construct() {
        $this->items = [];
        $this->head = 0;
    }
    public function enqueue($v) {
        $this->items[] = $v;
        return $this;
    }
    public function dequeue() {
        $v = $this->items[$this->head];
        $this->head++;
        return $v;
    }
    public function size() {
        return count($this->items) - $this->head;
    }
    public function isEmpty() {
        return $this->size() === 0;
    }
}

$q = new Queue();
$q->enqueue(10)->enqueue(20)->enqueue(30);
check("queue size", $q->size(), 3);
check("queue deq a", $q->dequeue(), 10);
check("queue deq b", $q->dequeue(), 20);
$q->enqueue(40);
check("queue deq c", $q->dequeue(), 30);
check("queue deq d", $q->dequeue(), 40);
check("queue empty", $q->isEmpty(), true);

// Drain a queue in FIFO order.
$q2 = new Queue();
foreach ([100, 200, 300] as $v) {
    $q2->enqueue($v);
}
$drained = [];
while (!$q2->isEmpty()) {
    $drained[] = $q2->dequeue();
}
checkArr("queue order", $drained, [100, 200, 300]);

// ----- singly linked list -----
class ListNode {
    public $value;
    public $next;
    public function __construct($v) {
        $this->value = $v;
        $this->next = null;
    }
}

class LinkedList {
    public $head;
    public $len;
    public function __construct() {
        $this->head = null;
        $this->len = 0;
    }
    public function prepend($v) {
        $node = new ListNode($v);
        $node->next = $this->head;
        $this->head = $node;
        $this->len++;
        return $this;
    }
    public function append($v) {
        $node = new ListNode($v);
        if ($this->head === null) {
            $this->head = $node;
        } else {
            $cur = $this->head;
            while ($cur->next !== null) {
                $cur = $cur->next;
            }
            $cur->next = $node;
        }
        $this->len++;
        return $this;
    }
    public function toArray() {
        $out = [];
        $cur = $this->head;
        while ($cur !== null) {
            $out[] = $cur->value;
            $cur = $cur->next;
        }
        return $out;
    }
    public function sum() {
        $s = 0;
        $cur = $this->head;
        while ($cur !== null) {
            $s += $cur->value;
            $cur = $cur->next;
        }
        return $s;
    }
    public function contains($v) {
        $cur = $this->head;
        while ($cur !== null) {
            if ($cur->value === $v) {
                return true;
            }
            $cur = $cur->next;
        }
        return false;
    }
    // Reverse the list in place, returning $this.
    public function reverse() {
        $prev = null;
        $cur = $this->head;
        while ($cur !== null) {
            $nxt = $cur->next;
            $cur->next = $prev;
            $prev = $cur;
            $cur = $nxt;
        }
        $this->head = $prev;
        return $this;
    }
}

$list = new LinkedList();
$list->append(1)->append(2)->append(3)->append(4);
checkArr("list append", $list->toArray(), [1, 2, 3, 4]);
check("list len", $list->len, 4);
check("list sum", $list->sum(), 10);
check("list contains", $list->contains(3), true);
check("list not contains", $list->contains(9), false);

$list->prepend(0);
checkArr("list prepend", $list->toArray(), [0, 1, 2, 3, 4]);
check("list len 2", $list->len, 5);

$list->reverse();
checkArr("list reverse", $list->toArray(), [4, 3, 2, 1, 0]);
check("list sum after reverse", $list->sum(), 10);

// ----- dictionary histogram -----
function dictHas($d, $k) {
    return in_array($k, array_keys($d));
}

// Count occurrences of each value in a non-empty array, returning an ordered map.
function countFreq($arr) {
    $n = count($arr);
    $freq = [$arr[0] => 0];
    for ($i = 0; $i < $n; $i++) {
        $x = $arr[$i];
        if (dictHas($freq, $x)) {
            $freq[$x] = $freq[$x] + 1;
        } else {
            $freq[$x] = 1;
        }
    }
    return $freq;
}

$freq = countFreq([1, 2, 2, 3, 3, 3, 1, 2]);
check("freq of 1", $freq[1], 2);
check("freq of 2", $freq[2], 3);
check("freq of 3", $freq[3], 3);
check("freq distinct", count($freq), 3);

// The frequencies must sum back to the number of elements.
$fsum = 0;
foreach ($freq as $val => $cnt) {
    $fsum += $cnt;
}
check("freq total", $fsum, 8);
check("freq has key", dictHas($freq, 2), true);
check("freq missing key", dictHas($freq, 9), false);

// ----- done -----
if ($fails === 0) {
    echo "php-test-big-1 (data structures & algorithms) passed\n";
}
exit($fails);
