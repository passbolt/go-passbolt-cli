package credential

import "testing"

func TestURLMatchingAndCandidateDeduplication(t *testing.T) {
	records := []Record{
		{ID: "one", Name: "alice", Username: "alice", URLs: []string{"https://git.example/team", "https://git.example/team/repo.git"}},
		{ID: "two", Username: "bob", URLs: []string{"https://git.example/team"}},
		{ID: "bad-host", URLs: []string{"https://git.example.attacker/team"}},
		{ID: "bad-path", URLs: []string{"https://git.example/teammate"}},
		{ID: "bad-scheme", URLs: []string{"http://git.example/team"}},
	}
	req, err := ParseURL("https://git.example/team/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	matches := FindMatches(req, records)
	if len(matches) != 2 {
		t.Fatalf("matches: %+v", matches)
	}
	req.Username = "alice"
	matches = FindMatches(req, records)
	if len(matches) != 1 || matches[0].ID != "one" {
		t.Fatalf("username/dedup: %+v", matches)
	}
	req, err = ParseURL("https://GIT.example:443/team/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if len(FindMatches(req, records)) != 2 {
		t.Fatal("default port or host case changed matching")
	}
	req.Path = ""
	if len(FindMatches(req, records)) != 3 {
		t.Fatal("host-only Git request did not include same-host path credentials")
	}
}
func TestDockerRegistryKeys(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"registry.example.com", "registry.example.com"},
		{"https://registry.example.com/v2/", "registry.example.com"},
		{"registry.example.com:5000", "registry.example.com:5000"},
		{"https://index.docker.io/v1/", "https://index.docker.io/v1/"},
		{"registry-1.docker.io", "https://index.docker.io/v1/"},
		{"https://git.example/repo.git", ""},
		{"https://user:secret@registry.example.com", ""},
	} {
		got, err := DockerRegistryKey(test.input)
		if test.want == "" {
			if err == nil {
				t.Fatalf("accepted non-registry URI %s", test.input)
			}
		} else if err != nil || got != test.want {
			t.Fatalf("%s: %q %v", test.input, got, err)
		}
	}
}
