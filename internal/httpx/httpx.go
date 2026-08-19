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

func Download(url string, destinationPath string) error {
	return DownloadVerified(url, destinationPath, "", "")
}

func DownloadVerified(url, destinationPath, algo, expectedHex string) error {
	logx.Info("Descargando desde: %s", url)

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("el servidor respondió con estado %s", response.Status)
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

	logx.Success("\nDescarga completada.")

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
		return fmt.Errorf("la API respondió con estado %d", response.StatusCode)
	}

	return json.NewDecoder(response.Body).Decode(target)
}

func GetXML(url string, target interface{}) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("la API respondió con estado %d", response.StatusCode)
	}

	return xml.NewDecoder(response.Body).Decode(target)
}

func GetText(url string) (string, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("la petición falló con estado %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
