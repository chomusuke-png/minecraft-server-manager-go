package downloader

import "time"

type MojangManifest struct {
	Versions []MojangVersion `json:"versions"`
}

type MojangVersion struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type MojangVersionDetails struct {
	Downloads struct {
		Server struct {
			URL  string `json:"url"`
			SHA1 string `json:"sha1"`
		} `json:"server"`
	} `json:"downloads"`
	// el Java que pide Mojang para correr esa version
	JavaVersion struct {
		MajorVersion int `json:"majorVersion"`
	} `json:"javaVersion"`
}

// PaperBuild es una build en la API v3 de Paper, que devuelve la lista de la
// mas nueva a la mas vieja y trae el canal y la descarga en la misma respuesta
type PaperBuild struct {
	ID        int                      `json:"id"`
	Channel   string                   `json:"channel"`
	Downloads map[string]PaperDownload `json:"downloads"`
}

type PaperDownload struct {
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	Checksums map[string]string `json:"checksums"`
}

type FabricLoader struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type FabricInstaller struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Url     string `json:"url"`
}

// QuiltLoader es una entrada del listado general de loaders de Quilt
type QuiltLoader struct {
	Version string `json:"version"`
}

// QuiltLoaderForMC es una entrada del listado filtrado por version de
// Minecraft, que anida el loader junto a sus mappings
type QuiltLoaderForMC struct {
	Loader QuiltLoader `json:"loader"`
}

type QuiltInstaller struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// ArclightListing es un directorio de la API de archivos de Arclight: las bases
// de una version de Minecraft o las builds de un canal
type ArclightListing struct {
	Files []ArclightEntry `json:"files"`
}

type ArclightEntry struct {
	Name         string    `json:"name"`
	LastModified time.Time `json:"last-modified"`
	// solo lo traen las builds, no los directorios
	Permlink string `json:"permlink"`
}

type ForgePromotions struct {
	Homepage string            `json:"homepage"`
	Promos   map[string]string `json:"promos"`
}

// NeoForgeMavenMetadata es la respuesta de maven-metadata.xml del maven de
// NeoForge: solo trae la lista plana de versiones, en orden de publicación.
type NeoForgeMavenMetadata struct {
	Versioning struct {
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
	} `xml:"versioning"`
}
