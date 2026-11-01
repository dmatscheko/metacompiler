package abnf

import (
	"fmt"
	"strings"
)

// The codecs and error handlers of str.encode and bytes.decode. Twin of
// languages/lib/python-codecs.js, which states the rules.

// pycdNames maps each implemented codec to its normalized names and aliases.
var pycdNames = []struct {
	codec string
	names []string
}{
	{"utf-8", []string{"utf_8", "u8", "utf", "utf8", "utf8_ucs2", "utf8_ucs4", "cp65001"}},
	{"ascii", []string{"ascii", "646", "ansi_x3_4_1968", "ansi_x3_4_1986", "cp367", "csascii", "ibm367", "iso646_us",
		"iso_646_irv_1991", "iso_ir_6", "us", "us_ascii"}},
	{"latin-1", []string{"latin_1", "8859", "cp819", "csisolatin1", "ibm819", "iso8859", "iso8859_1", "iso_8859_1",
		"iso_8859_1_1987", "iso_ir_100", "l1", "latin", "latin1"}},
	{"utf-16", []string{"utf_16", "u16", "utf16"}},
	{"utf-16-le", []string{"utf_16_le", "utf_16le", "unicodelittleunmarked"}},
	{"utf-16-be", []string{"utf_16_be", "utf_16be", "unicodebigunmarked"}},
	{"utf-32", []string{"utf_32", "u32", "utf32"}},
	{"utf-32-le", []string{"utf_32_le", "utf_32le"}},
	{"utf-32-be", []string{"utf_32_be", "utf_32be"}},
	{"cp1252", []string{"cp1252", "1252", "windows_1252"}},
}

// pycdCp1252 is cp1252's bytes 0x80..0x9f; 0 where the byte is undefined.
var pycdCp1252 = [32]int{0x20ac, 0, 0x201a, 0x192, 0x201e, 0x2026, 0x2020, 0x2021, 0x2c6, 0x2030, 0x160,
	0x2039, 0x152, 0, 0x17d, 0, 0, 0x2018, 0x2019, 0x201c, 0x201d, 0x2022, 0x2013, 0x2014, 0x2dc, 0x2122,
	0x161, 0x203a, 0x153, 0, 0x17e, 0x178}

func pycdLookup(enc string) string {
	var n strings.Builder
	punct := false
	for _, u := range strUnits(strings.ToLower(enc)) {
		if (u >= 'a' && u <= 'z') || (u >= '0' && u <= '9') || u == '.' {
			if punct && n.Len() > 0 {
				n.WriteByte('_')
			}
			n.WriteByte(byte(u))
			punct = false
		} else {
			punct = true
		}
	}
	norm := n.String()
	dotless := strings.ReplaceAll(norm, ".", "_")
	for _, c := range pycdNames {
		for _, a := range c.names {
			if a == norm || a == dotless {
				return c.codec
			}
		}
	}
	return ""
}

func pycdCp1252Byte(cp int) int {
	if cp < 0x80 || (cp >= 0xa0 && cp < 0x100) {
		return cp
	}
	for i, c := range pycdCp1252 {
		if c == cp && cp != 0 {
			return 0x80 + i
		}
	}
	return -1
}

func pycdPushUnit(out []int, u, size int, le bool) []int {
	for j := 0; j < size; j++ {
		sh := 8 * j
		if !le {
			sh = 8 * (size - 1 - j)
		}
		out = append(out, u>>sh&0xff)
	}
	return out
}

func pycdEncBad(codec string, cp int) bool {
	switch codec {
	case "ascii":
		return cp >= 0x80
	case "latin-1":
		return cp >= 0x100
	case "cp1252":
		return pycdCp1252Byte(cp) < 0
	}
	return cp >= 0xd800 && cp <= 0xdfff
}

func pycdEncOne(out []int, codec string, cp, size int, le bool) []int {
	switch {
	case size == 4:
		return pycdPushUnit(out, cp, 4, le)
	case size == 2 && cp >= 0x10000:
		out = pycdPushUnit(out, 0xd800+(cp-0x10000)>>10, 2, le)
		return pycdPushUnit(out, 0xdc00+(cp-0x10000)&0x3ff, 2, le)
	case size == 2:
		return pycdPushUnit(out, cp, 2, le)
	case codec == "cp1252":
		return append(out, pycdCp1252Byte(cp))
	case codec != "utf-8" || cp < 0x80:
		return append(out, cp)
	case cp < 0x800:
		return append(out, 0xc0|cp>>6, 0x80|cp&63)
	case cp < 0x10000:
		return append(out, 0xe0|cp>>12, 0x80|cp>>6&63, 0x80|cp&63)
	}
	return append(out, 0xf0|cp>>18, 0x80|cp>>12&63, 0x80|cp>>6&63, 0x80|cp&63)
}

