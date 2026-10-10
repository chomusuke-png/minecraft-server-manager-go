package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/java"
)

func testRunner() *Runner {
	return New(&config.Config{JavaPath: "java", JarName: "server.jar", RAMGB: 4})
}

// Sin launch_args el arranque sigue siendo el clásico de -jar.
func TestBuildJavaArgsJarLoader(t *testing.T) {
	got := testRunner().buildJavaArgs(&instance.InstanceMeta{LoaderType: "paper"}, 6)

	want := []string{"-Xmx6G", "-Xms6G", "-jar", "server.jar", "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// La RAM va después de @user_jvm_args.txt (para ganarle a un -Xmx del usuario) y
// antes del args file del loader (que declara la main class).
func TestBuildJavaArgsForgeRAMPosition(t *testing.T) {
	meta := &instance.InstanceMeta{
		LoaderType: "forge",
		LaunchArgs: []string{
			"@user_jvm_args.txt",
			"@libraries/net/minecraftforge/forge/1.20.1-47.2.0/win_args.txt",
			"nogui",
		},
	}

	got := testRunner().buildJavaArgs(meta, 8)

	want := []string{
		"@user_jvm_args.txt",
		"-Xmx8G",
		"-Xms8G",
		"@libraries/net/minecraftforge/forge/1.20.1-47.2.0/win_args.txt",
		"nogui",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildJavaArgsForgeWithoutUserJvmArgs(t *testing.T) {
	meta := &instance.InstanceMeta{
		LoaderType: "forge",
		LaunchArgs: []string{"@libraries/forge/win_args.txt", "nogui"},
	}

	got := testRunner().buildJavaArgs(meta, 4)

	want := []string{"-Xmx4G", "-Xms4G", "@libraries/forge/win_args.txt", "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// El sniffer tiene que reconocer las dos formas en que falla un Java equivocado, y
// dejar pasar la salida intacta.
func TestJavaMismatchSniffer(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{
			"java viejo",
			"java.lang.UnsupportedClassVersionError: net/minecraft/server/Main has been compiled by a more recent version of the Java Runtime",
			true,
		},
		{
			// Verificado con Forge 1.16.5 sobre Java 17.
			"java nuevo para un loader pre-1.17",
			"Exception in thread \"main\" java.lang.IllegalAccessError: class cpw.mods.modlauncher.SecureJarHandler cannot access class sun.security.util.ManifestEntryVerifier (in module java.base) because module java.base does not export sun.security.util to unnamed module",
			true,
		},
		{
			"arranque normal",
			"[Server thread/INFO]: Done (53.281s)! For help, type \"help\"",
			false,
		},
		{
			"crash no relacionado con Java",
			"java.lang.OutOfMemoryError: Java heap space",
			false,
		},
	}

	for _, c := range cases {
		var passedThrough bytes.Buffer
		sniffer := &javaMismatchSniffer{out: &passedThrough}

		if _, err := sniffer.Write([]byte(c.line)); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if sniffer.detected != c.want {
			t.Errorf("%s: detected = %v, want %v", c.name, sniffer.detected, c.want)
		}
		if passedThrough.String() != c.line {
			t.Errorf("%s: la salida no pasó intacta", c.name)
		}
	}
}

// buildJavaArgs no debe mutar el meta que después se persiste en instance.json.
func TestBuildJavaArgsDoesNotMutateMeta(t *testing.T) {
	original := []string{"@user_jvm_args.txt", "@libraries/forge/win_args.txt", "nogui"}
	meta := &instance.InstanceMeta{LoaderType: "forge", LaunchArgs: slices.Clone(original)}

	testRunner().buildJavaArgs(meta, 4)

	if !slices.Equal(meta.LaunchArgs, original) {
		t.Errorf("launch_args mutado: got %q, want %q", meta.LaunchArgs, original)
	}
}

// Quilt arranca con -jar sobre su lanzador y no con un args file: si falta hay
// que avisarlo antes de invocar a Java.
func TestVerifyLaunchTargetQuiltLauncher(t *testing.T) {
	dir := t.TempDir()
	meta := &instance.InstanceMeta{
		LoaderType: "quilt",
		LaunchArgs: []string{"-jar", "quilt-server-launch.jar", "nogui"},
	}

	if err := testRunner().verifyLaunchTarget(dir, meta); err == nil {
		t.Error("sin el lanzador de Quilt debería dar error")
	}

	if err := os.WriteFile(filepath.Join(dir, "quilt-server-launch.jar"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := testRunner().verifyLaunchTarget(dir, meta); err != nil {
		t.Errorf("con el lanzador presente no debería dar error: %v", err)
	}
}

func TestBuildJavaArgsQuiltRAMPosition(t *testing.T) {
	meta := &instance.InstanceMeta{
		LoaderType: "quilt",
		LaunchArgs: []string{"-jar", "quilt-server-launch.jar", "nogui"},
	}

	got := testRunner().buildJavaArgs(meta, 6)
	want := []string{"-Xmx6G", "-Xms6G", "-jar", "quilt-server-launch.jar", "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// sin version de Minecraft reconocida no se sabe que Java pedir: si el de la
// instancia ya no existe se vuelve al global, sin guardarlo
func TestReplaceMissingJavaVuelveAlGlobal(t *testing.T) {
	meta := &instance.InstanceMeta{JavaPath: filepath.Join(t.TempDir(), "jdk-viejo", "bin", "java.exe")}

	if !testRunner().replaceMissingJava(t.TempDir(), meta, meta.JavaPath, nil) {
		t.Fatal("debería poder seguir con el Java global")
	}
	if meta.JavaPath != "" {
		t.Errorf("java_path = %q, debería quedar vacío para usar el global", meta.JavaPath)
	}
}

func TestReplaceMissingJavaSinAlternativa(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "jdk-viejo", "bin", "java.exe")
	r := New(&config.Config{JavaPath: missing, JarName: "server.jar", RAMGB: 4})

	if r.replaceMissingJava(t.TempDir(), &instance.InstanceMeta{}, missing, nil) {
		t.Error("si el que falta es el global no hay a qué volver")
	}
}

func TestJavaRequirementUsaElJavaDeMojang(t *testing.T) {
	cases := []struct {
		name string
		meta instance.InstanceMeta
		want java.Requirement
	}{
		{"sin java_major queda la tabla", instance.InstanceMeta{LoaderType: "paper", MCVersion: "1.21.1"}, java.Requirement{Min: 21}},
		{"mojang pide mas que la tabla", instance.InstanceMeta{LoaderType: "paper", MCVersion: "26.3", JavaMajor: 26}, java.Requirement{Min: 26}},
		{"una numeracion que la tabla no conoce", instance.InstanceMeta{LoaderType: "fabric", MCVersion: "snapshot-raro", JavaMajor: 25}, java.Requirement{Min: 25}},
	}

	for _, c := range cases {
		if got := javaRequirement(&c.meta); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

// linesFor carga las respuestas en un canal cerrado, como el que alimenta
// forwardStdin cuando ya no queda entrada
func linesFor(answers ...string) <-chan string {
	lines := make(chan string, len(answers))
	for _, answer := range answers {
		lines <- answer + "\n"
	}
	close(lines)
	return lines
}

func TestFixMismatchedJavaArrancaIgualSiDiceQueNo(t *testing.T) {
	meta := &instance.InstanceMeta{LoaderType: "paper", MCVersion: "26.3", JavaPath: "java"}

	if testRunner().fixMismatchedJava(t.TempDir(), meta, linesFor("n")) {
		t.Error("diciendo que no no debería cambiar el runtime")
	}
	if meta.JavaPath != "java" {
		t.Errorf("java_path = %q, debería seguir siendo 'java'", meta.JavaPath)
	}
}

func TestFixMismatchedJavaArrancaIgualSiCancela(t *testing.T) {
	meta := &instance.InstanceMeta{LoaderType: "paper", MCVersion: "26.3", JavaPath: "java"}

	// 3 es Cancelar en el menu de como conseguir el Java
	if testRunner().fixMismatchedJava(t.TempDir(), meta, linesFor("s", "3")) {
		t.Error("cancelando no debería cambiar el runtime")
	}
	if meta.JavaPath != "java" {
		t.Errorf("java_path = %q, debería seguir siendo 'java'", meta.JavaPath)
	}
}

func TestAskYesNo(t *testing.T) {
	cases := []struct {
		name    string
		answers []string
		want    bool
	}{
		{"si", []string{"s"}, true},
		{"no", []string{"n"}, false},
		{"reintenta lo invalido", []string{"quizas", "y"}, true},
		{"sin entrada es no", nil, false},
	}

	for _, c := range cases {
		if got := askYesNo(askFromStdinLines(linesFor(c.answers...)), "[?] ¿Seguro?"); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
