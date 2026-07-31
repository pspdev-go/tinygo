//go:build baremetal && mipsle && psp && scheduler.none

package runtime

import (
	"unsafe"
	_ "unsafe"
)

//go:linkname mainMain main.main
func mainMain()

//export tinygo_psp_start
func tinygo_psp_start() {
	var stackMarker uintptr
	stackTop = uintptr(unsafe.Pointer(&stackMarker))

	initHeap()
	initAll()
	mainMain()
	exit(0)
}
