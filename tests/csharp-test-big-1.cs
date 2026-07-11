// C# subset self test -- BIG 1: data structures and algorithms.
//
// Theme: classic algorithms and hand-built containers. Sorting (bubble, insertion,
// selection, quicksort, merge sort), binary search, a List-backed stack and a circular
// queue, and a singly linked list with append/prepend/reverse. Every result is checked
// against an expected value; Program.Main returns the number of failed checks, so the
// metacompiler run exits 0 exactly when every algorithm agrees with its expectation.

using System;
using System.Collections.Generic;

namespace Demo
{
    // A LIFO stack backed by a List, reusing freed slots so no capacity is fixed.
    class IntStack
    {
        List<int> buf;
        int n;

        public IntStack()
        {
            this.buf = new List<int>();
            this.n = 0;
        }

        public void Push(int x)
        {
            if (this.n < this.buf.Count)
            {
                this.buf[this.n] = x;
            }
            else
            {
                this.buf.Add(x);
            }
            this.n = this.n + 1;
        }

        public int Pop()
        {
            this.n = this.n - 1;
            return this.buf[this.n];
        }

        public int Peek()
        {
            return this.buf[this.n - 1];
        }

        public int Size()
        {
            return this.n;
        }

        public bool IsEmpty()
        {
            return this.n == 0;
        }
    }

    // A circular FIFO queue over a pre-sized ring of List slots.
    class IntQueue
    {
        List<int> buf;
        int cap;
        int head;
        int tail;
        int n;

        public IntQueue(int capacity)
        {
            this.buf = new List<int>();
            this.cap = capacity;
            for (int i = 0; i < capacity; i++)
            {
                this.buf.Add(0);
            }
            this.head = 0;
            this.tail = 0;
            this.n = 0;
        }

        public void Enqueue(int x)
        {
            this.buf[this.tail] = x;
            this.tail = (this.tail + 1) % this.cap;
            this.n = this.n + 1;
        }

        public int Dequeue()
        {
            int v = this.buf[this.head];
            this.head = (this.head + 1) % this.cap;
            this.n = this.n - 1;
            return v;
        }

        public int Size()
        {
            return this.n;
        }
    }

    // A node in a singly linked list.
    class Node
    {
        public int Value;
        public Node Next;

        public Node(int v)
        {
            this.Value = v;
            this.Next = null;
        }
    }

    // A singly linked list built from Node cells.
    class IntList
    {
        Node head;
        int size;

        public IntList()
        {
            this.head = null;
            this.size = 0;
        }

        public void Prepend(int v)
        {
            Node node = new Node(v);
            node.Next = this.head;
            this.head = node;
            this.size = this.size + 1;
        }

        public void Append(int v)
        {
            Node node = new Node(v);
            if (this.head == null)
            {
                this.head = node;
            }
            else
            {
                Node cur = this.head;
                while (cur.Next != null)
                {
                    cur = cur.Next;
                }
                cur.Next = node;
            }
            this.size = this.size + 1;
        }

        public int Sum()
        {
            int s = 0;
            Node cur = this.head;
            while (cur != null)
            {
                s = s + cur.Value;
                cur = cur.Next;
            }
            return s;
        }

        public int At(int idx)
        {
            Node cur = this.head;
            int i = 0;
            while (i < idx)
            {
                cur = cur.Next;
                i = i + 1;
            }
            return cur.Value;
        }

        public void Reverse()
        {
            Node prev = null;
            Node cur = this.head;
            while (cur != null)
            {
                Node nxt = cur.Next;
                cur.Next = prev;
                prev = cur;
                cur = nxt;
            }
            this.head = prev;
        }

        public int Size()
        {
            return this.size;
        }
    }

    // Static List algorithms. Sorts return a fresh sorted List, leaving the input intact.
    class Algo
    {
        static List<int> Copy(List<int> a)
        {
            List<int> b = new List<int>();
            for (int i = 0; i < a.Count; i++)
            {
                b.Add(a[i]);
            }
            return b;
        }

        static void Swap(List<int> a, int i, int j)
        {
            int t = a[i];
            a[i] = a[j];
            a[j] = t;
        }

        static List<int> BubbleSort(List<int> src)
        {
            List<int> a = Algo.Copy(src);
            int len = a.Count;
            for (int i = 0; i < len - 1; i++)
            {
                for (int j = 0; j < len - 1 - i; j++)
                {
                    if (a[j] > a[j + 1])
                    {
                        Algo.Swap(a, j, j + 1);
                    }
                }
            }
            return a;
        }

