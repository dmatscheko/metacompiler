package abnf

import "unicode/utf8"

// A Ruby String's extended grapheme clusters (UAX #29 as MRI 2.6, Unicode
// 12.1, segments them): the Go twin of the grapheme section of
// languages/lib/ruby-str.js, over the same table (that file describes the
// encoding and the classes; abnf/ruby_gcb_test.go keeps the two equal).

const rubyGcbTable = "" +
	"00003000:2000;3000=1000>3000P0001p3002P0002Y>002Z0002^3002_>002`000<0400=a000B3400B:000FA400Fo000Fp4" +
	"00G0000G1400G3000G4400G6000G7400G8000H0700H6000H@400HK000HL300HM000I;400IP000Ia400Ib000KF400KM700KN0" +
	"00KO400KU000KW400KY000KZ400K_000L?700L@000LA400LB000La400M;000NV400Nb000O[400Oe000On400Oo000PF400PJ0" +
	"00PK400PT000PU400PX000PY400P_000QI400QL000SC400SR700SS400T3800T4000Tk400Tl800Tm400Tn000To800U1400U98" +
	"00U=400U>800U@000UA400UH000UR400UT000V1400V2800V4000Vm400Vn000Vo400Vp800W1400W5000W7800W9000W;800W=4" +
	"00W>000WG400WH000WR400WT000Wo400Wp000X1400X3800X4000Xm400Xn000Xo800Y1400Y3000Y7400Y9000Y;400Y>000YA4" +
	"00YB000Ya400Yc000Yf400Yg000Z1400Z3800Z4000Zm400Zn000Zo800[1400[6000[7400[9800[:000[;800[=400[>000[R4" +
	"00[T000[k400]0000]1400]2800]4000]m400]n000]o400^0800^1400^5000^7800^9000^;800^=400^>000^F400^H000^R4" +
	"00^T000_2400_3000_o400_p800`0400`1800`3000`6800`9000`:800`=400`>000`G400`H000a0400a1800a4400a5000ao4" +
	"00b1800b5000b6400b9000b:400b>000bE400bG000bR400bT000c1400c2800c4000cm400cn000co800cp400d0800d2400d38" +
	"00d5000d6400d7800d9000d:800d<400d>000dE400dG000dR400dT000e0400e2800e4000el400en000eo400ep800f1400f50" +
	"00f6800f9000f:800f=400f>700f?000fG400fH000fR400fT000g2800g4000h:400h;000h?400h@800hB400hE000hF400hG0" +
	"00hH800hO400hP000hc800he000ib400ic000id800ie400il000j7400j?000kb400kc000kd800ke400kn000l8400l>000mH4" +
	"00mJ000mf400mg000mh400mi000mj400mk000mo800n0000nb400np800o0400o5000o6400o8000o=400oH000oI400on000p64" +
	"00p70010^4010b8010c4010i0010j4010l8010n4010p0011F8011H4011J0011N4011Q0011b4011f001224012300124801254" +
	"01270012=4012>0012M4012N001409015P:016X;0180001=M401=P001LB401LE001Lc401Lf001MB401MD001Mc401Me001Ne4" +
	"01Ng801Nh401No801O6401O7801O9401OD001OM401ON001P;401P>301P?001R5401R7001RY401RZ001TP401TS801TW401TY8" +
	"01T]001Ta801Tc401Td801Tj401Tm001XG401XI801XK401XL001YE801YF401YG801YH401YO001YP401YQ001YR401YS001YU4" +
	"01Y^801Yd401Yn001Yp401Z0001Za401Zp001]0401]4801]5001]e401]l801]m401]n801^2401^3801^5001^[401^e001_04" +
	"01_2801_3001_Q801_R401_V801_X401_Z801_[401__001`V401`W801`X401`Z801`^401`_801``401`c801`e001aT801a]4" +
	"01ae801ag401ai001d@401dC001dD401dQ801dR401dY001d^401d_001de401df001dh801di401dk001h0401hk001hl401i00" +
	"020;3020<4020=5020>3020@0020X3020`0020m>020n00219>021:0021P3021a0023@4023b0024R>024S0024j>024k0026D>" +
	"026J0026Y>026[002<J>02<L002<X>02<Y002>8>02>9002??>02?@002?Y>02?e002?i>02?l002C2>02C3002FZ>02F]002Fg>" +
	"02Fh002G0>02G1002Gl>02Gp002H0>02H6002H7>02HC002HD>02J6002J@>02L6002L8>02LC002LD>02LE002LF>02LG002LM>" +
	"02LN002LQ>02LR002LX>02LY002Ld>02Lf002M4>02M5002M7>02M8002M<>02M=002M>>02M?002MC>02MF002MG>02MH002MS>" +
	"02MX002NE>02NH002NQ>02NR002Na>02Nb002Np>02O0002Te>02Tg002]5>02]8002]K>02]M002^@>02^A002^E>02^F002d`4" +
	"02dc002fp402g0002hP402i00030Z4030a>030b0030n>030o0032I4032K003:G>03:H003:I>03:J00:I`40:Id00:Ie40:Io0" +
	"0:JN40:JP00:Ka40:Kc00:P240:P300:P640:P700:P;40:P<00:PS80:PU40:PW80:PX00:R080:R200:Re80:S440:S600:SP4" +
	"0:Sc00:Sp40:T000:TV40:T_00:U740:UB80:UD00:UP90:Un00:V040:V380:V400:Vd40:Ve80:Vg40:Vk80:Vm40:Vo80:W10" +
	"0:WU40:WV00:XY40:X`80:Xb40:Xd80:Xf40:Xh00:Y340:Y400:Y<40:Y=80:Y>00:Ym40:Yn00:Za40:Zb00:Zc40:Zf00:Zh4" +
	"0:Zj00:Zo40:[000:[140:[200:[[80:[]40:[_80:[a00:[f80:[g40:[h00:`S80:`U40:`V80:`X40:`Y80:`[00:`]80:`^4" +
	"0:`_00:a0<0:a1=0:aL<0:aM=0:ai<0:aj=0:bD<0:bE=0:ba<0:bb=0:c<<0:c==0:cX<0:cY=0:d4<0:d5=0:dP<0:dQ=0:dm<" +
	"0:dn=0:eH<0:eI=0:ee<0:ef=0:f@<0:fA=0:f]<0:f^=0:g8<0:g9=0:gT<0:gU=0:h0<0:h1=0:hL<0:hM=0:hi<0:hj=0:iD<" +
	"0:iE=0:ia<0:ib=0:j<<0:j==0:jX<0:jY=0:k4<0:k5=0:kP<0:kQ=0:km<0:kn=0:lH<0:lI=0:le<0:lf=0:m@<0:mA=0:m]<" +
	"0:m^=0:n8<0:n9=0:nT<0:nU=0:o0<0:o1=0:oL<0:oM=0:oi<0:oj=0:pD<0:pE=0:pa<0:pb=0;0<<0;0==0;0X<0;0Y=0;14<" +
	"0;15=0;1P<0;1Q=0;1m<0;1n=0;2H<0;2I=0;2e<0;2f=0;3@<0;3A=0;3]<0;3^=0;48<0;49=0;4T<0;4U=0;50<0;51=0;5L<" +
	"0;5M=0;5i<0;5j=0;6D<0;6E=0;6a<0;6b=0;7<<0;7==0;7X<0;7Y=0;84<0;85=0;8P<0;8Q=0;8m<0;8n=0;9H<0;9I=0;9e<" +
	"0;9f=0;:@<0;:A=0;:]<0;:^=0;;8<0;;9=0;;T<0;;U=0;<0<0;<1=0;<L<0;<M=0;<i<0;<j=0;=D<0;=E=0;=a<0;=b=0;><<" +
	"0;>==0;>X<0;>Y=0;?4<0;?5=0;?P<0;?Q=0;?m<0;?n=0;@H<0;@I=0;@e<0;@f=0;A@<0;AA=0;A]<0;A^=0;B8<0;B9=0;BT<" +
	"0;BU=0;C0<0;C1=0;CL<0;CM=0;Ci<0;Cj=0;DD<0;DE=0;Da<0;Db=0;E<<0;E==0;EX<0;EY=0;F4<0;F5=0;FP<0;FQ=0;Fm<" +
	"0;Fn=0;GH<0;GI=0;Ge<0;Gf=0;H@<0;HA=0;H]<0;H^=0;I8<0;I9=0;IT<0;IU=0;J0<0;J1=0;JL<0;JM=0;Ji<0;Jj=0;KD<" +
	"0;KE=0;Ka<0;Kb=0;L<<0;L==0;LX<0;LY=0;M4<0;M5=0;MP<0;MQ=0;Mm<0;Mn=0;NH<0;NI=0;Ne<0;Nf=0;O@<0;OA=0;O]<" +
	"0;O^=0;P8<0;P9=0;PT<0;PU=0;Q0<0;Q1=0;QL<0;QM=0;Qi<0;Qj=0;RD<0;RE=0;Ra<0;Rb=0;S<<0;S==0;SX<0;SY=0;T4<" +
	"0;T5=0;TP<0;TQ=0;Tm<0;Tn=0;UH<0;UI=0;Ue<0;Uf=0;V@<0;VA=0;V]<0;V^=0;W8<0;W9=0;WT<0;WU=0;X0<0;X1=0;XL<" +
	"0;XM=0;Xi<0;Xj=0;YD<0;YE=0;Ya<0;Yb=0;Z<<0;Z==0;ZX<0;ZY=0;[4<0;[5=0;[P<0;[Q=0;[m<0;[n=0;]H<0;]I=0;]e<" +
	"0;]f=0;^@<0;^A=0;^]<0;^^=0;_8<0;_9=0;_T<0;_U=0;`0<0;`1=0;`L<0;`M=0;`i<0;`j=0;aD<0;aE=0;aa<0;ab=0;b<<" +
	"0;b==0;bX<0;bY=0;c4<0;c5=0;cP<0;cQ=0;cm<0;cn=0;dH<0;dI=0;de<0;df=0;e@<0;eA=0;e]<0;e^=0;f8<0;f9=0;fT<" +
	"0;fU=0;g0<0;g1=0;gL<0;gM=0;gi<0;gj=0;hD<0;hE=0;ha<0;hb=0;i<<0;i==0;iX<0;iY=0;j4<0;j5=0;jP<0;jQ=0;jm<" +
	"0;jn=0;kH<0;kI=0;ke<0;kf=0;l@<0;lA=0;l]<0;l^=0;m8<0;m9=0;mT<0;mU=0;n0<0;n1=0;nL<0;nM=0;ni<0;nj=0;oD<" +
	"0;oE=0;oa<0;ob=0;p<<0;p==0;pX<0;pY=0<04<0<05=0<0P<0<0Q=0<0m<0<0n=0<1H<0<1I=0<1e<0<1f=0<2@<0<2A=0<2]<" +
	"0<2^=0<38<0<39=0<3T<0<3U=0<40<0<41=0<4L<0<4M=0<4i<0<4j=0<5D<0<5E=0<5a<0<5b=0<6<<0<6==0<6X<0<6Y=0<74<" +
	"0<75=0<7P<0<7Q=0<7m<0<7n=0<8H<0<8I=0<8e<0<8f=0<9@<0<9A=0<9]<0<9^=0<:8<0<:9=0<:T<0<:U=0<;0<0<;1=0<;L<" +
	"0<;M=0<;i<0<;j=0<<D<0<<E=0<<a<0<<b=0<=<<0<===0<=X<0<=Y=0<>4<0<>5=0<>P<0<>Q=0<>m<0<>n=0<?H<0<?I=0<?e<" +
	"0<?f=0<@@<0<@A=0<@]<0<@^=0<A8<0<A9=0<AT<0<AU=0<B0<0<B1=0<BL<0<BM=0<Bi<0<Bj=0<CD<0<CE=0<Ca<0<Cb=0<D<<" +
	"0<D==0<DX<0<DY=0<E4<0<E5=0<EP<0<EQ=0<Em<0<En=0<FH<0<FI=0<Fe<0<Ff=0<G@<0<GA=0<G]<0<G^=0<H8<0<H9=0<HT<" +
	"0<HU=0<I0<0<I1=0<IL<0<IM=0<Ii<0<Ij=0<JD<0<JE=0<Ja<0<Jb=0<K<<0<K==0<KX<0<KY=0<L4<0<L5=0<LP<0<LQ=0<Lm<" +
	"0<Ln=0<MH<0<MI=0<Me<0<Mf=0<N@<0<NA=0<N]<0<N^=0<O8<0<O9=0<OT<0<OU=0<P0<0<P1=0<PL<0<PM=0<Pi<0<Pj=0<QD<" +
	"0<QE=0<Qa<0<Qb=0<R<<0<R==0<RX<0<RY=0<S4<0<S5=0<SP<0<SQ=0<Sm<0<Sn=0<TH<0<TI=0<Te<0<Tf=0<U@<0<UA=0<U]<" +
	"0<U^=0<V8<0<V9=0<VT<0<VU=0<W0<0<W1=0<WL<0<WM=0<Wi<0<Wj=0<XD<0<XE=0<Xa<0<Xb=0<Y<<0<Y==0<YX<0<YY=0<Z4<" +
	"0<Z5=0<ZP<0<ZQ=0<Zm<0<Zn=0<[H<0<[I=0<[e<0<[f=0<]@<0<]A=0<]]<0<]^=0<^8<0<^9=0<^T<0<^U=0<_0<0<_1=0<_L<" +
	"0<_M=0<_i<0<_j=0<`D<0<`E=0<`a<0<`b=0<a<<0<a==0<aX<0<aY=0<b4<0<b5=0<bP<0<bQ=0<bm<0<bn=0<cH<0<cI=0<ce<" +
	"0<cf=0<d@<0<dA=0<d]<0<d^=0<e8<0<e9=0<eT<0<eU=0<f0<0<f1=0<fL<0<fM=0<fi<0<fj=0<gD<0<gE=0<ga<0<gb=0<h<<" +
	"0<h==0<hX<0<hY=0<i4<0<i5=0<iP<0<iQ=0<im<0<in=0<jH<0<jI=0<je<0<jf=0<k@<0<kA=0<k]<0<k^=0<l8<0<l9=0<lT<" +
	"0<lU=0<m0<0<m1=0<mL<0<mM=0<mi<0<mj=0<nD<0<nE=0<na<0<nb=0<o<<0<o==0<oX<0<oY=0<p4<0<p5=0<pP<0<pQ=0<pm<" +
	"0<pn=0=0H<0=0I=0=0e<0=0f=0=1@<0=1A=0=1]<0=1^=0=28<0=29=0=2T<0=2U=0=30<0=31=0=3L<0=3M=0=3i<0=3j=0=4D<" +
	"0=4E=0=4a<0=4b=0=5<<0=5==0=5X<0=5Y=0=64<0=65=0=6P<0=6Q=0=6m<0=6n=0=7H<0=7I=0=7e<0=7f=0=8@<0=8A=0=8]<" +
	"0=8^=0=98<0=99=0=9T<0=9U=0=:0<0=:1=0=:L<0=:M=0=:i<0=:j=0=;D<0=;E=0=;a<0=;b=0=<<<0=<==0=<X<0=<Y=0==4<" +
	"0==5=0==P<0==Q=0==m<0==n=0=>H<0=>I=0=>e<0=>f=0=?@<0=?A=0=?]<0=?^=0=@8<0=@9=0=@T<0=@U=0=A0<0=A1=0=AL<" +
	"0=AM=0=Ai<0=Aj=0=BD<0=BE=0=Ba<0=Bb=0=C<<0=C==0=CX<0=CY=0=D4<0=D5=0=DP<0=DQ=0=Dm<0=Dn=0=EH<0=EI=0=Ee<" +
	"0=Ef=0=F@<0=FA=0=F]<0=F^=0=G8<0=G9=0=GT<0=GU=0=H0<0=H1=0=HL<0=HM=0=Hi<0=Hj=0=ID<0=IE=0=Ia<0=Ib=0=J<<" +
	"0=J==0=JX<0=JY=0=K4<0=K5=0=KP<0=KQ=0=Km<0=Kn=0=LH<0=LI=0=Le<0=Lf=0=M@<0=MA=0=M]<0=M^=0=N8<0=N9=0=NT0" +
	"0=Na:0=O700=O;;0=Om00=P030>0000?]N40?]O00?i040?i@00?iP40?ia00?lp30?m000?oN40?oP00?pa30?pm00@7n40@7o0" +
	"0@;P40@;Q00@=g40@=l00@X140@X400@X540@X700@X<40@X@00@Xi40@Xl00@Xp40@Y000@[U40@[W00@eT40@eX00@n640@nA0" +
	"0A0080A0140A0280A0300A0i40A1700A1p40A2280A2300A2a80A2d40A2h80A2j40A2l00A2n70A2o00A3=70A3>00A4040A430" +
	"0A4W40A4]80A4^40A4f00A5580A5700A5d40A5e00A6040A6280A6300A6d80A6g40A6p80A7100A7270A7400A7940A7=00A8]8" +
	"0A8`40A8c80A8e40A8f80A8g40A8i00A8o40A8p00A;O40A;P80A;S40A;[00A<040A<280A<400A<l40A<n00A<o40A<p80A=04" +
	"0A=180A=500A=780A=900A=;80A=>00A=G40A=H00A=R80A=T00A=V40A=^00A=a40A=f00A@f80A@i40AA080AA240AA580AA64" +
	"0AA700AAN40AAO00ABa40ABb80ABd40ABj80ABk40ABl80ABn40ABo80ABp40AC180AC240AC400AF`40AFa80AFc40AFg00AFi8" +
	"0AFm40AFo80AFp40AG100AGL40AGN00AHa80AHd40AHl80AHn40AHo80AHp40AI100AJ[40AJ]80AJ^40AJ_80AJa40AJg80AJh4" +
	"0AJi00ALM40ALP80ALR40ALV80ALW40AL]00AP]80AP`40APi80APj40APl00AWA80AWD40AWH00AWJ40AWL80AWP40AWQ00AWT8" +
	"0AWU00AX140AX;00AXd40AXj80AXk70AXl40AXp00AY740AY800AYA40AYG80AYI40AYL00AZ470AZ:40AZG80AZH40AZJ00Aa`8" +
	"0Aaa40Aah00Aai40Aao80Aap40Ab000AcB40AcX00AcY80AcZ40Acb80Acc40Ace80Acf40Ach00Aeb40Aeh00Aek40Ael00Aem4" +
	"0Aeo00Aep40Af670Af740Af800Ag:80Ag?00Ag@40AgB00AgC80AgE40AgF80AgG40AgH00Ald40Alf80Alh00C@a30C@j00F[a4" +
	"0F[f00F]a40F]h00Fn?40Fn@00FnA80Fo800Fo?40FoC00KcM40KcO00KcP30KcT00M5U40M5V80M5W40M5Z00M5^80M5_40M5d3" +
	"0M5l40M6300M6540M6<00M6Z40M6_00M9240M9500MX040MXh00MXl40MY^00MYf40MYg00MZ440MZ500MZK40MZP00MZQ40MZa0" +
	"0N0040N0700N0840N0I00N0K40N0R00N0S40N0U00N0V40N0[00N4a40N4h00N;]40N;a00NS@40NSG00NU440NU;00O00>0O400" +
	"0O4=>0O4@00O4`>0O4a00O5]>0O5c00O5o>0O6000O6>>0O6?00O6A>0O6K00O6^>0O7V60O8000O81>0O8@00O8J>0O8K00O8`>" +
	"0O8a00O8c>0O8l00O8m>0O9000O99>0O?l40O@0>0ODo00OE6>0OI@00OJ0>0OL000OMe>0ON000OOE>0OP000OP<>0OP@00OQ8>" +
	"0OQ@00OQJ>0OQP00OR8>0OR@00OR_>0OT000OT<>0OTl00OTm>0OU600OU7>0Opo03P0033P0P43P2033P4043P7a33Q000"

