package instance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func createInstance(t *testing.T, name string, meta InstanceMeta, properties string) {
	t.Helper()

	dir := filepath.Join(InstancesRootDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := SaveMeta(dir, meta); err != nil {
		t.Fatal(err)
	}
	if properties != "" {
		if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(properties), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFormatInstanceTable(t *testing.T) {
	t.Chdir(t.TempDir())

	createInstance(t, "mi_server", InstanceMeta{
		LoaderType:     "paper",
		MCVersion:      "1.20.1",
		LoaderVersion:  "133",
		RAMGB:          4,
		TunnelProvider: "playit",
	}, "motd=x\nserver-port=25565\n")

	header, rows := FormatInstanceTable([]string{"mi_server"})

	wantHeader := []string{"NOMBRE", "MINECRAFT", "LOADER", "VERSION", "RAM", "PUERTO", "TUNEL"}
	if got := strings.Fields(header); !slices.Equal(got, wantHeader) {
		t.Errorf("encabezado = %v, want %v", got, wantHeader)
	}

	wantRow := []string{"mi_server", "1.20.1", "paper", "133", "4GB", "25565", "playit"}
	if got := strings.Fields(rows[0]); !slices.Equal(got, wantRow) {
		t.Errorf("fila = %v, want %v", got, wantRow)
	}
}

func TestFormatInstanceTableSinVersionDeLoader(t *testing.T) {
	t.Chdir(t.TempDir())

	// vanilla no guarda version de loader, igual que las instancias creadas
	// antes de que existiera el campo
	createInstance(t, "puro_vanilla", InstanceMeta{
		LoaderType: "vanilla",
		MCVersion:  "1.21.8",
		RAMGB:      4,
	}, "server-port=25580\n")

	_, rows := FormatInstanceTable([]string{"puro_vanilla"})

	wantRow := []string{"puro_vanilla", "1.21.8", "vanilla", "-", "4GB", "25580", "playit"}
	if got := strings.Fields(rows[0]); !slices.Equal(got, wantRow) {
		t.Errorf("fila = %v, want %v", got, wantRow)
	}
}

func TestFormatInstanceTableAlineaLasColumnas(t *testing.T) {
	t.Chdir(t.TempDir())

	createInstance(t, "corta", InstanceMeta{LoaderType: "paper", MCVersion: "1.20.1", LoaderVersion: "133", RAMGB: 4}, "server-port=25565\n")
	createInstance(t, "un_nombre_bastante_largo", InstanceMeta{LoaderType: "neoforge", MCVersion: "1.21.4", LoaderVersion: "21.4.157", RAMGB: 8}, "server-port=25570\n")

	header, rows := FormatInstanceTable([]string{"corta", "un_nombre_bastante_largo"})

	loaderColumn := strings.Index(header, "LOADER")
	for _, row := range rows {
		if !strings.HasPrefix(row[loaderColumn:], "paper") && !strings.HasPrefix(row[loaderColumn:], "neoforge") {
			t.Errorf("la columna LOADER no arranca en la misma posición: %q", row)
		}
	}
}
