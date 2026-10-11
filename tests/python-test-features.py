# Fast feature-matrix test for the Python interpreter (python-interpreter.abnf) and
# the LLVM-IR compiler (python-to-llvm-ir.abnf): every implemented construct is
# exercised with the SMALLEST program that can prove it works - loops run 0, 1, 3
# or 4 times, recursion stays below depth 6. A failed check prints its id (so a
# diff pinpoints it) and the file ends with exit(fails[0]); exit 0 and
# byte-identical output on all four legs (interpreter/compiler x goja/-frozen)
# mean everything passed.

import math
import re

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
check_true("dict-items", len(its) == 3 and list(its)[0][0] == "alice" and list(its)[0][1] == 30)

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
# The module has an annotation of its own (mod_ann), and the class fill must not
# pick it up from the MODULE scope: python3 reads {} for AnnUnreached.
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
# The THIRD shape: an unreached annotation FOLLOWED by a reached one. The reached
# one must still create the class's dict (and must not land in the module's).
# python3 says {'am_yes': <class 'int'>}.
check_true("class-annotations-mixed", len(AnnMixed.__annotations__) == 1 and "am_yes" in AnnMixed.__annotations__ and "am_no" not in AnnMixed.__annotations__)

# Integer literals past 2**53. The box is chosen on the literal TEXT (parseFloat
# rounds 9007199254740993 down to exactly 2**53), and the 0x / 0o / 0b forms are
# boxed by the same rule.
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


# ----- str's method library -----
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


# ----- for-of drives a generator LAZILY -----
# A generator is STEPPED, so an endless one works when the body breaks out of
# it; everything else still normalizes to a list.
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

# ----- integer arithmetic PROMOTES past 2^53 -----
# A plain Python int is a double, exact to 2^53, so arithmetic must promote to
# the big-integer box above it. Every operand is read out of a list so the
# constant folder cannot answer the assertion instead of the runtime.
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


# ----- repr() of a str: CPython's unicode_repr -----
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

# ----- ord() and chr() -----
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

# ----- foreign method names on a user class -----
# isEmpty / removeLast / sumOf / forEach raise AttributeError on a str and a list
# in CPython. The denial is by NAME and sits after the user-class arms, so a
# Python class that defines a method with one of those names must still work -
# which is what this asserts (the failure itself is not catchable in the
# interpreter half, so it is verified by tests/probe.sh rather than here).
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

# ----- builtins shared by all three engines --------------------------------------
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
check("enumerate", str(list(enumerate(bt))), "[(0, 'a'), (1, 'b')]")
check("enumerate-start", str(list(enumerate(bt, bn[4]))), "[(7, 'a'), (8, 'b')]")
check("zip-equal", str(list(zip(bs, bs))), "[(3, 3), (1, 1), (2, 2)]")
check("zip-shortest", str(list(zip(bs, bt))), "[(3, 'a'), (1, 'b')]")
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

# next() on a GENERATOR.
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
check("divmod-positive", str(divmod(bn[4], bn[5])), "(3, 1)")
check("divmod-negative-floors", str(divmod(-bn[4], bn[5])), "(-4, 1)")

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

# A BOUND METHOD knows its own name: getattr([1,2], "count") is a FRESH closure
# that neither the def-name table nor the builtin-render table is keyed to find,
# and CPython says 'count'. Every name is read out of a list so the constant
# folder cannot answer it.
mnm = ["count", "upper", "m"]
blc = getattr([3, 1, 2], mnm[0])
check("bound-builtin-name", blc.__name__, "count")
check("bound-builtin-qualname", blc.__qualname__, "list.count")
check("bound-builtin-str-name", getattr("ab", mnm[1]).__name__, "upper")
check("bound-builtin-type", type(blc).__name__, "builtin_function_or_method")
check("bound-builtin-repr", str(blc), "<built-in method count of list object>")
check("bound-builtin-hasattr", hasattr(blc, "__name__"), True)
check("bound-builtin-call", blc(1), 1)
check("builtin-fn-type", type(len).__name__, "builtin_function_or_method")
check("class-obj-type", type(int).__name__, "type")
check("user-fn-type", type(lambda v: v).__name__, "function")

class Bnd:
    def m(self, q):
        return q + 1

bo = Bnd()
bm2 = getattr(bo, mnm[2])
check("bound-method-name", bm2.__name__, "m")
check("bound-method-qualname", bm2.__qualname__, "Bnd.m")
check("bound-method-type", type(bm2).__name__, "method")
check("bound-method-self", bm2.__self__ is bo, True)
check("bound-method-func", bm2.__func__ is Bnd.m, True)
check("bound-method-repr", str(bm2), "<bound method Bnd.m of <Bnd object>>")

# A missing attribute RAISES, catchably, with CPython's exact text - for a
# builtin value and for a class object.
def attr_miss(o):
    try:
        getattr(o, "nope_xyz")
        return "no-raise"
    except AttributeError as ex:
        return str(ex)

check("attr-miss-list", attr_miss([1, 2]), "'list' object has no attribute 'nope_xyz'")
check("attr-miss-inst", attr_miss(bo), "'Bnd' object has no attribute 'nope_xyz'")
check("attr-miss-class", attr_miss(Bnd), "type object 'Bnd' has no attribute 'nope_xyz'")

# An except HANDLER BODY IS NOT A SCOPE: a name first bound inside one survives
# the handler.
try:
    raise ValueError(mnm[0])
except ValueError as exv:
    hbound = "caught " + str(exv)
check("except-handler-not-a-scope", hbound, "caught count")

# issubclass(), and the builtin hierarchy under it: bool derives from int and
# object is the base of everything (isinstance(1, object)).
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

# ----- list.count(x) -----
cs = [1, 2, 2, 3]
check("list-count", cs.count(bn[5]), 2)
check("list-count-absent", cs.count(bn[4]), 0)
check("list-count-by-equality", [1.0, 2].count(bs[1]), 1)
check("list-count-bool-is-int", [True, 1, 2].count(bs[1]), 2)
# str.count is CPython's non-overlapping substring count.
check("str-count-substring", "abcab".count("ab"), 2)
check("str-count-non-overlapping", "aaaa".count("aa"), 2)

# ----- set.add and dict.pop -----
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

# ----- foreign method names on python receivers -----
# add / size / get / contains / map / filter / any on a list and length / charAt
# / equals / substring / indexOf on a str: CPython raises AttributeError, and all
# three engines raise a CATCHABLE one.
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

# ----- the rest of the dict and set method surface -----
# clear / copy / setdefault / update on a dict and pop / clear / copy / remove /
# discard / union / update on a set.
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
# The set ORDERINGS: <= and >= are subset / superset, not the host's own
# relational operator on two objects.
so1 = set()
so1.add(vs[0])
so2 = sm
check_true("set-subset-ordering", so1 <= so2 and so2 >= so1 and so1 < so2)
check_true("set-not-subset", not (so2 <= so1) and not (so1 > so2) and so1 <= so1)

# ----- a subscript miss is CATCHABLE, and a generator's `finally` runs when
# CPython runs it. The KeyError carries the key itself (repr(e) is not doubly
# quoted), the exception hierarchy is CPython's, and the interpreter's replay must
# not run a generator's `finally` at the first next().
ky = [3, "b", 9]
kd = {1: "one"}


def why(f):
    try:
        f()
        return "no-raise"
    except BaseException as e:
        return type(e).__name__ + "|" + str(e) + "|" + repr(e)


