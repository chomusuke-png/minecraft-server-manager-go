package eula

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readerFor(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

func TestEnsureEulaAcceptedShortCircuitsIfAlreadyTrue(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "eula.txt"), []byte("eula=true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureEulaAccepted(readerFor(""), dir); err != nil {
		t.Fatalf("no debería preguntar si ya está aceptado: %v", err)
	}
}

func TestEnsureEulaAcceptedWritesFileOnYes(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureEulaAccepted(readerFor("y\n"), dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "eula.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "eula=true") {
		t.Errorf("got %q", content)
	}
}

func TestEnsureEulaAcceptedReturnsErrorOnNo(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureEulaAccepted(readerFor("n\n"), dir); err == nil {
		t.Error("se esperaba error al rechazar el EULA")
	}
	if _, err := os.Stat(filepath.Join(dir, "eula.txt")); err == nil {
		t.Error("no debería haberse creado eula.txt al rechazar")
	}
}

func TestEnsureEulaAcceptedRetriesOnInvalidInput(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureEulaAccepted(readerFor("tal vez\ny\n"), dir); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureEulaAcceptedReAsksIfFileSaysFalse(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "eula.txt"), []byte("eula=false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureEulaAccepted(readerFor("y\n"), dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "eula.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "eula=true") {
		t.Errorf("got %q", content)
	}
}
