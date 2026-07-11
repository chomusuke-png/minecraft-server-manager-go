package instance

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/logx"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const InstancesRootDir = "instances"

func GetAvailableInstances() ([]string, error) {
	if _, err := os.Stat(InstancesRootDir); os.IsNotExist(err) {
		if err := os.Mkdir(InstancesRootDir, 0755); err != nil {
			return nil, err
		}
		return []string{}, nil
	}

	entries, err := os.ReadDir(InstancesRootDir)
	if err != nil {
		return nil, err
	}

	var instances []string
	for _, entry := range entries {
		if entry.IsDir() {
			instances = append(instances, entry.Name())
		}
	}
	return instances, nil
}

func CreateInstance(reader *bufio.Reader, defaultRAMGB int) (string, int, error) {
	var name, instancePath string
	for {
		fmt.Print("\n[?] Nombre para la nueva instancia (sin espacios): ")
		var readErr error
		name, readErr = reader.ReadString('\n')
		name = strings.TrimSpace(name)

		if readErr != nil {
			return "", 0, fmt.Errorf("no se pudo leer la entrada: %w", readErr)
		}

		if name == "" {
			logx.Error("El nombre no puede estar vacío. Entrada incorrecta, reintente.")
			continue
		}

		if strings.Contains(name, " ") || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			logx.Error("Nombre inválido (usa solo letras, números, guiones). Entrada incorrecta, reintente.")
			continue
		}

		instancePath = filepath.Join(InstancesRootDir, name)
		if _, err := os.Stat(instancePath); err == nil {
			logx.Error("La instancia '%s' ya existe. Entrada incorrecta, reintente.", name)
			continue
		}

		break
	}

	ramGB := promptRAM(reader, defaultRAMGB)

	if err := os.MkdirAll(instancePath, 0755); err != nil {
		return "", 0, fmt.Errorf("error creando directorio: %w", err)
	}

	logx.Success("Instancia '%s' creada en '%s' con %dGB de RAM.", name, instancePath, ramGB)
	return instancePath, ramGB, nil
}

func promptRAM(reader *bufio.Reader, defaultValue int) int {
	for {
		fmt.Printf("[?] RAM asignada en GB (Enter para usar %dGB): ", defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultValue
		}

		value, err := strconv.Atoi(input)
		if err == nil && value > 0 {
			return value
		}
		logx.Error("Error: ingresá un número entero válido mayor a 0.")
	}
}

func PrintInstanceInfo(instanceDir string) {
	meta, err := LoadMeta(instanceDir)
	if err != nil {
		return
	}
	if meta.RAMGB > 0 {
		fmt.Printf("    [%s %s | %dGB RAM]", meta.LoaderType, meta.MCVersion, meta.RAMGB)
	} else {
		fmt.Printf("    [%s %s]", meta.LoaderType, meta.MCVersion)
	}
}

func PromptRAMUpdate(reader *bufio.Reader, current int) int {
	if current == 0 {
		fmt.Print("[?] RAM asignada en GB (Enter para usar el valor global de config.json): ")
	} else {
		fmt.Printf("[?] RAM asignada en GB (Enter para mantener %dGB): ", current)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return current
	}

	value, err := strconv.Atoi(input)
	if err == nil && value > 0 {
		return value
	}

	logx.Error("Valor inválido, se mantiene el anterior.")
	return current
}
