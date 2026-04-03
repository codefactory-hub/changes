package versioning

import (
	"fmt"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

type Bump string

const (
	BumpNone  Bump = "none"
	BumpPatch Bump = "patch"
	BumpMinor Bump = "minor"
	BumpMajor Bump = "major"
)

type Version struct {
	Major         int
	Minor         int
	Patch         int
	Prerelease    string
	BuildMetadata string
}

func Parse(raw string) (Version, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return Version{}, fmt.Errorf("parse version %q: version is required", raw)
	}

	parsed, err := semver.StrictNewVersion(value)
	if err != nil {
		return Version{}, fmt.Errorf("parse version %q: %w", raw, err)
	}

	return Version{
		Major:         int(parsed.Major()),
		Minor:         int(parsed.Minor()),
		Patch:         int(parsed.Patch()),
		Prerelease:    parsed.Prerelease(),
		BuildMetadata: parsed.Metadata(),
	}, nil
}

func MustParse(raw string) Version {
	version, err := Parse(raw)
	if err != nil {
		panic(err)
	}
	return version
}

func (v Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		base += "-" + v.Prerelease
	}
	if v.BuildMetadata != "" {
		base += "+" + v.BuildMetadata
	}
	return base
}

func (v Version) Stable() Version {
	v.Prerelease = ""
	v.BuildMetadata = ""
	return v
}

func (v Version) WithoutBuildMetadata() Version {
	v.BuildMetadata = ""
	return v
}

func (v Version) IsPrerelease() bool {
	return v.Prerelease != ""
}

func (v Version) PrereleaseLabelNumber() (string, int, bool) {
	if v.Prerelease == "" {
		return "", 0, false
	}

	parts := strings.Split(v.Prerelease, ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", 0, false
	}

	number, err := strconv.Atoi(parts[1])
	if err != nil || number < 0 {
		return "", 0, false
	}
	return parts[0], number, true
}

func Compare(a, b Version) int {
	left, err := semver.StrictNewVersion(a.String())
	if err != nil {
		panic(err)
	}
	right, err := semver.StrictNewVersion(b.String())
	if err != nil {
		panic(err)
	}
	return left.Compare(right)
}

func HighestBump(values ...Bump) Bump {
	best := BumpNone
	for _, value := range values {
		if bumpRank(value) > bumpRank(best) {
			best = value
		}
	}
	return best
}

func NormalizeBump(raw string) (Bump, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(BumpNone):
		return BumpNone, nil
	case string(BumpPatch):
		return BumpPatch, nil
	case string(BumpMinor):
		return BumpMinor, nil
	case string(BumpMajor):
		return BumpMajor, nil
	default:
		return BumpNone, fmt.Errorf("unsupported bump %q", raw)
	}
}

func NextStable(latestStable *Version, initial Version, bump Bump) Version {
	if latestStable == nil {
		return initial.Stable()
	}

	next := latestStable.Stable()
	switch bump {
	case BumpMajor:
		next.Major++
		next.Minor = 0
		next.Patch = 0
	case BumpMinor:
		next.Minor++
		next.Patch = 0
	case BumpPatch:
		next.Patch++
	case BumpNone:
		return next
	default:
		return next
	}

	return next
}

func NextPrerelease(target Version, label string, current []Version) Version {
	next := target.Stable()
	next.Prerelease = label + ".1"
	next.BuildMetadata = ""

	for _, item := range current {
		if item.Stable() != target.Stable() {
			continue
		}
		currentLabel, currentNumber, ok := item.PrereleaseLabelNumber()
		if !ok || currentLabel != label {
			continue
		}
		_, nextNumber, _ := next.PrereleaseLabelNumber()
		if currentNumber >= nextNumber {
			next.Prerelease = fmt.Sprintf("%s.%d", label, currentNumber+1)
		}
	}

	return next
}

func bumpRank(value Bump) int {
	switch value {
	case BumpPatch:
		return 1
	case BumpMinor:
		return 2
	case BumpMajor:
		return 3
	default:
		return 0
	}
}
