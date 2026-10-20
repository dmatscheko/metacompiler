# python-decimal.py -- the decimal module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_decimal); helpers
# carry the _mec_dc prefix. The arithmetic is the General Decimal Arithmetic
# specification's, as Lib/_pydecimal.py implements it: a sign, a coefficient
# kept as a digit string and an exponent ('F' infinity, 'n' NaN, 'N' sNaN),
# each result rounded once into the context's precision. A trapped signal
# raises with the C module's text, the list of the signals. power() takes an
# integral exponent only; exp, ln, log10 and the logical operations are left
# out. An error naming a type says 'Decimal' where CPython says
# 'decimal.Decimal'.

import math as _mec_dc_math

# Decimal's own max and min methods would shadow the builtins inside its
# methods here (a class body's names are visible to the functions in it).
_mec_dc_max = max
_mec_dc_min = min

_mec_dc_ROUNDINGS = ["ROUND_DOWN", "ROUND_HALF_UP", "ROUND_HALF_EVEN", "ROUND_CEILING", "ROUND_FLOOR",
                     "ROUND_UP", "ROUND_HALF_DOWN", "ROUND_05UP"]


class _mec_dc_DecimalException(ArithmeticError):
    def handle(self, context, *args):
        pass


class _mec_dc_Clamped(_mec_dc_DecimalException):
    pass


class _mec_dc_InvalidOperation(_mec_dc_DecimalException):
    # The result is NaN, carrying the payload of an sNaN operand.
    def handle(self, context, *args):
        for a in args:
            if isinstance(a, _mec_dc_Decimal):
                return _mec_dc_make(a._sign, a._int, "n", True)._fix_nan(context)
        return _mec_dc_NaN


class _mec_dc_ConversionSyntax(_mec_dc_InvalidOperation):
    pass


class _mec_dc_DivisionByZero(_mec_dc_DecimalException, ZeroDivisionError):
    def handle(self, context, sign, *args):
        return _mec_dc_INF[sign]


class _mec_dc_DivisionImpossible(_mec_dc_InvalidOperation):
    pass


class _mec_dc_DivisionUndefined(_mec_dc_InvalidOperation, ZeroDivisionError):
    pass


class _mec_dc_Inexact(_mec_dc_DecimalException):
    pass


class _mec_dc_InvalidContext(_mec_dc_InvalidOperation):
    pass


class _mec_dc_Rounded(_mec_dc_DecimalException):
    pass


class _mec_dc_Subnormal(_mec_dc_DecimalException):
    pass


class _mec_dc_Overflow(_mec_dc_Inexact, _mec_dc_Rounded):
    def handle(self, context, sign, *args):
        r = context.rounding
        if r in ("ROUND_HALF_UP", "ROUND_HALF_EVEN", "ROUND_HALF_DOWN", "ROUND_UP"):
            return _mec_dc_INF[sign]
        if (sign == 0 and r == "ROUND_CEILING") or (sign == 1 and r == "ROUND_FLOOR"):
            return _mec_dc_INF[sign]
        return _mec_dc_make(sign, "9" * context.prec, context.Emax - context.prec + 1)


class _mec_dc_Underflow(_mec_dc_Inexact, _mec_dc_Rounded, _mec_dc_Subnormal):
    pass


class _mec_dc_FloatOperation(_mec_dc_DecimalException, TypeError):
    pass


# The signals a context keeps a flag and a trap for, in the order the C
# module's Context repr lists them; the conditions map onto InvalidOperation.
_mec_dc_SIGNALS = [_mec_dc_InvalidOperation, _mec_dc_Clamped, _mec_dc_DivisionByZero, _mec_dc_Inexact,
                   _mec_dc_FloatOperation, _mec_dc_Overflow, _mec_dc_Rounded, _mec_dc_Subnormal, _mec_dc_Underflow]
_mec_dc_CONDITIONS = [_mec_dc_ConversionSyntax, _mec_dc_DivisionImpossible, _mec_dc_DivisionUndefined,
                      _mec_dc_InvalidContext]
# libmpdec's status bits, whose order is the order a Context repr lists.
_mec_dc_BITS = {_mec_dc_Clamped: 1, _mec_dc_ConversionSyntax: 2, _mec_dc_DivisionByZero: 4,
                _mec_dc_DivisionImpossible: 8, _mec_dc_DivisionUndefined: 16, _mec_dc_Inexact: 64,
                _mec_dc_InvalidContext: 128, _mec_dc_InvalidOperation: 256, _mec_dc_FloatOperation: 1024,
                _mec_dc_Overflow: 2048, _mec_dc_Rounded: 4096, _mec_dc_Subnormal: 8192, _mec_dc_Underflow: 16384}


