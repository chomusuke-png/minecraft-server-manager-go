//go:build !windows

package playit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// un playit falso que imprime los argumentos con los que lo lanzaron y el link
// de vinculacion con colores, como el real
const fakePlayit = `#!/bin/sh
echo "argumentos: $@"
printf '\033[2mfecha\033[0m \033[32m INFO\033[0m Visit link to setup https://playit.gg/claim/falso\n'
`

func TestLaunchPasaStdoutYDejaElLogSinColores(t *testing.T) {
	t.Chdir(t.TempDir())

	binary, err := filepath.Abs("playit-falso")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte(fakePlayit), 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := launch(binary); err != nil {
		t.Fatal(err)
	}

	// launch no espera al proceso: el log se completa cuando el falso termina
	var content string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(logPath)
		content = string(data)
		if strings.Contains(content, "playit.gg/claim/falso") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// sin --stdout playit abre su interfaz interactiva y no escribe nada
	if !strings.Contains(content, "argumentos: --stdout") {
		t.Errorf("playit no recibió --stdout. Log: %q", content)
	}
	if !strings.Contains(content, "INFO Visit link to setup https://playit.gg/claim/falso") {
		t.Errorf("falta el link en el log: %q", content)
	}
	if strings.Contains(content, "\x1b") {
		t.Errorf("el log no debería tener códigos de color: %q", content)
	}
}
