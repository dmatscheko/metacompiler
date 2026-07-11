# Ruby subset self test 3: begin / rescue / ensure and raise.
#
# These parse but are NOT implemented (there is no exception model in the subset),
# so this file is meant to run under -warn-unsupported. With that flag every
# begin/rescue and raise is reported as "not implemented (ignored)" and the run
# continues: the begin body and any ensure body execute, the rescue handlers are
# dropped, and a raise just evaluates its expression and falls through (no
# unwinding). The self-checks below assert exactly that behavior, so the program
# still exits 0 and is byte-identical under goja and -frozen for both grammars.
#
# A DEFAULT run (without -warn-unsupported) instead aborts cleanly at the first
# begin/raise with a file:line message and a nonzero exit - that is by design.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# begin as an expression: rescue is dropped, so the value is the begin body's.
result = begin
  10 + 5
rescue
  999
end
check("begin body value", result, 15)

# ensure runs after the begin body (both bodies execute in sequence).
log = []
begin
  log << 1
ensure
  log << 2
end
check("ensure ran", log.size, 2)
check("ensure order", log[1], 2)

# raise is a no-op that still evaluates its expression, then falls through.
def make_reason
  "bad input"
end

def risky(n)
  if n < 0
    raise make_reason
  end
  n
end
check("raise falls through", risky(-7), -7)
check("normal path", risky(3), 3)

# begin/rescue with an error class and a bound variable still just runs the body.
handled = begin
  42
rescue RuntimeError => e
  -1
end
check("rescue with class dropped", handled, 42)

if fails == 0
  puts "Ruby subset self test 3 passed"
end
exit(fails)
