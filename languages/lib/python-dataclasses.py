# python-dataclasses.py -- the dataclasses module, written in Python and
# compiled with a program that imports it (core.importFile in both python
# grammars), so one implementation serves all three engines.
#
# Imports here are flat (the module's top-level names bind directly), so the
# public names are bound as themselves and the module object carrying them is
# _mec_mod_dataclasses; every helper carries the _mec_dc prefix so that a
# program's own names cannot replace it.
#
# The surface is Lib/dataclasses.py's common one: @dataclass with and without
# arguments (init, repr, eq, order, unsafe_hash, frozen, match_args), field()
# with default / default_factory / init / repr / hash / compare, inherited
# fields, __post_init__, the hash rules, FrozenInstanceError, fields, asdict,
# astuple, replace and is_dataclass. CPython generates the methods with exec;
# here they are closures, so __init__ checks its own arguments and words the
# TypeErrors as CPython does (without the "Did you mean" suffix).
#
# Not supported, and refused with NotImplementedError rather than run wrong:
# slots, kw_only, weakref_slot, KW_ONLY, InitVar and make_dataclass. A
# ClassVar annotation is not a field, as in CPython. An annotation is kept as
# its source text in these engines, so Field.type is that text, not the type.

# asdict and astuple deep-copy a leaf value as CPython does, with
# lib/python-copy.py's deepcopy; in this flat import model that also binds
# copy and deepcopy.
import copy


class FrozenInstanceError(AttributeError):
    pass


class _MISSING_TYPE:
    pass


MISSING = _MISSING_TYPE()


class Field:
    def __init__(self, default, default_factory, init, repr, hash, compare, metadata):
        self.name = None
        self.type = None
        self.default = default
        self.default_factory = default_factory
        self.init = init
        self.repr = repr
        self.hash = hash
        self.compare = compare
        self.metadata = {} if metadata is None else metadata

    def __repr__(self):
        return ("Field(name=" + repr(self.name) + ",type=" + repr(self.type) +
                ",default=" + repr(self.default) + ",default_factory=" + repr(self.default_factory) +
                ",init=" + repr(self.init) + ",repr=" + repr(self.repr) + ",hash=" + repr(self.hash) +
                ",compare=" + repr(self.compare) + ",metadata=" + repr(self.metadata) + ")")


def field(*, default=MISSING, default_factory=MISSING, init=True, repr=True, hash=None, compare=True,
          metadata=None, kw_only=MISSING, doc=None):
    if default is not MISSING and default_factory is not MISSING:
        raise ValueError("cannot specify both default and default_factory")
    if kw_only is not MISSING:
        raise NotImplementedError("dataclasses.field(kw_only=...) is not supported")
    return Field(default, default_factory, init, repr, hash, compare, metadata)


def _mec_dc_unsupported(*args, **kwargs):
    raise NotImplementedError("dataclasses.make_dataclass is not supported")


make_dataclass = _mec_dc_unsupported


# The class's own value for name, or MISSING: what cls.__dict__ holds.
def _mec_dc_own(cls, name):
    return cls.__dict__.get(name, MISSING)


def _mec_dc_is_classvar(ann):
    return isinstance(ann, str) and (ann == "ClassVar" or ann.startswith("ClassVar[") or
                                     ann == "typing.ClassVar" or ann.startswith("typing.ClassVar["))


def _mec_dc_is_special(ann):
    if not isinstance(ann, str):
        return False
    for p in ("InitVar", "dataclasses.InitVar", "KW_ONLY", "dataclasses.KW_ONLY"):
        if ann == p or ann.startswith(p + "["):
            return True
    return False


def _mec_dc_own_fields(cls):
    anns = cls.__annotations__
    out = []
    for name, ann in anns.items():
        if _mec_dc_is_classvar(ann):
            continue
        if _mec_dc_is_special(ann):
            raise NotImplementedError("dataclasses " + ann + " fields are not supported")
        default = _mec_dc_own(cls, name)
        if isinstance(default, Field):
            f = default
        else:
            f = Field(default, MISSING, True, True, None, True, None)
        f.name = name
        f.type = ann
        if f.default is not MISSING and type(f.default) in (list, dict, set):
            raise ValueError("mutable default " + repr(type(f.default)) + " for field " + name +
                             " is not allowed: use default_factory")
        if isinstance(default, Field):
            if f.default is MISSING:
                delattr(cls, name)
            else:
                setattr(cls, name, f.default)
        out.append(f)
    return out


