package tracker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectCallerSession_GivenConfigDirSet_WhenDetected_ThenScansConfigDirProjects(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfgDir)

	cwd := "/some/work/dir"
	projectDir := filepath.Join(cfgDir, "projects", strings.ReplaceAll(cwd, "/", "-"))
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "abc123.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectCallerSession(cwd)
	if got != "abc123" {
		t.Errorf("DetectCallerSession(%q) = %q, want %q", cwd, got, "abc123")
	}
}
