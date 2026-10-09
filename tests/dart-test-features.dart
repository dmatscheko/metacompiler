// Fast feature-matrix test for the Dart interpreter (dart-interpreter.abnf) and the
// LLVM-IR compiler (dart-to-llvm-ir.abnf). It replaces the four algorithm-themed
// dart-test-big-* stress tests: instead of large loops (sorting batteries, Ackermann,
// Roman numerals) every implemented construct is exercised with the SMALLEST program
// that can prove it works - loops run 0, 1, 3 or 4 times, recursion stays below
// depth 6. A failed check prints its id (so a diff pinpoints it) and main() returns
// the failure count; exit 0 and byte-identical output on all four legs
// (interpreter/compiler x goja/-frozen) mean everything passed.

int fails = 0;
int checks = 0;

void check(String id, bool cond) {
  checks = checks + 1;
  if (!cond) {
    print("FAIL $id");
    fails = fails + 1;
  }
}

// ----- functions used by the checks below -----
int add(int a, int b) {
  return a + b;
}

int fib(int n) {
  if (n < 2) {
    return n;
  }
  return fib(n - 1) + fib(n - 2);
}

bool isEven(int n) {
  if (n == 0) return true;
  return isOdd(n - 1);
}

bool isOdd(int n) {
  if (n == 0) return false;
  return isEven(n - 1);
}

int sideFx = 0;
bool bump() {
  sideFx = sideFx + 1;
  return true;
}

int earlyMark = 0;
void earlyReturn(int n) {
  if (n > 0) {
    return;                        // early return from a void function
  }
  earlyMark = earlyMark + 1;
}

String grade(int n) {
  if (n > 10) {
    return "big";
  } else if (n > 5) {
    return "mid";
  } else {
    return "small";
  }
}

// named parameters with defaults, plain and mixed with a positional one
int volume({int w = 1, int h = 1, int d = 1}) {
  return w * h * d;
}

int shift(int base, {int by = 10}) {
  return base + by;
}

// ----- classes -----
class Counter {
  int value = 0;                   // field initializer
  int step;
  Counter(this.step);              // field-initialising formal
  void increment() {
    value = value + step;          // bare field names resolve against this
  }
  int current() {
    return value;
  }
  int doubled() => this.current() * 2;   // arrow method calling a method via this
}

class Point {
  int x;
  int y;
  Point(this.x, this.y);
  int manhattan() {
    int ax = x < 0 ? -x : x;
    int ay = y < 0 ? -y : y;
    return ax + ay;
  }
  Point plus(Point other) => Point(x + other.x, y + other.y);
  String render() => "($x,$y)";
}

class Vec {
  int x;
  int y;
  Vec({this.x = 0, this.y = 0});   // named field-initialising formals with defaults
  int norm1() => x + y;
  void bumpBy({int dx = 0, int dy = 0}) {
    x = x + dx;
    y = y + dy;
  }
}

class Node {
  int value;
  Node? next;
  Node(this.value) {
    next = null;                   // constructor with a body
  }
}

class WhoA {
  String who() => "A";
}

class WhoB {
  String who() => "B";
}

String callWho(o) {
  return o.who();                  // dynamic dispatch through an untyped parameter
}

class Boom {
  int code;
  Boom(this.code);
}

// `implements` and `with` contribute a NOMINAL TYPE in Dart: `class A implements B`
// makes `A() is B` true and `on B catch` claim an A, and so does `class A with M`
// for M. Neither is on the __super chain, so a __super walk alone would answer false.
abstract class FShape { double area(); }
mixin FLoud { String shout() => "LOUD"; }
class FSq implements FShape { double area() => 4.0; }
class FSubSq extends FSq {}                 // conformance through an extends chain
class FMix with FLoud implements FShape { double area() => 1.0; }

// ----- typedef names and type PARAMETER names in a type test -----
// Both are names that stand for another type, which the class-chain walk (it knows
// only real classes) cannot resolve: `f is FIntFn` and `case T t` inside
// `class FBox<T>` need dtResolve.
typedef FIntFn = int Function(int);
typedef FNames = List<String>;
typedef FAlias = FNames;
typedef FOldStyle(bool e);
class FBox<T> {
  String w(Object v) { switch (v) { case T t: return "yes"; } return "no"; }
  bool isT(Object v) => v is T;
  bool isNotT(Object v) => v is! T;
  bool isTq(Object? v) => v is T?;
}
class FTwo<K, V> {
  bool k(Object v) => v is K;
  bool val(Object v) => v is V;
}
class FBounded<T extends num> { bool t(Object v) => v is T; }
class FSubBox extends FBox<int> { }
class FRelay<X> extends FBox<X> { }
class FColl<T> { bool t(Object v) => v is T; }
int fDouble(int x) => x * 2;

