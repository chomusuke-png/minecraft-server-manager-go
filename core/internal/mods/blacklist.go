package mods

import (
	"path/filepath"
)

const BlacklistFileName = "mods_blacklist.txt"

const blacklistTemplate = `# Mods que siempre se deshabilitan, aunque no se detecten como exclusivos de cliente.
# Un nombre de archivo .jar por línea (con o sin extensión, sin importar mayúsculas).
# Las líneas vacías o que empiezan con # se ignoran.
# Si un mod está también en ` + WhitelistFileName + `, gana la whitelist y no se toca.
`

func ensureBlacklist(instanceDir string) error {
	return ensureModList(blacklistPath(instanceDir), blacklistTemplate)
}

func blacklistPath(instanceDir string) string {
	return filepath.Join(instanceDir, BlacklistFileName)
}

func loadBlacklist(instanceDir string) map[string]bool {
	return loadModList(blacklistPath(instanceDir))
}
