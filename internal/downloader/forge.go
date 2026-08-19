package downloader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"minecraft-manager/internal/java"
	"minecraft-manager/internal/logx"
)

const forgeInstallerName = "forge-installer.jar"

type forgeLikeSpec struct {
	installerName string
	libraryGroup  []string
	// vacio = el loader no tiene era legacy: NeoForge arranca en MC 1.20.2, ya
	// con args file
	legacyJarPrefix string
}

var forgeSpec = forgeLikeSpec{
	installerName:   forgeInstallerName,
	libraryGroup:    []string{"net", "minecraftforge", "forge"},
	legacyJarPrefix: "forge",
}

// exigir el separador descarta el @user_jvm_args.txt que aparece antes en la
// misma linea
var argsFilePattern = regexp.MustCompile(`@(\S*[/\\](?:win|unix)_args\.txt)`)

func (d *Downloader) installForge(fullVersion string) ([]string, error) {
	return d.installForgeLike(forgeSpec, fullVersion)
}

func (d *Downloader) installForgeLike(spec forgeLikeSpec, fullVersion string) ([]string, error) {
	logx.Info("Ejecutando el instalador (--installServer)...")
	logx.Detail("Descarga las librerías del loader; puede tardar varios minutos.")

	installer := exec.Command(java.Absolute(d.javaPath), "-jar", spec.installerName, "--installServer", ".")
	installer.Dir = d.serverDir
	installer.Stdout = os.Stdout
	installer.Stderr = os.Stderr

	if err := installer.Run(); err != nil {
		return nil, fmt.Errorf("el instalador falló: %w", err)
	}

	launchArgs, err := d.resolveForgeLikeLaunch(spec, fullVersion)
	if err != nil {
		return nil, err
	}

	d.removeForgeLikeInstaller(spec)
	return launchArgs, nil
}

func (d *Downloader) resolveForgeLaunch(fullVersion string) ([]string, error) {
	return d.resolveForgeLikeLaunch(forgeSpec, fullVersion)
}

// resolveForgeLikeLaunch determina como arranca el server recien instalado:
// desde MC 1.17 el instalador no deja ningun jar y el comando vive en
// libraries/<libraryGroup>/<v>/win_args.txt, lo mismo que invoca el run.bat
// generado; hasta MC 1.16.5 (solo Forge) deja un jar ejecutable que se
// renombra a server.jar para que lo levante el flujo normal
func (d *Downloader) resolveForgeLikeLaunch(spec forgeLikeSpec, fullVersion string) ([]string, error) {
	if argsFile := d.findForgeLikeArgsFile(spec, fullVersion); argsFile != "" {
		logx.Detail("Loader moderno detectado (sin jar ejecutable).")
		logx.Detail("Comando de arranque: %s", argsFile)

		launchArgs := []string{}
		// los instaladores viejos de la era moderna no siempre lo generan, y
		// pasarlo sin que exista hace fallar a la JVM
		if fileExists(filepath.Join(d.serverDir, "user_jvm_args.txt")) {
			launchArgs = append(launchArgs, "@user_jvm_args.txt")
		}
		return append(launchArgs, "@"+argsFile, "nogui"), nil
	}

	if spec.legacyJarPrefix == "" {
		return nil, fmt.Errorf("el instalador terminó pero no se encontró el args file de %s", fullVersion)
	}

	legacyJar := d.findForgeLikeLegacyJar(spec, fullVersion)
	if legacyJar == "" {
		return nil, fmt.Errorf(
			"el instalador terminó pero no se encontró ni el args file ni un jar ejecutable de %s",
			fullVersion,
		)
	}

	logx.Detail("Loader legacy detectado: %s", legacyJar)

	target := filepath.Join(d.serverDir, "server.jar")
	// os.Rename no sobreescribe en Windows
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("no se pudo reemplazar el server.jar existente: %w", err)
	}
	if err := os.Rename(filepath.Join(d.serverDir, legacyJar), target); err != nil {
		return nil, fmt.Errorf("no se pudo renombrar %s a server.jar: %w", legacyJar, err)
	}

	logx.Detail("Renombrado a server.jar.")
	return nil, nil
}

func (d *Downloader) findForgeLikeArgsFile(spec forgeLikeSpec, fullVersion string) string {
	argsFileName := "unix_args.txt"
	if runtime.GOOS == "windows" {
		argsFileName = "win_args.txt"
	}

	expectedParts := append([]string{"libraries"}, spec.libraryGroup...)
	expectedParts = append(expectedParts, fullVersion, argsFileName)
	expected := filepath.ToSlash(filepath.Join(expectedParts...))
	if fileExists(filepath.Join(d.serverDir, expected)) {
		return expected
	}

	// el layout cambio en algunos builds y el script generado siempre apunta
	// al args file correcto. se prueba primero el del SO actual: el del otro
	// trae separadores y classpath incompatibles aunque el archivo exista
	scripts := []string{"run.sh", "run.bat"}
	if runtime.GOOS == "windows" {
		scripts = []string{"run.bat", "run.sh"}
	}

	for _, script := range scripts {
		content, err := os.ReadFile(filepath.Join(d.serverDir, script))
		if err != nil {
			continue
		}
		match := argsFilePattern.FindSubmatch(content)
		if match == nil {
			continue
		}
		// run.bat escribe la ruta con separadores de Windows y filepath solo los
		// interpreta como tales si el binario corre en Windows
		normalized := strings.ReplaceAll(string(match[1]), `\`, "/")
		found := filepath.ToSlash(filepath.Clean(normalized))
		if !strings.HasSuffix(found, argsFileName) {
			continue
		}
		if fileExists(filepath.Join(d.serverDir, found)) {
			logx.Detail("Args file resuelto desde %s.", script)
			return found
		}
	}

	return ""
}

func (d *Downloader) findForgeLikeLegacyJar(spec forgeLikeSpec, fullVersion string) string {
	matches, err := filepath.Glob(filepath.Join(d.serverDir, spec.legacyJarPrefix+"-"+fullVersion+"*.jar"))
	if err != nil || len(matches) == 0 {
		return ""
	}

	for _, match := range matches {
		name := filepath.Base(match)
		if name != spec.installerName {
			return name
		}
	}
	return ""
}

func (d *Downloader) removeForgeInstaller() {
	d.removeForgeLikeInstaller(forgeSpec)
}

// borra el instalador y su log. no toca libraries/, user_jvm_args.txt ni
// minecraft_server.<mc>.jar: el servidor instalado depende de los tres
func (d *Downloader) removeForgeLikeInstaller(spec forgeLikeSpec) {
	leftovers := []string{
		spec.installerName,
		// el nombre del log varia entre versiones del instalador
		"installer.log",
		spec.installerName + ".log",
	}

	for _, leftover := range leftovers {
		path := filepath.Join(d.serverDir, leftover)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			logx.Warn("No se pudo eliminar '%s': %v", leftover, err)
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
