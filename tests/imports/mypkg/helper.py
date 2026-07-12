# A project module imported by tests/python-test-multifile.py (via -i tests/imports).
# Its top level executes at import time, like real Python.

module_marker = "helper loaded"


def triple(n):
    return n * 3


def label(prefix, value):
    return f"{prefix}={value}"
