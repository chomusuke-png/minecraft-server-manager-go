package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"minecraft-manager/internal/backup"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/eula"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/mods"
	"minecraft-manager/internal/playit"
	"minecraft-manager/internal/properties"
	"minecraft-manager/internal/runner"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error cargando configuración global: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)

	selectedInstanceDir := selectInstanceFlow(reader)
	if selectedInstanceDir == "" {
		fmt.Println("[*] Operación cancelada.")
		return
	}

	fmt.Printf("\n[*] Trabajando sobre instancia: %s\n", selectedInstanceDir)

	dl := downloader.New(selectedInstanceDir)

	if !ensureServerJar(selectedInstanceDir, cfg, dl) {
		fmt.Println("[-] No se puede iniciar sin un archivo de servidor.")
		return
	}

	ensurePlayit(cfg, dl)

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

// --- Menu de instancias ---

func selectInstanceFlow(reader *bufio.Reader) string {
	instances, err := instance.GetAvailableInstances()
	if err != nil {
		fmt.Printf("[-] Error leyendo instancias: %v\n", err)
		return ""
	}

	fmt.Println("\n" + strings.Repeat("=", 30))
	fmt.Println("   SELECTOR DE INSTANCIAS")
	fmt.Println(strings.Repeat("=", 30))

	if len(instances) == 0 {
		fmt.Println("No hay instancias creadas.")
	} else {
		for i, inst := range instances {
			fmt.Printf("%d) %s\n", i+1, inst)
		}
	}
	fmt.Println("C) crear nueva instancia")
	fmt.Println("Q) salir")

	fmt.Print("\n[?] Opción: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.ToUpper(strings.TrimSpace(choice))

	if choice == "Q" {
		return ""
	}

	if choice == "C" {
		path, err := instance.CreateInstance(reader)
		if err != nil {
			fmt.Printf("[-] Error creando instancia: %v\n", err)
			return ""
		}
		return path
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(instances) {
		fmt.Println("[-] Opción inválida.")
		return ""
	}

	return filepath.Join(instance.InstancesRootDir, instances[idx-1])
}

// --- Helpers ---

func ensureServerJar(dir string, cfg *config.Config, dl *downloader.Downloader) bool {
	jarPath := filepath.Join(dir, cfg.JarName)

	if fileExists(jarPath) {
		return true
	}

	fmt.Printf("[!] No se encontró '%s' en '%s'.\n", cfg.JarName, dir)

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
