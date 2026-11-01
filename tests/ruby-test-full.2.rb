# Full-syntax test: Ruby - part 2, SECTION 122 onward. tests/ruby-test-full.rb
# leaves the -frozen interpreter leg too little of the IR interpreter's step
# brake (-max-steps, 100M per call) for more; see that file for the
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
# ===== SECTION 122: class-body local scope; a class statement in a module =====
# A class or module body is a new local scope: it cannot read the locals around
# it (the top level's or an enclosing body's), and a local it binds stays its
# own. `class X' in a module body makes M::X even beside a top-level X, a later
# `module M' body reopens M::X rather than X, and a method of M names M::X by a
# bare X. /usr/bin/ruby 2.6 settles each.
s122v = 1
class S122A
  S122R1 = begin; s122v; rescue NameError => e; e.message; end
  y = 2
  class B
    R2 = begin; y; rescue NameError => e; e.message; end
  end
  module M
    R3 = defined?(y).inspect
  end
  R4 = y
end
module S122N
  z = 3
  class C
    z = 4
    R5 = z
  end
  R6 = z
end
class S122X; def a; 1; end; end
module S122M
  class S122X; def b; 2; end; end
  class S122Y; def d; 4; end; end
  K = 7
  def self.mk; S122X.new.b + K; end
end
module S122M
  class S122X; def c; 3; end; end
end
class S122Y; def e; 5; end; end
def s122
  check("cg122a", S122A::S122R1 == "undefined local variable or method `s122v' for S122A:Class" &&
        S122A::B::R2 == "undefined local variable or method `y' for S122A::B:Class" && S122A::M::R3 == "nil" &&
        S122A::R4 == 2 && S122N::C::R5 == 4 && S122N::R6 == 3)
  check("nr122b", S122M::S122X != S122X && S122M::S122X.name == "S122M::S122X" && S122X.instance_methods(false) == [:a] &&
        S122M::S122X.instance_methods(false).sort == [:b, :c] && S122M.mk == 9 && S122X.new.a == 1 &&
        S122Y != S122M::S122Y && S122Y.instance_methods(false) == [:e] && S122M::S122Y.new.d == 4)
end

# ===== SECTION 123: visibility queries =====
# Module#public_method_defined? / private_method_defined? /
# protected_method_defined? answer the method's visibility along the ancestors
# (only the class itself with a false second argument); method_defined? is public
# or protected. initialize, the private Kernel methods and a mixed-in module's
# private methods are private, and a subclass may make an inherited method
# private or public again. /usr/bin/ruby 2.6 settles each.
class S123A
  def pub; end
  def initialize; end
  private
  def pri; end
  protected
  def pro; end
  public
  private def pri2; end
end
class S123B < S123A
  def pro; end
  private :pub
end
module S123M
  def mpub; mpri + 1; end
  private
  def mpri; 10; end
end
class S123C
  include S123M
end
def s123
  vk = lambda do |k, n|
    [k.method_defined?(n), k.public_method_defined?(n), k.private_method_defined?(n), k.protected_method_defined?(n)]
  end
  check("vq123a", vk[S123A, :pub] == [true, true, false, false] && vk[S123A, :pri] == [false, false, true, false] &&
        vk[S123A, :pro] == [true, false, false, true] && vk[S123A, :pri2] == [false, false, true, false] &&
        vk[S123A, :initialize] == [false, false, true, false] && vk[S123A, :puts] == [false, false, true, false] &&
        vk[S123A, :nope] == [false, false, false, false] && vk[S123A, :to_s] == [true, true, false, false])
  check("vq123b", vk[S123B, :pub] == [false, false, true, false] && vk[S123B, :pro] == [true, true, false, false] &&
        vk[S123B, :pri] == [false, false, true, false] && S123B.private_method_defined?(:pri, false) == false &&
        S123B.public_method_defined?(:pro, false) && String.public_method_defined?(:upcase) &&
        !String.private_method_defined?(:upcase) && Integer.method_defined?(:+))
  me = begin; S123C.new.mpri; rescue NoMethodError => e; e.message; end
  check("vq123c", vk[S123C, :mpub] == [true, true, false, false] && vk[S123C, :mpri] == [false, false, true, false] &&
        S123M.private_method_defined?(:mpri) && S123C.new.mpub == 11 && me.start_with?("private method `mpri' called for"))
end

