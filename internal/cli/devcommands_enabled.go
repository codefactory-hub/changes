//go:build devtools

package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/changes/internal/collection"
)

func (a *App) runOptionalCommand(ctx context.Context, args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] != "collect" {
		return false, nil
	}
	return true, a.runCollect(ctx, args[1:])
}

func (a *App) runCollect(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var catalogPath string
	var outputPath string
	var format string

	fs.StringVar(&catalogPath, "catalog", "", "Path to a TOML catalog of changelog sources")
	fs.StringVar(&outputPath, "output", "", "Output path")
	fs.StringVar(&format, "format", collection.OutputFormatMarkdown, "Output format (markdown|json)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(catalogPath) == "" {
		return fmt.Errorf("collect: --catalog is required")
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	absCatalogPath := catalogPath
	if !filepath.IsAbs(absCatalogPath) {
		absCatalogPath = filepath.Join(repoRoot, absCatalogPath)
	}

	catalog, err := collection.LoadCatalog(absCatalogPath)
	if err != nil {
		return err
	}

	var client collection.HTTPClient
	if a.HTTPClient != nil {
		typed, ok := a.HTTPClient.(collection.HTTPClient)
		if !ok {
			return fmt.Errorf("collect: app HTTPClient does not implement collection.HTTPClient")
		}
		client = typed
	}

	corpus, err := collection.Collect(ctx, repoRoot, cfg, client, absCatalogPath, catalog, a.Now())
	if err != nil {
		return err
	}

	content, err := collection.Render(corpus, format)
	if err != nil {
		return err
	}

	if strings.TrimSpace(outputPath) != "" {
		targetPath := outputPath
		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(repoRoot, targetPath)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write collection output: %w", err)
		}
		_, _ = fmt.Fprintf(a.Stdout, "%s\n", targetPath)
		return nil
	}

	_, _ = fmt.Fprint(a.Stdout, content)
	return nil
}
