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
# import/from-import (hence no stdlib), metaclasses, eval/exec, __slots__-free
# introspection helpers, and builtins beyond the small set the feature test
# already leans on plus type/str/int/isinstance and the Exception hierarchy
# (ExceptionGroup appears once: except* cannot be exercised without it).
# Parent methods are therefore called explicitly (Base.m(self)) instead of via
# super(), and property/staticmethod/classmethod are replaced by a hand-rolled
# descriptor and plain functions. Complex literals ARE included: they are core
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
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
