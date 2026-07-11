# Python try/except/else/finally and raise, genuinely executed (interpreter and
# compiler).
#
# raise unwinds through calls to the first except (exception types are parsed but not
# discriminated), which binds the value with `as`; else runs only when the body
# completed without an exception; finally always runs. A return/break/continue leaving
# a try or except body works in both engines. An uncaught raise is a clean runtime
# error. (Exceptions are raised as plain values here since the builtin exception
# classes are outside the subset.)
#
# The program runs top to bottom and ends with exit(fails[0]), so it exits 0 exactly
# when every check passes; the interpreter and compiler must agree.

fails = [0]

def check(cond):
    if not cond:
        fails[0] = fails[0] + 1

def risky(n):
    if n > 3:
        raise n
    return n * 2

# return out of a try, and out of an except.
def classify(n):
    try:
        if n > 0:
            return n * 10
        raise 0
    except Exception as e:
        return -1
    finally:
        pass

# A return out of an INNER try propagates through the OUTER try.
def nested_return():
    try:
        try:
            return 9
        finally:
            pass
    finally:
        pass
    return 0

# break / continue leaving a try body inside a loop.
def loop_break():
    total = 0
    for i in range(10):
        try:
            if i == 3:
                break
            total = total + i
        finally:
            pass
    return total            # 0+1+2 = 3

def loop_continue():
    total = 0
    for i in range(5):
        try:
            if i == 2:
                continue
            total = total + i
        except Exception as e:
            pass
    return total            # 0+1+3+4 = 8

# else runs only when the try body raised nothing.
def with_else(n):
    tag = 0
    try:
        if n < 0:
            raise n
        tag = 1
    except Exception as e:
        tag = 2
    else:
        tag = tag + 10
    return tag              # n>=0: 11 (else ran); n<0: 2 (except ran, else skipped)

log = [""]
try:
    log[0] = log[0] + "a"
    raise "boom"
    log[0] = log[0] + "X"
except Exception as e:
    log[0] = log[0] + "b"
finally:
    log[0] = log[0] + "c"
check(log[0] == "abc")

caught = [-1]
try:
    risky(5)
    check(False)
except Exception as e:
    caught[0] = e
check(caught[0] == 5)

check(risky(2) == 4)
check(classify(4) == 40)
check(classify(-1) == -1)
check(nested_return() == 9)
check(loop_break() == 3)
check(loop_continue() == 8)
check(with_else(7) == 11)
check(with_else(-1) == 2)

if fails[0] == 0:
    print("Python try/except OK")
exit(fails[0])
