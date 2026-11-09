package credential

import (
	"context"
	"errors"
	"testing"
)

type testStore struct {
	records []Record
	current *Record
	reads   int
}

func (s *testStore) Records(context.Context) ([]Record, error) { return s.records, nil }
func (s *testStore) Read(_ context.Context, id string) (Record, string, error) {
	s.reads++
	if s.current != nil {
		return *s.current, "secret", nil
	}
	for _, r := range s.records {
		if r.ID == id {
			return r, "secret", nil
		}
	}
	return Record{}, "", ErrNotFound
}
func TestAmbiguousLookupRequiresSelectionBeforeReadingSecret(t *testing.T) {
	records := []Record{{ID: "1", Username: "alice", URLs: []string{"https://git.example/team"}}, {ID: "2", Username: "bob", URLs: []string{"https://git.example/team"}}}
	store := &testStore{records: records}
	req, _ := ParseURL("https://git.example/team/repo")
	canceled := errors.New("canceled")
	resolver := Resolver{Store: store, Select: func(context.Context, []Match) (int, error) { return -1, canceled }}
	if _, _, err := resolver.Git(context.Background(), req); !errors.Is(err, canceled) || store.reads != 0 {
		t.Fatal("ambiguous lookup read a password before selection")
	}
	resolver.Select = func(_ context.Context, m []Match) (int, error) {
		for i, match := range m {
			if match.Username == "bob" {
				return i, nil
			}
		}
		t.Fatal("missing choice")
		return -1, nil
	}
	user, password, err := resolver.Git(context.Background(), req)
	if err != nil || user != "bob" || password != "secret" || store.reads != 1 {
		t.Fatalf("selection: %q %q %v", user, password, err)
	}
	changed := records[1]
	changed.Username = "other-account"
	store.current = &changed
	if _, _, err := resolver.Git(context.Background(), req); err == nil {
		t.Fatal("account changed after selection but password was released")
	}
	changed.Username = records[1].Username
	changed.URLs = []string{"https://elsewhere.example"}
	store.current = &changed
	if _, _, err := resolver.Git(context.Background(), req); err == nil {
		t.Fatal("URI changed after matching but password was released")
	}
}
func TestDockerListDoesNotReadSecrets(t *testing.T) {
	store := &testStore{records: []Record{{ID: "one", Username: "user", URLs: []string{"https://registry.example/v2/", "https://git.example/team/repo"}}, {ID: "two", Username: "user", URLs: []string{"https://registry.example"}}}}
	result, err := (Resolver{Store: store}).DockerList(context.Background())
	if err != nil || result["registry.example"] != "user" || len(result) != 1 || store.reads != 0 {
		t.Fatalf("Docker list: %+v %v reads=%d", result, err, store.reads)
	}
}
