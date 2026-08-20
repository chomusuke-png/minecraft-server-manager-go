package downloader

import (
	"bufio"
	"errors"
	"slices"
	"strings"
	"testing"
)

func readerFor(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

func TestPaperVersionsFrom(t *testing.T) {
	// la v3 devuelve de la mas nueva a la mas vieja
	builds := []PaperBuild{
		{ID: 102, Channel: "ALPHA"},
		{ID: 101, Channel: "STABLE"},
		{ID: 100, Channel: "STABLE"},
	}

	versions, err := paperVersionsFrom(builds, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if versions.latest != "102" {
		t.Errorf("latest = %q, want 102", versions.latest)
	}
	if versions.stable != "101" {
		t.Errorf("stable = %q, want 101", versions.stable)
	}
	if !slices.Equal(versions.known, []string{"102", "101", "100"}) {
		t.Errorf("known = %v", versions.known)
	}
}

func TestPaperVersionsFromSinBuilds(t *testing.T) {
	if _, err := paperVersionsFrom(nil, "1.20.1"); err == nil {
		t.Error("una versión sin builds debería dar error")
	}
}

func TestFabricVersionsFrom(t *testing.T) {
	loaders := []FabricLoader{
		{Version: "0.17.0", Stable: false},
		{Version: "0.16.14", Stable: true},
		{Version: "0.16.13", Stable: true},
	}

	versions, err := fabricVersionsFrom(loaders)
	if err != nil {
		t.Fatal(err)
	}
	if versions.latest != "0.17.0" {
		t.Errorf("latest = %q, want 0.17.0", versions.latest)
	}
	if versions.stable != "0.16.14" {
		t.Errorf("stable = %q, want 0.16.14", versions.stable)
	}
}

func TestForgeVersionsFrom(t *testing.T) {
	promos := ForgePromotions{Promos: map[string]string{
		"1.20.1-latest":      "47.4.0",
		"1.20.1-recommended": "47.3.0",
	}}

	versions, err := forgeVersionsFrom(promos, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if versions.latest != "47.4.0" || versions.stable != "47.3.0" {
		t.Errorf("got (%q, %q)", versions.latest, versions.stable)
	}
	if versions.known != nil {
		t.Errorf("Forge no publica el listado completo, known debería quedar vacío: %v", versions.known)
	}

	if _, err := forgeVersionsFrom(promos, "1.7.10"); err == nil {
		t.Error("una versión sin promos debería dar error")
	}
}

func TestNeoForgeVersionsFrom(t *testing.T) {
	published := []string{
		"20.4.100",
		"21.1.100",
		"21.1.101",
		"21.1.102-beta",
		"21.2.5",
	}

	versions, err := neoForgeVersionsFrom(published, "21.1.", "1.21.1")
	if err != nil {
		t.Fatal(err)
	}
	if versions.latest != "21.1.102-beta" {
		t.Errorf("latest = %q, want 21.1.102-beta", versions.latest)
	}
	if versions.stable != "21.1.101" {
		t.Errorf("stable = %q, want 21.1.101", versions.stable)
	}
	if !slices.Equal(versions.known, []string{"21.1.100", "21.1.101", "21.1.102-beta"}) {
		t.Errorf("known = %v", versions.known)
	}

	if _, err := neoForgeVersionsFrom(published, "19.9.", "1.19.9"); err == nil {
		t.Error("un prefijo sin versiones debería dar error")
	}
}

func TestVersionChoicesSeparaLatestDeStable(t *testing.T) {
	choices := versionChoices(loaderVersions{latest: "47.4.0", stable: "47.3.0"}, "")

	if len(choices) != 2 {
		t.Fatalf("se esperaban 2 opciones, got %d", len(choices))
	}
	if choices[0].value != "47.4.0" || choices[1].value != "47.3.0" {
		t.Errorf("orden inesperado: %v", choices)
	}
}

func TestVersionChoicesAgrupaLaMismaVersion(t *testing.T) {
	choices := versionChoices(loaderVersions{latest: "47.4.0", stable: "47.4.0"}, "47.4.0")

	if len(choices) != 1 {
		t.Fatalf("una sola versión debería dar una sola opción, got %v", choices)
	}
	if got := choices[0].label(); got != "47.4.0 — la actual, la más reciente y la estable" {
		t.Errorf("label = %q", got)
	}
}

func TestVersionChoicesSaltaLosVacios(t *testing.T) {
	choices := versionChoices(loaderVersions{latest: "0.17.0"}, "")

	if len(choices) != 1 || choices[0].label() != "0.17.0 — la más reciente" {
		t.Errorf("choices = %v", choices)
	}
}

func TestPromptLoaderVersionPorDefectoUsaLaPrimera(t *testing.T) {
	available := loaderVersions{latest: "47.4.0", stable: "47.3.0"}

	got, err := promptLoaderVersion(readerFor("\n"), "Forge", available, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "47.4.0" {
		t.Errorf("got %q, want 47.4.0", got)
	}
}

func TestPromptLoaderVersionMantieneLaActual(t *testing.T) {
	available := loaderVersions{latest: "47.4.0", stable: "47.3.0"}

	got, err := promptLoaderVersion(readerFor("\n"), "Forge", available, "47.2.20")
	if err != nil {
		t.Fatal(err)
	}
	if got != "47.2.20" {
		t.Errorf("got %q, want 47.2.20", got)
	}
}

func TestPromptLoaderVersionEligeLaEstable(t *testing.T) {
	available := loaderVersions{latest: "47.4.0", stable: "47.3.0"}

	got, err := promptLoaderVersion(readerFor("2\n"), "Forge", available, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "47.3.0" {
		t.Errorf("got %q, want 47.3.0", got)
	}
}

func TestPromptLoaderVersionEscritaAMano(t *testing.T) {
	available := loaderVersions{latest: "102", stable: "101", known: []string{"100", "101", "102"}}

	// opción 3 = escribir; la primera respuesta no está publicada y se repregunta
	got, err := promptLoaderVersion(readerFor("3\n999\n100\n"), "Paper", available, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "100" {
		t.Errorf("got %q, want 100", got)
	}
}

func TestPromptLoaderVersionEscritaSinListado(t *testing.T) {
	available := loaderVersions{latest: "47.4.0", stable: "47.3.0"}

	got, err := promptLoaderVersion(readerFor("3\n47.1.99\n"), "Forge", available, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "47.1.99" {
		t.Errorf("got %q, want 47.1.99", got)
	}
}

func TestPromptLoaderVersionCancela(t *testing.T) {
	available := loaderVersions{latest: "47.4.0", stable: "47.3.0"}

	if _, err := promptLoaderVersion(readerFor("4\n"), "Forge", available, ""); !errors.Is(err, ErrCancelled) {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

func TestChooseLoaderVersionVanillaNoPregunta(t *testing.T) {
	got, err := ChooseLoaderVersion(readerFor(""), "vanilla", "1.20.1", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q, want vacío", got)
	}
}

func TestChooseLoaderVersionLoaderDesconocido(t *testing.T) {
	if _, err := ChooseLoaderVersion(readerFor(""), "spigot", "1.20.1", ""); err == nil {
		t.Error("un loader desconocido debería dar error")
	}
}
