# Full-syntax test: Ruby (3.x core language).
#
# This file belongs to the SECOND test group (./test.sh --full): it is NOT part
# of the default matrix. The goal of the metacompiler is to support the full
# languages; this file is the ratchet that measures how far the ruby grammars
# are. It walks the whole practical Ruby 3.x syntax, one self-contained
# SECTION per language area. The --full runner runs the file, and whenever a
# grammar aborts it removes the section around the error and retries - so the
# report lists every unsupported section, not just the first.
#
# Conventions (shared by every *-test-full.* file):
#   - prologue (before the first SECTION marker): the check helper only
#   - each section: '# ===== SECTION <nn>: <name> =====', top-level,
#     self-contained, no references to other sections
#   - the top-level driver calls each section via a line tagged
#     'SECTION-CALL <nn>' and prints 'full: <checks> checks, <failures> failures'
#   - the file ends with exit(failures), like the feature-matrix file
#     (exit 0 == full support, verified)
#
# The checks assert REAL MRI semantics (integer division floors, raise "x"
# wraps in RuntimeError, symbols are not strings, Array#== is structural), so
# a section only counts as supported once a grammar matches Ruby itself, not
# the feature-matrix subset (which deliberately diverges in those spots).
#
# Deliberately out of scope (not core syntax, or unrunnable here): require/
# gems and the stdlib (only core Kernel/Object methods: puts/exit as in the
# feature file, plus loop, format %, dup), threads/fibers/ractors, eval/
# binding and reflection (send, instance_variable_get), flip-flops, BEGIN/END
# blocks, magic comments (frozen_string_literal), refinements, __END__/DATA,
# and the 3.4 'it' parameter. Rational 1r / Complex 2i literals ARE covered
# (literal syntax); define_method appears exactly once. Sections 21-23 need
# MRI >= 3.1 (case/in, endless def, hash shorthand); 01-20 run on MRI >= 2.6.
#
# Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
# code), organized after the ISO/IEC 30170 Ruby specification chapters and
# the Ruby 3.x documentation, with ruby/spec used only as a coverage
# checklist.
#
# FULLC[0] counts checks, FULLC[1] failures (real Ruby defs cannot see
# top-level locals, and $globals stay inside section 07).
FULLC = [0, 0]
def check(id, cond)
  FULLC[0] = FULLC[0] + 1
  if !cond
    puts "FAIL #{id}"
    FULLC[1] = FULLC[1] + 1
  end
end
# ===== SECTION 01: baseline =====
# Condensed re-assertion of the feature-matrix basics this file builds on.
def s01_add(a, b)
  a + b
end
def s01
  n = 0
  i = 0
  while i < 3
    n += i
    i += 1
  end
  check("bas1", n == 3)
  h = {"a" => 1, "b" => 2}
  h["c"] = h["a"] + h["b"]
  check("bas2", h["c"] == 3)
  arr = [1, 2, 3]
  check("bas3", arr.length == 3 && arr[2] == 3)
  check("bas4", s01_add(20, 22) == 42)
  check("bas5", [1, 2, 3].map { |v| v * 2 }.sum == 12)
  check("bas6", (5 > 3 ? "y" : "n") == "y" && "n=#{2 + 2}" == "n=4")
end
# ===== SECTION 02: numeric literal forms =====
def s02
  check("num1", 0xff == 255 && 0XFF == 255)
  check("num2", 0b1010 == 10 && 0o17 == 15 && 017 == 15)
  check("num3", 1_000_000 == 1000000 && 1_000.5 == 1000.5)
  check("num4", 1.5 + 2.25 == 3.75 && 1e3 == 1000.0 && 25e-2 == 0.25)
  check("num5", 3r == 3 && 1r / 2r == 0.5)
  check("num6", 2i * 2i == -4)
  check("num7", 10_000_000_000 * 10 == 100_000_000_000)
