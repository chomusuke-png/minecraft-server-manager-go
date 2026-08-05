package backup

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// chdirTemp mueve el cwd a un directorio temporal — New() y cleanOldBackups()
// trabajan con "backups/" relativo al cwd, igual que el resto de la app — y
// lo restaura al terminar el test.
func chdirTemp(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(original) })
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// makeFakeBackup crea un .zip vacío en dir con nombre name y le fuerza el
// mtime a "hace age", para poder probar la limpieza por retención sin
// depender de crear backups reales ni de dormir entre ellos.
func makeFakeBackup(t *testing.T, dir, name string, age time.Duration) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	mtime := time.Now().Add(-age)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func zipNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".zip" {
			names = append(names, e.Name())
		}
	}
	return names
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func singleZip(t *testing.T, dir string) string {
	t.Helper()
	names := zipNames(t, dir)
	if len(names) != 1 {
		t.Fatalf("se esperaba 1 zip en %s, hay %d: %v", dir, len(names), names)
	}
	return filepath.Join(dir, names[0])
}

func zipEntryNames(t *testing.T, zipPath string) map[string]bool {
	t.Helper()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	return names
}

func readZipEntry(t *testing.T, zipPath, entryName string) string {
	t.Helper()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name != entryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	t.Fatalf("entrada '%s' no encontrada en %s", entryName, zipPath)
	return ""
}

func TestNewCreatesBackupDir(t *testing.T) {
	chdirTemp(t)

	bm := New("instances/foo", "foo", 7, 3)

	if _, err := os.Stat(bm.backupDir); err != nil {
		t.Fatalf("no se creó el directorio de backups: %v", err)
	}
	want := filepath.Join("backups", "foo")
	if bm.backupDir != want {
		t.Errorf("backupDir = %q, quería %q", bm.backupDir, want)
	}
}

func TestCreateBackupNoopWithoutWorldDir(t *testing.T) {
	chdirTemp(t)
	serverDir := "instances/foo"
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatal(err)
	}

	bm := New(serverDir, "foo", 7, 3)
	if err := bm.CreateBackup(); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	if names := zipNames(t, bm.backupDir); len(names) != 0 {
		t.Errorf("se creó un backup sin haber mundo: %v", names)
	}
}

func TestCreateBackupReturnsErrorWhenBackupDirUnavailable(t *testing.T) {
	chdirTemp(t)
	serverDir := "instances/foo"
	writeFile(t, filepath.Join(serverDir, "world", "level.dat"), "x")

	// "backups" ya existe como archivo (no directorio): el MkdirAll de New()
	// falla, y CreateBackup no debería paniquear por eso, solo devolver error.
	if err := os.WriteFile("backups", []byte("no es un directorio"), 0644); err != nil {
		t.Fatal(err)
	}

	bm := New(serverDir, "foo", 7, 3)
	if err := bm.CreateBackup(); err == nil {
		t.Error("se esperaba error cuando no se puede crear el directorio de backups")
	}
}

func TestCreateBackupPreservesFileContent(t *testing.T) {
	chdirTemp(t)
	serverDir := "instances/foo"
	writeFile(t, filepath.Join(serverDir, "world", "level.dat"), "datos del mundo")

	bm := New(serverDir, "foo", 7, 3)
	if err := bm.CreateBackup(); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	zipPath := singleZip(t, bm.backupDir)
	if got := readZipEntry(t, zipPath, "world/level.dat"); got != "datos del mundo" {
		t.Errorf("contenido = %q, quería %q", got, "datos del mundo")
	}
}

func TestCreateBackupIncludesAllExistingDimensions(t *testing.T) {
	chdirTemp(t)
	serverDir := "instances/foo"
	writeFile(t, filepath.Join(serverDir, "world", "level.dat"), "overworld")
	writeFile(t, filepath.Join(serverDir, "world_nether", "level.dat"), "nether")
	// world_the_end no existe: no debe fallar ni aparecer en el zip.

	bm := New(serverDir, "foo", 7, 3)
	if err := bm.CreateBackup(); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	names := zipEntryNames(t, singleZip(t, bm.backupDir))
	for _, want := range []string{"world/level.dat", "world_nether/level.dat"} {
		if !names[want] {
			t.Errorf("falta %s en el zip: %v", want, names)
		}
	}
	if names["world_the_end/level.dat"] {
		t.Error("world_the_end no debería estar: no existía en el server dir")
	}
}

func TestCreateBackupWalksNestedDirectories(t *testing.T) {
	chdirTemp(t)
	serverDir := "instances/foo"
	writeFile(t, filepath.Join(serverDir, "world", "level.dat"), "level")
	writeFile(t, filepath.Join(serverDir, "world", "region", "r.0.0.mca"), "chunk")
	writeFile(t, filepath.Join(serverDir, "world", "playerdata", "uuid.dat"), "player")

	bm := New(serverDir, "foo", 7, 3)
	if err := bm.CreateBackup(); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	names := zipEntryNames(t, singleZip(t, bm.backupDir))
	for _, want := range []string{"world/level.dat", "world/region/r.0.0.mca", "world/playerdata/uuid.dat"} {
		if !names[want] {
			t.Errorf("falta %s en el zip. Entradas: %v", want, names)
		}
	}
}

