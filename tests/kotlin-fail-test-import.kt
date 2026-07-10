/* Kotlin unresolvable import. 'com.example.widget' is an external library the
 * runtime does not provide, so by default the compiler stops with a clean
 * "unresolved import" error (this file SHOULD fail without flags). With
 * -warn-imports the import is downgraded to a warning and ignored; the program
 * then compiles and runs (it never uses the imported symbol) and exits 0. **/

import com.example.widget.Widget

fun main() {
    println("ran without the widget")
    exitProcess(0)
}
