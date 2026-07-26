package abnf

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Byte order marks, as they appear at the head of a file. U+FEFF encoded in
// each encoding; spelled as escapes on purpose, since written literally a BOM
// is an invisible character that no reviewer can see and some editors eat.
const (
	bomUTF8    = "\uFEFF"   // EF BB BF
	bomUTF16BE = "\xfe\xff" // FE FF
	bomUTF16LE = "\xff\xfe" // FF FE
	bomUTF32BE = "\x00\x00\xfe\xff"
	bomUTF32LE = "\xff\xfe\x00\x00"
)

// StripBOM turns source text that begins with a byte order mark into plain
// UTF-8: the mark is removed, and if it announced UTF-16 or UTF-32 the text is
// transcoded. Text without a BOM is returned untouched.
//
// A BOM states an encoding; it is not a character of the program, and no
// grammar has a production for it, so left in place it makes the very first
// token unparsable - the parse dies at line 1 column 1 with no hint that a few
// invisible bytes are the reason. Real-world code carries marks routinely
// (Windows editors write them by default): 588 files of Microsoft's own
// TypeScript conformance suite open with a UTF-8 one.
//
// Everything downstream - the parser, SetTraceSource, c.lineOf() - assumes
// UTF-8, so every source text is normalized as it is read, before any of them
// sees it. They then all measure the same bytes, and a column reported on line
// 1 counts from the first real character, which is what an editor shows anyway.
//
// A file in UTF-16/UTF-32 with NO mark cannot be recognized from its content
// without guessing, so it is left alone; a BOM is the only reliable signal, and
// in practice those encodings are near-always written with one.
func StripBOM(src string) string {
	// UTF-32LE must be tested before UTF-16LE: its mark starts with the same
	// two bytes, so the shorter prefix would otherwise swallow it.
	switch {
	case strings.HasPrefix(src, bomUTF8):
		return src[len(bomUTF8):]
	case strings.HasPrefix(src, bomUTF32LE):
		return decodeUTF32(src[len(bomUTF32LE):], binary.LittleEndian)
	case strings.HasPrefix(src, bomUTF32BE):
		return decodeUTF32(src[len(bomUTF32BE):], binary.BigEndian)
	case strings.HasPrefix(src, bomUTF16LE):
		return decodeUTF16(src[len(bomUTF16LE):], binary.LittleEndian)
	case strings.HasPrefix(src, bomUTF16BE):
		return decodeUTF16(src[len(bomUTF16BE):], binary.BigEndian)
	}
	return src
}

// decodeUTF16 re-encodes UTF-16 code units (BOM already removed) as UTF-8.
// utf16.Decode joins surrogate pairs and maps an unpaired half to U+FFFD. A
// trailing odd byte is a truncated unit and is dropped.
func decodeUTF16(s string, order binary.ByteOrder) string {
	units := make([]uint16, 0, len(s)/2)
	for i := 0; i+1 < len(s); i += 2 {
		units = append(units, order.Uint16([]byte(s[i:i+2])))
	}
	return string(utf16.Decode(units))
}

// decodeUTF32 re-encodes UTF-32 code points (BOM already removed) as UTF-8.
// Anything outside Unicode or inside the surrogate range cannot be encoded and
// becomes U+FFFD, matching what utf16.Decode does with a lone surrogate.
func decodeUTF32(s string, order binary.ByteOrder) string {
	var b strings.Builder
	b.Grow(len(s) / 2)
	for i := 0; i+3 < len(s); i += 4 {
		r := rune(order.Uint32([]byte(s[i : i+4])))
		if r > utf8.MaxRune || (r >= 0xD800 && r <= 0xDFFF) {
			r = utf8.RuneError
		}
		b.WriteRune(r)
	}
	return b.String()
}
