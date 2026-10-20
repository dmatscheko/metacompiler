# python-fractions.py -- the fractions module, written in Python and compiled
# with a program that imports it (core.importFile in both python grammars), so
# one implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_fractions);
# helpers carry the _mec_fr prefix. Fraction follows Lib/fractions.py (3.14):
# normalized by gcd with the sign on the numerator, the same mixed-type rules
# (an int stays exact, a float or complex operand makes the result one), half
# to even rounding, limit_denominator's continued fractions and the numeric
# hash that makes hash(Fraction(1, 2)) == hash(0.5). __format__ takes the
# fill/align/sign/width/precision forms with f, F, % or no type; e, E, g, G go
# through float. There is no decimal module, so no Decimal operand.

import math as _mec_fr_math

_mec_fr_MODULUS = 2305843009213693951
_mec_fr_INF = 314159


def _mec_fr_make(n, d):
    f = object.__new__(_mec_fr_Fraction)
    f._numerator = n
    f._denominator = d
    return f


def _mec_fr_digits(s):
    return s != "" and all(c.isdecimal() or c == "_" for c in s) and not s.startswith("_") \
        and not s.endswith("_") and "__" not in s


# The string forms Fraction() reads: [sign] num [/ denom], or [sign] num
# [. decimal] [e exp], with optional surrounding whitespace and digit
# underscores. None when s is none of them.
def _mec_fr_parse(s):
    t = s.strip()
    neg = False
    if t[:1] in ("+", "-"):
        neg = t[0] == "-"
        t = t[1:]
    if not (t[:1].isdecimal() or (t[:1] == "." and t[1:2].isdecimal())):
        return None
    slash = t.find("/")
    if slash >= 0:
        num, den = t[:slash].rstrip(), t[slash + 1:].lstrip()
        if not _mec_fr_digits(num) or not _mec_fr_digits(den):
            return None
        n, d = int(num), int(den)
    else:
        exp = 0
        cut = -1
        for i, c in enumerate(t):
            if c in "eE":
                cut = i
                break
        if cut >= 0:
            es = t[cut + 1:]
            body = es[1:] if es[:1] in ("+", "-") else es
            if not _mec_fr_digits(body):
                return None
            exp = int(es)
            t = t[:cut]
        dot = t.find(".")
        num, dec = (t, "") if dot < 0 else (t[:dot], t[dot + 1:])
        if (num != "" and not _mec_fr_digits(num)) or (dec != "" and not _mec_fr_digits(dec)):
            return None
        n = int(num) if num != "" else 0
        d = 1
        if dec != "":
            dec = dec.replace("_", "")
            scale = 10 ** len(dec)
            n = n * scale + int(dec)
            d = d * scale
        if exp >= 0:
            n = n * 10 ** exp
        else:
            d = d * 10 ** -exp
    return (-n if neg else n), d


def _mec_fr_round_half_even(n, d):
    fl, rem = divmod(n, d)
    if rem * 2 < d:
        return fl
    if rem * 2 > d:
        return fl + 1
    return fl if fl % 2 == 0 else fl + 1


# Two operands of a binary operator as (numerator, denominator) pairs, None
# when the other one is no int or Fraction.
def _mec_fr_pair(b):
    if isinstance(b, _mec_fr_Fraction):
        return b._numerator, b._denominator
    if isinstance(b, int):
        return int(b), 1
    return None