def _mec_dc_quote_list(names):
    q = ["'" + n + "'" for n in names]
    if len(q) == 1:
        return q[0]
    if len(q) == 2:
        return q[0] + " and " + q[1]
    return ", ".join(q[:-1]) + ", and " + q[-1]


# CPython's argument errors, in its order: each keyword, then the positional
# count, then what is missing.
def _mec_dc_check(qual, names, nreq, args, kwargs):
    for k in kwargs:
        if k not in names:
            raise TypeError(qual + "() got an unexpected keyword argument '" + k + "'")
        if names.index(k) < len(args):
            raise TypeError(qual + "() got multiple values for argument '" + k + "'")
    if len(args) > len(names):
        if nreq == len(names):
            n = len(names) + 1
            raise TypeError(qual + "() takes " + str(n) + " positional argument" + ("" if n == 1 else "s") +
                            " but " + str(len(args) + 1) + " were given")
        raise TypeError(qual + "() takes from " + str(nreq + 1) + " to " + str(len(names) + 1) +
                        " positional arguments but " + str(len(args) + 1) + " were given")
    missing = [names[i] for i in range(len(args), nreq) if names[i] not in kwargs]
    if missing:
        raise TypeError(qual + "() missing " + str(len(missing)) + " required positional argument" +
                        ("" if len(missing) == 1 else "s") + ": " + _mec_dc_quote_list(missing))


def _mec_dc_make_init(cls, flds, frozen):
    params = [f for f in flds if f.init]
    names = [f.name for f in params]
    nreq = 0
    seen_default = None
    for f in params:
        if f.default is not MISSING or f.default_factory is not MISSING:
            seen_default = f
        elif seen_default is not None:
            raise TypeError("non-default argument '" + f.name + "' follows default argument '" +
                            seen_default.name + "'")
        else:
            nreq = nreq + 1
    qual = cls.__qualname__ + ".__init__"

    def __init__(self, *args, **kwargs):
        _mec_dc_check(qual, names, nreq, args, kwargs)
        for f in flds:
            if not f.init:
                if f.default_factory is not MISSING:
                    v = f.default_factory()
                elif f.default is not MISSING:
                    v = f.default
                else:
                    continue
            else:
                i = names.index(f.name)
                if i < len(args):
                    v = args[i]
                elif f.name in kwargs:
                    v = kwargs[f.name]
                elif f.default_factory is not MISSING:
                    v = f.default_factory()
                else:
                    v = f.default
            if frozen:
                object.__setattr__(self, f.name, v)
            else:
                setattr(self, f.name, v)
        if hasattr(self, "__post_init__"):
            self.__post_init__()
    return __init__


_mec_dc_repr_running = []


def _mec_dc_make_repr(flds):
    def __repr__(self):
        key = id(self)
        if key in _mec_dc_repr_running:
            return "..."
        _mec_dc_repr_running.append(key)
        try:
            parts = [f.name + "=" + repr(getattr(self, f.name)) for f in flds if f.repr]
        finally:
            _mec_dc_repr_running.remove(key)
        return self.__class__.__qualname__ + "(" + ", ".join(parts) + ")"
    return __repr__


def _mec_dc_values(self, flds):
    return tuple([getattr(self, f.name) for f in flds])


def _mec_dc_make_eq(flds):
    def __eq__(self, other):
        if self is other:
            return True
        if other.__class__ is self.__class__:
            return _mec_dc_values(self, flds) == _mec_dc_values(other, flds)
        return NotImplemented
    return __eq__


def _mec_dc_make_order(op, flds):
    def lt(self, other):
        if other.__class__ is self.__class__:
            return _mec_dc_values(self, flds) < _mec_dc_values(other, flds)
        return NotImplemented

    def le(self, other):
        if other.__class__ is self.__class__:
            return _mec_dc_values(self, flds) <= _mec_dc_values(other, flds)
        return NotImplemented

    def gt(self, other):
        if other.__class__ is self.__class__:
            return _mec_dc_values(self, flds) > _mec_dc_values(other, flds)
        return NotImplemented

    def ge(self, other):
        if other.__class__ is self.__class__:
            return _mec_dc_values(self, flds) >= _mec_dc_values(other, flds)
        return NotImplemented
    return {"__lt__": lt, "__le__": le, "__gt__": gt, "__ge__": ge}[op]


