package playit

import (
	"fmt"
	"minecraft-manager/internal/config"
	"os"
	"os/exec"
	"path/filepath"
)

type TunnelManager struct {
	process *os.Process
}

func New() *TunnelManager {
	return &TunnelManager{}
}

func (tm *TunnelManager) Start(cfg *config.Config) error {
	if _, err := os.Stat(cfg.PlayitPath); os.IsNotExist(err) {
		return nil
	}

	absPath, err := filepath.Abs(cfg.PlayitPath)

	fmt.Println("[*] Iniciando servicio Playit.gg (segundo plano)...")

	cmd := exec.Command(absPath)

	logFile, err := os.Create("playit.log")
	if err != nil {
		fmt.Printf("[-] No se pudo crear playit.log: %v\n", err)
	} else {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error al arrancar playit: %w", err)
	}

	tm.process = cmd.Process
	fmt.Printf("[+] Playit activo (PID: %d). Logs en 'playit.log'\n", tm.process.Pid)
	return nil
}

func (tm *TunnelManager) Stop() {
	if tm.process != nil {
		fmt.Println("[*] Deteniendo Playit...")
		if err := tm.process.Kill(); err != nil {
		}
		tm.process.Wait()
	}
}
