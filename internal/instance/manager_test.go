package instance

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

func TestGetAvailableInstancesCreatesRootDirIfMissing(t *testing.T) {
	t.Chdir(t.TempDir())

	instances, err := GetAvailableInstances()
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 0 {
		t.Errorf("got %v, want vacío", instances)
	}
	if _, err := os.Stat(InstancesRootDir); err != nil {
		t.Error("debería haber creado instances/")
	}
}

func TestGetAvailableInstancesListsOnlyDirs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.MkdirAll(filepath.Join(InstancesRootDir, "server1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(InstancesRootDir, "server2"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(InstancesRootDir, "no_es_instancia.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	instances, err := GetAvailableInstances()
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 2 {
		t.Errorf("got %v, want 2 instancias", instances)
	}
}

func TestDeleteInstanceRequiresExactName(t *testing.T) {
	dir := t.TempDir()
	instancePath := filepath.Join(dir, "mi_server")
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := DeleteInstance(readerFor("nombre_mal\notro_mal\n"), instancePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(instancePath); err != nil {
		t.Error("la instancia no debería haberse borrado sin confirmar")
	}
}

func TestDeleteInstanceEmptyInputCancels(t *testing.T) {
	dir := t.TempDir()
	instancePath := filepath.Join(dir, "mi_server")
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := DeleteInstance(readerFor("\n"), instancePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(instancePath); err != nil {
		t.Error("Enter en blanco debería cancelar sin borrar")
	}
}

func TestDeleteInstanceExactNameDeletes(t *testing.T) {
	dir := t.TempDir()
	instancePath := filepath.Join(dir, "mi_server")
	if err := os.MkdirAll(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := DeleteInstance(readerFor("mi_server\n"), instancePath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(instancePath); err == nil {
		t.Error("la instancia debería haberse borrado")
	}
}

func TestPromptRAMUpdate(t *testing.T) {
	if got := PromptRAMUpdate(readerFor("\n"), 4); got != 4 {
		t.Errorf("got %d, want 4 (default)", got)
	}
	if got := PromptRAMUpdate(readerFor("8\n"), 4); got != 8 {
		t.Errorf("got %d, want 8", got)
	}
	if got := PromptRAMUpdate(readerFor("-1\n2\n"), 4); got != 2 {
		t.Errorf("got %d, want 2 tras rechazar -1", got)
	}
}

func TestPromptBackupKeepMinUpdate(t *testing.T) {
	if got := PromptBackupKeepMinUpdate(readerFor("\n"), 3); got != 3 {
		t.Errorf("got %d, want 3 (default)", got)
	}
	if got := PromptBackupKeepMinUpdate(readerFor("0\n"), 3); got != 0 {
		t.Errorf("got %d, want 0 (0 es válido acá)", got)
	}
	if got := PromptBackupKeepMinUpdate(readerFor("-1\n5\n"), 3); got != 5 {
		t.Errorf("got %d, want 5 tras rechazar -1", got)
	}
}

func TestPromptTunnelProvider(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"\n", "playit"},
		{"1\n", "playit"},
		{"2\n", "ngrok"},
		{"3\n", "none"},
	}
	for _, c := range cases {
		if got := PromptTunnelProvider(readerFor(c.input)); got != c.want {
			t.Errorf("PromptTunnelProvider(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestPromptTunnelProviderUpdateKeepsCurrentOnEmpty(t *testing.T) {
	if got := PromptTunnelProviderUpdate(readerFor("\n"), "ngrok"); got != "ngrok" {
		t.Errorf("got %q, want %q", got, "ngrok")
	}
}

func TestPromptTunnelProviderUpdateDefaultsLegacyEmptyToPlayit(t *testing.T) {
	if got := PromptTunnelProviderUpdate(readerFor("\n"), ""); got != "playit" {
		t.Errorf("got %q, want %q", got, "playit")
	}
}

func TestPromptTunnelProviderUpdateChangesValue(t *testing.T) {
	if got := PromptTunnelProviderUpdate(readerFor("2\n"), "playit"); got != "ngrok" {
		t.Errorf("got %q, want %q", got, "ngrok")
	}
}
