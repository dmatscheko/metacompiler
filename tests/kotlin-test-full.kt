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
    // IntProgression.hashCode is 31 * (31 * first + last) + step, and -1 when the
    // progression is empty.
    val rh = listOf(1..5, 5..1, 1..10 step 4, 'a'..'c')
    var v = ""
    for (p in rh) { v += "" + p.hashCode() + " " }
    check("rng17b", v == "1117 -1 1244 96287 " && (5..1).hashCode() == (7..3).hashCode())
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
    // Kotlin evaluates the receiver before the right-hand side, and the index of an
    // indexed slot after both. Pinned so the two halves cannot drift on the order.
    order28 = ""
    lhs28().tag = rhs28()
    check("cta6", order28 == "LR" && shared28.tag == 7)
    order28 = ""
    boxes28()[idx28()] = rhs28()
    check("cta7", order28 == "RI" && store28[1] == 7)
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
    check("chr4", s.toCharArray().toString() == "[a, B, c]")
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
    check("st44", "" + List(3) { it } == "[0, 1, 2]" && "" + arrayOfNulls<Int>(2) == "[null, null]")
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
    // The `Name@1` fallback still reads the SIMPLE name. That is deliberate: the @1
    // is a fixed stand-in for the JVM's identity hash, so "S82Outer$S82Plain@1"
    // would be no closer to the JVM's answer than "S82Plain@1" is.
    check("ix82q", S82Outer.S82Plain(1).toString() == "S82Plain@1")
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
    println("full: $checks checks, $fails failures")
    exitProcess(fails)
}
