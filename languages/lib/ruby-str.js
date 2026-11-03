// ruby-str.js -- String algorithms of MRI's string.c that take a String (and a
// compiled regex.js pattern) and answer plain values, for ruby-interpreter.abnf
// (include) and languages/lib/ruby-rt.metajs (import). Positions are UTF-16 code
// units; a surrogate pair is kept whole. Go twins in abnf/jsrtruby.go.
//
// Written in the subset both engines accept: no split/join/lastIndexOf, and
// indexOf without a from-index.

function rbsIsSpace(c) { return c == 32 || (c >= 9 && c <= 13) }

// The length of the character at unit i: 2 for a surrogate pair, else 1.
function rbsCharLen(s, i) {
    var c = s.charCodeAt(i)
    if (c < 55296 || c > 56319 || i + 1 >= s.length) { return 1 }
    var d = s.charCodeAt(i + 1)
    return d >= 56320 && d <= 57343 ? 2 : 1
}

// The first unit index >= from where t occurs in s, or -1.
function rbsFind(s, t, from) {
    if (from > s.length) { return -1 }
    var k = s.substring(from, s.length).indexOf(t)
    return k < 0 ? -1 : from + k
}

// String#split (rb_str_split_m). sep is the separator String, or null for awk
// mode (no argument, nil or " "): runs of ASCII whitespace separate, leading
// whitespace is skipped. re, when not null, is a compiled pattern and wins over
// sep; its groups are spliced into the answer. lim is the limit (0 when absent):
// positive caps the field count, the last field holding the rest; zero drops
// trailing empty fields; negative keeps them. last[0] receives the result of
// the final pattern search (null when it failed), which MRI leaves in $~; it is
// untouched when no search ran.
function rbsSplit(s, sep, re, lim, last) {
    var out = []
    if (lim == 1) {
        if (s.length > 0) { out.push(s) }
        return out
    }
    var i = 1
    var beg = 0
    var p = 0
    if (re !== null) {
        var start = 0
        var lastNull = false
        while (true) {
            var m = rxSearch(re, s, start)
            last[0] = m
            if (m === null) { break }
            if (start == m.begin && m.begin == m.end) {
                if (s.length == 0) {
                    out.push("")
                    break
                } else if (lastNull) {
                    out.push(s.substring(beg, beg + rbsCharLen(s, beg)))
                    beg = start
                } else {
                    start = start + (start < s.length ? rbsCharLen(s, start) : 1)
                    lastNull = true
                    continue
                }
            } else {
                out.push(s.substring(beg, m.begin))
                beg = m.end
                start = m.end
            }
            lastNull = false
            var caps = m.caps
            var g = 1
            while (g <= re.ngroups) {
                if (caps[2 * g] >= 0 && caps[2 * g + 1] >= 0) { out.push(s.substring(caps[2 * g], caps[2 * g + 1])) }
                g = g + 1
            }
            i = i + 1
            if (lim > 0 && lim <= i) { break }
        }
    } else if (sep === null) {
        var skip = true
        var end = 0
        while (p < s.length) {
            var c = s.charCodeAt(p)
            p = p + 1
            if (skip) {
                if (rbsIsSpace(c)) {
                    beg = p
                } else {
                    end = p
                    skip = false
                    if (lim > 0 && lim <= i) { break }
                }
            } else if (rbsIsSpace(c)) {
                out.push(s.substring(beg, end))
                skip = true
                beg = p
                if (lim > 0) { i = i + 1 }
            } else {
                end = p
            }
        }
    } else if (sep.length == 0) {
        while (p < s.length) {
            var n = rbsCharLen(s, p)
            out.push(s.substring(p, p + n))
            p = p + n
            i = i + 1
            if (lim > 0 && lim <= i) { break }
        }
        beg = p
    } else {
        while (p < s.length) {
            var at = rbsFind(s, sep, p)
            if (at < 0) { break }
            out.push(s.substring(p, at))
            p = at + sep.length
            i = i + 1
            if (lim > 0 && lim <= i) { break }
        }
        beg = p
    }
    if (s.length > 0 && (lim > 0 || s.length > beg || lim < 0)) { out.push(s.substring(beg, s.length)) }
    if (lim == 0) {
        while (out.length > 0 && out[out.length - 1] == "") { out.pop() }
    }
    return out
}

// String#lines / #each_line: the pieces of s, each ending after an occurrence
// of sep, the rest last. chomp (the `chomp: true` keyword) drops the separator,
// and for "\n" a "\r" in front of it.
function rbsLines(s, sep, chomp) {
    var out = []
    var p = 0
    while (p < s.length && sep.length > 0) {
        var at = rbsFind(s, sep, p)
        if (at < 0) { break }
        var e = at + sep.length
        if (chomp) { e = sep == "\n" && at > p && s.charCodeAt(at - 1) == 13 ? at - 1 : at }
        out.push(s.substring(p, e))
        p = at + sep.length
    }
    if (p < s.length) { out.push(s.substring(p, s.length)) }
    return out
}

// Integer#chr("UTF-8"): the character of code point cp, or "" with the
// RangeError's message in err[0]. Go twin: rubyChrUtf8 in abnf/jsrtrubystr.go.
function rbsChrUtf8(cp, err) {
    if (cp < 0 || cp > 1114111) {
        err.push(cp + " out of char range")
        return ""
    }
    if (cp >= 55296 && cp <= 57343) {
        var hx = ""
        var v = cp
        while (v > 0) {
            hx = "0123456789ABCDEF".charAt(v % 16) + hx
            v = Math.floor(v / 16)
        }
        err.push("invalid codepoint 0x" + hx + " in UTF-8")
        return ""
    }
    if (cp < 65536) { return String.fromCharCode(cp) }
    var w = cp - 65536
    return String.fromCharCode(55296 + Math.floor(w / 1024), 56320 + (w % 1024))
}

// String#ord: the code point of the first character, a surrogate pair whole.
function rbsOrd(s) {
    if (s.length == 0) { return 0 }
    var c = s.charCodeAt(0)
    if (rbsCharLen(s, 0) == 2) { return (c - 55296) * 1024 + (s.charCodeAt(1) - 56320) + 65536 }
    return c
}

// The character index of code unit u, and the unit index of character c (-1 when
// the string is shorter).
function rbsCharIdx(s, u) {
    var i = 0
    var n = 0
    while (i < u) {
        i = i + rbsCharLen(s, i)
        n = n + 1
    }
    return n
}
function rbsUnitIdx(s, c) {
    var i = 0
    var n = 0
    while (n < c && i < s.length) {
        i = i + rbsCharLen(s, i)
        n = n + 1
    }
    return n < c ? -1 : i
}

// The character count of s. byteLen equal to the unit count means ASCII, where
// a unit is a character, so the common case does not scan.
function rbsLen(s) {
    if (byteLen(s) == s.length) { return s.length }
    return rbsCharIdx(s, s.length)
}

// Characters [c, c + k) of s, cut at the end (0 <= c <= rbsLen(s), k >= 0).
function rbsSub(s, c, k) {
    var a = c
    var b = c + k
    if (byteLen(s) != s.length) {
        a = rbsUnitIdx(s, c)
        b = a
        var n = 0
        while (n < k && b < s.length) {
            b = b + rbsCharLen(s, b)
            n = n + 1
        }
    }
    if (b > s.length) { b = s.length }
    return s.substring(a, b)
}

// String#[start, len] (start and len integers): nil for a negative len or a start
// outside the string, a negative start counting from the end.
function rbsSubseq(s, start, len) {
    var n = rbsLen(s)
    if (len < 0) { return null }
    if (start < 0) { start = start + n }
    if (start < 0 || start > n) { return null }
    return rbsSub(s, start, len)
}

// String#[i] (an integer): one character, or nil outside the string.
function rbsCharAt(s, i) {
    var n = rbsLen(s)
    if (i < 0) { i = i + n }
    if (i < 0 || i >= n) { return null }
    return rbsSub(s, i, 1)
}

// String#chars: one element per CHARACTER, a surrogate pair taken whole as one
// substring (rebuilt from its two halves it would be two U+FFFD).
function rbsChars(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var k = rbsCharLen(s, i)
        out.push(s.substring(i, i + k))
        i = i + k
    }
    return out
}

// ----- extended grapheme clusters (UAX #29, Unicode 12.1 as MRI 2.6 has it) -----
// rbsGcbTable holds each code point's Grapheme_Cluster_Break class, read from
// /usr/bin/ruby 2.6's own \p{Grapheme_Cluster_Break=...} and
// \p{Extended_Pictographic} over every code point: one record per run of code
// points of one class, five base-64 digits each (a digit is its value plus 48,
// plus 49 from 44 up, so no backslash): the run's first code point in four, its
// class in one. The classes:
//     0 Other  1 CR  2 LF  3 Control  4 Extend  5 ZWJ  6 Regional_Indicator
//     7 Prepend  8 SpacingMark  9 L  10 V  11 T  12 LV  13 LVT
//     14 Extended_Pictographic
// Go twin: rubyGcbTable in abnf/jsrtrubygcb.go, kept equal by
// abnf/ruby_unicode_test.go.
var rbsGcbTable = ""
function rbsGcbTab() {
    if (rbsGcbTable == "") {
        rbsGcbTable = ""
        + "00003000:2000;3000=1000>3000P0001p3002P0002Y>002Z0002^3002_>002`000<0400=a000B3400B:000FA400Fo000Fp4"
        + "00G0000G1400G3000G4400G6000G7400G8000H0700H6000H@400HK000HL300HM000I;400IP000Ia400Ib000KF400KM700KN0"
        + "00KO400KU000KW400KY000KZ400K_000L?700L@000LA400LB000La400M;000NV400Nb000O[400Oe000On400Oo000PF400PJ0"
        + "00PK400PT000PU400PX000PY400P_000QI400QL000SC400SR700SS400T3800T4000Tk400Tl800Tm400Tn000To800U1400U98"
        + "00U=400U>800U@000UA400UH000UR400UT000V1400V2800V4000Vm400Vn000Vo400Vp800W1400W5000W7800W9000W;800W=4"
        + "00W>000WG400WH000WR400WT000Wo400Wp000X1400X3800X4000Xm400Xn000Xo800Y1400Y3000Y7400Y9000Y;400Y>000YA4"
        + "00YB000Ya400Yc000Yf400Yg000Z1400Z3800Z4000Zm400Zn000Zo800[1400[6000[7400[9800[:000[;800[=400[>000[R4"
        + "00[T000[k400]0000]1400]2800]4000]m400]n000]o400^0800^1400^5000^7800^9000^;800^=400^>000^F400^H000^R4"
        + "00^T000_2400_3000_o400_p800`0400`1800`3000`6800`9000`:800`=400`>000`G400`H000a0400a1800a4400a5000ao4"
        + "00b1800b5000b6400b9000b:400b>000bE400bG000bR400bT000c1400c2800c4000cm400cn000co800cp400d0800d2400d38"
        + "00d5000d6400d7800d9000d:800d<400d>000dE400dG000dR400dT000e0400e2800e4000el400en000eo400ep800f1400f50"
        + "00f6800f9000f:800f=400f>700f?000fG400fH000fR400fT000g2800g4000h:400h;000h?400h@800hB400hE000hF400hG0"
        + "00hH800hO400hP000hc800he000ib400ic000id800ie400il000j7400j?000kb400kc000kd800ke400kn000l8400l>000mH4"
        + "00mJ000mf400mg000mh400mi000mj400mk000mo800n0000nb400np800o0400o5000o6400o8000o=400oH000oI400on000p64"
        + "00p70010^4010b8010c4010i0010j4010l8010n4010p0011F8011H4011J0011N4011Q0011b4011f001224012300124801254"
        + "01270012=4012>0012M4012N001409015P:016X;0180001=M401=P001LB401LE001Lc401Lf001MB401MD001Mc401Me001Ne4"
        + "01Ng801Nh401No801O6401O7801O9401OD001OM401ON001P;401P>301P?001R5401R7001RY401RZ001TP401TS801TW401TY8"
        + "01T]001Ta801Tc401Td801Tj401Tm001XG401XI801XK401XL001YE801YF401YG801YH401YO001YP401YQ001YR401YS001YU4"
        + "01Y^801Yd401Yn001Yp401Z0001Za401Zp001]0401]4801]5001]e401]l801]m401]n801^2401^3801^5001^[401^e001_04"
        + "01_2801_3001_Q801_R401_V801_X401_Z801_[401__001`V401`W801`X401`Z801`^401`_801``401`c801`e001aT801a]4"
        + "01ae801ag401ai001d@401dC001dD401dQ801dR401dY001d^401d_001de401df001dh801di401dk001h0401hk001hl401i00"
        + "020;3020<4020=5020>3020@0020X3020`0020m>020n00219>021:0021P3021a0023@4023b0024R>024S0024j>024k0026D>"
        + "026J0026Y>026[002<J>02<L002<X>02<Y002>8>02>9002??>02?@002?Y>02?e002?i>02?l002C2>02C3002FZ>02F]002Fg>"
        + "02Fh002G0>02G1002Gl>02Gp002H0>02H6002H7>02HC002HD>02J6002J@>02L6002L8>02LC002LD>02LE002LF>02LG002LM>"
        + "02LN002LQ>02LR002LX>02LY002Ld>02Lf002M4>02M5002M7>02M8002M<>02M=002M>>02M?002MC>02MF002MG>02MH002MS>"
        + "02MX002NE>02NH002NQ>02NR002Na>02Nb002Np>02O0002Te>02Tg002]5>02]8002]K>02]M002^@>02^A002^E>02^F002d`4"
        + "02dc002fp402g0002hP402i00030Z4030a>030b0030n>030o0032I4032K003:G>03:H003:I>03:J00:I`40:Id00:Ie40:Io0"
        + "0:JN40:JP00:Ka40:Kc00:P240:P300:P640:P700:P;40:P<00:PS80:PU40:PW80:PX00:R080:R200:Re80:S440:S600:SP4"
        + "0:Sc00:Sp40:T000:TV40:T_00:U740:UB80:UD00:UP90:Un00:V040:V380:V400:Vd40:Ve80:Vg40:Vk80:Vm40:Vo80:W10"
        + "0:WU40:WV00:XY40:X`80:Xb40:Xd80:Xf40:Xh00:Y340:Y400:Y<40:Y=80:Y>00:Ym40:Yn00:Za40:Zb00:Zc40:Zf00:Zh4"
        + "0:Zj00:Zo40:[000:[140:[200:[[80:[]40:[_80:[a00:[f80:[g40:[h00:`S80:`U40:`V80:`X40:`Y80:`[00:`]80:`^4"
        + "0:`_00:a0<0:a1=0:aL<0:aM=0:ai<0:aj=0:bD<0:bE=0:ba<0:bb=0:c<<0:c==0:cX<0:cY=0:d4<0:d5=0:dP<0:dQ=0:dm<"
        + "0:dn=0:eH<0:eI=0:ee<0:ef=0:f@<0:fA=0:f]<0:f^=0:g8<0:g9=0:gT<0:gU=0:h0<0:h1=0:hL<0:hM=0:hi<0:hj=0:iD<"
        + "0:iE=0:ia<0:ib=0:j<<0:j==0:jX<0:jY=0:k4<0:k5=0:kP<0:kQ=0:km<0:kn=0:lH<0:lI=0:le<0:lf=0:m@<0:mA=0:m]<"
        + "0:m^=0:n8<0:n9=0:nT<0:nU=0:o0<0:o1=0:oL<0:oM=0:oi<0:oj=0:pD<0:pE=0:pa<0:pb=0;0<<0;0==0;0X<0;0Y=0;14<"
        + "0;15=0;1P<0;1Q=0;1m<0;1n=0;2H<0;2I=0;2e<0;2f=0;3@<0;3A=0;3]<0;3^=0;48<0;49=0;4T<0;4U=0;50<0;51=0;5L<"
        + "0;5M=0;5i<0;5j=0;6D<0;6E=0;6a<0;6b=0;7<<0;7==0;7X<0;7Y=0;84<0;85=0;8P<0;8Q=0;8m<0;8n=0;9H<0;9I=0;9e<"
        + "0;9f=0;:@<0;:A=0;:]<0;:^=0;;8<0;;9=0;;T<0;;U=0;<0<0;<1=0;<L<0;<M=0;<i<0;<j=0;=D<0;=E=0;=a<0;=b=0;><<"
        + "0;>==0;>X<0;>Y=0;?4<0;?5=0;?P<0;?Q=0;?m<0;?n=0;@H<0;@I=0;@e<0;@f=0;A@<0;AA=0;A]<0;A^=0;B8<0;B9=0;BT<"
        + "0;BU=0;C0<0;C1=0;CL<0;CM=0;Ci<0;Cj=0;DD<0;DE=0;Da<0;Db=0;E<<0;E==0;EX<0;EY=0;F4<0;F5=0;FP<0;FQ=0;Fm<"
        + "0;Fn=0;GH<0;GI=0;Ge<0;Gf=0;H@<0;HA=0;H]<0;H^=0;I8<0;I9=0;IT<0;IU=0;J0<0;J1=0;JL<0;JM=0;Ji<0;Jj=0;KD<"
        + "0;KE=0;Ka<0;Kb=0;L<<0;L==0;LX<0;LY=0;M4<0;M5=0;MP<0;MQ=0;Mm<0;Mn=0;NH<0;NI=0;Ne<0;Nf=0;O@<0;OA=0;O]<"
        + "0;O^=0;P8<0;P9=0;PT<0;PU=0;Q0<0;Q1=0;QL<0;QM=0;Qi<0;Qj=0;RD<0;RE=0;Ra<0;Rb=0;S<<0;S==0;SX<0;SY=0;T4<"
        + "0;T5=0;TP<0;TQ=0;Tm<0;Tn=0;UH<0;UI=0;Ue<0;Uf=0;V@<0;VA=0;V]<0;V^=0;W8<0;W9=0;WT<0;WU=0;X0<0;X1=0;XL<"
        + "0;XM=0;Xi<0;Xj=0;YD<0;YE=0;Ya<0;Yb=0;Z<<0;Z==0;ZX<0;ZY=0;[4<0;[5=0;[P<0;[Q=0;[m<0;[n=0;]H<0;]I=0;]e<"
        + "0;]f=0;^@<0;^A=0;^]<0;^^=0;_8<0;_9=0;_T<0;_U=0;`0<0;`1=0;`L<0;`M=0;`i<0;`j=0;aD<0;aE=0;aa<0;ab=0;b<<"
        + "0;b==0;bX<0;bY=0;c4<0;c5=0;cP<0;cQ=0;cm<0;cn=0;dH<0;dI=0;de<0;df=0;e@<0;eA=0;e]<0;e^=0;f8<0;f9=0;fT<"
        + "0;fU=0;g0<0;g1=0;gL<0;gM=0;gi<0;gj=0;hD<0;hE=0;ha<0;hb=0;i<<0;i==0;iX<0;iY=0;j4<0;j5=0;jP<0;jQ=0;jm<"
        + "0;jn=0;kH<0;kI=0;ke<0;kf=0;l@<0;lA=0;l]<0;l^=0;m8<0;m9=0;mT<0;mU=0;n0<0;n1=0;nL<0;nM=0;ni<0;nj=0;oD<"
        + "0;oE=0;oa<0;ob=0;p<<0;p==0;pX<0;pY=0<04<0<05=0<0P<0<0Q=0<0m<0<0n=0<1H<0<1I=0<1e<0<1f=0<2@<0<2A=0<2]<"
        + "0<2^=0<38<0<39=0<3T<0<3U=0<40<0<41=0<4L<0<4M=0<4i<0<4j=0<5D<0<5E=0<5a<0<5b=0<6<<0<6==0<6X<0<6Y=0<74<"
        + "0<75=0<7P<0<7Q=0<7m<0<7n=0<8H<0<8I=0<8e<0<8f=0<9@<0<9A=0<9]<0<9^=0<:8<0<:9=0<:T<0<:U=0<;0<0<;1=0<;L<"
        + "0<;M=0<;i<0<;j=0<<D<0<<E=0<<a<0<<b=0<=<<0<===0<=X<0<=Y=0<>4<0<>5=0<>P<0<>Q=0<>m<0<>n=0<?H<0<?I=0<?e<"
        + "0<?f=0<@@<0<@A=0<@]<0<@^=0<A8<0<A9=0<AT<0<AU=0<B0<0<B1=0<BL<0<BM=0<Bi<0<Bj=0<CD<0<CE=0<Ca<0<Cb=0<D<<"
        + "0<D==0<DX<0<DY=0<E4<0<E5=0<EP<0<EQ=0<Em<0<En=0<FH<0<FI=0<Fe<0<Ff=0<G@<0<GA=0<G]<0<G^=0<H8<0<H9=0<HT<"
        + "0<HU=0<I0<0<I1=0<IL<0<IM=0<Ii<0<Ij=0<JD<0<JE=0<Ja<0<Jb=0<K<<0<K==0<KX<0<KY=0<L4<0<L5=0<LP<0<LQ=0<Lm<"
        + "0<Ln=0<MH<0<MI=0<Me<0<Mf=0<N@<0<NA=0<N]<0<N^=0<O8<0<O9=0<OT<0<OU=0<P0<0<P1=0<PL<0<PM=0<Pi<0<Pj=0<QD<"
        + "0<QE=0<Qa<0<Qb=0<R<<0<R==0<RX<0<RY=0<S4<0<S5=0<SP<0<SQ=0<Sm<0<Sn=0<TH<0<TI=0<Te<0<Tf=0<U@<0<UA=0<U]<"
        + "0<U^=0<V8<0<V9=0<VT<0<VU=0<W0<0<W1=0<WL<0<WM=0<Wi<0<Wj=0<XD<0<XE=0<Xa<0<Xb=0<Y<<0<Y==0<YX<0<YY=0<Z4<"
        + "0<Z5=0<ZP<0<ZQ=0<Zm<0<Zn=0<[H<0<[I=0<[e<0<[f=0<]@<0<]A=0<]]<0<]^=0<^8<0<^9=0<^T<0<^U=0<_0<0<_1=0<_L<"
        + "0<_M=0<_i<0<_j=0<`D<0<`E=0<`a<0<`b=0<a<<0<a==0<aX<0<aY=0<b4<0<b5=0<bP<0<bQ=0<bm<0<bn=0<cH<0<cI=0<ce<"
        + "0<cf=0<d@<0<dA=0<d]<0<d^=0<e8<0<e9=0<eT<0<eU=0<f0<0<f1=0<fL<0<fM=0<fi<0<fj=0<gD<0<gE=0<ga<0<gb=0<h<<"
        + "0<h==0<hX<0<hY=0<i4<0<i5=0<iP<0<iQ=0<im<0<in=0<jH<0<jI=0<je<0<jf=0<k@<0<kA=0<k]<0<k^=0<l8<0<l9=0<lT<"
        + "0<lU=0<m0<0<m1=0<mL<0<mM=0<mi<0<mj=0<nD<0<nE=0<na<0<nb=0<o<<0<o==0<oX<0<oY=0<p4<0<p5=0<pP<0<pQ=0<pm<"
        + "0<pn=0=0H<0=0I=0=0e<0=0f=0=1@<0=1A=0=1]<0=1^=0=28<0=29=0=2T<0=2U=0=30<0=31=0=3L<0=3M=0=3i<0=3j=0=4D<"
        + "0=4E=0=4a<0=4b=0=5<<0=5==0=5X<0=5Y=0=64<0=65=0=6P<0=6Q=0=6m<0=6n=0=7H<0=7I=0=7e<0=7f=0=8@<0=8A=0=8]<"
        + "0=8^=0=98<0=99=0=9T<0=9U=0=:0<0=:1=0=:L<0=:M=0=:i<0=:j=0=;D<0=;E=0=;a<0=;b=0=<<<0=<==0=<X<0=<Y=0==4<"
        + "0==5=0==P<0==Q=0==m<0==n=0=>H<0=>I=0=>e<0=>f=0=?@<0=?A=0=?]<0=?^=0=@8<0=@9=0=@T<0=@U=0=A0<0=A1=0=AL<"
        + "0=AM=0=Ai<0=Aj=0=BD<0=BE=0=Ba<0=Bb=0=C<<0=C==0=CX<0=CY=0=D4<0=D5=0=DP<0=DQ=0=Dm<0=Dn=0=EH<0=EI=0=Ee<"
        + "0=Ef=0=F@<0=FA=0=F]<0=F^=0=G8<0=G9=0=GT<0=GU=0=H0<0=H1=0=HL<0=HM=0=Hi<0=Hj=0=ID<0=IE=0=Ia<0=Ib=0=J<<"
        + "0=J==0=JX<0=JY=0=K4<0=K5=0=KP<0=KQ=0=Km<0=Kn=0=LH<0=LI=0=Le<0=Lf=0=M@<0=MA=0=M]<0=M^=0=N8<0=N9=0=NT0"
        + "0=Na:0=O700=O;;0=Om00=P030>0000?]N40?]O00?i040?i@00?iP40?ia00?lp30?m000?oN40?oP00?pa30?pm00@7n40@7o0"
        + "0@;P40@;Q00@=g40@=l00@X140@X400@X540@X700@X<40@X@00@Xi40@Xl00@Xp40@Y000@[U40@[W00@eT40@eX00@n640@nA0"
        + "0A0080A0140A0280A0300A0i40A1700A1p40A2280A2300A2a80A2d40A2h80A2j40A2l00A2n70A2o00A3=70A3>00A4040A430"
        + "0A4W40A4]80A4^40A4f00A5580A5700A5d40A5e00A6040A6280A6300A6d80A6g40A6p80A7100A7270A7400A7940A7=00A8]8"
        + "0A8`40A8c80A8e40A8f80A8g40A8i00A8o40A8p00A;O40A;P80A;S40A;[00A<040A<280A<400A<l40A<n00A<o40A<p80A=04"
        + "0A=180A=500A=780A=900A=;80A=>00A=G40A=H00A=R80A=T00A=V40A=^00A=a40A=f00A@f80A@i40AA080AA240AA580AA64"
        + "0AA700AAN40AAO00ABa40ABb80ABd40ABj80ABk40ABl80ABn40ABo80ABp40AC180AC240AC400AF`40AFa80AFc40AFg00AFi8"
        + "0AFm40AFo80AFp40AG100AGL40AGN00AHa80AHd40AHl80AHn40AHo80AHp40AI100AJ[40AJ]80AJ^40AJ_80AJa40AJg80AJh4"
        + "0AJi00ALM40ALP80ALR40ALV80ALW40AL]00AP]80AP`40APi80APj40APl00AWA80AWD40AWH00AWJ40AWL80AWP40AWQ00AWT8"
        + "0AWU00AX140AX;00AXd40AXj80AXk70AXl40AXp00AY740AY800AYA40AYG80AYI40AYL00AZ470AZ:40AZG80AZH40AZJ00Aa`8"
        + "0Aaa40Aah00Aai40Aao80Aap40Ab000AcB40AcX00AcY80AcZ40Acb80Acc40Ace80Acf40Ach00Aeb40Aeh00Aek40Ael00Aem4"
        + "0Aeo00Aep40Af670Af740Af800Ag:80Ag?00Ag@40AgB00AgC80AgE40AgF80AgG40AgH00Ald40Alf80Alh00C@a30C@j00F[a4"
        + "0F[f00F]a40F]h00Fn?40Fn@00FnA80Fo800Fo?40FoC00KcM40KcO00KcP30KcT00M5U40M5V80M5W40M5Z00M5^80M5_40M5d3"
        + "0M5l40M6300M6540M6<00M6Z40M6_00M9240M9500MX040MXh00MXl40MY^00MYf40MYg00MZ440MZ500MZK40MZP00MZQ40MZa0"
        + "0N0040N0700N0840N0I00N0K40N0R00N0S40N0U00N0V40N0[00N4a40N4h00N;]40N;a00NS@40NSG00NU440NU;00O00>0O400"
        + "0O4=>0O4@00O4`>0O4a00O5]>0O5c00O5o>0O6000O6>>0O6?00O6A>0O6K00O6^>0O7V60O8000O81>0O8@00O8J>0O8K00O8`>"
        + "0O8a00O8c>0O8l00O8m>0O9000O99>0O?l40O@0>0ODo00OE6>0OI@00OJ0>0OL000OMe>0ON000OOE>0OP000OP<>0OP@00OQ8>"
        + "0OQ@00OQJ>0OQP00OR8>0OR@00OR_>0OT000OT<>0OTl00OTm>0OU600OU7>0Opo03P0033P0P43P2033P4043P7a33Q000"
    }
    return rbsGcbTable
}
function rbsGcbDig(t, i) {
    var c = t.charCodeAt(i)
    return c <= 91 ? c - 48 : c - 49
}
// The class of code point cp.
function rbsGcb(cp) {
    var t = rbsGcbTab()
    var lo = 0
    var hi = t.length / 5 - 1
    while (lo < hi) {
        var mid = Math.floor((lo + hi + 1) / 2)
        var st = ((rbsGcbDig(t, mid * 5) * 64 + rbsGcbDig(t, mid * 5 + 1)) * 64 + rbsGcbDig(t, mid * 5 + 2)) * 64
            + rbsGcbDig(t, mid * 5 + 3)
        if (st <= cp) { lo = mid } else { hi = mid - 1 }
    }
    return rbsGcbDig(t, lo * 5 + 4)
}
// Whether rules GB3 ... GB999 break between classes p and k: ri counts the
// Regional_Indicators ending at p, emo is 1 after a pictograph and its Extends
// and 2 after the ZWJ that follows them (GB11).
function rbsGcbBreaks(p, k, ri, emo) {
    if (p == 1 && k == 2) { return false }
    if ((p >= 1 && p <= 3) || (k >= 1 && k <= 3)) { return true }
    if (p == 9 && (k == 9 || k == 10 || k == 12 || k == 13)) { return false }
    if ((p == 10 || p == 12) && (k == 10 || k == 11)) { return false }
    if ((p == 11 || p == 13) && k == 11) { return false }
    if (k == 4 || k == 5 || k == 8 || p == 7) { return false }
    if (emo == 2 && k == 14) { return false }
    return !(p == 6 && k == 6 && ri % 2 == 1)
}
// String#grapheme_clusters of the UTF-8 bytes b (one char per byte), as byte
// strings. A byte that starts no valid character is a cluster of its own, with a
// break on both sides. Go twin: rubyGraphemes in abnf/jsrtrubygcb.go.
function rbsGraphemeBytes(b) {
    var out = []
    var start = 0
    var i = 0
    var p = 0 - 1
    var ri = 0
    var emo = 0
    while (i < b.length) {
        var n = rbsU8At(b, i)
        var k = 0 - 1
        if (n > 0) { k = rbsGcb(n == 1 ? b.charCodeAt(i) : rbsU8Cp(b, i, n)) } else { n = 1 }
        if (i > start && (k < 0 || p < 0 || rbsGcbBreaks(p, k, ri, emo))) {
            out.push(b.substring(start, i))
            start = i
        }
        if (k == 6) { ri = ri + 1 } else { ri = 0 }
        if (k == 14) { emo = 1 }
        else if (emo == 1 && k == 4) { emo = 1 }
        else if (emo == 1 && k == 5) { emo = 2 }
        else { emo = 0 }
        p = k
        i = i + n
    }
    if (i > start) { out.push(b.substring(start, i)) }
    return out
}
// String#grapheme_clusters of the text s.
function rbsGraphemes(s) {
    var bs = rbsGraphemeBytes(rbsBinOf(s))
    var out = []
    var i = 0
    while (i < bs.length) {
        out.push(rbsBinDecode(bs[i]))
        i = i + 1
    }
    return out
}

