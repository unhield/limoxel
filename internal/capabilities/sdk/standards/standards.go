package standards

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

var (
	// ErrNamingStandardViolation indicates a violation of SDK naming conventions.
	ErrNamingStandardViolation = errors.New("standards: naming convention violation")

	// ErrDocumentationStandardViolation indicates a missing or non-conforming doc comment.
	ErrDocumentationStandardViolation = errors.New("standards: documentation convention violation")
)

// ValidateExportedName verifies that an exported SDK symbol name adheres to Go and Limoxel SDK standards:
// - Must begin with an uppercase ASCII letter (PascalCase).
// - Must not contain underscores.
// - Must not contain whitespace.
func ValidateExportedName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrNamingStandardViolation)
	}

	runes := []rune(trimmed)
	if !unicode.IsUpper(runes[0]) {
		return fmt.Errorf("%w: exported symbol %q must start with an uppercase letter", ErrNamingStandardViolation, name)
	}

	if strings.Contains(trimmed, "_") {
		return fmt.Errorf("%w: exported symbol %q must not contain underscores (use PascalCase)", ErrNamingStandardViolation, name)
	}

	if strings.Contains(trimmed, " ") {
		return fmt.Errorf("%w: symbol name %q must not contain spaces", ErrNamingStandardViolation, name)
	}

	return nil
}

// ValidateParameterName verifies that a parameter name adheres to camelCase conventions and is descriptive.
func ValidateParameterName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: parameter name cannot be empty", ErrNamingStandardViolation)
	}

	// Accepted standard short names
	if trimmed == "ctx" || trimmed == "q" || trimmed == "d" || trimmed == "v" || trimmed == "err" {
		return nil
	}

	runes := []rune(trimmed)
	if !unicode.IsLower(runes[0]) {
		return fmt.Errorf("%w: parameter %q must start with a lowercase letter (camelCase)", ErrNamingStandardViolation, name)
	}

	if strings.Contains(trimmed, "_") {
		return fmt.Errorf("%w: parameter %q must not contain underscores (use camelCase)", ErrNamingStandardViolation, name)
	}

	if strings.Contains(trimmed, " ") {
		return fmt.Errorf("%w: parameter %q must not contain spaces", ErrNamingStandardViolation, name)
	}

	return nil
}

// ValidateDocComment verifies that a doc comment for a named symbol exists and begins with the symbol's name.
func ValidateDocComment(symbolName, docComment string) error {
	trimmedSym := strings.TrimSpace(symbolName)
	trimmedDoc := strings.TrimSpace(docComment)

	if trimmedDoc == "" {
		return fmt.Errorf("%w: exported symbol %q must have a non-empty doc comment", ErrDocumentationStandardViolation, symbolName)
	}

	// Standard Go doc convention: first word of comment should match symbol name
	fields := strings.Fields(trimmedDoc)
	if len(fields) == 0 {
		return fmt.Errorf("%w: empty doc comment for symbol %q", ErrDocumentationStandardViolation, symbolName)
	}

	firstWord := strings.TrimSuffix(fields[0], ":")
	if firstWord != trimmedSym {
		return fmt.Errorf("%w: doc comment for %q should start with %q (got %q)", ErrDocumentationStandardViolation, symbolName, symbolName, firstWord)
	}

	return nil
}

// ValidateSDKErrorCompliance verifies that an error instance conforms to SDKError contracts.
func ValidateSDKErrorCompliance(err error) error {
	if err == nil {
		return nil
	}
	sdkErr, ok := sdkerr.GetSDKError(err)
	if !ok || sdkErr == nil {
		return sdkerr.NewInvalidState("NON_COMPLIANT_ERROR", fmt.Sprintf("error %v does not implement SDKError contract", err))
	}
	if sdkErr.Category() == "" {
		return sdkerr.NewInvalidState("EMPTY_ERROR_CATEGORY", "SDK error must have a non-empty SemanticCategory")
	}
	if sdkErr.Code() == "" {
		return sdkerr.NewInvalidState("EMPTY_ERROR_CODE", "SDK error must have a non-empty error code")
	}
	return nil
}
