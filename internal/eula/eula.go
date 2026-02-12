package eula

import (
	"fmt"
	"os"
	"strings"
)

func EnsureEulaAccepted(serverDir string) error {
	eulaPath := fmt.Sprintf("%s/eula.txt", serverDir)

	content, err := os.ReadFile(eulaPath)
	if err == nil {
		if strings.Contains(string(content), "eula=true") {
			return nil
		}
	}

	fmt.Println("\n[!] El EULA de Minecraft no ha sido aceptado.")
	fmt.Print("[?] ¿Aceptas el EULA? (y/n): ")

	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" {
		return fmt.Errorf("EULA rechazado por el usuario")
	}

	f, err := os.Create(eulaPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("eula=true\n")
	return err
}
