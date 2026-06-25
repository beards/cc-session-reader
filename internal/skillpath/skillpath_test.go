package skillpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillDir_GivenConfigDirSet_WhenResolved_ThenUsesConfigDir(t *testing.T) {
	cfg := filepath.Join("tmp", "custom-claude")
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)

	got, err := SkillDir()
	if err != nil {
		t.Fatalf("SkillDir() error = %v", err)
	}
	want := filepath.Join(cfg, "skills", SkillDirName)
	if got != want {
		t.Errorf("SkillDir() = %q, want %q", got, want)
	}
}

func TestSkillDir_GivenConfigDirUnset_WhenResolved_ThenUsesHomeClaude(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "placeholder")
	os.Unsetenv("CLAUDE_CONFIG_DIR")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	want := filepath.Join(home, ".claude", "skills", SkillDirName)

	got, err := SkillDir()
	if err != nil {
		t.Fatalf("SkillDir() error = %v", err)
	}
	if got != want {
		t.Errorf("SkillDir() = %q, want %q", got, want)
	}
}