class _mec_fr_Fraction:
    def __init__(self, numerator=0, denominator=None):
        if denominator is None:
            if isinstance(numerator, int):
                n, d = int(numerator), 1
            elif isinstance(numerator, _mec_fr_Fraction):
                n, d = numerator._numerator, numerator._denominator
            elif isinstance(numerator, float):
                n, d = numerator.as_integer_ratio()
            elif isinstance(numerator, str):
                nd = _mec_fr_parse(numerator)
                if nd is None:
                    raise ValueError("Invalid literal for Fraction: " + repr(numerator))
                n, d = nd
            elif hasattr(numerator, "as_integer_ratio"):
                n, d = numerator.as_integer_ratio()
            else:
                raise TypeError("argument should be a string or a Rational instance")
        else:
            a = _mec_fr_pair(numerator)
            b = _mec_fr_pair(denominator)
            if a is None or b is None:
                raise TypeError("both arguments should be Rational instances")
            n, d = a[0] * b[1], b[0] * a[1]
        if d == 0:
            raise ZeroDivisionError("Fraction(" + str(n) + ", 0)")
        g = _mec_fr_math.gcd(n, d)
        if d < 0:
            g = -g
        self._numerator = n // g
        self._denominator = d // g

    @classmethod
    def from_float(cls, f):
        if isinstance(f, int):
            return cls(f)
        if not isinstance(f, float):
            raise TypeError(cls.__name__ + ".from_float() only takes floats, not " + repr(f)
                            + " (" + type(f).__name__ + ")")
        return cls(*f.as_integer_ratio())

    @classmethod
    def from_number(cls, number):
        if isinstance(number, (int, _mec_fr_Fraction, float)) and not isinstance(number, bool) or hasattr(number, "as_integer_ratio"):
            return cls(number)
        raise TypeError(cls.__name__ + ".from_number() only takes numbers, not " + repr(number)
                        + " (" + type(number).__name__ + ")")

    @property
    def numerator(self):
        return self._numerator

    @property
    def denominator(self):
        return self._denominator

    def as_integer_ratio(self):
        return (self._numerator, self._denominator)

    def is_integer(self):
        return self._denominator == 1

    def limit_denominator(self, max_denominator=1000000):
        if max_denominator < 1:
            raise ValueError("max_denominator should be at least 1")
        if self._denominator <= max_denominator:
            return _mec_fr_make(self._numerator, self._denominator)
        p0, q0, p1, q1 = 0, 1, 1, 0
        n, d = self._numerator, self._denominator
        while True:
            a = n // d
            q2 = q0 + a * q1
            if q2 > max_denominator:
                break
            p0, q0, p1, q1 = p1, q1, p0 + a * p1, q2
            n, d = d, n - a * d
        k = (max_denominator - q0) // q1
        if 2 * d * (q0 + k * q1) <= self._denominator:
            return _mec_fr_make(p1, q1)
        return _mec_fr_make(p0 + k * p1, q0 + k * q1)

    def __repr__(self):
        return type(self).__name__ + "(" + str(self._numerator) + ", " + str(self._denominator) + ")"

    def __str__(self):
        if self._denominator == 1:
            return str(self._numerator)
        return str(self._numerator) + "/" + str(self._denominator)

    # [[fill]align][sign][width][.precision][f|F|%] on the exact value, rounded
    # half to even; no type is str() padded; e/E/g/G go through float.
    def __format__(self, spec):
        if spec == "":
            return str(self)
        typ = spec[-1] if spec[-1] in "eEfFgG%" else ""
        head = spec[:-1] if typ else spec
        if typ in "eEgG" and typ != "":
            return format(float(self), spec)
        fill, align = " ", ""
        if len(head) >= 2 and head[1] in "<>^=":
            fill, align, head = head[0], head[1], head[2:]
        elif head[:1] in ("<", ">", "^", "="):
            align, head = head[0], head[1:]
        sign = "-"
        if head[:1] in ("+", "-", " "):
            sign, head = head[0], head[1:]
        if head[:1] == "0" and align == "":
            fill, align, head = "0", "=", head[1:]
        prec = None
        if "." in head:
            head, p = head.split(".", 1)
            prec = int(p)
        width = int(head) if head else 0
        if typ == "":
            body, neg = str(abs(self)), self._numerator < 0
            if align == "":
                align = ">"
        else:
            if prec is None:
                prec = 6
            n, d = abs(self._numerator), self._denominator
            if typ == "%":
                n = n * 100
            q = _mec_fr_round_half_even(n * 10 ** prec, d)
            digits = str(q).rjust(prec + 1, "0")
            body = digits[:len(digits) - prec] + ("." + digits[len(digits) - prec:] if prec > 0 else "")
            if typ == "%":
                body = body + "%"
            neg = self._numerator < 0
            if align == "":
                align = ">"
        lead = "-" if neg else ("+" if sign == "+" else (" " if sign == " " else ""))
        pad = width - len(lead) - len(body)
        if pad <= 0:
            return lead + body
        if align == "<":
            return lead + body + fill * pad
        if align == "^":
            return fill * (pad // 2) + lead + body + fill * (pad - pad // 2)
        if align == "=":
            return lead + fill * pad + body
        return fill * pad + lead + body

    def __add__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            return _mec_fr_Fraction(a._numerator * p[1] + p[0] * a._denominator, a._denominator * p[1])
        if isinstance(b, (float, complex)):
            return (float(a) if isinstance(b, float) else complex(float(a))) + b
        return NotImplemented

    def __radd__(b, a):
        if isinstance(a, int):
            return _mec_fr_Fraction(a * b._denominator + b._numerator, b._denominator)
        if isinstance(a, (float, complex)):
            return a + float(b)
        return NotImplemented

    def __sub__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            return _mec_fr_Fraction(a._numerator * p[1] - p[0] * a._denominator, a._denominator * p[1])
        if isinstance(b, (float, complex)):
            return (float(a) if isinstance(b, float) else complex(float(a))) - b
        return NotImplemented

    def __rsub__(b, a):
        if isinstance(a, int):
            return _mec_fr_Fraction(a * b._denominator - b._numerator, b._denominator)
        if isinstance(a, (float, complex)):
            return a - float(b)
        return NotImplemented

    def __mul__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            return _mec_fr_Fraction(a._numerator * p[0], a._denominator * p[1])
        if isinstance(b, (float, complex)):
            return (float(a) if isinstance(b, float) else complex(float(a))) * b
        return NotImplemented

    def __rmul__(b, a):
        if isinstance(a, int):
            return _mec_fr_Fraction(a * b._numerator, b._denominator)
        if isinstance(a, (float, complex)):
            return a * float(b)
        return NotImplemented

    def __truediv__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            if p[0] == 0:
                raise ZeroDivisionError("Fraction(" + str(p[1]) + ", 0)")
            return _mec_fr_Fraction(a._numerator * p[1], a._denominator * p[0])
        if isinstance(b, (float, complex)):
            return (float(a) if isinstance(b, float) else complex(float(a))) / b
        return NotImplemented

    def __rtruediv__(b, a):
        if isinstance(a, int):
            if b._numerator == 0:
                raise ZeroDivisionError("Fraction(" + str(b._denominator) + ", 0)")
            return _mec_fr_Fraction(a * b._denominator, b._numerator)
        if isinstance(a, (float, complex)):
            return a / float(b)
        return NotImplemented

    def __floordiv__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            return (a._numerator * p[1]) // (a._denominator * p[0])
        if isinstance(b, float):
            return float(a) // b
        return NotImplemented

    def __rfloordiv__(b, a):
        if isinstance(a, int):
            return (a * b._denominator) // b._numerator
        if isinstance(a, float):
            return a // float(b)
        return NotImplemented

    def __mod__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            return _mec_fr_Fraction((a._numerator * p[1]) % (p[0] * a._denominator), a._denominator * p[1])
        if isinstance(b, float):
            return float(a) % b
        return NotImplemented

    def __rmod__(b, a):
        if isinstance(a, int):
            return _mec_fr_Fraction((a * b._denominator) % b._numerator, b._denominator)
        if isinstance(a, float):
            return a % float(b)
        return NotImplemented

    def __divmod__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            q, r = divmod(a._numerator * p[1], a._denominator * p[0])
            return q, _mec_fr_Fraction(r, a._denominator * p[1])
        if isinstance(b, float):
            return divmod(float(a), b)
        return NotImplemented

    def __rdivmod__(b, a):
        if isinstance(a, int):
            q, r = divmod(a * b._denominator, b._numerator)
            return q, _mec_fr_Fraction(r, b._denominator)
        if isinstance(a, float):
            return divmod(a, float(b))
        return NotImplemented

    def __pow__(a, b, modulo=None):
        if modulo is not None:
            return NotImplemented
        p = _mec_fr_pair(b)
        if p is not None:
            if p[1] == 1:
                power = p[0]
                if power >= 0:
                    return _mec_fr_make(a._numerator ** power, a._denominator ** power)
                if a._numerator > 0:
                    return _mec_fr_make(a._denominator ** -power, a._numerator ** -power)
                if a._numerator == 0:
                    raise ZeroDivisionError("Fraction(" + str(a._denominator ** -power) + ", 0)")
                return _mec_fr_make((-a._denominator) ** -power, (-a._numerator) ** -power)
            return float(a) ** (p[0] / p[1])
        if isinstance(b, (float, complex)):
            return float(a) ** b
        return NotImplemented

    def __rpow__(b, a, modulo=None):
        if modulo is not None:
            return NotImplemented
        if b._denominator == 1 and b._numerator >= 0:
            return a ** b._numerator
        if isinstance(a, int):
            return _mec_fr_Fraction(a) ** b
        if b._denominator == 1:
            return a ** b._numerator
        return a ** float(b)

    def __pos__(a):
        return _mec_fr_make(a._numerator, a._denominator)

    def __neg__(a):
        return _mec_fr_make(-a._numerator, a._denominator)

    def __abs__(a):
        return _mec_fr_make(abs(a._numerator), a._denominator)

    def __int__(a):
        if a._numerator < 0:
            return -(-a._numerator // a._denominator)
        return a._numerator // a._denominator

    def __trunc__(a):
        return a.__int__()

    def __floor__(a):
        return a._numerator // a._denominator

    def __ceil__(a):
        return -(-a._numerator // a._denominator)

    def __round__(self, ndigits=None):
        if ndigits is None:
            return _mec_fr_round_half_even(self._numerator, self._denominator)
        shift = 10 ** abs(ndigits)
        if ndigits > 0:
            return _mec_fr_Fraction(round(self * shift), shift)
        return _mec_fr_Fraction(round(self / shift) * shift)

    def __float__(a):
        return a._numerator / a._denominator

    def __complex__(a):
        return complex(float(a))

    def __hash__(self):
        try:
            dinv = pow(self._denominator, -1, _mec_fr_MODULUS)
        except ValueError:
            h = _mec_fr_INF
        else:
            h = hash(hash(abs(self._numerator)) * dinv)
        result = h if self._numerator >= 0 else -h
        return -2 if result == -1 else result

    def __eq__(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            return a._numerator == p[0] and a._denominator == p[1]
        if isinstance(b, complex) and b.imag == 0:
            b = b.real
        if isinstance(b, float):
            if _mec_fr_math.isnan(b) or _mec_fr_math.isinf(b):
                return 0.0 == b
            return a == _mec_fr_Fraction(b)
        return NotImplemented

    # The four orderings: exact against an int or a Fraction, exact against a
    # finite float, and as 0.0 does against an infinity or a NaN.
    def _mec_fr_cmp(a, b):
        p = _mec_fr_pair(b)
        if p is not None:
            return a._numerator * p[1], a._denominator * p[0]
        if isinstance(b, float):
            if _mec_fr_math.isnan(b) or _mec_fr_math.isinf(b):
                return 0.0, b
            f = _mec_fr_Fraction(b)
            return a._numerator * f._denominator, a._denominator * f._numerator
        return None

    def __lt__(a, b):
        c = a._mec_fr_cmp(b)
        return NotImplemented if c is None else c[0] < c[1]

    def __gt__(a, b):
        c = a._mec_fr_cmp(b)
        return NotImplemented if c is None else c[0] > c[1]

    def __le__(a, b):
        c = a._mec_fr_cmp(b)
        return NotImplemented if c is None else c[0] <= c[1]

    def __ge__(a, b):
        c = a._mec_fr_cmp(b)
        return NotImplemented if c is None else c[0] >= c[1]

    def __bool__(a):
        return a._numerator != 0

    def __copy__(self):
        return self

    def __deepcopy__(self, memo):
        return self


_mec_fr_Fraction.__name__ = "Fraction"
_mec_fr_Fraction.__qualname__ = "Fraction"
_mec_fr_Fraction.__module__ = "fractions"


class _mec_fr_module:
    __all__ = ["Fraction"]
    Fraction = _mec_fr_Fraction

    def __repr__(self):
        return "<module 'fractions'>"


_mec_mod_fractions = type("module", (_mec_fr_module,), {})()
_mec_mod_fractions.__name__ = "fractions"
