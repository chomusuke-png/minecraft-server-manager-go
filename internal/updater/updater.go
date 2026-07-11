package updater

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/logx"
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
	logx.Info("\nInstancia actual: %s %s | RAM: %s", meta.LoaderType, meta.MCVersion, ramDisplay)

	fmt.Printf("[?] Nueva versión de Minecraft (Enter para mantener '%s'): ", meta.MCVersion)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	newVersion := meta.MCVersion
	if input != "" {
		newVersion = input
	}

	newLoaderType := promptLoaderType(reader, meta.LoaderType)

	updatedRAMGB := instance.PromptRAMUpdate(reader, meta.RAMGB)

	serverDownloader := downloader.New(instanceDir)

	logx.Info("Descargando %s %s...", newLoaderType, newVersion)
	var newLoaderVersion string
	switch newLoaderType {
	case "paper":
		newLoaderVersion, err = serverDownloader.DownloadPaper(newVersion)
	case "fabric":
		newLoaderVersion, err = serverDownloader.DownloadFabric(newVersion)
	case "vanilla":
		newLoaderVersion, err = serverDownloader.DownloadVanilla(newVersion)
	default:
		return fmt.Errorf("tipo de loader desconocido: %s", newLoaderType)
	}

	if err != nil {
		return fmt.Errorf("error descargando: %w", err)
	}

	meta.MCVersion = newVersion
	meta.LoaderType = newLoaderType
	meta.LoaderVersion = newLoaderVersion
	meta.RAMGB = updatedRAMGB
	if err := instance.SaveMeta(instanceDir, *meta); err != nil {
		logx.Warn("Advertencia: no se pudo actualizar instance.json: %v", err)
	}

	logx.Success("Loader actualizado a %s %s exitosamente.", newLoaderType, newVersion)
	return nil
}

func promptLoaderType(reader *bufio.Reader, current string) string {
	fmt.Printf("\n[?] Tipo de loader (Enter para mantener '%s'):\n", current)
	fmt.Println("  1) Paper")
	fmt.Println("  2) Fabric")
	fmt.Println("  3) Vanilla")

	for {
		fmt.Print("\n[?] Opción (Enter para mantener actual): ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "":
			return current
		case "1":
			return "paper"
		case "2":
			return "fabric"
		case "3":
			return "vanilla"
		}

		logx.Error("Entrada incorrecta, reintente.")
	}
}
