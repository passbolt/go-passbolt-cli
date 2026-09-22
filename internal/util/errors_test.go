package util

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
)

// TestSentinelsMatchableWhenWrapped documents the contract that callers and
// tests rely on: the sentinel errors stay matchable with errors.Is even after
// being wrapped with additional context via fmt.Errorf and %w.
func TestSentinelsMatchableWhenWrapped(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"ErrNoID", ErrNoID},
		{"ErrNoColumns", ErrNoColumns},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Bare sentinel matches itself.
			if !errors.Is(tt.sentinel, tt.sentinel) {
				t.Fatalf("bare sentinel %v does not match itself", tt.sentinel)
			}
			// Wrapped with %w stays matchable.
			wrapped := fmt.Errorf("doing something: %w", tt.sentinel)
			if !errors.Is(wrapped, tt.sentinel) {
				t.Errorf("wrapped error %q is not matchable with errors.Is", wrapped)
			}
			// Wrapped with extra context (the %w + %v idiom) stays matchable.
			withContext := fmt.Errorf("%w (extra context %v)", tt.sentinel, 42)
			if !errors.Is(withContext, tt.sentinel) {
				t.Errorf("context-wrapped error %q is not matchable with errors.Is", withContext)
			}
		})
	}
}

// TestSentinelsAreDistinct guards against accidentally aliasing the sentinels
// to the same value, which would make errors.Is matching ambiguous.
func TestSentinelsAreDistinct(t *testing.T) {
	if errors.Is(ErrNoID, ErrNoColumns) || errors.Is(ErrNoColumns, ErrNoID) {
		t.Error("ErrNoID and ErrNoColumns must be distinct sentinel errors")
	}
}

// TestNoColumnsError checks the helper lists the valid columns and stays
// matchable with errors.Is(ErrNoColumns).
func TestNoColumnsError(t *testing.T) {
	err := NoColumnsError([]string{"ID", "Name"})
	if !errors.Is(err, ErrNoColumns) {
		t.Errorf("NoColumnsError must wrap ErrNoColumns, got %q", err)
	}
	if !strings.Contains(err.Error(), "ID") || !strings.Contains(err.Error(), "Name") {
		t.Errorf("NoColumnsError should list valid columns, got %q", err)
	}
}

// TestExplainAPIError verifies a friendly hint is added for known status codes
// while the underlying *api.APIError stays recoverable with errors.As, and that
// non-API errors are merely op-wrapped.
func TestExplainAPIError(t *testing.T) {
	apiErr := &api.APIError{StatusCode: 403, Message: "Forbidden"}
	err := ExplainAPIError("logging in", apiErr)
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("expected friendly 403 hint, got %q", err.Error())
	}
	if _, ok := errors.AsType[*api.APIError](err); !ok {
		t.Error("ExplainAPIError must preserve the *api.APIError in the chain")
	}

	base := errors.New("boom")
	wrapped := ExplainAPIError("doing thing", base)
	if !errors.Is(wrapped, base) {
		t.Error("ExplainAPIError must preserve a non-API error chain")
	}
	if !strings.HasPrefix(wrapped.Error(), "doing thing: ") {
		t.Errorf("expected op prefix, got %q", wrapped.Error())
	}
}

// TestAPIStatusHint covers every branch of the status→hint mapping, documenting
// the exact user-facing guidance per code. "" means no hint applies.
func TestAPIStatusHint(t *testing.T) {
	cases := []struct {
		code int
		want string // substring; "" asserts an empty hint
	}{
		{401, "authentication failed"},
		{403, "access denied"},
		{404, "not found"},
		{500, "internal error"},
		{503, "internal error"},
		{400, ""},
		{422, ""},
	}
	for _, c := range cases {
		got := apiStatusHint(c.code)
		if c.want == "" {
			if got != "" {
				t.Errorf("apiStatusHint(%d) = %q, want empty", c.code, got)
			}
			continue
		}
		if !strings.Contains(got, c.want) {
			t.Errorf("apiStatusHint(%d) = %q, want it to contain %q", c.code, got, c.want)
		}
	}
}

// TestExplainAPIError_NoHintStatus verifies that an API error with a status that
// has no hint (400) is only op-wrapped, with no spurious hint text, while the
// *api.APIError stays recoverable via errors.As.
func TestExplainAPIError_NoHintStatus(t *testing.T) {
	apiErr := &api.APIError{StatusCode: 400, Message: "Bad Request"}
	err := ExplainAPIError("doing thing", apiErr)

	if !strings.HasPrefix(err.Error(), "doing thing: ") {
		t.Errorf("expected op prefix, got %q", err.Error())
	}
	if _, ok := errors.AsType[*api.APIError](err); !ok {
		t.Error("ExplainAPIError must preserve the *api.APIError in the chain")
	}
	// The message should be just the op wrap around the API error, with no hint
	// phrasing injected.
	if strings.Contains(err.Error(), "authentication failed") ||
		strings.Contains(err.Error(), "access denied") ||
		strings.Contains(err.Error(), "internal error") {
		t.Errorf("no hint should be added for status 400, got %q", err.Error())
	}
}

