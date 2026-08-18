package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultWhenMissing(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	want := DefaultConfig()
	if *cfg != *want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}

	if _, err := os.Stat("config.json"); err != nil {
		t.Error("Load() debería haber creado config.json")
	}
}

func TestLoadReadsExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	custom := Config{
		JavaPath:            "/usr/bin/java21",
		JarName:             "custom.jar",
		RAMGB:               8,
		PlayitPath:          "playit",
		NgrokPath:           "ngrok",
		BackupRetentionDays: 14,
		BackupKeepMin:       5,
	}
	data, err := json.MarshalIndent(custom, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if *cfg != custom {
		t.Errorf("got %+v, want %+v", cfg, custom)
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("config.json", []byte("{esto no es json"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Error("se esperaba error con un config.json corrupto")
	}
}

func TestGetConfigPathPrefersCwd(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeFile(t, "config.json", "{}")

	if got := getConfigPath(); got != "config.json" {
		t.Errorf("got %q, want %q", got, "config.json")
	}
}

func TestGetConfigPathFallsBackToParent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"), "{}")

	sub := filepath.Join(dir, "cmd")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	want := "../config.json"
	if got := getConfigPath(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetConfigPathDefaultsToCwdNameWithoutAnyMarker(t *testing.T) {
	t.Chdir(t.TempDir())

	if got := getConfigPath(); got != "config.json" {
		t.Errorf("got %q, want %q", got, "config.json")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
