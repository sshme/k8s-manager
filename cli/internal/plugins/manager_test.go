package plugins

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"k8s-manager/cli/internal/market"
)

// fixedDownloader всегда отдаёт один и тот же payload
type fixedDownloader struct {
	payload []byte
	calls   int
}

func (f *fixedDownloader) DownloadArtifact(_ context.Context, _ int64, w io.Writer) (int64, error) {
	f.calls++
	n, err := w.Write(f.payload)
	return int64(n), err
}

// buildFakeZip собирает простой zip с одним файлом и возвращает его
func buildFakeZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("plugin.json")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write([]byte(`{"name":"fake"}`)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestManagerInstallEndToEnd(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfg := Config{Root: root}

	payload := buildFakeZip(t)
	sum := sha256.Sum256(payload)
	expectedChecksum := hex.EncodeToString(sum[:])

	dl := &fixedDownloader{payload: payload}
	mgr, err := newManager(cfg, dl)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	ref := InstallRef{
		PluginID:         42,
		PluginIdentifier: "acme.fake",
		PluginName:       "Fake Plugin",
		PublisherID:      1,
		PublisherName:    "Acme",
		ReleaseID:        7,
		Version:          "0.1.0",
		Artifact: market.Artifact{
			ID:       100,
			OS:       "linux",
			Arch:     "amd64",
			Type:     "zip",
			Size:     int64(len(payload)),
			Checksum: expectedChecksum,
		},
	}

	installed, err := mgr.Install(context.Background(), ref)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Файлы на месте
	if _, err := os.Stat(zipPath(installed.InstallDir)); err != nil {
		t.Errorf("artifact.zip missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(contentDir(installed.InstallDir), "plugin.json")); err != nil {
		t.Errorf("extracted content missing: %v", err)
	}
	if _, err := os.Stat(installMetaPath(installed.InstallDir)); err != nil {
		t.Errorf("install.json missing: %v", err)
	}

	// Реестр содержит запись
	if !mgr.IsInstalled(ref.PluginID, ref.Version, ref.Artifact.OS, ref.Artifact.Arch) {
		t.Errorf("expected IsInstalled true after Install")
	}

	// Структура пути соответствует спецификации
	wantSegment := filepath.Join("plugins", "store", "Acme", "acme.fake", "0.1.0", "linux-amd64")
	if !filepath.IsAbs(installed.InstallDir) || filepath.Base(installed.InstallDir) != "linux-amd64" {
		t.Errorf("install dir shape unexpected: %s", installed.InstallDir)
	}
	if !contains(installed.InstallDir, wantSegment) {
		t.Errorf("install dir %q should contain %q", installed.InstallDir, wantSegment)
	}

	// Идемпотентность, повторный Install не делает повторного запроса.
	prevCalls := dl.calls
	if _, err := mgr.Install(context.Background(), ref); err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if dl.calls != prevCalls {
		t.Errorf("second Install made extra download: %d -> %d", prevCalls, dl.calls)
	}
}

func TestManagerInstallChecksumMismatchKeepsRegistryClean(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfg := Config{Root: root}

	payload := buildFakeZip(t)
	dl := &fixedDownloader{payload: payload}
	mgr, err := newManager(cfg, dl)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	ref := InstallRef{
		PluginID:         42,
		PluginIdentifier: "acme.fake",
		Version:          "0.1.0",
		Artifact: market.Artifact{
			ID:       100,
			OS:       "linux",
			Arch:     "amd64",
			Type:     "zip",
			Size:     int64(len(payload)),
			Checksum: "abcdef", // не совпадает
		},
	}

	if _, err := mgr.Install(context.Background(), ref); err == nil {
		t.Fatalf("expected checksum error, got nil")
	}
	if mgr.IsInstalled(ref.PluginID, ref.Version, ref.Artifact.OS, ref.Artifact.Arch) {
		t.Errorf("registry should not record failed install")
	}
	// partial файл должен быть подчищен
	dir := storeDir(cfg, "", "acme.fake", "0.1.0", "linux", "amd64")
	if _, err := os.Stat(partialZipPath(dir)); err == nil {
		t.Errorf("partial file should be removed on failure")
	}
}

func TestManagerUninstallRemovesFilesAndEntry(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfg := Config{Root: root}

	payload := buildFakeZip(t)
	sum := sha256.Sum256(payload)
	expected := hex.EncodeToString(sum[:])
	dl := &fixedDownloader{payload: payload}
	mgr, err := newManager(cfg, dl)
	if err != nil {
		t.Fatalf("newManager: %v", err)
	}

	ref := InstallRef{
		PluginID:         77,
		PluginIdentifier: "acme.uninst",
		Version:          "0.1.0",
		PublisherName:    "Acme",
		Artifact:         market.Artifact{ID: 1, OS: "linux", Arch: "amd64", Size: int64(len(payload)), Checksum: expected},
	}
	installed, err := mgr.Install(context.Background(), ref)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := mgr.Uninstall(ref.PluginID, ref.Version, ref.Artifact.OS, ref.Artifact.Arch); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if mgr.IsInstalled(ref.PluginID, ref.Version, ref.Artifact.OS, ref.Artifact.Arch) {
		t.Errorf("IsInstalled should be false after Uninstall")
	}
	if _, err := os.Stat(installed.InstallDir); err == nil {
		t.Errorf("install dir should be removed")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
