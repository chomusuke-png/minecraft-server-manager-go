package app

import (
	"bufio"

	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
)

// recoverLoaderInstall ofrece reconstruir el instance.json de una instancia
// Forge o NeoForge que lo perdio, o que lo tiene sin el comando de arranque.
// Esos loaders no dejan un server.jar, asi que sin esto la instancia no arranca
// aunque todo lo necesario siga en disco. Devuelve true si quedo lista
func recoverLoaderInstall(reader *bufio.Reader, dir string) bool {
	detected, ok := downloader.DetectInstalledLoader(dir)
	if !ok {
		return false
	}

	meta, err := instance.LoadMeta(dir)
	missingMeta := err != nil
	if missingMeta {
		meta = &instance.InstanceMeta{}
	}

	describeDetectedLoader(detected, missingMeta)

	if !prompt.YesNo(reader, "[?] ¿Reconstruir instance.json con esto y arrancar?") {
		return false
	}

	// se completa encima de lo que hubiera, para no perder la RAM, el tunel o el
	// minimo de backups que la instancia ya tuviera configurados
	if detected.LoaderType != "" {
		meta.LoaderType = detected.LoaderType
		meta.LoaderVersion = detected.LoaderVersion
	}
	if detected.MCVersion != "" {
		meta.MCVersion = detected.MCVersion
	}
	meta.LaunchArgs = detected.LaunchArgs

	if err := instance.SaveMeta(dir, *meta); err != nil {
		logx.Error("No se pudo guardar instance.json: %v", err)
		return false
	}

	logx.Success("instance.json reconstruido.")
	return true
}

func describeDetectedLoader(detected *downloader.DetectedLoader, missingMeta bool) {
	situation := "Esta instancia no tiene instance.json"
	if !missingMeta {
		situation = "El instance.json de esta instancia no tiene el comando de arranque"
	}

	label, known := downloader.LoaderLabel(detected.LoaderType)
	if !known {
		logx.Warn("%s, pero encontré el comando de arranque de un loader:", situation)
		logx.Detail("%s", detected.ArgsFile)
		logx.Detail("No pude identificar qué loader es ni su versión.")
		return
	}

	logx.Warn("%s, pero encontré una instalación de %s:", situation, label)
	logx.Detail("Minecraft %s · %s %s", orUnknown(detected.MCVersion), label, orUnknown(detected.LoaderVersion))
}

func orUnknown(value string) string {
	if value == "" {
		return "?"
	}
	return value
}
