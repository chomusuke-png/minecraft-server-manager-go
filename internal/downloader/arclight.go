package downloader

import (
	"bufio"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"minecraft-manager/internal/httpx"
	"minecraft-manager/internal/logx"
	"minecraft-manager/internal/prompt"
)

// la misma API que usa la pagina de descargas de Arclight
const arclightFilesURL = "https://files.hypoglycemia.icu/v1/files/arclight/minecraft"

// Arclight corre sobre Forge, NeoForge o Fabric segun la version de Minecraft.
// Este es el orden en que se ofrecen cuando hay mas de uno
var arclightBaseOrder = []string{"neoforge", "forge", "fabric"}

var arclightBaseLabels = map[string]string{
	"neoforge": "NeoForge",
	"forge":    "Forge",
	"fabric":   "Fabric",
}

var sha1Pattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func arclightLoadersURL(mcVersion string) string {
	return fmt.Sprintf("%s/%s/loaders", arclightFilesURL, mcVersion)
}

func arclightChannelURL(mcVersion, base, channel string) string {
	return fmt.Sprintf("%s/%s/versions-%s", arclightLoadersURL(mcVersion), base, channel)
}

// la version guardada lleva la base adelante (ej. "neoforge-1.0.1-8ec9529")
// porque el mismo numero de Arclight existe para varias bases
func arclightVersionValue(base, build string) string {
	return base + "-" + build
}

func splitArclightVersion(value string) (base, build string, ok bool) {
	base, build, ok = strings.Cut(value, "-")
	if !ok || build == "" {
		return "", "", false
	}
	if _, known := arclightBaseLabels[base]; !known {
		return "", "", false
	}
	return base, build, true
}

// chooseArclightVersion pregunta primero la base y despues la version, porque
// cada base tiene su propio listado
func chooseArclightVersion(reader *bufio.Reader, mcVersion, current string) (string, error) {
	logx.Info("\nBuscando versiones de Arclight para %s...", mcVersion)

	bases, err := arclightBases(mcVersion)
	if err != nil {
		return "", err
	}

	currentBase, _, _ := splitArclightVersion(current)
	base, err := promptArclightBase(reader, bases, currentBase)
	if err != nil {
		return "", err
	}

	available, err := arclightVersions(mcVersion, base)
	if err != nil {
		return "", err
	}

	// la version actual solo sirve si es de la misma base
	if !strings.HasPrefix(current, base+"-") {
		current = ""
	}
	return promptLoaderVersion(reader, "Arclight", available, current)
}

func arclightBases(mcVersion string) ([]string, error) {
	var listing ArclightListing
	if err := getJSON(arclightLoadersURL(mcVersion), &listing); err != nil {
		if errors.Is(err, httpx.ErrNotFound) {
			return nil, fmt.Errorf("Arclight no tiene versiones para Minecraft %s", mcVersion)
		}
		return nil, fmt.Errorf("error obteniendo las bases de Arclight: %w", err)
	}
	return arclightBasesFrom(listing, mcVersion)
}

func arclightBasesFrom(listing ArclightListing, mcVersion string) ([]string, error) {
	published := make(map[string]bool, len(listing.Files))
	for _, entry := range listing.Files {
		published[entry.Name] = true
	}

	var bases []string
	for _, base := range arclightBaseOrder {
		if published[base] {
			bases = append(bases, base)
		}
	}
	if len(bases) == 0 {
		return nil, fmt.Errorf("Arclight no tiene versiones para Minecraft %s", mcVersion)
	}
	return bases, nil
}

func promptArclightBase(reader *bufio.Reader, bases []string, current string) (string, error) {
	if len(bases) == 1 {
		logx.Detail("Arclight corre sobre %s en esta versión.", arclightBaseLabels[bases[0]])
		return bases[0], nil
	}

	// Enter mantiene la base actual al actualizar, o elige la primera al crear
	defaultOption, note, enterHint := 1, prompt.OriginDefault, prompt.EnterPicksDefault
	if index := slices.Index(bases, current); index >= 0 {
		defaultOption, note, enterHint = index+1, prompt.OriginCurrent, prompt.EnterKeepsCurrent
	}
	cancelOption := len(bases) + 1

	labels := make([]string, len(bases))
	notes := make([]string, len(bases))
	for i, base := range bases {
		labels[i] = arclightBaseLabels[base]
	}
	notes[defaultOption-1] = note

	fmt.Printf("\n[?] Base de Arclight:\n")
	fmt.Print(prompt.Options("  ", labels, notes))
	fmt.Printf("  %d) Cancelar\n", cancelOption)

	promptText := prompt.MenuQuestion(cancelOption, enterHint)
	choice := prompt.LoopDefault(reader, promptText, defaultOption, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value < 1 || value > cancelOption {
			return 0, false, fmt.Sprintf("Opción inválida. Elige un número entre 1 y %d.", cancelOption)
		}
		return value, true, ""
	})

	if choice == cancelOption {
		return "", ErrCancelled
	}
	return bases[choice-1], nil
}

