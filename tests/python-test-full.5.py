# Full-syntax test: Python (3.12 core grammar) - part 5, section 151: random's
# str seed. Seeding MT19937 from a SHA-512 digest is costly enough in the -frozen
# interpreter leg to need a file of its own under the step brake (-max-steps,
# 100M per call); see tests/python-test-full.py for the conventions.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 151: random's str seed =====
# A str seed is followed by the SHA-512 digest of its UTF-8 bytes and read as one
# big-endian int, CPython 3.14's version 2 seeding: these are its draws.
import random as S151rd

def s151():
    S151rd.seed("\u00e9")
    check("seedstr", S151rd.random() == 0.9689252893653495 and S151rd.randint(1, 100) == 54)


def main():
    s151() # SECTION-CALL 151
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
