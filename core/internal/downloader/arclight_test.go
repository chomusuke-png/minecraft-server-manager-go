package downloader

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func arclightEntry(name, lastModified string) ArclightEntry {
	parsed, err := time.Parse(time.RFC3339Nano, lastModified)
	if err != nil {
		panic(err)
	}
	return ArclightEntry{Name: name, LastModified: parsed, Permlink: "https://example.invalid/v1/objects/" + name}
}

func TestArclightVersionsFrom(t *testing.T) {
	stable := []ArclightEntry{
		arclightEntry("1.0.5-1a8925b1", "2024-03-14T20:13:22.756Z"),
		arclightEntry("1.0.4-aaaaaaa", "2024-01-10T10:00:00Z"),
	}
	// la snapshot es mas nueva que cualquier estable
	snapshot := []ArclightEntry{
		arclightEntry("1.0.6-SNAPSHOT-5dc8683", "2026-09-15T04:37:28.469Z"),
		arclightEntry("1.0.6-SNAPSHOT-6de9fec", "2026-05-07T12:22:44.712Z"),
	}

	versions, err := arclightVersionsFrom("forge", stable, snapshot, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if versions.latest != "forge-1.0.6-SNAPSHOT-5dc8683" {
		t.Errorf("latest = %q", versions.latest)
	}
	if versions.stable != "forge-1.0.5-1a8925b1" {
		t.Errorf("stable = %q", versions.stable)
	}
	want := []string{
		"forge-1.0.5-1a8925b1",
		"forge-1.0.4-aaaaaaa",
		"forge-1.0.6-SNAPSHOT-5dc8683",
		"forge-1.0.6-SNAPSHOT-6de9fec",
	}
	if !slices.Equal(versions.known, want) {
		t.Errorf("known = %v", versions.known)
	}
}

// las fechas traen una cantidad variable de decimales, y comparadas como texto
// .6Z quedaria despues de .61Z
func TestArclightVersionsFromComparaFechasNoTexto(t *testing.T) {
	snapshot := []ArclightEntry{
		arclightEntry("vieja", "2026-09-22T11:45:37.6Z"),
		arclightEntry("nueva", "2026-09-22T11:45:37.61Z"),
	}

	versions, err := arclightVersionsFrom("neoforge", nil, snapshot, "1.21.11")
	if err != nil {
		t.Fatal(err)
	}
	if versions.latest != "neoforge-nueva" {
		t.Errorf("latest = %q, want neoforge-nueva", versions.latest)
	}
	if versions.stable != "" {
		t.Errorf("sin estables, stable debería quedar vacío: %q", versions.stable)
	}
}

func TestArclightVersionsFromSinVersiones(t *testing.T) {
	if _, err := arclightVersionsFrom("fabric", nil, nil, "1.21.1"); err == nil {
		t.Error("sin versiones debería dar error")
	}
}

func TestArclightBasesFromRespetaElOrden(t *testing.T) {
	listing := ArclightListing{Files: []ArclightEntry{{Name: "fabric"}, {Name: "forge"}, {Name: "neoforge"}, {Name: "quilt"}}}

	bases, err := arclightBasesFrom(listing, "1.20.4")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(bases, []string{"neoforge", "forge", "fabric"}) {
		t.Errorf("bases = %v", bases)
	}

	if _, err := arclightBasesFrom(ArclightListing{}, "1.20.4"); err == nil {
		t.Error("sin bases debería dar error")
	}
}

func TestSplitArclightVersion(t *testing.T) {
	cases := []struct {
		value string
		base  string
		build string
		ok    bool
	}{
		{"neoforge-1.0.1-8ec9529", "neoforge", "1.0.1-8ec9529", true},
		{"forge-1.0.6-SNAPSHOT-5dc8683", "forge", "1.0.6-SNAPSHOT-5dc8683", true},
		{"1.0.1-8ec9529", "", "", false},
		{"neoforge-", "", "", false},
		{"neoforge", "", "", false},
	}

	for _, c := range cases {
		base, build, ok := splitArclightVersion(c.value)
		if base != c.base || build != c.build || ok != c.ok {
			t.Errorf("splitArclightVersion(%q) = (%q, %q, %v)", c.value, base, build, ok)
		}
	}
}

func TestPromptArclightBase(t *testing.T) {
	bases := []string{"neoforge", "forge", "fabric"}

	if got, _ := promptArclightBase(readerFor("\n"), bases, ""); got != "neoforge" {
		t.Errorf("por defecto = %q, want neoforge", got)
	}
	if got, _ := promptArclightBase(readerFor("\n"), bases, "fabric"); got != "fabric" {
		t.Errorf("con base actual = %q, want fabric", got)
	}
	if got, _ := promptArclightBase(readerFor("2\n"), bases, ""); got != "forge" {
		t.Errorf("opción 2 = %q, want forge", got)
	}
	if _, err := promptArclightBase(readerFor("4\n"), bases, ""); !errors.Is(err, ErrCancelled) {
		t.Errorf("cancelar = %v, want ErrCancelled", err)
	}
	// con una sola base no pregunta
	if got, _ := promptArclightBase(readerFor(""), []string{"forge"}, ""); got != "forge" {
		t.Errorf("una sola base = %q, want forge", got)
	}
}

func TestFindArclightEntry(t *testing.T) {
	stable := []ArclightEntry{arclightEntry("1.0.5", "2024-03-14T20:13:22Z")}
	snapshot := []ArclightEntry{arclightEntry("1.0.6-SNAPSHOT", "2026-09-15T04:37:28Z")}

	if _, ok := findArclightEntry("1.0.6-SNAPSHOT", stable, snapshot); !ok {
		t.Error("debería encontrar la snapshot")
	}
	if _, ok := findArclightEntry("9.9.9", stable, snapshot); ok {
		t.Error("no debería encontrar una versión que no existe")
	}
}
