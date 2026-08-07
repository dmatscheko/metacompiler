# Fast feature-matrix test for the Ruby interpreter (ruby-interpreter.abnf) and the
# LLVM-IR compiler (ruby-to-llvm-ir.abnf). It replaces the four algorithm-themed
# ruby-test-big-* stress tests: instead of large loops (five sorting routines,
# Ackermann, sieves, Roman numerals) every implemented construct is exercised with
# the SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times,
# recursion stays below depth 6. Floats, class inheritance, modules-as-namespaces,
# def self.x, ||=, %w literals, for-in and yield are recognized but not implemented
# (see ruby-test-recognize.rb) and stay out. A failed check prints its id (so a
# diff pinpoints it) and the file ends with exit(fails); exit 0 and byte-identical
# output on all four legs (interpreter/compiler x goja/-frozen) mean everything
# passed.

fails = 0
checks = 0

def check(name, got, want)
  checks = checks + 1
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- Integer at arbitrary precision -----
# Ruby's Integer has no width. Every value here is outside what a double counts
# exactly (2^53 == 9007199254740992), which is the only width at which any of
# these can bite; one bit lower they all pass against a plain double too.
check("big-lit-dec", 9007199254740993.to_s, "9007199254740993")
check("big-lit-hex", 0x20000000000001.to_s, "9007199254740993")
check("big-lit-u64", 0xffffffffffffffff.to_s, "18446744073709551615")
check("big-lit-long", 12345678901234567890.to_s, "12345678901234567890")
check("big-lit-neg", (-9007199254740993).to_s, "-9007199254740993")
check("big-pow2-53", 9007199254740992.to_s, "9007199254740992")   # exact, NOT boxed
check("big-add", (9007199254740992 + 1).to_s, "9007199254740993")
check("big-mul", (9007199254740992 * 9007199254740992).to_s, "81129638414606681695789005144064")
check("big-pow", (2 ** 100).to_s, "1267650600228229401496703205376")
check("big-div-floor", ((-12345678901234567890) / 1000).to_s, "-12345678901234568")
check("big-mod-floor", ((-12345678901234567890) % 97).to_s, "94")
check("big-cmp-exact", 9007199254740993 == 9007199254740992, false)
check("big-cmp-float", 9007199254740993 == 9007199254740993.0, false)
check("big-narrow", 12345678901234567890 - 12345678901234567889, 1)
check("big-shl", (1 << 100).to_s, "1267650600228229401496703205376")
check("big-not", (~9007199254740992).to_s, "-9007199254740993")
check("big-to-s-16", 18446744073709551615.to_s(16), "ffffffffffffffff")
check("big-fmt-d", "%d" % 12345678901234567890, "12345678901234567890")
check("big-no-exponent", (2.0 ** 70).to_i.to_s, "1180591620717411303424")
check("big-class", 12345678901234567890.class.to_s, "Integer")
check("big-hash-key", {18446744073709551616 => "big"}[9223372036854775808 * 2], "big")

# ----- numbers, arithmetic, precedence (integers only; / truncates toward zero) -----
check("arith-precedence", 1 + 2 * 3, 7)
check("arith-paren", (1 + 2) * 3, 9)
check("arith-unary-minus", -5 + 2, -3)
check("arith-int-div", 7 / 2, 3)
check("arith-int-div-neg", -7 / 2, -4)   # Ruby floors: -3.5 -> -4, not toward zero
check("arith-mod", 7 % 3, 1)
check("arith-mod-neg", -7 % 3, 2)       # Ruby % takes the sign of the DIVISOR
check("arith-chain", 20 - 5 - 3, 12)
ca = 10
ca += 5
ca -= 3
ca *= 2
ca /= 4
ca %= 4
check("arith-compound", ca, 2)

# ----- comparison -----
check("cmp-lt", 2 < 3, true)
check("cmp-le", 3 <= 3, true)
check("cmp-gt", 3 > 2, true)
check("cmp-ge", 3 >= 4, false)
check("cmp-eq", 3 == 3, true)
check("cmp-ne", 3 != 4, true)

