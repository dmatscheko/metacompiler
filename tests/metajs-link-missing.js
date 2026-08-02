// metajs-link-hello.js plus one addition, so the module also declares js_add -
// which tests/metajs-link-stubrt.c deliberately does not define.
//
// This is the NEGATIVE half of the linking feature: the build declares a runtime,
// so the missing symbol is a link error naming js_add on stderr and a non-zero
// exit, NOT a zero-returning stub. Before the runtime existed there was nothing to
// link and stubbing was the only option; now a stub here would mean a compiled
// program whose additions silently answer 0.
function main() {
	return 7 + 1
}
