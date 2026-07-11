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
		fmt.Printf("[-] Error: No se encuentra %s. Ejecuta el downloader primero.\n", jarPath)
		return
	}

	ramGB := r.resolveRAM(instanceDir)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	stdinLines := make(chan string)
	go forwardStdin(stdinLines)

	for {
		fmt.Printf("[*] INICIANDO SERVIDOR (%dGB RAM) en '%s'...\n", ramGB, instanceDir)

		wasStoppedIntentionally := r.runServerInstance(instanceDir, ramGB, signalChannel, stdinLines)

		if wasStoppedIntentionally {
			fmt.Println("[*] Proceso de Manager finalizado limpiamente.")
			break
		}

		fmt.Println("[!] El servidor se detuvo de forma abrupta. Reiniciando en 10 segundos... (Ctrl+C para cancelar)")

		select {
		case <-time.After(10 * time.Second):
		case <-signalChannel:
			fmt.Println("\n[*] Reinicio cancelado. Saliendo...")
			return
		}
	}
}

// forwardStdin lee stdin una única vez durante toda la vida del proceso y
// publica cada línea en el canal para que la instancia de servidor activa
// la reenvíe a su pipe. Evita abrir un io.Copy(stdin) nuevo por reinicio,
// que quedaba bloqueado para siempre leyendo un stdin que nunca se cierra.
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
		fmt.Printf("[*] RAM configurada por instancia: %dGB\n", meta.RAMGB)
		return meta.RAMGB
	}
	fmt.Printf("[*] RAM configurada globalmente: %dGB\n", r.cfg.RAMGB)
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
		fmt.Printf("[-] Error obteniendo stdin: %v\n", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("[-] Error iniciando Java: %v\n", err)
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
			fmt.Printf("[-] El servidor crasheó o se cerró con error: %v\n", err)
			return false
		}

		fmt.Println("[*] Servidor detenido correctamente (vía comando interno).")
		return true

	case <-signalChannel:
		fmt.Println("\n[*] Interrupción detectada (Ctrl+C). Guardando el mundo de forma segura...")
		io.WriteString(serverInputPipe, "stop\n")
		<-serverExitChannel
		return true
	}
}
