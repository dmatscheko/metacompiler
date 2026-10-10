# Unknown \N{...} name fail test. CPython rejects the whole program before it
# runs: SyntaxError: (unicode error) 'unicodeescape' codec can't decode bytes in
# position 2-12: unknown Unicode character name. Both halves must refuse it too,
# rather than decode the escape to "".
print("never printed")
s = "ab\N{FOO BAR}"
print(len(s))
