# Full-syntax test: Python (3.12 core grammar) - part 5, sections 151 and 182:
# random's str and bytes seeds and a module that runs its own main(). Seeding
# MT19937 from a SHA-512 digest is some thirty million IR steps of int bit
# operations in the -frozen interpreter leg, so it has a file of its own under
# the step brake (-max-steps, 100M per call); see tests/python-test-full.py for
# the conventions.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 151: random's str seed =====
# A str seed is followed by the SHA-512 digest of its UTF-8 bytes and read as one
# big-endian int, CPython 3.14's version 2 seeding: these are its draws. The bit
# operators are infinite two's complement at any size.
import random as S151rd

def s151():
    S151rd.seed("\u00e9")
    check("seedstr", S151rd.random() == 0.9689252893653495 and S151rd.randint(1, 100) == 54)
    r = S151rd.Random(b"\x00\xffmetacompiler")
    check("seedbytes", r.random() == 0.7022149982222959 and r.getrandbits(64) == 5208907386699013314)
    v = [0xFFFFFFFFFFFFFFFF, -0x123456789abcdef01, 2**64 - 3, -(2**40), -(2**70), 0xFFFF, -12345678901234567890]
    check("bigbits", (v[0] & v[1], v[2] ^ v[3], (v[4] | v[5]) >> 17, v[6] << 33) ==
          (15905192667998458111, -18446742974197923843, -9007199254740992, -106048574255439366224483450880))


# ===== SECTION 182: a module that runs main() itself =====
# This file ends in the `if __name__ == "__main__":` guard, so its main() runs
# exactly once: the implicit entry a module with a main() otherwise gets is
# skipped when the top level calls main or tests __name__. Rows from CPython 3.14.
S182runs = [0]

def s182():
    S182runs[0] += 1
    check("entry1", S182runs[0] == 1 and __name__ == "__main__")

def main():
    s151() # SECTION-CALL 151
    s182() # SECTION-CALL 182
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]

if __name__ == "__main__":
    S182rc = main()
    if S182rc != 0:
        exit(S182rc)
