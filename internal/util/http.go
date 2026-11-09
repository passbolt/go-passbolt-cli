package util

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http/httpproxy"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

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

// ProxyEnvironment is nil for ordinary commands and a request-local environment
// for agent calls. A fresh proxy function avoids net/http's global env cache.
var ProxyEnvironment []string

func GetHTTPClient() (*http.Client, error) {
	tlsSkipVerify := viper.GetBool("tlsSkipVerify")
	cert, err := GetClientCertificate()
	if err != nil {
		return nil, err
	}
	proxy := http.ProxyFromEnvironment
	if ProxyEnvironment != nil {
		env := make(map[string]string)
		for _, entry := range ProxyEnvironment {
			if key, value, ok := strings.Cut(entry, "="); ok {
				env[key] = value
			}
		}
		first := func(upper, lower string) string {
			if env[upper] != "" {
				return env[upper]
			}
			return env[lower]
		}
		config := httpproxy.Config{HTTPProxy: first("HTTP_PROXY", "http_proxy"), HTTPSProxy: first("HTTPS_PROXY", "https_proxy"), NoProxy: first("NO_PROXY", "no_proxy"), CGI: env["REQUEST_METHOD"] != ""}
		resolve := config.ProxyFunc()
		proxy = func(req *http.Request) (*url.URL, error) { return resolve(req.URL) }
	}
	httpClient := http.Client{
		Transport: &http.Transport{
			Proxy: proxy,
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: tlsSkipVerify,
			},
		},
	}

	return &httpClient, nil
}
