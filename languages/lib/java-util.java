// java-util.java -- the part of java.util both java grammars provide, written in
// Java and compiled with the program: java-interpreter.abnf and
// java-to-llvm-ir.abnf load it (jLoadUtil) when the program imports one of its
// types, so a program that does not import java.util pays nothing for it.
//
// One implementation serves all three engines - the interpreter, the Go twin
// and layer 2 run it as the program's own classes. What it answers is java
// 24.0.2's, including the exception messages; the iteration order and growth
// of the backing array are not observable.
package java.util;

interface Iterator<E> {
    boolean hasNext();
    E next();
}

interface Collection<E> {
    int size();
    boolean isEmpty();
    boolean contains(Object o);
    boolean add(E e);
    boolean remove(Object o);
    void clear();
    Iterator<E> iterator();
}

interface List<E> extends Collection<E> {
    int size();
    boolean isEmpty();
    boolean contains(Object o);
    boolean add(E e);
    boolean remove(Object o);
    void clear();
    Iterator<E> iterator();
    E get(int index);
    E set(int index, E e);
    void add(int index, E e);
    E remove(int index);
    int indexOf(Object o);
    int lastIndexOf(Object o);

    @SafeVarargs
    static <E> List<E> of(E... es) {
        ArrayList<E> l = new ArrayList<E>(es.length);
        for (int i = 0; i < es.length; i++) {
            if (es[i] == null) { throw new NullPointerException(); }
            l.add(es[i]);
        }
        l.frozen = true;
        return l;
    }
}

class ArrayList<E> implements List<E> {
    Object[] elems;
    int count;
    boolean frozen;
    // Counts the structural changes (a size change; set is not one) so an
    // iterator or forEach running across one fails fast, as java's does.
    int modCount;

    public ArrayList() { elems = new Object[10]; }
    public ArrayList(int capacity) {
        if (capacity < 0) { throw new IllegalArgumentException("Illegal Capacity: " + capacity); }
        elems = new Object[capacity];
    }
    public ArrayList(Collection<? extends E> c) {
        elems = new Object[c.size()];
        addAll(c);
    }

    void writable() {
        if (frozen) { throw new UnsupportedOperationException(); }
    }
    void checkIndex(int index) {
        if (index < 0 || index >= count) {
            throw new IndexOutOfBoundsException("Index " + index + " out of bounds for length " + count);
        }
    }
    void grow() {
        Object[] more = new Object[elems.length * 2 + 1];
        System.arraycopy(elems, 0, more, 0, count);
        elems = more;
    }

