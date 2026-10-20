# python-bisect.py -- the bisect module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_bisect); helpers
# carry the _mec_bi prefix. The behaviour is Lib/bisect.py's, error texts
# included: key applies to the list's items, never to x.


# The bounds every function takes: lo must be non-negative, hi None means the end.
def _mec_bi_hi(a, lo, hi):
    if lo < 0:
        raise ValueError("lo must be non-negative")
    if hi is None:
        return len(a)
    return hi


# The first index in a[lo:hi] whose item is not less than x (left), or greater
# than x (right).
def _mec_bi_find(a, x, lo, hi, key, right):
    hi = _mec_bi_hi(a, lo, hi)
    while lo < hi:
        mid = (lo + hi) // 2
        v = a[mid] if key is None else key(a[mid])
        if (x < v) if right else (not v < x):
            hi = mid
        else:
            lo = mid + 1
    return lo


class _mec_bi_module:
    __all__ = ["bisect_left", "bisect_right", "insort_left", "insort_right", "bisect", "insort"]

    @staticmethod
    def bisect_left(a, x, lo=0, hi=None, *, key=None):
        return _mec_bi_find(a, x, lo, hi, key, False)

    @staticmethod
    def bisect_right(a, x, lo=0, hi=None, *, key=None):
        return _mec_bi_find(a, x, lo, hi, key, True)

    @staticmethod
    def bisect(a, x, lo=0, hi=None, *, key=None):
        return _mec_bi_find(a, x, lo, hi, key, True)

    @staticmethod
    def insort_left(a, x, lo=0, hi=None, *, key=None):
        a.insert(_mec_bi_find(a, x if key is None else key(x), lo, hi, key, False), x)

    @staticmethod
    def insort_right(a, x, lo=0, hi=None, *, key=None):
        a.insert(_mec_bi_find(a, x if key is None else key(x), lo, hi, key, True), x)

    @staticmethod
    def insort(a, x, lo=0, hi=None, *, key=None):
        a.insert(_mec_bi_find(a, x if key is None else key(x), lo, hi, key, True), x)

    def __repr__(self):
        return "<module 'bisect'>"


_mec_mod_bisect = type("module", (_mec_bi_module,), {})()
_mec_mod_bisect.__name__ = "bisect"
