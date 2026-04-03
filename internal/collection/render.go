//go:build devtools

package collection

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	OutputFormatMarkdown = "markdown"
	OutputFormatJSON     = "json"
)

func Render(corpus Corpus, format string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(format)) {
	case "", OutputFormatMarkdown:
		return renderMarkdown(corpus), nil
	case OutputFormatJSON:
		bytes, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			return "", fmt.Errorf("encode collection corpus: %w", err)
		}
		return string(bytes), nil
	default:
		return "", fmt.Errorf("unsupported collection output format %q", format)
	}
}

func renderMarkdown(corpus Corpus) string {
	var builder strings.Builder
	builder.WriteString("# Changelog Collection\n\n")
	builder.WriteString(fmt.Sprintf("Collected %d sources at `%s`.\n\n", corpus.SourceCount, corpus.CollectedAt.Format("2006-01-02T15:04:05Z07:00")))
	builder.WriteString(fmt.Sprintf("- Successes: %d\n", corpus.SuccessCount))
	builder.WriteString(fmt.Sprintf("- Failures: %d\n", corpus.FailureCount))
	builder.WriteString(fmt.Sprintf("- Snapshot directory: `%s`\n", corpus.SnapshotDir))
	if corpus.CatalogPath != "" {
		builder.WriteString(fmt.Sprintf("- Catalog: `%s`\n", corpus.CatalogPath))
	}

	for _, result := range corpus.Results {
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("## %s\n\n", result.Source.Name))
		builder.WriteString(fmt.Sprintf("- URL: `%s`\n", result.Source.URL))
		if result.StatusCode > 0 {
			builder.WriteString(fmt.Sprintf("- HTTP status: `%d`\n", result.StatusCode))
		}
		if result.ContentType != "" {
			builder.WriteString(fmt.Sprintf("- Content-Type: `%s`\n", result.ContentType))
		}
		if result.DetectedFormat != "" {
			builder.WriteString(fmt.Sprintf("- Detected format: `%s`\n", result.DetectedFormat))
		}
		if result.RawPath != "" {
			builder.WriteString(fmt.Sprintf("- Raw snapshot: `%s`\n", result.RawPath))
		}
		if result.NormalizedPath != "" {
			builder.WriteString(fmt.Sprintf("- Normalized snapshot: `%s`\n", result.NormalizedPath))
		}
		if result.Error != "" {
			builder.WriteString(fmt.Sprintf("- Error: `%s`\n", result.Error))
		}
		if result.Title != "" {
			builder.WriteString(fmt.Sprintf("- Extracted title: %s\n", result.Title))
		}

		if len(result.Headings) > 0 {
			builder.WriteString("\n### Headings\n\n")
			for _, heading := range result.Headings {
				builder.WriteString(fmt.Sprintf("- %s\n", heading))
			}
		}

		if result.Excerpt != "" {
			builder.WriteString("\n### Excerpt\n\n")
			builder.WriteString(result.Excerpt)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}
