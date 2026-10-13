// swift-chars.js -- Swift's Character and a String as Characters, for
// swift-interpreter.abnf and swift-to-llvm-ir.abnf (include) and
// languages/lib/swift-rt.metajs (import). abnf/jsrtswift.go is the Go twin's copy.
// It needs java-chars.js (its Unicode 16.0 table and its base-64 record decoders)
// in the same scope.
//
// A Character is its one-cluster String here. Its properties are those of
// swift 6.1.2's CharacterProperties.swift: most read the FIRST scalar's Unicode
// properties, the numeric and hex ones need a single-scalar Character, and
// "\r\n" is the one two-scalar Character with an ASCII value (10). isUppercase /
// isLowercase read the first scalar too, where Swift falls back to a case-mapping
// comparison for a multi-scalar Character; the two agree whenever the first
// scalar carries the case, which is every cluster a combining mark extends.

// Whether name is a Character property this file answers.
function swcIsProp(name) {
    return name == "isLetter" || name == "isUppercase" || name == "isLowercase" || name == "isNumber"
        || name == "isWholeNumber" || name == "wholeNumberValue" || name == "isWhitespace"
        || name == "isNewline" || name == "asciiValue" || name == "isASCII" || name == "isHexDigit"
        || name == "hexDigitValue" || name == "isPunctuation" || name == "isSymbol" || name == "isMathSymbol"
        || name == "isCurrencySymbol" || name == "isCased"
}

// The code point at UTF-16 offset i of s.
function swcCodeAt(s, i) {
    var c = s.charCodeAt(i)
    if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
        var d = s.charCodeAt(i + 1)
        if (d >= 56320 && d <= 57343) { return (c - 55296) * 1024 + (d - 56320) + 65536 }
    }
    return c
}

// Unicode's White_Space property.
function swcIsWhite(c) {
    if (c >= 9 && c <= 13) { return true }
    if (c == 32 || c == 133 || c == 160 || c == 5760) { return true }
    if (c >= 8192 && c <= 8202) { return true }
    return c == 8232 || c == 8233 || c == 8239 || c == 8287 || c == 12288
}

// The numeric value Swift's numericType / numericValue see: java's
// getNumericValue without its Latin-letter digits (a..z, A..Z and their
// fullwidth forms are 10..35 there and no number in Swift). -1 is no number,
// -2 a number that is no non-negative integer (1/2).
function swcNumVal(c) {
    if ((c >= 65 && c <= 90) || (c >= 97 && c <= 122)) { return 0 - 1 }
    if ((c >= 65313 && c <= 65338) || (c >= 65345 && c <= 65370)) { return 0 - 1 }
    return jcNumericValue(c)
}

// The value of a hex digit, ASCII or fullwidth, or -1.
function swcHexVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 65 && c <= 70) { return c - 55 }
    if (c >= 97 && c <= 102) { return c - 87 }
    if (c >= 65296 && c <= 65305) { return c - 65296 }
    if (c >= 65313 && c <= 65318) { return c - 65303 }
    if (c >= 65345 && c <= 65350) { return c - 65335 }
    return 0 - 1
}

// Character property `name` of the Character s; null is nil.
function swcProp(s, name) {
    var c = swcCodeAt(s, 0)
    var single = s.length == 1
    if (c > 65535) { single = s.length == 2 }
    if (name == "asciiValue" || name == "isASCII") {
        var av = 0 - 1
        if (s == "\r\n") { av = 10 }
        else if (single && c < 128) { av = c }
        if (name == "isASCII") { return av >= 0 }
        if (av < 0) { return null }
        return av
    }
    // The General_Category members read the first scalar, except that a
    // Character of several scalars is cased when either case mapping changes it.
    if (name == "isCased" && !single) { return unicodeCase(s, 3) != s || unicodeCase(s, 2) != s }
    var gk = 0
    if (name == "isPunctuation") { gk = 1 }
    else if (name == "isSymbol") { gk = 2 }
    else if (name == "isMathSymbol") { gk = 4 }
    else if (name == "isCurrencySymbol") { gk = 8 }
    else if (name == "isCased") { gk = 16 }
    if (gk > 0) { return Math.floor(swcGcBits(c) / gk) % 2 == 1 }
    if (name == "isLetter") { return jcIsAlphabetic(c) }
    if (name == "isUppercase") { return jcIsUpper(c) }
    if (name == "isLowercase") { return jcIsLower(c) }
    if (name == "isWhitespace") { return swcIsWhite(c) }
    if (name == "isNewline") { return (c >= 10 && c <= 13) || c == 133 || c == 8232 || c == 8233 }
    if (name == "isNumber") { return swcNumVal(c) != 0 - 1 }
    if (name == "isHexDigit" || name == "hexDigitValue") {
        var hv = 0 - 1
        if (single) { hv = swcHexVal(c) }
        if (name == "isHexDigit") { return hv >= 0 }
        if (hv < 0) { return null }
        return hv
    }
    var nv = 0 - 1
    if (single) { nv = swcNumVal(c) }
    if (name == "isWholeNumber") { return nv >= 0 }
    if (nv < 0) { return null }
    return nv
}

