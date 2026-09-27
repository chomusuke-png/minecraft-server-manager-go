package config

import (
	"encoding/json"
	"os"
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
