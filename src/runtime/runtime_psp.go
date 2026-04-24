//go:build baremetal && mipsle && psp

package runtime

import _ "unsafe"

//go:linkname mainMain main.main
func mainMain()

// tinygo_psp_start is the PSP entry hook called from bridge/main.c.
// Keep this function in package runtime so the exported symbol name stays simple.
//
//export tinygo_psp_start
func tinygo_psp_start() {
	// Initialize the allocator first.
	initHeap()

	// Run package initializers.
	initAll()

	// Enter the Go program.
	mainMain()

	// If main.main returns, stop here.
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
