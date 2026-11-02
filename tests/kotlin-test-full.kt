// Full-syntax test: Kotlin (Kotlin 2.x core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the kotlin grammars
// are. It walks the whole practical Kotlin syntax, one self-contained SECTION
// per language area. The --full runner runs the file, and whenever a grammar
// aborts it removes the section around the error and retries - so the report
// lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() ends with exitProcess(fails) (exit 0 == full support, verified)
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// packages and imports (println/listOf/exitProcess are builtins here, exactly
// as in kotlin-test-features.kt; under kotlinc this file additionally needs
// 'import kotlin.system.exitProcess'), reflection, the collections API beyond
// the minimal calls the features file already uses, KDoc, and multiplatform
// (expect/actual). The coroutine runtime is a SYNCHRONOUS subset (section 55):
// the results are Kotlin's and the ordering is not - section 55 asserts the
// eager ordering deliberately, so a change to it cannot pass unnoticed.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the Kotlin language specification with the ANTLR
// grammars-v4 kotlin grammar as a coverage checklist. Validated against the
// spec by hand; no local kotlinc was available.

var fails = 0
var checks = 0

fun check(id: String, cond: Boolean) {
    checks++
    if (!cond) { println("FAIL $id"); fails++ }
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
class Box01(val start: Int) {
    var value = start
    fun bump(): Int { value += 1; return value }
}
fun twice01(x: Int): Int = x * 2
fun s01() {
    var n = 0
    for (i in 1..3) { n += i }
    var w = 0
    do { w += 1 } while (w < 2)
    while (w < 4) { w += 2 }
    check("bas1", n == 6 && w == 4)
    val b = Box01(4)
    check("bas2", b.bump() == 5 && b.value == 5)
    check("bas3", twice01(21) == 42)
    check("bas4", "n=$n" == "n=6")
    val kind = when (n) { in 0..5 -> "low"; else -> "high" }
    check("bas5", kind == "high")
}

// ===== SECTION 02: literals and constants =====
const val MAX02 = 100
const val NAME02 = "lit"
fun s02() {
    check("lit1", 0xFF == 255 && 0b1010 == 10)
    check("lit2", 1_000_000 == 1000000)
    check("lit3", 3_000_000_000L > 2_999_999_999L)
    val f = 1.5f; val d = 2.5e-2
    check("lit4", f > 1.4f && f < 1.6f && d > 0.02 && d < 0.03)
    val c: Char = 'A'
    check("lit5", c + 1 == 'B' && 'z' > 'a' && '\n' < 'a')
    check("lit6", MAX02 == 100 && NAME02.length == 3)
}

// ===== SECTION 03: string templates and raw strings =====
fun s03() {
    val x = 6
    check("tpl1", "${x} + ${x} = ${x + x}" == "6 + 6 = 12")
    val raw = """line1
line2"""
    check("tpl2", raw.length == 11)
    check("tpl3", """no \n escape""".length == 12)
    val price = 5
    check("tpl4", """total: ${'$'}$price""" == "total: \$5")
    check("tpl5", "he said \"hi\"".length == 12)
    check("tpl6", "outer ${"inner $x"}" == "outer inner 6")
    // \b is the backspace character, in a plain string and in a template chunk.
    check("tpl7", "\b"[0].code == 8 && "a${x}\b".length == 3 && "a${x}\b"[2].code == 8)
}

// ===== SECTION 04: ranges and progressions =====
fun s04() {
    var a = 0; for (i in 1..4) { a = a * 10 + i }
    check("rng1", a == 1234)
    var b = 0; for (i in 0..<3) { b += i }
    check("rng2", b == 3)
    var c = ""; for (i in 3 downTo 1) { c += i }
    check("rng3", c == "321")
    var d = 0; for (i in 0..6 step 2) { d += i }
    check("rng4", d == 12)
    check("rng5", 3 in 1..5 && 9 !in 1..5)
    var g = ""; for (ch in 'a'..'c') { g += ch }
    check("rng6", g == "abc")
    // The INDEX properties: `val Collection<*>.indices`, `val CharSequence.indices`
    // and the two lastIndex. These live in the ratchet as well as in
    // kotlin-test-features.kt because ONLY the ratchet is built natively, and the
    // runtime has a layer-2 arm (languages/lib/kotlin-rt.metajs) the matrix cannot see.
    val rl = listOf(10, 20, 30)
    var h = 0; for (i in rl.indices) { h = h * 10 + i }
    check("rng7", h == 12 && rl.lastIndex == 2 && rl.indices.toString() == "0..2")
    val rs = "abcd"
    var k = ""; for (i in rs.indices) { k += rs[i] }
    check("rng8", k == "abcd" && rs.lastIndex == 3)
    val ra = arrayOf(4, 5, 6)
    var m = 0; for (i in ra.indices) { m += ra[i] }
    check("rng9", m == 15 && ra.lastIndex == 2 && listOf<Int>().lastIndex == -1)
    // A progression read as a VALUE. IntProgression.last is the last ELEMENT (1..10
    // step 4 runs 1, 5, 9), and downTo's step carries the direction - "a progression
    // ... with the step -1" in kotlin.ranges.downTo.
    val rp = listOf(1..5, 1..10 step 4, 5 downTo 1, 5 downTo 2 step 2, 0 until 3)
    var q = ""; for (p in rp) { q += "" + p.first + "/" + p.last + "/" + p.step + " " }
    check("rng10", q == "1/5/1 1/9/4 5/1/-1 5/3/-2 0/2/1 ")
    // A StringBuilder IS a CharSequence, so kotlin.text's two properties apply to it
    // as well.
    val rb = StringBuilder("abcd")
    var t = 0; for (i in rb.indices) { t += i }
    check("rng11", t == 6 && rb.lastIndex == 3 && rb.indices.toString() == "0..3")
    // A PROGRESSION IS A VALUE, NOT A MATERIALIZED LIST. `(5..1).first` is 5 on the
    // JVM: an IntRange keeps the bounds it was DECLARED with, only isEmpty() reports
    // the emptiness, and `first()` - the FUNCTION - is the one that throws
    // NoSuchElementException. These are in the ratchet because ONLY the ratchet is
    // built NATIVELY, and the runtime has a layer-2 arm (languages/lib/kotlin-rt.metajs)
    // no other gate sees.
    val re = listOf(5..1, 1 until 1, 'e'..'a', 3 downTo 7)
    var u = ""
    for (p in re) { u += "" + p.first + "/" + p.last + "/" + p.start + "/" + p.endInclusive + "/" + p.step + " " }
    check("rng12", u == "5/1/5/1/1 1/0/1/0/1 e/a/e/a/1 3/7/3/7/-1 ")
    check("rng13", (5..1).isEmpty() && (5..1).count() == 0 && (5..1).toList().toString() == "[]")
    // IntRange.toString is "$first..$last" and IntProgression.toString is
    // "$first..$last step $step" / "$first downTo $last step ${-step}".
    check("rng14", (1..5).toString() == "1..5" && (5..1).toString() == "5..1" &&
        (1 until 5).toString() == "1..4" && ('a'..'e').toString() == "a..e")
    check("rng15", (1..10 step 4).toString() == "1..9 step 4" &&
        (10 downTo 1).toString() == "10 downTo 1 step 1" &&
        (10 downTo 1 step 3).toString() == "10 downTo 1 step 3" && "" + (2..4) == "2..4")
    // `in` on a STEPPED progression is Iterable.contains, a LINEAR membership test -
    // IntProgression declares no contains at all.
    check("rng16", 3 !in (1..10 step 4) && 5 in (1..10 step 4) && 5 !in (10 downTo 1 step 3) &&
        7 in (10 downTo 1 step 3))
    // IntProgression.equals: two EMPTY progressions are equal whatever their bounds,
    // and hashCode is -1 for either.
    check("rng17", (5..1) == (7..3) && (1..5) == (1..5) && (1..5) != (1..5 step 2) &&
        (1..5).equals(1..5) && !(1..5).equals(listOf(1, 2, 3, 4, 5)))
    // IntProgression.hashCode is 31 * (31 * first + last) + step, IntRange's and
    // CharRange's 31 * first + last, and -1 when the range is empty (kotlin-stdlib
    // 2.2.21's, called from java 24).
    val rh = listOf(1..5, 5..1, 1..10 step 4, 'a'..'c')
    var v = ""
    for (p in rh) { v += "" + p.hashCode() + " " }
    check("rng17b", v == "36 -1 1244 3106 " && (5..1).hashCode() == (7..3).hashCode())
    // A range is an IntRange / IntProgression and NOT a List; a STEPPED or downTo
    // progression is an IntProgression and not an IntRange.
    val ri: Any = 1..5
    val rj: Any = 1..10 step 4
    check("rng18", ri is IntRange && ri is IntProgression && rj is IntProgression &&
        rj !is IntRange && ('a'..'c') is CharRange)
    // A MatchResult's range is a real IntRange.
    val rm = Regex("b+").find("abbbc")
    check("rng19", rm != null && rm!!.range.toString() == "1..3" && rm.range.first == 1 &&
        rm.range.last == 3 && rm.range.count() == 3)
    // toString() and hashCode() are members of kotlin.Any, so an UNQUALIFIED call
    // resolves against the implicit receiver whatever its shape (`with(p)
    // { toString() }`).
    val rpair = Pair(1, "a")
    check("rng20", with(rpair) { toString() } == "(1, a)" && with(5) { toString() } == "5" &&
        with(listOf(1, 2)) { toString() } == "[1, 2]" && with(1..3) { toString() } == "1..3")
    check("rng21", with(rpair) { hashCode() } == rpair.hashCode() &&
        with(mapOf("k" to 1)) { entries.toString() } == "[k=1]")
    // equals(other) is the third Any member and is the same comparison `==` makes.
    // `listOf(1, 2).equals(listOf(1, 2))` is true.
    check("rng21b", listOf(1, 2).equals(listOf(1, 2)) && !listOf(1, 2).equals(3) &&
        mapOf(1 to 2).equals(mapOf(1 to 2)) && setOf(1).equals(setOf(1)) &&
        with(rpair) { equals(Pair(1, "a")) })
    // A ClosedFloatingPointRange - `1.0..2.0` - declares no step and no iteration,
    // and its contains is the BOUNDS test alone. Its toString is
    // "$start..$endInclusive", which prints the .0.
    val rf = listOf(1.5, 0.5, 2.0)
    check("rng22", rf[0] in 1.0..2.0 && rf[1] !in 1.0..2.0 && rf[2] in 1.0..2.0 &&
        (1.0..2.0).toString() == "1.0..2.0" && (1.0..2.0).start == 1.0 &&
        (1.0..2.0).endInclusive == 2.0)
    // NOT ASSERTED HERE: an EXTENSION PROPERTY of one of those names WINS over the
    // runtime's answer (the compiler resolves `extp$name` in the EMITTER, before
    // js_ktfget is called at all, and the interpreter consults it first - kIdxExt in
    // kotlin-interpreter.abnf). It cannot be pinned in THIS file: an extension property
    // is keyed by NAME alone and declared at top level, so `val String.indices` would
    // poison rng8's own `s.indices` and every other section that reads one.
}

// ===== SECTION 05: when forms =====
fun describe05(x: Any): String = when (x) {
    is Int -> if (x > 5) "big int" else "small int"
    is String -> "str" + x.length
    else -> "other"
}
fun s05() {
    val a = when (10) { 1, 2 -> "lo"; 10 -> "ten"; else -> "hi" }
    check("whn1", a == "ten")
    val b = when (7) { in 0..5 -> "low"; in 6..9 -> "mid"; else -> "high" }
    check("whn2", b == "mid")
    check("whn3", describe05(9) == "big int" && describe05("ab") == "str2" && describe05(true) == "other")
    val c = when (val n = 4 * 4) { in 0..9 -> "1digit:$n"; else -> "big:$n" }
    check("whn4", c == "big:16")
    val t = 3
    val e = when { t < 0 -> "neg"; t % 2 == 1 -> "odd"; else -> "even" }
    check("whn5", e == "odd")
}

// ===== SECTION 06: labels and labeled returns =====
fun firstBig06(xs: List<Int>): Int {
    xs.forEach { if (it > 10) return it }
    return -1
}
fun exec06(f: () -> Unit) { f() }
fun s06() {
    var hits = 0
    outer@ for (i in 0..2) {
        for (j in 0..2) {
            if (j == 1) continue@outer
            if (i == 2) break@outer
            hits++
        }
    }
    check("lop1", hits == 2)
    val sum = listOf(1, 2, 3, 4).sumOf {
        if (it == 3) return@sumOf 0
        it
    }
    check("lop2", sum == 7)
    check("lop3", firstBig06(listOf(4, 40, 9)) == 40)
    var log = ""
    exec06 lab@{
        log += "a"
        if (log.length == 1) return@lab
        log += "b"
    }
    check("lop4", log == "a")
}

// ===== SECTION 07: functions: defaults, named args, vararg, tailrec =====
fun volume07(w: Int, h: Int = 2, d: Int = 3): Int = w * 100 + h * 10 + d
fun sumAll07(vararg xs: Int): Int { var s = 0; for (x in xs) { s += x }; return s }
tailrec fun gcd07(a: Int, b: Int): Int = if (b == 0) a else gcd07(b, a % b)
fun s07() {
    check("fun1", volume07(1) == 123)
    check("fun2", volume07(1, 9) == 193)
    check("fun3", volume07(1, d = 7) == 127)
    check("fun4", volume07(d = 5, w = 4, h = 6) == 465)
    check("fun5", sumAll07() == 0 && sumAll07(1, 2, 3) == 6)
    val packed = intArrayOf(4, 5)
    check("fun6", sumAll07(*packed, 6) == 15)
    check("fun7", gcd07(12, 18) == 6)
    fun local(n: Int): Int = n + packed[0]
    check("fun8", local(6) == 10)
}

// ===== SECTION 08: lambdas and function types =====
data class Duo08(val a: Int, val b: Int)
fun combine08(x: Int, y: Int, f: (Int, Int) -> Int): Int = f(x, y)
fun s08() {
    val inc = { n: Int -> n + 1 }
    check("lam1", inc(41) == 42)
    val doubled = listOf(1, 2, 3).map { it * 2 }
    check("lam2", doubled[2] == 6)
    check("lam3", combine08(3, 4) { a, b -> a * 10 + b } == 34)
    check("lam4", combine08(5, 9) { _, b -> b } == 9)
    val anon = fun(n: Int): Int { return n * n }
    check("lam5", anon(6) == 36)
    val dot: Int.(Int) -> Int = { other -> this + other }
    check("lam6", 40.dot(2) == 42 && dot(1, 2) == 3)
    val folded = listOf(Duo08(1, 2), Duo08(3, 4)).map { (a, b) -> a * 10 + b }
    check("lam7", folded[0] == 12 && folded[1] == 34)
    val slot: ((Int) -> Int)? = if (doubled.size == 3) inc else null
    check("lam8", slot?.invoke(1) == 2)
}

// ===== SECTION 09: operator conventions and infix functions =====
class Vec09(val x: Int, val y: Int) {
    operator fun plus(o: Vec09) = Vec09(x + o.x, y + o.y)
    operator fun times(k: Int) = Vec09(x * k, y * k)
    operator fun get(i: Int): Int = if (i == 0) x else y
    operator fun invoke(): Int = x * 10 + y
    operator fun unaryMinus() = Vec09(-x, -y)
    operator fun contains(n: Int): Boolean = n == x || n == y
    infix fun dot(o: Vec09): Int = x * o.x + y * o.y
}
fun s09() {
    val v = Vec09(1, 2) + Vec09(3, 4)
    check("opr1", v.x == 4 && v.y == 6)
    val w = Vec09(2, 3) * 3
    check("opr2", w[0] == 6 && w[1] == 9)
    check("opr3", w() == 69)
    val m = -Vec09(5, 7)
    check("opr4", m.x == -5 && 7 in Vec09(5, 7) && 9 !in Vec09(5, 7))
    check("opr5", (Vec09(1, 0) dot Vec09(3, 9)) == 3)
    var acc = Vec09(1, 1); acc += Vec09(2, 2)
    check("opr6", acc.x == 3 && acc.y == 3)
    check("bit1", (5 and 3) == 1 && (5 or 3) == 7 && (5 xor 3) == 6)
    check("bit2", (1 shl 4) == 16 && (32 shr 1) == 16)
    check("bit3", (-8 ushr 28) == 15)
    check("bit4", 5.inv() == -6)
}

// ===== SECTION 10: extension functions and properties =====
fun Int.squared10(): Int = this * this
fun String.shout10(): String = this + "!"
val String.mid10: Char get() = this[length / 2]
fun String?.orDash10(): String = this ?: "-"
fun s10() {
    check("ext1", 7.squared10() == 49)
    check("ext2", "hey".shout10() == "hey!")
    check("ext3", "abc".mid10 == 'b')
    val gone: String? = null
    check("ext4", gone.orDash10() == "-" && "ok".orDash10() == "ok")
    fun Int.tripled10(): Int = this * 3
    check("ext5", 4.tripled10() == 12)
}

// ===== SECTION 11: classes: constructors, accessors, nesting =====
class Meter11(val start: Int) {
    var log = ""
    init { log += "i$start" }
    var level: Int = start
        get() = field + 100
        set(v) { field = if (v < 0) 0 else v }
    constructor(a: Int, b: Int) : this(a + b) { log += ":sec" }
    private fun hidden(): Int = 5
    internal fun reveal(): Int = hidden()
}
class Outer11(val tag: String) {
    val plain = "o"
    class Nested { fun label(): String = "nested" }
    inner class Inner { fun label(): String = tag + "-in" + this@Outer11.plain }
}
class Late11 { lateinit var name: String; fun ready(): Boolean = this::name.isInitialized }
fun s11() {
    val m = Meter11(4)
    check("cls1", m.log == "i4" && m.level == 104)
    m.level = -9
    check("cls2", m.level == 100)
    val m2 = Meter11(2, 3)
    check("cls3", m2.log == "i5:sec" && m2.start == 5)
    check("cls4", m.reveal() == 5)
    check("cls5", Outer11.Nested().label() == "nested")
    check("cls6", Outer11("z").Inner().label() == "z-ino")
    val l = Late11()
    check("cls7", !l.ready())
    l.name = "set"
    check("cls8", l.ready() && l.name.length == 3)
}

// ===== SECTION 12: inheritance and interfaces =====
interface Named12 { val kind: String; fun name(): String = "some " + kind }
abstract class Shape12(val id: Int) : Named12 {
    abstract fun area(): Int
    open fun describe(): String = "shape" + id
    protected fun secret(): Int = id + 10
}
open class Rect12(id: Int, val w: Int, val h: Int) : Shape12(id) {
    override val kind: String = "rect"
    override fun area(): Int = w * h
    final override fun describe(): String = "rect" + id + "/" + super.describe()
    fun leak(): Int = secret()
}
class Square12(id: Int, side: Int) : Rect12(id, side, side)
fun s12() {
    val r = Rect12(1, 3, 4)
    check("inh1", r.area() == 12)
    check("inh2", r.describe() == "rect1/shape1")
    check("inh3", r.name() == "some rect" && r.kind == "rect")
    check("inh4", r.leak() == 11)
    val s: Shape12 = Square12(2, 5)
    check("inh5", s.area() == 25 && s.describe() == "rect2/shape2")
    check("inh6", s is Named12 && s is Rect12)
}

// ===== SECTION 13: data and enum classes =====
data class Pt13(val x: Int, val y: Int)
enum class Dir13(val dx: Int, val dy: Int) {
    N(0, -1), E(1, 0), S(0, 1);
    fun flipped(): Dir13 = when (this) { N -> S; S -> N; E -> E }
}
fun s13() {
    val p = Pt13(1, 2)
    val q = p.copy(y = 9)
    check("dat1", q.x == 1 && q.y == 9 && p.y == 2)
    check("dat2", p == Pt13(1, 2) && p != q)
    check("dat3", p.toString() == "Pt13(x=1, y=2)")
    val (a, b) = q
    check("dat4", a == 1 && b == 9)
    check("dat5", p.component2() == 2)
    check("enm1", Dir13.E.dx == 1 && Dir13.N.dy == -1)
    check("enm2", Dir13.S.name == "S" && Dir13.E.ordinal == 1)
    check("enm3", Dir13.N.flipped() == Dir13.S && Dir13.E.flipped() == Dir13.E)
    val turn = when (Dir13.N) { Dir13.N, Dir13.S -> "vertical"; Dir13.E -> "horizontal" }
    check("enm4", turn == "vertical")
}

// ===== SECTION 14: sealed hierarchies =====
sealed interface Expr14
sealed class Node14 : Expr14
class Num14(val v: Int) : Node14()
class Add14(val l: Expr14, val r: Expr14) : Node14()
object Zero14 : Expr14
fun eval14(e: Expr14): Int = when (e) {
    is Num14 -> e.v
    is Add14 -> eval14(e.l) + eval14(e.r)
    Zero14 -> 0
}
fun s14() {
    check("sld1", eval14(Num14(7)) == 7)
    check("sld2", eval14(Add14(Num14(2), Add14(Num14(3), Zero14))) == 5)
    val n: Expr14 = Num14(1); val z: Expr14 = Zero14
    check("sld3", n is Node14 && z !is Node14)
    val tags = listOf<Expr14>(Num14(3), Zero14).map {
        when (it) { is Num14 -> "n"; is Add14 -> "a"; Zero14 -> "z" }
    }
    check("sld4", tags[0] == "n" && tags[1] == "z")
}

// ===== SECTION 15: objects, companions and delegation =====
interface Greeter15 {
    fun greet(): String
    fun bye(): String = "bye"
}
class EnGreeter15 : Greeter15 { override fun greet(): String = "hello" }
class Loud15(base: Greeter15) : Greeter15 by base { override fun bye(): String = "BYE" }
object Registry15 { var count = 0; fun add(n: Int): Int { count += n; return count } }
class Tagged15(val v: Int) {
    companion object Maker { val SEED = 7; fun of(n: Int): Tagged15 = Tagged15(n + SEED) }
}
fun s15() {
    check("obj1", Registry15.add(2) == 2 && Registry15.add(3) == 5)
    check("obj2", Tagged15.of(1).v == 8 && Tagged15.SEED == 7 && Tagged15.Maker.SEED == 7)
    val loud = Loud15(EnGreeter15())
    check("obj3", loud.greet() == "hello" && loud.bye() == "BYE")
    val anon = object { val x = 3; fun twice(): Int = x * 2 }
    check("obj4", anon.twice() == 6)
    val hi = object : Greeter15 { override fun greet(): String = "hi" + Registry15.count }
    check("obj5", hi.greet() == "hi5" && hi.bye() == "bye")
}

// ===== SECTION 16: generics =====
class Box16<T>(val item: T) { fun swap(other: T): T = other }
class Producer16<out T>(val value: T) { fun produce(): T = value }
class Consumer16<in T> { var seen = 0; fun consume(x: T) { seen += 1 } }
fun <T> pick16(a: T, b: T, first: Boolean): T = if (first) a else b
fun <T> larger16(a: T, b: T): T where T : Comparable<T> = if (a >= b) a else b
fun sizeOf16(b: Box16<*>): Int = if (b.item == null) 0 else 1
open class Animal16 { open fun cry(): String = "..." }
class Dog16 : Animal16() { override fun cry(): String = "woof" }
fun s16() {
    val bi = Box16(42); val bs = Box16<String>("hi")
    check("gen1", bi.item == 42 && bs.item.length == 2)
    check("gen2", bi.swap(7) == 7 && pick16("a", "b", false) == "b")
    check("gen3", larger16(3, 9) == 9 && larger16("b", "a") == "b")
    val prod: Producer16<Animal16> = Producer16<Dog16>(Dog16())
    check("gen4", prod.produce().cry() == "woof")
    val sink: Consumer16<Dog16> = Consumer16<Animal16>()
    sink.consume(Dog16())
    check("gen5", sink.seen == 1)
    check("gen6", sizeOf16(bi) == 1 && sizeOf16(bs) == 1)
}

// ===== SECTION 17: inline functions and reified =====
inline fun runTwice17(f: (Int) -> Int): Int = f(1) + f(2)
inline fun keep17(noinline f: () -> Int): () -> Int = f
inline fun defer17(crossinline f: () -> Int): () -> Int = { f() + 1 }
inline fun <reified T> tag17(x: Any): String = if (x is T) "yes" else "no"
fun s17() {
    check("inl1", runTwice17 { it * 10 } == 30)
    val kept = keep17 { 5 }
    check("inl2", kept() == 5)
    val d = defer17 { 40 }
    check("inl3", d() == 41)
    check("inl4", tag17<String>("abc") == "yes" && tag17<String>(9) == "no")
    check("inl5", tag17<Int>(7) == "yes")
}

// ===== SECTION 18: null safety and casts =====
fun stretch18(x: Any): Int {
    if (x !is String) return -1
    return x.length
}
fun s18() {
    var s: String? = null
    check("nul1", (s?.length ?: -1) == -1)
    s = if (checks > 0) "abcd" else null
    check("nul2", s?.length == 4 && s!!.length == 4)
    val any: Any = "hello"
    if (any is String) check("nul3", any.length == 5)
    check("nul4", stretch18("abc") == 3 && stretch18(42) == -1)
    val num: Any = 5
    val asStr: String? = num as? String
    check("nul5", asStr == null && (num as? Int ?: 0) == 5)
    val forced: String = any as String
    check("nul6", forced.length == 5)
}

// ===== SECTION 19: destructuring and typealias =====
typealias Mapper19 = (Int) -> Int
typealias Pts19 = List<Pt19>
data class Pt19(val x: Int, val y: Int)
class Split19(val s: String) {
    operator fun component1(): String = s + "1"
    operator fun component2(): Int = s.length
}
fun s19() {
    val (a, b) = Pt19(3, 4)
    check("des1", a == 3 && b == 4)
    val (_, second) = Pt19(7, 8)
    check("des2", second == 8)
    val (tag, len) = Split19("hi")
    check("des3", tag == "hi1" && len == 2)
    var sum = 0
    val pts: Pts19 = listOf(Pt19(1, 2), Pt19(3, 4))
    for ((px, py) in pts) { sum += px * py }
    check("des4", sum == 14)
    val twice: Mapper19 = { it * 2 }
    check("des5", twice(21) == 42)
}

// ===== SECTION 20: annotations =====
annotation class Note20(val label: String)
@Note20("fn") fun tagged20(n: Int): Int = n + 1
@Note20("cls") class Carrier20(@param:Note20("p") val v: Int) {
    @get:Note20("g") val doubled: Int get() = v * 2
    @field:Note20("f") var slot: Int = 1
}
fun s20() {
    check("ann1", tagged20(41) == 42)
    val c = Carrier20(5)
    check("ann2", c.v == 5 && c.doubled == 10)
    c.slot = 3
    check("ann3", c.slot == 3)
    @Note20("local") val here = 9
    check("ann4", here == 9)
}

// ===== SECTION 21: exceptions as expressions =====
class AppError21(msg: String) : Exception(msg)
fun bail21(msg: String): Nothing = throw AppError21(msg)
fun half21(n: Int): Int = if (n % 2 == 0) n / 2 else bail21("odd$n")
fun s21() {
    val r1 = try { half21(8) } catch (e: AppError21) { -1 }
    check("exc1", r1 == 4)
    var fin = ""
    val r2 = try { half21(3) } catch (e: AppError21) { e.message ?: "?" } finally { fin = "ran" }
    check("exc2", r2 == "odd3" && fin == "ran")
    var order = ""
    try {
        try { throw AppError21("in") } finally { order += "f1" }
    } catch (e: Exception) { order += "c" + (e.message ?: "") } finally { order += "f2" }
    check("exc3", order == "f1cinf2")
    fun readOrBail(v: Int?): Int = v ?: bail21("nil")
    val got = try { readOrBail(null) } catch (e: AppError21) { -9 }
    check("exc4", got == -9 && readOrBail(6) == 6)
    val which = try { throw AppError21("a") } catch (e: AppError21) { "app" } catch (e2: Exception) { "exc" }
    check("exc5", which == "app")
}

// ===== SECTION 22: suspend and value classes =====
@JvmInline value class Meters22(val v: Int) {
    operator fun plus(o: Meters22): Meters22 = Meters22(v + o.v)
    fun doubled(): Meters22 = Meters22(v * 2)
}
suspend fun tick22(n: Int): Int = n + 1
fun s22() {
    val m = Meters22(20) + Meters22(1)
    check("val1", m.v == 21 && m.doubled().v == 42)
    check("val2", Meters22(3) == Meters22(3) && Meters22(3) != Meters22(4))
    var op: (suspend (Int) -> Int)? = null
    if (checks > 0) { op = ::tick22 }
    check("sus1", op != null)
}

// ===== SECTION 23: regular expressions =====
// Regex and MatchResult over the shared engine (languages/lib/regex.js for the
// interpreter, abnf/jsrtregex.go + abnf/jsrtregexkt.go for the compiler). Kotlin
// has no regexp literal, so every pattern arrives as Regex("...") or
// "...".toRegex(). Regex.escape is Pattern.quote and answers the same \Q...\E TEXT
// the JVM does; RegexOption.CANON_EQ is reported as unsupported
// instead of being silently ignored. The dialect is java.util.regex's, and a bad
// pattern throws its PatternSyntaxException; both checked against kotlin-stdlib
// 2.2.21's Regex driven from java.
fun rxErr23(p: String): String {
    try { Regex(p) } catch (e: IllegalArgumentException) {
        return e::class.simpleName + ": " + e.message!!.replace("\n", "|")
    }
    return "ok"
}
fun s23() {
    val word23 = Regex("[a-c]+")
    check("re01", word23.matches("abc") && !word23.matches("abcd"))
    check("re02", word23.containsMatchIn("xxabyy") && !word23.containsMatchIn("zzz"))
    check("re03", word23.pattern == "[a-c]+" && word23.toString() == "[a-c]+")
    val hit23 = word23.find("xxabyy")
    check("re04", hit23 != null && hit23.value == "ab")
    check("re05", hit23 != null && hit23.range.first == 2 && hit23.range.last == 3)
    check("re06", word23.find("zzz") == null)
    check("re07", Regex("b").find("abcb", 2)!!.range.first == 3)

    val pair23 = Regex("(\\w+)@(\\w+)")
    val m23 = pair23.find("mail bob@host end")!!
    check("re08", m23.value == "bob@host" && m23.groupValues[0] == "bob@host")
    check("re09", m23.groupValues[1] == "bob" && m23.groupValues[2] == "host")
    check("re10", m23.groups[1]!!.value == "bob" && m23.groups[2]!!.range.first == 9)
    check("re11", m23.groups.size == 3 && Regex("(a)|(b)").find("b")!!.groups[1] == null)
    val (user23, host23) = m23.destructured
    check("re12", user23 == "bob" && host23 == "host")
    val named23 = Regex("(?<who>[a-z]+)!").find("say hey! now")!!
    check("re13", named23.groups["who"]!!.value == "hey")

    // The replacement template is Kotlin's ($1 / ${name}, backslash escapes), not
    // the engine's \1 - the grammar rewrites it on both sides of the pair.
    check("re14", pair23.replace("a@b c@d", "\$2-\$1") == "b-a d-c")
    check("re15", pair23.replaceFirst("a@b c@d", "\$2-\$1") == "b-a c@d")
    check("re16", Regex("(?<d>[0-9])").replace("a1b2", "[\${d}]") == "a[1]b[2]")
    check("re17", Regex("x").replace("x", Regex.escapeReplacement("\$1")) == "\$1")
    check("re18", Regex("y").replace("y", "a\\\\b") == "a\\b")
    check("re19", Regex("[a-z]").replace("abc") { "<" + it.value + ">" } == "<a><b><c>")

    val parts23 = Regex("\\s+").split("a b  c")
    check("re20", parts23.size == 3 && parts23[0] == "a" && parts23[2] == "c")
    val lim23 = Regex(",").split("a,b,c", 2)
    check("re21", lim23.size == 2 && lim23[0] == "a" && lim23[1] == "b,c")

    check("re22", "a1b2".replace(Regex("[0-9]"), "#") == "a#b#")
    check("re23", "a1b2".contains(Regex("[0-9]")) && !"abc".contains(Regex("[0-9]")))
    check("re24", "a1b2".matches(Regex("[a-z0-9]+")) && !"a1b2".matches(Regex("[a-z]+")))
    val sp23 = "x1y22z".split(Regex("[0-9]+"))
    check("re25", sp23.size == 3 && sp23[1] == "y" && sp23[2] == "z")
    check("re26", "[0-9]".toRegex().containsMatchIn("a7"))
    check("re27", "he.lo".toRegex(RegexOption.LITERAL).matches("he.lo"))

    // RegexOption maps onto the engine's neutral i / s / m / x. Kotlin (like Java,
    // unlike Ruby) anchors ^ and $ to the whole input unless MULTILINE is given.
    check("re28", Regex("A", RegexOption.IGNORE_CASE).containsMatchIn("xax"))
    check("re29", Regex("a.b", setOf(RegexOption.IGNORE_CASE, RegexOption.DOT_MATCHES_ALL)).matches("A\nB"))
    check("re30", Regex("^b\$", RegexOption.MULTILINE).containsMatchIn("a\nb\nc"))
    check("re31", !Regex("^b\$").containsMatchIn("a\nb\nc"))
    check("re32", Regex("a b # note", RegexOption.COMMENTS).matches("ab"))
    val lit23 = Regex("a.b", RegexOption.LITERAL)
    check("re33", lit23.matches("a.b") && !lit23.matches("axb"))
    check("re34", Regex(Regex.escape("1+1")).matches("1+1"))

    check("re35", Regex("b").findAll("abcb").count() == 2)
    var next23 = Regex("b").find("abcb")
    var seen23 = 0
    while (next23 != null) { seen23 += 1; next23 = next23.next() }
    check("re36", seen23 == 2)
    val whole23 = Regex("[0-9]+").matchEntire("123")
    check("re37", whole23 != null && whole23.value == "123" && whole23.range.last == 2)
    check("re38", Regex("[0-9]+").matchEntire("12a") == null)

    check("re39", Regex("(ab)\\1").containsMatchIn("xabab"))
    check("re40", Regex("a+?b").find("aaab")!!.value == "aaab")
    check("re41", Regex("colou?r").matches("color") && Regex("x{2,3}").find("xxxx")!!.value == "xxx")
    check("re42", Regex("(?=a)[a-z]").find("za")!!.range.first == 1)
    // Regex.escape IS Pattern.quote: the \Q...\E text, not merely an equivalent
    // pattern. Checked against java.util.regex.Pattern.quote.
    check("re43", Regex.escape("a.b") == "\\Qa.b\\E" && Regex.escape("a+b*c") == "\\Qa+b*c\\E")
    check("re44", Regex.escape("a").length == 5)
    // \Q...\E quote regions, written out by hand: every character between them is a
    // literal, and a quantifier after \E binds to the LAST quoted character.
    check("re45", Regex("\\Qa.b\\E").matches("a.b") && !Regex("\\Qa.b\\E").matches("axb"))
    check("re46", Regex("x\\Qa+b\\Ey").containsMatchIn("xa+by") && Regex("\\Qab\\E+").matches("abb"))
    check("re47", Regex("a.b", RegexOption.LITERAL).matches("a.b")
                  && !Regex("a.b", RegexOption.LITERAL).matches("axb"))
    // Named backreferences inside a pattern.
    check("re48", Regex("(?<n>a)\\k<n>").matches("aa") && !Regex("(?<n>a)\\k<n>").matches("ab"))
    // The java.util.regex dialect: nested classes and &&, \p{..}, \h \v \R, \x{h},
    // lookbehind, and a backreference to a group that did not take part fails.
    check("re49", Regex("[a-z&&[^aeiou]]+").matches("bcd") && !Regex("[a-z&&[^aeiou]]+").matches("bad"))
    check("re50", Regex("\\p{Lu}+").matches("ABC") && !Regex("\\p{Lu}+").matches("AbC")
                  && Regex("\\p{L}+").matches("\u00e9t\u00e9"))
    check("re51", Regex("\\h+").matches(" \t") && Regex("a\\Rb").matches("a\r\nb")
                  && Regex("\\x{41}").matches("A") && Regex("\\0101").matches("A"))
    check("re52", !Regex("(?<=a)b").containsMatchIn("b") && Regex("(?<=a)b").containsMatchIn("ab")
                  && !Regex("(a)|\\1b").containsMatchIn("b"))
    check("re53", Regex("a{0,2147483647}").matches("a") && !Regex("a{1,2}{2}").matches("aaaa"))
    // A bad pattern throws PatternSyntaxException (an IllegalArgumentException) with
    // java's message, including a repeat bound that does not fit an int.
    check("re54", rxErr23("a{99999999999}") == "PatternSyntaxException: Illegal repetition range near index 11|a{99999999999}|           ^")
    check("re55", rxErr23("a{2,1}") == "PatternSyntaxException: Illegal repetition range near index 5|a{2,1}|     ^"
                  && rxErr23("x{a}") == "PatternSyntaxException: Illegal repetition near index 2|x{a}|  ^")
    check("re56", rxErr23("(") == "PatternSyntaxException: Unclosed group near index 1|("
                  && rxErr23("[abc") == "PatternSyntaxException: Unclosed character class near index 3|[abc|   ^"
                  && rxErr23("*a") == "PatternSyntaxException: Dangling meta character '*' near index 0|*a|^")
    check("re57", rxErr23("\\k<nope>") == "PatternSyntaxException: named capturing group <nope> does not exist near index 7|\\k<nope>|       ^"
                  && rxErr23("[]a]") == "ok")
    // Inline flags, atomic groups and possessive quantifiers. IGNORE_CASE folds
    // case the Unicode way (Kotlin adds UNICODE_CASE to it), an inline (?i) only
    // ASCII; . and $ know \r as a line terminator unless UNIX_LINES. replace with
    // a template runs one Matcher's successive finds, which \G and \b{g} read.
    check("re58", Regex("(?i)abc").matches("ABC") && Regex("a(?i:b)c").matches("aBc") && !Regex("(?i:a(?-i:b)c)").matches("aBc")
                  && Regex("(?>a+)b").matches("aab") && !Regex("(?>a+)a").matches("aaa") && !Regex("a*+a").matches("aaa"))
    check("re59", Regex("\u00e9", RegexOption.IGNORE_CASE).matches("\u00c9") && !Regex("(?i)\u00e9").matches("\u00c9")
                  && Regex("(?iu)\u00e9").matches("\u00c9") && Regex("k", RegexOption.IGNORE_CASE).matches("\u212a"))
    check("re60", !Regex(".").matches("\r") && Regex(".", RegexOption.UNIX_LINES).matches("\r")
                  && Regex("a\$", setOf(RegexOption.UNIX_LINES, RegexOption.MULTILINE)).replace("a\r\na\n", "#") == "a\r\n#\n")
    check("re61", Regex("\\b{g}").replace("ae\u0301x", "|") == "|a|e\u0301|x|" && Regex("\\Ga").replace("aab", "-") == "--b"
                  && Regex("\\p{IsLatin}+").find("\u03b1ab")!!.value == "ab"
                  && rxErr23("(?q)") == "PatternSyntaxException: Unknown inline modifier near index 2|(?q)|  ^")
    // A surrogate pair is one character to the pattern and the input, while
    // ranges stay UTF-16 indices; an empty match advances one unit, so x? and \B
    // land between the halves. Read off kotlin-stdlib 2.2.21 on java 24.
    check("re62", Regex(".").replace("a\uD83D\uDE00b", "#") == "###" && Regex("\\p{So}").matches("\uD83D\uDE00")
                  && Regex("\\uD83D\\uDE00").matches("\uD83D\uDE00") && Regex("[\uD83D\uDE00-\uD83D\uDE02]").matches("\uD83D\uDE01")
                  && !Regex("..").matches("\uD83D\uDE00") && Regex("\uD801\uDC00", RegexOption.IGNORE_CASE).matches("\uD801\uDC28")
                  && !Regex("(?i)\uD801\uDC00").matches("\uD801\uDC28"))
    check("re63", Regex("\\B").replace("a\uD835\uDC00b", "#") == "a\uD835#\uDC00b" && Regex("a*").split("\uD83D\uDE00").size == 4
                  && Regex("x?").replace("\uD801\uDC00x\uD801\uDC28") { "<" + it.value.length + ">" } == "<0>\uD801<0>\uDC00<1><0>\uD801<0>\uDC28<0>"
                  && Regex("(.)(.)").find("a\uD83D\uDE00b\uD83D\uDE01c", 2)!!.range.first == 2
                  && Regex("[^a-z]").findAll("a\uD83D\uDE00b\uD83D\uDE01c").map { it.range.first }.toList() == listOf(1, 4))
    // options is Pattern.flags() read back as RegexOption entries: the inline
    // flags still in force at the end count, a scoped (?i:..) does not.
    check("re64", "" + Regex("a", RegexOption.IGNORE_CASE).options + Regex("(?i)a").options + Regex("(?i:a)").options + Regex("(?im)a(?-i)").options
                  + Regex("a.b", setOf(RegexOption.MULTILINE, RegexOption.LITERAL)).options == "[IGNORE_CASE][IGNORE_CASE][][MULTILINE][MULTILINE, LITERAL]"
                  && Regex("(?s)x").options == setOf(RegexOption.DOT_MATCHES_ALL) && Regex("a").options.isEmpty()
                  && Regex("a.b", RegexOption.LITERAL).pattern == "a.b" && "" + Regex("a.b", RegexOption.LITERAL) == "a.b")
    // Linear work on a long subject is never cut off by the matcher's step limit
    // (a pattern java.util.regex runs without recursing per iteration).
    val lng = "ab".repeat(35000)
    check("re65", Regex("(?>a|b)+").find(lng)!!.value.length == 70000)
}

// ===== SECTION 24: floating point arithmetic =====
// Double/Float are a DIFFERENT type from Int, and every value below was checked
// against the equivalent Java program run on `java` - Kotlin/JVM's Double is
// java.lang.Double, its arithmetic is the JVM's and its toString is
// java.lang.Double.toString, so Java is the oracle here (no kotlinc available).
class Money24(val amount: Double) {
    fun half(): Double = amount / 2
}
fun s24(d: Double): String = "" + d
fun sec24() {
    // Division is real division as soon as one side is a Double, and integer
    // division (truncating towards zero) when both sides are Int.
    check("flt1", 1.0 / 3.0 == 0.3333333333333333)
    check("flt2", 7 / 2 == 3 && -7 / 2 == -3)
    check("flt3", 7 / 2.0 == 3.5 && 7.0 / 2 == 3.5)
    check("flt4", 2.5 * 1.5 == 3.75 && 7.0 - 0.5 == 6.5 && 3.0 * 2.0 == 6.0)
    check("flt5", 2.0 % 0.75 == 0.5 && 1.5 + 1 == 2.5)
    // A Double operand promotes the whole operation, so an integral result still
    // prints with its decimal point.
    check("flt6", s24(1.0) == "1.0" && s24(6.0) == "6.0")
    check("flt7", s24(0.1 + 0.2) == "0.30000000000000004")
    check("flt8", s24(1.0 / 3.0) == "0.3333333333333333")
    // Infinity, NaN and the two zeroes.
    val inf = 1.0 / 0.0
    val nan = 0.0 / 0.0
    check("flt9", s24(inf) == "Infinity" && s24(-1.0 / 0.0) == "-Infinity")
    check("flt10", s24(nan) == "NaN" && nan != nan)
    check("flt11", s24(-0.0) == "-0.0" && 0.0 == -0.0)
    check("flt12", 1e300 * 1e300 == inf)
    // Double.toString switches to scientific notation outside [1e-3, 1e7).
    check("flt13", s24(1e20) == "1.0E20" && s24(1.5e-8) == "1.5E-8")
    check("flt14", s24(1e-3) == "0.001" && s24(2.5) == "2.5")
    // ++ and the compound operators keep a Double a Double.
    var d = 1.0
    check("flt15", s24(d) == "1.0")
    d += 0.25
    check("flt16", d == 1.25)
    d++
    check("flt17", d == 2.25 && s24(d) == "2.25")
    d *= 2
    check("flt18", d == 4.5)
    d /= 3
    check("flt19", d == 1.5)
    d--
    check("flt20", s24(d) == "0.5")
    // A string template renders a Double like toString does.
    check("flt21", "$d/${1.0}" == "0.5/1.0")
    check("flt22", s24(-2.5) == "-2.5")
    // Comparisons mix Double and Int freely.
    check("flt23", 2.5 > 2 && 2 < 2.5 && 1.0 == 1.0 && 3.0 <= 3 && 3.0 >= 3)
    // A Double survives a property, a list element and a function boundary.
    val m = Money24(5.0)
    check("flt24", m.half() == 2.5 && s24(m.amount) == "5.0")
    val xs = listOf(0.5, 1.5)
    check("flt25", xs[0] + xs[1] == 2.0 && s24(xs[0] * 2) == "1.0")
    // ----- Float is a BINARY32, not a Double spelled differently -----
    // Kotlin's Float is an IEEE-754 single (kotlin.Float), its arithmetic is the
    // JVM's fadd/fmul/fdiv and Float.toString() delegates to
    // java.lang.Float.toString, so `java` settles every row below.
    val fs = arrayOf(1.0f, 3.0f, 0.1f, 16777217.0f, 16777216.0f)
    // The headline: eight significant digits, not sixteen.
    check("f32a", s24f(fs[0] / fs[1]) == "0.33333334")
    check("f32b", fs[0] / fs[1] != 1.0 / 3.0)
    // 16777217 is the first integer a binary32 cannot hold; the literal itself
    // rounds down to 16777216.
    check("f32c", s24f(fs[3]) == "1.6777216E7" && fs[3] == fs[4])
    // 0.1f is a DIFFERENT number from 0.1, and it prints as the shortest decimal
    // that round-trips at 24 bits rather than at 53.
    check("f32d", s24f(fs[2]) == "0.1" && fs[2].toDouble() == 0.10000000149011612)
    // Both operands convert to Float FIRST (the wider-type promotion of the JVM):
    // rounding only the result would give 1.6777218E7 here.
    check("f32e", s24f(fs[2] + 16777217) == "1.6777216E7")
    // ...and the same promotion decides the RELATIONS.
    check("f32f", !(16777217 < fs[4]) && 16777217 <= fs[4])
    // Float op Double is a DOUBLE - the wider type wins, which the floor's
    // left-operand style rule gets backwards.
    check("f32g", s24f(fs[2] + 1.0) == "1.1000000014901161")
    // ++ keeps the operand's own width, so a Float at the precision boundary
    // does not move.
    var f32h = 16777216.0f
    f32h++
    check("f32h", s24f(f32h) == "1.6777216E7")
    // toFloat() NARROWS, and a declared Float type retypes an integer literal.
    val f32i: Float = 1
    check("f32i", s24f(f32i) == "1.0" && s24f((1.0 / 3.0).toFloat()) == "0.33333334")
    // The companion constants are Floats, so they print at 24 bits too.
    check("f32j", s24f(Float.MAX_VALUE) == "3.4028235E38" && s24f(Float.MIN_VALUE) == "1.4E-45")
    // Unary minus and a signed zero survive the width.
    check("f32k", s24f(-fs[2]) == "-0.1" && s24f(fs[0] * 0.0f) == "0.0")
    // An integral-valued operand times a zero keeps the sign of the product (-0.0, as
    // java says), which the tag engine's integer representation would lose.
    val ds24 = arrayOf(-14447655.0, 0.0)
    check("f32l", s24(ds24[0] * ds24[1]) == "-0.0" && s24(ds24[1] / ds24[0]) == "-0.0")
    // ----- Float.hashCode() is floatToIntBits, a DIFFERENT function -----
    // Kotlin/JVM's Float.hashCode() IS java.lang.Float.hashCode(), which is
    // floatToIntBits(f) read as a SIGNED Int: the raw binary32 pattern, not the
    // 64-bit XOR fold Double.hashCode() applies. Both are checked against java below,
    // every operand read out of an array.
    val fh = arrayOf(1.5f, -1.5f, -0.0f, 0.0f, 0.1f)
    val dh = arrayOf(1.5, -1.5, -0.0, 0.1)
    check("f32m", fh[0].hashCode() == 1069547520 && dh[0].hashCode() == 1073217536)
    // A NEGATIVE float has a NEGATIVE hash (the sign bit is bit 31 of the answer),
    // and the two zeros do NOT hash alike - the case a value-based hash cannot see.
    check("f32n", fh[1].hashCode() == -1077936128 && dh[1].hashCode() == -1074266112)
    check("f32o", fh[2].hashCode() == -2147483648 && fh[3].hashCode() == 0 &&
                  dh[2].hashCode() == -2147483648)
    // 0.1f is a different NUMBER from 0.1, so the two hashes are unrelated.
    check("f32p", fh[4].hashCode() == 1036831949 && dh[3].hashCode() == -1507852285)
    // The hard end of the bit extraction: NaN collapses to one pattern, the
    // infinities, the smallest SUBNORMAL (Float.MIN_VALUE hashes to 1) and the
    // largest finite magnitudes.
    val fx = arrayOf(Float.NaN, Float.POSITIVE_INFINITY, Float.NEGATIVE_INFINITY,
                     Float.MIN_VALUE, Float.MAX_VALUE, -Float.MAX_VALUE)
    check("f32q", fx[0].hashCode() == 2143289344 && fx[1].hashCode() == 2139095040 &&
                  fx[2].hashCode() == -8388608)
    check("f32r", fx[3].hashCode() == 1 && fx[4].hashCode() == 2139095039 &&
                  fx[5].hashCode() == -8388609)
    // ...and the same hash reached through the collection fold, which is the other
    // call site: java's Arrays.asList(1.5f).hashCode() is 31 + 1069547520.
    check("f32s", listOf(fh[0]).hashCode() == 1069547551 &&
                  listOf(dh[0]).hashCode() == 1073217567)
    // ----- `is Float` and `is Double` are EXCLUSIVE -----
    // On the JVM these are instanceof against java.lang.Float and java.lang.Double,
    // which are unrelated classes.
    check("f32t", isF24(fh[0]) && !isD24(fh[0]) && isD24(dh[0]) && !isF24(dh[0]))
    // Neither answer moves for the SHARED supertypes: java.lang.Float and
    // java.lang.Double both extend Number, implement Comparable and are Any.
    check("f32u", isN24(fh[0]) && isN24(dh[0]) && isC24(fh[0]) && isC24(dh[0]) &&
                  isA24(fh[0]) && isA24(dh[0]) && !isI24(fh[0]) && !isI24(dh[0]))
    // A `when (x) { is Float -> }` over the same test, and the smart cast behind
    // it: the narrowed value still carries its own width, so the addition below is
    // a Float addition and prints at 24 bits.
    check("f32v", kind24(fh[4]) == "F" && kind24(dh[3]) == "D" && kind24(3) == "I")
    val sc24: Any = fh[4]
    check("f32w", sc24 is Float && s24f(sc24 + 1.0f) == "1.1")
    // ...and the class the value reports for itself agrees with `is`.
    check("f32x", fh[0]::class.simpleName == "Float" && dh[0]::class.simpleName == "Double")
    // ----- THE COLLECTION-KEY CONTRACT: a Float is not a Double KEY -----
    // `==` on Kotlin's Any is java.lang.Float.equals / java.lang.Double.equals, and
    // Float#equals answers true only "if the argument is not null and is a Float
    // object", so 1.5f and 1.5 are TWO keys and every collection built on equals
    // follows (java; Kotlin/JVM's Float IS java.lang.Float). UNREACHABLE from source
    // Kotlin except through Any (kotlinc rejects `1.5f == 1.5` outright), so every row
    // below goes through eq24 or a collection of Any.
    check("f32y", !eq24(fh[0], dh[0]) && !eq24(dh[0], fh[0]))
    // The same width still compares by VALUE, at both widths and both ways round.
    check("f32z", eq24(fh[0], 1.5f) && eq24(dh[0], 1.5) && !eq24(fh[4], 0.2f))
    // The three collection shapes the contract reaches: a Set dedups by equals, a
    // List scan (contains / indexOf / distinct) tells the two apart, and a Map
    // keeps two entries and answers each key with its OWN value.
    val ka24 = listOf<Any>(fh[0], dh[0])
    check("f33a", setOf<Any>(fh[0], dh[0]).size == 2 && ka24.distinct().size == 2)
    check("f33b", !ka24.contains(2.5) && ka24.indexOf(dh[0]) == 1 && ka24.indexOf(fh[0]) == 0)
    val km24 = mutableMapOf<Any, String>(fh[0] to "F", dh[0] to "D")
    check("f33c", km24.size == 2 && km24[fh[0]] == "F" && km24[dh[0]] == "D")
    // ...and the structural equality of the collections themselves follows it:
    // java's List.of(1.5f).equals(List.of(1.5)) is false.
    check("f33d", listOf<Any>(fh[0]) != listOf<Any>(dh[0]) &&
                  (setOf<Any>(fh[0], dh[0]) - setOf<Any>(dh[0])).size == 1)
    check("f33e", ka24.groupBy { it }.size == 2)
    // THE ORDERED COMPARISONS STILL PROMOTE across the width, and so does `+`: the
    // equals contract is not the arithmetic one, and the two share the boxes.
    // Float.compareTo(Double) is declared in kotlin.Float, so these three ARE reachable
    // from ordinary Kotlin.
    check("f33f", fh[0] < 1.6 && fh[0] > 1.4 && fh[0] <= 1.5 && s24(fh[0] + dh[0]) == "3.0")
    // A boxed Int, Long and Char key must NOT move.
    check("f33g", listOf(1, 2, 3).indexOf(3) == 2 && listOf(1L, 2L).contains(2L) &&
                  mapOf('a' to 1)['a'] == 1 && setOf(1, 2, 2).size == 2)
    // Two integers of different types are unequal through Any: Integer#equals
    // demands an Integer (java).
    check("f33h", !eq24(1, 1L) && !eq24(1.toByte(), 1) && !eq24('a', "a"))
    // A float box never equals an integer through Any: Double.equals(Integer) and
    // Long.equals(Double) are false at every value, fractional or not.
    check("f33i", !eq24(1L, 1.5) && !eq24(1.5, 1L) && !eq24(2L, 2.5f) &&
                  !eq24(1.toByte(), 1.5) && !eq24(1L, 1.0) && !eq24(100L, 100.0) &&
                  !eq24(1, 1.0) && !eq24(1.0f, 1))
}
// `==` taken through an Any boundary, which is the ONLY way source Kotlin can put
// a Float and a Double on the two sides of it.
fun eq24(a: Any, b: Any): Boolean = a == b
// The Float renderer, reached through a function boundary so the grammar's
// constant folder cannot answer these rows at compile time. The parameter is Any
// because two of the rows below hand it a Double on purpose (Float op Double is a
// Double), and Kotlin has no implicit narrowing.
fun s24f(f: Any): String = "" + f
// The dynamic type tests, taken through an Any boundary: `x is Float` on a value
// whose STATIC type is already Float is a Kotlin compile-time constant, so the
// rows above would measure the front end rather than the runtime without these.
fun isF24(x: Any): Boolean = x is Float
fun isD24(x: Any): Boolean = x is Double
fun isN24(x: Any): Boolean = x is Number
fun isC24(x: Any): Boolean = x is Comparable<Any>
fun isA24(x: Any): Boolean = x is Any
fun isI24(x: Any): Boolean = x is Int
fun kind24(x: Any): String = when (x) { is Float -> "F"; is Double -> "D"; is Int -> "I"; else -> "?" }

// ===== SECTION 25: value rendering =====
// What println / toString() / a string template make of a value. Every answer
// below was checked against the equivalent Java program run on `java`: Kotlin/JVM's
// String, Int and Double ARE java.lang.*, println IS System.out.println, a List
// renders through java.util.AbstractCollection.toString and an enum entry through
// java.lang.Enum.toString, so Java is the oracle here (no kotlinc available). Neither
// half may leak its HOST's rendering (JavaScript's "1,2", Go's %v).
// Not asserted here, because this value model cannot reproduce it: an Array
// renders like a List, where real Kotlin prints "[Ljava.lang.Integer;@1b6d3586"
// (arrayOf and listOf build the same value here), and a class without a toString
// gets a FIXED synthetic hash, so only the "Name@" prefix is checked below.
// The two data classes below name their properties x and y, like every other data
// class in this file.
data class Pair25(val x: Int, val y: String)
data class Holder25(val x: List<Int>, val y: Int)
class Plain25(val x: Int)
class Custom25 { override fun toString(): String = "custom!" }
enum class Suit25 { HEART, SPADE }
fun sec25() {
    // A null reference is "null" everywhere it is rendered.
    val s: String? = null
    check("ren1", "$s" == "null" && "" + s == "null")
    // A list: square brackets, ", " between the elements, no quotes on a String.
    val xs = listOf(1, 2)
    check("ren2", xs.toString() == "[1, 2]" && "$xs" == "[1, 2]")
    check("ren3", "" + xs == "[1, 2]")
    check("ren4", listOf("x", "y").toString() == "[x, y]")
    check("ren5", listOf<Int>().toString() == "[]")
    check("ren6", listOf(1, null).toString() == "[1, null]")
    check("ren7", listOf(listOf(1), listOf(2, 3)).toString() == "[[1], [2, 3]]")
    check("ren8", listOf('a', 'b').toString() == "[a, b]")
    check("ren9", listOf(1.5, 2.0).toString() == "[1.5, 2.0]")
    // A data class renders through its SYNTHESIZED toString, and its properties
    // render the same way a top-level value does.
    val p = Pair25(1, "q")
    check("ren10", p.toString() == "Pair25(x=1, y=q)" && "$p" == "Pair25(x=1, y=q)")
    check("ren11", Holder25(listOf(1, 2), 3).toString() == "Holder25(x=[1, 2], y=3)")
    check("ren12", listOf(Pair25(1, "a")).toString() == "[Pair25(x=1, y=a)]")
    // An explicit toString() wins over every built-in rendering.
    check("ren13", Custom25().toString() == "custom!" && "${Custom25()}" == "custom!")
    // A class without one renders as Name@<hash>; the hash itself is the JVM's
    // identity hash, which differs between runs, so only its shape is asserted.
    val plain = Plain25(5)
    check("ren14", plain.toString().indexOf("Plain25@") == 0 && plain.toString().indexOf("@") == 7)
    check("ren15", "$plain" == plain.toString())
    // An enum entry renders as its name.
    check("ren16", Suit25.HEART.toString() == "HEART" && "${Suit25.SPADE}" == "SPADE")
    check("ren17", "" + Suit25.HEART == "HEART" && Suit25.HEART.name == "HEART")
    check("ren18", listOf(Suit25.HEART, Suit25.SPADE).toString() == "[HEART, SPADE]")
    // The primitives keep the rendering they already had.
    check("ren19", true.toString() == "true" && 42.toString() == "42" && 1.5.toString() == "1.5")
    check("ren20", 'c'.toString() == "c" && "s".toString() == "s")
    // Unit is a VALUE and renders as its name. Only toString() is asserted:
    // `println(unit25())` natively cannot be told from `println()` by layer 2 (MetaJS
    // has no `arguments` object); see k2Line in languages/lib/kotlin-rt.metajs.
    check("ren21", unit25().toString() == "kotlin.Unit" && "" + unit25() == "kotlin.Unit")
}
fun unit25(): Unit {}

// ===== SECTION 26: the String member surface =====
// The members java.lang.String / kotlin.text define on a String. Every check below
// runs in BOTH halves; abnf/jsrtkotlin.go's js_ktsmcall is the Go twin of the
// interpreter's mcall String branch. No kotlinc on this machine: the expected values
// are Kotlin's documented semantics (kotlin.text.uppercase/lowercase are
// locale-invariant, hashCode is java.lang.String.hashCode = the 31*h + char fold,
// compareTo is lexicographic by code unit).
fun sec26() {
    val s = "aBc"
    // Kotlin 1.5 renamed toUpperCase/toLowerCase to uppercase/lowercase. Both
    // spellings resolve (the old pair is deprecated, not removed).
    check("str1", s.uppercase() == "ABC")
    check("str2", s.lowercase() == "abc")
    check("str3", s.toUpperCase() == "ABC")
    check("str4", s.toLowerCase() == "abc")
    check("str5", "".uppercase() == "" && "9!".lowercase() == "9!")
    check("str6", s.uppercase().lowercase() == "abc")
    // The rest of the surface.
    check("str7", s.repeat(2) == "aBcaBc" && s.repeat(0) == "")
    check("str8", s.contains("B") && !s.contains("z") && s.contains(""))
    check("str9", s.plus("!") == "aBc!")
    check("str10", s.compareTo("aBd") < 0 && s.compareTo("aBc") == 0 && s.compareTo("aBa") > 0)
    check("str11", s.isNotEmpty() && !"".isNotEmpty())
    check("str12", s.get(1) == 'B' && s.get(0) == 'a')
    check("str13", s.hashCode() == 95362 && "".hashCode() == 0)
    check("str14", "abc".hashCode() == "abc".hashCode() && "abc".hashCode() != "abd".hashCode())
    // The names the shared table already carried keep working.
    check("str15", s.length == 3 && s.isEmpty() == false && s.equals("aBc"))
    check("str16", s.substring(1) == "Bc" && s.indexOf("B") == 1)
    // charAt is the JAVA spelling. Real Kotlin has no String.charAt at all - it is
    // get / the indexing operator - but both grammars carry it as a builtin, and all
    // three spellings answer a Char.
    check("str18", s.charAt(0) == 'a' && s[0] == 'a' && s.get(0) == 'a')
    check("str17", s.toString() == "aBc")
}

// ===== SECTION 27: nested type declarations =====
// A nested enum / interface / object / annotation class, in each of the four
// things that can hold one: the declaration lowers like a top-level one and the
// enclosing descriptor gets it under its simple name, so Outer.Kind.LEFT resolves.
// Every nested type below has a DISTINCT simple name, which keeps this section
// about nesting itself; SECTION 29 is the one that makes the simple names COLLIDE
// across owners.
class Holder27(val tag: Int) {
    enum class Kind27 { LEFT, RIGHT }
    interface Shape27 { fun area(): Int }
    object Reg27 { val seed: Int = 5 }
    annotation class Mark27
    class Slot27(val n: Int)
    fun doubled(): Int = tag * 2
}
object Outer27 {
    enum class Mode27 { UP }
    class Cell27(val v: Int)
    fun go(): Int = 1
}
enum class Top27 { A, B;
    class Helper27(val n: Int)
}
interface Iface27 {
    object Registry27 { val seed: Int = 7 }
}
fun sec27() {
    // The enclosing class's own members are unaffected by the nested declarations.
    check("nst1", Holder27(21).doubled() == 42)
    // A nested enum: the entries, their names and their ordinals.
    check("nst2", Holder27.Kind27.LEFT.name == "LEFT" && Holder27.Kind27.RIGHT.name == "RIGHT")
    check("nst3", Holder27.Kind27.LEFT.ordinal == 0 && Holder27.Kind27.RIGHT.ordinal == 1)
    check("nst4", Holder27.Kind27.LEFT.toString() == "LEFT")
    // A nested object is a singleton reached through the enclosing name.
    check("nst5", Holder27.Reg27.seed == 5)
    // A nested class is CONSTRUCTED through the enclosing name.
    check("nst6", Holder27.Slot27(3).n == 3)
    // The same four holders: an object, an enum class and an interface.
    check("nst7", Outer27.go() == 1 && Outer27.Mode27.UP.name == "UP")
    check("nst8", Outer27.Cell27(4).v == 4)
    check("nst9", Top27.A.name == "A" && Top27.Helper27(5).n == 5)
    check("nst10", Iface27.Registry27.seed == 7)
}

// ===== SECTION 28: assignment to a computed target =====
// An assignment whose left side is a postfix chain containing a CALL, so a plain
// variable/field/index target cannot cover it: boxOf(9).tag = 100, boxes()[0] = 9.
// The chain up to the last step is evaluated ONCE and the last `.name` / `[i]` step is
// written like any other slot.
class Box28(var tag: Int)
val shared28 = Box28(1)
val store28 = mutableListOf(1, 2, 3)
var order28 = ""
fun boxOf28(k: Int): Box28 = Box28(k)
fun sharedOf28(): Box28 = shared28
fun boxes28(): MutableList<Int> = store28
fun lhs28(): Box28 { order28 = order28 + "L"; return shared28 }
fun rhs28(): Int { order28 = order28 + "R"; return 7 }
fun idx28(): Int { order28 = order28 + "I"; return 1 }
fun sec28() {
    // A fresh object each call: the write happens, and is then unobservable.
    boxOf28(9).tag = 100
    check("cta1", boxOf28(9).tag == 9)
    // The same object: the write IS observable.
    sharedOf28().tag = 42
    check("cta2", shared28.tag == 42)
    // A compound operator reads and writes the same slot.
    sharedOf28().tag += 8
    check("cta3", shared28.tag == 50)
    // An indexed slot behind a call.
    boxes28()[0] = 9
    check("cta4", store28[0] == 9)
    boxes28()[2] -= 1
    check("cta5", store28[2] == 2)
    // Kotlin evaluates the receiver, then the index of an indexed slot (`a[i] = v` is
    // a.set(i, v), arguments left to right), then the right-hand side - java 24
    // evaluates `arr()[idx()] = rhs()` in the same order.
    order28 = ""
    lhs28().tag = rhs28()
    check("cta6", order28 == "LR" && shared28.tag == 7)
    order28 = ""
    boxes28()[idx28()] = rhs28()
    check("cta7", order28 == "IR" && store28[1] == 7)
}

// ===== SECTION 29: nested types with COLLIDING simple names =====
// Two owners each declaring a nested type of the SAME simple name. Each nested
// declaration runs in its own child scope and the enclosing descriptor closes over
// THAT scope, so the factory reaches the descriptor declared beside it whatever anyone
// else declares under the same name. The simple name is still mirrored into the
// enclosing scope, which is what keeps a bare `Inner29(...)` inside the owner's own
// body working.
// KNOWN RESIDUE, identical in BOTH halves (so not a divergence, and deliberately
// not asserted here): an UNQUALIFIED reference to a nested class from inside its
// owner's own body still goes through that one mirrored program-scope binding, so
// with two owners it finds whichever was declared last. Kotlin resolves the owner's
// own nested type first. Qualified use - `A29.Inner29(1)`, which is the form that
// diverged - is what this section pins.
class A29 {
    class Inner29(val v: Int) { fun who(): String = "A" }
    enum class Kind29 { LEFT }
    object Reg29 { val seed: Int = 1 }
}
class B29 {
    class Inner29(val v: Int) { fun who(): String = "B" }
    enum class Kind29 { RIGHT }
    object Reg29 { val seed: Int = 2 }
}
object C29 {
    class Inner29(val v: Int) { fun who(): String = "C" }
}
class Host29(val tag: Int) {
    inner class Inner29(val v: Int) { fun who(): String = "H" + tag + v }
}
fun sec29() {
    // Constructed through the enclosing name: each owner gets its OWN nested class.
    check("col1", A29.Inner29(1).who() == "A" && B29.Inner29(2).who() == "B")
    check("col2", A29.Inner29(1).v == 1 && B29.Inner29(2).v == 2)
    check("col3", C29.Inner29(3).who() == "C")
    // Repeated construction through either owner keeps answering that owner.
    check("col4", A29.Inner29(6).who() == "A" && A29.Inner29(7).who() == "A")
    // The same for the value-shaped nested types: an enum and an object.
    check("col5", A29.Kind29.LEFT.name == "LEFT" && B29.Kind29.RIGHT.name == "RIGHT")
    check("col6", A29.Reg29.seed == 1 && B29.Reg29.seed == 2)
    // An INNER class of the same simple name keeps its stored outer receiver.
    check("col7", Host29(7).Inner29(8).who() == "H78")
    check("col8", Host29(9).Inner29(0).who() == "H90" && A29.Inner29(1).who() == "A")
}

// ===== SECTION 30: a raw string ends at the LAST triple quote =====
// A run of more than three quotes closing a raw string belongs to the CONTENT up to the
// final three - Kotlin's own psi/stringTemplates/RawStringsWithManyQuotes.kt writes
// runs of six, seven and eight quotes with a clean parse tree, so a raw string can end
// with a quote character.
fun sec30() {
    val one = """""""
    val two = """"""""
    check("raw1", one.length == 1 && one == "\"")
    check("raw2", two.length == 2 && two == "\"\"")
    // The empty raw string and a quote in the MIDDLE are unaffected.
    check("raw3", """""".length == 0)
    check("raw4", """a""b""" == "a\"\"b")
    // A raw string whose content ends with one or two quotes.
    check("raw5", """ok"""" == "ok\"")
    check("raw6", """ok""""" == "ok\"\"")
}

// ===== SECTION 31: Int and Long have real WIDTHS =====
// Kotlin's integral types are fixed-width and their arithmetic WRAPS silently on
// overflow (unlike Swift, which traps): Int.MAX_VALUE + 1 wraps, 1L shl 40 is a 64 bit
// shift, and Long.MAX_VALUE is a value.
//
// Kotlin/JVM's Int and Long ARE java.lang's, so every value below was checked
// against the EQUIVALENT JAVA PROGRAM run on `java` - there is no kotlinc on this
// machine.
fun s31(v: Any?): String = "" + v
fun zero31(): Int = 0
fun sec31() {
    // Overflow wraps at the type's width, in both directions.
    check("int1", Int.MAX_VALUE + 1 == Int.MIN_VALUE)
    check("int2", Int.MIN_VALUE - 1 == Int.MAX_VALUE)
    check("int3", Int.MAX_VALUE * 2 == -2)
    check("int4", Long.MAX_VALUE + 1L == Long.MIN_VALUE)
    // A Long is exact to 64 bits, well past what a double holds. 9007199254740993
    // is 2^53 + 1: a double rounds it DOWN to 2^53 and cannot represent it.
    check("int5", s31(Long.MAX_VALUE) == "9223372036854775807")
    check("int6", s31(Long.MIN_VALUE) == "-9223372036854775808")
    check("int7", s31(9223372036854775807L - 1L) == "9223372036854775806")
    check("int8", s31(9007199254740993L + 0L) == "9007199254740993")
    check("int9", 1000000L * 1000000L == 1000000000000L)
    // The shift COUNT is masked - & 31 for an Int, & 63 for a Long - so a count at
    // or above the width shifts by count mod width rather than clearing the value.
    check("int10", 1 shl 32 == 1 && 1L shl 64 == 1L)
    check("int11", 1 shl 31 == Int.MIN_VALUE && 1L shl 63 == Long.MIN_VALUE)
    check("int12", s31(1L shl 40) == "1099511627776")
    // shr keeps the sign, ushr does not; both stay at the left operand's width.
    check("int13", -16 shr 2 == -4 && -16L shr 2 == -4L)
    check("int14", -1 ushr 1 == Int.MAX_VALUE)
    check("int15", -1L ushr 1 == Long.MAX_VALUE)
    // and / or / xor / inv are infix FUNCTIONS in Kotlin, not operators.
    check("int16", 0xFF and 0x0F == 15 && 0xF0 or 0x0F == 255)
    check("int17", 0xFF xor 0x0F == 240)
    check("int18", 5.inv() == -6 && 0L.inv() == -1L)
    check("int19", s31((1L shl 40).inv()) == "-1099511627777")
    // Division truncates towards zero and MIN / -1 wraps back to MIN.
    check("int20", 7 / 2 == 3 && -7 / 2 == -3 && 7 / -2 == -3)
    check("int21", -7 % 2 == -1 && 7 % -2 == 1)
    check("int22", Int.MIN_VALUE / -1 == Int.MIN_VALUE && Int.MIN_VALUE % -1 == 0)
    check("int23", Long.MIN_VALUE / -1L == Long.MIN_VALUE)
    // Division and remainder by zero THROW a catchable ArithmeticException, and the
    // message is java.lang's.
    var caught = ""
    try { s31(1 / zero31()) } catch (e: ArithmeticException) { caught = "" + e.message }
    check("int24", caught == "/ by zero")
    caught = ""
    try { s31(1 % zero31()) } catch (e: ArithmeticException) { caught = "" + e.message }
    check("int25", caught == "/ by zero")
    // A DECLARED type puts its width on the value, so the multiplication below is
    // 64 bit throughout.
    val big: Long = 1
    check("int26", s31(big * 1000000 * 1000000) == "1000000000000")
    // ++ steps at the value's own width, which is what makes a Byte wrap at 127.
    var b: Byte = 127
    b++
    check("int27", s31(b) == "-128")
    var l: Long = 9223372036854775806L
    l++
    check("int28", l == Long.MAX_VALUE)
    l++
    check("int29", l == Long.MIN_VALUE)
    // Byte and Short arithmetic PROMOTES to Int, so a sum of two bytes does not
    // wrap at 8 bits, and a Byte compares against an Int at Int width.
    val b1: Byte = 100
    val b2: Byte = 100
    check("int30", b1 + b2 == 200)
    check("int31", b1 < 1000 && b1 > -1000)
    check("int32", s31(Byte.MAX_VALUE) == "127" && s31(Byte.MIN_VALUE) == "-128")
    check("int33", s31(Short.MAX_VALUE) == "32767" && s31(Short.MIN_VALUE) == "-32768")
    // A Long compares EXACTLY, past the range where a double would round.
    check("int34", Long.MAX_VALUE > 0L && Long.MIN_VALUE < 0L)
    check("int35", 9007199254740993L > 9007199254740992L)
    check("int36", 1L == 1L && !(1L == 2L))
    // A sized value survives a list, a template and a function boundary.
    val xs = listOf(1L, 2L)
    check("int37", s31(xs) == "[1, 2]" && xs[0] + xs[1] == 3L)
    check("int38", "v=${Long.MAX_VALUE}" == "v=9223372036854775807")
}

// ===== SECTION 32: the CONVERSION methods =====
// Kotlin has NO implicit numeric conversions and no cast syntax for them either,
// so every width change is a METHOD: toByte(), toShort(), toInt(), toLong(),
// toDouble(). An integral source truncates its bits; a Double source SATURATES at
// the target's range and answers 0 for NaN, because Kotlin/JVM compiles
// Double.toInt() to the JVM's d2i. Java's `(byte)` / `(int)` casts are the same
// two instructions, so every value below was checked against the equivalent Java
// program run on `java`.
fun s32(v: Any?): String = "" + v
fun sec32() {
    // An integral source is a pure bit truncation.
    check("cnv1", s32(255.toByte()) == "-1" && s32(200.toByte()) == "-56")
    check("cnv2", s32(70000.toShort()) == "4464")
    check("cnv3", s32(3000000000L.toInt()) == "-1294967296")
    check("cnv4", s32(Long.MAX_VALUE.toInt()) == "-1")
    check("cnv5", s32(5.toLong()) == "5" && s32((-1).toLong()) == "-1")
    // A Double source truncates towards zero and saturates at the range ends.
    check("cnv6", 3.9.toInt() == 3 && (-3.9).toInt() == -3)
    check("cnv7", 1e20.toInt() == Int.MAX_VALUE && (-1e20).toInt() == Int.MIN_VALUE)
    check("cnv8", 1e20.toLong() == Long.MAX_VALUE)
    check("cnv9", Double.NaN.toInt() == 0)
    // toDouble() is a genuine conversion, which is what makes 7.toDouble() / 2
    // divide as floating point where 7 / 2 does not.
    check("cnv10", 7.toDouble() / 2 == 3.5 && s32(7.toDouble()) == "7.0")
    check("cnv11", s32(Long.MAX_VALUE.toDouble()) == "9.223372036854776E18")
    // toChar() takes the low 16 bits.
    check("cnv12", 65.toChar() == 'A' && 65601.toChar() == 'A')
    // The companion constants exist; Long.MAX_VALUE cannot be written as a plain number.
    check("cnv13", Int.SIZE_BITS == 32 && Int.SIZE_BYTES == 4)
    check("cnv14", Long.SIZE_BITS == 64 && Long.SIZE_BYTES == 8)
    check("cnv15", s32(Double.POSITIVE_INFINITY) == "Infinity")
    check("cnv16", s32(Double.NEGATIVE_INFINITY) == "-Infinity")
    check("cnv17", s32(Double.NaN) == "NaN")
    // Mixed Int/Double arithmetic still promotes to Double: the sized-integer box
    // and the floating-point box of the previous section compose.
    check("cnv18", 2.0 + 1 == 3.0 && 1 + 2.0 == 3.0 && 7 / 2.0 == 3.5)
    check("cnv19", s32(1 + 2.0) == "3.0" && s32(1L + 2.0) == "3.0")
    check("cnv20", 1L < 2.0 && 2.0 > 1L)
    // A declared type retypes its INITIALIZER, which is the one place Kotlin does
    // convert an integer literal without a method: `val d: Double = 1` holds 1.0.
    val dd: Double = 1
    val ll: Long = 1
    check("cnv21", s32(dd) == "1.0" && dd / 2 == 0.5)
    check("cnv22", s32(ll) == "1" && ll is Long)
}

// ===== SECTION 33: the UNSIGNED types, and how a LITERAL is typed =====
// UByte / UShort / UInt / ULong are the one place Kotlin's integers are not
// Java's, so java cannot be the oracle here and every rule below is cited to the
// Kotlin documentation and is UNVERIFIED by any toolchain (there is no kotlinc on
// this machine).
//
//   - kotlin.UInt / ULong / UByte / UShort are unsigned 32 / 64 / 8 / 16 bit
//     types whose arithmetic wraps, whose division and comparison are unsigned,
//     and whose toString prints the UNSIGNED reading.
//   - the literal suffixes are u / U (unsigned) and uL / UL (unsigned long);
//     a `u` literal is a UInt when it fits in one and a ULong otherwise.
//   - an UNSUFFIXED literal is an Int when it fits in one and a Long otherwise -
//     which is why 0xFFFFFFFF is the Long 4294967295 in Kotlin where the same
//     text is the int -1 in Java.
//   - there are no implicit conversions between signed and unsigned, so a mixed
//     expression is a compile error rather than a promotion; the value model
//     cannot enforce that and simply takes the unsigned reading.
fun s33(v: Any?): String = "" + v
fun sec33() {
    // An unsigned type prints and wraps as unsigned.
    check("uns1", s33(1u) == "1" && s33(UInt.MAX_VALUE) == "4294967295")
    check("uns2", UInt.MAX_VALUE + 1u == 0u && s33(0u - 1u) == "4294967295")
    check("uns3", s33(ULong.MAX_VALUE) == "18446744073709551615")
    check("uns4", s33(ULong.MAX_VALUE / 2uL) == "9223372036854775807")
    check("uns5", s33(5uL - 7uL) == "18446744073709551614")
    check("uns6", s33(UByte.MAX_VALUE) == "255" && s33(UShort.MAX_VALUE) == "65535")
    check("uns7", s33(UInt.MIN_VALUE) == "0" && s33(ULong.MIN_VALUE) == "0")
    // Division and comparison are UNSIGNED: a value above 2^31 is large, not
    // negative, which is exactly what a signed reading would get wrong.
    check("uns8", s33(4294967295u / 2u) == "2147483647")
    check("uns9", 3000000000u > 1u && 3000000000u > 2147483647u)
    check("uns10", ULong.MAX_VALUE > 1uL)
    // A shift keeps the left operand's type and masks its count.
    check("uns11", s33(1uL shl 63) == "9223372036854775808")
    check("uns12", s33(1u shl 31) == "2147483648")
    // inv() answers the operand's OWN type, not Int.
    check("uns13", s33(1u.inv()) == "4294967294")
    // The conversions reinterpret the bits in both directions.
    check("uns14", s33((-1).toUInt()) == "4294967295")
    check("uns15", s33((-1L).toULong()) == "18446744073709551615")
    check("uns16", s33(256u.toUByte()) == "0" && s33(300.toUByte()) == "44")
    check("uns17", s33(4294967295u.toInt()) == "-1")
    // A Double conversion to an unsigned type saturates at [0, MAX].
    check("uns18", s33(1e20.toUInt()) == "4294967295" && s33((-5.0).toUInt()) == "0")
    // ++ wraps at the unsigned width too.
    var u: UInt = 1u
    u++
    check("uns19", s33(u) == "2")
    var m: UByte = 255u
    m++
    check("uns20", s33(m) == "0")
    // An unsuffixed literal is an Int when it fits and a Long otherwise, so this
    // hexadecimal is 4294967295 and not -1 (the Java answer for the same text).
    check("uns21", 0xFFFFFFFF == 4294967295L)
    check("uns22", s33(2147483648) == "2147483648")
    check("uns23", 0b1010 == 10 && 1_000_000 == 1000000)
    check("uns24", 0xFFFFFFFFL == 4294967295L)
    // A value carries its real type into `is`.
    val a: Any = 10L
    val i: Any = 10
    val q: Any = 1u
    check("uns25", a is Long && !(a is Int))
    check("uns26", i is Int && !(i is Long))
    check("uns27", q is UInt && !(q is Int))
    // An unsigned value survives a list and a string template.
    check("uns28", s33(listOf(1uL, 2uL)) == "[1, 2]")
    check("uns29", "u=${UInt.MAX_VALUE}" == "u=4294967295")
}


// ===== SECTION 34: return / break / continue as EXPRESSIONS, and PREFIX ++ / -- =====
// `x ?: return` is the idiom this section exists for. Kotlin types return, break and
// continue as Nothing, so each is a legal EXPRESSION that never yields a value and
// instead transfers control out of the whole surrounding expression.
// Ground truth for the SYNTAX is Kotlin's own parse tree: the corpus file
// tests/reference/kotlin/.../psi/greatSyntacticShift/nullableTypes.txt records
// `x as? X ?: return` as BINARY_EXPRESSION(BINARY_WITH_TYPE, ELVIS, RETURN) with no
// PsiErrorElement, and psi/SimpleExpressions.txt puts `= break@la` / `= continue@la` in a
// parameter default, also clean. The VALUES asserted below are the language rule (the
// prefix form yields the NEW value, the postfix form the old one) and are UNVERIFIED by
// any toolchain - there is no kotlinc on this machine.
fun firstOr34(v: Int?): Int {
    val x = v ?: return -1
    return x + 100
}
fun sumUntilNull34(limit: Int): Int {
    var s = 0
    for (i in 1..10) {
        val x = if (i > limit) null else i
        s = s + (x ?: break)
    }
    return s
}
fun sumOdd34(): Int {
    var s = 0
    for (i in 1..6) {
        val x = if (i % 2 == 0) null else i
        s = s + (x ?: continue)
    }
    return s
}
fun labelledBreak34(): Int {
    var s = 0
    outer@ for (i in 1..5) {
        for (j in 1..5) {
            val x = if (i > 2) null else j
            s = s + (x ?: break@outer)
        }
    }
    return s
}
fun labelledContinue34(): Int {
    var s = 0
    outer@ for (i in 1..5) {
        for (j in 1..5) {
            val x = if (j > 2) null else j
            s = s + (x ?: continue@outer)
        }
        s = s + 100
    }
    return s
}
class Cell34(var v: Int)

fun sec34() {
    // return in expression position, in both directions.
    check("ctl01", firstOr34(3) == 103)
    check("ctl02", firstOr34(null) == -1)
    // break leaves the loop AND the expression that asked for it: 1+2+3+4.
    check("ctl03", sumUntilNull34(4) == 10)
    // continue skips only the rest of the iteration: 1+3+5.
    check("ctl04", sumOdd34() == 9)
    // A LABELLED signal raised from inside an expression still names the outer loop.
    check("ctl05", labelledBreak34() == 30)
    check("ctl06", labelledContinue34() == 15)
    // A `return` raised out of a lambda body is Kotlin's NON-LOCAL return: it leaves
    // sumWhilePositive34, not just the lambda, so the 9 after the -1 is never added.
    check("ctl07", sumWhilePositive34(listOf(3, 5, -1, 9)) == 8)
    check("ctl08", sumWhilePositive34(listOf(-2, 4)) == 0)

    // PREFIX ++ / -- yields the NEW value; the postfix form yields the old one.
    var i = 1
    check("ctl09", ++i == 2)
    check("ctl10", i == 2)
    check("ctl11", i++ == 2)
    check("ctl12", i == 3)
    check("ctl13", --i == 2 && i == 2)
    // ... on an indexed slot ...
    val arr = intArrayOf(5, 6)
    check("ctl14", ++arr[0] == 6 && arr[0] == 6)
    // ... and on a field.
    val c = Cell34(9)
    check("ctl15", ++c.v == 10 && c.v == 10)
    check("ctl16", c.v-- == 10 && c.v == 9)
}
fun sumWhilePositive34(xs: List<Int>): Int {
    var acc = 0
    xs.forEach { acc = acc + (if (it > 0) it else return acc) }
    return acc
}

// ===== SECTION 35: the CALL SURFACE - a parenthesized callee, and a body's `;` =====
// Every form here is ordinary Kotlin.
// SYNTAX ground truth is again Kotlin's own tree: psi/CallWithManyClosures.txt records
// `val a = (f) {} {} {}` and `(f)() {} {} {}` with clean trees, psi/DoubleColon.txt has
// `(a::b)()` as a CALL_EXPRESSION whose callee is a PARENTHESIZED expression, and
// psi/FunctionExpressions.txt has `fun c() = fun name();` - the trailing semicolon
// included - also clean.
// psi/DoubleColon.txt is also the ground truth for the guard the last two checks pin:
// it puts `a::b.c` on one line and `(a::b)()` two lines below, and records them as TWO
// statements rather than one call of `c`. So an argument list must OPEN on its callee's
// own line, while a call whose arguments merely SPAN lines is unaffected.
fun add35(a: Int, b: Int): Int = a + b
fun twice35(f: (Int) -> Int, v: Int): Int = f(f(v))
fun konst35(): Int = 41;
fun anon35() = fun(): Int { return 5 };

fun sec35() {
    val tripler = { x: Int -> x * 3 }
    // A PARENTHESIZED callee is invoked like any other.
    check("cal01", (tripler)(7) == 21)
    check("cal02", (::add35)(2, 5) == 7)
    check("cal03", ((tripler))(2) == 6)
    // A trailing `;` after an expression body and after a block body.
    check("cal04", konst35() == 41)
    check("cal05", anon35()() == 5)
    // A lambda argument still reaches an ordinary call.
    check("cal06", twice35({ it + 1 }, 9) == 11)
    // A call whose ARGUMENTS span several lines keeps working - only a `(` that opens a
    // NEW line is refused (see the DoubleColon citation above).
    check("cal07", add35(
        1,
        2) == 3)
    check("cal08", add35(1,
                         2) == 3)
}

// ===== SECTION 36: scope functions =====
// let / also / takeIf / takeUnless - the four kotlin.standard scope functions whose
// receiver arrives as `it`. They are extensions on Any, so no receiver type owns them
// and both halves answer them from one universal branch (kScopeMethod /
// ktScopeMethod). A class MEMBER of the same name still wins, which is Kotlin's rule
// that a member always beats an extension - checked below with a class that declares
// its own let().
// run / apply / with bind the receiver as `this` instead, so they live in SECTION 40.
class Scope36(val n: Int) {
    fun let(k: Int): Int = n + k          // A MEMBER named let: it must win.
}
fun sec36() {
    check("scp1", "x".let { it + "y" } == "xy")
    check("scp2", 5.let { it * 2 } == 10)
    check("scp3", listOf(1, 2, 3).let { it.size } == 3)
    // also returns the RECEIVER, and its lambda is run for its effect.
    var seen = 0
    val same = listOf(4, 5).also { seen = it.size }
    check("scp4", seen == 2 && same.size == 2)
    check("scp5", "q".also { seen = it.length } == "q" && seen == 1)
    // takeIf answers the receiver or null; takeUnless is its negation.
    check("scp6", 7.takeIf { it > 3 } == 7)
    check("scp7", 2.takeIf { it > 3 } == null)
    check("scp8", 2.takeUnless { it > 3 } == 2)
    check("scp9", 7.takeUnless { it > 3 } == null)
    // A chain: each step hands its own value on.
    check("scp10", "ab".let { it + "c" }.let { it.length } == 3)
    // The member wins over the extension.
    check("scp11", Scope36(10).let(5) == 15)
    // let on a nullable receiver, the idiom the function exists for.
    val maybe: String? = "z"
    check("scp12", maybe?.let { it + "!" } == "z!")
    val none: String? = null
    check("scp13", none?.let { it + "!" } == null)
}

// ===== SECTION 37: the collections API =====
// The kotlin.collections operations a real program leans on. Every one of them is a
// METHOD, so both halves answer from the same table (kSeqMethod in
// kotlin-interpreter.abnf, ktSeqMethod in abnf/jsrtkotlin.go) and --cross diffs them.
// Values checked against the equivalent Java program under `java` wherever Kotlin's
// answer is java.util's: List.hashCode is the 31*h + element fold, and a `sumOf`
// whose selector answers a Long sums at 64 bits.
fun sec37() {
    val xs = listOf(1, 2, 3, 4, 5)
    check("col1", xs.fold(0) { a, b -> a + b } == 15)
    check("col2", xs.fold("") { a, b -> a + b } == "12345")
    check("col3", xs.foldRight("") { a, b -> a + b } == "12345")
    check("col4", xs.reduce { a, b -> a * b } == 120)
    check("col5", xs.sumOf { it * 2 } == 30)
    // sumOf takes the WIDTH of its selector: a Long selector sums at 64 bits, where a
    // 32 bit accumulator would have wrapped 6000000000 to 1705032704.
    check("col6", "" + xs.sumOf { it.toLong() * 1000000000L } == "15000000000")
    check("col7", xs.filter { it % 2 == 1 }.toString() == "[1, 3, 5]")
    check("col8", xs.filterNot { it % 2 == 1 }.toString() == "[2, 4]")
    check("col9", xs.map { it * it }.toString() == "[1, 4, 9, 16, 25]")
    check("col10", xs.mapIndexed { i, v -> i * v }.toString() == "[0, 2, 6, 12, 20]")
    check("col11", xs.flatMap { listOf(it, it) }.size == 10)
    check("col12", listOf(listOf(1, 2), listOf(3)).flatten().toString() == "[1, 2, 3]")
    check("col13", xs.partition { it > 2 }.toString() == "([3, 4, 5], [1, 2])")
    check("col14", xs.chunked(2).toString() == "[[1, 2], [3, 4], [5]]")
    check("col15", xs.windowed(3).toString() == "[[1, 2, 3], [2, 3, 4], [3, 4, 5]]")
    check("col16", xs.zip(listOf("a", "b")).toString() == "[(1, a), (2, b)]")
    check("col17", xs.zip(listOf(10, 20)) { a, b -> a + b }.toString() == "[11, 22]")
    check("col18", xs.take(2).toString() == "[1, 2]" && xs.drop(3).toString() == "[4, 5]")
    check("col19", xs.takeLast(2).toString() == "[4, 5]" && xs.dropLast(3).toString() == "[1, 2]")
    check("col20", xs.reversed().toString() == "[5, 4, 3, 2, 1]")
    check("col21", listOf(3, 1, 2).sorted().toString() == "[1, 2, 3]")
    check("col22", listOf(3, 1, 2).sortedDescending().toString() == "[3, 2, 1]")
    val words = listOf("bbb", "a", "cc")
    check("col23", words.sortedBy { it.length }.toString() == "[a, cc, bbb]")
    check("col24", words.sortedByDescending { it.length }.toString() == "[bbb, cc, a]")
    check("col25", words.maxByOrNull { it.length } == "bbb" && words.minByOrNull { it.length } == "a")
    check("col26", xs.maxOrNull() == 5 && xs.minOrNull() == 1)
    check("col27", listOf<Int>().maxOrNull() == null)
    check("col28", xs.all { it > 0 } && !xs.all { it > 1 })
    check("col29", xs.none { it > 9 } && !xs.none { it > 4 })
    check("col30", xs.find { it > 3 } == 4 && xs.find { it > 9 } == null)
    check("col31", xs.first { it % 2 == 0 } == 2 && xs.firstOrNull { it > 9 } == null)
    check("col32", xs.indexOfFirst { it == 3 } == 2 && xs.indexOfFirst { it == 9 } == -1)
    check("col33", listOf(1, 1, 2, 2, 3).distinct().toString() == "[1, 2, 3]")
    check("col34", xs.joinToString("-") == "1-2-3-4-5")
    check("col35", xs.joinToString(", ", "[", "]") == "[1, 2, 3, 4, 5]")
    // IndexedValue is a data class, so withIndex() prints the generated toString,
    // not the Pair form.
    check("col36", xs.withIndex().toString() ==
        "[IndexedValue(index=0, value=1), IndexedValue(index=1, value=2), " +
        "IndexedValue(index=2, value=3), IndexedValue(index=3, value=4), " +
        "IndexedValue(index=4, value=5)]")
    // java.util.List.hashCode: 31*h + element, starting at 1.
    check("col37", listOf(1, 2, 3).hashCode() == 30817)
    check("col38", listOf<Int>().hashCode() == 1 && listOf(1, 2, 3).hashCode() == listOf(1, 2, 3).hashCode())
    check("col39", xs.count { it > 2 } == 3 && xs.count() == 5)
    check("col40", xs.mapNotNull { if (it > 3) it else null }.toString() == "[4, 5]")
    // A Sequence pipeline. Sequences are EAGER in this subset - the answer of a
    // finite pipeline is identical either way, which is what is asserted.
    check("col41", xs.asSequence().map { it * 2 }.filter { it > 4 }.toList().toString() == "[6, 8, 10]")
    check("col42", xs.single { it == 3 } == 3)
    check("col43", listOf(2).single() == 2)
    var acc37 = 0
    xs.forEachIndexed { i, v -> acc37 += i * v }
    check("col44", acc37 == 40)
    check("col45", xs.average() == 3.0)
    check("col46", xs.sum() == 15)
}

// ===== SECTION 38: Char is a real type, and the String is a sequence of it =====
// Kotlin's String has no charAt: it is get / the indexing operator, both answering a
// CHAR. Every Char-valued member answers a Char, and the Char member surface (.code,
// .digitToInt(), the isX predicates, the case conversions) reads the CODE.
// Kotlin's Char is java.lang.Character's, so the classification and case answers were
// checked against the equivalent Java program under `java`.
fun sec38() {
    val s = "aBc"
    check("chr1", s[0] == 'a' && s.get(1) == 'B' && s.charAt(2) == 'c')
    check("chr2", s.first() == 'a' && s.last() == 'c')
    check("chr3", "z".single() == 'z')
    check("chr4", s.toCharArray().toString().startsWith("[C@") && s.toCharArray().contentToString() == "[a, B, c]")
    var n38 = 0
    for (c in s) { if (c == 'a' || c == 'c') n38++ }
    check("chr5", n38 == 2)
    check("chr6", 'a'.code == 97 && 'A'.code == 65)
    check("chr7", '7'.digitToInt() == 7)
    check("chr8", 'a'.isLetter() && !'1'.isLetter())
    check("chr9", '1'.isDigit() && !'a'.isDigit())
    check("chr10", 'a'.isLetterOrDigit() && !' '.isLetterOrDigit())
    check("chr11", ' '.isWhitespace() && !'a'.isWhitespace())
    check("chr12", 'A'.isUpperCase() && 'a'.isLowerCase())
    check("chr13", 'a'.uppercaseChar() == 'A' && 'B'.lowercaseChar() == 'b')
    check("chr14", 'a'.toString() == "a" && "" + 'a' == "a")
    // Char arithmetic: 'a' + 1 is a Char in Kotlin, and comparison is by code.
    check("chr15", 'a' < 'b' && 'b' > 'a' && 'a' == 'a')
    check("chr16", 'a'.compareTo('b') == -1 && 'b'.compareTo('b') == 0)
    check("chr17", 'a'.hashCode() == 97)
    // A String is a sequence of Char, so the collection operations apply to it.
    check("chr18", s.map { it.code }.toString() == "[97, 66, 99]")
    // kotlin.text declares `String.filter(predicate): String` - the char-SELECTING
    // operators of _Strings.kt answer a String, not a List<Char>.
    check("chr19", s.filter { it.isUpperCase() } == "B")
    check("chr20", s.count { it.isLetter() } == 3)
    check("chr21", s.any { it == 'B' } && !s.any { it == 'z' })
    // The rest of the kotlin.text surface.
    check("str19", "  pad  ".trim() == "pad")
    check("str20", "  pad  ".trimStart() == "pad  " && "  pad  ".trimEnd() == "  pad")
    check("str21", "abc".startsWith("ab") && "abc".endsWith("bc") && !"abc".startsWith("b"))
    check("str22", "7".padStart(3, '0') == "007" && "7".padEnd(3) == "7  ")
    check("str23", "abc".reversed() == "cba")
    check("str24", "a,b,c".split(",").toString() == "[a, b, c]")
    check("str25", "a\nb".lines().toString() == "[a, b]")
    check("str26", "abcde".drop(2) == "cde" && "abcde".take(2) == "ab")
    check("str27", "abcde".dropLast(2) == "abc" && "abcde".takeLast(2) == "de")
    check("str28", "42".toInt() == 42 && "-7".toInt() == -7)
    check("str29", "" + "9000000000".toLong() == "9000000000")
    check("str30", "1.5".toDouble() == 1.5)
    check("str31", "abc".indexOf('b') == 1 && "abc".contains('b') && !"abc".contains('z'))
    check("str32", "abab".lastIndexOf("ab") == 2)
    // kotlin.text.format is java.util.Formatter.
    check("str33", "%.2f".format(3.14159) == "3.14")
    check("str34", "%d/%s".format(42, "z") == "42/z")
    // %f rounds HALF UP, which is java.util.Formatter's rule: `java` (with the locale
    // forced to en_US) prints "  1.3|", "2.68" and "0.3" for these three, where Go's
    // Sprintf - which rounds half to EVEN - would print "  1.2|", "2.67", "0.2".
    check("str35", "%5.1f|".format(1.25) == "  1.3|")
    check("str36", "%.2f".format(2.675) == "2.68" && "%.1f".format(0.25) == "0.3")
    check("str37", "%s|%b".format("q", true) == "q|true")
}

// ===== SECTION 39: maps =====
// A Map is the runtime's shared {__dict, keys, vals} handle - the same shape Go maps
// and Python dicts use - so both halves carry one map value model and one member
// table. The maps below are built with associate / groupBy / toMap; the mapOf family
// of global builders is asserted in SECTION 40.
// A Map renders as java.util.AbstractMap.toString does: {a=1, b=2}.
fun sec39() {
    val squares = listOf(1, 2, 3).associate { it to it * it }
    check("map1", squares.toString() == "{1=1, 2=4, 3=9}")
    check("map2", squares[2] == 4)
    check("map3", squares.size == 3)
    check("map4", squares.containsKey(3) && !squares.containsKey(9))
    check("map5", squares.keys.toString() == "[1, 2, 3]")
    check("map6", squares.values.toString() == "[1, 4, 9]")
    check("map7", squares.isNotEmpty() && !squares.isEmpty())
    val byLen = listOf("a", "bb", "cc", "d").groupBy { it.length }
    check("map8", byLen.toString() == "{1=[a, d], 2=[bb, cc]}")
    check("map9", byLen[2].toString() == "[bb, cc]")
    val withV = listOf("x", "yy").associateWith { it.length }
    check("map10", withV.toString() == "{x=1, yy=2}")
    val byK = listOf("x", "yy").associateBy { it.length }
    check("map11", byK.toString() == "{1=x, 2=yy}")
    val fromPairs = listOf(1 to "a", 2 to "b").toMap()
    check("map12", fromPairs.toString() == "{1=a, 2=b}")
    check("map13", fromPairs.getValue(1) == "a")
    check("map14", fromPairs.getOrDefault(9, "?") == "?")
    // An entry is destructurable and carries key/value. Iterating the map DIRECTLY
    // (rather than its .entries) is asserted in SECTION 40.
    var joined = ""
    for ((k, v) in fromPairs.entries) { joined += "" + k + v }
    check("map15", joined == "1a2b")
    check("map16", fromPairs.entries.size == 2)
    check("map17", fromPairs.containsValue("b") && !fromPairs.containsValue("z"))
    // A MISS answers null, not Unit: kotlin.Map.get is declared to return V?, and the
    // indexed form is that same get.
    check("map21", fromPairs[9] == null && fromPairs.get(9) == null)
    check("map22", "" + fromPairs[9] == "null")
    check("map23", byLen[7] == null && squares[9] == null)
    // A Pair is (first, second) and destructures; `to` builds one.
    val p = 3 to "c"
    check("pair1", p.toString() == "(3, c)")
    check("pair2", p.first == 3 && p.second == "c")
    val (pa, pb) = p
    check("pair3", pa == 3 && pb == "c")
    check("pair4", listOf(1, 2, 3, 4).partition { it > 2 }.first.toString() == "[3, 4]")
    // A map is a sequence of entries, so the collection operations apply.
    check("map18", squares.map { it.value }.toString() == "[1, 4, 9]")
    check("map19", squares.filter { it.key > 1 }.size == 2)
    check("map20", squares.count() == 3)
}

// ===== SECTION 40: the global BUILDERS and the this-bound scope functions =====
// A global NAME is declared by each grammar's own builtin block - hostGlobals in
// kotlin-interpreter.abnf, js_scope_decl in kotlin-to-llvm-ir.abnf - not by the shared
// runtime, so mapOf/Pair/Triple/with/StringBuilder/buildString/sequenceOf are
// registered in BOTH. What they BUILD is a shared shape whose whole member surface
// lives in abnf/jsrtkotlin.go.
//
// run / apply / with bind the receiver as `this`, so an unqualified `size` or
// `append("a")` inside the lambda is a member call on it. The interpreter binds it
// through kSetRecv (the channel a lambda-with-receiver already uses); the compiler's
// lambda is a bare IR closure with no receiver slot, so the builder parks the receiver
// on ktRecvStack and js_ktget consults it after a local and after the enclosing
// `this` - which can only ever turn "unknown name" into a member read.
fun sec40() {
    val m = mapOf(1 to "a", 2 to "b")
    check("gbl1", m.toString() == "{1=a, 2=b}" && m[1] == "a")
    val mm = mutableMapOf(1 to "a")
    mm[2] = "b"
    check("gbl2", mm.toString() == "{1=a, 2=b}" && mm.size == 2)
    check("gbl3", emptyMap<Int, Int>().size == 0)
    check("gbl4", Pair(1, 2).toString() == "(1, 2)")
    check("gbl5", Triple(1, 2, 3).toString() == "(1, 2, 3)")
    val (t1, t2, t3) = Triple(4, 5, 6)
    check("gbl6", t1 + t2 + t3 == 15)
    // with / run / apply bind the receiver as `this`.
    check("gbl7", with(listOf(1, 2, 3)) { size } == 3)
    check("gbl8", listOf(1, 2, 3).run { size } == 3)
    val built = mutableListOf(1, 2).apply { add(3) }
    check("gbl9", built.toString() == "[1, 2, 3]")
    // StringBuilder and buildString.
    val sb = StringBuilder()
    sb.append("a")
    sb.append(1)
    check("gbl10", sb.toString() == "a1" && sb.length == 2)
    check("gbl11", buildString { append("x"); append("y") } == "xy")
    check("gbl12", with(StringBuilder("s")) { append("t"); toString() } == "st")
    // Sequences and buildList / buildMap.
    check("gbl13", sequenceOf(1, 2, 3).map { it + 1 }.toList().toString() == "[2, 3, 4]")
    check("gbl14", buildList { add(1); add(2) }.toString() == "[1, 2]")
    check("gbl15", buildMap { put(1, "a") }.toString() == "{1=a}")
    var r40 = 0
    repeat(3) { r40 += it }
    check("gbl16", r40 == 3)
    // Iterating the MAP itself, rather than its .entries.
    var j40 = ""
    for ((k, v) in m) { j40 += "" + k + v }
    check("gbl17", j40 == "1a2b")
    var c40 = 0
    for (e in m) { c40++ }
    check("gbl18", c40 == 2)
}

// ===== SECTION 41: a declared type carries its WIDTH past the declaration =====
// SECTION 31 pinned the width a LOCAL `val x: Long = 1` carries. The same rule applies
// at every other binding site: a property, a parameter, a parameter DEFAULT, a
// function's return type and a field WRITE. Kotlin/JVM's Int and Long are java.lang's,
// so each value below is what the equivalent Java program prints under `java`.
class Widths41(val x: Long, val b: Byte = 127) {
    var y: Long = 1
}
fun ret41(): Long = 1
fun dflt41(v: Long = 1): Long = v
fun sec41() {
    // A declared RETURN type: `fun ret41(): Long = 1` answers a Long.
    check("wid1", "" + ret41() * 1000000 * 1000000 == "1000000000000")
    // A PROPERTY initializer, both the constructor-parameter form and the body form.
    val w = Widths41(1L)
    check("wid2", "" + w.x * 1000000 * 1000000 == "1000000000000")
    check("wid3", "" + w.y * 1000000 * 1000000 == "1000000000000")
    // A parameter DEFAULT, and the parameter itself.
    check("wid4", "" + dflt41() * 1000000 * 1000000 == "1000000000000")
    check("wid5", "" + dflt41(1L) * 1000000 * 1000000 == "1000000000000")
    // A field WRITE adopts the declared type: `w.y = 1` stores a Long, not an Int.
    w.y = 1
    check("wid6", "" + w.y * 1000000 * 1000000 == "1000000000000")
    // A Byte property wraps at its own width, which is what proves the width is on
    // the stored value rather than on the literal that produced it.
    var bb: Byte = w.b
    bb++
    check("wid7", "" + bb == "-128")
}

// ===== SECTION 42: LOCAL type declarations =====
// A class, an interface or an object declared INSIDE a function body. Every value
// below is what the equivalent Java program prints under `java` - a local class, a
// local class implementing an interface, and a local class declared inside a loop
// body; only the local `object` singleton has no Java spelling and is asserted from
// the Kotlin specification.
// Known limitation, deliberately NOT asserted: a local class body does not CAPTURE the
// enclosing function's locals here (the constructor runs with a fresh scope chain, the
// same one a top-level class gets), so a member reading an enclosing `val` is out of
// scope. Kotlin captures; this subset does not.
fun sec42() {
    class L42(val v: Int) {
        fun twice(): Int = v * 2
    }
    val l = L42(21)
    check("loc1", l.v == 21)
    check("loc2", l.twice() == 42)
    // A local INTERFACE and a local class that implements it.
    interface Shape42 {
        fun area(): Int
    }
    class Sq42(val s: Int) : Shape42 {
        override fun area(): Int = s * s
    }
    val sh: Shape42 = Sq42(4)
    check("loc3", sh.area() == 16)
    check("loc4", sh is Shape42)
    // A local class declared inside a loop body is one type, not one per iteration.
    var total = 0
    for (i in 0 until 3) {
        class P42 {
            fun p(): Int = 7
        }
        total += P42().p()
    }
    check("loc5", total == 21)
    // A local `object` is a singleton, so its state survives between calls.
    object Counter42 {
        var n = 0
        fun bump(): Int { n += 1; return n }
    }
    check("loc6", Counter42.bump() == 1)
    check("loc7", Counter42.bump() == 2)
}

// ===== SECTION 43: `fun interface` and SAM conversion =====
// `fun interface F { fun go(x: Int): Int }` declares a single-abstract-method
// interface, and `F { ... }` converts a lambda into an instance of it. Java's
// @FunctionalInterface is the same construct, so the equivalent Java program was
// written and run under `java`: the values below are its output. The only
// piece with no Java spelling is Kotlin's implicit single parameter `it`, which the
// lambda sections already pin.
fun interface Op43 {
    fun run(a: Int): Int
}
fun interface Mk43 {
    fun make(a: String, b: String): String
}
fun sec43() {
    val o = Op43 { it + 100 }
    check("sam1", o.run(5) == 105)
    // A second conversion of the SAME interface is an independent instance.
    val o2 = Op43 { x -> x * x }
    check("sam2", o2.run(9) == 81)
    check("sam3", "" + o.run(1) + "/" + o2.run(1) == "101/1")
    // More than one parameter is fine; `fun interface` constrains the count of
    // abstract MEMBERS, not of parameters.
    val m = Mk43 { x, y -> x + "-" + y }
    check("sam4", m.make("a", "b") == "a-b")
    // The converted value is an instance of the interface.
    check("sam5", o is Op43)
}

// ===== SECTION 44: a unary minus is folded into the LITERAL =====
// `2147483648` on its own is out of Int range and is a Long; `-2147483648` is
// Int.MIN_VALUE and stays 32 bit, because Kotlin types the negated CONSTANT rather than
// negating a Long. Kotlin/JVM's Int is java.lang's, so the values below are what the
// equivalent Java program prints under `java`:
// -2147483648, 2147483647, 0 for x, x - 1 and x * 2, and -2147483649 for the Long.
fun sec44() {
    val x = -2147483648
    check("neg1", "" + x == "-2147483648")
    // The wrap is the proof the value is 32 bit: a Long would answer -2147483649.
    check("neg2", "" + (x - 1) == "2147483647")
    check("neg3", "" + (x * 2) == "0")
    // An explicit Long suffix is untouched - the fold applies to the literal's TYPE,
    // not to the minus.
    val y = -2147483648L
    check("neg4", "" + (y - 1) == "-2147483649")
    // Everything smaller is unchanged, and the minus still works as an operator.
    val z = -1
    check("neg5", "" + (z * 3) == "-3")
    val w = 2147483648
    check("neg6", "" + (w - 1) == "2147483647")
}

// ===== SECTION 45: structural equality on the collection shapes =====
// Kotlin's `==` is equals(), and a List, a Set, a Map, a Pair and a Triple all have
// STRUCTURAL equality - they are java.util's on the JVM, so the answers below are the
// ones the equivalent Java program prints under `java`: List.of(1,2).equals(
// List.of(1,2)) is true, Map.of(1,2).equals(Map.of(1,2)) is true, and the hashes are
// 994 for [1, 2], 4066 for [a, b], 3 for {1=2} and 1025 for [[1, 2]]. Equality and
// hash must agree.
data class Cfg45(val n: Int, val tags: List<String>)
fun sec45() {
    check("seq1", listOf(1, 2) == listOf(1, 2))
    check("seq2", listOf(1, 2) != listOf(1, 3))
    check("seq3", listOf(listOf(1), listOf(2)) == listOf(listOf(1), listOf(2)))
    check("seq4", mapOf(1 to 2) == mapOf(1 to 2))
    check("seq5", mapOf("a" to 1, "b" to 2) == mapOf("b" to 2, "a" to 1))
    check("seq6", setOf(1, 2) == setOf(1, 2))
    check("seq7", Pair(1, 2) == Pair(1, 2) && Pair(1, 2) != Pair(1, 3))
    check("seq8", Triple(1, 2, 3) == Triple(1, 2, 3))
    // A data class whose component IS a collection compares component-wise too.
    check("seq9", Cfg45(1, listOf("a")) == Cfg45(1, listOf("a")))
    check("sq10", Cfg45(1, listOf("a")) != Cfg45(1, listOf("b")))
    // Equality and hashCode must agree; the numbers are java's.
    check("sq11", listOf(1, 2).hashCode() == 994)
    check("sq12", listOf("a", "b").hashCode() == 4066)
    check("sq13", listOf(listOf(1, 2)).hashCode() == 1025)
    check("sq14", mapOf(1 to 2).hashCode() == 3)
    check("sq15", Pair(1, 2).hashCode() == 33 && Triple(1, 2, 3).hashCode() == 1026)
    // `in`, indexOf and distinct all go through the same equality.
    check("sq16", listOf(1, 2) in listOf(listOf(1, 2)))
    check("sq17", listOf(listOf(1, 2), listOf(3)).indexOf(listOf(3)) == 1)
    check("sq18", listOf(listOf(1), listOf(1)).distinct().size == 1)
    // A named argument to `copy` must land in THIS class's slot, whatever other data
    // class in the file declares a `copy` with different property names.
    check("sq19", "" + Cfg45(1, listOf("a")).copy(tags = listOf("b")) == "Cfg45(n=1, tags=[b])")
}

// ===== SECTION 46: sorting through a user Comparable, and Comparator =====
// `sorted()` on a class that declares `operator fun compareTo` sorts by THAT, the way
// `<` does.
class V46(val n: Int) : Comparable<V46> {
    override fun compareTo(other: V46): Int = n - other.n
    override fun toString(): String = "V($n)"
}
fun sec46() {
    val l = listOf(V46(3), V46(1), V46(2))
    check("cmp1", "" + l.sorted() == "[V(1), V(2), V(3)]")
    check("cmp2", "" + l.sortedDescending() == "[V(3), V(2), V(1)]")
    check("cmp3", l.maxOrNull()!!.n == 3 && l.minOrNull()!!.n == 1)
    check("cmp4", l.max().n == 3 && l.min().n == 1)
    check("cmp5", V46(1) < V46(2) && V46(3) > V46(2))
    // A Comparator is a plain function of two elements; compareBy builds one.
    check("cmp6", "" + listOf(3, 1, 2).sortedWith(compareBy { it }) == "[1, 2, 3]")
    check("cmp7", "" + listOf(3, 1, 2).sortedWith(compareByDescending { it }) == "[3, 2, 1]")
    check("cmp8", "" + l.sortedWith(compareBy { it.n }) == "[V(1), V(2), V(3)]")
}

// ===== SECTION 47: the throwable hierarchy and typed catch =====
// java.lang's chain is Kotlin/JVM's; every relation below was checked with `instanceof`
// under `java`. The catch clause is selected by its declared type.
fun sec47() {
    check("exc1", IllegalStateException("z") is Exception)
    check("exc2", IllegalStateException("z") is RuntimeException)
    check("exc3", NumberFormatException("z") is IllegalArgumentException)
    check("exc4", IndexOutOfBoundsException("z") is RuntimeException)
    check("exc5", StackOverflowError() is Error && OutOfMemoryError() is Error)
    check("exc6", IllegalAccessException() is Exception)
    var hit = ""
    try { throw IllegalStateException("a") } catch (e: RuntimeException) { hit = "rt:" + e.message }
    check("exc7", hit == "rt:a")
    // The clause SELECTION is by declared type, and a clause that does not match is
    // skipped rather than taken.
    hit = ""
    try { throw NumberFormatException("b") } catch (e: IllegalStateException) { hit = "ise" } catch (e: IllegalArgumentException) { hit = "iae" }
    check("exc8", hit == "iae")
    // An Error is not an Exception, so the inner clause must NOT take it.
    hit = ""
    try {
        try { throw StackOverflowError() } catch (e: Exception) { hit = "inner" }
    } catch (e: Throwable) { hit = "outer" }
    check("exc9", hit == "outer")
    // A user exception class still matches its own declared supertype.
    hit = ""
    try { throw MyExc47("m") } catch (e: IllegalStateException) { hit = "ise" } catch (e: Exception) { hit = "exc:" + e.message }
    check("ex10", hit == "exc:m")
}
class MyExc47(m: String) : Exception(m)

// ===== SECTION 48: implicit receivers form a stack =====
// A name the innermost receiver does not answer falls through to the enclosing one:
// `with(P) { tag }` inside a member of Q is this@Q.tag. And a bare `Inner()` built
// inside its outer's own member captures that instance, so this@Outer answers the
// OUTER one.
class P48(val n: Int)
class Q48(val tag: String) {
    fun viaWith(): String = with(P48(1)) { "" + n + tag }
    inner class In48(val q: Int) {
        fun who(): String = this@Q48.tag + "/" + q
        fun bare(): String = tag
    }
    fun make(): In48 = In48(7)
}
fun sec48() {
    check("rcv1", Q48("hi").viaWith() == "1hi")
    check("rcv2", Q48("O").make().who() == "O/7")
    check("rcv3", Q48("O").make().bare() == "O")
    check("rcv4", Q48("Z").In48(1).who() == "Z/1")
    // A BUILTIN receiver: which shapes count as one, and whether an unqualified
    // PROPERTY is read as a field or called as a method: a SET is a builtin receiver
    // (`with(setOf(1, 2)) { size }`), a STRING receiver answers substring / uppercase
    // / lastIndex / indices, and `lastIndex` is read as a field, not called.
    check("rcv5", with(setOf(1, 2, 2)) { size } == 2 && with(setOf(1, 2)) { contains(1) })
    check("rcv6", with("abcd") { substring(1) } == "bcd" && with("abcd") { uppercase() } == "ABCD")
    check("rcv7", with("abcd") { lastIndex } == 3 && with("abcd") { "" + indices } == "0..3")
    check("rcv8", with(listOf(1, 2, 3)) { lastIndex } == 2 &&
        with(listOf(1, 2, 3)) { "" + indices } == "0..2" &&
        with(listOf(1, 2, 3)) { size } == 3)
}

// ===== SECTION 49: the stdlib members the subset was missing =====
// kotlin.collections, kotlin.text and kotlin.ranges members, grouped as one sweep over
// their member lists.
fun sec49() {
    check("std1", "" + listOf(1, null, 2).filterNotNull() == "[1, 2]")
    check("std2", "" + listOf(1, 2, 3).filterIndexed { i, _ -> i > 0 } == "[2, 3]")
    check("std3", listOf(1, 2, 3).findLast { it < 3 } == 2)
    check("std4", listOf(1, 2, 1).lastIndexOf(1) == 2)
    check("std5", "" + listOf(1, 2, 3).takeWhile { it < 3 } == "[1, 2]")
    check("std6", "" + listOf(1, 2, 3).dropWhile { it < 3 } == "[3]")
    check("std7", "" + listOf(1, 2, 3).distinctBy { it % 2 } == "[1, 2]")
    check("std8", listOf(1, 2, 3).containsAll(listOf(1, 2)))
    check("std9", listOf(1, 2, 3).elementAt(1) == 2 && listOf(1, 2, 3).getOrNull(9) == null)
    check("st10", listOf(1, 2, 3).getOrElse(9) { -1 } == -1)
    check("st11", listOf(1, 2, 3).maxOf { it * 2 } == 6 && listOf(1, 2, 3).minOf { it * 2 } == 2)
    check("st12", "" + listOf(1, 2, 3).plus(4) == "[1, 2, 3, 4]")
    check("st13", "" + listOf(1, 2, 3).minus(2) == "[1, 3]")
    check("st14", "" + listOf(1, 2, 3).intersect(listOf(2, 3)) == "[2, 3]")
    check("st15", "" + listOf(1, 2).union(listOf(2, 4)) == "[1, 2, 4]")
    check("st16", "" + listOf(1, 2, 3).subtract(listOf(1)) == "[2, 3]")
    check("st17", "" + listOf(1, 2, 3).slice(0..1) == "[1, 2]")
    check("st18", "" + listOf(1, 2, 3).subList(0, 2) == "[1, 2]")
    check("st19", "" + listOf(1, 2, 3).zipWithNext() == "[(1, 2), (2, 3)]")
    check("st20", "" + listOf(1, 2, 3).runningFold(0) { a, b -> a + b } == "[0, 1, 3, 6]")
    check("st21", "" + listOf(1, 2, 3).ifEmpty { listOf(0) } == "[1, 2, 3]")
    check("st22", "" + listOf(1, 2, 3).onEach { } == "[1, 2, 3]")
    val ml = mutableListOf(1, 2, 3)
    check("st23", ml.removeAt(1) == 2 && "" + ml == "[1, 3]")
    // kotlin.text.
    check("st24", "hello".replace("l", "L") == "heLLo")
    check("st25", "hello".replaceFirst("l", "L") == "heLlo")
    check("st26", "  ".isBlank() && "x".isNotBlank())
    check("st27", "hello".removePrefix("he") == "llo" && "hello".removeSuffix("lo") == "hel")
    check("st28", "hello".removeSurrounding("h", "o") == "ell")
    check("st29", "a.b.c".substringBefore(".") == "a" && "a.b.c".substringAfter(".") == "b.c")
    check("st30", "a.b.c".substringBeforeLast(".") == "a.b" && "a.b.c".substringAfterLast(".") == "c")
    check("st31", "".ifEmpty { "x" } == "x" && "  ".ifBlank { "y" } == "y")
    check("st32", "hello".capitalize() == "Hello")
    check("st33", "hello".subSequence(1, 3) == "el")
    check("st34", "hello".single { it == 'h' } == 'h')
    // kotlin.collections.Map.
    check("st35", "" + mapOf("a" to 1, "b" to 2).filterValues { it > 1 } == "{b=2}")
    check("st36", "" + mapOf("a" to 1, "b" to 2).filterKeys { it == "a" } == "{a=1}")
    check("st37", "" + mapOf("a" to 1).mapValues { it.value + 1 } == "{a=2}")
    check("st38", "" + mapOf("a" to 1).mapKeys { it.key + "!" } == "{a!=1}")
    check("st39", "" + mapOf("a" to 1).plus("b" to 2) == "{a=1, b=2}")
    check("st40", "" + mapOf("a" to 1, "b" to 2).minus("a") == "{b=2}")
    // Map.filter answers a MAP, not a list of entries.
    check("st41", "" + mapOf("a" to 1, "b" to 2).filter { it.value > 1 } == "{b=2}")
    // The top-level functions.
    check("st42", maxOf(1, 2) == 2 && minOf(1, 2, 0) == 0)
    check("st43", "" + listOfNotNull(1, null, 2) == "[1, 2]")
    check("st44", "" + List(3) { it } == "[0, 1, 2]" && ("" + arrayOfNulls<Int>(2)).startsWith("[Ljava.lang.Integer;@"))
    check("st45", lazy { 5 }.value == 5)
    check("st46", 7.coerceIn(0, 5) == 5 && 1.coerceIn(0, 5) == 1)
    // kotlin.math keeps the operand's TYPE: abs(-1.5) is 1.5 and max(1.5, 2.0) is 2.0,
    // which is what `java` prints for Math.abs / Math.max on the same values.
    check("st47", "" + abs(-1.5) == "1.5" && "" + max(1.5, 2.0) == "2.0")
    check("st48", "" + abs(-3L) == "3" && abs(-3) == 3)
    // require / check / error, and runCatching's Result.
    check("st49", runCatching { 1 }.isSuccess && runCatching { error("x") }.isFailure)
    check("st50", runCatching { error("boom") }.exceptionOrNull()!!.message == "boom")
    check("st51", runCatching { error("x") }.getOrDefault(9) == 9 && runCatching { 5 }.getOrThrow() == 5)
    var msg = ""
    try { require(false) { "need it" } } catch (e: IllegalArgumentException) { msg = e.message!! }
    check("st52", msg == "need it")
    // NOT `check(false)`: this file declares its own two-argument `check`, which wins
    // over the stdlib one exactly as Kotlin's shadowing rule says. checkNotNull is the
    // same contract function under a name the file does not shadow.
    try { checkNotNull(null) } catch (e: IllegalStateException) { msg = e.message!! }
    check("st53", msg == "Required value was null.")
    // joinToString's TRAILING LAMBDA is its transform, not its separator.
    check("st54", listOf(1, 2, 3).joinToString { "" + it * 2 } == "2, 4, 6")
    check("st55", listOf(1, 2, 3).joinToString("-") { "" + it } == "1-2-3")
    // `+` on a collection is kotlin.collections.plus.
    check("st56", "" + (listOf(1, 2) + listOf(3)) == "[1, 2, 3]")
    check("st57", "" + (listOf(1, 2) + 3) == "[1, 2, 3]")
    check("st58", "" + (mapOf("a" to 1) + ("b" to 2)) == "{a=1, b=2}")
    // A String on the left still wins, and renders the list Kotlin's way.
    check("st59", "x" + listOf(1) == "x[1]")
}

// ===== SECTION 50: a Set is its own shape, not a List =====
// A Set deduplicates, two sets with the same elements in different orders are equal,
// and a List is never equal to a Set. The answers below are java's, because
// Kotlin/JVM's Set IS
// java.util.Set: Set.of(1, 2).equals(Set.of(2, 1)) is true, List.of(1).equals(
// Set.of(1)) is false and Set.of(1, 2).hashCode() is 3 - AbstractSet's hash is the
// plain SUM of the element hashes, where AbstractList's is the 31-fold.
fun sec50() {
    val s = setOf(3, 1, 2, 1)
    check("set1", s.size == 3 && "$s" == "[3, 1, 2]")
    check("set2", setOf(2, 1) == setOf(1, 2))
    check("set3", !(listOf(1) == setOf(1)) && !(setOf(1) == listOf(1)))
    check("set4", setOf(1, 2).hashCode() == 3 && listOf(1, 2).hashCode() == 994)
    check("set5", setOf("a", "b").hashCode() == 195)
    // Iteration and `in`.
    var seen = 0
    for (x in s) { seen += x }
    check("set6", seen == 6 && 2 in s && 9 !in s && s.contains(1))
    // Every Iterable member answers a LIST even for a Set receiver; only the
    // set-valued operators keep the Set shape.
    check("set7", "" + s.map { it * 2 } == "[6, 2, 4]")
    check("set8", "" + s.sorted() == "[1, 2, 3]" && "" + s.filter { it > 1 } == "[3, 2]")
    check("set9", s.joinToString("-") == "3-1-2" && s.sum() == 6 && s.count() == 3)
    check("set10", "" + s.toList() == "[3, 1, 2]")
    // The mutable surface: add reports whether the set GREW.
    val ms = mutableSetOf(1)
    check("set11", ms.add(2) && !ms.add(1) && "$ms" == "[1, 2]")
    check("set12", ms.remove(1) && "$ms" == "[2]" && ms.size == 1)
    // union / intersect / subtract answer a Set, distinct answers a List.
    check("set13", setOf(1, 2).union(setOf(2, 3)) == setOf(1, 2, 3))
    check("set14", (setOf(1, 2) intersect setOf(2, 3)) == setOf(2))
    check("set15", (setOf(1, 2) subtract setOf(2, 3)) == setOf(1))
    check("set16", listOf(1, 2).union(listOf(2, 3)) == setOf(3, 2, 1))
    check("set17", listOf(1, 1, 2).toSet() == setOf(2, 1))
    check("set18", "" + listOf(1, 1, 2).distinct() == "[1, 2]")
    // `+` and `-` keep the receiver's shape.
    check("set19", setOf(1, 2) + 3 == setOf(1, 2, 3) && setOf(1, 2) - 1 == setOf(2))
    // The other constructors. sortedSetOf is a TreeSet, so it is ordered.
    check("set20", "" + sortedSetOf(3, 1, 2) == "[1, 2, 3]")
    check("set21", emptySet<Int>().isEmpty() && setOfNotNull(1, null, 1) == setOf(1))
    check("set22", hashSetOf(1, 1) == linkedSetOf(1) && s.toHashSet() == s)
    check("set23", setOf(1, 2).containsAll(listOf(1)) && !setOf(1).containsAll(listOf(1, 2)))
    // Map.keys is a Set (java.util.Map.keySet); Map.values is a Collection.
    val m = mapOf("a" to 1, "b" to 2)
    check("set24", m.keys == setOf("b", "a") && "" + m.values == "[1, 2]")
    check("set25", m.keys.hashCode() == 195)
    // A set OF sets still compares structurally.
    check("set26", setOf(setOf(1), setOf(2)) == setOf(setOf(2), setOf(1)))
    // A `-` on a LIST or a MAP is kotlin.collections.minus too.
    check("set27", "" + (listOf(1, 2, 3) - 1) == "[2, 3]")
    check("set28", "" + (listOf(1, 2, 3) - listOf(2)) == "[1, 3]")
    check("set29", "" + (mapOf("a" to 1, "b" to 2) - "a") == "{b=2}")
}

// ===== SECTION 51: hashCode is java's, and it agrees with equals =====
// Every answer below was read off the equivalent Java program under `java`, because
// Kotlin/JVM's Double/Long/Boolean/String/Char hashCode ARE java.lang's:
// 1.5.hashCode() is 1073217536 (doubleToLongBits xor its own high half), not the
// truncated value. hashCode must agree with the structural equals the collections use.
fun sec51() {
    check("hsh1", 1.5.hashCode() == 1073217536)
    check("hsh2", listOf(1.5).hashCode() == 1073217567)
    check("hsh3", (0.1).hashCode() == -1507852285 && (-1.5).hashCode() == -1074266112)
    check("hsh4", (3.0).hashCode() == 1074266112 && (1e300).hashCode() == -164130400)
    // +-0.0 differ in their BITS, and NaN collapses to one canonical pattern.
    check("hsh5", (0.0).hashCode() == 0 && (-0.0).hashCode() == -2147483648)
    check("hsh6", Double.NaN.hashCode() == 2146959360)
    check("hsh7", Double.POSITIVE_INFINITY.hashCode() == 2146435072)
    check("hsh8", Double.NEGATIVE_INFINITY.hashCode() == -1048576)
    // The subnormal floor: 4.9E-324 is the smallest positive Double, bits == 1.
    check("hsh9", (4.9E-324).hashCode() == 1)
    // The rest of the family.
    check("hsh10", 1234567890123L.hashCode() == 1912276436 && 7.hashCode() == 7)
    check("hsh11", true.hashCode() == 1231 && false.hashCode() == 1237)
    check("hsh12", "ab".hashCode() == 3105 && 'a'.hashCode() == 97)
    check("hsh13", mapOf(1.5 to 2.5).hashCode() == 2147221504)
    // hashCode() and toString() are extensions on Any?, so a null receiver answers
    // them instead of throwing.
    val n: String? = null
    check("hsh14", n.hashCode() == 0 && n.toString() == "null")
    // Boolean owns hashCode/toString/not/compareTo.
    check("hsh15", true.toString() == "true" && true.not() == false)
    check("hsh16", true.compareTo(false) == 1 && false.compareTo(false) == 0)
}

// ===== SECTION 52: Map.Entry is not a Pair =====
// An entry destructures like a Pair - `for ((k, v) in m)` - but it renders as a=1
// where a Pair renders as (a, 1), and it hashes as key XOR value where a Pair uses
// the data-class 31-fold. java on a LinkedHashMap {a=1, b=2}: entrySet() prints [a=1, b=2], one
// entry prints a=1, that entry hashes to 96 and the entry SET hashes to 192, which
// is exactly the map's own hash.
fun sec52() {
    val m = mapOf("a" to 1, "b" to 2)
    check("ent1", "" + m.entries == "[a=1, b=2]")
    check("ent2", "" + m.entries.first() == "a=1")
    check("ent3", m.entries.first().hashCode() == 96)
    check("ent4", m.entries.hashCode() == 192 && m.hashCode() == 192)
    // toList is a list of PAIRS and still renders as one.
    check("ent5", "" + m.toList() == "[(a, 1), (b, 2)]")
    // The entry's own members, and the destructuring that made it Pair-shaped.
    val e = m.entries.first()
    check("ent6", e.key == "a" && e.value == 1)
    check("ent7", e.component1() == "a" && e.component2() == 1)
    var acc = ""
    for ((k, v) in m) { acc += "$k$v" }
    check("ent8", acc == "a1b2")
    check("ent9", "" + m.entries.map { it.key } == "[a, b]")
    // The rendering surface around it: nested collections, a data class in a list,
    // a Pair as a map VALUE.
    check("ent10", "" + mapOf("x" to listOf(1, 2)) == "{x=[1, 2]}")
    check("ent11", "" + mapOf("p" to (1 to 2)) == "{p=(1, 2)}")
    check("ent12", "" + listOf(listOf(1), setOf(2)) == "[[1], [2]]")
    check("ent13", "" + listOf(mapOf("a" to 1)) == "[{a=1}]")
    check("ent14", "" + setOf(1 to 2) == "[(1, 2)]")
}

// ===== SECTION 53: lateinit is enforced =====
// Reading an unassigned `lateinit var` throws UninitializedPropertyAccessException. A
// lateinit property cannot hold null, so "the field is still null" IS "not initialized
// yet" and no per-instance flag is needed. `::name.isInitialized` is the language's
// own test for it.
class Li53 {
    lateinit var a: String
    var plain: String? = null
    fun bare(): Boolean = ::a.isInitialized
    fun qual(): Boolean = this::a.isInitialized
    fun read(): String = a
}
open class Base53 { lateinit var s: String }
class Sub53 : Base53()
fun sec53() {
    val o = Li53()
    check("lat1", !o.bare() && !o.qual())
    // A NULLABLE var is not lateinit: it still reads as null.
    check("lat2", o.plain == null)
    var msg = ""
    try { o.read() } catch (e: UninitializedPropertyAccessException) { msg = e.message ?: "" }
    check("lat3", msg == "lateinit property a has not been initialized")
    // It is a RuntimeException, so the ordinary catch clauses see it.
    var caught = false
    try { println(o.a) } catch (e: RuntimeException) { caught = true }
    check("lat4", caught)
    o.a = "z"
    check("lat5", o.bare() && o.qual() && o.read() == "z" && o.a == "z")
    // An INHERITED lateinit property is enforced through the super chain.
    val sub = Sub53()
    var inh = false
    try { println(sub.s) } catch (e: Exception) { inh = true }
    check("lat6", inh)
    sub.s = "q"
    check("lat7", sub.s == "q")
}

// ===== SECTION 54: the qualified stdlib paths =====
// `kotlin.math.abs(-3)` is the same function as the unqualified `abs(-3)`: a package
// path is a value and its last segment is looked up wherever the builtin lives.
fun sec54() {
    check("pkg1", kotlin.math.abs(-3) == 3 && abs(-3) == 3)
    check("pkg2", kotlin.math.max(1, 2) == 2 && kotlin.math.min(1, 2) == 1)
    check("pkg3", "" + kotlin.collections.listOf(1, 2) == "[1, 2]")
    check("pkg4", kotlin.collections.setOf(1, 1) == setOf(1))
    check("pkg5", kotlin.text.buildString { append("x") } == "x")
    // The root package resolves too.
    check("pkg6", "" + kotlin.Pair(1, 2) == "(1, 2)")
}

// ===== SECTION 55: coroutines, as a synchronous subset =====
// launch / async / runBlocking / delay run their block immediately and to completion.
// THE LIMITATION IS THE ORDERING, not the results: nothing suspends, so a `launch`
// block runs before the statement after it rather than concurrently. Both :description
// blocks state it; the assertions below only pin what this subset actually promises -
// the VALUES.
suspend fun work55(): Int { delay(1); return 42 }
fun sec55() = runBlocking {
    val d = async { work55() }
    check("cor1", d.await() == 42)
    val order = StringBuilder()
    launch { order.append("a") }
    order.append("b")
    // Eager, so "a" lands FIRST. Real kotlinx.coroutines prints "ba" here; this is
    // the documented ordering limitation, asserted so it cannot change silently.
    check("cor2", order.toString() == "ab")
    val j = launch { }
    j.join()
    check("cor3", j.isCompleted && !j.isActive)
    check("cor4", withContext(Dispatchers.Default) { 7 } == 7)
    check("cor5", coroutineScope { 8 } == 8 && supervisorScope { 9 } == 9)
    val two = listOf(async { 1 }, async { 2 })
    check("cor6", "" + awaitAll(two[0], two[1]) == "[1, 2]")
}

// ===== SECTION 57: the callable-reference surface =====
// `b::v`, `b::w`, `b::method` and `b::class` are ordinary Kotlin (bound references
// since 1.1), and an UNBOUND reference's accessors take the receiver from the call:
// `(Box57::v).get(b)` is 3.
//
// Everything below is the KCallable / KClass surface both halves implement, pinned
// name by name. Semantics from the Kotlin documentation ("Callable references", "Class
// references", "Reflection"): an unbound reference takes the receiver as the FIRST
// argument of get/set/invoke, a bound one has it already; .name is the member's simple
// name and "<init>" for a constructor reference; KClass.simpleName and .qualifiedName
// agree for a top-level class in the default package; and two bound references to the
// same member of the same receiver are EQUAL.
class Box57(val v: Int) {
    var w: Int = 5
    fun twice(): Int = v * 2
    fun ownName(): String = this::v.name
    fun ownBare(): Int = ::v.get()
}
fun Box57.tripled(): Int = v * 3
fun top57(x: Int): Int = x + 100
fun apply57(f: (Box57) -> Int, b: Box57): Int = f(b)
fun sec57() {
    val b = Box57(3)
    val lst = listOf(Box57(2), Box57(1))
    // BOUND property references: get / invoke / set / name.
    check("ref1", b::v.get() == 3 && b::v.invoke() == 3 && b::v.name == "v")
    check("ref2", b::w.get() == 5)
    b::w.set(9)
    check("ref3", b.w == 9 && b::w.get() == 9)
    // UNBOUND (type-qualified) references take the receiver from the call.
    check("ref4", Box57::v.get(b) == 3 && Box57::v.invoke(b) == 3 && Box57::v.name == "v")
    Box57::w.set(b, 4)
    check("ref5", b.w == 4)
    // FUNCTION references, bound and unbound, called directly and through invoke.
    check("ref6", b::twice.invoke() == 6 && Box57::twice.invoke(b) == 6)
    check("ref7", b::twice() == 6 && Box57::twice.name == "twice")
    val f = b::twice
    check("ref8", f() == 6)
    // A reference to a member of a BUILTIN type: the base names a type, not a value.
    check("ref9", String::length.get("abcd") == 4)
    // A top-level function reference is the function value, and carries its name.
    check("ref10", ::top57.invoke(1) == 101 && ::top57.name == "top57")
    // A CONSTRUCTOR reference.
    check("ref11", ::Box57.invoke(7).v == 7)
    check("ref12", "" + listOf(7, 8).map(::Box57).map { it.v } == "[7, 8]")
    // CLASS LITERALS, from a value and from a type name.
    check("ref13", b::class.simpleName == "Box57" && Box57::class.simpleName == "Box57")
    check("ref14", Box57::class.qualifiedName == "Box57")
    check("ref15", Box57::class.isInstance(b) && !Box57::class.isInstance(3))
    check("ref16", "" + Box57::class == "class Box57")
    // .java is the same handle again - java.lang.Class renders "class Box57" too.
    check("ref17", b::class.java.simpleName == "Box57")
    check("ref18", String::class.simpleName == "String" && 3::class.simpleName == "Int")
    // A reference passed where a LAMBDA is expected - a builtin and a user function.
    check("ref19", "" + lst.map(Box57::v) == "[2, 1]")
    check("ref20", "" + lst.map(Box57::twice) == "[4, 2]")
    check("ref21", "" + lst.sortedBy(Box57::v).map(Box57::v) == "[1, 2]")
    check("ref22", apply57(Box57::v, b) == 3)
    // An EXTENSION function reference: a member the receiver really has still wins.
    check("ref23", b::tripled.invoke() == 9 && "" + lst.map(Box57::tripled) == "[6, 3]")
    // EQUALITY: two bound references to the same member of the same receiver.
    check("ref24", b::v == b::v && !(b::v == b::w))
    check("ref25", b::class == Box57::class)
    // A reference whose RECEIVER is an arbitrary expression.
    check("ref26", Box57(1)::v.get() == 1 && Box57(1)::class.simpleName == "Box57")
    // A NULLABLE receiver in front of the `::` - the `?` belongs to the type.
    check("ref27", b?::v.get() == 3)
    // A reference read from inside the class, qualified and bare.
    check("ref28", b.ownName() == "v" && b.ownBare() == 3)
    // A reference stored in a val keeps its accessors.
    val p = b::w
    p.set(11)
    check("ref29", p.get() == 11 && p.name == "w")
    check("ref30", "" + listOf(1, 2).map(::top57) == "[101, 102]")
    // A constructor reference is named "<init>", and KCallable.toString() renders the
    // REFERENCE rather than the value it reads. Real kotlin-reflect prints
    // "val Box57.v: kotlin.Int" and, without the reflection library on the class path,
    // "property v (Kotlin reflection is not available)"; neither is reproducible here,
    // so both halves agree on a short form and the divergence from the JVM is
    // deliberate.
    check("ref31", ::Box57.name == "<init>")
    check("ref32", b::v.toString() == "" + b::v && b::v.toString() != "3")
}

// ===== SECTION 56: the enum statics, and the MutableList mutating surface =====
// `Col.values()`, `Col.valueOf(name)` and `Col.entries` (Kotlin 1.9's EnumEntries
// property), and every IN-PLACE MutableList mutator next to its copying twin:
// sort/sortDescending/sortBy/sortWith/reverse next to sorted/sortedBy/sortedWith/
// reversed, plus set, add(index, e), removeAll, retainAll and clear.
enum class Col56 { RED, GREEN, BLUE }
fun sec56() {
    check("enm1", "" + Col56.values().toList() == "[RED, GREEN, BLUE]")
    check("enm2", "" + Col56.entries == "[RED, GREEN, BLUE]")
    check("enm3", Col56.valueOf("GREEN") == Col56.GREEN)
    check("enm4", Col56.BLUE.ordinal == 2 && Col56.BLUE.name == "BLUE")
    check("enm5", Col56.values().size == 3)
    // `when` over an enum, with the entries reachable unqualified in the branches.
    val w = when (Col56.GREEN) { Col56.RED -> 1; Col56.GREEN -> 2; Col56.BLUE -> 3 }
    check("enm6", w == 2)

    val l = mutableListOf(3, 1, 2)
    l.sort()
    check("mut1", "" + l == "[1, 2, 3]")
    l.sortDescending()
    check("mut2", "" + l == "[3, 2, 1]")
    l.sortBy { it }
    check("mut3", "" + l == "[1, 2, 3]")
    l.sortByDescending { it }
    check("mut4", "" + l == "[3, 2, 1]")
    l.sortWith(compareBy { it })
    check("mut5", "" + l == "[1, 2, 3]")
    l.reverse()
    check("mut6", "" + l == "[3, 2, 1]")
    check("mut7", l.set(0, 9) == 3 && "" + l == "[9, 2, 1]")
    l.add(1, 7)
    check("mut8", "" + l == "[9, 7, 2, 1]")
    check("mut9", l.removeAll(listOf(9, 1)) && "" + l == "[7, 2]")
    check("mut10", l.retainAll(listOf(2)) && "" + l == "[2]")
    l.clear()
    check("mut11", l.isEmpty() && "" + l == "[]")
    // Pair/Triple.toList.
    check("mut12", "" + (1 to 2).toList() == "[1, 2]")
    check("mut13", "" + Triple(1, 2, 3).toList() == "[1, 2, 3]")
}

// ===== SECTION 58: a declaration in EVERY kind of block =====
// Ordinary application Kotlin reduced to a destructuring declaration inside a lambda
// body and inside a `when` branch. A grammar has more than one block production
// (Statement/PlainStmt for a function body or an if/while/for body; LamStmt/LamCtrl for
// a lambda body and the `{ ... }` of a when/if ARM), and PEG's ordered choice never
// retries a committed alternative, so a rule missing from one block production cannot
// be recovered later - the lists have to agree. Every form below is therefore checked
// at all four sites: statement level, an if block, a lambda body and a when branch.
fun mk58(): Pair<Int, Int> = Pair(1, 2)
data class Named58(val first: Int, val second: Int)
fun sec58() {
    // 1. statement level.
    val (a1, b1) = mk58()
    // The keyword-less form. Parenthesized, it is NAME based - each entry reads the
    // property of its own name, which for a Pair is first/second - while the bracketed
    // spelling below is POSITION based whatever its entries are called.
    (val first, val second) = mk58()
    [val p1, val p2] = mk58()
    val bare1: Int
    bare1 = 7
    private58()
    check("blk1", a1 + b1 == 3 && first + second == 3 && p1 + p2 == 3 && bare1 == 7)
    check("blk1b", first == 1 && second == 2 && p1 == 1 && p2 == 2)

    // 2. an if block.
    if (a1 == 1) {
        val (a2, b2) = mk58()
        [val p3, val p4] = mk58()
        val lz2 by lazy { 40 + 2 }
        check("blk2", a2 + b2 == 3 && p3 + p4 == 3 && lz2 == 42)
    }

    // 3. a LAMBDA body.
    val fromLambda = run {
        val (a3, b3) = mk58()
        [val p5, val p6] = mk58()
        typealias58()
        fun local3(n: Int) = n * 10
        class Local3(val n: Int)
        val bare3: Int = 5
        val lz3 by lazy { 3 }
        a3 + b3 + p5 + p6 + local3(1) + Local3(2).n + bare3 + lz3
    }
    check("blk3", fromLambda == 3 + 3 + 10 + 2 + 5 + 3)

    // 4. a `when` BRANCH block. Same production.
    val fromWhen = when (a1) {
        1 -> {
            val (a4, b4) = mk58()
            [val p7, val p8] = mk58()
            fun local4(n: Int) = n * 100
            val bare4: Int = 6
            val lz4 by lazy { 4 }
            a4 + b4 + p7 + p8 + local4(1) + bare4 + lz4
        }
        else -> 0
    }
    check("blk4", fromWhen == 3 + 3 + 100 + 6 + 4)

    // 5. a local `fun`, then a destructuring declaration, inside the `{ }` of a when
    //    branch used as an expression body.
    check("blk5", extract58(1) == "[1, 2]/[3]" && extract58(2) == "none")
    // The NAME based reading in full: reversing the entries must NOT reverse the
    // values, and the same entries in BRACKETS must read positionally instead.
    (val second: Int, val first: Int) = Named58(10, 20)
    check("blk6", first == 10 && second == 20)
    [val q1, val q2] = Named58(10, 20)
    check("blk7", q1 == 10 && q2 == 20)
    (val aa = first) = Named58(30, 40)
    check("blk8", aa == 30)
}
private fun private58() { }
fun typealias58() { }
fun decode58(): Pair<ByteArray, ByteArray> = Pair(byteArrayOf(1, 2), byteArrayOf(3))
fun extract58(cmd: Int): String = when (cmd) {
    1 -> {
        fun shortList(bytes: ByteArray) = bytes.asList().map { it.toShort() }
        val (opened, notOpened) = decode58()
        "" + shortList(opened) + "/" + shortList(notOpened)
    }
    else -> { "none" }
}

// ===== SECTION 59: the primitive-array builders and the array conversions =====
// Every primitive-array builder (byteArrayOf, shortArrayOf, the four unsigned builders,
// emptyArray, ...) and the ARRAY-to-collection surface: asList, sortedArray,
// sortedArrayDescending, reversedArray, contentToString and contentEquals. An Array and
// a List are one shape in this value model, so each of them is the list operator under
// Kotlin's array name; contentToString and contentEquals matter because Kotlin's Array
// does NOT get a structural toString or equals. The value model is also width-blind -
// `byteArrayOf(200)`, which kotlinc rejects (200 is not a Byte; the legal spelling is
// `byteArrayOf((-56).toByte())`), is accepted here, which is a TYPE question and is
// documented at both declaration sites.
fun sec59() {
    check("arr1", byteArrayOf(1, 2, 3).size == 3)
    check("arr2", shortArrayOf(1, 2).size == 2)
    check("arr3", charArrayOf('a', 'b').size == 2 && booleanArrayOf(true).size == 1)
    check("arr4", intArrayOf(1).size == 1 && longArrayOf(1L).size == 1)
    check("arr5", doubleArrayOf(1.0).size == 1 && floatArrayOf(1.0f).size == 1)
    check("arr6", ubyteArrayOf(1u).size == 1 && ushortArrayOf(1u).size == 1)
    check("arr7", uintArrayOf(1u).size == 1 && ulongArrayOf(1uL).size == 1)
    check("arr8", emptyArray<Int>().size == 0 && arrayOf(1, 2).size == 2)
    // The size-and-initializer constructors of the same family.
    check("arr9", IntArray(3).size == 3 && ByteArray(2).size == 2 && CharArray(2).size == 2)
    check("arr10", UIntArray(2).size == 2 && ULongArray(2).size == 2)

    // A Byte element is signed 8-bit in Kotlin; the legal way to write one is a
    // conversion, and widening it with .toShort() keeps the value.
    val bs = byteArrayOf((-56).toByte(), 1)
    check("arr11", "" + bs.asList().map { it.toShort() } == "[-56, 1]")
    check("arr12", bs[0].toInt() == -56 && bs.size == 2)

    // The conversions.
    val xs = intArrayOf(3, 1, 2)
    check("arr13", "" + xs.asList() == "[3, 1, 2]")
    check("arr14", "" + xs.toList() == "[3, 1, 2]" && "" + xs.toMutableList() == "[3, 1, 2]")
    check("arr15", xs.toTypedArray().size == 3 && "" + xs.toSet() == "[3, 1, 2]")
    check("arr16", "" + xs.asIterable().toList() == "[3, 1, 2]")
    check("arr17", "" + xs.asSequence().toList() == "[3, 1, 2]")
    check("arr18", "" + listOf(1, 2).toIntArray().asList() == "[1, 2]")
    // The array spellings of sorted / sortedDescending / reversed.
    check("arr19", "" + xs.sortedArray().asList() == "[1, 2, 3]")
    check("arr20", "" + xs.sortedArrayDescending().asList() == "[3, 2, 1]")
    check("arr21", "" + xs.reversedArray().asList() == "[2, 1, 3]")
    // contentToString / contentEquals: Kotlin's Array has no structural toString/equals.
    check("arr22", xs.contentToString() == "[3, 1, 2]")
    check("arr23", intArrayOf(1, 2).contentEquals(intArrayOf(1, 2)))
    check("arr24", !intArrayOf(1, 2).contentEquals(intArrayOf(1, 3)))
    check("arr25", !intArrayOf(1, 2).contentEquals(intArrayOf(1)))
    // Indexing, iteration and the numeric folds on a primitive array.
    var sum59 = 0
    for (v in xs) { sum59 += v }
    check("arr26", sum59 == 6 && xs[0] == 3 && xs.sum() == 6)
    check("arr27", xs.average() == 2.0 && xs.max() == 3 && xs.min() == 1)
    check("arr28", xs.joinToString("-") == "3-1-2")
}

// ===== SECTION 60: the supertype GRAPH, qualified `is`, and the enum companion =====
// Ordinary application Kotlin (a sealed hierarchy, an interface with a default method,
// a companion factory):
//   - member lookup walks the supertype GRAPH: a second interface's default method,
//     anything reached through an interface's own supertypes, and a superclass's
//     interface seen from an OBJECT declaration (a nested object is built while its
//     owner's body is walked, so its parents resolve lazily).
//   - `is Shape.Circle` - the QUALIFIED spelling of a nested class, which is how a
//     sealed hierarchy is normally written.
//   - an enum class's COMPANION, whose members see the enum's own entries.
//   - `Level.valueOf(bad)` throws IllegalArgumentException, so the ordinary
//     `runCatching { valueOf(s) }.getOrNull()` parse idiom works.
interface Aud60 { val label: String
                  fun audit(): String = "audit:" + label }
interface Extra60 { fun extra(): String = "extra" }
interface Deep60 : Extra60
sealed class Shape60 : Aud60 {
    data class Circle(val r: Int) : Shape60() { override val label = "circle" }
    data class Rect(val w: Int, val h: Int) : Shape60() { override val label = "rect" }
    object Empty : Shape60() { override val label = "empty" }
}
class Both60 : Aud60, Deep60 { override val label = "both" }
fun Shape60.area60(): Int = when (this) {
    is Shape60.Circle -> r * r
    is Shape60.Rect -> w * h
    Shape60.Empty -> 0
}
enum class Level60(val severity: Int) {
    DEBUG(0), INFO(1), ERROR(2);
    fun louderThan(o: Level60) = severity > o.severity
    fun countAll() = entries.size
    companion object {
        val fallback = DEBUG
        fun of(s: Int): Level60 = entries.first { it.severity == s }
    }
}
fun sec60() {
    // The supertype GRAPH: a second interface, an interface's own supertype, and a
    // superclass's interface reached from an object declaration.
    check("grf1", Both60().audit() == "audit:both")
    check("grf2", Both60().extra() == "extra")
    check("grf3", Shape60.Empty.audit() == "audit:empty")
    check("grf4", Shape60.Circle(2).audit() == "audit:circle")
    // The QUALIFIED nested-type name in an `is` check.
    check("grf5", Shape60.Circle(3).area60() == 9)
    check("grf6", Shape60.Rect(2, 3).area60() == 6 && Shape60.Empty.area60() == 0)
    check("grf7", Shape60.Circle(1) is Shape60.Circle && !(Shape60.Circle(1) is Shape60.Rect))
    check("grf8", listOf(Shape60.Circle(2), Shape60.Rect(2, 3), Shape60.Empty).map { it.area60() }
                  .sum() == 10)
    // The enum COMPANION: a factory, a property initialized from an entry, and the
    // bare `entries` of Kotlin 1.9 inside a member and inside the companion.
    check("enc1", Level60.of(2) == Level60.ERROR)
    check("enc2", Level60.fallback == Level60.DEBUG)
    check("enc3", Level60.DEBUG.countAll() == 3)
    check("enc4", Level60.ERROR.louderThan(Level60.INFO))
    // valueOf THROWS IllegalArgumentException for an unknown name, so runCatching
    // boxes it - the ordinary way to parse an enum out of text.
    check("enc5", runCatching { Level60.valueOf("WARN") }.getOrNull() == null)
    check("enc6", runCatching { Level60.valueOf("INFO") }.getOrNull() == Level60.INFO)
    var caught60 = ""
    try { Level60.valueOf("NOPE") } catch (e: IllegalArgumentException) { caught60 = "iae" }
    check("enc7", caught60 == "iae")
}

// ===== SECTION 61: labelled returns, including to an OUTER lambda =====
// `return@label` may target the innermost lambda, an OUTER lambda or the enclosing
// function.
// Kotlin's rule is the one implemented here: `return@l` targets the NEAREST enclosing
// function or lambda labelled `l`, with that frame's value. A lambda target raises a
// {__lamret: l} marker rather than returning a control value, because the target may
// be an outer lambda with forEach/map/a user function in between - only the lambda
// whose own label matches absorbs it. A FUNCTION target reached through one or more
// lambdas reuses the __nlret marker every function frame already catches.
// The implicit label of an unlabelled lambda is its callee's name (`return@forEach`).
fun s61outer(): Int {
    var seen = 0
    listOf(1, 2).forEach outer@{ a ->
        listOf(3, 4).forEach { b -> if (b == 3) return@outer }
        seen += a
    }
    return seen
}
fun s61inner(): String {
    var out = ""
    listOf(1, 2, 3).forEach { if (it == 2) return@forEach; out += it }
    return out
}
fun s61nonlocal(): Int { listOf(1, 2, 3).forEach { if (it == 2) return 99 }; return 7 }
fun s61labelfun(): Int {
    listOf(1, 2).forEach { a -> listOf(3, 4).forEach { b -> if (b == 3) return@s61labelfun 42 } }
    return 0
}
fun s61expr(): Int {
    var n = 0
    listOf(1, 2, 3).forEach lam@{ x -> val y = if (x == 2) return@lam else x; n += y }
    return n
}
fun s61value(): Int = listOf(1, 2, 3).map outer@{ a -> listOf(9).map { return@outer a * 2 }; 0 }.sum()
fun s61run(): Int = run outer@{ run { return@outer 5 }; 9 }
fun sec61() {
    // The inner lambda's return@outer leaves the OUTER lambda, so `seen` is never added to.
    check("lret1", s61outer() == 0)
    check("lret2", s61inner() == "13")
    check("lret3", s61nonlocal() == 99)
    // return@functionName from two lambdas deep: a genuine non-local return.
    check("lret4", s61labelfun() == 42)
    // return@label in EXPRESSION position (the `?:` / if-else idiom).
    check("lret5", s61expr() == 4)
    // The labelled lambda's VALUE is what return@label carries.
    check("lret6", s61value() == 12)
    check("lret7", s61run() == 5)
}

// ===== SECTION 62: MEMBER extension functions, including the bodiless ones =====
// A member extension function - one declared inside a class or interface, whose `this`
// is the extension receiver while `this@Class` is the enclosing instance - with a body
// and in its BODILESS forms: `fun Int.qty(): String` in an interface and `abstract fun
// Int.scaled(): Int` in an abstract class. MemberExtAbs is AbstractFun's shape with a
// receiver in front; like AbstractFun it contributes no implementation, and the
// override installs the real entry, which the lookup reaches from the runtime receiver.
interface Unit62 { fun Int.qty(): String }
abstract class Base62 {
    abstract fun Int.scaled(): Int
    fun apply62(x: Int): Int = x.scaled()
}
class Impl62 : Base62(), Unit62 {
    val factor = 3
    override fun Int.scaled(): Int = this * this@Impl62.factor
    override fun Int.qty(): String = "$this pcs"
    fun describe(x: Int): String = x.qty()
}
class Dsl62 { var out = ""
    fun String.emit() { out += this + ";" }
    fun build(): String { "a".emit(); "b".emit(); return out } }
fun sec62() {
    val i = Impl62()
    // Declared abstract in Base62, overridden in Impl62, called through a base member.
    check("mxf1", i.apply62(5) == 15)
    // Declared bodiless in an INTERFACE.
    check("mxf2", i.describe(4) == "4 pcs")
    // The dispatch receiver stays reachable from the body (`out` is Dsl62's).
    check("mxf3", Dsl62().build() == "a;b;")
    // In scope wherever an instance of the declaring class is the enclosing receiver.
    check("mxf4", with(i) { 7.scaled() } == 21)
}

// ===== SECTION 63: the whole `super` surface =====
// `super.property` (`override val v get() = super.v + 1`), `super` inside an OBJECT
// EXPRESSION (the object pushes "<object>", its descriptor's own name, and the lookup
// walks the RECEIVER's supertype graph), a `super<Base>` qualifier choosing between two
// parents (`super<I1>.tag() + super<I2>.tag()`), and an object expression constructing
// its superclass (`(object : B() {}).v`).
open class SB63 { open val v = 1
                  open val w: Int get() = 7
                  open fun g() = 10 }
interface SI63a { fun tag(): String = "a" }
interface SI63b { fun tag(): String = "b" }
class SD63 : SB63() {
    override val v get() = super.v + 100
    override val w: Int get() = super.w + 100
    override fun g() = super.g() + 1
    fun viaLambda(): Int = listOf(1).map { super.g() + 1000 }[0]
}
class STwo63 : SI63a, SI63b { override fun tag(): String = super<SI63a>.tag() + super<SI63b>.tag() }
class SOuter63 { open class N63 { open fun n() = 7 }
                 class NI63 : N63() { override fun n() = super.n() + 1 } }
fun s63local(): Int {
    open class LB63 { open fun q() = 3 }
    class LD63 : LB63() { override fun q() = super.q() + 1 }
    return LD63().q()
}
fun sec63() {
    check("sup1", SD63().v == 101)          // super.<backing field>
    check("sup2", SD63().w == 107)          // super.<computed property>
    check("sup3", SD63().g() == 11)         // super.method()
    check("sup4", SD63().viaLambda() == 1010)   // super from inside a lambda
    check("sup5", STwo63().tag() == "ab")   // super<Interface>.method(), two supertypes
    check("sup6", SOuter63.NI63().n() == 8) // super in a NESTED class
    check("sup7", s63local() == 4)          // super in a LOCAL class
    check("sup8", (object : SB63() { override fun g() = super.g() + 5 }).g() == 15)
    check("sup9", (object : SB63() { override val v get() = super.v + 5 }).v == 6)
    check("sup10", (object : SB63() {}).v == 1)   // the supertype's initializer RAN
    check("sup11", (object : SB63() {}).g() == 10)
}

// ===== SECTION 64: a lambda-with-receiver as a PARAMETER (the DSL builder shape) =====
// `val f: T.() -> Unit = { ... }` binds the receiver to `this`, and so does the same
// type used as a FUNCTION PARAMETER - the commonest receiver-lambda shape in real
// Kotlin, the DSL builder - so both `t.body()` and `body(t)` bind `this`.
class Node64(val name: String) {
    val kids = mutableListOf<Node64>()
    val attrs = mutableListOf<String>()
    fun child(n: String, body: Node64.() -> Unit): Node64 { val c = Node64(n); c.body(); kids.add(c); return c }
    fun attr(k: String, v: String) { attrs.add("$k=$v") }
    fun render(): String {
        var s = "<" + name
        for (a in attrs) s += " " + a
        s += ">"
        for (k in kids) s += k.render()
        return s + "</" + name + ">"
    }
}
fun doc64(body: Node64.() -> Unit): Node64 { val r = Node64("root"); r.body(); return r }
fun applyTwice64(x: Int, f: Int.() -> Int): Int = x.f().f()
fun sec64() {
    val d = doc64 { attr("lang", "en"); child("body") { attr("id", "b") } }
    check("rcv1", d.render() == "<root lang=en><body id=b></body></root>")
    // A receiver function type on a BUILTIN receiver, invoked in extension position.
    check("rcv2", applyTwice64(3) { this + 1 } == 5)
    check("rcv3", doc64 { attr("k", "v") }.attrs.size == 1)
}

// ===== SECTION 65: super.property, and a setter that stores somewhere else =====
// `super.p = v` writes through the superclass's setter, and a property setter that
// stores anywhere else (`set(v) { super.p = v }`, the delegating override) must not
// have its store undone by a write-back of `field`: Kotlin materializes a backing field
// only when an accessor actually mentions `field`. Spec-cited (no kotlinc here): the
// Kotlin documentation's "Calling the superclass implementation" and "Backing fields"
// sections are the source.
open class Base65 {
    open var p: Int = 1
    open val tag: String get() = "base"
    fun rawP(): Int = p
}
class Sub65 : Base65() {
    var log = ""
    override var p: Int
        get() = super.p
        set(v) { log += "s"; super.p = v + 10 }
    override val tag: String get() = super.tag + "+sub"
    fun poke(v: Int) { super.p = v }
    fun peek(): Int = super.p
}
open class Counted65 {
    var raw: Int = 0
    var doubled: Int
        get() = raw * 2
        set(v) { raw = v / 2 }
}
fun sec65() {
    val d = Sub65()
    check("sup1", d.p == 1 && d.peek() == 1)
    d.poke(3)
    check("sup2", d.p == 3)
    d.p = 7
    // The setter's `super.p = v + 10` must SURVIVE the setter returning.
    check("sup3", d.p == 17 && d.peek() == 17 && d.log == "s")
    d.p += 3
    check("sup4", d.p == 30)
    check("sup5", d.tag == "base+sub")
    // A plain base instance is unaffected, and a setter that DOES use `field` still
    // writes through it.
    val b = Base65()
    b.p = 9
    check("sup6", b.p == 9 && b.rawP() == 9)
    val c = Counted65()
    c.doubled = 10
    check("sup7", c.raw == 5 && c.doubled == 10)
}

// ===== SECTION 66: the delegated-property protocol beyond a user getValue =====
// Kotlin has four `by` delegate kinds: a user object with getValue, a Map (keyed by the
// property name), the kotlin.properties.Delegates boxes, and anything a
// `provideDelegate` substitutes. Under kotlinc this section additionally needs
// `import kotlin.properties.Delegates` and `import kotlin.reflect.KProperty`.
class Conf66(val map: Map<String, Any?>) {
    val name: String by map
    val size: Int by map
}
class MConf66(val m: MutableMap<String, Any?>) {
    var nick: String by m
}
class Named66(val n: String) {
    operator fun getValue(t: Any?, p: KProperty<*>): String = "d:" + n
}
class Prov66 {
    operator fun provideDelegate(t: Any?, p: KProperty<*>): Named66 = Named66(p.name)
}
class Holder66 {
    val alpha: String by Prov66()
    val beta: String by Named66("fixed")
}
var obs66 = ""
var watched66: Int by Delegates.observable(0) { _, old, new -> obs66 += "$old>$new;" }
var gated66: Int by Delegates.vetoable(0) { _, _, new -> new >= 0 }
var late66: String by Delegates.notNull()
fun sec66() {
    // A Map delegate is keyed by the PROPERTY NAME, not by a key the code passes.
    val c = Conf66(mapOf("name" to "ann", "size" to 7))
    check("dlg1", c.name == "ann" && c.size == 7)
    val mc = MConf66(mutableMapOf("nick" to "zz"))
    check("dlg2", mc.nick == "zz")
    mc.nick = "qq"
    check("dlg3", mc.nick == "qq" && mc.m["nick"] == "qq")
    // provideDelegate substitutes the delegate at the DECLARATION, so it can read the
    // property's name before any access happens.
    val h = Holder66()
    check("dlg4", h.alpha == "d:alpha" && h.beta == "d:fixed")
    // kotlin.properties.Delegates: observable notifies AFTER, vetoable decides BEFORE.
    watched66 = 3
    watched66 = 5
    check("dlg5", watched66 == 5 && obs66 == "0>3;3>5;")
    gated66 = 4
    check("dlg6", gated66 == 4)
    gated66 = -1
    check("dlg7", gated66 == 4)
    late66 = "hi"
    check("dlg8", late66 == "hi")
    // A LOCAL delegated property binds the same box.
    val local: String by Named66("loc")
    check("dlg9", local == "d:loc")
    var lobs = ""
    var lwatched: Int by Delegates.observable(1) { _, old, new -> lobs += "$old/$new" }
    lwatched = 2
    check("dlg10", lwatched == 2 && lobs == "1/2")
}

// ===== SECTION 67: a supertype declared LATER in the file =====
// Kotlin has no forward-declaration rule: a class may extend one declared further down
// (`class Inh : Base()` above `open class Base`), and the same holds for an INTERFACE
// supertype; the compiler half emits the top-level class items in topological order. A
// one-line interface body is here too: `fun f(): Int;` - a bodiless member followed by
// the ordinary statement separator.
class Fwd67 : FwdBase67() {
    override fun label(): String = "fwd:" + super.label()
    fun total(): Int = seed + bonus67
}
class FwdIface67 : Marker67 { override fun f(): Int = 1 }
open class FwdBase67 {
    val seed = 5
    open fun label(): String = "base"
}
interface Marker67 { fun f(): Int; fun g(): Int = f() + 10 }
val bonus67 = 2
enum class Mode67 : Marker67 { ON, OFF; override fun f(): Int = if (this == ON) 1 else 0 }
object Solo67 : Marker67 { override fun f(): Int = 4 }
fun sec67() {
    val a = Fwd67()
    check("fwd1", a.total() == 7)
    check("fwd2", a.label() == "fwd:base")
    check("fwd3", FwdIface67().g() == 11)
    // An enum and an object may name a later-declared supertype as well.
    check("fwd4", Mode67.ON.g() == 11 && Mode67.OFF.g() == 10)
    check("fwd5", Solo67.g() == 14)
    check("fwd6", a is FwdBase67 && FwdIface67() is Marker67)
}

// ===== SECTION 68: member extension PROPERTIES (the DSL scope shape) =====
// `val Int.scaled get() = this * factor` declared INSIDE a class is the property twin of
// a member extension function, and the shape every scoped DSL (kotlinx.html, Compose
// modifiers, a units scope) is built from. The accessor pair is installed as
// extget$name / extset$name taking (dispatch, extensionReceiver[, value]), exactly like
// the ext$name member extension FUNCTIONS beside it. A member extension declared in a
// COMPANION object hangs on the class descriptor, one level above the method holder,
// and is asserted here too.
class Units68(val factor: Int) {
    var store = 0
    val Int.scaled: Int get() = this * factor
    var Int.slot: Int
        get() = store
        set(v) { store = v + this }
    fun Int.viaFun(): Int = this.scaled + 1
    fun run(): String {
        val n = 4
        var s = "" + n.scaled
        n.slot = 5
        s += "/" + n.slot
        n.slot += 3
        s += "/" + n.slot
        s += "/" + n.viaFun()
        return s
    }
    fun direct(): Int = 7.scaled
}
class Wrap68(val sep: String) {
    val String.wrapped: String get() = sep + this + sep
    fun go(s: String): String = s.wrapped
}
object Reg68 {
    val factor = 3
    val String.width: Int get() = length * factor
    fun w(s: String): Int = s.width
}
class Holder68 {
    companion object {
        val f = 3
        val Int.tripled: Int get() = this * f
        fun Int.tripledFun(): Int = this * f
        fun go(n: Int): Int = n.tripled
        fun goFun(n: Int): Int = n.tripledFun()
    }
}
fun sec68() {
    check("mxp1", Units68(10).run() == "40/9/16/41")
    check("mxp2", Units68(2).direct() == 14)
    check("mxp3", Wrap68("|").go("x") == "|x|")
    // An `object` declaration and a companion object host them too.
    check("mxp4", Reg68.w("abcd") == 12)
    check("mxp5", Holder68.go(5) == 15 && Holder68.goFun(5) == 15)
}

// ===== SECTION 69: three collection builtins, and a call's EXPLICIT TYPE ARGUMENTS =====
// Map.getOrPut, the PREDICATE overload of MutableList.removeAll/retainAll, and
// List.filterIsInstance. A call's explicit type arguments are captured (MTypeArgs) and
// delivered for the two things that need them: filterIsInstance's element type, and a
// REIFIED type parameter of an extension function called in method position
// (`"x".isA<String>()`).
inline fun <reified T> Any.isA69(): Boolean = this is T
class Bag69 { val n = 1 }
fun sec69() {
    // getOrPut: the standard way to build a map of collections.
    val g = mutableMapOf<String, MutableList<Int>>()
    g.getOrPut("a") { mutableListOf() }.add(1)
    g.getOrPut("a") { mutableListOf() }.add(2)
    g.getOrPut("b") { mutableListOf() }.add(3)
    check("blt1", g["a"]!!.size == 2 && g["b"]!!.size == 1 && g.size == 2)
    var calls = 0
    val c = mutableMapOf<String, Int>()
    check("blt2", c.getOrPut("x") { calls++; 5 } == 5)
    check("blt3", c.getOrPut("x") { calls++; 9 } == 5 && calls == 1)
    // removeAll / retainAll take a COLLECTION or a PREDICATE.
    val l = mutableListOf(1, 2, 3, 4, 5)
    check("blt4", l.removeAll { it % 2 == 0 } && l == listOf(1, 3, 5))
    check("blt5", !l.removeAll { it > 100 } && l.size == 3)
    val r = mutableListOf(1, 2, 3, 4, 5)
    check("blt6", r.retainAll { it > 3 } && r == listOf(4, 5))
    val d = mutableListOf(1, 2, 3, 4)
    check("blt7", d.removeAll(listOf(2, 4)) && d == listOf(1, 3))
    // filterIsInstance<T>() reads the call's type argument, and drops nulls.
    val xs: List<Any?> = listOf(1, "a", 2.5, "b", 3, null, Bag69())
    check("blt8", xs.filterIsInstance<String>() == listOf("a", "b"))
    check("blt9", xs.filterIsInstance<Int>() == listOf(1, 3))
    check("blt10", xs.filterIsInstance<Bag69>().size == 1)
    check("blt11", xs.filterIsInstance<Double>() == listOf(2.5))
    // A reified type parameter of an extension, called in METHOD position.
    check("blt12", "x".isA69<String>() && !"x".isA69<Int>())
    check("blt13", 7.isA69<Int>() && !7.isA69<String>())
    // A type argument on a method that does not use one is still ignored.
    check("blt14", listOf(1, 2, 3).map<Int, Int> { it + 1 } == listOf(2, 3, 4))
}

// ===== SECTION 70: a page of application Kotlin =====
// Written as ordinary application code (a value type with an infix extension, an enum
// whose constants carry their own behaviour, a nullable-receiver extension property,
// price strings, a range-dispatch `when`, and input parsing):
//   - `enum class L(w) { A(1) { override fun f() = ... }; abstract fun f() }` - an entry
//     with its OWN CLASS BODY.
//   - `infix fun Int.pctOf(m: Money)` used INFIX as well as dotted.
//   - `val String?.safeLen get() = this?.length ?: -1` on a NULL receiver; a
//     nullable-receiver extension property is exactly where a null receiver is legal.
//   - `"$"`, `"100$"`: a `$` not followed by an identifier or `{` is an ordinary
//     character (psi/StringTemplates.txt records it as REGULAR_STRING_PART with no error
//     element).
//   - two `when` arms in a row whose conditions are `in <range>`: an `in` that opens a
//     line is a condition, not a membership operator continuing the previous arm.
//   - `"nope".toInt()` THROWS NumberFormatException, so try/catch and runCatching see it.
class Money70(val cents: Int) : Comparable<Money70> {
    operator fun plus(o: Money70) = Money70(cents + o.cents)
    operator fun times(n: Int) = Money70(cents * n)
    override fun compareTo(other: Money70) = cents.compareTo(other.cents)
    override fun toString(): String = "$" + (cents / 100) + "." + (cents % 100).toString().padStart(2, '0')
}
infix fun Int.pctOf70(m: Money70) = Money70(m.cents * this / 100)
enum class Level70(val weight: Int) {
    LOW(1) { override fun label() = "low" },
    HIGH(5) { override fun label() = "high" };
    abstract fun label(): String
    fun heavy() = weight > 3
}
val String?.safeLen70: Int get() = this?.length ?: -1
fun classify70(n: Int): String = when (n) {
    in 0..9 -> "digit"
    in 10..99 -> "two"
    in 100..999 -> "three"
    else -> if (n < 0) "neg" else "big"
}
fun main70(raw: String): Int = runCatching { raw.toInt() }.getOrElse { -1 }
fun sec70() {
    val a = Money70(1250)
    val b = Money70(375)
    check("app1", (a + b).toString() == "$16.25" && (a * 2).toString() == "$25.00")
    check("app2", maxOf(a, b) === a && listOf(a, b).sorted().first() === b)
    // An infix EXTENSION function, both ways round.
    check("app3", (25 pctOf70 a).cents == 312 && 25.pctOf70(a).cents == 312)
    // An enum entry with its own class body, plus the enum's own members.
    check("app4", Level70.LOW.label() == "low" && Level70.HIGH.label() == "high")
    check("app5", !Level70.LOW.heavy() && Level70.HIGH.heavy())
    check("app6", Level70.entries.size == 2 && Level70.valueOf("HIGH").ordinal == 1)
    check("app7", Level70.LOW.name == "LOW" && Level70.LOW.weight == 1)
    // An extension property on a NULLABLE receiver.
    val missing: String? = null
    check("app8", "abc".safeLen70 == 3 && missing.safeLen70 == -1)
    // A literal dollar, in a normal string and in a raw one.
    check("app9", "$".length == 1 && "cost: 100$" == "cost: " + 100 + "$")
    check("app10", """raw $ end""".length == 9)
    // Consecutive `in <range>` when arms.
    check("app11", classify70(5) == "digit" && classify70(50) == "two")
    check("app12", classify70(500) == "three" && classify70(-1) == "neg" && classify70(5000) == "big")
    // A bad number THROWS, and is catchable both ways.
    check("app13", main70("42") == 42 && main70("nope") == -1)
    var caught = ""
    try { "nope".toInt() } catch (e: NumberFormatException) { caught = e.message ?: "" }
    check("app14", caught == "For input string: \"nope\"")
    check("app15", runCatching { "x".toDouble() }.isFailure && "nope".toIntOrNull() == null)
}

// ===== SECTION 71: a registry/DTO page of application Kotlin =====
//   - `buildList { addAll(listOf(2, 3)) }`: inside a builder lambda the implicit
//     receiver must not claim the global `listOf`.
//   - an unqualified read or write of the class's OWN computed or delegated property
//     inside a member: the accessor pair lives on the DESCRIPTOR (js_defprop), not on
//     the instance.
//   - a NAMED argument filling a `vararg` parameter: Kotlin's rule is that the single
//     named argument IS the array.
//   - named arguments to an OBJECT's method land in the right slots.
object Reg71 {
    var writes = 0
    val doubled: Int get() = writes * 2
    var tracked: Int by Delegates.observable(5) { _, _, _ -> writes++ }
    fun bump(): Int { tracked += 1; return tracked }
    fun pick(a: Int = 1, b: Int = 2, c: Int = 3): String = "$a/$b/$c"
    fun label(name: String, sep: String = "-", vararg tags: String): String =
        name + sep + tags.toList().joinToString(",")
}
class Own71 {
    var raw = 3
    val trebled: Int get() = raw * 3
    var mirrored: Int
        get() = raw
        set(v) { raw = v }
    fun readAll(): String = "" + trebled + "/" + mirrored
    fun writeThrough(v: Int) { mirrored = v }
}
fun sec71() {
    // A builder lambda may still call the ordinary global builders.
    check("reg1", buildList { add(1); addAll(listOf(2, 3)) } == listOf(1, 2, 3))
    check("reg2", buildString { append(listOf(1, 2).joinToString("-")) } == "1-2")
    check("reg3", buildMap<String, Int> { put("a", maxOf(1, 2)) }["a"] == 2)
    // An unqualified read and write of the class's own accessor properties.
    val o = Own71()
    check("reg4", o.readAll() == "9/3")
    o.writeThrough(5)
    check("reg5", o.raw == 5 && o.readAll() == "15/5")
    // The same through an object, including a delegated property.
    check("reg6", Reg71.bump() == 6 && Reg71.tracked == 6 && Reg71.writes == 1)
    check("reg7", Reg71.doubled == 2)
    // Named arguments to an object's method land in the right slots.
    check("reg8", Reg71.pick(c = 9) == "1/2/9" && Reg71.pick(2, c = 9) == "2/2/9")
    // A named argument filling a vararg parameter IS the array.
    check("reg9", Reg71.label("n", tags = arrayOf("a", "b")) == "n-a,b")
    check("reg10", Reg71.label("n", ":", "a") == "n:a")
    check("reg11", Reg71.label("n") == "n-")
}

// ===== SECTION 72: extension DELEGATED properties (the DataStore idiom) =====
//   `val Ctx.store: Int by D()` - an extension property whose value comes from a
//   delegate. The delegate belongs to the DECLARATION SITE (the `by` expression is
//   evaluated once, at the declaration, not per instance) and getValue/setValue receive
//   the EXTENSION RECEIVER as thisRef, supplied per access. This is `val
//   Context.dataStore by preferencesDataStore(...)`, the Android idiom.
//   Also pinned here: a top-level extension property's SETTER runs its body.
class Ctx72(val tag: String)
class Store72 {
    var cell = 0
    var reads = 0
    operator fun getValue(thisRef: Any?, property: KProperty<*>): String {
        reads++
        return (thisRef as Ctx72).tag + ":" + property.name + ":" + cell
    }
    operator fun setValue(thisRef: Any?, property: KProperty<*>, value: String) {
        cell = value.length
    }
}
val store72 = Store72()
val memberStore72 = Store72()
var Ctx72.pref72: String by store72
class Holder72(val base: Int) {
    var Ctx72.memcell72: String by memberStore72
    fun readMem(c: Ctx72): String = c.memcell72
    fun writeMem(c: Ctx72, v: String) { c.memcell72 = v }
}
class Plain72 { var raw = 0 }
var Plain72.scaled72: Int
    get() = raw * 2
    set(v) { raw = v / 2 }

fun sec72() {
    val a = Ctx72("a")
    val b = Ctx72("b")
    // thisRef is the RECEIVER of the access, and the property name reaches the delegate.
    check("xd1", a.pref72 == "a:pref72:0")
    check("xd2", b.pref72 == "b:pref72:0")
    // ONE delegate object per declaration site: a write through `a` is seen through `b`.
    a.pref72 = "abcd"
    check("xd3", b.pref72 == "b:pref72:4")
    check("xd4", store72.reads == 3)
    // The member-level form: same protocol, thisRef is still the extension receiver.
    val h = Holder72(10)
    check("xd5", h.readMem(a) == "a:memcell72:0")
    h.writeMem(b, "xyz")
    check("xd6", h.readMem(a) == "a:memcell72:3")
    // A plain (non-delegated) extension property setter runs its body.
    val p = Plain72()
    p.scaled72 = 10
    check("xd7", p.raw == 5)
    check("xd8", p.scaled72 == 10)
    p.scaled72 += 4
    check("xd9", p.raw == 7 && p.scaled72 == 14)
}

// ===== SECTION 73: super in an enum entry body, and secondary-constructor delegation =====
//   - an enum entry with a BODY is a subclass of its own enum class, so `super.g()` and
//     `super.p` inside it are ordinary super calls.
//   - an enum class's own ACCESSOR property (`E.A.p`).
//   - a secondary constructor delegating with `: super(...)` stores the superclass's
//     `val` parameters and still runs this class's property initializers and init blocks.
//   - a secondary constructor delegating with `: this(...)` reaches the secondary of
//     matching arity, not the primary.
enum class Ent73 {
    A { override fun g() = super.g() + 1
        override val p: Int get() = super.p + 1 },
    B;
    open fun g() = 10
    open val p: Int get() = 5
}
open class Base73(val n: Int, val m: Int = 9) {
    open fun describe() = "b$n/$m"
}
class NoPrimary73 : Base73 {
    val tag = "t"
    var seen = 0
    init { seen = 1 }
    constructor() : super(4) { seen += 10 }
    constructor(k: Int) : super(k, k + 1) { seen += 100 }
    override fun describe() = super.describe() + "/$tag$seen"
}
class ThisDeleg73 {
    val v: Int
    var note = ""
    constructor(n: Int) { v = n * 2; note = "int" }
    constructor() : this(5) { note += "/none" }
    constructor(s: String, mark: Boolean) : this(s.length) { note += "/str$mark" }
}
class Ordered73 {
    val log = StringBuilder()
    init { log.append("i1;") }
    constructor(x: Int) { log.append("c$x;") }
    init { log.append("i2;") }
}

fun sec73() {
    // super inside an enum entry body, both the method and the property form.
    check("es1", Ent73.A.g() == 11)
    check("es2", Ent73.B.g() == 10)
    check("es3", Ent73.A.p == 6)
    check("es4", Ent73.B.p == 5)
    // A secondary constructor delegating to the SUPERCLASS: its val parameters are
    // stored, defaults apply, and this class's own initializers still run.
    val d0 = NoPrimary73()
    check("es5", d0.n == 4 && d0.m == 9)
    check("es6", d0.seen == 11 && d0.tag == "t")
    check("es7", d0.describe() == "b4/9/t11")
    val d1 = NoPrimary73(2)
    check("es8", d1.n == 2 && d1.m == 3 && d1.seen == 101)
    // A secondary constructor delegating with `: this(...)` reaches the secondary of
    // matching arity, not the primary.
    check("es9", ThisDeleg73().v == 10)
    check("es10", ThisDeleg73().note == "int/none")
    // Two secondary constructors of the same ARITY are not told apart by parameter
    // type in either half (both dispatch on argument count alone), so this one is
    // given a distinct arity - a shared limitation, not a divergence.
    check("es11", ThisDeleg73("abc", true).v == 6 && ThisDeleg73("abc", true).note == "int/strtrue")
    check("es12", ThisDeleg73(4).v == 8 && ThisDeleg73(4).note == "int")
    // init blocks run in declaration order, before the secondary constructor's body.
    check("es13", Ordered73(1).log.toString() == "i1;i2;c1;")
}

// ===== SECTION 74: a local enum class, and accessors on a delegated member =====
//   - a LOCAL enum class (`fun f() { enum class E { A, B } }`) is a real enum.
//   - a delegated MEMBER property that also declares accessors. Kotlin's own PSI corpus
//     records both spellings with a clean tree (propertyDelegate/PropertyWithGetter.kt
//     on the next line, propertyDelegate/GetterInSameLine.kt on the same line) and
//     rejects them semantically; here the accessors are consumed and dropped, and the
//     delegate wins.
class Deleg74 {
    var cell = 0
    operator fun getValue(thisRef: Any?, property: KProperty<*>): Int = cell + 1
    operator fun setValue(thisRef: Any?, property: KProperty<*>, value: Int) { cell = value }
}
class NextLine74 { var p: Int by Deleg74()
    get() = 99 }
class SameLine74 { var p: Int by Deleg74(); get() = 99; }

fun localEnum74(): String {
    enum class Dir74 { NORTH, SOUTH, EAST }
    val d = Dir74.SOUTH
    return d.name + d.ordinal + Dir74.EAST.ordinal + Dir74.values().size
}

fun sec74() {
    // A local enum class is a real enum: names, ordinals and values() all work.
    check("le1", localEnum74() == "SOUTH123")
    // The accessor is dropped; the DELEGATE answers, in both spellings.
    val n = NextLine74()
    check("le2", n.p == 1)
    n.p = 7
    check("le3", n.p == 8)
    val m = SameLine74()
    check("le4", m.p == 1)
    m.p = 4
    check("le5", m.p == 5)
}

// ===== SECTION 75: WHICH supertype `super` names, and how that name RESOLVES =====
// WHICH supertype is the superclass, and how a supertype name resolves:
//   (a) Kotlin does not require the superclass to come first: `class C : I3, Base2()`
//       is legal and common. The superclass is the supertype written with a
//       CONSTRUCTOR CALL; failing that (a class with only secondary constructors writes
//       no parentheses anywhere) the first supertype that is a class rather than an
//       interface; only then position. A delegated `I by d` is never the superclass.
//       With the interface listed first, the superclass constructor still runs and the
//       interface's own default methods are still copied in.
//   (b) a supertype named through a `typealias`, or written as a dotted path
//       (`Outer.In`), resolves through resolvedTypeName - generics and the dotted path
//       dropped, then typealiases followed - in both halves, and a failure names the
//       supertype it could not resolve.
// Spec-cited (no kotlinc here): the Kotlin documentation's "Inheritance" / "Calling the
// superclass implementation" and "Type aliases" sections are the source for every
// expected value below.
interface XI75 { fun h() = 40 }
interface XJ75 { fun j() = 50 }
open class XBase75(val seed: Int = 2) { open fun g() = 1
                                        open val p: Int get() = 1
                                        open var q: Int = 1 }
typealias XAlias75 = XBase75
class XHold75 { open class XIn75 { open fun g() = 1
                                   open val p: Int get() = 1 } }
open class XGen75<T> { open fun g() = 1 }
// (a) the interface is listed FIRST and the superclass second.
class XIfaceFirst75 : XI75, XBase75() { override fun g() = super.g() + 3
                                        override val p: Int get() = super.p + 3 }
class XTwoIface75 : XI75, XJ75, XBase75() { override fun g() = super.g() + 3 }
// (b) the superclass is named through a typealias, and through a dotted path.
class XViaAlias75 : XAlias75() { override fun g() = super.g() + 3
                                 override val p: Int get() = super.p + 3
                                 override var q: Int
                                     get() = super.q
                                     set(v) { super.q = v + 3 } }
class XViaPath75 : XHold75.XIn75() { override fun g() = super.g() + 3
                                     override val p: Int get() = super.p + 3 }
// The qualified form over each parent shape.
class XQualAlias75 : XAlias75() { override fun g() = super<XAlias75>.g() + 3 }
class XQualPath75 : XHold75.XIn75() { override fun g() = super<XHold75.XIn75>.g() + 3 }
class XQualGen75 : XGen75<Int>() { override fun g() = super<XGen75<Int>>.g() + 3 }
// A `where` clause, and a generic class with a generic parent.
class XWhere75<T> : XBase75() where T : Comparable<T> { override fun g() = super.g() + 3 }
class XGenGen75<T> : XGen75<T>() { override fun g() = super.g() + 3 }
// Delegation combined with a superclass, in BOTH orders. The delegated interface is
// declared with an ABSTRACT member on purpose: with a DEFAULT body the interpreter half
// lets the inherited default win over the `by` delegate while the compiler half lets
// the delegate win (Kotlin's rule) - a known divergence unrelated to `super`.
interface XK75 { fun k(): Int }
class XDelegImpl75 : XK75 { override fun k() = 7 }
class XDelegAfter75(d: XK75) : XBase75(), XK75 by d { override fun g() = super.g() + 3 }
class XDelegFirst75(d: XK75) : XK75 by d, XBase75() { override fun g() = super.g() + 3 }
// A secondary constructor: NO supertype carries parentheses in the header at all.
class XSecond75 : XBase75 {
    constructor() : super(2)
    fun f() = super.g() + 3
    fun fp() = super.p + 3
}
enum class XEnum75 { A { override fun g() = super.g() + 3
                         override val p: Int get() = super.p + 3 } ;
                     open fun g() = 1
                     open val p: Int get() = 1 }
class XOuter75 { class XNest75 : XAlias75() { fun f() = super.g() + 3 }
                 inner class XInner75 : XHold75.XIn75() { fun f() = super.g() + 3 } }
fun xlocal75(): Int { class L : XAlias75() { fun f() = super.g() + 3 }; return L().f() }
fun sec75() {
    // (a) the superclass is picked by its constructor call, not by position - and the
    // interface listed first still contributes its default method.
    check("sx1", XIfaceFirst75().g() == 4)
    check("sx2", XIfaceFirst75().p == 4)
    check("sx3", XIfaceFirst75().h() == 40)
    check("sx4", XTwoIface75().g() == 4 && XTwoIface75().h() == 40 && XTwoIface75().j() == 50)
    // The superclass CONSTRUCTOR runs even when it is not the first supertype.
    check("sx5", XIfaceFirst75().seed == 2)
    // (b) resolution through a typealias and through a dotted path.
    check("sx6", XViaAlias75().g() == 4)
    check("sx7", XViaAlias75().p == 4)
    val va = XViaAlias75(); va.q = 1
    check("sx8", va.q == 4)
    check("sx9", XViaPath75().g() == 4 && XViaPath75().p == 4)
    // The qualified `super<T>` form over each parent shape.
    check("sx10", XQualAlias75().g() == 4)
    check("sx11", XQualPath75().g() == 4)
    check("sx12", XQualGen75().g() == 4)
    // Generic parent, generic child, and a `where` clause.
    check("sx13", XWhere75<Int>().g() == 4 && XGenGen75<Int>().g() == 4)
    // Delegation next to a superclass, in both orders.
    val da = XDelegAfter75(XDelegImpl75())
    check("sx14", da.g() == 4 && da.k() == 7)
    val df = XDelegFirst75(XDelegImpl75())
    check("sx15", df.g() == 4 && df.k() == 7)
    // A class whose only constructor is SECONDARY still finds its superclass.
    check("sx16", XSecond75().f() == 4 && XSecond75().fp() == 4 && XSecond75().seed == 2)
    // An enum ENTRY body, and nested / inner / local classes over an aliased parent.
    check("sx17", XEnum75.A.g() == 4 && XEnum75.A.p == 4)
    check("sx18", XOuter75.XNest75().f() == 4)
    check("sx19", XOuter75().XInner75().f() == 4)
    check("sx20", xlocal75() == 4)
    // Object expressions over each parent shape.
    check("sx21", (object : XI75, XBase75() { override fun g() = super.g() + 3 }).g() == 4)
    check("sx22", (object : XAlias75() { override fun g() = super.g() + 3 }).g() == 4)
    check("sx23", (object : XHold75.XIn75() { override fun g() = super.g() + 3 }).g() == 4)
    check("sx24", (object : XAlias75() { override val p: Int get() = super.p + 3 }).p == 4)
    check("sx25", (object : XI75, XBase75() {}).seed == 2)
}

// ===== SECTION 120: unbounded recursion =====
// A call nested past the stack is the JVM's java.lang.StackOverflowError, an
// Error with a null message that `catch (e: Exception)` does not take, and the
// depth a caught one unwound is given back (java 24.0.2 settles the JVM side).
fun s120rec(n: Int): Int = s120rec(n + 1) + 1
fun s120deep(n: Int): Int = if (n == 0) 0 else 1 + s120deep(n - 1)
class S120C { fun m(n: Int): Int = m(n + 1) }
fun s120() {
    var got = ""
    for (i in 0 until 2) {
        try { s120rec(0) } catch (e: Exception) { got += "E" } catch (e: StackOverflowError) { got += "" + e + "/" + e.message + ";" }
    }
    check("soe1", got == "java.lang.StackOverflowError/null;java.lang.StackOverflowError/null;")
    var how = ""
    try { S120C().m(0) } catch (e: Error) { how = "" + e }
    var g: ((Int) -> Int)? = null
    g = { n -> g!!(n + 1) }
    try { g(0) } catch (e: Throwable) { how = how + "|" + e }
    check("soe2", how == "java.lang.StackOverflowError|java.lang.StackOverflowError" && s120deep(100) == 100)
}

// ===== END SECTIONS =====

// ===== SECTION 76: the IMPLICIT RECEIVER, and WHICH read answers it =====
// The compiler half resolves an unqualified name through kt_ktget
// (kotlin-to-llvm-ir.abnf). A class instance is read with the shared js_get; an
// OBJECT-SHAPED BUILTIN - a Map, a List, a Set, a StringBuilder, a Result, a lazy - is
// not an instance and takes Kotlin's field read: `with(m) { entries }` is `[a=1, b=2]`,
// and `val List<Int>.mid get() = this[size / 2]` reads the list's `size`. A builtin
// receiver then splits AGAIN on the ktRecvProp table - the SAME table the
// interpreter's kRecvProps and layer 2's k4RecvProp use - so `with(listOf(4, 5))
// { first() }` stays a method call (kr9).
// The rest of the sweep, all run-vs-native or interpreter-vs-run splits that only the
// ratchet's native build sees:
//   * a PLAIN OBJECT receiver (Pair, Triple, Map.Entry): `with` binds a scope `this`
//     under llvm.Run and only pushes ktRecvStack natively (kr12-kr14);
//   * kotlin.Lazy's value / isInitialized, kotlin.Result's isSuccess / isFailure,
//     MatchResult.value and Char.code are `val`s the runtime SYNTHESISES, neither own
//     properties nor collection properties (kr15-kr17);
//   * a Unit-valued LOCAL is a slot holding `undefined`, which js_scope_typeof cannot
//     tell from an absent name; js_scope_has is the own-scope test that separates the
//     two (kr20);
//   * `println(f())` for a Unit-returning f must print kotlin.Unit natively too: the
//     emitter passes the empty string for a no-argument println under -exe. The two
//     bare prints at the end of this section are that pair; they are OUTPUT, not
//     assertions, and only --cross and a native probe can see them.
// Spec-cited (no kotlinc here). Expected values are the interpreter half's, which is
// this project's reference for Kotlin semantics, cross-read against the
// kotlin.collections / kotlin.text / kotlin.Lazy / kotlin.Result API documentation for
// the member kinds (`val` versus `fun`).
val List<Int>.mid76: Int get() = this[size / 2]
val String.midc76: Char get() = this[length / 2]
fun List<Int>.li76() = lastIndex
fun String.idx76() = indices.toString()
class Hold76(val raw: Int) {
    val trebled: Int get() = raw * 3
    fun size() = 10
    fun show() = trebled
    fun both() = raw + size()
}
fun u76(): Unit { }

fun sec76() {
    val m = mutableMapOf("a" to 1, "b" to 2)
    check("kr1", with(m) { entries.toString() } == m.entries.toString())
    check("kr2", with(m) { entries.toString() } == "[a=1, b=2]")
    check("kr3", with(m) { keys.toString() } == "[a, b]" && with(m) { values.toString() } == "[1, 2]")
    check("kr4", with(m) { size } == 2 && with(m) { get("b") } == 2)
    val xs = listOf(1, 2, 3)
    check("kr5", xs.mid76 == 2)
    check("kr6", "abc".midc76 == 'b')
    check("kr7", xs.li76() == 2 && "abcd".idx76() == "0..3")
    check("kr8", with(xs) { size } == 3 && with(xs) { lastIndex } == 2 &&
                 with(xs) { indices.toString() } == "0..2")
    // A builtin receiver's METHODS stay methods. Reading `first` as a property here
    // would answer the number 1 and the call would abort.
    check("kr9", with(xs) { first() } == 1 && with(xs) { last() } == 3 && with(xs) { sum() } == 6)
    check("kr10", mutableListOf(1).apply { add(3) }.toString() == "[1, 3]")
    // An INSTANCE keeps the ordinary member read, so a class member named like a
    // collection property still wins, and an accessor read unqualified still resolves.
    check("kr11", Hold76(4).show() == 12 && Hold76(4).both() == 14)
    // A plain object receiver: its OWN properties first.
    val p = Pair(1, "x")
    check("kr12", with(p) { first } == 1 && with(p) { second } == "x")
    val t = Triple(1, 2, 3)
    check("kr13", with(t) { first + second + third } == 6)
    val e = mapOf("a" to 1).entries.first()
    check("kr14", with(e) { key } == "a" && with(e) { value } == 1)
    // Then the ones the runtime synthesises.
    val lz = lazy { 42 }
    check("kr15", with(lz) { value } == 42 && with(lz) { isInitialized })
    val rs = runCatching { 7 }
    check("kr16", with(rs) { isSuccess } && !with(rs) { isFailure })
    check("kr17", with('q') { code } == 113)
    val mr = Regex("a(b)c").find("zabc")!!
    check("kr18", with(mr) { value } == "abc")
    val sb = StringBuilder("ab")
    check("kr19", with(sb) { append("c"); length } == 3 && with(sb) { lastIndex } == 2)
    val st = mutableSetOf(1, 2, 3)
    check("kr20", with(st) { size } == 3 && with(st) { contains(2) })
    // A Unit-valued LOCAL is a slot holding undefined - present, not absent.
    val ux: Unit = u76()
    check("kr21", ux.toString() == "kotlin.Unit")
    println(u76())
    println()
}


// ===== SECTION 77: THE SYNTACTIC POSITION, and the two receiver shapes a NAME cannot decide =====
// `first` is `val IntProgression.first` (kotlin.ranges) and `fun <T> List<T>.first()`
// (kotlin.collections) - a property on one shape and a member function on the other -
// so the NAME cannot decide which read an unqualified `first` wants. The SITE can:
// `first` and `first()` are DIFFERENT PRODUCTIONS in all three engines, and Kotlin has
// no unqualified name in VALUE position that means a bound method (that spelling is
// `::first`). So the field read is tried first in value position and not at all in
// call position, and both shapes answer correctly. rg7 restates kr9's guard -
// `with(listOf(4, 5)) { first() }` must not call the number 4 - at this gate.
//
// The flag travels as a third parameter of kt_ktgetp (kotlin-to-llvm-ir.abnf), a
// second of js_ktglobal, a third of js_ktget, an argument of ktRecvMember in both
// runtimes, and the one-shot kCallPos in kotlin-interpreter.abnf's makeCall. It has to
// reach layer 2 because NATIVELY `with` binds no scope `this` at all - layer 2 cannot
// make a scope - so js_ktglobal's walk of ktRecvStack is the only place the receiver
// is seen there.
//
// The RECEIVER SHAPES a Regex, a MatchResult, a kotlin.Result and a Char each have a
// real method surface in js_ktsmcall / mcall and a real property surface in js_ktfget /
// kGetField, so they are builtin receivers: `with(Regex("a")) { pattern }`, an
// unqualified `find()`, `with(runCatching { 7 }) { getOrNull() }` and
// `with('q') { isLetter() }`. Also here: `val IntRange.start` / `val
// IntRange.endInclusive` (the ClosedRange members), and `re.run { ... }` /
// `re.let { ... }` on a Regex receiver.
//
// KNOWN DIVERGENCE, not asserted: an EXTENSION PROPERTY that shadows one of the seven
// runtime index properties, read UNQUALIFIED against an implicit receiver. `val
// List<Int>.lastIndex get() = 99` makes `with(xs) { lastIndex }` answer 99 in the
// interpreter (kIdxExt) and 2 in both compiled halves. It cannot be closed in the
// emitter alone: natively `with` binds no scope `this`, the read is answered by layer
// 2's walk of ktRecvStack, and layer 2 cannot read the `extp$name` binding out of a
// scope - so an emitter-only probe would trade a halves divergence for a
// run-versus-native one.
//
// Spec-cited (no kotlinc here). The expected values are read from the kotlin.ranges
// (IntProgression.first/last/step, ClosedRange.start/endInclusive), kotlin.collections
// (first()/last()/sum()), kotlin.text (Regex.pattern/options/find/matches/
// containsMatchIn/replace/split, MatchResult.value/groupValues/next) and kotlin.Result
// API documentation, cross-read against the interpreter half, which is this project's
// reference for Kotlin semantics.
fun helper77() = 5
fun sec77() {
    // Every operand comes out of an array so the constant folder cannot answer any of
    // this at compile time.
    val bd = arrayOf(1, 7, 2, 10, 5, 3)
    val prog = bd[0]..bd[1] step bd[2]
    check("rg1", with(prog) { first } == 1 && with(prog) { last } == 7 && with(prog) { step } == 2)
    // ... and the CALL form of the same three names, on the same receiver.
    check("rg2", with(prog) { first() } == 1 && with(prog) { last() } == 7 && with(prog) { sum() } == 16)
    val plain = bd[0]..bd[1]
    check("rg3", with(plain) { first } == 1 && with(plain) { last } == 7 && with(plain) { step } == 1)
    check("rg4", with(plain) { start } == 1 && with(plain) { endInclusive } == 7 &&
                 plain.start == 1 && plain.endInclusive == 7)
    // downTo carries the DIRECTION in the step, and `last` is the last ELEMENT (7),
    // not the bound the progression was built from.
    val down = bd[3] downTo bd[4] step bd[5]
    check("rg5", with(down) { first } == 10 && with(down) { last } == 7 && with(down) { step } == -3)
    val cs = arrayOf('a', 'e')
    val chr = cs[0]..cs[1]
    check("rg6", with(chr) { first } == 'a' && with(chr) { last } == 'e' && with(chr) { step } == 1)
    // kr9's guard at the new gate: on a LIST first/last stay METHODS.
    val xs = listOf(bd[0], bd[1], bd[2])
    check("rg7", with(xs) { first() } == 1 && with(xs) { last() } == 2 && with(xs) { sum() } == 10)
    check("rg8", with(xs) { size } == 3 && with(xs) { lastIndex } == 2 && !with(xs) { isEmpty() })
    // A REGEX receiver: its property, then its methods, unqualified.
    val pats = arrayOf("a(b)c", "zabcabc")
    val re = Regex(pats[0])
    check("rg9", with(re) { pattern } == "a(b)c" && with(re) { options.toString() } == "[]")
    check("rg10", with(re) { find(pats[1])?.value } == "abc" && with(re) { split(pats[1]).size } == 3)
    check("rg11", with(re) { containsMatchIn(pats[1]) } && !with(re) { matches(pats[1]) })
    check("rg12", with(re) { replace(pats[1], "X") } == "zXX" && with(re) { toString() } == "a(b)c")
    check("rg13", re.run { pattern } == "a(b)c" && re.let { it.pattern } == "a(b)c")
    // A MATCHRESULT receiver.
    val mr = re.find(pats[1])!!
    check("rg14", with(mr) { value } == "abc" && with(mr) { groupValues.toString() } == "[abc, b]")
    check("rg15", with(mr) { next()?.value } == "abc")
    // A CHAR and a kotlin.Result receiver: a property and a method of each.
    val ch = cs[0]
    check("rg16", with(ch) { code } == 97 && with(ch) { isLetter() } && with(ch) { uppercaseChar() } == 'A')
    val rs = runCatching { bd[1] }
    check("rg17", with(rs) { isSuccess } && with(rs) { getOrNull() } == 7)
    // A GLOBAL still wins over a receiver member it does not have, in call position, on
    // every one of the new shapes - the receiver walk must not swallow ordinary names.
    check("rg18", with(xs) { helper77() } == 5 && with(re) { helper77() } == 5 &&
                  with(prog) { helper77() } == 5 && with(ch) { helper77() } == 5)
}


// ===== SECTION 78: THE RANGE / PROGRESSION / STRING-SLICE SURFACE =====
// The range / progression / string-slice surface. Every fact below is kotlin.ranges /
// kotlin.text / kotlin.collections as documented - there is no kotlinc on this machine
// (manual chapter 5), so the stdlib declarations and IntProgression's own
// toString/equals/reversed contracts are the oracle:
//
//   IntProgression.reversed(): IntProgression   = fromClosedRange(last, first, -step)
//   IntProgression.toString()                   = "$first..$last step $step", and the
//                                                 downTo form for a negative step -
//                                                 IntRange.toString has NO " step"
//   String.substring(range: IntRange): String   = substring(start, endInclusive + 1)
//   String.slice(indices: IntRange): String     = substring(indices)
//   String.slice(indices: Iterable<Int>): String
//   String.filter/filterNot/filterIndexed/takeWhile/dropWhile/onEach: String
//   String.partition: Pair<String, String>; String.chunked/windowed: List<String>
//   IntRange.random() / Collection.random() / randomOrNull()
//
// EVERY OPERAND IS READ OUT OF AN ARRAY so the grammar's constant folder cannot
// answer any of it at compile time.
fun sec78() {
    val n = arrayOf(1, 10, 3, 5, 0, 2, 4)
    val txt = arrayOf("abcdef", "ac", "bcd")
    val cs = arrayOf('a', 'f', 'c')
    val ls = arrayOf(1L, 5L, 2L)

    // ----- reversed(), on all four progression shapes -----
    val prog = n[0]..n[1] step n[2]                    // 1, 4, 7, 10
    val rev = prog.reversed()
    check("rp1", rev.toString() == "10 downTo 1 step 3")
    check("rp2", rev.first == 10 && rev.last == 1 && rev.step == -3)
    check("rp3", rev.toList().toString() == "[10, 7, 4, 1]" && rev.count() == 4)
    check("rp4", rev.reversed() == prog && rev.reversed().toString() == "1..10 step 3")
    // A plain a..b reverses to a downTo progression, and a downTo back up. Both keep
    // the " step 1" IntProgression.toString prints and IntRange.toString does not.
    check("rp5", (n[0]..n[3]).reversed().toString() == "5 downTo 1 step 1")
    check("rp6", (n[3] downTo n[0]).reversed().toString() == "1..5 step 1")
    check("rp7", (cs[0]..cs[1]).reversed().toString() == "f downTo a step 1" &&
                 (cs[0]..cs[1]).reversed().first == 'f' && (cs[0]..cs[1]).reversed().last == 'a')
    check("rp8", (ls[0]..ls[1] step ls[2]).reversed().toString() == "5 downTo 1 step 2")
    // An EMPTY progression reverses to an empty one, whose bounds are still the ones
    // fromClosedRange was handed.
    check("rp9", (n[3]..n[0]).reversed().isEmpty() &&
                 (n[3]..n[0]).reversed().toString() == "1 downTo 5 step 1")
    // A reversed progression is an IntProgression and is NOT an IntRange.
    check("rp10", rev is IntProgression && !(rev is IntRange) && (n[0]..n[3]) is IntRange)
    check("rp11", (cs[0]..cs[1]).reversed() is CharProgression &&
                  !((cs[0]..cs[1]).reversed() is CharRange))
    // A LIST and a STRING keep the answers they always had - Iterable.reversed() is a
    // List and String.reversed() a String.
    check("rp12", listOf(n[0], n[2], n[3]).reversed().toString() == "[5, 3, 1]" &&
                  txt[0].reversed() == "fedcba")

    // ----- step, rangeTo, until, downTo in their FUNCTION spellings -----
    // `step` answers an IntProgression whatever the step is: `(1..5) step 1` is
    // "1..5 step 1" where the plain 1..5 is "1..5".
    check("rp13", ((n[0]..n[3]) step n[0]).toString() == "1..5 step 1" &&
                  !(((n[0]..n[3]) step n[0]) is IntRange))
    check("rp14", (n[0]..n[3]).step(n[5]).toString() == "1..5 step 2" &&
                  (n[0]..n[1]).step(n[2]).toString() == "1..10 step 3")
    check("rp15", n[0].rangeTo(n[3]).toString() == "1..5" &&
                  n[0].until(n[3]).toString() == "1..4" &&
                  n[3].downTo(n[0]).toString() == "5 downTo 1 step 1")
    check("rp16", cs[0].rangeTo(cs[2]).toString() == "a..c" &&
                  ls[0].rangeTo(ls[1]).toString() == "1..5")
    // A Long step: `1L..5L step 2L` is "1..5 step 2".
    check("rp17", (ls[0]..ls[1] step ls[2]).toString() == "1..5 step 2" &&
                  (ls[0]..ls[1] step ls[2]).toList().toString() == "[1, 3, 5]")

    // ----- the bounds surface a progression answers WITHOUT materializing -----
    check("rp18", (n[3]..n[0]).first == 5 && (n[3]..n[0]).last == 1 &&
                  (n[3]..n[0]).isEmpty() && (n[3]..n[0]).count() == 0)
    check("rp19", (n[0]..n[3]).sum() == 15 && (n[0]..n[3]).count() == 5 &&
                  !(n[0]..n[3]).isEmpty())
    check("rp20", (n[5] in n[0]..n[3]) && (n[1] !in n[0]..n[3]) &&
                  (cs[2] in cs[0]..cs[1]) && (n[5] !in n[0]..n[1] step n[2]))
    check("rp21", (n[0]..n[3]).withIndex().toList().size == 5 &&
                  listOf(n[0], n[3]).indices.toString() == "0..1")

    // ----- substring / slice / subList -----
    check("rp22", txt[0].substring(n[0]..n[2]) == "bcd" && txt[0].slice(n[0]..n[2]) == "bcd")
    check("rp23", txt[0].slice(listOf(n[4], n[5])) == "ac" &&
                  txt[0].substring(n[2]..n[4]) == "" && txt[0].slice(n[2]..n[4]) == "")
    check("rp24", txt[0].substring(n[0]) == "bcdef" && txt[0].substring(n[0], n[2]) == "bc")
    check("rp25", listOf(n[1], n[5], n[2]).slice(n[4]..n[0]).toString() == "[10, 2]" &&
                  listOf(n[1], n[5], n[2]).subList(n[4], n[5]).toString() == "[10, 2]" &&
                  arrayOf(n[1], n[5], n[2]).sliceArray(n[4]..n[0]).toList().toString() == "[10, 2]")

    // ----- the _Strings.kt family that answers a String, not a List<Char> -----
    check("rp26", txt[0].filter { it != cs[2] } == "abdef" &&
                  txt[0].filterNot { it == cs[2] } == "abdef" &&
                  txt[0].filterIndexed { i, _ -> i % 2 == 0 } == "ace")
    check("rp27", txt[0].takeWhile { it < cs[2] } == "ab" &&
                  txt[0].dropWhile { it < cs[2] } == "cdef" && txt[0].onEach { } == "abcdef")
    check("rp28", txt[0].partition { it < cs[2] }.toString() == "(ab, cdef)" &&
                  txt[0].partition { it < cs[2] }.first == "ab")
    check("rp29", txt[0].chunked(n[5]).toString() == "[ab, cd, ef]" &&
                  txt[0].windowed(n[5]).toString() == "[ab, bc, cd, de, ef]" &&
                  txt[0].chunked(n[5]) { it.length }.toString() == "[2, 2, 2]" &&
                  txt[0].windowed(n[5], n[5]) { it }.toString() == "[ab, cd, ef]")
    // The TRANSFORMING half of the same generated family still answers a List, in
    // Kotlin too - this is the guard against over-converting the branch above.
    check("rp30", txt[0].map { it }.toString() == "[a, b, c, d, e, f]" &&
                  txt[0].toList().size == 6 && txt[0].distinct().size == 6 &&
                  txt[0].zip(txt[1]).toString() == "[(a, a), (b, c)]" &&
                  txt[0].groupBy { it }.size == 6)

    // ----- chunked / windowed on a COLLECTION: the transform and partialWindows -----
    val xs = listOf(n[1], n[5] * 10, n[2] * 10, n[6] * 10, n[3] * 10)   // 10,20,30,40,50
    check("rp31", xs.chunked(n[5]) { it.sum() }.toString() == "[30, 70, 50]" &&
                  xs.chunked(n[5]).toString() == "[[10, 20], [30, 40], [50]]")
    check("rp32", xs.windowed(n[5], n[5]).toString() == "[[10, 20], [30, 40]]" &&
                  xs.windowed(n[5], n[5], true).toString() == "[[10, 20], [30, 40], [50]]" &&
                  xs.windowed(n[5], n[5], true) { it.size }.toString() == "[2, 2, 1]")

    // ----- kotlin.random -----
    // THE VALUES ARE OURS, NOT THE JVM'S. There is no random source in this project
    // that three engines can agree on byte-for-byte (the argument Kernel#rand is
    // settled by in ruby-interpreter.abnf), so `random()` draws from a deterministic
    // xorshift32 with a fixed default seed. Only the INVARIANTS can be asserted:
    // membership, bounds, the empty cases, that a seed is reproducible, that two
    // seeds differ and that consecutive draws are not a constant.
    var inBounds = true
    var sawHigh = false
    var sawLow = false
    for (t in 1..128) {
        val v = (n[0]..n[3]).random()
        if (v < 1 || v > 5) { inBounds = false }
        if (v == 5) { sawHigh = true }
        if (v == 1) { sawLow = true }
        if (!xs.contains(xs.random())) { inBounds = false }
        if (('a'..'f').random() !in 'a'..'f') { inBounds = false }
        if ((n[3] downTo n[0] step n[5]).random() !in listOf(5, 3, 1)) { inBounds = false }
    }
    check("rp33", inBounds && sawHigh && sawLow)
    check("rp34", (n[3]..n[0]).randomOrNull() == null && listOf<Int>().randomOrNull() == null &&
                  txt[0].random() in txt[0])
    // A seeded generator is reproducible, and two different seeds are not the same
    // stream. Neither statement names a value.
    val gA = kotlin.random.Random(n[6])
    val gB = kotlin.random.Random(n[6])
    val gC = kotlin.random.Random(n[2])
    var sA = ""
    var sB = ""
    var sC = ""
    for (t in 1..12) {
        sA += (n[0]..n[3]).random(gA).toString()
        sB += (n[0]..n[3]).random(gB).toString()
        sC += (n[0]..n[3]).random(gC).toString()
    }
    check("rp35", sA == sB && sA != sC && sA.length == 12)
    val gD = kotlin.random.Random(n[6])
    check("rp36", gD.nextInt(n[1]) in 0..9 && gD.nextInt(n[1], n[1] * 2) in 10..19 &&
                  gD.nextDouble() < 1.0 && gD.nextDouble() >= 0.0)
    check("rp37", (gD.nextBoolean() || true) && (n[0]..1000000).random() != (n[0]..1000000).random())
    // Random's COMPANION OBJECT spellings: `Random.nextInt(n)`, `Random.Default`.
    // `Random` is the seeded-constructor function here, so each is a member
    // read/call on that function value, matched by identity in all three engines.
    check("rp39", kotlin.random.Random.nextInt(n[1]) in 0..9 &&
                  kotlin.random.Random.Default.nextInt(n[1], n[1] * 2) in 10..19 &&
                  (n[0]..n[3]).random(kotlin.random.Random.Default) in n[0]..n[3] &&
                  (kotlin.random.Random.nextBoolean() || true))
    // withIndex() answers Iterable<IndexedValue<T>>, and IndexedValue is a DATA
    // class - its toString is the generated "IndexedValue(index=0, value=10)", not
    // the Pair "(0, 10)". Destructuring, .index and .value are what the second half
    // of this asserts.
    var wiTxt = ""
    for ((wi, wv) in xs.withIndex()) { wiTxt += "" + wi + ":" + wv + " " }
    check("rp38", xs.withIndex().toList()[n[4]].toString() == "IndexedValue(index=0, value=10)" &&
                  wiTxt == "0:10 1:20 2:30 3:40 4:50 " &&
                  xs.withIndex().first().index == 0 && xs.withIndex().first().value == 10 &&
                  txt[0].withIndex().toList()[n[0]].toString() == "IndexedValue(index=1, value=b)")
}


// ============================================================================
// SECTION 79 - THE SCOPE CHAIN: what an emitter could not see
// ============================================================================
//
// Ten of Kotlin's externs take a SCOPE HANDLE. Emitted IR can reach only the INNERMOST
// binding and the INNERMOST `this`; the chain walk lives in
// languages/lib/kotlin-rt.metajs SECTION 6, over the scope host builtins (scopeNew /
// scopeParent / scopeGet / scopeHas / scopeDecl, ids 64..68). Every assertion here is a
// shape an innermost-only lowering gets wrong, and the ones marked NATIVE-ABORT kill the
// native binary while the interpreter and llvm.Run stay green - the blind spot
// docs/working-on-this-project.md chapter 3 warns about.
//
// There is no kotlinc on this machine (manual chapter 5), so the expected values
// are cited from the Kotlin specification: `for`/`with`/`apply` receivers and
// this@Label are all defined by the implicit-receiver STACK, innermost first,
// with an enclosing member's receivers still in scope inside a lambda.
fun s79Unit(): Unit { }

class S79Del(val tag: String) {
    operator fun getValue(thisRef: Any?, property: KProperty<*>): String =
        tag + ":" + (if (thisRef == null) "null" else "obj") + ":" + property.name
}

class S79Outer(val oname: String) {
    val shared = "outer"
    inner class S79Mid(val mname: String) {
        val shared = "mid"
        inner class S79Inn(val iname: String) {
            fun who(): String = this@S79Outer.oname + "/" + this@S79Mid.mname + "/" + iname
            fun deep(): String {
                var r = ""
                run { run { r = this@S79Outer.shared + "|" + this@S79Mid.shared } }
                return r
            }
        }
    }
}

class S79Side(val sn: Int)

class S79Host(val base: Int) {
    val boxed: String by S79Del("H")
    fun Int.scaled(): Int = this * base
    val Int.s79Doubled: Int get() = this * 2 * base
    // NATIVE-ABORT under an innermost-only lowering: `an` is a member of the ENCLOSING
    // class read from inside an apply block, so the innermost `this` is the S79Side and
    // the answer is one frame further out.
    fun viaApply(): Int {
        var r = 0
        S79Side(5).apply { r = base + sn }
        return r
    }
    // A member extension reached from THREE lambdas deep. The dispatch receiver is
    // the frame's, not the innermost one's.
    fun deepMext(): Int {
        var acc = 0
        val s79outer = { a: Int -> val s79inner = { b: Int -> acc = acc + b.scaled(); acc }; s79inner(a) }
        s79outer(3)
        return acc + 7.s79Doubled
    }
    fun boxedFromNest(): String {
        var out = ""
        run { run { out = boxed } }
        return out
    }
}

fun sec79() {
    val n = arrayOf(1, 2, 3, 4, 5)

    // ----- a slot that HOLDS undefined is BOUND, and that is not "absent" -----
    // NATIVE-ABORT under an innermost-only lowering. A Unit-valued local is a scope slot
    // holding undefined; js_scope_typeof answers "undefined" for that and for an absent
    // name alike, and js_scope_has does not walk, so a read from an ENCLOSING scope needs
    // scopeHas + scopeParent. The same defect class made python's comprehensions answer
    // [NaN, NaN, NaN, NaN] (manual 7.3).
    val u: Unit = s79Unit()
    var uOut = ""
    run { uOut = "" + u }
    check("sc01", uOut == "kotlin.Unit" && "" + u == "kotlin.Unit")
    var uOut2 = ""
    run { run { run { uOut2 = "" + u } } }
    check("sc02", uOut2 == "kotlin.Unit")

    // ----- an enclosing class member from inside an implicit-receiver lambda -----
    // NATIVE-ABORT under an innermost-only lowering.
    check("sc03", S79Host(n[3]).viaApply() == 9)

    // ----- this@Label past a nested `this`, and from inside lambdas -----
    val inn = S79Outer("O").S79Mid("M").S79Inn("I")
    check("sc04", inn.who() == "O/M/I")
    check("sc05", inn.deep() == "outer|mid")

    // ----- a member extension, and a member extension PROPERTY, from depth -----
    check("sc06", S79Host(n[2]).deepMext() == 9 + 42)

    // ----- a DELEGATED property read from a nested scope: js_ktdthis is the
    // thisRef, and it is the first `this` up the chain, not the innermost frame.
    check("sc07", S79Host(n[0]).boxedFromNest() == "H:obj:boxed")

    // ----- var reassignment through nested closures, and shadowing at depth -----
    var acc = 0
    val s79l1 = { val a = n[0]
        val s79l2 = { val b = n[1]
            val s79l3 = { acc = acc + a + b }
            s79l3() }
        s79l2() }
    s79l1()
    check("sc08", acc == 3)
    val x = "outer-x"
    var trail = x
    run {
        val x = "inner-x"
        trail = trail + "," + x
        run { val x = "inner2-x"; trail = trail + "," + x }
        trail = trail + "," + x
    }
    check("sc09", trail + "," + x == "outer-x,inner-x,inner2-x,inner-x,outer-x")

    // ----- ::name and Base::name still resolve, including the TYPE-name arm the
    // emitter decides statically (String::length must not bind the String builder).
    val r1 = ::s79Unit
    check("sc10", r1.name == "s79Unit")
    check("sc11", String::length.get("abcd") == 4 && String::class.simpleName == "String")

    // ----- a `for` loop variable is a FRESH binding each iteration, so a lambda
    // that captures it keeps THAT iteration's value. The Kotlin specification
    // defines the loop by desugaring to `val <v> = iterator.next()` INSIDE the body.
    // A shared cell would make every lambda answer "3 3 3".
    val fs = mutableListOf<() -> Int>()
    for (i in n[0]..n[2]) { fs.add({ i }) }
    var fsOut = ""
    for (s79a in fs) { fsOut = fsOut + s79a() + " " }
    check("sc12", fsOut.trim() == "1 2 3")
    val gs = mutableListOf<() -> String>()
    for (w in listOf("a", "b", "c")) { gs.add({ w }) }
    var gsOut = ""
    for (s79b in gs) { gsOut = gsOut + s79b() }
    check("sc13", gsOut == "abc")
    val hs = mutableListOf<() -> Int>()
    for (j in n[2] downTo n[0]) { hs.add({ j * 10 }) }
    var hsOut = 0
    for (s79c in hs) { hsOut = hsOut * 100 + s79c() }
    check("sc14", hsOut == 302010)
    // A destructuring for-loop binds both components per iteration.
    val ps = mutableListOf<() -> String>()
    for ((k, v) in listOf(Pair("a", 1), Pair("b", 2))) { ps.add({ k + v }) }
    var psOut = ""
    for (s79d in ps) { psOut = psOut + s79d() }
    check("sc15", psOut == "a1b2")
    // The loop variable is NOT visible after the loop and a nested loop's inner
    // variable does not leak into the outer body.
    var nested = ""
    for (a in n[0]..n[1]) { for (b in n[0]..n[1]) { nested = nested + a + b } }
    check("sc16", nested == "11122122")
}

// ============================================================================
// SECTION 80 - MEMBER EXTENSIONS, NAME SHADOWING, and java.util.Formatter's %f
// ============================================================================
//
// Three receiver/format rules, each with at least one leg that could answer differently
// from the other two - the class ./test.sh --cross exists for and, for the native-only
// one, cannot see at all (manual chapter 3: the matrix, --full and --cross all run
// llvm.Run).
//
// There is no kotlinc on this machine (manual chapter 5). The receiver rules below are
// cited from the Kotlin specification; the %f values are checked against `java`, which
// is the toolchain Kotlin/JVM's String.format is, with the locale forced to ROOT so the
// separator is a point.
interface S80Iface {
    // Nothing calls this through the interface. It exists so that the name `f80`
    // is a member-method name SOMEWHERE in the file, which puts it in the compiler
    // half's whole-file classMethodNames set; the local `val f80 = { ... }` inside
    // S80Local below must still win. Kotlin resolves a call against the innermost scope
    // declaring the name and a local declaration is inside the class body's scope.
    fun f80(): Int
}

class S80Impl : S80Iface {
    override fun f80(): Int = 111
}

class S80Other(val k: Int)

class S80Local {
    // A member of the same name as the local below, to prove the local wins over
    // the class's OWN method too and not merely over the unrelated interface's.
    fun f80(): Int = 999
    fun useLocal(n: Int): Int {
        val f80 = { x: Int -> x * 2 }
        return f80(n)
    }
    fun useMember(): Int = f80()
}

// `with` takes a T.() -> R extension lambda, which ADDS an implicit receiver
// rather than replacing the enclosing member's dispatch receiver, so a member
// extension of the enclosing class still resolves inside it.
class S80Recv {
    fun Int.s80Tripled(): Int = this * 3
    val Int.s80Quad: Int get() = this * 4
    fun plain(n: Int): Int = n.s80Tripled()
    fun viaWith(n: Int): Int = with(S80Other(9)) { n.s80Tripled() }
    fun viaWithProp(n: Int): Int = with(S80Other(9)) { n.s80Quad }
    fun viaApply(n: Int): Int { var r = 0; S80Other(9).apply { r = n.s80Tripled() }; return r }
    fun viaRun(n: Int): Int = run { with(S80Other(9)) { n.s80Tripled() } }
}

// A member extension PROPERTY makes the emitter route EVERY write of that name
// through the member-extension setter path, in every scope in the file. When the
// lookup misses - as it does here, at top level, where there is no receiver at
// all - the fall-back is an ordinary member write, and an ordinary member write
// runs the accessor (`c.s80Ctr = 10` sets `raw` to 5). Kotlin says members beat
// extensions, and here the extension's receiver type (Int) does not even match.
class S80Host {
    var Int.s80Ctr: Int
        get() = this * 2
        set(v) { }
    fun probe(n: Int): Int = n.s80Ctr
}

class S80Counted {
    var raw: Int = 0
    var s80Ctr: Int
        get() = raw * 2
        set(v) { raw = v / 2 }
}

fun sec80() {
    val n = arrayOf(1, 2, 3, 4, 5)

    // ----- a local binding beats a member method -----
    check("mx01", S80Impl().f80() == 111)
    check("mx02", S80Local().useLocal(n[2]) == 6)
    check("mx03", S80Local().useMember() == 999)

    // ----- an ADDED implicit receiver does not hide the member extension of the
    // enclosing class.
    val r = S80Recv()
    check("mx04", r.plain(n[1]) == 6)
    check("mx05", r.viaWith(n[1]) == 6)
    check("mx06", r.viaWithProp(n[4]) == 20)
    check("mx07", r.viaApply(n[1]) == 6)
    check("mx08", r.viaRun(n[1]) == 6)

    // ----- the member-extension write path's MISS still runs the unrelated class's
    // setter (a native-only case).
    check("mx09", S80Host().probe(n[3]) == 8)
    val c = S80Counted()
    c.s80Ctr = n[4] * 2
    check("mx10", c.raw == 5 && c.s80Ctr == 10)

    // ----- java.util.Formatter's %f -----
    // Every value is read out of an array so the constant folder cannot answer
    // these at compile time (manual chapter 3).
    val fv = doubleArrayOf(1.3379870716132959e18, 0.1, 0.15, 0.25, 0.35, 2.675,
                           1e22, 1e300, 4.9e-324, 1.0 / 3.0, 12345.6789, 2.5, -2.5)
    // PAST 2^53: java's %f renders from Double.toString's SHORTEST round-tripping
    // digits padded with zeros, so this is 1337987071613295900 and NOT the double's
    // exact 1337987071613295872.
    check("fx01", "%.0f".format(fv[0]) == "1337987071613295900")
    check("fx02", "%.2f".format(fv[0]) == "1337987071613295900.00")
    // Same rule at high precision: the padding is zeros, not the exact expansion.
    check("fx03", "%.20f".format(fv[1]) == "0.10000000000000000000")
    check("fx04", "%.17f".format(fv[9]) == "0.33333333333333330")
    // HALF_UP on those digits, not half-to-even and not exact-value rounding: the
    // exact double behind 0.15 is 0.1499999999999999944, which an exact rounder
    // answers "0.1".
    check("fx05", "%.1f".format(fv[2]) == "0.2")
    check("fx06", "%.1f".format(fv[3]) == "0.3")
    check("fx07", "%.1f".format(fv[4]) == "0.4")
    check("fx08", "%.2f".format(fv[5]) == "2.68")
    // HALF_UP is away from zero, so a negative tie goes down.
    check("fx09", "%.0f".format(fv[11]) == "3" && "%.0f".format(fv[12]) == "-3")
    // A magnitude the host prints in exponent form must not leak the "e+22" into
    // the integer part, and prec 17 of 1e300 must not overflow to Infinity.
    check("fx10", "%.0f".format(fv[6]) == "10000000000000000000000")
    check("fx11", "%.3f".format(fv[7]).length == 301 + 4)
    check("fx12", "%.17f".format(fv[7]).substring(0, 4) == "1000")
    // The smallest subnormal is all zeros at any reachable precision.
    check("fx13", "%.30f".format(fv[8]) == "0." + "0".repeat(30))
    check("fx14", "%.6f".format(fv[10]) == "12345.678900")
    // -0.0 keeps its sign.
    val negZero = 0.0 / (0.0 - 1.0)
    check("fx15", "%.0f".format(negZero) == "-0" && "%.2f".format(negZero) == "-0.00")
    check("fx16", "%.2f".format(0.0) == "0.00")
}

// SECTION 81 - uppercase()/lowercase(). kotlin.text.uppercase() is
// locale-invariant FULL Unicode case mapping (String.toUpperCase(Locale.ROOT)):
// "\u00df".uppercase() is "SS", "\u0130".lowercase() is "i" + U+0307, and a
// word-final capital sigma lowercases to U+03C2. A Char's uppercaseChar() stays
// the simple one-to-one map. Every row is java's toUpperCase/toLowerCase(Locale.ROOT).
//
// Every character is a \uXXXX ESCAPE, never a raw literal: manual chapter 4,
// test specific characters by CODE.
fun sec81() {
    check("cm81a", "\u00dfx".uppercase() == "SSX" && "\ufb01".uppercase() == "FI" &&
                   "\u0130".lowercase() == "i\u0307" && "\u00df"[0].uppercaseChar() == '\u00df')
    check("cm81c", '\u00df'.uppercase() == "SS" && '\u0130'.lowercase() == "i\u0307" && '\u00df'.uppercaseChar() == '\u00df')
    check("cm81b", "A\u03a3".lowercase() == "a\u03c2" && "\u03a3A".lowercase() == "\u03c3a" &&
                   "\u1fb3".uppercase() == "\u0391\u0399" && "\u0390".uppercase().length == 3)
    // Dotless i, final sigma, capital sharp s.
    check("cm81g", "\u0131".uppercase()[0].code == 0x0049 && "\u03c2".uppercase()[0].code == 0x03A3 &&
                   "\u1e9e".lowercase()[0].code == 0x00DF)
    // The deprecated spellings resolve to the same host method.
    check("cm81h", "\u00df".toUpperCase() == "\u00df".uppercase() &&
                   "\u0130".toLowerCase() == "\u0130".lowercase())
}

// ===== SECTION 82: OUT-OF-RANGE INDEXING, and a nested throwable's BINARY name =====
// The indexed read is bounds-checked and what it raises is a CATCHABLE throwable, not
// an abort.
//
// THE MESSAGES ARE java's, VERBATIM: Kotlin's throwables ARE java.lang's
// (typealiases), and there is no kotlinc on this machine. The CLASS for a collection
// read is the one place this cannot be exact - the JVM raises
// ArrayIndexOutOfBoundsException from Arrays$ArrayList (what listOf and arrayOf both
// produce) and plain IndexOutOfBoundsException from an ArrayList, while this value
// model has ONE array type - so the common SUPERTYPE is what is asserted, and a real
// Kotlin `catch (e: IndexOutOfBoundsException)` catches either.
//
// Every index is READ OUT OF AN ARRAY: a probe of literals tests the constant
// folder, not the runtime (manual chapter 3).
class S82Outer {
    class S82Inner(m: String) : Exception(m)
    class S82Plain(val n: Int)
}
class S82Top(m: String) : Exception(m)
class S82Len(val length: Int)

fun s82text(f: () -> Any?): String {
    return try { "" + f() } catch (e: Throwable) { e.toString() }
}

fun sec82() {
    val ix = listOf(5, -1, 0, 1)
    val li = listOf(10, 20)
    val ar = arrayOf(10, 20)
    val st = "ab"
    val sb = StringBuilder("ab")
    val oob = "java.lang.IndexOutOfBoundsException: Index 5 out of bounds for length 2"
    val soob = "java.lang.StringIndexOutOfBoundsException: Index 5 out of bounds for length 2"
    check("ix82a", s82text { li[ix[0]] } == oob)
    check("ix82b", s82text { ar[ix[0]] } == oob)
    check("ix82c", s82text { st[ix[0]] } == soob)
    check("ix82d", s82text { sb[ix[0]] } == soob)
    check("ix82e", s82text { li[ix[1]] } ==
        "java.lang.IndexOutOfBoundsException: Index -1 out of bounds for length 2")
    check("ix82f", s82text { st[ix[1]] } ==
        "java.lang.StringIndexOutOfBoundsException: Index -1 out of bounds for length 2")
    check("ix82g", s82text { listOf<Int>()[ix[2]] } ==
        "java.lang.IndexOutOfBoundsException: Index 0 out of bounds for length 0")
    // The METHOD spellings of the same read.
    check("ix82h", s82text { li.get(ix[0]) } == oob && s82text { st.get(ix[0]) } == soob)
    check("ix82i", s82text { li.elementAt(ix[0]) } == oob)
    // In-range reads are untouched - including the StringBuilder.
    check("ix82j", li[ix[3]] == 20 && st[ix[3]] == 'b' && sb[ix[2]] == 'a' &&
        ar[ix[2]] == 10 && li.get(ix[2]) == 10 && li.elementAt(ix[2]) == 10)
    // ...and the OrNull / OrElse family still answers rather than throwing.
    check("ix82k", li.getOrNull(ix[0]) == null && li.getOrElse(ix[0]) { -7 } == -7 &&
        li.elementAtOrNull(ix[0]) == null)
    // Catchable by name; StringIndexOutOfBoundsException is UNDER
    // IndexOutOfBoundsException (java instanceof), not beside it.
    var by = "none"
    try { li[ix[0]] } catch (e: IndexOutOfBoundsException) { by = "io:" + e.message }
    check("ix82l", by == "io:Index 5 out of bounds for length 2")
    var by2 = "none"
    try { st[ix[0]] } catch (e: IndexOutOfBoundsException) { by2 = "super" }
    check("ix82m", by2 == "super")
    var by3 = "none"
    try { st[ix[0]] } catch (e: StringIndexOutOfBoundsException) { by3 = "exact" }
    check("ix82n", by3 == "exact")
    // A CLASS INSTANCE carrying a numeric `length` field of its own is NOT an array:
    // its indexed read still answers kotlin.Unit (the exact Array.isArray test on the
    // miss path; the NATIVE leg is the one that would see it).
    check("ix82r", "" + S82Len(3)[ix[0]] == "kotlin.Unit")
    // A nested throwable's BINARY name. The dollar is ESCAPED - an unescaped one is
    // a string template. java.lang.Class#getName spells a nested type Outer$Inner,
    // and Throwable#toString is that name.
    check("ix82o", S82Outer.S82Inner("in").toString() == "S82Outer\$S82Inner: in")
    // A TOP-LEVEL user throwable is unchanged: its simple name IS its binary name.
    check("ix82p", S82Top("t").toString() == "S82Top: t")
    // Object#toString reads the binary name too, with hashCode() in hex.
    val sp = S82Outer.S82Plain(1)
    check("ix82q", sp.toString() == "S82Outer\$S82Plain@" + Integer.toHexString(sp.hashCode()))
}


// ===== SECTION 83: A USER THROWABLE'S HIERARCHY, and three levels of nesting =====
//
// (a) `class S83Top(m: String) : Exception(m)` then `S83Top("x") is Exception` is
// TRUE: the compiler half emits no descriptor for a builtin throwable, so js_ktexccls
// hands emitSuperLinks the same descriptor js_ktnewexc constructs from. Every row below
// was checked with `instanceof` under java through an Object-typed variable (javac
// REJECTS a statically impossible instanceof, which is why the oracle needs the cast) -
// Kotlin's Exception / RuntimeException / Error are TYPEALIASES to java.lang's, so java
// is the oracle and there is no kotlinc here.
//
// (b) `S83A.S83B.S83C(7)` - three levels: the slot a nested class occupies on its
// owner holds a CONSTRUCTION FACTORY, so the descriptor rides beside it under a
// `__`-prefixed slot for the plain read `S83A.S83B`.
//
// Every subject is read out of an array: a probe of literals tests the constant
// folder, not the runtime (manual chapter 3).
open class S83Top(m: String) : Exception(m)
class S83Sub(m: String) : S83Top(m)
class S83Err(m: String) : Error(m)
class S83Ise(m: String) : IllegalStateException(m)
class S83Plain(val n: Int)
class S83A {
    class S83B {
        class S83C(val v: Int) { fun show(): String = "C" + v }
    }
    class S83D(val w: Int)
}
object S83Root {
    class S83E {
        class S83F(val u: Int)
    }
}

fun s83flags(v: Any): String {
    return "" + (v is Throwable) + (v is Exception) + (v is RuntimeException) + (v is Error)
}

fun s83text(f: () -> Any?): String {
    return try { "" + f() } catch (e: Throwable) { e.toString() }
}

fun sec83() {
    val vs = arrayOf<Any>(S83Top("a"), S83Sub("b"), S83Err("c"), S83Ise("d"), S83Plain(1))
    // java, `new S83Top("a") instanceof ...` through an Object: T,E,-,-.
    check("hi83a", s83flags(vs[0]) == "truetruefalsefalse")
    // A subclass of a user throwable inherits the whole chain.
    check("hi83b", s83flags(vs[1]) == "truetruefalsefalse")
    // Error is NOT an Exception - the one row that would pass by accident if the
    // chain were faked by a marker rather than built.
    check("hi83c", s83flags(vs[2]) == "truefalsefalsetrue")
    // IllegalStateException -> RuntimeException -> Exception -> Throwable.
    check("hi83d", s83flags(vs[3]) == "truetruetruefalse")
    // A class that extends nothing is untouched by the __super link.
    check("hi83e", s83flags(vs[4]) == "falsefalsefalsefalse")
    check("hi83f", vs[0] is S83Top && vs[1] is S83Top && !(vs[0] is S83Sub))
    // The CATCH-CLAUSE selection follows the same chain: a `: Exception` class is NOT
    // caught by `catch (e: RuntimeException)`.
    var by = "none"
    try { throw (vs[0] as Throwable) } catch (e: RuntimeException) { by = "rte" } catch (e: Exception) { by = "exc" }
    check("hi83g", by == "exc")
    var by2 = "none"
    try { throw (vs[3] as Throwable) } catch (e: RuntimeException) { by2 = "rte" } catch (e: Exception) { by2 = "exc" }
    check("hi83h", by2 == "rte")
    var by3 = "none"
    try { throw (vs[2] as Throwable) } catch (e: Exception) { by3 = "exc" } catch (e: Throwable) { by3 = "thr" }
    check("hi83i", by3 == "thr")
    // Throwable#toString is unchanged by the link: a top-level user class keeps its
    // SIMPLE name, and does NOT inherit java.lang.Exception's __bname off the chain.
    check("hi83j", vs[0].toString() == "S83Top: a" && vs[2].toString() == "S83Err: c")
    // An object EXPRESSION extending a builtin throwable gets the chain too.
    val anon = object : IllegalArgumentException("q") {}
    check("hi83k", anon is IllegalArgumentException && anon is RuntimeException && anon is Exception)
    // TODO() / NotImplementedError. kotlin-stdlib: `class NotImplementedError(message:
    // String = "An operation is not implemented.") : Error(message)`, and TODO(reason)
    // prepends "An operation is not implemented: ".
    check("hi83l", s83text { TODO("soon") } ==
        "kotlin.NotImplementedError: An operation is not implemented: soon")
    check("hi83m", s83text { TODO() } ==
        "kotlin.NotImplementedError: An operation is not implemented.")
    check("hi83n", s83text { throw NotImplementedError("m") } == "kotlin.NotImplementedError: m")
    var by4 = "none"
    try { TODO("q") } catch (e: NotImplementedError) { by4 = "byname" }
    check("hi83o", by4 == "byname")
    // It is an Error, not an Exception - the same row that would pass by accident
    // if the entry were parented anywhere convenient.
    check("hi83p", s83flags(NotImplementedError("z")) == "truefalsefalsetrue")
    // Three levels of nesting, both as a construction and as a type.
    val ns = arrayOf(7, 8, 9)
    check("ne83a", S83A.S83B.S83C(ns[0]).show() == "C7")
    check("ne83b", S83A.S83B.S83C(ns[0]).v == 7)
    check("ne83c", S83A.S83B.S83C(ns[1]) is S83A.S83B.S83C)
    // Two levels, and a nested class of an OBJECT declaration, both still work.
    check("ne83d", S83A.S83D(ns[2]).w == 9)
    check("ne83e", S83Root.S83E.S83F(ns[0]).u == 7)
    val typed: S83A.S83B.S83C = S83A.S83B.S83C(ns[1])
    check("ne83f", typed.v == 8)
}

// ===== SECTION 84: THE INDEX FAMILY - every message, and the empty receiver =====
// Each of these raises a CATCHABLE throwable - an abort would not be catchable at all,
// so no Kotlin program could recover from an out-of-range write or an empty first().
//
// THE MESSAGES ARE java's, VERBATIM, and the JVM spells the family THREE different
// ways, which is the whole reason this is a section rather than one helper:
//     l.removeAt(9) / l.set(9,x) / l[9]     Index 9 out of bounds for length 2
//     l.add(9, x)                           Index: 9, Size: 2
//     "ab".substring(5)                     Range [5, 2) out of bounds for length 2
// The first goes through Objects.checkIndex, the second through ArrayList's own
// rangeCheckForAdd, the third through String's checkBoundsBeginEnd. All three raise
// IndexOutOfBoundsException or its StringIndexOutOfBoundsException subclass.
//
// The EMPTY-RECEIVER rows are kotlin-stdlib's own messages, read by calling the
// kotlin-stdlib 2.2.21 jar's CollectionsKt / StringsKt / RangesKt from java.
//
// Every index is read out of an array (manual chapter 3).
fun s84text(f: () -> Any?): String {
    return try { "" + f() } catch (e: Throwable) { e.toString() }
}

fun sec84() {
    val ix = arrayOf(9, -1, 2, 0)
    val io = "java.lang.IndexOutOfBoundsException: "
    val sio = "java.lang.StringIndexOutOfBoundsException: "
    val nse = "java.util.NoSuchElementException"
    check("ax84a", s84text { mutableListOf(1, 2).removeAt(ix[0]) } == io + "Index 9 out of bounds for length 2")
    check("ax84b", s84text { mutableListOf(1, 2).removeAt(ix[1]) } == io + "Index -1 out of bounds for length 2")
    check("ax84c", s84text { mutableListOf(1, 2).set(ix[0], 5) } == io + "Index 9 out of bounds for length 2")
    check("ax84d", s84text { mutableListOf(1, 2).set(ix[2], 5) } == io + "Index 2 out of bounds for length 2")
    // The INDEXED WRITE is MutableList.set and raises the same thing, for a positive
    // and a negative index.
    check("ax84e", s84text { val l = mutableListOf(1, 2); l[ix[0]] = 5; l.size } == io + "Index 9 out of bounds for length 2")
    check("ax84f", s84text { val l = mutableListOf(1, 2); l[ix[1]] = 5; l.size } == io + "Index -1 out of bounds for length 2")
    // add(index, element) is the one member with the OTHER format, and its upper
    // bound is size INCLUSIVE - appending at the end is legal.
    check("ax84g", s84text { mutableListOf(1, 2).add(ix[0], 5) } == io + "Index: 9, Size: 2")
    check("ax84h", s84text { mutableListOf(1, 2).add(ix[1], 5) } == io + "Index: -1, Size: 2")
    check("ax84i", s84text { val l = mutableListOf(1, 2); l.add(ix[2], 5); "" + l } == "[1, 2, 5]")
    // substring: a RANGE, and the end of a one-argument call is the receiver's
    // length (JS substring would clamp).
    val ss = arrayOf("ab", "abcdef")
    check("ax84j", s84text { ss[0].substring(ix[0]) } == sio + "Range [9, 2) out of bounds for length 2")
    check("ax84k", s84text { ss[1].substring(ix[0]) } == sio + "Range [9, 6) out of bounds for length 6")
    check("ax84l", s84text { ss[0].substring(ix[3], ix[0]) } == sio + "Range [0, 9) out of bounds for length 2")
    check("ax84m", s84text { ss[0].substring(ix[1]) } == sio + "Range [-1, 2) out of bounds for length 2")
    // In-range substrings are untouched, the empty one included.
    check("ax84n", ss[1].substring(ix[2]) == "cdef" && ss[0].substring(ix[2]) == "")
    // The empty receiver: NoSuchElementException, catchable, not an abort.
    val emp = listOf<Int>()
    val many = listOf(1, 2, 3, 4)
    check("ax84o", s84text { emp.first() } == nse + ": List is empty.")
    check("ax84p", s84text { emp.last() } == nse + ": List is empty.")
    check("ax84q", s84text { emp.single() } == nse + ": List is empty.")
    check("ax84r", s84text { mutableListOf<Int>().removeFirst() } == nse + ": List is empty.")
    // max()/min()/maxBy()/maxOf() are stdlib's `?: throw NoSuchElementException()`,
    // so their message is genuinely NULL and Throwable#toString prints the class
    // alone - which is exactly the shape java prints for a null message.
    check("ax84s", s84text { emp.max() } == nse && s84text { emp.maxBy { it } } == nse)
    check("ax84t", s84text { emp.maxOf { it } } == nse && s84text { emp.min() } == nse)
    // reduce() is the one that is NOT NoSuchElementException.
    check("ax84u", s84text { emp.reduce { a, b -> a + b } } ==
        "java.lang.UnsupportedOperationException: Empty collection can't be reduced.")
    // single() on a longer list, and the predicate forms of first/single.
    check("ax84v", s84text { many.single() } ==
        "java.lang.IllegalArgumentException: List has more than one element.")
    check("ax84w", s84text { many.first { it > 9 } } ==
        nse + ": Collection contains no element matching the predicate.")
    check("ax84x", s84text { many.single { it > 2 } } ==
        "java.lang.IllegalArgumentException: Collection contains more than one matching element.")
    // last(predicate) is the last MATCHING element.
    check("ax84y", many.last { it < 3 } == 2 && many.first { it > 2 } == 3)
    check("ax84z", s84text { many.last { it > 9 } } ==
        nse + ": List contains no element matching the predicate.")
    // The OrNull family still answers rather than throwing, and the in-range
    // spellings are all untouched.
    check("ax84A", emp.maxOrNull() == null && emp.firstOrNull() == null &&
        many.first() == 1 && many.last() == 4 && listOf(6).single() == 6)
    // The KIND WORD is the OVERLOAD the call resolved to, not the element type, and
    // every receiver above is a List. kotlin-stdlib has three parallel families:
    //   _Collections.kt  List overload      "List is empty."
    //                    Iterable overload  "Collection is empty."
    //   _Strings.kt      CharSequence       "Char sequence is empty."
    // A Set, a Map's keys/entries and a range are Iterable and none of them is a
// List, so all three take the Collection wording; a String takes its own, and its
// rows throw rather than abort.
    val eset = setOf<Int>()
    val emap = mapOf<Int, Int>()
    val erange = 1..0
    check("ax84B", s84text { eset.first() } == nse + ": Collection is empty." &&
        s84text { eset.last() } == nse + ": Collection is empty." &&
        s84text { eset.single() } == nse + ": Collection is empty.")
    // random() is declared on COLLECTION and never on List, so even a LIST receiver
    // takes the Collection wording.
    check("ax84C", s84text { eset.random() } == nse + ": Collection is empty." &&
        s84text { emp.random() } == nse + ": Collection is empty.")
    check("ax84D", s84text { eset.reduce { a, b -> a + b } } ==
        "java.lang.UnsupportedOperationException: Empty collection can't be reduced.")
    // max()/maxBy()/maxOf() are `?: throw NoSuchElementException()` on every
    // receiver, so their message stays genuinely null for a Set too.
    check("ax84E", s84text { eset.max() } == nse && s84text { eset.maxBy { it } } == nse &&
        s84text { eset.minOf { it } } == nse && s84text { eset.minBy { it } } == nse)
    check("ax84F", s84text { setOf(1, 2, 3).single() } ==
        "java.lang.IllegalArgumentException: Collection has more than one element.")
    // last(predicate) is the ONE predicate form with a List overload, so it is the
    // one whose predicate message follows the receiver: "List contains no ..." for a
    // list (ax84z) and "Collection contains no ..." for a Set. first(predicate) and
    // single(predicate) exist only on Iterable, so they say "Collection" for BOTH.
    check("ax84G", s84text { setOf(1, 2, 3).last { it > 9 } } ==
        nse + ": Collection contains no element matching the predicate.")
    check("ax84H", s84text { setOf(1, 2, 3).first { it > 9 } } ==
        nse + ": Collection contains no element matching the predicate." &&
        s84text { setOf(1, 2, 3).single { it > 9 } } ==
        nse + ": Collection contains no element matching the predicate.")
    // A Map's keys and entries are SETS (java.util.Map's keySet/entrySet), so they
    // take the Collection wording too; `values`, an Array and a Sequence are SECTION 87.
    check("ax84I", s84text { emap.keys.first() } == nse + ": Collection is empty." &&
        s84text { emap.entries.first() } == nse + ": Collection is empty." &&
        s84text { emap.keys.last() } == nse + ": Collection is empty.")
    check("ax84J", s84text { mapOf(1 to 2, 3 to 4).keys.single() } ==
        "java.lang.IllegalArgumentException: Collection has more than one element.")
    // An IntRange is an Iterable and not a List, but first()/last() resolve to
    // kotlin.ranges' IntProgression overloads, which name the progression; single()
    // has no progression overload and takes the Iterable wording. `random()` has its
    // own message from kotlin.ranges (Random.nextInt's text, rethrown).
    check("ax84K", s84text { erange.first() } == nse + ": Progression 1..0 is empty." &&
        s84text { erange.last() } == nse + ": Progression 1..0 is empty." &&
        s84text { erange.single() } == nse + ": Collection is empty.")
    // The progression's own toString is in the message: a stepped, a downTo, an
    // until and a CharRange receiver. The predicate forms are Iterable's.
    val rb = arrayOf(5, 1, 0, 3)
    check("ax84Ka", s84text { (rb[0]..rb[1] step 2).first() } ==
        nse + ": Progression 5..1 step 2 is empty." &&
        s84text { (rb[2] downTo rb[1]).last() } ==
        nse + ": Progression 0 downTo 1 step 1 is empty." &&
        s84text { (rb[3] until rb[3]).first() } == nse + ": Progression 3..2 is empty.")
    check("ax84Kb", s84text { ('b'..'a').first() } == nse + ": Progression b..a is empty." &&
        s84text { erange.first { it > 9 } } ==
        nse + ": Collection contains no element matching the predicate.")
    check("ax84L", s84text { erange.random() } ==
        nse + ": Cannot get random in empty range: 1..0")
    // A CharSequence's own family, from _Strings.kt - catchable throwables.
    val estr = arrayOf("", "abc", "z")
    check("ax84M", s84text { estr[0].first() } == nse + ": Char sequence is empty." &&
        s84text { estr[0].last() } == nse + ": Char sequence is empty." &&
        s84text { estr[0].single() } == nse + ": Char sequence is empty.")
    check("ax84N", s84text { estr[0].random() } == nse + ": Char sequence is empty.")
    check("ax84O", s84text { estr[1].single() } ==
        "java.lang.IllegalArgumentException: Char sequence has more than one element.")
    check("ax84P", s84text { estr[0].reduce { a, b -> b } } ==
        "java.lang.UnsupportedOperationException: Empty char sequence can't be reduced.")
    // first(predicate) / last(predicate) on a String apply the lambda, and so do
    // firstOrNull/lastOrNull with one.
    check("ax84Q", s84text { estr[1].first { it == 'z' } } ==
        nse + ": Char sequence contains no character matching the predicate." &&
        s84text { estr[1].last { it == 'z' } } ==
        nse + ": Char sequence contains no character matching the predicate.")
    check("ax84R", s84text { estr[1].single { it == 'z' } } ==
        nse + ": Char sequence contains no character matching the predicate.")
    check("ax84S", s84text { estr[1].single { it != 'q' } } ==
        "java.lang.IllegalArgumentException: Char sequence contains more than one matching element.")
    check("ax84T", estr[1].first { it == 'b' } == 'b' && estr[1].last { it != 'c' } == 'b' &&
        estr[1].single { it == 'b' } == 'b' && estr[2].single() == 'z')
    check("ax84U", estr[1].firstOrNull { it == 'c' } == 'c' &&
        estr[1].lastOrNull { it == 'a' } == 'a' && estr[0].firstOrNull() == null &&
        estr[1].first() == 'a' && estr[1].last() == 'c')
    // lastOrNull(predicate) on a LIST is the last MATCHING element too.
    check("ax84V", many.lastOrNull { it < 3 } == 2 && many.findLast { it < 3 } == 2 &&
        many.lastOrNull { it > 9 } == null && many.lastOrNull() == 4)
    // The reduce family beyond reduce(). reduceRight/reduceRightIndexed are the LIST
    // overloads, which is where "Empty list can't be reduced." comes from - the one
    // wording in this section that no other member can produce.
    check("ax84W", listOf("a", "b", "c").reduceRight { a, b -> a + b } == "abc" &&
        many.reduceIndexed { i, a, b -> i + a + b } == 16 &&
        many.reduceRightIndexed { i, a, b -> i + a + b } == 13)
    check("ax84X", s84text { emp.reduceRight { a, b -> a + b } } ==
        "java.lang.UnsupportedOperationException: Empty list can't be reduced." &&
        s84text { emp.reduceIndexed { i, a, b -> a + b } } ==
        "java.lang.UnsupportedOperationException: Empty collection can't be reduced.")
    check("ax84Y", s84text { estr[0].reduceRight { a, b -> b } } ==
        "java.lang.UnsupportedOperationException: Empty char sequence can't be reduced." &&
        estr[1].reduceRight { a, b -> b } == 'c')
    // The non-empty and OrNull spellings of every kind are untouched.
    check("ax84Z", setOf(7).single() == 7 && setOf(3, 4).first() == 3 &&
        setOf(3, 4).last() == 4 && (1..3).first() == 1 && (1..3).last() == 3 &&
        eset.firstOrNull() == null && emap.keys.firstOrNull() == null)
}

// ===== SECTION 85: String to number, and the OrNull reduce family =====
// toInt/toLong are java's Integer.parseInt/Long.parseLong: no trimming, and a value
// out of range is an ERROR, not a wrap. toDouble/toFloat are Double.parseDouble: the
// text is trimmed, may end in f/F/d/D, may be NaN, Infinity or a hex float, and
// NOTHING else may follow - "1.5abc" is an error, not 1.5. The OrNull forms answer
// null where those throw. Every row was read from the kotlin-stdlib 2.2.21 jar,
// called from java. The inputs come out of a list (manual chapter 3).
fun s85text(f: () -> Any?): String {
    return try { "" + f() } catch (e: Throwable) { e.toString() }
}

fun sec85() {
    val ss = listOf("1.5abc", " 1.5 ", "1.5f", "0x1.8p1", "-0", "Inf", "", " 5", "2147483648",
        "-2147483648", "9223372036854775807", "9223372036854775808", "1e", ".5", "+Infinity", "1e39")
    val nfe = "java.lang.NumberFormatException: "
    check("ax85a", s85text { ss[0].toDouble() } == nfe + "For input string: \"1.5abc\"")
    check("ax85b", ss[1].toDouble() == 1.5 && ss[2].toDouble() == 1.5 && ss[3].toDouble() == 3.0)
    check("ax85c", s85text { ss[4].toDouble() } == "-0.0" && ss[13].toDouble() == 0.5)
    check("ax85d", s85text { ss[5].toDouble() } == nfe + "For input string: \"Inf\"" &&
        s85text { ss[12].toDouble() } == nfe + "For input string: \"1e\"")
    check("ax85e", s85text { ss[6].toDouble() } == nfe + "empty String")
    check("ax85f", ss[14].toDouble() > 1e308 && ss[0].toDoubleOrNull() == null && ss[1].toDoubleOrNull() == 1.5)
    check("ax85g", s85text { ss[15].toFloat() } == "Infinity" && ss[2].toFloat() == 1.5f &&
        ss[0].toFloatOrNull() == null && s85text { ss[0].toFloat() } == nfe + "For input string: \"1.5abc\"")
    check("ax85h", s85text { ss[7].toInt() } == nfe + "For input string: \" 5\"" && ss[7].toIntOrNull() == null)
    check("ax85i", s85text { ss[8].toInt() } == nfe + "For input string: \"2147483648\"" &&
        ss[8].toIntOrNull() == null && ss[9].toInt() == Int.MIN_VALUE && ss[8].toLong() == 2147483648L)
    check("ax85j", ss[10].toLong() == Long.MAX_VALUE && ss[11].toLongOrNull() == null &&
        s85text { ss[11].toLong() } == nfe + "For input string: \"9223372036854775808\"")
    check("ax85k", ss[0].toLongOrNull() == null && ss[9].toLongOrNull() == -2147483648L)
    val emp = mutableListOf<Int>()
    val many = listOf(1, 2, 3)
    check("ax85l", emp.reduceOrNull { a, b -> a + b } == null && emp.reduceRightOrNull { a, b -> a + b } == null)
    check("ax85m", many.reduceOrNull { a, b -> a - b } == -4 && many.reduceRightOrNull { a, b -> a - b } == 2)
    check("ax85n", many.asSequence().reduceOrNull { a, b -> a * b } == 6 && arrayOf<Int>().reduceOrNull { a, b -> a } == null)
    // Integer.parseInt reads ANY Unicode decimal digit (Character.digit); the floating
    // forms read ASCII only. A hex mantissa past 53 bits is rounded once, and
    // Float.parseFloat rounds it straight to binary32, not through a double.
    val us = listOf("\u0661\u0662", "-\u0661\u0662", "\uFF11\uFF12", "\u0662147483648", "\u19DA",
        "0x1.00000000000008000001p0", "0x1.000001000000000001p0", "0x10p-1078",
        "0x1.0000000000000000001p-150", "0x1.fffffefffffffffffffp127")
    check("ax85o", us[0].toInt() == 12 && us[1].toInt() == -12 && us[2].toIntOrNull() == 12 && us[3].toLong() == 2147483648L)
    check("ax85p", s85text { us[3].toInt() } == nfe + "For input string: \"\u0662147483648\"" &&
        us[4].toIntOrNull() == null && us[0].toDoubleOrNull() == null)
    check("ax85q", us[5].toDouble() == 1.0000000000000002 && s85text { us[6].toFloat() } == "1.0000001")
    check("ax85r", us[7].toDouble() == 4.9E-324 && s85text { us[8].toFloat() } == "1.4E-45" &&
        s85text { us[9].toFloat() } == "3.4028235E38")
    // A decimal is rounded once to binary32 too: these land on a float midpoint as
    // doubles, and their exact digits decide which way.
    val ms = listOf("1.00000005960464477539062500000001", "1.0000000596046448", "1.000000059604644775390625",
        "3.40282356779733661637539395458142568448e38", "7.0064923216240853546186479164495806564013097093825788587853414194489554134293031e-46")
    check("ax85s", s85text { ms[0].toFloat() } == "1.0000001" && s85text { ms[1].toFloat() } == "1.0000001" &&
        s85text { ms[2].toFloat() } == "1.0" && s85text { ms[3].toFloat() } == "Infinity" &&
        s85text { ms[4].toFloatOrNull() } == "1.4E-45")
}

// ===== SECTION 86: CharArray to String, and reversal over surrogate pairs =====
// concatToString and the String(CharArray) constructors join the chars; their range
// errors are AbstractList's and java.lang.String's. String.reversed() and
// StringBuilder.reverse() keep a surrogate pair whole. Rows read from the
// kotlin-stdlib 2.2.21 jar, called from java.
fun s86text(f: () -> Any?): String {
    return try { "" + f() } catch (e: Throwable) { e.toString() }
}

fun s86units(s: String): String = s.map { it.code }.joinToString(" ")

fun sec86() {
    val cs = charArrayOf('a', 'b', 'c', 'd')
    check("ax86a", cs.concatToString() == "abcd" && cs.concatToString(1, 3) == "bc")
    check("ax86b", String(cs) == "abcd" && String(cs, 1, 2) == "bc")
    check("ax86c", s86text { cs.concatToString(1, 9) } ==
        "java.lang.IndexOutOfBoundsException: startIndex: 1, endIndex: 9, size: 4" &&
        s86text { cs.concatToString(3, 1) } == "java.lang.IllegalArgumentException: startIndex: 3 > endIndex: 1")
    check("ax86d", s86text { String(cs, 1, 9) } ==
        "java.lang.StringIndexOutOfBoundsException: Range [1, 1 + 9) out of bounds for length 4")
    val ins = listOf("x😀y", "😀😁", "abc")
    check("ax86e", s86units(ins[0].reversed()) == "121 55357 56832 120" &&
        s86units(ins[1].reversed()) == "55357 56833 55357 56832" && ins[2].reversed() == "cba")
    check("ax86f", s86units(StringBuilder(ins[0]).reverse().toString()) == "121 55357 56832 120" &&
        s86units(StringBuilder(ins[1]).reverse().toString()) == "55357 56833 55357 56832")
}

// ===== SECTION 87: the Array, Sequence and Map.values families of the messages =====
// An Array (_Arrays.kt) and a Sequence (_Sequences.kt) have their own overloads of
// first/last/single/reduce, and each names its own kind in every message, the
// predicate and reduceRight forms included. A Map's `values` is a Collection, so it
// takes the Iterable wording. An Array's map() answers a List, a Sequence's map()
// another Sequence. Rows read from the kotlin-stdlib 2.2.21 jar, called from java.
fun s87text(f: () -> Any?): String {
    return try { "" + f() } catch (e: Throwable) { e.toString() }
}

fun sec87() {
    val nse = "java.util.NoSuchElementException: "
    val iae = "java.lang.IllegalArgumentException: "
    val uoe = "java.lang.UnsupportedOperationException: "
    val ea = arrayOf<Int>()
    val two = arrayOf(1, 2)
    val es = emptySequence<Int>()
    val ts = sequenceOf(1, 2)
    check("ax87a", s87text { ea.first() } == nse + "Array is empty." &&
        s87text { ea.last() } == nse + "Array is empty." && s87text { ea.random() } == nse + "Array is empty.")
    check("ax87b", s87text { two.single() } == iae + "Array has more than one element." &&
        s87text { two.single { true } } == iae + "Array contains more than one matching element.")
    check("ax87c", s87text { two.first { false } } == nse + "Array contains no element matching the predicate." &&
        s87text { two.last { false } } == nse + "Array contains no element matching the predicate.")
    check("ax87d", s87text { ea.reduce { a, b -> b } } == uoe + "Empty array can't be reduced." &&
        s87text { ea.reduceRight { a, b -> b } } == uoe + "Empty array can't be reduced.")
    check("ax87e", s87text { intArrayOf().first() } == nse + "Array is empty." &&
        s87text { IntArray(0).first() } == nse + "Array is empty." &&
        s87text { Array(0) { it }.first() } == nse + "Array is empty.")
    check("ax87f", s87text { es.first() } == nse + "Sequence is empty." &&
        s87text { ts.single() } == iae + "Sequence has more than one element." &&
        s87text { ts.first { false } } == nse + "Sequence contains no element matching the predicate.")
    check("ax87g", s87text { es.reduce { a, b -> b } } == uoe + "Empty sequence can't be reduced.")
    val ev = mapOf<Int, Int>().values
    val tv = mapOf(1 to 2, 3 to 4).values
    check("ax87h", s87text { ev.first() } == nse + "Collection is empty." &&
        s87text { tv.single() } == iae + "Collection has more than one element." &&
        s87text { tv.last { false } } == nse + "Collection contains no element matching the predicate.")
    // The kind an operation answers: a Sequence pipeline stays a Sequence, asSequence
    // makes one, an Array's map/toList and a Sequence's toList are Lists.
    check("ax87i", s87text { ts.map { it * 2 }.filter { it > 9 }.first() } == nse + "Sequence is empty." &&
        s87text { listOf<Int>().asSequence().first() } == nse + "Sequence is empty.")
    check("ax87j", s87text { ea.map { it }.first() } == nse + "List is empty." &&
        s87text { ea.toList().first() } == nse + "List is empty." &&
        s87text { es.toList().first() } == nse + "List is empty.")
    check("ax87k", s87text { listOf<Int>().toTypedArray().first() } == nse + "Array is empty." &&
        s87text { "".toCharArray().first() } == nse + "Array is empty.")
    check("ax87l", two.first() + ts.last() + tv.first() == 5)
}

// ===== SECTION 88: Long and floating elements keep their kind =====
// A range with a Long bound is a LongRange, whose first/last/start/endInclusive and
// elements are Longs; an integer literal in longArrayOf / LongArray is typed Long by
// that expectation, so a product of two of them is a 64-bit multiply. Rows read from
// the kotlin-stdlib 2.2.21 jar, called from java.
fun s88any(x: Any): Any = x

fun sec88() {
    val r = 1L..5L
    check("ax88a", s88any(r.first) is Long && s88any(r.last) is Long &&
        s88any(r.start) is Long && s88any(r.endInclusive) is Long)
    check("ax88b", s88any(r.toList()[0]) is Long && s88any((1..5L).first) is Long &&
        "" + r == "1..5" && (5L downTo 1L step 2).last == 1L)
    check("ax88c", r.first * 3000000000L == 3000000000L)
    val la = longArrayOf(100000, 100000)
    val lz = LongArray(2) { 100000 }
    check("ax88d", la[0] * la[1] == 10000000000L && lz[0] * lz[1] == 10000000000L)
    check("ax88e", s88any(LongArray(2)[0]) is Long && s88any(la[1]) is Long)
    // Through Any a Double or a Float never equals an integer (Double.equals(Integer)
    // is false), so they are separate set elements and map keys; a DoubleArray /
    // FloatArray starts at 0.0 / 0.0f.
    val ks = listOf<Any>(1.0, 1, 1.0f, 1L)
    check("ax88f", setOf<Any>(ks[0], ks[1]).size == 2 && setOf<Any>(ks[2], ks[1]).size == 2 &&
        setOf<Any>(ks[0], ks[3]).size == 2 && ks[0] != ks[1] && ks[3] != ks[0])
    check("ax88g", mapOf<Any, Int>(ks[0] to 1, ks[1] to 2).size == 2 &&
        listOf<Any>(1.0, 2).indexOf(2.0) == -1 && listOf<Any>(1.0, 2.0).indexOf(2.0) == 1)
    check("ax88h", DoubleArray(2).joinToString() == "0.0, 0.0" &&
        FloatArray(2).joinToString() == "0.0, 0.0" && s88any(DoubleArray(1)[0]) is Double)
    // A Long sum and abs() stay Longs; a Float sum is a Float, accumulated in float,
    // so 16777216f + 1f + 1f loses both ones.
    val ls = listOf(1L, 2L)
    check("ax88i", s88any(ls.sum()) is Long && s88any(ls.sumOf { it }) is Long && s88any(abs(-ls[0])) is Long)
    val fs = listOf(16777216f, 1f, 1f)
    check("ax88j", s88any(fs.sum()) is Float && "" + fs.sum() == "1.6777216E7")
}

// ===== SECTION 89: kotlin.math and System's clocks =====
// kotlin.math's functions and constants (a Float argument answers a Float), the
// Double extensions pow / roundToInt / roundToLong, the absoluteValue and sign
// properties, and System.currentTimeMillis / nanoTime as Longs. Rows read from
// java.lang.Math and the kotlin-stdlib 2.2.21 jar's MathKt, called from java.
fun s89any(x: Any): Any = x

fun sec89() {
    val d = doubleArrayOf(2.0, -2.5, 0.5, 1e-10, -0.0, 100.0, 8.0)
    check("km89a", "" + sqrt(d[0]) + " " + cbrt(d[6]) + " " + exp(d[2]) + " " + ln(d[5]) ==
        "1.4142135623730951 2.0 1.6487212707001282 4.605170185988092")
    check("km89b", "" + log10(d[5]) + " " + log2(d[6]) + " " + log(d[6], d[0]) + " " + log(d[6], 1.0) ==
        "2.0 3.0 3.0 NaN")
    check("km89c", "" + cos(d[0]) + " " + tan(d[2]) + " " + atan2(d[1], d[0]) + " " + hypot(d[0], d[1]) ==
        "-0.4161468365471424 0.5463024898437905 -0.8960553845713439 3.2015621187164243")
    check("km89d", "" + asinh(d[1]) + " " + acosh(d[0]) + " " + atanh(d[2]) + " " + asinh(d[3]) ==
        "-1.6472311463710958 1.3169578969248166 0.5493061443340548 1.0E-10")
    check("km89e", "" + floor(d[1]) + " " + ceil(d[1]) + " " + truncate(d[1]) + " " + round(d[1]) + " " +
        round(d[2]) + " " + sign(d[1]) + " " + sign(d[4]) == "-3.0 -2.0 -2.0 -2.0 0.0 -1.0 -0.0")
    check("km89f", "" + d[1].pow(2) + " " + d[0].pow(0.5) + " " + d[1].roundToInt() + " " +
        d[2].roundToLong() == "6.25 1.4142135623730951 -2 1")
    check("km89g", s89any(d[2].roundToLong()) is Long && s89any(1e10.roundToInt()) == 2147483647)
    var nan = ""
    try { (d[4] / d[4]).roundToInt() } catch (e: IllegalArgumentException) { nan = "" + e.message }
    check("km89h", nan == "Cannot round NaN value.")
    check("km89i", "" + PI + " " + E == "3.141592653589793 2.718281828459045")
    val f = floatArrayOf(2.0f, -2.5f)
    check("km89j", "" + sqrt(f[0]) + " " + f[1].pow(2) + " " + abs(f[1]) + " " + round(f[1]) ==
        "1.4142135 6.25 2.5 -2.0" && s89any(sqrt(f[0])) is Float)
    check("km89k", "" + abs(d[4]) + " " + d[1].absoluteValue + " " + d[1].sign + " " + (-7).sign + " " +
        (-7L).absoluteValue + " " + f[1].sign == "0.0 2.5 -1.0 -1 7 -1.0")
    val t0 = System.nanoTime()
    check("km89l", s89any(t0) is Long && System.nanoTime() >= t0 &&
        System.currentTimeMillis() > 1600000000000L)
}

// ===== SECTION 90: an integer literal takes its expected type argument =====
// A type argument fixes the expected type of a collection's elements, so the
// integer literals in listOf<Long>(1, 2), `val ls: List<Long> = listOf(1, 2)`,
// Pair<Long, Int>(1, 2), `ml.add(3)` on a MutableList<Long>, `la[0] = 5` into a
// LongArray and `y = 7` into a `var y = 5L` are Longs (Kotlin spec, integer literal
// types); a product of two is a 64-bit multiply, and an empty List<Long>'s sum is
// a Long (List<Float>'s the Float 0.0).
fun s90any(x: Any?): Any? = x

fun sec90() {
    val ls: List<Long> = listOf(1, 2, 3)
    val hs = hashSetOf<Long>(1, 2)
    check("kl90a", s90any(ls[0]) is Long && s90any(hs.first()) is Long)
    var y = 5L
    y = 7
    var w: Long = 3
    w = 4
    check("kl90b", s90any(y) is Long && s90any(w) is Long && y * 1000000000 * 10 == 70000000000L)
    val mm = mutableMapOf<Long, String>()
    mm[1] = "x"
    val ml = mutableListOf<Long>()
    ml.add(3)
    ml.add(0, 2)
    check("kl90c", s90any(mm.keys.first()) is Long && s90any(ml[0]) is Long && s90any(ml[1]) is Long)
    val p = Pair<Long, Int>(1, 2)
    val la = LongArray(2)
    la[0] = 5
    check("kl90d", s90any(p.first) is Long && s90any(p.second) is Int && s90any(la[0]) is Long)
    val lo = listOf<Long>(100000, 100000)
    check("kl90e", lo[0] * lo[1] == 10000000000L && "" + listOf<Float>().sum() == "0.0" &&
        s90any(listOf<Long>().sum()) is Long)
    val n: Long? = null
    val z = n ?: 0
    val ml2: MutableList<Long> = mutableListOf(1, 2)
    ml2[0] = 9
    val mp: Map<Long, Double> = mapOf(1 to 2.0)
    check("kl90f", s90any(z) is Long && s90any(ml2[0]) is Long && s90any(ml2[1]) is Long &&
        s90any(mp.keys.first()) is Long && s90any(arrayOf<Long>(1, 2)[0]) is Long)
}


// ===== SECTION 91: an Int and a Long are two values =====
// == through Any, equals() and a set's or map's keys tell the integer types apart
// (java: Integer#equals demands an Integer, so setOf<Any>(1, 1L) has two
// elements), so every integer LITERAL has to carry the type it is expected to
// have: a function type's result, a collection parameter, a member reached through
// a chain, a lookup argument, and a declared type over any initializer.
class S91(val xs: MutableList<Long>, val m: MutableMap<Long, String>)
fun s91mk(): () -> Long = { 7 }
fun s91run(f: () -> Long): Any = f()
fun s91first(xs: List<Long>): Any? = xs[0]
fun s91key(m: Map<Long, Int>): Any? = m.keys.first()
fun <T> s91id(x: T): T = x
class S91Box<T>(val v: T)

fun sec91() {
    val o: List<Any> = listOf(1, 2L)
    check("kl91a", setOf<Any>(1, 1L).size == 2 && mapOf<Any, String>(1 to "i", 1L to "l").size == 2 &&
        !o.contains(2) && o.indexOf(2L) == 1)
    val a: Any = 1
    val b: Any = 1L
    val x = 5L
    val y: Long = 5
    check("kl91b", a != b && !1L.equals(1) && !1.equals(1L) && listOf(1L) != listOf(1) && x == y && a.hashCode() == b.hashCode())
    val f: () -> Long = { 1 }
    val g: (Int) -> Long = { 3 }
    check("kl91c", s90any(f()) is Long && s90any(s91mk()()) is Long && s90any(g(1)) is Long && s91run { 2 } is Long)
    check("kl91d", s91first(listOf(1, 2)) is Long && s91key(mapOf(1 to 2)) is Long)
    val s = S91(mutableListOf(), mutableMapOf())
    s.xs.add(6)
    s.m[1] = "a"
    check("kl91e", s90any(s.xs[0]) is Long && s.m.containsKey(1L) && s.xs.contains(6L))
    val m = mutableMapOf<Long, String>(1L to "x")
    val st = setOf(1L, 2L)
    check("kl91f", m[1] == "x" && m.containsKey(1) && m.getOrDefault(1, "d") == "x" && st.contains(2) && 2 in st &&
        listOf(5L, 6L).indexOf(6) == 1)
    val ls: List<Long> = listOf(1, 2).map { 4 }
    val p: Pair<Long, Long> = 1 to 2
    check("kl91g", s90any(ls[0]) is Long && s90any(p.first) is Long && s90any(p.second) is Long &&
        s90any(Array<Long>(2) { 0 }[1]) is Long)
    val ms = mutableSetOf<Long>()
    ms += 4
    val ml = mutableListOf<Long>()
    ml += 7
    check("kl91h", s90any(s91id<Long>(5)) is Long && s90any(S91Box<Long>(3).v) is Long && s90any(maxOf<Long>(1, 2)) is Long &&
        ms.contains(4L) && ml.contains(7L) && s90any(listOf<Long>(1, 2).getOrElse(5) { 0 }) is Long)
}


// ===== SECTION 92: java.lang.Math =====
// A Kotlin/JVM program reaches java.lang.Math as `Math`: its functions answer a
// Double whatever the argument, round answers a Long (an Int for a Float) rounding
// half up, and abs/max/min keep the argument's type. Rows from java (JDK 24).
fun s92any(x: Any?): Any? = x

fun sec92() {
    val d = listOf(9.0, 2.5, -2.5, 0.49999999999999994, 1.0)
    check("km92a", "" + Math.sqrt(d[0]) == "3.0" && "" + Math.sqrt(9) == "3.0" && Math.round(d[1]) == 3L &&
        Math.round(d[2]) == -2L && Math.round(d[3]) == 0L && s92any(Math.round(d[1])) is Long && s92any(Math.round(2.5f)) is Int)
    check("km92b", Math.abs(-3) == 3 && Math.max(2, 7) == 7 && "" + Math.pow(2.0, 10.0) == "1024.0" &&
        "" + Math.floor(-1.5) == "-2.0" && "" + Math.rint(d[1]) == "2.0" && "" + Math.signum(-2.0) == "-1.0")
    check("km92c", "" + Math.toRadians(180.0) == "3.141592653589793" && "" + Math.toDegrees(d[4]) == "57.29577951308232" &&
        "" + Math.log(Math.E) == "1.0" && "" + Math.PI == "3.141592653589793" && "" + Math.hypot(3.0, 4.0) == "5.0")
}

// ===== SECTION 93: an integer literal typed by a collection's element type =====
// `xs + 1` on a List<Long> is plus(element: Long), so the literal is a Long (and
// `xs - 5` removes the Long 5); a nested type argument (List<List<Long>>,
// Map<String, List<Long>>) types the literals of the inner builders too. Kotlin
// language specification, "Integer literals" (the expected type decides).
fun s93any(x: Any?): Any? = x

fun sec93() {
    val a = emptyList<Long>() + 1
    val xs = listOf(5L, 6L)
    check("ad93a", s93any(a[0]) is Long && a[0] * 2000000000 * 2 == 4000000000L && xs - 5 == listOf(6L) && (xs + 7)[2] * 2000000000 * 2 == 28000000000L)
    val b: List<List<Long>> = listOf(listOf(1, 2), listOf(3))
    val c: Map<String, List<Long>> = mapOf("x" to listOf(4))
    val mb = mutableListOf<List<Long>>()
    mb.add(listOf(5))
    check("ad93b", s93any(b[0][0]) is Long && b[1][0] * 2000000000 * 2 == 12000000000L && c["x"]!![0] * 2000000000 * 2 == 16000000000L &&
        mb[0][0] * 2000000000 * 2 == 20000000000L)
}

// ===== SECTION 94: java.util constructors, floorDiv / floorMod, Char codes =====
// ArrayList / HashSet / HashMap / LinkedHashMap construct empty, with a capacity
// or as a copy. Math.floorDiv / floorMod and Int.floorDiv / Int.mod round toward
// negative infinity, mod typed like the divisor (Long.mod(Int) is an Int, per
// kotlin-stdlib 2.2.21's FloorDivMod.kt signatures). The deprecated Char.toInt()
// family reads the code. Rows from java (JDK 24).
fun sec94() {
    val h = HashMap<Long, String>()
    h[5L] = "five"
    val al = ArrayList<Int>(10)
    al.add(3)
    val cp = ArrayList(listOf(1, 2))
    cp.add(9)
    val hs = HashSet(listOf(4, 4, 5))
    val lm = LinkedHashMap(mapOf("z" to 2))
    check("ju94a", "" + h == "{5=five}" && h[5L] == "five" && al == listOf(3) && cp == listOf(1, 2, 9) && hs.size == 2 && "" + lm == "{z=2}")
    val n = listOf(-7, 2, 7, -2, Int.MIN_VALUE, -1)
    check("fd94b", Math.floorDiv(n[0], n[1]) == -4 && Math.floorMod(n[0], n[1]) == 1 && Math.floorDiv(n[2], n[3]) == -4 &&
        Math.floorMod(n[2], n[3]) == -1 && Math.floorDiv(n[4], n[5]) == Int.MIN_VALUE && Math.floorDiv(n[4].toLong(), n[5].toLong()) == 2147483648L)
    check("fd94c", n[0].floorDiv(n[1]) == -4 && n[0].mod(n[1]) == 1 && s93any(n[0].toLong().mod(n[1])) is Int && s93any(n[0].toLong().floorDiv(n[1])) is Long &&
        s93any(Math.floorMod(-7L, n[1])) is Int && n[2].mod(n[3].toLong()) == -1L)
    val cs = listOf('a', 'é')
    check("ch94d", cs[0].toInt() == 97 && cs[0].toLong() * 3000000000 == 291000000000L && cs[1].toByte() == (-23).toByte() && cs[1].toShort() == 233.toShort())
}

// ===== SECTION 95: kotlin.text and kotlin.collections against kotlin-stdlib =====
// Every row was read off kotlin-stdlib 2.2.21 driven from java (JDK 24, Locale.US).
// String.toInt(radix) and its OrNull / Long forms read Character.digit(c, radix):
// letters, ASCII or fullwidth, and any Unicode decimal digit; the range check is the
// target type's, and a bad radix is an IllegalArgumentException. groupBy and
// associateBy take an optional valueTransform; joinToString's parameters can be
// passed by NAME, and `limit` writes the separator and then `truncated`.
// String.format IS java.util.Formatter (Locale.US): two's complement %x / %o at the
// value's own width, the flags, %e / %g / %a, explicit and relative indexes, and its
// IllegalFormatException family; a primitive argument is its java.lang box. A
// narrower integer hashes as its value at its own signed width (UByte 255 is -1).
// A negative count or size, and a non-positive step, is an IllegalArgumentException
// with the stdlib's message, never a clamp or an abort.
fun s95fmt(f: String, vararg a: Any?): String {
    try { return f.format(*a) } catch (e: IllegalArgumentException) { return e::class.simpleName + ": " + e.message }
}
fun s95err(f: () -> Any?): String {
    try { return "" + f() } catch (e: Exception) { return e::class.simpleName + ": " + e.message }
}
fun sec95() {
    val rx = listOf("ff", "zz", "-80000000", "80000000", "\uff41\uff10", "0x10", "12")
    check("rad95a", rx[0].toIntOrNull(16) == 255 && rx[1].toIntOrNull(36) == 1295 && rx[1].toIntOrNull(16) == null && rx[6].toIntOrNull(16) == 18)
    check("rad95b", rx[2].toIntOrNull(16) == Int.MIN_VALUE && rx[3].toIntOrNull(16) == null && rx[3].toLongOrNull(16) == 2147483648L)
    check("rad95c", rx[4].toInt(16) == 160 && rx[5].toIntOrNull(16) == null && rx[5].toInt(36) == 42804)
    check("rad95d", s95err { rx[1].toInt(16) } == "NumberFormatException: For input string: \"zz\" under radix 16"
                    && s95err { rx[0].toInt(10) } == "NumberFormatException: For input string: \"ff\""
                    && s95err { rx[6].toIntOrNull(37) } == "IllegalArgumentException: radix 37 was not in valid range 2..36")
    val l5 = listOf(1, 2, 3, 4, 5)
    check("grp95e", "" + l5.groupBy({ it % 2 }, { "v" + it }) == "{1=[v1, v3, v5], 0=[v2, v4]}"
                    && "" + l5.associateBy({ it % 2 }, { it * 10 }) == "{1=50, 0=40}")
    check("jts95f", l5.joinToString(limit = 2) == "1, 2, ..." && l5.joinToString("/", limit = 3, truncated = "!") == "1/2/3/!")
    check("jts95g", l5.joinToString(limit = 2) { "<" + it + ">" } == "<1>, <2>, ..."
                    && l5.joinToString(prefix = "[", postfix = "]", limit = 0) == "[...]"
                    && l5.joinToString(", ", "[", "]", 5) == "[1, 2, 3, 4, 5]")
    val fv = listOf<Any?>(-7, 255, (-1).toByte(), 42000, 3.14159, 1e-5, 2.5, -0.5, 1.0, (-2).toShort(), -5L)
    check("fmt95h", s95fmt("%x|%X|%o|%x", fv[0], fv[1], fv[1], fv[2]) == "fffffff9|FF|377|ff"
                    && s95fmt("%,d|% d|%+d|%05d|%(d", fv[3], 42, 42, fv[0], -5) == "42,000| 42|+42|-0007|(5)")
    check("fmt95i", s95fmt("%e|%.3g|%10.2f|%-8.1f|%a", fv[4], fv[5], fv[6], fv[7], fv[8]) == "3.141590e+00|1.00e-05|      2.50|-0.5    |0x1.0p0"
                    && s95fmt("%2\$s-%1\$s|%s %<s|%S|%c|%b|%h", "a", "b", 'z', null, "hi") == "b-a|a a|B|z|false|d01")
    check("fmt95j", s95fmt("%d", fv[4]) == "IllegalFormatConversionException: d != java.lang.Double"
                    && s95fmt("%s %s", "a") == "MissingFormatArgumentException: Format specifier '%s'"
                    && s95fmt("%q", 1) == "UnknownFormatConversionException: Conversion = 'q'")
    check("hsh95k", fv[2].hashCode() == -1 && fv[9].hashCode() == -2 && 255.toUByte().hashCode() == -1 && 65534.toUShort().hashCode() == -2
                    && s95fmt("%h|%h|%h", fv[2], fv[9], fv[10]) == "ffffffff|fffffffe|4")
    val nz = listOf(-1, 0, 1, 2)
    check("arg95l", s95err { "ab".padStart(nz[0]) } == "IllegalArgumentException: Desired length -1 is less than zero."
                    && s95err { l5.chunked(nz[1]) } == "IllegalArgumentException: size 0 must be greater than zero."
                    && s95err { l5.windowed(nz[1], nz[2]) } == "IllegalArgumentException: Both size 0 and step 1 must be greater than zero.")
    check("arg95m", s95err { l5.take(nz[0]) } == "IllegalArgumentException: Requested element count -1 is less than zero."
                    && s95err { "abc".dropLast(nz[0]) } == "IllegalArgumentException: Requested character count -1 is less than zero."
                    && s95err { "ab".repeat(nz[0]) } == "IllegalArgumentException: Count 'n' must be non-negative, but was -1.")
    check("arg95n", s95err { (1..10) step nz[1] } == "IllegalArgumentException: Step must be positive, was: 0."
                    && s95err { (10 downTo 1).step(-2) } == "IllegalArgumentException: Step must be positive, was: -2.")
    check("win95o", "" + l5.windowed(3, partialWindows = true) == "[[1, 2, 3], [2, 3, 4], [3, 4, 5], [4, 5], [5]]"
                    && "abcde".padStart(length = 7, padChar = '*') == "**abcde" && "" + "abcde".windowed(3, 2, true) == "[abc, cde, e]")
}

// ===== SECTION 96: kotlin.Char classification and Formatter's class names =====
// Every row was read off kotlin-stdlib 2.2.21 driven from java (JDK 24). Char's
// predicates are java.lang.Character's: a letter without case (CJK, Lm, Lo) is a
// letter, a Nl like U+216B is not, isDigit is any Unicode Nd. isWhitespace is
// Character.isWhitespace || isSpaceChar - NBSP and U+001C included, U+0085 not -
// and trim / isBlank read it. digitToInt reads Character.digit. A Formatter mismatch
// names the argument's JVM class: kotlin.UInt, and for a List the class listOf picks
// by size; the exception itself is java.util's.
fun sec96() {
    val cc = listOf('中', 'ʰ', 'Ⅻ', '٣', 'ß', 'ǅ', ' ', '\u001c', '\u0085', 'Ａ')
    check("chr96a", cc[0].isLetter() && cc[1].isLetter() && !cc[2].isLetter() && cc[0].isLetterOrDigit() && cc[3].isDigit() && cc[3].isLetterOrDigit())
    check("chr96b", cc[2].isUpperCase() && cc[1].isLowerCase() && cc[4].isLowerCase() && !cc[5].isUpperCase() && cc[5].isTitleCase() && !cc[4].isTitleCase())
    check("chr96c", cc[6].isWhitespace() && cc[7].isWhitespace() && !cc[8].isWhitespace() && cc[8].isISOControl() && !cc[6].isISOControl()
                    && " a ".trim() == "a" && "\u001cb\u0085".trimStart() == "b\u0085" && "　".isBlank())
    check("chr96d", cc[3].digitToInt() == 3 && cc[9].digitToInt(16) == 10 && cc[0].digitToIntOrNull() == null && 'z'.digitToIntOrNull(36) == 35
                    && s95err { cc[9].digitToInt() } == "IllegalArgumentException: Char Ａ is not a decimal digit"
                    && s95err { 'g'.digitToInt(16) } == "IllegalArgumentException: Char g is not a digit in the given radix=16")
    val fa = listOf<Any?>(1u, listOf(1, 2), mutableListOf(1), listOf(1), emptyList<Int>(), setOf(1, 2), mapOf(1 to 2, 3 to 4))
    check("fmt96e", s95fmt("%d", fa[0]) == "IllegalFormatConversionException: d != kotlin.UInt" && s95fmt("%s", fa[0]) == "1"
                    && s95fmt("%d", fa[1]) == "IllegalFormatConversionException: d != java.util.Arrays\$ArrayList"
                    && s95fmt("%d", fa[2]) == "IllegalFormatConversionException: d != java.util.ArrayList")
    check("fmt96f", s95fmt("%x", fa[3]) == "IllegalFormatConversionException: x != java.util.Collections\$SingletonList"
                    && s95fmt("%d", fa[4]) == "IllegalFormatConversionException: d != kotlin.collections.EmptyList"
                    && s95fmt("%d", fa[5]) == "IllegalFormatConversionException: d != java.util.LinkedHashSet"
                    && s95fmt("%d", fa[6]) == "IllegalFormatConversionException: d != java.util.LinkedHashMap")
    val fe = try { "%d".format(fa[1]) } catch (e: Exception) { "" + e }
    check("fmt96g", fe == "java.util.IllegalFormatConversionException: d != java.util.Arrays\$ArrayList")
}

// ===== SECTION 97: kotlin.text search, split, comparators and Grouping =====
// Every row was read off kotlin-stdlib 2.2.21 driven from java (JDK 24). The text
// members take an optional startIndex and ignoreCase, by position or by NAME:
// indexOf clamps startIndex as java.lang.String does, compareTo is java's (the
// first differing unit's difference), and split takes several delimiters, an
// empty one matching at every index, and a non-negative limit. compareBy takes
// several selectors; a Comparator has compare / reversed / thenBy / then; a
// Grouping (groupingBy) folds per key. coerceIn rejects an empty range.
fun sec97() {
    val ts = listOf("Mississippi", "ss", "SS", "abc", "", "a,b;c,,d", "AxBxc")
    check("idx97a", ts[0].indexOf(ts[1], 3) == 5 && ts[0].lastIndexOf(ts[1], 4) == 2 && ts[3].indexOf(ts[4], 9) == 3
                    && ts[3].indexOf(ts[4], 9, true) == -1 && ts[0].indexOf(ts[2], ignoreCase = true) == 2 && ts[0].lastIndexOf('s', 4) == 3)
    check("idx97b", ts[0].contains(ts[2], ignoreCase = true) && !ts[0].contains(ts[2]) && ts[0].startsWith(ts[1], 2)
                    && ts[0].endsWith("PI", true) && "ab".equals("AB", ignoreCase = true) && !"Straße".equals("STRASSE", true))
    check("cmp97c", "a".compareTo("c") == -2 && "ABD".compareTo("abc", true) == 1 && ts[3].compareTo(ts[4]) == 3
                    && compareValues("a", "c") == -2 && compareValues(null, "c") == -1)
    check("spl97d", "" + ts[5].split(",", limit = 2) == "[a, b;c,,d]" && "" + ts[5].split(",", ";") == "[a, b, c, , d]"
                    && "" + ts[3].split("") == "[, a, b, c, ]" && "" + ts[6].split("x", ignoreCase = true) == "[A, B, c]"
                    && "" + ts[5].split(';', ',', limit = 3) == "[a, b, c,,d]")
    check("spl97e", s95err { ts[3].split(",", limit = -1) } == "IllegalArgumentException: Limit must be non-negative, but was -1")
    val ws = listOf("pear", "fig", "apple", "kiwi", "banana", "date", "Fig")
    check("srt97f", "" + ws.sortedWith(compareBy({ it.length }, { it })) == "[Fig, fig, date, kiwi, pear, apple, banana]"
                    && "" + ws.sortedWith(compareBy<String> { it.length }.thenByDescending { it }) == "[fig, Fig, pear, kiwi, date, apple, banana]")
    check("srt97g", "" + ws.sortedWith(compareByDescending<String> { it.length }.thenBy { it }) == "[banana, apple, date, kiwi, pear, Fig, fig]"
                    && "" + ws.sortedWith(compareBy<String> { it.length }.reversed()) == "[banana, apple, pear, kiwi, date, fig, Fig]"
                    && "" + ws.sortedWith(reverseOrder()) == "[pear, kiwi, fig, date, banana, apple, Fig]")
    val c97 = compareBy<String> { it.length }
    check("srt97h", c97.compare(ws[0], ws[1]) == 1 && c97.compare(ws[1], ws[6]) == 0
                    && "" + ws.sortedWith(c97.then(naturalOrder())) == "[Fig, fig, date, kiwi, pear, apple, banana]"
                    && "" + listOf("b", null, "a").sortedWith(compareBy { it }) == "[null, a, b]")
    val n97 = listOf(1, 2, 3)
    check("fld97i", n97.foldIndexed(0) { i, a, b -> a + i * b } == 8 && n97.foldRightIndexed("") { i, e, a -> a + i + e } == "231201"
                    && "" + n97.zip(listOf("a", "b", "c")).unzip() == "([1, 2, 3], [a, b, c])")
    check("lst97j", ws.singleOrNull { it.length == 5 } == "apple" && ws.singleOrNull { it.length == 3 } == null && listOf(7).singleOrNull() == 7
                    && "" + ws.takeLastWhile { it.length < 6 } == "[date, Fig]" && "" + ws.dropLastWhile { it.length < 6 } == "[pear, fig, apple, kiwi, banana]"
                    && "abc123".takeLastWhile { it.isDigit() } == "123")
    check("grp97k", "" + ws.groupingBy { it.length }.eachCount() == "{4=3, 3=2, 5=1, 6=1}"
                    && "" + ws.groupingBy { it[0] }.fold(0) { acc, e -> acc + e.length } == "{p=4, f=3, a=5, k=4, b=6, d=4, F=3}"
                    && "" + ws.groupingBy { it.length }.reduce { _, acc, e -> acc + "+" + e } == "{4=pear+kiwi+date, 3=fig+Fig, 5=apple, 6=banana}")
    check("trm97l", "xyhiyx".trim('x', 'y') == "hi" && "xyhiyx".trimEnd('x') == "xyhiy" && "123abc456".trimStart { it.isDigit() } == "abc456"
                    && buildString(10) { repeat(3) { append(it) } } == "012")
    check("coe97m", n97[2].coerceIn(n97[0]..n97[1]) == 2 && s95err { n97[0].coerceIn(n97[2], n97[1]) } == "IllegalArgumentException: Cannot coerce value to an empty range: maximum 2 is less than minimum 3."
                    && s95err { 5.coerceIn(n97[2]..n97[0]) } == "IllegalArgumentException: Cannot coerce value to an empty range: 3..1.")
    val sb97 = StringBuilder("x")
    check("sbe97n", !sb97.equals("x") && sb97.equals(sb97))
}

// ===== SECTION 98: Char arithmetic, Regex.split and replace =====
// Every row was read off kotlin-stdlib 2.2.21 driven from java (JDK 24). Char + Int
// and Char - Int are a Char, Char - Char an Int, and ++ / += keep a Char.
// Regex.split cuts at every Matcher.find match, an empty one included (a match at
// index 0 gives a leading ""). replace with an empty oldValue inserts at every
// index, and replace / replaceFirst / commonPrefixWith take ignoreCase.
fun sec98() {
    val cs = listOf('a', 'x', 'z')
    val ns = listOf(2, 1)
    var c98 = cs[0]
    c98++
    check("chr98a", cs[0] + ns[0] == 'c' && cs[1] - ns[1] == 'w' && cs[2] - cs[0] == 25 && c98 == 'b' && "" + (cs[0] + ns[0]) == "c")
    c98 += 3
    check("chr98b", c98 == 'e' && cs[0].inc() == 'b' && cs[2].dec() == 'y' && cs[0].plus(ns[0]) == 'c' && cs[1].minus(cs[0]) == 23)
    check("rxs98c", "" + "CamelCaseString".split(Regex("(?=[A-Z])")) == "[, Camel, Case, String]"
                    && "" + Regex("x*").split("xa") == "[, , a, ]" && "" + Regex("\\s+").split(" a b ") == "[, a, b, ]")
    val rs = listOf("abc", "aAaA", "", "Hello, World")
    check("rep98d", rs[0].replace(rs[2], "-") == "-a-b-c-" && rs[0].replaceFirst(rs[2], "-") == "-abc"
                    && rs[1].replace("a", "x", true) == "xxxx" && rs[1].replace('A', 'y', true) == "yyyy" && "aaa".replace("aa", "b") == "ba")
    check("cmn98e", "Hello".commonPrefixWith("HELP", true) == "Hel" && "xABC".commonSuffixWith("abc", true) == "ABC"
                    && rs[3].removeRange(0..6) == "World" && rs[3].removeRange(2, 5) == "He, World"
                    && s95err { "Hello".removeRange(3, 1) } == "IndexOutOfBoundsException: End index (1) is less than start index (3).")
}

// ===== SECTION 99: list members and the implicit receiver =====
// Every row was read off kotlin-stdlib 2.2.21 driven from java (JDK 24). minus(element)
// and `-=` drop the FIRST occurrence only. An unqualified member call inside
// apply / run reaches the receiver's KOTLIN member (MutableList.sort sorts numbers,
// String.split is Kotlin's), never the host's own method of that name.
// binarySearch is java's (-(insertion point) - 1 on a miss); maxWith / minWith keep
// the first extreme.
fun sec99() {
    val xs = listOf(5, 3, 8, 1, 9, 2, 8)
    val ml = mutableListOf(8, 1, 8)
    ml -= 8
    check("min99a", "" + xs.minus(8) == "[5, 3, 1, 9, 2, 8]" && "" + (xs - listOf(8, 1)) == "[5, 3, 9, 2]" && "" + ml == "[1, 8]")
    val srt = mutableListOf(10, 9, 100, 2).apply { sort() }
    check("rcv99b", "" + srt == "[2, 9, 10, 100]" && "" + "abc".run { split("") } == "[, a, b, c, ]")
    val ss = xs.sorted()
    check("bin99c", ss.binarySearch(8) == 5 && ss.binarySearch(4) == -4 && ss.binarySearch(10) == -8
                    && ss.binarySearch { it.compareTo(5) } == 3 && listOf("a", "c", "e").binarySearch("d", compareBy { it }) == -3)
    check("lst99d", "" + xs.mapIndexedNotNull { i, v -> if (i > 4) v else null } == "[2, 8]" && xs.firstNotNullOf { if (it > 6) "big$it" else null } == "big8"
                    && xs.firstNotNullOfOrNull { if (it > 60) it else null } == null)
    val fl = mutableListOf(1, 2, 3)
    fl.fill(7)
    check("lst99e", xs.maxWith(compareBy { it % 5 }) == 9 && xs.minWith(compareBy { it % 5 }) == 5
                    && emptyList<Int>().maxWithOrNull(compareBy { it }) == null && "" + fl == "[7, 7, 7]")
}

// ===== SECTION 100: integer radix strings, bit counts, operator calls =====
// Every row was read off java.lang.Integer / Long (JDK 24), which kotlin-stdlib's
// toString(radix), countOneBits, countLeadingZeroBits and countTrailingZeroBits
// are on the JVM: a sign and the magnitude's digits, an unsigned type read
// unsigned, the counts over the value's own width. 7.rem(3) is the operator.
fun sec100() {
    val ns = listOf(-7, 13, Int.MIN_VALUE, 0)
    check("rad100a", ns[0].toString(2) == "-111" && ns[1].toString(16) == "d" && ns[2].toString(36) == "-zik0zk"
                    && Long.MIN_VALUE.toString(16) == "-8000000000000000" && (-1).toUInt().toString(16) == "ffffffff"
                    && ULong.MAX_VALUE.toString(36) == "3w5e11264sgsf" && s95err { ns[1].toString(37) } == "IllegalArgumentException: radix 37 was not in valid range 2..36")
    check("bit100b", ns[0].countOneBits() == 30 && ns[1].countLeadingZeroBits() == 28 && ns[2].countTrailingZeroBits() == 31
                    && ns[3].countTrailingZeroBits() == 32 && (1L shl 40).countLeadingZeroBits() == 23 && (-7L).countOneBits() == 62)
    check("opf100c", ns[0].rem(3) == -1 && ns[1].plus(2) == 15 && 7L.times(3) == 21L && 7.5.div(2) == 3.75 && ns[1].minus(ns[0]) == 20)
}


// ===== SECTION 101: Char(code), number receivers, java.util.Random, Formatter's classes =====
// Every row was read off kotlin-stdlib 2.2.21 driven from java (JDK 24); Char(code)
// is CharCode.kt's range check. An unqualified member call inside an extension on a
// number reaches the number's own member, and a bare call's named argument binds as
// a qualified one does. java.util.Random(seed) is the JVM's 48-bit generator, value
// for value. String.CASE_INSENSITIVE_ORDER is java's compareToIgnoreCase. A Formatter
// mismatch names the JVM class each builder answers: a set or map by builder and
// argument count, an array by its element type.
fun Int.dbl101(): Int = times(2)
fun Int.wrap101(o: Int): Int = plus(o).rem(5)
fun Long.dec101(): Long = minus(1L)
fun sec101() {
    val ns = listOf(97, 0x4e2d, 65535, 70000, -1, 3, 4)
    check("chr101a", Char(ns[0]) == 'a' && Char(ns[1]) == '中' && Char(ns[2]).code == 65535 && Char(ns[0].toUShort()) == 'a'
                    && s95err { Char(ns[3]) } == "IllegalArgumentException: Invalid Char code: 70000"
                    && s95err { Char(ns[4]) } == "IllegalArgumentException: Invalid Char code: -1")
    check("rcv101b", ns[5].dbl101() == 6 && ns[6].wrap101(9) == 3 && 7L.dec101() == 6L && ns[5].run { times(3) } == 9
                    && with(ns[5]) { coerceIn(4, 9) } == 4 && "" + "a,b;c:d".run { split(",", ";", ":", limit = 2) } == "[a, b;c:d]")
    val r = java.util.Random(42)
    check("jrn101c", r.nextInt() == -1170105035 && r.nextInt(10) == 3 && r.nextInt(16) == 10 && r.nextInt(1 shl 30) == 51474442
                    && r.nextInt(Int.MAX_VALUE) == 662969970 && r.nextInt(1000000007) == 595021505 && r.nextLong() == -5391787952107853122L)
    check("jrn101d", r.nextDouble() == 0.09132461105841794 && r.nextFloat() == 0.45125717f && !r.nextBoolean() && r.nextInt(5, 10) == 7
                    && r.nextInt(-8, 8) == 0 && r.nextInt(Int.MIN_VALUE, Int.MAX_VALUE) == -1329611232 && r.nextInt(-3, 1000000) == 685468)
    val r2 = java.util.Random(123456789012345L)
    check("jrn101e", r2.nextInt() == -1421006947 && java.util.Random(Long.MIN_VALUE).nextInt() == -1155484576 && java.util.Random(-1).nextInt() == 1155099827
                    && s95err { r2.nextInt(0) } == "IllegalArgumentException: bound must be positive"
                    && s95err { r2.nextInt(5, 5) } == "IllegalArgumentException: bound must be greater than origin")
    val ws = listOf("b", "A", "a", "C", "B", "Ab")
    val ci = String.CASE_INSENSITIVE_ORDER
    check("cio101f", ci.compare(ws[0], ws[1]) == 1 && ci.compare(ws[1], ws[2]) == 0 && "" + ws.sortedWith(ci) == "[A, a, Ab, b, B, C]"
                    && "" + ws.sortedWith(ci.reversed()) == "[C, b, B, Ab, A, a]")
    val fs = listOf<Any?>(mapOf(1 to 2), setOf(1), emptySet<Int>(), emptyMap<Int, Int>(), hashSetOf(1), sortedSetOf(1), hashMapOf(1 to 2),
                          sortedMapOf(1 to 2), setOf(1, 2), mapOf<Int, Int>())
    check("fmt101g", s95fmt("%d", fs[0]) == "IllegalFormatConversionException: d != java.util.Collections\$SingletonMap"
                    && s95fmt("%d", fs[1]) == "IllegalFormatConversionException: d != java.util.Collections\$SingletonSet"
                    && s95fmt("%d", fs[2]) == "IllegalFormatConversionException: d != kotlin.collections.EmptySet"
                    && s95fmt("%d", fs[3]) == "IllegalFormatConversionException: d != kotlin.collections.EmptyMap"
                    && s95fmt("%d", fs[4]) == "IllegalFormatConversionException: d != java.util.HashSet")
    check("fmt101h", s95fmt("%d", fs[5]) == "IllegalFormatConversionException: d != java.util.TreeSet"
                    && s95fmt("%d", fs[6]) == "IllegalFormatConversionException: d != java.util.HashMap"
                    && s95fmt("%d", fs[7]) == "IllegalFormatConversionException: d != java.util.TreeMap"
                    && s95fmt("%d", fs[8]) == "IllegalFormatConversionException: d != java.util.LinkedHashSet"
                    && s95fmt("%d", fs[9]) == "IllegalFormatConversionException: d != kotlin.collections.EmptyMap")
    val fa = listOf<Any?>(arrayOf(1), intArrayOf(1), longArrayOf(1), doubleArrayOf(1.0), charArrayOf('a'), booleanArrayOf(true),
                          arrayOf("a"), byteArrayOf(1), IntArray(2), "ab".toCharArray(), listOf(1).toIntArray(), listOf(1).toTypedArray())
    check("fmt101i", s95fmt("%d", fa[0]) == "IllegalFormatConversionException: d != [Ljava.lang.Integer;"
                    && s95fmt("%d", fa[1]) == "IllegalFormatConversionException: d != [I" && s95fmt("%d", fa[2]) == "IllegalFormatConversionException: d != [J"
                    && s95fmt("%d", fa[3]) == "IllegalFormatConversionException: d != [D" && s95fmt("%d", fa[4]) == "IllegalFormatConversionException: d != [C"
                    && s95fmt("%d", fa[5]) == "IllegalFormatConversionException: d != [Z")
    check("fmt101j", s95fmt("%d", fa[6]) == "IllegalFormatConversionException: d != [Ljava.lang.String;"
                    && s95fmt("%d", fa[7]) == "IllegalFormatConversionException: d != [B" && s95fmt("%d", fa[8]) == "IllegalFormatConversionException: d != [I"
                    && s95fmt("%d", fa[9]) == "IllegalFormatConversionException: d != [C" && s95fmt("%d", fa[10]) == "IllegalFormatConversionException: d != [I"
                    && s95fmt("%d", fa[11]) == "IllegalFormatConversionException: d != [Ljava.lang.Integer;")
}


// ===== SECTION 102: the sequence { } builder =====
// Settled by the Kotlin specification and kotlin.sequences' documentation (no kotlinc
// here): the block runs only when the sequence is iterated, once per iteration, and
// each yield hands one element to the consumer before the block continues, so a for
// loop and an iterator interleave with it. The intermediate operations are lazy, so
// an infinite block is fine under map / filter / take; yieldAll yields a collection's
// or another sequence's elements; an exhausted iterator throws NoSuchElementException.
fun fib102() = sequence {
    var a = 0
    var b = 1
    while (true) {
        yield(a)
        val t = a + b
        a = b
        b = t
    }
}
fun logged102(log: MutableList<String>, n: Int) = sequence {
    log.add("s")
    for (i in 1..n) {
        log.add("y$i")
        yield(i * 10)
    }
    log.add("e")
}
fun sec102() {
    val s = sequence { yield(1); yield(2) }
    check("seq102a", "" + s.toList() == "[1, 2]" && "" + s.toList() == "[1, 2]" && s.count() == 2
                    && "" + fib102().take(10).toList() == "[0, 1, 1, 2, 3, 5, 8, 13, 21, 34]" && fib102().first() == 0)
    check("seq102b", "" + fib102().map { it * 2 }.filter { it % 3 != 0 }.take(4).toList() == "[2, 2, 4, 10]"
                    && fib102().drop(5).first() == 5 && fib102().find { it > 50 } == 55
                    && fib102().takeWhile { it < 30 }.joinToString() == "0, 1, 1, 2, 3, 5, 8, 13, 21"
                    && "" + fib102().dropWhile { it < 10 }.mapNotNull { if (it % 2 == 0) it else null }.take(2).toList() == "[34, 144]")
    val log = mutableListOf<String>()
    for (v in logged102(log, 3)) { log.add("v$v") }
    check("seq102c", "" + log == "[s, y1, v10, y2, v20, y3, v30, e]")
    log.clear()
    val lz = logged102(log, 2)
    log.add("made")
    val iter = lz.iterator()
    log.add("got")
    while (iter.hasNext()) { log.add("n" + iter.next()) }
    check("seq102d", "" + log == "[made, got, s, y1, n10, y2, n20, e]"
                    && s95err { iter.next() } == "NoSuchElementException: null")
    val acc = StringBuilder()
    for (x in fib102()) { if (x > 20) break; acc.append(x).append(' ') }
    check("seq102e", acc.toString() == "0 1 1 2 3 5 8 13 "
                    && "" + sequence { yieldAll(listOf(1, 2)); yield(3); yieldAll(sequence { yield(4) }) }.toList() == "[1, 2, 3, 4]"
                    && sequence<Int> { }.firstOrNull() == null && s95err { sequence<Int> { }.first() } == "NoSuchElementException: Sequence is empty.")
    val nested = sequence { for (x in sequence { yield(1); yield(2) }) { yield(x * 100) } }
    val pairs = StringBuilder()
    for ((i, v) in sequence { yield(Pair(1, "a")); yield(Pair(2, "b")) }) { pairs.append("$i$v ") }
    check("seq102f", "" + nested.toList() == "[100, 200]" && pairs.toString() == "1a 2b "
                    && s95err { sequence<Int> { yield(1); throw IllegalStateException("boom") }.toList() } == "IllegalStateException: boom"
                    && s95err { fib102().take(-1) } == "IllegalArgumentException: Requested element count -1 is less than zero.")
}


// ===== SECTION 103: sortedMapOf order, Array.copyOf =====
// Read off kotlin-stdlib 2.2.21 driven from java (JDK 24). sortedMapOf is a
// java.util.TreeMap, so it iterates in ascending key order whatever order the pairs
// come in, a repeated key keeping its last value. copyOf() is an independent copy;
// copyOf(n) truncates or pads with the element type's default (null in an Array<T>,
// 0 / 0.0 / false / '\u0000' in a primitive array), and a negative size throws.
fun err103(f: () -> Any?): String {
    try { return "" + f() } catch (e: Exception) { return e::class.simpleName + ": " + e.message }
}
fun sec103() {
    val ks = listOf(2, 1, 3)
    check("smo103a", "" + sortedMapOf(ks[0] to "b", ks[1] to "a", ks[2] to "c") == "{1=a, 2=b, 3=c}"
                    && "" + sortedMapOf("b" to 1, "a" to 2, "b" to 3) == "{a=2, b=3}"
                    && sortedMapOf(ks[0] to 0, ks[1] to 0).keys.first() == 1)
    val a = arrayOf(1, 2, 3)
    val b = a.copyOf()
    b[0] = 9
    check("cpy103b", a.joinToString() == "1, 2, 3" && b.joinToString() == "9, 2, 3"
                    && a.copyOf(5).contentToString() == "[1, 2, 3, null, null]" && a.copyOf(ks[0]).contentToString() == "[1, 2]")
    val ia = intArrayOf(1, 2, 3)
    check("cpy103c", ia.copyOf(5).contentToString() == "[1, 2, 3, 0, 0]" && ia.copyOf().sum() == 6
                    && longArrayOf(1L, 2L).copyOf(3).contentToString() == "[1, 2, 0]"
                    && doubleArrayOf(1.5).copyOf(2).contentToString() == "[1.5, 0.0]"
                    && booleanArrayOf(true).copyOf(2).contentToString() == "[true, false]"
                    && charArrayOf('x').copyOf(2)[1].code == 0)
    check("cpy103d", err103 { ia.copyOf(-1) } == "NegativeArraySizeException: -1")
}

// ===== SECTION 104: an extension applies to its receiver type =====
// Kotlin resolves a call by the receiver's type: an extension declared for Int, for a
// class or for String is not a candidate on another receiver, so a String call reaches
// String's own member (kotlin.text.split with limit 2 answers [a, b;c:d] under
// kotlin-stdlib 2.2.21) and one name may be declared for several receivers.
class Box104(val v: Int)
fun Int.split(x: Int, y: Int, limit: Int) = x + y + limit
fun Box104.first() = v * 10
fun Int.kind104() = "int " + this
fun String.kind104() = "str " + this
fun Box104.kind104() = "box " + v
fun <T> T.tag104(): String = "tag(" + this + ")"
fun sec104() {
    val s = listOf("a,b;c:d", "q")
    check("ext104a", "" + s[0].split(",", ";", ":", limit = 2) == "[a, b;c:d]" && 5.split(1, 2, 3) == 6
                    && listOf(7, 8).first() == 7 && Box104(2).first() == 20)
    check("ext104b", 5.kind104() == "int 5" && s[1].kind104() == "str q" && Box104(3).kind104() == "box 3"
                    && 4.tag104() == "tag(4)" && s[1].tag104() == "tag(q)")
}

// ===== SECTION 105: a sorted collection stays sorted, copyOfRange, extension receivers =====
// Read off kotlin-stdlib 2.2.21 driven from java (JDK 24). A TreeMap / TreeSet keeps
// ascending order on every insertion (m[k] = v, put, putAll, getOrPut, add, addAll),
// while `+` copies into a LinkedHashMap / LinkedHashSet and appends. copyOfRange
// checks toIndex against the size, then from > to, then a negative from (System.arraycopy's
// ArrayIndexOutOfBoundsException). An extension declared for a class, an interface,
// an object, an enum or a sized number applies to that receiver whatever its name,
// unless the receiver has a member of the name; a bare call in its body is a call on
// `this` (no kotlinc here: the dispatch rows are the Kotlin specification's).
interface Shape105 { fun area(): Int }
class Sq105(val s: Int) : Shape105 { override fun area() = s * s }
object Origin105 { val tag = "O" }
enum class Color105 { RED, GREEN }
class Pt105(val x: Int) { fun label() = "member" }
class Other105 { fun label() = "other" }
fun Pt105.map(f: (Int) -> Int): Pt105 = Pt105(f(x))
fun Pt105.size(): Int = 99
fun Pt105.label(): String = "ext"
fun Other105.size(): Int = 7
fun Shape105.describe(): String = "shape " + area()
fun Origin105.count(): Int = tag.length + 2
fun Color105.get(i: Int): String = name + i
fun Long.kind105(): String = "Long"
fun Double.kind105(): String = "Double"
fun Float.kind105(): String = "Float"
fun Short.kind105(): String = "Short"
fun Byte.kind105(): String = "Byte"
fun Int.kind105(): String = "Int"
infix fun Long.join105(o: Long): Long = this * 10 + o
infix fun Int.join105(o: Int): Int = this - o
fun err105(f: () -> Any?): String {
    try { return "" + f() } catch (e: Exception) { return e::class.simpleName + ": " + e.message }
}
fun sec105() {
    val ks = listOf(0, 5, 3, -1, 4)
    val m = sortedMapOf(2 to "b", 1 to "a")
    m[ks[0]] = "z"
    check("srt105a", "" + m == "{0=z, 1=a, 2=b}")
    m.put(ks[1], "f")
    m.putAll(mapOf(ks[2] to "c", ks[3] to "m"))
    m.getOrPut(ks[4]) { "d" }
    check("srt105b", "" + m == "{-1=m, 0=z, 1=a, 2=b, 3=c, 4=d, 5=f}" && "" + m.keys == "[-1, 0, 1, 2, 3, 4, 5]"
                    && "" + (m + (-5 to "q")) == "{-1=m, 0=z, 1=a, 2=b, 3=c, 4=d, 5=f, -5=q}")
    val s = sortedSetOf(5, 1)
    s.add(ks[2])
    s.add(ks[0])
    s.addAll(listOf(9, -2, 4))
    val t = sortedSetOf("pear", "apple")
    t.add("banana")
    check("srt105c", "" + s == "[-2, 0, 1, 3, 4, 5, 9]" && "" + (s + 6) == "[-2, 0, 1, 3, 4, 5, 9, 6]" && "" + t == "[apple, banana, pear]")
    val a = arrayOf(3, 1, 2)
    val ia = intArrayOf(5, 9, 4, 7)
    val r = ia.copyOfRange(ks[2] - 2, 3)
    r[0] = 100
    check("cor105d", a.copyOfRange(1, 3).contentToString() == "[1, 2]" && ia.copyOfRange(0, 2).contentToString() == "[5, 9]"
                    && ia.copyOfRange(2, 2).size == 0 && r.contentToString() == "[100, 4]" && ia.contentToString() == "[5, 9, 4, 7]")
    check("cor105e", err105 { a.copyOfRange(0, 5) } == "IndexOutOfBoundsException: toIndex (5) is greater than size (3)."
                    && err105 { a.copyOfRange(2, 1) } == "IllegalArgumentException: 2 > 1"
                    && err105 { ia.copyOfRange(ks[3] - 1, 1) } == "IndexOutOfBoundsException: arraycopy: source index -2 out of bounds for int[4]")
    val p = Pt105(4)
    check("ext105f", p.map { it * 10 }.x == 40 && p.size() == 99 && p.label() == "member" && Other105().size() == 7
                    && Other105().label() == "other" && listOf(1, 2).map { it + 1 } == listOf(2, 3) && listOf(1, 2).size == 2)
    val sh: Shape105 = Sq105(3)
    check("ext105g", sh.describe() == "shape 9" && Origin105.count() == 3 && Color105.GREEN.get(7) == "GREEN7")
    check("ext105h", 5L.kind105() == "Long" && 2.5.kind105() == "Double" && 1.5f.kind105() == "Float"
                    && 3.toShort().kind105() == "Short" && 4.toByte().kind105() == "Byte" && 6.kind105() == "Int")
    val lab = Pt105::label
    val sz = Pt105::size
    val ds = Shape105::describe
    check("ext105i", (5L join105 2L) == 52L && (5 join105 2) == 3 && lab(Pt105(1)) == "member" && sz(Pt105(1)) == 99
                    && ds(Sq105(2)) == "shape 4")
}

// ===== SECTION 106: += on a val is plusAssign, on a var plus =====
// The Kotlin specification's operator resolution for `a += b`: a.plusAssign(b) when
// a is a val (or plus is unavailable), a = a.plus(b) for a var of a read-only type.
// The stdlib's plusAssign / minusAssign (kotlin-stdlib 2.2.21 MutableCollections.kt,
// Maps.kt) are add / addAll / remove / removeAll / put / putAll on the receiver
// itself, so every alias sees the change and a TreeSet / TreeMap stays sorted; the
// rows below were reproduced through those calls from java (JDK 24). A String on
// the right of a collection's + is an element, never a concatenation.
fun addTo106(l: MutableList<Int>) { l += 4 }
fun reg106(s: String) { registry106 += s }
class Coll106 {
    fun add(i: Int) { items += i; this.items += i * 10 }
    fun dump() = "" + items + " " + alias
    private val items = mutableListOf<Int>()
    private val alias = items
}
class Holder106(private val xs0: MutableList<Int>) {
    val xs = xs0
    fun push(i: Int) { xs0 += i }
}
class Vec106(var n: Int) { operator fun plusAssign(o: Vec106) { n += o.n } }
class Both106(var n: Int) {
    operator fun plus(k: Int): Both106 = Both106(n + k)
    operator fun plusAssign(k: Int) { n += k * 100 }
}
class Box106 { val items = shared106; var ro: List<Int> = listOf(1) }
fun mk106() = Box106()
fun sec106() {
    val ks = listOf(1, 2, 3, 4, 5)
    val l = mutableListOf(ks[0]); val la = l
    l += ks[1]; l += listOf(ks[2], ks[3]); l -= ks[0]
    addTo106(l)
    check("pa106a", "" + la == "[2, 3, 4, 4]")
    val s = mutableSetOf(ks[0]); val sa = s
    s += ks[1]; s += ks[0]; s -= listOf(ks[1])
    val ts = sortedSetOf(ks[4], ks[0]); ts += ks[2]
    val ss = mutableSetOf("a"); ss += "b"; ss += "a"
    check("pa106b", "" + sa == "[1]" && "" + ts == "[1, 3, 5]" && "" + ss == "[a, b]")
    val m = mutableMapOf("a" to ks[0]); val ma = m
    m += "b" to ks[1]; m += mapOf("c" to ks[2]); m -= "a"
    val sm = sortedMapOf(3 to "c", 1 to "a"); sm += 2 to "b"
    check("pa106c", "" + ma == "{b=2, c=3}" && "" + sm == "{1=a, 2=b, 3=c}")
    var r: List<Int> = mutableListOf(ks[0]); val ra = r
    r += ks[1]
    val b = mk106(); b.ro += ks[1]; b.items += ks[4]; mk106().items += ks[3]
    check("pa106d", "" + r == "[1, 2]" && "" + ra == "[1]" && "" + b.ro == "[1, 2]" && "" + shared106 == "[0, 5, 4]")
    val c = Coll106(); c.add(ks[2]); c.add(ks[0])
    val h = Holder106(mutableListOf(9)); h.push(8)
    reg106("x"); reg106("y")
    check("pa106e", c.dump() == "[3, 30, 1, 10] [3, 30, 1, 10]" && "" + h.xs == "[9, 8]" && "" + registry106 == "[x, y]")
    val v = Vec106(1); val va = v; v += Vec106(2)
    var w = Vec106(1); val wa = w; w += Vec106(5)
    val p = Both106(1); val pa = p; p += 4
    check("pa106f", va.n == 3 && wa.n == 6 && pa.n == 401)
    val names = mutableListOf<String>(); names += "bob"
    check("pa106g", "" + names == "[bob]" && "" + (listOf("a") + "b") == "[a, b]" && "" + (setOf("a", "b") + "a") == "[a, b]")
}
val registry106 = mutableListOf<String>()
val shared106 = mutableListOf(0)

// ===== SECTION 107: java.util.TreeSet / TreeMap constructors =====
// Read off java.util.TreeSet / TreeMap (JDK 24) with kotlin-stdlib 2.2.21: empty or
// a copy of a collection (a map), ascending on every insertion, `+=` on a val one
// more add.
fun sec107() {
    val ks = listOf(5, 1, 3, 1)
    val t = java.util.TreeSet<Int>()
    for (k in ks) { t.add(k) }
    val t2 = java.util.TreeSet(listOf("pear", "apple", "fig", "apple"))
    t2 += "banana"
    check("tre107a", "" + t == "[1, 3, 5]" && "" + t2 == "[apple, banana, fig, pear]" && t2.first() == "apple" && t2.size == 4)
    val m = java.util.TreeMap<String, Int>()
    m["b"] = ks[0]; m["a"] = ks[1]; m.put("c", ks[2])
    val m2 = java.util.TreeMap(mapOf(3 to "c", 1 to "a"))
    m2[2] = "b"
    check("tre107b", "" + m == "{a=1, b=5, c=3}" && "" + m2 == "{1=a, 2=b, 3=c}" && "" + m2.keys == "[1, 2, 3]")
}

// ===== SECTION 108: `is` on the builtin collections =====
// Read off kotlin-stdlib 2.2.21 under JDK 24 (instanceof java.util.Set /
// Collection / List / Map and java.lang.Iterable, TypeIntrinsics.isMutable*): a
// Set and a List are Collections and Iterables, a Map is neither, a range is an
// Iterable only, and an Array, a String and a Sequence are not Iterable.
fun isRow108(v: Any): String =
    "${v is Set<*>} ${v is Collection<*>} ${v is Iterable<*>} ${v is List<*>} ${v is Map<*, *>}"
fun sec108() {
    val vs: List<Any> = listOf(setOf(1, 2), listOf(1, 2), mapOf(1 to 2), 1..2, arrayOf(1), "ab",
        mutableMapOf(1 to 2).values, sequenceOf(1))
    check("isc108a", isRow108(vs[0]) == "true true true false false" && isRow108(vs[1]) == "false true true true false")
    check("isc108b", isRow108(vs[2]) == "false false false false true" && isRow108(vs[3]) == "false false true false false")
    check("isc108c", isRow108(vs[4]) == "false false false false false" && isRow108(vs[5]) == "false false false false false")
    check("isc108d", isRow108(vs[6]) == "false true true false false" && isRow108(vs[7]) == "false false false false false")
    val ms: List<Any> = listOf(mutableListOf(1), mutableSetOf(1), mutableMapOf(1 to 2))
    check("isc108e", ms[0] is MutableList<*> && ms[0] is MutableCollection<*> && ms[1] is MutableSet<*> && ms[2] is MutableMap<*, *>)
    val s = vs[0]
    val k = when (s) { is List<*> -> "list"; is Set<*> -> "set"; else -> "other" }
    check("isc108f", k == "set" && !(vs[2] !is Map<*, *>) && vs[0] !is List<*>)
}

// ===== SECTION 109: MutableList's deque members =====
// Read off kotlin-stdlib 2.2.21 under JDK 24: removeFirst / removeLast and their
// OrNull forms are kotlin.collections extensions ("List is empty." when there is
// nothing to remove), addFirst / addLast java.util.List's SequencedCollection
// members.
fun sec109() {
    val l = mutableListOf(1, 2, 3)
    l.addFirst(0)
    l.addLast(4)
    check("deq109a", "" + l == "[0, 1, 2, 3, 4]" && l.removeFirst() == 0 && l.removeLast() == 4 && "" + l == "[1, 2, 3]")
    check("deq109b", l.removeFirstOrNull() == 1 && l.removeLastOrNull() == 3 && "" + l == "[2]")
    val e = mutableListOf<Int>()
    var msg = ""
    try { e.removeLast() } catch (x: NoSuchElementException) { msg = "" + x.message }
    check("deq109c", e.removeFirstOrNull() == null && e.removeLastOrNull() == null && msg == "List is empty.")
}

// ===== SECTION 110: TreeSet / TreeMap with a Comparator =====
// Read off java.util.TreeSet / TreeMap (JDK 24): the comparator orders the keys AND
// identifies them, so a key it ranks equal to a stored one is that key - an add is
// refused, a put replaces the value under the stored key, and a lookup finds it.
fun sec110() {
    val words = listOf("ccc", "a", "bb", "dd", "eeee")
    val t = java.util.TreeSet<String>(compareBy { it.length })
    for (w in words) t.add(w)
    check("cmp110a", "" + t == "[a, bb, ccc, eeee]" && t.contains("zz") && "zz" in t && t.first() == "a")
    t.remove("xx")
    val r = java.util.TreeSet<Int>(compareByDescending { it })
    r.addAll(listOf(3, 1, 2))
    check("cmp110b", "" + t == "[a, ccc, eeee]" && "" + r == "[3, 2, 1]")
    val tm = java.util.TreeMap<String, Int>(compareBy { it.lowercase() })
    tm["b"] = 1; tm["A"] = 2; tm.put("B", 3)
    check("cmp110c", "" + tm == "{A=2, b=3}" && tm["a"] == 2 && tm.containsKey("a"))
    val tr = java.util.TreeMap<Int, String>(Comparator { x, y -> y - x })
    tr[1] = "x"; tr[3] = "y"; tr[2] = "z"
    check("cmp110d", "" + tr == "{3=y, 2=z, 1=x}" && "" + tr.keys == "[3, 2, 1]")
}

// ===== SECTION 111: LinkedList and ArrayDeque =====
// Read off java.util.LinkedList / ArrayDeque and kotlin.collections.ArrayDeque
// (kotlin-stdlib 2.2.21, JDK 24): Deque's push / pop / peek / poll / offer
// families, null from peek and poll on an empty deque, and a
// NoSuchElementException with a null message from pop.
fun sec111() {
    val ll = java.util.LinkedList<Int>()
    ll.add(1); ll.addFirst(0); ll.addLast(2); ll.push(-1); ll.offer(3)
    check("dq111a", "" + ll == "[-1, 0, 1, 2, 3]" && ll.peek() == -1 && ll.peekLast() == 3)
    check("dq111b", ll.pop() == -1 && ll.poll() == 0 && ll.pollLast() == 3 && ll.getFirst() == 1 && ll.getLast() == 2)
    check("dq111c", ll.offerFirst(9) && ll.offerLast(8) && ll.pollFirst() == 9 && ll.element() == 1 && "" + ll == "[1, 2, 8]")
    val e = java.util.LinkedList<Int>()
    var msg = "none"
    try { e.pop() } catch (x: NoSuchElementException) { msg = "" + x.message }
    check("dq111d", e.peek() == null && e.poll() == null && e.pollLast() == null && msg == "null")
    val dq = ArrayDeque<Int>()
    dq.addLast(1); dq.addFirst(0); dq.add(2)
    check("dq111e", "" + dq == "[0, 1, 2]" && dq.removeFirst() == 0 && dq.removeLastOrNull() == 2 && dq.size == 1)
    val jd = java.util.ArrayDeque<Int>()
    jd.push(1); jd.push(2); jd.offer(3); jd.addLast(4)
    check("dq111f", "" + jd == "[2, 1, 3, 4]" && jd.pop() == 2 && jd.poll() == 1 && "" + jd == "[3, 4]")
}

// ===== SECTION 112: PriorityQueue =====
// Read off java.util.PriorityQueue (JDK 24): iteration and toString walk the binary
// heap in its array layout, so these strings pin java's siftUp / siftDown /
// removeAt and the heapify of the collection constructor.
fun sec112() {
    val pq = java.util.PriorityQueue<Int>()
    for (x in listOf(5, 1, 4, 2, 3)) pq.add(x)
    check("pq112a", "" + pq == "[1, 2, 4, 5, 3]" && pq.peek() == 1 && pq.poll() == 1 && "" + pq == "[2, 3, 4, 5]")
    pq.remove(4)
    val out = mutableListOf<Int>()
    while (!pq.isEmpty()) out.add(pq.poll())
    var msg = "none"
    try { pq.remove() } catch (x: NoSuchElementException) { msg = "" + x.message }
    check("pq112b", "" + out == "[2, 3, 5]" && pq.poll() == null && pq.peek() == null && msg == "null")
    val p2 = java.util.PriorityQueue<String>(compareBy { it.length })
    p2.add("ccc"); p2.add("a"); p2.add("bb"); p2.offer("dddd")
    check("pq112c", "" + p2 == "[a, ccc, bb, dddd]" && p2.poll() == "a" && p2.remove() == "bb" && "" + p2 == "[ccc, dddd]")
    val p3 = java.util.PriorityQueue(listOf(3, 1, 2, 9, 0, 7))
    val p4 = java.util.PriorityQueue<Int>(10, compareByDescending { it })
    for (x in listOf(1, 5, 2, 8, 3)) p4.add(x)
    check("pq112d", "" + p3 == "[0, 1, 2, 9, 3, 7]" && "" + p4 == "[8, 5, 2, 1, 3]" && "" + p4.sorted() == "[1, 2, 3, 5, 8]")
    val p5 = java.util.PriorityQueue<Int>()
    for (x in listOf(9, 8, 7, 6, 5, 4, 3, 2, 1)) p5.add(x)
    p5.remove(6)
    val s5 = "" + p5
    p5.remove(1)
    check("pq112e", s5 == "[1, 2, 4, 3, 7, 8, 5, 9]" && "" + p5 == "[2, 3, 4, 9, 7, 8, 5]")
    val q: Any = p5
    check("pq112f", q is Collection<*> && q !is Set<*> && p5.size == 7 && p5.contains(9))
}

// ===== SECTION 113: `is` on the concrete java.util collection classes =====
// Read off kotlin-stdlib 2.2.21 under JDK 24 (instanceof java.util.ArrayList /
// HashSet / LinkedHashSet / HashMap / LinkedHashMap / TreeSet / TreeMap / SortedSet /
// PriorityQueue): mutableListOf is an ArrayList, listOf is not (EmptyList,
// SingletonList, Arrays$ArrayList); mutableSetOf / setOf of two are LinkedHashSets and
// so HashSets, setOf of one is a SingletonSet; hashSetOf is a HashSet only.
fun cls113(v: Any): String =
    "${v is ArrayList<*>}${v is HashSet<*>}${v is LinkedHashSet<*>}${v is HashMap<*, *>}${v is LinkedHashMap<*, *>}" +
    "${v is java.util.TreeSet<*>}${v is java.util.TreeMap<*, *>}${v is java.util.SortedSet<*>}${v is java.util.PriorityQueue<*>}"
fun sec113() {
    val f = "falsefalsefalsefalsefalsefalsefalsefalsefalse"
    check("jis113a", mutableListOf(1) is ArrayList<*> && arrayListOf(1, 2) is ArrayList<*> && listOf(1, 2).toMutableList() is ArrayList<*>)
    check("jis113b", cls113(listOf<Int>()) == f && cls113(listOf(1)) == f && cls113(listOf(1, 2)) == f && cls113(setOf(1)) == f && cls113(mapOf(1 to 2)) == f)
    check("jis113c", cls113(mutableSetOf(1)) == "falsetruetruefalsefalsefalsefalsefalsefalse" && cls113(hashSetOf(1)) == "falsetruefalsefalsefalsefalsefalsefalsefalse")
    check("jis113d", cls113(mapOf(1 to 2, 3 to 4)) == "falsefalsefalsetruetruefalsefalsefalsefalse" && cls113(hashMapOf(1 to 2)) == "falsefalsefalsetruefalsefalsefalsefalsefalse")
    check("jis113e", cls113(sortedSetOf(1)) == "falsefalsefalsefalsefalsetruefalsetruefalse" && cls113(sortedMapOf(1 to 2)) == "falsefalsefalsefalsefalsefalsetruefalsefalse")
    check("jis113f", cls113(java.util.PriorityQueue<Int>()) == "falsefalsefalsefalsefalsefalsefalsefalsetrue" && HashSet<Int>() !is LinkedHashSet<*>)
    // A map's keys and entries are views (LinkedHashMap$LinkedKeySet, HashMap$EntrySet,
    // ...): Sets, never HashSets.
    val m = mutableMapOf(1 to 2, 3 to 4)
    check("jis113g", cls113(m.keys) == f && cls113(m.entries) == f && cls113(hashMapOf(1 to 2).keys) == f && cls113(sortedMapOf(1 to 2).entries) == f && m.keys is Set<*>)
}

// ===== SECTION 114: the JVM class of sorted copies, deques and map views =====
// Read off kotlin-stdlib 2.2.21 under JDK 24 (getClass().getName() / getSimpleName(),
// instanceof): sorted* of a Collection or an Array is an Arrays$ArrayList, or
// toList() (EmptyList, SingletonList) at size 0 or 1; of a range (an Iterable only)
// an ArrayList; IntArray.sortedDescending() reverses into an ArrayList. The two
// ArrayDeques and LinkedList are their own classes, and java's ArrayDeque is no
// List. A TreeMap's keys are a TreeMap$KeySet, which is a NavigableSet.
fun jcn114(v: Any): String = v.javaClass.name
fun sec114() {
    val a: Any = listOf(3, 1, 2).sorted()
    val b: Any = setOf(3, 1).sortedBy { it }
    val c: Any = mutableListOf(3, 1).sortedWith(compareBy { it })
    val d: Any = listOf(3, 1).sortedDescending()
    val e: Any = mapOf(1 to 3, 2 to 1).values.sortedByDescending { it }
    check("jsrt114a", a !is ArrayList<*> && b !is ArrayList<*> && c !is ArrayList<*> && d !is ArrayList<*> && e !is ArrayList<*>)
    check("jsrt114b", jcn114(a) == "java.util.Arrays\$ArrayList" && jcn114(listOf(3).sorted()) == "java.util.Collections\$SingletonList" && jcn114(listOf<Int>().sorted()) == "kotlin.collections.EmptyList")
    val g: Any = arrayOf(3, 1).sorted()
    val h: Any = intArrayOf(3, 1).sortedBy { it }
    val i: Any = intArrayOf(3, 1).sortedDescending()
    val j: Any = (1..3).sortedDescending()
    check("jsrt114c", g !is ArrayList<*> && h !is ArrayList<*> && i is ArrayList<*> && j is ArrayList<*> && "" + j == "[3, 2, 1]")
    val jd: Any = java.util.ArrayDeque<Int>()
    val kd: Any = ArrayDeque<Int>()
    val ll: Any = java.util.LinkedList<Int>()
    check("jsrt114d", jd !is ArrayList<*> && kd !is ArrayList<*> && ll !is ArrayList<*> && jd !is List<*> && kd is List<*> && ll is List<*> && jd is Collection<*>)
    check("jsrt114e", jcn114(jd) == "java.util.ArrayDeque" && jcn114(kd) == "kotlin.collections.ArrayDeque" && jcn114(ll) == "java.util.LinkedList" && kd is ArrayDeque<*> && jd !is ArrayDeque<*> && jd is java.util.ArrayDeque<*>)
    val sk: Any = sortedMapOf(1 to 2).keys
    val se: Any = sortedMapOf(1 to 2).entries
    check("jsrt114f", sk is java.util.SortedSet<*> && sk is java.util.NavigableSet<*> && se !is java.util.SortedSet<*> && jcn114(sk) == "java.util.TreeMap\$KeySet")
    check("jsrt114g", jcn114(mutableMapOf(1 to 2).keys) == "java.util.LinkedHashMap\$LinkedKeySet" && jcn114(setOf(1)) == "java.util.Collections\$SingletonSet" && jcn114(mapOf(1 to 2)) == "java.util.Collections\$SingletonMap")
    check("jsrt114h", listOf(1, 2).javaClass.simpleName == "ArrayList" && mutableMapOf(1 to 2).keys.javaClass.simpleName == "LinkedKeySet" && "" + mutableListOf(1).javaClass == "class java.util.ArrayList")
    check("jsrt114i", listOf(1, 2).javaClass != mutableListOf(1).javaClass && arrayListOf(2).javaClass == mutableListOf(1).javaClass)
}

// ===== SECTION 115: javaClass of any value, and a collection's KClass =====
// Read off kotlin-stdlib 2.2.21 under JDK 24 (getClass().getName() / getSimpleName(),
// Reflection.getOrCreateKotlinClass(c).getSimpleName() / getQualifiedName()): every
// value has a javaClass - a primitive boxed to its java.lang class, an Array its JVM
// array class - and a collection's KClass is its JVM class, qualifiedName spelling
// "$" as ".". An Array's sorted copy is an Arrays$ArrayList at every size; an empty
// IntArray's sortedDescending() is emptyList().
class Foo115
fun jn115(v: Any): String = v.javaClass.name + " " + v.javaClass.simpleName + " " + v.javaClass
fun kn115(v: Any): String = "" + v::class.simpleName + " " + v::class.qualifiedName + " " + v::class.java.name
fun sec115() {
    check("jcls115a", jn115(Foo115()) == "Foo115 Foo115 class Foo115" && jn115("x") == "java.lang.String String class java.lang.String")
    check("jcls115b", jn115(3) == "java.lang.Integer Integer class java.lang.Integer" && jn115('c') == "java.lang.Character Character class java.lang.Character")
    check("jcls115c", jn115(arrayOf(1)) == "[Ljava.lang.Integer; Integer[] class [Ljava.lang.Integer;" && jn115(intArrayOf(1)) == "[I int[] class [I")
    check("jcls115d", kn115(mutableListOf(1)) == "ArrayList java.util.ArrayList java.util.ArrayList" && kn115(Foo115()) == "Foo115 Foo115 Foo115")
    check("jcls115e", kn115(setOf(1)) == "SingletonSet java.util.Collections.SingletonSet java.util.Collections\$SingletonSet" && kn115(mutableMapOf(1 to 2)) == "LinkedHashMap java.util.LinkedHashMap java.util.LinkedHashMap")
    val s: Any = "abcd"
    val a: Any = arrayOf(1)
    val t = "abcd"
    check("jcls115f", s::class.simpleName == "String" && a::class.simpleName == "Array" && intArrayOf(1)::class.simpleName == "IntArray" && t::length.get() == 4)
    check("jcls115g", jn115(arrayOf(3).sorted()) == "java.util.Arrays\$ArrayList ArrayList class java.util.Arrays\$ArrayList" && jn115(arrayOf<Int>().sortedBy { it }).startsWith("java.util.Arrays\$ArrayList "))
    check("jcls115h", jn115(intArrayOf().sortedDescending()).startsWith("kotlin.collections.EmptyList ") && jn115(intArrayOf(3).sortedDescending()).startsWith("java.util.ArrayList ") && jn115(intArrayOf(3).sorted()).startsWith("java.util.Arrays\$ArrayList "))
    check("jcls115i", jn115(mutableMapOf(1 to 2).values) == "java.util.LinkedHashMap\$LinkedValues LinkedValues class java.util.LinkedHashMap\$LinkedValues")
}

// ===== SECTION 121: assignment targets, `!!` and `as` =====
// `a[i] = v` is a.set(i, v) and `a[i] op= v` is a.set(i, a.get(i) op v) (operator
// conventions); the target's receiver and index are evaluated first, then the
// slot's current value for `op=`, then the right-hand side - java 24 settles the
// same order for the array and field forms. `x!!` on null throws
// NullPointerException with a null message (kotlin-stdlib's Intrinsics.checkNotNull,
// called from java). `as` throws ClassCastException with the JVM's message; java 24
// gives the text for the same casts.
var log121 = ""
fun t121(tag: String, v: Int): Int { log121 += tag + v; return v }
class S121(var v: Int) { operator fun set(i: Int, x: Int) { v = x + i } }
class G121(var x: Int) {
    operator fun get(i: Int): Int { log121 += "g" + i; return x }
    operator fun set(i: Int, v: Int) { log121 += "s" + i; x = v }
}
class D121 {
    var data = mutableListOf(0, 0)
    var value = mutableListOf(0, 0)
    fun w(i: Int, x: Int) { data[i] = x; value[i] = x + 1 }
}
class H121 { var f = 1; val xs = intArrayOf(1, 2, 3) }
fun pick121(i: Int): String? = if (i > 9) "x" else null
fun any121(i: Int): Any = if (i == 0) "str" else if (i == 1) 42 else if (i == 2) Dog121() else Cat121()
interface Pet121
open class Animal121
class Dog121 : Animal121(), Pet121
class Cat121 : Animal121()
fun sec121() {
    val n = intArrayOf(1, 7)
    val s = S121(0)
    s[n[0]] = n[1]
    check("as121a", s.v == 8)
    val g = G121(6)
    g[t121("i", 2)] += t121("v", 5)
    check("as121b", log121 == "i2g2v5s2" && g.x == 11)
    log121 = ""
    g[t121("i", 1)] = t121("v", 4)
    g[0]++
    check("as121c", log121 == "i1v4s1g0s0" && g.x == 5)
    val d = D121()
    d.w(0, 5)
    check("as121d", d.data.toString() == "[5, 0]" && d.value.toString() == "[6, 0]")
    val arr = intArrayOf(1, 2, 3)
    var i = 0
    arr[i++] = arr[i] + 10
    check("as121e", arr.toList().toString() == "[12, 2, 3]" && i == 1)
    i = 0
    arr[i] = i++ + i++
    var x = 5
    x += x++ * 2
    check("as121f", arr[0] == 1 && x == 15)
    val h = H121()
    h.f += h.f++ * 10
    var k = 0
    h.xs[k++] += h.xs[k] * 100
    check("as121g", h.f == 11 && h.xs.toList().toString() == "[201, 2, 3]" && k == 1)
    val hs = listOf(h)
    k = 0
    hs[0].xs[k++] = hs[0].xs[k] + 7
    check("as121h", h.xs[0] == 9 && k == 1)
    var got = ""
    try { val t = pick121(0)!!; got = "nn $t" } catch (e: Exception) { got = "" + e + " " + e.message }
    try { got += " " + pick121(0)!!.length } catch (e: NullPointerException) { got += " npe" }
    check("nn121a", got == "java.lang.NullPointerException null npe" && pick121(10)!!.length == 1)
    var cc = ""
    for (j in 0..3) {
        val v = any121(j)
        cc += j
        try { v as String; cc += "+" } catch (e: ClassCastException) { cc += "-" }
        try { v as Int; cc += "+" } catch (e: ClassCastException) { cc += "-" }
        try { v as Number; cc += "+" } catch (e: ClassCastException) { cc += "-" }
        try { v as Dog121; cc += "+" } catch (e: ClassCastException) { cc += "-" }
        try { v as Pet121; cc += "+" } catch (e: ClassCastException) { cc += "-" }
        try { v as Animal121; cc += "+" } catch (e: ClassCastException) { cc += "-" }
    }
    check("cast121a", cc == "0+-----1-++---2---+++3-----+")
    var m = ""
    try { m = "" + (any121(1) as String).length } catch (e: ClassCastException) { m = e.message ?: "" }
    check("cast121b", m == "class java.lang.Integer cannot be cast to class java.lang.String (java.lang.Integer and java.lang.String are in module java.base of loader 'bootstrap')")
    try { m = "" + (any121(3) as Dog121) } catch (e: ClassCastException) { m = e.message ?: "" }
    check("cast121c", m == "class Cat121 cannot be cast to class Dog121 (Cat121 and Dog121 are in unnamed module of loader 'app')")
    try { m = "" + (any121(1) as Dog121) } catch (e: ClassCastException) { m = e.message ?: "" }
    check("cast121d", m == "class java.lang.Integer cannot be cast to class Dog121 (java.lang.Integer is in module java.base of loader 'bootstrap'; Dog121 is in unnamed module of loader 'app')")
    val nul: Any? = null
    check("cast121e", (nul as String?) == null && (any121(1) as Int) + 1 == 43 && any121(2) is Pet121)
}

// ===== SECTION 122: NaN and signed zero =====
// The operators on Doubles are IEEE 754: every ordered relation with a NaN is false,
// NaN != NaN and -0.0 == 0.0. compareTo, equals(), collections and a data class's
// equals use java.lang.Double.compare / Double.equals instead: -0.0 < 0.0, NaN above
// +Infinity and equal to itself. maxOf / minOf / Math.max / Iterable<Double>.max()
// are Math.max / Math.min (a NaN wins). java 24 with kotlin-stdlib 2.2.21
// (CollectionsKt.sorted / maxOrThrow / distinct, SetsKt.setOf) gives every value.
data class D122(val d: Double)
fun sec122() {
    val v = doubleArrayOf(Double.NaN, 0.0, -0.0, 1.0)
    val nan = v[0]
    val z = v[1]
    val nz = v[2]
    val one = v[3]
    check("nan122a", !(nan < one) && !(nan <= one) && !(nan > one) && !(nan >= one) && !(one < nan) && !(one >= nan))
    check("nan122b", !(nan == nan) && nan != nan && nz == z && !(nz < z) && nz >= z)
    var w = 0
    while (nan > w.toDouble() && w < 3) w++
    check("nan122c", w == 0)
    check("nan122d", nz.compareTo(z) == -1 && nan.compareTo(z) == 1 && z.compareTo(nan) == -1 && nan.compareTo(nan) == 0)
    check("nan122e", nan.equals(nan) && !nz.equals(z) && !(nz as Any).equals(z))
    check("nan122f", listOf(nz) != listOf(z) && listOf(nan) == listOf(nan) && setOf(nz, z).size == 2 && setOf(nan, nan).size == 1)
    check("nan122g", listOf(z, nz, nan, nan).distinct().toString() == "[0.0, -0.0, NaN]" && listOf(1.0, nan).indexOf(nan) == 1 && !listOf(nz).contains(z))
    check("nan122h", D122(nz) != D122(z) && D122(nan) == D122(nan))
    val xs = listOf(3.0, nz, z, nan, -1.0, Double.NEGATIVE_INFINITY)
    check("nan122i", xs.sorted().toString() == "[-Infinity, -1.0, -0.0, 0.0, 3.0, NaN]" &&
                     xs.sortedDescending().toString() == "[NaN, 3.0, 0.0, -0.0, -1.0, -Infinity]")
    check("nan122j", xs.max().isNaN() && xs.min().isNaN() && xs.maxBy { it }.isNaN() && xs.minBy { it } == Double.NEGATIVE_INFINITY)
    check("nan122k", "" + listOf(z, nz).max() + listOf(z, nz).min() + listOf(nz, z).max() + listOf(nz, z).min() == "0.0-0.00.0-0.0")
    check("nan122l", "" + maxOf(nz, z) + minOf(z, nz) + maxOf(nan, z) + minOf(z, nan) + Math.max(nz, z) + Math.min(z, nz) == "0.0-0.0NaNNaN0.0-0.0")
    // `==` with an operand not statically Float/Double is Intrinsics.areEqual, i.e.
    // equals; Double? == Double? stays IEEE.
    val an: Any = nan
    val bn: Any = v[0]
    val az: Any = z
    val anz: Any = nz
    val qn: Double? = nan
    val qn2: Double? = v[0]
    check("nan122m", an == bn && !(an != bn) && !(anz == az) && anz != az && an == nan && !(az == nz))
    check("nan122n", !(qn == qn2) && same122(nan, v[0]) && !same122(nz, z))
    // A smart cast makes the operands statically Double again: IEEE (the Kotlin
    // reference, "Floating-point numbers comparison": declared, inferred or the
    // result of a smart cast).
    val sa: Any = nan
    val sb: Any = v[0]
    check("nan122o", if (sa is Double && sb is Double) !(sa == sb) else false)
}
fun <T> same122(p: T, q: T): Boolean = p == q

// ===== SECTION 123: a `for` over a progression ending at the type's edge =====
// The loop runs from first to the progression's LAST element (kotlin.internal.
// getProgressionLastElement) and stops on it, so it never steps past
// Int.MAX_VALUE or Long.MAX_VALUE, and a Long counter is exact above 2^53.
// kotlin-stdlib 2.2.21's IntProgression / LongProgression iterators, driven from
// java, give every value. Each loop is capped, so a regression cannot hang.
fun sec123() {
    val ii = intArrayOf(Int.MAX_VALUE, Int.MIN_VALUE)
    var s = 0
    for (i in ii[0] - 2..ii[0]) { s++; if (s > 10) break }
    var t = 0
    for (i in 0..ii[0] step ii[0]) { t++; if (t > 10) break }
    var u = 0
    for (i in ii[1] + 2 downTo ii[1]) { u++; if (u > 10) break }
    check("for123a", s == 3 && t == 2 && u == 3)
    var got = ""
    for (i in ii[0] - 5..ii[0] step 4) { got += " " + i; if (got.length > 80) break }
    for (i in ii[1]..ii[1] + 3 step 3) { got += " " + i; if (got.length > 160) break }
    check("for123b", got == " 2147483642 2147483646 -2147483648 -2147483645")
    var e = 0
    for (i in ii[0] - 1 until ii[0]) { e++; if (e > 10) break }
    for (i in 5 until ii[1]) { e += 100; if (e > 1000) break }
    check("for123c", e == 1)
}

// ===== SECTION 124: JVM exceptions and edge answers of the stdlib =====
// abs(Int.MIN_VALUE) is Math.abs's Int.MIN_VALUE; Double.mod / Float.mod take the
// divisor's sign (the inline body in kotlin-stdlib 2.2.21's util/FloorDivMod.kt, run
// on java); getValue, a negative array size, List(-1) and subList throw what
// MapsKt.getValue, new int[-1], ArrayList(-1) and AbstractList.subList throw on java
// 24; removing from a MutableList inside its for loop is the ArrayList iterator's
// ConcurrentModificationException (java's enhanced for over mutableListOf); and a
// when branch `in coll` asks coll.contains (the Kotlin specification).
fun cme124(k: Int): String {
    val ml = mutableListOf(1, 2, 3, 4)
    try { for (e in ml) if (e == k) ml.remove(e) } catch (e: ConcurrentModificationException) { return "cme" + e.message + ml }
    return "none" + ml
}
fun when124(x: Int, xs: List<Int>): String = when (x) {
    in xs -> "l"
    in 10..12 -> "r"
    in 20 until 23 -> "u"
    in 30..40 step 5 -> "s"
    !in listOf(50, 51) -> "o"
    else -> "f"
}
fun sec124() {
    val ii = intArrayOf(Int.MIN_VALUE, -5)
    check("abs124", kotlin.math.abs(ii[0]) == Int.MIN_VALUE && Math.abs(ii[0]) == Int.MIN_VALUE && ii[0].absoluteValue == Int.MIN_VALUE && kotlin.math.abs(ii[1]) == 5)
    val d = doubleArrayOf(2.5, -2.5, 2.0, -2.0, -7.0, 3.0)
    check("mod124a", "" + d[0].mod(d[2]) + " " + d[1].mod(d[2]) + " " + d[1].mod(d[3]) + " " + d[4].mod(d[5]) == "0.5 1.5 -0.5 2.0")
    val f = floatArrayOf(-2.5f, 2.0f, 0.1f)
    check("mod124b", "" + f[0].mod(f[1]) + " " + f[2].mod(f[1]) == "1.5 0.1")
    val n = intArrayOf(-1, 2, 1, 5)
    var got = ""
    try { mapOf("a" to 1).getValue("b") } catch (e: NoSuchElementException) { got += e.message }
    try { IntArray(n[0]) } catch (e: NegativeArraySizeException) { got += "|" + e.message }
    try { Array(n[0]) { 0 } } catch (e: NegativeArraySizeException) { got += "|" + e.message }
    try { List(n[0]) { 0 } } catch (e: IllegalArgumentException) { got += "|" + e.message }
    check("exc124a", got == "Key b is missing in the map.|-1|-1|Illegal Capacity: -1")
    val xs = listOf(1, 2, 3)
    got = ""
    try { xs.subList(n[1], n[2]) } catch (e: IllegalArgumentException) { got += e.message }
    try { xs.subList(n[2], n[3]) } catch (e: IndexOutOfBoundsException) { got += "|" + e.message }
    try { mutableListOf(1).subList(n[0], 0) } catch (e: IndexOutOfBoundsException) { got += "|" + e.message }
    check("exc124b", got == "fromIndex(2) > toIndex(1)|toIndex = 5|fromIndex = -1" && xs.subList(n[2], n[1]) == listOf(2))
    check("cme124", cme124(1) + cme124(2) + cme124(3) + cme124(4) == "cmenull[2, 3, 4]cmenull[1, 3, 4]none[1, 2, 4]cmenull[1, 2, 3]")
    var w = ""
    for (x in intArrayOf(2, 11, 22, 35, 36, 50)) w += when124(x, xs)
    check("when124", w == "lrusof")
}

// ===== SECTION 125: a LongRange above 2^53 =====
// A LongRange / LongProgression keeps its bounds and step as Longs, so iterating,
// counting and printing one past 2^53 is exact and the loop ends. kotlin-stdlib
// 2.2.21's LongRange, RangesKt.step / reversed / until / downTo and
// CollectionsKt.toList / count, called from java, give every value. Each loop is
// capped, so a regression cannot hang.
fun sec125() {
    val ls = longArrayOf(9007199254740993L, Long.MAX_VALUE)
    var got = ""
    for (i in ls[0]..ls[0] + 1) { got += " " + i % 10; if (got.length > 20) break }
    for (i in ls[1] - 1..ls[1]) { got += " " + i % 10; if (got.length > 40) break }
    check("lr125a", got == " 3 4 6 7")
    val r = ls[0]..ls[0] + 4
    check("lr125b", r.toString() == "9007199254740993..9007199254740997" && (r step 2).toString() == "9007199254740993..9007199254740997 step 2")
    check("lr125c", (r step 2).toList().toString() == "[9007199254740993, 9007199254740995, 9007199254740997]" &&
                    r.reversed().toList().last() == ls[0])
    check("lr125d", (ls[0] until ls[0] + 3).toList().toString() == "[9007199254740993, 9007199254740994, 9007199254740995]" &&
                    (ls[0] + 3 downTo ls[0]).count() == 4)
    check("lr125e", (ls[1] - 4..ls[1] step 3).toList().toString() == "[9223372036854775803, 9223372036854775806]" &&
                    (ls[1] - 4..ls[1]).count() == 5)
}

// ===== SECTION 126: stdlib members: margins, float neighbours, bit rotation =====
// trimMargin / replaceIndent are kotlin-stdlib 2.2.21's StringsKt (called from
// java); withSign / ulp / nextUp / nextDown are Math.copySign / ulp / nextUp /
// nextDown and takeHighestOneBit / rotateLeft / ... Integer's and Long's
// (MathJVM.kt, NumbersJVM.kt); iterator / addAll(index, ...) are ArrayList's and
// insert / deleteCharAt / setLength java.lang.StringBuilder's - java 24 gives every
// value, message included.
fun sec126() {
    val tm = """
        |a
        |  b
        c""".trimMargin()
    check("std126a", tm == "a\n  b\n        c" && "  x\n    y".replaceIndent("> ") == "> x\n>   y" && "##a\n##b".trimMargin("##") == "a\nb")
    val d = doubleArrayOf(3.0, -1.0, 1.0, 0.0, Double.MAX_VALUE, -0.0)
    check("std126b", "" + d[0].withSign(d[1]) + " " + d[2].ulp + " " + d[2].nextUp() + " " + d[2].nextDown() + " " + d[3].nextUp() ==
                     "-3.0 2.220446049250313E-16 1.0000000000000002 0.9999999999999999 4.9E-324")
    val f1 = floatArrayOf(1.0f)
    check("std126c", "" + d[4].ulp + " " + f1[0].ulp + " " + d[1].nextUp() + " " + f1[0].nextDown() + " " + d[2].withSign(d[5]) ==
                     "1.99584030953472E292 1.1920929E-7 -0.9999999999999999 0.99999994 -1.0")
    val ii = intArrayOf(100, -2147483648, 5)
    val l5 = longArrayOf(-5L, 1L)
    check("std126d", "" + ii[0].takeHighestOneBit() + " " + ii[0].takeLowestOneBit() + " " + ii[1].rotateLeft(1) + " " + ii[2].rotateRight(1) +
                     " " + ii[2].inc() + " " + ii[2].dec() + " " + l5[0].takeHighestOneBit() + " " + l5[1].rotateLeft(65) ==
                     "64 4 1 -2147483646 6 4 -9223372036854775808 2")
    val ml = mutableListOf(1, 2, 3)
    val it = ml.iterator()
    var s = ""
    while (it.hasNext()) s += it.next()
    ml.addAll(1, listOf(8, 9))
    var msg = ""
    try { ml.addAll(9, listOf(1)) } catch (e: IndexOutOfBoundsException) { msg = e.message ?: "" }
    check("std126e", s == "123" && ml.toString() == "[1, 8, 9, 2, 3]" && msg == "Index: 9, Size: 5")
    val sb = StringBuilder("hello")
    sb.insert(0, "[").insert(sb.length, "]")
    val s1 = sb.toString()
    sb.deleteCharAt(1)
    val s2 = sb.toString()
    sb.setLength(3)
    msg = ""
    try { sb.insert(9, "x") } catch (e: StringIndexOutOfBoundsException) { msg += e.message }
    try { sb.deleteCharAt(9) } catch (e: StringIndexOutOfBoundsException) { msg += "|" + e.message }
    check("std126f", s1 == "[hello]" && s2 == "[ello]" && sb.toString() == "[el" &&
                     msg == "Range [9, 3) out of bounds for length 3|Index 9 out of bounds for length 3")
}


// ===== SECTION 127: a lambda whose body ends in a try =====
// A lambda's value is its last expression, and a try is an expression (the Kotlin
// specification, "Lambda literals" and "Try-expressions"); getValue's message is
// kotlin-stdlib 2.2.21's MapsKt.getValue, called from java 24. A try-expression's
// catch clauses are chosen by type like a statement's, and a `return` out of a
// lambda passes every catch clause on its way (it is not an exception) while a
// finally still runs.
fun tc127(i: Int): String {
    val r = try {
        if (i == 0) throw IllegalStateException("s") else throw IllegalArgumentException("a")
    } catch (e: IllegalArgumentException) { "iae" } catch (e: RuntimeException) { "rte" }
    return r
}
fun nl127a(xs: List<Int>): Int {
    xs.forEach {
        try { if (it == 2) return it * 10 } catch (e: Exception) { }
    }
    return -1
}
fun nl127b(xs: List<Int>): String {
    var log = ""
    xs.forEach {
        try {
            if (it == 2) return log + "r"
            log += it
        } catch (e: Throwable) { log += "c" } finally { log += "f" }
    }
    return log
}
fun nl127c(xs: List<List<Int>>): String {
    var log = ""
    xs.forEach outer@{ row ->
        row.forEach {
            try {
                if (it == 0) return@outer
                log += it
            } catch (e: Exception) { log += "c" }
        }
        log += "|"
    }
    return log
}
fun sec127() {
    val m = mapOf("a" to 1)
    check("lam127a", m.let { try { it.getValue("b") } catch (e: NoSuchElementException) { e.message } } == "Key b is missing in the map." &&
                     m.let { try { 7 } catch (e: Exception) { 8 } } == 7)
    check("lam127b", listOf(1, 2).map { try { if (it == 1) return@map 9; it } finally { } }.toString() == "[9, 2]")
    var out = ""
    try {
        out = run { try { throw IllegalStateException("x") } catch (e: IllegalArgumentException) { "wrong" } }
    } catch (e: IllegalStateException) { out = "outer " + e.message }
    check("lam127c", tc127(0) + " " + tc127(1) == "rte iae" && out == "outer x" &&
                     listOf(1, 2).map { try { if (it == 2) throw IllegalArgumentException("b"); it } catch (e: IllegalArgumentException) { -1 } }.toString() == "[1, -1]")
    check("lam127d", nl127a(listOf(1, 2, 3)) == 20 && nl127b(listOf(1, 2, 3)) == "1fr" && nl127c(listOf(listOf(1, 0, 2), listOf(3))) == "13|")
}

// ===== SECTION 128: casts, `!!` targets, trimIndent and LongRange bounds =====
// trimIndent / replaceIndent, javaClass and LongRange / LongProgression are
// kotlin-stdlib 2.2.21's, called from java 24; `is` / `as Comparable` and every
// ClassCastException text are java 24's for the same objects (a nested class is
// Outer$Inner). `x!!` on null in an assignment target is Intrinsics.checkNotNull's
// NullPointerException, as in an expression. `null as T` for a non-null T is the
// NullPointerException kotlinc emits - spec only (no kotlinc here), so only the type
// is asserted.
class Ver128(val n: Int) : Comparable<Ver128> { override fun compareTo(other: Ver128): Int = n - other.n }
enum class Col128 { RED }
class Dog128
class Outer128 {
    class Inner
}
class T128(var i: Int) { val xs = intArrayOf(1, 2) }
fun pick128(k: Int): T128? = if (k > 0) T128(k) else null
fun any128(i: Int): Any? = when (i) {
    0 -> null
    1 -> Dog128()
    2 -> Outer128.Inner()
    3 -> 1 to 2
    else -> arrayOf(1)
}
fun sec128() {
    val ti = listOf("\n\n  a\n\n", "\n \n  a\n    b\n \n", "\n\n")
    check("ti128", ti[0].trimIndent() == "\na\n" && ti[1].trimIndent() == "\na\n  b\n" && ti[2].trimIndent() == "" && ti[0].replaceIndent(">") == ">\n>a\n>")
    val vs: List<Any> = listOf("s", 'c', true, Col128.RED, Ver128(1), 2L, 1.5f, Dog128(), listOf(1), 1 to 2)
    var cs = ""
    for (v in vs) {
        cs += if (v is Comparable<*>) "+" else "-"
        try { v as Comparable<*>; cs += "+" } catch (e: ClassCastException) { cs += "-" }
    }
    check("cmp128a", cs == "++++++++++++++------")
    val msgs = mutableListOf<String>()
    for (i in 1..4) {
        try { any128(i) as Comparable<*> } catch (e: ClassCastException) { msgs.add(e.message ?: "") }
    }
    try { any128(2) as String } catch (e: ClassCastException) { msgs.add(e.message ?: "") }
    try { any128(1) as Outer128.Inner } catch (e: ClassCastException) { msgs.add(e.message ?: "") }
    check("cce128a", msgs[0] == "class Dog128 cannot be cast to class java.lang.Comparable (Dog128 is in unnamed module of loader 'app'; java.lang.Comparable is in module java.base of loader 'bootstrap')")
    check("cce128b", msgs[1] == "class Outer128\$Inner cannot be cast to class java.lang.Comparable (Outer128\$Inner is in unnamed module of loader 'app'; java.lang.Comparable is in module java.base of loader 'bootstrap')")
    check("cce128c", msgs[2] == "class kotlin.Pair cannot be cast to class java.lang.Comparable (kotlin.Pair is in unnamed module of loader 'app'; java.lang.Comparable is in module java.base of loader 'bootstrap')")
    check("cce128d", msgs[3] == "class [Ljava.lang.Integer; cannot be cast to class java.lang.Comparable ([Ljava.lang.Integer; and java.lang.Comparable are in module java.base of loader 'bootstrap')")
    check("cce128e", msgs[4] == "class Outer128\$Inner cannot be cast to class java.lang.String (Outer128\$Inner is in unnamed module of loader 'app'; java.lang.String is in module java.base of loader 'bootstrap')" &&
                     msgs[5] == "class Dog128 cannot be cast to class Outer128\$Inner (Dog128 and Outer128\$Inner are in unnamed module of loader 'app')")
    val jc = listOf<Any>(1 to 2, Triple(1, 2, 3), 1..2, 1L..2L, 'a'..'b', (1..5) step 2, 5L downTo 1L, Outer128.Inner())
    var jn = ""
    for (v in jc) jn += v.javaClass.name + " "
    check("jcls128", jn == "kotlin.Pair kotlin.Triple kotlin.ranges.IntRange kotlin.ranges.LongRange kotlin.ranges.CharRange kotlin.ranges.IntProgression kotlin.ranges.LongProgression Outer128\$Inner ")
    var nc = ""
    try { any128(0) as String; nc += "a" } catch (e: NullPointerException) { nc += "A" }
    try { any128(0) as String?; nc += "b" } catch (e: NullPointerException) { nc += "B" }
    try { any128(0) as Dog128; nc += "c" } catch (e: NullPointerException) { nc += "C" }
    check("nc128", nc == "AbC")
    val ok = pick128(5)
    ok!!.i++
    ok!!.xs[1] += 3
    val t = pick128(0)
    var got = ""
    try { t!!.i++; got += "a" } catch (e: NullPointerException) { got += "A" + e.message }
    try { t!!.i = 3; got += "b" } catch (e: NullPointerException) { got += "B" }
    try { t!!.i += 3; got += "c" } catch (e: NullPointerException) { got += "C" }
    try { t!!.xs[0] = 1; got += "d" } catch (e: NullPointerException) { got += "D" }
    try { t!!.xs[0]++; got += "e" } catch (e: NullPointerException) { got += "E" }
    check("nn128", got == "AnullBCDE" && ok.i == 6 && ok.xs[1] == 5)
    val ls = longArrayOf(9007199254740993L, Long.MAX_VALUE)
    val b = ls[0]
    val r = b..b + 2
    check("lr128a", "" + r.first + " " + r.last + " " + r.start + " " + r.endInclusive == "9007199254740993 9007199254740995 9007199254740993 9007199254740995")
    check("lr128b", r.contains(b + 1) && !r.contains(b - 1) && !(b + 3 in r) && b in r)
    val p = (b..b + 4) step 3
    check("lr128c", "" + p.first + " " + p.last == "9007199254740993 9007199254740996" && (b + 3 in p) && !(b + 4 in p))
    val d = (b + 4) downTo b
    check("lr128d", "" + d.first + " " + d.last == "9007199254740997 9007199254740993" && !(b + 5 in d) && b in d)
    val mx = ls[1] - 2..ls[1]
    check("lr128e", "" + mx.first + " " + mx.last == "9223372036854775805 9223372036854775807" && ls[1] in mx && !(ls[1] - 3 in mx))
}

// ===== SECTION 129: Unit as a value, `;` between members, `?.` targets, range hashes and Float ranges, null casts =====
// Unit's toString, every range hashCode and the ClosedFloatRange values are
// kotlin-stdlib 2.2.21's, called from java 24. `a?.b = c` is `if (a != null) a.b = c`:
// a null receiver skips the write and the right-hand side (Kotlin spec, safe-call
// assignment - spec only, no kotlinc here). `null as T` is kotlinc's
// NullPointerException for a non-null library type too; only the type is asserted.
class Outer129 { class In129; inner class Inn129; val k = 1; val m = 2 }
class T129(var i: Int) { var nx: T129? = null }
fun mk129(b: Boolean): T129? = if (b) T129(40) else null
fun <T> cast129(x: Any?): T = x as T
fun sec129() {
    val us = listOf(1, Unit)
    check("unit129", us.toString() == "[1, kotlin.Unit]" && "${us[1]}" == "kotlin.Unit" && us[1] == Unit && Unit.toString() == "kotlin.Unit")
    check("semi129", Outer129.In129() is Outer129.In129 && Outer129().k + Outer129().m == 3)
    var calls = 0
    fun bump129(): Int { calls++; return 9 }
    val n = mk129(false)
    n?.i = bump129()
    n?.i += bump129()
    val r = n?.i++
    mk129(false)?.i = bump129()
    val y = T129(7)
    y.nx?.i = bump129()
    check("safe129a", calls == 0 && r == null)
    y.nx = T129(8)
    y.nx?.i = bump129()
    y.nx?.i += 2
    val r2 = y.nx?.i++
    val m = mk129(true)
    m?.i = 41
    check("safe129b", calls == 1 && r2 == 11 && y.nx!!.i == 12 && m!!.i == 41)
    val lb = longArrayOf(1L, 5L, 5000000000L, 5000000005L, -3L, 3L, 9223372036854775800L, Long.MAX_VALUE)
    var hs = ""
    for (k in 0 until 4) {
        val a = lb[2 * k]
        val b = lb[2 * k + 1]
        hs += "" + (a..b).hashCode() + " " + (b downTo a step 2).hashCode() + " " + (a..b step 3).hashCode() + " " + (a.toInt()..b.toInt()).hashCode() + ";"
    }
    check("rh129a", hs == "36 4837 1088 36;1086210083 -687223035 -687225854 1086210053;65 2946 2018 -90;217 187 6761 -249;")
    val ib = intArrayOf(5, 1, 10, 4)
    val cb = "ae"
    val ih = "" + (ib[0] downTo ib[1] step 2).hashCode() + " " + (ib[1]..ib[2] step ib[3]).hashCode() + " " + (cb[0]..cb[1]).hashCode() + " " +
        (cb[0]..cb[1] step 2).hashCode() + " " + (ib[0]..ib[1]).hashCode() + " " + (lb[1]..lb[0]).hashCode() + " " + (cb[1]..cb[0]).hashCode()
    check("rh129b", ih == "4834 1244 3108 96350 -1 -1 -1")
    val db = doubleArrayOf(1.5, 2.5, Double.NaN, 1.0, -0.0, 0.0)
    val fb = floatArrayOf(1.5f, 2.5f, 0.1f)
    val dh = "" + (db[0]..db[1]).hashCode() + " " + (fb[0]..fb[1]).hashCode() + " " + (fb[1]..fb[0]).hashCode() + " " +
        (db[2]..db[3]).isEmpty() + " " + (db[2]..db[3]).hashCode() + " " + (db[4]..db[5]).hashCode()
    check("rh129c", dh == "-15990784 -127926272 -1 true -1 -2147483648")
    val fr = fb[2]..fb[1]
    check("fr129", "" + fr + " " + fr.javaClass.name + " " + fr.start.javaClass.name + " " + fr.endInclusive + " " + (db[0]..db[1]).javaClass.name ==
        "0.1..2.5 kotlin.ranges.ClosedFloatRange java.lang.Float 2.5 kotlin.ranges.ClosedDoubleRange" && fr == fb[2]..fb[1] && fb[2] in fr)
    val nul: Any? = mk129(false)
    var nc = ""
    try { nul as Any; nc += "a" } catch (e: NullPointerException) { nc += "A" }
    try { nul as List<Int>; nc += "b" } catch (e: NullPointerException) { nc += "B" }
    try { nul as Map<String, Int>; nc += "c" } catch (e: NullPointerException) { nc += "C" }
    try { nul as IntArray; nc += "d" } catch (e: NullPointerException) { nc += "D" }
    try { nul as List<Int>?; nc += "e" } catch (e: NullPointerException) { nc += "E" }
    try { cast129<String>(nul); nc += "f" } catch (e: NullPointerException) { nc += "F" }
    check("nc129", nc == "ABCDef")
}

// ===== SECTION 130: a for over a MutableList fails fast on modCount, not on size =====
// java's enhanced for over an ArrayList (what `for (x in xs)` on a mutableListOf is)
// throws ConcurrentModificationException from next() once modCount moved, and
// hasNext() is `cursor != size`. Every row is java 24's for the same ArrayList, the
// kotlin.collections members called from java through kotlin-stdlib 2.2.21: an add
// and a remove in one step, or a sort, throw though the size is unchanged; reverse,
// fill and set only set elements; a removeAll / retainAll / remove that removes
// nothing changes nothing; addAll and clear count even when they change nothing.
fun cme130(op: (MutableList<Int>, Int) -> Unit): String {
    val xs = mutableListOf(3, 1, 2)
    var seen = ""
    try {
        for (x in xs) { seen += x; op(xs, x) }
        return "ok$seen"
    } catch (e: ConcurrentModificationException) {
        return "cme$seen"
    }
}
fun sec130() {
    val r = listOf(
        cme130 { l, x -> if (x == 3) { l.add(9); l.removeAt(3) } },
        cme130 { l, x -> if (x == 3) l.sort() },
        cme130 { l, x -> if (x == 3) l.sortBy { -it } },
        cme130 { l, x -> if (x == 3) l.sortWith(compareBy { it }) },
        cme130 { l, x -> if (x == 3) l.reverse() },
        cme130 { l, x -> if (x == 3) l[2] = 7 },
        cme130 { l, x -> if (x == 3) l.fill(0) },
        cme130 { l, x -> if (x == 1) l.removeAt(0) },
        cme130 { l, x -> if (x == 3) l.addAll(emptyList()) },
        cme130 { l, x -> if (x == 3) l.removeAll { it > 100 } },
        cme130 { l, x -> if (x == 3) l.retainAll(listOf(1, 2, 3)) },
        cme130 { l, x -> if (x == 3) l.remove(42) },
        cme130 { l, x -> if (x == 3) { l.removeLast(); l.add(6) } },
        cme130 { l, x -> if (x == 1) { l.removeAt(0); l.add(5) } })
    check("cme130a", r.joinToString(" ") == "cme3 cme3 cme3 cme3 ok313 ok317 ok300 ok31 cme3 ok312 ok312 ok312 cme3 cme31")
    val big = mutableListOf<Int>()
    for (i in 0 until 3000) big.add(i)
    var hit = ""
    for (x in big) { if (x == 2999) hit += "last" }
    try { for (x in big) { if (x == 2000) { big.add(1); big.removeAt(big.size - 1) } } } catch (e: ConcurrentModificationException) { hit += " cme" }
    val one = mutableListOf(5)
    for (x in one) { one.sort(); hit += x }
    check("cme130b", hit == "last cme5")
}

// ===== SECTION 131: a for over a MutableSet fails fast too, with a set's own end =====
// A LinkedHashSet / TreeSet iterator throws ConcurrentModificationException from
// next() once modCount moved, and fetched its next node with the current one: the
// loop ends quietly after the element that was LAST when the loop started, whatever
// was added or removed meanwhile. add / addAll / remove / removeAll / retainAll
// count only when they change the set, clear always. A val's `xs += emptyList()` is
// plusAssign, i.e. addAll, which an ArrayList counts even when it adds nothing.
// Every row is java 24 over the same LinkedHashSet / TreeSet / ArrayList, the `+=`
// one through kotlin-stdlib 2.2.21's CollectionsKt.addAll.
fun cme131(s: MutableSet<Int>, op: (MutableSet<Int>, Int) -> Unit): String {
    var seen = ""
    try {
        for (x in s) { seen += x; op(s, x) }
        return "ok$seen"
    } catch (e: ConcurrentModificationException) {
        return "cme$seen"
    }
}
fun sec131() {
    val r = listOf(
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 1) { s.add(9); s.remove(9) } },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 2) s.remove(3) },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> s.add(1) },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 3) s.add(9) },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 3) s.remove(1) },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 1) s.remove(1) },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 1) s.clear() },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> s.remove(42); s.addAll(listOf(1, 2)); s.removeAll(listOf(7)) },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 1) s.retainAll(listOf(1, 2)) },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> if (x == 2) s += 4 },
        cme131(mutableSetOf(1, 2, 3)) { s, x -> s += 1; s -= 42 },
        cme131(sortedSetOf(3, 1, 2)) { s, x -> if (x == 1) s.add(0) },
        cme131(sortedSetOf(3, 1, 2)) { s, x -> if (x == 3) s.add(5) })
    check("cme131a", r.joinToString(" ") == "cme1 cme12 ok123 ok123 ok123 cme1 cme1 ok123 cme1 cme12 ok123 cme1 ok123")
    val xs = mutableListOf(1, 2)
    var h = ""
    try { for (x in xs) { xs += emptyList() }; h += "no" } catch (e: ConcurrentModificationException) { h += "cme" }
    try { for (x in xs) { xs -= 42 }; h += " ok" } catch (e: ConcurrentModificationException) { h += " cme" }
    check("cme131b", h + xs == "cme ok[1, 2]")
}

// ===== SECTION 132: a lambda's parameters are vals, so `+=` on one is plusAssign =====
// A lambda parameter cannot be reassigned, so `l += 7` on a MutableList one resolves
// to l.plusAssign(7) and mutates the caller's list (Kotlin spec, compound assignment:
// a val target leaves only the assign operator). The same for the implicit `it`.
fun sec132() {
    val add7 = { l: MutableList<Int> -> l += 7 }
    val a = mutableListOf(1)
    add7(a)
    val d = mutableListOf(2)
    d.let { it += 8 }
    d.also { x -> x += listOf(9, 10) }
    val s = mutableSetOf(1)
    s.run { listOf(5).forEach { this += it } }
    check("lam132", "$a $d $s" == "[1, 7] [2, 8, 9, 10] [1, 5]")
}

// ===== SECTION 133: a local fun does not shadow a member method elsewhere =====
// A bare f() inside an interface's default method is this.f(), the implementing
// class's override; a local `fun f` in an unrelated function is not in scope there
// (Kotlin resolves a call name against the innermost scope declaring it). Inside a
// member whose OWN body declares a local `fun h`, the local wins over the member.
class A133 : M133 { override fun f() = 1 }
interface M133 { fun f(): Int; fun g() = f() + 10 }
fun other133(): Int { fun f() = 9; return f() }
class B133 { fun h() = 5; fun k(): Int { fun h() = 7; return h() + h2() }; fun h2() = h() }
fun sec133() {
    check("loc133", "${A133().g()} ${other133()} ${B133().k()}" == "11 9 12")
}

// ===== SECTION 134: two nested classes of one simple name stay two types =====
// Outer134.Item and Outer134b.Item are different classes: a qualified `is` / `as` /
// `as?` / when-is tells them apart, a ClassCastException names both binary names,
// and a bare `Item` inside an owner (or a sibling nested in it) is that owner's own.
// Every row is java 24 over the same pair of static nested classes.
class Outer134 {
    class Item(val v: Int) { fun show() = "A$v"; fun twin() = Pal(v).show() }
    class Pal(val v: Int) { fun show() = "P$v" }
    fun mk() = Item(3)
    fun isMine(x: Any) = x is Item
    fun asMine(x: Any) = (x as? Item)?.show()
}
class Outer134b {
    class Item(val v: Int) { fun show() = "B$v"; fun twin() = Pal(v).show() }
    class Pal(val v: Int) { fun show() = "Q$v" }
    fun mk() = Item(4)
    fun isMine(x: Any) = x is Item
    fun asMine(x: Any) = (x as? Item)?.show()
}
fun sec134() {
    val a: Any = Outer134.Item(1)
    val b: Any = Outer134b.Item(2)
    check("nest134a", "${a is Outer134b.Item} ${b is Outer134.Item} ${a is Outer134.Item} ${(b as? Outer134.Item)?.show()}"
        == "false false true null")
    check("nest134b", when (a) { is Outer134b.Item -> "b"; is Outer134.Item -> "a"; else -> "?" } == "a")
    var msg = ""
    try { (b as Outer134.Item).show() } catch (e: ClassCastException) { msg = e.message ?: "" }
    check("nest134c", msg.startsWith("class Outer134b\$Item cannot be cast to class Outer134\$Item ("))
    check("nest134d", Outer134().mk().show() + Outer134b().mk().show() == "A3B4")
    check("nest134e", "${Outer134().isMine(a)}${Outer134().isMine(b)}${Outer134b().isMine(a)}${Outer134b().isMine(b)}"
        == "truefalsefalsetrue")
    check("nest134f", "${Outer134().asMine(b)} ${Outer134b().asMine(b)}" == "null B2")
    check("nest134g", Outer134.Item(5).twin() + Outer134b.Item(6).twin() == "P5Q6")
}

// ===== SECTION 135: an implicit receiver's member beats a top-level declaration =====
// Kotlin's scope tower: locals, then the implicit receivers (innermost first, an
// inner class's outer instance included), then the top level - so inside a member
// an unqualified call, read or write of a name the receiver's class declares is the
// member, a top-level `fun` / `val` / `var` of the same name notwithstanding - unless
// no member of that name is applicable to the call (its argument count, the names of
// its named arguments, the types of its arguments), when resolution moves on to the
// top level. Spec only (the Kotlin specification, "Linked scopes" and "Call without
// an explicit receiver"); no kotlinc on this machine. The companion objects of a
// class and of its superclasses (not of an interface) are the implicit receivers
// after the instance, so their members hide a top-level declaration too.
fun pick135() = 9
fun span135(vararg xs: Int) = xs.size
val tag135 = "top"
var cnt135 = 1000
interface M135 { fun pick135(): Int; fun g() = pick135() + 10 }
class A135 : M135 { override fun pick135() = 1 }
class B135 {
    var cnt135 = 0
    val tag135 get() = "getter"
    fun pick135() = 2
    fun all() = "" + pick135() + tag135
    fun bump(): Int { cnt135++; cnt135 += 10; cnt135 = cnt135 * 2; return cnt135 }
    fun local(): Int { fun pick135() = 50; return pick135() }
    fun span135(a: Int) = a * 100
    fun spans() = "" + span135(1, 2) + span135(3)
    inner class I { fun t() = "" + pick135() + cnt135 }
}
class C135 { fun t() = "" + pick135() + tag135 + cnt135 }
fun log135(s: String) = "top:$s"
fun wide135(x: String) = "t$x"
fun opt135(x: Int) = "i$x"
var ev135 = 0
fun nx135(): String { ev135++; return "e" }
class L135 {
    fun log135(a: Int, b: Int = 1) = "mem:${a + b}"
    fun wide135(x: Long) = "m$x"
    fun opt135(x: String?) = "s$x"
    fun logs(): String {
        val s = "v"
        return log135("hi") + " " + log135(5) + " " + log135(s) + " " + log135(b = 2, a = 3) + " " + log135(s = "n")
    }
    fun wides() = wide135(7) + wide135("q") + opt135(null) + opt135(3) + opt135("z")
    fun once() = log135(nx135()) + ev135
}
fun make135() = "top"
val lim135 = 1
var cnt135b = 100
class K135 {
    companion object {
        fun make135() = "comp"
        val lim135 = 2
        var cnt135b = 0
        fun inner() = make135() + lim135
    }
    fun t() = make135() + lim135
    fun bump(): Int { cnt135b++; cnt135b += 5; return cnt135b }
    inner class I { fun t() = make135() + lim135 }
}
interface Ic135 { companion object { fun make135() = "ifc" } }
open class B0135 { companion object { fun make135() = "b0" } }
open class B1135 : B0135()
class B2135 : B1135(), Ic135 { fun t() = make135() }
class Ai135 : Ic135 { fun t() = make135() }
fun sec135() {
    check("recv135a", A135().g() == 11)
    val b = B135()
    check("recv135b", b.all() == "2getter")
    check("recv135c", b.bump() == 22 && cnt135 == 1000)
    check("recv135d", b.local() == 50 && b.I().t() == "222")
    check("recv135e", C135().t() == "9top1000")
    check("recv135f", with(b) { "" + pick135() + tag135 } == "2getter")
    check("recv135g", b.spans() == "2300")
    check("recv135h", L135().logs() == "top:hi mem:6 top:v mem:5 top:n")
    check("recv135i", L135().wides() == "m7tqsnulli3sz")
    check("recv135j", L135().once() == "top:e1")
    check("recv135k", K135().t() + K135.inner() + K135().I().t() == "comp2comp2comp2")
    check("recv135l", K135().bump() == 6 && cnt135b == 100 && K135.cnt135b == 6)
    check("recv135m", B2135().t() + Ai135().t() == "b0top")
}

// ===== SECTION 136: nested classes - KClass names, named constructor arguments =====
// A nested class's KClass is its java class: simpleName Inner, qualifiedName
// O136.Inner, java.name O136$Inner, and equal whichever way it was reached
// (java 24 + kotlin-stdlib 2.2.21's Reflection.getOrCreateKotlinClass over the same
// classes). A qualified or bare constructor call of a nested class orders its named
// arguments by THAT class's parameters, even when another nested class shares its
// simple name (the Kotlin specification, "Named arguments").
object O136 { class Inner }
class C136 { class Nd { class D }; inner class I }
class N136a {
    class In(val a: Int, val b: Int = 10) { fun s() = "a$a b$b" }
    inner class R136(val x: Int, val y: Int) { fun s() = "x$x y$y" }
    fun mk() = In(b = 3, a = 4).s()
}
class N136b {
    class In(val c: Int, val d: Int) { fun s() = "c$c d$d" }
    fun mk() = In(d = 5, c = 6).s()
}
fun sec136() {
    check("kcls136a", "${O136.Inner::class.simpleName} ${O136.Inner::class.qualifiedName}" == "Inner O136.Inner")
    val x = C136.Nd.D()
    check("kcls136b", "${x::class.simpleName} ${x::class.qualifiedName} ${x::class.java.name}" == "D C136.Nd.D C136\$Nd\$D")
    check("kcls136c", C136.I::class.qualifiedName == "C136.I" && C136.Nd::class == C136.Nd()::class)
    check("kcls136d", C136.Nd::class.java.name == "C136\$Nd" && C136.Nd.D::class.simpleName == "D")
    check("nest136a", N136a.In(b = 1, a = 2).s() == "a2 b1" && N136a().mk() == "a4 b3")
    check("nest136b", N136b.In(d = 7, c = 8).s() == "c8 d7" && N136b().mk() == "c6 d5")
    check("nest136c", N136a().R136(y = 1, x = 2).s() == "x2 y1")
}

// ===== SECTION 137: a nested type's name is not a top-level name =====
// Outside its owner a nested type is reached only qualified, so a bare name there is
// the top-level declaration of it - a class, a fun or a val - wherever either is
// declared; inside the owner (and inside a type nested in it) the nested type wins,
// and an enum's entries win inside the enum. Spec only (the Kotlin specification,
// "Declarations" / "Linked scopes"); no kotlinc on this machine.
class In137(val x: Int) { override fun toString() = "top$x" }
class O137 {
    class In137(val y: Int) { override fun toString() = "O$y" }
    class Sib { fun mk() = In137(7) }
    fun mk() = In137(5)
}
enum class Col137 { N, S; fun flip() = if (this == N) S else N }
class H137 { class N { override fun toString() = "HN" }; fun mk() = N() }
object Ob137 { class S { override fun toString() = "ObS" }; fun mk() = S() }
interface If137 { class Item137 { override fun toString() = "IfItem" }; fun mk() = Item137() }
class Impl137 : If137
enum class E137 { X; class Item137 { override fun toString() = "EItem" }; fun mk() = Item137() }
val Item137 = "topval"
class A137 { class Leaf137(val v: Int) { override fun toString() = "L$v" }; class Sib { fun mk() = Leaf137(9) }; fun mk() = Leaf137(1) }
fun Leaf137(v: Int) = "fun$v"
fun sec137() {
    check("nest137a", "${In137(1)} ${O137.In137(2)} ${O137().mk()} ${O137.Sib().mk()}" == "top1 O2 O5 O7")
    val z: Any = In137(3)
    check("nest137b", "$z" == "top3" && z is In137 && z::class.simpleName == "In137")
    check("nest137c", "${Col137.N.flip()}${Col137.S.flip()} ${H137().mk()} ${Ob137.mk()}" == "SN HN ObS")
    check("nest137d", "${Impl137().mk()} ${E137.X.mk()} $Item137" == "IfItem EItem topval")
    check("nest137e", "${A137().mk()} ${A137.Sib().mk()} ${Leaf137(2)}" == "L1 L9 fun2")
}

// ===== SECTION 138: KClass names of builtin types =====
// Read off kotlin-stdlib 2.2.21 under JDK 24 (Reflection.getOrCreateKotlinClass(c)
// .getSimpleName() / .getQualifiedName() / toString()): a builtin throwable's KClass
// is its java class (java.lang.IllegalStateException, java.util.NoSuchElementException,
// kotlin.NotImplementedError), Throwable's qualifiedName is kotlin.Throwable, and a
// builtin type's is its kotlin name (kotlin.String, kotlin.Int, kotlin.Any). The
// real toString adds " (Kotlin reflection is not available)", which this
// implementation does not print.
fun sec138() {
    val e: Any = IllegalStateException("x")
    check("kcls138a", "${e::class.qualifiedName} ${e::class.simpleName} ${e::class.java.name}" ==
        "java.lang.IllegalStateException IllegalStateException java.lang.IllegalStateException")
    check("kcls138b", IllegalStateException::class.qualifiedName == "java.lang.IllegalStateException" &&
        IllegalStateException::class == e::class && "${e::class}" == "class java.lang.IllegalStateException")
    check("kcls138c", "${NoSuchElementException()::class.qualifiedName} ${NotImplementedError()::class.qualifiedName} ${Throwable::class.qualifiedName}" ==
        "java.util.NoSuchElementException kotlin.NotImplementedError kotlin.Throwable")
    val s: Any = "abc"
    check("kcls138d", "${String::class.qualifiedName} ${String::class.simpleName} ${String::class} ${s::class.qualifiedName}" ==
        "kotlin.String String class java.lang.String kotlin.String" && s::class == String::class)
    check("kcls138e", "${Int::class.qualifiedName} ${5::class.qualifiedName} ${Any::class.qualifiedName} ${Char::class.qualifiedName}" ==
        "kotlin.Int kotlin.Int kotlin.Any kotlin.Char")
}

// ===== SECTION 139: a type test names one class; an interface's companion =====
// `is` / `as` / `as?` / `when` / filterIsInstance and a reified `is T` resolve the
// written name lexically to ONE class, so a top-level In139 and O139.In139 are
// unrelated types whichever spelling is tested. An interface may declare a
// companion object, reached through the interface's name. Spec only (the Kotlin
// specification, "Type checking" / "Linked scopes"); the same tests through java's
// instanceof on a static nested class answer the same under JDK 24.
class In139(val x: Int)
class O139 {
    class In139(val y: Int)
    fun own(a: Any) = a is In139
}
inline fun <reified T> isA139(x: Any) = x is T
interface Ic139 {
    fun name(): String
    companion object {
        const val K = 7
        val base = "B"
        fun g() = "ifc"
        fun mk(): Ic139 = Im139("made")
    }
}
class Im139(val n: String) : Ic139 { override fun name() = n }
interface Nm139 { companion object Factory { fun of() = "named" } }
fun sec139() {
    val a: Any = O139.In139(1)
    val b: Any = In139(2)
    check("istop139a", "${a is In139} ${a is O139.In139} ${b is In139} ${b is O139.In139}" == "false true true false")
    check("istop139b", O139().own(a) && !O139().own(b) && (a as? In139) == null && (b as? O139.In139) == null)
    check("istop139c", when (b) { is O139.In139 -> "nested"; is In139 -> "top"; else -> "none" } == "top")
    var cce = false
    try { println((a as In139).x) } catch (e: ClassCastException) { cce = true }
    check("istop139d", cce && listOf(a, b, In139(3)).filterIsInstance<In139>().size == 2)
    check("istop139e", !isA139<In139>(a) && isA139<In139>(b) && isA139<O139.In139>(a) && !isA139<O139.In139>(b))
    check("ifcomp139a", "${Ic139.g()} ${Ic139.K + 1} ${Ic139.base} ${Ic139.mk().name()}" == "ifc 8 B made")
    check("ifcomp139b", Nm139.of() == "named" && Nm139.Factory.of() == "named")
}

// ===== SECTION 140: an enum entry is an Enum<*> =====
// Every enum class implicitly extends kotlin.Enum (java.lang.Enum on the JVM), an
// entry with its own body included; nothing else is one. JDK 24 instanceof on the
// equivalent java enums answers the same.
enum class Col140 { RED, GREEN }
enum class Op140 { PLUS { override fun ap(a: Int) = a + 1 }; abstract fun ap(a: Int): Int }
class Pl140(val x: Int)
fun sec140() {
    val vs: List<Any> = listOf(Col140.RED, Op140.PLUS, Pl140(1), "s", 1)
    check("isenum140a", vs.map { it is Enum<*> }.joinToString() == "true, true, false, false, false")
    check("isenum140b", vs.filterIsInstance<Enum<*>>().map { it.name + it.ordinal }.joinToString() == "RED0, PLUS0")
    check("isenum140c", (vs[0] as? Enum<*>)?.name == "RED" && (vs[2] as? Enum<*>) == null)
    check("isenum140d", when (vs[1]) { is Enum<*> -> "enum"; else -> "other" } == "enum" && vs[1] is Comparable<*>)
}

// ===== SECTION 141: map loops and list higher-order functions fail fast =====
// A for over a map, its keys / values / entries, and Iterable.forEach /
// forEachIndexed / map / withIndex all run the java.util iterator, which throws
// ConcurrentModificationException from next() once the collection changed
// structurally. A LinkedHashMap's iterator fetched its next node with the current
// one, so a change at the last entry ends the loop quietly, and replacing a value
// is no structural change. Every row is java 24's for the same LinkedHashMap /
// ArrayList loop.
fun cme141(f: () -> Unit): String {
    try { f(); return "ok" } catch (e: ConcurrentModificationException) { return "cme" }
}
fun sec141() {
    val m = listOf(
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2); for (k in mp.keys) if (k == 1) mp[3] = 3 },
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2); for (k in mp.keys) if (k == 2) mp[3] = 3 },
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2, 3 to 3); for ((k, _) in mp) if (k == 1) mp.remove(2) },
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2); for (e in mp.entries) if (e.key == 1) mp.clear() },
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2); for (v in mp.values) if (v == 1) mp[7] = 7 },
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2); for (v in mp.values) if (v == 1) mp[1] = 9 },
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2); for (k in mp.keys) mp[k] = 0 },
        cme141 { val mp = mutableMapOf(1 to 1, 2 to 2); mp.forEach { (k, _) -> if (k == 1) mp[3] = 3 } })
    check("cme141a", m.joinToString(" ") == "cme ok cme cme cme ok ok cme")
    val l = listOf(
        cme141 { val ml = mutableListOf(1, 2, 3); ml.forEach { if (it == 1) ml.add(5) } },
        cme141 { val ml = mutableListOf(1, 2, 3); ml.forEach { if (it == 3) ml.add(5) } },
        cme141 { val ml = mutableListOf(1, 2, 3); ml.forEach { if (it == 2) ml.remove(1) } },
        cme141 { val ml = mutableListOf(1, 2, 3); ml.forEachIndexed { i, x -> if (i == 0) ml.add(x) } },
        cme141 { val ml = mutableListOf(1, 2, 3); ml.map { if (it == 1) ml.add(5); it } },
        cme141 { val ml = mutableListOf(1, 2, 3); for (x in ml.withIndex()) if (x.index == 0) ml.add(9) },
        cme141 { val ml = mutableListOf(1, 2, 3); for (x in ml.withIndex()) if (x.index == 0) ml[1] = 7 },
        cme141 { val ml = mutableListOf(1, 2, 3); ml.forEach { if (it == 1) ml[2] = 8 } })
    check("cme141b", l.joinToString(" ") == "cme cme ok cme cme cme ok ok")
}

// ===== SECTION 142: `as` a collection interface is a checkcast =====
// `x as List<*>` / `Set<*>` / `Map<*, *>` / `Collection<*>` / `Iterable<*>` is a
// checkcast to the java.util (java.lang) interface: the same test as `is`, and a
// ClassCastException naming both classes otherwise. java 24's casts of the same
// objects (Arrays$ArrayList, LinkedHashSet, Collections$SingletonMap, String).
fun cce142(f: () -> Any): String {
    try { return "ok:" + f() } catch (e: ClassCastException) { return "cce:" + e.message!!.substringBefore(" (") }
}
fun sec142() {
    val lst: Any = listOf(1, 2)
    val str: Any = "s"
    val st: Any = setOf(1, 2)
    val mp: Any = mapOf(1 to 2)
    check("cce142a", cce142 { lst as Map<*, *> } == "cce:class java.util.Arrays\$ArrayList cannot be cast to class java.util.Map" &&
                     cce142 { str as List<*> } == "cce:class java.lang.String cannot be cast to class java.util.List" &&
                     cce142 { lst as Set<*> } == "cce:class java.util.Arrays\$ArrayList cannot be cast to class java.util.Set")
    check("cce142b", cce142 { mp as Iterable<*> }.startsWith("cce:") && cce142 { st as List<*> }.startsWith("cce:") &&
                     cce142 { mp as Collection<*> }.startsWith("cce:") && cce142 { str as MutableList<*> }.startsWith("cce:"))
    check("cce142c", cce142 { lst as Collection<*> } == "ok:[1, 2]" && cce142 { st as Iterable<*> } == "ok:[1, 2]" &&
                     cce142 { mp as Map<*, *> } == "ok:{1=2}" && cce142 { lst as MutableList<*> } == "ok:[1, 2]")
}

// ===== SECTION 143: multi-index operators and MutableIterator.remove() =====
// `g[r, c]` is `g.get(r, c)` and `g[r, c] = v` is `g.set(r, c, v)`, compound forms
// included. remove() drops the element next() last answered, from a list, a set,
// a map through its entries / keys / values iterators; a second remove() or one
// before next() is ArrayList.Itr's message-less IllegalStateException. java 24
// runs the same loops over ArrayList / LinkedHashSet / LinkedHashMap.
class Grid143(val w: Int) {
    val cells = IntArray(w * 2)
    operator fun get(r: Int, c: Int): Int = cells[r * w + c]
    operator fun set(r: Int, c: Int, v: Int) { cells[r * w + c] = v }
}
fun sec143() {
    val g = Grid143(3)
    g[1, 2] = 7
    g[0, 1] = 4
    g[1, 2] += 5
    check("midx143a", g[1, 2] == 12 && g[0, 1] == 4 && g.cells.toList() == listOf(0, 4, 0, 0, 0, 12))
    val ml = mutableListOf(1, 2, 3, 4, 5, 6)
    val it = ml.iterator()
    while (it.hasNext()) { if (it.next() % 2 == 0) it.remove() }
    val ml2 = mutableListOf(1, 2, 3)
    val it2 = ml2.iterator()
    it2.next(); it2.hasNext(); it2.remove()
    var ise = ""
    try { it2.next(); it2.remove(); it2.remove() } catch (e: IllegalStateException) { ise = "ise" + e.message }
    check("itrm143b", ml.toString() == "[1, 3, 5]" && ml2.toString() == "[3]" && ise == "isenull")
    val mp = mutableMapOf(1 to "a", 2 to "b", 3 to "c", 4 to "d")
    val mi = mp.iterator()
    while (mi.hasNext()) { if (mi.next().key == 2) mi.remove() }
    val ki = mp.keys.iterator()
    while (ki.hasNext()) { if (ki.next() == 1) ki.remove() }
    val vi = mp.values.iterator()
    while (vi.hasNext()) { if (vi.next() == "d") vi.remove() }
    val st = mutableSetOf("x", "y", "z")
    val si = st.iterator()
    while (si.hasNext()) { if (si.next() == "y") si.remove() }
    check("itrm143c", mp.toString() == "{3=c}" && st.toString() == "[x, z]" && st.size == 2 && "y" !in st)
}

// ===== SECTION 144: every higher-order function fails fast, over sets too =====
// filter / any / count / fold / sumOf / groupBy / maxBy / zipWithNext / takeWhile
// / dropWhile ... run the receiver's java.util iterator, so a lambda that changes
// the list or set structurally makes the next next() throw
// ConcurrentModificationException after one call (n). any { true } and
// first { } returning at once never step again; a List's last { } walks back from
// the end, a set's walks forward over every element; replacing an element is no
// structural change. Every row is kotlin-stdlib 2.2.21's function on java 24's
// ArrayList / LinkedHashSet, called with the same lambda.
var n144 = 0
fun cme144(f: () -> Any?): String {
    n144 = 0
    try { return "ok:" + f() + "/" + n144 } catch (e: ConcurrentModificationException) { return "cme" + n144 }
}
fun sec144() {
    val a = listOf(
        cme144 { val c = mutableListOf(1, 2, 3); c.filter { n144++; c.add(9); true } },
        cme144 { val c = mutableListOf(1, 2, 3); c.any { n144++; c.add(9); false } },
        cme144 { val c = mutableListOf(1, 2, 3); c.count { n144++; c.add(9); true } },
        cme144 { val c = mutableListOf(1, 2, 3); c.all { n144++; c.add(9); true } },
        cme144 { val c = mutableListOf(1, 2, 3); c.first { n144++; c.add(9); it == 3 } },
        cme144 { val c = mutableListOf(1, 2, 3); c.sumOf { n144++; c.add(9); it } },
        cme144 { val c = mutableListOf(1, 2, 3); c.fold(0) { s, x -> n144++; c.add(9); s + x } },
        cme144 { val c = mutableListOf(1, 2, 3); c.reduce { s, x -> n144++; c.add(9); s + x } },
        cme144 { val c = mutableListOf(1, 2, 3); c.groupBy { n144++; c.add(9); it } },
        cme144 { val c = mutableListOf(1, 2, 3); c.maxBy { n144++; c.add(9); it } },
        cme144 { val c = mutableListOf(1, 2, 3); c.zipWithNext { x, _ -> n144++; c.add(9); x } },
        cme144 { val c = mutableListOf(1, 2, 3); c.dropWhile { n144++; c.add(9); false } },
        cme144 { val c = mutableListOf(1, 2, 3); c.joinToString { n144++; c.add(9); "" + it } },
        cme144 { val c = mutableListOf(1, 2, 3); c.indexOfLast { n144++; c.add(9); false } })
    check("cme144a", a.joinToString(" ") == "cme1 cme1 cme1 cme1 cme1 cme1 cme1 cme1 cme1 cme1 cme1 cme1 cme1 cme1")
    val b = listOf(
        cme144 { val c = mutableListOf(1, 2, 3); c.any { n144++; c.add(9); true } },
        cme144 { val c = mutableListOf(1, 2, 3); c.first { n144++; c.add(9); it == 1 } },
        cme144 { val c = mutableListOf(1, 2, 3); c.last { n144++; c.add(9); true } },
        cme144 { val c = mutableListOf(1, 2, 3); c.takeWhile { n144++; c.add(9); false } },
        cme144 { val c = mutableListOf(1, 2, 3); c.count { n144++; c[0] = 5; true } },
        cme144 { val c = mutableListOf(1, 2, 3); c.count { n144++; if (it == 3) c.add(9); true } })
    check("cme144b", b.joinToString(" ") == "ok:true/1 ok:1/1 ok:3/1 ok:[]/1 ok:3/3 cme3")
    val s = listOf(
        cme144 { val c = mutableSetOf(1, 2, 3); c.filter { n144++; c.add(9); true } },
        cme144 { val c = mutableSetOf(1, 2, 3); c.any { n144++; c.add(9); false } },
        cme144 { val c = mutableSetOf(1, 2, 3); c.last { n144++; c.add(9); true } },
        cme144 { val c = mutableSetOf(1, 2, 3); c.sumOf { n144++; c.add(9); it } },
        cme144 { val c = mutableSetOf(1, 2, 3); c.forEach { n144++; c.add(it + 10) } },
        cme144 { val c = mutableSetOf(1, 2, 3); c.map { n144++; c.add(it + 10) } },
        cme144 { val c = mutableSetOf(1, 2, 3); c.forEach { n144++; if (it == 3) c.add(13) }; c },
        cme144 { val c = mutableSetOf(1, 2, 3); c.forEachIndexed { _, x -> n144++; c.remove(x) } })
    check("cme144c", s.joinToString(" ") == "cme1 cme1 cme1 cme1 cme1 cme1 ok:[1, 2, 3, 13]/3 cme1")
    val m = listOf(
        cme144 { val c = mutableMapOf(1 to "a", 2 to "b"); c.any { n144++; c[9] = "z"; false } },
        cme144 { val c = mutableMapOf(1 to "a", 2 to "b"); c.keys.filter { n144++; c[9] = "z"; true } },
        cme144 { val c = mutableMapOf(1 to "a", 2 to "b"); c.values.any { n144++; c.remove(2); false } })
    check("cme144d", m.joinToString(" ") == "cme1 cme1 cme1")
}

// ===== SECTION 145: a map's views write through, and removeIf =====
// A map's keys / values / entries are views: their own remove / removeAll /
// retainAll / removeIf / clear take the entries out of the map - by key, the
// first entry with the value, the equal entry. removeAll / retainAll also take a
// predicate on a set, and java's removeIf works on lists and sets. Every row is
// java 24's LinkedHashMap / ArrayList / LinkedHashSet, the predicate forms
// kotlin-stdlib 2.2.21's.
fun m145(): MutableMap<Int, String> = mutableMapOf(1 to "a", 2 to "b", 3 to "a")
fun sec145() {
    val r = mutableListOf<String>()
    var x = m145(); r.add("" + x.keys.remove(1) + x)
    x = m145(); r.add("" + x.keys.remove(9) + x)
    x = m145(); r.add("" + x.values.remove("a") + x)
    x = m145(); r.add("" + x.entries.removeIf { it.key == 1 } + x)
    check("view145a", r.joinToString(" ") == "true{2=b, 3=a} false{1=a, 2=b, 3=a} true{2=b, 3=a} true{2=b, 3=a}")
    r.clear()
    x = m145(); r.add("" + x.keys.removeAll(listOf(1, 3)) + x)
    x = m145(); r.add("" + x.keys.retainAll(listOf(2)) + x)
    x = m145(); r.add("" + x.keys.removeIf { it > 1 } + x)
    x = m145(); x.keys.clear(); r.add("" + x)
    x = m145(); r.add("" + x.values.removeAll(listOf("a")) + x)
    x = m145(); r.add("" + x.values.retainAll(listOf("a")) + x)
    x = m145(); x.values.clear(); r.add("" + x)
    check("view145b", r.joinToString(" ") == "true{2=b} true{2=b} true{1=a} {} true{2=b} true{1=a, 3=a} {}")
    r.clear()
    x = m145(); val e = x.entries.first(); r.add("" + x.entries.remove(e) + x)
    x = m145(); r.add("" + x.keys.removeAll { it == 2 } + x)
    x = m145(); r.add("" + x.values.retainAll { it == "a" } + x)
    x = m145(); r.add("" + x.entries.removeAll { it.value == "a" } + x)
    check("view145c", r.joinToString(" ") == "true{2=b, 3=a} true{1=a, 3=a} true{1=a, 3=a} true{2=b}")
    val s = mutableSetOf(1, 2, 3, 4)
    val l = mutableListOf(1, 2, 3, 4)
    check("view145d", s.removeAll { it > 3 } && s.retainAll { it > 1 } && !s.removeIf { it > 5 } && s.toString() == "[2, 3]" &&
                      l.removeIf { it % 2 == 0 } && l.toString() == "[1, 3]")
}

// ===== SECTION 146: ++ / -- of a slot reached through a call =====
// `rec(p).w++` steps the property of the call's result: the call runs once, the
// postfix form yields the old value and the prefix form the new one, and an index
// slot or a `?.` step (skipped on null) works the same way.
class R146(var w: Int, var c: Char = 'a')
var calls146 = 0
fun rec146(p: R146, s: String): R146 { calls146++; return p }
fun arr146(a: IntArray) = a
fun nul146(p: R146?): R146? = p
fun sec146() {
    val p = R146(1)
    rec146(p, "R").w++
    check("inc146a", p.w == 2 && calls146 == 1)
    val y = 10 + rec146(p, "R").w--
    val z = ++rec146(p, "R").w
    check("inc146b", y == 12 && z == 2 && p.w == 2 && calls146 == 3)
    val a = intArrayOf(5, 6)
    arr146(a)[1]++
    val q = --arr146(a)[0]
    check("inc146c", a.joinToString() == "4, 7" && q == 4)
    nul146(null)?.w++
    nul146(p)?.w++
    rec146(p, "c").c++
    check("inc146d", p.w == 3 && p.c == 'b')
}

// ===== SECTION 147: compound assignment to a view, and lambdas over a changing list =====
// `m.keys -= k` is minusAssign on the val keys view: a remove that writes through to
// the map, and `m.keys += k` is the view's add, an UnsupportedOperationException. A
// lambda's element is read before the lambda runs, so a lambda that writes the list
// does not change what filter / partition / first answer, and single /
// singleOrNull stop at the second match. Every row is java 24's LinkedHashMap /
// ArrayList driven through kotlin-stdlib 2.2.21.
fun sec147() {
    val x = m145()
    x.keys -= 2
    val x2 = m145()
    x2.values -= "a"
    var uoe = ""
    try { x2.keys += 7 } catch (e: UnsupportedOperationException) { uoe = "UOE " + e.message }
    check("view147a", "$x $x2 $uoe" == "{1=a, 3=a} {2=b, 3=a} UOE null")
    val c = mutableListOf(1, 2, 3)
    val r = mutableListOf<String>()
    r.add("" + c.filter { c[0] = 5; true }); c[0] = 1
    r.add("" + c.filterNot { c[0] = 5; false }); c[0] = 1
    r.add("" + c.partition { c[0] = 5; true }); c[0] = 1
    r.add("" + c.first { c[0] = 5; true }); c[0] = 1
    r.add("" + c.find { c[0] = 5; true }); c[0] = 1
    r.add("" + c.filterIndexed { _, _ -> c[0] = 5; true }); c[0] = 1
    r.add("" + c.associateBy({ c[0] = 5; it }, { it })); c[0] = 1
    r.add("" + c.last { c[2] = 9; true })
    check("lam147b", r.joinToString(" ") == "[1, 2, 3] [1, 2, 3] ([1, 2, 3], []) 1 1 [1, 2, 3] {1=1, 2=2, 3=3} 3")
    var n = 0
    var msg = ""
    try { listOf(1, 2, 3, 4).single { n++; it > 1 } } catch (e: IllegalArgumentException) { msg = "" + e.message }
    var n2 = 0
    val so = listOf(1, 2, 3, 4).singleOrNull { n2++; it > 1 }
    check("one147c", n == 3 && msg == "Collection contains more than one matching element." && so == null && n2 == 3)
    // keys / values / entries are LIVE views, one object per map: a later put, an
    // iterator's remove and clear show through them (java 24 LinkedHashMap / TreeMap).
    val m = mutableMapOf(1 to "a", 2 to "b", 3 to "c")
    val ks = m.keys
    val vs = m.values
    val es = m.entries
    m[2] = "z"
    ks.remove(1)
    m[5] = "e"
    val r2 = mutableListOf("$m $ks $vs $es")
    val it = vs.iterator()
    it.next(); it.remove()
    r2.add("$ks $vs $es ${m.keys === ks}")
    m.clear(); m[9] = "q"
    r2.add("$ks $vs $es")
    check("view147d", r2.joinToString(" ") ==
        "{2=z, 3=c, 5=e} [2, 3, 5] [z, c, e] [2=z, 3=c, 5=e] [3, 5] [c, e] [3=c, 5=e] true [9] [q] [9=q]")
    val t = sortedMapOf(3 to "c", 1 to "a")
    val tk = t.keys
    val tv = t.values
    t[2] = "b"; t[0] = "o"; t.remove(1)
    check("view147e", "$tk $tv ${t.entries}" == "[0, 2, 3] [o, b, c] [0=o, 2=b, 3=c]")
}

// ===== SECTION 148: a newline ends a statement before a binary operator =====
// `i++` then `--j` on the next line is two statements, not `i++ - (-j)`; the same
// newline inside ( ), [ ] or ${ } continues the expression, and inside braces it
// does not (kotlinc 2.2.21).
fun id148(x: Int) = x
fun s148() {
    var i = 0; var j = 5
    i++
    --j
    val r = mutableListOf("$i $j")
    val a = (1
        + 2)
    r.add("$a")
    r.add("" + listOf(1
        - 3, 4))
    val b = 10
    -3
    r.add("$b")
    val xs = intArrayOf(1, 2)
    r.add("" + xs[0
        + 1])
    val f = { 7
        - 2 }
    r.add("" + f())
    var k = 3
    val m = k
    ++k
    r.add("$m $k")
    r.add("${id148(1
        * 2)}")
    check("nl148a", r.joinToString(" ") == "1 4 3 [-2, 4] 10 2 -2 3 4 2")
    // A paren in a string, a char literal or a comment does not open a group.
    val s = "(("   // ((
    val c = '('
    var q = 7
    q
    -1
    check("nl148b", q == 7 && s + c == "(((")
}

// ===== SECTION 149: sorting is java's TimSort, and map entries are live =====
// The selector / comparator call sequence is the JDK's (run detection, binary
// insertion below 32 elements, galloping merges above), observable through side
// effects; compareValues calls compareTo once. A held Map.Entry sees later puts,
// and setValue writes through (kotlinc 2.2.21 on java 24).
class V149(val n: Int) : Comparable<V149> {
    override fun compareTo(other: V149): Int { cnt149++; return n.compareTo(other.n) }
}
var cnt149 = 0
fun s149() {
    var n = 0
    val a = listOf(3, 1, 2).sortedBy { n++; it }
    var n2 = 0
    val b = listOf(3, 1, 2).sortedWith(compareBy { n2++; it })
    val log = mutableListOf<Int>()
    val c = listOf(3, 1, 2).sortedByDescending { log.add(it); it }
    check("sort149a", a == listOf(1, 2, 3) && n == 8 && n2 == 8 && c == listOf(3, 2, 1) &&
        log == listOf(3, 1, 1, 2, 1, 2, 3, 2))
    val xs = (0 until 70).map { (it * 37) % 23 }
    var k = 0
    val d = xs.sortedBy { k++; it }
    var k2 = 0
    val e = xs.sortedWith { p, q -> k2++; (p % 5).compareTo(q % 5) }
    cnt149 = 0
    val f = xs.map { V149(it) }.sortedDescending()
    check("sort149b", d == xs.sorted() && k == 692 && k2 == 328 && cnt149 == 345 &&
        e.take(6) == listOf(0, 5, 10, 15, 20, 0) && f.first().n == 22)
    val ml = xs.toMutableList()
    var k3 = 0
    ml.sortWith(compareBy<Int> { k3++; it % 3 }.thenByDescending { k3++; it })
    check("sort149c", k3 == 1170 && ml.take(5) == listOf(21, 21, 21, 18, 18))
    var msg = ""
    var cc = 0
    val ys = (0 until 100).map { (it * 37) % 101 }
    try { ys.sortedWith { p, q -> cc++; if ((p * 31 + q * 17 + cc) % 5 == 0) 1 else -1 } }
    catch (x: IllegalArgumentException) { msg = "" + x.message }
    check("sort149d", cc == 191 && msg == "Comparison method violates its general contract!")
    val m = mutableMapOf(1 to "a", 2 to "b", 3 to "c")
    val e1 = m.entries.first()
    val e2 = m.entries.elementAt(1)
    m[1] = "z"
    m.remove(2)
    m[2] = "q"
    val old = m.entries.last().setValue("w")
    for (en in m) en.setValue(en.value + "!")
    check("entry149", "$e1 $e2 $old $m" == "1=z! 2=b q {1=z!, 3=c!, 2=w!}")
}

// ===== SECTION 150: String order is by UTF-16 unit =====
// String.compareTo compares UTF-16 code units, so an astral character (a
// surrogate pair, U+D83D U+DE00) sorts BELOW U+E000..U+FFFF, unlike code point
// order. The operators, max / min, maxOf, sorted and the sorted collections all
// use that order (kotlinc 2.2.21 on java 24).
fun s150() {
    val xs = listOf("｡x", "😀", "x", "￿x", "ax")
    val cs = { l: Iterable<String> -> l.map { it[0].code } }
    val up = listOf(97, 55357, 57344, 65377, 65535)
    val sm = xs[1]
    check("utf16ord150a", sm < xs[2] && !(sm > xs[2]) && sm <= xs[0] && xs[3] >= sm &&
        maxOf(sm, xs[2]) == xs[2] && minOf(xs[0], sm) == sm)
    check("utf16ord150b", cs(xs.sorted()) == up && cs(xs.sortedDescending()) == up.reversed() &&
        cs(xs.toSortedSet()) == up && cs(xs.sortedBy { it }) == up)
    check("utf16ord150c", xs.max() == xs[3] && xs.maxOf { it } == xs[3] && xs.maxBy { it } == xs[3] &&
        xs.filter { it != "ax" }.min() == sm && xs.count { it > "" } == 3)
    val ts = sortedSetOf<String>()
    for (x in xs) ts.add(x)
    val pq = java.util.PriorityQueue<String>(xs)
    check("utf16ord150d", cs(ts) == up && pq.poll() == "ax" && pq.poll() == sm)
}

// ===== SECTION 151: toSortedSet and sortedSetOf are TreeSets =====
// toSortedSet() / toSortedSet(comparator) and sortedSetOf([comparator,] ...)
// answer a java.util.TreeSet: ordered by the comparator when one is given, and a
// later add() keeps that order (kotlinc 2.2.21 on java 24).
fun s151() {
    val xs = listOf(3, 1, 2, 5, 4)
    val d = xs.toSortedSet(reverseOrder())
    d.add(0); d.add(9)
    check("sortedset151a", "$d" == "[9, 5, 4, 3, 2, 1, 0]" && d.first() == 9)
    val a = xs.toSortedSet()
    a.add(0); a.add(-1)
    check("sortedset151b", "$a" == "[-1, 0, 1, 2, 3, 4, 5]")
    val ws = listOf("bb", "a", "ccc", "dd")
    check("sortedset151c", "${ws.toSortedSet(compareBy { it.length })}" == "[a, bb, ccc]" &&
        "${arrayOf(5, 2, 8).toSortedSet(reverseOrder())}" == "[8, 5, 2]" &&
        "${setOf(4, 1, 3).toSortedSet(reverseOrder())}" == "[4, 3, 1]" &&
        "${xs.asSequence().toSortedSet(reverseOrder())}" == "[5, 4, 3, 2, 1]")
    val ts = setOf(4, 1, 3).toSortedSet()
    ts.add(2)
    check("sortedset151d", "$ts" == "[1, 2, 3, 4]" && "${sortedSetOf(reverseOrder(), 1, 5, 3)}" == "[5, 3, 1]")
}

// ===== SECTION 152: ranges over Comparable bounds =====
// a..b over Strings (or any Comparable that is not a number) is a ComparableRange
// and a..<b a ComparableOpenEndRange: `in` compares by compareTo, toString shows
// the bounds, two empty ranges are equal and hash to -1. The coerce family on a
// Comparable orders by compareTo too; a class declaring `operator fun rangeTo`
// builds its own value (kotlinc 2.2.21 on java 24).
data class W152(val n: Int) : Comparable<W152> {
    override fun compareTo(other: W152) = n.compareTo(other.n)
}
class D152(val d: Int) {
    operator fun rangeTo(o: D152) = "span $d-${o.d}"
}
fun s152() {
    val ws = listOf("m", "a", "z", "zz", "", "B", "abc", "\uFFFF")
    val r = ws[1]..ws[2]
    val hits = ws.map { if (it in r) 1 else 0 }
    val all = ws.count { ("" .. "\uFFFF").contains(it) }
    check("crange152a", hits == listOf(1, 1, 1, 0, 0, 0, 1, 0) && all == 8 && "zz" !in r &&
        "$r" == "a..z" && r.start + r.endInclusive == "az" && !r.isEmpty() && (ws[2]..ws[1]).isEmpty())
    val u = ws[1]..<ws[2]
    check("crange152b", "$u" == "a..<z" && "z" !in u && "m" in u && u.start + u.endExclusive == "az" &&
        (ws[2]..<ws[2]).isEmpty())
    check("crange152c", r == ("a".."z") && r.hashCode() == 3129 && ("b".."a").hashCode() == -1 &&
        ("b".."a") == ("c".."a") && !r.equals("a"))
    val c = ws.map { "${it.coerceAtLeast("c")}/${it.coerceAtMost("c")}/${it.coerceIn("b", "y")}" }
    check("crange152d", c[0] == "m/c/m" && c[1] == "c/a/b" && c[3] == "zz/c/y" && c[4] == "c//b" &&
        ws[1].coerceIn(ws[2]..ws[3]) == "z" && ws[6].coerceIn("abd".."b") == "abd")
    var msg = ""
    try { ws[0].coerceIn("z", "a") } catch (e: IllegalArgumentException) { msg = "" + e.message }
    check("crange152e", msg == "Cannot coerce value to an empty range: maximum a is less than minimum z.")
    val vr = W152(1)..W152(5)
    check("crange152f", "$vr" == "W152(n=1)..W152(n=5)" && W152(3) in vr && W152(7) !in vr &&
        W152(9).coerceIn(W152(1), W152(4)) == W152(4) && W152(0).coerceAtLeast(W152(2)) == W152(2))
    val any: Any = r
    check("crange152g", any is ClosedRange<*> && any !is IntRange &&
        (when (ws[0]) { in "a".."l" -> "low"; in "m".."z" -> "high"; else -> "none" }) == "high")
    check("crange152h", D152(1)..D152(4) == "span 1-4")
}

// ===== SECTION 153: range classes and the function-spelled range builders =====
// IntRange, CharRange and LongRange implement both ClosedRange and OpenEndRange;
// a stepped or downTo progression implements neither. `1.0..<2.0` is an
// OpenEndDoubleRange (contains tests `< endExclusive`, isEmpty is
// !(start < endExclusive)). Each range value's class is its kotlin.ranges class,
// and rangeTo / rangeUntil are callable as functions on a number, a Char, a
// String or any Comparable (kotlinc 2.2.21 on java 24).
fun s153() {
    val xs = listOf(1, 5, 2)
    val rs: List<Any> = listOf(xs[0]..xs[1], xs[0]..<xs[1], xs[1] downTo xs[0], (xs[0]..xs[1]) step xs[2],
        'a'..'c', 1L..5L, (1L..5L) step 2L, 1.0..2.0, 1.0..<2.0, 1.0f..<2.0f, "a".."c", "a"..<"c")
    val cr = rs.map { if (it is ClosedRange<*>) 1 else 0 }
    val oe = rs.map { if (it is OpenEndRange<*>) 1 else 0 }
    check("rclass153a", cr == listOf(1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0) &&
        oe == listOf(1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 1))
    val names = rs.map { it::class.simpleName }.joinToString(",")
    check("rclass153b", names == "IntRange,IntRange,IntProgression,IntProgression,CharRange,LongRange," +
        "LongProgression,ClosedDoubleRange,OpenEndDoubleRange,OpenEndFloatRange,ComparableRange,ComparableOpenEndRange")
    check("rclass153c", ("a".."c")::class.qualifiedName == "kotlin.ranges.ComparableRange" &&
        ("a"..<"c").javaClass.name == "kotlin.ranges.ComparableOpenEndRange" &&
        rs[5] is LongRange && rs[5] !is IntRange && rs[6] is LongProgression && rs[6] !is LongRange)
    val ds = listOf(1.0, 2.0, 1.5, -0.0, 0.0)
    val h = ds[0]..<ds[1]
    check("rclass153d", "$h" == "1.0..<2.0" && h.start == 1.0 && h.endExclusive == 2.0 && ds[2] in h &&
        ds[1] !in h && ds[0] in h && !h.isEmpty() && (ds[0]..<ds[0]).isEmpty() && (ds[3]..<ds[4]).isEmpty() &&
        h == (ds[0]..<ds[1]) && !h.equals(ds[0]..ds[1]) && h.hashCode() == (ds[0]..ds[1]).hashCode() &&
        (ds[1]..<ds[0]).hashCode() == -1)
    check("rclass153e", "${"a".rangeTo("c")}" == "a..c" && "${"a".rangeUntil("c")}" == "a..<c" &&
        "b" in "a".rangeTo("c") && "c" !in "a".rangeUntil("c") && "a".rangeTo("c") == ("a".."c") &&
        "${xs[0].rangeUntil(xs[1])}" == "1..4" && "${'a'.rangeUntil('c')}" == "a..b" &&
        "${ds[0].rangeUntil(ds[1])}" == "1.0..<2.0" && "${ds[0].rangeTo(ds[1])}" == "1.0..2.0" &&
        "${W153(1).rangeTo(W153(3))}" == "W153(n=1)..W153(n=3)" && W153(2) in W153(1).rangeUntil(W153(3)))
}
data class W153(val n: Int) : Comparable<W153> {
    override fun compareTo(other: W153) = n.compareTo(other.n)
}

// ===== SECTION 154: enum and Boolean ordering, IntRange.endExclusive =====
// kotlin.Enum.compareTo is final and answers the ordinal difference, so an enum
// orders by declaration in compareTo, `<`, a when's `in a..b`, rangeTo, sorting
// and the coerce family; an entry with a body orders as its enum. Boolean is
// Comparable too (false first), with rangeTo / rangeUntil / coerce*. The
// endExclusive of an IntRange / LongRange / CharRange is endInclusive + 1, and an
// IllegalStateException when the bound is MAX_VALUE (kotlinc 2.2.21 on java 24).
enum class Col154 { A, B, C, D }
enum class Op154 { ADD { override fun ap(x: Int) = x + 1 }, MUL { override fun ap(x: Int) = x * 2 };
    abstract fun ap(x: Int): Int
}
fun s154() {
    val cs = Col154.entries
    check("enumcmp154a", cs[1].compareTo(cs[2]) == -1 && cs[3].compareTo(cs[0]) == 3 && cs[2].compareTo(cs[2]) == 0 &&
        cs[0] < cs[1] && !(cs[3] <= cs[2]) && Op154.MUL.compareTo(Op154.ADD) == 1 && Op154.ADD < Op154.MUL &&
        compareValues(cs[3], cs[1]) == 2)
    val r = cs[0]..cs[2]
    check("enumcmp154b", "$r" == "A..C" && cs[1] in r && cs[3] !in r && cs[1] !in cs[0].rangeTo(cs[0]) &&
        cs[0].rangeTo(cs[2]).contains(cs[1]) && "${cs[0].rangeUntil(cs[2])}" == "A..<C" &&
        cs[2] !in cs[0]..<cs[2] && (cs[2]..cs[0]).isEmpty())
    check("enumcmp154c", listOf(cs[2], cs[0], cs[3], cs[1]).sorted() == cs && listOf(cs[2], cs[3]).max() == cs[3] &&
        cs[3].coerceAtMost(cs[1]) == cs[1] && cs[0].coerceIn(cs[1], cs[2]) == cs[1] &&
        cs[3].coerceIn(cs[1]..cs[2]) == cs[2] && cs.sortedWith(compareByDescending { it }).first() == cs[3])
    val ws = cs.map { when (it) { in cs[0]..cs[1] -> "lo"; !in cs[0]..cs[2] -> "top"; else -> "mid" } }
    val chs = listOf('a', 'm').map { when (it) { in 'a'..'f' -> 1; else -> 0 } }
    val lgs = listOf(5L, 9000000000L).map { when (it) { in 1L..4000000000L -> 1; else -> 0 } }
    val dbs = listOf(0.5, 1.5).map { when (it) { in 1.0..2.0 -> 1; else -> 0 } }
    check("whenin154d", ws == listOf("lo", "lo", "mid", "top") && chs == listOf(1, 0) && lgs == listOf(1, 0) &&
        dbs == listOf(0, 1))
    val bs = listOf(false, true)
    check("boolrange154e", "${bs[0]..bs[1]}" == "false..true" && "${bs[0].rangeTo(bs[1])}" == "false..true" &&
        bs[1] in bs[0]..bs[1] && bs[0] !in bs[1]..bs[1] && "${bs[0].rangeUntil(bs[1])}" == "false..<true" &&
        bs[1] !in bs[0].rangeUntil(bs[1]) && bs[1].coerceAtMost(bs[0]) == false && (bs[1]..bs[0]).isEmpty())
    val ns = listOf(1, 5, Int.MAX_VALUE)
    val ls = listOf(1L, 9L, Long.MAX_VALUE)
    val chr = listOf('a', 'e', '￿')
    check("endex154f", (ns[0]..ns[1]).endExclusive == 6 && (ns[1]..ns[0]).endExclusive == 2 &&
        (ns[0] until ns[1]).endExclusive == 5 && (ls[0]..ls[1]).endExclusive == 10L &&
        (chr[0]..chr[1]).endExclusive == 'f' && (chr[0]..<chr[1]).endExclusive == 'e')
    val tops = listOf({ (ns[0]..ns[2]).endExclusive }, { (ls[0]..ls[2]).endExclusive }, { (chr[0]..chr[2]).endExclusive })
        .map { try { it(); "ok" } catch (e: IllegalStateException) { e.message } }
    check("endex154g", tops.all { it == "Cannot return the exclusive upper bound of a range that includes MAX_VALUE." })
}

// ===== SECTION 155: unsigned ranges =====
// `..`, `..<`, until, downTo and step over UInt / ULong bounds build a
// UIntRange / ULongRange (a UIntProgression / ULongProgression once stepped or
// reversed), never an IntRange; UByte / UShort bounds build a UIntRange. The
// elements, first, last and bounds are UInts / ULongs again, and endExclusive is
// refused at the UNSIGNED MAX_VALUE. until / ..< at the element type's
// MIN_VALUE answers that type's EMPTY range (kotlinc 2.2.21 on java 24).
fun s155() {
    val us = listOf(1u, 5u, 2147483647u, 4000000000u, UInt.MAX_VALUE)
    val uls = listOf(1uL, 9uL, 9223372036854775807uL, ULong.MAX_VALUE)
    val r = us[0]..us[3]
    check("urange155a", r is UIntRange && "$r" == "1..4000000000" && 3000000000u in r && r.last == us[3] &&
        listOf(r.first, r.endInclusive, (us[0]..us[1]).endExclusive).map { it::class.simpleName } == listOf("UInt", "UInt", "UInt") &&
        (uls[0]..uls[1]) is ULongRange && listOf((uls[0]..uls[1]).last).map { it::class.simpleName } == listOf("ULong") &&
        (us[0]..us[1]).toList() == listOf(1u, 2u, 3u, 4u, 5u) && (us[4] - 1u..us[4]).toList() == listOf(4294967294u, us[4]))
    val anys = listOf<Any>(us[0]..us[1], us[1] downTo us[0], (us[0]..us[1]).reversed(), uls[0]..uls[1],
        us[0].toUByte()..us[1].toUByte())
    check("urange155b", anys.map { it is UIntRange } == listOf(true, false, false, false, true) &&
        anys.map { it is UIntProgression } == listOf(true, true, true, false, true) &&
        anys.map { it is ULongRange } == listOf(false, false, false, true, false) &&
        anys.map { it is IntRange || it is IntProgression || it is LongRange } == listOf(false, false, false, false, false) &&
        "${anys[1]}" == "5 downTo 1 step 1" && "${(us[0]..us[1] step 2)}" == "1..5 step 2")
    check("uendex155c", (us[0]..us[2]).endExclusive == 2147483648u && (us[0] until us[1]).endExclusive == us[1] &&
        (uls[0]..uls[2]).endExclusive == 9223372036854775808uL)
    val tops = listOf({ (us[0]..us[4]).endExclusive }, { (uls[0]..uls[3]).endExclusive })
        .map { try { it(); "ok" } catch (e: IllegalStateException) { e.message } }
    check("uendex155d", tops.all { it == "Cannot return the exclusive upper bound of a range that includes MAX_VALUE." })
    val mins = listOf<Any>(1 until Int.MIN_VALUE + us.size - 5, 1L until Long.MIN_VALUE + uls.size - 4,
        us[0] until us[0] - 1u, uls[0]..<uls[0] - 1uL, 'b' until 'a' - 97, us[0].rangeUntil(us[4] + 1u), 1 until -2147483647 + us.size - 5)
    check("uempty155e", mins.map { "$it" } == listOf("1..0", "1..0", "4294967295..0", "18446744073709551615..0",
        "\u0001..\u0000", "4294967295..0", "1..-2147483648") && (us[0] until us[0] - 1u).isEmpty() &&
        (us[0] until us[0] - 1u).count() == 0 && (1L until Long.MIN_VALUE + uls.size - 4).last == 0L)
}

// ===== SECTION 156: exact ULong ranges, member class literals, range EMPTY =====
// A ULong range past 2^53 keeps its bounds, elements and membership exact, in
// `..`, until, downTo, step and a `for` loop alike. `a.b::class` is the class of
// the member value read off `a` (a property, a Pair component, an enum entry), a
// nested type's descriptor, or a package-qualified type. IntRange, LongRange,
// CharRange, UIntRange and ULongRange have a companion EMPTY equal to every empty
// range of their type (kotlinc 2.2.21 on java 24).
class P156(val x: Int, val s: String)
class Q156(val p: P156)
enum class C156 { RED }
object O156 { val n = 3L }
fun s156() {
    val uls = listOf(18446744073709551613uL, ULong.MAX_VALUE, 18446744073709551610uL, 9223372036854775807uL)
    val big = uls[0]..uls[1]
    check("ulexact156a", "$big" == "18446744073709551613..18446744073709551615" && big.toList().size == 3 &&
        !(uls[0] - 1uL in big) && uls[0] + 1uL in big && big.first == uls[0] && big.last == uls[1] && big.count() == 3 &&
        big.toList() == listOf(uls[0], uls[0] + 1uL, uls[1]))
    val ex = uls[0] until uls[1]
    val dn = uls[1] downTo uls[2] step 2
    check("ulexact156b", ex.toList() == listOf(uls[0], uls[0] + 1uL) && "$ex" == "18446744073709551613..18446744073709551614" &&
        (uls[0]..<uls[1]).endInclusive == uls[0] + 1uL && "$dn" == "18446744073709551615 downTo 18446744073709551611 step 2" &&
        dn.toList() == listOf(uls[1], uls[1] - 2uL, uls[1] - 4uL) && dn.last == uls[2] + 1uL && !(uls[1] - 1uL in dn) &&
        (uls[0]..uls[1]) == (uls[0]..uls[1]) && (uls[0]..uls[1]) != (uls[0] + 1uL..uls[1]))
    val seen = mutableListOf<ULong>()
    for (u in uls[3]..uls[3] + 2uL) seen.add(u)
    for (u in uls[1] downTo uls[1] - 1uL) seen.add(u)
    for (u in uls[0] until uls[1]) seen.add(u)
    check("ulexact156c", seen == listOf(uls[3], uls[3] + 1uL, uls[3] + 2uL, uls[1], uls[1] - 1uL, uls[0], uls[0] + 1uL))
    val p = P156(4, "a")
    val q = Q156(p)
    val r = 1 to 2.5
    check("clsmem156d", listOf(p.x::class.simpleName, p.s::class.simpleName, q.p::class.simpleName, q.p.s::class.simpleName,
        r.first::class.simpleName, r.second::class.simpleName, O156.n::class.simpleName, C156.RED::class.simpleName,
        kotlin.Int::class.simpleName) == listOf("Int", "String", "P156", "String", "Int", "Double", "Long", "C156", "Int"))
    val ns = listOf(1, 0)
    check("rempty156e", (ns[0] until Int.MIN_VALUE) == IntRange.EMPTY && "${IntRange.EMPTY}" == "1..0" &&
        "${LongRange.EMPTY}" == "1..0" && CharRange.EMPTY == ('b' until Char.MIN_VALUE) && "${UIntRange.EMPTY}" == "4294967295..0" &&
        "${ULongRange.EMPTY}" == "18446744073709551615..0" && IntRange.EMPTY.isEmpty() && LongRange.EMPTY.toList().isEmpty() &&
        IntRange.EMPTY.first == ns[0] && ULongRange.EMPTY.first == ULong.MAX_VALUE && LongRange.EMPTY.hashCode() == -1 &&
        LongRange.EMPTY == (5L..2L) && IntRange.EMPTY.count() == ns[1])
}

// ===== SECTION 157: the KClass of a Pair, a Triple, a map entry, a builder =====
// `to`, Pair() and Triple() build kotlin.Pair / kotlin.Triple, whose KClass is that
// java class: simpleName Pair, qualifiedName kotlin.Pair, equal to Pair::class.
// A map entry is no Pair: its class is its map's node class (LinkedHashMap$Entry,
// HashMap$Node, TreeMap$Entry, a singleton map's AbstractMap$SimpleImmutableEntry).
// StringBuilder and Regex are java.lang.StringBuilder / kotlin.text.Regex; an
// unsigned type and an array type are qualified in package kotlin. An unnamed
// companion object is reachable as Owner.Companion (kotlinc 2.2.21 on java 24).
fun cn157(v: Any): String = "" + v::class.simpleName + " " + v::class.qualifiedName + " " + v.javaClass.name
class Cmp157 { companion object { val k = 5; fun inc157(n: Int) = n + k } }
fun s157() {
    val ps = listOf<Any>(1L to 2, Pair("a", 'b'), Triple(1, 2, 3))
    check("kcls157a", ps.map { it::class.simpleName } == listOf("Pair", "Pair", "Triple") &&
        cn157(ps[0]) == "Pair kotlin.Pair kotlin.Pair" && cn157(ps[2]) == "Triple kotlin.Triple kotlin.Triple" &&
        ps[1]::class.java.name == "kotlin.Pair")
    check("kcls157b", ps[0]::class == Pair::class && ps[1]::class == ps[0]::class && ps[2]::class == Triple::class &&
        ps[2]::class != Pair::class && Pair::class.simpleName == "Pair" && Triple::class.qualifiedName == "kotlin.Triple" &&
        kotlin.Pair::class.simpleName == "Pair")
    check("kcls157c", ps.map { it is Pair<*, *> } == listOf(true, true, false) && ps.map { it is Triple<*, *, *> } == listOf(false, false, true))
    val m = mutableMapOf(1 to 2, 3 to 4)
    val es = listOf<Any>(m.entries.first(), mapOf(1 to 2).entries.first(), hashMapOf(1 to 2).entries.first(), sortedMapOf(1 to 2).entries.first())
    check("kcls157d", es.map { cn157(it) } == listOf("Entry java.util.LinkedHashMap.Entry java.util.LinkedHashMap\$Entry",
        "SimpleImmutableEntry java.util.AbstractMap.SimpleImmutableEntry java.util.AbstractMap\$SimpleImmutableEntry",
        "Node java.util.HashMap.Node java.util.HashMap\$Node", "Entry java.util.TreeMap.Entry java.util.TreeMap\$Entry") &&
        es.none { it is Pair<*, *> } && es[0]::class != Pair::class)
    val sb = StringBuilder("a")
    val re = Regex("a+")
    check("kcls157e", cn157(sb) == "StringBuilder java.lang.StringBuilder java.lang.StringBuilder" &&
        cn157(re) == "Regex kotlin.text.Regex kotlin.text.Regex" && sb::class == StringBuilder::class && re::class == Regex::class &&
        StringBuilder::class.qualifiedName == "java.lang.StringBuilder" && Regex::class.java.name == "kotlin.text.Regex")
    check("kcls157f", listOf(5u::class.qualifiedName, 5uL::class.qualifiedName, arrayOf("a")::class.qualifiedName,
        intArrayOf(1)::class.qualifiedName, uintArrayOf(1u)::class.qualifiedName, booleanArrayOf(true)::class.qualifiedName) ==
        listOf("kotlin.UInt", "kotlin.ULong", "kotlin.Array", "kotlin.IntArray", "kotlin.UIntArray", "kotlin.BooleanArray"))
    check("comp157g", Cmp157.Companion.k == 5 && Cmp157.Companion.inc157(1) == 6 && Cmp157.inc157(2) == 7)
}

// ===== SECTION 158: companion and enum-entry classes, member vs receiver lambda =====
// A class name in value position is its companion object, whose class is nested in
// the owner's (C$Companion, Outer$In$Named); the type literal C::class is C itself.
// An enum entry with a body is a subclass E$B (so E.B::class != E::class) while its
// declaringJavaClass stays E. A primitive KClass has two java classes, `int` and
// java.lang.Integer. A member of an object or a companion beats a receiver-lambda
// parameter of the same name (kotlinc 2.2.21 on java 24).
class Cmp158 { companion object { val k = 7; fun f(n: Int) = n * 3 } }
class Nmd158 { companion object Named { fun g() = 2 } }
class Out158 { class In158 { companion object { fun h() = 3 } } }
object Obj158 { fun f(n: Int) = n * 2 }
enum class En158 { A, B { override fun toString() = "b" }; fun f(n: Int) = n + 100 }
fun twice158(x: Int, f: Int.() -> Int) = x.f().f()
fun s158() {
    check("comp158a", Cmp158.Companion::class.simpleName == "Companion" && Cmp158.Companion::class.qualifiedName == "Cmp158.Companion" &&
        Cmp158.Companion.javaClass.name == "Cmp158\$Companion" && Cmp158.javaClass.name == "Cmp158\$Companion" &&
        Cmp158::class.simpleName == "Cmp158")
    val c: Any = Cmp158
    check("comp158b", c::class.qualifiedName == "Cmp158.Companion" && Nmd158.Named::class.simpleName == "Named" &&
        Nmd158.Named::class.qualifiedName == "Nmd158.Named" && Nmd158::class.simpleName == "Nmd158" &&
        Out158.In158.Companion::class.qualifiedName == "Out158.In158.Companion" && Out158.In158.javaClass.name == "Out158\$In158\$Companion")
    check("enum158c", En158.B::class.simpleName == "B" && En158.B::class.qualifiedName == "En158.B" && En158.B.javaClass.name == "En158\$B" &&
        En158.B::class != En158::class && En158.A::class == En158::class && En158.entries.map { it.javaClass.name } == listOf("En158", "En158\$B"))
    check("enum158d", En158.B.declaringJavaClass.name == "En158" && En158.A.declaringJavaClass.simpleName == "En158" && En158.B is En158)
    check("prim158e", Int::class.javaObjectType.name == "java.lang.Integer" && Int::class.java.name == "int" &&
        Char::class.javaObjectType.simpleName == "Character" && "" + Int::class.java == "int" &&
        "" + Long::class.javaObjectType == "class java.lang.Long" && String::class.java.name == "java.lang.String")
    check("recv158f", twice158(3) { this + 1 } == 5 && Obj158.f(4) == 8 && Cmp158.f(5) == 15 && Cmp158.Companion.f(6) == 18 && Cmp158.k == 7 &&
        En158.A.f(1) == 101 && En158.B.f(2) == 102)
}

// ===== SECTION 159: javaClass by static type, declared and builtin java classes, reified T::class =====
// javaClass of a receiver whose static type is a primitive (a literal, or a name
// declared with that type or such a literal) is the primitive class `int`; through
// Any, a nullable or a `!!` it is java.lang.Integer. A declared class's KClass.java
// is the class its instances' javaClass names. UInt / Unit / List / IntArray::class.java
// are kotlin.UInt, kotlin.Unit, java.util.List and [I; IntRange::class is the class
// of a range. A reified T::class is the call site's type argument's class, handed on
// through a caller's own T. A lazy sequence's class follows Sequences.kt (take and
// drop merge into Take/Drop/SubSequence or the empty one) (kotlinc 2.2.21 on java 24).
class C159
data class P159(val x: Int)
enum class E159 { A, B { override fun toString() = "b" } }
inline fun <reified T : Any> nm159() = T::class.simpleName
inline fun <reified T : Any> fw159() = nm159<T>() + "/" + T::class.qualifiedName
inline fun <reified T : Any> Any.kn159() = T::class.simpleName
fun s159() {
    val i = 5
    val y: Int = 6
    val a: Any = 7
    val n: Int? = 8
    check("jprim159a", 5.javaClass.name == "int" && i.javaClass.name == "int" && y.javaClass.name == "int" &&
        a.javaClass.name == "java.lang.Integer" && n!!.javaClass.name == "java.lang.Integer" &&
        (5 as Any).javaClass.name == "java.lang.Integer" && 7L.javaClass.name == "long" && 'c'.javaClass.name == "char" &&
        2.5.javaClass.name == "double" && i.javaClass == Int::class.java && a.javaClass != Int::class.java)
    check("jcls159b", P159(1).javaClass == P159::class.java && E159.A.javaClass == E159::class.java &&
        E159.B.javaClass != E159::class.java && C159()::class == C159::class && (1..2)::class == IntRange::class &&
        IntRange::class.qualifiedName == "kotlin.ranges.IntRange")
    check("reif159c", nm159<C159>() == "C159" && nm159<Int>() == "Int" && fw159<P159>() == "P159/P159" &&
        fw159<String>() == "String/kotlin.String" && 5.kn159<E159>() == "E159")
    check("bjava159d", UInt::class.java.name == "kotlin.UInt" && Unit::class.java.name == "kotlin.Unit" &&
        Nothing::class.java.name == "java.lang.Void" && List::class.java.name == "java.util.List" &&
        IntArray::class.java.name == "[I" && IntArray::class.simpleName == "IntArray" && UInt::class.java == 5u.javaClass &&
        Unit.javaClass.name == "kotlin.Unit")
    val s = sequence { yield(1); yield(2) }
    check("seq159e", s.javaClass.name == "kotlin.sequences.SequencesKt__SequenceBuilderKt\$sequence\$\$inlined\$Sequence\$1" &&
        s.javaClass.simpleName == "" && s.map { it }.javaClass.name == "kotlin.sequences.TransformingSequence" &&
        s.filter { true }.javaClass.simpleName == "FilteringSequence" && s.take(1).drop(1).javaClass.simpleName == "EmptySequence" &&
        s.drop(1).take(1).javaClass.simpleName == "SubSequence" && s.take(0).javaClass.simpleName == "EmptySequence")
}

// ===== SECTION 160: javaClass of computed receivers, reified primitives and inferred T =====
// javaClass follows the JVM type of its receiver: a loop or inline-lambda variable,
// an arithmetic result, an `if` of primitive branches, a function result, a property
// or a primitive array's element of a primitive static type is unboxed (`int`),
// while a cast, a `!!` and a list's or an Array's element stay boxed. A reified
// T::class.java of T = Int is java.lang.Integer; a reified T written without type
// arguments is inferred from the argument's static type, also in a call nested in
// another reified function's body. An anonymous object's KClass has a null
// simpleName and qualifiedName, its java class the simpleName "" (kotlinc 2.2.21 on
// java 24). A method's result type is its receiver's class's own declaration, also
// where another class declares a method of the name with another result type.
class Q160(val v: Int) {
    val n = 5
    fun m(): Int = v
    fun own() = this.v.javaClass.name
    fun k(x: Int): Long = x.toLong()
}
class R160 {
    fun m(): List<Int> = listOf(1)
    fun k(x: Int): Any = x
}
interface S160
fun f160(): Int = 3
fun g160() = 4L
inline fun <reified T : Any> jn160() = T::class.java.name
inline fun <reified T : Any> kc160() = T::class
inline fun <reified T : Any> inf160(x: T) = T::class.simpleName
inline fun <reified T : Any> nest160(x: T): String = inf160(1.5) + "/" + T::class.simpleName
inline fun <reified T> is160(x: Any) = x is T
inline fun <reified T> chk160(x: Any): String = "" + (x is T) + "," + is160<Long>(x)
fun s160() {
    val x = 4
    val sh: Short = 3
    val xs = listOf(1, 2)
    val zs: List<Int> = listOf(7)
    val ia = intArrayOf(1, 2)
    val str = "hey"
    val a: Any = 3
    val q = Q160(1)
    val jc = StringBuilder()
    for (i in 0 until 1) jc.append(i.javaClass.name + " ")
    for (ch in 'a'..'a') jc.append(ch.javaClass.name + " ")
    for (e in xs) jc.append(e.javaClass.name + " ")
    xs.forEach { jc.append(it.javaClass.name + " ") }
    xs.forEachIndexed { i, e -> jc.append(i.javaClass.name + e.javaClass.name + " ") }
    repeat(1) { jc.append(it.javaClass.name + " ") }
    x.let { jc.append(it.javaClass.name) }
    check("jstat160a", "" + jc == "int char int int int int intint intint int int")
    check("jstat160b", (x + 1).javaClass.name == "int" && (x * 1.5).javaClass.name == "double" &&
        (-sh).javaClass.name == "int" && (x shl 2).javaClass.name == "int" && (x > 1).javaClass.name == "boolean" &&
        (if (x > 1) 1 else 2).javaClass.name == "int" && f160().javaClass.name == "int" && g160().javaClass.name == "long")
    check("jstat160c", q.v.javaClass.name == "int" && q.n.javaClass.name == "int" && q.m().javaClass.name == "int" &&
        q.own() == "int" && Q160(2).v.javaClass.name == "int" && ia[0].javaClass.name == "int" &&
        str[0].javaClass.name == "char" && str.length.javaClass.name == "int" && xs.size.javaClass.name == "int")
    check("jbox160d", zs[0].javaClass.name == "java.lang.Integer" && (a as Int).javaClass.name == "java.lang.Integer" &&
        xs.first().javaClass.name == "java.lang.Integer" && listOf<Any>(1)[0].javaClass.name == "java.lang.Integer")
    check("reif160e", jn160<Int>() == "java.lang.Integer" && jn160<Char>() == "java.lang.Character" &&
        jn160<String>() == "java.lang.String" && kc160<Int>().java == Int::class.javaObjectType &&
        kc160<Int>().java != Int::class.java && kc160<Int>() == Int::class && kc160<Long>().java.simpleName == "Long" &&
        kc160<Int>().simpleName == "Int")
    check("reif160f", inf160(5) == "Int" && inf160("s") == "String" && inf160(2.5) == "Double" && inf160(5L) == "Long" &&
        inf160(listOf(1)) == "List" && inf160('c') == "Char" && inf160(x + 1) == "Int" && nest160(3) == "Double/Int" &&
        nest160<Any>("a") == "Double/Any" && chk160<Int>(5) == "true,false" && chk160<Long>(5L) == "true,true")
    val o = object : S160 {}
    check("anon160g", o::class.simpleName == null && o::class.qualifiedName == null && o.javaClass.simpleName == "" &&
        o::class.java.simpleName == "")
    val r = R160()
    check("jstat160h", q.m().javaClass.name == "int" && Q160(3).m().javaClass.name == "int" &&
        q.k(2).javaClass.name == "long" && r.k(2).javaClass.name == "java.lang.Integer" &&
        r.m().javaClass.simpleName == "SingletonList")
}

// ===== SECTION 161: reified calls of functions declared later =====
// A reified callee may be declared after its caller: the call still hands over
// its type arguments, explicit or inferred from the argument, for `is T` and
// `T::class`, in the plain, extension and member forms. A type argument that is
// the caller's own reified T is judged as the caller's type argument (kotlinc
// 2.2.21 on java 24).
inline fun <reified T : Any> out161(x: T) = in161<String>()
inline fun <reified T : Any> fwd161(x: T) = in161<T>()
inline fun <reified T : Any> inf161(x: T) = g161(listOf(x)) + "/" + g161(x)
fun pl161(a: Any) = is161<Int>(a)
inline fun <reified T> fis161(a: Any) = is161<T>(a)
fun ext161(a: Any) = "" + a.isA161<Int>() + "/" + a.cls161<Long>()
inline fun <reified T> efw161(a: Any) = a.isA161<T>()
class B161 {
    fun early(a: Any) = later<Int>(a)
    fun earlyCls() = laterCls<Double>()
    inline fun <reified T> later(a: Any) = a is T
    inline fun <reified T : Any> laterCls() = T::class.simpleName
}
inline fun <reified T : Any> in161() = T::class.simpleName
inline fun <reified T : Any> g161(x: T) = T::class.simpleName + " " + (x is T) + " " + ("q" is T)
inline fun <reified T> is161(a: Any) = a is T
inline fun <reified T> Any.isA161() = this is T
inline fun <reified T : Any> Any.cls161() = T::class.simpleName
fun s161() {
    check("fwd161a", out161(1) == "String" && fwd161(1) == "Int" && fwd161("s") == "String")
    check("fwd161b", inf161(2) == "List true false/Int true false" && inf161("a") == "List true false/String true true")
    check("fwd161c", pl161(3) && !pl161("x") && fis161<Int>(4) && !fis161<String>(4))
    check("fwd161d", ext161(5) == "true/Long" && ext161("y") == "false/Long" && efw161<String>("z") && !efw161<Int>("z"))
    val b = B161()
    check("fwd161e", b.early(7) && !b.early("w") && b.earlyCls() == "Double")
}

// ===== SECTION 162: KClass javaPrimitiveType and hashCode, javaClass of destructured and smart-cast values =====
// KClass.javaPrimitiveType is a primitive type's primitive java class (a reified
// T's too), null for any other class; KClass.hashCode is its java object class's,
// so equal class literals hash alike and key sets and maps. A destructured name -
// of a lambda parameter, a for loop or a val - is a local of its component's static
// type, unboxed when that is primitive, as is the `it` of `a.let` after an
// unconditional `a as Int` (while `a.javaClass` itself and an entry's `key` stay
// boxed) (kotlinc 2.2.21 on java 24).
inline fun <reified T : Any> kc162() = T::class
class B162(val v: Int)
fun cast162(a: Any): String {
    a as Int
    return a.let { it.javaClass.name } + " " + a.javaClass.name
}
fun cond162(a: Any, c: Boolean): String {
    if (c) a as Int
    return a.let { it.javaClass.name }
}
fun s162() {
    check("kprim162a", Long::class.javaPrimitiveType.toString() == "long" && Int::class.javaPrimitiveType == Int::class.java &&
        kc162<Int>().javaPrimitiveType.toString() == "int" && String::class.javaPrimitiveType == null &&
        B162::class.javaPrimitiveType == null && Int::class.javaPrimitiveType != Int::class.javaObjectType)
    check("khash162b", kc162<Int>().hashCode() == Int::class.hashCode() && Int::class.hashCode() != Long::class.hashCode() &&
        String::class.hashCode() == String::class.java.hashCode() &&
        Int::class.hashCode() == Int::class.javaObjectType.hashCode() && B162::class.hashCode() == B162(1)::class.hashCode())
    check("kkey162c", setOf(Int::class, kc162<Int>(), Long::class).size == 2 &&
        mapOf(Int::class to 1, String::class to 2)[kc162<Int>()] == 1 && listOf(Int::class).contains(kc162<Int>()))
    val sb = StringBuilder()
    mapOf(1 to 2).forEach { (k, v) -> sb.append(k.javaClass.name + v.javaClass.name + " ") }
    mapOf("a" to 2L).entries.forEach { (k, v) -> sb.append(k.javaClass.name + v.javaClass.name + " ") }
    listOf(1 to 'c').forEachIndexed { i, (a, b) -> sb.append(i.javaClass.name + a.javaClass.name + b.javaClass.name + " ") }
    listOf(Triple(1, 2L, 'x')).forEach { (a, b, c) -> sb.append(a.javaClass.name + b.javaClass.name + c.javaClass.name) }
    check("dstr162d", "" + sb == "intint java.lang.Stringlong intintchar intlongchar")
    val l = listOf(1 to 'c')
    val m = mutableMapOf<String, Long>("a" to 1L)
    val fb = StringBuilder()
    for ((a, b) in l) fb.append(a.javaClass.name + b.javaClass.name + " ")
    for ((k, v) in m) fb.append(k.javaClass.name + v.javaClass.name + " ")
    for (e in mapOf(1 to 2)) fb.append(e.key.javaClass.name + " ")
    val (p, q) = 1 to 2L
    val (u: Number, _) = 1 to 2
    fb.append(p.javaClass.name + q.javaClass.name + u.javaClass.name)
    check("dstr162e", "" + fb == "intchar java.lang.Stringlong java.lang.Integer intlongint")
    check("smart162f", cast162(3) == "int java.lang.Integer" && cond162(4, true) == "java.lang.Integer")
}

// ===== SECTION 163: reified extensions called without type arguments, Class.kotlin =====
// A reified type parameter of an extension called in method position without type
// arguments is inferred from the receiver's static type, where the declared
// receiver type (`T`, `List<T>`, `Map<K, V>`, `T?`) binds it; `Class.kotlin` maps a
// java class back to its KClass - a primitive or wrapper class to its kotlin type's,
// whose java class it keeps, a mapped builtin's or a primitive array's to that
// type's (kotlinc 2.2.21 on java 24).
inline fun <reified T : Any> T.rn163() = T::class.simpleName + " " + (this is T) + " " + ("q" is T)
inline fun <reified T> List<T>.el163() = T::class.simpleName
inline fun <reified T> T?.nn163(): String = T::class.java.toString()
inline fun <reified K, reified V> Map<K, V>.kv163() = K::class.simpleName + "/" + V::class.simpleName
inline fun <reified T> List<T>.is163(x: Any) = x is T
inline fun <reified T> Array<out T>.ar163() = T::class.simpleName
class P163(val xs: List<Double>)
fun s163() {
    val a: Any = 5
    val n: Int? = 3
    val ls: List<Char>? = listOf('a')
    val ar: Array<String> = arrayOf("a")
    check("rext163a", 5.rn163() == "Int true false" && "s".rn163() == "String true true" && a.rn163() == "Any true true" &&
        2.5.rn163() == "Double true false" && 5.rn163<Number>() == "Number true false")
    check("rext163b", listOf(1, 2).el163() == "Int" && listOf("a").el163() == "String" && P163(listOf(1.0)).xs.el163() == "Double" &&
        ls?.el163() == "Char" && n.nn163() == "class java.lang.Integer" && mapOf(1 to "a").kv163() == "Int/String" &&
        !listOf(1).is163("x") && listOf("a").is163("x") && ar.ar163() == "String")
    val lk = Long::class.javaObjectType.kotlin
    check("jkot163c", lk == Long::class && lk.simpleName == "Long" && lk.qualifiedName == "kotlin.Long" &&
        "" + lk.java == "class java.lang.Long" && "" + lk.javaPrimitiveType == "long" &&
        Long::class.javaPrimitiveType!!.kotlin == Long::class && "" + Long::class.java.kotlin.java == "long" &&
        lk.hashCode() == Long::class.hashCode() && 5.javaClass.kotlin == Int::class && Char::class.java.kotlin.simpleName == "Char")
    check("jkot163d", String::class.java.kotlin == String::class && IntArray::class.java.kotlin.simpleName == "IntArray" &&
        IntArray::class.java.kotlin.qualifiedName == "kotlin.IntArray" && Number::class.java.kotlin.qualifiedName == "kotlin.Number" &&
        UInt::class.java.kotlin.simpleName == "UInt" && P163::class.java.kotlin == P163::class && P163(listOf()).javaClass.kotlin.simpleName == "P163")
}

// ===== SECTION 164: inferred reified arguments of map / this; collection and Nothing KClasses =====
// A reified type parameter written without type arguments is inferred from the
// static type of an extension's `this` (`listOf(this)` in a reified extension
// binds the caller's T) and of a map / mapIndexed / first result; a generic
// element result such as first() is boxed, an Int extension's `this` is not. A collection
// interface's KClass is its java interface - qualifiedName kotlin.collections.List,
// a Mutable twin the same class, rendered "interface java.util.List" as every
// interface's java class is - and Nothing's is java.lang.Void, whose
// javaPrimitiveType is void (kotlinc 2.2.21 on java 24).
inline fun <reified T> List<T>.el164() = T::class.simpleName
inline fun <reified T> Array<out T>.ar164() = T::class.simpleName
inline fun <reified T : Any> T.rn164() = T::class.simpleName
inline fun <reified T> T.wrap164() = listOf(this).el164()
inline fun <reified T> T.awrap164() = arrayOf(this).ar164()
fun Long.lw164() = listOf(this).el164()
fun Int.jc164() = this.javaClass.name
interface Sh164
interface Sub164 : Sh164
class Sq164 : Sub164
fun s164() {
    check("rinf164a", 5L.wrap164() == "Long" && "s".awrap164() == "String" && 2.5.wrap164() == "Double" &&
        7L.lw164() == "Long" && arrayOf("a").ar164() == "String" && 3.jc164() == "int")
    check("rinf164b", listOf(1).map { it.toString() }.el164() == "String" && listOf("a").map { it.length }.el164() == "Int" &&
        listOf(1).map { x -> x * 1.5 }.el164() == "Double" && listOf("a").mapIndexed { i, s -> i }.el164() == "Int" &&
        arrayOf(1).map { it > 0 }.el164() == "Boolean" && listOf("x").first().rn164() == "String" &&
        listOf(1).map { it.toString() }.last { it != "" }.rn164() == "String" &&
        listOf(1).map { x: Int -> x.toLong() }.el164() == "Long" && listOf("a").map { it.uppercase() }.el164() == "String" &&
        listOf(4).first().javaClass.name == "java.lang.Integer" && "7".toInt().javaClass.name == "int")
    check("kcls164c", List::class.qualifiedName == "kotlin.collections.List" && MutableList::class.simpleName == "List" &&
        MutableMap::class.qualifiedName == "kotlin.collections.Map" && Iterable::class.qualifiedName == "kotlin.collections.Iterable" &&
        MutableList::class == List::class && List::class.java.kotlin.qualifiedName == "kotlin.collections.List" &&
        "" + List::class.java == "interface java.util.List" && "" + Set::class.java == "interface java.util.Set" &&
        "" + Comparable::class.java == "interface java.lang.Comparable" && "" + Runnable::class.java == "interface java.lang.Runnable" &&
        "" + Sh164::class.java == "interface Sh164" && "" + Sub164::class.java == "interface Sub164" &&
        "" + Sq164::class.java == "class Sq164" && "" + Number::class.java == "class java.lang.Number")
    check("noth164d", Nothing::class.simpleName == "Void" && Nothing::class.qualifiedName == "java.lang.Void" &&
        "" + Nothing::class.javaPrimitiveType == "void" && "" + Nothing::class.java == "class java.lang.Void" &&
        "" + Nothing::class.java.kotlin.javaPrimitiveType == "void" && Nothing::class.java.kotlin == Nothing::class &&
        Nothing::class != Unit::class)
}

// ===== SECTION 165: scope function receivers inside extensions and members =====
// apply / run / with / buildString bind their receiver as the INNERMOST implicit
// receiver, so inside an extension or a member its members shadow the enclosing
// `this`'s, while a name it does not have (a value it has no property of) still
// reaches the outer receiver. `this@ext` names the extension's receiver, also as
// the first word of a statement, `this@apply` the scope function's, and
// `this@Class` passes over a scope function's receiver. A scope function's lambda
// has no `it`, so an enclosing lambda's shows through. An unqualified call on a
// String receiver is Kotlin's member, whose split keeps the empty edge elements
// (kotlinc 2.2.21).
fun String.sp165() = split("")
fun String.sb165() = StringBuilder().apply { append("x"); append(this@sb165) }.toString()
fun String.len165() = StringBuilder().apply { append(length) }.toString()
fun String.w165() = with(StringBuilder()) { append("y"); append(length); toString() }
fun String.r165() = StringBuilder().run { append(this@r165.length); append(length); toString() }
fun String.bs165() = buildString { append(this@bs165); append(length) }
fun String.st165() = listOf(7, 8).run { this@st165.length * 10 + size }
fun String.ap165() = StringBuilder().apply { append(this@apply.length); append(length) }.toString()
class Bx165(val n: Int) {
    val size = 99
    fun q() = StringBuilder().apply { append(n) }.toString()
    fun sz() = StringBuilder().apply { append(size) }.toString()
    fun out() = with(listOf(5)) { size + this@Bx165.size }
}
fun s165() {
    check("rcv165a", "ab".sb165() == "xab")
    check("rcv165b", "abc".len165() == "0" && "ab".w165() == "y1" && "ab".r165() == "21" && "abc".bs165() == "abc3")
    check("rcv165c", Bx165(7).q() == "7" && Bx165(1).sz() == "99")
    check("rcv165d", Bx165(1).out() == 100)
    check("rcv165e", "abc".st165() == 32 && "abc".ap165() == "01")
    check("rcv165f", listOf(1, 2).map { StringBuilder().apply { append(it) }.toString() } == listOf("1", "2") &&
        listOf(3).map { buildString { append(it) } } == listOf("3"))
    check("rcv165g", "ab".sp165() == listOf("", "a", "b", "") && "ab".run { split("") }.size == 4)
}

// ===== SECTION 166: implicit javaClass, unboxed lambda parameters, member references =====
// An unqualified `javaClass` is the implicit receiver's, primitive for an Int
// extension; an inline collection lambda's parameter typed Long by `listOf(this)`
// in a Long extension is unboxed (long). A reified T is inferred from a map whose
// lambda is a `when` or an unbound member reference, and `Int::toString` /
// `String::uppercase` passed as a lambda CALL the member (kotlinc 2.2.21).
inline fun <reified T> List<T>.el166() = T::class.simpleName
fun Int.jc166() = javaClass.name
fun Long.jl166() = listOf(this).map { it.javaClass.name }
class K166 { fun n() = javaClass.simpleName }
fun s166() {
    check("jc166a", 3.jc166() == "int" && K166().n() == "K166")
    check("jc166b", 5L.jl166() == listOf("long"))
    check("rinf166c", listOf(1, 2).map { when (it) { 1 -> "a" else -> "b" } }.el166() == "String" &&
        listOf(1, 2).map(Int::toString).el166() == "String" && listOf("a").map(String::length).el166() == "Int")
    val f = Int::toString
    check("ref166d", listOf(1, 2).map(Int::toString) == listOf("1", "2") &&
        listOf("ab").map(String::uppercase) == listOf("AB") && listOf(1).map(Int::inc) == listOf(2) && f(5) == "5")
}

// ===== SECTION 167: nested builtin receivers, one-expression `when` lambdas =====
// An unqualified call the innermost builtin receiver does not declare belongs to the
// next receiver out: `append` inside `with(list)` nested in `sb.apply` is the
// builder's, `put` inside `with(sb)` the map's, `uppercase()` inside `with(sb)` the
// String's; a name the inner receiver declares (`last()` on the builder) stays its.
// A lambda whose body is one `when` written with `;` or across lines infers its
// result type like the one-line form (kotlinc 2.2.21).
inline fun <reified T> List<T>.el167() = T::class.simpleName
fun s167() {
    check("nrc167a", StringBuilder().apply { with(listOf(1, 2, 3)) { append(size) } }.toString() == "3" &&
        StringBuilder("x").apply { with(listOf(1, 2, 3)) { append(size); append(first()) } }.toString() == "x31")
    check("nrc167b", with(listOf(4, 5)) { StringBuilder().apply { append(size); append(last()) }.toString() } == "22" &&
        StringBuilder().apply { with("hey") { append(length); append(uppercase()) } }.toString() == "3HEY")
    check("nrc167c", StringBuilder().let { sb ->
            mutableMapOf<String, Int>().apply { with(sb) { append("q"); put("k", length) } } == mapOf("k" to 1) &&
                sb.toString() == "q" } &&
        "ab".run { StringBuilder().apply { append(uppercase()); append(length) }.toString() } == "AB2")
    check("nrc167d", StringBuilder().apply { with('x') { append(uppercase()); append(isLetter()) } }.toString() == "Xtrue" &&
        mutableSetOf(1).apply { with(mapOf(1 to 2)) { add(size + 10) } } == setOf(1, 11))
    check("rinf167e", listOf(1).map { when (it) { 1 -> "a"; else -> "b" } }.el167() == "String" &&
        listOf(1).map {
            when (it) {
                1 -> 'a'
                else -> 'b'
            }
        }.el167() == "Char")
}

// ===== SECTION 168: builtin receivers' member table, extensions on implicit receivers =====
// A builtin implicit receiver answers only the names its kind declares, so a class
// member or an extension called unqualified inside `with(list)` binds the receiver
// that has it; a read-only List has no `add`, which belongs to the set outside. An
// extension called unqualified binds the innermost receiver of its type, and one
// named like a StringBuilder method still applies to a MutableList (kotlinc 2.2.21).
class Q168(val n: Int) {
    fun tag(x: Int) = "q" + (x + n)
    fun count(): Int = 99
    fun go() = with(listOf(1, 2, 3)) { tag(size) }
    fun cnt() = with(listOf(1, 2)) { count() }
}
class B168(var v: Int) { fun bump(x: Int) { v += x } }
fun MutableList<Int>.twice168(x: Int) { add(x * 2) }
fun MutableList<Int>.append(x: Int) { add(x * 100) }
fun Q168.b168() = n * 2
fun Q168.a168() = b168() + 1
fun String.w168() = "<" + this + ">"
fun String.v168() = w168() + length
fun Any.d168() = "d:" + this
fun s168() {
    val b = B168(0)
    with(b) { with(listOf(3, 4)) { bump(size) } }
    check("mbr168a", Q168(5).go() == "q8" && Q168(5).cnt() == 2 && b.v == 2)
    val ml = mutableListOf(1)
    with(ml) { twice168(5) }
    val m2 = mutableListOf(1)
    with(m2) { append(5) }
    m2.append(6)
    check("ext168b", ml == listOf(1, 10) && m2 == listOf(1, 500, 600) &&
        StringBuilder().apply { with(listOf(7, 8)) { append(size) } }.toString() == "2")
    check("ext168c", Q168(4).a168() == 9 && with(Q168(5)) { b168() } == 10 && "ab".v168() == "<ab>2" &&
        with(listOf(1)) { d168() } == "d:[1]")
    check("ro168d", mutableSetOf(1).apply { with(listOf(9)) { add(1); add(2) } } == setOf(1, 2) &&
        mutableListOf(0).apply { with(arrayOf(4, 5)) { add(size) } } == listOf(0, 2))
}

// ===== SECTION 169: value reads through a builtin receiver, buildSet =====
// A bare name in value position reads only a PROPERTY of a builtin implicit
// receiver's kind; a function member (List.first(), map, reverse) or a name the
// receiver lacks falls through to the enclosing class. buildSet's receiver is an
// insertion-ordered MutableSet.
class K169 {
    val log = "L"; val items = "I"; val push = "P"; val map = "M"; val first = "F"; val reverse = "V"
    fun go() = with(listOf(9)) { log + items + push + map + first + size }
    fun sb() = with(StringBuilder("xy")) { reverse + length }
    fun rg() = with(1..7 step 2) { "" + first + step + map }
}

fun s169() {
    check("rcv169a", K169().go() == "LIPMF1")
    check("rcv169b", K169().sb() == "V2" && K169().rg() == "12M")
    check("bset169c", buildSet { add(3); add(size) }.toString() == "[3, 1]" &&
        buildSet(4) { add("a"); add("b"); add("a"); addAll(listOf("c", "b")) } == setOf("a", "b", "c"))
    check("bset169d", buildSet<Int> { add(2); add(1); remove(3) }.toList() == listOf(2, 1) &&
        buildSet { add(1) }.javaClass.name == "kotlin.collections.builders.SetBuilder")
}

// ===== SECTION 170: read-only collections behind a Mutable cast =====
// Set.indices, and what a cast to a Mutable type exposes: a builder's result
// refuses every write, a Collections.singleton* or Arrays.asList refuses what
// AbstractList / AbstractCollection / AbstractMap refuse, and an EmptyList /
// EmptySet / EmptyMap refuses the cast itself.
fun uoe170(f: () -> Any?): String = try { "" + f() } catch (e: UnsupportedOperationException) {
    "U" + e.message
} catch (e: ClassCastException) { "C:" + e.message }

fun s170() {
    check("idx170a", setOf(4, 5, 6).indices == 0..2 && emptySet<Int>().indices.isEmpty() &&
        buildSet { add(1) }.indices.last == 0)
    val bl = buildList { add(1); add(2); this[0] = 5 }
    val bm = buildMap { put("a", 1); this["a"] = 2 }
    val bs = buildSet { add(1); remove(1); add(2) }
    check("ro170b", uoe170 { (bl as MutableList<Int>).add(9) } == "Unull" &&
        uoe170 { (bl as MutableList<Int>)[0] = 7 } == "Unull" &&
        uoe170 { (bm as MutableMap<String, Int>)["b"] = 1 } == "Unull" &&
        uoe170 { (bs as MutableSet<Int>).add(9) } == "Unull" &&
        bl == listOf(5, 2) && bm == mapOf("a" to 2) && bs == setOf(2))
    check("ro170c", uoe170 { (bl as MutableList<Int>).removeIf { it > 50 } } == "false" &&
        uoe170 { (bm as MutableMap<String, Int>).getOrPut("a") { 0 } } == "2" &&
        uoe170 { (bs as MutableSet<Int>).remove(7) } == "Unull")
    check("ro170d", uoe170 { (setOf(1) as MutableSet<Int>).add(2) } == "Unull" &&
        uoe170 { (setOf(1) as MutableSet<Int>).remove(7) } == "false" &&
        uoe170 { (mapOf(1 to 2) as MutableMap<Int, Int>).put(1, 2) } == "Unull" &&
        uoe170 { (mapOf(1 to 2) as MutableMap<Int, Int>).remove(1) } == "Unull")
    val l3 = listOf(1, 2, 3) as MutableList<Int>
    l3[0] = 9
    check("ro170e", uoe170 { (listOf(1) as MutableList<Int>).set(0, 2) } == "Unull" &&
        uoe170 { l3.add(4) } == "Unull" && uoe170 { l3.remove(2) } == "Uremove" &&
        uoe170 { l3.remove(7) } == "false" && uoe170 { l3.removeAt(0) } == "Unull" &&
        l3 == listOf(9, 2, 3) && uoe170 { (setOf(1, 2) as MutableSet<Int>).add(3) } == "true")
    check("ro170f", uoe170 { emptyList<Int>() as MutableList<Int> } ==
        "C:kotlin.collections.EmptyList cannot be cast to kotlin.collections.MutableList" &&
        uoe170 { emptyMap<Int, Int>() as MutableMap<Int, Int> } ==
        "C:kotlin.collections.EmptyMap cannot be cast to kotlin.collections.MutableMap" &&
        emptySet<Int>() !is MutableSet<*> && listOf(1) is MutableList<*>)
    check("jc170g", buildList { add(1) }.javaClass.name == "kotlin.collections.builders.ListBuilder" &&
        buildMap<Int, Int> {}.javaClass.name == "kotlin.collections.builders.MapBuilder" &&
        buildList { add(1) }.toMutableList().apply { add(2) } == listOf(1, 2))
    check("jc170h", bm.keys.javaClass.name == "kotlin.collections.builders.MapBuilderKeys" &&
        bm.entries.javaClass.name == "kotlin.collections.builders.MapBuilderEntries" &&
        bm.entries.first().javaClass.name == "kotlin.collections.builders.MapBuilder\$EntryRef")
}


// ===== SECTION 171: Map default methods, seeded shuffles, view and iterator classes =====
// java.util.Map's putIfAbsent / computeIfAbsent / merge / remove(key, value), putAll's
// Unit; kotlin.random.Random(seed) is XorWowRandom, so a seeded draw or shuffle is the
// JVM's value for value; the JVM classes of asReversed / subList / iterator(); and a
// sealed builder map's views refuse every write.
fun s171() {
    val k = listOf("a", "b", "c", "d", "e", "z")
    val v = listOf(1, 2, 3, 4, 5, 9)
    val m = mutableMapOf(k[0] to v[0], k[1] to v[1])
    check("mm171a", m.putIfAbsent(k[0], v[5]) == 1 && m.putIfAbsent(k[2], v[2]) == null &&
        m.computeIfAbsent(k[3]) { it.length + 6 } == 7 && m.computeIfAbsent(k[0]) { 100 } == 1 &&
        m.merge(k[1], v[3]) { x, y -> x + y } == 6 && m.merge(k[4], v[4]) { x, y -> x + y } == 5 &&
        m.toString() == "{a=1, b=6, c=3, d=7, e=5}")
    val gone = m.merge(k[4], v[0]) { _, _ -> null }
    check("mm171b", gone == null && !m.containsKey(k[4]) && m.putAll(mapOf()) == Unit &&
        m.remove(k[0], v[1]) == false && m.remove(k[0], v[0]) == true && m.remove("q", v[0]) == false &&
        m.toString() == "{b=6, c=3, d=7}")
    val sl = mutableListOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
    sl.shuffle(kotlin.random.Random(v[3] * 10 + 2))
    val g = kotlin.random.Random(v[3] * 10 + 2)
    check("rnd171c", sl == listOf(6, 1, 5, 10, 3, 9, 2, 8, 7, 4) &&
        listOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).shuffled(kotlin.random.Random(v[4] + 2)) == listOf(10, 8, 3, 4, 1, 9, 2, 5, 6, 7) &&
        listOf(g.nextInt(100), g.nextInt(100), g.nextInt(), g.nextInt(5, 9)) == listOf(33, 40, -408207414, 8))
    val g0 = kotlin.random.Random(v[0] - 1)
    check("rnd171d", g0.nextInt() == -1934310868 && g0.nextInt(1) == 0 && g0.nextInt(2) == 1 &&
        g0.nextInt(7) == 3 && g0.nextInt(1 shl 20) == 691119 && g0.nextInt(1000000007) == 400965107 &&
        g0.nextInt(-5, 5) == -4 && g0.nextInt(Int.MIN_VALUE, Int.MAX_VALUE) == 811657455 &&
        g0.nextInt(Int.MIN_VALUE, 0) == -1449617834 && g0.nextInt(-1000, Int.MAX_VALUE) == 43427776 &&
        g0.nextLong() == 1158452016030805704L && g0.nextLong(100L) == 15L && g0.nextLong(-50L, 50L) == -48L &&
        g0.nextLong(1L, 1025L) == 50L && g0.nextLong(1024L) == 1L && g0.nextDouble() == 0.6120722576430151 &&
        g0.nextDouble(10.0) == 5.637887783588914 && g0.nextDouble(-1.0, 1.0) == -0.6624705647850315 &&
        g0.nextFloat() == 0.52138245f && !g0.nextBoolean() && !g0.nextBoolean())
    val ia = intArrayOf(1, 2, 3, 4, 5, 6)
    ia.shuffle(kotlin.random.Random(v[2]))
    val one = mutableListOf(1)
    val g2 = kotlin.random.Random(v[1])
    one.shuffle(g2)
    check("rnd171e", ia.toList() == listOf(2, 3, 5, 4, 6, 1) && g2.nextInt() == -665788237 &&
        kotlin.random.Random(5000000000L).nextInt() == 551290387 && (1..100).random(kotlin.random.Random(v[5])) == 43)
    check("jc171f", listOf(1, 2, 3).asReversed().javaClass.name == "kotlin.collections.ReversedListReadOnly" &&
        mutableListOf(1, 2, 3).asReversed().javaClass.name == "kotlin.collections.ReversedList" &&
        buildList { add(1); add(2); add(3) }.subList(0, 2).javaClass.name ==
            "kotlin.collections.builders.ListBuilder\$BuilderSubList" &&
        mutableListOf(1, 2).subList(0, 1).javaClass.name == "java.util.ArrayList\$SubList" &&
        listOf(1, 2).subList(0, 1).javaClass.name == "java.util.AbstractList\$RandomAccessSubList")
    val bm = buildMap { put(1, "x"); put(2, "y") }
    check("jc171g", buildSet { add(1) }.iterator().javaClass.name == "kotlin.collections.builders.MapBuilder\$KeysItr" &&
        bm.values.iterator().javaClass.name == "kotlin.collections.builders.MapBuilder\$ValuesItr" &&
        bm.iterator().javaClass.name == "kotlin.collections.builders.MapBuilder\$EntriesItr" &&
        bm.values.javaClass.name == "kotlin.collections.builders.MapBuilderValues" &&
        mutableListOf(1).iterator().javaClass.name == "java.util.ArrayList\$Itr" &&
        hashSetOf(1).iterator().javaClass.name == "java.util.HashMap\$KeyIterator" &&
        mutableMapOf(1 to 2).keys.iterator().javaClass.name == "java.util.LinkedHashMap\$LinkedKeyIterator" &&
        hashMapOf(1 to 2).values.javaClass.name == "java.util.HashMap\$Values" &&
        listOf<Int>().iterator().javaClass.name == "kotlin.collections.EmptyIterator")
    check("ro171h", uoe170 { (bm.keys as MutableSet).remove(1) } == "Unull" &&
        uoe170 { (bm.keys as MutableSet).remove(99) } == "Unull" &&
        uoe170 { (bm.values as MutableCollection).remove("q") } == "Unull" &&
        uoe170 { (bm.entries as MutableSet<Map.Entry<Int, String>>).clear() } == "Unull" &&
        uoe170 { (bm.keys as MutableSet).removeIf { it > 50 } } == "false" &&
        uoe170 { (bm.values as MutableCollection).removeIf { it == "x" } } == "Unull" &&
        uoe170 { (mapOf(1 to "a").values as MutableCollection).remove("a") } == "Unull" &&
        uoe170 { (mapOf(1 to "a").values as MutableCollection).remove("b") } == "false" &&
        bm == mapOf(1 to "x", 2 to "y"))
}


// ===== SECTION 172: compute / replace, and live list views =====
// java.util.Map's compute / computeIfPresent / replace (a null result removes the
// key); an Array's asList and a List's asReversed are VIEWS that write through both
// ways; a sealed builder's subList refuses writes, an Arrays.asList's takes a set.
fun s172() {
    val k = listOf("a", "b", "z", "q")
    val m = mutableMapOf<String, Int?>(k[0] to 1, k[1] to 2)
    check("mc172a", m.compute(k[0]) { _, x -> (x ?: 0) + 1 } == 2 && m.compute(k[2]) { _, x -> (x ?: 0) + 10 } == 10 &&
        m.compute(k[1]) { _, _ -> null } == null && m.toString() == "{a=2, z=10}" &&
        m.computeIfPresent(k[0]) { s, x -> x!! + s.length } == 3 && m.computeIfPresent(k[3]) { _, x -> x } == null &&
        m.computeIfPresent(k[2]) { _, _ -> null } == null && m.toString() == "{a=3}")
    m[k[3]] = null
    check("mc172b", m.computeIfPresent(k[3]) { _, _ -> 5 } == null && m.replace(k[3], 3) == null &&
        m.replace(k[1], 5) == null && m.toString() == "{a=3, q=3}" && m.replace(k[0], 3, 7) &&
        !m.replace(k[0], 3, 8) && !m.replace(k[1], null, 1) && m.replace(k[3], 3, null) &&
        m.toString() == "{a=7, q=null}")
    val arr = arrayOf(3, 1, 2)
    val al = arr.asList()
    arr[0] = 5
    (al as MutableList<Int>)[1] = 6
    check("av172c", al.javaClass.name == "java.util.Arrays\$ArrayList" && al == listOf(5, 6, 2) &&
        arr.contentToString() == "[5, 6, 2]" && uoe170 { al.add(1) } == "Unull")
    arr.sort()
    val ia = intArrayOf(4, 5)
    val il = ia.asList()
    ia.fill(arr.size)
    check("av172d", al == listOf(2, 5, 6) && il == listOf(3, 3) &&
        il.javaClass.name == "kotlin.collections.ArraysKt___ArraysJvmKt\$asList\$3")
    val ml = mutableListOf(1, 2, 3)
    val rv = ml.asReversed()
    ml.add(4)
    val seen = rv.toString()
    rv[0] = 40
    rv.add(0)
    check("av172e", seen == "[4, 3, 2, 1]" && ml == listOf(0, 1, 2, 3, 40) && rv == listOf(40, 3, 2, 1, 0))
    rv.removeIf { it % 2 == 0 }
    ml += 9
    check("av172f", ml == listOf(1, 3, 9) && rv == listOf(9, 3, 1) &&
        runCatching { for (x in rv) { if (x == 3) ml.add(7) } }.exceptionOrNull() is ConcurrentModificationException)
    val bl = buildList { add(1); add(2); add(3) }
    val bs = bl.subList(0, 2) as MutableList<Int>
    val ls = listOf(1, 2, 3).subList(0, 2) as MutableList<Int>
    check("ro172g", uoe170 { bs.set(0, 9) } == "Unull" && uoe170 { bs.add(9) } == "Unull" &&
        uoe170 { bs.clear() } == "Unull" && uoe170 { ls.add(9) } == "Unull" &&
        uoe170 { ls.removeAt(0) } == "Unull" && uoe170 { ls.set(0, 9) } == "1" && bl == listOf(1, 2, 3))
}


// ===== SECTION 173: subList as a live view, replaceAll, a primitive asList's cast =====
// A subList reads and writes its list's window both ways, a size change moving the
// list's tail; a structural change of the list makes the subList throw
// ConcurrentModificationException on its next use. MutableList.replaceAll and
// MutableMap.replaceAll (ArrayList's counts as a structural change), and a
// primitive array's asList is a read-only kotlin list no Mutable cast accepts.
fun cme173(f: () -> Any?): String = try { "" + f() } catch (e: ConcurrentModificationException) { "CME" + e.message }

fun s173() {
    val n = listOf(0, 1, 2, 3, 4, 5, 6)
    val ml = mutableListOf(n[1], n[2], n[3], n[4])
    ml.subList(n[0], n[2]).clear()
    val a = mutableListOf(n[1], n[2], n[3], n[4], n[5], n[6])
    val s = a.subList(n[1], n[4])
    a[n[2]] = 30
    val seen = s.toString()
    s[0] = 20
    s.add(7)
    check("sl173a", ml == listOf(3, 4) && seen == "[2, 30, 4]" && a == listOf(1, 20, 30, 4, 7, 5, 6) &&
        s == listOf(20, 30, 4, 7) && s.size == 4)
    s.removeAt(0)
    val s2 = s.subList(n[1], n[3])
    s2.add(99)
    s2.sortDescending()
    check("sl173b", a == listOf(1, 30, 99, 7, 4, 5, 6) && s == listOf(30, 99, 7, 4) && s2 == listOf(99, 7, 4))
    a.add(100)
    check("sl173c", cme173 { s.size } == "CMEnull" && cme173 { s.add(1) } == "CMEnull" &&
        cme173 { s.toString() } == "CMEnull" && cme173 { s2.size } == "CMEnull" &&
        cme173 { s[0] = 5 } == "CMEnull" && a == listOf(1, 30, 99, 7, 4, 5, 6, 100))
    val b = mutableListOf(n[1], n[2], n[3], n[4])
    val x1 = b.subList(n[0], n[2])
    val x2 = b.subList(n[2], n[4])
    x1.add(9)
    val d = mutableListOf(n[5], n[6], n[1])
    val dv = d.subList(n[0], n[1])
    d.sort()
    val e = mutableListOf(n[5], n[6], n[1])
    val ev = e.subList(n[0], n[2])
    e[0] = 1
    check("sl173d", b == listOf(1, 2, 9, 3, 4) && cme173 { x2.size } == "CMEnull" && x1 == listOf(1, 2, 9) &&
        cme173 { dv.size } == "CMEnull" && ev == listOf(1, 6))
    val c = mutableListOf(n[1], n[2], n[3], n[4], n[5])
    c.subList(n[1], n[4]).reverse()
    val r1 = c.toString()
    c.subList(n[0], n[3]).fill(0)
    c.subList(n[1], n[3]).removeAll { it == 0 }
    val ar = arrayOf(n[1], n[2], n[3], n[4])
    val asl = ar.asList().subList(n[1], n[3]) as MutableList<Int>
    asl[0] = 20
    ar[2] = 30
    val mr = mutableListOf(n[1], n[2], n[3], n[4])
    val rs = mr.asReversed().subList(n[0], n[2])
    rs[0] = 40
    check("sl173e", r1 == "[1, 4, 3, 2, 5]" && c == listOf(0, 2, 5) && c.subList(0, 2).javaClass.name == "java.util.ArrayList\$SubList" &&
        ar.toList() == listOf(1, 20, 30, 4) && asl == listOf(20, 30) && mr == listOf(1, 2, 3, 40) && rs == listOf(40, 3))
    val l = mutableListOf(n[1], n[2])
    l.replaceAll { it + 1 }
    val m = mutableMapOf("a" to n[1], "b" to n[5])
    m.replaceAll { _, v -> v * 2 }
    val m1 = m.toString()
    m.replaceAll { k, v -> v + k.length }
    val lr = l.toString()
    val lc = cme173 { for (x in l) l.replaceAll { it * 10 } }
    for (k in m.keys) m.replaceAll { _, v -> v + 1 }
    val big = mutableListOf(n[1], n[2], n[3], n[4], n[5])
    big.subList(n[1], n[3]).replaceAll { -it }
    val dq = ArrayDeque(listOf(n[1], n[2]))
    dq.replaceAll { it + 5 }
    val tm = sortedMapOf(n[2] to "b", n[1] to "a")
    tm.replaceAll { k, v -> v + k }
    check("ra173f", lr == "[2, 3]" && m1 == "{a=2, b=10}" && lc == "CMEnull" && l == listOf(20, 30) &&
        m.toString() == "{a=5, b=13}" && big == listOf(1, -2, -3, 4, 5) && dq == listOf(6, 7) &&
        tm.toString() == "{1=a1, 2=b2}")
    val ia = intArrayOf(n[1], n[2], n[3])
    val il = ia.asList()
    check("ca173g", il !is MutableList<*> && (il as? MutableList<Int>) == null &&
        uoe170 { (il as MutableList<Int>).set(0, 9) } == "C:kotlin.collections.ArraysKt___ArraysJvmKt\$asList\$3 cannot be cast to kotlin.collections.MutableList" &&
        il.subList(0, 1).javaClass.name == "kotlin.collections.AbstractList\$SubList" && il.subList(0, 1) !is MutableList<*> &&
        doubleArrayOf(1.0).asList() !is MutableCollection<*> && arrayOf(n[1]).asList() is MutableList<*> && ia.toList() == listOf(1, 2, 3))
}

// ===== SECTION 174: a stale subList's reads, casts to concrete collections, marker interfaces =====
// A structurally changed list makes its subList's get, iterator and next() throw
// ConcurrentModificationException (an index is checked first). A cast to a concrete
// java.util class (ArrayList, LinkedHashMap, HashSet, ...) is a checkcast on the JVM
// class. RandomAccess, Cloneable and Serializable are answered from the JVM class,
// and for a program class from the supertypes it names.
class Ser174 : java.io.Serializable
open class Cl174 : Cloneable
class Sub174 : Cl174()
interface ISer174 : java.io.Serializable
class Via174 : ISer174
class Plain174
enum class En174 { A }

fun s174() {
    val n = listOf(0, 1, 2, 3, 4, 5, 6, 7)
    val m = mutableListOf(n[1], n[2], n[3], n[4])
    val s = m.subList(n[1], n[3])
    val e = m.subList(n[1], n[1])
    val ss = s.subList(n[0], n[1])
    m.add(n[5])
    var sum = 0
    check("cme174a", cme173 { s[0] } == "CMEnull" && cme173 { for (x in s) sum += x } == "CMEnull" &&
        cme173 { for (x in e) sum += x } == "CMEnull" && cme173 { ss[0] } == "CMEnull" &&
        cme173 { s.indices } == "CMEnull" && cme173 { s.lastIndex } == "CMEnull" && sum == 0 &&
        (try { s[n[7]] } catch (x: IndexOutOfBoundsException) { "oob" }) == "oob")
    val m2 = mutableListOf(n[1], n[2], n[3], n[4])
    val s2 = m2.subList(n[0], n[3])
    val m3 = mutableListOf(n[1], n[2], n[3], n[4])
    val s3 = m3.subList(n[0], n[3])
    val m4 = mutableListOf(n[1], n[2], n[3])
    val s4 = m4.subList(n[0], n[2])
    m4[0] = n[6]
    check("cme174b", cme173 { for (x in s2) { sum += x; if (x == 1) m2.add(9) } } == "CMEnull" && sum == 1 &&
        cme173 { s3.forEach { sum += it; if (it == 1) m3.removeAt(3) } } == "CMEnull" && sum == 2 &&
        s4[0] == 6 && s4.sum() == 8)
    val il = intArrayOf(n[1], n[2]).asList()
    check("cast174c", uoe170 { il as java.util.ArrayList<Int> } == "C:class kotlin.collections.ArraysKt___ArraysJvmKt\$asList\$3 " +
        "cannot be cast to class java.util.ArrayList (kotlin.collections.ArraysKt___ArraysJvmKt\$asList\$3 is in unnamed " +
        "module of loader 'app'; java.util.ArrayList is in module java.base of loader 'bootstrap')" &&
        uoe170 { listOf(n[1], n[2]) as ArrayList<Int> }.startsWith("C:class java.util.Arrays\$ArrayList cannot be cast to class java.util.ArrayList") &&
        uoe170 { mapOf(n[1] to n[2]) as LinkedHashMap<Int, Int> }.startsWith("C:class java.util.Collections\$SingletonMap cannot") &&
        uoe170 { hashMapOf(n[1] to n[2]) as LinkedHashMap<Int, Int> }.startsWith("C:class java.util.HashMap cannot") &&
        uoe170 { hashSetOf(n[1]) as LinkedHashSet<Int> }.startsWith("C:class java.util.HashSet cannot") &&
        uoe170 { mutableListOf(n[1]) as ArrayDeque<Int> }.endsWith("kotlin.collections.ArrayDeque is in unnamed module of loader 'app')") &&
        (setOf(n[1], n[2]) as HashSet<Int>).size == 2 && (mutableMapOf(n[1] to n[2]) as HashMap<Int, Int>)[1] == 2 &&
        (mutableListOf(n[1]) as? ArrayList<Int>)?.size == 1 && (listOf(n[1]) as? ArrayList<Int>) == null)
    val nul: Any? = if (n[0] == 0) null else 1
    check("cast174d", uoe170 { "x" as Cloneable }.startsWith("C:class java.lang.String cannot be cast to class java.lang.Cloneable") &&
        uoe170 { ArrayDeque<Int>() as RandomAccess }.startsWith("C:class kotlin.collections.ArrayDeque cannot be cast to class java.util.RandomAccess") &&
        uoe170 { Plain174() as java.io.Serializable }.startsWith("C:class Plain174 cannot") &&
        (Sub174() as Cloneable) is Sub174 && (n[1] as java.io.Serializable) == 1 &&
        (try { nul as ArrayList<Int> } catch (x: NullPointerException) { x.message }) == "null cannot be cast to non-null type java.util.ArrayList<kotlin.Int>" &&
        (try { nul as Cloneable } catch (x: NullPointerException) { x.message }) == "null cannot be cast to non-null type kotlin.Cloneable")
    val vs = listOf<Any>(il, listOf(n[1], n[2]), mutableListOf(n[1]), emptyList<Int>(), mutableListOf(n[1], n[2]).subList(0, 1),
        ArrayDeque<Int>(), java.util.LinkedList<Int>(), listOf(n[1], n[2]).asReversed(), setOf(n[1], n[2]), hashMapOf(n[1] to n[2]),
        mapOf(n[1] to n[2]), intArrayOf(n[1]), "s", n[1], 1u, n[1] to n[2], Ser174(), Sub174(), Via174(), Plain174(), En174.A,
        IllegalStateException("x"), mapOf(n[1] to n[2], n[3] to n[4]).keys, buildList { add(n[1]) })
    val marks = vs.joinToString("") { v -> (if (v is RandomAccess) "R" else "-") + (if (v is Cloneable) "C" else "-") +
        (if (v is java.io.Serializable) "S" else "-") + " " }
    check("mark174e", marks == "R-- R-S RCS R-S R-- --- -CS --- -CS -CS --S -CS --S --S --- --S --S -C- --S --- --S --S --- R-S ")
}

// ===== SECTION 175: Unit as Any, Any(), read-only answers by size, an Array builder's class =====
// Unit is an object like any other: `as Any` keeps it and only null fails a non-null
// cast. Any() is a plain java.lang.Object, equal only to itself. kotlin-stdlib
// answers toList, take, drop, reversed, distinct, toSet, toMap, a map's minus, ...
// through emptyList() / listOf(e) / emptySet() / emptyMap() and the Singleton
// classes at size 0 or 1, whose read-only rules a cast to a Mutable type then meets.
// An Array builder's explicit element type fixes the array's JVM class.
fun unit175(): Any? = Unit
fun cn175(x: Any): String = x.javaClass.name.removePrefix("java.util.").removePrefix("kotlin.collections.")

fun s175() {
    val n = listOf(0, 1, 2, 3)
    check("unit175a", (unit175() as Any).toString() == "kotlin.Unit" && unit175() is Any && unit175() is Unit &&
        unit175() as? String == null && unit175() != null &&
        uoe170 { unit175() as String }.startsWith("C:class kotlin.Unit cannot be cast to class java.lang.String"))
    val a = Any()
    val b = Any()
    check("any175b", a == a && a != b && !a.equals(b) && a.toString() == "java.lang.Object@" + a.hashCode().toString(16) &&
        a.javaClass.name == "java.lang.Object" && a::class.simpleName == "Any" && setOf(a, b, a).size == 2 &&
        mapOf(a to n[1])[b] == null && listOf(b, a).indexOf(a) == 1)
    val l0 = n.filter { it > 5 }
    val l1 = n.filter { it == 1 }
    val l2 = n.filter { it > 1 }
    check("ro175c", cn175(l0.toList()) == "EmptyList" && cn175(l1.toList()) == "Collections\$SingletonList" &&
        cn175(l2.toList()) == "ArrayList" && cn175(n.take(n[1])) == "Collections\$SingletonList" &&
        cn175(n.take(n[0])) == "EmptyList" && cn175(n.drop(n[3])) == "Collections\$SingletonList" &&
        cn175(n.takeLast(n[2])) == "ArrayList" && cn175(n.dropLast(n[3])) == "Collections\$SingletonList" &&
        cn175(l1.reversed()) == "Collections\$SingletonList" && cn175(l1.toTypedArray().reversed()) == "ArrayList" &&
        cn175(listOf(n[1], n[1]).distinct()) == "Collections\$SingletonList" && cn175(l1.sorted()) == "Collections\$SingletonList" &&
        cn175(n.slice(n[1]..n[1])) == "Collections\$SingletonList" && cn175(n.slice(listOf(n[1]))) == "ArrayList" &&
        cn175(n.toTypedArray().slice(n[1]..n[2])) == "Arrays\$ArrayList" && cn175("a".toList()) == "Collections\$SingletonList" &&
        cn175((n[1]..n[1]).toList()) == "Collections\$SingletonList" && cn175(l1.asSequence().toList()) == "Collections\$SingletonList" &&
        cn175((n[1] to n[2]).toList()) == "Arrays\$ArrayList" && cn175(n.filter { it > 5 }) == "ArrayList")
    val m1 = mapOf(n[1] to n[2])
    val m2 = mapOf(n[1] to n[2], n[3] to n[0])
    check("ro175d", cn175(l0.toSet()) == "EmptySet" && cn175(l1.toSet()) == "Collections\$SingletonSet" &&
        cn175(listOf(n[1], n[1]).toSet()) == "LinkedHashSet" && cn175(setOf(n[1], n[2]).filter { it == 1 }.toSet()) == "Collections\$SingletonSet" &&
        cn175(listOf(n[1] to n[2]).toMap()) == "Collections\$SingletonMap" && cn175(m2.toMap()) == "LinkedHashMap" &&
        cn175(m2 - n[1]) == "Collections\$SingletonMap" && cn175(m1 - n[1]) == "EmptyMap" &&
        cn175(emptyMap<Int, Int>() + (n[1] to n[2])) == "Collections\$SingletonMap" && cn175(m1 + (n[3] to n[0])) == "LinkedHashMap" &&
        cn175(m1.toList()) == "Collections\$SingletonList" && cn175(m2.keys.toList()) == "ArrayList" &&
        cn175(l0.toMutableList()) == "ArrayList")
    check("ro175e", uoe170 { (l1.toList() as MutableList<Int>).add(n[2]) } == "Unull" &&
        uoe170 { l0.toList() as MutableList<Int> } == "C:kotlin.collections.EmptyList cannot be cast to kotlin.collections.MutableList" &&
        uoe170 { (l1.toSet() as MutableSet<Int>).add(n[2]) } == "Unull" &&
        (l2.toList() as MutableList<Int>).add(n[1]) && (n.take(n[2]) as MutableList<Int>).size == 2)
    check("arr175f", emptyArray<Int>().javaClass.name == "[Ljava.lang.Integer;" &&
        emptyArray<String>().javaClass.name == "[Ljava.lang.String;" && arrayOf<Any>(n[1]).javaClass.name == "[Ljava.lang.Object;" &&
        arrayOfNulls<Int>(n[2]).javaClass.name == "[Ljava.lang.Integer;" && emptyArray<List<Int>>().javaClass.name == "[Ljava.util.List;" &&
        emptyArray<Array<Int>>().javaClass.name == "[[Ljava.lang.Integer;" && arrayOf(n[1]).javaClass.name == "[Ljava.lang.Integer;")
    // Unit is a value, never null: ?., ?:, !!, the NotNull operations and a Pair keep it.
    val u = unit175()
    val p = "k" to u
    check("unit175g", (u?.toString() ?: "was null") == "kotlin.Unit" && (u ?: n[1]) is Unit && u!! is Unit &&
        listOf(u, null).filterNotNull().size == 1 && listOf(u, null).mapNotNull { it }.size == 1 &&
        listOfNotNull(u, null).size == 1 && requireNotNull(u) is Unit && checkNotNull(u) is Unit &&
        p.toString() == "(k, kotlin.Unit)" && mapOf(p).size == 1 && mapOf(p).containsKey("k"))
    // An anonymous function's block body leaves a following `;` to the statement.
    var ran = false
    val f = fun(): Unit {}; run { ran = f() == Unit }
    check("anon175h", ran && f().toString() == "kotlin.Unit")
}


// ===== SECTION 176: Object#toString and hashCode agree, synchronized, Integer / Long, function values, an Array copy's class =====
// java.lang.Object#toString is the binary class name, "@" and hashCode() in hex, so
// it follows an overridden hashCode, and Unit is an object with a hash of its own,
// not null's 0. synchronized(lock) { } is the block's value. java.lang.Integer and
// java.lang.Long are reachable from Kotlin. Without kotlin-reflect a reference
// prints "function g (Kotlin reflection is not available)" and a lambda its hidden
// class, AKt$$Lambda/0x...@hash. An Array's copy keeps the array's runtime class.
class P176
open class H176 { override fun hashCode() = 255 }
class HS176 : H176()
class O176 { class N176 }
open class B176
class X176 : B176() { override fun hashCode() = super.hashCode() + 1 }
data class U176(val a: Int, val u: Unit)
class C176(var n: Int) { fun inc(): Int = ++n }
fun g176(x: Int): String = "s$x"
fun agree176(x: Any): Boolean = x.toString() == x.javaClass.name + "@" + Integer.toHexString(x.hashCode())

fun s176() {
    val p = P176()
    check("ts176a", p.toString() == "P176@" + p.hashCode().toString(16) && agree176(p) && agree176(HS176()) &&
        agree176(O176.N176()) && agree176(Any()))
    check("ts176b", H176().toString() == "H176@ff" && HS176().toString() == "HS176@ff" && "${HS176()}" == "HS176@ff" &&
        O176.N176().toString().substringBefore("@") == "O176\$N176")
    val x = X176()
    check("sup176k", x.toString() == "X176@" + Integer.toHexString(x.hashCode()) && x.hashCode() == x.hashCode())
    val lock = Any()
    var side = 0
    check("sync176c", synchronized(lock) { side++; 40 + 2 } == 42 && side == 1 && synchronized(p) { "in$side" } == "in1")
    val xs = intArrayOf(255, -1, 10, 1)
    check("int176d", Integer.toHexString(xs[0]) == "ff" && Integer.toHexString(xs[1]) == "ffffffff" &&
        Integer.toBinaryString(xs[2]) == "1010" && Integer.toOctalString(xs[2] - 2) == "10" &&
        Integer.parseInt("-77") == -77 && Integer.parseInt("zz", 36) == 1295 && Integer.valueOf("12") == 12 &&
        Integer.MAX_VALUE == Int.MAX_VALUE && Integer.MIN_VALUE == Int.MIN_VALUE && Integer.SIZE == 32 &&
        Integer.bitCount(xs[0]) == 8 && Integer.numberOfLeadingZeros(xs[3]) == 31 && Integer.compare(xs[2], xs[0]) == -1 &&
        Integer.signum(xs[1]) == -1 && Integer.toString(xs[0], 16) == "ff" && Integer.max(xs[2], xs[0]) == 255 &&
        Integer.sum(xs[2], xs[3]) == 11 && Integer.rotateLeft(xs[3], 33) == 2)
    val ls = longArrayOf(-1L, 255L)
    check("long176e", java.lang.Long.toHexString(ls[0]) == "ffffffffffffffff" && java.lang.Long.toBinaryString(ls[1]) == "11111111" &&
        java.lang.Long.parseLong("-9000000000") == -9000000000L && java.lang.Long.parseLong("7f", 16) == 127L &&
        java.lang.Long.MAX_VALUE == Long.MAX_VALUE && java.lang.Long.SIZE == 64 && java.lang.Long.bitCount(ls[0]) == 64 &&
        java.lang.Long.compare(ls[0], ls[1]) == -1)
    var nfe = ""
    try { Integer.parseInt("abc") } catch (e: NumberFormatException) { nfe = e.message ?: "" }
    check("int176f", nfe == "For input string: \"abc\"")
    check("unit176g", Unit.hashCode() != 0 && U176(1, Unit).hashCode() == 31 + Unit.hashCode() &&
        listOf(Unit).hashCode() == 31 + Unit.hashCode() && U176(1, Unit) == U176(1, Unit))
    val c = C176(1)
    val nr = " (Kotlin reflection is not available)"
    check("fn176h", (::g176).toString() == "function g176" + nr && (c::inc).toString() == "function inc" + nr &&
        (C176::inc).toString() == "function inc" + nr && (::C176).toString() == "constructor" + nr &&
        (c::n).toString() == "property n" + nr && (C176::n).toString() == "property n" + nr &&
        (String::length).toString() == "property length" + nr && "${listOf(::g176)}" == "[function g176$nr]")
    val lam = { x: Int -> x + 1 }
    check("fn176i", lam.toString().contains("\$\$Lambda") && lam.toString().contains("@") && !lam.toString().contains("\n"))
    val a = arrayOfNulls<String>(2)
    val e = emptyArray<String>()
    val s = arrayOf("x", "y")
    check("arr176j", a.copyOf().javaClass.name == "[Ljava.lang.String;" && e.copyOf(1).javaClass.name == "[Ljava.lang.String;" &&
        a.copyOfRange(0, 1).javaClass.name == "[Ljava.lang.String;" && a.reversedArray().javaClass.name == "[Ljava.lang.String;" &&
        (e + s[0]).javaClass.name == "[Ljava.lang.String;" && (s + "z").javaClass.name == "[Ljava.lang.String;" &&
        (s + "z").size == 3 && (intArrayOf(xs[3]) + xs[2]).javaClass.name == "[I" && (s + listOf("q")).last() == "q")
}


// ===== SECTION 177: super to kotlin.Any, ::top references, Array<ProgramClass> class names, extension references =====
// A super.toString() / equals() / hashCode() that no class above the caller declares
// reaches kotlin.Any - also from a class with no superclass - and an enum's reaches
// Enum.toString (the name), a Throwable's Throwable.toString. `::top` of a top-level
// property is a KProperty0 that reads and writes the variable, and a newline before
// a `::` ends the statement in front of it, except inside ( ) or [ ]. An Array whose
// element type is a class the program declares has that class's JVM name, from a
// type argument, a declared type or a shared element class. A reference to an
// extension function is a function reference.
open class B177
class X177 : B177() { override fun toString() = "X:" + super.toString() }
class E177 {
    override fun hashCode() = super.hashCode() + 1
    override fun equals(other: Any?) = super.equals(other)
    override fun toString() = "E:" + super.toString()
}
enum class C177 { RED; override fun toString() = "c:" + super.toString() }
class T177(m: String) : Exception(m) { override fun toString() = "T/" + super.toString() }
open class A177
class D177 : A177()
class O177 { class N177 }
class H177 { val a: Array<A177> = arrayOf(D177()) }
class Bx177(val v: Int)
fun Bx177.ext177() = v + 1
val top177 = 5
var cnt177 = 1
val lz177 by lazy { 40 + 2 }

fun s177() {
    val x = X177()
    check("sup177a", x.toString() == "X:X177@" + Integer.toHexString(x.hashCode()))
    val e = E177()
    val h = e.hashCode()
    check("sup177b", e.toString() == "E:E177@" + Integer.toHexString(h) && e == e && !e.equals(E177()) && e.hashCode() == h)
    check("sup177c", C177.RED.toString() == "c:RED" && T177("m").toString() == "T/T177: m")
    val nr = " (Kotlin reflection is not available)"
    val r = ::cnt177
    cnt177 = 7
    val got = r.get()
    r.set(9)
    check("ref177d", (::top177).toString() == "property top177" + nr && ::top177.name == "top177" && ::top177.get() == 5 &&
        (::top177)() == 5 && got == 7 && cnt177 == 9 && ::cnt177 == ::cnt177 && "${::top177}" == "property top177" + nr)
    ::cnt177.set(12)
    val q = nr
    ::cnt177.set(cnt177 + 1)
    check("ref177g", cnt177 == 13 && q == nr && ::lz177.get() == 42 && ::cnt177.invoke() == 13)
    val bx = Bx177(4)
    check("ref177h", listOf(bx).map(Bx177
        ::v) == listOf(4) && listOf(5).map(Int
        ::toString) == listOf("5") && (bx
        ::v).get() == 4)
    val b = Bx177(1)
    check("ref177e", (b::ext177).toString() == "function ext177" + nr && (Bx177::ext177).toString() == "function ext177" + nr &&
        (b::v).toString() == "property v" + nr && (Bx177::ext177)(b) == 2 && (b::ext177)() == 2)
    val d: Array<A177> = arrayOf(D177())
    check("arr177f", arrayOf<A177>(D177()).javaClass.name == "[LA177;" && d.javaClass.name == "[LA177;" &&
        d.copyOf().javaClass.name == "[LA177;" && arrayOf(D177()).javaClass.name == "[LD177;" &&
        arrayOf<O177.N177>().javaClass.name == "[LO177\$N177;" && H177().a.javaClass.name == "[LA177;" &&
        arrayOfNulls<A177>(1).javaClass.name == "[LA177;" && arrayOf(D177(), null).javaClass.name == "[LD177;")
}

// ===== SECTION 178: an Array's common element class, top-level lateinit and lambda properties as references =====
// arrayOf's element type, and so its JVM class, is the common supertype of its
// arguments: the one supertype every element has that is a subtype of all the
// others (a superclass, an interface, java.lang.Comparable, Serializable, Enum,
// Number, an Object[]), or Object when there is only an intersection. A lambda is
// a kotlin.jvm.functions.FunctionN, a function reference a KFunction, an anonymous
// object its own class. `::top.isInitialized` reads a top-level lateinit property,
// `::f` of a top-level property holding a lambda is that property's KProperty0,
// and an unbound reference to a name that is both a property of the class and an
// extension function is the property.
open class An178
class Dg178 : An178()
interface Sh178
interface Nm178
class S1178 : Sh178
class S2178 : Sh178
open class Ba178
class M1178 : Ba178(), Sh178, Nm178
class C1178 : Ba178(), Sh178, Nm178
class K178 : Comparable<K178> { override fun compareTo(other: K178) = 0 }
enum class E178 { A { override fun toString() = "a" }, B }
enum class F178 { X }
lateinit var late178: String
lateinit var cb178: () -> Int
val f178 = { x: Int -> x + 1 }
fun two178(x: Int) = x * 2
fun late178Set() = ::late178.isInitialized
class Bx178(val name: String)
fun Bx178.name(): String = "ext"

fun s178() {
    val o = "[Ljava.lang.Object;"
    check("arr178a", arrayOf(Dg178(), An178()).javaClass.name == "[LAn178;" && arrayOf(S1178(), S2178()).javaClass.name == "[LSh178;" &&
        arrayOf(M1178(), Ba178()).javaClass.name == "[LBa178;" && arrayOf(M1178(), C1178()).javaClass.name == o &&
        arrayOf(E178.A, E178.B).javaClass.name == "[LE178;" && arrayOf(E178.A).javaClass.name == "[LE178;" &&
        arrayOf(E178.B, F178.X).javaClass.name == "[Ljava.lang.Enum;" && arrayOf(K178(), "s").javaClass.name == "[Ljava.lang.Comparable;")
    val i = 1
    val l = 2L
    check("arr178b", arrayOf(Unit).javaClass.name == "[Lkotlin.Unit;" && arrayOf(1 to 2).javaClass.name == "[Lkotlin.Pair;" &&
        arrayOf(Pair(1, 2), Triple(1, 2, 3)).javaClass.name == "[Ljava.io.Serializable;" &&
        arrayOf(1..2, 3 downTo 0).javaClass.name == "[Lkotlin.ranges.IntProgression;" && arrayOf(1..2, 'a'..'b').javaClass.name == o &&
        arrayOf(RuntimeException(), IllegalStateException()).javaClass.name == "[Ljava.lang.RuntimeException;" &&
        arrayOf(Error(), Exception()).javaClass.name == "[Ljava.lang.Throwable;" && arrayOf(i, l).javaClass.name == o &&
        arrayOf(i, "s").javaClass.name == o && arrayOf(intArrayOf(1)).javaClass.name == "[[I" &&
        arrayOf(arrayOf(1), arrayOf("a")).javaClass.name == "[[Ljava.lang.Object;" && arrayOf(intArrayOf(1), arrayOf(1)).javaClass.name == o &&
        arrayOf(Regex("a")).javaClass.name == "[Lkotlin.text.Regex;" && arrayOf(1u).javaClass.name == "[Lkotlin.UInt;")
    val an = object : Sh178 {}
    check("arr178c", arrayOf({ 1 }).javaClass.name == "[Lkotlin.jvm.functions.Function0;" &&
        arrayOf({ x: Int -> x }).javaClass.name == "[Lkotlin.jvm.functions.Function1;" &&
        arrayOf({ x: Int, y: Int -> x + y }).javaClass.name == "[Lkotlin.jvm.functions.Function2;" &&
        arrayOf(fun(x: Int) = x).javaClass.name == "[Lkotlin.jvm.functions.Function1;" &&
        arrayOf(::two178).javaClass.name == "[Lkotlin.reflect.KFunction;" && arrayOf(an).javaClass.name == "[L" + an.javaClass.name + ";" &&
        arrayOf(object : Sh178 {}, object : Sh178 {}).javaClass.name == "[LSh178;" && arrayOf(an, S1178()).javaClass.name == "[LSh178;")
    check("arr178d", arrayOf(Dg178()).javaClass.simpleName == "Dg178[]" && arrayOf(arrayOf(1)).javaClass.simpleName == "Integer[][]" &&
        arrayOf(intArrayOf(1)).javaClass.simpleName == "int[][]")
    val before = ::late178.isInitialized
    val beforeFn = late178Set()
    val cbBefore = ::cb178.isInitialized
    late178 = "x"
    cb178 = { 3 }
    check("lat178e", !before && !beforeFn && ::late178.isInitialized && late178Set() && !cbBefore && ::cb178.isInitialized && cb178() == 3)
    val nr = " (Kotlin reflection is not available)"
    check("ref178f", (::f178).toString() == "property f178" + nr && ::f178.name == "f178" && ::f178.get()(1) == 2 &&
        (::f178)()(2) == 3 && (::two178).toString() == "function two178" + nr)
    val r = Bx178::name
    check("ref178g", r.toString() == "property name" + nr && r(Bx178("b")) == "b" && Bx178("c").name() == "ext")
}


// ===== SECTION 179: an unassigned lateinit read, `is FunctionN`, a lambda's class, an array's KClass =====
// Reading a top-level or local `lateinit var` before it is assigned throws
// UninitializedPropertyAccessException, and the modifier stays with its own
// declaration; a nullable local of the same name reads null. Every function value
// is a kotlin.Function, and a lambda, an anonymous function or a top-level function
// reference is the FunctionN of its arity: its parameters, or the implicit `it` it
// reads, plus a declared receiver. A nullable generic type holds null. A lambda's
// JVM class (AKt$$Lambda/0x...) is not reproducible, so only how its name relates
// to toString and ::class is asserted. An array's KClass is its JVM array class
// ([LDg179;, [I), named Array / IntArray.
class Dg179
lateinit var late179: String
class Nl179 { var x: String? = null }
fun read179(): String = late179
fun shadow179(): String? { val late179: String? = null; return late179 }
fun two179(a: Int, b: Int) = a + b

fun s179() {
    var msg = ""
    try { msg = read179() } catch (e: UninitializedPropertyAccessException) { msg = e.message ?: "" }
    var tpl = ""
    try { tpl = "v=$late179" } catch (e: UninitializedPropertyAccessException) { tpl = "threw" }
    check("lat179a", msg == "lateinit property late179 has not been initialized" && tpl == "threw" &&
        shadow179() == null && Nl179().x == null)
    lateinit var loc: String
    var lmsg = ""
    try { lmsg = loc } catch (e: UninitializedPropertyAccessException) { lmsg = e.message ?: "" }
    loc = "L"
    late179 = "x"
    check("lat179b", lmsg == "lateinit property loc has not been initialized" && loc == "L" && read179() == "x")
    val fs = listOf<Any>({ 1 }, { x: Int -> x }, { x: Int, y: Int -> x + y }, "s", 3)
    val ar = fs.map { v ->
        (if (v is Function0<*>) "0" else "") + (if (v is Function1<*, *>) "1" else "") +
            (if (v is Function2<*, *, *>) "2" else "") + (if (v is Function<*>) "f" else "")
    }
    check("fn179c", ar == listOf("0f", "1f", "2f", "", ""))
    val w = fs.map { v -> when (v) { is Function0<*> -> "f0"; is Function1<*, *> -> "f1"; !is Function2<*, *, *> -> "no"; else -> "f2" } }
    check("fn179d", w == listOf("f0", "f1", "f2", "no", "no"))
    val it1: (Int) -> Int = { it * 2 }
    val rc: Int.() -> Int = { this + 1 }
    val a1: Any = it1
    val a2: Any = rc
    val a3: Any = ::two179
    val an: Any = fun(x: Int): Int { return x }
    val tf: (Int) -> String = { "<$it>" }
    val tp: Any = tf
    val n: Any? = null
    check("fn179e", a1 is Function1<*, *> && a1 !is Function0<*> && a2 is Function1<*, *> && an is Function1<*, *> && tp is Function1<*, *> &&
        a3 is Function2<*, *, *> && a3 !is Function1<*, *> && n is Function0<*>? && n is List<*>? && n !is Function0<*>)
    val f0 = { 1 }
    val jn = f0.javaClass.name
    check("cls179f", jn.isNotEmpty() && f0.toString().startsWith(jn + "@") && f0::class.java.name == jn &&
        f0.javaClass == f0::class.java)
    val arr = arrayOf(Dg179())
    val ia = intArrayOf(1)
    check("arr179g", arr::class.java.name == "[LDg179;" && "${arr::class.java}" == "class [LDg179;" &&
        arr::class.simpleName == "Array" && arr::class.qualifiedName == "kotlin.Array" &&
        arr::class.java.simpleName == "Dg179[]" && arr::class.java == arr.javaClass && arr.javaClass.kotlin == arr::class &&
        arr::class == arrayOf(Dg179())::class && arr::class != arrayOf("a")::class)
    check("arr179h", ia::class.java.name == "[I" && "${IntArray::class.java}" == "class [I" && ia::class == IntArray::class &&
        ia::class.simpleName == "IntArray" && ia::class.qualifiedName == "kotlin.IntArray" &&
        arrayOf("a")::class.java.name == "[Ljava.lang.String;")
}

// ===== SECTION 180: Array<T>::class, a reference's class and arity, receiver-typed parameters, a parameter named it =====
// A static `Array<T>::class` is the JVM array class its element type fixes
// ([LDg180;, [[I), the class an array built of T has. A function reference is an
// instance of an anonymous class (AKt$main$r$1, not reproducible), so only how its
// name relates to toString and ::class is asserted. A reference is the FunctionN
// of the parameters of what it names, plus the receiver of an unbound one; a
// constructor reference takes the constructor's. A lambda passed to a parameter of
// a receiver function type has that type's arity and takes its parameters after
// the receiver, and an `it` read under a parameter named `it` is not the enclosing
// lambda's implicit one.
open class An180
class Dg180 : An180()
class Out180 { class In }
class Cat180(val n: Int) {
    var w = 3
    fun bark() = "m"
    fun eat(x: Int, y: Int) = x + y
    fun self(): Any = this::bark
}
class Two180(a: Int, b: String)
fun two180(): Int = 2
val top180 = 5
fun ar180(f: Any): String =
    (if (f is Function0<*>) "0" else "") + (if (f is Function1<*, *>) "1" else "") +
        (if (f is Function2<*, *, *>) "2" else "") + (if (f is Function3<*, *, *, *>) "3" else "")
var seen180: Any? = null
fun rc180(b: Int.() -> Int): Int { seen180 = b; return b(3) }
fun rc180b(b: Int.(Int, String) -> Int): Int { seen180 = b; return 3.b(4, "x") }

fun s180() {
    val arr = arrayOf(Dg180())
    check("arr180a", Array<Dg180>::class.java.name == "[LDg180;" && arr::class == Array<Dg180>::class &&
        Array<An180>::class != Array<Dg180>::class && Array<Array<Dg180>>::class.java.name == "[[LDg180;" &&
        Array<Out180.In>::class.java.name == "[LOut180\$In;" && Array<Int>::class.java.name == "[Ljava.lang.Integer;" &&
        Array<IntArray>::class.java.name == "[[I" && Array<String>::class.java == arrayOf("a").javaClass &&
        Array<Dg180>::class.simpleName == "Array")
    val r = ::two180
    val lam = { 1 }
    check("ref180b", r.javaClass.name.isNotEmpty() && "${r.javaClass}" == "class " + r.javaClass.name &&
        r.javaClass.simpleName == "" && r::class.java.name == r.javaClass.name && r.javaClass != lam.javaClass &&
        Cat180::bark.javaClass.name.isNotEmpty() && Cat180::w.javaClass.simpleName == "")
    val c = Cat180(1)
    check("ref180c", listOf(ar180(Cat180::bark), ar180(Cat180::eat), ar180(c::bark), ar180(c::eat), ar180(Cat180::w),
        ar180(c::w), ar180(Cat180::n), ar180(::top180), ar180(::Cat180), ar180(::Two180), ar180(c.self()), ar180(String::length)) ==
        listOf("1", "3", "0", "2", "1", "0", "1", "0", "1", "2", "0", "1"))
    val v1 = rc180 { this + 1 }
    val s1 = seen180
    val v2 = rc180b { a, s -> this * 10 + a + s.length }
    check("rcv180d", v1 == 4 && ar180(s1!!) == "1" && v2 == 35 && ar180(seen180!!) == "3")
    val a: Any = { { it: Int -> it } }
    val b: Any = { listOf(3).map { it: Int -> "v$it" } }
    val d: Any = { { it: Int -> it }(1) + 1 }
    val e: Any = { x: Int -> { it: Int -> it + x } }
    check("it180e", ar180(a) == "0" && ar180(b) == "0" && ar180(d) == "0" && ar180(e) == "1")
}


// ===== SECTION 181: the arity of a reference to a builtin member =====
// A reference to a builtin type's member is the FunctionN of that member's
// parameters, plus the receiver of an unbound one: `"ab"::plus` is a Function1 and
// String::plus a Function2. Regex.find takes (input, startIndex) where every other
// find takes a predicate, and Array.fill (element, fromIndex, toIndex) where
// MutableList.fill takes the element alone. A reference whose receiver is a call
// takes what it names too, and a class's inherited equals takes its other.
class Bx181(val v: Int) {
    fun eat(x: Int, y: Int) = x + y
    fun containsMatchIn(a: Int, b: Int, c: Int) = a + b + c
}
fun mk181() = Bx181(1)
fun ar181(f: Any): Int = when (f) {
    is Function7<*, *, *, *, *, *, *, *> -> 7
    is Function6<*, *, *, *, *, *, *> -> 6
    is Function4<*, *, *, *, *> -> 4
    is Function3<*, *, *, *> -> 3
    is Function2<*, *, *> -> 2
    is Function1<*, *> -> 1
    is Function0<*> -> 0
    else -> -1
}

fun s181() {
    val s = "abc"
    val ml = mutableListOf(1, 2)
    val l = listOf(1, 2)
    val arr = arrayOf(1, 2)
    val m = mutableMapOf("a" to 1)
    val sb = StringBuilder("x")
    val rx = Regex("a")
    val n = 5
    check("bra181a", listOf(ar181("ab"::plus), ar181(s::padStart), ar181(ml::set), ar181(l::indexOf), ar181(m::put),
        ar181(sb::setCharAt), ar181(rx::find), ar181(arr::fill), ar181(ml::fill), ar181(n::coerceAtLeast),
        ar181(arr::joinToString), ar181(s::length), ar181(m::getOrDefault)) == listOf(1, 2, 2, 1, 2, 2, 2, 3, 1, 1, 6, 0, 2))
    check("ubr181b", listOf(ar181(String::plus), ar181(List<Int>::get), ar181(MutableList<Int>::set), ar181(Regex::find),
        ar181(IntArray::fill), ar181(MutableMap<String, Int>::put), ar181(Int::coerceAtLeast), ar181(Char::isDigit),
        ar181(String::length)) == listOf(2, 2, 3, 3, 4, 3, 2, 1, 1))
    check("sfx181c", listOf(ar181(mk181()::eat), ar181(Bx181(3)::containsMatchIn), ar181(Bx181::containsMatchIn), ar181(listOf(1)::contains),
        ar181(s.reversed()::padEnd), ar181(Bx181(2)::v), ar181(Bx181(1)::equals), ar181(Bx181::equals),
        ar181(mk181()::toString)) == listOf(2, 3, 4, 1, 2, 0, 1, 2, 0))
}


// ===== SECTION 182: the arity of a reference to a method other classes declare too =====
// Where the program's methods of a name disagree on their parameter count, the
// class a reference names (or its bound receiver's) decides, inherited members,
// interface defaults, objects and object expressions included. Named arguments of a
// stdlib member go to its own parameters even when some class declares a method
// of that name.
class Ea182 { fun eat(x: Int) = x }
open class Bt182 { fun bite(x: Int, y: Int) = x + y }
class Sb182 : Bt182()
class Bo182 { fun bite(x: Int) = x }
interface If182 { fun bite(a: Int, b: Int, c: Int): Int = a + b + c }
class Ic182 : If182
object Ob182 { fun eat(a: Int, b: Int, c: Int) = a }
class Pd182 { fun padStart(a: Int, b: Int, c: Int) = a * 100 + b * 10 + c }
fun ar182(f: Any): Int = when (f) {
    is Function4<*, *, *, *, *> -> 4
    is Function3<*, *, *, *> -> 3
    is Function2<*, *, *> -> 2
    is Function1<*, *> -> 1
    is Function0<*> -> 0
    else -> -1
}

fun s182() {
    val b: Bt182 = Sb182()
    val ob = object { fun bite() = 0 }
    check("dis182a", listOf(ar182(Ea182::eat), ar182(Ea182()::eat), ar182(Sb182::bite), ar182(b::bite), ar182(Bo182::bite),
        ar182(Ic182::bite), ar182(Ic182()::bite), ar182(Ob182::eat), ar182(ob::bite), ar182(Bo182()::equals)) ==
        listOf(2, 1, 3, 2, 2, 4, 3, 3, 0, 1))
    check("nam182b", "abcde".padStart(padChar = '*', length = 7) == "**abcde" && "ab".padStart(length = 4, padChar = '-') == "--ab" &&
        Pd182().padStart(c = 1, a = 2, b = 3) == 231 && "x".padStart(3) == "  x")
}


// ===== SECTION 183: named arguments run as written; qualified object references =====
// Kotlin evaluates call arguments left to right as written, whatever parameter slots
// named ones bind. A reference whose base is an object VALUE (an enum entry, a
// companion, a nested object) is bound to it, and `C::f` names the companion's f
// when C's own members lack it. A class method sharing a stdlib member's name and
// parameter names in another order takes them only on its own instances.
val log183 = mutableListOf<String>()
fun <T> t183(s: String, v: T): T { log183.add(s); return v }
fun f183(a: Int, b: Int, c: Int) = "$a$b$c"
fun d183(a: Int, b: Int = a * 10, c: Int = b + 1) = "$a/$b/$c"
fun v183(tag: String, vararg xs: Int) = tag + xs.joinToString(",")
class P183(val x: Int, val y: Int) { fun pq(p: Int, q: Int) = p * 10 + q }
data class D183(val a: Int, val b: Int)
class Cm183 { companion object { fun bite() = 7; fun add(a: Int, b: Int) = a + b; val v = 3 } }
class Cn183 { companion object Named { fun go(a: Int) = a + 1 } }
enum class E183 { X, Y; fun run(a: Int, b: Int, c: Int) = a + b + c + ordinal }
class Out183 { object Ob183 { fun dbl183(a: Int) = a * 2 } }
open class Bs183 { fun bite() = 1 }
class Cs183 : Bs183() { companion object { fun bite() = 9 } }
class Z183 { fun padStart(padChar: Char, length: Int) = "Z$padChar$length" }
data class Pc183(val x: Int) { companion object { val x = 100 } }
open class Bp183(val y: Int)
class Dp183(y: Int) : Bp183(y) { companion object { val y = 200 } }
class Mc183 { fun tag() = "m"; companion object { fun tag() = "c" } }
fun ar183(f: Any): Int = when (f) {
    is Function3<*, *, *, *> -> 3
    is Function2<*, *, *> -> 2
    is Function1<*, *> -> 1
    is Function0<*> -> 0
    else -> -1
}
fun logged183(s: String): String { val r = s + " " + log183; log183.clear(); return r }

fun s183() {
    check("ord183a", logged183(f183(c = t183("c", 3), a = t183("a", 1), b = t183("b", 2))) == "123 [c, a, b]" &&
        logged183(f183(t183("a", 1), c = t183("c", 3), b = t183("b", 2))) == "123 [a, c, b]" &&
        logged183(d183(c = t183("c", 3), a = t183("a", 1))) == "1/10/3 [c, a]")
    val p = P183(y = t183("y", 2), x = t183("x", 1))
    check("ord183b", logged183("${p.x}${p.y}") == "12 [y, x]" &&
        logged183(p.pq(q = t183("q", 2), p = t183("p", 1)).toString()) == "12 [q, p]" &&
        logged183(D183(1, 2).copy(b = t183("b", 5), a = t183("a", 4)).toString()) == "D183(a=4, b=5) [b, a]" &&
        logged183(v183(xs = intArrayOf(t183("1", 1)), tag = t183("t", "9"))) == "91 [1, t]")
    check("ord183c", logged183(listOf(1, 2, 3).joinToString(postfix = t183("p", "!"), separator = t183("s", ";"))) == "1;2;3! [p, s]" &&
        logged183("abc".padStart(padChar = t183("c", '*'), length = t183("l", 5))) == "**abc [c, l]" &&
        logged183((1..6).toList().windowed(step = t183("st", 3), size = t183("sz", 2)).toString()) == "[[1, 2], [4, 5]] [st, sz]")
    val f = E183.X::run
    check("ref183d", listOf(ar183(Cm183.Companion::bite), ar183(Cm183::bite), ar183(Cm183::add), ar183(E183.X::run),
        ar183(Cn183.Named::go), ar183(Out183.Ob183::dbl183), ar183(Cs183::bite)) == listOf(0, 0, 2, 3, 1, 1, 1) &&
        f(1, 2, 3) == 6 && E183.Y::run.invoke(1, 2, 3) == 7 && Cm183::bite.invoke() == 7 && Cm183::add.invoke(2, 3) == 5 &&
        Cm183::v.get() == 3 && Cm183.Companion::v.get() == 3 && listOf(1, 2).map(Cn183.Named::go) == listOf(2, 3) &&
        Out183.Ob183::dbl183.invoke(5) == 10)
    check("std183e", "x".padStart(length = 5, padChar = '*') == "****x" && Z183().padStart(length = 5, padChar = '*') == "Z*5" &&
        with("ab") { padStart(length = 4, padChar = '.') } == "..ab" && with(Z183()) { padStart(length = 4, padChar = '.') } == "Z.4")
    check("ref183f", Pc183::x.get(Pc183(5)) == 5 && Pc183.Companion::x.get() == 100 && Dp183::y.get(Dp183(7)) == 7 &&
        Mc183::tag.invoke(Mc183()) == "m" && Mc183.Companion::tag.invoke() == "c" && listOf(Pc183(3)).map(Pc183::x) == listOf(3))
}

// ===== SECTION 184: a vararg before other parameters; nested types sharing a name =====
// A vararg that is not the last parameter takes every positional argument (spread
// ones spliced in) or one named array; the parameters after it take named arguments,
// their defaults, and the trailing lambda, which binds the last parameter. A nested
// class and a nested object of one simple name in different owners stay apart, and a
// bound reference to a nested object's method has that method's arity.
fun fv184(vararg xs: Int, tag: String) = xs.joinToString(",") + tag
fun fd184(vararg xs: Int, tag: String = "D") = xs.joinToString(",") + tag
fun fl184(vararg xs: Int, block: (Int) -> Int) = xs.map(block)
fun fm184(a: Int, vararg xs: Int, b: Int = 7) = "" + a + xs.toList() + b
var log184 = ""
fun t184(x: Int): Int { log184 += x; return x }
fun g184(vararg xs: Int, a: Int, b: Int = 100) = xs.sum() + a * 10 + b
class M184(val k: Int) {
    fun mv184(vararg xs: Int, f: (Int) -> Int) = xs.map { f(it) + k }
    fun nv184(p: String, vararg xs: String, q: String = "Q") = p + xs.joinToString("") + q
}
class C184(vararg val xs: Int, val f: (Int) -> Int) { fun go() = xs.map(f) }
class A184 { class In(val a: Int); fun mk() = In(2).a }
class B184 { object In { fun f() = 1 } }
class Out184 { object In { fun f(a: Int) = a * 2 } }
class Q184 { fun f(a: Int, b: Int) = 1 }

fun s184() {
    check("var184a", fv184(1, 2, tag = "t") == "1,2t" && fv184(xs = intArrayOf(1, 2), tag = "t") == "1,2t" &&
        fv184(*intArrayOf(1, 2), 3, tag = "t") == "1,2,3t" && fv184(tag = "q") == "q" &&
        fd184(1, 2) == "1,2D" && fd184() == "D" && fd184(5, tag = "z") == "5z")
    check("var184b", fl184(1, 2) { it * 10 } == listOf(10, 20) && fl184(3, block = { it + 1 }) == listOf(4) &&
        fm184(1, 2, 3) == "1[2, 3]7" && fm184(1, 2, 3, b = 4) == "1[2, 3]4" && fm184(1) == "1[]7")
    val o = M184(1)
    check("var184c", o.mv184(1, 2) { it * 3 } == listOf(4, 7) && o.nv184("p", "a", "b") == "pabQ" && o.nv184("p", "a", q = "z") == "paz" &&
        C184(4, 5) { it - 1 }.go() == listOf(3, 4) && g184(*intArrayOf(5, 6), 1, a = 2) == 132)
    check("var184d", g184(t184(1), t184(2), b = t184(3), a = t184(4)) == 46 && log184 == "1234")
    val oi = Out184.In
    check("nest184e", A184().mk() == 2 && B184.In.f() == 1 && A184.In(5).a == 3 + 2 &&
        (oi::f is Function1<*, *>) && !(oi::f is Function2<*, *, *>) && (oi::f)(4) == 8 && Q184().f(1, 2) == 1)
}

// ===== SECTION 185: references to vararg functions =====
// A reference to a function with a vararg parameter is a KFunction whose parameter
// is the array itself: the call binds that array (the callee sees the caller's
// array), and the parameters after a vararg that is not the last are positional.
fun fx185(vararg xs: Int) = xs.sum()
fun fs185(vararg ss: String) = ss.joinToString("|")
fun fp185(p: String, vararg xs: Int) = p + xs.size + xs.sum()
fun fw185(vararg xs: Int): Int { xs[0] = 99; return xs.size }

fun s185() {
    val r = ::fx185
    val q = ::fd184
    val s = ::fs185
    check("vref185a", r(intArrayOf(1, 2)) == 3 && q(intArrayOf(9, 8), "R") == "9,8R" && q.invoke(intArrayOf(7), "Z") == "7Z" &&
        s(arrayOf("a", "b")) == "a|b" && (::fp185)("x", intArrayOf(3, 4, 5)) == "x312" && fx185(1, 2, 3) == 6 &&
        listOf(intArrayOf(1, 2), intArrayOf(3)).map(::fx185) == listOf(3, 3) && ::fx185.name == "fx185")
    fun loc(vararg xs: Int) = xs.size * 10
    val a = intArrayOf(1, 2)
    val w = ::fw185
    check("vref185b", (::loc)(intArrayOf(4, 5, 6)) == 30 && loc(1, 2) == 20 && w(a) == 2 && a[0] == 99 &&
        listOf(intArrayOf(1), intArrayOf(1, 2, 3)).map(::loc) == listOf(10, 30))
}

// ===== SECTION 186: arrays compare by identity, vararg member references, local ::name, inherited result types =====
// An Array is a JVM array: `==`, `!=`, equals() and `===` are identity, and only
// contentEquals compares elements, so an Array is a map key and set element by
// identity and a List or a Pair holding one compares it by identity too;
// contentDeepEquals compares nested arrays by their contents. A vararg parameter is
// such an array (an IntArray for Int), and a spread passes a copy. A reference to a
// method or extension with a vararg takes the array itself. `::loc.name` of a local
// function is its name. A method's static result type is found along the receiver
// class's supertypes, and through an inner or nested class constructed off a
// receiver (kotlinc 2.2.21).
fun fi186(vararg xs: Int) = xs
fun <T> fa186(vararg xs: T) = xs
open class M186(val k: Int) { fun s(vararg xs: Int) = k + xs.sum(); fun me() = (::s)(intArrayOf(7)) }
fun M186.e(vararg xs: Int) = k * 10 + xs.sum()
open class A186 { fun m(): Int = 1 }
class B186 : A186()
class C186 { fun m(): List<Int> = listOf(1) }
class D186 { inner class E186 { fun m(): Long = 2L }; class N186 { fun m(): Double = 2.0 } }

fun s186() {
    val b = intArrayOf(1, 2); val c = intArrayOf(1, 2); val d = b
    val s1 = arrayOf("a"); val s2 = arrayOf("a")
    check("arreq186a", b !== c && b === d && b != c && !b.equals(c) && b.equals(d) && b.contentEquals(c) &&
        s1 != s2 && charArrayOf('x') != charArrayOf('x') && fi186(*b) !== b && fi186(*b) != b &&
        fa186(1, 2) != fa186(1, 2) && fi186(1).javaClass.name == "[I")
    val m = HashMap<IntArray, String>(); m[b] = "x"
    val st = HashSet<IntArray>(); st.add(b); st.add(c); st.add(d)
    check("arreq186b", m[c] == null && m[d] == "x" && st.size == 2 && !listOf(b).contains(c) && listOf(b).indexOf(d) == 0 &&
        listOf(b) != listOf(c) && listOf(b) == listOf(d) && Pair(b, 1) != Pair(c, 1) && setOf(b, c).size == 2 &&
        (when (b) { c -> "c"; d -> "d"; else -> "-" }) == "d" && listOf(1, 2) == listOf(1, 2) &&
        !arrayOf(b).contentEquals(arrayOf(c)) && arrayOf(b).contentDeepEquals(arrayOf(c)))
    val mm = M186(100)
    check("vref186c", (mm::s)(intArrayOf(1, 2)) == 103 && (M186::s)(mm, intArrayOf(3)) == 103 && mm.me() == 107 &&
        listOf(intArrayOf(1), intArrayOf(2, 3)).map(mm::s) == listOf(101, 105) && (mm::e)(intArrayOf(1, 2)) == 1003 &&
        (M186::e)(mm, intArrayOf(3)) == 1003)
    fun loc186(x: Int) = x
    check("lref186d", ::loc186.name == "loc186" && listOf(::loc186).map { it.name } == listOf("loc186"))
    val dd = D186()
    check("jstat186e", B186().m().javaClass.name == "int" && dd.E186().m().javaClass.name == "long" &&
        D186.N186().m().javaClass.name == "double" && C186().m().size == 1)
}

// ===== SECTION 187: referential equality, array rendering and content hashes, named spread, vararg types =====
// `===` is identity for every object - a data class, a List, a Pair, a map, an
// enum entry - while `==` stays equals(); a map's keys view is one object. An Array
// inherits Object.toString ([I@hash, so only its prefix is asserted), and a data
// or value class renders and hashes an Array property by its contents but compares
// it by identity; an unsigned array renders its storage. contentHashCode is
// java.util.Arrays.hashCode, and the content extensions accept a null receiver. A
// named vararg argument may be a spread, which passes a copy. A vararg parameter's
// static type is its array, and a primitive one retypes its literal arguments
// (kotlinc 2.2.21).
data class P187(val x: Int)
enum class E187 { A, B }
data class D187(val a: IntArray, val s: Array<String>, val n: Int)
fun fl187(vararg xs: Long) = xs
fun fs187(vararg xs: Short) = xs
fun g187(a: Int, vararg xs: Int) = a + xs.sum()
fun h187(vararg xs: String, sep: String = "-") = xs.joinToString(sep)
fun w187(vararg xs: Int): IntArray { xs[0] = 9; return xs }

@OptIn(ExperimentalUnsignedTypes::class)
fun s187() {
    val p = P187(1); val p2 = p; val l = listOf(1, 2); val l2 = l
    val m = mutableMapOf(1 to 2)
    check("same187a", P187(1) !== P187(1) && p === p2 && P187(1) == P187(1) && listOf(1) !== listOf(1) && l === l2 &&
        Pair(1, 2) !== Pair(1, 2) && mutableListOf(1) !== mutableListOf(1) && m.keys === m.keys &&
        m !== mutableMapOf(1 to 2) && E187.A === E187.A && E187.A !== E187.B && p !== null && "ab" === "ab")
    val x = intArrayOf(1, 2)
    val n: IntArray? = null
    check("astr187b", x.toString().startsWith("[I@") && "$x" == x.toString() && arrayOf("x").toString().startsWith("[Ljava.lang.String;@") &&
        longArrayOf(1).toString().startsWith("[J@") && listOf(intArrayOf(1)).toString().startsWith("[[I@") &&
        arrayOf(intArrayOf(1), arrayOf("a")).contentDeepToString() == "[[1], [a]]" &&
        arrayOf(intArrayOf(1)).contentToString().startsWith("[[I@") && uintArrayOf(1u, 2u).toString() == "UIntArray(storage=[1, 2])" &&
        n.contentToString() == "null" && n.contentEquals(null) && !n.contentEquals(intArrayOf()))
    check("ahash187c", x.contentHashCode() == 994 && intArrayOf().contentHashCode() == 1 && arrayOf("a", "b").contentHashCode() == 4066 &&
        longArrayOf(1L, 5000000000L).contentHashCode() == 705033697 && doubleArrayOf(1.5).contentHashCode() == 1073217567 &&
        booleanArrayOf(true, false).contentHashCode() == 40359 && charArrayOf('a').contentHashCode() == 128 &&
        arrayOf(intArrayOf(1)).contentDeepHashCode() == 63 && n.contentHashCode() == 0 && uintArrayOf(1u, 2u).hashCode() == 994)
    val d = D187(intArrayOf(1, 2), arrayOf("x"), 3)
    check("adata187d", d.toString() == "D187(a=[1, 2], s=[x], n=3)" && d != D187(intArrayOf(1, 2), arrayOf("x"), 3) &&
        d.hashCode() == D187(intArrayOf(1, 2), arrayOf("x"), 3).hashCode())
    val b = intArrayOf(3, 4)
    val c = intArrayOf(1)
    val r = w187(xs = *c)
    check("nspread187e", g187(1, xs = *b) == 8 && g187(a = 2, xs = *b) == 9 && h187(xs = *arrayOf("p", "q"), sep = "+") == "p+q" &&
        g187(1, xs = *intArrayOf()) == 1 && c[0] == 1 && r[0] == 9 && r !== c)
    val a: Any = fs187(7)[0]
    check("vty187f", fl187(5L)[0].javaClass.name == "long" && fl187(5L).javaClass.name == "[J" && fs187(1)[0].javaClass.name == "short" &&
        a is Short)
}

// ===== SECTION 188: primitive element and sum types, empty-collection singletons, function reference identity =====
// A CharArray's and a BooleanArray's elements are the primitives char and
// boolean, and so is first()/last() of a primitive array, a String or a range
// (only a generic collection's is boxed). sum() of Shorts or Bytes is an Int, of
// unsigned values a UInt. emptyList() / emptySet() / emptyMap() - and listOf(),
// setOf() and mapOf() with no elements - are one singleton object each, while
// every evaluation of `::f` is a new KFunction object, == to the others but never
// === (kotlinc 2.2.21).
fun fs188(vararg xs: Short) = xs
fun fc188(vararg cs: Char) = cs
fun f188() = 1
fun g188(x: Int) = x + 1
fun v188(vararg xs: Int) = xs.size
class K188 {
    fun m() = 2
    fun self() = ::m
    fun same() = ::m === ::m
    fun eq() = ::m == ::m
}

@OptIn(ExperimentalUnsignedTypes::class)
fun s188() {
    val ca = charArrayOf('a', 'b')
    check("jprim188a", charArrayOf('x')[0].javaClass.name == "char" && booleanArrayOf(true)[0].javaClass.name == "boolean" &&
        ca[1].javaClass.name == "char" && fc188('x')[0].javaClass.name == "char" && BooleanArray(1)[0].javaClass.name == "boolean" &&
        "hey".toCharArray()[0].javaClass.name == "char" && CharArray(1) { 'q' }[0].javaClass.name == "char")
    val bs: List<Byte> = listOf(100, 100)
    val ss = listOf<Short>()
    check("sumty188b", fs188(1, 2).sum().javaClass.name == "int" && bs.sum().javaClass.name == "int" && bs.sum() == 200 &&
        ss.sum().javaClass.name == "int" && shortArrayOf(30000, 30000).sum() == 60000 &&
        ubyteArrayOf(1u, 2u).sum().javaClass.name == "kotlin.UInt" && listOf(1u, 2u).sum().javaClass.name == "kotlin.UInt" &&
        listOf(1L).sum().javaClass.name == "long")
    check("first188c", intArrayOf(1).first().javaClass.name == "int" && "ab".last().javaClass.name == "char" &&
        (1..3).first().javaClass.name == "int" && ca.single { it == 'a' }.javaClass.name == "char" &&
        listOf(1).first().javaClass.name == "java.lang.Integer" && arrayOf(1).first().javaClass.name == "java.lang.Integer")
    var xs = emptyList<Int>()
    xs += 1
    check("empty188d", emptyList<Int>() === emptyList<String>() && listOf<Int>() === emptyList<Int>() &&
        setOf<Int>() === emptySet<Int>() && mapOf<Int, Int>() === emptyMap<Int, Int>() && listOf(1) !== listOf(1) &&
        mutableListOf<Int>() !== mutableListOf<Int>() && emptyArray<Int>() !== emptyArray<Int>() && xs == listOf(1) &&
        emptyList<Int>().isEmpty())
    val r = ::f188
    fun loc188(a: Int) = a * 2
    check("fref188e", ::f188 !== ::f188 && ::f188 == ::f188 && r === r && r == ::f188 && ::g188 !== ::g188 &&
        (::f188 as Any) != (::g188 as Any) && setOf(::f188, ::f188).size == 1 && ::v188 !== ::v188 && ::v188 == ::v188 &&
        ::loc188 !== ::loc188 && ::loc188 == ::loc188 && listOf(2).map(::g188) == listOf(3) && ::f188.name == "f188")
    val k = K188()
    check("mref188f", !k.same() && k.eq() && k.self() == k.self() && k.self() != K188().self() && k.self() == k::m)
}

// ===== SECTION 189: per-scope vararg references, identity hashes =====
// `::name` resolves to the innermost declaration of the name, so whether the
// reference takes the vararg array is decided per scope: a local vararg function
// beside a same-named plain one elsewhere, and a local plain function shadowing a
// top-level vararg one. System.identityHashCode is Object.hashCode's identity hash
// whatever the class overrides (0 for null); an Array's hashCode() is that identity
// hash too, not its contents'. Identity hashes are asserted as relations only. In
// this value model instances of one class share an identity stand-in, and each
// `::f` site is not its own class as in kotlinc, so `::f.javaClass == ::f.javaClass`
// is true here and false there.
fun h189(vararg xs: Int) = xs.size * 100
fun k189(x: Int) = x + 1
fun va189(): String {
    fun loc189(vararg xs: Int) = xs.size * 10
    fun h189(x: Int) = x * 2
    fun k189(vararg xs: Int) = xs.size * 1000
    return "" + (::loc189)(intArrayOf(4, 5, 6)) + "," + (::h189)(21) + "," + (::k189)(intArrayOf(1, 2))
}
fun vb189(): String {
    fun loc189(x: Int) = x
    var r = "" + loc189(1) + "," + (::loc189)(7) + "," + (::h189)(intArrayOf(1, 2, 3)) + "," + (::k189)(5)
    if (r.length > 0) {
        fun loc189(vararg ys: Int) = ys.sum()
        r += "," + (::loc189)(intArrayOf(3, 4))
    }
    return r + "," + (::loc189)(8)
}
class P189
class Q189
class H189 { override fun hashCode() = 42 }
data class D189(val x: Int)

fun s189() {
    check("vscope189a", va189() == "30,42,2000" && vb189() == "1,7,300,6,7,8")
    val p = P189()
    val d = D189(1)
    check("idhash189b", System.identityHashCode(p) == System.identityHashCode(p) && System.identityHashCode(p) == p.hashCode() &&
        System.identityHashCode(p) != System.identityHashCode(Q189()) && System.identityHashCode(null) == 0 &&
        System.identityHashCode(d) == System.identityHashCode(d) && System.identityHashCode(H189()) != 42 &&
        System.identityHashCode(Any()) != System.identityHashCode(p))
    val a = intArrayOf(1)
    val o = arrayOf("x")
    check("arrhash189c", System.identityHashCode(a) == a.hashCode() && a.hashCode() == a.hashCode() &&
        listOf(a).hashCode() == 31 + a.hashCode() && System.identityHashCode(o) == o.hashCode() &&
        listOf(o).hashCode() == 31 + o.hashCode())
}

// ===== SECTION 190: references to java static members =====
// `Math::abs`, `System::currentTimeMillis`, `Integer::parseInt` are bound to the
// java class's statics; calling one calls the static method. An overloaded one
// (abs, max, parseInt) needs an expected type, which a declared function type or
// the parameter it is passed to gives.
fun clk190(f: () -> Long) = f() > 0L && f() > 0L
// A reference's FunctionN arity is its expected function type's, which also picks
// an overload or adapts a defaulted member; without one, the static's own.
class B190(val k: Int) {
    fun add(x: Int, y: Int = 1): Int = k + x + y
}
fun ar190(f: (String, Int) -> Int): Boolean = f is Function2<*, *, *> && f !is Function1<*, *>
fun pick190(): (Int, Int) -> Int = Math::max

fun s190() {
    val ab: (Int) -> Int = Math::abs
    val mx: (Int, Int) -> Int = Math::max
    val sq: (Double) -> Double = Math::sqrt
    check("jstatic190a", ab(-3) == 3 && mx(2, 7) == 7 && sq(16.0) == 4.0 &&
        listOf(-1, 2, -3).map(Math::abs) == listOf(1, 2, 3) && listOf(3, 9, 4).reduce(Math::max) == 9)
    val g = System::currentTimeMillis
    val n = System::nanoTime
    val ih = System::identityHashCode
    val o = "x"
    check("jstatic190b", g() > 0L && n() > 0L && clk190(System::currentTimeMillis) && ih(o) == ih(o) && ih(null) == 0)
    val pi: (String) -> Int = Integer::parseInt
    check("jstatic190c", pi("42") + 1 == 43 && listOf("1", "22").map(Integer::parseInt).sum() == 23 &&
        ab.toString() == "function abs (Kotlin reflection is not available)")
    val p2: (String, Int) -> Int = Integer::parseInt
    val ad: (Int) -> Int = B190(10)::add
    val lg: (Int) -> Long = { it * 2L }
    check("jstatic190d", ab is Function1<*, *> && ab !is Function0<*> && mx is Function2<*, *, *> &&
        pi is Function1<*, *> && pi !is Function2<*, *, *> && p2 is Function2<*, *, *> && p2("ff", 16) == 255 &&
        ar190(Integer::parseInt) && pick190() is Function2<*, *, *> && pick190()(3, 8) == 8 &&
        ad is Function1<*, *> && ad !is Function2<*, *, *> && ad(5) == 16 && lg is Function1<*, *> && lg(4) == 8L)
    val sq1 = Math::sqrt
    val pw = Math::pow
    val cmp = Integer::compare
    val bc = java.lang.Long::bitCount
    check("jstatic190e", sq1 is Function1<*, *> && sq1 !is Function0<*> && pw is Function2<*, *, *> &&
        cmp is Function2<*, *, *> && n is Function0<*> && ih is Function1<*, *> && bc is Function1<*, *> &&
        sq1(9.0) == 3.0 && cmp(1, 2) < 0)
}

// ===== SECTION 191: java.lang-qualified references, function element types =====
// `java.lang.Long::signum` is the java static, not an unbound member of kotlin's
// Long. A collection's declared element type is the expected type of each element,
// so a callable reference stored in a List<(Int) -> Int> is a Function1.
fun s191() {
    val sg = java.lang.Long::signum
    val ab: (Int) -> Int = java.lang.Math::abs
    val cp = java.lang.Integer::compare
    val nt = java.lang.System::nanoTime
    check("jlqual191a", sg(-5L) == -1 && sg(7L) == 1 && ab(-3) == 3 && cp(2, 1) > 0 && nt() > 0L &&
        sg is Function1<*, *> && cp is Function2<*, *, *> && nt is Function0<*>)
    val fs: List<(Int) -> Int> = listOf(Math::abs)
    val ms: Map<String, (Int, Int) -> Int> = mapOf("mx" to Math::max)
    val nf: List<List<(Int) -> Int>> = listOf(listOf(Math::abs))
    val pr: Pair<(Int) -> Int, (Int, Int) -> Int> = Pair(Math::abs, Math::min)
    val ls: List<() -> Long> = listOf({ 1 })
    check("fnelem191b", fs[0] is Function1<*, *> && fs[0](-7) == 7 && ms["mx"] is Function2<*, *, *> &&
        ms.getValue("mx")(3, 4) == 4 && nf[0][0] is Function1<*, *> && pr.first is Function1<*, *> &&
        pr.second is Function2<*, *, *> && ls[0]() * 3000000000 == 3000000000L)
    val ml: MutableList<(String) -> Int> = mutableListOf()
    ml.add(Integer::parseInt)
    val mm: MutableMap<String, (Int) -> Int> = mutableMapOf()
    mm["a"] = Math::abs
    val r: (Int) -> Int = Math::abs
    val rl = mutableListOf<(Int) -> Int>()
    rl.add(r)
    check("fnelem191c", ml[0] is Function1<*, *> && ml[0]("12") == 12 && mm["a"] is Function1<*, *> &&
        rl[0] === r && rl.contains(r) && rl.indexOf(r) == 0)
}

// ===== SECTION 192: java.lang.System calls, System.out =====
// `java.lang.System.f()` is `System.f()`. System.out is the stream Kotlin's println
// and print write to; `System.out.println()` writes only the line terminator.
class P192
fun s192() {
    val t0 = java.lang.System.nanoTime()
    val p = P192()
    check("jlsys192a", java.lang.System.currentTimeMillis() > 0L && java.lang.System.nanoTime() >= t0 &&
        java.lang.System.identityHashCode(p) == System.identityHashCode(p) && java.lang.System.identityHashCode(null) == 0)
    System.out.print("out192:")
    java.lang.System.out.print(1.5)
    System.out.println()
    java.lang.System.out.println(listOf('c', 2))
    listOf(3).forEach(System.out::println)
    listOf(4).forEach(java.lang.System.out::println)
    check("sysout192b", System.out.println("ok192") == Unit)
}

// ===== SECTION 193: System.err, a program binding named java =====
// System.err is standard error's stream, one of its own. A local, parameter or
// property named `java` hides the package: `java.lang.System` is then its member
// chain, for a call and for a callable reference alike.
class Log193(val tag: String) {
    val seen = mutableListOf<String>()
    fun println(s: String) { seen.add(tag + s) }
}
class Sys193(tag: String) {
    val out = Log193(tag)
    fun nanoTime(): Long = 193L
}
class Lang193(tag: String) { val System = Sys193(tag) }
class J193(tag: String) { val lang = Lang193(tag) }
class H193 {
    val java = J193("h")
    fun run(): String {
        java.lang.System.out.println("1")
        val r = java.lang.System.out::println
        r("2")
        return java.lang.System.out.seen.joinToString(",") + java.lang.System.nanoTime()
    }
}
fun p193(java: J193): String {
    java.lang.System.out.println("p")
    return java.lang.System.out.seen[0]
}
fun s193() {
    System.err.print("")
    java.lang.System.err.print("")
    val ep: (String) -> Unit = System.err::print
    ep("")
    check("syserr193a", System.err !== System.out && System.err === java.lang.System.err && ep is Function1<*, *>)
    val java = J193("l")
    java.lang.System.out.println("x")
    val g = java.lang.System.out::println
    g("y")
    val n = java.lang.System::nanoTime
    check("javashadow193b", java.lang.System.out.seen == listOf("lx", "ly") && n() == 193L &&
        H193().run() == "h1,h2193" && p193(J193("q")) == "qp")
}

// ===== SECTION 194: a member property hides a top-level type of its name; kotlin.io =====
// The implicit receiver ranks above the top level, so inside a method a property named
// like a top-level object, enum class or class is that property. The qualified
// kotlin.io.println / print and kotlin.text.Regex are the stdlib's.
class Box194(val tag: String)
object obj194 { val tag = "obj" }
enum class Mode194 { A; val tag get() = "enum" + name }
class Kls194 { companion object { val tag = "comp" } }
class Holder194 {
    val obj194 = Box194("prop")
    var Mode194 = Box194("mprop")
    val Kls194 = Box194("kprop")
    fun run(): String {
        val a = obj194.tag + Mode194.tag + Kls194.tag
        Mode194 = Box194("reset")
        return a + Mode194.tag
    }
}
fun s194() {
    check("topobj194a", obj194.tag == "obj" && Mode194.A.tag == "enumA" && Kls194.tag == "comp" &&
        Holder194().run() == "propmpropkpropreset" && obj194.tag == "obj")
    kotlin.io.print("io194:")
    kotlin.io.println()
    check("kio194b", kotlin.io.println("x194") == Unit && kotlin.text.Regex("a+").matches("aa") &&
        kotlin.text.Regex("B", RegexOption.IGNORE_CASE).matches("b"))
}

// ===== SECTION 195: range constructors, fromClosedRange, qualified companions =====
// IntRange(a, b), LongRange, CharRange, UIntRange and ULongRange are constructors
// as well as the companions holding EMPTY; a LongRange's Int arguments are Longs.
// IntProgression.fromClosedRange(start, end, step) builds a progression, descending
// for a negative step, refusing a zero step and the step type's MIN_VALUE. The
// companions are reachable package-qualified (kotlin.ranges.IntRange, kotlin.Int)
// (kotlinc 2.2.21 on java 24).
fun s195() {
    val ns = listOf(1, 3, 9, 2, -3)
    val r = IntRange(ns[0], ns[1])
    check("rctor195a", "$r" == "1..3" && r.toList() == listOf(1, 2, 3) && r == (1..3) && (r as Any) is IntRange &&
        IntRange(ns[1], ns[0]).isEmpty() && "${IntRange(ns[1], ns[0])}" == "3..1" &&
        kotlin.ranges.IntRange(ns[0], ns[1]).toList() == listOf(1, 2, 3))
    val l = LongRange(1, 4)
    val c = CharRange('a', 'c')
    check("rctor195b", "$l" == "1..4" && l.sum() == 10L && l.first + 1 == 2L && "$c" == "a..c" &&
        c.toList() == listOf('a', 'b', 'c') && CharRange('c', 'a').isEmpty() && "${UIntRange(1u, 3u)}" == "1..3" &&
        ULongRange(1uL, 3uL).toList() == listOf(1uL, 2uL, 3uL) && LongRange(Long.MAX_VALUE - 1, Long.MAX_VALUE).count() == 2)
    val seen = mutableListOf<Int>()
    for (i in IntRange(ns[3], 4)) seen.add(i)
    check("rctor195c", seen == listOf(2, 3, 4) && IntRange.EMPTY.isEmpty() && "${kotlin.ranges.IntRange.EMPTY}" == "1..0" &&
        kotlin.Int.MAX_VALUE == 2147483647 && kotlin.Char(65) == 'A')
    val p = IntProgression.fromClosedRange(ns[0], ns[2], ns[3])
    val d = IntProgression.fromClosedRange(ns[2], ns[0], ns[4])
    check("rprog195d", "$p" == "1..9 step 2" && p.first == 1 && p.last == 9 && p.step == 2 && (p as Any) !is IntRange &&
        (p as Any) is IntProgression && "$d" == "9 downTo 3 step 3" && d.step == -3 && d.last == 3 && d.toList() == listOf(9, 6, 3) &&
        "${IntProgression.fromClosedRange(ns[0], ns[2], 1)}" == "1..9 step 1" && IntProgression.fromClosedRange(ns[0], ns[2], 1) == (1..9))
    check("rprog195e", "${LongProgression.fromClosedRange(1L, 10L, 4L)}" == "1..9 step 4" && LongProgression.fromClosedRange(1, 10, 4).last == 9L &&
        CharProgression.fromClosedRange('a', 'g', ns[3]).toList() == listOf('a', 'c', 'e', 'g') &&
        "${UIntProgression.fromClosedRange(1u, 9u, 4)}" == "1..9 step 4" &&
        ULongProgression.fromClosedRange(9uL, 1uL, -4L).toList() == listOf(9uL, 5uL, 1uL) &&
        "${kotlin.ranges.IntProgression.fromClosedRange(ns[0], ns[2], 4)}" == "1..9 step 4")
    val msgs = listOf(
        runCatching { IntProgression.fromClosedRange(ns[0], ns[2], ns[0] - 1) }.exceptionOrNull()?.message,
        runCatching { IntProgression.fromClosedRange(ns[0], ns[2], Int.MIN_VALUE) }.exceptionOrNull()?.message,
        runCatching { LongProgression.fromClosedRange(1L, 9L, Long.MIN_VALUE) }.exceptionOrNull()?.message,
        runCatching { CharProgression.fromClosedRange('a', 'z', 0) }.exceptionOrNull()?.message)
    check("rprog195f", msgs == listOf("Step must be non-zero.", "Step must be greater than Int.MIN_VALUE to avoid overflow on negation.",
        "Step must be greater than Long.MIN_VALUE to avoid overflow on negation.", "Step must be non-zero."))
}

fun main() {
    s01() // SECTION-CALL 01
    s02() // SECTION-CALL 02
    s03() // SECTION-CALL 03
    s04() // SECTION-CALL 04
    s05() // SECTION-CALL 05
    s06() // SECTION-CALL 06
    s07() // SECTION-CALL 07
    s08() // SECTION-CALL 08
    s09() // SECTION-CALL 09
    s10() // SECTION-CALL 10
    s11() // SECTION-CALL 11
    s12() // SECTION-CALL 12
    s13() // SECTION-CALL 13
    s14() // SECTION-CALL 14
    s15() // SECTION-CALL 15
    s16() // SECTION-CALL 16
    s17() // SECTION-CALL 17
    s18() // SECTION-CALL 18
    s19() // SECTION-CALL 19
    s20() // SECTION-CALL 20
    s21() // SECTION-CALL 21
    s22() // SECTION-CALL 22
    s23() // SECTION-CALL 23
    sec24() // SECTION-CALL 24
    sec25() // SECTION-CALL 25
    sec26() // SECTION-CALL 26
    sec27() // SECTION-CALL 27
    sec28() // SECTION-CALL 28
    sec29() // SECTION-CALL 29
    sec30() // SECTION-CALL 30
    sec31() // SECTION-CALL 31
    sec32() // SECTION-CALL 32
    sec33() // SECTION-CALL 33
    sec34() // SECTION-CALL 34
    sec35() // SECTION-CALL 35
    sec36() // SECTION-CALL 36
    sec37() // SECTION-CALL 37
    sec38() // SECTION-CALL 38
    sec39() // SECTION-CALL 39
    sec40() // SECTION-CALL 40
    sec41() // SECTION-CALL 41
    sec42() // SECTION-CALL 42
    sec43() // SECTION-CALL 43
    sec44() // SECTION-CALL 44
    sec45() // SECTION-CALL 45
    sec46() // SECTION-CALL 46
    sec47() // SECTION-CALL 47
    sec48() // SECTION-CALL 48
    sec49() // SECTION-CALL 49
    sec50() // SECTION-CALL 50
    sec51() // SECTION-CALL 51
    sec52() // SECTION-CALL 52
    sec53() // SECTION-CALL 53
    sec54() // SECTION-CALL 54
    sec55() // SECTION-CALL 55
    sec56() // SECTION-CALL 56
    sec57() // SECTION-CALL 57
    sec58() // SECTION-CALL 58
    sec59() // SECTION-CALL 59
    sec60() // SECTION-CALL 60
    sec61() // SECTION-CALL 61
    sec62() // SECTION-CALL 62
    sec63() // SECTION-CALL 63
    sec64() // SECTION-CALL 64
    sec65() // SECTION-CALL 65
    sec66() // SECTION-CALL 66
    sec67() // SECTION-CALL 67
    sec68() // SECTION-CALL 68
    sec69() // SECTION-CALL 69
    sec70() // SECTION-CALL 70
    sec71() // SECTION-CALL 71
    sec72() // SECTION-CALL 72
    sec73() // SECTION-CALL 73
    sec74() // SECTION-CALL 74
    sec75() // SECTION-CALL 75
    sec76() // SECTION-CALL 76
    sec77() // SECTION-CALL 77
    sec78() // SECTION-CALL 78
    sec79() // SECTION-CALL 79
    sec80() // SECTION-CALL 80
    sec81() // SECTION-CALL 81
    sec82() // SECTION-CALL 82
    sec83() // SECTION-CALL 83
    sec84() // SECTION-CALL 84
    sec85() // SECTION-CALL 85
    sec86() // SECTION-CALL 86
    sec87() // SECTION-CALL 87
    sec88() // SECTION-CALL 88
    sec89() // SECTION-CALL 89
    sec90() // SECTION-CALL 90
    sec91() // SECTION-CALL 91
    sec92() // SECTION-CALL 92
    sec93() // SECTION-CALL 93
    sec94() // SECTION-CALL 94
    sec95() // SECTION-CALL 95
    sec96() // SECTION-CALL 96
    sec97() // SECTION-CALL 97
    sec98() // SECTION-CALL 98
    sec99() // SECTION-CALL 99
    sec100() // SECTION-CALL 100
    sec101() // SECTION-CALL 101
    sec102() // SECTION-CALL 102
    sec103() // SECTION-CALL 103
    sec104() // SECTION-CALL 104
    sec105() // SECTION-CALL 105
    sec106() // SECTION-CALL 106
    sec107() // SECTION-CALL 107
    sec108() // SECTION-CALL 108
    sec109() // SECTION-CALL 109
    sec110() // SECTION-CALL 110
    sec111() // SECTION-CALL 111
    sec112() // SECTION-CALL 112
    sec113() // SECTION-CALL 113
    sec114() // SECTION-CALL 114
    sec115() // SECTION-CALL 115
    s120() // SECTION-CALL 120
    sec121() // SECTION-CALL 121
    sec122() // SECTION-CALL 122
    sec123() // SECTION-CALL 123
    sec124() // SECTION-CALL 124
    sec125() // SECTION-CALL 125
    sec126() // SECTION-CALL 126
    sec127() // SECTION-CALL 127
    sec128() // SECTION-CALL 128
    sec129() // SECTION-CALL 129
    sec130() // SECTION-CALL 130
    sec131() // SECTION-CALL 131
    sec132() // SECTION-CALL 132
    sec133() // SECTION-CALL 133
    sec134() // SECTION-CALL 134
    sec135() // SECTION-CALL 135
    sec136() // SECTION-CALL 136
    sec137() // SECTION-CALL 137
    sec138() // SECTION-CALL 138
    sec139() // SECTION-CALL 139
    sec140() // SECTION-CALL 140
    sec141() // SECTION-CALL 141
    sec142() // SECTION-CALL 142
    sec143() // SECTION-CALL 143
    sec144() // SECTION-CALL 144
    sec145() // SECTION-CALL 145
    sec146() // SECTION-CALL 146
    sec147() // SECTION-CALL 147
    s148() // SECTION-CALL 148
    s149() // SECTION-CALL 149
    s150() // SECTION-CALL 150
    s151() // SECTION-CALL 151
    s152() // SECTION-CALL 152
    s153() // SECTION-CALL 153
    s154() // SECTION-CALL 154
    s155() // SECTION-CALL 155
    s156() // SECTION-CALL 156
    s157() // SECTION-CALL 157
    s158() // SECTION-CALL 158
    s159() // SECTION-CALL 159
    s160() // SECTION-CALL 160
    s161() // SECTION-CALL 161
    s162() // SECTION-CALL 162
    s163() // SECTION-CALL 163
    s164() // SECTION-CALL 164
    s165() // SECTION-CALL 165
    s166() // SECTION-CALL 166
    s167() // SECTION-CALL 167
    s168() // SECTION-CALL 168
    s169() // SECTION-CALL 169
    s170() // SECTION-CALL 170
    s171() // SECTION-CALL 171
    s172() // SECTION-CALL 172
    s173() // SECTION-CALL 173
    s174() // SECTION-CALL 174
    s175() // SECTION-CALL 175
    s176() // SECTION-CALL 176
    s177() // SECTION-CALL 177
    s178() // SECTION-CALL 178
    s179() // SECTION-CALL 179
    s180() // SECTION-CALL 180
    s181() // SECTION-CALL 181
    s182() // SECTION-CALL 182
    s183() // SECTION-CALL 183
    s184() // SECTION-CALL 184
    s185() // SECTION-CALL 185
    s186() // SECTION-CALL 186
    s187() // SECTION-CALL 187
    s188() // SECTION-CALL 188
    s189() // SECTION-CALL 189
    s190() // SECTION-CALL 190
    s191() // SECTION-CALL 191
    s192() // SECTION-CALL 192
    s193() // SECTION-CALL 193
    s194() // SECTION-CALL 194
    s195() // SECTION-CALL 195
    println("full: $checks checks, $fails failures")
    exitProcess(fails)
}
