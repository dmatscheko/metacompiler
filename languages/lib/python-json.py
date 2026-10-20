# python-json.py -- the json module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_json), never as
# a flat name, so `import json` cannot replace a program's own dumps or loads;
# helpers carry the _mec_js prefix. The module object is an instance of a class
# named "module", as type(json).__name__ is in CPython.
#
# The behaviour is Lib/json's (encoder.py, decoder.py, scanner.py), error texts
# included: dumps with indent, separators, sort_keys, ensure_ascii, allow_nan,
# skipkeys and default; loads with object_hook, object_pairs_hook, parse_float,
# parse_int and parse_constant. dump and load take a file object's write and
# read methods. JSONEncoder and JSONDecoder are the classes cls= names, so a
# subclass overriding default() (or decode()) is how a program extends them.

_mec_js_escapes = {"\\": "\\\\", '"': '\\"', "\b": "\\b", "\f": "\\f", "\n": "\\n", "\r": "\\r", "\t": "\\t"}


# encoder.py's py_encode_basestring(_ascii): the ASCII form escapes everything
# outside ' '..'~', an astral character as its UTF-16 surrogate pair.
def _mec_js_str(s, ascii_only):
    out = ['"']
    for ch in s:
        esc = _mec_js_escapes.get(ch)
        if esc is not None:
            out.append(esc)
            continue
        n = ord(ch)
        if n < 32 or (ascii_only and n > 126):
            if n > 0xFFFF:
                n -= 0x10000
                out.append("\\u%04x\\u%04x" % (0xD800 | (n >> 10), 0xDC00 | (n & 0x3FF)))
            else:
                out.append("\\u%04x" % n)
        else:
            out.append(ch)
    out.append('"')
    return "".join(out)


def _mec_js_float(o, allow_nan):
    if o != o:
        text = "NaN"
    elif o == float("inf"):
        text = "Infinity"
    elif o == float("-inf"):
        text = "-Infinity"
    else:
        return float.__repr__(o)
    if not allow_nan:
        raise ValueError("Out of range float values are not JSON compliant: " + repr(o))
    return text


class _mec_js_Encoder:
    def __init__(self, skipkeys, ensure_ascii, allow_nan, indent, separators, default, sort_keys):
        if indent is not None and not isinstance(indent, str):
            indent = " " * indent
        self.indent = indent
        if separators is not None:
            self.item_sep, self.key_sep = separators
        elif indent is not None:
            self.item_sep, self.key_sep = ",", ": "
        else:
            self.item_sep, self.key_sep = ", ", ": "
        self.skipkeys = skipkeys
        self.ensure_ascii = ensure_ascii
        self.allow_nan = allow_nan
        self.default = default
        self.sort_keys = sort_keys
        self.markers = []

    def key(self, k):
        if isinstance(k, str):
            return k
        if isinstance(k, float):
            return _mec_js_float(k, self.allow_nan)
        if k is True:
            return "true"
        if k is False:
            return "false"
        if k is None:
            return "null"
        if isinstance(k, int):
            return int.__repr__(k)
        if self.skipkeys:
            return None
        raise TypeError("keys must be str, int, float, bool or None, not " + type(k).__name__)

    def enter(self, o):
        for m in self.markers:
            if m is o:
                raise ValueError("Circular reference detected")
        self.markers.append(o)

    def encode(self, o, out, level):
        if isinstance(o, str):
            out.append(_mec_js_str(o, self.ensure_ascii))
        elif o is None:
            out.append("null")
        elif o is True:
            out.append("true")
        elif o is False:
            out.append("false")
        elif isinstance(o, int):
            out.append(int.__repr__(o))
        elif isinstance(o, float):
            out.append(_mec_js_float(o, self.allow_nan))
        elif isinstance(o, (list, tuple)):
            self.encode_list(o, out, level)
        elif isinstance(o, dict):
            self.encode_dict(o, out, level)
        else:
            if self.default is None:
                raise TypeError("Object of type " + type(o).__name__ + " is not JSON serializable")
            self.enter(o)
            self.encode(self.default(o), out, level)
            self.markers.pop()

    # The separator before the first item and between items, and the newline
    # that closes the container, for nesting depth `level`.
    def layout(self, level):
        if self.indent is None:
            return "", self.item_sep, ""
        inner = "\n" + self.indent * (level + 1)
        return inner, self.item_sep + inner, "\n" + self.indent * level

    def encode_list(self, o, out, level):
        if not o:
            out.append("[]")
            return
        self.enter(o)
        first, sep, close = self.layout(level)
        out.append("[" + first)
        for i, v in enumerate(o):
            if i:
                out.append(sep)
            self.encode(v, out, level + 1)
        out.append(close + "]")
        self.markers.pop()

    def encode_dict(self, o, out, level):
        if not o:
            out.append("{}")
            return
        self.enter(o)
        first, sep, close = self.layout(level)
        items = list(o.items())
        if self.sort_keys:
            items = sorted(items, key=_mec_js_first)
        out.append("{" + first)
        n = 0
        for k, v in items:
            k = self.key(k)
            if k is None:
                continue
            if n:
                out.append(sep)
            n += 1
            out.append(_mec_js_str(k, self.ensure_ascii) + self.key_sep)
            self.encode(v, out, level + 1)
        out.append(close + "}")
        self.markers.pop()


