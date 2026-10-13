// java-map.java -- java.util's Map, HashMap, Set and HashSet, written in Java
// and compiled with the program: java-interpreter.abnf and java-to-llvm-ir.abnf
// load it (after lib/java-util.java, which it builds on) when the program
// imports or creates one of its types, or uses java.util.stream (jMapWants in
// lib/java-decltypes.js).
// One implementation serves all three engines.
//
// A HashMap iterates in java 24.0.2's order because it IS java's table: the
// spread hash h ^ (h >>> 16), a power-of-two table made at the first insert,
// doubled when the size passes three quarters of it with every bin split in
// order, put appending to its bin while computeIfAbsent, compute and merge
// insert at the head and resize BEFORE inserting rather than after. The one
// departure is that a bin is never made a tree: java's order differs from this
// one only once eight keys share a bin of a table of 64 or more.
package java.util;

interface Map<K, V> {
    int size();
    boolean isEmpty();
    V get(Object key);
    V put(K key, V value);
    V remove(Object key);
    boolean containsKey(Object key);
    void clear();
    Collection<Map.Entry<K, V>> entrySet();

    interface Entry<K, V> {
        K getKey();
        V getValue();
        V setValue(V value);

        @SuppressWarnings("unchecked")
        static <K, V> java.util.Comparator<Map.Entry<K, V>> comparingByKey() {
            return (a, b) -> ((Comparable<Object>) a.getKey()).compareTo(b.getKey());
        }
        @SuppressWarnings("unchecked")
        static <K, V> java.util.Comparator<Map.Entry<K, V>> comparingByValue() {
            return (a, b) -> ((Comparable<Object>) a.getValue()).compareTo(b.getValue());
        }
    }
}

class HashMapNode<K, V> implements Map.Entry<K, V> {
    int hash;
    K key;
    V value;
    HashMapNode<K, V> next;
    HashMapNode(int hash, K key, V value, HashMapNode<K, V> next) {
        this.hash = hash;
        this.key = key;
        this.value = value;
        this.next = next;
    }
    public K getKey() { return key; }
    public V getValue() { return value; }
    public V setValue(V v) {
        V old = value;
        value = v;
        return old;
    }
    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof Map.Entry)) { return false; }
        Map.Entry<?, ?> e = (Map.Entry<?, ?>) o;
        return HashMap.same(key, e.getKey()) && HashMap.same(value, e.getValue());
    }
    public int hashCode() { return (key == null ? 0 : key.hashCode()) ^ (value == null ? 0 : value.hashCode()); }
    public String toString() { return key + "=" + value; }
}

class HashMap<K, V> implements Map<K, V> {
    HashMapNode<K, V>[] table;
    int size;
    int threshold;
    int modCount;

    public HashMap() {}
    public HashMap(int initialCapacity) {
        if (initialCapacity < 0) { throw new IllegalArgumentException("Illegal initial capacity: " + initialCapacity); }
        threshold = sizeFor(initialCapacity);
    }
    public HashMap(Map<? extends K, ? extends V> m) { putAll(m); }

    static int sizeFor(int cap) {
        int n = 1;
        while (n < cap) { n = n << 1; }
        return n;
    }
    static int hash(Object key) {
        if (key == null) { return 0; }
        int h = key.hashCode();
        return h ^ (h >>> 16);
    }
    static boolean same(Object a, Object b) { return a == null ? b == null : a.equals(b); }

