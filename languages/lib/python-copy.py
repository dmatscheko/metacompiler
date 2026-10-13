# python-copy.py -- the copy module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Imports here are flat (the module's top-level names bind directly), and the
# module object is _mec_mod_copy, which `import copy` binds and `from copy
# import` reads (core.importFile's callers in both grammars). Helpers carry
# the _mec_cp prefix so that a program's own names cannot replace them.
#
# copy(x) and deepcopy(x, memo) follow Lib/copy.py: a class's __copy__ /
# __deepcopy__ first, the immutable values as themselves, the builtin
# containers element by element, an instance through cls.__new__ and its
# __dict__, an exception through its args. Any other value raises copy.Error
# rather than sharing state by accident. Error lives on the module object only:
# as a flat name it would replace a program's own Error.


def _mec_cp_atomic(x):
    if x is None or isinstance(x, (int, float, complex, str, bytes, type)):
        return True
    return type(x).__name__ in ("function", "builtin_function_or_method", "method", "property")


# The new instance is recorded in memo (when there is one) before its state is
# copied, so a reference cycle through it comes back to the copy.
def _mec_cp_instance(x, fill, memo):
    cls = type(x)
    if isinstance(x, BaseException):
        y = cls(*x.args)
    else:
        y = cls.__new__(cls)
        # object.__reduce_ex__ caches the class's slot names as it copies.
        if "__slotnames__" not in cls.__dict__:
            cls.__slotnames__ = []
    if memo is not None:
        memo[id(x)] = y
    if isinstance(x, list):
        y.extend(fill(list(x)))
    elif isinstance(x, dict):
        y.update(fill(dict(x)))
    y.__dict__.update(fill(x.__dict__))
    return y


def _mec_cp_copy(x):
    cls = type(x)
    if _mec_cp_atomic(x) or cls is tuple or cls is frozenset:
        return x
    if hasattr(cls, "__copy__"):
        return x.__copy__()
    if cls is list or cls is dict or cls is set or cls is bytearray:
        return x.copy()
    if hasattr(x, "__dict__"):
        return _mec_cp_instance(x, _mec_cp_same, None)
    raise _mec_cp_module.Error("un(shallow)copyable object of type " + repr(cls))


def _mec_cp_same(v):
    return v


def _mec_cp_deepcopy(x, memo=None):
    if memo is None:
        memo = {}
    d = id(x)
    if d in memo:
        return memo[d]
    cls = type(x)
    if _mec_cp_atomic(x):
        return x
    if hasattr(cls, "__deepcopy__"):
        y = x.__deepcopy__(memo)
    elif cls is list:
        y = []
        memo[d] = y
        for a in x:
            y.append(_mec_cp_deepcopy(a, memo))
    elif cls is tuple:
        items = [_mec_cp_deepcopy(a, memo) for a in x]
        if d in memo:
            return memo[d]
        y = x
        for i in range(len(x)):
            if items[i] is not x[i]:
                y = tuple(items)
                break
    elif cls is dict:
        y = {}
        memo[d] = y
        for k, v in x.items():
            y[_mec_cp_deepcopy(k, memo)] = _mec_cp_deepcopy(v, memo)
    elif cls is set or cls is frozenset or cls is bytearray:
        y = cls([_mec_cp_deepcopy(a, memo) for a in x])
    elif hasattr(x, "__dict__"):
        y = _mec_cp_instance(x, lambda v: _mec_cp_deepcopy(v, memo), memo)
    else:
        raise _mec_cp_module.Error("un(deep)copyable object of type " + repr(cls))
    if y is not x:
        memo[d] = y
        # Lib/copy.py keeps the original alive for as long as the memo lives.
        memo.setdefault(id(memo), []).append(x)
    return y


# copy.replace(obj, **changes) is the class's __replace__. It lives on the
# module object only: a flat `replace` would collide with dataclasses.replace.
def _mec_cp_replace(obj, /, **changes):
    func = getattr(obj.__class__, "__replace__", None)
    if func is None:
        raise TypeError("replace() does not support " + obj.__class__.__name__ + " objects")
    return func(obj, **changes)


class _mec_cp_module:
    __all__ = ["Error", "copy", "deepcopy", "replace"]
    copy = staticmethod(_mec_cp_copy)
    deepcopy = staticmethod(_mec_cp_deepcopy)
    replace = staticmethod(_mec_cp_replace)

    class Error(Exception):
        pass

    def __repr__(self):
        return "<module 'copy'>"


_mec_mod_copy = type("module", (_mec_cp_module,), {})()
_mec_mod_copy.__name__ = "copy"
deepcopy = _mec_cp_deepcopy
