/* coro-switch.S - the stack switch under the C floor's generators (runtime.c,
 * "coroutines"). The one part of the floor that is not C: saving the callee-saved
 * registers and moving the stack pointer has no spelling in the C subset.
 *
 *   void mec_coro_switch(long *save, long to)
 *       push the callee-saved registers, *save = sp, sp = to, pop them, return -
 *       which returns into whichever mec_coro_switch or mec_coro_boot stored `to`.
 *   void mec_coro_boot(long *save, long top, long cb)
 *       push and save as above, then call coro_entry(cb) with sp = top. It does
 *       not return here: coro_entry ends by switching back for the last time.
 *
 * Native only - llvm.Run runs the Go twin's goroutines - and linked next to
 * lib/runtime.ll by every -exe build (abnf/llvmmap.go's linkInputs). */

#if defined(__APPLE__)
#define SYM(x) _##x
#else
#define SYM(x) x
#endif

	.text
	.globl SYM(mec_coro_switch)
	.globl SYM(mec_coro_boot)

#if defined(__aarch64__)

/* AAPCS64: x19-x28, x29 (fp), x30 (lr) and the low halves of v8-v15. */

	.p2align 2
SYM(mec_coro_switch):
	sub sp, sp, #160
	stp x19, x20, [sp, #0]
	stp x21, x22, [sp, #16]
	stp x23, x24, [sp, #32]
	stp x25, x26, [sp, #48]
	stp x27, x28, [sp, #64]
	stp x29, x30, [sp, #80]
	stp d8, d9, [sp, #96]
	stp d10, d11, [sp, #112]
	stp d12, d13, [sp, #128]
	stp d14, d15, [sp, #144]
	mov x9, sp
	str x9, [x0]
	mov sp, x1
	ldp x19, x20, [sp, #0]
	ldp x21, x22, [sp, #16]
	ldp x23, x24, [sp, #32]
	ldp x25, x26, [sp, #48]
	ldp x27, x28, [sp, #64]
	ldp x29, x30, [sp, #80]
	ldp d8, d9, [sp, #96]
	ldp d10, d11, [sp, #112]
	ldp d12, d13, [sp, #128]
	ldp d14, d15, [sp, #144]
	add sp, sp, #160
	ret

	.p2align 2
SYM(mec_coro_boot):
	sub sp, sp, #160
	stp x19, x20, [sp, #0]
	stp x21, x22, [sp, #16]
	stp x23, x24, [sp, #32]
	stp x25, x26, [sp, #48]
	stp x27, x28, [sp, #64]
	stp x29, x30, [sp, #80]
	stp d8, d9, [sp, #96]
	stp d10, d11, [sp, #112]
	stp d12, d13, [sp, #128]
	stp d14, d15, [sp, #144]
	mov x9, sp
	str x9, [x0]
	mov sp, x1
	mov x29, #0
	mov x0, x2
	bl SYM(coro_entry)
	brk #0

#elif defined(__x86_64__)

/* System V and darwin: rbx, rbp, r12-r15. */
	.p2align 4
SYM(mec_coro_switch):
	pushq %rbp
	pushq %rbx
	pushq %r12
	pushq %r13
	pushq %r14
	pushq %r15
	movq %rsp, (%rdi)
	movq %rsi, %rsp
	popq %r15
	popq %r14
	popq %r13
	popq %r12
	popq %rbx
	popq %rbp
	ret

	.p2align 4
SYM(mec_coro_boot):
	pushq %rbp
	pushq %rbx
	pushq %r12
	pushq %r13
	pushq %r14
	pushq %r15
	movq %rsp, (%rdi)
	movq %rsi, %rsp
	xorl %ebp, %ebp
	movq %rdx, %rdi
	callq SYM(coro_entry)
	ud2

#else
#error "coro-switch.S: no stack switch for this architecture"
#endif

#if defined(__linux__)
	.section .note.GNU-stack,"",@progbits
#endif