// ----- the General_Category properties -----
// isPunctuation, isSymbol, isMathSymbol, isCurrencySymbol and isCased of a
// one-scalar Character, from a table derived from swiftc 6.1.2's own Character
// properties over every code point: one record per run of code points with the
// same answers, five base-64 digits each in java-chars.js's encoding - the run's
// first code point in four, the answers in one as the bits
//     1 isPunctuation  2 isSymbol  4 isMathSymbol  8 isCurrencySymbol  16 isCased
// Twin: swcGcBits in abnf/jsrtswiftchars.go, over the same table.
var swcGcTable = ""
function swcGcTab() {
    if (swcGcTable == "") {
        swcGcTable = ""
        + "00000000Q1000T:000U1000[6000]1000a0000k1000m6000p10011@001K1001N6001O1001P2001Q@001l1001m6001n1001o6"
        + "001p0002Q1002R:002V2002W1002X2002Z@002[1002]6002^0002_2002b6002c0002e2002f@002g1002i2002j0002k@002l1"
        + "002m0002p10030@003G6003H@003h6003i@006l0006m@007000074@00:D000:F@00:j000;0@00;2200;6000;B200;P@00;U2"
        + "00;]000;^200;_000;`200<0000=5@00=6000=a@00=e000=f200=g@00=i000=k@00=o100=p@00>0000>4200>6@00>7100>8@"
        + "00>;000><@00>=000>>@00>R000>S@00?@D00?C@00?ED00?F@00?aD00?c@00?eD00?g600?h@00B2200B3000B:@00Da000Db@"
        + "00EG000EJ100EP@00F9100F;000F=200F?:00F@000Fo100Fp000G0100G1000G3100G4000G6100G7000Gd100Gf000H6600H91"
        + "00H;:00H<100H>200H@000HK100HL000HM100HP000IZ100I_000KD100KE000KN200KO000KY200KZ000Kn200Kp000L0100L>0"
        + "00Og200Oh100Ok000Oo:00P0000Pa100Pp000QN100QO000R8200R9000UT100UV000Ua100Ub000Wc:00We000Wk200Wl:00Wm0"
        + "00Wn100Wo000Yg100Yh000[a100[b:00[c000^a200^b000`d200`j:00`k200`l000bh100bi000bp200c0000c4100c5000f?2"
        + "00f@000fj200fk000he100hf000ip:00j0000j?100j@000jJ100jL000m1200m4100mC200mD100mE200mH000mJ200mP000me2"
        + "00mf000mg200mh000mi200mj000mk100mo000o5100o6000oo200p6000p7200p=000p>200p@100pE200pI100pK0011:1011@0"
        + "012N2012P@013600137@01380013=@013>0013@@013l1013m@0140001=P101=Y001>@201>J001>P@01?g001?i@01?o001@01"
        + "01@1001I^201I_101I`001JK101JM001K[101K_001Lf101Lh001OD101OG001OH101OK:01OL001P0101P;001U0201U1001U41"
        + "01U6001WN201X0001XN101XP001ZP101ZW001ZX101Z_001^>101^@001^J101^Q201^[001^e201^n101_0001`m101a0001al1"
        + "01b0001bo101c0@01c;001c@@01cl001cn@01d0101d8001dC101dD001e0@01h0001i0@01mF001mH@01mN001mP@01n6001n8@"
        + "01n>001n@@01nH001nI@01nJ001nK@01nL001nM@01nN001nO@01no001o0@01of001og@01on201oo@01op201p2@01p5001p6@"
        + "01p=201p@@01pD001pF@01pL001pM201pP@01p^201pa001pc@01pf001pg@01pn201pp0020@1020F5020G1020X0020a1020c5"
        + "020f102105021110214602151021B6021C1021O0021Q4021U0021b@021c0021k6021n5021p@02200022:6022=5022?0022@@"
        + "022M0022P:02320023@4023M0023Q4023R0023U4023W0023[4023a0024020242D024320247D02482024:D024D2024ED024F2"
        + "024H6024ID024N2024TD024U2024V@024W2024XD024Y6024Z@024]D024_2024`D024c@024dD024f4024j@024k2024mD02506"
        + "0255D025:2025;6025<2025>@025?2025@0025P@026000263@02650026:2026<0026@6026X2026Y6026`2026a6026c2026g6"
        + "026i2026m6027L2027M6027N2027T6027V2027e602<0202<8502<<202<P602<R202<Y102<[202=m602=n202>K602>g202>h6"
        + "02>i202?@602?A202?L602?S202@Z002A0202A;002BL202BgB02CZ002D0202FP602FR202F_602Fi202Fm602G2202G6602G82"
        + "02G:602G<202G?602GD202GR602GS202GT602GU202GW602G^202Gi602H0202H5602H7202I0602I1202I2602I3202IP602IT2"
        + "02I^602Ia202MX102Mg002ND202O0602O5502O7602OV502Oa602P0202T0602V3502VI602WH502WL602Wm502Wo602]0202]a6"
        + "02^5202^7602^=202^e002^g202a0@02dU202d[@02d`002dc@02de002dj102dn002do102e0@02eV002eW@02eX002e^@02e_0"
        + "02fa102fb002i0102i`002ia102j@202jB102jN002k0202kJ002kK202le002m0202pF002pa20300003011030420305003081"
        + "030B2030D1030P2030Q0030a1030b0030g2030i0030n1030o203100032K2032M0032P1032Q0033l1033m0036@2036B0036F2"
        + "036P003702037V0037`2037a003802038O0038Z203980039@2039A0039P203:0003::203:b003;0203@0004h0204i000:B@2"
        + "0:C700:Co10:D000:H=10:H@00:I0@0:I_00:Id10:Ie00:Io10:Ip00:J0@0:JN00:Kc10:Ki00:L020:LG00:LP20:LR@0:N80"
        + "0:N920:N;@0:N?00:N@@0:OM00:Ob@0:Oh00:Oi@0:Ol00:PX20:P]00:Pg20:Pi:0:Pj20:Pk00:Qe10:Qi00:S>10:S@00:Si1"
        + "0:Sl00:Sm10:Sn00:T_10:Ta00:UO10:UP00:W110:W>00:WN10:WP00:YL10:YP00:Yh20:Yk00:[N10:[P00:[a10:[c00:]a@"
        + "0:^K20:^L@0:^Z20:^]00:^a@0:`000:`[10:`]00?Rm40?Ro00?S040?S100?]0@0?]700?]C@0?]H00?]Y60?]Z00?_c20?`C0"
        + "0?eo10?f020?f@00?g@20?gB00?h820?h@00?hm:0?hn20?i000?i@10?iJ00?ia10?jC00?jD10?jQ50?jR60?jS50?jT60?jW0"
        + "0?jX50?jY:0?jZ10?j]00?m110?m4:0?m510?m;60?m<10?m@00?mJ10?mL60?mO10?mQ@0?ml10?mm50?mn10?mo60?mp10?n02"
        + "0?n1@0?nK10?nL60?nM10?nN60?nO10?nV00?pP:0?pR60?pS20?pU:0?pW00?pX20?pY60?p^20?p`00?pm20?po00@4010@430"
        + "0@4h20@5000@5j20@6:00@6<20@6?00@6@20@6M00@6P20@6Q00@7@20@7n00@>O10@>P00@?@10@?A00@@0@0@A@00@Ba@0@CD0"
        + "0@CH@0@Cm00@E`10@Ea@0@El00@Em@0@F;00@F<@0@FC00@FD@0@FF00@FG@0@FR00@FS@0@Fc00@Fd@0@Fk00@Fl@0@Fn00@N0@"
        + "0@N100@N3@0@N600@N7@0@Nb00@Nc@0@Nl00@QG10@QH00@Qh20@Qj00@TO10@TP00@Tp10@U000@Y@10@YI00@Yp10@Z000@[82"
        + "0@[900@[a10@[h00@]j10@^000@_I10@_M00@c0@0@cd00@d0@0@dd00@f@@0@fV00@f_10@f`00@fa@0@g600@g>60@g@00@k^1"
        + "0@k_00@l@10@lA20@lI00@nE10@nJ00@o610@o:00A1710A1>00A2l10A2n00A2o10A3200A5010A5400A5e10A5g00A7510A790"
        + "0A7=10A7>00A7K10A7L00A7M10A7P00A8i10A8o00A:Y10A:Z00A?D10A?F00A?G10A?I00AA;10AA@00AAJ10AAL00AAM10AAN0"
        + "0AC610AC700AG110AGH00AI110AI400AIP10AI^00AJj10AJk00ALm10ALp20AM000APl10APm00ARP@0ASP00AU410AU700AWR1"
        + "0AWS00AXp10AY700AZJ10AZM00AZN10AZS00A]010A]:00A`Q10A`R00Ab110Ab600Aba10Abc00Alh10Alj00An310An@00ApE2"
        + "0ApM:0ApQ20Apc00App10B0000BAa10BAf00Bpb10Bpd00FY_10FYa00F[f10F[g00F]h10F]m20F^000F^410F^520F^600Ff^1"
        + "0Ffa00Fj0@0Fk000FkG10FkK00FkP@0Fkj00Fkl@0FlD00FpR10FpS00KcL20KcM00KcO10KcP00La020Lda00Ldk20Ldn00Le02"
        + "0Lke00Lkk20LlA00LlP20Lla60Llb00Ln@20Lp400M0020M3g00M4020M4W00M4Y20M5U00M5Z20M5^00M6320M6500M6<20M6Z0"
        + "0M6_20M7[00M8020M9200M9520M9600M<020M=G00M@0D0MAE00MAFD0MBM00MBND0MBP00MBRD0MBS00MBUD0MBW00MBYD0MB^0"
        + "0MB_D0MBk00MBlD0MBm00MBnD0MC400MC5D0MD600MD7D0MD;00MD=D0MDE00MDFD0MDM00MDND0MDk00MDlD0MDp00ME0D0ME50"
        + "0ME6D0ME700ME:D0MEA00MEBD0MJV00MJXD0MK160MK2D0MKK60MKLD0MKl60MKmD0MLE60MLFD0MLf60MLgD0MM?60MM@D0MM`6"
        + "0MMaD0MN960MN:D0MNY60MNZD0MO360MO4D0MO<00MO>40MP020MX000MXh20MXl00MY^20MYf00MYg20MZ400MZ520MZ710MZ<0"
        + "0Mm0@0Mm:00Mm;@0MmO00MmU@0Mm[00N0a@0N1_00N5?20N5@00N;p:0N<000NGp10NH000NT0@0NU400NUN10NUP00Nc]20Nc^0"
        + "0Nca:0Ncb00Ne_20Ne`00Ni040Ni400Ni540NiP00NiQ40NiS00NiT40NiU00NiW40NiX00NiY40Nid00Nie40Nii00Nij40Nik0"
        + "0Nil40Nim00Nj240Nj300Nj740Nj800Nj940Nj:00Nj;40Nj<00Nj=40Nj@00NjA40NjC00NjD40NjE00NjG40NjH00NjI40NjJ0"
        + "0NjK40NjL00NjM40NjN00NjO40NjP00NjQ40NjS00NjT40NjU00NjW40Nj[00Nj]40Njd00Nje40Nji00Njj40Njn00Njo40Njp0"
        + "0Nk040Nk:00Nk;40NkL00NkQ40NkT00NkU40NkZ00Nk[40Nkm00Nla60Nlc00O0020O0]00O0a20O2D00O2P20O2`00O2b20O300"
        + "0O3120O3@00O3A20O3g00O4=20O4aB0O5:20O5@B0O5Z20O5aB0O6:20O6_00O7V20O8300O8@20O8m00O9020O9900O9@20O9B0"
        + "0O9P20O9V00O<020OKI00OKL20OK^00OKa20OKn00OL020OOJ00OOP20OO]00OOa20OOb00OP020OP<00OP@20OQ800OQ@20OQJ0"
        + "0OQP20OR800OR@20OR_00ORa20ORm00OS020OS200OS@60OSI00OT020OYH00OYP20OY_00OYa20OYn00OZ020OZ;00OZ>20O[70"
        + "0O[820O[900O[=20O[M00O[O20O[[00O[`20O[j00O]020O_C00O_D20O`a00O`k20O`l0"
    }
    return swcGcTable
}
function swcGcBits(c) {
    var t = swcGcTab()
    return jcDig(t, jcRun(t, c, 5) * 5 + 4)
}

