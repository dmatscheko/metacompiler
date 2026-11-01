# Full-syntax test: Python (3.12 core grammar) - part 8, sections 101 to 112:
# copy, dataclasses, collections, json, itertools, functools, string, os.path,
# random and the import forms.
#
# Like parts 3 and 4, these compile whole Python-source modules (lib/python-*.py),
# which a -frozen interpreter leg pays for in IR steps inside the one call that
# runs the file, so they have a file of their own to keep every file clear of
# the step brake (-max-steps, 100M per call); see tests/python-test-full.py for
# the conventions, which are the same.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

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
    # map over several sources steps them in turn and stops at the shortest, so an
    # endless one beside a finite one ends, and map(f, it, it) pairs neighbours.
    two = iter(xs)
    check("itt6", list(map(lambda a, b: a * b, it.cycle([1, 2]), [5, 6, 7])) == [5, 12, 7]
          and list(map(lambda a, b: a + b, xs[:3], it.count(10))) == [13, 12, 16]
          and list(map(lambda a, b: (a, b), two, two)) == [(3, 1), (4, 1), (5, 9), (2, 6)])

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

def main():
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
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
