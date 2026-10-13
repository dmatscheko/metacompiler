# python-operator.py -- the operator module, written in Python and compiled with
# a program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_operator), never
# as a flat name, so `import operator` cannot replace a program's own add or the
# builtin abs; helpers carry the _mec_op prefix. The behaviour is
# Lib/operator.py's, error texts and reprs included. The in-place forms (iadd,
# ...) are the operators' own augmented assignments.


class _mec_op_itemgetter:
    def __init__(self, *items):
        if not items:
            raise TypeError("itemgetter expected 1 argument, got 0")
        self._mec_op_items = items

    def __call__(self, obj, /):
        if len(self._mec_op_items) == 1:
            return obj[self._mec_op_items[0]]
        return tuple(obj[i] for i in self._mec_op_items)

    def __repr__(self):
        return "operator.itemgetter(" + ", ".join(map(repr, self._mec_op_items)) + ")"


def _mec_op_resolve(obj, attr):
    for name in attr.split("."):
        obj = getattr(obj, name)
    return obj


class _mec_op_attrgetter:
    def __init__(self, *attrs):
        if not attrs:
            raise TypeError("attrgetter expected 1 argument, got 0")
        for a in attrs:
            if not isinstance(a, str):
                raise TypeError("attribute name must be a string")
        self._mec_op_attrs = attrs

    def __call__(self, obj, /):
        if len(self._mec_op_attrs) == 1:
            return _mec_op_resolve(obj, self._mec_op_attrs[0])
        return tuple(_mec_op_resolve(obj, a) for a in self._mec_op_attrs)

    def __repr__(self):
        return "operator.attrgetter(" + ", ".join(map(repr, self._mec_op_attrs)) + ")"


class _mec_op_methodcaller:
    def __init__(self, *args, **kwargs):
        if not args:
            raise TypeError("methodcaller needs at least one argument, the method name")
        if not isinstance(args[0], str):
            raise TypeError("method name must be a string")
        self._mec_op_name = args[0]
        self._mec_op_args = args[1:]
        self._mec_op_kwargs = kwargs

    def __call__(self, obj, /):
        return getattr(obj, self._mec_op_name)(*self._mec_op_args, **self._mec_op_kwargs)

    def __repr__(self):
        args = [repr(self._mec_op_name)]
        args.extend(map(repr, self._mec_op_args))
        args.extend(k + "=" + repr(v) for k, v in self._mec_op_kwargs.items())
        return "operator.methodcaller(" + ", ".join(args) + ")"


def _mec_op_iadd(a, b):
    a += b
    return a


def _mec_op_isub(a, b):
    a -= b
    return a


def _mec_op_imul(a, b):
    a *= b
    return a


def _mec_op_itruediv(a, b):
    a /= b
    return a


def _mec_op_ifloordiv(a, b):
    a //= b
    return a


def _mec_op_imod(a, b):
    a %= b
    return a


def _mec_op_iconcat(a, b):
    if not hasattr(a, "__getitem__"):
        raise TypeError("'" + type(a).__name__ + "' object can't be concatenated")
    a += b
    return a


def _mec_op_concat(a, b):
    if not hasattr(a, "__getitem__"):
        raise TypeError("'" + type(a).__name__ + "' object can't be concatenated")
    return a + b


def _mec_op_countOf(a, b):
    return sum(1 for x in a if x is b or x == b)


def _mec_op_indexOf(a, b):
    for i, x in enumerate(a):
        if x is b or x == b:
            return i
    raise ValueError("sequence.index(x): x not in sequence")


def _mec_op_abs(a):
    return abs(a)


def _mec_op_setitem(a, b, c):
    a[b] = c


def _mec_op_delitem(a, b):
    del a[b]


class _mec_op_module:
    __all__ = ["abs", "add", "and_", "attrgetter", "concat", "contains", "countOf", "delitem",
               "eq", "floordiv", "ge", "getitem", "gt", "iadd", "iconcat", "ifloordiv", "imod",
               "imul", "index", "indexOf", "inv", "invert", "is_", "is_none", "is_not",
               "is_not_none", "isub", "itemgetter", "itruediv", "le", "lshift", "lt",
               "methodcaller", "mod", "mul", "ne", "neg", "not_", "or_", "pos", "pow",
               "rshift", "setitem", "sub", "truediv", "truth", "xor"]
    itemgetter = _mec_op_itemgetter
    attrgetter = _mec_op_attrgetter
    methodcaller = _mec_op_methodcaller

    @staticmethod
    def add(a, b, /):
        return a + b

    @staticmethod
    def sub(a, b, /):
        return a - b

    @staticmethod
    def mul(a, b, /):
        return a * b

    @staticmethod
    def truediv(a, b, /):
        return a / b

    @staticmethod
    def floordiv(a, b, /):
        return a // b

    @staticmethod
    def mod(a, b, /):
        return a % b

    @staticmethod
    def pow(a, b, /):
        return a ** b

    @staticmethod
    def lshift(a, b, /):
        return a << b

    @staticmethod
    def rshift(a, b, /):
        return a >> b

    @staticmethod
    def and_(a, b, /):
        return a & b

    @staticmethod
    def or_(a, b, /):
        return a | b

    @staticmethod
    def xor(a, b, /):
        return a ^ b

    @staticmethod
    def neg(a, /):
        return -a

    @staticmethod
    def pos(a, /):
        return +a

    @staticmethod
    def abs(a, /):
        return _mec_op_abs(a)

    @staticmethod
    def invert(a, /):
        return ~a

    inv = invert
    @staticmethod
    def not_(a, /):
        return not a

    @staticmethod
    def truth(a, /):
        return bool(a)

    @staticmethod
    def eq(a, b, /):
        return a == b

    @staticmethod
    def ne(a, b, /):
        return a != b

    @staticmethod
    def lt(a, b, /):
        return a < b

    @staticmethod
    def le(a, b, /):
        return a <= b

    @staticmethod
    def gt(a, b, /):
        return a > b

    @staticmethod
    def ge(a, b, /):
        return a >= b

    @staticmethod
    def is_(a, b, /):
        return a is b

    @staticmethod
    def is_not(a, b, /):
        return a is not b

    @staticmethod
    def is_none(a, /):
        return a is None

    @staticmethod
    def is_not_none(a, /):
        return a is not None

    @staticmethod
    def contains(a, b, /):
        return b in a

    @staticmethod
    def getitem(a, b, /):
        return a[b]

    @staticmethod
    def index(a, /):
        if isinstance(a, int):
            return int(a)
        return a.__index__()

    setitem = staticmethod(_mec_op_setitem)
    delitem = staticmethod(_mec_op_delitem)
    concat = staticmethod(_mec_op_concat)
    countOf = staticmethod(_mec_op_countOf)
    indexOf = staticmethod(_mec_op_indexOf)
    iadd = staticmethod(_mec_op_iadd)
    isub = staticmethod(_mec_op_isub)
    imul = staticmethod(_mec_op_imul)
    itruediv = staticmethod(_mec_op_itruediv)
    ifloordiv = staticmethod(_mec_op_ifloordiv)
    imod = staticmethod(_mec_op_imod)
    iconcat = staticmethod(_mec_op_iconcat)

    def __repr__(self):
        return "<module 'operator'>"


_mec_mod_operator = type("module", (_mec_op_module,), {})()
_mec_mod_operator.__name__ = "operator"