int risky(int n) {
  if (n > 3) {
    throw Boom(n);
  }
  return n * 2;
}

int classify(int n) {
  try {
    if (n > 0) {
      return n * 10;               // return out of a try body
    }
    throw Boom(0);
  } catch (e) {
    return -1;                     // return out of a catch body
  } finally {}
}

int nestedReturn() {
  try {
    try {
      return 9;                    // propagates through BOTH tries
    } finally {}
  } finally {}
  return 0;
}

int loopBreak() {
  int sum = 0;
  for (int i = 0; i < 9; i = i + 1) {
    try {
      if (i == 3) {
        break;                     // break out of a try body
      }
      sum = sum + i;
    } finally {}
  }
  return sum;
}

int loopContinue() {
  int sum = 0;
  for (int i = 0; i < 5; i = i + 1) {
    try {
      if (i == 2) {
        continue;                  // continue out of a try body
      }
      sum = sum + i;
    } catch (e) {}
  }
  return sum;
}

int returnInFinally() {
  try {
    return 1;
  } finally {
    return 2;                      // the finally's return OVERRIDES the try's
  }
}

int breakInFinally() {
  int i = 0;
  while (i < 9) {
    i = i + 1;
    try {
      bump();
    } finally {
      break;                       // a break out of a finally leaves the loop
    }
  }
  return i;
}

String finallyOverridesThrow() {
  try {
    try {
      throw "boom";
    } finally {
      return "fin";                // cancels the pending rethrow
    }
  } catch (e) {
    return "caught";
  }
}

String rethrower() {
  try {
    try {
      throw "deep";
    } catch (e) {
      throw (e as String) + "er";  // rethrow a derived value
    }
  } catch (e2) {
    return e2 as String;
  }
}

// ----- everything combined in one small pipeline (3-element data flow) -----
String transform(List<int> xs) {
  String out = "";
  for (var n in xs) {
    try {
      if (n < 0) {
        throw Boom(n);
      }
      out = out + (n % 2 == 0 ? "e$n" : "o$n");
    } catch (e) {
      out = out + "x";
    }
  }
  return out;
}

