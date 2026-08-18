package app

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"minecraft-manager/internal/config"
)

func readerFor(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "server.jar")
	if err := os.WriteFile(present, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	if !fileExists(present) {
		t.Error("debería existir")
	}
	if fileExists(filepath.Join(dir, "no-existe.jar")) {
		t.Error("no debería existir")
	}
}

func TestCleanIncompleteInstanceRemovesJarlessInstance(t *testing.T) {
	dir := t.TempDir()
	instancePath := filepath.Join(dir, "incompleta")
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instancePath, "instance.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	cleanIncompleteInstance(instancePath)

	if _, err := os.Stat(instancePath); err == nil {
		t.Error("una instancia con solo instance.json debería borrarse")
	}
}

func TestCleanIncompleteInstanceKeepsInstanceWithOtherFiles(t *testing.T) {
	dir := t.TempDir()
	instancePath := filepath.Join(dir, "con_progreso")
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instancePath, "instance.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instancePath, "server.jar"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	cleanIncompleteInstance(instancePath)

	if _, err := os.Stat(instancePath); err != nil {
		t.Error("una instancia con jar descargado no debería borrarse")
	}
}

func TestCleanIncompleteInstanceMissingDirIsNoop(t *testing.T) {
	cleanIncompleteInstance(filepath.Join(t.TempDir(), "no-existe"))
}

func TestCheckForUpdatesSkipsOnDevVersion(t *testing.T) {
	checkForUpdates(readerFor(""), &config.Config{}, "dev")
}

func TestCheckForUpdatesSkipsOnEmptyVersion(t *testing.T) {
	checkForUpdates(readerFor(""), &config.Config{}, "")
}

func TestCheckForUpdatesSkipsWhenDisabled(t *testing.T) {
	checkForUpdates(readerFor(""), &config.Config{DisableUpdateCheck: true}, "v1.0.0")
}
