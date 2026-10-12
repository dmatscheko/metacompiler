# python-collections.py -- the collections module's containers, written in
# Python and compiled with a program that imports them (core.importFile in both
# python grammars), so one implementation serves all three engines.
#
# Imports here are flat (the module's top-level names bind directly), so
# deque, defaultdict, OrderedDict and Counter are bound as themselves and the
# module object carrying them is _mec_mod_collections; helpers carry the
# _mec_co prefix so that a program's own names cannot replace them.
#
# The behaviour is Lib/collections/__init__.py's, error texts included.
# defaultdict, OrderedDict and Counter are dict subclasses, as there; deque
# keeps its items in a list, so popleft and appendleft are linear rather than
# constant time.


class deque:
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


def _mec_co_count(pair):
    return pair[1]


class _mec_co_module:
    def __repr__(self):
        return "<module 'collections'>"


_mec_mod_collections = _mec_co_module()
_mec_co_module.deque = deque
_mec_co_module.defaultdict = defaultdict
_mec_co_module.OrderedDict = OrderedDict
_mec_co_module.Counter = Counter
