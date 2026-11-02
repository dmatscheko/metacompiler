# Full-syntax test: Ruby - part 3, SECTIONs 97 to 126 and 184 onward. tests/ruby-test-full.rb
# leaves the -frozen interpreter leg too little of the IR interpreter's step
# brake (-max-steps, 100M per call) for them; see that file for the
# conventions.
#
# FULLC[0] counts checks, FULLC[1] failures.
FULLC = [0, 0]
def check(id, cond)
  FULLC[0] = FULLC[0] + 1
  if !cond
    puts "FAIL #{id}"
    FULLC[1] = FULLC[1] + 1
  end
end
# ===== SECTION 97: Strings a builtin builds are new objects =====
# s[i], s[a..b], s[i, n], format / sprintf and the Strings each_char / each_line /
# upto yield are new mutable Strings: changing one leaves the source alone and is
# seen through every reference to it. A Hash's String key reads out as one frozen
# object, the same for every read and for every Hash keyed by that text.
# /usr/bin/ruby 2.6 settles each.
def s97
  v = ["hello", "%d-x", 5, "a\nb\n", "a", "c"]
  s = v[0]
  t = s[1, 3]
  t2 = t
  t2 << "!"
  c = s[0]
  c2 = c
  c2 << "Z"
  rg = s[0..1]
  rg << "Q"
  check("sb97a", s == "hello" && t == "ell!" && c == "hZ" && rg == "heQ")
  f = format(v[1], v[2])
  g = f
  g << "y"
  f2 = sprintf(v[1], v[2])
  check("sb97b", f == "5-xy" && f.equal?(g) && f2.equal?(f2) && !f2.frozen?)
  outs = []
  s.each_char { |ch| x = ch; x << "1"; outs << ch }
  ls = []
  v[3].each_line { |l| l << "!"; ls << l }
  us = []
  v[4].upto(v[5]) { |ch| ch << "."; us << ch }
  check("sb97c", outs == ["h1", "e1", "l1", "l1", "o1"] && ls == ["a\n!", "b\n!"] && us == ["a.", "b.", "c."])
  h1 = {s.dup => 1}
  h2 = {(v[4] + "x") => 2, ("he" + "llo") => 3}
  check("sb97d", h1.keys[0].equal?(h1.keys[0]) && h1.keys[0].equal?(h2.keys[1]) && !h1.keys[0].equal?(s))
end
# ===== SECTION 98: Struct, class-body locals, instance_methods(false) =====
# Struct.new builds a class with a reader and a writer per member, ==, to_a,
# members, [] and keyword_init; `class P < Struct.new(...)' and a Struct.new block
# add methods. A class-body local is visible to a constant of the same body, and
# Module#instance_methods(false) lists a class's own public and protected methods
# (sorted here: MRI answers its method-table order). /usr/bin/ruby 2.6 settles each.
S98S = Struct.new(:a, :b)
S98K = Struct.new(:x, :y, keyword_init: true)
class S98P < Struct.new(:x, :y)
  def sum; x + y; end
end
S98Q = Struct.new(:a) do
  def dbl; a * 2; end
end
class S98L
  r = 5
  QQ = r * 2
  def initialize; end
  def f; end
  attr_reader :g
  alias h f
  private
  def i; end
end
def s98
  v = [1, 2, 10, 3, 4]
  s = S98S.new(v[0], v[1])
  s.a = v[2]
  t = S98S.new(v[0])
  check("st98a", s.a == 10 && s.b == 2 && s.to_a == [10, 2] && t.b.nil? && S98S.members == [:a, :b])
  check("st98b", s == S98S.new(10, 2) && s != S98S.new(1, 2) && s.inspect == "#<struct S98S a=10, b=2>")
  s[1] = v[4]
  check("st98c", s[0] == 10 && s[:b] == 4 && s["a"] == 10 && s.members == [:a, :b] && s.to_h == {a: 10, b: 4})
  k = S98K.new(x: v[0], y: v[1])
  e = begin; S98S.new(1, 2, 3); nil; rescue ArgumentError => x; x.message; end
  check("st98d", k.x == 1 && k.y == 2 && e == "struct size differs")
  pt = S98P.new(v[3], v[4])
  check("st98e", pt.sum == 7 && pt.to_a == [3, 4] && S98Q.new(v[4]).dbl == 8 && pt.to_s == "#<struct S98P x=3, y=4>")
  check("st98f", S98L::QQ == 10 && S98L.instance_methods(false).sort == [:f, :g, :h])
end

# ===== SECTION 99: setter methods, Struct members, private :name, instance_methods =====
# `o.a = v' calls a def `a=' (also `self.a = v', a class's `def self.a=', and the
# read half of `o.a += v'); a Struct's members are not instance variables and its
# each / hash / dig / values_at answer as MRI's; `nil << x' is a NoMethodError;
# `private :name' hides a method; instance_methods without an argument adds the
# inherited ones. /usr/bin/ruby 2.6 settles each.
class S99A
  attr_reader :q
  def a=(v); @q = v * 2; end
  def a; @q + 1; end
  def set(v); self.a = v; end
  def self.cfg=(v); @@cfg = v + 100; end
  def self.cfg; @@cfg; end
end
class S99B < S99A
  def b1; end
end
S99S = Struct.new(:x, :y) do
  def bump; @x = 99; x; end
end
S99T = Struct.new(:x, :y)
module S99M; def m=(v); @m = v + 1; end; def m; @m; end; end
class S99W
  include S99M
  define_method(:w=) { |v| @w = v - 1 }
  attr_reader :w, :r
  def r=(e); @r = e.message + "!"; end
  def go(v); self.sec = v; @s; end
  private
  def sec=(v); @s = v * 7; end
end
class S99D
  def self.make(name, &b); define_method(name, &b); end
  attr_reader :dv
  def initialize; @dv = 7; end
end
class S99P
  def pub; 1; end
  def hid; 2; end
  private :hid
end
def s99
  v = [5, 3, 7, 4, 10, 20]
  o = S99A.new
  o.a = v[0]
  r = (o.a = v[1])
  check("se99a", o.q == 6 && r == 3)
  o.set(v[2])
  o.a += v[3]
  check("se99b", o.q == 38)
  S99A.cfg = v[0]
  check("se99c", S99A.cfg == 105)
  s = S99S.new(v[0], [v[4], {k: v[5]}])
  s.x += v[1]
  check("se99d", s.x == 8 && s.instance_variables == [] && s.bump == 8 && s.instance_variables == [:@x])
  d = s.dup
  d.x = v[3]
  check("se99e", s.x == 8 && d.x == 4)
  seen = []
  e = s.each { |m| seen << m }
  check("se99f", seen == [8, [10, {k: 20}]] && e.equal?(s))
  check("se99g", s.hash == S99S.new(8, [10, {k: 20}]).hash && s.hash != S99S.new(1, 2).hash && S99T.new(1, 2).hash != S99S.new(1, 2).hash)
  check("se99h", s.dig(:y, 1, :k) == 20 && s.dig(:zz).nil? && s.dig(0) == 8 && s.values_at(1, 0) == [[10, {k: 20}], 8])
  ve = begin; s.values_at(v[0]); nil; rescue IndexError => x; x.message; end
  check("se99i", ve == "offset 5 too large for struct(size:2)")
  n = nil
  ne = begin; n << v[0]; nil; rescue NoMethodError => x; x.message; end
  check("se99j", ne == "undefined method `<<' for nil:NilClass")
  pe = begin; S99P.new.hid; nil; rescue NoMethodError => x; x.class; end
  check("se99k", S99P.new.pub == 1 && pe == NoMethodError && S99P.instance_methods(false) == [:pub])
  im = S99B.instance_methods
  check("se99l", im.include?(:b1) && im.include?(:a=) && im.include?(:to_s) && (im - Object.instance_methods).sort == [:a, :a=, :b1, :q, :set])
  w = S99W.new
  w.m = v[0]
  z, w.w = v[1], v[4]
  begin; raise "boom"; rescue => w.r; end
  so = Object.new
  def so.z=(v); @z = v * 3; end
  def so.z; @z; end
  so.z = v[2]
  se = begin; w.sec = 1; nil; rescue NoMethodError => x; x.class; end
  check("se99m", w.m == 6 && w.w == 9 && z == 3 && w.r == "boom!" && so.z == 21 && w.go(v[3]) == 28 && se == NoMethodError)
  S99D.make(:mk) { |k| [self.class, dv + k] }
  blk = proc { self.class }
  S99D.send(:define_method, :pk, &blk)
  so.define_singleton_method(:me, &proc { equal?(so) })
  check("se99n", S99D.new.mk(v[1]) == [S99D, 10] && S99D.new.pk == S99D && blk.call == Object && so.me)
end

# ===== SECTION 100: dig, values_at ranges, unrun locals, singleton attrs, setter aliases =====
# Array#dig / Hash#dig walk a path and stop at nil; Array#values_at takes a Range
# and pads with nil past the end. A local whose assignment never ran (a raise
# skipped it, a branch did not take it) is nil. attr_accessor inside
# `class << self' defines accessors on the class; `alias b= a=' and
# `alias_method :b=, :a=' alias setters; a class body calls its own and inherited
# class methods bare; a Float has no `<<'; define_method(&lambda) binds self to
# the receiver; `o.x = v' with no setter `x=' is a NoMethodError.
# /usr/bin/ruby 2.6 settles each.
class S100C
  class << self
    attr_accessor :cnt
    attr_writer :wo
    def bump; self.cnt = (cnt || 0) + 1; end
  end
end
class S100A
  attr_accessor :a
  alias_method :b=, :a=
  alias c= a=
  alias :d= :a=
  def e=(v); @a = v * 2; end
  alias f= e=
end
class S100W
  @made = []
  def self.make(n, &blk); @made << (blk ? blk.call(n) : n); end
  def self.made; @made; end
  make(:w) { |x| [x] }
  make :v
end
class S100V < S100W
  @made = []
  make(:sub)
end
class S100R
  begin
    raise "x"
    rv = 1
  rescue RuntimeError
  end
  RV = [rv]
  rn = nil
  RN = rn
end
class S100P; attr_writer :pw; end
class S100Q; end
class S100D
  define_method(:y, &lambda { self.class })
  define_method(:z, &->(q) { [self.class, q] })
  def initialize; @x = 1; end
  def x; @x; end
