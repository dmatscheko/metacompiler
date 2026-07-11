--[[
  lua-test-recognize.lua

  A real-world-looking Lua file that exercises the surface syntax the widened
  grammar newly RECOGNIZES. Some of the constructs cannot be lowered and are
  routed to notImplemented, so a plain run aborts at the first of them; run it
  with -warn-unsupported to parse the whole file and skip those constructs:

      mec languages/lua-to-llvm-ir.abnf tests/lua-test-recognize.lua -q -warn-unsupported

  This mirrors the widen tests of the other languages: it is a SHOULD-FAIL by
  default and a clean exit 0 under -warn-unsupported.
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

-- Not-yet-implemented surface (accepted, warned under -warn-unsupported) ------

-- Bitwise operators and exponentiation: the handle runtime has no bit ops and
-- no pow, so each is accepted structurally and reported as not implemented.
local pow  = 2 ^ 10                -- notImplemented: ^
local band = MASK & HEXY           -- notImplemented: bitwise &
local bor  = 1 | 2                 -- notImplemented: bitwise |
local shl  = 1 << 4                -- notImplemented: <<
local shr  = 256 >> 2              -- notImplemented: >>
local bnot = ~0                    -- notImplemented: unary ~
print("pow", pow, "band", band, "bor", bor, "shl", shl, "shr", shr, "bnot", bnot)

-- Varargs: the parameter list accepts a trailing '...'; using it in the body
-- is not implemented (the subset passes only the named parameters).
local function firstOf(label, ...)
  local rest = ...                 -- notImplemented: varargs ...
  return rest
end
print("firstOf", firstOf("xs", 10, 20, 30))

-- goto and its target label (Lua 5.2+): both accepted, not implemented, so the
-- jump is a no-op and every iteration falls through and prints.
for i = 1, 3 do
  if i == 2 then goto continue end -- notImplemented: goto
  print("i", i)
  ::continue::                     -- notImplemented: label
end

print("recognize: all constructs parsed")
