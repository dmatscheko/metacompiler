// java-cce.js -- java 24's ClassCastException message, for java-interpreter.abnf
// and kotlin-interpreter.abnf (include) and languages/lib/java-rt.metajs and
// languages/lib/kotlin-rt.metajs (import). Kotlin/JVM's casts throw the JVM's own
// exception, so both languages print the same text. Twin of jvCceMsg in
// abnf/jsrtjavastr.go.

// The message for a value of binary class src cast to class dst: the two classes
// and where each was loaded from.
function jvmCceMsg(src, dst) {
    var head = "class " + src + " cannot be cast to class " + dst + " ("
    var bs = jvmCceBase(src)
    var bd = jvmCceBase(dst)
    if (bs && bd) { return head + src + " and " + dst + " are in module java.base of loader 'bootstrap')" }
    if (!bs && !bd) { return head + src + " and " + dst + " are in unnamed module of loader 'app')" }
    return head + src + jvmCceWhere(bs) + "; " + dst + jvmCceWhere(bd) + ")"
}
function jvmCceWhere(base) {
    if (base) { return " is in module java.base of loader 'bootstrap'" }
    return " is in unnamed module of loader 'app'"
}
// Whether a binary class name is in java.base: a java.* class, an array of one,
// or an array of a primitive ([I, [[D). Everything else, kotlin-stdlib's classes
// and the program's, is in the app loader's unnamed module.
function jvmCceBase(n) {
    var i = 0
    while (n.charAt(i) == "[") { i = i + 1 }
    if (i > 0 && n.charAt(i) != "L") { return true }
    if (i > 0) { i = i + 1 }
    return n.substring(i, i + 5) == "java."
}
