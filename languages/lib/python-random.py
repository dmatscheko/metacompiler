# python-random.py -- the random module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_random), never as
# a flat name, so `import random` cannot replace a program's own choice or
# shuffle; helpers carry the _mec_rd prefix. The module object is an instance of
# a class named "module", as type(random).__name__ is in CPython.
#
# The generator is CPython's: MT19937 (Modules/_randommodule.c), seeded from an
# int by init_by_array over its 32-bit words, and the derived methods are
# Lib/random.py's, so a seeded sequence is CPython's draw for draw. There is no
# entropy source here: an unseeded generator starts from seed(0) rather than
# from os.urandom, and the module's own generator is seeded at its first draw
# rather than at import.

_mec_rd_mask = 0xFFFFFFFF
_mec_rd_unseeded = object()


# An index of -1 is a state not seeded yet, which the first draw seeds as seed(0).
# The seeding loops multiply a 32-bit word by a constant below 2**31 modulo 2**32
# by 16-bit halves, (c * (x >> 16) & 0xFFFF) << 16 plus c * (x & 0xFFFF), so no
# partial product leaves the integers a double holds exactly.
class _mec_rd_MT:
    def __init__(self):
        self.mt = [0] * 624
        self.index = -1

    def init_genrand(self, s):
        mt = self.mt
        mt[0] = s & _mec_rd_mask
        for i in range(1, 624):
            x = mt[i - 1] ^ (mt[i - 1] >> 30)
            mt[i] = ((((1812433253 * (x >> 16)) & 0xFFFF) << 16) + 1812433253 * (x & 0xFFFF) + i) & _mec_rd_mask
        self.index = 624

    def init_by_array(self, key):
        self.init_genrand(19650218)
        mt = self.mt
        i, j = 1, 0
        n = len(key)
        for _ in range(max(624, n)):
            x = mt[i - 1] ^ (mt[i - 1] >> 30)
            x = ((((1664525 * (x >> 16)) & 0xFFFF) << 16) + 1664525 * (x & 0xFFFF)) & _mec_rd_mask
            mt[i] = ((mt[i] ^ x) + key[j] + j) & _mec_rd_mask
            i += 1
            j += 1
            if i >= 624:
                mt[0] = mt[623]
                i = 1
            if j >= n:
                j = 0
        for _ in range(623):
            x = mt[i - 1] ^ (mt[i - 1] >> 30)
            x = ((((1566083941 * (x >> 16)) & 0xFFFF) << 16) + 1566083941 * (x & 0xFFFF)) & _mec_rd_mask
            mt[i] = ((mt[i] ^ x) - i) & _mec_rd_mask
            i += 1
            if i >= 624:
                mt[0] = mt[623]
                i = 1
        mt[0] = 0x80000000

    def genrand(self):
        mt = self.mt
        if self.index < 0:
            self.init_by_array([0])
        if self.index >= 624:
            for k in range(624):
                y = (mt[k] & 0x80000000) | (mt[(k + 1) % 624] & 0x7FFFFFFF)
                v = mt[(k + 397) % 624] ^ (y >> 1)
                if y & 1:
                    v ^= 0x9908B0DF
                mt[k] = v
            self.index = 0
        y = mt[self.index]
        self.index += 1
        y ^= y >> 11
        y ^= (y << 7) & 0x9D2C5680
        y ^= (y << 15) & 0xEFC60000
        y ^= y >> 18
        return y


