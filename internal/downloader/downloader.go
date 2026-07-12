package downloader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Downloader struct {
	serverDir string
}

type DownloadResult struct {
	LoaderType    string
	MCVersion     string
	LoaderVersion string
}

func New(serverDir string) *Downloader {
	return &Downloader{
		serverDir: serverDir,
	}
}

func (d *Downloader) DownloadFile(url string, filename string) error {
	if err := os.MkdirAll(d.serverDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return downloadTo(url, filepath.Join(d.serverDir, filename))
}

func downloadTo(url string, destinationPath string) error {
	logx.Info("Downloading from: %s", url)

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status: %s", response.Status)
	}

	outputFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	size := response.ContentLength
	progressReader := &ProgressReader{
		Reader: response.Body,
		Total:  size,
	}

	if _, err = io.Copy(outputFile, progressReader); err != nil {
		return err
	}

	logx.Info("\nDownload completed.")
	return nil
}

func (d *Downloader) DownloadPaper(version string) (string, error) {
	logx.Info("Searching latest Paper build for %s...", version)

	paperAPIBaseURL := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s", version)

	var data PaperBuildsResponse
	if err := getJSON(paperAPIBaseURL, &data); err != nil {
		return "", err
	}

	if len(data.Builds) == 0 {
		return "", fmt.Errorf("no builds found for version %s", version)
	}

	latestBuild := data.Builds[len(data.Builds)-1]
	jarFileName := fmt.Sprintf("paper-%s-%d.jar", version, latestBuild)
	jarDownloadURL := fmt.Sprintf("%s/builds/%d/downloads/%s", paperAPIBaseURL, latestBuild, jarFileName)

	if err := d.DownloadFile(jarDownloadURL, "server.jar"); err != nil {
		return "", err
	}
	return strconv.Itoa(latestBuild), nil
}

func (d *Downloader) DownloadFabric(version string) (string, error) {
	logx.Info("Fetching Fabric installer for %s...", version)

	var loaders []FabricLoader
	if err := getJSON("https://meta.fabricmc.net/v2/versions/loader", &loaders); err != nil {
		return "", fmt.Errorf("error fetching loaders: %w", err)
	}

	loaderVersion := ""
	for _, loader := range loaders {
		if loader.Stable {
			loaderVersion = loader.Version
			break
		}
	}
	if loaderVersion == "" {
		loaderVersion = "0.15.7"
	}

	var installers []FabricInstaller
	if err := getJSON("https://meta.fabricmc.net/v2/versions/installer", &installers); err != nil {
		return "", fmt.Errorf("error fetching installers: %w", err)
	}

	installerVersion := ""
	for _, installer := range installers {
		if installer.Stable {
			installerVersion = installer.Version
			break
		}
	}
	if installerVersion == "" {
		installerVersion = "1.0.0"
	}

	logx.Detail("Loader: %s | Installer: %s", loaderVersion, installerVersion)

	jarDownloadURL := fmt.Sprintf(
		"https://meta.fabricmc.net/v2/versions/loader/%s/%s/%s/server/jar",
		version, loaderVersion, installerVersion,
	)

	if err := d.DownloadFile(jarDownloadURL, "server.jar"); err != nil {
		return "", err
	}
	return loaderVersion, nil
}

func (d *Downloader) DownloadVanilla(version string) (string, error) {
	versionManifestURL := "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

	var manifest MojangManifest
	if err := getJSON(versionManifestURL, &manifest); err != nil {
		return "", err
	}

	var versionDetailsURL string
	for _, manifestVersion := range manifest.Versions {
		if manifestVersion.ID == version {
			versionDetailsURL = manifestVersion.URL
			break
		}
	}

	if versionDetailsURL == "" {
		return "", fmt.Errorf("version %s not found in Mojang manifest", version)
	}

	var details MojangVersionDetails
	if err := getJSON(versionDetailsURL, &details); err != nil {
		return "", err
	}

	serverJarURL := details.Downloads.Server.URL
	if serverJarURL == "" {
		return "", fmt.Errorf("no server download available for %s", version)
	}

	if err := d.DownloadFile(serverJarURL, "server.jar"); err != nil {
		return "", err
	}
	return "", nil
}

func (d *Downloader) DownloadPlayit(playitPath string) error {
	logx.Info("Downloading Playit.gg Agent...")
	url := "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-windows-x86_64.exe"

	if dir := filepath.Dir(playitPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return downloadTo(url, playitPath)
}

func (d *Downloader) PromptUser(reader *bufio.Reader) *DownloadResult {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("   AUTOMATIC INSTALLATION SELECTOR")
	fmt.Println(strings.Repeat("=", 40))

	version, ok := prompt.Loop(reader, "[?] Enter Minecraft version (e.g., 1.20.1): ", func(input string) (string, bool, string) {
		if input == "" {
			return "", false, "Entrada incorrecta, reintente."
		}
		return input, true, ""
	})
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return nil
	}

	fmt.Printf("\nSelect server type for %s:\n", version)
	fmt.Println("1) Paper")
	fmt.Println("2) Fabric")
	fmt.Println("3) Vanilla")
	fmt.Println("4) Cancel")

	choice, ok := prompt.Loop(reader, "\n[?] Option [1-4]: ", func(input string) (string, bool, string) {
		if input == "1" || input == "2" || input == "3" || input == "4" {
			return input, true, ""
		}
		return "", false, "Entrada incorrecta, reintente."
	})
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return nil
	}

	var err error
	var loaderType string
	var loaderVersion string

	switch choice {
	case "1":
		loaderType = "paper"
		loaderVersion, err = d.DownloadPaper(version)
	case "2":
		loaderType = "fabric"
		loaderVersion, err = d.DownloadFabric(version)
	case "3":
		loaderType = "vanilla"
		loaderVersion, err = d.DownloadVanilla(version)
	case "4":
		logx.Info("Cancelled.")
		return nil
	}

	if err != nil {
		logx.Error("\nError installing server: %v", err)
		return nil
	}

	logx.Success("Success! 'server.jar' installed for version %s.", version)
	return &DownloadResult{LoaderType: loaderType, MCVersion: version, LoaderVersion: loaderVersion}
}

func getJSON(url string, target interface{}) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", response.StatusCode)
	}

	return json.NewDecoder(response.Body).Decode(target)
}

type ProgressReader struct {
	Reader io.Reader
	Total  int64
	read   int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.read += int64(n)

	if pr.Total > 0 {
		percent := float64(pr.read) / float64(pr.Total) * 100
		fmt.Printf("\r[*] Progress: %.1f%%", percent)
	}

	return n, err
}
