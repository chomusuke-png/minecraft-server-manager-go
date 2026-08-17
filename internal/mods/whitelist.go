package mods

import (
	"path/filepath"
)

const WhitelistFileName = "mods_whitelist.txt"

const whitelistTemplate = `# Mods que nunca se deshabilitan aunque se detecten como exclusivos de cliente.
# Un nombre de archivo .jar por línea (con o sin extensión, sin importar mayúsculas).
# Las líneas vacías o que empiezan con # se ignoran.
`

func ensureWhitelist(instanceDir string) error {
	return ensureModList(whitelistPath(instanceDir), whitelistTemplate)
}

func whitelistPath(instanceDir string) string {
	return filepath.Join(instanceDir, WhitelistFileName)
}

func loadWhitelist(instanceDir string) map[string]bool {
	return loadModList(whitelistPath(instanceDir))
}
