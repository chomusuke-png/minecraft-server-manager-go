package approot

import (
	"path/filepath"
	"testing"
)

func TestRootForUsaElDirectorioDelEjecutable(t *testing.T) {
	executableDir := filepath.Join(t.TempDir(), "msm")

	if got := rootFor(executableDir); got != executableDir {
		t.Errorf("got %q, want %q", got, executableDir)
	}
}

func TestRootForCaeAlCwdConBuildCache(t *testing.T) {
	executableDir := filepath.Join(t.TempDir(), "go-build123", "b001", "exe")

	if got := rootFor(executableDir); got != "." {
		t.Errorf("got %q, want %q", got, ".")
	}
}

func TestResolveDejaLasAbsolutasIntactas(t *testing.T) {
	absolute := filepath.Join(t.TempDir(), "playit")

	if got := Resolve(absolute); got != absolute {
		t.Errorf("got %q, want %q", got, absolute)
	}
	if got := Resolve(""); got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestPathAnclaAlDirectorioDeDatos(t *testing.T) {
	want := filepath.Join(Dir(), "instances", "mi_server")

	if got := Path("instances", "mi_server"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
