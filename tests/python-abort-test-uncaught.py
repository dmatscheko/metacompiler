# ABORT PROBE for an exception nobody catches: every leg must end with CPython's
# last traceback line, `KeyError: ('k', 2)` - the class name, then str(e), which
# for KeyError is the repr of its argument and for two arguments the args tuple.
# See the "SHOULD ABORT" rows in .vscode/launch.json and abort_norm() in test.sh.
def lookup(d, k):
    if k not in d:
        raise KeyError(k, 2)
    return d[k]

print("before the abort")
print(lookup({}, "k"))
print("never reached")
