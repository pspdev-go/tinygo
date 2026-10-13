//go:build gc.psp && baremetal && mipsle && psp && !scheduler.tasks

package runtime

func pspGCStackBounds() (activeStackTop, systemStackBottom uintptr) {
	return stackTop, 0
}