// El backup es de las dimensiones del mundo, no de la instancia entera: si se
// colara todo el server dir, un config.json con datos personales terminaría
// adentro del zip.
func TestCreateBackupOnlyIncludesWorldDirs(t *testing.T) {
	chdirTemp(t)
	serverDir := "instances/foo"
	writeFile(t, filepath.Join(serverDir, "world", "level.dat"), "level")
	writeFile(t, filepath.Join(serverDir, "server.jar"), "no es del mundo")
	writeFile(t, filepath.Join(serverDir, "config.json"), "tampoco")

	bm := New(serverDir, "foo", 7, 3)
	if err := bm.CreateBackup(); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	names := zipEntryNames(t, singleZip(t, bm.backupDir))
	if len(names) != 1 || !names["world/level.dat"] {
		t.Errorf("el backup incluyó algo fuera de las carpetas de dimensión: %v", names)
	}
}

func TestCleanOldBackupsRespectsKeepMinFloor(t *testing.T) {
	chdirTemp(t)
	bm := New("instances/foo", "foo", 1, 3) // retención de 1 día, piso de 3

	// 5 backups, todos muy por encima del día de retención.
	makeFakeBackup(t, bm.backupDir, "1.zip", 100*24*time.Hour)
	makeFakeBackup(t, bm.backupDir, "2.zip", 90*24*time.Hour)
	makeFakeBackup(t, bm.backupDir, "3.zip", 80*24*time.Hour)
	makeFakeBackup(t, bm.backupDir, "4.zip", 70*24*time.Hour)
	makeFakeBackup(t, bm.backupDir, "5.zip", 60*24*time.Hour)

	bm.cleanOldBackups()

	remaining := zipNames(t, bm.backupDir)
	if len(remaining) != 3 {
		t.Fatalf("quedaron %d backups, se esperaban 3 (el piso keepMin): %v", len(remaining), remaining)
	}
	// Deben sobrevivir los 3 más recientes, pese a estar igual de "vencidos".
	for _, want := range []string{"3.zip", "4.zip", "5.zip"} {
		if !contains(remaining, want) {
			t.Errorf("se esperaba que %s sobreviviera por el piso mínimo, quedaron: %v", want, remaining)
		}
	}
}

func TestCleanOldBackupsDeletesExpiredBeyondFloor(t *testing.T) {
	chdirTemp(t)
	bm := New("instances/foo", "foo", 7, 1) // retención 7 días, piso 1

	makeFakeBackup(t, bm.backupDir, "old.zip", 10*24*time.Hour) // vencido y fuera del piso
	makeFakeBackup(t, bm.backupDir, "new.zip", time.Hour)       // dentro de la retención

	bm.cleanOldBackups()

	remaining := zipNames(t, bm.backupDir)
	if contains(remaining, "old.zip") {
		t.Error("old.zip debería haberse borrado: vencido y fuera del piso mínimo")
	}
	if !contains(remaining, "new.zip") {
		t.Error("new.zip no debería haberse borrado: está dentro de la retención")
	}
}

func TestCleanOldBackupsKeepsRecentEvenBeyondFloor(t *testing.T) {
	chdirTemp(t)
	bm := New("instances/foo", "foo", 7, 1) // piso de 1, pero ninguno está vencido

	makeFakeBackup(t, bm.backupDir, "a.zip", time.Hour)
	makeFakeBackup(t, bm.backupDir, "b.zip", 2*time.Hour)
	makeFakeBackup(t, bm.backupDir, "c.zip", 3*time.Hour)

	bm.cleanOldBackups()

	if remaining := zipNames(t, bm.backupDir); len(remaining) != 3 {
		t.Errorf("se borraron backups todavía dentro de la retención: quedaron %v", remaining)
	}
}

func TestCleanOldBackupsIgnoresNonZipFiles(t *testing.T) {
	chdirTemp(t)
	bm := New("instances/foo", "foo", 1, 0)

	makeFakeBackup(t, bm.backupDir, "old.zip", 100*24*time.Hour)

	notesPath := filepath.Join(bm.backupDir, "notes.txt")
	writeFile(t, notesPath, "no tocar")
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	if err := os.Chtimes(notesPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	bm.cleanOldBackups()

	if _, err := os.Stat(notesPath); err != nil {
		t.Error("un archivo no-zip no debería tocarse por la limpieza de backups")
	}
}

func TestCreateBackupCleansExpiredBackupsBeforeCreatingNewOne(t *testing.T) {
	chdirTemp(t)
	serverDir := "instances/foo"
	writeFile(t, filepath.Join(serverDir, "world", "level.dat"), "level")

	bm := New(serverDir, "foo", 1, 0) // retención 1 día, sin piso mínimo
	makeFakeBackup(t, bm.backupDir, "viejo.zip", 10*24*time.Hour)

	if err := bm.CreateBackup(); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	remaining := zipNames(t, bm.backupDir)
	if contains(remaining, "viejo.zip") {
		t.Error("el backup expirado debería haberse limpiado al crear uno nuevo")
	}
	if len(remaining) != 1 {
		t.Errorf("se esperaba solo el backup nuevo, quedaron: %v", remaining)
	}
}
