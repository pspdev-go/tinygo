//go:build baremetal && mipsle && psp && scheduler.tasks

package runtime

//export tinygo_psp_start
func tinygo_psp_start() {
	stackTop = uintptr(stacksave())
	run()
	exit(0)
}
