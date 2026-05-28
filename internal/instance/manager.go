package instance

import (
	"bufio"
	"fmt"
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
	fmt.Print("\n[?] Nombre para la nueva instancia (sin espacios): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if name == "" {
		return "", 0, fmt.Errorf("el nombre no puede estar vacío")
	}

	if strings.Contains(name, " ") || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", 0, fmt.Errorf("nombre inválido (usa solo letras, números, guiones)")
	}

	instancePath := filepath.Join(InstancesRootDir, name)
	if _, err := os.Stat(instancePath); err == nil {
		return "", 0, fmt.Errorf("la instancia '%s' ya existe", name)
	}

	ramGB := promptRAM(reader, defaultRAMGB)

	if err := os.MkdirAll(instancePath, 0755); err != nil {
		return "", 0, fmt.Errorf("error creando directorio: %w", err)
	}

	fmt.Printf("[+] Instancia '%s' creada en '%s' con %dGB de RAM.\n", name, instancePath, ramGB)
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
		fmt.Println("[-] Error: ingresá un número entero válido mayor a 0.")
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

	fmt.Println("[-] Valor inválido, se mantiene el anterior.")
	return current
}
