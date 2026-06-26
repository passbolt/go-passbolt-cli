package util

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/go-sqlite/sqlite3"
	"github.com/spf13/viper"
)

// GetSSOCookie reads the session cookie of an SSO reverse proxy (for example
// oauth2-proxy's "_oauth_cookie_") directly from a Firefox-based browser's
// cookie store, so the CLI can transparently pass it on every request when the
// Passbolt instance sits behind such a proxy.
//
// It returns a Cookie header value like "_oauth_cookie_=...", reading the value
// fresh on every call so an expiring proxy session is handled as long as the
// browser is still logged in. The actual cookie value is never written to the
// CLI config: only the cookie name/host/database path are persisted.
//
// It returns an empty string (and no error) when the feature is disabled, i.e.
// when ssoCookieName is not set.
func GetSSOCookie() (string, error) {
	name := viper.GetString("ssoCookieName")
	if name == "" {
		return "", nil
	}

	host := viper.GetString("ssoCookieHost")
	if host == "" {
		if serverAddress := viper.GetString("serverAddress"); serverAddress != "" {
			if u, err := url.Parse(serverAddress); err == nil {
				host = u.Hostname()
			}
		}
	}

	databases, err := ssoCookieDatabases()
	if err != nil {
		return "", err
	}
	if len(databases) == 0 {
		return "", fmt.Errorf("no Firefox-based browser cookie database found, set ssoCookieDatabase to the path of a cookies.sqlite")
	}

	for _, db := range databases {
		cookie, err := readFirefoxCookie(db, name, host)
		if err != nil {
			if viper.GetBool("debug") {
				fmt.Fprintf(os.Stderr, "sso cookie: skipping %s: %v\n", db, err)
			}
			continue
		}
		if cookie != "" {
			return cookie, nil
		}
	}
	return "", fmt.Errorf("no %q cookie for host %q found in the browser cookie store, are you logged in?", name, host)
}

// ssoCookieDatabases returns the cookies.sqlite files to inspect. If
// ssoCookieDatabase is set it is used verbatim, otherwise Firefox-family
// profiles are auto-discovered for the current OS.
func ssoCookieDatabases() ([]string, error) {
	if explicit := viper.GetString("ssoCookieDatabase"); explicit != "" {
		return []string{explicit}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting Home Directory: %w", err)
	}

	var roots []string
	switch runtime.GOOS {
	case "darwin":
		base := filepath.Join(home, "Library", "Application Support")
		roots = []string{
			filepath.Join(base, "zen", "Profiles"),
			filepath.Join(base, "Firefox", "Profiles"),
			filepath.Join(base, "LibreWolf", "Profiles"),
		}
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			roots = []string{
				filepath.Join(appData, "zen", "Profiles"),
				filepath.Join(appData, "Mozilla", "Firefox", "Profiles"),
				filepath.Join(appData, "LibreWolf", "Profiles"),
			}
		}
	default: // linux and others
		roots = []string{
			filepath.Join(home, ".zen"),
			filepath.Join(home, ".mozilla", "firefox"),
			filepath.Join(home, ".librewolf"),
		}
	}

	var databases []string
	for _, root := range roots {
		matches, err := filepath.Glob(filepath.Join(root, "*", "cookies.sqlite"))
		if err != nil {
			continue
		}
		databases = append(databases, matches...)
	}
	return databases, nil
}

