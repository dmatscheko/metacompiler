# python-types.py -- the types module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_types); helpers
# carry the _mec_ty prefix. SimpleNamespace is CPython's, repr included; the
# type objects are what type() answers for a value of each kind.


class _mec_ty_SimpleNamespace:
    def __init__(self, mapping_or_iterable=(), /, **kwargs):
        for k, v in dict(mapping_or_iterable).items():
            if not isinstance(k, str):
                raise TypeError("keywords must be strings, not '" + type(k).__name__ + "'")
            setattr(self, k, v)
        for k, v in kwargs.items():
            setattr(self, k, v)

    def __repr__(self):
        name = "namespace" if type(self) is _mec_ty_SimpleNamespace else type(self).__name__
        items = [k + "=" + repr(v) for k, v in vars(self).items()]
        return name + "(" + ", ".join(items) + ")"

    def __eq__(self, other):
        if not isinstance(self, _mec_ty_SimpleNamespace) or not isinstance(other, _mec_ty_SimpleNamespace):
            return NotImplemented
        return vars(self) == vars(other)

    __hash__ = None


_mec_ty_SimpleNamespace.__name__ = "SimpleNamespace"
_mec_ty_SimpleNamespace.__qualname__ = "SimpleNamespace"


def _mec_ty_fn():
    pass


def _mec_ty_gen():
    yield


class _mec_ty_C:
    def m(self):
        pass


class _mec_ty_module:
    __all__ = ["SimpleNamespace", "FunctionType", "LambdaType", "GeneratorType",
               "BuiltinFunctionType", "BuiltinMethodType", "MethodType"]
    SimpleNamespace = _mec_ty_SimpleNamespace
    FunctionType = type(_mec_ty_fn)
    LambdaType = type(_mec_ty_fn)
    GeneratorType = type(_mec_ty_gen())
    BuiltinFunctionType = type(len)
    BuiltinMethodType = type(len)
    MethodType = type(_mec_ty_C().m)

    def __repr__(self):
        return "<module 'types'>"


_mec_mod_types = type("module", (_mec_ty_module,), {})()
_mec_mod_types.__name__ = "types"
