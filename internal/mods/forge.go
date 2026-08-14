package mods

import (
	"archive/zip"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
)

type forgeDependency struct {
	ModID string `toml:"modId"`
	Side  string `toml:"side"`
}

type forgeModsToml struct {
	Mods []struct {
		ModID string `toml:"modId"`
	} `toml:"mods"`
	Dependencies map[string][]forgeDependency `toml:"dependencies"`
}

func getForgeModEnvironment(jarPath string) (string, error) {
	zipReader, err := zip.OpenReader(jarPath)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	for _, name := range []string{"META-INF/neoforge.mods.toml", "META-INF/mods.toml"} {
		for _, zipEntry := range zipReader.File {
			if zipEntry.Name != name {
				continue
			}
			return parseForgeModsToml(zipEntry)
		}
	}

	return "", fmt.Errorf("mods.toml no encontrado")
}

func parseForgeModsToml(zipEntry *zip.File) (string, error) {
	entryReader, err := zipEntry.Open()
	if err != nil {
		return "", err
	}
	defer entryReader.Close()

	content, err := io.ReadAll(entryReader)
	if err != nil {
		return "", err
	}

	var meta forgeModsToml
	if err := toml.Unmarshal(content, &meta); err != nil {
		return "", err
	}

	if isForgeModClientOnly(meta) {
		return "client", nil
	}
	return "*", nil
}

func isForgeModClientOnly(meta forgeModsToml) bool {
	if len(meta.Mods) == 0 {
		return false
	}

	for _, mod := range meta.Mods {
		if !selfDeclaresClientOnly(meta.Dependencies[mod.ModID], mod.ModID) {
			return false
		}
	}
	return true
}

func selfDeclaresClientOnly(deps []forgeDependency, modID string) bool {
	for _, dep := range deps {
		if dep.ModID == modID && dep.Side == "CLIENT" {
			return true
		}
	}
	return false
}
