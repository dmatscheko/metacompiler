/* Java subset self test (big 1): data structures and algorithms.
 *
 * Builds real containers and algorithms out of the implemented subset only:
 *   - Stack: a singly linked LIFO stack (push/pop/peek/isEmpty/size)
 *   - Queue: a singly linked FIFO queue with head+tail (enqueue/dequeue/size)
 *   - IntList: a singly linked list with addFront/addBack/get/sum/in-place reverse
 *   - BST: an unbalanced binary search tree (insert/contains/height, inorder walk
 *          serialised to a String so the traversal order can be asserted)
 *   - three in-place array sorts (insertion, quicksort/Lomuto, heapsort) that must
 *     all agree, plus binary search over the sorted result
 * Dynamic storage is done with linked nodes and fixed array literals only; the sorts
 * work in place on array literals (whose length is exact), so nothing depends on a
 * sized `new int[n]` allocation. Every result is checked against an independently
 * computed expectation. Main.main exits with the number of failed checks, so the run
 * exits 0 exactly when every structure and algorithm agrees. **/

class SNode {
    int val;
    SNode next;
    SNode(int v) { this.val = v; }
}

class Stack {
    SNode top;
    int n;

    Stack() { this.n = 0; }

    void push(int v) {
        SNode x = new SNode(v);
        x.next = this.top;
        this.top = x;
        this.n++;
    }

    int pop() {
        SNode x = this.top;
        this.top = x.next;
        this.n--;
        return x.val;
    }

    int peek() { return this.top.val; }
    boolean isEmpty() { return this.n == 0; }
    int size() { return this.n; }
}

class QNode {
    int val;
    QNode next;
    QNode(int v) { this.val = v; }
}

class Queue {
    QNode head;
    QNode tail;
    int n;

    Queue() { this.n = 0; }

    void enqueue(int v) {
        QNode x = new QNode(v);
        if (this.tail == null) {
            this.head = x;
            this.tail = x;
        } else {
            this.tail.next = x;
            this.tail = x;
        }
        this.n++;
    }

    int dequeue() {
        QNode x = this.head;
        this.head = x.next;
        if (this.head == null) {
            this.tail = null;
        }
        this.n--;
        return x.val;
    }

    int size() { return this.n; }
}

class LNode {
    int val;
    LNode next;
    LNode(int v) { this.val = v; }
}

class IntList {
    LNode head;
    int len;

    IntList() { this.len = 0; }

    void addFront(int v) {
        LNode x = new LNode(v);
        x.next = this.head;
        this.head = x;
        this.len++;
    }

    void addBack(int v) {
        LNode x = new LNode(v);
        if (this.head == null) {
            this.head = x;
        } else {
            LNode c = this.head;
            while (c.next != null) {
                c = c.next;
            }
            c.next = x;
        }
        this.len++;
    }

    int get(int i) {
        LNode c = this.head;
        int k = 0;
        while (k < i) {
            c = c.next;
            k++;
        }
        return c.val;
    }

    int sum() {
        int s = 0;
        LNode c = this.head;
        while (c != null) {
            s += c.val;
            c = c.next;
        }
        return s;
    }

    void reverse() {
        LNode prev = null;
        LNode c = this.head;
        while (c != null) {
            LNode nx = c.next;
            c.next = prev;
            prev = c;
            c = nx;
        }
        this.head = prev;
    }

    int size() { return this.len; }
}

class TNode {
    int key;
    TNode left;
    TNode right;
    TNode(int k) { this.key = k; }
}

class BST {
    TNode root;
    int n;

    BST() { this.n = 0; }

    void insert(int k) { this.root = this.insertAt(this.root, k); }

    TNode insertAt(TNode node, int k) {
        if (node == null) {
            this.n++;
            return new TNode(k);
        }
        if (k < node.key) {
            node.left = this.insertAt(node.left, k);
        } else if (k > node.key) {
            node.right = this.insertAt(node.right, k);
        }
        return node;                                    // duplicates ignored
    }

    boolean contains(int k) {
        TNode c = this.root;
        while (c != null) {
            if (k == c.key) {
                return true;
            }
            if (k < c.key) {
                c = c.left;
            } else {
                c = c.right;
            }
        }
        return false;
    }

    int height(TNode node) {
        if (node == null) {
            return 0;
        }
        return 1 + Math.max(this.height(node.left), this.height(node.right));
    }

