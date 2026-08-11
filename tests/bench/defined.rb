# NOT part of the shared bench gate: tests/bench.sh globs tests/bench/mod.*, so
# this file is only measured deliberately. It exists for the same reason
# tests/bench/meth.php does - mod.rb is a WHILE loop over two locals and emits the
# `defined?` probe ZERO times, so the ruby row cannot see the one question
# docs/todo.md 5.3 asks: what a scope probe costs when it leaves the emitter for
# layer 2.
#
# The recipe (meth.php's, unchanged; the kotlin analogue is forcap.kt at a6df3a8):
#
#     rm -rf /tmp/a /tmp/b && mkdir /tmp/a /tmp/b
#     git archive <base> | tar x -C /tmp/a ; git archive <base> | tar x -C /tmp/b
#     ... copy the change into /tmp/b ...
#     cp tests/bench/defined.rb /tmp/a/tests/bench/mod.rb
#     cp tests/bench/defined.rb /tmp/b/tests/bench/mod.rb
#     (cd /tmp/a && go build -o mec .) ; (cd /tmp/b && go build -o mec .)
#     (cd /tmp/b && tests/bench.sh --draws 9 --paired /tmp/a ruby)
#
# FIVE `defined?` PER ITERATION over the four shapes the probe can take - a local
# that hits in the innermost scope, a name bound further out, a constant, and a
# name bound nowhere (the miss, which walks the whole chain to the root). That is
# far more pressure than any real Ruby program puts on it, which is the point: a
# cost that does not show here does not exist.
#
# MEASURED, 9 draws, paired against a clean 61aa3de with only this round's four
# ruby files copied in. THREE READINGS, because the shape of the split is the
# whole question and only the third one shipped:
#
#   every call routed to layer 2          +9.70%   SLOWER in 9/9 pairs
#   hot arms inline, MISS arm in layer 2  +5.45%   SLOWER in 9/9 pairs   <- shipped
#   tests/bench/mod.rb, the GATE row      +0.15%   8/9, inside its 4.14% spread
#
# So the emitter's two inline arms (js_scope_typeof, then js_scope_has for the
# innermost scope) are worth 4.25 points of the 9.70, and what is left is the
# 17,075 instructions a MISS pays for one MetaJS chain walk: 80,000 of the 200,000
# probes here answer nil, and a probe that answers nil answered nil before too, so
# that arm is charged for the correctness of the arm beside it. NOTHING IN THE
# CORPUS LOOKS LIKE THIS FILE - `defined?` occurs twice in a 899-check ratchet and
# nowhere in a loop - and the gate row above is what a program that never asks pays.
#
# The cheaper exact design was identified and NOT built, and it is written down
# here rather than lost: the only slot a Ruby program can leave holding undefined
# is a PARAMETER the caller did not supply (js_arg's answer for a missing index -
# every other binding form stores a real value, and a declared local stores nil).
# The emitter therefore KNOWS statically whether a name is a parameter of an
# enclosing def/block, so `defined?` on one could be answered with a constant and
# no probe at all - the same argument that closed python's half of todo.md 5.3 in
# cc28d07 (pyInnerNames). It needs a stack of parameter names across nested block
# emitters, which is a bigger change to ruby-to-llvm-ir.abnf than the correctness
# fix was, so it is a follow-up and not a decline.
OUTER = 7
def bench
  outer_local = 3
  s = 0
  i = 0
  while i < 40000
    inner = i % 7
    s = s + inner
    s = s + 1 if defined?(inner) == "local-variable"
    s = s + 1 if defined?(outer_local) == "local-variable"
    s = s + 1 if defined?(OUTER) == "constant"
    s = s + 1 if defined?(never_bound_anywhere).nil?
    s = s + 1 if defined?(also_not_bound).nil?
    i = i + 1
  end
  s
end
puts bench
