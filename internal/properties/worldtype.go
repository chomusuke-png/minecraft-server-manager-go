package properties

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"minecraft-manager/internal/prompt"
)

type worldType struct {
	label  string
	modern string
	legacy string
}

var worldTypes = []worldType{
	{"Normal", "minecraft:normal", "default"},
	{"Plano (superflat)", "minecraft:flat", "flat"},
	{"Biomas grandes", "minecraft:large_biomes", "largeBiomes"},
	{"Amplificado", "minecraft:amplified", "amplified"},
}

func promptWorldType(reader *bufio.Reader, mcVersion string) string {
	fmt.Println("\n[?] Tipo de mundo:")
	for i, wt := range worldTypes {
		fmt.Printf("%d) %s\n", i+1, wt.label)
	}

	choice := prompt.LoopDefault(reader, "[?] Opción [1-4] [1]: ", 1, func(input string) (int, bool, string) {
		value, err := strconv.Atoi(input)
		if err != nil || value < 1 || value > len(worldTypes) {
			return 0, false, fmt.Sprintf("Opción inválida. Elegí un número entre 1 y %d.", len(worldTypes))
		}
		return value, true, ""
	})

	return levelTypeFor(worldTypes[choice-1], mcVersion)
}

func levelTypeFor(wt worldType, mcVersion string) string {
	if isLegacyLevelType(mcVersion) {
		return wt.legacy
	}
	return wt.modern
}

func isLegacyLevelType(mcVersion string) bool {
	parts := strings.Split(strings.TrimSpace(mcVersion), ".")
	if len(parts) < 2 || parts[0] != "1" {
		return false
	}

	end := 0
	for end < len(parts[1]) && parts[1][end] >= '0' && parts[1][end] <= '9' {
		end++
	}
	minor, err := strconv.Atoi(parts[1][:end])
	if err != nil {
		return false
	}

	return minor < 19
}

func escapePropertyValue(value string) string {
	return strings.ReplaceAll(value, ":", `\:`)
}
