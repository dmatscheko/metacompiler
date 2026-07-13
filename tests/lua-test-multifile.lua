-- Multi-file Lua test: vec_dot / vec_scale_x live in tests/imports/geomlib.lua and are
-- found via the -i include root (mec -i tests/imports ...). The required file is parsed
-- with the same grammar; its functions register in the shared global scope. 'math' is a
-- builtin no-op require, mixed in on purpose. exit(fails) => the run exits 0 on success.

require "math"
require "geomlib"

fails = 0

function check(cond)
    if not cond then
        fails = fails + 1
    end
end

check(vec_dot(3, 4, 2, -1) == 2)
check(vec_scale_x(3, 2) == 6)

if fails == 0 then
    print("lua multifile test passed")
end
exit(fails)
