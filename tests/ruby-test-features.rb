# Fast feature-matrix test for the Ruby interpreter (ruby-interpreter.abnf) and the
# LLVM-IR compiler (ruby-to-llvm-ir.abnf). Every implemented construct is exercised
# with the SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times,
# recursion stays below depth 6. A failed check prints its id (so a diff pinpoints
# it) and the file ends with exit(FEAT[1]); exit 0 and byte-identical output on all
# four legs (interpreter/compiler x goja/-frozen) mean everything passed.

# FEAT[0] counts checks, FEAT[1] failures: a constant, because a def cannot see
# the top-level locals.
FEAT = [0, 0]

def check(name, got, want)
  FEAT[0] = FEAT[0] + 1
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    FEAT[1] = FEAT[1] + 1
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
# String#+ takes a String (or a #to_str object) only: MRI raises a TypeError.
check("str-plus-num", ("n=" + 5 rescue $!.message), "no implicit conversion of Integer into String")
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

# ----- Kernel#format directives %u, %B, %c and %p -----
# A width and a precision count CHARACTERS in MRI, not bytes.
check("fmt-u", "%u" % -5, "-5")
check("fmt-B", "%#B" % 5, "0B101")
check("fmt-c", "%c" % 65, "A")
check("fmt-p", "%p" % "a", "\"a\"")
check("fmt-width-chars", "%6s|" % "café", "  café|")
check("fmt-prec-chars", "%.2s" % "café", "ca")

# ----- the ARGUMENT SELECTORS and %a/%A -----
# WHICH argument a directive takes (MRI: "%1$s-%2$s" % ["a", "b"] is "a-b"), and
# %a/%A, the hex-float form.
check("fmt-numbered", "%2$s-%1$s" % ["a", "b"], "b-a")
check("fmt-named", "%<x>03d/%{y}" % {:x => 7, :y => "z"}, "007/z")
check("fmt-star", "%*.*f|" % [8, 2, 3.14159], "    3.14|")
check("fmt-star-neg", "%*d|" % [-5, 1], "1    |")
check("fmt-hexfloat", "%a %A %.2a" % [3.5, 3.5, 1.0 / 3.0], "0x1.cp+1 0X1.CP+1 0x1.55p-2")
check("fmt-hexfloat-sub", "%a" % 5.0e-324, "0x1p-1074")
check("fmt-hexfloat-pad", "%020a" % 100.0, "0x0000000000001.9p+6")

# ----- a MALFORMED format string raises, and #round/#floor/#ceil take digits ---
# Values and messages off ruby 2.6.10.
def mfmt(f, a)
  begin
    return "OK:" + (f % a)
  rescue ArgumentError => e
    return "AE:" + e.message
  end
end
check("fmt-bad-conv", mfmt("%z", 5), "AE:malformed format string - %z")
check("fmt-bad-width", mfmt("%5", 5), "AE:malformed format string - %*[0-9]")
check("fmt-bad-flags", mfmt("%-", 5), "AE:invalid format character - %")
check("fmt-bad-trail", mfmt("100%", []), "AE:incomplete format specifier; use %% (double %) instead")
check("fmt-bad-paren", mfmt("%<x", 5), "AE:malformed name - unmatched parenthesis")
check("fmt-bad-nl", mfmt("%\n%d", 5), "OK:%\n5")
check("round-digits", [2.345.round(2), 2.675.round(2), 1.15.round(1)].inspect, "[2.35, 2.68, 1.2]")
check("floor-digits", [2.345.floor(2), 2.345.ceil(2), 3.7.floor(1)].inspect, "[2.34, 2.35, 3.7]")
check("round-neg-digits", [1234.567.round(-2), 1234.567.round(-2).class].inspect, "[1200, Integer]")
check("round-int-digits", [15.round(-1), (-25).round(-1), 5.ceil, (-1234).floor(-20)].inspect, "[20, -30, 5, 0]")
check("round-big-digits", 12345678901234567890.round(-5).to_s, "12345678901234600000")

# ----- Float's scoped constants, and the Float -> Integer domain error -----
# #to_i of a non-finite Float raises FloatDomainError; #nan? / #infinite? / #finite?.
check("float-inf-const", Float::INFINITY.to_s, "Infinity")
check("float-nan-const", Float::NAN.nan?, true)
check("float-max-const", Float::MAX.to_s, "1.7976931348623157e+308")
check("float-dig-const", Float::DIG, 15)
check("float-infinite-p", [(1.0 / 0.0).infinite?, (-1.0 / 0.0).infinite?, 1.0.infinite?].inspect, "[1, -1, nil]")
check("float-domain-to-i", (Float::INFINITY.to_i rescue "raised"), "raised")
check("float-domain-msg", (begin; Float::NAN.to_i; rescue FloatDomainError => e; e.message; end), "NaN")
check("float-domain-fmt", (begin; "%d" % [1.0 / 0.0]; rescue FloatDomainError => e; e.message; end), "Infinity")

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

HITS = [0]
def bump
  HITS[0] = HITS[0] + 1
  true
end
sc1 = false && bump()
sc2 = true && bump()
sc3 = true || bump()
check("logic-short-circuit", HITS[0], 1)
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
  bstr = bstr + bi.to_s
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
  nstr = nstr + ni.to_s
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
    nres = nres + oi.to_s + ii.to_s
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
  fx.call(nx)
end
# A method AS A VALUE is `method(:name)'; a bare name is always a call.
check("fn-first-class", apply_fn(method(:double), 21), 42)
funcs = [method(:double), method(:negate)]
fsel = funcs[1]
check("fn-in-array", fsel.call(3), -3)
# A bare name is a CALL whatever the method's parameters are, as long as none of
# them is required: MRI prints 3 for `def q(a = 1); 3; end; q'.
def bare_opt(a = 4); a + 1; end
def bare_splat(*a); a.size; end
def bare_kw(k: 5); k; end
def bare_blk(&b); b.nil? ? 7 : 8; end
check("bare-call-optional", bare_opt, 5)
check("bare-call-splat", bare_splat, 0)
check("bare-call-keyword", bare_kw, 5)
check("bare-call-blockparam", bare_blk, 7)

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
  # The block is named with &fx, which is how Ruby spells it.
  def my_map(&fx)
    mout = []
    @items.each { |x| mout << fx.call(x) }
    mout
  end
  def my_reduce(init, &fx)
    macc = init
    @items.each { |x| macc = fx.call(macc, x) }
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

POKES = [0]
def poke
  POKES[0] = POKES[0] + 1
  "ell"
end
sn1 = gone&.include?(poke())
check("safe-nav-args-skipped", POKES[0], 0)
check("safe-nav-nil-result", sn1, nil)
sn2 = here&.include?(poke())
check("safe-nav-args-evaluated", POKES[0], 1)
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
  exlog = exlog + "b" + e.message
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

