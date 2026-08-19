package playit

import (
	"encoding/json"
	"fmt"
	"minecraft-manager/internal/approot"
	"minecraft-manager/internal/config"
	"os"
	"path/filepath"
	"time"
)

const (
	lockStaleAfter = 5 * time.Second
	lockTimeout    = 10 * time.Second
)

var (
	registryPath = approot.Path("playit_registry.json")
	lockPath     = registryPath + ".lock"
)

type registry struct {
	PlayitPID int   `json:"playit_pid"`
	Clients   []int `json:"clients"`
}

func Acquire(cfg *config.Config) error {
	if _, err := os.Stat(cfg.PlayitPath); os.IsNotExist(err) {
		return nil
	}

	unlock, err := lock()
	if err != nil {
		return err
	}
	defer unlock()

	reg := loadRegistry()
	pruneDead(&reg)

	if reg.PlayitPID == 0 {
		absolutePlayitPath, err := filepath.Abs(cfg.PlayitPath)
		if err != nil {
			return fmt.Errorf("error obteniendo ruta absoluta: %w", err)
		}

		pid, err := launch(absolutePlayitPath)
		if err != nil {
			return err
		}
		reg.PlayitPID = pid
	}

	reg.Clients = addPID(reg.Clients, os.Getpid())
	return saveRegistry(reg)
}

func Release() {
	unlock, err := lock()
	if err != nil {
		return
	}
	defer unlock()

	reg := loadRegistry()
	reg.Clients = removePID(reg.Clients, os.Getpid())
	pruneDead(&reg)

	if len(reg.Clients) == 0 && reg.PlayitPID != 0 {
		kill(reg.PlayitPID)
		reg.PlayitPID = 0
	}

	_ = saveRegistry(reg)
}

// launch, kill e isAlive están en playit_windows.go / playit_unix.go: lanzar
// y matar un proceso "suelto" (fuera del árbol que exec.Cmd puede esperar)
// se hace con herramientas completamente distintas en cada SO.

func pruneDead(reg *registry) {
	alive := reg.Clients[:0]
	for _, pid := range reg.Clients {
		if isAlive(pid) {
			alive = append(alive, pid)
		}
	}
	reg.Clients = alive

	if reg.PlayitPID != 0 && !isAlive(reg.PlayitPID) {
		reg.PlayitPID = 0
	}
}

func addPID(pids []int, pid int) []int {
	for _, p := range pids {
		if p == pid {
			return pids
		}
	}
	return append(pids, pid)
}

func removePID(pids []int, pid int) []int {
	result := pids[:0]
	for _, p := range pids {
		if p != pid {
			result = append(result, p)
		}
	}
	return result
}

func loadRegistry() registry {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return registry{}
	}
	var reg registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return registry{}
	}
	return reg
}

func saveRegistry(reg registry) error {
	data, err := json.MarshalIndent(reg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(registryPath, data, 0644)
}

func lock() (unlock func(), err error) {
	deadline := time.Now().Add(lockTimeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			f.Close()
			return func() { os.Remove(lockPath) }, nil
		}

		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > lockStaleAfter {
			os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no se pudo obtener el lock de %s", lockPath)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
