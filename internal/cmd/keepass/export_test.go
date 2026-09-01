package keepass

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/passbolt/go-passbolt/api"
	"github.com/tobischo/gokeepasslib/v3"
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

// addCustomFields delegates the metadata/secret merge to
// helper.ParseCustomFields; these tests cover the KDBX-specific part:
// ordering, the Protected flag, and dropping fields KeePass cannot key.
func TestAddCustomFields(t *testing.T) {
	const idA = "11111111-1111-1111-1111-111111111111"
	const idB = "22222222-2222-2222-2222-222222222222"

	metadata := map[string]any{
		"custom_fields": []any{
			map[string]any{"id": idA, "metadata_key": "api-key"},
			// Cleartext value: the secret half must still carry a (empty)
			// secret_value, so this used to export as blank.
			map[string]any{"id": idB, "metadata_key": "env", "metadata_value": "production"},
		},
	}
	secretFields := map[string]any{
		"custom_fields": []any{
			map[string]any{"id": idA, "secret_value": "sk-secret-123"},
			map[string]any{"id": idB, "secret_value": ""},
		},
	}

	entry := gokeepasslib.NewEntry()
	addCustomFields(&entry, metadata, secretFields)

	want := []struct{ key, value string }{
		{"api-key", "sk-secret-123"},
		{"env", "production"},
	}
	if len(entry.Values) != len(want) {
		t.Fatalf("got %d values, want %d: %+v", len(entry.Values), len(want), entry.Values)
	}
	for i, w := range want {
		got := entry.Values[i]
		if got.Key != w.key {
			t.Errorf("value %d: key = %q, want %q (order must follow the metadata array)", i, got.Key, w.key)
		}
		if got.Value.Content != w.value {
			t.Errorf("value %d (%s): content = %q, want %q", i, w.key, got.Value.Content, w.value)
		}
		if !got.Value.Protected.Bool {
			t.Errorf("value %d (%s): not protected, want every custom field protected", i, w.key)
		}
	}
}

func entryContent(entry *gokeepasslib.Entry, key string) (string, bool) {
	for _, v := range entry.Values {
		if v.Key == key {
			return v.Value.Content, true
		}
	}
	return "", false
}

func TestAddTotp_WritesBothConventions(t *testing.T) {
	entry := gokeepasslib.NewEntry()
	addTotp(&entry, map[string]any{
		"totp": map[string]any{
			"secret_key": "jbsw y3dp ehpk3pxp",
			"algorithm":  "SHA256",
			"digits":     float64(8),
			"period":     float64(60),
		},
	}, "Acme", "user", "https://acme.example.com")

	otp, ok := entryContent(&entry, "otp")
	if !ok || !strings.HasPrefix(otp, "otpauth://totp/") {
		t.Errorf("otp = %q, want otpauth://totp/ URI", otp)
	}

	want := map[string]string{
		"TimeOtp-Secret-Base32": "JBSWY3DPEHPK3PXP", // normalized: upper, no spaces
		"TimeOtp-Length":        "8",
		"TimeOtp-Period":        "60",
		"TimeOtp-Algorithm":     "HMAC-SHA-256",
	}
	for key, wantValue := range want {
		if got, ok := entryContent(&entry, key); !ok || got != wantValue {
			t.Errorf("%s = %q (present=%v), want %q", key, got, ok, wantValue)
		}
	}

	for _, v := range entry.Values {
		if v.Key == "otp" || v.Key == "TimeOtp-Secret-Base32" {
			if !v.Value.Protected.Bool {
				t.Errorf("%s must be protected", v.Key)
			}
		}
	}
}

func TestAddTotp_SHA1OmitsAlgorithmField(t *testing.T) {
	// SHA-1 is the KeePass default; writing the field would just be noise.
	entry := gokeepasslib.NewEntry()
	addTotp(&entry, map[string]any{
		"totp": map[string]any{"secret_key": "JBSWY3DPEHPK3PXP", "algorithm": "SHA1", "digits": float64(6), "period": float64(30)},
	}, "n", "u", "")
	if _, ok := entryContent(&entry, "TimeOtp-Algorithm"); ok {
		t.Error("TimeOtp-Algorithm written for SHA1, want omitted")
	}
	if got, ok := entryContent(&entry, "TimeOtp-Length"); !ok || got != "6" {
		t.Errorf("TimeOtp-Length = %q, want 6", got)
	}
}

func TestAddTotp_NoTotpOrNoSecret(t *testing.T) {
	entry := gokeepasslib.NewEntry()
	addTotp(&entry, map[string]any{"password": "p"}, "n", "u", "")
	if len(entry.Values) != 0 {
		t.Errorf("no totp: got %+v, want no values", entry.Values)
	}

	entry = gokeepasslib.NewEntry()
	addTotp(&entry, map[string]any{"totp": map[string]any{"algorithm": "SHA1"}}, "n", "u", "")
	if len(entry.Values) != 0 {
		t.Errorf("totp without secret_key: got %+v, want no values", entry.Values)
	}
}

