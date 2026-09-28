package java

import (
	"os"
	"path/filepath"
	"testing"

	"minecraft-manager/internal/approot"
)

func TestRequire(t *testing.T) {
	cases := []struct {
		mcVersion string
		want      Requirement
	}{
		// La era pre-1.17 tiene tope: no corre en Java 17+.
		{"1.8.9", Requirement{Min: 8, Max: 11}},
		{"1.12.2", Requirement{Min: 8, Max: 11}},
		{"1.16.5", Requirement{Min: 8, Max: 11}},
		{"1.17", Requirement{Min: 17}},
		{"1.17.1", Requirement{Min: 17}},
		{"1.19.1", Requirement{Min: 17}},
		{"1.20", Requirement{Min: 17}},
		{"1.20.4", Requirement{Min: 17}},
		{"1.20.5", Requirement{Min: 21}}, // el corte de Java 21 cae en un patch, no en un minor
		{"1.20.6", Requirement{Min: 21}},
		{"1.21", Requirement{Min: 21}},
		{"1.21.4", Requirement{Min: 21}},
		{"1.20.5-pre1", Requirement{Min: 21}}, // el sufijo no debe romper el parseo del patch
		{"", Requirement{}},
		{"snapshot", Requirement{}},
		{"23w31a", Requirement{}},
	}

	for _, c := range cases {
		if got := Require(c.mcVersion); got != c.want {
			t.Errorf("Require(%q) = %+v, want %+v", c.mcVersion, got, c.want)
		}
	}
}

// Verificado a mano: Forge 1.19.1 arranca en Java 21, así que el requisito de la
// era moderna es un mínimo y no un valor exacto.
func TestModernRequirementHasNoUpperBound(t *testing.T) {
	modern := Require("1.19.1")
	for _, major := range []int{17, 21, 25} {
		if !modern.Satisfies(major) {
			t.Errorf("1.19.1 debería aceptar Java %d", major)
		}
	}
	if modern.Satisfies(8) {
		t.Error("1.19.1 no debería aceptar Java 8")
	}
}

// La era pre-1.17 sí tiene tope superior.
func TestLegacyRequirementRejectsModernJava(t *testing.T) {
	legacy := Require("1.16.5")
	for _, major := range []int{8, 11} {
		if !legacy.Satisfies(major) {
			t.Errorf("1.16.5 debería aceptar Java %d", major)
		}
	}
	for _, major := range []int{17, 21} {
		if legacy.Satisfies(major) {
			t.Errorf("1.16.5 no debería aceptar Java %d", major)
		}
	}
}

func TestRequirementString(t *testing.T) {
	cases := []struct {
		req  Requirement
		want string
	}{
		{Requirement{}, "cualquier Java"},
		{Requirement{Min: 17}, "Java 17 o superior"},
		{Requirement{Min: 8, Max: 11}, "Java 8 a 11"},
		{Requirement{Min: 8, Max: 8}, "Java 8"},
	}
	for _, c := range cases {
		if got := c.req.String(); got != c.want {
			t.Errorf("%+v.String() = %q, want %q", c.req, got, c.want)
		}
	}
}

// Una versión no reconocida no debe bloquear el arranque.
func TestUnknownRequirementSatisfiedByAnything(t *testing.T) {
	unknown := Require("23w31a")
	if unknown.Min != 0 {
		t.Fatalf("Min = %d, want 0", unknown.Min)
	}
	for _, major := range []int{8, 17, 21} {
		if !unknown.Satisfies(major) {
			t.Errorf("un requisito desconocido debería aceptar Java %d", major)
		}
	}
}

