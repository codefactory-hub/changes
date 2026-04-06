package config

type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeRepo   Scope = "repo"
)

type Style string

const (
	StyleXDG  Style = "xdg"
	StyleHome Style = "home"
)

type ResolutionStatus string

const (
	StatusUninitialized ResolutionStatus = "uninitialized"
	StatusResolved      ResolutionStatus = "resolved"
	StatusLegacyOnly    ResolutionStatus = "legacy_only"
	StatusAmbiguous     ResolutionStatus = "ambiguous"
	StatusInvalid       ResolutionStatus = "invalid_manifest"
)

type LayoutPaths struct {
	Root   string
	Config string
	Data   string
	State  string
}

type LayoutManifest struct {
	SchemaVersion int
	Scope         Scope
	Style         Style
	Symbolic      LayoutPaths
	Resolved      LayoutPaths
}

type CandidateEvidence struct {
	Kind   string
	Name   string
	Path   string
	Exists bool
	Detail string
}

type Candidate struct {
	Scope    Scope
	Style    Style
	Status   ResolutionStatus
	Paths    LayoutPaths
	Manifest *LayoutManifest
	Evidence []CandidateEvidence
}

type ScopeResolution struct {
	Scope         Scope
	Status        ResolutionStatus
	Preferred     *Candidate
	Authoritative *Candidate
	Candidates    []Candidate
}

type Resolution struct {
	Global ScopeResolution
	Repo   ScopeResolution
}

type ResolveOptions struct {
	RepoRoot      string
	HomeDir       string
	ChangesHome   string
	XDGConfigHome string
	XDGDataHome   string
	XDGStateHome  string
}