// ----- Unicode normalization (UAX #15, Unicode 12.1 as MRI 2.6 has it) -----
// The tables are /usr/bin/ruby 2.6's lib/unicode_normalize/tables.rb, code points
// in three digits of rbsGcbTable's encoding:
//     ccc   runs of one canonical combining class: start(3) class(2)
//     nfd   key(3) n(1), then the n code points of its full canonical
//           decomposition
//     nfkd  key(3) n(1), then the n code points of its compatibility mapping;
//           n = 0 is a run instead, count(2) target(3): key + i maps to target + i
//     nfc   a(3) b(3) c(3): c is the primary composite of a followed by b, sorted
//           by a, then b
// Go twins: rubyNorm*Table in abnf/jsrtrubynorm.go, kept equal by
// abnf/ruby_unicode_test.go.
var rbsNormTabs = null
function rbsNormTab() {
    if (rbsNormTabs !== null) { return rbsNormTabs }
    var ccc = ""
        + "000000<03V0<E3X0<F3L0<J3X0<K3H0<L3L0<Q3:0<S3L0<W3:0<Y3L0<e010<j3L0<n3V0=53a0=63V0=73L0=:3V0==3L0=?00"
        + "0=@3V0=C3L0=G3V0=H3X0=I3L0=K3V0=L3Y0=M3Z0=O3Y0=P3Z0=R3Y0=S3V0=a000B33V0B8000FA3L0FB3V0FF3L0FG3V0FJ3N"
        + "0FK3L0FL3V0FR3L0FX3V0FZ3L0F[3V0F^3N0F_3T0F`3V0Fa0:0Fb0;0Fc0<0Fd0=0Fe0>0Ff0?0Fg0@0Fh0A0Fi0B0Fj0C0Fl0D"
        + "0Fm0E0Fn0F0Fo000Fp0G0G0000G10H0G20I0G3000G43V0G53L0G6000G70B0G8000H@3V0HH0N0HI0O0HJ0P0HK000I;0K0I<0L"
        + "0I=0M0I>0N0I?0O0I@0P0IA0Q0IB0R0IC3V0IE3L0IG3V0IL3L0IM3V0IO3L0IP000Ia0S0Ib000KF3V0KM000KO3V0KS3L0KT3V"
        + "0KU000KW3V0KY000KZ3L0K[3V0K^3L0K_000LA0T0LB000La3V0Lb3L0Lc3V0Le3L0Lf3V0Lh3L0Lk3V0Ll3L0Ln3V0Lo3L0Lp3V"
        + "0M23L0M33V0M43L0M53V0M63L0M73V0M83L0M93V0M;000O[3V0Oc3L0Od3V0Oe000On3L0Oo000PF3V0PJ000PK3V0PT000PU3V"
        + "0PX000PY3V0P_000QI3L0QL000SC3L0SD3V0SR000SS3L0ST3V0SV3L0SW3V0SY3L0SZ3V0S^3L0Sa0K0Sb0L0Sc0M0Sd3V0Sg3L"
        + "0Sh3V0Sj3L0Sl3V0T0000Tm070Tn000U=090U>000UA3V0UB3L0UC3V0UE000Vm070Vn000W=090W>000Wo3V0Wp000Xm070Xn00"
        + "0Y=090Y>000Zm070Zn000[=090[>000]m070]n000^=090^>000`=090`>000b=090b>000bE1D0bF1K0bG000cm070cn000d=09"
        + "0d>000el090en000f=090f>000h:090h;000ii1W0ik090il000j81[0j<000ki1g0kk090kl000l81k0l<000mH3L0mJ000mf3L"
        + "0mg000mh3L0mi000mj3H0mk000nb210nc220nd000ne240nf000nk220no000o0220o1000o23V0o4090o5000o63V0o8000p63L"
        + "0p70010h0710i0010j0910l0012=3L12>001=M3V1=P001LD091LE001Le091Lf001OB091OC001OM3V1ON001RY3T1RZ001Tj3N"
        + "1Tk3V1Tl3L1Tm001XG3V1XH3L1XI001YP091YQ001Yf3V1Yn001Yp3L1Z0001Za3V1Zf3L1Zl3V1Zn3L1Zo001]e071]f001^409"
        + "1^5001^[3V1^]3L1^^3V1^e001_Z091_]001`V071`W001`c091`e001ah071ai001d@3V1dC001dD011dE3L1dJ3V1dL3L1dP3V"
        + "1dQ001dR011dY001d^3L1d_001de3V1df001di3V1dk001h03V1h23L1h33V1h:3L1h;3V1h=3Z1h>3F1h?3L1h@3:1hA3V1hg3X"
        + "1hh3T1hj3L1hk001hl3V1hm3Y1hn3L1ho3V1hp3L1i00023@3V23B0123D3V23H0123K3V23M0023Q3V23R0023U0123W3V23X3L"
        + "23Y3V23Z0123]3L23a3V23b002d`3V2dc002fp092g0002hP3V2i00030Z3J30[3T30]3X30^3N30_3P30a0032I0832K00:I`3V"
        + ":Ia00:Ie3V:Io00:JN3V:JP00:Ka3V:Kc00:P609:P700:S409:S500:SP3V:Sc00:T[3L:T_00:UC09:UD00:Vd07:Ve00:W009"
        + ":W100:Za3V:Zb00:Zc3V:Ze3L:Zf00:Zh3V:Zj00:Zo3V:[000:[13V:[200:[g09:[h00:`^09:`_00?]N0J?]O00?iP3V?iW3L"
        + "?i_3V?ia00@7n3L@7o00@;P3L@;Q00@=g3V@=l00@X=3L@X>00@X?3V@X@00@Xi3V@Xj01@Xk3L@Xl00@Xp09@Y000@[U3V@[V3L"
        + "@[W00@eT3V@eX00@n63L@n83V@n;3L@n<3V@n=3L@nA00A1609A1700A1p09A2000A2j09A2k07A2l00A403VA4300A4d09A4f00"
        + "A5d07A5e00A7009A7100A7:07A7;00A8f09A8g07A8h00A;Y07A;Z09A;[00A<l07A<n00A==09A=>00A=V3VA=^00A=a3VA=f00"
        + "AA209AA300AA607AA700AAN3VAAO00AC209AC307AC400AFp09AG007AG100AHp09AI000AJg09AJh07AJi00AL[09AL]00APj09"
        + "APk07APl00AWP09AWQ00AXe09AXf00AY709AY800AZI09AZJ00Aap09Ab000Af207Af300Af409Af600AgG09AgH00F[a01F[f00"
        + "F]a3VF]h00KcN01KcO00M5U3HM5W01M5Z00M5^3RM5_3HM5d00M5l3LM6300M653VM6:3LM6<00M6Z3VM6_00M923VM9500N003V"
        + "N0700N083VN0I00N0K3VN0R00N0S3VN0U00N0V3VN0[00N4a3VN4h00N;]3VN;a00NS@3LNSG00NU43VNU:07NU;00"
    var nfd = ""
        + "03020110<003120110<103220110<203320110<303420110<803520110<:03720130<W03820150<003920150<103:20150<2"
        + "03;20150<803<20190<003=20190<103>20190<203?20190<803A201>0<303B201?0<003C201?0<103D201?0<203E201?0<3"
        + "03F201?0<803I201E0<003J201E0<103K201E0<203L201E0<803M201I0<103P201Q0<003Q201Q0<103R201Q0<203S201Q0<3"
        + "03T201Q0<803U201Q0<:03W201S0<W03X201U0<003Y201U0<103Z201U0<203[201U0<803]201Y0<003^201Y0<103_201Y0<2"
        + "03`201Y0<803b201_0<303c201`0<003d201`0<103e201`0<203f201`0<303g201`0<803j201f0<003k201f0<103l201f0<2"
        + "03m201f0<803n201j0<103p201j0<804020110<4041201Q0<404220110<6043201Q0<604420110<X045201Q0<X04620130<1"
        + "047201S0<104820130<2049201S0<204:20130<704;201S0<704<20130<<04=201S0<<04>20140<<04?201T0<<04B20150<4"
        + "04C201U0<404D20150<604E201U0<604F20150<704G201U0<704H20150<X04I201U0<X04J20150<<04K201U0<<04L20170<2"
        + "04M201W0<204N20170<604O201W0<604P20170<704Q201W0<704R20170<W04S201W0<W04T20180<204U201X0<204X20190<3"
        + "04Y201Y0<304Z20190<404[201Y0<404]20190<604^201Y0<604_20190<X04`201Y0<X04a20190<704e201:0<204f201Z0<2"
        + "04g201;0<W04h201[0<W04j201<0<104k201]0<104l201<0<W04m201]0<W04n201<0<<04o201]0<<053201>0<1054201_0<1"
        + "055201>0<W056201_0<W057201>0<<058201_0<<05<201?0<405=201`0<405>201?0<605?201`0<605@201?0<;05A201`0<;"
        + "05D201B0<105E201c0<105F201B0<W05G201c0<W05H201B0<<05I201c0<<05J201C0<105K201d0<105L201C0<205M201d0<2"
        + "05N201C0<W05O201d0<W05P201C0<<05Q201d0<<05R201D0<W05S201e0<W05T201D0<<05U201e0<<05X201E0<305Y201f0<3"
        + "05Z201E0<405[201f0<405]201E0<605^201f0<605_201E0<:05`201f0<:05a201E0<;05b201f0<;05c201E0<X05d201f0<X"
        + "05e201G0<205f201h0<205g201I0<205h201j0<205i201I0<805j201J0<105k201k0<105l201J0<705m201k0<705n201J0<<"
        + "05o201k0<<06P201?0<K06Q201`0<K06`201E0<K06a201f0<K07=20110<<07>201Q0<<07?20190<<07@201Y0<<07A201?0<<"
        + "07B201`0<<07C201E0<<07D201f0<<07E301E0<80<407F301f0<80<407G301E0<80<107H301f0<80<107I301E0<80<<07J30"
        + "1f0<80<<07K301E0<80<007L301f0<80<007N30110<80<407O301Q0<80<407P30110<70<407Q301Q0<70<407R20360<407S2"
        + "03V0<407V20170<<07W201W0<<07X201;0<<07Y201[0<<07Z201?0<X07[201`0<X07]301?0<X0<407^301`0<X0<407_206h0"
        + "<<07`20:B0<<07a201Z0<<07e20170<107f201W0<107i201>0<007j201_0<007k30110<:0<107l301Q0<:0<107m20360<107"
        + "n203V0<107o203H0<107p203i0<108020110<?081201Q0<?08220110<A083201Q0<A08420150<?085201U0<?08620150<A08"
        + "7201U0<A08820190<?089201Y0<?08:20190<A08;201Y0<A08<201?0<?08=201`0<?08>201?0<A08?201`0<A08@201B0<?08"
        + "A201c0<?08B201B0<A08C201c0<A08D201E0<?08E201f0<?08F201E0<A08G201f0<A08H201C0<V08I201d0<V08J201D0<V08"
        + "K201e0<V08N20180<<08O201X0<<08V20110<708W201Q0<708X20150<W08Y201U0<W08Z301?0<80<408[301`0<80<408]301"
        + "?0<30<408^301`0<30<408_201?0<708`201`0<708a301?0<70<408b301`0<70<408c201I0<408d201j0<40=010<00=110<1"
        + "0=310<C0=420<80<10=e10:j0=o100l0>5202X0<10>620>A0<10>7102h0>820>E0<10>920>G0<10>:20>I0<10><20>O0<10>"
        + ">20>U0<10>?20>Y0<10>@30>j0<80<10>Z20>I0<80>[20>U0<80>]20>b0<10>^20>f0<10>_20>h0<10>`20>j0<10>a30?50<"
        + "80<10?:20>j0<80?;20?50<80?<20>p0<10?=20?50<10?>20?90<10?C20?B0<10?D20?B0<80@020@E0<00@120@E0<80@320@"
        + "C0<10@720@60<80@<20@J0<10@=20@H0<00@>20@S0<60@I20@H0<60@j20@i0<60A@20@f0<00AA20@f0<80AC20@d0<10AG20A"
        + "F0<80AL20@k0<10AM20@i0<00AN20A30<60Ag20Ae0<?0Ah20Af0<?0C120@F0<60C220@g0<60C@20@@0<60CA20@a0<60CB20@"
        + "@0<80CC20@a0<80CF20@E0<60CG20@f0<60CJ20CH0<80CK20CI0<80CL20@F0<80CM20@g0<80CN20@G0<80CO20@h0<80CR20@"
        + "H0<40CS20@i0<40CT20@H0<80CU20@i0<80CV20@N0<80CW20@o0<80CZ20CX0<80C[20CY0<80C]20@^0<80C^20A=0<80C_20@"
        + "S0<40C`20A30<40Ca20@S0<80Cb20A30<80Cc20@S0<;0Cd20A30<;0Ce20@W0<80Cf20A70<80Ci20@[0<80Cj20A;0<80HR20H"
        + "W0IC0HS20HW0ID0HT20I80ID0HU20HW0IE0HV20I:0ID0K020KE0ID0K220K10ID0KC20KB0ID0TY20TX0Tm0Tb20Ta0Tm0Te20T"
        + "d0Tm0UH20TE0Tm0UI20TF0Tm0UJ20TG0Tm0UK20TL0Tm0UL20TQ0Tm0UM20TR0Tm0UN20T[0Tm0UO20T`0Tm0W;20W70Vo0W<20W"
        + "70WG0WL20VQ0Vm0WM20VR0Vm0WO20V`0Vm0Xd20Xc0Xm0Xg20Xi0Xm0YI20XF0Xm0YJ20XG0Xm0YK20XL0Xm0YN20X[0Xm0^820^"
        + "70^F0^;20^70]o0^<20^70^G0^L20]Q0]m0^M20]R0]m0_D20_B0`G0`:20`60_o0`;20`70_o0`<20`60`G0b820b60bF0d020c"
        + "p0dE0d720d60dE0d820d60dF0d:20d60d20d;30d60d20dE0f:20f60eo0f;20f70eo0f<20f60fG0hJ20hI0h:0hL20hI0h?0hM"
        + "30hI0h?0h:0hN20hI0hO0n320n20oh0n=20n<0oh0nB20nA0oh0nG20nF0oh0nL20nK0oh0nY20n00of0nd20nb0nc0nf20nb0ne"
        + "0ng20oc0o00ni20od0o00o120nb0o00oC20oB0oh0oM20oL0oh0oR20oQ0oh0oW20oV0oh0o]20o[0oh0oj20o@0of10V210U10_"
        + "1]621]51]f1]821]71]f1]:21]91]f1]<21];1]f1]>21]=1]f1]B21]A1]f1]l21]k1]f1]n21]m1]f1^021]o1]f1^121]p1]f"
        + "1^321^21]f1i020110<U1i1201Q0<U1i220120<71i3201R0<71i420120<S1i5201R0<S1i620120<b1i7201R0<b1i830130<W"
        + "0<11i9301S0<W0<11i:20140<71i;201T0<71i<20140<S1i=201T0<S1i>20140<b1i?201T0<b1i@20140<W1iA201T0<W1iB2"
        + "0140<^1iC201T0<^1iD30150<40<01iE301U0<40<01iF30150<40<11iG301U0<40<11iH20150<^1iI201U0<^1iJ20150<a1i"
        + "K201U0<a1iL30150<W0<61iM301U0<W0<61iN20160<71iO201V0<71iP20170<41iQ201W0<41iR20180<71iS201X0<71iT201"
        + "80<S1iU201X0<S1iV20180<81iW201X0<81iX20180<W1iY201X0<W1iZ20180<_1i[201X0<_1i]20190<a1i^201Y0<a1i_301"
        + "90<80<11i`301Y0<80<11ia201;0<11ib201[0<11ic201;0<S1id201[0<S1ie201;0<b1if201[0<b1ig201<0<S1ih201]0<S"
        + "1ii301<0<S0<41ij301]0<S0<41ik201<0<b1il201]0<b1im201<0<^1in201]0<^1io201=0<11ip201^0<11j0201=0<71j12"
        + "01^0<71j2201=0<S1j3201^0<S1j4201>0<71j5201_0<71j6201>0<S1j7201_0<S1j8201>0<b1j9201_0<b1j:201>0<^1j;2"
        + "01_0<^1j<301?0<30<11j=301`0<30<11j>301?0<30<81j?301`0<30<81j@301?0<40<01jA301`0<40<01jB301?0<40<11jC"
        + "301`0<40<11jD201@0<11jE201a0<11jF201@0<71jG201a0<71jH201B0<71jI201c0<71jJ201B0<S1jK201c0<S1jL301B0<S"
        + "0<41jM301c0<S0<41jN201B0<b1jO201c0<b1jP201C0<71jQ201d0<71jR201C0<S1jS201d0<S1jT301C0<10<71jU301d0<10"
        + "<71jV301C0<<0<71jW301d0<<0<71jX301C0<S0<71jY301d0<S0<71jZ201D0<71j[201e0<71j]201D0<S1j^201e0<S1j_201"
        + "D0<b1j`201e0<b1ja201D0<^1jb201e0<^1jc201E0<T1jd201f0<T1je201E0<a1jf201f0<a1jg201E0<^1jh201f0<^1ji301"
        + "E0<30<11jj301f0<30<11jk301E0<40<81jl301f0<40<81jm201F0<31jn201g0<31jo201F0<S1jp201g0<S1k0201G0<01k12"
        + "01h0<01k2201G0<11k3201h0<11k4201G0<81k5201h0<81k6201G0<71k7201h0<71k8201G0<S1k9201h0<S1k:201H0<71k;2"
        + "01i0<71k<201H0<81k=201i0<81k>201I0<71k?201j0<71k@201J0<21kA201k0<21kB201J0<S1kC201k0<S1kD201J0<b1kE2"
        + "01k0<b1kF201X0<b1kG201e0<81kH201h0<:1kI201j0<:1kK205p0<71kP20110<S1kQ201Q0<S1kR20110<91kS201Q0<91kT3"
        + "0110<20<11kU301Q0<20<11kV30110<20<01kW301Q0<20<01kX30110<20<91kY301Q0<20<91kZ30110<20<31k[301Q0<20<3"
        + "1k]30110<S0<21k^301Q0<S0<21k_30110<60<11k`301Q0<60<11ka30110<60<01kb301Q0<60<01kc30110<60<91kd301Q0<"
        + "60<91ke30110<60<31kf301Q0<60<31kg30110<S0<61kh301Q0<S0<61ki20150<S1kj201U0<S1kk20150<91kl201U0<91km2"
        + "0150<31kn201U0<31ko30150<20<11kp301U0<20<11l030150<20<01l1301U0<20<01l230150<20<91l3301U0<20<91l4301"
        + "50<20<31l5301U0<20<31l630150<S0<21l7301U0<S0<21l820190<91l9201Y0<91l:20190<S1l;201Y0<S1l<201?0<S1l=2"
        + "01`0<S1l>201?0<91l?201`0<91l@301?0<20<11lA301`0<20<11lB301?0<20<01lC301`0<20<01lD301?0<20<91lE301`0<"
        + "20<91lF301?0<20<31lG301`0<20<31lH301?0<S0<21lI301`0<S0<21lJ301?0<K0<11lK301`0<K0<11lL301?0<K0<01lM30"
        + "1`0<K0<01lN301?0<K0<91lO301`0<K0<91lP301?0<K0<31lQ301`0<K0<31lR301?0<K0<S1lS301`0<K0<S1lT201E0<S1lU2"
        + "01f0<S1lV201E0<91lW201f0<91lX301E0<K0<11lY301f0<K0<11lZ301E0<K0<01l[301f0<K0<01l]301E0<K0<91l^301f0<"
        + "K0<91l_301E0<K0<31l`301f0<K0<31la301E0<K0<S1lb301f0<K0<S1lc201I0<01ld201j0<01le201I0<S1lf201j0<S1lg2"
        + "01I0<91lh201j0<91li201I0<31lj201j0<31m020>b0<C1m120>b0<D1m230>b0<C0<01m330>b0<D0<01m430>b0<C0<11m530"
        + ">b0<D0<11m630>b0<C0=21m730>b0<D0=21m820>A0<C1m920>A0<D1m:30>A0<C0<01m;30>A0<D0<01m<30>A0<C0<11m=30>A"
        + "0<D0<11m>30>A0<C0=21m?30>A0<D0=21m@20>f0<C1mA20>f0<D1mB30>f0<C0<01mC30>f0<D0<01mD30>f0<C0<11mE30>f0<"
        + "D0<11mH20>E0<C1mI20>E0<D1mJ30>E0<C0<01mK30>E0<D0<01mL30>E0<C0<11mM30>E0<D0<11mP20>h0<C1mQ20>h0<D1mR3"
        + "0>h0<C0<01mS30>h0<D0<01mT30>h0<C0<11mU30>h0<D0<11mV30>h0<C0=21mW30>h0<D0=21mX20>G0<C1mY20>G0<D1mZ30>"
        + "G0<C0<01m[30>G0<D0<01m]30>G0<C0<11m^30>G0<D0<11m_30>G0<C0=21m`30>G0<D0=21ma20>j0<C1mb20>j0<D1mc30>j0"
        + "<C0<01md30>j0<D0<01me30>j0<C0<11mf30>j0<D0<11mg30>j0<C0=21mh30>j0<D0=21mi20>I0<C1mj20>I0<D1mk30>I0<C"
        + "0<01ml30>I0<D0<01mm30>I0<C0<11mn30>I0<D0<11mo30>I0<C0=21mp30>I0<D0=21n020>p0<C1n120>p0<D1n230>p0<C0<"
        + "01n330>p0<D0<01n430>p0<C0<11n530>p0<D0<11n820>O0<C1n920>O0<D1n:30>O0<C0<01n;30>O0<D0<01n<30>O0<C0<11"
        + "n=30>O0<D0<11n@20?50<C1nA20?50<D1nB30?50<C0<01nC30?50<D0<01nD30?50<C0<11nE30?50<D0<11nF30?50<C0=21nG"
        + "30?50<D0=21nI20>U0<D1nK30>U0<D0<01nM30>U0<D0<11nO30>U0<D0=21nP20?90<C1nQ20?90<D1nR30?90<C0<01nS30?90"
        + "<D0<01nT30?90<C0<11nU30?90<D0<11nV30?90<C0=21nW30?90<D0=21nX20>Y0<C1nY20>Y0<D1nZ30>Y0<C0<01n[30>Y0<D"
        + "0<01n]30>Y0<C0<11n^30>Y0<D0<11n_30>Y0<C0=21n`30>Y0<D0=21na20>b0<01nb20>b0<11nc20>f0<01nd20>f0<11ne20"
        + ">h0<01nf20>h0<11ng20>j0<01nh20>j0<11ni20>p0<01nj20>p0<11nk20?50<01nl20?50<11nm20?90<01nn20?90<11o030"
        + ">b0<C0=51o130>b0<D0=51o240>b0<C0<00=51o340>b0<D0<00=51o440>b0<C0<10=51o540>b0<D0<10=51o640>b0<C0=20="
        + "51o740>b0<D0=20=51o830>A0<C0=51o930>A0<D0=51o:40>A0<C0<00=51o;40>A0<D0<00=51o<40>A0<C0<10=51o=40>A0<"
        + "D0<10=51o>40>A0<C0=20=51o?40>A0<D0=20=51o@30>h0<C0=51oA30>h0<D0=51oB40>h0<C0<00=51oC40>h0<D0<00=51oD"
        + "40>h0<C0<10=51oE40>h0<D0<10=51oF40>h0<C0=20=51oG40>h0<D0=20=51oH30>G0<C0=51oI30>G0<D0=51oJ40>G0<C0<0"
        + "0=51oK40>G0<D0<00=51oL40>G0<C0<10=51oM40>G0<D0<10=51oN40>G0<C0=20=51oO40>G0<D0=20=51oP30?90<C0=51oQ3"
        + "0?90<D0=51oR40?90<C0<00=51oS40?90<D0<00=51oT40?90<C0<10=51oU40?90<D0<10=51oV40?90<C0=20=51oW40?90<D0"
        + "=20=51oX30>Y0<C0=51oY30>Y0<D0=51oZ40>Y0<C0<00=51o[40>Y0<D0<00=51o]40>Y0<C0<10=51o^40>Y0<D0<10=51o_40"
        + ">Y0<C0=20=51o`40>Y0<D0=20=51oa20>b0<61ob20>b0<41oc30>b0<00=51od20>b0=51oe30>b0<10=51og20>b0=21oh30>b"
        + "0=20=51oi20>A0<61oj20>A0<41ok20>A0<01ol20>A0<11om20>A0=51oo10>j1p1202X0=21p230>h0<00=51p320>h0=51p43"
        + "0>h0<10=51p620>h0=21p730>h0=20=51p820>E0<01p920>E0<11p:20>G0<01p;20>G0<11p<20>G0=51p=21op0<01p>21op0"
        + "<11p?21op0=21p@20>j0<61pA20>j0<41pB30>j0<80<01pC30>j0<80<11pF20>j0=21pG30>j0<80=21pH20>I0<61pI20>I0<"
        + "41pJ20>I0<01pK20>I0<11pM21po0<01pN21po0<11pO21po0=21pP20?50<61pQ20?50<41pR30?50<80<01pS30?50<80<11pT"
        + "20?10<C1pU20?10<D1pV20?50=21pW30?50<80=21pX20>U0<61pY20>U0<41pZ20>U0<01p[20>U0<11p]20>Q0<D1p^202X0<0"
        + "1p_202X0<11p`101P1pc30?90<00=51pd20?90=51pe30?90<10=51pg20?90=21ph30?90=20=51pi20>O0<01pj20>O0<11pk2"
        + "0>Y0<01pl20>Y0<11pm20>Y0=51pn102e2001202201120324V10>Y24Z101;24[20110<:26J226@0<i26K226B0<i26_226D0<"
        + "i27=227@0<i27>227D0<i27?227B0<i28422830<i28922880<i28<228;0<i28T228S0<i28V228U0<i291228m0<i29422930<"
        + "i29722950<i29922980<i29P200n0<i29R229Q0<i29^229=0<i29_200m0<i29`200o0<i29a229T0<i29b229U0<i29e229c0<"
        + "i29f229d0<i29i229g0<i29j229h0<i2:0229k0<i2:1229l0<i2:422:20<i2:522:30<i2:822:60<i2:922:70<i2:]22:R0<"
        + "i2:^22:X0<i2:_22:Y0<i2:`22:[0<i2;P229m0<i2;Q229n0<i2;R22:A0<i2;S22:B0<i2;Z22:c0<i2;[22:d0<i2;]22:e0<"
        + "i2;^22:f0<i2<Y13082<Z13092[L22[M0<i31<231;32I31>231=32I31@231?32I31B231A32I31D231C32I31F231E32I31H23"
        + "1G32I31J231I32I31L231K32I31N231M32I31P231O32I31R231Q32I31U231T32I31W231V32I31Y231X32I31a231`32I31b23"
        + "1`32J31d231c32I31e231c32J31g231f32I31h231f32J31j231i32I31k231i32J31m231l32I31n231l32J32D231632I32N23"
        + "2M32I32]232[32I32_232^32I32a232`32I32c232b32I32e232d32I32g232f32I32i232h32I32k232j32I32m232l32I32o23"
        + "2n32I330232p32I332233132I335233432I337233632I339233832I33@233?32I33A233?32J33C233B32I33D233B32J33F23"
        + "3E32I33G233E32J33I233H32I33J233H32J33L233K32I33M233K32J33e232V32I33h233`32I33i233a32I33j233b32I33k23"
        + "3c32I33o233n32I?T018b8?T116Ke?T218l:?T318d8?T416lA?T514ic?T615?U?T719oL?T819oL?T915UA?T:197A?T;15F7?"
        + "T<15U8?T=167g?T>17IY?T?17o5?T@18Hp?TA18Nk?TB18Si?TC192??TD16X2?TE16eK?TF173I?TG17?N?TH18@n?TI195Z?TJ"
        + "19Wb?TK14k2?TL15=f?TM16]4?TN178K?TO18H^?TP19iN?TQ15f@?TR16p[?TS18G=?TT18UT?TU16;9?TV187H?TW18PO?TX15"
        + "l:?TY16LG?TZ16fZ?T[17;m?T]193>?T^14o6?T_156h?T`15;N?Ta16C4?Tb16[C?Tc178@?Td17KW?Te1801?Tf18H6?Tg18IL"
        + "?Th18h`?Ti19Lc?Tj19^`?Tk19hk?Tl17R<?Tm17Up?Tn17gP?To18?9?Tp19<4?U019jp?U118[F?U215SO?U315m4?U417bP?U"
        + "5181o?U6179R?U717S:?U818d2?U919Kh?U:15SH?U;15bR?U<16XC?U=16hJ?U>16m??U?17e`?U@17ih?UA19I;?UB15;B?UC1"
        + "82;?UD157L?UE157<?UF17XL?UG17go?UH18?b?UI19If?UJ18_0?UK16;??UL16X2?UM18[o?UN14ij?UO15`W?UP160B?UQ17>"
        + "7?UR17Ea?US15<G?UT17Sl?UU14op?UV15oY?UW14i=?UX16d<?UY16Ei?UZ17eR?U[15?3?U]15QN?U^17L1?U_18A9?U`18ZZ?"
        + "Ua16_k?Ub18oa?Uc16c8?Ud16;o?Ue18;U?Uf16>P?Ug17EU?Uh14k_?Ui155Y?Uj1579?Uk16R1?Ul17dW?Um189`?Un18[B?Uo"
        + "197??Up15;f?V015A2?V115Ud?V215l]?V316G5?V416po?V517TZ?V619F^?V719YZ?V819kG?V919l>?V:15:K?V;16K6?V<16"
        + "^h?V=18nR?V>15je?V?166@?V@1680?VA16BJ?VB16mS?VC1759?VD17B9?VE17W:?VF17he?VG181`?VH18mV?VI18C_?VJ190S"
        + "?VK19=:?VL158G?VM15:S?VN15Bn?VO1738?VP18S2?VQ18ZZ?VR15l9?VS15pf?VT16=l?VU16__?VV17ao?VW17=f?VX14lT?V"
        + "Y15Kj?VZ15`W?V[15gk?V]160L?V^17>c?V_17AY?V`17oJ?Va1816?Vb198e?Vc19Kg?Vd19M8?Ve19PH?Vf14o;?Vg17V_?Vh1"
        + "96e?Vi19Ji?Vj163Q?Vk14k6?Vl153J?Vm15`_?Vn15ap?Vo16FI?Vp16X2?W0177>?W117I2?W218Cm?W3191m?W419o=?W516J"
        + "8?W619H_?W715:9?W816Ml?W916Od?W:16f1?W;16kL?W<17@9?W=17EI?W>17Q[?W?17e@?W@19QN?WA155^?WB168_?WC19Ii?"
        + "WD150[?WE15eI?WF16hZ?WG18mZ?WH15o;?WI1654?WJ16PG?WK17>7?WL19J6?WM158Y?WN15@??WO15bU?WP16HC?WQ16M>?WR"
        + "16RX?WS16dU?WT17@6?WU17GR?WV17nj?WW18S??WX18SQ?WY197<?WZ19KR?W[15<p?W]16kk?W^15@M?W_177@?W`17BH?Wa18"
        + "Gk?Wb19JS?Wc19bG?Wd19kO?We16NG?Wf16h;?Wg187X?Wh17[;?Wi17]P?Wj17cB?Wk17;0?Wl172I?Wm18^H?Wn14l0?Wo18<g"
        + "?Wp158k?X01587?X115kV?X216;C?X317dF?X415_5?X516eN?X616Je?X718ml?X818Q<?X919I=?X:18V;?X;15lC?X<1550?X"
        + "=15G0?X@15QJ?XB16Ie?XE157N?XF17<Z?XG17K:?XH17Tm?XI17UN?XJ17UU?XK17V??XL19MF?XM17co?XN17on?XP18HB?XR1"
        + "8[i?XU190i?XV193n?XZ19S`?X[19Sm?X]19TX?X^19ge?X_193N?X`19Jh?Xa14o_?Xb153W?Xc155=?Xd15;9?Xe15;T?Xf15="
        + "A?Xg15FM?Xh15H6?Xi15IX?Xj15Q0?Xk15RX?Xl15bT?Xm15b_?Xn162D?Xo165X?Xp166>?Y0167c?Y116E??Y216GR?Y316JA?"
        + "Y416R5?Y516fh?Y616iJ?Y716mR?Y8175_?Y9178[?Y:17@R?Y;17RA?Y<17To?Y=17U9?Y>17U8?Y?17U@?Y@17UF?YA17UM?YB"
        + "17V=?YC17V>?YD17Y0?YE17Z1?YF17`0?YG17he?YH17i9?YI17j1?YJ17nc?YK1805?YL187^?YM189j?YN189j?YO18AG?YP18"
        + "T@?YQ18VF?YR18]1?YS18]j?YT18dC?YU18e8?YV18og?YW190i?YX19KS?YY19Op?YZ19Pl?Y[161f?Y]1T;_?Y^188H?Ya14iV"
        + "?Yb156f?Yc155X?Yd14o0?Ye1555?Yf1560?Yg15;7?Yh15;k?Yi15FM?Yj15EE?Yk15FI?Yl15GR?Ym15QJ?Yn15Rd?Yo15U4?Y"
        + "p15UD?Z015YR?Z115]X?Z215lB?Z315lI?Z415nY?Z515o^?Z6163H?Z7165>?Z81648?Z9166>?Z:165P?Z;167c?Z<168e?Z=1"
        + "6?4?Z>16@L?Z?16AB?Z@16EF?ZA16Ie?ZB16LG?ZC16LK?ZD16MF?ZE16^j?ZF16_k?ZG16f1?ZH16lK?ZI16l;?ZJ16mR?ZK170"
        + "N?ZL175_?ZM17NW?ZN178f?ZO17:`?ZP17<Z?ZQ17Ab?ZR17D6?ZS17Dl?ZT17HM?ZU17HO?ZV17K:?ZW17KK?ZX17Ke?ZY17M:?"
        + "ZZ17M0?Z[17S<?Z]17Zb?Z^17`0?Z_17bl?Z`17fK?Za17he?Zb17mo?Zc1805?Zd18=B?Ze18?`?Zf18Mj?Zg18U1?Zh18V6?Zi"
        + "18VF?Zj18Zp?Zk18[i?Zl18[;?Zm18]1?Zn18[o?Zo18[^?Zp18]j?[018_:?[118e8?[218mi?[3191c?[4196I?[5199g?[619"
        + "Im?[719KS?[819MF?[919OK?[:19Op?[;19P;?[<19Pl?[=19]B?[>19oL?[?1RQ:?[@1RQ4?[A1S?E?[B13_M?[C140H?[D140j"
        + "?[E1U99?[F1Ud@?[G1WlC?[H19n3?[I19o>?]M20GI0Fe?]O20Gc0Fh?]Z20GY0G1?][20GY0G2?]]30GY0Fm0G1?]^30GY0Fm0G"
        + "2?]_20G@0Fh?]`20G@0Fi?]a20G@0Fm?]b20GA0Fm?]c20GB0Fm?]d20GC0Fm?]e20GD0Fm?]f20GE0Fm?]g20GF0Fm?]i20GH0F"
        + "m?]j20GI0Fm?]k20GJ0Fm?]l20GK0Fm?]m20GL0Fm?]o20GN0Fm?^020GP0Fm?^120GQ0Fm?^320GS0Fm?^420GT0Fm?^620GV0F"
        + "m?^720GW0Fm?^820GX0Fm?^920GY0Fm?^:20GZ0Fm?^;20GE0Fj?^<20GA0Fp?^=20GK0Fp?^>20GT0FpA2J2A2IA2kA2L2A2KA2"
        + "kA2[2A2UA2kA4_2A4bA4WA4`2A4cA4WA=;2A=7A<oA=<2A=7A=GABl2ABjABkABm2ABjABaABo2ABjABnAFk2AFiAF`AFl2AFjAF"
        + "`M5N2M5GM5UM5O2M5HM5UM5P3M5HM5UM5_M5Q3M5HM5UM5`M5R3M5HM5UM5aM5S3M5HM5UM5bM5T3M5HM5UM5cM6l2M6jM5UM6m2"
        + "M6kM5UM6n3M6jM5UM5_M6o3M6kM5UM5_M6p3M6jM5UM5`M703M6kM5UM5``P014in`P114ii`P214j1`P31P4R`P414nP`P514o_"
        + "`P614ol`P71502`P8151k`P9152I`P:153W`P;153?`P<13BN`P=1PHk`P>155=`P?155D`P@155T`PA155h`PB1PDL`PC13Bj`P"
        + "D155W`PE156=`PF1PE;`PG156G`PH156T`PI14l<`PJ156]`PK156f`PL1Y7O`PM157f`PN1583`PO13CO`PP158l`PQ1596`PR1"
        + "59c`PS159h`PT13DE`PU15;7`PV15;9`PW15;T`PX15;k`PY15<5`PZ15<6`P[15<G`P]15=9`P^15=A`P_15=J`P`15=d`Pa15="
        + "n`Pb15=p`Pc15=p`Pd15=p`Pe1PX]`Pf171a`Pg15?:`Ph15?O`Pi1P^S`Pj15?[`Pk15?b`Pl15@6`Pm15BN`Pn15@i`Po15A8`"
        + "Pp15AX`Q015BR`Q115Cg`Q215D@`Q315EC`Q415ES`Q515F4`Q615F4`Q715FI`Q815F[`Q915Fd`Q:15G2`Q;15LF`Q<15H6`Q="
        + "15LG`Q>15IA`Q?15Ie`Q@1587`QA15S_`QB15O>`QC15Oe`QD15P=`QE15N;`QF15Pc`QG15Pb`QH15R]`QI1QCT`QJ15Sc`QK15"
        + "Sh`QL15T6`QM15TJ`QN15TR`QO15UR`QP1QJX`QQ1QKZ`QR15W]`QS15XK`QT15XW`QU15WH`QV15YV`QW13K_`QX13Km`QY15]8"
        + "`QZ15]o`Q[15]o`Q]1QW8`Q^15`3`Q_15`H`Q`15`W`Qa15`d`Qb1Q]H`Qc15`p`Qd15a6`Qe15nC`Qf15aR`Qg13N1`Qh15bP`Q"
        + "i15b_`Qj15d0`Qk15c=`Ql1QhT`Qm15f3`Qn1QhV`Qo15f_`Qp15f[`R015fm`R115hQ`R215hR`R313P``R415hn`R515iX`R61"
        + "5in`R715jY`R813QR`R91R63`R:13Qm`R;15ka`R<15kd`R=15kg`R>15l:`R?1Z>B`R@15lo`RA1R<b`RB1R<b`RC1881`RD15m"
        + "R`RE15mR`RF13S7`RG1S:i`RH1V7J`RI15nR`RJ15n[`RK13SS`RL15oJ`RM15p=`RN15pG`RO15pj`RP1621`RQ13Tk`RR13TL`"
        + "RS162D`RT1RKD`RU1637`RV1658`RW165<`RX165>`RY165<`RZ165k`R[166>`R]166c`R^166T`R_166``R`167N`Ra167c`Rb"
        + "167g`Rc168@`Rd168K`Re169M`Rf16:b`Rg16;D`Rh16=@`Ri1R]<`Rj16<n`Rk16;m`Rl16=X`Rm16>3`Rn16?T`Ro1R`b`Rp16"
        + "@R`S016?5`S116>Y`S213X_`S316AY`S416Ao`S516BM`S616Ah`S713Y]`S816E?`S916E]`S:1S0:`S;16GS`S<16Ki`S=16I9"
        + "`S>13]I`S?16JA`S@13]8`SA13[T`SB156B`SC156E`SD16L0`SE16JL`SF182^`SG14?I`SH16LG`SI16LK`SJ16LQ`SK16MN`S"
        + "L16MC`SM1S?3`SN13^9`SO16Ok`SP16N5`SQ16QB`SR16R5`SS1SA^`ST16R>`SU16PO`SV16TD`SW13_M`SX16U2`SY16VS`SZ1"
        + "6WZ`S[16ZX`S]1SJS`S^16[K`S_13aH`S`16]Q`Sa1SRW`Sb16^D`Sc13b>`Sd16^c`Se16_O`Sf16_k`Sg16_l`Sh1SZ=`Si1Qe"
        + ";`Sj1S[k`Sk16b>`Sl1Scm`Sm16cp`Sn16d=`So16bW`Sp16eF`T016eo`T116fh`T216f1`T316fY`T416fi`T516g5`T61SeN`"
        + "T716ee`T816i``T916j_`T:13ed`T;16l;`T<16l7`T=1SlA`T>16hj`T?16n_`T@1SnN`TA1So>`TB16p6`TC170j`TD170N`TE"
        + "170K`TF13gF`TG171:`TH171n`TI171h`TJ172^`TK1PDU`TL1755`TM1T9S`TN176L`TO1T>[`TP178X`TQ178f`TR179@`TS1T"
        + "H8`TT17:0`TU17:E`TV1TLf`TW1TPD`TX17=k`TY17>;`TZ13k]`T[17>U`T]13ki`T^13ki`T_17A7`T`17AL`Ta17Ab`Tb17B5"
        + "`Tc17C:`Td13mK`Te17DT`Tf1Tag`Tg17Do`Th1TcB`Ti17Ea`Tj1R6O`Tk17H@`Tl1ToQ`Tm1Toi`Tn1U14`To13pm`Tp1408`U"
        + "017Ke`U11U3d`U21U3c`U31U4I`U41U4d`U517LN`U617LO`U717LO`U817M:`U9140j`U:17N;`U;1416`U<142F`U=1U@M`U>1"
        + "7Q>`U?17R<`U@17S<`UA143S`UB1UHV`UC17UF`UD1UJJ`UE1UK5`UF17V?`UG17W[`UH144``UI17Y0`UJ17Y:`UK17Y?`UL1UU"
        + "m`UM1UZW`UN1UZW`UO17[_`UP1482`UQ1U_[`UR17`6`US17`9`UT148W`UU1Uc0`UV17dB`UW14:P`UX17dX`UY17dS`UZ17e0`"
        + "U[1Uo6`U]17fS`U^14<1`U_17h7`U`17i2`Ua17j5`Ub14<e`Uc1V8X`Ud1V97`Ue14=I`Uf1V;I`Ug17nk`Uh1V<o`Ui17oE`Uj"
        + "17pk`Uk1805`Ul1VCJ`Um1VDS`Un181P`Uo1VFX`Up181a`V01S=O`V114?E`V2182c`V31843`V414@;`V5184o`V615Zf`V71V"
        + "NW`V81VNf`V91S>C`V:1S>L`V;1881`V<1884`V=18oN`V>14A[`V?18:A`V@18:;`VA18:M`VB15:d`VC18:b`VD18:d`VE18:n"
        + "`VF18;V`VG1V]m`VH18;U`VI18<M`VJ18=S`VK18>^`VL18<S`VM18>n`VN18?W`VO18AG`VP18=C`VQ18?:`VR18?<`VS18?L`V"
        + "T1Vag`VU1Vf[`VV1VdE`VW14D[`VX18Cb`VY18Cd`VZ18DF`V[1W?:`V]18ET`V^1Vm]`V_14EM`V`14EQ`Va1Vob`Vb1W3B`Vc1"
        + "4E[`Vd18I@`Ve18IL`Vf18IW`Vg18IY`Vh18JY`Vi18J8`Vj18L>`Vk18KR`Vl18Mj`Vm18LX`Vn18M[`Vo18N6`Vp14GG`W018O"
        + "Q`W118P1`W214Gj`W318QP`W418QS`W51WIW`W618SG`W718SN`W814Hf`W918Sk`W:13Bl`W;1WR_`W<1WUV`W=14Jo`W>14K7`"
        + "W?18ZP`W@18[^`WA18_:`WB18bE`WC1WcX`WD18c[`WE18d1`WF18eK`WG18fh`WH1Wm``WI1PP4`WJ18h;`WK18gm`WL18ha`WM"
        + "1PSN`WN18lD`WO18mi`WP1XGB`WQ1XG^`WR192D`WS193b`WT194A`WU1XL_`WV194K`WW198i`WX19;G`WY19;H`WZ199m`W[19"
        + "?j`W]19@E`W^1X`k`W_19F;`W`14VE`Wa19Fh`Wb1Xfh`Wc14WV`Wd19K3`We15gc`Wf19LS`Wg1Y55`Wh1Y8J`Wi14Y_`Wj14Yg"
        + "`Wk19OP`Wl1Y@:`Wm14Zc`Wn1YBF`Wo19P;`Wp19P;`X019PY`X11YFg`X219SR`X314]d`X419TY`X519VW`X619W2`X719Wo`X"
        + "814`>`X91Y]a`X:19]B`X;19b0`X<19dn`X=14d>`X>14d^`X?19fW`X@1Z3>`XA14di`XB1Z45`XC1Z8>`XD1Z:A`XE19kl`XF1"
        + "4fF`XG19lj`XH19lo`XI19m5`XJ19m?`XK19mF`XL19ml`XM1ZH0"
    var nfkd = ""
        + "02P100P02X200P0<802Z101Q02`200P0<402c100c02d100d02e200P0<102f10>m02i200P0<W02j100b02k101`02m300b2140"
        + "0e02n300b21400c02o300d21400e04c201901:04d201Y01Z04p201<02h050201]02h05920:m01_05p101d074201405n07520"
        + "1405o076201T05o077201<01:078201<01Z079201]01Z07:201>01:07;201>01Z07<201_01Z07b201401J07c201401k07d20"
        + "1T01k0:a101X0:b109V0:c101Z0:d101c0:e109j0:f109l0:g10:10:h101h0:i101j0;H200P0<60;I200P0<70;J200P0<:0;"
        + "K200P0<X0;L200P0<30;M200P0<;0;P109S0;Q101]0;R101d0;S101i0;T10:E0=k200P0=50>4200P0<10>5300P0<80<10?@1"
        + "0>c0?A10>i0?B10>U0?C20>U0<10?D20>U0<80?E10?60?F10?00?a10>k0?b10?10?c10?20?e10>H0?f10>f0?j10>S0F720EU"
        + "0F20If20HW0Ie0Ig20I80Ie0Ih20K70Ie0Ii20I:0Ie0id20j=0ic0kd20l=0kc0lL20k[0kI0lM20k[0kQ0m<10m;0nh20oc0o1"
        + "0nj20od0o113m113L1e]10111e^10361e_10121ea10141eb10151ec106>1ed0080171em101?1en108R1eo101@1ep101B1f01"
        + "01D1f1101E1f2101G1f3101Q1f4109@1f5109A1f611e21f7101R1f8101T1f9101U1f:109I1f;109K1f<109L1f=101W1f?101"
        + "[1f@101^1fA105;1fB101`1fC109D1fD11eF1fE11eG1fF101a1fG101e1fH101f1fI11eM1fJ109`1fK101g1fL11eU1fM0030>"
        + "c1fP10?61fQ10?71fR101Y1fS101c1fT101f1fU101g1fV10>c1fW10>d1fX10?11fY10?61fZ10?71fi10@n1gK109B1gL101S1"
        + "gM109E1gN103a1gO109L1gP101V1gQ109O1gR109Q1gS109U1gT00309X1gW11fl1gX10:M1gY109^1gZ11g51g[10:O1g]109b1"
        + "g^109a1g_00409c1gc109i1gd10:21ge10:31gf106[1gg10:91gh10::1gi11eL1gj10:;1gk10:<1gl101k1gm0030:@1gp10>"
        + "i1kJ201Q0:o1kK201d0<71on200P0<C1op200P0<C1p0200P0=21p1300P0<80=21p=300P0<C0<01p>300P0<C0<11p?300P0<C"
        + "0=21pM300P0<D0<01pN300P0<D0<11pO300P0<D0=21p^300P0<80<01p_300P0<80<11pn200P0<11po200P0<D200100P20110"
        + "0P202100P203100P204100P205100P206100P207100P208100P209100P20:100P20A120@20G200P0<d20T100_20U200_00_2"
        + "0V300_00_00_20`100P20d220c20c20e320c20c20c20g220f20f20h320f20f20f20m200Q00Q20o200P0<5217200p00p21820"
        + "0p00Q219200Q00p21G420c20c20c20c21O100P21a100a21b101Y21e00600e21k100[21l128B21m100n21n100X21o100Y21p1"
        + "01_22000:00a22:100[22;128B22<100n22=100X22>100Y22@101Q22A101U22B101`22C101i22D109I22E101X22F00401[22"
        + "J101a22K101d22L101e22X201B01d240301Q00`01S241301Q00`01d2421013243202a013245301S00`01`246301S00`01f24"
        + "7106@249202a01624:101W24;101824<101824=101824>101X24?104W24@101924A101924B101<24C101]24E101>24F201>0"
        + "1`24I00301@24L101B24M101B24P201C01=24Q301D01501<24R201D01=24T101J24X101J24]101224^101324`101U24a1015"
        + "24b101624d101=24e101`24f0040G@24j101Y24l301601101H24m10?024n10>d24o10>C24p10>P250128A2551014256101T2"
        + "57101U258101Y259101Z25@300b21400h25A300b21400j25B400b21400b00a25C300b21400d25D300c21400d25E300b21400"
        + "f25F300c21400f25G300d21400f25H300e21400f25I300b21400g25J300f21400g25K300b21400i25L300d21400i25M300f2"
        + "1400i25N300h21400i25O200b21425P101925Q201901925R301901901925S201901F25T101F25U201F01925V301F01901925"
        + "W401F01901901925X201901H25Y101H25Z201H01925[301H01901925]101<25^101325_101425`101=25a101Y25b201Y01Y2"
        + "5c301Y01Y01Y25d201Y01g25e101g25f201g01Y25g301g01Y01Y25h401g01Y01Y01Y25i201Y01i25j101i25k201i01Y25l30"
        + "1i01Y01Y25m101]25n101S25o101T25p101^269300a21400d28]228[28[28^328[28[28[28`228_28_28a328_28_28_2AP00"
        + "900b2AY200b00a2AZ200b00b2A[200b00c2A]200b00d2A^200b00e2A_200b00f2A`200b00g2Aa200b00h2Ab200b00i2Ac200"
        + "b00j2Ad200c00a2Ae300X00b00Y2Af300X00c00Y2Ag300X00d00Y2Ah300X00e00Y2Ai300X00f00Y2Aj300X00g00Y2Ak300X0"
        + "0h00Y2Al300X00i00Y2Am300X00j00Y2An400X00b00a00Y2Ao400X00b00b00Y2Ap400X00b00c00Y2B0400X00b00d00Y2B140"
        + "0X00b00e00Y2B2400X00b00f00Y2B3400X00b00g00Y2B4400X00b00h00Y2B5400X00b00i00Y2B6400X00b00j00Y2B7400X00"
        + "c00a00Y2B8200b00_2B9200c00_2B:200d00_2B;200e00_2B<200f00_2B=200g00_2B>200h00_2B?200i00_2B@200j00_2BA"
        + "300b00a00_2BB300b00b00_2BC300b00c00_2BD300b00d00_2BE300b00e00_2BF300b00f00_2BG300b00g00_2BH300b00h00"
        + "_2BI300b00i00_2BJ300b00j00_2BK300c00a00_2BL300X01Q00Y2BM300X01R00Y2BN300X01S00Y2BO300X01T00Y2BP300X0"
        + "1U00Y2BQ300X01V00Y2BR300X01W00Y2BS300X01X00Y2BT300X01Y00Y2BU300X01Z00Y2BV300X01[00Y2BW300X01]00Y2BX3"
        + "00X01^00Y2BY300X01_00Y2BZ300X01`00Y2B[300X01a00Y2B]300X01b00Y2B^300X01c00Y2B_300X01d00Y2B`300X01e00Y"
        + "2Ba300X01f00Y2Bb300X01g00Y2Bc300X01h00Y2Bd300X01i00Y2Be300X01j00Y2Bf300X01k00Y2Bg00J0112C@00J01Q2CZ1"
        + "00a2X<428[28[28[28[2Ye300k00k00n2Yf200n00n2Yg300n00n00n2bm101Z2bn101F2f`12fQ2kO16`=2ld19oO2m014i02m1"
        + "14iX2m214ig2m314ip2m414jI2m514k52m614k<2m714kP2m814kk2m9154p2m:155U2m;155[2m<15622m=156F2m>156[2m?15"
        + "7P2m@157f2mA15802mB15:K2mC15;j2mD15<E2mE15<J2mF15<i2mG15=12mH15=L2mI15=Y2mJ15>22mK15>g2mL15?82mM15?S"
        + "2mN15KG2mO15LO2mP15S[2mQ15T22mR15T:2mS15TE2mT15TW2mU15Ud2mV15^@2mW15_02mX15`i2mY15a?2mZ15aR2m[15ai2m"
        + "]15b_2m^15bb2m_15hK2m`15hU2ma15hb2mb15ho2mc15jc2md15jk2me15jp2mf15le2mg15lo2mh15m;2mi15mC2mj15n@2mk1"
        + "5nQ2ml15nd2mm15p32mn16882mo168g2mp169;2n016D`2n116De2n216F72n316FG2n416FT2n516Fj2n616GP2n716GU2n816K"
        + "a2n916L82n:16LX2n;16]P2n<16^R2n=16^j2n>16_d2n?16`;2n@16`D2nA16`K2nB16a?2nC16aD2nD16ae2nE171[2nF178Z2"
        + "nG178g2nH178l2nI178p2nJ17972nK179I2nL179K2nM17:]2nN17>42nO17>92nP17CL2nQ17CV2nR17DH2nS17DO2nT17DX2nU"
        + "17Da2nV17F;2nW17FB2nX17Ig2nY17In2nZ17J_2n[17Jp2n]17K_2n^17OK2n_17OR2n`17Od2na17Tk2nb17Vi2nc17Vo2nd17"
        + "Ye2ne17[;2nf17[j2ng17bd2nh17di2ni17mg2nj17nA2nk17o:2nl17on2nm18012nn180<2no180B2np180d2o0181p2o11829"
        + "2o2187S2o3187Z2o4187d2o5187m2o6188<2o7188K2o8188O2o9189_2o:189c2o;189i2o<18I=2o=18I[2o>18Q02o?18Q<2o"
        + "@18QS2oA18Uo2oB18V;2oC18WB2oD18X02oE18ah2oF18b62oG18bE2oH18bi2oI18cM2oJ18fT2oK18fa2oL18gd2oM18k[2oN1"
        + "8l:2oO18oK2oP18oa2oQ18of2oR192A2oS19592oT19762oU197<2oV197A2oW19Eh2oX19F02oY19HL2oZ19Jg2o[19Jj2o]19K"
        + "X2o^19MA2o_19MN2o`19MR2oa19MY2ob19O;2oc19O^2od19Od2oe19P12of19RX2og19SK2oh19SO2oi19VF2oj19VI2ok19V]2"
        + "ol19ZX2om19[H2on19[O2oo19]U2op19]`2p019]c2p119]m2p219^J2p319dU2p419jf2p519jp2p619kU2p719kl2p819l32p9"
        + "19l=2p:19lA2p;19lj2p<19ln2p=19m>2p>19mC2p?19mP2p@19ml2pA19n:2pB19nB2pC19o=2pD19oL2pE19oP300100P30g13"
        + "0B30i15=130j15=430k15=532K200P32I32L200P32J32O232832:33p232d33834b114034c114134d116Z34e114234f116]34"
        + "g116^34h00314334k00616a350114J351003146354114Q35500:14935?00E15Q35T115P35U114D35V114E35W117735X11783"
        + "5Y117<35Z117>35[117C35]117G35^117I35_114L35`117M35a117O35b114M35c114N35d114P35e114R35f114S35g114W35h"
        + "114Y35i00514[35n114c35o114g35p11503601157361115<362117b363117c36400315G36711643681165369116836:116A3"
        + "6;116B36<116D36=116N36>116Q36B14i036C14k<36D14i936E15KK36F14i:36G14i^36H14i;36I17Dc36J14jI36K14iI36L"
        + "14i136M15TY36N15La36O14kk380300X14000Y381300X14200Y382300X14300Y383300X14500Y384300X14600Y385300X147"
        + "00Y386300X14900Y387300X14;00Y388300X14<00Y389300X14>00Y38:300X14?00Y38;300X14@00Y38<300X14A00Y38=300"
        + "X14B00Y38>400X14015Q00Y38?400X14215Q00Y38@400X14315Q00Y38A400X14515Q00Y38B400X14615Q00Y38C400X14715Q"
        + "00Y38D400X14915Q00Y38E400X14;15Q00Y38F400X14<15Q00Y38G400X14>15Q00Y38H400X14?15Q00Y38I400X14@15Q00Y3"
        + "8J400X14A15Q00Y38K400X14B15Q00Y38L400X14<15_00Y38M700X14;15Y14<15U16[00Y38N600X14;15Y14B15_00Y38P300"
        + "X4i000Y38Q300X4k<00Y38R300X4i900Y38S300X5KK00Y38T300X4kD00Y38U300X55^00Y38V300X4i300Y38W300X55[00Y38"
        + "X300X4jM00Y38Y300X5=100Y38Z300X6L800Y38[300X71[00Y38]300X6ae00Y38^300X6LX00Y38_300X97A00Y38`300X5LO0"
        + "0Y38a300X6GU00Y38b300X6PZ00Y38c300X6L900Y38d300X7To00Y38e300X5@=00Y38f300X79j00Y38g300X8cQ00Y38h300X"
        + "7UM00Y38i300X5:e00Y38j300X4lS00Y38k300X5Am00Y38l300X5^V00Y38m300X7KS00Y38n300X4m100Y38o300X8d700Y38p"
        + "300X5=D00Y390300X7U^00Y391300X4mA00Y392300X87Z00Y393300X87d00Y39415E?39515jm39616F739717_?39@301@01D"
        + "01539A200c00b39B200c00c39C200c00d39D200c00e39E200c00f39F200c00g39G200c00h39H200c00i39I200c00j39J200d"
        + "00a39K200d00b39L200d00c39M200d00d39N200d00e39O200d00f39P114039Q114239R114339S00314539V114939W114;39X"
        + "114<39Y00514>39_214015Q39`214215Q39a214315Q39b214515Q39c214615Q39d214715Q39e214915Q39f214;15Q39g214<"
        + "15Q39h214>15Q39i214?15Q39j214@15Q39k214A15Q39l214B15Q39m514>15Q16h14015Y39n414<15_14;15e39o214;15_3:"
        + "014i03:114k<3:214i93:315KK3:414kD3:5155^3:614i33:7155[3:814jM3:915=13::16L83:;171[3:<16ae3:=16LX3:>1"
        + "97A3:?15LO3:@16GU3:A16PZ3:B16L93:C17To3:D15@=3:E179j3:F18cQ3:G17UM3:H15:e3:I17WH3:J17Dh3:K15Ud3:L191"
        + "Y3:M154Z3:N15=a3:O16dX3:P19P53:Q14mA3:R156I3:S16^S3:T14i:3:U14i^3:V14i;3:W15hV3:X15?d3:Y15<l3:Z15_G3"
        + ":[15^V3:]17KS3:^14m13:_18d73:`15=D3:a15TL3:b200d00g3:c200d00h3:d200d00i3:e200d00j3:f200e00a3:g200e00"
        + "b3:h200e00c3:i200e00d3:j200e00e3:k200e00f3:l200e00g3:m200e00h3:n200e00i3:o200e00j3:p200f00a3;0200b6L"
        + "83;1200c6L83;2200d6L83;3200e6L83;4200f6L83;5200g6L83;6200h6L83;7200i6L83;8200j6L83;9300b00a6L83;:300"
        + "b00b6L83;;300b00c6L83;<201801W3;=301U01c01W3;>201U01F3;?301<01D0143;@132R3;A132T3;B132V3;C132X3;D132"
        + "Z3;E132[3;F132^3;G132`3;H132b3;I132d3;J132f3;K132h3;L132j3;M132l3;N132n3;O132p3;P13313;Q13343;R13363"
        + ";S13383;T00633:3;Z133B3;[133E3;]133H3;^133K3;_00533N3;d133T3;e133V3;f00633X3;l00433`3;p24lT5B<3<0432"
        + "R33A33m3383<1432R33[33E32Q3<2432R33d33J32R3<3332R33m33[3<4432T33;33d32a3<5332T33d3313<6332V32Y33d3<7"
        + "532X32j32`33m3393<8432X33m32[33m3<9332Z33d32j3<:332Z33m33P3<;332[32T33Z3<<432[33Y3333383<=432[33^33Z"
        + "33m3<>332]33^33d3<?332]33d33N3<@232_32]3<A332_33;33m3<B432^33U33Z33m3<C432_33[33033m3<D232^33^3<E532"
        + "^33^32a33Y33P3<F632^33^33Q33m33833[3<G532^33^33`3333383<H332a33Y33P3<I532a33Y33P33833d3<J532`33[32m3"
        + "2T33^3<K432`33^33m33=3<L332b33m32j3<M332d33[33:3<N332d33m33M3<O432f32T32`33[3<P532f33d33133m33P3<Q43"
        + "2h33Z33d32a3<R332l33d3313<S332l33d3383<T333033m32j3<U233732h3<V233933[3<W233833d3<X233:33>3<Y333>333"
        + "3383<Z333?32T3343<[533A33m32l33d3383<]333A33m3343<^433@33m33]33[3<_533D32R32j33833[3<`333D32`33[3<a2"
        + "33D32d3<b233C33[3<c533E32Q33Y3333393<d433E32S33m3383<e533F33332h32W33[3<f333E33Y33d3<g533H32`32p33m3"
        + "3[3<h233J32n3<i333J33;33B3<j333H33[3343<k333J33d32j3<l333J33m32i3<m333I33m32p3<n433M32T33d3383<o333L"
        + "33[3383<p233K33d3=0333M33d3393=1333K33m33[3=2333K33m33d3=3433N32T32`33^3=4333N32T33[3=5333N33333?3=6"
        + "333N33[32`3=7533N33d32h33W33d3=8433O32`33^33d3=9233O33Z3=:533O33Z33@33m33[3=;233Q32]3=<433Q32]33833d"
        + "3==433Q33m33833[3=>333T33m3393=?333T33m33[3=@333V32R33d3=A433Z33333833[3=B233Z33Y3=C333[33D33m3=D433"
        + "[33m33F33[3=E233]33P3=F533]33d33832c33d3=G333`3333383=H200a72j3=I200b72j3=J200c72j3=K200d72j3=L200e7"
        + "2j3=M200f72j3=N200g72j3=O200h72j3=P200i72j3=Q200j72j3=R300b00a72j3=S300b00b72j3=T300b00c72j3=U300b00"
        + "d72j3=V300b00e72j3=W300b00f72j3=X300b00g72j3=Y300b00h72j3=Z300b00i72j3=[300b00j72j3=]300c00a72j3=^30"
        + "0c00b72j3=_300c00c72j3=`300c00d72j3=a300c00e72j3=b301X01@01Q3=c201T01Q3=d201101E3=e301R01Q01c3=f201`"
        + "01F3=g201a01S3=h201T01^3=i301T01^00c3=j301T01^00d3=k201901E3=l25jd68@3=m26H^5B<3=n25TW6^S3=o26H>6cl3"
        + "=p46PZ5m?4mJ7To3>0201a0113>1201_0113>220>m0113>3201^0113>4201[0113>5201;0123>6201=0123>720170123>830"
        + "1S01Q01]3>9401[01S01Q01]3>:201a0163>;201_0163><20>m0163>=20>m01W3>>201^01W3>?201[01W3>@201801k3>A301"
        + "[01801k3>B301=01801k3>C301701801k3>D301D01801k3>E20>m01]3>F201^01]3>G201T01]3>H201[01]3>I201V01^3>J2"
        + "01_01^3>K20>m01^3>L201^01^3>M201S01^3>N201[01^3>O301^01^00c3>P301S01^00c3>Q201^00c3>R301[01^00c3>S30"
        + "1^01^00d3>T301S01^00d3>U201^00d3>V301[01^00d3>W301^28E01d3>X401^28E01d00c3>Y201@01Q3>Z301[01@01Q3>[3"
        + "01=01@01Q3>]301701@01Q3>^301c01Q01T3>_501c01Q01T28E01d3>`601c01Q01T28E01d00c3>a201a01d3>b201_01d3>c2"
        + "0>m01d3>d201^01d3>e201a01F3>f201_01F3>g20>m01F3>h201^01F3>i201[01F3>j201=01F3>k201a01G3>l201_01G3>m2"
        + "0>m01G3>n201^01G3>o201[01G3>p201=01G3?0201[0>Y3?1201=0>Y3?2401Q00_01^00_3?3201201b3?4201S01S3?5201S0"
        + "1T3?6401328E01[01W3?7301301`00_3?8201T0123?9201701j3?:201X01Q3?;201801@3?<201Y01_3?=201;01;3?>201;01"
        + "=3??201[01e3?@201]01^3?A201]01_3?B301]01`01W3?C201]01i3?D201^01R3?E301^01Y01]3?F301^01`01]3?G201@018"
        + "3?H401a00_01^00_3?I301@01@01=3?J201@01B3?K201d01c3?L201C01g3?M201G01R3?N301F28E01^3?O301128E01^3?P20"
        + "0b6GU3?Q200c6GU3?R200d6GU3?S200e6GU3?T200f6GU3?U200g6GU3?V200h6GU3?W200i6GU3?X200j6GU3?Y300b00a6GU3?"
        + "Z300b00b6GU3?[300b00c6GU3?]300b00d6GU3?^300b00e6GU3?_300b00f6GU3?`300b00g6GU3?a300b00h6GU3?b300b00i6"
        + "GU3?c300b00j6GU3?d300c00a6GU3?e300c00b6GU3?f300c00c6GU3?g300c00d6GU3?h300c00e6GU3?i300c00f6GU3?j300c"
        + "00g6GU3?k300c00h6GU3?l300c00i6GU3?m300c00j6GU3?n300d00a6GU3?o300d00b6GU3?p301W01Q01]:JL10A::JM10A<:M"
        + "a1:M`:Oi104V:Oj105C:^L1:LW:^M1:]h:^N109[:^O1:^B?]0201V01V?]1201V01Y?]2201V01]?]3301V01V01Y?]4301V01V"
        + "01]?]5201d01e?]6201d01e?]C20Ee0Eg?]D20Ee0EU?]E20Ee0E[?]F20Eo0Eg?]G20Ee0E^?]P10GR?]Q10G@?]R10GC?]S10G"
        + "D?]T0030GK?]W10GX?]X10GZ?]Y100[?^?20G@0GL?^@10Ib?^A10Ib?^B10Il?^C10Il?^D10Il?^E10Il?^F10Io?^G10Io?^H"
        + "10Io?^I10Io?^J10J0?^K10J0?^L10J0?^M10J0?^N10Ik?^O10Ik?^P10Ik?^Q10Ik?^R10Ip?^S10Ip?^T10Ip?^U10Ip?^V10"
        + "Ij?^W10Ij?^X10Ij?^Y10Ij?^Z10JT?^[10JT?^]10JT?^^10JT?^_10JV?^`10JV?^a10JV?^b10JV?^c10J4?^d10J4?^e10J4"
        + "?^f10J4?^g10J3?^h10J3?^i10J3?^j10J3?^k10J6?^l10J6?^m10J6?^n10J6?^o10J7?^p10J7?_010J7?_110J7?_210J=?_"
        + "310J=?_410J<?_510J<?_610J>?_710J>?_810J8?_910J8?_:10JH?_;10JH?_<10JA?_=10JA?_>10JY?_?10JY?_@10JY?_A1"
        + "0JY?_B10J`?_C10J`?_D10J`?_E10J`?_F10Jd?_G10Jd?_H10Jd?_I10Jd?_J10Jb?_K10Jb?_L10Jb?_M10Jb?_N10Jk?_O10J"
        + "k?_P10Jl?_Q10Jl?_R10Jl?_S10Jl?_T10K0?_U10K0?_V10K1?_W10K1?_X10K1?_Y10K1?_Z10Jo?_[10Jo?_]10Jo?_^10Jo?"
        + "__10KB?_`10KB?_a10KC?_b10KC?`C10J^?`D10J^?`E10J^?`F10J^?`G10K7?`H10K7?`I10K6?`J10K6?`K10K8?`L10K8?`M"
        + "20K70Ie?`N10K;?`O10K;?`P10K5?`Q10K5?`R10K9?`S10K9?`T10K@?`U10K@?`V10K@?`W10K@?`X10I9?`Y10I9?`Z20HV0H"
        + "W?`[20HV0HW?`]20HV0KE?`^20HV0KE?`_20HV0I8?``20HV0I8?`a20HV0K7?`b20HV0K7?`c20HV0K6?`d20HV0K6?`e20HV0K"
        + "8?`f20HV0K8?`g20HV0K@?`h20HV0K@?`i20HV0K@?`j20HV0I9?`k20HV0I9?`l20HV0I9?`m10K<?`n10K<?`o10K<?`p10K<?"
        + "a020HV0H]?a120HV0H^?a220HV0I5?a320HV0I9?a420HV0I:?a520HX0H]?a620HX0H^?a720HX0H_?a820HX0I5?a920HX0I9?"
        + "a:20HX0I:?a;20HZ0H]?a<20HZ0H^?a=20HZ0H_?a>20HZ0I5?a?20HZ0I9?a@20HZ0I:?aA20H[0H]?aB20H[0I5?aC20H[0I9?"
        + "aD20H[0I:?aE20H]0H^?aF20H]0I5?aG20H^0H]?aH20H^0I5?aI20H_0H]?aJ20H_0H^?aK20H_0I5?aL20Hd0H]?aM20Hd0H^?"
        + "aN20Hd0H_?aO20Hd0I5?aP20Hf0H^?aQ20Hf0I5?aR20Hg0H]?aS20Hg0H^?aT20Hg0H_?aU20Hg0I5?aV20Hh0H^?aW20Hh0I5?"
        + "aX20Hi0I5?aY20Hj0H]?aZ20Hj0I5?a[20Hk0H]?a]20Hk0I5?a^20I10H]?a_20I10H^?a`20I10H_?aa20I10I5?ab20I10I9?"
        + "ac20I10I:?ad20I20H^?ae20I20I5?af20I20I9?ag20I20I:?ah20I30HW?ai20I30H]?aj20I30H^?ak20I30H_?al20I30I4?"
        + "am20I30I5?an20I30I9?ao20I30I:?ap20I40H]?b020I40H^?b120I40H_?b220I40I5?b320I40I9?b420I40I:?b520I50H]?"
        + "b620I50H^?b720I50H_?b820I50I5?b920I50I9?b:20I50I:?b;20I60H]?b<20I60H^?b=20I60H_?b>20I60I5?b?20I60I9?"
        + "b@20I60I:?bA20I70H]?bB20I70I5?bC20I70I9?bD20I70I:?bE20I:0H]?bF20I:0H^?bG20I:0H_?bH20I:0I5?bI20I:0I9?"
        + "bJ20I:0I:?bK20Ha0Ia?bL20Hb0Ia?bM20I90Ia?bN300P0I<0IA?bO300P0I=0IA?bP300P0I>0IA?bQ300P0I?0IA?bR300P0I"
        + "@0IA?bS300P0IA0Ia?bT20HV0Hb?bU20HV0Hc?bV20HV0I5?bW20HV0I6?bX20HV0I9?bY20HV0I:?bZ20HX0Hb?b[20HX0Hc?b]"
        + "20HX0I5?b^20HX0I6?b_20HX0I9?b`20HX0I:?ba20HZ0Hb?bb20HZ0Hc?bc20HZ0I5?bd20HZ0I6?be20HZ0I9?bf20HZ0I:?bg"
        + "20H[0Hb?bh20H[0Hc?bi20H[0I5?bj20H[0I6?bk20H[0I9?bl20H[0I:?bm20I10I9?bn20I10I:?bo20I20I9?bp20I20I:?c0"
        + "20I30HW?c120I30I4?c220I30I5?c320I30I9?c420I30I:?c520I40I5?c620I40I9?c720I40I:?c820I50HW?c920I50I5?c:"
        + "20I60Hb?c;20I60Hc?c<20I60I5?c=20I60I6?c>20I60I9?c?20I60I:?c@20I90Ia?cA20I:0Hb?cB20I:0Hc?cC20I:0I5?cD"
        + "20I:0I6?cE20I:0I9?cF20I:0I:?cG20HV0H]?cH20HV0H^?cI20HV0H_?cJ20HV0I5?cK20HV0I7?cL20HX0H]?cM20HX0H^?cN"
        + "20HX0H_?cO20HX0I5?cP20HX0I7?cQ20HZ0H]?cR20HZ0H^?cS20HZ0H_?cT20HZ0I5?cU20HZ0I7?cV20H[0I5?cW20H]0H^?cX"
        + "20H]0I5?cY20H^0H]?cZ20H^0I5?c[20H_0H]?c]20H_0I5?c^20Hd0H]?c_20Hd0H^?c`20Hd0H_?ca20Hd0I5?cb20Hf0H^?cc"
        + "20Hf0H_?cd20Hf0I5?ce20Hg0H]?cf20Hg0H^?cg20Hg0H_?ch20Hg0I5?ci20Hh0H^?cj20Hi0I5?ck20Hj0H]?cl20Hj0I5?cm"
        + "20Hk0H]?cn20Hk0I5?co20I10H]?cp20I10H^?d020I10H_?d120I10I5?d220I20H^?d320I20I5?d420I30H]?d520I30H^?d6"
        + "20I30H_?d720I30I4?d820I30I5?d920I40H]?d:20I40H^?d;20I40H_?d<20I40I5?d=20I40I7?d>20I50H]?d?20I50H^?d@"
        + "20I50H_?dA20I50I5?dB20I60H]?dC20I60H^?dD20I60H_?dE20I60I5?dF20I60I7?dG20I70H]?dH20I70I5?dI20I70Ia?dJ"
        + "20I:0H]?dK20I:0H^?dL20I:0H_?dM20I:0I5?dN20I:0I7?dO20HV0I5?dP20HV0I7?dQ20HX0I5?dR20HX0I7?dS20HZ0I5?dT"
        + "20HZ0I7?dU20H[0I5?dV20H[0I7?dW20Hd0I5?dX20Hd0I7?dY20He0I5?dZ20He0I7?d[20I30I4?d]20I30I5?d^20I40I5?d_"
        + "20I60I5?d`20I60I7?da20I:0I5?db20I:0I7?dc30I00I>0IA?dd30I00I?0IA?de30I00I@0IA?df20Hh0I9?dg20Hh0I:?dh2"
        + "0Hj0I9?di20Hj0I:?dj20Hk0I9?dk20Hk0I:?dl20Hd0I9?dm20Hd0I:?dn20He0I9?do20He0I:?dp20H^0I9?e020H^0I:?e12"
        + "0H]0I9?e220H]0I:?e320H_0I9?e420H_0I:?e520Hf0I9?e620Hf0I:?e720Hg0I9?e820Hg0I:?e920He0H]?e:20He0H^?e;2"
        + "0He0H_?e<20He0I5?e=20He0Hb?e>20Hd0Hb?e?20Hf0Hb?e@20Hg0Hb?eA20Hh0I9?eB20Hh0I:?eC20Hj0I9?eD20Hj0I:?eE2"
        + "0Hk0I9?eF20Hk0I:?eG20Hd0I9?eH20Hd0I:?eI20He0I9?eJ20He0I:?eK20H^0I9?eL20H^0I:?eM20H]0I9?eN20H]0I:?eO2"
        + "0H_0I9?eP20H_0I:?eQ20Hf0I9?eR20Hf0I:?eS20Hg0I9?eT20Hg0I:?eU20He0H]?eV20He0H^?eW20He0H_?eX20He0I5?eY2"
        + "0He0Hb?eZ20Hd0Hb?e[20Hf0Hb?e]20Hg0Hb?e^20He0H]?e_20He0H^?e`20He0H_?ea20He0I5?eb20Hd0I7?ec20He0I7?ed2"
        + "0Hh0I5?ee20Hd0H]?ef20Hd0H^?eg20Hd0H_?eh20He0H]?ei20He0H^?ej20He0H_?ek20Hh0I5?el20Hi0I5?em20HW0I;?en2"
        + "0HW0I;?f@30HZ0H]0I5?fA30HZ0H^0H]?fB30HZ0H^0H]?fC30HZ0H^0I5?fD30HZ0H_0I5?fE30HZ0I50H]?fF30HZ0I50H^?fG"
        + "30HZ0I50H_?fH30H]0I50H^?fI30H]0I50H^?fJ30H^0I50I:?fK30H^0I50I9?fL30Hd0H^0H]?fM30Hd0H]0H^?fN30Hd0H]0I"
        + "9?fO30Hd0I50H^?fP30Hd0I50H^?fQ30Hd0I50H]?fR30Hd0I50I5?fS30Hd0I50I5?fT30Hf0H^0H^?fU30Hf0H^0H^?fV30Hf0"
        + "I50I5?fW30He0H^0I5?fX30He0H^0I5?fY30He0H]0I:?fZ30He0I50H_?f[30He0I50H_?f]30He0I50I5?f^30He0I50I5?f_3"
        + "0Hg0H^0I9?f`30Hg0H_0I5?fa30Hg0H_0I5?fb30Hh0I50H^?fc30Hh0I50H^?fd30Hh0I50I5?fe30Hh0I50I:?ff30Hj0H]0I5"
        + "?fg30Hj0I50I5?fh30Hj0I50I5?fi30Hj0I50I9?fj30Hk0I50I5?fk30Hk0I50I:?fl30Hk0I50I9?fm30I10H_0I5?fn30I10H"
        + "_0I5?fo30I20I50H^?fp30I20I50I5?g030I40H^0I5?g130I40H^0I:?g230I40H^0I9?g330I40H]0H]?g430I40H]0H]?g530"
        + "I40H_0I5?g630I40H_0I5?g730I40I50H^?g830I40I50H^?g930I50H^0H]?g:30I50H^0I5?g;30I50H^0I:?g<30I50H]0H^?"
        + "g=30I50H]0I5?g>30I50H_0H]?g?30I50H_0I5?gB30I50H]0H_?gC30I70I50H]?gD30I70I50I5?gE30I60H^0I5?gF30I60H^"
        + "0I9?gG30I60H]0I5?gH30I60H]0I5?gI30I60H]0I9?gJ30I60I50I:?gK30I60I50I9?gL30I:0I50I5?gM30I:0I50I5?gN30H"
        + "X0H_0I:?gO30HZ0H]0I:?gP30HZ0H]0I9?gQ30HZ0H_0I:?gR30HZ0H_0I9?gS30HZ0I50I:?gT30HZ0I50I9?gU30H]0I50I:?g"
        + "V30H]0H^0I9?gW30H]0I50I9?gX30Hd0H_0I9?gY30Hf0H^0I:?gZ30He0H^0I:?g[30Hg0H^0I:?g]30I40H]0I:?g^30I40I50"
        + "I:?g_30I:0H^0I:?g`30I:0H]0I:?ga30I:0I50I:?gb30I50I50I:?gc30I20I50I:?gd30I60H^0I:?ge30I20I50H^?gf30I4"
        + "0H^0I5?gg30Hj0I50I:?gh30I30I50I:?gi30I60H]0H^?gj30I50H_0I:?gk30I40H]0I5?gl30I30I50I5?gm30I40H]0I5?gn"
        + "30I60H]0H^?go30H]0H^0I:?gp30H^0H]0I:?h030I50H]0I:?h130I10I50I:?h230HX0H^0I:?h330I30I50I5?h430Hj0H]0I"
        + "5?h530Hf0I50I5?h630Hd0H_0I:?h730I60H]0I:?ha30Hf0I40KB?hb30I20I40KB?hc40HW0I40I40I7?hd40HW0I30HX0Hb?h"
        + "e40I50H^0I50H`?hf40Hf0I40Hj0I5?hg40Hb0Hd0I80I4?hh40Hj0I40I:0I7?hi40I80Hd0I40I5?hj30Hf0I40I9?hkB0Hf0I"
        + "40I900P0HW0I40I40I700P0Hj0I40I:0I700P0I80Hd0I40I5?hl80H]0I400P0H]0I40HW0I40I7?hm40Hb0K<0HW0I4?i@100]"
        + "?iA1301?iB1302?iC100k?iD100l?iE100Q?iF100p?iG130F?iH130G?iI300_00_00_?ia200_00_?ib120D?ic120C?id101O"
        + "?ie101O?if100X?ig100Y?ih101l?ii101n?ij130D?ik130E?il130@?im130A?in130:?io130;?ip1308?j01309?j100430<"
        + "?j7101K?j8101M?j9200P0<5?j:200P0<5?j;200P0<5?j<200P0<5?j=101O?j>101O?j?101O?j@100]?jA1301?jB100_?jD1"
        + "00l?jE100k?jF100p?jG100Q?jH120D?jI100X?jJ100Y?jK101l?jL101n?jM130D?jN130E?jO100S?jP100V?jQ100Z?jR100"
        + "[?jS100^?jT100m?jU100o?jV100n?jX101L?jY100T?jZ100U?j[1010?ja200P0I;?jb20I00I;?jc200P0I<?je200P0I=?jg"
        + "200P0I>?jh20I00I>?ji200P0I??jj20I00I??jk200P0I@?jl20I00I@?jm200P0IA?jn20I00IA?jo200P0IB?jp20I00IB?k0"
        + "10HQ?k110HR?k210HR?k310HS?k410HS?k510HT?k610HT?k710HU?k810HU?k910HV?k:10HV?k;10HV?k<10HV?k=10HW?k>10"
        + "HW?k?10HX?k@10HX?kA10HX?kB10HX?kC10HY?kD10HY?kE10HZ?kF10HZ?kG10HZ?kH10HZ?kI10H[?kJ10H[?kK10H[?kL10H["
        + "?kM10H]?kN10H]?kO10H]?kP10H]?kQ10H^?kR10H^?kS10H^?kT10H^?kU10H_?kV10H_?kW10H_?kX10H_?kY10H`?kZ10H`?k"
        + "[10Ha?k]10Ha?k^10Hb?k_10Hb?k`10Hc?ka10Hc?kb10Hd?kc10Hd?kd10Hd?ke10Hd?kf10He?kg10He?kh10He?ki10He?kj1"
        + "0Hf?kk10Hf?kl10Hf?km10Hf?kn10Hg?ko10Hg?kp10Hg?l010Hg?l110Hh?l210Hh?l310Hh?l410Hh?l510Hi?l610Hi?l710H"
        + "i?l810Hi?l910Hj?l:10Hj?l;10Hj?l<10Hj?l=10Hk?l>10Hk?l?10Hk?l@10Hk?lA10I1?lB10I1?lC10I1?lD10I1?lE10I2?"
        + "lF10I2?lG10I2?lH10I2?lI10I3?lJ10I3?lK10I3?lL10I3?lM10I4?lN10I4?lO10I4?lP10I4?lQ10I5?lR10I5?lS10I5?lT"
        + "10I5?lU10I6?lV10I6?lW10I6?lX10I6?lY10I7?lZ10I7?l[10I7?l]10I7?l^10I8?l_10I8?l`10I9?la10I9?lb10I:?lc10"
        + "I:?ld10I:?le10I:?lf20I40HR?lg20I40HR?lh20I40HS?li20I40HS?lj20I40HU?lk20I40HU?ll20I40HW?lm20I40HW?m10"
        + "1N00Q?nO12V5?nP12V6?nQ1302?nR130<?nS130=?nT1301?nU133l?nV133c?nW132Q?nX132S?nY132U?nZ132W?n[132Y?n]1"
        + "33S?n^133U?n_133W?n`1333?na133m?nb132R?nc132T?nd132V?ne132X?nf132Z?ng132[?nh132^?ni132`?nj132b?nk132"
        + "d?nl132f?nm132h?nn132j?no132l?np132n?o0132p?o11331?o21334?o31336?o41338?o500633:?o;133B?o<133E?o=133"
        + "H?o>133K?o?00533N?oD133T?oE133V?oF00633X?oL133`?oM133d?oN132I?oO132J?oP115P?oQ1140?oR1141?oS116Z?oT1"
        + "142?oU116]?oV116^?oW003143?oZ00616a?oa114J?ob003146?oe114Q?of00:149?p200615Q?p:00615W?pB00615^?pJ003"
        + "15d?pP102R?pQ102S?pR102]?pS200P0<4?pT102V?pU102U?pV122Y?pX12D2?pY00426@?p^12FP?p_12G;M@000J011M@J00J"
        + "01QM@e00J011MA>00701QMAF00B01YMAX00J011MB200J01QMBL1011MBN1013MBO1014MBR1017MBU101:MBV101;MBY00401>M"
        + "B_00801CMBg00401QMBl101VMBn00701XMC500;01aMC@00J011MCZ00J01QMD41011MD51012MD7004014MD=00801:MDF00701"
        + "CMDN00J01QMDi1011MDj1012MDl004014ME0005019ME6101?ME:00701CMEB00J01QME]00J011MF600J01QMFP00J011MFk00J"
        + "01QMGD00J011MG_00J01QMH800J011MHR00J01QMHm00J011MIF00J01QMIa00J011MJ:00J01QMJT104bMJU108hMJX00A0>AMJ"
        + "j10>HMJk0070>SMK11287MK200I0>bMKK1282MKL10>fMKM10>iMKN10>kMKO10?6MKP10?1MKQ10?0MKR00A0>AMKd10>HMKe00"
        + "70>SMKl1287MKm00I0>bMLE1282MLF10>fMLG10>iMLH10>kMLI10?6MLJ10?1MLK10?0MLL00A0>AML^10>HML_0070>SMLf128"
        + "7MLg00I0>bMM?1282MM@10>fMMA10>iMMB10>kMMC10?6MMD10?1MME10?0MMF00A0>AMMW10>HMMX0070>SMM`1287MMa00I0>b"
        + "MN91282MN:10>fMN;10>iMN<10>kMN=10?6MN>10?1MN?10?0MN@00A0>AMNQ10>HMNR0070>SMNY1287MNZ00I0>bMO31282MO4"
        + "10>fMO510>iMO610>kMO710?6MO810?1MO910?0MO:10?LMO;10?MMO>00:00aMOH00:00aMOR00:00aMO]00:00aMOg00:00aNi"
        + "010HWNi110HXNi210H]Ni310H`Ni510I8Ni610HcNi710H^Ni810HhNi910I:Ni:0040I3Ni>10HdNi?10HjNi@10I1NiA10HfNi"
        + "B10I2NiC10HbNiD10HeNiE10HZNiF10H[NiG10H_NiH10HaNiI10HgNiJ10HiNiK10HkNiL10I_NiM10JkNiN10JQNiO10I`NiQ1"
        + "0HXNiR10H]NiT10I7NiW10H^NiY10I:NiZ0040I3Ni_10HdNi`10HjNia10I1Nib10HfNic10I2Nie10HeNif10HZNig10H[Nih1"
        + "0H_Nij10HgNil10HkNj210H]Nj710H^Nj910I:Nj;10I4Nj=10I6Nj>10HdNj?10HjNjA10HfNjB10I2NjD10HeNjG10H_NjI10H"
        + "gNjK10HkNjM10JkNjO10I`NjQ10HXNjR10H]NjT10I7NjW10H^NjX10HhNjY10I:NjZ10I3Nj]10I5Nj^10I6Nj_10HdNj`10HjN"
        + "ja10I1Njb10HfNjc10I2Nje10HeNjf10HZNjg10H[Njh10H_Njj10HgNjk10HiNjl10HkNjm10I_Njo10JQNk010HWNk110HXNk2"
        + "10H]Nk310H`Nk410I7Nk510I8Nk610HcNk710H^Nk810HhNk910I:Nk;0030I4Nk>10HdNk?10HjNk@10I1NkA10HfNkB10I2NkC"
        + "10HbNkD10HeNkE10HZNkF10H[NkG10H_NkH10HaNkI10HgNkJ10HiNkK10HkNkQ10HXNkR10H]NkS10H`NkU10I8NkV10HcNkW10"
        + "H^NkX10HhNkY10I:Nk[0030I4Nk_10HdNk`10HjNka10I1Nkb10HfNkc10I2Nkd10HbNke10HeNkf10HZNkg10H[Nkh10H_Nki10"
        + "HaNkj10HgNkk10HiNkl10HkO40200a00_O41200a00]O42200b00]O43200c00]O44200d00]O45200e00]O46200f00]O47200g"
        + "00]O48200h00]O49200i00]O4:200j00]O4@300X01100YO4A300X01200YO4B300X01300YO4C300X01400YO4D300X01500YO4"
        + "E300X01600YO4F300X01700YO4G300X01800YO4H300X01900YO4I300X01:00YO4J300X01;00YO4K300X01<00YO4L300X01=0"
        + "0YO4M300X01>00YO4N300X01?00YO4O300X01@00YO4P300X01A00YO4Q300X01B00YO4R300X01C00YO4S300X01D00YO4T300X"
        + "01E00YO4U300X01F00YO4V300X01G00YO4W300X01H00YO4X300X01I00YO4Y300X01J00YO4Z330D01C30EO4[1013O4]101BO4"
        + "^2013014O4_201G01JO4a00J011O5:201801FO5;201=01FO5<201C014O5=201C01CO5>301@01@01FO5?201G013O5Z201=013"
        + "O5[201=014O5]201=01BO6@201401:O80231l31;O81232d32dO82132fO8@169;O8A15^GO8B15?<O8C1337O8D14k<O8E15TJO"
        + "8F18WSO8G15TYO8H14kTO8I16HPO8J174QO8K16FIO8L159=O8M15o<O8N156=O8O16FaO8P158MO8Q17f2O8R17DOO8S18cYO8T"
        + "15SaO8U15@jO8V16mDO8W16:EO8X16=EO8Y14i0O8Z14i9O8[191:O8]15hVO8^14i^O8_15?dO8`16<7O8a18faO8b169CO8c17"
        + "V1O8d17YkO8e15@8O8f16k0O8g16L9O8h16L8O8i17DdO8j159cO8k15FgO8l195=O90330D6L]30EO91330D4i930EO92330D4k"
        + "<30EO93330D5_930EO94330D72j30EO95330D69C30EO96330D7KG30EO97330D5;M30EO98330D6EG30EO9@15oGO9A15?`"
    var nfc = ""
        + "00m0<i29_00n0<i29P00o0<i29`0110<00300110<10310110<20320110<30330110<40400110<60420110<708V0110<80340"
        + "110<91kR0110<:0350110<<07=0110<?0800110<A0820110<S1kP0110<U1i00110<X0440120<71i20120<S1i40120<b1i601"
        + "30<10460130<20480130<704:0130<<04<0130<W0370140<71i:0140<<04>0140<S1i<0140<W1i@0140<^1iB0140<b1i>015"
        + "0<00380150<10390150<203:0150<31km0150<404B0150<604D0150<704F0150<803;0150<91kk0150<<04J0150<?0840150"
        + "<A0860150<S1ki0150<W08X0150<X04H0150<^1iH0150<a1iJ0160<71iN0170<107e0170<204L0170<41iP0170<604N0170<"
        + "704P0170<<07V0170<W04R0180<204T0180<71iR0180<81iV0180<<08N0180<S1iT0180<W1iX0180<_1iZ0190<003<0190<1"
        + "03=0190<203>0190<304X0190<404Z0190<604]0190<704a0190<803?0190<91l80190<<07?0190<?0880190<A08:0190<S1"
        + "l:0190<X04_0190<a1i]01:0<204e01;0<11ia01;0<<07X01;0<S1ic01;0<W04g01;0<b1ie01<0<104j01<0<<04n01<0<S1i"
        + "g01<0<W04l01<0<^1im01<0<b1ik01=0<11io01=0<71j001=0<S1j201>0<007i01>0<105301>0<303A01>0<71j401>0<<057"
        + "01>0<S1j601>0<W05501>0<^1j:01>0<b1j801?0<003B01?0<103C01?0<203D01?0<303E01?0<405<01?0<605>01?0<708_0"
        + "1?0<803F01?0<91l>01?0<;05@01?0<<07A01?0<?08<01?0<A08>01?0<K06P01?0<S1l<01?0<X07Z01@0<11jD01@0<71jF01"
        + "B0<105D01B0<71jH01B0<<05H01B0<?08@01B0<A08B01B0<S1jJ01B0<W05F01B0<b1jN01C0<105J01C0<205L01C0<71jP01C"
        + "0<<05P01C0<S1jR01C0<V08H01C0<W05N01D0<71jZ01D0<<05T01D0<S1j]01D0<V08J01D0<W05R01D0<^1ja01D0<b1j_01E0"
        + "<003I01E0<103J01E0<203K01E0<305X01E0<405Z01E0<605]01E0<803L01E0<91lV01E0<:05_01E0<;05a01E0<<07C01E0<"
        + "?08D01E0<A08F01E0<K06`01E0<S1lT01E0<T1jc01E0<X05c01E0<^1jg01E0<a1je01F0<31jm01F0<S1jo01G0<01k001G0<1"
        + "1k201G0<205e01G0<71k601G0<81k401G0<S1k801H0<71k:01H0<81k<01I0<01lc01I0<103M01I0<205g01I0<31li01I0<40"
        + "8c01I0<71k>01I0<805i01I0<91lg01I0<S1le01J0<105j01J0<21k@01J0<705l01J0<<05n01J0<S1kB01J0<b1kD01Q0<003"
        + "P01Q0<103Q01Q0<203R01Q0<303S01Q0<404101Q0<604301Q0<708W01Q0<803T01Q0<91kS01Q0<:03U01Q0<<07>01Q0<?081"
        + "01Q0<A08301Q0<S1kQ01Q0<U1i101Q0<X04501R0<71i301R0<S1i501R0<b1i701S0<104701S0<204901S0<704;01S0<<04=0"
        + "1S0<W03W01T0<71i;01T0<<04?01T0<S1i=01T0<W1iA01T0<^1iC01T0<b1i?01U0<003X01U0<103Y01U0<203Z01U0<31kn01"
        + "U0<404C01U0<604E01U0<704G01U0<803[01U0<91kl01U0<<04K01U0<?08501U0<A08701U0<S1kj01U0<W08Y01U0<X04I01U"
        + "0<^1iI01U0<a1iK01V0<71iO01W0<107f01W0<204M01W0<41iQ01W0<604O01W0<704Q01W0<<07W01W0<W04S01X0<204U01X0"
        + "<71iS01X0<81iW01X0<<08O01X0<S1iU01X0<W1iY01X0<_1i[01X0<b1kF01Y0<003]01Y0<103^01Y0<203_01Y0<304Y01Y0<"
        + "404[01Y0<604^01Y0<803`01Y0<91l901Y0<<07@01Y0<?08901Y0<A08;01Y0<S1l;01Y0<X04`01Y0<a1i^01Z0<204f01Z0<<"
        + "07a01[0<11ib01[0<<07Y01[0<S1id01[0<W04h01[0<b1if01]0<104k01]0<<04o01]0<S1ih01]0<W04m01]0<^1in01]0<b1"
        + "il01^0<11ip01^0<71j101^0<S1j301_0<007j01_0<105401_0<303b01_0<71j501_0<<05801_0<S1j701_0<W05601_0<^1j"
        + ";01_0<b1j901`0<003c01`0<103d01`0<203e01`0<303f01`0<405=01`0<605?01`0<708`01`0<803g01`0<91l?01`0<;05A"
        + "01`0<<07B01`0<?08=01`0<A08?01`0<K06Q01`0<S1l=01`0<X07[01a0<11jE01a0<71jG01c0<105E01c0<71jI01c0<<05I0"
        + "1c0<?08A01c0<A08C01c0<S1jK01c0<W05G01c0<b1jO01d0<105K01d0<205M01d0<71jQ01d0<<05Q01d0<S1jS01d0<V08I01"
        + "d0<W05O01e0<71j[01e0<81kG01e0<<05U01e0<S1j^01e0<V08K01e0<W05S01e0<^1jb01e0<b1j`01f0<003j01f0<103k01f"
        + "0<203l01f0<305Y01f0<405[01f0<605^01f0<803m01f0<91lW01f0<:05`01f0<;05b01f0<<07D01f0<?08E01f0<A08G01f0"
        + "<K06a01f0<S1lU01f0<T1jd01f0<X05d01f0<^1jh01f0<a1jf01g0<31jn01g0<S1jp01h0<01k101h0<11k301h0<205f01h0<"
        + "71k701h0<81k501h0<:1kH01h0<S1k901i0<71k;01i0<81k=01j0<01ld01j0<103n01j0<205h01j0<31lj01j0<408d01j0<7"
        + "1k?01j0<803p01j0<91lh01j0<:1kI01j0<S1lf01k0<105k01k0<21kA01k0<705m01k0<<05o01k0<S1kC01k0<b1kE02X0<01"
        + "p^02X0<10>502X0=21p10320<01kV0320<11kT0320<31kZ0320<91kX0340<407N0350<107k0360<107m0360<407R0370<11i"
        + "803:0<01l003:0<11ko03:0<31l403:0<91l203?0<11i_03D0<01lB03D0<11l@03D0<31lF03D0<91lD03E0<11j<03E0<408]"
        + "03E0<81j>03F0<408Z03H0<107o03L0<007K03L0<107G03L0<407E03L0<<07I03R0<01kW03R0<11kU03R0<31k[03R0<91kY0"
        + "3T0<407O03U0<107l03V0<107n03V0<407S03W0<11i903Z0<01l103Z0<11kp03Z0<31l503Z0<91l303`0<11i`03e0<01lC03"
        + "e0<11lA03e0<31lG03e0<91lE03f0<11j=03f0<408^03f0<81j?03g0<408[03i0<107p03m0<007L03m0<107H03m0<407F03m"
        + "0<<07J0420<01ka0420<11k_0420<31ke0420<91kc0430<01kb0430<11k`0430<31kf0430<91kd04B0<01iD04B0<11iF04C0"
        + "<01iE04C0<11iG05<0<01j@05<0<11jB05=0<01jA05=0<11jC05J0<71jT05K0<71jU05P0<71jV05Q0<71jW05X0<11ji05Y0<"
        + "11jj05Z0<81jk05[0<81jl05p0<71kK06P0<01lL06P0<11lJ06P0<31lP06P0<91lN06P0<S1lR06Q0<01lM06Q0<11lK06Q0<3"
        + "1lQ06Q0<91lO06Q0<S1lS06`0<01lZ06`0<11lX06`0<31l_06`0<91l]06`0<S1la06a0<01l[06a0<11lY06a0<31l`06a0<91"
        + "l^06a0<S1lb06h0<<07_07Z0<407]07[0<407^08V0<407P08W0<407Q08X0<61iL08Y0<61iM08_0<408a08`0<408b0:B0<<07"
        + "`0>A0<01ok0>A0<10>60>A0<41oj0>A0<61oi0>A0<C1m80>A0<D1m90>A0=51om0>E0<01p80>E0<10>80>E0<C1mH0>E0<D1mI"
        + "0>G0<01p:0>G0<10>90>G0<C1mX0>G0<D1mY0>G0=51p<0>I0<01pJ0>I0<10>:0>I0<41pI0>I0<61pH0>I0<80>Z0>I0<C1mi0"
        + ">I0<D1mj0>O0<01pi0>O0<10><0>O0<C1n80>O0<D1n90>Q0<D1p]0>U0<01pZ0>U0<10>>0>U0<41pY0>U0<61pX0>U0<80>[0>"
        + "U0<D1nI0>Y0<01pk0>Y0<10>?0>Y0<C1nX0>Y0<D1nY0>Y0=51pm0>]0=51oe0>_0=51p40>b0<01na0>b0<10>]0>b0<41ob0>b"
        + "0<61oa0>b0<C1m00>b0<D1m10>b0=21og0>b0=51od0>f0<01nc0>f0<10>^0>f0<C1m@0>f0<D1mA0>h0<01ne0>h0<10>_0>h0"
        + "<C1mP0>h0<D1mQ0>h0=21p60>h0=51p30>j0<01ng0>j0<10>`0>j0<41pA0>j0<61p@0>j0<80?:0>j0<C1ma0>j0<D1mb0>j0="
        + "21pF0>p0<01ni0>p0<10?<0>p0<C1n00>p0<D1n10?10<C1pT0?10<D1pU0?50<01nk0?50<10?=0?50<41pQ0?50<61pP0?50<8"
        + "0?;0?50<C1n@0?50<D1nA0?50=21pV0?90<01nm0?90<10?>0?90<C1nP0?90<D1nQ0?90=21pg0?90=51pd0?:0<01pB0?:0<10"
        + ">@0?:0=21pG0?;0<01pR0?;0<10>a0?;0=21pW0?>0=51pe0?B0<10?C0?B0<80?D0@60<80@70@@0<60C@0@@0<80CB0@C0<10@"
        + "30@E0<00@00@E0<60CF0@E0<80@10@F0<60C10@F0<80CL0@G0<80CN0@H0<00@=0@H0<40CR0@H0<60@I0@H0<80CT0@J0<10@<"
        + "0@N0<80CV0@S0<40C_0@S0<60@>0@S0<80Ca0@S0<;0Cc0@W0<80Ce0@[0<80Ci0@^0<80C]0@a0<60CA0@a0<80CC0@d0<10AC0"
        + "@f0<00A@0@f0<60CG0@f0<80AA0@g0<60C20@g0<80CM0@h0<80CO0@i0<00AM0@i0<40CS0@i0<60@j0@i0<80CU0@k0<10AL0@"
        + "o0<80CW0A30<40C`0A30<60AN0A30<80Cb0A30<;0Cd0A70<80Cf0A;0<80Cj0A=0<80C^0AF0<80AG0Ae0<?0Ag0Af0<?0Ah0CH"
        + "0<80CJ0CI0<80CK0CX0<80CZ0CY0<80C[0HW0IC0HR0HW0ID0HS0HW0IE0HU0I80ID0HT0I:0ID0HV0K10ID0K20KB0ID0KC0KE0"
        + "ID0K00TX0Tm0TY0Ta0Tm0Tb0Td0Tm0Te0W70Vo0W;0W70WG0W<0^70]o0^;0^70^F0^80^70^G0^<0_B0`G0_D0`60_o0`:0`60`"
        + "G0`<0`70_o0`;0b60bF0b80cp0dE0d00d60d20d:0d60dE0d70d60dF0d80d:0dE0d;0f60eo0f:0f60fG0f<0f70eo0f;0hI0h:"
        + "0hJ0hI0h?0hL0hI0hO0hN0hL0h:0hM10U10_10V1]51]f1]61]71]f1]81]91]f1]:1];1]f1]<1]=1]f1]>1]A1]f1]B1]k1]f1"
        + "]l1]m1]f1]n1]o1]f1^01]p1]f1^11^21]f1^31ig0<41ii1ih0<41ij1jJ0<41jL1jK0<41jM1jR0<71jX1jS0<71jY1kP0<21k"
        + "]1kP0<61kg1kQ0<21k^1kQ0<61kh1ki0<21l61kj0<21l71l<0<21lH1l=0<21lI1m00<01m21m00<11m41m00=21m61m00=51o0"
        + "1m10<01m31m10<11m51m10=21m71m10=51o11m20=51o21m30=51o31m40=51o41m50=51o51m60=51o61m70=51o71m80<01m:1"
        + "m80<11m<1m80=21m>1m80=51o81m90<01m;1m90<11m=1m90=21m?1m90=51o91m:0=51o:1m;0=51o;1m<0=51o<1m=0=51o=1m"
        + ">0=51o>1m?0=51o?1m@0<01mB1m@0<11mD1mA0<01mC1mA0<11mE1mH0<01mJ1mH0<11mL1mI0<01mK1mI0<11mM1mP0<01mR1mP"
        + "0<11mT1mP0=21mV1mP0=51o@1mQ0<01mS1mQ0<11mU1mQ0=21mW1mQ0=51oA1mR0=51oB1mS0=51oC1mT0=51oD1mU0=51oE1mV0"
        + "=51oF1mW0=51oG1mX0<01mZ1mX0<11m]1mX0=21m_1mX0=51oH1mY0<01m[1mY0<11m^1mY0=21m`1mY0=51oI1mZ0=51oJ1m[0="
        + "51oK1m]0=51oL1m^0=51oM1m_0=51oN1m`0=51oO1ma0<01mc1ma0<11me1ma0=21mg1mb0<01md1mb0<11mf1mb0=21mh1mi0<0"
        + "1mk1mi0<11mm1mi0=21mo1mj0<01ml1mj0<11mn1mj0=21mp1n00<01n21n00<11n41n10<01n31n10<11n51n80<01n:1n80<11"
        + "n<1n90<01n;1n90<11n=1n@0<01nB1n@0<11nD1n@0=21nF1nA0<01nC1nA0<11nE1nA0=21nG1nI0<01nK1nI0<11nM1nI0=21n"
        + "O1nP0<01nR1nP0<11nT1nP0=21nV1nP0=51oP1nQ0<01nS1nQ0<11nU1nQ0=21nW1nQ0=51oQ1nR0=51oR1nS0=51oS1nT0=51oT"
        + "1nU0=51oU1nV0=51oV1nW0=51oW1nX0<01nZ1nX0<11n]1nX0=21n_1nX0=51oX1nY0<01n[1nY0<11n^1nY0=21n`1nY0=51oY1"
        + "nZ0=51oZ1n[0=51o[1n]0=51o]1n^0=51o^1n_0=51o_1n`0=51o`1na0=51oc1ne0=51p21nm0=51pc1og0=51oh1op0<01p=1o"
        + "p0<11p>1op0=21p?1p60=51p71pg0=51ph1po0<01pM1po0<11pN1po0=21pO26@0<i26J26B0<i26K26D0<i26_27@0<i27=27B"
        + "0<i27?27D0<i27>2830<i2842880<i28928;0<i28<28S0<i28T28U0<i28V28m0<i2912930<i2942950<i2972980<i29929=0"
        + "<i29^29Q0<i29R29T0<i29a29U0<i29b29c0<i29e29d0<i29f29g0<i29i29h0<i29j29k0<i2:029l0<i2:129m0<i2;P29n0<"
        + "i2;Q2:20<i2:42:30<i2:52:60<i2:82:70<i2:92:A0<i2;R2:B0<i2;S2:R0<i2:]2:X0<i2:^2:Y0<i2:_2:[0<i2:`2:c0<i"
        + "2;Z2:d0<i2;[2:e0<i2;]2:f0<i2;^31632I32D31;32I31<31=32I31>31?32I31@31A32I31B31C32I31D31E32I31F31G32I3"
        + "1H31I32I31J31K32I31L31M32I31N31O32I31P31Q32I31R31T32I31U31V32I31W31X32I31Y31`32I31a31`32J31b31c32I31"
        + "d31c32J31e31f32I31g31f32J31h31i32I31j31i32J31k31l32I31m31l32J31n32M32I32N32V32I33e32[32I32]32^32I32_"
        + "32`32I32a32b32I32c32d32I32e32f32I32g32h32I32i32j32I32k32l32I32m32n32I32o32p32I33033132I33233432I3353"
        + "3632I33733832I33933?32I33@33?32J33A33B32I33C33B32J33D33E32I33F33E32J33G33H32I33I33H32J33J33K32I33L33"
        + "K32J33M33`32I33h33a32I33i33b32I33j33c32I33k33n32I33oA2IA2kA2JA2KA2kA2LA2UA2kA2[A4bA4WA4_A4cA4WA4`A=7"
        + "A<oA=;A=7A=GA=<ABjABaABmABjABkABlABjABnABoAFiAF`AFkAFjAF`AFl"
    rbsNormTabs = {ccc: ccc, nfc: nfc, dk: [], dv: [], kk: [], kv: []}
    rbsNormParse(nfd, rbsNormTabs.dk, rbsNormTabs.dv)
    rbsNormParse(nfkd, rbsNormTabs.kk, rbsNormTabs.kv)
    return rbsNormTabs
}
// The number in the n digits of t at i.
function rbsNormNum(t, i, n) {
    var v = 0
    var k = 0
    while (k < n) {
        v = v * 64 + rbsGcbDig(t, i + k)
        k = k + 1
    }
    return v
}
// The nfd / nfkd records of t as sorted keys and their code point lists.
function rbsNormParse(t, keys, vals) {
    var i = 0
    while (i < t.length) {
        var key = rbsNormNum(t, i, 3)
        var n = rbsNormNum(t, i + 3, 1)
        if (n == 0) {
            var cnt = rbsNormNum(t, i + 4, 2)
            var tgt = rbsNormNum(t, i + 6, 3)
            var r = 0
            while (r < cnt) {
                keys.push(key + r)
                vals.push([tgt + r])
                r = r + 1
            }
            i = i + 9
            continue
        }
        var v = []
        var k = 0
        while (k < n) {
            v.push(rbsNormNum(t, i + 4 + k * 3, 3))
            k = k + 1
        }
        keys.push(key)
        vals.push(v)
        i = i + 4 + n * 3
    }
}
// The index of cp in the sorted keys, or -1.
function rbsNormFind(keys, cp) {
    var lo = 0
    var hi = keys.length - 1
    while (lo <= hi) {
        var mid = Math.floor((lo + hi) / 2)
        if (keys[mid] == cp) { return mid }
        if (keys[mid] < cp) { lo = mid + 1 } else { hi = mid - 1 }
    }
    return 0 - 1
}
// The canonical combining class of cp.
function rbsCcc(cp) {
    var t = rbsNormTab().ccc
    var lo = 0
    var hi = t.length / 5 - 1
    while (lo < hi) {
        var mid = Math.floor((lo + hi + 1) / 2)
        if (rbsNormNum(t, mid * 5, 3) <= cp) { lo = mid } else { hi = mid - 1 }
    }
    return rbsNormNum(t, lo * 5 + 3, 2)
}
// Appends to out the full decomposition of cp: canonical, or compatibility when
// compat. A Hangul syllable decomposes by arithmetic.
function rbsNormDecomp(cp, compat, out) {
    var tb = rbsNormTab()
    var s = cp - 44032
    if (s >= 0 && s < 11172) {
        out.push(4352 + Math.floor(s / 588))
        out.push(4449 + Math.floor((s % 588) / 28))
        if (s % 28 != 0) { out.push(4519 + s % 28) }
        return
    }
    var k = compat ? rbsNormFind(tb.kk, cp) : 0 - 1
    if (k >= 0) {
        var j = 0
        while (j < tb.kv[k].length) {
            rbsNormDecomp(tb.kv[k][j], false, out)
            j = j + 1
        }
        return
    }
    var d = rbsNormFind(tb.dk, cp)
    if (d < 0) {
        out.push(cp)
        return
    }
    var m = 0
    while (m < tb.dv[d].length) {
        out.push(tb.dv[d][m])
        m = m + 1
    }
}
// The primary composite of a followed by b, or -1. A Hangul L V or LV T pair
// composes by arithmetic.
function rbsNormCompose(a, b) {
    if (a >= 4352 && a < 4371 && b >= 4449 && b < 4470) { return 44032 + ((a - 4352) * 21 + (b - 4449)) * 28 }
    var s = a - 44032
    if (s >= 0 && s < 11172 && s % 28 == 0 && b > 4519 && b < 4547) { return a + (b - 4519) }
    var t = rbsNormTab().nfc
    var lo = 0
    var hi = t.length / 9 - 1
    while (lo <= hi) {
        var mid = Math.floor((lo + hi) / 2)
        var ma = rbsNormNum(t, mid * 9, 3)
        var mb = rbsNormNum(t, mid * 9 + 3, 3)
        if (ma == a && mb == b) { return rbsNormNum(t, mid * 9 + 6, 3) }
        if (ma < a || (ma == a && mb < b)) { lo = mid + 1 } else { hi = mid - 1 }
    }
    return 0 - 1
}
// String#unicode_normalize of the text s to form "nfc", "nfd", "nfkc" or
// "nfkd": the full decomposition, the canonical ordering of each run of
// non-starters, and for nfc / nfkc the canonical composition. Go twin:
// rubyNormalize in abnf/jsrtrubynorm.go.
function rbsNormalize(s, form) {
    var plain = true
    var i = 0
    while (i < s.length && plain) {
        plain = s.charCodeAt(i) < 128
        i = i + 1
    }
    if (plain) { return s }
    var cps = []
    var compat = form == "nfkc" || form == "nfkd"
    i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        var w = rbsCharLen(s, i)
        if (w == 2) { c = (c - 55296) * 1024 + (s.charCodeAt(i + 1) - 56320) + 65536 }
        rbsNormDecomp(c, compat, cps)
        i = i + w
    }
    var cl = []
    var k = 0
    while (k < cps.length) {
        cl.push(rbsCcc(cps[k]))
        var j = k
        while (j > 0 && cl[j] != 0 && cl[j - 1] > cl[j]) {
            var tc = cps[j]
            cps[j] = cps[j - 1]
            cps[j - 1] = tc
            var tk = cl[j]
            cl[j] = cl[j - 1]
            cl[j - 1] = tk
            j = j - 1
        }
        k = k + 1
    }
    var out = cps
    if (form == "nfc" || form == "nfkc") {
        out = []
        var st = 0 - 1
        var last = 0 - 1
        var q = 0
        while (q < cps.length) {
            var cc = cl[q]
            var comp = st >= 0 && last < cc ? rbsNormCompose(out[st], cps[q]) : 0 - 1
            if (comp >= 0) {
                out[st] = comp
            } else {
                if (cc == 0) {
                    st = out.length
                    last = 0 - 1
                } else {
                    last = cc
                }
                out.push(cps[q])
            }
            q = q + 1
        }
    }
    var r = ""
    var e = 0
    while (e < out.length) {
        r = r + rbsChrUtf8(out[e], [])
        e = e + 1
    }
    return r
}

