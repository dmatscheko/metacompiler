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
# rather than at import. A str, bytes or bytearray seed is followed by its
# SHA-512 digest (written out here) and read as one big-endian int, CPython's
# version 2 seeding; a float seed is its hash(), as there.

import math as _mec_rd_math

_mec_rd_mask = 0xFFFFFFFF
_mec_rd_unseeded = object()


_mec_rd_M64 = 0xFFFFFFFFFFFFFFFF
_mec_rd_K512 = [
    0x428a2f98d728ae22, 0x7137449123ef65cd, 0xb5c0fbcfec4d3b2f, 0xe9b5dba58189dbbc, 0x3956c25bf348b538,
    0x59f111f1b605d019, 0x923f82a4af194f9b, 0xab1c5ed5da6d8118, 0xd807aa98a3030242, 0x12835b0145706fbe,
    0x243185be4ee4b28c, 0x550c7dc3d5ffb4e2, 0x72be5d74f27b896f, 0x80deb1fe3b1696b1, 0x9bdc06a725c71235,
    0xc19bf174cf692694, 0xe49b69c19ef14ad2, 0xefbe4786384f25e3, 0x0fc19dc68b8cd5b5, 0x240ca1cc77ac9c65,
    0x2de92c6f592b0275, 0x4a7484aa6ea6e483, 0x5cb0a9dcbd41fbd4, 0x76f988da831153b5, 0x983e5152ee66dfab,
    0xa831c66d2db43210, 0xb00327c898fb213f, 0xbf597fc7beef0ee4, 0xc6e00bf33da88fc2, 0xd5a79147930aa725,
    0x06ca6351e003826f, 0x142929670a0e6e70, 0x27b70a8546d22ffc, 0x2e1b21385c26c926, 0x4d2c6dfc5ac42aed,
    0x53380d139d95b3df, 0x650a73548baf63de, 0x766a0abb3c77b2a8, 0x81c2c92e47edaee6, 0x92722c851482353b,
    0xa2bfe8a14cf10364, 0xa81a664bbc423001, 0xc24b8b70d0f89791, 0xc76c51a30654be30, 0xd192e819d6ef5218,
    0xd69906245565a910, 0xf40e35855771202a, 0x106aa07032bbd1b8, 0x19a4c116b8d2d0c8, 0x1e376c085141ab53,
    0x2748774cdf8eeb99, 0x34b0bcb5e19b48a8, 0x391c0cb3c5c95a63, 0x4ed8aa4ae3418acb, 0x5b9cca4f7763e373,
    0x682e6ff3d6b2b8a3, 0x748f82ee5defb2fc, 0x78a5636f43172f60, 0x84c87814a1f0ab72, 0x8cc702081a6439ec,
    0x90befffa23631e28, 0xa4506cebde82bde9, 0xbef9a3f7b2c67915, 0xc67178f2e372532b, 0xca273eceea26619c,
    0xd186b8c721c0c207, 0xeada7dd6cde0eb1e, 0xf57d4f7fee6ed178, 0x06f067aa72176fba, 0x0a637dc5a2c898a6,
    0x113f9804bef90dae, 0x1b710b35131c471b, 0x28db77f523047d84, 0x32caab7b40c72493, 0x3c9ebe0a15c9bebc,
    0x431d67c49c100d4c, 0x4cc5d4becb3e42b6, 0x597f299cfc657e2a, 0x5fcb6fab3ad6faec, 0x6c44198c4a475817]


def _mec_rd_rotr(x, n):
    return ((x >> n) | (x << (64 - n))) & _mec_rd_M64


