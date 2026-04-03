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
	"github.com/example/changes/internal/config"
)

func (a *App) runOptionalCommand(ctx context.Context, args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] != "collect" {
		return false, nil
	}
	if len(args) > 1 && args[1] == "reconstruct" {
		return true, a.runCollectReconstruct(ctx, args[2:])
	}
	if len(args) > 1 && args[1] == "drafts" {
		return true, a.runCollectDrafts(ctx, args[2:])
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

func (a *App) runCollectDrafts(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("collect drafts", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var inputPath string
	var outputDir string

	fs.StringVar(&inputPath, "input", "", "Path to collection snapshot manifest.json or rendered collection JSON")
	fs.StringVar(&outputDir, "output-dir", "", "Root directory for collect-changes product workspaces")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(inputPath) == "" {
		return fmt.Errorf("collect drafts: --input is required")
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	absInputPath := inputPath
	if !filepath.IsAbs(absInputPath) {
		absInputPath = filepath.Join(repoRoot, absInputPath)
	}

	absOutputDir := outputDir
	if strings.TrimSpace(absOutputDir) != "" && !filepath.IsAbs(absOutputDir) {
		absOutputDir = filepath.Join(repoRoot, absOutputDir)
	}

	resultSet, err := collection.LoadResultSet(absInputPath)
	if err != nil {
		return err
	}

	batch, err := collection.WriteDraftBatch(repoRoot, cfg, absInputPath, resultSet, a.Now(), a.Random, absOutputDir)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", batch.OutputDir)
	return nil
}

func (a *App) runCollectReconstruct(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("collect reconstruct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var inputPath string

	fs.StringVar(&inputPath, "input", "", "Path to collection snapshot manifest.json or rendered collection JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(inputPath) == "" {
		return fmt.Errorf("collect reconstruct: --input is required")
	}

	repoRoot, _, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	absInputPath := inputPath
	if !filepath.IsAbs(absInputPath) {
		absInputPath = filepath.Join(repoRoot, absInputPath)
	}

	resultSet, err := collection.LoadResultSet(absInputPath)
	if err != nil {
		return err
	}

	report, err := collection.Reconstruct(repoRoot, absInputPath, resultSet)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", filepath.Join(config.CollectChangesDir(repoRoot), "reconstruction-report.json"))
	_ = report
	return nil
}
