# Ruby begin/rescue/else/ensure and raise, genuinely executed (interpreter and
# compiler).
#
# A raise unwinds through calls to the first rescue (exception types are parsed but not
# discriminated), which binds the value with `=> e`; else runs only when the body raised
# nothing; ensure always runs. A return/break/next leaving a begin body works in both
# engines. (Exceptions are raised as plain values here - integers, strings and a hash -
# since builtin exception classes are outside the subset.)
#
# The program runs top to bottom and ends with exit(fails), so it exits 0 exactly when
# every check passes; the interpreter and compiler must agree.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

def risky(n)
  if n > 3
    raise n
  end
  n * 2
end

# raise a hash from a nested method, so the rescue can read a field off the value.
def raise_info
  raise({"code" => 42})
end

# return out of a begin body, and out of a rescue body.
def classify(n)
  begin
    if n > 0
      return n * 10
    end
    raise 0
  rescue => e
    return -1
  ensure
    # ensure always runs; keep no control flow in it.
  end
end

# A return out of an INNER begin propagates through the OUTER begin.
def nested_return
  begin
    begin
      return 9
    ensure
    end
  ensure
  end
  0
end

# break leaving a begin body inside a loop.
def loop_break
  total = 0
  i = 0
  while i < 10
    begin
      break if i == 3
      total = total + i
    ensure
    end
    i = i + 1
  end
  total            # 0+1+2 = 3
end

# next leaving a begin body inside a loop (increment BEFORE the begin so next still
# makes progress).
def loop_next
  total = 0
  i = -1
  while i < 4
    i = i + 1
    begin
      next if i == 2
      total = total + i
    rescue => e
    end
  end
  total            # 0+1+3+4 = 8
end

# the no-raise path: the body value flows out and the rescue is skipped.
def safe_div(a, b)
  begin
    a / b
  rescue => e
    -1
  end
end

# else runs only when the begin body raised nothing.
def with_else(n)
  tag = 0
  begin
    if n < 0
      raise n
    end
    tag = 1
  rescue => e
    tag = 2
  else
    tag = tag + 10
  end
  tag              # n>=0: 11 (else ran); n<0: 2 (rescue ran, else skipped)
end

# basic begin / rescue / ensure with an ordered log: the body runs up to the raise,
# the rescue runs, and ensure runs last.
log = ""
begin
  log = log + "a"
  raise "boom"
  log = log + "X"        # unreachable
rescue => e
  log = log + "b"
ensure
  log = log + "c"
end
check("basic log order", log, "abc")

# a raise from a nested method is caught here and its value bound with `=> e`.
caught = -1
begin
  risky(5)
  check("risky(5) should have raised", true, false)   # must not run
rescue => e
  caught = e
end
check("rescue binding value", caught, 5)

# read a field off the rescued value.
info = 0
begin
  raise_info()
rescue => e
  info = e["code"]
end
check("rescued value field read", info, 42)

check("no-raise body value", safe_div(20, 4), 5)
check("risky no-raise", risky(2), 4)
check("classify positive (return from begin)", classify(4), 40)
check("classify rescue return", classify(-1), -1)
check("nested return", nested_return(), 9)
check("loop break", loop_break(), 3)
check("loop next", loop_next(), 8)
check("else runs (no raise)", with_else(7), 11)
check("else skipped (raise)", with_else(-1), 2)

if fails == 0
  puts "Ruby begin/rescue OK"
end
exit(fails)
