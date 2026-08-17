package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"minecraft-manager/internal/httpx"
)

const releasesAPI = "https://api.github.com/repos/chomusuke-png/minecraft-server-manager-go/releases/latest"

type Release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name            string `json:"name"`
		BrowserDownload string `json:"browser_download_url"`
		Digest          string `json:"digest"`
	} `json:"assets"`
}

// FetchLatest consulta el último release publicado en GitHub
func FetchLatest() (*Release, error) {
	var rel Release
	if err := httpx.GetJSON(releasesAPI, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// IsNewer compara dos tags "vMAJOR.MINOR.PATCH". Si alguno no se puede interpretar, devuelve false
func IsNewer(candidate, current string) bool {
	cMajor, cMinor, cPatch, ok := parseVersion(candidate)
	if !ok {
		return false
	}
	curMajor, curMinor, curPatch, ok := parseVersion(current)
	if !ok {
		return false
	}

	if cMajor != curMajor {
		return cMajor > curMajor
	}
	if cMinor != curMinor {
		return cMinor > curMinor
	}
	return cPatch > curPatch
}

func parseVersion(tag string) (major, minor, patch int, ok bool) {
	parts := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(tag), "v"), ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}

	var err error
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, 0, false
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, 0, false
	}
	if patch, err = strconv.Atoi(parts[2]); err != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

// assetName devuelve el nombre del binario que corresponde a este SO
func assetName() (string, error) {
	if runtime.GOARCH != "amd64" {
		return "", fmt.Errorf("no hay build automática de msm para %s/%s, compilá desde el código fuente", runtime.GOOS, runtime.GOARCH)
	}

	switch runtime.GOOS {
	case "windows":
		return "msm-windows-amd64.exe", nil
	case "linux":
		return "msm-linux-amd64", nil
	default:
		return "", fmt.Errorf("autoactualización no soportada en %s", runtime.GOOS)
	}
}

func Apply(rel *Release) error {
	name, err := assetName()
	if err != nil {
		return err
	}

	var downloadURL, digest string
	for _, asset := range rel.Assets {
		if asset.Name == name {
			downloadURL, digest = asset.BrowserDownload, asset.Digest
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("el release %s no tiene un binario '%s'", rel.Tag, name)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return err
	}

	dir := filepath.Dir(currentExe)
	newPath := filepath.Join(dir, name+".new")
	oldPath := currentExe + ".old"

	algo, hex := "", ""
	if trimmed, ok := strings.CutPrefix(digest, "sha256:"); ok {
		algo, hex = "sha256", trimmed
	}

	if err := httpx.DownloadVerified(downloadURL, newPath, algo, hex); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(newPath, 0755); err != nil {
			os.Remove(newPath)
			return fmt.Errorf("no se pudo marcar el binario nuevo como ejecutable: %w", err)
		}
	}

	return swap(currentExe, newPath, oldPath)
}

func swap(currentExe, newPath, oldPath string) error {
	os.Remove(oldPath)

	if err := os.Rename(currentExe, oldPath); err != nil {
		return fmt.Errorf("no se pudo mover el binario en uso: %w", err)
	}

	if err := os.Rename(newPath, currentExe); err != nil {
		os.Rename(oldPath, currentExe)
		return fmt.Errorf("no se pudo instalar el binario nuevo: %w", err)
	}

	return nil
}
