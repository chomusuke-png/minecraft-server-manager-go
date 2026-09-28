package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"minecraft-manager/internal/approot"
	"minecraft-manager/internal/logx"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// worldDirs son las carpetas de dimensión estándar que genera un servidor
// vanilla/Paper/Fabric/Quilt/Forge/NeoForge/Arclight con la config por defecto (level-name=world).
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
	backupDir := filepath.Join(approot.Path("backups"), instanceName)
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

	if err := bm.writeBackup(zipPath, existingDirs); err != nil {
		// un zip a medio escribir no es un backup: si queda, la retencion lo
		// cuenta como valido y puede desplazar a uno que si sirve
		discardPartialBackup(zipPath)
		return err
	}

	logx.Success("Backup completado exitosamente.")
	return nil
}

// writeBackup escribe el zip y solo devuelve nil cuando el backup quedo
// completo y confirmado en disco
func (bm *BackupManager) writeBackup(zipPath string, existingDirs []string) error {
	file, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo zip: %w", err)
	}

	zipWriter := zip.NewWriter(file)

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

			// el formato zip siempre usa '/', sin importar el SO. filepath.Rel
			// devuelve '\' en Windows, y sin este ToSlash el zip queda con
			// entradas no estándar
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
			zipWriter.Close()
			file.Close()
			return fmt.Errorf("error comprimiendo '%s': %w", dir, err)
		}
	}

	// el zip recien es valido cuando Close escribe el directorio central, asi
	// que ese error no se puede descartar: hasta que no vuelve bien, lo que hay
	// en disco es un archivo truncado
	if err := zipWriter.Close(); err != nil {
		file.Close()
		return fmt.Errorf("no se pudo finalizar el zip: %w", err)
	}

	// sin Sync el zip puede estar solo en el cache del sistema operativo, y a un
	// backup le importa sobrevivir un corte de luz
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("no se pudo escribir el backup a disco: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("no se pudo cerrar el backup: %w", err)
	}
	return nil
}

// discardPartialBackup borra el zip que quedo de un backup fallido
func discardPartialBackup(zipPath string) {
	if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
		logx.Warn("Quedó un backup incompleto en '%s' y no se pudo borrar: %v", zipPath, err)
	}
}

// cleanOldBackups borra los backups más viejos que retentionDays, pero nunca
// deja menos de keepMin backups en total (los más recientes primero)
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
