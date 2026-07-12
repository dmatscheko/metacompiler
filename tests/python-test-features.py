# Fast feature-matrix test for the Python interpreter (python-interpreter.abnf) and
# the LLVM-IR compiler (python-to-llvm-ir.abnf). It replaces the four algorithm-
# themed python-test-big-* stress tests: instead of large loops (five sorting
# routines, Ackermann, sieves, matrix products) every implemented construct is
# exercised with the SMALLEST program that can prove it works - loops run 0, 1, 3
# or 4 times, recursion stays below depth 6. Classes, tuples-as-values, closures/
# lambdas/nested def, ** and the bit operators are recognized but not implemented
# (see python-test-recognize.py) and stay out. A failed check prints its id (so a
# diff pinpoints it) and the file ends with exit(fails[0]); exit 0 and
# byte-identical output on all four legs (interpreter/compiler x goja/-frozen)
# mean everything passed.

fails = [0]
checks = [0]

def check(name, got, want):
    checks[0] += 1
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1

def check_true(name, got):
    checks[0] += 1
    if not got:
        print("FAIL", name, "expected a true value")
        fails[0] += 1

# ----- numbers, arithmetic, precedence -----
check("arith-precedence", 2 + 3 * 4, 14)
check("arith-paren", (2 + 3) * 4, 20)
check("arith-unary-minus", -3 + 5, 2)
check("arith-double-neg", -(-5), 5)
check("arith-true-div", 7 / 2, 3.5)
check("arith-floor-div", 7 // 2, 3)
check("arith-floor-div-neg", -7 // 2, -4)
check("arith-mod", 7 % 3, 1)
check("arith-mod-neg", -7 % 2, 1)
check("arith-float", 0.5 * 4, 2)
check_true("arith-float-imprecision", 0.1 + 0.2 != 0.3)
check("arith-chain", 20 - 5 - 3, 12)

x = 5
x += 3
x -= 2
x *= 4
x /= 6
check("arith-compound", x, 4)

check("ternary-true", 1 if True else 2, 1)
check("ternary-false", "a" if 0 else "b", "b")
n = 7
check("ternary-nested", "neg" if n < 0 else ("zero" if n == 0 else "pos"), "pos")

# ----- strings -----
name = "world"
check("str-concat", "hello " + name, "hello world")
check("str-len", len(name), 5)
check("str-len-empty", len(""), 0)
check("str-index", name[0], "w")
check("str-index-neg", name[-1], "d")
check("str-slice", "hello"[1:3], "el")
check("str-slice-open-lo", "hello"[:2], "he")
check("str-slice-neg", "hello"[-3:], "llo")
check("str-slice-clamped", len("hello"[1:100]), 4)
check("str-in", "ell" in "hello", True)
check("str-not-in", "z" not in "hello", True)
check_true("str-compare", "apple" < "banana")
check("str-quotes", 'raw text', "raw text")
check("str-escape-tab", len("a\tb"), 3)
check("str-escape-newline", "a\nb"[1], "\n")
check("str-escape-backslash", len("\\"), 1)
check("str-escape-quote", len("\""), 1)
check("str-unicode-len", len("héllo"), 5)
check("str-unicode-index", "héllo"[1], "é")
check("str-unicode-slice", "héllo"[:2], "hé")
check("f-string", f"hi {name}!", "hi world!")
check("f-string-expr", f"sum={1 + 2}", "sum=3")
check("f-string-empty", f"", "")

cs = ""
for ch in "abc":
    cs += ch + "."
check("for-over-string", cs, "a.b.c.")

td = """ab
cd"""
check("str-triple-quoted", len(td), 5)

# ----- equality, identity, logic -----
check("eq-num", 3 == 3, True)
check("ne", 3 != 4, True)
check_true("cmp-ops", 2 < 3 and 3 > 2 and 2 <= 2 and 3 >= 3)
nothing = None
check("is-none", nothing is None, True)
check("is-not-none", 5 is not None, True)
shared = [1, 2]
alias = shared
check("is-identity", alias is shared, True)
check("is-not-identity", [1] is not [1], True)
check("or-value", 0 or "x", "x")
check("and-value", 5 and 7, 7)
check("or-empty-list", [] or "fb", "fb")
check_true("not-zero", not 0)
check_true("empty-str-falsy", not "")
check_true("empty-list-falsy", not [])
check_true("empty-dict-falsy", not {})
check_true("full-list-truthy", [1])

hits = [0]

def bump():
    hits[0] += 1
    return True

t1 = False and bump()
t2 = True and bump()
t3 = True or bump()
check("logic-short-circuit", hits[0], 1)
check_true("logic-results", t2 and t3 and not t1)

# ----- control flow: if / while / for / break / continue -----
def classify(n):
    if n < 0:
        return "negative"
    elif n == 0:
        return "zero"
    else:
        return "positive"

check("if-neg", classify(-4), "negative")
check("if-zero", classify(0), "zero")
check("if-pos", classify(9), "positive")

flag = 0
if True:
    pass
else:
    flag = 1
check("pass-stmt", flag, 0)

w = 0
while w > 0:
    w -= 1
check("while-zero", w, 0)

w3 = 0
while w3 < 3:
    w3 += 1
check("while-three", w3, 3)

dw = 0
while True:
    dw += 1
    break
check("while-true-once", dw, 1)

fs = 0
for i in range(1, 4):
    fs += i
check("for-range-two-args", fs, 6)

f1 = 0
for i in range(4):
    f1 += i
check("for-range-one-arg", f1, 6)

fz = 0
for i in range(0):
    fz += 1
check("for-range-zero", fz, 0)

brk = ""
for i in range(9):
    if i == 2:
        break
    brk += f"{i}"
check("for-break", brk, "01")

cont = ""
for i in range(4):
    if i % 2 == 1:
        continue
    cont += f"{i}"
check("for-continue", cont, "02")

nested = ""
for oi in range(2):
    for ii in range(3):
        if ii == 1:
            break
        nested += f"{oi}{ii}"
check("nested-break", nested, "0010")

lsum = 0
for v in [4, 5, 6]:
    lsum += v
check("for-over-list", lsum, 15)

# ----- functions, recursion, multiple assignment -----
def add(a, b):
    return a + b

check("fn-args", add(2, 3), 5)

def sign(n):
    if n < 0:
        return -1
    return 1

check("fn-early-return", sign(-8), -1)
check("fn-fallthrough", sign(3), 1)

def fib(n):
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

check("fn-recursion", fib(6), 8)

def is_even(n):
    if n == 0:
        return True
    return is_odd(n - 1)

def is_odd(n):
    if n == 0:
        return False
    return is_even(n - 1)

check_true("fn-mutual-recursion", is_even(4) and is_odd(5))

def describe(v):
    '''A docstring: parsed and discarded.'''
    return "labelled"

check("fn-docstring", describe(1), "labelled")

a, b = 1, 2
check_true("multi-assign", a == 1 and b == 2)
a, b = b, a
check_true("multi-swap", a == 2 and b == 1)
p, q, r = 10, 20, 30
check("multi-triple", p + q + r, 60)
u1, u2 = [4, 5]
check_true("multi-unpack-list", u1 == 4 and u2 == 5)
sw = [7, 8]
sw[0], sw[1] = sw[1], sw[0]
check("multi-swap-indexed", f"{sw}", "[8, 7]")

count: int = 10
count += 5
check("annotated-assign", count, 15)
assert count == 15
assert count == 15, "count must be 15"
check("assert-passed", count, 15)

# ----- lists -----
lst = [3, 1, 4]
check("list-len", len(lst), 3)
check("list-index", lst[1], 1)
check("list-index-neg", lst[-1], 4)
lst[1] = 10
check("list-assign", lst[1], 10)
lst.append(9)
check_true("list-append", len(lst) == 4 and lst[-1] == 9)
check("list-pop", lst.pop(), 9)
check("list-pop-len", len(lst), 3)
check("list-in", 4 in lst, True)
check("list-not-in", 7 not in lst, True)
lcopy = list(lst)
lcopy.append(99)
check_true("list-copy-independent", len(lcopy) == 4 and len(lst) == 3)
check("list-nested", [[1, 2], [3]][0][1], 2)
check("list-render", f"{lst}", "[3, 10, 4]")

sl = [0, 1, 2, 3, 4]
check("list-slice", f"{sl[1:4]}", "[1, 2, 3]")
check("list-slice-open-hi", len(sl[2:]), 3)
check("list-slice-neg-chain", sl[-2:][0], 3)

squares = [v * v for v in range(4)]
check("comp", f"{squares}", "[0, 1, 4, 9]")
evens = [v for v in range(5) if v % 2 == 0]
check("comp-if", f"{evens}", "[0, 2, 4]")
check("comp-over-string", f"{[c for c in 'xy']}", "['x', 'y']")
grid = [[0 for c in range(2)] for r in range(2)]
grid[0][0] = 7
check_true("comp-nested-fresh-rows", grid[0][0] == 7 and grid[1][0] == 0)
leak = 99
check("comp-len", len([leak for leak in range(3)]), 3)
check("comp-no-leak", leak, 99)

# ----- dicts -----
ages = {"alice": 30, "bob": 25}
check("dict-get", ages["alice"], 30)
ages["carol"] = 35
check("dict-set-new", ages["carol"], 35)
ages["bob"] += 1
check("dict-aug-assign", ages["bob"], 26)
check("dict-len", len(ages), 3)
check("dict-in", "alice" in ages, True)
check("dict-not-in", "dave" not in ages, True)
check("dict-get-method", ages.get("alice"), 30)
check("dict-get-default", ages.get("dave", -1), -1)
check("dict-keys", f"{list(ages.keys())}", "['alice', 'bob', 'carol']")
check("dict-values", list(ages.values())[2], 35)
its = ages.items()
check_true("dict-items", len(its) == 3 and its[0][0] == "alice" and its[0][1] == 30)

ksum = ""
for k in ages:
    ksum += k[0]
check("dict-iterate-order", ksum, "abc")

counts = {}
for w in ["a", "b", "a"]:
    counts[w] = counts.get(w, 0) + 1
check_true("dict-counter-idiom", counts["a"] == 2 and counts["b"] == 1)
check("dict-render", f"{counts}", "{'a': 2, 'b': 1}")

dsq = {v: v * v for v in range(3)}
check("dict-comp", f"{dsq}", "{0: 0, 1: 1, 2: 4}")
dodd = {v: v for v in range(4) if v % 2 == 1}
check_true("dict-comp-if", 1 in dodd and 2 not in dodd and len(dodd) == 2)

# ----- exceptions: raise / except / else / finally / control flow -----
exc_log = [""]
exc_num = 0
try:
    raise Exception("boom", 7)
except Exception as exc_e:
    exc_log[0] += "c" + exc_e.args[0]
    exc_num = exc_e.args[1]
finally:
    exc_log[0] += "f"
check("exception-object-args", exc_log[0], "cboomf")
check("exception-args-index", exc_num, 7)

verr = ValueError("nope")
verr.note = "extra"
check("attr-read-write", verr.note + verr.args[0], "extranope")

def risky(n):
    if n > 3:
        raise n
    return n * 2

log = [""]
try:
    log[0] += "t"
    raise "boom"
    log[0] += "X"
except Exception as e:
    log[0] += "c" + e
finally:
    log[0] += "f"
check("try-raise-catch-finally", log[0], "tcboomf")

quiet = [""]
try:
    quiet[0] += "t"
finally:
    quiet[0] += "f"
check("try-no-raise", quiet[0], "tf")

caught = [-1]
try:
    risky(5)
    check_true("unreachable-after-raise", False)
except Exception as e:
    caught[0] = e
check("raise-unwinds-calls", caught[0], 5)
check("raise-untaken-path", risky(2), 4)

def with_else(n):
    tag = 0
    try:
        if n < 0:
            raise n
        tag = 1
    except Exception as e:
        tag = 2
    else:
        tag += 10
    return tag

check("try-else-runs", with_else(7), 11)
check("try-else-skipped", with_else(-1), 2)

def ret_across_try():
    try:
        return "from-try"
    finally:
        hits[0] += 1

before = hits[0]
check("return-across-try", ret_across_try(), "from-try")
check("finally-ran-on-return", hits[0], before + 1)

def ret_in_finally():
    try:
        return 1
    finally:
        return 2

check("return-in-finally", ret_in_finally(), 2)

def finally_overrides_raise():
    try:
        try:
            raise 1
        finally:
            return "inner-finally-wins"
    except Exception as e:
        return "outer-caught"

check("finally-overrides-raise", finally_overrides_raise(), "inner-finally-wins")

def break_in_finally():
    i = 0
    while True:
        i += 1
        try:
            pass
        finally:
            break
    return i

check("break-in-finally", break_in_finally(), 1)

def continue_across_try():
    total = 0
    for i in range(4):
        try:
            if i == 2:
                continue
            total += i
        finally:
            pass
    return total

check("continue-across-try", continue_across_try(), 4)

def rethrow():
    try:
        try:
            raise "deep"
        except Exception as e:
            raise e + "er"
    except Exception as e2:
        return e2

check("rethrow", rethrow(), "deeper")

# ----- lambdas and lexical closures -----
double = lambda k: k * 2
check("lambda-direct", double(7), 14)
check("lambda-arg", (lambda a, b: a + b)(3, 4), 7)

def make_counter(start):
    def bump(step):
        return start + step
    return bump

check("closure-nested-def", make_counter(10)(5), 15)
check("closure-lambda", (lambda n: lambda k: k + n)(3)(4), 7)

# ----- global and nonlocal declarations -----
hits = 0

def record():
    global hits
    hits = hits + 1

record()
record()
check("global-decl", hits, 2)

def accumulator():
    total = 0
    def add(n):
        nonlocal total
        total = total + n
        return total
    return add

acc = accumulator()
acc(5)
check("nonlocal-decl", acc(7), 12)

# ----- dynamic typing: a variable may change its type -----
dyn = 1
dyn = "now a string"
dyn = [dyn]
check("dynamic-retype", dyn[0], "now a string")

# ----- everything combined in one small pipeline (3-element data flow) -----
def transform(items):
    out = []
    for n in items:
        try:
            if n < 0:
                raise "neg"
            if n % 2 == 0:
                out.append(f"e{n}")
            else:
                out.append(f"o{n}")
        except Exception as e:
            out.append("x")
    return out

check("combined-pipeline", f"{transform([1, 2, -3])}", "['o1', 'e2', 'x']")

print(f"features: {checks[0]} checks, {fails[0]} failures")
exit(fails[0])
