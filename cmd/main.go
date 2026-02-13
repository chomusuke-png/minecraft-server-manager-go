package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"minecraft-manager/internal/backup"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/eula"
	"minecraft-manager/internal/mods"
	"minecraft-manager/internal/runner"
)

const (
	serverDirName = "server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error cargando configuración: %v", err)
	}

	if err := os.MkdirAll(serverDirName, 0755); err != nil {
		log.Fatalf("[-] No se pudo crear el directorio '%s': %v", serverDirName, err)
	}

	dl := downloader.New(serverDirName)

	if !ensureServerJar(cfg, dl) {
		fmt.Println("[-] No se puede iniciar sin un archivo de servidor (server.jar).")
		return
	}

	ensurePlayit(cfg, dl)
	startPlayit(cfg)

	if err := eula.EnsureEulaAccepted(serverDirName); err != nil {
		fmt.Printf("[-] Error con el EULA: %v\n", err)
		return
	}

	mods.DisableClientMods(serverDirName)

	fmt.Println("\n[*] Ejecutando tareas de mantenimiento...")
	bm := backup.New(serverDirName, cfg.BackupRetentionDays)
	if err := bm.CreateBackup(); err != nil {
		fmt.Printf("[-] Alerta de backup: %v (continuando igual...)\n", err)
	}

	svr := runner.New(cfg)
	svr.Start()
}

func startPlayit(cfg *config.Config) {
	if !fileExists(cfg.PlayitPath) {
		return
	}

	fmt.Println("[*] Iniciando túnel de Playit.gg...")

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "start", cfg.PlayitPath)
	} else {
		cmd = exec.Command(cfg.PlayitPath)
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("[-] Error al ejecutar Playit: %v\n", err)
	}
}

func ensureServerJar(cfg *config.Config, dl *downloader.Downloader) bool {
	jarPath := filepath.Join(serverDirName, cfg.JarName)

	if fileExists(jarPath) {
		return true
	}

	fmt.Printf("[!] No se encontró '%s' en '%s'.\n", cfg.JarName, serverDirName)

	if !askYesNo("[?] ¿Descargar servidor automáticamente?") {
		return false
	}

	return dl.PromptUser()
}

func ensurePlayit(cfg *config.Config, dl *downloader.Downloader) {
	if fileExists(cfg.PlayitPath) {
		return
	}

	fmt.Printf("[!] No se encontró '%s'.\n", cfg.PlayitPath)
	if askYesNo("[?] ¿Deseas descargar Playit.gg automáticamente?") {
		if err := dl.DownloadPlayit(); err != nil {
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

func askYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/n): ", question)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "s"
}