# ----- strings, interpolation, symbols -----
sname = "world"
check("str-interp", "hello #{sname}!", "hello world!")
check("str-interp-expr", "sum=#{2 + 3}", "sum=5")
check("str-concat", "a" + "b", "ab")
check("str-plus-num", "n=" + 5, "n=5")
check("str-to-s", "n=" + 5.to_s, "n=5")
check("str-length", "hello".length, 5)
check("str-size", "hello".size, 5)
check("str-index", "abc"[1], "b")
check("str-single-quote", 'raw text', "raw text")
check("str-escape-tab", "a\tb".length, 3)
check("str-upcase", "aBc".upcase, "ABC")
check("str-downcase", "aBc".downcase, "abc")
check("str-include", "hello".include?("ell"), true)
check("str-include-miss", "hello".include?("z"), false)
check("str-structural-eq", "ab" == "a" + "b", true)
check("str-compare-lt", "apple" < "banana", true)
check("str-unicode-len", "héllo".length, 5)
check("symbol-not-string", :hello == "hello", false)  # a Symbol is its own type

# ----- booleans, truthiness (only nil and false are falsy), short-circuit -----
check("and-value", 3 && 4, 4)
check("or-value", false || 7, 7)
check("or-nil", nil || 7, 7)
check("and-nil", nil && 5, nil)
check("or-zero-truthy", 0 || 9, 0)
check("not", !false, true)
check("word-and", (1 < 2 and 2 < 3), true)
check("word-or", (false or 5), 5)
check("word-not", (not false), true)

hits = 0
def bump
  hits = hits + 1
  true
end
sc1 = false && bump()
sc2 = true && bump()
sc3 = true || bump()
check("logic-short-circuit", hits, 1)
check("logic-and-false", sc1, false)
check("logic-and-true", sc2, true)
check("logic-or-skip", sc3, true)

check("ternary-true", (3 > 2 ? "yes" : "no"), "yes")
check("ternary-false", (1 > 2 ? "yes" : "no"), "no")
check("ternary-nested", (5 < 0 ? -1 : (5 == 0 ? 0 : 1)), 1)

# ----- if / elsif / else / unless, if as an expression -----
def classify(n)
  if n < 0
    "negative"
  elsif n == 0
    "zero"
  else
    "positive"
  end
end
check("if-neg", classify(-4), "negative")
check("if-zero", classify(0), "zero")
check("if-pos", classify(9), "positive")

g5 = 5
rif = if g5 > 3 then "big" else "small" end
check("if-expression", rif, "big")

ub = 0
unless g5 > 100
  ub = 1
end
check("unless-block", ub, 1)

# ----- while / until, zero and three iterations, post-condition loops -----
w0 = 0
while w0 > 0
  w0 -= 1
end
check("while-zero-iterations", w0, 0)

w3 = 0
while w3 < 3
  w3 += 1
end
check("while-three", w3, 3)

u0 = 5
until u0 >= 5
  u0 += 1
end
check("until-zero-iterations", u0, 5)

u3 = 0
until u3 >= 3
  u3 += 1
end
check("until-three", u3, 3)

runs = 0
begin
  runs += 1
end while runs < 3
check("do-while-repeats", runs, 3)

once = 0
begin
  once += 1
end while once > 9
check("do-while-once", once, 1)

duntil = 0
begin
  duntil += 1
end until duntil >= 3
check("do-until", duntil, 3)

# ----- break / next, nested loops, statement modifiers -----
bstr = ""
bi = 0
while bi < 9
  if bi == 2
    break
  end
  bstr = bstr + bi
  bi += 1
end
check("while-break", bstr, "01")

nstr = ""
ni = 0
while ni < 4
  ni += 1
  if ni % 2 == 0
    next
  end
  nstr = nstr + ni
end
check("while-next", nstr, "13")

nres = ""
oi = 0
while oi < 2
  ii = 0
  while ii < 3
    if ii == 1
      break
    end
    nres = nres + oi + ii
    ii += 1
  end
  oi += 1
end
check("nested-inner-break", nres, "0010")

