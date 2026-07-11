-- Exercises the generic for over pairs(), which is accepted structurally but not
-- implemented (key enumeration would need a runtime extern the subset does not have).
-- A default run aborts cleanly at the pairs loop; a -warn-unsupported run warns, skips
-- the loop body, and reaches the exit(0) below. Both engines must agree byte for byte.

local t = {a = 1, b = 2, c = 3}
local total = 0
for k, v in pairs(t) do
    total = total + v
end
print("pairs total: " .. total)
exit(0)