// TestExplainWriteError_SchemaMismatch covers the guidance shown when strict write validation
// rejects a document: name the type, lead with the likelier cause, keep the chain intact.
func TestExplainWriteError_SchemaMismatch(t *testing.T) {
	inner := fmt.Errorf("validating metadata: %w: additional properties 'bogus' not allowed",
		helper.ErrSchemaMismatch)

	err := ExplainWriteError("creating Resource", "v5-default", inner)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, helper.ErrSchemaMismatch) {
		t.Error("the original chain must stay matchable with errors.Is")
	}

	msg := err.Error()
	for _, want := range []string{
		"creating Resource",
		`Resource type "v5-default"`,
		"check the field names",
		"update go-passbolt-cli",
		"bogus", // the jsonschema detail must survive
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should contain %q, got %q", want, msg)
		}
	}

	// "check the field names" must come before the update advice, since a typo is the commoner
	// cause of this error.
	if strings.Index(msg, "check the field names") > strings.Index(msg, "update go-passbolt-cli") {
		t.Error("the field-name explanation should precede the update advice")
	}
}

// TestExplainWriteError_UnknownSlug covers the update path, which never resolves a slug.
func TestExplainWriteError_UnknownSlug(t *testing.T) {
	inner := fmt.Errorf("wrapped: %w", helper.ErrSchemaMismatch)

	msg := ExplainWriteError("updating Resource", "", inner).Error()
	if !strings.Contains(msg, "this Resource type") {
		t.Errorf("expected the no-slug phrasing, got %q", msg)
	}
	if strings.Contains(msg, `""`) {
		t.Errorf("an empty slug must not render as empty quotes, got %q", msg)
	}
}

// TestExplainWriteError_FallsThroughToAPIHint checks that routing write commands through
// ExplainWriteError did not cost them ExplainAPIError's status guidance.
func TestExplainWriteError_FallsThroughToAPIHint(t *testing.T) {
	apiErr := &api.APIError{StatusCode: http.StatusForbidden}

	msg := ExplainWriteError("creating Resource", "v5-default", apiErr).Error()
	if !strings.Contains(msg, "access denied") {
		t.Errorf("expected the 403 hint from ExplainAPIError, got %q", msg)
	}
	if strings.Contains(msg, "bundled schema") {
		t.Errorf("a non-schema error must not get the schema hint, got %q", msg)
	}
}

// TestExplainReadError_UnsupportedType covers a Resource whose type this build has no schema for.
func TestExplainReadError_UnsupportedType(t *testing.T) {
	inner := fmt.Errorf("getting metadata: %w: v5-quantum", helper.ErrUnsupportedResourceType)

	err := ExplainReadError("decrypting Resource", "v5-quantum", inner)
	if !errors.Is(err, helper.ErrUnsupportedResourceType) {
		t.Error("the original chain must stay matchable with errors.Is")
	}

	msg := err.Error()
	for _, want := range []string{"decrypting Resource", "has no schema", `"v5-quantum"`, "update go-passbolt-cli"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should contain %q, got %q", want, msg)
		}
	}

	// An unrelated error keeps the plain wrapping.
	plain := ExplainReadError("decrypting Resource", "v5-default", errors.New("boom")).Error()
	if strings.Contains(plain, "has no schema") {
		t.Errorf("an unrelated error must not get the unsupported-type hint, got %q", plain)
	}
}

// TestExplainWriteError_SchemaValidation covers a write whose values, not names, break the
// schema: the hint points at the values and never at the field names.
func TestExplainWriteError_SchemaValidation(t *testing.T) {
	inner := fmt.Errorf("validating metadata: %w: 'name' length must be <= 255", helper.ErrSchemaValidation)

	err := ExplainWriteError("updating Resource", "", inner)
	if !errors.Is(err, helper.ErrSchemaValidation) {
		t.Error("the original chain must stay matchable with errors.Is")
	}

	msg := err.Error()
	for _, want := range []string{"updating Resource", "this Resource type", "check the field values", "update go-passbolt-cli", "255"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should contain %q, got %q", want, msg)
		}
	}
	if strings.Contains(msg, "check the field names") {
		t.Errorf("a declared-constraint failure must not blame the field names, got %q", msg)
	}
}

// The SDK's write guards return ErrUnsupportedResourceType, which needs its own hint.
func TestExplainWriteError_UnsupportedType(t *testing.T) {
	inner := fmt.Errorf("%w: v5-brandnew", helper.ErrUnsupportedResourceType)

	err := ExplainWriteError("creating Resource", "v5-brandnew", inner)
	if !errors.Is(err, helper.ErrUnsupportedResourceType) {
		t.Error("the original chain must stay matchable with errors.Is")
	}

	msg := err.Error()
	for _, want := range []string{"creating Resource", "has no schema", `"v5-brandnew"`, "update go-passbolt-cli"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should contain %q, got %q", want, msg)
		}
	}
}

// A stored document this build rejects: the user supplied nothing, so the advice is to update.
func TestExplainReadError_SchemaOutdated(t *testing.T) {
	inner := fmt.Errorf("getting metadata: %w: at '/icon/type': value must be one of", helper.ErrSchemaValidation)

	err := ExplainReadError("decrypting Resource", "v5-default", inner)
	if !errors.Is(err, helper.ErrSchemaValidation) {
		t.Error("the original chain must stay matchable with errors.Is")
	}

	msg := err.Error()
	for _, want := range []string{"stored data does not match", `"v5-default"`, "update go-passbolt-cli"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should contain %q, got %q", want, msg)
		}
	}
	if strings.Contains(msg, "check the field names") {
		t.Error("the read path must not suggest checking field names; the user supplied none")
	}
}
