# python-struct.py -- the struct module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_struct); helpers
# carry the _mec_sc prefix. The formats are Modules/_struct.c's: native (@,
# the default) sizes and alignment are a 64-bit little-endian machine's, = < >
# and ! the standard sizes with no alignment. Floats are encoded exactly from
# as_integer_ratio, rounding half to even into e and f, as a C cast does.
# pack_into and the buffer protocol beyond bytes and bytearray are left out.

import math as _mec_sc_math


class _mec_sc_error(Exception):
    pass


# Per code: (standard size, native size); native alignment is the native size.
_mec_sc_SIZES = {"x": (1, 1), "c": (1, 1), "b": (1, 1), "B": (1, 1), "?": (1, 1), "h": (2, 2), "H": (2, 2),
                 "i": (4, 4), "I": (4, 4), "l": (4, 8), "L": (4, 8), "q": (8, 8), "Q": (8, 8), "n": (0, 8),
                 "N": (0, 8), "e": (2, 2), "f": (4, 4), "d": (8, 8), "s": (1, 1), "p": (1, 1), "P": (0, 8)}


# fmt as (little, [(code, count, offset, size)], total size). An s or p item's
# count is its length; every other code appears count times, one entry each.
def _mec_sc_parse(fmt):
    if isinstance(fmt, (bytes, bytearray)):
        fmt = fmt.decode("ascii")
    if not isinstance(fmt, str):
        raise TypeError("Struct() argument 1 must be a str or bytes object, not " + type(fmt).__name__)
    native = True
    little = True
    i = 0
    if fmt[:1] in ("@", "=", "<", ">", "!"):
        native = fmt[0] == "@"
        little = fmt[0] in ("@", "=", "<")
        i = 1
    items = []
    off = 0
    n = len(fmt)
    while i < n:
        c = fmt[i]
        if c.isspace():
            i += 1
            continue
        count = 1
        if c.isdigit():
            j = i
            while j < n and fmt[j].isdigit():
                j += 1
            count = int(fmt[i:j])
            if j >= n:
                raise _mec_sc_error("repeat count given without format specifier")
            i = j
            c = fmt[i]
        if c not in _mec_sc_SIZES or (not native and c in "nNP"):
            raise _mec_sc_error("bad char in struct format")
        size = _mec_sc_SIZES[c][1 if native else 0]
        if native and c not in "xcbB?sp":
            off = (off + size - 1) // size * size
        if c in "sp":
            items.append((c, count, off, count))
            off += count
        elif c == "x":
            off += count
        else:
            for _ in range(count):
                items.append((c, 1, off, size))
                off += size
        i += 1
    return little, items, off


def _mec_sc_int(v):
    if isinstance(v, bool):
        return int(v)
    if isinstance(v, int):
        return v
    if hasattr(type(v), "__index__"):
        return v.__index__()
    raise _mec_sc_error("required argument is not an integer")


# The IEEE 754 bits of x with eb exponent and mb fraction bits, rounded half to
# even.
def _mec_sc_enc(x, eb, mb, code):
    bias = (1 << (eb - 1)) - 1
    top = (1 << eb) - 1
    sign = (1 << (eb + mb)) if _mec_sc_math.copysign(1.0, x) < 0 else 0
    if x != x:
        return sign | (top << mb) | (1 << (mb - 1))
    x = abs(x)
    if x == _mec_sc_math.inf:
        return sign | (top << mb)
    if x == 0:
        return sign
    num, den = x.as_integer_ratio()
    e = num.bit_length() - den.bit_length()
    if (num << max(0, -e)) < (den << max(0, e)):
        e -= 1
    if e < 1 - bias:
        e = 1 - bias
    sh = mb - e
    if sh >= 0:
        num = num << sh
    else:
        den = den << -sh
    q, r = divmod(num, den)
    if 2 * r > den or (2 * r == den and q % 2 == 1):
        q += 1
    if q >= (2 << mb):
        q >>= 1
        e += 1
    if e > bias:
        raise OverflowError("float too large to pack with " + code + " format")
    if q < (1 << mb):
        return sign | q
    return sign | ((e + bias) << mb) | (q - (1 << mb))


def _mec_sc_dec(bits, eb, mb):
    bias = (1 << (eb - 1)) - 1
    top = (1 << eb) - 1
    neg = (bits >> (eb + mb)) & 1
    exp = (bits >> mb) & top
    frac = bits & ((1 << mb) - 1)
    if exp == top:
        v = _mec_sc_math.nan if frac else _mec_sc_math.inf
    elif exp == 0:
        v = _mec_sc_math.ldexp(float(frac), 1 - bias - mb)
    else:
        v = _mec_sc_math.ldexp(float(frac + (1 << mb)), exp - bias - mb)
    return -v if neg else v


_mec_sc_FLOAT = {"e": (5, 10), "f": (8, 23), "d": (11, 52)}


