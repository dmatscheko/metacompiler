// python-codecs.js -- str.encode and bytes.decode for CPython 3.14's text
// codecs other than the multibyte CJK ones: utf-8, utf-8-sig, utf-7, ascii,
// latin-1, utf-16 and utf-32 (each with -le and -be), undefined,
// unicode_escape, raw_unicode_escape, punycode, idna (its ASCII labels), and
// every single-byte charmap
// codec (cp437, mac_roman, koi8_r, iso8859_2 ...), with the strict, ignore, replace,
// backslashreplace, xmlcharrefreplace and surrogateescape error handlers, for
// python-interpreter.abnf (include) and languages/lib/python-rt.metajs
// (import). Go twin: abnf/jsrtpycodecs.go.
//
// Neither direction raises: each answers {err, msg, b, s} - err the exception
// class name ("" on success), b the encoded byte values, s the decoded text -
// and the caller raises in its own engine.

// The codec an encoding name selects: lower-cased, every run of characters
// other than ASCII letters, digits and "." collapsed to one "_" (CPython's
// normalize_encoding), then looked up among the codecs' module names and
// aliases (encodings.aliases) by pycdNameIs. "" for a codec not implemented here.
function pycdLookup(enc) {
    var n = ""
    var punct = false
    var low = enc.toLowerCase()
    for (var i = 0; i < low.length; i++) {
        var c = low.charCodeAt(i)
        if ((c >= 97 && c <= 122) || (c >= 48 && c <= 57) || c == 46) {
            if (punct && n != "") { n = n + "_" }
            n = n + String.fromCharCode(c)
            punct = false
        } else {
            punct = true
        }
    }
    var d = ""
    for (var j = 0; j < n.length; j++) {
        if (n.charAt(j) == ".") { d = d + "_" } else { d = d + n.charAt(j) }
    }
    var names = pycdNames
    for (var k = 0; k < names.length; k = k + 2) {
        if (pycdNameIs(names[k + 1], n, d)) { return names[k] }
    }
    var maps = pycdCharmaps
    for (var q = 0; q < maps.length; q = q + 3) {
        if (pycdNameIs(maps[q + 1], n, d)) { return maps[q] }
    }
    return ""
}
// Does the normalized name n (d: its dots made underscores) select the codec
// whose module name is al[0] and whose aliases are the rest? An alias matches n
// or d, the module name only n (encodings.search_function). Twin of pycdNameIs
// in abnf/jsrtpycodecs.go.
function pycdNameIs(al, n, d) {
    if (al[0] == n) { return true }
    for (var m = 1; m < al.length; m++) {
        if (al[m] == n || al[m] == d) { return true }
    }
    return false
}
// "binary" stands for the bytes-to-bytes and str-to-str codecs, which
// str.encode and bytes.decode refuse.
var pycdNames = [
    "utf-8", ["utf_8", "u8", "utf", "utf8", "utf8_ucs2", "utf8_ucs4", "cp65001"],
    "utf-8-sig", ["utf_8_sig"],
    "utf-7", ["utf_7", "u7", "unicode_1_1_utf_7", "utf7"],
    "ascii", ["ascii", "646", "ansi_x3.4_1968", "ansi_x3_4_1968", "ansi_x3.4_1986", "cp367", "csascii", "ibm367",
        "iso646_us", "iso_646.irv_1991", "iso_ir_6", "us", "us_ascii"],
    "latin-1", ["latin_1", "8859", "cp819", "csisolatin1", "ibm819", "iso8859", "iso8859_1", "iso_8859_1",
        "iso_8859_1_1987", "iso_ir_100", "l1", "latin", "latin1"],
    "utf-16", ["utf_16", "u16", "utf16"],
    "utf-16-le", ["utf_16_le", "utf_16le", "unicodelittleunmarked"],
    "utf-16-be", ["utf_16_be", "utf_16be", "unicodebigunmarked"],
    "utf-32", ["utf_32", "u32", "utf32"],
    "utf-32-le", ["utf_32_le", "utf_32le"],
    "utf-32-be", ["utf_32_be", "utf_32be"],
    "undefined", ["undefined"],
    "unicode-escape", ["unicode_escape"],
    "raw-unicode-escape", ["raw_unicode_escape"],
    "punycode", ["punycode"],
    "idna", ["idna"],
    "binary", ["base64_codec", "base64", "base_64"],
    "binary", ["bz2_codec", "bz2"],
    "binary", ["hex_codec", "hex"],
    "binary", ["quopri_codec", "quopri", "quoted_printable", "quotedprintable"],
    "binary", ["rot_13", "rot13"],
    "binary", ["uu_codec", "uu"],
    "binary", ["zlib_codec", "zip", "zlib"]]

