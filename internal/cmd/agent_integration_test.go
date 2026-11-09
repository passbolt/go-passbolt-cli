//go:build !windows

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/passbolt/go-passbolt-cli/internal/agent"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/viper"
)

// Exercises real GPG authentication, the SDK's MFA callback and CLI handlers.
func TestAgentAuthenticationReuseApprovalAndExpiry(t *testing.T) {
	oldOut, oldErr := util.Out, util.Err
	oldAgent := inAgent
	inAgent = true
	defer func() { inAgent = oldAgent; util.Out = oldOut; util.Err = oldErr; viper.Reset(); newCommandTree() }()
	pgp := crypto.PGP()
	key, err := pgp.KeyGeneration().AddUserId("Agent test", "agent@example.invalid").New().GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	defer key.ClearPrivateParams()
	locked, err := pgp.LockKey(key, []byte("test-passphrase"))
	if err != nil {
		t.Fatal(err)
	}
	armor, err := locked.Armor()
	if err != nil {
		t.Fatal(err)
	}
	cryptoClient, err := api.NewClient(nil, "", "https://example.invalid", armor, "test-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cryptoClient.EncryptMessage("gpgauthv1.3.0|4|test|gpgauthv1.3.0")
	if err != nil {
		t.Fatal(err)
	}
	credentialSecret, err := cryptoClient.EncryptMessage(`{"password":"hcloud-smoke-token","description":""}`)
	if err != nil {
		t.Fatal(err)
	}

	var logins, mfas, logouts, writes atomic.Int32
	var valid atomic.Bool
	var failWrite atomic.Bool
	respond := func(w http.ResponseWriter, body any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"header": map[string]any{"status": "success", "code": 200}, "body": body})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login.json":
			var body api.Login
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if body.Auth.Token == "" {
				w.Header().Set("X-GPGAuth-User-Auth-Token", url.QueryEscape(encrypted))
			} else {
				logins.Add(1)
				valid.Store(true)
				http.SetCookie(w, &http.Cookie{Name: "passbolt_session", Value: "session"})
			}
			respond(w, nil)
		case "/auth/is-authenticated.json":
			if !valid.Load() {
				json.NewEncoder(w).Encode(map[string]any{"header": map[string]any{"status": "error", "code": 401}})
				return
			}
			respond(w, nil)
		case "/users/me.json":
			http.SetCookie(w, &http.Cookie{Name: "csrfToken", Value: "csrf"})
			if _, err := r.Cookie("passbolt_mfa"); err != nil {
				json.NewEncoder(w).Encode(map[string]any{"header": map[string]any{"status": "error", "code": 403, "url": "/mfa/verify/error.json"}, "body": map[string]any{"providers": map[string]any{"totp": "totp"}}})
				return
			}
			respond(w, map[string]any{"id": "11111111-1111-1111-1111-111111111111"})
		case "/mfa/verify/totp.json":
			var challenge api.MFAChallengeResponse
			json.NewDecoder(r.Body).Decode(&challenge)
			if challenge.TOTP != "123456" {
				t.Errorf("unexpected TOTP %q", challenge.TOTP)
			}
			mfas.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "passbolt_mfa", Value: "mfa"})
			respond(w, nil)
		case "/settings.json":
			respond(w, map[string]any{"passbolt": map[string]any{"plugins": map[string]any{}}})
		case "/resources/01234567-89ab-cdef-0123-456789abcdef.json":
			respond(w, map[string]any{"id": "01234567-89ab-cdef-0123-456789abcdef", "resource_type_id": "22222222-2222-2222-2222-222222222222", "name": "hcloud token", "username": "token"})
		case "/resource-types/22222222-2222-2222-2222-222222222222.json":
			respond(w, map[string]any{"id": "22222222-2222-2222-2222-222222222222", "slug": "password-and-description"})
		case "/secrets/resource/01234567-89ab-cdef-0123-456789abcdef.json":
			respond(w, map[string]any{"resource_id": "01234567-89ab-cdef-0123-456789abcdef", "data": credentialSecret})

		case "/users.json":
			respond(w, []any{})
		case "/users/11111111-1111-1111-1111-111111111111.json":
			writes.Add(1)
			if failWrite.Load() {
				json.NewEncoder(w).Encode(map[string]any{"header": map[string]any{"status": "error", "code": 500, "message": "ambiguous write failure"}})
				return
			}
			respond(w, nil)
		case "/auth/logout.json":
			logouts.Add(1)
			valid.Store(false)
			respond(w, nil)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	log := filepath.Join(dir, "prompts")
	helper := filepath.Join(dir, "pinentry")
	script := `#!/bin/sh
printf 'OK ready\n'
while IFS= read -r line; do
 case "$line" in
 SETDESC*) desc="$line"; printf 'OK\n';;
 CONFIRM) printf 'confirm\n' >> "$TEST_PROMPT_LOG"; if [ "$TEST_DENY" = 1 ]; then printf 'ERR 1 denied\n'; else printf 'OK\n'; fi;;
 GETPIN) case "$desc" in *TOTP*) printf 'totp\n' >> "$TEST_PROMPT_LOG"; printf 'D 123456\nOK\n';; *) printf 'password\n' >> "$TEST_PROMPT_LOG"; printf 'D test-passphrase\nOK\n';; esac;;
 *) printf 'OK\n';;
 esac
done
`
	if err := os.WriteFile(helper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	req := agent.Request{Version: agent.ProtocolVersion, Action: "run", Args: []string{"list", "user", "--json"}, Settings: map[string]any{"serverAddress": server.URL, "userPrivateKey": armor, "mfaMode": "interactive-totp", "timeout": "5s", "workers": 1}, Directory: dir, Environment: append(os.Environ(), "TEST_PROMPT_LOG="+log), Dialog: "pinentry", Pinentry: helper}
	manager := newCommandAgent()
	now := time.Now()
	manager.sessions = agent.NewSessions(func() time.Time { return now })
	defer manager.Close(context.Background())
	run := func(request agent.Request) (string, error) {
		var out, stderr bytes.Buffer
		_, err := manager.Run(context.Background(), request, strings.NewReader(""), &out, &stderr)
		return out.String(), err
	}
	for range 2 {
		output, err := run(req)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output, "[]") {
			t.Fatalf("unexpected JSON output %q", output)
		}
	}
	if logins.Load() != 1 || mfas.Load() != 1 {
		t.Fatalf("session not reused: login=%d MFA=%d", logins.Load(), mfas.Load())
	}
	prompts, _ := os.ReadFile(log)
	if strings.Count(string(prompts), "confirm") != 2 || strings.Count(string(prompts), "password") != 1 || strings.Count(string(prompts), "totp") != 1 {
		t.Fatalf("unexpected prompts: %s", prompts)
	}
	denied := req
	denied.Environment = append(append([]string(nil), req.Environment...), "TEST_DENY=1")
	now = now.Add(30 * time.Minute)
	if _, err := run(denied); err == nil {
		t.Fatal("denied invocation executed")
	}
	now = now.Add(30 * time.Minute)
	manager.Expire(context.Background())
	if len(manager.clients) != 0 || logouts.Load() != 1 {
		t.Fatal("denied use extended idle lease")
	}
	if _, err := run(req); err != nil {
		t.Fatal(err)
	}
	if logins.Load() != 2 || mfas.Load() != 2 {
		t.Fatal("expired session did not require fresh password and MFA")
	}
	// Revoking the server session causes reauthentication before user code.
	valid.Store(false)
	if _, err := run(req); err != nil {
		t.Fatal(err)
	}
	if logins.Load() != 3 {
		t.Fatal("revoked server session was reused")
	}
	write := req
	write.Args = []string{"delete", "user", "--id", "11111111-1111-1111-1111-111111111111"}
	failWrite.Store(true)
	if _, err := run(write); err == nil {
		t.Fatal("write failure disappeared")
	}
	if writes.Load() != 1 {
		t.Fatalf("write was replayed %d times", writes.Load())
	}
	// A change in transport policy must not borrow the already-unlocked client.
	isolated := req
	isolated.Settings = maps.Clone(req.Settings)
	isolated.Settings["tlsSkipVerify"] = true
	beforeLogin := logins.Load()
	if _, err := run(isolated); err != nil {
		t.Fatal(err)
	}
	if len(manager.clients) != 2 || logins.Load() != beforeLogin+1 {
		t.Fatal("different TLS policy reused a cached client")
	}

	_, err = manager.Control(context.Background(), agent.Request{Action: "lock", Settings: req.Settings, Environment: req.Environment})
	if err != nil {
		t.Fatal(err)
	}
	if len(manager.clients) != 1 {
		t.Fatal("lock did not preserve the other policy's session")
	}
	if _, err := manager.Control(context.Background(), agent.Request{Action: "lock", Settings: isolated.Settings, Environment: isolated.Environment}); err != nil {
		t.Fatal(err)
	}
	if len(manager.clients) != 0 {
		t.Fatal("second lock did not clear isolated session")
	}
	if got := viper.GetString("userPrivateKey"); got != "" {
		t.Fatal("request configuration retained private key")
	}
	beforeCLILogin, beforeCLIMFA := logins.Load(), mfas.Load()
	testAgentBinary(t, req)
	if logins.Load() != beforeCLILogin+1 || mfas.Load() != beforeCLIMFA+1 {
		t.Fatal("competing CLI processes did not reuse one login and MFA")
	}

	// The pinned SDK normally skips private-key cleanup on failed remote logout.
	if _, err := run(req); err != nil {
		t.Fatal(err)
	}
	var retained *api.Client
	for _, cached := range manager.clients {
		retained = cached.client
	}
	server.Close()
	if _, err := manager.Control(context.Background(), agent.Request{Action: "lock", Settings: req.Settings, Environment: req.Environment}); err != nil {
		t.Fatal(err)
	}
	if _, err := retained.EncryptMessage("must be locked"); !errors.Is(err, api.ErrNoPrivateKey) {
		t.Fatalf("offline lock retained decrypted private key: %v", err)
	}

}

