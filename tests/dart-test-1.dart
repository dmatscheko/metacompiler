// Dart self-checking test for the metacompiler.
//
// Exercises arithmetic and precedence, comparisons, boolean logic, strings and string
// interpolation, lists and maps, control flow (if/else, while, C-style for, for-in with
// break/continue), functions (recursion and closures), and Dart's signature OO features
// (classes with fields, methods, constructors, field-initialising formals, implicit and
// explicit this). Every result is checked against an expected value; int main() returns
// the number of failures, so the exit code is 0 on success.

class Rectangle {
  int width;
  int height;
  Rectangle(this.width, this.height);
  int area() => this.width * this.height;
  int perimeter() {
    return 2 * (this.width + this.height);
  }
  bool isSquare() {
    return this.width == this.height;
  }
  String describe() => 'Rect ${this.width}x${this.height}';
}

class Counter {
  int value = 0;
  int step;
  Counter(this.step);
  void increment() {
    value = value + step; // bare field names resolve against this
  }
  int current() {
    return value;
  }
}

int fib(int n) {
  if (n < 2) {
    return n;
  }
  return fib(n - 1) + fib(n - 2);
}

int sumList(List<int> xs) {
  int total = 0;
  for (int x in xs) {
    total = total + x;
  }
  return total;
}

int main() {
  int failures = 0;

  // --- arithmetic and precedence ---
  if (2 + 3 * 4 != 14) failures = failures + 1;
  if ((2 + 3) * 4 != 20) failures = failures + 1;
  if (20 ~/ 3 != 6) failures = failures + 1;
  if (17 % 5 != 2) failures = failures + 1;
  if (-5 + 2 != -3) failures = failures + 1;
  if (100 - 40 - 10 != 50) failures = failures + 1;

  // --- comparisons and boolean logic ---
  if (!(3 < 5 && 5 <= 5)) failures = failures + 1;
  if (!(1 == 1 || 1 == 2)) failures = failures + 1;
  bool flag = (10 > 3) && !(4 == 4);
  if (flag != false) failures = failures + 1;
  if ((2 >= 3) || (7 < 1)) failures = failures + 1;

  // --- ternary ---
  int t = (7 > 4) ? 100 : 200;
  if (t != 100) failures = failures + 1;
  int u = (7 < 4) ? 100 : 200;
  if (u != 200) failures = failures + 1;

  // --- strings and interpolation ---
  String name = 'Dart';
  String greeting = 'Hello, $name!';
  if (greeting != 'Hello, Dart!') failures = failures + 1;
  if (greeting.length != 12) failures = failures + 1;
  if (greeting.substring(0, 5) != 'Hello') failures = failures + 1;
  if (greeting.indexOf('Dart') != 7) failures = failures + 1;
  if ('ab' + 'cd' != 'abcd') failures = failures + 1;
  if ('sum=${2 + 3}' != 'sum=5') failures = failures + 1;
  int n = 3;
  if ('n squared is ${n * n}' != 'n squared is 9') failures = failures + 1;

  // --- lists ---
  List<int> nums = [10, 20, 30, 40];
  if (nums.length != 4) failures = failures + 1;
  if (nums[2] != 30) failures = failures + 1;
  nums.add(50);
  if (nums.length != 5) failures = failures + 1;
  if (!nums.contains(50)) failures = failures + 1;
  if (nums.contains(999)) failures = failures + 1;
  nums[0] = 11;
  if (nums[0] != 11) failures = failures + 1;
  if (sumList([1, 2, 3, 4, 5]) != 15) failures = failures + 1;
  if (sumList(nums) != 151) failures = failures + 1;

  // --- for-in with break and continue ---
  int acc = 0;
  for (int v in [1, 2, 3, 4, 5, 6]) {
    if (v == 4) continue;
    if (v == 6) break;
    acc = acc + v;
  }
  if (acc != 11) failures = failures + 1;

  // --- maps ---
  Map<String, int> ages = {'alice': 30, 'bob': 25};
  if (ages.length != 2) failures = failures + 1;
  if (ages['alice'] != 30) failures = failures + 1;
  if (!ages.containsKey('bob')) failures = failures + 1;
  if (ages.containsKey('carol')) failures = failures + 1;
  ages['carol'] = 40;
  if (ages['carol'] != 40) failures = failures + 1;
  if (ages.length != 3) failures = failures + 1;
  if (ages.keys.length != 3) failures = failures + 1;

  // --- classes ---
  Rectangle r = new Rectangle(3, 4);
  if (r.area() != 12) failures = failures + 1;
  if (r.perimeter() != 14) failures = failures + 1;
  if (r.isSquare()) failures = failures + 1;
  if (r.width != 3) failures = failures + 1;
  if (r.describe() != 'Rect 3x4') failures = failures + 1;
  Rectangle sq = Rectangle(5, 5); // construction without new
  if (!sq.isSquare()) failures = failures + 1;
  if (sq.area() != 25) failures = failures + 1;

  Counter c = Counter(10);
  c.increment();
  c.increment();
  c.increment();
  if (c.current() != 30) failures = failures + 1;
  if (c.value != 30) failures = failures + 1;

  // --- recursion ---
  if (fib(10) != 55) failures = failures + 1;
  if (fib(15) != 610) failures = failures + 1;

  // --- closures with captured scope ---
  int base = 100;
  var addBase = (int x) => x + base;
  if (addBase(5) != 105) failures = failures + 1;
  var makeAdder = (int step) => (int m) => step + m;
  var add10 = makeAdder(10);
  if (add10(7) != 17) failures = failures + 1;

  // --- while loop ---
  int i = 0;
  int wsum = 0;
  while (i < 5) {
    wsum = wsum + i;
    i++;
  }
  if (wsum != 10) failures = failures + 1;

  // --- C-style for loop ---
  int fsum = 0;
  for (int j = 0; j < 6; j = j + 1) {
    fsum = fsum + j;
  }
  if (fsum != 15) failures = failures + 1;

  print('Dart test complete with $failures failures');
  return failures;
}