// The single-byte charmap codecs: name, names and aliases, and the table of
// what bytes 0..255 decode to, as runs - each a code point in base 36, then
// "." and the run's length in base 36 when longer than one byte - of code
// points rising by one, or of undefined bytes (U+FFFE, as CPython's
// decoding_table spells them). Generated from python3 3.14.
var pycdCharmaps = [
    "cp037", ["cp037", "037", "csibm037", "ebcdic_cp_ca", "ebcdic_cp_nl", "ebcdic_cp_us", "ebcdic_cp_wt", "ibm037", "ibm039"],
        "0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 6f 6p 4i 1a 1o 14 17 3g 12 6h.3 6g 6l.3 6k 67 x 10 16 15 1n 4s 19 1b 5e 5g 5c.2 5f 5h 5j 5t 4m 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 1s 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 3i 37.8 4h 5b 5s 65.2 4u 2m 4j 4l 53 4p 4n 52 58.3 2j 2l 4v 4o 50 5z 3f 1t.9 4t 6s 6u 6q.2 6t 3h 22.9 55 6z.2 6x.2 73 2k 6v 2b.8 4y 5w 5y 5u.2 5x 1c.a 4z 63.2 61.2 4f",
    "cp273", ["cp273", "273", "csibm273", "ibm273"],
        "0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 3f 68.2 6b 6d 6f 6p 5g 1a 1o 14 17 x 12 6h.3 6g 6l.3 6k 3i 64 10 16 15 1n 2m 19 1b 5e 2j 5c.2 5f 5h 5j 5t 6u 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 4n 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 67 37.8 4h 5b 5s 65.2 4u 4i.2 4l 53 4p 1s 52 58.3 4s 3g 6da 4o 50 5z 6c 1t.9 4t 6s 4m 6q.2 6t 70 22.9 55 6z 3h 6x.2 73 5y 6v 2b.8 4y 5w 2k 5u.2 5x 1c.a 4z 63 2l 61.2 4f",
    "cp424", ["cp424", "424", "csibm424", "ebcdic_cp_he", "ibm424"],
        "0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 15c.9 4i 1a 1o 14 17 3g 12 15l.9 x 10 16 15 1n 4s 19 1b 15u.8 4m 18 11 2n 1q.2 1eke 162 1eke.2 4g 1eke.3 6c7 2o 1m z 1s 13 1p y 1eke 2p.9 4r 57 1eke.3 4x 4w 2y.9 1eke.3 54 1eke 4k 51 3i 37.8 1eke.5 4u 2m 4j 4l 53 4p 4n 52 58.3 2j 2l 4v 4o 50 5z 3f 1t.9 4t 1eke.5 3h 22.9 55 1eke.5 2k 6v 2b.8 4y 1eke.5 1c.a 4z 1eke.4 4f",
    "cp437", ["cp437", "437", "cspc8codepage437", "ibm437"],
        "0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 4i.2 4l 6g7 b6 69 6l 6r 6y 6p 5t 4q 56 5b 6xc 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp500", ["cp500", "500", "csibm500", "ebcdic_cp_be", "ebcdic_cp_ch", "ibm500"],
        "0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 6f 6p 2j 1a 1o 14 17 x 12 6h.3 6g 6l.3 6k 67 2l 10 16 15 1n 2m 19 1b 5e 5g 5c.2 5f 5h 5j 5t 4m 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 1s 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 3i 37.8 4h 5b 5s 65.2 4u 4i.2 4l 53 4p 4n 52 58.3 4s 3g 4v 4o 50 5z 3f 1t.9 4t 6s 6u 6q.2 6t 3h 22.9 55 6z.2 6x.2 73 2k 6v 2b.8 4y 5w 5y 5u.2 5x 1c.a 4z 63.2 61.2 4f",
    "cp720", ["cp720"],
        "0.3m 6h 6a 3o 68 3q 6f 6i.2 6g 6n 6m 3x.4 18x.2 6s 4k 18g 6z 6x 17l.4 4j 17p.h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo 186.5 18h 51 18i.9 6sh 18r.6 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp737", ["cp737"],
        "0.3k pd.h pv.7 q9.h qr qq qs.5 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo qx q4.3 qy q7 r0.2 qz r2 p2 p4.3 p8 pa.2 4x 6sl 6sk q2.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp775", ["cp775", "775", "cspc775baltic", "ibm775"],
        "0.3k 7a 70 6h 75 6c 83 6d 7b 8y 7n 9i.2 8b ah 5g.2 5l 6e 5i 99 6u 82 4i 9m.2 5y 64 6w 4j 60 5z 4k 74 8a 6r aj.2 ai 6cd 4m 4p 4u 4s 59 58 8x 4r 57 7f5.3 7b6 7c4 78 7g 7s 7q 7dv 7dd 7dj 7dp 8e 9s 7bk 7bo 7ck 7cc 7bw 7b4 7cs aa a2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 al 79 7h 7t 7r 8f 9t ab a3 am 7bs 7bg 7ew 7es 7f0 7f4 7eo 5v 67 98 8z 6t 5x 51 90 8m.2 8r.2 92 7m 91 6c9 4t 4x 6cc 5a 52 4n 6v 6ce 4w 6qh 53 55 4z 4y 7fk 4g",
    "cp850", ["cp850", "850", "cspc850multilingual", "ibm850"],
        "0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 6w 4j 60 5z b6 69 6l 6r 6y 6p 5t 4q 56 5b 4u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 5d.2 5c 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 6b 5f 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 6o 5s 5m.2 5k 8h 5p.3 7bs 7bg 7ew 7es 4m 5o 7eo 5v 67 5w 5u 6t 5x 51 72 66 62.2 61 71 65 4v 50 4t 4x 6c7 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g",
    "cp852", ["cp852", "852", "cspcp852", "ibm852"],
        "0.3k 5j 70 6h 6a 6c a7 7b 6f 8y 6j 9c.2 6m ah 5g 7a 5l 8p.2 6s 6u 8t.2 9m.2 5y 64 9w.2 8x 5z 7h 69 6l 6r 6y 78.2 al.2 7s.2 4s ai 7g 9r 4r 57 7f5.3 7b6 7c4 5d.2 7u 9q 7dv 7dd 7dj 7dp aj.2 7bk 7bo 7ck 7cc 7bw 7b4 7cs 76.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 7l 7k 7i 5n 7j 93 5p.2 7v 7bs 7bg 7ew 7es 9u a6 7eo 5v 67 5w 8z.2 94 9s.2 9g 62 9h a8 71 65 9v 50 4t kd kb jr k8 4n 6v 54 4w 4o k9 a9 9k.2 7fk 4g",
    "cp855", ["cp855", "855", "csibm855", "ibm855"],
        "0.3k uq si ur sj up sh us sk ut sl uu sm uv sn uw so ux sp uy sq uz sr v0 ss v2 su v3 sv um tq ui tm ts sw tt sx ue ti tw t0 tx t1 uc tg tv sz 4r 57 7f5.3 7b6 7c4 ud th u0 t4 7dv 7dd 7dj 7dp u1 t5 7bk 7bo 7ck 7cc 7bw 7b4 7cs u2 t6 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k u3 t7 u4 t8 u5 t9 u6 ta u7 7bs 7bg 7ew 7es tb un 7eo tr u8 tc u9 td ua te ub tf ty t2 tu sy uk to 6ja 4t uj tn tz t3 ug tk ul tp uh tl uf tj 4n 7fk 4g",
    "cp856", ["cp856"],
        "0.3k 15c.r 1eke 4j 1eke 5z 1eke.a 4u 4s 59 58 1eke 4r 57 7f5.3 7b6 7c4 1eke.3 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 1eke.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 1eke.9 7bs 7bg 7ew 7es 4m 1eke 7eo 1eke.6 51 1eke.7 4v 50 4t 4x 6c7 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g",
    "cp857", ["cp857", "857", "csibm857", "ibm857"],
        "0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 8h 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 8g 5y 64 6w 4j 60 9q.2 69 6l 6r 6y 6p 5t 7y.2 5b 4u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 5d.2 5c 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 6b 5f 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 56 4q 5m.2 5k 1eke 5p.3 7bs 7bg 7ew 7es 4m 5o 7eo 5v 67 5w 5u 6t 5x 51 1eke 5z 62.2 61 6k 73 4v 50 4t 4x 1eke 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g",
    "cp858", ["cp858", "858", "cp00858", "csibm00858", "csibm858", "ibm00858", "ibm858", "pc_multilingual_850_euro"],
        "0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 6w 4j 60 5z b6 69 6l 6r 6y 6p 5t 4q 56 5b 4u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 5d.2 5c 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 6b 5f 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 6o 5s 5m.2 5k 6gc 5p.3 7bs 7bg 7ew 7es 4m 5o 7eo 5v 67 5w 5u 6t 5x 51 72 66 62.2 61 71 65 4v 50 4t 4x 6c7 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g",
    "cp860", ["cp860", "860", "csibm860", "ibm860"],
        "0.3k 5j 70 6h 6a.2 68 5d 6f 6i 5m 6g 5p 5w 6k 5f 5e 5l 5c 5k 6s.2 6q 62 6x 5o 5x 64 4i.2 61 6g7 5v 69 6l 6r 6y 6p 5t 4q 56 5b 5u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp861", ["cp861", "861", "cp_is", "csibm861", "ibm861"],
        "0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 5s 6o 66 5g.2 5l 6e 5i 6s 6u 72 6z 65 71 5y 64 6w 4j 60 6g7 b6 69 6l 6r 6y 5d 5p 5v 62 5b 6xc 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp862", ["cp862", "862", "cspc862latinhebrew", "ibm862"],
        "0.3k 15c.r 4i.2 4l 6g7 b6 69 6l 6r 6y 6p 5t 4q 56 5b 6xc 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp863", ["cp863", "863", "csibm863", "ibm863"],
        "0.3k 5j 70 6h 6a 5e 68 52 6f 6i.2 6g 6n 6m 6c7 5c 4n 5l 5k 5m 6s 5n 5r 6z 6x 4k 5w 64 4i.2 61 63 b6 4m 50 6r 6y 4o 54 4z 4v 5q 6xc 4s 59 58 5a 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp864", ["cp864", "864", "csibm864", "ibm864"],
        "0.11 19m 12.2i 4w 53 6qh.2 7f6 7b4 7b6 7cs 7c4 7cc 7bw 7ck 7bk 7bg 7bo 7bs qa 6qm qu 4x 59 58 6rs 4r 57 1ed3.2 1eke.2 1ed7.2 1eke 4g 4t 1e9u 4j.2 1e9w 1eke.2 1ea6.2 1ead 1eah 170 1eal 1eap 1eat 19c.a 1ec1 17f 1eb5 1eb9 1ebd 17j 4i 1e9s.2 1e9v 1e9x 1ebu 1ea3 1ea5 1ea9 1eab 1eaf 1eaj 1ean 1ear 1eav 1eax 1eaz 1eb1 1eb3 1eb7 1ebb 1ebf 1ebj 1ebl 1ebp 1ebv 1ebz 4m 4s 6v 5z 1ebt 18g 1ec3 1ec7 1ecb 1ecf 1ecj 1ecn 1ecr 1ect 1ecv 1ecz 1ebh 1ebw 1eby 1ebx 1ech 1e9p 18x 1ecl 1ecp 1ecs 1ecw 1ecy 1ec0 1ec5 1ed1.2 1ecd 1ec9 1ecx 7fk 1eke",
    "cp865", ["cp865", "865", "csibm865", "ibm865"],
        "0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 6w 4j 60 6g7 b6 69 6l 6r 6y 6p 5t 4q 56 5b 6xc 4s 59 58 4h 4r 4k 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g",
    "cp866", ["cp866", "866", "csibm866", "ibm866"],
        "0.3k sw.1c 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo u8.g sh up sk us sn uv su v2 4w 6qh 53 6qi 6ja 4k 7fk 4g",
    "cp869", ["cp869", "869", "cp_gr", "csibm869", "ibm869"],
        "0.3k 1eke.6 p2 1eke 53 4s 4m 6c8.2 p4 6c5 p5.2 q2 p8 1eke.2 pa q3 4p pb 4y.2 q4 4j q5.3 qy pc r0.2 pd.7 59 pk.2 4r 57 7f5.3 7b6 7c4 pm.4 7dv 7dd 7dj 7dp pq.2 7bk 7bo 7ck 7cc 7bw 7b4 7cs ps.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 pv.7 q9.3 7bs 7bg 7ew 7es qc.2 7eo qe.c qr qq qs p0 4t 4x qt.3 4n qw p1 4w 4o qx qz q8 r2 7fk 4g",
    "cp874", ["cp874", "874", "ms874", "windows_874"],
        "0.3k 6gc 1eke.4 6cm 1eke.b 6c8.2 6cc.2 6ci 6c3.2 1eke.8 4g 2rl.1m 1eke.4 2tb.t 1eke.4",
    "cp875", ["cp875"],
        "0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w pd.9 2j 1a 1o 14 17 x 12 pm.8 pv 2l 10 16 15 1n 2m 19 1b pw.8 3g 18 11 2n 1q.2 4o p2 p4.2 4g p6 p8 pa.2 2o 1m z 1s 13 1p y p1 2p.9 q9.6 4w 2y.9 qf.6 50 3i 37.8 ql.5 qr 4j q4.3 qy q7 r0.2 qz r2 qq qs.5 3f 1t.9 4t qx pc q8 6c8 6c5 3h 22.9 4x 59 q p3 6c9 4m 2k q 2b.8 4y 4n q q 4r.2 1c.a 4z 4p q q 57 4f",
    "cp1006", ["cp1006"],
        "0.4h 1dc.a 170 17f 4t 17j 1e9t 1ea5.2 1ea6.2 1ea9 1dna 1dnc 1eab 1ead 1eaf 1dnq 1dns 1eah 1eaj 1eal 1ean 1doa 1doc 1eap 1ear 1eat 1eav 1eax 1dok 1eaz 1eb1 1dos 1eb3 1doq 1eb5 1eb7 1eb9 1ebb 1ebd 1ebf 1ebh 1ebj 1ebl 1ebp 1ebt.9 1ec3 1ec5 1ec7 1ec9 1ecb 1doy 1dp0 1ecd 1ecf.3 1ecj 1dpa 1ecl 1ecn 1e9x 1ect 1dpi 1dpk.3 1e9s 1ea1.3 1ecx.3 1dps 1dpq 1e9o.2",
    "cp1026", ["cp1026", "1026", "csibm1026", "ibm1026"],
        "0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 3f 6p 5j 1a 1o 14 17 x 12 6h.3 6g 6l.3 6k 67 7y 8g 16 15 1n 2m 19 1b 5e 5g 5c.2 5f 5h 2j 5t 9r 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 8h 1m 5y 9q 13 1p 64 60 2p.9 4r 57 3h 2o 4m 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 6u 37.8 4h 5b 2l 10 1s 4u 4i.2 4l 53 4p 4n 52 58.3 4s 3g 4v 4o 50 5z 6f 1t.9 4t 6s 3i 6q.2 6t 7z 22.9 55 6z 2k 6x.2 73 70 6v 2b.8 4y 5w z 5u.2 5x 1c.a 4z 63 y 61.2 4f",
    "cp1125", ["cp1125", "1125", "cp866u", "ibm1125", "ruscii"],
        "0.3k sw.1c 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo u8.g sh up wg.2 sk us sm uu sn uv 53 6qi 6ja 4k 7fk 4g",
    "cp1140", ["cp1140", "1140", "cp01140", "csibm01140", "ebcdic_us_37_euro", "ibm01140", "ibm1140"],
        "0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 6f 6p 4i 1a 1o 14 17 3g 12 6h.3 6g 6l.3 6k 67 x 10 16 15 1n 4s 19 1b 5e 5g 5c.2 5f 5h 5j 5t 4m 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 1s 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 6gc 51 3i 37.8 4h 5b 5s 65.2 4u 2m 4j 4l 53 4p 4n 52 58.3 2j 2l 4v 4o 50 5z 3f 1t.9 4t 6s 6u 6q.2 6t 3h 22.9 55 6z.2 6x.2 73 2k 6v 2b.8 4y 5w 5y 5u.2 5x 1c.a 4z 63.2 61.2 4f",
    "cp1250", ["cp1250", "1250", "windows_1250"],
        "0.3k 6gc 1eke 6ca 1eke 6ce 6cm 6cg.2 1eke 6cw 9s 6d5 9m 9w al ah 1eke 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 9t 6d6 9n 9x am ai 4g jr k8 8x 4k 78 4m.4 9q 4r.4 aj 4w.2 kb 8y 50.5 79 9r 57 8t kd 8u ak 9g 5d.2 76 5g 8p 7a 5j 7g 5l 7s 5n 7u 5p.2 7i 7k 8z 93 5v.2 9c 5y.2 9k a6 62 a8 64.2 9u 67 9h 69.2 77 6c 8q 7b 6f 7h 6h 7t 6j 7v 6l.2 7j 7l 90 94 6r.2 9d 6u.2 9l a7 6y a9 70.2 9v k9",
    "cp1251", ["cp1251", "1251", "windows_1251"],
        "0.3k si.2 6ca ur 6ce 6cm 6cg.2 6gc 6cw sp 6d5 sq ss sr sv uq 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm ux 6d6 uy v0 uz v3 4g su v2 so 4k wg 4m.2 sh 4p sk 4r.4 sn 4w.2 sm uu wh 51.3 up 6ja us 57 uw sl ut uv sw.1s",
    "cp1252", ["cp1252", "1252", "windows_1252"],
        "0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 9s 6d5 9e 1eke al 1eke.2 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 9t 6d6 9f 1eke am ag 4g.2o",
    "cp1253", ["cp1253", "1253", "windows_1253"],
        "0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 1eke 6cw 1eke 6d5 1eke.5 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 1eke 6d6 1eke.4 4g p1.2 4j.7 1eke 4r.4 6c5 4w.4 p0 51.3 p4.3 57 p8 59 pa.k 1eke pv.18 1eke",
    "cp1254", ["cp1254", "1254", "windows_1254"],
        "0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 9s 6d5 9e 1eke.4 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 9t 6d6 9f 1eke.2 ag 4g.1c 7y 5t.c 8g 9q 67.h 7z 6p.c 8h 9r 73",
    "cp1255", ["cp1255", "1255", "windows_1255"],
        "0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 1eke 6d5 1eke.5 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 1eke 6d6 1eke.4 4g.4 6ga 4l.5 5z 4r.f 6v 57.5 14g.a 1eke 14r.9 168.5 1eke.7 15c.r 1eke.2 6by.2 1eke",
    "cp1256", ["cp1256", "1256", "windows_1256"],
        "0.3k 6gc 1a6 6ca b6 6ce 6cm 6cg.2 jq 6cw 1a1 6d5 9e 1ae 1aw 1ag 1bj 6c8.2 6cc.2 6ci 6c3.2 1bd 6jm 1ap 6d6 9f 6bw.2 1bu 4g 170 4i.8 1by 4r.f 17f 57.4 17j 1c1 17l.m 5z 187.4 18g.4 68 18k 6a 18l.4 6f.5 18p.2 6m.2 18r.4 6s 18v.2 6v 18x 6x 18y 6z.2 6by.2 1ci",
    "cp1257", ["cp1257", "1257", "windows_1257"],
        "0.3k 6gc 1eke 6ca 1eke 6ce 6cm 6cg.2 1eke 6cw 1eke 6d5 1eke 4o jr 54 1eke 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 1eke 6d6 1eke 4v kb 1eke 4g 1eke 4i.3 1eke 4m.2 60 4p 9i 4r.4 5i 4w.8 6w 55 9j 57.4 6e 78 8e 74 7a 5g.2 7s 7m 7g 5l ah 7q 82 8m 8a 8r 9s 8z 91 5v 98 5x.3 aa 8x 9m a2 64 aj al 67 79 8f 75 7b 6c.2 7t 7n 7h 6h ai 7r 83 8n 8b 8s 9t 90 92 6r 99 6t.3 ab 8y 9n a3 70 ak am k9",
    "cp1258", ["cp1258", "1258", "windows_1258"],
        "0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 1eke 6d5 9e 1eke.4 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 1eke 6d6 9f 1eke.2 ag 4g.z 76 5g.8 lc 5p.3 7k 5t ll 5v.2 bk 5y.7 bz lf 67.4 77 6c.8 ld 6l.3 7l 6p mb 6r.2 bl 6u.7 c0 6gb 73",
    "hp_roman8", ["hp_roman8", "cp1051", "csHPRoman8", "ibm1051", "r8", "roman8"],
        "0.4h 5c 5e 5k 5m.2 5q.2 50 jv jq 4o kc 61 63 6g4 4v 65 71 4w 5j 6f 5t 6p 4h 5b 4k 4j 4l 4n b6 4i 6a 6i 6s 6z 69 6h 6r 6y 68 6g 6q 6x 6c 6j 6u 70 5h 6m 60 5i 6d 6l 6w 6e 5g 6k 5y 64 5l 6n 67 5w 5d 5f 6b 5s 6o 5p 5o 5v 5u 5x 6t 9s.2 62 ag 73 66 72 53 51.2 5a 6c4 58.2 4q 56 4r 7fk 57 4x 1eke",
    "iso8859_2", ["iso8859_2", "csisolatin2", "iso_8859_2", "iso_8859_2_1987", "iso_ir_101", "l2", "latin2"],
        "0.4h 78 k8 8x 4k 8t 9m 4n.2 9s 9q 9w ah 4t al aj 4w 79 kb 8y 50 8u 9n jr 54 9t 9r 9x ai kd am ak 9g 5d.2 76 5g 8p 7a 5j 7g 5l 7s 5n 7u 5p.2 7i 7k 8z 93 5v.2 9c 5y.2 9k a6 62 a8 64.2 9u 67 9h 69.2 77 6c 8q 7b 6f 7h 6h 7t 6j 7v 6l.2 7j 7l 90 94 6r.2 9d 6u.2 9l a7 6y a9 70.2 9v k9",
    "iso8859_3", ["iso8859_3", "csisolatin3", "iso_8859_3", "iso_8859_3_1988", "iso_ir_109", "l3", "latin3"],
        "0.4h 86 k8 4j.2 1eke 84 4n.2 8g 9q 7y 8k 4t 1eke aj 4w 87 4y.4 85 53.2 8h 9r 7z 8l 59 1eke ak 5c.3 1eke 5g 7e 7c 5j.9 1eke 5t.4 80 5y.2 7w 61.4 a4 9o 67.4 1eke 6c 7f 7d 6f.9 1eke 6p.4 81 6u.2 7x 6x.4 a5 9p k9",
    "iso8859_4", ["iso8859_4", "csisolatin4", "iso_8859_4", "iso_8859_4_1988", "iso_ir_110", "l4", "latin4"],
        "0.4h 78 8o 9i 4k 88 8r 4n.2 9s 7m 82 9y 4t al 4v.2 79 kb 9j 50 89 8s jr 54 9t 7n 83 9z 96 am 97 74 5d.6 8e 7g 5l 7s 5n 7q 5p.2 8a 7k 91 98 8m 5w.5 aa 62.3 a0 a2 67 75 69.6 8f 7h 6h 7t 6j 7r 6l.2 8b 7l 92 99 8n 6s.5 ab 6y.3 a1 a3 k9",
    "iso8859_5", ["iso8859_5", "csisolatincyrillic", "cyrillic", "iso_8859_5", "iso_8859_5_1988", "iso_ir_144"],
        "0.4h sh.c 4t su.1u 6ja up.c 4n v2.2",
    "iso8859_6", ["iso8859_6", "arabic", "asmo_708", "csisolatinarabic", "ecma_114", "iso_8859_6", "iso_8859_6_1987", "iso_ir_127"],
        "0.4h 1eke.3 4k 1eke.7 170 4t 1eke.d 17f 1eke.3 17j 1eke 17l.q 1eke.5 18g.j 1eke.d",
    "iso8859_7", ["iso8859_7", "csisolatingreek", "ecma_118", "elot_928", "greek", "greek8", "iso_8859_7", "iso_8859_7_1987", "iso_ir_126"],
        "0.4h 6c8.2 4j 6gc 6gf 4m.4 oq 4r.3 1eke 6c5 4w.4 p0.3 53 p4.3 57 p8 59 pa.k 1eke pv.18 1eke",
    "iso8859_8", ["iso8859_8", "csisolatinhebrew", "hebrew", "iso_8859_8", "iso_8859_8_1988", "iso_8859_8_e", "iso_8859_8_i", "iso_ir_138"],
        "0.4h 1eke 4i.8 5z 4r.f 6v 57.4 1eke.w 6c7 15c.r 1eke.2 6by.2 1eke",
    "iso8859_9", ["iso8859_9", "csisolatin5", "iso_8859_9", "iso_8859_9_1989", "iso_ir_148", "l5", "latin5"],
        "0.5s 7y 5t.c 8g 9q 67.h 7z 6p.c 8h 9r 73",
    "iso8859_10", ["iso8859_10", "csisolatin6", "iso_8859_10", "iso_8859_10_1992", "iso_ir_157", "l6", "latin6"],
        "0.4h 78 7m 82 8a 88 8m 4n 8r 7k 9s 9y al 4t a2 96 4w 79 7n 83 8b 89 8n 53 8s 7l 9t 9z am 6c5 a3 97 74 5d.6 8e 7g 5l 7s 5n 7q 5p.4 91 98 5v.4 a0 60 aa 62.6 75 69.6 8f 7h 6h 7t 6j 7r 6l.4 92 99 6r.4 a1 6w ab 6y.5 8o",
    "iso8859_11", ["iso8859_11", "iso_8859_11", "iso_8859_11_2001", "thai"],
        "0.4h 2rl.1m 1eke.4 2tb.t 1eke.4",
    "iso8859_13", ["iso8859_13", "iso_8859_13", "l7", "latin7"],
        "0.4h 6cd 4i.3 6ce 4m.2 60 4p 9i 4r.4 5i 4w.4 6cc 51.3 6w 55 9j 57.4 6e 78 8e 74 7a 5g.2 7s 7m 7g 5l ah 7q 82 8m 8a 8r 9s 8z 91 5v 98 5x.3 aa 8x 9m a2 64 aj al 67 79 8f 75 7b 6c.2 7t 7n 7h 6h ai 7r 83 8n 8b 8s 9t 90 92 6r 99 6t.3 ab 8y 9n a3 70 ak am 6c9",
    "iso8859_14", ["iso8859_14", "iso_8859_14", "iso_8859_14_1998", "iso_celtic", "iso_ir_199", "l8", "latin8"],
        "0.4h 5xe.2 4j 7e.2 5xm 4n 60w 4p 60y 5xn 642 4t.2 ag 5y6.2 80.2 5z4.2 52 5zq 60x 5zr 60z 600 643 610.2 601 5c.g ac 5t.6 60a 60.6 ae 67.h ad 6p.6 60b 6w.6 af 73",
    "iso8859_15", ["iso8859_15", "iso_8859_15", "l9", "latin9"],
        "0.4k 6gc 4l 9s 4n 9t 4p.b al 51.3 am 55.3 9e.2 ag 5b.1t",
    "iso8859_16", ["iso8859_16", "iso_8859_16", "iso_8859_16_2001", "iso_ir_226", "l10", "latin10"],
        "0.4h 78.2 8x 6gc 6ce 9s 4n 9t 4p ew 4r ah 4t ai.2 4w.2 7g 8y al 6cd 52.2 am 7h ex 57 9e.2 ag ak 5c.3 76 5g 7a 5i.a 7k 8z 5u.3 9c 5y 9m a8 61.4 7s ey 67.4 77 6c 7b 6e.a 7l 90 6q.3 9d 6u 9n a9 6x.4 7t ez 73",
    "koi8_r", ["koi8_r", "cskoi8r"],
        "0.3k 7b4 7b6 7bg 7bk 7bo 7bs 7bw 7c4 7cc 7ck 7cs 7eo 7es 7ew 7f0 7f4.4 6xs 7fk 6qh.2 6rs 6sk.2 4g 6xt 4w 4y 53 6v 7dc.3 up 7df.f sh 7du.b 4p um ts.2 ue tw.2 uc tv ud u0.8 un u8.4 ty tu uk uj tz ug ul uh uf ui tq sw.2 ti t0.2 tg sz th t4.8 tr tc.4 t2 sy to tn t3 tk tp tl tj tm",
    "koi8_t", ["koi8_t"],
        "0.3k wr wj 6ca wi 6ce 6cm 6cg.2 1eke 6cw xf 6d5 xe xj xi 1eke wq 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 1eke 6d6 1eke.5 z3 z2 up 4k yr 4m.2 1eke.3 4r.4 1eke 4w.3 sh 1eke yq 52.2 1eke 6ja 1eke 57 1eke.3 4p um ts.2 ue tw.2 uc tv ud u0.8 un u8.4 ty tu uk uj tz ug ul uh uf ui tq sw.2 ti t0.2 tg sz th t4.8 tr tc.4 t2 sy to tn t3 tk tp tl tj tm",
    "koi8_u", ["koi8_u"],
        "0.3k 7b4 7b6 7bg 7bk 7bo 7bs 7bw 7c4 7cc 7ck 7cs 7eo 7es 7ew 7f0 7f4.4 6xs 7fk 6qh.2 6rs 6sk.2 4g 6xt 4w 4y 53 6v 7dc.3 up us 7dg uu.2 7dj.5 wh 7dp.5 sh sk 7dv sm.2 7dy.5 wg 7e4 4p um ts.2 ue tw.2 uc tv ud u0.8 un u8.4 ty tu uk uj tz ug ul uh uf ui tq sw.2 ti t0.2 tg sz th t4.8 tr tc.4 t2 sy to tn t3 tk tp tl tj tm",
    "kz1048", ["kz1048", "kz_1048", "rk1048", "strk1048_2002"],
        "0.3k si.2 6ca ur 6ce 6cm 6cg.2 6gc 6cw sp 6d5 sq wq xm sv uq 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm ux 6d6 uy wr xn v3 4g xc.2 yg 4k yw 4m.2 sh 4p wi 4r.4 xa 4w.2 sm uu yx 51.3 up 6ja wj 57 yh wy.2 xb sw.1s",
    "mac_arabic", ["mac_arabic"],
        "0.3k 5g 4g 5j 5l 5t 5y 64 69 68 6a 6c 1bu 4r 6f 6h 6g 6i.2 6l 6cm 6m.2 6p 6r 57 6s 6u.2 6y 6x 6z.2 w.5 19m 12.6 170 19.3 19c.a 1m 17f 1o.3 17j 7re 17l.q 2j.5 18g.j 1a6 1a1 1ae 1cl 1b8 1bj 1ag 1ap 3f.3 1aw 1ci",
    "mac_croatian", ["mac_croatian"],
        "0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 9s 6jm 50 4o 6sg al 60 6qm 4x 6sk.2 6py 51 6pu 6q9 6q7 9t 6qz 4q 56 q1 am 6w 5b 4h 4s 6qi b6 6rs 7a 4r 7g 6cm 4g 5c 5f 5x 9e.2 7k 6c4 6cc.2 6c8.2 6v 7gq 1d6n 4p 6dg 6gc 6d5.2 5i 57 6c3 53 6ca 6ce 6cw 5e 7b 5d 7h 5k 5p.3 5o 5v.2 7l 5u 62.2 61 8h jq kc 4v qo 5n ka 54 5m 6e jr",
    "mac_cyrillic", ["mac_cyrillic", "maccyrillic"],
        "0.3k sw.w 6cg 4w wg 4j 4n 6ci 52 sm 4u 4p 6jm si uq 6sg sj ur 6qm 4x 6sk.2 uu 51 wh so sk us sn uv sp ux sq uy uw sl 4s 6qi b6 6rs 6py 4r 57 6cm 4g sr uz ss v0 ut 6c3.2 6cc.2 6c8.2 6v 6ce su v2 sv v3 6ja sh up un ts.v 6gc",
    "mac_farsi", ["mac_farsi"],
        "0.3k 5g 4g 5j 5l 5t 5y 64 69 68 6a 6c 1bu 4r 6f 6h 6g 6i.2 6l 6cm 6m.2 6p 6r 57 6s 6u.2 6y 6x 6z.2 w.5 19m 12.6 170 19.3 1dc.a 1m 17f 1o.3 17j 7re 17l.q 2j.5 18g.j 1a6 1a1 1ae 1cl 1b8 1bj 1ag 1ap 3f.3 1aw 1ci",
    "mac_greek", ["mac_greek", "macgreek"],
        "0.3k 5g 55 4y 5l 4z 5y 64 p1 68 6a 6c p0 4o 6f 6h 6g 6i.2 4j 6jm 6m.2 6ci 59 6cw 6s 6u 4m 6gc 6x 6z.2 6cg pf.2 pk pn pq ps 67 4u 4p pv q2 4n 6sg 4w 53 pd 4x 6sk.2 4l pe ph.3 pl.2 po py q3 q0.2 q4 pp 4s pr pt 6rs pw 4r 57 6cm 4g px pz p2 p4 9f 6c3 6c5 6cc.2 6c8.2 6v p5.2 p8 pa q5.3 r0 pb r1 q9.2 qw qc.2 qu qb qf qh qm qi.4 qn.2 r2 qp qr.2 qg qx qq qv qt qe qy.2 pc q8 4t",
    "mac_iceland", ["mac_iceland", "maciceland"],
        "0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 65 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 5i 60 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 6e 6w 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 6dg 6gc 5s 6o 66 72 71 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 8h jq kc 4v k8.3 54 kd kb jr",
    "mac_latin2", ["mac_latin2", "mac_centeuro", "maccentraleurope", "maclatin2"],
        "0.3k 5g 74.2 5l 78 5y 64 69 79 7g 6c 7h 7a.2 6h ah.2 7i 6l 7j 7m.2 7q 6r 7r 6s 6u 6t 6y 7u.2 70 6cg 4w 7s 4j 4n 6ci 52 67 4u 4p 6jm 7t 4o 6sg 83 8e.2 8a 6sk.2 8b 8m 6pu 6q9 8y 8r.4 8p.2 91.2 8z 4s 6qi 90 93 6py 4r 57 6cm 4g 94 9c 5x 9d 98 6c3.2 6cc.2 6c8.2 6v 7gq 99 9g.2 9k 6d5.2 9l 9i.2 9s 6ca 6ce 9t 9m.2 5d 9w.2 5p al.2 a2 5v.2 a3 a6 62 a7.5 65 71 8n aj 8x ak 82 jr",
    "mac_roman", ["mac_roman", "macintosh", "macroman"],
        "0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 5i 60 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 6e 6w 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 6dg 6gc 6d5.2 1dkx.2 6ch 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 8h jq kc 4v k8.3 54 kd kb jr",
    "mac_romanian", ["mac_romanian"],
        "0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 76 ew 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 77 ex 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 6dg 6gc 6d5.2 ey.2 6ch 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 8h jq kc 4v k8.3 54 kd kb jr",
    "mac_turkish", ["mac_turkish", "macturkish"],
        "0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 5i 60 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 6e 6w 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 7y.2 8g.2 9q.2 6ch 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 1d40 jq kc 4v k8.3 54 kd kb jr",
    "palmos", ["palmos"],
        "0.3k 6gc 3l 6ca b6 6ce 6cm 6cg.2 jq 6cw 9s 6d5 9e 7l2 7kz 7l1 7kw 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 9t 6d6 9f 4d.2 ag 4g.2o",
    "ptcp154", ["ptcp154", "cp154", "csptcp154", "cyrillic_asian", "pt154"],
        "0.3k wm wi z2 wj 6ce 6cm xi xa xe xb ww yq wy wq xm xk wn 6c8.2 6cc.2 6ci 6c3.2 xf xj wx yr wz wr xn xl 4g su v2 so yw wo xc 4n sh 4p yg 4r.2 z3 4u ws 4w xd sm uu wp yx 52.2 up 6ja yh 57 uw x6.2 wt sw.1s",
    "tis_620", ["tis_620", "iso_ir_166", "tis620", "tis_620_0", "tis_620_2529_0", "tis_620_2529_1"],
        "0.4g 1eke 2rl.1m 1eke.4 2tb.t 1eke.4"]

