# python-collections.py -- the collections module's containers, written in
# Python and compiled with a program that imports them (core.importFile in both
# python grammars), so one implementation serves all three engines.
#
# Imports here are flat (the module's top-level names bind directly), so
# deque, defaultdict, OrderedDict, Counter and namedtuple are bound as themselves and the
# module object carrying them is _mec_mod_collections; helpers carry the
# _mec_co prefix so that a program's own names cannot replace them.
#
# The behaviour is Lib/collections/__init__.py's, error texts included.
# defaultdict, OrderedDict and Counter are dict subclasses, as there; deque
# keeps its items in a list, so popleft and appendleft are linear rather than
# constant time. deque and ChainMap take type parameters (deque[int]) as
# CPython's do through __class_getitem__.


class deque[T]:
    def __init__(self, iterable=(), maxlen=None):
        if maxlen is not None and maxlen < 0:
            raise ValueError("maxlen must be non-negative")
        self._mec_co_items = []
        self._mec_co_max = maxlen
        self.extend(iterable)

    @property
    def maxlen(self):
        return self._mec_co_max

    def append(self, x):
        self._mec_co_items.append(x)
        if self._mec_co_max is not None and len(self._mec_co_items) > self._mec_co_max:
            self._mec_co_items.pop(0)

    def appendleft(self, x):
        self._mec_co_items.insert(0, x)
        if self._mec_co_max is not None and len(self._mec_co_items) > self._mec_co_max:
            self._mec_co_items.pop()

    def pop(self):
        if not self._mec_co_items:
            raise IndexError("pop from an empty deque")
        return self._mec_co_items.pop()

    def popleft(self):
        if not self._mec_co_items:
            raise IndexError("pop from an empty deque")
        return self._mec_co_items.pop(0)

    def extend(self, iterable):
        for x in list(iterable):
            self.append(x)

    def extendleft(self, iterable):
        for x in list(iterable):
            self.appendleft(x)

    def rotate(self, n=1):
        k = len(self._mec_co_items)
        if k:
            n = n % k
            self._mec_co_items[:] = self._mec_co_items[k - n:] + self._mec_co_items[:k - n]

    def clear(self):
        self._mec_co_items.clear()

    def copy(self):
        return deque(self._mec_co_items, self._mec_co_max)

    def __copy__(self):
        return self.copy()

    def count(self, x):
        return self._mec_co_items.count(x)

    def index(self, x, start=0, stop=None):
        k = len(self._mec_co_items)
        if stop is None:
            stop = k
        for i in range(k)[start:stop]:
            if self._mec_co_items[i] == x:
                return i
        raise ValueError("deque.index(x): x not in deque")

    def remove(self, x):
        for i in range(len(self._mec_co_items)):
            if self._mec_co_items[i] == x:
                del self._mec_co_items[i]
                return
        raise ValueError("deque.remove(x): x not in deque")

    def insert(self, i, x):
        if self._mec_co_max is not None and len(self._mec_co_items) >= self._mec_co_max:
            raise IndexError("deque already at its maximum size")
        self._mec_co_items.insert(i, x)

    def reverse(self):
        self._mec_co_items.reverse()

    def __len__(self):
        return len(self._mec_co_items)

    def __iter__(self):
        return iter(self._mec_co_items)

    def __reversed__(self):
        return reversed(self._mec_co_items)

    def __contains__(self, x):
        return x in self._mec_co_items

    def _mec_co_at(self, i):
        k = len(self._mec_co_items)
        if i < 0:
            i += k
        if i < 0 or i >= k:
            raise IndexError("deque index out of range")
        return i

    def __getitem__(self, i):
        return self._mec_co_items[self._mec_co_at(i)]

    def __setitem__(self, i, x):
        self._mec_co_items[self._mec_co_at(i)] = x

    def __delitem__(self, i):
        del self._mec_co_items[self._mec_co_at(i)]

    def __eq__(self, other):
        if not isinstance(other, deque):
            return NotImplemented
        return self._mec_co_items == other._mec_co_items

    def __add__(self, other):
        if not isinstance(other, deque):
            raise TypeError('can only concatenate deque (not "' + type(other).__name__ + '") to deque')
        out = self.copy()
        out.extend(other)
        return out

    def __iadd__(self, other):
        self.extend(other)
        return self

    def __mul__(self, n):
        return deque(self._mec_co_items * n, self._mec_co_max)

    def __repr__(self):
        if self._mec_co_max is None:
            return "deque(" + repr(self._mec_co_items) + ")"
        return "deque(" + repr(self._mec_co_items) + ", maxlen=" + str(self._mec_co_max) + ")"


