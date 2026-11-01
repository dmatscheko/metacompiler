package abnf

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// The codecs and error handlers of str.encode and bytes.decode. Twin of
// languages/lib/python-codecs.js, which states the rules.

// pycdNames maps each codec other than the charmap ones to its normalized
// names and aliases; "binary" stands for the codecs that are not text
// encodings.
var pycdNames = []struct {
	codec string
	names []string
}{
	{"utf-8", []string{"utf_8", "u8", "utf", "utf8", "utf8_ucs2", "utf8_ucs4", "cp65001"}},
	{"utf-8-sig", []string{"utf_8_sig"}},
	{"utf-7", []string{"utf_7", "u7", "unicode_1_1_utf_7", "utf7"}},
	{"ascii", []string{"ascii", "646", "ansi_x3.4_1968", "ansi_x3_4_1968", "ansi_x3.4_1986", "cp367", "csascii", "ibm367",
		"iso646_us", "iso_646.irv_1991", "iso_ir_6", "us", "us_ascii"}},
	{"latin-1", []string{"latin_1", "8859", "cp819", "csisolatin1", "ibm819", "iso8859", "iso8859_1", "iso_8859_1",
		"iso_8859_1_1987", "iso_ir_100", "l1", "latin", "latin1"}},
	{"utf-16", []string{"utf_16", "u16", "utf16"}},
	{"utf-16-le", []string{"utf_16_le", "utf_16le", "unicodelittleunmarked"}},
	{"utf-16-be", []string{"utf_16_be", "utf_16be", "unicodebigunmarked"}},
	{"utf-32", []string{"utf_32", "u32", "utf32"}},
	{"utf-32-le", []string{"utf_32_le", "utf_32le"}},
	{"utf-32-be", []string{"utf_32_be", "utf_32be"}},
	{"undefined", []string{"undefined"}},
	{"unicode-escape", []string{"unicode_escape"}},
	{"raw-unicode-escape", []string{"raw_unicode_escape"}},
	{"punycode", []string{"punycode"}},
	{"idna", []string{"idna"}},
	{"binary", []string{"base64_codec", "base64", "base_64"}},
	{"binary", []string{"bz2_codec", "bz2"}},
	{"binary", []string{"hex_codec", "hex"}},
	{"binary", []string{"quopri_codec", "quopri", "quoted_printable", "quotedprintable"}},
	{"binary", []string{"rot_13", "rot13"}},
	{"binary", []string{"uu_codec", "uu"}},
	{"binary", []string{"zlib_codec", "zip", "zlib"}},
}

