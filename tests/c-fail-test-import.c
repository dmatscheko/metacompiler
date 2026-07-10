/* Import resolution fail-test.
 * <stdio.h> is a system header the runtime provides, so it resolves; but
 * "mylib.h" is an external library header the metacompiler cannot resolve.
 * The default run must ABORT with a clean "unresolved import" message, while
 * -warn-imports must warn and run main() to a normal exit 0. **/

#include <stdio.h>
#include "mylib.h"

int main(void) {
    return 0;
}
