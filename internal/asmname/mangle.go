// Package asmname contains symbol mangling shared by the compiler and the
// later stages that look up compiler-generated symbols.
package asmname

// Mangle returns a symbol name that is safe to use with external assemblers.
// The encoding is injective: every byte other than an ASCII letter or digit is
// encoded as an underscore followed by its two-digit hexadecimal value. In
// particular, this keeps SSA-generated names such as init#1 and init$1
// distinct.
func Mangle(name string) string {
	const hex = "0123456789abcdef"
	result := make([]byte, 0, 3+len(name)*3)
	result = append(result, "tg_"...)
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			result = append(result, c)
		} else {
			result = append(result, '_', hex[c>>4], hex[c&0xf])
		}
	}
	return string(result)
}
