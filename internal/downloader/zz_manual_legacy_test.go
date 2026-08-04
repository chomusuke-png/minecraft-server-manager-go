//go:build manual

package downloader

import (
	"os"
	"path/filepath"
	"testing"

	"minecraft-manager/internal/java"
)

// Instalación real de Forge 1.16.5, que es la rama legacy: el instalador sí deja
// un jar ejecutable y hay que renombrarlo a server.jar. Correr con:
//
//	go test -tags manual ./internal/downloader/ -run TestManualForgeLegacy -v -count=1 -timeout 30m
func TestManualForgeLegacyInstall(t *testing.T) {
	t.Chdir("../..")

	// Forge 1.16.5 necesita Java 8; su instalador tampoco corre en 17+.
	javaPath, err := java.Download(8)
	if err != nil {
		t.Fatalf("Download(8): %v", err)
	}
	t.Logf("Java 8 en: %s", javaPath)

	dir := filepath.Join("instances", "test-1165")
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	loaderVersion, launchArgs, err := New(dir, javaPath).DownloadForge("1.16.5")
	if err != nil {
		t.Fatalf("DownloadForge: %v", err)
	}

	t.Logf("loaderVersion = %s", loaderVersion)
	t.Logf("launchArgs    = %q", launchArgs)

	// La rama legacy no usa args file: arranca con -jar server.jar.
	if len(launchArgs) != 0 {
		t.Errorf("1.16.5 no debería devolver launch args, got %q", launchArgs)
	}

	serverJar, err := os.Stat(filepath.Join(dir, "server.jar"))
	if err != nil {
		t.Fatalf("no se creó server.jar: %v", err)
	}
	t.Logf("server.jar = %d bytes", serverJar.Size())

	// El jar de Forge referencia al vanilla, que no se debe haber borrado.
	if _, err := os.Stat(filepath.Join(dir, "minecraft_server.1.16.5.jar")); err != nil {
		t.Errorf("falta minecraft_server.1.16.5.jar: %v", err)
	}

	for _, leftover := range []string{forgeInstallerName, "installer.log", forgeInstallerName + ".log"} {
		if fileExists(filepath.Join(dir, leftover)) {
			t.Errorf("quedó sin borrar: %s", leftover)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("--- contenido de la instancia ---")
	for _, entry := range entries {
		if entry.IsDir() {
			t.Logf("  %s/", entry.Name())
			continue
		}
		info, _ := entry.Info()
		t.Logf("  %s (%d bytes)", entry.Name(), info.Size())
	}
}