// pycdEncode encodes the code points cps; a non-empty exc names the exception
// to raise with msg.
func pycdEncode(cps []int, codec, errs string) (out []int, exc, msg string) {
	size, le := 0, codec != "utf-16-be" && codec != "utf-32-be"
	switch codec {
	case "utf-16", "utf-16-le", "utf-16-be":
		size = 2
	case "utf-32", "utf-32-le", "utf-32-be":
		size = 4
	}
	if codec == "utf-16" || codec == "utf-32" {
		out = pycdPushUnit(out, 0xfeff, size, true)
	}
	why, errCodec := "surrogates not allowed", codec
	switch codec {
	case "ascii":
		why = "ordinal not in range(128)"
	case "latin-1":
		why = "ordinal not in range(256)"
	case "cp1252":
		why, errCodec = "character maps to <undefined>", "charmap"
	}
	for k := 0; k < len(cps); {
		if !pycdEncBad(codec, cps[k]) {
			out = pycdEncOne(out, codec, cps[k], size, le)
			k++
			continue
		}
		j := k + 1
		for size == 0 && j < len(cps) && pycdEncBad(codec, cps[j]) {
			j++
		}
		bad := errs == "strict" || (errs == "surrogateescape" && size > 0)
		if errs == "surrogateescape" && size == 0 {
			q := k
			for q < j && cps[q] >= 0xdc80 && cps[q] <= 0xdcff {
				q++
			}
			if q < j {
				if codec != "cp1252" {
					k = q
				}
				bad = true
			}
		}
		if bad {
			where := fmt.Sprintf("character '%s' in position %d", pysEncEsc(cps[k]), k)
			if j-k > 1 {
				where = fmt.Sprintf("characters in position %d-%d", k, j-1)
			}
			return nil, "UnicodeEncodeError", "'" + errCodec + "' codec can't encode " + where + ": " + why
		}
		for _, c := range cps[k:j] {
			rep := ""
			switch errs {
			case "replace":
				rep = "?"
			case "backslashreplace":
				rep = pysEncEsc(c)
			case "xmlcharrefreplace":
				rep = fmt.Sprintf("&#%d;", c)
			case "surrogateescape":
				out = append(out, c-0xdc00)
			case "ignore":
			default:
				return nil, "LookupError", "unknown error handler name '" + errs + "'"
			}
			for i := 0; i < len(rep); i++ {
				out = pycdEncOne(out, codec, int(rep[i]), size, le)
			}
		}
		k = j
	}
	return out, "", ""
}

