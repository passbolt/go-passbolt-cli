package util

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// ReadPassword reads a Password interactively or via Pipe. Interactive input
// echoes one '*' per typed rune; like term.ReadPassword it does not echo the
// final newline, callers print their own.
func ReadPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		reader := bufio.NewReader(os.Stdin)
		s, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.Replace(s, "\n", "", 1), nil
	}

	fmt.Fprint(os.Stderr, prompt)
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	var pass []byte
	buf := make([]byte, 1)
	inEscape := false
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		c := buf[0]

		// Arrow and function keys arrive as ESC sequences in raw mode; swallow
		// them so they don't leak bytes into the password. CSI sequences end
		// with a byte in 0x40..0x7E.
		if inEscape {
			if c >= 0x40 && c <= 0x7E {
				inEscape = false
			}
			continue
		}

		switch {
		case c == '\r' || c == '\n':
			return string(pass), nil
		case c == 3: // Ctrl+C
			return "", fmt.Errorf("input aborted")
		case c == 27: // ESC
			inEscape = true
		case c == 8 || c == 127: // Backspace / DEL: drop one rune, not one byte
			if len(pass) == 0 {
				continue
			}
			for len(pass) > 1 && pass[len(pass)-1]&0xC0 == 0x80 {
				pass = pass[:len(pass)-1]
			}
			pass = pass[:len(pass)-1]
			fmt.Fprint(os.Stderr, "\b \b")
		case c >= 32: // printable, including all UTF-8 multi-byte content
			pass = append(pass, c)
			if c&0xC0 != 0x80 { // one '*' per rune, not per continuation byte
				fmt.Fprint(os.Stderr, "*")
			}
		}
	}
}

// ReadNewPassword prompts for a new password and a confirmation (both masked
// with '*'), refusing empty input and giving three attempts for two matching
// entries. On a pipe it reads a single line and only enforces non-emptiness,
// so scripted use stays a one-liner.
func ReadNewPassword(prompt, confirmPrompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		pass, err := ReadPassword(prompt)
		if err != nil {
			return "", err
		}
		if pass == "" {
			return "", fmt.Errorf("the password cannot be empty")
		}
		return pass, nil
	}

	for range 3 {
		first, err := ReadPassword(prompt)
		if err != nil {
			return "", err
		}
		fmt.Fprintln(os.Stderr)
		if first == "" {
			fmt.Fprintln(os.Stderr, "The password cannot be empty.")
			continue
		}
		second, err := ReadPassword(confirmPrompt)
		if err != nil {
			return "", err
		}
		fmt.Fprintln(os.Stderr)
		if first == second {
			return first, nil
		}
		fmt.Fprintln(os.Stderr, "The passwords do not match, try again.")
	}
	return "", fmt.Errorf("no matching non-empty password after 3 attempts")
}

// SaveSessionKeysAndLogout saves any pending session keys to the server and then logs out.
// This should be used instead of client.Logout() to ensure session keys are persisted.
func SaveSessionKeysAndLogout(ctx context.Context, client *api.Client) {
	// Save any pending session keys that were discovered during decryption
	if count := client.GetPendingSessionKeysCount(); count > 0 {
		saved, err := client.SavePendingSessionKeys(ctx)
		if err != nil {
			// Log but don't fail - session keys can be re-discovered on next access
			fmt.Fprintf(os.Stderr, "Warning: failed to save session keys: %v\n", err)
		} else if saved > 0 {
			fmt.Fprintf(os.Stderr, "Saved %d session keys to server\n", saved)
		}
	}
	client.Logout(ctx)
}

// WithClient runs fn with a logged-in client, handling context, login,
// session-key persistence + logout, and SilenceUsage. SilenceUsage is set after
// login, so flag and login errors still print usage while runtime errors do not.
func WithClient(cmd *cobra.Command, fn func(ctx context.Context, client *api.Client) error) error {
	ctx, cancel := GetContext()
	defer cancel()

	client, err := GetClient(ctx)
	if err != nil {
		return err
	}
	defer SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	return fn(ctx, client)
}