class defaultdict(dict):
    def __init__(self, default_factory=None, /, *args, **kwargs):
        if default_factory is not None and not callable(default_factory):
            raise TypeError("first argument must be callable or None")
        super().__init__(*args, **kwargs)
        self.default_factory = default_factory

    def __missing__(self, key):
        if self.default_factory is None:
            raise KeyError(key)
        value = self.default_factory()
        self[key] = value
        return value

    def copy(self):
        return defaultdict(self.default_factory, self)

    def __copy__(self):
        return self.copy()

    def __repr__(self):
        return "defaultdict(" + repr(self.default_factory) + ", " + dict.__repr__(self) + ")"


class OrderedDict(dict):
    def move_to_end(self, key, last=True):
        value = self[key]
        del self[key]
        if last:
            self[key] = value
        else:
            rest = list(self.items())
            self.clear()
            self[key] = value
            self.update(rest)

    def popitem(self, last=True):
        if not self:
            raise KeyError("dictionary is empty")
        if last:
            key = list(self)[-1]
        else:
            key = list(self)[0]
        return key, self.pop(key)

    def copy(self):
        return OrderedDict(self)

    def __copy__(self):
        return self.copy()

    def __eq__(self, other):
        if isinstance(other, OrderedDict):
            return dict(self) == dict(other) and list(self) == list(other)
        return dict(self) == other

    def __ne__(self, other):
        return not self == other

    def __repr__(self):
        if not self:
            return "OrderedDict()"
        return "OrderedDict(" + dict.__repr__(self) + ")"


