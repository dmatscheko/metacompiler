# Full-syntax test: Python (3.12 core grammar) - part 1 of 2.
#
# THE PYTHON RATCHET IS TWO FILES, so that neither -frozen interpreter leg comes
# near the IR interpreter's step brake (-max-steps, 100M per call). Sections 90, 91
# and 97 build ints of thousands of digits, which the interpreter's arbitrary precision
# arithmetic makes the costliest rows of all, so they have a file of their own:
#
#   tests/python-test-full.py     every other section
#   tests/python-test-full.2.py   sections 90, 91 and 97
#
# The section NUMBERS are global, the two files together run the assertions, and
# ./test.sh --full reports them as the units python and python2. Add a new section
# to tests/python-test-full.py, and keep the numbers globally unique across the two.
#
# This file belongs to the SECOND test group (./test.sh --full): it is NOT part
# of the default matrix. The goal of the metacompiler is to support the full
# languages; this file is the ratchet that measures how far the python grammars
# are. It walks the whole practical Python 3.12 syntax, one self-contained
# SECTION per language area. The --full runner runs the file, and whenever a
# grammar aborts it removes the section around the error and retries - so the
# report lists every unsupported section, not just the first.
#
# Conventions (shared by every *-test-full.* file):
#   - prologue (before the first SECTION marker): the check helper only
#   - each section: '# ===== SECTION <nn>: <name> =====', top-level,
#     self-contained, no references to other sections
#   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
#     and prints the summary line 'full: <checks> checks, <failures> failures'
#   - main() returns the failure count (exit 0 == full support, verified)
#
# Deliberately out of scope (not syntax, or unrunnable in this harness):
# eval/exec, __slots__-free
# introspection helpers, and builtins beyond the small set the feature test
# already leans on plus type/str/int/isinstance and the Exception hierarchy
# (ExceptionGroup appears once: except* cannot be exercised without it).
# Parent methods are therefore called explicitly (Base.m(self)) instead of via
# super(), and staticmethod/classmethod are replaced by a hand-rolled
# descriptor and plain functions. The ONE stdlib module used is `re` (section 22),
# which the shared regular-expression engine backs. Complex literals ARE included: they are core
# literals needing no library. Async SYNTAX is covered; running it needs an
# event loop, so async functions are only defined and inspected, never awaited.
#
# Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
# code), organized after the Python 3.12 language reference with the ANTLR
# grammars-v4 Python grammar as a coverage checklist.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 01: baseline =====
# Condensed re-assertion of the feature-matrix basics this file builds on.
def s01():
    n = 0
    for i in range(4):
        n += i
    check("bas1", n == 6)
    d = {"a": 1}
    d["b"] = d["a"] + 1
    check("bas2", d["b"] == 2 and "b" in d and len(d) == 2)
    xs = [1, 2]
    xs.append(3)
    check("bas3", xs[-1] == 3 and xs[1:][0] == 2)
    check("bas4", (5 if 3 > 2 else 9) == 5 and (0 or "x") == "x" and None is None)
    log = ""
    try:
        raise Exception("boom")
    except Exception as e:
        log += "c" + e.args[0]
    finally:
        log += "f"
    check("bas5", log == "cboomf")

# ===== SECTION 02: numeric literal forms =====
def s02():
    check("num1", 0xff == 255 and 0o17 == 15 and 0b1010 == 10)
    check("num2", 1_000_000 == 1000000 and 0x_ff == 255 and 0b_10_10 == 10)
    check("num3", .5 == 0.5 and 5. == 5.0 and 1_0.2_5 == 10.25)
    check("num4", 1e3 == 1000.0 and 2.5e-2 == 0.025 and 1_2e1 == 120.0)
    check("num5", 10000000000000000000000000001 - 10000000000000000000000000000 == 1)
    check("num6", 1j * 1j == -1 and (2 + 3j).real == 2.0 and (2 + 3j).imag == 3.0)
    check("num7", 7 // 2 == 3 and -7 // 2 == -4 and 7.5 // 2 == 3.0)
    check("num8", -7 % 2 == 1 and 7 % -2 == -1 and -7 == (-7 // 2) * 2 + (-7 % 2))

# ===== SECTION 03: string and bytes literals =====
def s03():
    check("str1", len(r"a\nb") == 4 and r'\t' == "\\t")
    check("str2", "\x41" == "A" and "A" == "A" and "\101" == "A")
    check("str3", "\N{BULLET}" == "•" and len("\0") == 1 and len("\U0001F600") == 1)
    check("str4", "ab" "cd" == "abcd" and ("one "
                                           "two") == "one two")
    check("str5", '''a"b'c''' == 'a"b\'c' and len("""x
y""") == 3)
    check("str6", "a\
b" == "ab")
    check("str7", b"AB"[0] == 65 and b"a" + b"b" == b"ab" and len(b"\x00\xff") == 2)
    check("str8", rb"\n" == b"\\n" and len(rb"\n") == 2)

# ===== SECTION 04: f-strings =====
def s04():
    x = 5
    check("fst1", f"x is {x}" == "x is 5" and f"{x} + {x} = {x + x}" == "5 + 5 = 10")
    check("fst2", f"{3.14159:.2f}" == "3.14" and f"{7:05d}" == "00007")
    check("fst3", f"{'hi':>4}" == "  hi" and f"{5:*^3}" == "*5*")
    w = 6
    check("fst4", f"{'hi':>{w}}" == "    hi")
    check("fst5", f"{x=}" == "x=5" and f"{x = }" == "x = 5")
    check("fst6", f"{'a'!r}" == "'a'" and f"a{1}" "b" == "a1b")
    check("fst7", f"o{f'i{1 + 1}'}" == "oi2")
    check("fst8", f"{"q" + "r"}" == "qr")  # 3.12: quotes may repeat inside
    check("fst9", f"""t{x}
u""" == "t5\nu")
    # A leading `{{` is the escaped brace even when a set display could follow.
    check("fst10", f"{{x}}" == "{x}" and f'{{x}}{x}' == "{x}5" and f"{{{x}}}" == "{5}" and f"{ {x} }" == "{5}")

# ===== SECTION 05: tuples, sets and slicing =====
def s05():
    t = (1, 2, 3)
    check("tup1", t[1] == 2 and len(t) == 3 and t[-1] == 3)
    one = 7,
    check("tup2", one == (7,) and len(()) == 0 and ((1, 2), (3, 4))[1][0] == 3)
    s = {1, 2, 2, 3}
    check("set1", len(s) == 3 and 2 in s and 9 not in s)
    check("set2", ({1, 2} | {2, 3}) == {1, 2, 3} and ({1, 2} & {2, 3}) == {2})
    check("set3", ({1, 2, 3} - {2}) == {1, 3} and ({1, 2} ^ {2, 3}) == {1, 3})
    xs = [0, 1, 2, 3, 4, 5]
    check("slc1", xs[1:5:2] == [1, 3] and xs[::2] == [0, 2, 4] and xs[::-1][0] == 5)
    check("slc2", xs[-3:-1] == [3, 4] and xs[4:1:-1] == [4, 3, 2])
    ys = [0, 1, 2, 3]
    ys[1:3] = [9]
    zs = [0, 1, 2, 3]
    zs[::2] = [7, 8]
    check("slc3", ys == [0, 9, 3] and zs == [7, 1, 8, 3])
    del zs[1]
    ws = [0, 1, 2, 3, 4]
    del ws[::2]
    check("slc4", zs == [7, 8, 3] and ws == [1, 3])

# ===== SECTION 06: unpacking and starred expressions =====
def s06():
    a, *b = [1, 2, 3]
    *c, d = (1, 2, 3)
    check("unp1", a == 1 and b == [2, 3] and c == [1, 2] and d == 3)
    e, *f, g = range(5)
    check("unp2", e == 0 and f == [1, 2, 3] and g == 4)
    (h, (i, j)) = (1, (2, 3))
    check("unp3", h + i + j == 6)
    def add3(p, q, r):
        return p + q + r
    check("unp4", add3(*[1, 2, 3]) == 6 and add3(1, *(2, 3)) == 6)
    check("unp5", add3(**{"p": 1, "q": 2, "r": 3}) == 6 and add3(1, **{"r": 5, "q": 2}) == 8)
    check("unp6", [0, *[1, 2]] == [0, 1, 2] and (0, *(1, 2)) == (0, 1, 2) and {0, *{1}} == {0, 1})
    merged = {**{"x": 1, "y": 2}, "z": 3, **{"x": 9}}
    check("unp7", merged == {"x": 9, "y": 2, "z": 3})
    m = n = 3
    check("unp8", m == 3 and n == 3)
    total = 0
    for head, *tail in [[1, 2, 3], [4]]:
        total += head + len(tail)
    check("unp9", total == 7)

# ===== SECTION 07: chained comparison, walrus and conditional expressions =====
def s07():
    check("cmp1", 1 < 2 < 3 and 3 > 2 > 1 and 1 <= 1 < 2)
    check("cmp2", not (1 < 5 < 3) and (3 == 3 == 3))
    calls = [0]
    def mid():
        calls[0] += 1
        return 2
    check("cmp3", 1 < mid() < 3 and calls[0] == 1)
    check("wal1", (w := 5) + w == 10)
    k = 3
    steps = 0
    while (k := k - 1) > 0:
        steps += 1
    check("wal2", steps == 2 and k == 0)
    check("wal3", [y for x in [1, 2, 3] if (y := x * 2) > 2] == [4, 6])
    check("cnd1", ("a" if False else "b") == "b" and ("neg" if -1 < 0 else "pos") == "neg")
    grade = "A" if 87 >= 90 else "B" if 87 >= 80 else "C"
    check("cnd2", grade == "B")

# ===== SECTION 08: lambdas =====
def s08():
    lam = lambda x, y=10: x + y
    check("lam1", lam(1) == 11 and lam(1, 2) == 3)
    var = lambda *args: len(args)
    check("lam2", var(1, 2, 3) == 3)
    kw = lambda **kws: kws["z"] + len(kws)
    check("lam3", kw(z=9) == 10)
    check("lam4", (lambda v: v * 3)(4) == 12)
    cur = lambda p: lambda q: p + q
    check("lam5", cur(1)(2) == 3)
    tag = lambda n: "e" if n % 2 == 0 else "o"
    check("lam6", tag(4) + tag(5) == "eo")
    ops = {"inc": lambda v: v + 1, "dbl": lambda v: v * 2}
    check("lam7", ops["inc"](4) == 5 and ops["dbl"](4) == 8)

# ===== SECTION 09: function signatures =====
def s09():
    def pos_only(a, b, /, c):
        return a * 100 + b * 10 + c
    check("sig1", pos_only(1, 2, 3) == 123 and pos_only(1, 2, c=4) == 124)
    def kw_only(a, *, b=2, c):
        return a + b + c
    check("sig2", kw_only(1, c=3) == 6 and kw_only(1, b=5, c=3) == 9)
    def both(a, /, b, *, c):
        return f"{a}{b}{c}"
    check("sig3", both(1, 2, c=3) == "123" and both(1, b=2, c=3) == "123")
    def variadic(a, *args, mid=5, **kws):
        return a + len(args) + mid + len(kws)
    check("sig4", variadic(1, 2, 3, x=1, y=2) == 10)
    def sub(x, y):
        return x - y
    check("sig5", sub(y=1, x=5) == 4)
    def accum(v, bag=[]):  # the shared-mutable-default semantics, on purpose
        bag.append(v)
        return len(bag)
    check("sig6", accum(1) == 1 and accum(2) == 2 and accum(9, []) == 1)
    def ann(x: int, y: "str" = "s") -> int:
        z: int = len(y)
        return x + z
    check("sig7", ann(3) == 4)
    def trail(a, b,):
        return a + b
    check("sig8", trail(1, 2,) == 3)
    # A *args / **kwargs def uses the extended prologue layout
    # [p0..pn-1, *args, **kwargs], so the bound argument array is
    # names.length + 2 long whatever the call looks like. sig11 and sig16 are
    # the calls whose POSITIONAL count hits that same number - the one case a
    # length test cannot tell from "binding was a no-op".
    def st_args(a, *rest, **kws):
        return a * 100 + len(rest) * 10 + len(kws)
    check("sig9", st_args(1) == 100)
    check("sig10", st_args(1, 2) == 110)
    check("sig11", st_args(1, 2, 3) == 120)
    check("sig12", st_args(1, 2, 3, 4) == 130)
    check("sig13", st_args(1, x=5) == 101)
    check("sig14", st_args(1, 2, x=5) == 111)
    def st_args2(a, b, *rest, **kws):
        return a + b * 10 + len(rest) * 100 + len(kws) * 1000
    check("sig15", st_args2(1, 2) == 21)
    check("sig16", st_args2(1, 2, 3, 4) == 221)

# ===== SECTION 10: closures and scopes =====
def s10():
    def make_counter():
        n = 0
        def bump():
            nonlocal n
            n += 1
            return n
        return bump
    c1 = make_counter()
    c1()
    check("scp1", c1() == 2 and make_counter()() == 1)
    def outer():
        v = "o"
        def middle():
            def inner():
                nonlocal v
                v = "i"
            inner()
        middle()
        return v
    check("scp2", outer() == "i")
    late = []
    for i in range(3):
        late.append(lambda: i)  # late binding: all three see the final i
    check("scp3", late[0]() == 2 and late[2]() == 2)
    fixed = []
    for i in range(3):
        fixed.append(lambda i=i: i)  # the default-argument capture idiom
    check("scp4", fixed[0]() == 0 and fixed[2]() == 2)
    def set_global():
        global g10_val
        g10_val = 7
    set_global()
    check("scp5", g10_val == 7)
    leak = 99
    check("scp6", [leak for leak in range(3)] == [0, 1, 2] and leak == 99)

# ===== SECTION 11: decorators =====
def s11():
    check("dec1", three() == 6)
    check("dec2", one() == 4)      # stacked: twice(add_one(f))
    check("dec3", ten() == 50)     # decorator factory with an argument
    check("dec4", Tagged.tag == "yes")
    check("dec5", pick() == 14)    # PEP 614: any expression as decorator
def twice(f):
    def wrap(*args):
        return f(*args) * 2
    return wrap
def add_one(f):
    def wrap():
        return f() + 1
    return wrap
def times(k):
    def deco(f):
        def wrap():
            return f() * k
        return wrap
    return deco
def tag_class(cls):
    cls.tag = "yes"
    return cls
registry = {"t": twice}
@twice
def three():
    return 3
@twice
@add_one
def one():
    return 1
@times(5)
def ten():
    return 10
@tag_class
class Tagged:
    pass
@registry["t"]
def pick():
    return 7

# ===== SECTION 12: generators =====
def s12():
    def gen():
        yield 1
        yield 2
        yield from [3, 4]
        return 99
    out = []
    for v in gen():
        out.append(v)
    check("gen1", out == [1, 2, 3, 4])
    it = gen()
    check("gen2", it.send(None) == 1 and it.send(None) == 2)
    def echo():
        got = yield 1
        yield got * 2
    e = echo()
    e.send(None)
    check("gen3", e.send(21) == 42)
    fin = gen()
    fin.send(None); fin.send(None); fin.send(None); fin.send(None)
    ret = 0
    try:
        fin.send(None)
    except Exception as stop:  # StopIteration carries the return value
        ret = stop.value
    check("gen4", ret == 99)
    check("gen5", list(v * v for v in range(4)) == [0, 1, 4, 9])
    ge = (v + 1 for v in [1, 2])
    tot = 0
    for v in ge:
        tot += v
    check("gen6", tot == 5)

# ===== SECTION 13: classes and inheritance =====
def s13():
    a = Animal("rex")
    check("cls1", a.speak() == "rex speaks" and a.__repr__() == "<rex>")
    check("cls2", Animal.kind == "generic" and a.kind == "generic")
    a.kind = "dog"  # instance attribute shadows the class attribute
    check("cls3", a.kind == "dog" and Animal.kind == "generic")
    d = Dog("fido")
    check("cls4", d.name == "fido!" and d.speak() == "woof fido! speaks")
    check("cls5", isinstance(d, Animal) and isinstance(d, Dog) and not isinstance(a, Dog))
    check("cls6", type(d) is Dog and type(a) is Animal)
    check("cls7", First().who() == "A" and Second().who() == "B")  # MRO order
    Animal.seen.append(1)
    b = Animal("b")
    check("cls8", len(b.seen) == 1 and b.seen is a.seen)
class Animal:
    kind = "generic"
    seen = []
    def __init__(self, name):
        self.name = name
    def speak(self):
        return self.name + " speaks"
    def __repr__(self):
        return "<" + self.name + ">"
class Dog(Animal):
    def __init__(self, name):
        Animal.__init__(self, name + "!")
    def speak(self):
        return "woof " + Animal.speak(self)
class MixA:
    def who(self): return "A"
class MixB:
    def who(self): return "B"
class First(MixA, MixB):
    pass
class Second(MixB, MixA):
    pass

# ===== SECTION 14: operator overloading =====
def s14():
    v1 = Vec(1, 2)
    v2 = Vec(3, 4)
    check("ops1", (v1 + v2) == Vec(4, 6))
    check("ops2", v1 != v2 and v1 < v2 and not (v2 < v1))
    check("ops3", v1[0] == 1 and v1[1] == 2)
    m = Vec(5, 6)
    m[0] = 7
    check("ops4", m[0] == 7 and m == Vec(7, 6))
    check("ops5", 2 in v1 and 9 not in v1)
    check("ops6", len(Vec(3, 4)) == 7)
    check("ops7", v1(10) == Vec(10, 20))
    acc = Vec(1, 1)
    acc @= v2  # falls back to acc = acc.__matmul__(v2)
    check("ops8", v1 @ v2 == 11 and acc == 7)
    check("ops9", -v1 == Vec(-1, -2))
class Vec:
    def __init__(self, x, y): self.x = x; self.y = y
    def __add__(self, o): return Vec(self.x + o.x, self.y + o.y)
    def __eq__(self, o): return isinstance(o, Vec) and self.x == o.x and self.y == o.y
    def __lt__(self, o): return self.x + self.y < o.x + o.y
    def __getitem__(self, idx): return self.x if idx == 0 else self.y
    def __setitem__(self, idx, val):
        if idx == 0: self.x = val
        else: self.y = val
    def __contains__(self, v): return v == self.x or v == self.y
    def __len__(self): return self.x + self.y
    def __call__(self, k): return Vec(self.x * k, self.y * k)
    def __matmul__(self, o): return self.x * o.x + self.y * o.y
    def __neg__(self): return Vec(-self.x, -self.y)

# ===== SECTION 15: descriptors and class machinery =====
def s15():
    b = Box()
    b.val = 4
    check("dsc1", b._v == 5 and b.val == 10)  # __set__ then __get__
    s = Slim()
    s.a = 1
    s.b = 2
    check("slt1", s.a + s.b == 3)
    blocked = False
    try:
        s.c = 3  # not in __slots__
    except Exception:
        blocked = True
    check("slt2", blocked)
    check("cbd1", Cfg.mode == "fast" and Cfg.limit == 60 and Cfg.double(5) == 10)
    check("cbd2", "_v" in Box().__dict__ and Box.__name__ == "Box")
class Cell:
    def __get__(self, obj, owner): return obj._v * 2
    def __set__(self, obj, val): obj._v = val + 1
class Box:
    val = Cell()  # a hand-rolled property
    def __init__(self): self._v = 0
class Slim:
    __slots__ = ("a", "b")
class Cfg:  # a class body is a suite: statements are allowed
    mode = "fast" if True else "slow"
    limit = 0
    for _step in range(3):
        limit += 20
    def double(v): return v * 2

# ===== SECTION 16: match statement =====
def s16():
    check("mat1", describe(0) == "zero" and describe(99) == "limit")
    check("mat2", describe(2) == "small" and describe(9.5) == "other")
    check("mat3", describe([7]) == "one:7" and describe((8,)) == "one:8")
    check("mat4", describe([1, 2, 3]) == "seq:1+2")
    check("mat5", describe({"op": "+", "z": 1}) == "op+:1")
    check("mat6", describe(Pt(0, 5)) == "y-axis:5")
    check("mat7", describe(Pt(3, 3)) == "diag" and describe(Pt(1, 2)) == "pt:1,2")
    check("mat8", describe("hi") == "s:hi")
class Pt:
    __match_args__ = ("x", "y")
    def __init__(self, x, y): self.x = x; self.y = y
class K:
    LIMIT = 99
def describe(v):
    match v:
        case 0: return "zero"
        case K.LIMIT: return "limit"           # value pattern (dotted name)
        case 1 | 2: return "small"             # or-pattern
        case str() as sv: return "s:" + sv     # class pattern + capture
        case [x]: return f"one:{x}"            # matches lists AND tuples
        case [x, y, *rest]: return f"seq:{x}+{y}"
        case {"op": o, **extra}: return f"op{o}:{len(extra)}"
        case Pt(x=0, y=yy): return f"y-axis:{yy}"
        case Pt(a, b) if a == b: return "diag" # guard; positional needs __match_args__
        case Pt(a, b): return f"pt:{a},{b}"
        case _: return "other"

# ===== SECTION 17: context managers =====
def s17():
    log = []
    with Ctx(log, "a") as got:
        log.append("in" + got)
    check("ctx1", log == ["+a", "ina", "-a"])
    log2 = []
    with Ctx(log2, "a") as x, Ctx(log2, "b") as y:
        log2.append(x + y)
    check("ctx2", log2 == ["+a", "+b", "ab", "-b", "-a"])
    log3 = []
    with (Ctx(log3, "a") as x, Ctx(log3, "b")):  # 3.10 parenthesized form
        log3.append("m")
    check("ctx3", log3 == ["+a", "+b", "m", "-b", "-a"])
    ran = False
    with Quiet():  # __exit__ returns True: the exception is swallowed
        raise Exception("swallowed")
        ran = True
    check("ctx4", not ran)
    seen = []
    try:
        with Ctx(seen, "z"):
            raise Exception("esc")
    except Exception as e:
        seen.append(e.args[0])
    check("ctx5", seen == ["+z", "-z", "esc"])
class Ctx:
    def __init__(self, log, tag): self.log = log; self.tag = tag
    def __enter__(self): self.log.append("+" + self.tag); return self.tag
    def __exit__(self, et, ev, tb): self.log.append("-" + self.tag); return False
class Quiet:
    def __enter__(self): return self
    def __exit__(self, et, ev, tb): return True

# ===== SECTION 18: exception machinery =====
def s18():
    check("exc1", trip(SubError("bad")) == "app:bad")
    check("exc2", trip(OtherError("x")) == "pair")
    tags = []
    try:
        try:
            raise SubError("inner")
        except AppError as e:
            tags.append(e.args[0])
            raise  # bare re-raise of the active exception
    except SubError as e2:
        tags.append("re:" + e2.args[0])
    check("exc3", tags == ["inner", "re:inner"])
    cause = None
    try:
        try:
            raise AppError("low")
        except AppError as e:
            raise OtherError("high") from e
    except OtherError as e2:
        cause = e2.__cause__
    check("exc4", isinstance(cause, AppError) and cause.args[0] == "low")
    path = ""
    try:
        path += "t"
    except AppError:
        path += "x"
    else:
        path += "e"
    finally:
        path += "f"
    check("exc5", path == "tef")
    check("exc6", CodedError("m", 7).code == 7 and isinstance(CodedError("m", 7), AppError))
    groups = []
    try:
        raise ExceptionGroup("g", [SubError("s"), OtherError("o")])
    except* SubError as gs:
        groups.append(f"S{len(gs.exceptions)}")
    except* OtherError as go:
        groups.append(f"O{len(go.exceptions)}")
    check("exc7", groups == ["S1", "O1"])
    # A raise inside an except body chains the handled exception as __context__;
    # `from` sets __cause__ and suppresses the context; nothing chained reads None.
    chain = []
    for frm in [False, True]:
        try:
            try:
                raise AppError("first")
            except AppError as e:
                if frm:
                    raise OtherError("second") from None
                raise OtherError("second")
        except OtherError as e2:
            chain.append((repr(e2.__context__), e2.__cause__, e2.__suppress_context__))
    try:
        try:
            raise AppError("only")
        except AppError as e3:
            raise
    except AppError as e4:
        chain.append((e4.__context__, e4.__cause__, e4.__suppress_context__, hasattr(e4, "__context__")))
    check("exc8", chain == [("AppError('first')", None, False), ("AppError('first')", None, True), (None, None, False, True)])
class AppError(Exception):
    pass
class SubError(AppError):
    pass
class OtherError(Exception):
    pass
class CodedError(AppError):
    def __init__(self, msg, code):
        AppError.__init__(self, msg)
        self.code = code
def trip(err):
    try:
        raise err
    except SubError as e:
        return "app:" + e.args[0]
    except (AppError, OtherError):  # a tuple of exception classes
        return "pair"

# ===== SECTION 19: loop else and small statement forms =====
def s19():
    hits = ""
    for i in range(3):
        hits += f"{i}"
    else:  # runs because the loop was not broken
        hits += "!"
    check("els1", hits == "012!")
    found = ""
    for i in range(5):
        if i == 2:
            found = "brk"
            break
    else:
        found = "none"
    check("els2", found == "brk")
    w = 0
    while w < 2:
        w += 1
    else:
        w += 10
    check("els3", w == 12)
    a = 1; b = 2; a += b
    check("stm1", a == 3)
    if True: t = 5
    check("stm2", t == 5)
    def inline(): return 3
    def todo(): ...
    check("stm3", inline() == 3 and todo() is None)
    ell = ...
    check("stm4", ell is ...)
    v = 1
    del v
    gone = False
    try:
        v += 1
    except Exception:  # UnboundLocalError after del
        gone = True
    check("stm5", gone)
    x1, y1 = 1, 2
    del x1, y1
    gone2 = False
    try:
        y1 += 1
    except Exception:
        gone2 = True
    check("stm6", gone2)

# ===== SECTION 20: type hint syntax =====
def s20():
    check("typ1", first([3, 4]) == 3 and firstname("ab") == "ab")
    v: int | None = None
    check("typ2", v is None)
    w: str  # a bare annotation is a statement of its own
    w = "s"
    check("typ3", w == "s")
    check("typ4", tf([1, 2]) == [1, 2] and tf([1], {"a": 1}) == [1])
    check("typ5", IntList.__name__ == "IntList" and IntList.__value__ == list[int])
    p = Pair(5, "x")
    check("typ6", p.a == 5 and p.b == "x")
    check("typ7", Rec.n == 3 and "n" in Rec.__annotations__)
type IntList = list[int]  # 3.12 type-alias statement
def tf(xs: list[int], m: dict[str, int] | None = None, t: tuple[int, ...] = ()) -> list[int]:
    return xs
def first[T](xs: list[T]) -> T:  # 3.12 generic function (PEP 695)
    return xs[0]
def firstname(s: "str") -> "str":  # string annotations
    return s
class Pair[A, B]:  # 3.12 generic class
    def __init__(self, a: A, b: B):
        self.a = a
        self.b = b
class Rec:
    n: int = 3

# ===== SECTION 21: async syntax =====
# Defined and inspected only: running these needs an event loop.
def s21():
    check("asy1", type(af).__name__ == "function" and af.__name__ == "af")
    check("asy2", wa.__name__ == "wa" and loop_all.__name__ == "loop_all")
    check("asy3", agen.__name__ == "agen")
    check("asy4", comp.__name__ == "comp")
    check("asy5", Waiter().ping() == "pong" and Waiter.fetch.__name__ == "fetch")
async def af():
    return 5
async def wa(p):
    r = await p
    return r + 1
async def loop_all(xs, ctx):
    total = 0
    async with ctx as c:
        async for v in xs:
            total += v + c
    return total
async def agen(n):
    yield n
    yield n + 1
async def comp(xs):
    return [v * 2 async for v in xs]
class Waiter:
    async def fetch(self):
        return await af()
    def ping(self):
        return "pong"

# ===== SECTION 22: regular expressions (the re module) =====
# Every assertion here was checked against CPython 3.14 first. `re` is the only
# stdlib module this file uses; the matcher behind it is the shared engine in
# languages/lib/regex.js (and its Go twin abnf/jsrtregex.go for the compiler).
import re

def s22():
    m = re.search(r"(\d+)-(\d+)", "ab 12-34 cd")
    check("re01", m.group(0) == "12-34" and m.group(1) == "12" and m.group(2) == "34")
    check("re02", m.start() == 3 and m.end() == 8 and m.span() == (3, 8))
    check("re03", m.span(1) == (3, 5) and len(m.groups()) == 2)
    n = re.search(r"(?P<k>[a-z]+)=(?P<v>\d+)", "x: size=42")
    check("re04", n.group("k") == "size" and n.group("v") == "42")
    check("re05", n.groupdict()["k"] == "size" and n.groupdict()["v"] == "42")
    # match anchors at the start, fullmatch at both ends - and fullmatch must
    # BACKTRACK into an alternation, which a leftmost search plus a length check
    # cannot do: re.fullmatch("a|ab", "ab") is "ab", not None.
    check("re06", re.match(r"\d", "a1") == None and re.match(r"a", "a1").group(0) == "a")
    check("re07", re.fullmatch(r"a|ab", "ab").group(0) == "ab" and re.fullmatch(r"a", "ab") == None)
    check("re08", re.findall(r"\d+", "a1 b22 c333") == ["1", "22", "333"])
    check("re09", re.findall(r"(\w)=(\d)", "a=1,b=2") == [("a", "1"), ("b", "2")])
    check("re10", re.sub(r"\s+", " ", "a   b \t c") == "a b c")
    check("re11", re.sub(r"(\w)(\d)", r"\2\1", "a1 b2") == "1a 2b")
    check("re12", re.sub(r"(\d)", r"[\g<1>]", "a1") == "a[1]")
    check("re13", re.subn(r"o", "0", "foo boo") == ("f00 b00", 4))
    check("re14", re.split(r"\s*,\s*", "a , b,c") == ["a", "b", "c"])
    # A capture group in the separator is SPLICED into the result (Python's rule).
    check("re15", re.split(r"(,)", "a,b") == ["a", ",", "b"])
    check("re16", re.split(r"\d", "a1b2c", 1) == ["a", "b2c"])
    # The flag bitmask, mapped onto the engine's neutral letters.
    p = re.compile(r"^ab", re.I | re.M)
    check("re17", p.pattern == "^ab" and len(p.findall("xx\nABc\nabz")) == 2)
    check("re18", re.search(r"a.c", "a\nc", re.S).group(0) == "a\nc")
    check("re19", re.search(r"a.c", "a\nc") == None)
    check("re20", re.escape("a.b") == "a\\.b")
    total = 0
    for it in re.finditer(r"\d", "a1b2c3"):
        total += int(it.group(0))
    check("re21", total == 6)
    # A callable replacement is the interpreter's own job, not the engine's.
    check("re22", re.sub(r"\d", lambda z: "<" + z.group(0) + ">", "a1b2") == "a<1>b<2>")
    # Backreferences, in the pattern and in Python's own (?P=name) spelling.
    check("re23", re.search(r"(ab)\1", "xabab").group(0) == "abab")
    check("re24", re.search(r"(?P<w>ab)(?P=w)", "abab").group(0) == "abab")
    check("re25", re.search(r"x", "abc") == None)
    # Lookahead, lazy and bounded quantifiers, classes and alternation.
    check("re26", re.search(r"\d+(?=px)", "w 24px").group(0) == "24")
    check("re27", re.search(r"<(.+?)>", "<a><b>").group(1) == "a")
    check("re28", re.findall(r"[bc]{2,3}", "abccbcx") == ["bcc", "bc"])
    check("re29", re.search(r"\bcat\b", "a cat.").group(0) == "cat")
    check("re30", re.compile(r"[A-Z]+").search("xxYZz").group(0) == "YZ")
    # re.X: unescaped whitespace and # comments in the PATTERN are ignored.
    check("re31", re.search(r"\d+ [a-z]  # a number then a letter", "w 24px", re.X).group(0) == "24p")
    check("re32", re.search(r"(\d)(\w)", "a1b").expand(r"<\2\1>") == "<b1>")
    check("re33", re.compile(r"a", re.I).search("A").group(0) == "A")
    # match/fullmatch anchor by WRAPPING the pattern, and under re.X a trailing
    # comment would otherwise swallow the wrapper's own closing paren.
    check("re34", re.fullmatch(r"\d+ # num", "24", re.X).group(0) == "24")
    check("re35", re.match(r"\d+ # num", "24x", re.X).group(0) == "24")
    # The pos argument: match/fullmatch anchor at pos, but ^ still means the real
    # start of the string (CPython's documented rule).
    q = re.compile(r"\d+")
    check("re36", q.match("ab12", 2).group(0) == "12" and q.match("ab12", 0) == None)
    check("re37", q.fullmatch("ab12", 2).group(0) == "12")
    check("re38", re.compile(r"^b").match("ab", 1) == None)
    # (?P=name) is a named backreference; the group it names may be past number 9.
    ten = r"(?P<a>x)(?P<b>y)(?P<c>z)(?P<d>1)(?P<e>2)(?P<f>3)(?P<g>4)(?P<h>5)(?P<i>6)(?P<j>7)(?P=j)"
    check("re39", re.search(ten, "xyz123456777").group(0) == "xyz12345677")
    check("re40", re.search(ten, "xyz1234567 7") == None)

# ===== SECTION 23: interpreter/compiler agreement ratchet =====
# Every check here is a place where python-interpreter.abnf and
# python-to-llvm-ir.abnf can disagree. The matrix cannot see this class of bug - it
# compares each engine against itself under goja and -frozen - so the assertions
# live here. Expected values are CPython 3's.
def s23():
    # Sequence repetition, not numeric arithmetic.
    check("agr1", "ab" * 3 == "ababab")
    check("agr2", [0] * 3 == [0, 0, 0])
    check("agr3", 3 * [0] == [0, 0, 0])
    check("agr4", [1, 2] * 0 == [])
    check("agr5", "x" * 0 == "")
    # Conversions both halves must have.
    check("agr6", bool(0) == False and bool([]) == False and bool("x") == True)
    check("agr7", float("1.5") == 1.5 and float(2) == 2)
    check("agr8", repr("x") == "'x'")
    check("agr9", len(set([1, 2, 2])) == 2)
    check("agr10", sum([1, 2, 3]) == 6 and sum([]) == 0)
    check("agr11", len(tuple([1, 2])) == 2)
    # max/min, in both halves.
    check("agr12", max(1, 2) == 2 and min(1, 2) == 1)
    check("agr13", max([3, 1, 2]) == 3 and min([3, 1, 2]) == 1)

# ===== SECTION 24: the syntax the reference corpus asked for =====
# Every check below is a construct CPython's OWN test suite writes. Grouped by
# the gap each one closes.
def s24():
    # An import is a SMALL statement, not only a module-level one: inside a
    # function and inside a try are how CPython's suite writes most of them.
    import re
    try:
        import re
    except ImportError:
        pass
    check("corp01", re.search(r"\d", "a1").group(0) == "1")

    # A bare tuple after return and after yield, and a parameter list written
    # across lines with an empty-tuple default.
    def pair(a,
             b=(),
             c=None):
        return a, b
    check("corp02", pair(1) == (1, ()))

    def gen():
        yield 1, 2
    check("corp03", list(gen()) == [(1, 2)])

    # An expression statement may itself be a bare tuple.
    1, 2, 3

    # A subscript takes an expression LIST, on the read and on the write side.
    class Grid:
        def __init__(self):
            self.cells = {}
        def __setitem__(self, k, v):
            self.cells[str(k)] = v
        def __getitem__(self, k):
            return self.cells[str(k)]
    g = Grid()
    g[1, 2] = "pair"
    check("corp04", g[1, 2] == "pair")

    # The iterable of a for statement is an expression list too.
    seen = []
    for v in 1, 2, 3:
        seen.append(v)
    check("corp05", seen == [1, 2, 3])

    # Comprehensions: several `for` clauses, interleaved `if`s, a tuple target,
    # a starred rest in the target and a target that is a SUBSCRIPT.
    check("corp06", [a for b in ["xy", "z"] for a in b] == ["x", "y", "z"])
    check("corp07", {k: v for k in range(3) for v in range(3) if k == v} == {0: 0, 1: 1, 2: 2})
    check("corp08", [line for _, _, line in [(1, 2, 3), (4, 5, 6)]] == [3, 6])
    check("corp09", [kind for (kind, s, *rest) in [(1, 2, 3, 4), (5, 6, 7)]] == [1, 5])
    slot = [None]
    check("corp10", [0 for slot[0] in "ab"] == [0, 0] and slot == ["b"])
    # The comprehension variable stays LOCAL to the comprehension.
    shadow = 5
    check("corp11", [shadow for shadow in range(2)] == [0, 1] and shadow == 5)

    # An assignment target is a SEED plus a suffix chain: the seed may be a
    # literal or a parenthesized expression, and a CALL may sit on the way.
    box = {}
    def hold():
        return box
    hold()["k"] = 1
    check("corp12", box == {"k": 1})
    nest = {"k": [10, 20]}
    def outer():
        return nest
    outer()["k"][1] = 99
    check("corp13", nest["k"][1] == 99)
    grid = [[0]]
    [grid][0][0] = 7
    check("corp14", grid == [7])
    paren = {}
    (paren)["k"] = 2
    check("corp15", paren["k"] == 2)

    # Target lists: a RECORDED trailing comma makes a one-element tuple that
    # unpacks, and a BRACKETED target always unpacks however few elements it has.
    one, = [7]
    check("corp16", one == 7)
    (two,) = [8]
    check("corp17", two == 8)
    [three] = [9]
    check("corp18", three == 9)
    plain = [9]
    check("corp19", plain == [9])
    tup = []
    for x, in [(1,), (2,), (3,)]:
        tup.append(x)
    check("corp20", tup == [1, 2, 3])

    # `in` and the other reserved words are not target names, which is what lets
    # the trailing comma above stop where it does. A name that merely STARTS
    # with a keyword is fine.
    information = 4
    notably = 5
    check("corp21", information + notably == 9)

    # del: a trailing comma, a parenthesized target and the empty one.
    da, db, dc = "xyz"
    del da
    del db,
    del (dc)
    del ()

    # An annotation is an arbitrary EXPRESSION, not a little type language.
    def ann(p: not (int is int), q: 1 if True else 2 = 3) -> [int][0]:
        return q
    check("corp22", ann(0) == 3)
    marked: dict[str, int] = {}
    check("corp23", marked == {})

    # A triple-quoted string may open with a backslash-newline continuation.
    cont = """\
one {"two"} three
"""
    check("corp24", cont == 'one {"two"} three\n')

    # PEP 701: a replacement field may span LINES and carry `#` comments. The short
    # form must not end the field early and render the rest as literal text.
    spread = f'''
head
{ # a comment, with an apostrophe: it's fine
3 # a number
* 2}'''
    check("corp31", spread == "\nhead\n6")
    # An f-string may open with a backslash-newline line continuation.
    joined = f"""\
{1 + 1}"""
    check("corp32", joined == "2")

    # A nested field inside a format SPEC may carry its own conversion and its own
    # (non-nesting) spec. They only decide how the WIDTH is rendered, so all three
    # spellings must agree - which is what this checks, rather than the exact digits.
    fv = 12.34567
    fw = 10
    fp = 4
    check("corp33", f"{fv:{fw!r}.{fp}}" == f"{fv:{fw}.{fp}}")
    check("corp34", f"{fv:{fw:0}.{fp:1}}" == f"{fv:{fw}.{fp}}")
    # The nested spec nests AGAIN - PEP 701 puts no limit on the depth.
    check("corp35", f"{fv:{fw:{0}}.{fp:1}}" == f"{fv:{fw}.{fp}}")
    # A replacement field holds an expression LIST, so a bare tuple works in one.
    check("corp36", f"{3,}" == str((3,)))

    # PEP 758 (3.14): an except clause may list its types without parentheses.
    caught = ""
    try:
        raise TypeError("t")
    except KeyError, TypeError, ValueError:
        caught = "multi"
    check("corp25", caught == "multi")

    # Parenthesized with-items, written across lines, and an unpacking `as`.
    class CM:
        def __init__(self, v):
            self.v = v
        def __enter__(self):
            return self.v
        def __exit__(self, a, b, c):
            return False
    with (
        CM(1) as w1,
        CM((2, 3)) as (w2, w3)
    ):
        check("corp26", w1 == 1 and w2 == 2 and w3 == 3)

    # lambda takes the '/' positional-only marker and a trailing comma.
    lam = lambda a, /, b=2: a + b
    check("corp27", lam(1) == 3)
    lam2 = lambda z,: z
    check("corp28", lam2(4) == 4)

    # Unary plus, and a class header written across lines with a keyword argument.
    check("corp29", +42 == 42)
    class Base:
        pass
    class Mixin:
        pass
    class Derived(Base,
                  Mixin,
                  metaclass=type):
        pass
    check("corp30", Derived().__class__ is Derived)

    # A subscript may hold a comma-separated list that MIXES plain expressions and
    # slices - CPython's test_compile writes `d[1:2, 1:2] = 1`. The key is a TUPLE
    # whose sliced entries are slice objects; this value model has no slice type, so
    # a sliced entry is the three-element list [start, stop, step]. The checks below
    # only ask about the shape both readings share (the key arrives, and it has as
    # many entries as there were items), so they hold under CPython too.
    class Cube:
        def __init__(self):
            self.log = []
            self.key = None
        def __getitem__(self, k):
            self.key = k
            return len(k)
        def __setitem__(self, k, v):
            self.key = k
            self.log.append(v)
    cb = Cube()
    check("corp31", cb[1:2, 1:2] == 2)
    check("corp32", len(cb.key) == 2)
    cb[1:2, 1:2] = 7
    check("corp33", cb.log == [7])
    check("corp34", len(cb.key) == 2)
    cb[1:2, 1:2] += 1
    check("corp35", cb.log == [7, 3])
    check("corp36", cb[0, 1:3, ::2] == 3)
    check("corp37", cb[:, :] == 2)
    check("corp38", cb[1:2, 3] == 2)
    # The plain single slice and the plain index keep their own paths.
    row = [0, 1, 2, 3, 4]
    check("corp39", row[1:3] == [1, 2])
    check("corp40", row[::2] == [0, 2, 4])
    check("corp41", row[2] == 2)
    check("corp42", row[1:4:2] == [1, 3])

    # \N{NAME} inside an f-string: the brace after the backslash-N belongs to the
    # ESCAPE and must not open a replacement field (test_fstring line 990). Only a
    # small table of names is supported; these four are in it.
    check("corp43", f'\N{GREEK CAPITAL LETTER DELTA}' == '\u0394')
    check("corp44", f'{2}\N{GREEK CAPITAL LETTER DELTA}{3}' == '2\u03943')
    check("corp45", f'\N{LEFT CURLY BRACKET}1+1\N{RIGHT CURLY BRACKET}' == '{1+1}')
    check("corp46", f'2\N{AMPERSAND}3' == '2&3')
    # And the other side of the same rule: a backslash before a brace is a literal
    # backslash, and the brace still opens a field (test_fstring line 1016).
    check("corp47", f'\\{6*7}' == '\\42')
    check("corp48", f'\\N{6*7}' == '\\N42')

    # A non-ASCII identifier (PEP 3131). test_fstring declares one deliberately,
    # "to trigger the code in ast.c that deals with non-ascii expression values".
    tenπ = 31.4
    check("corp49", tenπ == 31.4)
    check("corp50", f"{tenπ}" == "31.4")

    # The `=` debug form inside a NESTED format spec. `{y=}` renders as the spec text
    # `y=20` - fill `y`, align `=`, width 20 - which this model does not reproduce (it
    # reads the nested field as the bare width), so only the resulting WIDTH is
    # asserted, and CPython agrees on that.
    y = 20
    check("corp51", len(f"{2:{y=}}") == 20)

# ===== SECTION 25: a bool IS an int =====
# `class bool(int)`, so True == 1 and hash(True) == hash(1), and a dict or a set
# holds ONE entry for the two of them. A section of its own next to 23 because
# the halves agree with each other here and must agree with CPython. Expected
# values are CPython 3.14.6's.
def s25():
    # Value equality, and every container equality built on it.
    check("bki1", True == 1 and False == 0 and True == 1.0)
    check("bki2", not (True == 2) and not (True != 1) and not (False == 1))
    check("bki3", [True] == [1] and (True,) == (1,) and [False, 1] == [0, True])
    # Membership in a list is ==, not identity.
    check("bki4", 1 in [True] and True in [1] and 0 in [False])
    # A dict: one key, and the FIRST key object is the one that stays - only the
    # VALUE is overwritten. str() of the key list is what makes that observable;
    # `== [True]` could not, since [True] == [1] now holds.
    d = {}
    d[True] = "t"
    d[1] = "one"
    check("bki5", len(d) == 1 and d[True] == "one" and d[1] == "one")
    check("bki6", str(list(d.keys())) == "[True]")
    e = {}
    e[1] = "one"
    e[True] = "t"
    check("bki7", len(e) == 1 and e[1] == "t" and str(list(e.keys())) == "[1]")
    # False / 0, the other half of the pair.
    g = {}
    g[False] = "F"
    g[0] = "z"
    check("bki8", len(g) == 1 and g[False] == "z")
    check("bki9", str(list(g.keys())) == "[False]")
    # The read paths that are not the subscript: `in`, .get, del, dict ==.
    check("bki10", True in {1: "x"} and 1 in {True: "x"})
    check("bki11", {1: "a"}.get(True, "miss") == "a")
    k = {True: "a"}
    del k[1]
    check("bki12", len(k) == 0)
    check("bki13", {True: 1} == {1: 1} and {False: 1} == {0: 1})
    # A set is the same rule through a different container.
    check("bki14", len({1, True}) == 1 and len({False, 0}) == 1)
    check("bki15", 1 in {True} and True in {1})
    # And the guard rails: an int that is not 0 or 1 is unaffected, and a bool
    # against a non-number stays false rather than becoming equal to 1.
    check("bki16", len({True: 1, 2: 2}) == 2 and 2 in {True: 1, 2: 2})
    check("bki17", not (True == "1") and not (True == None) and not (True == [1]))

# ===== SECTION 26: float is a TYPE, not a value =====
# Python's int and float are two different types: they print differently (1 vs
# 1.0), answer different type()s, and `/` makes a float out of two ints. One
# double cannot carry that (2.0 and 2 ARE the same double), so the float is a
# BOXED value in all three engines. Expected values are CPython 3.14.6's, and the
# renderer window (exponential iff decpt <= -4 or decpt > 16, no forced ".0" in
# that form) is Python's own - ruby, java and dart each use a different one.
def s26():
    # The type, and the literal forms that produce it. type() is compared
    # through str().
    check("flo1", str(type(1.0)) == "<class 'float'>" and str(type(1)) == "<class 'int'>")
    check("flo2", str(type(1e3)) == "<class 'float'>" and str(type(5.)) == "<class 'float'>")
    check("flo3", str(type(.5)) == "<class 'float'>" and str(type(0xff)) == "<class 'int'>")
    # repr/str, which are the same function for a float in Python 3.
    check("flo4", str(1.0) == "1.0" and repr(1.0) == "1.0" and str(1) == "1")
    check("flo5", str(2500.0) == "2500.0" and str(-0.0) == "-0.0" and str(0.5) == "0.5")
    # The exponent window: decpt <= -4 or decpt > 16, and NO forced ".0" there.
    check("flo6", str(1e15) == "1000000000000000.0" and str(1e16) == "1e+16")
    check("flo7", str(2e16) == "2e+16" and str(1.5e16) == "1.5e+16")
    check("flo8", str(1e-4) == "0.0001" and str(1e-5) == "1e-05")
    check("flo9", str(5e-324) == "5e-324" and str(1e100) == "1e+100")
    check("flo10", str(0.1 + 0.2) == "0.30000000000000004" and str(1 / 3) == "0.3333333333333333")
    # Non-finite is LOWERCASE, and -nan renders as nan.
    # The infinity by OVERFLOW, not by the 1e400 literal: parseFloat("1e400") is
    # NaN in the frozen tag engine and Infinity under goja.
    inf = 1e308 * 10
    check("flo11", str(inf) == "inf" and str(-inf) == "-inf" and str(inf - inf) == "nan")
    check("flo12", str(-(inf - inf)) == "nan")
    # `/` is TRUE division and never answers an int; // and % keep the int.
    check("flo13", str(4 / 2) == "2.0" and str(5 / 2) == "2.5" and 4 / 2 == 2)
    check("flo14", str(7 // 2) == "3" and str(7.5 // 2) == "3.0" and str(-7.5 // 2) == "-4.0")
    check("flo15", str(7 % 2) == "1" and str(7.5 % 2) == "1.5" and str(-7.5 % 2) == "0.5")
    # ** keeps the int except for a negative integer exponent.
    check("flo16", str(2 ** 3) == "8" and str(2.0 ** 3) == "8.0" and str(2 ** -1) == "0.5")
    check("flo17", str(2 ** 0.5) == "1.4142135623730951")
    # Mixed arithmetic promotes, and the int side stays an int.
    check("flo18", str(1 + 2.0) == "3.0" and str(3 - 1.0) == "2.0" and str(2 * 1.5) == "3.0")
    check("flo19", str(1 + 2) == "3" and str(abs(-2)) == "2" and str(abs(-1.5)) == "1.5")
    # == crosses the int/float line by VALUE; `is` does not.
    check("flo20", 1.0 == 1 and 1 == 1.0 and 0.0 == 0 and not (1.0 == 2))
    check("flo21", 1.0 < 2 and 2 > 1.5 and 1.5 <= 1.5 and not (1.0 != 1))
    x = 1.0
    y = x
    check("flo22", (x is y) and (1.0 is 1.0) and not (1.0 is 1))
    nan = inf - inf
    check("flo23", not (nan == nan) and (nan is nan) and not (nan < 1) and not (nan > 1))
    # Containers render their elements with repr, so a float keeps its point.
    check("flo24", str([1.0, 2]) == "[1.0, 2]" and str({"k": 1.0}) == "{'k': 1.0}")
    check("flo25", str([1.0, [2.0, 3]]) == "[1.0, [2.0, 3]]")
    # A dict and a set key a float with the int of the same value, and the FIRST
    # key object is the one that stays - the bool/int rule of section 25, now
    # three-way.
    d = {1.0: "a"}
    d[1] = "b"
    check("flo26", len(d) == 1 and d[1] == "b" and str(d) == "{1.0: 'b'}")
    e = {}
    e[2.0] = "x"
    check("flo27", e[2] == "x" and 2 in e and 2.0 in e)
    check("flo28", len({1, 1.0, 2.0, 2}) == 2 and 1.0 in {1} and 1 in {1.0})
    check("flo29", {True: "t"}[1.0] == "t" and {1.0: "f"}[True] == "f")
    check("flo30", len({True: 1, 1.0: 2}) == 1)
    # Conversions and the reducers.
    check("flo31", str(float(1)) == "1.0" and str(float("2.5")) == "2.5")
    check("flo32", str(int(2.9)) == "2" and str(int(-2.9)) == "-2")
    check("flo33", bool(1.0) and not bool(0.0) and not bool(-0.0))
    check("flo34", str(sum([1.0, 2])) == "3.0" and str(sum([1, 2])) == "3")
    check("flo35", str(max(1, 2.5)) == "2.5" and str(min(1.0, 2)) == "1.0")
    # f-strings: the default rendering is str(), !r is repr(), and the numeric
    # presentation types read the double inside the box.
    v = 2.5
    check("flo36", f"{v}" == "2.5" and f"{v!r}" == "2.5" and f"{1.0}" == "1.0")
    check("flo37", f"{v:.2f}" == "2.50" and f"{1 / 3:.4f}" == "0.3333")
    check("flo38", f"{v:>6}" == "   2.5" and f"{1.0:<5}" == "1.0  ")
    # An arbitrary precision int is still an INT, so `/` on it is a float too.
    check("flo39", str(10000000000000000000000000000 / 1) == "1e+28")
    check("flo40", str(10000000000000000000000000001 - 10000000000000000000000000000) == "1")

# ===== SECTION 27: float conversion, builtin types, rendering, big ints =====
# Expected values are CPython 3.14.6's.
def s27():
    # float(str) is CPython's float_from_string, not JavaScript's parseFloat: it
    # takes the three non-finite SPELLINGS, case-insensitively and with an
    # optional sign. Infinity is BUILT (1e308 * 10) rather than written, because
    # parseFloat("1e400") differs between goja and -frozen.
    inf = 1e308 * 10
    check("nf1", float("inf") == inf and float("-inf") == 0 - inf)
    check("nf2", float("INFINITY") == inf and float("infinity") == inf and float("-Inf") == 0 - inf)
    check("nf3", str(float("nan")) == "nan" and str(float("-nan")) == "nan" and str(float("NaN")) == "nan")
    check("nf4", float("  inf  ") == inf and str(float("1.5")) == "1.5")
    # repr and str of a float are the SHORTEST digits that read back; these
    # doubles need all seventeen.
    sd = [0.0015425378683206609, 1.7385704740834339e+92, -7.209885073274179e-254]
    check("nf5", repr(sd[0]) == "0.0015425378683206609" and str(sd[1]) == "1.7385704740834339e+92" and
                 str(sd[2]) == "-7.209885073274179e-254")
    # The builtin TYPE OBJECTS. `int` is the class, not a conversion function, so
    # type(1) IS int.
    check("ty1", isinstance("a", str) and isinstance(1, int) and isinstance(1.0, float))
    check("ty2", isinstance([1], list) and isinstance({}, dict) and isinstance(True, bool))
    check("ty3", type(1) == int and type(1) is int and type("a") is str)
    check("ty4", type(1.0) is float and type([1]) is list)
    # ...and calling the name still converts, because the conversion closure rides
    # on the class object as __conv.
    check("ty5", int("42") == 42 and str(float("1.5")) == "1.5" and list("ab") == ["a", "b"])
    check("ty6", str(int) == "<class 'int'>" and str(str) == "<class 'str'>")
    # __repr__ / __str__ of a user object. CPython prefers __str__ for
    # str()/print() and __repr__ for repr() and container elements.
    # rp2 and rp3 pin OUR default, "<__main__.Name object>": CPython writes
    # "<__main__.Name object at 0x...>" and no engine here can reproduce the
    # address, so those two right-hand sides are deliberately not CPython's.
    check("rp1", str(S27V(3)) == "V(3)" and repr(S27V(3)) == "V(3)" and str([S27V(3)]) == "[V(3)]")
    check("rp2", str(S27W(1)) == "W!1" and repr(S27W(1)) == "<__main__.S27W object>")
    check("rp3", str(S27X()) == "<__main__.S27X object>" and str([S27X()]) == "[<__main__.S27X object>]")
    # BaseException.__str__, which an exception INHERITS: str(e) is the message,
    # repr(e) is Name(args...).
    check("rp4", str(ValueError("boom")) == "boom" and repr(ValueError("boom")) == "ValueError('boom')")
    check("rp5", str(S27E("m")) == "m" and repr(S27E("m")) == "REPR")
    check("rp6", str(S27F()) == "" and str(S27F("a")) == "a" and str(S27F("a", 1)) == "('a', 1)")
    # A Python int is ARBITRARY PRECISION, so ** stays exact (10**30, not a
    # wrapped or rounded double).
    check("bi1", str(10 ** 30) == "1000000000000000000000000000000")
    check("bi2", str(2 ** 70) == "1180591620717411303424" and str(2 ** 63) == "9223372036854775808")
    check("bi3", str(3 ** 40) == "12157665459056928801" and 10 ** 18 == 1000000000000000000)
    check("bi4", str(2 ** -1) == "0.5" and 2 ** 10 == 1024 and str(type(2 ** 10)) == "<class 'int'>")
    # ...and an int operand meeting a big PROMOTES: +, -, *, //, % and every
    # ORDER comparison.
    check("bi5", str(10 ** 30 + 1) == "1000000000000000000000000000001")
    check("bi6", str(1 + 10 ** 30) == "1000000000000000000000000000001" and str(10 ** 30 * 3) == "3000000000000000000000000000000")
    check("bi7", str(10 ** 30 // 7) == "142857142857142857142857142857" and str(10 ** 30 % 7) == "1")
    check("bi8", str((0 - 10 ** 30) // 7) == "-142857142857142857142857142858" and str((0 - 10 ** 30) % 7) == "6")
    check("bi9", str(10 ** 30 % (0 - 7)) == "-6" and 10 ** 30 > 1 and 1 < 10 ** 30 and not (10 ** 30 <= 1))
    # Long division by divisors whose leading limb is tiny, huge, or neither.
    check("bi9a", str(7 ** 60 // (10 ** 14 + 1)) == "5080218607396182851035808014693654659" and
                  str(7 ** 60 % (10 ** 21 - 1)) == "956125137757221055513" and
                  str(7 ** 60 // 3 ** 50) == "707651880141060459748108418")
    # Quotient limbs of 9999 and a two-limb divisor whose second limb pulls the
    # estimated limb off by one.
    check("bi9b", str((10 ** 40 - 1) // (10 ** 20 - 1)) == "100000000000000000001" and
                  str(10 ** 40 % (10 ** 20 - 1)) == "1" and
                  str((10 ** 24 - 1) * (10 ** 12 - 1) // (10 ** 12 - 1)) == "999999999999999999999999" and
                  str(2 ** 200 // (10 ** 8 + 9999)) == "16067773825885052509019244089195147402433260636096925" and
                  str(2 ** 200 % (10 ** 8 + 9999)) == "2148301")
    # An unreached annotation followed by a reached one: the class must still
    # get its __annotations__ dict.
    check("an1", len(S27Ann.__annotations__) == 1 and "am_yes" in S27Ann.__annotations__)

class S27V:
    def __init__(self, n):
        self.n = n
    def __repr__(self):
        return "V(" + str(self.n) + ")"

class S27W:
    def __init__(self, n):
        self.n = n
    def __str__(self):
        return "W!" + str(self.n)

class S27X:
    pass

class S27E(Exception):
    def __repr__(self):
        return "REPR"

class S27F(Exception):
    pass

class S27Ann:
    if False:
        am_no: int = 1
    am_yes: int = 2

# ===== SECTION 28: printf-style % formatting =====
# Every right-hand side is CPython 3.14.6's own answer.
#
# The arguments come from a TUPLE; any other value, a list included, is a single
# value. pct20 is that rule from the "single value" side.
def s28():
    check("pct01", ("%s" % "hi") == "hi" and ("%s and %s" % ("a", "b")) == "a and b")
    check("pct02", ("%d" % 42) == "42" and ("%5d|" % 42) == "   42|" and ("%-5d|" % 42) == "42   |")
    check("pct03", ("%05d|" % 42) == "00042|" and ("%05d|" % -42) == "-0042|")
    check("pct04", ("%+d" % 42) == "+42" and ("% d" % 42) == " 42")
    check("pct05", ("%x %X %o" % (255, 255, 8)) == "ff FF 10")
    check("pct06", ("%.2f" % 3.14159) == "3.14" and ("%f" % 1.5) == "1.500000")
    check("pct07", ("%8.2f|" % 3.14159) == "    3.14|" and ("%-8.2f|" % 3.14159) == "3.14    |")
    check("pct08", ("%.3s|" % "hello") == "hel|" and ("%r" % "hi") == "'hi'")
    check("pct09", ("%c%c" % (65, "b")) == "Ab" and ("%d%% done" % 50) == "50% done")
    check("pct10", ("%(a)s-%(b)d" % {"a": "x", "b": 7}) == "x-7")
    check("pct11", ("%10s|" % "hi") == "        hi|" and ("%-10s|" % "hi") == "hi        |")
    check("pct12", ("%s" % 1.0) == "1.0" and ("%s" % True) == "True" and ("%s" % None) == "None")
    check("pct13", ("%e" % 1234.5678) == "1.234568e+03" and ("%E" % 1234.5678) == "1.234568E+03")
    check("pct14", ("%.2e" % 1234.5678) == "1.23e+03" and ("%e" % 0.0) == "0.000000e+00")
    check("pct15", ("%g" % 1234.5678) == "1234.57" and ("%g" % 0.00001234) == "1.234e-05")
    check("pct16", ("%G" % 1234567890.0) == "1.23457E+09" and ("%.3g" % 1234.5678) == "1.23e+03")
    check("pct17", ("%g %g" % (100000.0, 1000000.0)) == "100000 1e+06")
    # An arbitrary precision int is exact through %s and %d.
    check("pct18", ("%s" % (10 ** 30)) == "1000000000000000000000000000000")
    check("pct19", ("%d" % (10 ** 30)) == "1000000000000000000000000000000")
    check("pct20", ("%s" % [1, 2]) == "[1, 2]" and ("%s %d" % ("n", 3.9)) == "n 3")
    # The .Nf verb rounds HALF TO EVEN, which is strconv.FormatFloat's rule and
    # CPython's (f"{2.5:.0f}" is "2").
    check("pct21", f"{2.5:.0f}" == "2" and f"{3.5:.0f}" == "4" and f"{0.125:.2f}" == "0.12")
    check("pct22", ("%.0f %.1f" % (2.5, 0.05)) == "2 0.1")
    # ...and the {:x} spec: (255).toString(16) works under goja and ABORTS under
    # -frozen. The matrix's goja/-frozen byte-identity check is the gate.
    check("pct23", f"{255:x} {255:X} {8:o} {5:b}" == "ff FF 10 101")

# ===== SECTION 29: the bitwise operators are ARBITRARY PRECISION =====
# Python's rule is INFINITE two's complement over arbitrary precision ints, not
# ECMAScript's ToInt32 with the shift count masked to five bits (-1 & 0xffffffff
# is 4294967295, 1 << 40 is 1099511627776). Every value below is CPython
# 3.14.6's own answer.
#
# The operands come out of a list on purpose, so a constant folder cannot
# answer the assertion instead of the runtime; the paired literal forms in
# bit01/bit03/bit05 are there to hold the folding path as well.
def s29():
    v = [-1, 0xffffffff, 1, 40, -5, 2, 0, 12, 10 ** 30, -(10 ** 30), 255, -256]
    check("bit01", (v[0] & v[1]) == 4294967295 and (-1 & 0xffffffff) == 4294967295)
    check("bit02", (v[0] | v[1]) == -1 and (v[0] ^ v[1]) == -4294967296)
    check("bit03", (v[2] << v[3]) == 1099511627776 and (1 << 40) == 1099511627776)
    check("bit04", (v[2] << 62) == 4611686018427387904 and (1 << 100) == 1267650600228229401496703205376)
    check("bit05", (v[4] >> v[5]) == -2 and (-5 >> 1) == -3 and (v[1] >> 4) == 268435455)
    check("bit06", ~v[0] == 0 and ~v[6] == -1 and ~v[10] == -256 and ~(1 << 40) == -1099511627777)
    check("bit07", (v[4] & v[1]) == 4294967291 and (v[4] | v[1]) == -1 and (v[4] ^ v[1]) == -4294967292)
    check("bit08", (v[8] & 255) == 0 and (v[8] | 1) == 1000000000000000000000000000001)
    check("bit09", (v[9] & 255) == 0 and (v[9] >> 40) == -909494701772928238 and (v[8] >> 60) == 867361737988)
    check("bit10", (v[0] & v[8]) == v[8] and (v[0] ^ v[8]) == -1000000000000000000000000000001)
    # 2**53 | 1 is 2**53+1, which no double holds: the narrowers have to REBUILD
    # and compare before they hand a box back as a plain number. Asserted through
    # str(), the sharper assertion.
    check("bit11", str((2 ** 53) | 1) == "9007199254740993" and str((2 ** 53) ^ 1) == "9007199254740993")
    # bool & bool is a BOOL; a bool meeting an int is an int, and True << 1 is 2.
    check("bit12", (True & True) is True and (True | False) is True and (True ^ True) is False)
    check("bit13", (True & 1) == 1 and not ((True & 1) is True) and (True << 1) == 2)
    check("bit14", (v[10] << 45) == 8972014882652160 and (v[10] << 46) == 17944029765304320)
    check("bit15", (v[7] >> 200) == 0 and (v[4] >> 200) == -1 and (v[2] << 99) == 633825300114114700748351602688)

# ===== SECTION 30: integer literals past 2**53 =====
# Whether an integer literal needs the arbitrary precision box is decided on the
# literal TEXT, in the lexer of python-interpreter.abnf and the emitter of
# python-to-llvm-ir.abnf: parseFloat would round "9007199254740993" down to
# exactly 2**53, and a bare parseInt of a 0x / 0o / 0b form would round
# 0x7fffffffffffffff to 9223372036854776000. Every value below is representable
# only as an arbitrary precision int, so a rounded literal shows up directly in
# str().
def s30():
    lit = [9007199254740993, 0x20000000000001, 0o400000000000000001,
           0b100000000000000000000000000000000000000000000000000001,
           0x7fffffffffffffff, 0xffffffffffffffff, -9007199254740993,
           0x10000000000000000000]
    check("lit01", str(9007199254740993) == "9007199254740993" and str(lit[0]) == "9007199254740993")
    check("lit02", lit[0] - 1 == 9007199254740992 and lit[0] != 9007199254740992)
    check("lit03", str(0x20000000000001) == "9007199254740993" and lit[1] == lit[0])
    check("lit04", str(0o400000000000000001) == "9007199254740993" and lit[2] == lit[0])
    check("lit05", str(0b100000000000000000000000000000000000000000000000000001) == "9007199254740993" and lit[3] == lit[0])
    check("lit06", str(0x7fffffffffffffff) == "9223372036854775807")
    check("lit07", str(0xffffffffffffffff) == "18446744073709551615" and lit[5] == lit[4] * 2 + 1)
    check("lit08", str(-9007199254740993) == "-9007199254740993" and lit[6] < -9007199254740992)
    check("lit09", str(0x10000000000000000000) == "75557863725914323419136")
    check("lit10", str(lit[7] & 0xff) == "0" and str(lit[7] >> 60) == "65536")
    # 2**53 itself IS exact and must NOT be boxed - the boundary is strictly `>`.
    check("lit11", str(9007199254740992) == "9007199254740992" and str(0x20000000000000) == "9007199254740992")
    check("lit12", str(0xff) == "255" and str(0o17) == "15" and str(0b1011) == "11" and str(0x0) == "0")
    # '_' separators are stripped before the text is measured.
    check("lit13", str(0x1F_FF_FF_FF_FF_FF_FF) == "9007199254740991" and str(0x20_0000_0000_0001) == "9007199254740993")
    check("lit14", (9007199254740993 & 0xff) == 1 and (9007199254740993 >> 1) == 4503599627370496)
    check("lit15", str(0xFFFFFFFFFFFFFFFFFFFF) == "1208925819614629174706175" and str(type(0x20000000000001).__name__) == "int")

# ===== END SECTIONS =====



# ===== SECTION 37: a subscript MISS is catchable, and a generator's finally =====
#
#  * A subscript miss (`try: d[3] / except KeyError`, list and string indexing
#    too) is a catchable exception in all three engines.
#  * The KeyError's argument is the KEY, not its repr: CPython's KeyError.__str__
#    (repr of args[0], and only when there is exactly one) is where the repr
#    happens, so repr(e) is KeyError(3) and e.args[0] is the key.
#  * The builtin exception hierarchy is CPython's: `except LookupError` catches a
#    KeyError, `except ArithmeticError` a ZeroDivisionError, and BaseException is
#    bound.
#  * A generator's `finally` runs when CPython runs it, not at the first next().
#    The interpreter's replay repeats side effects, so the order assertions below
#    are written on FIRST OCCURRENCE, which is the property that survives replay:
#    repetition inserts duplicates, and only a reordering can move a first.
#
# Every operand is read out of a list so the constant folder cannot fold a row.
def firsts(xs):
    out = []
    for x in xs:
        if x not in out:
            out.append(x)
    return out


def s37():
    K = [3, "b", 7, "zz", 0]
    D = {1: "one", "a": "A"}
    L = [10, 20]

    def caught(f):
        try:
            f()
            return "no-raise"
        except BaseException as e:
            return type(e).__name__ + "|" + str(e) + "|" + repr(e)

    # --- the subscript miss is an exception, in all three engines ---------
    check("ky01", caught(lambda: D[K[0]]) == "KeyError|3|KeyError(3)")
    check("ky02", caught(lambda: D[K[1]]) == "KeyError|'b'|KeyError('b')")
    check("ky03", caught(lambda: L[K[2]]) ==
          "IndexError|list index out of range|IndexError('list index out of range')")
    check("ky04", caught(lambda: "ab"[K[2]]) ==
          "IndexError|string index out of range|IndexError('string index out of range')")
    check("ky05", caught(lambda: D.pop(K[3])) == "KeyError|'zz'|KeyError('zz')")
    check("ky06", caught(lambda: "%(q)s" % D) == "KeyError|'q'|KeyError('q')")
    check("ky07", caught(lambda: "{q}".format(a=K[0])) == "KeyError|'q'|KeyError('q')")

    # --- the argument is the KEY, not its text ---------------------------
    e1 = None
    try:
        D[K[0]]
    except KeyError as ex:
        e1 = ex
    check("ky08", len(e1.args) == 1 and e1.args[0] == 3 and type(e1.args[0]).__name__ == "int")
    e2 = KeyError(K[1])
    check("ky09", str(e2) == "'b'" and repr(e2) == "KeyError('b')" and e2.args[0] == "b")
    # Zero arguments falls back to BaseException.__str__, which is the empty text.
    check("ky10", str(KeyError()) == "" and repr(KeyError()) == "KeyError()")
    # Only KeyError repr's its argument; its siblings do not.
    check("ky11", str(ValueError(K[1])) == "b" and repr(ValueError(K[1])) == "ValueError('b')")
    check("ky12", str(IndexError(K[0])) == "3" and repr(IndexError(K[0])) == "IndexError(3)")

    # --- the hierarchy above the raise -----------------------------------
    def by(t):
        try:
            D[K[0]]
        except t as ex:
            return "caught-" + type(ex).__name__
        except BaseException:
            return "missed"
    check("ky13", by(KeyError) == "caught-KeyError")
    check("ky14", by(LookupError) == "caught-KeyError")
    check("ky15", by(Exception) == "caught-KeyError")
    check("ky16", by(BaseException) == "caught-KeyError")
    check("ky17", by(IndexError) == "missed")
    check("ky18", issubclass(KeyError, LookupError) and issubclass(IndexError, LookupError))
    check("ky19", issubclass(LookupError, Exception) and issubclass(Exception, BaseException))
    check("ky20", issubclass(ZeroDivisionError, ArithmeticError) and
                  issubclass(UnboundLocalError, NameError))
    check("ky21", not issubclass(LookupError, KeyError))
    # A user subclass INHERITS KeyError.__str__, which is why the test is the
    # class chain rather than the name.
    check("ky22", str(MyKeyErr(K[0])) == "3" and repr(MyKeyErr(K[0])) == "MyKeyErr(3)")
    check("ky23", isinstance(MyKeyErr(K[0]), LookupError))

    # --- the generator's finally runs when CPython runs it ----------------
    lg = []

    def g37():
        try:
            with Mark(lg):
                lg.append("t1")
                yield K[0]
                lg.append("t2")
                yield K[1]
        finally:
            lg.append("fin")

    it = g37()
    lg.append("d-a")
    next(it)
    lg.append("d-b")
    next(it)
    lg.append("d-c")
    try:
        next(it)
    except StopIteration:
        lg.append("d-stop")
    check("ky24", firsts(lg) ==
          ["d-a", "enter", "t1", "d-b", "t2", "d-c", "exit", "fin", "d-stop"])

    # close() mid-run: the finally runs THERE, not at the first next().
    lc = []

    def g37b():
        try:
            lc.append("c1")
            yield K[0]
            lc.append("c2")
            yield K[1]
        finally:
            lc.append("c-fin")

    ck = g37b()
    lc.append("d0")
    next(ck)
    lc.append("d1")
    check("ky25", ck.close() is None)
    lc.append("d2")
    check("ky26", firsts(lc) == ["d0", "c1", "d1", "c-fin", "d2"])
    # A second close is a no-op, and the body does not run again.
    check("ky27", ck.close() is None and firsts(lc) == ["d0", "c1", "d1", "c-fin", "d2"])

    # Two generators driven alternately keep their events in CPython's order.
    li = []

    def ga():
        li.append("a1")
        yield K[0]
        li.append("a2")
        yield K[1]

    def gb():
        li.append("b1")
        yield K[0]
        li.append("b2")
        yield K[1]

    A = ga()
    B = gb()
    li.append("d0")
    next(A)
    li.append("d1")
    next(B)
    li.append("d2")
    next(A)
    li.append("d3")
    next(B)
    li.append("d4")
    check("ky28", firsts(li) == ["d0", "a1", "d1", "b1", "d2", "a2", "d3", "b2", "d4"])

    # --- g.throw(exc) raises AT the parked yield ---------------------------
    # Here and not only in the features file because the features file is NEVER
    # BUILT NATIVELY: layer 2's pyGenValue is only reachable through
    # tests/native-full.sh and tests/clang-check.sh.
    lt = []

    def g37c():
        try:
            lt.append("t1")
            yield K[0]
            lt.append("t2")
            yield K[1]
        except ValueError as e:
            lt.append("caught:" + str(e))
            yield K[2]
        finally:
            lt.append("t-fin")

    tk = g37c()
    lt.append("d0")
    next(tk)
    lt.append("d1")
    check("ky29", tk.throw(ValueError("V")) == K[2])
    lt.append("d2")
    try:
        next(tk)
    except StopIteration:
        lt.append("d-stop")
    check("ky30", firsts(lt) ==
          ["d0", "t1", "d1", "caught:V", "d2", "t-fin", "d-stop"])
    # Uncaught: the value propagates out of throw() and the generator is done.
    lu = []

    def g37d():
        try:
            yield K[0]
            yield K[1]
        finally:
            lu.append("u-fin")

    uk = g37d()
    next(uk)
    up = ""
    try:
        uk.throw(KeyError(K[0]))
    except KeyError as e:
        up = repr(e)
    check("ky31", up == "KeyError(3)" and lu == ["u-fin"])
    ud = ""
    try:
        next(uk)
    except StopIteration:
        ud = "StopIteration"
    check("ky32", ud == "StopIteration")
    # A generator that never ran has no suspension point and so no `finally`.
    lv = []

    def g37e():
        try:
            yield K[0]
        finally:
            lv.append("v-fin")

    vk = g37e()
    vp = ""
    try:
        vk.throw(ValueError("W"))
    except ValueError as e:
        vp = str(e)
    check("ky33", vp == "W" and lv == [])
    # StopIteration's ARGS: empty for a body that returned None, (v,) otherwise.

    def g37f():
        yield K[0]
        return K[2]

    fk = g37f()
    next(fk)
    fr = ""
    try:
        next(fk)
    except StopIteration as e:
        fr = repr(e) + "|" + str(e.value)
    check("ky34", fr == "StopIteration(7)|7")
    nk = g37d()
    next(nk)
    next(nk)
    nr = ""
    try:
        next(nk)
    except StopIteration as e:
        nr = repr(e) + "|" + str(e.value)
    check("ky35", nr == "StopIteration()|None")


class MyKeyErr(KeyError):
    pass


class Mark:
    def __init__(self, log):
        self.log = log

    def __enter__(self):
        self.log.append("enter")
        return 1

    def __exit__(self, a, b, c):
        self.log.append("exit")
        return False


# ===== SECTION 31: str's method library =====
# Python's str method library is carried three times
# (languages/python-interpreter.abnf, the str-methods chapter of
# languages/lib/python-rt.metajs, and abnf/pystrmethod.go), and what these
# assertions pin is the SEMANTICS rather than the names. Every value below is
# CPython 3.14.6's own answer.
#
# The operands are read out of a list so a constant folder cannot answer them.
def s31():
    v = ["  Hello World  ", "a,b,,c", "abc", " a  b ", "they're bill's 12ab",
         "xyxhixy", "-42", "a\nb\r\nc", "Ab Cd", "aaa", "ABC", "MiXeD", "abcabc"]
    check("str01", v[2].upper() == "ABC" and v[10].lower() == "abc" and v[10].casefold() == "abc")
    # The two split ALGORITHMS: no argument splits on runs of whitespace and drops
    # the empties, split(" ") does neither.
    check("str02", v[3].split() == ["a", "b"] and v[3].split(" ") == ["", "a", "", "b", ""])
    check("str03", v[1].split(",") == ["a", "b", "", "c"] and v[1].split(",", 1) == ["a", "b,,c"])
    check("str04", v[1].rsplit(",", 1) == ["a,b,", "c"] and v[3].rsplit(None, 1) == [" a", "b"])
    check("str05", v[3].split(None, 1) == ["a", "b "] and "".split(",") == [""] and "".split() == [])
    # strip(chars) is a SET of characters, not a prefix.
    check("str06", v[0].strip() == "Hello World" and v[5].lstrip("xy") == "hixy" and v[5].rstrip("xy") == "xyxhi")
    check("str07", "abcba".strip("ab") == "c" and "abab".strip("ab") == "")
    check("str08", "|".join(["a", "b", "c"]) == "a|b|c" and "|".join([]) == "" and "-".join(v[2]) == "a-b-c")
    check("str09", v[9].replace("a", "b", 2) == "bba" and v[9].replace("a", "") == "")
    check("str10", v[2].replace("", "-") == "-a-b-c-" and v[2].replace("", "-", 2) == "-a-bc")
    # find answers -1 where index raises, and the bounds follow ADJUST_INDICES:
    # a start past the end is -1 even for the empty needle.
    check("str11", v[2].find("c") == 2 and v[2].find("z") == -1 and v[2].find("c", 0, 2) == -1)
    check("str12", v[2].find("", 3) == 3 and v[2].find("", 4) == -1 and v[2].find("c", -1) == 2)
    check("str13", v[12].rfind("b") == 4 and v[2].rfind("") == 3 and v[12].rindex("a") == 3)
    check("str14", v[2].index("b") == 1 and "aaa".count("aa") == 1 and v[2].count("") == 4)
    check("str15", "aaa".count("a", 1) == 2 and v[2].startswith(("x", "a")) and v[2].endswith("bc", 0, 3))
    check("str16", not v[2].endswith("bc", 0, 2) and v[2].startswith("ab"))
    # title's word boundary is "the previous character was not cased".
    check("str17", v[4].title() == "They'Re Bill'S 12Ab" and v[0].capitalize() == "  hello world  ")
    check("str18", v[11].swapcase() == "mIxEd" and v[8].istitle() and not v[11].istitle())
    check("str19", "123".isdigit() and not "".isdigit() and v[2].isalpha() and "ab1".isalnum())
    check("str20", "  ".isspace() and "ab1".islower() and "AB1".isupper() and not "123".islower())
    # center puts the odd extra pad on the LEFT only when the width is odd too.
    check("str21", v[2].center(7, "*") == "**abc**" and "ab".center(7, "*") == "***ab**")
    check("str22", v[2].center(2) == "abc" and v[2].ljust(5, ".") == "abc.." and v[2].rjust(5, ".") == "..abc")
    check("str23", v[6].zfill(5) == "-0042" and v[6].zfill(1) == "-42" and "".zfill(3) == "000")
    # splitlines: "\r\n" is ONE break and the boundary set is wider than "\n".
    check("str24", v[7].splitlines() == ["a", "b", "c"] and "".splitlines() == [])
    check("str25", "a\nb\n".splitlines(True) == ["a\n", "b\n"])
    check("str26", list(v[2].partition("b")) == ["a", "b", "c"] and list(v[2].partition("z")) == ["abc", "", ""])
    check("str27", list(v[2].rpartition("z")) == ["", "", "abc"] and list("abcb".rpartition("b")) == ["abc", "b", ""])
    check("str28", v[2].removeprefix("ab") == "c" and v[2].removeprefix("z") == "abc" and v[2].removesuffix("bc") == "a")
    check("str29", "a\tb".expandtabs() == "a       b" and "ab\tc".expandtabs(4) == "ab  c")
    # format shares the mini-language with the f-strings.
    check("str30", "{}-{}".format(v[2], 3) == "abc-3" and "{1}{0}{1}".format("a", "b") == "bab")
    check("str31", "{x}/{y}".format(x=1, y="q") == "1/q" and "{{{0}}}".format(1) == "{1}")
    check("str32", "{0:>6}".format(v[2]) == "   abc" and "{0:*^7}".format(v[2]) == "**abc**")
    check("str33", "{0[1]}".format([7, 8]) == "8" and "{0:{1}}".format(v[2], ">5") == "  abc")
    check("str34", "{0!r}".format(v[2]) == "'abc'")
    # A malformed field is a catchable ValueError in CPython's parse_field wording;
    # the character after '!' is the conversion whatever it is, and it is checked
    # only after the field has been looked up.
    def fmterr(f, *a):
        try:
            return f.format(*a)
        except Exception as e:
            return type(e).__name__ + ":" + str(e)
    check("str35", fmterr("{0!z}", 1) == "ValueError:Unknown conversion specifier z" and
                   fmterr("{0!}}", 1) == "ValueError:Unknown conversion specifier }" and
                   fmterr("{0!\x01}", 1) == "ValueError:Unknown conversion specifier \\x1")
    check("str36", fmterr("{0:", 1) == "ValueError:unmatched '{' in format spec" and
                   fmterr("{0!", 1) == "ValueError:end of string while looking for conversion specifier" and
                   fmterr("{0!rr}", 1) == "ValueError:expected ':' after conversion specifier")
    check("str37", fmterr("{0[}", v[2]) == "ValueError:expected '}' before end of string" and
                   fmterr("{a{}", 1) == "ValueError:unexpected '{' in field name" and
                   fmterr("{:{:{}}}", 1, 2, 3) == "ValueError:Max string recursion exceeded" and
                   fmterr("{5!z}", 1) == "IndexError:Replacement index 5 out of range for positional args tuple")
    # Automatic and manual field numbering may not be mixed, and a field name's
    # accessors are checked as they are read.
    check("str40", fmterr("{0}{}", 1) == "ValueError:cannot switch from manual field specification to automatic field numbering" and
                   fmterr("{}{0}", 1) == "ValueError:cannot switch from automatic field numbering to manual field specification" and
                   fmterr("{0.}", 1) == "ValueError:Empty attribute in format string" and
                   fmterr("{0[0]x}", [1]) == "ValueError:Only '.' or '[' may follow ']' in format field specifier" and
                   fmterr("{:{}}", v[2], 5) == "abc  ")
    # ord() and chr() refuse their domain with catchable exceptions.
    def bierr(f, x):
        try:
            return f(x)
        except Exception as e:
            return type(e).__name__ + ":" + str(e)
    check("str38", bierr(ord, 1) == "TypeError:ord() expected string of length 1, but int found" and
                   bierr(ord, v[2]) == "TypeError:ord() expected a character, but string of length 3 found")
    check("str39", bierr(chr, -1) == "ValueError:chr() arg not in range(0x110000)" and
                   bierr(chr, v[2]) == "TypeError:'str' object cannot be interpreted as an integer" and
                   bierr(chr, 65) == "A")


# ===== SECTION 32: integer arithmetic PROMOTES past 2**53 =====
# SECTION 30 pinned the LITERALS; this is the arithmetic. A plain Python int is a
# double, so it is exact to 2**53 and must promote to the box above it. The guard
# is two comparisons on the fast path (see pyOver53 / pyAOver53 /
# pyPromoteArith); these assertions say it fires when it must and stays out of
# the way when it must not.
#
# The operands come out of a list so a constant folder cannot answer them; the
# paired literal form in ar01 holds the folding path as well.
def s32():
    v = [9007199254740992, 1, 2, 3037000500, 94906266, 123456789, -9007199254740992,
         4503599627370496, 987654321, 65536, 1000000007]
    check("ar01", str(v[0] + v[1]) == "9007199254740993" and str(9007199254740992 + 1) == "9007199254740993")
    check("ar02", str(v[6] - v[1]) == "-9007199254740993" and v[0] + v[1] != v[0])
    check("ar03", str(v[0] * v[2]) == "18014398509481984" and str(v[3] * v[3]) == "9223372037000250000")
    check("ar04", str(v[4] * v[4]) == "9007199326062756" and str(v[5] * v[8]) == "121932631112635269")
    check("ar05", str(v[7] + v[7] + v[1]) == "9007199254740993")
    check("ar06", str(v[6] * v[3]) == "-27354868640248020074496000")
    check("ar07", str(v[9] * v[9] * v[9]) == "281474976710656" and str(v[10] * v[10]) == "1000000014000000049")
    # Below the boundary NOTHING changes shape: the answers stay plain ints.
    check("ar08", v[5] + v[8] == 1111111110 and v[5] * v[2] == 246913578 and v[5] - v[8] == -864197532)
    check("ar09", (v[0] + v[1]) - v[1] == v[0] and (v[0] + v[1]) % 2 == 1)
    # A string + and the two sequence repetitions must NOT reach the box.
    check("ar10", "99999999999999999999" + "9" == "999999999999999999999" and [0] * 3 == [0, 0, 0] and "ab" * 3 == "ababab")
    # A bool IS an int, and the promotion has to accept it as an operand.
    check("ar11", str(v[0] + True) == "9007199254740993" and str(v[0] * True) == "9007199254740992")

# ===== SECTION 33: for-of drives a generator LAZILY =====
# A generator is STEPPED through the js_get(g, "next") / {value, done} protocol,
# so `for x in endless(): break` works - the shape csharp-to-llvm-ir.abnf's
# foreach has. Every other source still goes through js_pyiter, which normalizes
# a dict to its keys and a string to its characters.
def s33():
    def nat():
        i = 0
        while True:
            yield i
            i = i + 1
    def upto(n):
        i = 0
        while i < n:
            yield i
            i = i + 1
    out = []
    for x in nat():
        out.append(x)
        if x >= 3:
            break
    check("lazy01", out == [0, 1, 2, 3])
    first = -1
    for x in nat():
        first = x
        break
    check("lazy02", first == 0)
    check("lazy03", [x for x in upto(3)] == [0, 1, 2] and list(upto(2)) == [0, 1])
    tot = 0
    for x in upto(4):
        tot = tot + x
    else:
        tot = tot + 100
    check("lazy04", tot == 106)
    kept = []
    for x in upto(5):
        if x % 2 == 0:
            continue
        kept.append(x)
    check("lazy05", kept == [1, 3])
    nested = []
    for a in upto(2):
        for b in upto(2):
            nested.append(a * 10 + b)
    check("lazy06", nested == [0, 1, 10, 11])
    # The array path: a list, a string, a dict and a range iterate as before.
    check("lazy07", [c for c in "hi"] == ["h", "i"] and [k for k in {"a": 1, "b": 2}] == ["a", "b"])
    check("lazy08", [e for e in [1, 2, 3]] == [1, 2, 3] and [i for i in range(3)] == [0, 1, 2])
    # A generator that is never exhausted must not leave the loop broken for the
    # next one: two independent endless generators in a row.
    seen = 0
    for x in nat():
        seen = seen + 1
        break
    for x in nat():
        seen = seen + 1
        break
    check("lazy09", seen == 2)

# ===== SECTION 34: generator.close(), and a `finally` around a yield =====
#
#   * g.close() answers None and closes the cell itself (runtime.c's gen_close,
#     reached through the tag-15 `return` member), so `for v in g` after
#     g.close() yields nothing.
#   * a `yield` inside a `try` still makes the def around it a generator
#     function.
#
# CPython does NOT close a generator when a `for` over it breaks - that is
# JavaScript's rule, not Python's - so nothing here asserts that it does.
# The finally COUNT is taken after exactly one step, where the interpreter's
# replay (one finally per next()) and the compiler's one close-time finally agree.
def s34():
    state = [0]
    def g34():
        try:
            yield 1
            yield 2
            yield 3
        finally:
            state[0] = state[0] + 1
    # A `finally` around a yield: the values are unaffected and the clause runs.
    got = []
    for v in g34():
        got.append(v)
    check("gcl1", got == [1, 2, 3])
    # No finally COUNT here: replay runs one per next(), so exhausting three
    # yields reaches four in the interpreter half and one in the compiler's.
    # close() answers None, and closes: the loop that follows it yields nothing.
    state[0] = 0
    h = g34()
    first = []
    for v in h:
        first.append(v)
        break
    check("gcl3", first == [1])
    check("gcl4", h.close() is None)
    check("gcl5", state[0] == 1)
    rest = []
    for v in h:
        rest.append(v)
    check("gcl6", rest == [])
    # close() is idempotent, and answers None on a generator that never started
    # and on one that ran to the end.
    check("gcl7", h.close() is None)
    k = g34()
    check("gcl8", k.close() is None)
    empty = []
    for v in k:
        empty.append(v)
    check("gcl9", empty == [])
    m = g34()
    for v in m:
        pass
    check("gcl10", m.close() is None)
    # CPython's own rule: a `for` that BREAKS leaves the generator open, so the
    # next loop over it RESUMES. (node closes it; python does not.)
    n = g34()
    a = []
    for v in n:
        a.append(v)
        break
    b = []
    for v in n:
        b.append(v)
    check("gcl11", a == [1] and b == [2, 3])
    # Nested finally clauses unwind outward, and send() still works through a try.
    def g34b():
        try:
            try:
                yield 1
                yield 2
            finally:
                state[0] = state[0] + 10
        finally:
            state[0] = state[0] + 100
    state[0] = 0
    p = g34b()
    one = []
    for v in p:
        one.append(v)
        break
    p.close()
    check("gcl12", one == [1] and state[0] == 110)
    # A generator whose try has an EXCEPT arm still yields through it.
    def g34c():
        try:
            yield 1
            raise ValueError("boom")
        except ValueError:
            yield 2
    q = []
    for v in g34c():
        q.append(v)
    check("gcl13", q == [1, 2])


# ===== SECTION 35: list methods, hasattr and getattr =====
# Every value is read out of a list so the grammar's
# constant folder cannot pre-compute a row, and every error is CAUGHT - the
# point of raising a real Python exception rather than aborting is that the
# behaviour becomes assertable in one file, in all three engines.
def s35():
    # --- the mutators ---------------------------------------------------
    xs = [3, 1, 2]
    check("lm01", xs.append(4) is None and xs == [3, 1, 2, 4])
    check("lm02", xs.pop() == 4 and xs == [3, 1, 2])
    # pop(i) removes index i, not the last element.
    ys = [3, 1, 2]
    check("lm03", ys.pop(0) == 3 and ys == [1, 2])
    zs = [3, 1, 2]
    check("lm04", zs.pop(-2) == 1 and zs == [3, 2])
    cs = [3, 1, 2]
    check("lm05", cs.clear() is None and cs == [] and len(cs) == 0)
    ds = [3, 1, 2]
    es = ds.copy()
    es.append(9)
    check("lm06", ds == [3, 1, 2] and es == [3, 1, 2, 9])
    rs = [3, 1, 2]
    check("lm07", rs.reverse() is None and rs == [2, 1, 3])
    ins = [3, 1, 2]
    ins.insert(1, 9)
    check("lm08", ins == [3, 9, 1, 2])
    # insert CLAMPS out of range at both ends rather than raising.
    lo = [3, 1, 2]
    lo.insert(-99, 9)
    hi = [3, 1, 2]
    hi.insert(99, 9)
    check("lm09", lo == [9, 3, 1, 2] and hi == [3, 1, 2, 9])
    rm = [3, 1, 2, 1]
    check("lm10", rm.remove(1) is None and rm == [3, 2, 1])
    ex = [3, 1]
    check("lm11", ex.extend([7, 8]) is None and ex == [3, 1, 7, 8])
    exs = [3]
    exs.extend("ab")
    check("lm12", exs == [3, "a", "b"])
    # extend(self) DOUBLES the list; it must not loop forever.
    dbl = [1, 2]
    dbl.extend(dbl)
    check("lm13", dbl == [1, 2, 1, 2])
    # --- index/count ----------------------------------------------------
    ix = [3, 1, 2, 1]
    check("lm14", ix.index(1) == 1 and ix.index(1, 2) == 3)
    check("lm15", ix.index(2, -2) == 2 and ix.index(1, 0, 2) == 1)
    check("lm16", ix.count(1) == 2 and ix.count(9) == 0)
    # Equality, not identity: a bool IS an int and 1.0 == 1.
    mixed = [1.0, True, 2]
    check("lm17", mixed.count(1) == 2 and mixed.index(1) == 0)
    # --- sort: stable, key=, reverse= -----------------------------------
    st = [5, 3, 1, 4]
    check("lm18", st.sort() is None and st == [1, 3, 4, 5])
    sr = [5, 3, 1, 4]
    sr.sort(reverse=True)
    check("lm19", sr == [5, 4, 3, 1])
    sk = ["ccc", "a", "bb"]
    sk.sort(key=len)
    check("lm20", sk == ["a", "bb", "ccc"])
    # STABILITY, and that reverse= keeps equal elements in their original order
    # (CPython reverses, sorts ascending and reverses again - an inverted
    # comparison would swap the pairs below).
    pairs = [[1, "b"], [1, "a"], [0, "c"], [1, "d"]]
    fwd = pairs.copy()
    fwd.sort(key=lambda p: p[0])
    rev = pairs.copy()
    rev.sort(key=lambda p: p[0], reverse=True)
    check("lm21", fwd == [[0, "c"], [1, "b"], [1, "a"], [1, "d"]])
    check("lm22", rev == [[1, "b"], [1, "a"], [1, "d"], [0, "c"]])
    strs = ["b", "a", "c"]
    strs.sort()
    check("lm23", strs == ["a", "b", "c"])
    # sort() agrees with sorted() on the same input.
    src = [4, 1, 3]
    cp = src.copy()
    cp.sort()
    check("lm24", cp == sorted(src))
    # --- the errors, all CATCHABLE --------------------------------------
    log = []
    try:
        [].pop()
    except IndexError as e:
        log.append("A" + str(e))
    try:
        [1].pop(5)
    except IndexError as e:
        log.append("B" + str(e))
    try:
        [1, 2].remove(9)
    except ValueError as e:
        log.append("C" + str(e))
    try:
        [1, 2].index(9)
    except ValueError as e:
        log.append("D" + str(e))
    try:
        [1, 2].extend(5)
    except TypeError as e:
        log.append("E" + str(e))
    try:
        [1, 2].sort(len)
    except TypeError as e:
        log.append("F" + str(e))
    check("lm25", log[0] == "Apop from empty list")
    check("lm26", log[1] == "Bpop index out of range")
    check("lm27", log[2] == "Clist.remove(x): x not in list")
    check("lm28", log[3] == "Dlist.index(x): x not in list")
    check("lm29", log[4] == "E'int' object is not iterable")
    check("lm30", log[5] == "Fsort() takes no positional arguments")
    check("lm31", len(log) == 6)
    # A ValueError from index() is a subclass of Exception and unwinds normally.
    def find(seq, v):
        try:
            return seq.index(v)
        except ValueError:
            return -1
    check("lm32", find([1, 2], 2) == 1 and find([1, 2], 9) == -1)
    # --- hasattr / getattr over BUILT-IN methods ---------------------------
    names = ["append", "pop", "clear", "copy", "count", "sort", "insert",
             "remove", "extend", "index", "reverse"]
    seen = 0
    for nm in names:
        if hasattr([3, 1, 2], nm):
            seen += 1
    check("lm33", seen == 11)
    check("lm34", hasattr([3, 1, 2], "count") and hasattr("s", "upper"))
    check("lm35", hasattr({"a": 1}, "get") and hasattr({1, 2}, "add"))
    # A name no type has stays False, and so does a foreign one.
    check("lm36", not hasattr([3, 1, 2], "nope") and not hasattr("s", "charAt"))
    check("lm37", not hasattr([3, 1, 2], "length") and not hasattr(5, "upper"))
    # getattr hands back a BOUND method that actually runs.
    check("lm38", getattr([3, 1, 2], "count")(1) == 1)
    check("lm39", getattr("ab", "upper")() == "AB")
    gl = [5, 2, 9]
    getattr(gl, "sort")()
    check("lm40", gl == [2, 5, 9])
    check("lm41", getattr({"a": 7}, "get")("a") == 7)
    check("lm42", getattr([3, 1, 2], "nope", "dflt") == "dflt")
    # __class__ is on every value, and it is the object type() hands out.
    check("lm43", [].__class__ is list and "s".__class__ is str)
    check("lm44", (3).__class__ is int and None.__class__ is type(None))
    check("lm45", hasattr(3, "__class__") and hasattr(None, "__class__"))
    # A user class is unaffected by any of it.
    class C:
        def __init__(self):
            self.v = 1

        def m(self):
            return 2
    c = C()
    check("lm46", hasattr(c, "v") and hasattr(c, "m") and not hasattr(c, "z"))
    check("lm47", getattr(c, "v") == 1 and getattr(c, "m")() == 2)
    # A Python class that DEFINES one of these names keeps its own.
    class L:
        def sort(self):
            return "mine"
    check("lm48", L().sort() == "mine" and hasattr(L(), "sort"))


# ===== SECTION 36: dict and set methods =====
# dict.clear/copy/setdefault/update and set.pop/clear/copy/remove/discard/
# union/update. Every value is read out of a list so the constant folder cannot
# pre-compute a row, and every error is CAUGHT, which is the whole point of
# raising a real Python exception instead of aborting.
def s36():
    src = [["a", 1], ["b", 2], ["c", 3]]

    def mkd():
        d = {}
        for kv in src:
            d[kv[0]] = kv[1]
        return d

    # --- dict.copy is SHALLOW -------------------------------------------
    d = mkd()
    c = d.copy()
    c["d"] = 4
    check("dm01", d == mkd() and len(c) == 4 and c is not d)
    inner = [1, 2]
    nd = {"k": inner}
    nc = nd.copy()
    nc["k"].append(3)
    check("dm02", nd["k"] is nc["k"] and len(inner) == 3)
    # --- dict.clear -----------------------------------------------------
    e = mkd()
    check("dm03", e.clear() is None and e == {} and len(e) == 0)
    check("dm04", list(e.keys()) == [] and ("a" in e) == False)
    # --- dict.setdefault INSERTS and returns -----------------------------
    sd = mkd()
    check("dm05", sd.setdefault("a", 99) == 1 and len(sd) == 3)
    check("dm06", sd.setdefault("z", 9) == 9 and sd["z"] == 9 and len(sd) == 4)
    # The default DEFAULTS TO None, and None is inserted too.
    check("dm07", sd.setdefault("n") is None and "n" in sd and sd["n"] is None)
    # --- dict.update: mapping, pairs, keywords, and all of them ----------
    u = mkd()
    check("dm08", u.update({"a": 9, "d": 4}) is None)
    check("dm09", u["a"] == 9 and u["d"] == 4 and len(u) == 4)
    u2 = mkd()
    u2.update([["a", 9], ["d", 4]])
    check("dm10", u2["a"] == 9 and u2["d"] == 4 and len(u2) == 4)
    # A two-CHARACTER string is a valid pair, as it is in CPython.
    u3 = {}
    u3.update(["ab"])
    check("dm11", u3 == {"a": "b"})
    # KEYWORD arguments, which only a WRITTEN call can carry.
    u4 = mkd()
    u4.update(a=9, d=4)
    check("dm12", u4["a"] == 9 and u4["d"] == 4 and len(u4) == 4)
    u5 = mkd()
    u5.update([["d", 4]], b=8)
    check("dm13", u5["b"] == 8 and u5["d"] == 4 and len(u5) == 4)
    u6 = mkd()
    check("dm14", u6.update() is None and u6 == mkd())
    # Insertion ORDER: an updated key keeps its place, a new one goes last.
    u7 = mkd()
    u7.update([["a", 9], ["d", 4]])
    check("dm15", list(u7.keys()) == ["a", "b", "c", "d"])

    # --- set.copy is SHALLOW, set.clear ---------------------------------
    els = [1, 2, 3]

    def mks():
        s = set()
        for x in els:
            s.add(x)
        return s

    s = mks()
    sc = s.copy()
    sc.add(9)
    check("dm16", len(s) == 3 and len(sc) == 4 and (9 in s) == False)
    sl = mks()
    check("dm17", sl.clear() is None and len(sl) == 0 and (1 in sl) == False)
    # --- set.pop removes an ARBITRARY element ---------------------------
    # CPython promises NO order, so only the invariants are assertable:
    # the size drops by one, the value was a member and no longer is.
    sp = mks()
    v = sp.pop()
    check("dm18", len(sp) == 2 and v in s and (v in sp) == False)
    # --- remove RAISES on a missing member and discard does NOT ----------
    sr = mks()
    check("dm19", sr.remove(2) is None and len(sr) == 2 and (2 in sr) == False)
    sd2 = mks()
    check("dm20", sd2.discard(2) is None and len(sd2) == 2)
    check("dm21", sd2.discard(99) is None and len(sd2) == 2)
    # --- set.union takes ANY iterable, several of them, and does not mutate
    su = mks()
    un = su.union([4], "x", {5})
    check("dm22", len(su) == 3 and len(un) == 6)
    check("dm23", 4 in un and "x" in un and 5 in un and 1 in un)
    check("dm24", len(mks().union()) == 3)
    # --- set.update is the same walk IN PLACE, returning None ------------
    sup = mks()
    check("dm25", sup.update([4], "x") is None and len(sup) == 5)
    check("dm26", 4 in sup and "x" in sup)

    # --- every error is a CATCHABLE exception with CPython's own text -----
    log = []

    def note(tag, f):
        try:
            f()
            log.append(tag + "-none")
        except Exception as ex:
            log.append(tag + type(ex).__name__ + ":" + str(ex))

    note("A", lambda: set().pop())
    note("B", lambda: mks().remove(99))
    note("C", lambda: mkd().pop("zz"))
    note("D", lambda: mkd().update([[1]]))
    note("E", lambda: mkd().update([1]))
    note("F", lambda: mkd().update(1))
    note("G", lambda: mkd().update({}, {}))
    note("H", lambda: mkd().setdefault())
    note("I", lambda: mks().union(1))
    check("dm27", log[0] == "AKeyError:'pop from an empty set'")
    check("dm28", log[1] == "BKeyError:99")
    check("dm29", log[2] == "CKeyError:'zz'")
    check("dm30", log[3] == "DValueError:dictionary update sequence element #0 has length 1; 2 is required")
    check("dm31", log[4] == "ETypeError:object is not iterable")
    check("dm32", log[5] == "FTypeError:'int' object is not iterable")
    check("dm33", log[6] == "GTypeError:update expected at most 1 argument, got 2")
    check("dm34", log[7] == "HTypeError:setdefault expected at least 1 argument, got 0")
    check("dm35", log[8] == "ITypeError:'int' object is not iterable")

    # --- a name the OTHER type owns raises AttributeError -----------------
    note("J", lambda: mks().keys())
    note("K", lambda: mks().items())
    note("L", lambda: mks().setdefault(1))
    note("M", lambda: mkd().discard("a"))
    note("N", lambda: mkd().union([1]))
    check("dm36", log[9] == "JAttributeError:'set' object has no attribute 'keys'")
    check("dm37", log[10] == "KAttributeError:'set' object has no attribute 'items'")
    check("dm38", log[11] == "LAttributeError:'set' object has no attribute 'setdefault'")
    check("dm39", log[12] == "MAttributeError:'dict' object has no attribute 'discard'")
    check("dm40", log[13] == "NAttributeError:'dict' object has no attribute 'union'")

    # --- hasattr answers exactly what the dispatcher can run --------------
    dnames = ["keys", "values", "items", "get", "pop", "clear", "copy",
              "setdefault", "update", "popitem"]
    snames = ["add", "pop", "clear", "copy", "remove", "discard", "union",
              "update"]
    dseen = 0
    for nm in dnames:
        if hasattr(mkd(), nm):
            dseen += 1
    sseen = 0
    for nm in snames:
        if hasattr(mks(), nm):
            sseen += 1
    check("dm41", dseen == 10 and sseen == 8)
    # And False for the OTHER type's names, and for what this project lacks.
    check("dm42", not hasattr(mks(), "keys") and not hasattr(mkd(), "discard"))
    # popitem is a DICT name only, so a set answers False for it - as CPython
    # does.
    check("dm43", hasattr(mkd(), "popitem") and not hasattr(mks(), "popitem"))
    # issubset is a SET name: it takes any iterable, as union does.
    check("dm43b", hasattr(mks(), "issubset") and not hasattr(mkd(), "issubset"))
    check("dm43c", mks().issubset([3, 2, 1, 0]) and not mks().issubset({1: 0, 2: 0}) and
          set().issubset("") and getattr(mks(), "issubset")(range(4)))
    # getattr hands back a BOUND method that runs. It cannot carry KEYWORD
    # arguments, so update()'s keyword form is only available on a written call.
    gd = mkd()
    check("dm44", getattr(gd, "setdefault")("z", 9) == 9 and gd["z"] == 9)
    gs = mks()
    check("dm45", getattr(gs, "discard")(1) is None and len(gs) == 2)
    check("dm46", len(getattr(mks(), "copy")()) == 3)
    # The bound items() and keys() run too.
    check("dm47", [list(p) for p in getattr(mkd(), "items")()] == [["a", 1], ["b", 2], ["c", 3]])
    check("dm48", list(getattr(mkd(), "keys")()) == ["a", "b", "c"])

    # --- the shapes NEXT DOOR --------------------------------------------
    # Key ALIASING survives copy/setdefault/update: a bool, an int and a float
    # of the same value are ONE key (docs/working-on-this-project.md 7.6).
    ka = {True: "t"}
    check("dm49", ka.get(1) == "t" and ka.get(1.0) == "t")
    kb = ka.copy()
    kb[1.0] = "f"
    check("dm50", len(kb) == 1 and kb.get(True) == "f")
    kc = {1: "a"}
    kc.setdefault(1.0, "b")
    check("dm51", len(kc) == 1 and kc[1] == "a")
    kd = {}
    kd.update([[1.0, "a"], [1, "b"]])
    check("dm52", len(kd) == 1)
    ks = set()
    ks.add(1)
    ks.discard(1.0)
    check("dm53", len(ks) == 0)
    # Iteration and membership after the new mutators have run.
    it = mkd()
    it.pop("a")
    it.setdefault("d", 4)
    order = ""
    for k in it:
        order += k
    check("dm54", order == "bcd" and len(it) == 3 and ("a" in it) == False)
    check("dm55", [kv[1] for kv in it.items()] == [2, 3, 4])
    # The set OPERATORS: <= / >= are subset / superset, not the host relational
    # operator on two objects.
    o1 = mks()
    o2 = set()
    o2.add(els[0])
    check("dm56", o2 <= o1 and o1 >= o2 and o2 < o1 and (o1 < o1) == False)
    check("dm57", (o1 <= o2) == False and (o1 > o1) == False and o1 <= o1)
    check("dm58", len(o1 | o2) == 3 and len(o1 & o2) == 1 and len(o1 - o2) == 2)

# ===== SECTION 38: super(), the MRO, dict.popitem and bound methods =====
# super() (zero- and two-argument), dict.popitem, C3 member resolution - so
# `class D(B, C)` finds the right method in a diamond - and a method read off an
# instance comes back bound (`f = a.m; f()`). Every operand comes out of a list
# so the constant folder cannot pre-compute a row.
def s38():
    names = ["A", "B", "C", "D"]
    vals = [1, 2, 3]

    class A:
        def __init__(self, v):
            self.v = v
        def who(self):
            return names[0]
        def tag(self):
            return "t" + self.who()

    class B(A):
        def __init__(self, v):
            super().__init__(v * 2)
        def who(self):
            return names[1] + super().who()

    class C(A):
        def who(self):
            return names[2]

    class D(B, C):
        def who(self):
            return names[3] + super().who()

    # --- zero-argument super() finds the DEFINING class, not type(self) ---
    b = B(vals[0])
    check("su01", b.v == 2 and b.who() == "BA")
    # An inherited method still dispatches virtually through self.
    check("su02", b.tag() == "tBA")
    # --- the diamond: D -> B -> C -> A, which is C3 and NOT depth first ---
    d = D(vals[1])
    check("su03", d.who() == "DBC" and d.v == 4)
    check("su04", [k.__name__ for k in type(d).__mro__] == ["D", "B", "C", "A", "object"])
    check("su05", [k.__name__ for k in D.__bases__] == ["B", "C"])
    # B's own super() lands on C when the instance is a D, and on A when it is a
    # B - the whole point of a linearization rather than a parent pointer.
    check("su06", B(vals[0]).who() == "BA" and D(vals[0]).who() == "DBC")
    # A class that DOES NOT override still resolves through the linearization.
    class E(B, C):
        pass
    check("su07", E(vals[0]).who() == "BC")
    # --- the explicit two-argument form ---------------------------------
    check("su08", super(D, d).who() == "BC")
    check("su09", super(B, d).who() == "C")
    # --- super() on a class deriving from a builtin exception -----------
    class Err(Exception):
        def __init__(self, m):
            super().__init__("E:" + m)
    caught = []
    try:
        raise Err(names[0])
    except Err as ex:
        caught.append(str(ex))
    check("su10", caught == ["E:A"])
    # --- super() as a VALUE: the member comes back bound -----------------
    class F(A):
        def who(self):
            f = super().who
            return "F" + f()
    check("su11", F(vals[0]).who() == "FA")
    # --- and a plain method read off an instance is bound too ------------
    m = A(vals[2]).tag
    check("su12", m() == "tA")
    check("su13", A(vals[2]).who() == "A")
    # --- a name super() has nowhere to find is an AttributeError ---------
    class G(A):
        def bad(self):
            return super().nope
    hit = 0
    try:
        G(vals[0]).bad()
    except AttributeError:
        hit = 1
    check("su14", hit == 1)

    # --- dict.popitem: LIFO, and a KeyError on an empty dict -------------
    pairs = [["a", 1], ["b", 2], ["c", 3]]

    def mkd():
        out = {}
        for kv in pairs:
            out[kv[0]] = kv[1]
        return out

    p = mkd()
    check("su15", p.popitem() == ("c", 3) and len(p) == 2)
    check("su16", p.popitem() == ("b", 2) and list(p.keys()) == ["a"])
    check("su17", p.popitem() == ("a", 1) and p == {})
    msg = ""
    try:
        p.popitem()
    except KeyError as ke:
        msg = str(ke)
    check("su18", msg == "'popitem(): dictionary is empty'")
    # It is a DICT method, not a set one, so a set answers AttributeError.
    sh = 0
    try:
        st = set()
        st.add(pairs[0][0])
        st.popitem()
    except AttributeError:
        sh = 1
    check("su19", sh == 1)
    check("su20", hasattr(mkd(), "popitem") and hasattr(D, "__mro__"))

    # --- __slots__ applies over the WHOLE MRO, not the nearest declaration ---
    # `class SB(SA)` where BOTH declare __slots__: writing SA's own slot name
    # through an SB is allowed in CPython - the lookup must not stop at SB's
    # declaration.
    class SA:
        __slots__ = ["s"]

    class SB(SA):
        __slots__ = ["t"]

    sb = SB()
    sb.s = vals[0]
    sb.t = vals[1]
    check("su21", sb.s == 1 and sb.t == 2)
    blocked = 0
    try:
        sb.zz = vals[0]
    except AttributeError:
        blocked = 1
    check("su22", blocked == 1)
    # A class on the MRO with NO __slots__ gives its instances a __dict__, so
    # anything may be written from there down.
    class SC(SA):
        pass

    sc = SC()
    sc.anything = vals[0]
    check("su23", sc.anything == 1)


# ===== SECTION 39: iterator cursors, delegated send, and builtin repr =====
#
# A CURSOR (what iter/map/zip/enumerate/filter/reversed answer) is NOT a
# generator: `it.send(7)` and `it.close()` raise AttributeError in CPython. A
# structural "has a callable next" test (layer 2's pyIsGen) would call it one.
#
# This section is a RATCHET section and not only a feature-matrix one on
# purpose: the feature matrix's native rows only BUILD, and most of what is
# asserted here lives in languages/lib/python-rt.metajs, which only a native RUN
# can see.
def s39():
    vals = [[1, 2, 3, 4], "ab", {"a": 1, "b": 2}, {7, 8}]

    # ----- a cursor carries __next__ and __iter__ and nothing else -----
    it = iter(vals[0])
    check("it01", it.__next__() == 1)
    check("it02", it.__iter__() is it)
    check("it03", next(it) == 2)
    # What is LEFT of it, and the read exhausts it.
    check("it04", list(it) == [3, 4])
    check("it05", list(it) == [])

    def denied(f):
        try:
            f()
        except AttributeError:
            return 1
        except Exception:
            return 2
        return 0

    it2 = iter(vals[0])
    check("it06", denied(lambda: it2.send(None)) == 1)
    check("it07", denied(lambda: it2.send(7)) == 1)
    check("it08", denied(lambda: it2.close()) == 1)
    check("it09", denied(lambda: it2.throw(ValueError("x"))) == 1)
    check("it10", denied(lambda: it2.nosuch()) == 1)
    # The message names the CURSOR's own type, not "object".
    msg = ""
    try:
        it2.send(7)
    except AttributeError as e:
        msg = str(e)
    check("it11", msg == "'list_iterator' object has no attribute 'send'")

    # CPython's own cursor type names, which type(it).__name__ reads.
    check("it12", type(iter(vals[0])).__name__ == "list_iterator")
    check("it13", type(iter(vals[1])).__name__ == "str_ascii_iterator")
    check("it14", type(iter(vals[2])).__name__ == "dict_keyiterator")
    check("it15", type(iter(vals[3])).__name__ == "set_iterator")
    check("it16", type(iter("\u00e4b")).__name__ == "str_iterator")
    check("it17", type(map(lambda x: x, vals[0])).__name__ == "map")
    check("it18", type(iter(iter(vals[0]))).__name__ == "list_iterator")

    # An exhausted cursor raises StopIteration from __next__, not a hard stop.
    it3 = iter([])
    stopped = 0
    try:
        it3.__next__()
    except StopIteration:
        stopped = 1
    check("it19", stopped == 1)

    # ----- the sites that must not take a cursor for a generator -----
    check("it20", sum(iter(vals[0])) == 10)
    check("it21", max(iter(vals[0])) == 4)
    xs = [9, 9, 9, 9]
    xs[1:3] = iter([5, 6])
    check("it22", xs == [9, 5, 6, 9])
    # `in` CONSUMES a cursor up to the match and leaves the rest.
    it4 = iter(vals[0])
    check("it23", 2 in it4)
    check("it24", list(it4) == [3, 4])
    check("it25", 99 not in iter(vals[0]))

    # ----- `yield from`: a non-generator delegate has no send() -----
    def over_list():
        yield from vals[0]

    g = over_list()
    check("it26", next(g) == 1)
    check("it27", denied(lambda: g.send(7)) == 1)
    # The AttributeError was raised INSIDE the generator, so it is finished.
    ended = 0
    try:
        g.send(None)
    except StopIteration:
        ended = 1
    check("it28", ended == 1)
    # None still delegates, one value at a time.
    g2 = over_list()
    check("it29", [next(g2), g2.send(None), g2.send(None)] == [1, 2, 3])

    # A GENERATOR delegate does receive the sent value.
    def inner():
        got = yield 10
        yield got

    def outer():
        yield from inner()

    g3 = outer()
    check("it30", next(g3) == 10)
    check("it31", g3.send(5) == 5)

    # A generator whose body RAISES is finished: the next step is StopIteration
    # and not the same exception replayed.
    def boom():
        yield 1
        raise ValueError("bang")

    g4 = boom()
    check("it32", next(g4) == 1)
    raised = 0
    try:
        next(g4)
    except ValueError:
        raised = 1
    check("it33", raised == 1)
    again = 0
    try:
        next(g4)
    except StopIteration:
        again = 1
    check("it34", again == 1)

    # ----- how a BUILTIN prints -----
    fns = [len, abs, sorted, print, isinstance, getattr]
    fnames = ["len", "abs", "sorted", "print", "isinstance", "getattr"]
    ok = 0
    for i in range(len(fns)):
        if str(fns[i]) == "<built-in function " + fnames[i] + ">":
            ok += 1
    check("it35", ok == 6)
    # The six builtins that really are TYPES print as classes.
    clss = [range, enumerate, zip, map, filter, reversed]
    cnames = ["range", "enumerate", "zip", "map", "filter", "reversed"]
    ok2 = 0
    for i in range(len(clss)):
        if str(clss[i]) == "<class '" + cnames[i] + "'>":
            ok2 += 1
    check("it36", ok2 == 6)
    check("it37", str(str) == "<class 'str'>")
    check("it38", repr(len) == "<built-in function len>")

    # A USER function keeps its own name: the two tables are keyed by identity,
    # so a def that SHADOWS a builtin name renders as the user's function.
    def sorted_like(q):
        return q

    check("it39", "sorted_like" in str(sorted_like) and "built-in" not in str(sorted_like))
    lam = lambda q: q
    check("it40", "<lambda>" in str(lam))

    # ----- and what a BUILTIN's __name__ answers -----
    # The render table above and the def-name table are the SAME table read two
    # ways, so a builtin has a __name__. CPython 3.14.6 is the oracle for every
    # row here.
    nok = 0
    for i in range(len(fns)):
        if fns[i].__name__ == fnames[i]:
            nok += 1
    check("it41", nok == 6)
    # The class-form builtins drop the '*' the code carries: map.__name__ is
    # 'map', not '*map' and not "<class 'map'>".
    nok2 = 0
    for i in range(len(clss)):
        if clss[i].__name__ == cnames[i]:
            nok2 += 1
    check("it42", nok2 == 6)
    # The def-name table is UNDISTURBED: a user def, a lambda, and a def that
    # shadows a builtin name.
    check("it43", sorted_like.__name__ == "sorted_like" and lam.__name__ == "<lambda>")
    check("it44", str.__name__ == "str" and int.__name__ == "int")
    # hasattr has to agree with getattr, because the three-argument getattr asks
    # it first and would otherwise hand back its default for a readable name.
    check("it45", hasattr(len, "__name__") and getattr(len, "__name__", "NOPE") == "len")
    check("it46", hasattr(map, "__name__") and getattr(map, "__name__", "NOPE") == "map")
    check("it47", hasattr(str, "__name__") and getattr(str, "__name__", "NOPE") == "str")
    check("it48", getattr(sorted_like, "__name__", "NOPE") == "sorted_like")

    # ----- a BOUND METHOD's introspection -----
    # getattr([1,2], "count").__name__ is 'count': a bound method is a FRESH
    # closure and neither name table is keyed to find it. Every name below is read
    # out of a LIST so the constant folder cannot answer it.
    mnames = ["count", "upper", "keys", "m"]
    bl = getattr([3, 1, 2], mnames[0])
    bs = getattr("ab", mnames[1])
    bd = getattr({"a": 1}, mnames[2])
    check("it49", bl.__name__ == "count" and bs.__name__ == "upper" and bd.__name__ == "keys")
    # __qualname__ is CPython's exactly, because a bound method knows its receiver.
    check("it50", bl.__qualname__ == "list.count" and bs.__qualname__ == "str.upper")
    check("it51", bd.__qualname__ == "dict.keys")
    # __self__ IS the receiver, by identity for the list.
    gl2 = [3, 1, 2]
    check("it52", getattr(gl2, mnames[0]).__self__ is gl2 and bs.__self__ == "ab")
    # A BUILT-IN method has no __func__ - CPython raises AttributeError, and so
    # does every engine here, catchably.
    nof = 0
    try:
        bl.__func__
    except AttributeError:
        nof = 1
    check("it53", nof == 1)
    # hasattr has to agree with getattr on every one of them.
    check("it54", hasattr(bl, "__name__") and hasattr(bl, "__self__") and not hasattr(bl, "__func__"))
    # type() names the two CPython callable types.
    check("it55", type(bl).__name__ == "builtin_function_or_method")
    check("it56", type(len).__name__ == "builtin_function_or_method")
    check("it57", type(sorted_like).__name__ == "function" and type(lam).__name__ == "function")
    # ... and 'type' for a class object.
    check("it58", type(str).__name__ == "type" and type(map).__name__ == "type")

    class Bm:
        def mth(self, q):
            return q + 1

    bi = Bm()
    bnd = getattr(bi, mnames[3] + "th")
    # A class defined inside a function is qualified by it.
    check("it59", bnd.__name__ == "mth" and bnd.__qualname__ == "s39.<locals>.Bm.mth")
    check("it60", bnd.__self__ is bi and type(bnd).__name__ == "method")
    check("it61", bnd.__func__ is Bm.mth and bnd(1) == 2)
    # How the two forms PRINT. CPython appends the receiver's address to both and
    # the address cannot be matched, so it is left off - the same rule <function f>
    # and <map object> already follow. The rest is byte-for-byte CPython.
    check("it62", str(bl).startswith("<built-in method count of list object"))
    check("it63", str(bnd).startswith("<bound method ") and "Bm.mth of <" in str(bnd))
    # __qualname__ of a plain function and of a class names the scopes it was
    # defined in; a builtin's is its name.
    check("it64", sorted_like.__qualname__ == "s39.<locals>.sorted_like" and lam.__qualname__ == "s39.<locals>.<lambda>")
    check("it65", len.__qualname__ == "len" and str.__qualname__ == "str" and map.__qualname__ == "map")
    check("it66", Bm.__qualname__ == "s39.<locals>.Bm" and Bm.mth.__qualname__ == "s39.<locals>.Bm.mth")

    class Outer:
        class Inner:
            def f(self):
                return 0
    class Sub(Outer.Inner):
        pass
    oq = Outer.Inner.__qualname__
    check("it67", oq == "s39.<locals>.Outer.Inner" and Outer.Inner.__name__ == "Inner"
          and Sub().f.__qualname__ == oq + ".f" and repr(Outer.Inner).endswith("." + oq + "'>")
          and str(Outer.Inner.f).startswith("<function " + oq + ".f")
          and str(Sub().f).startswith("<bound method " + oq + ".f of "))

    # ----- an attribute a value does NOT have raises, catchably -----
    # It raises CPython's exact text, catchably, for a builtin value and for a
    # class object alike.
    misses = [[3, 1, 2], "ab", {"a": 1}, bi, Bm, 5]
    mwant = ["'list' object has no attribute 'nope_xyz'",
             "'str' object has no attribute 'nope_xyz'",
             "'dict' object has no attribute 'nope_xyz'",
             "'Bm' object has no attribute 'nope_xyz'",
             "type object 'Bm' has no attribute 'nope_xyz'",
             "'int' object has no attribute 'nope_xyz'"]
    mok = 0
    for i in range(len(misses)):
        try:
            getattr(misses[i], "nope_xyz")
        except AttributeError as ex:
            if str(ex) == mwant[i]:
                mok += 1
    check("it67", mok == 6)
    check("it68", getattr(misses[0], "nope_xyz", "dflt") == "dflt" and not hasattr(misses[4], "nope_xyz"))

    # ----- an except HANDLER BODY IS NOT A SCOPE -----
    # A name FIRST bound inside a handler survives it (`except E: r = 1` then
    # print(r)).
    try:
        raise ValueError(mnames[0])
    except ValueError as ex2:
        hv1 = "caught " + str(ex2)
    check("it69", hv1 == "caught count")

    def handler_scope():
        try:
            raise KeyError("k")
        except KeyError:
            hv2 = 41
        return hv2 + 1

    check("it70", handler_scope() == 42)
    # The `as` name does NOT escape the handler: CPython DELETES it, so an outer
    # binding of the same name is deleted too and a later read raises.
    ex2 = "outer"
    try:
        raise ValueError("v")
    except ValueError as ex2:
        pass
    try:
        after_as = ex2
    except Exception:
        after_as = "DELETED"
    check("it71", after_as == "DELETED")

    # ----- repr() of the three host-backed shapes -----
    # A GENERATOR, a compiled PATTERN and a MATCH, through str() and repr().
    # CPython appends an address to all three and names the generating function;
    # neither is recoverable here, so the short form is printed, as <map object>
    # already is.
    def gsrc():
        yield 1

    gob = gsrc()
    check("it72", repr(gob).startswith("<generator object") and type(gob).__name__ == "generator")
    rpat = re.compile("a(b)")
    rmat = rpat.search("ab")
    check("it73", repr(rpat) == "re.compile('a(b)')")
    check("it74", repr(rmat) == "<re.Match object; span=(0, 2), match='ab'>")
    # ... and as a CONTAINER element, which is the path str() never reached.
    check("it75", repr([rpat]) == "[re.compile('a(b)')]")


# ===== SECTION 40: a name with no binding is a catchable NameError =====
# Reading a name that is nowhere bound raises a NameError a plain try/except
# recovers. The site is the FLOOR's scope_get (and abnf/jsrt.go's scopeGet),
# shared by all sixteen languages, so it is reached through a per-language hook
# the emitter registers (js_rt_missvar); a language that registers none is
# unchanged, which is what keeps the three SHOULD-ABORT matrix rows where they
# are.
#
# The `del` rows ride along: CPython 3.14 answers UnboundLocalError for a
# deleted LOCAL and NameError for a deleted GLOBAL.
#
# Every row is CPython 3.14.6's answer, verified against it.
def s40():
    r1 = "none"
    try:
        r1 = str(nosuchname_zz)
    except NameError as e:
        r1 = f"{type(e).__name__}|{e}|{isinstance(e, Exception)}"
    check("nm1", r1 == "NameError|name 'nosuchname_zz' is not defined|True")

    # The program CONTINUES afterwards.
    n = 0
    for _ in range(3):
        try:
            n = n + nosuchname_zz
        except NameError:
            n = n + 1
    check("nm2", n == 3)

    r2 = "none"
    try:
        nosuchname_zz
    except Exception as e:
        r2 = type(e).__name__
    check("nm3", r2 == "NameError")

    log = []
    try:
        try:
            log.append("t")
            nosuchname_zz
        finally:
            log.append("f")
    except NameError:
        log.append("c")
    check("nm4", ",".join(log) == "t,f,c")

    def inner2():
        y = 1
        del y
        try:
            return str(y)
        except NameError as e:
            return type(e).__name__
    check("nm5", inner2() == "UnboundLocalError")
    check("nm6", issubclass(UnboundLocalError, NameError))

    # None has no attributes and is not subscriptable: `None.x` and `None[0]`
    # raise catchably.
    nones = [None, 1]
    r4 = "none"
    try:
        r4 = str(nones[0].nope)
    except AttributeError as e:
        r4 = f"{type(e).__name__}|{e}"
    check("nm8", r4 == "AttributeError|'NoneType' object has no attribute 'nope'")
    r5 = "none"
    try:
        r5 = str(nones[0][0])
    except TypeError as e:
        r5 = f"{type(e).__name__}|{e}"
    check("nm9", r5 == "TypeError|'NoneType' object is not subscriptable")
    check("nm10", nones[0].__class__ is type(None))

    global nm_gzz
    nm_gzz = 1
    del nm_gzz
    r3 = "none"
    try:
        r3 = str(nm_gzz)
    except NameError as e:
        r3 = type(e).__name__
    check("nm7", r3 == "NameError")


# ===== SECTION 41: case mapping =====
# CPython maps case FULLY ("\u00df".upper() is "SS", "\u0130".lower() is "i"
# plus U+0307) with Final_Sigma in lower(), title(), capitalize() and
# swapcase(), and casefold() is the full case folding.
#
# Every character is built with chr(<code>) rather than written as a literal -
# manual chapter 4, test specific characters by CODE.
def s41():
    check("cmp4", ord(chr(0x0131).upper()) == 0x0049 and ord(chr(0x03C2).upper()) == 0x03A3 and
                  ord(chr(0x1E9E).lower()) == 0x00DF)
    # Pairs added in Unicode 16: the case table must not be older than that.
    u16 = [0x019B, 0x0264, 0x1C89, 0xA7DC]
    check("cmp5", chr(u16[0]).upper() == chr(0xA7DC) and chr(u16[1]).upper() == chr(0xA7CB) and
                  chr(u16[2]).lower() == chr(0x1C8A) and chr(u16[3]).lower() == chr(0x019B))
    sz = chr(0xDF)
    sg = chr(0x3A3)
    check("cmp6", (sz + "x").upper() == "SSX" and chr(0x130).lower() == "i" + chr(0x307) and
                  chr(0xFB01).upper() == "FI" and len(chr(0x390).upper()) == 3 and chr(0x1FB3).upper() == chr(0x391) + chr(0x399))
    check("cmp7", ("A" + sg).lower() == "a" + chr(0x3C2) and (sg + "A").lower() == chr(0x3C3) + "a" and
                  ("A" + sg + ".").lower() == "a" + chr(0x3C2) + "." and ("1" + sg).lower() == "1" + chr(0x3C3))
    check("cmp8", (sz + "a").title() == "Ssa" and (sz + "a").capitalize() == "Ssa" and (sz + "A").swapcase() == "SSa" and
                  ("A" + sg + " " + sg).title() == "A" + chr(0x3C2) + " " + sg and (sz + chr(0x3C2)).casefold() == "ss" + chr(0x3C3))


# ===== SECTION 42: an unhashable set member or dict key =====
# `a = [1]; b = [1]; len({a, b})` raises TypeError in CPython. A list, a dict, a
# set and a tuple holding one are refused, in all three engines, with CPython
# 3.14's exact wording - which differs between the two positions:
#   {[1]}    cannot use 'list' as a set element (unhashable type: 'list')
#   {{}: 1}  cannot use 'dict' as a dict key (unhashable type: 'dict')
# with one exception CPython really has: set.__contains__ / .discard / .remove
# convert a SET argument to a frozenset instead of refusing it, so `{1} in {1}`
# is False while `{} in {1}` and `{1} in {1: 2}` both raise. A tuple key compares
# by CONTENTS (`d[1, 2] = v; d[1, 2]`).
#
# Every value is read out of a list so the constant folder cannot pre-compute a
# row, and every error is CAUGHT.
def s42():
    def msg(f):
        try:
            f()
            return "no raise"
        except TypeError as e:
            return str(e)

    boxes = [{}, {1}, [1], (1, 2)]
    empty_d = boxes[0]
    a_set = boxes[1]
    a_list = boxes[2]
    a_tup = boxes[3]

    # --- dict and set are unhashable, and the two wordings differ ---------
    check("uh01", msg(lambda: {empty_d: 1}) ==
          "cannot use 'dict' as a dict key (unhashable type: 'dict')")
    check("uh02", msg(lambda: {a_set: 1}) ==
          "cannot use 'set' as a dict key (unhashable type: 'set')")
    check("uh03", msg(lambda: {empty_d}) ==
          "cannot use 'dict' as a set element (unhashable type: 'dict')")
    check("uh04", msg(lambda: {a_set}) ==
          "cannot use 'set' as a set element (unhashable type: 'set')")
    # Every dict WRITE and READ site, not only the literal.
    def dset(k):
        d = {}
        d[k] = 1
        return len(d)
    check("uh05", msg(lambda: dset(empty_d)) ==
          "cannot use 'dict' as a dict key (unhashable type: 'dict')")
    check("uh06", msg(lambda: {}[a_set]) ==
          "cannot use 'set' as a dict key (unhashable type: 'set')")
    check("uh07", msg(lambda: {}.get(empty_d, "d")) ==
          "cannot use 'dict' as a dict key (unhashable type: 'dict')")
    check("uh08", msg(lambda: {}.setdefault(empty_d, 1)) ==
          "cannot use 'dict' as a dict key (unhashable type: 'dict')")
    check("uh09", msg(lambda: empty_d in {1: 2}) ==
          "cannot use 'dict' as a dict key (unhashable type: 'dict')")
    # Every set INSERT site.
    def sadd(v):
        s = set()
        s.add(v)
        return len(s)
    check("uh10", msg(lambda: sadd(empty_d)) ==
          "cannot use 'dict' as a set element (unhashable type: 'dict')")
    check("uh11", msg(lambda: set([empty_d])) ==
          "cannot use 'dict' as a set element (unhashable type: 'dict')")
    check("uh12", msg(lambda: {x for x in [empty_d]}) ==
          "cannot use 'dict' as a set element (unhashable type: 'dict')")
    check("uh13", msg(lambda: sadd(a_set)) ==
          "cannot use 'set' as a set element (unhashable type: 'set')")

    # --- the frozenset conversion, which is a LOOKUP-only exception -------
    check("uh14", msg(lambda: empty_d in a_set) ==
          "cannot use 'dict' as a set element (unhashable type: 'dict')")
    check("uh15", (a_set in a_set) == False and (a_set in {1, 2}) == False)
    check("uh16", {1}.discard(a_set) is None and {1}.discard({9}) is None)
    check("uh17", msg(lambda: {1}.discard(empty_d)) ==
          "cannot use 'dict' as a set element (unhashable type: 'dict')")
    check("uh18", msg(lambda: {1}.remove(empty_d)) ==
          "cannot use 'dict' as a set element (unhashable type: 'dict')")
    # A set used as a DICT key is refused even so - the exception is set-side only.
    check("uh19", msg(lambda: a_set in {1: 2}) ==
          "cannot use 'set' as a dict key (unhashable type: 'set')")

    # --- a tuple key compares by CONTENTS, off two different objects ------
    g = {}
    g[1, 2] = "pair"
    g[3, 4] = "other"
    check("uh20", g[1, 2] == "pair" and g[3, 4] == "other" and len(g) == 2)
    check("uh21", ((1, 2) in g) and not ((9, 9) in g))
    t1 = boxes[3]
    # Built from the elements rather than written as a literal: CPython folds and
    # INTERNS a constant tuple, so a literal (1, 2) really is `is` boxes[3] there.
    t2 = (t1[0] + 0, t1[1] + 0)
    check("uh22", t1 is not t2 and {t1: "x"}[t2] == "x")
    check("uh23", len({t1, t2, (3, 4)}) == 2)
    check("uh24", len({(1, 2): "a", (1, 2): "b"}) == 1)
    # By CONTENTS means by contents: a tuple with a different element misses.
    nested = {}
    nested[a_tup] = 5
    check("uh25", nested[(1, 2)] == 5 and not ((1, 3) in nested))

    # --- a list is unhashable, and so is a tuple holding one -----------------
    def lsetlen():
        try:
            return len({[1], [1], [2]})
        except TypeError:
            return 0 - 1
    check("uh26", msg(lambda: {a_list}) == "cannot use 'list' as a set element (unhashable type: 'list')")
    check("uh27", msg(lambda: {a_list: 1}) == "cannot use 'list' as a dict key (unhashable type: 'list')")
    check("uh28", lsetlen() == 0 - 1 and msg(lambda: {(1, a_list): 1}) == "cannot use 'tuple' as a dict key (unhashable type: 'list')"
          and msg(lambda: a_list in {1: 2}) == "cannot use 'list' as a dict key (unhashable type: 'list')")

    # --- a BOXED number on the left of `in` -------------------------------
    # `in` and `not in` are CONTAINER operators, so a boxed operand (`3j in xs`,
    # `10**30 in xs`) must fall through to pyIn.
    boxes2 = [10 ** 30, 3j, 1]
    big = boxes2[0]
    cx = boxes2[1]
    check("uh29", (big in boxes2) and (cx in boxes2) and (2 not in boxes2))
    check("uh30", (big in {big}) and (cx in {cx}) and (big in {big: "b"}))
    check("uh31", (big not in [1, 2]) and (cx not in [1, 2]))


# ===== SECTION 43: global / nonlocal across the emitter's inner scopes =====
# A comprehension target, an `except ... as e` and a `with ... as w` name are the
# only names the python emitter declares BELOW a binding boundary, and the
# runtime's own chain walk - "assign where the chain already holds it" - knows
# nothing about `global` and `nonlocal`. With an intermediate function binding
# the same name, `[q for q in range(2)]` plus `global q` must still make a later
# `q = 55` write the module's q.
def s43():
    def lvl2():
        def deep():
            global s43g
            seq = [s43g for s43g in range(2)]
            s43g = 55
            return seq
        s43g = 100
        inner = deep()
        return [s43g, inner]

    def readg():
        global s43g
        return s43g

    pair = lvl2()
    check("gsc1", pair[0] == 100)                 # the intermediate local is untouched
    check("gsc2", pair[1] == [0, 1])              # the target stayed the comprehension's
    check("gsc3", readg() == 55)                  # `global` reached the module

    # `nonlocal` takes the same route, two levels down from the binding it names.
    def outn():
        n43 = 100
        def mid():
            def deep2():
                nonlocal n43
                seq = [n43 for n43 in range(3)]
                n43 = 42
                return seq
            return deep2()
        inner = mid()
        return [n43, inner]

    pair2 = outn()
    check("gsc4", pair2[0] == 42 and pair2[1] == [0, 1, 2])

    # A `global` at depth ONE (the chain walk reaches the module scope, which the
    # `global` statement itself seeds).
    def one():
        global s43h
        s43h = 3
    def readh():
        global s43h
        return s43h
    one()
    check("gsc5", readh() == 3)

    # No mark in sight: the comprehension target must not leak into the function,
    # and an `except ... as` name is rebindable after its handler.
    def leak():
        z = 7
        w = [z for z in range(3)]
        return [z, w]
    check("gsc6", leak() == [7, [0, 1, 2]])

    def exname():
        seen = []
        try:
            raise ValueError("v")
        except ValueError as e:
            seen.append(str(e))
        e = "after"
        return [seen, e]
    check("gsc7", exname() == [["v"], "after"])

    # An `except ... as x` target is a BINDING SITE like any other, so a `global x`
    # in the same function covers it and everything after it (CPython: 77) - the
    # declaration is consulted before the first scope that already has the name.
    # The `nonlocal` row is the same question one scope further in.
    def exglobal():
        global s43e
        try:
            raise ValueError("x")
        except ValueError as s43e:
            pass
        s43e = 77
    def reade():
        global s43e
        return s43e
    exglobal()
    check("gsc8", reade() == 77)

    def exnonlocal():
        w43 = 100
        def inner():
            nonlocal w43
            try:
                raise ValueError("y")
            except ValueError as w43:
                pass
            w43 = 88
        inner()
        return w43
    check("gsc9", exnonlocal() == 88)

# ===== SECTION 44: a class repr carries its module =====
# A class a `class` statement makes renders as <class '__main__.B'> and answers
# __module__; a builtin type or exception stays <class 'int'>.
class Mod44A:
    pass
class Mod44B(Mod44A):
    pass
class Mod44E(ValueError):
    pass
class Mod44C:
    __module__ = "elsewhere"
def s44():
    check("cmod1", str(Mod44B) == "<class '__main__.Mod44B'>" and repr(Mod44A) == "<class '__main__.Mod44A'>")
    check("cmod2", str(Mod44B.__mro__[1]) == "<class '__main__.Mod44A'>" and str(Mod44B.__mro__[2]) == "<class 'object'>")
    check("cmod3", Mod44B.__module__ == "__main__" and Mod44B().__module__ == "__main__")
    check("cmod4", str(Mod44E) == "<class '__main__.Mod44E'>" and str(ValueError) == "<class 'ValueError'>")
    check("cmod5", str(type(Mod44B())) == "<class '__main__.Mod44B'>" and str(int) == "<class 'int'>")
    check("cmod6", str(Mod44C) == "<class 'elsewhere.Mod44C'>" and Mod44C.__name__ == "Mod44C")

# ===== SECTION 45: a function carries attributes of its own =====
# f.k = v stores into the function's __dict__; a bound method reads its
# function's attributes and, like a builtin, refuses a new one.
class Fat45:
    def m(self):
        return 1
def s45():
    def f45():
        f45.calls += 1
        return f45.calls
    f45.calls = 0
    f45()
    f45()
    check("fat1", f45.calls == 2 and hasattr(f45, "calls") and getattr(f45, "calls") == 2)
    setattr(f45, "tag", "x")
    check("fat2", f45.tag == "x" and vars(f45) == {"calls": 2, "tag": "x"} and f45.__dict__["tag"] == "x")
    del f45.tag
    check("fat3", not hasattr(f45, "tag") and getattr(f45, "tag", "dflt") == "dflt")
    msg = ""
    try:
        del f45.tag
    except AttributeError as e:
        msg = str(e)
    check("fat4", msg == "'function' object has no attribute 'tag'")
    Fat45.m.k = 5
    check("fat5", Fat45().m.k == 5 and Fat45.m.k == 5)
    msgs = []
    for target in [len, Fat45().m]:
        try:
            target.k = 1
        except AttributeError as e:
            msgs.append(str(e))
    check("fat6", msgs == ["'builtin_function_or_method' object has no attribute 'k' and no __dict__ for setting new attributes",
                           "'method' object has no attribute 'k' and no __dict__ for setting new attributes"])
    g = lambda: 0
    g.n = 3
    check("fat7", g.n == 3 and g() == 0)

# ===== SECTION 46: an atom as the base of an attribute target =====
# `None.k = 1` parses and raises AttributeError at run time, as do the other
# atoms; the read of an augmented target and `del` start from the atom too.
def s46():
    msgs = []
    for k in range(6):
        try:
            if k == 0:
                None.k = 1
            elif k == 1:
                True.k = 1
            elif k == 2:
                {}.k = 1
            elif k == 3:
                {1}.k = 1
            elif k == 4:
                del None.k
            else:
                None.k += 1
        except AttributeError as e:
            msgs.append(str(e))
    nd = " and no __dict__ for setting new attributes"
    check("atg1", msgs[0] == "'NoneType' object has no attribute 'k'" + nd and msgs[1] == "'bool' object has no attribute 'k'" + nd)
    check("atg2", msgs[2] == "'dict' object has no attribute 'k'" + nd and msgs[3] == "'set' object has no attribute 'k'" + nd)
    check("atg3", msgs[4] == "'NoneType' object has no attribute 'k'" + nd and msgs[5] == "'NoneType' object has no attribute 'k'")
    check("atg4", len(msgs) == 6)
    m5 = ""
    try:
        del (1).k
    except AttributeError as e:
        m5 = str(e)
    xs = [1, 2, 3]
    del (xs)[0]
    check("atg5", m5 == "'int' object has no attribute 'k' and no __dict__ for setting new attributes" and xs == [2, 3])

# ===== SECTION 47: an astral character written into a string literal =====
# A character above U+FFFF typed as itself is one character, equal to the one
# chr() and the \U escape build - in plain, triple-quoted, raw and f-strings.
def s47():
    s = "𝐀"
    check("ast1", len(s) == 1 and s == chr(0x1D400) and ord(s) == 119808)
    check("ast2", len("a😀b") == 3 and "😀" in "a😀b" and "a😀b".upper() == "A😀B")
    check("ast3", len("""😀""") == 1 and len(r"😀") == 1 and len('😀x') == 2)
    check("ast4", f"𝐀{1}" == chr(0x1D400) + "1" and len(f"x{2}😀") == 3)
    check("ast5", "\U0001D400" == s and repr(s) == "'𝐀'" and ascii(s) == "'\\U0001d400'")

# ===== SECTION 48: complex values print, construct and format =====
# repr(complex) leaves out a +0.0 real part, drops each part's ".0", and keeps
# a zero imaginary part; complex() takes numbers, complexes and strings; the
# format mini-language formats each part as a float.
def s48():
    xs = [3j, 1+2j, 1.5-2j, (1+2j)-2j, 1e16j, -0j, 1e-5j]
    check("cpx1", [str(x) for x in xs] == ["3j", "(1+2j)", "(1.5-2j)", "(1+0j)", "1e+16j", "(-0-0j)", "1e-05j"])
    check("cpx2", repr([1-1j]) == "[(1-1j)]" and type((1+2j)-2j).__name__ == "complex" and str((1+2j).real) == "1.0")
    check("cpx3", str(complex(1, 2)) == "(1+2j)" and str(complex(3)) == "(3+0j)" and str(complex()) == "0j" and str(complex(2, 1j)) == "(1+0j)")
    check("cpx4", complex(" (1-2.5J) ") == 1-2.5j and complex("-j") == -1j and complex("1+j") == 1+1j and isinstance(complex(1), complex))
    errs = []
    for bad in [None, "1+2"]:
        try:
            complex(bad)
        except (TypeError, ValueError) as e:
            errs.append(str(e))
    check("cpx5", errs == ["complex() argument must be a string or a number, not NoneType", "complex() arg is a malformed string"])
    check("cpx6", format(1+2j, ">12") == "      (1+2j)" and format(1+2j, ".2f") == "1.00+2.00j" and "{:10.3f}|".format(1.5-2.25j) == "1.500-2.250j|")
    check("cpx7", format(2j, "+") == "+2j" and format(1+2j, "#") == "(1.+2.j)" and format(1234567.5+1e16j, ",") == "(1,234,567.5+1e+16j)")
    check("cpx8", format(-0j, "z.0f") == "0+0j" and f"{1-1j:.3g}" == "1-1j" and format(1+2j, "#.3") == "(1.00+2.00j)" and format(1+2j, "E") == "1.000000E+00+2.000000E+00j")
    msgs = []
    for spec in ["=10", "010", "d"]:
        try:
            format(1+2j, spec)
        except ValueError as e:
            msgs.append(str(e))
    check("cpx9", msgs == ["'=' alignment flag is not allowed in complex format specifier",
                           "Zero padding is not allowed in complex format specifier",
                           "Unknown format code 'd' for object of type 'complex'"])
    check("cpx10", (1.1+2.3j) ** 4 == complex(-8.957200000000004, -41.289599999999986) and str((1+2j) ** 0) == "(1+0j)")
    check("cpx11", (1e10-3j) * (1.1+2.3j) == complex(11000000006.9, 22999999996.7) and format(float("-inf"), "z") == "-inf")

# ===== SECTION 49: int / int is the exact quotient rounded once =====
# With an arbitrary precision operand the quotient is not the quotient of the
# two rounded doubles: it is correctly rounded (half to even, subnormals
# included), and past the double range it is an OverflowError.
def s49():
    big = [3**70, 7, 123456789012345678901234567890, 987654321, 2**64, 3, -(10**25), 7, 10**22, 3**5]
    got = [repr(big[2 * i] / big[2 * i + 1]) for i in range(len(big) // 2)]
    check("itd1", got == ["3.575936435704631e+32", "1.249999988734375e+20", "6.148914691236517e+18",
                          "-1.4285714285714285e+24", "4.1152263374485594e+19"])
    check("itd2", repr(10**400 / 10**399) == "10.0" and repr(2**1100 / 2**1000) == "1.2676506002282294e+30")
    check("itd3", repr(1 / 2**1075) == "0.0" and repr(1 / (2**1075 - 1)) == "5e-324" and repr(0 / -(10**30)) == "-0.0")
    check("itd4", repr(3 / 2**1075) == "1e-323" and repr(-1 / 2**1076) == "-0.0" and repr((2**54 + 3) / 2) == "9007199254740994.0")
    msg = ""
    try:
        print(7 * 10**330 / 10**22)
    except OverflowError as e:
        msg = str(e)
    check("itd5", msg == "integer division result too large for a float")

# ===== SECTION 50: range() takes a step =====
# range(start, stop, step) counts by step in either direction, a zero step is a
# ValueError, and a float argument is a TypeError.
def s50():
    a = [0, 10, 3, -3, 1.5]
    check("rng1", list(range(a[0], a[1], a[2])) == [0, 3, 6, 9] and list(range(a[1], a[0], a[3])) == [10, 7, 4, 1])
    check("rng2", list(range(a[2], a[1], a[3])) == [] and [i for i in range(a[1], a[2], -4)] == [10, 6] and len(range(a[0], a[1], a[2])) == 4)
    msgs = []
    for args in [[a[0], a[1], a[0]], [a[4]]]:
        try:
            range(*args)
        except (ValueError, TypeError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("rng3", msgs == ["ValueError: range() arg 3 must not be zero", "TypeError: 'float' object cannot be interpreted as an integer"])

# ===== SECTION 51: item assignment and deletion errors are catchable =====
# Writing, reading or deleting an item of a value with no such slot is CPython's
# TypeError, a list or str index must be an int, and a user class may define
# __delitem__.
def s51():
    def fn():
        pass
    class C:
        pass
    class D:
        def __delitem__(self, k):
            got.append(k)
    got = []
    for v in [None, 5, 2.5, "ab", fn, 1+2j, {1}, C()]:
        for op in range(3):
            try:
                if op == 0:
                    v[0] = 1
                elif op == 1:
                    got.append(v[1])
                else:
                    del v[0]
            except TypeError as e:
                got.append(str(e))
    check("item1", got[:6] == ["'NoneType' object does not support item assignment", "'NoneType' object is not subscriptable",
                               "'NoneType' object does not support item deletion", "'int' object does not support item assignment",
                               "'int' object is not subscriptable", "'int' object does not support item deletion"])
    check("item2", got[9:12] == ["'str' object does not support item assignment", "b", "'str' object doesn't support item deletion"])
    check("item3", got[12:] == ["'function' object does not support item assignment", "'function' object is not subscriptable",
                                "'function' object does not support item deletion", "'complex' object does not support item assignment",
                                "'complex' object is not subscriptable", "'complex' object does not support item deletion",
                                "'set' object does not support item assignment", "'set' object is not subscriptable",
                                "'set' object doesn't support item deletion", "'C' object does not support item assignment",
                                "'C' object is not subscriptable", "'C' object doesn't support item deletion"])
    xs = [1, 2]
    msgs = []
    for k in ["a", 1.5, None, 5]:
        for op in range(3):
            try:
                if op == 0:
                    xs[k] = 0
                elif op == 1:
                    msgs.append("ab"[k])
                else:
                    del xs[k]
            except (TypeError, IndexError) as e:
                msgs.append(str(e))
    check("item4", msgs[:3] == ["list indices must be integers or slices, not str", "string indices must be integers, not 'str'",
                                "list indices must be integers or slices, not str"] and msgs[5] == "list indices must be integers or slices, not float")
    check("item5", msgs[9:] == ["list assignment index out of range", "string index out of range", "list assignment index out of range"])
    del D()[3]
    xs[True] = 7
    check("item6", got[-1] == 3 and xs == [1, 7])

# ===== SECTION 52: an augmented assignment walks its target once =====
# `f().k += v` calls f once, and `a[g()] += v()` evaluates the container, then
# the subscript, then reads, then the right side, then writes.
def s52():
    log = []
    class C:
        pass
    c = C()
    c.k = 1
    a = [1, 2, 3, 4]
    def f():
        log.append("f")
        return c
    def g(i):
        log.append("g" + str(i))
        return i
    def v():
        log.append("v")
        return 5
    class K:
        def __getitem__(self, k):
            log.append("get" + str(k))
            return 10
        def __setitem__(self, k, x):
            log.append("set" + str(k) + "=" + str(x))
    f().k += v()
    a[g(1)] += v()
    a[g(0):g(2)] *= 2
    K()[g(3)] -= v()
    check("aug1", c.k == 6 and a == [1, 7, 1, 7, 3, 4])
    check("aug2", log == ["f", "v", "g1", "v", "g0", "g2", "g3", "get3", "v", "set3=5"])

# ===== SECTION 53: a \N{...} name is matched without regard to case =====
def s53():
    check("uname1", "\N{bullet}" == "\u2022" and "\N{Euro Sign}" == "\u20ac" and f"x\N{em dash}{1}" == "x\u20141")

# ===== SECTION 54: a str is indexed, sliced, iterated and searched by code point =====
# An astral character is ONE character to every position-taking operation, not
# the two UTF-16 units the strings are stored as.
def s54():
    s = "a😀b𝐀c"
    check("cp1", [ord(x) for x in chr(0x1D400)] == [119808] and [ord(x) for x in s] == [97, 128512, 98, 119808, 99])
    check("cp2", s[1] == "😀" and s[2] == "b" and s[-2] == "𝐀" and s[1:3] == "😀b" and s[::-1] == "c𝐀b😀a" and s[::2] == "abc")
    check("cp3", s.find("b") == 2 and s.index("c") == 4 and s.rfind("😀") == 1 and s.find("c", -1) == 4 and s.find("", 9) == -1)
    check("cp4", s.startswith("😀", 1) and s.endswith("𝐀", 0, 4) and s.count("b", 2) == 1 and list(s) == ["a", "😀", "b", "𝐀", "c"])
    check("cp5", s.strip("a😀c") == "b𝐀" and s.rstrip("c𝐀") == "a😀b" and max("a😀b") == "😀" and min("ba") == "a")
    check("cp6", [i for i, c in enumerate(s) if c == "𝐀"] == [3] and "".join(reversed(s)) == "c𝐀b😀a")

# ===== SECTION 55: complex abs, infinities, powers and division =====
# abs is the libm hypot; a product recovers an infinity that the textbook
# formula turns into nan+nanj (C11 Annex G); an integral power up to 100 is
# square-and-multiply and any other power the polar form; division is Smith's
# algorithm with CPython's fused multiply-adds; a real operand takes the
# mixed-mode rule, so a -0.0 imaginary part survives `z + 1.0`.
def s55():
    inf = float("inf")
    nan = float("nan")
    zs = [3+4j, complex(inf, nan), 1e10-3j, complex(1, -0.0), complex(1e308, 1.7e308)]
    check("cx1", abs(zs[0]) == 5.0 and repr(abs(-3j)) == "3.0" and abs(complex(nan, inf)) == inf and abs(0.3+0.4j) == 0.5)
    check("cx2", repr(zs[1] * (1+1j)) == "(inf+infj)" and repr((1+2j) ** -3) == "(-0.08800000000000001+0.016j)" and (1+2j) ** 0 == 1)
    check("cx3", repr((0.1+0.1j) / (0.1+0.3j)) == "(0.4-0.2j)" and repr((0.1+0.1j) / (0.3+0.9j)) == "(0.13333333333333333-0.06666666666666667j)")
    check("cx4", repr(zs[3] + 1.0) == "(2-0j)" and repr(1.0 - zs[3]) == "0j" and repr(2.0 / zs[3]) == "(2+0j)" and repr(zs[3] * 2.0) == "(2-0j)")
    check("cx5", repr((1+2j) ** 0.5) == "(1.272019649514069+0.7861513777574233j)" and repr(2 ** (1+1j)) == "(1.5384778027279442+1.2779225526272695j)")
    msgs = []
    for f in [lambda: zs[2] ** 100, lambda: abs(zs[4]), lambda: 0j ** -1, lambda: (1+2j) / 0, lambda: 1 / 0j]:
        try:
            f()
        except (OverflowError, ZeroDivisionError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("cx6", msgs == ["OverflowError: complex exponentiation", "OverflowError: absolute value too large",
                          "ZeroDivisionError: zero to a negative or complex power", "ZeroDivisionError: division by zero",
                          "ZeroDivisionError: division by zero"])


# ===== SECTION 56: `+` stays within one kind; `xs += it` extends in place =====
# str + int, int + str, list + str and None + 1 raise TypeError with CPython's
# texts; a list's `+=` is list.extend, so an alias sees it and any iterable is
# accepted; sum() folds with the same `+` and refuses a str start.
def s56():
    vals = ["a", 1, 2.5, None, [1], 1j, 10**30]
    msgs = []
    for a in vals:
        for b in vals:
            try:
                a + b
            except TypeError as e:
                msgs.append(str(e))
    check("add1", len(msgs) == 31 and msgs[0] == 'can only concatenate str (not "int") to str'
          and msgs[6] == "unsupported operand type(s) for +: 'int' and 'str'"
          and "unsupported operand type(s) for +: 'NoneType' and 'NoneType'" in msgs
          and 'can only concatenate list (not "complex") to list' in msgs
          and "unsupported operand type(s) for +: 'complex' and 'str'" in msgs)
    y = ["s"]
    try:
        y[0] += 1
    except TypeError as e:
        msgs.append(str(e))
    check("add2", msgs[-1] == 'can only concatenate str (not "int") to str' and y == ["s"] and 3 * "x" == "xxx")
    a = [1]
    b = a
    a += "ab"
    a += (i for i in range(2))
    a += a
    check("add3", b == [1, "a", "b", 0, 1, 1, "a", "b", 0, 1])
    try:
        a += 5
    except TypeError as e:
        msgs.append(str(e))
    check("add4", msgs[-1] == "'int' object is not iterable")
    check("add5", sum([[1], [2]], []) == [1, 2] and sum([1.5, 2]) == 3.5 and sum([10**30, 1]) == 10**30 + 1
          and sum([], 5) == 5 and sum(x for x in range(4)) == 6)
    for bad in [lambda: sum(["a", "b"]), lambda: sum(["a"], "")]:
        try:
            bad()
        except TypeError as e:
            msgs.append(str(e))
    check("add6", msgs[-2:] == ["unsupported operand type(s) for +: 'int' and 'str'",
                                "sum() can't sum strings [use ''.join(seq) instead]"])


# ===== SECTION 57: print() takes sep, end, file and flush =====
# sep and end are None or a str; a file other than None gets one write() per
# piece and a flush() when flush is true; an unknown keyword is a TypeError.
# The keywords also reach print through **kw, an alias and map().
def s57():
    class W:
        def __init__(self):
            self.parts = []
        def write(self, s):
            self.parts.append(s)
        def flush(self):
            self.parts.append("<flush>")
    w = W()
    print(1, "two", sep="/", end=";", file=w)
    print({"sep": 1}, file=w, flush=True)
    out = print
    out(file=w, **{"end": "", "sep": None})
    list(map(lambda x: print(x, sep="?", end=".", file=w), [7, 8]))
    check("prt1", w.parts == ["1", "/", "two", ";", "{'sep': 1}", "\n", "<flush>", "", "7", ".", "8", "."])
    msgs = []
    for bad in [{"sep": 5}, {"end": [1]}, {"bogus": 1}]:
        try:
            print("a", **bad)
        except TypeError as e:
            msgs.append(str(e))
    check("prt2", msgs == ["sep must be None or a string, not int", "end must be None or a string, not list",
                           "print() got an unexpected keyword argument 'bogus'"])
    print("prt3", "a", sep="-", end=" | ")
    print("b", end="\n", flush=True)


# ===== SECTION 58: dict() builds from a mapping, pairs and keywords =====
# dict(x, **kw) reads x by update()'s rules - a mapping is copied, any other
# iterable must yield 2-item pairs - then adds the keywords; at most one
# positional argument.
def s58():
    d = {"q": 1}
    e = dict(d)
    e["r"] = 2
    check("dic1", dict(enumerate("ab")) == {0: "a", 1: "b"} and dict(zip("ab", [1, 2])) == {"a": 1, "b": 2}
          and d == {"q": 1} and e == {"q": 1, "r": 2})
    check("dic2", dict((k, k * 2) for k in range(3)) == {0: 0, 1: 2, 2: 4} and dict([("a", 1), "cd"]) == {"a": 1, "c": "d"}
          and dict(a=1) == {"a": 1} and dict([("k", 1)], k=3) == {"k": 3} and dict(**{"m": 5}) == {"m": 5})
    check("dic3", list(map(dict, [[("z", 0)]])) == [{"z": 0}] and dict() == {} and len(dict(map(lambda i: (i, -i), [1, 2]))) == 2)
    msgs = []
    for bad in [lambda: dict(5), lambda: dict([1]), lambda: dict([(1, 2, 3)]), lambda: dict({}, {})]:
        try:
            bad()
        except (TypeError, ValueError) as ex:
            msgs.append(type(ex).__name__ + ": " + str(ex))
    check("dic4", msgs == ["TypeError: 'int' object is not iterable", "TypeError: object is not iterable",
                           "ValueError: dictionary update sequence element #0 has length 3; 2 is required",
                           "TypeError: dict expected at most 1 argument, got 2"])


# ===== SECTION 59: calling, iterating and sizing a value that cannot =====
# CPython's TypeErrors: "'C' object is not callable", "'C' object is not
# iterable", "object of type 'C' has no len()" and "cannot unpack non-iterable
# C object" - catchable, not an abort.
def s59():
    class C:
        pass
    c = C()
    five = 5
    none = None
    def unpack():
        a, b = c
    tests = [lambda: c(), lambda: five(), lambda: none(), lambda: len(c), lambda: len(5), lambda: len(None),
             lambda: iter(c), lambda: list(c), lambda: [x for x in c], lambda: tuple(c), lambda: sorted(c), unpack]
    msgs = []
    for t in tests:
        try:
            t()
        except TypeError as e:
            msgs.append(str(e))
    try:
        for x in 3:
            pass
    except TypeError as e:
        msgs.append(str(e))
    check("typ1", msgs[:6] == ["'C' object is not callable", "'int' object is not callable", "'NoneType' object is not callable",
                               "object of type 'C' has no len()", "object of type 'int' has no len()",
                               "object of type 'NoneType' has no len()"])
    check("typ2", msgs[6:] == ["'C' object is not iterable"] * 5 + ["cannot unpack non-iterable C object", "'int' object is not iterable"])


# ===== SECTION 60: a bytes literal has no \u, \U or \N escape =====
# They stay as written (a backslash and the letter), while \x, \n and the
# octal escapes still apply.
def s60():
    check("byt1", list(b"\N{BULLET}") == [92, 78, 123, 66, 85, 76, 76, 69, 84, 125] and len(b"\u00e9\U0001F600") == 16)
    check("byt2", list(b"a\x41\n\101\q") == [97, 65, 10, 65, 92, 113] and "\N{BULLET}" == "\u2022")


# ===== SECTION 61: math overflows raise instead of answering inf =====
# exp, expm1, sinh and cosh of a finite argument with an infinite result are
# OverflowError("math range error"); math.pow raises ValueError("math domain
# error") for 0 ** negative and a NaN result, OverflowError otherwise.
def s61():
    import math
    inf = float("inf")
    msgs = []
    for f in [lambda: math.exp(1000), lambda: math.expm1(800), lambda: math.sinh(-1000), lambda: math.cosh(1000),
              lambda: math.pow(10, 400), lambda: math.pow(0, -1), lambda: math.pow(-8, 0.5)]:
        try:
            f()
        except (OverflowError, ValueError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("mrg1", msgs == ["OverflowError: math range error"] * 5 + ["ValueError: math domain error"] * 2)
    check("mrg2", math.exp(inf) == inf and math.exp(-1000) == 0.0 and math.cosh(-inf) == inf and math.pow(inf, 2) == inf
          and math.pow(0.0, -inf) == inf and repr(math.exp(709)) == "8.218407461554972e+307")
    # OverflowError derives from ArithmeticError.
    caught = ""
    try:
        math.exp(1000)
    except ArithmeticError as e:
        caught = type(e).__name__
    check("mrg3", caught == "OverflowError")


# ===== SECTION 62: sorted, min, max and list.sort take key= and reverse= =====
# The keywords are keyword-only; sort and sorted are STABLE, also under
# reverse=True. min/max take default= for an empty iterable and raise a
# catchable ValueError without it. A list orders element-wise, so a key that
# returns a list (a tuple here) sorts by its fields, and a class orders by __lt__.
def s62():
    xs = [3, 1, 2, 5, 4]
    words = ["bb", "a", "ccc", "dd", "e"]
    check("srt1", sorted(xs, key=lambda x: -x) == [5, 4, 3, 2, 1] and sorted(xs, reverse=True) == [5, 4, 3, 2, 1])
    check("srt2", sorted(words, key=len) == ["a", "e", "bb", "dd", "ccc"]
          and sorted(words, key=len, reverse=True) == ["ccc", "bb", "dd", "a", "e"])
    check("srt3", max(xs, key=lambda x: -x) == 1 and min(words, key=len) == "a" and max(words, key=len) == "ccc"
          and max(1, 2, key=lambda x: -x) == 1)
    check("srt4", min([], default=7) == 7 and max([], default="d") == "d" and max([3, 1], default=None) == 3)
    ys = list(xs)
    ys.sort(key=lambda x: -x)
    ws = list(words)
    ws.sort(key=len, reverse=True)
    d = {"a": 3, "b": 1, "c": 2}
    check("srt5", ys == [5, 4, 3, 2, 1] and ws == ["ccc", "bb", "dd", "a", "e"] and sorted(d, key=d.get) == ["b", "c", "a"]
          and max(d, key=d.get) == "a" and sorted([3, 10, 2], key=str) == [10, 2, 3])
    check("srt6", sorted([[1, 3], [1, 2], [0, 5]]) == [[0, 5], [1, 2], [1, 3]] and [1, 2] < [1, 3] and not [2] < [1, 3]
          and sorted([3, 1, 2, 4], key=lambda x: (x % 2, x)) == [2, 4, 1, 3])
    class P:
        def __init__(self, v):
            self.v = v
        def __lt__(self, o):
            return self.v < o.v
    ps = sorted([P(3), P(1), P(2)])
    check("srt7", [p.v for p in ps] == [1, 2, 3] and min([P(3), P(1), P(2)]).v == 1)
    msgs = []
    for f in [lambda: max([]), lambda: min([]), lambda: max(1, 2, default=3), lambda: max([1], kk=1),
              lambda: sorted([1], [2]), lambda: [].sort(kk=1), lambda: sorted([1, 2], kk=1)]:
        try:
            f()
        except (TypeError, ValueError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("srt8", msgs == ["ValueError: max() iterable argument is empty", "ValueError: min() iterable argument is empty",
                           "TypeError: Cannot specify a default for max() with multiple positional arguments",
                           "TypeError: max() got an unexpected keyword argument 'kk'",
                           "TypeError: sorted expected 1 argument, got 2",
                           "TypeError: sort() got an unexpected keyword argument 'kk'",
                           "TypeError: sort() got an unexpected keyword argument 'kk'"])
    # A keyword close enough to a parameter's name is suggested: CPython's
    # edit distance, a case flip costing half an edit, within a third of the
    # characters involved.
    def kwf(alpha=0, *, value=0, Name=0):
        return alpha
    msgs = []
    for kw in ["alpah", "valeu", "name", "NAME", "zz", "vlue"]:
        try:
            kwf(**{kw: 1})
        except TypeError as e:
            msgs.append(str(e).split("argument ")[1])
    for f in [lambda: sorted([1], keyy=1), lambda: print(1, sepp=""), lambda: int("1", bas=2)]:
        try:
            f()
        except TypeError as e:
            msgs.append(str(e))
    check("srt9", msgs == ["'alpah'. Did you mean 'alpha'?", "'valeu'. Did you mean 'value'?",
                           "'name'. Did you mean 'Name'?", "'NAME'. Did you mean 'Name'?", "'zz'", "'vlue'. Did you mean 'value'?",
                           "sort() got an unexpected keyword argument 'keyy'. Did you mean 'key'?",
                           "print() got an unexpected keyword argument 'sepp'. Did you mean 'sep'?",
                           "int() got an unexpected keyword argument 'bas'. Did you mean 'base'?"])


# ===== SECTION 63: sum() of floats, math.cbrt, int() with a base, conversion keywords =====
# sum() adds a run of floats with Neumaier's compensated summation (CPython
# 3.12+), so sum([0.1] * 10) is exactly 1.0. int(x, base) parses a str in base
# 2..36 or 0 (the prefix chooses), with underscores between digits, and raises
# CPython's errors. float/bool/list/tuple/set take no keyword arguments.
def s63():
    import math
    tenth = [0.1] * 10
    big = [1e16, 1.0, -1e16]
    check("sum1", sum(tenth) == 1.0 and sum(big) == 1.0 and sum([1e100, 1.0, -1e100]) == 1.0
          and sum([3, 0.1, 0.1, 0.1]) == 3.3000000000000003)
    check("sum2", sum(tenth, 0.5) == 1.5 and sum(tenth, start=1) == 2.0 and sum([], start=2.5) == 2.5
          and sum([1, 2, 3], start=10) == 16 and sum([1e308, 1e308, -1e308]) == float("inf"))
    check("sum3", repr(sum([-0.0], -0.0)) == "-0.0" and repr(sum([0.1, 0.2, 0.3, 0.4], -1)) == "0.0"
          and sum(x / 7 for x in range(50)) == 175.0)
    check("cbrt1", math.cbrt(27) == 3.0 and math.cbrt(-8) == -2.0 and math.cbrt(64) == 4.0
          and repr(math.cbrt(-0.0)) == "-0.0" and math.cbrt(float("inf")) == float("inf"))
    check("int1", [int("10", base=2), int("ff", 16), int("Z", base=36), int("0x1f", base=0), int(" +0b1_01 ", 0),
                   int("0o17", 8), int("1_000"), int("000", 0), int("0x_1F", 16)] == [2, 255, 35, 31, 5, 15, 1000, 0, 31])
    check("int2", int("ffffffffffffffffffff", 16) == 2 ** 80 - 1 and int("-9007199254740993") == -9007199254740993
          and int() == 0 and int("\t12\n") == 12)
    msgs = []
    for f in [lambda: int("010", 0), lambda: int("1__0"), lambda: int("12a"), lambda: int("0x", 16), lambda: int("3.9"),
              lambda: int(""), lambda: int("2", base=2), lambda: int(5, base=2), lambda: int("1", base=37),
              lambda: int("1", base=2.0), lambda: int(base=10), lambda: int(x=3), lambda: int("1", 2, 3),
              lambda: float(x=1), lambda: list(iterable=[1]), lambda: str(5, foo=1), lambda: sum([1], st=2),
              lambda: sum([1], 2, start=3), lambda: complex(1, real=2)]:
        try:
            f()
        except (TypeError, ValueError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("int3", msgs[:6] == ["ValueError: invalid literal for int() with base 0: '010'",
                               "ValueError: invalid literal for int() with base 10: '1__0'",
                               "ValueError: invalid literal for int() with base 10: '12a'",
                               "ValueError: invalid literal for int() with base 16: '0x'",
                               "ValueError: invalid literal for int() with base 10: '3.9'",
                               "ValueError: invalid literal for int() with base 10: ''"])
    check("int4", msgs[6:13] == ["ValueError: invalid literal for int() with base 2: '2'",
                                 "TypeError: int() can't convert non-string with explicit base",
                                 "ValueError: int() base must be >= 2 and <= 36, or 0",
                                 "TypeError: 'float' object cannot be interpreted as an integer",
                                 "TypeError: int() missing string argument",
                                 "TypeError: int() got an unexpected keyword argument 'x'",
                                 "TypeError: int expected at most 2 arguments, got 3"])
    check("int5", msgs[13:] == ["TypeError: float() takes no keyword arguments", "TypeError: list() takes no keyword arguments",
                                "TypeError: str() got an unexpected keyword argument 'foo'",
                                "TypeError: sum() got an unexpected keyword argument 'st'",
                                "TypeError: sum() takes at most 2 arguments (3 given)",
                                "TypeError: argument for complex() given by name ('real') and position (1)"])
    check("int6", str(object=5) == "5" and complex(real=1, imag=2) == 1 + 2j and complex(imag=2) == 2j
          and int("7", base=10) == 7 and str(None) == "None" and str(object=None) == "None" and str(encoding="u") == "")


# ===== SECTION 64: unpacking the wrong number of values is a catchable ValueError =====
# "too many values to unpack (expected N, got M)" - the count only for a list,
# a tuple or a dict - and "not enough values to unpack (expected [at least] N,
# got M)", from assignments, for targets, nested and starred patterns alike.
def s64():
    def g(n):
        yield from range(n)
    def c1():
        a, b = [1, 2, 3]
    def c2():
        a, b = [1]
    def c3():
        a, b = "abc"
    def c4():
        a, b = g(1)
    def c5():
        a, b = {1: 1, 2: 2, 3: 3}
    def c6():
        a, *b, c = [1]
    def c7():
        for a, b in [[1, 2, 3]]:
            pass
    def c8():
        a, (b, c) = [1, [2]]
    def c9():
        a, = []
    def c10():
        a, b = {1, 2, 3}
    msgs = []
    for f in [c1, c2, c3, c4, c5, c6, c7, c8, c9, c10]:
        try:
            f()
        except ValueError as e:
            msgs.append(str(e))
    check("unp1", msgs[:5] == ["too many values to unpack (expected 2, got 3)", "not enough values to unpack (expected 2, got 1)",
                               "too many values to unpack (expected 2)", "not enough values to unpack (expected 2, got 1)",
                               "too many values to unpack (expected 2, got 3)"])
    check("unp2", msgs[5:] == ["not enough values to unpack (expected at least 2, got 1)",
                               "too many values to unpack (expected 2, got 3)", "not enough values to unpack (expected 2, got 1)",
                               "not enough values to unpack (expected 1, got 0)", "too many values to unpack (expected 2)"])
    a, *b, c = [1, 2]
    d, e = "xy"
    check("unp3", a == 1 and b == [] and c == 2 and d + e == "xy")


# ===== SECTION 65: subclasses of builtin types; __iter__, __bool__; bytes.decode =====
# An instance of `class S(str)` (list, dict, int, float alike) behaves as its
# value for every builtin operation its class does not override - methods,
# len, indexing, slicing, operators, iteration, str/repr, conversions - and a
# builtin method answers the plain type. An instance's __iter__/__next__ and
# __bool__/__len__ drive iteration and truthiness. A bytes value decodes as
# UTF-8 (strict), ASCII or Latin-1, with CPython's UnicodeDecodeError texts.
def s65():
    class Name(str):
        def shout(self):
            return self.upper() + "!"
        def __repr__(self):
            return "Name(" + str.__repr__(self) + ")"
    n = Name("bob")
    check("sub1", n.shout() == "BOB!" and str(n) == "bob" and repr(n) == "Name('bob')" and len(n) == 3 and n[0] == "b"
          and n[1:] == "ob" and "o" in n and n.split("o") == ["b", "b"] and n.startswith("b"))
    check("sub2", n == "bob" and not (n != "bob") and n < "c" and "x" + n == "xbob" and n * 2 == "bobbob"
          and isinstance(n, str) and isinstance(n, Name) and hasattr(n, "upper") and type(n.upper()).__name__ == "str")
    class Stack(list):
        def __init__(self, *items):
            super().__init__(items)
            self.pushes = len(items)
        def push(self, x):
            self.append(x)
            self.pushes += 1
    st = Stack(1, 2)
    st.push(3)
    check("sub3", st == [1, 2, 3] and len(st) == 3 and st[-1] == 3 and st.pushes == 3 and [x * 2 for x in st] == [2, 4, 6]
          and sum(st) == 6 and max(st) == 3 and 3 in st and bool(st) and not bool(Stack()) and str(st) == "[1, 2, 3]")
    st.sort(reverse=True)
    st[0] = 7
    del st[1]
    check("sub4", st == [7, 1] and st.pop() == 1 and st + [9] == [7, 9] and type(st).__name__ == "Stack"
          and type(st + [1]).__name__ == "list" and st[0:1] == [7])
    class Counts(dict):
        def total(self):
            return sum(self.values())
    c = Counts(a=1, b=2)
    c["z"] = 3
    check("sub5", c.total() == 6 and c["a"] == 1 and len(c) == 3 and sorted(c) == ["a", "b", "z"] and c.get("q", 0) == 0
          and "a" in c)
    class Meters(int):
        def __str__(self):
            return str(int(self)) + "m"
    m = Meters(5)
    class F(float):
        pass
    f = F(2.5)
    check("sub6", str(m) == "5m" and m + 1 == 6 and 1 + m == 6 and m * 2 == 10 and m > 3 and -m == -5 and int(m) == 5
          and float(m) == 5.0 and m == 5 and f + 1 == 3.5 and int(f) == 2 and repr(f) == "2.5")
    class Up:
        def __init__(self, n):
            self.i = 0
            self.n = n
        def __iter__(self):
            return self
        def __next__(self):
            if self.i >= self.n:
                raise StopIteration
            self.i += 1
            return self.i
    class Gen:
        def __iter__(self):
            yield 1
            yield 2
    class Box:
        def __init__(self, k):
            self.k = k
        def __len__(self):
            return self.k
    class No:
        def __bool__(self):
            return False
    a, b = Up(2)
    got = []
    for x in Up(2):
        got.append(x)
    check("sub7", list(Up(3)) == [1, 2, 3] and sum(Up(4)) == 10 and a + b == 3 and got == [1, 2] and list(Gen()) == [1, 2]
          and sorted(Gen(), reverse=True) == [2, 1] and not Box(0) and Box(2) and not No() and bool(Box(1)))
    check("byt3", b"caf\xc3\xa9".decode() == "café" and b"\xf0\x9f\x98\x80".decode("utf-8") == "\U0001F600"
          and b"\xe9".decode("latin-1") == "é" and b"ab".decode(encoding="ascii") == "ab" and str(b"ab", "utf-8") == "ab")
    msgs = []
    for g in [lambda: b"\xff".decode(), lambda: b"a\xe9b".decode(), lambda: b"\xe2\x82".decode(), lambda: b"\xed\xa0\x80".decode(),
              lambda: b"\xe9".decode("ascii"), lambda: b"a".decode("nope"), lambda: str("ab", "utf-8")]:
        try:
            g()
        except (ValueError, LookupError, TypeError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("byt4", msgs == ["UnicodeDecodeError: 'utf-8' codec can't decode byte 0xff in position 0: invalid start byte",
                           "UnicodeDecodeError: 'utf-8' codec can't decode byte 0xe9 in position 1: invalid continuation byte",
                           "UnicodeDecodeError: 'utf-8' codec can't decode bytes in position 0-1: unexpected end of data",
                           "UnicodeDecodeError: 'utf-8' codec can't decode byte 0xed in position 0: invalid continuation byte",
                           "UnicodeDecodeError: 'ascii' codec can't decode byte 0xe9 in position 0: ordinal not in range(128)",
                           "LookupError: unknown encoding: nope", "TypeError: decoding str is not supported"])


# ===== SECTION 66: ordering across kinds =====
# Only numbers, two strs, two sequences and two sets have an order; any other
# pair raises CPython's TypeError, after a user class's __lt__ or reflected
# __gt__ has had its chance - and a comparison's value is the dunder's own.
def s66():
    def err(f):
        try:
            f()
            return "no error"
        except TypeError as e:
            return str(e)
    vals = [1, "a", 1.5, None, True, {1: 2}, {1}, 2 ** 70]
    check("ord1", err(lambda: sorted([vals[0], vals[1]])) == "'<' not supported between instances of 'str' and 'int'")
    check("ord2", err(lambda: [vals[0]] < [vals[1]]) == "'<' not supported between instances of 'int' and 'str'")
    check("ord3", err(lambda: vals[1] > vals[0]) == "'>' not supported between instances of 'str' and 'int'")
    check("ord4", err(lambda: vals[1] <= vals[2]) == "'<=' not supported between instances of 'str' and 'float'")
    check("ord5", err(lambda: vals[0] >= vals[1]) == "'>=' not supported between instances of 'int' and 'str'")
    check("ord6", err(lambda: vals[3] < vals[0]) == "'<' not supported between instances of 'NoneType' and 'int'")
    check("ord7", err(lambda: vals[5] < vals[5]) == "'<' not supported between instances of 'dict' and 'dict'")
    check("ord8", err(lambda: vals[6] < vals[0]) == "'<' not supported between instances of 'set' and 'int'")
    check("ord9", err(lambda: vals[7] < vals[1]) == "'<' not supported between instances of 'int' and 'str'")
    check("ord10", err(lambda: max(vals[0], vals[1])) == "'>' not supported between instances of 'str' and 'int'")
    check("ord11", err(lambda: min([vals[1], vals[0]])) == "'<' not supported between instances of 'int' and 'str'")
    check("ord12", (vals[4] < vals[2]) and (vals[7] > vals[2]) and not (vals[6] < vals[6]) and [1, "a"] <= [1, "a"])
    class A:
        def __gt__(self, o):
            return "A.gt"
        def __lt__(self, o):
            return "A.lt"
    class B:
        pass
    check("ord13", (vals[0] < A()) == "A.gt" and (A() < vals[0]) == "A.lt" and (vals[0] > A()) == "A.lt")
    check("ord14", err(lambda: B() < vals[0]) == "'<' not supported between instances of 'B' and 'int'")
    check("ord15", (1 < 3 < 2) is False and (1 < 2 < 3) is True)

# ===== SECTION 67: str keywords, str.encode, no-argument constructors =====
# split/rsplit/splitlines/expandtabs/encode/replace take keywords under CPython
# 3.14's names and every other str method refuses them; encode() answers the bytes
# (a list of byte values here) with the codec's UnicodeEncodeError; float(), list()
# and friends have a value with no argument; int() and float() read any Unicode
# decimal digit.
def s67():
    def err(f):
        try:
            f()
            return "no error"
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
    s = ["bob", "a,b,,c", "  x y  ", "é", "Hello", "abcabc", "a\tb", "a\nb\n", "€😀"]
    check("skw1", s[0].split(sep="o") == ["b", "b"] and s[1].split(sep=",", maxsplit=1) == ["a", "b,,c"])
    check("skw2", s[1].rsplit(sep=",", maxsplit=1) == ["a,b,", "c"] and s[2].split(maxsplit=1) == ["x", "y  "])
    check("skw3", s[5].replace("a", "X", count=1) == "Xbcabc" and s[6].expandtabs(tabsize=4) == "a   b")
    check("skw4", s[7].splitlines(keepends=True) == ["a\n", "b\n"])
    check("skw5", err(lambda: s[2].strip(chars=" x")) == "TypeError: str.strip() takes no keyword arguments")
    check("skw6", err(lambda: s[4].endswith(suffix="lo")) == "TypeError: str.endswith() takes no keyword arguments")
    check("skw7", err(lambda: s[4].ljust(width=9)) == "TypeError: str.ljust() takes no keyword arguments")
    check("skw8", err(lambda: s[0].split(foo=1)) == "TypeError: split() got an unexpected keyword argument 'foo'")
    check("skw9", err(lambda: s[0].split(",", sep=",")) == "TypeError: argument for split() given by name ('sep') and position (1)")
    check("skw10", err(lambda: s[0].replace(old="a", new="b")) == "TypeError: replace() takes at least 2 positional arguments (0 given)")
    check("enc1", len(s[3].encode()) == 2 and s[3].encode().decode() == "é" and list(s[3].encode("utf-8")) == [195, 169])
    check("enc2", list(s[3].encode(encoding="latin-1")) == [233] and list(s[8].encode()) == [226, 130, 172, 240, 159, 152, 128])
    check("enc3", err(lambda: s[3].encode("ascii")) == "UnicodeEncodeError: 'ascii' codec can't encode character '\\xe9' in position 0: ordinal not in range(128)")
    check("enc4", err(lambda: ("a" + s[8]).encode("latin-1")) == "UnicodeEncodeError: 'latin-1' codec can't encode characters in position 1-2: ordinal not in range(256)")
    check("enc5", list(s[3].encode("ascii", errors="replace")) == [63] and list(s[3].encode("ascii", "ignore")) == [])
    check("enc6", s[8].encode("ascii", "backslashreplace").decode() == "\\u20ac\\U0001f600")
    check("enc7", s[3].encode("ascii", "xmlcharrefreplace").decode() == "&#233;")
    check("enc8", err(lambda: s[3].encode("klingon")) == "LookupError: unknown encoding: klingon")
    check("ctor1", repr(float()) == "0.0" and int() == 0 and str() == "" and list() == [] and len(tuple()) == 0)
    check("ctor2", dict() == {} and repr(set()) == "set()" and bool() is False and complex() == 0)
    check("ctor3", err(lambda: list(None)) == "TypeError: 'NoneType' object is not iterable")
    nd = ["٣", "١٢٣", "٣.٥", "３", " ٤٢ ", "𝟗"]
    check("ctor4", int(nd[0]) == 3 and int(nd[1]) + 1 == 124 and float(nd[2]) == 3.5 and int(nd[3]) == 3)
    check("ctor5", int(nd[4]) == 42 and int(nd[5]) == 9 and int(nd[1], 16) == 291)

# ===== SECTION 68: str method argument errors, builtin-subclass ordering =====
# A str method's wrong argument type, wrong arity, empty separator or missing
# substring is a CATCHABLE exception with CPython's own text; a str subclass is a
# str argument. Ordering a builtin subclass with another kind names the
# subclass, in the operator the program wrote.
def s68():
    def err(f):
        try:
            f()
            return "no error"
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
    class S(str): pass
    class I(int): pass
    v = ["ab", 1, None, 1.5, "", "xy"]
    s = v[0]
    check("sarg1", err(lambda: s.startswith(v[1])) == "TypeError: startswith first arg must be str or a tuple of str, not int")
    check("sarg2", err(lambda: s.endswith((v[1],))) == "TypeError: tuple for endswith must only contain str, not int")
    check("sarg3", err(lambda: s.rstrip(v[1])) == "TypeError: rstrip arg must be None or str")
    check("sarg4", err(lambda: s.split(v[1])) == "TypeError: must be str or None, not int" and err(lambda: s.rsplit(v[4])) == "ValueError: empty separator")
    check("sarg5", err(lambda: s.join(["a", v[1]])) == "TypeError: sequence item 1: expected str instance, int found")
    check("sarg6", err(lambda: s.replace("a", v[1])) == "TypeError: replace() argument 2 must be str, not int")
    check("sarg7", err(lambda: s.rfind(v[2])) == "TypeError: rfind() argument 1 must be str, not None" and err(lambda: s.index("z")) == "ValueError: substring not found")
    check("sarg8", err(lambda: s.center(5, v[1])) == "TypeError: The fill character must be a unicode character, not int" and err(lambda: s.ljust(5, v[5])) == "TypeError: The fill character must be exactly one character long")
    check("sarg9", err(lambda: s.partition(v[4])) == "ValueError: empty separator" and err(lambda: s.rpartition(v[1])) == "TypeError: must be str, not int")
    check("sarg10", err(lambda: s.removesuffix(v[1])) == "TypeError: removesuffix() argument must be str, not int")
    check("sarg11", err(lambda: s.zfill(v[0])) == "TypeError: 'str' object cannot be interpreted as an integer" and err(lambda: s.center(v[3])) == "TypeError: 'float' object cannot be interpreted as an integer")
    check("sarg12", err(lambda: s.find("a", v[0])) == "TypeError: slice indices must be integers or None or have an __index__ method" and s.find("b", v[2]) == 1)
    check("sarg13", err(lambda: s.split(None, v[2])) == "TypeError: 'NoneType' object cannot be interpreted as an integer")
    check("sarg14", err(lambda: s.upper(v[1])) == "TypeError: str.upper() takes no arguments (1 given)" and err(lambda: s.join()) == "TypeError: str.join() takes exactly one argument (0 given)")
    check("sarg15", err(lambda: s.count()) == "TypeError: count expected at least 1 argument, got 0" and err(lambda: s.strip("a", "b")) == "TypeError: strip expected at most 1 argument, got 2")
    check("sarg16", err(lambda: s.split(",", 1, 2)) == "TypeError: split() takes at most 2 arguments (3 given)" and err(lambda: s.replace("a")) == "TypeError: replace() takes at least 2 positional arguments (1 given)")
    check("sarg17", err(lambda: "{0}".format()) == "IndexError: Replacement index 0 out of range for positional args tuple")
    check("sarg18", s.startswith(S("a")) and "-".join([S("a"), S("b")]) == "a-b" and s.replace(S("a"), S("z")) == "zb")
    check("sarg19", s.center(I(4), S("*")) == "*ab*" and "abc".find("c", I(1)) == 2 and s.find("b", True) == 1)
    check("sord1", err(lambda: S("b") < v[1]) == "TypeError: '<' not supported between instances of 'S' and 'int'")
    check("sord2", err(lambda: v[1] < S("b")) == "TypeError: '<' not supported between instances of 'int' and 'S'")
    check("sord3", err(lambda: v[1] >= S("b")) == "TypeError: '>=' not supported between instances of 'int' and 'S'")
    check("sord4", err(lambda: I(1) < v[0]) == "TypeError: '<' not supported between instances of 'I' and 'str'")
    check("sord5", S("a") < S("b") and S("a") < "b" and I(3) < 4 and "a" < S("b") and 2 < I(3))

# ===== SECTION 69: bytes(...) and frozenset =====
# bytes(...) builds the bytes value (a list of byte values here) from a count, a
# str under an encoding or an iterable of ints in range(256). frozenset is a
# hashable set without the mutating methods; its algebra answers a frozenset
# when it is the left operand or the receiver.
def s69():
    def err(f):
        try:
            f()
            return "no error"
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
    v = [3, "hé", [104, 105], None, -1, 1.5, [256], ["a"], 0]
    check("byn1", list(bytes(v[0])) == [0, 0, 0] and list(bytes(v[1], "utf-8")) == [104, 195, 169] and list(bytes(v[2])) == [104, 105])
    check("byn2", list(bytes()) == [] and list(bytes(v[8])) == [] and list(bytes(True)) == [0] and list(bytes(range(3))) == [0, 1, 2])
    check("byn3", list(bytes(v[1], encoding="latin-1")) == [104, 233] and list(bytes(v[1], "ascii", "replace")) == [104, 63])
    check("byn4", err(lambda: bytes(v[1])) == "TypeError: string argument without an encoding" and err(lambda: bytes(v[0], "utf-8")) == "TypeError: encoding without a string argument")
    check("byn5", err(lambda: bytes(v[3])) == "TypeError: cannot convert 'NoneType' object to bytes" and err(lambda: bytes(v[5])) == "TypeError: cannot convert 'float' object to bytes")
    check("byn6", err(lambda: bytes(v[4])) == "ValueError: negative count" and err(lambda: bytes(v[6])) == "ValueError: bytes must be in range(0, 256)")
    check("byn7", err(lambda: bytes(v[7])) == "TypeError: 'str' object cannot be interpreted as an integer" and err(lambda: bytes(v[1], "ascii")).startswith("UnicodeEncodeError: 'ascii' codec"))
    check("byn8", isinstance(b"x", bytes) == isinstance(bytes(v[2]), bytes) and bytes(v[2]).decode() == "hi")
    a = frozenset([1, 2, 2])
    check("fzs1", repr(a) == "frozenset({1, 2})" and repr(frozenset()) == "frozenset()" and str(frozenset("aa")) == "frozenset({'a'})" and len(a) == 2)
    check("fzs2", type(a).__name__ == "frozenset" and isinstance(a, frozenset) and not isinstance(a, set) and not isinstance({1}, frozenset))
    check("fzs3", a == {1, 2} and {2, 1} == a and 1 in a and 5 not in a and a < {1, 2, 3} and {1} < a)
    check("fzs4", type(a | {3}).__name__ == "frozenset" and type({3} | a).__name__ == "set" and a & {2} == {2} and a - {1} == {2})
    check("fzs5", type(a.union([3])).__name__ == "frozenset" and type(a.copy()).__name__ == "frozenset" and a.symmetric_difference([1, 5]) == {2, 5})
    d = {a: "fs"}
    check("fzs6", d[frozenset([2, 1])] == "fs" and d.get(frozenset([1])) is None and frozenset([1, 2]) in d)
    s = {a, frozenset([2, 1]), frozenset([1])}
    check("fzs7", len(s) == 2 and frozenset([1]) in s and {1, 2} in s and repr(frozenset([frozenset([1])])) == "frozenset({frozenset({1})})")
    check("fzs8", err(lambda: a.add(3)) == "AttributeError: 'frozenset' object has no attribute 'add'" and err(lambda: a.discard(1)) == "AttributeError: 'frozenset' object has no attribute 'discard'")
    check("fzs9", err(lambda: a.update([1])) == "AttributeError: 'frozenset' object has no attribute 'update'" and not hasattr(a, "pop") and hasattr(a, "union"))
    check("fzs10", err(lambda: {a: 1}[{1, 2}]) == "TypeError: cannot use 'set' as a dict key (unhashable type: 'set')" and err(lambda: frozenset(1)) == "TypeError: 'int' object is not iterable")
    b = a
    b |= {9}
    check("fzs11", a == {1, 2} and b == {1, 2, 9} and type(b).__name__ == "frozenset")

# ===== SECTION 70: the attributes every number carries =====
# An int (a bool and a big int too) is its own .real and .numerator as an int,
# with .imag 0 and .denominator 1; a float is its own .real with .imag 0.0 and no
# numerator; conjugate() of a real number is its .real.
def s70():
    v = [7, 2.5, True, 10**30, 3+4j, -0.0]
    check("nat1", [repr(x.real) for x in v[:4]] == ["7", "2.5", "1", "1000000000000000000000000000000"] and repr(v[1].imag) == "0.0" and v[0].imag == 0)
    check("nat2", v[2].numerator == 1 and type(v[2].real).__name__ == "int" and v[3].denominator == 1 and v[0].numerator == 7)
    check("nat3", [repr(x.conjugate()) for x in v] == ["7", "2.5", "1", "1000000000000000000000000000000", "(3-4j)", "-0.0"])
    check("nat4", "{0.real}|{0.imag}".format(v[0]) == "7|0" and f"{v[1].imag}" == "0.0" and getattr(v[2], "denominator") == 1)
    check("nat5", hasattr(v[0], "numerator") and not hasattr(v[1], "numerator") and hasattr(v[1], "conjugate") and getattr(v[1], "conjugate")() == 2.5)
    try:
        msg = str(v[1].numerator)
    except AttributeError as e:
        msg = str(e)
    check("nat6", msg == "'float' object has no attribute 'numerator'")

# ===== SECTION 71: int's and float's own methods =====
# bit_length / bit_count / to_bytes / from_bytes on an int (a bool and a big int
# too), hex / fromhex / is_integer / as_integer_ratio on a float, and the same
# methods read off the TYPE (int.conjugate(7), float.hex(2.0)). Rows from CPython
# 3.14.
def s71():
    v = [7, -5, True, 10**30, 255]
    check("nim1", [x.bit_length() for x in v] == [3, 3, 1, 100, 8] and [x.bit_count() for x in v] == [3, 2, 1, 37, 8])
    check("nim2", list(v[4].to_bytes(2, "little")) == [255, 0] and list(v[1].to_bytes(2, "big", signed=True)) == [255, 251] and list(v[0].to_bytes()) == [7])
    check("nim3", int.from_bytes([255, 251], "big", signed=True) == -5 and int.from_bytes([1, 2]) == 258 and int.from_bytes((-v[3]).to_bytes(13, "big", signed=True), "big", signed=True) == -v[3])
    msgs = []
    for f in [lambda: v[4].to_bytes(1, signed=True), lambda: v[1].to_bytes(1), lambda: v[0].to_bytes(1, "middle")]:
        try:
            f()
        except (OverflowError, ValueError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("nim4", msgs == ["OverflowError: int too big to convert", "OverflowError: can't convert negative int to unsigned",
                           "ValueError: byteorder must be either 'little' or 'big'"])
    w = [1.5, -0.25, 3.0, 0.1, 5e-324, -0.0]
    check("nim5", [x.hex() for x in w] == ["0x1.8000000000000p+0", "-0x1.0000000000000p-2", "0x1.8000000000000p+1",
                                           "0x1.999999999999ap-4", "0x0.0000000000001p-1022", "-0x0.0p+0"])
    check("nim6", [float.fromhex(x.hex()) == x for x in w] == [True] * 6 and float.fromhex(" -0x1.8p1 ") == -3.0 and float.fromhex("0x1.00000000000018p0") == 1.0000000000000004)
    check("nim7", list(w[3].as_integer_ratio()) == [3602879701896397, 36028797018963968] and list(w[1].as_integer_ratio()) == [-1, 4] and [x.is_integer() for x in w[:3]] == [False, False, True])
    check("nim8", hasattr(int, "conjugate") and int.conjugate(7) == 7 and float.hex(2.0) == "0x1.0000000000000p+1" and int.bit_length(255) == 8 and hasattr(3, "bit_length") and not hasattr(2.5, "bit_length"))
    try:
        r = float.fromhex("0x1p1024")
    except OverflowError as e:
        r = str(e)
    check("nim9", r == "hexadecimal value too large to represent as a float" and float.fromhex("0x1p-1075") == 0.0 and float.fromhex("0x1.8p-1075") == 5e-324)

# ===== SECTION 72: dunders called on a number =====
# (5).__add__(3) is int's own __add__ with 5 as self; the conversion and rounding
# dunders answer what the builtin they serve answers; any other name on a number
# is a catchable AttributeError. The type object's own methods print as bound
# built-in methods, and float.from_number takes a real number only. Rows from
# CPython 3.14.
def s72():
    v = [5, 2.5, True, -0.0, 2**70, -7]
    check("ndu1", v[0].__add__(3) == 8 and v[0].__radd__(1) == 6 and v[1].__mul__(2) == 5.0 and v[2].__add__(1) == 2 and v[5].__floordiv__(2) == -4 and v[5].__rmod__(10) == -4)
    check("ndu2", [v[0].__str__(), v[1].__repr__(), v[2].__str__(), v[3].__str__(), v[4].__str__()] == ["5", "2.5", "True", "-0.0", "1180591620717411303424"])
    check("ndu3", [x.__int__() for x in v[:3]] == [5, 2, 1] and repr(v[0].__float__()) == "5.0" and v[2].__index__() == 1 and repr(v[2].__pos__()) == "1" and v[5].__invert__() == 6 and v[4].__invert__() == -(2**70) - 1)
    check("ndu4", repr(v[3].__neg__()) == "0.0" and repr(v[3].__abs__()) == "0.0" and repr(abs(v[3])) == "0.0" and repr(abs(v[2])) == "1" and v[5].__abs__() == 7 and v[0].__lt__(7) and v[0].__eq__(5) and v[0].__bool__())
    w = [-2.5, 2.5, 7.0, 2.567]
    check("ndu5", [x.__floor__() for x in w[:3]] == [-3, 2, 7] and [x.__ceil__() for x in w[:3]] == [-2, 3, 7] and [x.__trunc__() for x in w[:2]] == [-2, 2] and v[0].__floor__() == 5)
    check("ndu6", w[1].__round__() == 2 and repr(w[3].__round__(2)) == "2.57" and repr(v[0].__round__(1)) == "5" and repr(round(v[2], 1)) == "1" and round(v[4]) == 2**70 and list(v[5].__divmod__(2)) == [-4, 1] and list(v[0].__rdivmod__(17)) == [3, 2] and v[0].__format__("03d") == "005")
    msgs = []
    for f in [lambda: v[0].nosuch(), lambda: v[1].__invert__(), lambda: v[2].foo, lambda: v[1].__index__(), lambda: (1 + 2j).foo()]:
        try:
            f()
        except AttributeError as e:
            msgs.append(str(e))
    check("ndu7", msgs == ["'int' object has no attribute 'nosuch'", "'float' object has no attribute '__invert__'", "'bool' object has no attribute 'foo'",
                           "'float' object has no attribute '__index__'", "'complex' object has no attribute 'foo'"])
    m = getattr(v[0], "__mul__")
    check("ndu8", m(7) == 35 and hasattr(v[0], "__add__") and hasattr(v[2], "__index__") and not hasattr(v[1], "__invert__") and not hasattr(v[0], "nosuch"))
    class I(int):
        pass
    check("ndu9", I(5).__add__(1) == 6 and I(5).__str__() == "5" and abs(I(-3)) == 3 and I(4).__floor__() == 4)
    check("ndu10", [repr(float.from_number(x)) for x in [3, 2.5, True, I(4)]] == ["3.0", "2.5", "1.0", "4.0"] and float.from_number(2**70) == 1.1805916207174113e+21)
    msgs = []
    for x in ["1.5", None, 1j]:
        try:
            float.from_number(x)
        except TypeError as e:
            msgs.append(str(e))
    check("ndu11", msgs == ["must be real number, not str", "must be real number, not NoneType", "must be real number, not complex"])
    check("ndu12", [repr(f).split(" at ")[0].rstrip(">") for f in [int.from_bytes, float.fromhex, dict.fromkeys, float.from_number]] ==
          ["<built-in method from_bytes of type object", "<built-in method fromhex of type object", "<built-in method fromkeys of type object", "<built-in method from_number of type object"]
          and float.from_number.__name__ == "from_number" and int.from_bytes.__self__ is int)

# ===== SECTION 73: a lone surrogate is one code point =====
# A high surrogate pairs only with a low one after it; before anything else it
# is a character of its own, for len, indexing, ord, %c and a precision or
# width. (A lone high and a lone low that end up ADJACENT are one pair in this
# UTF-16 value model where CPython keeps two, so no row builds that.)
def s73():
    v = ["a\ud83db", "\ude00", "\ud83d\ud83d!", "\U0001F600x"]
    t = v[0] + v[1]
    check("lsur1", len(t) == 4 and ord(t[1]) == 55357 and ord(t[2]) == 98 and t[-1] == "\ude00" and len(v[2]) == 3)
    check("lsur2", [ord(c) for c in t[1:3]] == [55357, 98] and t.find("b") == 2 and len(v[3]) == 2)
    check("lsur3", [ord(c) for c in "%3s|%.1s|%c" % (v[2][0], v[0][1:], v[2][1])] == [32, 32, 55357, 124, 55357, 124, 55357])
    check("lsur4", [ord(c) for c in "{:>3}|{:.1}|{:.1}".format(v[2][0], v[2], v[3])] == [32, 32, 55357, 124, 55357, 124, 128512])

# ===== SECTION 74: a float power is correctly rounded =====
# x ** y and math.pow are C's pow, which answers the double nearest the exact
# power: a large y*log(x) must not cost the result its last digits. 25 ** 11.5
# is 5 ** 23, a midpoint between two doubles, and rounds to the even one.
def s74():
    import math
    xs = [1.9687173381259235, 2.0, 1.0000005275492378, 0.9303244752705773, 25.0]
    ys = [596.2128067674018, -0.5, -48986194.85211566, 5961.696604822106, 11.5]
    check("pow1", xs[0] ** ys[0] == 2.488229202381568e+175 and math.pow(xs[0], ys[0]) == 2.488229202381568e+175)
    check("pow2", xs[1] ** ys[1] == 0.7071067811865476 and math.pow(xs[1], ys[1]) == 0.7071067811865476)
    check("pow3", xs[2] ** ys[2] == 5.979866080580326e-12 and xs[3] ** ys[3] == 1.018795851248988e-187)
    check("pow4", xs[4] ** ys[4] == 11920928955078124.0)

# ===== SECTION 75: close() raises a catchable GeneratorExit =====
# g.close() is throw(GeneratorExit()) at the parked yield. GeneratorExit derives
# from BaseException, so `except GeneratorExit` and a bare `except` catch it and
# `except Exception` does not; a body that returns makes close() answer the value,
# one that yields again is a RuntimeError, and at a `yield from` the delegate is
# closed and the GeneratorExit raised in the outer generator. Values read off
# CPython 3.14.
def s75():
    log = []
    def g1():
        try:
            yield 1
        except GeneratorExit:
            log.append("ge")
            raise
        finally:
            log.append("fin")
    def g2():
        try:
            yield 1
        except Exception:
            log.append("WRONG")
    def g3():
        try:
            yield 1
        except:
            log.append("bare")
    def g4():
        try:
            yield 1
        except GeneratorExit:
            return 5
    def g5():
        try:
            yield 1
        except GeneratorExit:
            pass
        yield 2
        yield 3
    def sw():
        try:
            yield 1
        except GeneratorExit:
            log.append("sw")
        return 7
    def outer():
        r = yield from sw()
        log.append("resumed " + str(r))
        yield 9
    rs = []
    for g in [g1, g2, g3, g4]:
        it = g()
        next(it)
        rs.append(it.close())
    check("gex1", log == ["ge", "fin", "bare"] and rs == [None, None, None, 5])
    it = g5()
    next(it)
    msg = ""
    try:
        it.close()
    except RuntimeError as e:
        msg = str(e)
    check("gex2", msg == "generator ignored GeneratorExit" and next(it) == 3)
    it = outer()
    next(it)
    check("gex3", it.close() is None and log[-1] == "sw")
    check("gex4", issubclass(GeneratorExit, BaseException) and not issubclass(GeneratorExit, Exception)
          and isinstance(GeneratorExit(), BaseException) and repr(GeneratorExit()) == "GeneratorExit()")

# ===== SECTION 76: NotImplemented and the operator fallbacks =====
# A builtin type's binary dunder answers NotImplemented for an operand it does
# not serve (int.__add__(5, 2.5)); the operator then asks the other operand, a
# builtin subclass falls back to its value, == to identity, and anything else is
# CPython's TypeError named by the operands' own types. NotImplemented and
# Ellipsis are named singletons. Rows from CPython 3.14.
class NiA:
    def __init__(self, v):
        self.v = v
    def __add__(self, o):
        if isinstance(o, NiA):
            return NiA(self.v + o.v)
        return NotImplemented
    def __eq__(self, o):
        if isinstance(o, NiA):
            return self.v == o.v
        return NotImplemented
class NiB:
    def __radd__(self, o):
        return "B.radd"
    def __eq__(self, o):
        return "B.eq"
class NiS(int):
    pass
def s76():
    v = [5, 2.5, True, "s", None, 7.5, 5.0]
    check("ni1", [repr(NotImplemented), str(NotImplemented), type(NotImplemented).__name__, repr(Ellipsis), type(...).__name__, str([...])] ==
          ["NotImplemented", "NotImplemented", "NotImplementedType", "Ellipsis", "ellipsis", "[Ellipsis]"] and Ellipsis is ...)
    check("ni2", int.__add__(v[0], v[1]) is NotImplemented and int.__eq__(v[0], v[6]) is NotImplemented and int.__lt__(v[0], v[5]) is NotImplemented
          and int.__add__(v[0], v[2]) == 6 and float.__add__(v[1], v[0]) == 7.5 and v[0].__radd__(v[1]) is NotImplemented and str.__lt__(v[3], v[0]) is NotImplemented)
    msgs = []
    try:
        bool(NotImplemented)
    except TypeError as e:
        msgs.append(str(e))
    a = NiA(1)
    for f in [lambda: a + v[0], lambda: v[0] - a, lambda: a * v[3], lambda: a ** v[0], lambda: a < v[0]]:
        try:
            f()
        except TypeError as e:
            msgs.append(str(e))
    check("ni3", msgs == ["NotImplemented should not be used in a boolean context", "unsupported operand type(s) for +: 'NiA' and 'int'",
                          "unsupported operand type(s) for -: 'int' and 'NiA'", "can't multiply sequence by non-int of type 'NiA'",
                          "unsupported operand type(s) for ** or pow(): 'NiA' and 'int'", "'<' not supported between instances of 'NiA' and 'int'"])
    check("ni4", (a + NiA(2)).v == 3 and a + NiB() == "B.radd" and a == NiA(1) and not (a == v[0]) and a != v[0] and a == a and (a == NiB()) == "B.eq")
    s = NiS(3)
    check("ni5", [s + v[1], v[1] + s, s < v[1], v[1] > s, s == 3.0, s + v[2], s << s, s | 4] == [5.5, 5.5, False, False, True, 4, 24, 7]
          and type(s + v[1]).__name__ == "float")
    check("ni6", repr(v[0].__add__).split(" at ")[0].rstrip(">") == "<method-wrapper '__add__' of int object" and repr(v[1].__round__).split(" at ")[0].rstrip(">") == "<built-in method __round__ of float object"
          and type(v[0].__add__).__name__ == "method-wrapper")

# ===== SECTION 77: round() to tens, shift counts, unary plus =====
# round(x, -n) rounds to a multiple of 10**n, half to even on the exact value;
# round(float) is the EXACT int of the rounded double. A negative shift count is
# a catchable ValueError, a shift past the cap is 0 or -1 for >> and an
# OverflowError for <<. Unary + makes a bool an int. Rows from CPython 3.14.
def s77():
    n = [1250, 1350, -1250, 5, -15, 2**70 + 5 * 10**5, 1250.4, -5.0, 1e300, -2]
    check("rt1", [round(n[0], n[9]), round(n[1], n[9]), round(n[2], n[9]), round(n[3], -1), round(n[4], -1), round(n[5], -6)] == [1200, 1400, -1200, 0, -20, 1180591620717412000000])
    check("rt2", [repr(round(n[6], n[9])), repr(round(n[7], n[9])), repr(round(float(n[1]), n[9]))] == ["1300.0", "-0.0", "1400.0"]
          and round(n[8]) == 10**300 + 52504760255204420248704468581108159154915854115511802457988908195786371375080447864043704443832883878176942523235360430575644792184786706982848387200926575803737830233794788090059368953234970799945081119038967640880074652742780142494579258788820056842838115669472196386865459400540160
          and type(round(n[8])).__name__ == "int")
    msgs = []
    for f in [lambda: n[3] << -7, lambda: n[3] >> n[9], lambda: n[3] << 10**20, lambda: +"s", lambda: -"s", lambda: ~n[6]]:
        try:
            f()
        except (ValueError, OverflowError, TypeError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("rt3", msgs == ["ValueError: negative shift count", "ValueError: negative shift count", "OverflowError: too many digits in integer",
                          "TypeError: bad operand type for unary +: 'str'", "TypeError: bad operand type for unary -: 'str'", "TypeError: bad operand type for unary ~: 'float'"])
    t = [True, 5, -5]
    check("rt4", [repr(+t[0]), repr(-t[0]), repr(+t[1])] == ["1", "-1", "5"] and t[1] >> 10**20 == 0 and t[2] >> 10**20 == -1 and 0 << 10**20 == 0)

# ===== SECTION 78: operand types, repetition counts, builtin dunders =====
# An arithmetic or bit operator on a pair its builtin types do not serve is a
# TypeError naming both types; a repetition count is any int, outside a C
# ssize_t an OverflowError and past what memory can hold a MemoryError. dict |
# dict merges, and |= updates a dict or a set in place. The builtin types' own
# dunders read off a value and off the type. Rows from CPython 3.14.
def s78():
    v = [4, 2.5, "s", [1], None, {1: 2}, {1}, True, 3]
    msgs = []
    for f in [lambda: v[0] - v[2], lambda: v[1] * v[2], lambda: v[1] << v[7], lambda: v[0] - v[3], lambda: v[4] // v[0],
              lambda: v[5] - v[6], lambda: v[2] ** v[0], lambda: v[3] % v[0], lambda: v[5] | v[0], lambda: v[3] * v[1]]:
        try:
            f()
        except TypeError as e:
            msgs.append(str(e))
    check("opty1", msgs == ["unsupported operand type(s) for -: 'int' and 'str'", "can't multiply sequence by non-int of type 'float'",
                            "unsupported operand type(s) for <<: 'float' and 'bool'", "unsupported operand type(s) for -: 'int' and 'list'",
                            "unsupported operand type(s) for //: 'NoneType' and 'int'", "unsupported operand type(s) for -: 'dict' and 'set'",
                            "unsupported operand type(s) for ** or pow(): 'str' and 'int'", "unsupported operand type(s) for %: 'list' and 'int'",
                            "unsupported operand type(s) for |: 'dict' and 'int'", "can't multiply sequence by non-int of type 'float'"])
    big = [10**20, 2**63 - 1, 2**60, -2**63]
    msgs = []
    for f in [lambda: v[2] * big[0], lambda: big[0] * v[3], lambda: "ab" * big[1], lambda: v[2] * big[2], lambda: v[3] * big[1], lambda: -big[0] * ""]:
        try:
            f()
        except (OverflowError, MemoryError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("opty2", msgs == ["OverflowError: cannot fit 'int' into an index-sized integer", "OverflowError: cannot fit 'int' into an index-sized integer",
                            "OverflowError: repeated string is too long", "MemoryError: ", "MemoryError: ", "OverflowError: cannot fit 'int' into an index-sized integer"])
    check("opty3", "" * big[1] == "" and [] * big[2] == [] and v[2] * big[3] == "" and v[2] * v[7] == "s" and v[8] * v[2] == "sss")
    d = {1: 2}
    e = d
    d |= [(5, 6)]
    s = {1, 2}
    t = s
    s |= {3}
    s -= {1}
    s &= {2, 3, 9}
    s ^= {7}
    check("opty4", v[5] | {3: 4, 1: 5} == {1: 5, 3: 4} and list(v[5] | {3: 4}) == [1, 3] and e is d and e == {1: 2, 5: 6} and t is s and sorted(t) == [2, 3, 7])
    check("opty5", hasattr(int, "__add__") and hasattr(str, "__add__") and hasattr(list, "__len__") and hasattr(dict, "__getitem__") and not hasattr(str, "__sub__") and not hasattr(int, "__nope__"))
    check("opty6", v[2].__add__("t") == "st" and v[3].__len__() == 1 and v[5].__len__() == 1 and "ab".__mul__(2) == "abab" and [1].__add__([2]) == [1, 2]
          and v[5].__getitem__(1) == 2 and "abc".__contains__("b") and getattr("x", "__add__")("y") == "xy" and hasattr(v[3], "__len__") and not hasattr(v[2], "__sub__"))

# ===== SECTION 79: the tuple is its own type =====
# A tuple prints parenthesized (a one-element one with its comma), is type
# tuple, never equals or orders against a list, concatenates only with a
# tuple, is immutable, and is hashable when its elements are. dict.items(),
# enumerate, zip, divmod, *args, exception args and the re results are
# tuples; a starred target is a list. "%" takes its arguments from a tuple
# only. Rows from CPython 3.14.
def s79():
    v = [(1, 2), (5,), (), [1, 2], {"a": 1}]
    t = v[0]
    check("tup1", [repr(x) for x in v[:4]] == ["(1, 2)", "(5,)", "()", "[1, 2]"] and str(((1, 2), [3, (4,)])) == "((1, 2), [3, (4,)])")
    check("tup2", type(t).__name__ == "tuple" and isinstance(t, tuple) and not isinstance(t, list) and not isinstance(v[3], tuple) and type(v[2]) is tuple)
    check("tup3", t != v[3] and not (t == v[3]) and [t] != [v[3]] and t == (1, 2) and v[3] not in [t] and list(t) == v[3] and tuple(v[3]) == t)
    check("tup4", repr(t + (3,)) == "(1, 2, 3)" and repr(t * 2) == "(1, 2, 1, 2)" and repr(t[1:]) == "(2,)" and repr(v[1] * 0) == "()" and repr(v[3][1:]) == "[2]")
    msgs = []
    def setit():
        t[0] = 9
    def delit():
        del t[0]
    for f in [lambda: t < v[3], lambda: v[3] + t, lambda: t + v[3], lambda: t.append(3), setit, delit, lambda: t.index(9)]:
        try:
            f()
        except (TypeError, AttributeError, ValueError) as e:
            msgs.append(str(e))
    check("tup5", msgs == ["'<' not supported between instances of 'tuple' and 'list'", 'can only concatenate list (not "tuple") to list',
                           'can only concatenate tuple (not "list") to tuple', "'tuple' object has no attribute 'append'",
                           "'tuple' object does not support item assignment", "'tuple' object doesn't support item deletion",
                           "tuple.index(x): x not in tuple"])
    a = (1, 2)
    b = a
    a += (3,)
    c = [1]
    e = c
    c += (2,)
    p, *q = (1, 2, 3)
    check("tup6", repr(a) == "(1, 2, 3)" and repr(b) == "(1, 2)" and e is c and c == [1, 2] and repr(q) == "[2, 3]")
    def f(*args):
        return args
    def g():
        return 1, 2
    check("tup7", repr(f()) == "()" and repr(f(1)) == "(1,)" and repr(g()) == "(1, 2)" and repr(divmod(7, 2)) == "(3, 1)")
    check("tup8", str(list(enumerate("ab"))) == "[(0, 'a'), (1, 'b')]" and str(list(zip([1, 2], "ab"))) == "[(1, 'a'), (2, 'b')]"
          and str(list(v[4].items())) == "[('a', 1)]" and repr(v[4].copy().popitem()) == "('a', 1)" and repr("a b".partition(" ")) == "('a', ' ', 'b')")
    err = ValueError("boom")
    err2 = Exception(1, 2)
    check("tup9", repr(err.args) == "('boom',)" and str(err2) == "(1, 2)" and repr(err2.args) == "(1, 2)" and repr(err2) == "Exception(1, 2)")
    import re
    m = re.match(r"(a)(b)?", "ac")
    check("tup10", repr(m.groups()) == "('a', None)" and repr(m.span(1)) == "(0, 1)" and repr(re.findall(r"(\w)(\d)", "a1 b2")) == "[('a', '1'), ('b', '2')]"
          and repr(re.subn("a", "b", "aa")) == "('bb', 2)")
    d = {(1, 2): "a"}
    d[3, 4] = "b"
    check("tup11", repr(d) == "{(1, 2): 'a', (3, 4): 'b'}" and d[1, 2] == "a" and len({(1, 2), (1, 2)}) == 1)
    check("tup12", "%d %d" % t == "1 2" and "%s" % v[3] == "[1, 2]" and "%s" % (t,) == "(1, 2)")
    msgs = []
    for f2 in [lambda: "%d %d" % (1, 2, 3), lambda: "%d %d" % (1,), lambda: {v[3]: 1}, lambda: {(1, v[3])}]:
        try:
            f2()
        except TypeError as e3:
            msgs.append(str(e3))
    check("tup13", msgs == ["not all arguments converted during string formatting", "not enough arguments for format string",
                            "cannot use 'list' as a dict key (unhashable type: 'list')", "cannot use 'tuple' as a set element (unhashable type: 'list')"])
    x = *t, *(9,)
    check("tup14", repr(x) == "(1, 2, 9)" and repr(tuple("xy")) == "('x', 'y')" and repr((7).as_integer_ratio()) == "(7, 1)" and t.count(1) == 1 and t.index(2) == 1)

# ===== SECTION 80: pow() is **, and a float power's two errors =====
# Two-argument pow is the ** operator, big operands included. A zero to a
# negative power is ZeroDivisionError, a finite pair whose float power overflows
# is OverflowError(34, 'Result too large'), and an int too large for a double
# cannot be a float base. Rows from CPython 3.14.
def s80():
    v = [-10**20, 2.0, -1, 10**20, 0, 10**400, 2, "s"]
    check("fpw1", pow(v[0], v[1]) == 1e40 and repr(pow(v[2], v[3])) == "1" and repr(v[6] ** -v[3]) == "0.0" and repr(pow(v[0], -1)) == "-1e-20")
    msgs = []
    for f in [lambda: v[4] ** -1, lambda: pow(0.0, -2.5), lambda: v[1] ** v[3], lambda: pow(10.0, 400), lambda: v[5] ** -1, lambda: pow(v[7], 2)]:
        try:
            f()
        except (ZeroDivisionError, OverflowError, TypeError) as e:
            msgs.append(type(e).__name__ + ": " + str(e))
    check("fpw2", msgs == ["ZeroDivisionError: zero to a negative power", "ZeroDivisionError: zero to a negative power",
                           "OverflowError: (34, 'Result too large')", "OverflowError: (34, 'Result too large')",
                           "OverflowError: int too large to convert to float", "TypeError: unsupported operand type(s) for ** or pow(): 'str' and 'int'"])

# ===== SECTION 81: float(str) takes the whole string or raises =====
# float_from_string: whitespace stripped (U+001C..U+001F are not stripped), a
# sign, inf / infinity / nan in any case, or a decimal literal whose underscores
# stand between digits. Rows from CPython 3.14.
def s81():
    v = ["1_000", " 1_0.5_5e1_0\n", "\u00a0-.5\u3000", "5.", "1e400"]
    check("fls1", [float(x) for x in v] == [1000.0, 105500000000.0, -0.5, 5.0, float("inf")])
    bad = ["Infinitx", "-Infinitx", "Infinityy", "1.5abc", "", "  ", "1__0", "_1", "1_", "1_.5", "1e_5", "0x10", "1e", ".", "-", "\x1c1", "1 2"]
    msgs = []
    for x in bad:
        try:
            msgs.append(repr(float(x)))
        except ValueError as e:
            msgs.append(str(e))
    check("fls2", msgs[0] == "could not convert string to float: 'Infinitx'" and msgs[4] == "could not convert string to float: ''")
    check("fls3", all(m.startswith("could not convert string to float: ") for m in msgs) and len(msgs) == 17)
    check("fls4", msgs[15] == "could not convert string to float: '\\x1c1'")

# ===== SECTION 82: a negative base to a fractional power is complex =====
# float_pow hands a negative finite base with a non-integral exponent to
# complex_pow (_Py_c_pow), whose cos and sin are the platform libm's. Rows from
# CPython 3.14.
def s82():
    import math
    v = [-3, 0.5, -8, 1/3, -2.0, 2.5, -1e300, 1.5, -1, -0.5, -0.0, 0.5]
    check("cpw1", repr(v[0] ** v[1]) == "(1.0605752387249068e-16+1.7320508075688772j)")
    check("cpw2", repr(pow(v[2], v[3])) == "(1.0000000000000002+1.7320508075688772j)" and repr(v[4] ** v[5]) == "(1.7319121124709869e-15+5.656854249492381j)")
    check("cpw3", repr(v[8] ** v[9]) == "(6.123233995736766e-17-1j)" and repr(v[10] ** v[11]) == "0.0")
    try:
        v[6] ** v[7]
        check("cpw4", False)
    except OverflowError as e:
        check("cpw4", str(e) == "complex exponentiation")
    x = -4
    x **= 0.5
    check("cpw5", type(x).__name__ == "complex" and repr(math.cos(math.pi / 2)) == "6.123233995736766e-17")
    check("cpw6", repr(math.sin(math.pi)) == "1.2246467991473532e-16" and repr(math.atan2(1.0, -1.0)) == "2.356194490192345")

# ===== SECTION 83: augmented assignment names its own operator =====
# PyNumber_InPlace*: an instance's __isub__ first; when neither operand serves
# the operator the TypeError says "-=". `x *= seq` repeats a right-hand sequence
# only when x has no sequence slots. str % a list or dict never complains about
# leftover arguments. Rows from CPython 3.14.
def s83():
    class C:
        def __init__(self):
            self.v = 1
        def __isub__(self, o):
            self.v = self.v + o
            return self
    class A:
        def __init__(self):
            self.v = 1
        def __sub__(self, o):
            return self.v - o
    vals = [4, "s", {1}, [1], 2.5, A()]
    def aug(op, x, y):
        try:
            if op == "-":
                x -= y
            elif op == "**":
                x **= y
            elif op == "*":
                x *= y
            elif op == "+":
                x += y
            else:
                x <<= y
            return repr(x)
        except TypeError as e:
            return str(e)
    msgs = [aug("-", vals[0], vals[1]), aug("**", vals[0], vals[1]), aug("*", vals[2], vals[1]),
            aug("+", vals[4], vals[3]), aug("<<", vals[4], vals[0]), aug("-", vals[5], vals[1]), aug("*", vals[0], "ab")]
    check("aug1", msgs[0] == "unsupported operand type(s) for -=: 'int' and 'str'" and msgs[1] == "unsupported operand type(s) for **=: 'int' and 'str'")
    check("aug2", msgs[2] == "unsupported operand type(s) for *=: 'set' and 'str'" and msgs[3] == "unsupported operand type(s) for +=: 'float' and 'list'")
    check("aug3", msgs[4] == "unsupported operand type(s) for <<=: 'float' and 'int'" and msgs[5] == "unsupported operand type(s) for -: 'int' and 'str'")
    check("aug4", msgs[6] == "'abababab'")
    c = C()
    c -= 5
    check("aug5", c.v == 6 and "s" % [1] == "s" and "%s" % {"a": 1} == "{'a': 1}")

# ===== SECTION 84: hash() =====
# CPython's values for the numbers, None, tuples and frozensets (Py_uhash_t
# arithmetic modulo 2^61 - 1, xxHash for a tuple); str's is randomized per
# process, so only its invariants are asserted. Rows from CPython 3.14.
def s84():
    class H:
        def __hash__(self):
            return 2 ** 70
    class E:
        def __eq__(self, o):
            return True
    class P:
        pass
    v = [-1, 2**61, -2**61, 10**30, 1.5, 0.1, 1e300, 5e-324, 2.0**70, float("-inf"), 1+2j, (1, 2), ((1, 2), 3), (None, True, 1.5), frozenset({1, 2}), None]
    h = [hash(x) for x in v]
    check("hsh1", h[0:4] == [-2, 1, -2, 465258685558744706] and h[4] == 1152921504606846977)
    check("hsh2", h[5:10] == [230584300921369408, 1224995262755759164, 16777216, 512, -314159] and h[10] == 2000007)
    check("hsh3", h[11:14] == [-3550055125485641917, -333907151259015829, -4953812201219247177])
    check("hsh4", h[14] == -1826646154956904602 and h[15] == 4238894112 and hash(frozenset()) == 133146708735736)
    p = P()
    check("hsh5", hash(H()) == 512 and hash(p) == hash(p) and hash("") == 0 and hash("ab") == hash("ab") and hash(1) == hash(1.0) == hash(True))
    msgs = []
    for x in [[1], {1: 2}, {1}, E(), (1, [2])]:
        try:
            hash(x)
        except TypeError as e:
            msgs.append(str(e))
    check("hsh6", msgs == ["unhashable type: 'list'", "unhashable type: 'dict'", "unhashable type: 'set'", "unhashable type: 'E'", "unhashable type: 'list'"])

# ===== SECTION 85: a tuple is its own tuple, and tuple has list's operators =====
# tuple(t) hands back t; tuple's dunders are list's less the mutating ones, and
# its named methods are count and index. Rows from CPython 3.14.
def s85():
    t = (1, 2)
    check("tpl1", tuple(t) is t and tuple([1]) is not tuple([1]) and list(t) is not t)
    check("tpl2", hasattr(tuple, "__add__") and hasattr(tuple, "__len__") and hasattr(tuple, "count") and not hasattr(tuple, "append"))
    check("tpl3", not hasattr(tuple, "__setitem__") and not hasattr((), "__imul__") and not hasattr(tuple, "__delitem__"))

# ===== SECTION 86: dict views =====
# keys(), values() and items() are live views: they print as dict_keys([...]),
# follow later changes to the dict, test membership, are not subscriptable, and
# a keys or items view is set-like against any iterable. Rows from CPython 3.14.
def s86():
    d = {"a": 1, "b": 2}
    k = d.keys()
    v = d.values()
    it = d.items()
    check("vw1", repr(k) == "dict_keys(['a', 'b'])" and str(v) == "dict_values([1, 2])" and repr(it) == "dict_items([('a', 1), ('b', 2)])")
    d["c"] = 3
    check("vw2", len(k) == 3 and list(v) == [1, 2, 3] and type(it).__name__ == "dict_items" and bool({}.keys()) == False)
    check("vw3", "c" in k and 3 in v and ("a", 1) in it and ("a", 2) not in it and "a" not in it)
    check("vw4", k == {"a", "b", "c"} and k != ["a", "b", "c"] and v != d.values() and it == {("a", 1), ("b", 2), ("c", 3)})
    check("vw5", sorted(k & ["a", "z"]) == ["a"] and sorted(k - {"a"}) == ["b", "c"] and sorted({"q"} | k) == ["a", "b", "c", "q"])
    check("vw6", k.isdisjoint({"q"}) and not k.isdisjoint(["a"]) and type(iter(v)).__name__ == "dict_valueiterator")
    msgs = []
    for f in [lambda: k[0], lambda: hash(it)]:
        try:
            f()
        except TypeError as e:
            msgs.append(str(e))
    check("vw7", msgs == ["'dict_keys' object is not subscriptable", "unhashable type: 'dict_items'"])
    x, y, z = d.values()
    check("vw8", [x, y, z] == [1, 2, 3] and ", ".join(d.keys()) == "a, b, c" and [*d.items()][2] == ("c", 3) and dict(d.items()) == d)

# ===== SECTION 87: attributes bound to None, and id() =====
# A class or instance attribute holding None is still an attribute, and
# `__hash__ = None` makes a class unhashable. id() is an int that is equal
# exactly for the same object, and hash(o) == id(o) >> 4 for an object with
# the default hash. Rows from CPython 3.14.
def s87():
    class N:
        __hash__ = None
        x = None
    class M(N):
        pass
    class I:
        def __init__(self):
            self.v = None
    i = I()
    check("non1", N.__hash__ is None and N.x is None and N().x is None and i.v is None and hasattr(i, "v") and hasattr(N, "__hash__"))
    msgs = []
    for o in [N(), M()]:
        try:
            hash(o)
        except TypeError as e:
            msgs.append(str(e))
    check("non2", msgs == ["unhashable type: 'N'", "unhashable type: 'M'"])
    del i.v
    check("non3", not hasattr(i, "v"))
    i.v = None
    check("non4", hasattr(i, "v") and vars(i) == {"v": None})
    class O:
        pass
    o = O()
    xs = [1]
    ys = xs
    check("id1", id(o) == id(o) and id(o) != id(O()) and id(xs) == id(ys) and id(xs) != id([1]) and type(id(o)) is int)
    check("id2", hash(o) == id(o) >> 4)

# ===== SECTION 88: the builtin types' dunders =====
# hasattr on a type object answers every dunder the type has, object's
# included (list.__hash__ is None, dict.__lt__ declines); set, frozenset,
# complex and bool are served like list and int, so a subclass of set or
# complex works; xs *= n repeats in place; 0j is false. Rows from CPython 3.14
# (left out: __new__, __sizeof__, and type's own __or__).
def s88():
    names = ["__hash__", "__lt__", "__eq__", "__imul__", "__iadd__", "__radd__", "__rmul__", "__ior__", "__reversed__", "__init__", "__getitem__", "__complex__", "__format__", "__rand__"]
    rows = []
    for t in [list, tuple, dict, set, frozenset, bool, complex, str]:
        rows.append("".join(["1" if hasattr(t, n) else "0" for n in names]))
    check("dun1", rows[0:4] == ["11111010111010", "11100010011010", "11100001111010", "11100001010011"])
    check("dun2", rows[4:8] == ["11100000010011", "11100110010011", "11100110010110", "11100010011010"])
    check("dun3", list.__hash__ is None and [].__hash__ is None and dict.__lt__({}, {}) is NotImplemented and set.__or__({1}, [2]) is NotImplemented)
    s1 = {1, 2}
    r = set.__ior__(s1, {3})
    check("dun4", r is s1 and sorted(s1) == [1, 2, 3] and set.__lt__({1}, s1) and frozenset.__hash__(frozenset([1])) == hash(frozenset([1])))
    class S(set):
        pass
    class C(complex):
        pass
    st = S([1, 2])
    st.add(5)
    check("dun5", len(st) == 3 and 5 in st and sorted(st | {9}) == [1, 2, 5, 9] and C(1, 2) == 1 + 2j and C(3, 4) * 2 == 6 + 8j)
    xs = [1, 2]
    ys = xs
    xs *= 2
    zs = [0j, 1j]
    check("dun6", ys == [1, 2, 1, 2] and xs is ys and not zs[0] and bool(zs[1]) and list.__iadd__(ys, (3,)) is xs)

# ===== SECTION 89: object() =====
# object() is a featureless instance: equal and hashed by identity, an instance
# of object alone, and without a __dict__, so it takes no attribute - the same
# refusal, word for word, as a write a class's __slots__ does not allow.
# Rows from CPython 3.14.
def s89():
    o = object()
    p = object()
    check("obj1", type(o) is object and o.__class__ is object and type(o).__name__ == "object")
    check("obj2", o == o and o != p and not (o == p) and o is not p and bool(o))
    check("obj3", isinstance(o, object) and not isinstance(o, int) and not callable(o) and callable(object))
    d = {o: 1, p: 2}
    check("obj4", hash(o) == hash(o) and d[o] == 1 and d[p] == 2 and len({object() for _ in range(3)}) == 3)
    check("obj5", repr(o).startswith("<object object") and str(o) == repr(o) and [o, p].index(p) == 1)
    msgs = []
    try:
        o.x = 1
    except AttributeError as e:
        msgs.append(str(e))
    try:
        del o.x
    except AttributeError as e:
        msgs.append(str(e))
    for a in [(1,), ()]:
        try:
            object(*a) if a else object(x=1)
        except TypeError as e:
            msgs.append(str(e))
    check("obj6", msgs == ["'object' object has no attribute 'x' and no __dict__ for setting new attributes"] * 2
          + ["object() takes no arguments"] * 2)
    check("obj7", not hasattr(o, "x") and getattr(o, "x", 5) == 5)

    class Sl:
        __slots__ = ("a",)
    try:
        Sl().b = 1
    except AttributeError as e:
        msgs.append(str(e))
    check("obj8", msgs[-1] == "'Sl' object has no attribute 'b' and no __dict__ for setting new attributes")
    sentinel = object()

    def f(a=sentinel):
        return a is sentinel
    check("obj9", f() and not f(0))

# ===== SECTION 92: X | Y over types =====
# type's __or__ makes a typing.Union: the arguments flattened and deduplicated
# in order (int | int is int itself), None standing as NoneType and printed as
# None, equal and hashed regardless of order, and accepted by isinstance and
# issubclass. A non-type operand is still a TypeError. Rows from CPython 3.14.
class S92A:
    pass

class S92B(S92A):
    pass

def s92():
    ts = [int, str, None, S92A, S92B, float, type(None), bool]
    u = ts[0] | ts[1]
    check("uni1", repr(u) == "int | str" and str(ts[1] | ts[2]) == "str | None" and repr(type(u)) == "<class 'typing.Union'>"
          and type(u).__name__ == "Union" and str(ts[3] | ts[2]) == "__main__.S92A | None" and str(ts[2] | ts[0]) == "None | int")
    check("uni2", (ts[0] | ts[0]) is int and str((u | ts[5]) | ts[2]) == "int | str | float | None" and str(ts[0] | (ts[1] | ts[0])) == "int | str"
          and str(ts[6] | ts[0]) == "None | int" and (ts[6] | ts[2]) is type(None))
    check("uni3", u == (ts[1] | ts[0]) and u != ts[0] and not (u != (ts[1] | ts[0])) and hash(u) == hash(ts[1] | ts[0])
          and {u: 1}[ts[1] | ts[0]] == 1 and type(u) is type(ts[3] | ts[2]) and not isinstance(u, type))
    check("uni4", u.__args__ == (int, str) and (ts[2] | ts[0]).__args__ == (type(None), int) and [u] == [ts[1] | ts[0]])
    check("uni5", isinstance(None, ts[0] | ts[2]) and isinstance(S92B(), ts[3] | ts[2]) and not isinstance(2.5, u)
          and isinstance(3, (ts[1] | ts[2], ts[0])) and issubclass(ts[7], ts[0] | ts[1]) and not issubclass(ts[0], ts[1] | ts[2]))
    msgs = []
    for x, y in [(ts[0], 3), (3, ts[0]), (ts[0], "s"), (None, None)]:
        try:
            x | y
        except TypeError as e:
            msgs.append(str(e))
    check("uni6", msgs == ["unsupported operand type(s) for |: 'type' and 'int'", "unsupported operand type(s) for |: 'int' and 'type'",
                           "unsupported operand type(s) for |: 'type' and 'str'", "unsupported operand type(s) for |: 'NoneType' and 'NoneType'"])

# ===== SECTION 93: object's own methods, and __new__ =====
# Every class inherits object's __init__, __new__, __eq__, __ne__, __hash__,
# __repr__, __str__, __format__, __setattr__, __getattribute__ and __delattr__:
# super().__init__() works in a class whose only base is object, and
# A.__init__ is object.__init__. A class's own __new__ makes the instance and
# __init__ runs only on an instance of the class; T.__new__ of a builtin type
# converts, or makes a bare subclass instance. A class with neither __init__ nor
# __new__ takes no arguments. Rows from CPython 3.14.
class S93A:
    pass

class S93P(int):
    def __new__(cls, v):
        return super().__new__(cls, v * 2)

class S93S:
    inst = None
    def __new__(cls):
        if cls.inst is None:
            cls.inst = super().__new__(cls)
            cls.inst.n = 0
        return cls.inst
    def __init__(self):
        self.n += 1

class S93W:
    ran = []
    def __new__(cls, x):
        return x
    def __init__(self, x):
        S93W.ran.append(x)

class S93Up(str):
    def __new__(cls, s):
        o = str.__new__(cls, s.upper())
        o.tag = 1
        return o

class S93Base:
    def __init__(self, a, b=2):
        super().__init__()
        self.a = a
        self.b = b

class S93Kid(S93Base):
    def __init__(self, a, **kw):
        super().__init__(a, **kw)
        self.k = True

class S93Bad:
    def __init__(self, x):
        super().__init__(x)

class S93Err(Exception):
    def __new__(cls, *a):
        o = super().__new__(cls, *a)
        o.extra = len(a)
        return o

def s93():
    check("new1", int.__new__(int, "12") + 1 == 13 and float.__new__(float, "1.5") == 1.5 and str.__new__(str, 5) == "5"
          and list.__new__(list, [1]) == [] and tuple.__new__(tuple, [1, 2]) == (1, 2))
    check("new2", S93P(4) == 8 and type(S93P(4)) is S93P and S93P(4) + 1 == 9 and isinstance(S93P(1), int))
    a = S93S()
    b = S93S()
    check("new3", a is b and b.n == 2 and S93W(5) == 5 and S93W.ran == [])
    check("new4", S93Up("hi") == "HI" and type(S93Up("x")) is S93Up and S93Up("ab").tag == 1 and S93Up("ab") + "!" == "AB!")
    check("new5", object.__new__(S93A).__class__ is S93A and S93A.__new__ is object.__new__ and int.__new__ is int.__new__
          and hasattr(int, "__new__") and hasattr(S93A(), "__new__") and S93P.__new__ is not int.__new__)
    msgs = []
    for f in [lambda: object.__new__(S93A, 1), lambda: object.__new__(S93P), lambda: object.__new__(5),
              lambda: int.__new__(S93A), lambda: S93A(1), lambda: S93A(k=1), lambda: S93Bad(1),
              lambda: object.__init__(S93A(), 1), lambda: object.__format__(S93A(), "x")]:
        try:
            f()
        except TypeError as e:
            msgs.append(str(e))
    check("new6", msgs == ["S93A() takes no arguments", "object.__new__(S93P) is not safe, use int.__new__()",
                           "object.__new__(X): X is not a type object (int)", "int.__new__(S93A): S93A is not a subtype of int",
                           "S93A() takes no arguments", "S93A() takes no arguments",
                           "object.__init__() takes exactly one argument (the instance to initialize)",
                           "S93A.__init__() takes exactly one argument (the instance to initialize)",
                           "unsupported format string passed to S93A.__format__"])
    o = S93A()
    check("obj1", S93A.__init__ is object.__init__ and object.__init__(o) is None and object.__eq__(1, 1) is True
          and object.__eq__(o, S93A()) is NotImplemented and object.__ne__(1, 2) is True)
    check("obj2", o.__eq__(o) is True and o.__ne__(o) is False and o.__eq__(S93A()) is NotImplemented
          and o.__hash__() == hash(o) == object.__hash__(o) and object.__str__(o) == object.__repr__(o) == repr(o))
    names = ["__init__", "__new__", "__eq__", "__ne__", "__hash__", "__repr__", "__str__", "__format__",
             "__setattr__", "__getattribute__", "__delattr__", "__class__"]
    check("obj3", all(hasattr(object, n) and hasattr(o, n) and hasattr(S93A, n) for n in names) and object.__class__ is type)
    object.__setattr__(o, "q", 3)
    seen = object.__getattribute__(o, "q")
    object.__delattr__(o, "q")
    try:
        object.__setattr__(object(), "q", 1)
    except AttributeError as e:
        msgs.append(str(e))
    check("obj4", seen == 3 and not hasattr(o, "q") and object.__format__(5, "") == "5"
          and msgs[-1] == "'object' object has no attribute 'q' and no __dict__ for setting new attributes")
    k = S93Kid(1, b=5)
    check("obj5", (k.a, k.b, k.k) == (1, 5, True) and S93Base(7).b == 2)
    try:
        raise S93Err("x", "y")
    except S93Err as e:
        check("obj6", e.args == ("x", "y") and e.extra == 2)
    check("obj7", str(object.__init__) == "<slot wrapper '__init__' of 'object' objects>" and type(object.__init__).__name__ == "wrapper_descriptor"
          and object.__eq__.__name__ == "__eq__" and type(o.__init__).__name__ == "method-wrapper")
    check("obj8", object.__lt__(1, 2) is NotImplemented and o.__lt__(o) is NotImplemented and o.__ge__(1) is NotImplemented
          and object.__lt__ is not object.__le__ and str(object.__gt__) == "<slot wrapper '__gt__' of 'object' objects>")

# ===== SECTION 94: an explicit None is an argument =====
# A defaulted parameter takes its default only when the call supplies no
# argument: None passed by position or by keyword is a value like any other.
# Rows from CPython 3.14.
class S94C:
    def m(self, x=3, *, y=4):
        return (x, y)

def s94():
    def k(a=5):
        return a
    def k2(a, b=None, *c, d=7, **e):
        return [a, b, c, d, e]
    def k3(a=5, b=6):
        return (a, b)
    def g(a=1):
        yield a
    nn = [None]
    f = lambda x=4: x
    check("none1", k(nn[0]) is None and k() == 5 and f(nn[0]) is None and f() == 4 and list(g(nn[0])) == [None])
    check("none2", k2(1, nn[0], d=nn[0]) == [1, None, (), None, {}] and k2(1) == [1, None, (), 7, {}])
    check("none3", k3(b=nn[0]) == (5, None) and k3(nn[0], nn[0]) == (None, None) and k3(a=nn[0]) == (None, 6))
    check("none4", S94C().m(nn[0]) == (None, 4) and S94C().m(y=nn[0]) == (3, None) and S94C().m() == (3, 4))

# ===== SECTION 95: the attribute hooks, and deleting an attribute =====
# Only an object's OWN attribute can be deleted: a class member, a missing
# name or an already deleted one is an AttributeError, on an instance and on a
# class. A class's __getattribute__ runs for every instance attribute read and
# method call, its __getattr__ only when the ordinary lookup raises
# AttributeError, __setattr__ / __delattr__ for every write and delete;
# object.__getattribute__ / __setattr__ / __delattr__ are the plain ones.
# getattr's default and hasattr catch exactly AttributeError. Rows from
# CPython 3.14.
class S95C:
    k = 1
    def m(self):
        return 1

class S95D(S95C):
    pass

class S95Get:
    k = 2
    def __getattr__(self, name):
        if name == "f":
            return lambda x: x * 3
        if name == "no":
            raise AttributeError("no " + name)
        return "ga:" + name

class S95Set:
    def __init__(self):
        object.__setattr__(self, "log", [])
    def __setattr__(self, name, v):
        self.log.append(name)
        object.__setattr__(self, name, v * 2)
    def __delattr__(self, name):
        self.log.append("-" + name)
        object.__delattr__(self, name)

class S95Kid(S95Set):
    def __init__(self):
        super().__init__()
        self.n = 7

class S95GA:
    reads = []
    def __init__(self):
        self.v = 1
    def __getattribute__(self, name):
        S95GA.reads.append(name)
        if name == "secret":
            return 42
        if name == "bad":
            raise ValueError("bad")
        return object.__getattribute__(self, name)
    def __getattr__(self, name):
        return "fb:" + name
    def m(self):
        return "m"

def s95():
    msgs = []
    c = S95C()
    c.x = None
    del c.x
    for t in [c, c, c, S95D, S95D, S95C]:
        for n in ["missing", "k", "m", "x"]:
            try:
                delattr(t, n)
                msgs.append(n)
            except AttributeError as e:
                msgs.append(str(e))
    check("dela1", msgs[:4] == ["'S95C' object has no attribute 'missing'", "'S95C' object has no attribute 'k'",
                                "'S95C' object has no attribute 'm'", "'S95C' object has no attribute 'x'"])
    check("dela2", msgs[12:16] == ["type object 'S95D' has no attribute 'missing'", "type object 'S95D' has no attribute 'k'",
                                   "type object 'S95D' has no attribute 'm'", "type object 'S95D' has no attribute 'x'"]
          and msgs[20:] == ["type object 'S95C' has no attribute 'missing'", "k", "m", "type object 'S95C' has no attribute 'x'"]
          and not hasattr(S95C(), "m"))
    g = S95Get()
    g.x = 5
    check("hook1", (g.x, g.k, g.zz, getattr(g, "yy"), g.f(4)) == (5, 2, "ga:zz", "ga:yy", 12)
          and hasattr(g, "q") and not hasattr(g, "no") and getattr(g, "no", "d") == "d")
    s = S95Set()
    s.a = 3
    setattr(s, "b", 4)
    s.a += 1
    del s.b
    check("hook2", (s.a, hasattr(s, "b"), s.log) == (14, False, ["a", "b", "a", "-b"]) and S95Kid().n == 14)
    ga = S95GA()
    S95GA.reads.clear()
    check("hook3", (ga.v, ga.secret, ga.m(), ga.other) == (1, 42, "m", "fb:other")
          and S95GA.reads == ["v", "secret", "m", "other"])
    try:
        hasattr(ga, "bad")
    except ValueError as e:
        msgs.append(str(e))
    check("hook4", msgs[-1] == "bad" and object.__getattribute__(ga, "v") == 1)

# ===== SECTION 96: generic aliases, unions of values, huge exponents, sys =====
# list[int] is a types.GenericAlias: printed as written (a tuple subscript is
# the argument list, a class module-qualified, ... and None by name), its
# __args__ a tuple, and calling it calls list. A union takes any value but a
# tuple beside it (int | str | 1.5); a type alone does not. 0, 1 and -1 raised
# to an int past 2^53 are exact. sys.set_int_max_str_digits moves the limit
# str() and int() enforce. Rows from CPython 3.14.
class S96C:
    pass

def s96():
    ts = [int, str, list, dict, tuple, S96C, None, 1.5, (1, 2), 3, 3.0]
    a = ts[2][ts[0]]
    check("gal1", (repr(a), str(ts[3][ts[1], ts[0]]), repr(ts[4][ts[0], ...]), repr(ts[2][ts[5]]))
          == ("list[int]", "dict[str, int]", "tuple[int, ...]", "list[__main__.S96C]"))
    check("gal2", (repr(ts[2][ts[6]]), repr(ts[2][ts[9]]), repr(ts[2]["x"]), repr(ts[3][ts[1], a]), repr(ts[4][()]))
          == ("list[None]", "list[3]", "list['x']", "dict[str, list[int]]", "tuple[()]"))
    check("gal3", a.__args__ == (int,) and ts[3][ts[1], ts[0]].__args__ == (str, int) and a.__origin__ is list
          and a == ts[2][ts[0]] and a != ts[2][ts[1]] and repr(type(a)) == "<class 'types.GenericAlias'>"
          and type(a).__name__ == "GenericAlias" and a() == [] and f"{[a]}" == "[list[int]]")
    u = ts[0] | ts[1]
    check("uni7", (repr(u | ts[7]), repr(ts[9] | u), repr(u | ts[9] | ts[10]), (u | ts[7]).__args__)
          == ("int | str | 1.5", "3 | int | str", "int | str | 3", (int, str, 1.5)))
    check("uni8", (repr(ts[0] | a), repr(a | ts[6]), repr(ts[2][ts[0] | ts[6]])) == ("int | list[int]", "list[int] | None", "list[int | None]"))
    msgs = []
    for x, y in [(u, ts[8]), (ts[0], ts[7]), (a, ts[9])]:
        try:
            x | y
        except TypeError as e:
            msgs.append(str(e))
    check("uni9", msgs == ["Union[arg, ...]: each arg must be a type. Got (1, 2).", "unsupported operand type(s) for |: 'type' and 'float'",
                           "unsupported operand type(s) for |: 'types.GenericAlias' and 'int'"])
    es = [10**20 + 1, 10**20, 2**53 + 1, 2**53 + 2, 10**30 + 7]
    check("pow1", [(-1) ** e for e in es] == [-1, 1, -1, 1, -1] and [b ** es[0] for b in [0, 1, True]] == [0, 1, 1]
          and pow(-1, es[0]) == -1 and (-1) ** (2**64 + 1) == -1)
    import sys
    check("sys1", sys.get_int_max_str_digits() == 4300)
    sys.set_int_max_str_digits(640)
    try:
        str(10**700)
    except ValueError as e:
        msgs.append(str(e))
    try:
        sys.set_int_max_str_digits(639)
    except ValueError as e:
        msgs.append(str(e))
    check("sys2", msgs[-2:] == ["Exceeds the limit (640 digits) for integer string conversion; use sys.set_int_max_str_digits() to increase the limit",
                                "maxdigits must be >= 640 or 0 for unlimited"] and len(str(-10**639)) == 641)
    sys.set_int_max_str_digits(maxdigits=0)
    check("sys3", sys.get_int_max_str_digits() == 0 and len(str(10**700)) == 701 and int("9" * 700) % 1000 == 999)
    sys.set_int_max_str_digits(4300)

# ===== SECTION 98: a call that does not fit the signature =====
# Every mismatch is a TypeError naming the function by its __qualname__, in
# CPython's order: a keyword error, too many positionals, the missing
# positional, the missing keyword-only parameters. A generator function checks
# at the call, not at the first next(). Rows from CPython 3.14.
class S98C:
    def m(self, a):
        return a
    def __init__(self, x=1):
        self.x = x

def s98():
    def err(fn):
        try:
            fn()
        except TypeError as e:
            return str(e)
        return "no error"
    def f(a, b=2):
        return a
    def g(a, b, c, d=1):
        return a
    def h(x, *, k, j):
        return k
    def q(a, b=1, *, k):
        return a
    def u():
        return 0
    def v(a, *rest):
        return a
    def p(a, b, /, c):
        return a
    def p3(a, /, **kw):
        return kw
    def gen(a):
        yield a
    lam = lambda x: x
    check("arity1", err(lambda: f()) == "s98.<locals>.f() missing 1 required positional argument: 'a'"
          and err(lambda: f(b=3)) == err(lambda: f()) and err(lambda: v()) == "s98.<locals>.v() missing 1 required positional argument: 'a'")
    check("arity2", err(lambda: g(1)) == "s98.<locals>.g() missing 2 required positional arguments: 'b' and 'c'"
          and err(lambda: g()) == "s98.<locals>.g() missing 3 required positional arguments: 'a', 'b', and 'c'")
    check("arity3", err(lambda: h(1)) == "s98.<locals>.h() missing 2 required keyword-only arguments: 'k' and 'j'"
          and err(lambda: h(1, j=2)) == "s98.<locals>.h() missing 1 required keyword-only argument: 'k'")
    check("arity4", err(lambda: u(1)) == "s98.<locals>.u() takes 0 positional arguments but 1 was given"
          and err(lambda: f(1, 2, 3)) == "s98.<locals>.f() takes from 1 to 2 positional arguments but 3 were given"
          and err(lambda: lam(1, 2)) == "s98.<locals>.<lambda>() takes 1 positional argument but 2 were given")
    check("arity5", err(lambda: q(1, 2, 3, k=1)) == "s98.<locals>.q() takes from 1 to 2 positional arguments but 3 positional arguments (and 1 keyword-only argument) were given")
    check("arity6", err(lambda: f(1, a=2)) == "s98.<locals>.f() got multiple values for argument 'a'"
          and err(lambda: f(1, z=2)) == "s98.<locals>.f() got an unexpected keyword argument 'z'"
          and err(lambda: p(b=1, a=2, c=3)) == "s98.<locals>.p() got some positional-only arguments passed as keyword arguments: 'a, b'"
          and err(lambda: p3(a=1)) == "s98.<locals>.p3() missing 1 required positional argument: 'a'" and p3(1, a=2) == {"a": 2})
    check("arity7", err(lambda: S98C().m()) == "S98C.m() missing 1 required positional argument: 'a'"
          and err(lambda: S98C(1, 2)) == "S98C.__init__() takes from 1 to 2 positional arguments but 3 were given")
    check("arity8", err(lambda: gen()) == "s98.<locals>.gen() missing 1 required positional argument: 'a'"
          and err(lambda: f(None)) == "no error" and err(lambda: h(None, k=None, j=None)) == "no error")

# ===== SECTION 99: an instance's __dict__ is its attribute storage =====
# o.__dict__ is the same dict at every read, and a write through either side
# is seen through the other: an item store, an item delete or a dict method
# changes the attributes, an attribute write or delete changes a dict read
# earlier. A deleted then re-created attribute goes last. Rows from CPython
# 3.14.
class S99P:
    cv = 9
    def __init__(self, **kw):
        self.__dict__.update(kw)

def s99():
    o = S99P(a=1, b=None)
    check("dict1", o.a == 1 and o.b is None and hasattr(o, "b") and vars(o) == {"a": 1, "b": None})
    d = o.__dict__
    o.__dict__["x"] = 2
    o.z = 3
    check("dict2", hasattr(o, "x") and o.x == 2 and list(d.items()) == [("a", 1), ("b", None), ("x", 2), ("z", 3)]
          and d is o.__dict__ and vars(o) is d)
    del d["a"]
    o.b = 4
    check("dict3", not hasattr(o, "a") and d == {"b": 4, "x": 2, "z": 3})
    del o.x
    o.x = 5
    check("dict4", list(d) == ["b", "z", "x"] and d.pop("z") == 3 and not hasattr(o, "z"))
    d.setdefault("s", 6)
    vars(o)["w"] = 7
    check("dict5", o.s == 6 and o.w == 7 and len(d) == 4 and d.get("q", 8) == 8)
    d.clear()
    check("dict6", o.__dict__ == {} and not hasattr(o, "b") and o.cv == 9)
    o.k = 1
    check("dict7", d == {"k": 1} and type(d).__name__ == "dict" and S99P(i=1).__dict__ is not d)
    # A deleted attribute is absent: the class member of that name shows
    # through, and a name the class lacks is a catchable AttributeError.
    o2 = S99P(cv=1, b=2)
    del o2.cv
    del o2.__dict__["b"]
    try:
        o2.b
        msg = "no error"
    except AttributeError as e:
        msg = str(e)
    check("dict8", o2.cv == 9 and msg == "'S99P' object has no attribute 'b'")
    # A __-named attribute is the program's like any other, in write order;
    # an exception keeps args and its chaining attributes out of __dict__.
    o3 = S99P(a=1)
    o3.__x__ = 2
    o3.b = 3
    del o3.a
    o3.a = 4
    check("dict9", vars(o3) == {"__x__": 2, "b": 3, "a": 4} and list(vars(o3)) == ["__x__", "b", "a"])
    o3.__dict__["__y__"] = 5
    del o3.__dict__["__x__"]
    check("dict10", o3.__y__ == 5 and not hasattr(o3, "__x__") and list(vars(o3)) == ["b", "a", "__y__"])
    e = ValueError("v")
    e.__cause__ = KeyError("k")
    e.n = 1
    check("dict11", vars(ValueError("v")) == {} and e.__dict__ == {"n": 1} and e.args == ("v",))

# ===== SECTION 100: property =====
# property(fget, fset, fdel, doc) is a data descriptor: an instance read,
# write, delete or method call runs the accessor, a missing one is CPython's
# AttributeError naming the attribute and the class, and hasattr is a getattr
# that catches it. Read off the class it is the property object, whose
# getter / setter / deleter answer a copy with one accessor replaced. The
# decorator expressions are evaluated before the def rebinds the name, which
# is what `@x.setter def x` relies on. Rows from CPython 3.14.
class S100C:
    def __init__(self):
        self._x = 1
    @property
    def x(self):
        return self._x
    @x.setter
    def x(self, v):
        self._x = v * 2
    @x.deleter
    def x(self):
        del self._x
    def gety(self):
        return 10
    y = property(gety)
    z = property(doc="zz")
    @property
    def fn(self):
        return lambda a: a + self._x

class S100D(S100C):
    @S100C.x.getter
    def x(self):
        return -self._x
    @property
    def y(self):
        return super().y + 1

def s100():
    def err(fn):
        try:
            fn()
        except AttributeError as e:
            return str(e)
        return "no error"
    c = S100C()
    check("prop1", c.x == 1 and c.y == 10 and c.fn(1) == 2)
    c.x = 5
    check("prop2", c.x == 10 and c._x == 10)
    del c.x
    check("prop3", not hasattr(c, "_x") and not hasattr(c, "x") and hasattr(c, "y") and not hasattr(c, "z"))
    check("prop4", err(lambda: setattr(c, "y", 1)) == "property 'y' of 'S100C' object has no setter"
          and err(lambda: c.z) == "property 'z' of 'S100C' object has no getter"
          and err(lambda: delattr(c, "y")) == "property 'y' of 'S100C' object has no deleter")
    check("prop5", type(S100C.x).__name__ == "property" and isinstance(S100C.x, property) and type(S100C.z) is property
          and S100C.y.fget is S100C.gety and S100C.y.fset is None and S100C.z.__doc__ == "zz" and S100C.y.__doc__ is None)
    c.x = 3
    check("prop6", S100C.x.__get__(c, S100C) == 6 and S100C.x.__get__(None, S100C) is S100C.x and getattr(c, "z", "d") == "d")
    d = S100D()
    d.x = 4
    check("prop7", d.x == -8 and d._x == 8 and d.y == 11 and S100C().x == 1)
    p = property(lambda s: 42)
    S100C.q = p
    check("prop8", S100C().q == 42 and S100C.q is p and property(fget=S100C.gety, doc="g").__doc__ == "g")
    # A property's name in these errors is its own: the class-body name
    # __set_name__ gave it, else its getter's __name__, else none at all.
    S100C.q2 = property(lambda s: 1)
    S100C.q3 = S100C.y
    S100C.q4 = property()
    check("prop9", err(lambda: setattr(c, "q2", 1)) == "property '<lambda>' of 'S100C' object has no setter"
          and err(lambda: setattr(c, "q3", 1)) == "property 'y' of 'S100C' object has no setter"
          and err(lambda: c.q4) == "property of 'S100C' object has no getter"
          and S100C.y.__name__ == "y" and p.__name__ == "<lambda>")
    # Any other class-body member whose type defines __set_name__ is told its
    # owner and name once the class exists, in body order; a later class
    # attribute is not.
    log = []
    class Named:
        def __set_name__(self, owner, name):
            log.append(owner.__name__ + "." + name)
    class Own:
        a = Named()
        b = 1
        c = Named()
    Own.d = Named()
    check("prop10", log == ["Own.a", "Own.c"])

# ===== SECTION 101: copy, dataclasses and dir() =====
# copy.copy shares what the copy holds, copy.deepcopy copies it once per
# object (a cycle comes back to the copy); a class's __copy__ / __deepcopy__
# wins and an immutable value is itself. @dataclass generates __init__,
# __repr__, __eq__, the orderings and the hash its flags ask for, with
# CPython's argument errors; frozen=True refuses an assignment. dir() of a user
# class or instance is the sorted merge of object's names, every class's own
# members and the instance's attributes. A class's __dict__ holds its own
# members. Rows from CPython 3.14.
import copy
from dataclasses import dataclass, field, fields, asdict, astuple, replace, is_dataclass, FrozenInstanceError

class S101Node:
    def __init__(self, v):
        self.v = v
        self.me = self

class S101Own:
    def __copy__(self):
        return "copied"
    def __deepcopy__(self, memo):
        return "deep"

@dataclass
class S101P:
    x: int
    y: int = 0
    tags: list = field(default_factory=list)

@dataclass(frozen=True)
class S101F:
    a: int
    b: str = "s"

@dataclass(order=True)
class S101O:
    k: int
    n: str

@dataclass
class S101Base:
    a: int
    b: int = 2

@dataclass
class S101Sub(S101Base):
    c: int = 3
    a: int = 9
    def __post_init__(self):
        self.total = self.a + self.b + self.c

@dataclass
class S101Opts:
    a: int
    b: list = field(default_factory=list, init=False, repr=False)
    c: int = field(default=4, compare=False)

class S101Dir:
    x = 1
    def m(self):
        return self.__class__(2)

class S101DirSub(S101Dir):
    y: int = 2
    def __init__(self, v=0):
        self.z = v

def s101():
    a = [1, [2, 3], {"k": [4]}, (5, [6]), (7, 8)]
    b = copy.copy(a)
    d = copy.deepcopy(a)
    a[1].append(9)
    a[3][1].append(11)
    check("cpy1", b[1] == [2, 3, 9] and d[1] == [2, 3] and d[3] == (5, [6]) and d[4] is a[4] and d[3] is not a[3]
          and copy.copy((1, 2)) is not None and copy.deepcopy("s") == "s")
    n = S101Node([1])
    sh = copy.copy(n)
    dp = copy.deepcopy(n)
    n.v.append(2)
    shared = [0]
    pair = copy.deepcopy([shared, shared])
    check("cpy2", sh.v == [1, 2] and dp.v == [1] and sh.me is n and dp.me is dp and pair[0] is pair[1]
          and copy.copy(S101Own()) == "copied" and copy.deepcopy(S101Own()) == "deep")
    e = ValueError("bad", 3)
    e.note = "n"
    ec = copy.copy(e)
    check("cpy3", type(ec) is ValueError and ec.args == ("bad", 3) and ec.note == "n" and ec is not e)

    p = S101P(1)
    check("dc1", repr(p) == "S101P(x=1, y=0, tags=[])" and p == S101P(1) and p != S101P(2) and not (p == 3)
          and S101P(1).tags is not S101P(1).tags and repr(S101P(3, 4, ["a"])) == "S101P(x=3, y=4, tags=['a'])")
    msgs = []
    for f in [lambda: S101P(), lambda: S101P(1, 2, 3, 4), lambda: S101P(1, z=2), lambda: S101P(1, x=2)]:
        try:
            f()
        except TypeError as err:
            msgs.append(str(err))
    check("dc2", msgs == ["S101P.__init__() missing 1 required positional argument: 'x'",
                          "S101P.__init__() takes from 2 to 4 positional arguments but 5 were given",
                          "S101P.__init__() got an unexpected keyword argument 'z'",
                          "S101P.__init__() got multiple values for argument 'x'"])
    fr = S101F(1)
    try:
        fr.a = 2
        msg = "no error"
    except FrozenInstanceError as err:
        msg = type(err).__name__ + ": " + str(err) + " " + str(isinstance(err, AttributeError))
    try:
        hash(p)
        unh = "hashable"
    except TypeError as err:
        unh = str(err)
    check("dc3", msg == "FrozenInstanceError: cannot assign to field 'a' True" and hash(fr) == hash((1, "s"))
          and unh == "unhashable type: 'S101P'" and repr(fr) == "S101F(a=1, b='s')")
    check("dc4", S101O(1, "b") < S101O(2, "a") and not (S101O(1, "b") < S101O(1, "a")) and S101O(1, "a") <= S101O(1, "a")
          and [o.n for o in sorted([S101O(2, "x"), S101O(1, "y")])] == ["y", "x"]
          and repr(S101Sub()) == "S101Sub(a=9, b=2, c=3)" and S101Sub.__match_args__ == ("a", "b", "c")
          and S101Sub(1).total == 6 and S101P.__init__.__qualname__ == "S101P.__init__")
    check("dc5", [f.name for f in fields(S101P)] == ["x", "y", "tags"] and asdict(S101P(1, 2, [S101P(3)])) ==
          {"x": 1, "y": 2, "tags": [{"x": 3, "y": 0, "tags": []}]} and astuple(S101P(1)) == (1, 0, [])
          and repr(replace(S101P(1), y=5)) == "S101P(x=1, y=5, tags=[])" and is_dataclass(S101P)
          and is_dataclass(p) and not is_dataclass(1) and repr(S101Opts(1)) == "S101Opts(a=1, c=4)"
          and S101Opts(1, c=9) == S101Opts(1, c=8) and S101Opts(1).b == [] and not hasattr(S101P, "tags"))
    errs = []
    class Mut:
        a: list = []
    class Order:
        a: int = 1
        b: int
    for f in [lambda: field(default=1, default_factory=list), lambda: dataclass(Mut), lambda: dataclass(Order)]:
        try:
            f()
        except (TypeError, ValueError) as err:
            errs.append(str(err))
    check("dc6", errs == ["cannot specify both default and default_factory",
                          "mutable default <class 'list'> for field a is not allowed: use default_factory",
                          "non-default argument 'b' follows default argument 'a'"])

    o = S101DirSub(3)
    o.w = 4
    check("dir1", [n for n in dir(o) if not n.startswith("__")] == ["m", "w", "x", "y", "z"]
          and dir(object) == sorted(dir(object)) and len(dir(object)) == 24
          and "__init__" in dir(S101DirSub) and "__weakref__" in dir(S101Dir) and "__annotate_func__" in dir(o)
          and "__annotate_func__" not in dir(S101Dir))
    own = [k for k in S101DirSub.__dict__ if k in ("__module__", "x", "y", "m", "__init__")]
    check("dir2", own == ["__module__", "y", "__init__"] and "x" not in S101DirSub.__dict__
          and S101Dir.__dict__.get("x") == 1 and o.m().__class__ is S101DirSub and o.m().z == 2)

# ===== SECTION 102: a user __hash__ / __eq__ keys dicts and sets =====
# A dict or set compares keys by hash first and then by ==, so a class defining
# both is found by an equal instance it never saw: lookup, insert, `in`, del,
# get, pop, discard and the set algebra all agree. An __eq__ without __hash__
# makes the class unhashable, named by its own class. A list or tuple holding an
# instance compares through its __eq__. Rows from CPython 3.14.
from dataclasses import dataclass

class S102K:
    def __init__(self, v):
        self.v = v
    def __hash__(self):
        return hash(self.v)
    def __eq__(self, o):
        return isinstance(o, S102K) and self.v == o.v

class S102Eq:
    def __eq__(self, o):
        return True

class S102Seen:
    def __init__(self, v, log):
        self.v = v
        self.log = log
    def __hash__(self):
        return self.v % 2
    def __eq__(self, o):
        self.log.append((self.v, o.v))
        return True

@dataclass(frozen=True)
class S102P:
    x: int
    y: int

def s102():
    d = {S102K(1): "one", S102K(2): "two"}
    d[S102K(1)] = "uno"
    check("ukey1", d[S102K(1)] == "uno" and len(d) == 2 and S102K(2) in d and S102K(3) not in d
          and d.get(S102K(2)) == "two" and d.get(S102K(9), "no") == "no")
    del d[S102K(2)]
    check("ukey2", d.pop(S102K(1)) == "uno" and len(d) == 0)
    s = {S102K(1), S102K(2), S102K(1)}
    s.add(S102K(2))
    s.discard(S102K(1))
    check("ukey3", len(s) == 1 and S102K(2) in s and S102K(1) not in s
          and sorted(k.v for k in {S102K(1), S102K(2)} | {S102K(2), S102K(3)}) == [1, 2, 3]
          and [k.v for k in {S102K(1), S102K(2)} & {S102K(2), S102K(3)}] == [2]
          and [k.v for k in {S102K(1), S102K(2)} - {S102K(2), S102K(3)}] == [1])
    check("ukey4", {S102P(1, 2): "p"}[S102P(1, 2)] == "p" and len({S102P(1, 2), S102P(1, 2)}) == 1
          and {(S102K(1), 2): "t"}[(S102K(1), 2)] == "t")
    check("ukey5", [S102K(1)] == [S102K(1)] and (S102K(1), 2) == (S102K(1), 2) and S102K(1) in [S102K(1)]
          and [S102K(0), S102K(1)].index(S102K(1)) == 1 and [S102K(1), S102K(1)].count(S102K(1)) == 2
          and {1: S102K(1)} == {1: S102K(1)})
    log = []
    s2 = {S102Seen(1, log), S102Seen(2, log)}
    found = S102Seen(3, log) in s2
    check("ukey6", len(s2) == 2 and found and log == [(1, 3)])
    msgs = []
    for f in [lambda: {S102Eq(): 1}, lambda: {S102Eq()}, lambda: {(1, S102Eq()): 2}, lambda: S102Eq() in {1: 2}]:
        try:
            f()
        except TypeError as err:
            msgs.append(str(err))
    check("ukey7", msgs == ["cannot use 'S102Eq' as a dict key (unhashable type: 'S102Eq')",
                            "cannot use 'S102Eq' as a set element (unhashable type: 'S102Eq')",
                            "cannot use 'tuple' as a dict key (unhashable type: 'S102Eq')",
                            "cannot use 'S102Eq' as a dict key (unhashable type: 'S102Eq')"])

# ===== SECTION 103: descriptors on the class, type(), function attributes, notes =====
# A descriptor read off the CLASS answers __get__(None, cls). type(name, bases,
# ns) builds the class a class statement would. A function stored as another
# function's attribute is called through it. BaseException.add_note appends to
# __notes__. An arity error names the function by its __qualname__, an assigned
# one included. SystemExit derives from BaseException, carries .code, and
# sys.exit raises it. Rows from CPython 3.14.
import sys

class S103D:
    def __get__(self, obj, typ=None):
        if obj is None:
            return ("cls", typ.__name__)
        return ("inst", type(obj).__name__)

class S103K:
    a = S103D()

class S103S(S103K):
    pass

class S103B:
    def hi(self):
        return "hi " + type(self).__name__

def s103():
    check("desc1", S103K.a == ("cls", "S103K") and S103S.a == ("cls", "S103S") and S103K().a == ("inst", "S103K")
          and getattr(S103K, "a") == ("cls", "S103K") and type(S103K.__dict__["a"]).__name__ == "S103D")
    Q = type("Q", (), {"a": 1})
    R = type("R", (S103B,), {"b": 2, "m": lambda self: self.b * 10})
    r = R()
    check("type3", Q.__name__ == "Q" and Q.a == 1 and Q().a == 1 and type(Q()) is Q and isinstance(Q(), Q)
          and r.b == 2 and r.m() == 20 and r.hi() == "hi R" and isinstance(r, S103B) and R.__mro__ == (R, S103B, object)
          and type(Q) is type and Q.__module__ == "__main__")
    try:
        type("Q", ())
        msg = "no error"
    except TypeError as err:
        msg = str(err)
    check("type3e", msg == "type() takes 1 or 3 arguments")

    def f():
        return 1
    def g(x):
        return x * 2
    f.g = g
    f.h = lambda a, b=4: a + b
    check("fnattr1", f.g(3) == 6 and f.h(1) == 5 and f.h(1, b=2) == 3)

    e = ValueError("bad")
    had = hasattr(e, "__notes__")
    e.add_note("first")
    e.add_note("second")
    try:
        e.add_note(3)
        nmsg = "no error"
    except TypeError as err:
        nmsg = str(err)
    check("note1", not had and e.__notes__ == ["first", "second"] and e.add_note("x") is None
          and nmsg == "add_note() argument must be str, not int")

    def ar(a, b):
        return a + b
    ar.__qualname__ = "Renamed.ar"
    msgs = []
    for c in [lambda: ar(1), lambda: ar(1, 2, 3), lambda: ar(1, c=2), lambda: (lambda x: x)()]:
        try:
            c()
        except TypeError as err:
            msgs.append(str(err))
    check("qual1", msgs == ["Renamed.ar() missing 1 required positional argument: 'b'",
                            "Renamed.ar() takes 2 positional arguments but 3 were given",
                            "Renamed.ar() got an unexpected keyword argument 'c'",
                            "s103.<locals>.<lambda>.<locals>.<lambda>() missing 1 required positional argument: 'x'"])

    rows = []
    for c in [lambda: sys.exit(3), lambda: sys.exit(), lambda: sys.exit(1, 2)]:
        try:
            c()
        except Exception as err:
            rows.append(str(err))
        except BaseException as err:
            rows.append((type(err).__name__, err.code, err.args))
    check("sysexit1", rows == [("SystemExit", 3, (3,)), ("SystemExit", None, ()), "exit expected at most 1 argument, got 2"]
          and SystemExit("a", "b").code == ("a", "b")
          and [c.__name__ for c in SystemExit.__mro__] == ["SystemExit", "BaseException", "object"])

# ===== SECTION 104: collections, builtin subclasses and bytearray =====
# deque, defaultdict, OrderedDict and Counter (lib/python-collections.py), their
# error texts included. A dict subclass's __missing__ answers an absent key, its
# str() goes through an overriding __repr__, and dict() and ** read it as a
# mapping. bytearray builds the same list of byte values bytes does here.
# Rows from CPython 3.14.
from collections import deque, defaultdict, OrderedDict, Counter
import collections

class S104D(dict):
    def __missing__(self, k):
        return k * 2
    def __repr__(self):
        return "S104D(" + dict.__repr__(self) + ")"

class S104I(int):
    def __repr__(self):
        return "I!"

def s104():
    d = S104D(a=1)
    check("bsub1", d["zz"] == "zzzz" and d.get("zz") is None and str(d) == "S104D({'a': 1})" and f"{d}" == "S104D({'a': 1})"
          and dict(d) == {"a": 1} and {**d} == {"a": 1} and str(S104I(3)) == "I!" and str(5) == "5" and (5).__str__() == "5")

    q = collections.deque(range(5))
    q.rotate(2)
    q.appendleft(-1)
    q.extendleft([7, 8])
    check("deque1", repr(q) == "deque([8, 7, -1, 3, 4, 0, 1, 2])" and q.pop() == 2 and q.popleft() == 8 and len(q) == 6
          and 3 in q and q[0] == 7 and q[-1] == 1 and list(reversed(q)) == [1, 0, 4, 3, -1, 7]
          and repr(deque(range(10), maxlen=3)) == "deque([7, 8, 9], maxlen=3)" and deque([1, 2]) == deque([1, 2]))
    errs = []
    for f in [lambda: deque([1]).index(5), lambda: deque([1]).remove(5), lambda: deque().pop(), lambda: deque().popleft(),
              lambda: deque([1])[5], lambda: deque([1], maxlen=1).insert(0, 2), lambda: OrderedDict().popitem(),
              lambda: defaultdict(None)["x"], lambda: defaultdict(3)]:
        try:
            f()
        except (ValueError, IndexError, KeyError, TypeError) as err:
            errs.append(type(err).__name__ + " " + repr(err))
    check("coll2", errs == ["ValueError ValueError('deque.index(x): x not in deque')",
                            "ValueError ValueError('deque.remove(x): x not in deque')",
                            "IndexError IndexError('pop from an empty deque')",
                            "IndexError IndexError('pop from an empty deque')",
                            "IndexError IndexError('deque index out of range')",
                            "IndexError IndexError('deque already at its maximum size')",
                            "KeyError KeyError('dictionary is empty')", "KeyError KeyError('x')",
                            "TypeError TypeError('first argument must be callable or None')"])
    g = defaultdict(list)
    for w in ["apple", "bob", "avocado"]:
        g[w[0]].append(w)
    cnt = defaultdict(int)
    for ch in "mississippi":
        cnt[ch] += 1
    check("ddict1", repr(g) == "defaultdict(<class 'list'>, {'a': ['apple', 'avocado'], 'b': ['bob']})"
          and g["zz"] == [] and len(g) == 3 and isinstance(g, dict) and dict(cnt) == {"m": 1, "i": 4, "s": 4, "p": 2}
          and cnt.get("q") is None and "q" not in cnt)
    o = OrderedDict([("x", 1), ("y", 2), ("z", 3)])
    o.move_to_end("x")
    o.move_to_end("z", last=False)
    check("odict1", repr(o) == "OrderedDict({'z': 3, 'y': 2, 'x': 1})" and o.popitem() == ("x", 1)
          and o.popitem(last=False) == ("z", 3) and list(o) == ["y"] and repr(OrderedDict()) == "OrderedDict()"
          and OrderedDict(a=1, b=2) != OrderedDict(b=2, a=1) and OrderedDict(a=1, b=2) == {"b": 2, "a": 1})
    c = Counter("hello world")
    c.update("lol")
    c.subtract({"l": 2})
    check("counter1", c.most_common(2) == [("l", 3), ("o", 3)] and c["zzz"] == 0 and c.total() == 12
          and repr(Counter("abca")) == "Counter({'a': 2, 'b': 1, 'c': 1})" and repr(Counter()) == "Counter()"
          and Counter(a=1) == Counter(a=1, b=0) and list(Counter("aab").elements()) == ["a", "a", "b"])
    check("counter2", Counter(a=3, b=1) - Counter(a=1, b=2) == Counter(a=2) and Counter(a=3) + Counter(a=1, b=2) == Counter(a=4, b=2)
          and Counter(a=3, b=1) | Counter(a=1, b=2) == Counter(a=3, b=2) and Counter(a=3, b=1) & Counter(a=1, b=2) == Counter(a=1, b=1)
          and +Counter(a=1, b=-2) == Counter(a=1) and -Counter(a=1, b=-2) == Counter(b=2))

    ba = bytearray(b"ab")
    ba[0] = 65
    ba.append(67)
    bmsgs = []
    for f in [lambda: bytearray(1, 2, 3, 4), lambda: bytearray(None), lambda: bytearray([300]), lambda: bytes([300])]:
        try:
            f()
        except (TypeError, ValueError) as err:
            bmsgs.append(str(err))
    check("barr1", list(ba) == [65, 98, 67] and len(bytearray(3)) == 3 and list(bytearray("hé", "utf-8")) == [104, 195, 169]
          and bmsgs == ["bytearray() takes at most 3 arguments (4 given)", "cannot convert 'NoneType' object to bytearray",
                        "byte must be in range(0, 256)", "bytes must be in range(0, 256)"])

# ===== SECTION 105: aliased and module-qualified imports =====
# `import M as A` binds A to the module object and `from M import N as A` binds
# A to its attribute N, at the point the import runs - a local in a function.
# copy.copy, copy.replace and copy.Error are the module's own names, and a name
# the module lacks is an ImportError. Rows from CPython 3.14.
from dataclasses import dataclass as S105dc, field as S105field
import collections as S105co

@S105dc
class S105P:
    a: int
    b: list = S105field(default_factory=list)

def s105():
    import copy as cp
    from collections import Counter as C, deque as D
    from dataclasses import replace as rep, fields as flds, asdict
    import dataclasses
    import collections
    p = S105P(1, [2])
    check("impas1", repr(p) == "S105P(a=1, b=[2])" and cp.deepcopy(p) == p and cp.deepcopy(p).b is not p.b
          and cp.copy(p).b is p.b and rep(p, a=5) == S105P(5, [2]) and [f.name for f in flds(p)] == ["a", "b"]
          and asdict(p) == {"a": 1, "b": [2]} and dataclasses.is_dataclass(p) and dataclasses.astuple(p) == (1, [2]))
    check("impas2", C("abca").most_common(1) == [("a", 2)] and repr(D([1, 2, 3], maxlen=2)) == "deque([2, 3], maxlen=2)"
          and repr(S105co.OrderedDict(a=1)) == "OrderedDict({'a': 1})" and collections.Counter("xx")["x"] == 2
          and repr(collections.deque([4])) == "deque([4])" and collections.defaultdict(int)["q"] == 0)
    from copy import copy, Error, replace
    errs = []
    try:
        replace(1)
    except TypeError as err:
        errs.append(str(err))
    try:
        from copy import S105nosuch
    except ImportError as err:
        errs.append(str(err).split(" (")[0])
    try:
        from collections import S105Nope as N
    except ImportError as err:
        errs.append(type(err).__name__ + " " + str(err).split(" (")[0])
    check("impas3", copy([1, 2]) == [1, 2] and type(copy).__name__ == "function" and issubclass(Error, Exception)
          and Error.__name__ == "Error" and errs == ["replace() does not support int objects", "cannot import name 'S105nosuch' from 'copy'",
                       "ImportError cannot import name 'S105Nope' from 'collections'"])

# ===== SECTION 106: aliased imports of the runtime-provided modules =====
# math, re, sys, builtins and the other stdlib modules the runtime provides bind
# a module object like any other: `import M as A` binds A to it, `from M import N
# as A` binds A to its attribute, and a module function read off the module is a
# first-class value. An unbound name counts as a failed row, not an abort.
# Rows from CPython 3.14.
import math as S106m
from typing import List as S106List

def S106first(xs: S106List[int]) -> int:
    return xs[0]

def S106chk(name, fn):
    try:
        ok = fn()
    except (NameError, AttributeError):
        ok = False
    check(name, ok)

def s106():
    import math
    import re as r
    import sys as system
    import builtins as bi
    import json, os.path
    import json as j2
    from math import sqrt as sq, floor, pi
    from re import sub, findall as fa, IGNORECASE, compile as rc
    from sys import get_int_max_str_digits as gimd
    from builtins import abs as ab
    xs = [2.0, 9.0, 0.25]
    S106chk("impst1", lambda: S106m.sqrt(16) == 4.0 and S106m is math and type(S106m).__name__ == "module"
            and sq(25) == 5.0 and floor(2.5) == 2 and pi == 3.141592653589793
            and [S106m.sqrt(x) for x in xs] == [2.0 ** 0.5, 3.0, 0.5] and S106m.sqrt.__name__ == "sqrt")
    S106chk("impst2", lambda: r.sub("a", "b", "aaa") == "bbb" and r.I == r.IGNORECASE == IGNORECASE
            and sub("x", "y", "xx") == "yy" and fa("[a-z]", "a1b") == ["a", "b"]
            and rc("ab").search("zab").group(0) == "ab" and hasattr(r, "sub") and not hasattr(r, "S106nosuch")
            and hasattr(system, "exit"))
    S106chk("impst3", lambda: system.get_int_max_str_digits() == 4300 and gimd() == 4300 and bi.len([1, 2]) == 2
            and bi.len is len and ab(-3) == 3)
    def inner():
        import math as S106mm
        return S106mm.floor(9.5)
    errs = []
    try:
        S106mm
    except NameError as e:
        errs.append(str(e))
    S106chk("impst4", lambda: type(json).__name__ == "module" and type(os).__name__ == "module" and j2 is json
            and S106first([7, 8]) == 7 and inner() == 9 and errs == ["name 'S106mm' is not defined"])

# ===== SECTION 107: the json module =====
# dumps with indent, separators, sort_keys, ensure_ascii, allow_nan, skipkeys
# and default, and loads with its hooks and its JSONDecodeError positions
# (lib/python-json.py). Rows from CPython 3.14.
import json

def S107err(f):
    try:
        f()
    except ValueError as e:
        return type(e).__name__ + ": " + str(e)
    except TypeError as e:
        return "TypeError: " + str(e)
    return "no error"

def s107():
    v = [{"b": 1, "a": [1, 2.5, None, True]}, "x\"\n\u00e9\U0001f600", (), {}]
    check("json1", json.dumps(v) == '[{"b": 1, "a": [1, 2.5, null, true]}, "x\\"\\n\\u00e9\\ud83d\\ude00", [], {}]'
          and json.dumps(v, ensure_ascii=False) == '[{"b": 1, "a": [1, 2.5, null, true]}, "x\\"\\n\u00e9\U0001f600", [], {}]'
          and json.dumps(v[0], separators=(",", ":"), sort_keys=True) == '{"a":[1,2.5,null,true],"b":1}'
          and json.dumps({1: 2, 2.5: 3, False: None}) == '{"1": 2, "2.5": 3, "false": null}'
          and json.dumps([float("inf"), float("nan"), 1e20, -0.0]) == "[Infinity, NaN, 1e+20, -0.0]")
    check("json2", json.dumps(v[0], indent=2, sort_keys=True) == '{\n  "a": [\n    1,\n    2.5,\n    null,\n    true\n  ],\n  "b": 1\n}'
          and json.dumps([[], {"k": {}}], indent="\t") == '[\n\t[],\n\t{\n\t\t"k": {}\n\t}\n]'
          and json.dumps([1], indent=0) == "[\n1\n]")
    r = []
    r.append(r)
    check("json3", [S107err(lambda: json.dumps(object())), S107err(lambda: json.dumps({(1,): 2})),
                    S107err(lambda: json.dumps(float("nan"), allow_nan=False)), S107err(lambda: json.dumps(r))]
          == ["TypeError: Object of type object is not JSON serializable",
              "TypeError: keys must be str, int, float, bool or None, not tuple",
              "ValueError: Out of range float values are not JSON compliant: nan",
              "ValueError: Circular reference detected"]
          and json.dumps({(1,): 2, "a": 1}, skipkeys=True) == '{"a": 1}'
          and json.dumps([object()], default=lambda o: "obj") == '["obj"]')
    d = json.loads(' {"a": [1, -2.5e1, "\\u00e9\\ud83d\\ude00\\n", true, false, null], "b": {}} ')
    check("json4", d == {"a": [1, -25.0, "\u00e9\U0001f600\n", True, False, None], "b": {}}
          and type(d["a"][0]) is int and type(d["a"][1]) is float and json.loads("[NaN, -Infinity]")[1] == float("-inf")
          and json.loads('{"a": 1, "b": 2}', object_pairs_hook=lambda p: p) == [("a", 1), ("b", 2)]
          and json.loads('{"a": 1}', object_hook=lambda o: list(o)) == ["a"]
          and json.loads("[1, 2.5]", parse_int=lambda s: "i" + s, parse_float=lambda s: "f" + s) == ["i1", "f2.5"]
          and json.loads(json.dumps(v)) == [{"b": 1, "a": [1, 2.5, None, True]}, "x\"\n\u00e9\U0001f600", [], {}])
    bad = ["", "[1,]", '{"a" 1}', "{a: 1}", "[1 2]", '"abc', '"a\\qb"', "01", '{"a": 1}\n  x', "[\n1,\n\n2,\n]"]
    check("json5", [S107err(lambda: json.loads(t)) for t in bad] == [
        "JSONDecodeError: Expecting value: line 1 column 1 (char 0)",
        "JSONDecodeError: Illegal trailing comma before end of array: line 1 column 3 (char 2)",
        "JSONDecodeError: Expecting ':' delimiter: line 1 column 6 (char 5)",
        "JSONDecodeError: Expecting property name enclosed in double quotes: line 1 column 2 (char 1)",
        "JSONDecodeError: Expecting ',' delimiter: line 1 column 4 (char 3)",
        "JSONDecodeError: Unterminated string starting at: line 1 column 1 (char 0)",
        "JSONDecodeError: Invalid \\escape: line 1 column 3 (char 2)",
        "JSONDecodeError: Extra data: line 1 column 2 (char 1)",
        "JSONDecodeError: Extra data: line 2 column 3 (char 11)",
        "JSONDecodeError: Illegal trailing comma before end of array: line 4 column 2 (char 7)"])
    try:
        json.loads("[1,\n x]")
    except json.JSONDecodeError as e:
        ex = [e.msg, e.pos, e.lineno, e.colno, isinstance(e, ValueError)]
    check("json6", ex == ["Expecting value", 5, 2, 2, True] and type(json).__name__ == "module"
          and json.dumps.__name__ == "dumps" and hasattr(json, "loads") and not hasattr(json, "S107nope"))

# ===== SECTION 108: the itertools module =====
# Every tool of Modules/itertoolsmodule.c but the lazy-input ones' laziness
# (lib/python-itertools.py): count, cycle and repeat are endless, and islice
# and takewhile read an endless input only as far as CPython does. Rows from
# CPython 3.14.
import itertools
from itertools import islice as S108islice, count as S108count, chain as S108chain

def S108verr(f):
    try:
        list(f())
    except ValueError as e:
        return str(e)
    return "no error"

def s108():
    it = itertools
    xs = [3, 1, 4, 1, 5, 9, 2, 6]
    check("itt1", list(S108islice(S108count(), 4)) == [0, 1, 2, 3] and list(it.islice(it.count(2.5, -0.5), 3)) == [2.5, 2.0, 1.5]
          and list(it.islice(it.cycle("ab"), 5)) == ["a", "b", "a", "b", "a"] and list(it.repeat(7, 2)) == [7, 7]
          and list(it.repeat(1, -1)) == [] and list(it.takewhile(lambda n: n < 4, it.count(1))) == [1, 2, 3]
          and list(it.islice(xs, 1, 7, 2)) == [1, 1, 9] and list(it.islice(xs, 6, None)) == [2, 6])
    src = iter(range(10))
    check("itt2", list(it.islice(src, 2, 6, 3)) == [2, 5] and next(src) == 6
          and [S108verr(lambda: it.islice(xs, -1)), S108verr(lambda: it.islice(xs, 0, 5, 0)),
               S108verr(lambda: it.combinations("a", -1)), S108verr(lambda: it.batched("a", 0))]
          == ["Stop argument for islice() must be None or an integer: 0 <= x <= sys.maxsize.",
              "Step for islice() must be a positive integer or None.", "r must be non-negative", "n must be at least one"])
    check("itt3", list(it.accumulate(xs, max)) == [3, 3, 4, 4, 5, 9, 9, 9] and list(it.accumulate([1, 2], initial=10)) == [10, 11, 13]
          and list(it.batched("abcde", 2)) == [("a", "b"), ("c", "d"), ("e",)] and list(S108chain("ab", [1])) == ["a", "b", 1]
          and list(it.chain.from_iterable(["xy", "z"])) == ["x", "y", "z"] and list(it.compress("abc", [1, 0, 1])) == ["a", "c"]
          and list(it.dropwhile(lambda n: n < 5, xs)) == [5, 9, 2, 6] and list(it.filterfalse(None, [0, 1, "", "a"])) == [0, ""]
          and list(it.pairwise("abc")) == [("a", "b"), ("b", "c")] and list(it.starmap(pow, [(2, 3)])) == [8])
    check("itt4", [(k, list(g)) for k, g in it.groupby("aabccc")] == [("a", ["a", "a"]), ("b", ["b"]), ("c", ["c", "c", "c"])]
          and [k for k, g in it.groupby(xs, key=lambda n: n % 2)] == [1, 0, 1, 0]
          and list(it.zip_longest("ab", [1], fillvalue="-")) == [("a", 1), ("b", "-")]
          and [list(t) for t in it.tee(iter([1, 2]), 3)] == [[1, 2], [1, 2], [1, 2]])
    check("itt5", list(it.product("ab", repeat=2)) == [("a", "a"), ("a", "b"), ("b", "a"), ("b", "b")]
          and list(it.product([1, 2], "x")) == [(1, "x"), (2, "x")] and list(it.product()) == [()]
          and list(it.permutations("abc", 2)) == [("a", "b"), ("a", "c"), ("b", "a"), ("b", "c"), ("c", "a"), ("c", "b")]
          and len(list(it.permutations(range(5)))) == 120 and list(it.permutations("ab", 3)) == []
          and list(it.combinations("abcd", 3)) == [("a", "b", "c"), ("a", "b", "d"), ("a", "c", "d"), ("b", "c", "d")]
          and list(it.combinations_with_replacement("ab", 2)) == [("a", "a"), ("a", "b"), ("b", "b")]
          and type(it).__name__ == "module" and it.chain.__name__ == "chain")

# ===== SECTION 109: the functools module =====
# reduce, partial, lru_cache / cache with cache_info, wraps / update_wrapper,
# cmp_to_key and total_ordering (lib/python-functools.py). Rows from CPython
# 3.14.
import functools
from functools import reduce as S109reduce, partial, lru_cache, wraps

def S109f(a, b, c=0, *, d=1):
    return (a, b, c, d)

def S109cmp(a, b):
    return (a > b) - (a < b)

@functools.total_ordering
class S109V:
    def __init__(self, v):
        self.v = v
    def __eq__(self, o):
        return self.v == o.v
    def __lt__(self, o):
        return self.v < o.v

def s109():
    errs = []
    try:
        S109reduce(lambda a, b: a, [])
    except TypeError as e:
        errs.append(str(e))
    check("ftl1", S109reduce(lambda a, b: a * b, [1, 2, 3, 4]) == 24 and functools.reduce(max, [3, 9], 100) == 100
          and S109reduce(lambda a, b: a - b, [1, 2], initial=10) == 7 and errs == ["reduce() of empty iterable with no initial value"])
    p = partial(S109f, 1, d=5)
    pp = partial(p, 7, c=8)
    check("ftl2", p(2) == (1, 2, 0, 5) and p(2, d=9) == (1, 2, 0, 9) and p.args == (1,) and p.keywords == {"d": 5}
          and p.func is S109f and pp() == (1, 7, 8, 5) and pp.func is S109f and type(p).__name__ == "partial")
    calls = []
    @lru_cache(maxsize=2)
    def sq(n):
        calls.append(n)
        return n * n
    vals = [sq(x) for x in [1, 2, 1, 3, 2, 1]]
    info = sq.cache_info()
    check("ftl3", vals == [1, 4, 1, 9, 4, 1] and calls == [1, 2, 3, 2, 1] and repr(info) == "CacheInfo(hits=1, misses=5, maxsize=2, currsize=2)"
          and info.hits == 1 and info == (1, 5, 2, 2) and sq.__name__ == "sq")
    @functools.cache
    def fib(n):
        return n if n < 2 else fib(n - 1) + fib(n - 2)
    def deco(fn):
        @wraps(fn)
        def inner(*a):
            return fn(*a) + 1
        return inner
    @deco
    def g(x):
        return x
    check("ftl4", fib(60) == 1548008755920 and fib.cache_info().misses == 61 and g(1) == 2 and g.__name__ == "g"
          and g.__wrapped__(1) == 1)
    K = functools.cmp_to_key(S109cmp)
    check("ftl5", sorted([3, 1, 2], key=K) == [1, 2, 3] and sorted("bca", key=functools.cmp_to_key(lambda a, b: S109cmp(b, a))) == ["c", "b", "a"]
          and K(1) < K(2) and K(2) >= K(2) and S109V(1) <= S109V(1) and S109V(2) > S109V(1) and not S109V(2) >= S109V(3))

# ===== SECTION 110: the string module and os.path =====
# string's constants, capwords and Template (lib/python-string.py), and
# posixpath's string arithmetic as os.path (lib/python-os.py). Rows from
# CPython 3.14.
import string
import os.path
from os.path import join as S110join

def s110():
    st = string
    check("strm1", st.ascii_letters == st.ascii_lowercase + st.ascii_uppercase and st.digits == "0123456789"
          and st.hexdigits == "0123456789abcdefABCDEF" and st.octdigits == "01234567" and len(st.printable) == 100
          and st.punctuation == "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~" and st.whitespace == " \t\n\r\x0b\x0c"
          and st.capwords(" hello   wORLD ") == "Hello World" and st.capwords("a-b", "-") == "A-B")
    t = st.Template("$who likes ${what}; $$5")
    errs = []
    for f in [lambda: st.Template("cost $5").substitute(), lambda: st.Template("a\nb ${x").substitute()]:
        try:
            f()
        except ValueError as e:
            errs.append(str(e))
    try:
        t.substitute(who="x")
    except KeyError as e:
        errs.append(repr(e))
    check("strm2", t.substitute(who="tim", what="tea") == "tim likes tea; $5" and t.safe_substitute(who="x") == "x likes ${what}; $5"
          and t.substitute({"who": "a", "what": "b"}, who="z") == "z likes b; $5" and t.get_identifiers() == ["who", "what"]
          and errs == ["Invalid placeholder in string: line 1, col 6", "Invalid placeholder in string: line 2, col 3", "KeyError('what')"])
    op = os.path
    check("osp1", S110join("a", "b", "c") == "a/b/c" and op.join("/a", "/b", "c") == "/b/c" and op.join("a/", "b") == "a/b"
          and op.split("/a/b.c") == ("/a", "b.c") and op.basename("a/b/") == "" and op.dirname("//x//y") == "//x"
          and op.splitext("a/b.tar.gz") == ("a/b.tar", ".gz") and op.splitext(".cshrc") == (".cshrc", "")
          and op.normpath("./a/./b/../../c") == "c" and op.normpath("//a/b/..") == "//a" and op.normpath("") == "."
          and op.isabs("/x") and not op.isabs("x") and op.commonprefix(["/usr/lib", "/usr/local"]) == "/usr/l")
    errs = []
    try:
        op.basename(1)
    except TypeError as e:
        errs.append(str(e))
    check("osp2", errs == ["expected str, bytes or os.PathLike object, not int"] and os.sep == "/" and os.path.sep == "/"
          and os.name == "posix" and op.__name__ == "posixpath" and type(os).__name__ == "module")

# ===== SECTION 111: a module is unbound until imported =====
# math and builtins are imported nowhere at this file's top level, so reading
# either name is a NameError, as it is in CPython; an import inside a function
# binds a local. Rows from CPython 3.14.
def S111name(f):
    try:
        f()
    except NameError as e:
        return str(e)
    return "bound"

def s111():
    def inner():
        import math
        return math.floor(2.5)
    check("modvis1", [S111name(lambda: math.sqrt(4)), S111name(lambda: builtins.len), inner(), S111name(lambda: math)]
          == ["name 'math' is not defined", "name 'builtins' is not defined", 2, "name 'math' is not defined"])

# ===== SECTION 112: the random module =====
# The module (lib/python-random.py) and the argument checks Lib/random.py makes
# before it draws. No row draws: seeding the Mersenne Twister alone costs the
# -frozen interpreter leg most of its step budget, so the seeded sequences are
# pinned by a differential probe against CPython instead. Rows from CPython 3.14.
import random
from random import Random as S112Random

def s112():
    errs = []
    for f in [lambda: random.randrange(5, 5), lambda: random.randrange(0), lambda: random.choice([]),
              lambda: random.sample([1], 2), lambda: random.randrange(1, 9, 0), lambda: random.getrandbits(-1)]:
        try:
            f()
        except (ValueError, IndexError) as e:
            errs.append(type(e).__name__ + ": " + str(e))
    check("rnd1", errs == ["ValueError: empty range in randrange(5, 5)", "ValueError: empty range for randrange()",
                           "IndexError: Cannot choose from an empty sequence",
                           "ValueError: Sample larger than population or is negative", "ValueError: zero step for randrange()",
                           "ValueError: Cannot convert negative int"]
          and random.getrandbits(0) == 0 and type(random).__name__ == "module" and random.randint.__name__ == "randint"
          and S112Random.__name__ == "Random")

# ===== SECTION 113: one-shot sources in generators, user iterators, descriptor methods, docstrings =====
# A generator stepping an iterator made OUTSIDE it pulls each value once, even
# interleaved with the driver's own pulls and through a generator pipeline; a
# user class with __next__ is an iterator to next(), and a for loop or a
# `yield from` steps it one __next__ at a time, so an endless one stops at a
# break; a descriptor instance on a class is called through its __get__; a def's
# or a class's leading string is its __doc__ (cleaned as CPython 3.13+ does);
# a lone surrogate reprs as an escape. Rows from CPython 3.14.
from functools import lru_cache as s113_lru

def s113_g(it):
    for x in it:
        yield x

def s113_dbl(it):
    for y in s113_g(it):
        yield y * 2

def s113_pairs(it):
    for a in it:
        yield (a, next(it, None))

def s113_rest(it):
    yield next(it)
    yield list(it)

class S113Count:
    def __init__(self, k):
        self.n = 0
        self.k = k
    def __iter__(self):
        return self
    def __next__(self):
        self.n += 1
        if self.n > self.k:
            raise StopIteration
        return self.n

class S113Endless:
    def __init__(self):
        self.i = 0
    def __iter__(self):
        return self
    def __next__(self):
        self.i += 1
        return self.i

class S113Bag:
    def __iter__(self):
        return S113Count(2)

class S113Bad:
    def __iter__(self):
        return [1]

class S113Stop:
    def __next__(self):
        raise StopIteration("why")

def s113_loop_bad():
    for v in S113Bad():
        pass

def s113_yf(src):
    yield from src
    yield "end"

class S113Desc:
    def __init__(self, f):
        self.f = f
    def __get__(self, obj, typ=None):
        if obj is None:
            return self
        def bound(*a):
            return ("bound", self.f(obj, *a))
        return bound

class S113A:
    def __init__(self, k):
        self.k = k
    @S113Desc
    def m(self, x):
        return self.k + x
    @s113_lru(maxsize=None)
    def sq(self, n):
        return n * n + self.k

def s113_doc():
    # a comment comes first
    """Doc line.

    Indented line."""
    return 1

def s113_nodoc():
    x = 1
    "not a docstring"
    return x

def s113_inline(): "inline"; return 2

class S113Base:
    "Base doc"
    def m(self):
        "m doc"

class S113Sub(S113Base):
    def m(self):
        pass

class S113Shadow(S113Base):
    m = None

def s113():
    src = iter(range(10))
    gg = s113_g(src)
    seq = [next(gg), next(src), next(gg), next(src), next(gg)]
    check("it1", list(s113_g(iter([5, 6, 7, 8]))) == [5, 6, 7, 8]
          and list(s113_dbl(iter([1, 2, 3]))) == [2, 4, 6] and seq == [0, 1, 2, 3, 4])
    check("it2", list(s113_pairs(iter([1, 2, 3, 4, 5]))) == [(1, 2), (3, 4), (5, None)]
          and list(s113_rest(iter("abc"))) == ["a", ["b", "c"]])
    c = S113Count(4)
    check("it3", list(s113_g(c)) == [1, 2, 3, 4] and c.n == 5)
    c2 = S113Count(3)
    first = [next(c2), next(c2)]
    check("it4", first == [1, 2] and list(c2) == [3] and next(c2, "d") == "d" and iter(c2) is c2)
    stop = []
    try:
        next(S113Stop())
    except StopIteration as e:
        stop.append(e.args)
    check("it5", stop == [("why",)])
    out = []
    for v in S113Endless():
        if v > 3:
            break
        out.append(v)
    e = S113Endless()
    pre = [next(e), next(e)]
    for v in e:
        out.append(v)
        break
    check("it6", out == [1, 2, 3, 3] and pre == [1, 2] and next(e) == 4)
    nest = []
    for a in S113Count(2):
        for b in S113Bag():
            nest.append(a * 10 + b)
    check("it7", nest == [11, 12, 21, 22] and list(s113_yf(S113Count(2))) == [1, 2, "end"])
    errs = []
    for f in [lambda: [v for v in S113Bad()], s113_loop_bad, lambda: next(5)]:
        try:
            f()
        except TypeError as te:
            errs.append(str(te))
    check("it8", errs == ["iter() returned non-iterator of type 'list'", "iter() returned non-iterator of type 'list'",
                          "'int' object is not an iterator"])
    a = S113A(10)
    f = a.m
    check("desc1", a.m(5) == ("bound", 15) and f(6) == ("bound", 16)
          and a.sq(3) == 19 and a.sq(3) == 19 and S113A.sq(a, 5) == 35)
    check("doc1", s113_doc.__doc__ == "Doc line.\n\nIndented line." and s113_nodoc.__doc__ is None
          and s113_inline.__doc__ == "inline" and (lambda: 0).__doc__ is None)
    check("doc2", S113Base.__doc__ == "Base doc" and S113Sub.__doc__ is None and S113Sub().__doc__ is None
          and S113Base().m.__doc__ == "m doc" and S113Sub().m.__doc__ is None and S113Base.m.__doc__ == "m doc")
    check("doc3", S113Shadow.m is None and S113Shadow().m is None)
    check("sur1", repr(chr(0xd800)) == "'\\ud800'" and repr("a" + chr(0xdfff) + "b") == "'a\\udfffb'"
          and len(chr(0xdc00)) == 1 and ord(chr(0xdbff)) == 0xdbff)

# ===== SECTION 114: Unicode printability in repr and str.isprintable =====
# A code point is printable unless its category is Cc, Cf, Cs, Co, Cn, Zl, Zp,
# or Zs other than U+0020; repr escapes every other one, so an UNASSIGNED code
# point (Cn) escapes too, and str.isprintable asks the same question of every
# code point ("" is True). The code points come out of an array so no constant
# folder answers. Rows from CPython 3.14 (Unicode 16.0.0).
def s114():
    cps = [0xd7ff, 0x378, 0x1fffe, 0xe0100, 0x1f600, 0x20ac, 0xa0, 0xd800, 0x10ffff, 0x30000, 0x2fa1e, 0x9]
    ch = [chr(c) for c in cps]
    check("upr1", repr(ch[0]) == "'\\ud7ff'" and repr(ch[1]) == "'\\u0378'"
          and repr(ch[2]) == "'\\U0001fffe'" and repr(ch[8]) == "'\\U0010ffff'"
          and repr(ch[10]) == "'\\U0002fa1e'")
    check("upr2", repr(ch[3]) == "'" + ch[3] + "'" and repr(ch[4]) == "'" + ch[4] + "'"
          and repr(ch[9]) == "'" + ch[9] + "'")
    check("upr3", repr("a" + ch[1] + "é" + ch[2]) == "'a\\u0378é\\U0001fffe'")
    check("upr4", "".isprintable() and "a b".isprintable() and (ch[4] + ch[5] + ch[3]).isprintable()
          and ch[9].isprintable())
    check("upr5", not ch[0].isprintable() and not ch[1].isprintable() and not ch[6].isprintable()
          and not ch[7].isprintable() and not ch[8].isprintable() and not ("ok" + ch[11]).isprintable()
          and not ("ok" + ch[2]).isprintable())

# ===== SECTION 115: any/all/max/min/sum step their argument =====
# The five builtins pull one item at a time, so any/all over an endless source
# stop at the first deciding item, what they leave of an iterator is still
# there, and a key function sees the items in order. Rows from CPython 3.14.
from itertools import count as S115count

def S115gen():
    i = 0
    while True:
        yield i
        i += 1

def s115():
    check("stp1", any(x > 2 for x in S115count()) and not all(x < 3 for x in S115count())
          and any(x == 5 for x in S115gen()) and not all(x < 5 for x in S115gen()))
    it = iter([1, 0, 3, 4])
    r = all(it)
    it2 = iter([0, 0, 3, 4])
    r2 = any(it2)
    check("stp2", not r and list(it) == [3, 4] and r2 and list(it2) == [4])
    seen = []
    def key(v):
        seen.append(v)
        return -v
    check("stp3", max((x for x in [3, 1, 2]), key=key) == 1 and seen == [3, 1, 2]
          and min(iter([]), default="d") == "d" and sum(x * 2 for x in range(4)) == 12
          and max(3, 9, 2) == 9 and min(iter([5, 4, 6])) == 4)

# ===== SECTION 116: divmod dispatch, and a dict iterator over a changing dict =====
# divmod() asks __divmod__ / __rdivmod__ and otherwise serves two real numbers
# only, so __floordiv__ and __mod__ alone are not enough. A dict iterator walks
# the dict's entries as they are when reached: a key deleted before it is
# reached is skipped and one added is reached, with no error unless more items
# come out than the dict held. Rows from CPython 3.14.
class S116FM:
    def __floordiv__(self, o): return 1
    def __mod__(self, o): return 2

class S116DM:
    def __divmod__(self, o): return ("dm", o)
    def __rdivmod__(self, o): return ("rdm", o)

def S116err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

def S116walk(view, at, drop, add):
    d = {"a": 1, "b": 2, "c": 3}
    out = []
    try:
        for x in (d if view == "k" else d.items()):
            out.append(x)
            if len(out) == at:
                d.pop(drop)
                d[add] = 9
                d["a" if drop != "a" else "c"] = 7
    except RuntimeError as e:
        out.append(str(e))
    return out

def s116():
    vals = ["a", 1, 2.5, None]
    check("dvm1", S116err(lambda: divmod(vals[0], vals[1])) == "TypeError: unsupported operand type(s) for divmod(): 'str' and 'int'"
          and S116err(lambda: divmod(vals[2], vals[3])) == "TypeError: unsupported operand type(s) for divmod(): 'float' and 'NoneType'"
          and S116err(lambda: divmod(S116FM(), vals[1])) == "TypeError: unsupported operand type(s) for divmod(): 'S116FM' and 'int'"
          and S116err(lambda: divmod(1j, vals[1])) == "TypeError: unsupported operand type(s) for divmod(): 'complex' and 'int'")
    check("dvm2", divmod(S116DM(), 3) == ("dm", 3) and divmod(3, S116DM()) == ("rdm", 3)
          and divmod(-7, vals[1] + 1) == (-4, 1) and divmod(7.5, 2) == (3.0, 1.5))
    check("dvm3", S116walk("k", 1, "b", "z") == ["a", "c", "z"]
          and S116walk("k", 2, "a", "z") == ["a", "b", "c", "dictionary keys changed during iteration"]
          and S116walk("k", 1, "c", "z") == ["a", "b", "z"])
    check("dvm4", S116walk("i", 1, "b", "z") == [("a", 1), ("c", 3), ("z", 9)]
          and S116walk("i", 2, "c", "y") == [("a", 1), ("b", 2), ("y", 9)]
          and S116walk("i", 1, "a", "z") == [("a", 1), ("b", 2), ("c", 7), "dictionary keys changed during iteration"])

# ===== SECTION 117: str.maketrans, translate over any mapping, isidentifier, __import__ =====
# str.maketrans is a static method answering a dict of code points; translate
# subscripts any table, a LookupError keeping the character; isidentifier asks
# Unicode's XID_Start / XID_Continue, not printability; __import__ answers the
# top-level module (the named one with a fromlist). Rows from CPython 3.14.
def S117err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

def s117():
    ab = ["ab", "xy", "c"]
    t = str.maketrans(ab[0], ab[1], ab[2])
    check("mkt1", t == {97: 120, 98: 121, 99: None} and "abcab".translate(t) == "xyxy"
          and "q".maketrans(ab[0], ab[1]) == {97: 120, 98: 121}
          and str.maketrans({"é": "e", 98: None}) == {233: "e", 98: None})
    check("mkt2", S117err(lambda: str.maketrans(ab[0], ab[2])) == "ValueError: the first two maketrans arguments must have equal length"
          and S117err(lambda: str.maketrans(ab)) == "TypeError: if you give only one argument to maketrans it must be a dict"
          and S117err(lambda: str.maketrans({ab[0]: 1})) == "ValueError: string keys in translatetable must be of length 1"
          and S117err(lambda: str.maketrans(1, ab[0])) == "TypeError: first maketrans argument must be a string if there is a second argument")
    tbl = [str(i) for i in range(100)]
    tbl[97] = "A"
    class Up:
        def __getitem__(self, k):
            if k == 98:
                raise KeyError(k)
            return chr(k).upper()
    check("mkt3", "abc".translate(tbl) == "A9899" and "abc".translate(Up()) == "AbC"
          and "aé".translate("xyz" * 80) == "yz"
          and S117err(lambda: "a".translate(5)) == "TypeError: 'int' object is not subscriptable")
    ids = ["_x1", "é", "\U0001f600", "a\U0001f600", "x·", "·x", "a₁", "Ⅷ", "á", "x‍"]
    check("xid1", [s.isidentifier() for s in ids] == [True, True, False, False, True, False, False, True, True, True])
    names = ["math", "os.path", "nope.x"]
    check("imp1", __import__(names[0]).floor(2.5) == 2 and __import__(names[1]).path.join("a", "b") == "a/b"
          and __import__(names[1], fromlist=["join"]).join("c", "d") == "c/d"
          and S117err(lambda: __import__(names[2])) == "ModuleNotFoundError: No module named 'nope'"
          and S117err(lambda: __import__("os.nope")) == "ModuleNotFoundError: No module named 'os.nope'; 'os' is not a package"
          and issubclass(ModuleNotFoundError, ImportError))

# ===== SECTION 118: bytes and bytearray are their own types =====
# Neither is a list: type, repr, == against a list, hashing and immutability
# are bytes', int() and float() read the bytes as ASCII text, and the str-like
# methods answer bytes. A bytearray is the mutable one and equals bytes of the
# same bytes. Rows from CPython 3.14.
def S118err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

def S118set(b):
    b[0] = 120
    return b

def s118():
    bs = [b"ab", bytearray(b"ab"), b"a'b\x00\xff"]
    n = [255, 2]
    check("byt1", (n[0]).to_bytes(n[1], "big", signed=True) == b"\x00\xff" and type(bs[0]).__name__ == "bytes"
          and type(bs[1]).__name__ == "bytearray" and repr(bs[2]) == "b\"a'b\\x00\\xff\"" and str(bs[1]) == "bytearray(b'ab')")
    check("byt2", not (bs[0] == [97, 98]) and bs[0] == bs[1] and bs[0] != (97, 98) and list(bs[0]) == [97, 98]
          and hash(bs[0]) == hash(b"ab") and {bs[0]: 1}[b"ab"] == 1
          and S118err(lambda: hash(bs[1])) == "TypeError: unhashable type: 'bytearray'")
    check("byt3", int(b"12") == 12 and float(b"1.5") == 1.5 and int(bytearray(b" -7 ")) == -7
          and S118err(lambda: int(b"1x")) == "ValueError: invalid literal for int() with base 10: b'1x'"
          and S118err(lambda: float(b"x")) == "ValueError: could not convert string to float: b'x'")
    check("byt4", S118err(lambda: S118set(bs[0])) == "TypeError: 'bytes' object does not support item assignment"
          and S118set(bytearray(b"ab")) == b"xb" and bs[0] + bs[1] == b"abab" and type(bs[1] + bs[0]).__name__ == "bytearray"
          and S118err(lambda: bs[0] + [1]) == "TypeError: can't concat list to bytes"
          and bs[0][1:] == b"b" and bs[0] * 2 == b"abab" and S118err(lambda: bs[0].append(1)).startswith("AttributeError"))
    check("byt5", b"a,b".split(b",") == [b"a", b"b"] and b"-".join([b"x", b"y"]) == b"x-y" and b"aZ\xe9".upper() == b"AZ\xe9"
          and b"x" in bs[0] + b"x" and 97 in bs[0] and "café".encode() == b"caf\xc3\xa9" and b"\x01\xab".hex() == "01ab"
          and bytes.fromhex("01 ab") == b"\x01\xab" and b"%d-%s" % (5, b"z") == b"5-z")

# ===== SECTION 119: module names, a type's own methods, the bytes dunders and methods =====
# Every module answers __name__. dict.fromkeys and the other methods a builtin
# type object carries refuse keywords (int.from_bytes alone takes them), called
# or read as a value. bytes and bytearray answer their dunders like list does;
# splitlines breaks at \n \r \r\n only; translate/maketrans, hex(sep),
# decode(errors=) and bytes % follow CPython's per-conversion rules. Rows from
# CPython 3.14.
def S119err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

class S119B:
    def __bytes__(self):
        return b"BB"

def s119():
    import math
    import collections
    mods = [math, sys, collections]
    check("mod1", [m.__name__ for m in mods] == ["math", "sys", "collections"] and hasattr(math, "__name__")
          and getattr(sys, "__name__") == "sys")
    fk = dict.fromkeys
    check("tkw1", S119err(lambda: dict.fromkeys("ab", value=1)) == "TypeError: dict.fromkeys() takes no keyword arguments"
          and S119err(lambda: fk("ab", value=1)) == "TypeError: dict.fromkeys() takes no keyword arguments"
          and fk("ab", {}) == {"a": {}, "b": {}} and fk.__qualname__ == "dict.fromkeys")
    fb = int.from_bytes
    check("tkw2", S119err(lambda: float.fromhex("1", x=1)) == "TypeError: float.fromhex() takes no keyword arguments"
          and S119err(lambda: bytes.fromhex("01", x=1)) == "TypeError: bytes.fromhex() takes no keyword arguments"
          and fb(b"\x01\x00", byteorder="little") == 1 and fb(b"\xff", "big", signed=True) == -1
          and S119err(lambda: int.from_bytes(b"\x01", x=1)) == "TypeError: from_bytes() got an unexpected keyword argument 'x'")
    ba = bytearray(b"abcdef")
    ba.__delitem__(0)
    ba.__setitem__(0, 65)
    bs = [b"a", b"b", bytearray(b"a")]
    check("bdun1", ba == b"Acdef" and ba.__getitem__(1) == 99 and ba.__len__() == 5 and ba.__contains__(99)
          and S119err(lambda: ba.__setitem__(0, 300)) == "ValueError: byte must be in range(0, 256)"
          and S119err(lambda: bs[0].__delitem__(0)) == "AttributeError: 'bytes' object has no attribute '__delitem__'"
          and hasattr(ba, "__delitem__") and not hasattr(bs[0], "__setitem__") and not hasattr(ba, "__reversed__"))
    check("bdun2", bs[0].__add__(bs[1]) == b"ab" and bs[0].__mul__(3) == b"aaa" and bs[0].__rmul__(2) == b"aa"
          and bs[0].__eq__(bs[2]) is NotImplemented and bs[0].__lt__(bs[1]) and bytearray.__hash__ is None
          and bs[0].__hash__() == hash(b"a") and bytes.__len__(b"abc") == 3 and "%d".__rmod__(3) is NotImplemented
          and S119err(lambda: bs[0].__add__([1])) == "TypeError: can't concat list to bytes")
    tb = bytes.maketrans(b"abc", b"xyz")
    ls = [b"a\nb\r\nc", b"d\x0be\x0cf\x1cg\r\r\n"]
    check("bmeth1", ls[0].splitlines() == [b"a", b"b", b"c"] and ls[1].splitlines(keepends=True) == [b"d\x0be\x0cf\x1cg\r", b"\r\n"]
          and bytearray(b"a\nb").splitlines() == [bytearray(b"a"), bytearray(b"b")]
          and len(tb) == 256 and tb[97:100] == b"xyz" and b"abcabc".translate(tb, b"b") == b"xzxz"
          and bytearray(b"abc").translate(None, b"ac") == bytearray(b"b")
          and S119err(lambda: b"a".translate(b"x")) == "ValueError: translation table must be 256 characters long")
    hb = b"\x01\x02\x03\x04\x05"
    check("bmeth2", hb.hex(":") == "01:02:03:04:05" and hb.hex(":", 2) == "01:0203:0405" and hb.hex(b"-", -2) == "0102-0304-05"
          and hb.hex(sep="|", bytes_per_sep=9) == "0102030405" and S119err(lambda: hb.hex("::")) == "ValueError: sep must be length 1.")
    ds = [b"a\xe2\x82b", b"\xf0\x9f\x98"]
    check("bdec1", [d.decode("utf-8", "replace") for d in ds] == ["a�b", "�"]
          and ds[0].decode(errors="ignore") == "ab" and ds[0].decode("ascii", "backslashreplace") == "a\\xe2\\x82b"
          and str(ds[1], "utf-8", "backslashreplace") == "\\xf0\\x9f\\x98" and b"ok".decode("utf-8", "bogus") == "ok"
          and S119err(lambda: ds[0].decode("utf-8", "bogus")) == "LookupError: unknown error handler name 'bogus'")
    pa = [5, b"x", "\xe9", 200]
    check("bpct1", S119err(lambda: b"%s" % pa[0]) == "TypeError: %b requires a bytes-like object, or an object that implements __bytes__, not 'int'"
          and b"%r|%a" % (pa[1], pa[2]) == b"b'x'|'\\xe9'" and b"%c%c" % (pa[3], pa[1]) == b"\xc8x" and b"%b" % pa[1] == b"x"
          and b"%s" % S119B() == b"BB" and b"%(k)s" % {b"k": b"v"} == b"v" and b"%5.2r|" % pa[1] == b"   b'|"
          and S119err(lambda: b"" % pa[1]) == "TypeError: not all arguments converted during bytes formatting")

# ===== SECTION 120: slice objects =====
# slice() builds the object the slice syntax stands for: its bounds read back
# as start/stop/step, indices() clamps them, subscripting a sequence with one
# slices it (reading, assigning, deleting), equal slices are one dict key, and
# an instance's __getitem__/__setitem__/__delitem__ receives one for a[i:j],
# alone or inside a tuple. Rows from CPython 3.14.
def S120err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

class S120C:
    def __init__(self):
        self.log = []
    def __getitem__(self, k):
        return k
    def __setitem__(self, k, v):
        self.log.append(("set", k, v))
    def __delitem__(self, k):
        self.log.append(("del", k))

class S120L(list):
    pass

class S120D:
    pass

def s120():
    bs = [(1, 3), (None, None, -1), (5,), (4, 0, -2)]
    ss = [slice(*b) for b in bs]
    check("slc1", [repr(s) for s in ss] == ["slice(1, 3, None)", "slice(None, None, -1)", "slice(None, 5, None)", "slice(4, 0, -2)"]
          and (ss[0].start, ss[0].stop, ss[0].step) == (1, 3, None) and type(ss[0]) is slice and isinstance(ss[1], slice)
          and ss[1].indices(6) == (5, -1, -1) and ss[3].indices(2) == (1, 0, -2) and ss[2].indices(9) == (0, 5, 1))
    xs = [0, 1, 2, 3, 4, 5]
    check("slc2", xs[ss[0]] == [1, 2] and "abcdef"[ss[1]] == "fedcba" and (0, 1, 2, 3, 4, 5)[ss[3]] == (4, 2)
          and b"abcdef"[ss[2]] == b"abcde" and ss[0] == slice(1, 3) and ss[0] != slice(1, 4)
          and {ss[0]: "a"}[slice(1, 3)] == "a" and hash(slice(1, 2)) == hash(slice(1, 2)))
    ys = [0, 1, 2, 3, 4, 5]
    ys[slice(1, 3)] = ["a"]
    del ys[slice(None, None, 2)]
    check("slc3", ys == ["a", 4] and S120err(lambda: slice()) == "TypeError: slice expected at least 1 argument, got 0"
          and S120err(lambda: slice(None, None, 0).indices(3)) == "ValueError: slice step cannot be zero"
          and S120err(lambda: slice(1).nope) == "AttributeError: 'slice' object has no attribute 'nope'")
    c = S120C()
    c[1:2] = [5]
    del c[2:]
    check("slc4", c[1:3] == slice(1, 3) and c[::2] == slice(None, None, 2) and c[1:2, 3] == (slice(1, 2), 3)
          and c.log == [("set", slice(1, 2), [5]), ("del", slice(2, None))])
    sl = S120L([0, 1, 2, 3])
    sl[0:2] = [9]
    check("slc5", sl[1:] == [2, 3] and type(sl[1:]) is list and sl == [9, 2, 3]
          and S120err(lambda: S120D()[1:2]) == "TypeError: 'S120D' object is not subscriptable")
    # Slices order and hash as their (start, stop, step) tuples.
    check("slc6", ss[0] < slice(1, 4) and ss[3] > ss[0] and ss[0] <= slice(1, 3) and not (ss[0] >= slice(2, 0))
          and sorted([ss[3], ss[0], slice(1, 2)]) == [slice(1, 2), ss[0], ss[3]] and max([ss[0], ss[3]]) == ss[3]
          and S120err(lambda: slice(1, 2, 3) >= slice(1, 2)) == "TypeError: '>=' not supported between instances of 'int' and 'NoneType'"
          and S120err(lambda: ss[0] < (1, 3)) == "TypeError: '<' not supported between instances of 'slice' and 'tuple'")
    check("slc7", S120err(lambda: {slice(xs): 0}) == "TypeError: cannot use 'slice' as a dict key (unhashable type: 'list')"
          and S120err(lambda: {slice(1, {})}) == "TypeError: cannot use 'slice' as a set element (unhashable type: 'dict')"
          and S120err(lambda: hash(slice(1, xs))) == "TypeError: unhashable type: 'list'")

# ===== SECTION 121: namedtuple and tuple subclasses =====
# collections.namedtuple (lib/python-collections.py): field access, defaults,
# keywords, _make/_replace/_asdict, and the texts its generated __new__ raises.
# Under it sit tuple subclasses: their count/index, `%` reading one as the
# argument tuple, `__slots__ = ()` leaving no __dict__, and an unbound builtin
# method run on a subclass instance. Rows from CPython 3.14.
from collections import namedtuple as S121nt

def S121err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

class S121T(tuple):
    __slots__ = ()

class S121D(dict):
    pass

def s121():
    P = S121nt("P", "x y", defaults=[0])
    Q = S121nt("Q", ["a", "b", "c"])
    q = Q(1, b=2, c=3)
    check("ntp1", repr(q) == "Q(a=1, b=2, c=3)" and (q.a, q[1], q.c) == (1, 2, 3) and q == (1, 2, 3)
          and isinstance(q, tuple) and repr(P(5)) == "P(x=5, y=0)" and P(y=1, x=2) == (2, 1)
          and q._asdict() == {"a": 1, "b": 2, "c": 3} and q._replace(a=9) == (9, 2, 3)
          and Q._make(range(3)) == (0, 1, 2) and Q._fields == ("a", "b", "c") and P._field_defaults == {"y": 0})
    check("ntp2", [S121err(f) for f in [lambda: P(), lambda: P(1, 2, 3), lambda: P(1, z=2), lambda: P(1, x=2),
                                         lambda: Q(), lambda: P(1)._replace(q=1), lambda: setattr(q, "a", 0),
                                         lambda: S121nt("R", "x x"), lambda: S121nt("R", "_x")]]
          == ["TypeError: P.__new__() missing 1 required positional argument: 'x'",
              "TypeError: P.__new__() takes from 2 to 3 positional arguments but 4 were given",
              "TypeError: P.__new__() got an unexpected keyword argument 'z'",
              "TypeError: P.__new__() got multiple values for argument 'x'",
              "TypeError: Q.__new__() missing 3 required positional arguments: 'a', 'b', and 'c'",
              "TypeError: Got unexpected field names: ['q']",
              "AttributeError: can't set attribute",
              "ValueError: Encountered duplicate field name: 'x'",
              "ValueError: Field names cannot start with an underscore: '_x'"])
    t = S121T((1, 2, 2))
    check("ntp3", t.count(2) == 2 and t.index(2) == 1 and tuple.index(t, 2) == 1 and q.count(2) == 1
          and "%s-%s-%s" % t == "1-2-2" and "%(k)s" % S121D(k=3) == "3"
          and S121err(lambda: setattr(t, "z", 1)) == "AttributeError: 'S121T' object has no attribute 'z' and no __dict__ for setting new attributes")

# ===== SECTION 122: math's integer helpers, prod and fsum =====
# isqrt / comb / perm / gcd / lcm are exact on ints of any size (lcm of two
# small operands can pass 2^53), prod multiplies with Python's own `*` from
# start, and fsum is the exactly rounded sum. Rows from CPython 3.14.
import math as S122m

def S122err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

def s122():
    ns = [0, 3, 4, 99, 10**20, 10**20 - 1, 2**100 + 7, True]
    check("mint1", [S122m.isqrt(n) for n in ns] == [0, 1, 2, 9, 10**10, 10**10 - 1, 2**50, 1]
          and [S122m.comb(n, k) for n, k in [(5, 2), (3, 5), (50, 25), (60, 30)]] == [10, 0, 126410606437752, 118264581564861424]
          and S122m.perm(5, 2) == 20 and S122m.perm(5) == 120 and S122m.perm(30, 25) == 2210440498434925488635904000000
          and S122m.gcd(2**70, 2**65 * 3) == 2**65 and S122m.gcd(10**30 + 1, 10**15 + 1) == 1
          and S122m.lcm(2**40, 3**30) == 226379693794030958489370624 and S122m.lcm() == 1 and S122m.lcm(4, 0) == 0)
    check("mint2", S122m.prod([1, 2, 3, 4]) == 24 and S122m.prod([]) == 1 and S122m.prod(range(1, 30)) == 8841761993739701954543616000000
          and S122m.prod([2, 3], start=10) == 60 and S122m.prod(x for x in [2, 2.5]) == 5.0 and S122m.prod(["ab"], start=2) == "abab"
          and S122m.fsum([0.1] * 10) == 1.0 and S122m.fsum([1e100, 1.0, -1e100, 1e-100, 1e50, -1.0, -1e50]) == 1e-100
          and S122m.fsum([2.0**53, -0.5, -2.0**-54]) == 9007199254740991.0 and S122m.fsum([1.0, 1e-16, 1e-16]) == 1.0000000000000002
          and S122m.fsum([float("inf"), 1.0]) == float("inf") and S122m.fsum([]) == 0.0)
    check("mint3", [S122err(f) for f in [lambda: S122m.isqrt(-1), lambda: S122m.isqrt(2.0), lambda: S122m.comb(-1, 2),
                                          lambda: S122m.comb(3, -2), lambda: S122m.perm(-3),
                                          lambda: S122m.fsum([float("inf"), -float("inf")]), lambda: S122m.fsum([1e308, 1e308])]]
          == ["ValueError: isqrt() argument must be nonnegative",
              "TypeError: 'float' object cannot be interpreted as an integer",
              "ValueError: n must be a non-negative integer", "ValueError: k must be a non-negative integer",
              "ValueError: factorial() not defined for negative values", "ValueError: -inf + inf in fsum",
              "OverflowError: intermediate overflow in fsum"])

# ===== SECTION 123: __init_subclass__, bound methods with keywords, operator, __format__ =====
# A class statement ends by calling the nearest base's __init_subclass__ with
# the new class (implicitly a classmethod, ending at object's). A bound method
# held as a value - a user one or a builtin one - takes keyword arguments.
# The operator module (lib/python-operator.py), and a class's own __format__.
# Rows from CPython 3.14.
import operator as S123op
from operator import itemgetter as S123ig, attrgetter as S123ag, methodcaller as S123mc

class S123Plugin:
    seen = []
    def __init_subclass__(cls, **kw):
        super().__init_subclass__(**kw)
        S123Plugin.seen.append(cls.__name__)

class S123A(S123Plugin):
    pass

class S123B(S123A):
    pass

class S123Tag:
    @classmethod
    def __init_subclass__(cls):
        cls.tag = cls.__name__.lower()

class S123X(S123Tag):
    pass

class S123P:
    def __init__(self, n):
        self.n = n
    def f(self, a, b=0):
        return self.n * a + b

class S123F:
    def __format__(self, spec):
        return "<" + spec + ">"
    def __str__(self):
        return "S"

def S123err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

def s123():
    check("isub1", S123Plugin.seen == ["S123A", "S123B"] and S123X.tag == "s123x" and not hasattr(S123Tag, "tag")
          and type("S123C", (S123A,), {}).__name__ == "S123C" and S123Plugin.seen[-1] == "S123C")
    p = S123P(3)
    g = p.f
    d = {}
    upd = d.update
    upd({"a": 1}, b=2)
    xs = [3, 1, 2]
    srt = xs.sort
    srt(reverse=True)
    check("bmkw1", g(2, b=1) == 7 and g(**{"a": 1, "b": 5}) == 8 and d == {"a": 1, "b": 2} and xs == [3, 2, 1]
          and getattr("a b c", "split")(maxsplit=1) == ["a", "b c"])
    rows = [("x", 3), ("y", 1), ("x", 2)]
    check("oper1", sorted(rows, key=S123ig(1)) == [("y", 1), ("x", 2), ("x", 3)] and S123ig(1, 0)(rows[0]) == (3, "x")
          and S123ag("n")(p) == 3 and S123mc("f", 2, b=1)(p) == 7 and S123op.add(2, 3) == 5 and S123op.neg(4) == -4
          and S123op.contains([1, 2], 2) and S123op.indexOf("abc", "c") == 2 and S123op.countOf([1, 1, 2], 1) == 2
          and repr(S123ig(1)) == "operator.itemgetter(1)" and repr(S123mc("f", 1, k=2)) == "operator.methodcaller('f', 1, k=2)"
          and S123op.index(True) == 1 and S123op.abs(-2.5) == 2.5 and S123op.not_(0) is True)
    # format(), f-strings and str.format all ask the class's own __format__; a
    # plain field is format(v, ""), and object's refuses a non-empty spec.
    m = S123F()
    check("ufmt1", [format(m, "usd"), f"{m:abc}", f"{m}", "{:q}".format(m), "{}".format(m), f"{m!s:>3}"]
          == ["<usd>", "<abc>", "<>", "<q>", "<>", "  S"]
          and S123err(lambda: f"{p:x}") == "TypeError: unsupported format string passed to S123P.__format__"
          and S123err(lambda: format(p, ">3")) == "TypeError: unsupported format string passed to S123P.__format__")

# ===== SECTION 124: regex inline flags, unset backreferences, $ and \Z =====
# (?aiLmsux) at the start sets flags for the whole pattern and shows in .flags;
# (?imsx-imsx:...) scopes them. A backreference to a group that did not take
# part fails. $ also matches before a final newline; \Z only at the very end.
import re as S124re

def S124m(p, s, fl=0):
    m = S124re.search(p, s, fl)
    return None if m is None else m.group(0)

def s124():
    check("rxif1", [S124m(r"(?i)A", "xa"), S124m(r"(?s)a.b", "a\nb"), S124m(r"(?x) a b # c", "ab"),
                    S124m(r"(?m)^b$", "a\nb\nc"), S124m(r"(?im)^B$", "x\nb\ny"), S124m(r"(?i)[^a]", "AaB")]
          == ["a", "a\nb", "ab", "b", "b", "B"])
    check("rxif2", [S124m(r"a(?i:B)c", "aBc"), S124m(r"a(?-i:B)c", "abc", S124re.I), S124m(r"(?i)a(?-i:b)", "AB"),
                    S124m(r"(?i)a(?-i:b)", "Ab"), S124m(r"x(?i:[a-c]+)y", "xBcAy"), S124m(r"(?x:a b)c d", "abc d"),
                    S124m(r"(?s:a.)b.", "a\nb\n"), S124m(r"(?s:a.)b.", "a\nbx"), S124m(r"(?m:^b)", "a\nb")]
          == ["aBc", None, None, "Ab", "xBcAy", "abc d", None, "a\nbx", "b"])
    check("rxif3", S124re.compile("(?i)a").flags == 34 and S124re.compile("(?s)a", S124re.M).flags == 56
          and S124re.compile("(?im)a").flags == 42 and S124re.compile("a(?i:b)").flags == 32)
    check("rxif4", S124m(r"(?i)(a)\1", "aA") == "aA" and S124m(r"(?i:(a)\1)", "aA") == "aA" and S124m(r"(a)\1", "aA") is None)
    check("rxbr1", S124re.fullmatch(r"(b)?a\1", "a") is None and S124re.fullmatch(r"(b)?a\1", "bab").group(0) == "bab"
          and S124re.search(r"(?:(x)|y)z\1", "yzx") is None)
    check("rxeol1", [S124m("a$", "a\n"), S124m(r"a\Z", "a\n"), S124m(r"a\Z", "xa"), S124m("^b", "a\nb"),
                     S124m("a$", "a\nb", S124re.M)] == ["a", None, "a", None, "a"]
          and S124re.findall(r"\w$", "ab\ncd\n") == ["d"] and S124re.findall(r"^\w", "ab\ncd", S124re.M) == ["a", "c"])

# ===== SECTION 125: private names, the legacy sequence protocol, keyword clashes, __context__ =====
# Inside a class, __x is _Class__x. An instance with __getitem__ and no
# __iter__ iterates by __getitem__(0), (1), ... until IndexError. A keyword
# given twice through ** is a TypeError naming the callee. An exception raised
# while another is handled - anywhere, even in a called function - records it
# as __context__.
class S125A:
    __cv = 7
    def __init__(self):
        self.__x = 1
    def __priv(self):
        return "A"
    def peek(self):
        return (self.__x, self.__priv(), self.__cv)

class S125B(S125A):
    def __init__(self):
        S125A.__init__(self)
        self.__x = 2
    def mine(self):
        return self.__x

class S125Seq:
    def __getitem__(self, i):
        if i < 3:
            return i * 10
        raise IndexError(i)

class S125Nat:
    def __getitem__(self, i):
        return i

def S125f(a, b=0, **kw):
    return (a, b, kw)

def S125err(f):
    try:
        f()
    except Exception as e:
        return type(e).__name__ + ": " + str(e)
    return "no error"

def S125bad():
    raise KeyError("inner")

def S125ctx(k):
    try:
        try:
            raise ValueError("outer")
        except ValueError:
            if k == 0:
                S125bad()
            1 / 0
    except Exception as e:
        return repr(e.__context__)

def S125gen():
    try:
        raise ValueError("g")
    except ValueError:
        yield 1

def s125():
    b = S125B()
    check("mng1", sorted(vars(b)) == ["_S125A__x", "_S125B__x"] and b.peek() == (1, "A", 7) and b.mine() == 2
          and b._S125A__x == 1 and S125A._S125A__priv.__name__ == "__priv" and not hasattr(b, "__x")
          and "_S125A__cv" in S125A.__dict__)
    s = S125Seq()
    it = iter(s)
    seen = []
    for x in S125Nat():
        if x > 3:
            break
        seen.append(x)
    check("lgi1", list(s) == [0, 10, 20] and [*s] == [0, 10, 20] and 20 in s and 5 not in s and 7 in S125Nat()
          and type(it).__name__ == "iterator" and next(it) == 0 and list(it) == [10, 20] and seen == [0, 1, 2, 3]
          and sum(s) == 30 and dict.fromkeys(s) == {0: None, 10: None, 20: None})
    check("kwd1", S125err(lambda: S125f(1, b=1, **{"b": 2})) == "TypeError: __main__.S125f() got multiple values for keyword argument 'b'"
          and S125err(lambda: S125f(1, **{"z": 1}, z=2)) == "TypeError: __main__.S125f() got multiple values for keyword argument 'z'"
          and S125err(lambda: dict(a=1, **{"a": 2})) == "TypeError: dict() got multiple values for keyword argument 'a'"
          and S125err(lambda: {}.update(a=1, **{"a": 2})) == "TypeError: dict.update() got multiple values for keyword argument 'a'"
          and S125f(1, **{"b": 2}, c=3) == (1, 2, {"c": 3}))
    out = []
    for v in S125gen():
        try:
            raise KeyError(v)
        except KeyError as e:
            out.append(e.__context__)
    check("ctx1", S125ctx(0) == "ValueError('outer')" and S125ctx(1) == "ValueError('outer')" and out == [None])

# ===== SECTION 126: a generator's body runs once =====
# Every statement of a generator body runs exactly once however it is stepped:
# next(), send(), throw(), close() and yield from resume it where it is parked.
S126log = []

def S126g():
    S126log.append("start")
    try:
        x = yield 1
        S126log.append(("got", x))
        yield 2
    except ValueError as e:
        S126log.append(("caught", str(e)))
        yield 3
    finally:
        S126log.append("fin")

def S126inner():
    S126log.append("inner")
    r = yield "a"
    S126log.append(("inner got", r))
    return "ret"

def S126outer():
    v = yield from S126inner()
    S126log.append(("ret", v))
    yield "b"

def S126app(xs):
    xs.append(1)
    yield
    xs.append(2)
    yield

def s126():
    g = S126g()
    vals = [next(g), g.send("s"), g.throw(ValueError("v"))]
    g.close()
    check("gen1", vals == [1, 2, 3] and S126log == ["start", ("got", "s"), ("caught", "v"), "fin"])
    del S126log[:]
    o = S126outer()
    vals2 = [next(o), o.send(5)] + list(o)
    shared = []
    check("gen2", vals2 == ["a", "b"] and S126log == ["inner", ("inner got", 5), ("ret", "ret")]
          and list(S126app(shared)) == [None, None] and shared == [1, 2])

# ===== SECTION 127: math at the double level, object repr, OSError, __hash__ = None =====
# frexp/ldexp/modf/ulp/nextafter are exact; hypot and dist are CPython's
# vector_norm. A plain instance's repr names its module. OSError(errno, ...)
# answers the subclass the errno names and carries errno/strerror/filename. A
# class defining __eq__ without __hash__ has __hash__ None.
import math as S127m

class S127G:
    class In:
        pass

class S127E:
    def __eq__(self, o):
        return True

def s127():
    check("mth1", S127m.frexp(12.5) == (0.78125, 4) and S127m.frexp(5e-324) == (0.5, -1073)
          and S127m.ldexp(0.75, -1074) == 5e-324 and S127m.ldexp(0.5, -1075) == 0.0 and S127m.ldexp(0.75, -1075) == 0.0
          and repr(S127m.modf(-3.25)) == "(-0.25, -3.0)" and repr(S127m.modf(-2.0)) == "(-0.0, -2.0)"
          and S127m.ulp(1.0) == 2.220446049250313e-16 and S127m.ulp(1.7976931348623157e308) == 1.99584030953472e+292
          and S127m.nextafter(1.0, 0.0) == 0.9999999999999999 and repr(S127m.nextafter(-0.0, 0.0)) == "0.0"
          and S127m.nextafter(5e-324, 0.0) == 0.0)
    check("mth2", repr(S127m.hypot(0.1, 0.2, 0.3)) == "0.3741657386773941" and S127m.dist((1, 2), (4, 6)) == 5.0
          and repr(S127m.hypot(1e-310, 3e-310)) == "3.1622776601684e-310" and repr(S127m.hypot(5.4, 1.3, 2.2)) == "5.974110812497539"
          and S127m.hypot(S127m.inf, S127m.nan) == S127m.inf and repr(S127m.ldexp(1.5, 1023)) == "1.348269851146737e+308")
    check("orp1", repr(S127G())[:25] == "<__main__.S127G object" + repr(S127G())[22:25]
          and str(S127G.In()).startswith("<__main__.S127G.In object"))
    e = OSError(2, "nf", "f.txt")
    check("ose1", type(e) is FileNotFoundError and e.args == (2, "nf") and e.errno == 2 and e.filename == "f.txt"
          and str(e) == "[Errno 2] nf: 'f.txt'" and repr(OSError(13, "p")) == "PermissionError(13, 'p')"
          and type(OSError(99, "z")) is OSError and OSError("a").errno is None and IOError is OSError
          and issubclass(BrokenPipeError, ConnectionError) and OSError(2, "a", None).args == (2, "a", None))
    check("hsh1", S127E.__hash__ is None and "__hash__" in vars(S127E))

# ===== SECTION 128: metaclasses and the enum module =====
# metaclass= (or a base's metaclass) makes the class through M.__new__ and
# M.__init__; iter(C), C[k], len(C), `in`, C(...) and repr(C) ask M, and an
# attribute C lacks is M's. enum is written on top of that.
from enum import Enum as S128Enum, IntEnum as S128IntEnum, auto as S128auto, unique as S128unique

S128log = []

class S128M(type):
    def __new__(mcs, name, bases, ns, **kw):
        S128log.append(("new", name, sorted(k for k in ns if not k.startswith("__")), kw))
        cls = super().__new__(mcs, name, bases, ns)
        cls.tag = name.lower()
        return cls
    def __init__(cls, name, bases, ns, **kw):
        S128log.append(("init", name))
        super().__init__(name, bases, ns)
    def __iter__(cls):
        return iter([1, 2, 3])
    def __getitem__(cls, k):
        return k * 2
    def __call__(cls, *a):
        S128log.append(("call", a))
        return super().__call__(*a)
    def __len__(cls):
        return 3
    def __contains__(cls, x):
        return x == 2
    def hello(cls):
        return "hello " + cls.__name__
    @property
    def prop(cls):
        return "p:" + cls.__name__
    def __repr__(cls):
        return "<M " + cls.__name__ + ">"

class S128C(metaclass=S128M, flag=1):
    x = 1
    def __init__(self, v=0):
        self.v = v

class S128D(S128C):
    pass

class S128Color(S128Enum):
    RED = S128auto()
    GREEN = S128auto()
    BLUE = "b"

class S128Num(S128IntEnum):
    ONE = 1
    TWO = 2
    UNO = 1

def s128():
    c = S128C(7)
    check("meta1", [list(S128C), S128C[5], len(S128C), 2 in S128C, 3 in S128C, S128C.hello(), S128C.prop, S128C.tag, repr(S128C)]
          == [[1, 2, 3], 10, 3, True, False, "hello S128C", "p:S128C", "s128c", "<M S128C>"]
          and type(S128C) is S128M and isinstance(S128C, S128M) and isinstance(S128C, type) and c.v == 7)
    check("meta2", S128log == [("new", "S128C", ["x"], {"flag": 1}), ("init", "S128C"), ("new", "S128D", [], {}),
                               ("init", "S128D"), ("call", (7,))]
          and type(S128D) is S128M and S128D.tag == "s128d" and S128M("Dyn", (), {"y": 2}).y == 2)
    check("enum1", [m.name for m in S128Color] == ["RED", "GREEN", "BLUE"] and S128Color.GREEN.value == 2
          and repr(S128Color.BLUE) == "<S128Color.BLUE: 'b'>" and str(S128Color.RED) == "S128Color.RED"
          and S128Color(2) is S128Color.GREEN and S128Color["RED"] is S128Color.RED and len(S128Color) == 3
          and repr(S128Color) == "<enum 'S128Color'>" and isinstance(S128Color.RED, S128Color)
          and list(S128Color.__members__) == ["RED", "GREEN", "BLUE"])
    check("enum2", repr(S128Num.ONE) == "<S128Num.ONE: 1>" and str(S128Num.TWO) == "2" and S128Num.ONE + 1 == 2
          and S128Num.UNO is S128Num.ONE and list(S128Num) == [S128Num.ONE, S128Num.TWO] and 1 in S128Num
          and f"{S128Num.TWO:>3}" == "  2" and S128Num(2) is S128Num.TWO)
    msg = ""
    try:
        S128Color(9)
    except ValueError as e:
        msg = str(e)
    dup = ""
    try:
        S128unique(S128Num)
    except ValueError as e:
        dup = str(e)
    check("enum3", msg == "9 is not a valid S128Color" and dup == "duplicate values found in <enum 'S128Num'>: UNO -> ONE")

# ===== SECTION 129: a set of ints iterates in CPython's order =====
# The slot order of CPython's open-addressed table: hash(i) == i, so small ints
# come out by value modulo the table size, and collisions probe by
# perturbation. The sets are built by adding members one at a time (set(),
# add, comprehensions); a display of three or more constants goes through a
# frozenset constant in CPython, whose merge sizes the table differently.
def s129():
    check("sord1", str({x % 3 for x in [5, 4, 3]}) == "{0, 1, 2}" and list(set([33, 1, 17, 9])) == [33, 9, 1, 17]
          and list(set(range(20, 0, -1))) == list(range(1, 21)) and list(set([-1, -2, -3, -8, -16, -9])) == [-16, -2, -9, -8, -3, -1]
          and list(set([100, 8, 16, 0, -1, -2, 3])) == [0, 3, 100, 8, 16, -2, -1]
          and str(frozenset([9, 1, 17])) == "frozenset({9, 1, 17})" and sorted({5, "a"}, key=str) == [5, "a"])
    s = set([5, 4, 3])
    check("sord2", s.pop() == 3 and len(s) == 2
          and [v for v in set([2**40 + 3, 2**33 + 5, 1, -2**35, 9, 7])] == [-34359738368, 1, 1099511627779, 8589934597, 7, 9])

def main():
    s01() # SECTION-CALL 01
    s02() # SECTION-CALL 02
    s03() # SECTION-CALL 03
    s04() # SECTION-CALL 04
    s05() # SECTION-CALL 05
    s06() # SECTION-CALL 06
    s07() # SECTION-CALL 07
    s08() # SECTION-CALL 08
    s09() # SECTION-CALL 09
    s10() # SECTION-CALL 10
    s11() # SECTION-CALL 11
    s12() # SECTION-CALL 12
    s13() # SECTION-CALL 13
    s14() # SECTION-CALL 14
    s15() # SECTION-CALL 15
    s16() # SECTION-CALL 16
    s17() # SECTION-CALL 17
    s18() # SECTION-CALL 18
    s19() # SECTION-CALL 19
    s20() # SECTION-CALL 20
    s21() # SECTION-CALL 21
    s22() # SECTION-CALL 22
    s23() # SECTION-CALL 23
    s24() # SECTION-CALL 24
    s25() # SECTION-CALL 25
    s26() # SECTION-CALL 26
    s27() # SECTION-CALL 27
    s28() # SECTION-CALL 28
    s29() # SECTION-CALL 29
    s30() # SECTION-CALL 30
    s31() # SECTION-CALL 31
    s32() # SECTION-CALL 32
    s33() # SECTION-CALL 33
    s34() # SECTION-CALL 34
    s35() # SECTION-CALL 35
    s36() # SECTION-CALL 36
    s37() # SECTION-CALL 37
    s38() # SECTION-CALL 38
    s39() # SECTION-CALL 39
    s40() # SECTION-CALL 40
    s41() # SECTION-CALL 41
    s42() # SECTION-CALL 42
    s43() # SECTION-CALL 43
    s44() # SECTION-CALL 44
    s45() # SECTION-CALL 45
    s46() # SECTION-CALL 46
    s47() # SECTION-CALL 47
    s48() # SECTION-CALL 48
    s49() # SECTION-CALL 49
    s50() # SECTION-CALL 50
    s51() # SECTION-CALL 51
    s52() # SECTION-CALL 52
    s53() # SECTION-CALL 53
    s54() # SECTION-CALL 54
    s55() # SECTION-CALL 55
    s56() # SECTION-CALL 56
    s57() # SECTION-CALL 57
    s58() # SECTION-CALL 58
    s59() # SECTION-CALL 59
    s60() # SECTION-CALL 60
    s61() # SECTION-CALL 61
    s62() # SECTION-CALL 62
    s63() # SECTION-CALL 63
    s64() # SECTION-CALL 64
    s65() # SECTION-CALL 65
    s66() # SECTION-CALL 66
    s67() # SECTION-CALL 67
    s68() # SECTION-CALL 68
    s69() # SECTION-CALL 69
    s70() # SECTION-CALL 70
    s71() # SECTION-CALL 71
    s72() # SECTION-CALL 72
    s73() # SECTION-CALL 73
    s74() # SECTION-CALL 74
    s75() # SECTION-CALL 75
    s76() # SECTION-CALL 76
    s77() # SECTION-CALL 77
    s78() # SECTION-CALL 78
    s79() # SECTION-CALL 79
    s80() # SECTION-CALL 80
    s81() # SECTION-CALL 81
    s82() # SECTION-CALL 82
    s83() # SECTION-CALL 83
    s84() # SECTION-CALL 84
    s85() # SECTION-CALL 85
    s86() # SECTION-CALL 86
    s87() # SECTION-CALL 87
    s88() # SECTION-CALL 88
    s89() # SECTION-CALL 89
    s92() # SECTION-CALL 92
    s93() # SECTION-CALL 93
    s94() # SECTION-CALL 94
    s95() # SECTION-CALL 95
    s96() # SECTION-CALL 96
    s98() # SECTION-CALL 98
    s99() # SECTION-CALL 99
    s100() # SECTION-CALL 100
    s101() # SECTION-CALL 101
    s102() # SECTION-CALL 102
    s103() # SECTION-CALL 103
    s104() # SECTION-CALL 104
    s105() # SECTION-CALL 105
    s106() # SECTION-CALL 106
    s107() # SECTION-CALL 107
    s108() # SECTION-CALL 108
    s109() # SECTION-CALL 109
    s110() # SECTION-CALL 110
    s111() # SECTION-CALL 111
    s112() # SECTION-CALL 112
    s113() # SECTION-CALL 113
    s114() # SECTION-CALL 114
    s115() # SECTION-CALL 115
    s116() # SECTION-CALL 116
    s117() # SECTION-CALL 117
    s118() # SECTION-CALL 118
    s119() # SECTION-CALL 119
    s120() # SECTION-CALL 120
    s121() # SECTION-CALL 121
    s122() # SECTION-CALL 122
    s123() # SECTION-CALL 123
    s124() # SECTION-CALL 124
    s125() # SECTION-CALL 125
    s126() # SECTION-CALL 126
    s127() # SECTION-CALL 127
    s128() # SECTION-CALL 128
    s129() # SECTION-CALL 129
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
