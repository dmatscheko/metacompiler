// Recognition test for the widened Dart grammar.
//
// This file uses many real-world Dart constructs that the grammar RECOGNIZES but does
// not implement. Because those constructs abort by default, a normal run fails; under
// -warn-unsupported each one warns and the program runs to completion, so main() returns
// 0. The genuinely-supported additions (bitwise/shift operators, the `as` identity cast,
// typed list/map literals, and `?.` on a non-null target) actually execute.
//
//   ./mec languages/dart-to-llvm-ir.abnf tests/dart-test-recognize.dart -q                    (fails)
//   ./mec languages/dart-to-llvm-ir.abnf tests/dart-test-recognize.dart -q -warn-unsupported  (exit 0)

// --- Directives: import now resolves ('dart:*' is a builtin no-op, project files load via
//     -i); library / export / part remain not implemented ---
library recognize_demo;
import 'dart:math';
import 'dart:collection' as coll;
export 'src/helpers.dart' show Helper hide Internal;
part 'recognize_part.dart';

// --- Annotations are parsed and erased ---
@deprecated
const int schemaVersion = 3;

// --- Top-level declarations (enum / mixin / extension / typedef: not implemented) ---
enum Direction { north, south, east, west }

mixin Logger {
  void log(String m) { print(m); }
}

extension NumberParsing on String {
  int toIntOr(int fallback) => fallback;
}

typedef IntTransform = int Function(int);
typedef void OldStyle(int value);

// --- An abstract class with an abstract method (modifier erased, empty body) ---
abstract class Shape {
  int sides();
  String describe() => "a shape";
}

// --- A class exercising members that are recognized but not implemented ---
@immutable
class Circle extends Shape {
  final int radius;
  static const double pi = 3;
  late int cached;

  Circle(this.radius);

  factory Circle.unit() => Circle(1);

  int get area => (pi * radius * radius) ~/ 1;
  get diameter => radius * 2;
  set scale(int factor) { cached = factor; }

  Circle operator +(Circle other) => Circle(radius + other.radius);

  @override
  int sides() => 0;
}

// --- A function full of not-implemented statements (it is never called, but every tag
//     still fires its warning while compiling). ---
int classify(int n) {
  switch (n) {
    case 0:
      return 0;
    case 1:
    case 2:
      return 1;
    default:
      break;
  }
  try {
    assert(n >= 0, "must be non-negative");
    if (n < 0) { throw "negative"; }
  } on FormatException catch (e) {
    print(e);
  } catch (e, stack) {
    rethrow;
  } finally {
    print("checked");
  }
  var i = 0;
  assert(i >= 0);
  do { i = i + 1; } while (i < 0);
  outer:
  for (var j = 0; j < 3; j++) {
    for (var k = 0; k < 3; k++) {
      if (k == 1) { continue outer; }
      break outer;
    }
  }
  return n;
}

int main() {
  // Bitwise and shift operators are genuinely compiled.
  var bits = (6 & 3) | (1 << 2) ^ (~1 >>> 30);
  var masked = bits & 255;

  var name = "dart";
  var len = name?.length;              // ?. on a non-null string behaves like .
  var first = name?.indexOf("d");      // ?. method call on a non-null string
  var isText = name is String;         // is: not implemented (operand still evaluated)
  var seven = 7 as int;                // as: identity cast
  var chosen = len ?? 0;               // ??: len is non-null, so it is returned
  chosen ??= 99;                       // ??=: lowers to a plain assignment
  var picked = true ? seven : throw "unreachable";  // throw only in the untaken branch

  var typedList = <int>[1, 2, 3];      // typed list literal (genuine)
  var typedMap = <String, int>{"a": 1}; // typed map literal (genuine)
  var aSet = {10, 20, 30};             // set literal: not implemented

  // A genuine self-check: total==4 (list/map lengths + ?. result + masked bits - seven),
  // bits==7 (all bitwise/shift/~ operators), picked==7 (ternary, throw untaken). The sum
  // is 0 only if every genuinely-compiled construct produced the expected value.
  var total = typedList.length + typedMap.length + first + masked - seven;
  return total + bits + picked - 18;
}
