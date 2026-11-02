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

# ===== SECTION 148: constants below a class body's top level, $~ after split, =~ on any object, a reopened builtin's methods =====
# A constant assigned inside begin / rescue, an if or a block of a class or module
# body belongs to that class: its methods read it and Module#constants lists it,
# unless the assignment never ran. String#split with a pattern leaves $~ at its
# final search (nil once a search fails); a separator String leaves it alone.
# `obj =~ re' is nil for an object that is neither a String nor a Symbol
# (Object#=~ in 2.6) and Regexp#=~ raises TypeError on one; a class may define its
# own =~. A builtin class reopened by a body lists its own methods and the new one.
# /usr/bin/ruby 2.6 settles each.
class S148K
  begin
    S148KC = 3
  rescue
  end
  if true then S148KD = 4 end
  [1].each { S148KE = 5 }
  if false then S148KG = 6 end
  begin
    raise "x"
    S148KH = 7
  rescue
  end
  begin
    class S148In; S148KJ = 8; end
  rescue
  end
  def g; [S148KC, S148KD, S148KE, S148In::S148KJ]; end
end
module S148M
  begin
    S148MC = 9
  ensure
    S148MZ = 1
  end
  def self.h; S148MC + S148MZ; end
end
class S148Q
  def =~(o); 42; end
end
class String
  def s148shout; upcase + "!"; end
end
def s148
  check("cb148a", S148K.new.g == [3, 4, 5, 8] && S148K.constants.sort == [:S148In, :S148KC, :S148KD, :S148KE] &&
                  S148M.h == 10 && S148M.constants.sort == [:S148MC, :S148MZ] && S148K::S148In.constants == [:S148KJ])
  "xay" =~ /a/
  "a,b".split(/,/)
  m1 = $~
  "xay" =~ /a/
  "a,b,c".split(/,/, 2)
  m2 = $~
  "xay" =~ /a/
  "a,b".split(",")
  m3 = $~
  "xay" =~ /a/
  "a,b".split(/,/, 1)
  m4 = $~
  check("sp148b", m1.nil? && m2[0] == "," && m3[0] == "a" && m4[0] == "a")
  vs = [15, 1.5, [1], nil, :a5, "a5"]
  rs = vs.map { |v| v =~ /5/ }
  "xay" =~ /a/
  t = (15 =~ /5/)
  m5 = $~
  e1 = nil
  begin
    /5/ =~ 15
  rescue TypeError => e
    e1 = e.message
  end
  check("rx148c", rs == [nil, nil, nil, nil, 1, 1] && t.nil? && m5[0] == "a" && (15 !~ /5/) == true &&
                  e1 == "no implicit conversion of Integer into String" && (S148Q.new =~ /x/) == 42)
  own = String.instance_methods(false)
  check("im148d", own.include?(:s148shout) && own.include?(:upcase) && own.include?(:split) && !own.include?(:frozen?) &&
                  String.instance_methods.include?(:s148shout) && String.method_defined?(:s148shout) &&
                  Integer.instance_methods(false).include?(:even?) && !Integer.instance_methods(false).include?(:upcase))
end

# ===== SECTION 149: a new key while a Hash iteration runs =====
# While any block-taking Hash method (each, each_key, select, map, count, ...) or
# an Enumerator over the Hash runs its block, storing a key the Hash does not
# hold raises RuntimeError AT the store, so the rest of the block never runs;
# writing an existing key, deleting, and Hash#sort's comparator are allowed. A
# break or raise out of the block ends the iteration. /usr/bin/ruby 2.6 settles
# each.
def s149try
  yield
  nil
rescue RuntimeError => e
  e.message
end
def s149
  ms = "can't add a new key into hash during iteration"
  h = {a: 1, b: 2}
  e1 = s149try { h.each { h.delete(:b); h[:b] = 3 } }
  ran = false
  g = {a: 1}
  e2 = s149try { g.each { g[:c] = 9; ran = true } }
  check("hi149a", e1 == ms && h == {a: 1} && e2 == ms && !ran && g == {a: 1})
  rs = [:each_key, :each_value, :select, :map, :count, :sort_by, :each_with_index, :delete_if].map do |m|
    w = {a: 1}
    [s149try { w.send(m) { w[:z] = 5; 0 } }, w]
  end
  check("hi149b", rs.all? { |r| r == [ms, {a: 1}] })
  x = {a: 1, b: 2}
  x.each { |k, v| x[k] = v + 1 }
  x.each { break }
  x[:c] = 0
  s149try { x.each { raise "out" } }
  x[:d] = 0
  x.sort { |p, q| x[:e] = 1; 0 }
  y = {a: 1}
  e3 = s149try { y.each { y.each { }; y[:n] = 1 } }
  ran = false
  e4 = s149try { y.each { y.merge!(q: 1); ran = true } }
  y.each { y.update(a: 2) }
  check("hi149c", x == {a: 2, b: 3, c: 0, d: 0, e: 1} && e3 == ms && e4 == ms && !ran && y == {a: 2})
  z = {a: 1}
  e5 = s149try { z.each_with_index.map { z[:q] = 1 } }
  e6 = s149try { z.each.each { z[:q] = 1 } }
  d = Hash.new { |hh, k| hh[k] = 0 }
  d[:a] = 1
  e7 = s149try { d.each { d[:m] } }
  check("hi149d", e5 == ms && e6 == ms && z == {a: 1} && e7 == ms && d == {a: 1})
end

# ===== SECTION 150: for over a Hash; Hash#default= / #default_proc=; more Hash methods =====
# `for k, v in h` is h.each: the live [key, value] pairs, a deleted entry skipped,
# a new key refused while the loop runs and allowed again however it was left.
# default= and default_proc= each replace the other. store, fetch_values, shift,
# replace (which takes the default too), rehash, transform_values! /
# transform_keys! (2.6: the Hash is emptied and refilled, so a collision keeps
# the first place and the last value), find_all / cycle / chunk_while / lazy over
# the pairs, and zip with a block, which yields each tuple and answers nil.
# /usr/bin/ruby 2.6 settles each.
def s150try
  yield
  nil
rescue RuntimeError => e
  e.message
end
def s150ret(h)
  for k, v in h
    return k
  end
end
def s150
  ms = "can't add a new key into hash during iteration"
  h = {a: 1, b: 2, c: 3}
  seen = []
  for k, v in h
    h.delete(:b) if k == :a
    h[:c] = 30 if k == :a
    seen << [k, v]
  end
  whole = []
  for kv in {x: 1}
    whole << kv
  end
  check("fh150a", seen == [[:a, 1], [:c, 30]] && whole == [[:x, 1]] && k == :c && v == 30)
  g = {a: 1}
  e1 = s150try { for k in g; g[:n] = 1; end }
  for k in g
    break
  end
  g[:b] = 2
  s150try { for k in g; raise "out"; end }
  g[:c] = 3
  r = s150ret(g)
  g[:d] = 4
  check("fh150b", e1 == ms && r == :a && g == {a: 1, b: 2, c: 3, d: 4})
  q = {a: 1}
  q.default = 5
  d1 = [q[:zz], q.default, q.default_proc]
  q.default_proc = proc { |hh, kk| hh[kk] = kk.to_s * 2 }
  d2 = [q[:ab], q.default, q.default_proc.class, q]
  q.default_proc = nil
  d3 = [q[:nn], q.default, q.default_proc]
  w = Hash.new { |hh, kk| kk }
  w.default = 9
  check("fh150c", d1 == [5, 5, nil] && d2 == ["abab", nil, Proc, {a: 1, ab: "abab"}] && d3 == [nil, nil, nil] &&
        w.default_proc.nil? && w[:x] == 9 && q.send(:default=, 4) == 4 && q[:m] == 4)
  s = {a: 1}
  sk = "key"
  st = [s.store(:b, 2), s.store(sk, 3)]
  sk << "x"
  fv = [s.fetch_values(:a, :b), s.fetch_values(:a, :zz) { |kk| kk.to_s }]
  fe = begin; s.fetch_values(:zz); rescue KeyError => err; err.message; end
  check("fh150d", st == [2, 3] && s == {a: 1, b: 2, "key" => 3} && fv == [[1, 2], [1, "zz"]] && fe == "key not found: :zz" &&
        s150try { s.each { s.store(:new, 1) } } == ms)
  t = {a: 1, b: 2}
  sh = [t.shift, t, {}.shift]
  dflt = Hash.new(7)
  dflt[:a] = 1
  rp = {z: 0}
  rp2 = rp.replace(dflt)
  rk = [[1], [2]]
  rh = {rk[0] => :one, rk[1] => :two}
  rk[1][0] = 1
  rh.rehash
  check("fh150e", sh == [[:a, 1], {b: 2}, nil] && rp2.equal?(rp) && rp == {a: 1} && rp[:q] == 7 &&
        rh == {[1] => :two} && rh.size == 1)
  tv = {a: 1, b: 2}
  tv2 = tv.transform_values! { |x| x * 10 }
  tk = {a: 1, b: 2}
  tk.transform_keys! { |kk| kk == :a ? :b : :a }
  tc = {a: 1, b: 2}.transform_keys! { |kk| :same }
  check("fh150f", tv2.equal?(tv) && tv == {a: 10, b: 20} && tk == {b: 1, a: 2} && tc == {same: 2})
  e = {a: 1, b: 2, c: 4}
  cy = []
  zy = []
  zr = e.zip([7, 8]) { |row| zy << row }
  check("fh150g", e.find_all { |kk, x| x > 1 } == [[:b, 2], [:c, 4]] && e.cycle.first(4) == [[:a, 1], [:b, 2], [:c, 4], [:a, 1]] &&
        e.cycle(2) { |kk, x| cy << kk }.nil? && cy == [:a, :b, :c, :a, :b, :c] &&
        e.chunk_while { |p1, p2| p2[1] == p1[1] + 1 }.to_a == [[[:a, 1], [:b, 2]], [[:c, 4]]] &&
        e.lazy.map { |kk, x| x }.first(2) == [1, 2] && e.lazy.inspect == "#<Enumerator::Lazy: {:a=>1, :b=>2, :c=>4}>" &&
        zr.nil? && zy == [[[:a, 1], 7], [[:b, 2], 8], [[:c, 4], nil]] && [1, 2].zip([3]) { } .nil? &&
        [1, 2, 3].find_all(&:odd?) == [1, 3])
end

# ===== SECTION 151: for-loop targets and value; compare_by_identity; default_proc=; Lazy =====
# A for loop assigns each element as a multiple assignment does - nested groups, a
# splat, a trailing comma - and its value is the collection (or a break's value).
# compare_by_identity keys a Hash by equal?, so two equal Arrays are two keys (a
# String is the exception here: it has no identity in this value model). default_proc=
# takes a Proc or a Symbol and raises TypeError otherwise; #lazy answers an
# Enumerator::Lazy < Enumerator. /usr/bin/ruby 2.6 settles each.
def s151
  out = []
  for a, (b, c) in [[1, [2, 3]], [4, [5, 6, 7]], [8, 9]]
    out << [a, b, c]
  end
  check("fd151a", out == [[1, 2, 3], [4, 5, 6], [8, 9, nil]])
  out = []
  for a, *r in [[1, 2, 3], [4]]; out << [a, r]; end
  for i, in [[1, 2], [3, 4]]; out << i; end
  for (p1, (p2, *p3)), p4 in [[[1, [2, 3, 4]], 5]]; out << [p1, p2, p3, p4]; end
  for *h, t in [[1, 2, 3]]; out << [h, t]; end
  check("fd151b", out == [[1, [2, 3]], [4, []], 1, 3, [1, 2, [3, 4], 5], [[1, 2], 3]])
  for q1, (q2, q3) in []; end
  check("fd151c", [q1, q2, q3] == [nil, nil, nil])
  src = [1, 2]
  rng = (1..3)
  hs = {a: 1}
  v1 = for e in src do end
  v2 = for e in rng do end
  v3 = for k, v in hs do end
  v4 = for e in [] do end
  v5 = for e in [1, 2, 3] do break e * 10 if e == 2 end
  v6 = for e in [1, 2, 3] do break if e == 2 end
  check("fd151d", v1.equal?(src) && v2 == (1..3) && v3.equal?(hs) && v4 == [] && v5 == 20 && v6.nil?)
  ka = [1]
  kb = [1]
  ih = {}
  check("cbi151a", !ih.compare_by_identity? && ih.compare_by_identity.equal?(ih) && ih.compare_by_identity?)
  ih[ka] = :a
  ih[kb] = :b
  ih[:s] = 1
  ih[:s] = 2
  ih[2**70] = 3
  check("cbi151b", ih.size == 4 && ih[ka] == :a && ih[kb] == :b && ih[[1]].nil? && ih[:s] == 2 && ih[2**70].nil? &&
        ih.key?(kb) && !ih.key?([1]) && ih.fetch([1], :no) == :no && ih.delete([1]).nil? && ih.delete(ka) == :a)
  check("cbi151c", ih.dup.compare_by_identity? && ih.merge({}).compare_by_identity? &&
        !ih.select { true }.compare_by_identity? && !ih.transform_values { 1 }.compare_by_identity? &&
        {}.replace(ih).compare_by_identity? && !ih.dup.replace({}).compare_by_identity?)
  vh = {[1] => 1, [1].dup => 2}
  vh.compare_by_identity
  check("cbi151d", vh.size == 1 && vh[[1]].nil? && vh.keys == [[1]])
  bad = [5, "s", 1.5, true].map do |x|
    dh = {}
    dh.default = 7
    begin
      dh.default_proc = x
      :ok
    rescue TypeError => err
      [err.message, dh.default]
    end
  end
  check("dp151a", bad == [["wrong default_proc type Integer (expected Proc)", 7],
                          ["wrong default_proc type String (expected Proc)", 7],
                          ["wrong default_proc type Float (expected Proc)", 7],
                          ["wrong default_proc type TrueClass (expected Proc)", 7]])
  sh = {}
  sh.default = 7
  sh.default_proc = :to_s
  ph = {}
  ph.default_proc = proc { |hh, kk| kk * 2 }
  ph.default_proc = nil
  check("dp151b", sh.default_proc.class == Proc && sh.default.nil? && ph.default_proc.nil? && ph[3].nil?)
  lz = [1, 2].lazy
  check("lz151a", lz.class == Enumerator::Lazy && lz.class.name == "Enumerator::Lazy" && lz.is_a?(Enumerator) &&
        lz.is_a?(Enumerator::Lazy) && Enumerator::Lazy === lz && (1..3).lazy.class == Enumerator::Lazy &&
        lz.map { |x| x }.class == Enumerator::Lazy && Enumerator::Lazy.superclass == Enumerator &&
        [1].each.class == Enumerator && !([1].each.is_a?(Enumerator::Lazy)) && lz.force.class == Array)
end

