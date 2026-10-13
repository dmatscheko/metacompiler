package abnf

// A Swift String as Characters - UAX #29's extended grapheme clusters, canonical
// equivalence and String ordering: the Go twin's copy of
// languages/lib/swift-chars.js, over the same table (derived from swiftc 6.1.2's
// own segmentation; that file's header describes the encoding and the classes).

const swcGcbTable = "" +
	"00003000:2000;3000=1000>3000P0001p3002P0002Y>002Z0002^3002_>002`000<0?00=a000B3?00B:000FA?00Fo000Fp?" +
	"00G0000G1?00G3000G4?00G6000G7?00G8000H0700H6000H@?00HK000HL300HM000I;?00IP000Ia?00Ib000KF?00KM700KN0" +
	"00KO?00KU000KW?00KY000KZ?00K_000L?700L@000LA?00LB000La?00M;000NV?00Nb000O[?00Oe000On?00Oo000PF?00PJ0" +
	"00PK?00PT000PU?00PX000PY?00P_000QI?00QL000R@700RB000RG?00RP000S:?00SR700SS?00T3800T4000TEA00Tk?00Tl8" +
	"00Tm?00Tn000To800U1?00U9800U=@00U>800U@000UA?00UHA00UP000UR?00UT000UiA00V0000V1?00V2800V4000VEA00VY0" +
	"00VZA00Vb000VcA00Vd000VgA00Vk000Vm?00Vn000Vo?00Vp800W1?00W5000W7800W9000W;800W=@00W>000WG?00WH000WLA" +
	"00WN000WOA00WP000WR?00WT000WaA00Wc000Wo?00Wp000X1?00X3800X4000Xm?00Xn000Xo800Y1?00Y3000Y7?00Y9000Y;?" +
	"00Y>000YA?00YB000Ya?00Yc000Yf?00Yg000Z1?00Z3800Z4000ZEA00ZY000ZZA00Zb000ZcA00Ze000ZfA00Zk000Zm?00Zn0" +
	"00Zo800[1?00[6000[7?00[9800[:000[;800[=@00[>000[R?00[T000[jA00[k?00]0000]1?00]2800]4000]EA00]Y000]ZA" +
	"00]b000]cA00]e000]fA00]k000]m?00]n000]o?00^0800^1?00^5000^7800^9000^;800^=@00^>000^E?00^H000^LA00^N0" +
	"00^OA00^P000^R?00^T000^bA00^c000_2?00_3000_o?00_p800`0?00`1800`3000`6800`9000`:800`=?00`>000`G?00`H0" +
	"00a0?00a1800a4?00a5000aEA00aY000aZA00ak000am?00an000ao?00b1800b5000b6?00b9000b:?00b=@00b>000bE?00bG0" +
	"00bHA00bK000bR?00bT000c1?00c2800c4000cm?00cn000co800cp?00d1800d2?00d3800d5000d6?00d9000d:?00d>000dE?" +
	"00dG000dR?00dT000dd800de000e0?00e2800e4000eEA00el?00en000eo?00ep800f1?00f5000f6800f9000f:800f=@00f>7" +
	"00f?000fG?00fH000fR?00fT000g1?00g2800g4000h:?00h;000h??00h@800hB?00hE000hF?00hG000hH800hO?00hP000hc8" +
	"00he000ib?00ic000id800ie?00il000j7?00j?000kb?00kc000kd800ke?00kn000l8?00l?000mH?00mJ000mf?00mg000mh?" +
	"00mi000mj?00mk000mo800n0000nb?00np800o0?00o5000o6?00o8000o=?00oH000oI?00on000p6?00p700100A010[0010^?" +
	"010b8010c?010i0010j?010l8010n?010pA01100011@A011F8011H?011JA011N?011QA011R0011UA011W0011_A011b?011fA" +
	"0122?01230012480125?01270012=?012>A012?0012M?012N001409015P:016X;0180001=M?01=P001LB?01LF001Lc?01Lf0" +
	"01MB?01MD001Mc?01Me001N0A01Ne?01Ng801Nh?01No801O6?01O7801O9?01OD001OM?01ON001P;?01P>301P??01P@001R5?" +
	"01R7001RY?01RZ001TP?01TS801TW?01TY801T]001Ta801Tc?01Td801Tj?01Tm001XG?01XI801XK?01XL001XPA01YE801YF?" +
	"01YG801YH?01YO001YP?01YQ001YR?01YS001YU?01Y^801Yd?01Yn001Yp?01Z0001Za?01[N001[P?01[]001]0?01]4801]50" +
	"01];A01]=001]CA01]e?01]o801^2?01^5A01^=001^[?01^e001_0?01_2801_3A01_Q801_R?01_V801_X?01__A01_a001_lA" +
	"01_o001`V?01`W801`X?01`Z801`^?01`_801``?01`e001aT801a]?01ae801ag?01ai001d@?01dC001dD?01dQ801dR?01dY0" +
	"01d^?01d_001de?01df001dh801di?01dk001h0?01i00020;3020<4020=5020>3020@0020X3020`0020m>020n00219>021:0" +
	"021P3021a0023@?023b0024R>024S0024j>024k0026D>026J0026Y>026[002<J>02<L002<X>02<Y002??>02?@002?Y>02?e0" +
	"02?i>02?l002C2>02C3002FZ>02F]002Fg>02Fh002G0>02G1002Gl>02Gp002H0>02H5002H>>02H?002HA>02HB002HD>02HF0" +
	"02HH>02HI002HM>02HN002HP>02HQ002HR>02HT002HV>02HW002HZ>02H[002H_>02Ha002Hi>02Hl002I0>02I1002I2>02I30" +
	"02I8>02ID002IO>02IQ002IS>02IT002IU>02IW002IX>02IY002Il>02Im002Io>02J0002JB>02JH002JI>02JJ002JK>02JM0" +
	"02JP>02JR002JW>02JX002JZ>02J]002Ja>02Jc002Jn>02Jp002K4>02K6002K8>02K9002K>>02K@002KA>02KB002KC>02KE0" +
	"02KY>02K[002Ka>02Kg002Kh>02Kl002Kn>02Ko002L2>02L3002L5>02L6002L8>02L>002L?>02L@002LB>02LC002LD>02LE0" +
	"02LF>02LG002LM>02LN002LQ>02LR002LX>02LY002Ld>02Lf002M4>02M5002M7>02M8002M<>02M=002M>>02M?002MC>02MF0" +
	"02MG>02MH002MS>02MU002NE>02NH002NQ>02NR002Na>02Nb002Np>02O0002Te>02Tg002]5>02]8002]K>02]M002^@>02^A0" +
	"02^E>02^F002d`?02dc002fp?02g0002hP?02i00030Z?030a>030b0030n>030o0032I?032K003:G>03:H003:I>03:J00:I`?" +
	"0:Id00:Ie?0:Io00:JN?0:JP00:Ka?0:Kc00:P2?0:P300:P6?0:P700:P;?0:P<00:PS80:PU?0:PW80:PX00:P]?0:P^00:R08" +
	"0:R200:Re80:S4?0:S600:SP?0:Sc00:Sp?0:T000:TV?0:T_00:U7?0:UB80:UC?0:UD00:UP90:Un00:V0?0:V380:V400:V9A" +
	"0:V<00:V?A0:Vd?0:Ve80:Vg?0:Vk80:Vm?0:Vo80:W0?0:W100:WPA0:WU?0:WV00:WWA0:Wa00:WkA0:Wp00:XY?0:X`80:Xb?" +
	"0:Xd80:Xf?0:Xh00:Y3?0:Y400:Y<?0:Y=80:Y>00:YPA0:Ya00:YbA0:Ye00:YkA0:Yl00:Ym?0:Yn00:YoA0:Z000:Za?0:Zb0" +
	"0:Zc?0:Zf00:Zh?0:Zj00:Zo?0:[000:[1?0:[200:[PA0:[[80:[]?0:[_80:[a00:[f80:[g?0:[h00:`0A0:`K00:`S80:`U?" +
	"0:`V80:`X?0:`Y80:`[00:`]80:`^?0:`_00:a0<0:a1=0:aL<0:aM=0:ai<0:aj=0:bD<0:bE=0:ba<0:bb=0:c<<0:c==0:cX<" +
	"0:cY=0:d4<0:d5=0:dP<0:dQ=0:dm<0:dn=0:eH<0:eI=0:ee<0:ef=0:f@<0:fA=0:f]<0:f^=0:g8<0:g9=0:gT<0:gU=0:h0<" +
	"0:h1=0:hL<0:hM=0:hi<0:hj=0:iD<0:iE=0:ia<0:ib=0:j<<0:j==0:jX<0:jY=0:k4<0:k5=0:kP<0:kQ=0:km<0:kn=0:lH<" +
	"0:lI=0:le<0:lf=0:m@<0:mA=0:m]<0:m^=0:n8<0:n9=0:nT<0:nU=0:o0<0:o1=0:oL<0:oM=0:oi<0:oj=0:pD<0:pE=0:pa<" +
	"0:pb=0;0<<0;0==0;0X<0;0Y=0;14<0;15=0;1P<0;1Q=0;1m<0;1n=0;2H<0;2I=0;2e<0;2f=0;3@<0;3A=0;3]<0;3^=0;48<" +
	"0;49=0;4T<0;4U=0;50<0;51=0;5L<0;5M=0;5i<0;5j=0;6D<0;6E=0;6a<0;6b=0;7<<0;7==0;7X<0;7Y=0;84<0;85=0;8P<" +
	"0;8Q=0;8m<0;8n=0;9H<0;9I=0;9e<0;9f=0;:@<0;:A=0;:]<0;:^=0;;8<0;;9=0;;T<0;;U=0;<0<0;<1=0;<L<0;<M=0;<i<" +
	"0;<j=0;=D<0;=E=0;=a<0;=b=0;><<0;>==0;>X<0;>Y=0;?4<0;?5=0;?P<0;?Q=0;?m<0;?n=0;@H<0;@I=0;@e<0;@f=0;A@<" +
	"0;AA=0;A]<0;A^=0;B8<0;B9=0;BT<0;BU=0;C0<0;C1=0;CL<0;CM=0;Ci<0;Cj=0;DD<0;DE=0;Da<0;Db=0;E<<0;E==0;EX<" +
	"0;EY=0;F4<0;F5=0;FP<0;FQ=0;Fm<0;Fn=0;GH<0;GI=0;Ge<0;Gf=0;H@<0;HA=0;H]<0;H^=0;I8<0;I9=0;IT<0;IU=0;J0<" +
	"0;J1=0;JL<0;JM=0;Ji<0;Jj=0;KD<0;KE=0;Ka<0;Kb=0;L<<0;L==0;LX<0;LY=0;M4<0;M5=0;MP<0;MQ=0;Mm<0;Mn=0;NH<" +
	"0;NI=0;Ne<0;Nf=0;O@<0;OA=0;O]<0;O^=0;P8<0;P9=0;PT<0;PU=0;Q0<0;Q1=0;QL<0;QM=0;Qi<0;Qj=0;RD<0;RE=0;Ra<" +
	"0;Rb=0;S<<0;S==0;SX<0;SY=0;T4<0;T5=0;TP<0;TQ=0;Tm<0;Tn=0;UH<0;UI=0;Ue<0;Uf=0;V@<0;VA=0;V]<0;V^=0;W8<" +
	"0;W9=0;WT<0;WU=0;X0<0;X1=0;XL<0;XM=0;Xi<0;Xj=0;YD<0;YE=0;Ya<0;Yb=0;Z<<0;Z==0;ZX<0;ZY=0;[4<0;[5=0;[P<" +
	"0;[Q=0;[m<0;[n=0;]H<0;]I=0;]e<0;]f=0;^@<0;^A=0;^]<0;^^=0;_8<0;_9=0;_T<0;_U=0;`0<0;`1=0;`L<0;`M=0;`i<" +
	"0;`j=0;aD<0;aE=0;aa<0;ab=0;b<<0;b==0;bX<0;bY=0;c4<0;c5=0;cP<0;cQ=0;cm<0;cn=0;dH<0;dI=0;de<0;df=0;e@<" +
	"0;eA=0;e]<0;e^=0;f8<0;f9=0;fT<0;fU=0;g0<0;g1=0;gL<0;gM=0;gi<0;gj=0;hD<0;hE=0;ha<0;hb=0;i<<0;i==0;iX<" +
	"0;iY=0;j4<0;j5=0;jP<0;jQ=0;jm<0;jn=0;kH<0;kI=0;ke<0;kf=0;l@<0;lA=0;l]<0;l^=0;m8<0;m9=0;mT<0;mU=0;n0<" +
	"0;n1=0;nL<0;nM=0;ni<0;nj=0;oD<0;oE=0;oa<0;ob=0;p<<0;p==0;pX<0;pY=0<04<0<05=0<0P<0<0Q=0<0m<0<0n=0<1H<" +
	"0<1I=0<1e<0<1f=0<2@<0<2A=0<2]<0<2^=0<38<0<39=0<3T<0<3U=0<40<0<41=0<4L<0<4M=0<4i<0<4j=0<5D<0<5E=0<5a<" +
	"0<5b=0<6<<0<6==0<6X<0<6Y=0<74<0<75=0<7P<0<7Q=0<7m<0<7n=0<8H<0<8I=0<8e<0<8f=0<9@<0<9A=0<9]<0<9^=0<:8<" +
	"0<:9=0<:T<0<:U=0<;0<0<;1=0<;L<0<;M=0<;i<0<;j=0<<D<0<<E=0<<a<0<<b=0<=<<0<===0<=X<0<=Y=0<>4<0<>5=0<>P<" +
	"0<>Q=0<>m<0<>n=0<?H<0<?I=0<?e<0<?f=0<@@<0<@A=0<@]<0<@^=0<A8<0<A9=0<AT<0<AU=0<B0<0<B1=0<BL<0<BM=0<Bi<" +
	"0<Bj=0<CD<0<CE=0<Ca<0<Cb=0<D<<0<D==0<DX<0<DY=0<E4<0<E5=0<EP<0<EQ=0<Em<0<En=0<FH<0<FI=0<Fe<0<Ff=0<G@<" +
	"0<GA=0<G]<0<G^=0<H8<0<H9=0<HT<0<HU=0<I0<0<I1=0<IL<0<IM=0<Ii<0<Ij=0<JD<0<JE=0<Ja<0<Jb=0<K<<0<K==0<KX<" +
	"0<KY=0<L4<0<L5=0<LP<0<LQ=0<Lm<0<Ln=0<MH<0<MI=0<Me<0<Mf=0<N@<0<NA=0<N]<0<N^=0<O8<0<O9=0<OT<0<OU=0<P0<" +
	"0<P1=0<PL<0<PM=0<Pi<0<Pj=0<QD<0<QE=0<Qa<0<Qb=0<R<<0<R==0<RX<0<RY=0<S4<0<S5=0<SP<0<SQ=0<Sm<0<Sn=0<TH<" +
	"0<TI=0<Te<0<Tf=0<U@<0<UA=0<U]<0<U^=0<V8<0<V9=0<VT<0<VU=0<W0<0<W1=0<WL<0<WM=0<Wi<0<Wj=0<XD<0<XE=0<Xa<" +
	"0<Xb=0<Y<<0<Y==0<YX<0<YY=0<Z4<0<Z5=0<ZP<0<ZQ=0<Zm<0<Zn=0<[H<0<[I=0<[e<0<[f=0<]@<0<]A=0<]]<0<]^=0<^8<" +
	"0<^9=0<^T<0<^U=0<_0<0<_1=0<_L<0<_M=0<_i<0<_j=0<`D<0<`E=0<`a<0<`b=0<a<<0<a==0<aX<0<aY=0<b4<0<b5=0<bP<" +
	"0<bQ=0<bm<0<bn=0<cH<0<cI=0<ce<0<cf=0<d@<0<dA=0<d]<0<d^=0<e8<0<e9=0<eT<0<eU=0<f0<0<f1=0<fL<0<fM=0<fi<" +
	"0<fj=0<gD<0<gE=0<ga<0<gb=0<h<<0<h==0<hX<0<hY=0<i4<0<i5=0<iP<0<iQ=0<im<0<in=0<jH<0<jI=0<je<0<jf=0<k@<" +
	"0<kA=0<k]<0<k^=0<l8<0<l9=0<lT<0<lU=0<m0<0<m1=0<mL<0<mM=0<mi<0<mj=0<nD<0<nE=0<na<0<nb=0<o<<0<o==0<oX<" +
	"0<oY=0<p4<0<p5=0<pP<0<pQ=0<pm<0<pn=0=0H<0=0I=0=0e<0=0f=0=1@<0=1A=0=1]<0=1^=0=28<0=29=0=2T<0=2U=0=30<" +
	"0=31=0=3L<0=3M=0=3i<0=3j=0=4D<0=4E=0=4a<0=4b=0=5<<0=5==0=5X<0=5Y=0=64<0=65=0=6P<0=6Q=0=6m<0=6n=0=7H<" +
	"0=7I=0=7e<0=7f=0=8@<0=8A=0=8]<0=8^=0=98<0=99=0=9T<0=9U=0=:0<0=:1=0=:L<0=:M=0=:i<0=:j=0=;D<0=;E=0=;a<" +
	"0=;b=0=<<<0=<==0=<X<0=<Y=0==4<0==5=0==P<0==Q=0==m<0==n=0=>H<0=>I=0=>e<0=>f=0=?@<0=?A=0=?]<0=?^=0=@8<" +
	"0=@9=0=@T<0=@U=0=A0<0=A1=0=AL<0=AM=0=Ai<0=Aj=0=BD<0=BE=0=Ba<0=Bb=0=C<<0=C==0=CX<0=CY=0=D4<0=D5=0=DP<" +
	"0=DQ=0=Dm<0=Dn=0=EH<0=EI=0=Ee<0=Ef=0=F@<0=FA=0=F]<0=F^=0=G8<0=G9=0=GT<0=GU=0=H0<0=H1=0=HL<0=HM=0=Hi<" +
	"0=Hj=0=ID<0=IE=0=Ia<0=Ib=0=J<<0=J==0=JX<0=JY=0=K4<0=K5=0=KP<0=KQ=0=Km<0=Kn=0=LH<0=LI=0=Le<0=Lf=0=M@<" +
	"0=MA=0=M]<0=M^=0=N8<0=N9=0=NT00=Na:0=O700=O;;0=Om00=P030>0000?]N?0?]O00?i0?0?i@00?iP?0?ia00?lp30?m00" +
	"0?oN?0?oP00?pa30?pm00@7n?0@7o00@;P?0@;Q00@=g?0@=l00@X0A0@X1?0@X400@X5?0@X700@X<?0@X@A0@XD00@XEA0@XH0" +
	"0@XIA0@Xg00@Xi?0@Xl00@Xp?0@Y000@[U?0@[W00@eT?0@eX00@fY?0@f_00@k[?0@k^00@lk?0@m000@n6?0@nA00@o2?0@o60" +
	"0A0080A01?0A0280A0300A0i?0A1700A1a?0A1b00A1d?0A1f00A1p?0A2280A2300A2a80A2d?0A2h80A2j?0A2l00A2n70A2o0" +
	"0A32?0A3300A3=70A3>00A40?0A43A0A4W?0A4]80A4^?0A4f00A54A0A5580A57A0A5800A5d?0A5e00A60?0A6280A6300A6d8" +
	"0A6g?0A6p80A70?0A7100A7270A7400A79?0A7=00A7>80A7??0A7@00A8]80A8`?0A8c80A8e?0A8i00A8o?0A8p00A91?0A920" +
	"0A;O?0A;P80A;S?0A;[00A<0?0A<280A<400A<l?0A<n00A<o?0A<p80A=0?0A=180A=500A=780A=900A=;80A==?0A=>00A=G?" +
	"0A=H00A=R80A=T00A=V?0A=^00A=a?0A=f00A>0A0A>:00A>;A0A><00A>>A0A>?00A>@A0A>g00A>i?0A>j80A>l?0A?100A?2?" +
	"0A?300A?5?0A?600A?7?0A?:80A?;00A?<80A?>?0A?A70A?B?0A?C00A?Q?0A?S00A@f80A@i?0AA080AA2?0AA580AA6?0AA70" +
	"0AAN?0AAO00ABa?0ABb80ABd?0ABj80ABk?0ABl80ABn?0ABo80ABp?0AC180AC2?0AC400AF`?0AFa80AFc?0AFg00AFi80AFm?" +
	"0AFo80AFp?0AG100AGL?0AGN00AHa80AHd?0AHl80AHn?0AHo80AHp?0AI100AJ[?0AJ]80AJ^?0AJ_80AJa?0AJi00ALM?0ALN8" +
	"0ALO?0ALP00ALR?0ALV80ALW?0AL]00AP]80AP`?0APi80APj?0APl00AT0A0AT700AT9A0AT:00AT<A0ATD00ATEA0ATG00ATHA" +
	"0ATa?0ATb80ATg00ATh80ATj00ATl?0ATp70AU080AU170AU280AU3?0AU400AWA80AWD?0AWH00AWJ?0AWL80AWP?0AWQ00AWT8" +
	"0AWU00AX0A0AX1?0AX;A0AXd?0AXj80AXk00AXl?0AXp00AY7?0AY800AY@A0AYA?0AYG80AYI?0AYLA0AZ470AZ:?0AZG80AZH?" +
	"0AZJ00A^P?0A^Q80A^R?0A^U80A^V?0A^W80A^X00Aa`80Aaa?0Aah00Aai?0Aao80Aap?0Ab000AcB?0AcX00AcY80AcZ?0Acb8" +
	"0Acc?0Ace80Acf?0Ach00Aeb?0Aeh00Aek?0Ael00Aem?0Aeo00Aep?0Af670Af7?0Af800Ag:80Ag?00Ag@?0AgB00AgC80AgE?" +
	"0AgF80AgG?0AgH00Ald?0Alf80Alh00Am0?0Am270Am380Am4A0AmA00AmBA0Ame80Amg?0Aml00Amo80An0?0An300AnJ?0AnK0" +
	"0C@a30CA0?0CA100CA7?0CAF00F4N?0F4Z80F4^?0F4a00F[a?0F[f00F]a?0F]h00Fn??0Fn@00FnA80Fo800Fo??0FoC00FpT?" +
	"0FpU00Fpa?0Fpc00KcM?0KcO00KcP30KcT00Lm0?0Lm_00Lma?0Ln700M5U?0M5Z00M5^?0M5d30M5l?0M6300M65?0M6<00M6Z?" +
	"0M6_00M92?0M9500MX0?0MXh00MXl?0MY^00MYf?0MYg00MZ4?0MZ500MZK?0MZP00MZQ?0MZa00N00?0N0700N08?0N0I00N0K?" +
	"0N0R00N0S?0N0U00N0V?0N0[00N2??0N2@00N4a?0N4h00N:_?0N:`00N;]?0N;a00NC]?0NCa00NG_?0NGa00NKS?0NKT00NKV?" +
	"0NKW00NK_?0NKa00NKf?0NKg00NS@?0NSG00NU4?0NU;00O04>0O0500O0]>0O0a00O2D>0O2P00O2`>0O2b00O30>0O3100O3?>" +
	"0O3A00O3g>0O4000O5a>0O5c00O5o>0O6000O6>>0O6?00O6A>0O6K00O6_>0O7V60O8000O81>0O8@00O8J>0O8K00O8`>0O8a0" +
	"0O8c>0O8l00O8m>0O9000O99>0O9P00O9V>0O<R00O<T>0O>D00O>F>0O>H00O>I>0O>L00O>N>0O?b00O?d>0O?g00O?h>0O?l?" +
	"0O@0>0OCo00OCp>0ODo00OE9>0OE?00OE@>0OEX00OE`>0OEb00OEd>0OEl00OF7>0OF800OF:>0OF>00OF@>0OFA00OFE>0OFG0" +
	"0OFT>0OFV00OFX>0OFY00OFb>0OFd00OFm>0OFn00OG2>0OG500OGA>0OGD00OGL>0OGO00OGQ>0OGR00OGS>0OGT00OGX>0OGY0" +
	"0OG`>0OGa00OGd>0OGe00OGk>0OI@00OJ0>0OK600OK;>0OKC00OKE>0OKV00OKY>0OKZ00OK[>0OKb00OKd>0OL000OOJ>0OP00" +
	"0OP<>0OP@00OQ8>0OQ@00OQJ>0OQP00OR8>0OR@00OR_>0ORa00ORm>0OS000OS2>0OS@00OSI>0OT000OT<>0OTl00OTm>0OU60" +
	"0OU7>0OX000OYH>0OYP00OY_>0O]000Oa0>0Opo03P0033P0P?3P2033P40?3P7a33Q000"

