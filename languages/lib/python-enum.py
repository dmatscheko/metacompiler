# python-enum.py -- the enum module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# The classes are Lib/enum.py's in behaviour, built on the metaclass protocol:
# EnumType makes a class's members as the class is created and answers iter(),
# len(), [], `in` and the call by value on it. Enum, IntEnum, StrEnum, auto and
# unique are offered; Flag, IntFlag, the functional API (Enum("C", "A B")),
# _missing_, a member-defining __new__ and member reassignment checks are not.
# Every public name lives on the module object only (_mec_mod_enum), never as a
# flat name; helpers carry the _mec_en prefix.


class _mec_en_auto:
    def __init__(self):
        self.value = _mec_en_auto_null

    def __repr__(self):
        return "auto(" + ("_auto_null" if self.value is _mec_en_auto_null else repr(self.value)) + ")"


_mec_en_auto_null = object()


def _mec_en_is_dunder(n):
    return len(n) > 4 and n[:2] == "__" and n[-2:] == "__"


def _mec_en_is_sunder(n):
    return len(n) > 2 and n[0] == "_" and n[-1] == "_" and n[1] != "_" and n[-2] != "_"


# A class body name that is no member: a dunder or sunder, a descriptor (a
# function, a property, a static or class method) or a nested class.
def _mec_en_not_member(n, v):
    if _mec_en_is_dunder(n) or _mec_en_is_sunder(n) or isinstance(v, type):
        return True
    return type(v).__name__ in ("function", "property", "staticmethod", "classmethod", "builtin_function_or_method")


# The builtin data type an enum mixes in (IntEnum's int), or None.
def _mec_en_mixin(cls):
    for b in cls.__mro__:
        if b is int or b is str or b is float:
            return b
    return None


class _mec_en_EnumType(type):
    def __new__(metacls, cls, bases, classdict, **kwds):
        names = []
        values = {}
        body = {}
        for k in classdict:
            v = classdict[k]
            if _mec_en_not_member(k, v):
                body[k] = v
            else:
                names.append(k)
                values[k] = v
        enum_class = super().__new__(metacls, cls, bases, body)
        enum_class._member_names_ = []
        enum_class._member_map_ = {}
        enum_class._value2member_map_ = {}
        mixin = _mec_en_mixin(enum_class)
        # A data type's own __repr__, __str__ and __format__ do not win over the
        # enum's: each one the body does not define comes from the nearest enum
        # class in the MRO that does.
        if mixin is not None:
            for name in ("__repr__", "__str__", "__format__"):
                if name in classdict:
                    continue
                for c in enum_class.__mro__[1:]:
                    if issubclass(c, _mec_en_Enum) and name in c.__dict__:
                        setattr(enum_class, name, c.__dict__[name])
                        break
        last = []
        for n in names:
            v = values[n]
            if isinstance(v, _mec_en_auto):
                if v.value is _mec_en_auto_null:
                    v.value = enum_class._generate_next_value_(n, 1, len(last), last[:])
                v = v.value
            alias = None
            for m in enum_class._member_names_:
                if enum_class._member_map_[m]._value_ == v:
                    alias = enum_class._member_map_[m]
                    break
            if alias is not None:
                member = alias
            else:
                if mixin is None:
                    member = object.__new__(enum_class)
                else:
                    member = mixin.__new__(enum_class, v)
                member._name_ = n
                member._value_ = v
                enum_class._member_names_.append(n)
                enum_class._value2member_map_[v] = member
            enum_class._member_map_[n] = member
            setattr(enum_class, n, member)
            last.append(v)
        return enum_class

    def __call__(cls, value):
        if isinstance(value, cls):
            return value
        if value in cls._value2member_map_:
            return cls._value2member_map_[value]
        raise ValueError(repr(value) + " is not a valid " + cls.__qualname__)

    def __getitem__(cls, name):
        return cls._member_map_[name]

    def __iter__(cls):
        return iter([cls._member_map_[n] for n in cls._member_names_])

    def __reversed__(cls):
        return iter([cls._member_map_[n] for n in reversed(cls._member_names_)])

    def __len__(cls):
        return len(cls._member_names_)

    def __contains__(cls, value):
        if isinstance(value, cls):
            return True
        return value in cls._value2member_map_

    def __bool__(cls):
        return True

    @property
    def __members__(cls):
        return dict(cls._member_map_)

    def __repr__(cls):
        return "<enum " + repr(cls.__name__) + ">"


class _mec_en_Enum(metaclass=_mec_en_EnumType):
    def _generate_next_value_(name, start, count, last_values):
        for last in reversed(last_values):
            try:
                return last + 1
            except TypeError:
                pass
        return start

    @property
    def name(self):
        return self._name_

    @property
    def value(self):
        return self._value_

    def __repr__(self):
        return "<" + type(self).__name__ + "." + self._name_ + ": " + repr(self._value_) + ">"

    def __str__(self):
        return type(self).__name__ + "." + self._name_

    def __format__(self, spec):
        return format(str(self), spec)

    def __hash__(self):
        return hash(self._name_)


class _mec_en_IntEnum(int, _mec_en_Enum):
    def __str__(self):
        return str(self._value_)

    def __format__(self, spec):
        return format(self._value_, spec)


class _mec_en_StrEnum(str, _mec_en_Enum):
    def _generate_next_value_(name, start, count, last_values):
        return name.lower()

    def __str__(self):
        return self._value_

    def __format__(self, spec):
        return format(self._value_, spec)


def _mec_en_unique(enumeration):
    dups = []
    for n in enumeration._member_map_:
        m = enumeration._member_map_[n]
        if n != m._name_:
            dups.append(n + " -> " + m._name_)
    if dups:
        raise ValueError("duplicate values found in " + repr(enumeration) + ": " + ", ".join(dups))
    return enumeration


for _mec_en_c, _mec_en_n in [(_mec_en_EnumType, "EnumType"), (_mec_en_Enum, "Enum"),
                            (_mec_en_IntEnum, "IntEnum"), (_mec_en_StrEnum, "StrEnum"),
                            (_mec_en_auto, "auto")]:
    _mec_en_c.__name__ = _mec_en_n
    _mec_en_c.__qualname__ = _mec_en_n
    _mec_en_c.__module__ = "enum"


class _mec_en_module:
    __all__ = ["EnumType", "EnumMeta", "Enum", "IntEnum", "StrEnum", "auto", "unique"]
    EnumType = _mec_en_EnumType
    EnumMeta = _mec_en_EnumType
    Enum = _mec_en_Enum
    IntEnum = _mec_en_IntEnum
    StrEnum = _mec_en_StrEnum
    auto = _mec_en_auto
    unique = staticmethod(_mec_en_unique)

    def __repr__(self):
        return "<module 'enum'>"


_mec_mod_enum = type("module", (_mec_en_module,), {})()
_mec_mod_enum.__name__ = "enum"