end
# ===== SECTION 03: string literals, escapes, percent forms =====
def s03
  check("str1", "dq" == 'dq' && "ad" == "a" 'd')
  check("str2", "\x41B" == "AB" && "tab\tnl\n".length == 7)
  check("str3", 'kept \n #{raw}'.length == 14 && 'q\'q'.length == 3)
  check("str4", ?a == "a" && ?\n == "\n")
  check("str5", "ab" * 3 == "ababab")
  check("pct1", %q(it's raw) == "it's raw" && %Q{n=#{3 + 4}} == "n=7")
  check("pct2", %(bare #{1}) == "bare 1")
  check("pct3", %w[one two three] == ["one", "two", "three"])
  check("pct4", %W[a#{1} b#{2}] == ["a1", "b2"])
  check("pct5", %i[x y] == [:x, :y] && %I[k#{9}] == [:k9])
end
# ===== SECTION 04: heredocs =====
def s04
  a = <<~HD
    alpha #{2 * 3}
      indented
  HD
  check("hd1", a == "alpha 6\n  indented\n")
  b = <<-HD
mixed
  HD
  check("hd2", b == "mixed\n")
  c = <<~'HD'
    raw #{n}
  HD
  check("hd3", c == "raw \#{n}\n")
end
# ===== SECTION 05: symbols and ranges =====
def s05
  check("sym1", :ok == :ok && :ok != :no && :ok != "ok")
  check("sym2", :ok.to_s == "ok" && "up".to_sym == :up)
  check("sym3", :"two words".to_s == "two words" && :"n#{1 + 1}" == :n2)
  check("rng1", (1..4).to_a == [1, 2, 3, 4] && (1...4).to_a == [1, 2, 3])
  check("rng2", ("a".."c").to_a == ["a", "b", "c"] && ((1..3) === 2) == true)
  check("rng3", (1..9).include?(5) && (5..).cover?(99) && !(5..).cover?(4))
end
# ===== SECTION 06: string interpolation and formatting =====
def s06
  x = 6
  check("fmt1", "x is #{x}" == "x is 6" && "#{x}+#{x}=#{x + x}" == "6+6=12")
  check("fmt2", "outer #{"inner #{x}"}" == "outer inner 6")
  check("fmt3", "%d-%d" % [3, 4] == "3-4" && "%04d" % 42 == "0042")
  check("fmt4", "%.2f" % 1.5 == "1.50" && "%5s|" % "ab" == "   ab|")
  check("fmt5", "%x/%o/%b" % [255, 8, 5] == "ff/10/101")
  check("fmt6", "%s and %s" % ["a", "b"] == "a and b")
end
# ===== SECTION 07: variables and scope =====
S07_LIMIT = 40
$s07_hits = 0
class S07Counter
  @@made = 0
  def initialize(v); @v = v; @@made += 1; end
  def v; @v; end
  def self.made; @@made; end
end
def s07
  local = 2
  check("var1", S07_LIMIT + local == 42)
  $s07_hits += 1
  $s07_hits += 1
  check("var2", $s07_hits == 2)
  a = S07Counter.new(7); b = S07Counter.new(9)
  check("var3", a.v + b.v == 16 && S07Counter.made == 2)
  check("var4", defined?(S07_LIMIT) == "constant" && defined?($s07_hits) == "global-variable")
  check("var5", defined?(local) == "local-variable" && defined?(puts) == "method" && defined?(zz_undefined) == nil)
end
# ===== SECTION 08: multiple assignment and splats =====
def s08_pair
  return 3, 4
end
def s08
  a, b = 1, 2
  a, b = b, a
  check("mas1", a == 2 && b == 1) # swap
  x, y, z = [10, 20]
  check("mas2", x == 10 && y == 20 && z == nil)
  first, *rest = [1, 2, 3, 4]
  *init, last = [1, 2, 3]
  check("mas3", first == 1 && rest == [2, 3, 4] && init == [1, 2] && last == 3)
  (m, n), o = [5, 6], 7
  check("mas4", m == 5 && n == 6 && o == 7)
  pa, qa = s08_pair
  check("mas5", pa == 3 && qa == 4)
  check("mas6", [0, *[1, 2], 3] == [0, 1, 2, 3])
  base = {a: 1}
  check("mas7", {**base, b: 2} == {a: 1, b: 2} && {"k" => 1, v: 2}.size == 2)
end
# ===== SECTION 09: case/when dispatch =====
def s09_kind(v); case v; when Integer then "int"; when String, Symbol then "text"; when Array then "arr"; when nil then "nil"; else "other"; end; end
def s09_band(v); fours = [4, 5]; case v; when *fours then "45"; when 6..9 then "hi"; else "lo"; end; end
def s09
  check("cas1", s09_kind(3) == "int" && s09_kind(1.5) == "other")
  check("cas2", s09_kind("s") == "text" && s09_kind(:s) == "text")
  check("cas3", s09_kind([1]) == "arr" && s09_kind(nil) == "nil")
  check("cas4", s09_band(5) == "45" && s09_band(7) == "hi" && s09_band(1) == "lo")
  v = if false then "a" elsif true then "b" else "c" end
  u = unless false then "took" else "no" end
  check("cas5", v == "b" && u == "took")
end
# ===== SECTION 10: loop statements =====
def s10
  sum = 0
  for i in 1..3
    sum += i
  end
  check("lop1", sum == 6 && i == 3)
  got = loop { break 42 }
  check("lop2", got == 42)
  r = [1, 2, 3].each { |v| break v * 10 if v == 2 }
  check("lop3", r == 20)
  passes = 0; redone = 0
  [7, 8].each { |v| passes += 1; next if v == 7 || redone == 1; redone = 1; redo }
  check("lop4", passes == 3 && redone == 1)
  vals = [1, 2, 3].map { |v| next v * 2 if v < 3; v }
  check("lop5", vals == [2, 4, 3])
end
# ===== SECTION 11: blocks and yield =====
def s11_three; return "noblock" unless block_given?; yield 1; yield 2; yield 3; "done"; end
def s11_pass(&blk); [10, 20].each(&blk); end
def s11
  acc = 0
  r = s11_three { |v| acc += v }
  check("blk1", acc == 6 && r == "done")
  check("blk2", s11_three == "noblock")
  txt = ""
  [[1, [2, 3]], [4, [5, 6]]].each { |a, (b, c)| txt += "#{a}#{b}#{c}." }
  check("blk3", txt == "123.456.")
  got = []
  s11_pass { |v| got << v + 1 }
  check("blk4", got == [11, 21])
  check("blk5", [1, 2, 3].map(&:to_s) == ["1", "2", "3"] && ["ab", "cd"].map(&:upcase) == ["AB", "CD"])
end
# ===== SECTION 12: procs and lambdas =====
def s12
  sq = lambda { |x| x * x }
  check("prc1", sq.call(4) == 16 && sq.(5) == 25 && sq[6] == 36)
  add = ->(a, b = 10) { a + b }
  check("prc2", add.call(1, 2) == 3 && add.call(5) == 15)
  pr = proc { |a, b| "#{a}-#{b}" }
  check("prc3", pr.call(1, 2, 9) == "1-2" && pr.call(1) == "1-")
  check("prc4", add.lambda? == true && pr.lambda? == false)
  check("prc5", Proc.new { |x| x + 1 }.call(41) == 42)
  check("prc6", -> { return 7 }.call == 7)
end
# ===== SECTION 13: method parameter forms =====
def s13_def(a, b = 2, c = a + b); a + b + c; end
def s13_rest(first, *rest); "#{first}|#{rest.length}|#{rest[0]}"; end
def s13_kw(a, req:, opt: 5); a + req + opt; end
def s13_opts(k: 0, **rest); "#{k}:#{rest.size}"; end
def s13_blk(x, &blk); blk.call(x) + blk.call(x + 1); end
def s13_big?(n); n > 9; end
def s13_shout!(s); s.upcase; end
def s13_me; __method__; end
def s13
  check("mth1", s13_def(1) == 6 && s13_def(1, 5) == 12 && s13_def(1, 2, 3) == 6)
  check("mth2", s13_rest(9) == "9|0|" && s13_rest(1, 2, 3) == "1|2|2")
  check("mth3", s13_kw(1, req: 2) == 8 && s13_kw(1, opt: 1, req: 1) == 3)
  check("mth4", s13_opts(k: 3, z: 1, w: 2) == "3:2")
  check("mth5", s13_blk(5) { |v| v * 2 } == 22)
  check("mth6", s13_big?(10) == true && s13_big?(3) == false && s13_shout!("ok") == "OK")
  check("mth7", s13_me == :s13_me)
end
# ===== SECTION 14: operator method definitions =====
class S14Vec
  attr_reader :x, :y
  def initialize(x, y); @x = x; @y = y; end
  def +(o); S14Vec.new(@x + o.x, @y + o.y); end
  def -@; S14Vec.new(0 - @x, 0 - @y); end
  def ==(o); @x == o.x && @y == o.y; end
  def [](i); i == 0 ? @x : @y; end
  def []=(i, v); if i == 0 then @x = v else @y = v end; end
  def <=>(o); @x + @y <=> o.x + o.y; end
  def to_s; "(#{@x},#{@y})"; end
end
def s14
  check("opm1", S14Vec.new(1, 2) + S14Vec.new(3, 4) == S14Vec.new(4, 6))
  neg = -S14Vec.new(2, 5)
  check("opm2", neg[0] == -2 && neg[1] == -5)
  w = S14Vec.new(0, 0)
  w[0] = 7; w[1] = 8
  check("opm3", w[0] == 7 && w[1] == 8)
  check("opm4", (S14Vec.new(1, 1) <=> S14Vec.new(3, 3)) == -1)
  check("opm5", "#{S14Vec.new(9, 9)}" == "(9,9)")
  check("opm6", (1 <=> 2) == -1 && ("b" <=> "a") == 1 && (1 <=> "x") == nil)
end
# ===== SECTION 15: classes and inheritance =====
class S15Animal
  attr_reader :name
  def initialize(name); @name = name; end
  def speak; "..."; end
  def intro; "#{@name}:#{speak}"; end
  def tag(x); "A#{x}"; end
end
class S15Dog < S15Animal
  def initialize(name, bones); super(name); @bones = bones; end
  def speak; "woof#{@bones}"; end
  def tag(x); "D" + super; end
end
class S15Pup < S15Dog
  def speak; "yip+" + super; end
  def tag(x); "P" + super(x + 1); end
end
def s15
  d = S15Dog.new("rex", 2)
  check("cls1", d.name == "rex" && d.intro == "rex:woof2")
  p2 = S15Pup.new("pip", 1)
  check("cls2", p2.speak == "yip+woof1")
  check("cls3", d.tag(3) == "DA3" && p2.tag(3) == "PDA4")
  check("cls4", p2.is_a?(S15Animal) && p2.is_a?(S15Dog) && !d.is_a?(S15Pup))
  check("cls5", d.instance_of?(S15Dog) && !d.instance_of?(S15Animal))
end
# ===== SECTION 16: visibility, alias, singleton class =====
class S16Tool
  def self.kind; "tool"; end
  class << self; def twice; kind + kind; end; end
  def pub; helper + 1; end
  def real; 5; end
  alias short real
  alias_method :other, :real
  define_method(:dyn) { |x| x * 3 }
  private
  def helper; 41; end
end
class S16Acct
  def initialize(b); @b = b; end
  def richer?(o); balance > o.balance; end
  protected
  def balance; @b; end
end
def s16
  check("vis1", S16Tool.kind == "tool" && S16Tool.twice == "tooltool")
  t = S16Tool.new
  check("vis2", t.pub == 42 && (t.helper rescue "blocked") == "blocked")
  check("vis3", t.short == 5 && t.other == 5 && t.dyn(4) == 12)
  a = S16Acct.new(10); b = S16Acct.new(3)
  check("vis4", a.richer?(b) == true && b.richer?(a) == false)
  check("vis5", (a.balance rescue "blocked") == "blocked")
end
# ===== SECTION 17: modules and mixins =====
module S17Greet; WORD = "hi"; def greet; "#{WORD} #{gname}"; end; end
module S17Util; def util_tag; "u!"; end; end
module S17Loud; def word; super.upcase; end; end
module S17Outer
  module Inner; VALUE = 7; def self.value; VALUE * 2; end; end
end
module S17Calc
  module_function
  def calc_double(x); x * 2; end
end
class S17Person
  include S17Greet
  extend S17Util
  def initialize(n); @n = n; end
  def gname; @n; end
end
class S17Word; prepend S17Loud; def word; "abc"; end; end
class S17Size
  include Comparable
  attr_reader :n
  def initialize(n); @n = n; end
  def <=>(o); n <=> o.n; end
end
def s17
  check("mod1", S17Person.new("bo").greet == "hi bo")
  check("mod2", S17Person.util_tag == "u!")
  check("mod3", S17Word.new.word == "ABC")
  check("mod4", S17Outer::Inner::VALUE == 7 && S17Outer::Inner.value == 14)
  check("mod5", S17Calc.calc_double(21) == 42)
  check("mod6", S17Size.new(1) < S17Size.new(2) && S17Size.new(5).between?(S17Size.new(1), S17Size.new(9)))
end
# ===== SECTION 18: exceptions =====
class S18Err < StandardError; end
class S18Sub < S18Err; def initialize(msg = "sub-default"); super; end; end
def s18_boom(k)
  raise S18Err, "coded #{k}" if k == 1
  raise S18Sub if k == 2
  raise "plain" if k == 3
  raise ArgumentError.new("arg") if k == 4
  "ok"
end
def s18_class(k)
  s18_boom(k)
rescue S18Sub => e; "sub:#{e.message}"
rescue S18Err => e; "err:#{e.message}"
rescue ArgumentError, TypeError => e; "either:#{e.message}"
rescue => e; "std:#{e.message}"
end
def s18_retry; tries = 0; begin; tries += 1; raise S18Err if tries < 3; tries; rescue S18Err; retry; end; end
def s18_order(fail_it)
  log = ""
  begin; log += "b"; raise S18Err, "x" if fail_it; log += "B"; rescue S18Err; log += "r"; else; log += "e"; ensure; log += "n"; end
  log
end
def s18_reraise
  begin; raise S18Err, "orig"; rescue S18Err; raise; end
rescue S18Err => e; "re:#{e.message}"
end
def s18_ensure_ret(k)
  begin; return "body" if k == 0; raise S18Err, "boom"; ensure; return "ensure-wins" if k == 2; end
rescue S18Err => e; "caught:#{e.message}"
end
def s18
  check("exc1", s18_class(1) == "err:coded 1" && s18_class(9) == "ok")
  check("exc2", s18_class(2) == "sub:sub-default")
  check("exc3", s18_class(3) == "std:plain")
  check("exc4", s18_class(4) == "either:arg")
  check("exc5", s18_retry == 3)
  check("exc6", s18_order(false) == "bBen" && s18_order(true) == "brn")
  check("exc7", s18_reraise == "re:orig")
  check("exc8", (s18_boom(1) rescue "rescued") == "rescued" && (s18_boom(0) rescue "x") == "ok")
  check("exc9", s18_ensure_ret(0) == "body" && s18_ensure_ret(1) == "caught:boom")
  check("exc10", s18_ensure_ret(2) == "ensure-wins")
end
# ===== SECTION 19: operator zoo =====
def s19
  check("ops1", 2 ** 10 == 1024 && 2 ** 3 ** 2 == 512)
  check("ops2", -7 / 2 == -4 && 7 / 2 == 3 && -7 % 3 == 2 && 7.0 / 2 == 3.5)
  check("ops3", (5 & 3) == 1 && (5 | 3) == 7 && (5 ^ 3) == 6)
  check("ops4", ~5 == -6 && (1 << 4) == 16 && (32 >> 2) == 8)
  la = (true and false)
  lo = (false or true) # rubocop hates these; the parser must not
  check("ops5", la == false && lo == true && (not false) == true)
  x = false or true
  check("ops6", x == false)
  s = "a".dup
  s << "bc"
  arr = [1]; arr << 2 << 3
  check("ops7", s == "abc" && arr == [1, 2, 3])
  check("ops8", "x"&.upcase == "X" && nil&.upcase == nil)
end
# ===== SECTION 20: regexp literals =====
def s20
  re = /ab+c/
  check("rex1", re.match?("xabbcy") == true && re.match?("ac") == false)
  check("rex2", ("cabbage" =~ /b+/) == 2 && ("zzz" =~ /b/) == nil)
  m = /(\d+)-(\d+)/.match("on 10-25!")
  check("rex3", m[1] == "10" && m[2] == "25" && "id=42".match(/(?<num>\d+)/)[:num] == "42")
  check("rex4", /abc/i.match?("xABCy") && /a b/x.match?("ab"))
  check("rex5", /a.c/m.match?("a\nc") == true && /a.c/.match?("a\nc") == false)
  check("rex6", %r{a/b}.match?("xa/by") == true)
  "y7" =~ /(\d)/
  check("rex7", $1 == "7")
  kind = case "grape"; when /gr/ then "match"; else "no"; end
  check("rex8", kind == "match" && 8 / 2 / 2 == 2)
end
# ===== SECTION 21: pattern matching with case/in =====
def s21_pm(v)
  case v
  in 0 then "zero"
  in Integer => n if n > 100
    "big:#{n}"
  in 1 | 2 | 3 then "small"
  in [x, y] then "pair:#{x + y}"
  in {kind: "circle", r:} then "circle:#{r}"
  in String then "str"
  else "other"
  end
end
def s21
  check("pma1", s21_pm(0) == "zero" && s21_pm(500) == "big:500")
  check("pma2", s21_pm(2) == "small" && s21_pm(9.5) == "other")
  check("pma3", s21_pm([2, 3]) == "pair:5")
  check("pma4", s21_pm({kind: "circle", r: 7}) == "circle:7")
  check("pma5", s21_pm("s") == "str")
  r = case [1, [2, 3]]; in [a, [b, c]] then a + b + c; end
  check("pma6", r == 6)
  h = case {u: 1, v: 2, w: 3}; in {u: Integer => uu, **rest} then "#{uu}+#{rest.size}"; end
  check("pma7", h == "1+2")
end
# ===== SECTION 22: pattern deconstruction, pin, find =====
class S22Pt
  attr_reader :x, :y
  def initialize(x, y); @x = x; @y = y; end
  def deconstruct; [x, y]; end
  def deconstruct_keys(keys); {x: x, y: y}; end
end
def s22
  pin = 5
  r1 = case [5, 6]; in [^pin, b] then "pin:#{b}"; else "no"; end
  r2 = case [8, 6]; in [^pin, b] then "pin:#{b}"; else "no"; end
  check("pmb1", r1 == "pin:6" && r2 == "no")
  r3 = case [1, 7, 42, 9]; in [*pre, 42, *post] then "#{pre.size}/#{post.size}"; else "no"; end
  check("pmb2", r3 == "2/1")
  r4 = case S22Pt.new(1, 2); in [a, b] then a + b; else -1; end
  r5 = case S22Pt.new(3, 4); in {x:, y:} then x * y; else -1; end
  check("pmb3", r4 == 3 && r5 == 12)
  r6 = case {a: 1}; in {a: Integer} => whole then whole[:a] + 10; end
  check("pmb4", r6 == 11)
  {u: 9, w: 2} => {u:}
  check("pmb5", u == 9)
  check("pmb6", (5 in Integer) == true && ("x" in Integer) == false)
end
# ===== SECTION 23: ruby 3 shorthands =====
def s23_sq(x) = x * x
def s23_answer = 42
def s23_sum3(a, b, c); a + b + c; end
def s23_fwd(...); s23_sum3(...); end
def s23
  check("r3a", s23_sq(9) == 81 && s23_answer == 42)
  x = 4
  y = 6
  check("r3b", {x:, y:} == {x: 4, y: 6}) # value omission needs plain locals
  check("r3c", [1, 2, 3].map { _1 * 2 } == [2, 4, 6])
  check("r3d", [[1, 2], [30, 40]].map { _1 + _2 } == [3, 70])
  br = (..9)
  check("r3e", br.cover?(3) == true && br.cover?(10) == false)
  check("r3f", s23_fwd(20, 21, 1) == 42)
end
# ===== END SECTIONS =====
s01() # SECTION-CALL 01
s02() # SECTION-CALL 02
s03() # SECTION-CALL 03
s04() # SECTION-CALL 04
s05() # SECTION-CALL 05
s06() # SECTION-CALL 06
s07() # SECTION-CALL 07
s08() # SECTION-CALL 08
s09() # SECTION-CALL 09
s10() # SECTION-CALL 10
s11() # SECTION-CALL 11
s12() # SECTION-CALL 12
s13() # SECTION-CALL 13
s14() # SECTION-CALL 14
s15() # SECTION-CALL 15
s16() # SECTION-CALL 16
s17() # SECTION-CALL 17
s18() # SECTION-CALL 18
s19() # SECTION-CALL 19
s20() # SECTION-CALL 20
s21() # SECTION-CALL 21
s22() # SECTION-CALL 22
s23() # SECTION-CALL 23
puts "full: #{FULLC[0]} checks, #{FULLC[1]} failures"
exit(FULLC[1])
