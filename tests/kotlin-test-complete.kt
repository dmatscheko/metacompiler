/* Kotlin completion self test. Exercises the newly completed features:
 *   - object singletons (real backing objects: val/var fields, methods with implicit
 *     property access, expression-body methods, this.m() self dispatch, and an object
 *     method that runs a trailing-lambda higher-order call);
 *   - companion object members (const + mutable static-like state and methods reachable
 *     as ClassName.member / ClassName.m(), coexisting with the instance side);
 *   - when used as an EXPRESSION (subject and subject-less, comma values, else) AND as a
 *     STATEMENT (subject and subject-less, value discarded);
 *   - trailing-lambda higher-order calls / chaining and multi-statement lambda bodies.
 * main() counts failed checks and ends with exitProcess(fails), so the run exits 0
 * exactly when everything passes. **/

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

// ---- object singleton: val + var fields, methods, implicit this, this.m() dispatch ----
object Bank {
    val name = "acme"
    var balance = 100

    fun deposit(n: Int): Int {
        balance = balance + n        // implicit property write against this
        return balance
    }
    fun withdraw(n: Int): Int {
        balance = balance - n
        return balance
    }
    fun label(): String = name + ":" + balance   // expression-body method

    fun net(a: Int, b: Int): Int {   // calls sibling methods via this.m()
        this.deposit(a)
        this.withdraw(b)
        return balance
    }
}

// ---- object whose methods run trailing-lambda higher-order calls ----
object Stats {
    val data = listOf(1, 2, 3, 4, 5)
    fun sumSquares(): Int = data.sumOf { it * it }
    fun evensDoubled(): Int {
        val e = data.filter { it % 2 == 0 }
        return e.map { it * 2 }.sumOf { it }
    }
}

// ---- companion object: const + mutable members and methods, plus the instance side ----
class Account(val owner: String) {
    var funds = 0
    fun add(n: Int): Int {
        funds = funds + n
        return funds
    }

    companion object {
        val BANKCODE = 42
        var opened = 0
        fun open(): Int {
            opened = opened + 1          // mutable companion state
            return BANKCODE * 10 + opened
        }
        fun code(): Int = BANKCODE       // reads a companion const via implicit this
    }
}

fun grade(score: Int): String = when {   // subject-less when as an expression body
    score >= 90 -> "A"
    score >= 80 -> "B"
    score >= 70 -> "C"
    else -> "F"
}

fun main() {
    // object singleton
    checkS("object val field", Bank.name, "acme")
    check("object var init", Bank.balance, 100)
    check("object method mutate", Bank.deposit(50), 150)
    check("object method mutate2", Bank.withdraw(30), 120)
    check("object var persists", Bank.balance, 120)
    checkS("object expr-body method", Bank.label(), "acme:120")
    check("object this.m() dispatch", Bank.net(10, 5), 125)

    // object methods that use higher-order lambdas
    check("object sumOf lambda", Stats.sumSquares(), 55)
    check("object filter+map+sumOf", Stats.evensDoubled(), 12)

    // companion object: reachable by ClassName.member / ClassName.m()
    check("companion const read", Account.BANKCODE, 42)
    check("companion method const", Account.code(), 42)
    check("companion var+method 1", Account.open(), 421)
    check("companion var+method 2", Account.open(), 422)
    check("companion var persists", Account.opened, 2)

    // the instance side is independent of the companion
    val ac = Account("alice")
    checkS("instance ctor prop", ac.owner, "alice")
    check("instance method", ac.add(7), 7)
    check("instance method 2", ac.add(3), 10)
    check("companion unaffected by instance", Account.BANKCODE, 42)

    // when as an EXPRESSION, subject-less
    checkS("when-expr no subject A", grade(95), "A")
    checkS("when-expr no subject C", grade(72), "C")
    checkS("when-expr no subject F", grade(50), "F")

    // when as an EXPRESSION, with subject, comma values and else
    val day = 6
    val kind = when (day) {
        1, 2, 3, 4, 5 -> "week"
        6, 7 -> "end"
        else -> "?"
    }
    checkS("when-expr subject", kind, "end")

    // when as a STATEMENT (value discarded), subject form, mutating a var
    var bucket = 0
    when (day) {
        6, 7 -> bucket = 2
        else -> bucket = 1
    }
    check("when-stmt subject", bucket, 2)

    // when as a STATEMENT, subject-less
    var sign = 0
    val probe = -4
    when {
        probe > 0 -> sign = 1
        probe < 0 -> sign = -1
        else -> sign = 0
    }
    check("when-stmt no subject", sign, -1)

    // trailing-lambda higher-order calls and chaining
    val nums = listOf(3, 1, 4, 1, 5, 9, 2, 6)
    check("map+sumOf chain", nums.map { it + 1 }.sumOf { it }, 39)
    check("filter size", nums.filter { it > 4 }.size, 3)
    var acc = 0
    nums.forEach { acc = acc + it }
    check("forEach closure write", acc, 31)

    // multi-statement lambda body (local decls then a tail expression)
    val transform = { x: Int ->
        val a = x + 1
        val b = a * a
        b - 1
    }
    check("multi-stmt lambda", transform(4), 24)

    // multi-statement lambda whose tail is a when-expression (value preserved)
    val classify = { x: Int ->
        val doubled = x * 2
        when {
            doubled > 10 -> 1
            doubled > 4 -> 0
            else -> -1
        }
    }
    check("multi-stmt lambda when tail hi", classify(7), 1)
    check("multi-stmt lambda when tail mid", classify(3), 0)
    check("multi-stmt lambda when tail lo", classify(1), -1)

    // multi-statement lambda as a higher-order argument
    val mapped = nums.map { it ->
        val sq = it * it
        sq + it
    }
    check("multi-stmt lambda as arg", mapped[2], 20)

    if (fails == 0) { println("Kotlin completion self test passed") }
    exitProcess(fails)
}
