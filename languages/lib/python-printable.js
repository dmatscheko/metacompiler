// python-printable.js -- CPython's Py_UNICODE_ISPRINTABLE, which repr() of a str
// and str.isprintable() both ask, for python-interpreter.abnf (include) and
// languages/lib/python-rt.metajs (import). Go twin: abnf/pyprintable.go.
//
// A code point is printable unless its general category is Cc, Cf, Cs, Co, Cn,
// Zl, Zp, or Zs other than U+0020. The table is the printable runs over
// U+0000..U+10FFFF as python3 3.14 (Unicode 16.0.0) answers them: alternately
// the gap before a run and the run's length, in base 36. It is decoded on first
// use into PY_PRT, run starts and (exclusive) ends.
var PY_PRT = []
function pyIsPrintable(cp) {
    if (cp >= 32 && cp < 127) { return true }
    if (cp < 161) { return false }
    if (PY_PRT.length == 0) { pyPrintRuns() }
    var lo = 0
    var hi = PY_PRT.length / 2
    while (lo < hi) {
        var mid = Math.floor((lo + hi) / 2)
        if (cp < PY_PRT[2 * mid]) { hi = mid }
        else if (cp >= PY_PRT[2 * mid + 1]) { lo = mid + 1 }
        else { return true }
    }
    return false
}

// str.isprintable(): every code point printable, so "" is True. A surrogate
// PAIR is one astral code point; a lone surrogate is Cs.
function pyStrIsPrintable(s) {
    var k = 0
    while (k < s.length) {
        var cp = s.charCodeAt(k)
        var w = 1
        if (cp >= 55296 && cp <= 56319 && k + 1 < s.length) {
            var lo = s.charCodeAt(k + 1)
            if (lo >= 56320 && lo <= 57343) {
                cp = 65536 + (cp - 55296) * 1024 + (lo - 56320)
                w = 2
            }
        }
        if (!pyIsPrintable(cp)) { return false }
        k = k + w
    }
    return true
}