m1 = 0
m1 = 5 if true
check("if-modifier-taken", m1, 5)
m2 = 0
m2 = 5 if false
check("if-modifier-skipped", m2, 0)
m3 = 0
m3 = 7 unless false
check("unless-modifier", m3, 7)
wcount = 0
wcount += 1 while wcount < 3
check("while-modifier", wcount, 3)
wzero = 0
wzero += 1 while wzero > 9
check("while-modifier-zero", wzero, 0)
dcount = 0
dcount += 1 until dcount >= 3
check("until-modifier", dcount, 3)

# ----- case / when: scalars, several values, ranges, expression, no subject -----
def bucket(n)
  case n
  when 0 then "zero"
  when 1, 2, 3 then "small"
  when 4..6 then "mid"
  else "big"
  end
end
check("case-scalar", bucket(0), "zero")
check("case-multi-value", bucket(2), "small")
check("case-range", bucket(5), "mid")
check("case-range-inclusive-end", bucket(6), "mid")
check("case-else", bucket(9), "big")

def band(v)
  case v
  when 0...4 then "lo"
  else "hi"
  end
end
check("case-range-exclusive-in", band(3), "lo")
check("case-range-exclusive-out", band(4), "hi")

cg = "B"
clabel = case cg
         when "A" then "first"
         when "B" then "second"
         else "other"
         end
check("case-expression", clabel, "second")

c9 = 7
ckind = case
        when c9 < 0 then "neg"
        when c9 == 0 then "zero"
        else "pos"
        end
check("case-no-subject", ckind, "pos")

# ----- methods: implicit/explicit return, recursion, first-class functions -----
def add(ax, bx)
  ax + bx
end
check("fn-implicit-return", add(20, 22), 42)

def sign(n)
  if n < 0
    return -1
  end
  1
end
check("fn-early-return", sign(-8), -1)
check("fn-fallthrough", sign(3), 1)

def fib(n)
  if n < 2
    return n
  end
  fib(n - 1) + fib(n - 2)
end
check("fn-recursion", fib(6), 8)

def even_p(n)
  if n == 0
    return true
  end
  odd_p(n - 1)
end
def odd_p(n)
  if n == 0
    return false
  end
  even_p(n - 1)
end
check("fn-mutual-recursion", even_p(4) && odd_p(5), true)

def pick(digs, cx)
  di = 0
  while di < digs.size
    return di if digs[di] == cx
    di += 1
  end
  return -1
end
check("return-if-modifier", pick(["a", "b", "c"], "b"), 1)
check("return-if-modifier-miss", pick(["a"], "z"), -1)

def double(nx)
  nx * 2
end
def negate(nx)
  0 - nx
end
def apply_fn(fx, nx)
  fx(nx)
end
check("fn-first-class", apply_fn(double, 21), 42)
funcs = [double, negate]
fsel = funcs[1]
check("fn-in-array", fsel(3), -3)

# ----- ranges and blocks (do..end and { }, real closures) -----
rsum = 0
(1..3).each do |k|
  rsum += k
end
check("range-each-inclusive", rsum, 6)

xsum = 0
(1...3).each { |k| xsum += k }
check("range-each-exclusive", xsum, 3)

esum = 0
[10, 20, 30].each do |v|
  esum += v
end
check("block-each-do", esum, 60)

bacc = 100
[1, 2, 3].each { |v| bacc += v }
check("block-brace-closure", bacc, 106)

mapped = [1, 2, 3].map { |v| v * 3 }
check("block-map", mapped[1], 6)
check("block-select", [1, 2, 3, 4, 5].select { |v| v % 2 == 1 }.size, 3)
check("block-reject-sum", [1, 2, 3, 4].reject { |v| v % 2 == 0 }.sum, 4)
check("block-chain", [1, 2, 3, 4].map { |v| v * 2 }.select { |v| v > 4 }.sum, 14)

ewi = 0
[10, 20, 30].each_with_index { |v, ix| ewi += v * ix }
check("block-each-with-index", ewi, 80)

check("truthy-zero-in-select", [0, 1, 2].select { |v| v }.size, 3)
check("falsy-nil-false-dropped", [0, nil, 5, false, 7].select { |v| v }.sum, 12)

bnest = 0
[1, 2].each do |av|
  [10, 20].each { |bv| bnest += av * bv }
end
check("block-nested", bnest, 90)

