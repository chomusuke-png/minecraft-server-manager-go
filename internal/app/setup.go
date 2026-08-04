package app

import (
	"bufio"
	"os"
	"path/filepath"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
)

func ensureServerJar(reader *bufio.Reader, dir string, cfg *config.Config, dl *downloader.Downloader) bool {
	jarPath := filepath.Join(dir, cfg.JarName)

	if fileExists(jarPath) {
		return true
	}

	// Forge >= 1.17 no deja ningún jar ejecutable: la instancia está completa si
	// tiene un comando de arranque persistido. Sin esto se re-preguntaría la
	// descarga en cada inicio.
	if meta, err := instance.LoadMeta(dir); err == nil && len(meta.LaunchArgs) > 0 {
		return true
	}

	logx.Warn("No se encontró '%s' en '%s'.", cfg.JarName, dir)

	if !prompt.YesNo(reader, "[?] ¿Descargar servidor automáticamente?") {
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
	meta.LaunchArgs = result.LaunchArgs
	meta.JavaPath = result.JavaPath

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
	if prompt.YesNo(reader, "[?] ¿Deseas descargar Playit.gg automáticamente?") {
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
