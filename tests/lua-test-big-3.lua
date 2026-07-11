-- Lua subset self test (big 3): Lua's signature closure + table-based OO + functional
-- features. Exercises closures (counters, accumulators, currying, memoization, a closure
-- iterator), higher-order functions (map/filter/reduce/compose over arrays), table-based
-- classes whose methods dispatch through self (obj:method(args), method chaining), and
-- inheritance-by-delegation without metatables. Counts failures and exits with that count
-- (0 == all pass). Run by both the tree-walking interpreter and the LLVM-IR compiler;
-- both engines must agree byte for byte.

local fails = 0

local function check(name, got, want)
    if got ~= want then
        print("FAIL " .. name .. ": got " .. got .. " want " .. want)
        fails = fails + 1
    end
end

-- ----- closures: counters keep private state, fresh per construction -----
local function makeCounter()
    local n = 0
    return function()
        n = n + 1
        return n
    end
end
local c1 = makeCounter()
local c2 = makeCounter()
check("counter c1 first", c1(), 1)
check("counter c1 second", c1(), 2)
check("counter c1 third", c1(), 3)
check("counter c2 independent", c2(), 1)
check("counter c1 continues", c1(), 4)

-- ----- accumulator closure: keeps a running total -----
local function makeAccumulator(start)
    local total = start
    return function(delta)
        total = total + delta
        return total
    end
end
local acc = makeAccumulator(100)
check("acc +10", acc(10), 110)
check("acc +5", acc(5), 115)
check("acc -15", acc(-15), 100)

-- ----- currying: makeAdder(x) returns a function of y -----
local function makeAdder(x)
    return function(y)
        return x + y
    end
end
local add10 = makeAdder(10)
local add100 = makeAdder(100)
check("curry add10(5)", add10(5), 15)
check("curry add100(5)", add100(5), 105)
check("curry inline", makeAdder(3)(4), 7)

-- ----- function composition -----
local function compose(f, g)
    return function(x)
        return f(g(x))
    end
end
local function inc(x)
    return x + 1
end
local function dbl(x)
    return x * 2
end
local incThenDbl = compose(dbl, inc)
local dblThenInc = compose(inc, dbl)
check("compose dbl.inc (5)", incThenDbl(5), 12)
check("compose inc.dbl (5)", dblThenInc(5), 11)
-- composing three functions
local triple = compose(inc, compose(dbl, dbl))
check("compose triple (3)", triple(3), 13)

-- ----- higher-order functions over 1-based arrays -----
local function map(f, a)
    local r = {}
    for i = 1, #a do
        r[i] = f(a[i])
    end
    return r
end
local function filter(pred, a)
    local r = {}
    for i = 1, #a do
        if pred(a[i]) then
            table.insert(r, a[i])
        end
    end
    return r
end
local function reduce(f, init, a)
    local acc = init
    for i = 1, #a do
        acc = f(acc, a[i])
    end
    return acc
end

