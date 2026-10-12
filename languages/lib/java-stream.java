// java-stream.java -- the part of java.util.stream both java grammars provide,
// written in Java and compiled with the program: java-interpreter.abnf and
// java-to-llvm-ir.abnf load it (with lib/java-util.java, which it builds on)
// when the program imports a java.util.stream type or calls a method named
// stream (jStreamWants in lib/java-decltypes.js). One implementation serves
// all three engines. What it answers is java 24.0.2's for a SEQUENTIAL stream.
//
// Stream and IntStream are classes here, not interfaces, and Collector is one
// class holding its three functions. Every stage of a pipeline is an Iterator
// over the elements it passes on, so pulling an element from the last stage
// pulls it through every stage before it: each element passes the whole chain
// before the next one is read, which is the order a sequential java stream
// runs its lambdas in, and a short-circuiting operation (limit, findFirst,
// anyMatch) reads no further than it needs. sorted() is the one barrier: its
// first read drains everything upstream.
package java.util.stream;

// The source of Stream.of / IntStream.of / Arrays.stream: a[from..to).
class ArrayPipe implements Iterator<Object> {
    Object[] a;
    int i;
    int to;
    ArrayPipe(Object[] a, int from, int to) {
        this.a = a;
        i = from;
        this.to = to;
    }
    public boolean hasNext() { return i < to; }
    public Object next() {
        i++;
        return a[i - 1];
    }
}

// IntStream.range / rangeClosed: the counter is a long, so a range ending at
// Integer.MAX_VALUE stops instead of wrapping.
class RangePipe implements Iterator<Object> {
    long cur;
    long end;
    RangePipe(long from, long end) {
        cur = from;
        this.end = end;
    }
    public boolean hasNext() { return cur < end; }
    public Object next() {
        cur++;
        return (int) (cur - 1);
    }
}

// Collection.stream(): the collection's iterator is taken at the first read,
// not when the stream is made, as a late-binding spliterator does.
class LazyPipe implements Iterator<Object> {
    Collection<?> c;
    Iterator<?> it;
    LazyPipe(Collection<?> c) { this.c = c; }
    public boolean hasNext() {
        if (it == null) { it = c.iterator(); }
        return it.hasNext();
    }
    public Object next() {
        if (it == null) { it = c.iterator(); }
        return it.next();
    }
}

// map / mapToObj (Function.apply, IntFunction.apply) and mapToInt / IntStream's
// map (ToIntFunction.applyAsInt, IntUnaryOperator.applyAsInt).
class MapPipe implements Iterator<Object> {
    Iterator<Object> up;
    Object f;
    boolean asInt;
    MapPipe(Iterator<Object> up, Object f, boolean asInt) {
        this.up = up;
        this.f = f;
        this.asInt = asInt;
    }
    public boolean hasNext() { return up.hasNext(); }
    @SuppressWarnings("unchecked")
    public Object next() {
        Object x = up.next();
        if (asInt) { return ((java.util.function.ToIntFunction<Object>) f).applyAsInt(x); }
        return ((java.util.function.Function<Object, Object>) f).apply(x);
    }
}

// filter (Predicate.test / IntPredicate.test): hasNext reads ahead to the next
// element that passes and holds it.
class FilterPipe implements Iterator<Object> {
    Iterator<Object> up;
    java.util.function.Predicate<Object> p;
    Object held;
    boolean holding;
    FilterPipe(Iterator<Object> up, java.util.function.Predicate<Object> p) {
        this.up = up;
        this.p = p;
    }
    public boolean hasNext() {
        while (!holding && up.hasNext()) {
            Object x = up.next();
            if (p.test(x)) {
                held = x;
                holding = true;
            }
        }
        return holding;
    }
    public Object next() {
        if (!hasNext()) { throw new NoSuchElementException(); }
        holding = false;
        return held;
    }
}

// peek (Consumer.accept / IntConsumer.accept).
class PeekPipe implements Iterator<Object> {
    Iterator<Object> up;
    java.util.function.Consumer<Object> f;
    PeekPipe(Iterator<Object> up, java.util.function.Consumer<Object> f) {
        this.up = up;
        this.f = f;
    }
    public boolean hasNext() { return up.hasNext(); }
    public Object next() {
        Object x = up.next();
        f.accept(x);
        return x;
    }
}