// The 256 code points a charmap codec decodes bytes to (65534 where undefined),
// [] for any other codec; built on first use.
var pycdTabs = {}
function pycdTable(codec) {
    var t = pycdTabs[codec]
    if (t !== undefined) { return t }
    var out = []
    var maps = pycdCharmaps
    for (var q = 0; q < maps.length; q = q + 3) {
        if (maps[q] == codec) { pycdRuns(maps[q + 2], out) }
    }
    pycdTabs[codec] = out
    return out
}
function pycdRuns(t, out) {
    var v = 0
    var len = 0
    var inLen = false
    var i = 0
    while (i <= t.length) {
        var c = i < t.length ? t.charCodeAt(i) : 32
        if (c == 32) {
            if (!inLen) { len = 1 }
            for (var k = 0; k < len; k++) { out.push(v == 65534 ? v : v + k) }
            v = 0
            len = 0
            inLen = false
        } else if (c == 46) {
            inLen = true
        } else if (inLen) {
            len = len * 36 + (c <= 57 ? c - 48 : c - 87)
        } else {
            v = v * 36 + (c <= 57 ? c - 48 : c - 87)
        }
        i = i + 1
    }
}

// The byte a charmap codec encodes cp to, or -1: the inverse of its table,
// where of two bytes decoding to one code point the higher wins, as in
// CPython's encoding maps.
var pycdRevs = {}
function pycdCharByte(codec, cp) {
    var rev = pycdRevs[codec]
    if (rev === undefined) {
        rev = {}
        var t = pycdTable(codec)
        for (var b = 0; b < t.length; b++) {
            if (t[b] != 65534) { rev["c" + t[b]] = b }
        }
        pycdRevs[codec] = rev
    }
    var v = rev["c" + cp]
    if (v === undefined) { return 0 - 1 }
    return v
}

