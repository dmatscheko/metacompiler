// The smallest MetaJS program, used to prove the -exe LINK, not the semantics.
//
// Its module declares exactly ten js_* externs (js_setret, js_scope_new,
// js_str_mem, js_tdecl, js_num_i, js_closure, js_getret, js_scope_get,
// js_arr_new, js_call) and tests/metajs-link-stubrt.c defines all ten, so the
// binary links with nothing left over. Adding a statement here can add an
// extern and break that pairing on purpose - see tests/metajs-link-missing.js,
// which is the same program plus one operator the stand-in does not define.
function main() {
	return 7
}
