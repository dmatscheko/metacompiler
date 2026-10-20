# python-statistics.py -- the statistics module, written in Python and compiled
# with a program that imports it (core.importFile in both python grammars), so
# one implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_statistics);
# helpers carry the _mec_st prefix. The averages and spreads follow
# Lib/statistics.py (3.14): sums are exact (Fractions), the result takes the
# data's type (an int mean that is no integer becomes a float), and stdev is
# the correctly rounded square root of the exact variance. Offered: mean,
# fmean, geometric_mean, harmonic_mean, median (and _low, _high, _grouped),
# mode, multimode, quantiles, variance, pvariance, stdev, pstdev. Not offered:
# covariance, correlation and linear_regression (they need math.sumprod),
# NormalDist and kde.

import math as _mec_st_math
from fractions import Fraction as _mec_st_Fraction


class _mec_st_StatisticsError(ValueError):
    pass


_mec_st_StatisticsError.__name__ = "StatisticsError"
_mec_st_StatisticsError.__qualname__ = "StatisticsError"
_mec_st_StatisticsError.__module__ = "statistics"


# The common type of two data types: an int (or bool) takes the other one, a
# Fraction meeting a float is a float.
def _mec_st_coerce(t, s):
    if t is s:
        return t
    if s is int or s is bool:
        return t
    if t is int:
        return s
    if issubclass(s, t):
        return s
    if issubclass(t, s):
        return t
    if issubclass(t, int):
        return s
    if issubclass(s, int):
        return t
    if issubclass(t, _mec_st_Fraction) and issubclass(s, float):
        return s
    if issubclass(t, float) and issubclass(s, _mec_st_Fraction):
        return t
    raise TypeError("don't know how to coerce " + t.__name__ + " and " + s.__name__)


def _mec_st_ratio(x):
    try:
        return x.as_integer_ratio()
    except (OverflowError, ValueError):
        return (x, None)
    except AttributeError:
        raise TypeError("can't convert type '" + type(x).__name__ + "' to numerator/denominator")


# The exact sum of the data: (type, total, count). A non-finite float makes the
# total that float (its sum with the others).
def _mec_st_sum(data):
    count = 0
    t = int
    total = _mec_st_Fraction(0)
    special = None
    for x in data:
        t = _mec_st_coerce(t, type(x))
        n, d = _mec_st_ratio(x)
        count += 1
        if d is None:
            special = n if special is None else special + n
        else:
            total += _mec_st_Fraction(n, d)
    if special is not None:
        return t, special, count
    return t, total, count


def _mec_st_convert(value, t):
    if type(value) is t:
        return value
    if not isinstance(value, _mec_st_Fraction):
        return value
    if issubclass(t, int) and value.denominator != 1:
        t = float
    return t(value)


# (type, exact sum of squared deviations, count) of the data about its mean,
# or about c when one is given.
def _mec_st_ss(data, c=None):
    if c is not None:
        t, ssd, count = _mec_st_sum([(x - c) * (x - c) for x in data])
        return t, ssd, count
    t, total, count = _mec_st_sum(data)
    if count == 0:
        return t, _mec_st_Fraction(0), 0
    if not isinstance(total, _mec_st_Fraction):
        return t, total, count
    mean = total / count
    ssd = _mec_st_Fraction(0)
    for x in data:
        n, d = _mec_st_ratio(x)
        dev = _mec_st_Fraction(n, d) - mean
        ssd += dev * dev
    return t, ssd, count


