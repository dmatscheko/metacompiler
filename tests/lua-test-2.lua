-- Lua subset self test 2: repeat/until, the generic for over ipairs, and the method
-- call/definition sugar (obj:m(args) and function tbl:m()). Counts failures and exits
-- with that count (exit 0 == all pass). Run by both the tree-walking interpreter and the
-- LLVM-IR compiler; both engines must agree byte for byte.

local fails = 0

local function check(name, got, want)
    if got ~= want then
        print("FAIL " .. name .. ": got " .. got .. " want " .. want)
        fails = fails + 1
    end
end

-- repeat ... until: the body always runs at least once and loops while cond is false
local r = 0
local n = 0
repeat
    n = n + 1
    r = r + n
until n >= 5
check("repeat sum", r, 15)
check("repeat count", n, 5)

local once = 0
repeat
    once = once + 1
until true
check("repeat once", once, 1)

-- the until condition sees a local declared inside the body (same scope)
local acc = 0
repeat
    local step = 2
    acc = acc + step
until acc >= 6
check("repeat local in cond", acc, 6)

-- repeat with break
local b = 0
repeat
    b = b + 1
    if b == 3 then
        break
    end
until b >= 10
check("repeat break", b, 3)

-- generic for over ipairs: (index, value) pairs, 1-based, stopping at the first nil
local arr = {10, 20, 30, 40}
local sum = 0
local lastI = 0
for i, v in ipairs(arr) do
    sum = sum + v
    lastI = i
end
check("ipairs value sum", sum, 100)
check("ipairs last index", lastI, 4)

-- ipairs stops at the sequence border (the first nil), ignoring later keys
local sparse = {1, 2}
sparse[4] = 99
local seen = 0
for i, v in ipairs(sparse) do
    seen = seen + 1
end
check("ipairs stops at nil", seen, 2)

-- a single loop variable binds the index only
local idxSum = 0
for i in ipairs(arr) do
    idxSum = idxSum + i
end
check("ipairs index only", idxSum, 10)

-- ipairs with break
local hit = 0
for i, v in ipairs(arr) do
    hit = i
    if v == 30 then
        break
    end
end
check("ipairs break", hit, 3)

-- method definition and call: methods live on the instance table itself (no metatables)
local box = {value = 0}
function box:inc()
    self.value = self.value + 1
    return self.value
end
function box:add(delta)
    self.value = self.value + delta
    return self.value
end
function box:get()
    return self.value
end
check("method inc 1", box:inc(), 1)
check("method inc 2", box:inc(), 2)
check("method add arg", box:add(10), 12)
check("method get", box:get(), 12)

-- the colon call passes the receiver as self; an explicit dot call must do so by hand
check("explicit self call", box.get(box), 12)

-- a dotted function definition writes a plain field (no implicit self)
local M = {}
function M.square(x)
    return x * x
end
function M.cube(x)
    return x * x * x
end
check("dotted square", M.square(6), 36)
check("dotted cube", M.cube(3), 27)

-- a method that reads and writes several fields through self
local pt = {x = 1, y = 2}
function pt:move(dx, dy)
    self.x = self.x + dx
    self.y = self.y + dy
    return self.x + self.y
end
check("method two args", pt:move(4, 5), 12)

check("no fails yet", fails, 0)

if fails == 0 then
    print("Lua subset self test 2 passed")
end
exit(fails)
