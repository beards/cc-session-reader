package claudepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir_GivenConfigDirSet_WhenResolved_ThenReturnsThatDir(t *testing.T) {
	want := filepath.Join("tmp", "custom-claude")
	t.Setenv("CLAUDE_CONFIG_DIR", want)

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestDir_GivenConfigDirUnset_WhenResolved_ThenReturnsHomeClaude(t *testing.T) {
	// Ensure t restores any pre-existing value, then clear it for this test.
	t.Setenv("CLAUDE_CONFIG_DIR", "placeholder")
	os.Unsetenv("CLAUDE_CONFIG_DIR")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	want := filepath.Join(home, ".claude")

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestDir_GivenConfigDirEmpty_WhenResolved_ThenFallsBackToHomeClaude(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	want := filepath.Join(home, ".claude")

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}
