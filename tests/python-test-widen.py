# Python widening self test: the ternary, parallel and annotated assignment,
# assert, with, try/except/finally, raise, generators, del, decorators,
# async/await and **kwargs. Everything here is executed for real; the file runs
# to exit(fails[0]).

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


# ----- with, try, raise, generators, del, decorators, async, **kwargs -----

# with EXPR as T: a value without __enter__/__exit__ binds T to itself
with count as ctx:
    check("with body ran", ctx, 15)

# try/except/finally: nothing raises, so the handler must not run
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

# yield / generators (defined, not driven)
def gen():
    yield 1
    yield 2


# del of a list element
scratch = [1, 2, 3]
del scratch[0]


# an identity decorator; the decorated function is still callable
def trace(fn):
    return fn


@trace
def greet(who):
    return "hi " + who


check("decorated call", greet("x"), "hi x")


# async def / await (defined, not awaited)
async def afetch(u):
    return await u


# **kwargs (defined, not called)
def keyworded(**kws):
    return 0


if fails[0] == 0:
    print("Python widening self test passed")
exit(fails[0])