// The index of character ch in the list, or -1.
function rbsListIdx(list, ch) {
    var i = 0
    while (i < list.length) {
        if (list[i] == ch) { return i }
        i = i + 1
    }
    return -1
}

// MRI's character-set notation, shared by tr / tr_s / count / delete / squeeze
// (tr_setup_table in string.c): a leading ^ negates, `a-z` is a range of code
// points, and a backslash escapes the next character. Answers {neg, chars} with
// every range expanded, one character per entry.
function rbsSetExpand(spec) {
    var cs = rbsChars(spec)
    var neg = false
    var i = 0
    if (cs.length > 1 && cs[0] == "^") { neg = true; i = 1 }
    var chars = []
    while (i < cs.length) {
        if (cs[i] == "\\" && i + 1 < cs.length) {
            chars.push(cs[i + 1])
            i = i + 2
            continue
        }
        if (i + 2 < cs.length && cs[i + 1] == "-") {
            var k = rbsOrd(cs[i])
            var b = rbsOrd(cs[i + 2])
            while (k <= b) {
                if (k < 65536) { chars.push(String.fromCharCode(k)) }
                else { chars.push(String.fromCharCode(55296 + Math.floor((k - 65536) / 1024), 56320 + ((k - 65536) % 1024))) }
                k = k + 1
            }
            i = i + 3
            continue
        }
        chars.push(cs[i])
        i = i + 1
    }
    return {neg: neg, chars: chars}
}

