// Dart self-checking test 3/4 for the metacompiler: OO + FUNCTIONAL SIGNATURE FEATURES.
//
// Value classes with methods that return fresh instances (a reduced Fraction, a small Vector),
// fluent method chaining (methods returning this), cascades that configure a freshly built
// object, closures with real captured state (counters, accumulators, currying, function
// composition, a memoizer that proves it only invokes the underlying function once per
// distinct argument), higher-order list utilities implemented by hand (map / filter / reduce /
// any / forEach take closures), and named parameters with defaults. int main() returns the
// number of failures, so the exit code is 0 on success and goja and frozen must agree.

int gcd(int a, int b) {
  if (a < 0) a = -a;
  if (b < 0) b = -b;
  while (b != 0) {
    int t = a % b;
    a = b;
    b = t;
  }
  return a;
}

// A reduced rational number. The constructor normalises the sign and divides out the gcd,
// so structurally-equal fractions have identical fields.
class Fraction {
  int numer;
  int denom;
  // Plain params normalised into the fields via this. (a field-initialising formal would
  // bind a shadowing local, so reassigning the bare name would not update the field).
  Fraction(int n, int d) {
    if (d < 0) {
      n = -n;
      d = -d;
    }
    int g = gcd(n, d);
    if (g > 1) {
      n = n ~/ g;
      d = d ~/ g;
    }
    this.numer = n;
    this.denom = d;
  }
  Fraction add(Fraction o) => Fraction(numer * o.denom + o.numer * denom, denom * o.denom);
  Fraction mul(Fraction o) => Fraction(numer * o.numer, denom * o.denom);
  bool equals(Fraction o) => numer == o.numer && denom == o.denom;
  bool isWhole() => denom == 1;
  String render() => '$numer/$denom';
}

// A small 3D integer vector, configured with cascades and combined with methods.
class Vec3 {
  int x;
  int y;
  int z;
  Vec3(this.x, this.y, this.z);
  Vec3 plus(Vec3 o) => Vec3(x + o.x, y + o.y, z + o.z);
  Vec3 scale(int k) => Vec3(x * k, y * k, z * k);
  int dot(Vec3 o) => x * o.x + y * o.y + z * o.z;
  int manhattan() => (x < 0 ? -x : x) + (y < 0 ? -y : y) + (z < 0 ? -z : z);
}

// A fluent builder: every mutator returns this so calls can be chained.
class Accumulator {
  List<int> items;
  Accumulator() {
    items = [];
  }
  Accumulator addItem(int x) {
    items.add(x);
    return this;
  }
  Accumulator addAll(List<int> xs) {
    for (int i = 0; i < xs.length; i = i + 1) {
      items.add(xs[i]);
    }
    return this;
  }
  int total() {
    int s = 0;
    for (int i = 0; i < items.length; i = i + 1) {
      s = s + items[i];
    }
    return s;
  }
  int size() => items.length;
}

// A rectangle configured entirely through named constructor parameters with defaults.
class Rect {
  int w;
  int h;
  Rect({this.w = 1, this.h = 1});
  int area() => w * h;
  int perimeter() => 2 * (w + h);
}

// ---------- higher-order list utilities (take closures) ----------

List<int> mapList(List<int> xs, f) {
  List<int> out = [];
  for (int i = 0; i < xs.length; i = i + 1) {
    out.add(f(xs[i]));
  }
  return out;
}

List<int> filterList(List<int> xs, pred) {
  List<int> out = [];
  for (int i = 0; i < xs.length; i = i + 1) {
    if (pred(xs[i])) out.add(xs[i]);
  }
  return out;
}

int reduceList(List<int> xs, int seed, f) {
  int acc = seed;
  for (int i = 0; i < xs.length; i = i + 1) {
    acc = f(acc, xs[i]);
  }
  return acc;
}

bool anyMatch(List<int> xs, pred) {
  for (int i = 0; i < xs.length; i = i + 1) {
    if (pred(xs[i])) return true;
  }
  return false;
}

void forEachList(List<int> xs, action) {
  for (int i = 0; i < xs.length; i = i + 1) {
    action(xs[i]);
  }
}

bool listEq(List<int> a, List<int> b) {
  if (a.length != b.length) return false;
  for (int i = 0; i < a.length; i = i + 1) {
    if (a[i] != b[i]) return false;
  }
  return true;
}

// ---------- top-level closures / combinators ----------

var makeAdder = (int n) => (int x) => x + n;
var compose = (f, g) => (x) => f(g(x));

// A counter factory: each call returns an independent closure over its own state.
var makeCounter = () {
  int n = 0;
  return () {
    n = n + 1;
    return n;
  };
};

// A memoizer: wraps f so it is only invoked once per distinct argument.
var memoize = (f) {
  Map<int, int> cache = {};
  return (n) {
    if (cache.containsKey(n)) return cache[n];
    int v = f(n);
    cache[n] = v;
    return v;
  };
};

