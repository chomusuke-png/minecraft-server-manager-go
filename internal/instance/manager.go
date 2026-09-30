package instance

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/approot"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
	"minecraft-manager/internal/properties"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

var InstancesRootDir = approot.Path("instances")

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

func CreateInstance(reader *bufio.Reader, defaultRAMGB int) (string, int, string, error) {
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
		return "", 0, "", fmt.Errorf("no se pudo leer la entrada")
	}
	name, instancePath := choice.name, choice.path

	ramGB := promptRAM(reader, defaultRAMGB)
	tunnelProvider := PromptTunnelProvider(reader)

	if err := os.MkdirAll(instancePath, 0755); err != nil {
		return "", 0, "", fmt.Errorf("error creando directorio: %w", err)
	}

	logx.Success("Instancia '%s' creada en '%s' con %dGB de RAM.", name, instancePath, ramGB)
	return instancePath, ramGB, tunnelProvider, nil
}

func promptRAM(reader *bufio.Reader, defaultValue int) int {
	promptText := "[?] RAM asignada en GB" + prompt.Default(prompt.OriginDefault, defaultValue) + ": "
	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value <= 0 {
			return 0, false, "Error: ingresa un número entero válido mayor a 0."
		}
		return value, true, ""
	})
}

func DeleteInstance(reader *bufio.Reader, instancePath string) error {
	name := filepath.Base(instancePath)

	logx.Warn("\nEsto borra '%s' por completo (mundo, backups, todo). No se puede deshacer.", instancePath)
	confirmed, ok := prompt.Loop(reader, fmt.Sprintf("[?] Escribe '%s' para confirmar (Enter para cancelar): ", name), func(input string) (bool, bool, string) {
		if input == "" {
			return false, true, ""
		}
		if input == name {
			return true, true, ""
		}
		return false, false, "No coincide, reintenta (o Enter para cancelar)."
	})
	if !ok || !confirmed {
		logx.Info("Cancelado, no se borró nada.")
		return nil
	}

	if err := os.RemoveAll(instancePath); err != nil {
		return fmt.Errorf("no se pudo borrar '%s': %w", instancePath, err)
	}

	logx.Success("Instancia '%s' borrada.", name)
	return nil
}

// instanceColumns son los encabezados de la tabla de instancias del menu
var instanceColumns = []string{"NOMBRE", "MINECRAFT", "LOADER", "VERSION", "RAM", "PUERTO", "TUNEL"}

// FormatInstanceTable arma la tabla de instancias
func FormatInstanceTable(names []string) (string, []string) {
	cells := make([][]string, 0, len(names))
	for _, name := range names {
		cells = append(cells, instanceCells(name))
	}

	widths := make([]int, len(instanceColumns))
	for i, column := range instanceColumns {
		widths[i] = len(column)
	}
	for _, row := range cells {
		for i, value := range row {
			if width := utf8.RuneCountInString(value); width > widths[i] {
				widths[i] = width
			}
		}
	}

	rows := make([]string, 0, len(cells))
	for _, row := range cells {
		rows = append(rows, joinColumns(row, widths))
	}
	return joinColumns(instanceColumns, widths), rows
}

// instanceCells lee los datos de una instancia y devuelve el valor de cada
// columna, con "-" en los que todavia no existen
func instanceCells(name string) []string {
	instanceDir := filepath.Join(InstancesRootDir, name)
	// sin proveedor guardado se usa Playit, haya o no instance.json, igual que
	// hace setupTunnel al arrancar
	loader, version, loaderVersion, ram, tunnel := "-", "-", "-", "-", "playit"

	if meta, err := LoadMeta(instanceDir); err == nil {
		if meta.LoaderType != "" {
			loader = meta.LoaderType
		}
		if meta.MCVersion != "" {
			version = meta.MCVersion
		}
		// vanilla no tiene version propia, y las instancias viejas se crearon
		// antes de que se guardara
		if meta.LoaderVersion != "" {
			loaderVersion = meta.LoaderVersion
		}
		if meta.RAMGB > 0 {
			ram = fmt.Sprintf("%dGB", meta.RAMGB)
		}
		if meta.TunnelProvider != "" {
			tunnel = meta.TunnelProvider
		}
	}

	port := "-"
	if value, ok := properties.ReadPort(instanceDir); ok {
		port = strconv.Itoa(value)
	}

	return []string{name, version, loader, loaderVersion, ram, port, tunnel}
}

