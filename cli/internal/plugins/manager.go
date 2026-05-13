package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"k8s-manager/cli/internal/market"
)

// storeDirMode задаёт права на каталоги под артефакты
const storeDirMode fs.FileMode = 0o700

// Manager управляет жизненным циклом локально скачанных артефактов.
// Хранит конфиг, ссылку на маркет-сервис и загруженный реестр.
type Manager struct {
	cfg      Config
	market   artifactDownloader
	registry *Registry
}

// NewManager создаёт корневые папки и реестр.
// Если файла реестра нет, он будет создан при первой успешной установке.
func NewManager(cfg Config, mkt *market.Service) (*Manager, error) {
	if mkt == nil {
		return nil, fmt.Errorf("plugins manager: market service is required")
	}
	return newManager(cfg, mkt)
}

func newManager(cfg Config, mkt artifactDownloader) (*Manager, error) {
	if cfg.Root == "" {
		return nil, fmt.Errorf("plugins config: root is required")
	}

	if err := os.MkdirAll(pluginsRoot(cfg), storeDirMode); err != nil {
		return nil, fmt.Errorf("create plugins root: %w", err)
	}

	reg, err := LoadRegistry(registryPath(cfg))
	if err != nil {
		return nil, err
	}

	return &Manager{
		cfg:      cfg,
		market:   mkt,
		registry: reg,
	}, nil
}

// Root возвращает корневую папку, где лежат плагины
func (m *Manager) Root() string { return pluginsRoot(m.cfg) }

// IsInstalled проверяет, есть ли в реестре запись с точным совпадением ключа
func (m *Manager) IsInstalled(pluginID int64, version, osName, arch string) bool {
	_, ok := m.registry.FindExact(pluginID, version, osName, arch)
	return ok
}

// FindForPlugin возвращает все записи для указанного pluginID
func (m *Manager) FindForPlugin(pluginID int64) []InstalledArtifact {
	return m.registry.FindByPlugin(pluginID)
}

// ListInstalled отдаёт копию полного списка установленных артефактов.
func (m *Manager) ListInstalled() []InstalledArtifact {
	return m.registry.List()
}

// Install скачивает артефакт (если не скачан с тем же checksum), верифицирует,
// распаковывает и обновляет реестр. Возвращает финальную запись из реестра.
func (m *Manager) Install(ctx context.Context, ref InstallRef) (*InstalledArtifact, error) {
	if err := validateInstallRef(ref); err != nil {
		return nil, err
	}

	// Если уже стоит с правильным checksum и все файлы на месте, выходим сразу
	if existing, ok := m.registry.FindExact(ref.PluginID, ref.Version, ref.Artifact.OS, ref.Artifact.Arch); ok {
		if existing.Checksum == ref.Artifact.Checksum && filesPresent(existing.InstallDir) {
			return &existing, nil
		}
	}

	dir := storeDir(m.cfg, ref.PublisherName, ref.PluginIdentifier, ref.Version, ref.Artifact.OS, ref.Artifact.Arch)
	if err := os.MkdirAll(dir, storeDirMode); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	// Скачиваем во временный файл, чтобы при сбое не оставить мусор
	partial := partialZipPath(dir)
	if err := os.Remove(partial); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("clean partial: %w", err)
	}
	checksum, size, err := writePartialAndVerify(ctx, m.market, ref.Artifact.ID, partial, ref.Artifact.Size, ref.Artifact.Checksum)
	if err != nil {
		// При несовпадении checksum или size удаляем мусор
		_ = os.Remove(partial)
		return nil, err
	}

	// Атомарный rename в финальное имя
	finalZip := zipPath(dir)
	if err := os.Rename(partial, finalZip); err != nil {
		_ = os.Remove(partial)
		return nil, fmt.Errorf("rename artifact: %w", err)
	}

	// Удаляем старый content/ (если был от прошлой попытки) и распаковываем заново
	content := contentDir(dir)
	if err := os.RemoveAll(content); err != nil {
		return nil, fmt.Errorf("clean content dir: %w", err)
	}
	if err := extractZip(finalZip, content); err != nil {
		return nil, fmt.Errorf("extract artifact: %w", err)
	}

	// Собираем запись реестра
	item := InstalledArtifact{
		PluginID:         ref.PluginID,
		PluginIdentifier: ref.PluginIdentifier,
		PluginName:       ref.PluginName,
		PublisherID:      ref.PublisherID,
		PublisherName:    ref.PublisherName,
		ReleaseID:        ref.ReleaseID,
		Version:          ref.Version,
		ArtifactID:       ref.Artifact.ID,
		OS:               ref.Artifact.OS,
		Arch:             ref.Artifact.Arch,
		Type:             ref.Artifact.Type,
		Size:             size,
		Checksum:         checksum,
		InstallDir:       dir,
		DownloadedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	// Создаём файл с метаданными рядом с zip
	if err := writeInstallMeta(installMetaPath(dir), item); err != nil {
		return nil, err
	}

	// Обновляем реестр и сохраняем
	m.registry.Upsert(item)
	if err := m.registry.Save(); err != nil {
		return nil, err
	}
	return &item, nil
}

