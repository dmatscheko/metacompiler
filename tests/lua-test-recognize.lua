--[[
  lua-test-recognize.lua

  A real-world-looking Lua file that exercises the surface syntax of the widened
  grammar. Every construct in it is genuinely lowered now, in both grammars, so
  the file runs and exits 0 with or without -warn-unsupported, and a run with
  the flag reports no unsupported construct at all:

      mec languages/lua-to-llvm-ir.abnf tests/lua-test-recognize.lua -q

  It is an ordinary matrix entry, not a SHOULD-FAIL guard like the recognize
  tests of the other languages. (The trailing "notImplemented" notes below are
  stale: every construct runs.)
]]

-- Genuine, fully supported additions ----------------------------------------

-- Hexadecimal and exponent number literals, plus a Lua 5.4 <const> attribute.
local MASK <const> = 0xFF          -- 255
local KILO = 1e3                   -- 1000 via exponent notation
local HEXY = 0x10                  -- 16
local TINY = 2.5e-1                -- 0.25

-- A long-bracket string (verbatim, spans lines); note the leading newline is
-- dropped, matching Lua. Empty ';' statements are accepted too.
local banner = [[
=== recognize ===]];
;
print(banner)
print("MASK = " .. MASK)
print("KILO = " .. KILO)
print("HEXY = " .. HEXY)
print("TINY = " .. TINY)

--[=[ a level-1 long comment, whose body may itself contain ]] without ending ]=]

-- Further surface ------------------------------------------------------------

-- Bitwise operators and exponentiation.
local pow  = 2 ^ 10                -- notImplemented: ^
local band = MASK & HEXY           -- notImplemented: bitwise &
local bor  = 1 | 2                 -- notImplemented: bitwise |
local shl  = 1 << 4                -- notImplemented: <<
local shr  = 256 >> 2              -- notImplemented: >>
local bnot = ~0                    -- notImplemented: unary ~
print("pow", pow, "band", band, "bor", bor, "shl", shl, "shr", shr, "bnot", bnot)

-- Varargs: the parameter list accepts a trailing '...', and '...' in the body
-- yields the extra arguments.
local function firstOf(label, ...)
  local rest = ...                 -- notImplemented: varargs ...
  return rest
end
print("firstOf", firstOf("xs", 10, 20, 30))

-- goto and its target label (Lua 5.2+): the jump skips the print for i == 2.
for i = 1, 3 do
  if i == 2 then goto continue end -- notImplemented: goto
  print("i", i)
  ::continue::                     -- notImplemented: label
end

print("recognize: all constructs parsed")
