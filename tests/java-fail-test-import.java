/* Import resolution fail-test.
 * Mixes resolvable stdlib imports (java.util, java.lang via a static import) with
 * one clearly-external module the program never uses. By default the unresolved
 * import 'com.acme.external.Widget' must ABORT the run; with -warn-imports it must
 * warn and run to a clean exit 0. This is a SHOULD-FAIL entry by default. **/

package org.example.demo;

import java.util.List;
import static java.lang.Math.abs;
import com.acme.external.Widget;

public class Main {
    public static void main(String[] args) {
        int x = Math.abs(-5) + 5;       // uses only builtin Math, never Widget
        System.exit(x - 10);            // exits 0 when imports were tolerated
    }
}