func TestAppendComments(t *testing.T) {
	created := api.Time{Time: time.Date(2026, 3, 4, 10, 30, 0, 0, time.UTC)}
	entry := gokeepasslib.NewEntry()
	entry.Values = append(entry.Values, gokeepasslib.ValueData{
		Key: "Notes", Value: gokeepasslib.V{Content: "existing description"},
	})

	appendComments(&entry, []api.Comment{
		{Content: "top comment", Created: &created, Children: []api.Comment{
			{Content: "a reply"},
		}},
	})

	notes, _ := entryContent(&entry, "Notes")
	want := "existing description\n\nComments:\n2026-03-04 10:30 top comment\n  a reply"
	if notes != want {
		t.Errorf("Notes = %q, want %q", notes, want)
	}
}

func TestAppendComments_NoComments(t *testing.T) {
	entry := gokeepasslib.NewEntry()
	entry.Values = append(entry.Values, gokeepasslib.ValueData{
		Key: "Notes", Value: gokeepasslib.V{Content: "desc"},
	})
	appendComments(&entry, nil)
	if notes, _ := entryContent(&entry, "Notes"); notes != "desc" {
		t.Errorf("Notes = %q, want unchanged desc", notes)
	}
}

func TestAddTags(t *testing.T) {
	entry := gokeepasslib.NewEntry()
	addTags(&entry, api.Resource{Tags: []api.Tag{{Slug: "prod"}, {Slug: "server"}}})
	if entry.Tags != "prod;server" {
		t.Errorf("Tags = %q, want prod;server", entry.Tags)
	}

	entry = gokeepasslib.NewEntry()
	addTags(&entry, api.Resource{})
	if entry.Tags != "" {
		t.Errorf("no tags: Tags = %q, want empty", entry.Tags)
	}
}

func TestAddTimes(t *testing.T) {
	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	modified := time.Date(2025, 6, 7, 8, 9, 10, 0, time.UTC)
	expired := time.Date(2026, 11, 12, 13, 14, 15, 0, time.UTC)

	entry := gokeepasslib.NewEntry()
	addTimes(&entry, api.Resource{
		Created:  &api.Time{Time: created},
		Modified: &api.Time{Time: modified},
		Expired:  &api.Time{Time: expired},
	})

	if !entry.Times.CreationTime.Time.Equal(created) {
		t.Errorf("CreationTime = %v, want %v", entry.Times.CreationTime.Time, created)
	}
	if !entry.Times.LastModificationTime.Time.Equal(modified) {
		t.Errorf("LastModificationTime = %v, want %v", entry.Times.LastModificationTime.Time, modified)
	}
	if !entry.Times.Expires.Bool {
		t.Error("Expires flag not set for expired resource")
	}
	if entry.Times.ExpiryTime == nil || !entry.Times.ExpiryTime.Time.Equal(expired) {
		t.Errorf("ExpiryTime = %v, want %v", entry.Times.ExpiryTime, expired)
	}
}

func TestAddTimes_NoTimestamps(t *testing.T) {
	// Without timestamps the NewEntry defaults ("now") stay untouched and no
	// expiry is set.
	entry := gokeepasslib.NewEntry()
	before := entry.Times.CreationTime.Time
	addTimes(&entry, api.Resource{})
	if !entry.Times.CreationTime.Time.Equal(before) {
		t.Errorf("CreationTime changed: %v -> %v", before, entry.Times.CreationTime.Time)
	}
	if entry.Times.Expires.Bool {
		t.Error("Expires flag set without an expiry date")
	}
	if entry.Times.ExpiryTime != nil {
		t.Errorf("ExpiryTime = %v, want nil", entry.Times.ExpiryTime)
	}
}

func TestAddCustomFields_SkipsUnnamedAndEmpty(t *testing.T) {
	const idA = "11111111-1111-1111-1111-111111111111"

	// A field with no name on either side cannot be keyed in KDBX.
	entry := gokeepasslib.NewEntry()
	addCustomFields(&entry,
		map[string]any{"custom_fields": []any{map[string]any{"id": idA}}},
		map[string]any{"custom_fields": []any{map[string]any{"id": idA, "secret_value": "v"}}},
	)
	if len(entry.Values) != 0 {
		t.Errorf("unnamed field: got %+v, want no values", entry.Values)
	}

	// A resource without custom fields must add nothing.
	entry = gokeepasslib.NewEntry()
	addCustomFields(&entry, map[string]any{"name": "x"}, map[string]any{"password": "p"})
	if len(entry.Values) != 0 {
		t.Errorf("no custom fields: got %+v, want no values", entry.Values)
	}
}
