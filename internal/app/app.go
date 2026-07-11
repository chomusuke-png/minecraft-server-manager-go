package app

import (
	"bufio"
	"fmt"
	"os"

	"minecraft-manager/internal/backup"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/eula"
	"minecraft-manager/internal/mods"
	"minecraft-manager/internal/playit"
	"minecraft-manager/internal/properties"
	"minecraft-manager/internal/runner"
)

func Run(cfg *config.Config) {
	reader := bufio.NewReader(os.Stdin)

	selectedInstanceDir := runMenuLoop(reader, cfg)
	if selectedInstanceDir == "" {
		fmt.Println("[*] Operación cancelada.")
		return
	}

	fmt.Printf("\n[*] Trabajando sobre instancia: %s\n", selectedInstanceDir)

	dl := downloader.New(selectedInstanceDir)

	if !ensureServerJar(reader, selectedInstanceDir, cfg, dl) {
		fmt.Println("[-] No se puede iniciar sin un archivo de servidor.")
		return
	}

	ensurePlayit(reader, cfg, dl)

	tunnel := playit.New()
	if err := tunnel.Start(cfg); err != nil {
		fmt.Printf("[-] Error iniciando Playit: %v\n", err)
	}
	defer tunnel.Stop()

	if err := properties.SetupInitialProperties(selectedInstanceDir); err != nil {
		fmt.Printf("[-] Error configurando propiedades: %v\n", err)
		return
	}

	if err := eula.EnsureEulaAccepted(selectedInstanceDir); err != nil {
		fmt.Printf("[-] Error con el EULA: %v\n", err)
		return
	}

	mods.DisableClientMods(selectedInstanceDir)

	fmt.Println("\n[*] Ejecutando tareas de mantenimiento...")
	bm := backup.New(selectedInstanceDir, cfg.BackupRetentionDays)
	if err := bm.CreateBackup(); err != nil {
		fmt.Printf("[-] Alerta de backup: %v\n", err)
	}

	svr := runner.New(cfg)
	svr.Start(selectedInstanceDir)
}
