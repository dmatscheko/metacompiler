# Worked differential probes

Run one with `tests/probe.sh <lang> <file> --oracle '<cmd> %s'`. Each of these is
a template rather than a fixture: copy it, change the operand pools, and point it
at whatever you are about to change.

    tests/probe.sh python tests/probe/python-intarith.py --oracle 'python3 %s'
    tests/probe.sh lua    tests/probe/lua-float.lua      --oracle 'lua %s'

Both are green on all four legs as of `4b62d37`, which is the point: a probe that
passes is how you establish that an area is *not* where your defect lives, and
that is most of what probing is for.

**The one rule.** Every operand is read out of an ARRAY, never written as a
literal in the expression. A probe over literals measures the grammar's constant
folder and not the runtime, and this project has been fooled by exactly that.

`lua-float.lua` covers `-0.0` deliberately: unary minus spelled `0 - x` loses the
sign of zero, and that mistake has been found in `d_pow`, `d_mod_go`, Swift's
`rounded()`, Java, Dart's `math.Mod` and Ruby's interpreter. It is the single
most-repeated defect in the project's history, so it gets a standing probe.