    @SuppressWarnings("unchecked")
    HashMapNode<K, V>[] resize() {
        HashMapNode<K, V>[] old = table;
        int oldCap = old == null ? 0 : old.length;
        int newCap;
        int newThr = 0;
        if (oldCap > 0) {
            newCap = oldCap << 1;
            if (oldCap >= 16) { newThr = threshold << 1; }
        } else if (threshold > 0) {
            newCap = threshold;
        } else {
            newCap = 16;
            newThr = 12;
        }
        if (newThr == 0) { newThr = newCap * 3 / 4; }
        threshold = newThr;
        HashMapNode<K, V>[] tab = (HashMapNode<K, V>[]) new HashMapNode[newCap];
        table = tab;
        for (int j = 0; j < oldCap; j++) {
            HashMapNode<K, V> e = old[j];
            if (e == null) { continue; }
            HashMapNode<K, V> loHead = null;
            HashMapNode<K, V> loTail = null;
            HashMapNode<K, V> hiHead = null;
            HashMapNode<K, V> hiTail = null;
            while (e != null) {
                HashMapNode<K, V> nx = e.next;
                e.next = null;
                if ((e.hash & oldCap) == 0) {
                    if (loTail == null) { loHead = e; } else { loTail.next = e; }
                    loTail = e;
                } else {
                    if (hiTail == null) { hiHead = e; } else { hiTail.next = e; }
                    hiTail = e;
                }
                e = nx;
            }
            tab[j] = loHead;
            tab[j + oldCap] = hiHead;
        }
        return tab;
    }
    // java makes a bin of eight or more a tree once the table holds 64 bins;
    // below that it grows the table instead, which is the part that moves keys.
    void longBin() {
        if (table.length < 64) { resize(); }
    }

    HashMapNode<K, V> find(Object key) {
        if (table == null) { return null; }
        int h = hash(key);
        HashMapNode<K, V> e = table[h & (table.length - 1)];
        while (e != null) {
            if (e.hash == h && same(key, e.key)) { return e; }
            e = e.next;
        }
        return null;
    }

    V putVal(K key, V value, boolean onlyIfAbsent) {
        int h = hash(key);
        if (table == null) { resize(); }
        int i = h & (table.length - 1);
        HashMapNode<K, V> p = table[i];
        if (p == null) {
            table[i] = new HashMapNode<K, V>(h, key, value, null);
        } else {
            int count = 0;
            while (true) {
                if (p.hash == h && same(key, p.key)) {
                    V old = p.value;
                    if (!onlyIfAbsent || old == null) { p.value = value; }
                    return old;
                }
                if (p.next == null) {
                    p.next = new HashMapNode<K, V>(h, key, value, null);
                    if (count >= 7) { longBin(); }
                    break;
                }
                p = p.next;
                count++;
            }
        }
        modCount++;
        size++;
        if (size > threshold) { resize(); }
        return null;
    }

    // The first steps computeIfAbsent, compute and merge share: grow the table
    // if it is due, and find the key's bin.
    int binFor(int h) {
        if (size > threshold || table == null) { resize(); }
        return h & (table.length - 1);
    }
    // Inserts at the head of bin i, as those three do; `count` is how many keys
    // the bin held.
    void addFirst(int i, int h, K key, V value, int count) {
        table[i] = new HashMapNode<K, V>(h, key, value, table[i]);
        if (count >= 7) { longBin(); }
        modCount++;
        size++;
    }
    static int binCount(HashMapNode<?, ?> e) {
        int n = 0;
        while (e != null) {
            n++;
            e = e.next;
        }
        return n;
    }

    V removeKey(Object key) {
        if (table == null) { return null; }
        int h = hash(key);
        int i = h & (table.length - 1);
        HashMapNode<K, V> prev = null;
        HashMapNode<K, V> e = table[i];
        while (e != null) {
            if (e.hash == h && same(key, e.key)) {
                if (prev == null) { table[i] = e.next; } else { prev.next = e.next; }
                modCount++;
                size--;
                return e.value;
            }
            prev = e;
            e = e.next;
        }
        return null;
    }

    public int size() { return size; }
    public boolean isEmpty() { return size == 0; }
    public V get(Object key) {
        HashMapNode<K, V> e = find(key);
        return e == null ? null : e.value;
    }
    public V getOrDefault(Object key, V dflt) {
        HashMapNode<K, V> e = find(key);
        return e == null ? dflt : e.value;
    }
    public boolean containsKey(Object key) { return find(key) != null; }
    public boolean containsValue(Object value) {
        HashMapItr<K, V> it = new HashMapItr<K, V>(this, 1);
        while (it.hasNext()) {
            if (same(value, it.next())) { return true; }
        }
        return false;
    }
    public V put(K key, V value) { return putVal(key, value, false); }
    public V putIfAbsent(K key, V value) { return putVal(key, value, true); }
    // Sized for m's entries up front - an unmade table for ceil(size / 0.75),
    // rounded up to a power of two - then filled in m's own order.
    @SuppressWarnings("unchecked")
    public void putAll(Map<? extends K, ? extends V> m) {
        int s = m.size();
        if (s == 0) { return; }
        if (table == null) {
            int t = (int) Math.ceil(s / 0.75);
            if (t > threshold) { threshold = sizeFor(t); }
        } else {
            while (s > threshold) { resize(); }
        }
        Iterator<?> it = m.entrySet().iterator();
        while (it.hasNext()) {
            Map.Entry<K, V> e = (Map.Entry<K, V>) it.next();
            putVal(e.getKey(), e.getValue(), false);
        }
    }
    public V remove(Object key) { return removeKey(key); }
    public void clear() {
        modCount++;
        if (table != null && size > 0) {
            size = 0;
            for (int i = 0; i < table.length; i++) { table[i] = null; }
        }
    }

