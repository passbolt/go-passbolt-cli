// Package credential implements standard credential protocols and URI matching.
package credential

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const maxRequest = 1 << 20

var ErrRequest = errors.New("invalid credential request")
var ErrLookup = errors.New("credential lookup failed; check Passbolt configuration, resource access and approval")
var ErrResponse = errors.New("resource cannot be represented as credentials")
var ErrNotFound = errors.New("credentials not found in native keychain")
var ErrReadOnly = errors.New("credential writes are not supported")

type Request struct{ Protocol, Host, Path, Username string }

func ReadRequest(input io.Reader) (Request, error) {
	var req Request
	var rawURL string
	seen := map[string]bool{}
	limited := &io.LimitedReader{R: input, N: maxRequest + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), 65536)
	for scanner.Scan() {
		line := scanner.Text()
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
			dest = &req.Path
		case "username":
			dest = &req.Username
		case "url":
			dest = &rawURL
		default:
			continue
		}
		if seen[key] {
			return Request{}, ErrRequest
		}
		seen[key] = true
		*dest = value
	}
	if scanner.Err() != nil || limited.N <= 0 {
		return Request{}, ErrRequest
	}
	if seen["url"] {
		if seen["protocol"] || seen["host"] || seen["path"] {
			return Request{}, ErrRequest
		}
		parsed, err := ParseURL(rawURL)
		if err != nil {
			return Request{}, err
		}
		if req.Username != "" && parsed.Username != "" && req.Username != parsed.Username {
			return Request{}, ErrRequest
		}
		if req.Username != "" {
			parsed.Username = req.Username
		}
		return parsed, nil
	}
	req.Protocol = strings.ToLower(req.Protocol)
	host, err := canonicalAuthority(req.Protocol, req.Host)
	if err != nil {
		return Request{}, err
	}
	req.Host = host
	path, err := cleanPath(req.Path)
	if err != nil {
		return Request{}, err
	}
	req.Path = path
	return req, nil
}

func ParseURL(value string) (Request, error) {
	if strings.ContainsAny(value, "\r\n\x00") {
		return Request{}, ErrRequest
	}
	u, err := url.Parse(value)
	if err != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(value, "#") {
		return Request{}, ErrRequest
	}
	if u.User != nil {
		if _, password := u.User.Password(); password {
			return Request{}, ErrRequest
		}
	}
	protocol := strings.ToLower(u.Scheme)
	host, err := canonicalAuthority(protocol, u.Host)
	if err != nil {
		return Request{}, err
	}
	path, err := cleanPath(strings.TrimPrefix(u.EscapedPath(), "/"))
	if err != nil {
		return Request{}, err
	}
	req := Request{Protocol: protocol, Host: host, Path: path}
	if u.User != nil {
		req.Username = u.User.Username()
		if strings.ContainsAny(req.Username, "\r\n\x00") {
			return Request{}, ErrRequest
		}
	}
	return req, nil
}
func cleanPath(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n\x00?#\\") {
		return "", ErrRequest
	}
	value = strings.Trim(value, "/")
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", ErrRequest
	}
	for _, part := range strings.Split(decoded, "/") {
		if part == "." || part == ".." {
			return "", ErrRequest
		}
	}
	return value, nil
}
func canonicalAuthority(protocol, host string) (string, error) {
	if protocol != "http" && protocol != "https" {
		return "", ErrRequest
	}
	u, err := url.Parse(protocol + "://" + host)
	if err != nil || u.User != nil || u.Hostname() == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery || strings.ContainsAny(host, "%?#\\") || strings.HasSuffix(host, ":") {
		return "", ErrRequest
	}
	name := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if name == "" {
		return "", ErrRequest
	}
	port := u.Port()
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", ErrRequest
		}
	}
	if (protocol == "https" && port == "443") || (protocol == "http" && port == "80") {
		port = ""
	}
	if port != "" {
		return net.JoinHostPort(name, port), nil
	}
	if strings.Contains(name, ":") {
		name = "[" + name + "]"
	}
	return name, nil
}
func (r Request) Encode() string {
	value := "protocol=" + r.Protocol + "\nhost=" + r.Host + "\n"
	if r.Path != "" {
		value += "path=" + r.Path + "\n"
	}
	if r.Username != "" {
		value += "username=" + r.Username + "\n"
	}
	return value + "\n"
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
