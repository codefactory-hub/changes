//go:build devtools

package collection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/render"
	"github.com/example/changes/internal/templates"
)

type ReconstructionReport struct {
	InputPath string                  `json:"input_path"`
	Products  []ProductReconstruction `json:"products"`
}

type ProductReconstruction struct {
	Product   string             `json:"product"`
	Workspace string             `json:"workspace"`
	Sources   []SourceComparison `json:"sources"`
}

type SourceComparison struct {
	SourceID           string   `json:"source_id"`
	SourceName         string   `json:"source_name"`
	FragmentCount      int      `json:"fragment_count"`
	ReleaseRecordCount int      `json:"release_record_count"`
	OriginalPath       string   `json:"original_path"`
	RenderedPath       string   `json:"rendered_path"`
	Recall             float64  `json:"recall"`
	Precision          float64  `json:"precision"`
	SharedLines        int      `json:"shared_lines"`
	OriginalLines      int      `json:"original_lines"`
	RenderedLines      int      `json:"rendered_lines"`
	MissingExamples    []string `json:"missing_examples"`
	UnexpectedExamples []string `json:"unexpected_examples"`
}

func Reconstruct(repoRoot string, inputPath string, resultSet ResultSet) (ReconstructionReport, error) {
	products := map[string][]Result{}
	for _, result := range resultSet.Results {
		if result.Error != "" {
			continue
		}
		productSlug := slugify(valueOrFallback(result.Source.Product, result.Source.Name))
		products[productSlug] = append(products[productSlug], result)
	}

	productNames := make([]string, 0, len(products))
	for name := range products {
		productNames = append(productNames, name)
	}
	slices.Sort(productNames)

	report := ReconstructionReport{
		InputPath: inputPath,
		Products:  make([]ProductReconstruction, 0, len(productNames)),
	}

	for _, productSlug := range productNames {
		workspace := filepath.Join(config.CollectChangesDir(repoRoot), productSlug)
		cfg := collectWorkspaceConfig(productSlug)
		if _, err := templates.EnsureDefaultFiles(workspace, cfg); err != nil {
			return ReconstructionReport{}, fmt.Errorf("ensure templates for %s: %w", productSlug, err)
		}

		allFragments, err := fragments.List(workspace, cfg)
		if err != nil {
			return ReconstructionReport{}, fmt.Errorf("list fragments for %s: %w", productSlug, err)
		}

		productReport := ProductReconstruction{
			Product:   productSlug,
			Workspace: workspace,
			Sources:   make([]SourceComparison, 0, len(products[productSlug])),
		}

		results := products[productSlug]
		slices.SortFunc(results, func(a, b Result) int {
			return strings.Compare(a.Source.Name, b.Source.Name)
		})

		for _, result := range results {
			sourceFragments := filterFragmentsForSource(allFragments, result.Source.ID)
			sections := extractSections(result)
			records, err := buildSourceReleaseRecords(result, sourceFragments, sections, resultSet.CollectedAt)
			if err != nil {
				return ReconstructionReport{}, fmt.Errorf("build release records for %s: %w", result.Source.Name, err)
			}
			if len(records) == 0 {
				continue
			}

			recordDir := config.ReleasesDir(workspace, cfg)
			if err := os.MkdirAll(recordDir, 0o755); err != nil {
				return ReconstructionReport{}, fmt.Errorf("create release-record dir for %s: %w", result.Source.Name, err)
			}
			for _, record := range records {
				filename := fmt.Sprintf("%s-%s.toml", result.Source.ID, record.Version)
				if err := writeReleaseRecordAt(filepath.Join(recordDir, filename), record); err != nil {
					return ReconstructionReport{}, fmt.Errorf("write release record for %s: %w", result.Source.Name, err)
				}
			}

			renderedPath := filepath.Join(workspace, "changes", "rendered", result.Source.ID, "repository_markdown.md")
			if err := os.MkdirAll(filepath.Dir(renderedPath), 0o755); err != nil {
				return ReconstructionReport{}, fmt.Errorf("create rendered dir for %s: %w", result.Source.Name, err)
			}

			renderer, err := render.New(workspace, cfg, config.RenderProfileRepositoryMarkdown)
			if err != nil {
				return ReconstructionReport{}, fmt.Errorf("create renderer for %s: %w", result.Source.Name, err)
			}
			head := records[len(records)-1]
			bundles, err := releases.AssembleReleaseLineage(head, records, sourceFragments)
			if err != nil {
				return ReconstructionReport{}, fmt.Errorf("build render doc for %s: %w", result.Source.Name, err)
			}
			rendered, err := renderer.Render(render.Document{Bundles: bundles})
			if err != nil {
				return ReconstructionReport{}, fmt.Errorf("render %s: %w", result.Source.Name, err)
			}
			if err := os.WriteFile(renderedPath, []byte(rendered), 0o644); err != nil {
				return ReconstructionReport{}, fmt.Errorf("write rendered output for %s: %w", result.Source.Name, err)
			}

			comparison := compareRendered(result, rendered)
			comparison.SourceID = result.Source.ID
			comparison.SourceName = result.Source.Name
			comparison.FragmentCount = len(sourceFragments)
			comparison.ReleaseRecordCount = len(records)
			comparison.OriginalPath = result.NormalizedPath
			comparison.RenderedPath = renderedPath
			productReport.Sources = append(productReport.Sources, comparison)
		}

		report.Products = append(report.Products, productReport)
	}

	if err := writeReconstructionReport(repoRoot, report); err != nil {
		return ReconstructionReport{}, err
	}
	return report, nil
}

