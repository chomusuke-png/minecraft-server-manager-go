// Package httpx concentra las descargas HTTP con barra de progreso y el consumo
// de APIs JSON. Vive aparte de downloader para que otros paquetes (java) puedan
// descargar sin importarlo y generar un ciclo.
package httpx

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"minecraft-manager/internal/logx"
)

// Download baja url a destinationPath sin verificar integridad. Solo debería
// usarse cuando la fuente no publica ningún checksum (ver DownloadVerified).
func Download(url string, destinationPath string) error {
	return DownloadVerified(url, destinationPath, "", "")
}

// DownloadVerified baja url a destinationPath y, si expectedHex no está vacío,
// verifica el contenido contra ese hash (algo: "sha1" o "sha256") a medida que
// se escribe a disco, sin pasada extra de lectura.
//
// Si el hash no coincide, el archivo se borra y se devuelve error: preferimos
// un downloader roto a ejecutar en la máquina del usuario un binario que no es
// el que la fuente dice que es (instalador de Forge, JDK, server.jar, etc.).
func DownloadVerified(url, destinationPath, algo, expectedHex string) error {
	logx.Info("Downloading from: %s", url)

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status: %s", response.Status)
	}

	outputFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}

	hasher, err := newHasher(algo)
	if err != nil {
		outputFile.Close()
		return err
	}

	progressReader := &ProgressReader{
		Reader: response.Body,
		Total:  response.ContentLength,
	}

	var writer io.Writer = outputFile
	if hasher != nil {
		writer = io.MultiWriter(outputFile, hasher)
	}

	_, copyErr := io.Copy(writer, progressReader)
	closeErr := outputFile.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	logx.Info("\nDownload completed.")

	if hasher == nil || expectedHex == "" {
		return nil
	}

	got := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(got, expectedHex) {
		os.Remove(destinationPath)
		return fmt.Errorf(
			"checksum %s no coincide (se esperaba %s, se obtuvo %s): archivo descartado, puede estar corrupto o alterado",
			algo, expectedHex, got,
		)
	}

	logx.Detail("Checksum %s verificado.", algo)
	return nil
}

func newHasher(algo string) (hash.Hash, error) {
	switch algo {
	case "":
		return nil, nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("algoritmo de hash no soportado: %s", algo)
	}
}

func GetJSON(url string, target interface{}) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", response.StatusCode)
	}

	return json.NewDecoder(response.Body).Decode(target)
}

// GetXML es GetJSON pero para endpoints que responden XML, como el
// maven-metadata.xml que usa NeoForge para publicar sus versiones.
func GetXML(url string, target interface{}) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", response.StatusCode)
	}

	return xml.NewDecoder(response.Body).Decode(target)
}

// GetText trae el body de url como texto plano, para los sidecars de checksum
// (*.sha1) que publican los repositorios Maven junto a cada artefacto. Best
// effort: un error acá no debería frenar la descarga, solo dejarla sin
// verificar (ver uso en DownloadForge).
func GetText(url string) (string, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type ProgressReader struct {
	Reader io.Reader
	Total  int64
	read   int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.read += int64(n)

	if pr.Total > 0 {
		percent := float64(pr.read) / float64(pr.Total) * 100
		fmt.Printf("\r[*] Progress: %.1f%%", percent)
	}

	return n, err
}