end
def s100
  v = [1, 0, 5, 2, 3]
  a = [v[0], [v[3], {k: v[4]}]]
  h = {a: {b: [v[2], v[4]]}}
  check("dg100a", a.dig(1, 1, :k) == 3 && a.dig(1, 0) == 2 && a.dig(5, 1).nil? &&
        h.dig(:a, :b, 1) == 3 && h.dig(:x, :b).nil?)
  x = [v[3], v[0]]
  re = begin; x.values_at(-5..v[0]); nil; rescue RangeError => e; e.message; end
  check("va100b", x.values_at(v[1]..v[2]) == [2, 1, nil, nil, nil, nil] &&
        x.values_at(v[0], v[1]..v[4], v[1]) == [1, 2, 1, nil, nil, 2] && x.values_at(v[0]..-1) == [1] &&
        x.values_at(v[3]...v[4]) == [nil] && x.values_at(v[0]..v[1]) == [] && re == "-5..1 out of range")
  begin
    raise "x"
    ve = 1
  rescue RuntimeError
  end
  w = v[3] if v[1] > 0
  seen = []
  v.first(3).each_with_index { |n, i| t = n if i == 0; seen << t }
  check("lv100c", ve.nil? && w.nil? && seen == [1, nil, nil] && S100R::RV == [nil])
  S100C.cnt = v[0]
  S100C.cnt += v[3]
  S100C.bump
  S100C.wo = v[2]
  check("sa100d", S100C.cnt == 4 && S100C.instance_variable_get(:@wo) == 5)
  o = S100A.new
  o.b = v[2]
  r1 = o.a
  o.c = v[3]
  r2 = o.a
  o.d = v[4]
  r3 = o.a
  o.f = v[3]
  check("al100e", [r1, r2, r3, o.a] == [5, 2, 3, 4])
  check("cb100f", S100W.made == [[:w], :v] && S100V.made == [:sub])
  f = [1.5, 2.0]
  fe = begin; f[0] << v[0]; nil; rescue NoMethodError => e; e.class; end
  ge = begin; f[1] >> v[0]; nil; rescue NoMethodError => e; e.class; end
  check("fl100g", fe == NoMethodError && ge == NoMethodError && (v[4] << v[3]) == 12)
  d = S100D.new
  check("dm100h", d.y == S100D)
  we = begin; d.x = v[2]; nil; rescue NoMethodError => e; e.class; end
  check("ws100i", we == NoMethodError && d.x == 1)
  check("dm100j", d.z(v[2]) == [S100D, 5])
  kk = kk
  check("lv100k", kk.nil?)
  check("cn100l", S100R::RN.nil?)
  q = S100Q.new
  qe = begin; q.pw = v[2]; nil; rescue NoMethodError => e; e.name; end
  check("aw100m", qe == :pw= && q.instance_variables == [])
end
# ===== SECTION 101: Method#arity, inherited private setters, locals over methods, nil.x =, missing keywords =====
# `obj.method(:m).arity' counts like a lambda's; a subclass may call an inherited
# private setter through self; `A.new.x = v' constructs before it writes; a local
# assigned so far hides a method of its name and `k = k' reads nil; `nil.x = v'
# calls NilClass#x=; an absent required keyword is ArgumentError in a def, a proc
# and a define_method body. /usr/bin/ruby 2.6 settles each.
class S101D
  def y(a); a; end
  def z(a, b = 1); end
  define_method(:q) { |a, b| a }
  define_method(:k) { |a:| a }
  define_method(:j) { |a:, b: 2| [a, b] }
  def m; 7; end
  def t; m = m; m; end
  def u(v); m = v; m; end
end
class S101A; attr_accessor :x; private :x=; end
class S101B < S101A; def w(v); self.x = v; x; end; end
class NilClass; def s101=(v); $s101 = v; end; end
def s101_k; 9; end
s101_k = s101_k
S101K = [s101_k]
def s101_g(a:, b:, c: 3); [a, b, c]; end
def s101
  v = [4, 1, 2]
  d = S101D.new
  check("ar101a", d.method(:y).arity == 1 && d.method(:z).arity == -2 && d.method(:q).arity == 2)
  check("ps101b", S101B.new.w(v[0]) == 4)
  pe = begin; S101A.new.x = v[1]; nil; rescue NoMethodError => e; e.name; end
  check("ps101c", pe == :x=)
  check("lv101d", d.t.nil? && d.u(v[2]) == 2 && S101K == [nil])
  nil.s101 = v[0]
  check("ns101e", $s101 == 4)
  m1 = begin; s101_g(b: v[1]); rescue ArgumentError => e; e.message; end
  m2 = begin; s101_g; rescue ArgumentError => e; e.message; end
  pr = proc { |a:| a }
  m3 = begin; pr.call; rescue ArgumentError => e; e.message; end
  m4 = begin; d.k; rescue ArgumentError => e; e.message; end
  m5 = begin; d.j(b: v[2]); rescue ArgumentError => e; e.message; end
  check("kw101f", m1 == "missing keyword: a" && m2 == "missing keywords: a, b" && m3 == "missing keyword: a" &&
        m4 == "missing keyword: a" && m5 == "missing keyword: a")
  check("kw101g", d.k(a: v[0]) == 4 && d.j(a: v[1]) == [1, 2] && pr.call(a: v[2]) == 2 && s101_g(a: v[1], b: v[2]) == [1, 2, 3])
end

