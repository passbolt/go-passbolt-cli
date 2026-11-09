package agent

import (
	"fmt"
	"strings"
)

// Requester is best-effort display context, never authorization evidence. Only
// executable names are inspected: process arguments may contain credentials.
func Requester(pid int) string {
	if pid <= 0 {
		return "unknown requester"
	}
	name, parent := processIdentity(pid)
	label := processLabel(pid, name)
	if parent > 0 && parent != pid {
		parentName, _ := processIdentity(parent)
		label += "; parent " + processLabel(parent, parentName)
	}
	return label
}
func processLabel(pid int, name string) string {
	if name == "" {
		name = "process"
	}
	return fmt.Sprintf("%s (PID %d)", Text(name), pid)
}

// ResourceApproval keeps the resource name first so long account/URI labels
// cannot hide it. Dialog backends sanitize all displayed text before rendering.
func ResourceApproval(account, command, requester, id, name string) (string, string) {
	title := "Passbolt resource approval"
	if name != "" {
		title += " — " + name
	} else {
		name = "(name unavailable)"
	}
	return title, strings.Join([]string{"Resource: " + name, "ID: " + id, "Account: " + account, "Command: " + command, "Requester: " + requester}, " | ")
}