# ===== SECTION 152: is_a? / instance_of? on builtins; Hash#to_proc; default_proc= arity =====
# An Enumerator (a lazy one too) is Enumerable, a Symbol is Comparable, and
# instance_of? is identity with #class - never is_a? - so `1.instance_of?(Numeric)'
# and `[1].lazy.instance_of?(Enumerator)' are false. Hash#to_proc takes exactly one
# argument and answers h[arg] (the default included), and is what `&hash' passes.
# Hash#default_proc= rejects a lambda that cannot take (hash, key). /usr/bin/ruby
# 2.6 settles each.
def s152m1(a); end
def s152m2(a, b); a.to_s + b.to_s; end
def s152
  vals = [1, 1.5, "s", :a, [1], {1 => 2}, (1..2), [1].each, [1].lazy, nil, true, 2**70]
  clss = [Integer, Numeric, Comparable, Enumerable, Object, Kernel, Enumerator, Array, Hash, Range, String, Float]
  isa = vals.map { |v| clss.map { |c| v.is_a?(c) ? 1 : 0 }.join }
  iof = vals.map { |v| clss.map { |c| v.instance_of?(c) ? 1 : 0 }.join }
  ceq = vals.map { |v| clss.map { |c| (c === v) ? 1 : 0 }.join }
  want = ["111011000000", "011011000001", "001011000010", "001011000000", "000111010000", "000111001000",
          "000111000100", "000111100000", "000111100000", "000011000000", "000011000000", "111011000000"]
  check("ia152a", isa == want && ceq == want)
  check("ia152b", iof == ["100000000000", "000000000001", "000000000010", "000000000000", "000000010000",
                          "000000001000", "000000000100", "000000100000", "000000000000", "000000000000",
                          "000000000000", "100000000000"])
  check("ia152c", [1].cycle.is_a?(Enumerable) && [1, 2].lazy.kind_of?(Enumerable) && Enumerable === [1].each_slice(1) &&
        [1].lazy.instance_of?(Enumerator::Lazy) && !(1..2).instance_of?(Enumerable) && [1].each.instance_of?(Enumerator))
  hp = {1 => 2, 3 => 4}.to_proc
  check("hp152a", hp.call(3) == 4 && hp[1] == 2 && hp.call(5).nil? && hp.yield(3) == 4 && hp.class == Proc)
  errs = [[], [1, 2]].map do |a|
    begin
      hp.call(*a)
    rescue ArgumentError => err
      err.message
    end
  end
  check("hp152b", errs == ["wrong number of arguments (given 0, expected 1)", "wrong number of arguments (given 2, expected 1)"])
  dh = Hash.new(9)
  dh[1] = 0
  check("hp152c", [1, 2, 3].map(&{1 => :x}) == [:x, nil, nil] && [3, 1].select(&{1 => true}) == [1] &&
        [1, 5].map(&dh) == [0, 9] && dh.to_proc.call(7) == 9)
  bad = [lambda { |x| }, lambda { |x, y, z| }, ->() { }, lambda { |a, b, c, *d| }, method(:s152m1)].map do |l|
    hh = {}
    hh.default = 5
    begin
      hh.default_proc = l
      :ok
    rescue TypeError => err
      [err.message, hh.default]
    end
  end
  check("dp152a", bad == [["default_proc takes two arguments (2 for 1)", 5], ["default_proc takes two arguments (2 for 3)", 5],
                          ["default_proc takes two arguments (2 for 0)", 5], ["default_proc takes two arguments (2 for 3)", 5],
                          ["default_proc takes two arguments (2 for 1)", 5]])
  good = [lambda { |x, y| 7 }, lambda { |*x| 8 }, lambda { |x, y = 1| 9 }, proc { |x| 10 }, ->(a, *b) { 11 },
          lambda { |a, b, *c| 12 }, method(:s152m2)].map do |l|
    hh = {}
    hh.default_proc = l
    hh[3]
  end
  check("dp152b", good == [7, 8, 9, 10, 11, 12, "{}3"])
end
# ===== SECTION 153: the value of ===; the modules a builtin class includes =====
# Proc#=== / Method#=== and a user #=== answer the call's own result, not a
# boolean; Array#=== is ==, not membership (a `when *list' splat is). Struct,
# Array, Hash, Range and Enumerator include Enumerable, String, Symbol and Numeric
# Comparable - in #ancestors, #include?, the module relations and is_a?, and a
# Struct.new instance is_a? Struct. /usr/bin/ruby 2.6 settles each.
S153 = Struct.new(:a)
class P153 < Struct.new(:x); end
class X153
  def ===(o); o * 5; end
end
def s153m(x); x * 3; end
def s153
  pr = proc { |x| x * 2 }
  check("ceq153a", [pr === 3, {a: 1}.to_proc === :a, :upcase.to_proc === "ab", method(:s153m) === 4,
                    proc { nil } === 1, X153.new === 2] == [6, 1, "AB", 12, nil, 10])
  big = ->(x) { x > 2 }
  hits = [1, 5].map do |v|
    case v
    when big then :big
    else :small
    end
  end
  wx = case 3
       when X153.new then :hit
       end
  ws = case 7
       when 1, *[5, 7] then :splat
       end
  check("ceq153b", hits == [:small, :big] && wx == :hit && ws == :splat && ([1, 2] === 2) == false && [1, 2] === [1, 2])
  check("mix153a", S153.ancestors == [S153, Struct, Enumerable, Object, Kernel, BasicObject] &&
        [Array, Hash, Range, Enumerator, Struct].all? { |c| c.ancestors == [c, Enumerable, Object, Kernel, BasicObject] } &&
        [String, Symbol].all? { |c| c.ancestors == [c, Comparable, Object, Kernel, BasicObject] } &&
        Numeric.ancestors == [Numeric, Comparable, Object, Kernel, BasicObject])
  check("mix153b", [S153.include?(Enumerable), Struct.include?(Enumerable), S153 < Enumerable, Struct <= Enumerable,
                    Enumerable > Struct, Array.include?(Enumerable), Hash < Enumerable, String.include?(Comparable),
                    String < Enumerable, Array.include?(Comparable)] == [true, true, true, true, true, true, true, true, nil, false])
  s = S153.new(1)
  check("mix153c", [s.is_a?(Enumerable), s.kind_of?(Enumerable), Enumerable === s, P153.new(1).is_a?(Enumerable),
                    P153.new(1).is_a?(Struct), s.is_a?(Struct), Struct === s, s.is_a?(Comparable), P153.ancestors.size,
                    (case s when Enumerable then :enum end)] == [true, true, true, true, true, true, true, false, 7, :enum])
end
# ===== SECTION 154: Enumerable over a user #each; Numeric in the numeric classes =====
# A class that includes Enumerable and defines #each answers Enumerable's methods
# over what #each yields, a Struct.new instance too, and respond_to? says so; a
# class with #each that does not include it does not. each_with_index and
# reverse_each with a block answer their receiver: the object, a Range, a Hash.
# Integer, Float, Rational and Complex are Numerics - superclass, ancestors, the
# module relations - and a method a program gives Numeric reaches them.
# /usr/bin/ruby 2.6 settles each.
class B154
  include Enumerable
  def initialize(a); @a = a; end
  def each
    @a.each { |x| yield x }
    self
  end
end
class E154
  def each; yield 1; end
end
class Numeric
  def s154_half; self / 2.0; end
end
def s154
  b = B154.new([3, 1, 2])
  r = begin
    [b.map { |x| x * 2 }, b.sort, b.min, b.max, b.include?(2), b.to_a, b.select { |x| x > 1 },
     b.reject { |x| x > 1 }, b.inject(:+), b.sort_by { |x| -x }, b.first, b.first(2), b.count,
     b.min_by { |x| -x }, b.group_by(&:odd?), b.partition { |x| x > 1 }, b.each_slice(2).to_a,
     b.sum { |x| x * 10 }, b.minmax, b.find_index(2), b.to_h { |x| [x, -x] }, b.entries]
  rescue NoMethodError
    nil
  end
  check("enm154a", r == [[6, 2, 4], [1, 2, 3], 1, 3, true, [3, 1, 2], [3, 2], [1], 6, [3, 2, 1], 3, [3, 1], 3,
                         3, {true => [3, 1], false => [2]}, [[3, 2], [1]], [[3, 1], [2]], 60, [1, 3], 2,
                         {3 => -3, 1 => -1, 2 => -2}, [3, 1, 2]])
  st = Struct.new(:a, :b).new(4, 5)
  w = begin
    [st.map { |v| v * 2 }, st.sort.reverse, st.max, b.respond_to?(:map), b.respond_to?(:size),
     b.each_with_index { |x, i| x }.equal?(b), b.reverse_each { |x| x }.equal?(b),
     (begin; E154.new.map { |x| x }; rescue NoMethodError; :nm; end)]
  rescue NoMethodError
    nil
  end
  check("enm154b", w == [[8, 10], [5, 4], 5, true, false, true, true, :nm])
  check("enm154c", (1..3).each_with_index { |x, i| x } == (1..3) && (1..3).reverse_each { |x| x } == (1..3) &&
        {a: 1}.each_with_index { |x, i| x } == {a: 1} && [1, 2].each_with_index { |x, i| x } == [1, 2])
  check("num154a", [Integer.superclass, Float.superclass, Rational.superclass, Complex.superclass,
                    Numeric.superclass, 3.class.superclass.superclass] == [Numeric, Numeric, Numeric, Numeric, Object, Object])
  check("num154b", Float.ancestors == [Float, Numeric, Comparable, Object, Kernel, BasicObject] &&
        Integer.ancestors == [Integer, Numeric, Comparable, Object, Kernel, BasicObject] &&
        [Integer < Numeric, Integer < Comparable, Numeric > Float, Float < Integer, Integer.include?(Comparable),
         Integer.ancestors.include?(Numeric)] == [true, true, true, nil, true, true])
  h = begin
    [3.s154_half, 2.5.s154_half, 3.respond_to?(:s154_half)]
  rescue NoMethodError
    nil
  end
  check("num154c", h == [1.5, 1.25, true])
end
# ===== SECTION 155: Enumerable's grep / chunk / slice / chain; multi-value yields; Enumerable Methods =====
# Array, Range, Hash and an includer of Enumerable answer member?, grep, grep_v,
# minmax_by, slice_before / slice_after, chunk and chain; each_slice / each_cons
# with a block answer nil (MRI 2.6); zip takes any value that responds to each.
# A `yield a, b` reaches the block of map / flat_map / count / find_index /
# take_while / uniq / all? as two values and the block of select / reject /
# partition as one Array. An includer without #each fails on each; a Method or
# UnboundMethod of an Enumerable method is owned by `C(Enumerable)'.
class P155
  include Enumerable
  def each
    yield 1, 2
    yield 3, 4
    yield
  end
end
class B155
  include Enumerable
  def initialize(*a); @a = a; end
  def each
    @a.each { |x| yield x }
    self
  end
end
class N155
  include Enumerable
end
def s155
  a = [3, 1, 2, 5]
  r = begin
    [a.member?(2), a.member?(7), a.grep(1..2), a.grep(1..2) { |x| x * 10 }, a.grep_v(1..2),
     %w[ab cd ax].grep(/a/), [1, "a", :b, 2.0].grep(Integer), a.minmax_by { |x| -x }, [].minmax_by { |x| x },
     [[1, :a], [1, :b], [2, :c], [2, :d]].minmax_by { |x| x[0] },
     a.slice_before { |x| x.even? }.to_a, a.slice_before(2).to_a, a.slice_after { |x| x.even? }.to_a,
     a.slice_after(1).to_a, a.chunk { |x| x.odd? }.to_a, a.chain([9]).to_a, a.chain([9], 7..8).to_a]
  rescue NoMethodError
    nil
  end
  check("ary155a", r == [true, false, [1, 2], [10, 20], [3, 5], ["ab", "ax"], [1], [5, 1], [nil, nil], [[1, :a], [2, :c]],
                         [[3, 1], [2, 5]], [[3, 1], [2, 5]], [[3, 1, 2], [5]], [[3, 1], [2, 5]],
                         [[true, [3, 1]], [false, [2]], [true, [5]]], [3, 1, 2, 5, 9], [3, 1, 2, 5, 9, 7, 8]])
  c = begin
    [[3, 1, 4].chunk { |v| v == 1 ? :_alone : true }.to_a, [3, 1, 4].chunk { |v| v == 1 ? :_separator : v.odd? }.to_a,
     [3, 1, 4].chunk { |v| v == 1 ? nil : true }.to_a, (begin; [1].chunk { :_x }.to_a; rescue RuntimeError => e; e.message; end),
     (1..4).grep(2..3), (1..4).minmax_by { |x| -x }, (1..4).slice_before(&:odd?).to_a, {a: 1, b: 2}.grep(Array),
     {a: 1}.member?(:a), [3, 1].chunk { |v| v }.class, [1, 2].chain([3]).map { |x| x * 2 }, [1, 2].chain([3]).size,
     [1].respond_to?(:grep_v), (1..2).respond_to?(:chain), {}.respond_to?(:slice_after)]
  rescue NoMethodError
    nil
  end
  check("ary155b", c == [[[true, [3]], [:_alone, [1]], [true, [4]]], [[true, [3]], [false, [4]]], [[true, [3]], [true, [4]]],
                         "symbols beginning with an underscore are reserved", [2, 3], [4, 1], [[1, 2], [3, 4]],
                         [[:a, 1], [:b, 2]], true, Enumerator, [2, 4, 6], 3, true, true, true])
  check("ary155c", [1].each_slice(2) {}.nil? && (1..3).each_slice(2) {}.nil? && {a: 1}.each_slice(2) {}.nil? &&
                   [1, 2].each_cons(1) {}.nil? && (1..3).each_cons(2) {}.nil?)
  b = B155.new(3, 1, 2)
  z = begin
    [b.zip(b), b.zip([4, 5, 6], b), [1, 2].zip(3..4), [1, 2].zip({a: 1}), [1, 2].zip([5].each), [1, 2].zip(b),
     (begin; [1].zip(7); rescue TypeError => e; e.message; end), b.member?(2), b.grep(1..2), b.minmax_by { |x| -x },
     b.chunk { |x| x.odd? }.to_a, b.chain([9]).to_a, b.slice_before { |x| x == 1 }.to_a, b.each_slice(2) {}]
  rescue NoMethodError
    nil
  end
  check("enm155a", z == [[[3, 3], [1, 1], [2, 2]], [[3, 4, 3], [1, 5, 1], [2, 6, 2]], [[1, 3], [2, 4]], [[1, [:a, 1]], [2, nil]],
                         [[1, 5], [2, nil]], [[1, 3], [2, 1]], "wrong argument type Integer (must respond to :each)",
                         true, [1, 2], [3, 1], [[true, [3, 1]], [false, [2]]], [3, 1, 2, 9], [[3], [1, 2]], nil])
  pr = P155.new
  y = [pr.map { |x| x }, pr.map { |x, w| w }, pr.select { |x| x }, pr.to_a, pr.flat_map { |x| [x] }, pr.count { |x| x == 3 },
       pr.find_index { |x| x == 3 }, pr.take_while { |x| x }, pr.uniq { |x| x }, pr.reject { |x| x.nil? },
       pr.each_with_object([]) { |x, acc| acc << x }, pr.partition { |x| x }, pr.all? { |x| x.is_a?(Integer) },
       pr.to_h { |x| [x, 0] }]
  check("enm155b", y == [[1, 3, nil], [2, 4, nil], [[1, 2], [3, 4]], [[1, 2], [3, 4], nil], [1, 3, nil], 1, 1,
                         [[1, 2], [3, 4]], [[1, 2], [3, 4], nil], [[1, 2], [3, 4]], [[1, 2], [3, 4], nil],
                         [[[1, 2], [3, 4]], [nil]], false, {1 => 0, 3 => 0, nil => 0}])
  ne = begin
    N155.new.map { |x| x }
    nil
  rescue NoMethodError => e
    [e.name, e.message.start_with?("undefined method `each' for #<N155")]
  end
  check("enm155c", ne == [:each, true] && N155.new.respond_to?(:map))
  u = begin
    um = B155.instance_method(:map)
    [b.method(:map).inspect, b.method(:map).call { |x| x * 2 }, b.method(:map).name, b.method(:each_slice).call(2).to_a,
     um.inspect, um.to_s, um.name, um.class, um.bind(b).call { |x| x + 1 }, B155.instance_method(:each).inspect,
     (begin; B155.instance_method(:nope); rescue NameError => e; [e.name, e.message]; end),
     (begin; B155.instance_method(:each).bind(5); rescue TypeError => e; e.message; end), um.bind(5).inspect,
     [1, 2].method(:map).call { |x| x * 3 },
     B155.instance_method(:each).bind(b).call { |x| x }.equal?(b)]
  rescue NameError
    nil
  end
  check("mth155a", u == ["#<Method: B155(Enumerable)#map>", [6, 2, 4], :map, [[3, 1], [2]],
                         "#<UnboundMethod: B155(Enumerable)#map>", "#<UnboundMethod: B155(Enumerable)#map>", :map,
                         UnboundMethod, [4, 2, 3], "#<UnboundMethod: B155#each>", [:nope, "undefined method `nope' for class `B155'"],
                         "bind argument must be an instance of B155", "#<Method: B155(Enumerable)#map>", [3, 6], true])
