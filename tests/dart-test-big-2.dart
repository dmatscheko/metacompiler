// Dart self-checking test 2/4 for the metacompiler: RECURSION + NUMBER THEORY + CONTROL FLOW.
//
// Deep and mutual recursion (fibonacci three ways, factorial, fast exponentiation, gcd/lcm,
// Ackermann, Collatz, Tower of Hanoi, digit manipulation, binomial coefficients), the sieve
// of Eratosthenes cross-checked against trial division, and nested-loop matrix work. Many
// checks compare independent implementations against each other so the assertions verify
// themselves. int main() returns the number of failures, so the exit code is 0 on success
// and the goja and frozen engines must agree byte-for-byte.

// ---------- fibonacci, three ways ----------

int fibRec(int n) {
  if (n < 2) return n;
  return fibRec(n - 1) + fibRec(n - 2);
}

int fibIter(int n) {
  int a = 0;
  int b = 1;
  for (int i = 0; i < n; i = i + 1) {
    int t = a + b;
    a = b;
    b = t;
  }
  return a;
}

int fibMemo(int n, Map<int, int> memo) {
  if (n < 2) return n;
  if (memo.containsKey(n)) return memo[n];
  int v = fibMemo(n - 1, memo) + fibMemo(n - 2, memo);
  memo[n] = v;
  return v;
}

// ---------- factorial and exponentiation ----------

int factRec(int n) {
  if (n <= 1) return 1;
  return n * factRec(n - 1);
}

int powLoop(int base, int exp) {
  int r = 1;
  for (int i = 0; i < exp; i = i + 1) {
    r = r * base;
  }
  return r;
}

// fast exponentiation by squaring (recursive)
int powFast(int base, int exp) {
  if (exp == 0) return 1;
  int half = powFast(base, exp ~/ 2);
  if (exp % 2 == 0) return half * half;
  return half * half * base;
}

// ---------- gcd / lcm ----------

int gcd(int a, int b) {
  if (b == 0) return a;
  return gcd(b, a % b);
}

int lcm(int a, int b) => (a ~/ gcd(a, b)) * b;

// ---------- Ackermann (small arguments only) ----------

int ackermann(int m, int n) {
  if (m == 0) return n + 1;
  if (n == 0) return ackermann(m - 1, 1);
  return ackermann(m - 1, ackermann(m, n - 1));
}

// ---------- Collatz ----------

int collatzSteps(int n) {
  int steps = 0;
  while (n != 1) {
    if (n % 2 == 0) {
      n = n ~/ 2;
    } else {
      n = 3 * n + 1;
    }
    steps = steps + 1;
  }
  return steps;
}

// ---------- Tower of Hanoi (move count) ----------

int hanoi(int n) {
  if (n == 0) return 0;
  return 2 * hanoi(n - 1) + 1;
}

// ---------- mutual recursion ----------

bool isEven(int n) {
  if (n == 0) return true;
  return isOdd(n - 1);
}

bool isOdd(int n) {
  if (n == 0) return false;
  return isEven(n - 1);
}

// ---------- digit manipulation ----------

int sumDigits(int n) {
  if (n < 10) return n;
  return n % 10 + sumDigits(n ~/ 10);
}

int numDigits(int n) {
  int d = 0;
  while (n > 0) {
    d = d + 1;
    n = n ~/ 10;
  }
  return d;
}

int reverseNumber(int n) {
  int r = 0;
  while (n > 0) {
    r = r * 10 + n % 10;
    n = n ~/ 10;
  }
  return r;
}

// ---------- primes ----------

bool isPrime(int n) {
  if (n < 2) return false;
  int i = 2;
  while (i * i <= n) {
    if (n % i == 0) return false;
    i = i + 1;
  }
  return true;
}

int countPrimesTrial(int limit) {
  int c = 0;
  for (int n = 2; n < limit; n = n + 1) {
    if (isPrime(n)) c = c + 1;
  }
  return c;
}

