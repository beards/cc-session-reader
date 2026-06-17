package parser

import (
	"os"
	"path/filepath"
)

// Store points at Claude Code's on-disk session data.
type Store struct {
	ProjectsDir    string
	SessionMetaDir string
}

// DefaultStore returns a Store derived from the current user's ~/.claude.
func DefaultStore() Store {
	claudeDir := filepath.Join(homeDir(), ".claude")
	return Store{
		ProjectsDir:    filepath.Join(claudeDir, "projects"),
		SessionMetaDir: filepath.Join(claudeDir, "usage-data", "session-meta"),
	}
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