# ===== SECTION 124: US-ASCII and invalid UTF-8 Strings; pack / unpack =====
# Integer#chr below 128 is a US-ASCII String; joining Strings keeps the left
# side's encoding when the right is ASCII-only and takes the right side's when
# the left is; an interpolated binary String makes the result binary. A "\xHH"
# escape is a byte: "\xC3\xA9" is "é", "\xFF" an invalid UTF-8 String that
# counts the byte as one character and inspects as \xFF; force_encoding on such
# a String changes it in place. pack / unpack read and write the integer
# directives (sizes, byte orders, signs), a A Z B b H h m, and x X @; pack's
# String is UTF-8 when U comes first, US-ASCII for m alone, else binary.
# /usr/bin/ruby 2.6 settles each.
def s124
  cs = [65, 66, 255]
  a = cs[0].chr
  ab = [a, "B"].join
  s = cs[0].chr
  s << "x"
  check("ua124a", a.encoding == Encoding::US_ASCII && a == "A" && a.valid_encoding? && (a + "x").encoding == Encoding::US_ASCII &&
        ("x" + a).encoding == Encoding::UTF_8 && (a + "é").encoding == Encoding::UTF_8 && (a + cs[2].chr).encoding == Encoding::BINARY &&
        a.upcase.encoding == Encoding::US_ASCII && ab.encoding == Encoding::US_ASCII && s.encoding == Encoding::US_ASCII &&
        "#{a}".encoding == Encoding::UTF_8 && a.to_sym == :A && "é".force_encoding("US-ASCII").valid_encoding? == false)
  y = "x\xFFy"
  q = "#{cs[2].chr}"
  ie = begin; "é#{cs[2].chr}"; rescue EncodingError => e; e.class; end
  check("iv124b", y.encoding == Encoding::UTF_8 && y.size == 3 && y.bytes == [120, 255, 121] && !y.valid_encoding? &&
        y.inspect == "\"x\\xFFy\"" && (y + "z").inspect == "\"x\\xFFyz\"" && "\xC3\xA9" == "é" && "\303\251".size == 1 &&
        "a\xFFé".inspect == "\"a\\xFFé\"" && ?\M-a.bytes == [225] && q.encoding == Encoding::BINARY &&
        "a#{cs[2].chr}".encoding == Encoding::BINARY && ie == Encoding::CompatibilityError && y.b.encoding == Encoding::BINARY)
  w = "\xFF"
  w.force_encoding("ASCII-8BIT")
  r = "abc".b
  r.force_encoding("US-ASCII")
  check("fe124c", w.encoding == Encoding::BINARY && r.encoding == Encoding::US_ASCII && 200.chr.force_encoding("UTF-8").valid_encoding? == false)
  x = [1, 258, 65535, -1].pack("n*")
  check("pk124d", x.unpack("n*") == [1, 258, 65535, 65535] && x.unpack("s>*") == [1, 258, -1, -1] && [1, -2].pack("s<s>") == "\x01\x00\xFF\xFE".b &&
        [1, 2].pack("N V").bytes == [0, 0, 0, 1, 2, 0, 0, 0] && [2**31, 3].pack("L<L>").unpack("L<L>") == [2147483648, 3] &&
        [-5, 2**40].pack("q<Q>").unpack("q<Q>") == [-5, 1099511627776] && [-1].pack("q").unpack("Q") == [18446744073709551615] &&
        [2**63 - 1, -2**63].pack("q>q<").unpack("q>q<") == [9223372036854775807, -9223372036854775808] && "a".unpack("N") == [nil])
  check("ps124e", ["ab", "c", "d e"].pack("a3A3Z*") == "ab\x00c  d e\x00".b && "h\xC3\xA9llo\x00 wo".unpack("a3 A4 Z*") == ["h\xC3\xA9".b, "llo", " wo"] &&
        ["10110", "0011"].pack("B*b*") == "\xB0\f".b && "AB".unpack("B*b*") == ["0100000101000010", ""] && "AB".unpack("B4b3") == ["0100", "010"] &&
        ["4142", "1424"].pack("H*h*") == "ABAB" && "\xAB".unpack("H") == ["a"] && "AB".unpack("H*")[0].encoding == Encoding::US_ASCII &&
        ["hello world"].pack("m") == "aGVsbG8gd29ybGQ=\n" && ["hello world"].pack("m0") == "aGVsbG8gd29ybGQ=" && "aGVsbG8gd29ybGQ=\n".unpack("m") == ["hello world"] &&
        [1, 2].pack("CxC") == "\x01\x00\x02" && "A\x00B".unpack("CxC") == [65, 66] && "abc".unpack("a1@0a2") == ["a", "ab"])
  te = begin; ["x"].pack("C"); rescue TypeError => e; e.message; end
  check("pe124f", [65, 66].pack("U*").encoding == Encoding::UTF_8 && [8364, 65].pack("UC").encoding == Encoding::BINARY &&
        ["a"].pack("m").encoding == Encoding::US_ASCII && [1].pack("n").encoding == Encoding::BINARY && [0x110000].pack("U").bytes == [244, 144, 128, 128] &&
        te == "no implicit conversion of String into Integer" && [1.9, -1.5].pack("c*").bytes == [1, 255])
end

# ===== SECTION 125: Hash keys across encodings =====
# A String key is eql? to a String of the same bytes in the same encoding, or
# in any encoding when the bytes are ASCII-only, so 65.chr (US-ASCII) and "A"
# are one key while "é" and "é".b are two. /usr/bin/ruby 2.6 settles each.
def s125
  cs = [65, 255]
  h = {cs[0].chr => 1}
  g = {}
  g["A"] = 1
  g[cs[0].chr] = 2
  check("hk125a", h["A"] == 1 && {"A" => 2}[cs[0].chr] == 2 && h.key?("A") && {"A".b => 5}["A"] == 5 &&
        {"\xFF".b => 3}["\xFF"].nil? && {"é" => 4}["é".b].nil? && g.size == 1 && g["A"] == 2 &&
        g.keys[0].encoding == Encoding::UTF_8 && h.keys[0].encoding == Encoding::US_ASCII)
end

# ===== SECTION 126: private_instance_methods / protected_instance_methods =====
# Each lists the names its ancestors give that visibility themselves, nearest
# first; `private :m' or `public :m' over an inherited m gives the class its own
# entry (also in instance_methods(false)); initialize is private; with
# inheritance a class also answers the top-level defs and the private Kernel
# methods. A module body takes `private :m' too. /usr/bin/ruby 2.6 settles each.
class S126C; def a; end; protected def c; end; private; def b; end; def initialize; end; end
module S126M
  def x; end
  def z; end
  private :z
  private
  def y; end
end
class S126D < S126C; private def q; end; private :a; include S126M; end
class S126E < S126C; protected :a; public :b; end
def s126top; end
def s126
  check("pi126a", S126C.private_instance_methods(false).sort == [:b, :initialize] && S126M.private_instance_methods.sort == [:y, :z] &&
        S126D.private_instance_methods(false).sort == [:a, :q] && S126D.private_instance_methods.include?(:z) &&
        !S126D.private_instance_methods.include?(:x) && S126D.private_instance_methods.include?(:b))
  check("pi126b", S126C.private_instance_methods.include?(:puts) && S126C.private_instance_methods.include?(:s126top) &&
        !S126C.private_instance_methods.include?(:a) && S126C.protected_instance_methods == [:c] &&
        S126E.protected_instance_methods(false) == [:a] && S126E.instance_methods(false).sort == [:a, :b] &&
        !S126E.private_instance_methods.include?(:b) && S126M.private_method_defined?(:z))
end

# ===== SECTION 127: constants of a class or module body stay its own =====
# A class, module or constant a body binds is a member of it, not a top-level
# name (MRI's NameError outside); a method the body calls already sees it, and a
# reopened body sees its own before a top-level one of the name. A module
# method copied into a class reads the module's constants. /usr/bin/ruby 2.6
# settles each.
module S127M
  K = 1
  class S127X; def k; K; end; end
  module S127Y; def self.x; S127X; end; end
  def self.f; K; end
  R = f
  def kk; K; end
end
class S127Z; include S127M; end
S127X = 5
module S127M; R2 = S127X; end
def s127
  y = begin; S127Y; rescue NameError => e; e.message; end
  k = begin; K; rescue NameError => e; e.message; end
  check("nc127a", y.start_with?("uninitialized constant S127Y") && k.start_with?("uninitialized constant K") && S127M::S127X.new.k == 1 &&
        S127M::S127Y.x == S127M::S127X && S127M::R == 1 && S127X == 5 && S127M::R2 == S127M::S127X && S127Z.new.kk == 1)
end