// readFirefoxCookie extracts the named cookie (and any oauth2-proxy chunks) for
// the given host from a Firefox-based cookies.sqlite. The database is copied to
// a temporary location first so the live store the browser is using is never
// touched.
//
// Two limitations stem from the minimal pure-Go reader used here
// (github.com/go-sqlite/sqlite3): it reads the committed database and does not
// replay the write-ahead log, so a cookie set moments ago may not be visible
// until the browser checkpoints; and it truncates TEXT values at the first NUL
// byte. Neither affects oauth2-proxy session cookies, which are set at login
// and are NUL-free base64, but both are worth knowing when extending this.
func readFirefoxCookie(dbPath, name, host string) (string, error) {
	tmp, err := os.MkdirTemp("", "passbolt-cookie-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	dst := filepath.Join(tmp, "cookies.sqlite")
	if err := copyFile(dbPath, dst); err != nil {
		return "", fmt.Errorf("copying cookie database: %w", err)
	}

	db, err := sqlite3.Open(dst)
	if err != nil {
		return "", err
	}
	defer func() { _ = db.Close() }()

	rows, err := readTableColumns(db, "moz_cookies", []string{"name", "value", "host"})
	if err != nil {
		return "", err
	}

	cookies := make([]cookieRow, 0, len(rows))
	for _, r := range rows {
		cookies = append(cookies, cookieRow{name: r[0], value: r[1], host: r[2]})
	}
	return selectCookies(cookies, name, host), nil
}

type cookieRow struct {
	name  string
	value string
	host  string
}

// selectCookies filters cookies by a name prefix (so oauth2-proxy chunks
// name, name_0, name_1, ... are all captured) and an optional host, then
// returns them as a Cookie header value with chunks ordered by name.
func selectCookies(cookies []cookieRow, namePrefix, host string) string {
	var matches []cookieRow
	for _, c := range cookies {
		if !strings.HasPrefix(c.name, namePrefix) {
			continue
		}
		if !hostMatches(c.host, host) {
			continue
		}
		matches = append(matches, c)
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].name < matches[j].name })

	parts := make([]string, 0, len(matches))
	for _, m := range matches {
		parts = append(parts, m.name+"="+m.value)
	}
	return strings.Join(parts, "; ")
}

// hostMatches reports whether a cookie stored for cookieHost should be sent to
// the configured want host. An empty want matches any host. Otherwise it
// accepts an exact match or a dot-boundary subdomain match (so want
// "passbolt.io" matches "my.passbolt.io" but not "evilpassbolt.io"), and
// tolerates the leading dot Firefox stores on domain cookies (".passbolt.io").
// The dot boundary is what a bare strings.HasSuffix would miss.
func hostMatches(cookieHost, want string) bool {
	if want == "" {
		return true
	}
	cookieHost = strings.TrimPrefix(cookieHost, ".")
	return cookieHost == want || strings.HasSuffix(cookieHost, "."+want)
}

// readTableColumns reads the given columns of every row of a table into string
// rows. Record values are stored in schema-column order, so the column name to
// index mapping derived from the table definition lines up with rec.Values.
func readTableColumns(db *sqlite3.DbFile, table string, columns []string) ([][]string, error) {
	var cols []sqlite3.Column
	found := false
	for _, t := range db.Tables() {
		if t.Name() == table {
			cols = t.Columns()
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("no %s table (not the expected database format?)", table)
	}

	// Keep the first occurrence of each column name. The underlying reader
	// derives columns by splitting the CREATE TABLE text on commas and only
	// drops CHECK/FOREIGN KEY/UNIQUE/PRIMARY KEY clauses, so a table-level
	// "CONSTRAINT ... UNIQUE (name, host, path, ...)" (as in Firefox's
	// moz_cookies) leaks its referenced names back in as phantom duplicate
	// columns. Those phantoms always follow the real column definitions, so
	// first-occurrence-wins resolves each name to its real position.
	index := map[string]int{}
	for i, c := range cols {
		if _, ok := index[c.Name()]; !ok {
			index[c.Name()] = i
		}
	}
	wanted := make([]int, len(columns))
	for i, name := range columns {
		idx, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("unexpected %s schema: missing column %q", table, name)
		}
		wanted[i] = idx
	}

	var rows [][]string
	err := db.VisitTableRecords(table, func(_ *int64, rec sqlite3.Record) error {
		row := make([]string, len(wanted))
		for i, idx := range wanted {
			if idx >= 0 && idx < len(rec.Values) {
				if s, ok := rec.Values[idx].(string); ok {
					row[i] = s
				}
			}
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	// Surface a close error (e.g. a failed flush) only when the copy itself
	// succeeded, so the more specific copy error is never masked.
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
