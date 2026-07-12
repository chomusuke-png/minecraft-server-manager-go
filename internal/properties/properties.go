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

	fileContent := fmt.Sprintf("motd=%s\n"+
		"difficulty=%s\n"+
		"max-players=%d\n"+
		"online-mode=%t\n",
		motd, difficulty, maxPlayers, onlineMode)

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
