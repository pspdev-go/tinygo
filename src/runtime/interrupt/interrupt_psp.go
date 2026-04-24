//go:build baremetal && mipsle && psp

package interrupt

type State uintptr

func Disable() State {
	return 0
}

func Restore(state State) {
	_ = state
}

func In() bool {
	return false
}
