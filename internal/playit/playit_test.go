package playit

import (
	"os"
	"testing"
)

func TestAddPID(t *testing.T) {
	pids := addPID([]int{1, 2}, 3)
	if len(pids) != 3 || pids[2] != 3 {
		t.Errorf("got %v", pids)
	}
}

func TestAddPIDNoDuplicate(t *testing.T) {
	pids := addPID([]int{1, 2}, 2)
	if len(pids) != 2 {
		t.Errorf("got %v, no debería duplicar", pids)
	}
}

func TestRemovePID(t *testing.T) {
	pids := removePID([]int{1, 2, 3}, 2)
	if len(pids) != 2 || pids[0] != 1 || pids[1] != 3 {
		t.Errorf("got %v", pids)
	}
}

func TestRemovePIDNotPresent(t *testing.T) {
	pids := removePID([]int{1, 2, 3}, 99)
	if len(pids) != 3 {
		t.Errorf("got %v", pids)
	}
}

// un PID gigante y poco probable no deberia estar vivo en ningun SO
const definitelyDeadPID = 999999999

func TestPruneDeadRemovesDeadClients(t *testing.T) {
	reg := registry{
		PlayitPID: os.Getpid(),
		Clients:   []int{os.Getpid(), definitelyDeadPID},
	}
	pruneDead(&reg)

	if len(reg.Clients) != 1 || reg.Clients[0] != os.Getpid() {
		t.Errorf("got %v", reg.Clients)
	}
	if reg.PlayitPID != os.Getpid() {
		t.Errorf("PlayitPID vivo no debería tocarse, got %d", reg.PlayitPID)
	}
}

func TestPruneDeadClearsDeadPlayitPID(t *testing.T) {
	reg := registry{PlayitPID: definitelyDeadPID}
	pruneDead(&reg)

	if reg.PlayitPID != 0 {
		t.Errorf("got %d, want 0", reg.PlayitPID)
	}
}

func TestLoadRegistryMissingFileReturnsEmpty(t *testing.T) {
	t.Chdir(t.TempDir())

	reg := loadRegistry()
	if reg.PlayitPID != 0 || len(reg.Clients) != 0 {
		t.Errorf("got %+v", reg)
	}
}

func TestLoadRegistryCorruptFileReturnsEmpty(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(registryPath, []byte("esto no es json"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := loadRegistry()
	if reg.PlayitPID != 0 || len(reg.Clients) != 0 {
		t.Errorf("got %+v", reg)
	}
}

func TestSaveAndLoadRegistryRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	original := registry{PlayitPID: 1234, Clients: []int{1, 2, 3}}
	if err := saveRegistry(original); err != nil {
		t.Fatal(err)
	}

	got := loadRegistry()
	if got.PlayitPID != original.PlayitPID || len(got.Clients) != len(original.Clients) {
		t.Errorf("got %+v, want %+v", got, original)
	}
}

func TestLockPreventsSecondAcquireUntilUnlocked(t *testing.T) {
	t.Chdir(t.TempDir())

	unlock, err := lock()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(lockPath); err != nil {
		t.Error("el archivo de lock debería existir mientras está tomado")
	}

	unlock()

	if _, err := os.Stat(lockPath); err == nil {
		t.Error("el archivo de lock debería desaparecer al liberar")
	}

	unlock2, err := lock()
	if err != nil {
		t.Fatal(err)
	}
	unlock2()
}