// ----- grapheme clusters (UAX #29) -----
// A String is a sequence of extended grapheme clusters. swcGcb is a code point's
// break class, read from a table derived from swiftc 6.1.2's own segmentation over
// every code point (so it is the Unicode version swift ships): one record per run
// of code points of one class, five base-64 digits each in java-chars.js's
// encoding - the run's first code point in four, its class in one. The classes:
//     0 Other  1 CR  2 LF  3 Control  4 Extend  5 ZWJ  6 Regional_Indicator
//     7 Prepend  8 SpacingMark  9 L  10 V  11 T  12 LV  13 LVT
//     14 Extended_Pictographic  15 Extend with InCB=Extend
//     16 Extend with InCB=Linker  17 InCB=Consonant
// Twin: swcGcb and swcBounds in abnf/jsrtswift.go, over the same table.
var swcGcbTable = ""
function swcGcbTab() {
    if (swcGcbTable == "") {
        swcGcbTable = ""
        + "00003000:2000;3000=1000>3000P0001p3002P0002Y>002Z0002^3002_>002`000<0?00=a000B3?00B:000FA?00Fo000Fp?"
        + "00G0000G1?00G3000G4?00G6000G7?00G8000H0700H6000H@?00HK000HL300HM000I;?00IP000Ia?00Ib000KF?00KM700KN0"
        + "00KO?00KU000KW?00KY000KZ?00K_000L?700L@000LA?00LB000La?00M;000NV?00Nb000O[?00Oe000On?00Oo000PF?00PJ0"
        + "00PK?00PT000PU?00PX000PY?00P_000QI?00QL000R@700RB000RG?00RP000S:?00SR700SS?00T3800T4000TEA00Tk?00Tl8"
        + "00Tm?00Tn000To800U1?00U9800U=@00U>800U@000UA?00UHA00UP000UR?00UT000UiA00V0000V1?00V2800V4000VEA00VY0"
        + "00VZA00Vb000VcA00Vd000VgA00Vk000Vm?00Vn000Vo?00Vp800W1?00W5000W7800W9000W;800W=@00W>000WG?00WH000WLA"
        + "00WN000WOA00WP000WR?00WT000WaA00Wc000Wo?00Wp000X1?00X3800X4000Xm?00Xn000Xo800Y1?00Y3000Y7?00Y9000Y;?"
        + "00Y>000YA?00YB000Ya?00Yc000Yf?00Yg000Z1?00Z3800Z4000ZEA00ZY000ZZA00Zb000ZcA00Ze000ZfA00Zk000Zm?00Zn0"
        + "00Zo800[1?00[6000[7?00[9800[:000[;800[=@00[>000[R?00[T000[jA00[k?00]0000]1?00]2800]4000]EA00]Y000]ZA"
        + "00]b000]cA00]e000]fA00]k000]m?00]n000]o?00^0800^1?00^5000^7800^9000^;800^=@00^>000^E?00^H000^LA00^N0"
        + "00^OA00^P000^R?00^T000^bA00^c000_2?00_3000_o?00_p800`0?00`1800`3000`6800`9000`:800`=?00`>000`G?00`H0"
        + "00a0?00a1800a4?00a5000aEA00aY000aZA00ak000am?00an000ao?00b1800b5000b6?00b9000b:?00b=@00b>000bE?00bG0"
        + "00bHA00bK000bR?00bT000c1?00c2800c4000cm?00cn000co800cp?00d1800d2?00d3800d5000d6?00d9000d:?00d>000dE?"
        + "00dG000dR?00dT000dd800de000e0?00e2800e4000eEA00el?00en000eo?00ep800f1?00f5000f6800f9000f:800f=@00f>7"
        + "00f?000fG?00fH000fR?00fT000g1?00g2800g4000h:?00h;000h??00h@800hB?00hE000hF?00hG000hH800hO?00hP000hc8"
        + "00he000ib?00ic000id800ie?00il000j7?00j?000kb?00kc000kd800ke?00kn000l8?00l?000mH?00mJ000mf?00mg000mh?"
        + "00mi000mj?00mk000mo800n0000nb?00np800o0?00o5000o6?00o8000o=?00oH000oI?00on000p6?00p700100A010[0010^?"
        + "010b8010c?010i0010j?010l8010n?010pA01100011@A011F8011H?011JA011N?011QA011R0011UA011W0011_A011b?011fA"
        + "0122?01230012480125?01270012=?012>A012?0012M?012N001409015P:016X;0180001=M?01=P001LB?01LF001Lc?01Lf0"
        + "01MB?01MD001Mc?01Me001N0A01Ne?01Ng801Nh?01No801O6?01O7801O9?01OD001OM?01ON001P;?01P>301P??01P@001R5?"
        + "01R7001RY?01RZ001TP?01TS801TW?01TY801T]001Ta801Tc?01Td801Tj?01Tm001XG?01XI801XK?01XL001XPA01YE801YF?"
        + "01YG801YH?01YO001YP?01YQ001YR?01YS001YU?01Y^801Yd?01Yn001Yp?01Z0001Za?01[N001[P?01[]001]0?01]4801]50"
        + "01];A01]=001]CA01]e?01]o801^2?01^5A01^=001^[?01^e001_0?01_2801_3A01_Q801_R?01_V801_X?01__A01_a001_lA"
        + "01_o001`V?01`W801`X?01`Z801`^?01`_801``?01`e001aT801a]?01ae801ag?01ai001d@?01dC001dD?01dQ801dR?01dY0"
        + "01d^?01d_001de?01df001dh801di?01dk001h0?01i00020;3020<4020=5020>3020@0020X3020`0020m>020n00219>021:0"
        + "021P3021a0023@?023b0024R>024S0024j>024k0026D>026J0026Y>026[002<J>02<L002<X>02<Y002??>02?@002?Y>02?e0"
        + "02?i>02?l002C2>02C3002FZ>02F]002Fg>02Fh002G0>02G1002Gl>02Gp002H0>02H5002H>>02H?002HA>02HB002HD>02HF0"
        + "02HH>02HI002HM>02HN002HP>02HQ002HR>02HT002HV>02HW002HZ>02H[002H_>02Ha002Hi>02Hl002I0>02I1002I2>02I30"
        + "02I8>02ID002IO>02IQ002IS>02IT002IU>02IW002IX>02IY002Il>02Im002Io>02J0002JB>02JH002JI>02JJ002JK>02JM0"
        + "02JP>02JR002JW>02JX002JZ>02J]002Ja>02Jc002Jn>02Jp002K4>02K6002K8>02K9002K>>02K@002KA>02KB002KC>02KE0"
        + "02KY>02K[002Ka>02Kg002Kh>02Kl002Kn>02Ko002L2>02L3002L5>02L6002L8>02L>002L?>02L@002LB>02LC002LD>02LE0"
        + "02LF>02LG002LM>02LN002LQ>02LR002LX>02LY002Ld>02Lf002M4>02M5002M7>02M8002M<>02M=002M>>02M?002MC>02MF0"
        + "02MG>02MH002MS>02MU002NE>02NH002NQ>02NR002Na>02Nb002Np>02O0002Te>02Tg002]5>02]8002]K>02]M002^@>02^A0"
        + "02^E>02^F002d`?02dc002fp?02g0002hP?02i00030Z?030a>030b0030n>030o0032I?032K003:G>03:H003:I>03:J00:I`?"
        + "0:Id00:Ie?0:Io00:JN?0:JP00:Ka?0:Kc00:P2?0:P300:P6?0:P700:P;?0:P<00:PS80:PU?0:PW80:PX00:P]?0:P^00:R08"
        + "0:R200:Re80:S4?0:S600:SP?0:Sc00:Sp?0:T000:TV?0:T_00:U7?0:UB80:UC?0:UD00:UP90:Un00:V0?0:V380:V400:V9A"
        + "0:V<00:V?A0:Vd?0:Ve80:Vg?0:Vk80:Vm?0:Vo80:W0?0:W100:WPA0:WU?0:WV00:WWA0:Wa00:WkA0:Wp00:XY?0:X`80:Xb?"
        + "0:Xd80:Xf?0:Xh00:Y3?0:Y400:Y<?0:Y=80:Y>00:YPA0:Ya00:YbA0:Ye00:YkA0:Yl00:Ym?0:Yn00:YoA0:Z000:Za?0:Zb0"
        + "0:Zc?0:Zf00:Zh?0:Zj00:Zo?0:[000:[1?0:[200:[PA0:[[80:[]?0:[_80:[a00:[f80:[g?0:[h00:`0A0:`K00:`S80:`U?"
        + "0:`V80:`X?0:`Y80:`[00:`]80:`^?0:`_00:a0<0:a1=0:aL<0:aM=0:ai<0:aj=0:bD<0:bE=0:ba<0:bb=0:c<<0:c==0:cX<"
        + "0:cY=0:d4<0:d5=0:dP<0:dQ=0:dm<0:dn=0:eH<0:eI=0:ee<0:ef=0:f@<0:fA=0:f]<0:f^=0:g8<0:g9=0:gT<0:gU=0:h0<"
        + "0:h1=0:hL<0:hM=0:hi<0:hj=0:iD<0:iE=0:ia<0:ib=0:j<<0:j==0:jX<0:jY=0:k4<0:k5=0:kP<0:kQ=0:km<0:kn=0:lH<"
        + "0:lI=0:le<0:lf=0:m@<0:mA=0:m]<0:m^=0:n8<0:n9=0:nT<0:nU=0:o0<0:o1=0:oL<0:oM=0:oi<0:oj=0:pD<0:pE=0:pa<"
        + "0:pb=0;0<<0;0==0;0X<0;0Y=0;14<0;15=0;1P<0;1Q=0;1m<0;1n=0;2H<0;2I=0;2e<0;2f=0;3@<0;3A=0;3]<0;3^=0;48<"
        + "0;49=0;4T<0;4U=0;50<0;51=0;5L<0;5M=0;5i<0;5j=0;6D<0;6E=0;6a<0;6b=0;7<<0;7==0;7X<0;7Y=0;84<0;85=0;8P<"
        + "0;8Q=0;8m<0;8n=0;9H<0;9I=0;9e<0;9f=0;:@<0;:A=0;:]<0;:^=0;;8<0;;9=0;;T<0;;U=0;<0<0;<1=0;<L<0;<M=0;<i<"
        + "0;<j=0;=D<0;=E=0;=a<0;=b=0;><<0;>==0;>X<0;>Y=0;?4<0;?5=0;?P<0;?Q=0;?m<0;?n=0;@H<0;@I=0;@e<0;@f=0;A@<"
        + "0;AA=0;A]<0;A^=0;B8<0;B9=0;BT<0;BU=0;C0<0;C1=0;CL<0;CM=0;Ci<0;Cj=0;DD<0;DE=0;Da<0;Db=0;E<<0;E==0;EX<"
        + "0;EY=0;F4<0;F5=0;FP<0;FQ=0;Fm<0;Fn=0;GH<0;GI=0;Ge<0;Gf=0;H@<0;HA=0;H]<0;H^=0;I8<0;I9=0;IT<0;IU=0;J0<"
        + "0;J1=0;JL<0;JM=0;Ji<0;Jj=0;KD<0;KE=0;Ka<0;Kb=0;L<<0;L==0;LX<0;LY=0;M4<0;M5=0;MP<0;MQ=0;Mm<0;Mn=0;NH<"
        + "0;NI=0;Ne<0;Nf=0;O@<0;OA=0;O]<0;O^=0;P8<0;P9=0;PT<0;PU=0;Q0<0;Q1=0;QL<0;QM=0;Qi<0;Qj=0;RD<0;RE=0;Ra<"
        + "0;Rb=0;S<<0;S==0;SX<0;SY=0;T4<0;T5=0;TP<0;TQ=0;Tm<0;Tn=0;UH<0;UI=0;Ue<0;Uf=0;V@<0;VA=0;V]<0;V^=0;W8<"
        + "0;W9=0;WT<0;WU=0;X0<0;X1=0;XL<0;XM=0;Xi<0;Xj=0;YD<0;YE=0;Ya<0;Yb=0;Z<<0;Z==0;ZX<0;ZY=0;[4<0;[5=0;[P<"
        + "0;[Q=0;[m<0;[n=0;]H<0;]I=0;]e<0;]f=0;^@<0;^A=0;^]<0;^^=0;_8<0;_9=0;_T<0;_U=0;`0<0;`1=0;`L<0;`M=0;`i<"
        + "0;`j=0;aD<0;aE=0;aa<0;ab=0;b<<0;b==0;bX<0;bY=0;c4<0;c5=0;cP<0;cQ=0;cm<0;cn=0;dH<0;dI=0;de<0;df=0;e@<"
        + "0;eA=0;e]<0;e^=0;f8<0;f9=0;fT<0;fU=0;g0<0;g1=0;gL<0;gM=0;gi<0;gj=0;hD<0;hE=0;ha<0;hb=0;i<<0;i==0;iX<"
        + "0;iY=0;j4<0;j5=0;jP<0;jQ=0;jm<0;jn=0;kH<0;kI=0;ke<0;kf=0;l@<0;lA=0;l]<0;l^=0;m8<0;m9=0;mT<0;mU=0;n0<"
        + "0;n1=0;nL<0;nM=0;ni<0;nj=0;oD<0;oE=0;oa<0;ob=0;p<<0;p==0;pX<0;pY=0<04<0<05=0<0P<0<0Q=0<0m<0<0n=0<1H<"
        + "0<1I=0<1e<0<1f=0<2@<0<2A=0<2]<0<2^=0<38<0<39=0<3T<0<3U=0<40<0<41=0<4L<0<4M=0<4i<0<4j=0<5D<0<5E=0<5a<"
        + "0<5b=0<6<<0<6==0<6X<0<6Y=0<74<0<75=0<7P<0<7Q=0<7m<0<7n=0<8H<0<8I=0<8e<0<8f=0<9@<0<9A=0<9]<0<9^=0<:8<"
        + "0<:9=0<:T<0<:U=0<;0<0<;1=0<;L<0<;M=0<;i<0<;j=0<<D<0<<E=0<<a<0<<b=0<=<<0<===0<=X<0<=Y=0<>4<0<>5=0<>P<"
        + "0<>Q=0<>m<0<>n=0<?H<0<?I=0<?e<0<?f=0<@@<0<@A=0<@]<0<@^=0<A8<0<A9=0<AT<0<AU=0<B0<0<B1=0<BL<0<BM=0<Bi<"
        + "0<Bj=0<CD<0<CE=0<Ca<0<Cb=0<D<<0<D==0<DX<0<DY=0<E4<0<E5=0<EP<0<EQ=0<Em<0<En=0<FH<0<FI=0<Fe<0<Ff=0<G@<"
        + "0<GA=0<G]<0<G^=0<H8<0<H9=0<HT<0<HU=0<I0<0<I1=0<IL<0<IM=0<Ii<0<Ij=0<JD<0<JE=0<Ja<0<Jb=0<K<<0<K==0<KX<"
        + "0<KY=0<L4<0<L5=0<LP<0<LQ=0<Lm<0<Ln=0<MH<0<MI=0<Me<0<Mf=0<N@<0<NA=0<N]<0<N^=0<O8<0<O9=0<OT<0<OU=0<P0<"
        + "0<P1=0<PL<0<PM=0<Pi<0<Pj=0<QD<0<QE=0<Qa<0<Qb=0<R<<0<R==0<RX<0<RY=0<S4<0<S5=0<SP<0<SQ=0<Sm<0<Sn=0<TH<"
        + "0<TI=0<Te<0<Tf=0<U@<0<UA=0<U]<0<U^=0<V8<0<V9=0<VT<0<VU=0<W0<0<W1=0<WL<0<WM=0<Wi<0<Wj=0<XD<0<XE=0<Xa<"
        + "0<Xb=0<Y<<0<Y==0<YX<0<YY=0<Z4<0<Z5=0<ZP<0<ZQ=0<Zm<0<Zn=0<[H<0<[I=0<[e<0<[f=0<]@<0<]A=0<]]<0<]^=0<^8<"
        + "0<^9=0<^T<0<^U=0<_0<0<_1=0<_L<0<_M=0<_i<0<_j=0<`D<0<`E=0<`a<0<`b=0<a<<0<a==0<aX<0<aY=0<b4<0<b5=0<bP<"
        + "0<bQ=0<bm<0<bn=0<cH<0<cI=0<ce<0<cf=0<d@<0<dA=0<d]<0<d^=0<e8<0<e9=0<eT<0<eU=0<f0<0<f1=0<fL<0<fM=0<fi<"
        + "0<fj=0<gD<0<gE=0<ga<0<gb=0<h<<0<h==0<hX<0<hY=0<i4<0<i5=0<iP<0<iQ=0<im<0<in=0<jH<0<jI=0<je<0<jf=0<k@<"
        + "0<kA=0<k]<0<k^=0<l8<0<l9=0<lT<0<lU=0<m0<0<m1=0<mL<0<mM=0<mi<0<mj=0<nD<0<nE=0<na<0<nb=0<o<<0<o==0<oX<"
        + "0<oY=0<p4<0<p5=0<pP<0<pQ=0<pm<0<pn=0=0H<0=0I=0=0e<0=0f=0=1@<0=1A=0=1]<0=1^=0=28<0=29=0=2T<0=2U=0=30<"
        + "0=31=0=3L<0=3M=0=3i<0=3j=0=4D<0=4E=0=4a<0=4b=0=5<<0=5==0=5X<0=5Y=0=64<0=65=0=6P<0=6Q=0=6m<0=6n=0=7H<"
        + "0=7I=0=7e<0=7f=0=8@<0=8A=0=8]<0=8^=0=98<0=99=0=9T<0=9U=0=:0<0=:1=0=:L<0=:M=0=:i<0=:j=0=;D<0=;E=0=;a<"
        + "0=;b=0=<<<0=<==0=<X<0=<Y=0==4<0==5=0==P<0==Q=0==m<0==n=0=>H<0=>I=0=>e<0=>f=0=?@<0=?A=0=?]<0=?^=0=@8<"
        + "0=@9=0=@T<0=@U=0=A0<0=A1=0=AL<0=AM=0=Ai<0=Aj=0=BD<0=BE=0=Ba<0=Bb=0=C<<0=C==0=CX<0=CY=0=D4<0=D5=0=DP<"
        + "0=DQ=0=Dm<0=Dn=0=EH<0=EI=0=Ee<0=Ef=0=F@<0=FA=0=F]<0=F^=0=G8<0=G9=0=GT<0=GU=0=H0<0=H1=0=HL<0=HM=0=Hi<"
        + "0=Hj=0=ID<0=IE=0=Ia<0=Ib=0=J<<0=J==0=JX<0=JY=0=K4<0=K5=0=KP<0=KQ=0=Km<0=Kn=0=LH<0=LI=0=Le<0=Lf=0=M@<"
        + "0=MA=0=M]<0=M^=0=N8<0=N9=0=NT00=Na:0=O700=O;;0=Om00=P030>0000?]N?0?]O00?i0?0?i@00?iP?0?ia00?lp30?m00"
        + "0?oN?0?oP00?pa30?pm00@7n?0@7o00@;P?0@;Q00@=g?0@=l00@X0A0@X1?0@X400@X5?0@X700@X<?0@X@A0@XD00@XEA0@XH0"
        + "0@XIA0@Xg00@Xi?0@Xl00@Xp?0@Y000@[U?0@[W00@eT?0@eX00@fY?0@f_00@k[?0@k^00@lk?0@m000@n6?0@nA00@o2?0@o60"
        + "0A0080A01?0A0280A0300A0i?0A1700A1a?0A1b00A1d?0A1f00A1p?0A2280A2300A2a80A2d?0A2h80A2j?0A2l00A2n70A2o0"
        + "0A32?0A3300A3=70A3>00A40?0A43A0A4W?0A4]80A4^?0A4f00A54A0A5580A57A0A5800A5d?0A5e00A60?0A6280A6300A6d8"
        + "0A6g?0A6p80A70?0A7100A7270A7400A79?0A7=00A7>80A7??0A7@00A8]80A8`?0A8c80A8e?0A8i00A8o?0A8p00A91?0A920"
        + "0A;O?0A;P80A;S?0A;[00A<0?0A<280A<400A<l?0A<n00A<o?0A<p80A=0?0A=180A=500A=780A=900A=;80A==?0A=>00A=G?"
        + "0A=H00A=R80A=T00A=V?0A=^00A=a?0A=f00A>0A0A>:00A>;A0A><00A>>A0A>?00A>@A0A>g00A>i?0A>j80A>l?0A?100A?2?"
        + "0A?300A?5?0A?600A?7?0A?:80A?;00A?<80A?>?0A?A70A?B?0A?C00A?Q?0A?S00A@f80A@i?0AA080AA2?0AA580AA6?0AA70"
        + "0AAN?0AAO00ABa?0ABb80ABd?0ABj80ABk?0ABl80ABn?0ABo80ABp?0AC180AC2?0AC400AF`?0AFa80AFc?0AFg00AFi80AFm?"
        + "0AFo80AFp?0AG100AGL?0AGN00AHa80AHd?0AHl80AHn?0AHo80AHp?0AI100AJ[?0AJ]80AJ^?0AJ_80AJa?0AJi00ALM?0ALN8"
        + "0ALO?0ALP00ALR?0ALV80ALW?0AL]00AP]80AP`?0APi80APj?0APl00AT0A0AT700AT9A0AT:00AT<A0ATD00ATEA0ATG00ATHA"
        + "0ATa?0ATb80ATg00ATh80ATj00ATl?0ATp70AU080AU170AU280AU3?0AU400AWA80AWD?0AWH00AWJ?0AWL80AWP?0AWQ00AWT8"
        + "0AWU00AX0A0AX1?0AX;A0AXd?0AXj80AXk00AXl?0AXp00AY7?0AY800AY@A0AYA?0AYG80AYI?0AYLA0AZ470AZ:?0AZG80AZH?"
        + "0AZJ00A^P?0A^Q80A^R?0A^U80A^V?0A^W80A^X00Aa`80Aaa?0Aah00Aai?0Aao80Aap?0Ab000AcB?0AcX00AcY80AcZ?0Acb8"
        + "0Acc?0Ace80Acf?0Ach00Aeb?0Aeh00Aek?0Ael00Aem?0Aeo00Aep?0Af670Af7?0Af800Ag:80Ag?00Ag@?0AgB00AgC80AgE?"
        + "0AgF80AgG?0AgH00Ald?0Alf80Alh00Am0?0Am270Am380Am4A0AmA00AmBA0Ame80Amg?0Aml00Amo80An0?0An300AnJ?0AnK0"
        + "0C@a30CA0?0CA100CA7?0CAF00F4N?0F4Z80F4^?0F4a00F[a?0F[f00F]a?0F]h00Fn??0Fn@00FnA80Fo800Fo??0FoC00FpT?"
        + "0FpU00Fpa?0Fpc00KcM?0KcO00KcP30KcT00Lm0?0Lm_00Lma?0Ln700M5U?0M5Z00M5^?0M5d30M5l?0M6300M65?0M6<00M6Z?"
        + "0M6_00M92?0M9500MX0?0MXh00MXl?0MY^00MYf?0MYg00MZ4?0MZ500MZK?0MZP00MZQ?0MZa00N00?0N0700N08?0N0I00N0K?"
        + "0N0R00N0S?0N0U00N0V?0N0[00N2??0N2@00N4a?0N4h00N:_?0N:`00N;]?0N;a00NC]?0NCa00NG_?0NGa00NKS?0NKT00NKV?"
        + "0NKW00NK_?0NKa00NKf?0NKg00NS@?0NSG00NU4?0NU;00O04>0O0500O0]>0O0a00O2D>0O2P00O2`>0O2b00O30>0O3100O3?>"
        + "0O3A00O3g>0O4000O5a>0O5c00O5o>0O6000O6>>0O6?00O6A>0O6K00O6_>0O7V60O8000O81>0O8@00O8J>0O8K00O8`>0O8a0"
        + "0O8c>0O8l00O8m>0O9000O99>0O9P00O9V>0O<R00O<T>0O>D00O>F>0O>H00O>I>0O>L00O>N>0O?b00O?d>0O?g00O?h>0O?l?"
        + "0O@0>0OCo00OCp>0ODo00OE9>0OE?00OE@>0OEX00OE`>0OEb00OEd>0OEl00OF7>0OF800OF:>0OF>00OF@>0OFA00OFE>0OFG0"
        + "0OFT>0OFV00OFX>0OFY00OFb>0OFd00OFm>0OFn00OG2>0OG500OGA>0OGD00OGL>0OGO00OGQ>0OGR00OGS>0OGT00OGX>0OGY0"
        + "0OG`>0OGa00OGd>0OGe00OGk>0OI@00OJ0>0OK600OK;>0OKC00OKE>0OKV00OKY>0OKZ00OK[>0OKb00OKd>0OL000OOJ>0OP00"
        + "0OP<>0OP@00OQ8>0OQ@00OQJ>0OQP00OR8>0OR@00OR_>0ORa00ORm>0OS000OS2>0OS@00OSI>0OT000OT<>0OTl00OTm>0OU60"
        + "0OU7>0OX000OYH>0OYP00OY_>0O]000Oa0>0Opo03P0033P0P?3P2033P40?3P7a33Q000"
    }
    return swcGcbTable
}
function swcGcb(c) {
    if (c < 128) {
        if (c == 13) { return 1 }
        if (c == 10) { return 2 }
        if (c < 32 || c == 127) { return 3 }
        return 0
    }
    if (c > 1114111) { return 3 }
    var t = swcGcbTab()
    return jcDig(t, jcRun(t, c, 5) * 5 + 4)
}
function swcIsExt(k) { return k == 4 || k == 15 || k == 16 }
// Whether the rules GB3 ... GB999 break between classes p and k; ri is the run of
// Regional_Indicators ending at p, emo the GB11 state (1 after a pictograph and
// its Extends, 2 after the ZWJ that follows them), cb the GB9c state (1 after a
// consonant and its Extends, 2 once a Linker followed).
function swcBreak(p, k, ri, emo, cb) {
    if (p == 1 && k == 2) { return false }
    if (p >= 1 && p <= 3) { return true }
    if (k >= 1 && k <= 3) { return true }
    if (p == 9 && (k == 9 || k == 10 || k == 12 || k == 13)) { return false }
    if ((p == 10 || p == 12) && (k == 10 || k == 11)) { return false }
    if ((p == 11 || p == 13) && k == 11) { return false }
    if (swcIsExt(k) || k == 5 || k == 8) { return false }
    if (p == 7) { return false }
    if (cb == 2 && k == 17) { return false }
    if (emo == 2 && k == 14) { return false }
    if (p == 6 && k == 6 && ri % 2 == 1) { return false }
    return true
}
// The UTF-16 offset at which each cluster of s starts, then s.length.
function swcBounds(s) {
    var out = []
    var i = 0
    var p = 0 - 1
    var ri = 0
    var emo = 0
    var cb = 0
    while (i < s.length) {
        var c = swcCodeAt(s, i)
        var k = swcGcb(c)
        if (p < 0 || swcBreak(p, k, ri, emo, cb)) { out.push(i) }
        if (k == 6) { ri = ri + 1 } else { ri = 0 }
        if (k == 14) { emo = 1 }
        else if (emo == 1 && swcIsExt(k)) { emo = 1 }
        else if (emo == 1 && k == 5) { emo = 2 }
        else { emo = 0 }
        if (k == 17) { cb = 1 }
        else if (cb > 0 && k == 16) { cb = 2 }
        else if (cb > 0 && (k == 15 || k == 5)) { cb = cb + 0 }
        else { cb = 0 }
        p = k
        if (c > 65535) { i = i + 2 } else { i = i + 1 }
    }
    out.push(s.length)
    return out
}
// Whether s is ASCII with no CR LF pair, i.e. one cluster per code unit.
function swcPlain(s) {
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        if (c > 127) { return false }
        if (c == 13 && i + 1 < s.length && s.charCodeAt(i + 1) == 10) { return false }
    }
    return true
}
function swcGraphemeCount(s) {
    if (swcPlain(s)) { return s.length }
    return swcBounds(s).length - 1
}
function swcGraphemes(s) {
    var out = []
    if (swcPlain(s)) {
        for (var i = 0; i < s.length; i++) { out.push(s.charAt(i)) }
        return out
    }
    var b = swcBounds(s)
    for (var j = 0; j + 1 < b.length; j++) { out.push(s.substring(b[j], b[j + 1])) }
    return out
}

