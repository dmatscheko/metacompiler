# python-textwrap.py -- the textwrap module, written in Python and compiled with
# a program that imports it (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_textwrap);
# helpers carry the _mec_tw prefix. The behaviour is Lib/textwrap.py's (3.14):
# TextWrapper, wrap, fill, shorten, dedent and indent. CPython splits a text
# into chunks with a regular expression that needs look-behind; _mec_tw_chunks
# is that expression's match written out as a scanner.

_mec_tw_ws = "\t\n\x0b\x0c\r "


def _mec_tw_isw(c):
    return c.isalnum() or c == "_"


def _mec_tw_letter(c):
    return _mec_tw_isw(c) and not c.isdecimal()


def _mec_tw_punct(c):
    return _mec_tw_isw(c) or c in "!\"'&.,?"


def _mec_tw_at(text, i):
    return text[i] if 0 <= i < len(text) else ""


# The length of the run of hyphens at i, if two or more of them are followed
# by a word character; else 0.
def _mec_tw_dashes(text, i):
    j = i
    while j < len(text) and text[j] == "-":
        j += 1
    if j - i >= 2 and j < len(text) and _mec_tw_isw(text[j]):
        return j - i
    return 0


# wordsep_re: a run of whitespace; an em-dash (two or more hyphens) between
# words; or a word, which ends at whitespace or the end, before an em-dash, or
# just after a hyphen that sits between letters.
def _mec_tw_chunks(text):
    out = []
    n = len(text)
    i = 0
    while i < n:
        if text[i] in _mec_tw_ws:
            j = i
            while j < n and text[j] in _mec_tw_ws:
                j += 1
        elif i > 0 and _mec_tw_punct(text[i - 1]) and _mec_tw_dashes(text, i):
            j = i + _mec_tw_dashes(text, i)
        else:
            j = i + 1
            while True:
                at = _mec_tw_at
                lt = _mec_tw_letter
                if (at(text, j) == "-"
                        and ((lt(at(text, j - 2)) and lt(at(text, j - 1)))
                             or (lt(at(text, j - 3)) and at(text, j - 2) == "-" and lt(at(text, j - 1))))
                        and lt(at(text, j + 1))
                        and (lt(at(text, j + 2)) or (at(text, j + 2) == "-" and lt(at(text, j + 3))))):
                    j += 1
                    break
                if j >= n or text[j] in _mec_tw_ws:
                    break
                if _mec_tw_punct(text[j - 1]) and _mec_tw_dashes(text, j):
                    break
                j += 1
        out.append(text[i:j])
        i = j
    return out


# wordsep_simple_re: runs of whitespace and runs of anything else.
def _mec_tw_simple(text):
    out = []
    i = 0
    while i < len(text):
        ws = text[i] in _mec_tw_ws
        j = i + 1
        while j < len(text) and (text[j] in _mec_tw_ws) == ws:
            j += 1
        out.append(text[i:j])
        i = j
    return out


def _mec_tw_sentence_end(chunk):
    k = len(chunk)
    if k >= 1 and chunk[k - 1] in "\"'":
        k -= 1
    return k >= 2 and chunk[k - 1] in ".!?" and "a" <= chunk[k - 2] <= "z"


