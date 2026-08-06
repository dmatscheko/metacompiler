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
# Section 31 needs MRI >= 3.1 as well; its expected values come from ruby/spec's own
# descriptions in language/pattern_matching_spec.rb, which name each form outright.
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
  # The FUSED `x - floor(x/y)*y`. Read out of an array so nothing folds it, and
  # written where fusion CAN bite: q*y leaves the exactly-representable range, so
  # the two-rounding answer is one whole unit out. abnf/jsrt.go says math.FMA,
  # ruby-rt.metajs says rbFmaSub and ruby-interpreter.abnf says fmaSub; all three
  # answer what /usr/bin/ruby answers, which is the fused one.
  fm = [9007199254740991, -3, -7, 9007199254740992, 18014398509481982]
  check("num8", fm[0] % fm[1] == -2 && fm[0] % fm[2] == -4)
  check("num9", fm[3] % fm[1] == -1 && fm[3] % fm[2] == -3 && fm[4] % fm[2] == -1)
  check("num10", fm[0].divmod(fm[1]) == [-3002399751580331, -2] && fm[3].divmod(fm[2]) == [-1286742750677285, -3])
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
  # %g/%G/%E, and the + / space / 0 flags. Every expected string here was read
  # off ruby 2.6.10, whose Float#% is unchanged in 3.x.
  check("fmt7", "%g" % 1234567.0 == "1.23457e+06" && "%g" % 100.0 == "100" && "%g" % 0.0001 == "0.0001")
  check("fmt8", "%g" % 0.00001 == "1e-05" && "%.3g" % 123456.0 == "1.23e+05" && "%.10g" % 1234567.0 == "1234567")
  check("fmt9", "%G" % 1234567.0 == "1.23457E+06" && "%g" % 999999.5 == "1e+06" && "%.0g" % 2.5 == "2")
  check("fmt10", "%e" % 1.5 == "1.500000e+00" && "%.2e" % 1234.5678 == "1.23e+03" && "%E" % 1.5 == "1.500000E+00")
  check("fmt11", "%+d" % 7 == "+7" && "% d" % 7 == " 7" && "%+g" % 1.5 == "+1.5" && "%05d" % -42 == "-0042")
  check("fmt12", "%05s" % "ab" == "   ab" && "%f" % (1.0 / 0.0) == "Inf" && "%.0f" % 2.5 == "2" && "%012.4g" % 0.0 == "000000000000")
  # The # (alternate) flag: %g keeps its trailing zeros, a decimal point is forced
  # where there would be none, %x/%X/%o/%b take a base prefix (but a zero does
  # not), and the 0 flag pads BEHIND that prefix. Read off ruby 2.6.10 as well.
  check("fmt13", "%#g" % 100.0 == "100.000" && "%#g" % 1.5 == "1.50000" && "%#G" % 1.0e20 == "1.00000E+20")
  check("fmt14", "%#.3g" % 100.0 == "100." && "%#.1g" % 1234.0 == "1.e+03" && "%#.0g" % 1.5 == "2.")
  check("fmt15", "%#.3g" % 0.0001 == "0.000100" && "%#12.4g" % 0.0 == "       0.000" && "%#-12.3g" % 1.5 == "1.50        ")
  check("fmt16", "%#012.3g" % -1.5 == "-00000001.50" && "%#+g" % 1.5 == "+1.50000" && "%#g" % (1.0 / 0.0) == "Inf")
  check("fmt17", "%#x" % 255 == "0xff" && "%#X" % 255 == "0XFF" && "%#o" % 8 == "010" && "%#b" % 5 == "0b101" && "%#x" % 0 == "0")
  check("fmt18", "%#010x" % 255 == "0x000000ff" && "%#-10x" % 255 == "0xff      " && ("%#g %d" % [1.5, 7]) == "1.50000 7" && "%#.0f" % 1.0 == "1." && "%#.0e" % 1.0 == "1.e+00")
  # %d TRUNCATES a negative float, it does not floor it. The interpreter floored.
  check("fmt19", "%d" % -1.5 == "-1" && "%d" % -0.5 == "0" && "%i" % 2.9 == "2" && "%05d" % -1.5 == "-0001")
  # ----- the three integer directives, all read out of an ARRAY ---------------
  # An integer directive converts the Float with to_i and prints the resulting
  # Integer EXACTLY. %d used to print the double's shortest form ("1e+30") and
  # %x/%o/%b saturated at int64 ("7fffffffffffffff").
  big = [1.0e20, 1.0e30, -1.0e20, 9007199254740992.0, 1.0e300]
  check("fmt20", "%d" % big[0] == "100000000000000000000" && "%d" % big[1] == "1000000000000000019884624838656")
  check("fmt21", "%x" % big[0] == "56bc75e2d63100000" && "%o" % big[3] == "400000000000000000" &&
                 "%b" % big[3] == "100000000000000000000000000000000000000000000000000000")
  check("fmt22", ("%d" % big[4]).length == 301 && ("%x" % big[4]).length == 250 && "%d" % -big[1] == "-1000000000000000019884624838656")
  # A NEGATIVE under %x/%o/%b is MRI's INFINITE two's complement, and the leading
  # run of f/7/1 collapses to exactly one digit.
  neg = [-1, -5, -255, -256, -4096, -17]
  check("fmt23", "%x" % neg[1] == "..fb" && "%o" % neg[1] == "..73" && "%b" % neg[1] == "..1011" && "%X" % neg[0] == "..F")
  check("fmt24", "%x" % neg[2] == "..f01" && "%x" % neg[3] == "..f00" && "%x" % neg[4] == "..f000" && "%x" % neg[5] == "..fef")
  # The 0 flag fills that form with the base's LARGEST digit, behind the ".." and
  # behind any # prefix; + or SPACE switches back to sign-and-magnitude entirely.
  check("fmt25", "%010x" % neg[1] == "..fffffffb" && "%#010x" % neg[1] == "0x..fffffb" && "%10x" % neg[1] == "      ..fb")
  check("fmt26", "%+x" % neg[2] == "-ff" && "% x" % neg[1] == "-5" && "%#+x" % neg[1] == "-0x5" && "%#o" % neg[1] == "..73")
  # A precision on an integer directive is a MINIMUM DIGIT COUNT that also cancels
  # the 0 flag - and "%.0x" % 0 is the empty string. In the ".." form the two dots
  # count as two of the precision's characters.
  zed = [0, 255, 1, -0.0]
  check("fmt27", "%.0x" % zed[0] == "" && "%.0d" % zed[0] == "" && "%.0d" % zed[3] == "" && "%#.0o" % zed[0] == "0")
  check("fmt28", "%.3x" % zed[1] == "0ff" && "%.3d" % neg[0] == "-001" && "%08.3x" % zed[2] == "     001")
  check("fmt29", "%.5x" % neg[0] == "..fff" && "%.5b" % neg[0] == "..111" && "%.3x" % neg[1] == "..fb" && "%#.5x" % zed[0] == "00000")
  check("fmt30", "%#.5o" % zed[1] == "00377" && "%#.1o" % zed[1] == "0377" && "%#.5x" % zed[1] == "0x000ff")
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
  # Ruby has POSIX BRACKET EXPRESSIONS, plus [[:word:]] and [[:ascii:]] of its own.
  # It is NOT a POSIX ERE, though: \d, \s, lazy quantifiers and (?...) all still
  # work, so the shared engine has to be told "POSIX classes" and nothing else.
  check("rex9", /[[:alpha:]]/.match?("x") == true && /[[:alpha:]]/.match?("1") == false)
  check("rex10", /^[[:digit:]]+$/.match?("123") == true &&
                 /[[:upper:][:space:]]/.match?("a b") == true)
  check("rex11", /[[:word:]]/.match?("_") == true && /[^[:digit:]]/.match?("x") == true)
  check("rex12", /[[:alpha:]]\d/.match?("a1") == true && /[[:alpha:]]\s/.match?("a ") == true)
  check("rex13", /(a+?)/.match("aaa")[1] == "a")
  # Ruby is on the OTHER side of the repeat-semantics divide from JavaScript and bash:
  # the empty final iteration is KEPT, so group 1 reads "" and not "aaa".
  check("rex14", /^(a*)*$/.match("aaa")[1] == "")
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
# ===== SECTION 24: assignment targets and compound operators =====
class S24Box
  def initialize
    @v = 0
  end
  def v
    @v
  end
  def v=(n)
    @v = n
  end
  def bump
    self.v = self.v + 1
    self.v
  end
