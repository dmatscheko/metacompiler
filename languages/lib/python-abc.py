# python-abc.py -- the abc module, written in Python and compiled with a program
# that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_abc); helpers
# carry the _mec_abc prefix. The behaviour is Lib/abc.py's (with _py_abc's
# registry), error texts included: ABCMeta computes __abstractmethods__ when a
# class is made, instantiating a class that still has some is CPython's
# TypeError, and register() makes a virtual subclass.


def _mec_abc_is_abstract(value):
    if getattr(value, "__isabstractmethod__", False):
        return True
    if isinstance(value, property):
        return any(getattr(f, "__isabstractmethod__", False) for f in (value.fget, value.fset, value.fdel))
    return getattr(getattr(value, "__func__", None), "__isabstractmethod__", False)


class _mec_abc_ABCMeta(type):
    def __new__(mcls, name, bases, namespace, /, **kwargs):
        cls = super().__new__(mcls, name, bases, namespace, **kwargs)
        abstracts = set()
        for k, v in namespace.items():
            if _mec_abc_is_abstract(v):
                abstracts.add(k)
        for base in bases:
            for k in getattr(base, "__abstractmethods__", ()):
                if _mec_abc_is_abstract(getattr(cls, k, None)):
                    abstracts.add(k)
        cls.__abstractmethods__ = frozenset(abstracts)
        cls._mec_abc_registry = []
        return cls

    def __call__(cls, *args, **kwargs):
        missing = sorted(cls.__abstractmethods__)
        if missing:
            which = "method " if len(missing) == 1 else "methods "
            raise TypeError("Can't instantiate abstract class " + cls.__name__
                            + " without an implementation for abstract " + which
                            + ", ".join(["'" + m + "'" for m in missing]))
        return super().__call__(*args, **kwargs)

    def register(cls, subclass):
        if not isinstance(subclass, type):
            raise TypeError("Can only register classes")
        if issubclass(subclass, cls):
            return subclass
        cls._mec_abc_registry.append(subclass)
        return subclass

    def __instancecheck__(cls, instance):
        return cls.__subclasscheck__(type(instance))

    def __subclasscheck__(cls, subclass):
        if cls in getattr(subclass, "__mro__", ()):
            return True
        for r in cls._mec_abc_registry:
            if issubclass(subclass, r):
                return True
        return False


class _mec_abc_ABC(metaclass=_mec_abc_ABCMeta):
    __slots__ = ()


for _mec_abc_c, _mec_abc_n in [(_mec_abc_ABCMeta, "ABCMeta"), (_mec_abc_ABC, "ABC")]:
    _mec_abc_c.__name__ = _mec_abc_n
    _mec_abc_c.__qualname__ = _mec_abc_n


class _mec_abc_module:
    __all__ = ["ABC", "ABCMeta", "abstractmethod", "get_cache_token", "update_abstractmethods"]
    ABC = _mec_abc_ABC
    ABCMeta = _mec_abc_ABCMeta

    @staticmethod
    def abstractmethod(funcobj):
        funcobj.__isabstractmethod__ = True
        return funcobj

    @staticmethod
    def get_cache_token():
        return 0

    @staticmethod
    def update_abstractmethods(cls):
        if not hasattr(cls, "__abstractmethods__"):
            return cls
        abstracts = set()
        for scls in cls.__bases__:
            for name in getattr(scls, "__abstractmethods__", ()):
                if _mec_abc_is_abstract(getattr(cls, name, None)):
                    abstracts.add(name)
        for name, value in vars(cls).items():
            if _mec_abc_is_abstract(value):
                abstracts.add(name)
        cls.__abstractmethods__ = frozenset(abstracts)
        return cls

    def __repr__(self):
        return "<module 'abc' (frozen)>"


_mec_mod_abc = type("module", (_mec_abc_module,), {})()
_mec_mod_abc.__name__ = "abc"