    public V computeIfAbsent(K key, java.util.function.Function<? super K, ? extends V> f) {
        int h = hash(key);
        int i = binFor(h);
        HashMapNode<K, V> old = null;
        int count = 0;
        for (HashMapNode<K, V> e = table[i]; e != null; e = e.next) {
            if (e.hash == h && same(key, e.key)) {
                old = e;
                break;
            }
            count++;
        }
        if (old != null && old.value != null) { return old.value; }
        int mc = modCount;
        V v = f.apply(key);
        if (mc != modCount) { throw new ConcurrentModificationException(); }
        if (v == null) { return null; }
        if (old != null) {
            old.value = v;
            return v;
        }
        addFirst(i, h, key, v, count);
        return v;
    }
    public V computeIfPresent(K key, java.util.function.BiFunction<? super K, ? super V, ? extends V> f) {
        HashMapNode<K, V> e = find(key);
        if (e == null || e.value == null) { return null; }
        int mc = modCount;
        V v = f.apply(key, e.value);
        if (mc != modCount) { throw new ConcurrentModificationException(); }
        if (v == null) {
            removeKey(key);
            return null;
        }
        e.value = v;
        return v;
    }
    public V compute(K key, java.util.function.BiFunction<? super K, ? super V, ? extends V> f) {
        int h = hash(key);
        int i = binFor(h);
        HashMapNode<K, V> old = null;
        for (HashMapNode<K, V> e = table[i]; e != null; e = e.next) {
            if (e.hash == h && same(key, e.key)) {
                old = e;
                break;
            }
        }
        int mc = modCount;
        V v = f.apply(key, old == null ? null : old.value);
        if (mc != modCount) { throw new ConcurrentModificationException(); }
        if (old != null) {
            if (v != null) { old.value = v; } else { removeKey(key); }
        } else if (v != null) {
            addFirst(i, h, key, v, binCount(table[i]));
        }
        return v;
    }
    public V merge(K key, V value, java.util.function.BiFunction<? super V, ? super V, ? extends V> f) {
        if (value == null || f == null) { throw new NullPointerException(); }
        int h = hash(key);
        int i = binFor(h);
        HashMapNode<K, V> old = null;
        for (HashMapNode<K, V> e = table[i]; e != null; e = e.next) {
            if (e.hash == h && same(key, e.key)) {
                old = e;
                break;
            }
        }
        if (old == null) {
            addFirst(i, h, key, value, binCount(table[i]));
            return value;
        }
        V v = value;
        if (old.value != null) {
            int mc = modCount;
            v = f.apply(old.value, value);
            if (mc != modCount) { throw new ConcurrentModificationException(); }
        }
        if (v != null) { old.value = v; } else { removeKey(key); }
        return v;
    }
    public void forEach(java.util.function.BiConsumer<? super K, ? super V> f) {
        int mc = modCount;
        if (table != null) {
            for (int i = 0; i < table.length && modCount == mc; i++) {
                for (HashMapNode<K, V> e = table[i]; e != null; e = e.next) { f.accept(e.key, e.value); }
            }
        }
        if (modCount != mc) { throw new ConcurrentModificationException(); }
    }
    public void replaceAll(java.util.function.BiFunction<? super K, ? super V, ? extends V> f) {
        int mc = modCount;
        if (table != null) {
            for (int i = 0; i < table.length; i++) {
                for (HashMapNode<K, V> e = table[i]; e != null; e = e.next) { e.value = f.apply(e.key, e.value); }
            }
        }
        if (modCount != mc) { throw new ConcurrentModificationException(); }
    }

