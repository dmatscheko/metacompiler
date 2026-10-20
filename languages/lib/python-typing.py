# python-typing.py -- the typing module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_typing); helpers
# carry the _mec_tp prefix. typing is a runtime library, and this is its
# runtime surface (3.14): every name of typing.__all__ exists, a subscription
# answers an alias that prints, compares and introspects (get_origin,
# get_args) as CPython's does, an alias of a concrete class can be called and
# subclassed (PEP 560 __mro_entries__), an unsubscripted one answers
# isinstance, and Generic, Protocol (with runtime_checkable), NamedTuple,
# TypedDict, TypeVar, ParamSpec, NewType, cast and overload work. Annotations
# themselves stay source text in this project, so nothing here evaluates one.

import abc as _mec_tp_abc
import collections as _mec_tp_collections

_mec_tp_NoneType = type(None)

# What Callable's aliases name as their origin: collections.abc.Callable has
# no class here.
_mec_tp_callable_origin = object()


# typing._type_repr: a class by its (qualified) name, ... and None by name, a
# list of argument types bracketed, anything else by repr.
def _mec_tp_repr(x):
    if x is ...:
        return "..."
    if x is None:
        return "None"
    if isinstance(x, list):
        return "[" + ", ".join([_mec_tp_repr(a) for a in x]) + "]"
    if isinstance(x, str):
        return "ForwardRef(" + repr(x) + ")"
    if isinstance(x, type):
        mod = getattr(x, "__module__", "builtins")
        if mod == "builtins" or mod is None:
            return x.__qualname__
        return mod + "." + x.__qualname__
    return repr(x)


def _mec_tp_tuple(params):
    return params if isinstance(params, tuple) else (params,)


# An alias argument as CPython stores it: None is NoneType.
def _mec_tp_arg(a):
    return _mec_tp_NoneType if a is None else a


class _mec_tp_Final:
    def __or__(self, other):
        return _mec_tp_Union[self, other]

    def __ror__(self, other):
        return _mec_tp_Union[other, self]


# A subscripted generic: List[int], Box[int], Literal['a'], Callable[[int], str].
class _mec_tp_Alias(_mec_tp_Final):
    def __init__(self, origin, args, name=None, inst=True, shown=None):
        self.__origin__ = origin
        self.__args__ = args
        self._mec_tp_name = name
        self._mec_tp_inst = inst
        # The arguments as they print, where that differs from __args__
        # (Callable's parameter list, Literal's values).
        self._mec_tp_shown = shown

    def __repr__(self):
        if self._mec_tp_name is not None:
            head = "typing." + self._mec_tp_name
        else:
            head = _mec_tp_repr(self.__origin__)
        if self._mec_tp_shown is not None:
            return head + "[" + self._mec_tp_shown + "]"
        if self.__args__ == ():
            return head + "[()]"
        return head + "[" + ", ".join([_mec_tp_repr(a) for a in self.__args__]) + "]"

    def __eq__(self, other):
        if not isinstance(other, _mec_tp_Alias):
            return NotImplemented
        return self.__origin__ == other.__origin__ and self.__args__ == other.__args__

    def __hash__(self):
        return hash((self._mec_tp_name, len(self.__args__)))

    def __call__(self, *args, **kwargs):
        if not self._mec_tp_inst:
            raise TypeError("Type " + str(self._mec_tp_name) + " cannot be instantiated; use "
                            + self.__origin__.__name__ + "() instead")
        return self.__origin__(*args, **kwargs)

    def __mro_entries__(self, bases):
        if isinstance(self.__origin__, type):
            return (self.__origin__,)
        return ()

    def __instancecheck__(self, obj):
        raise TypeError("Subscripted generics cannot be used with class and instance checks")

    def __subclasscheck__(self, cls):
        raise TypeError("Subscripted generics cannot be used with class and instance checks")

    @property
    def __parameters__(self):
        return tuple(a for a in self.__args__ if isinstance(a, (_mec_tp_TypeVar, _mec_tp_ParamSpec)))


