# Operands read out of an array so the constant folder cannot fold them.
xs = [0, 1, -1, 7, -7, 255, 65535, 2147483647, -2147483648]
ys = [1, 2, 3, 7, 255, 65536]
i = 0
while i < len(xs):
    j = 0
    while j < len(ys):
        a = xs[i]
        b = ys[j]
        print(a, b, a + b, a - b, a * b, a % b, a // b)
        j = j + 1
    i = i + 1
