package asmname

import "testing"

func TestMangleIsInjective(t *testing.T) {
	names := []string{
		"github.com/sago35/koebiten.init#1",
		"github.com/sago35/koebiten.init$1",
		"github.com/sago35/koebiten.init_231",
		"a.b/c",
		"a/b.c",
	}
	seen := make(map[string]string, len(names))
	for _, name := range names {
		mangled := Mangle(name)
		if previous, ok := seen[mangled]; ok {
			t.Errorf("Mangle(%q) = %q, same as Mangle(%q)", name, mangled, previous)
		}
		seen[mangled] = name
	}
}
