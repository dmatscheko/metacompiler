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

    public ArrayList() { elems = new Object[10]; }
    public ArrayList(int capacity) {
        if (capacity < 0) { throw new IllegalArgumentException("Illegal Capacity: " + capacity); }
        elems = new Object[capacity];
    }
    public ArrayList(Collection<? extends E> c) {
        elems = new Object[c.size()];
        Iterator<? extends E> it = c.iterator();
        while (it.hasNext()) { add(it.next()); }
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
        for (int i = 0; i < count; i++) { more[i] = elems[i]; }
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
        if (count == elems.length) { grow(); }
        for (int i = count; i > index; i--) { elems[i] = elems[i - 1]; }
        elems[index] = e;
        count++;
    }
    @SuppressWarnings("unchecked")
    public E remove(int index) {
        writable();
        checkIndex(index);
        E old = (E) elems[index];
        for (int i = index; i < count - 1; i++) { elems[i] = elems[i + 1]; }
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
        for (int i = 0; i < count; i++) { elems[i] = null; }
        count = 0;
    }
    public boolean addAll(Collection<? extends E> c) {
        writable();
        Object[] all = new Object[c.size()];
        int n = 0;
        Iterator<? extends E> it = c.iterator();
        while (it.hasNext()) { all[n] = it.next(); n++; }
        for (int i = 0; i < n; i++) {
            if (count == elems.length) { grow(); }
            elems[count] = all[i];
            count++;
        }
        return n > 0;
    }
    public Object[] toArray() {
        Object[] out = new Object[count];
        for (int i = 0; i < count; i++) { out[i] = elems[i]; }
        return out;
    }
    public Iterator<E> iterator() { return new ArrayListItr<E>(this); }
    @SuppressWarnings("unchecked")
    public void forEach(java.util.function.Consumer<? super E> action) {
        for (int i = 0; i < count; i++) { action.accept((E) elems[i]); }
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
        for (int i = kept; i < count; i++) { elems[i] = null; }
        count = kept;
        return any;
    }

    // Any List is equal to one with equal elements in the same order; the
    // ArrayList is the only List this file declares.
    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof ArrayList)) { return false; }
        ArrayList<?> other = (ArrayList<?>) o;
        if (other.size() != count) { return false; }
        for (int i = 0; i < count; i++) {
            Object a = elems[i];
            Object b = other.get(i);
            if (!(a == null ? b == null : a.equals(b))) { return false; }
        }
        return true;
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

class ArrayListItr<E> implements Iterator<E> {
    ArrayList<E> list;
    int cursor;

    ArrayListItr(ArrayList<E> list) { this.list = list; }
    public boolean hasNext() { return cursor < list.count; }
    @SuppressWarnings("unchecked")
    public E next() {
        if (cursor >= list.count) { throw new NoSuchElementException(); }
        cursor++;
        return (E) list.elems[cursor - 1];
    }
}
