//go:build devtools

package collection

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
)

type ResultSet struct {
	CatalogPath string
	CollectedAt time.Time
	Results     []Result
}

type DraftBatch struct {
	InputPath   string
	CollectedAt time.Time
	OutputDir   string
	Drafts      []fragments.Fragment
}

type extractedSection struct {
	Title    string
	Body     string
	Type     string
	Bump     string
	Breaking bool
}

type extractedReleaseDocument struct {
	Heading  string
	Version  string
	Sections []extractedSection
}

var (
	markdownHeadingPattern = regexp.MustCompile(`^#{1,6}\s+`)
	htmlCommentPattern     = regexp.MustCompile(`<!--.*?-->`)
	releaseHeadingPattern  = regexp.MustCompile(`(?i)^(version\s+)?[a-z0-9 ._-]*\d+\.\d+(?:\.\d+)?(?:[-+][a-z0-9._-]+)?(?:\s+-\s+.+)?$`)
	versionTokenPattern    = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)
)

func LoadResultSet(path string) (ResultSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ResultSet{}, fmt.Errorf("read collection input %s: %w", path, err)
	}

	var payload struct {
		CatalogPath string    `json:"catalog_path"`
		CollectedAt time.Time `json:"collected_at"`
		Results     []Result  `json:"results"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ResultSet{}, fmt.Errorf("decode collection input %s: %w", path, err)
	}
	if len(payload.Results) == 0 {
		return ResultSet{}, fmt.Errorf("collection input %s does not contain any results", path)
	}

	return ResultSet{
		CatalogPath: payload.CatalogPath,
		CollectedAt: payload.CollectedAt,
		Results:     payload.Results,
	}, nil
}

func WriteDraftBatch(repoRoot string, cfg config.Config, inputPath string, resultSet ResultSet, now time.Time, random io.Reader, outputDir string) (DraftBatch, error) {
	_ = cfg
	if random == nil {
		random = rand.Reader
	}

	collectedAt := now.UTC().Truncate(time.Second)
	targetDir := strings.TrimSpace(outputDir)
	if targetDir == "" {
		targetDir = config.CollectChangesDir(repoRoot)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return DraftBatch{}, fmt.Errorf("create collect-changes root dir: %w", err)
	}

	results := slices.Clone(resultSet.Results)
	slices.SortFunc(results, func(a, b Result) int {
		return strings.Compare(a.Source.Name, b.Source.Name)
	})

	drafts := make([]fragments.Fragment, 0, len(results))
	for _, result := range results {
		if result.Error != "" {
			continue
		}

		doc, hasReleaseDoc := extractStructuredReleaseDocument(result)
		sections := doc.Sections
		if !hasReleaseDoc {
			sections = extractSections(result)
		}
		if len(sections) == 0 {
			sections = []extractedSection{fallbackSection(result)}
		}

		productSlug := slugify(valueOrFallback(result.Source.Product, result.Source.Name))
		productDir := filepath.Join(targetDir, productSlug, "changes")
		fragmentsDir := filepath.Join(productDir, "fragments")
		for _, dir := range []string{
			fragmentsDir,
			filepath.Join(productDir, "releases"),
			filepath.Join(productDir, "templates"),
		} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return DraftBatch{}, fmt.Errorf("create collect-changes product dir %s: %w", dir, err)
			}
		}

		for idx, section := range sections {
			sectionCreatedAt := collectedAt.Add(time.Duration(idx) * time.Second)
			id, err := fragments.GenerateID(sectionCreatedAt, section.Title, random)
			if err != nil {
				return DraftBatch{}, fmt.Errorf("generate draft fragment id: %w", err)
			}

			item := fragments.Fragment{
				Metadata: fragments.Metadata{
					ID:        id,
					CreatedAt: sectionCreatedAt,
					Title:     section.Title,
					Type:      section.Type,
					Bump:      section.Bump,
					Breaking:  section.Breaking,
					Scopes:    []string{"collect-changes", productSlug, result.Source.ID},
					Authors:   []string{"changes collect drafts"},
				},
				Body: section.Body,
				Path: filepath.Join(fragmentsDir, id+".md"),
			}
			if err := item.Validate(); err != nil {
				return DraftBatch{}, fmt.Errorf("validate collect-changes draft fragment %s: %w", section.Title, err)
			}
			if err := os.WriteFile(item.Path, []byte(item.Format()), 0o644); err != nil {
				return DraftBatch{}, fmt.Errorf("write collect-changes draft fragment %s: %w", section.Title, err)
			}
			drafts = append(drafts, item)
		}
	}

	if len(drafts) == 0 {
		return DraftBatch{}, fmt.Errorf("collection input %s did not contain any successful results to draft", inputPath)
	}

	return DraftBatch{
		InputPath:   inputPath,
		CollectedAt: collectedAt,
		OutputDir:   targetDir,
		Drafts:      drafts,
	}, nil
}

func extractSections(result Result) []extractedSection {
	if doc, ok := extractStructuredReleaseDocument(result); ok {
		return doc.Sections
	}

	lines := loadNormalizedLines(result)
	if len(lines) == 0 {
		return nil
	}
	lines = stripLeadingFrontMatter(lines)
	if len(lines) == 0 {
		return nil
	}

	type headingAt struct {
		Index int
		Text  string
	}
	headings := make([]headingAt, 0)
	for idx, line := range lines {
		heading := cleanHeadingLine(line)
		if !looksLikeReleaseHeading(heading) {
			continue
		}
		if len(headings) > 0 && headings[len(headings)-1].Text == heading {
			continue
		}
		headings = append(headings, headingAt{Index: idx, Text: heading})
	}
	if len(headings) == 0 {
		return nil
	}

	const maxSectionsPerSource = 12
	sections := make([]extractedSection, 0, min(len(headings), maxSectionsPerSource))
	for i, heading := range headings {
		if len(sections) >= maxSectionsPerSource {
			break
		}
		end := len(lines)
		if i+1 < len(headings) {
			end = headings[i+1].Index
		}
		bodyLines := trimEmptyLines(lines[heading.Index+1 : end])
		if len(bodyLines) == 0 {
			continue
		}
		body := strings.Join(bodyLines, "\n")
		title := limitTitle(fmt.Sprintf("%s %s", result.Source.Name, heading.Text))
		sectionType, bump, breaking := classifySection(heading.Text, body)
		sections = append(sections, extractedSection{
			Title:    title,
			Body:     body,
			Type:     sectionType,
			Bump:     bump,
			Breaking: breaking,
		})
	}
	return sections
}

func valueOrFallback(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fallbackSection(result Result) extractedSection {
	body := strings.TrimSpace(result.Excerpt)
	if body == "" {
		body = strings.TrimSpace(strings.Join(result.Headings[:min(len(result.Headings), 12)], "\n"))
	}
	if body == "" {
		body = fmt.Sprintf("Source URL: %s", result.Source.URL)
	}
	sectionType, bump, breaking := classifySection(result.Source.Name, body)
	return extractedSection{
		Title:    limitTitle(result.Source.Name),
		Body:     body,
		Type:     sectionType,
		Bump:     bump,
		Breaking: breaking,
	}
}

func loadNormalizedLines(result Result) []string {
	if strings.TrimSpace(result.NormalizedPath) == "" {
		return nil
	}
	raw, err := os.ReadFile(result.NormalizedPath)
	if err != nil {
		return nil
	}
	return strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
}

func cleanHeadingLine(line string) string {
	value := strings.TrimSpace(line)
	value = markdownHeadingPattern.ReplaceAllString(value, "")
	value = htmlCommentPattern.ReplaceAllString(value, "")
	value = strings.TrimSpace(strings.Trim(value, "*"))
	return value
}

func looksLikeReleaseHeading(value string) bool {
	value = strings.TrimSpace(htmlCommentPattern.ReplaceAllString(value, ""))
	if value == "" {
		return false
	}
	if len([]rune(value)) > 120 {
		return false
	}
	if len(versionTokenPattern.FindAllString(value, -1)) > 1 {
		return false
	}
	if releaseHeadingPattern.MatchString(value) {
		return true
	}
	return strings.Contains(strings.ToLower(value), "version ") && versionTokenPattern.MatchString(value)
}

func extractStructuredReleaseDocument(result Result) (extractedReleaseDocument, bool) {
	lines := loadNormalizedLines(result)
	if len(lines) == 0 {
		return extractedReleaseDocument{}, false
	}

	frontMatter, stripped := splitFrontMatter(lines)
	if len(stripped) == 0 {
		return extractedReleaseDocument{}, false
	}

	firstContent := -1
	for idx, line := range stripped {
		if strings.TrimSpace(line) == "" {
			continue
		}
		firstContent = idx
		break
	}
	if firstContent == -1 {
		return extractedReleaseDocument{}, false
	}

	rawHeading := strings.TrimSpace(stripped[firstContent])
	if !strings.HasPrefix(rawHeading, "# ") {
		return extractedReleaseDocument{}, false
	}

	heading := cleanHeadingLine(rawHeading)
	if !looksLikeReleaseHeading(heading) {
		return extractedReleaseDocument{}, false
	}

	version, ok := releaseIdentity(result, heading, frontMatter)
	if !ok {
		return extractedReleaseDocument{}, false
	}

	type headingAt struct {
		Index int
		Text  string
	}
	h2Headings := make([]headingAt, 0)
	for idx := firstContent + 1; idx < len(stripped); idx++ {
		raw := strings.TrimSpace(stripped[idx])
		if !strings.HasPrefix(raw, "## ") {
			continue
		}
		h2Headings = append(h2Headings, headingAt{
			Index: idx,
			Text:  cleanHeadingLine(raw),
		})
	}

	sections := make([]extractedSection, 0, max(len(h2Headings), 1))
	if len(h2Headings) == 0 {
		bodyLines := trimEmptyLines(stripped[firstContent+1:])
		if len(bodyLines) == 0 {
			return extractedReleaseDocument{}, false
		}
		body := strings.Join(bodyLines, "\n")
		sectionType, bump, breaking := classifySection(heading, body)
		sections = append(sections, extractedSection{
			Title:    singleReleaseTitle(result.Source.Name, heading),
			Body:     body,
			Type:     sectionType,
			Bump:     bump,
			Breaking: breaking,
		})
	} else {
		for i, item := range h2Headings {
			end := len(stripped)
			if i+1 < len(h2Headings) {
				end = h2Headings[i+1].Index
			}
			bodyLines := trimEmptyLines(stripped[item.Index+1 : end])
			if len(bodyLines) == 0 {
				continue
			}
			body := strings.Join(bodyLines, "\n")
			sectionType, bump, breaking := classifySection(item.Text, body)
			sections = append(sections, extractedSection{
				Title:    limitTitle(item.Text),
				Body:     body,
				Type:     sectionType,
				Bump:     bump,
				Breaking: breaking,
			})
		}
	}

	if len(sections) == 0 {
		return extractedReleaseDocument{}, false
	}

	return extractedReleaseDocument{
		Heading:  heading,
		Version:  version,
		Sections: sections,
	}, true
}

func stripLeadingFrontMatter(lines []string) []string {
	_, stripped := splitFrontMatter(lines)
	return stripped
}

func singleReleaseTitle(sourceName, heading string) string {
	sourceName = strings.TrimSpace(sourceName)
	heading = strings.TrimSpace(heading)
	sourceVersion := versionTokenPattern.FindString(sourceName)
	headingVersion := versionTokenPattern.FindString(heading)
	if sourceName != "" && sourceVersion != "" && sourceVersion == headingVersion {
		return limitTitle(sourceName)
	}
	if sourceName == "" {
		return limitTitle(heading)
	}
	return limitTitle(fmt.Sprintf("%s %s", sourceName, heading))
}

func splitFrontMatter(lines []string) (map[string]string, []string) {
	fields := map[string]string{}
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fields, lines
	}
	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) == "---" {
			return fields, lines[idx+1:]
		}
		key, value, ok := strings.Cut(lines[idx], ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return fields, lines
}

func releaseIdentity(result Result, heading string, frontMatter map[string]string) (string, bool) {
	rawVersion := firstNonEmpty(
		frontMatter["DownloadVersion"],
		normalizeShortVersion(frontMatter["Milestone"]),
		normalizeShortVersion(versionTokenPattern.FindString(heading)),
		normalizeShortVersion(versionTokenPattern.FindString(result.Source.Name)),
	)
	if rawVersion == "" {
		return "", false
	}

	productEdition := strings.ToLower(frontMatter["ProductEdition"])
	if strings.Contains(productEdition, "insiders") || strings.Contains(strings.ToLower(heading), "insiders") {
		return rawVersion + "-insiders.1", true
	}

	return rawVersion, true
}

func normalizeShortVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ".")
	if len(parts) == 2 {
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return ""
		}
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return ""
		}
		return value + ".0"
	}
	if len(parts) != 3 {
		return ""
	}
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return ""
		}
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func trimEmptyLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return slices.Clone(lines[start:end])
}

func classifySection(title, body string) (string, string, bool) {
	text := strings.ToLower(title + "\n" + body)
	breaking := strings.Contains(text, "breaking")
	switch {
	case breaking:
		return "changed", "major", true
	case strings.Contains(text, "**feat**") || strings.Contains(text, " feat") || strings.Contains(text, "feature") || strings.Contains(text, "added"):
		return "added", "minor", false
	case strings.Contains(text, "**fix**") || strings.Contains(text, "bug fix") || strings.Contains(text, "fixed"):
		return "fixed", "patch", false
	case strings.Contains(text, "security"):
		return "security", "patch", false
	case strings.Contains(text, "removed") || strings.Contains(text, "deprecat"):
		return "removed", "minor", false
	default:
		return "changed", "patch", false
	}
}

func limitTitle(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= 100 {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:100]))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
