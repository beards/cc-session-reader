package tokens

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
)

func TestNewCounter_GivenNoKeyAndConfigDirSet_WhenCreated_ThenErrorMentionsConfigDirPath(t *testing.T) {
	config.Reset()
	defer config.Reset()

	cfgDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfgDir)
	t.Setenv("ANTHROPIC_API_KEY", "")

	_, err := NewCounter("")
	if err == nil {
		t.Fatal("NewCounter returned nil error, want missing key error")
	}

	want := filepath.Join(cfgDir, "skills", "cc-session", "config.json")
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}
