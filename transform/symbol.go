package transform

import (
	"github.com/tinygo-org/tinygo/internal/asmname"
	"tinygo.org/x/go-llvm"
)

// asmSafeName matches the compiler-side symbol mangling for external assemblers.
func asmSafeName(name string) string {
	return asmname.Mangle(name)
}

func maybeAsmSafeName(name string) string {
	switch name {
	case "tinygo_psp_start", "runtime_putchar":
		return name
	default:
		return asmSafeName(name)
	}
}

// namedFunction looks up a TinyGo/Go function by its canonical name, accepting
// either the raw or asm-safe symbol spelling in the module.
func namedFunction(mod llvm.Module, name string) llvm.Value {
	fn := mod.NamedFunction(name)
	if !fn.IsNil() {
		return fn
	}
	return mod.NamedFunction(maybeAsmSafeName(name))
}
