-- Fast feature-matrix test for the Lua interpreter (lua-interpreter.abnf) and the
-- LLVM-IR compiler (lua-to-llvm-ir.abnf). It replaces the four algorithm-themed
-- lua-test-big-* stress tests: instead of large loops (sorting batteries, Ackermann,
-- expression parsers) every implemented construct is exercised with the SMALLEST
-- program that can prove it works - loops run 0, 1, 3 or 4 times, recursion stays
-- below depth 6. A failed check prints its id (so a diff pinpoints it) and the
-- program ends with exit(fails); exit 0 and byte-identical output on all four legs
-- (interpreter/compiler x goja/-frozen) mean everything passed.

local fails = 0
local checks = 0

local function check(name, got, want)
    checks = checks + 1
    if got ~= want then
        print("FAIL " .. name)
        fails = fails + 1
    end
end

-- ----- numbers, arithmetic, precedence (floats; / true, // floor, % floor) -----
check("arith-precedence", 2 + 3 * 4, 14)
check("arith-paren", (2 + 3) * 4, 20)
check("arith-unary-minus", -3 + 5, 2)
check("arith-chain", 20 - 5 - 3, 12)
check("arith-true-div", 7 / 2, 3.5)
check("arith-floor-div", 7 // 2, 3)
check("arith-floor-div-neg", -7 // 2, -4)
check("arith-mod", 7 % 3, 1)
check("arith-mod-neg", -7 % 3, 2)
check("arith-mod-neg-divisor", 7 % -3, -2)

-- ----- string concatenation (with number coercion) -----
check("concat", "foo" .. "bar", "foobar")
check("concat-chain", "a" .. "b" .. "c", "abc")
check("concat-num", "n=" .. 5, "n=5")
check("concat-two-nums", 1 .. 2, "12")
check("concat-float", "v=" .. 3.5, "v=3.5")
check("concat-neg", "v=" .. (0 - 3), "v=-3")

-- ----- comparisons -----
check("cmp-lt", 2 < 3, true)
check("cmp-le", 3 <= 3, true)
check("cmp-gt", 9 > 2, true)
check("cmp-ge", 3 >= 4, false)
check("cmp-eq", 4 == 4, true)
check("cmp-ne", 3 ~= 4, true)
check("cmp-str", ("a" < "b") and ("apple" < "banana") and not ("b" < "a"), true)
check("cmp-mixed-types", "1" == 1, false)

-- ----- booleans, nil, truthiness (only false and nil are falsy) -----
check("nil-eq-nil", nil == nil, true)
check("nil-ne-false", nil ~= false, true)
check("and-value", 1 and 2, 2)
check("and-nil", nil and 2, nil)
check("or-value", false or "x", "x")
check("or-first", 7 or 9, 7)
check("zero-truthy", 0 and 42, 42)
check("empty-str-truthy", "" and 1, 1)
check("not", (not nil) and (not false) and not (not true), true)
local sideFx = 0
local function bump()
    sideFx = sideFx + 1
    return true
end
local noRun = false and bump()
local oneRun = true and bump()
local skipRun = true or bump()
check("short-circuit", sideFx, 1)
check("short-circuit-values", (noRun == false) and (oneRun == true) and (skipRun == true), true)

-- ----- strings library -----
check("str-len", string.len("hello"), 5)
check("str-len-op", #"hello", 5)
check("str-len-op-var", (function() local s = "abc" return #s end)(), 3)
check("str-len-empty", string.len(""), 0)
-- A Lua string is a BYTE string, so its length is its byte count, not its character
-- count: lua 5.5 answers 6 here, and this line used to assert 5 - the answer both
-- halves gave while they held UTF-16 text. Both now hold one host character per byte.
check("str-len-unicode", string.len("héllo"), 6)
check("str-len-bytes", #"日本", 6)                       -- three bytes per character
check("str-sub-bytes", string.sub("héllo", 1, 3), "hé") -- sub cuts at BYTE offsets
check("str-sub-byte-mid", string.sub("日本", 4), "本")
check("str-esc-byte", "\xc3" == "\195", true)           -- an escape names one byte
check("str-esc-utf8", #"\u{65e5}", 3)                   -- \u{...} writes UTF-8
check("str-upper-bytes", string.upper("héllo"), "HéLLO") -- C locale: bytes > 127 unchanged
-- Every string shares one metatable whose __index is the string library, so the
-- s:method() form works without naming the library table.
check("str-meta-len", ("héllo"):len(), 6)
check("str-meta-sub", ("hello"):sub(-3), "llo")
check("str-meta-upper", ("ab"):upper() .. ("CD"):lower(), "ABcd")
check("str-meta-rep", ("ab"):rep(2), "abab")
check("str-meta-index-nil", ("abc")[1], nil)            -- only the library is reachable
check("str-escapes", string.len("a\tb") + string.len("x\ny"), 6)
check("str-single-quote", 'abc', "abc")
check("str-upper-lower", string.upper("Lua") .. string.lower("Lua"), "LUAlua")
check("str-sub-range", string.sub("compiler", 2, 4), "omp")
check("str-sub-tail", string.sub("compiler", 5), "iler")
check("str-sub-single", string.sub("abc", 2, 2), "b")
-- Lua's negative index is INCLUSIVE and 1-based: -1 is the last character, so it
-- normalises to #s + i + 1, not to #s + i, and both ends clamp into 1..#s instead of
-- being reinterpreted. Handing i-1 to a JS-style slice got every one of these wrong.
-- Checked against lua 5.5.
check("str-sub-neg-start", string.sub("hello", -3), "llo")
check("str-sub-neg-end", string.sub("hello", 2, -2), "ell")
check("str-sub-neg-both", string.sub("hello", -2, -1), "lo")
check("str-sub-neg-last", string.sub("hello", -1), "o")
check("str-sub-full", string.sub("hello", 1, -1), "hello")
check("str-sub-zero-start", string.sub("hello", 0, 2), "he")
check("str-sub-under", string.sub("hello", -100, 2), "he")
check("str-sub-over-end", string.sub("hello", 4, 100), "lo")
check("str-sub-empty-range", string.sub("hello", 4, 2), "")
check("str-rep", string.rep("ab", 3), "ababab")
check("str-rep-zero", string.rep("x", 0), "")
check("str-combo", string.upper(string.sub("handle", 1, 4)), "HAND")

-- ----- control flow: if / while / repeat / numeric for -----
local function grade(n)
    if n > 10 then
        return "big"
    elseif n > 5 then
        return "mid"
    else
        return "small"
    end
end
check("if-elseif-else", grade(11) .. grade(7) .. grade(1), "bigmidsmall")

local w0 = 0
while w0 > 0 do w0 = w0 - 1 end            -- runs zero times
check("while-zero", w0, 0)
local w3 = 0
while w3 < 3 do w3 = w3 + 1 end            -- runs three times
check("while-three", w3, 3)
local wb = 0
while true do
    wb = wb + 1
    if wb == 2 then break end
end
check("while-break", wb, 2)

local once = 0
repeat once = once + 1 until true          -- body runs exactly once
check("repeat-once", once, 1)
local rep = 0
repeat rep = rep + 1 until rep >= 3
check("repeat-three", rep, 3)
local acc = 0
repeat
    local step = 2                         -- the until condition sees body locals
    acc = acc + step
until acc >= 4
check("repeat-local-in-cond", acc, 4)
local rb = 0
repeat
    rb = rb + 1
    if rb == 2 then break end
until rb >= 9
check("repeat-break", rb, 2)

local forSum = 0
for i = 1, 3 do forSum = forSum + i end
check("for-basic", forSum, 6)
local forZero = 0
for i = 1, 0 do forZero = forZero + 1 end  -- runs zero times
check("for-zero", forZero, 0)
local forStep = ""
for i = 1, 5, 2 do forStep = forStep .. i end
check("for-step", forStep, "135")
local forDown = ""
for i = 3, 1, -1 do forDown = forDown .. i end
check("for-step-negative", forDown, "321")
local forBrk = ""
for i = 0, 8 do
    if i == 2 then break end
    forBrk = forBrk .. i
end
check("for-break", forBrk, "01")

local nested = ""
for o = 0, 1 do
    for i = 0, 2 do
        if i == 1 then break end           -- inner break must not end the outer loop
        nested = nested .. o .. i
    end
end
check("nested-break", nested, "0010")

-- ----- functions, closures, recursion -----
local function add(a, b) return a + b end
check("fn-args", add(20, 22), 42)
check("fn-missing-arg", (function(a, b) return b end)(1), nil)

local function fib(n)
    if n < 2 then return n end
    return fib(n - 1) + fib(n - 2)
end
check("fn-recursion", fib(6), 8)

function isEven(n)                          -- globals, so mutual recursion resolves
    if n == 0 then return true end
    return isOdd(n - 1)
end
function isOdd(n)
    if n == 0 then return false end
    return isEven(n - 1)
end
check("fn-mutual-recursion", isEven(4) and isOdd(5), true)

local function makeCounter()
    local n = 0
    return function()
        n = n + 1
        return n
    end
end
local c1 = makeCounter()
local c2 = makeCounter()
c1()
c1()
check("closure-independent", c1() == 3 and c2() == 1, true)

local function applyTwice(f, x) return f(f(x)) end
check("fn-higher-order", applyTwice(function(n) return n * 2 end, 3), 12)
local double = function(n) return n + n end
check("fn-expression", double(21), 42)

local function makeAdder(a)                 -- closure over a parameter
    return function(b) return a + b end
end
local add10 = makeAdder(10)
check("closure-over-param", add10(5), 15)

-- ----- tables as arrays (1-based) -----
local arr = {10, 20, 30}
check("arr-index", arr[1] + arr[3], 40)
check("arr-len", #arr, 3)
check("arr-len-empty", #{}, 0)
arr[2] = 21
check("arr-write", arr[2], 21)
arr[#arr + 1] = 40
check("arr-append", #arr == 4 and arr[4] == 40, true)
local grid = {{1, 2}, {3, 4}}
check("arr-nested", grid[2][1], 3)
check("arr-missing-is-nil", arr[9], nil)

-- ----- tables as maps -----
local m = {x = 1, y = 2}
check("map-field", m.x, 1)
check("map-bracket", m["y"], 2)
m.z = 9
m["w"] = 8
check("map-assign", m.z + m.w, 17)
local key = "x"
check("map-dyn-key", m[key], 1)
check("map-missing-field", m.absent, nil)
check("map-missing-index", m["nope"], nil)
local mixed = {5, 6, tag = "t"}            -- array part and hash part together
check("map-mixed", #mixed == 2 and mixed.tag == "t" and mixed[2] == 6, true)
local node = {value = 1}                   -- linked-node pattern: table field chain
node.next = {value = 2}
check("map-deep-dot", node.next.value, 2)
local ops = {}
ops.twice = double                         -- a function VALUE stored in a field
check("table-fn-field", ops.twice(4), 8)
local t1 = {}
local t2 = t1
check("table-identity", (t1 == t2) and ({} == {}) == false, true)

-- ----- generic for: ipairs -----
local seq = {10, 20, 30, 40}
local ipSum = 0
local lastI = 0
for i, v in ipairs(seq) do
    ipSum = ipSum + v
    lastI = i
end
check("ipairs-sum", ipSum == 100 and lastI == 4, true)
local sparse = {1, 2}
sparse[4] = 99
local seen = 0
for i, v in ipairs(sparse) do seen = seen + 1 end
check("ipairs-stops-at-nil", seen, 2)
local idxOnly = 0
for i in ipairs({7, 7, 7}) do idxOnly = idxOnly + i end
check("ipairs-index-only", idxOnly, 6)
local hit = 0
for i, v in ipairs(seq) do
    hit = i
    if v == 30 then break end
end
check("ipairs-break", hit, 3)
local none = 0
for i, v in ipairs({}) do none = none + 1 end
check("ipairs-empty", none, 0)

-- ----- generic for: pairs (deterministic insertion order) and next -----
local conf = {width = 4, height = 3, depth = 2}
local total = 0
local order = ""
for k, v in pairs(conf) do
    total = total + v
    order = order .. k .. ","
end
check("pairs-sum", total, 9)
check("pairs-order", order, "width,height,depth,")
local kOnly = ""
for k in pairs(conf) do kOnly = kOnly .. k end
check("pairs-key-only", kOnly, "widthheightdepth")
local pBreak = 0
for k, v in pairs(conf) do
    pBreak = pBreak + 1
    if k == "height" then break end
end
check("pairs-break", pBreak, 2)
local removed = {a = 1, b = 2, c = 3}
removed.b = nil                            -- a nil'ed key drops out of pairs
local remSum = 0
for k, v in pairs(removed) do remSum = remSum + v end
check("pairs-removed-key", remSum, 4)
local akeys = ""
for k, v in pairs({9, 8, 7}) do akeys = akeys .. k end
check("pairs-array-keys", akeys, "123")
local three = {5, 6, 7}
check("next-start", next(three), 1)
check("next-middle", next(three, 1), 2)
check("next-end", next(three, 3), nil)
local nsum = 0
local ni = next(three)
while ni ~= nil do
    nsum = nsum + three[ni]
    ni = next(three, ni)
end
check("next-walk", nsum, 18)

-- ----- table library -----
local stack = {}
table.insert(stack, "x")
table.insert(stack, "y")
table.insert(stack, "z")
check("table-insert", #stack == 3 and stack[1] == "x" and stack[3] == "z", true)
local top = table.remove(stack)
check("table-remove", top == "z" and #stack == 2, true)
check("table-remove-empty", table.remove({}), nil)

-- ----- math library -----
check("math-floor", math.floor(7.8), 7)
check("math-floor-neg", math.floor(-2.5), -3)
check("math-ceil", math.ceil(7.2), 8)
check("math-ceil-neg", math.ceil(-2.5), -2)
check("math-abs", math.abs(-9) + math.abs(9), 18)
check("math-max-min", math.max(3, 8) + math.min(3, 8), 11)

-- ----- multiple assignment -----
local ma, mb = 1, 2
check("multi-assign", ma == 1 and mb == 2, true)
ma, mb = mb, ma                            -- both right sides evaluated first
check("multi-swap", ma == 2 and mb == 1, true)
local p, q = 1
check("multi-extra-nil", p == 1 and q == nil, true)
local noInit
check("local-no-init", noInit, nil)

-- ----- methods: colon definition/call sugar, dotted functions, auto-globals -----
local box = {value = 10}
function box:add(delta)
    self.value = self.value + delta
    return self.value
end
function box:get()
    return self.value
end
check("method-colon", box:add(5), 15)
check("method-explicit-self", box.get(box), 15)
local M = {}
function M.square(x) return x * x end
check("method-dotted", M.square(6), 36)
local pt = {x = 1, y = 2}
function pt:move(dx, dy)
    self.x = self.x + dx
    self.y = self.y + dy
    return self.x + self.y
end
check("method-two-args", pt:move(4, 5), 12)

answer = 42                                -- bare assignment creates a global
local function readGlobal()
    return answer
end
check("auto-global", readGlobal(), 42)

-- ----- everything combined in one small pipeline (3-element data flow) -----
local function transform(list)
    local out = ""
    for i, n in ipairs(list) do
        if n < 0 then
            out = out .. "x"
        elseif n % 2 == 0 then
            out = out .. "e" .. n
        else
            out = out .. "o" .. n
        end
    end
    return out
end
check("combined-pipeline", transform({1, 2, -3}), "o1e2x")

-- ----- float rendering: the 15-then-17 window, chosen by ROUND-TRIP -----
-- docs/todo.md 1.8. lua 5.5.0 writes a float with "%.15g" and falls back to
-- "%.17g" only when the text does not read back; it NEVER tries 16. luaGStr used
-- to pick between them by counting the digits the host printed for `"" + n`, and
-- goja's Number-to-String is not always the shortest round-tripping form: for the
-- first seven values below it prints fifteen digits that read back as a DIFFERENT
-- double (and for #4..#6, which sit at the subnormal boundary, it emits a
-- non-digit character outright - "B.70548258834231e-309"). The choice is now made
-- the way lua makes it, by parsing floPrec(n, 15) back, so the answer is the same
-- under goja, under -frozen, in the compiled half and natively.
--
-- Every operand is read out of THIS TABLE, never written as a literal at the
-- check, so the constant folder cannot answer in the renderer's place. All twelve
-- expected strings are `lua` 5.5.0's own output.
local gflo = {
    -1.4487572656910099e-208,     -- goja: -1.44875726569101e-208, fifteen digits
    5.179975601040939e21,
    -9.772720132874869e-147,
    1.870548258834231e-308,       -- goja: "B.70548258834231e-309"
    -1.2658976361944994e-308,     -- goja: "-<.658976361944994e-309"
    -1.3184270331886705e-308,     -- goja: "-=.184270331886705e-309"
    0.0009339729831804499,
    1 / 3,                        -- 17 digits, NOT the 16-digit shortest form
    1e15,                         -- P is 15 and exp is 15, so %g goes exponential
    1234567890123456.0,           -- P is 17 and exp is 15, so it does not
    5e-324,                       -- fifteen digits round-trip the smallest subnormal
    1e16,
}
check("flostr-goja-short-1", "" .. gflo[1], "-1.4487572656910099e-208")
check("flostr-goja-short-2", "" .. gflo[2], "5.1799756010409385e+21")
check("flostr-goja-short-3", "" .. gflo[3], "-9.7727201328748685e-147")
check("flostr-goja-nondigit-1", "" .. gflo[4], "1.8705482588342312e-308")
check("flostr-goja-nondigit-2", "" .. gflo[5], "-1.2658976361944994e-308")
check("flostr-goja-nondigit-3", "" .. gflo[6], "-1.3184270331886705e-308")
check("flostr-goja-short-4", "" .. gflo[7], "0.00093397298318044985")
check("flostr-window-17", "" .. gflo[8], "0.33333333333333331")
check("flostr-window-15-exp", "" .. gflo[9], "1e+15")
check("flostr-window-17-plain", "" .. gflo[10], "1234567890123456.0")
check("flostr-min-subnormal", "" .. gflo[11], "4.94065645841247e-324")
check("flostr-1e16", "" .. gflo[12], "1e+16")

-- ----- done -----
print("features: " .. checks .. " checks, " .. fails .. " failures")
exit(fails)
