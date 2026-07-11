package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/instance"
)

func ensureServerJar(reader *bufio.Reader, dir string, cfg *config.Config, dl *downloader.Downloader) bool {
	jarPath := filepath.Join(dir, cfg.JarName)

	if fileExists(jarPath) {
		return true
	}

	fmt.Printf("[!] No se encontró '%s' en '%s'.\n", cfg.JarName, dir)

	if !askYesNo(reader, "[?] ¿Descargar servidor automáticamente?") {
		cleanIncompleteInstance(dir)
		return false
	}

	result := dl.PromptUser(reader)
	if result == nil {
		cleanIncompleteInstance(dir)
		return false
	}

	meta, err := instance.LoadMeta(dir)
	if err != nil {
		meta = &instance.InstanceMeta{}
	}
	meta.LoaderType = result.LoaderType
	meta.MCVersion = result.MCVersion

	if err := instance.SaveMeta(dir, *meta); err != nil {
		fmt.Printf("[!] Advertencia: no se pudo guardar instance.json: %v\n", err)
	}

	return true
}

func cleanIncompleteInstance(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.Name() != "instance.json" {
			return
		}
	}

	if err := os.RemoveAll(dir); err != nil {
		fmt.Printf("[!] No se pudo limpiar instancia incompleta: %v\n", err)
		return
	}
	fmt.Println("[*] Instancia incompleta eliminada.")
}

func ensurePlayit(reader *bufio.Reader, cfg *config.Config, dl *downloader.Downloader) {
	if fileExists(cfg.PlayitPath) {
		return
	}

	fmt.Printf("[!] No se encontró '%s'.\n", cfg.PlayitPath)
	if askYesNo(reader, "[?] ¿Deseas descargar Playit.gg automáticamente?") {
		if err := dl.DownloadPlayit(cfg.PlayitPath); err != nil {
			fmt.Printf("[-] Error descargando Playit: %v\n", err)
		}
	} else {
		fmt.Println("[!] Continuando en modo LAN (sin túnel).")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func askYesNo(reader *bufio.Reader, question string) bool {
	for {
		fmt.Printf("%s (y/n): ", question)
		response, err := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "y", "s", "si", "yes":
			return true
		case "n", "no":
			return false
		}

		if err != nil {
			fmt.Println("\n[-] No se pudo leer la respuesta, se asume 'no'.")
			return false
		}

		fmt.Println("[-] Entrada incorrecta, reintente.")
	}
}