// DetectMajor tiene que entender el formato legacy 1.8.0_x y el moderno.
func TestParseMajor(t *testing.T) {
	cases := []struct {
		output string
		want   int
	}{
		{`openjdk version "17.0.20" 2026-01-20`, 17},
		{`openjdk version "1.8.0_502"`, 8},
		{`java version "21.0.12" 2025-07-15 LTS`, 21},
		// las versiones GA salen sin minor ni patch
		{`openjdk version "21" 2023-09-19`, 21},
		// java -version escribe varias lineas y la version va en la primera
		{"openjdk version \"17.0.9\" 2023-10-17\nOpenJDK Runtime Environment Temurin-17.0.9+9", 17},
	}

	for _, c := range cases {
		got, err := parseMajor([]byte(c.output))
		if err != nil {
			t.Errorf("parseMajor(%q) falló: %v", c.output, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseMajor(%q) = %d, want %d", c.output, got, c.want)
		}
	}
}

func TestParseMajorRechazaSalidasIlegibles(t *testing.T) {
	for _, output := range []string{
		"'java' no se reconoce como un comando interno o externo",
		// legacy sin el numero de despues del 1.
		`java version "1"`,
	} {
		if got, err := parseMajor([]byte(output)); err == nil {
			t.Errorf("parseMajor(%q) = %d, se esperaba error", output, got)
		}
	}
}

func TestRequireFor(t *testing.T) {
	cases := []struct {
		loaderType string
		mcVersion  string
		want       Requirement
	}{
		// sin tope propio queda igual que Require
		{"neoforge", "1.21.1", Requirement{Min: 21}},
		{"forge", "1.20.1", Requirement{Min: 17}},
		// arclight no corre en Java mas nuevos que 22
		{"arclight", "1.21.1", Requirement{Min: 21, Max: 22}},
		{"arclight", "1.20.1", Requirement{Min: 17, Max: 22}},
		// el tope de la era pre-1.17 ya es mas bajo y se respeta
		{"arclight", "1.16.5", Requirement{Min: 8, Max: 11}},
		// sin version reconocida no se valida nada
		{"arclight", "snapshot", Requirement{}},
	}

	for _, c := range cases {
		if got := RequireFor(c.loaderType, c.mcVersion); got != c.want {
			t.Errorf("RequireFor(%q, %q) = %+v, want %+v", c.loaderType, c.mcVersion, got, c.want)
		}
	}

	if RequireFor("arclight", "1.21.1").Satisfies(25) {
		t.Error("Java 25 no debería servir para Arclight")
	}
}

func TestPortable(t *testing.T) {
	root, err := filepath.Abs(approot.Dir())
	if err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(root, "runtimes", "jdk-17", "jdk-17.0.20+8", "bin", "java.exe")
	outside := filepath.Join(t.TempDir(), "bin", "java.exe")

	cases := []struct {
		name string
		path string
		want string
	}{
		{"dentro del directorio de datos queda relativa", inside, "runtimes/jdk-17/jdk-17.0.20+8/bin/java.exe"},
		{"fuera queda igual", outside, outside},
		{"un comando pelado queda igual", "java", "java"},
		{"vacia queda igual", "", ""},
	}

	for _, c := range cases {
		if got := Portable(c.path); got != c.want {
			t.Errorf("%s: Portable(%q) = %q, want %q", c.name, c.path, got, c.want)
		}
	}
}

// Absolute tiene que deshacer lo que hace Portable, sin importar el directorio
// actual: el servidor se lanza con cmd.Dir en la instancia
func TestAbsoluteAnclaAlDirectorioDeDatos(t *testing.T) {
	root, err := filepath.Abs(approot.Dir())
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "runtimes", "jdk-17", "bin", "java.exe")

	if got := Absolute("runtimes/jdk-17/bin/java.exe"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := Absolute(Portable(want)); got != want {
		t.Errorf("ida y vuelta: got %q, want %q", got, want)
	}
	if got := Absolute("java"); got != "java" {
		t.Errorf("un comando pelado no se toca: got %q", got)
	}
}

func TestMissing(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "java.exe")
	if err := os.WriteFile(existing, nil, 0755); err != nil {
		t.Fatal(err)
	}

	if Missing(existing) {
		t.Error("un archivo que existe no falta")
	}
	if !Missing(filepath.Join(t.TempDir(), "jdk-25.0.2.10-hotspot", "bin", "java.exe")) {
		t.Error("una ruta que no existe falta")
	}
	if Missing("java") {
		t.Error("un comando pelado lo resuelve el PATH, no cuenta como faltante")
	}
}

func TestPickRuntimeEligeElMajorMasBajoQueCumple(t *testing.T) {
	// el orden en que llegan es el alfabetico de las carpetas
	candidates := []runtimeCandidate{
		{path: "runtimes/jdk-11/bin/java", major: 11},
		{path: "runtimes/jdk-17/bin/java", major: 17},
		{path: "runtimes/jdk-21/bin/java", major: 21},
		{path: "runtimes/jdk-8/bin/java", major: 8},
	}

	cases := []struct {
		req  Requirement
		want string
	}{
		{Requirement{Min: 17}, "runtimes/jdk-17/bin/java"},
		{Requirement{Min: 21, Max: 22}, "runtimes/jdk-21/bin/java"},
		// antes ganaba jdk-11 por venir primero
		{Requirement{Min: 8, Max: 11}, "runtimes/jdk-8/bin/java"},
		{Requirement{Min: 25}, ""},
	}

	for _, c := range cases {
		if got := pickRuntime(candidates, c.req); got != c.want {
			t.Errorf("pickRuntime(%+v) = %q, want %q", c.req, got, c.want)
		}
	}

	if got := pickRuntime(nil, Requirement{Min: 17}); got != "" {
		t.Errorf("sin candidatos = %q, want vacío", got)
	}
}
