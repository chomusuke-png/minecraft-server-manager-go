package downloader

import (
	"bufio"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
)

const (
	paperProjectURL = "https://fill.papermc.io/v3/projects/paper"
	fabricGameURL   = "https://meta.fabricmc.net/v2/versions/game"
	quiltGameURL    = "https://meta.quiltmc.org/v3/versions/game"
)

// mcVersions son las versiones de Minecraft que publica un loader, de la mas
// nueva a la mas vieja
type mcVersions struct {
	// solo releases, sin rc, pre ni snapshots: dan el rango y las sugerencias
	stable []string
	// todas las que el loader acepta, para validar una escrita a mano
	known []string
}

// mcVersionResolvers tiene una entrada por loader, vanilla incluido
var mcVersionResolvers = map[string]func() (mcVersions, error){
	"paper":    paperMCVersions,
	"fabric":   func() (mcVersions, error) { return flaggedMCVersions(fabricGameURL, "Fabric") },
	"quilt":    func() (mcVersions, error) { return flaggedMCVersions(quiltGameURL, "Quilt") },
	"forge":    forgeMCVersions,
	"neoforge": neoForgeMCVersions,
	"arclight": arclightMCVersions,
	"vanilla":  vanillaMCVersions,
}

// ChooseMCVersion pregunta la version de Minecraft ya sabiendo el loader, asi
// que valida contra las que ese loader soporta. current es la version que ya
// tiene la instancia y solo se usa al actualizar, si el loader la soporta.
// Si la API no responde, se pide escrita a mano sin validar
func ChooseMCVersion(reader *bufio.Reader, loaderType, current string) (string, error) {
	label, ok := LoaderLabel(loaderType)
	if !ok {
		return "", fmt.Errorf("tipo de loader desconocido: %s", loaderType)
	}

	resolver, ok := mcVersionResolvers[loaderType]
	if !ok {
		return "", fmt.Errorf("tipo de loader desconocido: %s", loaderType)
	}

	logx.Info("\nBuscando las versiones de Minecraft que soporta %s...", label)
	available, err := resolver()
	if err != nil {
		logx.Warn("No se pudo consultar qué versiones soporta %s: %v", label, err)
		return promptFreeMCVersion(reader, current)
	}

	return promptMCVersion(reader, label, available, current)
}

// promptMCVersion pide la version escrita, mostrando el rango que soporta el
// loader. Enter toma la actual al actualizar o la mas reciente al crear, y una
// version que el loader no publico se rechaza en el momento sugiriendo las mas
// cercanas
func promptMCVersion(reader *bufio.Reader, label string, available mcVersions, current string) (string, error) {
	defaultVersion := mcVersionDefault(available, current)
	if current != "" && defaultVersion == "" {
		// no hay default: un Enter distraido cambiaria la version del mundo
		logx.Warn("%s no soporta la versión actual (%s): hay que escribir otra.", label, current)
	}

	promptText := fmt.Sprintf("[?] Versión de Minecraft para %s%s", label, mcVersionRange(available))
	if defaultVersion != "" {
		origin := prompt.OriginNewest
		if current != "" {
			origin = prompt.OriginCurrent
		}
		promptText += prompt.Default(origin, defaultVersion)
	}
	promptText += ": "

	validate := func(input string) (string, bool, string) {
		if input == "" {
			return "", false, "Ingresa una versión."
		}
		if !slices.Contains(available.known, input) {
			return "", false, unsupportedMCVersionMessage(label, input, available)
		}
		return input, true, ""
	}

	if defaultVersion != "" {
		return prompt.LoopDefault(reader, promptText, defaultVersion, validate), nil
	}

	version, ok := prompt.Loop(reader, promptText, validate)
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return "", ErrCancelled
	}
	return version, nil
}

// mcVersionDefault es lo que toma Enter: la actual si el loader la soporta, o
// la mas reciente si se esta creando la instancia. vacio si no hay default
func mcVersionDefault(available mcVersions, current string) string {
	if current != "" {
		if slices.Contains(available.known, current) {
			return current
		}
		return ""
	}
	if len(available.stable) > 0 {
		return available.stable[0]
	}
	return ""
}

// mcVersionRange resume lo que soporta el loader sin listar todo, ej. " (1.7.10 a 26.3)"
func mcVersionRange(available mcVersions) string {
	if len(available.stable) == 0 {
		return ""
	}
	newest := available.stable[0]
	oldest := available.stable[len(available.stable)-1]
	if newest == oldest {
		return fmt.Sprintf(" (sólo %s)", newest)
	}
	return fmt.Sprintf(" (%s a %s)", oldest, newest)
}

