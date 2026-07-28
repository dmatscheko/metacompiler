/* Kotlin unresolvable import. 'com.example.widget' is an external library the
 * runtime does not provide, so by default the compiler stops with a clean
 * "unresolved import" error (this file SHOULD fail without flags). With
 * -warn-imports the import is downgraded to a warning and ignored; the program
 * then compiles and runs and exits 0.
 *
 * It also pins what happens to a `super` whose SUPERTYPE came from such an
 * import. This is the commonest `super` in a real Android tree - every
 * Application / Activity / Service / ViewModel subclass has exactly this shape -
 * and it is a DIFFERENT class of failure from a supertype the grammar merely
 * failed to resolve (a typealias, a dotted path, an interface listed first;
 * those are bugs here and stay hard failures that NAME the supertype). There is
 * no such class anywhere, so `super.onCreate()` has nothing to dispatch to and
 * never will. Both halves therefore degrade it to Unit and let the override's
 * own body run, rather than aborting - the same warn-and-continue decision the
 * user already made by passing -warn-imports, which is the only way this state
 * is reachable at all. The property read and the property write degrade the same
 * way, and the assignment's right-hand side still runs.
 *
 * Before this was pinned the two halves disagreed on SEVERITY: the interpreter
 * ERRORED with "no super method 'onCreate'" and stopped, while the compiler
 * warned "super call outside a subclass not implemented (ignored)" and continued
 * - and only under -warn-unsupported, so the default was a hard stop there too. **/

import com.example.widget.Widget
import com.example.widget.Base as Aliased

class Sub : Widget() {
    var side = 0
    fun onCreate() {
        super.onCreate()          // no-op: Widget is outside the program
        side = 1
        println("override body ran, side=" + side)
    }
    fun readSuper(): String = "" + super.missingProp
    fun writeSuper(): Int {
        var seen = 0
        super.missingProp = seen + 9
        seen = 1
        return seen               // the right-hand side still ran
    }
}

// The `as` alias of an unresolvable import is external under BOTH names.
class Sub2 : Aliased() {
    fun poke(): String = "" + super.anything
}

fun main() {
    println("ran without the widget")
    val s = Sub()
    s.onCreate()
    println("read: " + s.readSuper())
    println("rhs ran: " + s.writeSuper())
    println("alias: " + Sub2().poke())
    exitProcess(0)
}
