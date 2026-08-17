package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBlacklistCreatesTemplate(t *testing.T) {
	dir := t.TempDir()

	if err := ensureBlacklist(dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(blacklistPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != blacklistTemplate {
		t.Errorf("contenido inesperado: %q", content)
	}
}

func TestEnsureBlacklistDoesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, blacklistPath(dir), "mimod.jar\n")

	if err := ensureBlacklist(dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(blacklistPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "mimod.jar\n" {
		t.Errorf("se pisó un blacklist existente: %q", content)
	}
}

func TestLoadBlacklist(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, blacklistPath(dir), "# comentario\n\nMiMod.jar\nOtroMod\n")

	blacklist := loadBlacklist(dir)

	for _, name := range []string{"mimod.jar", "MIMOD", "otromod.jar", "OtroMod"} {
		if !blacklist[normalizeModName(name)] {
			t.Errorf("%q debería estar en la blacklist", name)
		}
	}
	if blacklist["comentario"] {
		t.Error("la línea de comentario no debería quedar en la blacklist")
	}
}

func TestLoadBlacklistMissingFile(t *testing.T) {
	blacklist := loadBlacklist(t.TempDir())
	if len(blacklist) != 0 {
		t.Errorf("se esperaba blacklist vacía, got %v", blacklist)
	}
}

func TestDisableClientModsAppliesBlacklist(t *testing.T) {
	dir := t.TempDir()
	modsDir := filepath.Join(dir, "mods")

	writeFile(t, blacklistPath(dir), "bloqueado\n")

	writeJar(t, filepath.Join(modsDir, "bloqueado.jar"), "META-INF/mods.toml", `
[[mods]]
modId="bloqueado"
`)
	writeJar(t, filepath.Join(modsDir, "permitido.jar"), "META-INF/mods.toml", `
[[mods]]
modId="permitido"
`)

	DisableClientMods(dir)

	if !fileExists(filepath.Join(modsDir, "bloqueado.jar.disabled")) {
		t.Error("bloqueado.jar está en la blacklist, debería haberse deshabilitado")
	}
	if !fileExists(filepath.Join(modsDir, "permitido.jar")) {
		t.Error("permitido.jar no es de cliente ni está en la blacklist, no debería tocarse")
	}
}

func TestWhitelistGanaSobreBlacklist(t *testing.T) {
	dir := t.TempDir()
	modsDir := filepath.Join(dir, "mods")

	writeFile(t, whitelistPath(dir), "ambos.jar\n")
	writeFile(t, blacklistPath(dir), "ambos.jar\n")

	writeJar(t, filepath.Join(modsDir, "ambos.jar"), "META-INF/mods.toml", `
[[mods]]
modId="ambos"
`)

	DisableClientMods(dir)

	if !fileExists(filepath.Join(modsDir, "ambos.jar")) {
		t.Error("ambos.jar está en las dos listas, debería ganar la whitelist")
	}
}
