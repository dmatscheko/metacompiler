# Full-syntax test: Python (3.12 core grammar) - part 7, sections 199-202: the
# data model's bound-method equality, three-argument pow's operand protocol,
# list.sort's timsort and the utf-16, utf-32 and cp1252 codecs with the
# surrogateescape handler. A file of its own because tests/python-test-full.py is
# close to the step brake (-max-steps, 100M per call) of the -frozen interpreter
# leg; see tests/python-test-full.py for the conventions.

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

def main():
    s199() # SECTION-CALL 199
    s200() # SECTION-CALL 200
    s201() # SECTION-CALL 201
    s202() # SECTION-CALL 202
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