    String inorder(TNode node) {                        // sorted keys, comma-terminated
        if (node == null) {
            return "";
        }
        return this.inorder(node.left) + node.key + "," + this.inorder(node.right);
    }

    int size() { return this.n; }
}

public class Main {
    static int fails = 0;

    static void check(String name, int got, int want) {
        if (got != want) {
            System.out.println("FAIL " + name + ": got " + got + " want " + want);
            Main.fails++;
        }
    }

    static void checkB(String name, boolean got, boolean want) {
        Main.check(name, got ? 1 : 0, want ? 1 : 0);
    }

    static void checkS(String name, String got, String want) {
        if (!got.equals(want)) {
            System.out.println("FAIL " + name + ": got " + got + " want " + want);
            Main.fails++;
        }
    }

    // ----- array helpers (operate on array literals; .length is exact) -----

    static boolean eqArr(int[] a, int[] b) {
        if (a.length != b.length) {
            return false;
        }
        for (int i = 0; i < a.length; i++) {
            if (a[i] != b[i]) {
                return false;
            }
        }
        return true;
    }

    static boolean isSorted(int[] a) {
        for (int i = 1; i < a.length; i++) {
            if (a[i - 1] > a[i]) {
                return false;
            }
        }
        return true;
    }

    static int sumArr(int[] a) {
        int s = 0;
        for (var v : a) {
            s += v;
        }
        return s;
    }

    static String joinArr(int[] a) {
        String s = "";
        for (int i = 0; i < a.length; i++) {
            s = s + a[i] + ",";
        }
        return s;
    }

    // ----- three in-place sorts -----

    static void insertionSort(int[] a) {
        for (int i = 1; i < a.length; i++) {
            int key = a[i];
            int j = i - 1;
            while (j >= 0 && a[j] > key) {
                a[j + 1] = a[j];
                j--;
            }
            a[j + 1] = key;
        }
    }

    static int partition(int[] a, int lo, int hi) {
        int pivot = a[hi];
        int i = lo - 1;
        for (int j = lo; j < hi; j++) {
            if (a[j] <= pivot) {
                i++;
                int t = a[i];
                a[i] = a[j];
                a[j] = t;
            }
        }
        int t = a[i + 1];
        a[i + 1] = a[hi];
        a[hi] = t;
        return i + 1;
    }

    static void quickSort(int[] a, int lo, int hi) {
        if (lo >= hi) {
            return;
        }
        int p = Main.partition(a, lo, hi);
        Main.quickSort(a, lo, p - 1);
        Main.quickSort(a, p + 1, hi);
    }

    static void siftDown(int[] a, int start, int end) {
        int root = start;
        while (root * 2 + 1 <= end) {
            int child = root * 2 + 1;
            int swap = root;
            if (a[swap] < a[child]) {
                swap = child;
            }
            if (child + 1 <= end && a[swap] < a[child + 1]) {
                swap = child + 1;
            }
            if (swap == root) {
                return;
            }
            int t = a[root];
            a[root] = a[swap];
            a[swap] = t;
            root = swap;
        }
    }

    static void heapSort(int[] a) {
        int n = a.length;
        for (int start = (n - 2) / 2; start >= 0; start--) {
            Main.siftDown(a, start, n - 1);
        }
        for (int end = n - 1; end > 0; end--) {
            int t = a[0];
            a[0] = a[end];
            a[end] = t;
            Main.siftDown(a, 0, end - 1);
        }
    }

    static int binarySearch(int[] a, int target) {
        int lo = 0, hi = a.length - 1;
        while (lo <= hi) {
            int mid = (lo + hi) / 2;
            if (a[mid] == target) {
                return mid;
            }
            if (a[mid] < target) {
                lo = mid + 1;
            } else {
                hi = mid - 1;
            }
        }
        return -1;
    }

