package downloader

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func argsFileName() string {
	if runtime.GOOS == "windows" {
		return "win_args.txt"
	}
	return "unix_args.txt"
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// Forge >= 1.17: no hay jar ejecutable, el comando sale del args file canónico.
func TestResolveForgeLaunchModern(t *testing.T) {
	dir := t.TempDir()
	expected := filepath.Join("libraries", "net", "minecraftforge", "forge", "1.20.1-47.2.0", argsFileName())
	writeFile(t, filepath.Join(dir, expected), "-p libraries/... cpw.mods.bootstraplauncher.BootstrapLauncher")
	writeFile(t, filepath.Join(dir, "user_jvm_args.txt"), "# -Xmx4G\n")

	got, err := (&Downloader{serverDir: dir}).resolveForgeLaunch("1.20.1-47.2.0")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"@user_jvm_args.txt", "@" + filepath.ToSlash(expected), "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Sin user_jvm_args.txt no se debe pasar el @, o la JVM falla al abrirlo.
func TestResolveForgeLaunchModernWithoutUserJvmArgs(t *testing.T) {
	dir := t.TempDir()
	expected := filepath.Join("libraries", "net", "minecraftforge", "forge", "1.20.1-47.2.0", argsFileName())
	writeFile(t, filepath.Join(dir, expected), "cpw.mods.bootstraplauncher.BootstrapLauncher")

	got, err := (&Downloader{serverDir: dir}).resolveForgeLaunch("1.20.1-47.2.0")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"@" + filepath.ToSlash(expected), "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Si Forge movió el layout, el args file se resuelve parseando el run.bat generado.
func TestResolveForgeLaunchFallbackToRunScript(t *testing.T) {
	dir := t.TempDir()
	actual := filepath.Join("libraries", "net", "neoforged", "moved", argsFileName())
	writeFile(t, filepath.Join(dir, actual), "main.Class")
	writeFile(t, filepath.Join(dir, "run.bat"),
		"@ECHO OFF\r\njava @user_jvm_args.txt @libraries\\net\\neoforged\\moved\\"+argsFileName()+" %*\r\n")

	got, err := (&Downloader{serverDir: dir}).resolveForgeLaunch("1.20.1-47.2.0")
	if err != nil {
		t.Fatal(err)
	}

	// user_jvm_args.txt no existe en disco, así que no debe aparecer.
	want := []string{"@" + filepath.ToSlash(actual), "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Forge <= 1.16.5: sí hay jar ejecutable y se renombra a server.jar.
func TestResolveForgeLaunchLegacyRenamesJar(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "forge-1.16.5-36.2.34.jar"), "jar")
	writeFile(t, filepath.Join(dir, forgeInstallerName), "installer")

	got, err := (&Downloader{serverDir: dir}).resolveForgeLaunch("1.16.5-36.2.34")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("legacy no debería devolver launch args, got %q", got)
	}
	if !fileExists(filepath.Join(dir, "server.jar")) {
		t.Error("no se creó server.jar")
	}
}

// El universal jar de las versiones más viejas tiene sufijo en el nombre.
func TestResolveForgeLaunchLegacyUniversalJar(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "forge-1.12.2-14.23.5.2859-universal.jar"), "jar")
	// server.jar preexistente: os.Rename no sobreescribe en Windows.
	writeFile(t, filepath.Join(dir, "server.jar"), "viejo")

	if _, err := (&Downloader{serverDir: dir}).resolveForgeLaunch("1.12.2-14.23.5.2859"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "jar" {
		t.Errorf("no se reemplazó el server.jar viejo, got %q", content)
	}
}

func TestResolveForgeLaunchNothingInstalled(t *testing.T) {
	if _, err := (&Downloader{serverDir: t.TempDir()}).resolveForgeLaunch("1.20.1-47.2.0"); err == nil {
		t.Error("se esperaba error cuando el instalador no dejó nada usable")
	}
}

func TestRemoveForgeInstaller(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, forgeInstallerName), "installer")
	writeFile(t, filepath.Join(dir, "installer.log"), "log")
	writeFile(t, filepath.Join(dir, forgeInstallerName+".log"), "log")
	writeFile(t, filepath.Join(dir, "user_jvm_args.txt"), "keep")

	(&Downloader{serverDir: dir}).removeForgeInstaller()

	for _, leftover := range []string{forgeInstallerName, "installer.log", forgeInstallerName + ".log"} {
		if fileExists(filepath.Join(dir, leftover)) {
			t.Errorf("quedó sin borrar: %s", leftover)
		}
	}
	if !fileExists(filepath.Join(dir, "user_jvm_args.txt")) {
		t.Error("se borró user_jvm_args.txt, que el servidor necesita")
	}
}
