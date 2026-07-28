package service

import (
	"strings"
	"testing"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
)

func TestValidateCredentialInputDefaultsAndTrims(t *testing.T) {
	in := &CredentialInput{
		Name:        "  production database  ",
		Kind:        "  database  ",
		Username:    "  app_user  ",
		Description: "  managed credential  ",
		Secret:      &CredentialSecret{Password: "value"},
	}
	if err := validateCredentialInput(in, true); err != nil {
		t.Fatalf("validate credential: %v", err)
	}
	if in.Name != "production database" || in.Kind != "database" || in.Username != "app_user" {
		t.Fatalf("input was not normalized: %#v", in)
	}
	if in.Scope != "private" || in.Status != "active" {
		t.Fatalf("unexpected defaults: scope=%q status=%q", in.Scope, in.Status)
	}
}

func TestValidateCredentialInputRejectsInvalidSecurityMetadata(t *testing.T) {
	tests := []CredentialInput{
		{Name: "credential", Kind: "password", Scope: "everyone", Secret: &CredentialSecret{Password: "value"}},
		{Name: "credential", Kind: "password", Status: "unknown", Secret: &CredentialSecret{Password: "value"}},
		{Name: strings.Repeat("x", 121), Kind: "password", Secret: &CredentialSecret{Password: "value"}},
		{Name: "credential", Kind: "password"},
	}
	for i := range tests {
		if err := validateCredentialInput(&tests[i], true); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestCredentialAvailable(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	if !credentialAvailable(&model.Credential{Status: "active", ExpiresAt: &future}) {
		t.Fatal("active, unexpired credential should be available")
	}
	if credentialAvailable(&model.Credential{Status: "disabled", ExpiresAt: &future}) {
		t.Fatal("disabled credential should be unavailable")
	}
	if credentialAvailable(&model.Credential{Status: "active", ExpiresAt: &past}) {
		t.Fatal("expired credential should be unavailable")
	}
}

func TestValidateDependencyRejectsAmbiguousOrInvalidTarget(t *testing.T) {
	tests := []DependencyInput{
		{SourceApplicationID: 1, SourceEnvironmentID: 1, TargetApplicationID: 2, TargetComponentID: 3, Protocol: "HTTP", Endpoint: "http://target"},
		{SourceApplicationID: 1, SourceEnvironmentID: 1, TargetName: "external", Protocol: "HTTPS", Endpoint: "https://target", Port: 70000},
		{SourceApplicationID: 1},
	}
	for i := range tests {
		if err := validateDependency(&tests[i]); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
	valid := &DependencyInput{SourceApplicationID: 1, SourceEnvironmentID: 1, TargetName: "external", Protocol: "HTTPS", Endpoint: "https://target", Port: 443}
	if err := validateDependency(valid); err != nil {
		t.Fatalf("valid external dependency rejected: %v", err)
	}
}

func TestValidateRevealReason(t *testing.T) {
	if err := validateRevealReason("incident investigation"); err != nil {
		t.Fatalf("valid reason rejected: %v", err)
	}
	if err := validateRevealReason("   "); err == nil {
		t.Fatal("empty reason should be rejected")
	}
	if err := validateRevealReason(strings.Repeat("x", 501)); err == nil {
		t.Fatal("oversized reason should be rejected")
	}
}

func TestValidateCredentialGrantInput(t *testing.T) {
	valid := &CredentialGrantInput{SubjectType: "user", SubjectID: 7, Permissions: model.CredentialPermissionView | model.CredentialPermissionReveal}
	if err := validateCredentialGrantInput(valid); err != nil {
		t.Fatalf("valid grant rejected: %v", err)
	}
	invalid := []CredentialGrantInput{
		{SubjectType: "team", SubjectID: 7, Permissions: model.CredentialPermissionView},
		{SubjectType: "user", SubjectID: 0, Permissions: model.CredentialPermissionView},
		{SubjectType: "role", SubjectID: 7, Permissions: 1 << 10},
	}
	for i := range invalid {
		if err := validateCredentialGrantInput(&invalid[i]); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}