MARKS = [0]
def note_mark
  MARKS[0] = MARKS[0] + 1
end
def ret_across_ensure
  begin
    return "from-begin"
  ensure
    note_mark()
  end
end
check("return-across-ensure", ret_across_ensure(), "from-begin")
check("ensure-ran-on-return", MARKS[0], 1)

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
# A yield nested in a rescue / ensure / block still finds the method's block (the
# compiler half reads it through the scope chain, not out of the ctl closure's own
# argument array).
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
# block_given? at the TOP LEVEL is false in MRI.
check("blockgiven-toplevel", block_given?, false)

# ----- Hash defaults -----
# Hash.new(v) and Hash.new { |h, k| } keep the default in a hidden slot honoured at
# EVERY read - including the op-assign and the append reads, which do not go through Hash#[].
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
# `rand` without parentheses is a call of Kernel#rand (rand.class is Float), the
# same as `rand()`.
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
# A Proc handed over as an ORDINARY argument is not the call's block: an optional
# parameter takes the proc, `block_given?' is false, and `&b' is nil.
pb_proc = proc { |v| v }
def pb_opt_proc(f = 3); f.class.to_s; end
def pb_splat_proc(*r); r.size.to_s; end
def pb_bg(f = 3); block_given? ? "blk" : "noblk"; end
def pb_amp(f = 3, &b); "#{f.class}|#{b.nil?}"; end
def pb_yield(x = 5); yield x; end
check("blockmark-optional", pb_opt_proc(pb_proc), "Proc")
check("blockmark-splat", pb_splat_proc(pb_proc), "1")
check("blockmark-given", pb_bg(pb_proc) + "/" + pb_bg { 1 }, "noblk/blk")
check("blockmark-amp", pb_amp(pb_proc), "Proc|true")
check("blockmark-real-block", pb_yield { |v| v + 1 }, 6)
# `&nil' is Ruby for "no block", and it is how a method forwards a block it may
# not have been given.
def fwd_target(a = 1, &b); "#{a.class}|#{b.nil?}"; end
def fwd_caller(*fa, &fb); fwd_target(*fa, &fb); end
check("forward-absent-block", fwd_caller(7), "Integer|true")
check("forward-given-block", fwd_caller(7) { }, "Integer|false")
# Object#send / #__send__ / #public_send. An OPERATOR reached by name is the one
# shape a type's method table cannot answer.
class SendC
  def sm(a = 1); a.class.to_s; end
  def +(o); "plus#{o}"; end
end
sd = [3, "a", [1], SendC.new]
check("send-named", sd[1].send(:upcase), "A")
check("send-user-method", sd[3].send(:sm, 2), "Integer")
check("send-with-block", [1, 2].send(:map) { |x| x + 1 }.inspect, "[2, 3]")
check("send-operator-num", sd[0].send(:+, 4), 7)
check("send-operator-str", sd[1].send(:+, "b"), "ab")
check("send-operator-cmp", sd[0].send(:<=>, 4), -1)
check("send-operator-user", sd[3].send(:+, 5), "plus5")
check("send-aliases", sd[0].public_send(:abs).to_s + sd[0].__send__(:zero?).to_s, "3false")
# String#inspect is the quoted SOURCE form, so it ESCAPES. The four control
# spellings Ruby has and JavaScript does not (\a \b \f \v \e), \s, octal and the
# braced \u{...} are escapes the LEXER has to know.
check("inspect-escapes", ["a\nb", 1].inspect, '["a\nb", 1]')
check("inspect-quote-backslash", "g\"h\\i".inspect, '"g\"h\\\\i"')
check("inspect-control", "\0".inspect + "\x7f".inspect, '"\\u0000""\\u007F"')
check("inspect-interp-hash", "p#{1}#q".inspect + "r\#{s".inspect, '"p1#q""r\#{s"')
check("escape-named", "\a".ord + "\b".ord + "\f".ord + "\v".ord + "\e".ord, 65)
check("escape-space-octal", "\s" + "\101\102" + "\1010" + "\u{41}", " ABA0A")
# Proc#arity needs the closure's parameter shape, which no function value carries.
check("arity-proc", "#{proc { }.arity}|#{proc { |x| }.arity}|#{proc { |*x| }.arity}|#{proc { |x, y = 1| }.arity}", "0|1|-1|1")
check("arity-lambda", "#{lambda { |x| }.arity}|#{lambda { |x = 1| }.arity}|#{->(a, b = 10) { }.arity}", "1|-1|-2")

# ----- Method objects, Proc#to_s and the argument count -----
#
# The behaviour is asserted natively by SECTION 39 of tests/ruby-test-full.rb -
# a matrix row is only BUILT, never run, and half of this lives in layer 2. What
# is here is the EXACT rendering and the exact messages, which the ratchet
# deliberately does not pin because two of them are recorded divergences from
# MRI rather than MRI itself.
def arity_def(a, b = 1); a + b; end
def arity_none; 0; end
# A def closure's parameter shape reaches Proc#arity, gated as a block's is:
# markParamArity / emitArity record nothing unless the program names `arity`.
# A METHOD counts optionals like a LAMBDA and not like a proc, so `def m(a, b = 1)`
# is -2 and not 1 (/usr/bin/ruby 2.6.10; #arity is unchanged in 3.x).
check("arity-def", "#{method(:arity_def).arity}|#{method(:arity_none).arity}", "-2|0")
# HOW A PROC AND A METHOD PRINT. MRI writes `#<Proc:0x0000...@f.rb:12 (lambda)>`
# and, for a Method, `#<Method: Object#arity_none>`. A Proc's address and source
# location are the process's own and cannot be matched, so they are LEFT OFF rather
# than invented - the same rule python's `<function fn>` follows. THE METHOD FORM
# CARRIES NO ADDRESS AT ALL in MRI 2.6 (3.0 appends a source location), so that
# row is byte-exact against the oracle rather than a near miss.
check("proc-to-s", "#{proc { }}|#{->(x) { x }}|#{lambda { |x| x }}", "#<Proc>|#<Proc (lambda)>|#<Proc (lambda)>")
check("method-to-s", "#{method(:arity_none)}|#{method(:arity_none).class}", "#<Method: Object#arity_none>|Method")
# Method#name, #to_proc and the two type tests. A Method is NOT a Proc in MRI and a
# Method#to_proc IS one - a LAMBDA one, carrying the Method's arity.
check("method-name", "#{method(:arity_def).name.inspect}|#{method(:arity_def).is_a?(Method)}|#{method(:arity_def).is_a?(Proc)}", ":arity_def|true|false")
check("method-to-proc", "#{method(:arity_def).to_proc.class}|#{method(:arity_def).to_proc.arity}|#{method(:arity_def).to_proc.lambda?}", "Proc|-2|true")
# method(:name) reaches an INSTANCE METHOD of the running self, bound to it.
class MethodValC
  def inst(a); a + 10; end
  def probe; "#{method(:inst).call(5)}|#{method(:inst).arity}|#{method(:inst).class}"; end
