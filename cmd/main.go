package main

import (
	"fmt"
	"log"

	"minecraft-manager/internal/backup"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/runner"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error configuración: %v", err)
	}

	dl := downloader.New("server")
	if cfg.PlayitPath == "playit.exe" {
		dl.DownloadPlayit()
	}

	fmt.Println("[*] Ejecutando tareas de mantenimiento...")
	bm := backup.New("server", cfg.BackupRetentionDays)
	if err := bm.CreateBackup(); err != nil {
		fmt.Printf("[-] Alerta de backup: %v (continuando igual...)\n", err)
	}

	svr := runner.New(cfg)
	svr.Start()
}