# An unsubscripted alias of a builtin or collections class: typing.List.
class _mec_tp_SpecialAlias(_mec_tp_Final):
    def __init__(self, name, origin, nparams, inst=True):
        self._mec_tp_name = name
        self.__origin__ = origin
        self._mec_tp_nparams = nparams
        self._mec_tp_inst = inst

    def __repr__(self):
        return "typing." + self._mec_tp_name

    def __getitem__(self, params):
        params = tuple(_mec_tp_arg(p) for p in _mec_tp_tuple(params))
        return _mec_tp_Alias(self.__origin__, params, self._mec_tp_name, self._mec_tp_inst)

    def __call__(self, *args, **kwargs):
        if not self._mec_tp_inst:
            raise TypeError("Type " + self._mec_tp_name + " cannot be instantiated; use "
                            + self.__origin__.__name__ + "() instead")
        return self.__origin__(*args, **kwargs)

    def __mro_entries__(self, bases):
        return (self.__origin__,)

    def __instancecheck__(self, obj):
        return isinstance(obj, self.__origin__)

    def __subclasscheck__(self, cls):
        return issubclass(cls, self.__origin__)


# An alias of a collections.abc type, answered by what an object can do.
class _mec_tp_AbcAlias(_mec_tp_SpecialAlias):
    def __init__(self, name, check):
        _mec_tp_SpecialAlias.__init__(self, name, None, -1)
        self._mec_tp_check = check

    def __call__(self, *args, **kwargs):
        raise TypeError("Can't instantiate abstract class " + self._mec_tp_name)

    def __mro_entries__(self, bases):
        return ()

    def __instancecheck__(self, obj):
        return self._mec_tp_check(obj)

    def __subclasscheck__(self, cls):
        return self._mec_tp_check(cls, True)


def _mec_tp_has(obj, names, is_cls):
    for n in names:
        if not hasattr(obj, n):
            return False
    return True


def _mec_tp_abc_check(names, classes=()):
    def check(obj, is_cls=False):
        if is_cls:
            return issubclass(obj, classes) if classes else _mec_tp_has(obj, names, True)
        if classes and isinstance(obj, classes):
            return True
        return not classes and _mec_tp_has(obj, names, False)
    return check


def _mec_tp_hashable(obj, is_cls=False):
    if is_cls:
        return getattr(obj, "__hash__", None) is not None
    try:
        hash(obj)
        return True
    except TypeError:
        return False


# Callable[[args], result] / Callable[..., result].
class _mec_tp_CallableAlias(_mec_tp_SpecialAlias):
    def __getitem__(self, params):
        if not isinstance(params, tuple) or len(params) != 2:
            raise TypeError("Callable must be used as Callable[[arg, ...], result].")
        args, result = params
        if args is ...:
            shown = "..., " + _mec_tp_repr(_mec_tp_arg(result))
            flat = (..., _mec_tp_arg(result))
        else:
            if not isinstance(args, list):
                raise TypeError("Callable[args, result]: args must be a list. Got " + _mec_tp_repr(args))
            shown = _mec_tp_repr([_mec_tp_arg(a) for a in args]) + ", " + _mec_tp_repr(_mec_tp_arg(result))
            flat = tuple(_mec_tp_arg(a) for a in args) + (_mec_tp_arg(result),)
        return _mec_tp_Alias(self.__origin__, flat, self._mec_tp_name, True, shown)

    def __instancecheck__(self, obj):
        return callable(obj)

    def __subclasscheck__(self, cls):
        return hasattr(cls, "__call__")


class _mec_tp_TupleAlias(_mec_tp_SpecialAlias):
    def __getitem__(self, params):
        if params == ():
            return _mec_tp_Alias(tuple, (), "Tuple", False)
        return _mec_tp_Alias(tuple, tuple(_mec_tp_arg(p) for p in _mec_tp_tuple(params)), "Tuple", False)


# A special form: typing.Any, Optional, Literal, ClassVar, ...
class _mec_tp_SpecialForm(_mec_tp_Final):
    def __init__(self, name, getitem=None):
        self._mec_tp_name = name
        self._mec_tp_getitem = getitem

    def __repr__(self):
        return "typing." + self._mec_tp_name

    def __getitem__(self, params):
        if self._mec_tp_getitem is None:
            raise TypeError(repr(self) + " is not subscriptable")
        return self._mec_tp_getitem(self, params)

    def __call__(self, *args, **kwargs):
        raise TypeError("Cannot instantiate " + repr(self))

    def __instancecheck__(self, obj):
        raise TypeError(repr(self) + " cannot be used with isinstance()")

    def __subclasscheck__(self, cls):
        raise TypeError(repr(self) + " cannot be used with issubclass()")