local nums = {1, 2, 3, 4, 5, 6}
local doubled = map(dbl, nums)
check("map doubled 1", doubled[1], 2)
check("map doubled 6", doubled[6], 12)
check("map length kept", #doubled, 6)

local function isEven(n)
    return n % 2 == 0
end
local evens = filter(isEven, nums)
check("filter evens length", #evens, 3)
check("filter evens first", evens[1], 2)
check("filter evens last", evens[3], 6)

local function addOp(a, b)
    return a + b
end
local function mulOp(a, b)
    return a * b
end
check("reduce sum", reduce(addOp, 0, nums), 21)
check("reduce product", reduce(mulOp, 1, nums), 720)

-- map/filter/reduce pipeline: sum of squares of the even numbers
local function square(x)
    return x * x
end
local pipeline = reduce(addOp, 0, map(square, filter(isEven, nums)))
check("pipeline sum sq of evens", pipeline, 56)

-- ----- closure iterator driven by a while loop -----
local function makeRange(lo, hi)
    local cur = lo - 1
    return function()
        if cur < hi then
            cur = cur + 1
            return cur
        end
        return nil
    end
end
local nextV = makeRange(1, 5)
local rangeSum = 0
local v = nextV()
while v ~= nil do
    rangeSum = rangeSum + v
    v = nextV()
end
check("range iterator sum", rangeSum, 15)

-- ----- memoized Fibonacci via a captured cache table -----
local function makeMemoFib()
    local cache = {}
    local fib
    fib = function(n)
        if n < 2 then
            return n
        end
        if cache[n] ~= nil then
            return cache[n]
        end
        local r = fib(n - 1) + fib(n - 2)
        cache[n] = r
        return r
    end
    return fib
end
local mfib = makeMemoFib()
check("memo fib 10", mfib(10), 55)
check("memo fib 25", mfib(25), 75025)
check("memo fib 30", mfib(30), 832040)

-- ----- table-based class: Stack with methods dispatched through self -----
local Stack = {}
function Stack.push(self, v)
    self.n = self.n + 1
    self.items[self.n] = v
end
function Stack.pop(self)
    local v = self.items[self.n]
    self.items[self.n] = nil
    self.n = self.n - 1
    return v
end
function Stack.peek(self)
    return self.items[self.n]
end
function Stack.size(self)
    return self.n
end
function Stack.isEmpty(self)
    return self.n == 0
end
function Stack.new()
    local self = {items = {}, n = 0}
    self.push = Stack.push
    self.pop = Stack.pop
    self.peek = Stack.peek
    self.size = Stack.size
    self.isEmpty = Stack.isEmpty
    return self
end

local s = Stack.new()
check("stack starts empty", s:isEmpty(), true)
s:push(10)
s:push(20)
s:push(30)
check("stack size 3", s:size(), 3)
check("stack peek", s:peek(), 30)
check("stack not empty", s:isEmpty(), false)
check("stack pop 30", s:pop(), 30)
check("stack pop 20", s:pop(), 20)
check("stack size 1", s:size(), 1)
check("stack pop 10", s:pop(), 10)
check("stack empty again", s:isEmpty(), true)

-- ----- Vec2 class: methods return new vectors, enabling method chaining -----
local Vec = {}
function Vec.add(self, o)
    return Vec.new(self.x + o.x, self.y + o.y)
end
function Vec.scale(self, k)
    return Vec.new(self.x * k, self.y * k)
end
function Vec.dot(self, o)
    return self.x * o.x + self.y * o.y
end
function Vec.lenSq(self)
    return self.x * self.x + self.y * self.y
end
function Vec.new(x, y)
    local self = {x = x, y = y}
    self.add = Vec.add
    self.scale = Vec.scale
    self.dot = Vec.dot
    self.lenSq = Vec.lenSq
    return self
end

local a = Vec.new(1, 2)
local b = Vec.new(3, 4)
local sum = a:add(b)
check("vec add x", sum.x, 4)
check("vec add y", sum.y, 6)
check("vec dot", a:dot(b), 11)
check("vec lenSq 3,4", b:lenSq(), 25)
local scaled = a:scale(3)
check("vec scale x", scaled.x, 3)
check("vec scale y", scaled.y, 6)
-- method chaining: (a + b) then scaled by 2, then + a
local chained = a:add(b):scale(2):add(a)
check("vec chain x", chained.x, 9)
check("vec chain y", chained.y, 14)

-- ----- inheritance by delegation (no metatables): Square reuses Rectangle -----
local Rectangle = {}
function Rectangle.area(self)
    return self.w * self.h
end
function Rectangle.perimeter(self)
    return 2 * (self.w + self.h)
end
function Rectangle.new(w, h)
    local self = {w = w, h = h, kind = "rect"}
    self.area = Rectangle.area
    self.perimeter = Rectangle.perimeter
    return self
end
local Square = {}
function Square.new(side)
    -- a square is constructed as a rectangle with equal sides, inheriting its methods
    local self = Rectangle.new(side, side)
    self.kind = "square"
    return self
end

local rect = Rectangle.new(3, 4)
check("rect area", rect:area(), 12)
check("rect perimeter", rect:perimeter(), 14)
check("rect kind", rect.kind, "rect")
local sq = Square.new(5)
check("square area inherited", sq:area(), 25)
check("square perimeter inherited", sq:perimeter(), 20)
check("square kind overridden", sq.kind, "square")

-- ----- a small bank account object with encapsulated balance -----
local Account = {}
function Account.deposit(self, amt)
    self.balance = self.balance + amt
    return self.balance
end
function Account.withdraw(self, amt)
    if amt > self.balance then
        return false
    end
    self.balance = self.balance - amt
    return true
end
function Account.new(opening)
    local self = {balance = opening}
    self.deposit = Account.deposit
    self.withdraw = Account.withdraw
    return self
end

local acct = Account.new(50)
check("acct deposit", acct:deposit(25), 75)
check("acct withdraw ok", acct:withdraw(30), true)
check("acct balance", acct.balance, 45)
check("acct overdraw blocked", acct:withdraw(100), false)
check("acct balance unchanged", acct.balance, 45)

check("no fails yet", fails, 0)

if fails == 0 then
    print("Lua subset big self test 3 passed")
end
exit(fails)
