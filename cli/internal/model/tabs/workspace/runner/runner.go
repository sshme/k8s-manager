// Package runner собирает *exec.Cmd для запуска установленного плагина
// под bash. Stdout и stderr плагина пишутся одновременно в терминал и в лог-файл.
package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pluginsmgr "k8s-manager/cli/internal/plugins"
)

type pluginManifest struct {
	Runtime struct {
		Entrypoint string `json:"entrypoint"`
	} `json:"runtime"`
}

// Build готовит bash-команду для запуска плагина через go run.
// subcommand принимает только "plan" или "apply".
// при необходимости создаёт logDir.
func Build(a pluginsmgr.InstalledArtifact, subcommand, ctxName, namespace, deployment, logDir string) (*exec.Cmd, error) {
	if subcommand != "plan" && subcommand != "apply" {
		return nil, fmt.Errorf("unsupported subcommand %q", subcommand)
	}
	if strings.TrimSpace(a.InstallDir) == "" {
		return nil, fmt.Errorf("plugin install dir is empty")
	}

	contentDir := filepath.Join(a.InstallDir, "content")

	manifestPath := filepath.Join(contentDir, "plugin.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read plugin.json: %w", err)
	}
	var m pluginManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse plugin.json: %w", err)
	}
	entrypoint := strings.TrimSpace(m.Runtime.Entrypoint)
	if entrypoint == "" {
		return nil, fmt.Errorf("plugin.json: runtime.entrypoint is empty")
	}

	if _, err := os.Stat(filepath.Join(contentDir, "go.mod")); err != nil {
		return nil, fmt.Errorf("plugin content has no go.mod at %s: %w", contentDir, err)
	}

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return nil, fmt.Errorf("bash not found in PATH: %w", err)
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logPath := filepath.Join(logDir, logFileName(a.PluginIdentifier, subcommand))

	script := buildScript(entrypoint, subcommand, ctxName, namespace, deployment, logPath)
	cmd := exec.Command(bashPath, "-c", script)
	cmd.Dir = contentDir
	return cmd, nil
}

func buildScript(entrypoint, subcommand, ctxName, namespace, deployment, logPath string) string {
	return strings.Join([]string{
		"set -o pipefail", // если хотя бы одна команда упала
		"clear",
		fmt.Sprintf("go run %s %s --context %s --namespace %s --deployment %s 2>&1 | tee %s",
			shellQuote("./"+entrypoint),
			shellQuote(subcommand),
			shellQuote(ctxName),
			shellQuote(namespace),
			shellQuote(deployment),
			shellQuote(logPath),
		),
		"rc=$?",
		`printf '\nPress Enter to return...'`,
		"read -r _dummy",
		"exit $rc",
	}, "\n")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func logFileName(identifier, subcommand string) string {
	id := sanitizeForFilename(identifier)
	if id == "" {
		id = "plugin"
	}
	stamp := time.Now().Format("20060102-150405.000")
	return fmt.Sprintf("%s-%s-%s-pid%d.log", id, subcommand, stamp, os.Getpid())
}

func sanitizeForFilename(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ' ', ':':
			return '-'
		}
		return r
	}, s)
}