check("subscript-miss-is-catchable", why(lambda: kd[ky[0]]), "KeyError|3|KeyError(3)")
check("subscript-miss-str-key", why(lambda: kd[ky[1]]), "KeyError|'b'|KeyError('b')")
check("index-miss-is-catchable", why(lambda: [1][ky[2]]),
      "IndexError|list index out of range|IndexError('list index out of range')")
check("keyerror-arg-is-the-key", str(KeyError(ky[0]).args[0]), "3")
check_true("keyerror-hierarchy",
           issubclass(KeyError, LookupError) and issubclass(LookupError, Exception) and
           issubclass(Exception, BaseException) and not issubclass(LookupError, KeyError))
kg = []


def kgen():
    try:
        kg.append("body")
        yield ky[0]
        kg.append("after")
        yield ky[1]
    finally:
        kg.append("fin")


kit = kgen()
kg.append("d0")
next(kit)
kg.append("d1")
kit.close()
kg.append("d2")
korder = []
for k in kg:
    if k not in korder:
        korder.append(k)
check("generator-finally-order", str(korder), "['d0', 'body', 'd1', 'fin', 'd2']")

# ----- g.throw(exc): raise AT the parked yield -----
# The record travels on the RESUME VALUE, exactly as close()'s exit record does,
# which is what makes it per-coroutine by construction. runtime.c's gen_throw and
# abnf/jsrt.go's throwInto answer the compiled halves; genThrow answers this one.
# Every append is in the DRIVER or in a `finally`: a write before the parked yield
# repeats once per replay in the interpreter half, so the two halves would disagree.
tg = []


def tgen():
    try:
        yield ky[0]
        yield ky[1]
    except ValueError as e:
        yield "c:" + str(e)
    finally:
        tg.append("fin")


tit = tgen()
tg.append("n" + str(next(tit)))
tg.append("t" + str(tit.throw(ValueError("X"))))
try:
    next(tit)
except StopIteration:
    tg.append("stop")
torder = []
for t in tg:
    if t not in torder:
        torder.append(t)
check("gen-throw-caught", str(torder), "['n3', 'tc:X', 'fin', 'stop']")


def tgen2():
    yield ky[0]
    yield ky[1]


tit2 = tgen2()
next(tit2)
check("gen-throw-propagates", why(lambda: tit2.throw(KeyError(ky[0]))), "KeyError|3|KeyError(3)")
check("gen-throw-leaves-it-done", why(lambda: next(tit2)), "StopIteration||StopIteration()")
# A generator that never ran has no suspension point and therefore no `finally` to
# unwind: CPython runs nothing at all and the value propagates. Same for a done one.
tg3 = []


def tgen3():
    try:
        yield ky[0]
    finally:
        tg3.append("fin3")


tit3 = tgen3()
check("gen-throw-not-started", why(lambda: tit3.throw(ValueError("Z"))), "ValueError|Z|ValueError('Z')")
check("gen-throw-not-started-no-finally", str(tg3), "[]")
# A body that CATCHES the throw and returns ends the generator, and the
# StopIteration carries that return value - send()'s protocol, not JavaScript's
# {value, done} record, which is what layer 2's pyGenValue unwraps.


def tgen4():
    try:
        yield ky[0]
    except ValueError:
        return


tit4 = tgen4()
next(tit4)
check("gen-throw-catch-and-return", why(lambda: tit4.throw(ValueError("Q"))),
      "StopIteration||StopIteration()")

# StopIteration's ARGS: empty when the body returned None and (v,) when it returned
# a value, which repr() shows.


def tgen5():
    yield ky[1]
    return ky[0]


tit5 = tgen5()
next(tit5)
check("stopiter-args-value", why(lambda: next(tit5)), "StopIteration|3|StopIteration(3)")


def tgen6():
    yield ky[0]


tit6 = tgen6()
next(tit6)
check("stopiter-args-none", why(lambda: next(tit6)), "StopIteration||StopIteration()")

# ----- super(), the MRO, and dict.popitem -----
# The diamond is the point: depth-first over the bases finds A.who, C3 finds
# C.who, and CPython says C.
snm = ["A", "B", "C", "D"]


class SA:
    def who(self):
        return snm[0]


class SB(SA):
    pass


class SC(SA):
    def who(self):
        return snm[2]


class SD(SB, SC):
    def who(self):
        return snm[3] + super().who()


check("super-diamond-c3", SD().who(), "DC")
check("super-mro", str([k.__name__ for k in type(SD()).__mro__]),
      "['SD', 'SB', 'SC', 'SA', 'object']")
check("super-bases", str([k.__name__ for k in SD.__bases__]), "['SB', 'SC']")
check("super-explicit", super(SD, SD()).who(), "C")
check_true("super-bound-value", SD().who() == SD().who())


class SE(SA):
    def __init__(self, v):
        self.v = v * 2

    def who(self):
        return "E" + super().who()


check("super-init-chain", SE(3).v, 6)
check("super-through-value", SE(1).who(), "EA")
sbm = SE(2).who
check("bound-method-off-instance", sbm(), "EA")
spd = {}
for spk in ["a", "b"]:
    spd[spk] = len(spd)
check("dict-popitem-lifo", str(spd.popitem()), "('b', 1)")
check("dict-popitem-rest", str(spd), "{'a': 0}")
check("dict-popitem-empty", why(lambda: {}.popitem()),
      "KeyError|'popitem(): dictionary is empty'|KeyError('popitem(): dictionary is empty')")

# ----- `yield from` DELEGATES LAZILY -----
# js/ts forward lazily and match node, which is the precedent this follows
# (js-interpreter.abnf's doYield(star) / js-to-llvm-ir.abnf's makeYieldStar).
yf = [0, 1, 2, 3]


def yf_endless():
    i = yf[0]
    while True:
        yield i
        i = i + 1


def yf_take():
    yield from yf_endless()


# LAZY: an endless delegate is only stepped as far as it is asked.
yt = yf_take()
check("yieldfrom-lazy", str([next(yt), next(yt), next(yt)]), "[0, 1, 2]")


def yf_ret():
    yield yf[1]
    return yf[3]


def yf_outer_ret():
    v = yield from yf_ret()
    yield v


# The value of `yield from` is the delegate's RETURN value.
yr = yf_outer_ret()
check("yieldfrom-value-is-return", str([next(yr), next(yr)]), "[1, 3]")


def yf_echo():
    while True:
        got = yield yf[0]
        if got is not None:
            yield got


def yf_send():
    yield from yf_echo()


# send(v) reaches a DELEGATED generator: the sent value is forwarded to its next().
ys = yf_send()
next(ys)
check("yieldfrom-send-reaches-delegate", ys.send(yf[2]), 2)


def yf_inner_catch():
    try:
        yield yf[1]
    except ValueError:
        yield yf[2]


def yf_outer_catch():
    yield from yf_inner_catch()


# g.throw(e) at a `yield from` is FORWARDED to the delegate's own throw(), so the
# DELEGATE's except arm takes it.
yc = yf_outer_catch()
next(yc)
check("yieldfrom-throw-reaches-delegate", yc.throw(ValueError("y")), 2)


def yf_list_outer():
    try:
        yield from [yf[1], yf[2]]
    except ValueError as e:
        yield "outer:" + str(e)


# A cursor has no throw() to forward to, so the value is raised AT the yield from
# and the OUTER body's except arm takes it - CPython's rule for a list_iterator.
yl = yf_list_outer()
next(yl)
check("yieldfrom-throw-no-delegate-method", yl.throw(ValueError("z")), "outer:z")
yfo = []


def yf_fin_inner():
    try:
        yield yf[1]
    finally:
        yfo.append("inner-fin")


def yf_fin_outer():
    try:
        yield from yf_fin_inner()
    finally:
        yfo.append("outer-fin")


