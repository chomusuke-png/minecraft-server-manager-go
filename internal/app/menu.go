package app

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
	"minecraft-manager/internal/updater"
)

const (
	appName     = "MINECRAFT SERVER MANAGER"
	appAuthor   = "chomusuke-png (Zumito)"
	headerWidth = 60
)

func runMenuLoop(reader *bufio.Reader, cfg *config.Config, version string) string {
	for {
		selectedInstanceDir, action := selectInstanceFlow(reader, cfg, version)
		if selectedInstanceDir == "" {
			return ""
		}

		if action == "update" {
			if err := updater.UpdateLoader(selectedInstanceDir, reader, cfg.JavaPath); err != nil {
				logx.Error("Error actualizando loader: %v", err)
			}
			continue
		}

		if action == "delete" {
			if selectedInstanceDir != "" {
				if err := instance.DeleteInstance(reader, selectedInstanceDir); err != nil {
					logx.Error("Error borrando instancia: %v", err)
				}
			}
			continue
		}

		return selectedInstanceDir
	}
}

func selectInstanceFlow(reader *bufio.Reader, cfg *config.Config, version string) (string, string) {
	instances, err := instance.GetAvailableInstances()
	if err != nil {
		logx.Error("Error leyendo instancias: %v", err)
		return "", ""
	}

	clearScreen()

	printHeader(version)
	printInstances(instances)
	printActions()

	result, ok := prompt.Loop(reader, "\n[?] Opción: ", func(input string) (menuChoice, bool, string) {
		choice := strings.ToUpper(input)

		switch choice {
		case "Q":
			return menuChoice{}, true, ""

		case "C":
			path, ramGB, tunnelProvider, err := instance.CreateInstance(reader, cfg.RAMGB)
			if err != nil {
				logx.Error("Error creando instancia: %v", err)
				return menuChoice{}, true, ""
			}
			pendingMeta := instance.InstanceMeta{RAMGB: ramGB, TunnelProvider: tunnelProvider}
			if err := instance.SaveMeta(path, pendingMeta); err != nil {
				logx.Warn("Advertencia: no se pudo guardar instance.json parcial: %v", err)
			}
			return menuChoice{path: path}, true, ""

		case "U":
			if len(instances) == 0 {
				return menuChoice{}, false, "No hay instancias disponibles para actualizar."
			}
			return menuChoice{path: selectExistingInstance(reader, instances, "actualizar"), action: "update"}, true, ""

		case "D":
			if len(instances) == 0 {
				return menuChoice{}, false, "No hay instancias disponibles para borrar."
			}
			return menuChoice{path: selectExistingInstance(reader, instances, "borrar"), action: "delete"}, true, ""

		default:
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(instances) {
				return menuChoice{}, false, "Entrada incorrecta, reintente."
			}
			return menuChoice{path: filepath.Join(instance.InstancesRootDir, instances[idx-1])}, true, ""
		}
	})

	if !ok {
		return "", ""
	}
	return result.path, result.action
}

// printHeader muestra el nombre del proyecto con la version del ejecutable
// alineada al margen derecho, y el autor debajo
func printHeader(version string) {
	if version == "" {
		version = "dev"
	}

	line := strings.Repeat("=", headerWidth)
	title := "  " + appName
	padding := headerWidth - len(title) - len(version)
	if padding < 1 {
		padding = 1
	}

	fmt.Println("\n" + line)
	fmt.Printf("%s%s%s\n", title, strings.Repeat(" ", padding), version)
	fmt.Printf("  por %s\n", appAuthor)
	fmt.Println(line)
}

func printInstances(instances []string) {
	fmt.Println("\n  INSTANCIAS")

	if len(instances) == 0 {
		fmt.Println("    (no hay instancias creadas)")
		return
	}

	for i, inst := range instances {
		instDir := filepath.Join(instance.InstancesRootDir, inst)
		fmt.Printf("    %d) %s", i+1, inst)
		instance.PrintInstanceInfo(instDir)
		fmt.Println()
	}
}

func printActions() {
	fmt.Println("\n  ACCIONES")
	fmt.Println("    C) crear nueva instancia")
	fmt.Println("    U) actualizar loader de una instancia")
	fmt.Println("    D) borrar una instancia")
	fmt.Println("    Q) salir")
}

type menuChoice struct {
	path   string
	action string
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func selectExistingInstance(reader *bufio.Reader, instances []string, purpose string) string {
	fmt.Printf("\n[?] Seleccioná la instancia a %s:\n", purpose)
	for i, inst := range instances {
		instDir := filepath.Join(instance.InstancesRootDir, inst)
		fmt.Printf("  %d) %s", i+1, inst)
		instance.PrintInstanceInfo(instDir)
		fmt.Println()
	}

	path, ok := prompt.Loop(reader, "[?] Opción: ", func(input string) (string, bool, string) {
		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(instances) {
			return "", false, "Entrada incorrecta, reintente."
		}
		return filepath.Join(instance.InstancesRootDir, instances[idx-1]), true, ""
	})
	if !ok {
		return ""
	}
	return path
}
