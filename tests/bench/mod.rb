# See mod.kt.
s = 0
i = 0
while i < 40000
  s = s + i % 7
  i = i + 1
end
puts s