# A view over a list of mappings, searched first to last; writes and deletes go
# to the first. Not a dict subclass, as in CPython. keys() / values() / items()
# answer the views of the merged dict, whose repr is dict_keys(...) where
# CPython's is KeysView(ChainMap(...)).
class ChainMap[K, V]:
    def __init__(self, *maps):
        self.maps = list(maps) or [{}]

    def __missing__(self, key):
        raise KeyError(key)

    def __getitem__(self, key):
        for mapping in self.maps:
            if key in mapping:
                return mapping[key]
        return self.__missing__(key)

    def get(self, key, default=None):
        return self[key] if key in self else default

    def _mec_co_flat(self):
        d = {}
        for mapping in reversed(self.maps):
            for k in mapping:
                d[k] = None
        for k in d:
            d[k] = self[k]
        return d

    def __len__(self):
        return len(self._mec_co_flat())

    def __iter__(self):
        return iter(list(self._mec_co_flat()))

    def __contains__(self, key):
        for mapping in self.maps:
            if key in mapping:
                return True
        return False

    def __bool__(self):
        for mapping in self.maps:
            if mapping:
                return True
        return False

    def keys(self):
        return self._mec_co_flat().keys()

    def values(self):
        return self._mec_co_flat().values()

    def items(self):
        return self._mec_co_flat().items()

    def __eq__(self, other):
        if isinstance(other, ChainMap):
            return self._mec_co_flat() == other._mec_co_flat()
        if isinstance(other, dict):
            return self._mec_co_flat() == other
        return NotImplemented

    def __repr__(self):
        return type(self).__name__ + "(" + ", ".join([repr(m) for m in self.maps]) + ")"

    @classmethod
    def fromkeys(cls, iterable, value=None):
        return cls(dict.fromkeys(iterable, value))

    def copy(self):
        return type(self)(self.maps[0].copy(), *self.maps[1:])

    def __copy__(self):
        return self.copy()

    def new_child(self, m=None, **kwargs):
        if m is None:
            m = kwargs
        elif kwargs:
            m.update(kwargs)
        return type(self)(m, *self.maps)

    @property
    def parents(self):
        return type(self)(*self.maps[1:])

    def __setitem__(self, key, value):
        self.maps[0][key] = value

    def __delitem__(self, key):
        if key not in self.maps[0]:
            raise KeyError("Key not found in the first mapping: " + repr(key))
        del self.maps[0][key]

    def setdefault(self, key, default=None):
        if key not in self:
            self[key] = default
        return self[key]

    def update(self, other=(), /, **kwds):
        self.maps[0].update(other, **kwds)

    def popitem(self):
        if not self.maps[0]:
            raise KeyError("No keys found in the first mapping.")
        return self.maps[0].popitem()

    def pop(self, key, *args):
        if key in self.maps[0]:
            return self.maps[0].pop(key)
        if args:
            return args[0]
        raise KeyError("Key not found in the first mapping: " + repr(key))

    def clear(self):
        self.maps[0].clear()

    def __ior__(self, other):
        self.maps[0].update(other)
        return self

    def __or__(self, other):
        if not isinstance(other, (dict, ChainMap)):
            return NotImplemented
        m = self.copy()
        m.maps[0].update(other)
        return m

    def __ror__(self, other):
        if not isinstance(other, (dict, ChainMap)):
            return NotImplemented
        m = dict(other)
        for child in reversed(self.maps):
            m.update(child)
        return type(self)(m)


class Counter(dict):
    def __init__(self, iterable=None, /, **kwds):
        super().__init__()
        self.update(iterable, **kwds)

    def __missing__(self, key):
        return 0

    def total(self):
        return sum(self.values())

    # sorted() is stable, so equal counts keep their first-seen order, which is
    # what CPython's sorted / heapq.nlargest answer too.
    def most_common(self, n=None):
        ordered = sorted(self.items(), key=_mec_co_count, reverse=True)
        if n is None:
            return ordered
        return ordered[:max(n, 0)]

    def elements(self):
        out = []
        for elem, count in self.items():
            for _ in range(count):
                out.append(elem)
        return iter(out)

    def update(self, iterable=None, /, **kwds):
        if iterable is not None:
            if hasattr(iterable, "keys"):
                for elem in iterable.keys():
                    self[elem] = self.get(elem, 0) + iterable[elem]
            else:
                for elem in iterable:
                    self[elem] = self.get(elem, 0) + 1
        if kwds:
            self.update(kwds)

    def subtract(self, iterable=None, /, **kwds):
        if iterable is not None:
            if hasattr(iterable, "keys"):
                for elem in iterable.keys():
                    self[elem] = self.get(elem, 0) - iterable[elem]
            else:
                for elem in iterable:
                    self[elem] = self.get(elem, 0) - 1
        if kwds:
            self.subtract(kwds)

    def copy(self):
        return Counter(self)

    def __copy__(self):
        return self.copy()

    def __delitem__(self, elem):
        if elem in self:
            dict.__delitem__(self, elem)

    def __eq__(self, other):
        if not isinstance(other, Counter):
            return NotImplemented
        for c in (self, other):
            for e in c:
                if self[e] != other[e]:
                    return False
        return True

    def __ne__(self, other):
        if not isinstance(other, Counter):
            return NotImplemented
        return not self == other

    def __repr__(self):
        if not self:
            return "Counter()"
        return "Counter(" + repr(dict(self.most_common())) + ")"

    def __add__(self, other):
        if not isinstance(other, Counter):
            return NotImplemented
        result = Counter()
        for elem, count in self.items():
            if count + other[elem] > 0:
                result[elem] = count + other[elem]
        for elem, count in other.items():
            if elem not in self and count > 0:
                result[elem] = count
        return result

    def __sub__(self, other):
        if not isinstance(other, Counter):
            return NotImplemented
        result = Counter()
        for elem, count in self.items():
            if count - other[elem] > 0:
                result[elem] = count - other[elem]
        for elem, count in other.items():
            if elem not in self and count < 0:
                result[elem] = 0 - count
        return result

    def __or__(self, other):
        if not isinstance(other, Counter):
            return NotImplemented
        result = Counter()
        for elem, count in self.items():
            best = max(count, other[elem])
            if best > 0:
                result[elem] = best
        for elem, count in other.items():
            if elem not in self and count > 0:
                result[elem] = count
        return result

    def __and__(self, other):
        if not isinstance(other, Counter):
            return NotImplemented
        result = Counter()
        for elem, count in self.items():
            least = min(count, other[elem])
            if least > 0:
                result[elem] = least
        return result

    def __pos__(self):
        result = Counter()
        for elem, count in self.items():
            if count > 0:
                result[elem] = count
        return result

    def __neg__(self):
        result = Counter()
        for elem, count in self.items():
            if count < 0:
                result[elem] = 0 - count
        return result


