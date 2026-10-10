package mods

import (
	"path/filepath"
	"testing"
)

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
