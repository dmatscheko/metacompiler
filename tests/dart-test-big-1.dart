// Dart self-checking test 1/4 for the metacompiler: DATA STRUCTURES + ALGORITHMS.
//
// Sorting (bubble / insertion / selection / quicksort), binary search (iterative and
// recursive), merging sorted runs, and hand-built container classes (a Stack and a Queue
// backed by a growable List with an explicit size cursor, and a singly linked list built
// from Node objects). Every result is checked against an expected value; int main() returns
// the number of failures, so the process exit code is 0 on success and the goja and frozen
// engines must agree byte-for-byte.

// ---------- list helpers ----------

List<int> copyList(List<int> xs) {
  List<int> out = [];
  for (int i = 0; i < xs.length; i = i + 1) {
    out.add(xs[i]);
  }
  return out;
}

bool listEq(List<int> a, List<int> b) {
  if (a.length != b.length) return false;
  for (int i = 0; i < a.length; i = i + 1) {
    if (a[i] != b[i]) return false;
  }
  return true;
}

bool isSorted(List<int> a) {
  for (int i = 1; i < a.length; i = i + 1) {
    if (a[i - 1] > a[i]) return false;
  }
  return true;
}

void swap(List<int> a, int i, int j) {
  int t = a[i];
  a[i] = a[j];
  a[j] = t;
}

// ---------- sorting ----------

List<int> bubbleSort(List<int> input) {
  List<int> a = copyList(input);
  int n = a.length;
  for (int i = 0; i < n - 1; i = i + 1) {
    for (int j = 0; j < n - 1 - i; j = j + 1) {
      if (a[j] > a[j + 1]) {
        swap(a, j, j + 1);
      }
    }
  }
  return a;
}

List<int> insertionSort(List<int> input) {
  List<int> a = copyList(input);
  for (int i = 1; i < a.length; i = i + 1) {
    int key = a[i];
    int j = i - 1;
    while (j >= 0 && a[j] > key) {
      a[j + 1] = a[j];
      j = j - 1;
    }
    a[j + 1] = key;
  }
  return a;
}

List<int> selectionSort(List<int> input) {
  List<int> a = copyList(input);
  int n = a.length;
  for (int i = 0; i < n - 1; i = i + 1) {
    int min = i;
    for (int j = i + 1; j < n; j = j + 1) {
      if (a[j] < a[min]) min = j;
    }
    if (min != i) swap(a, i, min);
  }
  return a;
}

// In-place Lomuto quicksort over a[lo..hi].
void quickSortRange(List<int> a, int lo, int hi) {
  if (lo >= hi) return;
  int pivot = a[hi];
  int i = lo - 1;
  for (int j = lo; j < hi; j = j + 1) {
    if (a[j] <= pivot) {
      i = i + 1;
      swap(a, i, j);
    }
  }
  swap(a, i + 1, hi);
  int p = i + 1;
  quickSortRange(a, lo, p - 1);
  quickSortRange(a, p + 1, hi);
}

List<int> quickSort(List<int> input) {
  List<int> a = copyList(input);
  quickSortRange(a, 0, a.length - 1);
  return a;
}

// Merge two already-sorted lists into one sorted list.
List<int> mergeSorted(List<int> a, List<int> b) {
  List<int> out = [];
  int i = 0;
  int j = 0;
  while (i < a.length && j < b.length) {
    if (a[i] <= b[j]) {
      out.add(a[i]);
      i = i + 1;
    } else {
      out.add(b[j]);
      j = j + 1;
    }
  }
  while (i < a.length) {
    out.add(a[i]);
    i = i + 1;
  }
  while (j < b.length) {
    out.add(b[j]);
    j = j + 1;
  }
  return out;
}

// ---------- searching ----------

int binarySearch(List<int> a, int target) {
  int lo = 0;
  int hi = a.length - 1;
  while (lo <= hi) {
    int mid = (lo + hi) ~/ 2;
    if (a[mid] == target) return mid;
    if (a[mid] < target) {
      lo = mid + 1;
    } else {
      hi = mid - 1;
    }
  }
  return -1;
}

int binarySearchRec(List<int> a, int target, int lo, int hi) {
  if (lo > hi) return -1;
  int mid = (lo + hi) ~/ 2;
  if (a[mid] == target) return mid;
  if (a[mid] < target) return binarySearchRec(a, target, mid + 1, hi);
  return binarySearchRec(a, target, lo, mid - 1);
}

// ---------- containers ----------

// A stack backed by a growable list with an explicit size cursor (avoids removeLast).
class Stack {
  List<int> data;
  int size;
  Stack() {
    data = [];
    size = 0;
  }
  bool isEmptyStack() => size == 0;
  void push(int x) {
    if (size < data.length) {
      data[size] = x;
    } else {
      data.add(x);
    }
    size = size + 1;
  }
  int pop() {
    size = size - 1;
    return data[size];
  }
  int peek() => data[size - 1];
}

// A FIFO queue: enqueue appends, dequeue advances a head cursor.
class Queue {
  List<int> data;
  int head;
  Queue() {
    data = [];
    head = 0;
  }
  bool isEmptyQueue() => head >= data.length;
  void enqueue(int x) {
    data.add(x);
  }
  int dequeue() {
    int v = data[head];
    head = head + 1;
    return v;
  }
  int count() => data.length - head;
}

// A singly linked list built from Node objects.
class Node {
  int value;
  Node next;
  Node(this.value) {
    next = null;
  }
}

