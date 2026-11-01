# Full-syntax test: Python (3.12 core grammar) - part 7, sections 199-208 and
# 152: the data model's bound-method equality, three-argument pow's operand
# protocol, list.sort's timsort, the codecs with the surrogateescape handler,
# statistics' paired measures, a generator's gi_running and dir(), the
# attributes an iterator or a builtin refuses, and the escape, punycode and
# idna codecs with nameprep, and range as a type of its own. A file of its own
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

# ===== SECTION 205: dir() of a generator; an iterator and a builtin have no __dict__ =====
# dir() of a generator is object's names and the generator type's own, sorted.
# An iterator refuses a new attribute and its deletion like any builtin value,
# and so does a builtin or a bound method; only a plain function has a __dict__.
# Rows from CPython 3.14.
def s205():
    def gen():
        yield 1
    names = dir(gen())
    check("gendir", len(names) == 38 and names[:3] == ["__class__", "__class_getitem__", "__del__"]
          and [n for n in names if not n.startswith("__")]
          == ["close", "gi_code", "gi_frame", "gi_running", "gi_suspended", "gi_yieldfrom", "send", "throw"])
    def err(f):
        try:
            f()
        except AttributeError as e:
            return str(e)
        return "none"
    def setfoo(o):
        o.foo = 1
    def delfoo(o):
        del o.foo
    msgs = [err(lambda: setfoo(o)) for o in [iter([1]), iter("ab"), iter({1: 2}), zip([1]), enumerate([1])]]
    check("itattr", msgs == ["'" + n + "' object has no attribute 'foo' and no __dict__ for setting new attributes"
                             for n in ["list_iterator", "str_ascii_iterator", "dict_keyiterator", "zip", "enumerate"]])
    check("itdel", err(lambda: delfoo(iter((1,))))
          == "'tuple_iterator' object has no attribute 'foo' and no __dict__ for setting new attributes")
    def f():
        pass
    f.foo = 1
    xs = []
    check("fndel", err(lambda: delfoo(abs))
          == "'builtin_function_or_method' object has no attribute 'foo' and no __dict__ for setting new attributes"
          and err(lambda: delfoo(xs.append))
          == "'builtin_function_or_method' object has no attribute 'foo' and no __dict__ for setting new attributes"
          and err(lambda: delfoo(f)) == "none" and err(lambda: delfoo(f)) == "'function' object has no attribute 'foo'")

