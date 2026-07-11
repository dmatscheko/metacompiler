# Python subset self test - BIG 2: recursion and control flow.
#
# Theme: deep recursion and branching - factorial/gcd/fibonacci in both recursive
# and iterative form, exponentiation by squaring, the Ackermann function, the
# Collatz sequence, Towers of Hanoi (with the actual move list), memoized grid-path
# counting, mutual recursion (is_even/is_odd), a balanced-bracket checker driven by
# a stack, and two small finite state machines. The file runs top to bottom and ends
# with exit(fails[0]); the interpreter and the LLVM-IR compiler (both engines) must
# agree byte for byte.

fails = [0]


def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1


def check_true(name, got):
    if not got:
        print("FAIL", name, "expected a true value")
        fails[0] += 1


# ----- factorial: recursive and iterative -----

def fact_rec(n):
    if n <= 1:
        return 1
    return n * fact_rec(n - 1)


def fact_iter(n):
    acc = 1
    k = 2
    while k <= n:
        acc *= k
        k += 1
    return acc


check("fact_rec 0", fact_rec(0), 1)
check("fact_rec 5", fact_rec(5), 120)
check("fact_iter 6", fact_iter(6), 720)
check("fact agree 10", fact_rec(10), fact_iter(10))
check("fact 12", fact_rec(12), 479001600)


# ----- gcd (Euclid) recursive and iterative, plus lcm -----

def gcd_rec(a, b):
    if b == 0:
        return a
    return gcd_rec(b, a % b)


def gcd_iter(a, b):
    while b != 0:
        t = a % b
        a = b
        b = t
    return a


def lcm(a, b):
    return a // gcd_rec(a, b) * b


check("gcd 48 36", gcd_rec(48, 36), 12)
check("gcd 17 5", gcd_rec(17, 5), 1)
check("gcd agree", gcd_rec(1071, 462), gcd_iter(1071, 462))
check("gcd 462 1071", gcd_iter(462, 1071), 21)
check("lcm 4 6", lcm(4, 6), 12)
check("lcm 21 6", lcm(21, 6), 42)


# ----- fibonacci: naive recursion and a memoized version -----

fib_memo = {}


def fib_rec(n):
    if n < 2:
        return n
    return fib_rec(n - 1) + fib_rec(n - 2)


def fib_fast(n):
    if n < 2:
        return n
    if n in fib_memo:
        return fib_memo[n]
    r = fib_fast(n - 1) + fib_fast(n - 2)
    fib_memo[n] = r
    return r


check("fib_rec 10", fib_rec(10), 55)
check("fib_fast 20", fib_fast(20), 6765)
check("fib agree 15", fib_rec(15), fib_fast(15))
check("fib_fast 30", fib_fast(30), 832040)
check("memo filled", len(fib_memo) > 10, True)


# ----- exponentiation by squaring (there is no ** operator) -----

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


