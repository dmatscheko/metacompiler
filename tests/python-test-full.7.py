# Full-syntax test: Python (3.12 core grammar) - part 7, sections 199-204 and
# 152: the data model's bound-method equality, three-argument pow's operand
# protocol, list.sort's timsort, the codecs with the surrogateescape handler,
# statistics' paired measures and a generator's gi_running. A file of its own
# because tests/python-test-full.py is close to the step brake (-max-steps, 100M
# per call) of the -frozen interpreter leg; see tests/python-test-full.py for
# the conventions.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 199: bound-method equality =====
# Two bound methods are equal when their receivers are the same object and their
# functions are equal (a builtin method's: the same method of the same object),
# and they hash alike, so a dict or set finds one by the other. Rows from
# CPython 3.14.
class S199A:
    def m(self): return 1
    def n(self): return 2
    @classmethod
    def c(cls): return 3

def s199():
    a = S199A()
    b = S199A()
    xs = [1]
    ys = [1]
    check("bmeq1", a.m == a.m and not (a.m != a.m) and a.m != b.m and a.m != a.n)
    check("bmeq2", S199A.c == S199A.c and xs.append == xs.append and xs.append != ys.append and xs.append != xs.pop)
    check("bmeq3", hash(a.m) == hash(a.m) and hash(xs.append) == hash(xs.append) and [a.m] == [a.m])
    check("bmeq4", len({a.m, a.m}) == 1 and {a.m: 5}.get(a.m) == 5 and a.m in {a.m: 1})

# ===== SECTION 200: three-argument pow =====
# CPython's ternary_op offers each operand's type in turn: int computes only when
# all three are ints, float refuses any modulus, complex refuses one once the
# base and exponent are numbers, and anything else declines. Rows from CPython
# 3.14.
def s200():
    def err(a, b, m):
        try:
            pow(a, b, m)
        except (TypeError, ValueError) as e:
            return type(e).__name__ + ": " + str(e)
        return "none"
    check("pow3a", err(2, 3, int) == "TypeError: unsupported operand type(s) for ** or pow(): 'int', 'int', 'type'")
    check("pow3b", err("a", 3, 2) == "TypeError: unsupported operand type(s) for ** or pow(): 'str', 'int', 'int'")
    check("pow3c", err(2, 3, 2.0) == "TypeError: pow() 3rd argument not allowed unless all arguments are integers"
          and err("a", 3, 2.0) == err(2, 3, 2.0))
    check("pow3d", err(2j, 3, 2) == "ValueError: complex modulo" and err(2, 3, 1j) == "ValueError: complex modulo"
          and err("a", 2, 1j) == "TypeError: unsupported operand type(s) for ** or pow(): 'str', 'int', 'complex'")
    check("pow3e", pow(3, 4, 5) == 1 and pow(True, 2, 3) == 1)

# ===== SECTION 201: list.sort's timsort =====
# sort() makes exactly timsort's comparisons, which a NaN (no order at all) and
# a counting __lt__ both observe; a raising comparison leaves some permutation
# of the items, and keys are computed in order before the sort. Rows from
# CPython 3.14.
class S201C:
    calls = [0]
    def __init__(self, v): self.v = v
    def __lt__(self, o):
        S201C.calls[0] += 1
        return self.v < o.v

def s201():
    nan = float("nan")
    r1 = sorted([3, nan, 1, 2])
    r2 = sorted([2, 1, nan, 0, 5, 4])
    check("tsnan", r1[0] == 3 and r1[1] != r1[1] and r1[2:] == [1, 2]
          and r2[0] == 1 and r2[1] != r2[1] and r2[2:] == [0, 2, 4, 5])
    cs = [S201C((i * 37) % 101) for i in range(300)]
    S201C.calls[0] = 0
    cs.sort()
    n1 = S201C.calls[0]
    S201C.calls[0] = 0
    ds = sorted(cs, reverse=True)
    check("tscount", n1 == 2099 and S201C.calls[0] == 497 and [c.v for c in ds[:3]] == [100, 100, 100])
    l = [3, 1, 2, "a", 0]
    try:
        l.sort()
    except TypeError:
        pass
    check("tsperm", l == [1, 2, 3, "a", 0])
    order = []
    def k(x):
        order.append(x)
        return x % 3
    check("tskey", sorted([5, 3, 9, 1, 4], key=k, reverse=True) == [5, 1, 4, 3, 9] and order == [5, 3, 9, 1, 4])

