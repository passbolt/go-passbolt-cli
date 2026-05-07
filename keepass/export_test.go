package keepass

import (
	"net/url"
	"strings"
	"testing"
)

// encodeQuery is a copy of url.Values.Encode that uses %20 instead of '+' for
// spaces, so authenticator apps (Google Authenticator) parse otpauth URIs
// correctly. These tests lock in that distinction.

func TestEncodeQuery_Empty(t *testing.T) {
	if got := encodeQuery(nil); got != "" {
		t.Errorf("nil input = %q, want empty", got)
	}
	if got := encodeQuery(url.Values{}); got != "" {
		t.Errorf("empty input = %q, want empty", got)
	}
}

func TestEncodeQuery_SpacesUsePercent20(t *testing.T) {
	v := url.Values{"issuer": {"My Service"}}
	got := encodeQuery(v)
	if !strings.Contains(got, "%20") {
		t.Errorf("encodeQuery should encode space as %%20, got %q", got)
	}
	if strings.Contains(got, "+") {
		t.Errorf("encodeQuery must not use + for spaces, got %q", got)
	}
}

func TestEncodeQuery_MatchesStdlibAfterPlusSubstitution(t *testing.T) {
	// For values with no plus characters, encodeQuery's output should be
	// identical to url.Values.Encode after replacing '+' with '%20'.
	v := url.Values{
		"issuer": {"Acme"},
		"period": {"30"},
		"digits": {"6"},
	}
	got := encodeQuery(v)
	std := strings.ReplaceAll(v.Encode(), "+", "%20")
	if got != std {
		t.Errorf("encodeQuery(%v) = %q, want %q", v, got, std)
	}
}

func TestEncodeQuery_KeysAreSorted(t *testing.T) {
	v := url.Values{
		"zeta":  {"z"},
		"alpha": {"a"},
		"mu":    {"m"},
	}
	got := encodeQuery(v)
	if !strings.HasPrefix(got, "alpha=a&") || !strings.HasSuffix(got, "&zeta=z") {
		t.Errorf("expected keys in lexicographic order, got %q", got)
	}
}

func TestEncodeQuery_MultiValueRetained(t *testing.T) {
	v := url.Values{"tag": {"foo", "bar"}}
	got := encodeQuery(v)
	if got != "tag=foo&tag=bar" {
		t.Errorf("multi-value encoding = %q, want tag=foo&tag=bar", got)
	}
}

func TestEncodeQuery_DoesNotEscapeAmpersandOrEquals(t *testing.T) {
	// encodeQuery uses url.PathEscape (not QueryEscape) to keep %20 for
	// spaces (required by Google Authenticator). PathEscape does NOT escape
	// '&' or '=', so values containing them produce an ambiguous query
	// string. Acceptable for otpauth labels in practice (they don't contain
	// either character) but a sharp edge — this test locks in current
	// behavior so a future change to QueryEscape is a deliberate decision.
	v := url.Values{"label": {"foo&bar=baz"}}
	got := encodeQuery(v)
	if got != "label=foo&bar=baz" {
		t.Errorf("encodeQuery(%v) = %q, want label=foo&bar=baz (current behavior)", v, got)
	}
}
