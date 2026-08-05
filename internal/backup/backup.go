package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"minecraft-manager/internal/logx"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// worldDirs son las carpetas de dimensión estándar que genera un servidor
// vanilla/Paper/Fabric/Forge con la config por defecto (level-name=world).
// Solo se incluyen en el backup las que realmente existan.
var worldDirs = []string{"world", "world_nether", "world_the_end"}

type BackupManager struct {
	serverDir     string
	backupDir     string
	instanceName  string
	retentionDays int
	keepMin       int
}

// New arma el manager de backups de una instancia. keepMin es el piso de
// backups que nunca se borra, sin importar cuántos días de retención hayan
// pasado.
func New(serverDir, instanceName string, retentionDays, keepMin int) *BackupManager {
	backupDir := filepath.Join("backups", instanceName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logx.Error("No se pudo crear carpeta de backups '%s': %v", backupDir, err)
	}

	return &BackupManager{
		serverDir:     serverDir,
		backupDir:     backupDir,
		instanceName:  instanceName,
		retentionDays: retentionDays,
		keepMin:       keepMin,
	}
}

func (bm *BackupManager) CreateBackup() error {
	var existingDirs []string
	for _, dir := range worldDirs {
		if info, err := os.Stat(filepath.Join(bm.serverDir, dir)); err == nil && info.IsDir() {
			existingDirs = append(existingDirs, dir)
		}
	}
	if len(existingDirs) == 0 {
		return nil
	}

	bm.cleanOldBackups()

	timestamp := time.Now().Format("20060102_150405")
	zipName := timestamp + ".zip"
	zipPath := filepath.Join(bm.backupDir, zipName)

	logx.Info("Creando backup de '%s': %s...", bm.instanceName, zipName)

	file, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo zip: %w", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for _, dir := range existingDirs {
		worldPath := filepath.Join(bm.serverDir, dir)

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

			// El formato zip siempre usa '/', sin importar el SO. filepath.Rel
			// devuelve '\' en Windows, y sin este ToSlash el zip queda con
			// entradas no estándar.
			zipEntry, err := zipWriter.Create(filepath.ToSlash(relPath))
			if err != nil {
				return err
			}

			sourceFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer sourceFile.Close()

			_, err = io.Copy(zipEntry, sourceFile)
			return err
		})

		if err != nil {
			return fmt.Errorf("error comprimiendo '%s': %w", dir, err)
		}
	}

	logx.Success("Backup completado exitosamente.")
	return nil
}

// cleanOldBackups borra los backups más viejos que retentionDays, pero nunca
// deja menos de keepMin backups en total (los más recientes primero).
func (bm *BackupManager) cleanOldBackups() {
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		logx.Error("Error leyendo carpeta backups: %v", err)
		return
	}

	type backupFile struct {
		name    string
		modTime time.Time
	}

	var zips []backupFile
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".zip" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		zips = append(zips, backupFile{name: entry.Name(), modTime: info.ModTime()})
	}

	sort.Slice(zips, func(i, j int) bool { return zips[i].modTime.After(zips[j].modTime) })

	retentionDuration := time.Duration(bm.retentionDays) * 24 * time.Hour
	cutoff := time.Now().Add(-retentionDuration)

	logx.Info("Verificando backups antiguos de '%s'...", bm.instanceName)

	count := 0
	for i, zip := range zips {
		if i < bm.keepMin {
			continue
		}
		if !zip.modTime.Before(cutoff) {
			continue
		}

		backupFilePath := filepath.Join(bm.backupDir, zip.name)
		if err := os.Remove(backupFilePath); err == nil {
			logx.Detail("Eliminado: %s", zip.name)
			count++
		} else {
			logx.Error("Error borrando %s: %v", zip.name, err)
		}
	}

	if count == 0 {
		logx.Detail("Ningún backup expirado.")
	}
}