# close() at a `yield from` closes the DELEGATE first, so its finally runs before
# the outer's.
yfi = yf_fin_outer()
yfo.append("d0")
next(yfi)
# The marker between the next() and the close() is what makes this discriminate:
# a MATERIALIZED delegate would run its finally during the first next(), i.e.
# BEFORE this line.
yfo.append("mid")
yfi.close()
yfo.append("d1")
yforder = []
for yfk in yfo:
    if yfk not in yforder:
        yforder.append(yfk)
check("yieldfrom-close-closes-delegate", str(yforder),
      "['d0', 'mid', 'inner-fin', 'outer-fin', 'd1']")


def yf_plain_list():
    yield from [yf[1], yf[2]]


check("yieldfrom-list", str(list(yf_plain_list())), "[1, 2]")

# ----- `raise SomeClass` raises an INSTANCE -----
# A bare class must be instantiated (js_pyraiseval), or it matches no except
# clause.


class RErr(Exception):
    pass


def rc_bare():
    raise RErr


check("raise-bare-class", why(rc_bare), "RErr||RErr()")


def rc_bare_builtin():
    raise ValueError


check("raise-bare-value-error", why(rc_bare_builtin), "ValueError||ValueError()")


def rt_gen():
    try:
        yield yf[1]
    except RErr:
        yield yf[2]


# g.throw(SomeClass) instantiates it the same way `raise SomeClass` does.
rtg = rt_gen()
next(rtg)
check("gen-throw-bare-class", rtg.throw(RErr), 2)

# ----- dict.fromkeys and vars() -----
# The receiver of fromkeys is the `dict` TYPE OBJECT, which reaches neither the
# dict dispatcher nor a class MRO - a different arm in each of the three engines,
# and both the mcall path and the VALUE path need it.
fk = ["a", "b", "a"]
check("dict-fromkeys-default", str(dict.fromkeys(fk)), "{'a': None, 'b': None}")
check("dict-fromkeys-value", str(dict.fromkeys(fk, yf[0])), "{'a': 0, 'b': 0}")
fkf = dict.fromkeys
check("dict-fromkeys-as-value", str(fkf(fk, yf[1])), "{'a': 1, 'b': 1}")
check("dict-fromkeys-from-str", str(dict.fromkeys("ab")), "{'a': None, 'b': None}")


class VC:
    def __init__(self, n):
        self.n = n


vco = VC(yf[2])
# vars(o) IS o.__dict__.
check("vars-instance", str(vars(vco)), "{'n': 2}")
check("vars-is-dunder-dict", str(vars(vco)), str(vco.__dict__))


def vfn(a):
    return a


# A function prints as <function NAME>: CPython's address cannot be matched, so
# it is left off - the same rule <map object> already follows.
check("function-str", str(vfn), "<function vfn>")
check("function-lambda-str", str(lambda vz: vz), "<function <lambda>>")
check("function-name", vfn.__name__, "vfn")
check("lambda-name", (lambda vz: vz).__name__, "<lambda>")


# ----- iterator cursors, delegated send, builtin repr -----
# The full assertion set is tests/python-test-full.py SECTION 39: most of what
# these cover lives in languages/lib/python-rt.metajs, and the matrix's native
# rows only BUILD, so only the ratchet runs them natively.
itv = [[1, 2, 3, 4], "ab", {"a": 1}]
itc = iter(itv[0])
check("cursor-dunder-next", itc.__next__(), 1)
check_true("cursor-dunder-iter", itc.__iter__() is itc)
check("cursor-rest", str(list(itc)), "[2, 3, 4]")


def it_denied(f):
    try:
        f()
    except AttributeError as e:
        return str(e)
    return "no error"


itc2 = iter(itv[0])
check("cursor-send-denied", it_denied(lambda: itc2.send(7)),
      "'list_iterator' object has no attribute 'send'")
check("cursor-close-denied", it_denied(lambda: itc2.close()),
      "'list_iterator' object has no attribute 'close'")
check("cursor-kind-list", type(iter(itv[0])).__name__, "list_iterator")
check("cursor-kind-str", type(iter(itv[1])).__name__, "str_ascii_iterator")
check("cursor-kind-dict", type(iter(itv[2])).__name__, "dict_keyiterator")
check("cursor-sum", sum(iter(itv[0])), 10)
check_true("cursor-in", 3 in iter(itv[0]))
itsl = [9, 9, 9, 9]
itsl[1:3] = iter([5, 6])
check("cursor-slice-assign", str(itsl), "[9, 5, 6, 9]")


def it_over_list():
    yield from itv[0]


itg = it_over_list()
check("yieldfrom-first", next(itg), 1)
check("yieldfrom-send-denied", it_denied(lambda: itg.send(7)),
      "'list_iterator' object has no attribute 'send'")
check("builtin-str-len", str(len), "<built-in function len>")
check("builtin-str-sorted", str(sorted), "<built-in function sorted>")
check("builtin-str-map", str(map), "<class 'map'>")
check("builtin-str-range", str(range), "<class 'range'>")
check("builtin-str-type-name", str(str), "<class 'str'>")



# ----- SECTION: Python's STATIC binding analysis -----
#
# Every name a function body binds is a local for the WHOLE body, so a read
# before the binding is an UnboundLocalError - and, crucially, does NOT fall
# through to an enclosing binding of the same name (`sb_shadow` below must not
# answer "global").
sb_global = "global"
UNBOUND = "cannot access local variable"


def sb_caught(f):
    try:
        f()
        return "no-raise"
    except UnboundLocalError as e:
        return "UnboundLocalError " + ("yes" if UNBOUND in str(e) else "no")
    except NameError as e:
        return "NameError"


def sb_read_before():
    r = sb_global
    sb_global = 1
    return r


def sb_augassign():
    sb_aug += 1
    return sb_aug


def sb_branch(flag):
    if flag:
        q = 3
    return q


def sb_delete():
    z = 1
    del z
    return z


def sb_del_only():
    del never_bound_here
    return 1


def sb_loopvar():
    for k in []:
        pass
    return k


def sb_except_as():
    try:
        raise ValueError("v")
    except ValueError as sb_e:
        pass
    return sb_e


def sb_nested_def():
    r = sb_inner_fn
    def sb_inner_fn():
        pass
    return r


def sb_nested_class():
    r = SbInner
    class SbInner:
        pass
    return r


def sb_tuple_target():
    r = sb_t1
    sb_t1, sb_t2 = 1, 2
    return r


def sb_annotation():
    r = sb_ann
    sb_ann: int = 3
    return r


def sb_match_capture():
    r = sb_cap
    match 1:
        case sb_cap:
            pass
    return r


def sb_walrus():
    r = sb_w
    if (sb_w := 2) > 1:
        pass
    return r


def sb_typealias():
    r = SbAlias
    type SbAlias = int
    return r


check("sb-read-before", sb_caught(sb_read_before), "UnboundLocalError yes")
check("sb-augassign", sb_caught(sb_augassign), "UnboundLocalError yes")
check("sb-branch-not-taken", sb_caught(lambda: sb_branch(False)), "UnboundLocalError yes")
check("sb-branch-taken", sb_branch(True), 3)
check("sb-delete", sb_caught(sb_delete), "UnboundLocalError yes")
check("sb-del-unbound", sb_caught(sb_del_only), "UnboundLocalError yes")
check("sb-loopvar", sb_caught(sb_loopvar), "UnboundLocalError yes")
check("sb-except-as", sb_caught(sb_except_as), "UnboundLocalError yes")
check("sb-nested-def", sb_caught(sb_nested_def), "UnboundLocalError yes")
check("sb-nested-class", sb_caught(sb_nested_class), "UnboundLocalError yes")
check("sb-tuple-target", sb_caught(sb_tuple_target), "UnboundLocalError yes")
check("sb-annotation", sb_caught(sb_annotation), "UnboundLocalError yes")
check("sb-match-capture", sb_caught(sb_match_capture), "UnboundLocalError yes")
check("sb-walrus", sb_caught(sb_walrus), "UnboundLocalError yes")
check("sb-typealias", sb_caught(sb_typealias), "UnboundLocalError yes")