int countPrimesSieve(int limit) {
  List<bool> sieve = [];
  for (int i = 0; i < limit; i = i + 1) {
    sieve.add(true);
  }
  int c = 0;
  for (int p = 2; p < limit; p = p + 1) {
    if (sieve[p]) {
      c = c + 1;
      int mul = p * p;
      while (mul < limit) {
        sieve[mul] = false;
        mul = mul + p;
      }
    }
  }
  return c;
}

// ---------- binomial coefficients ----------

int binomRec(int n, int k) {
  if (k == 0 || k == n) return 1;
  return binomRec(n - 1, k - 1) + binomRec(n - 1, k);
}

// Pascal's triangle: build rows with nested loops, return C(n, k).
int binomPascal(int n, int k) {
  List<List<int>> tri = [];
  for (int r = 0; r <= n; r = r + 1) {
    List<int> row = [];
    for (int c = 0; c <= r; c = c + 1) {
      if (c == 0 || c == r) {
        row.add(1);
      } else {
        row.add(tri[r - 1][c - 1] + tri[r - 1][c]);
      }
    }
    tri.add(row);
  }
  return tri[n][k];
}

// ---------- matrices (nested lists) ----------

List<List<int>> matMul(List<List<int>> a, List<List<int>> b, int n) {
  List<List<int>> out = [];
  for (int i = 0; i < n; i = i + 1) {
    List<int> row = [];
    for (int j = 0; j < n; j = j + 1) {
      int s = 0;
      for (int k = 0; k < n; k = k + 1) {
        s = s + a[i][k] * b[k][j];
      }
      row.add(s);
    }
    out.add(row);
  }
  return out;
}

bool matEq(List<List<int>> a, List<List<int>> b, int n) {
  for (int i = 0; i < n; i = i + 1) {
    for (int j = 0; j < n; j = j + 1) {
      if (a[i][j] != b[i][j]) return false;
    }
  }
  return true;
}

