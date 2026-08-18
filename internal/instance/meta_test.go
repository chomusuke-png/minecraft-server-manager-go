package instance

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoadMetaRoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := InstanceMeta{
		LoaderType:     "forge",
		MCVersion:      "1.20.1",
		LoaderVersion:  "47.2.0",
		RAMGB:          6,
		LaunchArgs:     []string{"@user_jvm_args.txt", "@libraries/forge/win_args.txt", "nogui"},
		JavaPath:       "runtimes/17/bin/java",
		BackupKeepMin:  5,
		TunnelProvider: "ngrok",
	}

	if err := SaveMeta(dir, original); err != nil {
		t.Fatal(err)
	}

	got, err := LoadMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*got, original) {
		t.Errorf("got %+v, want %+v", got, original)
	}

	if _, err := filepath.Abs(filepath.Join(dir, "instance.json")); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMetaMissingFile(t *testing.T) {
	if _, err := LoadMeta(t.TempDir()); err == nil {
		t.Error("se esperaba error sin instance.json")
	}
}

func TestSaveMetaOverwritesExisting(t *testing.T) {
	dir := t.TempDir()

	if err := SaveMeta(dir, InstanceMeta{LoaderType: "paper", MCVersion: "1.20.1"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveMeta(dir, InstanceMeta{LoaderType: "fabric", MCVersion: "1.21.0"}); err != nil {
		t.Fatal(err)
	}

	got, err := LoadMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.LoaderType != "fabric" || got.MCVersion != "1.21.0" {
		t.Errorf("got %+v", got)
	}
}
