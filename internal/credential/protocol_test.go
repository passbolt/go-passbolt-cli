package credential

import (
	"bytes"
	"strings"
	"testing"
)

const id = "01234567-89ab-cdef-0123-456789abcdef"

func TestReadRequest(t *testing.T) {
	req, err := ReadRequest(strings.NewReader("protocol=https\r\nhost=api.hetzner.cloud\r\npath=" + id + "\r\nusername=token\r\ncapability[]=unknown\r\n\r\n"))
	if err != nil || req.Path != id || req.Username != "token" {
		t.Fatalf("%+v %v", req, err)
	}
	for _, data := range []string{
		"protocol=https\nhost=example.com/invalid\n\n",
		"protocol=https\nhost=example.com\npath=../secret\n\n",
		"protocol=ssh\nhost=example.com\npath=" + id + "\n\n",
		"protocol=https\nhost=example.com\npath=" + id + "\npath=" + id + "\n\n",
		"protocol=https\nhost=bad\x00host\npath=" + id + "\n\n",
		"broken-field\n\n", strings.Repeat("x", 65537),
	} {
		if _, err := ReadRequest(strings.NewReader(data)); err == nil {
			t.Fatalf("accepted invalid request %q", data[:min(80, len(data))])
		}
	}
}
func TestWriteResponsePreservesCredentialBytes(t *testing.T) {
	var output bytes.Buffer
	if err := WriteResponse(&output, "user", "a=☃\u200bb"); err != nil {
		t.Fatal(err)
	}
	if output.String() != "username=user\npassword=a=☃\u200bb\n\n" {
		t.Fatalf("credential bytes changed: %q", output.String())
	}
	output.Reset()
	if err := WriteResponse(&output, "", "token"); err != nil || output.String() != "password=token\n\n" {
		t.Fatalf("token-only response: %q %v", output.String(), err)
	}
	for _, value := range []string{"", "  ", "secret\nusername=injected", "secret\rvalue", "secret\x00value"} {
		output.Reset()
		if err := WriteResponse(&output, "user", value); err == nil || output.Len() != 0 {
			t.Fatalf("unsafe password produced output: %q", output.String())
		}
	}
	output.Reset()
	if err := WriteResponse(&output, "bad\nuser", "token"); err == nil || output.Len() != 0 {
		t.Fatal("username injection was accepted")
	}
}
