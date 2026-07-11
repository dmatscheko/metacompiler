# Python subset self test: dict .items() views and dict comprehensions.
# The file runs top to bottom and ends with exit(fails[0]), so the run exits 0
# exactly when every check passes. The same program is run by both the
# interpreter and the LLVM-IR compiler, under both engines.

fails = [0]

def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1

def check_true(name, got):
    if not got:
        print("FAIL", name, "expected a true value")
        fails[0] += 1

d = {"a": 1, "b": 2, "c": 3}

# ---- dict.items(): a list of [key, value] pairs in insertion order ----
its = d.items()
check("items len", len(its), 3)
check("first pair key", its[0][0], "a")
check("first pair value", its[0][1], 1)
check("third pair key", its[2][0], "c")
check("third pair value", its[2][1], 3)
check("items render", f"{d.items()}", "[['a', 1], ['b', 2], ['c', 3]]")

# iterating items (single loop target, then index the pair)
ksum = ""
vsum = 0
for pair in d.items():
    ksum += pair[0]
    vsum += pair[1]
check("items iterate keys", ksum, "abc")
check("items iterate values", vsum, 6)

# items of an empty dict is empty
empty = {}
check("empty items", len(empty.items()), 0)

# rebuilding a dict from its items round-trips
rebuilt = {}
for p in d.items():
    rebuilt[p[0]] = p[1]
check("rebuild len", len(rebuilt), 3)
check("rebuild value", rebuilt["b"], 2)
check("rebuild render", f"{rebuilt}", "{'a': 1, 'b': 2, 'c': 3}")

# .items() still coexists with the existing .keys()/.values() views
check("keys still work", list(d.keys())[1], "b")
check("values still work", list(d.values())[1], 2)

# ---- dict comprehensions ----
squares = {n: n * n for n in range(5)}
check("comp len", len(squares), 5)
check("comp value", squares[4], 16)
check("comp value zero", squares[0], 0)
check("comp render", f"{squares}", "{0: 0, 1: 1, 2: 4, 3: 9, 4: 16}")

# a plain dict literal must still parse as a literal, not a comprehension
lit = {"x": 10, "y": 20}
check("literal still works", lit["y"], 20)
check("literal len", len(lit), 2)

# comprehension with a filter condition
evens = {n: n * n for n in range(6) if n % 2 == 0}
check("comp if len", len(evens), 3)
check_true("comp if has key", 4 in evens)
check_true("comp if drops key", 3 not in evens)
check("comp if value", evens[4], 16)

# comprehension iterating a dict directly (its keys) with a computed value
lengths = {w: len(w) for w in {"hi": 0, "hey": 0, "there": 0}}
check("comp over dict len", len(lengths), 3)
check("comp over dict value", lengths["there"], 5)

# comprehension with a computed key from a list element
words = ["apple", "banana", "cherry"]
by_first = {w[0]: w for w in words}
check("comp key expr a", by_first["a"], "apple")
check("comp key expr c", by_first["c"], "cherry")

# comprehension fed by .items() (both new features together)
doubled = {p[0]: p[1] * 2 for p in d.items()}
check("comp from items b", doubled["b"], 4)
check("comp from items render", f"{doubled}", "{'a': 2, 'b': 4, 'c': 6}")

# the comprehension variable is comprehension-local and does not leak
n = 99
sq = {n: n for n in range(3)}
check("comp no leak value", n, 99)
check("comp no leak len", len(sq), 3)

check("fails so far", fails[0], 0)
if fails[0] == 0:
    print("Python items/dict-comprehension self test passed")
exit(fails[0])