# ===== SECTION 206: unicode_escape, raw_unicode_escape, punycode and idna =====
# The escape codecs write what a string literal reads back and read Python's
# escapes (raw: only \u and \U); punycode is RFC 3492 with CPython's error
# positions, at any delta size; idna checks an ASCII name's labels. No bytes
# decode to "" under any encoding name. Rows from CPython 3.14.
def s206():
    def err(f):
        try:
            f()
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
        return "none"
    check("cdesc", "a\\\t\x00\x7f\xe9€\U0001f600'".encode("unicode_escape")
          == b"a\\\\\\t\\x00\\x7f\\xe9\\u20ac\\U0001f600'"
          and "\xe9€\ud800".encode("Raw-Unicode-Escape", "bogus") == b"\xe9\\u20ac\\ud800"
          and b"\\x41\\101\\u00e9\\U0001F600\\q\\\n\\a".decode("unicode_escape") == "AA\xe9\U0001f600\\q\x07"
          and b"\\u0041\\\\u0041\\x41\xe9".decode("raw_unicode_escape") == "A\\\\u0041\\x41\xe9")
    check("cdescerr", err(lambda: b"x\\x4g".decode("unicode_escape"))
          == "UnicodeDecodeError: 'unicodeescape' codec can't decode bytes in position 1-3: truncated \\xXX escape"
          and err(lambda: b"\\U00110000".decode("raw_unicode_escape"))
          == "UnicodeDecodeError: 'rawunicodeescape' codec can't decode bytes in position 0-9: \\Uxxxxxxxx out of range"
          and err(lambda: b"a\\".decode("unicode_escape"))
          == "UnicodeDecodeError: 'unicodeescape' codec can't decode byte 0x5c in position 1: \\ at end of string"
          and b"x\\u12zz".decode("unicode_escape", "replace") == "x�zz"
          and b"\\x4".decode("unicode_escape", "backslashreplace") == "\\x5c\\x78\\x34")
    check("cdpuny", "b\xfccher".encode("punycode") == b"bcher-kva" and "他们".encode("punycode") == b"8mqxb"
          and "\U0001f600x".encode("punycode") == b"x-iv3s" and "abc".encode("punycode") == b"abc-"
          and b"bcher-KVA".decode("punycode") == "b\xfccher" and b"b\xffcher-kva".decode("punycode", "replace") == "b�c\xeaher"
          and b"x-99999999999a".decode("punycode", "replace") == "x?")
    check("cdpunyerr", err(lambda: b"bcher-kv!".decode("punycode"))
          == "UnicodeDecodeError: 'punycode' codec can't decode byte 0x21 in position 8: Invalid extended code point '33'"
          and err(lambda: b"bcher-kv".decode("punycode"))
          == "UnicodeDecodeError: 'punycode' codec can't decode bytes in position 8-8: incomplete punycode string"
          and err(lambda: b"x-99999999999a".decode("punycode"))
          == "UnicodeDecodeError: 'punycode' codec can't decode bytes in position 4763888885386-4763888885386: Invalid character U+22a96f3cfc4"
          and err(lambda: b"999999999999999999a".decode("punycode")) == "OverflowError: Python int too large to convert to C ssize_t"
          and err(lambda: b"a".decode("punycode", "backslashreplace")) == "UnicodeError: Unsupported error handling: backslashreplace")
    check("cdidna", "Example.COM.".encode("idna") == b"Example.COM." and "a。b．c｡".encode("IDNA") == b"a.b.c."
          and b"a..b".decode("idna") == "a..b"
          and err(lambda: "a..b".encode("idna"))
          == "UnicodeEncodeError: 'idna' codec can't encode character '\\x2e' in position 2: label empty"
          and err(lambda: ("a." + "y" * 70).encode("idna"))
          == "UnicodeEncodeError: 'idna' codec can't encode characters in position 2-71: label too long"
          and err(lambda: b"xn--abc-.x".decode("idna"))
          == "UnicodeDecodeError: 'idna' codec can't decode bytes in position 0-7: IDNA does not round-trip, 'b'xn--abc-'' != 'b'abc''"
          and err(lambda: b"ok.xn--zzz9!".decode("idna"))
          == "UnicodeDecodeError: 'idna' codec can't decode byte 0x21 in position 11: Invalid extended code point '33'"
          and err(lambda: "x".encode("idna", "replace")) == "UnicodeError: Unsupported error handling: replace")
    check("cdempty", b"".decode("nonexistent") == "" and b"".decode("punycode", "bogus") == ""
          and err(lambda: "".encode("nonexistent")) == "LookupError: unknown encoding: nonexistent")

# ===== SECTION 207: range is a type of its own; the reverse iterators =====
# A range iterates, indexes and compares by value like a sequence, but is its
# own immutable type: repr range(a, b[, s]), slices that are ranges, start /
# stop / step, no + or * or ordering, and the hash of (len, start, step).
# iter and reversed answer CPython's per-source cursor types, and reversed of a
# user sequence reads __getitem__ lazily from len - 1 down. A dotted codec
# name matches an alias with its dots made underscores, never a module name.
# Rows from CPython 3.14.
class S207Seq:
    def __init__(self):
        self.got = []
    def __len__(self):
        return 3
    def __getitem__(self, i):
        self.got.append(i)
        if i == 0:
            raise IndexError("first")
        return i * 10

