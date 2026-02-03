package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type BackupManager struct {
	serverDir     string
	backupDir     string
	retentionDays int
}

func New(serverDir string, retentionDays int) *BackupManager {
	backupDir := "backups"
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.Mkdir(backupDir, 0755)
	}

	return &BackupManager{
		serverDir:     serverDir,
		backupDir:     backupDir,
		retentionDays: retentionDays,
	}
}

func (bm *BackupManager) CreateBackup() error {
	worldPath := filepath.Join(bm.serverDir, "world")

	if _, err := os.Stat(worldPath); os.IsNotExist(err) {
		return nil
	}

	bm.cleanOldBackups()

	timestamp := time.Now().Format("20060102_150405")
	zipName := fmt.Sprintf("world_backup_%s.zip", timestamp)
	zipPath := filepath.Join(bm.backupDir, zipName)

	fmt.Printf("[*] Creando backup: %s...\n", zipName)

	file, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo zip: %w", err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	err = filepath.Walk(worldPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(bm.serverDir, path)
		if err != nil {
			return err
		}

		zipFile, err := w.Create(relPath)
		if err != nil {
			return err
		}

		fsFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fsFile.Close()

		_, err = io.Copy(zipFile, fsFile)
		return err
	})

	if err != nil {
		return fmt.Errorf("error comprimiendo archivos: %w", err)
	}

	fmt.Println("[+] Backup completado exitosamente.")
	return nil
}

func (bm *BackupManager) cleanOldBackups() {
	files, err := os.ReadDir(bm.backupDir)
	if err != nil {
		fmt.Printf("[-] Error leyendo carpeta backups: %v\n", err)
		return
	}

	retentionDuration := time.Duration(bm.retentionDays) * 24 * time.Hour
	cutoff := time.Now().Add(-retentionDuration)

	fmt.Println("[*] Verificando backups antiguos...")

	count := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".zip" {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			fullPath := filepath.Join(bm.backupDir, file.Name())
			if err := os.Remove(fullPath); err == nil {
				fmt.Printf("    -> Eliminado: %s\n", file.Name())
				count++
			} else {
				fmt.Printf("    [-] Error borrando %s: %v\n", file.Name(), err)
			}
		}
	}

	if count == 0 {
		fmt.Println("    -> Ningún backup expirado.")
	}
}
