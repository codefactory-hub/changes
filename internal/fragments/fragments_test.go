package fragments

import (
	"bytes"
	"strings"
	"testing"
	"time"
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

func TestFormatParseRoundTrip(t *testing.T) {
	item := Fragment{
		Metadata: Metadata{
			ID:                   "amber-beacon-carries",
			CreatedAt:            time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC),
			Type:                 "fixed",
			Bump:                 "patch",
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
	if strings.Contains(raw, "title = ") || strings.Contains(raw, "id = ") || strings.Contains(raw, "created_at = ") {
		t.Fatalf("formatted fragment should omit derived metadata: %s", raw)
	}
}