# ===== SECTION 202: utf-16, utf-32, cp1252 and surrogateescape =====
# The two UTF codecs write a byte order mark and read one, their -le and -be
# forms do neither, cp1252 is the charmap codec, and surrogateescape carries an
# undecodable byte through a str as U+DC80..U+DCFF. Rows from CPython 3.14.
def s202():
    def err(f):
        try:
            f()
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
        return "none"
    check("cd16", "h\u00e9\U0001f600".encode("utf-16") == b"\xff\xfeh\x00\xe9\x00=\xd8\x00\xde"
          and "a".encode("UTF_16BE") == b"\x00a" and b"\xfe\xff\x00a\xd8=\xde\x00".decode("utf-16") == "a\U0001f600")
    check("cd32", "a\u20ac".encode("utf-32") == b"\xff\xfe\x00\x00a\x00\x00\x00\xac\x20\x00\x00"
          and b"\x00\x00\x00a".decode("utf-32-be") == "a" and "a".encode("u32") == b"\xff\xfe\x00\x00a\x00\x00\x00")
    check("cdcp", "\u20ac\u2030\xe9".encode("cp1252") == b"\x80\x89\xe9" and b"\x80\x9f".decode("windows-1252") == "\u20ac\u0178")
    check("cderr1", err(lambda: b"\x00\xd8a\x00".decode("utf-16"))
          == "UnicodeDecodeError: 'utf-16-le' codec can't decode bytes in position 0-1: illegal UTF-16 surrogate"
          and err(lambda: b"a\x00\x00".decode("utf-32"))
          == "UnicodeDecodeError: 'utf-32-le' codec can't decode bytes in position 0-2: truncated data")
    check("cderr2", err(lambda: "\u20ad\u20ae".encode("cp1252"))
          == "UnicodeEncodeError: 'charmap' codec can't encode characters in position 0-1: character maps to <undefined>"
          and err(lambda: b"\x81".decode("cp1252"))
          == "UnicodeDecodeError: 'charmap' codec can't decode byte 0x81 in position 0: character maps to <undefined>"
          and err(lambda: "a\ud800".encode("utf-16-le"))
          == "UnicodeEncodeError: 'utf-16-le' codec can't encode character '\\ud800' in position 1: surrogates not allowed")
    check("cdse1", b"a\xff\xfe".decode("utf-8", "surrogateescape") == "a\udcff\udcfe"
          and "a\udcff\udc80".encode("utf-8", "surrogateescape") == b"a\xff\x80"
          and "a\udc80\udcffb".encode("cp1252", "surrogateescape") == b"a\x80\xffb")
    check("cdse2", err(lambda: "a\udcff\udc41\udc90".encode("utf-8", "surrogateescape"))
          == "UnicodeEncodeError: 'utf-8' codec can't encode characters in position 2-3: surrogates not allowed"
          and err(lambda: b"\xff\xfea\x00".decode("utf-32", "surrogateescape"))
          == "UnicodeDecodeError: 'utf-32-le' codec can't decode bytes in position 2-3: truncated data")
    check("cdhandler", b"\x80\x81\x8d".decode("utf-16", "replace") == "\u8180\ufffd"
          and err(lambda: b"\xff".decode("utf-8", "xmlcharrefreplace"))
          == "TypeError: don't know how to handle UnicodeDecodeError in error callback")

# ===== SECTION 152: statistics.covariance, correlation, linear_regression =====
# Built on math.sumprod; every value here is CPython 3.14's.
import statistics as S152st

def s152():
    x = [1, 2, 3, 4, 5, 6, 7, 8, 9]
    y = [1, 2, 3, 1, 2, 3, 1, 2, 3]
    z = [2.5, -1.25, 7.0, 3.3, 0.1, 9.9, -4.0, 2.2, 5.5]
    lr = S152st.linear_regression(x, z)
    check("stat2", S152st.covariance(x, y) == 0.75 and S152st.covariance(x, z) == 0.8687500000000001
          and S152st.correlation(x, z) == 0.0742632702055162 and S152st.correlation(x, [9 - v for v in x]) == -1.0
          and S152st.correlation([1, 2, 2, 3], [5, 6, 6, 8], method="ranked") == 1.0
          and repr(lr) == "LinearRegression(slope=0.11583333333333336, intercept=2.2263888888888888)"
          and S152st.linear_regression(x, z, proportional=True).slope == 0.46736842105263166)
    msgs = []
    for f in [lambda: S152st.covariance([1], [2]), lambda: S152st.correlation([1, 1], [1, 2]),
              lambda: S152st.linear_regression([2, 2], [1, 3])]:
        try:
            f()
        except S152st.StatisticsError as e:
            msgs.append(str(e))
    check("stat3", msgs == ["covariance requires at least two data points", "at least one of the inputs is constant",
                            "x is constant"])

