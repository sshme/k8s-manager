package plugins

import (
	"path/filepath"
	"strings"
)

// pluginsSubdir это подпапка под корнем, где раскладываются артефакты
const pluginsSubdir = "plugins"

// storeSubdir хранит распакованные релизы по платформам
const storeSubdir = "store"

// Имена служебных файлов внутри каталога релиза
const (
	zipFileName     = "artifact.zip"
	partialFileName = "artifact.zip.partial"
	contentDirName  = "content"
	installMetaName = "install.json"
	registryFile    = "registry.json"
)

// pluginsRoot возвращает корень для плагинов
func pluginsRoot(cfg Config) string {
	return filepath.Join(cfg.Root, pluginsSubdir)
}

// registryPath это путь к локальному реестру установленных плагинов
func registryPath(cfg Config) string {
	return filepath.Join(pluginsRoot(cfg), registryFile)
}

// storeDir возвращает каталог релиза по полному набору ключей.
func storeDir(cfg Config, publisher, identifier, version, osName, arch string) string {
	return filepath.Join(
		pluginsRoot(cfg),
		storeSubdir,
		sanitizeSegment(publisher),
		sanitizeSegment(identifier),
		sanitizeSegment(version),
		sanitizeSegment(osName)+"-"+sanitizeSegment(arch),
	)
}

// zipPath путь к финальному zip файлу артефакта внутри каталога релиза
func zipPath(storeDir string) string {
	return filepath.Join(storeDir, zipFileName)
}

// partialZipPath путь к временному zip файлу для атомарной записи
func partialZipPath(storeDir string) string {
	return filepath.Join(storeDir, partialFileName)
}

// contentDir путь к распакованному содержимому артефакта
func contentDir(storeDir string) string {
	return filepath.Join(storeDir, contentDirName)
}

// installMetaPath путь к install.json внутри каталога релиза
func installMetaPath(storeDir string) string {
	return filepath.Join(storeDir, installMetaName)
}

// sanitizeSegment превращает произвольную строку в безопасный сегмент пути.
// Разрешены только латинские буквы, цифры, точка, подчёркивание и дефис.
// Остальные символы заменяются на _. Пустая строка превращается в "_",
// чтобы не получить пустой сегмент в пути.
func sanitizeSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	// На случай если строка состояла только из недопустимых символов
	out := b.String()
	if strings.Trim(out, "_") == "" {
		return "_"
	}
	return out
}
