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

// forgeLikeSpec agrupa lo que distingue a Forge de NeoForge: mismo instalador
// (--installServer) y mismo mecanismo de arranque (args file moderno o jar
// legacy), pero nombre de instalador y ruta de librería distintos.
type forgeLikeSpec struct {
	installerName string
	// libraryGroup es la ruta bajo libraries/ donde el instalador moderno deja
	// el args file, p.ej. {"net", "minecraftforge", "forge"}.
	libraryGroup []string
	// legacyJarPrefix es el prefijo del jar ejecutable que dejan las versiones
	// pre-1.17 (p.ej. "forge" para forge-<v>.jar). Vacío si el loader no tiene
	// era legacy: NeoForge solo existe para MC >= 1.20.2, siempre vía args file.
	legacyJarPrefix string
}

var forgeSpec = forgeLikeSpec{
	installerName:   forgeInstallerName,
	libraryGroup:    []string{"net", "minecraftforge", "forge"},
	legacyJarPrefix: "forge",
}

// El args file siempre está bajo un directorio, así que exigir el separador
// descarta el @user_jvm_args.txt que aparece antes en la misma línea.
var argsFilePattern = regexp.MustCompile(`@(\S*[/\\](?:win|unix)_args\.txt)`)

func (d *Downloader) installForge(fullVersion string) ([]string, error) {
	return d.installForgeLike(forgeSpec, fullVersion)
}

// installForgeLike corre el instalador descargado con --installServer y devuelve
// los argumentos de arranque que el runner debe usar, o nil si la instalación
// dejó un server.jar ejecutable y alcanza con el flujo normal. Forge y NeoForge
// comparten este instalador.
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

// resolveForgeLikeLaunch determina cómo se arranca el servidor recién
// instalado. Forge (y NeoForge, que hereda el mismo instalador) tienen dos
// eras y el instalador produce cosas distintas en cada una:
//
//   - MC >= 1.17 no deja ningún jar ejecutable. El classpath, el module-path y la
//     main class viven en libraries/<libraryGroup>/<v>/win_args.txt, que es
//     exactamente lo que invoca el run.bat generado. Ese archivo es el comando.
//   - MC <= 1.16.5 (solo Forge; NeoForge no llega tan atrás) sí deja un jar
//     ejecutable, que renombramos a server.jar para que el flujo normal lo
//     levante con -jar.
func (d *Downloader) resolveForgeLikeLaunch(spec forgeLikeSpec, fullVersion string) ([]string, error) {
	if argsFile := d.findForgeLikeArgsFile(spec, fullVersion); argsFile != "" {
		logx.Detail("Loader moderno detectado (sin jar ejecutable).")
		logx.Detail("Comando de arranque: %s", argsFile)

		launchArgs := []string{}
		// Los instaladores viejos de la era moderna no siempre lo generan, y
		// pasarlo sin que exista hace fallar a la JVM.
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
	// os.Rename no sobreescribe en Windows.
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("no se pudo reemplazar el server.jar existente: %w", err)
	}
	if err := os.Rename(filepath.Join(d.serverDir, legacyJar), target); err != nil {
		return nil, fmt.Errorf("no se pudo renombrar %s a server.jar: %w", legacyJar, err)
	}

	logx.Detail("Renombrado a server.jar.")
	return nil, nil
}

// findForgeLikeArgsFile devuelve la ruta del args file relativa al directorio de
// la instancia, o "" si esta versión de este loader no usa ese mecanismo.
func (d *Downloader) findForgeLikeArgsFile(spec forgeLikeSpec, fullVersion string) string {
	argsFileName := "unix_args.txt"
	if runtime.GOOS == "windows" {
		argsFileName = "win_args.txt"
	}

	// Ruta canónica: ya conocemos la versión, así que no hace falta buscar.
	expectedParts := append([]string{"libraries"}, spec.libraryGroup...)
	expectedParts = append(expectedParts, fullVersion, argsFileName)
	expected := filepath.ToSlash(filepath.Join(expectedParts...))
	if fileExists(filepath.Join(d.serverDir, expected)) {
		return expected
	}

	// Fallback a prueba de versiones: el loader movió el layout en algunos
	// builds, y el script generado siempre apunta al args file correcto.
	// run.bat y run.sh se generan juntos sin importar el SO del host, así que
	// hay que probar primero el del SO actual: usar el del otro (rutas con '\'
	// y classpath separado por ';' en vez de '/' y ':') rompería el arranque
	// aunque el archivo "exista" en disco.
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
		// El regex captura la ruta tal como la escribió el script: run.bat usa
		// '\', run.sh usa '/'. filepath.Clean/ToSlash solo reinterpretan '\'
		// como separador si el binario corre en Windows, así que se normaliza
		// a mano antes para que esto funcione en cualquier SO.
		normalized := strings.ReplaceAll(string(match[1]), `\`, "/")
		found := filepath.ToSlash(filepath.Clean(normalized))
		if !strings.HasSuffix(found, argsFileName) {
			// Este script apunta al args file del otro SO; no sirve acá.
			continue
		}
		if fileExists(filepath.Join(d.serverDir, found)) {
			logx.Detail("Args file resuelto desde %s.", script)
			return found
		}
	}

	return ""
}

// findForgeLikeLegacyJar busca el jar ejecutable que dejan los instaladores
// legacy, cuyo nombre varía entre <prefix>-<v>.jar y <prefix>-<v>-universal.jar.
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

// removeForgeLikeInstaller borra el instalador y su log para dejar la
// instancia limpia. No toca libraries/, user_jvm_args.txt ni
// minecraft_server.<mc>.jar: el servidor instalado depende de los tres.
func (d *Downloader) removeForgeLikeInstaller(spec forgeLikeSpec) {
	leftovers := []string{
		spec.installerName,
		// El instalador escribe "installer.log" a secas, no "<jar>.log": se borran
		// los dos nombres porque varía entre versiones.
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
