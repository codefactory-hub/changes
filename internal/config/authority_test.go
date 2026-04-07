package config

import (
	"strings"
	"testing"
)

func TestCheckScopeAuthorityReturnsAmbiguousError(t *testing.T) {
	resolution := ScopeResolution{
		Scope:  ScopeRepo,
		Status: StatusAmbiguous,
		Candidates: []Candidate{
			{Scope: ScopeRepo, Style: StyleXDG, Status: StatusResolved},
			{Scope: ScopeRepo, Style: StyleHome, Status: StatusResolved},
		},
	}

	check, err := CheckScopeAuthority(resolution)
	if err == nil {
		t.Fatalf("CheckScopeAuthority error = nil, want authority error")
	}
	if check.Authoritative != nil {
		t.Fatalf("authoritative = %#v, want nil", check.Authoritative)
	}

	authErr, ok := err.(*AuthorityError)
	if !ok {
		t.Fatalf("error type = %T, want *AuthorityError", err)
	}
	if authErr.Scope != ScopeRepo {
		t.Fatalf("scope = %q, want %q", authErr.Scope, ScopeRepo)
	}
	if authErr.Status != StatusAmbiguous {
		t.Fatalf("status = %q, want %q", authErr.Status, StatusAmbiguous)
	}
	if len(authErr.Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(authErr.Candidates))
	}
}

func TestCheckScopeAuthorityReturnsLegacyOnlyError(t *testing.T) {
	resolution := ScopeResolution{
		Scope:  ScopeRepo,
		Status: StatusLegacyOnly,
		Candidates: []Candidate{
			{Scope: ScopeRepo, Style: StyleHome, Status: StatusLegacyOnly},
		},
	}

	_, err := CheckScopeAuthority(resolution)
	if err == nil {
		t.Fatalf("CheckScopeAuthority error = nil, want authority error")
	}

	authErr, ok := err.(*AuthorityError)
	if !ok {
		t.Fatalf("error type = %T, want *AuthorityError", err)
	}
	if authErr.Status != StatusLegacyOnly {
		t.Fatalf("status = %q, want %q", authErr.Status, StatusLegacyOnly)
	}
}

func TestCheckScopeAuthorityReturnsInvalidManifestError(t *testing.T) {
	resolution := ScopeResolution{
		Scope:  ScopeRepo,
		Status: StatusInvalid,
		Candidates: []Candidate{
			{Scope: ScopeRepo, Style: StyleHome, Status: StatusInvalid},
		},
	}

	_, err := CheckScopeAuthority(resolution)
	if err == nil {
		t.Fatalf("CheckScopeAuthority error = nil, want authority error")
	}

	authErr, ok := err.(*AuthorityError)
	if !ok {
		t.Fatalf("error type = %T, want *AuthorityError", err)
	}
	if authErr.Status != StatusInvalid {
		t.Fatalf("status = %q, want %q", authErr.Status, StatusInvalid)
	}
}

func TestCheckScopeAuthorityReturnsStructuredWarnings(t *testing.T) {
	authoritative := Candidate{Scope: ScopeRepo, Style: StyleXDG, Status: StatusResolved}
	warnings := []AuthorityWarning{{
		Scope:  ScopeRepo,
		Style:  StyleHome,
		Status: StatusLegacyOnly,
		Path:   "/tmp/repo/.changes/config",
	}}
	resolution := ScopeResolution{
		Scope:         ScopeRepo,
		Status:        StatusResolved,
		Authoritative: &authoritative,
		Warnings:      warnings,
	}

	check, err := CheckScopeAuthority(resolution)
	if err != nil {
		t.Fatalf("CheckScopeAuthority returned error: %v", err)
	}
	if check.Authoritative == nil {
		t.Fatalf("authoritative = nil")
	}
	if check.Authoritative.Style != StyleXDG {
		t.Fatalf("authoritative style = %q, want %q", check.Authoritative.Style, StyleXDG)
	}
	if len(check.Warnings) != 1 {
		t.Fatalf("warning count = %d, want 1", len(check.Warnings))
	}
	if check.Warnings[0] != warnings[0] {
		t.Fatalf("warning = %#v, want %#v", check.Warnings[0], warnings[0])
	}
}

func TestAuthorityErrorMessageIncludesDoctorHint(t *testing.T) {
	err := &AuthorityError{
		Scope:  ScopeRepo,
		Status: StatusAmbiguous,
		Candidates: []Candidate{
			{Scope: ScopeRepo, Style: StyleXDG, Status: StatusResolved},
			{Scope: ScopeRepo, Style: StyleHome, Status: StatusResolved},
		},
	}

	message := err.Error()
	if !strings.Contains(message, "repo") {
		t.Fatalf("message = %q, want scope", message)
	}
	if !strings.Contains(message, "changes doctor --scope repo") {
		t.Fatalf("message = %q, want doctor hint", message)
	}
	if !strings.Contains(message, "xdg") || !strings.Contains(message, "home") {
		t.Fatalf("message = %q, want competing styles", message)
	}
}
