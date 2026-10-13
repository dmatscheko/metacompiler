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

import java.lang.reflect.Array;

interface Iterator<E> {
    boolean hasNext();
    E next();
    default void remove() { throw new UnsupportedOperationException("remove"); }
}

// java.util.Comparator: naturalOrder, reverseOrder and comparing order by
// Comparable's compareTo, as java's do. Each comparator is built in a local of
// type Comparator, which makes the lambda an instance of this interface, so a
// chained reversed() or thenComparing() reaches the defaults. thenComparing's
// Function and Comparator forms are told apart by the argument's arity where
// the call site is resolved statically (jFnArity in lib/java-decltypes.js).
interface Comparator<T> {
    int compare(T a, T b);

    @SuppressWarnings("unchecked")
    static <T extends Comparable<? super T>> Comparator<T> naturalOrder() {
        Comparator<T> c = (a, b) -> ((Comparable<Object>) a).compareTo(b);
        return c;
    }
    @SuppressWarnings("unchecked")
    static <T extends Comparable<? super T>> Comparator<T> reverseOrder() {
        Comparator<T> c = (a, b) -> ((Comparable<Object>) b).compareTo(a);
        return c;
    }
    @SuppressWarnings("unchecked")
    static <T, U extends Comparable<? super U>> Comparator<T> comparing(java.util.function.Function<? super T, ? extends U> key) {
        Comparator<T> c = (a, b) -> ((Comparable<Object>) key.apply(a)).compareTo(key.apply(b));
        return c;
    }
    static <T, U> Comparator<T> comparing(java.util.function.Function<? super T, ? extends U> key, Comparator<? super U> keyCmp) {
        Comparator<T> c = (a, b) -> keyCmp.compare(key.apply(a), key.apply(b));
        return c;
    }
    static <T> Comparator<T> comparingInt(java.util.function.ToIntFunction<? super T> key) {
        Comparator<T> c = (a, b) -> Integer.compare(key.applyAsInt(a), key.applyAsInt(b));
        return c;
    }
    static <T> Comparator<T> comparingLong(java.util.function.ToLongFunction<? super T> key) {
        Comparator<T> c = (a, b) -> Long.compare(key.applyAsLong(a), key.applyAsLong(b));
        return c;
    }
    static <T> Comparator<T> comparingDouble(java.util.function.ToDoubleFunction<? super T> key) {
        Comparator<T> c = (a, b) -> Double.compare(key.applyAsDouble(a), key.applyAsDouble(b));
        return c;
    }
    default Comparator<T> reversed() {
        Comparator<T> self = this;
        Comparator<T> c = (a, b) -> self.compare(b, a);
        return c;
    }
    default Comparator<T> thenComparing(Comparator<? super T> other) {
        Comparator<T> self = this;
        Comparator<T> c = (a, b) -> {
            int r = self.compare(a, b);
            return r != 0 ? r : other.compare(a, b);
        };
        return c;
    }
    default <U> Comparator<T> thenComparing(java.util.function.Function<? super T, ? extends U> key, Comparator<? super U> keyCmp) {
        return thenComparing(comparing(key, keyCmp));
    }
    default <U extends Comparable<? super U>> Comparator<T> thenComparing(java.util.function.Function<? super T, ? extends U> key) {
        return thenComparing(comparing(key));
    }
    default Comparator<T> thenComparingInt(java.util.function.ToIntFunction<? super T> key) {
        return thenComparing(comparingInt(key));
    }
    default Comparator<T> thenComparingLong(java.util.function.ToLongFunction<? super T> key) {
        return thenComparing(comparingLong(key));
    }
    default Comparator<T> thenComparingDouble(java.util.function.ToDoubleFunction<? super T> key) {
        return thenComparing(comparingDouble(key));
    }
}

interface Collection<E> {
    int size();
    boolean isEmpty();
    boolean contains(Object o);
    boolean add(E e);
    boolean remove(Object o);
    void clear();
    Iterator<E> iterator();

