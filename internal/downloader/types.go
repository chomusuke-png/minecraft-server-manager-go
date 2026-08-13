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

type PaperBuildsResponse struct {
	Builds []int `json:"builds"`
}

// PaperBuildDetails es la respuesta de /builds/{build}, que trae el nombre real
// del jar y su sha256. El endpoint de versión (PaperBuildsResponse) solo da
// números de build, sin esto.
type PaperBuildDetails struct {
	Downloads struct {
		Application struct {
			Name   string `json:"name"`
			SHA256 string `json:"sha256"`
		} `json:"application"`
	} `json:"downloads"`
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
