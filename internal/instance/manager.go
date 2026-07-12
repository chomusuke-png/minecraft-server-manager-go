package instance

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
	"minecraft-manager/internal/properties"
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

type nameChoice struct {
	name string
	path string
}

func CreateInstance(reader *bufio.Reader, defaultRAMGB int) (string, int, error) {
	choice, ok := prompt.Loop(reader, "\n[?] Nombre para la nueva instancia (sin espacios): ", func(input string) (nameChoice, bool, string) {
		if input == "" {
			return nameChoice{}, false, "El nombre no puede estar vacío. Entrada incorrecta, reintente."
		}

		if strings.Contains(input, " ") || strings.Contains(input, "..") || strings.Contains(input, "/") || strings.Contains(input, "\\") {
			return nameChoice{}, false, "Nombre inválido (usa solo letras, números, guiones). Entrada incorrecta, reintente."
		}

		path := filepath.Join(InstancesRootDir, input)
		if _, err := os.Stat(path); err == nil {
			return nameChoice{}, false, fmt.Sprintf("La instancia '%s' ya existe. Entrada incorrecta, reintente.", input)
		}

		return nameChoice{name: input, path: path}, true, ""
	})
	if !ok {
		return "", 0, fmt.Errorf("no se pudo leer la entrada")
	}
	name, instancePath := choice.name, choice.path

	ramGB := promptRAM(reader, defaultRAMGB)

	if err := os.MkdirAll(instancePath, 0755); err != nil {
		return "", 0, fmt.Errorf("error creando directorio: %w", err)
	}

	logx.Success("Instancia '%s' creada en '%s' con %dGB de RAM.", name, instancePath, ramGB)
	return instancePath, ramGB, nil
}

func promptRAM(reader *bufio.Reader, defaultValue int) int {
	promptText := fmt.Sprintf("[?] RAM asignada en GB (Enter para usar %dGB): ", defaultValue)
	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value <= 0 {
			return 0, false, "Error: ingresá un número entero válido mayor a 0."
		}
		return value, true, ""
	})
}

func PrintInstanceInfo(instanceDir string) {
	meta, err := LoadMeta(instanceDir)
	if err != nil {
		return
	}

	info := fmt.Sprintf("%s %s", meta.LoaderType, meta.MCVersion)
	if meta.RAMGB > 0 {
		info += fmt.Sprintf(" | %dGB RAM", meta.RAMGB)
	}
	if port, ok := properties.ReadPort(instanceDir); ok {
		info += fmt.Sprintf(" | puerto %d", port)
	}
	fmt.Printf("    [%s]", info)
}

func PromptRAMUpdate(reader *bufio.Reader, current int) int {
	promptText := fmt.Sprintf("[?] RAM asignada en GB (Enter para mantener %dGB): ", current)
	if current == 0 {
		promptText = "[?] RAM asignada en GB (Enter para usar el valor global de config.json): "
	}

	return prompt.LoopDefault(reader, promptText, current, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value <= 0 {
			return 0, false, "Valor inválido, ingresá un número entero mayor a 0."
		}
		return value, true, ""
	})
}