end
# ===== SECTION 156: all? / any? / none? / one? stop early; Array#+ - & | operand =====
# Each predicate stops at the element that decides it (one? at the second hit),
# over an Array, a Range and a Hash. Array#+, #-, #& and #| take an Array or a
# #to_ary object, a singleton one too (as String#+ takes a singleton #to_str);
# anything else is a TypeError, so `[[1], nil].sum([])` is rescuable.
# /usr/bin/ruby 2.6 settles each.
class S156A; def to_ary; [9]; end; end
def s156
  r = []
  s = []; r << [1, 2, 3].any? { |x| s << x; true } << s
  s = []; r << [1, 2, 3].all? { |x| s << x; false } << s
  s = []; r << [1, 2, 3].none? { |x| s << x; true } << s
  s = []; r << [1, 2, 3, 4].one? { |x| s << x; true } << s
  s = []; r << (1..5).any? { |x| s << x; x == 2 } << s
  s = []; r << (1..5).one? { |x| s << x; x > 1 } << s
  s = []; r << {a: 1, b: 2}.all? { |k, v| s << k; false } << s
  check("enm156a", r == [true, [1], false, [1], false, [1], false, [1, 2], true, [1, 2], false, [1, 2, 3],
                         false, [:a]])
  e = [[[1], nil], [[1], 3], ["a", nil]].map do |a|
    begin; a.sum(a[0].is_a?(Array) ? [] : ""); rescue TypeError => x; x.message; end
  end
  w = [:+, :-, :&, :|].map do |op|
    [begin; [1].send(op, nil); rescue TypeError => x; x.message; end, [1].send(op, S156A.new)]
  end
  check("ary156b", e == ["no implicit conversion of nil into Array", "no implicit conversion of Integer into Array",
                         "no implicit conversion of nil into String"] &&
                   w == [["no implicit conversion of nil into Array", [1, 9]], ["no implicit conversion of nil into Array", [1]],
                         ["no implicit conversion of nil into Array", []], ["no implicit conversion of nil into Array", [1, 9]]] &&
                   [[1], [2]].sum([]) == [1, 2])
  o = Object.new
  def o.to_ary; [7]; end
  t = Object.new
  def t.to_str; "zz"; end
  check("ary156c", [[1] + o, [1, 7] - o, [1, 7] & o, [1] | o, "a" + t] == [[1, 7], [1], [7], [1, 7], "azz"])
end
# ===== SECTION 157: Enumerable over a user #each stops early; Array#* operand =====
# An Enumerable method that the elements so far decide (any? all? none? one?
# find find_index include? first take take_while) ends the includer's #each
# there, running its ensure clauses; the block runs as each element arrives, and
# first(0) never calls #each; an element whose block raised into a rescue in
# #each is skipped. find(ifnone) answers ifnone.call. Array#* takes an
# Integer (or #to_int) or a String and is a TypeError for anything else; Array
# has no / % **. /usr/bin/ruby 2.6 settles each.
class S157E
  include Enumerable
  def initialize(l); @l = l; end
  def each
    begin
      [1, 2, 3].each { |x| @l << x; yield x }
    ensure
      @l << :e
    end
  end
end
class S157M
  include Enumerable
  def initialize(l); @l = l; end
  def each
    @l << :a; yield 1, 2
    @l << :b; yield 3, 4
    @l << :c; yield 5, 6
  end
end
class S157T; def to_int; 2; end; end
class S157R
  include Enumerable
  def each
    [1, 2, 3].each { |x| begin; yield x; rescue StandardError; end }
  end
end
def s157
  r = []
  l = []; r << S157E.new(l).any? { |x| l << "b#{x}"; x == 2 } << l
  l = []; r << S157E.new(l).all? { |x| x < 2 } << l
  l = []; r << S157E.new(l).none?(1..1) << l
  l = []; r << S157E.new(l).one? { |x| x > 0 } << l
  l = []; r << S157E.new(l).find { |x| x == 1 } << l
  l = []; r << S157E.new(l).find_index(2) << l
  l = []; r << S157E.new(l).include?(1) << l
  l = []; r << S157E.new(l).first << l
  l = []; r << S157E.new(l).first(2) << l
  l = []; r << S157E.new(l).first(0) << l
  l = []; r << S157E.new(l).take(5) << l
  l = []; r << S157E.new(l).take_while { |x| x < 2 } << l
  check("enm157a", r == [true, [1, "b1", 2, "b2", :e], false, [1, 2, :e], false, [1, :e], false, [1, 2, :e],
                         1, [1, :e], 1, [1, 2, :e], true, [1, :e], 1, [1, :e], [1, 2], [1, 2, :e], [], [],
                         [1, 2, 3], [1, 2, 3, :e], [1], [1, 2, :e]])
  m = []
  l = []; m << S157M.new(l).any? { |x| x == 3 } << l
  l = []; m << S157M.new(l).find { |x| x == [3, 4] } << l
  l = []; m << S157M.new(l).find_index { |x, y| y == 4 } << l
  l = []; m << S157M.new(l).first(2) << l
  e = begin; S157E.new(l = []).first(-1); rescue ArgumentError => x; [x.message, l]; end
  check("enm157b", m == [true, [:a, :b], [3, 4], [:a, :b], 1, [:a, :b], [[1, 2], [3, 4]], [:a, :b]] &&
                   e == ["attempt to take negative size", []] &&
                   S157E.new([]).find(-> { :none }) { |x| x > 5 } == :none &&
                   [1, 2].detect(-> { 0 }) { |x| x > 5 } == 0 &&
                   S157R.new.take_while { |x| raise "r" if x == 2; true } == [1, 3])
  w = [nil, :a, {}, true, Object.new, [2]].map do |v|
    begin; [1] * v; rescue TypeError => x; x.message; end
  end
  o = [:/, :%, :**].map do |op|
    begin; [1].send(op, 2); rescue NoMethodError => x; [x.message.lines[0].chomp, x.name, x.receiver]; end
  end
  check("ary157c", w == ["no implicit conversion from nil to integer", "no implicit conversion of Symbol into Integer",
                         "no implicit conversion of Hash into Integer", "no implicit conversion of true into Integer",
                         "no implicit conversion of Object into Integer", "no implicit conversion of Array into Integer"] &&
                   o == [["undefined method `/' for [1]:Array", :/, [1]], ["undefined method `%' for [1]:Array", :%, [1]],
                         ["undefined method `**' for [1]:Array", :**, [1]]] &&
                   [1, 2] * S157T.new == [1, 2, 1, 2] && [1, 2] * 1.9 == [1, 2] && [1, 2] * "-" == "1-2")
end
# ===== SECTION 158: < > <= >= are Comparable's, Hash's and Module's methods =====
# Array, nil, true, false, Range, Regexp, Complex and a plain object have no <:
# a NoMethodError, not a failed comparison - even `[1] < [2]' and `nil <= nil'.
# A class's own < answers; a class with <=> but without Comparable has none of
# the comparison methods (between?, clamp). Hash's are the subset relation, and
# its right side must be a Hash. Module < non-module is a TypeError. The
# ArgumentError names a user class on either side. /usr/bin/ruby 2.6 settles each.
class S158A; def <(o); "lt#{o}"; end; def >=(o); [:ge, o]; end; end
class S158R; def <=>(o); 0; end; end
module S158M; include Comparable; end
class S158W
  include S158M
  attr_reader :v
  def initialize(v); @v = v; end
  def <=>(o); o.is_a?(S158W) ? v <=> o.v : nil; end
end
def s158m(&b)
  b.call
rescue NoMethodError, ArgumentError, TypeError => e
  [e.class, e.message.lines[0].chomp]
end
def s158
  ops = [:<, :>, :<=, :>=]
  nm = [[1], nil, true, false, (1..2), /a/, Complex(1, 2)].map do |v|
    ops.map { |op| s158m { v.send(op, 2) } }
  end
  check("cmp158a", nm[0] == ops.map { |op| [NoMethodError, "undefined method `#{op}' for [1]:Array"] } &&
                   nm[1][1] == [NoMethodError, "undefined method `>' for nil:NilClass"] &&
                   nm[2][2] == [NoMethodError, "undefined method `<=' for true:TrueClass"] &&
                   nm[3][3] == [NoMethodError, "undefined method `>=' for false:FalseClass"] &&
                   nm[4][0] == [NoMethodError, "undefined method `<' for 1..2:Range"] &&
                   nm[5][1] == [NoMethodError, "undefined method `>' for /a/:Regexp"] &&
                   nm[6][0] == [NoMethodError, "undefined method `<' for (1+2i):Complex"] &&
                   s158m { [1] < [2] } == [NoMethodError, "undefined method `<' for [1]:Array"] &&
                   s158m { nil <= nil } == [NoMethodError, "undefined method `<=' for nil:NilClass"] &&
                   s158m { true >= true } == [NoMethodError, "undefined method `>=' for true:TrueClass"] &&
                   s158m { S158R.new < 1 }[0] == NoMethodError && s158m { S158R.new.between?(1, 2) }[0] == NoMethodError &&
                   s158m { S158R.new.clamp(1, 2) }[0] == NoMethodError)
  check("cmp158b", (S158A.new < 3) == "lt3" && (S158A.new >= 4) == [:ge, 4] && S158A.new.send(:<, 5) == "lt5" &&
                   (S158W.new(1) < S158W.new(2)) == true && S158W.new(3).send(:>=, S158W.new(2)) == true &&
                   S158W.new(2).between?(S158W.new(1), S158W.new(3)) &&
                   s158m { S158W.new(1) < 5 } == [ArgumentError, "comparison of S158W with 5 failed"] &&
                   s158m { 5 < S158W.new(1) } == [ArgumentError, "comparison of Integer with S158W failed"] &&
                   s158m { 1.5 >= S158W.new(1) } == [ArgumentError, "comparison of Float with S158W failed"] &&
                   s158m { :a < 2 } == [ArgumentError, "comparison of Symbol with 2 failed"] &&
                   s158m { String < 2 } == [TypeError, "compared with non class/module"] &&
                   (String < Comparable) == true && (:a < :b) == true && (2**70 < 2**71) == true)
  h = {1 => 2}
  check("hsh158c", (h < {1 => 2, 3 => 4}) == true && (h < {1 => 2}) == false && (h <= {1 => 2}) == true &&
                   (h <= {1 => 3}) == false && ({1 => 2, 3 => 4} > h) == true && (h > h) == false &&
                   ({1 => 2.0, 3 => 4} >= h) == true && ({} <= h) == true && (h >= {}) == true &&
                   ops.map { |op| s158m { h.send(op, 2) } } == ops.map { [TypeError, "no implicit conversion of Integer into Hash"] } &&
                   s158m { h < nil } == [TypeError, "no implicit conversion of nil into Hash"])
end
# ===== SECTION 159: a blockless Enumerator or a Lazy over a user #each runs nothing until used =====
# `find', `map', `each_slice(2)' ... without a block answer an Enumerator and
# do not call #each; its #size is nil (the class has no #size), its #each with a
# block is the method with that block (so find's stops #each early), and
# anything else runs it. #lazy runs #each only as far as its first(n) / take
# needs. /usr/bin/ruby 2.6 settles each.
class S159E
  include Enumerable
  def initialize(l); @l = l; end
  def each
    [1, 2, 3].each { |x| @l << x; yield x }
    @l << :end
    self
  end
end
def s159
  r = []
  [:find, :map, :select, :sort_by, :each_with_index, :group_by, :min_by, :reverse_each].each do |m|
    l = []; e = S159E.new(l).send(m); r << [e.class, e.size, l]
  end
  l = []; e = S159E.new(l).each_slice(2); r << [e.class, e.size, l]
  check("enm159a", r == [[Enumerator, nil, []]] * 9)
  w = []
  l = []; e = S159E.new(l).find; w << e.each { |x| x == 2 } << l
  l = []; e = S159E.new(l).take_while; w << e.each { |x| x < 2 } << l
  l = []; e = S159E.new(l).map; w << e.with_index { |x, i| x * i } << l
  l = []; e = S159E.new(l).each_slice(2); w << e.map { |s| s.sum } << l
  l = []; e = S159E.new(l).each_with_index; w << e.to_a << l
  l = []; e = S159E.new(l).sort_by; w << e.each { |x| -x } << l
  check("enm159b", w == [2, [1, 2], [1], [1, 2], [0, 2, 6], [1, 2, 3, :end], [3, 3], [1, 2, 3, :end],
                         [[1, 0], [2, 1], [3, 2]], [1, 2, 3, :end], [3, 2, 1], [1, 2, 3, :end]])
  z = []
  l = []; z << S159E.new(l).lazy.map { |x| x * 3 }.first(1) << l
  l = []; z << S159E.new(l).lazy.select(&:odd?).first(2) << l
  l = []; z << S159E.new(l).lazy.take(2).to_a << l
  l = []; z << S159E.new(l).lazy.take_while { |x| x < 2 }.to_a << l
  l = []; z << S159E.new(l).lazy.each_slice(2).first(1) << l
  l = []; z << S159E.new(l).lazy.drop_while { |x| x < 2 }.first << l
  l = []; z << S159E.new(l).lazy.map { |x| x * 2 }.to_a << l
  l = []; z << S159E.new(l).lazy.first(0) << l
  l = []; zl = S159E.new(l).lazy; z << zl.class << zl.size << l
  check("lzy159c", z == [[3], [1], [1, 3], [1, 2, 3], [1, 2], [1, 2], [1], [1, 2], [[1, 2]], [1, 2], 2, [1, 2],
                         [2, 4, 6], [1, 2, 3, :end], [], [], Enumerator::Lazy, nil, []])
end

# ===== SECTION 160: splat of a non-Array =====
# `*v' splices v itself when it is an Array, nothing for nil, what #to_a answers
# when v responds to it (a Range, a Hash's pairs, an Enumerator or Lazy, a
# Struct, a user object) and v alone otherwise - in an Array literal, a call's
# arguments and a right-hand side alike. /usr/bin/ruby 2.6 settles each.
S160S = Struct.new(:x, :y)
class S160T; def to_a; [7, 8]; end; end
def s160f(*a); a; end
def s160
  x = *[1, 2].each
  y = *nil
  z = *{z: 1}
  check("spl160a", [*[1, 2].each_with_index] == [[1, 0], [2, 1]] && [0, *[1, 2, 3].each_slice(2), 9] == [0, [1, 2], [3], 9] &&
        [*{a: 1, b: 2}] == [[:a, 1], [:b, 2]] && [*nil] == [] && [*5] == [5] && [*"s"] == ["s"] &&
        [*[1, 2].lazy.map { |v| v * 2 }] == [2, 4] && [*S160S.new(1, 2)] == [1, 2] && [*S160T.new] == [7, 8] &&
        x == [1, 2] && y == [] && z == [[:z, 1]])
  check("spl160b", s160f(*[3, 4].each_with_index) == [[3, 0], [4, 1]] && s160f(*{k: 1}) == [[:k, 1]] &&
        s160f(*nil, 1) == [1] && s160f(*S160T.new) == [7, 8] && s160f(*(1..2).each_slice(1)) == [[1], [2]])
end

# ===== SECTION 161: a multi-value yield through a Lazy =====
# `yield 1, 2' reaches the block of a lazy map, take_while or drop_while (and of
# Lazy#each) as two values, so `|x|' takes 1, and that of a select or reject as
# the Array [1, 2]; a value no map has changed comes out as that Array. Lazy#each
# runs its block on each value as #each yields it. /usr/bin/ruby 2.6 settles each.
class S161Y
  include Enumerable
  def initialize(l = []); @l = l; end
  def each; @l << 1; yield 1, 2; @l << 2; yield 3, 4; @l << 3; yield; yield 5; end
