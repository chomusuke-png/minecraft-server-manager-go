package eula

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func EnsureEulaAccepted(serverDir string) error {
	eulaPath := filepath.Join(serverDir, "eula.txt")

	content, err := os.ReadFile(eulaPath)
	if err == nil {
		if strings.Contains(string(content), "eula=true") {
			return nil
		}
	}

	fmt.Println("\n[!] El EULA de Minecraft no ha sido aceptado.")

	accepted := false
	for {
		fmt.Print("[?] ¿Aceptas el EULA? (y/n): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("no se pudo leer la respuesta: %w", err)
			}
			fmt.Println("[-] Entrada incorrecta, reintente.")
			continue
		}

		switch strings.ToLower(response) {
		case "y", "s", "si", "yes":
			accepted = true
		case "n", "no":
			accepted = false
		default:
			fmt.Println("[-] Entrada incorrecta, reintente.")
			continue
		}
		break
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
