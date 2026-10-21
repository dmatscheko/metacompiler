-- Full-syntax test: Lua (5.4 core language) - part 2, SECTION 60: a recursion
-- with no base case. Ten thousand calls to the limit and back, several times,
-- cost the -frozen interpreter leg more than the step brake (-max-steps, 100M
-- per call) leaves the main file; see tests/lua-test-full.lua for the
-- conventions.

local failures = 0
local checks = 0

local function check(id, got, want)
    checks = checks + 1
    if got ~= want then
        print("FAIL " .. id)
        failures = failures + 1
    end
end

-- ===== SECTION 60: unbounded recursion =====
-- A call nested past the stack is the error "stack overflow", positioned at the
-- call, which pcall catches; a proper tail call never nests, and the depth a
-- caught error unwound is given back. lua 5.5 settles every value (its limit
-- is deeper than this one, which no assertion depends on).
local function s60rec(n) return 1 + s60rec(n + 1) end
local function s60deep(n) if n == 0 then return 0 end return 1 + s60deep(n - 1) end
local function s60tail(n) if n == 0 then return "done" end return s60tail(n - 1) end
local function s60()
    local got = {}
    for i = 1, 2 do
        local ok, e = pcall(s60rec, 1)
        got[#got + 1] = tostring(ok) .. " " .. string.gsub(e, "^.*:%d+: ", "")
    end
    check("so1", table.concat(got, ";"), "false stack overflow;false stack overflow")
    local co = coroutine.wrap(function() local ok, e = pcall(s60rec, 1); coroutine.yield(ok); return "end" end)
    check("so2", co() == false and co() == "end" and s60tail(12000) == "done" and s60deep(100) == 100, true)
end

-- ===== END SECTIONS =====
s60() -- SECTION-CALL 60
print("full: " .. checks .. " checks, " .. failures .. " failures")
exit(failures)
