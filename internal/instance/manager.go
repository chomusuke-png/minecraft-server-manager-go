package instance

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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

func CreateInstance(reader *bufio.Reader) (string, error) {
	fmt.Print("\n[?] Nombre para la nueva instancia (sin espacios): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if name == "" {
		return "", fmt.Errorf("el nombre no puede estar vacío")
	}

	if strings.Contains(name, " ") || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", fmt.Errorf("nombre inválido (usa solo letras, números, guiones)")
	}

	instancePath := filepath.Join(InstancesRootDir, name)
	if _, err := os.Stat(instancePath); err == nil {
		return "", fmt.Errorf("la instancia '%s' ya existe", name)
	}

	if err := os.MkdirAll(instancePath, 0755); err != nil {
		return "", fmt.Errorf("error creando directorio: %w", err)
	}

	fmt.Printf("[+] Instancia '%s' creada en '%s'.\n", name, instancePath)
	return instancePath, nil
}