// Is ch in EVERY one of the expanded sets? MRI intersects the sets of count /
// delete / squeeze when more than one is given.
function rbsInSets(sets, ch) {
    var i = 0
    while (i < sets.length) {
        var found = rbsListIdx(sets[i].chars, ch) >= 0
        if (sets[i].neg) { found = !found }
        if (!found) { return false }
        i = i + 1
    }
    return sets.length > 0
}
function rbsSetsOf(specs) {
    var out = []
    var i = 0
    while (i < specs.length) {
        out.push(rbsSetExpand(specs[i]))
        i = i + 1
    }
    return out
}

// String#tr / #tr_s. An empty `to` DELETES; a shorter one repeats its last
// character; a negated `from` maps everything outside the set to that last
// character. tr_s additionally squeezes the runs it produced.
function rbsTr(s, from, to, sq) {
    var f = rbsSetExpand(from)
    var t = rbsSetExpand(to)
    var tc = t.chars
    var last = tc.length > 0 ? tc[tc.length - 1] : ""
    var cs = rbsChars(s)
    var out = ""
    var prev = ""
    var i = 0
    while (i < cs.length) {
        var ch = cs[i]
        i = i + 1
        var k = rbsListIdx(f.chars, ch)
        var rep = ch
        var did = false
        if (f.neg) {
            if (k < 0) { rep = last; did = true }
        } else if (k >= 0) {
            if (tc.length == 0) { rep = "" } else if (k >= tc.length) { rep = last } else { rep = tc[k] }
            did = true
        }
        if (rep == "") { prev = ""; continue }
        if (sq && did && rep == prev) { continue }
        out = out + rep
        prev = did ? rep : ""
    }
    return out
}

