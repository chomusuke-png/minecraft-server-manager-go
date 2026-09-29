package runner

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/java"
	"minecraft-manager/internal/logx"
)

type Runner struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Runner {
	return &Runner{
		cfg: cfg,
	}
}

func (r *Runner) Start(instanceDir string) {
	meta, err := instance.LoadMeta(instanceDir)
	if err != nil {
		// Instancias viejas sin instance.json siguen arrancando por el camino
		// clásico de '-jar server.jar'.
		meta = &instance.InstanceMeta{}
	}

	if err := r.verifyLaunchTarget(instanceDir, meta); err != nil {
		logx.Error("%v", err)
		return
	}

	// arranca antes de resolver Java porque reemplazar un java faltante ya
	// necesita preguntar
	stdinLines := make(chan string)
	go forwardStdin(stdinLines)

	javaPath := r.resolveJava(meta)
	if java.Missing(javaPath) {
		if !r.replaceMissingJava(instanceDir, meta, javaPath, stdinLines) {
			return
		}
		javaPath = r.resolveJava(meta)
	}

	ramGB := r.resolveRAM(meta)
	javaArgs := r.buildJavaArgs(meta, ramGB)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	for {
		logx.Info("INICIANDO SERVIDOR (%dGB RAM) en '%s'...", ramGB, instanceDir)

		sniffer := &javaMismatchSniffer{out: os.Stdout}
		wasStoppedIntentionally := r.runServerInstance(instanceDir, javaPath, javaArgs, sniffer, signalChannel, stdinLines)

		if wasStoppedIntentionally {
			logx.Info("Proceso de Manager finalizado limpiamente.")
			break
		}

		if r.offerJavaFix(instanceDir, meta, javaPath, sniffer.detected, stdinLines) {
			javaPath = r.resolveJava(meta)
			logx.Info("Reintentando con el nuevo runtime...")
			continue
		}

		logx.Warn("El servidor se detuvo de forma abrupta. Reiniciando en 10 segundos... (Ctrl+C para cancelar)")

		select {
		case <-time.After(10 * time.Second):
		case <-signalChannel:
			logx.Info("\nReinicio cancelado. Saliendo...")
			return
		}
	}
}

// Lee stdin una única vez durante toda la vida del proceso: abrir un
// io.Copy(stdin) nuevo por cada reinicio dejaba goroutines bloqueadas
// para siempre leyendo un stdin que nunca se cierra.
func forwardStdin(lines chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines <- scanner.Text() + "\n"
	}
	close(lines)
}

// resolveJava elige el runtime de la instancia y avisa si no coincide con lo que
// pide la versión, para que un UnsupportedClassVersionError crudo no sea la
// primera pista.
func (r *Runner) resolveJava(meta *instance.InstanceMeta) string {
	javaPath := r.cfg.JavaPath
	if meta.JavaPath != "" {
		javaPath = meta.JavaPath
		logx.Info("Java por instancia: %s", javaPath)
	}

	requirement := javaRequirement(meta)
	// si falta, Start ofrece reemplazarlo y verificarlo aca solo daria un aviso
	// confuso
	if requirement.Min == 0 || java.Missing(javaPath) {
		return javaPath
	}

	major, err := java.DetectMajor(javaPath)
	if err != nil {
		logx.Warn("No se pudo verificar la versión de '%s': %v", javaPath, err)
		return javaPath
	}
	if !requirement.Satisfies(major) {
		logx.Warn("'%s' es Java %d pero %s %s requiere %s. El servidor probablemente no arranque.",
			javaPath, major, meta.LoaderType, meta.MCVersion, requirement)
	}
	return javaPath
}

// javaRequirement es el Java que necesita la instancia: el de la tabla, subido
// al que declaro Mojang al instalarla si ese es mas alto
func javaRequirement(meta *instance.InstanceMeta) java.Requirement {
	return java.RequireFor(meta.LoaderType, meta.MCVersion).WithMojangMinimum(meta.JavaMajor)
}

