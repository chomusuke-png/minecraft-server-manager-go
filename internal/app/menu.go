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

func runMenuLoop(reader *bufio.Reader, cfg *config.Config) string {
	for {
		selectedInstanceDir, action := selectInstanceFlow(reader, cfg)
		if selectedInstanceDir == "" {
			return ""
		}

		if action == "update" {
			if err := updater.UpdateLoader(selectedInstanceDir, reader); err != nil {
				logx.Error("Error actualizando loader: %v", err)
			}
			continue
		}

		return selectedInstanceDir
	}
}

func selectInstanceFlow(reader *bufio.Reader, cfg *config.Config) (string, string) {
	instances, err := instance.GetAvailableInstances()
	if err != nil {
		logx.Error("Error leyendo instancias: %v", err)
		return "", ""
	}

	clearScreen()

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

	result, ok := prompt.Loop(reader, "\n[?] Opción: ", func(input string) (menuChoice, bool, string) {
		choice := strings.ToUpper(input)

		switch choice {
		case "Q":
			return menuChoice{}, true, ""

		case "C":
			path, ramGB, err := instance.CreateInstance(reader, cfg.RAMGB)
			if err != nil {
				logx.Error("Error creando instancia: %v", err)
				return menuChoice{}, true, ""
			}
			pendingMeta := instance.InstanceMeta{RAMGB: ramGB}
			if err := instance.SaveMeta(path, pendingMeta); err != nil {
				logx.Warn("Advertencia: no se pudo guardar instance.json parcial: %v", err)
			}
			return menuChoice{path: path}, true, ""

		case "U":
			if len(instances) == 0 {
				return menuChoice{}, false, "No hay instancias disponibles para actualizar."
			}
			return menuChoice{path: selectExistingInstance(reader, instances), action: "update"}, true, ""

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

func selectExistingInstance(reader *bufio.Reader, instances []string) string {
	fmt.Println("\n[?] Seleccioná la instancia a actualizar:")
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
