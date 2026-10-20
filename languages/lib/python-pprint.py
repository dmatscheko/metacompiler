# python-pprint.py -- the pprint module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_pprint); helpers
# carry the _mec_pp prefix. The layout is Lib/pprint.py's (3.14): a value whose
# repr fits the width is written whole, a dict, list, tuple, set, frozenset or
# str that does not is broken one item per line, dicts sorted by key unless
# sort_dicts is false. Instances of other types, dataclasses and bytes included,
# are written as their repr.


class _mec_pp_SafeKey:
    def __init__(self, obj):
        self.obj = obj

    def __lt__(self, other):
        try:
            return self.obj < other.obj
        except TypeError:
            return ((str(type(self.obj)), id(self.obj)) < (str(type(other.obj)), id(other.obj)))


def _mec_pp_safe_tuple(t):
    return _mec_pp_SafeKey(t[0]), _mec_pp_SafeKey(t[1])


def _mec_pp_recursion(obj):
    return "<Recursion on " + type(obj).__name__ + " with id=" + str(id(obj)) + ">"


class _mec_pp_Out:
    def __init__(self):
        self.parts = []

    def write(self, s):
        self.parts.append(s)

    def getvalue(self):
        return "".join(self.parts)


class _mec_pp_PrettyPrinter:
    def __init__(self, indent=1, width=80, depth=None, stream=None, *, compact=False, sort_dicts=True,
                 underscore_numbers=False):
        indent = int(indent)
        width = int(width)
        if indent < 0:
            raise ValueError("indent must be >= 0")
        if depth is not None and depth <= 0:
            raise ValueError("depth must be > 0")
        if not width:
            raise ValueError("width must be != 0")
        self._depth = depth
        self._indent_per_level = indent
        self._width = width
        self._stream = stream
        self._compact = bool(compact)
        self._sort_dicts = sort_dicts
        self._underscore_numbers = underscore_numbers

    def pprint(self, obj):
        if self._stream is not None:
            self._stream.write(self.pformat(obj) + "\n")
        else:
            print(self.pformat(obj))

    def pformat(self, obj):
        out = _mec_pp_Out()
        self._format(obj, out, 0, 0, {}, 0)
        return out.getvalue()

    def isrecursive(self, obj):
        return self.format(obj, {}, 0, 0)[2]

    def isreadable(self, obj):
        s, readable, recursive = self.format(obj, {}, 0, 0)
        return readable and not recursive

    def _format(self, obj, stream, indent, allowance, context, level):
        objid = id(obj)
        if objid in context:
            stream.write(_mec_pp_recursion(obj))
            return
        rep = self._repr(obj, context, level)
        max_width = self._width - indent - allowance
        if len(rep) > max_width:
            p = _mec_pp_dispatch(obj)
            if p is not None:
                context[objid] = 1
                p(self, obj, stream, indent, allowance, context, level + 1)
                del context[objid]
                return
        stream.write(rep)

    def _pprint_dict(self, obj, stream, indent, allowance, context, level):
        stream.write("{")
        if self._indent_per_level > 1:
            stream.write((self._indent_per_level - 1) * " ")
        if len(obj):
            items = sorted(obj.items(), key=_mec_pp_safe_tuple) if self._sort_dicts else list(obj.items())
            self._format_dict_items(items, stream, indent, allowance + 1, context, level)
        stream.write("}")

    def _pprint_list(self, obj, stream, indent, allowance, context, level):
        stream.write("[")
        self._format_items(obj, stream, indent, allowance + 1, context, level)
        stream.write("]")

    def _pprint_tuple(self, obj, stream, indent, allowance, context, level):
        stream.write("(")
        endchar = ",)" if len(obj) == 1 else ")"
        self._format_items(obj, stream, indent, allowance + len(endchar), context, level)
        stream.write(endchar)

    def _pprint_set(self, obj, stream, indent, allowance, context, level):
        if not len(obj):
            stream.write(repr(obj))
            return
        if type(obj) is set:
            stream.write("{")
            endchar = "}"
        else:
            stream.write(type(obj).__name__ + "({")
            endchar = "})"
            indent += len(type(obj).__name__) + 1
        items = sorted(obj, key=_mec_pp_SafeKey)
        self._format_items(items, stream, indent, allowance + len(endchar), context, level)
        stream.write(endchar)

    def _pprint_str(self, obj, stream, indent, allowance, context, level):
        if not len(obj):
            stream.write(repr(obj))
            return
        chunks = []
        lines = obj.splitlines(True)
        if level == 1:
            indent += 1
            allowance += 1
        max_width1 = max_width = self._width - indent
        rep = ""
        for i, line in enumerate(lines):
            rep = repr(line)
            if i == len(lines) - 1:
                max_width1 -= allowance
            if len(rep) <= max_width1:
                chunks.append(rep)
            else:
                parts = _mec_pp_words(line)
                max_width2 = max_width
                current = ""
                for j, part in enumerate(parts):
                    candidate = current + part
                    if j == len(parts) - 1 and i == len(lines) - 1:
                        max_width2 -= allowance
                    if len(repr(candidate)) > max_width2:
                        if current:
                            chunks.append(repr(current))
                        current = part
                    else:
                        current = candidate
                if current:
                    chunks.append(repr(current))
        if len(chunks) == 1:
            stream.write(rep)
            return
        if level == 1:
            stream.write("(")
        for i, rep in enumerate(chunks):
            if i > 0:
                stream.write("\n" + " " * indent)
            stream.write(rep)
        if level == 1:
            stream.write(")")

    def _format_dict_items(self, items, stream, indent, allowance, context, level):
        indent += self._indent_per_level
        delimnl = ",\n" + " " * indent
        last_index = len(items) - 1
        for i, kv in enumerate(items):
            last = i == last_index
            rep = self._repr(kv[0], context, level)
            stream.write(rep)
            stream.write(": ")
            self._format(kv[1], stream, indent + len(rep) + 2, allowance if last else 1, context, level)
            if not last:
                stream.write(delimnl)

    def _format_items(self, items, stream, indent, allowance, context, level):
        indent += self._indent_per_level
        if self._indent_per_level > 1:
            stream.write((self._indent_per_level - 1) * " ")
        delimnl = ",\n" + " " * indent
        delim = ""
        width = max_width = self._width - indent + 1
        items = list(items)
        n = len(items)
        for k in range(n):
            ent = items[k]
            last = k == n - 1
            if last:
                max_width -= allowance
                width -= allowance
            if self._compact:
                rep = self._repr(ent, context, level)
                w = len(rep) + 2
                if width < w:
                    width = max_width
                    if delim:
                        delim = delimnl
                if width >= w:
                    width -= w
                    stream.write(delim)
                    delim = ", "
                    stream.write(rep)
                    continue
            stream.write(delim)
            delim = delimnl
            self._format(ent, stream, indent, allowance if last else 1, context, level)

    def _repr(self, obj, context, level):
        return self.format(obj, dict(context), self._depth, level)[0]

    # (repr, readable, recursive) of obj, its containers' items formatted the
    # same way to maxlevels levels.
    def format(self, obj, context, maxlevels, level):
        typ = type(obj)
        if typ in (str, bytes, bytearray, float, complex, bool, type(None)):
            return repr(obj), True, False
        if typ is int:
            if self._underscore_numbers:
                return f"{obj:_d}", True, False
            return repr(obj), True, False
        if typ is dict:
            if not obj:
                return "{}", True, False
            objid = id(obj)
            if maxlevels and level >= maxlevels:
                return "{...}", False, objid in context
            if objid in context:
                return _mec_pp_recursion(obj), False, True
            context[objid] = 1
            readable = True
            recursive = False
            comps = []
            level += 1
            items = sorted(obj.items(), key=_mec_pp_safe_tuple) if self._sort_dicts else obj.items()
            for k, v in items:
                krepr, kreadable, krecur = self.format(k, context, maxlevels, level)
                vrepr, vreadable, vrecur = self.format(v, context, maxlevels, level)
                comps.append(krepr + ": " + vrepr)
                readable = readable and kreadable and vreadable
                if krecur or vrecur:
                    recursive = True
            del context[objid]
            return "{" + ", ".join(comps) + "}", readable, recursive
        if typ is list or typ is tuple:
            if typ is list:
                if not obj:
                    return "[]", True, False
                open_, close = "[", "]"
            elif len(obj) == 1:
                open_, close = "(", ",)"
            else:
                if not obj:
                    return "()", True, False
                open_, close = "(", ")"
            objid = id(obj)
            if maxlevels and level >= maxlevels:
                return open_ + "..." + close, False, objid in context
            if objid in context:
                return _mec_pp_recursion(obj), False, True
            context[objid] = 1
            readable = True
            recursive = False
            comps = []
            level += 1
            for o in obj:
                orepr, oreadable, orecur = self.format(o, context, maxlevels, level)
                comps.append(orepr)
                if not oreadable:
                    readable = False
                if orecur:
                    recursive = True
            del context[objid]
            return open_ + ", ".join(comps) + close, readable, recursive
        rep = repr(obj)
        return rep, bool(rep) and not rep.startswith("<"), False