int main() {
  // ----- numbers, arithmetic, precedence (~/ truncates; / is integer here) -----
  check("arith-precedence", 2 + 3 * 4 == 14);
  check("arith-paren", (2 + 3) * 4 == 20);
  check("arith-unary-minus", -3 + 5 == 2);
  check("arith-chain", 20 - 5 - 3 == 12);
  check("arith-truncdiv", 20 ~/ 3 == 6);
  check("arith-truncdiv-neg", -7 ~/ 2 == -3);
  check("arith-mod", 7 % 3 == 1);
  // Dart's % returns a result in [0, b.abs()), so this is 2, not -1.
  // (-1 is what (-7).remainder(3) gives.)
  check("arith-mod-neg", -7 % 3 == 2);
  // num.operator/ ALWAYS returns a double in Dart; ~/ is the truncating one
  // (used correctly two lines above). dart-test-full.dart:72 asserts the same.
  check("arith-div", 7 / 2 == 3.5);
  int ca = 5;
  ca += 3;
  ca -= 2;
  ca *= 4;
  check("compound-arith", ca == 24);
  // ca is an int, and /= yields a double, which is a compile error in Dart.
  // ~/= is the int-preserving form and keeps the intended value.
  ca ~/= 5;
  check("compound-div", ca == 4);
  ca %= 3;
  check("compound-mod", ca == 1);
  int inc = 5;
  inc++;
  check("post-inc", inc == 6);
  int old = inc++;
  check("post-inc-value", old == 6 && inc == 7);
  inc--;
  check("post-dec", inc == 6);

  // ----- bitwise and shifts -----
  check("bit-and-or-xor", (6 & 3) == 2 && (6 | 3) == 7 && (6 ^ 3) == 5);
  check("bit-not", (~5) == -6);
  check("bit-shl", (1 << 4) == 16);
  check("bit-shr-neg", (-8 >> 1) == -4);
  // Dart's int is 64-bit, so `>>>` shifts a 64-bit unsigned value: -1 is
  // 0xFFFFFFFFFFFFFFFF and >>> 28 leaves 0xFFFFFFFFF (15 would be the JS 32-bit
  // answer). Neighbouring 64-bit shift
  // behaviour is pinned by the corpus in
  // language/operator/bit_operations_test.dart:50 (0xF00000000 >> 32 == 15) and :54.
  check("bit-ushr", (-1 >>> 28) == 68719476735);

  // ----- comparisons, logic, ternary -----
  check("cmp-ops", 2 < 3 && 3 <= 3 && 3 > 2 && 3 >= 3 && !(2 >= 3));
  check("cmp-eq-ne", 3 == 3 && 3 != 4);
  check("logic-not", !(true && false) && (false || true));
  var noRun = false && bump();
  var oneRun = true && bump();
  var skipRun = true || bump();
  check("logic-short-circuit", sideFx == 1);
  check("logic-short-values", noRun == false && oneRun == true && skipRun == true);
  check("ternary", (5 > 3 ? "a" : "b") == "a" && (5 < 3 ? "a" : "b") == "b");

  // ----- null and the as cast -----
  var nothing = null;
  check("null-eq", nothing == null);
  check("null-ne", !("x" == null));
  check("as-cast", (7 as int) == 7);

  // ----- strings -----
  check("str-concat", "foo" + "bar" == "foobar");
  check("str-eq", "abc" == "abc" && "a" != "b");
  check("str-length", "hello".length == 5 && "".length == 0);
  check("str-isempty", "".isEmpty && "x".isNotEmpty);
  check("str-charat", "abc"[0] == "a" && "abc"[2] == "c");
  check("str-substring", "hello".substring(1, 3) == "el" && "hello".substring(3) == "lo");
  check("str-indexof", "abcabc".indexOf("b") == 1 && "abc".indexOf("z") == -1);
  String name = "Dart";
  check("str-interp-var", "Hi $name!" == "Hi Dart!");
  int three = 3;
  check("str-interp-expr", "sq=${three * three}" == "sq=9");
  check("str-interp-nested", "q=${"abc".substring(0, 1)}" == "q=a");
  check("str-escapes", "a\tb".length == 3 && "x\ny".length == 3);
  check("str-unicode-len", "héllo".length == 5);
  check("str-single-quotes", 'ab' + 'cd' == "abcd");

  // ----- control flow -----
  check("if-elseif-else", grade(11) + grade(7) + grade(1) == "bigmidsmall");
  int w0 = 0;
  while (w0 > 0) {
    w0 = w0 - 1;                   // runs zero times
  }
  check("while-zero", w0 == 0);
  int w3 = 0;
  while (w3 < 3) {
    w3 = w3 + 1;                   // runs three times
  }
  check("while-three", w3 == 3);
  int forSum = 0;
  for (int i = 1; i <= 3; i++) {
    forSum = forSum + i;
  }
  check("for-basic", forSum == 6);
  String brk = "";
  for (int i = 0; i < 9; i = i + 1) {
    if (i == 2) {
      break;
    }
    brk = brk + "$i";
  }
  check("for-break", brk == "01");
  String cont = "";
  for (int i = 0; i < 4; i++) {
    if (i % 2 == 1) {
      continue;
    }
    cont = cont + "$i";
  }
  check("for-continue", cont == "02");
  String nested = "";
  for (int o = 0; o < 2; o++) {
    for (int i = 0; i < 3; i++) {
      if (i == 1) {
        break;                     // inner break must not end the outer loop
      }
      nested = nested + "$o$i";
    }
  }
  check("nested-break", nested == "0010");
  int feSum = 0;
  for (int v in [1, 2, 3]) {
    feSum = feSum + v;
  }
  check("for-in-list", feSum == 6);
  int feNone = 0;
  for (var v in []) {
    feNone = feNone + 1;           // runs zero times
  }
  check("for-in-empty", feNone == 0);
  String feCtl = "";
  for (var v in [1, 2, 3, 4]) {
    if (v == 2) continue;
    if (v == 4) break;
    feCtl = feCtl + "$v";
  }
  check("for-in-break-continue", feCtl == "13");

  // ----- functions, closures, recursion -----
  check("fn-args", add(20, 22) == 42);
  earlyReturn(5);
  check("fn-early-return", earlyMark == 0);
  check("fn-recursion", fib(6) == 8);
  check("fn-mutual-recursion", isEven(4) && isOdd(5));
  var makeCounter = () {
    int n = 0;
    return () {
      n = n + 1;
      return n;
    };
  };
  var c1 = makeCounter();
  var c2 = makeCounter();
  c1();
  c1();
  check("closure-independent", c1() == 3 && c2() == 1);
  var addBase = (int x) => x + 100;
  check("arrow-lambda", addBase(5) == 105);
  var makeAdder = (int step) => (int m) => step + m;
  var add10 = makeAdder(10);
  check("curried-closure", add10(7) == 17);
  var applyTwice = (f, v) => f(f(v));
  check("fn-higher-order", applyTwice((int n) => n * 2, 3) == 12);
  check("named-defaults", volume() == 1 && volume(w: 2, h: 3) == 6 && volume(d: 4, w: 5) == 20);
  check("named-after-positional", shift(100) == 110 && shift(100, by: 5) == 105);

  // ----- lists -----
  List<int> nums = [10, 20, 30];
  check("list-literal-index", nums.length == 3 && nums[0] == 10 && nums[2] == 30);
  nums[1] = 21;
  check("list-write", nums[1] == 21);
  nums.add(40);
  check("list-add", nums.length == 4 && nums[3] == 40);
  int taken = nums.removeLast();
  check("list-removeLast", taken == 40 && nums.length == 3);
  check("list-contains", nums.contains(30) && !nums.contains(99));
  List<int> empty = [];
  check("list-empty", empty.length == 0 && empty.isEmpty && nums.isNotEmpty);
  empty.add(9);
  check("list-grow-from-empty", empty.length == 1 && empty[0] == 9);
  List<List<int>> grid = [[1, 2], [3]];
  check("list-nested", grid[0][1] == 2 && grid.length == 2);
  List<int> typed = <int>[4, 5];
  check("list-typed-literal", typed[1] == 5);
  List<int> parts = [10, 20];
  List<int> combined = [0, ...parts, if (true) 30, if (false) 99, 40];
  check("list-spread-collection-if", combined.length == 5 && combined[1] == 10 && combined[3] == 30);
  List<int> labels = [if (false) 1 else 2, ...parts];
  check("collection-if-else", labels.length == 3 && labels[0] == 2);
  // dart:core, List.operator +: a NEW list of this list's elements followed by
  // other's, neither operand touched. JavaScript's + would render both sides
  // and answer the string "1,23" - length 4, not 3 - so the length and the
  // rendering are both asserted.
  List<int> plusL = [1, 2] + [3];
  check("list-operator-plus", plusL.length == 3 && plusL[0] == 1 && plusL[2] == 3);
  check("list-operator-plus-text", "$plusL" == "[1, 2, 3]");
  List<int> plusLeft = [1, 2];
  List<int> plusRight = [3];
  List<int> plusBoth = plusLeft + plusRight;
  plusBoth[0] = 9;
  check("list-operator-plus-copies",
        plusBoth.length == 3 && plusBoth[0] == 9 && plusLeft[0] == 1 && plusRight[0] == 3);
  check("list-operator-plus-empty", ([] + [1])[0] == 1 && ([1] + [])[0] == 1);

  // ----- maps (insertion-ordered) -----
  Map<String, int> ages = {"alice": 30, "bob": 25};
  check("map-get", ages["alice"] == 30);
  ages["carol"] = 40;
  check("map-set-new", ages["carol"] == 40 && ages.length == 3);
  ages["bob"] = 26;
  check("map-overwrite", ages["bob"] == 26 && ages.length == 3);
  check("map-containskey", ages.containsKey("bob") && !ages.containsKey("dave"));
  check("map-keys-values", ages.keys.length == 3 && ages.values.length == 3);
  String keyOrder = "";
  for (var k in ages.keys) {
    keyOrder = keyOrder + k + ",";  // a map's keys iterate in insertion order
  }
  check("map-for-in-order", keyOrder == "alice,bob,carol,");
  int vSum = 0;
  for (var v in ages.values) {
    vSum = vSum + v;
  }
  check("map-values-iterate", vSum == 96);
  String mapCtl = "";
  for (var k in ages.keys) {
    if (k == "alice") continue;
    if (k == "carol") break;
    mapCtl = mapCtl + k;
  }
  check("map-for-in-break-continue", mapCtl == "bob");
  Map<String, int> typedMap = <String, int>{"a": 1};
  check("map-typed-literal", typedMap["a"] == 1);

  // ----- classes -----
  Counter c = Counter(10);
  c.increment();
  c.increment();
  check("class-field-init-and-method", c.current() == 20 && c.value == 20);
  check("class-arrow-this", c.doubled() == 40);
  Point p = new Point(3, -4);      // construction with new
  Point q = Point(1, 6);           // and without
  check("class-ctor-formals", p.x == 3 && p.y == -4);
  check("class-method", p.manhattan() == 7);
  check("class-object-arg", p.plus(q).x == 4);
  check("class-chaining", p.plus(Point(2, 2)).manhattan() == 7);
  check("class-interp-method", q.render() == "(1,6)");
  check("class-dispatch", callWho(WhoA()) + callWho(WhoB()) == "AB");
  Node head = Node(1);
  head.next = Node(2);             // an object-typed field forms a chain
  check("class-object-field", head.next!.value == 2);
  check("class-null-field", head.next!.next == null);
  Vec v0 = Vec();
  Vec v1 = Vec(x: 1, y: 2);
  check("class-named-ctor-defaults", v0.norm1() == 0 && v1.norm1() == 3);
  Vec v2 = Vec()..x = 3..y = 4..bumpBy(dx: 1, dy: 1);
  check("cascade", v2.x == 4 && v2.y == 5 && v2.norm1() == 9);
  List<int> built = [1, 2]..add(3)..add(4);
  check("cascade-on-list", built.length == 4 && built[3] == 4);

  // ----- exceptions -----
  String exLog = "";
  try {
    exLog = exLog + "t";
    throw Boom(1);
  } catch (e) {
    exLog = exLog + "c";
  } finally {
    exLog = exLog + "f";
  }
  check("try-throw-catch-finally", exLog == "tcf");
  String noThrow = "";
  try {
    noThrow = noThrow + "t";
  } catch (e) {
    noThrow = noThrow + "!";
  } finally {
    noThrow = noThrow + "f";
  }
  check("try-no-throw", noThrow == "tf");
  int caught = -1;
  try {
    risky(5);
    check("throw-unreachable", false);
  } catch (e) {
    caught = (e as Boom).code;     // catch binds the thrown object
  }
  check("catch-binding", caught == 5);
  check("throw-not-taken", risky(2) == 4);
  int onCaught = -1;
  try {
    throw Boom(7);
  } on Boom catch (e) {            // typed on-catch still binds
    onCaught = e.code;
  }
  check("on-type-catch", onCaught == 7);
  int twoBind = -1;
  try {
    throw Boom(8);
  } catch (e, st) {                // two-binder catch: the stack trace is dropped
    twoBind = (e as Boom).code;
  }
  check("catch-two-binders", twoBind == 8);
  String strCaught = "";
  try {
    throw "plain";                 // any value can be thrown
  } catch (e) {
    strCaught = e as String;
  }
  check("throw-string", strCaught == "plain");
  check("return-from-try", classify(4) == 40);
  check("return-from-catch", classify(-1) == -1);
  check("return-nested-tries", nestedReturn() == 9);
  check("break-out-of-try", loopBreak() == 3);
  check("continue-out-of-try", loopContinue() == 8);
  check("return-in-finally-overrides", returnInFinally() == 2);
  check("break-in-finally", breakInFinally() == 1);
  check("finally-overrides-throw", finallyOverridesThrow() == "fin");
  check("rethrow-derived", rethrower() == "deeper");

  // ----- dart:core's errors and exceptions -----
  // The sixteen engine-provided names. The full battery - every constructor,
  // every toString, the whole supertype graph - is SECTION 31 of
  // tests/dart-test-full.dart, cited line by line to sdk/lib/core/errors.dart and
  // sdk/lib/core/exceptions.dart. These are the shapes the matrix should see.
  check("exc-bound-format", FormatException("bad").toString() == "FormatException: bad");
  check("exc-bound-state", StateError("empty").toString() == "Bad state: empty");
  check("exc-bound-named-ctor",
        ArgumentError.value(7, "p").toString() == "Invalid argument (p): 7");
  // Error and Exception are DISJOINT roots, and `on ... catch` has to say so.
  check("exc-roots-disjoint",
        FormatException("x") is Exception && !(FormatException("x") is Error)
        && StateError("x") is Error && !(StateError("x") is Exception));
  var excKind = "";
  try { throw FormatException("x"); } on Error catch (e) { excKind = "error"; }
  on Exception catch (e) { excKind = "exception"; }
  check("exc-on-exception-claims-format", excKind == "exception");
  // `on int catch` and friends: Dart's own type names (the shared js_is_type knows
  // only the JVM spellings).
  var primKind = "";
  try { throw 7; } on int catch (e) { primKind = "int"; } catch (e) { primKind = "other"; }
  check("exc-on-int", primKind == "int");
  var mapKind = "";
  try { throw {"a": 1}; } on Map catch (e) { mapKind = "map"; } catch (e) { mapKind = "other"; }
  check("exc-on-map", mapKind == "map");
  // An int divisor of zero is a CATCHABLE IntegerDivisionByZeroException
  // (sdk/lib/_internal/vm/lib/integers.dart:25-44).
  var zeros = [0];
  var divKind = "";
  try { divKind = "${5 ~/ zeros[0]}"; }
  on IntegerDivisionByZeroException catch (e) { divKind = "$e"; }
  check("exc-int-div-by-zero", divKind == "IntegerDivisionByZeroException");
  // A failing assert raises a catchable AssertionError in both halves.
  var assertKind = "";
  try { assert(zeros[0] == 1, "nope"); } on AssertionError catch (e) { assertKind = "$e"; }
  check("exc-assert-catchable", assertKind == "Assertion failed: \"nope\"");

  // ----- implements / with are nominal types -----
  check("iface-direct", FSq() is FShape);
  check("iface-self", FSq() is FSq);
  check("iface-through-extends", FSubSq() is FSq && FSubSq() is FShape);
  check("iface-mixin", FMix() is FLoud && FMix().shout() == "LOUD");
  check("iface-mixin-and-implements", FMix() is FShape);
  check("iface-negative", !(FSq() is FLoud) && !(7 is FShape) && !("x" is FShape));
  var ifaceKind = "";
  try { throw FSq(); } on FShape catch (e) { ifaceKind = "shape"; }
  catch (e) { ifaceKind = "other"; }
  check("iface-on-catch", ifaceKind == "shape");
  var ifaceKind2 = "";
  try { throw FSq(); } on FLoud catch (e) { ifaceKind2 = "loud"; }
  catch (e) { ifaceKind2 = "other"; }
  check("iface-on-catch-negative", ifaceKind2 == "other");

  // ----- a list index out of range is dart's IndexError -----
  // Not python's (the shared js_pyget's). dart:core's List.operator [] throws a
  // RangeError, and the object
  // is an IndexError (errors.dart:445); the text is dxArgStr/dxExpl's.
  var idxs = [10, 20, 30];
  var idxMsg = "";
  try { idxMsg = "${idxs[7]}"; } on IndexError catch (e) { idxMsg = "$e"; }
  check("idx-range-error",
        idxMsg == "RangeError (index): Index out of range: index should be less than 3: 7");
  var idxNeg = "";
  try { idxNeg = "${idxs[0 - 1]}"; } on RangeError catch (e) { idxNeg = "$e"; }
  check("idx-negative-no-wrap",
        idxNeg == "RangeError (index): Index out of range: index must not be negative: -1");
  var idxSet = "";
  try { idxs[5] = 1; } on IndexError catch (e) { idxSet = "set"; }
  check("idx-set-range-error", idxSet == "set" && idxs[2] == 30);
  var idxStr = "";
  try { idxStr = "abc"[9]; } on IndexError catch (e) { idxStr = "str"; }
  check("idx-string-range-error", idxStr == "str");
  // IndexError's place in the hierarchy, and the fields dart:core documents on it.
  var idxObj = "";
  try { idxs[9]; } catch (e) {
    idxObj = "${e is IndexError}${e is RangeError}${e is ArgumentError}"
             "${e is Error}${e is Exception}";
    var r = e as IndexError;
    idxObj = "$idxObj|${r.invalidValue}|${r.start}|${r.end}|${r.name}|${r.message}";
  }
  check("idx-error-hierarchy",
        idxObj == "truetruetruetruefalse|9|0|2|index|Index out of range");
  // A Map miss is still NULL, not an error - the guard around it must survive.
  var idxMap = {"a": 1};
  check("idx-map-miss-still-null", idxMap["zz"] == null && idxMap["a"] == 1);

  // ----- a TYPED pattern TESTS its type -----
  // The type annotation on a variable pattern is a type TEST, so a typed switch does
  // not simply take its FIRST case. The position
  // decides what a failed test means: a switch case / if-case is REFUTABLE and falls
  // to the next case, a declaration or a for-in binder is IRREFUTABLE and raises
  // Dart's TypeError (patterns feature spec, "Variable"; errors.dart:144).
  Object tpSub = FSubSq();
  check("tpat-switch-expr-picks-the-subtype",
        switch (tpSub) { FSubSq _ => "sub", FSq _ => "sq", _ => "no" } == "sub");
  Object tpSq = FSq();
  check("tpat-switch-expr-skips-a-non-match",
        switch (tpSq) { FSubSq _ => "sub", FSq _ => "sq", _ => "no" } == "sq");
  check("tpat-switch-expr-falls-through-to-wildcard",
        switch (7) { FSubSq _ => "sub", FSq _ => "sq", _ => "no" } == "no");
  // The named form binds only in the case whose type matched.
  check("tpat-named-binder",
        switch (tpSq) { FSubSq s => "sub", FSq s => "sq${s.area()}", _ => "no" } == "sq4.0");
  // dart:core's own names, and int/double being DISJOINT.
  check("tpat-core-types",
        switch (3) { String s => "s", double d => "d", int i => "i$i", _ => "no" } == "i3");
  check("tpat-core-double",
        switch (2.5) { int i => "i", double d => "d$d", _ => "no" } == "d2.5");
  check("tpat-core-null-is-not-int",
        switch (null) { int i => "i", Null n => "nul", _ => "no" } == "nul");
  check("tpat-nullable-type-matches-null",
        switch (null) { int? i => "opt", _ => "no" } == "opt");
  // A switch STATEMENT and an if-case take the same route.
  var tpStmt = "";
  switch (tpSq) { case FSubSq _: tpStmt = "sub"; case FSq _: tpStmt = "sq"; }
  check("tpat-switch-statement", tpStmt == "sq");
  var tpIf = "no";
  if (tpSq case FSubSq q) { tpIf = "sub"; }
  check("tpat-if-case-refuses", tpIf == "no");
  if (tpSq case FSq q) { tpIf = "sq${q.area()}"; }
  check("tpat-if-case-accepts", tpIf == "sq4.0");
  // Nested: a record field, a list element, a map value, an object-pattern field.
  check("tpat-record-field",
        switch ((1, "s")) { (String a, int b) => "1", (int a, String b) => "2", _ => "3" } == "2");
  check("tpat-list-element",
        switch ([1, 2]) { [String a, String b] => "ss", [int a, int b] => "ii", _ => "no" } == "ii");
  check("tpat-map-value",
        switch ({"k": 1}) { {"k": String v} => "s", {"k": int v} => "i$v", _ => "no" } == "i1");
  // An IRREFUTABLE position: the declaration downcasts, and a failure is a TypeError.
  // The bad value has to reach the pattern as `dynamic`: a statically mistyped value in
  // an irrefutable position is a compile-time error, while `dynamic` inserts a cast.
  var (int tpOk, String tpOk2) = (4, "z");
  check("tpat-declaration-binds", tpOk == 4 && tpOk2 == "z");
  var tpDeclErr = "none";
  try {
    dynamic tpBadSrc = (1, "s");
    var (int tpBad, int tpBad2) = tpBadSrc;
    tpDeclErr = "ran$tpBad$tpBad2";
  } on TypeError catch (e) { tpDeclErr = "typeerror"; }
  check("tpat-declaration-raises-typeerror", tpDeclErr == "typeerror");
  // A for-in binder is a declaration too. Assert the TRIP COUNT: skipping the element
  // instead of raising would leave a short loop that still exits 0.
  var tpTrips = 0;
  for (var (int n, String s) in [(1, "a"), (2, "b"), (3, "c")]) { tpTrips = tpTrips + n; }
  check("tpat-for-in-runs-every-trip", tpTrips == 6);
  var tpLoopErr = "none";
  var tpShort = 0;
  try {
    List<dynamic> tpMixed = [(1, "a"), (2, 9), (3, "c")];
    for (var (int n, String s) in tpMixed) { tpShort = tpShort + 1; }
    tpLoopErr = "ran";
  } on TypeError catch (e) { tpLoopErr = "typeerror"; }
  check("tpat-for-in-raises-not-skips", tpLoopErr == "typeerror" && tpShort == 1);
  // An object pattern's own type sees the `implements` edge the shared js_is_type
  // cannot: FSq implements FShape and is not on any __super chain.
  check("tpat-object-pattern-implements",
        switch (tpSq) { FShape() => "iface", _ => "no" } == "iface");
  check("tpat-object-pattern-negative",
        switch (7) { FShape() => "iface", _ => "no" } == "no");

  // ----- `o is T ? a : b` parses -----
  // The type's trailing "?" is greedy and a PEG never gives it back, so without the
  // NotCondQ guard the whole program would fail to parse at the ":".
  check("iscond-plain", (tpSq is FShape ? "y" : "n") == "y");
  check("iscond-negated", (tpSq is! FShape ? "y" : "n") == "n");
  check("iscond-nullable-type-still-parses", (tpSq is FShape? ? "y" : "n") == "y");
  int? iscondN;
  check("iscond-nullable-plain", iscondN is int? && !(iscondN is int));
  check("iscond-nested", (tpSq is FShape ? (tpSub is FSq ? "a" : "b") : "c") == "a");
  check("iscond-unary-arms", (tpSq is FShape ? -1 : -2) == -1);
  // `is List<T>` must be a boolean, not the NUMBER 1 a BITWISE js_band leaves (the
  // element-type refinement). Interpolate it, so the value is compared and not just
  // tested - in an `if` a raw 1 is truthy.
  check("is-typed-list-answers-a-bool", "${<int>[1, 2] is List<int>}" == "true");
  check("is-typed-list-negative", "${<int>[1, 2] is List<String>}" == "false");

  // ----- typedef / type-parameter names in a type test -----
  // A typedef DENOTES the aliased type, so the test is the test of that type. A type
  // parameter is REIFIED in Dart: `case T t` inside `class FBox<T>` tests the argument
  // the instance was created with, so FBox<int> claims 3 and refuses "s" - erasing T to
  // dynamic would have made both answer yes.
  check("tdef-function-alias", (fDouble is FIntFn));
  check("tdef-function-alias-negative", !(3 is FIntFn));
  check("tdef-legacy-form", (fDouble is FOldStyle));
  check("tdef-collection-alias", <String>["a"] is FNames);
  check("tdef-collection-alias-element-type", !(<int>[1] is FNames));
  check("tdef-alias-of-alias", <String>["a"] is FAlias);
  check("tparam-pattern-matches", FBox<int>().w(3) == "yes");
  check("tparam-pattern-refuses-other-argument", FBox<int>().w("s") == "no");
  check("tparam-pattern-second-instantiation", FBox<String>().w("s") == "yes");
  check("tparam-is", FBox<int>().isT(3) && !FBox<int>().isT("s"));
  check("tparam-is-not", FBox<int>().isNotT("s"));
  check("tparam-nullable", FBox<int>().isTq(null));
  check("tparam-two-parameters-are-positional",
        FTwo<int, String>().k(1) && !FTwo<int, String>().k("a") && FTwo<int, String>().val("a"));
  check("tparam-bounded", FBounded<double>().t(1.5) && !FBounded<double>().t(1));
  check("tparam-generic-argument", FColl<List<int>>().t(<int>[1]));
  check("tparam-generic-argument-element-type", !FColl<List<int>>().t(<String>["a"]));
  // The `extends` clause fixes the superclass's parameter for every instance, and a
  // subclass that forwards its own parameter carries the VALUE, not the letter.
  check("tparam-from-extends-clause", FSubBox().isT(3) && !FSubBox().isT("s"));
  check("tparam-forwarded-through-subclass", FRelay<int>().isT(3) && !FRelay<int>().isT("s"));
  // Nothing supplies an argument: Dart infers FBox<dynamic> here, and so does this.
  check("tparam-uninstantiated-is-dynamic", FBox().isT("anything"));
  // A boolean, not the raw 1 a bitwise js_band would leave (see is-typed-list above).
  check("tparam-answers-a-bool", "${FBox<int>().isT(3)}" == "true");

  // ----- a CHAIN of type-test postfixes parses -----
  // TypeTestExpr takes a repetition, so `o as FShape is FShape` parses.
  Object tchO = tpSq;
  check("typetest-chain-as-then-is", (tchO as FShape is FShape));
  check("typetest-chain-as-as-isnot", (tchO as FShape as FShape is! int));
  check("typetest-chain-still-binds-below-equality", (1 is int == true));

  // ----- everything combined in one small pipeline -----
  check("combined-pipeline", transform([1, 2, -3]) == "o1e2x");

  print("features: $checks checks, $fails failures");
  return fails;
}
