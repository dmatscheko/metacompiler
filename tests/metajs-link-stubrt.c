/*
 * metajs-link-stubrt.c - a STAND-IN runtime for the -exe link, not a real one.
 *
 * WHAT IT IS FOR
 *
 * metajs-to-llvm-ir.abnf emits HANDLE IR: every value is an i64 handle and the
 * semantics live in js_* externs. Until the C floor of phase 3 exists there is
 * nothing to link those against, so this file defines exactly the ten symbols
 * that tests/metajs-link-hello.js declares - by hand, one line each - and a main()
 * that calls jsmain(). That is enough to prove the LINKING FEATURE end to end:
 *
 *     mec languages/metajs-to-llvm-ir.abnf tests/metajs-link-hello.js \
 *         -exe tests/metajs-link.out -rt tests/metajs-link-stubrt.c
 *
 * builds a binary, and the linker resolves every js_* the module declared.
 *
 * WHAT IT IS NOT
 *
 * It is not a runtime and must not grow into one - docs/runtime-rework-plan.md
 * phase 3 owns that, in C compiled by c-to-llvm-ir.abnf, not in clang-compiled C
 * checked in beside the tests. Nothing here allocates, dispatches or answers
 * anything: js_call cannot call the closure it is handed, because the emitted
 * module has no function table for it to index. The binary therefore builds and
 * runs and returns 0; it does NOT run the MetaJS program. Do not read a passing
 * link as a passing program.
 *
 * Its OTHER job is the negative case: delete a definition below (or use
 * tests/metajs-link-missing.js, which needs one this file does not define) and
 * the build must FAIL naming the symbol, instead of stubbing it to zero.
 */

typedef long h; /* an i64 handle */

extern h jsmain(h env, h args);

h js_num_i(h n) { return n; }
h js_str_mem(const char *p, h n) { return (h)p + n * 0; }
h js_obj_new(void) { return 0; }
h js_arr_new(void) { return 0; }
h js_scope_new(h parent) { return parent * 0; }
h js_scope_get(h scope, h name) { return scope * 0 + name * 0; }
h js_tdecl(h scope, h name, h val) { return scope * 0 + name * 0 + val * 0; }
h js_closure(h fn, h env) { return fn * 0 + env * 0; }
h js_call(h fn, h self, h args) { return fn * 0 + self * 0 + args * 0; }
h js_getret(void) { return 0; }
h js_setret(h v) { return v * 0; }

int main(void) {
	jsmain(0, 0);
	return 0;
}
