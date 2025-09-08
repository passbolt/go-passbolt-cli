package resource

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/passbolt/go-passbolt/api"
)

// SetResourceExpiry updates only the expiry date of a resource.
func SetResourceExpiry(ctx context.Context, client *api.Client, id string, expiryInput string) error {
	if expiryInput == "" {
		return nil
	}

	// allow a single keyword to clear expiry
	switch strings.ToLower(strings.TrimSpace(expiryInput)) {
	case "none":
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
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil
	}
	// Try absolute timestamp first
	if t, err := tryParseAbsoluteTime(trimmed); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	// Fallback to human duration(s)
	d, err := parseHumanDuration(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid expiry value %q: %w", input, err)
	}
	return time.Now().UTC().Add(d).Format(time.RFC3339), nil
}

func tryParseAbsoluteTime(s string) (time.Time, error) {
	// Try common RFC3339 variants
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05-0700", // without colon in offset
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

// parseHumanDuration supports segments like 1w2d3h4m5s (order-insensitive)
func parseHumanDuration(s string) (time.Duration, error) {
	// If it is a plain Go duration, delegate
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	re := regexp.MustCompile(`(?i)(\d+)\s*([wdhms])`)
	matches := re.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("cannot parse duration")
	}
	var total time.Duration
	for _, m := range matches {
		numStr := m[1]
		unit := strings.ToLower(m[2])
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, err
		}
		switch unit {
		case "w":
			total += time.Duration(n) * 7 * 24 * time.Hour
		case "d":
			total += time.Duration(n) * 24 * time.Hour
		case "h":
			total += time.Duration(n) * time.Hour
		case "m":
			total += time.Duration(n) * time.Minute
		case "s":
			total += time.Duration(n) * time.Second
		default:
			return 0, fmt.Errorf("unknown unit %q", unit)
		}
	}
	return total, nil
}
