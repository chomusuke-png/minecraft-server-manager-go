package mods

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
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

	files, err := os.ReadDir(modsDir)
	if err != nil {
		fmt.Printf("[-] Error leyendo carpeta mods: %v\n", err)
		return
	}

	fmt.Println("[*] Escaneando mods incompatibles (Client-Side)...")
	count := 0

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".jar" {
			continue
		}

		fullPath := filepath.Join(modsDir, file.Name())
		env, err := getModEnvironment(fullPath)
		if err != nil {
			continue
		}

		if env == "client" {
			fmt.Printf("    -> DESHABILITANDO: %s (Es solo de cliente)\n", file.Name())

			disabledPath := fullPath + ".disabled"
			if err := os.Rename(fullPath, disabledPath); err != nil {
				fmt.Printf("       [-] Error al deshabilitar: %v\n", err)
			} else {
				count++
			}
		}
	}

	if count == 0 {
		fmt.Println("    -> Todo limpio. No se encontraron mods exclusivos de cliente.")
	} else {
		fmt.Printf("    -> Se deshabilitaron %d mods incompatibles.\n", count)
	}
}

func getModEnvironment(jarPath string) (string, error) {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "fabric.mod.json" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
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