func unsupportedMCVersionMessage(label, input string, available mcVersions) string {
	message := fmt.Sprintf("%s no publicó %s.", label, input)
	if closest := closestMCVersions(input, available.stable, 3); len(closest) > 0 {
		message += " Las más cercanas: " + strings.Join(closest, ", ")
	}
	return message
}

// closestMCVersions sugiere las estables vecinas a lo escrito, prefiriendo las
// de la misma linea: para 1.21.12 son 1.21.11, 1.21.10 y 1.21.9 aunque la 26.1
// este mas cerca en la lista. si lo escrito no tiene formato de release, se
// sugieren las mas recientes
func closestMCVersions(input string, stable []string, limit int) []string {
	if !isMCRelease(input) {
		return stable[:min(limit, len(stable))]
	}

	candidates := stable
	line := mcVersionLine(input)
	var sameLine []string
	for _, version := range stable {
		if mcVersionLine(version) == line {
			sameLine = append(sameLine, version)
		}
	}
	if len(sameLine) > 0 {
		candidates = sameLine
	}

	// candidates va de la mas nueva a la mas vieja: se busca donde caeria lo
	// escrito y se toman las que lo rodean
	insertAt := len(candidates)
	for i, version := range candidates {
		if compareMCVersions(version, input) < 0 {
			insertAt = i
			break
		}
	}
	start := max(0, min(insertAt-1, len(candidates)-limit))
	return candidates[start:min(start+limit, len(candidates))]
}

// mcVersionLine son los dos primeros numeros: 1.21 para 1.21.11 y 26.1 para 26.1.2
func mcVersionLine(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1]
}

// promptFreeMCVersion es el camino de siempre, sin lista contra la cual validar
func promptFreeMCVersion(reader *bufio.Reader, current string) (string, error) {
	if current != "" {
		promptText := "[?] Versión de Minecraft" + prompt.Default(prompt.OriginCurrent, current) + ": "
		return prompt.LoopDefault(reader, promptText, current, func(input string) (string, bool, string) {
			return input, true, ""
		}), nil
	}

	version, ok := prompt.Loop(reader, "[?] Versión de Minecraft (ej. 1.20.1): ", func(input string) (string, bool, string) {
		if input == "" {
			return "", false, "Ingresa una versión."
		}
		return input, true, ""
	})
	if !ok {
		logx.Error("\nNo se pudo leer la entrada. Cancelado.")
		return "", ErrCancelled
	}
	return version, nil
}

// una release es solo numeros separados por puntos, en las dos numeraciones:
// 1.21.11 y 26.3. descarta rc, pre, snapshots y las versiones de abril
var releasePattern = regexp.MustCompile(`^\d+(\.\d+)+$`)

func isMCRelease(version string) bool {
	return releasePattern.MatchString(version)
}

// compareMCVersions compara numero por numero, y sirve entre las dos
// numeraciones porque 26 es mayor que 1. solo para releases
func compareMCVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	for i := 0; i < max(len(partsA), len(partsB)); i++ {
		numberA, numberB := 0, 0
		if i < len(partsA) {
			numberA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numberB, _ = strconv.Atoi(partsB[i])
		}
		if numberA != numberB {
			return numberA - numberB
		}
	}
	return 0
}

// sortedMCReleases deja las releases sin repetir, de la mas nueva a la mas vieja
func sortedMCReleases(versions []string) []string {
	var releases []string
	for _, version := range versions {
		if isMCRelease(version) && !slices.Contains(releases, version) {
			releases = append(releases, version)
		}
	}
	slices.SortFunc(releases, func(a, b string) int { return compareMCVersions(b, a) })
	return releases
}

func paperMCVersions() (mcVersions, error) {
	var project PaperProject
	if err := getJSON(paperProjectURL, &project); err != nil {
		return mcVersions{}, err
	}
	return paperMCVersionsFrom(project)
}

// la API agrupa por version mayor en un objeto, que en Go llega sin orden
func paperMCVersionsFrom(project PaperProject) (mcVersions, error) {
	var all []string
	for _, group := range project.Versions {
		all = append(all, group...)
	}
	return releasesAsMCVersions(all, "Paper")
}

// flaggedMCVersions lee el formato de Fabric y Quilt, que ya viene ordenado y
// marca las estables
func flaggedMCVersions(url, label string) (mcVersions, error) {
	var games []FlaggedGameVersion
	if err := getJSON(url, &games); err != nil {
		return mcVersions{}, err
	}
	return flaggedMCVersionsFrom(games, label)
}