end
check("method-bound-self", MethodValC.new.probe, "15|1|Method")
# The OWNER a bound Method prints is the receiver's class, which is what MRI writes
# for a method defined there. (MRI writes `D(C)#m` for one INHERITED from C; this
# writes `D#m`, which can only shorten the answer.)
check("method-bound-to-s", "#{MethodValC.new.method(:inst)}", "#<Method: MethodValC#inst>")
# A missing name is a CATCHABLE NameError, not a host abort.
def method_miss
  begin
    method(:no_such_method_anywhere)
  rescue NameError => e
    "#{e.class}"
  end
end
check("method-nameerror", method_miss, "NameError")
# TOO FEW POSITIONAL ARGUMENTS is an ArgumentError, in MRI's own wording. Ruby
# checks methods and lambdas, never blocks.
def argc_one(a); a; end
def argc_opt(a, b = 2); a; end
def argc_star(a, *r); a; end
def argc_msg
  begin
    argc_one
  rescue ArgumentError => e
    e.message
  end
end
def argc_msg2
  begin
    argc_opt
  rescue ArgumentError => e
    e.message
  end
end
def argc_msg3
  begin
    argc_star
  rescue ArgumentError => e
    e.message
  end
end
check("argc-required", argc_msg, "wrong number of arguments (given 0, expected 1)")
check("argc-optional", argc_msg2, "wrong number of arguments (given 0, expected 1..2)")
check("argc-splat", argc_msg3, "wrong number of arguments (given 0, expected 1+)")


# ----- assorted MRI behaviours ----------------------------------------------
# Every value here is /usr/bin/ruby 2.6.10's answer, unchanged in 3.x.

# super(m) inside a user Exception subclass' initialize: the superclass is a
# BUILTIN and carries no compiled `initialize'.
class T13Err < StandardError
  def initialize(m)
    super("wrapped: " + m)
  end
end
def t13_super
  begin
    raise T13Err, "boom"
  rescue => e
    e.message
  end
end
check("t13-super-init", t13_super, "wrapped: boom")
check("t13-super-isa", (T13Err.new("x").is_a?(Exception)), true)
check("t13-plain-isa", (RuntimeError.new("x").is_a?(Exception)), true)

# Exception#inspect / #message / #to_s. MRI's #message IS #to_s, and #inspect is
# "#<Class: to_s>" - or the bare class name when to_s is empty.
class T13Sub < StandardError
  def to_s
    "TS"
  end
end
check("t13-inspect-plain", RuntimeError.new("nm").inspect, "#<RuntimeError: nm>")
check("t13-inspect-sub", T13Sub.new("m").inspect, "#<T13Sub: TS>")
check("t13-message-is-to-s", T13Sub.new("m").message, "TS")
check("t13-inspect-noarg", StandardError.new.inspect, "#<StandardError: StandardError>")
class T13Plain
end
check("t13-obj-to-s", T13Plain.new.to_s, "#<T13Plain>")

# The modifier rescue. It binds TIGHTER than `=' (so the FALLBACK is assigned),
# and it applies to a whole statement, so `raise "a" rescue 7' is caught.
def t13_boom
  raise "e"
end
t13_a = t13_boom rescue "c"
check("t13-rescue-rhs", t13_a, "c")
t13_b = (raise "a" rescue 7)
check("t13-rescue-stmt", t13_b, 7)
t13_c = 1 / 0 rescue "div"
check("t13-rescue-zerodiv", t13_c, "div")
def t13_zde
  begin
    1 / 0
  rescue ZeroDivisionError => e
    e.message
  end
end
check("t13-zerodiv-catchable", t13_zde, "divided by 0")

# Kernel#Integer / #Float / #String / #Array: the conversion functions, not the
# classes of the same name.
check("t13-kernel-int", Integer("0x1f"), 31)
check("t13-kernel-int-base", Integer("ff", 16), 255)
check("t13-kernel-int-under", Integer("1_000"), 1000)
t13_d = Integer("z") rescue 5
check("t13-kernel-int-raise", t13_d, 5)
check("t13-kernel-float", Float("1_0.5"), 10.5)
check("t13-kernel-string", String(5), "5")
check("t13-kernel-array", Array(nil), [])

# A bare `puts' must not read the next line's statement as its argument.
t13_e = 5
puts
check("t13-bare-puts", t13_e, 5)

# Two-argument and range index ASSIGNMENT, and String element assignment.
t13_f = [1, 2, 3, 4, 5]
t13_f[1, 2] = 9
check("t13-idx2-assign", t13_f, [1, 9, 4, 5])
t13_g = [1, 2, 3, 4, 5]
t13_g[1, 2] = [7, 8, 9]
check("t13-idx2-splice", t13_g, [1, 7, 8, 9, 4, 5])
t13_h = [1, 2, 3, 4, 5]
t13_h[1..2] = 0
check("t13-range-assign", t13_h, [1, 0, 4, 5])
t13_i = "hello world"
t13_i[0, 5] = "HI"
check("t13-str-slice-assign", t13_i, "HI world")
t13_j = "hello"
t13_j[0] = "H"
check("t13-str-elem-assign", t13_j, "Hello")
t13_k = [1, 2, 3]
t13_k[5, 0] = 9
check("t13-idx2-grow", t13_k, [1, 2, 3, nil, nil, 9])

# A bounded range with a NEGATIVE endpoint, held in a VARIABLE.
t13_r = 1..-1
t13_s = "abcdefgh"
t13_t = [1, 2, 3, 4, 5, 6, 7, 8]
check("t13-negrange-str", t13_s[t13_r], "bcdefgh")
check("t13-negrange-arr", t13_t[t13_r], [2, 3, 4, 5, 6, 7, 8])
t13_r2 = 2..-2
check("t13-negrange-mid", t13_s[t13_r2], "cdefg")
t13_r3 = 1...-1
check("t13-negrange-excl", t13_s[t13_r3], "bcdefg")
check("t13-negrange-size", t13_r.size, 0)
check("t13-negrange-toa", t13_r.to_a, [])
check("t13-negrange-str-of", t13_r.to_s, "1..-1")
t13_r4 = 1..3
check("t13-range-size", t13_r4.size, 3)
check("t13-range-min-empty", t13_r.min, nil)

