package fragments

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/versioning"
)

const suffixAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

type Metadata struct {
	ID        string    `toml:"id"`
	CreatedAt time.Time `toml:"created_at"`
	Title     string    `toml:"title"`
	Type      string    `toml:"type"`
	Bump      string    `toml:"bump"`
	Breaking  bool      `toml:"breaking"`
	Scopes    []string  `toml:"scopes"`
	Authors   []string  `toml:"authors"`
}

type Fragment struct {
	Metadata
	Body string
	Path string
}

type NewInput struct {
	Title    string
	Type     string
	Bump     versioning.Bump
	Breaking bool
	Scopes   []string
	Authors  []string
	Body     string
}

func Create(repoRoot string, cfg config.Config, now time.Time, random io.Reader, input NewInput) (Fragment, error) {
	if random == nil {
		random = rand.Reader
	}

	id, err := GenerateID(now, input.Title, random)
	if err != nil {
		return Fragment{}, err
	}

	item := Fragment{
		Metadata: Metadata{
			ID:        id,
			CreatedAt: now.UTC().Truncate(time.Second),
			Title:     strings.TrimSpace(input.Title),
			Type:      normalizeType(input.Type),
			Bump:      string(input.Bump),
			Breaking:  input.Breaking,
			Scopes:    slices.Clone(input.Scopes),
			Authors:   slices.Clone(input.Authors),
		},
		Body: strings.TrimSpace(input.Body),
		Path: filepath.Join(config.FragmentsDir(repoRoot, cfg), id+".md"),
	}

	if err := item.Validate(); err != nil {
		return Fragment{}, err
	}

	if err := os.MkdirAll(filepath.Dir(item.Path), 0o755); err != nil {
		return Fragment{}, fmt.Errorf("create fragments directory: %w", err)
	}

	if err := os.WriteFile(item.Path, []byte(item.Format()), 0o644); err != nil {
		return Fragment{}, fmt.Errorf("write fragment: %w", err)
	}

	return item, nil
}

func List(repoRoot string, cfg config.Config) ([]Fragment, error) {
	dir := config.FragmentsDir(repoRoot, cfg)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read fragments dir: %w", err)
	}

	frags := make([]Fragment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		item, err := Load(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		frags = append(frags, item)
	}

	slices.SortFunc(frags, func(a, b Fragment) int {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	return frags, nil
}

func Load(path string) (Fragment, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Fragment{}, fmt.Errorf("read fragment %s: %w", path, err)
	}

	item, err := Parse(raw)
	if err != nil {
		return Fragment{}, fmt.Errorf("parse fragment %s: %w", path, err)
	}
	item.Path = path

	return item, nil
}

func Parse(raw []byte) (Fragment, error) {
	content := string(raw)
	if !strings.HasPrefix(content, "+++\n") {
		return Fragment{}, fmt.Errorf("fragment missing TOML front matter")
	}

	rest := strings.TrimPrefix(content, "+++\n")
	idx := strings.Index(rest, "\n+++\n")
	if idx < 0 {
		return Fragment{}, fmt.Errorf("fragment front matter terminator not found")
	}

	metaText := rest[:idx]
	body := strings.TrimLeft(rest[idx+5:], "\n")

	var meta Metadata
	if _, err := toml.Decode(metaText, &meta); err != nil {
		return Fragment{}, fmt.Errorf("decode front matter: %w", err)
	}

	item := Fragment{
		Metadata: meta,
		Body:     strings.TrimSpace(body),
	}
	if err := item.Validate(); err != nil {
		return Fragment{}, err
	}
	return item, nil
}

func (f Fragment) Validate() error {
	if strings.TrimSpace(f.ID) == "" {
		return fmt.Errorf("fragment id is required")
	}
	if strings.TrimSpace(f.Title) == "" {
		return fmt.Errorf("fragment title is required")
	}
	if strings.TrimSpace(f.Body) == "" {
		return fmt.Errorf("fragment body is required")
	}
	if f.CreatedAt.IsZero() {
		return fmt.Errorf("fragment created_at is required")
	}
	if _, err := versioning.NormalizeBump(f.Bump); err != nil {
		return err
	}
	return nil
}

func (f Fragment) Format() string {
	var buf bytes.Buffer
	buf.WriteString("+++\n")
	_ = toml.NewEncoder(&buf).Encode(f.Metadata)
	buf.WriteString("+++\n\n")
	buf.WriteString(strings.TrimSpace(f.Body))
	buf.WriteString("\n")
	return buf.String()
}

func GenerateID(now time.Time, title string, random io.Reader) (string, error) {
	if random == nil {
		random = rand.Reader
	}

	suffix, err := randomSuffix(random, 4)
	if err != nil {
		return "", fmt.Errorf("generate suffix: %w", err)
	}

	timestamp := now.UTC().Truncate(time.Second).Format("20060102T150405Z")
	slug := slugify(title)

	return fmt.Sprintf("%s--%s--%s", timestamp, slug, suffix), nil
}

func normalizeType(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return "changed"
	}
	return value
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
		return "change"
	}
	return slug
}

func randomSuffix(r io.Reader, length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(r, bytes); err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.Grow(length)
	for _, b := range bytes {
		builder.WriteByte(suffixAlphabet[int(b)%len(suffixAlphabet)])
	}

	return builder.String(), nil
}
