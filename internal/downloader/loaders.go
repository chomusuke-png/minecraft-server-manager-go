package downloader

import (
	"fmt"
	"strconv"
)

// Loader es un tipo de servidor que la herramienta sabe instalar. El orden de
// la lista es el que se numera en los menus
type Loader struct {
	Type  string
	Label string
}

var Loaders = []Loader{
	{Type: "paper", Label: "Paper"},
	{Type: "fabric", Label: "Fabric"},
	{Type: "forge", Label: "Forge"},
	{Type: "neoforge", Label: "NeoForge"},
	{Type: "vanilla", Label: "Vanilla"},
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

// PrintLoaderOptions imprime la lista numerada de loaders
func PrintLoaderOptions(indent string) {
	for i, loader := range Loaders {
		fmt.Printf("%s%d) %s\n", indent, i+1, loader.Label)
	}
}