# isqrt(n / m) rounded to odd: the truncated root with its low bit set when it
# is inexact, which keeps the one rounding to a double below correct.
def _mec_st_isqrt_odd(n, m):
    a = _mec_st_math.isqrt(n // m)
    return a | (a * a * m != n)


# sqrt(n / m) as the correctly rounded double, from a root carried to 109 bits.
def _mec_st_sqrt_frac(n, m):
    if n <= 0:
        if not n:
            return 0.0
        n, m = -n, -m
    q = (n.bit_length() - m.bit_length() - 109) // 2
    if q >= 0:
        return (_mec_st_isqrt_odd(n, m << (2 * q)) << q) / 1
    return _mec_st_isqrt_odd(n << (-2 * q), m) / (1 << -q)


def _mec_st_spread_sqrt(v):
    if isinstance(v, _mec_st_Fraction):
        return _mec_st_sqrt_frac(v.numerator, v.denominator)
    return _mec_st_math.sqrt(v)


class _mec_st_module:
    __all__ = ["StatisticsError", "fmean", "geometric_mean", "harmonic_mean", "mean", "median",
               "median_grouped", "median_high", "median_low", "mode", "multimode", "pstdev", "pvariance",
               "quantiles", "stdev", "variance"]
    StatisticsError = _mec_st_StatisticsError

    @staticmethod
    def mean(data):
        t, total, n = _mec_st_sum(data)
        if n < 1:
            raise _mec_st_StatisticsError("mean requires at least one data point")
        return _mec_st_convert(total / n, t)

    @staticmethod
    def fmean(data, weights=None):
        data = list(data)
        if weights is None:
            if not data:
                raise _mec_st_StatisticsError("fmean requires at least one data point")
            return _mec_st_math.fsum(data) / len(data)
        weights = list(weights)
        if len(weights) != len(data):
            raise _mec_st_StatisticsError("data and weights must be the same length")
        num = _mec_st_math.fsum([x * w for x, w in zip(data, weights)])
        den = _mec_st_math.fsum(weights)
        if not den:
            raise _mec_st_StatisticsError("sum of weights must be non-zero")
        return num / den

    @staticmethod
    def geometric_mean(data):
        n = 0
        found_zero = False
        logs = []
        for x in data:
            n += 1
            if x > 0.0 or _mec_st_math.isnan(x):
                logs.append(_mec_st_math.log(x))
            elif x == 0.0:
                found_zero = True
            else:
                raise _mec_st_StatisticsError("No negative inputs allowed", x)
        total = _mec_st_math.fsum(logs)
        if not n:
            raise _mec_st_StatisticsError("Must have a non-empty dataset")
        if _mec_st_math.isnan(total):
            return _mec_st_math.nan
        if found_zero:
            return _mec_st_math.nan if total == _mec_st_math.inf else 0.0
        return _mec_st_math.exp(total / n)

    @staticmethod
    def harmonic_mean(data, weights=None):
        data = list(data)
        errmsg = "harmonic mean does not support negative values"
        n = len(data)
        if n < 1:
            raise _mec_st_StatisticsError("harmonic_mean requires at least one data point")
        if n == 1 and weights is None:
            x = data[0]
            if x < 0:
                raise _mec_st_StatisticsError(errmsg)
            return x
        if weights is None:
            weights = [1] * n
            sum_weights = n
        else:
            weights = list(weights)
            if len(weights) != n:
                raise _mec_st_StatisticsError("Number of weights does not match data size")
            sum_weights = _mec_st_sum([w for w in weights])[1]
            for w in weights:
                if w < 0:
                    raise _mec_st_StatisticsError("Weights cannot be negative")
        # In data order: a negative value is refused when it is reached, and a
        # zero reached first makes the mean 0.
        terms = []
        for w, x in zip(weights, data):
            if x < 0:
                raise _mec_st_StatisticsError(errmsg)
            if w:
                if x == 0:
                    return 0
                terms.append(w / x)
            else:
                terms.append(0)
        t, total, count = _mec_st_sum(terms)
        if total <= 0:
            raise _mec_st_StatisticsError("Weighted sum must be positive")
        return _mec_st_convert(sum_weights / total, t)

    @staticmethod
    def median(data):
        data = sorted(data)
        n = len(data)
        if n == 0:
            raise _mec_st_StatisticsError("no median for empty data")
        if n % 2 == 1:
            return data[n // 2]
        i = n // 2
        return (data[i - 1] + data[i]) / 2

    @staticmethod
    def median_low(data):
        data = sorted(data)
        n = len(data)
        if n == 0:
            raise _mec_st_StatisticsError("no median for empty data")
        if n % 2 == 1:
            return data[n // 2]
        return data[n // 2 - 1]

    @staticmethod
    def median_high(data):
        data = sorted(data)
        n = len(data)
        if n == 0:
            raise _mec_st_StatisticsError("no median for empty data")
        return data[n // 2]

    @staticmethod
    def median_grouped(data, interval=1.0):
        data = sorted(data)
        n = len(data)
        if not n:
            raise _mec_st_StatisticsError("no median for empty data")
        x = data[n // 2]
        i = data.index(x)
        j = i
        while j < n and data[j] == x:
            j += 1
        interval = float(interval)
        x = float(x)
        lower = x - interval / 2.0
        return lower + interval * (n / 2 - i) / (j - i)

    @staticmethod
    def mode(data):
        counts = {}
        for x in data:
            counts[x] = counts.get(x, 0) + 1
        if not counts:
            raise _mec_st_StatisticsError("no mode for empty data")
        best, bestc = None, 0
        for k, c in counts.items():
            if c > bestc:
                best, bestc = k, c
        return best

    @staticmethod
    def multimode(data):
        counts = {}
        for x in data:
            counts[x] = counts.get(x, 0) + 1
        if not counts:
            return []
        top = max(counts.values())
        return [k for k, c in counts.items() if c == top]

    @staticmethod
    def quantiles(data, *, n=4, method="exclusive"):
        if n < 1:
            raise _mec_st_StatisticsError("n must be at least 1")
        data = sorted(data)
        ld = len(data)
        if ld < 2:
            if ld == 1:
                return data * (n - 1)
            raise _mec_st_StatisticsError("must have at least one data point")
        if method == "inclusive":
            m = ld - 1
            result = []
            for i in range(1, n):
                j, delta = divmod(i * m, n)
                result.append((data[j] * (n - delta) + data[j + 1] * delta) / n)
            return result
        if method == "exclusive":
            m = ld + 1
            result = []
            for i in range(1, n):
                j = i * m // n
                j = 1 if j < 1 else ld - 1 if j > ld - 1 else j
                delta = i * m - j * n
                result.append((data[j - 1] * (n - delta) + data[j] * delta) / n)
            return result
        raise ValueError("Unknown method: " + repr(method))

    @staticmethod
    def variance(data, xbar=None):
        data = list(data)
        t, ss, n = _mec_st_ss(data, xbar)
        if n < 2:
            raise _mec_st_StatisticsError("variance requires at least two data points")
        return _mec_st_convert(ss / (n - 1), t)

    @staticmethod
    def pvariance(data, mu=None):
        data = list(data)
        t, ss, n = _mec_st_ss(data, mu)
        if n < 1:
            raise _mec_st_StatisticsError("pvariance requires at least one data point")
        return _mec_st_convert(ss / n, t)

    @staticmethod
    def stdev(data, xbar=None):
        data = list(data)
        t, ss, n = _mec_st_ss(data, xbar)
        if n < 2:
            raise _mec_st_StatisticsError("stdev requires at least two data points")
        return _mec_st_spread_sqrt(ss / (n - 1))

    @staticmethod
    def pstdev(data, mu=None):
        data = list(data)
        t, ss, n = _mec_st_ss(data, mu)
        if n < 1:
            raise _mec_st_StatisticsError("pstdev requires at least one data point")
        return _mec_st_spread_sqrt(ss / n)

    def __repr__(self):
        return "<module 'statistics'>"


_mec_mod_statistics = type("module", (_mec_st_module,), {})()
_mec_mod_statistics.__name__ = "statistics"
