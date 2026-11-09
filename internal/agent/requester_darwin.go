package agent

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func processIdentity(pid int) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "/bin/ps", "-p", strconv.Itoa(pid), "-o", "ppid=", "-o", "comm=").Output()
	if err != nil {
		return "", 0
	}
	text := strings.TrimSpace(string(data))
	at := strings.IndexAny(text, " \t")
	if at < 0 {
		return "", 0
	}
	parent, _ := strconv.Atoi(text[:at])
	return strings.TrimSpace(text[at:]), parent
}
