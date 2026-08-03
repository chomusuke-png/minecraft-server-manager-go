// Package java resuelve qué runtime de Java usar para una instancia.
//
// Un único java_path global no alcanza en cuanto hay instancias de distintas
// eras: lo anterior a 1.17 necesita Java 8 y no corre en 17+, mientras que 1.20.5+
// pide 21. Los rangos son mutuamente excluyentes, así que cada instancia tiene que
// poder apuntar a su propio runtime.
package java

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"minecraft-manager/internal/logx"
)

// RuntimesRootDir guarda los JDK descargados, al lado de instances/.
const RuntimesRootDir = "runtimes"

// Requirement es el rango de majors de Java que sirven para una instancia.
// Min 0 significa que no se pudo determinar y no se valida nada; Max 0 significa
// que no hay tope superior.
type Requirement struct {
	Min int
	Max int
}

// Require deduce el Java necesario a partir de la versión de Minecraft.
//
// El tope superior sólo existe para la era pre-1.17, que es anterior al sistema
// de módulos de Java y no corre en 17+. De 1.17 en adelante no se pone tope: está
// verificado que Forge 1.19.1 (que pide 17) arranca sin problemas en Java 21, así
// que tratar el requisito como exacto haría descargar un runtime al vacío.
func Require(mcVersion string) Requirement {
	minor, patch, ok := parseMCVersion(mcVersion)
	if !ok {
		return Requirement{}
	}

	switch {
	case minor < 17:
		return Requirement{Min: 8, Max: 11}
	case minor < 20, minor == 20 && patch < 5:
		return Requirement{Min: 17}
	default:
		return Requirement{Min: 21}
	}
}

func (r Requirement) Satisfies(major int) bool {
	if r.Min == 0 {
		return true
	}
	if major < r.Min {
		return false
	}
	return r.Max == 0 || major <= r.Max
}

func (r Requirement) String() string {
	switch {
	case r.Min == 0:
		return "cualquier Java"
	case r.Max == 0:
		return fmt.Sprintf("Java %d o superior", r.Min)
	case r.Max == r.Min:
		return fmt.Sprintf("Java %d", r.Min)
	default:
		return fmt.Sprintf("Java %d a %d", r.Min, r.Max)
	}
}

// parseMCVersion saca el minor y el patch de un "1.<minor>.<patch>".
func parseMCVersion(mcVersion string) (minor int, patch int, ok bool) {
	parts := strings.Split(strings.TrimSpace(mcVersion), ".")
	if len(parts) < 2 || parts[0] != "1" {
		return 0, 0, false
	}

	minor, ok = leadingInt(parts[1])
	if !ok {
		return 0, 0, false
	}

	if len(parts) > 2 {
		// Ignora sufijos tipo "5-pre1": alcanza el número de adelante.
		patch, _ = leadingInt(parts[2])
	}
	return minor, patch, true
}

func leadingInt(value string) (int, bool) {
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	parsed, err := strconv.Atoi(value[:end])
	return parsed, err == nil
}

// java -version reporta 1.8.0_502 en Java 8 y 17.0.20 de Java 9 en adelante.
var reportedVersionPattern = regexp.MustCompile(`version "(\d+)(?:\.(\d+))?`)

// DetectMajor ejecuta el binario y devuelve su major de Java.
func DetectMajor(javaPath string) (int, error) {
	// java -version escribe en stderr, no en stdout.
	output, err := exec.Command(javaPath, "-version").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("no se pudo ejecutar '%s': %w", javaPath, err)
	}

	match := reportedVersionPattern.FindSubmatch(output)
	if match == nil {
		return 0, fmt.Errorf("no se pudo interpretar la versión de '%s': %s", javaPath, output)
	}

	first, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, err
	}
	if first != 1 {
		return first, nil
	}

	second, err := strconv.Atoi(string(match[2]))
	if err != nil {
		return 0, fmt.Errorf("versión legacy ilegible: %s", output)
	}
	return second, nil
}

// AskLine lee una línea de respuesta del usuario. Se inyecta en vez de recibir un
// *bufio.Reader porque el runner ya le cedió os.Stdin a su forwarder y tiene que
// leer del canal compartido: un reader nuevo competiría por la misma entrada.
// El segundo valor es false cuando ya no se puede leer más (EOF).
type AskLine func(promptText string) (string, bool)

// FromReader adapta el bufio.Reader compartido a AskLine.
func FromReader(reader *bufio.Reader) AskLine {
	return func(promptText string) (string, bool) {
		fmt.Print(promptText)
		raw, err := reader.ReadString('\n')
		line := strings.TrimSpace(raw)
		if err != nil && line == "" {
			return "", false
		}
		return line, true
	}
}

// Resolve elige el runtime de una instancia que se está creando o actualizando.
// Si el Java del sistema ya cumple, ofrece igual la opción de atar la instancia a
// un runtime propio; si no cumple, va directo a conseguir uno.
func Resolve(reader *bufio.Reader, req Requirement, candidate string) (string, error) {
	ask := FromReader(reader)

	if req.Min == 0 {
		// Versión no reconocida: no hay nada que validar.
		return candidate, nil
	}

	logx.Info("Esta instancia requiere %s.", req)

	if candidate != "" {
		major, err := DetectMajor(candidate)
		switch {
		case err != nil:
			logx.Detail("'%s' no está disponible.", candidate)
		case req.Satisfies(major):
			if !wantsDedicated(ask, candidate, major) {
				return candidate, nil
			}
		default:
			logx.Warn("'%s' es Java %d y no cumple el requisito.", candidate, major)
		}
	}

	if found := findInRuntimes(req); found != "" {
		logx.Detail("Usando runtime ya descargado: %s", found)
		return found, nil
	}

	return obtain(ask, req)
}

