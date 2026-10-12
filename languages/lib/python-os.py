# python-os.py -- the os module and its os.path submodule, written in Python
# and compiled with a program that imports either (core.importFile in both
# python grammars), so one implementation serves all three engines.
#
# Every public name lives on a module object only (_mec_mod_os, and its path
# attribute), never as a flat name, so `import os` cannot replace a program's
# own join or sep; helpers carry the _mec_os prefix. Each module object is an
# instance of a class named "module", as type(os).__name__ is in CPython.
#
# os.path is Lib/posixpath.py's string arithmetic: join, split, basename,
# dirname, splitext, splitdrive, isabs, normpath and commonprefix, error texts
# included. Nothing here touches the file system or the process environment,
# so exists, abspath, getcwd and environ are not offered.


# os.fspath's check, which every os.path function makes first.
def _mec_os_fspath(p):
    if not isinstance(p, str):
        raise TypeError("expected str, bytes or os.PathLike object, not " + type(p).__name__)
    return p


class _mec_os_path_module:
    __all__ = ["basename", "commonprefix", "dirname", "isabs", "join", "normpath", "split", "splitdrive",
               "splitext", "altsep", "curdir", "extsep", "pardir", "pathsep", "sep"]

    sep = "/"
    altsep = None
    curdir = "."
    pardir = ".."
    extsep = "."
    pathsep = ":"

    @staticmethod
    def join(a, *p):
        path = _mec_os_fspath(a)
        for b in p:
            if not isinstance(b, str):
                raise TypeError("join() argument must be str, bytes, or os.PathLike object, not '" + type(b).__name__ + "'")
            if b.startswith("/") or not path:
                path = b
            elif path.endswith("/"):
                path += b
            else:
                path += "/" + b
        return path

    @staticmethod
    def split(p):
        p = _mec_os_fspath(p)
        i = p.rfind("/") + 1
        head, tail = p[:i], p[i:]
        if head and head != "/" * len(head):
            head = head.rstrip("/")
        return head, tail

    @staticmethod
    def basename(p):
        p = _mec_os_fspath(p)
        return p[p.rfind("/") + 1:]

    @staticmethod
    def dirname(p):
        p = _mec_os_fspath(p)
        head = p[:p.rfind("/") + 1]
        if head and head != "/" * len(head):
            head = head.rstrip("/")
        return head

    @staticmethod
    def splitext(p):
        p = _mec_os_fspath(p)
        sep_index = p.rfind("/")
        dot_index = p.rfind(".")
        if dot_index > sep_index:
            i = sep_index + 1
            while i < dot_index:
                if p[i] != ".":
                    return p[:dot_index], p[dot_index:]
                i += 1
        return p, ""

    @staticmethod
    def splitdrive(p):
        return "", _mec_os_fspath(p)

    @staticmethod
    def isabs(s):
        return _mec_os_fspath(s).startswith("/")

    @staticmethod
    def normpath(path):
        if not isinstance(path, str):
            raise TypeError("_path_normpath: path should be string, bytes or os.PathLike, not " + type(path).__name__)
        if not path:
            return "."
        initial = 0
        if path.startswith("/"):
            initial = 2 if path.startswith("//") and not path.startswith("///") else 1
        comps = []
        for comp in path.split("/"):
            if comp == "" or comp == ".":
                continue
            if comp != ".." or (not initial and not comps) or (comps and comps[-1] == ".."):
                comps.append(comp)
            elif comps:
                comps.pop()
        return ("/" * initial + "/".join(comps)) or "."

    @staticmethod
    def commonprefix(m):
        if not m:
            return ""
        s1 = min(m)
        s2 = max(m)
        for i, c in enumerate(s1):
            if c != s2[i]:
                return s1[:i]
        return s1

    def __repr__(self):
        return "<module 'posixpath' (frozen)>"


class _mec_os_module:
    __all__ = ["altsep", "curdir", "extsep", "linesep", "name", "pardir", "path", "pathsep", "sep"]

    name = "posix"
    sep = "/"
    altsep = None
    curdir = "."
    pardir = ".."
    extsep = "."
    pathsep = ":"
    linesep = "\n"

    def __repr__(self):
        return "<module 'os' (frozen)>"


_mec_mod_os = type("module", (_mec_os_module,), {})()
_mec_mod_os.__name__ = "os"
_mec_mod_os.path = type("module", (_mec_os_path_module,), {})()
_mec_mod_os.path.__name__ = "posixpath"