# The SHA-512 digest of a list of byte values, as a list of 64 byte values.
def _mec_rd_sha512(data):
    h = [0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
         0x510e527fade682d1, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179]
    n = len(data)
    msg = list(data) + [0x80] + [0] * ((111 - n) % 128) + list((n * 8).to_bytes(16, "big"))
    k = _mec_rd_K512
    rotr = _mec_rd_rotr
    m64 = _mec_rd_M64
    for at in range(0, len(msg), 128):
        w = [int.from_bytes(bytes(msg[at + 8 * i:at + 8 * i + 8]), "big") for i in range(16)]
        for i in range(16, 80):
            x, y = w[i - 15], w[i - 2]
            s0 = rotr(x, 1) ^ rotr(x, 8) ^ (x >> 7)
            s1 = rotr(y, 19) ^ rotr(y, 61) ^ (y >> 6)
            w.append((w[i - 16] + s0 + w[i - 7] + s1) & m64)
        a, b, c, d, e, f, g, hh = h
        for i in range(80):
            t1 = (hh + (rotr(e, 14) ^ rotr(e, 18) ^ rotr(e, 41)) + ((e & f) ^ (~e & g)) + k[i] + w[i]) & m64
            t2 = ((rotr(a, 28) ^ rotr(a, 34) ^ rotr(a, 39)) + ((a & b) ^ (a & c) ^ (b & c))) & m64
            hh, g, f, e, d, c, b, a = g, f, e, (d + t1) & m64, c, b, a, (t1 + t2) & m64
        h = [(x + y) & m64 for x, y in zip(h, [a, b, c, d, e, f, g, hh])]
    out = []
    for x in h:
        out.extend(x.to_bytes(8, "big"))
    return out


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
    __all__ = ["Random", "betavariate", "choice", "choices", "expovariate", "gammavariate", "gauss",
               "getrandbits", "getstate", "lognormvariate", "normalvariate", "paretovariate", "randbytes",
               "randint", "random", "randrange", "sample", "seed", "setstate", "shuffle", "triangular",
               "uniform", "weibullvariate"]

    class Random:
        def __init__(self, x=None):
            self._mec_rd_mt = _mec_rd_MT()
            self.gauss_next = None
            if x is not _mec_rd_unseeded:
                self.seed(x)

        def seed(self, a=None):
            self.gauss_next = None
            if a is None:
                a = 0
            if isinstance(a, float):
                a = hash(a)
            if isinstance(a, (str, bytes, bytearray)):
                data = list(a.encode() if isinstance(a, str) else a)
                a = int.from_bytes(bytes(data + _mec_rd_sha512(data)), "big")
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

        # (version, the 624 state words and the position, gauss_next), as
        # CPython's getstate answers.
        def getstate(self):
            mt = self._mec_rd_mt
            if mt.index < 0:
                mt.init_by_array([0])
            return (3, tuple(mt.mt) + (mt.index,), self.gauss_next)

        def setstate(self, state):
            if state[0] != 3:
                raise ValueError("state with version " + str(state[0]) + " passed to Random.setstate() "
                                 "of version 3")
            words = state[1]
            self._mec_rd_mt.mt = list(words[:624])
            self._mec_rd_mt.index = words[624]
            self.gauss_next = state[2]

        def randbytes(self, n):
            return self.getrandbits(n * 8).to_bytes(n, "little")

        def triangular(self, low=0.0, high=1.0, mode=None):
            u = self.random()
            try:
                c = 0.5 if mode is None else (mode - low) / (high - low)
            except ZeroDivisionError:
                return low
            if u > c:
                u = 1.0 - u
                c = 1.0 - c
                low, high = high, low
            return low + (high - low) * _mec_rd_math.sqrt(u * c)

        # Kinderman and Monahan's ratio of uniforms.
        def normalvariate(self, mu=0.0, sigma=1.0):
            while True:
                u1 = self.random()
                u2 = 1.0 - self.random()
                z = _mec_rd_NV_MAGICCONST * (u1 - 0.5) / u2
                zz = z * z / 4.0
                if zz <= -_mec_rd_math.log(u2):
                    break
            return mu + z * sigma

        # Box-Muller, the second value of a pair kept for the next call.
        def gauss(self, mu=0.0, sigma=1.0):
            z = self.gauss_next
            self.gauss_next = None
            if z is None:
                x2pi = self.random() * _mec_rd_math.tau
                g2rad = _mec_rd_math.sqrt(-2.0 * _mec_rd_math.log(1.0 - self.random()))
                z = _mec_rd_math.cos(x2pi) * g2rad
                self.gauss_next = _mec_rd_math.sin(x2pi) * g2rad
            return mu + z * sigma

        def lognormvariate(self, mu, sigma):
            return _mec_rd_math.exp(self.normalvariate(mu, sigma))

        def expovariate(self, lambd=1.0):
            return -_mec_rd_math.log(1.0 - self.random()) / lambd

        def paretovariate(self, alpha):
            u = 1.0 - self.random()
            return u ** (-1.0 / alpha)

        def weibullvariate(self, alpha, beta):
            u = 1.0 - self.random()
            return alpha * (-_mec_rd_math.log(u)) ** (1.0 / beta)

        # Cheng's rejection method above 1, an exponential at 1, and
        # algorithm GS (Kennedy and Gentle) below.
        def gammavariate(self, alpha, beta):
            if alpha <= 0.0 or beta <= 0.0:
                raise ValueError("gammavariate: alpha and beta must be > 0.0")
            if alpha > 1.0:
                ainv = _mec_rd_math.sqrt(2.0 * alpha - 1.0)
                bbb = alpha - _mec_rd_LOG4
                ccc = alpha + ainv
                while True:
                    u1 = self.random()
                    if not 1e-7 < u1 < 0.9999999:
                        continue
                    u2 = 1.0 - self.random()
                    v = _mec_rd_math.log(u1 / (1.0 - u1)) / ainv
                    x = alpha * _mec_rd_math.exp(v)
                    z = u1 * u1 * u2
                    r = bbb + ccc * v - x
                    if r + _mec_rd_SG_MAGICCONST - 4.5 * z >= 0.0 or r >= _mec_rd_math.log(z):
                        return x * beta
            if alpha == 1.0:
                return -_mec_rd_math.log(1.0 - self.random()) * beta
            while True:
                u = self.random()
                b = (_mec_rd_math.e + alpha) / _mec_rd_math.e
                p = b * u
                if p <= 1.0:
                    x = p ** (1.0 / alpha)
                else:
                    x = -_mec_rd_math.log((b - p) / alpha)
                u1 = self.random()
                if p > 1.0:
                    if u1 <= x ** (alpha - 1.0):
                        break
                elif u1 <= _mec_rd_math.exp(-x):
                    break
            return x * beta

        def betavariate(self, alpha, beta):
            y = self.gammavariate(alpha, 1.0)
            if y:
                return y / (y + self.gammavariate(beta, 1.0))
            return 0.0

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

    @staticmethod
    def getstate():
        return _mec_rd_inst.getstate()

    @staticmethod
    def setstate(state):
        _mec_rd_inst.setstate(state)

    @staticmethod
    def randbytes(n):
        return _mec_rd_inst.randbytes(n)

    @staticmethod
    def triangular(low=0.0, high=1.0, mode=None):
        return _mec_rd_inst.triangular(low, high, mode)

    @staticmethod
    def normalvariate(mu=0.0, sigma=1.0):
        return _mec_rd_inst.normalvariate(mu, sigma)

    @staticmethod
    def gauss(mu=0.0, sigma=1.0):
        return _mec_rd_inst.gauss(mu, sigma)

    @staticmethod
    def lognormvariate(mu, sigma):
        return _mec_rd_inst.lognormvariate(mu, sigma)

    @staticmethod
    def expovariate(lambd=1.0):
        return _mec_rd_inst.expovariate(lambd)

    @staticmethod
    def paretovariate(alpha):
        return _mec_rd_inst.paretovariate(alpha)

    @staticmethod
    def weibullvariate(alpha, beta):
        return _mec_rd_inst.weibullvariate(alpha, beta)

    @staticmethod
    def gammavariate(alpha, beta):
        return _mec_rd_inst.gammavariate(alpha, beta)

    @staticmethod
    def betavariate(alpha, beta):
        return _mec_rd_inst.betavariate(alpha, beta)

    def __repr__(self):
        return "<module 'random'>"


_mec_rd_NV_MAGICCONST = 4 * _mec_rd_math.exp(-0.5) / _mec_rd_math.sqrt(2.0)
_mec_rd_LOG4 = _mec_rd_math.log(4.0)
_mec_rd_SG_MAGICCONST = 1.0 + _mec_rd_math.log(4.5)


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
