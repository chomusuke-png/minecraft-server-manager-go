package downloader

import (
	"slices"
	"testing"
)

func TestQuiltVersionsFrom(t *testing.T) {
	// el listado general viene de la mas nueva a la mas vieja
	all := []QuiltLoader{
		{Version: "0.31.0-beta.1"},
		{Version: "0.30.1"},
		{Version: "0.30.1-beta.2"},
		{Version: "0.30.0"},
		{Version: "0.16.1-beta.1"},
	}
	// el filtrado por version de Minecraft no respeta ese orden
	supported := map[string]bool{
		"0.30.0":        true,
		"0.31.0-beta.1": true,
		"0.30.1-beta.2": true,
		"0.30.1":        true,
	}

	versions, err := quiltVersionsFrom(all, supported, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if versions.latest != "0.31.0-beta.1" {
		t.Errorf("latest = %q, want 0.31.0-beta.1", versions.latest)
	}
	if versions.stable != "0.30.1" {
		t.Errorf("stable = %q, want 0.30.1", versions.stable)
	}
	if !slices.Equal(versions.known, []string{"0.31.0-beta.1", "0.30.1", "0.30.1-beta.2", "0.30.0"}) {
		t.Errorf("known = %v", versions.known)
	}
}

func TestQuiltVersionsFromSinCompatibles(t *testing.T) {
	all := []QuiltLoader{{Version: "0.30.0"}}
	if _, err := quiltVersionsFrom(all, map[string]bool{}, "1.12.2"); err == nil {
		t.Error("una versión sin loaders compatibles debería dar error")
	}
}

func TestLatestQuiltInstallerSaltaPrereleases(t *testing.T) {
	installers := []QuiltInstaller{
		{Version: "0.16.0-beta.1", URL: "https://example.invalid/beta.jar"},
		{Version: "0.15.1", URL: "https://example.invalid/0.15.1.jar"},
		{Version: "0.15.0", URL: "https://example.invalid/0.15.0.jar"},
	}

	installer, err := latestQuiltInstaller(installers)
	if err != nil {
		t.Fatal(err)
	}
	if installer.Version != "0.15.1" {
		t.Errorf("got %q, want 0.15.1", installer.Version)
	}

	if _, err := latestQuiltInstaller(nil); err == nil {
		t.Error("sin instaladores debería dar error")
	}
}
