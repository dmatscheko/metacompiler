-- Full-syntax test: Lua (5.4 core language).
--
-- This file belongs to the SECOND test group (./test.sh --full): it is NOT part
-- of the default matrix. The goal of the metacompiler is to support the full
-- languages; this file is the ratchet that measures how far the lua grammars
-- are. It walks the whole practical Lua 5.4 syntax, one self-contained
-- SECTION per language area. The --full runner runs the file, and whenever a
-- grammar aborts it removes the section around the error and retries - so the
-- report lists every unsupported section, not just the first.
--
-- Conventions (shared by every *-test-full.* file):
--   - prologue (before the first SECTION marker): the check helper only
--   - each section: '-- ===== SECTION <nn>: <name> =====', top-level,
--     self-contained, no references to other sections
--   - the main chunk calls each section via a line tagged 'SECTION-CALL <nn>'
--     and prints the summary line 'full: <checks> checks, <failures> failures'
--   - the file ends with exit(failures) (exit 0 == full support, verified)
--
-- Deliberately out of scope (not syntax, or unrunnable in this harness):
-- require/modules, load, _ENV, and the standard library beyond what
-- lua-test-features.lua already uses (print, exit, string.len/sub/upper/
-- lower/rep, ipairs, pairs, next, table.insert/remove, math.floor/ceil/abs/
-- max/min). That rules out setmetatable - so metatables, the operator
-- metamethods and the <close> attribute (needs __close) stay untested - and
-- likewise error/pcall, tostring, select (varargs are counted by packing
-- into {...}) and coroutine.* (library calls, not syntax). Lua has no
-- chained comparisons (a < b < c is a type error by design), so none appear.
--
-- Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
-- code), organized after the Lua 5.4 reference manual with the ANTLR
-- grammars-v4 Lua grammar as a coverage checklist.

local failures = 0
local checks = 0

local function check(id, got, want)
    checks = checks + 1
    if got ~= want then
        print("FAIL " .. id)
        failures = failures + 1
    end
end

-- ===== SECTION 01: baseline =====
-- Condensed re-assertion of the feature-matrix basics this file builds on.
function s01()
    local n = 0
    for i = 1, 3 do n = n + i end
    check("bas1", n, 6)
    local t = {x = 1, "a"}
    t.y = t.x + 1
    check("bas2", t.y == 2 and t[1] == "a", true)
    local function add(a, b) return a + b end
    check("bas3", add(2, 3), 5)
    local w = 0
    while w < 2 do w = w + 1 end
    repeat w = w + 1 until w >= 3
    check("bas4", w, 3)
    check("bas5", (1 and "x" or "y") .. string.len("ab"), "x2")
end

-- ===== SECTION 02: numeric literal forms =====
function s02()
    check("num1", 0xff, 255)
    check("num2", 0XFF + 0x10, 271)
    check("num3", 1e3 == 1000 and 2.5e-2 == 0.025, true)
    check("num4", .5 + 3., 3.5)
    check("num5", 0x1p4, 16)            -- hex float with binary exponent
    check("num6", 0xA.8p0, 10.5)
    check("num7", 0x.8p1, 1)
    check("num8", "" .. 1 .. "," .. 1.0, "1,1.0") -- integers format apart
    check("num9", "" .. 1e2, "100.0")   -- exponent literals are floats
end

-- ===== SECTION 03: string literals and escapes =====
function s03()
    check("str1", "\65\66", "AB")       -- decimal escapes
    check("str2", "\x41", "A")          -- hex escape
    check("str3", "\u{48}\u{49}", "HI") -- unicode escapes
    check("str4", "\u{20AC}", "€")
    check("str5", string.len("a\tb\\c"), 5)
    check("str6", "it's" == 'it\'s' and '"' == "\"", true)
    local cont = "one\z
        two"                            -- \z skips the newline and indent
    check("str7", cont, "onetwo")
    local nl = "a\
b"                                      -- escaped newline stays in the string
    check("str8", nl, "a\nb")
end

-- ===== SECTION 04: long brackets: strings and comments =====
function s04()
    check("lng1", [[plain]], "plain")
    check("lng2", [==[level two]==], "level two")
    check("lng3", [=[holds ]] safely]=], "holds ]] safely")
    check("lng4", string.len([[a\nb]]), 4) -- no escape processing
    local ml = [[
line1
line2]]                                 -- a first newline is skipped
    check("lng5", ml, "line1\nline2")
    local v = 1
    --[[ v = 2
    still inside the comment ]]
    --[==[ a long comment holding ]] brackets ]==]
    check("lng6", v, 1) --[[inline]] check("lng7", v + 1, 2)