// The LookupError or UnicodeError a codec raises before coding anything: for
// an unknown name, for a codec that is not a text encoding (the message names
// the encoding as given, and codecs.encode or codecs.decode after verb), and
// for the undefined codec. Sets r.err; true when it did.
function pycdRefuse(r, enc, codec, verb) {
    if (codec == "") {
        r.err = "LookupError"
        r.msg = "unknown encoding: " + enc
    } else if (codec == "binary") {
        r.err = "LookupError"
        r.msg = "'" + enc + "' is not a text encoding; use codecs." + verb + "() to handle arbitrary codecs"
    } else if (codec == "undefined") {
        r.err = "UnicodeError"
        r.msg = "undefined encoding"
    }
    return r.err != ""
}

// A code point as repr spells it inside a string: \xhh, \uhhhh or \Uhhhhhhhh.
function pycdEsc(cp) {
    var w = 8
    var head = "\\U"
    if (cp < 256) {
        w = 2
        head = "\\x"
    } else if (cp < 65536) {
        w = 4
        head = "\\u"
    }
    return head + pycdHex(cp, w)
}
function pycdHex(v0, w) {
    var d = "0123456789abcdef"
    var hex = ""
    var v = v0
    for (var i = 0; i < w; i++) {
        hex = d.charAt(v % 16) + hex
        v = Math.floor(v / 16)
    }
    return hex
}

// One code unit (utf-16) or code point (utf-32) as bytes, in the byte order le.
function pycdPushUnit(out, u, size, le) {
    var bs = []
    var v = u
    for (var i = 0; i < size; i++) {
        bs.push(v % 256)
        v = Math.floor(v / 256)
    }
    for (var j = 0; j < size; j++) { out.push(le ? bs[j] : bs[size - 1 - j]) }
}

// Encode the code points cps under the encoding named enc. A UTF-16/32 codec
// meets each unencodable character on its own; the others hand a whole run of
// them to the handler. surrogateescape takes U+DC80..U+DCFF to the byte they
// escape and refuses anything else - from the first refused character on for
// utf-8, ascii and latin-1, whose encoders escape a prefix of the run
// themselves. utf-7 encodes every code point, lone surrogates included.
function pycdEncode(cps, enc, errs) {
    var r = {err: "", msg: "", b: [], s: ""}
    var codec = pycdLookup(enc)
    if (pycdRefuse(r, enc, codec, "encode")) { return r }
    var out = r.b
    if (codec == "utf-7") {
        pycdEncodeUtf7(cps, out)
        return r
    }
    if (codec == "unicode-escape" || codec == "raw-unicode-escape") {
        pycdEncodeEscape(cps, out, codec == "raw-unicode-escape")
        return r
    }
    if (codec == "punycode") {
        pycdEncodePuny(cps, out)
        return r
    }
    if (codec == "idna") {
        pycdEncodeIdna(r, cps, errs)
        return r
    }
    if (codec == "utf-8-sig") {
        out.push(239)
        out.push(187)
        out.push(191)
        codec = "utf-8"
    }
    var cm = pycdTable(codec).length > 0
    var size = 0
    var le = true
    if (codec == "utf-16" || codec == "utf-16-le" || codec == "utf-16-be") { size = 2 }
    if (codec == "utf-32" || codec == "utf-32-le" || codec == "utf-32-be") { size = 4 }
    if (codec == "utf-16-be" || codec == "utf-32-be") { le = false }
    if (codec == "utf-16" || codec == "utf-32") { pycdPushUnit(out, 65279, size, true) }
    var why = "surrogates not allowed"
    var errCodec = codec
    if (codec == "ascii") { why = "ordinal not in range(128)" }
    if (codec == "latin-1") { why = "ordinal not in range(256)" }
    if (cm) {
        why = "character maps to <undefined>"
        errCodec = "charmap"
    }
    var k = 0
    while (k < cps.length) {
        var cp = cps[k]
        if (!pycdEncBad(codec, cm, cp)) {
            pycdEncOne(out, codec, cm, cp, size, le)
            k = k + 1
            continue
        }
        var j = k + 1
        if (size == 0) {
            while (j < cps.length && pycdEncBad(codec, cm, cps[j])) { j = j + 1 }
        }
        var bad = errs == "strict" || (errs == "surrogateescape" && size > 0)
        if (errs == "surrogateescape" && size == 0) {
            var q = k
            while (q < j && cps[q] >= 56448 && cps[q] <= 56575) { q = q + 1 }
            if (q < j) {
                if (!cm) { k = q }
                bad = true
            }
        }
        if (bad) {
            var where = "character '" + pycdEsc(cps[k]) + "' in position " + k
            if (j - k > 1) { where = "characters in position " + k + "-" + (j - 1) }
            r.err = "UnicodeEncodeError"
            r.msg = "'" + errCodec + "' codec can't encode " + where + ": " + why
            return r
        }
        for (var p = k; p < j; p++) {
            var rep = ""
            if (errs == "replace") { rep = "?" }
            else if (errs == "backslashreplace") { rep = pycdEsc(cps[p]) }
            else if (errs == "xmlcharrefreplace") { rep = "&#" + cps[p] + ";" }
            else if (errs == "surrogateescape") { out.push(cps[p] - 56320) }
            else if (errs != "ignore") {
                r.err = "LookupError"
                r.msg = "unknown error handler name '" + errs + "'"
                return r
            }
            for (var t = 0; t < rep.length; t++) { pycdEncOne(out, codec, cm, rep.charCodeAt(t), size, le) }
        }
        k = j
    }
    return r
}

