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

// Verificación empírica de que Forge 1.19.1 SÍ arranca en Java 21, que es lo que
// justifica que el requisito de la era moderna sea un mínimo sin tope superior.
// Correr con:
//
//	go test -tags manual ./internal/runner/ -run TestManualJava21 -v -count=1 -timeout 20m
func TestManualJava21RunsForge(t *testing.T) {
	t.Chdir("../..")

	dir := filepath.Join("instances", "test-1191")
	meta, err := instance.LoadMeta(dir)
	if err != nil {
		t.Skipf("no hay instancia de prueba: %v", err)
	}

	java21, err := java.Download(21)
	if err != nil {
		t.Fatalf("Download(21): %v", err)
	}
	t.Logf("Java 21 en: %s", java21)

	major, err := java.DetectMajor(java21)
	if err != nil || major != 21 {
		t.Fatalf("DetectMajor = %d, %v; want 21", major, err)
	}

	// El requisito de 1.19.1 tiene que aceptar 21: es un mínimo, no un valor exacto.
	if req := java.Require("1.19.1"); !req.Satisfies(21) {
		t.Errorf("Require(1.19.1) = %s, y debería aceptar Java 21", req)
	}

	r := New(&config.Config{JavaPath: "java", JarName: "server.jar", RAMGB: 2})
	javaArgs := r.buildJavaArgs(meta, 2)

	cmd := exec.Command(java.Absolute(java21), javaArgs...)
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
		t.Fatalf("no arrancó Java 21: %v", err)
	}

	booted := make(chan string, 1)
	reportedJVM := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "starting: java version") {
				trySend(reportedJVM, line)
			}
			if strings.Contains(line, "Done (") && strings.Contains(line, "For help") {
				trySend(booted, line)
			}
		}
	}()

	select {
	case line := <-booted:
		t.Logf(">>> ARRANCÓ EN JAVA 21: %s", line)
		select {
		case jvm := <-reportedJVM:
			t.Logf(">>> JVM reportada por ModLauncher: %s", jvm)
		default:
		}
		io.WriteString(stdin, "stop\n")
	case <-time.After(8 * time.Minute):
		cmd.Process.Kill()
		t.Fatal("no llegó a 'Done' en 8 minutos")
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("no cerró limpio: %v", err)
	}
}

func trySend(channel chan string, value string) {
	select {
	case channel <- value:
	default:
	}
}
