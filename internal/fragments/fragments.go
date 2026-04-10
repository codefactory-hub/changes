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
	Bootstrap            bool      `toml:"bootstrap"`
	PublicAPI            string    `toml:"public_api"`
	Behavior             string    `toml:"behavior"`
	Dependency           string    `toml:"dependency"`
	Runtime              string    `toml:"runtime"`
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
	NameStem             string
	Type                 string
	Bootstrap            bool
	PublicAPI            string
	Behavior             string
	Dependency           string
	Runtime              string
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
			Bootstrap:            input.Bootstrap,
			PublicAPI:            normalizePublicAPI(input.PublicAPI),
			Behavior:             normalizeBehavior(input.Behavior),
			Dependency:           normalizeDependency(input.Dependency),
			Runtime:              normalizeRuntime(input.Runtime),
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
	nameStem, err := NormalizeNameStem(input.NameStem)
	if err != nil {
		return Fragment{}, err
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
		if nameStem != "" {
			id = fmt.Sprintf("%s--%s--%s", item.CreatedAt.Format("20060102-150405"), nameStem, id)
		}
		item.ID = id
		item.Path = filepath.Join(dir, id+".md")
		file, err := os.OpenFile(item.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			item.ID = ""
			item.Path = ""
			continue
		}
		if err != nil {
			return Fragment{}, fmt.Errorf("create fragment: %w", err)
		}
		if _, err := io.WriteString(file, item.Format()); err != nil {
			_ = file.Close()
			_ = os.Remove(item.Path)
			return Fragment{}, fmt.Errorf("write fragment: %w", err)
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(item.Path)
			return Fragment{}, fmt.Errorf("close fragment: %w", err)
		}
		break
	}
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Path) == "" {
		return Fragment{}, fmt.Errorf("generate fragment id: unable to allocate unique path")
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
	item.PublicAPI = normalizePublicAPI(item.PublicAPI)
	item.Behavior = normalizeBehavior(item.Behavior)
	item.Dependency = normalizeDependency(item.Dependency)
	item.Runtime = normalizeRuntime(item.Runtime)
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
	if err := validateEnum("public_api", f.PublicAPI, "add", "change", "remove"); err != nil {
		return err
	}
	if err := validateEnum("behavior", f.Behavior, "new", "fix", "redefine"); err != nil {
		return err
	}
	if err := validateEnum("dependency", f.Dependency, "refresh", "relax", "restrict"); err != nil {
		return err
	}
	if err := validateEnum("runtime", f.Runtime, "expand", "reduce"); err != nil {
		return err
	}
	return nil
}

func (f Fragment) Format() string {
	var buf bytes.Buffer
	buf.WriteString("+++\n")

	var metadata struct {
		Type                 string   `toml:"type"`
		Bootstrap            bool     `toml:"bootstrap,omitempty"`
		PublicAPI            string   `toml:"public_api,omitempty"`
		Behavior             string   `toml:"behavior,omitempty"`
		Dependency           string   `toml:"dependency,omitempty"`
		Runtime              string   `toml:"runtime,omitempty"`
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
		ReleaseNotesPriority *int     `toml:"release_notes_priority,omitempty"`
		DisplayOrder         *int     `toml:"display_order,omitempty"`
	}
	metadata.Type = normalizeType(f.Type)
	metadata.Bootstrap = f.Bootstrap
	metadata.PublicAPI = normalizePublicAPI(f.PublicAPI)
	metadata.Behavior = normalizeBehavior(f.Behavior)
	metadata.Dependency = normalizeDependency(f.Dependency)
	metadata.Runtime = normalizeRuntime(f.Runtime)
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
	if f.ReleaseNotesPriority > 0 {
		metadata.ReleaseNotesPriority = &f.ReleaseNotesPriority
	}
	if f.DisplayOrder > 0 {
		metadata.DisplayOrder = &f.DisplayOrder
	}

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

func NormalizeNameStem(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}

	var parts []string
	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		parts = append(parts, field)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("fragment name stem must contain letters or digits")
	}
	return strings.Join(parts, "-"), nil
}

func normalizeType(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return "changed"
	}
	return value
}

func normalizePublicAPI(raw string) string {
	return normalizeOptionalValue(raw)
}

func normalizeBehavior(raw string) string {
	return normalizeOptionalValue(raw)
}

func normalizeDependency(raw string) string {
	return normalizeOptionalValue(raw)
}

func normalizeRuntime(raw string) string {
	return normalizeOptionalValue(raw)
}

func normalizeOptionalValue(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

func validateEnum(field, value string, allowed ...string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return fmt.Errorf("fragment %s must be one of %s", field, strings.Join(allowed, ", "))
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
