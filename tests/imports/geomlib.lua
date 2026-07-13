-- A project module imported by tests/lua-test-multifile.lua (via -i tests/imports).
-- Its top-level functions register in the shared global scope when the file is required.

function vec_dot(ax, ay, bx, by)
    return ax * bx + ay * by
end

function vec_scale_x(x, f)
    return x * f
end
