-- Lua subset self test: counts failures and exits with that count (exit 0 == all pass).
-- The same program is run by the tree-walking interpreter and by the LLVM-IR compiler;
-- both must agree.

local fails = 0

local function check(name, got, want)
    if got ~= want then
        print("FAIL " .. name .. ": got " .. got .. " want " .. want)
        fails = fails + 1
    end
end

-- arithmetic (numbers are floats: / is true division, // floor, % floor modulo)
check("add", 2 + 3, 5)
check("sub", 10 - 4, 6)
check("mul", 6 * 7, 42)
check("div", 7 / 2, 3.5)
check("floordiv", 7 // 2, 3)
check("mod", 7 % 3, 1)
check("neg", -5, 0 - 5)
check("precedence", 2 + 3 * 4, 14)
check("parens", (2 + 3) * 4, 20)

-- string concatenation (with number coercion)
check("concat str", "ab" .. "cd", "abcd")
check("concat num", 1 .. 2, "12")
check("concat mix", "n=" .. 5, "n=5")
check("concat chain", "a" .. "b" .. "c", "abc")

-- comparisons
check("lt", 3 < 5, true)
check("le", 5 <= 5, true)
check("gt", 9 > 2, true)
check("ne", 3 ~= 4, true)
check("eq num", 4 == 4, true)
check("str lt", "a" < "b", true)

-- booleans and nil
check("true is true", true, true)
check("nil eq nil", nil == nil, true)
check("nil ne false", nil ~= false, true)

-- and / or / not with Lua truthiness (only false and nil are falsy; 0 and "" are true)
check("and value", 1 and 2, 2)
check("and nil", nil and 2, nil)
check("or value", false or "x", "x")
check("or first", 7 or 9, 7)
check("zero is truthy", 0 and 42, 42)
check("not nil", not nil, true)
check("not zero", not 0, false)
check("not false", not false, true)

-- if / elseif / else
local function sign(n)
    if n > 0 then
        return "pos"
    elseif n < 0 then
        return "neg"
    else
        return "zero"
    end
end
check("if pos", sign(5), "pos")
check("if neg", sign(-3), "neg")
check("if zero", sign(0), "zero")

-- while
local i = 1
local s = 0
while i <= 5 do
    s = s + i
    i = i + 1
end
check("while sum", s, 15)

-- numeric for, ascending and with a negative step
local f = 0
for k = 1, 5 do
    f = f + k
end
check("for sum", f, 15)
local d = 0
for k = 3, 1, -1 do
    d = d + k
end
check("for step", d, 6)
local prod = 1
for k = 1, 4 do
    prod = prod * k
end
check("for factorial", prod, 24)

-- functions, closures and recursion
local function makeCounter()
    local n = 0
    return function()
        n = n + 1
        return n
    end
end
local c1 = makeCounter()
check("closure 1", c1(), 1)
check("closure 2", c1(), 2)
local c2 = makeCounter()
check("closure fresh", c2(), 1)

local function fib(k)
    if k < 2 then
        return k
    end
    return fib(k - 1) + fib(k - 2)
end
check("fib 10", fib(10), 55)

local add = function(a, b) return a + b end
check("func expr", add(3, 4), 7)
check("missing arg is nil", (function(a, b) return b end)(1), nil)

-- tables as arrays (1 based) and the length operator
local arr = {10, 20, 30}
check("arr[1]", arr[1], 10)
check("arr[3]", arr[3], 30)
check("arr len", #arr, 3)
arr[#arr + 1] = 40
check("arr append", arr[4], 40)
check("arr len grew", #arr, 4)

-- tables as maps
local m = {x = 1, y = 2}
check("map field", m.x, 1)
check("map bracket", m["y"], 2)
m.z = 9
check("map assign", m.z, 9)
local key = "x"
check("map dyn key", m[key], 1)

-- mixed table and nested indexing
local grid = {{1, 2}, {3, 4}}
check("nested", grid[2][1], 3)

-- table identity (tables compare by reference, not value)
local t1 = {}
local t2 = t1
check("same ref", t1 == t2, true)
check("diff ref", {} == {}, false)

-- multiple assignment (all right hand sides evaluated before assigning)
local a, b = 1, 2
check("multi a", a, 1)
check("multi b", b, 2)
a, b = b, a
check("swap a", a, 2)
check("swap b", b, 1)

-- auto-global written at top level, read inside a function
answer = 42
local function readGlobal()
    return answer
end
check("global read", readGlobal(), 42)

check("no fails yet", fails, 0)

if fails == 0 then
    print("Lua subset self test passed")
end
exit(fails)
