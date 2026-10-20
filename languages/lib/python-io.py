# python-io.py -- the io module's in-memory text stream, written in Python and
# compiled with a program that imports it (core.importFile in both python
# grammars), so one implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_io); helpers carry
# the _mec_io prefix. StringIO is CPython's for the default newline="\n" (no
# translation), error texts included. There is no file I/O in this project's
# runtime, so open() and the file classes are absent.


class _mec_io_StringIO:
    def __init__(self, initial_value="", newline="\n"):
        if initial_value is None:
            initial_value = ""
        if not isinstance(initial_value, str):
            raise TypeError("initial_value must be str or None, not " + type(initial_value).__name__)
        self._mec_io_buf = initial_value
        self._mec_io_pos = 0
        self._mec_io_closed = False

    def _mec_io_check(self):
        if self._mec_io_closed:
            raise ValueError("I/O operation on closed file")

    @property
    def closed(self):
        return self._mec_io_closed

    def close(self):
        self._mec_io_closed = True

    def getvalue(self):
        self._mec_io_check()
        return self._mec_io_buf

    # Writing at a position inside the text overwrites it; beyond the end, the
    # gap is filled with NULs.
    def write(self, s):
        self._mec_io_check()
        if not isinstance(s, str):
            raise TypeError("string argument expected, got '" + type(s).__name__ + "'")
        buf = self._mec_io_buf
        pos = self._mec_io_pos
        if pos > len(buf):
            buf = buf + "\0" * (pos - len(buf))
        self._mec_io_buf = buf[:pos] + s + buf[pos + len(s):]
        self._mec_io_pos = pos + len(s)
        return len(s)

    def writelines(self, lines):
        for line in lines:
            self.write(line)

    def read(self, size=-1):
        self._mec_io_check()
        buf = self._mec_io_buf
        pos = min(self._mec_io_pos, len(buf))
        if size is None or size < 0:
            end = len(buf)
        else:
            end = min(len(buf), pos + size)
        self._mec_io_pos = max(self._mec_io_pos, end)
        return buf[pos:end]

    def readline(self, size=-1):
        self._mec_io_check()
        buf = self._mec_io_buf
        pos = min(self._mec_io_pos, len(buf))
        nl = buf.find("\n", pos)
        end = len(buf) if nl < 0 else nl + 1
        if size is not None and size >= 0:
            end = min(end, pos + size)
        self._mec_io_pos = max(self._mec_io_pos, end)
        return buf[pos:end]

    def readlines(self, hint=-1):
        out = []
        total = 0
        while True:
            line = self.readline()
            if not line:
                return out
            out.append(line)
            total += len(line)
            if hint is not None and 0 < hint <= total:
                return out

    def seek(self, pos, whence=0):
        self._mec_io_check()
        if whence == 1:
            if pos != 0:
                raise OSError("Can't do nonzero cur-relative seeks")
            return self._mec_io_pos
        if whence == 2:
            if pos != 0:
                raise OSError("Can't do nonzero end-relative seeks")
            self._mec_io_pos = len(self._mec_io_buf)
            return self._mec_io_pos
        if whence != 0:
            raise ValueError("Invalid whence (" + str(whence) + ", should be 0, 1 or 2)")
        if pos < 0:
            raise ValueError("Negative seek position " + str(pos))
        self._mec_io_pos = pos
        return pos

    def tell(self):
        self._mec_io_check()
        return self._mec_io_pos

    def truncate(self, size=None):
        self._mec_io_check()
        if size is None:
            size = self._mec_io_pos
        if size < 0:
            raise ValueError("Negative size value " + str(size))
        self._mec_io_buf = self._mec_io_buf[:size]
        return size

    def readable(self):
        self._mec_io_check()
        return True

    def writable(self):
        self._mec_io_check()
        return True

    def seekable(self):
        self._mec_io_check()
        return True

    def flush(self):
        pass

    def isatty(self):
        self._mec_io_check()
        return False

    def __iter__(self):
        self._mec_io_check()
        return self

    def __next__(self):
        line = self.readline()
        if not line:
            raise StopIteration
        return line

    def __enter__(self):
        self._mec_io_check()
        return self

    def __exit__(self, *args):
        self.close()


_mec_io_StringIO.__name__ = "StringIO"
_mec_io_StringIO.__qualname__ = "StringIO"
_mec_io_StringIO.__module__ = "_io"


class _mec_io_module:
    __all__ = ["StringIO"]
    StringIO = _mec_io_StringIO

    def __repr__(self):
        return "<module 'io'>"


_mec_mod_io = type("module", (_mec_io_module,), {})()
_mec_mod_io.__name__ = "io"