// String#squeeze: collapse a run of the same character, restricted to the sets
// when any are given.
function rbsSqueeze(s, specs) {
    var sets = rbsSetsOf(specs)
    var cs = rbsChars(s)
    var out = ""
    var prev = ""
    var i = 0
    while (i < cs.length) {
        var ch = cs[i]
        i = i + 1
        if (ch == prev && (sets.length == 0 || rbsInSets(sets, ch))) { continue }
        out = out + ch
        prev = ch
    }
    return out
}

// String#count / #delete: the characters in every set, counted or removed.
function rbsCount(s, specs) {
    var sets = rbsSetsOf(specs)
    var cs = rbsChars(s)
    var n = 0
    var i = 0
    while (i < cs.length) {
        if (rbsInSets(sets, cs[i])) { n = n + 1 }
        i = i + 1
    }
    return n
}
function rbsDelete(s, specs) {
    var sets = rbsSetsOf(specs)
    var cs = rbsChars(s)
    var out = ""
    var i = 0
    while (i < cs.length) {
        if (!rbsInSets(sets, cs[i])) { out = out + cs[i] }
        i = i + 1
    }
    return out
}

// String#reverse, a surrogate pair kept whole.
function rbsReverse(s) {
    var out = ""
    var i = 0
    while (i < s.length) {
        var k = rbsCharLen(s, i)
        out = s.substring(i, i + k) + out
        i = i + k
    }
    return out
}

