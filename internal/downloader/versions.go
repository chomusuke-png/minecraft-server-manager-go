package downloader

import (
	"bufio"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
)

var ErrCancelled = errors.New("descarga cancelada")

var loaderLabels = map[string]string{
	"paper":    "Paper",
	"fabric":   "Fabric",
	"forge":    "Forge",
	"neoforge": "NeoForge",
	"vanilla":  "Vanilla",
}

type loaderVersions struct {
	latest string
	stable string
	// queda vacio en Forge, que publica solo latest y recommended en vez del listado completo
	known []string
}

type versionChoice struct {
	descriptors []string
	value       string
}

func (c versionChoice) label() string {
	return fmt.Sprintf("%s — %s", c.value, joinDescriptors(c.descriptors))
}

func joinDescriptors(descriptors []string) string {
	if len(descriptors) == 1 {
		return descriptors[0]
	}
	last := len(descriptors) - 1
	return fmt.Sprintf("%s y %s", strings.Join(descriptors[:last], ", "), descriptors[last])
}

// ChooseLoaderVersion pregunta que version del loader instalar
// current es la que ya tiene la instancia y solo se ofrece al actualizar
func ChooseLoaderVersion(reader *bufio.Reader, loaderType, mcVersion, current string) (string, error) {
	if loaderType == "vanilla" {
		return "", nil
	}

	label, ok := loaderLabels[loaderType]
	if !ok {
		return "", fmt.Errorf("tipo de loader desconocido: %s", loaderType)
	}

	logx.Info("\nBuscando versiones de %s para %s...", label, mcVersion)
	available, err := resolveLoaderVersions(loaderType, mcVersion)
	if err != nil {
		return "", err
	}

	return promptLoaderVersion(reader, label, available, current)
}

func resolveLoaderVersions(loaderType, mcVersion string) (loaderVersions, error) {
	switch loaderType {
	case "paper":
		return paperVersions(mcVersion)
	case "fabric":
		return fabricVersions()
	case "forge":
		return forgeVersions(mcVersion)
	case "neoforge":
		return neoForgeVersions(mcVersion)
	}
	return loaderVersions{}, fmt.Errorf("tipo de loader desconocido: %s", loaderType)
}

func paperVersions(mcVersion string) (loaderVersions, error) {
	var builds []PaperBuild
	if err := getJSON(paperBuildsURL(mcVersion), &builds); err != nil {
		return loaderVersions{}, fmt.Errorf("error obteniendo las builds de Paper: %w", err)
	}
	return paperVersionsFrom(builds, mcVersion)
}

// paperVersionsFrom espera la lista tal cual la devuelve la API v3: de la build
// mas nueva a la mas vieja
func paperVersionsFrom(builds []PaperBuild, mcVersion string) (loaderVersions, error) {
	if len(builds) == 0 {
		return loaderVersions{}, fmt.Errorf("no se encontraron builds de Paper para la versión %s", mcVersion)
	}

	versions := loaderVersions{latest: strconv.Itoa(builds[0].ID)}
	for _, build := range builds {
		number := strconv.Itoa(build.ID)
		versions.known = append(versions.known, number)
		if build.Channel == paperStableChannel && versions.stable == "" {
			versions.stable = number
		}
	}
	return versions, nil
}

func fabricVersions() (loaderVersions, error) {
	var loaders []FabricLoader
	if err := getJSON(fabricLoadersURL, &loaders); err != nil {
		return loaderVersions{}, fmt.Errorf("error obteniendo los loaders de Fabric: %w", err)
	}
	return fabricVersionsFrom(loaders)
}

func fabricVersionsFrom(loaders []FabricLoader) (loaderVersions, error) {
	if len(loaders) == 0 {
		return loaderVersions{}, errors.New("la API de Fabric no devolvió ningún loader")
	}

	versions := loaderVersions{latest: loaders[0].Version}
	for _, loader := range loaders {
		versions.known = append(versions.known, loader.Version)
		if loader.Stable && versions.stable == "" {
			versions.stable = loader.Version
		}
	}
	return versions, nil
}

func forgeVersions(mcVersion string) (loaderVersions, error) {
	var promos ForgePromotions
	if err := getJSON(forgePromotionsURL, &promos); err != nil {
		return loaderVersions{}, fmt.Errorf("error obteniendo las versiones publicadas de Forge: %w", err)
	}
	return forgeVersionsFrom(promos, mcVersion)
}

func forgeVersionsFrom(promos ForgePromotions, mcVersion string) (loaderVersions, error) {
	versions := loaderVersions{
		latest: promos.Promos[mcVersion+"-latest"],
		stable: promos.Promos[mcVersion+"-recommended"],
	}

	if versions.latest == "" && versions.stable == "" {
		return loaderVersions{}, fmt.Errorf("no se encontró una versión de Forge para Minecraft %s", mcVersion)
	}
	return versions, nil
}