# ----- arrays -----
arr = [3, 1, 4]
check("arr-index", arr[2], 4)
check("arr-size", arr.size, 3)
check("arr-length", arr.length, 3)
arr[0] = 9
check("arr-set", arr[0], 9)
arr.push(2)
check("arr-push", arr.size, 4)
check("arr-pop", arr.pop, 2)
check("arr-include", arr.include?(4), true)
check("arr-include-miss", arr.include?(7), false)
check("arr-neg-index", [10, 20, 30][-1], 30)

buf = []
buf << 1
buf << 2 << 3
check("arr-append-chain", buf.size, 3)
check("arr-append-last", buf[2], 3)
sel = []
sel << 5 if true
sel << 6 if false
check("arr-append-if-modifier", sel.size, 1)
check("arr-first-last", [10, 20, 30].first + [10, 20, 30].last, 40)
check("arr-empty-first", [].first, nil)
check("arr-empty-pop", [].pop, nil)
check("arr-to-a", [1, 2, 3].to_a.size, 3)
check("arr-nested", [[1, 2], [3]][0][1], 2)

# ----- hashes (string keys, symbol keys, iteration order) -----
h = {"a" => 1, "b" => 2}
check("hash-get", h["a"], 1)
h["c"] = 3
check("hash-set", h["c"], 3)
check("hash-size", h.size, 3)
check("hash-length", h.length, 3)
check("hash-keys-size", h.keys.size, 3)
check("hash-values-sum", h.values.sum, 6)
check("hash-include", h.include?("a"), true)
check("hash-has-key", h.has_key?("b"), true)
check("hash-key-missing", h.key?("z"), false)

horder = ""
h.each { |hk, hv| horder = horder + hk }
check("hash-each-order", horder, "abc")
hvals = 0
h.each { |hk, hv| hvals += hv }
check("hash-each-values", hvals, 6)

opts = { color: "red", size: 4 }
check("hash-symbol-key", opts[:color], "red")
check("hash-symbol-not-string", opts["size"], nil)    # :size and "size" are different keys

# ----- multiple assignment -----
ma, mb = 1, 2
check("multi-a", ma, 1)
check("multi-b", mb, 2)
ma, mb = mb, ma
check("multi-swap-a", ma, 2)
check("multi-swap-b", mb, 1)
mp, mq, mr = 10, 20, 30
check("multi-triple", mp + mq + mr, 60)
mu1, mu2 = [100, 200]
check("multi-unpack-a", mu1, 100)
check("multi-unpack-b", mu2, 200)

# ----- classes: @ivars, methods, self, C.new, attr_*, blocks as parameters -----
class Counter
  def initialize(start, step)
    @value = start
    @step = step
  end
  def increment
    @value += @step
    @value
  end
  def value
    @value
  end
  def reset
    @value = 0
    self
  end
end

c1 = Counter.new(10, 5)
c2 = Counter.new(0, 1)
check("class-init", c1.value(), 10)
check("class-method", c1.increment(), 15)
c2.increment()
c2.increment()
check("class-instances-independent", c1.value() + c2.value(), 17)
rself = c1.reset()
check("class-self-return", rself.value(), 0)

class Box
  attr_accessor :width
  attr_reader :label
  attr_writer :secret
  def initialize(wv)
    @width = wv
    @label = "box"
    @secret = 0
  end
  def area
    @width * @width
  end
  def reveal
    @secret
  end
end
box = Box.new(3)
check("attr-read", box.width, 3)
box.width = 10
check("attr-write", box.width, 10)
check("attr-affects-method", box.area(), 100)
check("attr-reader", box.label, "box")
box.secret = 42
check("attr-writer", box.reveal(), 42)

class Coll
  def initialize(items)
    @items = items
  end
  def my_map(fx)
    mout = []
    @items.each { |x| mout << fx(x) }
    mout
  end
  def my_reduce(init, fx)
    macc = init
    @items.each { |x| macc = fx(macc, x) }
    macc
  end
end
coll = Coll.new([1, 2, 3])
doubled = coll.my_map { |x| x * 2 }
check("class-block-param", doubled[2], 6)
reduced = coll.my_reduce(10) { |acc, x| acc + x }
check("class-block-two-params", reduced, 16)

