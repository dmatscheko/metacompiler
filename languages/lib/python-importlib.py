# python-importlib.py -- the __import__ builtin and the importlib module,
# written in Python and compiled with a program that names __import__ or
# imports importlib (core.importFile in both python grammars), so one
# implementation serves all three engines.
#
# Modules are bound when the program is compiled, so a name only known at run
# time can reach a module only if it is already compiled in: this file imports
# every module the runtime offers, and __import__ answers one of them. Helpers
# carry the _mec_il prefix; the module object is an instance of a class named
# "module", as type(importlib).__name__ is in CPython. The error texts are
# CPython 3.14's importlib._bootstrap.

import abc as _mec_il_abc
import builtins as _mec_il_builtins
import collections as _mec_il_collections
import copy as _mec_il_copy
import dataclasses as _mec_il_dataclasses
import functools as _mec_il_functools
import itertools as _mec_il_itertools
import json as _mec_il_json
import math as _mec_il_math
import os as _mec_il_os
import random as _mec_il_random
import re as _mec_il_re
import string as _mec_il_string
import sys as _mec_il_sys
import time as _mec_il_time
import typing as _mec_il_typing

_mec_il_mods = {"builtins": _mec_il_builtins, "sys": _mec_il_sys, "math": _mec_il_math,
                "re": _mec_il_re, "typing": _mec_il_typing, "time": _mec_il_time, "abc": _mec_il_abc,
                "collections": _mec_il_collections, "copy": _mec_il_copy,
                "dataclasses": _mec_il_dataclasses, "functools": _mec_il_functools,
                "itertools": _mec_il_itertools, "json": _mec_il_json, "os": _mec_il_os,
                "random": _mec_il_random, "string": _mec_il_string}


def __import__(name, globals=None, locals=None, fromlist=(), level=0):
    if not isinstance(name, str):
        raise TypeError("module name must be a string")
    if level < 0:
        raise ValueError("level must be >= 0")
    if level > 0:
        raise ImportError("attempted relative import with no known parent package")
    if name == "":
        raise ValueError("Empty module name")
    parts = name.split(".")
    if parts[0] not in _mec_il_mods:
        raise ModuleNotFoundError("No module named '" + parts[0] + "'")
    top = _mec_il_mods[parts[0]]
    m = top
    for i in range(1, len(parts)):
        sub = getattr(m, parts[i], None)
        if type(sub).__name__ != "module":
            raise ModuleNotFoundError("No module named '" + ".".join(parts[:i + 1]) + "'; '" +
                                      ".".join(parts[:i]) + "' is not a package")
        m = sub
    return m if fromlist else top


class _mec_il_module:
    __all__ = ["import_module"]

    @staticmethod
    def import_module(name, package=None):
        if name.startswith("."):
            if not package:
                raise TypeError("the 'package' argument is required to perform a relative import for " +
                                repr(name))
            level = len(name) - len(name.lstrip("."))
            base = package.rsplit(".", level - 1)[0]
            name = base + "." + name[level:] if name[level:] else base
        return __import__(name, None, None, ["__name__"])

    def __repr__(self):
        return "<module 'importlib' (frozen)>"


_mec_mod_importlib = type("module", (_mec_il_module,), {})()
_mec_mod_importlib.__name__ = "importlib"
