package updater

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/instance"
	"strings"
)

func UpdateLoader(instanceDir string, reader *bufio.Reader) error {
	meta, err := instance.LoadMeta(instanceDir)
	if err != nil {
		return fmt.Errorf(
			"instancia sin metadata: %w\nTip: las instancias creadas antes de esta versión no tienen instance.json.\nPodés crearlo manualmente con el formato: {\"loader_type\": \"paper\", \"mc_version\": \"1.20.1\", \"ram_gb\": 4}",
			err,
		)
	}

	if meta.LoaderType == "" || meta.MCVersion == "" {
		return fmt.Errorf(
			"metadata incompleta: loader_type='%s', mc_version='%s'\n"+
				"La instancia puede haberse creado sin completar la descarga del JAR.\n"+
				"Editá instance.json manualmente o eliminá la instancia y volvé a crearla.",
			meta.LoaderType, meta.MCVersion,
		)
	}

	ramDisplay := "global (config.json)"
	if meta.RAMGB > 0 {
		ramDisplay = fmt.Sprintf("%dGB", meta.RAMGB)
	}
	fmt.Printf("\n[*] Instancia actual: %s %s | RAM: %s\n", meta.LoaderType, meta.MCVersion, ramDisplay)

	fmt.Printf("[?] Nueva versión de Minecraft (Enter para mantener '%s'): ", meta.MCVersion)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	newVersion := meta.MCVersion
	if input != "" {
		newVersion = input
	}

	newLoaderType, err := promptLoaderType(reader, meta.LoaderType)
	if err != nil {
		return err
	}

	newRAMGB := instance.PromptRAMUpdate(reader, meta.RAMGB)

	dl := downloader.New(instanceDir)

	fmt.Printf("[*] Descargando %s %s...\n", newLoaderType, newVersion)
	switch newLoaderType {
	case "paper":
		err = dl.DownloadPaper(newVersion)
	case "fabric":
		err = dl.DownloadFabric(newVersion)
	case "vanilla":
		err = dl.DownloadVanilla(newVersion)
	default:
		return fmt.Errorf("tipo de loader desconocido: %s", newLoaderType)
	}

	if err != nil {
		return fmt.Errorf("error descargando: %w", err)
	}

	meta.MCVersion = newVersion
	meta.LoaderType = newLoaderType
	meta.RAMGB = newRAMGB
	if err := instance.SaveMeta(instanceDir, *meta); err != nil {
		fmt.Printf("[!] Advertencia: no se pudo actualizar instance.json: %v\n", err)
	}

	fmt.Printf("[+] Loader actualizado a %s %s exitosamente.\n", newLoaderType, newVersion)
	return nil
}

func promptLoaderType(reader *bufio.Reader, current string) (string, error) {
	fmt.Printf("\n[?] Tipo de loader (Enter para mantener '%s'):\n", current)
	fmt.Println("  1) Paper")
	fmt.Println("  2) Fabric")
	fmt.Println("  3) Vanilla")
	fmt.Print("\n[?] Opción (Enter para mantener actual): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return current, nil
	}

	switch input {
	case "1":
		return "paper", nil
	case "2":
		return "fabric", nil
	case "3":
		return "vanilla", nil
	default:
		return "", fmt.Errorf("opción inválida: %s", input)
	}
}
