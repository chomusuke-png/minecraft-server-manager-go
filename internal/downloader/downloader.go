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
	// LaunchArgs queda vacío salvo para loaders que no producen un jar ejecutable
	// (Forge >= 1.17), donde reemplaza al '-jar server.jar' del flujo normal.
	LaunchArgs []string
	// JavaPath es el runtime que se validó contra la versión de Minecraft, que no
	// necesariamente es el java_path global de config.json.
	JavaPath string
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

// DownloadFileVerified es DownloadFile pero verificando el contenido contra un
// checksum conocido (algo: "sha1"/"sha256", expectedHex vacío = sin verificar).
// Ver httpx.DownloadVerified para el detalle de por qué esto importa: todo lo
// que pasa por acá se termina ejecutando (jars, instaladores) en la máquina del
// usuario.
func (d *Downloader) DownloadFileVerified(url, filename, algo, expectedHex string) error {
	if err := os.MkdirAll(d.serverDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return httpx.DownloadVerified(url, filepath.Join(d.serverDir, filename), algo, expectedHex)
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

	// El endpoint de versión solo da el número de build; el nombre real del jar
	// y su sha256 viven en el endpoint del build puntual.
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

	// A diferencia de Paper/Vanilla/Forge, Fabric arma este jar al vuelo por
	// combinación de versión+loader+installer y no publica un checksum para
	// esa combinación puntual en ningún endpoint de la API. Sigue siendo HTTPS
	// directo contra la fuente oficial (sin mirror de por medio), pero sin la
	// verificación de integridad que sí tienen los demás loaders.
	if err := d.DownloadFile(jarDownloadURL, "server.jar"); err != nil {
		return "", err
	}
	return loaderVersion, nil
}

// DownloadForge descarga el instalador, lo ejecuta con --installServer y devuelve
// la versión del loader junto a los argumentos de arranque resultantes. Ver
// resolveForgeLaunch para por qué los args pueden venir vacíos.
func (d *Downloader) DownloadForge(version string) (string, []string, error) {
	logx.Info("Fetching Forge installer for %s...", version)

	var promos ForgePromotions
	if err := getJSON("https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json", &promos); err != nil {
		return "", nil, fmt.Errorf("error fetching Forge promotions: %w", err)
	}

	// Intenta obtener la versión recomendada y luego la más reciente.
	forgeVersion := promos.Promos[version+"-recommended"]
	if forgeVersion == "" {
		forgeVersion = promos.Promos[version+"-latest"]
	}

	if forgeVersion == "" {
		return "", nil, fmt.Errorf("no Forge version found for Minecraft %s", version)
	}

	logx.Detail("Forge Version: %s", forgeVersion)

	fullVersion := fmt.Sprintf("%s-%s", version, forgeVersion)

	downloadURL := fmt.Sprintf(
		"https://maven.minecraftforge.net/net/minecraftforge/forge/%[1]s/forge-%[1]s-installer.jar",
		fullVersion,
	)

	// Maven publica un sidecar '<artefacto>.sha1' junto a cada jar. Es best
	// effort: si por lo que sea no está (versiones viejas, hiccup del maven),
	// se sigue sin verificar en vez de bloquear una instalación que hoy
	// funciona.
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
		// El instalador quedó a medias: no dejarlo tirado en la instancia.
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

	if err := d.DownloadFileVerified(serverJarURL, "server.jar", "sha1", details.Downloads.Server.SHA1); err != nil {
		return "", err
	}
	return "", nil
}

// playitAssetName devuelve el nombre del asset de playit-agent para este
// SO/arquitectura, tal como lo publica su release de GitHub.
func playitAssetName() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return "playit-windows-x86_64.exe", nil
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "playit-linux-amd64", nil
		case "arm64":
			return "playit-linux-arm64", nil
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

	logx.Info("Downloading Playit.gg Agent...")
	url := "https://github.com/playit-cloud/playit-agent/releases/latest/download/" + assetName

	if dir := filepath.Dir(playitPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// GitHub expone en su API el digest sha256 que calculó al recibir el
	// asset. Es best effort: si la consulta falla (rate limit sin auth, asset
	// viejo sin digest) se descarga igual, solo que sin verificar.
	algo, expectedHex := "", ""
	if hex, err := fetchPlayitSHA256(assetName); err == nil {
		algo, expectedHex = "sha256", hex
	} else {
		logx.Warn("No se pudo obtener el checksum de Playit, se descarga sin verificar: %v", err)
	}

	if err := httpx.DownloadVerified(url, playitPath, algo, expectedHex); err != nil {
		return err
	}

	// A diferencia del .exe de Windows, el binario de Linux no llega con el
	// bit de ejecución puesto.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(playitPath, 0755); err != nil {
			return fmt.Errorf("no se pudo marcar '%s' como ejecutable: %w", playitPath, err)
		}
	}

	return nil
}

// fetchPlayitSHA256 consulta el último release en la API de GitHub y devuelve
// el digest sha256 del asset pedido, tal como lo calculó GitHub al recibir
// el archivo.
func fetchPlayitSHA256(assetName string) (string, error) {
	var release struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := getJSON("https://api.github.com/repos/playit-cloud/playit-agent/releases/latest", &release); err != nil {
		return "", err
	}

	for _, asset := range release.Assets {
		if asset.Name != assetName {
			continue
		}
		hex, ok := strings.CutPrefix(asset.Digest, "sha256:")
		if !ok || hex == "" {
			return "", fmt.Errorf("release de playit sin digest sha256 para '%s'", assetName)
		}
		return hex, nil
	}
	return "", fmt.Errorf("asset '%s' no encontrado en el último release de playit", assetName)
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
	fmt.Println("3) Forge")
	fmt.Println("4) Vanilla")
	fmt.Println("5) Cancel")

	choice, ok := prompt.Loop(reader, "\n[?] Option [1-5]: ", func(input string) (string, bool, string) {
		switch input {
		case "1", "2", "3", "4", "5":
			return input, true, ""
		}
		return "", false, "Entrada incorrecta, reintente."
	})
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return nil
	}

	loaderTypes := map[string]string{"1": "paper", "2": "fabric", "3": "forge", "4": "vanilla"}
	loaderType, isLoader := loaderTypes[choice]
	if !isLoader {
		logx.Info("Cancelled.")
		return nil
	}

	// Se resuelve antes de descargar porque el instalador de Forge corre con este
	// mismo Java, y porque no tiene sentido bajar 300MB de servidor para descubrir
	// después que no hay runtime capaz de levantarlo.
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
	case "vanilla":
		loaderVersion, err = d.DownloadVanilla(version)
	}

	if err != nil {
		logx.Error("\nError installing server: %v", err)
		return nil
	}

	if len(launchArgs) > 0 {
		logx.Success("Success! %s %s installed (arranca vía args file, sin server.jar).", loaderType, version)
	} else {
		logx.Success("Success! 'server.jar' installed for version %s.", version)
	}

	return &DownloadResult{
		LoaderType:    loaderType,
		MCVersion:     version,
		LoaderVersion: loaderVersion,
		LaunchArgs:    launchArgs,
		JavaPath:      d.javaPath,
	}
}

// resolveJava fija el runtime que usará esta instancia, descargándolo si hace
// falta. Muta d.javaPath para que el instalador de Forge lo herede.
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
