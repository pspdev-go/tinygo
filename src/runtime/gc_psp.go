//go:build gc.psp && baremetal && mipsle && psp

package runtime

import "unsafe"

const (
	pspHeapSize    = 2 * 1024 * 1024
	gcAlign        = 8
	maxGCBlocks    = 8192
	gcBlockReserve = 64
)

const needsStaticHeap = false

type gcBlock struct {
	start  uintptr
	size   uintptr
	layout unsafe.Pointer
	mark   bool
	free   bool
}

var (
	pspHeap       [pspHeapSize]byte
	pspHeapOffset uintptr

	gcBlocks     [maxGCBlocks]gcBlock
	gcMarkQueue  [maxGCBlocks]uint16
	gcMarkCount  int
	gcBlockCount int
	gcRunning    bool

	gcTotalAlloc uint64
	gcMallocs    uint64
	gcFrees      uint64
)

func initHeap() {
	pspHeapOffset = 0
	gcMarkCount = 0
	gcBlockCount = 0
	gcRunning = false
	gcTotalAlloc = 0
	gcMallocs = 0
	gcFrees = 0
}

func alloc(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	if size == 0 {
		return nil
	}
	if size > pspHeapSize {
		runtimePanic("out of memory")
		return nil
	}

	size = alignUp(size, gcAlign)

	if ptr := allocFromFreeBlock(size); ptr != nil {
		return ptr
	}

	// Important:
	// We can run out of metadata blocks before we run out of heap bytes.
	// If the side table is almost full, run GC before bump allocation.
	if !gcRunning && gcBlockCount >= maxGCBlocks-gcBlockReserve {
		GC()

		if ptr := allocFromFreeBlock(size); ptr != nil {
			return ptr
		}
	}

	if ptr := allocBump(size, layout); ptr != nil {
		return ptr
	}

	if !gcRunning {
		GC()

		if ptr := allocFromFreeBlock(size); ptr != nil {
			return ptr
		}

		if ptr := allocBump(size, layout); ptr != nil {
			return ptr
		}
	}

	runtimePanic("out of memory")
	return nil
}

func allocBump(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	if gcBlockCount >= maxGCBlocks {
		return nil
	}

	// Align the absolute address, not only the offset: a byte array's base is
	// not guaranteed to have the alignment required by Go objects or task stacks.
	heapBase := uintptr(unsafe.Pointer(&pspHeap[0]))
	offset := alignUp(heapBase+pspHeapOffset, gcAlign) - heapBase

	heapSize := uintptr(len(pspHeap))
	if size > heapSize || offset > heapSize-size {
		return nil
	}

	ptr := unsafe.Pointer(&pspHeap[offset])
	pspHeapOffset = offset + size

	registerBlock(uintptr(ptr), size, layout)
	gcTotalAlloc += uint64(size)
	gcMallocs++

	return ptr
}

func allocFromFreeBlock(size uintptr) unsafe.Pointer {
	for i := 0; i < gcBlockCount; i++ {
		b := &gcBlocks[i]

		if !b.free {
			continue
		}
		if b.size < size {
			continue
		}

		b.free = false
		b.mark = false

		ptr := unsafe.Pointer(b.start)
		memzero(ptr, b.size)
		gcTotalAlloc += uint64(size)
		gcMallocs++

		return ptr
	}

	return nil
}

func registerBlock(ptr uintptr, size uintptr, layout unsafe.Pointer) {
	if gcBlockCount >= maxGCBlocks {
		runtimePanic("too many heap blocks")
		return
	}

	gcBlocks[gcBlockCount] = gcBlock{
		start:  ptr,
		size:   size,
		layout: layout,
		mark:   false,
		free:   false,
	}
	gcBlockCount++
}

func realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	if ptr == nil {
		return alloc(size, nil)
	}

	if size == 0 {
		free(ptr)
		return nil
	}

	size = alignUp(size, gcAlign)

	old := findBlock(uintptr(ptr))
	if old != nil && old.size >= size {
		return ptr
	}

	newptr := alloc(size, nil)
	if newptr == nil {
		return nil
	}

	copySize := size
	if old != nil && old.size < copySize {
		copySize = old.size
	}

	memcpy(newptr, ptr, copySize)
	free(ptr)

	return newptr
}

func free(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}

	b := findExactBlock(uintptr(ptr))
	if b == nil || b.free {
		return
	}

	b.free = true
	b.mark = false
	gcFrees++
}

func calloc(size uintptr) unsafe.Pointer {
	ptr := alloc(size, nil)
	if ptr != nil {
		memzero(ptr, size)
	}
	return ptr
}

//go:noinline
func GC() {
	if gcRunning {
		return
	}

	gcRunning = true
	gcMarkCount = 0

	clearMarks()
	markRoots()
	drainMarkQueue()
	sweep()

	gcRunning = false
}

func clearMarks() {
	for i := 0; i < gcBlockCount; i++ {
		gcBlocks[i].mark = false
	}
}

//go:noinline
func markRoots() {
	var stackMarker uintptr

	stackBottom := uintptr(unsafe.Pointer(&stackMarker))
	activeStackTop, systemStackBottom := pspGCStackBounds()
	markRangeOrdered(stackBottom, activeStackTop)
	if systemStackBottom != 0 {
		markRangeOrdered(systemStackBottom, stackTop)
	}

	markGlobalRoots()
}

