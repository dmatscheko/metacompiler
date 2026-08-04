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
# Every check here was a live divergence between python-interpreter.abnf and
# python-to-llvm-ir.abnf. The matrix cannot see this class of bug - it compares
# each engine against itself under goja and -frozen - so the assertions live here.
# Expected values are CPython 3's.
def s23():
    # Sequence repetition. The compiler fell through to numeric arithmetic and
    # answered NaN for every one of these; the interpreter always repeated.
    check("agr1", "ab" * 3 == "ababab")
    check("agr2", [0] * 3 == [0, 0, 0])
    check("agr3", 3 * [0] == [0, 0, 0])
    check("agr4", [1, 2] * 0 == [])
    check("agr5", "x" * 0 == "")
    # Conversions the interpreter had and the compiler did not.
    check("agr6", bool(0) == False and bool([]) == False and bool("x") == True)
    check("agr7", float("1.5") == 1.5 and float(2) == 2)
    check("agr8", repr("x") == "'x'")
    check("agr9", len(set([1, 2, 2])) == 2)
    check("agr10", sum([1, 2, 3]) == 6 and sum([]) == 0)
    check("agr11", len(tuple([1, 2])) == 2)
    # max/min: the COMPILER had them and the interpreter did not.
    check("agr12", max(1, 2) == 2 and min(1, 2) == 1)
    check("agr13", max([3, 1, 2]) == 3 and min([3, 1, 2]) == 1)

# ===== SECTION 24: the syntax the reference corpus asked for =====
# Every check below is a construct CPython's OWN test suite writes and one or
# both grammars could not parse. They took the cpython-tests corpus from
# 53.3% / 23.3% (interpreter / compiler) to 83.3% / 86.7%, so this section is
# the ratchet that keeps them. Grouped by the gap each one closes.
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
    # form of this did not even fail - it ended the field early and rendered the rest
    # of the expression as literal text.
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
# holds ONE entry for the two of them. Every check here was wrong in ALL THREE
# engines at 9689f81 - the interpreter, llvm.Run and the native binary agreed with
# each other and disagreed with CPython - so neither the --full halves comparison
# nor --cross nor the byte-identity matrix could see it. That is the reason for a
# section of its own next to 23: 23 is "the halves disagreed", this is "the halves
# agreed and were both wrong". Expected values are CPython 3.14.6's.
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
# BOXED value in all three engines - and every check here was wrong in ALL THREE
# at 913bd95, which is why this sits next to 25 rather than in 23: the halves
# agreed with each other and disagreed with CPython. Expected values are
# CPython 3.14.6's, and the renderer window (exponential iff
# decpt <= -4 or decpt > 16, no forced ".0" in that form) is Python's own -
# ruby, java and dart each use a different one.
def s26():
    # The type, and the literal forms that produce it. type() is compared
    # through str() because binding the builtin type OBJECTS is a separate
    # compiler-half gap (isinstance(x, str) is False there too).
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
    # NaN in the frozen tag engine and Infinity under goja, which is a FROZEN-DIFF
    # of its own and not this section's subject.
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

