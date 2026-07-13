/* Multi-file C test: vec_dot / vec_scale_x live in tests/imports/geomlib.h and are found
   via the -i include root (mec -i tests/imports ...). The included file is parsed with the
   same grammar; its functions register like the main file's. <stdio.h> is a builtin no-op
   include, mixed in on purpose. main() returns the failed-check count, so the run exits 0
   exactly when the imported functions resolve and compute correctly. */

#include <stdio.h>
#include "geomlib.h"

int main() {
    int nfail = 0;
    if (vec_dot(3, 4, 2, -1) != 2) { nfail = nfail + 1; }   /* 3*2 + 4*-1 = 2 */
    if (vec_scale_x(3, 2) != 6) { nfail = nfail + 1; }
    return nfail;
}
