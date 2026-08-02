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
-- max/min) plus the five names through which Lua 5.3+ makes the integer
-- subtype visible at all - math.type/tointeger/fmod/maxinteger/mininteger,
-- without which sections 21 and 22 could not be written. That rules out setmetatable - so metatables, the operator
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
    -- A long literal holding non-ASCII text did not PARSE in either grammar: the
    -- :script that scans it built its token one c.peek BYTE at a time, and a byte above
    -- 127 appended as its own code point is re-encoded as two bytes on the way into Go,
    -- so the token no longer matched the input. It decodes UTF-8 by hand now.
    check("lng6", [[héllo]], "héllo")
    check("lng7", #[[héllo]], 6)           -- and a long literal is a BYTE string too
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

-- ===== SECTION 19: interpreter/compiler agreement ratchet =====
-- Shifts are 64-BIT and logical in Lua. lua-to-llvm-ir.abnf emitted JavaScript's
-- 32-bit js_shl / js_ushr (with the count masked to 5 bits), so 1 << 32 was 1 and
-- 255 << 24 was -16777216 while lua-interpreter.abnf and real Lua 5.4 answered
-- 4294967296 and 4278190080. Values below are from /opt/homebrew/bin/lua.
local function s19()
  check("agr1", 1 << 32, 4294967296)
  check("agr2", 1 << 33, 8589934592)
  check("agr3", 255 << 24, 4278190080)
  check("agr4", 0xFFFFFFFF << 4, 68719476720)
  check("agr5", 1 >> 1, 0)
  check("agr6", 1 << 64, 0)
  check("agr7", 1 << -1, 0)
  check("agr8", 4294967296 >> 32, 1)
  check("agr9", 0xFF & 0x0F, 15)
  check("agr10", 0xFF | 0x100, 511)
  check("agr11", ~0, -1)
end
-- ===== SECTION 20: the syntax the reference corpus asked for =====
-- Every check below is a construct Lua's OWN test suite writes and neither
-- grammar could parse. They took the lua-tests corpus from 54.5% to 100.0% in
-- both halves, so this section is the ratchet that keeps them.
local function s20()
  -- A target is Lua's `var`, so a CALL may sit in front of the final field or
  -- index: getmetatable(t).__index = f is the shape the suite uses 5 times.
  local function id(t) return t end
  local a = {}
  id(a).x = 7
  check("corp01", a.x, 7)
  id(a)["y"] = 8
  check("corp02", a.y, 8)

  -- The same inside a multiple assignment, mixed with plain targets.
  local b, c = {}, {}
  local g
  a[1], id(b)[2], g = 10, 20, 30
  check("corp03", a[1], 10)
  check("corp04", b[2], 20)
  check("corp05", g, 30)

  -- And with a METHOD call on the way (o:m().x = v).
  local o = {t = {}}
  o.m = function(self) return self.t end
  o:m().z = 9
  check("corp06", o.t.z, 9)

  -- A chain of two calls, and a call seeded from a field.
  local outer = {inner = function() return a end}
  outer.inner().w = 11
  check("corp07", a.w, 11)

  -- A bare `return` is followed by a block terminator, not by an expression.
  -- `end`, `else`, `elseif` and `until` are ordinary identifiers to the Id rule,
  -- so `return end` used to parse as "return the variable named end" and the
  -- block then ate the NEXT `end`: the parse slid one keyword out of step and
  -- died at EOF, four files later.
  local function bareIf(n)
    if n == 0 then return end
    return n
  end
  check("corp08", bareIf(0), nil)
  check("corp09", bareIf(5), 5)

  local function bareElse(n)
    if n == 0 then
      return
    else
      return n
    end
  end
  check("corp10", bareElse(0), nil)
  check("corp11", bareElse(3), 3)

  local function bareElseif(n)
    if n == 0 then
      return
    elseif n == 1 then
      return
    end
    return n
  end
  check("corp12", bareElseif(1), nil)
  check("corp13", bareElseif(4), 4)

  -- A bare return closing a while, a repeat/until, a numeric for and a do block.
  local function bareWhile()
    while true do return end
  end
  check("corp14", bareWhile(), nil)

  local function bareRepeat()
    repeat return until true
  end
  check("corp15", bareRepeat(), nil)

  local function bareFor()
    for i = 1, 3 do return end
  end
  check("corp16", bareFor(), nil)

  local function bareDo()
    do return end
  end
  check("corp17", bareDo(), nil)

  -- A bare return as the last statement of an anonymous function that is itself
  -- a call argument - `f(function () ... return end)` - which is the exact shape
  -- coroutine.lua dies on: the closing `)` had nothing left to close.
  local function apply(f) return f() end
  check("corp18", apply(function () return end), nil)
  check("corp19", apply(function () if false then return end return 6 end), 6)

  -- A bare return still takes its values when there ARE any on the same line.
  local function two() return 1, 2 end
  local x, y = two()
  check("corp20", x, 1)
  check("corp21", y, 2)
end

-- ===== SECTION 21: the integer / float subtype =====
function s21()
    check("sub1", math.type(3), "integer")
    check("sub2", math.type(3.0), "float")
    check("sub3", math.type("3"), nil)   -- a string is not a number at all
    check("sub4", math.type(true), nil)
    check("sub5", "" .. 3 .. "|" .. 3.0, "3|3.0")
    check("sub6", math.type(7 / 2), "float")
    check("sub7", math.type(6 / 2), "float") -- '/' is ALWAYS a float
    check("sub8", "" .. (6 / 2), "3.0")
    check("sub9", math.type(7 // 2) .. math.type(7.0 // 2), "integerfloat")
    check("sub10", math.type(7 % 3) .. math.type(7.5 % 2), "integerfloat")
    check("sub11", math.type(2 ^ 31), "float") -- '^' is ALWAYS a float
    check("sub12", "" .. 1e3 .. "|" .. math.type(1e3), "1000.0|float")
    check("sub13", 1 == 1.0 and math.type(1) ~= math.type(1.0), true)
    check("sub14", math.type(math.floor(3.7)) .. math.type(math.ceil(3.2)), "integerinteger")
    check("sub15", math.type(#"abc"), "integer")
    check("sub16", math.tointeger(3.0), 3)
    check("sub17", math.tointeger(3.5), nil)
    check("sub18", math.tointeger("8"), 8)
    check("sub19", math.tointeger(true), nil)
    check("sub20", math.type(math.abs(-1.0)), "float")
    check("sub21", math.type(math.abs(-1)), "integer")
    check("sub22", math.fmod(7, -2) .. "|" .. math.type(math.fmod(7, -2)), "1|integer")
    check("sub23", math.type("10" + 5) .. math.type("10.0" + 5), "integerfloat")
    check("sub24", math.type(math.max(1, 1.0)), "integer") -- max keeps its argument
    check("sub25", "" .. (7 // -2) .. (-7 // -2) .. (7 % -2) .. (-7 % -2), "-43-1-1")
end

-- ===== SECTION 22: exact 64-bit integers =====
function s22()
    check("i64a", "" .. math.maxinteger, "9223372036854775807")
    check("i64b", "" .. math.mininteger, "-9223372036854775808")
    check("i64c", math.maxinteger + 1 == math.mininteger, true)
    check("i64d", math.mininteger - 1 == math.maxinteger, true)
    check("i64e", "" .. (math.maxinteger * 2), "-2")
    check("i64f", "" .. 9223372036854775807, "9223372036854775807")
    check("i64g", math.type(9223372036854775808), "float") -- too big: a float
    check("i64h", 0xffffffffffffffff, -1)                  -- hex WRAPS
    check("i64i", 0x7fffffffffffffff == math.maxinteger, true)
    check("i64j", 0x10000000000000000, 0)
    check("i64k", 1 << 62, 4611686018427387904)
    check("i64l", "" .. (1 << 63), "-9223372036854775808")
    check("i64m", (1 << 64) + (1 << -1), 0)
    check("i64n", -1 >> 1 == math.maxinteger, true)         -- >> is LOGICAL
    check("i64o", ~0, -1)
    check("i64p", math.mininteger // -1 == math.mininteger, true)
    check("i64q", math.mininteger % -1, 0)
    check("i64r", -math.mininteger == math.mininteger, true)
    check("i64s", math.abs(math.mininteger) == math.mininteger, true)
    check("i64t", math.maxinteger // 1 == math.maxinteger, true)
    check("i64u", math.type(math.maxinteger // 1), "integer")
    check("i64v", math.tointeger(2 ^ 53), 9007199254740992)
    check("i64w", math.tointeger(2 ^ 63), nil)              -- out of range
    check("i64x", math.max(math.maxinteger, math.maxinteger - 1) == math.maxinteger, true)
    check("i64y", math.floor(math.maxinteger) == math.maxinteger, true)
    -- Table keys: a float with an exact integer value IS the integer key, and two
    -- neighbouring 64-bit keys stay apart (their double readings would not).
    local t = {}
    t[1] = "a"
    t[2.0] = "b"
    t[math.maxinteger] = "M"
    t[math.maxinteger - 1] = "N"
    check("i64z1", t[1.0] .. t[2], "ab")
    check("i64z2", t[math.maxinteger] .. t[math.maxinteger - 1], "MN")
    -- A loop that runs up to the last integer stops on the exact value.
    local n = 0
    for i = math.maxinteger - 2, math.maxinteger do n = n + 1 end
    check("i64z3", n, 3)
    -- Comparison past 2^53 is exact.
    check("i64z4", math.maxinteger > math.maxinteger - 1, true)
    check("i64z5", math.maxinteger == math.maxinteger - 1, false)
end

-- ===== SECTION 23: the float number model, four engines =====
-- ./test.sh compares each ENGINE against itself, so it is structurally blind to
-- lua-interpreter.abnf, the Go compiler runtime (abnf/jsrtlua.go) and the NATIVE
-- binary (languages/lib/runtime.c + languages/lib/lua-rt.metajs) computing a
-- float differently. These assertions are the pin for that, and every value below
-- was measured across all three AND against the installed lua 5.5.
--
-- The '%' cases are the reproducing ones. Lua's float modulo is luai_nummod -
-- fmod, then one correction - and it was written in all three halves as
-- a - floor(a/b)*b, which agrees only while a/b fits 53 bits. Two separate
-- defects sat behind that:
--   - real lua answers 10 % 0.1 with 0.09999999999999945; the floor form gives 0
--   - on arm64 Go CONTRACTS a - floor(a/b)*b into a fused multiply-subtract,
--     which goja and the C floor cannot do, so the two Go halves of this one
--     language disagreed in 128 of a 12,557-line differential probe
-- The comparisons against the floor form are deliberate: they are what fails if
-- anyone puts the old formula back.
function s23()
    local x = 10.0
    local y = 3.14159265358979
    check("flo1", x % y == math.fmod(x, y), true)
    check("flo2", x % y == x - math.floor(x / y) * y, false)
    check("flo3", 10 % 0.1, 0.09999999999999945)
    check("flo4", 10 - math.floor(10 / 0.1) * 0.1, 0.0)
    -- The float modulo takes the sign of the DIVISOR, unlike math.fmod.
    check("flo5", -0.5 % 2, 1.5)
    check("flo6", math.fmod(-0.5, 2), -0.5)
    check("flo7", 5.5 % -2, -0.5)
    check("flo8", math.fmod(5.5, -2), 1.5)
    check("flo9", math.type(2^53 % 3), "float")
    -- The shortest round-tripping spelling, which the C floor's own formatter has
    -- to reach digit for digit. (Real lua prints %.14g/%.17g, so it spells the
    -- same VALUES with more digits - a formatter difference both halves share and
    -- section 23 does not try to hide: 1/3 is 0.33333333333333331 there.)
    check("flo10", "" .. (x % y), "0.57522203923063")
    check("flo11", "" .. (1 / 3), "0.3333333333333333")
    check("flo12", "" .. (0.1 + 0.2), "0.30000000000000004")
    -- The non-finite spellings are Lua's, not JavaScript's.
    check("flo13", ("" .. (1 / 0)) .. " " .. ("" .. (-1 / 0)) .. " " .. ("" .. (0 / 0)), "inf -inf nan")
    -- An integral float always carries its fraction; an integer never does.
    check("flo14", ("" .. (1.0)) .. " " .. ("" .. (1)), "1.0 1")
    -- math.type separates the subtypes, and / is always a float.
    check("flo15", math.type(4 / 2), "float")
    check("flo16", math.type(4 // 2), "integer")
    check("flo17", math.type(4.0 // 2), "float")
    -- Integer overflow wraps; the float path does not.
    check("flo18", math.maxinteger + 1 == math.mininteger, true)
    check("flo19", math.maxinteger + 1.0 == math.mininteger, false)
    -- A string coerces with its own spelling deciding the subtype.
    check("flo20", math.type("10" + 0), "integer")
    check("flo21", math.type("10.0" + 0), "float")
    check("flo22", math.type("0x10" + 0), "integer")
    check("flo23", "0xffffffffffffffff" + 0, -1)
    check("flo24", math.type("9223372036854775808" + 0), "float")
end

-- ===== SECTION 24: the sized integer, as a TYPE =====
-- Section 22 pins the exact 64-bit VALUES; this one pins the TYPE that carries
-- them. An integer outside +-2^53 cannot live in a double, so both halves put it
-- in a sized integer: abnf/jsrtint.go's jsGInt in the Go runtime, tag 13 of
-- languages/lib/runtime.c in the native binary (added 2026-08 - before that the
-- native half had only an object box, for which js_typeof answered "object"
-- where the Go twin answers "number").
--
-- MEASURED DISCRIMINATING POWER, and it is the finding rather than a weakness:
-- 0 of these 67 fail against a clean archive of 660c47a, in all three engines.
-- No LUA-LEVEL program can tell the object box from the tag - which is exactly
-- the "Lua survived by luck of ordering" result of the phase-4 probe, restated
-- as a measurement. What they do catch is the conversion's own failure modes:
-- against a build whose js_lucmp asks `typeof` instead of luPlain, 2 fail;
-- against one whose js_lutype does, 3 fail. Every one of the 67 is also checked
-- against the installed lua 5.5.
function s24()
    -- The 2^53 boundary. Below it a Lua integer is one double; at and above it
    -- the runtime carries a SIZED INTEGER (abnf/jsrtint.go's jsGInt in the Go
    -- half, tag 13 of languages/lib/runtime.c in the native one). Both sides of
    -- the boundary are the same Lua type, and that is the first thing to pin.
    local e = 9007199254740992          -- 2^53
    check("si01", math.type(e - 1), "integer")
    check("si02", math.type(e), "integer")
    check("si03", math.type(e + 1), "integer")
    check("si04", math.type(math.maxinteger), "integer")
    check("si05", math.type(math.mininteger), "integer")
    -- ... and the FLOAT of the same magnitude is still a float, so the tag has
    -- not swallowed the subtype distinction.
    check("si06", math.type(e + 0.0), "float")
    check("si07", math.type(9007199254740993.0), "float")

    -- EXACTNESS past 2^53: the neighbours of 2^53 share one double, so an
    -- implementation that reads a big integer as a double answers these wrong.
    check("si08", e + 1 == e, false)
    check("si09", e + 1 == e + 2, false)
    check("si10", "" .. (e + 1), "9007199254740993")
    check("si11", "" .. (e * 2 + 1), "18014398509481985")
    check("si12", (e + 1) - (e - 1), 2)
    check("si13", (e + 3) // 2 == e // 2 + 1, true)
    check("si14", (e + 1) % 2, 1)
    check("si15", (e + 2) % 2, 0)

    -- ORDERED comparison past 2^53. This is the site that reads its operands as
    -- doubles unless it is told not to: 2^53+1 and 2^53+2 are the SAME double.
    check("si16", e + 1 < e + 2, true)
    check("si17", e + 2 <= e + 1, false)
    check("si18", math.maxinteger - 1 < math.maxinteger, true)
    check("si19", math.mininteger < math.mininteger + 1, true)
    check("si20", (e + 1) <= (e + 1), true)

    -- WIDTH and SIGNEDNESS. Lua's integer is 64 bit and SIGNED, and every
    -- arithmetic operator wraps at that width rather than saturating or
    -- promoting - which is giTrunc(v, 64, false) in the specification.
    check("si21", math.maxinteger + 1 == math.mininteger, true)
    check("si22", math.mininteger - 1 == math.maxinteger, true)
    check("si23", "" .. (math.maxinteger * 2), "-2")
    check("si24", "" .. (math.maxinteger + math.maxinteger), "-2")
    check("si25", math.mininteger // -1 == math.mininteger, true)  -- the one that cannot be represented
    check("si26", math.mininteger % -1, 0)
    check("si27", -math.mininteger == math.mininteger, true)       -- unary minus wraps too
    check("si28", math.maxinteger < 0, false)                      -- SIGNED, not unsigned
    check("si29", math.mininteger < 0, true)
    check("si30", math.abs(math.mininteger) == math.mininteger, true)

    -- The bitwise operators read the same 64 bits, and '>>' is LOGICAL (Lua
    -- shifts an unsigned reading) while the arithmetic is signed.
    check("si31", -1 >> 1 == math.maxinteger, true)
    check("si32", 1 << 63 == math.mininteger, true)
    check("si33", "" .. (1 << 62), "4611686018427387904")
    check("si34", ~0, -1)
    check("si35", (math.maxinteger & math.mininteger), 0)
    check("si36", (math.maxinteger | math.mininteger) == -1, true)
    check("si37", math.maxinteger ~ math.mininteger == -1, true)
    check("si38", "" .. (0xffffffffffffffff), "-1")

    -- The STRING form is the exact decimal text, never the double's rounding.
    check("si39", "" .. math.maxinteger, "9223372036854775807")
    check("si40", "" .. math.mininteger, "-9223372036854775808")
    check("si41", #("" .. (e + 1)), 16)
    check("si42", ("" .. (e + 1)) .. "|", "9007199254740993|")

    -- ORDERING INDEPENDENCE. The runtime renders a value by asking a series of
    -- type questions. When a big integer was an OBJECT box the "is it a table"
    -- question had to be asked last or a big integer printed as a table; the
    -- sized-integer tag answers "number" to the type question, so the order no
    -- longer matters. These pin the answers the order used to decide.
    check("si43", math.type(e + 1) ~= nil, true)      -- it IS a number, not a table
    check("si44", (e + 1) == (e + 1), true)
    local t = {}
    t[e + 1] = "a"
    t[e + 2] = "b"
    t[e] = "c"
    check("si45", t[e + 1] .. t[e + 2] .. t[e], "abc")   -- three DISTINCT keys
    -- t[2^53 + 1.0] is t[2^53]: the float addition rounds back to 2^53 and an
    -- integral float key NORMALISES to that integer. Both halves of the key rule
    -- in one line - measured against lua 5.5, which answers "c" here.
    check("si46", t[e + 1.0], "c")
    t[math.maxinteger] = "m"
    check("si47", t[math.maxinteger], "m")
    check("si48", t[math.maxinteger - 1], nil)
    -- math.tointeger / math.floor keep the exact value rather than routing it
    -- through a double.
    check("si49", math.tointeger(e + 1) == e + 1, true)
    check("si50", math.floor(math.maxinteger) == math.maxinteger, true)
    check("si51", math.max(math.maxinteger, e) == math.maxinteger, true)
    check("si52", math.min(math.mininteger, e) == math.mininteger, true)
    -- The consumers that genuinely want a DOUBLE - a string index, a repeat
    -- count, the exponent of '^' - must get one out of a sized integer rather
    -- than the integer itself.
    check("si53", string.sub("abcdefgh", math.maxinteger), "")
    check("si54", string.sub("abcdefgh", 1, math.maxinteger), "abcdefgh")
    check("si55", string.sub("abcdefgh", math.mininteger), "abcdefgh")
    check("si56", string.rep("ab", e - e), "")
    check("si57", 2.0 ^ 53 == e + 0.0, true)
    local n = 0
    for i = 1, 3, math.maxinteger do n = n + 1 end
    check("si58", n, 1)
    local m = 0
    for i = math.maxinteger - 1, math.maxinteger do m = m + 1 end
    check("si59", m, 2)

    -- A NaN operand makes all four ordered comparisons false, and an infinity
    -- orders against every integer. Both go through the same path a sized
    -- integer does, so they pin that the fast path's fall-through is a real
    -- fall-through and not an "equal" answer.
    local nan = 0 / 0
    local inf = 1 / 0
    check("si60", nan < nan or nan <= nan or nan > nan or nan >= nan, false)
    check("si61", nan < math.maxinteger or nan >= math.maxinteger, false)
    check("si62", math.maxinteger < inf and math.mininteger > -inf, true)
    check("si63", math.maxinteger <= math.maxinteger and math.maxinteger >= math.maxinteger, true)
    check("si64", (e + 1) <= (e + 2) and not ((e + 2) <= (e + 1)), true)
    check("si65", math.type((e * 2 + 1) // 2), "integer")
    check("si66", math.type((e * 2 + 1) % 2), "integer")
    check("si67", "" .. ((e * 2 + 1) // 2), "9007199254740992")
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
s19() -- SECTION-CALL 19
s20() -- SECTION-CALL 20
s21() -- SECTION-CALL 21
s22() -- SECTION-CALL 22
s23() -- SECTION-CALL 23
s24() -- SECTION-CALL 24
print("full: " .. checks .. " checks, " .. failures .. " failures")
exit(failures)