# The String surface: bytes / tr / center / ljust / rjust / count / delete and
# their families.
check("t13-bytes", "caf\u00e9".bytes, [99, 97, 102, 195, 169])
check("t13-bytesize", "caf\u00e9".bytesize, 5)
check("t13-getbyte", "hello".getbyte(1), 101)
check("t13-codepoints", "hi".codepoints, [104, 105])
check("t13-sum", "hello".sum, 532)
check("t13-tr-range", "abcdef".tr("a-c", "x"), "xxxdef")
check("t13-tr-pair", "hello".tr("el", "ip"), "hippo")
check("t13-tr-neg", "hello".tr("^aeiou", "*"), "*e**o")
check("t13-tr-shift", "hello".tr("a-y", "b-z"), "ifmmp")
check("t13-tr-s", "hello".tr_s("l", "r"), "hero")
check("t13-squeeze", "aaabbbccc".squeeze, "abc")
check("t13-squeeze-set", "aaabbbccc".squeeze("a"), "abbbccc")
check("t13-count", "hello world".count("lo"), 5)
check("t13-count-two", "hello world".count("lo", "o"), 2)
check("t13-count-neg", "hello".count("^l"), 3)
check("t13-delete", "hello world".delete("l"), "heo word")
check("t13-center", "hi".center(10), "    hi    ")
check("t13-center-pad", "hi".center(10, "12"), "1212hi1212")
check("t13-center-narrow", "hi".center(1), "hi")
check("t13-ljust", "hi".ljust(6, "."), "hi....")
check("t13-rjust", "hi".rjust(6, "."), "....hi")
check("t13-center-chars", "caf\u00e9".center(8, "-"), "--caf\u00e9--")
check("t13-chop", "hello".chop, "hell")
check("t13-chomp", "hello\n".chomp, "hello")
check("t13-chomp-crlf", "abc\r\n".chomp, "abc")
check("t13-chr", "hello".chr, "h")
check("t13-hex", "0x1f".hex, 31)
check("t13-oct", "17".oct, 15)
check("t13-swapcase", "HeLLo".swapcase, "hEllO")
check("t13-insert", "hello".insert(1, "X"), "hXello")
check("t13-insert-neg", "hello".insert(-1, "X"), "helloX")
check("t13-prepend", "hello".prepend("X"), "Xhello")
check("t13-partition", "hello".partition("l"), ["he", "l", "lo"])
check("t13-rpartition", "hello".rpartition("l"), ["hel", "l", "o"])
check("t13-partition-miss", "hello".partition("z"), ["hello", "", ""])
check("t13-delete-prefix", "hello".delete_prefix("he"), "llo")
check("t13-delete-suffix", "hello".delete_suffix("lo"), "hel")
check("t13-casecmp", "abc".casecmp("ABD"), -1)
check("t13-casecmp-q", "hello".casecmp?("HELLO"), true)
check("t13-lines", "a\nb\n".lines, ["a\n", "b\n"])
check("t13-rindex", "hello".rindex("l"), 3)
check("t13-lstrip", "  hi  ".lstrip, "hi  ")
check("t13-rstrip", "  hi  ".rstrip, "  hi")
check("t13-ascii-only", "caf\u00e9".ascii_only?, false)
check("t13-dump", "hello".dump, "\"hello\"")
# The BANG forms answer the new string, or nil when nothing changed.
check("t13-upcase-bang", "hello".upcase!, "HELLO")
check("t13-upcase-bang-nil", "HELLO".upcase!, nil)
check("t13-strip-bang-nil", "hello".strip!, nil)
check("t13-tr-bang-nil", "hello".tr!("z", "Z"), nil)
t13_bytes = []
"hi".each_byte { |b| t13_bytes << b }
check("t13-each-byte", t13_bytes, [104, 105])
t13_up = []
"ab".upto("ad") { |x| t13_up << x }
check("t13-upto", t13_up, ["ab", "ac", "ad"])

# ----- A String is mutable: a mutator changes the OBJECT every alias holds -----
# A literal is a fresh String at every evaluation, and so is a String a method
# call, an operator or to_s builds - an Array a String method builds holds fresh
# ones; a frozen one raises FrozenError. Ruby-2 forms; every want is
# /usr/bin/ruby 2.6.10's answer.
ms_a = "ab"
ms_b = ms_a
ms_a.upcase!
check("mstr-bang-alias", [ms_a, ms_b], ["AB", "AB"])
ms_a << "c" << 100
check("mstr-append-alias", ms_b, "ABcd")
ms_arr = [ms_a]
ms_a.concat("e", "f")
check("mstr-elem-alias", ms_arr[0], "ABcdef")
ms_h = {k: "x"}
ms_v = ms_h[:k]
ms_v.replace("yz")
check("mstr-hash-alias", ms_h[:k], "yz")
def ms_add(buf)
  buf << "!"
  buf.capitalize!
  nil
end
ms_c = "hi"
ms_add(ms_c)
check("mstr-param-alias", ms_c, "Hi!")
@ms_iv = "iv"
ms_iv2 = @ms_iv
@ms_iv.swapcase!
check("mstr-ivar-alias", ms_iv2, "IV")
ms_s = "abc"
ms_t = ms_s
ms_s[0] = "X"
ms_s.insert(1, "-")
ms_s.prepend("<")
check("mstr-index-set-alias", ms_t, "<X-bc")
check("mstr-slice-bang", [ms_t.slice!(1, 2), ms_s], ["X-", "<bc"])
ms_s.clear
check("mstr-clear-alias", ms_t, "")
ms_src = ["ab", "cd", 3]
ms_d = ms_src[0] + ms_src[1]
ms_d.reverse!
ms_e = ms_d
ms_d.chop!
check("mstr-computed", [ms_d, ms_e], ["dcb", "dcb"])
ms_l = [ms_src[0] + "1", ms_src[1] + "2"]
ms_l[0].upcase!
ms_l[1] << "z"
check("mstr-computed-elem", ms_l, ["AB1", "cd2z"])
ms_w = ms_src[2].to_s
ms_w.concat("0")
check("mstr-computed-int", ms_w, "30")
def ms_lit; "lit"; end
ms_f = ms_lit
ms_f << "!"
check("mstr-literal-fresh", [ms_f, ms_lit], ["lit!", "lit"])
ms_words = %w[a b]
ms_words.each { |w| w.upcase! }
check("mstr-pct-w", ms_words, ["A", "B"])
check("mstr-equal", ["a".equal?("a"), ms_f.equal?(ms_f), ms_f.to_s.equal?(ms_f), ms_f.dup.equal?(ms_f)],
      [false, true, true, false])
check("mstr-sub-nil", ["hello".sub!("z", "y"), "hello".gsub!(/z/, "y"), "aa".gsub!("a", "a")],
      [nil, nil, "aa"])
check("mstr-bang-nil", ["AB".upcase!, "ab".strip!, "x".succ!], [nil, nil, "y"])
ms_z = "lit"
ms_z2 = ms_z
ms_z.freeze
ms_err = begin
  ms_z2 << "x"
  "none"
