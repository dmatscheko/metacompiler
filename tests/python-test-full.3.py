# Full-syntax test: Python (3.12 core grammar) - part 3, the library-module
# sections 131, 134, 136, 138, 143, 144 and 145.
#
# Each of these compiles whole Python-source modules (lib/python-*.py), and
# seeding the random module's MT19937 alone runs some sixty million IR steps in
# the -frozen interpreter leg, so they have a file of their own to keep every
# file clear of the step brake (-max-steps, 100M per call); see
# tests/python-test-full.py for the conventions, which are the same.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 131: heapq, bisect and collections.ChainMap =====
# A heap's list layout is CPython's item for item (the same sift algorithm);
# every value here is CPython 3.14's.
import heapq
import bisect
from collections import ChainMap

def s131():
    h = []
    for v in [5, 1, 8, 3, 2, 9, 0, 7]:
        heapq.heappush(h, v)
    xs = [9, 4, 7, 1, 8, 2, 6]
    heapq.heapify(xs)
    check("heapq1", h == [0, 2, 1, 5, 3, 9, 8, 7] and xs == [1, 4, 2, 9, 8, 7, 6]
          and [heapq.heappop(h) for _ in range(3)] == [0, 1, 2] and h == [3, 5, 7, 9, 8])
    m = [3, 1, 4, 1, 5]
    heapq.heapify_max(m)
    check("heapq2", heapq.heappushpop(xs, 0) == 0 and heapq.heapreplace(xs, 10) == 1 and xs == [2, 4, 6, 9, 8, 7, 10]
          and m == [5, 3, 4, 1, 1] and heapq.heappop_max(m) == 5
          and list(heapq.merge([1, 4], [2, 3], [0])) == [0, 1, 2, 3, 4]
          and list(heapq.merge([5, 2], [4, 1], reverse=True)) == [5, 4, 2, 1]
          and heapq.nlargest(2, ["a", "bbb", "cc", "dd"], key=len) == ["bbb", "cc"]
          and heapq.nsmallest(2, [4, 1, 3]) == [1, 3])
    try:
        heapq.heappop([])
        check("heapq3", False)
    except IndexError as e:
        check("heapq3", str(e) == "index out of range")
    a = [1, 2, 4, 4, 7]
    bisect.insort(a, 4)
    bisect.insort_left(a, 5)
    recs = [("a", 1), ("c", 3)]
    bisect.insort(recs, ("b", 2), key=lambda r: r[1])
    check("bisect1", bisect.bisect_left(a, 4) == 2 and bisect.bisect_right(a, 4) == 5 and bisect.bisect(a, 0) == 0
          and bisect.bisect_left(a, 4, 0, 2) == 2 and a == [1, 2, 4, 4, 4, 5, 7]
          and recs == [("a", 1), ("b", 2), ("c", 3)] and bisect.bisect_left(recs, 3, key=lambda r: r[1]) == 2)
    try:
        bisect.bisect_left(a, 1, -1)
        check("bisect2", False)
    except ValueError as e:
        check("bisect2", str(e) == "lo must be non-negative")
    cm = ChainMap({"a": 1}, {"a": 2, "b": 3})
    cm["c"] = 4
    kid = cm.new_child({"a": 9})
    check("chainmap1", cm["a"] == 1 and cm["b"] == 3 and len(cm) == 3 and list(cm) == ["a", "b", "c"]
          and cm.maps[0] == {"a": 1, "c": 4} and kid["a"] == 9 and kid.parents["a"] == 1
          and repr(cm) == "ChainMap({'a': 1, 'c': 4}, {'a': 2, 'b': 3})" and cm.get("zz", 0) == 0
          and dict(cm) == {"a": 1, "b": 3, "c": 4} and not ChainMap() and str(ChainMap()) == "ChainMap({})")
    try:
        del cm["b"]
        check("chainmap2", False)
    except KeyError as e:
        check("chainmap2", str(e) == "\"Key not found in the first mapping: 'b'\"")