// pycdCharmaps is the single-byte charmap codecs: name, names and aliases, and
// the run-encoded table of what bytes 0..255 decode to.
var pycdCharmaps = []struct {
	codec string
	names []string
	runs  string
}{
	{"cp037", []string{"cp037", "037", "csibm037", "ebcdic_cp_ca", "ebcdic_cp_nl", "ebcdic_cp_us", "ebcdic_cp_wt", "ibm037", "ibm039"},
		"0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 6f 6p 4i 1a 1o 14 17 3g 12 6h.3 6g 6l.3 6k 67 x 10 16 15 1n 4s 19 1b 5e 5g 5c.2 5f 5h 5j 5t 4m 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 1s 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 3i 37.8 4h 5b 5s 65.2 4u 2m 4j 4l 53 4p 4n 52 58.3 2j 2l 4v 4o 50 5z 3f 1t.9 4t 6s 6u 6q.2 6t 3h 22.9 55 6z.2 6x.2 73 2k 6v 2b.8 4y 5w 5y 5u.2 5x 1c.a 4z 63.2 61.2 4f"},
	{"cp273", []string{"cp273", "273", "csibm273", "ibm273"},
		"0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 3f 68.2 6b 6d 6f 6p 5g 1a 1o 14 17 x 12 6h.3 6g 6l.3 6k 3i 64 10 16 15 1n 2m 19 1b 5e 2j 5c.2 5f 5h 5j 5t 6u 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 4n 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 67 37.8 4h 5b 5s 65.2 4u 4i.2 4l 53 4p 1s 52 58.3 4s 3g 6da 4o 50 5z 6c 1t.9 4t 6s 4m 6q.2 6t 70 22.9 55 6z 3h 6x.2 73 5y 6v 2b.8 4y 5w 2k 5u.2 5x 1c.a 4z 63 2l 61.2 4f"},
	{"cp424", []string{"cp424", "424", "csibm424", "ebcdic_cp_he", "ibm424"},
		"0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 15c.9 4i 1a 1o 14 17 3g 12 15l.9 x 10 16 15 1n 4s 19 1b 15u.8 4m 18 11 2n 1q.2 1eke 162 1eke.2 4g 1eke.3 6c7 2o 1m z 1s 13 1p y 1eke 2p.9 4r 57 1eke.3 4x 4w 2y.9 1eke.3 54 1eke 4k 51 3i 37.8 1eke.5 4u 2m 4j 4l 53 4p 4n 52 58.3 2j 2l 4v 4o 50 5z 3f 1t.9 4t 1eke.5 3h 22.9 55 1eke.5 2k 6v 2b.8 4y 1eke.5 1c.a 4z 1eke.4 4f"},
	{"cp437", []string{"cp437", "437", "cspc8codepage437", "ibm437"},
		"0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 4i.2 4l 6g7 b6 69 6l 6r 6y 6p 5t 4q 56 5b 6xc 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp500", []string{"cp500", "500", "csibm500", "ebcdic_cp_be", "ebcdic_cp_ch", "ibm500"},
		"0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 6f 6p 2j 1a 1o 14 17 x 12 6h.3 6g 6l.3 6k 67 2l 10 16 15 1n 2m 19 1b 5e 5g 5c.2 5f 5h 5j 5t 4m 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 1s 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 3i 37.8 4h 5b 5s 65.2 4u 4i.2 4l 53 4p 4n 52 58.3 4s 3g 4v 4o 50 5z 3f 1t.9 4t 6s 6u 6q.2 6t 3h 22.9 55 6z.2 6x.2 73 2k 6v 2b.8 4y 5w 5y 5u.2 5x 1c.a 4z 63.2 61.2 4f"},
	{"cp720", []string{"cp720"},
		"0.3m 6h 6a 3o 68 3q 6f 6i.2 6g 6n 6m 3x.4 18x.2 6s 4k 18g 6z 6x 17l.4 4j 17p.h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo 186.5 18h 51 18i.9 6sh 18r.6 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp737", []string{"cp737"},
		"0.3k pd.h pv.7 q9.h qr qq qs.5 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo qx q4.3 qy q7 r0.2 qz r2 p2 p4.3 p8 pa.2 4x 6sl 6sk q2.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp775", []string{"cp775", "775", "cspc775baltic", "ibm775"},
		"0.3k 7a 70 6h 75 6c 83 6d 7b 8y 7n 9i.2 8b ah 5g.2 5l 6e 5i 99 6u 82 4i 9m.2 5y 64 6w 4j 60 5z 4k 74 8a 6r aj.2 ai 6cd 4m 4p 4u 4s 59 58 8x 4r 57 7f5.3 7b6 7c4 78 7g 7s 7q 7dv 7dd 7dj 7dp 8e 9s 7bk 7bo 7ck 7cc 7bw 7b4 7cs aa a2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 al 79 7h 7t 7r 8f 9t ab a3 am 7bs 7bg 7ew 7es 7f0 7f4 7eo 5v 67 98 8z 6t 5x 51 90 8m.2 8r.2 92 7m 91 6c9 4t 4x 6cc 5a 52 4n 6v 6ce 4w 6qh 53 55 4z 4y 7fk 4g"},
	{"cp850", []string{"cp850", "850", "cspc850multilingual", "ibm850"},
		"0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 6w 4j 60 5z b6 69 6l 6r 6y 6p 5t 4q 56 5b 4u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 5d.2 5c 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 6b 5f 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 6o 5s 5m.2 5k 8h 5p.3 7bs 7bg 7ew 7es 4m 5o 7eo 5v 67 5w 5u 6t 5x 51 72 66 62.2 61 71 65 4v 50 4t 4x 6c7 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g"},
	{"cp852", []string{"cp852", "852", "cspcp852", "ibm852"},
		"0.3k 5j 70 6h 6a 6c a7 7b 6f 8y 6j 9c.2 6m ah 5g 7a 5l 8p.2 6s 6u 8t.2 9m.2 5y 64 9w.2 8x 5z 7h 69 6l 6r 6y 78.2 al.2 7s.2 4s ai 7g 9r 4r 57 7f5.3 7b6 7c4 5d.2 7u 9q 7dv 7dd 7dj 7dp aj.2 7bk 7bo 7ck 7cc 7bw 7b4 7cs 76.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 7l 7k 7i 5n 7j 93 5p.2 7v 7bs 7bg 7ew 7es 9u a6 7eo 5v 67 5w 8z.2 94 9s.2 9g 62 9h a8 71 65 9v 50 4t kd kb jr k8 4n 6v 54 4w 4o k9 a9 9k.2 7fk 4g"},
	{"cp855", []string{"cp855", "855", "csibm855", "ibm855"},
		"0.3k uq si ur sj up sh us sk ut sl uu sm uv sn uw so ux sp uy sq uz sr v0 ss v2 su v3 sv um tq ui tm ts sw tt sx ue ti tw t0 tx t1 uc tg tv sz 4r 57 7f5.3 7b6 7c4 ud th u0 t4 7dv 7dd 7dj 7dp u1 t5 7bk 7bo 7ck 7cc 7bw 7b4 7cs u2 t6 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k u3 t7 u4 t8 u5 t9 u6 ta u7 7bs 7bg 7ew 7es tb un 7eo tr u8 tc u9 td ua te ub tf ty t2 tu sy uk to 6ja 4t uj tn tz t3 ug tk ul tp uh tl uf tj 4n 7fk 4g"},
	{"cp856", []string{"cp856"},
		"0.3k 15c.r 1eke 4j 1eke 5z 1eke.a 4u 4s 59 58 1eke 4r 57 7f5.3 7b6 7c4 1eke.3 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 1eke.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 1eke.9 7bs 7bg 7ew 7es 4m 1eke 7eo 1eke.6 51 1eke.7 4v 50 4t 4x 6c7 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g"},
	{"cp857", []string{"cp857", "857", "csibm857", "ibm857"},
		"0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 8h 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 8g 5y 64 6w 4j 60 9q.2 69 6l 6r 6y 6p 5t 7y.2 5b 4u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 5d.2 5c 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 6b 5f 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 56 4q 5m.2 5k 1eke 5p.3 7bs 7bg 7ew 7es 4m 5o 7eo 5v 67 5w 5u 6t 5x 51 1eke 5z 62.2 61 6k 73 4v 50 4t 4x 1eke 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g"},
	{"cp858", []string{"cp858", "858", "cp00858", "csibm00858", "csibm858", "ibm00858", "ibm858", "pc_multilingual_850_euro"},
		"0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 6w 4j 60 5z b6 69 6l 6r 6y 6p 5t 4q 56 5b 4u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 5d.2 5c 4p 7dv 7dd 7dj 7dp 4i 4l 7bk 7bo 7ck 7cc 7bw 7b4 7cs 6b 5f 7dm 7dg 7e1 7dy 7ds 7dc 7e4 4k 6o 5s 5m.2 5k 6gc 5p.3 7bs 7bg 7ew 7es 4m 5o 7eo 5v 67 5w 5u 6t 5x 51 72 66 62.2 61 71 65 4v 50 4t 4x 6c7 5a 52 4n 6v 54 4w 4o 53 55 4z 4y 7fk 4g"},
	{"cp860", []string{"cp860", "860", "csibm860", "ibm860"},
		"0.3k 5j 70 6h 6a.2 68 5d 6f 6i 5m 6g 5p 5w 6k 5f 5e 5l 5c 5k 6s.2 6q 62 6x 5o 5x 64 4i.2 61 6g7 5v 69 6l 6r 6y 6p 5t 4q 56 5b 5u 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp861", []string{"cp861", "861", "cp_is", "csibm861", "ibm861"},
		"0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 5s 6o 66 5g.2 5l 6e 5i 6s 6u 72 6z 65 71 5y 64 6w 4j 60 6g7 b6 69 6l 6r 6y 5d 5p 5v 62 5b 6xc 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp862", []string{"cp862", "862", "cspc862latinhebrew", "ibm862"},
		"0.3k 15c.r 4i.2 4l 6g7 b6 69 6l 6r 6y 6p 5t 4q 56 5b 6xc 4s 59 58 4h 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp863", []string{"cp863", "863", "csibm863", "ibm863"},
		"0.3k 5j 70 6h 6a 5e 68 52 6f 6i.2 6g 6n 6m 6c7 5c 4n 5l 5k 5m 6s 5n 5r 6z 6x 4k 5w 64 4i.2 61 63 b6 4m 50 6r 6y 4o 54 4z 4v 5q 6xc 4s 59 58 5a 4r 57 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp864", []string{"cp864", "864", "csibm864", "ibm864"},
		"0.11 19m 12.2i 4w 53 6qh.2 7f6 7b4 7b6 7cs 7c4 7cc 7bw 7ck 7bk 7bg 7bo 7bs qa 6qm qu 4x 59 58 6rs 4r 57 1ed3.2 1eke.2 1ed7.2 1eke 4g 4t 1e9u 4j.2 1e9w 1eke.2 1ea6.2 1ead 1eah 170 1eal 1eap 1eat 19c.a 1ec1 17f 1eb5 1eb9 1ebd 17j 4i 1e9s.2 1e9v 1e9x 1ebu 1ea3 1ea5 1ea9 1eab 1eaf 1eaj 1ean 1ear 1eav 1eax 1eaz 1eb1 1eb3 1eb7 1ebb 1ebf 1ebj 1ebl 1ebp 1ebv 1ebz 4m 4s 6v 5z 1ebt 18g 1ec3 1ec7 1ecb 1ecf 1ecj 1ecn 1ecr 1ect 1ecv 1ecz 1ebh 1ebw 1eby 1ebx 1ech 1e9p 18x 1ecl 1ecp 1ecs 1ecw 1ecy 1ec0 1ec5 1ed1.2 1ecd 1ec9 1ecx 7fk 1eke"},
	{"cp865", []string{"cp865", "865", "csibm865", "ibm865"},
		"0.3k 5j 70 6h 6a 6c 68 6d 6f 6i.2 6g 6n 6m 6k 5g.2 5l 6e 5i 6s 6u 6q 6z 6x 73 5y 64 6w 4j 60 6g7 b6 69 6l 6r 6y 6p 5t 4q 56 5b 6xc 4s 59 58 4h 4r 4k 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo q9 67 pf qo pv qr 51 qs py pk q1 qc 6qm qu qd 6qx 6sh 4x 6sl 6sk 6xs.2 6v 6rs 4w 6qh 53 6qi 6f3 4y 7fk 4g"},
	{"cp866", []string{"cp866", "866", "csibm866", "ibm866"},
		"0.3k sw.1c 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo u8.g sh up sk us sn uv su v2 4w 6qh 53 6qi 6ja 4k 7fk 4g"},
	{"cp869", []string{"cp869", "869", "cp_gr", "csibm869", "ibm869"},
		"0.3k 1eke.6 p2 1eke 53 4s 4m 6c8.2 p4 6c5 p5.2 q2 p8 1eke.2 pa q3 4p pb 4y.2 q4 4j q5.3 qy pc r0.2 pd.7 59 pk.2 4r 57 7f5.3 7b6 7c4 pm.4 7dv 7dd 7dj 7dp pq.2 7bk 7bo 7ck 7cc 7bw 7b4 7cs ps.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 pv.7 q9.3 7bs 7bg 7ew 7es qc.2 7eo qe.c qr qq qs p0 4t 4x qt.3 4n qw p1 4w 4o qx qz q8 r2 7fk 4g"},
	{"cp874", []string{"cp874", "874", "ms874", "windows_874"},
		"0.3k 6gc 1eke.4 6cm 1eke.b 6c8.2 6cc.2 6ci 6c3.2 1eke.8 4g 2rl.1m 1eke.4 2tb.t 1eke.4"},
	{"cp875", []string{"cp875"},
		"0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w pd.9 2j 1a 1o 14 17 x 12 pm.8 pv 2l 10 16 15 1n 2m 19 1b pw.8 3g 18 11 2n 1q.2 4o p2 p4.2 4g p6 p8 pa.2 2o 1m z 1s 13 1p y p1 2p.9 q9.6 4w 2y.9 qf.6 50 3i 37.8 ql.5 qr 4j q4.3 qy q7 r0.2 qz r2 qq qs.5 3f 1t.9 4t qx pc q8 6c8 6c5 3h 22.9 4x 59 q p3 6c9 4m 2k q 2b.8 4y 4n q q 4r.2 1c.a 4z 4p q q 57 4f"},
	{"cp1006", []string{"cp1006"},
		"0.4h 1dc.a 170 17f 4t 17j 1e9t 1ea5.2 1ea6.2 1ea9 1dna 1dnc 1eab 1ead 1eaf 1dnq 1dns 1eah 1eaj 1eal 1ean 1doa 1doc 1eap 1ear 1eat 1eav 1eax 1dok 1eaz 1eb1 1dos 1eb3 1doq 1eb5 1eb7 1eb9 1ebb 1ebd 1ebf 1ebh 1ebj 1ebl 1ebp 1ebt.9 1ec3 1ec5 1ec7 1ec9 1ecb 1doy 1dp0 1ecd 1ecf.3 1ecj 1dpa 1ecl 1ecn 1e9x 1ect 1dpi 1dpk.3 1e9s 1ea1.3 1ecx.3 1dps 1dpq 1e9o.2"},
	{"cp1026", []string{"cp1026", "1026", "csibm1026", "ibm1026"},
		"0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 3f 6p 5j 1a 1o 14 17 x 12 6h.3 6g 6l.3 6k 67 7y 8g 16 15 1n 2m 19 1b 5e 5g 5c.2 5f 5h 2j 5t 9r 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 8h 1m 5y 9q 13 1p 64 60 2p.9 4r 57 3h 2o 4m 4x 4w 2y.9 4q 56 6e 54 5i 4k 51 6u 37.8 4h 5b 2l 10 1s 4u 4i.2 4l 53 4p 4n 52 58.3 4s 3g 4v 4o 50 5z 6f 1t.9 4t 6s 3i 6q.2 6t 7z 22.9 55 6z 2k 6x.2 73 70 6v 2b.8 4y 5w z 5u.2 5x 1c.a 4z 63 y 61.2 4f"},
	{"cp1125", []string{"cp1125", "1125", "cp866u", "ibm1125", "ruscii"},
		"0.3k sw.1c 7f5.3 7b6 7c4 7dt.2 7di 7dh 7dv 7dd 7dj 7dp 7do 7dn 7bk 7bo 7ck 7cc 7bw 7b4 7cs 7dq.2 7dm 7dg 7e1 7dy 7ds 7dc 7e4 7dz.2 7dw.2 7dl 7dk 7de.2 7e3 7e2 7bs 7bg 7ew 7es 7f0 7f4 7eo u8.g sh up wg.2 sk us sm uu sn uv 53 6qi 6ja 4k 7fk 4g"},
	{"cp1140", []string{"cp1140", "1140", "cp01140", "csibm01140", "ebcdic_us_37_euro", "ibm01140", "ibm1140"},
		"0.4 4c 9 3q 3j 47 3x.2 b.9 4d 3p 8 3r o.2 42 3z s.4 3k.5 a n r 3s.5 5.3 40.2 m 43.4 4 48.4 k.2 4e q w 4g 6a 6c 68.2 6b 6d 6f 6p 4i 1a 1o 14 17 3g 12 6h.3 6g 6l.3 6k 67 x 10 16 15 1n 4s 19 1b 5e 5g 5c.2 5f 5h 5j 5t 4m 18 11 2n 1q.2 6w 5l.3 5k 5p.3 5o 2o 1m z 1s 13 1p y 60 2p.9 4r 57 6o 71.2 4x 4w 2y.9 4q 56 6e 54 5i 6gc 51 3i 37.8 4h 5b 5s 65.2 4u 2m 4j 4l 53 4p 4n 52 58.3 2j 2l 4v 4o 50 5z 3f 1t.9 4t 6s 6u 6q.2 6t 3h 22.9 55 6z.2 6x.2 73 2k 6v 2b.8 4y 5w 5y 5u.2 5x 1c.a 4z 63.2 61.2 4f"},
	{"cp1250", []string{"cp1250", "1250", "windows_1250"},
		"0.3k 6gc 1eke 6ca 1eke 6ce 6cm 6cg.2 1eke 6cw 9s 6d5 9m 9w al ah 1eke 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 9t 6d6 9n 9x am ai 4g jr k8 8x 4k 78 4m.4 9q 4r.4 aj 4w.2 kb 8y 50.5 79 9r 57 8t kd 8u ak 9g 5d.2 76 5g 8p 7a 5j 7g 5l 7s 5n 7u 5p.2 7i 7k 8z 93 5v.2 9c 5y.2 9k a6 62 a8 64.2 9u 67 9h 69.2 77 6c 8q 7b 6f 7h 6h 7t 6j 7v 6l.2 7j 7l 90 94 6r.2 9d 6u.2 9l a7 6y a9 70.2 9v k9"},
	{"cp1251", []string{"cp1251", "1251", "windows_1251"},
		"0.3k si.2 6ca ur 6ce 6cm 6cg.2 6gc 6cw sp 6d5 sq ss sr sv uq 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm ux 6d6 uy v0 uz v3 4g su v2 so 4k wg 4m.2 sh 4p sk 4r.4 sn 4w.2 sm uu wh 51.3 up 6ja us 57 uw sl ut uv sw.1s"},
	{"cp1252", []string{"cp1252", "1252", "windows_1252"},
		"0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 9s 6d5 9e 1eke al 1eke.2 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 9t 6d6 9f 1eke am ag 4g.2o"},
	{"cp1253", []string{"cp1253", "1253", "windows_1253"},
		"0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 1eke 6cw 1eke 6d5 1eke.5 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 1eke 6d6 1eke.4 4g p1.2 4j.7 1eke 4r.4 6c5 4w.4 p0 51.3 p4.3 57 p8 59 pa.k 1eke pv.18 1eke"},
	{"cp1254", []string{"cp1254", "1254", "windows_1254"},
		"0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 9s 6d5 9e 1eke.4 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 9t 6d6 9f 1eke.2 ag 4g.1c 7y 5t.c 8g 9q 67.h 7z 6p.c 8h 9r 73"},
	{"cp1255", []string{"cp1255", "1255", "windows_1255"},
		"0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 1eke 6d5 1eke.5 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 1eke 6d6 1eke.4 4g.4 6ga 4l.5 5z 4r.f 6v 57.5 14g.a 1eke 14r.9 168.5 1eke.7 15c.r 1eke.2 6by.2 1eke"},
	{"cp1256", []string{"cp1256", "1256", "windows_1256"},
		"0.3k 6gc 1a6 6ca b6 6ce 6cm 6cg.2 jq 6cw 1a1 6d5 9e 1ae 1aw 1ag 1bj 6c8.2 6cc.2 6ci 6c3.2 1bd 6jm 1ap 6d6 9f 6bw.2 1bu 4g 170 4i.8 1by 4r.f 17f 57.4 17j 1c1 17l.m 5z 187.4 18g.4 68 18k 6a 18l.4 6f.5 18p.2 6m.2 18r.4 6s 18v.2 6v 18x 6x 18y 6z.2 6by.2 1ci"},
	{"cp1257", []string{"cp1257", "1257", "windows_1257"},
		"0.3k 6gc 1eke 6ca 1eke 6ce 6cm 6cg.2 1eke 6cw 1eke 6d5 1eke 4o jr 54 1eke 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 1eke 6d6 1eke 4v kb 1eke 4g 1eke 4i.3 1eke 4m.2 60 4p 9i 4r.4 5i 4w.8 6w 55 9j 57.4 6e 78 8e 74 7a 5g.2 7s 7m 7g 5l ah 7q 82 8m 8a 8r 9s 8z 91 5v 98 5x.3 aa 8x 9m a2 64 aj al 67 79 8f 75 7b 6c.2 7t 7n 7h 6h ai 7r 83 8n 8b 8s 9t 90 92 6r 99 6t.3 ab 8y 9n a3 70 ak am k9"},
	{"cp1258", []string{"cp1258", "1258", "windows_1258"},
		"0.3k 6gc 1eke 6ca b6 6ce 6cm 6cg.2 jq 6cw 1eke 6d5 9e 1eke.4 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 1eke 6d6 9f 1eke.2 ag 4g.z 76 5g.8 lc 5p.3 7k 5t ll 5v.2 bk 5y.7 bz lf 67.4 77 6c.8 ld 6l.3 7l 6p mb 6r.2 bl 6u.7 c0 6gb 73"},
	{"hp_roman8", []string{"hp_roman8", "cp1051", "csHPRoman8", "ibm1051", "r8", "roman8"},
		"0.4h 5c 5e 5k 5m.2 5q.2 50 jv jq 4o kc 61 63 6g4 4v 65 71 4w 5j 6f 5t 6p 4h 5b 4k 4j 4l 4n b6 4i 6a 6i 6s 6z 69 6h 6r 6y 68 6g 6q 6x 6c 6j 6u 70 5h 6m 60 5i 6d 6l 6w 6e 5g 6k 5y 64 5l 6n 67 5w 5d 5f 6b 5s 6o 5p 5o 5v 5u 5x 6t 9s.2 62 ag 73 66 72 53 51.2 5a 6c4 58.2 4q 56 4r 7fk 57 4x 1eke"},
	{"iso8859_2", []string{"iso8859_2", "csisolatin2", "iso_8859_2", "iso_8859_2_1987", "iso_ir_101", "l2", "latin2"},
		"0.4h 78 k8 8x 4k 8t 9m 4n.2 9s 9q 9w ah 4t al aj 4w 79 kb 8y 50 8u 9n jr 54 9t 9r 9x ai kd am ak 9g 5d.2 76 5g 8p 7a 5j 7g 5l 7s 5n 7u 5p.2 7i 7k 8z 93 5v.2 9c 5y.2 9k a6 62 a8 64.2 9u 67 9h 69.2 77 6c 8q 7b 6f 7h 6h 7t 6j 7v 6l.2 7j 7l 90 94 6r.2 9d 6u.2 9l a7 6y a9 70.2 9v k9"},
	{"iso8859_3", []string{"iso8859_3", "csisolatin3", "iso_8859_3", "iso_8859_3_1988", "iso_ir_109", "l3", "latin3"},
		"0.4h 86 k8 4j.2 1eke 84 4n.2 8g 9q 7y 8k 4t 1eke aj 4w 87 4y.4 85 53.2 8h 9r 7z 8l 59 1eke ak 5c.3 1eke 5g 7e 7c 5j.9 1eke 5t.4 80 5y.2 7w 61.4 a4 9o 67.4 1eke 6c 7f 7d 6f.9 1eke 6p.4 81 6u.2 7x 6x.4 a5 9p k9"},
	{"iso8859_4", []string{"iso8859_4", "csisolatin4", "iso_8859_4", "iso_8859_4_1988", "iso_ir_110", "l4", "latin4"},
		"0.4h 78 8o 9i 4k 88 8r 4n.2 9s 7m 82 9y 4t al 4v.2 79 kb 9j 50 89 8s jr 54 9t 7n 83 9z 96 am 97 74 5d.6 8e 7g 5l 7s 5n 7q 5p.2 8a 7k 91 98 8m 5w.5 aa 62.3 a0 a2 67 75 69.6 8f 7h 6h 7t 6j 7r 6l.2 8b 7l 92 99 8n 6s.5 ab 6y.3 a1 a3 k9"},
	{"iso8859_5", []string{"iso8859_5", "csisolatincyrillic", "cyrillic", "iso_8859_5", "iso_8859_5_1988", "iso_ir_144"},
		"0.4h sh.c 4t su.1u 6ja up.c 4n v2.2"},
	{"iso8859_6", []string{"iso8859_6", "arabic", "asmo_708", "csisolatinarabic", "ecma_114", "iso_8859_6", "iso_8859_6_1987", "iso_ir_127"},
		"0.4h 1eke.3 4k 1eke.7 170 4t 1eke.d 17f 1eke.3 17j 1eke 17l.q 1eke.5 18g.j 1eke.d"},
	{"iso8859_7", []string{"iso8859_7", "csisolatingreek", "ecma_118", "elot_928", "greek", "greek8", "iso_8859_7", "iso_8859_7_1987", "iso_ir_126"},
		"0.4h 6c8.2 4j 6gc 6gf 4m.4 oq 4r.3 1eke 6c5 4w.4 p0.3 53 p4.3 57 p8 59 pa.k 1eke pv.18 1eke"},
	{"iso8859_8", []string{"iso8859_8", "csisolatinhebrew", "hebrew", "iso_8859_8", "iso_8859_8_1988", "iso_8859_8_e", "iso_8859_8_i", "iso_ir_138"},
		"0.4h 1eke 4i.8 5z 4r.f 6v 57.4 1eke.w 6c7 15c.r 1eke.2 6by.2 1eke"},
	{"iso8859_9", []string{"iso8859_9", "csisolatin5", "iso_8859_9", "iso_8859_9_1989", "iso_ir_148", "l5", "latin5"},
		"0.5s 7y 5t.c 8g 9q 67.h 7z 6p.c 8h 9r 73"},
	{"iso8859_10", []string{"iso8859_10", "csisolatin6", "iso_8859_10", "iso_8859_10_1992", "iso_ir_157", "l6", "latin6"},
		"0.4h 78 7m 82 8a 88 8m 4n 8r 7k 9s 9y al 4t a2 96 4w 79 7n 83 8b 89 8n 53 8s 7l 9t 9z am 6c5 a3 97 74 5d.6 8e 7g 5l 7s 5n 7q 5p.4 91 98 5v.4 a0 60 aa 62.6 75 69.6 8f 7h 6h 7t 6j 7r 6l.4 92 99 6r.4 a1 6w ab 6y.5 8o"},
	{"iso8859_11", []string{"iso8859_11", "iso_8859_11", "iso_8859_11_2001", "thai"},
		"0.4h 2rl.1m 1eke.4 2tb.t 1eke.4"},
	{"iso8859_13", []string{"iso8859_13", "iso_8859_13", "l7", "latin7"},
		"0.4h 6cd 4i.3 6ce 4m.2 60 4p 9i 4r.4 5i 4w.4 6cc 51.3 6w 55 9j 57.4 6e 78 8e 74 7a 5g.2 7s 7m 7g 5l ah 7q 82 8m 8a 8r 9s 8z 91 5v 98 5x.3 aa 8x 9m a2 64 aj al 67 79 8f 75 7b 6c.2 7t 7n 7h 6h ai 7r 83 8n 8b 8s 9t 90 92 6r 99 6t.3 ab 8y 9n a3 70 ak am 6c9"},
	{"iso8859_14", []string{"iso8859_14", "iso_8859_14", "iso_8859_14_1998", "iso_celtic", "iso_ir_199", "l8", "latin8"},
		"0.4h 5xe.2 4j 7e.2 5xm 4n 60w 4p 60y 5xn 642 4t.2 ag 5y6.2 80.2 5z4.2 52 5zq 60x 5zr 60z 600 643 610.2 601 5c.g ac 5t.6 60a 60.6 ae 67.h ad 6p.6 60b 6w.6 af 73"},
	{"iso8859_15", []string{"iso8859_15", "iso_8859_15", "l9", "latin9"},
		"0.4k 6gc 4l 9s 4n 9t 4p.b al 51.3 am 55.3 9e.2 ag 5b.1t"},
	{"iso8859_16", []string{"iso8859_16", "iso_8859_16", "iso_8859_16_2001", "iso_ir_226", "l10", "latin10"},
		"0.4h 78.2 8x 6gc 6ce 9s 4n 9t 4p ew 4r ah 4t ai.2 4w.2 7g 8y al 6cd 52.2 am 7h ex 57 9e.2 ag ak 5c.3 76 5g 7a 5i.a 7k 8z 5u.3 9c 5y 9m a8 61.4 7s ey 67.4 77 6c 7b 6e.a 7l 90 6q.3 9d 6u 9n a9 6x.4 7t ez 73"},
	{"koi8_r", []string{"koi8_r", "cskoi8r"},
		"0.3k 7b4 7b6 7bg 7bk 7bo 7bs 7bw 7c4 7cc 7ck 7cs 7eo 7es 7ew 7f0 7f4.4 6xs 7fk 6qh.2 6rs 6sk.2 4g 6xt 4w 4y 53 6v 7dc.3 up 7df.f sh 7du.b 4p um ts.2 ue tw.2 uc tv ud u0.8 un u8.4 ty tu uk uj tz ug ul uh uf ui tq sw.2 ti t0.2 tg sz th t4.8 tr tc.4 t2 sy to tn t3 tk tp tl tj tm"},
	{"koi8_t", []string{"koi8_t"},
		"0.3k wr wj 6ca wi 6ce 6cm 6cg.2 1eke 6cw xf 6d5 xe xj xi 1eke wq 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm 1eke 6d6 1eke.5 z3 z2 up 4k yr 4m.2 1eke.3 4r.4 1eke 4w.3 sh 1eke yq 52.2 1eke 6ja 1eke 57 1eke.3 4p um ts.2 ue tw.2 uc tv ud u0.8 un u8.4 ty tu uk uj tz ug ul uh uf ui tq sw.2 ti t0.2 tg sz th t4.8 tr tc.4 t2 sy to tn t3 tk tp tl tj tm"},
	{"koi8_u", []string{"koi8_u"},
		"0.3k 7b4 7b6 7bg 7bk 7bo 7bs 7bw 7c4 7cc 7ck 7cs 7eo 7es 7ew 7f0 7f4.4 6xs 7fk 6qh.2 6rs 6sk.2 4g 6xt 4w 4y 53 6v 7dc.3 up us 7dg uu.2 7dj.5 wh 7dp.5 sh sk 7dv sm.2 7dy.5 wg 7e4 4p um ts.2 ue tw.2 uc tv ud u0.8 un u8.4 ty tu uk uj tz ug ul uh uf ui tq sw.2 ti t0.2 tg sz th t4.8 tr tc.4 t2 sy to tn t3 tk tp tl tj tm"},
	{"kz1048", []string{"kz1048", "kz_1048", "rk1048", "strk1048_2002"},
		"0.3k si.2 6ca ur 6ce 6cm 6cg.2 6gc 6cw sp 6d5 sq wq xm sv uq 6c8.2 6cc.2 6ci 6c3.2 1eke 6jm ux 6d6 uy wr xn v3 4g xc.2 yg 4k yw 4m.2 sh 4p wi 4r.4 xa 4w.2 sm uu yx 51.3 up 6ja wj 57 yh wy.2 xb sw.1s"},
	{"mac_arabic", []string{"mac_arabic"},
		"0.3k 5g 4g 5j 5l 5t 5y 64 69 68 6a 6c 1bu 4r 6f 6h 6g 6i.2 6l 6cm 6m.2 6p 6r 57 6s 6u.2 6y 6x 6z.2 w.5 19m 12.6 170 19.3 19c.a 1m 17f 1o.3 17j 7re 17l.q 2j.5 18g.j 1a6 1a1 1ae 1cl 1b8 1bj 1ag 1ap 3f.3 1aw 1ci"},
	{"mac_croatian", []string{"mac_croatian"},
		"0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 9s 6jm 50 4o 6sg al 60 6qm 4x 6sk.2 6py 51 6pu 6q9 6q7 9t 6qz 4q 56 q1 am 6w 5b 4h 4s 6qi b6 6rs 7a 4r 7g 6cm 4g 5c 5f 5x 9e.2 7k 6c4 6cc.2 6c8.2 6v 7gq 1d6n 4p 6dg 6gc 6d5.2 5i 57 6c3 53 6ca 6ce 6cw 5e 7b 5d 7h 5k 5p.3 5o 5v.2 7l 5u 62.2 61 8h jq kc 4v qo 5n ka 54 5m 6e jr"},
	{"mac_cyrillic", []string{"mac_cyrillic", "maccyrillic"},
		"0.3k sw.w 6cg 4w wg 4j 4n 6ci 52 sm 4u 4p 6jm si uq 6sg sj ur 6qm 4x 6sk.2 uu 51 wh so sk us sn uv sp ux sq uy uw sl 4s 6qi b6 6rs 6py 4r 57 6cm 4g sr uz ss v0 ut 6c3.2 6cc.2 6c8.2 6v 6ce su v2 sv v3 6ja sh up un ts.v 6gc"},
	{"mac_farsi", []string{"mac_farsi"},
		"0.3k 5g 4g 5j 5l 5t 5y 64 69 68 6a 6c 1bu 4r 6f 6h 6g 6i.2 6l 6cm 6m.2 6p 6r 57 6s 6u.2 6y 6x 6z.2 w.5 19m 12.6 170 19.3 1dc.a 1m 17f 1o.3 17j 7re 17l.q 2j.5 18g.j 1a6 1a1 1ae 1cl 1b8 1bj 1ag 1ap 3f.3 1aw 1ci"},
	{"mac_greek", []string{"mac_greek", "macgreek"},
		"0.3k 5g 55 4y 5l 4z 5y 64 p1 68 6a 6c p0 4o 6f 6h 6g 6i.2 4j 6jm 6m.2 6ci 59 6cw 6s 6u 4m 6gc 6x 6z.2 6cg pf.2 pk pn pq ps 67 4u 4p pv q2 4n 6sg 4w 53 pd 4x 6sk.2 4l pe ph.3 pl.2 po py q3 q0.2 q4 pp 4s pr pt 6rs pw 4r 57 6cm 4g px pz p2 p4 9f 6c3 6c5 6cc.2 6c8.2 6v p5.2 p8 pa q5.3 r0 pb r1 q9.2 qw qc.2 qu qb qf qh qm qi.4 qn.2 r2 qp qr.2 qg qx qq qv qt qe qy.2 pc q8 4t"},
	{"mac_iceland", []string{"mac_iceland", "maciceland"},
		"0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 65 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 5i 60 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 6e 6w 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 6dg 6gc 5s 6o 66 72 71 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 8h jq kc 4v k8.3 54 kd kb jr"},
	{"mac_latin2", []string{"mac_latin2", "mac_centeuro", "maccentraleurope", "maclatin2"},
		"0.3k 5g 74.2 5l 78 5y 64 69 79 7g 6c 7h 7a.2 6h ah.2 7i 6l 7j 7m.2 7q 6r 7r 6s 6u 6t 6y 7u.2 70 6cg 4w 7s 4j 4n 6ci 52 67 4u 4p 6jm 7t 4o 6sg 83 8e.2 8a 6sk.2 8b 8m 6pu 6q9 8y 8r.4 8p.2 91.2 8z 4s 6qi 90 93 6py 4r 57 6cm 4g 94 9c 5x 9d 98 6c3.2 6cc.2 6c8.2 6v 7gq 99 9g.2 9k 6d5.2 9l 9i.2 9s 6ca 6ce 9t 9m.2 5d 9w.2 5p al.2 a2 5v.2 a3 a6 62 a7.5 65 71 8n aj 8x ak 82 jr"},
	{"mac_roman", []string{"mac_roman", "macintosh", "macroman"},
		"0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 5i 60 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 6e 6w 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 6dg 6gc 6d5.2 1dkx.2 6ch 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 8h jq kc 4v k8.3 54 kd kb jr"},
	{"mac_romanian", []string{"mac_romanian"},
		"0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 76 ew 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 77 ex 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 6dg 6gc 6d5.2 ey.2 6ch 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 8h jq kc 4v k8.3 54 kd kb jr"},
	{"mac_turkish", []string{"mac_turkish", "macturkish"},
		"0.3k 5g.2 5j 5l 5t 5y 64 69 68 6a 6c 6b 6d 6f 6h 6g 6i.2 6l 6k 6m.2 6p 6r 6q 6s 6u 6t 6y 6x 6z.2 6cg 4w 4i.2 4n 6ci 52 67 4u 4p 6jm 50 4o 6sg 5i 60 6qm 4x 6sk.2 4l 51 6pu 6q9 6q7 qo 6qz 4q 56 q1 6e 6w 5b 4h 4s 6qi b6 6rs 6py 4r 57 6cm 4g 5c 5f 5x 9e.2 6c3.2 6cc.2 6c8.2 6v 7gq 73 ag 7y.2 8g.2 9q.2 6ch 53 6ca 6ce 6cw 5e 5m 5d 5n 5k 5p.3 5o 5v.2 1d6n 5u 62.2 61 1d40 jq kc 4v k8.3 54 kd kb jr"},
	{"palmos", []string{"palmos"},
		"0.3k 6gc 3l 6ca b6 6ce 6cm 6cg.2 jq 6cw 9s 6d5 9e 7l2 7kz 7l1 7kw 6c8.2 6cc.2 6ci 6c3.2 kc 6jm 9t 6d6 9f 4d.2 ag 4g.2o"},
	{"ptcp154", []string{"ptcp154", "cp154", "csptcp154", "cyrillic_asian", "pt154"},
		"0.3k wm wi z2 wj 6ce 6cm xi xa xe xb ww yq wy wq xm xk wn 6c8.2 6cc.2 6ci 6c3.2 xf xj wx yr wz wr xn xl 4g su v2 so yw wo xc 4n sh 4p yg 4r.2 z3 4u ws 4w xd sm uu wp yx 52.2 up 6ja yh 57 uw x6.2 wt sw.1s"},
	{"tis_620", []string{"tis_620", "iso_ir_166", "tis620", "tis_620_0", "tis_620_2529_0", "tis_620_2529_1"},
		"0.4g 1eke 2rl.1m 1eke.4 2tb.t 1eke.4"},
}