// swcGcb is a code point's grapheme-cluster-break class (the classes are listed
// in languages/lib/swift-chars.js).
func swcGcb(c int) int {
	if c < 128 {
		switch {
		case c == 13:
			return 1
		case c == 10:
			return 2
		case c < 32 || c == 127:
			return 3
		}
		return 0
	}
	if c > 0x10FFFF {
		return 3
	}
	return jcDig(swcGcbTable, jcRun(swcGcbTable, c, 5)*5+4)
}

func swcIsExt(k int) bool { return k == 4 || k == 15 || k == 16 }

// swcBreak is UAX #29's GB3 ... GB999 between classes p and k; ri, emo and cb
// are the Regional_Indicator run, the GB11 state and the GB9c state.
func swcBreak(p, k, ri, emo, cb int) bool {
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
	case swcIsExt(k) || k == 5 || k == 8:
		return false
	case p == 7:
		return false
	case cb == 2 && k == 17:
		return false
	case emo == 2 && k == 14:
		return false
	case p == 6 && k == 6 && ri%2 == 1:
		return false
	}
	return true
}

// swcGraphemes splits s into its extended grapheme clusters. Twin of swcBounds /
// swcGraphemes in languages/lib/swift-chars.js.
func swcGraphemes(s string) []string {
	out := []string{}
	start, p, ri, emo, cb := 0, -1, 0, 0, 0
	for i, r := range s {
		k := swcGcb(int(r))
		if p >= 0 && swcBreak(p, k, ri, emo, cb) {
			out = append(out, s[start:i])
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
		case emo == 1 && swcIsExt(k):
		case emo == 1 && k == 5:
			emo = 2
		default:
			emo = 0
		}
		switch {
		case k == 17:
			cb = 1
		case cb > 0 && k == 16:
			cb = 2
		case cb > 0 && (k == 15 || k == 5):
		default:
			cb = 0
		}
		p = k
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// swcNfdBase / swcNfdMark: the Latin-1 precomposed letters' canonical
// decomposition, base letter and combining mark (0 for none). Twin of the table in
// languages/lib/swift-chars.js.
const swcNfdBase = "AAAAAA\x00CEEEEIIII\x00NOOOOO\x00\x00UUUUY\x00\x00aaaaaa\x00ceeeeiiii\x00nooooo\x00\x00uuuuy\x00y"

var swcNfdMark = [64]rune{0x300, 0x301, 0x302, 0x303, 0x308, 0x30A, 0, 0x327,
	0x300, 0x301, 0x302, 0x308, 0x300, 0x301, 0x302, 0x308,
	0, 0x303, 0x300, 0x301, 0x302, 0x303, 0x308, 0,
	0, 0x300, 0x301, 0x302, 0x308, 0x301, 0, 0,
	0x300, 0x301, 0x302, 0x303, 0x308, 0x30A, 0, 0x327,
	0x300, 0x301, 0x302, 0x308, 0x300, 0x301, 0x302, 0x308,
	0, 0x303, 0x300, 0x301, 0x302, 0x303, 0x308, 0,
	0, 0x300, 0x301, 0x302, 0x308, 0x301, 0, 0x308}

func swcNFD(s string) []rune {
	out := []rune{}
	for _, r := range s {
		if r >= 0xC0 && r <= 0xFF && swcNfdBase[r-0xC0] != 0 {
			out = append(out, rune(swcNfdBase[r-0xC0]), swcNfdMark[r-0xC0])
			continue
		}
		out = append(out, r)
	}
	return out
}

// swcNFC composes the pairs swcNFD splits, for the ordering.
func swcNFC(s string) []rune {
	d := swcNFD(s)
	out := []rune{}
	for i := 0; i < len(d); i++ {
		if i+1 < len(d) {
			hit := -1
			for q := 0; q < 64; q++ {
				if rune(swcNfdBase[q]) == d[i] && swcNfdMark[q] == d[i+1] && swcNfdMark[q] != 0 {
					hit = q
				}
			}
			if hit >= 0 {
				out = append(out, rune(0xC0+hit))
				i++
				continue
			}
		}
		out = append(out, d[i])
	}
	return out
}

// swcStrEq is ==: canonical equivalence over the Latin-1 table.
func swcStrEq(a, b string) bool {
	return a == b || string(swcNFD(a)) == string(swcNFD(b))
}

// swcStrCmp is Swift's String ordering, the NFC forms scalar by scalar.
func swcStrCmp(a, b string) int {
	x, y := swcNFC(a), swcNFC(b)
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] {
			if x[i] < y[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(x) < len(y):
		return -1
	case len(x) > len(y):
		return 1
	}
	return 0
}

// swcHasPrefix / swcHasSuffix / swcContains compare CHARACTERS: "é" neither
// starts with nor contains "e".
func swcHasPrefix(s, p string) bool {
	cs, cp := swcGraphemes(s), swcGraphemes(p)
	if len(cp) > len(cs) {
		return false
	}
	for i := range cp {
		if !swcStrEq(cs[i], cp[i]) {
			return false
		}
	}
	return true
}

func swcHasSuffix(s, p string) bool {
	cs, cp := swcGraphemes(s), swcGraphemes(p)
	off := len(cs) - len(cp)
	if off < 0 {
		return false
	}
	for i := range cp {
		if !swcStrEq(cs[off+i], cp[i]) {
			return false
		}
	}
	return true
}

func swcContains(s, p string) bool {
	cs, cp := swcGraphemes(s), swcGraphemes(p)
	for at := 0; at+len(cp) <= len(cs); at++ {
		ok := true
		for i := 0; i < len(cp) && ok; i++ {
			ok = swcStrEq(cs[at+i], cp[i])
		}
		if ok {
			return true
		}
	}
	return false
}
