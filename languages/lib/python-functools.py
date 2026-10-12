# python-functools.py -- the functools module, written in Python and compiled
# with a program that imports it (core.importFile in both python grammars), so
# one implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_functools), never
# as a flat name, so `import functools` cannot replace a program's own reduce or
# wraps; helpers carry the _mec_ft prefix. The module object is an instance of a
# class named "module", as type(functools).__name__ is in CPython.
#
# The behaviour is Lib/functools.py's, error texts included. lru_cache keys a
# call by its positional arguments followed by its keyword items, in order, as
# _make_key does; `typed` is accepted and ignored, and so is update_wrapper's
# `updated`, since a function carries no __dict__ here.

_mec_ft_missing = object()

_mec_ft_assigned = ("__module__", "__name__", "__qualname__", "__doc__")


def _mec_ft_update_wrapper(wrapper, wrapped, assigned=_mec_ft_assigned, updated=("__dict__",)):
    for attr in assigned:
        try:
            value = getattr(wrapped, attr)
        except AttributeError:
            continue
        setattr(wrapper, attr, value)
    wrapper.__wrapped__ = wrapped
    return wrapper


# total_ordering's derived comparisons, by the root the class defines; each
# answers NotImplemented when the root does.
def _mec_ft_gt_from_lt(self, other):
    r = type(self).__lt__(self, other)
    return r if r is NotImplemented else not r and self != other


def _mec_ft_le_from_lt(self, other):
    r = type(self).__lt__(self, other)
    return r if r is NotImplemented else r or self == other


def _mec_ft_ge_from_lt(self, other):
    r = type(self).__lt__(self, other)
    return r if r is NotImplemented else not r


def _mec_ft_ge_from_le(self, other):
    r = type(self).__le__(self, other)
    return r if r is NotImplemented else not r or self == other


def _mec_ft_lt_from_le(self, other):
    r = type(self).__le__(self, other)
    return r if r is NotImplemented else r and self != other


def _mec_ft_gt_from_le(self, other):
    r = type(self).__le__(self, other)
    return r if r is NotImplemented else not r


def _mec_ft_lt_from_gt(self, other):
    r = type(self).__gt__(self, other)
    return r if r is NotImplemented else not r and self != other


def _mec_ft_ge_from_gt(self, other):
    r = type(self).__gt__(self, other)
    return r if r is NotImplemented else r or self == other


def _mec_ft_le_from_gt(self, other):
    r = type(self).__gt__(self, other)
    return r if r is NotImplemented else not r


def _mec_ft_le_from_ge(self, other):
    r = type(self).__ge__(self, other)
    return r if r is NotImplemented else not r or self == other


def _mec_ft_gt_from_ge(self, other):
    r = type(self).__ge__(self, other)
    return r if r is NotImplemented else r and self != other


def _mec_ft_lt_from_ge(self, other):
    r = type(self).__ge__(self, other)
    return r if r is NotImplemented else not r


_mec_ft_convert = {
    "__lt__": [("__gt__", _mec_ft_gt_from_lt), ("__le__", _mec_ft_le_from_lt), ("__ge__", _mec_ft_ge_from_lt)],
    "__le__": [("__ge__", _mec_ft_ge_from_le), ("__lt__", _mec_ft_lt_from_le), ("__gt__", _mec_ft_gt_from_le)],
    "__gt__": [("__lt__", _mec_ft_lt_from_gt), ("__ge__", _mec_ft_ge_from_gt), ("__le__", _mec_ft_le_from_gt)],
    "__ge__": [("__le__", _mec_ft_le_from_ge), ("__gt__", _mec_ft_gt_from_ge), ("__lt__", _mec_ft_lt_from_ge)],
}