# namedtuple builds a tuple subclass whose fields read through
# _mec_co_tuplegetter. Lib/collections builds __new__ with eval() to get a real
# signature; here one generic __new__ binds the arguments itself and raises the
# texts that signature would.
_mec_co_kwlist = ["False", "None", "True", "and", "as", "assert", "async", "await",
                  "break", "class", "continue", "def", "del", "elif", "else", "except",
                  "finally", "for", "from", "global", "if", "import", "in", "is",
                  "lambda", "nonlocal", "not", "or", "pass", "raise", "return", "try",
                  "while", "with", "yield"]


class _mec_co_tuplegetter:
    def __init__(self, index, doc):
        self._mec_co_index = index
        self.__doc__ = doc

    def __get__(self, obj, owner=None):
        if obj is None:
            return self
        return obj[self._mec_co_index]

    def __set__(self, obj, value):
        raise AttributeError("can't set attribute")

    def __repr__(self):
        return "_tuplegetter(" + repr(self._mec_co_index) + ", " + repr(self.__doc__) + ")"


def _mec_co_missing(names):
    quoted = [repr(n) for n in names]
    if len(quoted) == 1:
        return quoted[0]
    if len(quoted) == 2:
        return quoted[0] + " and " + quoted[1]
    return ", ".join(quoted[:-1]) + ", and " + quoted[-1]


