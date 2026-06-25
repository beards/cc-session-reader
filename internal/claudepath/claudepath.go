// Package claudepath resolves Claude Code's configuration directory.
package claudepath

import (
	"os"
	"path/filepath"
)

// Dir returns Claude Code's configuration directory.
//
// It honors the CLAUDE_CONFIG_DIR environment variable, which Claude Code uses
// to relocate the entire ~/.claude tree (settings, credentials, session
// history, skills, usage data). When CLAUDE_CONFIG_DIR is unset or empty, it
// falls back to ~/.claude in the user's home directory.
func Dir() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}
