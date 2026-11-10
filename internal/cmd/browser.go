package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/passbolt/go-passbolt-cli/internal/agent"
	"github.com/passbolt/go-passbolt-cli/internal/browserauth"
	"github.com/spf13/cobra"
)

func newBrowserCommand() *cobra.Command {
	command := &cobra.Command{Use: "browser", Short: "Install browser authorization integration"}
	var executable string
	var extraChromiumIDs []string
	install := &cobra.Command{Use: "install", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error {
		if executable == "" {
			self, err := os.Executable()
			if err != nil {
				return err
			}
			executable = filepath.Join(filepath.Dir(self), "passbolt-browser-host")
			if filepath.Ext(self) == ".exe" {
				executable += ".exe"
			}
		}
		return browserauth.Install(executable, extraChromiumIDs...)
	}}
	install.Flags().StringVar(&executable, "host-path", "", "Absolute path to the passbolt-browser-host executable")
	install.Flags().StringSliceVar(&extraChromiumIDs, "extra-chromium-id", nil, "Additional Chromium extension IDs to allow (e.g. unpacked development extension)")
	host := &cobra.Command{
		Use:    "host",
		Short:  "Run native messaging host",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return RunBrowserHost(os.Stdin, os.Stdout)
		},
	}
	command.AddCommand(install, host, &cobra.Command{Use: "uninstall", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error { return browserauth.Uninstall() }})
	return command
}

type safeWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *safeWriter) WriteNative(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return browserauth.WriteNative(s.w, v)
}

// RunBrowserHost handles only native messages and never loads CLI configuration.
// Its stdout belongs exclusively to the browser native messaging protocol.
func RunBrowserHost(in io.Reader, out io.Writer) error {
	safeOut := &safeWriter{w: out}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var listenOnce sync.Once

	for {
		var request struct {
			Action   string               `json:"action"`
			Envelope browserauth.Envelope `json:"envelope"`
		}
		if err := browserauth.ReadNative(in, &request); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if request.Action == "listen" {
			listenOnce.Do(func() {
				go watchAgentNotifications(ctx, safeOut)
			})
			_ = safeOut.WriteNative(map[string]any{"listening": true})
			continue
		}
		if request.Action == "ping" {
			_ = safeOut.WriteNative(map[string]any{"pong": true})
			continue
		}

		response, err := relayBrowserMessage(request.Action, request.Envelope)
		if err != nil {
			response = map[string]any{"error": "Browser authorization unavailable. Start a CLI command using --agent --agent-auth=browser and try again."}
		}
		if err = safeOut.WriteNative(response); err != nil {
			return err
		}
	}
}

func watchAgentNotifications(ctx context.Context, out *safeWriter) {
	var lastNotifiedID string
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		endpoint, err := agent.Endpoint()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		conn, err := agent.Dial(ctx, endpoint)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		req := agent.Request{Version: agent.ProtocolVersion, Action: "browser-wait"}
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := agent.WriteFrame(conn, req); err != nil {
			_ = conn.Close()
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		_ = conn.SetWriteDeadline(time.Time{})

		_ = conn.SetReadDeadline(time.Now().Add(35 * time.Second))
		var result agent.Message
		err = agent.ReadFrame(conn, &result)
		_ = conn.Close()

		if err != nil {
			continue
		}

		if result.Kind == "browser" && result.Error == "" {
			var pending browserauth.Request
			if err := json.Unmarshal(result.Data, &pending); err == nil && pending.ID != "" {
				if pending.ID != lastNotifiedID {
					lastNotifiedID = pending.ID
					_ = out.WriteNative(map[string]any{
						"action": "open-auth-dialog",
					})
				}
			}
		} else {
			lastNotifiedID = ""
		}
	}
}
func relayBrowserMessage(action string, envelope browserauth.Envelope) (any, error) {
	if action != "pending" && action != "submit" && action != "cancel" {
		return nil, fmt.Errorf("unknown native browser action")
	}
	endpoint, err := agent.Endpoint()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := agent.Dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	req := agent.Request{Version: agent.ProtocolVersion, Action: "browser-" + action}
	if action == "submit" {
		req.BrowserData, err = json.Marshal(envelope)
		if err != nil {
			return nil, err
		}
	}
	if action == "cancel" {
		req.BrowserData = []byte("cancelled by user")
	}
	if err = agent.WriteFrame(conn, req); err != nil {
		return nil, err
	}
	var result agent.Message
	if err = agent.ReadFrame(conn, &result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("browser request failed")
	}
	if action == "cancel" {
		return map[string]bool{"cancelled": true}, nil
	}
	if action == "submit" {
		return map[string]bool{"submitted": true}, nil
	}
	var pending browserauth.Request
	if err = json.Unmarshal(result.Data, &pending); err != nil {
		return nil, err
	}
	return map[string]any{"request": pending}, nil
}
