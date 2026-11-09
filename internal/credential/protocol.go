// Package credential implements the read-only Git credential helper protocol.
package credential

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strings"
)

const maxRequest = 1 << 20

var resourceUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var ErrRequest = errors.New("invalid credential request: protocol, host and a resource UUID path are required")
var ErrLookup = errors.New("credential lookup failed; check Passbolt configuration, resource access and approval")
var ErrResponse = errors.New("resource cannot be represented as Git credentials")

type Request struct{ Protocol, Host, ResourceID, Username string }

func ReadRequest(input io.Reader) (Request, error) {
	var req Request
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(io.LimitReader(input, maxRequest+1))
	scanner.Buffer(make([]byte, 4096), 65536)
	size := 0
	for scanner.Scan() {
		line := scanner.Text()
		size += len(line) + 1
		if size > maxRequest {
			return Request{}, ErrRequest
		}
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" || strings.ContainsAny(line, "\r\x00") {
			return Request{}, ErrRequest
		}
		var dest *string
		switch key {
		case "protocol":
			dest = &req.Protocol
		case "host":
			dest = &req.Host
		case "path":
			dest = &req.ResourceID
		case "username":
			dest = &req.Username
		default:
			continue
		}
		if seen[key] {
			return Request{}, ErrRequest
		}
		seen[key] = true
		*dest = value
	}
	if scanner.Err() != nil || strings.TrimSpace(req.Protocol) == "" || strings.TrimSpace(req.Host) == "" || !resourceUUID.MatchString(req.ResourceID) {
		return Request{}, ErrRequest
	}
	req.ResourceID = strings.ToLower(req.ResourceID)
	return req, nil
}

// Encode replays only validated lookup fields to the authenticated agent.
func (r Request) Encode() string {
	result := "protocol=" + r.Protocol + "\nhost=" + r.Host + "\npath=" + r.ResourceID + "\n"
	if r.Username != "" {
		result += "username=" + r.Username + "\n"
	}
	return result + "\n"
}

// WriteResponse validates the complete response before writing any credentials.
// Credential bytes bypass terminal sanitization: this is a machine protocol.
func WriteResponse(output io.Writer, username, password string) error {
	if strings.TrimSpace(password) == "" || strings.ContainsAny(username+password, "\r\n\x00") {
		return ErrResponse
	}
	response := ""
	if username != "" {
		response = "username=" + username + "\n"
	}
	response += "password=" + password + "\n\n"
	_, err := io.WriteString(output, response)
	return err
}
