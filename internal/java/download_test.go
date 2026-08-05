package java

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func buildTarGz(t *testing.T, path string, entries func(tw *tar.Writer)) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	entries(tw)
}

func TestUntarGzExtractsFileAndSymlinkWithMode(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "jdk.tar.gz")
	destination := filepath.Join(dir, "out")

	buildTarGz(t, archivePath, func(tw *tar.Writer) {
		content := []byte("#!/bin/sh\necho fake java\n")
		hdr := &tar.Header{
			Name: "jdk-17.0.9+9/bin/java",
			Mode: 0755,
			Size: int64(len(content)),
		}
		tw.WriteHeader(hdr)
		tw.Write(content)

		// symlink de compatibilidad, como trae un JDK real.
		linkHdr := &tar.Header{
			Name:     "jdk-17.0.9+9/jre/bin/java",
			Typeflag: tar.TypeSymlink,
			Linkname: "../../bin/java",
		}
		tw.WriteHeader(linkHdr)
	})

	if err := untarGz(archivePath, destination); err != nil {
		t.Fatalf("untarGz: %v", err)
	}

	binPath := filepath.Join(destination, "jdk-17.0.9+9", "bin", "java")
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("no se extrajo %s: %v", binPath, err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("bin/java no quedó ejecutable, mode=%v", info.Mode())
	}
	t.Logf("bin/java mode: %v", info.Mode())

	linkPath := filepath.Join(destination, "jdk-17.0.9+9", "jre", "bin", "java")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("symlink no se creó: %v", err)
	}
	t.Logf("symlink -> %s", target)

	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatalf("symlink roto: %v", err)
	}
	absBin, _ := filepath.Abs(binPath)
	if resolved != absBin {
		t.Errorf("symlink resuelve a %s, esperaba %s", resolved, absBin)
	}
}

func TestUntarGzRejectsTarSlip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "evil.tar.gz")
	destination := filepath.Join(dir, "out")

	buildTarGz(t, archivePath, func(tw *tar.Writer) {
		content := []byte("pwned")
		hdr := &tar.Header{
			Name: "../../evil.txt",
			Mode: 0644,
			Size: int64(len(content)),
		}
		tw.WriteHeader(hdr)
		tw.Write(content)
	})

	err := untarGz(archivePath, destination)
	if err == nil {
		t.Fatal("se esperaba error por tar-slip, no pasó nada")
	}
	t.Logf("rechazado correctamente: %v", err)

	if _, statErr := os.Stat(filepath.Join(dir, "evil.txt")); statErr == nil {
		t.Fatal("el archivo malicioso se escribió fuera del destino")
	}
}

func TestUntarGzRejectsSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "evil-link.tar.gz")
	destination := filepath.Join(dir, "out")

	buildTarGz(t, archivePath, func(tw *tar.Writer) {
		hdr := &tar.Header{
			Name:     "escape",
			Typeflag: tar.TypeSymlink,
			Linkname: "../../../etc/passwd",
		}
		tw.WriteHeader(hdr)
	})

	err := untarGz(archivePath, destination)
	if err == nil {
		t.Fatal("se esperaba error por symlink fuera del destino, no pasó nada")
	}
	t.Logf("rechazado correctamente: %v", err)
}