# The alternating runs of non-space and space characters of s, as
# re.findall(r'\S*\s*', s) gives them without its final empty match.
def _mec_pp_words(s):
    parts = []
    i = 0
    n = len(s)
    while i < n:
        j = i
        while j < n and not s[j].isspace():
            j += 1
        while j < n and s[j].isspace():
            j += 1
        parts.append(s[i:j])
        i = j
    return parts


def _mec_pp_dispatch(obj):
    t = type(obj)
    if t is dict:
        return _mec_pp_PrettyPrinter._pprint_dict
    if t is list:
        return _mec_pp_PrettyPrinter._pprint_list
    if t is tuple:
        return _mec_pp_PrettyPrinter._pprint_tuple
    if t is set or t is frozenset:
        return _mec_pp_PrettyPrinter._pprint_set
    if t is str:
        return _mec_pp_PrettyPrinter._pprint_str
    return None


def _mec_pp_pprint(obj, stream=None, indent=1, width=80, depth=None, *, compact=False, sort_dicts=True,
                   underscore_numbers=False):
    _mec_pp_PrettyPrinter(stream=stream, indent=indent, width=width, depth=depth, compact=compact,
                          sort_dicts=sort_dicts, underscore_numbers=underscore_numbers).pprint(obj)


def _mec_pp_pformat(obj, indent=1, width=80, depth=None, *, compact=False, sort_dicts=True,
                    underscore_numbers=False):
    return _mec_pp_PrettyPrinter(indent=indent, width=width, depth=depth, compact=compact,
                                 sort_dicts=sort_dicts, underscore_numbers=underscore_numbers).pformat(obj)


