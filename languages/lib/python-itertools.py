# python-itertools.py -- the itertools module, written in Python and compiled
# with a program that imports it (core.importFile in both python grammars), so
# one implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_itertools), never
# as a flat name, so `import itertools` cannot replace a program's own count or
# product; helpers carry the _mec_it prefix. The module object is an instance
# of a class named "module", as type(itertools).__name__ is in CPython.
#
# The results and error texts are Modules/itertoolsmodule.c's. A tool over a
# FINITE input computes its whole result and answers an iterator over it, the
# line this runtime draws for map, zip and filter too: the interpreter runs a
# generator by replaying its body, so a generator that pulled from a one-shot
# input across a yield would pull again on every replay. count, cycle and
# repeat are endless and are generators over state of their own; islice and
# takewhile stop pulling from their input where CPython does, so they work on
# an endless one. A group from groupby is therefore a complete list iterator
# that later steps of the outer iterator do not empty.


def _mec_it_count(start, step):
    n = start
    while True:
        yield n
        n += step


def _mec_it_cycle(saved):
    while saved:
        for x in saved:
            yield x


def _mec_it_repeat(object):
    while True:
        yield object


def _mec_it_index(v, what):
    if v is None:
        return None
    if not isinstance(v, int) or v < 0:
        raise ValueError(what + " for islice() must be None or an integer: 0 <= x <= sys.maxsize.")
    return v


def _mec_it_from_iterable(iterables):
    return iter([x for it in iterables for x in it])


