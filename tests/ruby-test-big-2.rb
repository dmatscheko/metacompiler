# Ruby subset - big self test 2: RECURSION AND CONTROL FLOW.
# Fibonacci (recursive / iterative / hash-memoized), factorial, fast exponentiation,
# Ackermann, Towers of Hanoi, Collatz, mutual recursion (even/odd), the sieve of
# Eratosthenes and trial-division primality, Pascal's triangle, a 2D dynamic-
# programming grid-path count, a mod-3 finite state machine, bracket balancing and
# a triple-nested Pythagorean-triple search. Counts failures and ends with
# exit(fails), so exit 0 means the interpreter and the LLVM-IR compiler agree.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- Fibonacci: three ways, all must agree -----

def fib_rec(n)
  return n if n < 2
  fib_rec(n - 1) + fib_rec(n - 2)
end

def fib_iter(n)
  a = 0
  b = 1
  i = 0
  while i < n
    t = a + b
    a = b
    b = t
    i += 1
  end
  a
end

def fib_memo(n, memo)
  return n if n < 2
  return memo[n] if memo.include?(n)
  v = fib_memo(n - 1, memo) + fib_memo(n - 2, memo)
  memo[n] = v
  v
end

check("fib_rec 10", fib_rec(10), 55)
check("fib_iter 10", fib_iter(10), 55)
check("fib_memo 10", fib_memo(10, {}), 55)
check("fib_iter 20", fib_iter(20), 6765)
check("fib_memo 30", fib_memo(30, {}), 832040)
# the three definitions agree across a whole range
agree = true
n = 0
while n <= 20
  if fib_rec(n) != fib_iter(n)
    agree = false
  end
  n += 1
end
check("fib methods agree", agree, true)

# ----- factorial and fast exponentiation -----

def factorial(n)
  result = 1
  i = 2
  while i <= n
    result *= i
    i += 1
  end
  result
end

check("factorial 0", factorial(0), 1)
check("factorial 5", factorial(5), 120)
check("factorial 10", factorial(10), 3628800)

def power(base, exp)
  return 1 if exp == 0
  half = power(base, exp / 2)
  if exp % 2 == 0
    half * half
  else
    half * half * base
  end
end

check("power 2^0", power(2, 0), 1)
check("power 2^10", power(2, 10), 1024)
check("power 3^7", power(3, 7), 2187)
check("power 5^5", power(5, 5), 3125)

# ----- Ackermann (deep recursion, small arguments) -----

def ackermann(m, n)
  return n + 1 if m == 0
  return ackermann(m - 1, 1) if n == 0
  ackermann(m - 1, ackermann(m, n - 1))
end

check("ack(0,0)", ackermann(0, 0), 1)
check("ack(2,3)", ackermann(2, 3), 9)
check("ack(3,3)", ackermann(3, 3), 61)

# ----- Towers of Hanoi: minimum move count -----

def hanoi_moves(n)
  return 0 if n == 0
  2 * hanoi_moves(n - 1) + 1
end

check("hanoi 1", hanoi_moves(1), 1)
check("hanoi 5", hanoi_moves(5), 31)
check("hanoi 10", hanoi_moves(10), 1023)

# ----- Collatz sequence length -----

def collatz_length(n)
  steps = 0
  while n != 1
    if n % 2 == 0
      n = n / 2
    else
      n = 3 * n + 1
    end
    steps += 1
  end
  steps
end

check("collatz 1", collatz_length(1), 0)
check("collatz 6", collatz_length(6), 8)
check("collatz 27", collatz_length(27), 111)

# ----- mutual recursion: even / odd -----

def my_even(n)
  return true if n == 0
  my_odd(n - 1)
end

def my_odd(n)
  return false if n == 0
  my_even(n - 1)
end

check("even 0", my_even(0), true)
check("even 10", my_even(10), true)
check("even 7", my_even(7), false)
check("odd 7", my_odd(7), true)

# ----- digit sum and digital root (recursion + loops) -----

def digit_sum(n)
  s = 0
  while n > 0
    s += n % 10
    n = n / 10
  end
  s
end

def digital_root(n)
  while n >= 10
    n = digit_sum(n)
  end
  n
end

check("digit_sum 1234", digit_sum(1234), 10)
check("digit_sum 99999", digit_sum(99999), 45)
check("digital_root 9875", digital_root(9875), 2)

# ----- recursive gcd -----

def gcd_rec(a, b)
  return a if b == 0
  gcd_rec(b, a % b)
end

check("gcd_rec", gcd_rec(1071, 462), 21)
check("gcd_rec coprime", gcd_rec(13, 7), 1)

# ----- sieve of Eratosthenes + trial division -----

def sieve(limit)
  flags = []
  i = 0
  while i <= limit
    flags << (i >= 2)
    i += 1
  end
  p = 2
  while p * p <= limit
    if flags[p]
      k = p * p
      while k <= limit
        flags[k] = false
        k += p
      end
    end
    p += 1
  end
  primes = []
  i = 2
  while i <= limit
    primes << i if flags[i]
    i += 1
  end
  primes