# ===== SECTION 134: json.JSONEncoder / JSONDecoder and cls= =====
import json as S134json

class S134Enc(S134json.JSONEncoder):
    def default(self, o):
        if isinstance(o, complex):
            return [o.real, o.imag]
        return S134json.JSONEncoder.default(self, o)

class S134Dec(S134json.JSONDecoder):
    def __init__(self, **kw):
        super().__init__(object_hook=self.hook, **kw)

    def hook(self, d):
        return {k.upper(): v for k, v in d.items()}

def s134():
    check("jsoncls1", S134json.dumps({"z": 1 + 2j}, cls=S134Enc, sort_keys=True) == '{"z": [1.0, 2.0]}'
          and S134Enc(indent=1).encode([1j]) == "[\n [\n  0.0,\n  1.0\n ]\n]"
          and S134json.JSONEncoder(separators=(",", ":")).encode([1, 2]) == "[1,2]"
          and S134json.loads('{"a": {"b": 1}}', cls=S134Dec) == {"A": {"B": 1}}
          and S134json.JSONDecoder().decode("[1, 2]") == [1, 2])
    try:
        S134json.dumps(object(), cls=S134Enc)
        check("jsoncls2", False)
    except TypeError as e:
        check("jsoncls2", str(e) == "Object of type object is not JSON serializable")

# ===== SECTION 136: contextlib, io.StringIO, types.SimpleNamespace, textwrap =====
import contextlib as S136cl
import io as S136io
import types as S136ty
import textwrap as S136tw

@S136cl.contextmanager
def S136tag(log, name):
    log.append("<" + name)
    try:
        yield name.upper()
    except KeyError:
        log.append("swallowed")
    finally:
        log.append(name + ">")

def s136():
    log = []
    with S136tag(log, "a") as t:
        log.append(t)
    with S136tag(log, "b"):
        raise KeyError("k")
    try:
        with S136tag(log, "c"):
            raise ValueError("v")
    except ValueError:
        log.append("propagated")
    with S136cl.suppress(ZeroDivisionError):
        1 / 0
    with S136cl.ExitStack() as st:
        st.callback(log.append, "cb")
        log.append(st.enter_context(S136tag(log, "d")))
    check("ctxlib1", log == ["<a", "A", "a>", "<b", "swallowed", "b>", "<c", "c>", "propagated",
                             "<d", "D", "d>", "cb"]
          and type(S136cl.suppress()).__name__ == "suppress")
    s = S136io.StringIO()
    print("x", 1, file=s)
    s.write("yz")
    s.seek(1)
    s.write("Q")
    s.seek(0)
    check("stringio1", s.getvalue() == "xQ1\nyz" and s.readline() == "xQ1\n" and s.read() == "yz"
          and S136io.StringIO("a\nb").readlines() == ["a\n", "b"])
    ns = S136ty.SimpleNamespace(a=1, b="x")
    ns.c = 2
    del ns.a
    check("simplens1", repr(ns) == "namespace(b='x', c=2)" and ns == S136ty.SimpleNamespace(b="x", c=2)
          and isinstance(len, S136ty.BuiltinFunctionType) and isinstance(s136, S136ty.FunctionType))
    check("textwrap1", S136tw.wrap("The quick brown fox jumps over the lazy dog", 12)
          == ["The quick", "brown fox", "jumps over", "the lazy dog"]
          and S136tw.wrap("well--known text-book e-mail", 9) == ["well--", "known", "text-book", "e-mail"]
          and S136tw.shorten("Hello  world, how are you?", 15) == "Hello [...]"
          and S136tw.dedent("  a\n    b\n\n  c") == "a\n  b\n\nc"
          and S136tw.indent("a\n\nb", "> ") == "> a\n\n> b"
          and S136tw.fill("one two three", 7, initial_indent="* ") == "* one\ntwo\nthree")

