package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/instance"
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
	jarPath := filepath.Join(instanceDir, r.cfg.JarName)

	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		logx.Error("Error: No se encuentra %s. Ejecuta el downloader primero.", jarPath)
		return
	}

	ramGB := r.resolveRAM(instanceDir)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	stdinLines := make(chan string)
	go forwardStdin(stdinLines)

	for {
		logx.Info("INICIANDO SERVIDOR (%dGB RAM) en '%s'...", ramGB, instanceDir)

		wasStoppedIntentionally := r.runServerInstance(instanceDir, ramGB, signalChannel, stdinLines)

		if wasStoppedIntentionally {
			logx.Info("Proceso de Manager finalizado limpiamente.")
			break
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

func (r *Runner) resolveRAM(instanceDir string) int {
	meta, err := instance.LoadMeta(instanceDir)
	if err == nil && meta.RAMGB > 0 {
		logx.Info("RAM configurada por instancia: %dGB", meta.RAMGB)
		return meta.RAMGB
	}
	logx.Info("RAM configurada globalmente: %dGB", r.cfg.RAMGB)
	return r.cfg.RAMGB
}

func (r *Runner) runServerInstance(dir string, ramGB int, signalChannel chan os.Signal, stdinLines <-chan string) bool {
	maxRAMArgument := fmt.Sprintf("-Xmx%dG", ramGB)
	initialRAMArgument := fmt.Sprintf("-Xms%dG", ramGB)

	cmd := exec.Command(r.cfg.JavaPath, maxRAMArgument, initialRAMArgument, "-jar", r.cfg.JarName, "nogui")
	cmd.Dir = dir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