def _mec_dc_make_hash(flds):
    def __hash__(self):
        return hash(_mec_dc_values(self, flds))
    return __hash__


def _mec_dc_make_frozen(cls, flds):
    names = [f.name for f in flds]

    def __setattr__(self, name, value):
        if type(self) is cls or name in names:
            raise FrozenInstanceError("cannot assign to field " + repr(name))
        object.__setattr__(self, name, value)

    def __delattr__(self, name):
        if type(self) is cls or name in names:
            raise FrozenInstanceError("cannot delete field " + repr(name))
        object.__delattr__(self, name)
    return __setattr__, __delattr__


# Sets a generated method unless the class defines that name itself; True
# when it did.
def _mec_dc_set(cls, name, fn):
    if _mec_dc_own(cls, name) is not MISSING:
        return True
    fn.__qualname__ = cls.__qualname__ + "." + name
    setattr(cls, name, fn)
    return False


def _mec_dc_process(cls, init, repr, eq, order, unsafe_hash, frozen, match_args):
    fmap = {}
    any_frozen_base = False
    has_dc_bases = False
    for b in cls.__mro__[-1:0:-1]:
        bf = getattr(b, "__dataclass_fields__", None)
        if bf is not None:
            has_dc_bases = True
            for f in bf.values():
                fmap[f.name] = f
            if b.__dataclass_params__.frozen:
                any_frozen_base = True
    for f in _mec_dc_own_fields(cls):
        fmap[f.name] = f
    if has_dc_bases:
        if any_frozen_base and not frozen:
            raise TypeError("cannot inherit non-frozen dataclass from a frozen one")
        if not any_frozen_base and frozen:
            raise TypeError("cannot inherit frozen dataclass from a non-frozen one")
    flds = list(fmap.values())
    setattr(cls, "__dataclass_fields__", fmap)
    setattr(cls, "__dataclass_params__", _mec_dc_params(init, repr, eq, order, unsafe_hash, frozen))
    class_hash = _mec_dc_own(cls, "__hash__")
    has_explicit_hash = not (class_hash is MISSING or (class_hash is None and _mec_dc_own(cls, "__eq__") is not MISSING))
    if init:
        _mec_dc_set(cls, "__init__", _mec_dc_make_init(cls, flds, frozen))
    if repr:
        _mec_dc_set(cls, "__repr__", _mec_dc_make_repr(flds))
    cmp = [f for f in flds if f.compare]
    if eq:
        _mec_dc_set(cls, "__eq__", _mec_dc_make_eq(cmp))
    if order:
        if not eq:
            raise ValueError("eq must be true if order is true")
        for op in ("__lt__", "__le__", "__gt__", "__ge__"):
            if _mec_dc_set(cls, op, _mec_dc_make_order(op, cmp)):
                raise TypeError("Cannot overwrite attribute " + op + " in class " + cls.__name__ +
                                " Consider using functools.total_ordering")
    if frozen:
        sa, da = _mec_dc_make_frozen(cls, flds)
        for name, fn in (("__setattr__", sa), ("__delattr__", da)):
            if _mec_dc_set(cls, name, fn):
                raise TypeError("Cannot overwrite attribute " + name + " in class " + cls.__name__)
    hflds = [f for f in flds if (f.compare if f.hash is None else f.hash)]
    if unsafe_hash:
        if has_explicit_hash:
            raise TypeError("Cannot overwrite attribute __hash__ in class " + cls.__name__)
        cls.__hash__ = _mec_dc_make_hash(hflds)
        cls.__hash__.__qualname__ = cls.__qualname__ + ".__hash__"
    elif eq and not has_explicit_hash:
        if frozen:
            cls.__hash__ = _mec_dc_make_hash(hflds)
            cls.__hash__.__qualname__ = cls.__qualname__ + ".__hash__"
        else:
            cls.__hash__ = None
    if match_args and _mec_dc_own(cls, "__match_args__") is MISSING:
        cls.__match_args__ = tuple([f.name for f in flds if f.init])
    if _mec_dc_own(cls, "__replace__") is MISSING:
        cls.__replace__ = _mec_dc_replace
    return cls