end
def s161
  y = S161Y.new
  check("lzm161a", y.lazy.map { |x| x }.first(1) == [1] && y.lazy.map { |x| x }.to_a == [1, 3, nil, 5] &&
        y.lazy.map { |*a| a }.to_a == [[1, 2], [3, 4], [nil], [5]] && y.lazy.map { |a, b| b }.to_a == [2, 4, nil, nil] &&
        y.lazy.drop_while { |x| x == 1 }.to_a == [[3, 4], nil, 5] && y.lazy.take_while { |*a| a.size == 2 }.to_a == [[1, 2], [3, 4]] &&
        y.lazy.select { |x| x == [1, 2] }.to_a == [[1, 2]] && y.lazy.reject { |x| x == [1, 2] }.to_a == [[3, 4], nil, 5] &&
        y.lazy.select { |x| true }.map { |x| x }.to_a == [1, 3, nil, 5] && y.lazy.first(2) == [[1, 2], [3, 4]])
  l = []
  r = []
  S161Y.new(l).lazy.each { |x| r << x; l << :b }
  w = []
  y.lazy.map { |*a| a }.each { |x| w << x }
  check("lzm161b", r == [1, 3, nil, 5] && l == [1, :b, 2, :b, 3, :b, :b] && w == [[1, 2], [3, 4], [nil], [5]])
end

# ===== SECTION 162: Comparable over Object#<=> =====
# Object#<=> answers 0 for the same object or one #== accepts, nil otherwise, so
# Comparable without its own <=> compares an object only with itself and raises
# ArgumentError "comparison of X with 1 failed" otherwise; Comparable#== guards
# against a #== that calls back into it. /usr/bin/ruby 2.6 settles each.
class S162X; include Comparable; end
class S162Q; def ==(o); true; end; end
class S162R; include Comparable; def ==(o); true; end; end
def s162
  x = S162X.new
  e = begin; x < 1; rescue ArgumentError => er; er.message; end
  c = begin; x.clamp(1, 2); rescue ArgumentError => er; er.class; end
  check("cmp162a", e == "comparison of S162X with 1 failed" && c == ArgumentError &&
        (x <= x) == true && (x >= x) == true && (x < x) == false && (x <=> x) == 0 && (x <=> 1).nil? && x == x && x != 1)
  check("cmp162b", (Object.new <=> 1).nil? && (S162Q.new <=> 1) == 0 && (S162R.new <=> 1) == 0 && (S162R.new <= 1) == true &&
        [S162X.new].max.class == S162X)
end

# ===== SECTION 163: for over an Enumerator =====
# `for' over an Enumerator runs over its elements, and answers what the
# Enumerator's #each does: nil after each_slice / each_cons, the receiver after
# the other each* methods; a break still answers its value. It binds each yield
# as a block parameter list would: one loop variable takes the first value of a
# multi-value yield (each_with_index, with_index, a user `yield 1, 2' through
# map), several destructure. /usr/bin/ruby 2.6 settles each.
class S163Y
  include Enumerable
  def each; yield 1, 2; yield 3; yield [4, 5]; yield; end
end
def s163
  y = []
  for v in [5, 6].each_with_index; y << v; end
  for v in [7, 8].each.with_index(1); y << v; end
  for v in [7, 8].each_with_object([]); y << v; end
  for v in [1, 2, 3, 4].each_slice(2); y << v; end
  for a, b in [1, 2, 3, 4].each_slice(2); y << b; end
  for a, b in [7, 8].map.with_index; y << b; end
  check("for163c", y == [5, 6, 7, 8, 7, 8, [1, 2], [3, 4], 2, 4, 0, 1])
  z = []
  for v in S163Y.new.map; z << v; end
  for v in S163Y.new.select; z << v; end
  for a, b in S163Y.new.map; z << [a, b]; end
  for v in S163Y.new.each_with_index; z << v; end
  for v, i in S163Y.new.each_with_index; z << i; end
  check("for163d", z == [1, 3, [4, 5], nil, [1, 2], 3, [4, 5], nil, [1, 2], [3, nil], [4, 5], [nil, nil],
                         [1, 2], 3, [4, 5], nil, 0, 1, 2, 3])

  r = []
  v = for x in [1, 2, 3].each_slice(2); r << x; end
  w = []
  u = for a, i in [5, 6].each_with_index; w << [a, i]; end
  check("for163a", r == [[1, 2], [3]] && v.nil? && w == [[5, 0], [6, 1]] && u == [5, 6])
  c = []
  t = for x in (1..4).each_cons(3); c << x; end
  s = for x in "ab".each_char; c << x; end
  b = for x in [1, 2].each_slice(1); break 7; end
  check("for163b", c == [[1, 2, 3], [2, 3, 4], "a", "b"] && t.nil? && s == "ab" && b == 7)
end
# ===== SECTION 164: Enumerator#each's value; for over an Enumerator; Object#inspect =====
# Enumerator#each with a block answers what the method that made it answers: the
# receiver of each / each_with_index / times / each_char, nil of each_slice, the
# mapped Array of map, the memo of with_object, and through with_index the
# pending op beneath it. `for' is #each with the body as the block, so its value
# is the same, built from the body's values (`next v' included); a Lazy answers
# nil once an op is pending. Object#inspect lists the @ivars after the address
# (twice the object_id here, as in MRI 2.6) and cuts a cycle with ` ...'; a Lazy
# is an Enumerator::Lazy, and respond_to? names what it dispatches. /usr/bin/ruby
# 2.6 settles each.
class S164E
  include Enumerable
  def each; yield 1; yield 2; :done; end
end
class S164Q
  def initialize(a); @a = a; @b = "x"; end
end
class S164R
  def initialize; @me = self; end
end
class S164N; end
def s164
  v = [5, 6]
  check("ev164a", v.each.each { |x| } == v && v.each_slice(1).each { |x| }.nil? && v.each_with_index.each { |x, i| } == v &&
        v.each.with_index.each { |x, i| } == v && v.map.with_index.each { |x, i| x * i } == [0, 6] &&
        3.times.each { |x| } == 3 && "ab".each_char.each { |c| } == "ab" && (1..4).each_slice(2).each { |x| }.nil?)
  w = v.each.with_object([])
  check("ev164b", w.inspect == "#<Enumerator: #<Enumerator: [5, 6]:each>:with_object([])>" && w.to_a == [[5, []], [6, []]] &&
        w.each { |x, m| m << x } == [5, 6] && v.each.with_object([]) { |x, m| m << x * 2 } == [10, 12])
  r1 = for x in v.map; x * 2; end
  r2 = for x in v.select; x > 5; end
  r3 = for x, i in v.map.with_index; x * i; end
  r4 = for x in v.map; next 9 if x == 5; x; end
  r5 = for x in v.each_slice(1); x; end
  r6 = for x in S164E.new.map; x * 3; end
  check("ev164c", r1 == [10, 12] && r2 == [6] && r3 == [0, 6] && r4 == [9, 6] && r5.nil? && r6 == [3, 6])
  l = (1..3).lazy
  r7 = for x in l.map { |y| y * 2 }; end
  r8 = for x in l; end
  check("ev164d", r7.nil? && r8 == (1..3) && l.map { |y| y }.each { |y| }.nil? && v.lazy.each { |y| } == v &&
        S164E.new.lazy.each { |y| } == :done)
  check("ev164e", l.class == Enumerator::Lazy && [l].map(&:class) == [Enumerator::Lazy] && l.respond_to?(:force) &&
        l.respond_to?(:map) && l.respond_to?(:next) && !l.respond_to?(:zork) && !v.each.respond_to?(:force))
  q = S164Q.new(1)
  hx = "0x" + format("%016x", q.object_id * 2)
  check("in164a", q.inspect == "#<S164Q:#{hx} @a=1, @b=\"x\">" && q.to_s == "#<S164Q:#{hx}>" && [q].inspect == "[#{q.inspect}]")
  rr = S164R.new
  rx = "0x" + format("%016x", rr.object_id * 2)
  n = S164N.new
  check("in164b", rr.inspect == "#<S164R:#{rx} @me=#<S164R:#{rx} ...>>" &&
        n.inspect == "#<S164N:0x" + format("%016x", n.object_id * 2) + ">" && n.inspect == n.to_s)
end
# ===== SECTION 165: Enumerator::Lazy's eager Enumerable surface; zip / uniq / flat_map / grep / drop / chunk_while; freeze, dup, clone =====
# A Lazy answers Enumerable's eager methods (sum, count, min, sort, inject,
# group_by, ...) over all its values; find, include?, find_index, any?, all? and
# none? stop at the value that settles them, so an endless source still answers.
# zip, uniq, flat_map, grep, grep_v, drop, chunk_while, slice_when and a blockless
# with_index answer a Lazy, one value at a time. with_index / each_with_object
# with a block run it as each value is made. freeze sets frozen?, clone keeps it
# and dup drops it. A Lazy over an Enumerator answers that Enumerator's #each
# value, and one over a user #each what that #each answered on its last full
# run. /usr/bin/ruby 2.6 settles each.
class S165E
  include Enumerable
  def each; yield 1; yield 2; :done; end
end
class S165M
  include Enumerable
  def each; yield 1, 2; yield 3, 4; :e; end
end
def s165
  l = (1..3).lazy
  inf = (1..Float::INFINITY).lazy
  check("lz165a", l.sum == 6 && l.count == 3 && l.count(2) == 1 && l.count { |x| x > 1 } == 2 && l.include?(2) &&
        !l.include?(5) && l.min == 1 && l.max == 3 && l.sort == [1, 2, 3] && l.reduce(:+) == 6 &&
        l.inject { |a, b| a * b } == 6 && l.find { |x| x > 1 } == 2 && l.detect { |x| x > 5 }.nil? &&
        l.group_by { |x| x.odd? } == {true => [1, 3], false => [2]} && l.partition { |x| x.odd? } == [[1, 3], [2]] &&
        l.min_by { |x| -x } == 3 && l.sum(10) == 16 && l.sum { |x| x * 2 } == 12 && l.min(2) == [1, 2] &&
        l.minmax == [1, 3] && l.sort_by { |x| -x } == [3, 2, 1] && l.to_h { |x| [x, x * x] } == {1 => 1, 2 => 4, 3 => 9})
  calls = []
  f = inf.map { |x| calls << x; x * 2 }.find { |x| x > 10 }
  check("lz165b", f == 12 && calls == [1, 2, 3, 4, 5, 6] && inf.map { |x| x * 2 }.include?(20) && inf.detect { |x| x > 10 } == 11 &&
        inf.find_index { |x| x > 2 } == 2 && inf.find_index(4) == 3 && inf.any? { |x| x > 2 } && !inf.all? { |x| x < 5 } &&
        !inf.none? { |x| x > 5 } && l.any?(2..2) && l.all?(Integer) && l.none?(String) && ![nil, false].lazy.any?)
  check("lz165c", l.each_with_index { |x, i| } == (1..3) && l.each_with_object([]) { |x, a| a << x } == [1, 2, 3] &&
        l.with_index { |x, i| } == (1..3) && l.map { |x| x }.each_with_index { |x, i| }.nil? &&
        l.with_index.to_a == [[1, 0], [2, 1], [3, 2]] && l.with_index(1).to_a == [[1, 1], [2, 2], [3, 3]] &&
        l.each_with_index.map { |x, i| x * i }.to_a == [0, 2, 6] && l.with_index(1).class == Enumerator::Lazy)
  z = l.zip([4, 5, 6])
  check("lz165d", z.class == Enumerator::Lazy && z.inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: 1..3>:zip([4, 5, 6])>" &&
        z.to_a == [[1, 4], [2, 5], [3, 6]] && l.zip([4], 7..9).to_a == [[1, 4, 7], [2, nil, 8], [3, nil, 9]] &&
        inf.zip(10..Float::INFINITY).first(2) == [[1, 10], [2, 11]] && l.zip(l.map { |x| x * 10 }).to_a == [[1, 10], [2, 20], [3, 30]])
  u = [1, 1, 2].lazy.uniq
  check("lz165e", u.inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: [1, 1, 2]>:uniq>" && u.to_a == [1, 2] && u.force == [1, 2] &&
        inf.uniq { |x| x % 3 }.first(3) == [1, 2, 3] && [1, 1.0].lazy.uniq.to_a == [1, 1.0])
  check("lz165f", l.flat_map { |x| [x, x] }.to_a == [1, 1, 2, 2, 3, 3] && inf.flat_map { |x| [x, -x] }.first(3) == [1, -1, 2] &&
        inf.flat_map { |x| x }.first(3) == [1, 2, 3] && inf.flat_map { |x| [[x]] }.first(2) == [[1], [2]] &&
        inf.take(3).flat_map { |x| [x, x] }.to_a == [1, 1, 2, 2, 3, 3] && inf.flat_map { |x| [x, x] }.take(3).to_a == [1, 1, 2] &&
        l.collect_concat { |x| [x] * x }.to_a == [1, 2, 2, 3, 3, 3] && l.find_all { |x| x > 1 }.to_a == [2, 3])
  g = l.grep(2..3)
  check("lz165g", g.inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: 1..3>:grep(2..3)>" && g.to_a == [2, 3] &&
        inf.grep(2..4) { |x| x * 10 }.first(2) == [20, 30] && inf.grep_v(2..4).first(2) == [1, 5] && l.grep(2..3).map { |x| x * 2 }.to_a == [4, 6])
  d = l.drop(1)
  dn = begin; l.drop(-1); rescue ArgumentError => e; e.message; end
  check("lz165h", d.inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: 1..3>:drop(1)>" && d.to_a == [2, 3] && d.to_a == [2, 3] &&
        inf.drop(3).first(2) == [4, 5] && l.drop(5).to_a == [] && l.take(2).drop(1).to_a == [2] && dn == "attempt to drop negative size")
  check("lz165i", [1, 2, 4, 5].lazy.chunk_while { |a, b| b == a + 1 }.to_a == [[1, 2], [4, 5]] &&
        inf.chunk_while { |a, b| b == a + 1 && b % 4 != 0 }.first(2) == [[1, 2, 3], [4, 5, 6, 7]] &&
        (1..10).lazy.take(5).chunk_while { |a, b| b == a + 1 && b != 4 }.to_a == [[1, 2, 3], [4, 5]] &&
        (1..10).lazy.slice_when { |a, b| b % 3 == 0 }.first(2) == [[1, 2], [3, 4, 5]])
  m = S165M.new.lazy
  check("lz165j", m.flat_map { |a, b| [a, b] }.to_a == [1, 2, 3, 4] && m.find { |a, b| b == 4 } == [3, 4] && m.include?([3, 4]) &&
        m.zip([9]).to_a == [[1, 9], [3, nil]] && m.with_index.to_a == [[[1, 2], 0], [[3, 4], 1]] && m.drop(1).to_a == [[3, 4]])
  fz = l.map { |x| x }.freeze
  e = [1, 2].each
  check("lz165k", fz.frozen? && fz.clone.frozen? && !fz.dup.frozen? && fz.clone.to_a == [1, 2, 3] && !l.frozen? &&
        l.dup.inspect == "#<Enumerator::Lazy: 1..3>" && l.clone.class == Enumerator::Lazy && !l.dup.equal?(l) &&
        e.freeze.equal?(e) && e.frozen? && l.respond_to?(:sum) && l.respond_to?(:zip))
  r1 = for x in S165E.new.lazy; end
  r2 = for x in [1, 2].each.lazy; end
  check("lz165l", [1, 2].each.lazy.each { } == [1, 2] && r1 == :done && r2 == [1, 2] && [1, 2].map.lazy.each { |x| x * 3 } == [3, 6] &&
        [1, 2].each_slice(1).lazy.each { |x| }.nil? && {a: 1}.each.lazy.each { |x| } == {a: 1} && [1, 2].each.lazy.each_with_index { |x, i| } == [1, 2])
