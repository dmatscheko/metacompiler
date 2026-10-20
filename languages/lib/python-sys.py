# python-sys.py -- the part of the sys module written in Python, compiled with a
# program that imports sys (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# The module object itself is the runtime's (_mec_mod_sys, with exit and the
# int-digits limit); this file adds its data and the text streams to it.
# Helpers carry the _mec_sys prefix. stdout writes through print; stderr is
# line buffered onto the runtime's diagnostic channel (_mec_eprintln), so a
# trailing fragment without a newline is not written. There is no stdin, and
# argv holds only an empty script name.


class _mec_sys_Stream:
    def __init__(self, name, err):
        self.name = "<" + name + ">"
        self._mec_sys_err = err
        self._mec_sys_buf = ""

    def write(self, s):
        if not isinstance(s, str):
            raise TypeError("write() argument must be str, not " + type(s).__name__)
        if not self._mec_sys_err:
            print(s, end="")
            return len(s)
        buf = self._mec_sys_buf + s
        nl = buf.rfind("\n")
        if nl >= 0:
            for line in buf[:nl].split("\n"):
                _mec_eprintln(line)
            buf = buf[nl + 1:]
        self._mec_sys_buf = buf
        return len(s)

    def writelines(self, lines):
        for line in lines:
            self.write(line)

    def flush(self):
        pass

    def isatty(self):
        return False

    def fileno(self):
        return 2 if self._mec_sys_err else 1

    def writable(self):
        return True

    def readable(self):
        return False

    @property
    def encoding(self):
        return "utf-8"

    def __repr__(self):
        return "<_io.TextIOWrapper name='" + self.name + "' mode='w' encoding='utf-8'>"


class _mec_sys_VersionInfo(tuple):
    @property
    def major(self):
        return self[0]

    @property
    def minor(self):
        return self[1]

    @property
    def micro(self):
        return self[2]

    @property
    def releaselevel(self):
        return self[3]

    @property
    def serial(self):
        return self[4]

    def __repr__(self):
        return ("sys.version_info(major=" + str(self[0]) + ", minor=" + str(self[1]) + ", micro="
                + str(self[2]) + ", releaselevel=" + repr(self[3]) + ", serial=" + str(self[4]) + ")")


_mec_sys_limit = [1000]


def _mec_sys_getrecursionlimit():
    return _mec_sys_limit[0]


def _mec_sys_setrecursionlimit(limit, /):
    if not isinstance(limit, int):
        raise TypeError("'" + type(limit).__name__ + "' object cannot be interpreted as an integer")
    if limit < 1:
        raise ValueError("recursion limit must be greater or equal than 1")
    _mec_sys_limit[0] = limit


_mec_sys_getrecursionlimit.__name__ = "getrecursionlimit"
_mec_sys_setrecursionlimit.__name__ = "setrecursionlimit"

_mec_mod_sys.stdout = _mec_sys_Stream("stdout", False)
_mec_mod_sys.stderr = _mec_sys_Stream("stderr", True)
_mec_mod_sys.__stdout__ = _mec_mod_sys.stdout
_mec_mod_sys.__stderr__ = _mec_mod_sys.stderr
_mec_mod_sys.maxsize = 9223372036854775807
_mec_mod_sys.byteorder = "little"
_mec_mod_sys.argv = [""]
_mec_mod_sys.version_info = _mec_sys_VersionInfo((3, 14, 0, "final", 0))
_mec_mod_sys.version = "3.14.0"
_mec_mod_sys.getrecursionlimit = _mec_sys_getrecursionlimit
_mec_mod_sys.setrecursionlimit = _mec_sys_setrecursionlimit
