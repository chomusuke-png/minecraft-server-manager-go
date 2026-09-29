package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// DetectedLoader es lo que se puede reconstruir de una instalacion de Forge o
// NeoForge a partir de los archivos que deja el instalador
type DetectedLoader struct {
	// LoaderType y LoaderVersion quedan vacios si el args file esta fuera del
	// layout conocido: el comando sirve igual para arrancar, pero no se sabe de
	// que loader es
	LoaderType    string
	MCVersion     string
	LoaderVersion string
	ArgsFile      string
	LaunchArgs    []string
}

var forgeLikeSpecs = []forgeLikeSpec{forgeSpec, neoForgeSpec}

// el args file termina con los argumentos del programa, entre ellos la version
// de minecraft para la que se instalo el loader
var mcVersionFlagPattern = regexp.MustCompile(`--fml\.mcVersion\s+(\S+)`)

// DetectInstalledLoader busca una instalacion de Forge o NeoForge en la carpeta
// sin depender de instance.json. Devuelve false si no hay ninguna, o si hay
// varias y no se puede saber cual es la actual
func DetectInstalledLoader(serverDir string) (*DetectedLoader, bool) {
	argsFile := argsFileFromRunScripts(serverDir)
	if argsFile == "" {
		// sin el script del instalador no hay forma de saber cual es la actual
		// si quedaron restos de una version anterior
		candidates := argsFilesInLibraries(serverDir)
		if len(candidates) != 1 {
			return nil, false
		}
		argsFile = candidates[0]
	}

	detected := &DetectedLoader{
		ArgsFile:   argsFile,
		LaunchArgs: forgeLikeLaunchArgs(serverDir, argsFile),
	}
	detected.LoaderType, detected.MCVersion, detected.LoaderVersion = identifyArgsFile(argsFile)

	// la version que declara el args file es mas confiable que la deducida de
	// la ruta, sobre todo en NeoForge, donde no aparece literal
	if content, err := os.ReadFile(filepath.Join(serverDir, argsFile)); err == nil {
		if match := mcVersionFlagPattern.FindSubmatch(content); match != nil {
			detected.MCVersion = string(match[1])
		}
	}

	return detected, true
}

// argsFilesInLibraries lista los args files del SO actual que hay en el layout
// canonico de cada loader, uno por version instalada
func argsFilesInLibraries(serverDir string) []string {
	var found []string
	for _, spec := range forgeLikeSpecs {
		groupParts := append([]string{"libraries"}, spec.libraryGroup...)
		entries, err := os.ReadDir(filepath.Join(append([]string{serverDir}, groupParts...)...))
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			parts := append(append([]string{}, groupParts...), entry.Name(), platformArgsFileName())
			candidate := filepath.ToSlash(filepath.Join(parts...))
			if fileExists(filepath.Join(serverDir, candidate)) {
				found = append(found, candidate)
			}
		}
	}
	return found
}

// identifyArgsFile deduce el loader y sus versiones de la ruta del args file,
// que en el layout canonico es libraries/<grupo>/<version>/<args file>
func identifyArgsFile(argsFile string) (loaderType, mcVersion, loaderVersion string) {
	for _, spec := range forgeLikeSpecs {
		prefix := "libraries/" + strings.Join(spec.libraryGroup, "/") + "/"
		rest, ok := strings.CutPrefix(argsFile, prefix)
		if !ok {
			continue
		}

		versionDir, fileName, ok := strings.Cut(rest, "/")
		if !ok || versionDir == "" || strings.Contains(fileName, "/") {
			return spec.loaderType, "", ""
		}

		if spec.loaderType == "neoforge" {
			return spec.loaderType, neoForgeMCVersion(versionDir), versionDir
		}

		// forge nombra la carpeta <mc>-<forge>
		mc, forge, ok := strings.Cut(versionDir, "-")
		if !ok || mc == "" || forge == "" {
			return spec.loaderType, "", ""
		}
		return spec.loaderType, mc, forge
	}
	return "", "", ""
}

// neoForgeMCVersion es la inversa de neoForgeVersionPrefix. Con tres numeros es
// el esquema viejo: 21.1.x es para Minecraft 1.21.1 y 21.0.x para 1.21. Con
// cuatro es la numeracion por año: 26.3.0.x es para 26.3 y 26.1.2.x para 26.1.2
func neoForgeMCVersion(neoForgeVersion string) string {
	parts := strings.Split(neoForgeVersion, ".")
	if len(parts) < 2 {
		return ""
	}

	first, err := strconv.Atoi(parts[0])
	if err != nil {
		return ""
	}
	second, err := strconv.Atoi(parts[1])
	if err != nil {
		return ""
	}

	// en el esquema viejo el tercer numero es el build y puede traer sufijo
	// (20.4.80-beta); en el nuevo es el patch de minecraft y siempre es numero
	if len(parts) >= 4 {
		patch, err := strconv.Atoi(parts[2])
		if err != nil {
			return ""
		}
		if patch == 0 {
			return fmt.Sprintf("%d.%d", first, second)
		}
		return fmt.Sprintf("%d.%d.%d", first, second, patch)
	}

	if second == 0 {
		return fmt.Sprintf("1.%d", first)
	}
	return fmt.Sprintf("1.%d.%d", first, second)
}