    public static void main(String[] args) {
        // ----- Stack (linked LIFO) -----
        Stack st = new Stack();
        Main.checkB("stack empty", st.isEmpty(), true);
        for (int i = 1; i <= 10; i++) {
            st.push(i * i);
        }
        Main.check("stack size", st.size(), 10);
        Main.check("stack peek", st.peek(), 100);
        int popSum = 0;
        int firstPop = st.pop();
        popSum = firstPop;
        Main.check("stack LIFO first", firstPop, 100);
        while (!st.isEmpty()) {
            popSum += st.pop();
        }
        Main.check("stack drained sum", popSum, 385);    // 1+4+9+...+100
        Main.checkB("stack empty again", st.isEmpty(), true);

        // ----- Queue (linked FIFO) -----
        Queue q = new Queue();
        q.enqueue(1);
        q.enqueue(2);
        q.enqueue(3);
        Main.check("queue drop 1", q.dequeue(), 1);
        Main.check("queue drop 2", q.dequeue(), 2);
        q.enqueue(4);
        q.enqueue(5);
        q.enqueue(6);
        Main.check("queue size", q.size(), 4);
        int qseq = 0;
        while (q.size() > 0) {
            qseq = qseq * 10 + q.dequeue();
        }
        Main.check("queue FIFO order", qseq, 3456);

        // ----- IntList (in-place reverse) -----
        IntList list = new IntList();
        list.addBack(10);
        list.addBack(20);
        list.addBack(30);
        list.addFront(5);                                // 5,10,20,30
        Main.check("list size", list.size(), 4);
        Main.check("list get head", list.get(0), 5);
        Main.check("list get tail", list.get(3), 30);
        Main.check("list sum", list.sum(), 65);
        list.reverse();                                  // 30,20,10,5
        Main.check("list reversed head", list.get(0), 30);
        Main.check("list reversed tail", list.get(3), 5);
        Main.check("list sum after reverse", list.sum(), 65);

        // ----- BST (inorder yields a sorted key sequence) -----
        BST tree = new BST();
        int[] keys = new int[]{50, 30, 70, 20, 40, 60, 80, 30, 50};
        for (var kk : keys) {
            tree.insert(kk);                             // last two are duplicates
        }
        Main.check("bst size (dups ignored)", tree.size(), 7);
        Main.checkB("bst contains present", tree.contains(60), true);
        Main.checkB("bst contains absent", tree.contains(45), false);
        Main.check("bst height", tree.height(tree.root), 3);
        Main.checkS("bst inorder sorted", tree.inorder(tree.root), "20,30,40,50,60,70,80,");

        // ----- three sorts must agree (each on its own copy of the same data) -----
        int[] byInsertion = new int[]{9, 3, 7, 1, 8, 2, 6, 5, 4, 0, 3, 7};
        int[] byQuick = new int[]{9, 3, 7, 1, 8, 2, 6, 5, 4, 0, 3, 7};
        int[] byHeap = new int[]{9, 3, 7, 1, 8, 2, 6, 5, 4, 0, 3, 7};
        String expectSorted = "0,1,2,3,3,4,5,6,7,7,8,9,";
        int expectSum = 55;

        Main.insertionSort(byInsertion);
        Main.quickSort(byQuick, 0, byQuick.length - 1);
        Main.heapSort(byHeap);

        Main.checkB("insertion sorted", Main.isSorted(byInsertion), true);
        Main.checkB("quick sorted", Main.isSorted(byQuick), true);
        Main.checkB("heap sorted", Main.isSorted(byHeap), true);
        Main.checkS("insertion order", Main.joinArr(byInsertion), expectSorted);
        Main.checkB("insertion == quick", Main.eqArr(byInsertion, byQuick), true);
        Main.checkB("quick == heap", Main.eqArr(byQuick, byHeap), true);
        Main.check("sort preserves sum", Main.sumArr(byHeap), expectSum);
        Main.check("sort preserves length", byHeap.length, 12);

        // ----- binary search over the sorted result -----
        Main.check("search finds min", Main.binarySearch(byQuick, 0), 0);
        Main.check("search finds max", Main.binarySearch(byQuick, 9), 11);
        Main.checkB("search finds middle", Main.binarySearch(byQuick, 6) >= 0, true);
        Main.check("search misses", Main.binarySearch(byQuick, 42), -1);
        boolean allFound = true;
        int[] raw = new int[]{9, 3, 7, 1, 8, 2, 6, 5, 4, 0, 3, 7};
        for (var v : raw) {
            if (Main.binarySearch(byQuick, v) < 0) {
                allFound = false;
            }
        }
        Main.checkB("search finds every element", allFound, true);

        if (Main.fails == 0) {
            System.out.println("Java big-1 (data structures + algorithms) passed");
        }
        System.exit(Main.fails);
    }
}