end
def s24
  b = S24Box.new
  b.v = 7
  check("t1", b.v == 7)
  check("t2", b.bump == 8)
  n = 32
  n |= 64
  check("t3", n == 96)
  n &= 12
  check("t4", n == 0)
  m = 5
  m ^= 3
  check("t5", m == 6)
  p2 = 3
  p2 **= 3
  check("t6", p2 == 27)
  sh = 1
  sh <<= 4
  check("t7", sh == 16)
  a, = [1, 2, 3]
  check("t8", a == 1)
  x = nil
  true && false && x = 1
  check("t9", x == nil)
  y = 1 || false && y = 2
  check("t10", y == 1)
end
# ===== SECTION 25: statement sequences, rescue targets, nested declarations =====
class S25Outer
  class S25Inner
    def who
      "inner"
    end
  end
  S25_IN_CLASS = 3
  def make
    S25Inner.new
  end
end
S25_CONST = 11
class S25Err
  def initialize
    @e = nil
    @caught = nil
  end
  def caught
    @caught
  end
  def run
    begin
      raise "boom"
    rescue => @caught
      1
    end
    @caught.message
  end
  def late
    begin
      raise "again"
    rescue
      S25_CONST
    end
  end
end
def s25_ensured
  [1, 2].each do |i|
    i
  ensure
    $s25_ensure = ($s25_ensure || 0) + 1
  end
  :done
end
def s25
  check("p1", (1; 2; 3) == 3)
  check("p2", () == nil)
  check("p3", [0, (), 2].size == 3)
  check("p4", S25Outer.new.make.who == "inner")
  check("p4b", S25Outer::S25Inner.new.who == "inner")
  check("p4c", S25Outer::S25_IN_CLASS == 3)
  check("p5", S25Err.new.run == "boom")
  check("p6", S25Err.new.late == 11)
  check("p7", ::S25_CONST == 11)
  $s25_ensure = 0
  check("p8", s25_ensured == :done && $s25_ensure == 2)
end
# ===== SECTION 26: symbols, heredoc placement and command calls =====
def s26_take(text, extra)
  text + extra.to_s
end
def s26
  check("y1", :foo?.to_s == "foo?")
  check("y2", :foo=.to_s == "foo=")
  check("y3", :+.to_s == "+")
  check("y4", :[]=.to_s == "[]=")
  check("y5", :@iv.to_s == "@iv")
  check("y6", :@@cv.to_s == "@@cv")
  check("y7", :'quoted'.to_s == "quoted")
  check("y8", (+"copy") == "copy")
  $0 == $0
  glued = s26_take(<<~TXT, 5)
    line one
  TXT
  check("y9", glued == "line one\n5")
  tail = <<-RAW.size
      abc
      RAW
  check("y10", tail == 10)
  acc = []
  acc.push 1
  acc.push 2
  check("y11", acc == [1, 2])
  f = lambda { |a = 5, b = 4| a + b }
  check("y12", f.call == 9)
  r = [1].map { next 7, 8 }
  check("y13", r == [[7, 8]])
end
# ===== SECTION 27: destructuring and splat assignment =====
def s27_pair
  [7, 8]
end
def s27
  a = *[1, 2]
  check("d1", a == [1, 2])
  b = *nil
  check("d2", b == [])
  c = 1, 2, 3
  check("d3", c == [1, 2, 3])
  d, e = 1
  check("d4", d == 1 && e == nil)
  f, = [4, 5, 6]
  check("d5", f == 4)
  g, h, = [4, 5, 6]
  check("d6", g == 4 && h == 5)
  (*i) = nil
  check("d7", i == [nil])
  *, j = [1, 2, 3]
  check("d8", j == 3)
  (k, l), m = [1, 2], 3
  check("d9", k == 1 && l == 2 && m == 3)
  n, (o, p) = 1, [2, 3]
  check("d10", n == 1 && o == 2 && p == 3)
  q, r, *s = *[1, 2, 3, 4]
  check("d11", q == 1 && r == 2 && s == [3, 4])
  t, u = s27_pair
  check("d12", t == 7 && u == 8)
end
# ===== SECTION 28: calls, arguments and blocks =====
def s28_join(text, extra)
  text + extra.to_s
end
def s28_hash(h)
  h
end
def s28_block(a)
  yield a
end
def s28_one
  1
end
class S28Sup
  def hi(x)
    "sup:" + x
  end
end
class S28Sub < S28Sup
  def hi(x)
    super(x) + "|" + (block_given? ? yield : "noblk")
  end
end
def s28
  check("c1", s28_join("a", 1) == "a1")
  taken = s28_join "b", 2
  check("c2", taken == "b2")
  h = s28_hash("a" => 1, :b => 2)
  check("c3", h.size == 2 && h["a"] == 1)
  arr = ["foo" => :bar, baz: 42]
  check("c4", arr.size == 1 && arr[0].size == 2)
  check("c5", s28_block(3) { |v| v * 2 } == 6)
  outer = 9
  check("c6", [1].map { |one; outer| one } == [1])
  check("c7", outer == 9)
  check("c8", S28Sub.new.hi("x") { "blk" } == "sup:x|blk")
  check("c9", __FILE__.length > 0 && __LINE__ > 0)
  check("c10", [10, 20][1] == 20)
  f = lambda { |a = 5, b = 4| a + b }
  check("c11", f.call == 9)
  check("c12", s28_hash(1, ) == 1)
  # A parenthesis-less lambda parameter list: the braces are the lambda's BODY, not a
  # block on the default value's call.
  lam = -> a=s28_one() { a }
  check("c13", lam.call == 1)
  lam2 = -> b { b + 1 }
  check("c14", lam2.call(1) == 2)
