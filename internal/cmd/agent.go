package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/passbolt/go-passbolt-cli/internal/agent"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// inAgent is set only by the private daemon entrypoint, never by a flag or env var.
var inAgent bool
var agentExecEnvironment []string
var agentExecArgs []string
var invocationEnvironment []string

func installAgent(root *cobra.Command) {
	flags := root.PersistentFlags()
	flags.Bool("agent", false, "Use the opt-in in-memory authentication agent")
	flags.String("agent-dialog", "auto", "Agent dialogs: auto, terminal, or pinentry")
	flags.String("agent-pinentry", "", "Pinentry executable (only with --agent-dialog=pinentry)")
	for _, key := range []string{"agent", "agent-dialog", "agent-pinentry"} {
		_ = viper.BindPFlag(key, flags.Lookup(key))
	}
	root.PersistentPreRunE = routeAgent
	controls := &cobra.Command{Use: "agent", Short: "Manage the in-memory authentication agent"}
	for _, action := range []string{"status", "lock", "stop"} {
		action := action
		controls.AddCommand(&cobra.Command{Use: action, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return callAgent(cmd, action) }})
	}
	root.AddCommand(controls)
}
func authenticatedCommand(cmd *cobra.Command) bool {
	for cmd.Parent() != nil && cmd.Parent().Parent() != nil {
		cmd = cmd.Parent()
	}
	switch cmd.Name() {
	case "get", "list", "create", "update", "delete", "move", "share", "export", "exec":
		return true
	}
	return false
}
func routeAgent(cmd *cobra.Command, _ []string) error {
	if inAgent || !viper.GetBool("agent") || !authenticatedCommand(cmd) {
		return nil
	}
	// Preserve Cobra's argument and required-flag validation before the remote call.
	cmd.Run = nil
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return callAgent(cmd, "run") }
	return nil
}
func requestFor(action string) (agent.Request, error) {
	directory, err := os.Getwd()
	if err != nil {
		return agent.Request{}, err
	}
	pinentry := viper.GetString("agent-pinentry")
	if strings.ContainsAny(pinentry, "/\\") && !filepath.IsAbs(pinentry) {
		pinentry, err = filepath.Abs(pinentry)
		if err != nil {
			return agent.Request{}, err
		}
	}
	return agent.Request{Version: agent.ProtocolVersion, Action: action, Args: os.Args[1:], Settings: viper.AllSettings(), Directory: directory, Environment: os.Environ(), Dialog: viper.GetString("agent-dialog"), Pinentry: pinentry, Terminal: agent.CurrentTerminal(), PID: os.Getpid()}, nil
}
func callAgent(cmd *cobra.Command, action string) error {
	if action == "run" && viper.GetBool("debug") {
		return fmt.Errorf("--debug cannot be used with the authentication agent")
	}
	req, err := requestFor(action)
	if err != nil {
		return err
	}
	if req.Dialog != "auto" && req.Dialog != "terminal" && req.Dialog != "pinentry" {
		return fmt.Errorf("unknown agent dialog backend %q", req.Dialog)
	}
	endpoint, err := agent.Endpoint()
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()
	conn, err := agent.Connect(ctx, endpoint, action == "run")
	if err != nil {
		if action == "status" || action == "stop" {
			util.Println("Agent is not running.")
			return nil
		}
		return err
	}
	result, err := agent.Exchange(ctx, conn, req, os.Stdin, os.Stdout, os.Stderr) //nolint:forbidigo // Relay already-sanitized agent output without altering JSON/binary bytes.
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	if action == "status" {
		if len(result.Status) == 0 {
			util.Println("Agent is running; no unlocked sessions.")
		}
		for _, status := range result.Status {
			util.Printf("%s (expires %s)\n", status.Account, status.Expires.Format(time.RFC3339))
		}
	}
	if result.Kind == "exec" {
		// Only resolved resource values leave the daemon, never its key or cookies.
		child := exec.CommandContext(ctx, result.Args[0], result.Args[1:]...)
		child.Env = result.Environment
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout //nolint:forbidigo // The foreground child owns its terminal streams.
		child.Stderr = os.Stderr //nolint:forbidigo // The foreground child owns its terminal streams.
		if err := child.Run(); err != nil {
			return fmt.Errorf("running command: %w", err)
		}
	}
	return nil
}

