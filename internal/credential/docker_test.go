package credential

import (
	"bytes"
	"encoding/json"
	dockercredentials "github.com/docker/docker-credential-helpers/credentials"
	"strings"
	"testing"
)

func TestDockerWireFormats(t *testing.T) {
	original, key, err := ReadDockerServer(strings.NewReader("https://registry.example/v2/\n"))
	if err != nil || original != "https://registry.example/v2/" || key != "registry.example" {
		t.Fatal(original, key, err)
	}
	var out bytes.Buffer
	if err := HandleDocker(dockerTestHelper{}, dockercredentials.ActionGet, strings.NewReader(original), &out); err != nil {
		t.Fatal(err)
	}
	var response map[string]string
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response) != 3 || response["ServerURL"] != original || response["Username"] != "user" || response["Secret"] != " \n☃\u200btoken " {
		t.Fatalf("wrong Docker schema or altered credential: %q", out.String())
	}
	out.Reset()
	if err := HandleDocker(dockerTestHelper{}, dockercredentials.ActionList, strings.NewReader(""), &out); err != nil || out.String() != "{\"registry.example\":\"user\"}\n" {
		t.Fatal(out.String(), err)
	}
	for _, input := range []string{"", "registry.example\npassword=leaked", strings.Repeat("x", 65537)} {
		if _, _, err := ReadDockerServer(strings.NewReader(input)); err == nil {
			t.Fatal("accepted invalid Docker input")
		}
	}
}

type dockerTestHelper struct{}

func (dockerTestHelper) Add(*dockercredentials.Credentials) error { return ErrReadOnly }
func (dockerTestHelper) Delete(string) error                      { return ErrReadOnly }
func (dockerTestHelper) Get(string) (string, string, error)       { return "user", " \n☃\u200btoken ", nil }
func (dockerTestHelper) List() (map[string]string, error) {
	return map[string]string{"registry.example": "user"}, nil
}
