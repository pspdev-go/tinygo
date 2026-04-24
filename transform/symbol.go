package transform

import (
	"strings"

	"tinygo.org/x/go-llvm"
)

// asmSafeName matches the compiler-side symbol mangling for external assemblers.
func asmSafeName(name string) string {
	var b strings.Builder
	b.Grow(len(name) + 3)
	b.WriteString("tg_")

	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z':
			b.WriteByte(c)
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c)
		case c >= '0' && c <= '9':
			b.WriteByte(c)
		case c == '_':
			b.WriteByte(c)
		default:
			b.WriteByte('_')
		}
	}

	return b.String()
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
