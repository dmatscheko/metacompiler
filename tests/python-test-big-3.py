# Python subset self test - BIG 3: number theory and integer math.
#
# Theme: primes and factorization - a Sieve of Eratosthenes, trial-division
# primality, prime factorization into a dict and reconstruction, divisor sums and
# perfect numbers, an integer square root by binary search, Pascal's triangle,
# base conversion to and from digit strings (there is no ** / str() / int()), digit
# sums, Armstrong numbers and fraction reduction. The file runs top to bottom and
# ends with exit(fails[0]); the interpreter and the LLVM-IR compiler (both engines)
# must agree byte for byte.

fails = [0]


def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1


def check_true(name, got):
    if not got:
        print("FAIL", name, "expected a true value")
        fails[0] += 1


def render(a):
    return f"{a}"


# integer power by squaring (no ** operator in the subset)
def ipow(base, exp):
    result = 1
    b = base
    e = exp
    while e > 0:
        if e % 2 == 1:
            result *= b
        b *= b
        e = e // 2
    return result


# ----- primality by trial division -----

def is_prime(n):
    if n < 2:
        return False
    if n < 4:
        return True
    if n % 2 == 0:
        return False
    d = 3
    while d * d <= n:
        if n % d == 0:
            return False
        d += 2
    return True


check("2 prime", is_prime(2), True)
check("3 prime", is_prime(3), True)
check("4 not prime", is_prime(4), False)
check("1 not prime", is_prime(1), False)
check("17 prime", is_prime(17), True)
check("221 not prime", is_prime(221), False)
check("worst 9973 prime", is_prime(9973), True)

primes_below_30 = [n for n in range(30) if is_prime(n)]
check("primes below 30", render(primes_below_30), "[2, 3, 5, 7, 11, 13, 17, 19, 23, 29]")


# ----- Sieve of Eratosthenes -----

def sieve(limit):
    flags = [True for i in range(limit + 1)]
    flags[0] = False
    if limit >= 1:
        flags[1] = False
    p = 2
    while p * p <= limit:
        if flags[p]:
            multiple = p * p
            while multiple <= limit:
                flags[multiple] = False
                multiple += p
        p += 1
    return [i for i in range(limit + 1) if flags[i]]


sieved = sieve(50)
check("sieve count to 50", len(sieved), 15)
check("sieve first", sieved[0], 2)
check("sieve last", sieved[-1], 47)
check("sieve render to 30", render(sieve(30)), "[2, 3, 5, 7, 11, 13, 17, 19, 23, 29]")

# the sieve and trial-division must agree on every value up to the limit
disagree = 0
for n in range(51):
    in_sieve = n in sieved
    if in_sieve != is_prime(n):
        disagree += 1
check("sieve agrees with trial", disagree, 0)


# ----- prime factorization into a dict, and reconstruction -----

def factorize(n):
    factors = {}
    d = 2
    while d * d <= n:
        while n % d == 0:
            factors[d] = factors.get(d, 0) + 1
            n = n // d
        d += 1
    if n > 1:
        factors[n] = factors.get(n, 0) + 1
    return factors


def reconstruct(factors):
    product = 1
    for pair in factors.items():
        product *= ipow(pair[0], pair[1])
    return product


check("factorize 12", render(factorize(12)), "{2: 2, 3: 1}")
check("factorize 360", render(factorize(360)), "{2: 3, 3: 2, 5: 1}")
check("factorize prime", render(factorize(97)), "{97: 1}")
check("factorize 1", len(factorize(1)), 0)

rebuilt_ok = 0
for n in range(2, 60):
    if reconstruct(factorize(n)) == n:
        rebuilt_ok += 1
check("reconstruct round-trips", rebuilt_ok, 58)

# the number of prime factors (with multiplicity) of 2^k is k
def factor_count(factors):
    total = 0
    for v in factors.values():
        total += v
    return total


check("factor count 1024", factor_count(factorize(1024)), 10)
check("factor count 360", factor_count(factorize(360)), 6)


# ----- divisor sums and perfect numbers -----

def proper_divisor_sum(n):
    if n < 2:
        return 0
    total = 1
    d = 2
    while d * d <= n:
        if n % d == 0:
            total += d
            other = n // d
            if other != d:
                total += other
        d += 1
    return total


def is_perfect(n):
    return proper_divisor_sum(n) == n


check("divisor sum 12", proper_divisor_sum(12), 16)
check("divisor sum 28", proper_divisor_sum(28), 28)
check_true("6 perfect", is_perfect(6))
check_true("28 perfect", is_perfect(28))
check("12 not perfect", is_perfect(12), False)