# `global` and `nonlocal` take the name OUT of the local set again, so a read
# before the assignment sees the outer binding exactly as CPython does.
sb_g = "outer"


def sb_uses_global():
    global sb_g
    r = sb_g
    sb_g = "inner"
    return r


check("sb-global-declared", sb_uses_global(), "outer")
check("sb-global-written", sb_g, "inner")


def sb_nonlocal_outer():
    n = "outer"

    def inner():
        nonlocal n
        r = n
        n = "inner"
        return r

    return [inner(), n]


check("sb-nonlocal-declared", str(sb_nonlocal_outer()), "['outer', 'inner']")


# A COMPREHENSION is its own scope, so its target is NOT a local of the function
# around it - a bare read after it is a NameError, not an UnboundLocalError.
def sb_comp_target():
    r = [sb_ci for sb_ci in range(3)]
    try:
        return sb_ci
    except NameError:
        return "NameError " + str(r)


check("sb-comp-target-not-local", sb_comp_target(), "NameError [0, 1, 2]")


# A CLASS BODY compiles its reads as LOAD_NAME, so the same shape is a NameError
# there and an UnboundLocalError in a function.
sb_cls_probe = ["unset"]


class SbClassBody:
    try:
        sb_cls_probe[0] = sb_cb
    except NameError:
        sb_cls_probe[0] = "NameError"
    except UnboundLocalError:
        sb_cls_probe[0] = "UnboundLocalError"
    sb_cb = 1


check("sb-class-body-is-nameerror", sb_cls_probe[0], "NameError")


# A FREE variable of an enclosing function that never got a value is a NameError,
# not an UnboundLocalError: the empty slot is not this frame's own.
def sb_free_var():
    def inner():
        return sb_fv
    try:
        return inner()
    except NameError as e:
        return "NameError " + str(e)
    sb_fv = 1


check("sb-free-var-is-nameerror", sb_free_var(),
      "NameError name 'sb_fv' is not defined")


# `obj.attr = v` and `xs[0] = v` bind NOTHING, so the base name must stay a read
# of the enclosing binding. Getting this wrong turns every attribute store in a
# function into an UnboundLocalError.
class SbHolder:
    pass


sb_holder = SbHolder()
sb_holder.v = 1
sb_list = [0]


def sb_attr_store():
    sb_holder.v = 2
    sb_list[0] = 3
    return [sb_holder.v, sb_list[0]]


check("sb-attr-store-binds-nothing", str(sb_attr_store()), "[2, 3]")

# An ordinary expression statement whose whole expression is a bare NAME must not
# make that name a local - the parser tries Target three times on it before
# ExprSmall reads it, and a tag on Target would have fired for all three.
sb_bare = 7


def sb_bare_expr_stmt():
    sb_bare
    sb_bare == 1
    return sb_bare


check("sb-bare-expression-statement", sb_bare_expr_stmt(), 7)


# ----- SECTION: del obj.attr, and the re type names --------------------------
class DelAttrHolder:
    pass


_dh = DelAttrHolder()
_dh.gone = 1
check("delattr-before", _dh.gone, 1)
del _dh.gone
check("delattr-after", hasattr(_dh, "gone"), False)

_pat = re.compile("a(b)")
_mat = _pat.match("ab")
check("re-pattern-type-name", type(_pat).__name__, "Pattern")
check("re-match-type-name", type(_mat).__name__, "Match")
check("re-pattern-type-repr", repr(type(_pat)), "<class 're.Pattern'>")
check("re-match-type-repr", repr(type(_mat)), "<class 're.Match'>")
check_true("re-pattern-type-interned", type(_pat) is type(re.compile("z")))
check("re-pattern-attr-error", it_denied(lambda: _pat.nope),
      "'re.Pattern' object has no attribute 'nope'")
check("re-match-attr-error", it_denied(lambda: _mat.nope),
      "'re.Match' object has no attribute 'nope'")
# A BUILTIN type object is interned in every engine, so `type(x) is type(y)`
# holds.
check_true("type-obj-interned-cursor", type(iter([1])) is type(iter([2])))
check_true("type-obj-interned-builtin", type(len) is type(sorted))


# ----- SECTION: the math module -----
mv = [2.0, -2.7, 0.5, 100.0, 8.0, 3.0, 4.0, 12.0, 18.0]
check("math-pi", str(math.pi), "3.141592653589793")
check("math-e", str(math.e), "2.718281828459045")
check("math-tau", str(math.tau), "6.283185307179586")
check("math-inf", str(math.inf), "inf")
check("math-nan", str(math.nan), "nan")
check("math-sqrt", math.sqrt(mv[0]), 1.4142135623730951)
check("math-sqrt-is-float", str(math.sqrt(mv[6])), "2.0")
check("math-floor-neg", math.floor(mv[1]), -3)
check("math-ceil-neg", math.ceil(mv[1]), -2)
check("math-trunc-neg", math.trunc(mv[1]), -2)
check("math-floor-is-int", str(math.floor(mv[0])), "2")
check("math-fabs-is-float", str(math.fabs(mv[1])), "2.7")
check("math-pow-is-float", str(math.pow(mv[0], mv[4])), "256.0")
check("math-exp", str(math.exp(0.0)), "1.0")
check("math-log-e", str(math.log(math.e)), "1.0")
check("math-log-base", str(math.log(mv[4], mv[0])), "3.0")
check("math-log2", str(math.log2(mv[4])), "3.0")
check("math-log10", str(math.log10(100.0)), "2.0")
check("math-hypot", str(math.hypot(mv[5], mv[6])), "5.0")
check("math-hypot3", str(math.hypot(mv[0], mv[0], mv[0] + 1.0)), "4.123105625617661")
check("math-atan2", str(math.atan2(0.0, 1.0)), "0.0")
check("math-sin0", str(math.sin(0.0)), "0.0")
check("math-cos0", str(math.cos(0.0)), "1.0")
check("math-tan0", str(math.tan(0.0)), "0.0")
check("math-degrees", str(math.degrees(0.0)), "0.0")
check("math-radians", str(math.radians(0.0)), "0.0")
check("math-fmod", str(math.fmod(7.0, mv[5])), "1.0")
check("math-fmod-neg", str(math.fmod(-7.0, mv[5])), "-1.0")
check("math-copysign", str(math.copysign(1.0, -2.0)), "-1.0")
check("math-copysign-negzero", str(math.copysign(1.0, 0.0 * -1.0)), "-1.0")
check("math-gcd", math.gcd(12, 18), 6)
check("math-gcd3", math.gcd(12, 18, 30), 6)
check("math-lcm", math.lcm(4, 6), 12)
check("math-lcm3", math.lcm(4, 6, 10), 60)
check("math-factorial-5", math.factorial(5), 120)
check("math-factorial-20", math.factorial(20), 2432902008176640000)
# Past 20! the product leaves the exact-double range and finishes on the
# arbitrary-precision integer, which is what makes this row exact.
check("math-factorial-25", str(math.factorial(25)), "15511210043330985984000000")
check("math-factorial-30", str(math.factorial(30)),
      "265252859812191058636308480000000")