def s207():
    def err(f):
        try:
            f()
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
        return "none"
    args = [(3,), (1, 10, 2), (5, 2), (10, 0, -3)]
    rs = [range(*a) for a in args]
    check("rng1", [repr(r) for r in rs] == ["range(0, 3)", "range(1, 10, 2)", "range(5, 2)", "range(10, 0, -3)"]
          and str(rs[0]) == "range(0, 3)" and f"{[rs[1]]}" == "[range(1, 10, 2)]"
          and [(r.start, r.stop, r.step) for r in rs] == [(0, 3, 1), (1, 10, 2), (5, 2, 1), (10, 0, -3)]
          and [len(r) for r in rs] == [3, 5, 0, 4] and list(rs[3]) == [10, 7, 4, 1] and rs[1][2] == 5
          and hasattr(rs[0], "step") and not hasattr(rs[0], "append"))
    check("rng2", [type(r).__name__ for r in [rs[0], iter(rs[0]), reversed(rs[0])]] == ["range", "range_iterator", "range_iterator"]
          and isinstance(rs[0], range) and not isinstance(rs[0], list) and not isinstance(rs[0], tuple)
          and isinstance(enumerate([]), enumerate) and not isinstance(reversed([1]), reversed)
          and str(type(rs[1])) == "<class 'range'>")
    check("rng3", [repr(rs[1][1:]), repr(rs[1][::-1]), repr(rs[3][5:]), repr(rs[2][::-1]), repr(rs[3][::2])]
          == ["range(3, 11, 2)", "range(9, -1, -2)", "range(-2, -2, -3)", "range(4, 4, -1)", "range(10, -2, -6)"])
    check("rng4", rs[0] == range(0, 3, 1) and rs[0] != [0, 1, 2] and rs[0] != (0, 1, 2) and range(0) == range(5, 2)
          and range(0, 5, 3) == range(0, 6, 3) and hash(rs[0]) == hash((3, 0, 1)) and hash(range(5, 6, 7)) == hash((1, 5, None))
          and hash(rs[2]) == hash((0, None, None)) and {rs[0]: 1}[range(0, 3)] == 1
          and rs[1].index(5) == 2 and rs[1].count(4) == 0)
    def setit():
        rs[0][0] = 5
    def delit():
        del rs[0][0]
    def delsl():
        del rs[0][0:1]
    def aug():
        r = rs[0]
        r += [1]
    check("rng5", [err(lambda: rs[0] + rs[0]), err(lambda: [1] + rs[0]), err(lambda: rs[0] * 2), err(lambda: rs[0] < rs[1]),
                   err(setit), err(delit), err(delsl), err(aug), err(lambda: rs[0][5]), err(lambda: rs[0].index(9)),
                   err(lambda: rs[0].append(1))]
          == ["TypeError: unsupported operand type(s) for +: 'range' and 'range'",
              "TypeError: can only concatenate list (not \"range\") to list",
              "TypeError: unsupported operand type(s) for *: 'range' and 'int'",
              "TypeError: '<' not supported between instances of 'range' and 'range'",
              "TypeError: 'range' object does not support item assignment",
              "TypeError: 'range' object doesn't support item deletion",
              "TypeError: 'range' object does not support item deletion",
              "TypeError: unsupported operand type(s) for +=: 'range' and 'list'",
              "IndexError: range object index out of range", "ValueError: range.index(x): x not in range",
              "AttributeError: 'range' object has no attribute 'append'"])
    d = {1: 2}
    check("rng6", [type(reversed(v)).__name__ for v in [[1], (1,), "ab", b"a", d, d.keys(), d.values(), d.items()]]
          == ["list_reverseiterator", "reversed", "reversed", "reversed", "dict_reversekeyiterator",
              "dict_reversekeyiterator", "dict_reversevalueiterator", "dict_reverseitemiterator"]
          and [type(iter(v)).__name__ for v in [[1], (1,), b"a", bytearray(b"a")]]
          == ["list_iterator", "tuple_iterator", "bytes_iterator", "bytearray_iterator"])
    sq = S207Seq()
    rv = reversed(sq)
    first = sq.got == []
    check("rng7", first and type(rv).__name__ == "reversed" and list(rv) == [20, 10] and sq.got == [2, 1, 0])
    names = ["unicode.escape", "utf.8", "latin.1", "us.ascii", "iso.8859.2", "windows.1252", "ansi_x3.4_1968",
             "ansi_x3_4_1968", "iso_646.irv_1991", "iso_646_irv_1991", "rot.13", "kz.1048"]
    check("rng8", [err(lambda: "x".encode(n)) for n in names]
          == ["LookupError: unknown encoding: unicode.escape", "LookupError: unknown encoding: utf.8",
              "LookupError: unknown encoding: latin.1", "none", "none", "none", "none", "none", "none",
              "LookupError: unknown encoding: iso_646_irv_1991", "LookupError: unknown encoding: rot.13", "none"])

