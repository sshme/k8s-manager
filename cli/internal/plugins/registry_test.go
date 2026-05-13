package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryLoadMissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, err := LoadRegistry(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(r.List()) != 0 {
		t.Errorf("expected empty registry, got %d items", len(r.List()))
	}
}

func TestRegistryUpsertAndRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}

	item := InstalledArtifact{
		PluginID:         1,
		PluginIdentifier: "acme.x",
		Version:          "0.1.0",
		OS:               "linux",
		Arch:             "amd64",
		Checksum:         "abc",
		InstallDir:       "/tmp/store",
	}
	r.Upsert(item)

	// Upsert по тому же ключу должен заменить, а не дублировать
	item2 := item
	item2.Checksum = "def"
	r.Upsert(item2)

	if got := r.List(); len(got) != 1 {
		t.Fatalf("expected 1 item after duplicate upsert, got %d", len(got))
	}

	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Файл должен быть создан с приватными правами
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != registryFileMode {
		t.Errorf("registry file mode = %v, want %v", info.Mode().Perm(), registryFileMode)
	}

	// Перечитываем и проверяем, что данные сохранились
	r2, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry round-trip: %v", err)
	}
	round := r2.List()
	if len(round) != 1 {
		t.Fatalf("expected 1 item after reload, got %d", len(round))
	}
	if round[0].Checksum != "def" {
		t.Errorf("checksum not updated by Upsert: got %q, want %q", round[0].Checksum, "def")
	}
}

func TestRegistryFindExact(t *testing.T) {
	t.Parallel()
	r := &Registry{path: "ignored"}
	r.Upsert(InstalledArtifact{PluginID: 1, Version: "0.1.0", OS: "linux", Arch: "amd64", Checksum: "a"})
	r.Upsert(InstalledArtifact{PluginID: 1, Version: "0.2.0", OS: "linux", Arch: "amd64", Checksum: "b"})
	r.Upsert(InstalledArtifact{PluginID: 1, Version: "0.1.0", OS: "darwin", Arch: "arm64", Checksum: "c"})

	got, ok := r.FindExact(1, "0.1.0", "linux", "amd64")
	if !ok || got.Checksum != "a" {
		t.Errorf("FindExact returned %+v ok=%v", got, ok)
	}
	if _, ok := r.FindExact(1, "0.9.0", "linux", "amd64"); ok {
		t.Errorf("FindExact unexpectedly found missing version")
	}

	if v := r.FindByPlugin(1); len(v) != 3 {
		t.Errorf("FindByPlugin expected 3, got %d", len(v))
	}
}

func TestRegistryRemove(t *testing.T) {
	t.Parallel()
	r := &Registry{path: "ignored"}
	r.Upsert(InstalledArtifact{PluginID: 1, Version: "0.1.0", OS: "linux", Arch: "amd64"})
	r.Upsert(InstalledArtifact{PluginID: 2, Version: "0.1.0", OS: "linux", Arch: "amd64"})

	if !r.Remove(1, "0.1.0", "linux", "amd64") {
		t.Errorf("Remove returned false for existing item")
	}
	if r.Remove(99, "9", "x", "y") {
		t.Errorf("Remove returned true for missing item")
	}
	if len(r.List()) != 1 {
		t.Errorf("expected 1 item after remove, got %d", len(r.List()))
	}
}
