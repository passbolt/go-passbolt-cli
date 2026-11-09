package util

import (
	"net/http"
	"testing"
)

func TestInvocationProxyPolicyDoesNotLeak(t *testing.T) {
	defer func() { ProxyEnvironment = nil }()
	for _, test := range []struct {
		env  []string
		want string
	}{
		{[]string{"HTTPS_PROXY=http://proxy.example:8080"}, "http://proxy.example:8080"},
		{[]string{}, ""},
		{[]string{"HTTPS_PROXY=http://proxy.example:8080", "NO_PROXY=passbolt.example"}, ""},
	} {
		ProxyEnvironment = test.env
		client, err := GetHTTPClient()
		if err != nil {
			t.Fatal(err)
		}
		req, _ := http.NewRequest("GET", "https://passbolt.example", nil)
		proxy, err := client.Transport.(*http.Transport).Proxy(req)
		if err != nil {
			t.Fatal(err)
		}
		got := ""
		if proxy != nil {
			got = proxy.String()
		}
		if got != test.want {
			t.Fatalf("proxy=%q want %q", got, test.want)
		}
	}
}
