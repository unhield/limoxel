package errors_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

func TestStandardConstructors(t *testing.T) {
	t.Run("InvalidInput", func(t *testing.T) {
		err := sdkerr.NewInvalidInput("path must not be empty")
		if !sdkerr.IsInvalidInput(err) {
			t.Errorf("expected IsInvalidInput to be true")
		}
		if err.Category() != sdkerr.CategoryInvalidInput {
			t.Errorf("got category %v, want %v", err.Category(), sdkerr.CategoryInvalidInput)
		}
		if err.Code() != "ERR_INVALID_INPUT" {
			t.Errorf("got code %v, want ERR_INVALID_INPUT", err.Code())
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		err := sdkerr.NewNotFound("Symbol", "NewCommand")
		if !sdkerr.IsNotFound(err) {
			t.Errorf("expected IsNotFound to be true")
		}
		if err.Details()["entity_type"] != "Symbol" || err.Details()["identity"] != "NewCommand" {
			t.Errorf("unexpected details: %v", err.Details())
		}
	})

	t.Run("Unsupported", func(t *testing.T) {
		err := sdkerr.NewUnsupported("export_format", "PDF rendering is unsupported")
		if !sdkerr.IsUnsupported(err) {
			t.Errorf("expected IsUnsupported to be true")
		}
	})

	t.Run("InvalidState", func(t *testing.T) {
		err := sdkerr.NewInvalidState("UNINITIALIZED", "workspace is not loaded")
		if !sdkerr.IsInvalidState(err) {
			t.Errorf("expected IsInvalidState to be true")
		}
	})

	t.Run("LifecycleViolation", func(t *testing.T) {
		err := sdkerr.NewLifecycleViolation("OpenLegacy", "REMOVED", "this API was removed in v2.0.0")
		if !sdkerr.IsLifecycleViolation(err) {
			t.Errorf("expected IsLifecycleViolation to be true")
		}
	})

	t.Run("Unavailable", func(t *testing.T) {
		err := sdkerr.NewUnavailable("IndexDB", "database locked")
		if !sdkerr.IsUnavailable(err) {
			t.Errorf("expected IsUnavailable to be true")
		}
	})

	t.Run("InternalAndSanitizedMessage", func(t *testing.T) {
		cause := errors.New("raw postgres connection timeout at 10.0.0.1:5432")
		err := sdkerr.NewInternal("failed to query backend", cause)
		if !sdkerr.IsInternal(err) {
			t.Errorf("expected IsInternal to be true")
		}
		if err.SafePublicMessage() != "an internal error occurred during SDK operation" {
			t.Errorf("unexpected safe public message: %s", err.SafePublicMessage())
		}
		if !strings.Contains(err.Error(), "raw postgres connection timeout") {
			t.Errorf("expected cause in Error() output: %s", err.Error())
		}
		if !errors.Is(err, cause) {
			t.Errorf("expected errors.Is to match wrapped cause")
		}
	})
}

func TestErrorBuildersAndImmutability(t *testing.T) {
	base := sdkerr.NewInvalidInput("initial message")
	withDetail := base.WithDetail("param", "repoPath")
	withCause := withDetail.WithCause(fmt.Errorf("underlying io error"))
	withSafe := withCause.WithSafePublicMessage("sanitized message")

	// Verify original base is untouched
	if len(base.Details()) != 0 {
		t.Errorf("base details modified: %v", base.Details())
	}
	if withDetail.Details()["param"] != "repoPath" {
		t.Errorf("detail missing: %v", withDetail.Details())
	}
	if withCause.Unwrap() == nil {
		t.Errorf("cause missing in withCause")
	}
	if withSafe.SafePublicMessage() != "sanitized message" {
		t.Errorf("safe message not updated")
	}
}

func TestFormatDetails(t *testing.T) {
	err := sdkerr.NewNotFound("Package", "internal/cli").
		WithDetail("caller", "testSuite").
		WithCause(errors.New("filesystem entry missing"))

	formatted := err.FormatDetails()
	if !strings.Contains(formatted, "Category : ERR_NOT_FOUND") ||
		!strings.Contains(formatted, "entity_type: Package") ||
		!strings.Contains(formatted, "Cause    : filesystem entry missing") {
		t.Errorf("unexpected FormatDetails output:\n%s", formatted)
	}
}

func TestGetSDKError(t *testing.T) {
	rawErr := sdkerr.NewInvalidInput("test error")
	wrapped := fmt.Errorf("outer context: %w", rawErr)

	sdkErr, ok := sdkerr.GetSDKError(wrapped)
	if !ok || sdkErr == nil {
		t.Fatalf("expected GetSDKError to find wrapped SDK error")
	}
	if sdkErr.Category() != sdkerr.CategoryInvalidInput {
		t.Errorf("got %v, want CategoryInvalidInput", sdkErr.Category())
	}

	stdErr := errors.New("plain standard error")
	_, ok = sdkerr.GetSDKError(stdErr)
	if ok {
		t.Errorf("expected plain error to return ok=false")
	}

	_, ok = sdkerr.GetSDKError(nil)
	if ok {
		t.Errorf("expected nil error to return ok=false")
	}
}

func TestNilErrorReceiverSafety(t *testing.T) {
	var nilErr *sdkerr.Error

	if nilErr.Error() != "<nil>" {
		t.Errorf("expected <nil>, got %s", nilErr.Error())
	}
	if nilErr.Category() != sdkerr.CategoryInternal {
		t.Errorf("expected CategoryInternal on nil")
	}
	if nilErr.Code() != string(sdkerr.CategoryInternal) {
		t.Errorf("expected ERR_INTERNAL on nil")
	}
	if nilErr.Message() != "" {
		t.Errorf("expected empty message on nil")
	}
	if nilErr.SafePublicMessage() != "" {
		t.Errorf("expected empty safe message on nil")
	}
	if nilErr.Details() != nil {
		t.Errorf("expected nil details on nil")
	}
	if nilErr.Unwrap() != nil {
		t.Errorf("expected nil cause on nil")
	}
	if nilErr.FormatDetails() != "<nil>" {
		t.Errorf("expected <nil> details on nil")
	}
	if nilErr.WithDetail("k", "v") != nil {
		t.Errorf("expected nil on WithDetail")
	}
	if nilErr.WithCause(nil) != nil {
		t.Errorf("expected nil on WithCause")
	}
	if nilErr.WithSafePublicMessage("safe") != nil {
		t.Errorf("expected nil on WithSafePublicMessage")
	}
	if sdkerr.IsInvalidInput(nil) {
		t.Errorf("expected IsInvalidInput(nil) to be false")
	}
}
