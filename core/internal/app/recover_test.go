package app

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
	"minecraft-manager/internal/instance"
)

func readerFor(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// instala un Forge 1.20.1 falso como lo deja el instalador, con su mundo, y
// devuelve la ruta del args file
func fakeForgeInstall(t *testing.T, dir string) string {
	t.Helper()

	argsFileName := "unix_args.txt"
	if runtime.GOOS == "windows" {
		argsFileName = "win_args.txt"
	}
	argsFile := filepath.ToSlash(filepath.Join("libraries", "net", "minecraftforge", "forge", "1.20.1-47.3.0", argsFileName))

	writeTestFile(t, filepath.Join(dir, argsFile), "cpw.mods.bootstraplauncher.BootstrapLauncher")
	writeTestFile(t, filepath.Join(dir, "user_jvm_args.txt"), "# -Xmx4G\n")
	writeTestFile(t, filepath.Join(dir, "world", "level.dat"), "mundo")
	return argsFile
}

func TestRecoverLoaderInstallReconstruyeInstanceJSON(t *testing.T) {
	dir := t.TempDir()
	argsFile := fakeForgeInstall(t, dir)

	if !recoverLoaderInstall(readerFor("y\n"), dir) {
		t.Fatal("aceptando, debería quedar lista para arrancar")
	}

	meta, err := instance.LoadMeta(dir)
	if err != nil {
		t.Fatalf("no se creó instance.json: %v", err)
	}
	if meta.LoaderType != "forge" || meta.MCVersion != "1.20.1" || meta.LoaderVersion != "47.3.0" {
		t.Errorf("got (%q, %q, %q)", meta.LoaderType, meta.MCVersion, meta.LoaderVersion)
	}
	want := []string{"@user_jvm_args.txt", "@" + argsFile, "nogui"}
	if !slices.Equal(meta.LaunchArgs, want) {
		t.Errorf("LaunchArgs = %q, want %q", meta.LaunchArgs, want)
	}
}

func TestRecoverLoaderInstallConservaLaConfigExistente(t *testing.T) {
	dir := t.TempDir()
	fakeForgeInstall(t, dir)

	// un instance.json que perdio el comando de arranque pero tiene config propia
	existing := instance.InstanceMeta{RAMGB: 6, TunnelProvider: "ngrok", BackupKeepMin: 5}
	if err := instance.SaveMeta(dir, existing); err != nil {
		t.Fatal(err)
	}

	if !recoverLoaderInstall(readerFor("y\n"), dir) {
		t.Fatal("aceptando, debería quedar lista para arrancar")
	}

	meta, err := instance.LoadMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if meta.RAMGB != 6 || meta.TunnelProvider != "ngrok" || meta.BackupKeepMin != 5 {
		t.Errorf("se perdió la config de la instancia: %+v", meta)
	}
	if meta.LoaderType != "forge" || len(meta.LaunchArgs) == 0 {
		t.Errorf("no se completó el loader: %+v", meta)
	}
}

func TestRecoverLoaderInstallRechazadoNoTocaNada(t *testing.T) {
	dir := t.TempDir()
	fakeForgeInstall(t, dir)

	if recoverLoaderInstall(readerFor("n\n"), dir) {
		t.Error("rechazando no debería darla por lista")
	}
	if _, err := instance.LoadMeta(dir); err == nil {
		t.Error("rechazando no debería crear instance.json")
	}
}

func TestRecoverLoaderInstallSinLoaderNoPregunta(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "world", "level.dat"), "mundo")

	// si preguntara, el "y" lo aceptaria y crearia el instance.json
	if recoverLoaderInstall(readerFor("y\n"), dir) {
		t.Error("sin loader instalado no hay nada que reconstruir")
	}
	if _, err := instance.LoadMeta(dir); err == nil {
		t.Error("no debería crear instance.json sin loader")
	}
}

func TestEnsureServerJarArrancaUnaInstanciaReconstruida(t *testing.T) {
	dir := t.TempDir()
	fakeForgeInstall(t, dir)

	cfg := &config.Config{JarName: "server.jar"}
	// si ofreciera descargar, el "n" siguiente cortaria el arranque
	if !ensureServerJar(readerFor("y\nn\n"), dir, cfg, downloader.New(dir, "java")) {
		t.Error("con la instalación reconstruida no debería pedir descargar nada")
	}
}

func TestEnsureServerJarRechazarLaReconstruccionNoBorraElMundo(t *testing.T) {
	dir := t.TempDir()
	fakeForgeInstall(t, dir)

	cfg := &config.Config{JarName: "server.jar"}
	// no a reconstruir, y no a descargar
	if ensureServerJar(readerFor("n\nn\n"), dir, cfg, downloader.New(dir, "java")) {
		t.Error("rechazando las dos cosas no debería arrancar")
	}
	if _, err := os.Stat(filepath.Join(dir, "world", "level.dat")); err != nil {
		t.Errorf("se borró el mundo: %v", err)
	}
}
