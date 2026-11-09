package credential

import (
	"context"
	"sort"
	"strings"
)

type Store interface {
	Records(context.Context) ([]Record, error)
	Read(context.Context, string) (Record, string, error)
}
type Resolver struct {
	Store  Store
	Select func(context.Context, []Match) (int, error)
}

func (r Resolver) choose(ctx context.Context, matches []Match) (Match, error) {
	if len(matches) == 0 {
		return Match{}, ErrNotFound
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if r.Select == nil {
		return Match{}, ErrLookup
	}
	at, err := r.Select(ctx, matches)
	if err != nil {
		return Match{}, err
	}
	if at < 0 || at >= len(matches) {
		return Match{}, ErrLookup
	}
	return matches[at], nil
}
func (r Resolver) Git(ctx context.Context, req Request) (string, string, error) {
	records, err := r.Store.Records(ctx)
	if err != nil {
		return "", "", err
	}
	selected, err := r.choose(ctx, FindMatches(req, records))
	if err != nil {
		return "", "", err
	}
	current, password, err := r.Store.Read(ctx, selected.ID)
	if err != nil {
		return "", "", err
	}
	// Revalidate metadata after reading: changed URI/username must not release a
	// newly edited credential to a requester matched against an older listing.
	if current.ID != selected.ID || current.Username != selected.Username || len(FindMatches(req, []Record{current})) != 1 {
		return "", "", ErrLookup
	}
	username := current.Username
	if username == "" {
		username = req.Username
	}
	return username, password, nil
}
func (r Resolver) Docker(ctx context.Context, key string) (string, string, error) {
	records, err := r.Store.Records(ctx)
	if err != nil {
		return "", "", err
	}
	selected, err := r.choose(ctx, DockerMatches(key, records))
	if err != nil {
		return "", "", err
	}
	current, password, err := r.Store.Read(ctx, selected.ID)
	if err != nil {
		return "", "", err
	}
	if current.ID != selected.ID || current.Username != selected.Username || len(DockerMatches(key, []Record{current})) != 1 || current.Username == "" {
		return "", "", ErrLookup
	}
	return current.Username, password, nil
}
func (r Resolver) DockerList(ctx context.Context) (map[string]string, error) {
	records, err := r.Store.Records(ctx)
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	for _, record := range records {
		if record.Username != "" {
			for _, raw := range record.URLs {
				if key, err := DockerRegistryKey(raw); err == nil {
					keys[key] = true
				}
			}
		}
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	result := make(map[string]string, len(ordered))
	for _, key := range ordered {
		matches := DockerMatches(key, records)
		filtered := matches[:0]
		for _, match := range matches {
			if match.Username != "" {
				filtered = append(filtered, match)
			}
		}
		matches = filtered
		if len(matches) == 0 {
			return nil, ErrLookup
		}
		same := true
		for _, match := range matches[1:] {
			if match.Username != matches[0].Username {
				same = false
			}
		}
		selected := matches[0]
		if !same {
			selected, err = r.choose(ctx, matches)
			if err != nil {
				return nil, err
			}
		}
		result[key] = selected.Username
	}
	return result, nil
}
func MatchLabel(match Match) string {
	return strings.Join(match.URLs, " | ") + " — " + match.Username + " — " + match.Name + " [" + match.ID + "]"
}
