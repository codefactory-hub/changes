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

func TestEvaluateAggregatesHighestBumpAndReasons(t *testing.T) {
	pending := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", PublicAPI: "add", Behavior: "fix"}},
		{Metadata: fragments.Metadata{ID: "f2", Dependency: "restrict", Runtime: "reduce"}},
		{Metadata: fragments.Metadata{ID: "f3", Dependency: "refresh"}},
	}

	got := Evaluate(StabilityStable, pending)

	if got.SuggestedBump != versioning.BumpMajor {
		t.Fatalf("suggested bump = %q, want major", got.SuggestedBump)
	}
	if len(got.Assessments) != 3 {
		t.Fatalf("assessments = %d, want 3", len(got.Assessments))
	}
	if got.Assessments[0].SuggestedBump != versioning.BumpMinor {
		t.Fatalf("assessment[0] bump = %q, want minor", got.Assessments[0].SuggestedBump)
	}
	if got.Assessments[1].SuggestedBump != versioning.BumpMajor {
		t.Fatalf("assessment[1] bump = %q, want major", got.Assessments[1].SuggestedBump)
	}
	if got.Assessments[2].SuggestedBump != versioning.BumpNone || len(got.Assessments[2].Reasons) != 2 {
		t.Fatalf("assessment[2] = %#v, want refresh note plus fallback note", got.Assessments[2])
	}
}

func TestEvaluateUsesUnstableBreakingPolicyAcrossSemanticLevers(t *testing.T) {
	pending := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", PublicAPI: "remove"}},
		{Metadata: fragments.Metadata{ID: "f2", Behavior: "redefine"}},
		{Metadata: fragments.Metadata{ID: "f3", Dependency: "restrict"}},
		{Metadata: fragments.Metadata{ID: "f4", Runtime: "reduce"}},
	}

	got := Evaluate(StabilityUnstable, pending)

	for _, assessment := range got.Assessments {
		if assessment.SuggestedBump != versioning.BumpMinor {
			t.Fatalf("assessment %#v should use minor under unstable policy", assessment)
		}
		if len(assessment.Reasons) == 0 {
			t.Fatalf("assessment %#v missing reasons", assessment)
		}
	}
}