class _mec_it_module:
    __all__ = ["accumulate", "batched", "chain", "combinations", "combinations_with_replacement", "compress",
               "count", "cycle", "dropwhile", "filterfalse", "groupby", "islice", "pairwise", "permutations",
               "product", "repeat", "starmap", "takewhile", "tee", "zip_longest"]

    @staticmethod
    def count(start=0, step=1):
        return _mec_it_count(start, step)

    @staticmethod
    def cycle(iterable):
        return _mec_it_cycle(list(iterable))

    @staticmethod
    def repeat(object, times=None):
        if times is None:
            return _mec_it_repeat(object)
        return iter([object] * max(times, 0))

    @staticmethod
    def accumulate(iterable, func=None, *, initial=None):
        out = [] if initial is None else [initial]
        for x in iterable:
            if not out:
                out.append(x)
            else:
                out.append(out[-1] + x if func is None else func(out[-1], x))
        return iter(out)

    @staticmethod
    def batched(iterable, n, *, strict=False):
        if n < 1:
            raise ValueError("n must be at least one")
        items = list(iterable)
        out = [tuple(items[i:i + n]) for i in range(0, len(items), n)]
        if strict and out and len(out[-1]) != n:
            raise ValueError("batched(): incomplete batch")
        return iter(out)

    @staticmethod
    def chain(*iterables):
        return iter([x for it in iterables for x in it])

    @staticmethod
    def compress(data, selectors):
        return iter([d for d, s in zip(data, selectors) if s])

    @staticmethod
    def dropwhile(predicate, iterable):
        out = []
        dropping = True
        for x in iterable:
            if dropping and predicate(x):
                continue
            dropping = False
            out.append(x)
        return iter(out)

    @staticmethod
    def takewhile(predicate, iterable):
        out = []
        for x in iterable:
            if not predicate(x):
                break
            out.append(x)
        return iter(out)

    @staticmethod
    def filterfalse(predicate, iterable):
        return iter([x for x in iterable if not (x if predicate is None else predicate(x))])

    @staticmethod
    def groupby(iterable, key=None):
        out = []
        for x in iterable:
            k = x if key is None else key(x)
            if out and out[-1][0] == k:
                out[-1][1].append(x)
            else:
                out.append((k, [x]))
        return iter([(k, iter(g)) for k, g in out])

    # Exhausting the result reads the input up to `stop` and no further, as in
    # CPython, so an endless input works with a stop.
    @staticmethod
    def islice(iterable, *args):
        if len(args) == 1:
            start, stop, step = 0, args[0], 1
        elif len(args) == 2 or len(args) == 3:
            start, stop = args[0], args[1]
            step = args[2] if len(args) == 3 else 1
        else:
            raise TypeError("islice expected at most 4 arguments, got " + str(len(args) + 1))
        stop = _mec_it_index(stop, "Stop argument")
        start = _mec_it_index(start, "Indices")
        if start is None:
            start = 0
        if step is None:
            step = 1
        if not isinstance(step, int) or step < 1:
            raise ValueError("Step for islice() must be a positive integer or None.")
        it = iter(iterable)
        if stop is None:
            return iter(list(it)[start::step])
        out = []
        i = 0
        nexti = start
        while i < stop:
            try:
                x = next(it)
            except StopIteration:
                break
            if i == nexti:
                out.append(x)
                nexti += step
            i += 1
        return iter(out)

    @staticmethod
    def pairwise(iterable):
        items = list(iterable)
        return iter([(items[i], items[i + 1]) for i in range(len(items) - 1)])

    @staticmethod
    def starmap(function, iterable):
        return iter([function(*args) for args in iterable])

    @staticmethod
    def tee(iterable, n=2):
        if n < 0:
            raise ValueError("n must be >= 0")
        items = list(iterable)
        return tuple(iter(items) for _ in range(n))

    @staticmethod
    def zip_longest(*iterables, fillvalue=None):
        pools = [list(x) for x in iterables]
        width = max([len(p) for p in pools], default=0)
        return iter([tuple(p[i] if i < len(p) else fillvalue for p in pools) for i in range(width)])

    @staticmethod
    def product(*iterables, repeat=1):
        if repeat < 0:
            raise ValueError("repeat argument cannot be negative")
        pools = [tuple(p) for p in iterables] * repeat
        result = [[]]
        for pool in pools:
            result = [x + [y] for x in result for y in pool]
        return iter([tuple(prod) for prod in result])

    @staticmethod
    def permutations(iterable, r=None):
        pool = tuple(iterable)
        n = len(pool)
        r = n if r is None else r
        if r < 0:
            raise ValueError("r must be non-negative")
        out = []
        if r > n:
            return iter(out)
        indices = list(range(n))
        cycles = list(range(n, n - r, -1))
        out.append(tuple(pool[k] for k in indices[:r]))
        while n:
            for i in reversed(range(r)):
                cycles[i] -= 1
                if cycles[i] == 0:
                    indices[i:] = indices[i + 1:] + indices[i:i + 1]
                    cycles[i] = n - i
                else:
                    j = cycles[i]
                    indices[i], indices[-j] = indices[-j], indices[i]
                    out.append(tuple(pool[k] for k in indices[:r]))
                    break
            else:
                break
        return iter(out)

    @staticmethod
    def combinations(iterable, r):
        pool = tuple(iterable)
        n = len(pool)
        if r < 0:
            raise ValueError("r must be non-negative")
        out = []
        if r > n:
            return iter(out)
        indices = list(range(r))
        out.append(tuple(pool[k] for k in indices))
        while True:
            for i in reversed(range(r)):
                if indices[i] != i + n - r:
                    break
            else:
                break
            indices[i] += 1
            for j in range(i + 1, r):
                indices[j] = indices[j - 1] + 1
            out.append(tuple(pool[k] for k in indices))
        return iter(out)

    @staticmethod
    def combinations_with_replacement(iterable, r):
        pool = tuple(iterable)
        n = len(pool)
        if r < 0:
            raise ValueError("r must be non-negative")
        out = []
        if not n and r:
            return iter(out)
        indices = [0] * r
        out.append(tuple(pool[k] for k in indices))
        while True:
            for i in reversed(range(r)):
                if indices[i] != n - 1:
                    break
            else:
                break
            indices[i:] = [indices[i] + 1] * (r - i)
            out.append(tuple(pool[k] for k in indices))
        return iter(out)

    def __repr__(self):
        return "<module 'itertools' (built-in)>"


_mec_it_module.chain.from_iterable = _mec_it_from_iterable
_mec_mod_itertools = type("module", (_mec_it_module,), {})()
_mec_mod_itertools.__name__ = "itertools"
