package config

import (
	"fmt"
	"strings"
)

type AuthorityWarning struct {
	Scope  Scope
	Style  Style
	Status ResolutionStatus
	Path   string
}

type AuthorityCheck struct {
	Scope         Scope
	Authoritative *Candidate
	Warnings      []AuthorityWarning
}

type AuthorityError struct {
	Scope      Scope
	Status     ResolutionStatus
	Candidates []Candidate
}

func (e *AuthorityError) Error() string {
	command := fmt.Sprintf("changes doctor --scope %s", e.Scope)

	switch e.Status {
	case StatusAmbiguous:
		styles := distinctCandidateStyles(e.Candidates)
		if len(styles) == 0 {
			return fmt.Sprintf("%s authority is ambiguous; run %s", e.Scope, command)
		}
		return fmt.Sprintf("%s authority is ambiguous between %s; run %s", e.Scope, strings.Join(styles, ", "), command)
	case StatusLegacyOnly:
		if e.Scope == ScopeRepo {
			return fmt.Sprintf("%s authority is legacy-only; run %s --repair", e.Scope, command)
		}
		return fmt.Sprintf("%s authority is legacy-only; run %s", e.Scope, command)
	case StatusInvalid:
		return fmt.Sprintf("%s authority has an invalid manifest; run %s", e.Scope, command)
	case StatusUninitialized:
		return fmt.Sprintf("%s authority is uninitialized; run %s", e.Scope, command)
	default:
		return fmt.Sprintf("%s authority is unavailable; run %s", e.Scope, command)
	}
}

func CheckScopeAuthority(resolution ScopeResolution) (AuthorityCheck, error) {
	if resolution.Status == StatusResolved {
		if resolution.Authoritative == nil {
			return AuthorityCheck{}, fmt.Errorf("check %s authority: resolved scope missing authoritative candidate", resolution.Scope)
		}
		return AuthorityCheck{
			Scope:         resolution.Scope,
			Authoritative: resolution.Authoritative,
			Warnings:      append([]AuthorityWarning(nil), resolution.Warnings...),
		}, nil
	}

	return AuthorityCheck{}, &AuthorityError{
		Scope:      resolution.Scope,
		Status:     resolution.Status,
		Candidates: append([]Candidate(nil), resolution.Candidates...),
	}
}

func distinctCandidateStyles(candidates []Candidate) []string {
	styles := make([]string, 0, len(candidates))
	seen := make(map[Style]struct{})
	for _, candidate := range candidates {
		if _, ok := seen[candidate.Style]; ok {
			continue
		}
		seen[candidate.Style] = struct{}{}
		styles = append(styles, string(candidate.Style))
	}
	return styles
}
