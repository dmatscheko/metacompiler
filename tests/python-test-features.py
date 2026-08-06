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

# float is a TYPE of its own, not just a value: it prints with a point, `/`
# always makes one, and `//` / `%` / `**` keep the int. The full window of the
# renderer and the dict-key rule are section 26 of tests/python-test-full.py.
check("float-type", str(type(1.0)), "<class 'float'>")
check("int-type", str(type(1)), "<class 'int'>")
check("float-repr", str(1.0), "1.0")
check("float-repr-exp", str(1e16), "1e+16")
check("float-repr-small", str(1e-5), "1e-05")
check("float-true-div", str(4 / 2), "2.0")
check("float-floor-div-int", str(7 // 2), "3")
check("float-floor-div-flo", str(7.5 // 2), "3.0")
check("float-pow-int", str(2 ** 3), "8")
check("float-pow-neg", str(2 ** -1), "0.5")
check("float-promote", str(1 + 2.0), "3.0")
check("float-abs", str(abs(-1.5)), "1.5")
check("float-eq-int", 1.0 == 1, True)
check("float-in-list", str([1.0, 2]), "[1.0, 2]")
check("float-dict-key", len({1.0: "a", 1: "b"}), 1)

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

# ----- default parameters and *args -----
def greet(who, greeting="hi"):
    return greeting + " " + who

check("param-default", greet("x"), "hi x")
check("param-override", greet("x", "yo"), "yo x")

snap = 5
capture = lambda v=snap: v
snap = 6
check("default-def-time", capture(), 5)

def total(*nums):
    t = 0
    for n in nums:
        t = t + n
    return t

check("varargs", total(1, 2, 3, 4), 10)
check("varargs-empty", total(), 0)

def mixed(base, *extra):
    return base + len(extra)

check("varargs-mixed", mixed(10, "a", "b"), 12)

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

# CPython's SETUP_ANNOTATIONS: a class body that CONTAINS an annotation gets its
# __annotations__ dict before any of its statements run, so a class whose only
# annotation sits in a branch the body never takes still has an EMPTY one.
# Two defects met here, both in the compiled half only:
#  - the dict was created at the annotation SITE, i.e. inside the branch, so
#    AnnUnreached.__annotations__ raised AttributeError;
#  - the class fill's "did the body bind this name" guard was a CHAIN WALK, so
#    the name was found in the MODULE scope instead and the module's own
#    annotations dict was copied onto the class - the check below then read
#    {'mod_ann': 'int'} where python3 and the interpreter read {}.
mod_ann: int = 1

class AnnReached:
    ar: int = 1

class AnnUnreached:
    if False:
        au: int = 1

class AnnMixed:
    if False:
        am_no: int = 1
    am_yes: int = 2

check_true("class-annotations-reached", len(AnnReached.__annotations__) == 1 and "ar" in AnnReached.__annotations__)
check_true("class-annotations-unreached", len(AnnUnreached.__annotations__) == 0)
# The THIRD shape, and the one that CRASHED: an unreached annotation FOLLOWED by a
# reached one. The compiler answered "is there an __annotations__ dict in this
# scope" with a static flag, and the flag was STICKY - the unreached `am_no` set
# it, so `am_yes` skipped the js_scope_decl and the dict was never created. On its
# own that class body DIES with "variable not defined: __annotations__" in both
# compiled halves; here, where the module above has an annotation of its own, the
# js_scope_get walks the chain and finds the MODULE's dict instead - so the class
# gets {} and the module's annotations are silently polluted with am_yes. That
# quieter shape is the one this check sees. It is js_scope_has at run time now,
# which is the question abnf/jsrt.go's js_pyannot always asked. python3 says
# {'am_yes': <class 'int'>}.
check_true("class-annotations-mixed", len(AnnMixed.__annotations__) == 1 and "am_yes" in AnnMixed.__annotations__ and "am_no" not in AnnMixed.__annotations__)

# Integer literals past 2**53. The box is chosen on the literal TEXT: the decimal
# predicate used to go through parseFloat (which rounds 9007199254740993 down to
# exactly 2**53 and answered "not big"), and the 0x / 0o / 0b forms were never
# boxed at all. Both halves agreed, so only python3 could see it.
check("lit-dec-2p53p1", str(9007199254740993), "9007199254740993")
check("lit-hex-2p53p1", str(0x20000000000001), "9007199254740993")
check("lit-oct-2p53p1", str(0o400000000000000001), "9007199254740993")
check("lit-bin-2p53p1", str(0b100000000000000000000000000000000000000000000000000001), "9007199254740993")
check("lit-hex-int64max", str(0x7fffffffffffffff), "9223372036854775807")
check("lit-hex-uint64max", str(0xffffffffffffffff), "18446744073709551615")
check("lit-neg-2p53p1", str(-9007199254740993), "-9007199254740993")
# 2**53 is exact and must NOT be boxed; small radix literals must stay plain.
check("lit-2p53-exact", str(9007199254740992) + " " + str(0x20000000000000), "9007199254740992 9007199254740992")
check("lit-small-radix", str(0xff) + " " + str(0o17) + " " + str(0b1011), "255 15 11")


# ----- str's method library (docs/todo.md 1.5) -----
# Every operand is read out of a list so the constant folder cannot answer these
# at compile time. The semantics, not just the names, are what these pin: the two
# split algorithms, strip's character SET, find-vs-index, and title's word rule.
SS = ["  Hello World  ", "a,b,,c", "abc", " a  b ", "they're bill's 12ab",
      "xyxhixy", "-42", "a\nb\r\nc", "Ab Cd", "aaa", "ABC", "café", "MiXeD"]
check("str-upper", SS[2].upper(), "ABC")
check("str-lower", SS[10].lower(), "abc")
check("str-upper-latin1", SS[11].upper(), "CAFÉ")
check("str-casefold", SS[10].casefold(), "abc")
# split() with NO argument splits on runs of whitespace and drops the empties;
# split(" ") does neither. Two algorithms, not one with a default.
check("str-split-none", str(SS[3].split()), "['a', 'b']")
check("str-split-space", str(SS[3].split(" ")), "['', 'a', '', 'b', '']")
check("str-split-sep", str(SS[1].split(",")), "['a', 'b', '', 'c']")
check("str-split-max", str(SS[1].split(",", 1)), "['a', 'b,,c']")
check("str-rsplit-max", str(SS[1].rsplit(",", 1)), "['a,b,', 'c']")
check("str-split-none-max", str(SS[3].split(None, 1)), "['a', 'b ']")
check("str-rsplit-none-max", str(SS[3].rsplit(None, 1)), "[' a', 'b']")
check("str-split-empty", str("".split(",")), "['']")
check("str-split-empty-none", str("".split()), "[]")
# strip(chars) is a SET of characters, not a prefix.
check("str-strip-ws", SS[0].strip(), "Hello World")
check("str-lstrip-set", SS[5].lstrip("xy"), "hixy")
check("str-rstrip-set", SS[5].rstrip("xy"), "xyxhi")
check("str-strip-set", "abcba".strip("ab"), "c")
check("str-strip-set-all", "abab".strip("ab"), "")
check("str-join", "|".join(["a", "b", "c"]), "a|b|c")
check("str-join-empty", "|".join([]), "")
check("str-join-str", "-".join(SS[2]), "a-b-c")
check("str-replace", SS[9].replace("a", "b", 2), "bba")
check("str-replace-empty-pat", SS[2].replace("", "-"), "-a-b-c-")
check("str-replace-empty-pat-max", SS[2].replace("", "-", 2), "-a-bc")
check("str-replace-drop", SS[9].replace("a", ""), "")
# find answers -1 where index raises; the bounds follow CPython's ADJUST_INDICES,
# so a start past the end is -1 even for the empty needle.
check("str-find", SS[2].find("c"), 2)
check("str-find-miss", SS[2].find("z"), -1)
check("str-find-bounded", SS[2].find("c", 0, 2), -1)
check("str-find-empty-at-end", SS[2].find("", 3), 3)
check("str-find-empty-past-end", SS[2].find("", 4), -1)
check("str-find-neg-start", SS[2].find("c", -1), 2)
check("str-rfind", "abcabc".rfind("b"), 4)
check("str-rfind-empty", SS[2].rfind(""), 3)
check("str-index", SS[2].index("b"), 1)
check("str-rindex", "abcabc".rindex("a"), 3)
check("str-count", "aaa".count("aa"), 1)
check("str-count-empty", SS[2].count(""), 4)
check("str-count-bounded", "aaa".count("a", 1), 2)
check_true("str-startswith", SS[2].startswith("ab"))
check_true("str-startswith-tuple", SS[2].startswith(("x", "a")))
check_true("str-endswith-bounded", SS[2].endswith("bc", 0, 3))
check_true("str-not-endswith-bounded", not SS[2].endswith("bc", 0, 2))
# title's word boundary is "the previous character was not cased".
check("str-title", SS[4].title(), "They'Re Bill'S 12Ab")
check("str-capitalize", SS[0].capitalize(), "  hello world  ")
check("str-swapcase", SS[12].swapcase(), "mIxEd")
check_true("str-istitle", SS[8].istitle())
check_true("str-not-istitle", not SS[12].istitle())
check_true("str-isdigit", "123".isdigit())
check_true("str-isdigit-empty", not "".isdigit())
check_true("str-isalpha", SS[2].isalpha())
check_true("str-isalpha-latin1", SS[11].isalpha())
check_true("str-isalnum", "ab1".isalnum())
check_true("str-isspace", "  ".isspace())
check_true("str-islower", "ab1".islower())
check_true("str-isupper", "AB1".isupper())
check_true("str-islower-uncased-only", not "123".islower())
check_true("str-isascii", SS[2].isascii())
check_true("str-not-isascii", not SS[11].isascii())
# center puts the ODD extra pad on the left only when the width is odd too.
check("str-center-odd", SS[2].center(7, "*"), "**abc**")
check("str-center-even", "ab".center(7, "*"), "***ab**")
check("str-center-narrow", SS[2].center(2), "abc")
check("str-ljust", SS[2].ljust(5, "."), "abc..")
check("str-rjust", SS[2].rjust(5, "."), "..abc")
check("str-zfill", SS[6].zfill(5), "-0042")
check("str-zfill-narrow", SS[6].zfill(1), "-42")
check("str-zfill-empty", "".zfill(3), "000")
# splitlines: \r\n is ONE break and the boundary set is wider than \n.
check("str-splitlines", str(SS[7].splitlines()), "['a', 'b', 'c']")
check("str-splitlines-keep", "|".join("a\nb\n".splitlines(True)).replace("\n", "N"), "aN|bN")
check("str-splitlines-empty", str("".splitlines()), "[]")
check("str-partition", str(list(SS[2].partition("b"))), "['a', 'b', 'c']")
check("str-partition-miss", str(list(SS[2].partition("z"))), "['abc', '', '']")
check("str-rpartition-miss", str(list(SS[2].rpartition("z"))), "['', '', 'abc']")
check("str-rpartition", str(list("abcb".rpartition("b"))), "['abc', 'b', '']")
check("str-removeprefix", SS[2].removeprefix("ab"), "c")
check("str-removeprefix-miss", SS[2].removeprefix("z"), "abc")
check("str-removesuffix", SS[2].removesuffix("bc"), "a")
check("str-expandtabs", "a\tb".expandtabs(), "a       b")
check("str-expandtabs-4", "ab\tc".expandtabs(4), "ab  c")
# format shares the mini-language with the f-strings.
check("str-format-auto", "{}-{}".format(SS[2], 3), "abc-3")
check("str-format-index", "{1}{0}{1}".format("a", "b"), "bab")
check("str-format-kw", "{x}/{y}".format(x=1, y="q"), "1/q")
check("str-format-spec", "{0:>6}".format(SS[2]), "   abc")
check("str-format-fill", "{0:*^7}".format(SS[2]), "**abc**")
check("str-format-braces", "{{{0}}}".format(1), "{1}")
check("str-format-item", "{0[1]}".format([7, 8]), "8")
check("str-format-nested", "{0:{1}}".format(SS[2], ">5"), "  abc")
check("str-format-conv", "{0!r}".format(SS[2]), "'abc'")


# ----- for-of drives a generator LAZILY (docs/todo.md 1.7) -----
# js_pyiter / iterToList MATERIALIZE, so an endless generator used to hang the
# interpreter and hit -max-steps in the compiler even when the body broke out of
# it on the first round. A generator is now STEPPED - the same shape C#'s foreach
# has always had - and everything else still normalizes as before.
def nat_lazy():
    i = 0
    while True:
        yield i
        i = i + 1

def upto_lazy(n):
    i = 0
    while i < n:
        yield i
        i = i + 1

lazy_out = []
for lz in nat_lazy():
    lazy_out.append(lz)
    if lz >= 3:
        break
check("gen-for-infinite", str(lazy_out), "[0, 1, 2, 3]")
lazy_first = -1
for lz in nat_lazy():
    lazy_first = lz
    break
check("gen-for-first-only", lazy_first, 0)
check("gen-for-finite", str([lz for lz in upto_lazy(3)]), "[0, 1, 2]")
lazy_sum = 0
for lz in upto_lazy(4):
    lazy_sum = lazy_sum + lz
else:
    lazy_sum = lazy_sum + 100
check("gen-for-else", lazy_sum, 106)
lazy_cont = []
for lz in upto_lazy(5):
    if lz % 2 == 0:
        continue
    lazy_cont.append(lz)
check("gen-for-continue", str(lazy_cont), "[1, 3]")
# The non-generator sources still take the array path: a dict iterates its keys,
# a string its characters, a set its elements.
check("gen-for-dict", str([lz for lz in {"a": 1, "b": 2}]), "['a', 'b']")
check("gen-for-str", str([lz for lz in "hi"]), "['h', 'i']")
lazy_nested = []
for lz in upto_lazy(2):
    for lz2 in upto_lazy(2):
        lazy_nested.append(lz * 10 + lz2)
check("gen-for-nested", str(lazy_nested), "[0, 1, 10, 11]")

# ----- integer arithmetic PROMOTES past 2^53 (docs/todo.md 1.1) -----
# A plain Python int is a double, exact to 2^53 and silently rounding above it.
# 54 of the 55 differing rows of a 626-literal probe against CPython 3.14.6 were
# the `*` column. Every operand is read out of a list so the constant folder
# cannot answer the assertion instead of the runtime.
bigv = [9007199254740992, 1, 2, 3037000500, 94906266, 123456789, -9007199254740992,
        4503599627370496, 987654321]
check("int-add-2p53", str(bigv[0] + bigv[1]), "9007199254740993")
check("int-add-2p53-folded", str(9007199254740992 + 1), "9007199254740993")
check("int-sub-neg-2p53", str(bigv[6] - bigv[1]), "-9007199254740993")
check("int-mul-2p53", str(bigv[0] * bigv[2]), "18014398509481984")
check("int-mul-square", str(bigv[3] * bigv[3]), "9223372037000250000")
check("int-mul-square2", str(bigv[4] * bigv[4]), "9007199326062756")
check("int-mul-big", str(bigv[5] * bigv[8]), "121932631112635269")
check("int-add-halves", str(bigv[7] + bigv[7] + bigv[1]), "9007199254740993")
check("int-mul-neg", str(bigv[6] * bigv[3]), "-27354868640248020074496000")
check("int-still-exact", str(bigv[5] + bigv[8]) + " " + str(bigv[5] * bigv[2]), "1111111110 246913578")
check("int-add-str-untouched", "99999999999999999999" + "9", "999999999999999999999")
check("int-mul-list-untouched", str([0] * 3), "[0, 0, 0]")
check("int-mul-str-untouched", "ab" * 3, "ababab")


check("combined-pipeline", f"{transform([1, 2, -3])}", "['o1', 'e2', 'x']")


# ----- repr() of a str: CPython's unicode_repr (docs/todo.md 1.4) -----
# repr escapes \ and the quote in force, has short forms for \t \n \r, and
# renders every non-printable code point as \xNN / \uNNNN / \UNNNNNNNN; the
# quote is ' unless the string holds a ' and no ". Every operand is read out of
# an array so the constant folder cannot answer the assertion, and the code
# points are built with chr() rather than written as literals.
rs = ["a\tb", "a\nb", "a\rb", "back\\slash", "it's", 'say "hi"', 'both \' and "',
      "café", "plain", ""]
check("repr-tab", repr(rs[0]), "'a\\tb'")
check("repr-newline", repr(rs[1]), "'a\\nb'")
check("repr-return", repr(rs[2]), "'a\\rb'")
check("repr-backslash", repr(rs[3]), "'back\\\\slash'")
check("repr-quote-switch", repr(rs[4]), '"it\'s"')
check("repr-double-kept", repr(rs[5]), "'say \"hi\"'")
check("repr-both-quotes", repr(rs[6]), "'both \\' and \"'")
check("repr-non-ascii-literal", repr(rs[7]), "'café'")
check("repr-plain", repr(rs[8]), "'plain'")
check("repr-empty", repr(rs[9]), "''")
cps = [0, 27, 127, 128, 159, 160, 173, 8232, 8239, 12288, 55296, 65279, 128512, 69821, 233, 8364]
check("repr-nul", repr(chr(cps[0])), "'\\x00'")
check("repr-esc-c0", repr(chr(cps[1])), "'\\x1b'")
check("repr-del", repr(chr(cps[2])), "'\\x7f'")
check("repr-c1-low", repr(chr(cps[3])), "'\\x80'")
check("repr-c1-high", repr(chr(cps[4])), "'\\x9f'")
check("repr-nbsp", repr(chr(cps[5])), "'\\xa0'")
check("repr-soft-hyphen", repr(chr(cps[6])), "'\\xad'")
check("repr-line-sep", repr(chr(cps[7])), "'\\u2028'")
check("repr-narrow-nbsp", repr(chr(cps[8])), "'\\u202f'")
check("repr-ideographic-space", repr(chr(cps[9])), "'\\u3000'")
check("repr-bom", repr(chr(cps[11])), "'\\ufeff'")
check("repr-astral-printable", repr(chr(cps[12])), "'" + chr(cps[12]) + "'")
check("repr-astral-format", repr(chr(cps[13])), "'\\U000110bd'")
check("repr-latin1-printable", repr(chr(cps[14])), "'é'")
check("repr-bmp-printable", repr(chr(cps[15])), "'€'")
check("repr-in-list", repr([rs[0]]), "['a\\tb']")
check("repr-in-dict", repr({rs[4]: rs[0]}), '{"it\'s": \'a\\tb\'}')
check("repr-f-string-conv", f"{rs[0]!r}", "'a\\tb'")

# ----- ord() and chr() (docs/todo.md 2.9) -----
# They were missing from ALL THREE engines, not just the interpreter.
ocs = ["A", "z", "0", " ", "é", "中"]
ons = [65, 122, 48, 32, 233, 20013, 0, 1114111, 128512]
check("ord-upper", ord(ocs[0]), 65)
check("ord-lower", ord(ocs[1]), 122)
check("ord-digit", ord(ocs[2]), 48)
check("ord-space", ord(ocs[3]), 32)
check("ord-latin1", ord(ocs[4]), 233)
check("ord-bmp", ord(ocs[5]), 20013)
check("chr-upper", chr(ons[0]), "A")
check("chr-digit", chr(ons[2]), "0")
check("chr-latin1", chr(ons[4]), "é")
check("chr-bmp", chr(ons[5]), "中")
check("chr-nul-len", len(chr(ons[6])), 1)
# An astral code point is ONE character to Python and a surrogate PAIR to the
# UTF-16 engines: ord must decode the pair rather than read the high surrogate.
check("chr-astral-len", len(chr(ons[8])), 1)
check("ord-chr-astral", ord(chr(ons[8])), 128512)
check("ord-chr-max", ord(chr(ons[7])), 1114111)
check("chr-round-trip", chr(ord(ocs[4])), "é")
check("ord-bool-is-int", chr(True), "\x01")

# ----- the four foreign method arms are gone (docs/todo.md 5.2) -----
# isEmpty / removeLast / sumOf / forEach were reachable on a str and a list in
# both COMPILED halves and raise AttributeError in CPython. The denial is by
# NAME and sits after the user-class arms, so a Python class that defines a
# method with one of those names must still work - which is what this asserts
# (the failure itself is not catchable in the interpreter half, so it is
# verified by tests/probe.sh rather than here).
class ForeignNames:
    def __init__(self, xs):
        self.xs = xs
    def isEmpty(self):
        return len(self.xs) == 0
    def removeLast(self):
        return self.xs.pop()
    def sumOf(self, f):
        t = 0
        for e in self.xs:
            t += f(e)
        return t
    def forEach(self, f):
        for e in self.xs:
            f(e)
        return None

fn = ForeignNames([1, 2, 3])
check("foreign-name-isEmpty", fn.isEmpty(), False)
check("foreign-name-sumOf", fn.sumOf(lambda v: v * 2), 12)
check("foreign-name-removeLast", fn.removeLast(), 3)
check("foreign-name-forEach", fn.forEach(lambda v: v), None)
check("foreign-name-after", str(fn.xs), "[1, 2]")
check("list-pop-still-there", str([9, 8].pop()), "8")

# ----- docs/todo.md 1.4: the builtins that were missing from all three engines --
# Every operand is read out of an ARRAY so the grammars' constant folder cannot
# evaluate the call at compile time.
bs = [3, 1, 2]
bt = ["a", "b"]
bn = [10, 255, 8, -5, 7, 2, 0]
bf = [2.5, 0.5, 1.5, -2.5, 2.675]

check("all-true", all([bn[5], bn[5]]), True)
check("all-false", all([bn[5], bn[6]]), False)
check("all-empty", all([]), True)
check("any-true", any([bn[6], bn[5]]), True)
check("any-false", any([bn[6], bn[6]]), False)
check("any-empty", any([]), False)
check("sorted-list", str(sorted(bs)), "[1, 2, 3]")
check("sorted-copy-untouched", str(bs), "[3, 1, 2]")
check("sorted-str", str(sorted("cba")), "['a', 'b', 'c']")
check("reversed-list", str(list(reversed(bs))), "[2, 1, 3]")
check("reversed-str", str(list(reversed("abc"))), "['c', 'b', 'a']")
check("enumerate", str(list(enumerate(bt))), "[[0, 'a'], [1, 'b']]")
check("enumerate-start", str(list(enumerate(bt, bn[4]))), "[[7, 'a'], [8, 'b']]")
check("zip-equal", str(list(zip(bs, bs))), "[[3, 3], [1, 1], [2, 2]]")
check("zip-shortest", str(list(zip(bs, bt))), "[[3, 'a'], [1, 'b']]")
check("map-builtin", str(list(map(abs, [-1, -2]))), "[1, 2]")
check("map-class-as-fn", ", ".join(map(str, bs)), "3, 1, 2")
check("map-two-sources", str(list(map(lambda a, b: a + b, bs, bs))), "[6, 2, 4]")
check("filter-none", str(list(filter(None, [bn[6], bn[5], bn[4]]))), "[2, 7]")
check("filter-fn", str(list(filter(lambda v: v > 1, bs))), "[3, 2]")
check("sum-of-map", sum(map(abs, [-1, -2])), 3)
check("max-of-iter", max(iter([bn[4], bn[5]])), 7)
check("set-of-str", str(sorted(set("aab"))), "['a', 'b']")

# The iterator object, and what makes it one: next() drives it, a partial read
# leaves the rest, and a for loop that BREAKS leaves it partially consumed.
it = iter(bs)
check("next-first", next(it), 3)
check("next-second", next(it), 1)
check("next-rest-after-partial", str(list(it)), "[2]")
check("next-default-exhausted", next(it, "END"), "END")
it2 = iter(bs)
for _v in it2:
    break
check("iter-break-leaves-rest", str(list(it2)), "[1, 2]")
check("iter-of-iter-is-same", str(list(iter(iter(bt)))), "['a', 'b']")

# next() on a GENERATOR: the compiler half had no `next` at all, so this is the
# sharpest of the item's two halves gaps.
def counter():
    yield bn[5]
    yield bn[4]

g = counter()
check("next-generator-1", next(g), 2)
check("next-generator-2", next(g), 7)
check("next-generator-default", next(g, "END"), "END")

def counter2():
    yield bn[5]

g2 = counter2()
check("next-generator-only", next(g2), 2)
stopped = [False]
try:
    next(g2)
except StopIteration:
    stopped[0] = True
check("next-generator-stopiteration", stopped[0], True)
stopped2 = [False]
try:
    next(iter([]))
except StopIteration:
    stopped2[0] = True
check("next-empty-stopiteration", stopped2[0], True)

check("bin", bin(bn[0]), "0b1010")
check("hex", hex(bn[1]), "0xff")
check("oct", oct(bn[2]), "0o10")
check("bin-negative-sign-outside", bin(bn[3]), "-0b101")
check("hex-negative-sign-outside", hex(-bn[1]), "-0xff")
check("bin-zero", bin(bn[6]), "0b0")
# Arbitrary precision: past 2^53 a double has lost digits, so this is the whole
# reason the digits come out through the big-integer path in all three engines.
check("hex-bignum", hex(bn[5] ** 70), "0x400000000000000000")
check("divmod-positive", str(divmod(bn[4], bn[5])), "[3, 1]")
check("divmod-negative-floors", str(divmod(-bn[4], bn[5])), "[-4, 1]")

# round() rounds HALF TO EVEN, which is wrong on every other integer if it is
# written as floor(x + 0.5).
check("round-half-even-2.5", round(bf[0]), 2)
check("round-half-even-0.5", round(bf[1]), 0)
check("round-half-even-1.5", round(bf[2]), 2)
check("round-half-even--2.5", round(bf[3]), -2)
check("round-ndigits", round(bf[4], bn[5]), 2.67)
check("round-int-stays-int", round(bn[4]), 7)

check("pow-two-args", pow(bn[5], bs[0]), 8)
check("pow-negative-exponent", pow(bn[5], -bn[5]), 0.25)
check("pow-modular", pow(bn[5], bn[0], 1000), 24)
check("pow-modular-big", pow(bn[4], 100, 13), 9)

check("callable-builtin", callable(len), True)
check("callable-number", callable(bn[0]), False)
check("callable-class", callable(int), True)
check("callable-lambda", callable(lambda v: v), True)
check("ascii-latin1", ascii("café"), "'caf\\xe9'")
check("ascii-plain", ascii(bt[0]), "'a'")

class AttrBag:
    def __init__(self):
        self.x = bn[5]

ab = AttrBag()
check("getattr-present", getattr(ab, "x"), 2)
check("getattr-default", getattr(ab, "zz", bn[4]), 7)
check("hasattr-present", hasattr(ab, "x"), True)
check("hasattr-absent", hasattr(ab, "zz"), False)
setattr(ab, "y", bn[0])
check("setattr", ab.y, 10)
check("hasattr-after-setattr", hasattr(ab, "y"), True)

# issubclass(), and the builtin hierarchy under it. issubclass was missing from
# the compiler half entirely; bool-derives-from-int and object-is-the-base were
# wrong in EVERY engine (isinstance(1, object) was even a live halves split).
class Base:
    pass

class Derived(Base):
    pass

check("issubclass-user", issubclass(Derived, Base), True)
check("issubclass-user-reverse", issubclass(Base, Derived), False)
check("issubclass-self", issubclass(Base, Base), True)
check("issubclass-tuple", issubclass(Derived, (int, Base)), True)
check("issubclass-user-object", issubclass(Derived, object), True)
check("issubclass-bool-int", issubclass(bool, int), True)
check("issubclass-int-bool", issubclass(int, bool), False)
check("issubclass-int-object", issubclass(int, object), True)
check("issubclass-str-int", issubclass(str, int), False)
check("isinstance-true-is-int", isinstance(True, int), True)
check("isinstance-int-object", isinstance(1, object), True)
check("isinstance-instance-object", isinstance(Derived(), object), True)
check("isinstance-str-not-int", isinstance(bt[0], int), False)

# ----- docs/todo.md 1.5: list.count(x) -----
# The interpreter had no arm; both compiled halves read the argument as a KOTLIN
# predicate and died with "call of a non function value: 2".
cs = [1, 2, 2, 3]
check("list-count", cs.count(bn[5]), 2)
check("list-count-absent", cs.count(bn[4]), 0)
check("list-count-by-equality", [1.0, 2].count(bs[1]), 1)
check("list-count-bool-is-int", [True, 1, 2].count(bs[1]), 2)
# str.count was already CPython's non-overlapping substring count - a NULL
# RESULT, asserted so it stays one.
check("str-count-substring", "abcab".count("ab"), 2)
check("str-count-non-overlapping", "aaaa".count("aa"), 2)

# ----- docs/todo.md 1.4: set.add and dict.pop -----
sa = {1, 2}
sa.add(bs[0])
sa.add(bn[5])
check("set-add", str(sorted(sa)), "[1, 2, 3]")
check("set-add-duplicate-is-noop", len(sa), 3)
dp = {"a": 1, "b": 2}
check("dict-pop", dp.pop("a"), 1)
check("dict-pop-removed", str(dp), "{'b': 2}")
check("dict-pop-default", dp.pop("zz", bn[4]), 7)
popped = [False]
try:
    dp.pop("zz")
except KeyError:
    popped[0] = True
check("dict-pop-keyerror", popped[0], True)

# ----- docs/todo.md 2.4: twelve foreign method names on python receivers -----
# add / size / get / contains / map / filter / any on a list and length / charAt
# / equals / substring / indexOf on a str all SUCCEEDED under llvm.Run where
# CPython raises AttributeError. All three engines now raise a CATCHABLE one, so
# unlike the four names closed in 555af82 this is assertable in one file.
def denied(thunk):
    try:
        thunk()
    except AttributeError:
        return True
    return False

fl = [1, 2]
check("foreign-list-add", denied(lambda: fl.add(3)), True)
check("foreign-list-size", denied(lambda: fl.size()), True)
check("foreign-list-get", denied(lambda: fl.get(0)), True)
check("foreign-list-contains", denied(lambda: fl.contains(1)), True)
check("foreign-list-map", denied(lambda: fl.map(abs)), True)
check("foreign-list-filter", denied(lambda: fl.filter(abs)), True)
check("foreign-list-any", denied(lambda: fl.any(abs)), True)
check("foreign-list-isEmpty", denied(lambda: fl.isEmpty()), True)
check("foreign-list-removeLast", denied(lambda: fl.removeLast()), True)
check("foreign-list-sumOf", denied(lambda: fl.sumOf(abs)), True)
check("foreign-list-forEach", denied(lambda: fl.forEach(abs)), True)
check("foreign-str-length", denied(lambda: bt[0].length()), True)
check("foreign-str-charAt", denied(lambda: bt[0].charAt(0)), True)
check("foreign-str-equals", denied(lambda: bt[0].equals("a")), True)
check("foreign-str-substring", denied(lambda: bt[0].substring(0, 1)), True)
check("foreign-str-indexOf", denied(lambda: bt[0].indexOf("a")), True)
check("foreign-list-untouched", str(fl), "[1, 2]")
# The denial is by RECEIVER TYPE, not by name: `get` is a real dict method and
# `add` a real set method, and a flat name switch would have broken both.
check("dict-get-still-works", {"a": 1}.get("a"), 1)
check("dict-get-default-still-works", {"a": 1}.get("zz", bn[4]), 7)
check("set-add-still-works", len(sa), 3)
# A name Python HAS and this project does not (list.sort) is deliberately NOT
# denied - it still aborts, so "unimplemented" cannot be silently swallowed.

# ----- docs/todo.md 1.3: the rest of the dict and set method surface -----
# clear / copy / setdefault / update on a dict and pop / clear / copy / remove /
# discard / union / update on a set all ABORTED in every engine at f4ff228.
ks = ["a", "b", "c"]
vs = [1, 2, 3]
dm = {}
dm[ks[0]] = vs[0]
dm[ks[1]] = vs[1]
dmc = dm.copy()
dmc[ks[2]] = vs[2]
check("dict-copy-shallow", str(dm) + "|" + str(dmc), "{'a': 1, 'b': 2}|{'a': 1, 'b': 2, 'c': 3}")
check("dict-setdefault-hit", dm.setdefault(ks[0], vs[2]), 1)
check("dict-setdefault-new", dm.setdefault(ks[2], vs[2]), 3)
check("dict-setdefault-none-default", str(dm.setdefault("zz")), "None")
dmu = {}
dmu.update({ks[0]: vs[0]})
dmu.update([[ks[1], vs[1]]])
dmu.update(c=vs[2])
check("dict-update-map-pairs-kw", str(dmu), "{'a': 1, 'b': 2, 'c': 3}")
dmu.clear()
check("dict-clear", str(dmu) + "|" + str(len(dmu)), "{}|0")
sm = set()
sm.add(vs[0])
sm.add(vs[1])
smc = sm.copy()
smc.add(vs[2])
check("set-copy-shallow", str(len(sm)) + "|" + str(len(smc)), "2|3")
# pop() takes an ARBITRARY element: only the invariants are assertable.
smp = sm.copy()
pv = smp.pop()
check_true("set-pop-invariants", len(smp) == 1 and pv not in smp and pv in sm)
check("set-discard-absent-is-quiet", str(sm.discard(vs[2])), "None")
check("set-remove-present", str(sm.remove(vs[1])) + "|" + str(len(sm)), "None|1")
removed = [False]
try:
    sm.remove(vs[2])
except KeyError:
    removed[0] = True
check("set-remove-keyerror", removed[0], True)
emptied = [False]
try:
    set().pop()
except KeyError:
    emptied[0] = True
check("set-pop-empty-keyerror", emptied[0], True)
smu = sm.union([vs[1]], "x")
check("set-union-any-iterable", str(sorted([str(x) for x in smu])), "['1', '2', 'x']")
check("set-union-does-not-mutate", len(sm), 1)
sm.update([vs[1], vs[2]])
check("set-update-in-place", len(sm), 3)
# hasattr tracks the surface exactly, and denies the OTHER type's names.
check_true("hasattr-dict-surface",
           hasattr(dm, "clear") and hasattr(dm, "copy") and
           hasattr(dm, "setdefault") and hasattr(dm, "update"))
check_true("hasattr-set-surface",
           hasattr(sm, "pop") and hasattr(sm, "clear") and hasattr(sm, "copy") and
           hasattr(sm, "remove") and hasattr(sm, "discard") and
           hasattr(sm, "union") and hasattr(sm, "update"))
check_true("hasattr-cross-type-false",
           not hasattr(sm, "keys") and not hasattr(dm, "discard"))
check("set-keys-is-attributeerror", denied(lambda: sm.keys()), True)
check("dict-union-is-attributeerror", denied(lambda: dm.union([1])), True)
# The set ORDERINGS: <= and >= answered True for ANY pair of sets in the
# interpreter half at f4ff228, because two objects fell into the host's own
# relational operator.
so1 = set()
so1.add(vs[0])
so2 = sm
check_true("set-subset-ordering", so1 <= so2 and so2 >= so1 and so1 < so2)
check_true("set-not-subset", not (so2 <= so1) and not (so1 > so2) and so1 <= so1)

print(f"features: {checks[0]} checks, {fails[0]} failures")
exit(fails[0])