// ----- canonical equivalence and ordering -----
// == compares by CANONICAL EQUIVALENCE, so the precomposed and the decomposed
// spelling of a letter are the same string. The decomposition table covers the
// Latin-1 precomposed letters, which is the range real programs mix; anything
// outside it compares by code units. Twin: swcNFD in abnf/jsrtswift.go.
var swcNfdBase = "AAAAAA\x00CEEEEIIII\x00NOOOOO\x00\x00UUUUY\x00\x00aaaaaa\x00ceeeeiiii\x00nooooo\x00\x00uuuuy\x00y"
var swcNfdMark = [0x300, 0x301, 0x302, 0x303, 0x308, 0x30A, 0, 0x327,
                  0x300, 0x301, 0x302, 0x308, 0x300, 0x301, 0x302, 0x308,
                  0, 0x303, 0x300, 0x301, 0x302, 0x303, 0x308, 0,
                  0, 0x300, 0x301, 0x302, 0x308, 0x301, 0, 0,
                  0x300, 0x301, 0x302, 0x303, 0x308, 0x30A, 0, 0x327,
                  0x300, 0x301, 0x302, 0x308, 0x300, 0x301, 0x302, 0x308,
                  0, 0x303, 0x300, 0x301, 0x302, 0x303, 0x308, 0,
                  0, 0x300, 0x301, 0x302, 0x308, 0x301, 0, 0x308]
