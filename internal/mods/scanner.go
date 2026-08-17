package mods

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"minecraft-manager/internal/logx"
	"os"
	"path/filepath"
)

type FabricModMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

func DisableClientMods(serverDir string) {
	modsDir := filepath.Join(serverDir, "mods")

	if _, err := os.Stat(modsDir); os.IsNotExist(err) {
		return
	}

	if err := ensureWhitelist(serverDir); err != nil {
		logx.Warn("No se pudo crear %s: %v", WhitelistFileName, err)
	}
	whitelist := loadWhitelist(serverDir)

	if err := ensureBlacklist(serverDir); err != nil {
		logx.Warn("No se pudo crear %s: %v", BlacklistFileName, err)
	}
	blacklist := loadBlacklist(serverDir)

	files, err := os.ReadDir(modsDir)
	if err != nil {
		logx.Error("Error leyendo carpeta mods: %v", err)
		return
	}

	logx.Info("Escaneando mods incompatibles (Client-Side)...")
	count := 0

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".jar" {
			continue
		}

		if whitelist[normalizeModName(file.Name())] {
			logx.Detail("OMITIENDO: %s (protegido por %s)", file.Name(), WhitelistFileName)
			continue
		}

		modFilePath := filepath.Join(modsDir, file.Name())

		if blacklist[normalizeModName(file.Name())] {
			logx.Detail("DESHABILITANDO: %s (listado en %s)", file.Name(), BlacklistFileName)
			if disableMod(modFilePath) {
				count++
			}
			continue
		}

		modEnvironment, err := getModEnvironment(modFilePath)
		if err != nil {
			continue
		}

		if modEnvironment == "client" {
			logx.Detail("DESHABILITANDO: %s (Es solo de cliente)", file.Name())
			if disableMod(modFilePath) {
				count++
			}
		}
	}

	if count == 0 {
		logx.Detail("Todo limpio. No se encontraron mods exclusivos de cliente.")
	} else {
		logx.Detail("Se deshabilitaron %d mods incompatibles.", count)
	}
}

func disableMod(modFilePath string) bool {
	if err := os.Rename(modFilePath, modFilePath+".disabled"); err != nil {
		logx.Error("Error al deshabilitar: %v", err)
		return false
	}
	return true
}

func getModEnvironment(jarPath string) (string, error) {
	if env, err := getFabricModEnvironment(jarPath); err == nil {
		return env, nil
	}
	return getForgeModEnvironment(jarPath)
}

func getFabricModEnvironment(jarPath string) (string, error) {
	zipReader, err := zip.OpenReader(jarPath)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	for _, zipEntry := range zipReader.File {
		if zipEntry.Name == "fabric.mod.json" {
			entryReader, err := zipEntry.Open()
			if err != nil {
				return "", err
			}
			defer entryReader.Close()

			content, err := io.ReadAll(entryReader)
			if err != nil {
				return "", err
			}

			var meta FabricModMetadata
			if err := json.Unmarshal(content, &meta); err != nil {
				return "", err
			}

			if meta.Environment == "" {
				return "*", nil
			}

			return meta.Environment, nil
		}
	}

	return "", fmt.Errorf("fabric.mod.json no encontrado")
}