# ----- safe navigation &. -----
gone = nil
here = "yellow"
check("safe-nav-present", "hello"&.upcase, "HELLO")
check("safe-nav-nil", gone&.upcase, nil)
check("safe-nav-chain-nil", gone&.upcase&.length, nil)
check("safe-nav-chain-present", here&.upcase&.length, 6)

pokes = 0
def poke
  pokes = pokes + 1
  "ell"
end
sn1 = gone&.include?(poke())
check("safe-nav-args-skipped", pokes, 0)
check("safe-nav-nil-result", sn1, nil)
sn2 = here&.include?(poke())
check("safe-nav-args-evaluated", pokes, 1)
check("safe-nav-present-result", sn2, true)

# ----- exceptions: begin / rescue / else / ensure, raise, control flow -----
def risky(n)
  if n > 3
    raise n
  end
  n * 2
end

def raise_info
  raise({"code" => 42})
end

exlog = ""
begin
  exlog = exlog + "a"
  raise "boom"
  exlog = exlog + "X"
rescue => e
  exlog = exlog + "b" + e
ensure
  exlog = exlog + "c"
end
check("begin-rescue-ensure-order", exlog, "abboomc")

caught = -1
begin
  risky(5)
  check("unreachable-after-raise", true, false)
rescue => e
  caught = e
end
check("raise-unwinds-calls", caught, 5)
check("raise-untaken-path", risky(2), 4)

info = 0
begin
  raise_info()
rescue => e
  info = e["code"]
end
check("raise-hash-value", info, 42)

def safe_div(av, bv)
  begin
    av / bv
  rescue => e
    -1
  end
end
check("begin-value-flows", safe_div(20, 4), 5)

def with_else(n)
  tagv = 0
  begin
    if n < 0
      raise n
    end
    tagv = 1
  rescue => e
    tagv = 2
  else
    tagv = tagv + 10
  end
  tagv
end
check("rescue-else-runs", with_else(7), 11)
check("rescue-else-skipped", with_else(-1), 2)

def ret_from_begin(n)
  begin
    if n > 0
      return n * 10
    end
    raise 0
  rescue => e
    return -1
  ensure
    # ensure always runs; no control flow on this path
  end
end
check("return-from-begin", ret_from_begin(4), 40)
check("return-from-rescue", ret_from_begin(0), -1)

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
check("nested-return", nested_return(), 9)

marks = 0
def note_mark
  marks = marks + 1
end
def ret_across_ensure
  begin
    return "from-begin"
  ensure
    note_mark()
  end
end
check("return-across-ensure", ret_across_ensure(), "from-begin")
check("ensure-ran-on-return", marks, 1)

def ret_in_ensure_overrides
  begin
    return "from-begin"
  ensure
    return "from-ensure"
  end
end
check("return-in-ensure-overrides", ret_in_ensure_overrides(), "from-ensure")

def ensure_return_cancels_raise
  begin
    begin
      raise "boom"
    ensure
      return "cancelled"
    end
  rescue
    return "rescued"
  end
end
check("ensure-return-cancels-raise", ensure_return_cancels_raise(), "cancelled")

mi = 0
while true
  mi = mi + 1
  break if mi >= 3
end
check("break-if-modifier-in-while", mi, 3)

mj = 0
mk = 0
while mj < 5
  mj = mj + 1
  next if mj == 2
  mk = mk + 1
end
check("next-if-modifier-in-while", mk, 4)

def ret_if_in_while(n)
  while true
    n = n + 1
    return "hit" if n >= 3
  end
  "unreached"
end
check("return-if-modifier-in-while", ret_if_in_while(0), "hit")

def loop_break_in_begin
  btotal = 0
  bx = 0
  while bx < 9
    begin
      break if bx == 2
      btotal = btotal + bx
    ensure
    end
    bx += 1
  end
  btotal
end
check("break-leaves-begin", loop_break_in_begin(), 1)

def loop_next_in_begin
  ntotal = 0
  nx = -1
  while nx < 3
    nx += 1
    begin
      next if nx == 1
      ntotal = ntotal + nx
    rescue => e
    end
  end
  ntotal
