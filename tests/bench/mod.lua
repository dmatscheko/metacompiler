-- See mod.kt.
local s = 0
local i = 0
while i < 40000 do
    s = s + i % 7
    i = i + 1
end
print(s)