function pyPrintRuns() {
    var t = "w,2n,y,c,1,ju,2,6,4,7,1,1,1,k,1,b1,1,12,2,1e,2,3,1,1j,8,r,4,6,h,m,1,5c,1,1c," +
        "2,1n,2,2t,e,1n,2,1d,2,f,1,s,2,1,1,b,5,v,8,23,1,4h,1,8,2,2,2,m,1,7,1,1,3,4,2," +
        "9,2,2,2,4,8,1,4,2,1,5,2,p,2,3,1,6,4,2,2,m,1,7,1,2,1,2,1,2,2,1,1,5,4,2,2,3,3," +
        "1,7,4,1,1,7,h,a,3,1,9,1,3,1,m,1,7,1,2,1,5,2,a,1,3,1,3,2,1,f,4,2,c,7,7,1,3,1," +
        "8,2,2,2,m,1,7,1,2,1,5,2,9,2,2,2,3,7,3,4,2,1,5,2,i,a,2,1,6,3,3,1,4,3,2,1,1,1," +
        "2,3,2,3,3,3,c,4,5,3,3,1,4,2,1,6,1,e,l,5,d,1,3,1,n,1,g,2,9,1,3,1,4,7,2,1,3,2," +
        "1,2,4,2,a,7,m,1,3,1,n,1,a,1,5,2,9,1,3,1,4,7,2,6,2,1,4,2,a,1,3,c,d,1,3,1,1f," +
        "1,3,1,6,4,g,2,q,1,3,1,i,3,o,1,9,1,1,2,7,3,1,4,6,1,1,1,8,6,a,2,3,c,1m,4,t,11," +
        "2,1,1,1,5,1,o,1,1,1,n,2,5,1,1,1,7,1,a,2,4,w,20,1,10,4,13,1,10,1,f,1,d,11,5i," +
        "1,1,5,1,2,ah,1,4,2,7,1,1,1,4,2,15,1,4,2,x,1,4,2,7,1,1,1,4,2,f,1,1l,1,4,2,1v," +
        "2,w,3,q,6,2e,2,6,2,hs,1,s,3,2h,7,m,9,o,9,k,c,d,1,3,1,2,c,2m,2,a,6,a,6,e,1,b," +
        "6,2h,7,17,5,1y,a,v,1,c,4,c,4,1,3,16,2,5,b,18,4,q,6,b,3,1q,2,1t,1,t,2,b,6,a," +
        "6,e,2,v,1d,25,1,4m,8,1o,3,f,3,1q,5,17,2,b,8,17,5,eu,2,6,2,12,2,6,2,8,1,1,1," +
        "1,1,1,1,v,2,1h,1,f,1,e,2,6,1,j,2,3,1,9,h,o,8,1b,h,2,2,r,1,d,3,x,f,x,f,3w,4," +
        "ii,m,b,l,1ec,2,w,1,9p,5,19,1,1,5,1,2,1k,7,2,e,o,9,7,1,7,1,7,1,7,1,7,1,7,1,7," +
        "1,7,1,3i,y,q,1,2h,c,5y,q,g,1,1r,1,2e,2,2v,5,17,1,2m,1,2e,9,1c,1,mlp,3,1j,9," +
        "9o,k,54,8,5q,2,2,1,1,1,8,l,1n,3,a,6,1k,8,1y,8,c,6,38,b,u,3,26,1,b,4,x,1,1j," +
        "9,e,2,a,2,2v,o,s,a,6,2,6,2,6,9,7,1,7,1,1o,4,3i,2,a,6,8mc,c,n,4,1d,6is,a6,2," +
        "2y,12,7,c,5,5,q,1,5,1,1,1,2,1,2,1,3h,g,cd,2,1i,7,1,w,16,6,1f,1,j,1,4,4,5,1," +
        "3r,4,5a,3,6,2,6,2,6,2,3,3,7,1,7,d,2,2,c,1,q,1,j,1,2,1,f,2,e,y,3f,5,3,4,19,3," +
        "2g,1,d,3,1,1b,1a,3m,t,3,1d,f,s,4,10,9,u,5,17,5,u,1,11,4,e,16,4e,2,a,6,10,4," +
        "10,4,14,8,1g,b,c,1,f,1,7,1,2,1,b,1,f,1,7,1,2,3,1g,c,8n,9,m,a,8,o,6,1,16,1,9," +
        "1x,6,2,1,1,18,1,2,3,1,2,n,1,20,8,9,1c,j,1,2,5,x,3,r,5,1,1s,1k,4,k,2,1e,1,2," +
        "5,8,1,3,1,t,2,3,4,a,7,9,7,1s,w,13,4,c,9,1i,3,t,2,r,5,q,7,4,c,7,28,21,1j,1f," +
        "d,1f,7,1a,8,a,6,12,3,t,8,2,5s,v,1,16,1,3,2,2,g,3,1j,18,8,16,m,q,12,s,k,n,9," +
        "26,4,10,9,1q,1,5,d,p,7,a,6,1h,1,i,8,13,9,2o,1,k,b,i,1,1b,1q,7,1,1,1,4,1,f,1," +
        "b,6,1n,5,a,6,4,1,8,2,2,2,m,1,7,1,2,1,5,1,a,2,2,2,3,2,1,6,1,5,7,2,7,3,5,b,a," +
        "1,1,2,1,1,12,1,a,1,1,2,1,1,4,1,a,1,2,8,2,t,2k,1,5,u,20,8,a,4m,1i,2,12,y,1x," +
        "b,a,6,d,j,1m,6,a,6,k,s,r,2,f,4,n,55,1o,2s,2b,c,8,2,1,2,8,1,2,1,u,1,2,2,c,9," +
        "a,1y,8,2,1a,2,b,r,20,8,2b,d,21,7,a,52,y,e,a,6,9,1,19,1,e,a,t,3,w,2,m,1,e,21," +
        "7,1,2,1,18,3,1,1,2,1,9,8,a,6,6,1,2,1,11,1,2,1,6,7,a,8m,p,7,h,1,15,3,t,2d,1," +
        "f,1e,d,pn,2u,33,1,5,b,5g,218,2r,d,ts,g,m,a,32z,5,g7,5a1,1m,1c6,ft,7,v,1,a,4," +
        "29,1,a,6,u,2,6,a,1y,a,a,1,7,1,l,5,j,c0,1m,5i,2j,2t,23,4,1l,7,h,1s,5,b,2,e," +
        "4qg,8,ye,15,a,6w7,4,1,7,1,2,1,83,f,1,t,3,2,1,e,4,8,b0,1s4,2z,5,d,3,9,7,a,2," +
        "4,31c,6y,6,c4,24,1a,2,n,9,38,1o,6u,a,13,2,22,8,34,l,1y,3e,k,c,k,c,2f,9,p,3r," +
        "2d,1,1z,1,2,2,1,2,2,2,4,1,c,1,1,1,7,1,1t,1,4,2,8,1,7,1,s,1,4,1,5,1,1,3,7,1," +
        "9g,2,84,2,ji,f,5,1,f,uo,v,6,6,5x,7,1,h,2,7,1,2,1,5,5,1q,x,1,34,19,3,e,2,a,4," +
        "2,8w,v,h,1m,5,1,cw,16,5y,17,4,1,dc,7,1,4,1,2,1,f,1,5h,2,g,15,24,4,a,4,2,lt," +
        "1w,24,1p,5e,4,1,r,1,2,1,1,2,1,1,a,1,4,1,1,1,1,6,1,4,1,1,1,1,1,1,3,1,2,1,1,2," +
        "1,1,1,1,1,1,1,1,1,1,2,1,1,2,4,1,7,1,4,1,4,1,1,1,a,1,h,5,3,1,5,1,h,1g,2,7i," +
        "18,4,2s,c,f,2,f,1,f,1,11,a,4u,1k,t,d,18,4,9,7,2,e,6,4a,rc,4,h,3,d,3,3b,4,2n," +
        "6,c,4,1,f,c,4,1k,8,a,6,14,8,u,2,c,4,2,1q,9g,c,e,2,d,3,a,5,1k,7,f,2,b,6,9,7," +
        "43,1,2u,sm,wyo,w,37e,6,66,2,4g2,e,5rl,f,ha,1wi,f2,15u,3t7,5,38g,f9e8,6o"
    var at = 0
    var n = 0
    var i = 0
    while (i <= t.length) {
        var c = i < t.length ? t.charCodeAt(i) : 44
        if (c == 44) {
            at = at + n
            PY_PRT.push(at)
            n = 0
        } else { n = n * 36 + (c <= 57 ? c - 48 : c - 87) }
        i = i + 1
    }
}
