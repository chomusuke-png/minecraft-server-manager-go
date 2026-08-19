package downloader

import (
	"bufio"
	"fmt"
	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/java"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Downloader struct {
	serverDir string
	javaPath  string
}

type DownloadResult struct {
	LoaderType    string
	MCVersion     string
	LoaderVersion string
	LaunchArgs    []string
	JavaPath      string
}

func New(serverDir string, javaPath string) *Downloader {
	return &Downloader{
		serverDir: serverDir,
		javaPath:  javaPath,
	}
}

func (d *Downloader) DownloadFile(url string, filename string) error {
	return d.DownloadFileVerified(url, filename, "", "")
}

func (d *Downloader) DownloadFileVerified(url, filename, algo, expectedHex string) error {
	if err := os.MkdirAll(d.serverDir, 0755); err != nil {
		return fmt.Errorf("error creando el directorio: %w", err)
	}

	return httpx.DownloadVerified(url, filepath.Join(d.serverDir, filename), algo, expectedHex)
}

func (d *Downloader) DownloadPaper(version string) (string, error) {
	logx.Info("Buscando la última build de Paper para %s...", version)

	paperAPIBaseURL := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s", version)

	var data PaperBuildsResponse
	if err := getJSON(paperAPIBaseURL, &data); err != nil {
		return "", err
	}

	if len(data.Builds) == 0 {
		return "", fmt.Errorf("no se encontraron builds para la versión %s", version)
	}

	latestBuild := data.Builds[len(data.Builds)-1]

	buildDetailsURL := fmt.Sprintf("%s/builds/%d", paperAPIBaseURL, latestBuild)
	var buildDetails PaperBuildDetails
	if err := getJSON(buildDetailsURL, &buildDetails); err != nil {
		return "", fmt.Errorf("error obteniendo detalles del build %d: %w", latestBuild, err)
	}

	jarFileName := buildDetails.Downloads.Application.Name
	if jarFileName == "" {
		jarFileName = fmt.Sprintf("paper-%s-%d.jar", version, latestBuild)
	}
	jarDownloadURL := fmt.Sprintf("%s/downloads/%s", buildDetailsURL, jarFileName)

	if err := d.DownloadFileVerified(jarDownloadURL, "server.jar", "sha256", buildDetails.Downloads.Application.SHA256); err != nil {
		return "", err
	}
	return strconv.Itoa(latestBuild), nil
}

func (d *Downloader) DownloadFabric(version string) (string, error) {
	logx.Info("Buscando el instalador de Fabric para %s...", version)

	var loaders []FabricLoader
	if err := getJSON("https://meta.fabricmc.net/v2/versions/loader", &loaders); err != nil {
		return "", fmt.Errorf("error obteniendo los loaders de Fabric: %w", err)
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
		return "", fmt.Errorf("error obteniendo los instaladores de Fabric: %w", err)
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

	logx.Detail("Loader: %s | Instalador: %s", loaderVersion, installerVersion)

	jarDownloadURL := fmt.Sprintf(
		"https://meta.fabricmc.net/v2/versions/loader/%s/%s/%s/server/jar",
		version, loaderVersion, installerVersion,
	)

	if err := d.DownloadFile(jarDownloadURL, "server.jar"); err != nil {
		return "", err
	}
	return loaderVersion, nil
}

func (d *Downloader) DownloadForge(version string) (string, []string, error) {
	logx.Info("Buscando el instalador de Forge para %s...", version)

	var promos ForgePromotions
	if err := getJSON("https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json", &promos); err != nil {
		return "", nil, fmt.Errorf("error obteniendo las versiones publicadas de Forge: %w", err)
	}

	forgeVersion := promos.Promos[version+"-latest"]
	if forgeVersion == "" {
		forgeVersion = promos.Promos[version+"-recommended"]
	}

	if forgeVersion == "" {
		return "", nil, fmt.Errorf("no se encontró una versión de Forge para Minecraft %s", version)
	}

	logx.Detail("Versión de Forge: %s", forgeVersion)

	fullVersion := fmt.Sprintf("%s-%s", version, forgeVersion)

	downloadURL := fmt.Sprintf(
		"https://maven.minecraftforge.net/net/minecraftforge/forge/%[1]s/forge-%[1]s-installer.jar",
		fullVersion,
	)

	sha1Hex := ""
	if sidecar, err := httpx.GetText(downloadURL + ".sha1"); err == nil {
		if fields := strings.Fields(sidecar); len(fields) > 0 {
			sha1Hex = fields[0]
		}
	} else {
		logx.Warn("No se pudo obtener el checksum del instalador de Forge, se descarga sin verificar: %v", err)
	}

	if err := d.DownloadFileVerified(downloadURL, forgeInstallerName, "sha1", sha1Hex); err != nil {
		return "", nil, err
	}

	launchArgs, err := d.installForge(fullVersion)
	if err != nil {
		d.removeForgeInstaller()
		return "", nil, err
	}

	return forgeVersion, launchArgs, nil
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
		return "", fmt.Errorf("la versión %s no figura en el manifest de Mojang", version)
	}

	var details MojangVersionDetails
	if err := getJSON(versionDetailsURL, &details); err != nil {
		return "", err
	}

	serverJarURL := details.Downloads.Server.URL
	if serverJarURL == "" {
		return "", fmt.Errorf("no hay descarga de servidor disponible para %s", version)
	}

	if err := d.DownloadFileVerified(serverJarURL, "server.jar", "sha1", details.Downloads.Server.SHA1); err != nil {
		return "", err
	}
	return "", nil
}

