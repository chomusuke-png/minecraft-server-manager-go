package downloader

import (
	"fmt"
	"strconv"

	"minecraft-manager/internal/prompt"
)

// Loader es un tipo de servidor que la herramienta sabe instalar. El orden de
// la lista es el que se numera en los menus
type Loader struct {
	Type  string
	Label string
	// Supports dice que acepta el servidor, para poder elegir sin conocer el loader
	Supports string
}

var Loaders = []Loader{
	{Type: "paper", Label: "Paper", Supports: "plugins"},
	{Type: "fabric", Label: "Fabric", Supports: "mods"},
	{Type: "quilt", Label: "Quilt", Supports: "mods (también de Fabric)"},
	{Type: "forge", Label: "Forge", Supports: "mods"},
	{Type: "neoforge", Label: "NeoForge", Supports: "mods"},
	{Type: "arclight", Label: "Arclight", Supports: "plugins y mods"},
	{Type: "vanilla", Label: "Vanilla", Supports: "sin plugins ni mods"},
}

// LoaderLabel devuelve el nombre para mostrar de un loader
func LoaderLabel(loaderType string) (string, bool) {
	for _, loader := range Loaders {
		if loader.Type == loaderType {
			return loader.Label, true
		}
	}
	return "", false
}

// LoaderByChoice traduce el numero que eligio el usuario al tipo de loader
func LoaderByChoice(input string) (string, bool) {
	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(Loaders) {
		return "", false
	}
	return Loaders[index-1].Type, true
}

// LoaderChoice es el inverso de LoaderByChoice: el numero con que aparece el
// loader en los menus
func LoaderChoice(loaderType string) (int, bool) {
	for i, loader := range Loaders {
		if loader.Type == loaderType {
			return i + 1, true
		}
	}
	return 0, false
}

// PrintLoaderOptions imprime la lista numerada de loaders, con lo que acepta
// cada uno en una columna alineada. current se marca como "— actual", y va
// vacio al crear una instancia
func PrintLoaderOptions(indent, current string) {
	fmt.Print(loaderOptions(indent, current))
}

func loaderOptions(indent, current string) string {
	width := 0
	for _, loader := range Loaders {
		width = max(width, len(loader.Label))
	}

	labels := make([]string, len(Loaders))
	notes := make([]string, len(Loaders))
	for i, loader := range Loaders {
		labels[i] = fmt.Sprintf("%-*s   %s", width, loader.Label, loader.Supports)
		if loader.Type == current {
			notes[i] = prompt.OriginCurrent
		}
	}
	return prompt.Options(indent, labels, notes)
}
