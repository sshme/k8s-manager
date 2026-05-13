package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// registrySchemaVersion версия формата на диске
const registrySchemaVersion = 1

// registryFileMode и registryDirMode задают приватные права для папок и
// файла реестра
const (
	registryFileMode fs.FileMode = 0o600
	registryDirMode  fs.FileMode = 0o700
)

// Registry это потокобезопасный json файл со списком установленных артефактов.
// Все мутации атомарно записываются на диск через временный файл и rename.
type Registry struct {
	mu      sync.Mutex
	path    string
	version int
	items   []InstalledArtifact
}

// registryFileLayout это форма json на диске. Отделяем от Registry,
// чтобы внутри держать sync.Mutex и кэш в памяти.
type registryFileLayout struct {
	Version   int                 `json:"version"`
	Installed []InstalledArtifact `json:"installed"`
}

// LoadRegistry открывает (или создаёт) реестр по указанному пути.
// Если файла нет, возвращается пустой реестр в памяти, на Save он будет создан.
func LoadRegistry(path string) (*Registry, error) {
	r := &Registry{
		path:    path,
		version: registrySchemaVersion,
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var layout registryFileLayout
	if err := json.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	if layout.Version == 0 {
		layout.Version = registrySchemaVersion
	}
	r.version = layout.Version
	r.items = layout.Installed
	return r, nil
}

// Save атомарно записывает реестр на диск. Сначала пишем во временный файл
// в той же папке, потом rename, чтобы при падении не получить
// полупустой файл
func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveLocked()
}

// saveLocked это внутренняя реализация Save, вызываемая когда мьютекс уже захвачен
func (r *Registry) saveLocked() error {
	layout := registryFileLayout{
		Version:   r.version,
		Installed: r.items,
	}
	if layout.Installed == nil {
		layout.Installed = []InstalledArtifact{}
	}

	data, err := json.MarshalIndent(layout, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, registryDirMode); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "registry-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp registry: %w", err)
	}
	tmpPath := tmp.Name()

	// Если что-то пойдёт не так до Rename, чистим временный файл
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp registry: %w", err)
	}
	if err := tmp.Chmod(registryFileMode); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp registry: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp registry: %w", err)
	}
	if err := os.Rename(tmpPath, r.path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp registry: %w", err)
	}
	return nil
}

// List возвращает копию всех записей в реестре
func (r *Registry) List() []InstalledArtifact {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]InstalledArtifact, len(r.items))
	copy(out, r.items)
	return out
}

// FindByPlugin возвращает все записи для конкретного pluginID
func (r *Registry) FindByPlugin(pluginID int64) []InstalledArtifact {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []InstalledArtifact
	for _, it := range r.items {
		if it.PluginID == pluginID {
			out = append(out, it)
		}
	}
	return out
}

// FindExact ищет точное совпадение по составному ключу
func (r *Registry) FindExact(pluginID int64, version, osName, arch string) (InstalledArtifact, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, it := range r.items {
		if it.PluginID == pluginID && it.Version == version && it.OS == osName && it.Arch == arch {
			return it, true
		}
	}
	return InstalledArtifact{}, false
}

// Upsert добавляет новую запись или заменяет существующую по составному ключу.
// На диск ничего не пишется, для сохранения нужен явный Save.
func (r *Registry) Upsert(item InstalledArtifact) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, it := range r.items {
		if it.PluginID == item.PluginID && it.Version == item.Version && it.OS == item.OS && it.Arch == item.Arch {
			r.items[i] = item
			return
		}
	}
	r.items = append(r.items, item)
}

// Remove удаляет запись по составному ключу. Возвращает true если что-то удалили.
func (r *Registry) Remove(pluginID int64, version, osName, arch string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, it := range r.items {
		if it.PluginID == pluginID && it.Version == version && it.OS == osName && it.Arch == arch {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return true
		}
	}
	return false
}