// offerJavaFix ofrece conseguir el runtime correcto cuando el arranque falló por
// una incompatibilidad de Java, y persiste la elección. Devuelve true si algo
// cambió y vale la pena reintentar ya, sin esperar los 10 segundos.
func (r *Runner) offerJavaFix(instanceDir string, meta *instance.InstanceMeta, javaPath string, sawMismatch bool, stdinLines <-chan string) bool {
	requirement := javaRequirement(meta)
	if requirement.Min == 0 {
		return false
	}

	// Si la JVM no se quejó y el runtime en uso cumple, el problema es otro: RAM,
	// un mod roto, el puerto ocupado. No tiene sentido preguntar por Java.
	if !sawMismatch {
		major, err := java.DetectMajor(javaPath)
		if err == nil && requirement.Satisfies(major) {
			return false
		}
	}

	logx.Warn("El arranque falló y parece un problema de versión de Java: %s %s requiere %s.",
		meta.LoaderType, meta.MCVersion, requirement)

	return r.repairJava(instanceDir, meta, requirement, stdinLines)
}

// replaceMissingJava ofrece otro runtime cuando la ruta configurada ya no
// existe, antes de arrancar: pasa cuando el JDK se actualiza solo y cambia de
// carpeta, o cuando se movio la carpeta del programa. Devuelve false si no hay
// con que arrancar
func (r *Runner) replaceMissingJava(instanceDir string, meta *instance.InstanceMeta, javaPath string, stdinLines <-chan string) bool {
	logx.Warn("No se encuentra el Java configurado: '%s'.", javaPath)
	logx.Detail("Puede que se haya actualizado a otra carpeta o que se haya movido el programa.")

	requirement := javaRequirement(meta)
	if requirement.Min == 0 {
		// sin version reconocida no se sabe que pedir: se vuelve al java global,
		// sin guardarlo, para no pisar una eleccion hecha a mano
		if meta.JavaPath == "" || java.Missing(r.cfg.JavaPath) {
			logx.Error("Corrige java_path en instance.json o en config.json.")
			return false
		}
		logx.Info("Se usa el Java global: %s", r.cfg.JavaPath)
		meta.JavaPath = ""
		return true
	}

	return r.repairJava(instanceDir, meta, requirement, stdinLines)
}

// repairJava consigue un runtime que cumpla y lo guarda en instance.json
func (r *Runner) repairJava(instanceDir string, meta *instance.InstanceMeta, requirement java.Requirement, stdinLines <-chan string) bool {
	resolved, err := java.Repair(askFromStdinLines(stdinLines), requirement)
	if err != nil {
		logx.Error("%v", err)
		return false
	}

	meta.JavaPath = resolved
	if err := instance.SaveMeta(instanceDir, *meta); err != nil {
		logx.Warn("No se pudo guardar java_path en instance.json: %v", err)
	}
	return true
}

// askFromStdinLines lee del canal que alimenta forwardStdin. Es seguro usarlo
// antes de arrancar el servidor o después de que terminó, porque ahí nadie más
// está consumiendo el canal.
func askFromStdinLines(lines <-chan string) java.AskLine {
	return func(promptText string) (string, bool) {
		fmt.Print(promptText)
		line, ok := <-lines
		if !ok {
			return "", false
		}
		return strings.TrimSpace(line), true
	}
}

// javaMismatchSniffer deja pasar la salida del servidor a la consola y de paso
// detecta el error que tira la JVM cuando el major no corresponde. Es una
// heurística: si el marcador llegara partido entre dos Write no se detecta, y ahí
// cae el chequeo de DetectMajor de offerJavaFix.
type javaMismatchSniffer struct {
	out      io.Writer
	detected bool
}

var javaMismatchMarkers = [][]byte{
	// Java demasiado viejo para los .class del servidor.
	[]byte("UnsupportedClassVersionError"),
	[]byte("has been compiled by a more recent version of the Java Runtime"),
	[]byte("Unsupported class file major version"),
	// Java demasiado nuevo: correr un loader pre-1.17 en 17+ no falla por versión
	// de clase sino con un IllegalAccessError, porque el sistema de módulos ya no
	// exporta los internals de los que dependía modlauncher.
	[]byte("does not export sun.security.util"),
	[]byte("because module java.base does not export"),
}

func (s *javaMismatchSniffer) Write(p []byte) (int, error) {
	if !s.detected {
		for _, marker := range javaMismatchMarkers {
			if bytes.Contains(p, marker) {
				s.detected = true
				break
			}
		}
	}
	return s.out.Write(p)
}