    // A sequential stream over the elements (lib/java-stream.java).
    default java.util.stream.Stream<E> stream() { return Stream.ofCollection(this); }
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
        l.listOf = true;
        return l;
    }
}

class ArrayList<E> implements List<E> {
    Object[] elems;
    int count;
    boolean frozen;
    // List.of's list, whose get answers java's ImmutableCollections: List12
    // (one or two elements) and ListN (any other count) word an index out of
    // range differently.
    boolean listOf;
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
        if (listOf && (index < 0 || index >= count)) {
            if (count == 1 || count == 2) { throw new IndexOutOfBoundsException("Index: " + index + " Size: " + count); }
            throw new ArrayIndexOutOfBoundsException("Index " + index + " out of bounds for length " + count);
        }
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
    // Into a, or a new array of a's type when a is too short; the slot after
    // the last element is set to null.
    @SuppressWarnings("unchecked")
    public <T> T[] toArray(T[] a) {
        T[] out = a.length >= count ? a : (T[]) Array.newInstance(a.getClass().getComponentType(), count);
        System.arraycopy(elems, 0, out, 0, count);
        if (out.length > count) { out[count] = null; }
        return out;
    }
    public Iterator<E> iterator() { return new ArrayListItr<E>(this); }
    // Stable, and c == null is natural order (List#sort).
    @SuppressWarnings("unchecked")
    public void replaceAll(java.util.function.UnaryOperator<E> op) {
        writable();
        for (int i = 0; i < count; i++) { elems[i] = op.apply((E) elems[i]); }
        modCount++;
    }
    // A view of [from, to): reads and writes go through to this list.
    public List<E> subList(int from, int to) {
        if (from < 0) { throw new IndexOutOfBoundsException("fromIndex = " + from); }
        if (to > count) { throw new IndexOutOfBoundsException("toIndex = " + to); }
        if (from > to) { throw new IllegalArgumentException("fromIndex(" + from + ") > toIndex(" + to + ")"); }
        return new SubList<E>(this, from, to - from);
    }
    public void sort(Comparator<? super E> c) {
        writable();
        Arrays.sortRange(elems, 0, count, c);
        modCount++;
    }
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
    // The index next() answered last, -1 once remove() has used it.
    int lastRet = -1;

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
        lastRet = cursor - 1;
        return (E) list.elems[cursor - 1];
    }
    public void remove() {
        if (lastRet < 0) { throw new IllegalStateException(); }
        if (list.modCount != expected) { throw new ConcurrentModificationException(); }
        list.remove(lastRet);
        cursor = lastRet;
        lastRet = -1;
        expected = list.modCount;
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

// ArrayList.subList's view and Collections.unmodifiableList's: get and size
// read through to the backing list; a SubList also sets, adds and removes
// through it.
class SubList<E> extends AbstractList<E> {
    ArrayList<E> parent;
    int offset;
    int size;
    SubList(ArrayList<E> parent, int offset, int size) {
        this.parent = parent;
        this.offset = offset;
        this.size = size;
    }
    void check(int index, int n) {
        if (index < 0 || index >= n) { throw new IndexOutOfBoundsException("Index " + index + " out of bounds for length " + n); }
    }
    public E get(int index) {
        check(index, size);
        return parent.get(offset + index);
    }
    public int size() { return size; }
    public E set(int index, E e) {
        check(index, size);
        return parent.set(offset + index, e);
    }
    public void add(int index, E e) {
        check(index, size + 1);
        parent.add(offset + index, e);
        size++;
    }
    public E remove(int index) {
        check(index, size);
        size--;
        return parent.remove(offset + index);
    }
}
class UnmodifiableList<E> extends AbstractList<E> {
    List<? extends E> list;
    UnmodifiableList(List<? extends E> list) { this.list = list; }
    public E get(int index) { return list.get(index); }
    public int size() { return list.size(); }
}

// java.util.Objects.
final class Objects {
    private Objects() {}
    public static boolean equals(Object a, Object b) { return a == b || (a != null && a.equals(b)); }
    public static int hashCode(Object o) { return o == null ? 0 : o.hashCode(); }
    public static int hash(Object... values) { return Arrays.hashCode(values); }
    public static String toString(Object o) { return String.valueOf(o); }
    public static String toString(Object o, String nullDefault) { return o == null ? nullDefault : o.toString(); }
    public static boolean isNull(Object o) { return o == null; }
    public static boolean nonNull(Object o) { return o != null; }
    public static <T> T requireNonNull(T obj) {
        if (obj == null) { throw new NullPointerException(); }
        return obj;
    }
    public static <T> T requireNonNull(T obj, String message) {
        if (obj == null) { throw new NullPointerException(message); }
        return obj;
    }
    public static <T> T requireNonNullElse(T obj, T dflt) {
        if (obj != null) { return obj; }
        if (dflt == null) { throw new NullPointerException("defaultObj"); }
        return dflt;
    }
    public static <T> int compare(T a, T b, Comparator<? super T> c) { return a == b ? 0 : c.compare(a, b); }
    public static int checkIndex(int index, int length) {
        if (index < 0 || index >= length) { throw new IndexOutOfBoundsException("Index " + index + " out of bounds for length " + length); }
        return index;
    }
}

// java.util.Collections, for lists and plain collections.
final class Collections {
    private Collections() {}
    public static <T extends Comparable<? super T>> void sort(List<T> list) { sort(list, null); }
    @SuppressWarnings("unchecked")
    public static <T> void sort(List<T> list, Comparator<? super T> c) {
        Object[] a = list.toArray();
        Arrays.sortRange(a, 0, a.length, (Comparator<Object>) c);
        for (int i = 0; i < a.length; i++) { list.set(i, (T) a[i]); }
    }
    @SuppressWarnings("unchecked")
    public static void reverse(List<?> list) {
        List<Object> l = (List<Object>) list;
        for (int i = 0, j = l.size() - 1; i < j; i++, j--) {
            Object t = l.get(i);
            l.set(i, l.get(j));
            l.set(j, t);
        }
    }
    @SuppressWarnings("unchecked")
    public static void swap(List<?> list, int i, int j) {
        List<Object> l = (List<Object>) list;
        l.set(i, l.set(j, l.get(i)));
    }
    public static <T extends Comparable<? super T>> T max(Collection<? extends T> coll) { return pick(coll, null, 1); }
    public static <T> T max(Collection<? extends T> coll, Comparator<? super T> c) { return pick(coll, c, 1); }
    public static <T extends Comparable<? super T>> T min(Collection<? extends T> coll) { return pick(coll, null, -1); }
    public static <T> T min(Collection<? extends T> coll, Comparator<? super T> c) { return pick(coll, c, -1); }
    // The greatest (sign 1) or least (sign -1) element, the first of equals.
    @SuppressWarnings("unchecked")
    static <T> T pick(Collection<? extends T> coll, Comparator<? super T> c, int sign) {
        Iterator<? extends T> it = coll.iterator();
        T best = it.next();
        while (it.hasNext()) {
            T e = it.next();
            int r = c == null ? ((Comparable<Object>) e).compareTo(best) : c.compare(e, best);
            if (r * sign > 0) { best = e; }
        }
        return best;
    }
    public static int frequency(Collection<?> c, Object o) {
        int n = 0;
        Iterator<?> it = c.iterator();
        while (it.hasNext()) {
            Object e = it.next();
            if (o == null ? e == null : o.equals(e)) { n++; }
        }
        return n;
    }
    public static <T> List<T> nCopies(int n, T o) {
        if (n < 0) { throw new IllegalArgumentException("List length = " + n); }
        ArrayList<T> l = new ArrayList<T>(n);
        for (int i = 0; i < n; i++) { l.add(o); }
        l.frozen = true;
        return l;
    }
    public static <T> List<T> emptyList() {
        ArrayList<T> l = new ArrayList<T>(0);
        l.frozen = true;
        return l;
    }
    public static <T> List<T> singletonList(T o) {
        ArrayList<T> l = new ArrayList<T>(1);
        l.add(o);
        l.frozen = true;
        return l;
    }
    public static <T> List<T> unmodifiableList(List<? extends T> list) { return new UnmodifiableList<T>(list); }
    @SafeVarargs
    public static <T> boolean addAll(Collection<? super T> c, T... elements) {
        boolean changed = false;
        for (int i = 0; i < elements.length; i++) { if (c.add(elements[i])) { changed = true; } }
        return changed;
    }
}

// java.util.Queue, Deque and ArrayDeque: a circular buffer that doubles when
// full and holds no null, as java's does.
interface Queue<E> extends Collection<E> {
    boolean offer(E e);
    E poll();
    E peek();
    E remove();
    E element();
}
interface Deque<E> extends Queue<E> {
    void addFirst(E e);
    void addLast(E e);
    boolean offerFirst(E e);
    boolean offerLast(E e);
    E removeFirst();
    E removeLast();
    E pollFirst();
    E pollLast();
    E getFirst();
    E getLast();
    E peekFirst();
    E peekLast();
    void push(E e);
    E pop();
}
class ArrayDeque<E> implements Deque<E> {
    Object[] elems;
    int head;
    int count;

    public ArrayDeque() { elems = new Object[16]; }
    public ArrayDeque(int capacity) { elems = new Object[capacity < 1 ? 1 : capacity]; }
    public ArrayDeque(Collection<? extends E> c) {
        elems = new Object[c.size() + 1];
        Iterator<? extends E> it = c.iterator();
        while (it.hasNext()) { addLast(it.next()); }
    }
    void grow() {
        Object[] more = new Object[elems.length * 2];
        for (int i = 0; i < count; i++) { more[i] = elems[(head + i) % elems.length]; }
        elems = more;
        head = 0;
    }
    @SuppressWarnings("unchecked")
    E at(int i) { return (E) elems[(head + i) % elems.length]; }
    public void addFirst(E e) {
        if (e == null) { throw new NullPointerException(); }
        if (count == elems.length) { grow(); }
        head = (head - 1 + elems.length) % elems.length;
        elems[head] = e;
        count++;
    }
    public void addLast(E e) {
        if (e == null) { throw new NullPointerException(); }
        if (count == elems.length) { grow(); }
        elems[(head + count) % elems.length] = e;
        count++;
    }
    public boolean offerFirst(E e) { addFirst(e); return true; }
    public boolean offerLast(E e) { addLast(e); return true; }
    public E pollFirst() {
        if (count == 0) { return null; }
        E e = at(0);
        elems[head] = null;
        head = (head + 1) % elems.length;
        count--;
        return e;
    }
    public E pollLast() {
        if (count == 0) { return null; }
        E e = at(count - 1);
        elems[(head + count - 1) % elems.length] = null;
        count--;
        return e;
    }
    public E removeFirst() {
        if (count == 0) { throw new NoSuchElementException(); }
        return pollFirst();
    }
    public E removeLast() {
        if (count == 0) { throw new NoSuchElementException(); }
        return pollLast();
    }
    public E getFirst() {
        if (count == 0) { throw new NoSuchElementException(); }
        return at(0);
    }
    public E getLast() {
        if (count == 0) { throw new NoSuchElementException(); }
        return at(count - 1);
    }
    public E peekFirst() { return count == 0 ? null : at(0); }
    public E peekLast() { return count == 0 ? null : at(count - 1); }
    public void push(E e) { addFirst(e); }
    public E pop() { return removeFirst(); }
    public boolean add(E e) { addLast(e); return true; }
    public boolean offer(E e) { addLast(e); return true; }
    public E poll() { return pollFirst(); }
    public E peek() { return peekFirst(); }
    public E remove() { return removeFirst(); }
    public E element() { return getFirst(); }
    public int size() { return count; }
    public boolean isEmpty() { return count == 0; }
    public boolean contains(Object o) {
        for (int i = 0; i < count; i++) { if (o != null && o.equals(at(i))) { return true; } }
        return false;
    }
    public boolean remove(Object o) {
        for (int i = 0; i < count; i++) {
            if (o != null && o.equals(at(i))) {
                for (int k = i; k < count - 1; k++) { elems[(head + k) % elems.length] = elems[(head + k + 1) % elems.length]; }
                elems[(head + count - 1) % elems.length] = null;
                count--;
                return true;
            }
        }
        return false;
    }
    public void clear() {
        for (int i = 0; i < count; i++) { elems[(head + i) % elems.length] = null; }
        head = 0;
        count = 0;
    }
    public Iterator<E> iterator() { return new ArrayDequeItr<E>(this); }
    public String toString() {
        String s = "[";
        for (int i = 0; i < count; i++) { s = s + (i > 0 ? ", " : "") + at(i); }
        return s + "]";
    }
}
class ArrayDequeItr<E> implements Iterator<E> {
    ArrayDeque<E> dq;
    int cursor;
    ArrayDequeItr(ArrayDeque<E> dq) { this.dq = dq; }
    public boolean hasNext() { return cursor < dq.count; }
    public E next() {
        if (cursor >= dq.count) { throw new NoSuchElementException(); }
        cursor++;
        return dq.at(cursor - 1);
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
    // A new array of a's own type: copyOf of a String[] is a String[].
    static Object newLike(Object[] a, int n) { return Array.newInstance(a.getClass().getComponentType(), n); }
    static int span(int from, int to) {
        if (from > to) { throw new IllegalArgumentException(from + " > " + to); }
        return to - from;
    }
    @SuppressWarnings("unchecked")
    public static <T> T[] copyOf(T[] a, int n) { return (T[]) copyInto(a, a.length, 0, n, newLike(a, n)); }
    public static int[] copyOf(int[] a, int n) { return (int[]) copyInto(a, a.length, 0, n, new int[n]); }
    public static long[] copyOf(long[] a, int n) { return (long[]) copyInto(a, a.length, 0, n, new long[n]); }
    public static double[] copyOf(double[] a, int n) { return (double[]) copyInto(a, a.length, 0, n, new double[n]); }
    public static float[] copyOf(float[] a, int n) { return (float[]) copyInto(a, a.length, 0, n, new float[n]); }
    public static short[] copyOf(short[] a, int n) { return (short[]) copyInto(a, a.length, 0, n, new short[n]); }
    public static byte[] copyOf(byte[] a, int n) { return (byte[]) copyInto(a, a.length, 0, n, new byte[n]); }
    public static char[] copyOf(char[] a, int n) { return (char[]) copyInto(a, a.length, 0, n, new char[n]); }
    public static boolean[] copyOf(boolean[] a, int n) { return (boolean[]) copyInto(a, a.length, 0, n, new boolean[n]); }
    @SuppressWarnings("unchecked")
    public static <T> T[] copyOfRange(T[] a, int from, int to) { return (T[]) copyInto(a, a.length, from, to, newLike(a, span(from, to))); }
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

    // A sequential stream over the array (lib/java-stream.java).
    public static <T> java.util.stream.Stream<T> stream(T[] a) { return Stream.of(a); }
    public static java.util.stream.IntStream stream(int[] a) { return IntStream.of(a); }
    public static java.util.stream.LongStream stream(long[] a) { return LongStream.of(a); }
    public static java.util.stream.DoubleStream stream(double[] a) { return DoubleStream.of(a); }

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

// Optional, OptionalInt and OptionalDouble: a value or none. The empty one of
// each kind is a single instance, as java's is, made at its first use.
final class Optional<T> {
    static Optional<?> none;
    T value;
    Optional(T value) { this.value = value; }

    @SuppressWarnings("unchecked")
    public static <T> Optional<T> empty() {
        if (none == null) { none = new Optional<Object>(null); }
        return (Optional<T>) none;
    }
    public static <T> Optional<T> of(T value) {
        if (value == null) { throw new NullPointerException(); }
        return new Optional<T>(value);
    }
    public static <T> Optional<T> ofNullable(T value) { return value == null ? empty() : new Optional<T>(value); }

    public boolean isPresent() { return value != null; }
    public boolean isEmpty() { return value == null; }
    public T get() { return orElseThrow(); }
    public T orElseThrow() {
        if (value == null) { throw new NoSuchElementException("No value present"); }
        return value;
    }
    public T orElse(T other) { return value != null ? value : other; }
    public T orElseGet(java.util.function.Supplier<? extends T> f) { return value != null ? value : f.get(); }
    public void ifPresent(java.util.function.Consumer<? super T> f) {
        if (value != null) { f.accept(value); }
    }
    public <U> Optional<U> map(java.util.function.Function<? super T, ? extends U> f) {
        return value == null ? empty() : ofNullable(f.apply(value));
    }
    public Optional<T> filter(java.util.function.Predicate<? super T> p) {
        return value == null || p.test(value) ? this : empty();
    }
    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof Optional)) { return false; }
        Object v = ((Optional<?>) o).value;
        return value == null ? v == null : value.equals(v);
    }
    public int hashCode() { return value == null ? 0 : value.hashCode(); }
    public String toString() { return value == null ? "Optional.empty" : "Optional[" + value + "]"; }
}

