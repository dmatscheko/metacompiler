# python-string.py -- the string module, written in Python and compiled with a
# program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_string), never as
# a flat name, so `import string` cannot replace a program's own digits or
# Template; helpers carry the _mec_st prefix. The module object is an instance
# of a class named "module", as type(string).__name__ is in CPython.
#
# The constants, capwords and Template are Lib/string/__init__.py's, error texts
# included; Template's placeholder scan is written out by hand rather than with
# the re module. Formatter is not offered.

_mec_st_ident_start = "_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
_mec_st_ident_rest = _mec_st_ident_start + "0123456789"


# The placeholder at s[i] == delimiter: (kind, name, end), kind one of
# "escaped", "named", "braced" or "invalid".
def _mec_st_scan(s, i, delim):
    j = i + len(delim)
    if s[j:j + len(delim)] == delim:
        return "escaped", None, j + len(delim)
    braced = s[j:j + 1] == "{"
    k = j + 1 if braced else j
    if k < len(s) and s[k] in _mec_st_ident_start:
        e = k + 1
        while e < len(s) and s[e] in _mec_st_ident_rest:
            e += 1
        if not braced:
            return "named", s[k:e], e
        if s[e:e + 1] == "}":
            return "braced", s[k:e], e + 1
    return "invalid", None, i


# Where an invalid placeholder is reported: just past its delimiter, the column
# counted from 0 as Template._invalid counts it.
def _mec_st_lineno(s, i):
    lines = s[:i].split("\n")
    return len(lines), len(lines[-1])


def _mec_st_substitute(tmpl, mapping, safe):
    s = tmpl.template
    delim = tmpl.delimiter
    out = []
    i = 0
    while True:
        j = s.find(delim, i)
        if j < 0:
            out.append(s[i:])
            return "".join(out)
        out.append(s[i:j])
        kind, name, end = _mec_st_scan(s, j, delim)
        if kind == "escaped":
            out.append(delim)
        elif kind == "invalid":
            if not safe:
                line, col = _mec_st_lineno(s, j + len(delim))
                raise ValueError("Invalid placeholder in string: line %d, col %d" % (line, col))
            out.append(delim)
            end = j + len(delim)
        elif name in mapping:
            out.append(str(mapping[name]))
        elif safe:
            out.append(s[j:end])
        else:
            raise KeyError(name)
        i = end


class _mec_st_module:
    __all__ = ["ascii_letters", "ascii_lowercase", "ascii_uppercase", "capwords", "digits", "hexdigits",
               "octdigits", "printable", "punctuation", "whitespace", "Template"]

    whitespace = " \t\n\r\x0b\x0c"
    ascii_lowercase = "abcdefghijklmnopqrstuvwxyz"
    ascii_uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    ascii_letters = ascii_lowercase + ascii_uppercase
    digits = "0123456789"
    hexdigits = digits + "abcdef" + "ABCDEF"
    octdigits = "01234567"
    punctuation = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
    printable = digits + ascii_letters + punctuation + whitespace

    @staticmethod
    def capwords(s, sep=None):
        return (sep or " ").join(map(str.capitalize, s.split(sep)))

    class Template:
        delimiter = "$"

        def __init__(self, template):
            self.template = template

        def substitute(self, mapping=None, /, **kws):
            return _mec_st_substitute(self, _mec_st_merge(mapping, kws), False)

        def safe_substitute(self, mapping=None, /, **kws):
            return _mec_st_substitute(self, _mec_st_merge(mapping, kws), True)

        def get_identifiers(self):
            s = self.template
            ids = []
            i = s.find(self.delimiter)
            while i >= 0:
                kind, name, end = _mec_st_scan(s, i, self.delimiter)
                if name is not None and name not in ids:
                    ids.append(name)
                i = s.find(self.delimiter, max(end, i + 1))
            return ids

        def is_valid(self):
            s = self.template
            i = s.find(self.delimiter)
            while i >= 0:
                kind, name, end = _mec_st_scan(s, i, self.delimiter)
                if kind == "invalid":
                    return False
                i = s.find(self.delimiter, end)
            return True

    def __repr__(self):
        return "<module 'string'>"


def _mec_st_merge(mapping, kws):
    if mapping is None:
        return kws
    if kws:
        merged = dict(mapping)
        merged.update(kws)
        return merged
    return mapping


_mec_mod_string = type("module", (_mec_st_module,), {})()
_mec_mod_string.__name__ = "string"