func flaggedMCVersionsFrom(games []FlaggedGameVersion, label string) (mcVersions, error) {
	var versions mcVersions
	for _, game := range games {
		versions.known = append(versions.known, game.Version)
		if game.Stable {
			versions.stable = append(versions.stable, game.Version)
		}
	}
	if len(versions.stable) == 0 {
		return mcVersions{}, fmt.Errorf("%s no publicó ninguna versión estable de Minecraft", label)
	}
	return versions, nil
}

func forgeMCVersions() (mcVersions, error) {
	var promos ForgePromotions
	if err := getJSON(forgePromotionsURL, &promos); err != nil {
		return mcVersions{}, err
	}
	return forgeMCVersionsFrom(promos)
}

// las claves de promotions son "<mc>-latest" y "<mc>-recommended"
func forgeMCVersionsFrom(promos ForgePromotions) (mcVersions, error) {
	var all []string
	for key := range promos.Promos {
		version, _, found := strings.Cut(key, "-")
		if found {
			all = append(all, version)
		}
	}
	return releasesAsMCVersions(all, "Forge")
}

func neoForgeMCVersions() (mcVersions, error) {
	var metadata NeoForgeMavenMetadata
	if err := httpx.GetXML(neoForgeMavenMetadataURL, &metadata); err != nil {
		return mcVersions{}, err
	}
	return neoForgeMCVersionsFrom(metadata.Versioning.Versions.Version)
}

func neoForgeMCVersionsFrom(published []string) (mcVersions, error) {
	var all []string
	for _, version := range published {
		if mcVersion, ok := mcVersionFromNeoForge(version); ok {
			all = append(all, mcVersion)
		}
	}
	return releasesAsMCVersions(all, "NeoForge")
}

// neoForgeNumbering captura los numeros de adelante de una version de NeoForge,
// sin el sufijo -beta ni el numero de build que sobra
var neoForgeNumbering = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:\.(\d+))?`)

// mcVersionFromNeoForge es el inverso de neoForgeVersionPrefix: 21.1.77 es
// Minecraft 1.21.1, 20.2.3 es 1.20.2, y con la numeracion por año 26.1.2.5 es
// 26.1.2 y 26.3.0.38 es 26.3
func mcVersionFromNeoForge(version string) (string, bool) {
	match := neoForgeNumbering.FindStringSubmatch(version)
	if match == nil {
		return "", false
	}
	first, _ := strconv.Atoi(match[1])

	switch {
	case first >= 26:
		if match[4] == "" {
			return "", false
		}
		mcVersion := match[1] + "." + match[2]
		if match[3] != "0" {
			mcVersion += "." + match[3]
		}
		return mcVersion, true
	case first >= 20:
		// las primeras de 1.20.1 venian del proyecto de Forge y no siguen este esquema
		mcVersion := "1." + match[1]
		if match[2] != "0" {
			mcVersion += "." + match[2]
		}
		return mcVersion, true
	}
	return "", false
}

func arclightMCVersions() (mcVersions, error) {
	var listing ArclightListing
	if err := getJSON(arclightFilesURL, &listing); err != nil {
		return mcVersions{}, err
	}
	var all []string
	for _, entry := range listing.Files {
		all = append(all, entry.Name)
	}
	return releasesAsMCVersions(all, "Arclight")
}

func vanillaMCVersions() (mcVersions, error) {
	var manifest MojangManifest
	if err := getJSON(mojangManifestURL, &manifest); err != nil {
		return mcVersions{}, err
	}
	return vanillaMCVersionsFrom(manifest)
}

// el manifest viene de la mas nueva a la mas vieja y marca las releases
func vanillaMCVersionsFrom(manifest MojangManifest) (mcVersions, error) {
	var versions mcVersions
	for _, version := range manifest.Versions {
		versions.known = append(versions.known, version.ID)
		if version.Type == "release" {
			versions.stable = append(versions.stable, version.ID)
		}
	}
	if len(versions.stable) == 0 {
		return mcVersions{}, fmt.Errorf("el manifest de Mojang no trae ninguna release")
	}
	return versions, nil
}

// releasesAsMCVersions es para las APIs que no marcan las estables: se sugieren
// las que tienen formato de release, y se acepta escrita cualquiera publicada
func releasesAsMCVersions(all []string, label string) (mcVersions, error) {
	// NeoForge repite la misma version de Minecraft en cada build
	known := slices.Clone(all)
	slices.Sort(known)
	versions := mcVersions{stable: sortedMCReleases(all), known: slices.Compact(known)}
	if len(versions.stable) == 0 {
		return mcVersions{}, fmt.Errorf("%s no publicó ninguna versión de Minecraft", label)
	}
	return versions, nil
}