end
check("next-leaves-begin", loop_next_in_begin(), 5)

# a return out of an ensure replaces the begin body's return value
def ret_in_ensure
  begin
    return 1
  ensure
    return 2
  end
end
check("return-in-ensure-overrides", ret_in_ensure(), 2)

def break_in_ensure
  ix = 0
  while true
    begin
      ix = ix + 1
    ensure
      break
    end
  end
  ix
end
check("break-in-ensure", break_in_ensure(), 1)

def rethrow
  begin
    begin
      raise "deep"
    rescue => e
      raise e + "er"
    end
  rescue => e2
    return e2
  end
end
check("rethrow", rethrow(), "deeper")

# ----- yield across a begin/rescue, and block_given? without a method -----
# `yield' inside a begin/rescue used to ABORT the compiler half ("call of a non
# function value: null") while the interpreter answered: js_rblock read the block
# out of the argument array of the IR function being emitted, and inside a
# begin/rescue that is the ctl closure, not the method. The block is published into
# the method's scope now and every yield site reads it through the scope chain, so a
# yield nested in a rescue / ensure / block still finds the method's block.
def yr(l)
  begin
    return l + (yield).to_s
  rescue Exception => e
    return "E"
  end
end
check("yield-in-begin", yr("a") { 1 }, "a1")
def yr2
  begin
    raise "boom"
  rescue => e
    return yield(e.message)
  end
end
check("yield-in-rescue", yr2 { |m| m.upcase }, "BOOM")
def yr3
  begin
    return yield(2)
  ensure
    $ens = "f"
  end
end
check("yield-in-ensure-body", yr3 { |v| v * 3 }, 6)
def yr4
  out = []
  [1, 2].each { |v| out << (yield v) }
  out
end
check("yield-inside-block", yr4 { |v| v * 10 }.inspect, "[10, 20]")
def bg1
  r = block_given?
  begin
    return [r, block_given?]
  rescue
    return nil
  end
end
check("blockgiven-no-block", bg1.inspect, "[false, false]")
check("blockgiven-with-block", (bg1 { 1 }).inspect, "[true, true]")
# block_given? at the TOP LEVEL is false in MRI; it used to fail the interpreter's
# scope lookup outright ("unknown name: $blk").
check("blockgiven-toplevel", block_given?, false)

# ----- Hash defaults -----
# Hash.new(v) and Hash.new { |h, k| } keep the default in a hidden slot honoured at
# EVERY read - including the op-assign and the append reads, which bypassed Hash#[].
hd = Hash.new(7)
check("hash-default-value", hd["nope"], 7)
check("hash-default-no-insert", hd.size, 0)
check("hash-default-keys", hd.keys.inspect, "[]")
check("hash-default-inspect", hd.inspect, "{}")
hd["a"] = 1
check("hash-default-hit", hd["a"], 1)
check("hash-default-miss", hd["b"], 7)
check("hash-default-reader", hd.default, 7)
check("hash-plain-default", Hash.new.default.inspect, "nil")
check("hash-plain-miss", ({})["x"].inspect, "nil")
hc = Hash.new(0)
["a", "b", "a"].each { |w| hc[w] += 1 }
check("hash-default-opassign", hc.inspect, "{\"a\"=>2, \"b\"=>1}")
hb = Hash.new { |h, k| h[k] = k * 2 }
check("hash-default-block", hb["ab"], "abab")
check("hash-default-block-wrote", hb.inspect, "{\"ab\"=>\"abab\"}")
check("hash-default-block-reader", hb.default.inspect, "nil")
hn = Hash.new { |h, k| h[k] = [] }
hn["x"] << 1
hn["x"] << 2
check("hash-default-append", hn.inspect, "{\"x\"=>[1, 2]}")
# fetch DELIBERATELY ignores the default, which is MRI.
check("hash-fetch-hit", hd.fetch("a"), 1)
check("hash-fetch-default-arg", hd.fetch("zz", 5), 5)
check("hash-fetch-block", hd.fetch("zz") { |k| k + "!" }, "zz!")

