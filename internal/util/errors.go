package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/passbolt/go-passbolt/helper"
)

// Sentinel errors for recurring CLI input-validation failures. They are
// returned (optionally wrapped with extra context via fmt.Errorf and %w) so
// callers and tests can match them with errors.Is rather than comparing
// message strings.
var (
	// ErrNoID is returned by commands that require an --id flag when none is provided.
	ErrNoID = errors.New("no ID provided")

	// ErrNoColumns is returned by list/get commands when no output column is selected.
	ErrNoColumns = errors.New("no columns specified")
)

// NoColumnsError wraps ErrNoColumns with the set of valid column names so the
// user knows what to choose from. The result stays matchable with
// errors.Is(err, ErrNoColumns).
func NoColumnsError(valid []string) error {
	return fmt.Errorf("%w; choose at least one of: %s", ErrNoColumns, strings.Join(valid, ", "))
}

// ExplainWriteError wraps a write failure, adding guidance for a schema mismatch, any other
// schema validation failure, or an unsupported Resource type; slug may be "". Other errors go
// through ExplainAPIError. ErrSchemaMismatch is checked first because it also matches
// ErrSchemaValidation.
func ExplainWriteError(op, slug string, err error) error {
	switch {
	case errors.Is(err, helper.ErrSchemaMismatch):
		return fmt.Errorf("%s: %s: %w", op, schemaMismatchHint(slug), err)
	case errors.Is(err, helper.ErrSchemaValidation):
		return fmt.Errorf("%s: %s: %w", op, schemaInvalidHint(slug), err)
	case errors.Is(err, helper.ErrUnsupportedResourceType):
		return fmt.Errorf("%s: %s: %w", op, unsupportedTypeHint(slug), err)
	}
	return ExplainAPIError(op, err)
}

// ExplainReadError is the read-path counterpart: no schema for the type, or a stored document
// the bundled schema rejects. Both mean "update the CLI", so say so.
func ExplainReadError(op, slug string, err error) error {
	switch {
	case errors.Is(err, helper.ErrUnsupportedResourceType):
		return fmt.Errorf("%s: %s: %w", op, unsupportedTypeHint(slug), err)
	case errors.Is(err, helper.ErrSchemaValidation):
		return fmt.Errorf("%s: %s: %w", op, schemaOutdatedHint(slug), err)
	}
	return ExplainAPIError(op, err)
}

// schemaMismatchHint leads with the field names on purpose: strict validation rejects any
// undeclared property, so a mistyped --field is the commoner cause of this error.
func schemaMismatchHint(slug string) string {
	return fmt.Sprintf("the data does not match this CLI's bundled schema for %s; "+
		"check the field names, or update go-passbolt-cli if this server is newer", typeSubject(slug))
}

// schemaInvalidHint covers a write whose field names are all declared but whose values break a
// constraint: a missing required field, a value outside an enum, a string over its maxLength.
func schemaInvalidHint(slug string) string {
	return fmt.Sprintf("the data does not match this CLI's bundled schema for %s; "+
		"check the field values, or update go-passbolt-cli if this server is newer", typeSubject(slug))
}

func unsupportedTypeHint(slug string) string {
	return fmt.Sprintf("this version of go-passbolt-cli has no schema for %s; "+
		"update go-passbolt-cli to read it", typeSubject(slug))
}

// schemaOutdatedHint covers a stored document this CLI's schema rejects. On a read the caller
// supplied nothing, so the schema being behind the server is the only explanation.
func schemaOutdatedHint(slug string) string {
	return fmt.Sprintf("the stored data does not match this CLI's bundled schema for %s; "+
		"update go-passbolt-cli to read it", typeSubject(slug))
}

// typeSubject names a Resource type for a message, falling back when the slug is unknown.
func typeSubject(slug string) string {
	if slug == "" {
		return "this Resource type"
	}
	return fmt.Sprintf("Resource type %q", slug)
}
