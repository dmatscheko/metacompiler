local xs = {0.0, 1.5, -1.5, 0.1, 100.0, -0.0}
local ys = {1.0, 3.0, 7.0, 0.5}
for i = 1, #xs do
  for j = 1, #ys do
    local a = xs[i]
    local b = ys[j]
    print(a, b, a + b, a * b, a / b, a % b)
  end
end