// Uninstall удаляет конкретный установленный артефакт. Возвращает ошибку
// только если есть запись и не удалось почистить файлы, если его и не было,
// то операция считается успешной
func (m *Manager) Uninstall(pluginID int64, version, osName, arch string) error {
	item, ok := m.registry.FindExact(pluginID, version, osName, arch)
	if !ok {
		return nil
	}

	if item.InstallDir != "" {
		if err := os.RemoveAll(item.InstallDir); err != nil {
			return fmt.Errorf("remove install dir: %w", err)
		}
	}
	m.registry.Remove(pluginID, version, osName, arch)
	return m.registry.Save()
}

// writePartialAndVerify скачивает артефакт в указанный путь и сверяет
// checksum и size с тем, что ожидается. Возвращает фактический checksum и размер.
func writePartialAndVerify(ctx context.Context, downloader artifactDownloader, artifactID int64, partialPath string, expectedSize int64, expectedChecksum string) (string, int64, error) {
	f, err := os.OpenFile(partialPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("create partial file: %w", err)
	}

	checksum, size, downloadErr := downloadVerifiedTo(ctx, downloader, artifactID, f)
	closeErr := f.Close()
	if downloadErr != nil {
		return "", 0, downloadErr
	}
	if closeErr != nil {
		return "", 0, fmt.Errorf("close partial file: %w", closeErr)
	}

	if err := verifyDownloaded(size, checksum, expectedSize, expectedChecksum); err != nil {
		return "", 0, err
	}
	return checksum, size, nil
}

// writeInstallMeta атомарно пишет install.json рядом с zip файлом
func writeInstallMeta(path string, item InstalledArtifact) error {
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal install meta: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "install-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp install meta: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp install meta: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp install meta: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename install meta: %w", err)
	}
	return nil
}

// validateInstallRef проверяет, что в запросе на установку заполнено всё
// нужное чтобы построить путь и начать установку
func validateInstallRef(ref InstallRef) error {
	if ref.PluginID <= 0 {
		return fmt.Errorf("install: plugin id is required")
	}
	if ref.PluginIdentifier == "" {
		return fmt.Errorf("install: plugin identifier is required")
	}
	if ref.Version == "" {
		return fmt.Errorf("install: release version is required")
	}
	if ref.Artifact.ID <= 0 {
		return fmt.Errorf("install: artifact id is required")
	}
	if ref.Artifact.OS == "" || ref.Artifact.Arch == "" {
		return fmt.Errorf("install: artifact os/arch are required")
	}
	return nil
}

// filesPresent говорит о факте наличия файлов артефакта на диске
func filesPresent(dir string) bool {
	if dir == "" {
		return false
	}
	if _, err := os.Stat(zipPath(dir)); err != nil {
		return false
	}
	if _, err := os.Stat(contentDir(dir)); err != nil {
		return false
	}
	return true
}