// GetClient gets a Logged in Passbolt Client
func GetClient(ctx context.Context) (*api.Client, error) {
	serverAddress := viper.GetString("serverAddress")
	if serverAddress == "" {
		return nil, fmt.Errorf("serverAddress is not defined")
	}

	userPrivateKey := viper.GetString("userPrivateKey")
	if userPrivateKey == "" {
		return nil, fmt.Errorf("userPrivateKey is not defined")
	}

	userPassword := viper.GetString("userPassword")
	if userPassword == "" {
		cliPassword, err := ReadPassword("Enter Password:")
		if err != nil {
			fmt.Println()
			return nil, fmt.Errorf("reading Password: %w", err)
		}

		userPassword = cliPassword
		fmt.Println()
	}

	httpClient, err := GetHTTPClient()
	if err != nil {
		return nil, err
	}
	client, err := api.NewClient(httpClient, "", serverAddress, userPrivateKey, userPassword)
	if err != nil {
		return nil, fmt.Errorf("creating Client: %w", err)
	}

	client.Debug = viper.GetBool("debug")

	token := viper.GetString("serverVerifyToken")
	encToken := viper.GetString("serverVerifyEncToken")

	if token != "" {
		err = client.VerifyServer(ctx, token, encToken)
		if err != nil {
			return nil, fmt.Errorf("verifying Server: %w", err)
		}
	}

	switch viper.GetString("mfaMode") {
	case "interactive-totp":
		client.MFACallback = func(ctx context.Context, c *api.Client, res *api.APIResponse) (http.Cookie, error) {
			challenge := api.MFAChallenge{}
			err := json.Unmarshal(res.Body, &challenge)
			if err != nil {
				return http.Cookie{}, fmt.Errorf("parsing MFA Challenge")
			}
			if challenge.Provider.TOTP == "" {
				return http.Cookie{}, fmt.Errorf("server Provided no TOTP Provider")
			}
			for range 3 {
				var code string
				code, err := ReadPassword("Enter TOTP:")
				if err != nil {
					fmt.Printf("\n")
					return http.Cookie{}, fmt.Errorf("reading TOTP: %w", err)
				}
				fmt.Printf("\n")
				req := api.MFAChallengeResponse{
					TOTP: code,
				}
				var raw *http.Response
				raw, _, err = c.DoCustomRequestAndReturnRawResponseV5(ctx, "POST", "mfa/verify/totp.json", req, nil)
				if err != nil {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						return http.Cookie{}, fmt.Errorf("doing MFA Challenge Response: %w", err)
					}
					fmt.Println("TOTP Verification Failed")
				} else {
					// MFA worked so lets find the cookie and return it
					for _, cookie := range raw.Cookies() {
						if cookie.Name == "passbolt_mfa" {
							return *cookie, nil
						}
					}
					return http.Cookie{}, fmt.Errorf("unable to find Passbolt MFA Cookie")
				}
			}
			return http.Cookie{}, fmt.Errorf("failed MFA Challenge 3 times: %w", err)
		}
	case "noninteractive-totp":
		// if new flag is unset, use old flag instead
		totpToken := viper.GetString("mfaTotpToken")
		if totpToken == "" {
			totpToken = viper.GetString("totpToken")
		}

		totpOffset := viper.GetDuration("mfaTotpOffset")
		if totpOffset == time.Duration(0) {
			totpOffset = viper.GetDuration("totpOffset")
		}

		helper.AddMFACallbackTOTP(client, viper.GetUint("mfaRetrys"), viper.GetDuration("mfaDelay"), totpOffset, totpToken)
	case "none":
	default:
	}

	err = client.Login(ctx)
	if err != nil {
		return nil, ExplainAPIError("logging in", err)
	}
	return client, nil
}

// ExplainAPIError wraps err with operation-level context and, when err carries
// an *api.APIError, prepends a human-friendly explanation of the HTTP status
// code so users get actionable guidance instead of a bare status. The original
// error chain is preserved for errors.Is / errors.As.
func ExplainAPIError(op string, err error) error {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		if hint := apiStatusHint(apiErr.StatusCode); hint != "" {
			return fmt.Errorf("%s: %s: %w", op, hint, err)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

// apiStatusHint returns a short human-readable explanation for a Passbolt API
// HTTP status code, or "" when no specific guidance applies.
func apiStatusHint(code int) string {
	switch code {
	case http.StatusUnauthorized: // 401
		return "authentication failed, check your private key and password"
	case http.StatusForbidden: // 403
		return "access denied, you may lack the required permission or MFA may be needed"
	case http.StatusNotFound: // 404
		return "not found, check the requested ID and the server address"
	default:
		if code >= 500 {
			return "the Passbolt server reported an internal error, try again later"
		}
		return ""
	}
}
