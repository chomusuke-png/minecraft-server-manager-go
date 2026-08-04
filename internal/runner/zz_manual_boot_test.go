//go:build manual

package runner

import (
	"bufio"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/instance"
	"minecraft-manager/internal/java"
)

// Arranque real del servidor usando el comando que arma buildJavaArgs. Correr con:
//
//	go test -tags manual ./internal/runner/ -run TestManualBoot -v -count=1 -timeout 15m
func TestManualBootForge(t *testing.T) {
	t.Chdir("../..")

	dir := filepath.Join("instances", "test-1191")
	meta, err := instance.LoadMeta(dir)
	if err != nil {
		t.Skipf("no hay instancia de prueba: %v", err)
	}

	r := New(&config.Config{JavaPath: "java", JarName: "server.jar", RAMGB: 2})

	if err := r.verifyLaunchTarget(dir, meta); err != nil {
		t.Fatalf("verifyLaunchTarget: %v", err)
	}

	javaPath := r.resolveJava(meta)
	javaArgs := r.buildJavaArgs(meta, r.resolveRAM(meta))
	t.Logf("comando: %s %s", java.Absolute(javaPath), strings.Join(javaArgs, " "))

	cmd := exec.Command(java.Absolute(javaPath), javaArgs...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout.(io.Writer)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("no arrancó Java: %v", err)
	}

	booted := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			t.Log(line)
			// "Done (12.345s)! For help, type help"
			if strings.Contains(line, `Done (`) && strings.Contains(line, "For help") {
				select {
				case booted <- line:
				default:
				}
			}
		}
	}()

	select {
	case line := <-booted:
		t.Logf("\n>>> SERVIDOR ARRIBA: %s", line)
		io.WriteString(stdin, "stop\n")
	case <-time.After(10 * time.Minute):
		cmd.Process.Kill()
		t.Fatal("el servidor no llegó a 'Done' en 10 minutos")
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("el servidor no cerró limpio: %v", err)
	}
	t.Log(">>> apagado limpio con 'stop', exit 0")
}