func neoForgeVersions(mcVersion string) (loaderVersions, error) {
	prefix, ok := neoForgeVersionPrefix(mcVersion)
	if !ok {
		return loaderVersions{}, fmt.Errorf("no se pudo interpretar la versión de Minecraft %s", mcVersion)
	}

	var metadata NeoForgeMavenMetadata
	if err := httpx.GetXML(neoForgeMavenMetadataURL, &metadata); err != nil {
		return loaderVersions{}, fmt.Errorf("error obteniendo versiones de NeoForge: %w", err)
	}
	return neoForgeVersionsFrom(metadata.Versioning.Versions.Version, prefix, mcVersion)
}

func neoForgeVersionsFrom(published []string, prefix, mcVersion string) (loaderVersions, error) {
	var versions loaderVersions
	for _, version := range published {
		if !strings.HasPrefix(version, prefix) {
			continue
		}
		versions.known = append(versions.known, version)
		versions.latest = version
		if !isNeoForgePrerelease(version) {
			versions.stable = version
		}
	}

	if versions.latest == "" {
		return loaderVersions{}, fmt.Errorf("no se encontró ninguna versión de NeoForge para Minecraft %s", mcVersion)
	}
	return versions, nil
}

func isNeoForgePrerelease(version string) bool {
	return strings.Contains(version, "-beta") || strings.Contains(version, "-alpha")
}

// versionChoices arma las opciones fijas del menu. Una misma version puede ser
// la actual, la mas reciente y la estable a la vez: ahi se lista una sola vez
// con los tres carteles
func versionChoices(available loaderVersions, current string) []versionChoice {
	var choices []versionChoice

	add := func(descriptor, value string) {
		if value == "" {
			return
		}
		for i := range choices {
			if choices[i].value == value {
				choices[i].descriptors = append(choices[i].descriptors, descriptor)
				return
			}
		}
		choices = append(choices, versionChoice{descriptors: []string{descriptor}, value: value})
	}

	add("la actual", current)
	add("la más reciente", available.latest)
	add("la estable", available.stable)

	return choices
}

func promptLoaderVersion(reader *bufio.Reader, label string, available loaderVersions, current string) (string, error) {
	choices := versionChoices(available, current)
	customOption := len(choices) + 1
	cancelOption := customOption + 1

	fmt.Printf("\n[?] Versión de %s:\n", label)
	for i, choice := range choices {
		fmt.Printf("  %d) %s\n", i+1, choice.label())
	}
	fmt.Printf("  %d) Escribir una versión\n", customOption)
	fmt.Printf("  %d) Cancelar\n", cancelOption)

	promptText := fmt.Sprintf("[?] Opción [1-%d] [1]: ", cancelOption)
	choice := prompt.LoopDefault(reader, promptText, 1, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value < 1 || value > cancelOption {
			return 0, false, fmt.Sprintf("Opción inválida. Elegí un número entre 1 y %d.", cancelOption)
		}
		return value, true, ""
	})

	switch choice {
	case cancelOption:
		return "", ErrCancelled
	case customOption:
		return promptCustomVersion(reader, label, available)
	}
	return choices[choice-1].value, nil
}

func promptCustomVersion(reader *bufio.Reader, label string, available loaderVersions) (string, error) {
	promptText := fmt.Sprintf("[?] Versión de %s", label)
	if available.latest != "" {
		promptText += fmt.Sprintf(" (ej. %s)", available.latest)
	}
	promptText += ": "

	version, ok := prompt.Loop(reader, promptText, func(input string) (string, bool, string) {
		if input == "" {
			return "", false, "Ingresá una versión."
		}
		if len(available.known) > 0 && !slices.Contains(available.known, input) {
			return "", false, fmt.Sprintf("%s no publicó esa versión para esta versión de Minecraft.", label)
		}
		return input, true, ""
	})
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return "", ErrCancelled
	}
	return version, nil
}

// Install descarga el loader ya elegido y devuelve la version instalada junto a
// los args de arranque, que solo traen Forge y NeoForge
func (d *Downloader) Install(loaderType, mcVersion, loaderVersion string) (string, []string, error) {
	switch loaderType {
	case "paper":
		build, err := d.DownloadPaper(mcVersion, loaderVersion)
		return build, nil, err
	case "fabric":
		version, err := d.DownloadFabric(mcVersion, loaderVersion)
		return version, nil, err
	case "forge":
		return d.DownloadForge(mcVersion, loaderVersion)
	case "neoforge":
		return d.DownloadNeoForge(mcVersion, loaderVersion)
	case "vanilla":
		version, err := d.DownloadVanilla(mcVersion)
		return version, nil, err
	}
	return "", nil, fmt.Errorf("tipo de loader desconocido: %s", loaderType)
}
