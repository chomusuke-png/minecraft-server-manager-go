package mods

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestIsForgeModClientOnly(t *testing.T) {
	newMeta := func(modID string, deps ...forgeDependency) forgeModsToml {
		meta := forgeModsToml{
			Mods: []struct {
				ModID string `toml:"modId"`
			}{{ModID: modID}},
		}
		if deps != nil {
			meta.Dependencies = map[string][]forgeDependency{modID: deps}
		}
		return meta
	}

	cases := []struct {
		name string
		meta forgeModsToml
		want bool
	}{
		{
			name: "se autodeclara CLIENT",
			meta: newMeta("examplemod", forgeDependency{ModID: "examplemod", Side: "CLIENT"}),
			want: true,
		},
		{
			name: "se autodeclara BOTH",
			meta: newMeta("examplemod", forgeDependency{ModID: "examplemod", Side: "BOTH"}),
			want: false,
		},
		{
			name: "sin autodeclaración, solo dependencia de forge",
			meta: newMeta("examplemod", forgeDependency{ModID: "forge", Side: "BOTH"}),
			want: false,
		},
		{
			name: "sin bloque de dependencias",
			meta: newMeta("examplemod"),
			want: false,
		},
		{
			name: "sin mods declarados",
			meta: forgeModsToml{},
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isForgeModClientOnly(c.meta); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func writeJar(t *testing.T, path string, entryName string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	entry, err := writer.Create(entryName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGetForgeModEnvironmentClientOnly(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "clientmod.jar")
	writeJar(t, jarPath, "META-INF/mods.toml", `
[[mods]]
modId="clientmod"

[[dependencies.clientmod]]
modId="clientmod"
side="CLIENT"
`)

	env, err := getForgeModEnvironment(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	if env != "client" {
		t.Errorf("got %q, want %q", env, "client")
	}
}

func TestGetForgeModEnvironmentBoth(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "servermod.jar")
	writeJar(t, jarPath, "META-INF/mods.toml", `
[[mods]]
modId="servermod"

[[dependencies.servermod]]
modId="forge"
side="BOTH"
`)

	env, err := getForgeModEnvironment(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	if env != "*" {
		t.Errorf("got %q, want %q", env, "*")
	}
}

func TestGetForgeModEnvironmentPrefersNeoForgeDescriptor(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "crossmod.jar")

	file, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)

	forgeEntry, err := writer.Create("META-INF/mods.toml")
	if err != nil {
		t.Fatal(err)
	}
	forgeEntry.Write([]byte(`
[[mods]]
modId="crossmod"

[[dependencies.crossmod]]
modId="forge"
side="BOTH"
`))

	neoEntry, err := writer.Create("META-INF/neoforge.mods.toml")
	if err != nil {
		t.Fatal(err)
	}
	neoEntry.Write([]byte(`
[[mods]]
modId="crossmod"

[[dependencies.crossmod]]
modId="crossmod"
side="CLIENT"
`))

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	file.Close()

	env, err := getForgeModEnvironment(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	if env != "client" {
		t.Errorf("got %q, want %q (debería haber leído neoforge.mods.toml)", env, "client")
	}
}

func TestGetForgeModEnvironmentNoDescriptor(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "notamod.jar")
	writeJar(t, jarPath, "META-INF/MANIFEST.MF", "Manifest-Version: 1.0\n")

	if _, err := getForgeModEnvironment(jarPath); err == nil {
		t.Error("se esperaba error sin mods.toml ni neoforge.mods.toml")
	}
}
