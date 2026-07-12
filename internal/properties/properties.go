package properties

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func SetupInitialProperties(reader *bufio.Reader, serverDir string) error {
	propertiesPath := filepath.Join(serverDir, "server.properties")

	if _, err := os.Stat(propertiesPath); err == nil {
		return nil
	}

	logx.Info("\nConfiguración Inicial del Mundo (server.properties)")

	motd := promptString(reader, "[?] Nombre/Mensaje del servidor (MOTD)", "Un servidor de Minecraft")
	difficulty := promptOptions(reader, "[?] Dificultad (peaceful, easy, normal, hard)", []string{"peaceful", "easy", "normal", "hard"}, "normal")
	maxPlayers := promptInt(reader, "[?] Jugadores máximos", 20)
	onlineMode := promptBoolean(reader, "[?] ¿Habilitar online-mode (requiere cuenta premium)? (true/false)", false)
	port := promptPort(reader, "[?] Puerto del servidor", 25565)

	fileContent := fmt.Sprintf("motd=%s\n"+
		"difficulty=%s\n"+
		"max-players=%d\n"+
		"online-mode=%t\n"+
		"server-port=%d\n",
		motd, difficulty, maxPlayers, onlineMode, port)

	err := os.WriteFile(propertiesPath, []byte(fileContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write server.properties: %w", err)
	}

	logx.Success("Archivo server.properties generado exitosamente.")
	return nil
}

func promptString(reader *bufio.Reader, message, defaultValue string) string {
	promptText := fmt.Sprintf("%s [%s]: ", message, defaultValue)
	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (string, bool, string) {
		return input, true, ""
	})
}

func promptOptions(reader *bufio.Reader, message string, validOptions []string, defaultValue string) string {
	promptText := fmt.Sprintf("%s [%s]: ", message, defaultValue)
	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (string, bool, string) {
		input = strings.ToLower(input)
		for _, option := range validOptions {
			if input == option {
				return input, true, ""
			}
		}
		return "", false, fmt.Sprintf("Opción inválida. Valores permitidos: %v", validOptions)
	})
}

func promptInt(reader *bufio.Reader, message string, defaultValue int) int {
	promptText := fmt.Sprintf("%s [%d]: ", message, defaultValue)
	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (int, bool, string) {
		parsedValue, err := strconv.Atoi(input)
		if err != nil || parsedValue <= 0 {
			return 0, false, "Error: Por favor, ingresa un número entero válido mayor a 0."
		}
		return parsedValue, true, ""
	})
}

func promptBoolean(reader *bufio.Reader, message string, defaultValue bool) bool {
	defaultStr := "false"
	if defaultValue {
		defaultStr = "true"
	}
	promptText := fmt.Sprintf("%s [%s]: ", message, defaultStr)

	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (bool, bool, string) {
		switch strings.ToLower(input) {
		case "true", "t", "y", "yes", "si":
			return true, true, ""
		case "false", "f", "n", "no":
			return false, true, ""
		}
		return false, false, "Error: Responde 'true' o 'false'."
	})
}

func promptPort(reader *bufio.Reader, message string, defaultValue int) int {
	promptText := fmt.Sprintf("%s [%d]: ", message, defaultValue)
	return prompt.LoopDefault(reader, promptText, defaultValue, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value <= 0 || value > 65535 {
			return 0, false, "Error: ingresá un puerto válido (1-65535)."
		}
		return value, true, ""
	})
}

// ReadPort busca la línea "server-port=" en el server.properties de la
// instancia. ok=false si el archivo no existe o no tiene esa clave.
func ReadPort(serverDir string) (int, bool) {
	data, err := os.ReadFile(filepath.Join(serverDir, "server.properties"))
	if err != nil {
		return 0, false
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		value, found := strings.CutPrefix(line, "server-port=")
		if !found {
			continue
		}
		port, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return port, true
	}
	return 0, false
}

// UpdatePort deja cambiar el puerto de una instancia ya configurada,
// reescribiendo solo la línea "server-port=" y dejando el resto del
// archivo intacto. Si la instancia todavía no tiene server.properties
// (nunca se llegó a lanzar), avisa y no hace nada.
func UpdatePort(reader *bufio.Reader, serverDir string) error {
	propertiesPath := filepath.Join(serverDir, "server.properties")

	data, err := os.ReadFile(propertiesPath)
	if err != nil {
		logx.Warn("Esta instancia todavía no tiene server.properties; el puerto se configurará la primera vez que la inicies.")
		return nil
	}

	current, _ := ReadPort(serverDir)
	if current == 0 {
		current = 25565
	}

	newPort := promptPort(reader, "[?] Puerto del servidor (Enter para mantener el actual)", current)

	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "server-port=") {
			lines[i] = fmt.Sprintf("server-port=%d", newPort)
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, fmt.Sprintf("server-port=%d", newPort))
	}

	if err := os.WriteFile(propertiesPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to update server-port: %w", err)
	}

	logx.Success("Puerto actualizado a %d.", newPort)
	return nil
}
