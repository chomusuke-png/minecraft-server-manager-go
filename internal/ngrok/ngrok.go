package ngrok

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/logx"
)

func assetURL() (string, error) {
	var arch string
	switch runtime.GOARCH {
	case "amd64", "arm64":
		arch = runtime.GOARCH
	default:
		return "", fmt.Errorf("ngrok no publica un binario para %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-windows-%s.zip", arch), nil
	case "linux":
		return fmt.Sprintf("https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-%s.tgz", arch), nil
	default:
		return "", fmt.Errorf("descarga automática de ngrok no soportada en %s", runtime.GOOS)
	}
}

func Download(ngrokPath string) error {
	url, err := assetURL()
	if err != nil {
		return err
	}

	logx.Info("Downloading ngrok...")

	if dir := filepath.Dir(ngrokPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creando el directorio: %w", err)
		}
	}

	tmpFile, err := os.CreateTemp("", "ngrok-download-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := httpx.Download(url, tmpPath); err != nil {
		return err
	}

	if strings.HasSuffix(url, ".zip") {
		err = extractFirstFileFromZip(tmpPath, ngrokPath)
	} else {
		err = extractFirstFileFromTarGz(tmpPath, ngrokPath)
	}
	if err != nil {
		return fmt.Errorf("no se pudo extraer ngrok: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(ngrokPath, 0755); err != nil {
			return fmt.Errorf("no se pudo marcar '%s' como ejecutable: %w", ngrokPath, err)
		}
	}

	return nil
}

func extractFirstFileFromZip(archivePath, destPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if len(reader.File) == 0 {
		return fmt.Errorf("zip vacío")
	}

	src, err := reader.File[0].Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func extractFirstFileFromTarGz(archivePath, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	header, err := tarReader.Next()
	if err != nil {
		return err
	}
	if header.Typeflag != tar.TypeReg {
		return fmt.Errorf("se esperaba un archivo regular en el tar, se encontró %v", header.Typeflag)
	}

	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, tarReader)
	return err
}

type Tunnel struct {
	cmd       *exec.Cmd
	logFile   *os.File
	PublicURL string
}

func Start(instanceDir, ngrokPath, authToken string, localPort int) (*Tunnel, error) {
	if authToken == "" {
		return nil, fmt.Errorf(
			"falta ngrok_authtoken en config.json (conseguí uno en https://dashboard.ngrok.com/get-started/your-authtoken)",
		)
	}

	absPath, err := filepath.Abs(ngrokPath)
	if err != nil {
		return nil, err
	}

	webPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("no se pudo reservar un puerto para la API local de ngrok: %w", err)
	}

	configPath := filepath.Join(instanceDir, "ngrok.yml")
	configContent := fmt.Sprintf("version: \"3\"\nagent:\n  web_addr: 127.0.0.1:%d\n", webPort)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return nil, err
	}

	cmd := exec.Command(
		absPath, "tcp", strconv.Itoa(localPort),
		"--authtoken", authToken,
		"--config", configPath,
	)
	cmd.Dir = instanceDir

	logFile, logErr := os.OpenFile(filepath.Join(instanceDir, "ngrok.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if logErr != nil {
		logx.Warn("No se pudo abrir ngrok.log: %v", logErr)
	} else {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, fmt.Errorf("error al lanzar ngrok: %w", err)
	}

	publicURL, err := waitForPublicURL(webPort, 20*time.Second)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		if logFile != nil {
			logFile.Close()
		}
		return nil, err
	}

	return &Tunnel{cmd: cmd, logFile: logFile, PublicURL: publicURL}, nil
}

func (t *Tunnel) Stop() {
	if t == nil || t.cmd == nil || t.cmd.Process == nil {
		return
	}
	logx.Info("Cerrando ngrok...")
	t.cmd.Process.Kill()
	t.cmd.Wait()
	if t.logFile != nil {
		t.logFile.Close()
	}
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

type ngrokEndpointsResponse struct {
	Endpoints []struct {
		URL string `json:"url"`
	} `json:"endpoints"`
}

func waitForPublicURL(webPort int, timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/endpoints", webPort)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(apiURL)
		if err == nil {
			var parsed ngrokEndpointsResponse
			decodeErr := json.NewDecoder(resp.Body).Decode(&parsed)
			resp.Body.Close()
			if decodeErr == nil && len(parsed.Endpoints) > 0 && parsed.Endpoints[0].URL != "" {
				return parsed.Endpoints[0].URL, nil
			}
		}
		time.Sleep(400 * time.Millisecond)
	}

	return "", fmt.Errorf("ngrok no levantó el túnel a tiempo (revisá ngrok.log en la instancia)")
}