perfects = [n for n in range(1, 500) if is_perfect(n)]
check("perfects under 500", render(perfects), "[6, 28, 496]")


# ----- integer square root by binary search -----

def isqrt(n):
    if n < 2:
        return n
    lo = 1
    hi = n
    while lo <= hi:
        mid = (lo + hi) // 2
        if mid * mid <= n:
            lo = mid + 1
        else:
            hi = mid - 1
    return hi


check("isqrt 0", isqrt(0), 0)
check("isqrt 15", isqrt(15), 3)
check("isqrt 16", isqrt(16), 4)
check("isqrt 24", isqrt(24), 4)
check("isqrt 10000", isqrt(10000), 100)
isqrt_ok = 0
for n in range(0, 200):
    r = isqrt(n)
    if r * r <= n and (r + 1) * (r + 1) > n:
        isqrt_ok += 1
check("isqrt invariant", isqrt_ok, 200)


# ----- Pascal's triangle -----

def pascal_row(n):
    row = [1]
    for k in range(n):
        row.append(row[k] * (n - k) // (k + 1))
    return row


check("pascal row 0", render(pascal_row(0)), "[1]")
check("pascal row 4", render(pascal_row(4)), "[1, 4, 6, 4, 1]")
check("pascal row 6", render(pascal_row(6)), "[1, 6, 15, 20, 15, 6, 1]")
# each row sums to 2^n
row_sum_ok = 0
for n in range(10):
    row = pascal_row(n)
    s = 0
    for v in row:
        s += v
    if s == ipow(2, n):
        row_sum_ok += 1
check("pascal row sums", row_sum_ok, 10)


# ----- base conversion to and from a digit string -----

DIGITS = "0123456789abcdef"


def idx_in(s, ch):
    for i in range(len(s)):
        if s[i] == ch:
            return i
    return -1


def to_base(n, base):
    if n == 0:
        return "0"
    out = ""
    x = n
    while x > 0:
        out = DIGITS[x % base] + out
        x = x // base
    return out


def from_base(s, base):
    val = 0
    for ch in s:
        val = val * base + idx_in(DIGITS, ch)
    return val


check("13 to binary", to_base(13, 2), "1101")
check("255 to hex", to_base(255, 16), "ff")
check("0 to binary", to_base(0, 2), "0")
check("10 to base 3", to_base(10, 3), "101")
check("parse 1101 binary", from_base("1101", 2), 13)
check("parse ff hex", from_base("ff", 16), 255)

roundtrip_ok = 0
for n in range(0, 100):
    if from_base(to_base(n, 2), 2) == n and from_base(to_base(n, 16), 16) == n:
        roundtrip_ok += 1
check("base round-trip", roundtrip_ok, 100)


# ----- digit sums and Armstrong numbers -----

def digit_sum(n):
    s = 0
    x = n
    while x > 0:
        s += x % 10
        x = x // 10
    return s


def num_digits(n):
    if n == 0:
        return 1
    count = 0
    x = n
    while x > 0:
        count += 1
        x = x // 10
    return count


def is_armstrong(n):
    p = num_digits(n)
    total = 0
    x = n
    while x > 0:
        total += ipow(x % 10, p)
        x = x // 10
    return total == n


check("digit sum 12345", digit_sum(12345), 15)
check("digit sum 9999", digit_sum(9999), 36)
check("digits of 12345", num_digits(12345), 5)
check_true("153 armstrong", is_armstrong(153))
check_true("9474 armstrong", is_armstrong(9474))
check("154 not armstrong", is_armstrong(154), False)
armstrongs = [n for n in range(100, 1000) if is_armstrong(n)]
check("3-digit armstrongs", render(armstrongs), "[153, 370, 371, 407]")


# ----- fraction reduction via gcd -----

def gcd(a, b):
    while b != 0:
        t = a % b
        a = b
        b = t
    return a


def reduce_fraction(num, den):
    g = gcd(num, den)
    return [num // g, den // g]


check("reduce 6/8", render(reduce_fraction(6, 8)), "[3, 4]")
check("reduce 100/25", render(reduce_fraction(100, 25)), "[4, 1]")
check("reduce 17/5", render(reduce_fraction(17, 5)), "[17, 5]")

# sum of the first n odd numbers is n squared
odd_sum_ok = 0
for n in range(1, 20):
    s = 0
    for k in range(n):
        s += 2 * k + 1
    if s == n * n:
        odd_sum_ok += 1
check("odd sums are squares", odd_sum_ok, 19)


check("no failures", fails[0], 0)
if fails[0] == 0:
    print("Python big-3 (number theory) self test passed")
exit(fails[0])