class _mec_dc_Context:
    def __init__(self, prec=None, rounding=None, Emin=None, Emax=None, capitals=None, clamp=None, flags=None,
                 traps=None):
        dflt = _mec_dc_DEFAULT[0] if _mec_dc_DEFAULT else None
        self.prec = prec if prec is not None else (dflt.prec if dflt else 28)
        self.rounding = rounding if rounding is not None else (dflt.rounding if dflt else "ROUND_HALF_EVEN")
        self.Emin = Emin if Emin is not None else (dflt.Emin if dflt else -999999)
        self.Emax = Emax if Emax is not None else (dflt.Emax if dflt else 999999)
        self.capitals = capitals if capitals is not None else (dflt.capitals if dflt else 1)
        self.clamp = clamp if clamp is not None else (dflt.clamp if dflt else 0)
        if self.rounding not in _mec_dc_ROUNDINGS:
            raise TypeError("valid values for rounding are:\n  [ROUND_CEILING, ROUND_FLOOR, ROUND_UP, ROUND_DOWN,\n"
                            "   ROUND_HALF_UP, ROUND_HALF_DOWN, ROUND_HALF_EVEN,\n   ROUND_05UP]")
        self.flags = {s: False for s in _mec_dc_SIGNALS}
        self._mec_bits = {}
        if flags is not None:
            for s in flags:
                self.flags[s] = bool(flags[s]) if isinstance(flags, dict) else True
        if traps is None:
            on = dflt.traps if dflt else {_mec_dc_InvalidOperation: True, _mec_dc_DivisionByZero: True,
                                          _mec_dc_Overflow: True}
            self.traps = {s: bool(on.get(s, False)) for s in _mec_dc_SIGNALS}
        else:
            self.traps = {s: False for s in _mec_dc_SIGNALS}
            for s in traps:
                self.traps[s] = bool(traps[s]) if isinstance(traps, dict) else True

    def Etiny(self):
        return self.Emin - self.prec + 1

    def Etop(self):
        return self.Emax - self.prec + 1

    def copy(self):
        c = _mec_dc_Context(self.prec, self.rounding, self.Emin, self.Emax, self.capitals, self.clamp,
                            dict(self.flags), dict(self.traps))
        c._mec_bits = dict(self._mec_bits)
        return c

    __copy__ = copy

    def clear_flags(self):
        for s in self.flags:
            self.flags[s] = False
        self._mec_bits = {}

    def clear_traps(self):
        for s in self.traps:
            self.traps[s] = False

    # Signal cond (a signal or a condition class): set its flag, raise it when
    # trapped, else answer its default result.
    def _raise(self, cond, *args):
        sig = self._mec_flag(cond)
        if not self.traps[sig]:
            return cond().handle(self, *args)
        raise sig("[" + repr(cond) + "]")

    # Set the flag of cond's signal, noting the condition for the repr's order.
    def _mec_flag(self, cond):
        sig = _mec_dc_InvalidOperation if cond in _mec_dc_CONDITIONS else cond
        self.flags[sig] = True
        bit = _mec_dc_BITS[cond]
        if bit < self._mec_bits.get(sig, 1 << 20):
            self._mec_bits[sig] = bit
        return sig

    def create_decimal(self, num="0"):
        d = _mec_dc_Decimal(num, self)
        if d._special_nan() and len(d._int) > self.prec - self.clamp:
            return self._raise(_mec_dc_ConversionSyntax)
        return d._fix(self)

    def create_decimal_from_float(self, f):
        return _mec_dc_Decimal.from_float(f)._fix(self)

    # The signals set in d, ordered as libmpdec prints them: by the lowest
    # condition bit seen (bits), else the signal's own lowest bit.
    def _sigs(self, d, bits):
        on = [s for s in _mec_dc_SIGNALS if d[s]]
        on.sort(key=lambda s: bits.get(s, 2 if s is _mec_dc_InvalidOperation else _mec_dc_BITS[s]))
        return "[" + ", ".join(s.__name__ for s in on) + "]"

    def __repr__(self):
        return ("Context(prec=" + str(self.prec) + ", rounding=" + self.rounding + ", Emin=" + str(self.Emin)
                + ", Emax=" + str(self.Emax) + ", capitals=" + str(self.capitals) + ", clamp=" + str(self.clamp)
                + ", flags=" + self._sigs(self.flags, self._mec_bits) + ", traps=" + self._sigs(self.traps, {})
                + ")")

    def add(self, a, b):
        return _mec_dc_conv(a, True).__add__(b, self)

    def subtract(self, a, b):
        return _mec_dc_conv(a, True).__sub__(b, self)

    def multiply(self, a, b):
        return _mec_dc_conv(a, True).__mul__(b, self)

    def divide(self, a, b):
        return _mec_dc_conv(a, True).__truediv__(b, self)

    def divide_int(self, a, b):
        return _mec_dc_conv(a, True).__floordiv__(b, self)

    def remainder(self, a, b):
        return _mec_dc_conv(a, True).__mod__(b, self)

    def power(self, a, b):
        return _mec_dc_conv(a, True).__pow__(b, None, self)

    def sqrt(self, a):
        return _mec_dc_conv(a, True).sqrt(self)

    def quantize(self, a, b):
        return _mec_dc_conv(a, True).quantize(b, context=self)

    def plus(self, a):
        return _mec_dc_conv(a, True).__pos__(self)

    def minus(self, a):
        return _mec_dc_conv(a, True).__neg__(self)

    def abs(self, a):
        return _mec_dc_conv(a, True).__abs__(self)

    def compare(self, a, b):
        return _mec_dc_conv(a, True).compare(b, self)

    def to_sci_string(self, a):
        return _mec_dc_conv(a, True).__str__(context=self)


_mec_dc_DEFAULT = []
_mec_dc_CUR = []


def _mec_dc_getcontext():
    if not _mec_dc_CUR:
        _mec_dc_CUR.append(_mec_dc_DEFAULT[0].copy())
    return _mec_dc_CUR[0]


def _mec_dc_setcontext(ctx):
    if not isinstance(ctx, _mec_dc_Context):
        raise TypeError("argument must be a context")
    if ctx is _mec_dc_DEFAULT[0] or ctx is _mec_dc_BASIC or ctx is _mec_dc_EXTENDED:
        ctx = ctx.copy()
        ctx.clear_flags()
    if _mec_dc_CUR:
        _mec_dc_CUR[0] = ctx
    else:
        _mec_dc_CUR.append(ctx)


class _mec_dc_LocalContext:
    def __init__(self, ctx):
        self.new = ctx.copy()

    def __enter__(self):
        self.saved = _mec_dc_getcontext()
        _mec_dc_setcontext(self.new)
        return self.new

    def __exit__(self, t, v, tb):
        _mec_dc_setcontext(self.saved)


def _mec_dc_localcontext(ctx=None, **kwargs):
    lc = _mec_dc_LocalContext(_mec_dc_getcontext() if ctx is None else ctx)
    for k in kwargs:
        if not hasattr(lc.new, k) or k.startswith("_"):
            raise TypeError("'" + k + "' is an invalid keyword argument for this function")
        v = kwargs[k]
        if k in ("flags", "traps"):
            # A list names the signals that are set.
            v = {sg: (bool(v[sg]) if isinstance(v, dict) else sg in v) for sg in _mec_dc_SIGNALS}
        setattr(lc.new, k, v)
    return lc


# ---------------------------------------------------------------------------

def _mec_dc_make(sign, coeff, exp, special=False):
    d = object.__new__(_mec_dc_Decimal)
    d._sign = sign
    d._int = coeff
    d._exp = exp
    d._is_special = special
    return d


# s parsed as a Decimal: (sign, digits, exp, special), None when it is no number.
def _mec_dc_parse(s):
    t = s.strip().replace("_", "")
    if t == "":
        return None
    sign = 0
    if t[0] in "+-":
        sign = 1 if t[0] == "-" else 0
        t = t[1:]
    low = t.lower()
    if low in ("inf", "infinity"):
        return sign, "0", "F", True
    for tag, kind in (("snan", "N"), ("nan", "n")):
        if low.startswith(tag):
            payload = t[len(tag):]
            if payload == "":
                return sign, "", kind, True
            if not (payload.isdigit() and payload.isascii()):
                return None
            return sign, str(int(payload)) if int(payload) else "", kind, True
    exp = 0
    cut = low.find("e")
    if cut >= 0:
        es = t[cut + 1:]
        body = es[1:] if es[:1] in ("+", "-") else es
        if not (body.isdigit() and body.isascii()):
            return None
        exp = int(es)
        t = t[:cut]
    dot = t.find(".")
    if dot >= 0:
        intpart, frac = t[:dot], t[dot + 1:]
    else:
        intpart, frac = t, ""
    if intpart == "" and frac == "":
        return None
    for part in (intpart, frac):
        if part and not (part.isdigit() and part.isascii()):
            return None
    digits = str(int(intpart + frac)) if intpart + frac else "0"
    return sign, digits, exp - len(frac), False


def _mec_dc_conv(v, raiseit=False):
    if isinstance(v, _mec_dc_Decimal):
        return v
    if isinstance(v, int):
        return _mec_dc_Decimal(v)
    if raiseit:
        raise TypeError("Unable to convert " + type(v).__name__ + " to Decimal")
    return NotImplemented


# For comparisons: an int or a float converts too, a float exactly (and it
# sets FloatOperation).
def _mec_dc_conv_cmp(v, ctx):
    if isinstance(v, float):
        ctx._mec_flag(_mec_dc_FloatOperation)
        return _mec_dc_Decimal.from_float(v)
    return _mec_dc_conv(v)


def _mec_dc_all_zeros(s, n):
    return s[n:].strip("0") == ""


def _mec_dc_exact_half(s, n):
    return s[n:n + 1] == "5" and _mec_dc_all_zeros(s, n + 1)