check_true("math-isnan", math.isnan(math.nan))
check("math-isnan-false", math.isnan(mv[0]), False)
check_true("math-isinf", math.isinf(math.inf))
check_true("math-isinf-neg", math.isinf(-math.inf))
check_true("math-isfinite", math.isfinite(mv[0]))
check("math-isfinite-inf", math.isfinite(math.inf), False)
check_true("math-isclose", math.isclose(1.0, 1.0))
check("math-isclose-nan", math.isclose(math.nan, math.nan), False)
check_true("math-isclose-inf", math.isclose(math.inf, math.inf))
check("math-expm1-0", str(math.expm1(0.0)), "0.0")
check("math-log1p-0", str(math.log1p(0.0)), "0.0")


def math_err(f):
    try:
        f()
        return "no-raise"
    except Exception as e:
        return type(e).__name__ + ": " + str(e)


# CPython 3.14's own texts, quoted from python3 rather than invented.
check("math-sqrt-domain", math_err(lambda: math.sqrt(-1.0)),
      "ValueError: expected a nonnegative input, got -1.0")
check("math-log-domain", math_err(lambda: math.log(0.0)),
      "ValueError: expected a positive input, got 0.0")
check("math-log2-domain", math_err(lambda: math.log2(0.0)),
      "ValueError: expected a positive input, got 0.0")
check("math-log10-domain", math_err(lambda: math.log10(-1.0)),
      "ValueError: expected a positive input, got -1.0")
check("math-asin-domain", math_err(lambda: math.asin(2.0)),
      "ValueError: expected a number in range from -1 up to 1, got 2.0")
check("math-acos-domain", math_err(lambda: math.acos(-2.0)),
      "ValueError: expected a number in range from -1 up to 1, got -2.0")
check("math-log1p-domain", math_err(lambda: math.log1p(-1.0)),
      "ValueError: expected argument value > -1, got -1.0")
check("math-log-base-1", math_err(lambda: math.log(5.0, 1.0)),
      "ZeroDivisionError: division by zero")
check("math-floor-inf", math_err(lambda: math.floor(math.inf)),
      "OverflowError: cannot convert float infinity to integer")
check("math-floor-nan", math_err(lambda: math.floor(math.nan)),
      "ValueError: cannot convert float NaN to integer")
check("math-factorial-negative", math_err(lambda: math.factorial(-1)),
      "ValueError: factorial() not defined for negative values")
check("math-unknown-member", math_err(lambda: math.nope(1.0)),
      "AttributeError: module 'math' has no attribute 'nope'")
# OverflowError joined the builtin hierarchy with the math module, so it must
# derive from ArithmeticError the way CPython's does.
check_true("math-overflow-is-arithmetic",
           issubclass(OverflowError, ArithmeticError))


# ----- SECTION: staticmethod / classmethod -----
#
# They are descriptors: the decorator only wraps, and the unwrapping happens at
# the class-member lookup - the one place that knows both the receiver and its
# class. Every row below is a shape CPython 3.14.6 was asked about directly.
class SmShape:
    kind = "shape"

    def __init__(self, n):
        self.n = n

    @classmethod
    def make(cls, n):
        return cls(n)

    @classmethod
    def named(cls):
        return cls.kind

    @staticmethod
    def helper(x):
        return x * 10

    def describe(self):
        return self.kind + str(self.n)


class SmCircle(SmShape):
    kind = "circle"


class SmSquare(SmShape):
    kind = "square"

    @classmethod
    def named(cls):
        return "sq:" + cls.kind


# cls is the ACTUAL class of the receiver, which is what makes an alternative
# constructor build the subclass rather than the base.
check("sm-cls-is-receiver-base", type(SmShape.make(1)).__name__, "SmShape")
check("sm-cls-is-receiver-sub", type(SmCircle.make(2)).__name__, "SmCircle")
check("sm-classmethod-on-class", SmShape.named(), "shape")
check("sm-classmethod-inherited", SmCircle.named(), "circle")
check("sm-classmethod-overridden", SmSquare.named(), "sq:square")
check("sm-classmethod-via-instance", SmCircle(1).named(), "circle")
check("sm-classmethod-via-instance-sub", SmSquare(1).named(), "sq:square")
# A staticmethod gets NO self, through the class or through an instance.
check("sm-static-on-class", SmShape.helper(2), 20)
check("sm-static-inherited", SmCircle.helper(3), 30)
check("sm-static-via-instance", SmCircle(0).helper(4), 40)
check("sm-ordinary-method-still-bound", SmCircle(5).describe(), "circle5")
check("sm-static-via-getattr", getattr(SmShape, "helper")(1), 10)
check("sm-static-via-getattr-instance", getattr(SmCircle(0), "helper")(2), 20)
check("sm-classmethod-via-getattr", getattr(SmCircle, "named")(), "circle")
check_true("sm-hasattr-static", hasattr(SmShape, "helper"))
check_true("sm-hasattr-classmethod", hasattr(SmShape, "make"))
check("sm-hasattr-absent", hasattr(SmShape, "nope"), False)


# The BARE form, not the decorator - `f = staticmethod(g)` in a class body.
class SmBare:
    g = staticmethod(lambda a: a * 2)
    h = classmethod(lambda cls, a: cls.__name__ + str(a))


check("sm-bare-static-class", SmBare.g(3), 6)
check("sm-bare-static-instance", SmBare().g(3), 6)
check("sm-bare-classmethod-class", SmBare.h(4), "SmBare4")
check("sm-bare-classmethod-instance", SmBare().h(4), "SmBare4")
check("sm-wrapper-type-static", type(staticmethod(len)).__name__, "staticmethod")
check("sm-wrapper-type-class", type(classmethod(len)).__name__, "classmethod")


# super() inside a classmethod: the zero-argument form reads the frame's first
# argument, which for a classmethod is the CLASS - and the ORIGINAL class, so
# C.who() still passes C to the base implementation.
class SmA:
    @classmethod
    def who(cls):
        return "A:" + cls.__name__


class SmB(SmA):
    @classmethod
    def who(cls):
        return "B<" + super().who() + ">"


class SmC(SmB):
    pass


check("sm-super-classmethod-base", SmA.who(), "A:SmA")
check("sm-super-classmethod-mid", SmB.who(), "B<A:SmB>")
check("sm-super-classmethod-leaf", SmC.who(), "B<A:SmC>")
check("sm-super-classmethod-instance", SmC().who(), "B<A:SmC>")


# cls.<attr> writes through to the class the call was made on - the shape an
# alternative-constructor registry uses.
class SmReg:
    items = []

    @classmethod
    def add(cls, v):
        cls.items.append(v)
        return len(cls.items)


check("sm-classmethod-mutates-class", SmReg.add(1), 1)
check("sm-classmethod-mutates-class-2", SmReg.add(2), 2)
check("sm-classmethod-state", str(SmReg.items), "[1, 2]")


# ----- SECTION: unbound method descriptors -----
#
# `list.count` read off the TYPE OBJECT is not a bound method: CPython answers
# <method 'count' of 'list' objects>, a descriptor with no __self__ that takes
# the receiver as its FIRST argument.
check("desc-repr-list-count", repr(list.count), "<method 'count' of 'list' objects>")
check("desc-repr-list-append", repr(list.append), "<method 'append' of 'list' objects>")
check("desc-repr-list-sort", repr(list.sort), "<method 'sort' of 'list' objects>")
check("desc-repr-str-upper", repr(str.upper), "<method 'upper' of 'str' objects>")
check("desc-repr-str-replace", repr(str.replace), "<method 'replace' of 'str' objects>")
check("desc-repr-dict-get", repr(dict.get), "<method 'get' of 'dict' objects>")
check("desc-repr-dict-keys", repr(dict.keys), "<method 'keys' of 'dict' objects>")
check("desc-repr-set-add", repr(set.add), "<method 'add' of 'set' objects>")
check("desc-type-name", type(list.count).__name__, "method_descriptor")
check("desc-name", list.count.__name__, "count")
check("desc-qualname", list.count.__qualname__, "list.count")
check_true("desc-objclass", list.count.__objclass__ is list)