// limit(n) and skip(n). skip reads past its first n elements at the first read.
class SlicePipe implements Iterator<Object> {
    Iterator<Object> up;
    long skip;
    long left;
    SlicePipe(Iterator<Object> up, long skip, long limit) {
        this.up = up;
        this.skip = skip;
        left = limit;
    }
    public boolean hasNext() {
        while (skip > 0 && up.hasNext()) {
            up.next();
            skip--;
        }
        return left != 0 && up.hasNext();
    }
    public Object next() {
        if (!hasNext()) { throw new NoSuchElementException(); }
        left--;
        return up.next();
    }
}

// sorted() and sorted(Comparator): the first read drains the upstream stages
// and sorts what they produced, stably (Arrays.sort's merge sort).
class SortPipe implements Iterator<Object> {
    Iterator<Object> up;
    java.util.Comparator<Object> c;
    Object[] all;
    int i;
    SortPipe(Iterator<Object> up, java.util.Comparator<Object> c) {
        this.up = up;
        this.c = c;
    }
    public boolean hasNext() {
        if (all == null) {
            all = Streams.drain(up);
            if (c == null) { Arrays.sort(all); } else { Arrays.sort(all, c); }
        }
        return i < all.length;
    }
    public Object next() {
        if (!hasNext()) { throw new NoSuchElementException(); }
        i++;
        return all[i - 1];
    }
}

// distinct(): the first occurrence of each element by equals, in encounter order.
class DistinctPipe implements Iterator<Object> {
    Iterator<Object> up;
    ArrayList<Object> seen = new ArrayList<Object>();
    Object held;
    boolean holding;
    DistinctPipe(Iterator<Object> up) { this.up = up; }
    public boolean hasNext() {
        while (!holding && up.hasNext()) {
            Object x = up.next();
            if (!seen.contains(x)) {
                seen.add(x);
                held = x;
                holding = true;
            }
        }
        return holding;
    }
    public Object next() {
        if (!hasNext()) { throw new NoSuchElementException(); }
        holding = false;
        return held;
    }
}

class Streams {
    private Streams() {}
    static Object[] drain(Iterator<Object> it) {
        Object[] buf = new Object[16];
        int n = 0;
        while (it.hasNext()) {
            if (n == buf.length) { buf = Arrays.copyOf(buf, n * 2); }
            buf[n] = it.next();
            n++;
        }
        return Arrays.copyOf(buf, n);
    }
    static void checkCount(long n) {
        if (n < 0) { throw new IllegalArgumentException(Long.toString(n)); }
    }
}

// What both kinds of stream share: the last stage, and the single-use check
// every operation makes before it links a new stage or runs.
abstract class BaseStream {
    Iterator<Object> it;
    boolean linked;
    Iterator<Object> take() {
        if (linked) { throw new IllegalStateException("stream has already been operated upon or closed"); }
        linked = true;
        return it;
    }
}

class Stream<T> extends BaseStream {
    Stream(Iterator<Object> it) { this.it = it; }

    @SafeVarargs
    public static <T> Stream<T> of(T... a) { return new Stream<T>(new ArrayPipe(a, 0, a.length)); }
    public static <T> Stream<T> empty() { return new Stream<T>(new ArrayPipe(new Object[0], 0, 0)); }
    static <T> Stream<T> ofCollection(Collection<T> c) { return new Stream<T>(new LazyPipe(c)); }

    public <R> Stream<R> map(java.util.function.Function<? super T, ? extends R> f) { return new Stream<R>(new MapPipe(take(), f, false)); }
    public IntStream mapToInt(java.util.function.ToIntFunction<? super T> f) { return new IntStream(new MapPipe(take(), f, true)); }
    @SuppressWarnings("unchecked")
    public Stream<T> filter(java.util.function.Predicate<? super T> p) { return new Stream<T>(new FilterPipe(take(), (java.util.function.Predicate<Object>) p)); }
    @SuppressWarnings("unchecked")
    public Stream<T> peek(java.util.function.Consumer<? super T> f) { return new Stream<T>(new PeekPipe(take(), (java.util.function.Consumer<Object>) f)); }
    public Stream<T> limit(long n) {
        Streams.checkCount(n);
        return new Stream<T>(new SlicePipe(take(), 0, n));
    }
    public Stream<T> skip(long n) {
        Streams.checkCount(n);
        return new Stream<T>(new SlicePipe(take(), n, -1));
    }
    public Stream<T> sorted() { return new Stream<T>(new SortPipe(take(), null)); }
    @SuppressWarnings("unchecked")
    public Stream<T> sorted(java.util.Comparator<? super T> c) { return new Stream<T>(new SortPipe(take(), (java.util.Comparator<Object>) c)); }
    public Stream<T> distinct() { return new Stream<T>(new DistinctPipe(take())); }

