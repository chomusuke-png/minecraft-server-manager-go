package downloader

import (
	"path/filepath"
	"slices"
	"testing"
)

func forgeArgsPath(version string) string {
	return filepath.ToSlash(filepath.Join("libraries", "net", "minecraftforge", "forge", version, argsFileName()))
}

func neoForgeArgsPath(version string) string {
	return filepath.ToSlash(filepath.Join("libraries", "net", "neoforged", "neoforge", version, argsFileName()))
}

// el run.bat que genera el instalador, apuntando al args file dado
func writeRunScript(t *testing.T, dir, argsFile string) {
	t.Helper()
	windowsPath := filepath.FromSlash(argsFile)
	writeFile(t, filepath.Join(dir, "run.bat"), "@ECHO OFF\r\njava @user_jvm_args.txt @"+windowsPath+" %*\r\n")
}

func TestDetectInstalledLoaderForgeDesdeElScript(t *testing.T) {
	dir := t.TempDir()
	argsFile := forgeArgsPath("1.20.1-47.3.0")
	writeFile(t, filepath.Join(dir, argsFile), "cpw.mods.bootstraplauncher.BootstrapLauncher")
	writeFile(t, filepath.Join(dir, "user_jvm_args.txt"), "# -Xmx4G\n")
	writeRunScript(t, dir, argsFile)

	detected, ok := DetectInstalledLoader(dir)
	if !ok {
		t.Fatal("no detectó la instalación de Forge")
	}

	if detected.LoaderType != "forge" || detected.MCVersion != "1.20.1" || detected.LoaderVersion != "47.3.0" {
		t.Errorf("got (%q, %q, %q), want (forge, 1.20.1, 47.3.0)", detected.LoaderType, detected.MCVersion, detected.LoaderVersion)
	}

	// el mismo comando que arma resolveForgeLikeLaunch despues de instalar
	want := []string{"@user_jvm_args.txt", "@" + argsFile, "nogui"}
	if !slices.Equal(detected.LaunchArgs, want) {
		t.Errorf("LaunchArgs = %q, want %q", detected.LaunchArgs, want)
	}
}

func TestDetectInstalledLoaderNeoForgeSinScript(t *testing.T) {
	dir := t.TempDir()
	argsFile := neoForgeArgsPath("21.1.248")
	writeFile(t, filepath.Join(dir, argsFile), "cpw.mods.bootstraplauncher.BootstrapLauncher")

	detected, ok := DetectInstalledLoader(dir)
	if !ok {
		t.Fatal("con un solo args file en libraries debería detectarlo aunque no haya script")
	}

	if detected.LoaderType != "neoforge" || detected.MCVersion != "1.21.1" || detected.LoaderVersion != "21.1.248" {
		t.Errorf("got (%q, %q, %q), want (neoforge, 1.21.1, 21.1.248)", detected.LoaderType, detected.MCVersion, detected.LoaderVersion)
	}

	// sin user_jvm_args.txt no se pasa el @, o la JVM falla al abrirlo
	want := []string{"@" + argsFile, "nogui"}
	if !slices.Equal(detected.LaunchArgs, want) {
		t.Errorf("LaunchArgs = %q, want %q", detected.LaunchArgs, want)
	}
}

func TestDetectInstalledLoaderPrefiereLaVersionDelArgsFile(t *testing.T) {
	dir := t.TempDir()
	// la ruta sugiere 1.21.1, pero el args file es la fuente autoritativa
	argsFile := neoForgeArgsPath("21.1.248")
	writeFile(t, filepath.Join(dir, argsFile), "cpw.mods.bootstraplauncher.BootstrapLauncher --launchTarget forgeserver --fml.mcVersion 1.21.2 --fml.neoForgeVersion 21.1.248")

	detected, ok := DetectInstalledLoader(dir)
	if !ok {
		t.Fatal("no detectó la instalación")
	}
	if detected.MCVersion != "1.21.2" {
		t.Errorf("MCVersion = %q, se esperaba la del args file (1.21.2)", detected.MCVersion)
	}
}