rescue FrozenError => e
  e.message
end
check("mstr-frozen", [ms_err, ms_z2.frozen?, "abc".frozen?, "a".upcase.frozen?], ["can't modify frozen String", true, false, false])
check("mstr-frozen-copies", [ms_z.dup.frozen?, ms_z.clone.frozen?, (+ms_z).frozen?, (-"q").frozen?], [false, true, false, true])
ms_k = "key"
ms_kh = {ms_k => 1}
ms_k << "!"
check("mstr-hash-key-copy", [ms_kh, ms_kh["key"], ms_k], [{"key" => 1}, 1, "key!"])
cb_src = ["a b", "xy", 7]
cb_w = cb_src[0].split
cb_w.each { |w| w.upcase! }
cb_c = cb_src[1].chars
cb_c.each { |c| c << "!" }
cb_m = cb_w.map(&:downcase)
cb_m.each { |w| w << "1" }
check("mstr-computed-array", [cb_w, cb_c, cb_m], [["A", "B"], ["x!", "y!"], ["a1", "b1"]])
cb_a = cb_src[1] + "z"
cb_b = cb_a
cb_a << "+"
cb_t = cb_src[2].to_s
cb_u = cb_t
cb_t.concat("0")
cb_f = "%s." % cb_src[1]
cb_g = cb_f
cb_f.upcase!
cb_r = cb_src[1].reverse
cb_s = cb_r
cb_r.capitalize!
check("mstr-computed-alias", [cb_b, cb_u, cb_g, cb_s], ["xyz+", "70", "XY.", "Yx"])
check("mstr-computed-identity", [cb_r.equal?(cb_r.to_s), cb_src[1].upcase.equal?(cb_src[1].upcase), cb_r.frozen?],
      [true, false, false])


# ----- mixins as ancestors: is_a?, ancestors, and the module operators -----
# ruby-to-llvm-ir.abnf FLATTENS `include M' at build time - M's method bodies are
# re-emitted into the including class - so the ancestry is recorded separately
# ($mix$/$xmix$ + __mixorder/__preorder) for `obj.is_a?(M)', `case obj when M',
# Module#ancestors / #include? / #< / #<= / #> / #>= and a module included INTO A
# MODULE. Every want here is /usr/bin/ruby 2.6.10's answer, and the constructs
# are all Ruby-2 forms whose behaviour is unchanged in 3.x.
module MixM
  def mix_hello; "hi"; end
end
module MixN
  def mix_bye; "bye"; end
end
module MixP
  include MixN
end
module MixL
  def mix_word; super.upcase; end
end
class MixA
  include MixM
end
class MixB < MixA
end
class MixC
  include MixP
end
class MixD
  extend MixM
end
class MixW
  prepend MixL
  include MixM
  def mix_word; "abc"; end
end
class MixErr < StandardError
end
class MixSz
  include Comparable
  attr_reader :n
  def initialize(n); @n = n; end
  def <=>(o); n <=> o.n; end
end
mix_a = MixA.new
check("mix-isa", mix_a.is_a?(MixM), true)
check("mix-kindof", mix_a.kind_of?(MixM), true)
check("mix-instanceof", mix_a.instance_of?(MixM), false)
check("mix-caseeq", (MixM === mix_a), true)
check("mix-when", (case mix_a when MixM then "m" else "no" end), "m")
check("mix-unrelated", mix_a.is_a?(MixN), false)
# The module is inherited through the SUPERCLASS as well.
check("mix-sub-isa", MixB.new.is_a?(MixM), true)
# A module included into a MODULE: the methods and the ancestry both carry.
check("mix-modmod-method", MixC.new.mix_bye, "bye")
check("mix-modmod-isa-outer", MixC.new.is_a?(MixP), true)
check("mix-modmod-isa-inner", MixC.new.is_a?(MixN), true)
# extend M puts M on the class OBJECT, not on its instances.
check("mix-extend-cls", MixD.is_a?(MixM), true)
check("mix-extend-inst", MixD.new.is_a?(MixM), false)
# prepend: the module is an ancestor IN FRONT of the class.
check("mix-prepend-super", MixW.new.mix_word, "ABC")
check("mix-prepend-isa", MixW.new.is_a?(MixL), true)
check("mix-prepend-order", MixW.ancestors, [MixL, MixW, MixM, Object, Kernel, BasicObject])
# Comparable is NOT universal.
check("mix-comparable-yes", MixSz.new(1).is_a?(Comparable), true)
check("mix-comparable-no", mix_a.is_a?(Comparable), false)
check("mix-comparable-op", (MixSz.new(1) < MixSz.new(2)), true)
# Module#ancestors, including through a BUILTIN superclass.
check("mix-anc-class", MixA.ancestors, [MixA, MixM, Object, Kernel, BasicObject])
check("mix-anc-sub", MixB.ancestors, [MixB, MixA, MixM, Object, Kernel, BasicObject])
check("mix-anc-module", MixM.ancestors, [MixM])
check("mix-anc-modmod", MixP.ancestors, [MixP, MixN])
check("mix-anc-exc", MixErr.ancestors, [MixErr, StandardError, Exception, Object, Kernel, BasicObject])
check("mix-include?", MixA.include?(MixM), true)
check("mix-include?-self", MixA.include?(MixA), false)
check("mix-include?-no", MixA.include?(MixN), false)
# Module#< / #<= / #> / #>=: the ancestry relation, and NIL for two unrelated ones.
check("mix-lt-mod", (MixA < MixM), true)
check("mix-lt-super", (MixB < MixA), true)
check("mix-lt-rev", (MixA < MixB), false)
check("mix-le-same", (MixA <= MixA), true)
check("mix-gt", (MixA > MixM), false)
check("mix-ge-same", (MixA >= MixA), true)
check("mix-lt-unrelated", (MixA < String), nil)
# Class and Module as constants, and #class / #superclass on a class object.
check("mix-class-of-class", MixA.class, Class)
check("mix-class-of-module", MixM.class, Module)
check("mix-instof-class", MixA.instance_of?(Class), true)
check("mix-instof-module", MixM.instance_of?(Module), true)
check("mix-mod-not-class", MixM.is_a?(Class), false)
check("mix-class-is-module", MixA.is_a?(Module), true)
check("mix-superclass-root", MixA.superclass, Object)
check("mix-superclass-sub", MixB.superclass, MixA)
check("mix-superclass-exc", MixErr.superclass, StandardError)
check("mix-superclass-prepend", MixW.superclass, Object)