class _mec_tw_TextWrapper:
    def __init__(self, width=70, initial_indent="", subsequent_indent="", expand_tabs=True,
                 replace_whitespace=True, fix_sentence_endings=False, break_long_words=True,
                 drop_whitespace=True, break_on_hyphens=True, tabsize=8, *, max_lines=None,
                 placeholder=" [...]"):
        self.width = width
        self.initial_indent = initial_indent
        self.subsequent_indent = subsequent_indent
        self.expand_tabs = expand_tabs
        self.replace_whitespace = replace_whitespace
        self.fix_sentence_endings = fix_sentence_endings
        self.break_long_words = break_long_words
        self.drop_whitespace = drop_whitespace
        self.break_on_hyphens = break_on_hyphens
        self.tabsize = tabsize
        self.max_lines = max_lines
        self.placeholder = placeholder

    def _munge_whitespace(self, text):
        if self.expand_tabs:
            text = text.expandtabs(self.tabsize)
        if self.replace_whitespace:
            for c in _mec_tw_ws:
                text = text.replace(c, " ")
        return text

    def _split(self, text):
        if self.break_on_hyphens is True:
            return _mec_tw_chunks(text)
        return _mec_tw_simple(text)

    def _fix_sentence_endings(self, chunks):
        i = 0
        while i < len(chunks) - 1:
            if chunks[i + 1] == " " and _mec_tw_sentence_end(chunks[i]):
                chunks[i + 1] = "  "
                i += 2
            else:
                i += 1

    def _handle_long_word(self, reversed_chunks, cur_line, cur_len, width):
        if width < 1:
            space_left = 1
        else:
            space_left = width - cur_len
        if self.break_long_words and space_left > 0:
            end = space_left
            chunk = reversed_chunks[-1]
            if self.break_on_hyphens and len(chunk) > space_left:
                hyphen = chunk.rfind("-", 0, space_left)
                if hyphen > 0 and any(c != "-" for c in chunk[:hyphen]):
                    end = hyphen + 1
            cur_line.append(chunk[:end])
            reversed_chunks[-1] = chunk[end:]
        elif not cur_line:
            cur_line.append(reversed_chunks.pop())

    def _wrap_chunks(self, chunks):
        lines = []
        if self.width <= 0:
            raise ValueError("invalid width %r (must be > 0)" % self.width)
        if self.max_lines is not None:
            if self.max_lines > 1:
                indent = self.subsequent_indent
            else:
                indent = self.initial_indent
            if len(indent) + len(self.placeholder.lstrip()) > self.width:
                raise ValueError("placeholder too large for max width")
        chunks.reverse()
        while chunks:
            cur_line = []
            cur_len = 0
            if lines:
                indent = self.subsequent_indent
            else:
                indent = self.initial_indent
            width = self.width - len(indent)
            if self.drop_whitespace and chunks[-1].strip() == "" and lines:
                del chunks[-1]
            while chunks:
                l = len(chunks[-1])
                if cur_len + l <= width:
                    cur_line.append(chunks.pop())
                    cur_len += l
                else:
                    break
            if chunks and len(chunks[-1]) > width:
                self._handle_long_word(chunks, cur_line, cur_len, width)
                cur_len = sum(map(len, cur_line))
            if self.drop_whitespace and cur_line and cur_line[-1].strip() == "":
                cur_len -= len(cur_line[-1])
                del cur_line[-1]
            if cur_line:
                if (self.max_lines is None or len(lines) + 1 < self.max_lines
                        or (not chunks or self.drop_whitespace and len(chunks) == 1 and not chunks[0].strip())
                        and cur_len <= width):
                    lines.append(indent + "".join(cur_line))
                else:
                    placed = False
                    while cur_line:
                        if cur_line[-1].strip() and cur_len + len(self.placeholder) <= width:
                            cur_line.append(self.placeholder)
                            lines.append(indent + "".join(cur_line))
                            placed = True
                            break
                        cur_len -= len(cur_line[-1])
                        del cur_line[-1]
                    if not placed:
                        if lines:
                            prev_line = lines[-1].rstrip()
                            if len(prev_line) + len(self.placeholder) <= self.width:
                                lines[-1] = prev_line + self.placeholder
                                break
                        lines.append(indent + self.placeholder.lstrip())
                    break
        return lines

    def _split_chunks(self, text):
        return self._split(self._munge_whitespace(text))

    def wrap(self, text):
        chunks = self._split_chunks(text)
        if self.fix_sentence_endings:
            self._fix_sentence_endings(chunks)
        return self._wrap_chunks(chunks)

    def fill(self, text):
        return "\n".join(self.wrap(text))


_mec_tw_TextWrapper.__name__ = "TextWrapper"
_mec_tw_TextWrapper.__qualname__ = "TextWrapper"


class _mec_tw_module:
    __all__ = ["TextWrapper", "wrap", "fill", "dedent", "indent", "shorten"]
    TextWrapper = _mec_tw_TextWrapper

    @staticmethod
    def wrap(text, width=70, **kwargs):
        return _mec_tw_TextWrapper(width=width, **kwargs).wrap(text)

    @staticmethod
    def fill(text, width=70, **kwargs):
        return _mec_tw_TextWrapper(width=width, **kwargs).fill(text)

    @staticmethod
    def shorten(text, width, **kwargs):
        w = _mec_tw_TextWrapper(width=width, max_lines=1, **kwargs)
        return w.fill(" ".join(text.strip().split()))

    # The margin is the longest run of spaces and tabs every non-blank line
    # starts with; a blank line comes out empty.
    @staticmethod
    def dedent(text):
        if not isinstance(text, str):
            raise TypeError("expected str object, not " + repr(type(text).__qualname__))
        lines = text.split("\n")
        non_blank = [l for l in lines if l and not l.isspace()]
        l1 = min(non_blank, default="")
        l2 = max(non_blank, default="")
        margin = 0
        for margin, c in enumerate(l1):
            if c != l2[margin] or c not in " \t":
                break
        return "\n".join([l[margin:] if not l.isspace() else "" for l in lines])

    @staticmethod
    def indent(text, prefix, predicate=None):
        out = []
        for line in text.splitlines(True):
            if (predicate(line) if predicate is not None else not line.isspace()):
                out.append(prefix)
            out.append(line)
        return "".join(out)

    def __repr__(self):
        return "<module 'textwrap'>"


_mec_mod_textwrap = type("module", (_mec_tw_module,), {})()
_mec_mod_textwrap.__name__ = "textwrap"
