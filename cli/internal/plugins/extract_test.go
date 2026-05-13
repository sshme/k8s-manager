package plugins

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type zipEntry struct {
	body []byte
	mode os.FileMode
}

// makeZip собирает zip в памяти из мапы и пишет на диск
func makeZip(t *testing.T, path string, entries map[string]zipEntry) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, entry := range entries {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		header.SetMode(entry.mode)
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("create zip header: %v", err)
		}
		if _, err := w.Write(entry.body); err != nil {
			t.Fatalf("write zip body: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}
}

func TestExtractZipHappyPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "a.zip")
	dest := filepath.Join(dir, "out")

	makeZip(t, zipPath, map[string]zipEntry{
		"plugin.json":     {body: []byte(`{"name":"p"}`), mode: 0o644},
		"bin/prometheus":  {body: []byte{0x7f, 'E', 'L', 'F'}, mode: 0o755},
		"docs/readme.txt": {body: []byte("hi"), mode: 0o644},
	})

	if err := extractZip(zipPath, dest); err != nil {
		t.Fatalf("extractZip: %v", err)
	}

	if got, err := os.ReadFile(filepath.Join(dest, "plugin.json")); err != nil || !bytes.Equal(got, []byte(`{"name":"p"}`)) {
		t.Errorf("plugin.json content mismatch err=%v got=%s", err, got)
	}

	// На windows бит x в perm бессмыслен, проверяем только на unix
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dest, "bin/prometheus"))
		if err != nil {
			t.Fatalf("stat bin: %v", err)
		}
		if info.Mode()&0o100 == 0 {
			t.Errorf("expected owner-executable bit, got mode %v", info.Mode())
		}
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	dest := filepath.Join(dir, "out")

	makeZip(t, zipPath, map[string]zipEntry{
		"../escape.txt": {body: []byte("nope"), mode: 0o644},
	})

	err := extractZip(zipPath, dest)
	if err == nil {
		t.Fatalf("expected zip slip error, got nil")
	}
	if !strings.Contains(err.Error(), "illegal zip entry") {
		t.Errorf("error should mention illegal entry, got %v", err)
	}
	// И файл наружу записан не был
	if _, err := os.Stat(filepath.Join(dir, "escape.txt")); err == nil {
		t.Errorf("escape file was created despite check")
	}
}

func TestExtractZipOverwritesExistingFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "a.zip")
	dest := filepath.Join(dir, "out")

	makeZip(t, zipPath, map[string]zipEntry{
		"data.txt": {body: []byte("new"), mode: 0o644},
	})

	// Эмулируем повторную установку: файл уже лежит со старым содержимым
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "data.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write old: %v", err)
	}

	if err := extractZip(zipPath, dest); err != nil {
		t.Fatalf("extractZip: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "data.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("expected overwrite to 'new', got %q", got)
	}
}
