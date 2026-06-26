package util

import (
	"testing"

	"github.com/go-sqlite/sqlite3"
)

func TestSelectCookies(t *testing.T) {
	cookies := []cookieRow{
		{name: "_oauth_cookie_", value: "abc", host: "my.passbolt.io"},
		{name: "passbolt_session", value: "sess", host: "my.passbolt.io"},
		{name: "_oauth_cookie_", value: "other", host: "elsewhere.example.com"},
	}

	tests := []struct {
		name       string
		cookies    []cookieRow
		namePrefix string
		hostSuffix string
		want       string
	}{
		{
			name:       "name and host match",
			cookies:    cookies,
			namePrefix: "_oauth_cookie_",
			hostSuffix: "passbolt.io",
			want:       "_oauth_cookie_=abc",
		},
		{
			name:       "host filter excludes other domains",
			cookies:    cookies,
			namePrefix: "_oauth_cookie_",
			hostSuffix: "my.passbolt.io",
			want:       "_oauth_cookie_=abc",
		},
		{
			name:       "empty host matches any",
			cookies:    []cookieRow{{name: "_oauth_cookie_", value: "abc", host: "anything"}},
			namePrefix: "_oauth_cookie_",
			hostSuffix: "",
			want:       "_oauth_cookie_=abc",
		},
		{
			name: "chunked cookies are ordered and joined",
			cookies: []cookieRow{
				{name: "_oauth_cookie__1", value: "two", host: "h"},
				{name: "_oauth_cookie_", value: "zero", host: "h"},
				{name: "_oauth_cookie__0", value: "one", host: "h"},
			},
			namePrefix: "_oauth_cookie_",
			hostSuffix: "",
			want:       "_oauth_cookie_=zero; _oauth_cookie__0=one; _oauth_cookie__1=two",
		},
		{
			name:       "no match returns empty",
			cookies:    cookies,
			namePrefix: "_oauth_cookie_",
			hostSuffix: "no-such-host",
			want:       "",
		},
		{
			name:       "dot-boundary subdomain matches but lookalike host does not",
			cookies:    []cookieRow{{name: "_oauth_cookie_", value: "abc", host: "my.passbolt.io"}},
			namePrefix: "_oauth_cookie_",
			hostSuffix: "passbolt.io",
			want:       "_oauth_cookie_=abc",
		},
		{
			name:       "lookalike host is rejected",
			cookies:    []cookieRow{{name: "_oauth_cookie_", value: "abc", host: "evilpassbolt.io"}},
			namePrefix: "_oauth_cookie_",
			hostSuffix: "passbolt.io",
			want:       "",
		},
		{
			name:       "leading-dot domain cookie matches",
			cookies:    []cookieRow{{name: "_oauth_cookie_", value: "abc", host: ".passbolt.io"}},
			namePrefix: "_oauth_cookie_",
			hostSuffix: "passbolt.io",
			want:       "_oauth_cookie_=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectCookies(tt.cookies, tt.namePrefix, tt.hostSuffix)
			if got != tt.want {
				t.Fatalf("selectCookies() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestReadTableColumns exercises the real SQLite reader against a genuine
// Firefox database file, proving that the column-name to rec.Values index
// mapping is correct on the actual on-disk format (the same format
// cookies.sqlite uses).
func TestReadTableColumns(t *testing.T) {
	db, err := sqlite3.Open("testdata/firefox-history.sqlite")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer db.Close()

	rows, err := readTableColumns(db, "moz_places", []string{"url", "title"})
	if err != nil {
		t.Fatalf("readTableColumns: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}

	// The first stored place is the Firefox firstrun page; assert we read the
	// url column (index resolved by name, not position) as a real string.
	wantURL := "https://www.mozilla.org/en-US/firefox/57.0/firstrun/"
	if rows[0][0] != wantURL {
		t.Fatalf("rows[0] url = %q, want %q", rows[0][0], wantURL)
	}

	if _, err := readTableColumns(db, "moz_cookies", []string{"name"}); err == nil {
		t.Fatal("expected error for missing moz_cookies table")
	}
}

// TestReadFirefoxCookie reads a genuine Firefox-format cookies.sqlite whose
// moz_cookies table carries a table-level
// "CONSTRAINT moz_uniqueid UNIQUE (name, host, path, originAttributes)" clause.
// The underlying reader mis-parses that clause into phantom duplicate columns
// (host, path, originAttributes appear twice), which previously shifted the
// resolved "host" index past the real value and made every cookie fail the host
// filter. This proves the named cookie is now extracted for the right host.
func TestReadFirefoxCookie(t *testing.T) {
	cookie, err := readFirefoxCookie("testdata/firefox-cookies.sqlite", "_oauth_cookie_", "my.passbolt.io")
	if err != nil {
		t.Fatalf("readFirefoxCookie: %v", err)
	}
	if want := "_oauth_cookie_=THE-OAUTH-VALUE"; cookie != want {
		t.Fatalf("readFirefoxCookie() = %q, want %q", cookie, want)
	}

	// A non-matching host must still exclude the cookie.
	cookie, err = readFirefoxCookie("testdata/firefox-cookies.sqlite", "_oauth_cookie_", "no-such-host")
	if err != nil {
		t.Fatalf("readFirefoxCookie (wrong host): %v", err)
	}
	if cookie != "" {
		t.Fatalf("readFirefoxCookie (wrong host) = %q, want empty", cookie)
	}
}
