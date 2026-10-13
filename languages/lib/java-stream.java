// java-stream.java -- the part of java.util.stream both java grammars provide,
// written in Java and compiled with the program: java-interpreter.abnf and
// java-to-llvm-ir.abnf load it (after lib/java-util.java and lib/java-map.java,
// which it builds on)
// when the program imports a java.util.stream type or calls a method named
// stream (jStreamWants in lib/java-decltypes.js). One implementation serves
// all three engines. What it answers is java 24.0.2's for a SEQUENTIAL stream.
//
// Stream, IntStream, LongStream and DoubleStream are classes here, not
// interfaces, and Collector is one
// class holding its three functions. Every stage of a pipeline is an Iterator
// over the elements it passes on, so pulling an element from the last stage
// pulls it through every stage before it: each element passes the whole chain
// before the next one is read, which is the order a sequential java stream
// runs its lambdas in, and a short-circuiting operation (limit, findFirst,
// anyMatch) reads no further than it needs. sorted() is the one barrier: its
// first read drains everything upstream.
package java.util.stream;

// The source of Stream.of / IntStream.of / Arrays.stream: a[from..to).
class StreamOfArray implements Iterator<Object> {
    Object[] a;
    int i;
    int to;
    StreamOfArray(Object[] a, int from, int to) {
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

// IntStream and LongStream's range / rangeClosed: from up to and including
// last, which ends a range at Long.MAX_VALUE without wrapping. An IntStream's
// elements are ints.
class StreamOfRange implements Iterator<Object> {
    long cur;
    long last;
    boolean done;
    boolean ints;
    StreamOfRange(long from, long last, boolean ints) {
        cur = from;
        this.last = last;
        done = from > last;
        this.ints = ints;
    }
    public boolean hasNext() { return !done; }
    public Object next() {
        long v = cur;
        if (v == last) { done = true; } else { cur++; }
        if (ints) { return (int) v; }
        return v;
    }
}

// Stream.iterate: seed, then f of the element before, each made when it is
// read, ending at the first element hasNext rejects (null: never).
class StreamIterate implements Iterator<Object> {
    Object cur;
    java.util.function.UnaryOperator<Object> f;
    java.util.function.Predicate<Object> more;
    boolean started;
    boolean ready;
    boolean done;
    StreamIterate(Object seed, java.util.function.UnaryOperator<Object> f, java.util.function.Predicate<Object> more) {
        cur = seed;
        this.f = f;
        this.more = more;
    }
    public boolean hasNext() {
        if (!ready && !done) {
            if (started) { cur = f.apply(cur); }
            started = true;
            ready = true;
            if (more != null && !more.test(cur)) { done = true; }
        }
        return !done;
    }
    public Object next() {
        hasNext();
        ready = false;
        return cur;
    }
}

// Stream.generate: the supplier's next value, without end.
class StreamGenerate implements Iterator<Object> {
    java.util.function.Supplier<Object> s;
    StreamGenerate(java.util.function.Supplier<Object> s) { this.s = s; }
    public boolean hasNext() { return true; }
    public Object next() { return s.get(); }
}

// Collection.stream(): the collection's iterator is taken at the first read,
// not when the stream is made, as a late-binding spliterator does.
class StreamOfCollection implements Iterator<Object> {
    Collection<?> c;
    Iterator<?> it;
    StreamOfCollection(Collection<?> c) { this.c = c; }
    public boolean hasNext() {
        if (it == null) { it = c.iterator(); }
        return it.hasNext();
    }
    public Object next() {
        if (it == null) { it = c.iterator(); }
        return it.next();
    }
}

// map and its mapToX forms. The function's method is named after the element
// type it answers: apply (kind OBJ), applyAsInt, applyAsLong, applyAsDouble.
class StreamMap implements Iterator<Object> {
    static final int OBJ = 0;
    static final int INT = 1;
    static final int LONG = 2;
    static final int DOUBLE = 3;
    Iterator<Object> up;
    Object f;
    int kind;
    StreamMap(Iterator<Object> up, Object f, int kind) {
        this.up = up;
        this.f = f;
        this.kind = kind;
    }
    public boolean hasNext() { return up.hasNext(); }
    @SuppressWarnings("unchecked")
    public Object next() {
        Object x = up.next();
        if (kind == INT) { return ((java.util.function.ToIntFunction<Object>) f).applyAsInt(x); }
        if (kind == LONG) { return ((java.util.function.ToLongFunction<Object>) f).applyAsLong(x); }
        if (kind == DOUBLE) { return ((java.util.function.ToDoubleFunction<Object>) f).applyAsDouble(x); }
        return ((java.util.function.Function<Object, Object>) f).apply(x);
    }
}

// filter (Predicate.test / IntPredicate.test): hasNext reads ahead to the next
// element that passes and holds it.
// flatMap and concat: the elements of each stream f makes of an upstream
// element, one stream after the other; a null stream contributes none.
class StreamFlat implements Iterator<Object> {
    Iterator<Object> up;
    java.util.function.Function<Object, Object> f;
    Iterator<Object> cur;
    StreamFlat(Iterator<Object> up, java.util.function.Function<Object, Object> f) {
        this.up = up;
        this.f = f;
    }
    public boolean hasNext() {
        while (cur == null || !cur.hasNext()) {
            if (!up.hasNext()) { return false; }
            Object s = f.apply(up.next());
            cur = s == null ? null : ((BaseStream) s).take();
        }
        return true;
    }
    public Object next() {
        if (!hasNext()) { throw new NoSuchElementException(); }
        return cur.next();
    }
}

// takeWhile passes elements until the first that fails p and stops there;
// dropWhile skips them and passes that one and every one after.
class StreamWhile implements Iterator<Object> {
    Iterator<Object> up;
    java.util.function.Predicate<Object> p;
    boolean taking;
    boolean done;
    Object held;
    boolean holding;
    StreamWhile(Iterator<Object> up, java.util.function.Predicate<Object> p, boolean taking) {
        this.up = up;
        this.p = p;
        this.taking = taking;
    }
    public boolean hasNext() {
        if (holding) { return true; }
        if (done && taking) { return false; }
        while (up.hasNext()) {
            Object x = up.next();
            if (taking) {
                if (!p.test(x)) {
                    done = true;
                    return false;
                }
            } else if (!done && p.test(x)) {
                continue;
            }
            done = !taking;
            held = x;
            holding = true;
            return true;
        }
        return false;
    }
    public Object next() {
        if (!hasNext()) { throw new NoSuchElementException(); }
        holding = false;
        return held;
    }
}

class StreamFilter implements Iterator<Object> {
    Iterator<Object> up;
    java.util.function.Predicate<Object> p;
    Object held;
    boolean holding;
    StreamFilter(Iterator<Object> up, java.util.function.Predicate<Object> p) {
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
class StreamPeek implements Iterator<Object> {
    Iterator<Object> up;
    java.util.function.Consumer<Object> f;
    StreamPeek(Iterator<Object> up, java.util.function.Consumer<Object> f) {
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
class StreamSlice implements Iterator<Object> {
    Iterator<Object> up;
    long skip;
    long left;
    StreamSlice(Iterator<Object> up, long skip, long limit) {
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
class StreamSort implements Iterator<Object> {
    Iterator<Object> up;
    java.util.Comparator<Object> c;
    Object[] all;
    int i;
    StreamSort(Iterator<Object> up, java.util.Comparator<Object> c) {
        this.up = up;
        this.c = c;
    }
    public boolean hasNext() {
        if (all == null) {
            all = StreamPipes.drain(up);
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
class StreamDistinct implements Iterator<Object> {
    Iterator<Object> up;
    ArrayList<Object> seen = new ArrayList<Object>();
    Object held;
    boolean holding;
    StreamDistinct(Iterator<Object> up) { this.up = up; }
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

class StreamPipes {
    private StreamPipes() {}
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
    // DoubleStream's sum and average: Kahan summation, as java's
    // Collectors.sumWithCompensation and computeFinalSum do it. acc is
    // {sum, compensation, count, plain sum}; the plain sum answers when the
    // compensated one is NaN from adding opposite infinities.
    static void addCompensated(double[] acc, double v) {
        double tmp = v - acc[1];
        double sum = acc[0];
        double velvel = sum + tmp;
        acc[1] = (velvel - sum) - tmp;
        acc[0] = velvel;
        acc[2]++;
        acc[3] += v;
    }
    static double finalSum(double[] acc) {
        double tmp = acc[0] - acc[1];
        if (Double.isNaN(tmp) && Double.isInfinite(acc[3])) { return acc[3]; }
        return tmp;
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
    public long count() {
        Iterator<Object> i = take();
        long n = 0;
        while (i.hasNext()) {
            i.next();
            n++;
        }
        return n;
    }
}

class Stream<T> extends BaseStream {
    Stream(Iterator<Object> it) { this.it = it; }

    @SafeVarargs
    public static <T> Stream<T> of(T... a) { return new Stream<T>(new StreamOfArray(a, 0, a.length)); }
    public static <T> Stream<T> empty() { return new Stream<T>(new StreamOfArray(new Object[0], 0, 0)); }
    static <T> Stream<T> ofCollection(Collection<T> c) { return new Stream<T>(new StreamOfCollection(c)); }
    @SuppressWarnings("unchecked")
    public static <T> Stream<T> iterate(T seed, java.util.function.UnaryOperator<T> f) {
        return new Stream<T>(new StreamIterate(seed, (java.util.function.UnaryOperator<Object>) (Object) f, null));
    }
    @SuppressWarnings("unchecked")
    public static <T> Stream<T> iterate(T seed, java.util.function.Predicate<? super T> more, java.util.function.UnaryOperator<T> f) {
        return new Stream<T>(new StreamIterate(seed, (java.util.function.UnaryOperator<Object>) (Object) f, (java.util.function.Predicate<Object>) more));
    }
    @SuppressWarnings("unchecked")
    public static <T> Stream<T> generate(java.util.function.Supplier<? extends T> s) {
        return new Stream<T>(new StreamGenerate((java.util.function.Supplier<Object>) s));
    }

    public <R> Stream<R> map(java.util.function.Function<? super T, ? extends R> f) { return new Stream<R>(new StreamMap(take(), f, StreamMap.OBJ)); }
    public IntStream mapToInt(java.util.function.ToIntFunction<? super T> f) { return new IntStream(new StreamMap(take(), f, StreamMap.INT)); }
    public LongStream mapToLong(java.util.function.ToLongFunction<? super T> f) { return new LongStream(new StreamMap(take(), f, StreamMap.LONG)); }
    public DoubleStream mapToDouble(java.util.function.ToDoubleFunction<? super T> f) { return new DoubleStream(new StreamMap(take(), f, StreamMap.DOUBLE)); }
    @SuppressWarnings("unchecked")
    public Stream<T> filter(java.util.function.Predicate<? super T> p) { return new Stream<T>(new StreamFilter(take(), (java.util.function.Predicate<Object>) p)); }
    @SuppressWarnings("unchecked")
    public Stream<T> peek(java.util.function.Consumer<? super T> f) { return new Stream<T>(new StreamPeek(take(), (java.util.function.Consumer<Object>) f)); }
    public Stream<T> limit(long n) {
        StreamPipes.checkCount(n);
        return new Stream<T>(new StreamSlice(take(), 0, n));
    }
    public Stream<T> skip(long n) {
        StreamPipes.checkCount(n);
        return new Stream<T>(new StreamSlice(take(), n, -1));
    }
    public Stream<T> sorted() { return new Stream<T>(new StreamSort(take(), null)); }
    @SuppressWarnings("unchecked")
    public Stream<T> sorted(java.util.Comparator<? super T> c) { return new Stream<T>(new StreamSort(take(), (java.util.Comparator<Object>) c)); }
    public Stream<T> distinct() { return new Stream<T>(new StreamDistinct(take())); }
    @SuppressWarnings("unchecked")
    public <R> Stream<R> flatMap(java.util.function.Function<? super T, ? extends Stream<? extends R>> f) {
        return new Stream<R>(new StreamFlat(take(), (java.util.function.Function<Object, Object>) (Object) f));
    }
    public static <T> Stream<T> concat(Stream<? extends T> a, Stream<? extends T> b) {
        return new Stream<T>(new StreamFlat(new StreamOfArray(new Object[]{a, b}, 0, 2), s -> s));
    }
    @SuppressWarnings("unchecked")
    public Stream<T> takeWhile(java.util.function.Predicate<? super T> p) { return new Stream<T>(new StreamWhile(take(), (java.util.function.Predicate<Object>) p, true)); }
    @SuppressWarnings("unchecked")
    public Stream<T> dropWhile(java.util.function.Predicate<? super T> p) { return new Stream<T>(new StreamWhile(take(), (java.util.function.Predicate<Object>) p, false)); }

    @SuppressWarnings("unchecked")
    public void forEach(java.util.function.Consumer<? super T> f) {
        Iterator<Object> i = take();
        while (i.hasNext()) { f.accept((T) i.next()); }
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
    public Object[] toArray() { return StreamPipes.drain(take()); }
    // toArray(String[]::new): the generator's array of the element count.
    public <A> A[] toArray(java.util.function.IntFunction<A[]> g) {
        Object[] all = StreamPipes.drain(take());
        A[] out = g.apply(all.length);
        System.arraycopy(all, 0, out, 0, all.length);
        return out;
    }
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

    public static IntStream of(int... a) { return new IntStream(new StreamOfArray(Arrays.elems(a), 0, a.length)); }
    public static IntStream empty() { return new IntStream(new StreamOfArray(new Object[0], 0, 0)); }
    public static IntStream range(int from, int to) { return new IntStream(new StreamOfRange(from, (long) to - 1, true)); }
    public static IntStream rangeClosed(int from, int to) { return new IntStream(new StreamOfRange(from, to, true)); }
    public static IntStream iterate(int seed, java.util.function.IntUnaryOperator f) {
        return new IntStream(new StreamIterate(seed, x -> f.applyAsInt((int) x), null));
    }
    public static IntStream iterate(int seed, java.util.function.IntPredicate more, java.util.function.IntUnaryOperator f) {
        return new IntStream(new StreamIterate(seed, x -> f.applyAsInt((int) x), x -> more.test((int) x)));
    }

    public IntStream map(java.util.function.IntUnaryOperator f) { return new IntStream(new StreamMap(take(), f, StreamMap.INT)); }
    public <U> Stream<U> mapToObj(java.util.function.IntFunction<? extends U> f) { return new Stream<U>(new StreamMap(take(), f, StreamMap.OBJ)); }
    public LongStream mapToLong(java.util.function.IntToLongFunction f) { return new LongStream(new StreamMap(take(), f, StreamMap.LONG)); }
    public DoubleStream mapToDouble(java.util.function.IntToDoubleFunction f) { return new DoubleStream(new StreamMap(take(), f, StreamMap.DOUBLE)); }
    public LongStream asLongStream() { return mapToLong(x -> (long) x); }
    public DoubleStream asDoubleStream() { return mapToDouble(x -> (double) x); }
    public Stream<Integer> boxed() { return new Stream<Integer>(take()); }
    @SuppressWarnings("unchecked")
    public IntStream filter(java.util.function.IntPredicate p) { return new IntStream(new StreamFilter(take(), (java.util.function.Predicate<Object>) (Object) p)); }
    @SuppressWarnings("unchecked")
    public IntStream peek(java.util.function.IntConsumer f) { return new IntStream(new StreamPeek(take(), (java.util.function.Consumer<Object>) (Object) f)); }
    public IntStream limit(long n) {
        StreamPipes.checkCount(n);
        return new IntStream(new StreamSlice(take(), 0, n));
    }
    public IntStream skip(long n) {
        StreamPipes.checkCount(n);
        return new IntStream(new StreamSlice(take(), n, -1));
    }
    public IntStream sorted() { return new IntStream(new StreamSort(take(), null)); }
    public IntStream distinct() { return new IntStream(new StreamDistinct(take())); }

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
    public IntSummaryStatistics summaryStatistics() {
        IntSummaryStatistics st = new IntSummaryStatistics();
        Iterator<Object> i = take();
        while (i.hasNext()) { st.accept((int) i.next()); }
        return st;
    }
    public int[] toArray() {
        Object[] all = StreamPipes.drain(take());
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

// java.util.IntSummaryStatistics: count, sum, min and max of the ints seen;
// with none, min is Integer.MAX_VALUE and max Integer.MIN_VALUE.
class IntSummaryStatistics {
    long count;
    long sum;
    int min = Integer.MAX_VALUE;
    int max = Integer.MIN_VALUE;

    public void accept(int x) {
        count++;
        sum += x;
        min = Math.min(min, x);
        max = Math.max(max, x);
    }
    public long getCount() { return count; }
    public long getSum() { return sum; }
    public int getMin() { return min; }
    public int getMax() { return max; }
    public double getAverage() { return count > 0 ? (double) sum / count : 0.0; }
    public String toString() {
        return String.format("IntSummaryStatistics{count=%d, sum=%d, min=%d, average=%f, max=%d}", count, sum, min, getAverage(), max);
    }
}

class LongStream extends BaseStream {
    LongStream(Iterator<Object> it) { this.it = it; }

    public static LongStream of(long... a) { return new LongStream(new StreamOfArray(Arrays.elems(a), 0, a.length)); }
    public static LongStream empty() { return new LongStream(new StreamOfArray(new Object[0], 0, 0)); }
    public static LongStream range(long from, long to) {
        if (from >= to) { return empty(); }
        return new LongStream(new StreamOfRange(from, to - 1, false));
    }
    public static LongStream rangeClosed(long from, long to) { return new LongStream(new StreamOfRange(from, to, false)); }

    public LongStream map(java.util.function.LongUnaryOperator f) { return new LongStream(new StreamMap(take(), f, StreamMap.LONG)); }
    public <U> Stream<U> mapToObj(java.util.function.LongFunction<? extends U> f) { return new Stream<U>(new StreamMap(take(), f, StreamMap.OBJ)); }
    public IntStream mapToInt(java.util.function.LongToIntFunction f) { return new IntStream(new StreamMap(take(), f, StreamMap.INT)); }
    public DoubleStream mapToDouble(java.util.function.LongToDoubleFunction f) { return new DoubleStream(new StreamMap(take(), f, StreamMap.DOUBLE)); }
    public DoubleStream asDoubleStream() { return mapToDouble(x -> (double) x); }
    public Stream<Long> boxed() { return new Stream<Long>(take()); }
    @SuppressWarnings("unchecked")
    public LongStream filter(java.util.function.LongPredicate p) { return new LongStream(new StreamFilter(take(), (java.util.function.Predicate<Object>) (Object) p)); }
    @SuppressWarnings("unchecked")
    public LongStream peek(java.util.function.LongConsumer f) { return new LongStream(new StreamPeek(take(), (java.util.function.Consumer<Object>) (Object) f)); }
    public LongStream limit(long n) {
        StreamPipes.checkCount(n);
        return new LongStream(new StreamSlice(take(), 0, n));
    }
    public LongStream skip(long n) {
        StreamPipes.checkCount(n);
        return new LongStream(new StreamSlice(take(), n, -1));
    }
    public LongStream sorted() { return new LongStream(new StreamSort(take(), null)); }
    public LongStream distinct() { return new LongStream(new StreamDistinct(take())); }

    public void forEach(java.util.function.LongConsumer f) {
        Iterator<Object> i = take();
        while (i.hasNext()) { f.accept((long) i.next()); }
    }
    public long sum() { return reduce(0L, (x, y) -> x + y); }
    public OptionalDouble average() {
        Iterator<Object> i = take();
        long s = 0;
        long n = 0;
        while (i.hasNext()) {
            s += (long) i.next();
            n++;
        }
        if (n == 0) { return OptionalDouble.empty(); }
        return OptionalDouble.of((double) s / n);
    }
    public OptionalLong min() { return reduce((x, y) -> Math.min(x, y)); }
    public OptionalLong max() { return reduce((x, y) -> Math.max(x, y)); }
    public long reduce(long identity, java.util.function.LongBinaryOperator op) {
        Iterator<Object> i = take();
        long r = identity;
        while (i.hasNext()) { r = op.applyAsLong(r, (long) i.next()); }
        return r;
    }
    public OptionalLong reduce(java.util.function.LongBinaryOperator op) {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return OptionalLong.empty(); }
        long r = (long) i.next();
        while (i.hasNext()) { r = op.applyAsLong(r, (long) i.next()); }
        return OptionalLong.of(r);
    }
    public long[] toArray() {
        Object[] all = StreamPipes.drain(take());
        long[] out = new long[all.length];
        System.arraycopy(all, 0, out, 0, all.length);
        return out;
    }
    public OptionalLong findFirst() {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return OptionalLong.empty(); }
        return OptionalLong.of((long) i.next());
    }
    public boolean anyMatch(java.util.function.LongPredicate p) { return match(p, true); }
    public boolean allMatch(java.util.function.LongPredicate p) { return !match(p, false); }
    public boolean noneMatch(java.util.function.LongPredicate p) { return !match(p, true); }
    boolean match(java.util.function.LongPredicate p, boolean want) {
        Iterator<Object> i = take();
        while (i.hasNext()) {
            if (p.test((long) i.next()) == want) { return true; }
        }
        return false;
    }
}

// min and max are Math.min / Math.max, so NaN wins and -0.0 is below 0.0;
// sorted() is Double.compare's order.
class DoubleStream extends BaseStream {
    DoubleStream(Iterator<Object> it) { this.it = it; }

    public static DoubleStream of(double... a) { return new DoubleStream(new StreamOfArray(Arrays.elems(a), 0, a.length)); }
    public static DoubleStream empty() { return new DoubleStream(new StreamOfArray(new Object[0], 0, 0)); }

    public DoubleStream map(java.util.function.DoubleUnaryOperator f) { return new DoubleStream(new StreamMap(take(), f, StreamMap.DOUBLE)); }
    public <U> Stream<U> mapToObj(java.util.function.DoubleFunction<? extends U> f) { return new Stream<U>(new StreamMap(take(), f, StreamMap.OBJ)); }
    public IntStream mapToInt(java.util.function.DoubleToIntFunction f) { return new IntStream(new StreamMap(take(), f, StreamMap.INT)); }
    public LongStream mapToLong(java.util.function.DoubleToLongFunction f) { return new LongStream(new StreamMap(take(), f, StreamMap.LONG)); }
    public Stream<Double> boxed() { return new Stream<Double>(take()); }
    @SuppressWarnings("unchecked")
    public DoubleStream filter(java.util.function.DoublePredicate p) { return new DoubleStream(new StreamFilter(take(), (java.util.function.Predicate<Object>) (Object) p)); }
    @SuppressWarnings("unchecked")
    public DoubleStream peek(java.util.function.DoubleConsumer f) { return new DoubleStream(new StreamPeek(take(), (java.util.function.Consumer<Object>) (Object) f)); }
    public DoubleStream limit(long n) {
        StreamPipes.checkCount(n);
        return new DoubleStream(new StreamSlice(take(), 0, n));
    }
    public DoubleStream skip(long n) {
        StreamPipes.checkCount(n);
        return new DoubleStream(new StreamSlice(take(), n, -1));
    }
    public DoubleStream sorted() { return new DoubleStream(new StreamSort(take(), null)); }
    public DoubleStream distinct() { return new DoubleStream(new StreamDistinct(take())); }

    public void forEach(java.util.function.DoubleConsumer f) {
        Iterator<Object> i = take();
        while (i.hasNext()) { f.accept((double) i.next()); }
    }
    double[] sums() {
        Iterator<Object> i = take();
        double[] acc = new double[4];
        while (i.hasNext()) { StreamPipes.addCompensated(acc, (double) i.next()); }
        return acc;
    }
    public double sum() { return StreamPipes.finalSum(sums()); }
    public OptionalDouble average() {
        double[] acc = sums();
        if (acc[2] == 0) { return OptionalDouble.empty(); }
        return OptionalDouble.of(StreamPipes.finalSum(acc) / acc[2]);
    }
    public OptionalDouble min() { return reduce((x, y) -> Math.min(x, y)); }
    public OptionalDouble max() { return reduce((x, y) -> Math.max(x, y)); }
    public double reduce(double identity, java.util.function.DoubleBinaryOperator op) {
        Iterator<Object> i = take();
        double r = identity;
        while (i.hasNext()) { r = op.applyAsDouble(r, (double) i.next()); }
        return r;
    }
    public OptionalDouble reduce(java.util.function.DoubleBinaryOperator op) {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return OptionalDouble.empty(); }
        double r = (double) i.next();
        while (i.hasNext()) { r = op.applyAsDouble(r, (double) i.next()); }
        return OptionalDouble.of(r);
    }
    public double[] toArray() {
        Object[] all = StreamPipes.drain(take());
        double[] out = new double[all.length];
        System.arraycopy(all, 0, out, 0, all.length);
        return out;
    }
    public OptionalDouble findFirst() {
        Iterator<Object> i = take();
        if (!i.hasNext()) { return OptionalDouble.empty(); }
        return OptionalDouble.of((double) i.next());
    }
    public boolean anyMatch(java.util.function.DoublePredicate p) { return match(p, true); }
    public boolean allMatch(java.util.function.DoublePredicate p) { return !match(p, false); }
    public boolean noneMatch(java.util.function.DoublePredicate p) { return !match(p, true); }
    boolean match(java.util.function.DoublePredicate p, boolean want) {
        Iterator<Object> i = take();
        while (i.hasNext()) {
            if (p.test((double) i.next()) == want) { return true; }
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
class StreamJoiner {
    String delim;
    String text = "";
    boolean any;
    StreamJoiner(String delim) { this.delim = delim; }
    void add(Object x) {
        text = any ? text + delim + x : "" + x;
        any = true;
    }
}

// The container of counting, the summing and averaging collectors, minBy,
// maxBy and reducing: a count, an integral and a floating sum, and a value.
class StreamAcc {
    long n;
    long sum;
    double dsum;
    Object v;
}

final class Collectors {
    private Collectors() {}

    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, List<T>> toList() {
        return new Collector<T, Object, List<T>>(() -> new ArrayList<Object>(), (l, x) -> ((ArrayList<Object>) l).add(x), l -> l);
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Set<T>> toSet() {
        return new Collector<T, Object, Set<T>>(() -> new HashSet<Object>(), (h, x) -> ((HashSet<Object>) h).add(x), h -> h);
    }
    public static Collector<CharSequence, ?, String> joining() { return joining("", "", ""); }
    public static Collector<CharSequence, ?, String> joining(CharSequence delim) { return joining(delim, "", ""); }
    public static Collector<CharSequence, ?, String> joining(CharSequence delim, CharSequence prefix, CharSequence suffix) {
        String d = delim.toString();
        String p = prefix.toString();
        String s = suffix.toString();
        return new Collector<CharSequence, Object, String>(() -> new StreamJoiner(d), (j, x) -> ((StreamJoiner) j).add(x),
                                                           j -> p + ((StreamJoiner) j).text + s);
    }
    public static <T, K> Collector<T, ?, Map<K, List<T>>> groupingBy(java.util.function.Function<? super T, ? extends K> f) {
        return groupingBy(f, toList());
    }
    // Each element goes into its key's downstream container, made by
    // computeIfAbsent at the key's first element - which decides the HashMap's
    // order - and the downstream finisher then replaces every container by its
    // result.
    @SuppressWarnings("unchecked")
    public static <T, K, A, D> Collector<T, ?, Map<K, D>> groupingBy(java.util.function.Function<? super T, ? extends K> f,
                                                                     Collector<? super T, A, D> ds) {
        return new Collector<T, Object, Map<K, D>>(() -> new HashMap<Object, Object>(), (m, x) -> {
            Object k = f.apply((T) x);
            if (k == null) { throw new NullPointerException("element cannot be mapped to a null key"); }
            ds.accumulator.accept(((HashMap<Object, Object>) m).computeIfAbsent(k, key -> ds.supplier.get()), x);
        }, m -> {
            ((HashMap<Object, Object>) m).replaceAll((k, v) -> ds.finisher.apply(v));
            return m;
        });
    }
    // groupingBy into the map mapFactory makes (TreeMap::new): the same
    // computeIfAbsent and replaceAll, on that map.
    @SuppressWarnings("unchecked")
    public static <T, K, D, A, M extends Map<K, D>> Collector<T, ?, M> groupingBy(java.util.function.Function<? super T, ? extends K> f,
            java.util.function.Supplier<M> mapFactory, Collector<? super T, A, D> ds) {
        return new Collector<T, Object, M>(() -> mapFactory.get(), (m, x) -> {
            Object k = f.apply((T) x);
            if (k == null) { throw new NullPointerException("element cannot be mapped to a null key"); }
            ds.accumulator.accept(((Map<Object, Object>) m).computeIfAbsent(k, key -> ds.supplier.get()), x);
        }, m -> {
            ((Map<Object, Object>) m).replaceAll((k, v) -> ds.finisher.apply(v));
            return m;
        });
    }
    // partitioningBy: both keys always present, false first, as java's
    // Partition map prints.
    public static <T> Collector<T, ?, Map<Boolean, List<T>>> partitioningBy(java.util.function.Predicate<? super T> p) {
        return partitioningBy(p, toList());
    }
    @SuppressWarnings("unchecked")
    public static <T, D, A> Collector<T, ?, Map<Boolean, D>> partitioningBy(java.util.function.Predicate<? super T> p,
            Collector<? super T, A, D> ds) {
        return new Collector<T, Object, Map<Boolean, D>>(() -> {
            TreeMap<Object, Object> m = new TreeMap<Object, Object>();
            m.put(Boolean.FALSE, ds.supplier.get());
            m.put(Boolean.TRUE, ds.supplier.get());
            return m;
        }, (m, x) -> ds.accumulator.accept(((TreeMap<Object, Object>) m).get(p.test((T) x)), x), m -> {
            ((TreeMap<Object, Object>) m).replaceAll((k, v) -> ds.finisher.apply(v));
            return m;
        });
    }
    // toMap: a repeated key is an IllegalStateException naming both values,
    // unless a merge function combines them.
    @SuppressWarnings("unchecked")
    public static <T, K, U> Collector<T, ?, Map<K, U>> toMap(java.util.function.Function<? super T, ? extends K> kf,
            java.util.function.Function<? super T, ? extends U> vf) {
        return new Collector<T, Object, Map<K, U>>(() -> new HashMap<Object, Object>(), (m, x) -> {
            Object k = kf.apply((T) x);
            Object v = vf.apply((T) x);
            if (v == null) { throw new NullPointerException(); }
            Object old = ((HashMap<Object, Object>) m).putIfAbsent(k, v);
            if (old != null) { throw new IllegalStateException("Duplicate key " + k + " (attempted merging values " + old + " and " + v + ")"); }
        }, m -> m);
    }
    public static <T, K, U> Collector<T, ?, Map<K, U>> toMap(java.util.function.Function<? super T, ? extends K> kf,
            java.util.function.Function<? super T, ? extends U> vf, java.util.function.BinaryOperator<U> merge) {
        return toMap(kf, vf, merge, () -> new HashMap<K, U>());
    }
    @SuppressWarnings("unchecked")
    public static <T, K, U, M extends Map<K, U>> Collector<T, ?, M> toMap(java.util.function.Function<? super T, ? extends K> kf,
            java.util.function.Function<? super T, ? extends U> vf, java.util.function.BinaryOperator<U> merge,
            java.util.function.Supplier<M> mapFactory) {
        return new Collector<T, Object, M>(() -> mapFactory.get(),
            (m, x) -> ((Map<Object, Object>) m).merge(kf.apply((T) x), vf.apply((T) x), (java.util.function.BinaryOperator<Object>) merge),
            m -> m);
    }
    @SuppressWarnings("unchecked")
    public static <T, C extends Collection<T>> Collector<T, ?, C> toCollection(java.util.function.Supplier<C> f) {
        return new Collector<T, Object, C>(() -> f.get(), (c, x) -> ((Collection<Object>) c).add(x), c -> c);
    }
    @SuppressWarnings("unchecked")
    public static <T, U, A, R> Collector<T, ?, R> mapping(java.util.function.Function<? super T, ? extends U> f, Collector<? super U, A, R> ds) {
        return new Collector<T, Object, R>(ds.supplier, (c, x) -> ds.accumulator.accept(c, f.apply((T) x)), ds.finisher);
    }
    @SuppressWarnings("unchecked")
    public static <T, A, R> Collector<T, ?, R> filtering(java.util.function.Predicate<? super T> p, Collector<? super T, A, R> ds) {
        return new Collector<T, Object, R>(ds.supplier, (c, x) -> { if (p.test((T) x)) { ds.accumulator.accept(c, x); } }, ds.finisher);
    }
    @SuppressWarnings("unchecked")
    public static <T, A, R, RR> Collector<T, A, RR> collectingAndThen(Collector<T, A, R> ds, java.util.function.Function<R, RR> f) {
        return new Collector<T, A, RR>(ds.supplier, ds.accumulator, c -> f.apply((R) ds.finisher.apply(c)));
    }
    public static <T> Collector<T, ?, Long> counting() {
        return new Collector<T, Object, Long>(() -> new StreamAcc(), (k, x) -> ((StreamAcc) k).n++, k -> (long) ((StreamAcc) k).n);
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Integer> summingInt(java.util.function.ToIntFunction<? super T> f) {
        return new Collector<T, Object, Integer>(() -> new StreamAcc(), (a, x) -> ((StreamAcc) a).sum += f.applyAsInt((T) x),
                                                 a -> (int) ((StreamAcc) a).sum);
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Long> summingLong(java.util.function.ToLongFunction<? super T> f) {
        return new Collector<T, Object, Long>(() -> new StreamAcc(), (a, x) -> ((StreamAcc) a).sum += f.applyAsLong((T) x),
                                              a -> ((StreamAcc) a).sum);
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Double> summingDouble(java.util.function.ToDoubleFunction<? super T> f) {
        return new Collector<T, Object, Double>(() -> new StreamAcc(), (a, x) -> ((StreamAcc) a).dsum += f.applyAsDouble((T) x),
                                                a -> ((StreamAcc) a).dsum);
    }
    // The averages: the sum over the count, 0.0 for no element.
    static double mean(StreamAcc a, boolean integral) {
        if (a.n == 0) { return 0.0; }
        return (integral ? (double) a.sum : a.dsum) / a.n;
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Double> averagingInt(java.util.function.ToIntFunction<? super T> f) {
        return new Collector<T, Object, Double>(() -> new StreamAcc(), (a, x) -> {
            ((StreamAcc) a).sum += f.applyAsInt((T) x);
            ((StreamAcc) a).n++;
        }, a -> mean((StreamAcc) a, true));
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Double> averagingLong(java.util.function.ToLongFunction<? super T> f) {
        return new Collector<T, Object, Double>(() -> new StreamAcc(), (a, x) -> {
            ((StreamAcc) a).sum += f.applyAsLong((T) x);
            ((StreamAcc) a).n++;
        }, a -> mean((StreamAcc) a, true));
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Double> averagingDouble(java.util.function.ToDoubleFunction<? super T> f) {
        return new Collector<T, Object, Double>(() -> new StreamAcc(), (a, x) -> {
            ((StreamAcc) a).dsum += f.applyAsDouble((T) x);
            ((StreamAcc) a).n++;
        }, a -> mean((StreamAcc) a, false));
    }
    // minBy / maxBy: the first of equal extremes, as an Optional.
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Optional<T>> minBy(Comparator<? super T> c) {
        return reducing((a, b) -> ((Comparator<Object>) c).compare(a, b) <= 0 ? a : b);
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Optional<T>> maxBy(Comparator<? super T> c) {
        return reducing((a, b) -> ((Comparator<Object>) c).compare(a, b) >= 0 ? a : b);
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, Optional<T>> reducing(java.util.function.BinaryOperator<T> op) {
        return new Collector<T, Object, Optional<T>>(() -> new StreamAcc(), (a, x) -> {
            StreamAcc s = (StreamAcc) a;
            s.v = s.n == 0 ? x : op.apply((T) s.v, (T) x);
            s.n++;
        }, a -> ((StreamAcc) a).n == 0 ? Optional.empty() : Optional.of(((StreamAcc) a).v));
    }
    @SuppressWarnings("unchecked")
    public static <T> Collector<T, ?, T> reducing(T identity, java.util.function.BinaryOperator<T> op) {
        return new Collector<T, Object, T>(() -> {
            StreamAcc s = new StreamAcc();
            s.v = identity;
            return s;
        }, (a, x) -> ((StreamAcc) a).v = op.apply((T) ((StreamAcc) a).v, (T) x), a -> ((StreamAcc) a).v);
    }
}
