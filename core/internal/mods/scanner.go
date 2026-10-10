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

// QuiltModMetadata es la parte de quilt.mod.json que dice de que lado corre el
// mod. Quilt usa "dedicated_server" donde Fabric usa "server"
type QuiltModMetadata struct {
	Minecraft struct {
		Environment string `json:"environment"`
	} `json:"minecraft"`
}

func getModEnvironment(jarPath string) (string, error) {
	// quilt.mod.json va primero: un mod que trae los dos descriptores lo carga
	// Quilt con el suyo
	if env, err := getQuiltModEnvironment(jarPath); err == nil {
		return env, nil
	}
	if env, err := getFabricModEnvironment(jarPath); err == nil {
		return env, nil
	}
	return getForgeModEnvironment(jarPath)
}

func getQuiltModEnvironment(jarPath string) (string, error) {
	content, err := readJarEntry(jarPath, "quilt.mod.json")
	if err != nil {
		return "", err
	}

	var meta QuiltModMetadata
	if err := json.Unmarshal(content, &meta); err != nil {
		return "", err
	}

	if meta.Minecraft.Environment == "" {
		return "*", nil
	}
	return meta.Minecraft.Environment, nil
}

func readJarEntry(jarPath, name string) ([]byte, error) {
	zipReader, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()

	for _, zipEntry := range zipReader.File {
		if zipEntry.Name != name {
			continue
		}
		entryReader, err := zipEntry.Open()
		if err != nil {
			return nil, err
		}
		defer entryReader.Close()
		return io.ReadAll(entryReader)
	}
	return nil, fmt.Errorf("%s no encontrado", name)
}

func getFabricModEnvironment(jarPath string) (string, error) {
	content, err := readJarEntry(jarPath, "fabric.mod.json")
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