        static List<int> InsertionSort(List<int> src)
        {
            List<int> a = Algo.Copy(src);
            for (int i = 1; i < a.Count; i++)
            {
                int key = a[i];
                int j = i - 1;
                while (j >= 0 && a[j] > key)
                {
                    a[j + 1] = a[j];
                    j = j - 1;
                }
                a[j + 1] = key;
            }
            return a;
        }

        static List<int> SelectionSort(List<int> src)
        {
            List<int> a = Algo.Copy(src);
            for (int i = 0; i < a.Count - 1; i++)
            {
                int min = i;
                for (int j = i + 1; j < a.Count; j++)
                {
                    if (a[j] < a[min])
                    {
                        min = j;
                    }
                }
                if (min != i)
                {
                    Algo.Swap(a, i, min);
                }
            }
            return a;
        }

        static int Partition(List<int> a, int lo, int hi)
        {
            int pivot = a[hi];
            int i = lo - 1;
            for (int j = lo; j < hi; j++)
            {
                if (a[j] <= pivot)
                {
                    i = i + 1;
                    Algo.Swap(a, i, j);
                }
            }
            Algo.Swap(a, i + 1, hi);
            return i + 1;
        }

        static void QuickSortRange(List<int> a, int lo, int hi)
        {
            if (lo >= hi)
            {
                return;
            }
            int p = Algo.Partition(a, lo, hi);
            Algo.QuickSortRange(a, lo, p - 1);
            Algo.QuickSortRange(a, p + 1, hi);
        }

        static List<int> QuickSort(List<int> src)
        {
            List<int> a = Algo.Copy(src);
            Algo.QuickSortRange(a, 0, a.Count - 1);
            return a;
        }

        static List<int> Merge(List<int> a, List<int> b)
        {
            List<int> res = new List<int>();
            int i = 0;
            int j = 0;
            while (i < a.Count && j < b.Count)
            {
                if (a[i] <= b[j])
                {
                    res.Add(a[i]);
                    i = i + 1;
                }
                else
                {
                    res.Add(b[j]);
                    j = j + 1;
                }
            }
            while (i < a.Count)
            {
                res.Add(a[i]);
                i = i + 1;
            }
            while (j < b.Count)
            {
                res.Add(b[j]);
                j = j + 1;
            }
            return res;
        }

        static List<int> MergeSort(List<int> xs)
        {
            if (xs.Count <= 1)
            {
                return xs;
            }
            int mid = xs.Count / 2;
            List<int> left = new List<int>();
            List<int> right = new List<int>();
            for (int i = 0; i < mid; i++)
            {
                left.Add(xs[i]);
            }
            for (int i = mid; i < xs.Count; i++)
            {
                right.Add(xs[i]);
            }
            List<int> ls = Algo.MergeSort(left);
            List<int> rs = Algo.MergeSort(right);
            return Algo.Merge(ls, rs);
        }

        // Binary search over a sorted List; returns the index or -1.
        static int BinarySearch(List<int> a, int target)
        {
            int lo = 0;
            int hi = a.Count - 1;
            while (lo <= hi)
            {
                int mid = (lo + hi) / 2;
                if (a[mid] == target)
                {
                    return mid;
                }
                if (a[mid] < target)
                {
                    lo = mid + 1;
                }
                else
                {
                    hi = mid - 1;
                }
            }
            return -1;
        }

        static bool IsSorted(List<int> a)
        {
            for (int i = 1; i < a.Count; i++)
            {
                if (a[i - 1] > a[i])
                {
                    return false;
                }
            }
            return true;
        }

        static int SumList(List<int> a)
        {
            int s = 0;
            for (int i = 0; i < a.Count; i++)
            {
                s = s + a[i];
            }
            return s;
        }
    }

    class Program
    {
        static int Fails = 0;

        static void Check(string name, int got, int want)
        {
            if (got != want)
            {
                Console.WriteLine($"FAIL {name}: got {got} want {want}");
                Program.Fails++;
            }
        }

        static void CheckB(string name, bool got, bool want)
        {
            Program.Check(name, got ? 1 : 0, want ? 1 : 0);
        }

        // Compares a List against an expected array element by element.
        static void CheckList(string name, List<int> got, int[] want)
        {
            if (got.Count != want.Length)
            {
                Console.WriteLine($"FAIL {name}: count {got.Count} want {want.Length}");
                Program.Fails++;
                return;
            }
            for (int i = 0; i < want.Length; i++)
            {
                if (got[i] != want[i])
                {
                    Console.WriteLine($"FAIL {name} at {i}: got {got[i]} want {want[i]}");
                    Program.Fails++;
                }
            }
        }

