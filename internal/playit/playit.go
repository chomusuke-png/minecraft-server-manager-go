package playit

import (
	"fmt"
	"minecraft-manager/internal/config"
	"os"
	"os/exec"
	"path/filepath"
)

type TunnelManager struct {
}

func New() *TunnelManager {
	return &TunnelManager{}
}

func (tm *TunnelManager) Start(cfg *config.Config) error {
	if _, err := os.Stat(cfg.PlayitPath); os.IsNotExist(err) {
		return nil
	}

	absPath, err := filepath.Abs(cfg.PlayitPath)
	if err != nil {
		return fmt.Errorf("error obteniendo ruta absoluta: %w", err)
	}

	fmt.Println("[*] Lanzando Playit...")

	cmd := exec.Command("cmd", "/C", "start", "Playit Tunnel", absPath)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error al lanzar ventana de playit: %w", err)
	}

	return nil
}

func (tm *TunnelManager) Stop() {
	fmt.Println("[*] Cerrando ventanas de Playit...")

	killCmd := exec.Command("taskkill", "/F", "/IM", "playit.exe")

	killCmd.Stdout = nil
	killCmd.Stderr = nil

	_ = killCmd.Run()
}