# ===== SECTION 128: float pack directives; String#encode =====
# e E g G d D f F pack and unpack IEEE 754 binary32 (e g f F) and binary64 (E G
# d D), little-endian (e E), big-endian (g G) or native; a binary32 rounds half
# to even and a value past its largest is an infinity. String#encode converts
# between UTF-8, US-ASCII and ASCII-8BIT: a character the target has no code for
# is Encoding::UndefinedConversionError, an invalid byte sequence
# Encoding::InvalidByteSequenceError, each replaced by invalid: / undef:
# :replace; scrub replaces the invalid sequences. /usr/bin/ruby 2.6 settles each.
def s128
  v = [1.5, 0.1, -0.0, 1e-310, 1e-40, 3.4028235e38, 2**70]
  check("pf128a", [v[0]].pack("e").bytes == [0, 0, 192, 63] && [v[0]].pack("G").bytes == [63, 248, 0, 0, 0, 0, 0, 0] &&
        [v[1]].pack("g").unpack1("g") == 0.10000000149011612 && [v[1]].pack("D").unpack1("D") == 0.1 &&
        [v[2]].pack("E").bytes == [0, 0, 0, 0, 0, 0, 0, 128] && [v[3]].pack("d").unpack1("d") == 1e-310 &&
        [v[4]].pack("f").unpack1("F") == 9.99994610111476e-41 && [v[5]].pack("e").bytes == [0, 0, 128, 127] &&
        [v[6]].pack("E").unpack1("E") == 1.1805916207174113e+21 && [1, 2.5].pack("eg").unpack("eg") == [1.0, 2.5] &&
        "abc".unpack("e") == [nil] && [v[0]].pack("e").encoding == Encoding::BINARY)
  te = begin; [nil].pack("e"); rescue TypeError => e; e.message; end
  ae = begin; [v[0]].pack("e<"); rescue ArgumentError => e; e.message; end
  check("pf128b", te == "can't convert nil into Float" && ae == "'<' allowed only after types sSiIlLqQjJ")
  e1 = begin; "é".encode("US-ASCII"); rescue EncodingError => e; [e.class, e.message]; end
  e2 = begin; "\xFF".b.encode("UTF-8"); rescue EncodingError => e; [e.class, e.message]; end
  e3 = begin; "a\xE3\x81x".encode("US-ASCII"); rescue EncodingError => e; [e.class, e.message]; end
  e4 = begin; "a\xE3\x81".encode("BINARY"); rescue EncodingError => e; [e.class, e.message]; end
  check("en128c", e1 == [Encoding::UndefinedConversionError, "U+00E9 from UTF-8 to US-ASCII"] &&
        e2 == [Encoding::UndefinedConversionError, "\"\\xFF\" from ASCII-8BIT to UTF-8"] &&
        e3 == [Encoding::InvalidByteSequenceError, "\"\\xE3\\x81\" followed by \"x\" on UTF-8"] &&
        e4 == [Encoding::InvalidByteSequenceError, "incomplete \"\\xE3\\x81\" on UTF-8"] &&
        "abc".b.encode("UTF-8").encoding == Encoding::UTF_8 && "abc".encode("ascii").encoding == Encoding::US_ASCII &&
        "\xFF".encode("UTF-8").valid_encoding? == false)
  check("en128d", "aé\xFFb".encode("US-ASCII", invalid: :replace, undef: :replace) == "a??b" &&
        "aé\xFFb".encode("US-ASCII", invalid: :replace, undef: :replace, replace: "*") == "a**b" &&
        "a\xFFb".encode("UTF-8", invalid: :replace) == "a\u{FFFD}b" && "\xC0\xAF".encode("US-ASCII", invalid: :replace) == "??" &&
        "\xE3\x81\xE3\x81\x82".scrub == "\u{FFFD}あ" && "\xFF".scrub("?") == "?" && "é".encode(Encoding::BINARY, undef: :replace) == "?")
end

# ===== SECTION 129: a String re-encoded in place =====
# << of a binary String onto an ASCII-only UTF-8 String, and force_encoding,
# change the String object itself, so an alias sees the new encoding and bytes;
# << and force_encoding answer the receiver. A frozen String raises FrozenError.
# Identity methods of a binary String answer on the object. /usr/bin/ruby 2.6
# settles each.
def s129
  cs = [255, 200]
  z = +""
  a = z
  r = (z << cs[0].chr)
  z << "A"
  check("ri129a", a.encoding == Encoding::BINARY && r.equal?(z) && a == "\xFFA".b && a.size == 2 && a.bytes == [255, 65] &&
        a.inspect == "\"\\xFFA\"" && "#{a}".encoding == Encoding::BINARY && {a => 1}["\xFFA".b] == 1 && (a + "b") == "\xFFAb".b &&
        a.to_s.equal?(a) && a.itself.equal?(a))
  x = +"é"
  y = x
  q = x.force_encoding("BINARY")
  m = +"x"
  m2 = m
  m.concat(cs[0].chr, "y", cs[1].chr)
  fe = begin; "fz".freeze.force_encoding("BINARY"); rescue FrozenError => e; e.class; end
  ce = begin; (+"é") << cs[0].chr; rescue EncodingError => e; e.class; end
  check("ri129b", q.equal?(x) && y.encoding == Encoding::BINARY && y.bytes == [195, 169] && y.size == 2 && !(y == "é") &&
        m2.bytes == [120, 255, 121, 200] && m2.encoding == Encoding::BINARY && fe == FrozenError &&
        ce == Encoding::CompatibilityError)
end

# ===== SECTION 130: case mapping and bang methods of a binary String =====
# upcase / downcase / swapcase / capitalize of a String that is not UTF-8 text
# map its ASCII letters only, of invalid UTF-8 they raise ArgumentError; a bang
# method of a binary String changes it in place (nil when nothing changed), and
# force_encoding to bytes that are valid UTF-8 makes the object itself a UTF-8
# String, so an alias sees it. /usr/bin/ruby 2.6 settles each.
def s130
  bs = [255, 195, 169]
  h = bs[0].chr + "Ab"
  u = bs[1].chr + bs[2].chr + "ab"
  check("bc130a", h.upcase == "\xFFAB".b && h.downcase == "\xFFab".b && h.swapcase == "\xFFaB".b && h.capitalize == "\xFFab".b &&
        u.upcase.bytes == [195, 169, 65, 66] && u.downcase.bytes == [195, 169, 97, 98] && u.swapcase.encoding == Encoding::BINARY &&
        (bs[2].chr + "a").capitalize.bytes == [169, 97] && u.force_encoding("US-ASCII").upcase.bytes == [195, 169, 65, 66])
  iv = begin; ("\xFF" + "ab").upcase; rescue ArgumentError => e; e.message; end
  w = bs[1].chr + "ab" + bs[0].chr
  w2 = w
  r1 = w.upcase!
  r2 = (+"AB").b.upcase!
  t = (+" ab").b + bs[0].chr + "\n"
  t2 = t
  t.chomp!
  t.strip!
  check("bc130b", iv == "input string invalid" && r1.equal?(w) && w2.bytes == [195, 65, 66, 255] && r2.nil? &&
        t2.bytes == [97, 98, 255] && t.sub!("q", "x").nil? && t.sub!("a", "x").equal?(t) && t2.bytes == [120, 98, 255])
  x = bs[1].chr + bs[2].chr
  y = x
  x.force_encoding("UTF-8")
  z = bs[1].chr + "x" + bs[0].chr
  z2 = z
  z.chop!
  z.chop!
  z.force_encoding("UTF-8")
  check("bc130c", y.encoding == Encoding::UTF_8 && y == "\u00e9" && y.length == 1 && y.valid_encoding? && x.equal?(y) &&
        (y + "!") == "\u00e9!" && y.upcase == "\u00c9" && z2 == "\xC3".b.force_encoding("UTF-8") && z2.encoding == Encoding::UTF_8)
