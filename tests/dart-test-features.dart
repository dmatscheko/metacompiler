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
  Node next;
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
      throw e + "er";              // rethrow a derived value
    }
  } catch (e2) {
    return e2;
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
  check("arith-mod-neg", -7 % 3 == -1);
  check("arith-div", 7 / 2 == 3);
  int ca = 5;
  ca += 3;
  ca -= 2;
  ca *= 4;
  check("compound-arith", ca == 24);
  ca /= 5;
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
  check("bit-ushr", (-1 >>> 28) == 15);

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
  check("str-charat", "abc".charAt(0) == "a" && "abc".charAt(2) == "c");
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
  for (var k in ages) {
    keyOrder = keyOrder + k + ",";  // for-in over a map walks its keys in order
  }
  check("map-for-in-order", keyOrder == "alice,bob,carol,");
  int vSum = 0;
  for (var v in ages.values) {
    vSum = vSum + v;
  }
  check("map-values-iterate", vSum == 96);
  String mapCtl = "";
  for (var k in ages) {
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
  check("class-object-field", head.next.value == 2);
  check("class-null-field", head.next.next == null);
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
    caught = e.code;               // catch binds the thrown object
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
    twoBind = e.code;
  }
  check("catch-two-binders", twoBind == 8);
  String strCaught = "";
  try {
    throw "plain";                 // any value can be thrown
  } catch (e) {
    strCaught = e;
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

  // ----- everything combined in one small pipeline -----
  check("combined-pipeline", transform([1, 2, -3]) == "o1e2x");

  print("features: $checks checks, $fails failures");
  return fails;
}
