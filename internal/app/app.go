package app

import (
	"bufio"
	"os"
	"path/filepath"

	"minecraft-manager/internal/backup"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/eula"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/mods"
	"minecraft-manager/internal/properties"
	"minecraft-manager/internal/runner"
)

func Run(cfg *config.Config, version string) {
	reader := bufio.NewReader(os.Stdin)

	checkForUpdates(reader, cfg, version)

	selectedInstanceDir := runMenuLoop(reader, cfg)
	if selectedInstanceDir == "" {
		logx.Info("Operación cancelada.")
		return
	}

	logx.Info("\nTrabajando sobre instancia: %s", selectedInstanceDir)

	dl := downloader.New(selectedInstanceDir, cfg.JavaPath)

	if !ensureServerJar(reader, selectedInstanceDir, cfg, dl) {
		logx.Error("No se puede iniciar sin un archivo de servidor.")
		return
	}

	if err := properties.SetupInitialProperties(reader, selectedInstanceDir); err != nil {
		logx.Error("Error configurando propiedades: %v", err)
		return
	}

	stopTunnel := setupTunnel(reader, cfg, dl, selectedInstanceDir)
	defer stopTunnel()

	if err := eula.EnsureEulaAccepted(reader, selectedInstanceDir); err != nil {
		logx.Error("Error con el EULA: %v", err)
		return
	}

	mods.DisableClientMods(selectedInstanceDir)

	logx.Info("\nEjecutando tareas de mantenimiento...")
	instanceName := filepath.Base(selectedInstanceDir)
	keepMin := cfg.BackupKeepMin
	if meta, err := instance.LoadMeta(selectedInstanceDir); err == nil && meta.BackupKeepMin > 0 {
		keepMin = meta.BackupKeepMin
	}
	bm := backup.New(selectedInstanceDir, instanceName, cfg.BackupRetentionDays, keepMin)
	if err := bm.CreateBackup(); err != nil {
		logx.Error("Alerta de backup: %v", err)
	}

	svr := runner.New(cfg)
	svr.Start(selectedInstanceDir)
}
