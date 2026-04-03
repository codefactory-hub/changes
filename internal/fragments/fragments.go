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

	"github.com/BurntSushi/toml"
	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/versioning"
)

var (
	idAdjectives = []string{
		"amber", "brave", "calm", "daring", "eager", "fancy", "gentle", "hidden",
		"icy", "jolly", "keen", "lucky", "mellow", "navy", "olive", "proud",
	}
	idNouns = []string{
		"acorn", "beacon", "comet", "delta", "ember", "forest", "garden", "harbor",
		"island", "jungle", "king", "lantern", "meadow", "nectar", "ocean", "prairie",
	}
	idVerbs = []string{
		"adapts", "builds", "carries", "drifts", "echoes", "flows", "guides", "hovers",
		"improves", "jumps", "keeps", "lifts", "moves", "nudges", "opens", "protects",
	}
)

type Metadata struct {
	ID                   string    `toml:"id"`
	CreatedAt            time.Time `toml:"created_at"`
	Title                string    `toml:"title"`
	Type                 string    `toml:"type"`
	Bump                 string    `toml:"bump"`
	Breaking             bool      `toml:"breaking"`
	Scopes               []string  `toml:"scopes"`
	Authors              []string  `toml:"authors"`
	SectionKey           string    `toml:"section_key"`
	Area                 string    `toml:"area"`
	Platforms            []string  `toml:"platforms"`
	Audiences            []string  `toml:"audiences"`
	CustomerVisible      bool      `toml:"customer_visible"`
	SupportRelevance     bool      `toml:"support_relevance"`
	RequiresAction       bool      `toml:"requires_action"`
	ReleaseNotesPriority int       `toml:"release_notes_priority"`
	DisplayOrder         int       `toml:"display_order"`
}

type Fragment struct {
	Metadata
	Body string
	Path string
}

type NewInput struct {
	Type                 string
	Bump                 versioning.Bump
	Breaking             bool
	Scopes               []string
	Authors              []string
	SectionKey           string
	Area                 string
	Platforms            []string
	Audiences            []string
	CustomerVisible      bool
	SupportRelevance     bool
	RequiresAction       bool
	ReleaseNotesPriority int
	DisplayOrder         int
	Body                 string
}

