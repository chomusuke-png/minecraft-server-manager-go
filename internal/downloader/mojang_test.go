package downloader

import (
	"encoding/json"
	"testing"
)

func TestMojangVersionURL(t *testing.T) {
	manifest := MojangManifest{Versions: []MojangVersion{
		{ID: "26.3", URL: "https://mojang/26.3.json"},
		{ID: "1.21.11", URL: "https://mojang/1.21.11.json"},
	}}

	if url, ok := mojangVersionURL(manifest, "1.21.11"); !ok || url != "https://mojang/1.21.11.json" {
		t.Errorf("got (%q, %v)", url, ok)
	}
	if _, ok := mojangVersionURL(manifest, "26.9"); ok {
		t.Error("una versión que no está en el manifest no debería encontrarse")
	}
}

// el recorte de un detalle real de Mojang, con lo que se usa
func TestMojangVersionDetailsLeeElJava(t *testing.T) {
	raw := `{
		"downloads": {"server": {"sha1": "abc", "url": "https://mojang/server.jar"}},
		"javaVersion": {"component": "java-runtime-epsilon", "majorVersion": 25}
	}`

	var details MojangVersionDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		t.Fatal(err)
	}
	if details.JavaVersion.MajorVersion != 25 {
		t.Errorf("majorVersion = %d, want 25", details.JavaVersion.MajorVersion)
	}
	if details.Downloads.Server.URL != "https://mojang/server.jar" {
		t.Errorf("se rompió la lectura del jar: %q", details.Downloads.Server.URL)
	}
}
