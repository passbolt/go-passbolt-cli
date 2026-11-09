package agent

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestRequesterIncludesProcessAndParentWithoutArguments(t *testing.T) {
	text := Requester(os.Getpid())
	if !strings.Contains(text, strconv.Itoa(os.Getpid())) || !strings.Contains(text, strconv.Itoa(os.Getppid())) || !strings.Contains(text, "parent") {
		t.Fatalf("missing requester details: %q", text)
	}
	if text := Requester(-1); text != "unknown requester" {
		t.Fatalf("invalid pid: %q", text)
	}
}
func TestResourceApprovalDetails(t *testing.T) {
	name := strings.Repeat("🌐", 255) + "final-title"
	title, prompt := ResourceApproval("https://vault.example — user", "passbolt get resource", "git (PID 123)", "uuid", name)
	if !strings.Contains(title, name) || !strings.Contains(prompt, name) || !strings.Contains(prompt, "uuid") || !strings.Contains(prompt, "git (PID 123)") {
		t.Fatal("resource identity missing from approval")
	}
	if strings.Contains(Text(prompt+"\x1b"), "\x1b") {
		t.Fatal("unsafe approval text")
	}
}