int main() {
  int failures = 0;

  // ---------- Fraction: reduction, arithmetic, structural equality ----------
  Fraction half = Fraction(1, 2);
  Fraction third = Fraction(1, 3);
  if (half.render() != '1/2') failures = failures + 1;
  // 2/4 reduces to 1/2
  if (!Fraction(2, 4).equals(half)) failures = failures + 1;
  // negative denominator moves the sign to the numerator
  if (Fraction(1, -2).render() != '-1/2') failures = failures + 1;
  // 1/2 + 1/3 = 5/6
  if (!half.add(third).equals(Fraction(5, 6))) failures = failures + 1;
  // 1/2 * 2/3 = 1/3
  if (!Fraction(1, 2).mul(Fraction(2, 3)).equals(third)) failures = failures + 1;
  // 2/2 reduces to a whole number 1/1
  if (!Fraction(2, 2).isWhole()) failures = failures + 1;
  if (Fraction(6, 4).render() != '3/2') failures = failures + 1;
  // (1/2 + 1/3) + 1/6 == 1  (associativity to a whole)
  Fraction sixth = Fraction(1, 6);
  if (!half.add(third).add(sixth).equals(Fraction(1, 1))) failures = failures + 1;

  // ---------- Vec3: methods and cascades ----------
  Vec3 a = Vec3(1, 2, 3);
  Vec3 b = Vec3(4, 5, 6);
  if (a.dot(b) != 32) failures = failures + 1; // 4 + 10 + 18
  Vec3 sum = a.plus(b);
  if (sum.x != 5 || sum.y != 7 || sum.z != 9) failures = failures + 1;
  Vec3 scaled = a.scale(3);
  if (scaled.x != 3 || scaled.y != 6 || scaled.z != 9) failures = failures + 1;
  // configure a vector with a cascade, then read it back
  Vec3 v = Vec3(0, 0, 0)..x = 3..y = -4..z = 12;
  if (v.x != 3 || v.y != -4 || v.z != 12) failures = failures + 1;
  if (v.manhattan() != 19) failures = failures + 1;
  if (v.dot(v) != 169) failures = failures + 1; // 9 + 16 + 144

  // ---------- fluent builder (method chaining) ----------
  int chained = Accumulator().addItem(1).addItem(2).addItem(3).addItem(4).total();
  if (chained != 10) failures = failures + 1;
  Accumulator acc = Accumulator()..addItem(5)..addAll([10, 20])..addItem(1);
  if (acc.total() != 36) failures = failures + 1;
  if (acc.size() != 4) failures = failures + 1;

  // ---------- Rect via named parameters with defaults ----------
  if (Rect().area() != 1) failures = failures + 1;
  if (Rect(w: 3, h: 4).area() != 12) failures = failures + 1;
  if (Rect(w: 5).area() != 5) failures = failures + 1;     // h defaults to 1
  if (Rect(h: 6).perimeter() != 14) failures = failures + 1; // w defaults to 1

  // ---------- higher-order list utilities ----------
  List<int> nums = [1, 2, 3, 4, 5, 6];
  List<int> doubled = mapList(nums, (x) => x * 2);
  if (!listEq(doubled, [2, 4, 6, 8, 10, 12])) failures = failures + 1;
  List<int> evens = filterList(nums, (x) => x % 2 == 0);
  if (!listEq(evens, [2, 4, 6])) failures = failures + 1;
  int sumAll = reduceList(nums, 0, (a, b) => a + b);
  if (sumAll != 21) failures = failures + 1;
  int product = reduceList([1, 2, 3, 4], 1, (a, b) => a * b);
  if (product != 24) failures = failures + 1;
  if (!anyMatch(nums, (x) => x > 5)) failures = failures + 1;
  if (anyMatch(nums, (x) => x > 100)) failures = failures + 1;
  // sum of squares of the even numbers, composing three utilities
  int sumSqEven = reduceList(mapList(filterList(nums, (x) => x % 2 == 0), (x) => x * x), 0, (a, b) => a + b);
  if (sumSqEven != 56) failures = failures + 1; // 4 + 16 + 36

  // forEach mutating an external accumulator captured by the action closure
  int running = 0;
  forEachList([10, 20, 30], (x) {
    running = running + x;
  });
  if (running != 60) failures = failures + 1;

  // ---------- currying and composition ----------
  var add10 = makeAdder(10);
  var add100 = makeAdder(100);
  if (add10(5) != 15) failures = failures + 1;
  if (add100(5) != 105) failures = failures + 1;
  var incThenTriple = compose((x) => x * 3, (x) => x + 1);
  if (incThenTriple(4) != 15) failures = failures + 1; // (4+1)*3
  var tripleThenInc = compose((x) => x + 1, (x) => x * 3);
  if (tripleThenInc(4) != 13) failures = failures + 1; // (4*3)+1

  // ---------- counters with independent captured state ----------
  var c1 = makeCounter();
  var c2 = makeCounter();
  if (c1() != 1) failures = failures + 1;
  if (c1() != 2) failures = failures + 1;
  if (c2() != 1) failures = failures + 1; // independent
  if (c1() != 3) failures = failures + 1;
  if (c2() != 2) failures = failures + 1;

  // ---------- memoization: underlying fn called once per distinct argument ----------
  int calls = 0;
  var square = (x) {
    calls = calls + 1;
    return x * x;
  };
  var fastSquare = memoize(square);
  if (fastSquare(6) != 36) failures = failures + 1;
  if (fastSquare(6) != 36) failures = failures + 1;
  if (fastSquare(6) != 36) failures = failures + 1;
  if (calls != 1) failures = failures + 1; // three calls, one real evaluation
  if (fastSquare(7) != 49) failures = failures + 1;
  if (calls != 2) failures = failures + 1; // a new argument evaluates once more
  if (fastSquare(7) != 49) failures = failures + 1;
  if (calls != 2) failures = failures + 1;

  print('dart-test-big-3 (OO + functional) finished with $failures failures');
  return failures;
}
