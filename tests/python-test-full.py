# Full-syntax test: Python (3.12 core grammar).
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
# metaclasses, eval/exec, __slots__-free
# introspection helpers, and builtins beyond the small set the feature test
# already leans on plus type/str/int/isinstance and the Exception hierarchy
# (ExceptionGroup appears once: except* cannot be exercised without it).
# Parent methods are therefore called explicitly (Base.m(self)) instead of via
# super(), and property/staticmethod/classmethod are replaced by a hand-rolled
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
    check("re02", m.start() == 3 and m.end() == 8 and m.span() == [3, 8])
    check("re03", m.span(1) == [3, 5] and len(m.groups()) == 2)
    n = re.search(r"(?P<k>[a-z]+)=(?P<v>\d+)", "x: size=42")
    check("re04", n.group("k") == "size" and n.group("v") == "42")
    check("re05", n.groupdict()["k"] == "size" and n.groupdict()["v"] == "42")
    # match anchors at the start, fullmatch at both ends - and fullmatch must
    # BACKTRACK into an alternation, which a leftmost search plus a length check
    # cannot do: re.fullmatch("a|ab", "ab") is "ab", not None.
    check("re06", re.match(r"\d", "a1") == None and re.match(r"a", "a1").group(0) == "a")
    check("re07", re.fullmatch(r"a|ab", "ab").group(0) == "ab" and re.fullmatch(r"a", "ab") == None)
    check("re08", re.findall(r"\d+", "a1 b22 c333") == ["1", "22", "333"])
    check("re09", re.findall(r"(\w)=(\d)", "a=1,b=2") == [["a", "1"], ["b", "2"]])
    check("re10", re.sub(r"\s+", " ", "a   b \t c") == "a b c")
    check("re11", re.sub(r"(\w)(\d)", r"\2\1", "a1 b2") == "1a 2b")
    check("re12", re.sub(r"(\d)", r"[\g<1>]", "a1") == "a[1]")
    check("re13", re.subn(r"o", "0", "foo boo") == ["f00 b00", 4])
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
    # rp2 and rp3 pin OUR default, "<Name object>": CPython writes
    # "<__main__.Name object at 0x...>" and no engine here can reproduce the
    # address, so those two right-hand sides are deliberately not CPython's.
    check("rp1", str(S27V(3)) == "V(3)" and repr(S27V(3)) == "V(3)" and str([S27V(3)]) == "[V(3)]")
    check("rp2", str(S27W(1)) == "W!1" and repr(S27W(1)) == "<S27W object>")
    check("rp3", str(S27X()) == "<S27X object>" and str([S27X()]) == "[<S27X object>]")
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
# An ARRAY is read as the argument TUPLE exactly when its length equals the
# conversion count, and as a single value otherwise - because a tuple IS a list in
# this value model. pct20 is that rule from the "single value" side.
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
    check("su15", p.popitem() == ["c", 3] and len(p) == 2)
    check("su16", p.popitem() == ["b", 2] and list(p.keys()) == ["a"])
    check("su17", p.popitem() == ["a", 1] and p == {})
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
    # CPython qualifies a class defined inside a function as
    # 's39.<locals>.Bm.mth'; no engine here records the DEFINING scope, so the
    # row asserts the SUFFIX, which is exact in CPython and in all three engines.
    check("it59", bnd.__name__ == "mth" and bnd.__qualname__.endswith("Bm.mth"))
    check("it60", bnd.__self__ is bi and type(bnd).__name__ == "method")
    check("it61", bnd.__func__ is Bm.mth and bnd(1) == 2)
    # How the two forms PRINT. CPython appends the receiver's address to both and
    # the address cannot be matched, so it is left off - the same rule <function f>
    # and <map object> already follow. The rest is byte-for-byte CPython.
    check("it62", str(bl).startswith("<built-in method count of list object"))
    check("it63", str(bnd).startswith("<bound method ") and "Bm.mth of <" in str(bnd))
    # __qualname__ of a plain function and of a class: EXACT for a module-level
    # def, a lambda, a builtin and a builtin type. The one shape it is wrong for
    # is a function read off a class body - CPython says 'Bm.mth' and this says
    # 'mth' - because no engine records the DEFINING scope at a def site.
    check("it64", sorted_like.__qualname__.endswith("sorted_like") and lam.__qualname__.endswith("<lambda>"))
    check("it65", len.__qualname__ == "len" and str.__qualname__ == "str" and map.__qualname__ == "map")
    check("it66", Bm.__qualname__.endswith("Bm") and Bm.mth.__qualname__.endswith("mth"))

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


# ===== SECTION 41: case mapping of characters whose simple and full maps agree =====
# CPython does FULL Unicode case mapping ("\u00df".upper() is "SS"); this
# project answers the simple per-code-point map, so only characters on which the
# two agree are asserted here: dotless i, final sigma, capital sharp s.
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


# ===== SECTION 42: an unhashable set member or dict key =====
# `a = [1]; b = [1]; len({a, b})` raises TypeError in CPython. Two separate
# things follow from that row.
#
# 1. dict and set ARE refused, in all three engines, with CPython 3.14's exact
#    wording - which differs between the two positions:
#      {[1]}    cannot use 'list' as a set element (unhashable type: 'list')
#      {{}: 1}  cannot use 'dict' as a dict key (unhashable type: 'dict')
#    with one exception CPython really has: set.__contains__ / .discard / .remove
#    convert a SET argument to a frozenset instead of refusing it, so `{1} in {1}`
#    is False while `{} in {1}` and `{1} in {1: 2}` both raise.
#
# 2. The LIST is deliberately NOT refused, and that is the divergence this section
#    records. A tuple IS a list in this value model, and the two directions of
#    the mistake are not symmetric: treating a tuple as hashable only ACCEPTS a
#    program CPython would reject, while treating a list as unhashable would
#    abort `{(1, 2), (3, 4)}` and `d[i, j]` - legal Python, and among the most
#    common things Python programs do. So the array stays hashable, and a tuple
#    key compares by CONTENTS in all three engines (core.keyEq, pyKeyEq,
#    dictKeyEq), which is CPython's rule for a tuple (`d[1, 2] = v; d[1, 2]`).
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

    # --- RECORDED DIVERGENCE: a list is hashable here, because it is a tuple.
    # CPython raises for every row below.
    # They CATCH rather than assert directly, so this section still runs to the
    # end under CPython instead of aborting on the first of them.
    def lsetlen():
        try:
            return len({[1], [1], [2]})
        except TypeError:
            return 0 - 1
    check("uh26", msg(lambda: {a_list}) == "no raise")
    check("uh27", msg(lambda: {a_list: 1}) == "no raise")
    check("uh28", lsetlen() == 2)

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
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