// A code point the codec (cm: a charmap codec) cannot encode.
function pycdEncBad(codec, cm, cp) {
    if (cm) { return pycdCharByte(codec, cp) < 0 }
    if (codec == "ascii") { return cp >= 128 }
    if (codec == "latin-1") { return cp >= 256 }
    return cp >= 55296 && cp <= 57343
}

function pycdEncOne(out, codec, cm, cp, size, le) {
    if (size == 4) {
        pycdPushUnit(out, cp, 4, le)
    } else if (size == 2) {
        if (cp >= 65536) {
            pycdPushUnit(out, 55296 + Math.floor((cp - 65536) / 1024), 2, le)
            pycdPushUnit(out, 56320 + (cp - 65536) % 1024, 2, le)
        } else {
            pycdPushUnit(out, cp, 2, le)
        }
    } else if (cm) {
        out.push(pycdCharByte(codec, cp))
    } else if (codec != "utf-8" || cp < 128) {
        out.push(cp)
    } else if (cp < 2048) {
        out.push(192 + Math.floor(cp / 64))
        out.push(128 + cp % 64)
    } else if (cp < 65536) {
        out.push(224 + Math.floor(cp / 4096))
        out.push(128 + Math.floor(cp / 64) % 64)
        out.push(128 + cp % 64)
    } else {
        out.push(240 + Math.floor(cp / 262144))
        out.push(128 + Math.floor(cp / 4096) % 64)
        out.push(128 + Math.floor(cp / 64) % 64)
        out.push(128 + cp % 64)
    }
}

// UTF-7 (RFC 2152) as CPython writes it: tab, newline, carriage return and
// the printable ASCII other than "+", backslash and "~" stand for themselves,
// "+" is "+-", and every other run goes as "+", the base64 of its UTF-16 code
// units, and a "-" that is left out before a character that is neither a
// base64 digit nor "-".
function pycdEncodeUtf7(cps, out) {
    var shift = false
    var bits = 0
    var buf = 0
    for (var k = 0; k < cps.length; k++) {
        var c = cps[k]
        var direct = c == 9 || c == 10 || c == 13 || (c >= 32 && c <= 125 && c != 43 && c != 92)
        if (shift && direct) {
            if (bits > 0) { out.push(pycdB64Char(buf * pycdPow2(6 - bits))) }
            bits = 0
            buf = 0
            shift = false
            if (pycdB64Val(c) >= 0 || c == 45) { out.push(45) }
            out.push(c)
        } else if (!shift && c == 43) {
            out.push(43)
            out.push(45)
        } else if (!shift && direct) {
            out.push(c)
        } else {
            if (!shift) { out.push(43) }
            shift = true
            var units = [c]
            if (c >= 65536) { units = [55296 + Math.floor((c - 65536) / 1024), 56320 + (c - 65536) % 1024] }
            for (var u = 0; u < units.length; u++) {
                buf = buf * 65536 + units[u]
                bits = bits + 16
                while (bits >= 6) {
                    var p = pycdPow2(bits - 6)
                    out.push(pycdB64Char(Math.floor(buf / p)))
                    buf = buf % p
                    bits = bits - 6
                }
            }
        }
    }
    if (bits > 0) { out.push(pycdB64Char(buf * pycdPow2(6 - bits))) }
    if (shift) { out.push(45) }
}

function pycdB64Char(v) {
    return "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".charCodeAt(v)
}

// The value of a base64 digit, or -1.
function pycdB64Val(c) {
    if (c >= 65 && c <= 90) { return c - 65 }
    if (c >= 97 && c <= 122) { return c - 71 }
    if (c >= 48 && c <= 57) { return c + 4 }
    if (c == 43) { return 62 }
    if (c == 47) { return 63 }
    return 0 - 1
}

function pycdPow2(k) {
    var v = 1
    for (var i = 0; i < k; i++) { v = v * 2 }
    return v
}

function pycdChr(cp) {
    if (cp < 65536) { return String.fromCharCode(cp) }
    var v = cp - 65536
    return String.fromCharCode(55296 + Math.floor(v / 1024)) + String.fromCharCode(56320 + v % 1024)
}

// Decode the byte values bs0 under the encoding named enc. Each undecodable
// range [from, to) - CPython's UnicodeDecodeError start and end - goes to
// pycdBad. No bytes decode to "" under any name, as in CPython, which looks
// no codec up for them.
function pycdDecode(bs0, enc, errs) {
    var r = {err: "", msg: "", b: [], s: ""}
    if (bs0.length == 0) { return r }
    var codec = pycdLookup(enc)
    if (pycdRefuse(r, enc, codec, "decode")) { return r }
    if (codec == "unicode-escape" || codec == "raw-unicode-escape") {
        pycdDecodeEscape(r, bs0, codec == "raw-unicode-escape", errs)
        return r
    }
    if (codec == "punycode") {
        pycdDecodePuny(r, bs0, errs)
        return r
    }
    if (codec == "idna") {
        pycdDecodeIdna(r, bs0, errs)
        return r
    }
    var bs = bs0
    if (codec == "utf-8-sig") {
        // A leading BOM goes, and positions count from after it.
        if (bs0.length >= 3 && bs0[0] == 239 && bs0[1] == 187 && bs0[2] == 191) {
            bs = []
            for (var x = 3; x < bs0.length; x++) { bs.push(bs0[x]) }
        }
        codec = "utf-8"
    }
    var n = bs.length
    var i = 0
    if (codec == "latin-1") {
        while (i < n) {
            r.s = r.s + String.fromCharCode(bs[i])
            i = i + 1
        }
        return r
    }
    if (codec == "utf-7") {
        pycdDecodeUtf7(r, bs, errs)
        return r
    }
    var tab = pycdTable(codec)
    if (codec == "ascii" || tab.length > 0) {
        while (i < n) {
            var c0 = bs[i]
            if (tab.length > 0) { c0 = tab[c0] }
            if (codec == "ascii" && c0 >= 128) {
                pycdBad(r, "ascii", bs, i, i + 1, "ordinal not in range(128)", errs)
            } else if (c0 == 65534) {
                pycdBad(r, "charmap", bs, i, i + 1, "character maps to <undefined>", errs)
            } else {
                r.s = r.s + String.fromCharCode(c0)
            }
            if (r.err != "") { return r }
            i = i + 1
        }
        return r
    }
    if (codec == "utf-8") {
        pycdDecodeUtf8(r, bs, errs)
        return r
    }
    var size = 2
    if (codec == "utf-32" || codec == "utf-32-le" || codec == "utf-32-be") { size = 4 }
    var le = codec != "utf-16-be" && codec != "utf-32-be"
    if ((codec == "utf-16" && n >= 2) || (codec == "utf-32" && n >= 4)) {
        var bom = pycdUnit(bs, 0, size, true)
        if (bom == 65279) { i = size }
        if ((size == 2 && bom == 65534) || (size == 4 && bom == 4294836224)) {
            i = size
            le = false
        }
    }
    var name = "utf-16-be"
    if (size == 2 && le) { name = "utf-16-le" }
    if (size == 4) { name = le ? "utf-32-le" : "utf-32-be" }
    while (i < n) {
        if (n - i < size) {
            i = pycdBad(r, name, bs, i, n, "truncated data", errs)
        } else if (size == 4) {
            var u = pycdUnit(bs, i, 4, le)
            if (u >= 55296 && u <= 57343) {
                i = pycdBad(r, name, bs, i, i + 4, "code point in surrogate code point range(0xd800, 0xe000)", errs)
            } else if (u >= 1114112) {
                i = pycdBad(r, name, bs, i, i + 4, "code point not in range(0x110000)", errs)
            } else {
                r.s = r.s + pycdChr(u)
                i = i + 4
            }
        } else {
            var w = pycdUnit(bs, i, 2, le)
            if (w < 55296 || w > 57343) {
                r.s = r.s + String.fromCharCode(w)
                i = i + 2
            } else if (w >= 56320) {
                i = pycdBad(r, name, bs, i, i + 2, "illegal encoding", errs)
            } else if (n - i < 4) {
                i = pycdBad(r, name, bs, i, n, "unexpected end of data", errs)
            } else {
                var w2 = pycdUnit(bs, i + 2, 2, le)
                if (w2 >= 56320 && w2 <= 57343) {
                    r.s = r.s + String.fromCharCode(w) + String.fromCharCode(w2)
                    i = i + 4
                } else {
                    i = pycdBad(r, name, bs, i, i + 2, "illegal UTF-16 surrogate", errs)
                }
            }
        }
        if (r.err != "") { return r }
    }
    return r
}

// The size-byte unit at bs[i], in the byte order le.
function pycdUnit(bs, i, size, le) {
    var v = 0
    for (var j = 0; j < size; j++) {
        var b = le ? bs[i + size - 1 - j] : bs[i + j]
        v = v * 256 + b
    }
    return v
}

function pycdDecodeUtf8(r, bs, errs) {
    var n = bs.length
    var i = 0
    while (i < n) {
        var b0 = bs[i]
        if (b0 < 128) {
            r.s = r.s + String.fromCharCode(b0)
            i = i + 1
            continue
        }
        var need = 0
        var cp = 0
        if (b0 >= 194 && b0 <= 223) {
            need = 1
            cp = b0 % 32
        } else if (b0 >= 224 && b0 <= 239) {
            need = 2
            cp = b0 % 16
        } else if (b0 >= 240 && b0 <= 244) {
            need = 3
            cp = b0 % 8
        }
        var why = ""
        if (need == 0) { why = "invalid start byte" }
        var to = i + 1
        var k = 1
        while (why == "" && k <= need) {
            if (i + k >= n) {
                why = "unexpected end of data"
                to = n
            } else {
                var c = bs[i + k]
                var lo = 128
                var hi = 191
                if (k == 1 && b0 == 224) { lo = 160 }
                if (k == 1 && b0 == 237) { hi = 159 }
                if (k == 1 && b0 == 240) { lo = 144 }
                if (k == 1 && b0 == 244) { hi = 143 }
                if (c < lo || c > hi) {
                    why = "invalid continuation byte"
                    to = i + k
                } else {
                    cp = cp * 64 + c % 64
                    k = k + 1
                }
            }
        }
        if (why != "") {
            i = pycdBad(r, "utf-8", bs, i, to, why, errs)
            if (r.err != "") { return }
            continue
        }
        r.s = r.s + pycdChr(cp)
        i = i + need + 1
    }
}