def _mec_js_first(pair):
    return pair[0]


_mec_js_ws = " \t\n\r"
_mec_js_unescape = {'"': '"', "\\": "\\", "/": "/", "b": "\b", "f": "\f", "n": "\n", "r": "\r", "t": "\t"}
_mec_js_digits = "0123456789"
_mec_js_hex = "0123456789abcdefABCDEF"


class _mec_js_Decoder:
    def __init__(self, s, object_hook, object_pairs_hook, parse_float, parse_int, parse_constant):
        self.s = s
        self.n = len(s)
        self.object_hook = object_hook
        self.object_pairs_hook = object_pairs_hook
        self.parse_float = parse_float
        self.parse_int = parse_int
        self.parse_constant = parse_constant

    def fail(self, msg, pos):
        raise _mec_js_module.JSONDecodeError(msg, self.s, pos)

    def skip(self, i):
        while i < self.n and self.s[i] in _mec_js_ws:
            i += 1
        return i

    def at(self, i, word):
        return self.s[i:i + len(word)] == word

    def value(self, i):
        s = self.s
        if i >= self.n:
            self.fail("Expecting value", i)
        ch = s[i]
        if ch == '"':
            return self.string(i + 1)
        if ch == "{":
            return self.object(i + 1)
        if ch == "[":
            return self.array(i + 1)
        if ch == "n" and self.at(i, "null"):
            return None, i + 4
        if ch == "t" and self.at(i, "true"):
            return True, i + 4
        if ch == "f" and self.at(i, "false"):
            return False, i + 5
        if ch == "N" and self.at(i, "NaN"):
            return self.constant("NaN"), i + 3
        if ch == "I" and self.at(i, "Infinity"):
            return self.constant("Infinity"), i + 8
        if ch == "-" and self.at(i, "-Infinity"):
            return self.constant("-Infinity"), i + 9
        return self.number(i)

    def constant(self, name):
        if self.parse_constant is not None:
            return self.parse_constant(name)
        if name == "NaN":
            return float("nan")
        return float(name.replace("Infinity", "inf"))

    # scanner.py's NUMBER_RE: -?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][-+]?[0-9]+)?
    def number(self, i):
        s = self.s
        n = self.n
        j = i
        if j < n and s[j] == "-":
            j += 1
        if j < n and s[j] == "0":
            j += 1
        elif j < n and s[j] in "123456789":
            while j < n and s[j] in _mec_js_digits:
                j += 1
        else:
            self.fail("Expecting value", i)
        integral = True
        if j + 1 < n and s[j] == "." and s[j + 1] in _mec_js_digits:
            integral = False
            j += 2
            while j < n and s[j] in _mec_js_digits:
                j += 1
        if j < n and s[j] in "eE":
            k = j + 1
            if k < n and s[k] in "+-":
                k += 1
            if k < n and s[k] in _mec_js_digits:
                integral = False
                j = k
                while j < n and s[j] in _mec_js_digits:
                    j += 1
        text = s[i:j]
        if integral:
            return (int(text) if self.parse_int is None else self.parse_int(text)), j
        return (float(text) if self.parse_float is None else self.parse_float(text)), j

    def string(self, i):
        s = self.s
        n = self.n
        begin = i - 1
        out = []
        while True:
            j = i
            while j < n and s[j] != '"' and s[j] != "\\" and ord(s[j]) >= 32:
                j += 1
            out.append(s[i:j])
            if j >= n:
                self.fail("Unterminated string starting at", begin)
            ch = s[j]
            if ch == '"':
                return "".join(out), j + 1
            if ch != "\\":
                self.fail("Invalid control character at", j)
            if j + 1 >= n:
                self.fail("Unterminated string starting at", begin)
            esc = s[j + 1]
            if esc != "u":
                rep = _mec_js_unescape.get(esc)
                if rep is None:
                    self.fail("Invalid \\escape", j)
                out.append(rep)
                i = j + 2
                continue
            uni = self.hex4(j + 2, j + 1)
            i = j + 6
            if 0xD800 <= uni <= 0xDBFF and s[i:i + 2] == "\\u":
                lo = self.hex4(i + 2, i + 1)
                if 0xDC00 <= lo <= 0xDFFF:
                    uni = 0x10000 + (((uni - 0xD800) << 10) | (lo - 0xDC00))
                    i += 6
            out.append(chr(uni))

    # The four hex digits at i; an error is reported at the escape's "u", esc.
    def hex4(self, i, esc):
        h = self.s[i:i + 4]
        if len(h) < 4 or h[0] not in _mec_js_hex or h[1] not in _mec_js_hex or h[2] not in _mec_js_hex or h[3] not in _mec_js_hex:
            self.fail("Invalid \\uXXXX escape", esc)
        return int(h, 16)

    def object(self, i):
        s = self.s
        pairs = []
        i = self.skip(i)
        if i < self.n and s[i] == "}":
            return self.finish(pairs), i + 1
        while True:
            if i >= self.n or s[i] != '"':
                self.fail("Expecting property name enclosed in double quotes", i)
            key, i = self.string(i + 1)
            i = self.skip(i)
            if i >= self.n or s[i] != ":":
                self.fail("Expecting ':' delimiter", i)
            i = self.skip(i + 1)
            val, i = self.value(i)
            pairs.append((key, val))
            i = self.skip(i)
            if i < self.n and s[i] == "}":
                return self.finish(pairs), i + 1
            if i >= self.n or s[i] != ",":
                self.fail("Expecting ',' delimiter", i)
            comma = i
            i = self.skip(i + 1)
            if i < self.n and s[i] == "}":
                self.fail("Illegal trailing comma before end of object", comma)

    def finish(self, pairs):
        if self.object_pairs_hook is not None:
            return self.object_pairs_hook(pairs)
        d = dict(pairs)
        if self.object_hook is not None:
            return self.object_hook(d)
        return d

    def array(self, i):
        s = self.s
        out = []
        i = self.skip(i)
        if i < self.n and s[i] == "]":
            return out, i + 1
        while True:
            val, i = self.value(i)
            out.append(val)
            i = self.skip(i)
            if i < self.n and s[i] == "]":
                return out, i + 1
            if i >= self.n or s[i] != ",":
                self.fail("Expecting ',' delimiter", i)
            comma = i
            i = self.skip(i + 1)
            if i < self.n and s[i] == "]":
                self.fail("Illegal trailing comma before end of array", comma)