// pycdTabs is each charmap codec's 256 decoded code points (0xfffe where
// undefined) and pycdRevs their inverses.
var pycdTabs, pycdRevs = pycdBuildTabs()

func pycdBuildTabs() (map[string][]int, map[string]map[int]int) {
	tabs, revs := map[string][]int{}, map[string]map[int]int{}
	for _, c := range pycdCharmaps {
		var t []int
		for _, run := range strings.Fields(c.runs) {
			v, n := run, "1"
			if i := strings.IndexByte(run, '.'); i >= 0 {
				v, n = run[:i], run[i+1:]
			}
			cp, ln := pycdB36(v), pycdB36(n)
			for k := 0; k < ln; k++ {
				if cp == 0xfffe {
					t = append(t, cp)
				} else {
					t = append(t, cp+k)
				}
			}
		}
		rev := map[int]int{}
		for b, cp := range t {
			if cp != 0xfffe {
				rev[cp] = b
			}
		}
		tabs[c.codec], revs[c.codec] = t, rev
	}
	return tabs, revs
}

func pycdB36(s string) int {
	v := 0
	for i := 0; i < len(s); i++ {
		d := int(s[i] - '0')
		if s[i] >= 'a' {
			d = int(s[i]-'a') + 10
		}
		v = v*36 + d
	}
	return v
}

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
		if pycdNameIs(c.names, norm, dotless) {
			return c.codec
		}
	}
	for _, c := range pycdCharmaps {
		if pycdNameIs(c.names, norm, dotless) {
			return c.codec
		}
	}
	return ""
}