class _mec_dc_params:
    def __init__(self, init, repr, eq, order, unsafe_hash, frozen):
        self.init = init
        self.repr = repr
        self.eq = eq
        self.order = order
        self.unsafe_hash = unsafe_hash
        self.frozen = frozen


def dataclass(cls=None, /, *, init=True, repr=True, eq=True, order=False, unsafe_hash=False, frozen=False,
              match_args=True, kw_only=False, slots=False, weakref_slot=False):
    for flag, name in ((kw_only, "kw_only"), (slots, "slots"), (weakref_slot, "weakref_slot")):
        if flag:
            raise NotImplementedError("dataclass(" + name + "=True) is not supported")

    def wrap(c):
        return _mec_dc_process(c, init, repr, eq, order, unsafe_hash, frozen, match_args)
    if cls is None:
        return wrap
    return wrap(cls)


def _mec_dc_is_instance(obj):
    return not isinstance(obj, type) and hasattr(type(obj), "__dataclass_fields__")


def is_dataclass(obj):
    cls = obj if isinstance(obj, type) else type(obj)
    return hasattr(cls, "__dataclass_fields__")


def _mec_dc_fields(class_or_instance):
    fs = getattr(class_or_instance, "__dataclass_fields__", None)
    if fs is None:
        raise TypeError("must be called with a dataclass type or instance")
    return tuple(fs.values())


fields = _mec_dc_fields


def _mec_dc_asdict(obj, mk):
    if _mec_dc_is_instance(obj):
        return mk([(f.name, _mec_dc_asdict(getattr(obj, f.name), mk)) for f in _mec_dc_fields(obj)])
    if type(obj) is list:
        return [_mec_dc_asdict(v, mk) for v in obj]
    if type(obj) is tuple:
        return tuple([_mec_dc_asdict(v, mk) for v in obj])
    if type(obj) is dict:
        return dict([(_mec_dc_asdict(k, mk), _mec_dc_asdict(v, mk)) for k, v in obj.items()])
    return _mec_cp_deepcopy(obj)


def asdict(obj, *, dict_factory=dict):
    if not _mec_dc_is_instance(obj):
        raise TypeError("asdict() should be called on dataclass instances")
    return _mec_dc_asdict(obj, dict_factory)


def _mec_dc_pairs_to_tuple(pairs):
    return tuple([v for k, v in pairs])


def astuple(obj, *, tuple_factory=tuple):
    if not _mec_dc_is_instance(obj):
        raise TypeError("astuple() should be called on dataclass instances")
    return tuple_factory([_mec_dc_asdict(getattr(obj, f.name), _mec_dc_pairs_to_tuple) for f in _mec_dc_fields(obj)])


# Every dataclass's __replace__ (what copy.replace calls).
def _mec_dc_replace(self, /, **changes):
    for f in _mec_dc_fields(self):
        if not f.init and f.name in changes:
            raise TypeError("field " + f.name + " is declared with init=False, it cannot be specified with replace()")
        if f.init and f.name not in changes:
            changes[f.name] = getattr(self, f.name)
    return self.__class__(**changes)


_mec_dc_replace.__name__ = "_replace"
_mec_dc_replace.__qualname__ = "_replace"


def replace(obj, /, **changes):
    if not _mec_dc_is_instance(obj):
        raise TypeError("replace() should be called on dataclass instances")
    return _mec_dc_replace(obj, **changes)


class _mec_dc_module:
    dataclass = staticmethod(dataclass)
    field = staticmethod(field)
    fields = staticmethod(_mec_dc_fields)
    asdict = staticmethod(asdict)
    astuple = staticmethod(astuple)
    replace = staticmethod(replace)
    is_dataclass = staticmethod(is_dataclass)
    make_dataclass = staticmethod(_mec_dc_unsupported)
    Field = Field
    FrozenInstanceError = FrozenInstanceError
    MISSING = MISSING

    def __repr__(self):
        return "<module 'dataclasses'>"


_mec_mod_dataclasses = _mec_dc_module()