end
# ===== SECTION 166: ops on an eager endless Enumerator; Object#to_enum on a user class; Enumerator dup / clone =====
# Array#cycle's Enumerator is eager: take_while, map, select, uniq, zip, drop and
# grep on it are Enumerable's and run at once (forever, on a non-empty cycle,
# unless the op stops), a blockless with_index is an Enumerator, and
# chunk_while answers an Enumerator. to_enum / enum_for(meth = :each, *args) on
# any user instance is an Enumerator over what that method yields; its #each
# with a block calls the method and answers what it answers, and its size is
# nil. Enumerator#dup / #clone copy the Enumerator (clone keeps frozen?), and
# raise TypeError once next has begun an external iteration.
# /usr/bin/ruby 2.6 settles each.
class S166C
  include Enumerable
  def each; yield 1; yield 2, 3; :done; end
  def foo(a); yield a; yield a * 2; :foo; end
  def inspect; "C!"; end
end
class S166P
  def each; yield :a; end
end
class S166R
  def each
    return to_enum unless block_given?
    yield 1
    yield 2
    self
  end
  def pairs; return enum_for(:pairs) unless block_given?; yield 1, 2; end
end
def s166
  wi = [1, 2].cycle.with_index
  check("en166a", [1, 2].cycle.take_while { |x| x < 2 } == [1] && [1, 2].cycle.each_slice(3).take_while { |a| a[0] < 2 } == [[1, 2, 1]] &&
        wi.class == Enumerator && wi.inspect == "#<Enumerator: #<Enumerator: [1, 2]:cycle>:with_index>" &&
        wi.first(3) == [[1, 0], [2, 1], [1, 2]] && [1, 2].cycle.with_index(1).first(2) == [[1, 1], [2, 2]] &&
        [1, 2].cycle.each_with_index.class == Enumerator)
  check("en166b", [].cycle.map { |x| x } == [] && [].cycle.uniq == [] && [].cycle.zip([1]) == [] && [].cycle.drop(1) == [] &&
        [].cycle.grep(1) == [] && [].cycle.select { |x| x } == [] &&
        [1, 2].cycle.chunk_while { |a, b| b == a + 1 }.class == Enumerator &&
        [1, 2].cycle.chunk_while { |a, b| b == a + 1 }.first(2) == [[1, 2], [1, 2]] &&
        [1, 2].cycle.lazy.take_while { |x| x < 2 }.class == Enumerator::Lazy && [1, 2].cycle.lazy.with_index.class == Enumerator::Lazy)
  te = S166C.new.to_enum
  check("en166c", te.class == Enumerator && te.inspect == "#<Enumerator: C!:each>" && te.to_a == [1, [2, 3]] &&
        S166C.new.to_enum.lazy.each { } == :done && S166C.new.to_enum.each { |x| x } == :done && te.size.nil? &&
        S166C.new.each_entry.lazy.each { }.class == S166C)
  fe = S166C.new.enum_for(:foo, 5)
  check("en166d", fe.to_a == [5, 10] && fe.inspect == "#<Enumerator: C!:foo(5)>" && fe.each { |x| x } == :foo &&
        fe.map { |x| x + 1 } == [6, 11] && S166C.new.to_enum(:foo, 2).with_index.to_a == [[2, 0], [4, 1]] &&
        S166P.new.to_enum.to_a == [:a] && S166P.new.enum_for.next == :a && te.dup.to_a == [1, [2, 3]] &&
        S166R.new.each.to_a == [1, 2] && S166R.new.each.map { |x| x * 3 } == [3, 6] &&
        S166R.new.each.lazy.each { }.class == S166R && S166R.new.pairs.to_a == [[1, 2]])
  e = [1, 2, 3].each
  cx = begin; n = [1, 2].each; n.next; n.dup; rescue TypeError => x; x.message; end
  check("en166e", e.dup.class == Enumerator && e.clone.inspect == "#<Enumerator: [1, 2, 3]:each>" && !e.dup.equal?(e) &&
        [1, 2, 3].map.dup.each { |x| x * 2 } == [2, 4, 6] && [1, 2].each.freeze.clone.frozen? && ![1, 2].each.freeze.dup.frozen? &&
        (1..3).each_slice(2).dup.inspect == "#<Enumerator: 1..3:each_slice(2)>" && cx == "can't copy execution context" &&
        e.clone.to_a == [1, 2, 3])
end
# ===== SECTION 167: external iteration over a user #each: Enumerator#next / #peek / #rewind, Lazy#next, for =====
# Enumerator#next and #peek over a user #each run the method on a fiber, so each
# value is taken as #each yields it and #each runs no further: its side effects
# interleave with the caller's. Past the end StopIteration ("iteration reached an
# end") carries what the method answered with nil from its block (#result).
# rewind drops the fiber without running ensure, and next starts over. A Lazy
# over a user #each is pulled the same way, map / each_slice / with_index made
# without a block yield as their method would, to_enum / enum_for (an Array's
# too) take the method's own yields, and `for` over such an Enumerator runs its
# body between #each's steps, a break unwinding #each through its ensure. The
# fiber has its own catch tags and $!. /usr/bin/ruby 2.6 settles each.
class S167C
  include Enumerable
  def initialize(l); @l = l; end
  def each
    @l << 1; yield 10
    @l << 2; yield 20
    @l << 3; yield 30
    @l << :end
    :done
  end
end
class S167E
  include Enumerable
  def initialize(l); @l = l; end
  def each
    begin
      yield 1
      yield 2
    ensure
      @l << :ens
    end
  end
end
class S167T
  def each
    catch(:t) { yield 1 }
    begin
      raise "boom"
    rescue => e
      yield e.message
    end
    :t
  end
end
def s167
  l = []
  e = S167C.new(l).each_entry
  a = [e.next, l.dup, e.peek, l.dup, e.next, e.next]
  r = begin; e.next; rescue StopIteration => x; [x.message, x.result.class, l.dup]; end
  check("ext167a", a == [10, [1], 20, [1, 2], 20, 30] && r == ["iteration reached an end", S167C, [1, 2, 3, :end]])
  l = []
  e = S167E.new(l).map
  b = [e.next, e.next]
  e.rewind
  b << l.dup << e.next
  r = begin; e.next; e.next; rescue StopIteration => x; x.result; end
  check("ext167b", b == [1, 2, [], 1] && r == [nil, nil] && l == [:ens])
  l = []
  z = S167C.new(l).lazy.map { |v| v + 1 }
  c = [z.next, z.next, l.dup]
  z2 = S167C.new([]).lazy
  3.times { z2.next }
  r = begin; z2.next; rescue StopIteration => x; x.result; end
  check("ext167c", c == [11, 21, [1, 2]] && r == :done)
  l = []
  d = [S167C.new(l).map.next, l.dup, S167C.new([]).each_slice(2).next, S167C.new([]).each_with_index.next]
  sl = S167C.new([]).each_slice(2)
  d << sl.next << sl.next
  cy = S167C.new([]).cycle
  d << (1..5).map { cy.next }
  l = []
  tw = S167C.new(l).take_while
  d << tw.next << (begin; tw.next; rescue StopIteration => x; x.result; end) << l
  check("ext167d", d == [10, [1], [10, 20], [10, 0], [10, 20], [30], [10, 20, 30, 10, 20], 10, [], [1]])
  l = []
  te = S167C.new(l).to_enum
  ef = S167C.new([]).enum_for(:each)
  f = [te.next, l.dup, ef.next, ef.next, [1, 2].to_enum.next, [3, 4].enum_for(:each_slice, 2).next]
  3.times { ef.next } rescue f << $!.class
  r = begin; te.next; te.next; te.next; rescue StopIteration => x; x.result; end
  check("ext167e", f == [10, [1], 10, 20, 1, [3, 4], StopIteration] && r == :done)
  l = []
  for v in S167C.new(l).each_entry; l << v; end
  m = []
  w = for v in S167E.new(m).each_entry; m << v; break 7; end
  k = []
  for v in S167C.new(k).lazy.map { |x| x + 1 }; k << v; end
  check("ext167f", l == [1, 10, 2, 20, 3, 30, :end] && m == [1, :ens] && w == 7 && k == [1, 11, 2, 21, 3, 31, :end])
  t = S167T.new.to_enum
  g = [t.next]
  g << (begin; throw :t; rescue UncaughtThrowError => x; x.message; end)
  g << t.next << $!
  check("ext167g", g == [1, "uncaught throw :t", "boom", nil])
end

# ===== SECTION 168: block parameter auto-splat and binding shapes =====
# A proc called with ONE Array argument spreads it over its parameters when the
# list has a required parameter, more than one optional or a keyword one, unless
# the list is a lone name (`|a|`; `|a, |` with a trailing comma does spread, and
# `|a, &b|` does not). The spread values bind as a method's do: after a *rest
# from the end, an optional only with spares, but loosely. A group `(b, c)`
# destructures its element the same way (a non-Array is its one element), and
# block-locals after `;` take no argument. A lambda never spreads.
# /usr/bin/ruby 2.6 settles each.
def s168y; yield [4, 5, 6]; end
def s168y4; yield 4; end
def s168
  a = []
  [[1, 2, 3]].each { |m, *r| a << [m, r] }
  s168y { |m, *r| a << [m, r] }
  [[1, [2, 3], 4, 5]].each { |m, (x, y), *r| a << [m, x, y, r] }
  [[1, 2, 3]].each { |*r| a << r }
  {k: 1}.each { |k, *v| a << [k, v] }
  check("blk168a", a == [[1, [2, 3]], [4, [5, 6]], [1, 2, 3, [4, 5]], [[1, 2, 3]], [:k, [1]]])
  b = []
  s168y { |x, | b << x }
  s168y { |x, &f| b << x }
  s168y { |x = 1| b << x }
  s168y { |x = 1, y = 2| b << [x, y] }
  s168y { |x = 1, *r| b << [x, r] }
  s168y { |*r, k: 1| b << [r, k] }
  s168y { |*r, z| b << [r, z] }
  s168y { |x; t| b << [x, t] }
  check("blk168b", b == [4, [4, 5, 6], [4, 5, 6], [4, 5], [[4, 5, 6], []], [[4, 5, 6], 1], [[4, 5], 6], [[4, 5, 6], nil]])
  c = []
  s168y { |x, (y, z)| c << [x, y, z] }
  s168y4 { |x, (y, *z)| c << [x, y, z] }
  s168y { |x, y = 9, *r, z| c << [x, y, r, z] }
  s168y4 { |x, y = 9, *r, z| c << [x, y, r, z] }
  [[[1, 2], 3]].each { |(x, y)| c << [x, y] }
  check("blk168c", c == [[4, 5, nil], [4, nil, []], [4, 5, [], 6], [4, 9, [], nil], [[1, 2], 3]])
  d = []
  l = lambda { |m, *r| d << [m, r] }
  l.call([7, 8])
  [[1, 2]].each(&l)
  pr = proc { |m, *r| d << [m, r] }
  pr.call([7, 8])
  check("blk168d", d == [[[7, 8], []], [[1, 2], []], [7, [8]]])
end

# ===== SECTION 169: `for` over any object with #each; redo in loops =====
# `for x in obj` is `obj.each { |x| ... }` without a new scope, for any object
# that has #each: the body runs between #each's own steps, its value is what the
# yield answers, the loop answers what #each returned, and break, return and a
# raise leave #each through its ensure. The loop variable outlives the loop. A
# single variable takes the first value of a multi-value yield. redo re-runs a
# loop body (for, while, until) without re-testing or advancing.
# /usr/bin/ruby 2.6 settles each.
class S169C
  def initialize(l); @l = l; end
  def each
    @l << :start
    r = yield 10
    @l << r
    yield 20
    @l << -20
    :done
  ensure
    @l << :ensure
  end
end
class S169D; def each; yield 1; yield 2; end; end
class S169P; def each; yield 1, 2; yield [3, 4]; end; end
def s169ret(l)
  for v in S169C.new(l)
    return [:ret, v] if v == 20
  end
  :no
end
def s169
  a = []
  for x in S169D.new; a << x; end
  l = []
  for v in S169C.new(l)
    l << v
    break if v == 20
  end
  w = for v in S169C.new([]); break 7 if v == 20; end
  check("for169a", a == [1, 2] && x == 2 && l == [:start, 10, nil, 20, :ensure] && w == 7)
  l = []
  r = for v in S169C.new(l); l << v; next v * 2 if v == 10; end
  q = for y in S169D.new; y * 10; end
  m = []
  e = begin; for v in S169C.new(m); raise "boom" if v == 20; end; rescue => ex; ex.message; end
  check("for169b", r == :done && l == [:start, 10, 20, 20, -20, :ensure] && q == 20 && y == 2 &&
    e == "boom" && m == [:start, nil, :ensure])
  l = []
  p1 = []
  for g, h in S169P.new; p1 << [g, h]; end
  for g in S169P.new; p1 << g; end
  check("for169c", s169ret(l) == [:ret, 20] && l == [:start, nil, :ensure] && p1 == [[1, 2], [3, 4], 1, [3, 4]])
  c = []
  n = 0
  for v in [1, 2]; n += 1; redo if n == 2; c << [v, n]; end
  i = 0
  while i < 2; i += 1; n += 1; redo if n == 4; c << [:w, i, n]; end
  k = 0
  until k > 0; k += 1; n += 1; redo if n == 6; c << [:u, k, n]; end
  l = []
  for v in S169C.new(l); n += 1; redo if n == 8; l << v; nil; end
  check("for169d", c == [[1, 1], [2, 3], [:w, 2, 5], [:u, 2, 7]] && l == [:start, 10, nil, 20, -20, :ensure])
end

# ===== SECTION 170: Enumerators over a user #each: size, with_index, top-level to_enum =====
# An Enumerator over a user #each (a deferred Enumerable method, or to_enum's)
# has no size to ask, before or after it ran its method. with_index /
# each_with_index with a block on a to_enum Enumerator run the method, the
# values numbered, and answer what it returned. to_enum / enum_for at the top
# level (self is main) run the private top-level def of that name.
# /usr/bin/ruby 2.6 settles each.
class S170C
  include Enumerable
  def each; yield 1; yield 2; yield 3; :done; end
  def pairs; yield 1, 2; yield 3, 4; :pd; end
end
def s170each; yield 7; yield 8; :top; end
def s170two(a); yield a; yield a * 2; :two; end
def s170
  e = S170C.new.each_slice(1)
  m = S170C.new.map
  t = S170C.new.to_enum
  a = [e.size, e.to_a, e.size, m.size, m.to_a.size, m.size, t.size, t.to_a, t.size]
  check("enu170a", a == [nil, [[1], [2], [3]], nil, nil, 3, nil, nil, [1, 2, 3], nil])
  w = []
  r1 = t.with_index { |x, i| w << [x, i] }
  r2 = S170C.new.to_enum.with_index(1) { |x, i| w << [x, i] }
  r3 = S170C.new.to_enum.each_with_index { |x, i| w << i }
  r4 = S170C.new.to_enum(:pairs).with_index { |x, i| w << [x, i] }
  r5 = S170C.new.to_enum.with_index { |x, i| break x * 10 if i == 1 }
  check("enu170b", [r1, r2, r3, r4, r5] == [:done, :done, :done, :pd, 20] &&
    w == [[1, 0], [2, 1], [3, 2], [1, 1], [2, 2], [3, 3], 0, 1, 2, [[1, 2], 0], [[3, 4], 1]])
  te = to_enum(:s170each)
  ef = enum_for(:s170two, 5)
  b = [te.to_a, te.next, te.each { |x| }, ef.to_a, ef.with_index { |x, i| }, te.size]
  check("enu170c", b == [[7, 8], 7, :top, [5, 10], :two, nil])
end