// String#center / #ljust / #rjust. mode 0 centres, 1 pads on the right, 2 on the
// left. MRI counts CHARACTERS, and a width at or below the receiver's length
// answers the receiver.
function rbsPad(s, width, pad, mode) {
    var n = rbsLen(s)
    if (pad.length == 0 || width <= n) { return s }
    var total = width - n
    var left = 0
    if (mode == 0) { left = Math.floor(total / 2) } else if (mode == 2) { left = total }
    return rbsRepeatTo(pad, left) + s + rbsRepeatTo(pad, total - left)
}

// The first n characters of pad repeated.
function rbsRepeatTo(pad, n) {
    var out = ""
    var i = 0
    var c = 0
    while (c < n) {
        if (i >= pad.length) { i = 0 }
        var k = rbsCharLen(pad, i)
        out = out + pad.substring(i, i + k)
        i = i + k
        c = c + 1
    }
    return out
}

// Where String#index starts searching for character position pos (negative
// counts from the end): a unit index, or -1 when pos is outside the string.
function rbsIndexStart(s, pos) {
    var n = rbsCharIdx(s, s.length)
    var p = pos < 0 ? pos + n : pos
    if (p < 0 || p > n) { return -1 }
    return rbsUnitIdx(s, p)
}
// Where String#rindex searches back from: pos is clamped to the end, and -1 says
// a negative pos reached before the start.
function rbsRindexStart(s, pos, given) {
    var n = rbsCharIdx(s, s.length)
    var p = given ? pos : n
    if (p < 0) { p = p + n }
    if (p < 0) { return -1 }
    if (p > n) { p = n }
    return rbsUnitIdx(s, p)
}
// The last unit index <= from where t starts, or -1.
function rbsRfind(s, t, from) {
    var u = from
    if (u + t.length > s.length) { u = s.length - t.length }
    while (u >= 0) {
        if (s.substring(u, u + t.length) == t) { return u }
        u = u - 1
    }
    return -1
}
// The last unit index <= from where re matches, or -1 (rb_reg_search backward).
function rbsRfindRx(re, s, from) {
    var u = from
    while (u >= 0) {
        var m = rxMatchAt(re, s, u)
        if (m !== null && m.begin >= 0) { return u }
        u = u - 1
    }
    return -1
}

