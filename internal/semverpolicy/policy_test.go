package semverpolicy

import (
	"testing"

	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/versioning"
)

func TestEvaluateUsesStablePolicyAtMajorOneAndAbove(t *testing.T) {
	pending := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", PublicAPI: "change"}},
	}

	got := Evaluate(StabilityStable, pending)

	if got.Stability != StabilityStable {
		t.Fatalf("stability = %q, want %q", got.Stability, StabilityStable)
	}
	if got.SuggestedBump != versioning.BumpMajor {
		t.Fatalf("suggested bump = %q, want %q", got.SuggestedBump, versioning.BumpMajor)
	}
}

func TestEvaluateUsesUnstablePolicyBeforeMajorOne(t *testing.T) {
	pending := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", PublicAPI: "change"}},
	}

	got := Evaluate(StabilityUnstable, pending)

	if got.Stability != StabilityUnstable {
		t.Fatalf("stability = %q, want %q", got.Stability, StabilityUnstable)
	}
	if got.SuggestedBump != versioning.BumpMinor {
		t.Fatalf("suggested bump = %q, want %q", got.SuggestedBump, versioning.BumpMinor)
	}
}

func TestEvaluateInfersNoBumpWithoutSemanticLevers(t *testing.T) {
	pending := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1"}},
	}

	got := Evaluate(StabilityStable, pending)

	if got.SuggestedBump != versioning.BumpNone {
		t.Fatalf("suggested bump = %q, want %q", got.SuggestedBump, versioning.BumpNone)
	}
	if len(got.Assessments) != 1 || len(got.Assessments[0].Reasons) == 0 {
		t.Fatalf("expected fallback reason, got %#v", got.Assessments)
	}
}
