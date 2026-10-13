// java-regex.js -- the java.util.regex half that sits over lib/regex.js's J
// dialect: the \p{..} property knob and PatternSyntaxException's message. Kotlin/JVM's
// Regex IS java.util.regex, so this serves java-interpreter.abnf and
// kotlin-interpreter.abnf (include) and lib/java-rt.metajs and lib/kotlin-rt.metajs
// (import), each after lib/java-chars.js and lib/regex.js.

// jxPropHit is regex.js's rxPropHit knob over java.lang.Character's table
// (lib/java-chars.js). Twin of the rxPropHit switch in abnf/jsrtregex.go.
function jxPropHit(id, c) {
    if (id == RX_PL) { return jcIsLetter(c) }
    if (id == RX_PND) { return jcIsDigit(c) }
    if (id == RX_PALPHA) { return jcIsAlphabetic(c) }
    if (id == RX_PUPPER) { return jcIsUpper(c) }
    if (id == RX_PLOWER) { return jcIsLower(c) }
    if (id == RX_PLD) { return jcIsLetter(c) || jcIsDigit(c) }
    var k = jcClass(c) % 8
    if (id == RX_PLU) { return k == 1 }
    if (id == RX_PLL) { return k == 2 }
    if (id == RX_PLT) { return k == 3 }
    return id == RX_PNL && k == 6
}
// jxReMsg renders a regex.js error as Java's PatternSyntaxException message. A J
// error is Java's own description at its own index (at). For the shared kinds the
// index is read off k, the length of the shortest prefix of the \Q-expanded
// pattern that fails with the same kind (Java indexes the expanded pattern too);
// an end-of-input kind uses the full length. Unmapped kinds keep regex.js's text.
// Twin of jxReMsg in abnf/jsrtjavastr.go.
function jxReMsg(pattern, err, at) {
    if (at >= 0) { return jxReFmt(pattern, err, at) }
    var p = rxExpandQuotes(pattern)
    var n = p.length
    if (err == "unterminated group") { return jxReFmt(pattern, "Unclosed group", n) }
    if (err == "unterminated character class") { return jxReFmt(pattern, "Unclosed character class", n - 1) }
    if (err == "trailing backslash in pattern") { return jxReFmt(pattern, "Unescaped trailing backslash", n) }
    var k = 1
    while (k < n && rxCompile(p.substring(0, k), "J").err != err) { k = k + 1 }
    if (err == "unmatched ) in pattern") { return jxReFmt(pattern, "Unmatched closing ')'", k - 2) }
    if (err == "quantifier with nothing to repeat" || err == "multiple repeat") {
        return jxReFmt(pattern, "Dangling meta character '" + p.charAt(k - 1) + "'", k - 1)
    }
    if (err.indexOf("unsupported group form") == 0) { return jxReFmt(pattern, "Unknown inline modifier", k - 1) }
    if (err.indexOf("unknown group name '") == 0) {
        var name = err.substring(20, err.length - 1)
        return jxReFmt(pattern, "named capturing group <" + name + "> does not exist", k - 1)
    }
    return err
}
// PatternSyntaxException.getMessage: the description, " near index N" when N >= 0,
// the pattern, and a caret under N when N is inside it.
function jxReFmt(pattern, desc, at) {
    if (at < 0) { return desc + "\n" + pattern }
    var out = desc + " near index " + at + "\n" + pattern
    if (at >= pattern.length) { return out }
    out = out + "\n"
    var i = 0
    while (i < at) {
        out = out + " "
        i = i + 1
    }
    return out + "^"
}