// UTF-7 as CPython's decoder reads it (its errors name the codec "utf7"):
// "+-" is "+", "+" and base64 digits are UTF-16 code units, ended by any
// other byte, of which "-" is absorbed. A high surrogate waits for the next
// unit; one not followed by a low surrogate is written alone, except that a
// shift ending in a byte that is not ASCII drops it. An error inside a shift
// spans from its "+".
function pycdDecodeUtf7(r, bs, errs) {
    var n = bs.length
    var i = 0
    var shift = false
    var bits = 0
    var buf = 0
    var sur = 0
    var start = 0
    while (i < n) {
        var c = bs[i]
        if (shift) {
            var v = pycdB64Val(c)
            if (v >= 0) {
                buf = buf * 64 + v
                bits = bits + 6
                i = i + 1
                if (bits >= 16) {
                    var p = pycdPow2(bits - 16)
                    var u = Math.floor(buf / p)
                    buf = buf % p
                    bits = bits - 16
                    if (sur != 0 && u >= 56320 && u <= 57343) {
                        r.s = r.s + String.fromCharCode(sur) + String.fromCharCode(u)
                        sur = 0
                    } else {
                        if (sur != 0) { r.s = r.s + String.fromCharCode(sur) }
                        sur = 0
                        if (u >= 55296 && u <= 56319) { sur = u } else { r.s = r.s + String.fromCharCode(u) }
                    }
                }
                continue
            }
            shift = false
            var why = ""
            if (bits >= 6) { why = "partial character in shift sequence" }
            else if (bits > 0 && buf != 0) { why = "non-zero padding bits in shift sequence" }
            if (why != "") {
                i = pycdBad(r, "utf7", bs, start, i + 1, why, errs)
                if (r.err != "") { return }
                continue
            }
            if (sur != 0 && c < 128 && c != 43) { r.s = r.s + String.fromCharCode(sur) }
            sur = 0
            if (c == 45) { i = i + 1 }
        } else if (c == 43) {
            start = i
            i = i + 1
            if (i < n && bs[i] == 45) {
                i = i + 1
                r.s = r.s + "+"
            } else if (i < n && pycdB64Val(bs[i]) < 0) {
                i = pycdBad(r, "utf7", bs, start, i + 1, "ill-formed sequence", errs)
                if (r.err != "") { return }
            } else {
                shift = true
                sur = 0
                bits = 0
                buf = 0
            }
        } else if (c < 128) {
            r.s = r.s + String.fromCharCode(c)
            i = i + 1
        } else {
            i = pycdBad(r, "utf7", bs, i, i + 1, "unexpected special character", errs)
            if (r.err != "") { return }
        }
    }
    if (shift && (sur != 0 || bits >= 6 || (bits > 0 && buf != 0))) {
        pycdBad(r, "utf7", bs, start, n, "unterminated shift sequence", errs)
    }
}

// The error handler errs on the undecodable bytes [from, to): its text is
// appended to r.s, or r.err set; the answer is where decoding resumes.
// surrogateescape gives each of up to four leading bytes b >= 128 as U+DC00+b
// and resumes after them, refusing a range that starts with an ASCII byte. A
// handler name is only looked up once it is needed, as CPython does.
function pycdBad(r, codec, bs, from, to, why, errs) {
    var esc = 0
    if (errs == "surrogateescape") {
        while (esc < 4 && from + esc < to && bs[from + esc] >= 128) {
            r.s = r.s + String.fromCharCode(56320 + bs[from + esc])
            esc = esc + 1
        }
        if (esc > 0) { return from + esc }
    }
    if (errs == "strict" || errs == "surrogateescape") {
        var where = "byte 0x" + pycdHex(bs[from], 2) + " in position " + from
        if (to - from > 1) { where = "bytes in position " + from + "-" + (to - 1) }
        r.err = "UnicodeDecodeError"
        r.msg = "'" + codec + "' codec can't decode " + where + ": " + why
    } else if (errs == "replace") {
        r.s = r.s + String.fromCharCode(65533)
    } else if (errs == "backslashreplace") {
        for (var i = from; i < to; i++) { r.s = r.s + "\\x" + pycdHex(bs[i], 2) }
    } else if (errs == "xmlcharrefreplace" || errs == "namereplace") {
        r.err = "TypeError"
        r.msg = "don't know how to handle UnicodeDecodeError in error callback"
    } else if (errs != "ignore") {
        r.err = "LookupError"
        r.msg = "unknown error handler name '" + errs + "'"
    }
    return to
}

// unicode_escape writes a backslash, tab, newline and carriage return as
// their escapes, the other code points below 32 and from 127 to 255 as \xhh
// and everything above as \uhhhh or \Uhhhhhhhh; raw_unicode_escape writes
// every code point below 256 as its byte and only the rest escaped. Neither
// can fail, so the error handler is never asked.
function pycdEncodeEscape(cps, out, raw) {
    for (var k = 0; k < cps.length; k++) {
        var c = cps[k]
        var esc = ""
        if (c >= 65536) { esc = "\\U" + pycdHex(c, 8) }
        else if (c >= 256) { esc = "\\u" + pycdHex(c, 4) }
        else if (raw) { out.push(c) }
        else if (c == 92) { esc = "\\\\" }
        else if (c == 9) { esc = "\\t" }
        else if (c == 10) { esc = "\\n" }
        else if (c == 13) { esc = "\\r" }
        else if (c < 32 || c >= 127) { esc = "\\x" + pycdHex(c, 2) }
        else { out.push(c) }
        for (var i = 0; i < esc.length; i++) { out.push(esc.charCodeAt(i)) }
    }
}

// The value of a hexadecimal digit byte, or -1.
function pycdHexVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 97 && c <= 102) { return c - 87 }
    if (c >= 65 && c <= 70) { return c - 55 }
    return 0 - 1
}

// The two escape decoders. Every byte stands for the code point of its value
// except a backslash: raw_unicode_escape reads only \uhhhh and \Uhhhhhhhh,
// unicode_escape also Python's string-literal escapes - an unknown one stays
// as written, an octal one takes up to three digits, and backslash-newline is
// nothing. A malformed escape spans from its backslash to the first byte that
// is not a hex digit. \N{name} needs the Unicode name database, which is not
// here, and is refused.
function pycdDecodeEscape(r, bs, raw, errs) {
    var codec = raw ? "rawunicodeescape" : "unicodeescape"
    var n = bs.length
    var i = 0
    while (i < n) {
        var c = bs[i]
        if (c != 92 || (raw && i + 1 >= n)) {
            r.s = r.s + String.fromCharCode(c)
            i = i + 1
            continue
        }
        var start = i
        if (i + 1 >= n) {
            i = pycdBad(r, codec, bs, i, n, "\\ at end of string", errs)
            if (r.err != "") { return }
            continue
        }
        var e = bs[i + 1]
        i = i + 2
        var count = 0
        var why = ""
        if (e == 117) {
            count = 4
            why = "truncated \\uXXXX escape"
        } else if (e == 85) {
            count = 8
            why = "truncated \\UXXXXXXXX escape"
        } else if (e == 120 && !raw) {
            count = 2
            why = "truncated \\xXX escape"
        }
        if (count == 0) {
            i = pycdSimpleEscape(r, bs, raw, e, i)
            if (r.err != "") { return }
            continue
        }
        var v = 0
        var k = 0
        while (k < count && i < n && pycdHexVal(bs[i]) >= 0) {
            v = v * 16 + pycdHexVal(bs[i])
            i = i + 1
            k = k + 1
        }
        if (k == count && v > 1114111) {
            why = raw ? "\\Uxxxxxxxx out of range" : "illegal Unicode character"
            k = 0
        }
        if (k < count) {
            i = pycdBad(r, codec, bs, start, i, why, errs)
            if (r.err != "") { return }
            continue
        }
        r.s = r.s + pycdChr(v)
    }
}

// A backslash and the byte e that is not u, U or x, followed by bs[at]: its
// text appended to r.s; the answer is where decoding resumes.
function pycdSimpleEscape(r, bs, raw, e, at) {
    var simple = [92, 39, 34, 97, 98, 102, 110, 114, 116, 118]
    var means = [92, 39, 34, 7, 8, 12, 10, 13, 9, 11]
    var si = 0 - 1
    for (var q = 0; q < simple.length; q++) {
        if (simple[q] == e) { si = q }
    }
    var i = at
    if (raw || (si < 0 && e != 10 && e != 78 && (e < 48 || e > 55))) {
        r.s = r.s + "\\" + String.fromCharCode(e)
    } else if (si >= 0) {
        r.s = r.s + String.fromCharCode(means[si])
    } else if (e >= 48 && e <= 55) {
        var v = e - 48
        var k = 1
        while (k < 3 && i < bs.length && bs[i] >= 48 && bs[i] <= 55) {
            v = v * 8 + bs[i] - 48
            i = i + 1
            k = k + 1
        }
        r.s = r.s + String.fromCharCode(v)
    } else if (e == 78) {
        r.err = "NotImplementedError"
        r.msg = "the unicode_escape codec's \\N{...} escapes are not supported"
    }
    return i
}

// Punycode (RFC 3492) as CPython's encodings.punycode writes it: the basic
// code points in order, a "-" when there were any, then each other code
// point's insertion delta as a generalized variable-length integer, smallest
// code point first and for one code point left to right.
function pycdEncodePuny(cps, out) {
    var ext = []
    for (var i = 0; i < cps.length; i++) {
        var c = cps[i]
        var at = ext.length
        while (at > 0 && ext[at - 1] > c) { at = at - 1 }
        if (c < 128) {
            out.push(c)
        } else if (at == 0 || ext[at - 1] != c) {
            ext.push(c)
            for (var m = ext.length - 1; m > at; m = m - 1) { ext[m] = ext[m - 1] }
            ext[at] = c
        }
    }
    var baselen = out.length
    if (baselen > 0) { out.push(45) }
    var oldchar = 128
    var oldindex = 0 - 1
    var bias = 72
    var points = 0
    for (var e = 0; e < ext.length; e++) {
        var ch = ext[e]
        var below = 0
        for (var j = 0; j < cps.length; j++) {
            if (cps[j] < ch) { below = below + 1 }
        }
        var delta = (below + 1) * (ch - oldchar)
        var index = 0 - 1
        for (var p = 0; p < cps.length; p++) {
            if (cps[p] < ch) { index = index + 1 }
            if (cps[p] == ch) {
                index = index + 1
                delta = delta + index - oldindex
                pycdPunyDigits(out, delta - 1, bias)
                bias = pycdPunyAdapt(pycdBnOf(delta - 1), points == 0, baselen + points + 1)
                points = points + 1
                oldindex = index
                delta = 0
            }
        }
        oldchar = ch
    }
}

// RFC 3492's threshold t(j) for tmin 1, tmax 26, base 36.
function pycdPunyT(j, bias) {
    var t = 36 * (j + 1) - bias
    if (t < 1) { return 1 }
    if (t > 26) { return 26 }
    return t
}

function pycdPunyDigits(out, n0, bias) {
    var digits = "abcdefghijklmnopqrstuvwxyz0123456789"
    var n = n0
    var j = 0
    var t = pycdPunyT(0, bias)
    while (n >= t) {
        out.push(digits.charCodeAt(t + (n - t) % (36 - t)))
        n = Math.floor((n - t) / (36 - t))
        j = j + 1
        t = pycdPunyT(j, bias)
    }
    out.push(digits.charCodeAt(n))
}

// The bias after a delta (a pycdBn, which a malformed string makes arbitrarily
// large) of the first or a later code point, numchars code points in.
function pycdPunyAdapt(delta, first, numchars) {
    var d = pycdBnDiv(delta, first ? 700 : 2).q
    d = pycdBnAdd(d, pycdBnDiv(d, numchars).q)
    var divisions = 0
    while (pycdBnCmp(d, 455) > 0) {
        d = pycdBnDiv(d, 35).q
        divisions = divisions + 36
    }
    var v = pycdBnNum(d)
    return divisions + Math.floor(36 * v / (v + 38))
}

