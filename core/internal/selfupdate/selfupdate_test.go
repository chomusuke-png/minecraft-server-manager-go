package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		candidate, current string
		want               bool
	}{
		{"v1.2.0", "v1.1.0", true},
		{"v1.1.0", "v1.2.0", false},
		{"v1.1.0", "v1.1.0", false},
		{"v2.0.0", "v1.9.9", true},
		{"v1.10.0", "v1.9.0", true},
		{"v1.0.10", "v1.0.9", true},
		{"garbage", "v1.1.0", false},
		{"v1.1.0", "dev", false},
		{"v1.1.0", "", false},
	}

	for _, c := range cases {
		if got := IsNewer(c.candidate, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.candidate, c.current, got, c.want)
		}
	}
}

func TestSwapReplacesCurrentAndKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	currentExe := filepath.Join(dir, "msm.exe")
	newPath := filepath.Join(dir, "msm.exe.new")
	oldPath := currentExe + ".old"

	writeFile(t, currentExe, "version vieja")
	writeFile(t, newPath, "version nueva")

	if err := swap(currentExe, newPath, oldPath); err != nil {
		t.Fatal(err)
	}

	assertContent(t, currentExe, "version nueva")
	assertContent(t, oldPath, "version vieja")
	if _, err := os.Stat(newPath); err == nil {
		t.Error("newPath debería haberse consumido (renombrado a currentExe)")
	}
}

func TestSwapCleansUpPreviousBackup(t *testing.T) {
	dir := t.TempDir()
	currentExe := filepath.Join(dir, "msm.exe")
	newPath := filepath.Join(dir, "msm.exe.new")
	oldPath := currentExe + ".old"

	writeFile(t, oldPath, "backup de una actualización anterior")
	writeFile(t, currentExe, "version vieja")
	writeFile(t, newPath, "version nueva")

	if err := swap(currentExe, newPath, oldPath); err != nil {
		t.Fatal(err)
	}

	assertContent(t, oldPath, "version vieja")
}

func TestSwapFailsCleanlyWithoutCurrentExe(t *testing.T) {
	dir := t.TempDir()
	currentExe := filepath.Join(dir, "no-existe.exe")
	newPath := filepath.Join(dir, "msm.exe.new")
	oldPath := currentExe + ".old"

	writeFile(t, newPath, "version nueva")

	if err := swap(currentExe, newPath, oldPath); err == nil {
		t.Fatal("se esperaba error")
	}

	if _, err := os.Stat(newPath); err != nil {
		t.Error("newPath no debería haberse perdido si currentExe no existía")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}