// joinColumns pega los valores de una fila rellenando cada uno hasta el ancho
// de su columna, sin dejar espacios al final de la linea
func joinColumns(values []string, widths []int) string {
	var row strings.Builder
	for i, value := range values {
		if i > 0 {
			row.WriteString("  ")
		}
		row.WriteString(value)
		if i < len(values)-1 {
			row.WriteString(strings.Repeat(" ", widths[i]-utf8.RuneCountInString(value)))
		}
	}
	return row.String()
}

func PromptRAMUpdate(reader *bufio.Reader, current int) int {
	// 0 es usar el valor global de config.json
	promptText := "[?] RAM asignada en GB" + prompt.Default(prompt.OriginCurrent, current) + ": "
	if current == 0 {
		promptText = "[?] RAM asignada en GB [global de config.json]: "
	}

	return prompt.LoopDefault(reader, promptText, current, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value <= 0 {
			return 0, false, "Valor inválido, ingresa un número entero mayor a 0."
		}
		return value, true, ""
	})
}

// tunnelProviders es el orden en que se numeran los tuneles en los menus
var tunnelProviders = []struct {
	value string
	label string
}{
	{"playit", "Playit"},
	{"ngrok", "ngrok"},
	{"none", "Ninguno"},
}

const recommendedTunnel = "playit"

func PromptTunnelProvider(reader *bufio.Reader) string {
	fmt.Println("\n[?] Túnel para exponer el servidor a internet:")
	return promptTunnel(reader, "")
}

func PromptTunnelProviderUpdate(reader *bufio.Reader, current string) string {
	if current == "" {
		current = recommendedTunnel
	}

	fmt.Println("\n[?] Túnel:")
	return promptTunnel(reader, current)
}

// promptTunnel marca la recomendada y, al actualizar, la actual. Enter elige
// la actual si hay, o la recomendada al crear
func promptTunnel(reader *bufio.Reader, current string) string {
	labels := make([]string, len(tunnelProviders))
	notes := make([]string, len(tunnelProviders))
	for i, provider := range tunnelProviders {
		labels[i] = provider.label
		switch {
		case provider.value == recommendedTunnel && provider.value == current:
			notes[i] = "recomendado y actual"
		case provider.value == recommendedTunnel:
			notes[i] = "recomendado"
		case provider.value == current:
			notes[i] = "actual"
		}
	}
	fmt.Print(prompt.Options("  ", labels, notes))

	defaultValue, enterHint := recommendedTunnel, prompt.EnterPicksRecommended
	if current != "" {
		defaultValue, enterHint = current, prompt.EnterKeepsCurrent
	}

	promptText := prompt.MenuQuestion(len(tunnelProviders), enterHint)
	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (string, bool, string) {
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(tunnelProviders) {
			return "", false, "Entrada incorrecta, reintente."
		}
		return tunnelProviders[choice-1].value, true, ""
	})
}

func PromptBackupKeepMinUpdate(reader *bufio.Reader, current int) int {
	promptText := "[?] Mínimo de backups a conservar (0 = el global)" + prompt.Default(prompt.OriginCurrent, current) + ": "
	if current == 0 {
		promptText = "[?] Mínimo de backups a conservar (0 = el global) [global de config.json]: "
	}

	return prompt.LoopDefault(reader, promptText, current, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value < 0 {
			return 0, false, "Valor inválido, ingresa un número entero mayor o igual a 0."
		}
		return value, true, ""
	})
}