# ----- a Range is a VALUE, not a materialized Array -----
# Every want below is /usr/bin/ruby 2.6.10's own answer - ranges are a Ruby-2
# construct whose behaviour is unchanged in 3.x - EXCEPT the five beginless-range
# rows at the end, which 2.6 cannot even parse (beginless ranges are 2.7) and
# which are therefore spec-derived and marked.
rng_lo = [1, 3, 'a', 5, 1, 1]
rng_hi = [3, 1, 'e', 5, -1, 1]
rng_r   = (rng_lo[0]..rng_hi[0])
rng_x   = (rng_lo[0]...rng_hi[0])
rng_s   = (rng_lo[2]..rng_hi[2])
rng_rev = (rng_lo[1]..rng_hi[1])
rng_neg = (rng_lo[4]..rng_hi[4])
rng_arr = [10, 20, 30, 40]
# The range AS A VALUE: what it prints, what class it is, what it equals.
check("rng-inspect", rng_r.inspect, "1..3")
check("rng-inspect-excl", rng_x.inspect, "1...3")
check("rng-inspect-str", rng_s.inspect, "\"a\"..\"e\"")
check("rng-inspect-rev", rng_rev.inspect, "3..1")
check("rng-to-s", rng_r.to_s, "1..3")
check("rng-to-s-str", rng_s.to_s, "a..e")
check("rng-interp", "#{rng_r}", "1..3")
check("rng-class", rng_r.class, Range)
check("rng-is-a", rng_r.is_a?(Range), true)
check("rng-not-array", rng_r.is_a?(Array), false)
check("rng-eq", rng_r == (1..3), true)
check("rng-eq-excl", rng_r == rng_x, false)
check("rng-eql", (1..3).eql?((1..3)), true)
check("rng-uniq", [(1..2), (1..2)].uniq.size, 1)
# The endpoints. #end and #last with NO argument are the range's END, not its last
# element; with an argument they are the Array's.
check("rng-begin", rng_r.begin, 1)
check("rng-end", rng_r.end, 3)
check("rng-end-excl", rng_x.end, 3)
check("rng-first", rng_r.first, 1)
check("rng-last", rng_r.last, 3)
check("rng-last-excl", rng_x.last, 3)
check("rng-first-n", rng_r.first(2), [1, 2])
check("rng-last-n", rng_r.last(2), [2, 3])
check("rng-excl-false", rng_r.exclude_end?, false)
check("rng-excl-true", rng_x.exclude_end?, true)
# As a SEQUENCE.
check("rng-to-a", rng_r.to_a, [1, 2, 3])
check("rng-to-a-excl", rng_x.to_a, [1, 2])
check("rng-to-a-str", rng_s.to_a, ["a", "b", "c", "d", "e"])
check("rng-to-a-rev", rng_rev.to_a, [])
check("rng-to-a-neg", rng_neg.to_a, [])
check("rng-size", rng_r.size, 3)
check("rng-size-excl", rng_x.size, 2)
check("rng-count", rng_r.count, 3)
check("rng-min", rng_r.min, 1)
check("rng-max", rng_r.max, 3)
check("rng-max-excl", rng_x.max, 2)
check("rng-min-empty", rng_rev.min, nil)
check("rng-sum", rng_r.sum, 6)
check("rng-map", rng_r.map { |v| v * 2 }, [2, 4, 6])
check("rng-select", rng_r.select { |v| v > 1 }, [2, 3])
check("rng-reduce", rng_r.reduce(0) { |a, b| a + b }, 6)
check("rng-sort-by", rng_r.sort_by { |v| -v }, [3, 2, 1])
check("rng-partition", rng_r.partition { |v| v.odd? }, [[1, 3], [2]])
check("rng-zip", rng_r.zip([9, 8, 7]), [[1, 9], [2, 8], [3, 7]])
check("rng-each-slice", rng_r.each_slice(2).to_a, [[1, 2], [3]])
check("rng-reverse-each", rng_r.reverse_each.to_a, [3, 2, 1])
check("rng-each-with-index", rng_r.each_with_index.to_a, [[1, 0], [2, 1], [3, 2]])
check("rng-take", rng_r.take(2), [1, 2])
check("rng-group-by", rng_r.group_by { |v| v % 2 }, {1 => [1, 3], 0 => [2]})
rng_acc = []
rng_each_ret = rng_r.each { |v| rng_acc << v }
check("rng-each", rng_acc, [1, 2, 3])
check("rng-each-returns-self", rng_each_ret, rng_r)
# Range#step.
check("rng-step", (1..10).step(3).to_a, [1, 4, 7, 10])
check("rng-step-str", ('a'..'e').step(2).to_a, ["a", "c", "e"])
rng_st = []
(1..7).step(3) { |v| rng_st << v }
check("rng-step-block", rng_st, [1, 4, 7])
# for-in, splat and destructuring - all three lower to js_pylen + js_pyget, which
# need a range arm or each would run ZERO times.
rng_for = []
for rng_v in (1..3)
  rng_for << rng_v
end
check("rng-for-in", rng_for, [1, 2, 3])
check("rng-splat-literal", [*rng_r], [1, 2, 3])
rng_p1, rng_p2, rng_p3 = *rng_r
check("rng-splat-destructure", [rng_p1, rng_p2, rng_p3], [1, 2, 3])
def rng_three(a, b, c); [a, b, c]; end
check("rng-splat-args", rng_three(*rng_r), [1, 2, 3])
# Membership, ===, and case/when - both as a literal and through a VARIABLE, which
# is the form that has to reach js_rcase.
check("rng-include", rng_r.include?(2), true)
check("rng-include-no", rng_r.include?(4), false)
check("rng-include-excl", rng_x.include?(3), false)
check("rng-cover", rng_r.cover?(2), true)
check("rng-include-str", rng_s.include?('c'), true)
check("rng-case-eq", (rng_r === 2), true)
check("rng-case-eq-no", (rng_r === 9), false)
check("rng-when-literal", (case 2 when 1..3 then "in" else "out" end), "in")
check("rng-when-var", (case 2 when rng_r then "in" else "out" end), "in")
check("rng-when-var-miss", (case 9 when rng_r then "in" else "out" end), "out")
# A range as a Hash KEY. Two range literals are two distinct objects, so this needs
# the value comparison rbEql / core.keyEq reach.
rng_h = {(1..2) => "a", (3..4) => "b"}
check("rng-hash-key", rng_h[(3..4)], "b")
check("rng-hash-keys", rng_h.keys, [(1..2), (3..4)])
check("rng-hash-size", rng_h.size, 2)
# SLICING.
check("rng-slice-arr", rng_arr[rng_r], [20, 30, 40])
check("rng-slice-lit", rng_arr[1..2], [20, 30])
check("rng-slice-lit-excl", rng_arr[1...3], [20, 30])
check("rng-slice-neg", rng_arr[1..-1], [20, 30, 40])
check("rng-slice-neg-var", rng_arr[rng_neg], [20, 30, 40])
check("rng-slice-str", "hello"[1..3], "ell")
check("rng-slice-str-neg", "abcdef"[1..-2], "bcde")
check("rng-slice-method", [1, 2, 3].slice(1..2), [2, 3])
check("rng-slice-method-excl", [1, 2, 3].slice(1...3), [2, 3])
check("rng-slice-str-method", "hello".slice(1..3), "ell")
rng_w = [10, 20, 30, 40]
rng_w[1..2] = [9]
check("rng-slice-assign", rng_w, [10, 9, 40])
rng_w2 = [1, 2, 3, 4, 5]
rng_w2[1...3] = [0]
check("rng-slice-assign-excl", rng_w2, [1, 0, 4, 5])
# ENDLESS (2.6) ranges: no sequence, so only the lazy questions are answerable.
rng_open = (1..)
check("rng-open-begin", rng_open.begin, 1)
check("rng-open-end", rng_open.end, nil)
check("rng-open-include", rng_open.include?(99), true)
check("rng-open-min", rng_open.min, 1)
check("rng-open-case", (rng_open === 5), true)
check("rng-open-slice-str", "hello world"[6..], "world")
check("rng-open-slice-arr", rng_arr[2..], [30, 40])
# BEGINLESS ranges are Ruby 2.7 syntax, so /usr/bin/ruby 2.6.10 cannot parse them
# and these five wants are SPEC-DERIVED (ruby-doc Range: #begin is nil, the missing
# side never constrains) rather than corpus-derived. All three engines agree.
rng_bl = (..3)
check("rng-beginless-begin", rng_bl.begin, nil)
check("rng-beginless-end", rng_bl.end, 3)
check("rng-beginless-include", rng_bl.include?(1), true)
check("rng-beginless-include-no", rng_bl.include?(9), false)
check("rng-beginless-slice", "hello"[..4], "hello")

