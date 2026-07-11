package app

import (
	"bufio"
	"os"

	"minecraft-manager/internal/backup"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/eula"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/mods"
	"minecraft-manager/internal/playit"
	"minecraft-manager/internal/properties"
	"minecraft-manager/internal/runner"
)

func Run(cfg *config.Config) {
	reader := bufio.NewReader(os.Stdin)

	selectedInstanceDir := runMenuLoop(reader, cfg)
	if selectedInstanceDir == "" {
		logx.Info("Operación cancelada.")
		return
	}

	logx.Info("\nTrabajando sobre instancia: %s", selectedInstanceDir)

	dl := downloader.New(selectedInstanceDir)

	if !ensureServerJar(reader, selectedInstanceDir, cfg, dl) {
		logx.Error("No se puede iniciar sin un archivo de servidor.")
		return
	}

	ensurePlayit(reader, cfg, dl)

	tunnel := playit.New()
	if err := tunnel.Start(cfg); err != nil {
		logx.Error("Error iniciando Playit: %v", err)
	}
	defer tunnel.Stop()

	if err := properties.SetupInitialProperties(selectedInstanceDir); err != nil {
		logx.Error("Error configurando propiedades: %v", err)
		return
	}

	if err := eula.EnsureEulaAccepted(selectedInstanceDir); err != nil {
		logx.Error("Error con el EULA: %v", err)
		return
	}

	mods.DisableClientMods(selectedInstanceDir)

	logx.Info("\nEjecutando tareas de mantenimiento...")
	bm := backup.New(selectedInstanceDir, cfg.BackupRetentionDays)
	if err := bm.CreateBackup(); err != nil {
		logx.Error("Alerta de backup: %v", err)
	}

	svr := runner.New(cfg)
	svr.Start(selectedInstanceDir)
}
