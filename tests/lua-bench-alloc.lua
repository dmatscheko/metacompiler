-- The allocation benchmark of docs/runtime-rework-plan.md, phase 4a.
--
-- One arithmetic statement in a loop, and nothing else: `s = s + i % 7` is the
-- smallest program that reaches layer 2 (lua-rt.metajs) for every operator, so
-- the arena growth it causes IS the per-operation allocation cost of the
-- self-hosted runtime. The answer is deterministic, so ./test.sh can compare
-- the two engines on it; tests/bench-alloc.sh runs the same file at four sizes
-- and reports bytes per iteration.
--
-- 20,000 iterations here keeps the matrix entry under a second. The numbers in
-- the plan are measured at 200,000 by tests/bench-alloc.sh.
local s = 0
for i = 1, 20000 do s = s + i % 7 end
print(s)
