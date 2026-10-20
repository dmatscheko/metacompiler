# A project module imported by tests/python-test-multifile.py (via -i tests/imports).
# Its top level executes at import time, like real Python.

module_marker = "helper loaded"


def triple(n):
    return n * 3


def label(prefix, value):
    return f"{prefix}={value}"


# An imported module's __name__ is its dotted name, at its top level and in its
# functions alike, so its __main__ guard does not run on import.
helper_name = __name__


def own_name():
    return __name__


if __name__ == "__main__":
    module_marker = "run as a script"
