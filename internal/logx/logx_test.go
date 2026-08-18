package logx

import (
	"io"
	"os"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = original

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestInfo(t *testing.T) {
	got := captureStdout(t, func() { Info("hola %s", "mundo") })
	if got != "[*] hola mundo\n" {
		t.Errorf("got %q", got)
	}
}

func TestSuccess(t *testing.T) {
	got := captureStdout(t, func() { Success("listo") })
	if got != "[+] listo\n" {
		t.Errorf("got %q", got)
	}
}

func TestWarn(t *testing.T) {
	got := captureStdout(t, func() { Warn("cuidado") })
	if got != "[!] cuidado\n" {
		t.Errorf("got %q", got)
	}
}

func TestError(t *testing.T) {
	got := captureStdout(t, func() { Error("fallo") })
	if got != "[-] fallo\n" {
		t.Errorf("got %q", got)
	}
}

func TestInfoLeadingNewline(t *testing.T) {
	got := captureStdout(t, func() { Info("\nsegunda sección") })
	if got != "\n[*] segunda sección\n" {
		t.Errorf("got %q", got)
	}
}

func TestDetail(t *testing.T) {
	got := captureStdout(t, func() { Detail("mod.jar deshabilitado") })
	if got != "    -> mod.jar deshabilitado\n" {
		t.Errorf("got %q", got)
	}
}
