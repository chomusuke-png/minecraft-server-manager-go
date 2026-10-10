package mods

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"minecraft-manager/internal/logx"
)

func ensureModList(path string, template string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(template), 0644)
}

func loadModList(path string) map[string]bool {
	list := map[string]bool{}

	file, err := os.Open(path)
	if err != nil {
		return list
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		list[normalizeModName(line)] = true
	}
	if err := scanner.Err(); err != nil {
		logx.Warn("Error leyendo %s: %v", filepath.Base(path), err)
	}

	return list
}

func normalizeModName(name string) string {
	return strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
}