# ----- Enumerator -----
# A blockless each / each_slice / each_cons / each_with_index / combination /
# permutation is an Enumerator, not the array of elements it would produce. This one
# is EAGER (see the note at mkEnum): there is no coroutine in the compiler half to
# park a lazy one in.
ea = [1, 2, 3, 4, 5]
en = ea.each
check("enum-class", en.class.to_s, "Enumerator")
check("enum-is-a", en.is_a?(Enumerator), true)
check("enum-to-a", en.to_a.inspect, "[1, 2, 3, 4, 5]")
check("enum-size", en.size, 5)
check("enum-next-1", en.next, 1)
check("enum-next-2", en.next, 2)
check("enum-peek", en.peek, 3)
check("enum-next-after-peek", en.next, 3)
en.rewind
check("enum-rewind", en.next, 1)
check("enum-slice-class", ea.each_slice(2).class.to_s, "Enumerator")
check("enum-slice-to-a", ea.each_slice(2).to_a.inspect, "[[1, 2], [3, 4], [5]]")
check("enum-cons-to-a", ea.each_cons(2).to_a.length, 4)
check("enum-with-index", ea.each_with_index.to_a.first.inspect, "[1, 0]")
check("enum-delegates-map", ea.each.map { |v| v * 2 }.inspect, "[2, 4, 6, 8, 10]")
check("enum-with-index-offset", ea.each.with_index(1).to_a.first.inspect, "[1, 1]")
eacc = []
ea.each.with_index(1) { |v, i| eacc << (v * i) }
check("enum-with-index-block", eacc.inspect, "[1, 4, 9, 16, 25]")
check("enum-block-still-yields", ea.each_slice(2) { |g| }.inspect, "[1, 2, 3, 4, 5]")

# ----- sample / shuffle / cycle / combination / permutation -----
# sample and shuffle are asserted on INVARIANTS ONLY - the length, membership, and
# that a shuffle is a permutation of its receiver. There is no random source in this
# project (Kernel#rand is deterministically zero of the right class), so the draw is
# deterministic and its particular ordering is not a contract.
sa = [1, 2, 3, 4, 5]
check("sample-member", sa.include?(sa.sample), true)
check("sample-n-length", sa.sample(3).length, 3)
check("sample-n-clamped", sa.sample(99).length, 5)
check("sample-empty", [].sample.inspect, "nil")
check("sample-n-subset", (sa.sample(3) - sa).inspect, "[]")
sf = sa.shuffle
check("shuffle-length", sf.length, 5)
check("shuffle-permutation", sf.sort.inspect, sa.inspect)
check("shuffle-no-strangers", (sf - sa).inspect, "[]")
check("shuffle-loses-nothing", (sa - sf).inspect, "[]")
check("shuffle-not-in-place", sa.inspect, "[1, 2, 3, 4, 5]")
cyc = []
sa.cycle(2) { |v| cyc << v }
check("cycle-count", cyc.length, 10)
check("cycle-content", cyc.sort.inspect, "[1, 1, 2, 2, 3, 3, 4, 4, 5, 5]")
check("cycle-enum", sa.cycle(2).to_a.length, 10)
check("comb-count", sa.combination(2).to_a.length, 10)
check("comb-class", sa.combination(2).class.to_s, "Enumerator")
check("comb-order", ["a", "b", "c"].combination(2).to_a.inspect,
      "[[\"a\", \"b\"], [\"a\", \"c\"], [\"b\", \"c\"]]")
check("comb-zero", sa.combination(0).to_a.inspect, "[[]]")
check("comb-too-big", sa.combination(9).to_a.inspect, "[]")
check("perm-count", sa.permutation(2).to_a.length, 20)
check("perm-order", ["a", "b", "c"].permutation(2).to_a.inspect,
      "[[\"a\", \"b\"], [\"a\", \"c\"], [\"b\", \"a\"], [\"b\", \"c\"], [\"c\", \"a\"], [\"c\", \"b\"]]")
check("perm-full", ["a", "b", "c"].permutation.to_a.length, 6)
cb = []
["a", "b", "c"].combination(2) { |c| cb << c.join }
check("comb-block", cb.inspect, "[\"ab\", \"ac\", \"bc\"]")