// pycdNameIs says whether a normalized name selects the codec whose module
// name is names[0] and whose aliases are the rest: an alias matches the name
// or the name with its dots made underscores, the module name only the name
// itself (encodings.search_function). Twin of pycdNameIs in
// languages/lib/python-codecs.js.
func pycdNameIs(names []string, norm, dotless string) bool {
	if names[0] == norm {
		return true
	}
	for _, a := range names[1:] {
		if a == norm || a == dotless {
			return true
		}
	}
	return false
}

// pycdRefuse is the error a codec raises before coding anything, "" if none.
func pycdRefuse(enc, codec, verb string) (exc, msg string) {
	switch codec {
	case "":
		return "LookupError", "unknown encoding: " + enc
	case "binary":
		return "LookupError", "'" + enc + "' is not a text encoding; use codecs." + verb + "() to handle arbitrary codecs"
	case "undefined":
		return "UnicodeError", "undefined encoding"
	}
	return "", ""
}

func pycdCharByte(codec string, cp int) int {
	if b, ok := pycdRevs[codec][cp]; ok {
		return b
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

func pycdEncBad(codec string, cm bool, cp int) bool {
	switch {
	case cm:
		return pycdCharByte(codec, cp) < 0
	case codec == "ascii":
		return cp >= 0x80
	case codec == "latin-1":
		return cp >= 0x100
	}
	return cp >= 0xd800 && cp <= 0xdfff
}

func pycdEncOne(out []int, codec string, cm bool, cp, size int, le bool) []int {
	switch {
	case size == 4:
		return pycdPushUnit(out, cp, 4, le)
	case size == 2 && cp >= 0x10000:
		out = pycdPushUnit(out, 0xd800+(cp-0x10000)>>10, 2, le)
		return pycdPushUnit(out, 0xdc00+(cp-0x10000)&0x3ff, 2, le)
	case size == 2:
		return pycdPushUnit(out, cp, 2, le)
	case cm:
		return append(out, pycdCharByte(codec, cp))
	case codec != "utf-8" || cp < 0x80:
		return append(out, cp)
	case cp < 0x800:
		return append(out, 0xc0|cp>>6, 0x80|cp&63)
	case cp < 0x10000:
		return append(out, 0xe0|cp>>12, 0x80|cp>>6&63, 0x80|cp&63)
	}
	return append(out, 0xf0|cp>>18, 0x80|cp>>12&63, 0x80|cp>>6&63, 0x80|cp&63)
}

// pycdEncode encodes the code points cps under the encoding named enc; a
// non-empty exc names the exception to raise with msg.
func pycdEncode(cps []int, enc, errs string) (out []int, exc, msg string) {
	codec := pycdLookup(enc)
	if exc, msg := pycdRefuse(enc, codec, "encode"); exc != "" {
		return nil, exc, msg
	}
	switch codec {
	case "utf-7":
		return pycdEncodeUtf7(cps), "", ""
	case "unicode-escape", "raw-unicode-escape":
		return pycdEncodeEscape(cps, codec == "raw-unicode-escape"), "", ""
	case "punycode":
		return pycdEncodePuny(cps), "", ""
	case "idna":
		return pycdEncodeIdna(cps, errs)
	case "utf-8-sig":
		out, codec = []int{0xef, 0xbb, 0xbf}, "utf-8"
	}
	_, cm := pycdTabs[codec]
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
	switch {
	case codec == "ascii":
		why = "ordinal not in range(128)"
	case codec == "latin-1":
		why = "ordinal not in range(256)"
	case cm:
		why, errCodec = "character maps to <undefined>", "charmap"
	}
	for k := 0; k < len(cps); {
		if !pycdEncBad(codec, cm, cps[k]) {
			out = pycdEncOne(out, codec, cm, cps[k], size, le)
			k++
			continue
		}
		j := k + 1
		for size == 0 && j < len(cps) && pycdEncBad(codec, cm, cps[j]) {
			j++
		}
		bad := errs == "strict" || (errs == "surrogateescape" && size > 0)
		if errs == "surrogateescape" && size == 0 {
			q := k
			for q < j && cps[q] >= 0xdc80 && cps[q] <= 0xdcff {
				q++
			}
			if q < j {
				if !cm {
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
				out = pycdEncOne(out, codec, cm, int(rep[i]), size, le)
			}
		}
		k = j
	}
	return out, "", ""
}

const pycdB64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func pycdB64Val(c int) int {
	if c < 0x80 {
		return strings.IndexByte(pycdB64, byte(c))
	}
	return -1
}

func pycdEncodeUtf7(cps []int) []int {
	var out []int
	shift, bits, buf := false, 0, 0
	for _, c := range cps {
		direct := c == '\t' || c == '\n' || c == '\r' || (c >= ' ' && c <= '}' && c != '+' && c != '\\')
		switch {
		case shift && direct:
			if bits > 0 {
				out = append(out, int(pycdB64[buf<<(6-bits)&63]))
			}
			bits, buf, shift = 0, 0, false
			if pycdB64Val(c) >= 0 || c == '-' {
				out = append(out, '-')
			}
			out = append(out, c)
		case !shift && c == '+':
			out = append(out, '+', '-')
		case !shift && direct:
			out = append(out, c)
		default:
			if !shift {
				out = append(out, '+')
			}
			shift = true
			units := []int{c}
			if c >= 0x10000 {
				units = []int{0xd800 + (c-0x10000)>>10, 0xdc00 + (c-0x10000)&0x3ff}
			}
			for _, u := range units {
				buf, bits = buf<<16|u, bits+16
				for bits >= 6 {
					out = append(out, int(pycdB64[buf>>(bits-6)&63]))
					bits -= 6
					buf &= 1<<bits - 1
				}
			}
		}
	}
	if bits > 0 {
		out = append(out, int(pycdB64[buf<<(6-bits)&63]))
	}
	if shift {
		out = append(out, '-')
	}
	return out
}

// pycdDecode decodes the byte values bs under the encoding named enc into
// UTF-16 code units; a non-empty exc names the exception to raise with msg.
// No bytes decode to "" under any name (no codec is looked up for them).
func pycdDecode(bs []int, enc, errs string) (out []uint16, exc, msg string) {
	if len(bs) == 0 {
		return nil, "", ""
	}
	codec := pycdLookup(enc)
	if exc, msg := pycdRefuse(enc, codec, "decode"); exc != "" {
		return nil, exc, msg
	}
	switch codec {
	case "punycode":
		return pycdDecodePuny(bs, errs)
	case "idna":
		return pycdDecodeIdna(bs, errs)
	}
	if codec == "utf-8-sig" {
		if len(bs) >= 3 && bs[0] == 0xef && bs[1] == 0xbb && bs[2] == 0xbf {
			bs = bs[3:]
		}
		codec = "utf-8"
	}
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
	tab, cm := pycdTabs[codec]
	switch {
	case codec == "latin-1":
		for _, c := range bs {
			push(c)
		}
		return out, "", ""
	case codec == "ascii" || cm:
		for i, c := range bs {
			if cm {
				c = tab[c]
			}
			switch {
			case codec == "ascii" && c >= 0x80:
				if bad("ascii", i, i+1, "ordinal not in range(128)"); exc != "" {
					return nil, exc, msg
				}
			case c == 0xfffe:
				if bad("charmap", i, i+1, "character maps to <undefined>"); exc != "" {
					return nil, exc, msg
				}
			default:
				push(c)
			}
		}
		return out, "", ""
	case codec == "utf-7":
		shift, bits, buf, sur, start := false, 0, 0, 0, 0
		for i := 0; i < n; {
			c := bs[i]
			switch {
			case shift && pycdB64Val(c) >= 0:
				buf, bits = buf<<6|pycdB64Val(c), bits+6
				i++
				if bits >= 16 {
					u := buf >> (bits - 16)
					bits -= 16
					buf &= 1<<bits - 1
					switch {
					case sur != 0 && u >= 0xdc00 && u <= 0xdfff:
						out = append(out, uint16(sur), uint16(u))
						sur = 0
					default:
						if sur != 0 {
							push(sur)
						}
						sur = 0
						if u >= 0xd800 && u <= 0xdbff {
							sur = u
						} else {
							push(u)
						}
					}
				}
			case shift:
				shift = false
				why := ""
				if bits >= 6 {
					why = "partial character in shift sequence"
				} else if bits > 0 && buf != 0 {
					why = "non-zero padding bits in shift sequence"
				}
				if why != "" {
					i = bad("utf7", start, i+1, why)
					break
				}
				if sur != 0 && c < 0x80 && c != '+' {
					push(sur)
				}
				sur = 0
				if c == '-' {
					i++
				}
			case c == '+':
				start = i
				i++
				switch {
				case i < n && bs[i] == '-':
					i++
					push('+')
				case i < n && pycdB64Val(bs[i]) < 0:
					i = bad("utf7", start, i+1, "ill-formed sequence")
				default:
					shift, sur, bits, buf = true, 0, 0, 0
				}
			case c < 0x80:
				push(c)
				i++
			default:
				i = bad("utf7", i, i+1, "unexpected special character")
			}
			if exc != "" {
				return nil, exc, msg
			}
		}
		if shift && (sur != 0 || bits >= 6 || (bits > 0 && buf != 0)) {
			if bad("utf7", start, n, "unterminated shift sequence"); exc != "" {
				return nil, exc, msg
			}
		}
		return out, "", ""
	case codec == "unicode-escape" || codec == "raw-unicode-escape":
		raw := codec == "raw-unicode-escape"
		name := strings.ReplaceAll(codec, "-", "")
		for i := 0; i < n; {
			c := bs[i]
			if c != '\\' || (raw && i+1 >= n) {
				push(c)
				i++
				continue
			}
			start := i
			if i+1 >= n {
				if bad(name, i, n, "\\ at end of string"); exc != "" {
					return nil, exc, msg
				}
				break
			}
			e := bs[i+1]
			i += 2
			count, why := 0, ""
			switch {
			case e == 'u':
				count, why = 4, "truncated \\uXXXX escape"
			case e == 'U':
				count, why = 8, "truncated \\UXXXXXXXX escape"
			case e == 'x' && !raw:
				count, why = 2, "truncated \\xXX escape"
			}
			if count == 0 {
				if j := strings.IndexByte("\\'\"abfnrtv", byte(e)); !raw && j >= 0 {
					push(int("\\'\"\a\b\f\n\r\t\v"[j]))
				} else if !raw && e >= '0' && e <= '7' {
					v := e - '0'
					for k := 1; k < 3 && i < n && bs[i] >= '0' && bs[i] <= '7'; k++ {
						v = v*8 + bs[i] - '0'
						i++
					}
					push(v)
				} else if !raw && e == 'N' {
					return nil, "NotImplementedError", "the unicode_escape codec's \\N{...} escapes are not supported"
				} else if raw || e != '\n' {
					push('\\')
					push(e)
				}
				continue
			}
			v, k := 0, 0
			for ; k < count && i < n && pycdHexVal(bs[i]) >= 0; k++ {
				v = v*16 + pycdHexVal(bs[i])
				i++
			}
			if k == count && v > 0x10ffff {
				why, k = "illegal Unicode character", 0
				if raw {
					why = "\\Uxxxxxxxx out of range"
				}
			}
			if k < count {
				if i = bad(name, start, i, why); exc != "" {
					return nil, exc, msg
				}
				continue
			}
			push(v)
		}
		return out, "", ""
	case codec == "utf-8":
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

// pycdEncodeEscape is unicode_escape (raw: raw_unicode_escape) encoding.
func pycdEncodeEscape(cps []int, raw bool) []int {
	var out []int
	for _, c := range cps {
		esc := ""
		switch {
		case c >= 0x10000:
			esc = fmt.Sprintf("\\U%08x", c)
		case c >= 0x100:
			esc = fmt.Sprintf("\\u%04x", c)
		case raw:
			out = append(out, c)
		case c == '\\':
			esc = "\\\\"
		case c == '\t':
			esc = "\\t"
		case c == '\n':
			esc = "\\n"
		case c == '\r':
			esc = "\\r"
		case c < 0x20 || c >= 0x7f:
			esc = fmt.Sprintf("\\x%02x", c)
		default:
			out = append(out, c)
		}
		for i := 0; i < len(esc); i++ {
			out = append(out, int(esc[i]))
		}
	}
	return out
}

func pycdHexVal(c int) int {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return -1
}

func pycdPunyT(j, bias int) int {
	t := 36*(j+1) - bias
	if t < 1 {
		return 1
	}
	if t > 26 {
		return 26
	}
	return t
}

// pycdPunyAdapt is the bias after delta, numchars code points in.
func pycdPunyAdapt(delta *big.Int, first bool, numchars int) int {
	d := new(big.Int).Set(delta)
	if first {
		d.Quo(d, big.NewInt(700))
	} else {
		d.Quo(d, big.NewInt(2))
	}
	d.Add(d, new(big.Int).Quo(d, big.NewInt(int64(numchars))))
	divisions := 0
	for d.Cmp(big.NewInt(455)) > 0 {
		d.Quo(d, big.NewInt(35))
		divisions += 36
	}
	v := int(d.Int64())
	return divisions + 36*v/(v+38)
}

// pycdEncodePuny is punycode encoding.
func pycdEncodePuny(cps []int) []int {
	var out, ext []int
	for _, c := range cps {
		at := len(ext)
		for at > 0 && ext[at-1] > c {
			at--
		}
		if c < 0x80 {
			out = append(out, c)
		} else if at == 0 || ext[at-1] != c {
			ext = append(ext[:at], append([]int{c}, ext[at:]...)...)
		}
	}
	baselen := len(out)
	if baselen > 0 {
		out = append(out, '-')
	}
	const digits = "abcdefghijklmnopqrstuvwxyz0123456789"
	oldchar, oldindex, bias, points := 0x80, -1, 72, 0
	for _, ch := range ext {
		below := 0
		for _, c := range cps {
			if c < ch {
				below++
			}
		}
		delta, index := (below+1)*(ch-oldchar), -1
		for _, c := range cps {
			if c < ch {
				index++
			}
			if c != ch {
				continue
			}
			index++
			delta += index - oldindex
			n, j := delta-1, 0
			for t := pycdPunyT(0, bias); n >= t; t = pycdPunyT(j, bias) {
				out = append(out, int(digits[t+(n-t)%(36-t)]))
				n = (n - t) / (36 - t)
				j++
			}
			out = append(out, int(digits[n]))
			bias = pycdPunyAdapt(big.NewInt(int64(delta-1)), points == 0, baselen+points+1)
			points++
			oldindex, delta = index, 0
		}
		oldchar = ch
	}
	return out
}

// pycdDecodePuny is punycode decoding, over math/big as a malformed string's
// deltas are unbounded.
func pycdDecodePuny(bs []int, errs string) (out []uint16, exc, msg string) {
	if errs != "strict" && errs != "replace" && errs != "ignore" {
		return nil, "UnicodeError", "Unsupported error handling: " + errs
	}
	return pycdPunyRun(bs, 0, len(bs), errs, "punycode")
}

// pycdPunyRun decodes the punycode bs[lo:hi], its errors naming codec and
// counting positions in the whole of bs.
func pycdPunyRun(bs []int, lo, hi int, errs, codec string) (out []uint16, exc, msg string) {
	dash := -1
	for i := lo; i < hi; i++ {
		if bs[i] == '-' {
			dash = i
		}
	}
	var base []int
	for i := lo; i < dash; i++ {
		switch {
		case bs[i] < 0x80:
			base = append(base, bs[i])
		case errs == "strict":
			name := codec
			if codec == "punycode" {
				name = "ascii"
			}
			return nil, "UnicodeDecodeError", fmt.Sprintf("'%s' codec can't decode byte 0x%02x in position %d: ordinal not in range(128)", name, bs[i], i)
		case errs == "replace":
			base = append(base, 0xfffd)
		}
	}
	xs := lo
	if dash >= 0 {
		xs = dash + 1
	}
	var ext []int
	for _, c := range bs[xs:hi] {
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		ext = append(ext, c)
	}
	fail := func(at *big.Int, why string) ([]uint16, string, string) {
		if !new(big.Int).Add(at, big.NewInt(1)).IsInt64() {
			return nil, "OverflowError", "Python int too large to convert to C ssize_t"
		}
		if at.Cmp(big.NewInt(int64(len(bs)))) < 0 {
			i := int(at.Int64())
			return nil, "UnicodeDecodeError", fmt.Sprintf("'%s' codec can't decode byte 0x%02x in position %d: %s", codec, bs[i], i, why)
		}
		return nil, "UnicodeDecodeError", "'" + codec + "' codec can't decode bytes in position " + at.String() + "-" + at.String() + ": " + why
	}
	ch, pos := big.NewInt(0x80), big.NewInt(0)
	bias, extpos, first := 72, 0, true
	for extpos < len(ext) {
		delta, w := big.NewInt(0), big.NewInt(1)
		why, from := "", extpos
		for j := 0; why == ""; j++ {
			if extpos >= len(ext) {
				why, from = "incomplete punycode string", extpos
				break
			}
			c := ext[extpos]
			extpos++
			digit := -1
			switch {
			case c >= 'A' && c <= 'Z':
				digit = c - 'A'
			case c >= '0' && c <= '9':
				digit = c - 22
			}
			if digit < 0 {
				why, from = fmt.Sprintf("Invalid extended code point '%d'", c), extpos-1
				break
			}
			t := pycdPunyT(j, bias)
			delta.Add(delta, new(big.Int).Mul(w, big.NewInt(int64(digit))))
			if digit < t {
				break
			}
			w.Mul(w, big.NewInt(int64(36-t)))
		}
		if why != "" {
			if errs == "strict" {
				return fail(big.NewInt(int64(xs+from)), why)
			}
			break
		}
		pos.Add(pos, delta)
		if !first {
			pos.Add(pos, big.NewInt(1))
		}
		q, r := new(big.Int).QuoRem(pos, big.NewInt(int64(len(base)+1)), new(big.Int))
		ch.Add(ch, q)
		if ch.Cmp(big.NewInt(0x10ffff)) > 0 {
			if errs == "strict" {
				return fail(new(big.Int).Add(pos, big.NewInt(int64(xs-1))), fmt.Sprintf("Invalid character U+%x", ch))
			}
			ch.SetInt64('?')
		}
		at := int(r.Int64())
		base = append(base[:at], append([]int{int(ch.Int64())}, base[at:]...)...)
		bias = pycdPunyAdapt(delta, first, len(base))
		pos.SetInt64(int64(at))
		first = false
	}
	for _, c := range base {
		if c >= 0x10000 {
			out = append(out, uint16(0xd800+(c-0x10000)>>10), uint16(0xdc00+(c-0x10000)&0x3ff))
		} else {
			out = append(out, uint16(c))
		}
	}
	return out, "", ""
}

func pycdIdnaDot(c int) bool {
	return c == '.' || c == 0x3002 || c == 0xff0e || c == 0xff61
}

// pycdIdnaSplit is the labels of a name as [from, to) pairs: split at "."
// only (ascii), or at any of the four IDNA dots.
func pycdIdnaSplit(cps []int, ascii bool) []int {
	out := []int{0}
	for i, c := range cps {
		if c == '.' || (!ascii && pycdIdnaDot(c)) {
			out = append(out, i, i+1)
		}
	}
	return append(out, len(cps))
}

func pycdIdnaEncErr(cps []int, from, to int, why string) ([]int, string, string) {
	where := fmt.Sprintf("characters in position %d-%d", from, to-1)
	if to-from == 1 && from < len(cps) {
		where = fmt.Sprintf("character '%s' in position %d", pysEncEsc(cps[from]), from)
	}
	return nil, "UnicodeEncodeError", "'idna' codec can't encode " + where + ": " + why
}

func pycdIdnaDecErr(bs []int, from, to int, why string) ([]uint16, string, string) {
	where := fmt.Sprintf("bytes in position %d-%d", from, to-1)
	if to-from == 1 && from < len(bs) {
		where = fmt.Sprintf("byte 0x%02x in position %d", bs[from], from)
	}
	return nil, "UnicodeDecodeError", "'idna' codec can't decode " + where + ": " + why
}

// pycdIdnaRefused is the error of a non-ASCII label in a program that did not
// register the nameprep tables (python-to-llvm-ir.abnf links them only into a
// program that names 'idna' in a string literal).
const pycdIdnaRefused = "the idna codec's nameprep tables are linked only into a program that names 'idna' in a string literal"

// pycdEncodeIdna is idna encoding. Twin of pycdEncodeIdna in
// languages/lib/python-codecs.js, which states the rules.
func pycdEncodeIdna(cps []int, errs string) ([]int, string, string) {
	if errs != "strict" {
		return nil, "UnicodeError", "Unsupported error handling: " + errs
	}
	ascii := true
	for _, c := range cps {
		if c >= 0x80 {
			ascii = false
		}
	}
	labels := pycdIdnaSplit(cps, ascii)
	last := len(labels) - 2
	if ascii {
		for e := 0; e < last; e += 2 {
			if labels[e+1] == labels[e] {
				return pycdIdnaEncErr(cps, labels[e], labels[e]+1, "label empty")
			}
		}
		for k := 0; k <= last; k += 2 {
			if labels[k+1]-labels[k] >= 64 {
				return pycdIdnaEncErr(cps, labels[k], labels[k+1], "label too long")
			}
		}
		return append([]int(nil), cps...), "", ""
	}
	if labels[last+1] == labels[last] && len(labels) > 2 {
		last -= 2
	}
	var out []int
	for q := 0; q <= last; q += 2 {
		if q > 0 {
			out = append(out, '.')
		}
		t := pycdIdnaToAscii(cps[labels[q]:labels[q+1]])
		if t.refused {
			return nil, "NotImplementedError", pycdIdnaRefused
		}
		if t.why != "" {
			return pycdIdnaEncErr(cps, labels[q]+t.lo, labels[q]+t.hi, t.why)
		}
		out = append(out, t.b...)
	}
	if last < len(labels)-2 {
		out = append(out, '.')
	}
	return out, "", ""
}

// pycdIdnaLabel is a label's ToASCII bytes, or the error's reason and
// [lo, hi) within the label, or refused when nameprep has no tables.
type pycdIdnaLabel struct {
	b       []int
	why     string
	lo, hi  int
	refused bool
}

// pycdIdnaToAscii is ToASCII of one label. Twin of pycdIdnaToAscii in
// languages/lib/python-codecs.js.
func pycdIdnaToAscii(lab []int) pycdIdnaLabel {
	cps := lab
	if !pycdAllAscii(cps) {
		np := pycdNameprep(cps)
		if np.why != "" {
			return np
		}
		cps = np.b
	}
	if pycdAllAscii(cps) {
		switch {
		case len(cps) == 0:
			return pycdIdnaLabel{why: "label empty", lo: 0, hi: 1}
		case len(cps) >= 64:
			return pycdIdnaLabel{why: "label too long", lo: 0, hi: len(cps)}
		}
		return pycdIdnaLabel{b: cps}
	}
	if len(cps) >= 4 && pycdIdnaAce(pycdLowerAscii(cps), 0) {
		return pycdIdnaLabel{why: "Label starts with ACE prefix", lo: 0, hi: 4}
	}
	b := append([]int{'x', 'n', '-', '-'}, pycdEncodePuny(cps)...)
	if len(b) >= 64 {
		return pycdIdnaLabel{why: "label too long", lo: 0, hi: len(cps)}
	}
	return pycdIdnaLabel{b: b}
}

func pycdAllAscii(cps []int) bool {
	for _, c := range cps {
		if c >= 0x80 {
			return false
		}
	}
	return true
}

func pycdLowerAscii(cps []int) []int {
	out := make([]int, len(cps))
	for i, c := range cps {
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		out[i] = c
	}
	return out
}

// pycdPrepTables is the nameprep data of languages/lib/python-nameprep.txt,
// whose format python-codecs.js states; a compiled program registers it at
// boot through js_pynameprepdata, with lib/unicode-norm.txt for NFKC.
type pycdPrepTables struct {
	runs map[byte][]int // kind letter -> lo, hi, lo, hi, ... ascending
	m, n map[int][]int
}

var pycdPrep *pycdPrepTables

func pycdPrepData(text string) {
	if pycdPrep != nil {
		return
	}
	t := &pycdPrepTables{runs: map[byte][]int{}, m: map[int][]int{}, n: map[int][]int{}}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		var ns []int
		for _, f := range strings.Fields(line[1:]) {
			ns = append(ns, jsvNormHex(f))
		}
		switch line[0] {
		case 'm':
			t.m[ns[0]] = ns[1:]
		case 'n':
			t.n[ns[0]] = ns[1:]
		default:
			t.runs[line[0]] = append(t.runs[line[0]], ns[0], ns[1])
		}
	}
	pycdPrep = t
}

func (t *pycdPrepTables) in(kind byte, cp int) bool {
	runs := t.runs[kind]
	k := sort.Search(len(runs)/2, func(i int) bool { return runs[2*i+1] >= cp })
	return k < len(runs)/2 && runs[2*k] <= cp
}

// pycdNameprep is nameprep (RFC 3491) of a label. Twin of pycdNameprep in
// languages/lib/python-codecs.js, which states how Unicode 3.2's NFKC comes
// out of today's tables.
func pycdNameprep(cps []int) pycdIdnaLabel {
	t := pycdPrep
	if t == nil || jsvNorm == nil {
		return pycdIdnaLabel{why: "nameprep", refused: true}
	}
	var mapped []int
	for _, c := range cps {
		if t.in('x', c) {
			continue
		}
		if mv, ok := t.m[c]; ok {
			mapped = append(mapped, mv...)
		} else {
			mapped = append(mapped, c)
		}
	}
	var out, piece []int
	for k := 0; k <= len(mapped); k++ {
		d := -1
		if k < len(mapped) {
			d = mapped[k]
		}
		nv, isN := t.n[d]
		if d >= 0 && !isN && !t.in('u', d) && (d < 0xD800 || d > 0xDFFF) {
			piece = append(piece, d)
			continue
		}
		out = append(out, pycdStrCps(jsvNormString(pycdCpsStr(piece), "NFKC"))...)
		piece = piece[:0]
		if isN {
			out = append(out, nv...)
		} else if d >= 0 {
			out = append(out, d)
		}
	}
	for p, c := range out {
		if t.in('p', c) {
			return pycdIdnaLabel{why: "Invalid character " + pycdCpRepr(c), lo: p, hi: p + 1}
		}
	}
	ral := false
	for _, c := range out {
		ral = ral || t.in('r', c)
	}
	if ral {
		for l, c := range out {
			if t.in('l', c) {
				return pycdIdnaLabel{why: "Violation of BIDI requirement 2", lo: l, hi: l + 1}
			}
		}
		if !t.in('r', out[0]) {
			return pycdIdnaLabel{why: "Violation of BIDI requirement 3", lo: 0, hi: 1}
		}
		if !t.in('r', out[len(out)-1]) {
			return pycdIdnaLabel{why: "Violation of BIDI requirement 3", lo: len(out) - 1, hi: len(out)}
		}
	}
	return pycdIdnaLabel{b: out}
}

func pycdCpsStr(cps []int) string {
	u := make([]uint16, 0, len(cps))
	for _, v := range cps {
		if v >= 0x10000 {
			u = append(u, uint16(0xD800+(v-0x10000)/1024), uint16(0xDC00+(v-0x10000)%1024))
		} else {
			u = append(u, uint16(v))
		}
	}
	return strFromUnits(u)
}

func pycdStrCps(s string) []int {
	u := strUnits(s)
	var out []int
	for i := 0; i < len(u); i++ {
		c := int(u[i])
		if c >= 0xD800 && c <= 0xDBFF && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] <= 0xDFFF {
			out = append(out, 0x10000+(c-0xD800)*1024+int(u[i+1])-0xDC00)
			i++
			continue
		}
		out = append(out, c)
	}
	return out
}

// pycdCpRepr is repr() of a one-character str.
func pycdCpRepr(cp int) string {
	switch cp {
	case '\'':
		return `"'"`
	case '\\':
		return `'\\'`
	case '\t':
		return `'\t'`
	case '\n':
		return `'\n'`
	case '\r':
		return `'\r'`
	}
	if pyIsPrintable(rune(cp)) {
		return "'" + string(rune(cp)) + "'"
	}
	return "'" + pysEncEsc(cp) + "'"
}

func pycdIdnaAce(bs []int, i int) bool {
	return i+4 <= len(bs) && (bs[i] == 'x' || bs[i] == 'X') && (bs[i+1] == 'n' || bs[i+1] == 'N') && bs[i+2] == '-' && bs[i+3] == '-'
}

// pycdDecodeIdna is idna decoding. Twin of pycdDecodeIdna in
// languages/lib/python-codecs.js.
func pycdDecodeIdna(bs []int, errs string) (out []uint16, exc, msg string) {
	if errs != "strict" {
		return nil, "UnicodeError", "Unsupported error handling: " + errs
	}
	ace, ascii := false, true
	for i, c := range bs {
		ace = ace || pycdIdnaAce(bs, i)
		ascii = ascii && c < 0x80
	}
	if !ace && ascii {
		for _, c := range bs {
			out = append(out, uint16(c))
		}
		return out, "", ""
	}
	labels := pycdIdnaSplit(bs, true)
	last := len(labels) - 2
	dot := labels[last+1] == labels[last] && len(labels) > 2
	if dot {
		last -= 2
	}
	for k := 0; k <= last; k += 2 {
		from, to := labels[k], labels[k+1]
		if k > 0 {
			out = append(out, '.')
		}
		if to-from > 1024 {
			return pycdIdnaDecErr(bs, from, to, "label way too long")
		}
		if to-from < 4 || !pycdIdnaAce(bs, from) {
			for j := from; j < to; j++ {
				if bs[j] >= 0x80 {
					return pycdIdnaDecErr(bs, j, j+1, "ordinal not in range(128)")
				}
				out = append(out, uint16(bs[j]))
			}
			continue
		}
		res, exc, msg := pycdPunyRun(bs, from+4, to, "strict", "idna")
		if exc != "" {
			return nil, exc, msg
		}
		t := pycdIdnaToAscii(pycdStrCps(strFromUnits(res)))
		if t.refused {
			return nil, "NotImplementedError", pycdIdnaRefused
		}
		if t.why != "" {
			return pycdIdnaDecErr(bs, from+t.lo, from+t.hi, t.why)
		}
		lab := bs[from:to]
		low := pycdLowerAscii(lab)
		same := len(low) == len(t.b)
		for v := 0; same && v < len(low); v++ {
			same = low[v] == t.b[v]
		}
		if !same {
			la, ba := &jsArray{}, &jsArray{}
			for _, c := range lab {
				la.elems = append(la.elems, float64(c))
			}
			for _, c := range t.b {
				ba.elems = append(ba.elems, float64(c))
			}
			return pycdIdnaDecErr(bs, from, to, "IDNA does not round-trip, '"+pyBytesRepr(la)+"' != '"+pyBytesRepr(ba)+"'")
		}
		out = append(out, res...)
	}
	if dot {
		out = append(out, '.')
	}
	return out, "", ""
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		m["js_pynameprepdata"] = func(a []uint64) uint64 {
			jsvNormData(rt.toString(rt.unwrap(a[0])))
			pycdPrepData(rt.toString(rt.unwrap(a[1])))
			return 0
		}
	})
}