end

# ===== SECTION 131: visibility re-exposed; a rescue binding before `[' =====
# `public :m' over an inherited private or protected m, or a mixed-in module's
# private m, makes m public for the class (the superclass keeps it private);
# `private def m' in a module body makes m a private method of the module. The
# `=> e' of a rescue ends at its line, so a handler whose first line opens with
# `[' is an Array, not e[...]. /usr/bin/ruby 2.6 settles each.
class S131C; private def d; 4; end; protected def f; 6; end; end
class S131D < S131C; public :d, :f; end
class S131E < S131D; end
module S131M; private def mm; 8; end; def pm; mm + 1; end; end
class S131F; include S131M; public :mm; end
class S131G; include S131M; end
def s131rb
  raise ArgumentError, "q"
rescue ArgumentError => e
  [e.class, e.message]
end
def s131
  pe = begin; S131C.new.d; rescue NoMethodError => e; e.message[0, 18]; end
  ge = begin; S131G.new.mm; rescue NoMethodError => e; e.message[0, 19]; end
  check("pv131a", S131D.new.d == 4 && S131D.new.f == 6 && S131E.new.d == 4 && pe == "private method `d'" &&
        S131D.public_method_defined?(:d) && S131C.private_method_defined?(:d) && S131F.new.mm == 8 &&
        S131G.new.pm == 9 && ge == "private method `mm'" && S131M.private_method_defined?(:mm))
  check("rb131b", s131rb == [ArgumentError, "q"])
end

# ===== SECTION 132: Module#class_eval / #module_eval with a block =====
# The block runs with self the class and answers its last value; a def in it
# is an instance method of the class, which the class's other methods call on
# the implicit self; define_method in it adds one too, and class_exec hands
# the block its arguments. /usr/bin/ruby 2.6 settles each.
class S132A; def base; 1; end; end
def s132
  r = S132A.module_eval { def q; 7; end; self }
  S132A.class_eval do
    def name2; "e"; end
    define_method(:dm) { base + 10 }
    def uses; q + base; end
  end
  k = 3
  S132A.class_exec(5) { |n| define_method(:kn) { k + n } }
  v = S132A.class_eval { 40 + 2 }
  a = S132A.new
  check("ce132a", r == S132A && a.q == 7 && a.name2 == "e" && a.dm == 11 && a.uses == 8 && a.kn == 8 && v == 42 &&
        S132A.instance_methods(false).sort == [:base, :dm, :kn, :name2, :q, :uses] && a.respond_to?(:uses))
end

# ===== SECTION 133: frozen binary Strings; super above Object; class_eval mixins =====
# A binary String freezes like any other: frozen? answers true, its mutators and
# force_encoding raise FrozenError, clone keeps the flag, dup and +@ drop it, -@
# answers a frozen one. A super in a top-level def, a method of Object with
# nothing above it, is a rescuable NoMethodError naming the method. In a class_eval block `include M' mixes M
# into the class, a bare `private' makes the defs after it private (callable
# on the implicit self, with or without arguments), and Module#include with a
# receiver includes its last argument first; a method a module_eval gave M
# reaches a class that includes M later. /usr/bin/ruby 2.6 settles each.
def s133fz(s)
  begin; s.force_encoding("UTF-8"); :none; rescue FrozenError => e; e.class; end
end
def s133top
  super
rescue NoMethodError => e
  [6, e.class, e.name]
end
module S133M; def hi; "hi"; end; end
module S133M2; def hi; "m2"; end; def lo; "lo"; end; end
module S133N; end
S133N.module_eval { def m2; :m2; end }
class S133A; end
S133A.class_eval do
  include S133M
  def early; helper + 1; end
  private
  def helper; 41; end
  def twice(x); x * 2; end
  public
  def late; twice(5); end
end
class S133B; include S133N; end
class S133C; end
S133C.include(S133M, S133M2)
def s133
  x = "\xC3\xA9".b.freeze
  y = "\xff".b
  check("fz133a", x.frozen? && s133fz(x) == FrozenError && x.clone.frozen? && !x.dup.frozen? &&
        !(+x).frozen? && (-y).frozen? && !y.frozen? && (+y).equal?(y) && x.encoding == Encoding::ASCII_8BIT &&
        (begin; x << "a"; rescue FrozenError; true; end))
  pv = begin; S133A.new.twice(1); rescue NoMethodError => e; e.message[0, 22]; end
  check("su133b", s133top == [6, NoMethodError, :s133top])
  check("ce133c", S133A.new.hi == "hi" && S133A.new.early == 42 && S133A.new.late == 10 &&
        !S133A.new.respond_to?(:helper) && S133A.private_method_defined?(:helper) && pv == "private method `twice'" &&
        S133A.include?(S133M) && S133B.new.m2 == :m2 && S133C.new.hi == "hi" && S133C.new.lo == "lo" &&
        S133C.ancestors.take(3) == [S133C, S133M, S133M2])
