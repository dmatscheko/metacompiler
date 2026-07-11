-- Lua subset self test: the completed table/library surface. Exercises pairs() full
-- key/value enumeration (deterministic insertion order via the js_keys view), next()
-- over the array part, table.insert/table.remove, and the string.* / math.* libraries,
-- alongside the already-solid ipairs and the # length operator. Counts failures and
-- exits with that count (0 == all pass). Run by both the tree-walking interpreter and
-- the LLVM-IR compiler; both engines must agree byte for byte.

local fails = 0

local function check(name, got, want)
    if got ~= want then
        print("FAIL " .. name .. ": got " .. got .. " want " .. want)
        fails = fails + 1
    end
end

-- pairs over a string-keyed map: every key/value is visited exactly once, in insertion
-- order (both engines enumerate keys deterministically), so the value sum, the visit
-- count, and the concatenation of keys are all well defined.
local conf = {width = 4, height = 3, depth = 2}
local total = 0
local count = 0
local order = ""
for k, v in pairs(conf) do
    total = total + v
    count = count + 1
    order = order .. k .. ","
end
check("pairs value sum", total, 9)
check("pairs visit count", count, 3)
check("pairs key order", order, "width,height,depth,")

-- pairs over the array part yields the stringified 1-based keys ("1".."3").
local arr = {10, 20, 30}
local asum = 0
local akeys = ""
for k, v in pairs(arr) do
    asum = asum + v
    akeys = akeys .. k
end
check("pairs array value sum", asum, 60)
check("pairs array keys", akeys, "123")

-- a single loop variable binds the key only
local kcat = ""
for k in pairs(conf) do
    kcat = kcat .. k
end
check("pairs key only", kcat, "widthheightdepth")

-- pairs with an early break stops the enumeration
local seen = 0
for k, v in pairs(conf) do
    seen = seen + 1
    if k == "height" then
        break
    end
end
check("pairs break", seen, 2)

-- a key set to nil is removed from the enumeration (its value is nil, so pairs skips it)
local sparse = {a = 1, b = 2, c = 3}
sparse.b = nil
local sparseSum = 0
local sparseCount = 0
for k, v in pairs(sparse) do
    sparseSum = sparseSum + v
    sparseCount = sparseCount + 1
end
check("pairs skips removed key sum", sparseSum, 4)
check("pairs skips removed key count", sparseCount, 2)

-- next() walks the array part: next(t) is index 1, next(t, i) is i+1, nil past the end
check("next start", next(arr), 1)
check("next middle", next(arr, 1), 2)
check("next last", next(arr, 3), nil)
-- next as a manual iterator, summing the array through the index it hands back
local nsum = 0
local i = next(arr)
while i ~= nil do
    nsum = nsum + arr[i]
    i = next(arr, i)
end
check("next manual walk", nsum, 60)

-- table.insert appends at the border (#t + 1); table.remove drops and returns the last
local stack = {}
table.insert(stack, "x")
table.insert(stack, "y")
table.insert(stack, "z")
check("insert grows length", #stack, 3)
check("insert value 1", stack[1], "x")
check("insert value 3", stack[3], "z")
local top = table.remove(stack)
check("remove returns last", top, "z")
check("remove shrinks length", #stack, 2)
check("remove leaves prefix", stack[2], "y")

-- building a sequence with table.insert in a loop, then summing it with ipairs and #
local squares = {}
for n = 1, 5 do
    table.insert(squares, n * n)
end
check("built length", #squares, 5)
local sqSum = 0
for idx, val in ipairs(squares) do
    sqSum = sqSum + val
end
check("ipairs over built table", sqSum, 55)

-- string.* library: len, upper, lower, sub (3-arg inclusive and 2-arg to-end), rep
check("string.len", string.len("metacompiler"), 12)
check("string.upper", string.upper("Lua"), "LUA")
check("string.lower", string.lower("Lua"), "lua")
check("string.sub range", string.sub("compiler", 2, 4), "omp")
check("string.sub tail", string.sub("compiler", 5), "iler")
check("string.rep", string.rep("ab", 3), "ababab")
check("string.rep zero", string.rep("x", 0), "")
-- combined with concatenation
check("string combo", string.upper(string.sub("handle-ir", 1, 6)), "HANDLE")

-- math.* library: floor, ceil, abs, max, min
check("math.floor", math.floor(7.8), 7)
check("math.floor neg", math.floor(-2.5), -3)
check("math.ceil", math.ceil(7.2), 8)
check("math.ceil neg", math.ceil(-2.5), -2)
check("math.abs pos", math.abs(9), 9)
check("math.abs neg", math.abs(-9), 9)
check("math.max", math.max(3, 8), 8)
check("math.min", math.min(3, 8), 3)
-- an average rounded down with the numeric library
local nums = {4, 8, 15, 16, 23, 42}
local acc = 0
for i, v in ipairs(nums) do
    acc = acc + v
end
check("math average floor", math.floor(acc / #nums), 18)

check("no fails yet", fails, 0)

if fails == 0 then
    print("Lua subset complete self test passed")
end
exit(fails)