func arclightVersions(mcVersion, base string) (loaderVersions, error) {
	stable, snapshot, err := arclightChannels(mcVersion, base)
	if err != nil {
		return loaderVersions{}, err
	}
	return arclightVersionsFrom(base, stable, snapshot, mcVersion)
}

// arclightChannels trae los dos canales. Un canal sin publicaciones responde
// 404, asi que eso cuenta como vacio y no como error
func arclightChannels(mcVersion, base string) (stable, snapshot []ArclightEntry, err error) {
	channels := map[string]*[]ArclightEntry{"stable": &stable, "snapshot": &snapshot}
	for channel, target := range channels {
		var listing ArclightListing
		err := getJSON(arclightChannelURL(mcVersion, base, channel), &listing)
		if errors.Is(err, httpx.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error obteniendo las versiones de Arclight: %w", err)
		}
		*target = listing.Files
	}
	return stable, snapshot, nil
}

// arclightVersionsFrom ordena por fecha y no por el orden de la API, porque
// latest tiene que comparar entre los dos canales
func arclightVersionsFrom(base string, stable, snapshot []ArclightEntry, mcVersion string) (loaderVersions, error) {
	var versions loaderVersions
	var latest, newestStable ArclightEntry

	for _, entry := range stable {
		versions.known = append(versions.known, arclightVersionValue(base, entry.Name))
		if entry.LastModified.After(newestStable.LastModified) {
			newestStable = entry
		}
		if entry.LastModified.After(latest.LastModified) {
			latest = entry
		}
	}
	for _, entry := range snapshot {
		versions.known = append(versions.known, arclightVersionValue(base, entry.Name))
		if entry.LastModified.After(latest.LastModified) {
			latest = entry
		}
	}

	if latest.Name == "" {
		return loaderVersions{}, fmt.Errorf("no se encontró ninguna versión de Arclight sobre %s para Minecraft %s", arclightBaseLabels[base], mcVersion)
	}

	versions.latest = arclightVersionValue(base, latest.Name)
	if newestStable.Name != "" {
		versions.stable = arclightVersionValue(base, newestStable.Name)
	}
	return versions, nil
}

func (d *Downloader) DownloadArclight(mcVersion, version string) (string, error) {
	base, build, ok := splitArclightVersion(version)
	if !ok {
		return "", fmt.Errorf("versión de Arclight inválida: %s (se espera <base>-<versión>, ej. neoforge-1.0.1-8ec9529)", version)
	}

	logx.Info("Buscando Arclight %s sobre %s para %s...", build, arclightBaseLabels[base], mcVersion)

	stable, snapshot, err := arclightChannels(mcVersion, base)
	if err != nil {
		return "", err
	}
	entry, found := findArclightEntry(build, stable, snapshot)
	if !found {
		return "", fmt.Errorf("Arclight no publicó la versión %s sobre %s para Minecraft %s", build, arclightBaseLabels[base], mcVersion)
	}

	// el permlink apunta al objeto por su sha1, asi que sirve como checksum
	sha1Hex := path.Base(entry.Permlink)
	if !sha1Pattern.MatchString(sha1Hex) {
		logx.Warn("El link de Arclight no trae el checksum, se descarga sin verificar.")
		sha1Hex = ""
	}
	if err := d.DownloadFileVerified(entry.Permlink, "server.jar", "sha1", sha1Hex); err != nil {
		return "", err
	}

	logx.Detail("En el primer arranque Arclight descarga las librerías del loader; puede tardar unos minutos.")
	return version, nil
}

func findArclightEntry(build string, channels ...[]ArclightEntry) (ArclightEntry, bool) {
	for _, entries := range channels {
		for _, entry := range entries {
			if entry.Name == build && entry.Permlink != "" {
				return entry, true
			}
		}
	}
	return ArclightEntry{}, false
}