def ipow_rec(base, exp):
    if exp == 0:
        return 1
    half = ipow_rec(base, exp // 2)
    if exp % 2 == 0:
        return half * half
    return half * half * base


check("ipow 2^10", ipow(2, 10), 1024)
check("ipow 3^4", ipow(3, 4), 81)
check("ipow 5^0", ipow(5, 0), 1)
check("ipow_rec 2^16", ipow_rec(2, 16), 65536)
check("ipow agree", ipow(7, 5), ipow_rec(7, 5))


# ----- Ackermann (deeply recursive, kept small) -----

def ackermann(m, n):
    if m == 0:
        return n + 1
    if n == 0:
        return ackermann(m - 1, 1)
    return ackermann(m - 1, ackermann(m, n - 1))


check("ack 0 0", ackermann(0, 0), 1)
check("ack 2 2", ackermann(2, 2), 7)
check("ack 3 3", ackermann(3, 3), 61)
check("ack 2 4", ackermann(2, 4), 11)


# ----- Collatz sequence length -----

def collatz_len(n):
    steps = 0
    while n != 1:
        if n % 2 == 0:
            n = n // 2
        else:
            n = 3 * n + 1
        steps += 1
    return steps


check("collatz 1", collatz_len(1), 0)
check("collatz 6", collatz_len(6), 8)
check("collatz 27", collatz_len(27), 111)
peak = 0
peak_n = 0
for start in range(1, 20):
    c = collatz_len(start)
    if c > peak:
        peak = c
        peak_n = start
check("collatz peak under 20", peak_n, 18)
check("collatz peak value", peak, 20)


# ----- Towers of Hanoi: build the actual move list, then count -----

moves = []


def hanoi(n, src, dst, via):
    if n == 0:
        return
    hanoi(n - 1, src, via, dst)
    moves.append(f"{src}->{dst}")
    hanoi(n - 1, via, dst, src)


def hanoi_count(n):
    if n == 0:
        return 0
    return 2 * hanoi_count(n - 1) + 1


hanoi(3, "A", "C", "B")
check("hanoi move count", len(moves), 7)
check("hanoi first move", moves[0], "A->C")
check("hanoi last move", moves[6], "A->C")
check("hanoi middle move", moves[3], "A->C")
check("hanoi second move", moves[1], "A->B")

# the optimal move count for n disks is 2^n - 1
hanoi_ok = 0
for disks in range(1, 8):
    if hanoi_count(disks) == ipow(2, disks) - 1:
        hanoi_ok += 1
check("hanoi 2^n-1 rule", hanoi_ok, 7)


# ----- memoized grid-path counting (only right/down moves) -----

path_memo = {}


def count_paths(r, c):
    if r == 0 or c == 0:
        return 1
    key = f"{r},{c}"
    if key in path_memo:
        return path_memo[key]
    total = count_paths(r - 1, c) + count_paths(r, c - 1)
    path_memo[key] = total
    return total


check("paths 1x1", count_paths(1, 1), 2)
check("paths 2x2", count_paths(2, 2), 6)
check("paths 3x3", count_paths(3, 3), 20)
check("paths 4x4", count_paths(4, 4), 70)
# paths(r, c) equals the binomial coefficient C(r+c, r); C(8,4) = 70
check("paths symmetric", count_paths(2, 5), count_paths(5, 2))


# ----- mutual recursion between two top-level functions -----

def is_even(n):
    if n == 0:
        return True
    return is_odd(n - 1)


def is_odd(n):
    if n == 0:
        return False
    return is_even(n - 1)


check_true("10 is even", is_even(10))
check_true("7 is odd", is_odd(7))
check("0 not odd", is_odd(0), False)
evens_found = 0
for n in range(20):
    if is_even(n):
        evens_found += 1
check("count evens 0..19", evens_found, 10)


# ----- a balanced-bracket checker driven by an explicit stack -----

def matches(open_ch, close_ch):
    if open_ch == "(":
        return close_ch == ")"
    if open_ch == "[":
        return close_ch == "]"
    if open_ch == "{":
        return close_ch == "}"
    return False


def is_balanced(s):
    stack = []
    for ch in s:
        if ch == "(" or ch == "[" or ch == "{":
            stack.append(ch)
        elif ch == ")" or ch == "]" or ch == "}":
            if len(stack) == 0:
                return False
            top = stack.pop()
            if not matches(top, ch):
                return False
    return len(stack) == 0


check_true("balanced simple", is_balanced("()"))
check_true("balanced nested", is_balanced("([{}])"))
check_true("balanced mixed", is_balanced("a(b[c]d){e}"))
check("unbalanced open", is_balanced("(()"), False)
check("unbalanced order", is_balanced("([)]"), False)
check("unbalanced close", is_balanced("())"), False)
check_true("balanced empty", is_balanced(""))


# ----- FSM 1: accept binary strings whose value is divisible by 3 -----

def div_by_3(bits):
    state = 0
    for ch in bits:
        d = 0
        if ch == "1":
            d = 1
        state = (state * 2 + d) % 3
    return state == 0


check_true("0 div3", div_by_3("0"))
check_true("110 div3 (6)", div_by_3("110"))
check_true("1001 div3 (9)", div_by_3("1001"))
check("101 not div3 (5)", div_by_3("101"), False)
check("111 not div3 (7)", div_by_3("111"), False)
div3_count = 0
for v in range(16):
    # build the binary string of v by repeated division
    bits = ""
    x = v
    if x == 0:
        bits = "0"
    while x > 0:
        bits = f"{x % 2}" + bits
        x = x // 2
    if div_by_3(bits):
        div3_count += 1
check("div3 count 0..15", div3_count, 6)


# ----- FSM 2: a tiny turnstile state machine -----

def turnstile(events):
    state = "locked"
    opened = 0
    for e in events:
        if state == "locked":
            if e == "coin":
                state = "unlocked"
        else:
            if e == "push":
                opened += 1
                state = "locked"
    return opened


check("turnstile one pass", turnstile(["coin", "push"]), 1)
check("turnstile blocked push", turnstile(["push", "push"]), 0)
check("turnstile double coin", turnstile(["coin", "coin", "push"]), 1)
check("turnstile two passes", turnstile(["coin", "push", "coin", "push"]), 2)


check("no failures", fails[0], 0)
if fails[0] == 0:
    print("Python big-2 (recursion and control flow) self test passed")
exit(fails[0])
