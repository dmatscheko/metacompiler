# Python subset self test.
# The file runs top to bottom and ends with exit(fails[0]), so the
# metacompiler run exits with 0 exactly when everything works.

fails = [0]
lst0 = [3, 1, 4]

def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1

def check_true(name, got):
    if not got:
        print("FAIL", name, "expected a true value")
        fails[0] += 1

def add(a, b):
    return a + b

def fib(n):
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

def classify(n):
    if n < 0:
        return "negative"
    elif n == 0:
        return "zero"
    elif n < 10:
        return "small"
    else:
        return "big"

def total(items):
    s = 0
    for v in items:
        s += v
    return s

# arithmetic: / is true division, // floors, % is floor modulo
check("precedence", 1 + 2 * 3, 7)
check("true division", 7 / 2, 3.5)
check("floor division", 7 // 2, 3)
check("floor division negative", -7 // 2, -4)
check("floor modulo", -7 % 2, 1)
check("modulo", 7 % 3, 1)
check("unary minus", -(-5), 5)
check("float math", 0.5 * 4, 2)
check("call", add(20, 22), 42)

# if / elif / else
check("classify negative", classify(-1), "negative")
check("classify zero", classify(0), "zero")
check("classify small", classify(5), "small")
check("classify big", classify(50), "big")

# while, break, continue
n = 0
while n < 5:
    n += 1
check("while", n, 5)

odd = 0
k = 0
while True:
    k += 1
    if k > 100:
        break
    if k % 2 == 0:
        continue
    if k > 10:
        break
    odd += k
check("break continue", odd, 25)

# for over range and lists
s = 0
for i in range(1, 11):
    s += i
check("range for", s, 55)

r0 = 0
for i in range(4):
    r0 += i
check("range single arg", r0, 6)

lst = [3, 1, 4, 1, 5]
check("len", len(lst), 5)
check("index", lst[2], 4)
check("negative index", lst[-1], 5)
check("negative index 2", lst[-2], 1)
lst[1] = 10
check("element assign", lst[1], 10)
lst.append(9)
check("append", len(lst), 6)
check("appended", lst[-1], 9)
popped = lst.pop()
check("pop", popped, 9)
check("pop len", len(lst), 5)
check("sum list", total(lst), 23)

# nested loops with suites
grid = 0
for y in range(3):
    for x in range(3):
        if x == 2:
            break
        grid += 1
check("nested break", grid, 6)

# truthiness: empty lists are falsy, and/or return operands
empty = []
full = [1]
check_true("empty list falsy", not empty)
check_true("full list truthy", full)
check("or value", 0 or "x", "x")
check("and value", 5 and 7, 7)
check("or empty list", empty or "fallback", "fallback")
check_true("not zero", not 0)

# strings
name = "world"
check("concat", "hello " + name, "hello world")
check("string len", len(name), 5)
check("string index", name[0], "w")
check("string negative", name[-1], "d")
check_true("string compare", "apple" < "banana")

# recursion and a mutable counter across functions
# membership tests
check("in list", 1 in lst0, True)
check("not in list", 7 not in lst0, True)
check("in string", "ell" in "hello", True)
check("not in string", "z" not in "hello", True)

# f-strings
name = "world"
check("f-string", f"hi {name}!", "hi world!")
check("f-string expr", f"sum={1 + 2}", "sum=3")
check("f-string list", f"l={lst0}", "l=[3, 1, 4]")
check("f-string empty", f"", "")

check("fib", fib(10), 55)
check("fails so far", fails[0], 0)

if fails[0] == 0:
    print("Python subset self test passed")
exit(fails[0])
