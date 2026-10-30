// swift-format.js -- Foundation's String(format:, args...), for
// swift-interpreter.abnf (include) and languages/lib/swift-rt.metajs (import).
// The Go twin is swFormat in abnf/jsrtswift.go. It needs lib/c-printf.js in the
// same scope for the conversions themselves.
//
// CFString's format walk on this platform, as swiftc 6.1.2 on macOS 14 prints it:
//   %[n$][flags][width|*][.prec|.*][length]conv
// A conversion without a length modifier reads a C int, so %d and %x of an Int
// print its low 32 bits; hh and h read 8 and 16 bits, l, ll, q, j, z and t all 64.
// %@ is the argument's description, and ignores the width and precision (%s,
// which takes a C string, is read the same way). %c is the low byte and %C the
// low 16 bits as one character. A missing argument reads as 0. The includer supplies the value-model HOOKS:
//   swfArgDigits(v, bits) the integer v at `bits` as c-printf.js's {neg, d, u, x, o}
//   swfArgNum(v)          v as a host number
//   swfArgText(v)         String(describing: v)

function swfIsDigit(c) { return c >= 48 && c <= 57 }
// The integer read as a host number, for a `*` width or precision.
function swfArgInt(v) {
    var iv = swfArgDigits(v, 32)
    var n = 0
    for (var i = 0; i < iv.d.length; i++) { n = n * 10 + iv.d.charCodeAt(i) - 48 }
    return iv.neg ? 0 - n : n
}
function swfFormat(fmt, args) {
    var out = ""
    var next = 0
    var i = 0
    while (i < fmt.length) {
        var c = fmt.charCodeAt(i)
        if (c !== 37) { out = out + fmt.charAt(i); i = i + 1; continue }
        i = i + 1
        if (i < fmt.length && fmt.charCodeAt(i) === 37) { out = out + "%"; i = i + 1; continue }
        var pos = 0 - 1
        var j = i
        var n = 0
        while (j < fmt.length && swfIsDigit(fmt.charCodeAt(j))) { n = n * 10 + fmt.charCodeAt(j) - 48; j = j + 1 }
        if (j > i && j < fmt.length && fmt.charCodeAt(j) === 36) { pos = n - 1; i = j + 1 }
        var sp = {left: false, plus: false, space: false, alt: false, zero: false, width: 0, prec: 0 - 1}
        while (i < fmt.length) {
            var f = fmt.charCodeAt(i)
            if (f === 45) { sp.left = true }
            else if (f === 43) { sp.plus = true }
            else if (f === 32) { sp.space = true }
            else if (f === 35) { sp.alt = true }
            else if (f === 48) { sp.zero = true }
            else { break }
            i = i + 1
        }
        if (i < fmt.length && fmt.charCodeAt(i) === 42) {
            var wv = swfArgInt(next < args.length ? args[next] : 0)
            next = next + 1
            if (wv < 0) { sp.left = true; wv = 0 - wv }
            sp.width = wv
            i = i + 1
        } else {
            while (i < fmt.length && swfIsDigit(fmt.charCodeAt(i))) { sp.width = sp.width * 10 + fmt.charCodeAt(i) - 48; i = i + 1 }
        }
        if (i < fmt.length && fmt.charCodeAt(i) === 46) {
            i = i + 1
            sp.prec = 0
            if (i < fmt.length && fmt.charCodeAt(i) === 42) {
                var pv = swfArgInt(next < args.length ? args[next] : 0)
                next = next + 1
                sp.prec = pv < 0 ? 0 - 1 : pv
                i = i + 1
            } else {
                while (i < fmt.length && swfIsDigit(fmt.charCodeAt(i))) { sp.prec = sp.prec * 10 + fmt.charCodeAt(i) - 48; i = i + 1 }
            }
        }
        var bits = 32
        while (i < fmt.length) {
            var m = fmt.charCodeAt(i)
            // h halves what it modifies: h is 16 bits, hh 8.
            if (m === 104) { bits = bits === 16 ? 8 : 16 }
            else if (m === 108 || m === 113 || m === 106 || m === 122 || m === 116 || m === 76) { bits = 64 }
            else { break }
            i = i + 1
        }
        // A lone '%' at the end prints nothing.
        if (i >= fmt.length) { break }
        var conv = fmt.charCodeAt(i)
        i = i + 1
        var k = pos >= 0 ? pos : next
        if (pos < 0) { next = next + 1 }
        var arg = k < args.length ? args[k] : 0
        if (conv === 100 || conv === 105) {
            out = out + cpfIntConv(sp, 100, swfArgDigits(arg, bits))
        } else if (conv === 117 || conv === 111 || conv === 120 || conv === 88) {
            out = out + cpfIntConv(sp, conv, swfArgDigits(arg, bits))
        } else if (conv === 102 || conv === 70 || conv === 101 || conv === 69 || conv === 103 || conv === 71
                || conv === 97 || conv === 65) {
            out = out + cpfFloat(sp, conv, swfArgNum(arg))
        } else if (conv === 99 || conv === 67) {
            var cu = swfArgDigits(arg, conv === 99 ? 8 : 16).u
            var code = 0
            for (var q = 0; q < cu.length; q++) { code = code * 10 + cu.charCodeAt(q) - 48 }
            out = out + cpfPad(sp, "", String.fromCharCode(code), false)
        } else if (conv === 64 || conv === 115) {
            out = out + swfArgText(arg)
        } else {
            // Not a conversion: its letter is printed and no argument is read.
            out = out + String.fromCharCode(conv)
            if (pos < 0) { next = next - 1 }
        }
    }
    return out
}
