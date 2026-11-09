package credential

import (
	"net/url"
	"sort"
	"strings"
)

// Record contains only searchable resource metadata, never a decrypted password.
type Record struct {
	ID, Name, Username string
	URLs               []string
}
type Match struct {
	Record
	URLs []string
}

func FindMatches(req Request, records []Record) []Match {
	result := []Match{}
	seen := map[string]int{}
	for _, record := range records {
		if req.Username != "" && record.Username != "" && req.Username != record.Username {
			continue
		}
		urls := []string{}
		for _, raw := range record.URLs {
			stored, err := ParseURL(raw)
			if err != nil {
				continue
			}
			if stored.Username != "" && record.Username != "" && stored.Username != record.Username {
				continue
			}
			if stored.Protocol != req.Protocol || stored.Host != req.Host {
				continue
			}
			if req.Username != "" && stored.Username != "" && req.Username != stored.Username {
				continue
			}
			if req.Path != "" && stored.Path != "" && req.Path != stored.Path && !strings.HasPrefix(req.Path, stored.Path+"/") {
				continue
			}
			urls = append(urls, raw)
		}
		if len(urls) == 0 {
			continue
		}
		sort.Strings(urls)
		if at, exists := seen[record.ID]; exists {
			result[at].URLs = append(result[at].URLs, urls...)
			continue
		}
		seen[record.ID] = len(result)
		result = append(result, Match{Record: record, URLs: urls})
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		return strings.Join(a.URLs, " ")+a.Username+a.Name+a.ID < strings.Join(b.URLs, " ")+b.Username+b.Name+b.ID
	})
	return result
}

// DockerRegistryKey follows Docker's origin-oriented server keys. Paths that
// identify repositories/web pages are not registry entries for Docker list.
func DockerRegistryKey(value string) (string, error) {
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsed, err := ParseURL(value)
	if err != nil {
		return "", err
	}
	if parsed.Username != "" {
		return "", ErrRequest
	}
	u, err := url.Parse(value)
	if err != nil {
		return "", ErrRequest
	}
	host, err := canonicalAuthority("https", u.Host)
	if err != nil {
		return "", err
	}
	switch host {
	case "docker.io", "index.docker.io", "registry-1.docker.io":
		if parsed.Path == "" || parsed.Path == "v1" || parsed.Path == "v2" {
			return "https://index.docker.io/v1/", nil
		}
	}
	if parsed.Path != "" && parsed.Path != "v2" {
		return "", ErrRequest
	}
	return host, nil
}
func DockerMatches(key string, records []Record) []Match {
	matches := []Match{}
	seen := map[string]bool{}
	for _, record := range records {
		if seen[record.ID] {
			continue
		}
		urls := []string{}
		for _, raw := range record.URLs {
			if stored, err := DockerRegistryKey(raw); err == nil && stored == key {
				urls = append(urls, raw)
			}
		}
		if len(urls) > 0 {
			sort.Strings(urls)
			matches = append(matches, Match{Record: record, URLs: urls})
			seen[record.ID] = true
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Username+matches[i].Name+matches[i].ID < matches[j].Username+matches[j].Name+matches[j].ID
	})
	return matches
}
