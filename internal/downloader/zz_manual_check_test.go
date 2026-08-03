//go:build manual

package downloader

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// Chequeo manual contra la red real. Correr con:
//   go test -tags manual ./internal/downloader/ -run TestManualForge -v -count=1
func TestManualForgeResolution(t *testing.T) {
	var promos ForgePromotions
	if err := getJSON("https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json", &promos); err != nil {
		t.Fatalf("promotions_slim.json: %v", err)
	}
	t.Logf("total de promos: %d", len(promos.Promos))

	const version = "1.19.1"
	recommended := promos.Promos[version+"-recommended"]
	latest := promos.Promos[version+"-latest"]
	t.Logf("%s-recommended = %q", version, recommended)
	t.Logf("%s-latest      = %q", version, latest)

	forgeVersion := recommended
	if forgeVersion == "" {
		forgeVersion = latest
	}
	if forgeVersion == "" {
		t.Fatalf("no se resolvió ninguna versión de Forge para %s", version)
	}

	fullVersion := version + "-" + forgeVersion
	url := "https://maven.minecraftforge.net/net/minecraftforge/forge/" + fullVersion + "/forge-" + fullVersion + "-installer.jar"
	t.Logf("fullVersion = %s", fullVersion)
	t.Logf("url         = %s", url)

	resp, err := http.Head(url)
	if err != nil {
		t.Fatalf("HEAD: %v", err)
	}
	defer resp.Body.Close()
	t.Logf("HEAD status  = %s", resp.Status)
	t.Logf("Content-Length = %d bytes", resp.ContentLength)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("el instalador no existe en maven: %s", resp.Status)
	}
}

// Descarga real del instalador a un dir temporal, sin ejecutarlo.
func TestManualForgeDownloadOnly(t *testing.T) {
	dir := t.TempDir()
	d := New(dir, "java")

	var promos ForgePromotions
	if err := getJSON("https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json", &promos); err != nil {
		t.Fatal(err)
	}
	forgeVersion := promos.Promos["1.19.1-recommended"]
	if forgeVersion == "" {
		forgeVersion = promos.Promos["1.19.1-latest"]
	}
	fullVersion := "1.19.1-" + forgeVersion

	url := "https://maven.minecraftforge.net/net/minecraftforge/forge/" + fullVersion + "/forge-" + fullVersion + "-installer.jar"
	if err := d.DownloadFile(url, forgeInstallerName); err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, forgeInstallerName))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("descargado: %s (%d bytes)", forgeInstallerName, info.Size())
	if info.Size() < 100_000 {
		t.Errorf("el instalador parece truncado: %d bytes", info.Size())
	}

	// Confirmar que es un jar (zip) y no una página de error.
	f, err := os.Open(filepath.Join(dir, forgeInstallerName))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	magic := make([]byte, 2)
	if _, err := f.Read(magic); err != nil {
		t.Fatal(err)
	}
	if string(magic) != "PK" {
		t.Errorf("no es un zip/jar válido, magic = %q", magic)
	}
	t.Logf("magic = %q (zip/jar OK)", magic)
}
