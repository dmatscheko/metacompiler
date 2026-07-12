// Full-syntax test: Dart (Dart 3 core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the dart grammars
// are. It walks the whole practical Dart 3 syntax, one self-contained SECTION
// per language area. The --full runner runs the file, and whenever a grammar
// aborts it removes the section around the error and retries - so the report
// lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count (exit 0 == full support, verified)
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// import/export/library/part directives (single-file harness, so dart:core API
// stays at what the feature matrix already uses, plus Object/Function/identical),
// isolates, FFI, metadata annotations (including @override), doc comments.
// Async and generator SYNTAX is covered: sync* generators actually run, async
// bodies are only defined and type-checked, never awaited (no event loop).
// Targets Dart >= 3.6 (records, patterns, class modifiers, extension types,
// digit separators). Validated against the language spec by hand; no local
// dart toolchain exists on this machine.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the Dart language specification and tour (Dart 3)
// with the ANTLR grammars-v4 dart2 grammar as a coverage checklist.

int fails = 0;
int checks = 0;

void check(String id, bool cond) {
  checks = checks + 1;
  if (!cond) {
    print("FAIL $id");
    fails = fails + 1;
  }
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
class BlBox {
  int v = 0;
  void bump(int d) { v = v + d; }
}
void s01() {
  int n = 0;
  for (int i = 0; i < 3; i++) { n = n + i; }
  check("bl1", n == 3 && 20 ~/ 3 == 6 && 7 % 3 == 1);
  var b = BlBox();
  b.bump(2);
  b.bump(3);
  check("bl2", b.v == 5);
  var m = {"a": 1};
  m["b"] = 2;
  check("bl3", m["b"] == 2 && m.containsKey("a"));
  String log = "";
  try { throw "boom"; } catch (e) { log = "c:$e"; } finally { log = log + "!"; }
  check("bl4", log == "c:boom!");
}

// ===== SECTION 02: numeric and string literals =====
void s02() {
  check("li1", 0xFF == 255 && 0x7f == 127);
  check("li2", 1e3 == 1000 && 2.5e-2 == 0.025 && 12.5 * 2 == 25);
  check("li3", 1_000_000 == 1000000 && 0xF_F == 255); // digit separators (3.6)
  check("li4", 7 / 2 == 3.5 && 7 ~/ 2 == 3); // / is double division
  check("li5", 'sin' 'gle' == "single" && "dou" 'ble' == "double"); // adjacency
  var t = '''ab
cd''';
  check("li6", t.length == 5 && """x'y""" == "x'y"); // triple-quoted strings
  check("li7", r'a\nb'.length == 4 && r'$x' == '\$x'); // raw string, $ escape
  check("li8", 'A' == "A" && '\x42' == "B" && '\u{1F600}'.length == 2);
  var n = 6;
  check("li9", 'v=$n' == "v=6" && '${n + 1}!' == "7!");
}

// ===== SECTION 03: collection literals =====
List<int>? coPick(bool b) => b ? [9] : null;
void s03() {
  var st = {3, 1, 2}; // set literal
  check("co1", st.length == 3 && st.contains(2) && !st.contains(9));
  check("co2", <int>{}.isEmpty && {}.isEmpty); // typed empty set; bare {} is a map
  var seed = [2, 3];
  var xs = [1, ...seed, ...?coPick(false), 4]; // spread + null-aware spread
  check("co3", xs.length == 4 && xs[1] == 2 && xs[3] == 4 && [0, ...?coPick(true)][1] == 9);
  var flag = false;
  var ys = [0, if (flag) 1 else 2, for (var i = 0; i < 2; i++) 10 + i];
  check("co4", ys.length == 4 && ys[1] == 2 && ys[3] == 11); // collection if / for
  var mp = {"a": 1, if (!flag) "b": 2, ...{"c": 3}}; // map spread + collection-if
  check("co5", mp.length == 3 && mp["b"] == 2 && mp["c"] == 3);
  var sq = {for (var i in [1, 2, 3]) i: i * i}; // map comprehension
  check("co6", sq[3] == 9 && sq.length == 3);
}

// ===== SECTION 04: records =====
(int, String) rcMake() => (7, "w"); // record-typed return
void s04() {
  var pair = (1, "a");
  check("rc1", pair.$1 == 1 && pair.$2 == "a"); // positional field access
  var named = (x: 3, y: 4);
  check("rc2", named.x == 3 && named.y == 4);
  var mixed = (5, tag: "t", 6);
  check("rc3", mixed.$1 == 5 && mixed.$2 == 6 && mixed.tag == "t");
  check("rc4", (1, 2) == (1, 2) && (x: 1, y: 2) == (y: 2, x: 1) && (1, 2) != (2, 1));
  ({int w, int h}) dims = (w: 2, h: 3); // record type annotation
  check("rc5", dims.w * dims.h == 6);
  var single = (9,); // one-element record needs the trailing comma
  check("rc6", single.$1 == 9 && rcMake().$1 == 7 && rcMake().$2 == "w");
}

// ===== SECTION 05: destructuring and pattern assignment =====
(int, int) dsPoint() => (3, 4);
void s05() {
  var (a, b) = dsPoint(); // record destructuring declaration
  check("ds1", a == 3 && b == 4);
  final (x, y: why) = (1, y: 2); // named-field destructuring
  check("ds2", x == 1 && why == 2);
  var [h, ...tail] = [1, 2, 3]; // list pattern with a binding rest
  check("ds3", h == 1 && tail.length == 2 && tail[1] == 3);
  var [f, ..., l] = [9, 8, 7, 6]; // rest without a binding
  check("ds4", f == 9 && l == 6);
  var {"k": kv} = {"k": 5, "z": 0}; // map pattern ignores extra keys
  check("ds5", kv == 5);
  var ((n1, n2), n3) = ((1, 2), 3); // nested record pattern
  check("ds6", n1 + n2 + n3 == 6);
  int sa = 1, sb = 2;
  (sa, sb) = (sb, sa); // pattern assignment swaps
  check("ds7", sa == 2 && sb == 1);
  var total = 0;
  for (var (p, q) in [(1, 2), (3, 4)]) { total = total + p * q; } // for-in pattern
  check("ds8", total == 14);
}

// ===== SECTION 06: pattern kinds and if-case =====
int? pkMaybe(bool b) => b ? 5 : null;
void s06() {
  var flag = "no";
  if ((1, 2) case (int a, int b)) { flag = "ok$a$b"; } // if-case statement
  check("pk1", flag == "ok12");
  var g = "";
  if (7 case int v when v > 5) { g = "big$v"; } else { g = "small"; } // guard + else
  check("pk2", g == "big7");
  var m1 = switch (pkMaybe(true)) { var v? => v + 1, null => 0 }; // null-check pattern
  var m2 = switch (pkMaybe(false)) { var v? => v + 1, null => 0 };
  check("pk3", m1 == 6 && m2 == 0);
  (int?, int?) pos = (2, 3);
  var (px!, py!) = pos; // null-assert patterns
  check("pk4", px + py == 5);
  (num, Object) rc = (1, "k");
  var (ci as int, cs as String) = rc; // cast patterns
  check("pk5", ci == 1 && cs == "k");
  var band = switch (15) { < 10 => "lo", >= 10 && < 20 => "mid", _ => "hi" };
  var pick = switch (2) { 1 || 2 => "ab", _ => "z" }; // relational, &&, || patterns
  check("pk6", band == "mid" && pick == "ab");
}

// ===== SECTION 07: switch statements =====
String swWord(int n) {
  switch (n) {
    case 0: return "zero";
    case 1: // an empty case shares the next body
    case 2: return "few";
    case var v when v < 0: return "neg"; // pattern case with a guard
    default: return "many";
  }
}
void s07() {
  check("sw1", swWord(0) == "zero" && swWord(1) == "few" && swWord(2) == "few");
  check("sw2", swWord(-4) == "neg" && swWord(9) == "many");
  var log = "";
  switch ("b") {
    case "a": log = "A";
    case "b": log = "B"; break; // no fallthrough; break is allowed, not required
    case "c": log = "C";
  }
  check("sw3", log == "B");
}

// ===== SECTION 08: switch expressions and sealed classes =====
sealed class SeShape {}
class SeCircle extends SeShape { final int r; SeCircle(this.r); }
class SeSquare extends SeShape { final int side; SeSquare(this.side); }
int seArea(SeShape s) => switch (s) {
  SeCircle(r: var r) when r > 10 => 999, // object pattern + guard
  SeCircle(r: var r) => 3 * r * r,
  SeSquare(:var side) => side * side, // getter-name shorthand
}; // exhaustive without a wildcard: the class is sealed
String seSign(int n) => switch (n) { < 0 => "-", 0 => "0", _ => "+" };
void s08() {
  check("se1", seArea(SeCircle(2)) == 12 && seArea(SeCircle(11)) == 999);
  check("se2", seArea(SeSquare(4)) == 16);
  SeShape any = SeCircle(1);
  check("se3", seArea(any) == 3); // dispatch through the sealed supertype
  check("se4", seSign(-5) == "-" && seSign(0) == "0" && seSign(2) == "+");
}

// ===== SECTION 09: null safety =====
class NsBox { int v = 3; int twice() => v * 2; }
NsBox? nsPick(bool b) => b ? NsBox() : null;
List<int>? nsList(bool b) => b ? [4, 5] : null;
void s09() {
  int? empty;
  check("ns1", empty == null); // nullable locals default to null
  int? q;
  q ??= 9; // null-aware assignment
  check("ns2", q == 9);
  check("ns3", (nsPick(true)?.v ?? 8) == 3 && (nsPick(false)?.v ?? 8) == 8);
  check("ns4", nsPick(false)?.twice() == null && nsPick(true)!.twice() == 6);
  check("ns5", nsList(false)?[0] == null && nsList(true)?[1] == 5); // null-aware index
  int? sv = nsList(true)?[0];
  check("ns6", sv! + 1 == 5); // null-assert expression
  late int lv; lv = 7;
  late final int lf; lf = 8;
  check("ns7", lv + lf == 15); // late and late final locals
  int da; // definitely assigned below on every path
  if (checks >= 0) { da = 1; } else { da = 2; }
  check("ns8", da == 1);
}

// ===== SECTION 10: functions and tear-offs =====
int fnOpt(int a, [int b = 2, int? c]) => a + b + (c ?? 0); // optional positional
int fnNamed({int a = 1, required int b}) => a * 10 + b; // named with required
T fnFirst<T>(List<T> xs) => xs[0]; // generic function
class FnBox {
  static int made = 0; // static field
  static int twiceOf(int x) => x * 2; // static method
  final int v;
  FnBox(this.v) { made = made + 1; }
  FnBox.twin(int x) : v = x * 2;
  int plus(int d) => v + d;
}
void s10() {
  check("fn1", fnOpt(1) == 3 && fnOpt(1, 5) == 6 && fnOpt(1, 5, 10) == 16);
  check("fn2", fnNamed(b: 7) == 17 && fnNamed(a: 2, b: 3) == 23);
  int local(int x) => x + 100; // local function declaration
  check("fn3", local(1) == 101);
  check("fn4", fnFirst<int>([8, 9]) == 8 && fnFirst(["a"]) == "a");
  var box = FnBox(5);
  var tear = box.plus; // method tear-off
  var make = FnBox.new; // constructor tear-off (2.15)
  var twin = FnBox.twin; // named-constructor tear-off
  check("fn5", tear(2) == 7 && make(9).v == 9 && twin(3).v == 6);
  check("fn6", FnBox.made == 2 && FnBox.twiceOf(21) == 42);
}

// ===== SECTION 11: typedefs =====
typedef TdOp = int Function(int, int); // function-type alias
typedef int TdOld(int a); // legacy typedef form
typedef TdSelf<T> = T Function(T); // generic alias
typedef TdPair = (int, String); // record-type alias
int tdRun(TdOp op) => op(6, 7); // alias used as a parameter type
void s11() {
  TdOp mul = (a, b) => a * b;
  check("td1", mul(3, 4) == 12 && tdRun(mul) == 42);
  TdOld neg = (a) => -a;
  TdSelf<int> dbl = (x) => x + x;
  check("td2", neg(5) == -5 && dbl(8) == 16);
  TdPair pr = (1, "a");
  check("td3", pr.$1 == 1 && pr.$2 == "a");
}

// ===== SECTION 12: operator overloading and cascades =====
class OvVec {
  final int x, y;
  const OvVec(this.x, this.y);
  OvVec operator +(OvVec o) => OvVec(x + o.x, y + o.y);
  OvVec operator -() => OvVec(-x, -y);
  int operator [](int i) => i == 0 ? x : y;
  bool operator ==(Object o) => o is OvVec && o.x == x && o.y == y;
  int get hashCode => x * 31 + y;
}
class OvCell {
  int a = 0, b = 0;
  int get total => a + b; // instance getter
  set both(int nv) { a = nv; b = nv; } // instance setter
  int operator [](int i) => i == 0 ? a : b;
  void operator []=(int i, int nv) { if (i == 0) { a = nv; } else { b = nv; } }
}
OvCell? ovPick(bool b) => b ? OvCell() : null;
void s12() {
  var v = OvVec(1, 2) + OvVec(3, 4);
  check("op1", v.x == 4 && v.y == 6 && (-OvVec(1, 2)).y == -2);
  check("op2", v[0] == 4 && v[1] == 6); // user-defined []
  check("op3", OvVec(1, 2) == OvVec(1, 2) && OvVec(1, 2) != OvVec(2, 1));
  var c = OvCell();
  c[0] = 5;
  c[1] = c[0] + 1; // user-defined []=
  check("op4", c.a == 5 && c.b == 6);
  c.both = 7;
  check("op5", c.total == 14);
  var d = OvCell()..a = 1..[1] = 9; // cascade: setter + index assignment
  check("op6", d.a == 1 && d.b == 9);
  var e = ovPick(true)?..a = 2; // null-aware cascade returns the receiver
  var z = ovPick(false)?..a = 8;
  check("op7", e!.a == 2 && z == null);
}

// ===== SECTION 13: constructors =====
class Kn {
  final int a;
  final int b;
  Kn(this.a, this.b);
  Kn.unit() : this(1, 1); // redirecting generative constructor
  Kn.diag(int v) : a = v, b = v + v; // initializer list
  Kn.pos(int v) : assert(v >= 0), a = v, b = 0; // assert in the initializer list
  factory Kn.grown(int v) { return v > 100 ? Kn(100, 100) : Kn(v * 2, v); }
  const Kn.fix(this.a, this.b); // const constructor
  int sum() => a + b;
}
void s13() {
  check("ct1", Kn.unit().sum() == 2 && Kn.diag(3).b == 6 && Kn.pos(4).a == 4);
  check("ct2", Kn.grown(5).sum() == 15 && Kn.grown(200).a == 100);
  const k1 = Kn.fix(2, 3); // implicit const on the right-hand side
  check("ct3", identical(k1, const Kn.fix(2, 3)) && k1.b == 3); // canonicalized
}

// ===== SECTION 14: inheritance =====
class IhBase { final int v; IhBase(this.v); int twice() => v * 2; }
class IhKid extends IhBase {
  IhKid(super.v); // super parameter (2.17)
  int twice() => super.twice() + 1; // override + super call
}
class IhFake implements IhBase { final int v = 99; int twice() => 5; } // interface only
abstract class IhAbs { int go(); int go2() => go() * 2; } // abstract member
class IhImp extends IhAbs { int go() => 4; }
void s14() {
  check("ih1", IhBase(3).twice() == 6 && IhKid(3).twice() == 7);
  IhBase pb = IhKid(1);
  check("ih2", pb.twice() == 3); // dynamic dispatch
  IhBase fb = IhFake();
  check("ih3", fb.twice() == 5 && fb.v == 99);
  check("ih4", IhImp().go2() == 8);
}

// ===== SECTION 15: mixins and class modifiers =====
mixin MxA { int mxa() => 1; String who() => "A"; }
mixin MxB { int mxb() => 2; String who() => "B"; }
class MxHost { String who() => "H"; }
class MxUse extends MxHost with MxA, MxB {} // the later mixin wins
mixin MxOn on MxHost { String who() => "M+" + super.who(); String loud() => who() + "!"; }
class MxDeep extends MxHost with MxOn {} // on-clause: super goes through MxHost
base class MdB { int f() => 10; }
final class MdF extends MdB {} // base/final chain (3.0), same library
interface class MdI { int g() => 20; }
class MdUseI implements MdI { int g() => 21; } // same-library implements is allowed
mixin class MdMC { int h() => 30; } // usable as a mixin and as a class (3.0)
class MdWith with MdMC {}
void s15() {
  check("mx1", MxUse().mxa() == 1 && MxUse().mxb() == 2 && MxUse().who() == "B");
  check("mx2", MxDeep().who() == "M+H" && MxDeep().loud() == "M+H!");
  check("mx3", MdF().f() == 10 && MdWith().h() == 30);
  check("mx4", MdUseI().g() == 21);
}

// ===== SECTION 16: enums =====
enum EnDir { north, east, south }
enum EnPlanet { // enhanced enum with members
  mercury(1), earth(3);
  final int order;
  const EnPlanet(this.order);
  bool inner() => order < 2;
  String get label => "p$order";
}
void s16() {
  var d = EnDir.east;
  check("en1", d == EnDir.east && d != EnDir.south);
  check("en2", EnDir.east.index == 1 && EnDir.values.length == 3);
  check("en3", EnPlanet.mercury.inner() && !EnPlanet.earth.inner());
  check("en4", EnPlanet.earth.label == "p3");
  var s = switch (d) { EnDir.north => "n", EnDir.east => "e", EnDir.south => "s" };
  check("en5", s == "e"); // exhaustive without a wildcard
}

// ===== SECTION 17: extensions =====
extension ExInt on int {
  int get squared => this * this;
  int addTo(int o) => this + o;
}
extension ExList<T> on List<T> { T orElse(int i, T fb) => i < length ? this[i] : fb; }
extension type ExMeters(int value) { // extension type (3.3)
  ExMeters operator +(ExMeters o) => ExMeters(value + o.value);
  int get doubledUp => value * 2;
}
void s17() {
  check("ex1", 3.squared == 9 && 4.addTo(5) == 9);
  check("ex2", ExInt(6).squared == 36); // explicit extension application
  check("ex3", [7, 8].orElse(1, 0) == 8 && <int>[].orElse(0, 5) == 5); // generic ext
  var m = ExMeters(3) + ExMeters(4);
  check("ex4", m.value == 7 && ExMeters(2).doubledUp == 4);
}

// ===== SECTION 18: generics =====
class GnBox<T extends num> { // bounded type parameter
  final T v;
  GnBox(this.v);
  T get val => v;
  bool bigger(GnBox<T> o) => v > o.v;
}
T gnMax<T extends num>(T a, T b) => a > b ? a : b;
void s18() {
  var bi = GnBox<int>(3); // explicit type argument
  var bd = GnBox(2.5); // inferred as GnBox<double>
  check("gn1", bi.val == 3 && bd.val == 2.5 && bi.bigger(GnBox(1)));
  check("gn2", gnMax(4, 9) == 9 && gnMax<double>(1.5, 0.5) == 1.5);
  Object xs = <int>[1];
  check("gn3", xs is List<int> && <Object>[1] is! List<int>); // reified generics
  Map<String, List<int>> deep = {"a": [1, 2]};
  check("gn4", deep["a"]![1] == 2);
}

// ===== SECTION 19: control flow =====
void s19() {
  int dw = 0;
  do { dw = dw + 1; } while (dw < 3); // do-while
  check("cf1", dw == 3);
  int hits = 0;
  outer:
  for (int i = 0; i < 3; i++) {
    for (int j = 0; j < 3; j++) {
      if (j == 1) { continue outer; } // labeled continue
      if (i == 2) { break outer; } // labeled break
      hits = hits + 1;
    }
  }
  check("cf2", hits == 2);
  int twin = 0;
  for (int i = 0, j = 3; i < j; i++, j--) { twin = twin + 1; } // two decls, two updaters
  check("cf3", twin == 2);
  int setSum = 0;
  for (var v in {5, 6}) { setSum = setSum + v; } // for-in over a set literal
  assert(setSum > 0, "stays positive"); // may be a no-op unless asserts are enabled
  check("cf4", setSum == 11);
}

// ===== SECTION 20: exception refinements =====
class XeA { final int code; XeA(this.code); }
class XeB {}
int xeSteps = 0;
void xeRethrow() {
  try { throw XeA(3); }
  catch (e) { xeSteps = xeSteps + 1; rethrow; } // rethrow the original
}
String xeKind(Object t) {
  try { throw t; }
  on XeA catch (e) { return "A${e.code}"; }
  on XeB { return "B"; } // typed on-clause without a binding
  catch (e, st) { return "other"; } // catch-all with the stack-trace binder
}
int xePick(int? v) => v ?? (throw XeA(9)); // throw is an expression
void s20() {
  var got = "";
  try { xeRethrow(); } on XeA catch (e) { got = "rt${e.code}"; }
  check("xc1", got == "rt3" && xeSteps == 1);
  check("xc2", xeKind(XeA(1)) == "A1" && xeKind(XeB()) == "B" && xeKind("s") == "other");
  var caught = 0;
  try { xePick(null); } on XeA catch (e) { caught = e.code; }
  check("xc3", xePick(4) == 4 && caught == 9);
}

// ===== SECTION 21: async and generator syntax =====
// sync* generators run without an event loop; async bodies are defined only.
dynamic asPair() sync* { yield 2; yield 3; }
dynamic asTrio() sync* { yield 1; yield* asPair(); } // generator delegation
dynamic asOne() async => 1;
dynamic asGen() async* { yield 4; }
dynamic asCollect(dynamic s) async {
  var total = 0;
  await for (var v in s) { total = total + v; } // await-for, never executed
  return await asOne() + total;
}
void s21() {
  var out = "";
  for (var v in asTrio()) { out = out + "$v"; }
  check("as1", out == "123"); // the sync* pipeline really ran
  var fns = [asOne, asGen, asCollect]; // async functions are plain values
  dynamic dyn = asOne;
  var aa = () async => 5; // async closure, never awaited
  check("as2", fns.length == 3 && dyn is Function && [aa].length == 1);
}

// ===== SECTION 22: top-level accessors and const =====
int tvBack = 4;
int get tvTwice => tvBack * 2; // top-level getter
set tvTwice(int nv) { tvBack = nv ~/ 2; } // top-level setter
const int tvTen = 2 * 5; // top-level const expression
const tvNums = [1, 2, 3]; // const list
void s22() {
  check("tv1", tvTwice == 8);
  tvTwice = 10;
  check("tv2", tvBack == 5 && tvTwice == 10);
  check("tv3", tvTen == 10 && tvNums[2] == 3 && tvNums.length == 3);
  const local = tvTen + 1; // const local
  final int fin = local + 1; // final local
  dynamic shifty = 1;
  shifty = "y"; // dynamic rebinds across types
  check("tv4", local == 11 && fin == 12 && shifty == "y");
}

// ===== END SECTIONS =====

int main() {
  s01(); // SECTION-CALL 01
  s02(); // SECTION-CALL 02
  s03(); // SECTION-CALL 03
  s04(); // SECTION-CALL 04
  s05(); // SECTION-CALL 05
  s06(); // SECTION-CALL 06
  s07(); // SECTION-CALL 07
  s08(); // SECTION-CALL 08
  s09(); // SECTION-CALL 09
  s10(); // SECTION-CALL 10
  s11(); // SECTION-CALL 11
  s12(); // SECTION-CALL 12
  s13(); // SECTION-CALL 13
  s14(); // SECTION-CALL 14
  s15(); // SECTION-CALL 15
  s16(); // SECTION-CALL 16
  s17(); // SECTION-CALL 17
  s18(); // SECTION-CALL 18
  s19(); // SECTION-CALL 19
  s20(); // SECTION-CALL 20
  s21(); // SECTION-CALL 21
  s22(); // SECTION-CALL 22
  print("full: $checks checks, $fails failures");
  return fails;
}