# ===== SECTION 208: idna's nameprep =====
# A non-ASCII label goes through nameprep (RFC 3491: map, Unicode 3.2's NFKC,
# prohibit, bidi) and then punycode behind "xn--"; decoding checks that the
# punycode encodes back. An error's position is within the nameprepped label,
# offset by the label's start, so it can lie past the input's end. Rows from
# CPython 3.14.
def s208():
    def err(f):
        try:
            f()
        except Exception as e:
            return type(e).__name__ + ": " + str(e)
        return "none"
    check("idna1", [s.encode("idna") for s in ["b\xfccher.de", "B\xdcCHER.DE.", "\ufb01x.example", "stra\xdfe.de",
                                                "a\xadb.\xfc", "m\xfcnchen\u3002de", "\u216b.\xfc", "\u0627\u0628.\xfc",
                                                "\U0002f868\xfc", "\u0221\xfc", "\u1e9e\xfc", "\u03a3\xfc", "\u3000\xfc"]]
          == [b"xn--bcher-kva.de", b"xn--bcher-kva.DE.", b"fix.example", b"strasse.de", b"ab.xn--tda",
              b"xn--mnchen-3ya.de", b"xii.xn--tda", b"xn--mgbc.xn--tda", b"xn--tda3962y", b"xn--tda99c",
              b"xn--ss-yka", b"xn--tda53l", b"xn-- -eha"])
    check("idna2", [err(lambda: s.encode("idna")) for s in ["\u0627a\u0628", "a\u0627", "\xfc\u2028x", "xn--\xfc",
                                                             "\xe9" + "a" * 70, "\ufdfc\U00010054"]]
          == ["UnicodeEncodeError: 'idna' codec can't encode character '\\x61' in position 1: Violation of BIDI requirement 2",
              "UnicodeEncodeError: 'idna' codec can't encode character '\\x61' in position 0: Violation of BIDI requirement 2",
              "UnicodeEncodeError: 'idna' codec can't encode character '\\u2028' in position 1: Invalid character '\\u2028'",
              "UnicodeEncodeError: 'idna' codec can't encode characters in position 0-3: Label starts with ACE prefix",
              "UnicodeEncodeError: 'idna' codec can't encode characters in position 0-70: label too long",
              "UnicodeEncodeError: 'idna' codec can't encode characters in position 4-4: Violation of BIDI requirement 3"])
    check("idna3", b"XN--BCHER-KVA.DE.".decode("idna") == "B\xfcCHER.DE."
          and b"xn--bcher-kva.xn--mnchen-3ya".decode("idna") == "b\xfccher.m\xfcnchen"
          and b"xn--d1acufc.xn--p1ai".decode("idna") == "\u0434\u043e\u043c\u0435\u043d.\u0440\u0444"
          and err(lambda: b"xn--strae-oqa.de".decode("idna"))
          == "UnicodeDecodeError: 'idna' codec can't decode bytes in position 0-12: IDNA does not round-trip, 'b'xn--strae-oqa'' != 'b'strasse''"
          and err(lambda: b"caf\xc3\xa9.xn--bcher-kva".decode("idna"))
          == "UnicodeDecodeError: 'idna' codec can't decode byte 0xc3 in position 3: ordinal not in range(128)")

def main():
    s199() # SECTION-CALL 199
    s200() # SECTION-CALL 200
    s201() # SECTION-CALL 201
    s202() # SECTION-CALL 202
    s152() # SECTION-CALL 152
    s203() # SECTION-CALL 203
    s204() # SECTION-CALL 204
    s205() # SECTION-CALL 205
    s206() # SECTION-CALL 206
    s207() # SECTION-CALL 207
    s208() # SECTION-CALL 208
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
