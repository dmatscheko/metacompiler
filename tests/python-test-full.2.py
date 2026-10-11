# Full-syntax test: Python (3.12 core grammar) - part 2 of 2, sections 90-91.
#
# THE PYTHON RATCHET IS TWO FILES, so that neither -frozen interpreter leg comes
# near the IR interpreter's step brake (-max-steps, 100M per call). Sections 90-91
# build ints of thousands of digits, which the interpreter's arbitrary precision
# arithmetic makes the costliest rows of all, so they have a file of their own:
#
#   tests/python-test-full.py     every other section
#   tests/python-test-full.2.py   sections 90-91
#
# The section NUMBERS are global, the two files together run the assertions, and
# ./test.sh --full reports them as the units python and python2. Add a new section
# to tests/python-test-full.py, and keep the numbers globally unique across the two.
#
# This file belongs to the SECOND test group (./test.sh --full): it is NOT part
# of the default matrix. The goal of the metacompiler is to support the full
# languages; this file is the ratchet that measures how far the python grammars
# are. It walks the whole practical Python 3.12 syntax, one self-contained
# SECTION per language area. The --full runner runs the file, and whenever a
# grammar aborts it removes the section around the error and retries - so the
# report lists every unsupported section, not just the first.
#
# Conventions (shared by every *-test-full.* file):
#   - prologue (before the first SECTION marker): the check helper only
#   - each section: '# ===== SECTION <nn>: <name> =====', top-level,
#     self-contained, no references to other sections
#   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
#     and prints the summary line 'full: <checks> checks, <failures> failures'
#   - main() returns the failure count (exit 0 == full support, verified)
#
# Deliberately out of scope (not syntax, or unrunnable in this harness):
# metaclasses, eval/exec, __slots__-free
# introspection helpers, and builtins beyond the small set the feature test
# already leans on plus type/str/int/isinstance and the Exception hierarchy
# (ExceptionGroup appears once: except* cannot be exercised without it).
# Parent methods are therefore called explicitly (Base.m(self)) instead of via
# super(), and property/staticmethod/classmethod are replaced by a hand-rolled
# descriptor and plain functions. The ONE stdlib module used is `re` (section 22),
# which the shared regular-expression engine backs. Complex literals ARE included: they are core
# literals needing no library. Async SYNTAX is covered; running it needs an
# event loop, so async functions are only defined and inspected, never awaited.
#
# Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
# code), organized after the Python 3.12 language reference with the ANTLR
# grammars-v4 Python grammar as a coverage checklist.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 90: int ** int at a large exponent =====
# An int raised to a non-negative int is an exact int whatever the exponent's
# size; no exponent turns it into a float. Rows from CPython 3.14.
def s90():
    v = [10, 4097, 5000, 2, 7, 3]
    p = v[3] ** v[1]
    check("pwb1", p % 1000003 == 833217 and type(p) is int and p == 2 ** 4097)
    check("pwb2", pow(v[4], v[1]) % 1009 == 783 and (-2) ** v[1] == -p)
    check("pwb3", len(str(p)) == 1234 and str(p)[-6:] == "380672")
    x = v[3]
    x **= 4100
    check("pwb4", x == p * 8 and x % 997 == 12)

# ===== SECTION 91: the 4300-digit limit on int <-> str =====
# CPython refuses to render an int of more than sys.get_int_max_str_digits()
# decimal digits (4300 by default; the sign is not a digit) and to parse one from
# a str in a base that is not a power of two. The text is validated before its
# length is judged. Rows from CPython 3.14.
def s91():
    ok = 10 ** 4299
    over = ok * 10
    check("isd1", len(str(ok)) == 4300 and len(repr(-ok)) == 4301 and f"{ok}"[:3] == "100")
    msgs = []
    for f in [str, repr, lambda v: f"{v}", lambda v: "%d" % v, lambda v: format(v, ","), lambda v: str([v])]:
        for v in [over, -over]:
            try:
                f(v)
            except ValueError as e:
                msgs.append(str(e))
    lim = "Exceeds the limit (4300 digits) for integer string conversion"
    tail = "; use sys.set_int_max_str_digits() to increase the limit"
    check("isd2", msgs == [lim + tail] * 12)
    check("isd3", over > ok and over - ok * 9 == ok and len(str(over - 1)) == 4300)
    msgs = []
    for s, b in [("1" * 4301, 10), ("-" + "0" * 4301, 10), (" 1" + "_0" * 4300 + " ", 0), ("0" * 4400, 36), ("1x" + "1" * 4400, 10)]:
        try:
            int(s, b)
        except ValueError as e:
            msgs.append(str(e) if str(e).startswith("Exceeds") else str(e)[:50])
    check("isd4", msgs[0:4] == [lim + ": value has " + n + " digits" + tail for n in ["4301", "4301", "4301", "4400"]]
          and msgs[4] == "invalid literal for int() with base 10: '1x1111111")
    check("isd5", int("0" * 4300) == 0 and int("0_" * 2200 + "0") == 0 and int("0" * 5000, 16) == 0 and int("0" * 4400, 32) == 0)

def main():
    s90() # SECTION-CALL 90
    s91() # SECTION-CALL 91
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
