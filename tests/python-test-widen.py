# Python widening self test: exercises the newly accepted surface.
#
# The GENUINELY implemented constructs (ternary, parallel/tuple assignment,
# annotated assignment, assert) run and self-check under both the default run and
# -warn-unsupported.
#
# The ACCEPT-AND-NOT-IMPLEMENTED constructs (with, yield,
# del, decorators, async/await, **kwargs) abort a plain run at the first such
# construct with a clean file:line message; under -warn-unsupported they warn and the
# file runs to exit(fails[0]). (try/except/finally, raise, lambda, global/nonlocal
# and *args/default parameters are now implemented and run genuinely here.)

fails = [0]


def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1


# ----- genuinely implemented -----

# a if c else b (short-circuiting, nestable)
check("ternary true", 1 if True else 2, 1)
check("ternary false", "a" if 0 else "b", "b")
n = 7
check("ternary nested", ("neg" if n < 0 else ("zero" if n == 0 else "pos")), "pos")

# a, b = x, y  (and the swap idiom)
a, b = 1, 2
check("parallel a", a, 1)
check("parallel b", b, 2)
a, b = b, a
check("swap a", a, 2)
check("swap b", b, 1)
x, y, z = 10, 20, 30
check("triple assign", x + y + z, 60)
p, q = [4, 5]
check("unpack iterable a", p, 4)
check("unpack iterable b", q, 5)

# name: Type [= value]  (the annotation is parsed and ignored)
count: int = 10
check("annotated int", count, 10)
ratio: float = 1.5
check("annotated float", ratio, 1.5)
pair: list = [1, 2, 3]
check("annotated list", len(pair), 3)
count += 5
check("annotated then aug", count, 15)

# assert
assert count == 15
assert count == 15, "count must be 15"


def add(a, b):
    return a + b


check("plain call still works", add(2, 3), 5)

# lambda, with lexical closure over an enclosing function's locals
doubler = lambda k: k * 2
check("lambda", doubler(6), 12)


def make_adder(n):
    return lambda k: k + n


check("lambda closure", make_adder(3)(4), 7)

# global / nonlocal
tally = 0


def note():
    global tally
    tally = tally + 1


note()
note()
check("global mutate", tally, 2)


def make_step():
    at = 0

    def step():
        nonlocal at
        at = at + 1
        return at

    return step


stepper = make_step()
stepper()
check("nonlocal mutate", stepper(), 2)

# *args and default arguments
def variadic(*args):
    return len(args)


check("varargs", variadic(1, 2, 3), 3)


def defaulted(msg, prefix="> "):
    return prefix + msg


check("default used", defaulted("hi"), "> hi")
check("default overridden", defaulted("hi", "* "), "* hi")


# ----- accepted, not implemented (abort by default; warn + run under -warn) -----

# with EXPR as T:  simple form binds T to EXPR's value and runs the body
with count as ctx:
    check("with body ran", ctx, 15)

# try/except/finally: the try body and finally body run; handlers are dropped
try:
    check("try body ran", 1 + 1, 2)
except ValueError as e:
    check("except must not run", True, False)
finally:
    check("finally ran", 3, 3)


# raise (only reached on the untaken branch here)
def guard(v):
    if v < 0:
        raise ValueError("negative")
    return v


check("raise path skipped", guard(9), 9)

# yield / generators (accepted, not driven)
def gen():
    yield 1
    yield 2


# del (accepted)
scratch = [1, 2, 3]
del scratch[0]


# decorators parse and are ignored; the function is still callable
def trace(fn):
    return fn


@trace
def greet(who):
    return "hi " + who


check("decorated call", greet("x"), "hi x")


# async def / await (accepted)
async def afetch(u):
    return await u


# **kwargs (accepted; the dict does not bind)
def keyworded(**kws):
    return 0


if fails[0] == 0:
    print("Python widening self test passed")
exit(fails[0])