# ----- a BARE rand -----
# `rand` without parentheses used to resolve to the host function value in the
# interpreter (so rand.class said Proc) and to abort the compiler with "variable not
# defined: rand", while `rand()` worked everywhere.
check("bare-rand-class", rand.class.to_s, "Float")
check("bare-rand-range", rand >= 0 && rand < 1, true)
check("paren-rand-class", rand().class.to_s, "Float")
check("rand-n-class", rand(10).class.to_s, "Integer")
check("rand-n-range", rand(10) < 10, true)

# ----- everything combined in one small pipeline (3-element data flow) -----
def transform(items)
  tout = []
  items.each do |n|
    begin
      if n < 0
        raise "neg"
      end
      tout << (n % 2 == 0 ? "e#{n}" : "o#{n}")
    rescue => e
      tout << "x"
    end
  end
  tout
end
tres = transform([1, 2, -3])
check("combined-pipeline", tres[0] + tres[1] + tres[2], "o1e2x")

# ----- parameter binding, String#inspect and Proc#arity -----
# Ruby does not bind positional parameters left to right at fixed indices:
# everything after a *splat is counted from the END of the argument list, an
# OPTIONAL one only takes an argument when there are SPARE ones beyond what the
# required parameters need, and a trailing keyword Hash is not positional at all.
# The compiler half read each parameter from the front by index and answered
# "1|2|[3, 1]|3" for the first row - a live halves divergence --cross never reached
# because no program here had a parameter after a splat (docs/todo.md 1.5).
def pb_post(a, b = 2, *r, c); "#{a}|#{b}|#{r.inspect}|#{c}"; end
def pb_tail(a, *r, c, d); "#{a}|#{r.inspect}|#{c}|#{d}"; end
def pb_kwsplat(*a, **k); "#{a.inspect}|#{k.size}"; end
def pb_optz(x = 1, y = 2, z); "#{x}|#{y}|#{z}"; end
def pb_blk(&b); "#{b.nil?}|#{b.class}|#{b.arity}"; end
check("param-postsplat", pb_post(1, 2, 3, 1), "1|2|[3]|1")
check("param-postsplat-short", pb_post(1, 2), "1|2|[]|2")
check("param-two-after-splat", pb_tail(1, 2, 3, 4), "1|[2]|3|4")
check("param-splat-kwrest", pb_kwsplat(1, 2, z: 3), "[1, 2]|1")
check("param-opt-spare", pb_optz(1) + " " + pb_optz(1, 2), "1|2|1 1|2|2")
check("param-block-object", pb_blk { |q, r| q }, "false|Proc|2")
# String#inspect is the quoted SOURCE form, so it ESCAPES - all three engines
# printed the characters raw, and no gate here can see that because they agreed.
# The four control spellings Ruby has and JavaScript does not (\a \b \f \v \e),
# \s, octal and the braced \u{...} were wrong in the LEXER for the same reason
# (docs/todo.md 1.6).
check("inspect-escapes", ["a\nb", 1].inspect, '["a\nb", 1]')
check("inspect-quote-backslash", "g\"h\\i".inspect, '"g\"h\\\\i"')
check("inspect-control", "\0".inspect + "\x7f".inspect, '"\\u0000""\\u007F"')
check("inspect-interp-hash", "p#{1}#q".inspect + "r\#{s".inspect, '"p1#q""r\#{s"')
check("escape-named", "\a".ord + "\b".ord + "\f".ord + "\v".ord + "\e".ord, 65)
check("escape-space-octal", "\s" + "\101\102" + "\1010" + "\u{41}", " ABA0A")
# Proc#arity needs the closure's parameter shape, which no function value carries:
# the interpreter answered 0 for everything and the compiler half aborted.
check("arity-proc", "#{proc { }.arity}|#{proc { |x| }.arity}|#{proc { |*x| }.arity}|#{proc { |x, y = 1| }.arity}", "0|1|-1|1")
check("arity-lambda", "#{lambda { |x| }.arity}|#{lambda { |x = 1| }.arity}|#{->(a, b = 10) { }.arity}", "1|-1|-2")

puts "features: #{checks} checks, #{fails} failures"
exit(fails)