    public Collection<K> keySet() { return new HashMapView<K>(this, 0); }
    public Collection<V> values() { return new HashMapView<V>(this, 1); }
    public Collection<Map.Entry<K, V>> entrySet() { return new HashMapView<Map.Entry<K, V>>(this, 2); }

    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof Map)) { return false; }
        Map<?, ?> m = (Map<?, ?>) o;
        if (m.size() != size) { return false; }
        HashMapItr<K, V> it = new HashMapItr<K, V>(this, 2);
        while (it.hasNext()) {
            HashMapNode<?, ?> e = (HashMapNode<?, ?>) it.next();
            if (e.value == null ? !(m.get(e.key) == null && m.containsKey(e.key)) : !e.value.equals(m.get(e.key))) { return false; }
        }
        return true;
    }
    public int hashCode() {
        int h = 0;
        HashMapItr<K, V> it = new HashMapItr<K, V>(this, 2);
        while (it.hasNext()) { h += it.next().hashCode(); }
        return h;
    }
    public String toString() {
        String s = "{";
        HashMapItr<K, V> it = new HashMapItr<K, V>(this, 2);
        while (it.hasNext()) {
            HashMapNode<?, ?> e = (HashMapNode<?, ?>) it.next();
            if (s.length() > 1) { s = s + ", "; }
            s = s + (e.key == this ? "(this Map)" : "" + e.key) + "=" + (e.value == this ? "(this Map)" : "" + e.value);
        }
        return s + "}";
    }
}

// The table in bin order, each bin front to back; kind 0 answers the keys, 1 the
// values, 2 the entries. Fail-fast, as java's is.
class HashMapItr<K, V> implements Iterator<Object> {
    HashMap<K, V> map;
    int kind;
    int expected;
    HashMapNode<K, V> nextNode;
    int index;
    HashMapItr(HashMap<K, V> map, int kind) {
        this.map = map;
        this.kind = kind;
        expected = map.modCount;
        advance(null);
    }
    void advance(HashMapNode<K, V> from) {
        nextNode = from;
        HashMapNode<K, V>[] t = map.table;
        while (nextNode == null && t != null && index < t.length) {
            nextNode = t[index];
            index++;
        }
    }
    public boolean hasNext() { return nextNode != null; }
    public Object next() {
        if (map.modCount != expected) { throw new ConcurrentModificationException(); }
        HashMapNode<K, V> e = nextNode;
        if (e == null) { throw new NoSuchElementException(); }
        advance(e.next);
        return kind == 0 ? e.key : (kind == 1 ? (Object) e.value : e);
    }
}

// keySet(), values() and entrySet(): live views of the map. Removing through a
// view removes the mapping; adding throws, as java's views do.
class HashMapView<E> implements Collection<E> {
    HashMap<?, ?> map;
    int kind;
    HashMapView(HashMap<?, ?> map, int kind) {
        this.map = map;
        this.kind = kind;
    }
    public int size() { return map.size; }
    public boolean isEmpty() { return map.size == 0; }
    @SuppressWarnings("unchecked")
    public Iterator<E> iterator() { return (Iterator<E>) new HashMapItr(map, kind); }
    // The entry of this view holding o, or null.
    HashMapNode<?, ?> entryOf(Object o) {
        if (kind == 0) { return map.find(o); }
        HashMapItr<?, ?> it = new HashMapItr(map, 2);
        while (it.hasNext()) {
            HashMapNode<?, ?> e = (HashMapNode<?, ?>) it.next();
            if (kind == 1 ? HashMap.same(o, e.value) : e.equals(o)) { return e; }
        }
        return null;
    }
    public boolean contains(Object o) { return entryOf(o) != null; }
    public boolean remove(Object o) {
        HashMapNode<?, ?> e = entryOf(o);
        if (e == null) { return false; }
        map.removeKey(e.key);
        return true;
    }
    public boolean add(E e) { throw new UnsupportedOperationException(); }
    public void clear() { map.clear(); }
    public String toString() {
        String s = "[";
        Iterator<E> it = iterator();
        while (it.hasNext()) {
            Object e = it.next();
            if (s.length() > 1) { s = s + ", "; }
            s = s + (e == this ? "(this Collection)" : "" + e);
        }
        return s + "]";
    }
}

