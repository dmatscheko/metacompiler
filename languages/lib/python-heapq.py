# python-heapq.py -- the heapq module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_heapq); helpers
# carry the _mec_hq prefix. The sifts are Lib/heapq.py's algorithm, so a heap's
# list layout is CPython's item for item. nsmallest / nlargest are the
# sorted(...)[:n] the documentation states them equivalent to, and merge steps
# its inputs lazily, the earlier input first on a tie.


# Move the item at pos up towards startpos while it is smaller than its parent;
# a max-heap compares the other way round.
def _mec_hq_down(heap, startpos, pos, mx):
    newitem = heap[pos]
    while pos > startpos:
        parentpos = (pos - 1) >> 1
        parent = heap[parentpos]
        if (parent < newitem) if mx else (newitem < parent):
            heap[pos] = parent
            pos = parentpos
            continue
        break
    heap[pos] = newitem


# Carry the item at pos down to a leaf along the smaller children, then sift it
# back up: fewer comparisons than stopping early, and CPython's layout.
def _mec_hq_up(heap, pos, mx):
    endpos = len(heap)
    startpos = pos
    newitem = heap[pos]
    childpos = 2 * pos + 1
    while childpos < endpos:
        rightpos = childpos + 1
        if rightpos < endpos:
            if (not heap[rightpos] < heap[childpos]) if mx else (not heap[childpos] < heap[rightpos]):
                childpos = rightpos
        heap[pos] = heap[childpos]
        pos = childpos
        childpos = 2 * pos + 1
    heap[pos] = newitem
    _mec_hq_down(heap, startpos, pos, mx)


def _mec_hq_push(heap, item, mx):
    heap.append(item)
    _mec_hq_down(heap, 0, len(heap) - 1, mx)


def _mec_hq_pop(heap, mx):
    if not heap:
        raise IndexError("index out of range")
    lastelt = heap.pop()
    if heap:
        returnitem = heap[0]
        heap[0] = lastelt
        _mec_hq_up(heap, 0, mx)
        return returnitem
    return lastelt


def _mec_hq_replace(heap, item, mx):
    if not heap:
        raise IndexError("index out of range")
    returnitem = heap[0]
    heap[0] = item
    _mec_hq_up(heap, 0, mx)
    return returnitem


def _mec_hq_pushpop(heap, item, mx):
    if heap and ((item < heap[0]) if mx else (heap[0] < item)):
        item, heap[0] = heap[0], item
        _mec_hq_up(heap, 0, mx)
    return item


def _mec_hq_heapify(x, mx):
    for i in reversed(range(len(x) // 2)):
        _mec_hq_up(x, i, mx)


def _mec_hq_merge(iterables, key, reverse):
    its = [iter(it) for it in iterables]
    heads = []
    live = []
    for i in range(len(its)):
        for v in its[i]:
            heads.append(v)
            live.append(i)
            break
    while live:
        best = 0
        for j in range(1, len(live)):
            a = heads[j] if key is None else key(heads[j])
            b = heads[best] if key is None else key(heads[best])
            if (b < a) if reverse else (a < b):
                best = j
        yield heads[best]
        src = its[live[best]]
        got = False
        for v in src:
            heads[best] = v
            got = True
            break
        if not got:
            del heads[best]
            del live[best]


class _mec_hq_module:
    __all__ = ["heappush", "heappop", "heapify", "heapreplace", "merge", "nlargest",
               "nsmallest", "heappushpop", "heappush_max", "heappop_max", "heapify_max",
               "heapreplace_max", "heappushpop_max"]

    @staticmethod
    def heappush(heap, item, /):
        _mec_hq_push(heap, item, False)

    @staticmethod
    def heappop(heap, /):
        return _mec_hq_pop(heap, False)

    @staticmethod
    def heapify(x, /):
        _mec_hq_heapify(x, False)

    @staticmethod
    def heapreplace(heap, item, /):
        return _mec_hq_replace(heap, item, False)

    @staticmethod
    def heappushpop(heap, item, /):
        return _mec_hq_pushpop(heap, item, False)

    @staticmethod
    def heappush_max(heap, item, /):
        _mec_hq_push(heap, item, True)

    @staticmethod
    def heappop_max(heap, /):
        return _mec_hq_pop(heap, True)

    @staticmethod
    def heapify_max(x, /):
        _mec_hq_heapify(x, True)

    @staticmethod
    def heapreplace_max(heap, item, /):
        return _mec_hq_replace(heap, item, True)

    @staticmethod
    def heappushpop_max(heap, item, /):
        return _mec_hq_pushpop(heap, item, True)

    @staticmethod
    def nsmallest(n, iterable, key=None):
        return sorted(iterable, key=key)[:max(n, 0)]

    @staticmethod
    def nlargest(n, iterable, key=None):
        return sorted(iterable, key=key, reverse=True)[:max(n, 0)]

    @staticmethod
    def merge(*iterables, key=None, reverse=False):
        return _mec_hq_merge(iterables, key, reverse)

    def __repr__(self):
        return "<module 'heapq'>"


_mec_mod_heapq = type("module", (_mec_hq_module,), {})()
_mec_mod_heapq.__name__ = "heapq"
