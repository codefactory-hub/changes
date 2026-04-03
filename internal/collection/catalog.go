//go:build devtools

package collection

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

const (
	FormatAuto     = "auto"
	FormatMarkdown = "markdown"
	FormatText     = "text"
	FormatHTML     = "html"
)

type Catalog struct {
	Sources []Source `toml:"sources"`
}

type Source struct {
	ID      string `toml:"id"`
	Name    string `toml:"name"`
	Product string `toml:"product"`
	URL     string `toml:"url"`
	Format  string `toml:"format"`
}

func LoadCatalog(path string) (Catalog, error) {
	var catalog Catalog
	meta, err := toml.DecodeFile(path, &catalog)
	if err != nil {
		return Catalog{}, fmt.Errorf("decode collection catalog %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return Catalog{}, fmt.Errorf("decode collection catalog %s: unsupported keys: %s", path, joinKeys(undecoded))
	}
	if err := catalog.Validate(); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func WriteCatalog(path string, catalog Catalog) error {
	if err := catalog.Validate(); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create collection catalog: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(catalog); err != nil {
		return fmt.Errorf("encode collection catalog: %w", err)
	}
	return nil
}

func (c Catalog) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("collection catalog must define at least one source")
	}

	seen := make(map[string]struct{}, len(c.Sources))
	for _, source := range c.Sources {
		normalized, err := source.normalized()
		if err != nil {
			return err
		}
		if _, ok := seen[normalized.ID]; ok {
			return fmt.Errorf("collection catalog source id %q must be unique", normalized.ID)
		}
		seen[normalized.ID] = struct{}{}
	}

	return nil
}

func (s Source) normalized() (Source, error) {
	source := s
	source.Name = strings.TrimSpace(source.Name)
	source.Product = strings.TrimSpace(source.Product)
	source.URL = strings.TrimSpace(source.URL)

	if source.Name == "" && source.Product != "" {
		source.Name = source.Product
	}
	if source.Name == "" {
		return Source{}, fmt.Errorf("collection source name is required")
	}
	if source.URL == "" {
		return Source{}, fmt.Errorf("collection source %q url is required", source.Name)
	}

	source.ID = strings.TrimSpace(source.ID)
	if source.ID == "" {
		source.ID = slugify(source.Name)
	}

	format := strings.TrimSpace(strings.ToLower(source.Format))
	if format == "" {
		format = FormatAuto
	}
	switch format {
	case FormatAuto, FormatMarkdown, FormatText, FormatHTML:
	default:
		return Source{}, fmt.Errorf("collection source %q has unsupported format %q", source.Name, source.Format)
	}
	source.Format = format

	return source, nil
}

func (c Catalog) normalizedSources() ([]Source, error) {
	out := make([]Source, 0, len(c.Sources))
	for _, source := range c.Sources {
		normalized, err := source.normalized()
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func slugify(raw string) string {
	var out []rune
	lastDash := false
	for _, r := range strings.ToLower(raw) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			out = append(out, r)
			lastDash = false
		case !lastDash:
			out = append(out, '-')
			lastDash = true
		}
	}

	slug := strings.Trim(string(out), "-")
	if slug == "" {
		return "source"
	}
	return slug
}

func joinKeys(keys []toml.Key) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key.String())
	}
	return strings.Join(parts, ", ")
}
