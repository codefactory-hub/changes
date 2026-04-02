package versioning

import (
	"fmt"
	"strconv"
	"strings"
)

type Bump string

const (
	BumpNone  Bump = "none"
	BumpPatch Bump = "patch"
	BumpMinor Bump = "minor"
	BumpMajor Bump = "major"
)

type Version struct {
	Major    int
	Minor    int
	Patch    int
	PreLabel string
	PreNum   int
}

func Parse(raw string) (Version, error) {
	var parsed Version

	core, pre, hasPre := strings.Cut(raw, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("parse version %q: expected semantic version core", raw)
	}

	var err error
	if parsed.Major, err = strconv.Atoi(parts[0]); err != nil {
		return Version{}, fmt.Errorf("parse version %q: invalid major", raw)
	}
	if parsed.Minor, err = strconv.Atoi(parts[1]); err != nil {
		return Version{}, fmt.Errorf("parse version %q: invalid minor", raw)
	}
	if parsed.Patch, err = strconv.Atoi(parts[2]); err != nil {
		return Version{}, fmt.Errorf("parse version %q: invalid patch", raw)
	}

	if !hasPre {
		return parsed, nil
	}

	preParts := strings.Split(pre, ".")
	if len(preParts) != 2 {
		return Version{}, fmt.Errorf("parse version %q: expected prerelease label.number", raw)
	}
	parsed.PreLabel = preParts[0]
	parsed.PreNum, err = strconv.Atoi(preParts[1])
	if err != nil {
		return Version{}, fmt.Errorf("parse version %q: invalid prerelease number", raw)
	}

	return parsed, nil
}

func MustParse(raw string) Version {
	v, err := Parse(raw)
	if err != nil {
		panic(err)
	}
	return v
}

func (v Version) String() string {
	if v.PreLabel == "" {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}
	return fmt.Sprintf("%d.%d.%d-%s.%d", v.Major, v.Minor, v.Patch, v.PreLabel, v.PreNum)
}

func (v Version) Stable() Version {
	v.PreLabel = ""
	v.PreNum = 0
	return v
}

func (v Version) IsPrerelease() bool {
	return v.PreLabel != ""
}

func (v Version) ReleaseLine() string {
	if v.PreLabel == "" {
		return v.Stable().String()
	}
	return fmt.Sprintf("%s-%s", v.Stable().String(), v.PreLabel)
}

func Compare(a, b Version) int {
	if a.Major != b.Major {
		return cmpInt(a.Major, b.Major)
	}
	if a.Minor != b.Minor {
		return cmpInt(a.Minor, b.Minor)
	}
	if a.Patch != b.Patch {
		return cmpInt(a.Patch, b.Patch)
	}
	if a.PreLabel == "" && b.PreLabel == "" {
		return 0
	}
	if a.PreLabel == "" {
		return 1
	}
	if b.PreLabel == "" {
		return -1
	}
	if a.PreLabel != b.PreLabel {
		return strings.Compare(a.PreLabel, b.PreLabel)
	}
	return cmpInt(a.PreNum, b.PreNum)
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
		return initial
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
	next.PreLabel = label
	next.PreNum = 1

	for _, item := range current {
		if item.Stable() != target.Stable() {
			continue
		}
		if item.PreLabel != label {
			continue
		}
		if item.PreNum >= next.PreNum {
			next.PreNum = item.PreNum + 1
		}
	}

	return next
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
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
