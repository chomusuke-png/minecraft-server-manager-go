package httpx

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	progressBarWidth       = 30
	progressRedrawInterval = 100 * time.Millisecond
)

type ProgressReader struct {
	Reader io.Reader
	Total  int64

	read      int64
	startedAt time.Time
	lastPrint time.Time
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.read += int64(n)

	if pr.Total <= 0 {
		return n, err
	}

	if pr.startedAt.IsZero() {
		pr.startedAt = time.Now()
	}

	now := time.Now()
	if err != nil || now.Sub(pr.lastPrint) >= progressRedrawInterval {
		pr.lastPrint = now
		fmt.Print(formatProgressLine(pr.read, pr.Total, now.Sub(pr.startedAt)))
	}

	return n, err
}

func formatProgressLine(read, total int64, elapsed time.Duration) string {
	fraction := float64(read) / float64(total)
	if fraction > 1 {
		fraction = 1
	}
	if fraction < 0 {
		fraction = 0
	}

	filled := int(fraction * progressBarWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)

	return fmt.Sprintf("\r[*] Descargando... [%s] %5.1f%%  %s/%s  %s",
		bar, fraction*100, formatMB(read), formatMB(total), formatSpeed(read, elapsed))
}

func formatMB(bytes int64) string {
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}

func formatSpeed(bytesRead int64, elapsed time.Duration) string {
	seconds := elapsed.Seconds()
	if seconds <= 0 {
		return "-- KB/s"
	}

	bytesPerSec := float64(bytesRead) / seconds
	if bytesPerSec >= 1024*1024 {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
	}
	return fmt.Sprintf("%.0f KB/s", bytesPerSec/1024)
}
