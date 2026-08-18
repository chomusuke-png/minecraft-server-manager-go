package properties

import "testing"

func TestIsLegacyLevelType(t *testing.T) {
	cases := []struct {
		mcVersion string
		want      bool
	}{
		{"1.18.2", true},
		{"1.18", true},
		{"1.19", false},
		{"1.19.4", false},
		{"1.20.1", false},
		{"1.16.5", true},
		{"1.8.9", true},
		{"2.0", false},
		{"", false},
		{"garbage", false},
	}

	for _, c := range cases {
		if got := isLegacyLevelType(c.mcVersion); got != c.want {
			t.Errorf("isLegacyLevelType(%q) = %v, want %v", c.mcVersion, got, c.want)
		}
	}
}

func TestLevelTypeFor(t *testing.T) {
	flat := worldType{"Plano", "minecraft:flat", "flat"}

	if got := levelTypeFor(flat, "1.18.2"); got != "flat" {
		t.Errorf("legacy: got %q, want %q", got, "flat")
	}
	if got := levelTypeFor(flat, "1.20.1"); got != "minecraft:flat" {
		t.Errorf("modern: got %q, want %q", got, "minecraft:flat")
	}
}

func TestPromptWorldType(t *testing.T) {
	if got := promptWorldType(readerFor("\n"), "1.20.1"); got != "minecraft:normal" {
		t.Errorf("got %q", got)
	}
	if got := promptWorldType(readerFor("2\n"), "1.16.5"); got != "flat" {
		t.Errorf("got %q", got)
	}
	if got := promptWorldType(readerFor("9\n3\n"), "1.20.1"); got != "minecraft:large_biomes" {
		t.Errorf("got %q", got)
	}
}

func TestEscapePropertyValue(t *testing.T) {
	if got := escapePropertyValue("minecraft:flat"); got != `minecraft\:flat` {
		t.Errorf("got %q", got)
	}
	if got := escapePropertyValue("sin_dos_puntos"); got != "sin_dos_puntos" {
		t.Errorf("got %q", got)
	}
}
