//go:build gc.psp && baremetal && mipsle && psp && scheduler.tasks

package runtime

import "internal/task"

// pspGCStackBounds returns the upper bound for the active task and the saved
// lower bound for the scheduler's system stack. They are separate stacks and
// must not be scanned as one address range.
func pspGCStackBounds() (activeStackTop, systemStackBottom uintptr) {
	activeStackTop = task.CurrentStackTop()
	if activeStackTop == 0 {
		return stackTop, 0
	}
	return activeStackTop, task.SystemStack()
}
