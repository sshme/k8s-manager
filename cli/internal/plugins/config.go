// Package plugins отвечает за жизненный цикл локально скачанных
// артефактов плагинов
package plugins

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	// defaultDirName это имя корневой папки в $HOME, куда CLI кладёт плагины и реестр
	defaultDirName = ".k8s-manager"
	envHome        = "K8S_MANAGER_HOME"
)

// Config задаёт корневую папку, в которой располагаются плагины и
// хранится локальный реестр
type Config struct {
	// Root абсолютный путь к корню, внутри ожидается подпапка plugins/
	Root string
}

// LoadConfig определяет корневую папку
func LoadConfig() Config {
	if custom := strings.TrimSpace(os.Getenv(envHome)); custom != "" {
		return Config{Root: custom}
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return Config{Root: filepath.Join(os.TempDir(), "k8s-manager")}
	}
	return Config{Root: filepath.Join(home, defaultDirName)}
}
