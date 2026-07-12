package playit

import (
	"fmt"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/logx"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

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

	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"(Start-Process -FilePath $env:MCM_PLAYIT_PATH -PassThru).Id")
	cmd.Env = append(os.Environ(), "MCM_PLAYIT_PATH="+absolutePlayitPath)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error al lanzar ventana de playit: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return fmt.Errorf("no se pudo obtener el PID de playit: %w", err)
	}
	tm.pid = pid

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