class _mec_js_module:
    __all__ = ["dump", "dumps", "load", "loads", "JSONDecoder", "JSONDecodeError", "JSONEncoder"]

    class JSONEncoder:
        item_separator = ", "
        key_separator = ": "

        def __init__(self, *, skipkeys=False, ensure_ascii=True, check_circular=True, allow_nan=True,
                     sort_keys=False, indent=None, separators=None, default=None):
            self.skipkeys = skipkeys
            self.ensure_ascii = ensure_ascii
            self.check_circular = check_circular
            self.allow_nan = allow_nan
            self.sort_keys = sort_keys
            self.indent = indent
            if separators is not None:
                self.item_separator, self.key_separator = separators
            elif indent is not None:
                self.item_separator = ","
            if default is not None:
                self.default = default

        def default(self, o):
            raise TypeError("Object of type " + type(o).__name__ + " is not JSON serializable")

        def encode(self, o):
            out = []
            enc = _mec_js_Encoder(self.skipkeys, self.ensure_ascii, self.allow_nan, self.indent,
                                  (self.item_separator, self.key_separator), self.default, self.sort_keys)
            enc.encode(o, out, 0)
            return "".join(out)

        def iterencode(self, o, _one_shot=False):
            return iter([self.encode(o)])

    class JSONDecoder:
        def __init__(self, *, object_hook=None, parse_float=None, parse_int=None, parse_constant=None,
                     strict=True, object_pairs_hook=None):
            self.object_hook = object_hook
            self.parse_float = parse_float
            self.parse_int = parse_int
            self.parse_constant = parse_constant
            self.strict = strict
            self.object_pairs_hook = object_pairs_hook

        def decode(self, s):
            return _mec_js_module.loads(s, object_hook=self.object_hook, parse_float=self.parse_float,
                                        parse_int=self.parse_int, parse_constant=self.parse_constant,
                                        object_pairs_hook=self.object_pairs_hook)

    class JSONDecodeError(ValueError):
        def __init__(self, msg, doc, pos):
            lineno = doc.count("\n", 0, pos) + 1
            colno = pos - doc.rfind("\n", 0, pos)
            ValueError.__init__(self, "%s: line %d column %d (char %d)" % (msg, lineno, colno, pos))
            self.msg = msg
            self.doc = doc
            self.pos = pos
            self.lineno = lineno
            self.colno = colno

    @staticmethod
    def dumps(obj, *, skipkeys=False, ensure_ascii=True, check_circular=True, allow_nan=True, cls=None,
              indent=None, separators=None, default=None, sort_keys=False, **kw):
        if cls is not None:
            return cls(skipkeys=skipkeys, ensure_ascii=ensure_ascii, check_circular=check_circular,
                       allow_nan=allow_nan, indent=indent, separators=separators, default=default,
                       sort_keys=sort_keys, **kw).encode(obj)
        out = []
        enc = _mec_js_Encoder(skipkeys, ensure_ascii, allow_nan, indent, separators, default, sort_keys)
        enc.encode(obj, out, 0)
        return "".join(out)

    @staticmethod
    def dump(obj, fp, *, skipkeys=False, ensure_ascii=True, check_circular=True, allow_nan=True, cls=None,
             indent=None, separators=None, default=None, sort_keys=False, **kw):
        fp.write(_mec_js_module.dumps(obj, skipkeys=skipkeys, ensure_ascii=ensure_ascii, allow_nan=allow_nan,
                                      cls=cls, indent=indent, separators=separators, default=default,
                                      sort_keys=sort_keys, **kw))

    @staticmethod
    def loads(s, *, cls=None, object_hook=None, parse_float=None, parse_int=None, parse_constant=None,
              object_pairs_hook=None, **kw):
        if cls is not None:
            # Only the hooks that were given, as decoder.py's loads passes them.
            hooks = {"object_hook": object_hook, "parse_float": parse_float, "parse_int": parse_int,
                     "parse_constant": parse_constant, "object_pairs_hook": object_pairs_hook}
            for k, v in hooks.items():
                if v is not None:
                    kw[k] = v
            return cls(**kw).decode(s)
        if not isinstance(s, str):
            raise TypeError("the JSON object must be str, bytes or bytearray, not " + type(s).__name__)
        if s.startswith("﻿"):
            raise _mec_js_module.JSONDecodeError("Unexpected UTF-8 BOM (decode using utf-8-sig)", s, 0)
        dec = _mec_js_Decoder(s, object_hook, object_pairs_hook, parse_float, parse_int, parse_constant)
        val, end = dec.value(dec.skip(0))
        end = dec.skip(end)
        if end != len(s):
            dec.fail("Extra data", end)
        return val

    @staticmethod
    def load(fp, *, cls=None, object_hook=None, parse_float=None, parse_int=None, parse_constant=None,
             object_pairs_hook=None, **kw):
        return _mec_js_module.loads(fp.read(), cls=cls, object_hook=object_hook, parse_float=parse_float,
                                    parse_int=parse_int, parse_constant=parse_constant,
                                    object_pairs_hook=object_pairs_hook, **kw)

    def __repr__(self):
        return "<module 'json'>"


_mec_mod_json = type("module", (_mec_js_module,), {})()
_mec_mod_json.__name__ = "json"
