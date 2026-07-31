//go:build baremetal && mipsle && psp

package runtime

//export tinygo_psp_start
func tinygo_psp_start() {
	stackTop = uintptr(stacksave())
	run()
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

//go:linkname sceKernelGetSystemTimeWide sceKernelGetSystemTimeWide
func sceKernelGetSystemTimeWide() int64

//go:linkname sceKernelDelayThread sceKernelDelayThread
func sceKernelDelayThread(delay uint32) int32

func ticks() timeUnit {
	return timeUnit(sceKernelGetSystemTimeWide())
}

func ticksToNanoseconds(t timeUnit) int64 {
	return int64(t) * 1000
}

func nanosecondsToTicks(ns int64) timeUnit {
	return timeUnit(ns / 1000)
}

func sleepTicks(d timeUnit) {
	if d <= 0 {
		return
	}
	if d > timeUnit(^uint32(0)) {
		d = timeUnit(^uint32(0))
	}
	sceKernelDelayThread(uint32(d))
}
