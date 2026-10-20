# Full-syntax test: Python (3.12 core grammar) - part 5, sections 151 and 152:
# random's str seed and statistics' paired measures. Seeding MT19937 from a
# SHA-512 digest is costly enough in the -frozen interpreter leg to need a file
# of its own under the step brake (-max-steps, 100M per call); see
# tests/python-test-full.py for the conventions.

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


# ===== SECTION 152: statistics.covariance, correlation, linear_regression =====
# Built on math.sumprod; every value here is CPython 3.14's.
import statistics as S152st

def s152():
    x = [1, 2, 3, 4, 5, 6, 7, 8, 9]
    y = [1, 2, 3, 1, 2, 3, 1, 2, 3]
    z = [2.5, -1.25, 7.0, 3.3, 0.1, 9.9, -4.0, 2.2, 5.5]
    lr = S152st.linear_regression(x, z)
    check("stat2", S152st.covariance(x, y) == 0.75 and S152st.covariance(x, z) == 0.8687500000000001
          and S152st.correlation(x, z) == 0.0742632702055162 and S152st.correlation(x, [9 - v for v in x]) == -1.0
          and S152st.correlation([1, 2, 2, 3], [5, 6, 6, 8], method="ranked") == 1.0
          and repr(lr) == "LinearRegression(slope=0.11583333333333336, intercept=2.2263888888888888)"
          and S152st.linear_regression(x, z, proportional=True).slope == 0.46736842105263166)
    msgs = []
    for f in [lambda: S152st.covariance([1], [2]), lambda: S152st.correlation([1, 1], [1, 2]),
              lambda: S152st.linear_regression([2, 2], [1, 3])]:
        try:
            f()
        except S152st.StatisticsError as e:
            msgs.append(str(e))
    check("stat3", msgs == ["covariance requires at least two data points", "at least one of the inputs is constant",
                            "x is constant"])


def main():
    s151() # SECTION-CALL 151
    s152() # SECTION-CALL 152
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
