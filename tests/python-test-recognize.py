"""Recognition self-test for the widened Python surface.

Every GENUINELY compiled construct here (module and function docstrings, the
is / is not identity tests, plain calls and arithmetic) runs and self-checks
under both a default run and -warn-unsupported.

Every ACCEPT-AND-NOT-IMPLEMENTED construct (class definitions - plain, based and
decorated; attribute access obj.attr; keyword and *starred / **mapping call
arguments; tuples; set literals and set comprehensions; the ** power operator and
the | & ^ << >> ~ bit operators; for-loop tuple unpacking) aborts a plain run at
the first such construct with a clean file:line message; under -warn-unsupported
they warn and the file runs to exit(fails[0]).
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


# ----- accepted, not implemented (abort by default; warn + run under -warn) -----


# class definitions parse; the body (methods, class vars) is dropped
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


# a decorated class (the decorator parses and is dropped)
@register
class Plugin:
    version = 1


# attribute access (obj.attr with no call)
label = Shape.kind
width = nothing.bit_length

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

for (first, second) in pair:
    print(first, second)


if fails[0] == 0:
    print("Python recognition self test passed")
exit(fails[0])