end

-- ===== SECTION 05: blocks, scoping and semicolons =====
function s05()
    local x = 1
    local seen = 0
    do
        local x = 2                     -- shadows the outer x
        seen = x
    end
    check("blk1", x == 1 and seen == 2, true)
    local v = 1
    local v = v + 1                     -- redeclaration reads the old local
    check("blk2", v, 2)
    ;
    local y = 0; ; y = y + 1;
    check("blk3", y, 1);
    local n = 0
    repeat local step = 2; n = n + step until step == 2 and n >= 4 -- until sees body locals
    check("blk4", n, 4)
    local reached = 0
    while true do
        do reached = reached + 1; break end -- break inside a nested do-block
    end
    check("blk5", reached, 1)
end

-- ===== SECTION 06: multiple assignment and multiple returns =====
function s06()
    local function two() return 10, 20 end
    local function three() return 1, 2, 3 end
    local a, b = two()
    check("mul1", a + b, 30)
    local c, d, e = two()               -- extended with nil
    check("mul2", c == 10 and d == 20 and e == nil, true)
    local f = two()                     -- truncated to the first value
    check("mul3", f, 10)
    local g, h = two(), 5               -- a call before the end yields one value
    check("mul4", g == 10 and h == 5, true)
    check("mul5", (three()), 1)         -- parentheses truncate to one value
    local t = {three()}                 -- expands at the end of a constructor
    local u = {three(), 9}              -- truncated elsewhere
    check("mul6", #t == 3 and t[3] == 3 and #u == 2 and u[2] == 9, true)
    local function pass() return three() end
    local p, q, r = pass()              -- return propagates all values
    check("mul7", p + q + r, 6)
    local m = {}
    m[1], m.k = two()                   -- fields as assignment targets
    check("mul8", m[1] + m.k, 30)
    local i, j, k = 1, 2, 3
    i, j, k = k, i, j                   -- all right sides evaluated first
    check("mul9", "" .. i .. j .. k, "312")
end

-- ===== SECTION 07: goto and labels =====
function s07()
    local n = 0
    ::top::
    n = n + 1
    if n < 3 then goto top end          -- backward jump forms a loop
    check("lbl1", n, 3)
    local kept = ""
    for i = 1, 4 do
        if i % 2 == 0 then goto continue end
        kept = kept .. i
        ::continue::                    -- the continue pattern
    end
    check("lbl2", kept, "13")
    local hits = 0
    for i = 1, 3 do
        for j = 1, 3 do
            hits = hits + 1
            if i == 2 and j == 1 then goto done end
        end
    end
    ::done::                            -- one jump leaves both loops
    check("lbl3", hits, 4)
    local path = ""
    do path = path .. "x"; goto after end -- forward jump out of a do-block
    ::after::
    check("lbl4", path .. "y", "xy")
end

-- ===== SECTION 08: numeric for refinements =====
function s08()
    local cnt, sum = 0, 0
    for i = 1, 2, 0.5 do cnt = cnt + 1; sum = sum + i end -- float step
    check("for1", cnt == 3 and sum == 4.5, true)
    local down = 0
    for i = 2, 1, -0.5 do down = down + 1 end
    check("for2", down, 3)
    local m = 0
    for i = 1, 3.5 do m = m + i end     -- float limit, integer control
    check("for3", m, 6)
    local z = 0
    for i = 1, 0.5 do z = z + 1 end     -- zero iterations
    check("for4", z, 0)
    local runs = 0
    for i = 1, 3 do runs = runs + 1; i = i + 10 end -- assigning i does not steer it
    check("for5", runs, 3)
end

-- ===== SECTION 09: generic for and custom iterators =====
function s09()
    local function odds(t, i)           -- a stateless iterator triple
        i = i + 2
        if t[i] == nil then return nil end return i, t[i]
    end
    local data = {10, 20, 30, 40, 50}
    local sum = 0
    for i, v in odds, data, -1 do sum = sum + v end
    check("gen1", sum, 90)
    local idx = ""
    for i in odds, data, -1 do idx = idx .. i end
    check("gen2", idx, "135")
    local function countdown(n)         -- a stateful closure iterator
        return function() n = n - 1; if n >= 0 then return n end end
    end
    local cs = ""
    for v in countdown(3) do cs = cs .. v end
    check("gen3", cs, "210")
    local function tri(s, c)            -- three loop variables
        if c >= 2 then return nil end return c + 1, c * 10, s
    end
    local total = 0
    for a, b, c in tri, 7, 0 do total = total + a + b + c end
    check("gen4", total, 27)
    local seen = ""
    for k, v in pairs({9}) do seen = seen .. k .. v end
    check("gen5", seen, "19")
end

-- ===== SECTION 10: local attributes =====
-- <close> needs a __close metamethod, i.e. setmetatable: not testable here.
function s10()
    local limit <const> = 10
    check("att1", limit, 10)
    local greet <const> = "hi " .. "there"
    check("att2", greet, "hi there")
    local base <const>, counter = 100, 0
    counter = counter + base            -- only the tagged name is constant
    check("att3", counter, 100)
    local f <const> = function(x) return x + limit end
    check("att4", f(5), 15)
end

-- ===== SECTION 11: integer and float arithmetic =====
function s11()
    check("ift1", 1 == 1.0 and 3 == 3.0, true)
    check("ift2", 7 / 2, 3.5)
    check("ift3", "" .. (8 / 2), "4.0") -- division always yields a float
    check("ift4", "" .. (7 // 2), "3")  -- integer // integer stays integer
    check("ift5", "" .. (7.0 // 2), "3.0")
    check("ift6", 7.5 // 2 == 3.0 and -7.5 // 2 == -4.0, true)
    check("ift7", 7.5 % 2 == 1.5 and -0.5 % 2 == 1.5, true)
    check("ift8", "" .. (1 + 0.0), "1.0") -- mixed arithmetic goes float
    check("ift9", (1 / 0 > 1e308) and (0 / 0 ~= 0 / 0), true) -- inf and NaN
end

-- ===== SECTION 12: bitwise operators =====
function s12()
    check("bit1", (5 & 3) + (5 | 3), 8)
    check("bit2", 5 ~ 3, 6)             -- binary ~ is xor
    check("bit3", ~0, -1)               -- unary ~ is bitwise not
    check("bit4", ~5 & 7, 2)            -- unary binds tighter than &
    check("bit5", (1 << 4) + (256 >> 4), 32)
    check("bit6", -1 >> 63, 1)          -- shifts are logical, 64-bit
    check("bit7", 1 << 64, 0)           -- oversized shifts go to zero
    check("bit8", 1 | 2 & 3, 3)         -- & binds tighter than |
    check("bit9", (1 << 2 + 1) + (3.0 & 1), 9) -- + before shifts; floats convert
end

-- ===== SECTION 13: concatenation, length and coercions =====
function s13()
    check("cat1", "x" .. 1 + 2, "x3")   -- + binds tighter than ..
    check("cat2", 1 .. 2, "12")
    check("cat3", "v=" .. -2.5, "v=-2.5")
    check("cat4", #"hello" + #"", 5)    -- # works on strings too
    check("cat5", #"ab" .. "c", "2c")   -- unary # binds tighter than ..
    check("cat6", "10" + 1, 11)         -- strings coerce in arithmetic
    check("cat7", "2" * "3", 6)
    check("cat8", "0x10" + 0 == 16 and "3.5" + 0 == 3.5, true)
    check("cat9", #{7, 8, 9}, 3)
end

-- ===== SECTION 14: relational, logical and power operators =====
-- Lua has no chained comparisons: a < b < c is deliberately absent.
function s14()
    check("rel1", (5 > 3) and "yes" or "no", "yes") -- the ternary idiom
    check("rel2", (5 < 3) and "yes" or "no", "no")
    check("rel3", nil or false or "third", "third")
    check("rel4", 1 < 1.5 and 2.0 <= 2, true)       -- mixed int/float compare
    check("rel5", "A" < "a" and "ab" < "b", true)   -- byte order
    check("rel6", not 1 == false, true)             -- not binds tighter than ==
    check("rel7", not not "", true)                 -- only nil/false are falsy
    check("rel8", 2 ^ 10 == 1024 and 2 ^ 3 ^ 2 == 512, true) -- right-assoc power
    check("rel9", -2 ^ 2, -4)                       -- ^ before unary minus
    check("rel10", "" .. 2 ^ -1, "0.5")             -- unary in the exponent
end

-- ===== SECTION 15: table constructors =====
function s15()
    local t = {1, 2; 3, 4,}             -- ',' and ';' both separate fields
    check("tbl1", #t == 4 and t[3] == 3, true)
    local r = {x = 1, ["y z"] = 2, [3 + 4] = "seven"}
    check("tbl2", r.x == 1 and r["y z"] == 2 and r[7] == "seven", true)
    local mixed = {"a", k = "b", [10] = "c", "d"}
    check("tbl3", mixed[1] == "a" and mixed[2] == "d" and mixed.k == "b", true)
    check("tbl4", mixed[10], "c")       -- explicit keys do not shift the array part
    local kw = {["end"] = 1, ["function"] = 2}
    check("tbl5", kw["end"] + kw["function"], 3)
    local nested = {{1}, {2, {3}}}
    check("tbl6", nested[2][2][1], 3)
    local bt = {[true] = "T", [false] = "F"}
    check("tbl7", bt[true] .. bt[1 == 2], "TF")
    local fk = {}
    fk[2.0] = "two"                     -- float keys normalize to integers
    check("tbl8", fk[2], "two")
    local one = {5;}                    -- trailing semicolon
    check("tbl9", one[1], 5)
end

-- ===== SECTION 16: function call forms =====
function s16()
    local function len(s) return string.len(s) end
    local function pick(t) return t[2] end
    check("cal1", len "hello", 5)       -- string argument without parentheses
    check("cal2", len [[ab]], 2)        -- long string argument
    check("cal3", pick {7, 8, 9}, 8)    -- table constructor argument
    local box = {n = 3}
    function box:grow(d) self.n = self.n + d; return self end
    function box:label(s) return s .. self.n end
    check("cal4", box:grow(2):grow(5).n, 10)   -- chained method calls
    check("cal5", box:label "n=", "n=10")      -- method + string sugar
    check("cal6", box.grow(box, 1).n, 11)      -- colon is only sugar
    local lib = {geom = {}}
    function lib.geom.area(w, h) return w * h end
    lib.geom.unit = 5
    function lib.geom:scaled(f) return self.unit * f end
    check("cal7", lib.geom.area(6, 7), 42)     -- nested dotted definition
    check("cal8", lib.geom:scaled(3), 15)      -- nested colon definition
    check("cal9", (function(x) return x * 2 end)(21), 42)
end

-- ===== SECTION 17: varargs =====
function s17()
    local function count(...) return #{...} end
    check("var1", count(), 0)
    check("var2", count(7), 1)
    check("var3", count(1, 2, 3), 3)
    local function sum(...)
        local t, s = {...}, 0
        for i = 1, #t do s = s + t[i] end
        return s
    end
    check("var4", sum(1, 2, 3), 6)
    local function through(...) return sum(...) end
    check("var5", through(4, 5), 9)     -- forwarding ...
    local function firstTwo(...)
        local a, b = ...                -- ... on the right of an assignment
        return "" .. a .. b
    end
    check("var6", firstTwo(3, 4, 5), "34")
    local function labelled(tag, ...) return tag .. sum(...) end -- fixed param first
    check("var7", labelled("t", 1, 2), "t3")
    local function midpack(...)         -- ... not last: only its first value is taken
        local t = {..., 99}
        return #t .. ":" .. t[1]
    end
    check("var8", midpack(5, 6), "2:5")
    local function ident(...) return ... end
    local a, b = ident(30, 12)
    check("var9", a + b, 42)
end

-- ===== SECTION 18: closures, upvalues and tail calls =====
function s18()
    local function makePair()
        local n = 0
        local function inc() n = n + 1; return n end
        local function get() return n end
        return inc, get
    end
    local inc, get = makePair()
    inc(); inc()
    check("clo1", get(), 2)             -- both closures share the upvalue
    local fx = function() return fx end
    check("clo2", fx(), nil)            -- a plain local is invisible to its initializer
    local function fy() return fy end
    check("clo3", fy() == fy, true)     -- but 'local function' sees itself
    local function countTail(n)         -- the recursive call is a tail call
        if n == 0 then return "done" end return countTail(n - 1)
    end
    check("clo4", countTail(200), "done")
    local function outer()
        local hidden = 21
        return function() return function() return hidden * 2 end end
    end
    check("clo5", outer()()(), 42)      -- upvalue through two nesting levels
end

-- ===== END SECTIONS =====

s01() -- SECTION-CALL 01
s02() -- SECTION-CALL 02
s03() -- SECTION-CALL 03
s04() -- SECTION-CALL 04
s05() -- SECTION-CALL 05
s06() -- SECTION-CALL 06
s07() -- SECTION-CALL 07
s08() -- SECTION-CALL 08
s09() -- SECTION-CALL 09
s10() -- SECTION-CALL 10
s11() -- SECTION-CALL 11
s12() -- SECTION-CALL 12
s13() -- SECTION-CALL 13
s14() -- SECTION-CALL 14
s15() -- SECTION-CALL 15
s16() -- SECTION-CALL 16
s17() -- SECTION-CALL 17
s18() -- SECTION-CALL 18
print("full: " .. checks .. " checks, " .. failures .. " failures")
exit(failures)
