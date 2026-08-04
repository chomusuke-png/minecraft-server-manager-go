//go:build manual

package downloader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"minecraft-manager/internal/java"
)

// Instalación real de Forge 1.19.1 de punta a punta. Correr con:
//
//	go test -tags manual ./internal/downloader/ -run TestManualForgeInstall -v -count=1 -timeout 30m
func TestManualForgeInstall(t *testing.T) {
	t.Chdir("../..")

	javaPath := java.Absolute("runtimes/jdk-17/jdk-17.0.20+8/bin/java.exe")
	if _, err := os.Stat(javaPath); err != nil {
		t.Skipf("falta el JDK 17: %v", err)
	}

	dir := filepath.Join("instances", "test-1191")
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	loaderVersion, launchArgs, err := New(dir, javaPath).DownloadForge("1.19.1")
	if err != nil {
		t.Fatalf("DownloadForge: %v", err)
	}

	t.Logf("loaderVersion = %s", loaderVersion)
	t.Logf("launchArgs    = %q", launchArgs)

	for _, leftover := range []string{forgeInstallerName, "installer.log", forgeInstallerName + ".log"} {
		if fileExists(filepath.Join(dir, leftover)) {
			t.Errorf("quedó sin borrar: %s", leftover)
		}
	}

	if len(launchArgs) == 0 {
		t.Fatal("1.19.1 debería devolver launch args, no un server.jar")
	}

	// Todo @argfile referenciado tiene que existir relativo al dir de la instancia.
	for _, arg := range launchArgs {
		if !strings.HasPrefix(arg, "@") {
			continue
		}
		path := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(arg, "@")))
		if !fileExists(path) {
			t.Errorf("launch arg %q apunta a un archivo inexistente: %s", arg, path)
		} else {
			t.Logf("OK %s -> %s", arg, path)
		}
	}

	// Lo que el servidor necesita y NO se debe haber borrado.
	for _, keep := range []string{"libraries", "user_jvm_args.txt", "run.bat"} {
		if !fileExists(filepath.Join(dir, keep)) {
			t.Errorf("falta '%s' después de la limpieza", keep)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("--- contenido de la instancia ---")
	for _, entry := range entries {
		info, _ := entry.Info()
		if entry.IsDir() {
			t.Logf("  %s/", entry.Name())
			continue
		}
		t.Logf("  %s (%d bytes)", entry.Name(), info.Size())
	}
}
