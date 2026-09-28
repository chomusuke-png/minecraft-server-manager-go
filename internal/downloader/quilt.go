package downloader

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/java"
	"minecraft-manager/internal/logx"
)

const (
	quiltLoadersURL    = "https://meta.quiltmc.org/v3/versions/loader"
	quiltInstallersURL = "https://meta.quiltmc.org/v3/versions/installer"

	quiltInstallerName = "quilt-installer.jar"
	// el instalador deja este lanzador y al lado el server.jar vanilla, que el
	// lanzador carga por su nombre
	quiltLaunchJar = "quilt-server-launch.jar"
)

// Quilt no publica un jar de servidor listo como Fabric: hay que correr su
// instalador, asi que el arranque va por launch_args en vez de server.jar
var quiltLaunchArgs = []string{"-jar", quiltLaunchJar, "nogui"}

func quiltLoadersForMCURL(mcVersion string) string {
	return fmt.Sprintf("%s/%s", quiltLoadersURL, mcVersion)
}

func quiltVersions(mcVersion string) (loaderVersions, error) {
	// el listado general viene ordenado de la mas nueva a la mas vieja pero no
	// dice con que versiones de Minecraft sirve cada loader; el filtrado por
	// version si lo dice pero viene desordenado. se cruzan los dos
	var all []QuiltLoader
	if err := getJSON(quiltLoadersURL, &all); err != nil {
		return loaderVersions{}, fmt.Errorf("error obteniendo los loaders de Quilt: %w", err)
	}

	var compatible []QuiltLoaderForMC
	if err := getJSON(quiltLoadersForMCURL(mcVersion), &compatible); err != nil {
		return loaderVersions{}, fmt.Errorf("Quilt no tiene loaders para Minecraft %s: %w", mcVersion, err)
	}

	supported := make(map[string]bool, len(compatible))
	for _, entry := range compatible {
		supported[entry.Loader.Version] = true
	}
	return quiltVersionsFrom(all, supported, mcVersion)
}

func quiltVersionsFrom(all []QuiltLoader, supported map[string]bool, mcVersion string) (loaderVersions, error) {
	var versions loaderVersions
	for _, loader := range all {
		if !supported[loader.Version] {
			continue
		}
		versions.known = append(versions.known, loader.Version)
		if versions.latest == "" {
			versions.latest = loader.Version
		}
		if versions.stable == "" && !isQuiltPrerelease(loader.Version) {
			versions.stable = loader.Version
		}
	}

	if versions.latest == "" {
		return loaderVersions{}, fmt.Errorf("no se encontró ninguna versión de Quilt para Minecraft %s", mcVersion)
	}
	return versions, nil
}

// quilt versiona con semver: todo lo que lleva sufijo (-beta.N, -pre.N) es
// prerelease
func isQuiltPrerelease(version string) bool {
	return strings.Contains(version, "-")
}

func (d *Downloader) DownloadQuilt(mcVersion string, loaderVersion string) (string, []string, error) {
	logx.Info("Buscando el instalador de Quilt para %s...", mcVersion)

	var installers []QuiltInstaller
	if err := getJSON(quiltInstallersURL, &installers); err != nil {
		return "", nil, fmt.Errorf("error obteniendo los instaladores de Quilt: %w", err)
	}
	installer, err := latestQuiltInstaller(installers)
	if err != nil {
		return "", nil, err
	}

	logx.Detail("Loader: %s | Instalador: %s", loaderVersion, installer.Version)

	// los hashes que publica la meta de Quilt no coinciden con el jar del maven;
	// el sidecar del maven si
	sha256Hex := ""
	if sidecar, err := httpx.GetText(installer.URL + ".sha256"); err == nil {
		if fields := strings.Fields(sidecar); len(fields) > 0 {
			sha256Hex = fields[0]
		}
	} else {
		logx.Warn("No se pudo obtener el checksum del instalador de Quilt, se descarga sin verificar: %v", err)
	}

	if err := d.DownloadFileVerified(installer.URL, quiltInstallerName, "sha256", sha256Hex); err != nil {
		return "", nil, err
	}
	defer d.removeQuiltInstaller()

	// al cambiar desde Paper o Fabric queda su server.jar: se borra antes para
	// que la verificacion de abajo no lo tome por el vanilla si la descarga falla
	if err := os.Remove(filepath.Join(d.serverDir, "server.jar")); err != nil && !os.IsNotExist(err) {
		return "", nil, fmt.Errorf("no se pudo reemplazar el server.jar existente: %w", err)
	}

	if err := d.runQuiltInstaller(mcVersion, loaderVersion); err != nil {
		return "", nil, err
	}

	for _, required := range []string{quiltLaunchJar, "server.jar"} {
		if !fileExists(filepath.Join(d.serverDir, required)) {
			return "", nil, fmt.Errorf("el instalador de Quilt terminó pero no dejó %s", required)
		}
	}

	return loaderVersion, append([]string(nil), quiltLaunchArgs...), nil
}

// el listado de instaladores viene del mas nuevo al mas viejo y no marca
// estables
func latestQuiltInstaller(installers []QuiltInstaller) (QuiltInstaller, error) {
	for _, installer := range installers {
		if installer.URL != "" && !isQuiltPrerelease(installer.Version) {
			return installer, nil
		}
	}
	return QuiltInstaller{}, errors.New("la API de Quilt no devolvió ningún instalador")
}

func (d *Downloader) runQuiltInstaller(mcVersion, loaderVersion string) error {
	logx.Info("Ejecutando el instalador de Quilt...")
	logx.Detail("Descarga las librerías del loader y el servidor vanilla; puede tardar unos minutos.")

	installer := exec.Command(java.Absolute(d.javaPath),
		"-jar", quiltInstallerName,
		"install", "server", mcVersion, loaderVersion,
		"--download-server", "--install-dir=.",
	)
	installer.Dir = d.serverDir
	installer.Stdout = os.Stdout
	installer.Stderr = os.Stderr

	if err := installer.Run(); err != nil {
		return fmt.Errorf("el instalador de Quilt falló: %w", err)
	}
	return nil
}

func (d *Downloader) removeQuiltInstaller() {
	path := filepath.Join(d.serverDir, quiltInstallerName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		logx.Warn("No se pudo eliminar '%s': %v", quiltInstallerName, err)
	}
}
