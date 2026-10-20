# python-array.py -- the array module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_array); helpers
# carry the _mec_ar prefix. An array keeps its items in a list, each checked
# and converted as Modules/arraymodule.c does on the way in (an 'f' item is
# rounded to single precision), with the item sizes of a 64-bit little-endian
# machine; tobytes and frombytes go through the struct module. The buffer
# protocol, buffer_info and the pickle hooks are left out.

import struct as _mec_ar_struct

_mec_ar_CODES = "bBuwhHiIlLqQfd"
_mec_ar_SIZE = {"b": 1, "B": 1, "u": 4, "w": 4, "h": 2, "H": 2, "i": 4, "I": 4, "l": 8, "L": 8, "q": 8, "Q": 8,
                "f": 4, "d": 8}
# Per integer code: (low, high, too-small text, too-big text).
_mec_ar_RANGE = {
    "b": (-128, 127, "signed char is less than minimum", "signed char is greater than maximum"),
    "B": (0, 255, "unsigned byte integer is less than minimum", "unsigned byte integer is greater than maximum"),
    "h": (-32768, 32767, "signed short integer is less than minimum", "signed short integer is greater than maximum"),
    "H": (0, 65535, "unsigned short is less than minimum", "unsigned short is greater than maximum"),
    "i": (-2147483648, 2147483647, "signed integer is less than minimum", "signed integer is greater than maximum"),
    "I": (0, 4294967295, "can't convert negative value to unsigned int", "unsigned int is greater than maximum"),
    "l": (-9223372036854775808, 9223372036854775807, "Python int too large to convert to C long",
          "Python int too large to convert to C long"),
    "L": (0, 18446744073709551615, "can't convert negative value to unsigned int",
          "Python int too large to convert to C unsigned long"),
    "q": (-9223372036854775808, 9223372036854775807, "int too big to convert", "int too big to convert"),
    "Q": (0, 18446744073709551615, "can't convert negative int to unsigned", "int too big to convert"),
}


def _mec_ar_item(tc, v):
    if tc in "uw":
        if not isinstance(v, str):
            raise TypeError("array item must be a unicode character, not " + type(v).__name__)
        if len(v) != 1:
            raise TypeError("array item must be a unicode character, not a string of length " + str(len(v)))
        return v
    if tc in "fd":
        if isinstance(v, (int, float)):
            f = float(v)
        elif hasattr(type(v), "__float__"):
            f = v.__float__()
        else:
            raise TypeError("must be real number, not " + type(v).__name__)
        if tc == "f":
            return _mec_ar_struct.unpack("<f", _mec_ar_struct.pack("<f", f))[0]
        return f
    if isinstance(v, bool):
        n = int(v)
    elif isinstance(v, int):
        n = v
    elif hasattr(type(v), "__index__"):
        n = v.__index__()
    else:
        raise TypeError("'" + type(v).__name__ + "' object cannot be interpreted as an integer")
    lo, hi, small, big = _mec_ar_RANGE[tc]
    if n < lo:
        raise OverflowError(small)
    if n > hi:
        raise OverflowError(big)
    return n


