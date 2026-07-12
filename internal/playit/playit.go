package playit

import (
	"fmt"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/logx"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const createNewConsole = 0x00000010

type TunnelManager struct {
	pid int
}

func New() *TunnelManager {
	return &TunnelManager{}
}

func (tm *TunnelManager) Start(cfg *config.Config) error {
	if _, err := os.Stat(cfg.PlayitPath); os.IsNotExist(err) {
		return nil
	}

	absolutePlayitPath, err := filepath.Abs(cfg.PlayitPath)
	if err != nil {
		return fmt.Errorf("error obteniendo ruta absoluta: %w", err)
	}

	logx.Info("Lanzando Playit...")

	cmd := exec.Command(absolutePlayitPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewConsole}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error al lanzar ventana de playit: %w", err)
	}

	tm.pid = cmd.Process.Pid

	return nil
}

func (tm *TunnelManager) Stop() {
	if tm.pid == 0 {
		return
	}

	logx.Info("Cerrando ventana de Playit...")

	killCommand := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", tm.pid), "/T")

	killCommand.Stdout = nil
	killCommand.Stderr = nil

	_ = killCommand.Run()
}
