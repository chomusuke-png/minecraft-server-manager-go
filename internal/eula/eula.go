package eula

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
	"os"
	"path/filepath"
	"strings"
)

func EnsureEulaAccepted(reader *bufio.Reader, serverDir string) error {
	eulaPath := filepath.Join(serverDir, "eula.txt")

	content, err := os.ReadFile(eulaPath)
	if err == nil {
		if strings.Contains(string(content), "eula=true") {
			return nil
		}
	}

	logx.Warn("\nEl EULA de Minecraft no ha sido aceptado.")

	accepted, ok := prompt.Loop(reader, "[?] ¿Aceptas el EULA? (y/n): ", func(input string) (bool, bool, string) {
		switch strings.ToLower(input) {
		case "y", "s", "si", "yes":
			return true, true, ""
		case "n", "no":
			return false, true, ""
		}
		return false, false, "Entrada incorrecta, reintente."
	})
	if !ok {
		return fmt.Errorf("no se pudo leer la respuesta")
	}

	if !accepted {
		return fmt.Errorf("EULA rechazado por el usuario")
	}

	eulaFile, err := os.Create(eulaPath)
	if err != nil {
		return err
	}
	defer eulaFile.Close()

	_, err = eulaFile.WriteString("eula=true\n")
	return err
}
