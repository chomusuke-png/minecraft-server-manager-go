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

// NeoForge arranca en MC 1.20.2, ya en la era del args file: nunca tiene jar
// legacy.
var neoForgeSpec = forgeLikeSpec{
	installerName:   neoForgeInstallerName,
	libraryGroup:    []string{"net", "neoforged", "neoforge"},
	legacyJarPrefix: "",
}

func (d *Downloader) DownloadNeoForge(version string) (string, []string, error) {
	logx.Info("Fetching NeoForge installer for %s...", version)

	neoForgeVersion, err := d.findNeoForgeVersion(version)
	if err != nil {
		return "", nil, err
	}

	logx.Detail("NeoForge Version: %s", neoForgeVersion)

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

// findNeoForgeVersion resuelve la versión de NeoForge para una versión de
// Minecraft dada. A diferencia de Forge, NeoForge no publica un mapeo
// mc -> recomendada: solo la lista plana de versiones de maven-metadata.xml,
// en orden de publicación. Cada versión arranca con "<minorMC>.<patchMC>."
// (1.21.1 -> "21.1.", 1.21 -> "21.0."), así que se filtra por ese prefijo y
// se toma el último match (el más reciente), prefiriendo estable sobre
// pre-release si hay ambos.
func (d *Downloader) findNeoForgeVersion(mcVersion string) (string, error) {
	prefix, ok := neoForgeVersionPrefix(mcVersion)
	if !ok {
		return "", fmt.Errorf("no se pudo interpretar la versión de Minecraft '%s'", mcVersion)
	}

	var metadata NeoForgeMavenMetadata
	if err := httpx.GetXML(neoForgeMavenMetadataURL, &metadata); err != nil {
		return "", fmt.Errorf("error obteniendo versiones de NeoForge: %w", err)
	}

	stable, prerelease := "", ""
	for _, version := range metadata.Versioning.Versions.Version {
		if !strings.HasPrefix(version, prefix) {
			continue
		}
		if strings.Contains(version, "-beta") || strings.Contains(version, "-alpha") {
			prerelease = version
		} else {
			stable = version
		}
	}

	if stable != "" {
		return stable, nil
	}
	if prerelease != "" {
		logx.Warn("Solo hay builds pre-release de NeoForge para Minecraft %s, se usa %s.", mcVersion, prerelease)
		return prerelease, nil
	}
	return "", fmt.Errorf("no se encontró ninguna versión de NeoForge para Minecraft %s", mcVersion)
}

// neoForgeVersionPrefix deriva el prefijo "<minor>.<patch>." que usan las
// versiones de NeoForge a partir de una versión de Minecraft "1.X" o "1.X.Y".
func neoForgeVersionPrefix(mcVersion string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(mcVersion), ".")
	if len(parts) < 2 || parts[0] != "1" {
		return "", false
	}

	minor, err := strconv.Atoi(parts[1])
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

	return fmt.Sprintf("%d.%d.", minor, patch), true
}