def namedtuple(typename, field_names, *, rename=False, defaults=None, module=None):
    if isinstance(field_names, str):
        field_names = field_names.replace(",", " ").split()
    field_names = list(map(str, field_names))
    typename = str(typename)
    if rename:
        seen = set()
        for index, name in enumerate(field_names):
            if (not name.isidentifier() or name in _mec_co_kwlist
                    or name.startswith("_") or name in seen):
                field_names[index] = "_" + str(index)
            seen.add(name)
    for name in [typename] + field_names:
        if not name.isidentifier():
            raise ValueError("Type names and field names must be valid identifiers: " + repr(name))
        if name in _mec_co_kwlist:
            raise ValueError("Type names and field names cannot be a keyword: " + repr(name))
    seen = set()
    for name in field_names:
        if name.startswith("_") and not rename:
            raise ValueError("Field names cannot start with an underscore: " + repr(name))
        if name in seen:
            raise ValueError("Encountered duplicate field name: " + repr(name))
        seen.add(name)
    field_defaults = {}
    if defaults is not None:
        defaults = tuple(defaults)
        if len(defaults) > len(field_names):
            raise TypeError("Got more default values than field names")
        field_defaults = dict(reversed(list(zip(reversed(field_names), reversed(defaults)))))
    field_names = tuple(field_names)
    num_fields = len(field_names)
    repr_fmt = "(" + ", ".join(name + "=%r" for name in field_names) + ")"
    where = typename + ".__new__()"

    def __new__(cls, *args, **kwargs):
        if len(args) > num_fields:
            least = num_fields - len(field_defaults)
            if least == num_fields:
                takes = str(num_fields + 1)
            else:
                takes = "from " + str(least + 1) + " to " + str(num_fields + 1)
            raise TypeError(where + " takes " + takes + " positional arguments but "
                            + str(len(args) + 1) + " were given")
        values = list(args)
        for key in kwargs:
            if key not in field_names:
                raise TypeError(where + " got an unexpected keyword argument " + repr(key))
            if field_names.index(key) < len(args):
                raise TypeError(where + " got multiple values for argument " + repr(key))
        missing = []
        for index in range(len(args), num_fields):
            name = field_names[index]
            if name in kwargs:
                values.append(kwargs[name])
            elif name in field_defaults:
                values.append(field_defaults[name])
            else:
                missing.append(name)
        if missing:
            plural = "argument" if len(missing) == 1 else "arguments"
            raise TypeError(where + " missing " + str(len(missing)) + " required positional "
                            + plural + ": " + _mec_co_missing(missing))
        return tuple.__new__(cls, values)

    def _make(cls, iterable):
        result = tuple.__new__(cls, iterable)
        if len(result) != num_fields:
            raise TypeError("Expected " + str(num_fields) + " arguments, got " + str(len(result)))
        return result

    def _replace(self, /, **kwds):
        result = self._make(map(kwds.pop, field_names, self))
        if kwds:
            raise TypeError("Got unexpected field names: " + repr(list(kwds)))
        return result

    def __repr__(self):
        return self.__class__.__name__ + repr_fmt % self

    def _asdict(self):
        return dict(zip(self._fields, self))

    def __getnewargs__(self):
        return tuple(self)

    namespace = {
        "__doc__": typename + "(" + ", ".join(field_names) + ")",
        "__slots__": (),
        "_fields": field_names,
        "_field_defaults": field_defaults,
        "__new__": __new__,
        "_make": classmethod(_make),
        "__replace__": _replace,
        "_replace": _replace,
        "__repr__": __repr__,
        "_asdict": _asdict,
        "__getnewargs__": __getnewargs__,
        "__match_args__": field_names,
    }
    for index, name in enumerate(field_names):
        namespace[name] = _mec_co_tuplegetter(index, "Alias for field number " + str(index))
    result = type(typename, (tuple,), namespace)
    if module is not None:
        result.__module__ = module
    return result


def _mec_co_count(pair):
    return pair[1]


class _mec_co_module:
    __all__ = ["ChainMap", "Counter", "OrderedDict", "defaultdict", "deque", "namedtuple"]

    def __repr__(self):
        return "<module 'collections'>"


# collections.abc is a runtime module whose names stay unbound (its ABCs would
# need isinstance hooks); `import collections.abc` binds collections, so the
# submodule is an attribute here.
class _mec_co_abc_module:
    def __repr__(self):
        return "<module 'collections.abc'>"


_mec_co_module.abc = type("module", (_mec_co_abc_module,), {})()
_mec_co_module.abc.__name__ = "collections.abc"
_mec_mod_collections = type("module", (_mec_co_module,), {})()
_mec_mod_collections.__name__ = "collections"
_mec_co_module.deque = deque
_mec_co_module.defaultdict = defaultdict
_mec_co_module.OrderedDict = OrderedDict
_mec_co_module.Counter = Counter
_mec_co_module.ChainMap = ChainMap
_mec_co_module.namedtuple = staticmethod(namedtuple)
