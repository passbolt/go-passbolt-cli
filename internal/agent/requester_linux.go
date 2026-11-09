package agent

import (
	"os"
	"strconv"
	"strings"
)

func processIdentity(pid int) (string, int) {
	base := "/proc/" + strconv.Itoa(pid) + "/"
	comm, _ := os.ReadFile(base + "comm")
	status, _ := os.ReadFile(base + "status")
	parent := 0
	for _, line := range strings.Split(string(status), "\n") {
		if value, ok := strings.CutPrefix(line, "PPid:"); ok {
			parent, _ = strconv.Atoi(strings.TrimSpace(value))
			break
		}
	}
	return strings.TrimSuffix(string(comm), "\n"), parent
}
