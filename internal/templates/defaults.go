package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/render"
)

type FileSet struct {
	Paths        []string
	CreatedPaths []string
}

func EnsureDefaultFiles(repoRoot string, cfg config.Config) (FileSet, error) {
	dir := config.TemplatesDir(repoRoot, cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileSet{}, fmt.Errorf("create templates directory: %w", err)
	}

	files := render.BuiltinTemplateFiles()

	paths := make([]string, 0, len(files))
	created := make([]string, 0, len(files))
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		body := files[name]
		path := filepath.Join(dir, name)
		written, err := writeIfMissing(path, body)
		if err != nil {
			return FileSet{}, err
		}
		paths = append(paths, path)
		if written {
			created = append(created, path)
		}
	}

	return FileSet{Paths: paths, CreatedPaths: created}, nil
}

func writeIfMissing(path, body string) (bool, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create template %s: %w", path, err)
	}
	if _, err := file.WriteString(body); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return false, fmt.Errorf("write template %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return false, fmt.Errorf("close template %s: %w", path, err)
	}
	return true, nil
}