func markRangeOrdered(start, end uintptr) {
	if start > end {
		start, end = end, start
	}
	markRange(start, end)
}

type addressRange struct {
	start uintptr
	end   uintptr
}

func markGlobalRoots() {
	// The heap, block table, and mark queue are runtime bookkeeping. Scanning
	// them as globals both retains stale heap objects and makes every GC cycle
	// proportional to the full 2 MiB heap and metadata size.
	heapStart := uintptr(unsafe.Pointer(&pspHeap[0]))
	ranges := [3]addressRange{
		{heapStart, heapStart + uintptr(len(pspHeap))},
		{uintptr(unsafe.Pointer(&gcBlocks[0])), uintptr(unsafe.Pointer(&gcBlocks[0])) + unsafe.Sizeof(gcBlocks)},
		{uintptr(unsafe.Pointer(&gcMarkQueue[0])), uintptr(unsafe.Pointer(&gcMarkQueue[0])) + unsafe.Sizeof(gcMarkQueue)},
	}
	// Sort the three small exclusions by address so this also works if the
	// linker changes the order of zero-initialized globals.
	for i := 1; i < len(ranges); i++ {
		for j := i; j > 0 && ranges[j].start < ranges[j-1].start; j-- {
			ranges[j], ranges[j-1] = ranges[j-1], ranges[j]
		}
	}

	cursor := globalsStart
	for _, excluded := range ranges {
		if excluded.end <= cursor || excluded.start >= globalsEnd {
			continue
		}
		if excluded.start > cursor {
			markRange(cursor, excluded.start)
		}
		if excluded.end > cursor {
			cursor = excluded.end
		}
	}
	if cursor < globalsEnd {
		markRange(cursor, globalsEnd)
	}
}

func markRange(start, end uintptr) {
	start = alignUp(start, unsafe.Sizeof(uintptr(0)))
	for addr := start; addr+unsafe.Sizeof(uintptr(0)) <= end; addr += unsafe.Sizeof(uintptr(0)) {
		candidate := *(*uintptr)(unsafe.Pointer(addr))
		markPointer(candidate)
	}
}

func markPointer(candidate uintptr) {
	index := findBlockIndex(candidate)
	if index < 0 {
		return
	}

	b := &gcBlocks[index]
	if b.mark {
		return
	}
	b.mark = true
	gcMarkQueue[gcMarkCount] = uint16(index)
	gcMarkCount++
}

// drainMarkQueue scans each reachable block iteratively. Recursive marking
// could overflow the PSP's small stacks for a long chain of heap objects.
func drainMarkQueue() {
	for i := 0; i < gcMarkCount; i++ {
		b := &gcBlocks[gcMarkQueue[i]]
		start := b.start
		end := b.start + b.size
		for addr := start; addr+unsafe.Sizeof(uintptr(0)) <= end; addr += unsafe.Sizeof(uintptr(0)) {
			candidate := *(*uintptr)(unsafe.Pointer(addr))
			markPointer(candidate)
		}
	}
}

func sweep() {
	for i := 0; i < gcBlockCount; i++ {
		b := &gcBlocks[i]

		if b.free {
			continue
		}

		if b.mark {
			b.mark = false
			continue
		}

		b.free = true
		gcFrees++
	}
}

func ReadMemStats(m *MemStats) {
	var heapAlloc uint64
	var heapObjects uint64
	for i := 0; i < gcBlockCount; i++ {
		if !gcBlocks[i].free {
			heapAlloc += uint64(gcBlocks[i].size)
			heapObjects++
		}
	}

	m.HeapAlloc = heapAlloc
	m.HeapInuse = heapAlloc
	m.HeapIdle = uint64(len(pspHeap)) - heapAlloc
	m.HeapSys = uint64(len(pspHeap))
	m.GCSys = uint64(unsafe.Sizeof(gcBlocks))
	m.Sys = m.HeapSys + m.GCSys
	m.TotalAlloc = gcTotalAlloc
	m.Mallocs = gcMallocs
	m.Frees = gcFrees
	m.HeapObjects = heapObjects
	m.Alloc = heapAlloc
}

func SetFinalizer(obj interface{}, finalizer interface{}) {
	// Finalizers are not supported.
}

func setHeapEnd(newHeapEnd uintptr) {
	// The PSP heap has a fixed size.
}

func findBlock(ptr uintptr) *gcBlock {
	index := findBlockIndex(ptr)
	if index < 0 {
		return nil
	}
	return &gcBlocks[index]
}

// findBlockIndex finds the allocation containing ptr. Blocks are registered
// monotonically by the bump allocator and never moved, so their starts stay
// sorted even when a freed block is reused.
func findBlockIndex(ptr uintptr) int {
	low, high := 0, gcBlockCount
	for low < high {
		mid := low + (high-low)/2
		if gcBlocks[mid].start <= ptr {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low == 0 {
		return -1
	}
	index := low - 1
	b := &gcBlocks[index]
	if b.free || ptr >= b.start+b.size {
		return -1
	}
	return index
}

func findExactBlock(ptr uintptr) *gcBlock {
	low, high := 0, gcBlockCount
	for low < high {
		mid := low + (high-low)/2
		if gcBlocks[mid].start < ptr {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < gcBlockCount && gcBlocks[low].start == ptr {
		return &gcBlocks[low]
	}
	return nil
}

func alignUp(v, a uintptr) uintptr {
	return (v + a - 1) &^ (a - 1)
}