    @SuppressWarnings("unchecked")
    public void forEach(java.util.function.Consumer<? super T> f) {
        Iterator<Object> i = take();
        while (i.hasNext()) { f.accept((T) i.next()); }
    }
    public long count() {
        Iterator<Object> i = take();
        long n = 0;
        while (i.hasNext()) {
            i.next();
            n++;
        }
        return n;
    }
    @SuppressWarnings("unchecked")
    public T reduce(T identity, java.util.function.BinaryOperator<T> op) {
        Iterator<Object> i = take();
        T r = identity;
        while (i.hasNext()) { r = op.apply(r, (T) i.next()); }
        return r;
    }
    @SuppressWarnings("unchecked")
    public Optional<T> reduce(java.util.function.BinaryOperator<T> op) {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return Optional.empty(); }
        T r = (T) i.next();
        while (i.hasNext()) { r = op.apply(r, (T) i.next()); }
        return Optional.of(r);
    }
    @SuppressWarnings("unchecked")
    public <R> R collect(Collector<? super T, ?, R> c) {
        Iterator<Object> i = take();
        Object acc = c.supplier.get();
        while (i.hasNext()) { c.accumulator.accept(acc, i.next()); }
        return (R) c.finisher.apply(acc);
    }
    @SuppressWarnings("unchecked")
    public List<T> toList() {
        ArrayList<T> l = new ArrayList<T>();
        Iterator<Object> i = take();
        while (i.hasNext()) { l.add((T) i.next()); }
        l.frozen = true;
        return l;
    }
    public Object[] toArray() { return Streams.drain(take()); }
    @SuppressWarnings("unchecked")
    public Optional<T> findFirst() {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return Optional.empty(); }
        return Optional.of((T) i.next());
    }
    public Optional<T> min(java.util.Comparator<? super T> c) { return best(c, -1); }
    public Optional<T> max(java.util.Comparator<? super T> c) { return best(c, 1); }
    // The first of the extreme elements wins a tie for min and the last for max,
    // as reduce((a, b) -> c.compare(a, b) <= 0 ? a : b) and its mirror do.
    @SuppressWarnings("unchecked")
    Optional<T> best(java.util.Comparator<? super T> c, int sign) {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return Optional.empty(); }
        T r = (T) i.next();
        while (i.hasNext()) {
            T x = (T) i.next();
            int k = c.compare(r, x);
            if (sign < 0 ? k > 0 : k <= 0) { r = x; }
        }
        return Optional.of(r);
    }
    public boolean anyMatch(java.util.function.Predicate<? super T> p) { return match(p, true); }
    public boolean allMatch(java.util.function.Predicate<? super T> p) { return !match(p, false); }
    public boolean noneMatch(java.util.function.Predicate<? super T> p) { return !match(p, true); }
    // Does some element test `want`? It stops at the first that does.
    @SuppressWarnings("unchecked")
    boolean match(java.util.function.Predicate<? super T> p, boolean want) {
        Iterator<Object> i = take();
        while (i.hasNext()) {
            if (p.test((T) i.next()) == want) { return true; }
        }
        return false;
    }
}

class IntStream extends BaseStream {
    IntStream(Iterator<Object> it) { this.it = it; }

    public static IntStream of(int... a) { return new IntStream(new ArrayPipe(Arrays.elems(a), 0, a.length)); }
    public static IntStream empty() { return new IntStream(new ArrayPipe(new Object[0], 0, 0)); }
    public static IntStream range(int from, int to) { return new IntStream(new RangePipe(from, to)); }
    public static IntStream rangeClosed(int from, int to) { return new IntStream(new RangePipe(from, (long) to + 1)); }

    public IntStream map(java.util.function.IntUnaryOperator f) { return new IntStream(new MapPipe(take(), f, true)); }
    public <U> Stream<U> mapToObj(java.util.function.IntFunction<? extends U> f) { return new Stream<U>(new MapPipe(take(), f, false)); }
    public Stream<Integer> boxed() { return new Stream<Integer>(take()); }
    @SuppressWarnings("unchecked")
    public IntStream filter(java.util.function.IntPredicate p) { return new IntStream(new FilterPipe(take(), (java.util.function.Predicate<Object>) (Object) p)); }
    @SuppressWarnings("unchecked")
    public IntStream peek(java.util.function.IntConsumer f) { return new IntStream(new PeekPipe(take(), (java.util.function.Consumer<Object>) (Object) f)); }
    public IntStream limit(long n) {
        Streams.checkCount(n);
        return new IntStream(new SlicePipe(take(), 0, n));
    }
    public IntStream skip(long n) {
        Streams.checkCount(n);
        return new IntStream(new SlicePipe(take(), n, -1));
    }
    public IntStream sorted() { return new IntStream(new SortPipe(take(), null)); }
    public IntStream distinct() { return new IntStream(new DistinctPipe(take())); }