func testAgentBinary(t *testing.T, req agent.Request) {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "passbolt")
	build := exec.Command("go", "build", "-o", binary, "../..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building CLI: %v: %s", err, output)
	}
	// Keep Unix socket names below sockaddr_un's path limit.
	runtimeDir, err := os.MkdirTemp("", "pb-agent-cli-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(runtimeDir)
	config := filepath.Join(runtimeDir, "config.json")
	settings, _ := json.Marshal(req.Settings)
	if err := os.WriteFile(config, settings, 0600); err != nil {
		t.Fatal(err)
	}
	command := func(args ...string) *exec.Cmd {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		t.Cleanup(cancel)
		argv := append([]string{"--config", config}, args...)
		cmd := exec.CommandContext(ctx, binary, argv...)
		cmd.Env = append(append([]string{}, req.Environment...), "TMPDIR="+runtimeDir)
		return cmd
	}
	defer func() { _ = command("agent", "stop").Run() }()
	type reply struct {
		out []byte
		err error
	}
	results := make(chan reply, 2)
	// Competing foreground clients must converge on one daemon and one login.
	for range 2 {
		go func() {
			out, err := command("--agent", "--agent-dialog", "pinentry", "--agent-pinentry", req.Pinentry, "list", "user", "--json").CombinedOutput()
			results <- reply{out, err}
		}()
	}
	for range 2 {
		result := <-results
		if result.err != nil {
			t.Fatalf("CLI agent invocation: %v: %s", result.err, result.out)
		}
		if strings.TrimSpace(string(result.out)) != "[]" {
			t.Fatalf("CLI JSON corrupted: %q", result.out)
		}
		if strings.Contains(string(result.out), "test-passphrase") {
			t.Fatal("CLI leaked authentication material")
		}
	}
	child, err := command("--agent", "--agent-dialog", "pinentry", "--agent-pinentry", req.Pinentry, "exec", "--", "/bin/sh", "-c", "printf exec-ok").CombinedOutput()
	if err != nil || string(child) != "exec-ok" {
		t.Fatalf("foreground exec: %v %q", err, child)
	}

	const lookup = "protocol=https\nhost=api.hetzner.cloud\npath=01234567-89ab-cdef-0123-456789abcdef\n\n"
	read := command("--agent", "--agent-dialog", "pinentry", "--agent-pinentry", req.Pinentry, "git-credential", "get")
	read.Stdin = strings.NewReader(lookup)
	credentials, err := read.Output()
	if err != nil || string(credentials) != "username=token\npassword=hcloud-smoke-token\n\n" {
		t.Fatalf("agent credential protocol: %v %q", err, credentials)
	}
	denied := command("--agent", "--agent-dialog", "pinentry", "--agent-pinentry", req.Pinentry, "git-credential", "get")
	denied.Env = append(denied.Env, "TEST_DENY=1")
	denied.Stdin = strings.NewReader(lookup)
	rejected, err := denied.Output()
	if err == nil || len(rejected) != 0 {
		t.Fatalf("denied lookup released credentials: %v %q", err, rejected)
	}
	// Git itself appends the operation and consumes the helper's protocol output.
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	gitHelper := "!" + quote(binary) + " --config " + quote(config) + " --agent --agent-dialog pinentry --agent-pinentry " + quote(req.Pinentry) + " git-credential"
	gitArgs := []string{"-c", "credential.helper=", "-c", "credential.helper=" + gitHelper, "-c", "credential.useHttpPath=true", "credential", "fill"}
	git := exec.Command("git", gitArgs...)
	git.Env = append(append([]string{}, read.Env...), "GIT_TERMINAL_PROMPT=0")
	git.Stdin = strings.NewReader(lookup)
	filled, err := git.Output()
	if err != nil || !strings.Contains(string(filled), "password=hcloud-smoke-token\n") {
		t.Fatalf("git credential fill: %v %q", err, filled)
	}
	for _, operation := range []string{"store", "erase"} {
		noop := command("--agent", "--agent-dialog", "pinentry", "--agent-pinentry", "missing-helper", "git-credential", operation)
		noop.Stdin = strings.NewReader("password=ignored\n\n")
		if output, err := noop.CombinedOutput(); err != nil || len(output) != 0 {
			t.Fatalf("%s was not a no-op: %v %q", operation, err, output)
		}
	}

	if hcloudBinary := os.Getenv("PASSBOLT_TEST_HCLOUD_BINARY"); hcloudBinary != "" {
		var cloudRequests atomic.Int32
		cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/servers" {
				t.Errorf("unexpected hcloud endpoint %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			cloudRequests.Add(1)
			if r.Header.Get("Authorization") != "Bearer hcloud-smoke-token" {
				t.Error("hcloud did not use the Passbolt token")
			}
			json.NewEncoder(w).Encode(map[string]any{"servers": []any{}, "meta": map[string]any{"pagination": map[string]any{"page": 1, "per_page": 50, "last_page": 1, "total_entries": 0}}})
		}))
		defer cloud.Close()
		cloudConfig := filepath.Join(runtimeDir, "hcloud.toml")
		runCloud := func(args ...string) ([]byte, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			argv := append([]string{"--config", cloudConfig, "--endpoint", cloud.URL + "/v1"}, args...)
			child := exec.CommandContext(ctx, hcloudBinary, argv...)
			child.Env = append(append([]string{}, read.Env...), "HCLOUD_TOKEN=", "HCLOUD_CONTEXT=")
			return child.CombinedOutput()
		}
		args := []string{"context", "create", "production", "--credential-helper", binary, "--credential-url", "https://api.hetzner.cloud/01234567-89ab-cdef-0123-456789abcdef"}
		for _, argument := range []string{"--config=" + config, "--agent", "--agent-dialog=pinentry", "--agent-pinentry=" + req.Pinentry, "git-credential"} {
			args = append(args, "--credential-helper-arg="+argument)
		}
		if out, err := runCloud(args...); err != nil {
			t.Fatalf("hcloud context create: %v %s", err, out)
		}
		if out, err := runCloud("server", "list", "--output", "json"); err != nil || strings.TrimSpace(string(out)) != "[]" {
			t.Fatalf("hcloud authenticated command: %v %q", err, out)
		}
		if cloudRequests.Load() == 0 {
			t.Fatal("hcloud never reached the fake cloud API")
		}
		data, err := os.ReadFile(cloudConfig)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "hcloud-smoke-token") || strings.Contains(string(data), "test-passphrase") {
			t.Fatal("hcloud persisted a retrieved secret")
		}
	}

	status, err := command("agent", "status").CombinedOutput()
	if err != nil || !strings.Contains(string(status), "expires") {
		t.Fatalf("CLI status: %v %s", err, status)
	}
	if output, err := command("agent", "stop").CombinedOutput(); err != nil {
		t.Fatalf("CLI stop: %v %s", err, output)
	}
}
