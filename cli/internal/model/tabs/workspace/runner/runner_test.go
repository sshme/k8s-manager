package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pluginsmgr "k8s-manager/cli/internal/plugins"
)

func writeFakePlugin(t *testing.T, entrypoint string) string {
	t.Helper()
	dir := t.TempDir()
	content := filepath.Join(dir, "content")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"runtime":{"entrypoint":"` + entrypoint + `"}}`
	if err := os.WriteFile(filepath.Join(content, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "go.mod"), []byte("module fake\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestBuild_ReadsEntrypointFromManifest(t *testing.T) {
	dir := writeFakePlugin(t, "cmd/foo")
	a := pluginsmgr.InstalledArtifact{PluginIdentifier: "test.foo", InstallDir: dir}

	cmd, err := Build(a, "plan", "ctx-x", "ns-y", "dep-z", filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("Build returned nil cmd")
	}
	if len(cmd.Args) < 3 {
		t.Fatalf("expected bash -c <script>, got args: %v", cmd.Args)
	}
	script := cmd.Args[2]
	for _, want := range []string{"'./cmd/foo'", "'plan'", "'ctx-x'", "'ns-y'", "'dep-z'", "set -o pipefail", "clear", "tee ", "read -r _dummy"} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q\nscript:\n%s", want, script)
		}
	}
	wantDir := filepath.Join(dir, "content")
	if cmd.Dir != wantDir {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, wantDir)
	}
}

func TestBuild_MissingManifest_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := pluginsmgr.InstalledArtifact{PluginIdentifier: "x", InstallDir: dir}

	_, err := Build(a, "plan", "c", "n", "d", filepath.Join(dir, "logs"))
	if err == nil {
		t.Fatal("expected error for missing plugin.json")
	}
	if !strings.Contains(err.Error(), "plugin.json") {
		t.Errorf("error should mention plugin.json, got: %v", err)
	}
}

func TestBuild_EscapesSpecialChars(t *testing.T) {
	dir := writeFakePlugin(t, "cmd/foo")
	a := pluginsmgr.InstalledArtifact{PluginIdentifier: "test.foo", InstallDir: dir}

	ctxName := "arn:aws:eks:us-east-1:123/cluster"
	cmd, err := Build(a, "apply", ctxName, "ns'with'quote", "depl", filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	script := cmd.Args[2]

	wantCtx := "'arn:aws:eks:us-east-1:123/cluster'"
	if !strings.Contains(script, wantCtx) {
		t.Errorf("script missing escaped context %q\nscript:\n%s", wantCtx, script)
	}
	wantNS := `'ns'\''with'\''quote'`
	if !strings.Contains(script, wantNS) {
		t.Errorf("script missing escaped ns %q\nscript:\n%s", wantNS, script)
	}
}

func TestBuild_RejectsUnknownSubcommand(t *testing.T) {
	dir := writeFakePlugin(t, "cmd/foo")
	a := pluginsmgr.InstalledArtifact{PluginIdentifier: "x", InstallDir: dir}

	_, err := Build(a, "ohai", "c", "n", "d", filepath.Join(dir, "logs"))
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "ohai") {
		t.Errorf("error should mention bad subcommand, got: %v", err)
	}
}