// Decode punycode: the basic code points are the ASCII text before the last
// "-" (decoded under errs, the ascii codec's errors), the rest the deltas,
// case-insensitively. Only strict, replace and ignore are handled; the last
// two stop at the first malformed delta and answer the text so far, and turn
// a code point past U+10FFFF into "?". A delta is unbounded, so the
// arithmetic is on pycdBn values.
function pycdDecodePuny(r, bs, errs) {
    if (errs != "strict" && errs != "replace" && errs != "ignore") {
        r.err = "UnicodeError"
        r.msg = "Unsupported error handling: " + errs
        return
    }
    pycdPunyRun(r, bs, 0, bs.length, errs, "punycode")
}

// Decode the punycode bs[lo:hi], whose errors name codec and count positions in
// the whole of bs; the basic code points' error names the ascii codec when
// codec is punycode itself.
function pycdPunyRun(r, bs, lo, hi, errs, codec) {
    var dash = 0 - 1
    for (var i = lo; i < hi; i++) {
        if (bs[i] == 45) { dash = i }
    }
    var base = []
    for (var b = lo; b < dash; b++) {
        if (bs[b] < 128) {
            base.push(bs[b])
        } else if (errs == "strict") {
            r.err = "UnicodeDecodeError"
            r.msg = "'" + (codec == "punycode" ? "ascii" : codec) + "' codec can't decode byte 0x" + pycdHex(bs[b], 2) +
                    " in position " + b + ": ordinal not in range(128)"
            return
        } else if (errs == "replace") {
            base.push(65533)
        }
    }
    var xs = dash < 0 ? lo : dash + 1
    var ext = []
    for (var x = xs; x < hi; x++) {
        var u = bs[x]
        if (u >= 97 && u <= 122) { u = u - 32 }
        ext.push(u)
    }
    var ch = pycdBnOf(128)
    var pos = pycdBnOf(0)
    var first = true
    var bias = 72
    var extpos = 0
    var why = ""
    while (extpos < ext.length && why == "") {
        var delta = pycdBnOf(0)
        var w = pycdBnOf(1)
        var j = 0
        var done = false
        var from = extpos
        while (!done && why == "") {
            if (extpos >= ext.length) {
                why = "incomplete punycode string"
                from = extpos
            } else {
                var c = ext[extpos]
                extpos = extpos + 1
                var digit = 0 - 1
                if (c >= 65 && c <= 90) { digit = c - 65 }
                if (c >= 48 && c <= 57) { digit = c - 22 }
                if (digit < 0) {
                    why = "Invalid extended code point '" + c + "'"
                    from = extpos - 1
                } else {
                    var t = pycdPunyT(j, bias)
                    delta = pycdBnAdd(delta, pycdBnMul(w, digit))
                    if (digit < t) { done = true }
                    w = pycdBnMul(w, 36 - t)
                    j = j + 1
                }
            }
        }
        if (why != "") {
            if (errs == "strict") { pycdPunyErr(r, bs, pycdBnOf(xs + from), why, codec) }
            break
        }
        pos = pycdBnAdd(pos, delta)
        if (!first) { pos = pycdBnAdd(pos, pycdBnOf(1)) }
        var qr = pycdBnDiv(pos, base.length + 1)
        ch = pycdBnAdd(ch, qr.q)
        if (pycdBnCmp(ch, 1114111) > 0) {
            if (errs == "strict") {
                var start = xs < 1 ? pycdBnPred(pos) : pycdBnAdd(pos, pycdBnOf(xs - 1))
                pycdPunyErr(r, bs, start, "Invalid character U+" + pycdBnHex(ch), codec)
                break
            }
            ch = pycdBnOf(63)
        }
        var at = qr.r
        base.push(0)
        for (var m = base.length - 1; m > at; m = m - 1) { base[m] = base[m - 1] }
        base[at] = pycdBnNum(ch)
        bias = pycdPunyAdapt(delta, first, base.length)
        pos = pycdBnOf(at)
        first = false
    }
    if (r.err != "") { return }
    for (var k = 0; k < base.length; k++) { r.s = r.s + pycdChr(base[k]) }
}

// The UnicodeDecodeError of a punycode string at position at (a pycdBn),
// which the incomplete-string error places one past the last byte. An end
// position past a C ssize_t is CPython's OverflowError instead.
function pycdPunyErr(r, bs, at, why, codec) {
    var s = pycdBnDec(at)
    var end = pycdBnDec(pycdBnAdd(at, pycdBnOf(1)))
    if (end.length > 19 || (end.length == 19 && end > "9223372036854775807")) {
        r.err = "OverflowError"
        r.msg = "Python int too large to convert to C ssize_t"
        return
    }
    r.err = "UnicodeDecodeError"
    if (pycdBnCmp(at, bs.length - 1) <= 0) {
        r.msg = "'" + codec + "' codec can't decode byte 0x" + pycdHex(bs[pycdBnNum(at)], 2) + " in position " + s + ": " + why
    } else {
        r.msg = "'" + codec + "' codec can't decode bytes in position " + s + "-" + s + ": " + why
    }
}

// A non-negative integer of any size as little-endian base-10^7 limbs, worked
// on only by small numbers, so every intermediate stays exact.
function pycdBnOf(v0) {
    var out = []
    var v = v0
    while (v > 0) {
        out.push(v % 10000000)
        v = Math.floor(v / 10000000)
    }
    return out
}
function pycdBnAdd(a, b) {
    var out = []
    var carry = 0
    var n = a.length > b.length ? a.length : b.length
    for (var i = 0; i < n; i++) {
        var s = (i < a.length ? a[i] : 0) + (i < b.length ? b[i] : 0) + carry
        carry = s >= 10000000 ? 1 : 0
        out.push(s - carry * 10000000)
    }
    if (carry > 0) { out.push(carry) }
    return out
}
function pycdBnMul(a, m) {
    var out = []
    var carry = 0
    for (var i = 0; i < a.length; i++) {
        var p = a[i] * m + carry
        carry = Math.floor(p / 10000000)
        out.push(p - carry * 10000000)
    }
    while (carry > 0) {
        out.push(carry % 10000000)
        carry = Math.floor(carry / 10000000)
    }
    if (m == 0) { return [] }
    return out
}
// a - 1, for a of at least 1.
function pycdBnPred(a) {
    var out = []
    var borrow = 1
    for (var i = 0; i < a.length; i++) {
        var d = a[i] - borrow
        borrow = d < 0 ? 1 : 0
        out.push(d + borrow * 10000000)
    }
    while (out.length > 0 && out[out.length - 1] == 0) { out.pop() }
    return out
}
// {q, r}: the quotient (a pycdBn) and remainder of a by the small m.
function pycdBnDiv(a, m) {
    var q = []
    for (var z = 0; z < a.length; z++) { q.push(0) }
    var rem = 0
    for (var i = a.length - 1; i >= 0; i = i - 1) {
        var cur = rem * 10000000 + a[i]
        var d = Math.floor(cur / m)
        q[i] = d
        rem = cur - d * m
    }
    while (q.length > 0 && q[q.length - 1] == 0) { q.pop() }
    return {q: q, r: rem}
}
// The sign of a - v, for a small v.
function pycdBnCmp(a, v) {
    if (a.length > 2) { return 1 }
    var x = pycdBnNum(a)
    if (x > v) { return 1 }
    if (x < v) { return 0 - 1 }
    return 0
}
// The value of a pycdBn of at most two limbs.
function pycdBnNum(a) {
    var v = 0
    for (var i = a.length - 1; i >= 0; i = i - 1) { v = v * 10000000 + a[i] }
    return v
}
function pycdBnDec(a) {
    if (a.length == 0) { return "0" }
    var s = "" + a[a.length - 1]
    for (var i = a.length - 2; i >= 0; i = i - 1) {
        var d = "" + a[i]
        while (d.length < 7) { d = "0" + d }
        s = s + d
    }
    return s
}
function pycdBnHex(a0) {
    var a = a0
    var s = ""
    while (a.length > 0) {
        var qr = pycdBnDiv(a, 16)
        s = "0123456789abcdef".charAt(qr.r) + s
        a = qr.q
    }
    return s
}

// IDNA (RFC 3490) as CPython's encodings.idna applies it, strict errors only.
// An ASCII name checks its labels between "." only; any other name splits at
// the four IDNA dots and each label goes through ToASCII (pycdIdnaToAscii).
function pycdEncodeIdna(r, cps, errs) {
    if (errs != "strict") {
        r.err = "UnicodeError"
        r.msg = "Unsupported error handling: " + errs
        return
    }
    var ascii = true
    for (var i = 0; i < cps.length; i++) {
        if (cps[i] >= 128) { ascii = false }
    }
    var labels = pycdIdnaSplit(cps, ascii)
    var last = labels.length - 2
    if (ascii) {
        for (var e = 0; e < last; e = e + 2) {
            if (labels[e + 1] == labels[e]) {
                pycdIdnaEncErr(r, cps, labels[e], labels[e] + 1, "label empty")
                return
            }
        }
        for (var k = 0; k <= last; k = k + 2) {
            if (labels[k + 1] - labels[k] >= 64) {
                pycdIdnaEncErr(r, cps, labels[k], labels[k + 1], "label too long")
                return
            }
        }
        for (var c = 0; c < cps.length; c++) { r.b.push(cps[c]) }
        return
    }
    if (labels[last + 1] == labels[last] && labels.length > 2) { last = last - 2 }
    for (var q = 0; q <= last; q = q + 2) {
        if (q > 0) { r.b.push(46) }
        var lab = []
        for (var j = labels[q]; j < labels[q + 1]; j++) { lab.push(cps[j]) }
        var t = pycdIdnaToAscii(lab)
        if (t.refused) {
            pycdIdnaRefuse(r)
            return
        }
        if (t.why != "") {
            pycdIdnaEncErr(r, cps, labels[q] + t.lo, labels[q] + t.hi, t.why)
            return
        }
        for (var b = 0; b < t.b.length; b++) { r.b.push(t.b[b]) }
    }
    if (last < labels.length - 2) { r.b.push(46) }
}

// ToASCII of one label (RFC 3490 4.1, CPython's encodings.idna.ToASCII): its
// bytes, or the error's reason and [lo, hi) within the label.
function pycdIdnaToAscii(lab) {
    var t = {b: [], why: "", lo: 0, hi: 0, refused: false}
    var cps = lab
    if (!pycdAllAscii(cps)) {
        var np = pycdNameprep(cps)
        if (np.why != "") { return np }
        cps = np.b
    }
    if (pycdAllAscii(cps)) {
        if (cps.length > 0 && cps.length < 64) {
            t.b = cps
            return t
        }
        return pycdIdnaWhy(t, 0, cps.length == 0 ? 1 : cps.length, cps.length == 0 ? "label empty" : "label too long")
    }
    if (cps.length >= 4 && pycdIdnaAce(pycdLowerAscii(cps), 0)) { return pycdIdnaWhy(t, 0, 4, "Label starts with ACE prefix") }
    var puny = []
    pycdEncodePuny(cps, puny)
    t.b = [120, 110, 45, 45].concat(puny)
    if (t.b.length >= 64) { return pycdIdnaWhy(t, 0, cps.length, "label too long") }
    return t
}

function pycdIdnaWhy(t, lo, hi, why) {
    t.why = why
    t.lo = lo
    t.hi = hi
    return t
}

function pycdAllAscii(cps) {
    for (var i = 0; i < cps.length; i++) {
        if (cps[i] >= 128) { return false }
    }
    return true
}

function pycdLowerAscii(cps) {
    var out = []
    for (var i = 0; i < cps.length; i++) { out.push(cps[i] >= 65 && cps[i] <= 90 ? cps[i] + 32 : cps[i]) }
    return out
}