final class OptionalInt {
    static OptionalInt none;
    boolean present;
    int value;
    OptionalInt(boolean present, int value) {
        this.present = present;
        this.value = value;
    }

    public static OptionalInt empty() {
        if (none == null) { none = new OptionalInt(false, 0); }
        return none;
    }
    public static OptionalInt of(int value) { return new OptionalInt(true, value); }

    public boolean isPresent() { return present; }
    public boolean isEmpty() { return !present; }
    public int getAsInt() { return orElseThrow(); }
    public int orElseThrow() {
        if (!present) { throw new NoSuchElementException("No value present"); }
        return value;
    }
    public int orElse(int other) { return present ? value : other; }
    public void ifPresent(java.util.function.IntConsumer f) {
        if (present) { f.accept(value); }
    }
    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof OptionalInt)) { return false; }
        OptionalInt b = (OptionalInt) o;
        return present ? b.present && value == b.value : !b.present;
    }
    public int hashCode() { return present ? value : 0; }
    public String toString() { return present ? "OptionalInt[" + value + "]" : "OptionalInt.empty"; }
}

final class OptionalLong {
    static OptionalLong none;
    boolean present;
    long value;
    OptionalLong(boolean present, long value) {
        this.present = present;
        this.value = value;
    }

    public static OptionalLong empty() {
        if (none == null) { none = new OptionalLong(false, 0L); }
        return none;
    }
    public static OptionalLong of(long value) { return new OptionalLong(true, value); }

    public boolean isPresent() { return present; }
    public boolean isEmpty() { return !present; }
    public long getAsLong() { return orElseThrow(); }
    public long orElseThrow() {
        if (!present) { throw new NoSuchElementException("No value present"); }
        return value;
    }
    public long orElse(long other) { return present ? value : other; }
    public String toString() { return present ? "OptionalLong[" + value + "]" : "OptionalLong.empty"; }
}

final class OptionalDouble {
    static OptionalDouble none;
    boolean present;
    double value;
    OptionalDouble(boolean present, double value) {
        this.present = present;
        this.value = value;
    }

    public static OptionalDouble empty() {
        if (none == null) { none = new OptionalDouble(false, 0.0); }
        return none;
    }
    public static OptionalDouble of(double value) { return new OptionalDouble(true, value); }

    public boolean isPresent() { return present; }
    public boolean isEmpty() { return !present; }
    public double getAsDouble() { return orElseThrow(); }
    public double orElseThrow() {
        if (!present) { throw new NoSuchElementException("No value present"); }
        return value;
    }
    public double orElse(double other) { return present ? value : other; }
    public String toString() { return present ? "OptionalDouble[" + value + "]" : "OptionalDouble.empty"; }
}