dsc_xs = [3, 1, 2]
list.append(dsc_xs, 9)
check("desc-call-append", str(dsc_xs), "[3, 1, 2, 9]")
check("desc-call-count", list.count(dsc_xs, 9), 1)
check("desc-call-index", list.index(dsc_xs, 2), 2)
check("desc-call-str-replace", str.replace("aXbXc", "X", "-"), "a-b-c")
check("desc-call-str-split", str.split("a b c")[1], "b")
dsc_d = {"k": 1}
check("desc-call-dict-get", dict.get(dsc_d, "k"), 1)
check("desc-call-dict-get-default", dict.get(dsc_d, "z", 5), 5)
check("desc-call-dict-keys", str(list(dict.keys(dsc_d))), "['k']")
dsc_st = {1, 2}
set.add(dsc_st, 3)
check("desc-call-set-add", str(sorted(dsc_st)), "[1, 2, 3]")
# A descriptor read off a value is still the BOUND builtin method, unchanged.
check("desc-bound-still-bound", [1, 2, 2].count(2), 2)


def desc_err(f):
    try:
        f()
        return "no-raise"
    except Exception as e:
        return type(e).__name__ + ": " + str(e)


check("desc-no-receiver", desc_err(lambda: list.count()),
      "TypeError: unbound method list.count() needs an argument")
check("desc-no-self-attr", desc_err(lambda: list.count.__self__),
      "AttributeError: 'method_descriptor' object has no attribute '__self__'")
# The TYPE-OBJECT wording, which CPython uses for a type and not for a value.
check("desc-absent-name", desc_err(lambda: list.nope),
      "AttributeError: type object 'list' has no attribute 'nope'")


# ----- SECTION: the builtins module and module values -----
#
# len.__self__ IS the builtins module in CPython, and `builtins.len is len`
# holds - which is why the module's slots are filled from the very scope
# bindings the global names resolve to rather than from copies.
check("bi-module-repr", repr(builtins), "<module 'builtins' (built-in)>")
check("bi-module-type", type(builtins).__name__, "module")
check("bi-call-len", builtins.len([1, 2]), 2)
check("bi-call-abs", builtins.abs(-3), 3)
check("bi-call-sorted", str(builtins.sorted([2, 1])), "[1, 2]")
check("bi-call-str", builtins.str(12), "12")
check("bi-call-next", builtins.next(iter([7, 8])), 7)
check_true("bi-identity-len", builtins.len is len)
check_true("bi-identity-abs", builtins.abs is abs)
check_true("bi-identity-exception", builtins.ValueError is ValueError)
check_true("bi-identity-baseexception", builtins.BaseException is BaseException)
check_true("bi-len-self", len.__self__ is builtins)
check_true("bi-abs-self", abs.__self__ is builtins)
check_true("bi-getattr", getattr(builtins, "len") is len)

# A math member is a FIRST-CLASS VALUE, not only something callable through
# the module (`f = math.sqrt`).
check("mv-type", type(math).__name__, "module")
check("mv-repr-sqrt", repr(math.sqrt), "<built-in function sqrt>")
check("mv-repr-isclose", repr(math.isclose), "<built-in function isclose>")
check("mv-name", math.sqrt.__name__, "sqrt")
check("mv-qualname", math.sqrt.__qualname__, "sqrt")
check("mv-type-name", type(math.sqrt).__name__, "builtin_function_or_method")
check_true("mv-self-is-module", math.sqrt.__self__ is math)
mv_fns = [math.sqrt, math.floor, math.gcd, math.isnan]
check("mv-call-sqrt", str(mv_fns[0](9.0)), "3.0")
check("mv-call-floor", mv_fns[1](3.7), 3)
check("mv-call-gcd", mv_fns[2](12, 18), 6)
check("mv-call-isnan", mv_fns[3](1.0), False)
check("mv-getattr", getattr(math, "sqrt")(16.0), 4.0)
check_true("mv-hasattr", hasattr(math, "sqrt"))
check("mv-hasattr-absent", hasattr(math, "nope"), False)
check("mv-absent-name", desc_err(lambda: math.nope),
      "AttributeError: module 'math' has no attribute 'nope'")
# A MODULE is not a generator. The builtins module holds a member called `next`,
# so a structural "has next" test would call it one (layer 2's pyIsGen; only an
# -exe run sees it).
check("mv-module-not-generator", type(builtins).__name__, "module")



# ----- SECTION: division by zero -----
#
# `/`, `//`, `%` and divmod by a numeric zero (an int, a float, -0.0, False, or
# with an arbitrary precision left operand) raise ZeroDivisionError; CPython 3.14
# words all of them "division by zero".
def zd_err(f):
    try:
        return "no-raise " + str(f())
    except ArithmeticError as e:
        return type(e).__name__ + ": " + str(e)