func TestDetectInstalledLoaderNoAdivinaConVariosArgsFiles(t *testing.T) {
	dir := t.TempDir()
	// restos de una actualizacion y sin script que diga cual es la actual
	writeFile(t, filepath.Join(dir, forgeArgsPath("1.20.1-47.2.0")), "viejo")
	writeFile(t, filepath.Join(dir, forgeArgsPath("1.20.1-47.3.0")), "nuevo")

	if detected, ok := DetectInstalledLoader(dir); ok {
		t.Errorf("con dos candidatos no debería elegir uno, eligió %q", detected.ArgsFile)
	}
}

func TestDetectInstalledLoaderElScriptDesempataVariosArgsFiles(t *testing.T) {
	dir := t.TempDir()
	actual := forgeArgsPath("1.20.1-47.3.0")
	writeFile(t, filepath.Join(dir, forgeArgsPath("1.20.1-47.2.0")), "viejo")
	writeFile(t, filepath.Join(dir, actual), "nuevo")
	writeRunScript(t, dir, actual)

	detected, ok := DetectInstalledLoader(dir)
	if !ok {
		t.Fatal("el script dice cuál es la actual, debería detectarla")
	}
	if detected.LoaderVersion != "47.3.0" {
		t.Errorf("LoaderVersion = %q, se esperaba la del script (47.3.0)", detected.LoaderVersion)
	}
}

func TestDetectInstalledLoaderCarpetaSinLoader(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "server.properties"), "server-port=25565\n")
	writeFile(t, filepath.Join(dir, "world", "level.dat"), "mundo")

	if _, ok := DetectInstalledLoader(dir); ok {
		t.Error("no hay ningún loader instalado, no debería detectar nada")
	}
}

func TestDetectInstalledLoaderLayoutDesconocido(t *testing.T) {
	dir := t.TempDir()
	// algunos builds movieron el args file; el script sigue apuntando bien
	moved := filepath.ToSlash(filepath.Join("libraries", "otro", "lugar", argsFileName()))
	writeFile(t, filepath.Join(dir, moved), "main.Class")
	writeRunScript(t, dir, moved)

	detected, ok := DetectInstalledLoader(dir)
	if !ok {
		t.Fatal("el comando de arranque sirve aunque no se sepa de qué loader es")
	}
	if detected.LoaderType != "" || detected.LoaderVersion != "" {
		t.Errorf("no debería inventar loader ni versión: (%q, %q)", detected.LoaderType, detected.LoaderVersion)
	}
	if !slices.Equal(detected.LaunchArgs, []string{"@" + moved, "nogui"}) {
		t.Errorf("LaunchArgs = %q", detected.LaunchArgs)
	}
}

func TestNeoForgeMCVersion(t *testing.T) {
	cases := []struct {
		neoForge string
		want     string
	}{
		{"21.1.248", "1.21.1"},
		{"21.0.167", "1.21"},
		{"20.4.237", "1.20.4"},
		{"20.4.80-beta", "1.20.4"},
		{"basura", ""},
		{"21", ""},
	}
	for _, c := range cases {
		if got := neoForgeMCVersion(c.neoForge); got != c.want {
			t.Errorf("neoForgeMCVersion(%q) = %q, want %q", c.neoForge, got, c.want)
		}
	}
}

// neoForgeMCVersion tiene que ser la inversa exacta de neoForgeVersionPrefix,
// que es lo que se usa para buscar versiones al instalar
func TestNeoForgeMCVersionEsLaInversaDelPrefijo(t *testing.T) {
	for _, mcVersion := range []string{"1.20.2", "1.20.4", "1.21", "1.21.1", "1.21.4"} {
		prefix, ok := neoForgeVersionPrefix(mcVersion)
		if !ok {
			t.Fatalf("neoForgeVersionPrefix(%q) falló", mcVersion)
		}
		if got := neoForgeMCVersion(prefix + "100"); got != mcVersion {
			t.Errorf("ida y vuelta de %q dio %q", mcVersion, got)
		}
	}
}