func Create(repoRoot string, cfg config.Config, now time.Time, random io.Reader, input NewInput) (Fragment, error) {
	if random == nil {
		random = rand.Reader
	}

	item := Fragment{
		Metadata: Metadata{
			CreatedAt:            now.UTC().Truncate(time.Second),
			Type:                 normalizeType(input.Type),
			Bump:                 string(input.Bump),
			Breaking:             input.Breaking,
			Scopes:               slices.Clone(input.Scopes),
			Authors:              slices.Clone(input.Authors),
			SectionKey:           strings.TrimSpace(input.SectionKey),
			Area:                 strings.TrimSpace(input.Area),
			Platforms:            slices.Clone(input.Platforms),
			Audiences:            slices.Clone(input.Audiences),
			CustomerVisible:      input.CustomerVisible,
			SupportRelevance:     input.SupportRelevance,
			RequiresAction:       input.RequiresAction,
			ReleaseNotesPriority: input.ReleaseNotesPriority,
			DisplayOrder:         input.DisplayOrder,
		},
		Body: strings.TrimSpace(input.Body),
	}

	if err := item.Validate(); err != nil {
		return Fragment{}, err
	}

	dir := config.FragmentsDir(repoRoot, cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Fragment{}, fmt.Errorf("create fragments directory: %w", err)
	}

	for attempts := 0; attempts < 8; attempts++ {
		id, err := GenerateID(random)
		if err != nil {
			return Fragment{}, err
		}
		item.ID = id
		item.Path = filepath.Join(dir, id+".md")
		if _, err := os.Stat(item.Path); os.IsNotExist(err) {
			break
		} else if err != nil {
			return Fragment{}, fmt.Errorf("stat fragment path: %w", err)
		}
	}
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Path) == "" {
		return Fragment{}, fmt.Errorf("generate fragment id: unable to allocate unique path")
	}

	if err := os.WriteFile(item.Path, []byte(item.Format()), 0o644); err != nil {
		return Fragment{}, fmt.Errorf("write fragment: %w", err)
	}
	if err := os.Chtimes(item.Path, item.CreatedAt, item.CreatedAt); err != nil {
		return Fragment{}, fmt.Errorf("set fragment timestamp: %w", err)
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
	if strings.TrimSpace(item.ID) == "" {
		item.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if item.CreatedAt.IsZero() {
		info, err := os.Stat(path)
		if err != nil {
			return Fragment{}, fmt.Errorf("stat fragment %s: %w", path, err)
		}
		item.CreatedAt = info.ModTime().UTC().Truncate(time.Second)
	}
	item.Type = normalizeType(item.Type)
	item.Body = strings.TrimSpace(item.Body)
	if err := item.Validate(); err != nil {
		return Fragment{}, fmt.Errorf("validate fragment %s: %w", path, err)
	}

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
	return item, nil
}

func (f Fragment) Validate() error {
	if strings.TrimSpace(f.Body) == "" {
		return fmt.Errorf("fragment body is required")
	}
	if f.ReleaseNotesPriority < 0 {
		return fmt.Errorf("fragment release_notes_priority must be >= 0")
	}
	if f.DisplayOrder < 0 {
		return fmt.Errorf("fragment display_order must be >= 0")
	}
	if _, err := versioning.NormalizeBump(f.Bump); err != nil {
		return err
	}
	return nil
}

func (f Fragment) Format() string {
	var buf bytes.Buffer
	buf.WriteString("+++\n")

	var metadata struct {
		Type                 string   `toml:"type"`
		Bump                 string   `toml:"bump"`
		Breaking             bool     `toml:"breaking,omitempty"`
		Scopes               []string `toml:"scopes,omitempty"`
		Authors              []string `toml:"authors,omitempty"`
		SectionKey           string   `toml:"section_key,omitempty"`
		Area                 string   `toml:"area,omitempty"`
		Platforms            []string `toml:"platforms,omitempty"`
		Audiences            []string `toml:"audiences,omitempty"`
		CustomerVisible      bool     `toml:"customer_visible,omitempty"`
		SupportRelevance     bool     `toml:"support_relevance,omitempty"`
		RequiresAction       bool     `toml:"requires_action,omitempty"`
		ReleaseNotesPriority int      `toml:"release_notes_priority,omitempty"`
		DisplayOrder         int      `toml:"display_order,omitempty"`
	}
	metadata.Type = normalizeType(f.Type)
	metadata.Bump = strings.TrimSpace(f.Bump)
	metadata.Breaking = f.Breaking
	metadata.Scopes = slices.Clone(f.Scopes)
	metadata.Authors = slices.Clone(f.Authors)
	metadata.SectionKey = strings.TrimSpace(f.SectionKey)
	metadata.Area = strings.TrimSpace(f.Area)
	metadata.Platforms = slices.Clone(f.Platforms)
	metadata.Audiences = slices.Clone(f.Audiences)
	metadata.CustomerVisible = f.CustomerVisible
	metadata.SupportRelevance = f.SupportRelevance
	metadata.RequiresAction = f.RequiresAction
	metadata.ReleaseNotesPriority = f.ReleaseNotesPriority
	metadata.DisplayOrder = f.DisplayOrder

	_ = toml.NewEncoder(&buf).Encode(metadata)
	buf.WriteString("+++\n\n")
	buf.WriteString(strings.TrimSpace(f.Body))
	buf.WriteString("\n")
	return buf.String()
}

func GenerateID(random io.Reader) (string, error) {
	if random == nil {
		random = rand.Reader
	}

	bytes := make([]byte, 3)
	if _, err := io.ReadFull(random, bytes); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}

	return fmt.Sprintf(
		"%s-%s-%s",
		idAdjectives[int(bytes[0])%len(idAdjectives)],
		idNouns[int(bytes[1])%len(idNouns)],
		idVerbs[int(bytes[2])%len(idVerbs)],
	), nil
}

func normalizeType(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return "changed"
	}
	return value
}

func (f Fragment) BodyPreview() string {
	body := strings.TrimSpace(f.Body)
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	preview := strings.TrimSpace(lines[0])
	if preview == "" {
		return ""
	}
	return strings.Join(strings.Fields(preview), " ")
}
