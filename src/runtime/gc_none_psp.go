//go:build gc.none && baremetal && mipsle && psp

package runtime

import "unsafe"

// A very small static heap for PSP bring-up.
// Replace this later with a better allocator once main.main reaches stability.
var pspHeap [1024 * 1024]byte
var pspHeapOffset uintptr

// initHeap initializes the bump allocator state.
func initHeap() {
	pspHeapOffset = 0
}

// alloc allocates memory from the static heap.
// This is a bump allocator intended only for early bring-up.
func alloc(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	_ = layout

	if size == 0 {
		return nil
	}

	// Align to 8 bytes.
	offset := (pspHeapOffset + 7) &^ 7
	if offset+size > uintptr(len(pspHeap)) {
		runtimePanic("out of memory")
		return nil
	}

	ptr := unsafe.Pointer(&pspHeap[offset])
	pspHeapOffset = offset + size
	return ptr
}

// realloc allocates a new block and copies the existing contents.
// This is only a minimal bring-up implementation.
func realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	if ptr == nil {
		return alloc(size, nil)
	}

	newptr := alloc(size, nil)
	if newptr == nil {
		return nil
	}

	memcpy(newptr, ptr, size)
	return newptr
}

// free is a no-op for the bump allocator.
func free(ptr unsafe.Pointer) {
	_ = ptr
}

// calloc allocates zeroed memory.
func calloc(size uintptr) unsafe.Pointer {
	ptr := alloc(size, nil)
	if ptr == nil {
		return nil
	}
	memzero(ptr, size)
	return ptr
}
