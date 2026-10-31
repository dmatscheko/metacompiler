# Full-syntax test: Python (3.12 core grammar) - part 6, sections 183 and 185: a
# source file with CRLF line endings, which runs its main() indirectly. Every
# line of this file ends in CRLF, which is the point of it - keep it that way;
# see tests/python-test-full.py for the conventions.

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

# ===== SECTION 185: a module that runs main() through another function =====
# The top level below calls S185start, which calls main: main runs exactly
# once, as in CPython, and the implicit entry a module with a main() otherwise
# gets is skipped. Rows from CPython 3.14.
S185runs = [0]

def s185():
    S185runs[0] += 1
    check("entry2", S185runs[0] == 1)

def main():
    s183() # SECTION-CALL 183
    s185() # SECTION-CALL 185
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]

def S185start():
    rc = main()
    if rc != 0:
        exit(rc)

if checks[0] == 0: S185start()