# ----- the per-object singleton method table -----
# `o.extend(M)' and a def inside `class << VALUE' go to a per-object table that
# is checked ahead of the __class chain. Every want is /usr/bin/ruby 2.6.10's
# answer; all the forms here are Ruby-2 syntax. The
# ratchet's SECTION 42 carries the ordering rules; these are the rows the ratchet
# does not repeat - a module that includes another, and a CLASS object as the
# receiver, where `extend' has to reach the class-method side.
module SgA
  def sg_am; "A"; end
end
module SgB
  include SgA
  def sg_bm; "B"; end
end
class SgC
  def sg_cm; "C"; end
end
class SgD
  def self.sg_sm; "SM"; end
end
sg_o = SgC.new
sg_o.extend(SgB)
check("sing-obj-mixin", sg_o.sg_am + sg_o.sg_bm + sg_o.sg_cm, "ABC")
check("sing-obj-names", sg_o.singleton_methods.map { |s| s.to_s }.sort, ["sg_am", "sg_bm"])
check("sing-obj-isa", sg_o.is_a?(SgA) && sg_o.is_a?(SgB), true)
check("sing-obj-clean", SgC.new.respond_to?(:sg_am), false)
SgD.extend(SgA)
check("sing-cls-names", SgD.singleton_methods.map { |s| s.to_s }.sort, ["sg_am", "sg_sm"])
check("sing-cls-extend", SgD.sg_am, "A")
check("sing-cls-isa", SgD.is_a?(SgA), true)
check("sing-cls-own", SgD.sg_sm, "SM")
check("sing-cls-respond", SgD.respond_to?(:sg_sm), true)
sg_e = SgC.new
def sg_e.sg_only; "only"; end
check("sing-def-obj", sg_e.sg_only, "only")
check("sing-def-clean", SgC.new.respond_to?(:sg_only), false)
sg_f = SgC.new
class << sg_f
  def sg_cm; "override"; end
end
check("sing-eigen", sg_f.sg_cm + SgC.new.sg_cm, "overrideC")
# ----- Symbol is Comparable -----
# Symbol#<=> compares the NAMES, so #sort / #min / #max / #between? and the four
# relational operators all work on symbols. `:a <=> 1' is nil, which is what makes
# the mixed pair still raise. Ruby-2 forms; every want is /usr/bin/ruby 2.6.10's
# answer.
check("sym-sort", [:b, :a, :c].sort, [:a, :b, :c])
check("sym-cmp", [(:a <=> :b), (:b <=> :a), (:a <=> :a)], [-1, 1, 0])
check("sym-cmp-num", (:a <=> 1), nil)
check("sym-lt", (:a < :b), true)
check("sym-minmax", [[:b, :a].min, [:b, :a].max], [:a, :b])
check("sym-prefix", [:abc, :ab].sort, [:ab, :abc])
check("sym-between", :a.between?(:a, :z), true)
check("sym-keys-sort", {:b => 1, :a => 2}.keys.sort, [:a, :b])

# ----- puts / print as values, send to a top-level def, === by name -----
# puts and print are ordinary calls answering nil, so they are operands of && || and
# or not; a top-level def is a private Object method, so send reaches it on any
# receiver that has no method of the name; === by name is `when''s case equality.
# Ruby-2 forms; every want is /usr/bin/ruby 2.6.10's answer.
pe_v = [true, false, nil]
pe_r = []
pe_r << (pe_v[0] && print(""))
pe_r << (pe_v[1] || print(""))
pe_r << (pe_v[0] and print(""))
pe_r << (pe_v[2] or print "")
pe_r << (not pe_v[1] and print "")
check("puts-expr", pe_r, [nil, nil, nil, nil, nil])
def pe_h(n); n * 2; end
class PeC; def pe_h(n); n * 3; end; end
class PeD; end
check("send-toplevel-def", [self.send(:pe_h, 6), PeD.new.send(:pe_h, 2), PeD.new.__send__(:pe_h, 1),
                            3.send(:pe_h, 4), PeC.new.public_send(:pe_h, 1)], [12, 4, 2, 8, 3])
check("caseeq-by-name", ["a".===("a"), "a".send(:===, "b"), 5.===(5), Integer.===(3), (1..3).===(2),
                         /a/.===("cat"), String.send(:===, 1)], [true, false, true, true, true, true, false])

# ----- Array#to_h, define_method by String name, frozen Strings -----
# A Hash key is a frozen copy of the String it was stored under; nil, true, false,
# a Symbol and a number are frozen; `# frozen_string_literal: true' (in the
# required file) freezes that file's literals and shares the plain ones. Ruby-2
# forms; every want is /usr/bin/ruby 2.6.10's answer.
th_p = [["x", 1], [:y, 2], ["x", 3]]
check("array-to-h", [th_p.to_h, [].to_h, [1, 2].to_h { |x| [x, x * 2] }, {a: 1}.to_h { |k, v| [v, k] },
                     [].respond_to?(:to_h)], [{"x" => 3, :y => 2}, {}, {1 => 2, 2 => 4}, {1 => :a}, true])