// MRI's ISALNUM classes for String#succ: 1 a digit, 2 a letter, 0 neither.
function rbsSuccKind(c) {
    if (c >= 48 && c <= 57) { return 1 }
    if ((c >= 97 && c <= 122) || (c >= 65 && c <= 90)) { return 2 }
    return 0
}
// String#succ (str_succ): the rightmost alphanumeric is incremented, carrying
// leftwards through alphanumerics ("az" -> "ba", "zz" -> "aaa") and across
// non-alphanumerics ("1.9" -> "2.0") unless the next one is of the other kind
// ("a-9" -> "a-10"). A string without alphanumerics increments its last
// character within its UTF-8 length (enc_succ_char): past the largest one of
// that length it wraps to the smallest and the carry moves one character left,
// and a string the carry runs off gains "\x01" in front ("\x7f" -> "\x01\x00").
function rbsSucc(s) {
    if (s.length == 0) { return s }
    var out = s
    var i = s.length - 1
    var skipped = false
    var lastKind = 0
    var carryAt = -1
    var carry = ""
    while (i >= 0) {
        var c = s.charCodeAt(i)
        var k = rbsSuccKind(c)
        if (skipped && lastKind != 0 && k != 0 && k != lastKind) { break }
        if (k == 0) {
            skipped = true
            i = i - 1
            continue
        }
        skipped = false
        var w = ""
        if (c == 57) {
            w = "0"
            carry = "1"
        } else if (c == 122) {
            w = "a"
            carry = "a"
        } else if (c == 90) {
            w = "A"
            carry = "A"
        } else {
            return out.substring(0, i) + String.fromCharCode(c + 1) + out.substring(i + 1, out.length)
        }
        out = out.substring(0, i) + w + out.substring(i + 1, out.length)
        lastKind = k
        carryAt = i
        i = i - 1
    }
    if (carryAt >= 0) { return out.substring(0, carryAt) + carry + out.substring(carryAt, out.length) }
    var tail = ""
    var j = s.length
    var err = []
    while (j > 0) {
        var at = j - 1
        var cp = s.charCodeAt(at)
        if (at > 0 && cp >= 56320 && cp <= 57343 && rbsCharLen(s, at - 1) == 2) {
            at = at - 1
            cp = (s.charCodeAt(at) - 55296) * 1024 + (cp - 56320) + 65536
        }
        var wrap = rbsSuccWrap(cp)
        if (wrap < 0) { return s.substring(0, at) + rbsChrUtf8(cp == 55295 ? 57344 : cp + 1, err) + tail }
        tail = rbsChrUtf8(wrap, err) + tail
        j = at
    }
    return "\x01" + tail
}
// The smallest code point of cp's UTF-8 length when cp is the largest, else -1.
function rbsSuccWrap(cp) {
    if (cp == 127) { return 0 }
    if (cp == 2047) { return 128 }
    if (cp == 65535) { return 2048 }
    if (cp == 1114111) { return 65536 }
    return 0 - 1
}

// The text of a sub / gsub replacement template for match m of re in s
// (rb_reg_regsub): \0..\9 and \k<name> a group, \& the match, \` the text before
// it, \' the text after it, \+ the last group that matched, \\ one backslash; a
// backslash before anything else stays as written.
function rbsExpand(re, s, m, repl) {
    var out = ""
    var i = 0
    while (i < repl.length) {
        var c = repl.charCodeAt(i)
        if (c != 92 || i + 1 >= repl.length) {
            out = out + repl.charAt(i)
            i = i + 1
            continue
        }
        var d = repl.charCodeAt(i + 1)
        var g = -1
        if (d >= 48 && d <= 57) {
            g = d - 48
        } else if (d == 38) {
            g = 0
        } else if (d == 43) {
            g = re.ngroups
            while (g > 0 && m.caps[2 * g] < 0) { g = g - 1 }
        } else if (d == 96) {
            out = out + s.substring(0, m.begin)
        } else if (d == 39) {
            out = out + s.substring(m.end, s.length)
        } else if (d == 92) {
            out = out + "\\"
        } else if (d == 107 && i + 2 < repl.length && repl.charCodeAt(i + 2) == 60 && rbsFind(repl, ">", i + 3) >= 0) {
            var j = rbsFind(repl, ">", i + 3)
            g = rxIndexOfName(re, repl.substring(i + 3, j))
            i = j - 1
        } else {
            out = out + repl.substring(i, i + 2)
        }
        if (g >= 0) {
            var t = rxGroup(re, s, m, g)
            if (t !== null) { out = out + t }
        }
        i = i + 2
    }
    return out
}

// ----- Binary (ASCII-8BIT) Strings -----
// A binary String's text holds one char per byte, U+0000..U+00FF. Go twins in
// abnf/jsrtrubystr.go.

// The UTF-8 bytes of s, one char each: String#b.
function rbsBinOf(s) {
    var out = ""
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (rbsCharLen(s, i) == 2) {
            c = (c - 55296) * 1024 + (s.charCodeAt(i + 1) - 56320) + 65536
            i = i + 1
        }
        i = i + 1
        if (c < 128) { out = out + String.fromCharCode(c) }
        else if (c < 2048) { out = out + String.fromCharCode(192 + Math.floor(c / 64), 128 + (c % 64)) }
        else if (c < 65536) {
            out = out + String.fromCharCode(224 + Math.floor(c / 4096), 128 + (Math.floor(c / 64) % 64), 128 + (c % 64))
        } else {
            out = out + String.fromCharCode(240 + Math.floor(c / 262144), 128 + (Math.floor(c / 4096) % 64),
                                            128 + (Math.floor(c / 64) % 64), 128 + (c % 64))
        }
    }
    return out
}

// String#setbyte over the bytes b: byte i (counted from the end when negative)
// becomes v mod 256. Answers null when i is outside b. Go twin: rubySetByte in
// abnf/jsrtrubystr.go.
function rbsSetByte(b, i, v) {
    var k = i < 0 ? i + b.length : i
    if (k < 0 || k >= b.length) { return null }
    var m = v % 256
    if (m < 0) { m = m + 256 }
    return b.substring(0, k) + String.fromCharCode(m) + b.substring(k + 1)
}

// The text a binary String's bytes spell as UTF-8, or null when they are not
// valid UTF-8 (a stray continuation byte, an overlong form, a surrogate, past
// U+10FFFF, or a sequence cut short).
function rbsBinDecode(b) {
    var out = ""
    var i = 0
    var err = []
    while (i < b.length) {
        var n = rbsU8At(b, i)
        if (n == 0) { return null }
        out = out + (n == 1 ? b.charAt(i) : rbsChrUtf8(rbsU8Cp(b, i, n), err))
        i = i + n
    }
    return out
}

// A string literal whose \xHH or \nnn escapes past 127 are BYTES (raw lists
// where each landed in out, one char each): the literal is read as UTF-8 bytes,
// which spell its text when they are valid ("\xC3\xA9" is "é"). Invalid ones
// leave out as it is, and inv, when given, receives the bytes for a String
// literal to box.
function rbsEscBytes(out, raw, inv) {
    var b = ""
    var r = 0
    var i = 0
    while (i < out.length) {
        if (r < raw.length && raw[r] == i) {
            b = b + out.charAt(i)
            r = r + 1
            i = i + 1
        } else {
            var k = rbsCharLen(out, i)
            b = b + rbsBinOf(out.substring(i, i + k))
            i = i + k
        }
    }
    var t = rbsBinDecode(b)
    if (t !== null) { return t }
    if (inv !== undefined && inv !== null) { inv.push(b) }
    return out
}

// The byte length of the UTF-8 character at byte i of b, 0 when no valid one
// starts there.
function rbsU8At(b, i) {
    var c = b.charCodeAt(i)
    if (c < 128) { return 1 }
    var n = 0
    if (c >= 194 && c <= 223) { n = 2 }
    else if (c >= 224 && c <= 239) { n = 3 }
    else if (c >= 240 && c <= 244) { n = 4 }
    else { return 0 }
    if (i + n > b.length) { return 0 }
    var k = 1
    while (k < n) {
        var d = b.charCodeAt(i + k)
        if (d < 128 || d > 191) { return 0 }
        k = k + 1
    }
    var cp = rbsU8Cp(b, i, n)
    if ((n == 3 && (cp < 2048 || (cp >= 55296 && cp <= 57343))) || (n == 4 && (cp < 65536 || cp > 1114111))) { return 0 }
    return n
}

// The code point of the n-byte UTF-8 character at byte i of b.
function rbsU8Cp(b, i, n) {
    var cp = b.charCodeAt(i) % (n == 2 ? 32 : (n == 3 ? 16 : 8))
    var k = 1
    while (k < n) {
        cp = cp * 64 + (b.charCodeAt(i + k) - 128)
        k = k + 1
    }
    return cp
}

// The characters of UTF-8 bytes b where a byte that starts no valid character
// counts as one: String#size of a String with an invalid byte.
function rbsU8Len(b) {
    var n = 0
    var i = 0
    while (i < b.length) {
        var k = rbsU8At(b, i)
        i = i + (k == 0 ? 1 : k)
        n = n + 1
    }
    return n
}

// The code points of the bytes b of a String in encoding enc up to its first
// invalid UTF-8 sequence, where err receives MRI's message: String#codepoints /
// #each_codepoint of a box. Go twin: rubyBinCodepoints in abnf/jsrtrubystr.go.
function rbsBinCodepoints(b, enc, err) {
    var out = []
    var i = 0
    while (i < b.length) {
        var k = 1
        var cp = b.charCodeAt(i)
        if (enc == "UTF-8") {
            k = rbsU8At(b, i)
            if (k == 0) {
                err.push("invalid byte sequence in UTF-8")
                return out
            }
            if (k > 1) { cp = rbsU8Cp(b, i, k) }
        }
        out.push(cp)
        i = i + k
    }
    return out
}

// Whether every char of s is ASCII.
function rbsAsciiOnly(s) {
    var i = 0
    while (i < s.length) {
        if (s.charCodeAt(i) > 127) { return false }
        i = i + 1
    }
    return true
}

// upcase / downcase / swapcase / capitalize (name) of the bytes b of a String
// that is not UTF-8: MRI maps the ASCII letters only, every other byte stays.
function rbsAsciiCase(b, name) {
    var out = ""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var up = name == "upcase" || (name == "capitalize" && i == 0)
        var down = name == "downcase" || (name == "capitalize" && i > 0)
        if ((up || name == "swapcase") && c >= 97 && c <= 122) { c = c - 32 }
        else if ((down || name == "swapcase") && c >= 65 && c <= 90) { c = c + 32 }
        out = out + String.fromCharCode(c)
        i = i + 1
    }
    return out
}

// A binary String's #inspect: printable ASCII as itself, the named escapes, and
// every other byte as \xHH.
function rbsBinInspect(b) { return rbsDump(b, false) }

// String#dump of the bytes b: rbsBinInspect's form, where u8 (a UTF-8 String)
// writes each valid multi-byte character as \uXXXX, or \u{X} past U+FFFF. Go
// twin: rubyDump in abnf/jsrtrubystr.go.
function rbsDump(b, u8) {
    var hx = "0123456789ABCDEF"
    var out = "\""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var nx = i + 1 < b.length ? b.charCodeAt(i + 1) : 0
        var n = u8 && c >= 128 ? rbsU8At(b, i) : 0
        if (n > 1) {
            var cp = rbsHexUp(rbsU8Cp(b, i, n))
            out = out + (cp.length > 4 ? "\\u{" + cp + "}" : "\\u" + "0000".substring(cp.length) + cp)
            i = i + n
            continue
        }
        if (c == 34) { out = out + "\\\"" }
        else if (c == 92) { out = out + "\\\\" }
        else if (c == 10) { out = out + "\\n" }
        else if (c == 9) { out = out + "\\t" }
        else if (c == 13) { out = out + "\\r" }
        else if (c == 12) { out = out + "\\f" }
        else if (c == 11) { out = out + "\\v" }
        else if (c == 8) { out = out + "\\b" }
        else if (c == 7) { out = out + "\\a" }
        else if (c == 27) { out = out + "\\e" }
        else if (c == 35 && (nx == 123 || nx == 36 || nx == 64)) { out = out + "\\#" }
        else if (c < 32 || c >= 127) { out = out + "\\x" + hx.charAt(Math.floor(c / 16)) + hx.charAt(c % 16) }
        else { out = out + b.charAt(i) }
        i = i + 1
    }
    return out + "\""
}

// The encoding a name selects (case does not matter): ASCII-8BIT, US-ASCII or
// UTF-8, or "" for any other. Go twin: rubyEncCanon in abnf/jsrtrubystr.go.
function rbsEncCanon(n) {
    var u = n.toUpperCase()
    if (u == "BINARY" || u == "ASCII-8BIT") { return "ASCII-8BIT" }
    if (u == "US-ASCII" || u == "ASCII") { return "US-ASCII" }
    if (u == "UTF-8") { return "UTF-8" }
    return ""
}

// The value of the hex digits of s from i up to (not including) lim, and how
// many there were.
function rbsHexRun(s, i, lim) {
    var v = 0
    var k = i
    while (k < lim) {
        var d = "0123456789abcdef".indexOf(s.charAt(k).toLowerCase())
        if (d < 0) { break }
        v = v * 16 + d
        k = k + 1
    }
    return {v: v, n: k - i}
}

// String#undump of the bytes s, MRI's str_undump: s must be a double-quoted
// String#dump form, optionally followed by .force_encoding("NAME") (with or
// without .dup). Answers {ok: true, b: the bytes, enc: the encoding name, "" for
// the receiver's own} or {ok: false, msg: the RuntimeError's message}. A \u
// escape makes the result UTF-8 and cannot be mixed with \x. Go twin:
// rubyUndump in abnf/jsrtrubystr.go.
function rbsUndump(s) {
    var badForm = "invalid dumped string; not wrapped with '\"' nor '\"...\".force_encoding(\"...\")' form"
    if (!rbsAsciiOnly(s)) { return {ok: false, msg: "non-ASCII character detected"} }
    if (s.indexOf(String.fromCharCode(0)) >= 0) { return {ok: false, msg: "string contains null byte"} }
    if (s.length < 2 || s.charAt(0) != "\"") { return {ok: false, msg: badForm} }
    var n = s.length
    var out = ""
    var enc = ""
    var utf8 = false
    var hexed = false
    var i = 1
    while (true) {
        if (i >= n) { return {ok: false, msg: "unterminated dumped string"} }
        var c = s.charAt(i)
        if (c == "\"") { break }
        if (c != "\\") {
            out = out + c
            i = i + 1
            continue
        }
        i = i + 1
        if (i >= n) { return {ok: false, msg: "invalid escape"} }
        var e = s.charAt(i)
        var named = "nrtfvbae".indexOf(e)
        if (e == "\\" || e == "\"" || e == "#") {
            out = out + e
            i = i + 1
        } else if (named >= 0) {
            out = out + String.fromCharCode([10, 13, 9, 12, 11, 8, 7, 27][named])
            i = i + 1
        } else if (e == "u") {
            if (hexed) { return {ok: false, msg: "hex escape and Unicode escape are mixed"} }
            utf8 = true
            enc = "UTF-8"
            i = i + 1
            if (i >= n) { return {ok: false, msg: "invalid Unicode escape"} }
            var braced = s.charAt(i) == "{"
            if (braced) { i = i + 1 }
            while (true) {
                if (braced && i >= n) { return {ok: false, msg: "unterminated Unicode escape"} }
                var ch = s.charAt(i)
                if (braced && ch == "}") { i = i + 1; break }
                var cc = s.charCodeAt(i)
                if (braced && (cc == 32 || (cc >= 9 && cc <= 13))) { i = i + 1; continue }
                var h = rbsHexRun(s, i, braced || i + 4 > n ? n : i + 4)
                if (braced ? h.n == 0 || h.n > 6 : h.n != 4) { return {ok: false, msg: "invalid Unicode escape"} }
                if (h.v > 1114111) { return {ok: false, msg: "invalid Unicode codepoint (too large)"} }
                if (h.v >= 55296 && h.v <= 57343) { return {ok: false, msg: "invalid Unicode codepoint"} }
                out = out + rbsBinOf(rbsChrUtf8(h.v, []))
                i = i + h.n
                if (!braced) { break }
            }
        } else if (e == "x") {
            if (utf8) { return {ok: false, msg: "hex escape and Unicode escape are mixed"} }
            hexed = true
            i = i + 1
            var h2 = rbsHexRun(s, i, i + 2 < n ? i + 2 : n)
            if (h2.n != 2) { return {ok: false, msg: "invalid hex escape"} }
            out = out + String.fromCharCode(h2.v)
            i = i + 2
        } else {
            out = out + "\\" + e
            i = i + 1
        }
    }
    var rest = s.substring(i + 1)
    if (rest == "") { return {ok: true, b: out, enc: enc} }
    if (rest.length > 4 && rest.substring(0, 4) == ".dup") { rest = rest.substring(4) }
    var fe = ".force_encoding(\""
    if (rest.length <= fe.length || rest.substring(0, fe.length) != fe) { return {ok: false, msg: badForm} }
    if (utf8) { return {ok: false, msg: "dumped string contained Unicode escape but used force_encoding"} }
    rest = rest.substring(fe.length)
    var q = rest.indexOf("\"")
    if (q < 0 || rest.substring(q) != "\")") { return {ok: false, msg: badForm} }
    var en = rbsEncCanon(rest.substring(0, q))
    if (en == "") { return {ok: false, msg: "dumped string has unknown encoding name"} }
    return {ok: true, b: out, enc: en}
}

// The invalid UTF-8 sequence at byte i of b (where rbsU8At answers 0) as MRI's
// transcoder delimits it: a lead byte and the continuation bytes that may
// follow it. [its length, 0 for a byte that starts no character, 1 for a
// sequence the bytes end inside, 2 for one the next byte breaks].
function rbsU8Bad(b, i) {
    var c = b.charCodeAt(i)
    var n = 0
    if (c >= 194 && c <= 223) { n = 2 }
    else if (c >= 224 && c <= 239) { n = 3 }
    else if (c >= 240 && c <= 244) { n = 4 }
    else { return [1, 0] }
    var k = 1
    while (k < n) {
        if (i + k >= b.length) { return [k, 1] }
        var d = b.charCodeAt(i + k)
        var lo = 128
        var hi = 191
        if (k == 1 && c == 224) { lo = 160 }
        if (k == 1 && c == 237) { hi = 159 }
        if (k == 1 && c == 240) { lo = 144 }
        if (k == 1 && c == 244) { hi = 143 }
        if (d < lo || d > hi) { return [k, 2] }
        k = k + 1
    }
    return [n, 2]
}

// String#encode over the modelled encodings UTF-8, US-ASCII and ASCII-8BIT:
// the bytes b of encoding src written in dst. inv and undef are what replaces an
// invalid byte sequence and a character dst has no code for, null to raise;
// err receives MRI's message and class then. Bytes in one encoding are copied
// as they are unless inv is given (String#scrub's case).
// The bytes b of a String in encoding enc cut at its invalid sequences (as
// rbsU8Bad delimits them): the even entries are the valid runs, the odd ones
// the sequences #scrub hands its block. Go twin: rubyScrubSplit in
// abnf/jsrtrubystr.go.
function rbsScrubSplit(b, enc) {
    var parts = []
    var run = ""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var n = 1
        var bad = c >= 128 && enc == "US-ASCII"
        if (c >= 128 && enc == "UTF-8") {
            n = rbsU8At(b, i)
            if (n == 0) {
                n = rbsU8Bad(b, i)[0]
                bad = true
            }
        }
        if (bad) {
            parts.push(run)
            parts.push(b.substring(i, i + n))
            run = ""
        } else {
            run = run + b.substring(i, i + n)
        }
        i = i + n
    }
    parts.push(run)
    return parts
}

