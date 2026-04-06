package fragments

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
)

func TestGenerateID(t *testing.T) {
	id, err := GenerateID(bytes.NewReader([]byte{0, 1, 2}))
	if err != nil {
		t.Fatalf("GenerateID returned error: %v", err)
	}

	const want = "amber-beacon-carries"
	if id != want {
		t.Fatalf("GenerateID = %q, want %q", id, want)
	}
}

func TestNormalizeNameStem(t *testing.T) {
	stem, err := NormalizeNameStem(" Did Something Cool! ")
	if err != nil {
		t.Fatalf("NormalizeNameStem returned error: %v", err)
	}
	if stem != "did-something-cool" {
		t.Fatalf("NormalizeNameStem = %q, want %q", stem, "did-something-cool")
	}
}

func TestNormalizeNameStemRejectsEmptySlug(t *testing.T) {
	if _, err := NormalizeNameStem("!!!"); err == nil {
		t.Fatalf("NormalizeNameStem should reject non-alphanumeric stems")
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	item := Fragment{
		Metadata: Metadata{
			ID:                   "amber-beacon-carries",
			CreatedAt:            time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC),
			Type:                 "fixed",
			PublicAPI:            "change",
			Behavior:             "fix",
			Dependency:           "refresh",
			Runtime:              "expand",
			Scopes:               []string{"release"},
			SectionKey:           "fixes",
			Area:                 "rendering",
			Platforms:            []string{"cli"},
			Audiences:            []string{"engineers"},
			CustomerVisible:      true,
			SupportRelevance:     true,
			RequiresAction:       true,
			ReleaseNotesPriority: 2,
			DisplayOrder:         1,
		},
		Body: "Render whole entries only.",
	}

	raw := item.Format()
	parsed, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if parsed.ID != "" || parsed.Body != item.Body {
		t.Fatalf("Parse(Format()) mismatch: got %#v want %#v", parsed, item)
	}
	if parsed.SectionKey != item.SectionKey || parsed.Area != item.Area || parsed.DisplayOrder != item.DisplayOrder {
		t.Fatalf("Parse(Format()) metadata mismatch: got %#v want %#v", parsed.Metadata, item.Metadata)
	}
	if parsed.PublicAPI != item.PublicAPI || parsed.Behavior != item.Behavior || parsed.Dependency != item.Dependency || parsed.Runtime != item.Runtime {
		t.Fatalf("Parse(Format()) semantic metadata mismatch: got %#v want %#v", parsed.Metadata, item.Metadata)
	}
	if strings.Contains(raw, "title = ") || strings.Contains(raw, "id = ") || strings.Contains(raw, "created_at = ") {
		t.Fatalf("formatted fragment should omit derived metadata: %s", raw)
	}
}

func TestCreateDoesNotOverwriteExistingFragmentOnCollision(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)

	dir := config.FragmentsDir(repoRoot, cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir fragments dir: %v", err)
	}
	existingPath := filepath.Join(dir, "20260406-120000--sample--amber-beacon-carries.md")
	original := "existing fragment contents"
	if err := os.WriteFile(existingPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write existing fragment: %v", err)
	}

	item, err := Create(repoRoot, cfg, now, bytes.NewReader([]byte{0, 1, 2, 3, 4, 5}), NewInput{
		NameStem: "sample",
		Behavior: "fix",
		Body:     "Fix a release-visible bug.",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if item.Path == existingPath {
		t.Fatalf("Create reused colliding path %s", item.Path)
	}

	raw, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("read original fragment: %v", err)
	}
	if string(raw) != original {
		t.Fatalf("existing fragment was overwritten: got %q want %q", string(raw), original)
	}
}

func TestLoadDerivesIDAndTimestampAndNormalizesFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample-fragment.md")
	modTime := time.Date(2026, 4, 6, 15, 4, 5, 0, time.UTC)
	raw := []byte("+++\n" +
		"type = \" FIXED \"\n" +
		"public_api = \" ADD \"\n" +
		"behavior = \" FIX \"\n" +
		"dependency = \" REFRESH \"\n" +
		"runtime = \" EXPAND \"\n" +
		"+++\n\n" +
		"  First line.\n\nMore detail.\n")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write fragment: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("chtimes fragment: %v", err)
	}

	item, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if item.ID != "sample-fragment" {
		t.Fatalf("ID = %q, want sample-fragment", item.ID)
	}
	if !item.CreatedAt.Equal(modTime) {
		t.Fatalf("CreatedAt = %s, want %s", item.CreatedAt, modTime)
	}
	if item.Type != "fixed" || item.PublicAPI != "add" || item.Behavior != "fix" ||
		item.Dependency != "refresh" || item.Runtime != "expand" {
		t.Fatalf("normalized fields = %#v", item.Metadata)
	}
	if item.Body != "First line.\n\nMore detail." {
		t.Fatalf("Body = %q", item.Body)
	}
}

func TestListReturnsSortedMarkdownFragmentsOnly(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	dir := config.FragmentsDir(repoRoot, cfg)
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir fragments dir: %v", err)
	}

	first := []byte("+++\ncreated_at = 2026-04-06T12:00:00Z\n+++\n\nFirst body.\n")
	second := []byte("+++\ncreated_at = 2026-04-06T11:00:00Z\n+++\n\nSecond body.\n")
	if err := os.WriteFile(filepath.Join(dir, "b.md"), first, 0o644); err != nil {
		t.Fatalf("write first fragment: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), second, 0o644); err != nil {
		t.Fatalf("write second fragment: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatalf("write non-markdown file: %v", err)
	}

	items, err := List(repoRoot, cfg)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.ID)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("List IDs = %#v, want a,b", got)
	}
}

func TestBodyPreviewUsesFirstNonEmptyLine(t *testing.T) {
	item := Fragment{Body: "\n  First   line   here. \n\nSecond line.\n"}
	if got := item.BodyPreview(); got != "First line here." {
		t.Fatalf("BodyPreview = %q, want %q", got, "First line here.")
	}

	if got := (Fragment{}).BodyPreview(); got != "" {
		t.Fatalf("empty BodyPreview = %q, want empty", got)
	}
}