# ===== SECTION 138: typing at run time; __class_getitem__, __mro_entries__ =====
import typing as S138t

S138T = S138t.TypeVar("S138T")

class S138Box(S138t.Generic[S138T]):
    def __init__(self, v):
        self.v = v

@S138t.runtime_checkable
class S138Closer(S138t.Protocol):
    def close(self): ...

class S138Pt(S138t.NamedTuple):
    x: int
    y: int = 0

    def norm(self):
        return self.x + self.y

class S138Movie(S138t.TypedDict):
    name: str

class S138Entry:
    def __init__(self, origin):
        self.origin = origin

    def __mro_entries__(self, bases):
        return (self.origin,)

class S138Rows(S138Entry(list)):
    pass

class S138G:
    def __class_getitem__(cls, item):
        return (cls.__name__, item)

def s138():
    t = S138t
    check("typing1", [repr(x) for x in [t.List[int], t.Optional[int], t.Union[int, str], t.Callable[[int], str],
                                       t.Tuple[int, ...], t.Literal["a", 1], S138T, S138Box[int], t.Dict[str, None]]]
          == ["typing.List[int]", "int | None", "int | str", "typing.Callable[[int], str]", "typing.Tuple[int, ...]",
              "typing.Literal['a', 1]", "~S138T", "__main__.S138Box[int]", "typing.Dict[str, NoneType]"]
          and t.Union[int] is int and t.Optional[int] == (int | None))
    check("typing2", t.get_origin(t.List[int]) is list and t.get_args(t.Dict[str, int]) == (str, int)
          and t.get_args(t.Callable[[int], str]) == ([int], str) and t.get_origin(int) is None
          and isinstance([], t.List) and isinstance(iter([]), t.Iterator) and not isinstance(5, t.Iterable)
          and t.cast(int, "s") == "s" and not t.TYPE_CHECKING and S138Box[int](4).v == 4)
    class F:
        def close(self):
            pass
    check("typing3", isinstance(F(), S138Closer) and not isinstance(1, S138Closer)
          and S138Pt(1) == (1, 0) and S138Pt(2, 3).norm() == 5 and S138Pt._fields == ("x", "y")
          and S138Movie(name="m") == {"name": "m"} and type(S138Movie(name="m")) is dict)
    try:
        t.List()
        check("typing4", False)
    except TypeError as e:
        check("typing4", str(e) == "Type List cannot be instantiated; use list() instead")
    r = S138Rows([3, 1])
    r.sort()
    g = (x for x in [1])
    check("pep560", r == [1, 3] and S138Rows.__bases__ == (list,) and S138G[int] == ("S138G", int)
          and g.__iter__() is g and hasattr(g, "send") and hasattr(iter([]), "__next__")
          and hasattr(int | None, "__args__"))

# ===== SECTION 143: fractions.Fraction; math's real and integral arguments =====
from fractions import Fraction as S143F
import math as S143m

class S143Real:
    def __float__(self):
        return 2.25