    public int size() { return count; }
    public boolean isEmpty() { return count == 0; }
    public boolean contains(Object o) { return indexOf(o) >= 0; }
    public int indexOf(Object o) {
        for (int i = 0; i < count; i++) {
            if (o == null ? elems[i] == null : o.equals(elems[i])) { return i; }
        }
        return -1;
    }
    public int lastIndexOf(Object o) {
        for (int i = count - 1; i >= 0; i--) {
            if (o == null ? elems[i] == null : o.equals(elems[i])) { return i; }
        }
        return -1;
    }
    @SuppressWarnings("unchecked")
    public E get(int index) {
        checkIndex(index);
        return (E) elems[index];
    }
    @SuppressWarnings("unchecked")
    public E set(int index, E e) {
        writable();
        checkIndex(index);
        E old = (E) elems[index];
        elems[index] = e;
        return old;
    }
    public boolean add(E e) {
        writable();
        modCount++;
        if (count == elems.length) { grow(); }
        elems[count] = e;
        count++;
        return true;
    }
    public void add(int index, E e) {
        writable();
        if (index < 0 || index > count) {
            throw new IndexOutOfBoundsException("Index: " + index + ", Size: " + count);
        }
        modCount++;
        if (count == elems.length) { grow(); }
        System.arraycopy(elems, index, elems, index + 1, count - index);
        elems[index] = e;
        count++;
    }
    @SuppressWarnings("unchecked")
    public E remove(int index) {
        writable();
        checkIndex(index);
        E old = (E) elems[index];
        modCount++;
        System.arraycopy(elems, index + 1, elems, index, count - index - 1);
        count--;
        elems[count] = null;
        return old;
    }
    public boolean remove(Object o) {
        writable();
        int i = indexOf(o);
        if (i < 0) { return false; }
        remove(i);
        return true;
    }
    public void clear() {
        writable();
        modCount++;
        elems = new Object[elems.length];
        count = 0;
    }
    public boolean addAll(Collection<? extends E> c) {
        writable();
        Object[] all = new Object[c.size()];
        int n = 0;
        Iterator<? extends E> it = c.iterator();
        while (it.hasNext()) { all[n] = it.next(); n++; }
        modCount++;
        while (elems.length - count < n) { grow(); }
        System.arraycopy(all, 0, elems, count, n);
        count = count + n;
        return n > 0;
    }
    public Object[] toArray() {
        Object[] out = new Object[count];
        System.arraycopy(elems, 0, out, 0, count);
        return out;
    }
    public Iterator<E> iterator() { return new ArrayListItr<E>(this); }
    @SuppressWarnings("unchecked")
    public void forEach(java.util.function.Consumer<? super E> action) {
        int expected = modCount;
        for (int i = 0; modCount == expected && i < count; i++) { action.accept((E) elems[i]); }
        if (modCount != expected) { throw new ConcurrentModificationException(); }
    }
    @SuppressWarnings("unchecked")
    public boolean removeIf(java.util.function.Predicate<? super E> filter) {
        writable();
        int kept = 0;
        for (int i = 0; i < count; i++) {
            if (!filter.test((E) elems[i])) {
                elems[kept] = elems[i];
                kept++;
            }
        }
        boolean any = kept < count;
        if (any) { modCount++; }
        for (int i = kept; i < count; i++) { elems[i] = null; }
        count = kept;
        return any;
    }

    // Any List is equal to one with equal elements in the same order, read
    // through the other list's iterator().
    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof List)) { return false; }
        Iterator<?> it = ((List<?>) o).iterator();
        for (int i = 0; i < count; i++) {
            if (!it.hasNext()) { return false; }
            Object a = elems[i];
            Object b = it.next();
            if (!(a == null ? b == null : a.equals(b))) { return false; }
        }
        return !it.hasNext();
    }
    public int hashCode() {
        int h = 1;
        for (int i = 0; i < count; i++) {
            Object e = elems[i];
            h = 31 * h + (e == null ? 0 : e.hashCode());
        }
        return h;
    }
    public String toString() {
        String s = "[";
        for (int i = 0; i < count; i++) {
            if (i > 0) { s = s + ", "; }
            s = s + (elems[i] == this ? "(this Collection)" : "" + elems[i]);
        }
        return s + "]";
    }
}

class NoSuchElementException extends RuntimeException {
    public NoSuchElementException() { super(); }
    public NoSuchElementException(String message) { super(message); }
}

class ConcurrentModificationException extends RuntimeException {
    public ConcurrentModificationException() { super(); }
    public ConcurrentModificationException(String message) { super(message); }
}

// Fail-fast: next() throws once the list changed size behind the iterator. A
// loop that removes the next-to-last element therefore ends quietly, because
// hasNext() is a plain cursor != size test, as in java.
class ArrayListItr<E> implements Iterator<E> {
    ArrayList<E> list;
    int cursor;
    int expected;

    ArrayListItr(ArrayList<E> list) {
        this.list = list;
        expected = list.modCount;
    }
    public boolean hasNext() { return cursor != list.count; }
    @SuppressWarnings("unchecked")
    public E next() {
        if (list.modCount != expected) { throw new ConcurrentModificationException(); }
        if (cursor >= list.count) { throw new NoSuchElementException(); }
        cursor++;
        return (E) list.elems[cursor - 1];
    }
}

// A List from get(int) and size() alone; every mutator throws until a subclass
// overrides it, and add(E) / remove(Object) / clear() go through the indexed ones.
abstract class AbstractList<E> implements List<E> {
    public abstract E get(int index);
    public abstract int size();

