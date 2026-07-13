/* A project header imported by tests/c-test-multifile.c (via -i tests/imports).
   Header-only in this subset: the functions are DEFINED here, not just declared, so
   #including the header registers them exactly like the main file's own functions. */

int vec_dot(int ax, int ay, int bx, int by) {
    return ax * bx + ay * by;
}

int vec_scale_x(int x, int f) {
    return x * f;
}