class LinkedList {
  Node head;
  int count;
  LinkedList() {
    head = null;
    count = 0;
  }
  void pushFront(int v) {
    Node n = Node(v);
    n.next = head;
    head = n;
    count = count + 1;
  }
  int sum() {
    int s = 0;
    Node cur = head;
    while (cur != null) {
      s = s + cur.value;
      cur = cur.next;
    }
    return s;
  }
  List<int> toList() {
    List<int> out = [];
    Node cur = head;
    while (cur != null) {
      out.add(cur.value);
      cur = cur.next;
    }
    return out;
  }
  void reverse() {
    Node prev = null;
    Node cur = head;
    while (cur != null) {
      Node nxt = cur.next;
      cur.next = prev;
      prev = cur;
      cur = nxt;
    }
    head = prev;
  }
}

int main() {
  int failures = 0;

  List<int> data = [5, 2, 9, 1, 5, 6, 3, 8, 7, 0, 4];
  List<int> expected = [0, 1, 2, 3, 4, 5, 5, 6, 7, 8, 9];

  // ---------- sorting: every algorithm reproduces the same sorted order ----------
  List<int> bs = bubbleSort(data);
  if (!isSorted(bs)) failures = failures + 1;
  if (!listEq(bs, expected)) failures = failures + 1;

  List<int> is1 = insertionSort(data);
  if (!listEq(is1, expected)) failures = failures + 1;

  List<int> ss = selectionSort(data);
  if (!listEq(ss, expected)) failures = failures + 1;

  List<int> qs = quickSort(data);
  if (!listEq(qs, expected)) failures = failures + 1;

  // the original list is untouched (algorithms sort copies)
  if (data[0] != 5) failures = failures + 1;
  if (data.length != 11) failures = failures + 1;

  // sorting an already-sorted and a reversed list
  if (!listEq(quickSort([1, 2, 3, 4, 5]), [1, 2, 3, 4, 5])) failures = failures + 1;
  if (!listEq(bubbleSort([5, 4, 3, 2, 1]), [1, 2, 3, 4, 5])) failures = failures + 1;
  if (!listEq(insertionSort([42]), [42])) failures = failures + 1;

  // ---------- merging sorted runs ----------
  List<int> merged = mergeSorted([1, 4, 7, 9], [2, 3, 8, 10, 11]);
  if (!listEq(merged, [1, 2, 3, 4, 7, 8, 9, 10, 11])) failures = failures + 1;
  if (merged.length != 9) failures = failures + 1;

  // ---------- binary search on the sorted array ----------
  if (binarySearch(expected, 7) != 8) failures = failures + 1;
  if (binarySearch(expected, 0) != 0) failures = failures + 1;
  if (binarySearch(expected, 9) != 10) failures = failures + 1;
  if (binarySearch(expected, 42) != -1) failures = failures + 1;
  if (binarySearchRec(expected, 6, 0, expected.length - 1) != 7) failures = failures + 1;
  if (binarySearchRec(expected, 100, 0, expected.length - 1) != -1) failures = failures + 1;

  // ---------- Stack (LIFO) ----------
  Stack st = Stack();
  if (!st.isEmptyStack()) failures = failures + 1;
  st.push(10);
  st.push(20);
  st.push(30);
  if (st.peek() != 30) failures = failures + 1;
  if (st.pop() != 30) failures = failures + 1;
  if (st.pop() != 20) failures = failures + 1;
  st.push(40); // reuses freed slot
  if (st.pop() != 40) failures = failures + 1;
  if (st.pop() != 10) failures = failures + 1;
  if (!st.isEmptyStack()) failures = failures + 1;

  // stack-based list reversal
  Stack rev = Stack();
  for (int i = 1; i <= 5; i = i + 1) {
    rev.push(i);
  }
  List<int> reversed = [];
  while (!rev.isEmptyStack()) {
    reversed.add(rev.pop());
  }
  if (!listEq(reversed, [5, 4, 3, 2, 1])) failures = failures + 1;

  // ---------- Queue (FIFO) ----------
  Queue q = Queue();
  if (!q.isEmptyQueue()) failures = failures + 1;
  q.enqueue(1);
  q.enqueue(2);
  q.enqueue(3);
  if (q.count() != 3) failures = failures + 1;
  if (q.dequeue() != 1) failures = failures + 1;
  if (q.dequeue() != 2) failures = failures + 1;
  q.enqueue(4);
  if (q.dequeue() != 3) failures = failures + 1;
  if (q.dequeue() != 4) failures = failures + 1;
  if (!q.isEmptyQueue()) failures = failures + 1;

  // ---------- singly linked list ----------
  LinkedList ll = LinkedList();
  ll.pushFront(3);
  ll.pushFront(2);
  ll.pushFront(1); // list is now 1 -> 2 -> 3
  if (ll.count != 3) failures = failures + 1;
  if (!listEq(ll.toList(), [1, 2, 3])) failures = failures + 1;
  if (ll.sum() != 6) failures = failures + 1;
  ll.reverse(); // now 3 -> 2 -> 1
  if (!listEq(ll.toList(), [3, 2, 1])) failures = failures + 1;
  if (ll.sum() != 6) failures = failures + 1;
  ll.pushFront(9); // 9 -> 3 -> 2 -> 1
  if (ll.toList()[0] != 9) failures = failures + 1;
  if (ll.count != 4) failures = failures + 1;

  print('dart-test-big-1 (data structures) finished with $failures failures');
  return failures;
}