    public boolean isEmpty() { return size() == 0; }
    public boolean contains(Object o) { return indexOf(o) >= 0; }
    public int indexOf(Object o) {
        int n = size();
        for (int i = 0; i < n; i++) {
            Object e = get(i);
            if (o == null ? e == null : o.equals(e)) { return i; }
        }
        return -1;
    }
    public int lastIndexOf(Object o) {
        for (int i = size() - 1; i >= 0; i--) {
            Object e = get(i);
            if (o == null ? e == null : o.equals(e)) { return i; }
        }
        return -1;
    }
    public E set(int index, E e) { throw new UnsupportedOperationException(); }
    public void add(int index, E e) { throw new UnsupportedOperationException(); }
    public E remove(int index) { throw new UnsupportedOperationException(); }
    public boolean add(E e) {
        add(size(), e);
        return true;
    }
    public boolean remove(Object o) {
        int i = indexOf(o);
        if (i < 0) { return false; }
        remove(i);
        return true;
    }
    public void clear() {
        int n = size();
        for (int i = 0; i < n; i++) { remove(0); }
    }
    public Iterator<E> iterator() { return new AbstractListItr<E>(this); }

    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof List)) { return false; }
        Iterator<E> mine = iterator();
        Iterator<?> it = ((List<?>) o).iterator();
        while (mine.hasNext() && it.hasNext()) {
            Object a = mine.next();
            Object b = it.next();
            if (!(a == null ? b == null : a.equals(b))) { return false; }
        }
        return !(mine.hasNext() || it.hasNext());
    }
    public int hashCode() {
        int h = 1;
        Iterator<E> it = iterator();
        while (it.hasNext()) {
            Object e = it.next();
            h = 31 * h + (e == null ? 0 : e.hashCode());
        }
        return h;
    }
    public String toString() {
        String s = "[";
        Iterator<E> it = iterator();
        boolean first = true;
        while (it.hasNext()) {
            Object e = it.next();
            if (!first) { s = s + ", "; }
            first = false;
            s = s + (e == this ? "(this Collection)" : "" + e);
        }
        return s + "]";
    }
}

// Reads through get(int), so it sees whatever the subclass answers; running off
// the end turns the get's IndexOutOfBoundsException into a NoSuchElementException
// whose message is that exception's toString(), as java's cause constructor does.
class AbstractListItr<E> implements Iterator<E> {
    AbstractList<E> list;
    int cursor;

    AbstractListItr(AbstractList<E> list) { this.list = list; }
    public boolean hasNext() { return cursor != list.size(); }
    public E next() {
        E e;
        try {
            e = list.get(cursor);
        } catch (IndexOutOfBoundsException x) {
            throw new NoSuchElementException(x.toString());
        }
        cursor++;
        return e;
    }
}

// The value model carries no element type, so every array is an Object[] to
// these bodies (elems). The typed overloads exist where the element type
// decides the answer: the value a fill converts to it and the zero a copy pads
// with. Natural order is each element's compareTo, which the boxed primitives
// answer as their wrapper classes do - Double.compare's order for a double[].
class Arrays {
    private Arrays() {}

    static Object[] elems(Object a) { return (Object[]) a; }

    public static String toString(Object[] a) {
        if (a == null) { return "null"; }
        String s = "[";
        for (int i = 0; i < a.length; i++) {
            if (i > 0) { s = s + ", "; }
            s = s + a[i];
        }
        return s + "]";
    }
    public static int hashCode(Object[] a) {
        if (a == null) { return 0; }
        int h = 1;
        for (int i = 0; i < a.length; i++) { h = 31 * h + (a[i] == null ? 0 : a[i].hashCode()); }
        return h;
    }
    // Element by element with equals, so two double[]s compare as
    // Double.equals does: NaN equals NaN, 0.0 does not equal -0.0.
    public static boolean equals(Object[] a, Object[] b) {
        if (a == b) { return true; }
        if (a == null || b == null || a.length != b.length) { return false; }
        for (int i = 0; i < a.length; i++) {
            if (!(a[i] == null ? b[i] == null : a[i].equals(b[i]))) { return false; }
        }
        return true;
    }