class _mec_rd_module:
    __all__ = ["Random", "choice", "choices", "getrandbits", "randint", "random", "randrange", "sample", "seed",
               "shuffle", "uniform"]

    class Random:
        def __init__(self, x=None):
            self._mec_rd_mt = _mec_rd_MT()
            if x is not _mec_rd_unseeded:
                self.seed(x)

        def seed(self, a=None):
            if a is None:
                a = 0
            if isinstance(a, (str, bytes, float)):
                raise NotImplementedError("random.seed() of a " + type(a).__name__ + " is not supported")
            if not isinstance(a, int):
                raise TypeError("The only supported seed types are:\nNone, int, float, str, bytes, and bytearray.")
            a = abs(a)
            key = []
            while a:
                key.append(a & _mec_rd_mask)
                a >>= 32
            self._mec_rd_mt.init_by_array(key or [0])

        def random(self):
            a = self._mec_rd_mt.genrand() >> 5
            b = self._mec_rd_mt.genrand() >> 6
            return (a * 67108864.0 + b) * (1.0 / 9007199254740992.0)

        def getrandbits(self, k):
            if k < 0:
                raise ValueError("Cannot convert negative int")
            if k <= 32:
                return self._mec_rd_mt.genrand() >> (32 - k) if k else 0
            out = 0
            shift = 0
            while k > 0:
                r = self._mec_rd_mt.genrand()
                if k < 32:
                    r >>= 32 - k
                out |= r << shift
                shift += 32
                k -= 32
            return out

        def _randbelow(self, n):
            k = n.bit_length()
            r = self.getrandbits(k)
            while r >= n:
                r = self.getrandbits(k)
            return r

        def randrange(self, start, stop=None, step=1):
            if stop is None:
                if start > 0:
                    return self._randbelow(start)
                raise ValueError("empty range for randrange()")
            width = stop - start
            if step == 1:
                if width > 0:
                    return start + self._randbelow(width)
                raise ValueError("empty range in randrange(%d, %d)" % (start, stop))
            if step > 0:
                n = (width + step - 1) // step
            elif step < 0:
                n = (width + step + 1) // step
            else:
                raise ValueError("zero step for randrange()")
            if n <= 0:
                raise ValueError("empty range in randrange(%d, %d, %d)" % (start, stop, step))
            return start + step * self._randbelow(n)

        def randint(self, a, b):
            return self.randrange(a, b + 1)

        def choice(self, seq):
            if not len(seq):
                raise IndexError("Cannot choose from an empty sequence")
            return seq[self._randbelow(len(seq))]

        def shuffle(self, x):
            for i in reversed(range(1, len(x))):
                j = self._randbelow(i + 1)
                x[i], x[j] = x[j], x[i]

        def uniform(self, a, b):
            return a + (b - a) * self.random()

        def choices(self, population, weights=None, *, cum_weights=None, k=1):
            n = len(population)
            if cum_weights is None:
                if weights is None:
                    n += 0.0
                    return [population[int(self.random() * n)] for _ in range(k)]
                cum_weights = []
                total = 0
                for w in weights:
                    total += w
                    cum_weights.append(total)
            elif weights is not None:
                raise TypeError("Cannot specify both weights and cumulative weights")
            if len(cum_weights) != n:
                raise ValueError("The number of weights does not match the population")
            total = cum_weights[-1] + 0.0
            if total <= 0.0:
                raise ValueError("Total of weights must be greater than zero")
            out = []
            for _ in range(k):
                x = self.random() * total
                lo, hi = 0, n - 1
                while lo < hi:
                    mid = (lo + hi) // 2
                    if x < cum_weights[mid]:
                        hi = mid
                    else:
                        lo = mid + 1
                out.append(population[lo])
            return out

        def sample(self, population, k):
            n = len(population)
            if not 0 <= k <= n:
                raise ValueError("Sample larger than population or is negative")
            result = [None] * k
            setsize = 21
            if k > 5:
                setsize += 4 ** _mec_rd_ceil_log4(k * 3)
            if n <= setsize:
                pool = list(population)
                for i in range(k):
                    j = self._randbelow(n - i)
                    result[i] = pool[j]
                    pool[j] = pool[n - i - 1]
            else:
                selected = set()
                for i in range(k):
                    j = self._randbelow(n)
                    while j in selected:
                        j = self._randbelow(n)
                    selected.add(j)
                    result[i] = population[j]
            return result

    # The module functions are one hidden instance's methods, as there.
    @staticmethod
    def seed(a=None):
        _mec_rd_inst.seed(a)

    @staticmethod
    def random():
        return _mec_rd_inst.random()

    @staticmethod
    def getrandbits(k):
        return _mec_rd_inst.getrandbits(k)

    @staticmethod
    def randrange(start, stop=None, step=1):
        return _mec_rd_inst.randrange(start, stop, step)

    @staticmethod
    def randint(a, b):
        return _mec_rd_inst.randint(a, b)

    @staticmethod
    def choice(seq):
        return _mec_rd_inst.choice(seq)

    @staticmethod
    def shuffle(x):
        _mec_rd_inst.shuffle(x)

    @staticmethod
    def uniform(a, b):
        return _mec_rd_inst.uniform(a, b)

    @staticmethod
    def choices(population, weights=None, *, cum_weights=None, k=1):
        return _mec_rd_inst.choices(population, weights, cum_weights=cum_weights, k=k)

    @staticmethod
    def sample(population, k):
        return _mec_rd_inst.sample(population, k)

    def __repr__(self):
        return "<module 'random'>"


# ceil(log(x, 4)) for the integer x > 0 that sample() asks about, exactly.
def _mec_rd_ceil_log4(x):
    e = 0
    p = 1
    while p < x:
        p *= 4
        e += 1
    return e


_mec_rd_inst = _mec_rd_module.Random(_mec_rd_unseeded)
_mec_mod_random = type("module", (_mec_rd_module,), {})()
_mec_mod_random.__name__ = "random"