def _mec_tp_one(form, params):
    return _mec_tp_Alias(form, (_mec_tp_arg(params),), form._mec_tp_name)


def _mec_tp_many(form, params):
    return _mec_tp_Alias(form, tuple(_mec_tp_arg(p) for p in _mec_tp_tuple(params)), form._mec_tp_name)


def _mec_tp_literal(form, params):
    vals = _mec_tp_tuple(params)
    return _mec_tp_Alias(form, vals, "Literal", True, ", ".join([repr(v) for v in vals]))


def _mec_tp_annotated(form, params):
    params = _mec_tp_tuple(params)
    if len(params) < 2:
        raise TypeError("Annotated[...] should be used with at least two arguments (a type and an annotation).")
    shown = _mec_tp_repr(params[0]) + ", " + ", ".join([repr(m) for m in params[1:]])
    a = _mec_tp_Alias(form, (params[0],), "Annotated", True, shown)
    a.__metadata__ = params[1:]
    return a


def _mec_tp_optional(form, params):
    return _mec_tp_Union[params, None]


# Union[X, Y] and X | Y over typing's objects: flattened, deduplicated, and
# collapsed to the one type when only one is left. Its class is typing.Union.
class _mec_tp_Union(_mec_tp_Final):
    def __init__(self, args):
        self.__args__ = args

    def __class_getitem__(cls, params):
        out = []
        for p in _mec_tp_tuple(params):
            p = _mec_tp_arg(p)
            subs = p.__args__ if _mec_tp_is_union(p) else (p,)
            for s in subs:
                if s not in out:
                    out.append(s)
        if len(out) == 1:
            return out[0]
        return cls(tuple(out))

    def __repr__(self):
        return " | ".join(["None" if a is _mec_tp_NoneType else _mec_tp_repr(a) for a in self.__args__])

    def __eq__(self, other):
        if not _mec_tp_is_union(other):
            return NotImplemented
        return set(self.__args__) == set(other.__args__)

    def __hash__(self):
        return hash(frozenset(self.__args__))

    def __instancecheck__(self, obj):
        return isinstance(obj, self.__args__)

    def __subclasscheck__(self, cls):
        return issubclass(cls, self.__args__)


def _mec_tp_is_union(x):
    return isinstance(x, _mec_tp_Union) or (type(x).__name__ in ("Union", "UnionType") and hasattr(x, "__args__"))


class _mec_tp_TypeVar(_mec_tp_Final):
    def __init__(self, name, *constraints, bound=None, covariant=False, contravariant=False,
                 infer_variance=False, default=None):
        self.__name__ = name
        self.__constraints__ = constraints
        self.__bound__ = bound
        self.__covariant__ = covariant
        self.__contravariant__ = contravariant
        self.__infer_variance__ = infer_variance

    def __repr__(self):
        if self.__infer_variance__:
            return self.__name__
        if self.__covariant__:
            return "+" + self.__name__
        if self.__contravariant__:
            return "-" + self.__name__
        return "~" + self.__name__


class _mec_tp_ParamSpecPart:
    def __init__(self, origin, kind):
        self.__origin__ = origin
        self._mec_tp_kind = kind

    def __repr__(self):
        return self.__origin__.__name__ + "." + self._mec_tp_kind


class _mec_tp_ParamSpec(_mec_tp_Final):
    def __init__(self, name, *, bound=None, covariant=False, contravariant=False, infer_variance=False,
                 default=None):
        self.__name__ = name
        self.__bound__ = bound
        self.args = _mec_tp_ParamSpecPart(self, "args")
        self.kwargs = _mec_tp_ParamSpecPart(self, "kwargs")

    def __repr__(self):
        return "~" + self.__name__


class _mec_tp_TypeVarTuple:
    def __init__(self, name, *, default=None):
        self.__name__ = name

    def __repr__(self):
        return self.__name__

    def __iter__(self):
        yield _mec_tp_Unpack[self]


