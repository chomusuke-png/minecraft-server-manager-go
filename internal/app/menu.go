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
	"minecraft-manager/internal/updater"
)

// Actualizar el loader de una instancia no debe cerrar el programa, así
// que vuelve a mostrar el menú en vez de retornar.
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

	for {
		fmt.Print("\n[?] Opción: ")
		choice, err := reader.ReadString('\n')
		choice = strings.ToUpper(strings.TrimSpace(choice))

		if err != nil {
			return "", ""
		}

		switch choice {
		case "Q":
			return "", ""

		case "C":
			path, ramGB, port, err := instance.CreateInstance(reader, cfg.RAMGB)
			if err != nil {
				logx.Error("Error creando instancia: %v", err)
				return "", ""
			}
			pendingMeta := instance.InstanceMeta{RAMGB: ramGB, Port: port}
			if err := instance.SaveMeta(path, pendingMeta); err != nil {
				logx.Warn("Advertencia: no se pudo guardar instance.json parcial: %v", err)
			}
			return path, ""

		case "U":
			if len(instances) == 0 {
				logx.Error("No hay instancias disponibles para actualizar.")
				continue
			}
			return selectExistingInstance(reader, instances), "update"

		default:
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(instances) {
				logx.Error("Entrada incorrecta, reintente.")
				continue
			}
			return filepath.Join(instance.InstancesRootDir, instances[idx-1]), ""
		}
	}
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

	for {
		fmt.Print("[?] Opción: ")
		choice, readErr := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if readErr != nil {
			return ""
		}

		idx, err := strconv.Atoi(choice)
		if err != nil || idx < 1 || idx > len(instances) {
			logx.Error("Entrada incorrecta, reintente.")
			continue
		}

		return filepath.Join(instance.InstancesRootDir, instances[idx-1])
	}
}
