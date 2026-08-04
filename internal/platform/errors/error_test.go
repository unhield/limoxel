package errors_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	platErr "github.com/unhield/limoxel/internal/platform/errors"
)

func TestPlatformErrorConstructors(t *testing.T) {
	t.Run("New with default severity", func(t *testing.T) {
		err := platErr.New(platErr.CodeInvalidInput, "invalid argument")
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if err.Code() != platErr.CodeInvalidInput {
			t.Errorf("got code %v, want %v", err.Code(), platErr.CodeInvalidInput)
		}
		if err.Severity() != platErr.SeverityError {
			t.Errorf("got severity %v, want %v", err.Severity(), platErr.SeverityError)
		}
		if err.Message() != "invalid argument" {
			t.Errorf("got message %q, want %q", err.Message(), "invalid argument")
		}
		if err.Unwrap() != nil {
			t.Errorf("expected nil unwrap, got %v", err.Unwrap())
		}
		if err.Timestamp().IsZero() {
			t.Error("expected non-zero timestamp")
		}
	})

	t.Run("New with empty code defaults to CodeInternal", func(t *testing.T) {
		err := platErr.New("", "internal error")
		if err.Code() != platErr.CodeInternal {
			t.Errorf("got code %v, want %v", err.Code(), platErr.CodeInternal)
		}
	})

	t.Run("NewWithSeverity", func(t *testing.T) {
		err := platErr.NewWithSeverity(platErr.CodeNotFound, platErr.SeverityWarning, "resource missing")
		if err.Severity() != platErr.SeverityWarning {
			t.Errorf("got severity %v, want %v", err.Severity(), platErr.SeverityWarning)
		}
	})

	t.Run("Wrap", func(t *testing.T) {
		cause := errors.New("underlying cause")
		err := platErr.Wrap(cause, platErr.CodeTimeout, "operation timed out")
		if !errors.Is(err, cause) {
			t.Error("expected errors.Is to match cause")
		}
		if err.Unwrap() != cause {
			t.Errorf("got cause %v, want %v", err.Unwrap(), cause)
		}
		expectedStr := "[ERR_TIMEOUT] operation timed out: underlying cause"
		if err.Error() != expectedStr {
			t.Errorf("got Error() %q, want %q", err.Error(), expectedStr)
		}
	})
}

func TestPlatformErrorImmutabilityAndMetadata(t *testing.T) {
	orig := platErr.New(platErr.CodeUnauthorized, "access denied")
	metaErr := orig.WithMetadata("user", "admin")

	if len(orig.Metadata()) != 0 {
		t.Errorf("original metadata should be empty, got %v", orig.Metadata())
	}
	if metaErr.Metadata()["user"] != "admin" {
		t.Errorf("got metadata %v, want admin", metaErr.Metadata())
	}

	meta := metaErr.Metadata()
	meta["user"] = "hacked"
	if metaErr.Metadata()["user"] == "hacked" {
		t.Error("metadata map mutation leaked into error instance")
	}

	cause := errors.New("db error")
	causeErr := orig.WithCause(cause)
	if orig.Unwrap() != nil {
		t.Error("original error should have nil cause")
	}
	if causeErr.Unwrap() != cause {
		t.Errorf("got cause %v, want %v", causeErr.Unwrap(), cause)
	}

	cloned := causeErr.Clone()
	if cloned.Code() != causeErr.Code() || cloned.Message() != causeErr.Message() {
		t.Error("cloned error attributes do not match original")
	}
}

func TestPlatformErrorFormatDetails(t *testing.T) {
	cause := errors.New("network failure")
	err := platErr.Wrap(cause, platErr.CodeUnavailable, "service down").WithMetadata("retry", "true")
	details := err.(*platErr.PlatformError).FormatDetails()

	for _, substr := range []string{"Code: ERR_UNAVAILABLE", "Severity: ERROR", "Message: service down", "Cause: network failure", "retry: true"} {
		if !strings.Contains(details, substr) {
			t.Errorf("FormatDetails missing substring %q in:\n%s", substr, details)
		}
	}
}

