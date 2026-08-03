package downloader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"minecraft-manager/internal/java"
	"minecraft-manager/internal/logx"
)

const forgeInstallerName = "forge-installer.jar"

// El args file siempre está bajo un directorio, así que exigir el separador
// descarta el @user_jvm_args.txt que aparece antes en la misma línea.
var forgeArgsFilePattern = regexp.MustCompile(`@(\S*[/\\](?:win|unix)_args\.txt)`)

// installForge corre el instalador descargado con --installServer y devuelve los
// argumentos de arranque que el runner debe usar, o nil si la instalación dejó un
// server.jar ejecutable y alcanza con el flujo normal.
func (d *Downloader) installForge(fullVersion string) ([]string, error) {
	logx.Info("Ejecutando el instalador de Forge (--installServer)...")
	logx.Detail("Descarga las librerías del loader; puede tardar varios minutos.")

	installer := exec.Command(java.Absolute(d.javaPath), "-jar", forgeInstallerName, "--installServer", ".")
	installer.Dir = d.serverDir
	installer.Stdout = os.Stdout
	installer.Stderr = os.Stderr

	if err := installer.Run(); err != nil {
		return nil, fmt.Errorf("el instalador de Forge falló: %w", err)
	}

	launchArgs, err := d.resolveForgeLaunch(fullVersion)
	if err != nil {
		return nil, err
	}

	d.removeForgeInstaller()
	return launchArgs, nil
}

// resolveForgeLaunch determina cómo se arranca el servidor recién instalado.
//
// Forge tiene dos eras y el instalador produce cosas distintas en cada una:
//
//   - MC >= 1.17 no deja ningún jar ejecutable. El classpath, el module-path y la
//     main class viven en libraries/net/minecraftforge/forge/<v>/win_args.txt, que
//     es exactamente lo que invoca el run.bat generado. Ese archivo es el comando.
//   - MC <= 1.16.5 sí deja un forge-<v>.jar ejecutable, que renombramos a
//     server.jar para que el flujo normal lo levante con -jar.
func (d *Downloader) resolveForgeLaunch(fullVersion string) ([]string, error) {
	if argsFile := d.findForgeArgsFile(fullVersion); argsFile != "" {
		logx.Detail("Forge moderno detectado (sin jar ejecutable).")
		logx.Detail("Comando de arranque: %s", argsFile)

		launchArgs := []string{}
		// Los instaladores viejos de la era moderna no siempre lo generan, y
		// pasarlo sin que exista hace fallar a la JVM.
		if fileExists(filepath.Join(d.serverDir, "user_jvm_args.txt")) {
			launchArgs = append(launchArgs, "@user_jvm_args.txt")
		}
		return append(launchArgs, "@"+argsFile, "nogui"), nil
	}

	legacyJar := d.findForgeLegacyJar(fullVersion)
	if legacyJar == "" {
		return nil, fmt.Errorf(
			"el instalador terminó pero no se encontró ni el args file ni un jar ejecutable de Forge %s",
			fullVersion,
		)
	}

	logx.Detail("Forge legacy detectado: %s", legacyJar)

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

// findForgeArgsFile devuelve la ruta del args file relativa al directorio de la
// instancia, o "" si esta versión de Forge no usa ese mecanismo.
func (d *Downloader) findForgeArgsFile(fullVersion string) string {
	argsFileName := "unix_args.txt"
	if runtime.GOOS == "windows" {
		argsFileName = "win_args.txt"
	}

	// Ruta canónica: ya conocemos <mc>-<forge>, así que no hace falta buscar.
	expected := filepath.ToSlash(filepath.Join("libraries", "net", "minecraftforge", "forge", fullVersion, argsFileName))
	if fileExists(filepath.Join(d.serverDir, expected)) {
		return expected
	}

	// Fallback a prueba de versiones: Forge movió el layout en algunos builds, y
	// el script generado siempre apunta al args file correcto.
	for _, script := range []string{"run.bat", "run.sh"} {
		content, err := os.ReadFile(filepath.Join(d.serverDir, script))
		if err != nil {
			continue
		}
		match := forgeArgsFilePattern.FindSubmatch(content)
		if match == nil {
			continue
		}
		found := filepath.ToSlash(filepath.Clean(string(match[1])))
		if fileExists(filepath.Join(d.serverDir, found)) {
			logx.Detail("Args file resuelto desde %s.", script)
			return found
		}
	}

	return ""
}

// findForgeLegacyJar busca el jar ejecutable que dejan los instaladores <= 1.16.5,
// cuyo nombre varía entre forge-<v>.jar y forge-<v>-universal.jar.
func (d *Downloader) findForgeLegacyJar(fullVersion string) string {
	matches, err := filepath.Glob(filepath.Join(d.serverDir, "forge-"+fullVersion+"*.jar"))
	if err != nil || len(matches) == 0 {
		return ""
	}

	for _, match := range matches {
		name := filepath.Base(match)
		if name != forgeInstallerName {
			return name
		}
	}
	return ""
}

// removeForgeInstaller borra el instalador y su log para dejar la instancia limpia.
// No toca libraries/, user_jvm_args.txt ni minecraft_server.<mc>.jar: el servidor
// instalado depende de los tres.
func (d *Downloader) removeForgeInstaller() {
	leftovers := []string{
		forgeInstallerName,
		// El instalador escribe "installer.log" a secas, no "<jar>.log": se borran
		// los dos nombres porque varía entre versiones.
		"installer.log",
		forgeInstallerName + ".log",
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
