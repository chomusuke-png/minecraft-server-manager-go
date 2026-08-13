package downloader

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestNeoForgeVersionPrefix(t *testing.T) {
	cases := []struct {
		mcVersion string
		want      string
		ok        bool
	}{
		{"1.21.1", "21.1.", true},
		{"1.21", "21.0.", true},
		{"1.20.4", "20.4.", true},
		{"1.21.1-pre1", "21.1.", true},
		{"1.26.2", "26.2.", true},
		{"21.1", "", false},
		{"1", "", false},
		{"1.x", "", false},
	}

	for _, c := range cases {
		got, ok := neoForgeVersionPrefix(c.mcVersion)
		if ok != c.ok || got != c.want {
			t.Errorf("neoForgeVersionPrefix(%q) = (%q, %v), want (%q, %v)", c.mcVersion, got, ok, c.want, c.ok)
		}
	}
}

func TestResolveNeoForgeLaunchModern(t *testing.T) {
	dir := t.TempDir()
	expected := filepath.Join("libraries", "net", "neoforged", "neoforge", "21.1.248", argsFileName())
	writeFile(t, filepath.Join(dir, expected), "-p libraries/... cpw.mods.bootstraplauncher.BootstrapLauncher")

	got, err := (&Downloader{serverDir: dir}).resolveForgeLikeLaunch(neoForgeSpec, "21.1.248")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"@" + filepath.ToSlash(expected), "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveNeoForgeLaunchNothingInstalled(t *testing.T) {
	if _, err := (&Downloader{serverDir: t.TempDir()}).resolveForgeLikeLaunch(neoForgeSpec, "21.1.248"); err == nil {
		t.Error("se esperaba error cuando el instalador no dejó nada usable")
	}
}

func TestRemoveNeoForgeInstaller(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, neoForgeInstallerName), "installer")
	writeFile(t, filepath.Join(dir, "installer.log"), "log")
	writeFile(t, filepath.Join(dir, neoForgeInstallerName+".log"), "log")

	(&Downloader{serverDir: dir}).removeForgeLikeInstaller(neoForgeSpec)

	for _, leftover := range []string{neoForgeInstallerName, "installer.log", neoForgeInstallerName + ".log"} {
		if fileExists(filepath.Join(dir, leftover)) {
			t.Errorf("quedó sin borrar: %s", leftover)
		}
	}
}
