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
	"minecraft-manager/internal/updater"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error cargando configuración global: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)

	selectedInstanceDir, action := selectInstanceFlow(reader, cfg)
	if selectedInstanceDir == "" {
		fmt.Println("[*] Operación cancelada.")
		return
	}

	if action == "update" {
		if err := updater.UpdateLoader(selectedInstanceDir, reader); err != nil {
			fmt.Printf("[-] Error actualizando loader: %v\n", err)
		}
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

func selectInstanceFlow(reader *bufio.Reader, cfg *config.Config) (string, string) {
	instances, err := instance.GetAvailableInstances()
	if err != nil {
		fmt.Printf("[-] Error leyendo instancias: %v\n", err)
		return "", ""
	}

	fmt.Println("\n" + strings.Repeat("=", 30))
	fmt.Println("   SELECTOR DE INSTANCIAS")
	fmt.Println(strings.Repeat("=", 30))

	if len(instances) == 0 {
		fmt.Println("No hay instancias creadas.")
	} else {
		for i, inst := range instances {
			instDir := filepath.Join(instance.InstancesRootDir, inst)
			fmt.Printf("%d) %s", i+1, inst)
			instance.PrintInstanceInfo(instDir)
			fmt.Println()
		}
	}

	fmt.Println("C) crear nueva instancia")
	fmt.Println("U) actualizar loader de una instancia")
	fmt.Println("Q) salir")

	fmt.Print("\n[?] Opción: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.ToUpper(strings.TrimSpace(choice))

	switch choice {
	case "Q":
		return "", ""

	case "C":
		path, ramGB, err := instance.CreateInstance(reader, cfg.RAMGB)
		if err != nil {
			fmt.Printf("[-] Error creando instancia: %v\n", err)
			return "", ""
		}
		meta := instance.InstanceMeta{RAMGB: ramGB}
		if err := instance.SaveMeta(path, meta); err != nil {
			fmt.Printf("[!] Advertencia: no se pudo guardar instance.json: %v\n", err)
		}
		return path, ""

	case "U":
		if len(instances) == 0 {
			fmt.Println("[-] No hay instancias disponibles para actualizar.")
			return "", ""
		}
		instDir := selectExistingInstance(reader, instances)
		return instDir, "update"

	default:
		idx, err := strconv.Atoi(choice)
		if err != nil || idx < 1 || idx > len(instances) {
			fmt.Println("[-] Opción inválida.")
			return "", ""
		}
		return filepath.Join(instance.InstancesRootDir, instances[idx-1]), ""
	}
}

func selectExistingInstance(reader *bufio.Reader, instances []string) string {
	fmt.Println("\n[?] Seleccioná la instancia a actualizar:")
	for i, inst := range instances {
		instDir := filepath.Join(instance.InstancesRootDir, inst)
		fmt.Printf("  %d) %s", i+1, inst)
		instance.PrintInstanceInfo(instDir)
		fmt.Println()
	}

	fmt.Print("[?] Opción: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

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

	result := dl.PromptUser()
	if result == nil {
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
