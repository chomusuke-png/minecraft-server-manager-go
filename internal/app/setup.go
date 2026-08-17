package app

import (
	"bufio"
	"os"
	"path/filepath"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/ngrok"
	"minecraft-manager/internal/playit"
	"minecraft-manager/internal/prompt"
	"minecraft-manager/internal/properties"
	"minecraft-manager/internal/selfupdate"
)

func checkForUpdates(reader *bufio.Reader, cfg *config.Config, version string) {
	if cfg.DisableUpdateCheck || version == "" || version == "dev" {
		return
	}

	rel, err := selfupdate.FetchLatest()
	if err != nil || !selfupdate.IsNewer(rel.Tag, version) {
		return
	}

	logx.Info("\nHay una versión nueva disponible: %s (tenés %s).", rel.Tag, version)
	if !prompt.YesNo(reader, "[?] ¿Actualizar ahora?") {
		return
	}

	logx.Info("Descargando %s...", rel.Tag)
	if err := selfupdate.Apply(rel); err != nil {
		logx.Error("No se pudo actualizar: %v", err)
		return
	}

	logx.Success("Actualizado a %s. Volvé a ejecutar la herramienta.", rel.Tag)
	os.Exit(0)
}

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

func ensureNgrok(reader *bufio.Reader, cfg *config.Config) bool {
	if fileExists(cfg.NgrokPath) {
		return true
	}

	logx.Warn("No se encontró '%s'.", cfg.NgrokPath)
	if !prompt.YesNo(reader, "[?] ¿Deseas descargar ngrok automáticamente?") {
		logx.Warn("Continuando en modo LAN (sin túnel).")
		return false
	}

	if err := ngrok.Download(cfg.NgrokPath); err != nil {
		logx.Error("Error descargando ngrok: %v", err)
		return false
	}
	return true
}

func setupTunnel(reader *bufio.Reader, cfg *config.Config, dl *downloader.Downloader, instanceDir string) func() {
	tunnelProvider := "playit"
	if meta, err := instance.LoadMeta(instanceDir); err == nil && meta.TunnelProvider != "" {
		tunnelProvider = meta.TunnelProvider
	}

	switch tunnelProvider {
	case "ngrok":
		port, ok := properties.ReadPort(instanceDir)
		if !ok {
			logx.Warn("No se pudo leer el puerto del servidor, no se inicia ngrok.")
			return func() {}
		}
		if !ensureNgrok(reader, cfg) {
			return func() {}
		}

		tunnel, err := ngrok.Start(instanceDir, cfg.NgrokPath, cfg.NgrokAuthToken, port)
		if err != nil {
			logx.Error("Error iniciando ngrok: %v", err)
			return func() {}
		}
		logx.Success("Túnel ngrok activo: %s", tunnel.PublicURL)
		return tunnel.Stop

	case "none":
		return func() {}

	default:
		ensurePlayit(reader, cfg, dl)
		if err := playit.Acquire(cfg); err != nil {
			logx.Error("Error iniciando Playit: %v", err)
		}
		return playit.Release
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
