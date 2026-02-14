package properties

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func SetupInitialProperties(serverDir string) error {
	propertiesPath := filepath.Join(serverDir, "server.properties")

	if _, err := os.Stat(propertiesPath); err == nil {
		return nil
	}

	fmt.Println("\n[*] Configuración Inicial del Mundo (server.properties)")
	reader := bufio.NewReader(os.Stdin)

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

	fmt.Println("[+] Archivo server.properties generado exitosamente.")
	return nil
}

func promptString(reader *bufio.Reader, message, defaultValue string) string {
	fmt.Printf("%s [%s]: ", message, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func promptOptions(reader *bufio.Reader, message string, validOptions []string, defaultValue string) string {
	for {
		fmt.Printf("%s [%s]: ", message, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "" {
			return defaultValue
		}

		for _, option := range validOptions {
			if input == option {
				return input
			}
		}
		fmt.Printf("[-] Opción inválida. Valores permitidos: %v\n", validOptions)
	}
}

func promptInt(reader *bufio.Reader, message string, defaultValue int) int {
	for {
		fmt.Printf("%s [%d]: ", message, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultValue
		}

		parsedValue, err := strconv.Atoi(input)
		if err == nil && parsedValue > 0 {
			return parsedValue
		}
		fmt.Println("[-] Error: Por favor, ingresa un número entero válido mayor a 0.")
	}
}

func promptBoolean(reader *bufio.Reader, message string, defaultValue bool) bool {
	defaultStr := "false"
	if defaultValue {
		defaultStr = "true"
	}

	for {
		fmt.Printf("%s [%s]: ", message, defaultStr)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "" {
			return defaultValue
		}

		if input == "true" || input == "t" || input == "y" || input == "yes" || input == "si" {
			return true
		}

		if input == "false" || input == "f" || input == "n" || input == "no" {
			return false
		}

		fmt.Println("[-] Error: Responde 'true' o 'false'.")
	}
}
