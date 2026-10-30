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

s122() # SECTION-CALL 122
s123() # SECTION-CALL 123
s124() # SECTION-CALL 124
puts "full: #{FULLC[0]} checks, #{FULLC[1]} failures"
exit(FULLC[1])
