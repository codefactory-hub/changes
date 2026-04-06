package versioning

import "testing"

func TestParseAndStringRoundTrip(t *testing.T) {
	version, err := Parse("1.2.3-rc.4+build.5")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if version.Major != 1 || version.Minor != 2 || version.Patch != 3 {
		t.Fatalf("unexpected numeric parts: %#v", version)
	}
	if version.Prerelease != "rc.4" || version.BuildMetadata != "build.5" {
		t.Fatalf("unexpected prerelease metadata: %#v", version)
	}
	if got := version.String(); got != "1.2.3-rc.4+build.5" {
		t.Fatalf("String = %q, want 1.2.3-rc.4+build.5", got)
	}
}

func TestParseRejectsEmptyOrInvalidVersions(t *testing.T) {
	for _, raw := range []string{"", " ", "1.2", "v1.2.3"} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("Parse(%q) returned nil error", raw)
		}
	}
}

func TestMustParsePanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("MustParse should panic for invalid input")
		}
	}()

	_ = MustParse("not-a-version")
}

func TestStableWithoutBuildMetadataAndIsPrerelease(t *testing.T) {
	version := MustParse("2.3.4-rc.2+build.9")

	if !version.IsPrerelease() {
		t.Fatalf("IsPrerelease = false, want true")
	}

	stable := version.Stable()
	if got := stable.String(); got != "2.3.4" {
		t.Fatalf("Stable = %q, want 2.3.4", got)
	}

	withoutMetadata := version.WithoutBuildMetadata()
	if got := withoutMetadata.String(); got != "2.3.4-rc.2" {
		t.Fatalf("WithoutBuildMetadata = %q, want 2.3.4-rc.2", got)
	}
}

func TestPrereleaseLabelNumber(t *testing.T) {
	label, number, ok := MustParse("1.2.3-rc.7").PrereleaseLabelNumber()
	if !ok || label != "rc" || number != 7 {
		t.Fatalf("PrereleaseLabelNumber = (%q, %d, %v), want (rc, 7, true)", label, number, ok)
	}

	for _, raw := range []string{"1.2.3", "1.2.3-rc", "1.2.3-rc.x", "1.2.3-alpha.beta.gamma"} {
		if _, _, ok := MustParse(raw).PrereleaseLabelNumber(); ok {
			t.Fatalf("PrereleaseLabelNumber(%q) = ok, want false", raw)
		}
	}
}

func TestCompareHonorsSemverOrdering(t *testing.T) {
	if got := Compare(MustParse("1.2.3"), MustParse("1.2.3")); got != 0 {
		t.Fatalf("Compare equal = %d, want 0", got)
	}
	if got := Compare(MustParse("1.2.3-rc.1"), MustParse("1.2.3")); got >= 0 {
		t.Fatalf("Compare prerelease vs stable = %d, want < 0", got)
	}
	if got := Compare(MustParse("1.2.4"), MustParse("1.2.3+build.9")); got <= 0 {
		t.Fatalf("Compare newer patch = %d, want > 0", got)
	}
}

func TestHighestBumpAndNormalizeBump(t *testing.T) {
	if got := HighestBump(BumpPatch, BumpNone, BumpMinor, BumpMajor); got != BumpMajor {
		t.Fatalf("HighestBump = %q, want major", got)
	}

	for raw, want := range map[string]Bump{
		"":      BumpNone,
		"none":  BumpNone,
		"PATCH": BumpPatch,
		"minor": BumpMinor,
		" major ": BumpMajor,
	} {
		got, err := NormalizeBump(raw)
		if err != nil {
			t.Fatalf("NormalizeBump(%q) returned error: %v", raw, err)
		}
		if got != want {
			t.Fatalf("NormalizeBump(%q) = %q, want %q", raw, got, want)
		}
	}

	if _, err := NormalizeBump("invalid"); err == nil {
		t.Fatalf("NormalizeBump(invalid) returned nil error")
	}
}

func TestNextStable(t *testing.T) {
	initial := MustParse("0.1.0-rc.1")
	if got := NextStable(nil, initial, BumpMinor).String(); got != "0.1.0" {
		t.Fatalf("NextStable(nil) = %q, want 0.1.0", got)
	}

	latest := MustParse("1.2.3-rc.4+build.9")
	if got := NextStable(&latest, initial, BumpNone).String(); got != "1.2.3" {
		t.Fatalf("NextStable(none) = %q, want 1.2.3", got)
	}
	if got := NextStable(&latest, initial, BumpPatch).String(); got != "1.2.4" {
		t.Fatalf("NextStable(patch) = %q, want 1.2.4", got)
	}
	if got := NextStable(&latest, initial, BumpMinor).String(); got != "1.3.0" {
		t.Fatalf("NextStable(minor) = %q, want 1.3.0", got)
	}
	if got := NextStable(&latest, initial, BumpMajor).String(); got != "2.0.0" {
		t.Fatalf("NextStable(major) = %q, want 2.0.0", got)
	}
}

func TestNextPrereleaseUsesHighestExistingNumberForTargetAndLabel(t *testing.T) {
	target := MustParse("2.0.0")
	current := []Version{
		MustParse("2.0.0-rc.1"),
		MustParse("2.0.0-beta.9"),
		MustParse("2.0.0-rc.3+build.7"),
		MustParse("1.9.9-rc.99"),
	}

	got := NextPrerelease(target, "rc", current)
	if got.String() != "2.0.0-rc.4" {
		t.Fatalf("NextPrerelease = %q, want 2.0.0-rc.4", got.String())
	}
}
