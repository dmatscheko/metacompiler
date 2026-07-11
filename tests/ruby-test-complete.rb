# Ruby subset self test 5: the constructs added to complete the language.
# Exercises exactly the newly implemented features:
#   * safe navigation  obj&.m / obj&.m(args) / obj&.field  (nil-safe, args skipped
#     when the receiver is nil),
#   * trailing statement modifiers  STMT if / unless / while / until COND,
#   * subject 'case' with range 'when' clauses (Range#=== membership) mixed with
#     scalar values,
#   * post-condition loops  begin BODY end while / until COND  (body runs at least
#     once).
# Genuinely implemented, so this passes on a default run and is byte-identical
# under goja and -frozen for both the interpreter and the LLVM-IR compiler.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- safe navigation &. -----

# on a present receiver: like a normal method / field access
check("safe method present", "hello"&.upcase, "HELLO")
check("safe field present", [10, 20, 30]&.size, 3)
check("safe method args present", "hello"&.include?("ell"), true)

# on nil: the whole call yields nil instead of raising
gone = nil
check("safe method nil", gone&.upcase, nil)
check("safe field nil", gone&.size, nil)
check("safe chain nil", gone&.upcase&.length, nil)

# a present chain still runs end to end
here = "world"
check("safe chain present", here&.upcase&.length, 5)

# the arguments are NOT evaluated when the receiver is nil (short circuit)
side = [0]
def note(box)
  box[0] = box[0] + 1
  "ell"
end
r1 = gone&.include?(note(side))
check("safe nav short-circuits args", side[0], 0)
check("safe nav nil arg result", r1, nil)
r2 = here&.include?(note(side))
check("safe nav evaluates args when present", side[0], 1)
check("safe nav present arg result", r2, false)

# ----- trailing statement modifiers -----

# if / unless modifier
a = 0
a = 5 if true
check("if modifier taken", a, 5)
b = 0
b = 5 if false
check("if modifier skipped", b, 0)
c = 0
c = 7 unless false
check("unless modifier taken", c, 7)
d = 0
d = 7 unless true
check("unless modifier skipped", d, 0)

# a modifier binds to the whole statement value, and only same-line keywords count:
# the block 'if' below opens its own line and stays a separate statement.
e = 1
if a == 5
  e = 2
end
check("block if stays separate", e, 2)

# while / until modifier (pre-condition: guard is checked before each run)
count = 0
count += 1 while count < 5
check("while modifier", count, 5)
noop = 0
noop += 1 while noop > 100
check("while modifier zero times", noop, 0)
down = 0
down += 1 until down >= 3
check("until modifier", down, 3)

# ----- case with range 'when' clauses -----

def grade(score)
  case score
  when 0..59 then "F"
  when 60..69 then "D"
  when 70..79 then "C"
  when 80..89 then "B"
  else "A"
  end
end
check("case range low", grade(45), "F")
check("case range mid", grade(75), "C")
check("case range high", grade(95), "A")
check("case range inclusive end", grade(89), "B")
check("case range next bucket", grade(90), "A")

# ranges and scalars mixed inside one 'case', several values per 'when'
def kind(n)
  case n
  when 1, 2, 3 then "low"
  when 10..20 then "mid"
  when 100 then "hundred"
  else "other"
  end
end
check("case mixed scalar", kind(2), "low")
check("case mixed range", kind(15), "mid")
check("case mixed lone scalar", kind(100), "hundred")
check("case mixed else", kind(7), "other")

# an exclusive range excludes its upper bound (Range#=== membership)
def band(v)
  case v
  when 0...10 then "single"
  else "big"
  end
end
check("case exclusive in", band(9), "single")
check("case exclusive out", band(10), "big")

# ----- post-condition loops: begin BODY end while / until COND -----

# body always runs once, then repeats while the guard holds
runs = 0
begin
  runs += 1
end while runs < 3
check("do-while repeats", runs, 3)

# even when the guard is false at entry, the body runs exactly once
once = 0
begin
  once += 1
end while once > 100
check("do-while runs at least once", once, 1)

# begin/end until: repeat until the guard becomes true
downs = 0
begin
  downs += 1
end until downs >= 4
check("do-until repeats", downs, 4)

# a do-while accumulating into an array (body value is observable, loop yields nil)
acc = []
k = 0
begin
  k += 1
  acc << k * k
end while k < 4
check("do-while body effects", acc[3], 16)
check("do-while count", acc.size, 4)

# ----- done -----
if fails == 0
  puts "Ruby completion self test passed"
end
exit(fails)
