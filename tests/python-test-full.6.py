# Full-syntax test: Python (3.12 core grammar) - part 6, section 183: a source
# file with CRLF line endings. Every line of this file ends in CRLF, which is
# the point of it - keep it that way; see tests/python-test-full.py for the
# conventions.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 183: CRLF line endings =====
# A carriage return before a newline is part of the line ending, not of the
# line: blocks, blank and comment lines, bracketed and backslash continuations
# all read as with LF, and inside a literal CPython's universal-newline source
# reading makes "\n" of it. Rows from CPython 3.14.
def s183():
    xs = [1,
          2]  # a comment

    if xs:
        y = xs[0] + \
            xs[1]
    t = """a
b"""
    r = r'''c
d'''
    s = "e\
f"
    check("crlf1", y == 3 and t == "a\nb" and r == "c\nd" and s == "ef" and list(b"""p
q""") == [112, 10, 113] and len(f"""{y}
""") == 2)

def main():
    s183() # SECTION-CALL 183
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
