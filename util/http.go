package util

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

// headerRoundTripper injects user-configured HTTP headers into every request
// before delegating to the wrapped RoundTripper. This lets the CLI reach
// Passbolt instances that sit behind an authenticating reverse proxy (for
// example oauth2-proxy or Cloudflare Access), which require a session Cookie or
// service-token header before the request reaches Passbolt itself.
type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request so we don't mutate the caller's request, as the
	// RoundTripper contract requires.
	req = req.Clone(req.Context())
	for name, value := range h.headers {
		// The Passbolt SDK manages its own session and CSRF cookies via the
		// Cookie header, so append rather than overwrite to keep both.
		if strings.EqualFold(name, "Cookie") {
			if existing := req.Header.Get("Cookie"); existing != "" {
				req.Header.Set("Cookie", existing+"; "+value)
				continue
			}
		}
		req.Header.Set(name, value)
	}
	return h.base.RoundTrip(req)
}

// GetExtraHeaders parses the configured "headers" entries, each in
// "Name: Value" format, into a map.
func GetExtraHeaders() (map[string]string, error) {
	entries := viper.GetStringSlice("headers")
	if len(entries) == 0 {
		return nil, nil
	}
	headers := make(map[string]string, len(entries))
	for _, entry := range entries {
		name, value, found := strings.Cut(entry, ":")
		if !found {
			return nil, fmt.Errorf("invalid header %q, expected \"Name: Value\" format", entry)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("invalid header %q, header name is empty", entry)
		}
		headers[name] = strings.TrimSpace(value)
	}
	return headers, nil
}

func GetClientCertificate() (tls.Certificate, error) {
	cert := viper.GetString("tlsClientCert")
	certExists := cert != ""
	key := viper.GetString("tlsClientPrivateKey")
	keyExists := key != ""
	if !certExists && !keyExists {
		return tls.Certificate{}, nil
	}
	if certExists && !keyExists {
		return tls.Certificate{}, fmt.Errorf("client TLS private key is empty, but client TLS cert was set")
	}
	if !certExists && keyExists {
		return tls.Certificate{}, fmt.Errorf("client TLS cert is empty, but client TLS private key was set")
	}
	return tls.X509KeyPair([]byte(cert), []byte(key))
}

func GetHTTPClient() (*http.Client, error) {
	tlsSkipVerify := viper.GetBool("tlsSkipVerify")
	cert, err := GetClientCertificate()
	if err != nil {
		return nil, err
	}
	var transport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: tlsSkipVerify,
		},
	}

	headers, err := GetExtraHeaders()
	if err != nil {
		return nil, err
	}

	ssoCookie, err := GetSSOCookie()
	if err != nil {
		return nil, err
	}
	if ssoCookie != "" {
		if headers == nil {
			headers = map[string]string{}
		}
		// Merge into any Cookie header the user also set via --header.
		key := "Cookie"
		for k := range headers {
			if strings.EqualFold(k, "Cookie") {
				key = k
				break
			}
		}
		if existing := headers[key]; existing != "" {
			headers[key] = existing + "; " + ssoCookie
		} else {
			headers[key] = ssoCookie
		}
	}

	if len(headers) > 0 {
		transport = &headerRoundTripper{base: transport, headers: headers}
	}

	httpClient := http.Client{
		Transport: transport,
	}

	return &httpClient, nil
}
