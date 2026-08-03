// Package httpx concentra las descargas HTTP con barra de progreso y el consumo
// de APIs JSON. Vive aparte de downloader para que otros paquetes (java) puedan
// descargar sin importarlo y generar un ciclo.
package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"minecraft-manager/internal/logx"
)

// Download baja url a destinationPath, creando el archivo. Sigue redirects.
func Download(url string, destinationPath string) error {
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
	defer outputFile.Close()

	progressReader := &ProgressReader{
		Reader: response.Body,
		Total:  response.ContentLength,
	}

	if _, err = io.Copy(outputFile, progressReader); err != nil {
		return err
	}

	logx.Info("\nDownload completed.")
	return nil
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
