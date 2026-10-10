package downloader

import (
	"errors"
	"slices"
	"testing"
)

func TestCompareMCVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int // signo
	}{
		{"1.21.11", "1.21.9", 1},
		{"1.21", "1.21.1", -1},
		{"26.1", "1.21.11", 1}, // la numeracion por año va despues de la clasica
		{"26.3", "26.1.2", 1},
		{"1.20.1", "1.20.1", 0},
	}

	for _, c := range cases {
		got := compareMCVersions(c.a, c.b)
		if (got > 0) != (c.want > 0) || (got < 0) != (c.want < 0) {
			t.Errorf("compareMCVersions(%q, %q) = %d, want signo %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSortedMCReleasesDescartaPrereleasesYOrdena(t *testing.T) {
	got := sortedMCReleases([]string{"1.21.9", "26.3-rc-3", "1.20.1", "26.3", "1.21.11", "25w14craftmine", "1.21.9", "26.1.2"})
	want := []string{"26.3", "26.1.2", "1.21.11", "1.21.9", "1.20.1"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPaperMCVersionsFrom(t *testing.T) {
	project := PaperProject{Versions: map[string][]string{
		"1.20": {"1.20.6", "1.20.1"},
		"26.3": {"26.3", "26.3-rc-3"},
		"1.21": {"1.21.11", "1.21.11-rc3", "1.21.10"},
	}}

	versions, err := paperMCVersionsFrom(project)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(versions.stable, []string{"26.3", "1.21.11", "1.21.10", "1.20.6", "1.20.1"}) {
		t.Errorf("stable = %v", versions.stable)
	}
	// una rc no se sugiere pero se puede escribir
	if !slices.Contains(versions.known, "26.3-rc-3") {
		t.Errorf("known debería incluir la rc: %v", versions.known)
	}
}

func TestFlaggedMCVersionsFrom(t *testing.T) {
	games := []FlaggedGameVersion{
		{Version: "26.4-snapshot-2", Stable: false},
		{Version: "26.3", Stable: true},
		{Version: "26.3-rc-3", Stable: false},
		{Version: "1.21.11", Stable: true},
	}

	versions, err := flaggedMCVersionsFrom(games, "Fabric")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(versions.stable, []string{"26.3", "1.21.11"}) {
		t.Errorf("stable = %v", versions.stable)
	}
	if len(versions.known) != 4 {
		t.Errorf("known = %v", versions.known)
	}

	if _, err := flaggedMCVersionsFrom(games[:1], "Fabric"); err == nil {
		t.Error("sin estables debería dar error")
	}
}

func TestForgeMCVersionsFrom(t *testing.T) {
	promos := ForgePromotions{Promos: map[string]string{
		"1.20.1-latest":      "47.4.0",
		"1.20.1-recommended": "47.3.0",
		"26.3-latest":        "63.0.1",
		"1.7.10_pre4-latest": "10.12.2.1149",
	}}

	versions, err := forgeMCVersionsFrom(promos)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(versions.stable, []string{"26.3", "1.20.1"}) {
		t.Errorf("stable = %v", versions.stable)
	}
}

func TestMCVersionFromNeoForge(t *testing.T) {
	cases := []struct {
		neoForge string
		want     string
		ok       bool
	}{
		{"21.1.77", "1.21.1", true},
		{"20.2.3-beta", "1.20.2", true},
		{"21.0.0-beta", "1.21", true},
		{"26.1.2.5", "26.1.2", true},
		{"26.3.0.38-beta", "26.3", true},
		{"0.25w14craftmine.3-beta", "", false},
		{"47.1.82", "", false}, // las de Forge que se publicaron al principio para 1.20.1 no siguen el esquema
	}

	for _, c := range cases {
		got, ok := mcVersionFromNeoForge(c.neoForge)
		if got != c.want || ok != c.ok {
			t.Errorf("mcVersionFromNeoForge(%q) = (%q, %v), want (%q, %v)", c.neoForge, got, ok, c.want, c.ok)
		}
	}
}

// tiene que ser el inverso de neoForgeVersionPrefix
func TestMCVersionFromNeoForgeCoincideConElPrefijo(t *testing.T) {
	for _, neoForge := range []string{"21.1.77", "20.4.100", "26.1.2.5", "26.3.0.38-beta"} {
		mcVersion, ok := mcVersionFromNeoForge(neoForge)
		if !ok {
			t.Fatalf("%s no se pudo traducir", neoForge)
		}
		prefix, ok := neoForgeVersionPrefix(mcVersion)
		if !ok || len(neoForge) < len(prefix) || neoForge[:len(prefix)] != prefix {
			t.Errorf("%s -> %s -> prefijo %q, que no coincide", neoForge, mcVersion, prefix)
		}
	}
}

func TestVanillaMCVersionsFrom(t *testing.T) {
	manifest := MojangManifest{Versions: []MojangVersion{
		{ID: "26.4-snapshot-2", Type: "snapshot"},
		{ID: "26.3", Type: "release"},
		{ID: "1.21.11", Type: "release"},
	}}

	versions, err := vanillaMCVersionsFrom(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(versions.stable, []string{"26.3", "1.21.11"}) {
		t.Errorf("stable = %v", versions.stable)
	}
	if !slices.Contains(versions.known, "26.4-snapshot-2") {
		t.Error("una snapshot se puede escribir aunque no se sugiera")
	}
}

var paperLike = mcVersions{
	stable: []string{"26.3", "26.2", "26.1.2", "26.1.1", "1.21.11", "1.21.10", "1.21.9", "1.20.1", "1.7.10"},
	known:  []string{"26.3", "26.3-rc-3", "26.2", "26.1.2", "26.1.1", "1.21.11", "1.21.10", "1.21.9", "1.20.1", "1.7.10"},
}

func TestPromptMCVersion(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		current string
		want    string
	}{
		{"Enter al crear toma la más reciente", "\n", "", "26.3"},
		{"Enter al actualizar mantiene la actual", "\n", "1.20.1", "1.20.1"},
		{"escrita", "1.21.10\n", "", "1.21.10"},
		{"una rc publicada se acepta", "26.3-rc-3\n", "", "26.3-rc-3"},
		{"rechaza lo que no existe y vuelve a preguntar", "1.21.12\n1.21.11\n", "", "1.21.11"},
		// sin la actual soportada, Enter no elige nada
		{"actual no soportada exige escribir", "\n1.21.11\n", "1.12.2", "1.21.11"},
	}

	for _, c := range cases {
		got, err := promptMCVersion(readerFor(c.input), "Paper", paperLike, c.current)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}

	if _, err := promptMCVersion(readerFor(""), "Paper", paperLike, "1.12.2"); !errors.Is(err, ErrCancelled) {
		t.Errorf("sin entrada y sin default = %v, want ErrCancelled", err)
	}
}

func TestMCVersionDefault(t *testing.T) {
	cases := []struct {
		current string
		want    string
	}{
		{"", "26.3"},
		{"1.20.1", "1.20.1"},
		{"26.3-rc-3", "26.3-rc-3"},
		{"1.12.2", ""},
	}
	for _, c := range cases {
		if got := mcVersionDefault(paperLike, c.current); got != c.want {
			t.Errorf("mcVersionDefault(%q) = %q, want %q", c.current, got, c.want)
		}
	}
}

func TestMCVersionRange(t *testing.T) {
	if got := mcVersionRange(paperLike); got != " (1.7.10 a 26.3)" {
		t.Errorf("got %q", got)
	}
	if got := mcVersionRange(mcVersions{stable: []string{"1.20.1"}}); got != " (sólo 1.20.1)" {
		t.Errorf("una sola = %q", got)
	}
	if got := mcVersionRange(mcVersions{}); got != "" {
		t.Errorf("sin estables = %q", got)
	}
}

func TestClosestMCVersions(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		// prefiere la misma linea aunque la 26.1.1 este al lado en la lista
		{"1.21.12", []string{"1.21.11", "1.21.10", "1.21.9"}},
		{"1.21.8", []string{"1.21.11", "1.21.10", "1.21.9"}},
		{"26.1.3", []string{"26.1.2", "26.1.1"}},
		{"1.20.2", []string{"1.20.1"}},
		// una linea que no existe usa la lista completa alrededor de donde caeria
		{"1.19.4", []string{"1.21.9", "1.20.1", "1.7.10"}},
		// sin formato de release se sugieren las mas recientes
		{"latest", []string{"26.3", "26.2", "26.1.2"}},
	}

	for _, c := range cases {
		if got := closestMCVersions(c.input, paperLike.stable, 3); !slices.Equal(got, c.want) {
			t.Errorf("closestMCVersions(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestUnsupportedMCVersionMessage(t *testing.T) {
	got := unsupportedMCVersionMessage("Paper", "1.21.12", paperLike)
	want := "Paper no publicó 1.21.12. Las más cercanas: 1.21.11, 1.21.10, 1.21.9"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTodosLosLoadersTienenResolverDeVersionesDeMinecraft(t *testing.T) {
	for _, loader := range Loaders {
		if _, ok := mcVersionResolvers[loader.Type]; !ok {
			t.Errorf("%s no tiene resolver de versiones de Minecraft", loader.Type)
		}
	}
}

func TestNeoForgeMCVersionsFromSinRepetidos(t *testing.T) {
	versions, err := neoForgeMCVersionsFrom([]string{"21.1.1", "21.1.2", "21.1.3-beta", "20.4.100"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(versions.stable, []string{"1.21.1", "1.20.4"}) {
		t.Errorf("stable = %v", versions.stable)
	}
	if len(versions.known) != 2 {
		t.Errorf("known tiene repetidos: %v", versions.known)
	}
}