# Generic[T] answers an alias whose base entry is Generic itself; a subclass
# subscripted (Box[int]) answers an alias of that class, which can be called.
class _mec_tp_Generic:
    def __class_getitem__(cls, params):
        params = tuple(_mec_tp_arg(p) for p in _mec_tp_tuple(params))
        if cls is _mec_tp_Generic or cls is _mec_tp_Protocol:
            return _mec_tp_Alias(cls, params, cls.__name__)
        return _mec_tp_Alias(cls, params)


def _mec_tp_proto_members(cls):
    names = []
    for k in cls.__mro__:
        if k is _mec_tp_Protocol or k is _mec_tp_Generic or k is object:
            continue
        if not getattr(k, "_is_protocol", False):
            continue
        for name in list(vars(k)) + list(getattr(k, "__annotations__", {})):
            if name.startswith("_") and (name.startswith("__") and name.endswith("__") or name.startswith("_abc_")
                                         or name.startswith("_mec_") or name == "_is_protocol"
                                         or name == "_is_runtime_protocol"):
                continue
            if name not in names:
                names.append(name)
    return names


# A protocol class is one that lists Protocol itself among its bases.
class _mec_tp_ProtocolMeta(_mec_tp_abc.ABCMeta):
    def __new__(mcls, name, bases, namespace, /, **kwargs):
        cls = super().__new__(mcls, name, bases, namespace, **kwargs)
        cls._is_protocol = any(vars(b).get("_mec_tp_root_proto", False) for b in bases)
        return cls

    def __call__(cls, *args, **kwargs):
        if getattr(cls, "_is_protocol", False):
            raise TypeError("Protocols cannot be instantiated")
        return super().__call__(*args, **kwargs)

    def __instancecheck__(cls, instance):
        if cls in type(instance).__mro__:
            return True
        if not getattr(cls, "_is_protocol", False):
            return False
        if not getattr(cls, "_is_runtime_protocol", False):
            raise TypeError("Instance and class checks can only be used with @runtime_checkable protocols")
        for name in _mec_tp_proto_members(cls):
            if not hasattr(instance, name):
                return False
        return True

    def __subclasscheck__(cls, other):
        if cls in getattr(other, "__mro__", ()):
            return True
        if not getattr(cls, "_is_protocol", False):
            return False
        if not getattr(cls, "_is_runtime_protocol", False):
            raise TypeError("Instance and class checks can only be used with @runtime_checkable protocols")
        for name in _mec_tp_proto_members(cls):
            if not hasattr(other, name):
                return False
        return True


class _mec_tp_Protocol(_mec_tp_Generic, metaclass=_mec_tp_ProtocolMeta):
    _mec_tp_root_proto = True


_mec_tp_Protocol._is_protocol = True


# A class namespace's annotations: a class statement's body hands them over
# behind __annotate_func__ (PEP 649), a type() call may pass __annotations__.
def _mec_tp_ns_annotations(ns):
    if "__annotations__" in ns:
        return ns["__annotations__"]
    annotate = ns.get("__annotate_func__")
    return annotate(1) if annotate is not None else {}


# class P(NamedTuple): fields from the annotations, defaults from the body,
# methods copied onto the collections.namedtuple class it becomes. The call
# form NamedTuple("P", [("x", int)]) is the metaclass's __call__.
class _mec_tp_NamedTupleMeta(type):
    def __new__(mcls, typename, bases, ns, **kwargs):
        if ns.get("_mec_tp_root"):
            return super().__new__(mcls, typename, bases, ns)
        anns = _mec_tp_ns_annotations(ns)
        fields = list(anns)
        defaults = []
        for f in fields:
            if f in ns:
                defaults.append(ns[f])
            elif defaults:
                raise TypeError("Non-default namedtuple field " + f + " cannot follow default field"
                                + ("s " if len(defaults) > 1 else " ")
                                + ", ".join([g for g in fields if g in ns and fields.index(g) < fields.index(f)]))
        nt = _mec_tp_collections.namedtuple(typename, fields, defaults=defaults, module=ns.get("__module__"))
        nt.__annotations__ = dict(anns)
        for key, val in ns.items():
            if key in ("__module__", "__qualname__", "__annotations__", "__annotate_func__") or key in fields:
                continue
            setattr(nt, key, val)
        return nt

    def __call__(cls, typename, fields=None, /, **kwargs):
        if fields is None:
            fields = list(kwargs.items())
        names = [f[0] for f in fields]
        nt = _mec_tp_collections.namedtuple(typename, names)
        nt.__annotations__ = dict(fields)
        return nt


