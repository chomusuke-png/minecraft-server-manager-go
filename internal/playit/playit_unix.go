//go:build !windows

package playit

import (
	"fmt"
	"io"
	"minecraft-manager/internal/logx"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// launch arranca playit como proceso independiente. Se le da su propio grupo
// de sesión (Setsid) para que sobreviva si el manager se cierra sin pasar por
// Release(), y así el registro de PID en disco lo pueda seguir encontrando
// (o matando) en el próximo arranque.
//
// Windows abre una ventana de consola propia para playit; acá no hay un
// equivalente portable sin asumir un emulador de terminal instalado. En vez
// de eso, la salida se refleja en esta misma consola —importante en el
// primer arranque, que es cuando playit imprime el link para vincular la
// cuenta— y además se guarda en 'playit.log' por si hace falta revisarla
// después de que la consola ya se llenó con la del servidor.
//
// Sin --stdout playit arranca su interfaz interactiva, que sin terminal no
// dibuja nada: el link de vinculacion no aparecia en ningun lado.
func launch(absolutePlayitPath string) (int, error) {
	logx.Info("Lanzando Playit (salida también en '%s')...", logPath)

	cmd := exec.Command(absolutePlayitPath, "--stdout")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	logFile, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logErr != nil {
		logx.Warn("No se pudo abrir %s, su salida solo va a esta consola: %v", logPath, logErr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// un solo writer para las dos salidas: exec usa entonces un unico pipe,
		// asi el filtro de colores no se comparte entre dos goroutines
		output := io.MultiWriter(os.Stdout, &ansiStripper{w: logFile})
		cmd.Stdout = output
		cmd.Stderr = output
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return 0, fmt.Errorf("error al lanzar playit: %w", err)
	}

	// Nadie más va a esperar a este proceso: si no se cosecha acá, queda
	// zombie cuando termine (se cierre solo o lo mate kill()).
	go func() {
		cmd.Wait()
		if logFile != nil {
			logFile.Close()
		}
	}()

	return cmd.Process.Pid, nil
}

func kill(pid int) {
	logx.Info("Cerrando Playit...")

	// Negativo = todo el grupo de proceso: Setsid dejó a playit como líder del
	// suyo propio, así que esto también alcanza a cualquier hijo que haya
	// abierto.
	pgid := -pid
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
		return
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !isAlive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	// No se apagó solo con el aviso: se corta por las malas.
	syscall.Kill(pgid, syscall.SIGKILL)
}

func isAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// En Unix, FindProcess siempre "encuentra" el proceso sin chequear nada;
	// la señal 0 no lo mata, solo confirma si sigue existiendo.
	return process.Signal(syscall.Signal(0)) == nil
}