function rbsEncode(b, src, dst, inv, undef, err) {
    if (src == dst && inv === null) { return b }
    var out = ""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        if (c < 128) {
            out = out + b.charAt(i)
            i = i + 1
        } else if (src == "ASCII-8BIT") {
            if (dst == "ASCII-8BIT") { out = out + b.charAt(i) }
            else if (undef !== null) { out = out + undef }
            else {
                err.push(rbsBinInspect(b.charAt(i)) + " from ASCII-8BIT to " + dst)
                err.push("Encoding::UndefinedConversionError")
                return out
            }
            i = i + 1
        } else if (src == "US-ASCII") {
            if (inv === null) {
                err.push(rbsBinInspect(b.charAt(i)) + " on US-ASCII")
                err.push("Encoding::InvalidByteSequenceError")
                return out
            }
            out = out + inv
            i = i + 1
        } else {
            var n = rbsU8At(b, i)
            if (n > 0 && dst == "UTF-8") { out = out + b.substring(i, i + n) }
            else if (n > 0 && undef !== null) { out = out + undef }
            else if (n > 0) {
                var hx = rbsHexUp(rbsU8Cp(b, i, n))
                while (hx.length < 4) { hx = "0" + hx }
                err.push("U+" + hx + " from UTF-8 to " + dst)
                err.push("Encoding::UndefinedConversionError")
                return out
            } else {
                var bad = rbsU8Bad(b, i)
                n = bad[0]
                if (inv !== null) { out = out + inv }
                else {
                    var q = rbsBinInspect(b.substring(i, i + n))
                    if (bad[1] == 1) { q = "incomplete " + q }
                    if (bad[1] == 2) { q = q + " followed by " + rbsBinInspect(b.charAt(i + n)) }
                    err.push(q + " on UTF-8")
                    err.push("Encoding::InvalidByteSequenceError")
                    return out
                }
            }
            i = i + n
        }
    }
    return out
}

// Upper-case hex digits of a non-negative integer.
function rbsHexUp(v) {
    var hx = "0123456789ABCDEF"
    var out = ""
    var x = v
    while (x >= 16) {
        out = hx.charAt(x % 16) + out
        x = Math.floor(x / 16)
    }
    return hx.charAt(x) + out
}

// ----- Array#pack / String#unpack -----
// Integers (C c S s L l Q q J j I i n N v V U, with the _ ! < > modifiers),
// floats (e E g G d D f F), Strings (a A Z B b H h m) and positions (x X @),
// each with a count or `*'. Go twins: rubyPack / rubyUnpack in
// abnf/jsrtrubystr.go.
//
// The engine hands rbsPack its values as a number (an Integer), {flo: number}
// (a Float), {big: decimal text} (an Integer past 2^53), {str: bytes} (a
// String, one char per byte), null (nil) or {cls: class name}; rbsUnpack answers
// numbers, {flo: number}, {big: decimal text}, {str: bytes, enc: encoding name}
// and null. An error is err[0] (the message) and err[1] (the class,
// ArgumentError when absent).

// The byte width of an integer directive, 0 for any other one, -1 for one that
// is not modelled.
function rbsPackSize(d) {
    if (d == "C" || d == "c") { return 1 }
    if (d == "S" || d == "s" || d == "n" || d == "v") { return 2 }
    if (d == "L" || d == "l" || d == "N" || d == "V" || d == "I" || d == "i") { return 4 }
    if (d == "Q" || d == "q" || d == "J" || d == "j") { return 8 }
    if (d == "e" || d == "f" || d == "F" || d == "g") { return 4 }
    if (d == "E" || d == "d" || d == "D" || d == "G") { return 8 }
    if (d == "U" || d == "a" || d == "A" || d == "Z" || d == "B" || d == "b" || d == "H" || d == "h"
        || d == "m" || d == "x" || d == "X" || d == "@") { return 0 }
    return 0 - 1
}

// The directives as [letter, count, byte width, big-endian, signed] records;
// count -1 for `*'. err[0] names the first directive that is not modelled, or
// is MRI's message with err[1] its class for a modifier after a type that
// takes none.
function rbsPackDirs(fmt, err) {
    var out = []
    var i = 0
    while (i < fmt.length) {
        var d = fmt.charAt(i)
        i = i + 1
        if (d == " " || d == "\t" || d == "\n") { continue }
        var size = rbsPackSize(d)
        if (size < 0) {
            err.push(d)
            return out
        }
        var be = d == "n" || d == "N" || d == "g" || d == "G"
        var signed = d == "c" || d == "s" || d == "l" || d == "q" || d == "j" || d == "i"
        var native = "sSiIlLqQjJ"
        while (i < fmt.length && (fmt.charAt(i) == "_" || fmt.charAt(i) == "!" || fmt.charAt(i) == "<" || fmt.charAt(i) == ">")) {
            var mo = fmt.charAt(i)
            if (native.indexOf(d) < 0) {
                err.push("'" + mo + "' allowed only after types " + native)
                err.push("ArgumentError")
                return out
            }
            if (mo == "<" || mo == ">") { be = mo == ">" }
            else if (d == "l" || d == "L") { size = 8 }
            i = i + 1
        }
        var n = d == "m" ? 45 : 1
        if (i < fmt.length && fmt.charAt(i) == "*") {
            n = 0 - 1
            i = i + 1
        } else if (i < fmt.length && fmt.charCodeAt(i) >= 48 && fmt.charCodeAt(i) <= 57) {
            n = 0
            while (i < fmt.length && fmt.charCodeAt(i) >= 48 && fmt.charCodeAt(i) <= 57) {
                n = n * 10 + (fmt.charCodeAt(i) - 48)
                i = i + 1
            }
        }
        out.push([d, n, size, be, signed])
    }
    return out
}

// The value's class for a TypeError message.
function rbsPackCls(v) {
    if (v === null) { return "nil" }
    if (typeof v == "number") { return "Integer" }
    if (typeof v.big == "string") { return "Integer" }
    if (typeof v.flo == "number") { return "Float" }
    if (typeof v.str == "string") { return "String" }
    return v.cls
}

// A pack value for an integer directive: a Float truncated.
function rbsPackIntVal(v) {
    if (v !== null && typeof v == "object" && typeof v.flo == "number") { return Math.trunc(v.flo) }
    return v
}

// A float directive: e E g G d D f F.
function rbsFloDir(d) {
    return d == "e" || d == "E" || d == "g" || d == "G" || d == "d" || d == "D" || d == "f" || d == "F"
}

// x * 2^k, exact while the result is a double.
function rbsScale2(x, k) {
    var v = x
    var n = k
    while (n > 0) { v = v * 2; n = n - 1 }
    while (n < 0) { v = v / 2; n = n + 1 }
    return v
}

// The size (4 or 8) little-endian bytes of the IEEE 754 binary32 / binary64 of
// x, half to even. A value past the largest binary32 is an infinity there
// (MRI's VALUE_to_float), and a NaN is the quiet one with the sign clear.
function rbsFloBytes(x, size) {
    var mb = size == 4 ? 23 : 52
    var bias = size == 4 ? 127 : 1023
    var emax = size == 4 ? 255 : 2047
    var top = rbsScale2(1, mb)
    var zero = 0
    var sign = x < 0 || (x === 0 && 1 / x < 0) ? 1 : 0
    var a = sign == 1 ? -x : x
    var ex = 0
    var f = 0
    if (x !== x) {
        sign = 0
        ex = emax
        f = top / 2
    } else if (a !== 0 && (a / 2 === a || (size == 4 && a > rbsScale2(16777215, 104)))) {
        ex = emax
    } else if (a !== 0) {
        var e = 0
        var t = a
        while (t >= 2) { t = t / 2; e = e + 1 }
        while (t < 1) { t = t * 2; e = e - 1 }
        var q = e - mb
        if (q < 1 - bias - mb) { q = 1 - bias - mb }
        var s = rbsScale2(a, -q)
        var m = Math.floor(s)
        var d = s - m
        if (d > 0.5 || (d === 0.5 && m % 2 == 1)) { m = m + 1 }
        ex = q + mb + bias
        if (m >= top * 2) {
            m = m / 2
            ex = ex + 1
        }
        if (m < top) { ex = 0 }
        f = m < top ? m : m - top
        if (ex >= emax) {
            ex = emax
            f = 0
        }
    }
    var bits = []
    var v = f
    while (bits.length < mb) {
        bits.push(v % 2)
        v = Math.floor(v / 2)
    }
    v = ex
    while (bits.length < size * 8 - 1) {
        bits.push(v % 2)
        v = Math.floor(v / 2)
    }
    bits.push(sign)
    var out = []
    var k = 0
    while (k < size) {
        var by = 0
        var j = 7
        while (j >= 0) {
            by = by * 2 + bits[k * 8 + j]
            j = j - 1
        }
        out.push(by)
        k = k + 1
    }
    return out
}

// The Float the size bytes of b at p spell (big-endian when be).
function rbsFloOf(b, p, size, be) {
    var mb = size == 4 ? 23 : 52
    var bias = size == 4 ? 127 : 1023
    var emax = size == 4 ? 255 : 2047
    var bits = []
    var q = 0
    while (q < size) {
        var by = b.charCodeAt(be ? p + size - 1 - q : p + q)
        var j = 0
        while (j < 8) {
            bits.push(by % 2)
            by = Math.floor(by / 2)
            j = j + 1
        }
        q = q + 1
    }
    var f = 0
    var i = mb - 1
    while (i >= 0) {
        f = f * 2 + bits[i]
        i = i - 1
    }
    var ex = 0
    i = size * 8 - 2
    while (i >= mb) {
        ex = ex * 2 + bits[i]
        i = i - 1
    }
    var zero = 0
    var v = 0
    if (ex == emax) { v = f == 0 ? 1 / zero : zero / zero }
    else if (ex == 0) { v = rbsScale2(f, 1 - bias - mb) }
    else { v = rbsScale2(rbsScale2(1, mb) + f, ex - bias - mb) }
    return bits[size * 8 - 1] == 1 ? -v : v
}

// The k little-endian bytes of the decimal integer text dec modulo 2^(8k), two's
// complement for a negative one.
function rbsDecBytes(dec, k) {
    var neg = dec.charAt(0) == "-"
    var digits = []
    var i = neg ? 1 : 0
    while (i < dec.length) {
        digits.push(dec.charCodeAt(i) - 48)
        i = i + 1
    }
    var out = []
    while (out.length < k) {
        var rem = 0
        var j = 0
        while (j < digits.length) {
            var cur = rem * 10 + digits[j]
            digits[j] = Math.floor(cur / 256)
            rem = cur % 256
            j = j + 1
        }
        out.push(rem)
    }
    if (neg) {
        var carry = 1
        var q = 0
        while (q < k) {
            var b = 255 - out[q] + carry
            out[q] = b % 256
            carry = b >= 256 ? 1 : 0
            q = q + 1
        }
    }
    return out
}

// The decimal text of the little-endian bytes b (two's complement when neg).
function rbsBytesDec(b, neg) {
    var bytes = []
    var i = 0
    while (i < b.length) {
        bytes.push(neg ? 255 - b[i] : b[i])
        i = i + 1
    }
    if (neg) {
        var carry = 1
        var q = 0
        while (q < bytes.length && carry > 0) {
            var s = bytes[q] + carry
            bytes[q] = s % 256
            carry = s >= 256 ? 1 : 0
            q = q + 1
        }
    }
    var out = ""
    var nonzero = true
    while (nonzero) {
        var rem = 0
        nonzero = false
        var j = bytes.length - 1
        while (j >= 0) {
            var cur = rem * 256 + bytes[j]
            bytes[j] = Math.floor(cur / 10)
            rem = cur % 10
            if (bytes[j] > 0) { nonzero = true }
            j = j - 1
        }
        out = String.fromCharCode(48 + rem) + out
    }
    return (neg ? "-" : "") + out
}

// The k bytes of an Integer, little-endian.
function rbsIntBytes(v, k) {
    if (typeof v != "number") { return rbsDecBytes(v.big, k) }
    var out = []
    var x = v
    if (k == 8) {
        var hi = Math.floor(x / 4294967296)
        var lo = x - hi * 4294967296
        out = rbsIntBytes(lo, 4)
        var hb = rbsIntBytes(hi, 4)
        var t = 0
        while (t < 4) {
            out.push(hb[t])
            t = t + 1
        }
        return out
    }
    var m = Math.pow(2, 8 * k)
    x = ((x % m) + m) % m
    var i = 0
    while (i < k) {
        out.push(x % 256)
        x = Math.floor(x / 256)
        i = i + 1
    }
    return out
}

function rbsB64Char(n) {
    return "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".charAt(n)
}

// The base64 text of the bytes s: `m' wraps a line every `per' input bytes, and
// per 0 (`m0') writes no newline.
function rbsB64(s, per) {
    var out = ""
    var i = 0
    var line = 0
    while (i < s.length) {
        var b0 = s.charCodeAt(i)
        var b1 = i + 1 < s.length ? s.charCodeAt(i + 1) : 0
        var b2 = i + 2 < s.length ? s.charCodeAt(i + 2) : 0
        out = out + rbsB64Char(Math.floor(b0 / 4)) + rbsB64Char((b0 % 4) * 16 + Math.floor(b1 / 16))
        out = out + (i + 1 < s.length ? rbsB64Char((b1 % 16) * 4 + Math.floor(b2 / 64)) : "=")
        out = out + (i + 2 < s.length ? rbsB64Char(b2 % 64) : "=")
        i = i + 3
        line = line + 3
        if (per > 0 && (line >= per || i >= s.length)) {
            out = out + "\n"
            line = 0
        }
    }
    return out
}

// The bytes base64 text s spells; anything not in the alphabet is skipped.
function rbsUnB64(s) {
    var out = ""
    var acc = 0
    var bits = 0
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        i = i + 1
        var v = 0 - 1
        if (c >= 65 && c <= 90) { v = c - 65 }
        else if (c >= 97 && c <= 122) { v = c - 71 }
        else if (c >= 48 && c <= 57) { v = c + 4 }
        else if (c == 43) { v = 62 }
        else if (c == 47) { v = 63 }
        else if (c == 61) { break }
        if (v < 0) { continue }
        acc = acc * 64 + v
        bits = bits + 6
        if (bits >= 8) {
            bits = bits - 8
            var p = Math.pow(2, bits)
            out = out + String.fromCharCode(Math.floor(acc / p))
            acc = acc % p
        }
    }
    return out
}

// The nibble value of a hex digit as MRI reads it for H / h.
function rbsNibble(c) {
    if ((c >= 65 && c <= 90) || (c >= 97 && c <= 122)) { return ((c % 8) + 9) % 16 }
    return c % 16
}

// Array#pack: the bytes, one char per byte.
function rbsPack(vals, dirs, err) {
    var out = ""
    var k = 0
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        var n = dirs[i][1]
        var size = dirs[i][2]
        if (d == "x") {
            out = out + rbsRepeatTo(String.fromCharCode(0), n < 0 ? 0 : n)
            i = i + 1
            continue
        }
        if (d == "X") {
            var back = n < 0 ? 0 : n
            if (back > out.length) {
                err.push("X outside of string")
                return out
            }
            out = out.substring(0, out.length - back)
            i = i + 1
            continue
        }
        if (d == "@") {
            var at = n < 0 ? 0 : n
            out = at <= out.length ? out.substring(0, at) : out + rbsRepeatTo(String.fromCharCode(0), at - out.length)
            i = i + 1
            continue
        }
        if (size == 0 && d != "U") {
            if (k >= vals.length) {
                err.push("too few arguments")
                return out
            }
            var sv = vals[k]
            k = k + 1
            if (sv === null || typeof sv != "object" || typeof sv.str != "string") {
                err.push("no implicit conversion of " + rbsPackCls(sv) + " into String")
                err.push("TypeError")
                return out
            }
            out = out + rbsPackStr(d, n, sv.str)
            i = i + 1
            continue
        }
        if (n < 0) { n = vals.length - k }
        var j = 0
        while (j < n) {
            if (k >= vals.length) {
                err.push("too few arguments")
                return out
            }
            if (rbsFloDir(d)) {
                var fv = vals[k]
                k = k + 1
                var fx = 0
                if (typeof fv == "number") { fx = fv }
                else if (fv !== null && typeof fv.flo == "number") { fx = fv.flo }
                else if (fv !== null && typeof fv.big == "string") { fx = parseFloat(fv.big) }
                else {
                    err.push("can't convert " + rbsPackCls(fv) + " into Float")
                    err.push("TypeError")
                    return out
                }
                var fb = rbsFloBytes(fx, size)
                var fq = 0
                while (fq < size) {
                    out = out + String.fromCharCode(fb[dirs[i][3] ? size - 1 - fq : fq])
                    fq = fq + 1
                }
                j = j + 1
                continue
            }
            var v = rbsPackIntVal(vals[k])
            k = k + 1
            if (v === null || (typeof v != "number" && typeof v.big != "string")) {
                err.push("no implicit conversion of " + rbsPackCls(v) + " into Integer")
                err.push("TypeError")
                return out
            }
            if (d == "U") {
                if (typeof v != "number" || v < 0 || v > 2147483647) {
                    err.push("pack(U): value out of range")
                    err.push("RangeError")
                    return out
                }
                out = out + rbsU8Enc(v)
            } else {
                var bs = rbsIntBytes(v, size)
                var q = 0
                while (q < size) {
                    out = out + String.fromCharCode(bs[dirs[i][3] ? size - 1 - q : q])
                    q = q + 1
                }
            }
            j = j + 1
        }
        i = i + 1
    }
    return out
}

// The encoding of pack's String: UTF-8 when U comes first, US-ASCII when only
// m does, else ASCII-8BIT (MRI's pack.c).
function rbsPackEnc(dirs) {
    var e = "US-ASCII"
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        if (d == "U") {
            if (e == "US-ASCII") { e = "UTF-8" }
        } else if (d != "m") { e = "ASCII-8BIT" }
        i = i + 1
    }
    return e
}

// The bytes pack(U) writes for cp: UTF-8's scheme, extended to six bytes up to
// 2^31 - 1 as MRI does, so a surrogate or a value past U+10FFFF is written too.
function rbsU8Enc(cp) {
    if (cp < 128) { return String.fromCharCode(cp) }
    var n = 6
    var lead = 252
    if (cp < 2048) { n = 2; lead = 192 }
    else if (cp < 65536) { n = 3; lead = 224 }
    else if (cp < 2097152) { n = 4; lead = 240 }
    else if (cp < 67108864) { n = 5; lead = 248 }
    var tail = ""
    var v = cp
    var k = 1
    while (k < n) {
        tail = String.fromCharCode(128 + v % 64) + tail
        v = Math.floor(v / 64)
        k = k + 1
    }
    return String.fromCharCode(lead + v) + tail
}

// One String directive of pack over the bytes s.
function rbsPackStr(d, n, s) {
    if (d == "a" || d == "A" || d == "Z") {
        if (n < 0) { return d == "Z" ? s + String.fromCharCode(0) : s }
        if (s.length >= n) { return s.substring(0, n) }
        return s + rbsRepeatTo(d == "A" ? " " : String.fromCharCode(0), n - s.length)
    }
    if (d == "m") { return rbsB64(s, n == 0 ? 0 : (n < 3 ? 45 : n - n % 3)) }
    var out = ""
    var cnt = n < 0 ? s.length : n
    if (d == "B" || d == "b") {
        var acc = 0
        var bits = 0
        var i = 0
        while (i < cnt) {
            var bit = i < s.length ? s.charCodeAt(i) % 2 : 0
            acc = d == "B" ? acc * 2 + bit : acc + bit * Math.pow(2, bits)
            bits = bits + 1
            if (bits == 8) {
                out = out + String.fromCharCode(acc)
                acc = 0
                bits = 0
            }
            i = i + 1
        }
        if (bits > 0) { out = out + String.fromCharCode(d == "B" ? acc * Math.pow(2, 8 - bits) : acc) }
        return out
    }
    var hacc = 0
    var half = false
    var h = 0
    while (h < cnt) {
        var nib = h < s.length ? rbsNibble(s.charCodeAt(h)) : 0
        if (!half) { hacc = d == "H" ? nib * 16 : nib }
        else {
            hacc = d == "H" ? hacc + nib : hacc + nib * 16
            out = out + String.fromCharCode(hacc)
        }
        half = !half
        h = h + 1
    }
    if (half) { out = out + String.fromCharCode(hacc) }
    return out
}

// String#unpack of the bytes b (one char per byte).
function rbsUnpack(b, dirs, err) {
    var out = []
    var p = 0
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        var n = dirs[i][1]
        var size = dirs[i][2]
        if (d == "x" || d == "X" || d == "@") {
            var to = d == "@" ? (n < 0 ? 0 : n) : (d == "x" ? p + (n < 0 ? b.length - p : n) : p - (n < 0 ? 0 : n))
            if (to < 0 || to > b.length) {
                err.push(d + " outside of string")
                return out
            }
            p = to
        } else if (size == 0 && d != "U") {
            var r = rbsUnpackStr(d, n, b, p)
            out.push(r[0])
            p = r[1]
        } else {
            var j = 0
            while (n < 0 ? p + (d == "U" ? 1 : size) <= b.length : j < n) {
                if (d == "U") {
                    if (p >= b.length) { break }
                    var c = b.charCodeAt(p)
                    var len = c < 128 ? 1 : (c >= 240 ? 4 : (c >= 224 ? 3 : 2))
                    var t = rbsBinDecode(b.substring(p, p + len))
                    if (t === null) {
                        err.push("malformed UTF-8 character")
                        return out
                    }
                    out.push(rbsOrd(t))
                    p = p + len
                } else if (p + size > b.length) {
                    out.push(null)
                    p = b.length
                } else if (rbsFloDir(d)) {
                    out.push({flo: rbsFloOf(b, p, size, dirs[i][3])})
                    p = p + size
                } else {
                    out.push(rbsIntOf(b, p, size, dirs[i][3], dirs[i][4]))
                    p = p + size
                }
                j = j + 1
            }
        }
        i = i + 1
    }
    return out
}

// The integer of the size bytes of b at p.
function rbsIntOf(b, p, size, be, signed) {
    var le = []
    var q = 0
    while (q < size) {
        le.push(b.charCodeAt(be ? p + size - 1 - q : p + q))
        q = q + 1
    }
    var neg = signed && le[size - 1] >= 128
    if (size == 8) {
        var lo = le[0] + le[1] * 256 + le[2] * 65536 + le[3] * 16777216
        var hi = le[4] + le[5] * 256 + le[6] * 65536 + le[7] * 16777216
        if (neg) { hi = hi - 4294967296 }
        if (hi < 2097152 && hi >= 0 - 2097152) { return hi * 4294967296 + lo }
        return {big: rbsBytesDec(le, neg)}
    }
    var v = 0
    var k = size - 1
    while (k >= 0) {
        v = v * 256 + le[k]
        k = k - 1
    }
    return neg ? v - Math.pow(2, 8 * size) : v
}

// One String directive of unpack at byte p: [the String, the next position].
function rbsUnpackStr(d, n, b, p) {
    var rest = b.length - p
    if (d == "a" || d == "A" || d == "Z") {
        var cnt = n < 0 || n > rest ? rest : n
        var s = b.substring(p, p + cnt)
        var next = p + cnt
        if (d == "Z") {
            var z = s.indexOf(String.fromCharCode(0))
            if (z >= 0) {
                s = s.substring(0, z)
                if (n < 0) { next = p + z + 1 }
            }
        } else if (d == "A") {
            var e = s.length
            while (e > 0 && (s.charCodeAt(e - 1) == 32 || s.charCodeAt(e - 1) == 0)) { e = e - 1 }
            s = s.substring(0, e)
        }
        return [{str: s, enc: "ASCII-8BIT"}, next]
    }
    if (d == "m") { return [{str: rbsUnB64(b.substring(p, b.length)), enc: "ASCII-8BIT"}, b.length] }
    var out = ""
    if (d == "B" || d == "b") {
        var bits = n < 0 || n > rest * 8 ? rest * 8 : n
        var i = 0
        while (i < bits) {
            var by = b.charCodeAt(p + Math.floor(i / 8))
            var sh = d == "B" ? 7 - i % 8 : i % 8
            out = out + (Math.floor(by / Math.pow(2, sh)) % 2 == 1 ? "1" : "0")
            i = i + 1
        }
        return [{str: out, enc: "US-ASCII"}, p + Math.floor((bits + 7) / 8)]
    }
    var hx = "0123456789abcdef"
    var nibs = n < 0 || n > rest * 2 ? rest * 2 : n
    var h = 0
    while (h < nibs) {
        var hb = b.charCodeAt(p + Math.floor(h / 2))
        var hi = Math.floor(hb / 16)
        var lo = hb % 16
        out = out + hx.charAt((h % 2 == 0) == (d == "H") ? hi : lo)
        h = h + 1
    }
    return [{str: out, enc: "US-ASCII"}, p + Math.floor((nibs + 1) / 2)]
}
