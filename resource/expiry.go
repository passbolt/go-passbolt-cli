package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/passbolt/go-passbolt/api"
)

// SetResourceExpiry updates only the expiry date of a resource.
func SetResourceExpiry(ctx context.Context, client *api.Client, id string, expiryInput string) error {
	if expiryInput == "" {
		return nil
	}

	// Safety: ensure the resource id is a UUID to avoid unsafe URL construction
	if !isUUID(id) {
		return fmt.Errorf("invalid resource id: %q", id)
	}

	// allow a single keyword to clear expiry (no TrimSpace: flags shouldn't need quoting spaces)
	switch strings.ToLower(expiryInput) {
	case "none":
		// TODO: Should be handled in go-passbolt when the planned new Resource API is available
		_, _, err := client.DoCustomRequestAndReturnRawResponse(
			ctx,
			"PUT",
			fmt.Sprintf("resources/%s.json", id),
			"v2",
			map[string]*string{"expired": nil},
			nil,
		)
		if err != nil {
			return fmt.Errorf("Clearing expiry: %w", err)
		}
		return nil
	}

	isoExpiry, err := ParseExpiry(expiryInput)
	if err != nil {
		return err
	}
	// TODO: Should be handled in go-passbolt when the planned new Resource API is available
	_, _, err = client.DoCustomRequestAndReturnRawResponse(
		ctx,
		"PUT",
		fmt.Sprintf("resources/%s.json", id),
		"v2",
		map[string]string{"expired": isoExpiry},
		nil,
	)
	if err != nil {
		return fmt.Errorf("Setting expiry: %w", err)
	}
	return nil
}

// ParseExpiry accepts either an absolute time (ISO8601/RFC3339) or a human duration like "7d", "12h", "30m", or combinations like "1w2d3h".
// It returns an ISO8601 (RFC3339) timestamp string in UTC suitable for the API.
func ParseExpiry(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	// Try absolute timestamp first
	if t, err := tryParseAbsoluteTime(input); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	// Fallback to human duration(s)
	d, err := time.ParseDuration(input)
	if err != nil {
		return "", fmt.Errorf("invalid expiry value %q: %w", input, err)
	}
	return time.Now().UTC().Add(d).Format(time.RFC3339), nil
}

func tryParseAbsoluteTime(s string) (time.Time, error) {
	// Try RFC3339 variants only (avoid nonstandard timestamp formats)
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// isUUID performs a basic UUID validation in canonical 8-4-4-4-12 hex format.
func isUUID(s string) bool {
	re := regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return re.MatchString(s)
}
