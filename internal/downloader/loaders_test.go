package downloader

import "testing"

func TestLoaderLabel(t *testing.T) {
	if got, ok := LoaderLabel("neoforge"); !ok || got != "NeoForge" {
		t.Errorf("LoaderLabel(neoforge) = (%q, %v)", got, ok)
	}
	if _, ok := LoaderLabel("spigot"); ok {
		t.Error("un loader que no existe no debería tener etiqueta")
	}
}

func TestLoaderByChoice(t *testing.T) {
	cases := []struct {
		input string
		want  string
		ok    bool
	}{
		{"1", "paper", true},
		{"5", "vanilla", true},
		{"6", "", false},
		{"0", "", false},
		{"-1", "", false},
		{"", "", false},
		{"dos", "", false},
	}

	for _, c := range cases {
		got, ok := LoaderByChoice(c.input)
		if got != c.want || ok != c.ok {
			t.Errorf("LoaderByChoice(%q) = (%q, %v), want (%q, %v)", c.input, got, ok, c.want, c.ok)
		}
	}
}

// los menus numeran a partir de Loaders, asi que el orden es parte del contrato
func TestLoadersMantieneElOrdenDeLosMenus(t *testing.T) {
	want := []string{"paper", "fabric", "forge", "neoforge", "vanilla"}

	if len(Loaders) != len(want) {
		t.Fatalf("hay %d loaders, se esperaban %d", len(Loaders), len(want))
	}
	for i, loaderType := range want {
		if Loaders[i].Type != loaderType {
			t.Errorf("Loaders[%d] = %q, want %q", i, Loaders[i].Type, loaderType)
		}
	}
}

// si se suma un loader nuevo hay que darle tambien su resolver de versiones,
// salvo que sea como vanilla y no tenga version propia
func TestTodosLosLoadersTienenResolver(t *testing.T) {
	for _, loader := range Loaders {
		if loader.Type == "vanilla" {
			continue
		}
		if _, ok := loaderVersionResolvers[loader.Type]; !ok {
			t.Errorf("%s no tiene resolver de versiones", loader.Type)
		}
	}

	for loaderType := range loaderVersionResolvers {
		if _, ok := LoaderLabel(loaderType); !ok {
			t.Errorf("%s tiene resolver pero no está en Loaders", loaderType)
		}
	}
}
