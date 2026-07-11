-- Generic for over pairs(t): full key/value enumeration in insertion order.
-- (Since the language-completion pass, pairs is genuinely implemented via the
-- js_keys extern - this file used to be a not-implemented demo.) Self-checking:
-- the program exits 0 only when the pairs traversal sums the values correctly,
-- and the two engines must agree byte for byte.

local t = {a = 1, b = 2, c = 3}
local total = 0
local count = 0
for k, v in pairs(t) do
    total = total + v
    count = count + 1
end
print("pairs total: " .. total)

local fails = 0
if total ~= 6 then fails = fails + 1 end
if count ~= 3 then fails = fails + 1 end
exit(fails)
