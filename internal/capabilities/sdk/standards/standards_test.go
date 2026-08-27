package standards_test

import (
	"errors"
	"testing"

	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
	"github.com/unhield/limoxel/internal/capabilities/sdk/standards"
)

func TestValidateExportedName(t *testing.T) {
	validNames := []string{"Repository", "LookupSymbol", "GraphNode", "XMLParser", "HTTPClient"}
	for _, name := range validNames {
		if err := standards.ValidateExportedName(name); err != nil {
			t.Errorf("expected %q to be valid: %v", name, err)
		}
	}

	invalidNames := []string{"lowercase", "_LeadingUnderscore", "has_underscore", "Has Space", ""}
	for _, name := range invalidNames {
		if err := standards.ValidateExportedName(name); err == nil {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

func TestValidateParameterName(t *testing.T) {
	validParams := []string{"ctx", "repoPath", "symbolID", "query", "limit", "q", "opts"}
	for _, param := range validParams {
		if err := standards.ValidateParameterName(param); err != nil {
			t.Errorf("expected %q to be valid: %v", param, err)
		}
	}

	invalidParams := []string{"PascalCase", "has_underscore", "has space", ""}
	for _, param := range invalidParams {
		if err := standards.ValidateParameterName(param); err == nil {
			t.Errorf("expected %q to be invalid", param)
		}
	}
}

func TestValidateDocComment(t *testing.T) {
	if err := standards.ValidateDocComment("NewClient", "NewClient constructs an initialized SDK Client."); err != nil {
		t.Errorf("expected valid doc comment: %v", err)
	}

	if err := standards.ValidateDocComment("NewClient", "Constructs an initialized client."); err == nil {
		t.Errorf("expected error when doc comment does not start with symbol name")
	}

	if err := standards.ValidateDocComment("NewClient", ""); err == nil {
		t.Errorf("expected error for empty doc comment")
	}
}

func TestValidateSDKErrorCompliance(t *testing.T) {
	compliant := sdkerr.NewInvalidInput("invalid repo path")
	if err := standards.ValidateSDKErrorCompliance(compliant); err != nil {
		t.Errorf("expected compliant error to pass: %v", err)
	}

	nonCompliant := errors.New("raw standard library error")
	if err := standards.ValidateSDKErrorCompliance(nonCompliant); err == nil {
		t.Errorf("expected non-compliant error to fail")
	}
}