end
# ===== SECTION 134: super through mixins; multi-argument mixins; run-time visibility =====
# A super reaches the nearest ancestor above the running method: from a class's
# method into a module it includes (with or without a superclass), from one
# included module into the next or into the superclass, and from a prepending
# class's own method past the prepended module. Nothing above is a rescuable
# NoMethodError. `include M1, M2' (and prepend / extend, static or with a
# receiver) make M1 the nearer ancestor. `C.send(:private, :m)' (and :protected,
# :public) changes m's visibility at run time; an implicit-self call still
# reaches it, a missing name is a NameError. /usr/bin/ruby 2.6 settles each.
module S134N; def f; "N"; end; def g(x); "Ng#{x}"; end; end
module S134M; include S134N; def f; "M>" + super; end; end
module S134O; def f; "O>" + super; end; end
module S134P; def f; "P>" + super; end; end
class S134Base; def f; "B"; end; end
class S134A < S134Base
  include S134M
  include S134O
  def f; "A>" + super; end
end
class S134K; include S134N, S134O; def f; "K>" + super; end; def g(x); "Kg>" + super(x + 1); end; end
class S134S < S134A; def f; "S>" + super; end; end
class S134Q < S134Base; prepend S134P; include S134O; def f; "Q>" + super; end; end
class S134R < S134Base; include S134M; end
class S134L; def g(a); super; end; end
class S134T < S134Base; def h(a, b = 2); super; end; end
class S134U; include S134O; end
module S134X; def w; "x"; end; end
module S134Y; def w; "y"; end; end
class S134E; extend S134X, S134Y; end
class S134F; prepend S134X, S134Y; def w; "f"; end; end
class S134G; end
S134G.extend(S134X, S134Y)
class S134V
  def x; "x"; end
  def y; "y" + x; end
  def z(o); o.w; end
  def w; "w"; end
  private
  def v; "v"; end
end
def s134nm
  yield
rescue NoMethodError => e
  e.message[0, 31]
end
def s134
  check("su134a", S134A.new.f == "A>O>M>N" && S134K.new.f == "K>N" && S134K.new.g(1) == "Kg>Ng2" &&
        S134S.new.f == "S>A>O>M>N" && S134Q.new.f == "P>Q>O>B" && S134R.new.f == "M>N")
  check("su134b", s134nm { S134L.new.g(1) } == "super: no superclass method `g'" &&
        s134nm { S134T.new.h(1) } == "super: no superclass method `h'" &&
        s134nm { S134U.new.f } == "super: no superclass method `f'")
  o = Object.new
  o.extend(S134X, S134Y)
  check("mx134c", S134K.ancestors.take(3) == [S134K, S134N, S134O] && S134E.w == "x" &&
        S134F.ancestors.take(3) == [S134X, S134Y, S134F] && S134F.new.w == "x" && S134G.w == "x" && o.w == "x")
  a = S134V.new
  r = S134V.send(:private, :x)
  ay = a.y
  px = s134nm { a.x }
  S134V.send(:protected, :w)
  pw = begin; a.w; rescue NoMethodError => e; e.message[0, 27]; end
  zw = a.z(S134V.new)
  S134V.send(:public, :v, :x)
  ne = begin; S134V.send(:private, :nope); rescue NameError => e; e.message; end
  check("vs134d", r == S134V && ay == "yx" && px == "private method `x' called for #" && zw == "w" &&
        pw == "protected method `w' called" && a.v + a.x == "vx" && S134V.public_method_defined?(:v) &&
        S134V.protected_method_defined?(:w) && ne == "undefined method `nope' for class `S134V'")
end
# ===== SECTION 135: super into a builtin method; the value of a user == =====
# A super that no user class or module above answers reaches the builtin method:
# Object#initialize (which takes no arguments, so a zsuper passing one is an
# ArgumentError), #to_s, #hash, #frozen?, #==, #!=, #dup, Kernel's
# #respond_to_missing?, Comparable#== and Exception#message. An anonymous `*'
# parameter takes any arguments. A user #== answers its own value unconverted
# (`!=' negates it). /usr/bin/ruby 2.6 settles each.
class S135A; attr_reader :a; def initialize(a); super(); @a = a; end; end
class S135B; def initialize(a); super; end; end
class S135C; def initialize(*); super(); end; def to_s; "<" + super + ">"; end; def hash; super; end; def frozen?; super; end; end
class S135D; def respond_to_missing?(m, p = false); m == :zz || super; end; def method_missing(m, *a); m == :zz ? 1 : super; end; end
class S135E; def ==(o); o.is_a?(S135E) || super; end; def dup; [super]; end; def freeze; super; end; end
class S135V; include Comparable; attr_reader :v; def initialize(v); @v = v; end; def <=>(o); v <=> o.v; end; def ==(o); super; end; end
class S135Y; def ==(o); "yes"; end; end
class S135N; def ==(o); nil; end; end
class S135Err < StandardError; def initialize(m = "dflt"); super; end; def message; "m: " + super; end; end
def s135
  b = begin; S135B.new(1); "none"; rescue ArgumentError => e; e.message; end
  c = S135C.new(1, 2)
  check("su135a", S135A.new(3).a == 3 && b == "wrong number of arguments (given 1, expected 0)" && c.class == S135C)
  d = S135D.new
  e = S135E.new
  dv = e.dup
  m = begin; raise S135Err; rescue S135Err => x; x.message; end
  check("su135b", c.to_s.start_with?("<#<S135C") && c.hash == c.hash && c.frozen? == false &&
        d.respond_to?(:zz) && !d.respond_to?(:yy) && d.zz == 1 && e == e && !(e == 1) && e != 1 &&
        dv.length == 1 && dv[0].class == S135E && e.freeze.frozen? && m == "m: dflt")
  check("cm135c", S135V.new(1) == S135V.new(1) && !(S135V.new(1) == S135V.new(2)))
  y = S135Y.new
  n = S135N.new
  check("eq135d", (y == y) == "yes" && (y != y) == false && (n == n).nil? && (n != n) == true &&
        y.send(:==, 1) == "yes" && [y].include?(1))
end
# ===== SECTION 136: run-time visibility in reflection; directives are private =====
# `C.send(:private, :m)' (and :protected, :public) moves m between the lists
# private_instance_methods / protected_instance_methods / public_instance_methods
# answer, an inherited method included. Module#private, #public, #protected and
# #module_function are private methods: a call with an explicit receiver is a
# NoMethodError, while a user class may define its own #public. /usr/bin/ruby 2.6
# settles each.
class S136P; def inh; 1; end; end
class S136C < S136P; def a; 1; end; def b; 2; end; protected def c; 3; end; private def d; 4; end; end
module S136M; def mm; end; end
class S136R; def public; "pub"; end; end
def s136nm
  yield
  "ran"
rescue NoMethodError => e
  e.message.split("\n")[0]
