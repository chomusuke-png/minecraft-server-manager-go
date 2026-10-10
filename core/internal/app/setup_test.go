package app

import (
	"os"
	"path/filepath"
	"testing"
)

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