func collectWorkspaceConfig(productSlug string) config.Config {
	cfg := config.Default()
	cfg.Project.Name = productSlug
	cfg.Project.ChangelogFile = "changes/CHANGELOG.md"
	cfg.Paths.DataDir = "changes"
	cfg.Paths.StateDir = "changes/state"
	cfg.Paths.TemplatesDir = "changes/templates"
	return cfg
}

func filterFragmentsForSource(all []fragments.Fragment, sourceID string) []fragments.Fragment {
	out := make([]fragments.Fragment, 0)
	for _, item := range all {
		if slices.Contains(item.Scopes, sourceID) {
			out = append(out, item)
		}
	}
	return out
}

func buildSourceReleaseRecords(result Result, sourceFragments []fragments.Fragment, sections []extractedSection, collectedAt time.Time) ([]releases.ReleaseRecord, error) {
	if len(sourceFragments) == 0 {
		return nil, nil
	}
	if doc, ok := extractStructuredReleaseDocument(result); ok {
		ordered := orderFragmentsBySectionTitle(sourceFragments, doc.Sections)
		if len(ordered) == 0 {
			ordered = slices.Clone(sourceFragments)
			slices.SortFunc(ordered, func(a, b fragments.Fragment) int {
				return strings.Compare(a.ID, b.ID)
			})
		}
		record := releases.ReleaseRecord{
			Product:          result.Source.ID,
			Version:          doc.Version,
			CreatedAt:        collectedAt.UTC(),
			AddedFragmentIDs: fragmentIDs(ordered),
		}
		if err := record.Validate(); err != nil {
			return nil, err
		}
		return []releases.ReleaseRecord{record}, nil
	}

	ordered := orderFragmentsBySectionTitle(sourceFragments, sections)
	if len(ordered) == 0 {
		ordered = slices.Clone(sourceFragments)
		slices.SortFunc(ordered, func(a, b fragments.Fragment) int {
			return strings.Compare(a.ID, b.ID)
		})
	}

	slices.Reverse(ordered)
	records := make([]releases.ReleaseRecord, 0, len(ordered))
	parentVersion := ""
	for idx, item := range ordered {
		version := fmt.Sprintf("0.0.%d", idx+1)
		record := releases.ReleaseRecord{
			Product:          result.Source.ID,
			Version:          version,
			ParentVersion:    parentVersion,
			CreatedAt:        collectedAt.Add(time.Duration(idx) * time.Second).UTC(),
			AddedFragmentIDs: []string{item.ID},
		}
		if err := record.Validate(); err != nil {
			return nil, err
		}
		records = append(records, record)
		parentVersion = version
	}
	return records, releases.ValidateSet(records)
}

func orderFragmentsBySectionTitle(sourceFragments []fragments.Fragment, sections []extractedSection) []fragments.Fragment {
	fragmentByTitle := map[string][]fragments.Fragment{}
	for _, item := range sourceFragments {
		fragmentByTitle[item.Title] = append(fragmentByTitle[item.Title], item)
	}
	for title := range fragmentByTitle {
		queue := fragmentByTitle[title]
		slices.SortFunc(queue, func(a, b fragments.Fragment) int {
			return strings.Compare(a.ID, b.ID)
		})
		fragmentByTitle[title] = queue
	}

	ordered := make([]fragments.Fragment, 0)
	for _, section := range sections {
		queue := fragmentByTitle[section.Title]
		if len(queue) == 0 {
			continue
		}
		ordered = append(ordered, queue[0])
		fragmentByTitle[section.Title] = queue[1:]
	}
	return ordered
}

func fragmentIDs(items []fragments.Fragment) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func writeReleaseRecordAt(path string, record releases.ReleaseRecord) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := toml.NewEncoder(file).Encode(record); err != nil {
		return err
	}
	return nil
}

func compareRendered(result Result, rendered string) SourceComparison {
	originalLines := normalizedLineSetFromFile(result.NormalizedPath)
	renderedLines := normalizedLineSet(rendered)
	shared := intersection(originalLines, renderedLines)
	missing := difference(originalLines, renderedLines)
	unexpected := difference(renderedLines, originalLines)

	recall := ratio(len(shared), len(originalLines))
	precision := ratio(len(shared), len(renderedLines))

	return SourceComparison{
		Recall:             recall,
		Precision:          precision,
		SharedLines:        len(shared),
		OriginalLines:      len(originalLines),
		RenderedLines:      len(renderedLines),
		MissingExamples:    firstN(missing, 8),
		UnexpectedExamples: firstN(unexpected, 8),
	}
}

func normalizedLineSetFromFile(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return normalizedLineSet(string(raw))
}

func normalizedLineSet(value string) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized := normalizeCompareLine(line)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	slices.Sort(out)
	return out
}

func normalizeCompareLine(line string) string {
	value := strings.TrimSpace(strings.ToLower(line))
	value = strings.TrimLeft(value, "#")
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "-* ")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) < 4 {
		return ""
	}
	return value
}

func intersection(a, b []string) []string {
	set := map[string]struct{}{}
	for _, item := range b {
		set[item] = struct{}{}
	}
	out := make([]string, 0)
	for _, item := range a {
		if _, ok := set[item]; ok {
			out = append(out, item)
		}
	}
	return out
}

func difference(a, b []string) []string {
	set := map[string]struct{}{}
	for _, item := range b {
		set[item] = struct{}{}
	}
	out := make([]string, 0)
	for _, item := range a {
		if _, ok := set[item]; !ok {
			out = append(out, item)
		}
	}
	return out
}

func ratio(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom)
}

func firstN(values []string, n int) []string {
	if len(values) <= n {
		return values
	}
	return values[:n]
}

func writeReconstructionReport(repoRoot string, report ReconstructionReport) error {
	path := filepath.Join(config.CollectChangesDir(repoRoot), "reconstruction-report.json")
	bytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o644)
}