int main() {
  int failures = 0;

  // ---------- fibonacci: three implementations agree ----------
  Map<int, int> memo = {};
  for (int n = 0; n <= 20; n = n + 1) {
    int a = fibRec(n);
    int b = fibIter(n);
    int c = fibMemo(n, memo);
    if (a != b) failures = failures + 1;
    if (b != c) failures = failures + 1;
  }
  if (fibIter(10) != 55) failures = failures + 1;
  if (fibMemo(40, memo) != 102334155) failures = failures + 1;
  if (fibIter(40) != 102334155) failures = failures + 1;

  // ---------- factorial ----------
  if (factRec(0) != 1) failures = failures + 1;
  if (factRec(5) != 120) failures = failures + 1;
  if (factRec(10) != 3628800) failures = failures + 1;

  // ---------- exponentiation: loop and fast agree ----------
  for (int e = 0; e <= 10; e = e + 1) {
    if (powLoop(2, e) != powFast(2, e)) failures = failures + 1;
  }
  if (powFast(2, 10) != 1024) failures = failures + 1;
  if (powFast(3, 7) != 2187) failures = failures + 1;
  if (powLoop(5, 4) != 625) failures = failures + 1;

  // ---------- gcd / lcm ----------
  if (gcd(48, 36) != 12) failures = failures + 1;
  if (gcd(17, 5) != 1) failures = failures + 1;
  if (gcd(100, 80) != 20) failures = failures + 1;
  if (lcm(4, 6) != 12) failures = failures + 1;
  if (lcm(21, 6) != 42) failures = failures + 1;
  // gcd(a,b) * lcm(a,b) == a * b
  if (gcd(12, 18) * lcm(12, 18) != 12 * 18) failures = failures + 1;

  // ---------- Ackermann ----------
  if (ackermann(0, 0) != 1) failures = failures + 1;
  if (ackermann(2, 2) != 7) failures = failures + 1;
  if (ackermann(3, 3) != 61) failures = failures + 1;

  // ---------- Collatz (powers of two take exactly k steps) ----------
  if (collatzSteps(1) != 0) failures = failures + 1;
  if (collatzSteps(2) != 1) failures = failures + 1;
  if (collatzSteps(8) != 3) failures = failures + 1;
  if (collatzSteps(1024) != 10) failures = failures + 1;
  if (collatzSteps(6) != 8) failures = failures + 1;

  // ---------- Tower of Hanoi ----------
  if (hanoi(3) != 7) failures = failures + 1;
  if (hanoi(10) != 1023) failures = failures + 1;
  // hanoi(n) == 2^n - 1
  if (hanoi(12) != powFast(2, 12) - 1) failures = failures + 1;

  // ---------- mutual recursion ----------
  if (!isEven(10)) failures = failures + 1;
  if (isEven(7)) failures = failures + 1;
  if (!isOdd(9)) failures = failures + 1;
  for (int n = 0; n <= 12; n = n + 1) {
    if (isEven(n) == isOdd(n)) failures = failures + 1;
  }

  // ---------- digit manipulation ----------
  if (sumDigits(1234) != 10) failures = failures + 1;
  if (sumDigits(9999) != 36) failures = failures + 1;
  if (numDigits(1234) != 4) failures = failures + 1;
  if (numDigits(7) != 1) failures = failures + 1;
  if (reverseNumber(1234) != 4321) failures = failures + 1;
  if (reverseNumber(1000) != 1) failures = failures + 1;
  // reversing a palindrome number is a fixed point
  if (reverseNumber(12321) != 12321) failures = failures + 1;

  // ---------- primes: sieve and trial division agree ----------
  if (isPrime(2) != true) failures = failures + 1;
  if (isPrime(1)) failures = failures + 1;
  if (isPrime(15)) failures = failures + 1;
  if (!isPrime(29)) failures = failures + 1;
  if (countPrimesTrial(30) != 10) failures = failures + 1;
  if (countPrimesSieve(30) != countPrimesTrial(30)) failures = failures + 1;
  if (countPrimesSieve(100) != countPrimesTrial(100)) failures = failures + 1;
  if (countPrimesSieve(100) != 25) failures = failures + 1;

  // ---------- binomial coefficients: recursive and Pascal agree ----------
  if (binomRec(5, 2) != 10) failures = failures + 1;
  if (binomPascal(6, 3) != 20) failures = failures + 1;
  for (int n = 0; n <= 8; n = n + 1) {
    int rowSum = 0;
    for (int k = 0; k <= n; k = k + 1) {
      int viaRec = binomRec(n, k);
      int viaPascal = binomPascal(n, k);
      if (viaRec != viaPascal) failures = failures + 1;
      rowSum = rowSum + viaRec;
    }
    // the n-th row of Pascal's triangle sums to 2^n
    if (rowSum != powFast(2, n)) failures = failures + 1;
  }

  // ---------- matrices ----------
  List<List<int>> id = [[1, 0], [0, 1]];
  List<List<int>> m = [[1, 2], [3, 4]];
  // multiplying by the identity leaves m unchanged
  if (!matEq(matMul(m, id, 2), m, 2)) failures = failures + 1;
  // known product
  List<List<int>> prod = matMul(m, [[5, 6], [7, 8]], 2);
  if (!matEq(prod, [[19, 22], [43, 50]], 2)) failures = failures + 1;
  // matrix multiplication is associative: (AB)C == A(BC)
  List<List<int>> aa = [[1, 2], [0, 1]];
  List<List<int>> bb = [[2, 0], [1, 3]];
  List<List<int>> cc = [[1, 1], [1, 0]];
  List<List<int>> left = matMul(matMul(aa, bb, 2), cc, 2);
  List<List<int>> right = matMul(aa, matMul(bb, cc, 2), 2);
  if (!matEq(left, right, 2)) failures = failures + 1;

  print('dart-test-big-2 (recursion + number theory) finished with $failures failures');
  return failures;
}
