package parser

import (
	"path/filepath"
	"testing"
)

func TestDefaultStore_GivenConfigDirSet_WhenBuilt_ThenDerivesFromConfigDir(t *testing.T) {
	cfg := filepath.Join("tmp", "custom-claude")
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)

	s := DefaultStore()

	if want := filepath.Join(cfg, "projects"); s.ProjectsDir != want {
		t.Errorf("ProjectsDir = %q, want %q", s.ProjectsDir, want)
	}
	if want := filepath.Join(cfg, "usage-data", "session-meta"); s.SessionMetaDir != want {
		t.Errorf("SessionMetaDir = %q, want %q", s.SessionMetaDir, want)
	}
}
