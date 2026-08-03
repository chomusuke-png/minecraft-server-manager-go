//go:build manual

package java

import "testing"

// Descarga real de un JDK. Correr con:
//
//	go test -tags manual ./internal/java/ -run TestManualDownloadJDK -v -count=1 -timeout 20m
func TestManualDownloadJDK(t *testing.T) {
	// go test corre con el CWD en el dir del paquete; el runtime tiene que caer en
	// el runtimes/ de la raíz del proyecto, que es donde lo busca la app.
	t.Chdir("../..")

	javaPath, err := Download(17)
	if err != nil {
		t.Fatalf("Download(17): %v", err)
	}
	t.Logf("java en: %s", javaPath)

	major, err := DetectMajor(javaPath)
	if err != nil {
		t.Fatalf("DetectMajor: %v", err)
	}
	t.Logf("major detectado: %d", major)

	req := Require("1.19.1")
	t.Logf("1.19.1 requiere: %s", req)
	if !req.Satisfies(major) {
		t.Errorf("el JDK descargado no cumple el requisito de 1.19.1")
	}

	if found := findInRuntimes(req); found == "" {
		t.Error("findInRuntimes no encontró el JDK recién descargado")
	} else {
		t.Logf("findInRuntimes lo redescubre en: %s", found)
	}
}
