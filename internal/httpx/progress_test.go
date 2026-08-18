package httpx

import (
	"strings"
	"testing"
	"time"
)

func TestFormatMB(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0.0MB"},
		{1024 * 1024, "1.0MB"},
		{5 * 1024 * 1024, "5.0MB"},
		{1024 * 1024 / 2, "0.5MB"},
	}
	for _, c := range cases {
		if got := formatMB(c.bytes); got != c.want {
			t.Errorf("formatMB(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}

func TestFormatSpeedNoElapsedTime(t *testing.T) {
	if got := formatSpeed(1000, 0); got != "-- KB/s" {
		t.Errorf("got %q", got)
	}
}

func TestFormatSpeedKB(t *testing.T) {
	// 100 KB en 1 segundo.
	got := formatSpeed(100*1024, time.Second)
	if got != "100 KB/s" {
		t.Errorf("got %q", got)
	}
}

func TestFormatSpeedMB(t *testing.T) {
	// 5 MB en 1 segundo.
	got := formatSpeed(5*1024*1024, time.Second)
	if got != "5.0 MB/s" {
		t.Errorf("got %q", got)
	}
}

func TestFormatProgressLineContainsExpectedParts(t *testing.T) {
	total := int64(10 * 1024 * 1024)
	read := int64(5 * 1024 * 1024) // 50%

	line := formatProgressLine(read, total, time.Second)

	for _, want := range []string{"\r", "50.0%", "5.0MB/10.0MB"} {
		if !strings.Contains(line, want) {
			t.Errorf("falta %q en %q", want, line)
		}
	}
}

func TestFormatProgressLineBarFillsProportionally(t *testing.T) {
	total := int64(100)

	empty := formatProgressLine(0, total, time.Second)
	if strings.Count(empty, "█") != 0 {
		t.Errorf("barra vacía no debería tener bloques llenos: %q", empty)
	}
	if strings.Count(empty, "░") != progressBarWidth {
		t.Errorf("barra vacía debería estar toda vacía: %q", empty)
	}

	full := formatProgressLine(100, total, time.Second)
	if strings.Count(full, "█") != progressBarWidth {
		t.Errorf("barra completa debería estar toda llena: %q", full)
	}
}

func TestFormatProgressLineClampsOverflow(t *testing.T) {
	line := formatProgressLine(150, 100, time.Second)
	if !strings.Contains(line, "100.0%") {
		t.Errorf("debería clampear a 100%%: %q", line)
	}
	if strings.Count(line, "█") != progressBarWidth {
		t.Errorf("la barra debería estar toda llena: %q", line)
	}
}

func TestProgressReaderPassesThroughDataUnchanged(t *testing.T) {
	pr := &ProgressReader{Reader: strings.NewReader("contenido de prueba"), Total: 20}

	buf := make([]byte, 64)
	n, err := pr.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Fatal(err)
	}
	if string(buf[:n]) != "contenido de prueba" {
		t.Errorf("got %q", buf[:n])
	}
}

func TestProgressReaderNoOutputWithoutTotal(t *testing.T) {
	pr := &ProgressReader{Reader: strings.NewReader("x"), Total: 0}
	buf := make([]byte, 8)
	if _, err := pr.Read(buf); err != nil && err.Error() != "EOF" {
		t.Fatal(err)
	}
}
