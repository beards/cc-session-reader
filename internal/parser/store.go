package parser

import (
	"path/filepath"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/claudepath"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

// Store points at Claude Code's on-disk session data.
type Store struct {
	ProjectsDir    string
	SessionMetaDir string
	HeaderScanner  session.HeaderScanner
}

// DefaultStore returns a Store derived from Claude Code's configuration
// directory (CLAUDE_CONFIG_DIR, falling back to ~/.claude).
// Call DefaultStoreWith to inject a HeaderScanner.
func DefaultStore() Store {
	claudeDir, _ := claudepath.Dir()
	return Store{
		ProjectsDir:    filepath.Join(claudeDir, "projects"),
		SessionMetaDir: filepath.Join(claudeDir, "usage-data", "session-meta"),
	}
}

// DefaultStoreWith returns a Store with the given HeaderScanner injected.
func DefaultStoreWith(scanner session.HeaderScanner) Store {
	s := DefaultStore()
	s.HeaderScanner = scanner
	return s
}
