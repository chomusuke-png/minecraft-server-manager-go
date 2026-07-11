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
	"minecraft-manager/internal/logx"
)

func ensureServerJar(reader *bufio.Reader, dir string, cfg *config.Config, dl *downloader.Downloader) bool {
	jarPath := filepath.Join(dir, cfg.JarName)

	if fileExists(jarPath) {
		return true
	}

	logx.Warn("No se encontró '%s' en '%s'.", cfg.JarName, dir)

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
	meta.LoaderVersion = result.LoaderVersion

	if err := instance.SaveMeta(dir, *meta); err != nil {
		logx.Warn("Advertencia: no se pudo guardar instance.json: %v", err)
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
		logx.Warn("No se pudo limpiar instancia incompleta: %v", err)
		return
	}
	logx.Info("Instancia incompleta eliminada.")
}

func ensurePlayit(reader *bufio.Reader, cfg *config.Config, dl *downloader.Downloader) {
	if fileExists(cfg.PlayitPath) {
		return
	}

	logx.Warn("No se encontró '%s'.", cfg.PlayitPath)
	if askYesNo(reader, "[?] ¿Deseas descargar Playit.gg automáticamente?") {
		if err := dl.DownloadPlayit(cfg.PlayitPath); err != nil {
			logx.Error("Error descargando Playit: %v", err)
		}
	} else {
		logx.Warn("Continuando en modo LAN (sin túnel).")
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
			logx.Error("\nNo se pudo leer la respuesta, se asume 'no'.")
			return false
		}

		logx.Error("Entrada incorrecta, reintente.")
	}
}
