package downloader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Downloader struct {
	serverDir string
}

type DownloadResult struct {
	LoaderType string
	MCVersion  string
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

	destinationPath := filepath.Join(d.serverDir, filename)
	if filename == "playit.exe" {
		destinationPath = filename
	}

	fmt.Printf("[*] Downloading from: %s\n", url)

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

	fmt.Println("\n[*] Download completed.")
	return nil
}

func (d *Downloader) DownloadPaper(version string) error {
	fmt.Printf("[*] Searching latest Paper build for %s...\n", version)

	paperAPIBaseURL := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s", version)

	var data PaperBuildsResponse
	if err := getJSON(paperAPIBaseURL, &data); err != nil {
		return err
	}

	if len(data.Builds) == 0 {
		return fmt.Errorf("no builds found for version %s", version)
	}

	latestBuild := data.Builds[len(data.Builds)-1]
	jarFileName := fmt.Sprintf("paper-%s-%d.jar", version, latestBuild)
	jarDownloadURL := fmt.Sprintf("%s/builds/%d/downloads/%s", paperAPIBaseURL, latestBuild, jarFileName)

	return d.DownloadFile(jarDownloadURL, "server.jar")
}

func (d *Downloader) DownloadFabric(version string) error {
	fmt.Printf("[*] Fetching Fabric installer for %s...\n", version)

	var loaders []FabricLoader
	if err := getJSON("https://meta.fabricmc.net/v2/versions/loader", &loaders); err != nil {
		return fmt.Errorf("error fetching loaders: %w", err)
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
		return fmt.Errorf("error fetching installers: %w", err)
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

	fmt.Printf("    -> Loader: %s | Installer: %s\n", loaderVersion, installerVersion)

	jarDownloadURL := fmt.Sprintf(
		"https://meta.fabricmc.net/v2/versions/loader/%s/%s/%s/server/jar",
		version, loaderVersion, installerVersion,
	)

	return d.DownloadFile(jarDownloadURL, "server.jar")
}

func (d *Downloader) DownloadVanilla(version string) error {
	versionManifestURL := "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

	var manifest MojangManifest
	if err := getJSON(versionManifestURL, &manifest); err != nil {
		return err
	}

	var versionDetailsURL string
	for _, manifestVersion := range manifest.Versions {
		if manifestVersion.ID == version {
			versionDetailsURL = manifestVersion.URL
			break
		}
	}

	if versionDetailsURL == "" {
		return fmt.Errorf("version %s not found in Mojang manifest", version)
	}

	var details MojangVersionDetails
	if err := getJSON(versionDetailsURL, &details); err != nil {
		return err
	}

	serverJarURL := details.Downloads.Server.URL
	if serverJarURL == "" {
		return fmt.Errorf("no server download available for %s", version)
	}

	return d.DownloadFile(serverJarURL, "server.jar")
}

func (d *Downloader) DownloadPlayit() error {
	fmt.Println("[*] Downloading Playit.gg Agent...")
	url := "https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-windows-x86_64.exe"
	return d.DownloadFile(url, "playit.exe")
}

func (d *Downloader) PromptUser() *DownloadResult {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("   AUTOMATIC INSTALLATION SELECTOR")
	fmt.Println(strings.Repeat("=", 40))

	fmt.Print("[?] Enter Minecraft version (e.g., 1.20.1): ")
	version, _ := reader.ReadString('\n')
	version = strings.TrimSpace(version)

	if version == "" {
		fmt.Println("[!] Version cannot be empty.")
		return nil
	}

	fmt.Printf("\nSelect server type for %s:\n", version)
	fmt.Println("1) Paper")
	fmt.Println("2) Fabric")
	fmt.Println("3) Vanilla")
	fmt.Println("4) Cancel")

	fmt.Print("\n[?] Option [1-4]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var err error
	var loaderType string

	switch choice {
	case "1":
		loaderType = "paper"
		err = d.DownloadPaper(version)
	case "2":
		loaderType = "fabric"
		err = d.DownloadFabric(version)
	case "3":
		loaderType = "vanilla"
		err = d.DownloadVanilla(version)
	default:
		fmt.Println("[*] Cancelled.")
		return nil
	}

	if err != nil {
		fmt.Printf("\n[-] Error installing server: %v\n", err)
		return nil
	}

	fmt.Printf("[+] Success! 'server.jar' installed for version %s.\n", version)
	return &DownloadResult{LoaderType: loaderType, MCVersion: version}
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