def s143():
    F = S143F
    x = F(355, 113)
    check("frac1", F(1, 3) + F(1, 6) == F(1, 2) and F(3, -6) == F(-1, 2) and F("2_000/3") == F(2000, 3)
          and F(" -3 / 4 ") == F(-3, 4) and F("1.5e-2") == F(3, 200) and F(0.1) == F(3602879701896397, 36028797018963968)
          and repr(F(6, 4)) == "Fraction(3, 2)" and str(F(4, 2)) == "2" and F(1, 2) + 0.25 == 0.75
          and type(F(1) + 1).__name__ == "Fraction" and 2 ** F(3) == 8 and F(-2, 3) ** 3 == F(-8, 27))
    check("frac2", round(x) == 3 and round(x, 2) == F(157, 50) and S143m.floor(x) == 3 and S143m.ceil(-x) == -3
          and x.limit_denominator(10) == F(22, 7) and f"{x:.5f}" == "3.14159" and f"{-x:+.1%}" == "-314.2%"
          and hash(F(1, 2)) == hash(0.5) and sum(F(1, n) for n in range(1, 6)) == F(137, 60)
          and divmod(F(7, 2), 2) == (1, F(3, 2)) and sorted([F(1, 2), F(1, 3), 0.4]) == [F(1, 3), 0.4, F(1, 2)])
    try:
        F(1, 2) / 0
        check("frac3", False)
    except ZeroDivisionError as e:
        check("frac3", str(e) == "Fraction(1, 0)")
    try:
        S143m.sqrt("x")
        check("mathreal", False)
    except TypeError as e:
        check("mathreal", str(e) == "must be real number, not str" and S143m.sqrt(S143Real()) == 1.5
              and S143m.floor(10 ** 20 + 1) == 100000000000000000001 and S143m.trunc(-(2 ** 70) - 1) == -(2 ** 70) - 1
              and S143m.floor(S143Real()) == 2)

# ===== SECTION 144: statistics =====
import statistics as S144st

def s144():
    st = S144st
    check("stats1", st.mean([1, 2, 3, 4]) == 2.5 and type(st.mean([1, 2, 3])) is int and st.mean([0.1, 0.2, 0.3]) == 0.2
          and st.median([3, 1, 2]) == 2 and st.median([1, 2, 3, 4]) == 2.5 and st.median_low([1, 2, 3, 4]) == 2
          and st.median_high([1, 2, 3, 4]) == 3 and st.mode([1, 2, 2, 3]) == 2 and st.multimode("aabbc") == ["a", "b"]
          and st.fmean([1, 2, 3], weights=[3, 2, 1]) == 1.6666666666666667)
    check("stats2", st.variance([1, 2, 3, 4]) == 1.6666666666666667 and st.pvariance([1, 2, 3, 4]) == 1.25
          and st.stdev([1, 2, 3, 4]) == 1.2909944487358056 and st.pstdev([1, 2, 3]) == 0.816496580927726
          and st.quantiles(range(1, 11), n=4) == [2.75, 5.5, 8.25]
          and st.quantiles([1, 2, 3, 4, 5, 6, 7, 8, 9, 10], n=4, method="inclusive") == [3.25, 5.5, 7.75]
          and st.harmonic_mean([40, 60], weights=[5, 30]) == 56.0 and st.median_grouped([1, 2, 2, 3, 4, 4, 4, 4, 4, 5]) == 3.7)
    try:
        st.mean([])
        check("stats3", False)
    except st.StatisticsError as e:
        check("stats3", str(e) == "mean requires at least one data point" and isinstance(e, ValueError))

# ===== SECTION 145: random's distributions, state and float seeds =====
# One generator only: seeding MT19937 is the costliest Python this file runs.
import random as S145r

def s145():
    r = S145r.Random(1.5)
    first = r.random()
    g1 = r.gauss(0, 1)
    g2 = r.gauss(0, 1)
    st = r.getstate()
    a = r.random()
    r.setstate(st)
    check("random1", first == 0.551763726942059 and g1 == -0.050646651804881156 and g2 == 1.322273584180428
          and r.random() == a and st[0] == 3 and len(st[1]) == 625 and r.randbytes(3) == b':\xbf\xd5'
          and abs(r.expovariate(1.5) - 0.09191084084949591) < 1e-16 and 0 <= r.triangular(0, 10, 3) <= 10
          and 0 <= r.betavariate(2, 3) <= 1)

def main():
    s131() # SECTION-CALL 131
    s134() # SECTION-CALL 134
    s136() # SECTION-CALL 136
    s138() # SECTION-CALL 138
    s143() # SECTION-CALL 143
    s144() # SECTION-CALL 144
    s145() # SECTION-CALL 145
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