end
def s136
  S136C.send(:protected, :a)
  S136C.send(:public, :c, :d)
  S136C.send(:private, :inh)
  check("vr136a", S136C.protected_instance_methods(false) == [:a] && S136C.public_instance_methods(false).sort == [:b, :c, :d] &&
        S136C.private_instance_methods(false) == [:inh] && S136C.instance_methods(false).sort == [:a, :b, :c, :d])
  S136C.send(:public, :a)
  check("vr136b", S136C.protected_instance_methods(false) == [] && S136C.public_instance_methods(false).sort == [:a, :b, :c, :d])
  check("vd136c", s136nm { S136C.private(:b) } == "private method `private' called for S136C:Class" &&
        s136nm { S136M.module_function(:mm) } == "private method `module_function' called for S136M:Module" &&
        S136C.new.b == 2 && S136R.new.public == "pub")
end
# ===== SECTION 137: a class method `new'; super in a class method =====
# `def self.new' (or `new' in `class << self', or in a module the class
# extends) answers C.new and a bare `new' in a class method, a subclass
# inherits it, and its `super' is Class#new - with the arguments and the block.
# A super in any class method reaches the class method above, and a bare
# `name' in one is Module#name. /usr/bin/ruby 2.6 settles each.
class S137Pool
  @made = 0
  class << self
    attr_reader :made
    def new(*args, &blk)
      @made += 1
      super
    end
  end
  def initialize(a, b = 2)
    @s = a + b
    @s += yield if block_given?
  end
  attr_reader :s
end
class S137Sub < S137Pool
  @made = 100
end
class S137Cache
  def self.new(k)
    @c ||= {}
    @c[k] ||= super(k)
  end
  def self.make(k); new(k); end
  def initialize(k); @k = k; end
end
class S137A
  def self.f(x, y = 1); "A#{x}#{y}"; end
  def self.g; "Ag:#{name}"; end
end
class S137B < S137A
  def self.f(x, y = 2); super + "B"; end
  def self.g; super() + "Bg"; end
end
class S137C < S137B
  def self.f(x, y = 3); super(x, 9) + "C"; end
end
module S137Counted
  def new(*args)
    @count = (@count || 0) + 1
    super
  end
  def count; @count; end
end
class S137W
  extend S137Counted
  def initialize(n); @n = n; end
end
def s137
  check("sn137a", S137Pool.new(1).s == 3 && S137Pool.new(1, 5) { 10 }.s == 16 && S137Pool.made == 2)
  check("sn137b", S137Sub.new(3).s == 5 && S137Sub.made == 101 && S137Sub.new(3).class == S137Sub)
  check("sn137c", S137Cache.new(1).equal?(S137Cache.make(1)) && !S137Cache.new(1).equal?(S137Cache.new(2)))
  check("sn137d", S137B.f(1) == "A12B" && S137C.f(1) == "A19BC" && S137B.g == "Ag:S137BBg")
  S137W.new(1)
  check("sn137e", S137W.new(2).class == S137W && S137W.count == 2)
end
# ===== SECTION 138: the inherited / included / extended / prepended / method_added hooks =====
# A new subclass is passed to its superclass's (or an ancestor's) #inherited,
# already named; `include M' runs M.included(base) - every time, the last
# module of `include A, B' first - and #extended, #prepended likewise, at run
# time too (`base.extend(M)' in an #included is the ClassMethods idiom). A def
# or an attr_* after a `def self.method_added' reports each new method name.
# /usr/bin/ruby 2.6 settles each.
S138LOG = []
class S138Reg
  @subs = []
  def self.subs; @subs; end
  def self.inherited(k)
    super
    S138Reg.subs << k.name
  end
end
class S138R1 < S138Reg; end
class S138R2 < S138Reg; end
class S138R3 < S138R1; end
module S138Plug
  def self.included(base)
    S138LOG << "inc #{base}"
    base.extend(CM)
  end
  module CM
    def label; "#{name}!"; end
  end
end
module S138A; def self.included(b); S138LOG << "A #{b}"; end; end
module S138B; def self.included(b); S138LOG << "B #{b}"; end; end
module S138E
  def self.extended(o); S138LOG << "ext #{o.class}"; end
  def yo; "yo"; end
end
module S138Pre
  def self.prepended(b); S138LOG << "pre #{b}"; end
  def hi; "pre-" + super; end
end
class S138Host; include S138Plug; include S138Plug; end
class S138Y; include S138A, S138B; extend S138E; end
class S138PH; def hi; "ph"; end; prepend S138Pre; end
class S138Watch
  @log = []
  def self.log; @log; end
  def before; end
  def self.method_added(m)
    @log << m
  end
  def a; end
  attr_accessor :w
  define_method(:dm) { 1 }
  def b; end
end
class S138Watch2 < S138Watch
  @log = []
  def c; end
end
def s138
  check("hk138a", S138Reg.subs == ["S138R1", "S138R2", "S138R3"])
  check("hk138b", S138Host.label == "S138Host!" && S138PH.new.hi == "pre-ph")
  o = Object.new
  o.extend(S138E)
  check("hk138c", o.yo == "yo" && S138LOG == ["inc S138Host", "inc S138Host", "B S138Y", "A S138Y", "ext Class", "pre S138PH", "ext Object"])
  check("hk138d", S138Watch.log == [:a, :w, :w=, :dm, :b] && S138Watch2.log == [:c])
end
# ===== SECTION 139: a modifier after a bare yield; super with keywords =====
# `yield if block_given?' is the bare yield under a modifier, not a yield of an
# `if' expression, and an explicit `super(a, k: v)' passes the keyword (and a
# `*rest' its elements) through. /usr/bin/ruby 2.6 settles each.
def s139y
  r = [0]
  r << yield if block_given?
  r << (yield 2 unless !block_given?)
  return yield(3) if block_given?
  r
end
class S139P
  def initialize(a, b: 1); @v = [a, b]; end
  attr_reader :v
end
class S139K < S139P
  def initialize(a, b: 2); super(a, b: b); end
end
class S139S < S139P
  def initialize(*r); super(*r); end
end
def s139
  check("ym139a", s139y == [0, nil] && s139y { |x| x.nil? ? 7 : x * 10 } == 30)
  check("sk139b", S139K.new(1).v == [1, 2] && S139K.new(1, b: 5).v == [1, 5] && S139S.new(4, b: 6).v == [4, 6])
end

# ===== SECTION 140: private_class_method; send past visibility =====
# private_class_method :new is the Singleton idiom: a class method or a bare
# `new' inside the class reaches it, an explicit receiver outside raises, and a
# subclass inherits the restriction unless it says public_class_method. #send
# reaches a private method (instance or class) and #public_send does not; a
# class or module object answers respond_to? for the Class or Module methods; a
# reopened class or a module may restrict a class method it defined before.
# /usr/bin/ruby 2.6 settles each.
class S140L
  private_class_method :new
  def self.instance; @i ||= new; end
  def self.make(&b); new(&b); end
  def initialize(&b); @v = b ? b.call : 0; end
  attr_reader :v
