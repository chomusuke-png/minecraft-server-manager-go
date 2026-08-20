package downloader

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
