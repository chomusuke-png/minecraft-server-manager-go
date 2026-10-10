//go:build windows

package playit

import (
	"fmt"
	"minecraft-manager/internal/logx"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// launch abre playit en su propia ventana de consola, vía PowerShell, para
// que el usuario pueda ver su estado (y, en el primer arranque, el link para
// vincular la cuenta) mientras el manager sigue con lo suyo.
func launch(absolutePlayitPath string) (int, error) {
	logx.Info("Lanzando Playit...")

	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"(Start-Process -FilePath $env:MCM_PLAYIT_PATH -PassThru).Id")
	cmd.Env = append(os.Environ(), "MCM_PLAYIT_PATH="+absolutePlayitPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error al lanzar ventana de playit: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("no se pudo obtener el PID de playit: %w", err)
	}
	return pid, nil
}

func kill(pid int) {
	logx.Info("Cerrando ventana de Playit...")

	killCommand := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid), "/T")
	killCommand.Stdout = nil
	killCommand.Stderr = nil
	_ = killCommand.Run()
}

func isAlive(pid int) bool {
	output, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), strconv.Itoa(pid))
}
