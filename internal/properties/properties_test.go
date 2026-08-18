package properties

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

// Todo default: MOTD, dificultad, tipo de mundo, jugadores, online-mode y puerto.
func TestSetupInitialPropertiesWritesDefaults(t *testing.T) {
	dir := t.TempDir()

	if err := SetupInitialProperties(readerFor("\n\n\n\n\n\n"), dir, "1.20.1"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(content)

	for _, want := range []string{
		"motd=Un servidor de Minecraft",
		"difficulty=normal",
		"level-type=minecraft\\:normal",
		"max-players=20",
		"online-mode=true",
		"server-port=25565",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en:\n%s", want, got)
		}
	}
}

func TestSetupInitialPropertiesSkipsIfFileExists(t *testing.T) {
	dir := t.TempDir()
	existing := "motd=ya configurado\n"
	if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	// Reader vacío: si intentara preguntar de nuevo, fallaría al leer.
	if err := SetupInitialProperties(readerFor(""), dir, "1.20.1"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != existing {
		t.Errorf("no debería haber tocado un server.properties existente")
	}
}

func TestPromptDifficultyDefaultsToNormalOnEmpty(t *testing.T) {
	if got := promptDifficulty(readerFor("\n")); got != "normal" {
		t.Errorf("got %q, want %q", got, "normal")
	}
}

func TestPromptDifficultySelectsByNumber(t *testing.T) {
	cases := map[string]string{
		"1\n": "peaceful",
		"2\n": "easy",
		"3\n": "normal",
		"4\n": "hard",
	}
	for input, want := range cases {
		if got := promptDifficulty(readerFor(input)); got != want {
			t.Errorf("promptDifficulty(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPromptDifficultyRetriesOnInvalidInput(t *testing.T) {
	if got := promptDifficulty(readerFor("0\n9\n2\n")); got != "easy" {
		t.Errorf("got %q", got)
	}
}

func TestReadPort(t *testing.T) {
	dir := t.TempDir()
	writeProperties(t, dir, "motd=x\nserver-port=25566\ndifficulty=normal\n")

	port, ok := ReadPort(dir)
	if !ok || port != 25566 {
		t.Errorf("got (%d, %v), want (25566, true)", port, ok)
	}
}

func TestReadPortMissingFile(t *testing.T) {
	if _, ok := ReadPort(t.TempDir()); ok {
		t.Error("se esperaba ok=false sin server.properties")
	}
}

func TestReadPortMissingKey(t *testing.T) {
	dir := t.TempDir()
	writeProperties(t, dir, "motd=x\n")

	if _, ok := ReadPort(dir); ok {
		t.Error("se esperaba ok=false sin server-port")
	}
}

func TestUpdatePortReplacesExistingLine(t *testing.T) {
	dir := t.TempDir()
	writeProperties(t, dir, "motd=x\nserver-port=25565\ndifficulty=hard\n")

	if err := UpdatePort(readerFor("30000\n"), dir); err != nil {
		t.Fatal(err)
	}

	port, ok := ReadPort(dir)
	if !ok || port != 30000 {
		t.Errorf("got (%d, %v)", port, ok)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "difficulty=hard") {
		t.Error("no debería haber tocado el resto del archivo")
	}
}

func TestUpdatePortDefaultsToCurrentOnEmpty(t *testing.T) {
	dir := t.TempDir()
	writeProperties(t, dir, "server-port=25565\n")

	if err := UpdatePort(readerFor("\n"), dir); err != nil {
		t.Fatal(err)
	}

	port, ok := ReadPort(dir)
	if !ok || port != 25565 {
		t.Errorf("got (%d, %v)", port, ok)
	}
}

func TestUpdatePortNoOpWithoutExistingFile(t *testing.T) {
	dir := t.TempDir()

	// Reader vacío: si intentara preguntar, fallaría al leer.
	if err := UpdatePort(readerFor(""), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "server.properties")); err == nil {
		t.Error("no debería haber creado server.properties")
	}
}

func writeProperties(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
