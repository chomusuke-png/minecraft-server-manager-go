package runner

import (
	"fmt"
	"minecraft-manager/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Runner struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Runner {
	return &Runner{
		cfg: cfg,
	}
}

func (r *Runner) Start() {
	serverDir := "server"
	jarPath := filepath.Join(serverDir, r.cfg.JarName)

	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		fmt.Printf("[-] Error: No se encuentra %s. Ejecuta el downloader primero.\n", jarPath)
		return
	}

	for {
		fmt.Println("\n" + "========================================")
		fmt.Printf("[*] INICIANDO SERVIDOR (%d GB RAM)...\n", r.cfg.RAMGB)
		fmt.Println("========================================")

		if err := r.runServerInstance(serverDir); err != nil {
			fmt.Printf("[-] El servidor se cerró con error: %v\n", err)
		} else {
			fmt.Println("[*] Servidor detenido correctamente.")
			break
		}

		fmt.Println("[!] El servidor se reiniciará en 10 segundos... (Ctrl+C para cancelar)")
		time.Sleep(10 * time.Second)
	}
}

func (r *Runner) runServerInstance(dir string) error {
	ramArg := fmt.Sprintf("-Xmx%dG", r.cfg.RAMGB)
	initialRamArg := fmt.Sprintf("-Xms%dG", r.cfg.RAMGB)

	cmd := exec.Command(r.cfg.JavaPath, ramArg, initialRamArg, "-jar", r.cfg.JarName, "nogui")
	cmd.Dir = dir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