end

def is_prime(n)
  return false if n < 2
  d = 2
  while d * d <= n
    return false if n % d == 0
    d += 1
  end
  true
end

primes = sieve(50)
check("sieve count", primes.size, 15)
check("sieve first", primes[0], 2)
check("sieve last", primes[14], 47)
check("sieve sum", primes.sum, 328)
# the sieve and trial division must classify every number identically
mismatch = 0
n = 0
while n <= 50
  in_sieve = primes.include?(n)
  if in_sieve != is_prime(n)
    mismatch += 1
  end
  n += 1
end
check("sieve vs trial division", mismatch, 0)

# ----- nth prime via a counting loop (no negative sentinel) -----

def nth_prime(k)
  count = 0
  candidate = 1
  answer = 0
  while count < k
    candidate += 1
    if is_prime(candidate)
      count += 1
      answer = candidate
    end
  end
  answer
end

check("1st prime", nth_prime(1), 2)
check("6th prime", nth_prime(6), 13)
check("10th prime", nth_prime(10), 29)

# ----- Pascal's triangle rows -----

def pascal_row(n)
  row = [1]
  k = 1
  while k <= n
    row << row[k - 1] * (n - k + 1) / k
    k += 1
  end
  row
end

def arr_eq(a, b)
  na = a.size
  return false if na != b.size
  i = 0
  while i < na
    return false if a[i] != b[i]
    i += 1
  end
  true
end

check("pascal 0", arr_eq(pascal_row(0), [1]), true)
check("pascal 4", arr_eq(pascal_row(4), [1, 4, 6, 4, 1]), true)
check("pascal 6", arr_eq(pascal_row(6), [1, 6, 15, 20, 15, 6, 1]), true)
check("pascal row sum", pascal_row(8).sum, 256)

# ----- 2D dynamic programming: lattice paths in a grid -----

def grid_paths(rows, cols)
  dp = []
  r = 0
  while r < rows
    line = []
    c = 0
    while c < cols
      if r == 0 || c == 0
        line << 1
      else
        line << 0
      end
      c += 1
    end
    dp << line
    r += 1
  end
  r = 1
  while r < rows
    c = 1
    while c < cols
      dp[r][c] = dp[r - 1][c] + dp[r][c - 1]
      c += 1
    end
    r += 1
  end
  dp[rows - 1][cols - 1]
end

check("grid 1x1", grid_paths(1, 1), 1)
check("grid 3x3", grid_paths(3, 3), 6)
check("grid 4x4", grid_paths(4, 4), 20)
check("grid 3x7", grid_paths(3, 7), 28)

# ----- a mod-3 finite state machine over a binary string -----

def divisible_by_3(bits)
  state = 0
  i = 0
  while i < bits.length
    c = bits[i]
    if c == "0"
      state = (state * 2) % 3
    else
      state = (state * 2 + 1) % 3
    end
    i += 1
  end
  state == 0
end

check("dfa 0", divisible_by_3("0"), true)
check("dfa 110 (=6)", divisible_by_3("110"), true)
check("dfa 111 (=7)", divisible_by_3("111"), false)
check("dfa 1001 (=9)", divisible_by_3("1001"), true)
check("dfa 1010 (=10)", divisible_by_3("1010"), false)

# ----- bracket balancing with a depth counter -----

def is_balanced(s)
  depth = 0
  i = 0
  while i < s.length
    c = s[i]
    if c == "("
      depth += 1
    elsif c == ")"
      depth -= 1
      return false if depth < 0
    end
    i += 1
  end
  depth == 0
end

check("balanced empty", is_balanced(""), true)
check("balanced simple", is_balanced("()"), true)
check("balanced nested", is_balanced("((()))"), true)
check("balanced mixed", is_balanced("(a(b)c)"), true)
check("unbalanced open", is_balanced("(()"), false)
check("unbalanced close", is_balanced("())"), false)
check("unbalanced order", is_balanced(")("), false)

# ----- triple-nested loop: count Pythagorean triples a<=b<=c<=n -----

def count_triples(n)
  count = 0
  a = 1
  while a <= n
    b = a
    while b <= n
      c = b
      while c <= n
        count += 1 if a * a + b * b == c * c
        c += 1
      end
      b += 1
    end
    a += 1
  end
  count
end

check("triples <=20", count_triples(20), 6)
check("triples <=5", count_triples(5), 1)

# ----- case/when dispatch driving a tiny accumulator machine -----

def run_ops(ops)
  acc = 0
  ops.each do |op|
    case op
    when 1
      acc += 10
    when 2
      acc -= 3
    when 3
      acc *= 2
    when 4, 5
      acc += 1
    else
      acc = 0
    end
  end
  acc
end

check("ops machine", run_ops([1, 1, 3, 2, 4, 5]), 39)  # ((0+10+10)*2)-3+1+1
check("ops reset", run_ops([1, 1, 9]), 0)              # else branch resets

# ----- done -----
if fails == 0
  puts "Ruby big self test 2 (recursion and control flow) passed"
end
exit(fails)
