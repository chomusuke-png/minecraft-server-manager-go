package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureWhitelistCreatesTemplate(t *testing.T) {
	dir := t.TempDir()

	if err := ensureWhitelist(dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(whitelistPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != whitelistTemplate {
		t.Errorf("contenido inesperado: %q", content)
	}
}

func TestEnsureWhitelistDoesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, whitelistPath(dir), "mimod.jar\n")

	if err := ensureWhitelist(dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(whitelistPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "mimod.jar\n" {
		t.Errorf("se pisó un whitelist existente: %q", content)
	}
}

func TestLoadWhitelist(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, whitelistPath(dir), "# comentario\n\nMiMod.jar\nOtroMod\n")

	whitelist := loadWhitelist(dir)

	for _, name := range []string{"mimod.jar", "MIMOD", "otromod.jar", "OtroMod"} {
		if !whitelist[normalizeModName(name)] {
			t.Errorf("%q debería estar en la whitelist", name)
		}
	}
	if whitelist["comentario"] {
		t.Error("la línea de comentario no debería quedar en la whitelist")
	}
}

func TestLoadWhitelistMissingFile(t *testing.T) {
	whitelist := loadWhitelist(t.TempDir())
	if len(whitelist) != 0 {
		t.Errorf("se esperaba whitelist vacía, got %v", whitelist)
	}
}

func TestDisableClientModsRespectsWhitelist(t *testing.T) {
	dir := t.TempDir()
	modsDir := filepath.Join(dir, "mods")

	writeFile(t, whitelistPath(dir), "clientmod.jar\n")

	writeJar(t, filepath.Join(modsDir, "clientmod.jar"), "META-INF/mods.toml", `
[[mods]]
modId="clientmod"

[[dependencies.clientmod]]
modId="clientmod"
side="CLIENT"
`)
	writeJar(t, filepath.Join(modsDir, "othermod.jar"), "META-INF/mods.toml", `
[[mods]]
modId="othermod"

[[dependencies.othermod]]
modId="othermod"
side="CLIENT"
`)

	DisableClientMods(dir)

	if !fileExists(filepath.Join(modsDir, "clientmod.jar")) {
		t.Error("clientmod.jar está en la whitelist, no debería haberse deshabilitado")
	}
	if !fileExists(filepath.Join(modsDir, "othermod.jar.disabled")) {
		t.Error("othermod.jar no está en la whitelist, debería haberse deshabilitado")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