    static void rangeCheck(int length, int from, int to) {
        if (from > to) { throw new IllegalArgumentException("fromIndex(" + from + ") > toIndex(" + to + ")"); }
        if (from < 0) { throw new ArrayIndexOutOfBoundsException("Array index out of range: " + from); }
        if (to > length) { throw new ArrayIndexOutOfBoundsException("Array index out of range: " + to); }
    }

    static void fillRange(Object[] a, int from, int to, Object v) {
        rangeCheck(a.length, from, to);
        for (int i = from; i < to; i++) { a[i] = v; }
    }
    public static void fill(Object[] a, Object v) { fillRange(a, 0, a.length, v); }
    public static void fill(Object[] a, int from, int to, Object v) { fillRange(a, from, to, v); }
    public static void fill(int[] a, int v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(int[] a, int from, int to, int v) { fillRange(elems(a), from, to, v); }
    public static void fill(long[] a, long v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(long[] a, int from, int to, long v) { fillRange(elems(a), from, to, v); }
    public static void fill(double[] a, double v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(double[] a, int from, int to, double v) { fillRange(elems(a), from, to, v); }
    public static void fill(float[] a, float v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(float[] a, int from, int to, float v) { fillRange(elems(a), from, to, v); }
    public static void fill(short[] a, short v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(short[] a, int from, int to, short v) { fillRange(elems(a), from, to, v); }
    public static void fill(byte[] a, byte v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(byte[] a, int from, int to, byte v) { fillRange(elems(a), from, to, v); }
    public static void fill(char[] a, char v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(char[] a, int from, int to, char v) { fillRange(elems(a), from, to, v); }
    public static void fill(boolean[] a, boolean v) { fillRange(elems(a), 0, a.length, v); }
    public static void fill(boolean[] a, int from, int to, boolean v) { fillRange(elems(a), from, to, v); }

    // copyOf and copyOfRange: dst is the new array, already holding its type's
    // zero, and the source's elements from `from` fill it as far as they reach.
    static Object copyInto(Object src, int length, int from, int to, Object dst) {
        System.arraycopy(src, from, dst, 0, Math.min(length - from, to - from));
        return dst;
    }
    static int span(int from, int to) {
        if (from > to) { throw new IllegalArgumentException(from + " > " + to); }
        return to - from;
    }
    @SuppressWarnings("unchecked")
    public static <T> T[] copyOf(T[] a, int n) { return (T[]) copyInto(a, a.length, 0, n, new Object[n]); }
    public static int[] copyOf(int[] a, int n) { return (int[]) copyInto(a, a.length, 0, n, new int[n]); }
    public static long[] copyOf(long[] a, int n) { return (long[]) copyInto(a, a.length, 0, n, new long[n]); }
    public static double[] copyOf(double[] a, int n) { return (double[]) copyInto(a, a.length, 0, n, new double[n]); }
    public static float[] copyOf(float[] a, int n) { return (float[]) copyInto(a, a.length, 0, n, new float[n]); }
    public static short[] copyOf(short[] a, int n) { return (short[]) copyInto(a, a.length, 0, n, new short[n]); }
    public static byte[] copyOf(byte[] a, int n) { return (byte[]) copyInto(a, a.length, 0, n, new byte[n]); }
    public static char[] copyOf(char[] a, int n) { return (char[]) copyInto(a, a.length, 0, n, new char[n]); }
    public static boolean[] copyOf(boolean[] a, int n) { return (boolean[]) copyInto(a, a.length, 0, n, new boolean[n]); }
    @SuppressWarnings("unchecked")
    public static <T> T[] copyOfRange(T[] a, int from, int to) { return (T[]) copyInto(a, a.length, from, to, new Object[span(from, to)]); }
    public static int[] copyOfRange(int[] a, int from, int to) { return (int[]) copyInto(a, a.length, from, to, new int[span(from, to)]); }
    public static long[] copyOfRange(long[] a, int from, int to) { return (long[]) copyInto(a, a.length, from, to, new long[span(from, to)]); }
    public static double[] copyOfRange(double[] a, int from, int to) { return (double[]) copyInto(a, a.length, from, to, new double[span(from, to)]); }
    public static float[] copyOfRange(float[] a, int from, int to) { return (float[]) copyInto(a, a.length, from, to, new float[span(from, to)]); }
    public static short[] copyOfRange(short[] a, int from, int to) { return (short[]) copyInto(a, a.length, from, to, new short[span(from, to)]); }
    public static byte[] copyOfRange(byte[] a, int from, int to) { return (byte[]) copyInto(a, a.length, from, to, new byte[span(from, to)]); }
    public static char[] copyOfRange(char[] a, int from, int to) { return (char[]) copyInto(a, a.length, from, to, new char[span(from, to)]); }
    public static boolean[] copyOfRange(boolean[] a, int from, int to) { return (boolean[]) copyInto(a, a.length, from, to, new boolean[span(from, to)]); }

    // A stable merge sort, as java's object sort is: equal elements keep their
    // order, which a Comparator can observe. c == null is natural order.
    @SuppressWarnings("unchecked")
    static int order(Object x, Object y, Comparator<Object> c) {
        return c == null ? ((Comparable<Object>) x).compareTo(y) : c.compare(x, y);
    }
    static void mergeSort(Object[] a, Object[] tmp, int lo, int hi, Comparator<Object> c) {
        if (hi - lo < 2) { return; }
        int mid = (lo + hi) >>> 1;
        mergeSort(a, tmp, lo, mid, c);
        mergeSort(a, tmp, mid, hi, c);
        if (order(a[mid - 1], a[mid], c) <= 0) { return; }
        System.arraycopy(a, lo, tmp, lo, hi - lo);
        int i = lo;
        int j = mid;
        for (int k = lo; k < hi; k++) {
            if (j >= hi || (i < mid && order(tmp[i], tmp[j], c) <= 0)) {
                a[k] = tmp[i];
                i++;
            } else {
                a[k] = tmp[j];
                j++;
            }
        }
    }
    @SuppressWarnings("unchecked")
    static void sortRange(Object[] a, int from, int to, Comparator<?> c) {
        rangeCheck(a.length, from, to);
        mergeSort(a, new Object[a.length], from, to, (Comparator<Object>) c);
    }
    public static void sort(Object[] a) { sortRange(a, 0, a.length, null); }
    public static void sort(Object[] a, int from, int to) { sortRange(a, from, to, null); }
    public static <T> void sort(T[] a, Comparator<? super T> c) { sortRange(a, 0, a.length, c); }
    public static <T> void sort(T[] a, int from, int to, Comparator<? super T> c) { sortRange(a, from, to, c); }

    // -(insertion point) - 1 when the key is absent, over a sorted array.
    @SuppressWarnings("unchecked")
    static int search(Object[] a, Object key, Comparator<?> c) {
        int lo = 0;
        int hi = a.length - 1;
        while (lo <= hi) {
            int mid = (lo + hi) >>> 1;
            int r = order(a[mid], key, (Comparator<Object>) c);
            if (r < 0) {
                lo = mid + 1;
            } else if (r > 0) {
                hi = mid - 1;
            } else {
                return mid;
            }
        }
        return -(lo + 1);
    }
    public static int binarySearch(Object[] a, Object key) { return search(a, key, null); }
    public static <T> int binarySearch(T[] a, T key, Comparator<? super T> c) { return search(a, key, c); }

    @SafeVarargs
    public static <T> List<T> asList(T... a) { return new ArrayView<T>(a); }

    // asList's fixed-size list: reads and set write through to the array, and
    // the size never changes, so add and remove throw.
    private static class ArrayView<E> extends AbstractList<E> {
        E[] a;
        ArrayView(E[] a) {
            if (a == null) { throw new NullPointerException(); }
            this.a = a;
        }
        public int size() { return a.length; }
        public E get(int index) { return a[index]; }
        public E set(int index, E e) {
            E old = a[index];
            a[index] = e;
            return old;
        }
    }
}