    public void forEach(java.util.function.IntConsumer f) {
        Iterator<Object> i = take();
        while (i.hasNext()) { f.accept((int) i.next()); }
    }
    public int sum() {
        Iterator<Object> i = take();
        int s = 0;
        while (i.hasNext()) { s += (int) i.next(); }
        return s;
    }
    public long count() {
        Iterator<Object> i = take();
        long n = 0;
        while (i.hasNext()) {
            i.next();
            n++;
        }
        return n;
    }
    public OptionalDouble average() {
        Iterator<Object> i = take();
        long s = 0;
        long n = 0;
        while (i.hasNext()) {
            s += (int) i.next();
            n++;
        }
        if (n == 0) { return OptionalDouble.empty(); }
        return OptionalDouble.of((double) s / n);
    }
    public OptionalInt min() { return best(-1); }
    public OptionalInt max() { return best(1); }
    OptionalInt best(int sign) {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return OptionalInt.empty(); }
        int r = (int) i.next();
        while (i.hasNext()) {
            int x = (int) i.next();
            if (sign < 0 ? x < r : x > r) { r = x; }
        }
        return OptionalInt.of(r);
    }
    public int reduce(int identity, java.util.function.IntBinaryOperator op) {
        Iterator<Object> i = take();
        int r = identity;
        while (i.hasNext()) { r = op.applyAsInt(r, (int) i.next()); }
        return r;
    }
    public OptionalInt reduce(java.util.function.IntBinaryOperator op) {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return OptionalInt.empty(); }
        int r = (int) i.next();
        while (i.hasNext()) { r = op.applyAsInt(r, (int) i.next()); }
        return OptionalInt.of(r);
    }
    public int[] toArray() {
        Object[] all = Streams.drain(take());
        int[] out = new int[all.length];
        System.arraycopy(all, 0, out, 0, all.length);
        return out;
    }
    public OptionalInt findFirst() {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return OptionalInt.empty(); }
        return OptionalInt.of((int) i.next());
    }
    public boolean anyMatch(java.util.function.IntPredicate p) { return match(p, true); }
    public boolean allMatch(java.util.function.IntPredicate p) { return !match(p, false); }
    public boolean noneMatch(java.util.function.IntPredicate p) { return !match(p, true); }
    boolean match(java.util.function.IntPredicate p, boolean want) {
        Iterator<Object> i = take();
        while (i.hasNext()) {
            if (p.test((int) i.next()) == want) { return true; }
        }
        return false;
    }
}

// A mutable reduction: a new container from supplier, each element folded in
// by accumulator, the result read out by finisher.
final class Collector<T, A, R> {
    java.util.function.Supplier<Object> supplier;
    java.util.function.BiConsumer<Object, Object> accumulator;
    java.util.function.Function<Object, Object> finisher;
    Collector(java.util.function.Supplier<Object> s, java.util.function.BiConsumer<Object, Object> a,
              java.util.function.Function<Object, Object> f) {
        supplier = s;
        accumulator = a;
        finisher = f;
    }
}

// joining's container: the text so far and whether an element was added.
class Joiner {
    String delim;
    String text = "";
    boolean any;
    Joiner(String delim) { this.delim = delim; }
    void add(Object x) {
        text = any ? text + delim + x : "" + x;
        any = true;
    }
}

// counting's container.
class Counter {
    long n;
}

final class Collectors {
    private Collectors() {}

    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, List<T>> toList() {
        return new Collector<T, Object, List<T>>(() -> new ArrayList<Object>(), (l, x) -> ((ArrayList<Object>) l).add(x), l -> l);
    }
    public static Collector<CharSequence, ?, String> joining() { return joining("", "", ""); }
    public static Collector<CharSequence, ?, String> joining(CharSequence delim) { return joining(delim, "", ""); }
    public static Collector<CharSequence, ?, String> joining(CharSequence delim, CharSequence prefix, CharSequence suffix) {
        String d = delim.toString();
        String p = prefix.toString();
        String s = suffix.toString();
        return new Collector<CharSequence, Object, String>(() -> new Joiner(d), (j, x) -> ((Joiner) j).add(x),
                                                           j -> p + ((Joiner) j).text + s);
    }
    public static <T> Collector<T, ?, Long> counting() {
        return new Collector<T, Object, Long>(() -> new Counter(), (k, x) -> ((Counter) k).n++, k -> (long) ((Counter) k).n);
    }
}
