package mods

import (
	"path/filepath"
	"testing"
)

func TestGetModEnvironmentQuilt(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"solo cliente", `{"schema_version": 1, "minecraft": {"environment": "client"}}`, "client"},
		{"solo servidor", `{"schema_version": 1, "minecraft": {"environment": "dedicated_server"}}`, "dedicated_server"},
		{"sin environment", `{"schema_version": 1, "quilt_loader": {"id": "examplemod"}}`, "*"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			jarPath := filepath.Join(t.TempDir(), "quiltmod.jar")
			writeJar(t, jarPath, "quilt.mod.json", c.content)

			env, err := getModEnvironment(jarPath)
			if err != nil {
				t.Fatal(err)
			}
			if env != c.want {
				t.Errorf("got %q, want %q", env, c.want)
			}
		})
	}
}

func TestGetModEnvironmentFabric(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "fabricmod.jar")
	writeJar(t, jarPath, "fabric.mod.json", `{"id": "examplemod", "environment": "client"}`)

	env, err := getModEnvironment(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	if env != "client" {
		t.Errorf("got %q, want %q", env, "client")
	}
}
