package ngrok

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestExtractFirstFileFromZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "ngrok.zip")

	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)
	entry, err := zw.Create("ngrok.exe")
	if err != nil {
		t.Fatal(err)
	}
	entry.Write([]byte("contenido del binario"))
	zw.Close()
	zf.Close()

	destPath := filepath.Join(dir, "out.exe")
	if err := extractFirstFileFromZip(zipPath, destPath); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "contenido del binario" {
		t.Errorf("got %q", content)
	}
}

func TestExtractFirstFileFromTarGz(t *testing.T) {
	dir := t.TempDir()
	tgzPath := filepath.Join(dir, "ngrok.tgz")

	f, err := os.Create(tgzPath)
	if err != nil {
		t.Fatal(err)
	}
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	data := []byte("contenido del binario")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "ngrok",
		Typeflag: tar.TypeReg,
		Size:     int64(len(data)),
		Mode:     0755,
	}); err != nil {
		t.Fatal(err)
	}
	tw.Write(data)
	tw.Close()
	gzw.Close()
	f.Close()

	destPath := filepath.Join(dir, "out")
	if err := extractFirstFileFromTarGz(tgzPath, destPath); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "contenido del binario" {
		t.Errorf("got %q", content)
	}
}

func TestFreePort(t *testing.T) {
	port, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("puerto inválido: %d", port)
	}
}

func TestWaitForPublicURLFindsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ngrokEndpointsResponse{
			Endpoints: []struct {
				URL string `json:"url"`
			}{{URL: "tcp://0.tcp.ngrok.io:12345"}},
		})
	}))
	defer server.Close()

	webPort := serverPort(t, server)
	url, err := waitForPublicURL(webPort, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if url != "tcp://0.tcp.ngrok.io:12345" {
		t.Errorf("got %q", url)
	}
}

func TestWaitForPublicURLTimesOutWithoutEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ngrokEndpointsResponse{})
	}))
	defer server.Close()

	webPort := serverPort(t, server)
	if _, err := waitForPublicURL(webPort, 1*time.Second); err == nil {
		t.Error("se esperaba error por timeout")
	}
}

func TestTunnelStopNilSafe(t *testing.T) {
	var tunnel *Tunnel
	tunnel.Stop()
}

func serverPort(t *testing.T, server *httptest.Server) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	return port
}
