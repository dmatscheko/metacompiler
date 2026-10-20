// Package unicode: Go's unicode package WRITTEN IN GO, compiled with a program
// that imports "unicode" (core.importFile in both go grammars).
//
// Case mapping is package strings' (one rune at a time, so the same coverage), and
// IsSpace is TrimSpace's test. The category tests are exact for ASCII and Latin-1;
// beyond U+00FF they read a letter as a rune with a case mapping or one of the
// large uncased scripts below, and a digit, punctuation or mark from the main
// blocks only - Go's full category tables are not here, nor the RangeTable API.
package unicode

import "strings"

const (
	MaxRune         = '\U0010FFFF'
	ReplacementChar = '�'
	MaxASCII        = '\u007F'
	MaxLatin1       = 'ÿ'
)

const (
	UpperCase = iota
	LowerCase
	TitleCase
	MaxCase
)

func ToUpper(r rune) rune {
	if r < 0x80 {
		if 'a' <= r && r <= 'z' {
			r -= 'a' - 'A'
		}
		return r
	}
	return firstRune(strings.ToUpper(string(r)), r)
}

func ToLower(r rune) rune {
	if r < 0x80 {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r
	}
	return firstRune(strings.ToLower(string(r)), r)
}

func ToTitle(r rune) rune {
	if r < 0x80 {
		return ToUpper(r)
	}
	return firstRune(strings.ToTitle(string(r)), r)
}

func To(_case int, r rune) rune {
	switch _case {
	case UpperCase:
		return ToUpper(r)
	case LowerCase:
		return ToLower(r)
	case TitleCase:
		return ToTitle(r)
	}
	return ReplacementChar
}

func firstRune(s string, r rune) rune {
	for _, c := range s {
		return c
	}
	return r
}

func IsSpace(r rune) bool {
	if r < 0 || r > MaxRune || (r >= 0xD800 && r <= 0xDFFF) {
		return false
	}
	return strings.TrimSpace(string(r)) == ""
}

func IsUpper(r rune) bool {
	if r <= MaxLatin1 {
		return 'A' <= r && r <= 'Z' || 0xC0 <= r && r <= 0xDE && r != 0xD7
	}
	return ToLower(r) != r && ToUpper(r) == r
}

func IsLower(r rune) bool {
	if r <= MaxLatin1 {
		return 'a' <= r && r <= 'z' || r == 0xB5 || 0xDF <= r && r <= 0xFF && r != 0xF7
	}
	return ToUpper(r) != r && ToLower(r) == r
}

func IsTitle(r rune) bool {
	if r <= MaxLatin1 {
		return false
	}
	return ToTitle(r) == r && ToUpper(r) != r
}

// Uncased letter blocks: Hebrew, Arabic, Devanagari, Thai, Hiragana, Katakana,
// CJK ideographs, Hangul.
var letterRanges = [][2]rune{
	{0x05D0, 0x05EA}, {0x0620, 0x064A}, {0x0671, 0x06D3}, {0x0904, 0x0939}, {0x0E01, 0x0E30},
	{0x3041, 0x3096}, {0x30A1, 0x30FA}, {0x3400, 0x4DBF}, {0x4E00, 0x9FFF}, {0xAC00, 0xD7A3},
	{0xF900, 0xFAFF}, {0x20000, 0x2A6DF},
}

func inRanges(r rune, rs [][2]rune) bool {
	for _, x := range rs {
		if x[0] <= r && r <= x[1] {
			return true
		}
	}
	return false
}

func IsLetter(r rune) bool {
	if r <= MaxLatin1 {
		return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == 0xAA || r == 0xB5 || r == 0xBA ||
			0xC0 <= r && r != 0xD7 && r != 0xF7
	}
	return ToUpper(r) != r || ToLower(r) != r || inRanges(r, letterRanges)
}

// Decimal digit blocks (Nd): Arabic-Indic, Extended Arabic-Indic, Devanagari,
// Bengali, Thai, fullwidth.
var digitRanges = [][2]rune{
	{0x0660, 0x0669}, {0x06F0, 0x06F9}, {0x0966, 0x096F}, {0x09E6, 0x09EF}, {0x0E50, 0x0E59}, {0xFF10, 0xFF19},
}

func IsDigit(r rune) bool {
	if r <= MaxLatin1 {
		return '0' <= r && r <= '9'
	}
	return inRanges(r, digitRanges)
}

func IsNumber(r rune) bool {
	if r <= MaxLatin1 {
		return '0' <= r && r <= '9' || r == 0xB2 || r == 0xB3 || r == 0xB9 || 0xBC <= r && r <= 0xBE
	}
	return IsDigit(r) || 0x2150 <= r && r <= 0x2189 || 0x2460 <= r && r <= 0x249B
}

var punctRanges = [][2]rune{
	{0x2010, 0x2027}, {0x2030, 0x2043}, {0x2045, 0x2051}, {0x2053, 0x205E}, {0x3001, 0x3003}, {0x3008, 0x3011},
}

func IsPunct(r rune) bool {
	if r <= MaxLatin1 {
		switch r {
		case '!', '"', '#', '%', '&', '\'', '(', ')', '*', ',', '-', '.', '/', ':', ';', '?', '@',
			'[', '\\', ']', '_', '{', '}', 0xA1, 0xA7, 0xAB, 0xB6, 0xB7, 0xBB, 0xBF:
			return true
		}
		return false
	}
	return inRanges(r, punctRanges)
}

func IsSymbol(r rune) bool {
	if r <= MaxLatin1 {
		switch r {
		case '$', '+', '<', '=', '>', '^', '`', '|', '~', 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA8, 0xA9, 0xAC, 0xAE,
			0xAF, 0xB0, 0xB1, 0xB4, 0xB8, 0xD7, 0xF7:
			return true
		}
		return false
	}
	return 0x2190 <= r && r <= 0x21FF || 0x2200 <= r && r <= 0x22FF || 0x20A0 <= r && r <= 0x20C0 ||
		0x2600 <= r && r <= 0x27BF || 0x1F300 <= r && r <= 0x1FAFF
}

func IsMark(r rune) bool {
	return 0x0300 <= r && r <= 0x036F || 0x0483 <= r && r <= 0x0489 || 0x0591 <= r && r <= 0x05BD ||
		0x064B <= r && r <= 0x065F || 0x20D0 <= r && r <= 0x20F0 || 0xFE20 <= r && r <= 0xFE2F
}

func IsControl(r rune) bool {
	return 0 <= r && r < 0x20 || 0x7F <= r && r < 0xA0
}

func IsGraphic(r rune) bool {
	return IsLetter(r) || IsMark(r) || IsNumber(r) || IsPunct(r) || IsSymbol(r) || IsSpace(r) && !IsControl(r)
}

func IsPrint(r rune) bool {
	if r == ' ' {
		return true
	}
	return IsLetter(r) || IsMark(r) || IsNumber(r) || IsPunct(r) || IsSymbol(r)
}
