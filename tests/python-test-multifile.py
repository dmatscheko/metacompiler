# Multi-file Python test: mypkg.helper lives in tests/imports/mypkg/helper.py and
# is found via the -i include root (mec -i tests/imports ...). The imported file
# is parsed with the same grammar and its top level executes at import time.
# Deviation from real Python: the module's names bind directly (flat, no module
# object), so the from-import style is the natural fit. math is a builtin no-op.
import math
from mypkg.helper import triple, label

fails = 0

def check(name, got, want):
    global fails
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails = fails + 1

check("imported def", triple(14), 42)
check("imported f-string def", label("x", 7), "x=7")
check("module top level ran", module_marker, "helper loaded")

if fails == 0:
    print("python multifile test passed")
exit(fails)
