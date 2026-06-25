package skillpath

import (
	"path/filepath"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/claudepath"
)

const SkillDirName = "cc-session"

func SkillDir() (string, error) {
	dir, err := claudepath.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skills", SkillDirName), nil
}