class _mec_tp_NamedTuple(metaclass=_mec_tp_NamedTupleMeta):
    _mec_tp_root = True


# class Movie(TypedDict): a TypedDict class makes plain dicts.
class _mec_tp_TypedDictMeta(type):
    def __new__(mcls, name, bases, ns, total=True, **kwargs):
        cls = super().__new__(mcls, name, (dict,) if ns.get("_mec_tp_root") else bases, ns)
        ann = {}
        for b in bases:
            ann.update(getattr(b, "__annotations__", {}))
        ann.update(_mec_tp_ns_annotations(ns))
        cls.__annotations__ = ann
        cls.__total__ = total
        return cls

    def __call__(cls, *args, **kwargs):
        if cls.__dict__.get("_mec_tp_root") and args and isinstance(args[0], str):
            fields = args[1] if len(args) > 1 else kwargs
            return _mec_tp_TypedDictMeta(args[0], (cls,), {"__annotations__": dict(fields)})
        return dict(*args, **kwargs)

    def __instancecheck__(cls, obj):
        raise TypeError("TypedDict does not support instance and class checks")


class _mec_tp_TypedDict(metaclass=_mec_tp_TypedDictMeta):
    _mec_tp_root = True


class _mec_tp_NewType:
    def __init__(self, name, tp):
        self.__name__ = name
        self.__qualname__ = name
        self.__supertype__ = tp
        self.__module__ = "__main__"

    def __repr__(self):
        return self.__module__ + "." + self.__qualname__

    def __call__(self, x):
        return x

    def __or__(self, other):
        return _mec_tp_Union[self, other]

    def __ror__(self, other):
        return _mec_tp_Union[other, self]


for _mec_tp_c, _mec_tp_n in [(_mec_tp_Generic, "Generic"), (_mec_tp_Protocol, "Protocol"),
                             (_mec_tp_Union, "Union"), (_mec_tp_TypeVar, "TypeVar"),
                             (_mec_tp_ParamSpec, "ParamSpec"), (_mec_tp_TypeVarTuple, "TypeVarTuple"),
                             (_mec_tp_NamedTuple, "NamedTuple"), (_mec_tp_TypedDict, "TypedDict"),
                             (_mec_tp_NewType, "NewType")]:
    _mec_tp_c.__name__ = _mec_tp_n
    _mec_tp_c.__qualname__ = _mec_tp_n
    _mec_tp_c.__module__ = "typing"

_mec_tp_Unpack = _mec_tp_SpecialForm("Unpack", _mec_tp_one)


def _mec_tp_overload_dummy(*args, **kwds):
    raise NotImplementedError("You should not call an overloaded function. A series of @typing.overload-"
                              "decorated functions outside a stub module should always be followed by an "
                              "implementation that is not @typing.overload-decorated.")


def _mec_tp_get_origin(tp):
    if isinstance(tp, _mec_tp_Alias):
        return tp.__origin__
    if _mec_tp_is_union(tp):
        return _mec_tp_Union
    if isinstance(tp, _mec_tp_SpecialAlias):
        return tp.__origin__
    if isinstance(tp, _mec_tp_ParamSpecPart):
        return tp.__origin__
    if type(tp).__name__ == "GenericAlias" and hasattr(tp, "__origin__"):
        return tp.__origin__
    if tp is _mec_tp_Generic:
        return _mec_tp_Generic
    return None


def _mec_tp_get_args(tp):
    if isinstance(tp, _mec_tp_Alias):
        if tp._mec_tp_name == "Callable" and tp.__args__ and tp.__args__[0] is not ...:
            return (list(tp.__args__[:-1]), tp.__args__[-1])
        if tp._mec_tp_name == "Annotated":
            return (tp.__args__[0],) + tuple(tp.__metadata__)
        return tp.__args__
    if _mec_tp_is_union(tp):
        return tuple(tp.__args__)
    if type(tp).__name__ == "GenericAlias" and hasattr(tp, "__args__"):
        return tuple(tp.__args__)
    return ()