# ===== SECTION 171: break ends the call its block was written for =====
# A block's `break' ends the call the block LITERAL was given to, not the
# innermost call running a block: through a yield in another method's block,
# through a block passed on by &b, out of a recursion. Once that call has
# returned, a break of the block (a stored proc) raises LocalJumpError. In a
# lambda, a `->' and a define_method body, break ends that call only, like
# return. /usr/bin/ruby 2.6 settles each.
def s171b; [1].each { yield }; :b_end; end
def s171c; r = [1, 2].map { |v| yield v }; [:c, r]; end
def s171pass(&b); [1, 2].each(&b); :pass_end; end
def s171rec(n, &b); return yield(n) if n == 0; [n].each { |m| s171rec(m - 1, &b) }; :rec_end; end
def s171keep(&b); b; end
class S171D
  define_method(:dm) { break 6; 7 }
  def initialize; @v = block_given? ? yield : 0; end
  attr_reader :v
end
def s171
  check("brk171a", s171b { break 5 } == 5 && s171c { |v| break v * 7 } == 7 &&
        s171pass { |v| break [:p, v] if v == 2 } == [:p, 2] && s171rec(2) { |z| break [:z, z] } == [:z, 0])
  check("brk171b", [1, 2, 3].each { |v| [4].each { break }; break v if v == 2 } == 2 &&
        S171D.new { break 8 } == 8 && s171b { [1, 2].each { |v| break v }; :inner } == :b_end)
  st = s171keep { break 1 }
  e1 = begin; st.call; rescue LocalJumpError => e; [e.class, e.message]; end
  e2 = begin; [1].each { proc { break 2 }.call }; rescue LocalJumpError => e; e.message; end
  check("brk171c", e1 == [LocalJumpError, "break from proc-closure"] && e2 == "break from proc-closure")
  l = lambda { |x| break x * 3; 0 }
  m = ->(x) { break x * 2; 0 }
  check("brk171d", l.call(2) == 6 && m.call(4) == 8 && S171D.new.dm == 6 &&
        [1].map { lambda { break 9 }.call } == [9])
end

# ===== SECTION 172: LocalJumpError#reason and #exit_value =====
# The LocalJumpError a stray `break' raises records why (:break) and the value
# it carried; one built by LocalJumpError.new has neither, and no other
# exception class answers the two. /usr/bin/ruby 2.6 settles each.
def s172keep(&b); b; end
def s172
  v = [1, [:a, 2]]
  e1 = begin; proc { break v[0] }.call; rescue LocalJumpError => e; [e.reason, e.exit_value]; end
  st = s172keep { break v[1] }
  e2 = begin; st.call; rescue LocalJumpError => e; [e.reason, e.exit_value, e.message]; end
  check("lje172a", e1 == [:break, 1] && e2 == [:break, [:a, 2], "break from proc-closure"])
  e3 = begin; RuntimeError.new.reason; rescue NoMethodError => e; e.class; end
  check("lje172b", LocalJumpError.new("x").reason.nil? && LocalJumpError.new.exit_value.nil? && e3 == NoMethodError)
end

# ===== SECTION 173: super passes a literal block =====
# super with a literal block or &expr passes that block, &nil passes none, and
# without one super passes the method's own block whether or not it lists
# arguments. A `break' in the literal block ends the super call. Class methods
# and initialize too. /usr/bin/ruby 2.6 settles each.
class S173A
  def f(x); r = block_given? ? yield(x) : :nob; [:a, r]; end
  def self.cm(x); yield(x) * 10; end
  def initialize(n = 0); @b = block_given? ? yield(n) : :none; end
  attr_reader :b
end
class S173B < S173A
  def f(x); super { |v| v * 2 }; end
  def self.cm(x); super(x) { |v| v + 2 }; end
  def initialize(n); super { |m| m * 3 }; end
end
class S173C < S173A; def f(x); super(x + 1) { |v| break v * 7 }; end; end
class S173D < S173A; def f(x); super(x); end; end
class S173E < S173A; def f(x); super(x, &nil); end; end
class S173F < S173A; def f(x); [x].map { |q| super(q) }; end; end
def s173
  v = [5, 4]
  check("sup173a", S173B.new(v[1]).f(v[0]) == [:a, 10] && S173B.new(v[1]).f(v[0]) { |q| q + 100 } == [:a, 10] &&
        S173C.new.f(v[0]) == 42 && S173B.cm(v[1]) == 60 && S173B.new(v[1]).b == 12)
  check("sup173b", S173D.new.f(v[0]) { |q| q + 100 } == [:a, 105] && S173D.new.f(v[0]) == [:a, :nob] &&
        S173E.new.f(v[0]) { |q| q + 7 } == [:a, :nob] && S173F.new.f(v[0]) { |q| q * 2 } == [[:a, 10]])
end

# ===== SECTION 174: ranges and steps without an end =====
# An Integer range ending in nil or Float::INFINITY, and Numeric#step with no
# limit, a nil one or an infinite one it steps toward, make one value at a time:
# a break, a find hit, a take_while stop or first(n) ends them. A Float end or
# step makes Floats. #size of such a range is Infinity. /usr/bin/ruby 2.6
# settles each.
def s174
  v = [1, nil, Float::INFINITY, 50, 2, 3]
  r = (v[0]..v[1])
  ri = (v[0]..v[2])
  check("inf174a", ri.each { |i| break i if i * i > v[3] } == 8 && r.each { |i| break i if i * i > v[3] } == 8 &&
        ri.find { |i| i % 7 == 0 } == 7 && r.each_with_index { |x, i| break [x, i] if i == 2 } == [3, 2] &&
        r.map { |x| break x * 10 if x > 2; x } == 30 && r.take_while { |x| x < 4 } == [1, 2, 3] &&
        r.select { |x| break x if x > 4; x.odd? } == 5)
  check("inf174b", r.size == Float::INFINITY && ri.size == Float::INFINITY &&
        r.each_slice(v[4]) { |s| break s if s[0] > 2 } == [3, 4] && r.step(v[5]) { |x| break x if x > 5 } == 7 &&
        ri.step(v[5]) { |x| break x if x > 5 } == 7.0 && r.step(v[4]).first(3) == [1, 3, 5])
  check("inf174c", v[0].step { |i| break i if i > 3 } == 4 && v[0].step(v[1], v[4]).first(3) == [1, 3, 5] &&
        10.step(v[1], -v[5]) { |i| break i if i < 0 } == -2 && v[0].step(by: v[4]) { |i| break i if i > 6 } == 7 &&
        v[0].step(v[2], 0.5) { |i| break i if i > 2 } == 2.5 && v[0].step(-v[2], 1) { |i| break i } == 1 &&
        v[0].step(by: v[5], to: 7).to_a == [1, 4, 7])
end

# ===== SECTION 175: endless Float / String ranges, each_entry, ArithmeticSequence =====
# A range beginning at a Float cannot be iterated, endless or not (TypeError);
# it still has a size. An endless String range walks String#succ until a break.
# each_entry on a range is each. Numeric#step and a numeric Range#step without a
# block answer an Enumerator::ArithmeticSequence: #inspect shows the call in
# parentheses, and it has begin / end / step / exclude_end? / last / size / ==.
# Only iterating rejects a step of 0 or below. A non-numeric Range#step stays an
# Enumerator. /usr/bin/ruby 2.6 settles each.
def s175err
  yield
  "none"
rescue => e
  "#{e.class} #{e.message}"
end
def s175
  v = [2.5, 4, "a", "y", 1, nil, 10, 3, 0, -1, 0.5, 1.0, 2.0]
  check("ar175a", s175err { (v[0]..).each { } } == "TypeError can't iterate from Float" &&
        s175err { (v[0]..v[1]).each { } } == "TypeError can't iterate from Float" &&
        s175err { (v[0]..v[1]).to_a } == "TypeError can't iterate from Float" &&
        s175err { for x in (v[0]..v[1]) do end } == "TypeError can't iterate from Float" &&
        (v[0]..v[1]).size == 2 && (v[0]..v[1]).include?(3))
  check("ar175b", (v[2]..).each { |s| break s if s.size > 1 } == "aa" && (v[3]..).first(3) == ["y", "z", "aa"] &&
        (v[2]..).find { |s| s.size > 1 } == "aa" && (v[2]..).size.nil? &&
        (v[4]..v[5]).each_entry { |s| break s if s > 2 } == 3 && (v[4]..v[7]).each_entry { } == (1..3))
  a = v[4].step(v[6], v[7])
  r = (v[4]..v[6]).step(v[7])
  check("ar175c", a.inspect == "(1.step(10, 3))" && r.inspect == "((1..10).step(3))" &&
        a.class == Enumerator::ArithmeticSequence && r.is_a?(Enumerator) &&
        Enumerator::ArithmeticSequence.superclass == Enumerator &&
        v[4].step(v[6], v[8]).inspect == "(1.step(10, 0))" && v[4].step(v[6], by: v[7]).inspect == "(1.step(10, by: 3))" &&
        (v[11]..v[12]).step(v[10]).inspect == "((1.0..2.0).step(0.5))" && (v[4]...).step(2).inspect == "((1...).step(2))" &&
        (v[2].."e").step(2).inspect == "#<Enumerator: \"a\"..\"e\":step(2)>" &&
        a.lazy.inspect == "#<Enumerator::Lazy: (1.step(10, 3))>" && a.each_slice(2).inspect == "#<Enumerator: (1.step(10, 3)):each_slice(2)>")
  check("ar175d", a.begin == 1 && a.end == 10 && a.step == 3 && !a.exclude_end? && (v[4]...v[6]).step(v[7]).exclude_end? &&
        a.first == 1 && a.last == 10 && a.last(2) == [7, 10] && a.size == 4 && a.to_a == [1, 4, 7, 10] &&
        (v[4]...v[6]).step(v[7]).last == 7 && (v[4]...v[6]).step(v[7]).size == 3 &&
        a == r && a == v[4].step(v[6], v[7]) && a != v[4].step(v[6], 4) && a != [1, 4, 7, 10] &&
        a.each { } .equal?(a) && (v[11]..v[12]).step(v[10]).size == 3 && 10.step(1, -3).last == 1)
  check("ar175e", (v[4]..).step(2).size == Float::INFINITY && (v[4]..).step(2).end.nil? &&
        s175err { (v[4]..).step(2).last } == "RangeError cannot get the last element of endless arithmetic sequence" &&
        v[4].step(v[6], v[8]).first(2) == [1, 1] && v[4].step(v[6], v[8]).size == Float::INFINITY &&
        (v[4]..v[6]).step(v[9]).to_a == [] && (v[4]..v[6]).step(v[9]).size == 0 && (v[4]..v[6]).step(v[8]).first(2) == [1, 1] &&
        s175err { (v[4]..v[6]).step(v[8]) { } } == "ArgumentError step can't be 0" &&
        s175err { (v[4]..v[6]).step(v[9]) { } } == "ArgumentError step can't be negative" &&
        s175err { v[4].step(v[6], v[8]) { break } } == "ArgumentError step can't be 0" &&
        s175err { a.dup } == "TypeError allocator undefined for Enumerator::ArithmeticSequence")
end

# ===== SECTION 176: endless-range to_a, streaming step / each_entry, Range#%, ArithmeticSequence #hash / Float values =====
# An endless range has no Array: to_a, entries and a splat raise a rescuable
# RangeError before any Float check. An endless String range's #step and a
# blockless #each / #each_entry pull one value at a time. Range#% is Range#step.
# An ArithmeticSequence hashes what its == compares. Without a block,
# Integer#step toward a Float infinity yields Floats through #each, Integers
# through with_index, and #first alone answers the begin itself.
# /usr/bin/ruby 2.6 settles each.
def s176
  v = [1, nil, "a", 2.5, 10, 3, 2, Float::INFINITY, 4, "x"]
  re = begin; (v[0]..v[1]).to_a; "none"; rescue RangeError => e; e.message; end
  check("ar176a", re == "cannot convert endless range to an array" &&
        s175err { (v[0]..v[1]).entries } == "RangeError cannot convert endless range to an array" &&
        s175err { [*(v[2]..v[1])] } == "RangeError cannot convert endless range to an array" &&
        s175err { (v[3]..v[1]).to_a } == "RangeError cannot convert endless range to an array" &&
        s175err { [*(v[3]..v[1])] } == "RangeError cannot convert endless range to an array" &&
        s175err { (v[3]..v[1]).each { } } == "TypeError can't iterate from Float")
  r = []
  (v[9]..v[1]).step(v[6]) { |s| r << s; break if r.size > 3 }
  check("ar176b", (v[2]..v[1]).step(v[6]).first(2) == ["a", "c"] && r == ["x", "z", "ab", "ad"] &&
        (v[2]..v[1]).step(v[6]).inspect == "#<Enumerator: \"a\"..:step(2)>" &&
        (v[0]..v[1]).each_entry.first(2) == [1, 2] && (v[0]..v[1]).each_entry.inspect == "#<Enumerator: 1..:each_entry>" &&
        (v[0]..v[7]).each.first(2) == [1, 2] && (v[2]..v[1]).each_entry.first(3) == ["a", "b", "c"])
  pc = []
  rr = (v[0]..v[4]).%(v[8]) { |x| pc << x }
  check("ar176c", ((v[0]..v[4]) % v[5]).inspect == "((1..10).%(3))" && ((v[0]..v[4]) % v[5]).to_a == [1, 4, 7, 10] &&
        ((v[0]..v[4]) % v[5]) == (v[0]..v[4]).step(v[5]) && pc == [1, 5, 9] && rr == (1..10) &&
        ((v[2].."e") % v[6]).inspect == "#<Enumerator: \"a\"..\"e\":%(2)>" && ((v[2].."e") % v[6]).to_a == ["a", "c", "e"] &&
        ((v[2]..v[1]) % v[6]).first(2) == ["a", "c"] && s175err { (v[0]..v[5]).%(0) { } } == "ArgumentError step can't be 0")
  check("ar176d", v[0].step(v[4], v[5]).hash == v[0].step(v[4], v[5]).hash &&
        v[0].step(v[4], v[5]).hash != v[0].step(v[4], v[8]).hash &&
        (v[0]..v[4]).step(v[5]).hash == v[0].step(v[4], v[5]).hash && v[0].step(v[4], v[5]).hash.is_a?(Integer))
  f = v[0].step(v[7], v[6])
  fy = []
  v[0].step(v[7], v[6]) { |x| fy << x; break if x > 2 }
  check("ar176e", f.first(2) == [1.0, 3.0] && f.first(2)[0].is_a?(Float) && f.take(2) == [1.0, 3.0] &&
        f.lazy.map { |x| x * 2 }.first(2) == [2.0, 6.0] && fy == [1, 3] && fy[0].is_a?(Integer) &&
        f.each_slice(2).first(2) == [[1.0, 3.0], [5.0, 7.0]] && f.with_index.first(2) == [[1, 0], [3, 1]] &&
        f.each_with_index.first(1)[0][0].is_a?(Integer) && f.first == 1 && f.first.is_a?(Integer) &&
        f.next == 1.0 && v[0].step(v[4].to_f, v[5]).first.is_a?(Integer) && v[0].step(0, 1).first.nil?)
end