zd_l = [7, 2.5, 10**30, True]
zd_z = [0, 0.0, -0.0, False]
zd_seen = []
for zl in zd_l:
    for zz in zd_z:
        zd_seen.append(zd_err(lambda: zl / zz))
        zd_seen.append(zd_err(lambda: zl // zz))
        zd_seen.append(zd_err(lambda: zl % zz))
        zd_seen.append(zd_err(lambda: divmod(zl, zz)))
check("zd-all-raise", zd_seen, ["ZeroDivisionError: division by zero"] * 64)
zd_x = zd_l[0]
try:
    zd_x //= zd_z[0]
    zd_aug = "no-raise"
except ZeroDivisionError as e:
    zd_aug = str(e)
check("zd-augmented", zd_aug, "division by zero")
check("zd-nonzero-still-divides", [zd_l[0] // 2, zd_l[0] % 3, zd_l[1] / 0.5], [3, 1, 5.0])


# ----- SECTION: cased letters with no simple case mapping -----
#
# ß, ĸ and the modifier letters are LOWERCASE to CPython although no single
# character maps them, so a word goes on past them; a titlecase digraph (ǅ) opens
# a word like an uppercase letter and is what title() and capitalize() write.
cz = ["straße", "Straße", "ĸa ĸb", "ªb cd", "ǆa", "ǅa", "ǈǉ " + chr(454)]
check("case-title-sharp-s", cz[0].title(), "Straße")
check("case-istitle-sharp-s", [cz[0].istitle(), cz[1].istitle()], [False, True])
check("case-capitalize-sharp-s", cz[0].capitalize(), "Straße")
check("case-islower-sharp-s", [cz[0].islower(), cz[0][4].islower(), cz[0][4].isupper()], [True, True, False])
check("case-title-kra", cz[2].title(), "ĸa ĸb")
check("case-title-ordinal", cz[3].title(), "ªb Cd")
check("case-title-digraph", [cz[4].title(), cz[4].capitalize(), cz[6].title()], ["ǅa", "ǅa", "ǈǉ ǅ"])
check("case-istitle-digraph", [cz[5].istitle(), cz[5].isupper(), cz[5].islower()], [True, False, False])
check("case-swapcase-ordinal", cz[3].swapcase(), "ªB CD")


# ----- SECTION: the set algebra methods -----
#
# Each takes ANY iterable (the operator forms insist on sets); the _update forms
# write back and answer None. hasattr answers what the call can do, on a value
# and on the builtin type object alike.
sa = {1, 2, 3}
sa_args = [{2, 3, 4}, [2, 9], (3, 1), "ab"]
check("set-issuperset", [sa.issuperset({2}), sa.issuperset(sa_args[1])], [True, False])
check("set-isdisjoint", [sa.isdisjoint({9}), sa.isdisjoint(sa_args[2]), sa.isdisjoint(sa_args[3])], [True, False, True])
check("set-intersection", [sorted(sa.intersection(sa_args[0])), sorted(sa.intersection(sa_args[1], sa_args[2])), sorted(sa.intersection())], [[2, 3], [], [1, 2, 3]])
check("set-difference", [sorted(sa.difference({2})), sorted(sa.difference([1], sa_args[2]))], [[1, 3], [2]])
check("set-symmetric-difference", sorted(sa.symmetric_difference([3, 4])), [1, 2, 4])
check("set-algebra-new-set", [type(sa.intersection([1])).__name__, sorted(sa)], ["set", [1, 2, 3]])
sb = {1, 2, 3, 4}
check("set-intersection-update", sb.intersection_update({1, 2, 9}), None)
check("set-intersection-update-in-place", sorted(sb), [1, 2])
sb.difference_update([1])
sb.symmetric_difference_update({2, 5})
check("set-update-forms-in-place", sorted(sb), [5])
check("set-operators", [sorted(sa & {2, 3, 4}), sorted(sa - {2}), sorted(sa ^ {3, 4}), sa >= {2}, sa > sa], [[2, 3], [1, 3], [1, 2, 4], True, False])
sa_names = ["issuperset", "isdisjoint", "intersection", "difference", "symmetric_difference", "intersection_update"]
check("set-hasattr", [hasattr(sa, n) for n in sa_names], [True] * 6)
check("set-hasattr-not-on-dict", [hasattr({}, n) for n in sa_names], [False] * 6)
check("type-hasattr", [hasattr(dict, "fromkeys"), hasattr(list, "count"), hasattr(set, "isdisjoint"), hasattr(dict, "nope")], [True, True, True, False])
check("set-unbound-method", sorted(getattr(set, "intersection")({1, 2}, [2])), [2])


# ----- SECTION: `except E as NAME` unbinds NAME -----
#
# CPython ends every handler with `NAME = None; del NAME`, however the handler is
# left and whether or not NAME was bound before the try.
def ub_read(f):
    try:
        return repr(f())
    except NameError as e:
        return type(e).__name__ + ": " + str(e)


ub_vals = [5, "x"]
ub_w = ub_vals[0]
try:
    raise ValueError(ub_vals[1])
except ValueError as ub_w:
    ub_w = 7
check("except-as-unbinds-prior-global", ub_read(lambda: ub_w), "NameError: name 'ub_w' is not defined")
for ub_i in range(3):
    try:
        raise ValueError(ub_i)
    except ValueError as ub_b:
        if ub_i == 1:
            break
check("except-as-unbinds-on-break", ub_read(lambda: ub_b), "NameError: name 'ub_b' is not defined")
try:
    try:
        raise ValueError(ub_vals[1])
    except ValueError as ub_r:
        raise
except ValueError:
    pass
check("except-as-unbinds-on-raise", ub_read(lambda: ub_r), "NameError: name 'ub_r' is not defined")


def ub_local():
    ub_e = ub_vals[0]
    try:
        raise KeyError(1)
    except KeyError as ub_e:
        seen = str(ub_e)
    try:
        return seen + " " + str(ub_e)
    except UnboundLocalError as e:
        return seen + " " + type(e).__name__


check("except-as-unbinds-local", ub_local(), "1 UnboundLocalError")
ub_g = "g"


def ub_global():
    global ub_g
    try:
        raise ValueError(ub_vals[1])
    except ValueError as ub_g:
        return str(ub_g)


check("except-as-global", [ub_global(), ub_read(lambda: ub_g)], ["x", "NameError: name 'ub_g' is not defined"])


# ----- SECTION: every resolution order ends with object -----
class MoA:
    pass


class MoB(MoA):
    pass


class MoE(ValueError):
    pass


mo_names = lambda ks: [k.__name__ for k in ks]
check("mro-ends-with-object", mo_names(MoB.__mro__), ["MoB", "MoA", "object"])
check_true("mro-object-identity", MoB.__mro__[-1] is object)
check("mro-exception", mo_names(MoE.__mro__), ["MoE", "ValueError", "Exception", "BaseException", "object"])
check("mro-builtin-types", [mo_names(object.__mro__), mo_names(int.__mro__), mo_names(bool.__mro__)],
      [["object"], ["int", "object"], ["bool", "int", "object"]])
check("bases-root-class", [mo_names(MoA.__bases__), mo_names(MoB.__bases__), mo_names(object.__bases__), mo_names(bool.__bases__)],
      [["object"], ["MoA"], [], ["int"]])
check("mro-method", mo_names(MoB.mro()), ["MoB", "MoA", "object"])


# ----- SECTION: no attribute can be set on a builtin value -----
#
# None, a number, a str, a list, a dict and a set have no __dict__, so a new
# attribute - written, set with setattr() or deleted - is a catchable
# AttributeError.
def na_set(v):
    try:
        v.k = 3
        return "set"
    except AttributeError as e:
        return str(e)


na_vals = [None, 5, 2.5, True, "s", [1], {}, {1}, 3j]
check("noattr-set", [na_set(v) for v in na_vals],
      ["'" + t + "' object has no attribute 'k' and no __dict__ for setting new attributes"
       for t in ["NoneType", "int", "float", "bool", "str", "list", "dict", "set", "complex"]])
na_none = na_vals[0]
try:
    setattr(na_none, "k", 1)
    na_got = "set"
except AttributeError as e:
    na_got = type(e).__name__
check("noattr-setattr", na_got, "AttributeError")
try:
    del na_none.k
    na_got = "deleted"
except AttributeError as e:
    na_got = str(e)
check("noattr-del", na_got, "'NoneType' object has no attribute 'k' and no __dict__ for setting new attributes")


class NaC:
    pass


na_c = NaC()
na_c.k = 1
check("noattr-instance-still-sets", na_c.k, 1)


# ----- SECTION: an arbitrary precision int meeting a float -----
#
# float(n) is the nearest double (half to even) and raises OverflowError past
# the double range; int and float compare EXACTLY; int(f) is exact past 2^53.
bf_e = [54, 80, 1023]
bf_big = [2 ** e for e in bf_e]
check("bigflo-float", [str(float(b)) for b in bf_big],
      ["1.8014398509481984e+16", "1.2089258196146292e+24", "8.98846567431158e+307"])
check("bigflo-equal", [b == float(b) for b in bf_big], [True, True, True])
bf_tie = [2 ** 80 + 2 ** 27, 2 ** 80 + 2 ** 27 + 1, 10 ** 23]
check("bigflo-half-even", [str(float(bf_tie[0])), str(float(bf_tie[1]))],
      ["1.2089258196146292e+24", "1.2089258196146294e+24"])
check("bigflo-exact-compare", [bf_tie[2] == 1e23, bf_tie[2] > 1e23, 1e23 < bf_tie[2], bf_tie[2] != 1e23], [False, True, True, True])
check("bigflo-arith", [str(bf_big[1] + 0.5), str(bf_big[1] * 1.0), str(bf_big[1] / 2.0), str(0.5 - bf_big[1])],
      ["1.2089258196146292e+24", "1.2089258196146292e+24", "6.044629098073146e+23", "-1.2089258196146292e+24"])
bf_over = [2 ** 1024, 10 ** 400]


def bf_err(f):
    try:
        return "no-raise " + str(f())
    except OverflowError as e:
        return "OverflowError: " + str(e)


check("bigflo-overflow", [bf_err(lambda: float(bf_over[0])), bf_err(lambda: bf_over[1] + 0.5)],
      ["OverflowError: int too large to convert to float"] * 2)
check("bigflo-int-of-float", [str(int(1e23)), str(int(-2.0 ** 60)), str(int(2.5))],
      ["99999999999999991611392", "-1152921504606846976", "2"])


# ----- SECTION: printf-style operands and arity -----
#
# %d %i %u take any real number and truncate it exactly; %x %X %o need an int;
# %e %f %g a real number; %c an int or one character. Everything else, and a
# wrong argument count, is CPython's catchable error.
def pc_err(f, a):
    try:
        return f % a
    except (TypeError, ValueError, OverflowError) as e:
        return type(e).__name__ + ": " + str(e)


pc_vals = [1e300, 2.7, None, "s", 10 ** 30, -255]
check("pct-d-float-exact", pc_err("%d", pc_vals[0]), "1" + "0" * 16 + "52504760255204420248704468581108159154915854115511802457988908195786371375080447864043704443832883878176942523235360430575644792184786706982848387200926575803737830233794788090059368953234970799945081119038967640880074652742780142494579258788820056842838115669472196386865459400540160")
check("pct-d-truncates", [pc_err("%d", pc_vals[1]), pc_err("%i", -2.7), pc_err("%u", True)], ["2", "-2", "1"])
check("pct-d-not-real", [pc_err("%d", pc_vals[2]), pc_err("%d", pc_vals[3]), pc_err("%d", {})],
      ["TypeError: %d format: a real number is required, not NoneType",
       "TypeError: %d format: a real number is required, not str",
       "TypeError: %d format: a real number is required, not dict"])
check("pct-d-inf-nan", [pc_err("%d", float("inf")), pc_err("%d", float("nan"))],
      ["OverflowError: cannot convert float infinity to integer", "ValueError: cannot convert float NaN to integer"])
check("pct-x-needs-int", [pc_err("%x", pc_vals[1]), pc_err("%o", pc_vals[3]), pc_err("%x", pc_vals[4]), pc_err("%#X", pc_vals[5]), pc_err("%#.4x", pc_vals[5])],
      ["TypeError: %x format: an integer is required, not float", "TypeError: %o format: an integer is required, not str",
       "c9f2c9cd04674edea40000000", "-0XFF", "-0x00ff"])
check("pct-f-real", [pc_err("%f", pc_vals[3]), pc_err("%.1f", pc_vals[4]), pc_err("%f", float("inf")), pc_err("%e", True)],
      ["TypeError: must be real number, not str", "1000000000000000019884624838656.0", "inf", "1.000000e+00"])
check("pct-c", [pc_err("%c", 65), pc_err("%c", pc_vals[1]), pc_err("%c", -7), pc_err("%c", "ab")],
      ["A", "TypeError: %c requires an int or a unicode character, not float",
       "OverflowError: %c arg not in range(0x110000)", "TypeError: %c requires an int or a unicode character, not a string of length 2"])
check("pct-int-precision", [pc_err("%.3d", 7), pc_err("%.3o", 8)], ["007", "010"])
check("pct-arity", [pc_err("%s %s %s", (1, 2)), pc_err("%(a)d", 5), pc_err("%q", 1), pc_err("abc %", 1)],
      ["TypeError: not enough arguments for format string", "TypeError: format requires a mapping",
       "ValueError: unsupported format character 'q' (0x71) at index 1", "ValueError: incomplete format"])


# ----- SECTION: the format mini-language -----
def fm(v, s):
    try:
        return format(v, s)
    except (TypeError, ValueError, OverflowError) as e:
        return type(e).__name__ + ": " + str(e)


fm_vals = [2.7, None, "s", 10 ** 30, 1234567, -255, 1234.5, True]
check("fmt-d-not-float", [fm(fm_vals[0], "d"), fm(fm_vals[2], "x"), fm(fm_vals[0], "s")],
      ["ValueError: Unknown format code 'd' for object of type 'float'",
       "ValueError: Unknown format code 'x' for object of type 'str'",
       "ValueError: Unknown format code 's' for object of type 'float'"])
check("fmt-unsupported", [fm(fm_vals[1], "d"), fm([1], ">8"), fm(fm_vals[1], "")],
      ["TypeError: unsupported format string passed to NoneType.__format__",
       "TypeError: unsupported format string passed to list.__format__", "None"])
check("fmt-int-exact", [fm(fm_vals[3], "d"), fm(fm_vals[3], "x"), fm(fm_vals[3], ","), fm(fm_vals[3], "e")],
      ["1000000000000000000000000000000", "c9f2c9cd04674edea40000000", "1,000,000,000,000,000,000,000,000,000,000", "1.000000e+30"])
check("fmt-int-forms", [fm(fm_vals[4], "_"), fm(65535, "_b"), fm(fm_vals[5], "#010x"), fm(fm_vals[7], ">8"), fm(fm_vals[7], ""), fm(65, "c")],
      ["1_234_567", "1111_1111_1111_1111", "-0x00000ff", "       1", "True", "A"])
check("fmt-int-errors", [fm(5, ".2d"), fm(5, ",x"), fm(5, "+c"), fm(5, "5.2xx")],
      ["ValueError: Precision not allowed in integer format specifier", "ValueError: Cannot specify ',' with 'x'.",
       "ValueError: Sign not allowed with integer format specifier 'c'",
       "ValueError: Invalid format specifier '5.2xx' for object of type 'int'"])
check("fmt-float-types", [fm(fm_vals[6], ","), fm(fm_vals[6], ",.2f"), fm(2.5, "%"), fm(12345.678, "e"), fm(1e-5, "g"), fm(1.0, "#g"), fm(1.0, "#.0f")],
      ["1,234.5", "1,234.50", "250.000000%", "1.234568e+04", "1e-05", "1.00000", "1."])
check("fmt-float-none-precision", [fm(12.5, ".3"), fm(123.0, ".3"), fm(1.0, ".3"), fm(1.5, ".0"), fm(1e16, "")],
      ["12.5", "1.23e+02", "1.0", "2e+00", "1e+16"])
check("fmt-padding", [fm(-5, "05"), fm(1234, "010,"), fm(1.5, "=+8"), fm(1.5, "*^9"), fm("x", "0>3"), fm(5, "x<05"), fm(float("inf"), "010")],
      ["-0005", "00,001,234", "+    1.5", "***1.5***", "00x", "5xxxx", "0000000inf"])
check("fmt-z", [fm(-0.0, "z.1f"), fm(-0.04, "z.1f"), fm(-0.5, "z.1f")], ["0.0", "0.0", "-0.5"])
check("fmt-str", [fm("abcdef", ".2"), fm("ab", "^6"), fm("ab", "=5"), fm("ab", "+")],
      ["ab", "  ab  ", "ValueError: '=' alignment not allowed in string format specifier",
       "ValueError: Sign not allowed in string format specifier"])
check("fmt-strformat-fstring", ["{:>6,}".format(fm_vals[4]), f"{fm_vals[0]:08.3f}", "{0!r:>5}".format("a")],
      ["1,234,567", "0002.700", "  'a'"])

print(f"features: {checks[0]} checks, {fails[0]} failures")
exit(fails[0])