// rubyGcb is the class of code point cp. Twin of rbsGcb in lib/ruby-str.js.
func rubyGcb(cp rune) int {
	dig := func(i int) int {
		c := int(rubyGcbTable[i])
		if c <= 91 {
			return c - 48
		}
		return c - 49
	}
	lo, hi := 0, len(rubyGcbTable)/5-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		st := ((dig(mid*5)*64+dig(mid*5+1))*64+dig(mid*5+2))*64 + dig(mid*5+3)
		if st <= int(cp) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return dig(lo*5 + 4)
}

// rubyGcbBreaks is whether rules GB3 ... GB999 break between classes p and k.
// Twin of rbsGcbBreaks in lib/ruby-str.js.
func rubyGcbBreaks(p, k, ri, emo int) bool {
	switch {
	case p == 1 && k == 2:
		return false
	case p >= 1 && p <= 3, k >= 1 && k <= 3:
		return true
	case p == 9 && (k == 9 || k == 10 || k == 12 || k == 13):
		return false
	case (p == 10 || p == 12) && (k == 10 || k == 11):
		return false
	case (p == 11 || p == 13) && k == 11:
		return false
	case k == 4 || k == 5 || k == 8 || p == 7:
		return false
	case emo == 2 && k == 14:
		return false
	}
	return !(p == 6 && k == 6 && ri%2 == 1)
}

// rubyGraphemes is String#grapheme_clusters of the UTF-8 bytes b; a byte that
// starts no valid character is a cluster of its own. Twin of rbsGraphemeBytes
// in lib/ruby-str.js.
func rubyGraphemes(b string) []string {
	var out []string
	start, p, ri, emo := 0, -1, 0, 0
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRuneInString(b[i:])
		k := -1
		if r != utf8.RuneError || n > 1 {
			k = rubyGcb(r)
		}
		if i > start && (k < 0 || p < 0 || rubyGcbBreaks(p, k, ri, emo)) {
			out = append(out, b[start:i])
			start = i
		}
		if k == 6 {
			ri++
		} else {
			ri = 0
		}
		switch {
		case k == 14:
			emo = 1
		case emo == 1 && k == 4:
		case emo == 1 && k == 5:
			emo = 2
		default:
			emo = 0
		}
		p = k
		i += n
	}
	if len(b) > start {
		out = append(out, b[start:])
	}
	return out
}
