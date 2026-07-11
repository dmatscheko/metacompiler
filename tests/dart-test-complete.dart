// Self-checking test for the newly completed Dart features:
//   * for (var x in coll) over lists AND maps - a map yields its keys, in insertion order
//   * the cascade operator  target..m(args)..field = v  (evaluates the target once and
//     returns it), including named arguments in a cascaded call
//   * collection-if and spread inside list literals ([a, ...rest, if (c) x else y, b])
//   * named parameters with defaults, foo({int a = 1}), called as foo(a: 2) in any order,
//     as plain functions, as instance methods, and as field-initialising constructor formals
//
// Every result is checked against an expected value; main() returns the number of
// failures, so the process exit code is 0 on success.

// Named parameters on a plain function, each with a default.
int volume({int w = 1, int h = 1, int d = 1}) {
  return w * h * d;
}

// A positional parameter followed by a named one with a default.
int shift(int base, {int by = 10}) {
  return base + by;
}

class Vec {
  int x;
  int y;
  int z;
  // Named field-initialising formals with defaults.
  Vec({this.x = 0, this.y = 0, this.z = 0});
  int norm1() => this.x + this.y + this.z;
  // A method with named parameters and defaults.
  void bump({int dx = 0, int dy = 0}) {
    x = x + dx;
    y = y + dy;
  }
}

int sumList(List<int> xs) {
  int total = 0;
  for (var v in xs) {
    total = total + v;
  }
  return total;
}

int main() {
  int failures = 0;

  // ---------- for-in over a map walks its keys (insertion order) ----------
  Map<String, int> price = {'pen': 3, 'cup': 7, 'hat': 12};
  int totalPrice = 0;
  String keyOrder = '';
  for (var name in price) {
    keyOrder = keyOrder + name + ',';
    totalPrice = totalPrice + price[name];
  }
  if (totalPrice != 22) failures = failures + 1;
  if (keyOrder != 'pen,cup,hat,') failures = failures + 1;

  // for-in over a map with break and continue
  Map<String, int> weights = {'a': 1, 'b': 2, 'c': 3, 'd': 4};
  int acc = 0;
  for (var k in weights) {
    if (k == 'b') continue;
    if (k == 'd') break;
    acc = acc + weights[k];
  }
  if (acc != 4) failures = failures + 1; // a(1) + c(3); b skipped, stop at d

  // for-in over a plain list still works
  if (sumList([5, 6, 7]) != 18) failures = failures + 1;

  // ---------- cascade ----------
  // on a list literal: several adds, the whole expression is the list itself
  List<int> built = [1, 2]..add(3)..add(4)..add(5);
  if (built.length != 5) failures = failures + 1;
  if (sumList(built) != 15) failures = failures + 1;

  // on a constructed object: assign fields and call a (named-arg) method, result = object
  Vec v = Vec()..x = 3..y = 4..bump(dx: 1, dy: 1);
  if (v.x != 4) failures = failures + 1;
  if (v.y != 5) failures = failures + 1;
  if (v.norm1() != 9) failures = failures + 1;

  // ---------- collection-if and spread in list literals ----------
  List<int> parts = [10, 20];
  List<int> combined = [0, ...parts, if (true) 30, if (false) 99, 40];
  if (combined.length != 5) failures = failures + 1;
  if (sumList(combined) != 100) failures = failures + 1;
  if (combined[1] != 10) failures = failures + 1;
  if (combined[3] != 30) failures = failures + 1;

  // collection-if with an else branch, followed by a spread
  List<int> labels = [if (false) 1 else 2, ...parts];
  if (sumList(labels) != 32) failures = failures + 1; // 2 + 10 + 20
  if (labels.length != 3) failures = failures + 1;

  // ---------- named parameters with defaults ----------
  if (volume() != 1) failures = failures + 1;              // all defaults
  if (volume(w: 2, h: 3) != 6) failures = failures + 1;    // override two, d defaults
  if (volume(d: 4, w: 5) != 20) failures = failures + 1;   // any order, h defaults
  if (shift(100) != 110) failures = failures + 1;          // positional + default named
  if (shift(100, by: 5) != 105) failures = failures + 1;   // positional + named override

  // named field-initialising constructor formals
  Vec a = Vec();
  if (a.norm1() != 0) failures = failures + 1;
  Vec b = Vec(x: 1, y: 2, z: 3);
  if (b.norm1() != 6) failures = failures + 1;
  Vec c = Vec(y: 5);
  if (c.x != 0) failures = failures + 1;
  if (c.y != 5) failures = failures + 1;
  if (c.norm1() != 5) failures = failures + 1;

  // named parameters on an instance method (unsupplied ones keep their default)
  Vec d = Vec(x: 1, y: 1);
  d.bump(dx: 10);
  if (d.x != 11) failures = failures + 1;
  if (d.y != 1) failures = failures + 1;

  print('Dart completion test finished with $failures failures');
  return failures;
}