// Repair consigue un runtime después de que el arranque falló. Recibe el AskLine
// porque el runner no puede abrir un reader sobre stdin.
func Repair(ask AskLine, req Requirement) (string, error) {
	if found := findInRuntimes(req); found != "" {
		logx.Detail("Hay un runtime que sí cumple: %s", found)
		return found, nil
	}
	return obtain(ask, req)
}

// wantsDedicated pregunta si atar la instancia a su propio runtime aunque el Java
// del sistema alcance. Sirve para no romper instancias viejas cuando se actualiza
// el Java global de la máquina.
func wantsDedicated(ask AskLine, candidate string, major int) bool {
	fmt.Println("\n[?] Java para esta instancia:")
	fmt.Printf("  1) Usar el del sistema ('%s', Java %d)\n", candidate, major)
	fmt.Println("  2) Instalar un Java dedicado para esta instancia")

	choice, ok := choose(ask, "\n[?] Opción [1-2]: ", "1", "2")
	return ok && choice == "2"
}

// obtain consigue un runtime que cumpla req, automáticamente o a mano.
func obtain(ask AskLine, req Requirement) (string, error) {
	fmt.Printf("\n[?] ¿Cómo conseguir %s?\n", req)
	if runtime.GOOS == "windows" {
		fmt.Printf("  1) Descargarlo automáticamente de Adoptium (queda en %s/)\n", RuntimesRootDir)
	} else {
		fmt.Println("  1) (no disponible: la descarga automática sólo está implementada para Windows)")
	}
	fmt.Println("  2) Indicar la ruta a un Java que ya tengas instalado")
	fmt.Println("  3) Cancelar")

	choice, ok := choose(ask, "\n[?] Opción [1-3]: ", "1", "2", "3")
	if !ok || choice == "3" {
		return "", fmt.Errorf("hace falta %s para esta instancia", req)
	}

	if choice == "1" {
		if runtime.GOOS != "windows" {
			logx.Error("La descarga automática sólo está implementada para Windows.")
			return askManualPath(ask, req)
		}
		return Download(req.Min)
	}

	return askManualPath(ask, req)
}

func askManualPath(ask AskLine, req Requirement) (string, error) {
	for {
		promptText := fmt.Sprintf("[?] Ruta al ejecutable de %s (vacío para cancelar): ", req)
		path, ok := ask(promptText)
		if !ok || path == "" {
			return "", fmt.Errorf("hace falta %s para esta instancia", req)
		}

		// Arrastrar desde el explorador de Windows pega la ruta entre comillas.
		path = strings.Trim(path, `"`)

		major, err := DetectMajor(path)
		if err != nil {
			logx.Error("No se pudo ejecutar '%s': %v", path, err)
			continue
		}
		if !req.Satisfies(major) {
			logx.Error("'%s' es Java %d, y hace falta %s.", path, major, req)
			continue
		}

		logx.Success("Java %d validado en '%s'.", major, path)
		return path, nil
	}
}

func choose(ask AskLine, promptText string, valid ...string) (string, bool) {
	for {
		answer, ok := ask(promptText)
		if !ok {
			logx.Error("\nNo se pudo leer la entrada.")
			return "", false
		}
		if slices.Contains(valid, answer) {
			return answer, true
		}
		logx.Error("Entrada incorrecta, reintente.")
	}
}

// findInRuntimes busca un java ya descargado que cumpla req. Los zips de Adoptium
// traen un directorio raíz con la versión adentro, así que se prueban los dos
// niveles en vez de asumir la profundidad.
func findInRuntimes(req Requirement) string {
	patterns := []string{
		filepath.Join(RuntimesRootDir, "*", "bin", binaryName()),
		filepath.Join(RuntimesRootDir, "*", "*", "bin", binaryName()),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			major, err := DetectMajor(match)
			if err != nil {
				continue
			}
			if req.Satisfies(major) {
				return match
			}
		}
	}
	return ""
}

// Absolute expande una ruta de java relativa a absoluta.
//
// Hace falta porque tanto el instalador de Forge como el servidor se lanzan con
// cmd.Dir apuntando al directorio de la instancia: una ruta como
// "runtimes/jdk-17/.../java.exe" se buscaría desde ahí y no desde la raíz del
// proyecto. En instance.json se guarda relativa para que el proyecto siga siendo
// portable, y se expande recién al momento de ejecutar.
func Absolute(javaPath string) string {
	// Un comando pelado como "java" lo resuelve el PATH, no hay que tocarlo.
	if javaPath == "" || !strings.ContainsAny(javaPath, `/\`) {
		return javaPath
	}
	absolute, err := filepath.Abs(javaPath)
	if err != nil {
		return javaPath
	}
	return absolute
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}
