package java

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/logx"
)

// adoptiumAsset es la respuesta (parcial) de /v3/assets/latest/{major}/hotspot.
type adoptiumAsset struct {
	Binary struct {
		Package struct {
			Link     string `json:"link"`
			Checksum string `json:"checksum"`
		} `json:"package"`
	} `json:"binary"`
}

// Download baja el JDK del major pedido desde Adoptium y lo descomprime en
// runtimes/jdk-<major>/, devolviendo la ruta al ejecutable de java.
func Download(major int) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("la descarga automática de JDK sólo está implementada para Windows")
	}

	arch, err := adoptiumArch()
	if err != nil {
		return "", err
	}

	// El endpoint /v3/binary/latest/... redirige directo al asset sin dar
	// checksum. El endpoint /v3/assets/latest/... da el mismo binario más su
	// sha256, así que se usa este para poder verificar lo que se descarga y
	// después se descomprime y ejecuta.
	assetsURL := fmt.Sprintf(
		"https://api.adoptium.net/v3/assets/latest/%d/hotspot?os=windows&architecture=%s&image_type=jdk&jvm_impl=hotspot",
		major, arch,
	)

	var assets []adoptiumAsset
	if err := httpx.GetJSON(assetsURL, &assets); err != nil {
		return "", fmt.Errorf("no se pudo consultar Adoptium: %w", err)
	}
	if len(assets) == 0 || assets[0].Binary.Package.Link == "" {
		return "", fmt.Errorf("Adoptium no tiene un JDK %d para windows/%s", major, arch)
	}
	url := assets[0].Binary.Package.Link
	sha256Hex := assets[0].Binary.Package.Checksum

	destination := filepath.Join(RuntimesRootDir, fmt.Sprintf("jdk-%d", major))

	// Idempotente: si ya está extraído y funciona, no rebajar 180MB al vacío.
	if existing := findJavaBinary(destination); existing != "" {
		if detected, err := DetectMajor(existing); err == nil && detected == major {
			logx.Detail("Java %d ya estaba descargado en '%s'.", major, existing)
			return existing, nil
		}
	}

	if err := os.MkdirAll(destination, 0755); err != nil {
		return "", fmt.Errorf("no se pudo crear '%s': %w", destination, err)
	}

	archivePath := filepath.Join(destination, "jdk.zip")
	logx.Info("Descargando Java %d desde Adoptium...", major)
	if err := httpx.DownloadVerified(url, archivePath, "sha256", sha256Hex); err != nil {
		return "", fmt.Errorf("no se pudo descargar Java %d: %w", major, err)
	}
	defer os.Remove(archivePath)

	logx.Info("Descomprimiendo Java %d...", major)
	if err := unzip(archivePath, destination); err != nil {
		return "", fmt.Errorf("no se pudo descomprimir Java %d: %w", major, err)
	}

	javaPath := findJavaBinary(destination)
	if javaPath == "" {
		return "", fmt.Errorf("se descomprimió Java %d pero no se encontró %s dentro de '%s'", major, binaryName(), destination)
	}

	detected, err := DetectMajor(javaPath)
	if err != nil {
		return "", fmt.Errorf("el Java descargado no se puede ejecutar: %w", err)
	}
	if detected != major {
		return "", fmt.Errorf("se pidió Java %d pero el descargado reporta %d", major, detected)
	}

	logx.Success("Java %d instalado en '%s'.", major, javaPath)
	return javaPath, nil
}

func adoptiumArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("arquitectura no soportada por la descarga automática: %s", runtime.GOARCH)
	}
}

// findJavaBinary ubica el ejecutable dentro del árbol descomprimido. El zip de
// Adoptium envuelve todo en un directorio con la versión en el nombre
// (jdk-17.0.20+8/), así que la ruta no se puede hardcodear.
func findJavaBinary(root string) string {
	patterns := []string{
		filepath.Join(root, "bin", binaryName()),
		filepath.Join(root, "*", "bin", binaryName()),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			continue
		}
		return matches[0]
	}
	return ""
}

func unzip(archivePath string, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Prefijo con separador al final: sin él, "destino-otro" pasaría el chequeo.
	safePrefix := filepath.Clean(destination) + string(os.PathSeparator)

	for _, entry := range reader.File {
		target := filepath.Join(destination, filepath.FromSlash(entry.Name))

		// Zip-slip: una entrada con ../ escribiría fuera del destino.
		if !strings.HasPrefix(target, safePrefix) {
			return fmt.Errorf("entrada fuera del directorio destino: %s", entry.Name)
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := extractZipEntry(entry, target); err != nil {
			return fmt.Errorf("extrayendo '%s': %w", entry.Name, err)
		}
	}

	return nil
}

// extractZipEntry está separada para que los Close sucedan por entrada y no se
// acumulen hasta el final del unzip: un JDK trae más de 20.000 archivos.
func extractZipEntry(entry *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	source, err := entry.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	// Los JDK traen archivos con modo read-only, y en Windows abrir uno existente
	// para escritura falla con "Access is denied". Borrarlo primero hace que una
	// extracción parcial se pueda reintentar.
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}

	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, entry.Mode())
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, source)
	return err
}