th_e = [[1], [[1, 2, 3]]].map do |a|
  begin
    a.to_h
  rescue TypeError, ArgumentError => e
    "#{e.class}: #{e.message}"
  end
end
check("array-to-h-errors", th_e, ["TypeError: wrong element type Integer at 0 (expected array)",
                                  "ArgumentError: wrong array length at 0 (expected 2, was 3)"])
class DmC
  define_method("dm_str") { 42 }
  define_method('dm_sq') { |a, b| a + b }
  define_method(:dm_q?) { true }
end
check("define-method-names", [DmC.new.dm_str, DmC.new.dm_sq(1, 2), DmC.new.dm_q?], [42, 3, true])
fk_k = "key"
fk_h = {fk_k => 1, :s => 2}
fk_k2 = fk_h.keys[0]
fk_err = begin
  fk_k2 << "x"
  "none"
rescue FrozenError => e
  e.message
end
fk_fz = []
fk_h.each { |k, v| fk_fz << k.frozen? }
check("frozen-hash-key", [fk_err, fk_k.frozen?, fk_h.first[0].frozen?, fk_h.key(1).frozen?, fk_h.invert[1].frozen?, fk_fz],
      ["can't modify frozen String", false, true, true, true, [true, true]])
check("frozen-immediates", [nil.frozen?, true.frozen?, false.frozen?, :s.frozen?, 1.frozen?, [].frozen?],
      [true, true, true, true, true, false])
require_relative 'imports/frozenlit'
check("frozen-magic-comment", [fzl_lit.frozen?, fzl_interp(1).frozen?, fzl_words[1].frozen?, fzl_heredoc.frozen?,
                               fzl_append, "lit".frozen?, fzl_lit.dup.frozen?],
      [true, true, true, true, "can't modify frozen String", false, false])
check("frozen-magic-shared", [fzl_lit.equal?(fzl_lit), fzl_interp(1).equal?(fzl_interp(1)), fzl_words[0].equal?(fzl_words[0])],
      [true, false, true])

# ----- private setters, setter calls on any value, module attrs, a local hiding a method -----
# `o.x = v' is a call of x= whatever o is: a private one raises NoMethodError
# naming it, also where a subclass made an inherited attr_writer private, and a
# value with no x= (an Integer, a String) raises the same error.
class PvA
  attr_accessor :x
  private :x=
  def set(v); self.x = v; x; end
end
class PvB
  attr_accessor :y
end
class PvC < PvB
  private :y=
  def set(v); self.y = v; y; end
end
def pv_name
  yield
  :none
rescue NoMethodError => e
  e.name
end
pv_a = PvA.new
pv_c = PvC.new
pv_b = PvB.new
pv_b.y = 4
pv_n = 5
pv_s = "s"
pv_z = nil
check("private-setter", [pv_a.set(3), pv_name { pv_a.x = 5 }, pv_a.x], [3, :x=, 3])
check("private-setter-inherited", [pv_c.set(6), pv_name { pv_c.y = 7 }, pv_c.y, pv_b.y], [6, :y=, 6, 4])
check("setter-on-value", [pv_name { pv_n.x = 1 }, pv_name { pv_s.x = 1 }, pv_name { 5.x = 1 }, pv_name { pv_z.x = 1 }],
      [:x=, :x=, :x=, :x=])
module PvM
  attr_writer :w
  attr_accessor :a
end
class PvK
  include PvM
  def w; @w; end
end
pv_k = PvK.new
pv_k.w = 1
pv_k.a = 2
check("module-attr", [pv_k.w, pv_k.a, PvM.instance_methods.sort], [1, 2, [:a, :a=, :w=]])
def pv_k2; 9; end
def pv_shadow; pv_k2 = pv_k2; pv_k2; end
def pv_late; a = pv_k2; pv_k2 = 3; [a, pv_k2]; end
check("local-hides-method", [pv_shadow, pv_late], [nil, [9, 3]])

# ----- define_method binds its arguments as a method does -----
class PvD
  define_method(:one) { |a| a }
  define_method(:opt) { |a, b = 2| [a, b] }
  define_method(:two) { |a, b| [a, b] }
  [:dyn].each { |n| define_method(n) { |v| v } }
  pv_pr = proc { |a| a }
  define_method(:pv, pv_pr)
end
def pv_argc
  yield
  :none
rescue ArgumentError => e
  e.message
end
pv_d = PvD.new
check("define-method-argc", [pv_argc { pv_d.one }, pv_argc { pv_d.one(1, 2) }, pv_argc { pv_d.opt },
                             pv_argc { pv_d.dyn }, pv_argc { pv_d.pv }, pv_argc { pv_d.two([1, 2]) }],
      ["wrong number of arguments (given 0, expected 1)", "wrong number of arguments (given 2, expected 1)",
       "wrong number of arguments (given 0, expected 1..2)", "wrong number of arguments (given 0, expected 1)",
       "wrong number of arguments (given 0, expected 1)", "wrong number of arguments (given 1, expected 2)"])
check("define-method-bind", [pv_d.opt(1), pv_d.dyn(4), pv_d.two(1, 2)], [[1, 2], 4, [1, 2]])

# ----- Object#freeze on an instance, and Module reflection -----
# freeze sets the flag #frozen? and #clone read; a write to a frozen instance is
# not checked (MRI raises FrozenError).
class PvE
  PZ = 1
  PW = 2
  attr_accessor :v
  def m; end
  def pm; end
  protected :pm
  private
  def q; end
end
class PvF < PvE
  PN = 3
  def n; end
end
pv_e = PvE.new
pv_e.v = 1
check("freeze-instance", [pv_e.frozen?, pv_e.freeze.equal?(pv_e), pv_e.frozen?, pv_e.dup.frozen?, pv_e.clone.frozen?,
                          pv_e.clone.v, PvE.new.frozen?],
      [false, true, true, false, true, 1, false])
check("public-instance-methods", [PvE.public_instance_methods(false).sort, PvF.public_instance_methods(false),
                                  PvF.public_instance_methods.include?(:m), PvF.public_instance_methods.include?(:pm)],
      [[:m, :v, :v=], [:n], true, false])
check("method-defined", [PvE.method_defined?(:m), PvE.method_defined?(:pm), PvE.method_defined?(:q),
                         PvE.method_defined?("v="), PvF.method_defined?(:m), PvE.method_defined?(:zz)],
      [true, true, false, true, true, false])
check("module-constants", [PvE.constants.sort, PvF.constants.sort, PvF.constants(false)], [[:PW, :PZ], [:PN, :PW, :PZ], [:PN]])

puts "features: #{FEAT[0]} checks, #{FEAT[1]} failures"

exit(FEAT[1])