// java.util.TreeMap: the entries in key order (the comparator's, else the keys'
// compareTo) in one sorted array, found by binary search. keySet, values and
// entrySet are live views in that order.
class TreeMap<K, V> implements Map<K, V> {
    Comparator<? super K> cmp;
    Object[] keys;
    Object[] vals;
    int size;

    public TreeMap() { this(null); }
    public TreeMap(Comparator<? super K> cmp) {
        this.cmp = cmp;
        keys = new Object[8];
        vals = new Object[8];
    }
    @SuppressWarnings("unchecked")
    int order(Object a, Object b) {
        if (cmp == null) {
            if (a == null || b == null) { throw new NullPointerException(); }
            return ((Comparable<Object>) a).compareTo(b);
        }
        return ((Comparator<Object>) cmp).compare(a, b);
    }
    // The index of key, or -(insertion point) - 1.
    int search(Object key) {
        int lo = 0;
        int hi = size - 1;
        while (lo <= hi) {
            int mid = (lo + hi) >>> 1;
            int c = order(keys[mid], key);
            if (c < 0) { lo = mid + 1; } else if (c > 0) { hi = mid - 1; } else { return mid; }
        }
        return -(lo + 1);
    }
    @SuppressWarnings("unchecked")
    K keyAt(int i) { return (K) keys[i]; }
    @SuppressWarnings("unchecked")
    V valAt(int i) { return (V) vals[i]; }
    public int size() { return size; }
    public boolean isEmpty() { return size == 0; }
    public V get(Object key) {
        int i = search(key);
        return i >= 0 ? valAt(i) : null;
    }
    public V getOrDefault(Object key, V dflt) {
        int i = search(key);
        return i >= 0 ? valAt(i) : dflt;
    }
    public boolean containsKey(Object key) { return search(key) >= 0; }
    public boolean containsValue(Object value) {
        for (int i = 0; i < size; i++) { if (HashMap.same(value, vals[i])) { return true; } }
        return false;
    }
    public V put(K key, V value) {
        if (size == 0) { order(key, key); }
        int i = search(key);
        if (i >= 0) {
            V old = valAt(i);
            vals[i] = value;
            return old;
        }
        int at = -i - 1;
        if (size == keys.length) {
            Object[] nk = new Object[size * 2];
            Object[] nv = new Object[size * 2];
            System.arraycopy(keys, 0, nk, 0, size);
            System.arraycopy(vals, 0, nv, 0, size);
            keys = nk;
            vals = nv;
        }
        System.arraycopy(keys, at, keys, at + 1, size - at);
        System.arraycopy(vals, at, vals, at + 1, size - at);
        keys[at] = key;
        vals[at] = value;
        size++;
        return null;
    }
    public V putIfAbsent(K key, V value) {
        V old = get(key);
        if (old == null) { put(key, value); }
        return old;
    }
    public void putAll(Map<? extends K, ? extends V> m) {
        Iterator<? extends Map.Entry<? extends K, ? extends V>> it = m.entrySet().iterator();
        while (it.hasNext()) {
            Map.Entry<? extends K, ? extends V> e = it.next();
            put(e.getKey(), e.getValue());
        }
    }
    V removeAt(int i) {
        V old = valAt(i);
        System.arraycopy(keys, i + 1, keys, i, size - i - 1);
        System.arraycopy(vals, i + 1, vals, i, size - i - 1);
        size--;
        keys[size] = null;
        vals[size] = null;
        return old;
    }
    public V remove(Object key) {
        int i = search(key);
        return i >= 0 ? removeAt(i) : null;
    }
    public void clear() {
        for (int i = 0; i < size; i++) {
            keys[i] = null;
            vals[i] = null;
        }
        size = 0;
    }
    public V merge(K key, V value, java.util.function.BiFunction<? super V, ? super V, ? extends V> f) {
        int i = search(key);
        V v = i >= 0 && vals[i] != null ? f.apply(valAt(i), value) : value;
        if (v == null) { remove(key); } else { put(key, v); }
        return v;
    }
    public V computeIfAbsent(K key, java.util.function.Function<? super K, ? extends V> f) {
        V old = get(key);
        if (old != null) { return old; }
        V v = f.apply(key);
        if (v != null) { put(key, v); }
        return v;
    }
    public void forEach(java.util.function.BiConsumer<? super K, ? super V> f) {
        for (int i = 0; i < size; i++) { f.accept(keyAt(i), valAt(i)); }
    }
    Map.Entry<K, V> entryAt(int i) {
        if (i < 0 || i >= size) { return null; }
        HashMapNode<K, V> e = new HashMapNode<K, V>(0, keyAt(i), valAt(i), null);
        return e;
    }
    K keyOrThrow(int i) {
        if (size == 0) { throw new NoSuchElementException(); }
        return keyAt(i);
    }
    public K firstKey() { return keyOrThrow(0); }
    public K lastKey() { return keyOrThrow(size - 1); }
    public Map.Entry<K, V> firstEntry() { return entryAt(0); }
    public Map.Entry<K, V> lastEntry() { return entryAt(size - 1); }
    public Map.Entry<K, V> pollFirstEntry() {
        Map.Entry<K, V> e = entryAt(0);
        if (e != null) { removeAt(0); }
        return e;
    }
    public Map.Entry<K, V> pollLastEntry() {
        Map.Entry<K, V> e = entryAt(size - 1);
        if (e != null) { removeAt(size - 1); }
        return e;
    }
    // The index of the greatest key below (or at, when inclusive) key, or of
    // the least key above (or at) it; -1 or size when there is none.
    int below(K key, boolean inclusive) {
        int i = search(key);
        if (i >= 0) { return inclusive ? i : i - 1; }
        return -i - 2;
    }
    int above(K key, boolean inclusive) {
        int i = search(key);
        if (i >= 0) { return inclusive ? i : i + 1; }
        return -i - 1;
    }
    K keyOrNull(int i) { return i >= 0 && i < size ? keyAt(i) : null; }
    public K floorKey(K key) { return keyOrNull(below(key, true)); }
    public K lowerKey(K key) { return keyOrNull(below(key, false)); }
    public K ceilingKey(K key) { return keyOrNull(above(key, true)); }
    public K higherKey(K key) { return keyOrNull(above(key, false)); }
    public Map.Entry<K, V> floorEntry(K key) { return entryAt(below(key, true)); }
    public Map.Entry<K, V> ceilingEntry(K key) { return entryAt(above(key, true)); }
    public Collection<K> keySet() { return new TreeMapView<K>(this, 0); }
    public Collection<V> values() { return new TreeMapView<V>(this, 1); }
    public Collection<Map.Entry<K, V>> entrySet() { return new TreeMapView<Map.Entry<K, V>>(this, 2); }
    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof Map)) { return false; }
        Map<?, ?> m = (Map<?, ?>) o;
        if (m.size() != size) { return false; }
        for (int i = 0; i < size; i++) {
            if (!m.containsKey(keys[i]) || !HashMap.same(vals[i], m.get(keys[i]))) { return false; }
        }
        return true;
    }
    public int hashCode() {
        int h = 0;
        for (int i = 0; i < size; i++) { h += (keys[i] == null ? 0 : keys[i].hashCode()) ^ (vals[i] == null ? 0 : vals[i].hashCode()); }
        return h;
    }
    public String toString() {
        String s = "{";
        for (int i = 0; i < size; i++) {
            if (i > 0) { s = s + ", "; }
            s = s + (keys[i] == this ? "(this Map)" : "" + keys[i]) + "=" + (vals[i] == this ? "(this Map)" : "" + vals[i]);
        }
        return s + "}";
    }
}
// A TreeMap's keys (kind 0), values (1) or entries (2) in key order.
class TreeMapView<E> implements Collection<E> {
    TreeMap<?, ?> map;
    int kind;
    TreeMapView(TreeMap<?, ?> map, int kind) {
        this.map = map;
        this.kind = kind;
    }
    public int size() { return map.size; }
    public boolean isEmpty() { return map.size == 0; }
    @SuppressWarnings("unchecked")
    E at(int i) { return (E) (kind == 0 ? map.keys[i] : (kind == 1 ? map.vals[i] : map.entryAt(i))); }
    public Iterator<E> iterator() { return new TreeMapItr<E>(this); }
    public boolean contains(Object o) {
        for (int i = 0; i < map.size; i++) { if (HashMap.same(o, at(i))) { return true; } }
        return false;
    }
    public boolean remove(Object o) {
        for (int i = 0; i < map.size; i++) {
            if (HashMap.same(o, at(i))) {
                map.removeAt(i);
                return true;
            }
        }
        return false;
    }
    public boolean add(E e) { throw new UnsupportedOperationException(); }
    public void clear() { map.clear(); }
    public String toString() {
        String s = "[";
        for (int i = 0; i < map.size; i++) { s = s + (i > 0 ? ", " : "") + at(i); }
        return s + "]";
    }
}
class TreeMapItr<E> implements Iterator<E> {
    TreeMapView<E> view;
    int cursor;
    TreeMapItr(TreeMapView<E> view) { this.view = view; }
    public boolean hasNext() { return cursor < view.map.size; }
    public E next() {
        if (cursor >= view.map.size) { throw new NoSuchElementException(); }
        cursor++;
        return view.at(cursor - 1);
    }
    public void remove() {
        cursor--;
        view.map.removeAt(cursor);
    }
}

