// The exception-path allocation benchmark of docs/runtime-rework-plan.md,
// phase 4a. Every js_try used to malloc(512) for a jmp_buf per ENTRY - three of
// them for a try/catch/finally - and nothing ever frees it, so this loop leaked
// about 1.5 KB per iteration on top of the ordinary arena growth. The buffer is
// now pooled per DEPTH.
//
// The answer is deterministic, so ./test.sh can compare the two engines on it;
// tests/bench-alloc.sh runs it natively and reports time and peak RSS.
function main() {
	var s = 0
	var i = 0
	while (i < 20000) {
		try {
			if (i % 3 == 0) { throw i }
			s = s + 1
		} catch (e) {
			s = s + 2
		} finally {
			s = s + 1
		}
		i = i + 1
	}
	println(s)
	return 0
}
