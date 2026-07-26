/* Byte order mark self test.
 * This file is committed THREE times - byte for byte the same text in three
 * encodings, each introduced by its own BOM: tests/kotlin-test-bom-utf8.kt
 * (EF BB BF), -utf16le.kt (FF FE) and -utf32be.kt (00 00 FE FF). A BOM states
 * an encoding rather than a character of the program, so abnf.StripBOM strips
 * it - transcoding UTF-16/UTF-32 to UTF-8 - as the file is read, before the
 * parser or the line-number machinery sees a byte. All three runs must
 * therefore behave identically, whichever engine runs them.
 *
 * The checks are about text rather than arithmetic: the non-ASCII literals
 * only survive if the transcode is done right, and they are the parts that
 * differ most between the three encodings (one byte per character in UTF-8
 * ASCII, always two units in UTF-16, always four in UTF-32).
 *
 * The astral checks are the interesting ones: a character above U+FFFF is a
 * surrogate PAIR in UTF-16, so it is the case where the three encodings differ
 * most (four bytes in UTF-8, two units in UTF-16, one in UTF-32) AND the case
 * the -frozen string pipeline used to destroy, because it takes every string
 * literal apart one UTF-16 code unit at a time and a lone surrogate half is not
 * a Unicode scalar value. See the WTF-8 note in abnf/jsrt.go.
 *
 * main() ends with exitProcess(fails), so the run exits 0 exactly when the
 * whole file arrived intact. **/

var fails = 0

fun check(name: String, got: Int, want: Int) {
    if (got != want) {
        println("FAIL $name: got $got want $want")
        fails++
    }
}

fun checkS(name: String, got: String, want: String) {
    if (got != want) {
        println("FAIL $name: got $got want $want")
        fails++
    }
}

fun main() {
    // The first token of the file must be reachable at all: with the mark left
    // in place the parse would die right here, at line 1 column 1.
    check("plain arithmetic", 6 * 7, 42)

    // Latin-1 range: two bytes in UTF-8, one code unit in UTF-16.
    checkS("umlauts", "wörld", "wörld")
    check("umlaut length", "wörld".length, 5)

    // CJK: three bytes in UTF-8, still one UTF-16 code unit.
    checkS("cjk", "日本語", "日本語")
    check("cjk length", "日本語".length, 3)

    // Mixed, to be sure the pieces still concatenate in the right order and
    // no character was dropped or widened along the way.
    checkS("mixed", "a" + "ö" + "日" + "z", "aö日z")
    check("mixed length", ("a" + "ö" + "日" + "z").length, 4)

    // Astral: four bytes in UTF-8, a surrogate PAIR in UTF-16 - so length is 2,
    // as it is in Kotlin itself, whose String is a UTF-16 sequence too.
    checkS("astral", "🎉", "🎉")
    check("astral length", "🎉".length, 2)

    checkS("astral mixed", "a" + "🎉" + "ö" + "日", "a🎉ö日")
    check("astral mixed length", ("a" + "🎉" + "ö" + "日").length, 5)

    if (fails == 0) { println("Kotlin BOM self test passed") }
    exitProcess(fails)
}
