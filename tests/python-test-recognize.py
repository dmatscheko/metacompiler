"""Recognition self-test for the widened Python surface.

Every GENUINELY compiled construct here (module and function docstrings, the
is / is not identity tests, plain calls and arithmetic) runs and self-checks
under both a default run and -warn-unsupported.

Everything else here (class definitions - plain, based and decorated; keyword and
*starred / **mapping call arguments; tuples; set literals and set comprehensions;
the ** power operator and the | & ^ << >> ~ bit operators; for-loop tuple
unpacking) USED to be accept-and-not-implemented, so a plain run aborted at the
first one and this file was a by-design SHOULD-FAIL matrix entry. The full-syntax
work implemented every one of them, so the file now runs clean with or without
-warn-unsupported and the entry became an ordinary one. It is still worth running
both ways: the -warn-unsupported entry proves no construct here warns.
"""

fails = [0]


def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1


# ----- genuinely compiled: docstrings and is / is not identity -----


def describe(value):
    '''Return a fixed label (this triple-quoted docstring is discarded).'''
    return "labelled"


check("docstring fn still callable", describe(1), "labelled")

nothing = None
check("is None", nothing is None, True)
check("is not None", 5 is not None, True)

shared = [1, 2, 3]
alias = shared
check("is identity same", alias is shared, True)
check("is not identity distinct", [1] is not [1], True)

triple = """line one
line two"""
check("triple-quoted length", len(triple), 17)


# ----- classes, call arguments, tuples, sets, operators (all executed) -----


# a class definition with a class variable and a method
class Shape:
    kind = "shape"

    def area(self):
        return 0


# a class with positional bases and a keyword base
class Circle(Shape, metaclass=type):
    def __init__(self, radius):
        self.radius = radius

    def area(self):
        return 3 * self.radius * self.radius


def register(cls):
    return cls


# a decorated class (an identity decorator)
@register
class Plugin:
    version = 1


# keyword and *starred / **mapping call arguments
opts = dict(a=1, b=2)
biggest = max(*shared)
merged = dict(**opts)

# tuples: empty, singleton, pair, nested
empty = ()
single = (7,)
pair = (1, 2)
nested = (shared, alias)

# set literal and set comprehension
letters = {"a", "b", "c"}
squares = {n * n for n in shared}

# ** power and | & ^ << >> ~ bit operators
power = 2 ** 10
flags = 6 & 3
mask = 1 << 4
mixed = 5 | 2 ^ 1
inverted = ~0

# for-loop tuple unpacking (bare and parenthesized targets)
table = {"x": 1, "y": 2}
for key, val in table.items():
    print(key, val)

# The iterable has to yield PAIRS: `for (a, b) in pair` over one pair would
# iterate 1 and 2, which CPython rejects ("cannot unpack non-iterable int").
for (first, second) in [pair, (3, 4)]:
    print(first, second)


if fails[0] == 0:
    print("Python recognition self test passed")
exit(fails[0])