end
class S140D < S140L
  def self.mk; new; end
end
class S140U < S140L
  public_class_method :new
end
class S140K
  def self.a; 1; end
  def self.b; 2; end
  private_class_method :a, :b
  public_class_method :a
  private_class_method def self.z; 7; end
  def self.go; b + z; end
end
class S140K
  def self.c; 4; end
end
class S140K
  private_class_method :c
  def self.c2; c * 10; end
end
class S140B
  def self.helper; "h"; end
end
class S140E < S140B
  private_class_method :helper
  def self.use; helper * 2; end
end
module S140M
  def self.inner; 40; end
  private_class_method :inner
  def self.outer; inner + 2; end
end
class S140V
  def pub; 2; end
  private
  def pv(x); x * 3; end
end
def s140e
  yield
  "no error"
rescue NoMethodError => e
  e.message
end
def s140
  check("sg140a", S140L.instance.equal?(S140L.instance) && S140L.make { 5 }.v == 5)
  check("sg140b", s140e { S140L.new } == "private method `new' called for S140L:Class")
  check("sg140c", S140D.mk.class == S140D && s140e { S140D.new } == "private method `new' called for S140D:Class")
  check("sg140d", S140U.new.class == S140U && S140L.send(:new).class == S140L)
  check("sg140e", S140L.respond_to?(:new) == false && S140L.respond_to?(:new, true) && S140U.respond_to?(:new))
  check("rt140m", S140B.respond_to?(:new) && S140B.respond_to?(:superclass) && S140M.respond_to?(:name) && !S140M.respond_to?(:new))
  check("pc140f", S140K.a == 1 && S140K.go == 9 && s140e { S140K.b } == "private method `b' called for S140K:Class")
  check("pc140g", s140e { S140K.z } == "private method `z' called for S140K:Class" && S140K.send(:z) == 7)
  check("pc140h", S140K.c2 == 40 && s140e { S140K.c }.start_with?("private method `c' called for S140K:Class"))
  check("pc140i", S140E.use == "hh" && S140B.helper == "h" && s140e { S140E.helper } == "private method `helper' called for S140E:Class")
  check("pc140j", S140M.outer == 42 && s140e { S140M.inner } == "private method `inner' called for S140M:Module")
  check("sd140k", S140V.new.send(:pv, 4) == 12 && S140V.new.__send__(:pv, 2) == 6 && S140V.new.public_send(:pub) == 2)
  check("sd140l", s140e { S140V.new.public_send(:pv, 1) }.start_with?("private method `pv' called for"))
end

# ===== SECTION 141: CONST = Class.new(Super) { body } =====
# Class.new answers a new class (running the superclass's #inherited) and the
# constant assignment names it; the block is its body. The block-less form is
# the usual way to declare an exception class. /usr/bin/ruby 2.6 settles each.
class S141P
  def self.inherited(k); (@seen ||= []) << k; end
  def self.seen; @seen; end
  def g; 2; end
end
S141K = Class.new(S141P) { def f; 1; end }
S141E = Class.new(StandardError)
module S141M
  Err = Class.new(ArgumentError)
  Obj = Class.new do
    define_method(:d) { 6 }
    def h; 3; end
  end
end
def s141
  check("cn141a", S141K.new.f == 1 && S141K.new.g == 2 && S141K.superclass == S141P && S141K.name == "S141K")
  check("cn141b", S141P.seen == [S141K])
  e = begin; raise S141E, "boom"; rescue StandardError => x; x; end
  check("cn141c", e.class == S141E && e.message == "boom" && S141E.superclass == StandardError)
  check("cn141d", S141M::Err.superclass == ArgumentError && S141M::Err.name == "S141M::Err")
  check("cn141e", S141M::Obj.new.d == 6 && S141M::Obj.new.h == 3 && S141M::Obj.superclass == Object)
end

# ===== SECTION 142: an exception leaving a class or module body =====
# A raise out of a class, module or `class << self' body ends that body: the
# definitions after it are written where the statement itself was, not inside
# the class whose body raised. /usr/bin/ruby 2.6 settles each.
S142F = [true]
S142R = []
begin
  class S142G
    raise ArgumentError, "in body" if S142F[0]
  end
rescue ArgumentError => e
  S142R << e.message
end
class S142O; end
begin
  module S142M
    module In
      raise "x" if S142F[0]
    end
  end
rescue => e
  S142R << e.message
end
module S142Mo; end
class S142H
  begin
    class << self
      raise "y" if S142F[0]
    end
  rescue => e
    S142R << e.message
  end
  def self.k; 4; end
end
class S142P; end
# A constant, class or module written in a block or a begin clause is bound in
# the class body or top level around it (its cref), not in the block.
begin
  S142K1 = 1
  class S142B1; end
rescue
end
[2].each { |v| S142K2 = v }
1.times { module S142B2; end }
class S142Q
  [3].each { |v| K3 = v }
  begin
    K4 = 4
    class In; end
  rescue
  end
  1.times { class In2; end }
end
def s142
  check("cb142a", S142O.name == "S142O" && S142Mo.name == "S142Mo" && S142P.new.inspect.start_with?("#<S142P") &&
        S142R == ["in body", "x", "y"] && S142H.k == 4 && S142M::In.name == "S142M::In")
  check("cb142b", S142K1 == 1 && S142B1.name == "S142B1" && S142K2 == 2 && S142B2.name == "S142B2" &&
        S142Q::K3 == 3 && S142Q::K4 == 4 && S142Q::In.name == "S142Q::In" && S142Q::In2.name == "S142Q::In2" &&
        defined?(K3).nil?)
end

# ===== SECTION 143: `local !~ /re/` =====
# `!~` after a name is always the operator, a local's or a method's: it never
# opens the argument of a paren-less call. /usr/bin/ruby 2.6 settles each.
def s143m(*a); a.empty? ? "abc" : a; end
def s143
  s = ["abc"][0]
  z = s !~/z/
  w = s !~ /b/
  m = [s].map { |t| t !~ /q/ }
  check("nm143a", z == true && w == false && m == [true] && (s !~ /a/) == false && (s143m !~ /z/) == true)
end