def _mec_sc_pack(little, items, total, args):
    want = len(items)
    if len(args) != want:
        raise _mec_sc_error("pack expected " + str(want) + " items for packing (got " + str(len(args)) + ")")
    out = [0] * total
    for k in range(want):
        code, count, off, size = items[k]
        v = args[k]
        if code in "sp":
            if not isinstance(v, (bytes, bytearray)):
                raise _mec_sc_error("argument for '" + code + "' must be a bytes object")
            data = list(v)
            if code == "p":
                n = min(len(data), count - 1, 255) if count > 0 else 0
                if count > 0:
                    out[off] = n
                    for j in range(n):
                        out[off + 1 + j] = data[j]
            else:
                for j in range(min(len(data), count)):
                    out[off + j] = data[j]
            continue
        if code == "c":
            if not isinstance(v, (bytes, bytearray)) or len(v) != 1:
                raise _mec_sc_error("char format requires a bytes object of length 1")
            out[off] = v[0]
            continue
        if code == "?":
            out[off] = 1 if v else 0
            continue
        if code in _mec_sc_FLOAT:
            if not isinstance(v, (int, float)) and not hasattr(type(v), "__float__"):
                raise _mec_sc_error("required argument is not a float")
            eb, mb = _mec_sc_FLOAT[code]
            n = _mec_sc_enc(float(v), eb, mb, code)
        else:
            n = _mec_sc_int(v)
            bits = size * 8
            if code in "BHILQNP":
                lo, hi = 0, (1 << bits) - 1
            else:
                lo, hi = -(1 << (bits - 1)), (1 << (bits - 1)) - 1
            if not lo <= n <= hi:
                raise _mec_sc_error("'" + code + "' format requires " + str(lo) + " <= number <= " + str(hi))
            n &= (1 << bits) - 1
        for j in range(size):
            byte = (n >> (8 * j)) & 255
            out[off + (j if little else size - 1 - j)] = byte
    return bytes(out)


def _mec_sc_unpack(little, items, total, buf, start):
    data = list(buf)
    out = []
    for code, count, off, size in items:
        at = start + off
        if code == "s":
            out.append(bytes(data[at:at + count]))
            continue
        if code == "p":
            n = min(data[at], count - 1) if count > 0 else 0
            out.append(bytes(data[at + 1:at + 1 + n]))
            continue
        if code == "c":
            out.append(bytes([data[at]]))
            continue
        n = 0
        for j in range(size):
            byte = data[at + (j if little else size - 1 - j)]
            n |= byte << (8 * j)
        if code == "?":
            out.append(n != 0)
        elif code in _mec_sc_FLOAT:
            eb, mb = _mec_sc_FLOAT[code]
            out.append(_mec_sc_dec(n, eb, mb))
        elif code in "BHILQNP":
            out.append(n)
        else:
            out.append(n - (1 << (8 * size)) if n >> (8 * size - 1) else n)
    return tuple(out)


def _mec_sc_buflen(buf):
    if not isinstance(buf, (bytes, bytearray)):
        raise TypeError("a bytes-like object is required, not '" + type(buf).__name__ + "'")
    return len(buf)


class _mec_sc_Struct:
    def __init__(self, format):
        self._little, self._items, self.size = _mec_sc_parse(format)
        self.format = format if isinstance(format, str) else format.decode("ascii")

    def pack(self, *args):
        return _mec_sc_pack(self._little, self._items, self.size, args)

    def unpack(self, buffer):
        if _mec_sc_buflen(buffer) != self.size:
            raise _mec_sc_error("unpack requires a buffer of " + str(self.size) + " bytes")
        return _mec_sc_unpack(self._little, self._items, self.size, buffer, 0)

    def unpack_from(self, buffer, offset=0):
        n = _mec_sc_buflen(buffer)
        if offset < 0:
            offset += n
        if offset < 0 or n - offset < self.size:
            raise _mec_sc_error("unpack_from requires a buffer of at least " + str(self.size + offset)
                                + " bytes for unpacking " + str(self.size) + " bytes at offset " + str(offset)
                                + " (actual buffer size is " + str(n) + ")")
        return _mec_sc_unpack(self._little, self._items, self.size, buffer, offset)

    def iter_unpack(self, buffer):
        n = _mec_sc_buflen(buffer)
        if self.size == 0:
            raise _mec_sc_error("cannot iteratively unpack with a struct of length 0")
        if n % self.size:
            raise _mec_sc_error("iterative unpacking requires a buffer of a multiple of " + str(self.size)
                                + " bytes")
        return iter([_mec_sc_unpack(self._little, self._items, self.size, buffer, k)
                     for k in range(0, n, self.size)])

    def __repr__(self):
        return "Struct(" + repr(self.format) + ")"


for _mec_sc_cls, _mec_sc_name in ((_mec_sc_Struct, "Struct"), (_mec_sc_error, "error")):
    _mec_sc_cls.__name__ = _mec_sc_name
    _mec_sc_cls.__qualname__ = _mec_sc_name
    _mec_sc_cls.__module__ = "struct"


class _mec_sc_module:
    __all__ = ["calcsize", "pack", "unpack", "unpack_from", "iter_unpack", "Struct", "error"]
    Struct = _mec_sc_Struct
    error = _mec_sc_error

    @staticmethod
    def calcsize(format):
        return _mec_sc_Struct(format).size

    @staticmethod
    def pack(format, *args):
        return _mec_sc_Struct(format).pack(*args)

    @staticmethod
    def unpack(format, buffer):
        return _mec_sc_Struct(format).unpack(buffer)

    @staticmethod
    def unpack_from(format, buffer, offset=0):
        return _mec_sc_Struct(format).unpack_from(buffer, offset)

    @staticmethod
    def iter_unpack(format, buffer):
        return _mec_sc_Struct(format).iter_unpack(buffer)

    def __repr__(self):
        return "<module 'struct'>"


_mec_mod_struct = type("module", (_mec_sc_module,), {})()
_mec_mod_struct.__name__ = "struct"