# ===== SECTION 203: utf-7, utf-8-sig, the charmap codecs and the refused names =====
# utf-7 is RFC 2152 as CPython writes and reads it (its errors name the codec
# "utf7"), every single-byte charmap codec decodes through its table and
# encodes through the inverse (of two bytes for one character the higher), and
# a codec that is not a text encoding is a LookupError naming the encoding as
# given. Rows from CPython 3.14.
def s203():
    def err(f):
        try:
            f()
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
        return "none"
    check("cdu7", "\xe9".encode("utf-7") == b"+AOk-"
          and "a+b-c~\\ \xe9€\U0001f600 x".encode("UTF7") == b"a+-b-c+AH4AXA +AOkgrNg93gA x"
          and b"+AOk-x+-".decode("utf-7") == "\xe9x+" and b"+2D3cAA-".decode("u7") == "\U0001f400"
          and b"+AOk.x".decode("utf-7") == "\xe9.x" and b"a\x80b".decode("utf-7", "replace") == "a�b")
    check("cdu7err", err(lambda: b"+AOl-".decode("utf-7"))
          == "UnicodeDecodeError: 'utf7' codec can't decode bytes in position 0-4: non-zero padding bits in shift sequence"
          and err(lambda: b"x+A".decode("utf-7"))
          == "UnicodeDecodeError: 'utf7' codec can't decode bytes in position 1-2: unterminated shift sequence"
          and err(lambda: b"+A-".decode("utf-7"))
          == "UnicodeDecodeError: 'utf7' codec can't decode bytes in position 0-2: partial character in shift sequence"
          and err(lambda: b"+!".decode("utf-7"))
          == "UnicodeDecodeError: 'utf7' codec can't decode bytes in position 0-1: ill-formed sequence")
    check("cdmap", "\xe9".encode("cp437") == b"\x82" and "\xe9".encode("mac-roman") == b"\x8e"
          and b"\xff".decode("IBM437") == "\xa0" and "Жж".encode("koi8-r") == b"\xf6\xd6"
          and b"\xc1".decode("cp037") == "A" and " ".encode("mac_arabic") == b"\xa0"
          and "Ωπ".encode("greek") == b"\xd9\xf0"
          and "ab€Ā".encode("latin2", "xmlcharrefreplace") == b"ab&#8364;&#256;")
    check("cdmaperr", err(lambda: "€".encode("cp437"))
          == "UnicodeEncodeError: 'charmap' codec can't encode character '\\u20ac' in position 0: character maps to <undefined>"
          and err(lambda: b"\xa5".decode("iso8859_3"))
          == "UnicodeDecodeError: 'charmap' codec can't decode byte 0xa5 in position 0: character maps to <undefined>"
          and b"\x80\xa5x".decode("iso8859-3", "backslashreplace") == "\x80\\xa5x")
    check("cdsig", "\xe9".encode("utf-8-sig") == b"\xef\xbb\xbf\xc3\xa9"
          and b"\xef\xbb\xbf\xc3\xa9".decode("utf_8_sig") == "\xe9" and b"x\xef\xbb\xbf".decode("utf-8-sig") == "x﻿"
          and err(lambda: b"\xef\xbb\xbf\xff".decode("utf-8-sig"))
          == "UnicodeDecodeError: 'utf-8' codec can't decode byte 0xff in position 0: invalid start byte")
    check("cdrefuse", err(lambda: "x".encode("Base-64"))
          == "LookupError: 'Base-64' is not a text encoding; use codecs.encode() to handle arbitrary codecs"
          and err(lambda: b"x".decode("hex"))
          == "LookupError: 'hex' is not a text encoding; use codecs.decode() to handle arbitrary codecs"
          and err(lambda: "x".encode("undefined")) == "UnicodeError: undefined encoding"
          and err(lambda: "x".encode("utf8-sig")) == "LookupError: unknown encoding: utf8-sig")

# ===== SECTION 204: a generator's gi_running =====
# True only while the generator's own frame executes - read from inside its
# body, also when a throw() resumes it - and read-only; neither a generator
# nor a tuple takes a new attribute. Rows from CPython 3.14.
def s204():
    seen = []
    def gen():
        seen.append(g.gi_running)
        x = yield 1
        seen.append(g.gi_running)
        try:
            yield 2
        except ValueError:
            seen.append(g.gi_running)
            yield 3
    g = gen()
    seen.append(g.gi_running)
    next(g)
    seen.append(g.gi_running)
    g.send(5)
    g.throw(ValueError)
    g.close()
    seen.append(g.gi_running)
    check("girun", seen == [False, True, False, True, True, False] and hasattr(g, "gi_running")
          and type(getattr(g, "gi_running")).__name__ == "bool")
    msgs = []
    for f in [lambda: setattr(g, "gi_running", True), lambda: setattr(g, "nope", 1), lambda: setattr((1,), "nope", 1)]:
        try:
            f()
        except AttributeError as e:
            msgs.append(str(e))
    check("giro", msgs == ["attribute 'gi_running' of 'generator' objects is not writable",
                           "'generator' object has no attribute 'nope' and no __dict__ for setting new attributes",
                           "'tuple' object has no attribute 'nope' and no __dict__ for setting new attributes"])

def main():
    s199() # SECTION-CALL 199
    s200() # SECTION-CALL 200
    s201() # SECTION-CALL 201
    s202() # SECTION-CALL 202
    s152() # SECTION-CALL 152
    s203() # SECTION-CALL 203
    s204() # SECTION-CALL 204
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
