package semverpolicy

import (
	"fmt"
	"strings"

	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/versioning"
)

type Stability string

const (
	StabilityStable   Stability = "stable"
	StabilityUnstable Stability = "unstable"
)

type FragmentAssessment struct {
	FragmentID    string
	DeclaredBump  versioning.Bump
	SuggestedBump versioning.Bump
	Reasons       []string
}

type Recommendation struct {
	Stability     Stability
	DeclaredBump  versioning.Bump
	SuggestedBump versioning.Bump
	Assessments   []FragmentAssessment
}

func Evaluate(stability Stability, pending []fragments.Fragment) Recommendation {
	out := Recommendation{
		Stability:    stability,
		DeclaredBump: versioning.BumpNone,
		Assessments:  make([]FragmentAssessment, 0, len(pending)),
	}

	for _, item := range pending {
		assessment := assessFragment(stability, item)
		out.Assessments = append(out.Assessments, assessment)
		out.DeclaredBump = versioning.HighestBump(out.DeclaredBump, assessment.DeclaredBump)
		out.SuggestedBump = versioning.HighestBump(out.SuggestedBump, assessment.SuggestedBump)
	}
	return out
}

func assessFragment(stability Stability, item fragments.Fragment) FragmentAssessment {
	declared, err := versioning.NormalizeBump(item.Bump)
	if err != nil {
		declared = versioning.BumpNone
	}

	out := FragmentAssessment{
		FragmentID:    item.ID,
		DeclaredBump:  declared,
		SuggestedBump: versioning.BumpNone,
	}

	add := func(bump versioning.Bump, reason string) {
		out.SuggestedBump = versioning.HighestBump(out.SuggestedBump, bump)
		out.Reasons = append(out.Reasons, reason)
	}
	breakingBump := func() versioning.Bump {
		if stability == StabilityStable {
			return versioning.BumpMajor
		}
		return versioning.BumpMinor
	}

	switch strings.TrimSpace(item.PublicAPI) {
	case "add":
		add(versioning.BumpMinor, `public_api=add implies an additive release signal`)
	case "change":
		add(breakingBump(), fmt.Sprintf("public_api=change implies %s while public API policy is %s", breakingBump(), stability))
	case "remove":
		add(breakingBump(), fmt.Sprintf("public_api=remove implies %s while public API policy is %s", breakingBump(), stability))
	}

	switch strings.TrimSpace(item.Behavior) {
	case "new":
		add(versioning.BumpMinor, `behavior=new implies additive observable behavior`)
	case "fix":
		add(versioning.BumpPatch, `behavior=fix implies a patch-level correction`)
	case "redefine":
		add(breakingBump(), fmt.Sprintf("behavior=redefine implies %s while public API policy is %s", breakingBump(), stability))
	}

	switch strings.TrimSpace(item.Dependency) {
	case "refresh":
		out.Reasons = append(out.Reasons, `dependency=refresh changes selected versions without changing declared compatibility`)
	case "relax":
		add(versioning.BumpMinor, `dependency=relax broadens declared compatibility`)
	case "restrict":
		add(breakingBump(), fmt.Sprintf("dependency=restrict implies %s while public API policy is %s", breakingBump(), stability))
	}

	switch strings.TrimSpace(item.Runtime) {
	case "expand":
		add(versioning.BumpMinor, `runtime=expand broadens declared support`)
	case "reduce":
		add(breakingBump(), fmt.Sprintf("runtime=reduce implies %s while public API policy is %s", breakingBump(), stability))
	}

	if out.SuggestedBump == versioning.BumpNone && declared != versioning.BumpNone {
		out.SuggestedBump = declared
		out.Reasons = append(out.Reasons, fmt.Sprintf("no semantic levers present; fall back to declared bump=%s", declared))
	}

	return out
}