# ===== SECTION 177: ArithmeticSequence respond_to?, Hash key, blockless map / with_object, #last =====
# An ArithmeticSequence answers respond_to? for its own methods and an
# Enumerator's, and is a Hash key by eql? of begin / end / step / exclude_end?.
# Without a block, map / select / reject / flat_map of an endless one answer an
# Enumerator over its values (Floats toward a Float end) and each_with_object
# pairs the Integers with the memo. #last is (end - begin).div(step) steps from
# begin: an infinite or NaN Float quotient raises FloatDomainError, only a
# missing end is "endless". /usr/bin/ruby 2.6 settles each.
def s177
  v = [1, 10, 3, 2, Float::INFINITY, nil, 0, 1.0, Float::NAN, :m]
  a = v[0].step(v[1], v[2])
  check("as177a", [a.respond_to?(:first), a.respond_to?(:begin), a.respond_to?(:each), a.respond_to?(:step),
                   a.respond_to?(:exclude_end?), a.respond_to?(:last), a.respond_to?(:map), a.respond_to?(:force),
                   a.respond_to?(:foo)] == [true, true, true, true, true, true, true, false, false] &&
        Enumerator::ArithmeticSequence.instance_methods(false).sort ==
          [:==, :===, :begin, :each, :end, :eql?, :exclude_end?, :first, :hash, :inspect, :last, :size, :step] &&
        s175err { a.foo }.start_with?("NoMethodError undefined method `foo' for (1.step(10, 3)):Enumerator::ArithmeticSequence"))
  h = {a => :a}
  h[v[0].step(v[1], v[2])] = :c
  check("as177b", h[v[0].step(v[1], v[2])] == :c && h[(v[0]..v[1]).step(v[2])] == :c && h.size == 1 &&
        h[v[7].step(v[1], v[2])].nil? && h[(v[0]...v[1]).step(v[2])].nil? && h[v[0].step(v[1], v[3])].nil? &&
        h.key?((v[0]..v[1]).step(v[2])) && a.eql?(v[7].step(v[1], v[2])))
  e = v[0].step(v[4], v[3])
  check("as177c", e.map.first(1) == [1.0] && e.each_with_object([]).first(1) == [[1, []]] &&
        e.map.inspect == "#<Enumerator: (1.step(Infinity, 2)):map>" &&
        e.with_object(v[9]).inspect == "#<Enumerator: (1.step(Infinity, 2)):with_object(:m)>" &&
        e.select.first(2) == [1.0, 3.0] && e.reject.first(2) == [1.0, 3.0] && e.flat_map.first(2) == [1.0, 3.0] &&
        e.map.with_index.first(2) == [[1.0, 0], [3.0, 1]] && e.with_object(v[9]).first == [1, :m] &&
        v[0].step.map.first(2) == [1, 2] && (v[0]..v[5]).step(v[3]).each_with_object(v[9]).first(2) == [[1, :m], [3, :m]] &&
        [v[0], v[3]].cycle.map.first(3) == [1, 2, 1] && (v[0]..v[5]).lazy.each_with_object([]).first(2) == [[1, []], [2, []]])
  m = []
  v[0].step.each_with_object(m).each { |x, o| o << x; break if x > 2 }
  check("as177d", m == [1, 2, 3] &&
        v[0].step.each_with_object(v[9]).inspect == "#<Enumerator: (1.step):each_with_object(:m)>" &&
        (v[0]..v[5]).lazy.each_with_object([]).inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: 1..>:each_with_object([])>")
  check("as177e", s175err { e.last } == "FloatDomainError Infinity" && s175err { e.last(2) } == "FloatDomainError Infinity" &&
        s175err { v[0].step(-v[4], v[3]).last } == "FloatDomainError -Infinity" &&
        s175err { v[0].step(v[4], v[6]).last } == "FloatDomainError Infinity" &&
        s175err { v[0].step(v[8], v[3]).last } == "FloatDomainError NaN" &&
        s175err { v[0].step(v[1].to_f, v[6]).last } == "FloatDomainError Infinity" &&
        s175err { v[0].step(v[1], v[6]).last } == "ZeroDivisionError divided by 0" &&
        s175err { v[0].step(v[1], v[6].to_f).last } == "ZeroDivisionError divided by 0" &&
        s175err { (v[0]..v[4]).step(v[3]).last } == "FloatDomainError Infinity" &&
        s175err { (v[0]..v[5]).step(v[3]).last } == "RangeError cannot get the last element of endless arithmetic sequence" &&
        s175err { v[0].step(v[5], v[3]).last } == "RangeError cannot get the last element of endless arithmetic sequence" &&
        v[0].step(v[1], v[2]).last == 10 && v[7].step(v[1], v[2]).last(2) == [7.0, 10.0])
end

# ===== SECTION 178: Hash keys and Array set operations compare by eql? =====
# A Hash key matches by eql?, which never converts across classes: 1 and 1.0
# are two keys, and an Array, Hash or Range key applies eql? to its parts, so
# {[1] => 1}[[1.0]] is nil. A Float key is found by its value, 0.0 and -0.0
# alike. Array#uniq / #- / #& / #| use the same rule, and Hash#eql? and
# Range#eql? are the structural eql? itself. /usr/bin/ruby 2.6 settles each.
def s178
  v = [1, 1.0, 2, 2.0, 1.5, 0.0, -0.0, :a, :b]
  h = {[v[0]] => v[7], v[0] => v[8], [v[0], [v[2]]] => v[7], v[1] => v[7]}
  check("he178a", h[[v[1]]].nil? && h[[v[0]]] == v[7] && h[[v[0], [v[3]]]].nil? && h[[v[0], [v[2]]]] == v[7] &&
        h[v[1]] == v[7] && h[v[0]] == v[8] && {v[1] => v[7]}[v[0]].nil? && {v[0] => v[7]}[v[1]].nil? &&
        {v[4] => v[7]}[v[4]] == v[7] && {v[6] => v[7]}[v[5]] == v[7] && {[v[4]] => v[7]}[[v[4]]] == v[7])
  check("he178b", !h.key?([v[1]]) && h.key?([v[0]]) && h.fetch([v[3]], v[8]) == v[8] && h.delete([v[1]]).nil? &&
        h.size == 4 && {{v[0] => v[2]} => v[7]}[{v[0] => v[3]}].nil? && {(v[0]..v[2]) => v[7]}[v[1]..v[3]].nil? &&
        {(v[0]..v[2]) => v[7]}[v[0]..v[2]] == v[7] && {v[1] => v[7]} == {v[1] => v[7]})
  g = {}
  g[v[0]] = 1
  g[v[1]] = 2
  g[[v[0]]] = 3
  g[[v[1]]] = 4
  check("he178c", g.size == 4 && g.values == [1, 2, 3, 4] && g.inspect == "{1=>1, 1.0=>2, [1]=>3, [1.0]=>4}")
  check("he178d", [v[0], v[1], [v[0]], [v[1]], v[1]].uniq == [1, 1.0, [1], [1.0]] && [[v[0]], [v[1]]] - [[v[0]]] == [[1.0]] &&
        ([[v[0]], [v[1]]] & [[v[1]]]) == [[1.0]] && ([[v[0]]] | [[v[1]]]) == [[1], [1.0]] &&
        [{v[0] => v[2]}, {v[0] => v[3]}].uniq.size == 2 && [v[0]..v[2], v[1]..v[3]].uniq.size == 2 &&
        [[v[0], [v[2]]], [v[0], [v[3]]]].uniq.size == 2)
  check("he178e", !{v[0] => v[2]}.eql?({v[0] => v[3]}) && {v[0] => v[2]}.eql?({v[0] => v[2]}) &&
        !(v[0]..v[2]).eql?(v[1]..v[3]) && (v[0]..v[2]).eql?(v[0]..v[2]) && (v[0]..v[2]) == (v[1]..v[3]) &&
        ![v[0]].eql?([v[1]]) && [v[0]] == [v[1]])
end

# ===== SECTION 179: endless-stream Enumerators: blockless Range methods, Infinity #size, to_enum, Method#owner =====
# An endless (or +Infinity) Integer range answers map / select / reject /
# flat_map / each_with_index / each_with_object without a block with an
# Enumerator that makes one value at a time. The #size of an Enumerator over a
# stream that never ends (that range, a cycle, a step without an end) is
# Infinity through map / with_index / with_object / each_cons, and
# each_slice's (size + n - 1).div(n) raises FloatDomainError in MRI 2.6.
# Enumerator#to_enum pulls the receiver's #each and has a nil #size. Method#owner
# of a builtin's Method is the class or module defining it, and
# public_methods(false) lists the class's own. /usr/bin/ruby 2.6 settles each.
def s179err
  yield
rescue => e
  "#{e.class} #{e.message}"
end
def s179
  v = [1, 2, Float::INFINITY, nil, 10, 3, :m]
  r = (v[0]..v[3])
  check("es179a", r.map.first(2) == [1, 2] && r.each_with_object([]).first(1) == [[1, []]] &&
        r.each_with_index.first(2) == [[1, 0], [2, 1]] && r.select.first(1) == [1] && r.reject.first(2) == [1, 2] &&
        r.flat_map.first(2) == [1, 2] && r.map.with_index(1).first(2) == [[1, 1], [2, 2]] &&
        r.map.inspect == "#<Enumerator: 1..:map>" && r.each_with_object(v[6]).inspect == "#<Enumerator: 1..:each_with_object(:m)>" &&
        r.map.each { |x| break x * 7 } == 7 && r.select.each { |x| break x if x > 3 } == 4 && r.reject.next == 1 &&
        (v[0]..v[2]).map.first(3) == [1, 2, 3] && (v[0]..v[2]).each_with_index.first(1) == [[1, 0]])
  check("es179b", [v[0], v[1]].cycle.size == v[2] && v[0].step.with_index.size == v[2] && v[0].step.map.size == v[2] &&
        r.map.size == v[2] && r.select.size == v[2] && r.each_with_index.size == v[2] && r.each.size == v[2] &&
        r.each_cons(2).size == v[2] && [v[0]].cycle.map.with_index.size == v[2] && v[0].step(v[2], 2).with_object([]).size == v[2] &&
        r.lazy.size == v[2] && r.lazy.each_with_index.size == v[2] && r.lazy.select { |x| x }.size.nil? &&
        [].cycle.size == 0 && [v[0]].cycle(3).size == 3 && [v[0], v[1]].cycle.each_slice(2).first(1) == [[1, 2]] &&
        s179err { r.each_slice(2).size } == "FloatDomainError Infinity" &&
        s179err { v[0].step.each_slice(2).size } == "FloatDomainError Infinity")
  a = v[0].step(v[2], 2)
  check("es179c", a.to_enum.first(2) == [1.0, 3.0] && a.enum_for.first(3) == [1.0, 3.0, 5.0] && a.to_enum.size.nil? &&
        a.to_enum.inspect == "#<Enumerator: (1.step(Infinity, 2)):each>" && v[0].step.to_enum.first(2) == [1, 2] &&
        v[0].step(v[4], v[5]).to_enum.to_a == [1, 4, 7, 10] && v[0].step(v[4], v[5]).to_enum.size.nil? &&
        [v[0], v[1]].cycle.to_enum.first(3) == [1, 2, 1] && r.lazy.to_enum.first(2) == [1, 2] &&
        [v[0], v[1]].cycle.to_enum.inspect == "#<Enumerator: #<Enumerator: [1, 2]:cycle>:each>")
  q = v[0].step(v[4], v[5])
  check("es179d", q.method(:begin).owner == Enumerator::ArithmeticSequence && q.method(:first).owner == Enumerator::ArithmeticSequence &&
        q.method(:with_index).owner == Enumerator && q.method(:map).owner == Enumerable && [v[0]].method(:map).owner == Array &&
        [v[0]].method(:each_slice).owner == Enumerable && {v[0] => v[1]}.method(:member?).owner == Hash &&
        {v[0] => v[1]}.method(:map).owner == Enumerable && v[0].method(:+).owner == Integer && "a".method(:upcase).owner == String &&
        r.method(:each).owner == Range && q.method(:begin).inspect == "#<Method: Enumerator::ArithmeticSequence#begin>")
  check("es179e", q.public_methods(false).sort == [:==, :===, :begin, :each, :end, :eql?, :exclude_end?, :first, :hash, :inspect, :last, :size, :step] &&
        [v[0]].public_methods(false).sort == Array.instance_methods(false).sort && Hash.instance_methods(false).include?(:member?))
end

# ===== SECTION 180: blockless find / take_while on streams, Lazy #size, builtin Method#arity, Range identity =====
# find / detect / find_index / take_while / drop_while without a block answer an
# Enumerator over the same values whose #size is nil and whose #each with a
# block runs the method - on a cycle, a step, an endless range and an eager
# Array alike; a Lazy's find is a Lazy, while its map / select / take_while ...
# without a block raise ArgumentError. Enumerator#each without a block is the
# receiver, each_entry streams, and Range has no with_index / with_object. A
# Lazy's #size runs the source's count through its ops (take caps it). A builtin
# Method's #arity is MRI's, and Range#dup / #equal? are about identity.
# /usr/bin/ruby 2.6 settles each.
def s180err
  yield
rescue => e
  "#{e.class} #{e.message}"
end
def s180
  v = [1, 2, nil, Float::INFINITY, 3, 10, 5]
  c = [v[0], v[1]].cycle
  r = (v[0]..v[2])
  check("st180a", c.find.class == Enumerator && c.find.size.nil? && c.find_index.size.nil? && c.take_while.size.nil? &&
        c.find.inspect == "#<Enumerator: #<Enumerator: [1, 2]:cycle>:find>" && c.find.each { |x| x > 1 } == 2 &&
        c.detect.first(3) == [1, 2, 1] && c.find_index.each { |x| x > 1 } == 1 && c.take_while.each { |x| x < 2 } == [1] &&
        [v[0], v[1]].cycle(2).find.size.nil? && [v[0], v[1]].cycle(2).drop_while.each { |x| x < 2 } == [2, 1, 2] &&
        [v[0], v[1], v[4]].take_while.size.nil? && [v[0], v[1], v[4]].find.each { |x| x > 1 } == 2 &&
        {a: v[0], b: v[1]}.find.each { |k, x| x > 1 } == [:b, 2] && (v[0]..v[4]).find_index.size.nil? &&
        [v[0], v[1], v[4]].rindex.each { |x| x < 3 } == 1 && [v[0], v[1], v[4]].rindex.size.nil?)
  s = v[0].step
  check("st180b", s.find.size.nil? && s.find.inspect == "#<Enumerator: (1.step):find>" && s.find.each { |x| x > 2 } == 3 &&
        s.take_while.each { |x| x < 3 } == [1, 2] && r.each.find.size.nil? && r.find.each { |x| x > 4 } == 5 &&
        r.take_while.each { |x| x < 3 } == [1, 2] && (v[0]..v[3]).find.size.nil? && (v[0]..v[3]).find.each { |x| x > 2 } == 3 &&
        r.lazy.find.class == Enumerator::Lazy && r.lazy.find_index.each { |x| x > 1 } == 1 &&
        r.lazy.detect.inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: 1..>:detect>" && c.find.dup.size.nil?)
  check("st180c", c.each_entry.class == Enumerator && c.each_entry.first(3) == [1, 2, 1] && s.each_entry.size == v[3] &&
        r.lazy.each_entry.first(2) == [1, 2] && c.each.equal?(c) && c.send(:each).equal?(c) && r.lazy.each.class == Enumerator::Lazy &&
        s.send(:each).first(2) == [1, 2] && c.each_entry.inspect == "#<Enumerator: #<Enumerator: [1, 2]:cycle>:each_entry>")
  check("st180d", s180err { r.lazy.map } == "ArgumentError tried to call lazy map without a block" &&
        s180err { [v[0]].lazy.collect } == "ArgumentError tried to call lazy map without a block" &&
        s180err { [v[0]].lazy.filter } == "ArgumentError tried to call lazy select without a block" &&
        s180err { [v[0]].lazy.reject } == "ArgumentError tried to call lazy reject without a block" &&
        s180err { [v[0]].lazy.flat_map } == "ArgumentError tried to call lazy flat_map without a block" &&
        s180err { [v[0]].lazy.take_while } == "ArgumentError tried to call lazy take_while without a block" &&
        s180err { [v[0]].lazy.drop_while } == "ArgumentError tried to call lazy drop_while without a block" &&
        s180err { [v[0]].lazy.chunk_while } == "ArgumentError tried to create Proc object without a block" &&
        s180err { r.lazy.take(-v[0]) } == "ArgumentError attempt to take negative size" &&
        [v[0]].lazy.zip.inspect == "#<Enumerator::Lazy: #<Enumerator::Lazy: [1]>:zip>")
  check("st180e", s180err { r.with_index } == "NoMethodError undefined method `with_index' for 1..:Range" &&
        s180err { r.with_object([]) } == "NoMethodError undefined method `with_object' for 1..:Range" &&
        s180err { (v[0]..v[3]).with_index } == "NoMethodError undefined method `with_index' for 1..Infinity:Range" &&
        s180err { (v[0]..v[4]).with_object([]) } == "NoMethodError undefined method `with_object' for 1..3:Range")
  a = [v[0], v[1], v[4], v[6], v[5]]
  check("st180f", r.lazy.take(v[4]).size == 3 && r.lazy.map { |x| x }.take(v[6]).size == 5 && r.lazy.take(v[6]).take(v[4]).size == 3 &&
        r.lazy.select { |x| x }.take(v[4]).size.nil? && a.lazy.take(v[1]).size == 2 && a.lazy.take(0).size == 0 &&
        a.lazy.drop(v[1]).size == 3 && a.lazy.each_slice(v[1]).size == 3 && a.lazy.each_cons(v[1]).size == 4 &&
        a.lazy.map { |x| x }.size == 5 && a.lazy.uniq.size.nil? && (v[0]..v[5]).lazy.size == 10 && (v[0]...v[5]).lazy.take(v[6]).size == 5 &&
        r.lazy.take(v[6]).each_slice(v[1]).size == 3 && v[0].step.lazy.take(v[4]).size == 3 && c.lazy.take(v[4]).size == 3 &&
        a.each_slice(v[1]).lazy.take(1).size == 1 && s180err { r.lazy.each_slice(v[1]).size } == "FloatDomainError Infinity")
  q = v[0].step(v[5], v[4])
  check("st180g", q.method(:last).arity == -1 && q.method(:first).arity == -1 && q.method(:begin).arity == 0 &&
        q.method(:==).arity == 1 && [v[0]].method(:push).arity == -1 && v[0].method(:+).arity == 1 && "a".method(:tr).arity == 2 &&
        "a".method(:upcase).arity == -1 && [v[0]].method(:reverse_each).arity == 0 && {a: v[0]}.method(:fetch).arity == -1 &&
        r.method(:cover?).arity == 1 && "abc".method(:tr).curry["a"]["x"] == "xbc" && q.method(:last).call(2) == [7, 10])
  g = (v[0]..v[1])
  check("st180h", !g.equal?(v[0]..v[1]) && g.equal?(g) && !g.equal?(g.dup) && !g.equal?(g.clone) && g.dup == g &&
        g.dup.class == Range && g.clone.inspect == "1..2" && g.itself.equal?(g))
end
# ===== SECTION 181: blockless take_while / drop_while consumers, non-numeric Range sizes, Method of a Proc, 2.6's surface, Range#freeze =====
# A blockless take_while / drop_while Enumerator stops at the first nil its
# consumer's block answers, so first / next / to_a / any Enumerable method sees
# only the first value; #each and with_index / with_object given a block hand
# the method the caller's answers, and the first op on its Lazy sees them all.
# An Enumerator made from a String range, or a chunk_while / slice_when /
# slice_before / slice_after / chunk one, has a nil #size; a Float range's Lazy
# has the range's size and raises only once a value is pulled. Proc#method and
# Method#method are Methods, and one of an operator runs as #send does. MRI 2.6
# has no tally or Symbol#name, Range / Enumerator no index, and Range#freeze /
# #clone keep the frozen flag. /usr/bin/ruby 2.6 settles each.
def s181err
  yield
rescue => e
  "#{e.class} #{e.message}"
end
def s181
  v = [1, 2, 3, 4, "a", "c", "e", 1.0, 3.0, 3.5, 0]
  a = [v[0], v[1], v[2]]
  e = a.take_while
  check("tw181a", a.take_while.first(3) == [1] && a.take_while.first == 1 && e.next == 1 &&
        s181err { e.next } == "StopIteration iteration reached an end" && a.take_while.to_a == [1] &&
        a.take_while.map { |x| x } == [1] && a.take_while.count == 1 && a.take_while.each { |x| x < 3 } == [1, 2] &&
        a.take_while.with_index { |x, i| x < 3 } == [1, 2] && a.take_while.with_index.to_a == [[1, 0]] &&
        a.take_while.with_object([]) { |x, m| m << x; x < 3 } == [1, 2, 3] && a.take_while.zip(a) == [[1, 1]] &&
        a.drop_while.first(3) == [1] && a.drop_while.with_index { |x, i| x < 2 } == [2, 3] && (v[0]..v[2]).take_while.first(3) == [1] &&
        a.take_while.lazy.to_a == [1] && a.take_while.lazy.map { |x| x * 2 }.to_a == [2, 4, 6] &&
        a.take_while.lazy.select { |x| x > 1 }.to_a == [2, 3] && a.find.first(3) == [1, 2, 3] && a.take_while.size.nil?)
  s = (v[4]..v[6])
  check("sr181b", (v[4]..v[5]).lazy.size.nil? && s.each_slice(2).size.nil? && s.each.size.nil? && s.map.size.nil? &&
        s.each_cons(2).size.nil? && s.lazy.map { |x| x }.size.nil? && s.lazy.take(2).size.nil? && s.each.lazy.size.nil? &&
        s.map.with_index.size.nil? && s.each_slice(2).to_a == [["a", "b"], ["c", "d"], ["e"]] && s.lazy.first(2) == ["a", "b"] &&
        (v[4]..v[5]).lazy.map { |x| x * 2 }.to_a == ["aa", "bb", "cc"] && (v[4]...v[5]).lazy.to_a == ["a", "b"] &&
        (v[0]..v[2]).each.lazy.size == 3 && a.find.lazy.size.nil?)
  f = (v[7]..v[8])
  check("fr181c", f.lazy.size == 3 && f.lazy.take(2).size == 2 && f.lazy.map { |x| x }.size == 3 && f.lazy.select { |x| x }.size.nil? &&
        (v[7]...v[9]).lazy.size == 3 && (v[7]...v[8]).lazy.size == 2 && (v[7]..).lazy.size == Float::INFINITY &&
        (v[9]..v[7]).lazy.size == 0 && f.lazy.each_slice(2).size == 2 && f.lazy.take(0).to_a == [] &&
        f.lazy.inspect == "#<Enumerator::Lazy: 1.0..3.0>" && s181err { f.lazy.to_a } == "TypeError can't iterate from Float" &&
        s181err { f.lazy.first } == "TypeError can't iterate from Float" && (v[7]...v[9]).size == 3 &&
        (v[0]...v[9]).to_a == [1, 2, 3] && (v[0]...v[9]).size == 3)
  b = [v[0], v[1], v[3]]
  check("gn181d", b.chunk_while { |x, y| y == x + 1 }.class == Enumerator && b.chunk_while { |x, y| y == x + 1 }.size.nil? &&
        b.slice_when { |x, y| y != x + 1 }.size.nil? && b.slice_before { |x| x > 1 }.size.nil? && b.slice_after { |x| x > 1 }.size.nil? &&
        b.chunk { |x| x > 1 }.size.nil? && b.chunk_while { |x, y| y == x + 1 }.next == [1, 2] &&
        b.chunk_while { |x, y| y == x + 1 }.with_index.to_a == [[[1, 2], 0], [[4], 1]] && b.chunk_while { |x, y| true }.map.size.nil? &&
        b.chunk_while { |x, y| y == x + 1 }.each { |x| x }.nil? && b.chunk { |x| x > 1 }.each { |x| x }.nil? &&
        (v[0]..v[3]).slice_when { |x, y| true }.size.nil? && b.chunk_while { |x, y| y == x + 1 }.to_a == [[1, 2], [4]])
  check("mm181e", proc {}.method(:call).class == Method && proc { |x| x * 2 }.method(:call).call(4) == 8 &&
        lambda { |x| x + 1 }.method(:call).call(4) == 5 && v[0].method(:+).method(:call).call(4) == 5 &&
        v[0].method(:+).method(:arity).call == 1 && s181err { proc {}.method(:nope) } == "NameError undefined method `nope' for class `Proc'" &&
        v[0].method(:+).call(2) == 3 && v[3].method(:<).call(5) && v[3].method(:<=>).call(5) == -1 && 1.5.method(:<).call(2) &&
        b.map(&v[10].method(:+)) == [1, 2, 4] && b.select(&v[1].method(:<)) == [4] && "a".method(:+).call("b") == "ab" &&
        [v[0]].method(:<<).call(2) == [1, 2])
  check("tl181f", ![v[0]].respond_to?(:tally) && !{}.respond_to?(:tally) && !(v[0]..v[1]).respond_to?(:tally) &&
        ![v[0]].each.respond_to?(:tally) && !v[0].step(3).respond_to?(:tally) && !:a.respond_to?(:name) &&
        s181err { [v[0]].method(:tally) } == "NameError undefined method `tally' for class `Array'" &&
        s181err { [v[0]].tally } == "NoMethodError undefined method `tally' for [1]:Array" &&
        s181err { (v[0]..v[1]).tally } == "NoMethodError undefined method `tally' for 1..2:Range" &&
        s181err { :a.name } == "NoMethodError undefined method `name' for :a:Symbol" &&
        s181err { (v[0]..v[2]).index } == "NoMethodError undefined method `index' for 1..3:Range" &&
        s181err { (v[0]..v[2]).rindex(1) } == "NoMethodError undefined method `rindex' for 1..3:Range" &&
        s181err { [v[0]].cycle(2).index } == "NoMethodError undefined method `index' for #<Enumerator: [1]:cycle(2)>" &&
        s181err { [v[0]].each.index(1) } == "NoMethodError undefined method `index' for #<Enumerator: [1]:each>" &&
        (v[0]..v[2]).find_index.class == Enumerator)
  r = (v[0]..v[2])
  check("fz181g", !(v[0]..v[2]).frozen? && r.freeze.equal?(r) && r.frozen? && r.clone.frozen? && !r.dup.frozen? &&
        !Range.new(v[0], v[2]).frozen? && !(v[0]..v[2]).clone.frozen? && r == (v[0]..v[2]) && r.hash == (v[0]..v[2]).hash &&
        r.to_a == [1, 2, 3] && { r => 1 }[v[0]..v[2]] == 1 && (v[4]..v[5]).freeze.frozen?)
end

# ===== SECTION 182: unary operators by name, Integer shifts, Hash#index, Float range Enumerators, Range#size =====
# -@ / +@ on every Numeric and ~ on an Integer are methods that send, public_send
# and method reach by name; Integer responds to << >> ~ and its Method of one
# runs it. MRI 2.6's Hash#index is #key (and warns on stderr). A Float range
# cannot be walked, but its blockless #each and Enumerable methods answer an
# Enumerator sized from the range that raises only when iterated. Range#size is
# nil for a non-numeric range, whose #count and for-in still walk it, and Float
# Infinity for an endless or infinite one. /usr/bin/ruby 2.6 settles each.
def s182err
  yield
rescue => e
  "#{e.class} #{e.message}"
end
def s182
  v = [5, 5.5, 2**70, Rational(1, 2), 1.0, 3.0, "a", "c", 1, nil, Float::INFINITY, 2]
  n = v[0]
  check("un182a", n.send(:-@) == -5 && n.send(:+@) == 5 && n.send(:~) == -6 && v[1].send(:-@) == -5.5 &&
        v[1].public_send(:+@) == 5.5 && n.method(:-@).call == -5 && n.method(:~).call == -6 && v[2].send(:~) == -(2**70) - 1 &&
        v[2].send(:-@) == -(2**70) && v[3].send(:-@) == Rational(-1, 2) && n.__send__(:-@).class == Integer &&
        s182err { v[1].send(:~) }.start_with?("NoMethodError undefined method `~' for 5.5"))
  check("un182b", n.respond_to?(:<<) && n.respond_to?(:>>) && n.respond_to?(:~) && v[2].respond_to?(:~) && !v[1].respond_to?(:~) &&
        !v[1].respond_to?(:<<) && n.method(:<<).call(1) == 10 && n.method(:>>).call(1) == 2 && n.method(:>>).arity == 1 &&
        n.method(:~).arity == 0 && n.method(:<<).owner == Integer &&
        Integer.instance_methods(false).select { |m| [:<<, :>>, :~].include?(m) }.sort == [:<<, :>>, :~])
  h = {a: v[8], b: v[11]}
  check("hi182c", h.index(v[11]) == :b && h.index(v[0]).nil? && h&.index(v[8]) == :a && h.respond_to?(:index) &&
        h.method(:index).arity == 1 && Hash.instance_methods(false).include?(:index))
  r = (v[4]..v[5])
  x = (v[4]...v[5])
  check("fr182d", r.each.size == 3 && r.map.size == 3 && r.each_slice(v[11]).size == 2 && r.each_cons(v[11]).size == 2 &&
        x.each.size == 2 && r.select.size == 3 && r.sort_by.size == 3 && r.group_by.size == 3 && r.find.size.nil? &&
        r.cycle.size == v[10] && r.cycle(v[11]).size == 6 && (v[5]..v[4]).cycle.size == 0 && r.each_with_index.size == 3 &&
        r.each.inspect == "#<Enumerator: 1.0..3.0:each>" && r.each_slice(v[11]).inspect == "#<Enumerator: 1.0..3.0:each_slice(2)>" &&
        r.cycle(v[11]).inspect == "#<Enumerator: 1.0..3.0:cycle(2)>" && r.map.inspect == "#<Enumerator: 1.0..3.0:map>")
  made = []
  check("fr182e", s182err { e = r.each; made << e.size; e.to_a } == "TypeError can't iterate from Float" &&
        s182err { e = r.map; made << e.size; e.next } == "TypeError can't iterate from Float" &&
        s182err { e = r.sort_by; made << e.size; e.each { |y| y } } == "TypeError can't iterate from Float" &&
        s182err { e = r.each_slice(v[11]); made << e.size; e.first } == "TypeError can't iterate from Float" &&
        s182err { r.each { |y| y } } == "TypeError can't iterate from Float" && made == [3, 3, 3, 2])
  s = (v[6]..v[7])
  check("rs182f", s.size.nil? && s.send(:size).nil? && s.count == 3 && (v[6]..v[9]).size.nil? && (v[4]..v[9]).size == v[10] &&
        (v[4]..v[9]).size.class == Float && (v[8]..v[9]).size == v[10] && (v[8]..v[9]).size.class == Float &&
        (v[8]..v[10]).size == v[10] && (v[4]..v[10]).size == v[10] && r.size == 3 && (v[8]..v[0]).size == 5 &&
        s&.size.nil? && (v[4]..v[9]).each.size == v[10])
  fo = []
  for y in s do fo << y end
  check("rs182g", fo == ["a", "b", "c"] && [*s] == ["a", "b", "c"] && s.to_a.size == 3 && s.each.size.nil? && s.size.nil?)
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
s148() # SECTION-CALL 148
s149() # SECTION-CALL 149
s150() # SECTION-CALL 150
s151() # SECTION-CALL 151
s152() # SECTION-CALL 152
s153() # SECTION-CALL 153
s154() # SECTION-CALL 154
s155() # SECTION-CALL 155
s156() # SECTION-CALL 156
s157() # SECTION-CALL 157
s158() # SECTION-CALL 158
s159() # SECTION-CALL 159
s160() # SECTION-CALL 160
s161() # SECTION-CALL 161
s162() # SECTION-CALL 162
s163() # SECTION-CALL 163
s164() # SECTION-CALL 164
s165() # SECTION-CALL 165
s166() # SECTION-CALL 166
s167() # SECTION-CALL 167
s168() # SECTION-CALL 168
s169() # SECTION-CALL 169
s170() # SECTION-CALL 170
s171() # SECTION-CALL 171
s172() # SECTION-CALL 172
s173() # SECTION-CALL 173
s174() # SECTION-CALL 174
s175() # SECTION-CALL 175
s176() # SECTION-CALL 176
s177() # SECTION-CALL 177
s178() # SECTION-CALL 178
s179() # SECTION-CALL 179
s180() # SECTION-CALL 180
s181() # SECTION-CALL 181
s182() # SECTION-CALL 182
puts "full: #{FULLC[0]} checks, #{FULLC[1]} failures"
exit(FULLC[1])