// pycdDecode decodes the byte values bs into UTF-16 code units; a non-empty
// exc names the exception to raise with msg.
func pycdDecode(bs []int, codec, errs string) (out []uint16, exc, msg string) {
	push := func(cp int) {
		if cp >= 0x10000 {
			out = append(out, uint16(0xd800+(cp-0x10000)>>10), uint16(0xdc00+(cp-0x10000)&0x3ff))
		} else {
			out = append(out, uint16(cp))
		}
	}
	// bad runs the handler on bs[from:to] and answers where decoding resumes;
	// exc is set when it raised.
	bad := func(name string, from, to int, why string) int {
		esc := 0
		if errs == "surrogateescape" {
			for esc < 4 && from+esc < to && bs[from+esc] >= 0x80 {
				push(0xdc00 + bs[from+esc])
				esc++
			}
			if esc > 0 {
				return from + esc
			}
		}
		switch errs {
		case "strict", "surrogateescape":
			where := fmt.Sprintf("byte 0x%02x in position %d", bs[from], from)
			if to-from > 1 {
				where = fmt.Sprintf("bytes in position %d-%d", from, to-1)
			}
			exc, msg = "UnicodeDecodeError", "'"+name+"' codec can't decode "+where+": "+why
		case "ignore":
		case "replace":
			push(0xfffd)
		case "backslashreplace":
			for _, c := range bs[from:to] {
				for _, r := range fmt.Sprintf("\\x%02x", c) {
					push(int(r))
				}
			}
		case "xmlcharrefreplace", "namereplace":
			exc, msg = "TypeError", "don't know how to handle UnicodeDecodeError in error callback"
		default:
			exc, msg = "LookupError", "unknown error handler name '"+errs+"'"
		}
		return to
	}
	n := len(bs)
	switch codec {
	case "latin-1":
		for _, c := range bs {
			push(c)
		}
		return out, "", ""
	case "ascii", "cp1252":
		for i, c := range bs {
			switch {
			case codec == "ascii" && c >= 0x80:
				if bad("ascii", i, i+1, "ordinal not in range(128)"); exc != "" {
					return nil, exc, msg
				}
			case codec == "cp1252" && c >= 0x80 && c < 0xa0 && pycdCp1252[c-0x80] == 0:
				if bad("charmap", i, i+1, "character maps to <undefined>"); exc != "" {
					return nil, exc, msg
				}
			case codec == "cp1252" && c >= 0x80 && c < 0xa0:
				push(pycdCp1252[c-0x80])
			default:
				push(c)
			}
		}
		return out, "", ""
	case "utf-8":
		for i := 0; i < n; {
			b0 := bs[i]
			need := 0
			switch {
			case b0 < 0x80:
				push(b0)
				i++
				continue
			case b0 >= 0xc2 && b0 <= 0xdf:
				need = 1
			case b0 >= 0xe0 && b0 <= 0xef:
				need = 2
			case b0 >= 0xf0 && b0 <= 0xf4:
				need = 3
			}
			why, to := "", i+1
			if need == 0 {
				why = "invalid start byte"
			}
			cp := b0 & (0x3f >> need)
			for k := 1; why == "" && k <= need; k++ {
				if i+k >= n {
					why, to = "unexpected end of data", n
					break
				}
				c := bs[i+k]
				lo, hi := 0x80, 0xbf
				if k == 1 {
					switch b0 {
					case 0xe0:
						lo = 0xa0
					case 0xed:
						hi = 0x9f
					case 0xf0:
						lo = 0x90
					case 0xf4:
						hi = 0x8f
					}
				}
				if c < lo || c > hi {
					why, to = "invalid continuation byte", i+k
					break
				}
				cp = cp<<6 | c&0x3f
			}
			if why != "" {
				if i = bad("utf-8", i, to, why); exc != "" {
					return nil, exc, msg
				}
				continue
			}
			push(cp)
			i += need + 1
		}
		return out, "", ""
	}
	size := 2
	if strings.HasPrefix(codec, "utf-32") {
		size = 4
	}
	le := codec != "utf-16-be" && codec != "utf-32-be"
	unit := func(i, sz int) int {
		v := 0
		for j := 0; j < sz; j++ {
			b := bs[i+j]
			if le {
				b = bs[i+sz-1-j]
			}
			v = v<<8 | b
		}
		return v
	}
	i := 0
	if (codec == "utf-16" && n >= 2) || (codec == "utf-32" && n >= 4) {
		le = true
		switch bom := unit(0, size); {
		case bom == 0xfeff:
			i = size
		case (size == 2 && bom == 0xfffe) || (size == 4 && bom == 0xfffe0000):
			i, le = size, false
		}
	}
	name := "utf-16-be"
	switch {
	case size == 2 && le:
		name = "utf-16-le"
	case size == 4 && le:
		name = "utf-32-le"
	case size == 4:
		name = "utf-32-be"
	}
	for i < n {
		switch {
		case n-i < size:
			i = bad(name, i, n, "truncated data")
		case size == 4:
			switch u := unit(i, 4); {
			case u >= 0xd800 && u <= 0xdfff:
				i = bad(name, i, i+4, "code point in surrogate code point range(0xd800, 0xe000)")
			case u >= 0x110000:
				i = bad(name, i, i+4, "code point not in range(0x110000)")
			default:
				push(u)
				i += 4
			}
		default:
			switch u := unit(i, 2); {
			case u < 0xd800 || u > 0xdfff:
				push(u)
				i += 2
			case u >= 0xdc00:
				i = bad(name, i, i+2, "illegal encoding")
			case n-i < 4:
				i = bad(name, i, n, "unexpected end of data")
			default:
				if u2 := unit(i+2, 2); u2 >= 0xdc00 && u2 <= 0xdfff {
					out = append(out, uint16(u), uint16(u2))
					i += 4
				} else {
					i = bad(name, i, i+2, "illegal UTF-16 surrogate")
				}
			}
		}
		if exc != "" {
			return nil, exc, msg
		}
	}
	return out, "", ""
}