class _mec_ar_array:
    def __init__(self, typecode, initializer=None, /):
        if not isinstance(typecode, str):
            raise TypeError("array() argument 1 must be a unicode character, not " + type(typecode).__name__)
        if len(typecode) != 1:
            raise TypeError("array() argument 1 must be a unicode character, not a string of length "
                            + str(len(typecode)))
        if typecode not in _mec_ar_CODES:
            raise ValueError("bad typecode (must be b, B, u, w, h, H, i, I, l, L, q, Q, f or d)")
        self._tc = typecode
        self._a = []
        if initializer is None:
            return
        if isinstance(initializer, str):
            if typecode not in "uw":
                raise TypeError("cannot use a str to initialize an array with typecode '" + typecode + "'")
            self._a = list(initializer)
        elif isinstance(initializer, (bytes, bytearray)):
            self.frombytes(initializer)
        elif isinstance(initializer, _mec_ar_array) and initializer._tc in "uw" and typecode not in "uw":
            raise TypeError("cannot use a unicode array to initialize an array with typecode '" + typecode + "'")
        elif isinstance(initializer, _mec_ar_array):
            self.extend(list(initializer._a))
        else:
            self.extend(initializer)

    @property
    def typecode(self):
        return self._tc

    @property
    def itemsize(self):
        return _mec_ar_SIZE[self._tc]

    def _mec_new(self, items):
        a = _mec_ar_array(self._tc)
        a._a = items
        return a

    def append(self, v):
        self._a.append(_mec_ar_item(self._tc, v))

    def extend(self, it):
        if isinstance(it, _mec_ar_array):
            if it._tc != self._tc:
                raise TypeError("can only extend with array of same kind")
            self._a.extend(it._a)
            return
        for v in it:
            self._a.append(_mec_ar_item(self._tc, v))

    def fromlist(self, lst):
        if not isinstance(lst, list):
            raise TypeError("arg must be list")
        items = [_mec_ar_item(self._tc, v) for v in lst]
        self._a.extend(items)

    def tolist(self):
        return list(self._a)

    def insert(self, i, v):
        self._a.insert(i, _mec_ar_item(self._tc, v))

    def pop(self, i=-1):
        if not self._a:
            raise IndexError("pop from empty array")
        n = len(self._a)
        if not -n <= i < n:
            raise IndexError("pop index out of range")
        return self._a.pop(i)

    def remove(self, v):
        for k in range(len(self._a)):
            if self._a[k] == v:
                del self._a[k]
                return
        raise ValueError("array.remove(x): x not in array")

    def index(self, v, start=0, stop=None):
        n = len(self._a)
        stop = n if stop is None else stop
        for k in range(n)[start:stop]:
            if self._a[k] == v:
                return k
        raise ValueError("array.index(x): x not in array")

    def count(self, v):
        return sum(1 for x in self._a if x == v)

    def reverse(self):
        self._a.reverse()

    def _mec_fmt(self, n):
        return "<" + str(n) + ("I" if self._tc in "uw" else self._tc)

    def tobytes(self):
        items = [ord(c) for c in self._a] if self._tc in "uw" else self._a
        return _mec_ar_struct.pack(self._mec_fmt(len(items)), *items)

    def frombytes(self, b):
        if not isinstance(b, (bytes, bytearray)):
            raise TypeError("a bytes-like object is required, not '" + type(b).__name__ + "'")
        size = _mec_ar_SIZE[self._tc]
        if len(b) % size:
            raise ValueError("bytes length not a multiple of item size")
        items = list(_mec_ar_struct.unpack(self._mec_fmt(len(b) // size), b))
        if self._tc in "uw":
            items = [chr(c) for c in items]
        self._a.extend(items)

    def byteswap(self):
        if self._tc in "uw":
            raise RuntimeError("don't know how to byteswap this array type")
        size = _mec_ar_SIZE[self._tc]
        raw = list(self.tobytes())
        out = []
        for k in range(0, len(raw), size):
            out.extend(reversed(raw[k:k + size]))
        self._a = []
        self.frombytes(bytes(out))

    def tounicode(self):
        if self._tc not in "uw":
            raise ValueError("tounicode() may only be called on unicode type arrays ('u' or 'w')")
        return "".join(self._a)

    def fromunicode(self, s):
        if self._tc not in "uw":
            raise ValueError("fromunicode() may only be called on unicode type arrays ('u' or 'w')")
        self._a.extend(list(s))

    def __len__(self):
        return len(self._a)

    def __iter__(self):
        return iter(self._a)

    def __contains__(self, v):
        return v in self._a

    def __getitem__(self, i):
        if isinstance(i, slice):
            return self._mec_new(self._a[i])
        n = len(self._a)
        if not -n <= i < n:
            raise IndexError("array index out of range")
        return self._a[i]

    def __setitem__(self, i, v):
        if isinstance(i, slice):
            if not isinstance(v, _mec_ar_array):
                raise TypeError("can only assign array (not \"" + type(v).__name__ + "\") to array slice")
            if v._tc != self._tc:
                raise TypeError("bad argument type for built-in operation")
            self._a[i] = list(v._a)
            return
        n = len(self._a)
        if not -n <= i < n:
            raise IndexError("array assignment index out of range")
        self._a[i] = _mec_ar_item(self._tc, v)

    def __delitem__(self, i):
        if not isinstance(i, slice):
            n = len(self._a)
            if not -n <= i < n:
                raise IndexError("array assignment index out of range")
        del self._a[i]

    def __add__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        if other._tc != self._tc:
            raise TypeError("bad argument type for built-in operation")
        return self._mec_new(self._a + other._a)

    def __iadd__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        if other._tc != self._tc:
            raise TypeError("can only extend with array of same kind")
        self._a.extend(other._a)
        return self

    def __mul__(self, n):
        if not isinstance(n, int):
            return NotImplemented
        return self._mec_new(self._a * n)

    __rmul__ = __mul__

    def __imul__(self, n):
        self._a = self._a * n
        return self

    def __eq__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        return self._a == other._a

    def __ne__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        return self._a != other._a

    def __lt__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        return self._a < other._a

    def __le__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        return self._a <= other._a

    def __gt__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        return self._a > other._a

    def __ge__(self, other):
        if not isinstance(other, _mec_ar_array):
            return NotImplemented
        return self._a >= other._a

    __hash__ = None

    def __copy__(self):
        return self._mec_new(list(self._a))

    def __deepcopy__(self, memo):
        return self._mec_new(list(self._a))

    def __repr__(self):
        if not self._a:
            return "array('" + self._tc + "')"
        if self._tc in "uw":
            return "array('" + self._tc + "', " + repr("".join(self._a)) + ")"
        return "array('" + self._tc + "', " + repr(self._a) + ")"


_mec_ar_array.__name__ = "array"
_mec_ar_array.__qualname__ = "array"
_mec_ar_array.__module__ = "array"


class _mec_ar_module:
    __all__ = ["array", "ArrayType", "typecodes"]
    array = _mec_ar_array
    ArrayType = _mec_ar_array
    typecodes = _mec_ar_CODES

    def __repr__(self):
        return "<module 'array'>"


_mec_mod_array = type("module", (_mec_ar_module,), {})()
_mec_mod_array.__name__ = "array"
