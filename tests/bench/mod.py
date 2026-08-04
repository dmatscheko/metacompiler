# See mod.kt. Same loop, so the languages are comparable to each other.
s = 0
i = 0
while i < 40000:
    s = s + i % 7
    i = i + 1
print(s)