// agentIdentity includes all authentication and transport policy settings. Status
// carries only the public server and key fingerprint; hashes never reach logs.
func agentIdentity(settings map[string]any) (key, label string, err error) {
	config := viper.New()
	for name, value := range settings {
		config.Set(name, value)
	}
	server := config.GetString("serverAddress")
	armor := config.GetString("userPrivateKey")
	if server == "" || armor == "" {
		return "", "", fmt.Errorf("serverAddress and userPrivateKey must be defined")
	}
	publicKey, err := crypto.NewKeyFromArmored(armor)
	if err != nil {
		return "", "", fmt.Errorf("reading account key: %w", err)
	}
	fingerprint := publicKey.GetFingerprint()
	publicKey.ClearPrivateParams()
	identity := map[string]any{}
	for _, name := range []string{"serverAddress", "userPrivateKey", "userPassword", "serverVerifyToken", "serverVerifyEncToken", "mfaMode", "mfaTotpToken", "totpToken", "mfaTotpOffset", "totpOffset", "mfaRetrys", "mfaDelay", "tlsSkipVerify", "tlsClientPrivateKey", "tlsClientCert"} {
		identity[name] = config.Get(name)
	}
	// Proxy environment is transport policy too; it is added by the dispatcher.
	identity["proxy"] = settings["agent-proxy-policy"]
	data, err := json.Marshal(identity)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(data)
	clear(data)
	return hex.EncodeToString(digest[:]), agent.Text(server) + " / " + fingerprint, nil
}

func requestIdentity(req agent.Request) (string, string, error) {
	policy := []string{}
	for _, entry := range req.Environment {
		for _, prefix := range []string{"HTTP_PROXY=", "HTTPS_PROXY=", "NO_PROXY=", "http_proxy=", "https_proxy=", "no_proxy=", "REQUEST_METHOD="} {
			if strings.HasPrefix(entry, prefix) {
				policy = append(policy, entry)
			}
		}
	}
	sort.Strings(policy)
	settings := make(map[string]any, len(req.Settings)+1)
	for key, value := range req.Settings {
		settings[key] = value
	}
	settings["agent-proxy-policy"] = policy
	return agentIdentity(settings)
}

type cachedClient struct {
	client    *api.Client
	transport *managedTransport
	label     string
}
type commandAgent struct {
	sessions *agent.Sessions
	clients  map[string]cachedClient
	version  string
}

