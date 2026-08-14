package mods

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"minecraft-manager/internal/logx"
)

const WhitelistFileName = "mods_whitelist.txt"

const whitelistTemplate = `# Mods que nunca se deshabilitan aunque se detecten como exclusivos de cliente.
# Un nombre de archivo .jar por línea (con o sin extensión, sin importar mayúsculas).
# Las líneas vacías o que empiezan con # se ignoran.
`

func ensureWhitelist(instanceDir string) error {
	path := whitelistPath(instanceDir)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(whitelistTemplate), 0644)
}

func whitelistPath(instanceDir string) string {
	return filepath.Join(instanceDir, WhitelistFileName)
}

func loadWhitelist(instanceDir string) map[string]bool {
	whitelist := map[string]bool{}

	file, err := os.Open(whitelistPath(instanceDir))
	if err != nil {
		return whitelist
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		whitelist[normalizeModName(line)] = true
	}
	if err := scanner.Err(); err != nil {
		logx.Warn("Error leyendo %s: %v", WhitelistFileName, err)
	}

	return whitelist
}

func normalizeModName(name string) string {
	return strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
}