function swcNFD(str) {
    var out = ""
    for (var i = 0; i < str.length; i++) {
        var cc = str.charCodeAt(i)
        if (cc >= 0xC0 && cc <= 0xFF) {
            var b = swcNfdBase.charCodeAt(cc - 0xC0)
            if (b != 0) {
                out = out + String.fromCharCode(b) + String.fromCharCode(swcNfdMark[cc - 0xC0])
                continue
            }
        }
        out = out + str.charAt(i)
    }
    return out
}
// The same table read backwards: a base letter and its mark composed, for the
// ORDERING, which Swift defines on the NFC form ("e\u{301}" sorts as U+00E9).
function swcNFC(str) {
    var d = swcNFD(str)
    var out = ""
    var i = 0
    while (i < d.length) {
        var c = d.charCodeAt(i)
        if (i + 1 < d.length) {
            var m = d.charCodeAt(i + 1)
            var hit = 0 - 1
            for (var q = 0; q < 64; q++) {
                if (swcNfdBase.charCodeAt(q) == c && swcNfdMark[q] == m && m != 0) { hit = q }
            }
            if (hit >= 0) {
                out = out + String.fromCharCode(0xC0 + hit)
                i = i + 2
                continue
            }
        }
        out = out + d.charAt(i)
        i = i + 1
    }
    return out
}
function swcHasNonAscii(str) {
    for (var i = 0; i < str.length; i++) { if (str.charCodeAt(i) > 127) { return true } }
    return false
}
function swcStrEq(a, b) {
    if (a === b) { return true }
    if (!swcHasNonAscii(a) || !swcHasNonAscii(b)) { return false }
    return swcNFD(a) === swcNFD(b)
}
// Swift's String ordering: the NFC forms, scalar by scalar (not UTF-16 unit by
// unit, which sorts an astral scalar below U+E000). -1, 0 or 1.
function swcStrCmp(a, b) {
    var x = a
    var y = b
    if (swcHasNonAscii(a) || swcHasNonAscii(b)) { x = swcNFC(a); y = swcNFC(b) }
    var i = 0
    var j = 0
    while (i < x.length && j < y.length) {
        var cx = swcCodeAt(x, i)
        var cy = swcCodeAt(y, j)
        if (cx < cy) { return 0 - 1 }
        if (cx > cy) { return 1 }
        if (cx > 65535) { i = i + 2 } else { i = i + 1 }
        if (cy > 65535) { j = j + 2 } else { j = j + 1 }
    }
    if (i < x.length) { return 1 }
    if (j < y.length) { return 0 - 1 }
    return 0
}
// hasPrefix / hasSuffix / contains compare CHARACTERS, so "e\u{301}" neither
// starts with nor contains "e" (swiftc 6.1.2).
function swcHasPrefix(s, p) {
    var cs = swcGraphemes(s)
    var cp = swcGraphemes(p)
    if (cp.length > cs.length) { return false }
    for (var i = 0; i < cp.length; i++) { if (!swcStrEq(cs[i], cp[i])) { return false } }
    return true
}
function swcHasSuffix(s, p) {
    var cs = swcGraphemes(s)
    var cp = swcGraphemes(p)
    var off = cs.length - cp.length
    if (off < 0) { return false }
    for (var i = 0; i < cp.length; i++) { if (!swcStrEq(cs[off + i], cp[i])) { return false } }
    return true
}
function swcContains(s, p) {
    var cs = swcGraphemes(s)
    var cp = swcGraphemes(p)
    for (var at = 0; at + cp.length <= cs.length; at++) {
        var ok = true
        for (var i = 0; i < cp.length && ok; i++) { if (!swcStrEq(cs[at + i], cp[i])) { ok = false } }
        if (ok) { return true }
    }
    return false
}
