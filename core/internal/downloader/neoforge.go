package downloader

import (
	"fmt"
	"strconv"
	"strings"

	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/logx"
)

const neoForgeInstallerName = "neoforge-installer.jar"

const neoForgeMavenMetadataURL = "https://maven.neoforged.net/releases/net/neoforged/neoforge/maven-metadata.xml"

// NeoForge arranca en MC 1.20.2, ya en la era del args file: nunca tiene jar legacy
var neoForgeSpec = forgeLikeSpec{
	loaderType:      "neoforge",
	installerName:   neoForgeInstallerName,
	libraryGroup:    []string{"net", "neoforged", "neoforge"},
	legacyJarPrefix: "",
}

func (d *Downloader) DownloadNeoForge(version string, neoForgeVersion string) (string, []string, error) {
	logx.Info("Buscando el instalador de NeoForge %s para %s...", neoForgeVersion, version)

	downloadURL := fmt.Sprintf(
		"https://maven.neoforged.net/releases/net/neoforged/neoforge/%[1]s/neoforge-%[1]s-installer.jar",
		neoForgeVersion,
	)

	sha1Hex := ""
	if sidecar, err := httpx.GetText(downloadURL + ".sha1"); err == nil {
		if fields := strings.Fields(sidecar); len(fields) > 0 {
			sha1Hex = fields[0]
		}
	} else {
		logx.Warn("No se pudo obtener el checksum del instalador de NeoForge, se descarga sin verificar: %v", err)
	}

	if err := d.DownloadFileVerified(downloadURL, neoForgeInstallerName, "sha1", sha1Hex); err != nil {
		return "", nil, err
	}

	launchArgs, err := d.installForgeLike(neoForgeSpec, neoForgeVersion)
	if err != nil {
		d.removeForgeLikeInstaller(neoForgeSpec)
		return "", nil, err
	}

	return neoForgeVersion, launchArgs, nil
}

// neoForgeVersionPrefix deriva el prefijo con el que NeoForge numera sus
// versiones para una version de Minecraft. Hasta la 1.21.x usaba tres numeros,
// asi que 1.X.Y es "X.Y."; con la numeracion por año usa cuatro, y 26.D.P es
// "26.D.P." con el patch en 0 si falta: 26.3 es "26.3.0." y 26.1.2 es "26.1.2."
func neoForgeVersionPrefix(mcVersion string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(mcVersion), ".")
	if len(parts) < 2 {
		return "", false
	}

	first, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", false
	}
	second, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", false
	}

	patch := 0
	if len(parts) > 2 {
		// Ignora sufijos tipo "1-pre1": alcanza el número de adelante.
		digits := parts[2]
		end := 0
		for end < len(digits) && digits[end] >= '0' && digits[end] <= '9' {
			end++
		}
		if end > 0 {
			patch, _ = strconv.Atoi(digits[:end])
		}
	}

	switch {
	case first == 1:
		return fmt.Sprintf("%d.%d.", second, patch), true
	case first >= 26:
		return fmt.Sprintf("%d.%d.%d.", first, second, patch), true
	}
	return "", false
}
