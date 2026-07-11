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
			URL string `json:"url"`
		} `json:"server"`
	} `json:"downloads"`
}

type PaperBuildsResponse struct {
	Builds []int `json:"builds"`
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