func (r *Runner) resolveRAM(meta *instance.InstanceMeta) int {
	if meta.RAMGB > 0 {
		logx.Info("RAM configurada por instancia: %dGB", meta.RAMGB)
		return meta.RAMGB
	}
	logx.Info("RAM configurada globalmente: %dGB", r.cfg.RAMGB)
	return r.cfg.RAMGB
}

// verifyLaunchTarget confirma que exista lo que hace falta para arrancar antes de
// invocar a Java, que para un @argfile faltante falla con un error opaco.
func (r *Runner) verifyLaunchTarget(instanceDir string, meta *instance.InstanceMeta) error {
	if len(meta.LaunchArgs) == 0 {
		jarPath := filepath.Join(instanceDir, r.cfg.JarName)
		if _, err := os.Stat(jarPath); err != nil {
			return fmt.Errorf("no se encuentra %s. Ejecuta el downloader primero", jarPath)
		}
		return nil
	}

	for i, arg := range meta.LaunchArgs {
		var required string
		switch {
		case strings.HasPrefix(arg, "@"):
			required = strings.TrimPrefix(arg, "@")
		case arg == "-jar" && i+1 < len(meta.LaunchArgs):
			// Quilt arranca con su propio lanzador en vez de un args file
			required = meta.LaunchArgs[i+1]
		default:
			continue
		}
		// Java resuelve las rutas relativas al cwd, que es el dir de la instancia.
		requiredPath := filepath.Join(instanceDir, filepath.FromSlash(required))
		if _, err := os.Stat(requiredPath); err != nil {
			return fmt.Errorf(
				"falta '%s', requerido por launch_args de instance.json.\nReinstala el loader desde el menú de actualización",
				requiredPath,
			)
		}
	}
	return nil
}

// buildJavaArgs arma la línea completa de argumentos de la JVM.
func (r *Runner) buildJavaArgs(meta *instance.InstanceMeta, ramGB int) []string {
	ramArgs := []string{
		fmt.Sprintf("-Xmx%dG", ramGB),
		fmt.Sprintf("-Xms%dG", ramGB),
	}

	if len(meta.LaunchArgs) == 0 {
		return append(ramArgs, "-jar", r.cfg.JarName, "nogui")
	}

	// La RAM va después de @user_jvm_args.txt, para que el valor del manager gane
	// si el usuario descomentó un -Xmx ahí, y antes del args file del loader,
	// porque ese declara la main class y todo lo que le sigue ya es argumento del
	// programa y no de la JVM.
	insertAt := 0
	if meta.LaunchArgs[0] == "@user_jvm_args.txt" {
		insertAt = 1
	}
	return slices.Insert(slices.Clone(meta.LaunchArgs), insertAt, ramArgs...)
}

func (r *Runner) runServerInstance(dir string, javaPath string, javaArgs []string, output io.Writer, signalChannel chan os.Signal, stdinLines <-chan string) bool {
	cmd := exec.Command(java.Absolute(javaPath), javaArgs...)
	cmd.Dir = dir

	// Las dos salidas van al mismo writer para que el sniffer vea todo: el
	// UnsupportedClassVersionError sale por stderr.
	cmd.Stdout = output
	cmd.Stderr = output

	serverInputPipe, err := cmd.StdinPipe()
	if err != nil {
		logx.Error("Error obteniendo stdin: %v", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		logx.Error("Error iniciando Java: %v", err)
		return false
	}

	instanceDone := make(chan struct{})
	defer close(instanceDone)

	go func() {
		for {
			select {
			case line, ok := <-stdinLines:
				if !ok {
					return
				}
				if _, err := io.WriteString(serverInputPipe, line); err != nil {
					return
				}
			case <-instanceDone:
				return
			}
		}
	}()

	serverExitChannel := make(chan error, 1)
	go func() {
		serverExitChannel <- cmd.Wait()
	}()

	select {
	case err := <-serverExitChannel:
		if err != nil {
			logx.Error("El servidor crasheó o se cerró con error: %v", err)
			return false
		}

		logx.Info("Servidor detenido correctamente (vía comando interno).")
		return true

	case <-signalChannel:
		logx.Info("\nInterrupción detectada (Ctrl+C). Guardando el mundo de forma segura...")
		io.WriteString(serverInputPipe, "stop\n")
		<-serverExitChannel
		return true
	}
}
