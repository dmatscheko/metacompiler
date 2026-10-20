// java-function.java -- java.util.function's generic functional interfaces
// with their default and static methods, written in Java and compiled with the
// program when it imports one of them (jFuncWants in lib/java-decltypes.js).
// One implementation serves all three engines.
//
// A lambda stored in a local of one of these types becomes an instance of it
// (the grammars' SAM conversion), and that is what lets f.andThen(g) reach the
// default method; every combinator below builds its result in such a local for
// the same reason, so a chain keeps working. The primitive specializations
// (IntUnaryOperator, ...) are not declared here: lib/java-decltypes.js
// converts their arguments and results instead.
package java.util.function;

interface Function<T, R> {
    R apply(T t);

    default <V> Function<V, R> compose(Function<? super V, ? extends T> before) {
        Function<T, R> self = this;
        Function<V, R> f = (V v) -> self.apply(before.apply(v));
        return f;
    }
    default <V> Function<T, V> andThen(Function<? super R, ? extends V> after) {
        Function<T, R> self = this;
        Function<T, V> f = (T t) -> after.apply(self.apply(t));
        return f;
    }
    static <T> Function<T, T> identity() {
        Function<T, T> f = t -> t;
        return f;
    }
}

interface BiFunction<T, U, R> {
    R apply(T t, U u);

    default <V> BiFunction<T, U, V> andThen(Function<? super R, ? extends V> after) {
        BiFunction<T, U, R> self = this;
        BiFunction<T, U, V> f = (T t, U u) -> after.apply(self.apply(t, u));
        return f;
    }
}

interface UnaryOperator<T> extends Function<T, T> {
    static <T> UnaryOperator<T> identity() {
        UnaryOperator<T> f = t -> t;
        return f;
    }
}

interface BinaryOperator<T> extends BiFunction<T, T, T> {
    static <T> BinaryOperator<T> minBy(java.util.Comparator<? super T> cmp) {
        BinaryOperator<T> f = (a, b) -> cmp.compare(a, b) <= 0 ? a : b;
        return f;
    }
    static <T> BinaryOperator<T> maxBy(java.util.Comparator<? super T> cmp) {
        BinaryOperator<T> f = (a, b) -> cmp.compare(a, b) >= 0 ? a : b;
        return f;
    }
}

interface Predicate<T> {
    boolean test(T t);

    default Predicate<T> and(Predicate<? super T> other) {
        Predicate<T> self = this;
        Predicate<T> p = (T t) -> self.test(t) && other.test(t);
        return p;
    }
    default Predicate<T> or(Predicate<? super T> other) {
        Predicate<T> self = this;
        Predicate<T> p = (T t) -> self.test(t) || other.test(t);
        return p;
    }
    default Predicate<T> negate() {
        Predicate<T> self = this;
        Predicate<T> p = (T t) -> !self.test(t);
        return p;
    }
    static <T> Predicate<T> isEqual(Object target) {
        Predicate<T> p = t -> target == null ? t == null : target.equals(t);
        return p;
    }
    @SuppressWarnings("unchecked")
    static <T> Predicate<T> not(Predicate<? super T> target) {
        Predicate<T> p = t -> !target.test(t);
        return p;
    }
}

interface BiPredicate<T, U> {
    boolean test(T t, U u);

    default BiPredicate<T, U> and(BiPredicate<? super T, ? super U> other) {
        BiPredicate<T, U> self = this;
        BiPredicate<T, U> p = (T t, U u) -> self.test(t, u) && other.test(t, u);
        return p;
    }
    default BiPredicate<T, U> or(BiPredicate<? super T, ? super U> other) {
        BiPredicate<T, U> self = this;
        BiPredicate<T, U> p = (T t, U u) -> self.test(t, u) || other.test(t, u);
        return p;
    }
    default BiPredicate<T, U> negate() {
        BiPredicate<T, U> self = this;
        BiPredicate<T, U> p = (T t, U u) -> !self.test(t, u);
        return p;
    }
}

interface Consumer<T> {
    void accept(T t);

    default Consumer<T> andThen(Consumer<? super T> after) {
        Consumer<T> self = this;
        Consumer<T> c = (T t) -> {
            self.accept(t);
            after.accept(t);
        };
        return c;
    }
}

interface BiConsumer<T, U> {
    void accept(T t, U u);

    default BiConsumer<T, U> andThen(BiConsumer<? super T, ? super U> after) {
        BiConsumer<T, U> self = this;
        BiConsumer<T, U> c = (T t, U u) -> {
            self.accept(t, u);
            after.accept(t, u);
        };
        return c;
    }
}

interface Supplier<T> {
    T get();
}
