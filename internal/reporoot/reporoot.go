package reporoot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNotGitRepo = errors.New("not inside a git repository")

func Detect(start string) (string, error) {
	if start == "" {
		return "", fmt.Errorf("detect repo root: empty start path")
	}

	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("detect repo root: %w", err)
	}

	for {
		if isGitDir(current) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrNotGitRepo
		}
		current = parent
	}
}

func isGitDir(root string) bool {
	path := filepath.Join(root, ".git")
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir() || !info.IsDir()
}
