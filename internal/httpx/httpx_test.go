package httpx

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadWritesFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("contenido"))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "out.txt")
	if err := Download(server.URL, dest); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "contenido" {
		t.Errorf("got %q", got)
	}
}

func TestDownloadVerifiedAcceptsMatchingChecksum(t *testing.T) {
	body := []byte("contenido verificado")
	sum := sha256.Sum256(body)
	hexSum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := DownloadVerified(server.URL, dest, "sha256", hexSum); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Error("el archivo debería existir tras verificar OK")
	}
}

func TestDownloadVerifiedRejectsMismatchAndDeletesFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("contenido alterado"))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := DownloadVerified(server.URL, dest, "sha256", "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("se esperaba error de checksum")
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("el archivo debería haberse borrado tras el mismatch")
	}
}

func TestDownloadVerifiedRejectsUnsupportedAlgo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("x"))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := DownloadVerified(server.URL, dest, "md5", "deadbeef"); err == nil {
		t.Error("se esperaba error por algoritmo no soportado")
	}
}

func TestDownloadVerifiedFailsOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := Download(server.URL, dest); err == nil {
		t.Error("se esperaba error con status 404")
	}
}

func TestGetJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"quilt","count":3}`))
	}))
	defer server.Close()

	var target struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	if err := GetJSON(server.URL, &target); err != nil {
		t.Fatal(err)
	}
	if target.Name != "quilt" || target.Count != 3 {
		t.Errorf("got %+v", target)
	}
}

func TestGetJSONFailsOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var target map[string]any
	if err := GetJSON(server.URL, &target); err == nil {
		t.Error("se esperaba error con status 500")
	}
}

func TestGetXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<metadata><versioning><versions><version>1.0</version><version>2.0</version></versions></versioning></metadata>`))
	}))
	defer server.Close()

	var target struct {
		Versioning struct {
			Versions struct {
				Version []string `xml:"version"`
			} `xml:"versions"`
		} `xml:"versioning"`
	}
	if err := GetXML(server.URL, &target); err != nil {
		t.Fatal(err)
	}
	if len(target.Versioning.Versions.Version) != 2 {
		t.Errorf("got %+v", target)
	}
}

func TestGetText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("deadbeef  archivo.jar\n"))
	}))
	defer server.Close()

	got, err := GetText(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "deadbeef  archivo.jar\n" {
		t.Errorf("got %q", got)
	}
}

func TestGetTextFailsOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := GetText(server.URL); err == nil {
		t.Error("se esperaba error con status 404")
	}
}