func TestNilPlatformErrorSafety(t *testing.T) {
	var err *platErr.PlatformError
	if err.Error() != "<nil>" {
		t.Errorf("got %q, want <nil>", err.Error())
	}
	if err.Code() != platErr.CodeInternal {
		t.Errorf("got code %v, want CodeInternal", err.Code())
	}
	if err.Severity() != platErr.SeverityError {
		t.Errorf("got severity %v, want SeverityError", err.Severity())
	}
	if err.Message() != "" {
		t.Errorf("got message %q, want empty", err.Message())
	}
	if err.Unwrap() != nil {
		t.Error("expected nil unwrap")
	}
	if err.Metadata() != nil {
		t.Error("expected nil metadata")
	}
	if !err.Timestamp().IsZero() {
		t.Error("expected zero timestamp")
	}
	if err.WithMetadata("k", "v") != nil {
		t.Error("expected nil WithMetadata")
	}
	if err.WithCause(errors.New("err")) != nil {
		t.Error("expected nil WithCause")
	}
	if err.Clone() != nil {
		t.Error("expected nil Clone")
	}
	if err.FormatDetails() != "<nil>" {
		t.Errorf("got FormatDetails %q, want <nil>", err.FormatDetails())
	}
}

func TestCodeAndSeverityValidation(t *testing.T) {
	t.Run("ValidateCode", func(t *testing.T) {
		if err := platErr.ValidateCode(platErr.CodeInternal); err != nil {
			t.Errorf("unexpected error for valid code: %v", err)
		}
		if err := platErr.ValidateCode(""); !errors.Is(err, platErr.ErrCodeInvalid) {
			t.Errorf("got %v, want ErrCodeInvalid", err)
		}
		if err := platErr.ValidateCode("ERR INVALID"); err == nil {
			t.Error("expected error for code with spaces")
		}
	})

	t.Run("ParseSeverity", func(t *testing.T) {
		tests := []struct {
			input    string
			expected platErr.Severity
			err      bool
		}{
			{"INFO", platErr.SeverityInfo, false},
			{"WARN", platErr.SeverityWarning, false},
			{"WARNING", platErr.SeverityWarning, false},
			{"ERROR", platErr.SeverityError, false},
			{"CRITICAL", platErr.SeverityCritical, false},
			{"FATAL", platErr.SeverityFatal, false},
			{"UNKNOWN_SEV", platErr.SeverityError, true},
		}

		for _, tt := range tests {
			sev, err := platErr.ParseSeverity(tt.input)
			if tt.err && err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
			if !tt.err && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
			if !tt.err && sev != tt.expected {
				t.Errorf("input %q: got %v, want %v", tt.input, sev, tt.expected)
			}
		}
	})

	t.Run("Severity String()", func(t *testing.T) {
		if platErr.SeverityInfo.String() != "INFO" {
			t.Errorf("got %s, want INFO", platErr.SeverityInfo.String())
		}
		if platErr.Severity(99).String() != "UNKNOWN(99)" {
			t.Errorf("got %s, want UNKNOWN(99)", platErr.Severity(99).String())
		}
	})
}

func TestReexportedErrorHelpers(t *testing.T) {
	cause := errors.New("base error")
	wrapped := fmt.Errorf("wrapped: %w", cause)

	if !platErr.Is(wrapped, cause) {
		t.Error("platErr.Is failed")
	}
	if platErr.Unwrap(wrapped) != cause {
		t.Error("platErr.Unwrap failed")
	}

	var target *platErr.PlatformError
	pErr := platErr.New(platErr.CodeInternal, "fail")
	if !platErr.As(pErr, &target) || target != pErr {
		t.Error("platErr.As failed")
	}
}