        static List<int> ToList(int[] a)
        {
            List<int> r = new List<int>();
            for (int i = 0; i < a.Length; i++)
            {
                r.Add(a[i]);
            }
            return r;
        }

        static int Main()
        {
            List<int> data = new List<int> { 5, 2, 9, 1, 5, 6, 3, 8, 7, 0, 4 };
            int[] sorted = new int[] { 0, 1, 2, 3, 4, 5, 5, 6, 7, 8, 9 };

            // every sort produces the same fully ordered sequence
            Program.CheckList("bubble", Algo.BubbleSort(data), sorted);
            Program.CheckList("insertion", Algo.InsertionSort(data), sorted);
            Program.CheckList("selection", Algo.SelectionSort(data), sorted);
            Program.CheckList("quick", Algo.QuickSort(data), sorted);
            Program.CheckList("merge", Algo.MergeSort(data), sorted);
            Program.CheckB("original intact", data[0] == 5, true);
            Program.Check("original count intact", data.Count, 11);

            // a strictly descending input
            List<int> rev = Program.ToList(new int[] { 9, 8, 7, 6, 5, 4, 3, 2, 1, 0 });
            int[] revWant = new int[] { 0, 1, 2, 3, 4, 5, 6, 7, 8, 9 };
            Program.CheckList("quick reverse", Algo.QuickSort(rev), revWant);
            Program.CheckB("bubble is sorted", Algo.IsSorted(Algo.BubbleSort(rev)), true);
            Program.CheckB("unsorted detected", Algo.IsSorted(rev), false);

            // binary search hits and misses
            List<int> bs = Algo.QuickSort(data);
            Program.Check("bsearch found 0", Algo.BinarySearch(bs, 0), 0);
            Program.Check("bsearch found 9", Algo.BinarySearch(bs, 9), 10);
            Program.Check("bsearch found 6", Algo.BinarySearch(bs, 6), 7);
            Program.Check("bsearch miss high", Algo.BinarySearch(bs, 100), -1);
            Program.Check("bsearch miss low", Algo.BinarySearch(bs, -5), -1);
            Program.Check("sum preserved", Algo.SumList(bs), 50);

            // List-backed stack
            IntStack st = new IntStack();
            Program.CheckB("stack empty", st.IsEmpty(), true);
            st.Push(10);
            st.Push(20);
            st.Push(30);
            Program.Check("stack size", st.Size(), 3);
            Program.Check("stack peek", st.Peek(), 30);
            Program.Check("stack pop 1", st.Pop(), 30);
            Program.Check("stack pop 2", st.Pop(), 20);
            Program.Check("stack size after pop", st.Size(), 1);
            Program.CheckB("stack not empty", st.IsEmpty(), false);

            // stack reverses a sequence, reusing freed slots
            IntStack rstk = new IntStack();
            for (int i = 1; i <= 5; i++)
            {
                rstk.Push(i);
            }
            int reversed = 0;
            while (!rstk.IsEmpty())
            {
                reversed = reversed * 10 + rstk.Pop();
            }
            Program.Check("stack reversal", reversed, 54321);

            // circular queue preserves FIFO order and wraps around the ring
            IntQueue q = new IntQueue(4);
            q.Enqueue(1);
            q.Enqueue(2);
            q.Enqueue(3);
            Program.Check("queue deq 1", q.Dequeue(), 1);
            Program.Check("queue deq 2", q.Dequeue(), 2);
            q.Enqueue(4);
            q.Enqueue(5);
            q.Enqueue(6);
            Program.Check("queue size", q.Size(), 4);
            Program.Check("queue deq 3", q.Dequeue(), 3);
            Program.Check("queue deq 4", q.Dequeue(), 4);
            Program.Check("queue deq 5", q.Dequeue(), 5);
            Program.Check("queue deq 6", q.Dequeue(), 6);
            Program.Check("queue empty", q.Size(), 0);

            // linked list: append, prepend, index, sum, reverse
            IntList list = new IntList();
            list.Append(1);
            list.Append(2);
            list.Append(3);
            list.Prepend(0);
            Program.Check("list size", list.Size(), 4);
            Program.Check("list at 0", list.At(0), 0);
            Program.Check("list at 3", list.At(3), 3);
            Program.Check("list sum", list.Sum(), 6);
            list.Reverse();
            Program.Check("list reversed at 0", list.At(0), 3);
            Program.Check("list reversed at 3", list.At(3), 0);
            Program.Check("list sum after reverse", list.Sum(), 6);

            if (Program.Fails == 0)
            {
                Console.WriteLine("C# big test 1 (data structures) passed");
            }
            return Program.Fails;
        }
    }
}