# ===== SECTION 144: tr / count / delete / squeeze and regexp positions on astral text =====
# A character outside the Basic Multilingual Plane is ONE character to the
# character-set methods, to a regexp (`.` matches it whole) and to every
# position a match reports. /usr/bin/ruby 2.6 settles each.
def s144
  t = ["x😀😀y"][0]
  check("as144a", t.tr("😀", "o") == "xooy" && t.tr("x", "😀") == "😀😀😀y" && t.tr("^😀", "-") == "-😀😀-" &&
        t.tr_s("😀", "o") == "xoy" && "aé😀".tr("a-z", "A-Z") == "Aé😀")
  check("as144b", t.count("😀") == 2 && t.count("^😀") == 2 && t.delete("😀") == "xy" && t.delete("^😀") == "😀😀" &&
        t.squeeze == "x😀y" && t.squeeze("😀") == "x😀y" && "aé😀".count("é😀") == 2)
  m = "😀a😀".match(/a(😀)/)
  check("as144c", t.scan(/./).size == 4 && (t =~ /y/) == 3 && t.match(/y/).begin(0) == 3 && m.begin(0) == 1 &&
        m.end(1) == 3 && m.pre_match == "😀" && "x😀y".scan(//) == ["", "", "", ""] && "x😀y".gsub(//, "-") == "-x-😀-y-" &&
        "a😀b".sub(/[^a]/, "Z") == "aZb")
end

# ===== SECTION 145: a key put back during Hash#each =====
# A Hash#each block that deletes a key and stores it again ADDS a key: MRI raises
# RuntimeError and the hash ends without it. Writing an existing key is not an
# addition. /usr/bin/ruby 2.6 settles each.
def s145
  h = {a: 1, b: 2}
  msg = nil
  begin
    h.each { |k, v| h.delete(:a); h[:a] = 3 }
  rescue RuntimeError => e
    msg = e.message
  end
  check("he145a", msg == "can't add a new key into hash during iteration" && h == {b: 2})
  g = {a: 1, b: 2, c: 3}
  msg = nil
  begin
    g.each { |k, v| g.delete(:b) if k == :a; g[:b] = 9 if k == :c }
  rescue RuntimeError => e
    msg = e.message
  end
  w = {a: 1, b: 2}
  w.each { |k, v| w[k] = v * 10 }
  check("he145b", msg == "can't add a new key into hash during iteration" && g == {a: 1, c: 3} && w == {a: 10, b: 20})
end

# ===== SECTION 146: assignment targets rooted at a parenthesised call =====
# `f(x)[i] = v', `f(x).a = v' and a step with arguments (`o.get(0)[0] = v') assign
# through what the call answers; `recv&.a = v' on nil writes nothing.
# /usr/bin/ruby 2.6 settles each.
class S146O
  attr_accessor :f, :g
  def initialize; @f = 1; @g = [0, 0]; end
  def me(n); self; end
end
S146A = [[1, 2], [3, 4]]
S146OS = [S146O.new, S146O.new]
def s146arr(i); S146A[i]; end
def s146obj(i); S146OS[i]; end
def s146nil(i); nil; end
def s146
  s146arr(0)[1] = 5
  s146obj(1).f = 7
  s146arr(0)[0] += 1
  s146obj(1).f += 1
  s146arr(1)[0] ||= 9
  x = s146arr(1)[1] = 8
  check("ct146a", S146A == [[2, 5], [3, 8]] && S146OS[1].f == 8 && x == 8)
  S146OS[0].me(1).f = 11
  S146OS[0].me(1).me(2).g[1] = 5
  s146obj(1).g[1] ||= 7
  hits = 0
  s146nil(0)&.f = (hits += 1)
  s146obj(0)&.f -= 1
  check("ct146b", S146OS[0].f == 10 && S146OS[0].g == [0, 5] && S146OS[1].g == [0, 0] && hits == 0)
end

# ===== SECTION 147: class_eval on a builtin class, succ past a length's end, endless slices =====
# A def in a class_eval block of String / Integer / Array is callable on their
# values; String#succ of a string without alphanumerics wraps a character that is
# the largest of its UTF-8 length and carries; each_slice / each_cons on an endless
# range pull one value at a time. /usr/bin/ruby 2.6 settles each.
def s147
  String.class_eval { def s147shout; upcase + "!"; end }
  Integer.class_eval { def s147dbl; self * 2; end }
  Array.class_eval { def s147sec; self[1]; end }
  check("ce147a", "hi".s147shout == "HI!" && 4.s147dbl == 8 && [1, 2].s147sec == 2)
  check("su147b", "\x7f".succ.bytes == [1, 0] && "-\x7f".succ.bytes == [46, 0] && "\x7f\x7f".succ.bytes == [1, 0, 0] &&
                  "߿".succ.codepoints == [1, 128] && "￿".succ.codepoints == [1, 2048] &&
                  "퟿".succ.codepoints == [57344] && "a\x7f".succ.bytes == [98, 127])
  e = (1..).each_cons(2)
  check("es147c", (1..).each_slice(2).first(2) == [[1, 2], [3, 4]] && (1..).each_cons(3).first(2) == [[1, 2, 3], [2, 3, 4]] &&
                  (1..).each_slice(2).inspect == "#<Enumerator: 1..:each_slice(2)>" && e.next == [1, 2] && e.next == [2, 3] &&
                  (1..).lazy.take_while { |x| x < 6 }.each_slice(2).to_a == [[1, 2], [3, 4], [5]] &&
                  (1..).lazy.select(&:even?).each_slice(2).first(2) == [[2, 4], [6, 8]] &&
                  (1..).each_slice(2).lazy.map { |a| a.sum }.first(3) == [3, 7, 11])
end

s122() # SECTION-CALL 122
s123() # SECTION-CALL 123
s124() # SECTION-CALL 124
s125() # SECTION-CALL 125
s126() # SECTION-CALL 126
s127() # SECTION-CALL 127
s128() # SECTION-CALL 128
s129() # SECTION-CALL 129
s130() # SECTION-CALL 130
s131() # SECTION-CALL 131
s132() # SECTION-CALL 132
s133() # SECTION-CALL 133
s134() # SECTION-CALL 134
s135() # SECTION-CALL 135
s136() # SECTION-CALL 136
s137() # SECTION-CALL 137
s138() # SECTION-CALL 138
s139() # SECTION-CALL 139
s140() # SECTION-CALL 140
s141() # SECTION-CALL 141
s142() # SECTION-CALL 142
s143() # SECTION-CALL 143
s144() # SECTION-CALL 144
s145() # SECTION-CALL 145
s146() # SECTION-CALL 146
s147() # SECTION-CALL 147
puts "full: #{FULLC[0]} checks, #{FULLC[1]} failures"
exit(FULLC[1])