const playitPinnedVersion = "v0.17.1"

func playitAssetName() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return "playit-windows-x86_64.exe", nil
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "playit-linux-amd64", nil
		case "arm64":
			return "playit-linux-aarch64", nil
		default:
			return "", fmt.Errorf("playit no publica un binario para linux/%s", runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("descarga automática de playit no soportada en %s", runtime.GOOS)
	}
}

func (d *Downloader) DownloadPlayit(playitPath string) error {
	assetName, err := playitAssetName()
	if err != nil {
		return err
	}

	logx.Info("Descargando el agente de Playit.gg...")
	url := fmt.Sprintf(
		"https://github.com/playit-cloud/playit-agent/releases/download/%s/%s",
		playitPinnedVersion, assetName,
	)

	if dir := filepath.Dir(playitPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creando el directorio: %w", err)
		}
	}

	algo, expectedHex := "", ""
	if hex, err := fetchPlayitSHA256(assetName); err == nil {
		algo, expectedHex = "sha256", hex
	} else {
		logx.Warn("No se pudo obtener el checksum de Playit, se descarga sin verificar: %v", err)
	}

	if err := httpx.DownloadVerified(url, playitPath, algo, expectedHex); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(playitPath, 0755); err != nil {
			return fmt.Errorf("no se pudo marcar '%s' como ejecutable: %w", playitPath, err)
		}
	}

	return nil
}

func fetchPlayitSHA256(assetName string) (string, error) {
	var release struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	url := fmt.Sprintf("https://api.github.com/repos/playit-cloud/playit-agent/releases/tags/%s", playitPinnedVersion)
	if err := getJSON(url, &release); err != nil {
		return "", err
	}

	for _, asset := range release.Assets {
		if asset.Name != assetName {
			continue
		}
		hex, ok := strings.CutPrefix(asset.Digest, "sha256:")
		if !ok || hex == "" {
			return "", fmt.Errorf("release %s de playit sin digest sha256 para '%s'", playitPinnedVersion, assetName)
		}
		return hex, nil
	}
	return "", fmt.Errorf("asset '%s' no encontrado en el release %s de playit", assetName, playitPinnedVersion)
}

func (d *Downloader) PromptUser(reader *bufio.Reader) *DownloadResult {
	version, ok := prompt.Loop(reader, "[?] Introduzca la versión de Minecraft a descargar (e.g., 1.20.1): ", func(input string) (string, bool, string) {
		if input == "" {
			return "", false, "Entrada incorrecta, reintente."
		}
		return input, true, ""
	})
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return nil
	}

	fmt.Printf("\n[?] Tipo de servidor para %s:\n", version)
	fmt.Println("  1) Paper")
	fmt.Println("  2) Fabric")
	fmt.Println("  3) Forge")
	fmt.Println("  4) NeoForge")
	fmt.Println("  5) Vanilla")
	fmt.Println("  6) Cancelar")

	choice, ok := prompt.Loop(reader, "\n[?] Opción [1-6]: ", func(input string) (string, bool, string) {
		switch input {
		case "1", "2", "3", "4", "5", "6":
			return input, true, ""
		}
		return "", false, "Entrada incorrecta, reintente."
	})
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return nil
	}

	loaderTypes := map[string]string{"1": "paper", "2": "fabric", "3": "forge", "4": "neoforge", "5": "vanilla"}
	loaderType, isLoader := loaderTypes[choice]
	if !isLoader {
		logx.Info("Cancelado.")
		return nil
	}

	if err := d.resolveJava(reader, loaderType, version); err != nil {
		logx.Error("\n%v", err)
		return nil
	}

	var err error
	var loaderVersion string
	var launchArgs []string

	switch loaderType {
	case "paper":
		loaderVersion, err = d.DownloadPaper(version)
	case "fabric":
		loaderVersion, err = d.DownloadFabric(version)
	case "forge":
		loaderVersion, launchArgs, err = d.DownloadForge(version)
	case "neoforge":
		loaderVersion, launchArgs, err = d.DownloadNeoForge(version)
	case "vanilla":
		loaderVersion, err = d.DownloadVanilla(version)
	}

	if err != nil {
		logx.Error("\nError instalando el servidor: %v", err)
		return nil
	}

	if len(launchArgs) > 0 {
		logx.Success("%s %s instalado (arranca vía args file, sin server.jar).", loaderType, version)
	} else {
		logx.Success("'server.jar' instalado para la versión %s.", version)
	}

	return &DownloadResult{
		LoaderType:    loaderType,
		MCVersion:     version,
		LoaderVersion: loaderVersion,
		LaunchArgs:    launchArgs,
		JavaPath:      d.javaPath,
	}
}

func (d *Downloader) resolveJava(reader *bufio.Reader, loaderType string, mcVersion string) error {
	resolved, err := java.Resolve(reader, java.Require(mcVersion), d.javaPath)
	if err != nil {
		return err
	}
	d.javaPath = resolved
	return nil
}

func getJSON(url string, target interface{}) error {
	return httpx.GetJSON(url, target)
}
