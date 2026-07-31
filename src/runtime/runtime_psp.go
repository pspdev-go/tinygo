//go:build baremetal && mipsle && psp

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

func putchar(c byte) {
	runtime_putchar(c)
}

func abort() {
	for {
	}
}

func exit(code int) {
	_ = code
	abort()
}

func ticks() timeUnit {
	return 0
}

func ticksToNanoseconds(t timeUnit) int64 {
	return int64(t)
}

func nanosecondsToTicks(ns int64) timeUnit {
	return timeUnit(ns)
}

func sleepTicks(d timeUnit) {
	_ = d
}