end
# ===== SECTION 29: literals with unusual spellings =====
def s29
  ip = "xxx"
  check("l1", %!hey #{ip}! == "hey xxx")
  check("l2", %@hey #{ip}@ == "hey xxx")
  check("l3", %<hey #{ip}> == "hey xxx")
  check("l4", %=hey #{ip}= == "hey xxx")
  check("l5", %s{plain}.to_s == "plain")
  check("l6", %r!a+!.match("baaa").to_a[0] == "aaa")
  check("l7", "a#{}b" == "ab")
  multi = "one
two"
  check("l8", multi.length == 7)
  check("l9", ?z == "z")
  check("l10", ?\C-a == "\x01")
  check("l11", ?\M-\C-z == "\x9A")
  check("l12", {a!: 1, b?: 2}.size == 2)
  check("l13", {a!: 1}[:a!] == 1)
  check("l14", {"d": 4}[:d] == 4)
  check("l15", :ilétait.to_s.length == 7)
  check("l16", %q!raw #{ip}! == "raw \#{ip}")
end
# ===== SECTION 30: declarations as expressions, and body clauses =====
class S30Base
  def kind
    "base"
  end
end
# A class body is an EXPRESSION whose value is its last statement's; a def is one whose
# value is the method name. Both have to sit at top level (Ruby forbids a class
# definition inside a method body), so the section only asserts on what they produced.
S30_EMPTY_VAL = (class S30Empty; end)
S30_TWENTY_VAL = (class S30Twenty; 20; end)
S30_DEF_VAL = def s30_made; 5; end
# `class << OBJ ... end` opens the singleton class of a VALUE anywhere, not only as
# `class << self` inside a class body, and it is an EXPRESSION - the `.should` in a spec
# suite hangs off its `end`. Its value is its last statement's.
S30_META_VAL = (class << S30Base; 7; end)
def s30
  check("e1", S30_EMPTY_VAL == nil)
  check("e2", S30_TWENTY_VAL == 20)
  check("e3", S30_DEF_VAL == :s30_made && s30_made == 5)
  flag = true
  n = 0
  while flag do
    n += 1
    flag = false
  end
  check("e4", n == 1)
  m = 0
  for k in [1, 2] do
    m += k
  end
  check("e5", m == 3)
  obj = S30Holder.new
  for obj.slot in [1, 2, 3]
    m += obj.slot
  end
  check("e6", obj.slot == 3 && m == 9)
  for (a, b) in [[1, 2]]
    check("e7", a == 1 && b == 2)
  end
  check("e8", S30_META_VAL == 7)
end
class S30Holder
  attr_accessor :slot
end
# ===== SECTION 31: pattern matching, the bracket-less and Constant forms =====
# Needs MRI >= 3.1, so /usr/bin/ruby 2.6 cannot check these. The expected values come
# from ruby/spec's own descriptions in language/pattern_matching_spec.rb, which name
# each form outright: "supports form pat, pat, ...", "supports form id: pat, id: pat,
# ...", "supports form Constant(pat, pat, ...)", "supports form Constant[id: pat, ...]",
# "does match partially from the array beginning if list + , syntax used" and
# "matches anything with *".
class S31Point
  attr_reader :x, :y
  def initialize(x, y)
    @x = x
    @y = y
  end
  def deconstruct
    [@x, @y]
  end
  def deconstruct_keys(keys)
    {x: @x, y: @y}
  end
end
def s31_kind(v)
  case v
  in a: 1, b: 1
    "hash-bare"
  in Hash(k: Integer => n)
    "hash-const:" + n.to_s
  in Array(0, 1, 2)
    "arr-const"
  in Array[9, 8]
    "arr-brk"
  in [7, 6, ]
    "arr-partial"
  in 5, 4
    "arr-bare"
  in {"q": 1}
    "quoted-key"
  in *
    "any-array"
  else
    "other"
  end
end
# The pin `^@ivar` reads the CURRENT value of the instance variable, so it needs a
# method to run in (a top-level @ivar has no self to hang on here).
class S31Pinned
  def initialize
    @pin = 4
  end
  def test(v)
    case v
    in ^@pin
      "pinned"
    else
      "no"
    end
  end
end
def s31
  check("m1", s31_kind({a: 1, b: 1}) == "hash-bare")
  check("m2", s31_kind({k: 3}) == "hash-const:3")
  check("m3", s31_kind([0, 1, 2]) == "arr-const")
  check("m4", s31_kind([9, 8]) == "arr-brk")
  check("m5", s31_kind([7, 6, 5, 4]) == "arr-partial")
  check("m6", s31_kind([5, 4]) == "arr-bare")
  check("m7", s31_kind({"q": 1}) == "quoted-key")
  check("m8", s31_kind([1]) == "any-array")
  check("m9", s31_kind("z") == "other")
  p1 = S31Point.new(1, 2)
  check("m10", (case p1; in S31Point(x: 1, y: y); y; else 0; end) == 2)
  check("m11", (case p1; in S31Point[1, b]; b; else 0; end) == 2)
  check("m12", S31Pinned.new.test(4) == "pinned" && S31Pinned.new.test(5) == "no")
  check("m13", (case 2; in ^(1 + 1); "computed"; else "no"; end) == "computed")
  [1, 2] => q, w
  check("m14", [q, w] == [1, 2])
  {a: 3} => a:
  check("m15", a == 3)
end
# ===== SECTION 32: interpreter/compiler agreement ratchet =====
# Every check here is a divergence that WAS live between ruby-interpreter.abnf
# and ruby-to-llvm-ir.abnf (or a silent data loss in the shared runtime), with
# the expected value taken from MRI itself. The matrix cannot see this class of
# bug - it compares each engine against itself under goja and -frozen - so the
# assertions live here.
class S32Klass
end
def s32
  # Array#push is variadic; the compiler runtime kept only the first argument.
  a = [1]
  a.push(2, 3, 4)
  check("n01", a.length == 4)
  check("n02", a == [1, 2, 3, 4])
  # nil's conversions. nil.to_s was "null" in the compiler and "" in the interpreter.
  check("n03", nil.to_s == "")
  check("n04", "[" + nil.to_s + "]" == "[]")
  check("n05", "[#{nil}]" == "[]")
  check("n06", nil.inspect == "nil")
  check("n07", nil.to_i == 0)
  check("n08", nil.to_a == [])
  # A class object stringifies as its name, not as "[object Object]".
  check("n09", S32Klass.to_s == "S32Klass")
  check("n10", "#{S32Klass}" == "S32Klass")
  check("n11", 1.class.to_s == "Integer")
  # MatchData#to_a aborted in the compiler (the .to_a suffix was the identity).
  m = "hello world".match(/(\w+) (\w+)/)
  check("n12", m.to_a == ["hello world", "hello", "world"])
  check("n13", m.captures == ["hello", "world"])
  # Hash#to_a is the list of pairs; it used to return the hash itself.
  h = {"a" => 1, "b" => 2}
  check("n14", h.to_a == [["a", 1], ["b", 2]])
  # Ruby to_s rendering of the container types.
  check("n15", [1, 2].to_s == "[1, 2]")
  check("n16", 1.0.to_s == "1.0")
  # Float#to_s uses RUBY'S exponent window, not JavaScript's: numeric.c flo_to_s
  # goes exponential when decpt < -3 or decpt > 15, the mantissa always carries a
  # fraction digit and the exponent is always signed and at least two digits. Every
  # literal below is ruby 2.6.10p210's own answer. Before the fix the right-hand
  # sides read 1000000000000000.0 / 1234567890123456.0 / 0.00001 / 100000000000000000000.
  check("n16b", 1e15.to_s == "1.0e+15" && 999999999999999.0.to_s == "999999999999999.0")
  check("n16c", 1234567890123456.0.to_s == "1.234567890123456e+15" &&
                1e20.to_s == "1.0e+20")
  check("n16d", 1e-5.to_s == "1.0e-05" && 0.0001.to_s == "0.0001" &&
                5.0e-324.to_s == "5.0e-324")
  # -0.0 keeps its sign. The compiled half already answered this; the interpreter's
  # unary minus was `0 - x`, which returns +0.0 for a zero operand.
  check("n16e", (-0.0).to_s == "-0.0" && 0.0.to_s == "0.0" &&
                (-1e16).to_s == "-1.0e+16")
  # Float#% - numeric.c flodivmod. The remainder takes the DIVISOR's sign, but a
  # ZERO remainder keeps the DIVIDEND's, because flodivmod starts from fmod and
  # its `mod += y` correction cannot fire on a zero. Ours computed
  # x - floor(x/y)*y, where -4.0 - (-4.0) is +0.0 and the sign was lost. Every
  # right-hand side below is ruby 2.6.10p210's own answer.
  check("n16f", (-0.0 % 1.0).to_s == "-0.0" && (0.0 % 1.0).to_s == "0.0")
  check("n16g", (-4.0 % 2.0).to_s == "-0.0" && (4.0 % 2.0).to_s == "0.0")
  check("n16h", (-4.0 % -2.0).to_s == "-0.0" && (4.0 % -2.0).to_s == "0.0" &&
                (-7.5 % 2.5).to_s == "-0.0")
  # ...and the non-zero remainders, which the divisor's sign already governed and
  # which the fix must leave exactly where they were.
  check("n16i", (-5.0 % 2.0).to_s == "1.0" && (5.0 % -2.0).to_s == "-1.0" &&
                (-0.0 % -1.0).to_s == "-0.0" && (0.0 % -1.0).to_s == "0.0")
  # ...and the DIVIDENDS BIG ENOUGH THAT floor(x/y)*y IS NO LONGER EXACT, which is
  # the rest of flodivmod and the reason `%` is now literally
  # `mod = fmod(x, y); if (y*mod < 0) mod += y`. The old x - floor(x/y)*y answered
  # 0.0 for all of these in the interpreter and a rounded multiple of y in the
  # compiler - the two halves did not even agree. 1e16 is the smallest one here:
  # it is not an astronomical corner. Every right-hand side is ruby 2.6.10p210's.
  check("n16j", (1.0e308 % 3.0).to_s == "2.0" && (-1.0e308 % 3.0).to_s == "1.0")
  check("n16k", (1.0e308 % 2.5).to_s == "1.0" && (-1.0e308 % 2.5).to_s == "1.5" &&
                (1.0e16 % 3.0).to_s == "1.0")
  check("n16l", (123456789012345678.0 % 7.0).to_s == "3.0" &&
                (1.0e300 % 7.3).to_s == "0.7299664879653811" &&
                (-1.0e300 % 7.3).to_s == "6.570033512034619")
  # An infinite divisor needs no special case: fmod(-5, +Inf) is -5 and
  # +Inf * -5 is -Inf, so the correction fires and -5.0 % Infinity is Infinity.
  s32inf = 1.0 / 0.0
  check("n16m", (5.0 % s32inf).to_s == "5.0" && (-5.0 % s32inf).to_s == "Infinity" &&
                (5.0 % (0.0 - s32inf)).to_s == "-Infinity" &&
                (-5.0 % (0.0 - s32inf)).to_s == "-5.0")
  check("n17", true.to_s == "true")
  check("n18", :sym.to_s == "sym")
end

# ===== SECTION 33: Integer at arbitrary precision =====
# Ruby's Integer has no width. Every value here is outside what a double counts
# exactly (2^53 == 9007199254740992), which is the ONLY width at which these
# rules can bite: written one bit lower, every assertion below passes against a
# plain double too. The oracle is /usr/bin/ruby 2.6.10 - Integer values and
# Integer formatting are unchanged in 3.x - and all four engines are diffed
# against it by tests/probe.sh.
def s33
  # The literal, in all four radices plus the bare-zero octal and '_'.
  check("b01", 9007199254740993.to_s == "9007199254740993")
  check("b02", 0x20000000000001.to_s == "9007199254740993")
  check("b03", 0xffffffffffffffff.to_s == "18446744073709551615")
  check("b04", 12345678901234567890.to_s == "12345678901234567890")
  check("b05", 0b1000000000000000000000000000000000000000000000000000001.to_s == "18014398509481985")
  check("b06", 0o1000000000000000000000.to_s == "9223372036854775808")
  check("b07", 1_2345_6789_0123_4567_890.to_s == "12345678901234567890")
  check("b08", (-9007199254740993).to_s == "-9007199254740993")
  # 2^53 itself is exact and must NOT change shape, and neither must anything
  # below it - this is the guard on the promotion trigger.
  check("b09", 9007199254740992.to_s == "9007199254740992")
  check("b10", 9007199254740994.to_s == "9007199254740994")
  check("b11", 0xff == 255 && 017 == 15 && 0b1011 == 11 && 1_000_000 == 1000000)
  # Arithmetic. Every operand comes out of an array, so a constant folder cannot
  # answer any of these at compile time.
  v = [9007199254740992, 12345678901234567890, 1000, 97, 2, 70, -7]
  check("b12", (v[0] + 1).to_s == "9007199254740993")
  check("b13", (v[0] * v[0]).to_s == "81129638414606681695789005144064")
  check("b14", (v[1] - 1).to_s == "12345678901234567889")
  check("b15", (v[1] / v[2]).to_s == "12345678901234567")
  check("b16", (v[1] % v[3]).to_s == "3")
  # Integer / and % FLOOR at every width, as they do for a small Integer.
  check("b17", ((-v[1]) / v[2]).to_s == "-12345678901234568")
  check("b18", ((-v[1]) % v[3]).to_s == "94")
  check("b19", (v[4] ** v[5]).to_s == "1180591620717411303424")
  check("b20", v[1].divmod(v[6]) == [-1763668414462081128, -6])
  # Comparison has to be EXACT: through a double both sides of b21 round to 2^53.
  check("b21", 9007199254740993 != 9007199254740992)
  check("b22", (v[1] <=> v[1] + 1) == -1 && ((v[1] + 1) <=> v[1]) == 1)
  check("b23", v[1] > 9007199254740992 && v[1] >= v[1] && !(v[1] < v[1]))
  # ... including against a Float, which no rounding of the Integer can settle.
  check("b24", 9007199254740993 != 9007199254740993.0)
  check("b25", 18446744073709551616 == 18446744073709551616.0)
  # Rendering: an Integer is NEVER written in exponent form, at any magnitude.
  check("b26", (2 ** 100).to_s == "1267650600228229401496703205376")
  check("b27", "#{v[1] * v[1]}" == "152415787532388367501905199875019052100")
  check("b28", [v[1]].to_s == "[12345678901234567890]")
  check("b29", (1.0e21).to_i.to_s == "1000000000000000000000")
  check("b30", (2.0 ** 70).to_i.to_s == "1180591620717411303424")
  # to_s(base) and the integer format directives, all past 2^53.
  check("b31", 18446744073709551615.to_s(16) == "ffffffffffffffff")
  check("b32", 12345678901234567890.to_s(36) == "2lsohxawjui8i")
  check("b33", (2 ** 70).to_s(2).length == 71)
  check("b34", ("%d|%x|%o" % [v[1], v[1], v[1]]) == "12345678901234567890|ab54a98ceb1f0ad2|1255245230635307605322")
  check("b35", ("%030d" % v[1]) == "000000000012345678901234567890")
  check("b36", ("%x" % (-v[1])) == "..f54ab567314e0f52e")
  # The bit operators are INFINITE two's complement at arbitrary precision.
  check("b37", (1 << 100).to_s == "1267650600228229401496703205376")
  check("b38", ((1 << 100) >> 90).to_s == "1024")
  check("b39", (~18446744073709551615).to_s == "-18446744073709551616")
  check("b40", (~9007199254740992).to_s == "-9007199254740993")
  check("b41", (1 | 9007199254740992).to_s == "9007199254740993")
  check("b42", (-1 & 0xffffffffffffffff).to_s == "18446744073709551615")
  check("b43", (18446744073709551615 ^ 18446744073709551615) == 0)
  # succ and pred cross the boundary the double cannot.
  check("b44", 9007199254740992.succ.to_s == "9007199254740993")
  check("b45", 9007199254740993.pred.to_s == "9007199254740992")
  # The predicates and conversions, exact rather than through the double.
  check("b46", (-12345678901234567890).abs.to_s == "12345678901234567890")
  check("b47", 18446744073709551615.odd? && 18446744073709551616.even?)
  check("b48", 12345678901234567890.class.to_s == "Integer")
  check("b49", 12345678901234567890.is_a?(Integer) && 12345678901234567890.is_a?(Numeric))
  check("b50", (-12345678901234567890).negative? && 12345678901234567890.positive?)
  # A big narrows back to a plain Integer the moment it fits again, so the two
  # shapes are one value: ==, <=> and Hash keying all have to agree.
  check("b51", (12345678901234567890 - 12345678901234567889) == 1)
  check("b52", ((2 ** 70) / (2 ** 70)) == 1)
  bh = {18446744073709551616 => "big", 1 => "one"}
  check("b53", bh[18446744073709551616] == "big")
  check("b54", bh[9223372036854775808 * 2] == "big")
  check("b55", bh.size == 2)
  # A big meeting a Float promotes to Float, which is the tower's own rule.
  check("b56", (18446744073709551616 + 0.5).class.to_s == "Float")
  check("b57", 18446744073709551616.to_f.class.to_s == "Float")
end

# ===== SECTION 34: the value-key rule, Comparable and the Kernel writers =====
# docs/todo.md 1.6, 2.9 and 2.11. Every right-hand side is ruby 2.6.10p210's own
# answer, taken from tests/probe.sh's oracle leg; the four legs (interpreter,
# llvm.Run, the native binary and MRI) agree on all of it.
def s34
  # Float#to_s: MRI's HIGH test has TWO parts. `decpt > 15` alone is not it -
  # exponential also needs `decpt >= digits.length`, i.e. it is used only when the
  # fixed form would have to pad with zeros. So a 16-digit-point value with 17
  # digits stays fixed while one with 16 digits does not. Before the fix the first
  # right-hand side read 3.0023997515803305e+15. Settled against /usr/bin/ruby
  # over 3,000 random doubles in [1e15, 1e19).
  check("v01", 9007199254740991.fdiv(3).to_s == "3002399751580330.5")
  check("v02", 1234567890123456.5.to_s == "1234567890123456.5")
  check("v03", 1234567890123456.0.to_s == "1.234567890123456e+15")
  check("v04", 12345678901234567.0.to_s == "1.2345678901234568e+16")
  check("v05", 9999999999999999.0.to_s == "1.0e+16" && 1e14.to_s == "100000000000000.0")
  # Hash#inspect writes NO SPACES around the arrow.
  check("v06", {1 => 2}.to_s == "{1=>2}")
  check("v07", {1 => 2, "a" => [1, 2], :b => nil}.inspect == "{1=>2, \"a\"=>[1, 2], :b=>nil}")
  check("v08", {1 => {2 => 3}}.to_s == "{1=>{2=>3}}")
  # An ARRAY is a VALUE key: Array#hash and Array#eql? are structural. This was
  # nil in all three engines, and it is the language's own dict path that decides
  # it - not the strict_eq / rtDictFind that nine languages share.
  vh = {[1] => 5, [1, 2] => 8}
  check("v09", vh[[1]] == 5 && vh[[1, 2]] == 8)
  check("v10", vh[[2]] == nil)
  vh2 = {}
  vh2[[3, 4]] = 9
  vh2[[3, 4]] = 10
  check("v11", vh2[[3, 4]] == 10 && vh2.size == 1)
  check("v12", {{1 => 2} => 3}[{1 => 2}] == 3)
  check("v13", [[1], [2]].include?([2]) && [[1]].index([1]) == 0)
  # Array#- and Array#*: the interpreter had neither. `-` ABORTED and `*`
  # answered a silent NaN, which is the worse of the two.
  v14a = [1, 2, 3, 2]
  check("v14", (v14a - [2]) == [1, 3])
  check("v15", (v14a * 2) == [1, 2, 3, 2, 1, 2, 3, 2] && ([1] * 0) == [])
  check("v16", (["x", "y", "x"] - ["x"]) == ["y"])
  # Comparable#between? / #clamp are defined by <=> alone, so String answers them
  # too - and a big Integer answers EXACTLY rather than through the double.
  v17n = [5, 0, 20, 2.5]
  check("v17", v17n[0].between?(1, 9) && !v17n[1].between?(1, 9))
  check("v18", v17n[2].clamp(1, 9) == 9 && v17n[1].clamp(1, 9) == 1)
  check("v19", v17n[3].between?(1, 9) && v17n[3].clamp(4, 9) == 4)
  check("v20", "m".between?("a", "z") && "A".clamp("a", "z") == "a")
  check("v21", (2 ** 64 + 1).between?(2 ** 64, 2 ** 64 + 2))
  check("v22", (2 ** 64).clamp(2 ** 64 + 1, 2 ** 64 + 2) == 2 ** 64 + 1)
  # Array#inspect / index / rindex / uniq, and Hash#inspect: missing from EVERY
  # engine. index and uniq compare by value, like the Hash key rule above.
  v23a = [1, 2, 3, 2, 4]
  check("v23", v23a.inspect == "[1, 2, 3, 2, 4]" && [1, "a", nil, 2.5].inspect == "[1, \"a\", nil, 2.5]")
  check("v24", v23a.index(2) == 1 && v23a.rindex(2) == 3 && v23a.index(99) == nil)
  check("v25", v23a.uniq == [1, 2, 3, 4] && [[1], [2], [1]].uniq == [[1], [2]])
  check("v26", {1 => 2}.inspect == "{1=>2}")
  # TrueClass / FalseClass to_s and inspect: both compiled halves aborted with
  # "method call 'inspect' on a boolean".
  check("v27", true.inspect == "true" && false.inspect == "false" && (1 == 1).inspect == "true")
  # Kernel#p answers its argument (the whole list for more than one) and writes
  # INSPECT. Kernel#printf is print(format(...)): the compiler used to resolve the
  # bare name to the SHARED host printf, which is not Ruby's and was not even one
  # answer between llvm.Run and the native binary.
  check("v28", (p "a") == "a")
  check("v29", (p 1, 2) == [1, 2])
  check("v30", sprintf("%05.2f", 1.5) == "01.50" && format("%d", 3) == "3")
  # Ruby draws a line between == and eql?, and the Array methods do not all sit
  # on the same side of it: include?/index are ==, so 1 and 1.0 match; uniq and
  # Array#- are eql?/hash, so they do NOT. Every engine used to answer the `===`
  # of the shared runtime for all four, which is neither.
  check("v31", [1].include?(1.0) && [1].index(1.0) == 0)
  check("v32", [1, 1.0, 1].uniq == [1, 1.0] && ([1, 2.0] - [2]) == [1, 2.0])
  check("v33", ([2.0] - [2.0]) == [] && [2.0, 2.0].uniq == [2.0])
  # String#inspect is the QUOTED form, and it was missing from both compiled
  # halves - so `p "a"` and any .inspect reaching a String diverged.
  check("v34", "a".inspect == "\"a\"" && nil.inspect == "nil" && :y.inspect == ":y")
  # A Float is a BOX, and two boxes of one value are two objects: `[1.0] == [1.0]`
  # was FALSE in the NATIVE BINARY and true under llvm.Run (whose Float is a Go
  # value struct) and in MRI. A run-vs-native divergence in layer 2 that the
  # matrix, --full and --cross are all blind to by construction.
  check("v35", [1.0] == [1.0] && [1, 1.0] == [1, 1.0] && [1] == [1.0])
  check("v36", ({1 => 1.0} == {1 => 1.0}) && [[1.5]] == [[1.5]])
end

# ===== SECTION 35: the Array / Enumerable surface =====
# docs/todo.md 1.4. Every right-hand side is ruby 2.6.10p210's own answer, taken
# from a 5,729-line four-leg probe (interpreter, llvm.Run, the native binary and
# MRI) on which all four legs are byte-identical. Each operand is read out of an
# array, so no constant folder can answer these at compile time.
def s35
  a = [3, 1, 2]
  b = [1, [2, [3, [4]]]]
  c = [1, nil, 2, nil]
  d = ["bb", "a", "ccc"]
  # The twelve that aborted in BOTH halves.
  check("w01", a.sort == [1, 2, 3] && a.sort { |x, y| y <=> x } == [3, 2, 1])
  # join RECURSES into a nested array with the same separator and uses to_s, so
  # nil contributes "".
  check("w02", [1, 2, 3].join("-") == "1-2-3" && [1, [2, [3]]].join(",") == "1,2,3")
  check("w03", c.join("|") == "1||2|" && [1, 2].join == "12")
  check("w04", a.reverse == [2, 1, 3] && a.min == 1 && a.max == 3)
  # min/max take a COUNT (max's answer is DESCENDING) and a block.
  check("w05", a.min(2) == [1, 2] && a.max(2) == [3, 2] && a.min { |x, y| y <=> x } == 3)
  check("w06", [1, 2, 2, 3].count == 4 && [1, 2, 2, 3].count(2) == 2)
  check("w07", [1, 2, 2, 3].count { |x| x > 1 } == 3)
  # flatten takes a DEPTH argument; without one it goes all the way down.
  check("w08", b.flatten == [1, 2, 3, 4] && b.flatten(1) == [1, 2, [3, [4]]])
  check("w09", c.compact == [1, 2] && c.length == 4)
  # sort_by is the stable decorate-sort; sort is not stable in MRI, and one
  # stable sort satisfies both contracts.
  check("w10", d.sort_by { |s| s.length } == ["a", "bb", "ccc"])
  # zip pads a short argument with nil and is always as long as the RECEIVER.
  check("w11", [1, 2].zip([3, 4], [5, 6]) == [[1, 3, 5], [2, 4, 6]])
  check("w12", [1, 2].zip([9]) == [[1, 9], [2, nil]])
  e = [1, 2, 3]
  check("w13", e.shift == 1 && e == [2, 3])
  f = [1, 2, 3, 4, 5]
  sl = []
  f.each_slice(2) { |g| sl << g }
  check("w14", sl == [[1, 2], [3, 4], [5]])
  check("w15", f.each_cons(2).to_a == [[1, 2], [2, 3], [3, 4], [4, 5]])
  # Array#& and Array#| aborted in ALL THREE engines with "not a number in '&'".
  # Both keep the LEFT-hand order, both dedupe, and both compare with eql?.
  check("w16", ([1, 2, 3] & [3, 2, 9]) == [2, 3] && ([2, 1] | [1, 3]) == [2, 1, 3])
  check("w17", ([1, 1, 2] & [1, 2]) == [1, 2] && ([1, 1] | []) == [1])
  check("w18", ([1] & [1.0]) == [] && ([1] | [1.0]) == [1, 1.0])
  # first/last/pop/shift take a COUNT and then answer an ARRAY. Without it they
  # answered the single element in every engine - a SILENT wrong answer.
  check("w19", [1, 2, 3].first(2) == [1, 2] && [1, 2, 3].last(2) == [2, 3])
  g = [1, 2, 3]
  check("w20", g.pop(2) == [2, 3] && g == [1])
  h = [1, 2, 3]
  check("w21", h.shift(2) == [1, 2] && h == [3])
  # Array.new answered a bare object in the interpreter and the argument itself
  # under llvm.Run - wrong, and differently wrong, in every engine.
  check("w22", Array.new(3, 0) == [0, 0, 0] && Array.new == [] && Array.new(2) == [nil, nil])
  check("w23", Array.new(3) { |i| i * i } == [0, 1, 4])
  # sum accumulated through ToInt32, so it truncated Floats and wrapped at 2^31.
  check("w24", [1.5, 2.5].sum == 4.0 && [1, 2, 3].sum == 6 && [1, 2].sum(10) == 13)
  check("w25", [1, 2, 3].sum { |x| x * 2 } == 12)
  # inject/reduce in all four forms, including the Symbol one.
  check("w26", [1, 2, 3].inject(:+) == 6 && [1, 2, 3].inject(:*) == 6)
  check("w27", [1, 2, 3].inject(10) { |s, x| s + x } == 16 && [].inject(:+) == nil)
  check("w28", [1, 2, 3].find { |x| x > 1 } == 2 && [1, 2, 3].detect { |x| x > 9 } == nil)
  check("w29", [1, 2, 3].partition { |x| x > 1 } == [[2, 3], [1]])
  check("w30", [1, 2, 3].group_by { |x| x % 2 } == {1 => [1, 3], 0 => [2]})
  check("w31", [1, 2, 3].flat_map { |x| [x, x] } == [1, 1, 2, 2, 3, 3])
  check("w32", [1, 2, 3].each_with_object([]) { |x, m| m << x * 2 } == [2, 4, 6])
  check("w33", [1, 2, 3].take(2) == [1, 2] && [1, 2, 3].drop(2) == [3])
  check("w34", [1, 2, 3].take_while { |x| x < 3 } == [1, 2] && [1, 2, 3].drop_while { |x| x < 3 } == [3])
  check("w35", [1, 2, 3].all? { |x| x > 0 } && [1, 2, 3].any? { |x| x > 2 })
  check("w36", [1, 2, 3].none? { |x| x > 9 } && [1, 2, 3].one? { |x| x == 2 })
  # A non-block argument is tested with ===, so a class matches by is_a?.
  check("w37", [1, "a"].any?(String) && ![1, "a"].all?(Integer))
  check("w38", [3, 1].min_by { |x| -x } == 3 && [3, 1].max_by { |x| -x } == 1)
  check("w39", [3, 1, 2].minmax == [1, 3] && [].minmax == [nil, nil])
  check("w40", [1, 2, 3].rotate == [2, 3, 1] && [1, 2, 3].rotate(-1) == [3, 1, 2])
  check("w41", [1, 2, 3].values_at(0, 2, 9) == [1, 3, nil])
  check("w42", [1, 2, 3].slice(1, 2) == [2, 3] && [1, 2, 3].slice(1..2) == [2, 3])
  check("w43", [1, 2, 3].slice(1) == 2 && [1, 2, 3].slice(-1) == 3)
  check("w44", [1, 2, 3].insert(1, 9) == [1, 9, 2, 3])
  # delete answers the LAST element it removed, which is not always the argument:
  # `[1, 1.0].delete(1)` removes both (== , not eql?) and MRI answers 1.0.
  check("w45", [1, 1.0].delete(1) == 1.0 && [1, 2].delete(9) == nil)
  check("w46", [1, 2, 3].delete_at(1) == 2 && [1, 2, 3].delete_at(9) == nil)
  # select!/reject! answer NIL when they removed nothing; keep_if/delete_if
  # always answer self.
  check("w47", [1, 2].select! { |x| x > 0 } == nil && [1, 2].reject! { |x| x > 9 } == nil)
  check("w48", [1, 2].select! { |x| x > 1 } == [2] && [1, 2].keep_if { |x| x > 0 } == [1, 2])
  check("w49", [1, 2].uniq! == nil && [1, 1].uniq! == [1])
  check("w50", [1, 2].compact! == nil && [1, nil].compact! == [1])
  check("w51", [1, 2, 3].product([4, 5]).length == 6 && [1].product([2]) == [[1, 2]])
  check("w52", [[1, 2], [3, 4]].assoc(3) == [3, 4] && [[1, 2]].assoc(9) == nil)
  check("w53", [1, 2].empty? == false && [].empty? == true && [1, 2].clear == [])
  check("w54", [1, 2].unshift(0) == [0, 1, 2] && [1].concat([2, 3]) == [1, 2, 3])
  check("w55", [1, 2].fill(7) == [7, 7] && [1, 2, 3].index { |x| x > 1 } == 1)
  i = [1, 2, 3]
  check("w56", i.map! { |x| x * 2 } == [2, 4, 6] && i == [2, 4, 6])
  j = [3, 1]
  check("w57", j.sort! == [1, 3] && j == [1, 3] && [1, 2].reverse! == [2, 1])
  # Array#* is repetition with an Integer and JOIN with a String. The String form
  # answered an EMPTY ARRAY in both compiled halves - a silent wrong answer.
  check("w58", ([1, 2] * ",") == "1,2" && ([1, 2] * 2) == [1, 2, 1, 2])
  # The three halves divergences: String#to_f, String#to_i and Kernel#rand all
  # worked in the interpreter and aborted in the compiler.
  check("w59", "3.5".to_f == 3.5 && "-2.75".to_f == -2.75 && "abc".to_f == 0.0)
  check("w60", "12abc".to_i == 12 && "abc".to_i == 0 && "3".to_i == 3)
  # rand is deterministically zero here (no random source can be byte-identical
  # across the engines), but zero OF THE RIGHT CLASS, which is what is testable.
  check("w61", rand(5).is_a?(Integer) && rand().is_a?(Float) && rand(0).is_a?(Float))
  check("w62", rand(5) >= 0 && rand(5) < 5 && rand() < 1.0)
  # Kernel.format with an explicit receiver aborted in both halves, where the
  # bare name has always worked.
  check("w63", Kernel.format("%05.2f", 3.14159) == "03.14" && Kernel.format("%d", 5) == "5")
  # true/false are their own classes: `false == 0` is FALSE. The shared pyEqual
  # said true, so `[true, false].include?(0)` was a run-vs-everything divergence.
  check("w64", ![true, false, nil].include?(0) && [false].include?(false))
  # A bare Hash.new is an empty Hash - it answered a bare object here.
  k = Hash.new
  k[:a] = 1
  check("w65", k == {:a => 1})
  # docs/todo.md 1.4, the rest of it. Hash.new(v) / Hash.new { |h, k| } keep their
  # default in a hidden slot honoured at EVERY read, including the op-assign and the
  # append reads, which used to go straight to dictFind and answer nil.
  hd = Hash.new(7)
  check("w66", hd["nope"] == 7 && hd.size == 0 && hd.keys == [] && hd.inspect == "{}")
  hd[:a] = 1
  check("w67", hd[:a] == 1 && hd[:b] == 7 && hd.default == 7 && Hash.new.default == nil)
  hc = Hash.new(0)
  ["a", "b", "a"].each { |w| hc[w] += 1 }
  check("w68", hc == {"a" => 2, "b" => 1})
  hb = Hash.new { |h, k| h[k] = [] }
  hb[:x] << 1
  hb[:x] << 2
  check("w69", hb == {:x => [1, 2]} && hb.default == nil)
  # fetch DELIBERATELY ignores the default, which is MRI.
  hfb = hd.fetch(:zz) { |q| 9 }
  check("w70", hd.fetch(:a) == 1 && hd.fetch(:zz, 5) == 5 && hfb == 9)
  # A blockless each / each_slice / each_with_index / combination / permutation is an
  # Enumerator, not the array of elements it would produce. This one is EAGER - there
  # is no coroutine in the compiler half to park a lazy one in.
  ea = [1, 2, 3, 4, 5]
  en = ea.each
  check("w71", en.class.to_s == "Enumerator" && en.is_a?(Enumerator) && en.size == 5)
  check("w72", en.to_a == [1, 2, 3, 4, 5] && en.next == 1 && en.next == 2 && en.peek == 3)
  en.rewind
  check("w73", en.next == 1 && ea.each_slice(2).class.to_s == "Enumerator")
  check("w74", ea.each_slice(2).to_a == [[1, 2], [3, 4], [5]])
  check("w75", ea.each_with_index.to_a[0] == [1, 0] && ea.each.map { |v| v * 2 } == [2, 4, 6, 8, 10])
  check("w76", ea.each.with_index(1).to_a[0] == [1, 1])
  # sample and shuffle are asserted on INVARIANTS ONLY - length, membership and
  # permutation-ness. There is no random source here (Kernel#rand is
  # deterministically zero of the right class), so the draw is deterministic and its
  # particular ordering is not a contract.
  check("w77", ea.include?(ea.sample) && ea.sample(3).length == 3 && ea.sample(99).length == 5)
  check("w78", [].sample == nil && (ea.sample(3) - ea) == [])
  sf = ea.shuffle
  check("w79", sf.length == 5 && sf.sort == ea && (sf - ea) == [] && (ea - sf) == [])
  cy = []
  ea.cycle(2) { |v| cy << v }
  check("w80", cy.length == 10 && cy.sort == [1, 1, 2, 2, 3, 3, 4, 4, 5, 5])
  check("w81", ea.combination(2).to_a.length == 10 && ea.permutation(2).to_a.length == 20)
  check("w82", ["a", "b", "c"].combination(2).to_a == [["a", "b"], ["a", "c"], ["b", "c"]])
  check("w83", ["a", "b", "c"].permutation(2).to_a.length == 6 && ea.combination(0).to_a == [[]])
  check("w84", ea.combination(9).to_a == [] && ea.cycle(2).to_a.length == 10)
  # A BARE rand: it used to resolve to the host function value here and abort the
  # compiler with "variable not defined: rand", while rand() worked everywhere.
  check("w85", rand.class.to_s == "Float" && rand >= 0 && rand < 1 && rand().class.to_s == "Float")
  # yield inside a begin/rescue ABORTED the compiler half while this one answered.
  y1 = s35y("a") { 1 }
  y2 = s35y2 { |m| m.upcase }
  check("w86", y1 == "a1" && y2 == "BOOM")
  bg2 = s35bg { 1 }
  check("w87", s35bg == [false, false] && bg2 == [true, true] && !block_given?)
end
def s35y(l)
  begin
    return l + (yield).to_s
  rescue Exception => e
    return "E"
  end
end
def s35y2
  begin
    raise "boom"
  rescue => e
    return yield(e.message)
  end
end
def s35bg
  r = block_given?
  begin
    return [r, block_given?]
  rescue
    return nil
  end
end

# ===== SECTION 36: Array is Comparable, and the two errors sort and sum raise =====
# The interpreter half had NO Array#<=> while both compiled halves did, so
# `[[2],[1]].sort` aborted there and worked everywhere else - a halves divergence
# ./test.sh --cross could not see, because no test program sorts an array of
# arrays. Every right-hand side below is ruby 2.6.10p210's own answer.
def s36
  # Element-wise, recursing into nested arrays, and a shorter array that is a
  # PREFIX of the longer one sorts first.
  check("x01", [[2], [1]].sort == [[1], [2]])
  check("x02", [[1, 2], [1]].sort == [[1], [1, 2]])
  check("x03", [["b"], ["a"]].sort == [["a"], ["b"]])
  check("x04", [[1, 1], [1], [2], []].sort == [[], [1], [1, 1], [2]])
  check("x05", [[[2]], [[1]]].sort == [[[1]], [[2]]])
  check("x06", [[1, [2]], [1, [3]]].sort == [[1, [2]], [1, [3]]])
  # The answer is NORMALISED to -1/0/1: `[1,2,3] <=> [1]` is 1, not the length
  # difference 2, which is what a first shape of this answered.
  check("x07", ([1, 2] <=> [1, 3]) == -1 && ([1, 2, 3] <=> [1]) == 1)
  check("x08", ([1] <=> [1, 2, 3]) == -1 && ([] <=> []) == 0 && ([1, 2, 3] <=> [1, 2, 3]) == 0)
  check("x09", ([["b"]] <=> [["a"]]) == 1 && ([[1, 2]] <=> [[1, 3]]) == -1)
  # ONE incomparable element pair makes the WHOLE comparison nil - carrying on to
  # the next element made `[1,"a"] <=> [1,2]` answer 0.
  check("x10", ([1, "a"] <=> [1, 2]) == nil && ([1, 2] <=> [1, "a"]) == nil)
  check("x11", ([1, 2] <=> "x") == nil && (1 <=> "a") == nil)
  # min / max / sort_by go through the same comparator.
  check("x12", [[1], [2]].min == [1] && [[3], [1], [2]].max == [3])
  check("x13", [[1, 2], [1]].max == [1, 2] && [[1, 9], [1, 2]].min == [1, 2])
  check("x14", [[1], [2], [1]].sort_by { |x| x } == [[1], [1], [2]])
  check("x15", [1, 2].sort_by { |x| [x, x] } == [1, 2])
  # sort is STABLE here, which MRI does not promise but cannot contradict.
  check("x16", [[1], [1]].sort == [[1], [1]])
  # An incomparable pair raises a CATCHABLE ArgumentError, and rb_cmperr's message
  # names the first operand by its CLASS and the second by its class unless it is a
  # special constant or a Float, which are inspected.
  caught = nil
  begin
    [1, "a"].sort
  rescue ArgumentError => e
    caught = e.message
  end
  check("x17", caught == "comparison of Integer with String failed")
  caught2 = nil
  begin
    [[1], ["a"]].sort
  rescue StandardError => e2
    caught2 = e2.message
  end
  check("x18", caught2 == "comparison of Array with Array failed")
  caught3 = nil
  begin
    [1, nil].max
  rescue ArgumentError => e3
    caught3 = e3.message
  end
  check("x19", caught3 == "comparison of NilClass with 1 failed")
  caught4 = nil
  begin
    1 < "a"
  rescue ArgumentError => e4
    caught4 = e4.message
  end
  check("x20", caught4 == "comparison of Integer with String failed")
  # sum REFUSES a non-numeric element against a numeric accumulator with a
  # catchable TypeError. It used to answer 0 (the ToInt32 accumulator swallowed
  # everything) and then "0bac" once the accumulator became a real +.
  caught5 = nil
  begin
    ["b", "a", "c"].sum
  rescue TypeError => e5
    caught5 = e5.message
  end
  check("x21", caught5 == "String can't be coerced into Integer")
  caught6 = nil
  begin
    [1.0, "a"].sum
  rescue TypeError => e6
    caught6 = e6.message
  end
  check("x22", caught6 == "String can't be coerced into Float")
  caught7 = nil
  begin
    [nil].sum
  rescue StandardError => e7
    caught7 = e7.message
  end
  check("x23", caught7 == "nil can't be coerced into Integer")
  # The refusal is only against a NUMERIC accumulator: a String or an Array start
  # value is exactly how MRI concatenates.
  check("x24", ["a", "b"].sum("") == "ab" && [[1], [2]].sum([]) == [1, 2])
  check("x25", [1, 2].sum == 3 && [1.5, 2.5].sum == 4.0)
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
s24() # SECTION-CALL 24
s25() # SECTION-CALL 25
s26() # SECTION-CALL 26
s27() # SECTION-CALL 27
s28() # SECTION-CALL 28
s29() # SECTION-CALL 29
s30() # SECTION-CALL 30
s31() # SECTION-CALL 31
s32() # SECTION-CALL 32
s33() # SECTION-CALL 33
s34() # SECTION-CALL 34
s35() # SECTION-CALL 35
s36() # SECTION-CALL 36
puts "full: #{FULLC[0]} checks, #{FULLC[1]} failures"
exit(FULLC[1])