# Does keeping the first n digits of d's coefficient round away (1), stay exact
# (0) or round toward zero (-1) under mode r?
def _mec_dc_round_dir(d, n, r):
    s = d._int
    down = 0 if _mec_dc_all_zeros(s, n) else -1
    if r == "ROUND_DOWN":
        return down
    if r == "ROUND_UP":
        return -down
    if r == "ROUND_CEILING":
        return down if d._sign else -down
    if r == "ROUND_FLOOR":
        return -down if d._sign else down
    if r == "ROUND_05UP":
        if n and s[n - 1] not in "05":
            return down
        return -down
    if r == "ROUND_HALF_DOWN" and _mec_dc_exact_half(s, n):
        return -1
    if r == "ROUND_HALF_EVEN" and _mec_dc_exact_half(s, n) and (n == 0 or s[n - 1] in "02468"):
        return -1
    if s[n:n + 1] in ("5", "6", "7", "8", "9"):
        return 1
    return down


class _mec_dc_Decimal:
    def __init__(self, value="0", context=None):
        if isinstance(value, _mec_dc_Decimal):
            self._sign, self._int, self._exp, self._is_special = value._sign, value._int, value._exp, value._is_special
            return
        if isinstance(value, bool) or isinstance(value, int):
            n = int(value)
            self._sign = 1 if n < 0 else 0
            self._int = str(abs(n))
            self._exp = 0
            self._is_special = False
            return
        if isinstance(value, float):
            d = _mec_dc_Decimal.from_float(value)
            (context or _mec_dc_getcontext())._mec_flag(_mec_dc_FloatOperation)
            self._sign, self._int, self._exp, self._is_special = d._sign, d._int, d._exp, d._is_special
            return
        if isinstance(value, str):
            p = _mec_dc_parse(value)
            if p is None:
                d = (context or _mec_dc_getcontext())._raise(_mec_dc_ConversionSyntax)
                self._sign, self._int, self._exp, self._is_special = d._sign, d._int, d._exp, d._is_special
                return
            self._sign, self._int, self._exp, self._is_special = p
            return
        if isinstance(value, (list, tuple)):
            if len(value) != 3:
                raise ValueError("argument must be a sequence of length 3")
            sign, digits, exp = value
            if sign not in (0, 1):
                raise ValueError("sign must be an integer with the value 0 or 1")
            for x in digits:
                if not isinstance(x, int) or not 0 <= x <= 9:
                    raise ValueError("coefficient must be a tuple of digits")
            coeff = "".join(str(x) for x in digits).lstrip("0")
            self._sign = sign
            if exp == "F":
                self._int, self._exp, self._is_special = "0", "F", True
            elif exp in ("n", "N"):
                self._int, self._exp, self._is_special = coeff, exp, True
            else:
                if not isinstance(exp, int):
                    raise ValueError("exponent must be an integer")
                self._int, self._exp, self._is_special = coeff or "0", exp, False
            return
        raise TypeError("conversion from " + type(value).__name__ + " to Decimal is not supported")

    @classmethod
    def from_float(cls, f):
        if isinstance(f, int):
            return cls(f)
        if not isinstance(f, float):
            raise TypeError("argument must be int or float")
        sign = 1 if _mec_dc_math.copysign(1.0, f) < 0 else 0
        if f != f:
            return cls(_mec_dc_make(0, "", "n", True))
        if f in (_mec_dc_math.inf, -_mec_dc_math.inf):
            return cls(_mec_dc_make(sign, "0", "F", True))
        n, d = abs(f).as_integer_ratio()
        k = d.bit_length() - 1
        return cls(_mec_dc_make(sign, str(n * 5 ** k), -k))

    @classmethod
    def from_number(cls, n):
        if isinstance(n, (int, float, _mec_dc_Decimal)):
            return cls(n) if not isinstance(n, float) else cls.from_float(n)
        raise TypeError("conversion from " + type(n).__name__ + " to Decimal is not supported")

    # ----- classification

    def _isnan(self):
        if self._is_special:
            if self._exp == "n":
                return 1
            if self._exp == "N":
                return 2
        return 0

    def _special_nan(self):
        return self._is_special and self._exp in ("n", "N")

    def _isinf(self):
        if self._exp == "F":
            return -1 if self._sign else 1
        return 0

    def is_nan(self):
        return self._exp in ("n", "N")

    def is_snan(self):
        return self._exp == "N"

    def is_qnan(self):
        return self._exp == "n"

    def is_infinite(self):
        return self._exp == "F"

    def is_finite(self):
        return not self._is_special

    def is_zero(self):
        return not self._is_special and self._int == "0"

    def is_signed(self):
        return self._sign == 1

    def is_normal(self, context=None):
        ctx = context or _mec_dc_getcontext()
        return not self._is_special and self._int != "0" and ctx.Emin <= self.adjusted()

    def is_subnormal(self, context=None):
        ctx = context or _mec_dc_getcontext()
        return not self._is_special and self._int != "0" and self.adjusted() < ctx.Emin

    def is_canonical(self):
        return True

    def adjusted(self):
        if self._is_special:
            return 0
        return self._exp + len(self._int) - 1

    def __bool__(self):
        return self._is_special or self._int != "0"

    # ----- NaN propagation: the result for an operation with NaN operands, or None.

    def _check_nans(self, other, ctx):
        a = self._isnan()
        b = other._isnan() if other is not None else 0
        if a or b:
            if a == 2:
                return ctx._raise(_mec_dc_InvalidOperation, "sNaN", self)
            if b == 2:
                return ctx._raise(_mec_dc_InvalidOperation, "sNaN", other)
            if a:
                return self._fix_nan(ctx)
            return other._fix_nan(ctx)
        return None

    def _fix_nan(self, ctx):
        payload = self._int
        limit = ctx.prec - ctx.clamp
        if len(payload) > limit:
            payload = payload[len(payload) - limit:].lstrip("0")
        return _mec_dc_make(self._sign, payload, "n", True)

    # ----- rounding into a context

    def _fix(self, ctx):
        if self._is_special:
            if self._special_nan():
                return self._fix_nan(ctx)
            return self
        etiny = ctx.Etiny()
        etop = ctx.Etop()
        if self._int == "0":
            exp_max = etop if ctx.clamp else ctx.Emax
            new_exp = _mec_dc_min(_mec_dc_max(self._exp, etiny), exp_max)
            if new_exp != self._exp:
                ctx._raise(_mec_dc_Clamped)
                return _mec_dc_make(self._sign, "0", new_exp)
            return self
        exp_min = len(self._int) + self._exp - ctx.prec
        if exp_min > etop:
            ans = ctx._raise(_mec_dc_Overflow, self._sign)
            ctx._raise(_mec_dc_Inexact)
            ctx._raise(_mec_dc_Rounded)
            return ans
        subnormal = exp_min < etiny
        if subnormal:
            exp_min = etiny
        if self._exp < exp_min:
            d = self
            digits = len(d._int) + d._exp - exp_min
            if digits < 0:
                d = _mec_dc_make(d._sign, "1", exp_min - 1)
                digits = 0
            changed = _mec_dc_round_dir(d, digits, ctx.rounding)
            coeff = d._int[:digits] or "0"
            if changed > 0:
                coeff = str(int(coeff) + 1)
                if len(coeff) > ctx.prec:
                    coeff = coeff[:-1]
                    exp_min += 1
            if exp_min > etop:
                ans = ctx._raise(_mec_dc_Overflow, self._sign)
            else:
                ans = _mec_dc_make(self._sign, coeff, exp_min)
            if changed and subnormal:
                ctx._raise(_mec_dc_Underflow)
            if subnormal:
                ctx._raise(_mec_dc_Subnormal)
            if changed:
                ctx._raise(_mec_dc_Inexact)
            ctx._raise(_mec_dc_Rounded)
            if not ans:
                ctx._raise(_mec_dc_Clamped)
            return ans
        if subnormal:
            ctx._raise(_mec_dc_Subnormal)
        if ctx.clamp == 1 and self._exp > etop:
            ctx._raise(_mec_dc_Clamped)
            return _mec_dc_make(self._sign, self._int + "0" * (self._exp - etop), etop)
        return self

    # self with exponent exp, rounded by mode r and not into any context.
    def _rescale(self, exp, r):
        if self._is_special:
            return self
        if self._int == "0":
            return _mec_dc_make(self._sign, "0", exp)
        if self._exp >= exp:
            return _mec_dc_make(self._sign, self._int + "0" * (self._exp - exp), exp)
        d = self
        digits = len(d._int) + d._exp - exp
        if digits < 0:
            d = _mec_dc_make(d._sign, "1", exp - 1)
            digits = 0
        changed = _mec_dc_round_dir(d, digits, r)
        coeff = d._int[:digits] or "0"
        if changed == 1:
            coeff = str(int(coeff) + 1)
        return _mec_dc_make(self._sign, coeff, exp)

    # self rounded to `places` significant digits.
    def _round(self, places, r):
        if self._is_special or self._int == "0":
            return self
        ans = self._rescale(self.adjusted() + 1 - places, r)
        if ans.adjusted() != self.adjusted():
            ans = ans._rescale(ans.adjusted() + 1 - places, r)
        return ans

    # ----- text

    def __str__(self, eng=False, context=None):
        sign = "-" if self._sign else ""
        if self._is_special:
            if self._exp == "F":
                return sign + "Infinity"
            return sign + ("NaN" if self._exp == "n" else "sNaN") + self._int
        left = self._exp + len(self._int)
        if self._exp <= 0 and left > -6:
            dot = left
        elif not eng:
            dot = 1
        elif self._int == "0":
            dot = (left + 1) % 3 - 1
        else:
            dot = (left - 1) % 3 + 1
        if dot <= 0:
            intpart = "0"
            frac = "." + "0" * -dot + self._int
        elif dot >= len(self._int):
            intpart = self._int + "0" * (dot - len(self._int))
            frac = ""
        else:
            intpart = self._int[:dot]
            frac = "." + self._int[dot:]
        if left == dot:
            exp = ""
        else:
            ctx = context or _mec_dc_getcontext()
            e = left - dot
            exp = ("E" if ctx.capitals else "e") + ("+" if e >= 0 else "-") + str(abs(e))
        return sign + intpart + frac + exp

    def to_eng_string(self, context=None):
        return self.__str__(True, context)

    def __repr__(self):
        return "Decimal('" + str(self) + "')"

    def __format__(self, spec):
        return _mec_dc_format(self, spec, _mec_dc_getcontext())

    # ----- arithmetic

    def __add__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        if self._is_special or other._is_special:
            ans = self._check_nans(other, ctx)
            if ans is not None:
                return ans
            if self._isinf():
                if self._sign != other._sign and other._isinf():
                    return ctx._raise(_mec_dc_InvalidOperation, "-INF + INF")
                return self
            if other._isinf():
                return other
        exp = _mec_dc_min(self._exp, other._exp)
        negzero = 1 if ctx.rounding == "ROUND_FLOOR" and self._sign != other._sign else 0
        if self._int == "0" and other._int == "0":
            sign = _mec_dc_min(self._sign, other._sign)
            if negzero:
                sign = 1
            return _mec_dc_make(sign, "0", exp)._fix(ctx)
        if self._int == "0":
            exp = _mec_dc_max(exp, other._exp - ctx.prec - 1)
            return other._rescale(exp, ctx.rounding)._fix(ctx)
        if other._int == "0":
            exp = _mec_dc_max(exp, self._exp - ctx.prec - 1)
            return self._rescale(exp, ctx.rounding)._fix(ctx)
        s1, i1, e1 = self._sign, int(self._int), self._exp
        s2, i2, e2 = other._sign, int(other._int), other._exp
        # Align the exponents; an operand far below the other's precision
        # only matters for rounding, so it is shrunk to one digit there.
        if e1 < e2:
            s1, i1, e1, s2, i2, e2 = s2, i2, e2, s1, i1, e1
        lead = e1 + _mec_dc_min(-1, len(str(i1)) - ctx.prec - 2)
        if len(str(i2)) + e2 - 1 < lead:
            i2 = 1
            e2 = lead
        i1 *= 10 ** (e1 - e2)
        e1 = e2
        if s1 != s2:
            if i1 == i2:
                return _mec_dc_make(negzero, "0", exp)._fix(ctx)
            if i1 < i2:
                s1, i1, s2, i2 = s2, i2, s1, i1
            return _mec_dc_make(s1, str(i1 - i2), e1)._fix(ctx)
        return _mec_dc_make(s1, str(i1 + i2), e1)._fix(ctx)

    def __radd__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__add__(self, context)

    def __sub__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        if self._is_special or other._is_special:
            ans = self._check_nans(other, ctx)
            if ans is not None:
                return ans
        return self.__add__(other.copy_negate(), ctx)

    def __rsub__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__sub__(self, context)

    def __mul__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        sign = self._sign ^ other._sign
        if self._is_special or other._is_special:
            ans = self._check_nans(other, ctx)
            if ans is not None:
                return ans
            if self._isinf():
                if other._int == "0" and not other._is_special:
                    return ctx._raise(_mec_dc_InvalidOperation, "(+-)INF * 0")
                return _mec_dc_INF[sign]
            if other._isinf():
                if self._int == "0" and not self._is_special:
                    return ctx._raise(_mec_dc_InvalidOperation, "0 * (+-)INF")
                return _mec_dc_INF[sign]
        exp = self._exp + other._exp
        if self._int == "0" or other._int == "0":
            return _mec_dc_make(sign, "0", exp)._fix(ctx)
        return _mec_dc_make(sign, str(int(self._int) * int(other._int)), exp)._fix(ctx)

    def __rmul__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__mul__(self, context)

    def __truediv__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        sign = self._sign ^ other._sign
        if self._is_special or other._is_special:
            ans = self._check_nans(other, ctx)
            if ans is not None:
                return ans
            if self._isinf() and other._isinf():
                return ctx._raise(_mec_dc_InvalidOperation, "(+-)INF/(+-)INF")
            if self._isinf():
                return _mec_dc_INF[sign]
            if other._isinf():
                ctx._raise(_mec_dc_Clamped)
                return _mec_dc_make(sign, "0", ctx.Etiny())
        if other._int == "0":
            if self._int == "0":
                return ctx._raise(_mec_dc_DivisionUndefined)
            return ctx._raise(_mec_dc_DivisionByZero, sign)
        if self._int == "0":
            exp = self._exp - other._exp
            coeff = 0
        else:
            shift = len(other._int) - len(self._int) + ctx.prec + 1
            exp = self._exp - other._exp - shift
            a, b = int(self._int), int(other._int)
            if shift >= 0:
                coeff, rem = divmod(a * 10 ** shift, b)
            else:
                coeff, rem = divmod(a, b * 10 ** -shift)
            if rem:
                if coeff % 5 == 0:
                    coeff += 1
            else:
                ideal = self._exp - other._exp
                while exp < ideal and coeff % 10 == 0:
                    coeff //= 10
                    exp += 1
        return _mec_dc_make(sign, str(coeff), exp)._fix(ctx)

    def __rtruediv__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__truediv__(self, context)

    # (quotient, remainder) of the integer division, both finite and nonzero
    # divisor; the quotient too large for the precision is DivisionImpossible.
    def _divide(self, other, ctx):
        sign = self._sign ^ other._sign
        ideal = self._exp if other._isinf() else _mec_dc_min(self._exp, other._exp)
        expdiff = self.adjusted() - other.adjusted()
        if self._int == "0" or other._isinf() or expdiff <= -2:
            return _mec_dc_make(sign, "0", 0), self._rescale(ideal, ctx.rounding)
        if expdiff <= ctx.prec:
            a, b = int(self._int), int(other._int)
            if self._exp >= other._exp:
                a *= 10 ** (self._exp - other._exp)
            else:
                b *= 10 ** (other._exp - self._exp)
            q, r = divmod(a, b)
            if q < 10 ** ctx.prec:
                return _mec_dc_make(sign, str(q), 0), _mec_dc_make(self._sign, str(r), ideal)
        ans = ctx._raise(_mec_dc_DivisionImpossible)
        return ans, ans

    def __floordiv__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        ans = self._check_nans(other, ctx)
        if ans is not None:
            return ans
        if self._isinf():
            if other._isinf():
                return ctx._raise(_mec_dc_InvalidOperation, "INF // INF")
            return _mec_dc_INF[self._sign ^ other._sign]
        if not other:
            if self:
                return ctx._raise(_mec_dc_DivisionByZero, self._sign ^ other._sign)
            return ctx._raise(_mec_dc_DivisionUndefined)
        return self._divide(other, ctx)[0]

    def __rfloordiv__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__floordiv__(self, context)

    def __mod__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        ans = self._check_nans(other, ctx)
        if ans is not None:
            return ans
        if self._isinf():
            return ctx._raise(_mec_dc_InvalidOperation, "INF % x")
        if not other:
            if self:
                return ctx._raise(_mec_dc_InvalidOperation, "x % 0")
            return ctx._raise(_mec_dc_DivisionUndefined)
        return self._divide(other, ctx)[1]._fix(ctx)

    def __rmod__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__mod__(self, context)

    def __divmod__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        ans = self._check_nans(other, ctx)
        if ans is not None:
            return ans, ans
        sign = self._sign ^ other._sign
        if self._isinf():
            if other._isinf():
                ans = ctx._raise(_mec_dc_InvalidOperation, "divmod(INF, INF)")
                return ans, ans
            return _mec_dc_INF[sign], ctx._raise(_mec_dc_InvalidOperation, "INF % x")
        if not other:
            if not self:
                ans = ctx._raise(_mec_dc_DivisionUndefined)
                return ans, ans
            # Both signals at once, as libmpdec raises them: the trapped ones
            # are listed and the first of them raised.
            ctx._mec_flag(_mec_dc_InvalidOperation)
            ctx._mec_flag(_mec_dc_DivisionByZero)
            trapped = [c for c in (_mec_dc_InvalidOperation, _mec_dc_DivisionByZero) if ctx.traps[c]]
            if trapped:
                raise trapped[0]("[" + ", ".join(repr(c) for c in trapped) + "]")
            return _mec_dc_INF[sign], _mec_dc_NaN
        q, r = self._divide(other, ctx)
        return q, r._fix(ctx)

    def __rdivmod__(self, other, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__divmod__(self, context)

    def remainder_near(self, other, context=None):
        other = _mec_dc_conv(other, True)
        ctx = context or _mec_dc_getcontext()
        ans = self._check_nans(other, ctx)
        if ans is not None:
            return ans
        if self._isinf():
            return ctx._raise(_mec_dc_InvalidOperation, "remainder_near(infinity, x)")
        if not other:
            if self:
                return ctx._raise(_mec_dc_InvalidOperation, "remainder_near(x, 0)")
            return ctx._raise(_mec_dc_DivisionUndefined)
        if other._isinf():
            return self._fix(ctx)
        ideal = _mec_dc_min(self._exp, other._exp)
        if not self:
            return _mec_dc_make(self._sign, "0", ideal)._fix(ctx)
        expdiff = self.adjusted() - other.adjusted()
        if expdiff >= ctx.prec + 1:
            return ctx._raise(_mec_dc_DivisionImpossible)
        if expdiff <= -2:
            return self._rescale(ideal, ctx.rounding)._fix(ctx)
        a, b = int(self._int), int(other._int)
        if self._exp >= other._exp:
            a *= 10 ** (self._exp - other._exp)
        else:
            b *= 10 ** (other._exp - self._exp)
        q, r = divmod(a, b)
        if 2 * r + (q & 1) > b:
            r -= b
            q += 1
        if q >= 10 ** ctx.prec:
            return ctx._raise(_mec_dc_DivisionImpossible)
        sign = self._sign
        if r < 0:
            sign = 1 - sign
            r = -r
        return _mec_dc_make(sign, str(r), ideal)._fix(ctx)

    def __pow__(self, other, modulo=None, context=None):
        if modulo is not None:
            raise NotImplementedError("three-argument pow() of a Decimal is not supported")
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        ctx = context or _mec_dc_getcontext()
        ans = self._check_nans(other, ctx)
        if ans is not None:
            return ans
        if other._is_special or (other._exp < 0 and other._int[other._exp:].strip("0") != ""):
            raise NotImplementedError("Decimal power with a non-integral exponent is not supported")
        n = int(other)
        if self._is_special:
            if n == 0:
                return _mec_dc_make(0, "1", 0)
            if n < 0:
                return _mec_dc_make(self._sign if n % 2 else 0, "0", 0)
            return _mec_dc_INF[self._sign if n % 2 else 0]
        if not self:
            if n == 0:
                return ctx._raise(_mec_dc_InvalidOperation, "0 ** 0")
            sign = self._sign if n % 2 else 0
            if n < 0:
                return _mec_dc_INF[sign]
            return _mec_dc_make(sign, "0", 0)
        sign = self._sign if n % 2 else 0
        if n >= 0:
            return _mec_dc_make(sign, str(int(self._int) ** n), self._exp * n)._fix(ctx)
        base = _mec_dc_make(sign, str(int(self._int) ** -n), self._exp * -n)
        return _mec_dc_make(0, "1", 0).__truediv__(base, ctx)

    def __rpow__(self, other, modulo=None, context=None):
        other = _mec_dc_conv(other)
        if other is NotImplemented:
            return other
        return other.__pow__(self, modulo, context)

    def __neg__(self, context=None):
        ctx = context or _mec_dc_getcontext()
        if self._is_special:
            ans = self._check_nans(None, ctx)
            if ans is not None:
                return ans
        if self._int == "0" and not self._is_special and ctx.rounding != "ROUND_FLOOR":
            return _mec_dc_make(0, "0", self._exp)._fix(ctx)
        return self.copy_negate()._fix(ctx)

    def __pos__(self, context=None):
        ctx = context or _mec_dc_getcontext()
        if self._is_special:
            ans = self._check_nans(None, ctx)
            if ans is not None:
                return ans
        if self._int == "0" and not self._is_special and ctx.rounding != "ROUND_FLOOR":
            return _mec_dc_make(0, "0", self._exp)._fix(ctx)
        return self._fix(ctx)

    def __abs__(self, round=True, context=None):
        ctx = context or _mec_dc_getcontext()
        if self._is_special:
            ans = self._check_nans(None, ctx)
            if ans is not None:
                return ans
        return self.copy_abs()._fix(ctx)

    def copy_abs(self):
        return _mec_dc_make(0, self._int, self._exp, self._is_special)

    def copy_negate(self):
        return _mec_dc_make(1 - self._sign, self._int, self._exp, self._is_special)

    def copy_sign(self, other):
        other = _mec_dc_conv(other, True)
        return _mec_dc_make(other._sign, self._int, self._exp, self._is_special)

    def sqrt(self, context=None):
        ctx = context or _mec_dc_getcontext()
        if self._is_special:
            ans = self._check_nans(None, ctx)
            if ans is not None:
                return ans
            if self._isinf() and self._sign == 0:
                return self
        if self._int == "0":
            return _mec_dc_make(self._sign, "0", self._exp // 2)._fix(ctx)
        if self._sign == 1:
            return ctx._raise(_mec_dc_InvalidOperation, "sqrt(-x), x > 0")
        prec = ctx.prec + 1
        e = self._exp >> 1
        if self._exp & 1:
            c = int(self._int) * 10
            l = (len(self._int) >> 1) + 1
        else:
            c = int(self._int)
            l = (len(self._int) + 1) >> 1
        shift = prec - l
        if shift >= 0:
            c *= 100 ** shift
            exact = True
        else:
            c, rem = divmod(c, 100 ** -shift)
            exact = not rem
        e -= shift
        n = 10 ** prec
        while True:
            q = c // n
            if n <= q:
                break
            n = (n + q) >> 1
        exact = exact and n * n == c
        if exact:
            if shift >= 0:
                n //= 10 ** shift
            else:
                n *= 10 ** -shift
            e += shift
        elif n % 5 == 0:
            n += 1
        c2 = ctx.copy()
        c2.rounding = "ROUND_HALF_EVEN"
        c2.flags = ctx.flags
        return _mec_dc_make(0, str(n), e)._fix(c2)

    def quantize(self, exp, rounding=None, context=None):
        exp = _mec_dc_conv(exp, True)
        ctx = context or _mec_dc_getcontext()
        r = rounding or ctx.rounding
        if self._is_special or exp._is_special:
            ans = self._check_nans(exp, ctx)
            if ans is not None:
                return ans
            if exp._isinf() or self._isinf():
                if exp._isinf() and self._isinf():
                    return self
                return ctx._raise(_mec_dc_InvalidOperation, "quantize with one INF")
        if not ctx.Etiny() <= exp._exp <= ctx.Emax:
            return ctx._raise(_mec_dc_InvalidOperation, "target exponent out of bounds in quantize")
        if self._int == "0":
            return _mec_dc_make(self._sign, "0", exp._exp)._fix(ctx)
        if self.adjusted() > ctx.Emax:
            return ctx._raise(_mec_dc_InvalidOperation, "exponent of quantize result too large")
        if self.adjusted() - exp._exp + 1 > ctx.prec:
            return ctx._raise(_mec_dc_InvalidOperation, "quantize result has too many digits")
        ans = self._rescale(exp._exp, r)
        if ans.adjusted() > ctx.Emax or len(ans._int) > ctx.prec:
            return ctx._raise(_mec_dc_InvalidOperation, "quantize result has too many digits")
        if ans and ans.adjusted() < ctx.Emin:
            ctx._raise(_mec_dc_Subnormal)
        if ans._exp > self._exp:
            if ans != self:
                ctx._raise(_mec_dc_Inexact)
            ctx._raise(_mec_dc_Rounded)
        return ans._fix(ctx)

    def same_quantum(self, other, context=None):
        other = _mec_dc_conv(other, True)
        if self._is_special or other._is_special:
            return (self.is_nan() and other.is_nan()) or (self.is_infinite() and other.is_infinite())
        return self._exp == other._exp

    def to_integral_value(self, rounding=None, context=None):
        ctx = context or _mec_dc_getcontext()
        if self._is_special:
            ans = self._check_nans(None, ctx)
            if ans is not None:
                return ans
            return self
        if self._exp >= 0:
            return self
        return self._rescale(0, rounding or ctx.rounding)

    to_integral = to_integral_value

    def to_integral_exact(self, rounding=None, context=None):
        ctx = context or _mec_dc_getcontext()
        if self._is_special:
            ans = self._check_nans(None, ctx)
            if ans is not None:
                return ans
            return self
        if self._exp >= 0:
            return self
        if self._int == "0":
            return _mec_dc_make(self._sign, "0", 0)
        ans = self._rescale(0, rounding or ctx.rounding)
        if ans != self:
            ctx._raise(_mec_dc_Inexact)
        ctx._raise(_mec_dc_Rounded)
        return ans

    def normalize(self, context=None):
        ctx = context or _mec_dc_getcontext()
        if self._is_special:
            ans = self._check_nans(None, ctx)
            if ans is not None:
                return ans
        dup = self._fix(ctx)
        if dup._is_special:
            return dup
        if dup._int == "0":
            return _mec_dc_make(dup._sign, "0", 0)
        exp_max = ctx.Etop() if ctx.clamp else ctx.Emax
        end = len(dup._int)
        exp = dup._exp
        while dup._int[end - 1] == "0" and exp < exp_max:
            exp += 1
            end -= 1
        return _mec_dc_make(dup._sign, dup._int[:end], exp)

    def as_tuple(self):
        digits = tuple(int(c) for c in self._int) if self._int else ()
        return _mec_dc_DecimalTuple((self._sign, digits, self._exp))

    def as_integer_ratio(self):
        if self._is_special:
            if self.is_nan():
                raise ValueError("cannot convert NaN to integer ratio")
            raise OverflowError("cannot convert Infinity to integer ratio")
        if self._int == "0":
            return 0, 1
        n = int(self._int)
        if self._exp >= 0:
            n *= 10 ** self._exp
            d = 1
        else:
            d = 10 ** -self._exp
            g = _mec_dc_math.gcd(n, d)
            n //= g
            d //= g
        return (-n if self._sign else n), d

    def __int__(self):
        if self._is_special:
            if self.is_nan():
                raise ValueError("cannot convert NaN to integer")
            raise OverflowError("cannot convert Infinity to integer")
        s = -1 if self._sign else 1
        if self._exp >= 0:
            return s * int(self._int) * 10 ** self._exp
        return s * int(self._int[:self._exp] or "0")

    __trunc__ = __int__

    def __float__(self):
        if self.is_nan():
            if self.is_snan():
                raise ValueError("cannot convert signaling NaN to float")
            return float("-nan" if self._sign else "nan")
        return float(str(self))

    def __floor__(self):
        if self._is_special:
            return self.__int__()
        return int(self._rescale(0, "ROUND_FLOOR"))

    def __ceil__(self):
        if self._is_special:
            return self.__int__()
        return int(self._rescale(0, "ROUND_CEILING"))

    def __round__(self, n=None):
        if n is not None:
            if not isinstance(n, int):
                raise TypeError("Second argument to round should be integral")
            return self.quantize(_mec_dc_make(0, "1", -n))
        if self._is_special:
            if self.is_nan():
                raise ValueError("cannot convert NaN to integer")
            raise OverflowError("cannot convert Infinity to integer")
        return int(self._rescale(0, "ROUND_HALF_EVEN"))

    # ----- comparison

    # -1, 0 or 1 for two non-NaN values.
    def _cmp(self, other):
        if self._is_special or other._is_special:
            a, b = self._isinf(), other._isinf()
            if a == b:
                return 0
            return -1 if a < b else 1
        if self._int == "0":
            if other._int == "0":
                return 0
            return 1 if other._sign else -1
        if other._int == "0":
            return -1 if self._sign else 1
        if other._sign < self._sign:
            return -1
        if self._sign < other._sign:
            return 1
        sa, oa = self.adjusted(), other.adjusted()
        s = -1 if self._sign else 1
        if sa == oa:
            sp = self._int + "0" * (self._exp - other._exp)
            op = other._int + "0" * (other._exp - self._exp)
            if sp == op:
                return 0
            return -s if sp < op else s
        return s if sa > oa else -s

    def _cmp_op(self, other, op):
        ctx = _mec_dc_getcontext()
        other = _mec_dc_conv_cmp(other, ctx)
        if other is NotImplemented:
            return other
        if self.is_nan() or other.is_nan():
            if op == "==":
                return False
            if op == "!=":
                return True
            ctx._raise(_mec_dc_InvalidOperation, "comparison involving NaN")
            return False
        c = self._cmp(other)
        if op == "==":
            return c == 0
        if op == "!=":
            return c != 0
        if op == "<":
            return c < 0
        if op == "<=":
            return c <= 0
        if op == ">":
            return c > 0
        return c >= 0

    def __eq__(self, other):
        return self._cmp_op(other, "==")

    def __ne__(self, other):
        return self._cmp_op(other, "!=")

    def __lt__(self, other):
        return self._cmp_op(other, "<")

    def __le__(self, other):
        return self._cmp_op(other, "<=")

    def __gt__(self, other):
        return self._cmp_op(other, ">")

    def __ge__(self, other):
        return self._cmp_op(other, ">=")

    def compare(self, other, context=None):
        other = _mec_dc_conv(other, True)
        ctx = context or _mec_dc_getcontext()
        ans = self._check_nans(other, ctx)
        if ans is not None:
            return ans
        return _mec_dc_Decimal(self._cmp(other))

    def max(self, other, context=None):
        other = _mec_dc_conv(other, True)
        ctx = context or _mec_dc_getcontext()
        if self.is_nan() or other.is_nan():
            if self.is_qnan() and not other.is_nan():
                return other._fix(ctx)
            if other.is_qnan() and not self.is_nan():
                return self._fix(ctx)
            return self._check_nans(other, ctx)
        c = self._cmp(other)
        if c == 0:
            c = -1 if self._sign > other._sign or (self._sign == other._sign == 0 and self._exp < other._exp) \
                or (self._sign == other._sign == 1 and self._exp > other._exp) else 1
        return (other if c < 0 else self)._fix(ctx)

    def min(self, other, context=None):
        other = _mec_dc_conv(other, True)
        ctx = context or _mec_dc_getcontext()
        if self.is_nan() or other.is_nan():
            if self.is_qnan() and not other.is_nan():
                return other._fix(ctx)
            if other.is_qnan() and not self.is_nan():
                return self._fix(ctx)
            return self._check_nans(other, ctx)
        c = self._cmp(other)
        if c == 0:
            c = -1 if self._sign > other._sign or (self._sign == other._sign == 0 and self._exp < other._exp) \
                or (self._sign == other._sign == 1 and self._exp > other._exp) else 1
        return (self if c < 0 else other)._fix(ctx)

    def __hash__(self):
        if self._is_special:
            if self.is_snan():
                raise TypeError("Cannot hash a signaling NaN value.")
            if self.is_nan():
                return object.__hash__(self)
            return -314159 if self._sign else 314159
        mod = 2305843009213693951
        if self._exp >= 0:
            eh = pow(10, self._exp, mod)
        else:
            eh = pow(_mec_dc_TEN_INV, -self._exp, mod)
        h = int(self._int) * eh % mod
        ans = -h if self._sign else h
        return -2 if ans == -1 else ans

    def __copy__(self):
        return self

    def __deepcopy__(self, memo):
        return self

    def __reduce__(self):
        return (type(self), (str(self),))


_mec_dc_TEN_INV = pow(10, 2305843009213693951 - 2, 2305843009213693951)


class _mec_dc_DecimalTuple(tuple):
    @property
    def sign(self):
        return self[0]

    @property
    def digits(self):
        return self[1]

    @property
    def exponent(self):
        return self[2]

    def __repr__(self):
        return ("DecimalTuple(sign=" + str(self[0]) + ", digits=" + repr(self[1]) + ", exponent=" + repr(self[2])
                + ")")


# ----- format(), after the standard format-spec mini-language

def _mec_dc_spec(spec):
    out = {"fill": " ", "align": None, "sign": "-", "z": False, "alt": False, "zeropad": False, "width": 0,
           "sep": "", "prec": None, "type": None}
    i = 0
    n = len(spec)
    if n >= 2 and spec[1] in "<>=^":
        out["fill"] = spec[0]
        out["align"] = spec[1]
        i = 2
    elif n >= 1 and spec[0] in "<>=^":
        out["align"] = spec[0]
        i = 1
    if i < n and spec[i] in "+- ":
        out["sign"] = spec[i]
        i += 1
    if i < n and spec[i] == "z":
        out["z"] = True
        i += 1
    if i < n and spec[i] == "#":
        out["alt"] = True
        i += 1
    if i < n and spec[i] == "0":
        out["zeropad"] = True
        i += 1
    j = i
    while j < n and spec[j].isdigit():
        j += 1
    if j > i:
        out["width"] = int(spec[i:j])
    i = j
    if i < n and spec[i] in ",_":
        out["sep"] = spec[i]
        i += 1
    if i < n and spec[i] == ".":
        j = i + 1
        while j < n and spec[j].isdigit():
            j += 1
        if j == i + 1:
            raise ValueError("Invalid format specifier '" + spec + "' for object of type 'Decimal'")
        out["prec"] = int(spec[i + 1:j])
        i = j
    if i < n and spec[i] in "eEfFgGn%":
        out["type"] = spec[i]
        i += 1
    if i != n:
        raise ValueError("Invalid format specifier '" + spec + "' for object of type 'Decimal'")
    if out["zeropad"] and out["align"] is None:
        out["fill"] = "0"
        out["align"] = "="
    if out["align"] is None:
        out["align"] = ">"
    if out["prec"] == 0 and out["type"] in ("g", "G", "n"):
        out["prec"] = 1
    if out["type"] == "n":
        out["type"] = "g"
    return out


def _mec_dc_align(sign, body, sp):
    pad = sp["fill"] * (sp["width"] - len(sign) - len(body))
    a = sp["align"]
    if a == "<":
        return sign + body + pad
    if a == ">":
        return pad + sign + body
    if a == "=":
        return sign + pad + body
    half = len(pad) // 2
    return pad[:half] + sign + body + pad[half:]


def _mec_dc_group(digits, sep, min_width):
    if not sep:
        width = max(len(digits), min_width, 1)
        return "0" * (width - len(digits)) + digits
    groups = []
    while True:
        l = min(max(len(digits), min_width, 1), 3)
        groups.append("0" * (l - len(digits)) + digits[max(len(digits) - l, 0):])
        digits = digits[:max(len(digits) - l, 0)]
        min_width -= l
        if not digits and min_width <= 0:
            break
        min_width -= len(sep)
    return sep.join(reversed(groups))


def _mec_dc_format(d, spec, ctx):
    sp = _mec_dc_spec(spec)
    t = sp["type"]
    if d._is_special:
        # Zero padding does not apply to an infinity or a NaN.
        if sp["zeropad"] and sp["fill"] == "0" and sp["align"] == "=":
            sp["fill"] = " "
            sp["align"] = ">"
        sign = "-" if d._sign else (sp["sign"] if sp["sign"] in "+ " else "")
        body = str(d.copy_abs())
        if t == "%":
            body += "%"
        return _mec_dc_align(sign, body, sp)
    if t is None:
        t = "G" if ctx.capitals else "g"
    if t == "%":
        d = _mec_dc_make(d._sign, d._int, d._exp + 2)
    prec = sp["prec"]
    r = ctx.rounding
    if prec is not None:
        if t in "eE":
            d = d._round(prec + 1, r)
        elif t in "fF%":
            d = d._rescale(-prec, r)
        elif t in "gG" and len(d._int) > prec:
            d = d._round(prec, r)
    if d._int == "0" and d._exp > 0 and t in "fF%":
        d = d._rescale(0, r)
    neg = d._sign
    if d._int == "0" and sp["z"]:
        neg = 0
    left = d._exp + len(d._int)
    if t in "eE":
        dot = 1 - prec if d._int == "0" and prec is not None else 1
    elif t in "fF%":
        dot = left
    elif d._exp <= 0 and left > -6:
        dot = left
    else:
        dot = 1
    if dot < 0:
        intpart = "0"
        frac = "0" * -dot + d._int
    elif dot > len(d._int):
        intpart = d._int + "0" * (dot - len(d._int))
        frac = ""
    else:
        intpart = d._int[:dot] or "0"
        frac = d._int[dot:]
    exp = left - dot
    sign = "-" if neg else (sp["sign"] if sp["sign"] in "+ " else "")
    if frac or sp["alt"]:
        frac = "." + frac
    if exp != 0 or t in "eE":
        e = {"E": "E", "e": "e", "G": "E", "g": "e"}[t]
        frac += e + ("+" if exp >= 0 else "-") + str(abs(exp))
    if t == "%":
        frac += "%"
    min_width = sp["width"] - len(frac) - len(sign) if sp["zeropad"] else 0
    intpart = _mec_dc_group(intpart, sp["sep"], min_width)
    return _mec_dc_align(sign, intpart + frac, sp)


_mec_dc_NaN = _mec_dc_make(0, "", "n", True)
_mec_dc_INF = [_mec_dc_make(0, "0", "F", True), _mec_dc_make(1, "0", "F", True)]

for _mec_dc_cls in ([_mec_dc_Decimal, _mec_dc_Context, _mec_dc_DecimalException, _mec_dc_DecimalTuple]
                    + _mec_dc_SIGNALS + _mec_dc_CONDITIONS):
    _mec_dc_nm = _mec_dc_cls.__name__[8:] if _mec_dc_cls.__name__.startswith("_mec_dc_") else _mec_dc_cls.__name__
    _mec_dc_cls.__name__ = _mec_dc_nm
    _mec_dc_cls.__qualname__ = _mec_dc_nm
    _mec_dc_cls.__module__ = "decimal"

_mec_dc_DEFAULT.append(_mec_dc_Context(28, "ROUND_HALF_EVEN", -999999, 999999, 1, 0, None,
                                       [_mec_dc_InvalidOperation, _mec_dc_DivisionByZero, _mec_dc_Overflow]))
_mec_dc_BASIC = _mec_dc_Context(9, "ROUND_HALF_UP", -999999, 999999, 1, 0, None,
                                [_mec_dc_InvalidOperation, _mec_dc_DivisionByZero, _mec_dc_Overflow,
                                 _mec_dc_Underflow, _mec_dc_Clamped])
_mec_dc_EXTENDED = _mec_dc_Context(9, "ROUND_HALF_EVEN", -999999, 999999, 1, 0, None, [])


class _mec_dc_module:
    __all__ = ["Decimal", "Context", "DecimalTuple", "DefaultContext", "BasicContext", "ExtendedContext",
               "DecimalException", "Clamped", "InvalidOperation", "DivisionByZero", "Inexact", "Rounded",
               "Subnormal", "Overflow", "Underflow", "FloatOperation", "DivisionImpossible", "InvalidContext",
               "ConversionSyntax", "DivisionUndefined", "ROUND_DOWN", "ROUND_HALF_UP", "ROUND_HALF_EVEN",
               "ROUND_CEILING", "ROUND_FLOOR", "ROUND_UP", "ROUND_HALF_DOWN", "ROUND_05UP", "setcontext",
               "getcontext", "localcontext", "MAX_PREC", "MAX_EMAX", "MIN_EMIN", "MIN_ETINY", "HAVE_THREADS",
               "HAVE_CONTEXTVAR"]
    Decimal = _mec_dc_Decimal
    Context = _mec_dc_Context
    DecimalTuple = _mec_dc_DecimalTuple
    DecimalException = _mec_dc_DecimalException
    Clamped = _mec_dc_Clamped
    InvalidOperation = _mec_dc_InvalidOperation
    DivisionByZero = _mec_dc_DivisionByZero
    Inexact = _mec_dc_Inexact
    Rounded = _mec_dc_Rounded
    Subnormal = _mec_dc_Subnormal
    Overflow = _mec_dc_Overflow
    Underflow = _mec_dc_Underflow
    FloatOperation = _mec_dc_FloatOperation
    DivisionImpossible = _mec_dc_DivisionImpossible
    InvalidContext = _mec_dc_InvalidContext
    ConversionSyntax = _mec_dc_ConversionSyntax
    DivisionUndefined = _mec_dc_DivisionUndefined
    ROUND_DOWN = "ROUND_DOWN"
    ROUND_HALF_UP = "ROUND_HALF_UP"
    ROUND_HALF_EVEN = "ROUND_HALF_EVEN"
    ROUND_CEILING = "ROUND_CEILING"
    ROUND_FLOOR = "ROUND_FLOOR"
    ROUND_UP = "ROUND_UP"
    ROUND_HALF_DOWN = "ROUND_HALF_DOWN"
    ROUND_05UP = "ROUND_05UP"
    MAX_PREC = 999999999999999999
    MAX_EMAX = 999999999999999999
    MIN_EMIN = -999999999999999999
    MIN_ETINY = -1999999999999999997
    HAVE_THREADS = True
    HAVE_CONTEXTVAR = True
    DefaultContext = _mec_dc_DEFAULT[0]
    BasicContext = _mec_dc_BASIC
    ExtendedContext = _mec_dc_EXTENDED
    getcontext = staticmethod(_mec_dc_getcontext)
    setcontext = staticmethod(_mec_dc_setcontext)
    localcontext = staticmethod(_mec_dc_localcontext)

    def __repr__(self):
        return "<module 'decimal'>"


_mec_dc_getcontext.__name__ = "getcontext"
_mec_dc_setcontext.__name__ = "setcontext"
_mec_dc_localcontext.__name__ = "localcontext"
_mec_mod_decimal = type("module", (_mec_dc_module,), {})()
_mec_mod_decimal.__name__ = "decimal"