def _mec_pp_pp(obj, *args, sort_dicts=False, **kwargs):
    _mec_pp_pprint(obj, *args, sort_dicts=sort_dicts, **kwargs)


def _mec_pp_saferepr(obj):
    return _mec_pp_PrettyPrinter().format(obj, {}, None, 0)[0]


def _mec_pp_isreadable(obj):
    return _mec_pp_PrettyPrinter().isreadable(obj)


def _mec_pp_isrecursive(obj):
    return _mec_pp_PrettyPrinter().isrecursive(obj)


_mec_pp_PrettyPrinter.__name__ = "PrettyPrinter"
_mec_pp_PrettyPrinter.__qualname__ = "PrettyPrinter"
_mec_pp_PrettyPrinter.__module__ = "pprint"
_mec_pp_pprint.__name__ = "pprint"
_mec_pp_pformat.__name__ = "pformat"
_mec_pp_pp.__name__ = "pp"
_mec_pp_saferepr.__name__ = "saferepr"
_mec_pp_isreadable.__name__ = "isreadable"
_mec_pp_isrecursive.__name__ = "isrecursive"


class _mec_pp_module:
    __all__ = ["pprint", "pformat", "isreadable", "isrecursive", "saferepr", "PrettyPrinter", "pp"]
    PrettyPrinter = _mec_pp_PrettyPrinter
    pprint = staticmethod(_mec_pp_pprint)
    pformat = staticmethod(_mec_pp_pformat)
    pp = staticmethod(_mec_pp_pp)
    saferepr = staticmethod(_mec_pp_saferepr)
    isreadable = staticmethod(_mec_pp_isreadable)
    isrecursive = staticmethod(_mec_pp_isrecursive)

    def __repr__(self):
        return "<module 'pprint'>"


_mec_mod_pprint = type("module", (_mec_pp_module,), {})()
_mec_mod_pprint.__name__ = "pprint"