interface Set<E> extends Collection<E> {
}

// java.util.HashSet: the keys of a HashMap, as java's is, so it iterates in
// java's order. A collection's copy starts at the capacity java 24 picks,
// HashMap.newHashMap(max(size, 12)).
class HashSet<E> implements Set<E> {
    HashMap<E, Object> map;

    public HashSet() { map = new HashMap<E, Object>(); }
    public HashSet(int initialCapacity) { map = new HashMap<E, Object>(initialCapacity); }
    public HashSet(Collection<? extends E> c) {
        map = new HashMap<E, Object>((Math.max(c.size(), 12) * 4 + 2) / 3);
        addAll(c);
    }

    public int size() { return map.size(); }
    public boolean isEmpty() { return map.isEmpty(); }
    public boolean contains(Object o) { return map.containsKey(o); }
    public boolean add(E e) { return map.put(e, Boolean.TRUE) == null; }
    public boolean remove(Object o) { return map.remove(o) != null; }
    public void clear() { map.clear(); }
    public Iterator<E> iterator() { return map.keySet().iterator(); }
    public boolean addAll(Collection<? extends E> c) {
        boolean changed = false;
        Iterator<? extends E> it = c.iterator();
        while (it.hasNext()) {
            if (add(it.next())) { changed = true; }
        }
        return changed;
    }
    public boolean containsAll(Collection<?> c) {
        Iterator<?> it = c.iterator();
        while (it.hasNext()) {
            if (!contains(it.next())) { return false; }
        }
        return true;
    }
    public boolean removeIf(java.util.function.Predicate<? super E> filter) {
        boolean any = false;
        Iterator<E> it = iterator();
        Object[] gone = new Object[size()];
        int n = 0;
        while (it.hasNext()) {
            E e = it.next();
            if (filter.test(e)) { gone[n] = e; n++; }
        }
        for (int i = 0; i < n; i++) { map.remove(gone[i]); any = true; }
        return any;
    }
    public void forEach(java.util.function.Consumer<? super E> action) {
        Iterator<E> it = iterator();
        while (it.hasNext()) { action.accept(it.next()); }
    }
    public Object[] toArray() {
        Object[] out = new Object[size()];
        Iterator<E> it = iterator();
        for (int i = 0; it.hasNext(); i++) { out[i] = it.next(); }
        return out;
    }

    // Any Set is equal to one holding the same elements.
    public boolean equals(Object o) {
        if (o == this) { return true; }
        if (!(o instanceof Set)) { return false; }
        Set<?> s = (Set<?>) o;
        return s.size() == size() && containsAll(s);
    }
    public int hashCode() {
        int h = 0;
        Iterator<E> it = iterator();
        while (it.hasNext()) {
            E e = it.next();
            if (e != null) { h += e.hashCode(); }
        }
        return h;
    }
    public String toString() {
        String s = "[";
        Iterator<E> it = iterator();
        while (it.hasNext()) {
            E e = it.next();
            if (s.length() > 1) { s = s + ", "; }
            s = s + (e == this ? "(this Collection)" : "" + e);
        }
        return s + "]";
    }
}