class _mec_tp_module:
    __all__ = ["Annotated", "Any", "Callable", "ClassVar", "Concatenate", "Final", "ForwardRef",
               "Generic", "Literal", "Optional", "ParamSpec", "Protocol", "Tuple", "Type", "TypeVar",
               "TypeVarTuple", "Union", "AbstractSet", "ByteString", "Container", "ContextManager",
               "Hashable", "ItemsView", "Iterable", "Iterator", "KeysView", "Mapping", "MappingView",
               "MutableMapping", "MutableSequence", "MutableSet", "Sequence", "Sized", "ValuesView",
               "Awaitable", "AsyncIterator", "AsyncIterable", "Coroutine", "Collection",
               "AsyncGenerator", "AsyncContextManager", "Reversible", "SupportsAbs", "SupportsBytes",
               "SupportsComplex", "SupportsFloat", "SupportsIndex", "SupportsInt", "SupportsRound",
               "ChainMap", "Counter", "Deque", "Dict", "DefaultDict", "List", "OrderedDict", "Set",
               "FrozenSet", "NamedTuple", "TypedDict", "Generator", "BinaryIO", "IO", "Match",
               "Pattern", "TextIO", "AnyStr", "assert_type", "assert_never", "cast", "clear_overloads",
               "dataclass_transform", "evaluate_forward_ref", "final", "get_args", "get_origin",
               "get_overloads", "get_protocol_members", "get_type_hints", "is_protocol",
               "is_typeddict", "LiteralString", "Never", "NewType", "no_type_check",
               "no_type_check_decorator", "NoDefault", "NoReturn", "NotRequired", "overload",
               "override", "ParamSpecArgs", "ParamSpecKwargs", "ReadOnly", "Required", "reveal_type",
               "runtime_checkable", "Self", "Text", "TYPE_CHECKING", "TypeAlias", "TypeGuard",
               "TypeIs", "TypeAliasType", "Unpack"]
    TYPE_CHECKING = False
    Text = str
    Generic = _mec_tp_Generic
    Protocol = _mec_tp_Protocol
    Union = _mec_tp_Union
    TypeVar = _mec_tp_TypeVar
    ParamSpec = _mec_tp_ParamSpec
    TypeVarTuple = _mec_tp_TypeVarTuple
    ParamSpecArgs = _mec_tp_ParamSpecPart
    ParamSpecKwargs = _mec_tp_ParamSpecPart
    NamedTuple = _mec_tp_NamedTuple
    TypedDict = _mec_tp_TypedDict
    NewType = _mec_tp_NewType
    Unpack = _mec_tp_Unpack
    AnyStr = _mec_tp_TypeVar("AnyStr", bytes, str)
    NoDefault = object()

    Any = _mec_tp_SpecialForm("Any")
    NoReturn = _mec_tp_SpecialForm("NoReturn")
    Never = _mec_tp_SpecialForm("Never")
    Self = _mec_tp_SpecialForm("Self")
    LiteralString = _mec_tp_SpecialForm("LiteralString")
    TypeAlias = _mec_tp_SpecialForm("TypeAlias")
    Optional = _mec_tp_SpecialForm("Optional", _mec_tp_optional)
    Literal = _mec_tp_SpecialForm("Literal", _mec_tp_literal)
    Annotated = _mec_tp_SpecialForm("Annotated", _mec_tp_annotated)
    ClassVar = _mec_tp_SpecialForm("ClassVar", _mec_tp_one)
    Final = _mec_tp_SpecialForm("Final", _mec_tp_one)
    Required = _mec_tp_SpecialForm("Required", _mec_tp_one)
    NotRequired = _mec_tp_SpecialForm("NotRequired", _mec_tp_one)
    ReadOnly = _mec_tp_SpecialForm("ReadOnly", _mec_tp_one)
    TypeGuard = _mec_tp_SpecialForm("TypeGuard", _mec_tp_one)
    TypeIs = _mec_tp_SpecialForm("TypeIs", _mec_tp_one)
    Concatenate = _mec_tp_SpecialForm("Concatenate", _mec_tp_many)
    ForwardRef = str
    TypeAliasType = type

    List = _mec_tp_SpecialAlias("List", list, 1, False)
    Dict = _mec_tp_SpecialAlias("Dict", dict, 2, False)
    Set = _mec_tp_SpecialAlias("Set", set, 1, False)
    FrozenSet = _mec_tp_SpecialAlias("FrozenSet", frozenset, 1, False)
    Type = _mec_tp_SpecialAlias("Type", type, 1, False)
    Tuple = _mec_tp_TupleAlias("Tuple", tuple, -1, False)
    Callable = _mec_tp_CallableAlias("Callable", _mec_tp_callable_origin, 2)
    Deque = _mec_tp_SpecialAlias("Deque", _mec_tp_collections.deque, 1)
    DefaultDict = _mec_tp_SpecialAlias("DefaultDict", _mec_tp_collections.defaultdict, 2)
    OrderedDict = _mec_tp_SpecialAlias("OrderedDict", _mec_tp_collections.OrderedDict, 2)
    Counter = _mec_tp_SpecialAlias("Counter", _mec_tp_collections.Counter, 1)
    ChainMap = _mec_tp_SpecialAlias("ChainMap", _mec_tp_collections.ChainMap, 2)

    Hashable = _mec_tp_AbcAlias("Hashable", _mec_tp_hashable)
    Sized = _mec_tp_AbcAlias("Sized", _mec_tp_abc_check(["__len__"]))
    Container = _mec_tp_AbcAlias("Container", _mec_tp_abc_check(["__contains__"]))
    Iterable = _mec_tp_AbcAlias("Iterable", _mec_tp_abc_check(["__iter__"]))
    Iterator = _mec_tp_AbcAlias("Iterator", _mec_tp_abc_check(["__iter__", "__next__"]))
    Reversible = _mec_tp_AbcAlias("Reversible", _mec_tp_abc_check(["__reversed__"], (list, tuple, str, range, dict)))
    Collection = _mec_tp_AbcAlias("Collection", _mec_tp_abc_check(["__len__", "__iter__", "__contains__"]))
    Generator = _mec_tp_AbcAlias("Generator", _mec_tp_abc_check(["send", "throw", "__next__"]))
    Sequence = _mec_tp_AbcAlias("Sequence", _mec_tp_abc_check([], (list, tuple, str, range, bytes, bytearray)))
    MutableSequence = _mec_tp_AbcAlias("MutableSequence", _mec_tp_abc_check([], (list, bytearray)))
    Mapping = _mec_tp_AbcAlias("Mapping", _mec_tp_abc_check([], (dict,)))
    MutableMapping = _mec_tp_AbcAlias("MutableMapping", _mec_tp_abc_check([], (dict,)))
    AbstractSet = _mec_tp_AbcAlias("AbstractSet", _mec_tp_abc_check([], (set, frozenset)))
    MutableSet = _mec_tp_AbcAlias("MutableSet", _mec_tp_abc_check([], (set,)))
    MappingView = _mec_tp_AbcAlias("MappingView", _mec_tp_abc_check(["__len__"]))
    KeysView = _mec_tp_AbcAlias("KeysView", _mec_tp_abc_check(["__len__", "__iter__"]))
    ItemsView = _mec_tp_AbcAlias("ItemsView", _mec_tp_abc_check(["__len__", "__iter__"]))
    ValuesView = _mec_tp_AbcAlias("ValuesView", _mec_tp_abc_check(["__len__", "__iter__"]))
    ContextManager = _mec_tp_AbcAlias("ContextManager", _mec_tp_abc_check(["__enter__", "__exit__"]))
    AsyncContextManager = _mec_tp_AbcAlias("AsyncContextManager", _mec_tp_abc_check(["__aenter__", "__aexit__"]))
    Awaitable = _mec_tp_AbcAlias("Awaitable", _mec_tp_abc_check(["__await__"]))
    Coroutine = _mec_tp_AbcAlias("Coroutine", _mec_tp_abc_check(["send", "throw", "__await__"]))
    AsyncIterable = _mec_tp_AbcAlias("AsyncIterable", _mec_tp_abc_check(["__aiter__"]))
    AsyncIterator = _mec_tp_AbcAlias("AsyncIterator", _mec_tp_abc_check(["__aiter__", "__anext__"]))
    AsyncGenerator = _mec_tp_AbcAlias("AsyncGenerator", _mec_tp_abc_check(["__aiter__", "asend"]))
    ByteString = _mec_tp_AbcAlias("ByteString", _mec_tp_abc_check([], (bytes, bytearray)))
    SupportsAbs = _mec_tp_AbcAlias("SupportsAbs", _mec_tp_abc_check(["__abs__"]))
    SupportsBytes = _mec_tp_AbcAlias("SupportsBytes", _mec_tp_abc_check(["__bytes__"]))
    SupportsComplex = _mec_tp_AbcAlias("SupportsComplex", _mec_tp_abc_check(["__complex__"]))
    SupportsFloat = _mec_tp_AbcAlias("SupportsFloat", _mec_tp_abc_check(["__float__"]))
    SupportsIndex = _mec_tp_AbcAlias("SupportsIndex", _mec_tp_abc_check(["__index__"]))
    SupportsInt = _mec_tp_AbcAlias("SupportsInt", _mec_tp_abc_check(["__int__"]))
    SupportsRound = _mec_tp_AbcAlias("SupportsRound", _mec_tp_abc_check(["__round__"]))
    IO = _mec_tp_AbcAlias("IO", _mec_tp_abc_check(["read", "write"]))
    TextIO = _mec_tp_AbcAlias("TextIO", _mec_tp_abc_check(["read", "write"]))
    BinaryIO = _mec_tp_AbcAlias("BinaryIO", _mec_tp_abc_check(["read", "write"]))
    Pattern = _mec_tp_AbcAlias("Pattern", _mec_tp_abc_check(["search", "pattern"]))
    Match = _mec_tp_AbcAlias("Match", _mec_tp_abc_check(["group", "span"]))

    @staticmethod
    def cast(typ, val):
        return val

    @staticmethod
    def assert_type(val, typ, /):
        return val

    @staticmethod
    def assert_never(arg, /):
        raise AssertionError("Expected code to be unreachable, but got: " + repr(arg))

    @staticmethod
    def reveal_type(obj, /):
        return obj

    @staticmethod
    def overload(func):
        return _mec_tp_overload_dummy

    @staticmethod
    def get_overloads(func):
        return []

    @staticmethod
    def clear_overloads():
        pass

    @staticmethod
    def final(f):
        try:
            f.__final__ = True
        except (AttributeError, TypeError):
            pass
        return f

    @staticmethod
    def override(method, /):
        try:
            method.__override__ = True
        except (AttributeError, TypeError):
            pass
        return method

    @staticmethod
    def no_type_check(arg):
        return arg

    @staticmethod
    def no_type_check_decorator(decorator):
        return decorator

    @staticmethod
    def runtime_checkable(cls):
        if not getattr(cls, "_is_protocol", False):
            raise TypeError("@runtime_checkable can be only applied to protocol classes, got " + repr(cls))
        cls._is_runtime_protocol = True
        return cls

    @staticmethod
    def dataclass_transform(**kwargs):
        def decorator(cls_or_fn):
            cls_or_fn.__dataclass_transform__ = kwargs
            return cls_or_fn
        return decorator

    @staticmethod
    def get_origin(tp):
        return _mec_tp_get_origin(tp)

    @staticmethod
    def get_args(tp):
        return _mec_tp_get_args(tp)

    @staticmethod
    def get_type_hints(obj, globalns=None, localns=None, include_extras=False):
        hints = {}
        if isinstance(obj, type):
            for k in reversed(obj.__mro__):
                hints.update(getattr(k, "__annotations__", {}))
            return hints
        return dict(getattr(obj, "__annotations__", {}))

    @staticmethod
    def get_protocol_members(tp):
        if not getattr(tp, "_is_protocol", False):
            raise TypeError(repr(tp) + " is not a Protocol")
        return frozenset(_mec_tp_proto_members(tp))

    @staticmethod
    def is_protocol(tp):
        return isinstance(tp, type) and getattr(tp, "_is_protocol", False) and tp is not _mec_tp_Protocol

    @staticmethod
    def is_typeddict(tp):
        return isinstance(tp, _mec_tp_TypedDictMeta) and tp is not _mec_tp_TypedDict

    @staticmethod
    def evaluate_forward_ref(forward_ref, **kwargs):
        return forward_ref

    def __repr__(self):
        return "<module 'typing'>"


_mec_mod_typing = type("module", (_mec_tp_module,), {})()
_mec_mod_typing.__name__ = "typing"