class _mec_ft_module:
    __all__ = ["cache", "cmp_to_key", "lru_cache", "partial", "reduce", "total_ordering", "update_wrapper",
               "wraps", "WRAPPER_ASSIGNMENTS", "WRAPPER_UPDATES"]

    WRAPPER_ASSIGNMENTS = _mec_ft_assigned
    WRAPPER_UPDATES = ("__dict__",)

    class CacheInfo(tuple):
        def __repr__(self):
            return "CacheInfo(hits=%r, misses=%r, maxsize=%r, currsize=%r)" % tuple(self)

        @property
        def hits(self):
            return self[0]

        @property
        def misses(self):
            return self[1]

        @property
        def maxsize(self):
            return self[2]

        @property
        def currsize(self):
            return self[3]


    class _lru_cache_wrapper:
        def __init__(self, func, maxsize):
            self._mec_ft_func = func
            self._mec_ft_max = maxsize
            self._mec_ft_cache = {}
            self._mec_ft_hits = 0
            self._mec_ft_misses = 0
            _mec_ft_update_wrapper(self, func)

        def __call__(self, *args, **kwargs):
            key = args
            if kwargs:
                key = args + (_mec_ft_missing,) + tuple(kwargs.items())
            cache = self._mec_ft_cache
            if key in cache:
                self._mec_ft_hits += 1
                value = cache.pop(key)
                cache[key] = value
                return value
            self._mec_ft_misses += 1
            value = self._mec_ft_func(*args, **kwargs)
            if self._mec_ft_max is None or self._mec_ft_max > 0:
                cache[key] = value
                if self._mec_ft_max is not None and len(cache) > self._mec_ft_max:
                    del cache[next(iter(cache))]
            return value

        def __get__(self, obj, objtype=None):
            if obj is None:
                return self
            return _mec_ft_module.partial(self, obj)

        def cache_info(self):
            return _mec_ft_module.CacheInfo((self._mec_ft_hits, self._mec_ft_misses, self._mec_ft_max, len(self._mec_ft_cache)))

        def cache_clear(self):
            self._mec_ft_cache = {}
            self._mec_ft_hits = 0
            self._mec_ft_misses = 0

        def cache_parameters(self):
            return {"maxsize": self._mec_ft_max, "typed": False}

    @staticmethod
    def reduce(function, iterable, initial=_mec_ft_missing):
        it = iter(iterable)
        if initial is _mec_ft_missing:
            try:
                value = next(it)
            except StopIteration:
                raise TypeError("reduce() of empty iterable with no initial value") from None
        else:
            value = initial
        for element in it:
            value = function(value, element)
        return value

    class partial:
        def __init__(self, func, /, *args, **keywords):
            if not callable(func):
                raise TypeError("the first argument must be callable")
            if isinstance(func, _mec_ft_module.partial):
                args = func.args + args
                keywords = {**func.keywords, **keywords}
                func = func.func
            self.func = func
            self.args = args
            self.keywords = keywords

        def __call__(self, /, *args, **keywords):
            return self.func(*self.args, *args, **{**self.keywords, **keywords})

        def __repr__(self):
            parts = [repr(self.func)]
            for a in self.args:
                parts.append(repr(a))
            for k, v in self.keywords.items():
                parts.append(k + "=" + repr(v))
            return "functools.partial(" + ", ".join(parts) + ")"

    @staticmethod
    def lru_cache(maxsize=128, typed=False):
        if callable(maxsize) and not isinstance(maxsize, bool):
            return _mec_ft_module._lru_cache_wrapper(maxsize, 128)
        if maxsize is not None and not isinstance(maxsize, int):
            raise TypeError("Expected first argument to be an integer, a callable, or None")
        if maxsize is not None and maxsize < 0:
            maxsize = 0

        def decorating_function(user_function):
            return _mec_ft_module._lru_cache_wrapper(user_function, maxsize)
        return decorating_function

    @staticmethod
    def cache(user_function, /):
        return _mec_ft_module._lru_cache_wrapper(user_function, None)

    @staticmethod
    def update_wrapper(wrapper, wrapped, assigned=_mec_ft_assigned, updated=("__dict__",)):
        return _mec_ft_update_wrapper(wrapper, wrapped, assigned, updated)

    @staticmethod
    def wraps(wrapped, assigned=_mec_ft_assigned, updated=("__dict__",)):
        def decorator(wrapper):
            return _mec_ft_update_wrapper(wrapper, wrapped, assigned, updated)
        return decorator

    # Each comparison of two keys asks mycmp about the values they wrap.
    @staticmethod
    def cmp_to_key(mycmp):
        class K:
            def __init__(self, obj):
                self.obj = obj

            def __lt__(self, other):
                return mycmp(self.obj, other.obj) < 0

            def __gt__(self, other):
                return mycmp(self.obj, other.obj) > 0

            def __eq__(self, other):
                return mycmp(self.obj, other.obj) == 0

            def __le__(self, other):
                return mycmp(self.obj, other.obj) <= 0

            def __ge__(self, other):
                return mycmp(self.obj, other.obj) >= 0

            __hash__ = None

        return K

    @staticmethod
    def total_ordering(cls):
        roots = [op for op in ("__lt__", "__le__", "__gt__", "__ge__") if op in cls.__dict__]
        if not roots:
            raise ValueError("must define at least one ordering operation: < > <= >=")
        root = roots[0]
        for name, fn in _mec_ft_convert[root]:
            if name not in roots:
                setattr(cls, name, fn)
        return cls

    def __repr__(self):
        return "<module 'functools'>"


_mec_mod_functools = type("module", (_mec_ft_module,), {})()
_mec_mod_functools.__name__ = "functools"
