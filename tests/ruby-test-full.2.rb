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
puts "full: #{FULLC[0]} checks, #{FULLC[1]} failures"
exit(FULLC[1])
