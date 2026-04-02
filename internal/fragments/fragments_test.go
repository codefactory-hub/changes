package fragments

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	now := time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC)
	id, err := GenerateID(now, "Fix release note rendering", bytes.NewReader([]byte{0, 1, 2, 3}))
	if err != nil {
		t.Fatalf("GenerateID returned error: %v", err)
	}

	const want = "20260402T153045Z--fix-release-note-rendering--abcd"
	if id != want {
		t.Fatalf("GenerateID = %q, want %q", id, want)
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	item := Fragment{
		Metadata: Metadata{
			ID:        "20260402T153045Z--fix-release-note-rendering--abcd",
			CreatedAt: time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC),
			Title:     "Fix release note rendering",
			Type:      "fixed",
			Bump:      "patch",
			Scopes:    []string{"release"},
		},
		Body: "Render whole entries only.",
	}

	raw := item.Format()
	parsed, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if parsed.ID != item.ID || parsed.Title != item.Title || parsed.Body != item.Body {
		t.Fatalf("Parse(Format()) mismatch: got %#v want %#v", parsed, item)
	}
	if !strings.Contains(raw, "title = \"Fix release note rendering\"") {
		t.Fatalf("formatted fragment missing title: %s", raw)
	}
}