func newCommandAgent() *commandAgent {
	return &commandAgent{sessions: agent.NewSessions(time.Now), clients: make(map[string]cachedClient), version: rootCmd.Version}
}
func (a *commandAgent) Expire(ctx context.Context) { a.sessions.Expire(ctx) }
func (a *commandAgent) Close(ctx context.Context)  { a.sessions.Close(ctx) }
func (a *commandAgent) Control(ctx context.Context, req agent.Request) (agent.Message, error) {
	switch req.Action {
	case "status":
		statuses := a.sessions.Status()
		for i := range statuses {
			statuses[i].Account = a.clients[statuses[i].Account].label
		}
		return agent.Message{Status: statuses}, nil
	case "lock":
		key, _, err := requestIdentity(req)
		if err != nil {
			return agent.Message{}, err
		}
		a.sessions.Lock(ctx, key)
	case "stop":
		a.Close(ctx)
	}
	return agent.Message{}, nil
}
func (a *commandAgent) Run(ctx context.Context, req agent.Request, in io.Reader, out, errOut io.Writer) (result agent.Message, runErr error) {
	defer func() {
		util.FlushOutputs()
		util.WarnReplacements()
		util.ClientProvider = nil
		util.ClientRelease = nil
		util.PasswordPrompt = nil
		util.ConfirmationPrompt = nil
		util.InvocationContext = context.Background()
		util.ResourceApproval = nil
		util.ProxyEnvironment = nil
		util.Out = io.Discard
		util.Err = io.Discard
		invocationEnvironment = nil
		agentExecEnvironment = nil
		agentExecArgs = nil
		// Do not retain request configuration or its credentials between invocations.
		viper.Reset()
		// Discard flags too: a passphrase supplied as a flag must not remain in the
		// last executed Cobra tree after its session is locked or expires.
		newCommandTree()
		rootCmd.Version = a.version
		viper.Reset()
	}()
	if req.Settings == nil {
		return result, fmt.Errorf("missing agent configuration")
	}
	// Reset the global Viper instance and every command/flag object. The dispatcher
	// guarantees only one invocation can own these globals at a time.
	viper.Reset()
	cfgFile = ""
	cfgPath = ""
	root := newCommandTree()
	root.Version = a.version
	for key, value := range req.Settings {
		viper.Set(key, value)
	}
	timeout := viper.GetDuration("timeout")
	if timeout <= 0 {
		return result, fmt.Errorf("agent timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if viper.GetBool("debug") {
		return result, fmt.Errorf("debug logging is unavailable in agent mode")
	}
	root.SetArgs(req.Args)
	selected, _, err := root.Find(req.Args)
	if err != nil {
		return result, err
	}
	if !authenticatedCommand(selected) {
		return result, fmt.Errorf("command is not available through the agent")
	}
	// Bind the approval to the command path, never to argument values that may be secrets.
	summary := selected.CommandPath()

	key, label, err := requestIdentity(req)
	if err != nil {
		return result, err
	}
	dialog := agent.Dialog{Backend: req.Dialog, Pinentry: req.Pinentry, Terminal: req.Terminal, PID: req.PID, Environment: req.Environment}
	requester := agent.Requester(req.PID)
	if err := dialog.Confirm(ctx, "Passbolt approval", label+" — "+summary+" | Requester: "+requester); err != nil {
		return result, err
	}
	// Only approval refreshes the idle deadline. Existing expired entries have
	// already been removed by Serve before dispatching this request.
	a.sessions.Touch(key)
	previousDir, err := os.Getwd()
	if err != nil {
		return result, err
	}
	if err := os.Chdir(req.Directory); err != nil {
		return result, err
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil && runErr == nil {
			runErr = fmt.Errorf("restoring agent directory: %w", err)
		}
	}()
	util.Out = util.NewSanitizingWriter(out)
	util.Err = util.NewSanitizingWriter(errOut)
	root.SetOut(util.Out)
	root.SetErr(util.Err)
	util.InvocationContext = ctx
	util.ResourceApproval = func(resourceCtx context.Context, id, name string) error {
		title, prompt := agent.ResourceApproval(label, summary, requester, id, name)
		return dialog.Confirm(resourceCtx, title, prompt)
	}
	util.ProxyEnvironment = append([]string{}, req.Environment...)
	invocationEnvironment = req.Environment
	agentExecArgs = nil
	agentExecEnvironment = nil
	util.PasswordPrompt = func(prompt string) (string, error) {
		return dialog.Ask(ctx, "Passbolt authentication", label+" — "+prompt, true)
	}
	util.ConfirmationPrompt = func(prompt string) (bool, error) {
		err := dialog.Confirm(ctx, "Passbolt confirmation", prompt)
		if errors.Is(err, agent.ErrDenied) {
			return false, nil
		}
		return err == nil, err
	}
	util.ClientProvider = func(clientCtx context.Context) (*api.Client, error) {
		if cached, ok := a.clients[key]; ok {
			// Session checks happen before user code; a write is never retried.
			if !cached.client.CheckSession(clientCtx) {
				if err := cached.client.Login(clientCtx); err != nil {
					a.sessions.Lock(clientCtx, key)
					return nil, util.ExplainAPIError("renewing agent session", err)
				}
			}
			return cached.client, nil
		}
		httpClient, err := util.GetHTTPClient()
		if err != nil {
			return nil, err
		}
		transport := &managedTransport{network: httpClient.Transport}
		httpClient.Transport = transport
		client, err := util.NewClientWithHTTP(clientCtx, httpClient)
		if err != nil {
			return nil, err
		}
		a.clients[key] = cachedClient{client: client, transport: transport, label: label}
		a.sessions.Put(key, func(_ context.Context) {
			cached, ok := a.clients[key]
			if !ok {
				return
			}
			delete(a.clients, key)
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			closeCachedClient(cleanup, cached)

		})
		return client, nil
	}
	util.ClientRelease = func(clientCtx context.Context, client *api.Client) {
		if client.GetPendingSessionKeysCount() > 0 {
			if _, err := client.SavePendingSessionKeys(clientCtx); err != nil {
				util.Eprintln("Warning: failed to save session keys:", util.Sanitize(err.Error()))
			}
		}
	}

	root.SetContext(ctx)
	err = root.ExecuteContext(ctx)
	if err != nil {
		return result, err
	}
	if agentExecArgs != nil {
		return agent.Message{Kind: "exec", Args: agentExecArgs, Environment: agentExecEnvironment}, nil
	}
	return result, nil
}

func serveAgent() error {
	inAgent = true
	endpoint, err := agent.Endpoint()
	if err != nil {
		return err
	}
	listener, err := agent.Listen(endpoint)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	return agent.Serve(ctx, listener, newCommandAgent())
}

// SDK v0.8.4 clears its key only after a successful logout response. After an
// unsuccessful best-effort remote logout, a second, strictly local logout lets
// the SDK run its own key-zeroing path without exporting keys or using unsafe.
// This adapter can be removed once the SDK exposes unconditional local cleanup.
func closeCachedClient(ctx context.Context, cached cachedClient) {
	cached.client.MFACallback = nil
	err := cached.client.Logout(ctx)
	if transport, ok := cached.transport.network.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
	if err != nil {
		cached.transport.local = true
		_ = cached.client.Logout(context.Background())
	}
	cached.client.ClearCache()
}

// The SDK shallow-copies http.Client; switch behavior through the shared
// transport pointer rather than mutating the caller's copied Client fields.
type managedTransport struct {
	network http.RoundTripper
	local   bool
}

func (t *managedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.local {
		return (localLogoutTransport{}).RoundTrip(req)
	}
	return t.network.RoundTrip(req)
}

type localLogoutTransport struct{}

func (localLogoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "POST" || !strings.HasSuffix(req.URL.Path, "/auth/logout.json") {
		return nil, fmt.Errorf("local cleanup transport only permits logout")
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"header":{"status":"success","code":200},"body":null}`)), Request: req}, nil
}