# ===== SECTION 27: the six defects the float-box probes found and left =====
# Every check here was measured wrong at f2f8926, and each names which half was
# wrong. Expected values are CPython 3.14.6's.
def s27():
    # float(str) is CPython's float_from_string, not JavaScript's parseFloat: it
    # takes the three non-finite SPELLINGS, case-insensitively and with an
    # optional sign. float("inf") was nan in ALL THREE engines, and "Infinity"
    # was inf in the interpreter and nan in the compiler - a halves divergence on
    # top of the defect. Infinity is BUILT (1e308 * 10) rather than written,
    # because parseFloat("1e400") used to be a FROZEN-DIFF of its own.
    inf = 1e308 * 10
    check("nf1", float("inf") == inf and float("-inf") == 0 - inf)
    check("nf2", float("INFINITY") == inf and float("infinity") == inf and float("-Inf") == 0 - inf)
    check("nf3", str(float("nan")) == "nan" and str(float("-nan")) == "nan" and str(float("NaN")) == "nan")
    check("nf4", float("  inf  ") == inf and str(float("1.5")) == "1.5")
    # The builtin TYPE OBJECTS. `int` is the class, not a conversion function, so
    # type(1) IS int - and it was not, under the compiler only: declBuiltin bound
    # the name to the conversion closure and the class object was reachable only
    # through type(). Six for six wrong there, all six right in the interpreter.
    check("ty1", isinstance("a", str) and isinstance(1, int) and isinstance(1.0, float))
    check("ty2", isinstance([1], list) and isinstance({}, dict) and isinstance(True, bool))
    check("ty3", type(1) == int and type(1) is int and type("a") is str)
    check("ty4", type(1.0) is float and type([1]) is list)
    # ...and calling the name still converts, because the conversion closure rides
    # on the class object as __conv.
    check("ty5", int("42") == 42 and str(float("1.5")) == "1.5" and list("ab") == ["a", "b"])
    check("ty6", str(int) == "<class 'int'>" and str(str) == "<class 'str'>")
    # __repr__ / __str__ under the COMPILER, which had no user-object rendering
    # arm at all and printed [object Object] for every instance. CPython prefers
    # __str__ for str()/print() and __repr__ for repr() and container elements.
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
    # A Python int is ARBITRARY PRECISION, so ** stays exact. The interpreter
    # answered 5076944270305264000 for 10**30 (the int64 its tag engine wrapped
    # the double into) and the compiler folded 1e+30 - wrong twice and divergent.
    check("bi1", str(10 ** 30) == "1000000000000000000000000000000")
    check("bi2", str(2 ** 70) == "1180591620717411303424" and str(2 ** 63) == "9223372036854775808")
    check("bi3", str(3 ** 40) == "12157665459056928801" and 10 ** 18 == 1000000000000000000)
    check("bi4", str(2 ** -1) == "0.5" and 2 ** 10 == 1024 and str(type(2 ** 10)) == "<class 'int'>")
    # ...and an int operand meeting a big PROMOTES, which the compiled halves did
    # not do at all: +, -, *, //, % and every ORDER comparison were NaN or False
    # there whenever one side was a plain int.
    check("bi5", str(10 ** 30 + 1) == "1000000000000000000000000000001")
    check("bi6", str(1 + 10 ** 30) == "1000000000000000000000000000001" and str(10 ** 30 * 3) == "3000000000000000000000000000000")
    check("bi7", str(10 ** 30 // 7) == "142857142857142857142857142857" and str(10 ** 30 % 7) == "1")
    check("bi8", str((0 - 10 ** 30) // 7) == "-142857142857142857142857142858" and str((0 - 10 ** 30) % 7) == "6")
    check("bi9", str(10 ** 30 % (0 - 7)) == "-6" and 10 ** 30 > 1 and 1 < 10 ** 30 and not (10 ** 30 <= 1))
    # An unreached annotation followed by a reached one: the compiler answered
    # "is there an __annotations__ dict here" with a STICKY static flag, so the
    # dict was never created and the class body died with
    # "variable not defined: __annotations__".
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
# "fmt" % args was UNIMPLEMENTED in both halves - binOp("%") on a string fell into
# fmod, so "%s" % "hi" was NaN. The halves agreed, so --cross was blind to it.
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
    # CPython's. The interpreter used to scale and Math.round, i.e. half UP, so
    # f"{2.5:.0f}" was "3" here and "2" in both compiled halves - a live halves
    # divergence no test reached.
    check("pct21", f"{2.5:.0f}" == "2" and f"{3.5:.0f}" == "4" and f"{0.125:.2f}" == "0.12")
    check("pct22", ("%.0f %.1f" % (2.5, 0.05)) == "2 0.1")
    # ...and the {:x} spec: (255).toString(16) works under goja and ABORTS under
    # -frozen, so this was a live FROZEN-DIFF in the interpreter that no test
    # reached. The matrix's goja/-frozen byte-identity check is the gate.
    check("pct23", f"{255:x} {255:X} {8:o} {5:b}" == "ff FF 10 101")

# ===== SECTION 29: the bitwise operators are ARBITRARY PRECISION =====
# |, &, ^, ~, << and >> used to be ECMAScript's, i.e. ToInt32 with the shift
# count masked to five bits: -1 & 0xffffffff was -1 against CPython's
# 4294967295, 1 << 40 was 256, and every operand past 2^31 was truncated.
# BOTH HALVES AGREED on all of it, so --cross and the matrix were structurally
# blind - the defect class only an external oracle can reach. 439 of 944 rows
# of a differential probe were wrong. Python's rule is INFINITE two's
# complement over arbitrary precision ints; every value below is CPython
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
    # str() because a DECIMAL LITERAL of 9007199254740993 is a separate, still
    # open gap - pyIsBigText boxes on `parseFloat(txt) > 2**53`, and parseFloat
    # rounds that text down to exactly 2**53, so the literal is not boxed.
    check("bit11", str((2 ** 53) | 1) == "9007199254740993" and str((2 ** 53) ^ 1) == "9007199254740993")
    # bool & bool is a BOOL; a bool meeting an int is an int, and True << 1 is 2.
    check("bit12", (True & True) is True and (True | False) is True and (True ^ True) is False)
    check("bit13", (True & 1) == 1 and not ((True & 1) is True) and (True << 1) == 2)
    check("bit14", (v[10] << 45) == 8972014882652160 and (v[10] << 46) == 17944029765304320)
    check("bit15", (v[7] >> 200) == 0 and (v[4] >> 200) == -1 and (v[2] << 99) == 633825300114114700748351602688)

# ===== END SECTIONS =====

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
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