# ===== SECTION 102: String#split, #index / #rindex, #[] with a pattern, sub templates, #succ =====
# split is MRI's: no argument or " " is awk mode, a limit caps the fields, trailing
# empty fields go unless the limit is negative, and a Regexp's groups are kept.
# index / rindex take an offset and a Regexp; s[str] / s[re, n] read and write a
# match; a sub template knows \& \` \' \k<n>, a Hash maps each match; a pattern with
# a named group captures only the named ones; succ skips non-alphanumerics.
# /usr/bin/ruby 2.6 settles each.
def s102
  w = ["a,b,,c,,", " a  b ", "a1b2", "", "abc", ",", " "]
  check("sp102a", w[0].split(w[5]) == ["a", "b", "", "c"] && w[0].split(w[5], 2) == ["a", "b,,c,,"] &&
        w[0].split(w[5], -1) == ["a", "b", "", "c", "", ""])
  check("sp102b", w[1].split(w[6]) == ["a", "b"] && w[1].split == ["a", "b"] && w[1].split(w[6], 2) == ["a", "b "])
  check("sp102c", w[2].split(/(\d)/) == ["a", "1", "b", "2"] && w[3].split(w[5]) == [] && w[4].split(//) == ["a", "b", "c"] &&
        w[4].split("", 2) == ["a", "bc"])
  h = ["hello", "hello world", "ell", "z", "o", "l"]
  check("ix102d", h[1].index(h[4], 5) == 7 && h[1].rindex(h[4], 5) == 4 && h[1].index(h[4], 20).nil? &&
        h[1].index("", 11) == 11 && h[1].index(h[4], -4) == 7)
  check("ix102e", h[0].index(/l/) == 2 && h[0].rindex(/l/) == 3 && h[0].index(/l/, 3) == 3 &&
        h[0].index(h[5], 3) == 3 && h[0].rindex(h[5], 2) == 2 && h[0].rindex(/l/, 2) == 2)
  check("sb102f", h[0][/l+/] == "ll" && h[0][h[2]] == "ell" && h[0][h[3]].nil? && h[0][/(h)(e)/, 2] == "e" &&
        h[0].slice(/l+/) == "ll" && h[0].slice(h[2]) == "ell")
  t = h[1].dup
  t["world"] = "ruby"
  t[/l+/] = "L"
  e1 = begin; t["zz"] = "q"; rescue IndexError => e; e.message; end
  check("sb102g", t == "heLo ruby" && e1 == "string not matched")
  check("su102h", "abc".sub(/b/, "[\\&]") == "a[b]c" && "abc".sub(/b/, "\\`|\\'") == "aa|cc" &&
        "cat hat".gsub(/[ch]/, "c" => "C", "h" => "H") == "Cat Hat" && "abc".sub(/(?<x>b)/, "{\\k<x>}") == "a{b}c")
  md = "ab".match(/(?<first>a)(b)/)
  check("nm102i", md[2].nil? && md.size == 2 && md.captures == ["a"])
  u = ["1.9", "a-9", "a z", "<<koala>>", "Zz", "", "9", "***", "zz"]
  check("sc102j", u.map { |s| s.succ } == ["2.0", "a-10", "b a", "<<koalb>>", "AAa", "", "10", "**+", "aaa"])
end

# ===== SECTION 103: def without parentheses, literals on separate lines, assignment order, ||= / &&= =====
# `def m a, b' takes a parameter list without parentheses; a string literal at the
# start of a line is a new statement, not glued onto the previous line's literal;
# an index assignment evaluates the receiver and the index before the value, and
# an operator assignment reads the old value before the value; ||= and &&= only
# evaluate and write when the old value is falsy / truthy, and an unassigned local
# reads as nil. /usr/bin/ruby 2.6 settles each.
def s103_twice n
  n * 2
end
def s103_add a, b = 3
  a + b
end
def s103_lit
  r = "first"
  "second".upcase
end
def s103_t(log, x)
  log << x
  x
end
def s103_memo
  z ||= 4
  z &&= z + 1
  z
end
def s103
  v = [21, 1, 0]
  check("df103a", s103_twice(v[0]) == 42 && s103_add(v[1]) == 4 && s103_add(v[1], v[1]) == 2)
  check("sl103b", s103_lit == "SECOND")
  arr = [v[2], v[2]]
  i = v[2]
  arr[i += 1] = (i += 10)
  log = []
  hh = {}
  hh[s103_t(log, :k)] = s103_t(log, :v)
  c = {k: v[1]}
  c[s103_t(log, :k)] += s103_t(log, 2)
  check("as103c", arr == [0, 11] && log == [:k, :v, :k, 2] && c == {k: 3})
  a = nil
  a ||= s103_t(log, 5)
  a ||= s103_t(log, 6)
  b = v[1]
  b &&= s103_t(log, 7)
  h = {}
  h[:x] ||= []
  h[:x] << 1
  h[:x] ||= s103_t(log, 9)
  check("or103d", a == 5 && b == 7 && h == {x: [1]} && log == [:k, :v, :k, 2, 5, 7] && s103_memo == 5)
end

# ===== SECTION 104: return through ensure in a block, a proc's return, a loop's break value =====
# `return' in a begin body inside a block leaves the method after the ensure ran;
# `return' in a proc (proc { } / Proc.new) leaves the method the proc was written
# in, while a lambda's only leaves the lambda; `break v' makes a while / until loop
# answer v, which is nil when the loop ends by its condition. /usr/bin/ruby 2.6
# settles each.
def s104_m4(log)
  [1, 2].each { |x| begin; return x; ensure; log << x; end }
  :fell_through
end
def s104_m5
  pr = Proc.new { return 7 }
  pr.call
  :unreached
end
def s104_m6(v)
  pr = proc { |q| return q * 2 }
  [v].each { |q| pr.call(q) }
  :unreached
end
def s104_m7
  la = lambda { return 3 }
  la.call + 1
end
def s104
  log = []
  check("rt104a", s104_m4(log) == 1 && log == [1])
  check("pr104b", s104_m5 == 7 && s104_m6(3) == 6 && s104_m7 == 4)
  lim = [3, 5]
  w = 0
  r = while w < 10
    w += 1
    break w * 2 if w == lim[0]
  end
  r2 = while w < lim[1]
    w += 1
  end
  u = 0
  r3 = until u > 4
    u += 1
    break :done if u == 2
  end
  check("wb104c", r == 6 && r2.nil? && r3 == :done)
end

# ===== SECTION 105: Array#insert / fill / slice / uniq / max_by / flatten!, Hash#each mutation, !~, round(half:), sum, fdiv =====
# insert counts a negative position from after the last element and pads with nil;
# fill takes a start, a length or a range; slice(start, len) is nil out of range;
# uniq takes a block; max_by answers the first of ties and takes n; flatten!
# answers nil when nothing changed. Hash#each skips a deleted entry and refuses a
# new key. !~ is the negated match; round's half: is up, even or down; Array#sum
# compensates (Kahan-Babuska) once a Float is in; Integer#fdiv by a Bignum is
# exact. /usr/bin/ruby 2.6 settles each.
def s105
  v = [1, 2, 3, 4]
  ix = [-1, -2, 3, -4]
  check("in105a", [1, 2].insert(ix[0], :b) == [1, 2, :b] && [1, 2].insert(ix[1], :z) == [1, :z, 2] &&
        [1, 2].insert(ix[2], 9) == [1, 2, nil, 9])
  e1 = begin; [1, 2].insert(ix[3], :w); rescue IndexError => e; e.message; end
  check("in105b", e1 == "index -4 too small for array; minimum: -3")
  check("fl105c", v.dup.fill(9, 0, 2) == [9, 9, 3, 4] && v.dup.fill(9, 3, 5) == [1, 2, 3, 9, 9, 9, 9, 9] &&
        v.dup.fill(9, -2, 1) == [1, 2, 9, 4] && v.dup.fill(9, 1..2) == [1, 9, 9, 4] && v.dup.fill(2) { |i| i * 10 } == [1, 2, 20, 30])
  b = [1, 2, 3, 4, 5]
  check("sl105d", b.slice(6, 0).nil? && b.slice(-6, 1).nil? && b.slice(2, -1).nil? && b.slice(5, 1) == [])
  w = %w[a bb cc d]
  check("ub105e", (1..6).to_a.uniq { |x| x % 3 } == [1, 2, 3] && w.max_by { |s| s.size } == "bb" &&
        %w[a bb ccc d].max_by(2) { |s| s.size } == ["ccc", "bb"] && [1].flatten!.nil? && [[1]].flatten! == [1])
  h = {a: 1, b: 2, c: 3}
  seen = []
  h.each { |k, x| seen << k; h.delete(:b) if k == :a }
  h2 = {a: 1}
  e2 = begin; h2.each { |k, x| h2[:z] = 2 }; rescue RuntimeError => e; e.message; end
  check("he105f", seen == [:a, :c] && h == {a: 1, c: 3} && e2 == "can't add a new key into hash during iteration" && h2 == {a: 1})
  s = ["abc", "xyz"]
  check("nm105g", (s[0] !~ /z/) == true && (s[1] !~ /z/) == false)
  f = [2.5, -2.5, 3.5, 2.665, 0.125]
  check("rh105h", f[0].round(half: :even) == 2 && f[0].round(half: :down) == 2 && f[1].round(half: :even) == -2 &&
        f[2].round(half: :down) == 3 && f[3].round(2, half: :even) == 2.66 && f[4].round(2, half: :down) == 0.12 &&
        25.round(-1, half: :even) == 20 && 35.round(-1, half: :down) == 30 && f[0].round == 3)
  e3 = begin; f[0].round(half: :foo); rescue ArgumentError => e; e.message; end
  check("rh105i", e3 == "invalid rounding mode: foo")
  check("su105j", [0.1, 0.2, 0.3].sum == 0.6 && [1e100, 1.0, -1e100].sum == 1.0 && [3, 0.1, 0.2].sum == 3.3)
  n = [10**30, 10**29]
  check("fd105k", n[0].fdiv(n[1]) == 10.0)
end

# ===== SECTION 106: Comparable, <=>, === with a Proc, Range bounds and Float steps, eql? keys =====
# Comparable gives == (<=> answers 0) and clamp, and a nil <=> makes < raise;
# Object#<=> is 0 for equal values; a Proc in `when' is called; NaN is in no range;
# a range's min / max with a numeric end are its bounds; a Float step walks doubles;
# a Hash keys an object by its class's eql? / hash. /usr/bin/ruby 2.6 settles each.
class S106V
  include Comparable
  attr_reader :n
  def initialize(n); @n = n; end
  def <=>(o)
    return nil unless o.is_a?(S106V)
    n <=> o.n
  end
end
class S106K
  attr_reader :v
  def initialize(v); @v = v; end
  def hash; v.hash; end
  def eql?(o); o.is_a?(S106K) && v == o.v; end
end
def s106
  v = [1, 2, 5, 3]
  e1 = begin; S106V.new(v[0]) < v[2]; rescue ArgumentError => e; e.message; end
  check("cm106a", S106V.new(v[0]) == S106V.new(v[0]) && S106V.new(v[0]) != S106V.new(v[1]) &&
        S106V.new(v[2]).clamp(S106V.new(v[0]), S106V.new(v[3])).n == 3 && e1 == "comparison of S106V with 5 failed")
  check("sp106b", [nil <=> nil, true <=> true, false <=> true] == [0, 0, nil])
  f = ->(x) { x > 2 }
  r = case 2.5
      when f then "big"
      else "small"
      end
  nan = Float::NAN
  check("cw106c", r == "big" && !((0..9) === nan) && !(0..9).include?(nan))
  check("rg106d", (1.0..2.0).min == 1.0 && (1.0..2.0).max.is_a?(Float) && (1..1.5).max == 1.5 && (1...4).max == 3 &&
        (1.0..2.0).step(0.5).to_a == [1.0, 1.5, 2.0] && (1.0...2.0).step(0.5).to_a == [1.0, 1.5])
  hh = {S106K.new(v[0]) => "one"}
  check("ek106e", hh[S106K.new(v[0])] == "one" && [S106K.new(v[0]), S106K.new(v[0])].uniq.size == 1)
end

# ===== SECTION 107: rescuable index and size errors, first(n) of an endless range, assorted String / Array corners =====
# A write before the start of an Array, each_slice(0) and a negative repetition are
# rescuable errors with MRI's wording; first(n) of an infinite or endless range
# takes n elements; strip removes NULs and all ASCII whitespace; a Symbol proc
# sends operators too; chomp("") removes every trailing newline; a String range
# whose begin is past its end is empty. /usr/bin/ruby 2.6 settles each.
def s107
  b = [1, 2, 3, 4, 5, 6]
  i = [-10, 0, -1, 3]
  e1 = begin; b[i[0]] = 1; rescue IndexError => e; e.message; end
  e2 = begin; b.each_slice(i[1]) { |x| x }; rescue ArgumentError => e; e.message; end
  e3 = begin; b.each_cons(i[1]) { |x| x }; rescue ArgumentError => e; e.message; end
  check("ie107a", e1 == "index -10 too small for array; minimum: -6" && e2 == "invalid slice size" && e3 == "invalid size")
  check("fr107b", (1..Float::INFINITY).first(i[3]) == [1, 2, 3] && (1..).first(2) == [1, 2])
  e4 = begin; "x" * i[2]; rescue ArgumentError => e; e.message; end
  e5 = begin; [0] * i[2]; rescue ArgumentError => e; e.message; end
  check("ng107c", e4 == "negative argument" && e5 == "negative argument")
  s = ["\t\n hi \r\n\0", "abc\n\r\n\n", "y", "ab"]
  check("st107d", s[0].strip == "hi" && [1, 2].inject(&:+) == 3 && s[1].chomp("") == "abc" && (s[2]..s[3]).to_a == [])
end

# ===== SECTION 108: Array#at / transpose / slice! / bsearch / chunk_while, Hash#compact, each_line(sep), Numeric#step =====
# The Array and Hash methods MRI 2.6 has and the engines lacked; each_line and
# lines take a separator; Numeric#step walks Integers, or doubles when one of the
# three is a Float; upto without a limit is an ArgumentError. /usr/bin/ruby 2.6
# settles each.
def s108
  v = [1, 2, 3, 4]
  e1 = begin; [[1, 2], [3]].transpose; rescue IndexError => e; e.message; end
  check("ar108a", v.at(-1) == 4 && v.at(9).nil? && [[1, 2], [3, 4]].transpose == [[1, 3], [2, 4]] &&
        e1 == "element size differs (1 should be 2)")
  x = v.dup
  y = v.dup
  check("ar108b", x.slice!(1, 2) == [2, 3] && x == [1, 4] && y.slice!(-1) == 4 && y == [1, 2, 3] && y.slice!(9).nil?)
  check("ar108c", [1, 3, 5, 7].bsearch { |n| n >= 4 } == 5 && [1, 3, 5, 7].bsearch { |n| 5 <=> n } == 5 &&
        [1, 2, 4, 9, 10, 11].chunk_while { |a, b| b == a + 1 }.to_a == [[1, 2], [4], [9, 10, 11]] &&
        [1, 2, 4].slice_when { |a, b| b != a + 1 }.to_a == [[1, 2], [4]])
  check("hc108d", {a: nil, b: 1, c: false}.compact == {b: 1, c: false})
  check("el108e", "a--b--c".each_line("--").to_a == ["a--", "b--", "c"] && "x|y|".lines("|") == ["x|", "y|"])
  e2 = begin; 5.upto; rescue ArgumentError => e; e.message; end
  check("st108f", 3.step(10, 3).to_a == [3, 6, 9] && 10.step(1, -4).to_a == [10, 6, 2] && 1.step(2, 0.5).to_a == [1.0, 1.5, 2.0] &&
        1.0.step(2.0, 0.5).to_a == [1.0, 1.5, 2.0] && e2 == "wrong number of arguments (given 0, expected 1)")
end

# ===== SECTION 109: an astral character in ord, index and rindex =====
# A character outside the BMP is ONE character: its ord is its code point, and
# index / rindex count it once. /usr/bin/ruby 2.6 settles each.
def s109
  s = ["a\u{1F600}b", "\u{1F600}", "b"]
  check("as109a", s[1].ord == 128512 && s[0].index(s[2]) == 2 && s[0].rindex(s[1]) == 1 && s[0].length == 3)
end

# ===== SECTION 110: next / break out of a begin inside a block; $0 =====
# `next v' inside a begin/ensure or rescue in a block ends that block run with v,
# `break v' ends the iterating call; $0 and $PROGRAM_NAME are the script path, so
# the `if __FILE__ == $0' guard runs. /usr/bin/ruby 2.6 settles each.
def s110
  a = [1, 2, 3].map { |x| begin; next x * 10 if x == 2; x; ensure; nil; end }
  b = [1, 2, 3].map { |x| begin; raise "q" if x == 3; x; rescue; next -1; end }
  c = [1, 2, 3].map { |x| begin; begin; next x * 10 if x == 2; ensure; nil; end; x; ensure; nil; end }
  check("bn110a", a == [1, 20, 3] && b == [1, 2, -1] && c == [1, 20, 3])
  log = []
  d = [1, 2, 3].each { |x| begin; break x * 7 if x == 2; ensure; log << x; end }
  check("bb110b", d == 14 && log == [1, 2])
  check("pn110c", __FILE__ == $0 && $PROGRAM_NAME == $0)
end

# ===== SECTION 111: String#[], center and reverse count characters =====
# An astral character is one character to s[i], s[i, n], s[range], slice,
# center / ljust / rjust and reverse. /usr/bin/ruby 2.6 settles each.
def s111
  s = ["a\u{1F600}bc", "\u{1F600}", "日本\u{1F600}x"]
  t = s[0]
  check("ix111a", t[1] == s[1] && t[1, 2] == s[1] + "b" && t[2..3] == "bc" && t[-3] == s[1] && t[1..] == s[1] + "bc" &&
        t.slice(1) == s[1] && t[4].nil? && t[4, 1] == "" && t[5, 1].nil?)
  u = s[2]
  check("ix111b", u[0] == "日" && u[-1] == "x" && u[1, 10] == "本" + s[1] + "x" && u[1...-1] == "本" + s[1] && u[-5].nil?)
  check("pd111c", t.center(9, "*") == "**" + t + "***" && t.ljust(6, ".") == t + ".." && t.rjust(6) == "  " + t &&
        u.ljust(7, s[1]) == u + s[1] * 3 && "ab".center(7, "xy") == "xyabxyx")
  check("rv111d", t.reverse == "cb" + s[1] + "a" && u.reverse == "x" + s[1] + "本日")
end

# ===== SECTION 112: Integer.sqrt, Regexp.last_match, Exception#cause, pp, bare p and new =====
# Integer.sqrt is exact past 2**53; Regexp.last_match is $~; a raise inside a
# rescue records the handled exception as the new one's cause; pp answers its
# argument; a bare `p' is nil; a bare `new' in a class method builds an instance.
# /usr/bin/ruby 2.6 settles each.
class S112
  attr_reader :a
  def initialize(a = 0); @a = a; end
  def self.make; new; end
  def self.make2(a); new(a); end
end
def s112
  e1 = begin; Integer.sqrt(-1); rescue Math::DomainError => e; e.message; end
  check("is112a", Integer.sqrt(24) == 4 && Integer.sqrt(25) == 5 && Integer.sqrt(10**41) == 316227766016837933199 &&
        e1 == "Numerical argument is out of domain - \"isqrt\"")
  "abc" =~ /b(c)/
  m = [Regexp.last_match[0], Regexp.last_match(1)]
  "x" =~ /y/
  check("lm112b", m == ["bc", "c"] && Regexp.last_match.nil?)
  c1 = begin; begin; raise ArgumentError, "a"; rescue => e; raise TypeError, "b"; end; rescue => e2; e2.cause; end
  c2 = begin; raise "x"; rescue => e; e.cause; end
  c3 = begin; begin; raise "a"; rescue => e; raise e; end; rescue => e2; e2.cause; end
  check("ca112c", c1.class == ArgumentError && c1.message == "a" && c2.nil? && c3.nil?)
  x = p
  check("pp112d", x.nil? && S112.make.class == S112 && S112.make2(4).a == 4)
end

# ===== SECTION 113: Proc#>> / #<< / #curry; Rational#round(half:) =====
# >> runs the receiver first, << last, and the composition is a lambda when the
# receiver is; curry collects arguments up to the arity. Rational#round honours
# half: :even and :down on an exact tie. /usr/bin/ruby 2.6 settles each.
def s113
  inc = proc { |x| x + 1 }
  dbl = proc { |x| x * 2 }
  check("pc113a", (inc >> dbl).call(3) == 8 && (inc << dbl).call(3) == 7 && !(inc >> dbl).lambda? &&
        (lambda { |x| x } >> dbl).lambda?)
  add3 = proc { |a, b, c| a + b + c }
  add2 = lambda { |a, b| a + b }
  check("cu113b", add3.curry[1][2][3] == 6 && add3.curry[1, 2][3] == 6 && add2.curry[1][2] == 3 &&
        add2.curry.lambda? && !add3.curry.lambda? && proc { |a, b| [a, b] }.curry(3)[1][2][3] == [1, 2])
  r = [Rational(5, 2), Rational(-5, 2), Rational(7, 2), Rational(-7, 2)]
  check("rr113c", r.map { |x| x.round(half: :even) } == [2, -2, 4, -4] && r.map { |x| x.round(half: :down) } == [2, -2, 3, -3] &&
        r.map { |x| x.round(half: :up) } == [3, -3, 4, -4] && Rational(2**70 + 1, 2).round(half: :even) == 2**69 &&
        Rational(5, 4).round(1, half: :even) == Rational(6, 5))
end

# ===== SECTION 114: loop and StopIteration; catch / throw =====
# loop ends at a StopIteration and answers its #result; Enumerator#next past the
# end raises one. throw unwinds to its catch past rescue clauses, running ensure;
# a throw no catch waits for is an UncaughtThrowError. /usr/bin/ruby 2.6 settles each.
def s114_thrower
  throw :t114, 3
end
def s114
  x = 0
  r1 = loop { x += 1; raise StopIteration if x == 3 }
  en = [1, 2].each
  got = []
  r2 = loop { got << en.next }
  m = begin; en.next; rescue StopIteration => e; [e.message, e.result]; end
  check("lp114a", x == 3 && r1.nil? && got == [1, 2] && r2 == [1, 2] && m == ["iteration reached an end", [1, 2]])
  r3 = catch(:done) { 10.times { |i| throw :done, i if i == 3 }; :never }
  log = []
  r4 = catch(:x) { begin; throw :x, 1; rescue Exception; log << :rescued; ensure; log << :ensure; end }
  r5 = catch(:o) { catch(:i) { throw :o, 9 }; 0 }
  check("ct114b", r3 == 3 && r4 == 1 && log == [:ensure] && r5 == 9 && catch { |t| throw t, 5 } == 5 &&
        catch(:a) { throw :a }.nil? && catch(:t114) { s114_thrower; 1 } == 3)
  u = begin; throw :nope, 4; rescue ArgumentError => e; [e.class, e.message, e.tag, e.value]; end
  check("ut114c", u == [UncaughtThrowError, "uncaught throw :nope", :nope, 4])
end

# ===== SECTION 115: method_missing and respond_to_missing? =====
# A call no method answers goes to method_missing with the name as a Symbol;
# its `super' reaches Object's, which raises NoMethodError for that name;
# respond_to? asks respond_to_missing?. /usr/bin/ruby 2.6 settles each.
class S115A
  def method_missing(n, *a)
    n == :total ? a.sum : super
  end
  def respond_to_missing?(n, priv = false)
    n == :total
  end
end
class S115B < S115A
  def method_missing(n, *a)
    n == :one ? 1 : super
  end
end
def s115
  e = begin; S115A.new.nope(1); rescue NoMethodError => x; [x.name, x.message[0, 23]]; end
  check("mm115a", S115A.new.total(1, 2, 3) == 6 && e == [:nope, "undefined method `nope'"])
  check("mm115b", S115B.new.one == 1 && S115B.new.total(4) == 4 && S115A.new.respond_to?(:total) &&
        !S115A.new.respond_to?(:nope))
end

# ===== SECTION 116: $stdout / STDOUT, $stderr / STDERR =====
# The standard streams are IO objects: $stdout is STDOUT, write answers the
# bytes written, << answers the stream. /usr/bin/ruby 2.6 settles each.
def s116
  check("io116a", $stdout.equal?(STDOUT) && $stderr.equal?(STDERR) && $stdout.class == IO && STDOUT.fileno == 1 &&
        $stderr.fileno == 2 && $stdout.write("") == 0 && ($stdout << "").equal?($stdout) && $stdout.print.nil? &&
        $stdout.flush.equal?($stdout))
end

# ===== SECTION 117: a negative Float to a fractional power; chr("UTF-8") =====
# A negative base to a non-integral Float power is a Complex (MRI's polar form,
# exactly imaginary at an odd half); Integer#chr("UTF-8") is the character of a
# code point, a RangeError outside Unicode or on a surrogate. /usr/bin/ruby 2.6
# settles each.
def s117
  b = [-8.0, -4.0, -8]
  c1 = b[0]**(1.0 / 3)
  check("cp117a", c1.class == Complex && c1 == Complex(1.0000000000000002, 1.7320508075688772) &&
        b[1]**0.5 == Complex(0.0, 2.0) && b[1]**1.5 == Complex(0.0, -8.0) && b[2]**(1.0 / 3) == c1 &&
        b[1]**2.0 == 16.0)
  e1 = begin; 0x110000.chr("UTF-8"); rescue RangeError => e; e.message; end
  e2 = begin; 0xD800.chr("UTF-8"); rescue RangeError => e; e.message; end
  check("ch117b", 128512.chr("UTF-8") == "\u{1F600}" && 233.chr("UTF-8") == "é" && 128512.chr("UTF-8").length == 1 &&
        e1 == "1114112 out of char range" && e2 == "invalid codepoint 0xD800 in UTF-8")
end

# ===== SECTION 118: Array#cycle without a block; Enumerator::Lazy =====
# A blockless cycle is an endless Enumerator; #lazy chains map / select /
# reject / take_while / drop_while / take one value at a time, so an endless
# range answers, and nothing past a take is evaluated. /usr/bin/ruby 2.6 settles
# each.
def s118
  e = [1, 2].cycle
  n = [e.next, e.next, e.next]
  check("cy118a", [1, 2].cycle.first(5) == [1, 2, 1, 2, 1] && [1, 2].cycle.take(3) == [1, 2, 1] && n == [1, 2, 1] &&
        [].cycle.first(2) == [])
  seen = []
  t = (1..Float::INFINITY).lazy.map { |x| seen << x; x }.take(2).to_a
  check("lz118b", (1..Float::INFINITY).lazy.map { |x| x * 2 }.select { |x| x % 3 == 0 }.first(3) == [6, 12, 18] &&
        (1..20).lazy.select(&:even?).map { |x| x * x }.first(2) == [4, 16] && t == [1, 2] && seen == [1, 2] &&
        (1..10).lazy.take_while { |x| x < 4 }.to_a == [1, 2, 3] && (1..6).lazy.drop_while { |x| x < 4 }.force == [4, 5, 6] &&
        [1, 2].cycle.lazy.map { |x| x * 10 }.first(3) == [10, 20, 10])
  check("lz118c", (1..3).lazy.map { |x| x }.inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: 1..3>:map>" &&
        [1, 2].cycle.inspect == "#<Enumerator: [1, 2]:cycle>")
end

# ===== SECTION 119: one-line loops; Enumerator#next results; zsuper; superclass; private Kernel =====
# A `while` after `x = 1;` on one line is a loop, not a modifier, and `foo? end` a
# predicate call, not a ternary; StopIteration#result of an external map / select
# is the method's answer with nil blocks; a bare super passes keyword parameters as
# keywords (**rest first) and the current values; an unknown superclass is
# NameError; a private Kernel method with a receiver is "private method ...
# called"; a block left by a raise or a throw is not the next method's block;
# $PROGRAM_NAME is $0. /usr/bin/ruby 2.6 settles each.
class S119X
  def f(*a, k: 1) [a, k] end
  def g(a, **o) [a, o] end
  def h(a) a end
end
class S119Y < S119X
  def f(*r, k: 2) super end
  def g(a, k: 1, **r) super end
  def h(a) a = 9; super end
end
module S119M
  class C
    def v; 7; end
  end
end
class S119D < S119M::C; end
S119NE = [begin; class S119E < Qzxwvq; end; rescue NameError => e; [e.message, e.name]; end,
          begin; class S119F < Qzxwvq::X; end; rescue NameError => e; e.message; end,
          begin; class S119G < S119M::Qzxwvq; end; rescue NameError => e; e.message; end]
def s119_bg; block_given? end
def s119_m; yield; end
def s119_rs; begin; s119_m { raise "x" }; rescue; end; s119_bg; end
S119EN = [1].each
loop { S119EN.next }
catch(:s119) { throw :s119 }
def s119_after; block_given?; end
S119LK = s119_after
def s119
  i = 0; j = 1; while i < 3; i += 1; end
  k = 0; until k >= 2; k += 1; end
  check("ln119a", i == 3 && j == 1 && k == 2 && s119_bg == false && s119_bg { } == true)
  e1 = [1, 2, 3].map
  e2 = [1, 2, 3].select
  e3 = [1, 2].partition
  e4 = [1, 2].each
  check("en119b", loop { e1.next } == [nil, nil, nil] && loop { e2.next } == [] &&
        loop { e3.next } == [[], [1, 2]] && loop { e4.next } == [1, 2])
  check("zs119c", S119Y.new.f(1, 2) == [[1, 2], 2] && S119Y.new.f(1, k: 5) == [[1], 5] &&
        S119Y.new.g(1, z: 3) == [1, {z: 3, k: 1}] && S119Y.new.g(1, z: 3)[1].keys == [:z, :k] && S119Y.new.h(1) == 9)
  check("sc119d", S119NE == [["uninitialized constant Qzxwvq", :Qzxwvq], "uninitialized constant Qzxwvq",
                             "uninitialized constant S119M::Qzxwvq"] && S119D.new.v == 7)
  m1 = begin; $stdout.p 1; rescue NoMethodError => e; e.message; end
  m2 = begin; 5.puts; rescue NoMethodError => e; e.message; end
  check("pv119e", m1 == "private method `p' called for #<IO:<STDOUT>>" && m2 == "private method `puts' called for 5:Integer" &&
        $stderr.inspect == "#<IO:<STDERR>>")
  check("bk119g", S119LK == false && s119_rs == false)
  old = $0
  $0 = "s119"
  a = $PROGRAM_NAME
  $PROGRAM_NAME = old
  check("pn119f", a == "s119" && $0 == old)
end

# ===== SECTION 120: private def; reopen after a failed class; blockless enumerators; qualified names; constants =====
# `private def m' / `protected def m' make m private / protected, and calling it with
# a receiver says "private method `m' called for #<C...>"; a class statement whose
# superclass raised binds nothing, so the next statement of that name makes a new
# class; a blockless each_with_object / each_entry / find_index is an Enumerator; a
# class or module written in another one's body is named by the path (M::X), and a
# nested body sees the enclosing body's constants; an unbound constant is NameError
# "uninitialized constant", named under the lexically enclosing class unless
# written `::X'. /usr/bin/ruby 2.6 settles each.
class S120A
  private def q; 1; end
  protected def s; 3; end
  public def r; q + s; end
  def t(o) o.s; end
end
S120RE = [begin; class S120E < Qzxwvq1; end; rescue NameError => e; e.message; end,
          begin; class S120E < Qzxwvq2; end; rescue NameError => e; e.message; end]
class S120E; def v; 5; end; end
module S120M
  class X; end
  class Y < X; end
  module N
    class Z < X; end
  end
end
class S120C
  def f; Qzxwvq; end
  def g; ::Qzxwvq; end
  def h; S120M::Qzxwvq; end
  def self.k; Qzxwvq::X; end
  S120CB = begin; Qzxwvq; rescue NameError => e; e.message; end
end
class S120D < S120C; end
def s120
  m1 = begin; S120A.new.q; rescue NoMethodError => e; e.message; end
  m2 = begin; S120A.new.s; rescue NoMethodError => e; e.message; end
  check("pv120a", S120A.new.r == 4 && S120A.new.t(S120A.new) == 3 &&
        m1.start_with?("private method `q' called for #<S120A") && m2.start_with?("protected method `s' called for #<S120A") &&
        !S120A.new.respond_to?(:q) && S120A.new.respond_to?(:q, true))
  check("rc120b", S120RE == ["uninitialized constant Qzxwvq1", "uninitialized constant Qzxwvq2"] && S120E.new.v == 5 &&
        S120E.superclass == Object)
  e = [1, 2, 3].each_with_object([])
  n = [1, 2].each_with_object([])
  n.next; n.next
  nr = begin; n.next; rescue StopIteration => x; x.result; end
  check("ew120c", e.class == Enumerator && e.inspect == "#<Enumerator: [1, 2, 3]:each_with_object([])>" &&
        e.to_a == [[1, []], [2, []], [3, []]] && e.each { |v, acc| acc << v * 2 } == [2, 4, 6] && nr == [] &&
        {a: 1}.each_with_object([]).to_a == [[[:a, 1], []]] && (1..2).each_with_object({}).each { |v, h| h[v] = 1 } == {1 => 1, 2 => 1})
  seen = []
  fi = [4, 5, 6].find_index
  check("ee120d", [1, 2].each_entry.class == Enumerator && [1, 2].each_entry.to_a == [1, 2] &&
        [1, 2].each_entry { |v| v } == [1, 2] && {a: 1}.each_entry { |v| v } == {a: 1} &&
        fi.inspect == "#<Enumerator: [4, 5, 6]:find_index>" && fi.next == 4 &&
        [4, 5, 6].find_index.each { |v| seen << v; v == 5 } == 1 && seen == [4, 5] &&
        [4, 5].find_index.each { |v| v == 9 }.nil? && (1..4).find_index.each { |v| v > 2 } == 2)
  check("qn120e", S120M::Y.superclass.inspect == "S120M::X" && S120M::X.name == "S120M::X" &&
        S120M::N::Z.superclass == S120M::X && S120M::N::Z.to_s == "S120M::N::Z" && S120M::N.name == "S120M::N" &&
        S120M::Y.ancestors.take(2).inspect == "[S120M::Y, S120M::X]" && S120M::Y.new.class.inspect == "S120M::Y")
  u = [-> { S120C.new.f }, -> { S120C.new.g }, -> { S120C.new.h }, -> { S120C.k }, -> { S120D.new.f }, -> { Qzxwvq }].map do |l|
    begin; l.call; rescue NameError => x; [x.message, x.name]; end
  end
  check("uc120f", u == [["uninitialized constant S120C::Qzxwvq", :Qzxwvq], ["uninitialized constant Qzxwvq", :Qzxwvq],
                        ["uninitialized constant S120M::Qzxwvq", :Qzxwvq], ["uninitialized constant S120C::Qzxwvq", :Qzxwvq],
                        ["uninitialized constant S120C::Qzxwvq", :Qzxwvq], ["uninitialized constant Qzxwvq", :Qzxwvq]] &&
        S120C::S120CB == "uninitialized constant S120C::Qzxwvq")
end


# ===== SECTION 121: binary (ASCII-8BIT) Strings; Encoding =====
# Integer#chr past 127, String#b, force_encoding("BINARY") and Array#pack("C*")
# make a binary String: one byte per character, #inspect escapes a byte past
# ASCII as \xHH, #encoding is ASCII-8BIT; == across encodings holds only for
# ASCII-only text; + / << with a UTF-8 side keep an ASCII-only side's bytes and
# raise Encoding::CompatibilityError for two non-ASCII sides; unpack reads the
# bytes. /usr/bin/ruby 2.6 settles each.
def s121
  cs = [255, 200, 65]
  b = cs[0].chr
  check("bn121a", b.bytes == [255] && b.encoding.inspect == "#<Encoding:ASCII-8BIT>" && b.inspect == "\"\\xFF\"" &&
        b.size == 1 && b.bytesize == 1 && b.valid_encoding? && b.is_a?(String) && b.class == String &&
        cs[2].chr.encoding != Encoding::BINARY && (b * 2).inspect == "\"\\xFF\\xFF\"" && b.to_s.encoding == Encoding::BINARY)
  e = "é".b
  check("bn121b", e.bytes == [195, 169] && e.size == 2 && e[0].inspect == "\"\\xC3\"" && e[0, 2].size == 2 &&
        "é".b.force_encoding("UTF-8") == "é" && "é".b.force_encoding(Encoding::UTF_8).encoding.to_s == "UTF-8" &&
        "é".force_encoding("BINARY").size == 2 && e.reverse.bytes == [169, 195] && e.ascii_only? == false &&
        "\x7F".b.ascii_only? && e.getbyte(1) == 169)
  check("bn121c", "ab".b == "ab" && e != "é" && e == "é".b && {cs[0].chr => 1}[cs[0].chr] == 1 && e.hash == "é".b.hash &&
        e.eql?("é".b))
  x = "".b
  x << cs[0].chr
  x << 65
  r = []
  e.each_byte { |c| r << c }
  ce = begin; "é" + b; rescue EncodingError => err; [err.class, err.message]; end
  check("bn121d", x.inspect == "\"\\xFFA\"" && x.encoding == Encoding::BINARY && r == [195, 169] &&
        ("a" + b).inspect == "\"a\\xFF\"" && (b + "a").encoding == Encoding::BINARY && (e + "x".b).inspect == "\"\\xC3\\xA9x\"" &&
        ce == [Encoding::CompatibilityError, "incompatible character encodings: UTF-8 and ASCII-8BIT"])
  pk = cs.pack("C*")
  check("bn121e", pk.inspect == "\"\\xFF\\xC8A\"" && pk.encoding == Encoding::BINARY && [233].pack("U*") == "é" &&
        [233].pack("U*").encoding == Encoding::UTF_8 && pk.unpack("C*") == cs && "é".unpack("C*") == [195, 169] &&
        "é".unpack("U*") == [233] && e.unpack1("C") == 195 && [-1].pack("c").unpack("c") == [-1] && "A".unpack("C3") == [65, nil, nil])
  check("bn121f", "abc".encoding.inspect == "#<Encoding:UTF-8>" && "abc".encoding.to_s == "UTF-8" &&
        Encoding::BINARY.equal?(Encoding::ASCII_8BIT) && Encoding::BINARY.name == "ASCII-8BIT" &&
        Encoding::UTF_8.inspect == "#<Encoding:UTF-8>" && "abc".b.encoding.to_s == "ASCII-8BIT")
end

# ===== SECTION 122: Kernel#warn; warnings through $stderr's #write =====
# warn writes its messages as IO#puts would (an Array one line per element, nil
# an empty line, no newline doubled) in ONE call of $stderr's #write, nothing for
# no message or an empty Array, and answers nil. uplevel: 0 prefixes the calling
# line, 1 the line that called the running method or block (at the top level,
# the calling line again), and a trailing Hash of Symbol keys is the options.
# Hash#index's deprecation warning goes the same way. $stderr only takes an
# object with #write. /usr/bin/ruby 2.6 settles each.
class S122Sink
  attr_reader :got
  def initialize; @got = []; end
  def write(*s); @got.concat(s); 0; end
end
def s122up
  warn "old", uplevel: 1
end
def s122
  v = ["hi", ["x", ["y"]], nil, 2.5, :s, "nl\n"]
  k = S122Sink.new
  saved = $stderr
  $stderr = k
  r = warn(v[0])
  warn(v[1], v[2], v[3])
  warn
  warn([])
  warn(v[4], v[5])
  $stderr = saved
  check("wn122a", r.nil? && k.got == ["hi\n", "x\ny\n\n2.5\n", "", "s\nnl\n"])
  k = S122Sink.new
  $stderr = k
  ln = __LINE__; warn "now", uplevel: 0
  s122up
  s122up; l2 = __LINE__
  [1].each { |i| warn "blk", uplevel: 1 }; l3 = __LINE__
  h = {1 => 2}
  h.index(2); l4 = __LINE__
  warn "a", "b", uplevel: 0.9
  e = []
  begin; warn("x", uplevel: -1); rescue ArgumentError => x; e << x.message; end
  begin; warn("x", foo: 1); rescue ArgumentError => x; e << x.message; end
  begin; warn("x", uplevel: :q); rescue TypeError => x; e << x.message; end
  warn("y", {"k" => 1})
  $stderr = saved
  f = __FILE__
  check("wn122b", k.got[0] == "#{f}:#{ln}: warning: now\n" && k.got[1] == "#{f}:#{ln + 1}: warning: old\n" &&
        k.got[2] == "#{f}:#{l2}: warning: old\n" && k.got[3] == "#{f}:#{l3}: warning: blk\n" &&
        k.got[4] == "#{f}:#{l4}: warning: Hash#index is deprecated; use Hash#key\n" &&
        k.got[5] == "#{f}:#{l4 + 1}: warning: a\nb\n" && k.got[6] == "y\n{\"k\"=>1}\n" && k.got.size == 7)
  check("wn122c", e == ["negative level (-1)", "unknown keyword: foo", "no implicit conversion of Symbol into Integer"])
  m = []
  begin; $stderr = 5; rescue TypeError => x; m << x.message; end
  begin; $stdout = Object.new; rescue TypeError => x; m << x.message; end
  check("wn122d", m == ["$stderr must have write method, Integer given", "$stdout must have write method, Object given"] &&
        $stderr.equal?(STDERR) && $stdout.equal?(STDOUT))
end

# ===== SECTION 123: an unbounded Range from a big Integer =====
# (2**70..) and (2**70..Float::INFINITY) walk exact Integers one at a time:
# each, for, first(n), lazy, step, each_slice, Enumerator#next, and #size is
# Infinity. #last of an endless range raises RangeError. /usr/bin/ruby 2.6
# settles each.
def s123
  b = [2**70][0]
  w = []
  (b..).each { |x| w << x; break if w.size == 2 }
  f = []
  for x in (b..Float::INFINITY)
    f << x
    break if f.size == 2
  end
  check("bg123a", w == [b, b + 1] && f == w && (b..).first(2) == w && (b..Float::INFINITY).first(2) == w &&
        (b..).size == Float::INFINITY && (b..Float::INFINITY).size == Float::INFINITY)
  e = (b..).each
  check("bg123b", (b..).lazy.map { |x| x * 2 }.first(2) == [2 * b, 2 * b + 2] && (b..).step(3).first(2) == [b, b + 3] &&
        (b..).each_slice(2).first(2) == [[b, b + 1], [b + 2, b + 3]] && e.next == b && e.next == b + 1 &&
        (b..).find { |x| x % 7 == 0 } == b + 5 && (b..).each_with_index.first(2) == [[b, 0], [b + 1, 1]])
  g = []
  for x in (1..)
    g << x
    break if g.size == 2
  end
  m = []
  [(b..), (1..)].each { |r| begin; r.last; rescue RangeError => x; m << x.message; end }
  begin; (b..).last(2); rescue RangeError => x; m << x.message; end
  check("bg123c", g == [1, 2] && m == ["cannot get the last element of endless range"] * 3 &&
        (1..Float::INFINITY).last == Float::INFINITY)
end

# ===== SECTION 124: endless ranges, big Integer walks, top-level directives =====
# take(n) of a range without an end takes like first(n), min(n) is first(n), and a
# negative n raises each method's own ArgumentError. Integer#step / #upto /
# #downto / #times and Range#step walk exact Integers past 2**53. `for` walks an
# endless String range. At the top level main has no #alias_method. A `{` that
# opens a line after a call with arguments starts a new statement. /usr/bin/ruby
# 2.6 settles each.
def s124
  n = [2, 3, -1][0, 3]
  m = []
  [-> { (1..).take(n[2]) }, -> { (1..).first(n[2]) }, -> { ("a"..).first(n[2]) }].each do |l|
    begin; l.call; rescue ArgumentError => x; m << x.message; end
  end
  begin; (1..).min { |a, c| c <=> a }; rescue RangeError => x; m << x.message; end
  check("er124a", (1..).take(n[0]) == [1, 2] && ("a"..).take(n[1]) == ["a", "b", "c"] && (1...).take(0) == [] &&
        (1..).min(n[0]) == [1, 2] && ("a"..).min(n[0]) == ["a", "b"] && (2**70..).take(1) == [2**70] &&
        m == ["attempt to take negative size", "negative array size (or size too big)",
              "negative array size (or size too big)",
              "cannot get the minimum of endless range with custom comparison method"])
  b = [2**70][0]
  t = []
  b.times { |i| t << i; break if i == 1 }
  u = []
  b.upto(b + 1) { |x| u << x }
  check("bg124b", b.step(by: 3).first(2) == [b, b + 3] && b.step(b + 6, 3).to_a == [b, b + 3, b + 6] &&
        3.step(by: b).first(2) == [3, b + 3] && b.downto(b - 2).to_a == [b, b - 1, b - 2] && u == [b, b + 1] &&
        t == [0, 1] && b.times.first(2) == [0, 1] && (1..2 * b).step(b).to_a == [1, b + 1] &&
        (1..).step(b).first(2) == [1, b + 1] && (2**53 - 1).upto(2**53 + 1).to_a == [2**53 - 1, 2**53, 2**53 + 1])
  f = []
  for s in ("y"..)
    f << s
    break if s == "aa"
  end
  check("fr124c", f == ["y", "z", "aa"] && s == "aa")
  a = []
  begin; alias_method :a124, :s124; rescue NoMethodError => x; a << x.message; end
  check("tl124d", a == ["undefined method `alias_method' for main:Object"])
  o = []
  o.push 1
  {2 => 3}.each { |k, v| o << k + v }
  o.push(4)
  {5 => 6}.each { |k, v| o << k * v }
  check("nl124e", o == [1, 5, 4, 30])
end
# ===== SECTION 125: Kernel#then, count arguments, big #times =====
# Kernel#then and its alias yield_self answer the block's value; without a block
# they answer an Enumerator of size 1 whose #each answers the block's value. A
# count argument (take, first, drop, last ...) converts as a C long: a Float
# truncates, nil / a String / a Symbol is a TypeError, a value outside the long
# range a RangeError, and a negative one each method's own ArgumentError.
# Integer#times of a big Integer counts its #size exactly. /usr/bin/ruby 2.6
# settles each.
def s125
  v = [1, "ab", nil, 2**70, "x", -1, 1.9, 1e30]
  e = v[0].then
  check("th125a", v[0].then { |x| x + 1 } == 2 && v[1].yield_self { |s| s + "c" } == "abc" &&
    v[2].then { |x| x.inspect } == "nil" && v[0].respond_to?(:then) && v[1].respond_to?(:yield_self))
  check("th125b", e.size == 1 && e.to_a == [1] && e.each { |x| x + 5 } == 6 && e.inspect == "#<Enumerator: 1:then>")
  errs = []
  [-> { (1..3).take(v[4]) }, -> { (1..).take(v[4]) }, -> { [1].first(v[2]) }, -> { [1].drop(:s) },
   -> { [1].drop(v[5]) }, -> { {a: 1}.first(v[5]) }, -> { [1].take(v[3]) }, -> { [1].take(v[7]) }].each do |f|
    begin
      f.call
    rescue TypeError, ArgumentError, RangeError => x
      errs << x.class.name + ": " + x.message
    end
  end
  check("ct125c", errs == ["TypeError: no implicit conversion of String into Integer",
    "TypeError: no implicit conversion of String into Integer", "TypeError: no implicit conversion from nil to integer",
    "TypeError: no implicit conversion of Symbol into Integer", "ArgumentError: attempt to drop negative size",
    "ArgumentError: attempt to take negative size", "RangeError: bignum too big to convert into `long'",
    "RangeError: float 1e+30 out of range of integer"])
  check("ct125d", [1, 2, 3].take(v[6]) == [1] && (1..).first(v[6]) == [1] && [1, 2, 3].drop(v[6]) == [2, 3])
  check("bt125e", v[3].times.size == 1180591620717411303424 && (-v[3]).times.size == 0 && v[3].times.first(2) == [0, 1])
end
# ===== SECTION 126: methods a program gives Object =====
# `class Object; def m' defines a public method every receiver answers - main, a
# builtin value, a user instance and a class - after its own class's methods and
# before method_missing; a private one is reached only on the implicit self. A
# top-level define_method defines one too and answers its name. /usr/bin/ruby
# 2.6 settles each.
class Object
  def s126tw(x) x * 2; end
  def s126who; self.class.name; end
  def s126zz; 1; end
  def s126up?; respond_to?(:upcase); end
  private
  def s126pv; :pv; end
end
class S126F
  def s126zz; super + 1; end
  def go; [s126tw(4), s126who, s126pv]; end
  def method_missing(n, *a) "mm:#{n}"; end
end
S126DM = define_method(:s126dm) { |a| a * 3 }
def s126
  v = [3, "ab", [1], 1.5]
  check("ob126a", s126tw(v[0]) == 6 && v[0].s126tw(1) == 2 && v[1].s126tw(2) == 4 && v[2].s126who == "Array" &&
    v[3].s126who == "Float" && S126F.s126who == "Class" && Object.new.s126zz == 1)
  check("ob126b", S126F.new.go == [8, "S126F", :pv] && S126F.new.s126zz == 2 && S126F.new.s126nope == "mm:s126nope")
  check("ob126c", v[0].respond_to?(:s126tw) && S126F.new.respond_to?(:s126who) && !v[0].respond_to?(:s126pv) &&
    v[0].respond_to?(:s126pv, true) && v[1].s126up? && !v[0].s126up?)
  e = nil
  begin
    v[0].s126pv
  rescue NoMethodError => x
    e = x.message.lines.first.chomp
  end
  check("ob126d", e == "private method `s126pv' called for 3:Integer")
  check("ob126e", S126DM == :s126dm && s126dm(2) == 6 && v[0].s126dm(5) == 15)
end
# ===== SECTION 184: repeat counts, nil as self, NaN against big Integers, big walks =====
# String#* and Array#* take their count as a C long: a Float truncates, a big one
# or a non-finite Float is a RangeError, a count past the length cap is "argument
# too big", and an empty receiver answers at once. A method a program gives
# Object, run on nil, sees nil as self, bare calls included. A NaN against an
# Integer or Float is false under every relation and nil under <=>; a Rational's
# relations raise, as do between? and clamp, which compare min with max first.
# upto / downto / step with a bound past 2**53 walk exact Integers one at a time
# and know their exact size. /usr/bin/ruby 2.6 settles each.
class Object
  def s184me; self; end
  def s184via; s184me; end
  def s184d; "#{self.class} #{inspect}" + to_s; end
end
def s184err
  yield
rescue => e
  "#{e.class} #{e.message}"
end
def s184
  c = [2.9, -0.5, -1.5, 2**64, 2**62, 1e30, Float::NAN, 2**60]
  check("rp184a", "abc" * c[0] == "abcabc" && "abc" * c[1] == "" && [1, 2] * c[0] == [1, 2, 1, 2] && [1, 2] * c[1] == [] &&
        "ab".b * c[0] == "abab" && s184err { "a" * c[2] } == "ArgumentError negative argument" &&
        s184err { "a" * c[3] } == "RangeError bignum too big to convert into `long'" &&
        s184err { "ab" * c[4] } == "ArgumentError argument too big" && s184err { [1, 2] * c[4] } == "ArgumentError argument too big" &&
        s184err { [1] * c[7] } == "ArgumentError argument too big" && s184err { "a" * c[5] } == "RangeError float 1e+30 out of range of integer" &&
        s184err { [1] * c[6] } == "RangeError float NaN out of range of integer" && "" * c[4] == "" && [] * c[4] == [])
  s = "xy"
  s *= c[0]
  check("rp184b", s == "xyxy")
  n = [nil, false, 3]
  check("ns184c", n[0].s184me.nil? && n[0].s184via.nil? && n[0].s184d == "NilClass nil" && n[1].s184d == "FalseClass falsefalse" &&
        n[2].s184via == 3 && s184me.to_s == "main")
  b = [Float::NAN, 2**64, 3r/2, 1]
  check("nb184d", !(b[0] < b[1]) && !(b[1] < b[0]) && !(b[0] >= b[1]) && (b[0] <=> b[1]).nil? && (b[1] <=> b[0]).nil? &&
        !(b[0] == b[1]) && !(b[1] == b[0]) && b[1] != b[0] && !(b[0] == b[0]) && !b[0].eql?(b[0]) && !(b[3] > b[0]))
  check("nb184e", s184err { b[2] < b[0] } == "ArgumentError comparison of Rational with NaN failed" &&
        s184err { b[1].between?(b[0], 2**65) } == "ArgumentError comparison of Integer with NaN failed" &&
        s184err { b[1].clamp(b[0], 2**65) } == "ArgumentError comparison of Float with Integer failed" &&
        s184err { b[3].clamp(5, 2) } == "ArgumentError min argument must be smaller than max argument" &&
        !b[3].between?(b[1], b[0]) && b[1].clamp(1, 5) == 5 && b[3].clamp(b[3], b[1]) == 1)
  a = b[1]
  e = 1.upto(a)
  q = 1.step(2**70, 2**65)
  r = (a..a + 10).step(3)
  check("bw184f", e.size == a && e.first(2) == [1, 2] && e.inspect == "#<Enumerator: 1:upto(18446744073709551616)>" &&
        a.upto(a + 3).map { |x| x - a } == [0, 1, 2, 3] && a.upto(a - 1).size == 0 && a.downto(-a).size == 2 * a + 1 &&
        a.upto(a + 2).reverse_each.to_a == [a + 2, a + 1, a] && q.size == 32 && q.last == 31 * 2**65 + 1 &&
        q.inspect == "(1.step(1180591620717411303424, 36893488147419103232))" && 1.step(a, 3).size == 6148914691236517206 &&
        r.to_a == [a, a + 3, a + 6, a + 9] && r.last(2) == [a + 6, a + 9] && a.upto(1.8446744073709552e19 + 5000).count == 4097)
  t = 0
  1.upto(a) { |i| t += i; break if i == 4 }
  w = []
  a.step(a + 20, 7) { |x| w << x - a }
  check("bw184g", t == 10 && w == [0, 7, 14] && a.downto(1).first(2) == [a, a - 1] && a.step(1, -(2**62)).size == 4)
  m = "ArgumentError array size too big"
  check("fc184h", s184err { (1..Float::INFINITY).lazy.map { |x| x * 2 }.first(c[7]) } == m && s184err { [1, 2].each.first(c[4]) } == m &&
        s184err { (1..3).first(c[7]) } == m && s184err { 1.step.first(c[3] - 1) } == "RangeError bignum too big to convert into `long'" &&
        s184err { 3.times.first(c[7]) } == m && s184err { { a: 1 }.first(c[7]) } == m && [1, 2, 3].first(c[7]) == [1, 2, 3] &&
        (1..3).lazy.take(c[7]).to_a == [1, 2, 3] && s184err { (1..3).first(-1) } == "ArgumentError negative array size (or size too big)")
  x = [1, 2, 3, 4]
  check("ca184i", x[c[0]] == 3 && x[c[1]] == 1 && x[c[2]] == 4 && x.fetch(c[0]) == 3 && x.fetch(c[2]) == 4 && 10.to_s(c[0]) == "1010" &&
        (2**64).to_s(c[0] + 14) == "10000000000000000" && s184err { 10.to_s(c[1] + 1.4) } == "ArgumentError invalid radix 0" &&
        s184err { a.to_s(37) } == "ArgumentError invalid radix 37" && Array.new(c[0], 0) == [0, 0] &&
        s184err { Array.new(c[2]) } == "ArgumentError negative array size" && s184err { x.min(c[2]) } == "ArgumentError negative size (-1)" &&
        s184err { x.max(c[2]) } == "ArgumentError negative size (-1)" && s184err { x.sample(c[2]) } == "ArgumentError negative sample number" &&
        x.sample(c[1]) == [] && x.max(c[0]) == [4, 3])
end

# ===== SECTION 185: Float shift counts, byteslice, bare nil?, nil's ivars, stream breaks =====
# A Float or Rational shift width, bit index or digits radix goes through #to_int,
# which truncates; & | ^ refuse it with a coerce TypeError, and a non-number
# count is "no implicit conversion". String#byteslice indexes BYTES and keeps the
# encoding, so a cut through a character is an invalid String. A bare `nil?` is
# the method, not nil and a `?;` character literal. nil reads every instance
# variable as nil. A user Comparable's clamp checks min <= max first. upto with
# a block walks its numbers however far the bound, and count / sum / min_by ...
# with a block on an endless Enumerator end at a break. /usr/bin/ruby 2.6
# settles each.
class NilClass
  def s185iv; @s185z; end
end
class Object
  def s185ov; @s185q; end
  def s185np; nil? ? :nil : :obj; end
  def s185np2; [nil?, 1]; end
end
class S185V
  include Comparable
  attr_reader :v
  def initialize(v); @v = v; end
  def <=>(o); v <=> o.v; end
end
def s185err
  yield
rescue => e
  "#{e.class} #{e.message}"
end
def s185
  f = [2.9, 2.0, 0.5, -1.5, -0.9, 64.2, 1e30, Float::NAN, Rational(5, 2), 12.9, 16.5, nil, "a"]
  check("sh185a", (5 << f[0]) == 20 && (5 >> f[0]) == 1 && (5 << f[2]) == 5 && (5 << f[3]) == 2 && (5 >> f[3]) == 10 &&
        (-5 >> f[0]) == -2 && (5 << f[5]) == 92233720368547758080 && (5 >> f[5]) == 0 && (-5 >> f[5]) == -1 &&
        (2**70 >> f[0]) == 295147905179352825856 && (5 << f[8]) == 20 && 5[f[0]] == 1 && 5[f[3]] == 0 && 5[f[4]] == 1 &&
        -5[f[4]] == 1 && 5[f[6]] == 0 && -5[f[6]] == 1 && 5[f[8]] == 1 && (5 >> f[6]) == 0 &&
        s185err { 5 << f[6] } == "RangeError shift width too big" && s185err { 5 << f[7] } == "FloatDomainError NaN" &&
        s185err { 5 & f[0] } == "TypeError 2.9 can't be coerced into Integer" &&
        s185err { 5 | f[8] } == "TypeError Rational can't be coerced into Integer" &&
        s185err { 5 ^ f[11] } == "TypeError nil can't be coerced into Integer" &&
        s185err { 5 << f[11] } == "TypeError no implicit conversion of nil into Integer" &&
        s185err { 5 >> f[12] } == "TypeError no implicit conversion of String into Integer" &&
        (0 << 2**70) == 0 && (-5 >> 2**70) == -1 && s185err { 1 << 2**70 } == "RangeError shift width too big")
  check("dg185b", 5.digits(f[9]) == [5] && 255.digits(f[10]) == [15, 15] && 1000.digits(f[9]) == [4, 11, 6] &&
        5.digits(f[1]) == [1, 0, 1] && 5.digits(f[8]) == [1, 0, 1] && (2**70).digits(f[10]).last == 4 &&
        s185err { 5.digits(f[2]) } == "ArgumentError invalid radix 0" &&
        s185err { 5.digits(f[11]) } == "TypeError no implicit conversion of nil into Integer")
  s = ["héllo", "hello", "a€b"]
  check("bs185c", s[0].byteslice(1, 2) == "é" && s[1].byteslice(1) == "e" && s[1].byteslice(1..3) == "ell" &&
        s[1].byteslice(-3, 2) == "ll" && s[1].byteslice(5).nil? && s[1].byteslice(5, 2) == "" && s[1].byteslice(1, -1).nil? &&
        s[1].byteslice(6, 1).nil? && s[1].byteslice(2..) == "llo" && s[1].byteslice(3..1) == "" &&
        !s[0].byteslice(1, 1).valid_encoding? && s[0].byteslice(1, 1).bytesize == 1 &&
        s[0].byteslice(1, 1).encoding.to_s == "UTF-8" && s[2].byteslice(1, 3) == "€" &&
        s[2].byteslice(2, 5).bytes == [130, 172, 98] && "abc".b.byteslice(1, 2).encoding.to_s == "ASCII-8BIT" &&
        s[1].respond_to?(:byteslice))
  check("np185d", nil.s185np == :nil && 3.s185np == :obj && 3.s185np2 == [false, 1] && nil.s185np2 == [true, 1])
  check("ns185e", nil.s185iv.nil? && nil.s185ov.nil? && 5.s185ov.nil? && false.s185ov.nil?)
  check("cl185f", s185err { S185V.new(5).clamp(S185V.new(9), S185V.new(1)) } == "ArgumentError min argument must be smaller than max argument" &&
        S185V.new(5).clamp(S185V.new(1), S185V.new(3)).v == 3 && S185V.new(5).clamp(S185V.new(5), S185V.new(5)).v == 5)
  n = [10**15, 5]
  c = 0
  1.upto(n[0]) { |i| c += 1; break if i > 5 }
  d = 0
  r = n[0].downto(1) { |i| d += 1; break i if d > 3 }
  t = 0
  n[0].times { |i| t += i; break if i == n[1] }
  check("ut185g", c == 6 && r == n[0] - 3 && t == 15 && 2.upto(n[1]).to_a == [2, 3, 4, 5] && n[1].times.to_a == [0, 1, 2, 3, 4])
  e = (1..Float::INFINITY).each
  check("lz185h", e.count { |x| break 42 if x > 3; true } == 42 && 1.step.count { |x| break x if x > 5; true } == 6 &&
        [1, 2].cycle.count { |x| break :c if x > 1; true } == :c && e.sum { |x| break x if x > 3; x } == 4 &&
        e.min_by { |x| break x if x > 2; -x } == 3 && e.sort_by { |x| break :sb if x > 2; x } == :sb &&
        e.group_by { |x| break :gb if x > 2; x } == :gb && e.partition { |x| break :pt if x > 2; x } == :pt &&
        e.max_by { |x| break :mb if x > 2; x } == :mb && e.minmax_by { |x| break :mm if x > 2; x } == :mm &&
        e.to_h { |x| break :th if x > 2; [x, x] } == :th && e.lazy.map { |x| x * 2 }.count { |x| break x if x > 6; true } == 8 &&
        (1..4).each.count(&:even?) == 2 && (1..4).each.sum { |x| x * 2 } == 20 && [3, 1, 2].each.sort_by { |x| -x } == [3, 2, 1])
end

# ===== SECTION 186: include into Object, implicit-self builtins, folds on endless Enumerators, instance_eval =====
# A module included into Object reaches every value. Inside a method added to a
# builtin class a bare builtin name - even?, times with a block, sub(...) - is a
# call on self, and respond_to? answers those names. inject / reduce and min /
# max / minmax with a block, and one?, take an endless Enumerator one value at a
# time, so a break in the block ends them. instance_eval / instance_exec run the
# block with self the receiver and a Proc's argument rules. /usr/bin/ruby 2.6
# settles each.
module S186M
  def s186om; :om; end
end
class Object
  include S186M
end
class Integer
  def s186e; [frozen?, even?, odd?, pow(2), gcd(6)]; end
  def s186t; r = []; times { |i| r << i }; r; end
end
class String
  def s186s; [sub("a", "b"), match?(/b/), scan(/a/), gsub("a") { "c" }]; end
end
class Array
  def s186a; [at(1), dig(0), find_all { |x| x > 1 }]; end
end
class S186P
  def initialize; @a = 3; end
  def pub; :pub; end
  def twice(x); x * 2; end
  def inner; [1].map { |q| instance_eval { @a + q } }; end
  private def hid; :hid; end
end
def s186ret(o); o.instance_eval { return 11 }; 12; end
def s186
  check("ic186a", 5.s186om == :om && "x".s186om == :om && nil.s186om == :om && S186P.new.s186om == :om &&
        [1].s186om == :om && 5.respond_to?(:s186om))
  check("is186b", 4.s186e == [true, true, false, 16, 2] && 3.s186t == [0, 1, 2] &&
        "aba".s186s == ["bba", true, ["a", "a"], "cbc"] && [1, 2, 3].s186a == [2, 1, [2, 3]] &&
        4.respond_to?(:times) && 4.respond_to?(:upto) && "s".respond_to?(:gsub) && "s".respond_to?(:match?) &&
        [].respond_to?(:dig) && {}.respond_to?(:store) && (1..2).respond_to?(:to_h) && 1.5.respond_to?(:clamp))
  e = (1..Float::INFINITY).each
  check("lz186c", e.inject { |a, x| break a if x > 4; a + x } == 10 && e.reduce(0) { |a, x| break a if x > 3; a + x } == 6 &&
        e.max { break :mx } == :mx && e.min { break :mn } == :mn && e.minmax { break :mm } == :mm &&
        e.one? { |x| break :one if x > 2; false } == :one && e.one? { |x| x > 2 } == false && e.one?(3..4) == false &&
        e.max { |a, b| break [a, b] if a > 3; a <=> b } == [4, 3] && 1.step.inject { |a, x| break a if x > 5; a * x } == 120 &&
        e.lazy.map { |x| x * 2 }.inject { |a, x| break a if x > 8; a + x } == 20)
  f = (1..5).each
  check("lz186d", f.inject { |a, x| a + x } == 15 && f.inject(10) { |a, x| a + x } == 25 && f.reduce(:+) == 15 &&
        f.max { |a, b| a <=> b } == 5 && f.min { |a, b| a <=> b } == 1 && f.minmax { |a, b| b <=> a } == [5, 1] &&
        f.max(2) { |a, b| a <=> b } == [5, 4] && f.one? { |x| x == 3 } && !f.one?(2..3) && [].each.inject { |a, x| a + x }.nil? &&
        [].each.minmax { |a, b| a <=> b } == [nil, nil] && [[1, 2], [3, 4]].each.inject { |a, x| a + x } == [1, 2, 3, 4] &&
        {a: 1, b: 2}.each.max { |a, b| a[1] <=> b[1] } == [:b, 2])
  o = S186P.new
  x = 5
  o.instance_eval { x += @a }
  check("ie186e", o.instance_eval { @a } == 3 && o.instance_eval { self.class } == S186P && o.instance_exec(4) { |y| @a + y } == 7 &&
        o.instance_eval { [pub, twice(2), hid] } == [:pub, 4, :hid] && o.instance_exec { |p, q| [p, q] } == [nil, nil] &&
        o.instance_exec([5, 6]) { |p, q| [p, q] } == [5, 6] && o.instance_eval { |s| s.equal?(o) } &&
        o.instance_eval { break 7; 8 } == 7 && s186ret(o) == 11 && o.inner == [4] && x == 8 &&
        nil.instance_eval { self }.nil? && 5.instance_exec(2) { |y| self * y } == 10 && "abc".instance_eval { size } == 3 &&
        ({k: 1}.instance_exec(:k) { |k| fetch(k) }) == 1 && S186P.instance_eval { name } == "S186P")
  q = S186P.new
  q.instance_eval { @b = 9 }
  check("ie186f", q.instance_variable_get(:@b) == 9 && (o.instance_eval { raise "boom" } rescue $!.message) == "boom")
end

s97() # SECTION-CALL 97
s98() # SECTION-CALL 98
s99() # SECTION-CALL 99
s100() # SECTION-CALL 100
s101() # SECTION-CALL 101
s102() # SECTION-CALL 102
s103() # SECTION-CALL 103
s104() # SECTION-CALL 104
s105() # SECTION-CALL 105
s106() # SECTION-CALL 106
s107() # SECTION-CALL 107
s108() # SECTION-CALL 108
s109() # SECTION-CALL 109
s110() # SECTION-CALL 110
s111() # SECTION-CALL 111
s112() # SECTION-CALL 112
s113() # SECTION-CALL 113
s114() # SECTION-CALL 114
s115() # SECTION-CALL 115
s116() # SECTION-CALL 116
s117() # SECTION-CALL 117
s118() # SECTION-CALL 118
s119() # SECTION-CALL 119
s120() # SECTION-CALL 120
s121() # SECTION-CALL 121
s122() # SECTION-CALL 122
s123() # SECTION-CALL 123
s124() # SECTION-CALL 124
s125() # SECTION-CALL 125
s126() # SECTION-CALL 126
s184() # SECTION-CALL 184
s185() # SECTION-CALL 185
s186() # SECTION-CALL 186
puts "full: #{FULLC[0]} checks, #{FULLC[1]} failures"
exit(FULLC[1])