// Nameprep (RFC 3491) as CPython's encodings.idna.nameprep applies it, over the
// tables of lib/python-nameprep.txt (CPython's stringprep module and
// unicodedata.ucd_3_2_0), one entry per line, code points in hex:
//   x<lo> <hi>      B.1, mapped to nothing
//   m<cp> <x>...    B.2, the case-folding map
//   p<lo> <hi>      prohibited: C.1.2, C.2.2 and C.3 to C.9
//   r<lo> <hi>      D.1, RandALCat
//   l<lo> <hi>      D.2, LCat
//   u<lo> <hi>      unassigned in Unicode 3.2
//   n<cp> <x>...    Unicode 3.2's NFKC where it differs from today's
// NFKC is lib/js-norm.js's, over today's tables. A code point Unicode 3.2 left
// unassigned is a starter there that neither decomposes nor composes, so the
// text normalized piece by piece between those code points (and the n ones and
// lone surrogates) is exactly Unicode 3.2's NFKC. Unassigned code points are
// allowed, as CPython allows them. pycdPrepNeed is a knob: it makes both table
// sets ready (the interpreter loads the files) or answers false (a compiled
// program registers them at boot through js_pynameprepdata).
var pycdPrepRuns = {}    // kind letter -> [lo, hi, lo, hi, ...], ascending
var pycdPrepMap = {}     // "$" + cp -> the code points of its m or n line
var pycdPrepReady = [false]
var pycdPrepNeed = function() { return pycdPrepReady[0] }

function pycdPrepLoad(t) {
    if (pycdPrepReady[0]) { return }
    pycdPrepReady[0] = true
    var kinds = "xprlu"
    for (var k = 0; k < kinds.length; k++) { pycdPrepRuns[kinds.charAt(k)] = [] }
    var i = 0
    while (i < t.length) {
        var end = i
        while (end < t.length && t.charCodeAt(end) != 10) { end = end + 1 }
        var kind = t.charAt(i)
        var ns = jsnormNums(t, i + 1, end, false)
        if (kind == "m" || kind == "n") {
            var m = []
            for (var j = 1; j < ns.length; j++) { m.push(ns[j]) }
            pycdPrepMap[kind + ns[0]] = m
        } else {
            pycdPrepRuns[kind].push(ns[0])
            pycdPrepRuns[kind].push(ns[1])
        }
        i = end + 1
    }
}

// Whether cp lies in one of the runs of kind.
function pycdPrepIn(kind, cp) {
    var runs = pycdPrepRuns[kind]
    var lo = 0
    var hi = runs.length / 2 - 1
    while (lo <= hi) {
        var mid = Math.floor((lo + hi) / 2)
        if (cp < runs[2 * mid]) { hi = mid - 1 }
        else if (cp > runs[2 * mid + 1]) { lo = mid + 1 }
        else { return true }
    }
    return false
}

// The nameprepped label, as {b: code points}, or the error's reason and
// [lo, hi) within it.
function pycdNameprep(cps) {
    var t = {b: [], why: "", lo: 0, hi: 0, refused: false}
    if (!pycdPrepNeed()) {
        t.refused = true
        return pycdIdnaWhy(t, 0, 0, "nameprep")
    }
    var mapped = []
    for (var i = 0; i < cps.length; i++) {
        var c = cps[i]
        if (pycdPrepIn("x", c)) { continue }
        var mv = pycdPrepMap["m" + c]
        if (mv === undefined) { mapped.push(c) } else { for (var j = 0; j < mv.length; j++) { mapped.push(mv[j]) } }
    }
    var out = t.b
    var piece = []
    for (var k = 0; k <= mapped.length; k++) {
        var d = k < mapped.length ? mapped[k] : 0 - 1
        var nv = d < 0 ? undefined : pycdPrepMap["n" + d]
        if (d >= 0 && nv === undefined && !pycdPrepIn("u", d) && (d < 55296 || d > 57343)) {
            piece.push(d)
            continue
        }
        var norm = jsnormApply(pycdCpsStr(piece), "NFKC")
        pycdStrCps(norm, out)
        piece = []
        if (nv !== undefined) { for (var n = 0; n < nv.length; n++) { out.push(nv[n]) } }
        else if (d >= 0) { out.push(d) }
    }
    for (var p = 0; p < out.length; p++) {
        if (pycdPrepIn("p", out[p])) { return pycdIdnaWhy(t, p, p + 1, "Invalid character " + pycdCpRepr(out[p])) }
    }
    var ral = false
    for (var a = 0; a < out.length; a++) {
        if (pycdPrepIn("r", out[a])) { ral = true }
    }
    if (ral) {
        for (var l = 0; l < out.length; l++) {
            if (pycdPrepIn("l", out[l])) { return pycdIdnaWhy(t, l, l + 1, "Violation of BIDI requirement 2") }
        }
        if (!pycdPrepIn("r", out[0])) { return pycdIdnaWhy(t, 0, 1, "Violation of BIDI requirement 3") }
        if (!pycdPrepIn("r", out[out.length - 1])) {
            return pycdIdnaWhy(t, out.length - 1, out.length, "Violation of BIDI requirement 3")
        }
    }
    return t
}

// Code points as a UTF-16 string, and back (a pair is one code point).
function pycdCpsStr(cps) {
    var s = ""
    for (var i = 0; i < cps.length; i++) {
        var v = cps[i]
        if (v >= 65536) {
            s = s + String.fromCharCode(55296 + Math.floor((v - 65536) / 1024)) + String.fromCharCode(56320 + (v - 65536) % 1024)
        } else {
            s = s + String.fromCharCode(v)
        }
    }
    return s
}
function pycdStrCps(s, out) {
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
            var d = s.charCodeAt(i + 1)
            if (d >= 56320 && d <= 57343) {
                out.push(65536 + (c - 55296) * 1024 + (d - 56320))
                i = i + 2
                continue
            }
        }
        out.push(c)
        i = i + 1
    }
}

// repr() of a one-character str.
function pycdCpRepr(cp) {
    if (cp == 39) { return "\"'\"" }
    if (cp == 92) { return "'\\\\'" }
    if (cp == 9) { return "'\\t'" }
    if (cp == 10) { return "'\\n'" }
    if (cp == 13) { return "'\\r'" }
    if (pyIsPrintable(cp)) { return "'" + pycdCpsStr([cp]) + "'" }
    return "'" + pycdEsc(cp) + "'"
}

// The labels of a name as [from, to) pairs of positions: split at "." only
// (ascii), or at any of the four IDNA dots.
function pycdIdnaSplit(cps, ascii) {
    var out = [0]
    for (var i = 0; i < cps.length; i++) {
        if (cps[i] == 46 || (!ascii && pycdIdnaDot(cps[i]))) {
            out.push(i)
            out.push(i + 1)
        }
    }
    out.push(cps.length)
    return out
}

function pycdIdnaDot(c) {
    return c == 46 || c == 12290 || c == 65294 || c == 65377
}


// A compiled program registers the nameprep tables only when it names the idna
// codec in a string literal (python-to-llvm-ir.abnf).
function pycdIdnaRefuse(r) {
    r.err = "NotImplementedError"
    r.msg = "the idna codec's nameprep tables are linked only into a program that names 'idna' in a string literal"
}

function pycdIdnaEncErr(r, cps, from, to, why) {
    var where = "characters in position " + from + "-" + (to - 1)
    if (to - from == 1 && from < cps.length) { where = "character '" + pycdEsc(cps[from]) + "' in position " + from }
    r.err = "UnicodeEncodeError"
    r.msg = "'idna' codec can't encode " + where + ": " + why
}

function pycdIdnaDecErr(r, bs, from, to, why) {
    var where = "bytes in position " + from + "-" + (to - 1)
    if (to - from == 1 && from < bs.length) { where = "byte 0x" + pycdHex(bs[from], 2) + " in position " + from }
    r.err = "UnicodeDecodeError"
    r.msg = "'idna' codec can't decode " + where + ": " + why
}

// Decoding: a name without "xn--" anywhere is plain ASCII; otherwise each
// label between "." is ASCII text, or an "xn--" label whose punycode must
// encode back to it under ToASCII.
function pycdDecodeIdna(r, bs, errs) {
    if (errs != "strict") {
        r.err = "UnicodeError"
        r.msg = "Unsupported error handling: " + errs
        return
    }
    var ace = false
    var ascii = true
    for (var i = 0; i < bs.length; i++) {
        if (pycdIdnaAce(bs, i)) { ace = true }
        if (bs[i] >= 128) { ascii = false }
    }
    if (!ace && ascii) {
        for (var a = 0; a < bs.length; a++) { r.s = r.s + String.fromCharCode(bs[a]) }
        return
    }
    var labels = pycdIdnaSplit(bs, true)
    var last = labels.length - 2
    var dot = labels[last + 1] == labels[last] && labels.length > 2
    if (dot) { last = last - 2 }
    for (var k = 0; k <= last; k = k + 2) {
        var from = labels[k]
        var to = labels[k + 1]
        if (k > 0) { r.s = r.s + "." }
        if (to - from > 1024) {
            pycdIdnaDecErr(r, bs, from, to, "label way too long")
            return
        }
        if (to - from < 4 || !pycdIdnaAce(bs, from)) {
            for (var j = from; j < to; j++) {
                if (bs[j] >= 128) {
                    pycdIdnaDecErr(r, bs, j, j + 1, "ordinal not in range(128)")
                    return
                }
                r.s = r.s + String.fromCharCode(bs[j])
            }
            continue
        }
        var pr = {err: "", msg: "", b: [], s: ""}
        pycdPunyRun(pr, bs, from + 4, to, "strict", "idna")
        if (pr.err != "") {
            r.err = pr.err
            r.msg = pr.msg
            return
        }
        var res = []
        pycdStrCps(pr.s, res)
        var t = pycdIdnaToAscii(res)
        if (t.refused) {
            pycdIdnaRefuse(r)
            return
        }
        if (t.why != "") {
            pycdIdnaDecErr(r, bs, from + t.lo, from + t.hi, t.why)
            return
        }
        var lab = []
        for (var q = from; q < to; q++) { lab.push(bs[q]) }
        var low = pycdLowerAscii(lab)
        var same = low.length == t.b.length
        for (var v = 0; v < low.length && same; v++) { same = low[v] == t.b[v] }
        if (!same) {
            pycdIdnaDecErr(r, bs, from, to, "IDNA does not round-trip, '" + pycdBytesRepr(lab) + "' != '" + pycdBytesRepr(t.b) + "'")
            return
        }
        r.s = r.s + pr.s
    }
    if (dot) { r.s = r.s + "." }
}

// Whether bs[i..] starts with "xn--", in any case.
function pycdIdnaAce(bs, i) {
    if (i + 4 > bs.length) { return false }
    return (bs[i] == 120 || bs[i] == 88) && (bs[i + 1] == 110 || bs[i + 1] == 78) && bs[i + 2] == 45 && bs[i + 3] == 45
}

// repr() of ASCII bytes.
function pycdBytesRepr(bs) {
    var sq = false
    var dq = false
    for (var i = 0; i < bs.length; i++) {
        if (bs[i] == 39) { sq = true }
        if (bs[i] == 34) { dq = true }
    }
    var quote = sq && !dq ? 34 : 39
    var out = "b" + String.fromCharCode(quote)
    for (var j = 0; j < bs.length; j++) {
        var c = bs[j]
        if (c == 92 || c == quote) { out = out + "\\" + String.fromCharCode(c) }
        else if (c == 9) { out = out + "\\t" }
        else if (c == 10) { out = out + "\\n" }
        else if (c == 13) { out = out + "\\r" }
        else if (c < 32 || c >= 127) { out = out + "\\x" + pycdHex(c, 2) }
        else { out = out + String.fromCharCode(c) }
    }
    return out + String.fromCharCode(quote)
}
