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

	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// ReadPassword reads a Password interactively or via Pipe
func ReadPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	var pass string
	if term.IsTerminal(fd) {
		fmt.Print(prompt)

		inputPass, err := term.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		pass = string(inputPass)
	} else {
		reader := bufio.NewReader(os.Stdin)
		s, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		pass = s
	}

	return strings.Replace(pass, "\n", "", 1), nil
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
			return nil, fmt.Errorf("Reading Password: %w", err)
		}

		userPassword = cliPassword
		fmt.Println()
	}

	client, err := api.NewClient(nil, "", serverAddress, userPrivateKey, userPassword)
	if err != nil {
		return nil, fmt.Errorf("Creating Client: %w", err)
	}

	client.Debug = viper.GetBool("debug")

	token := viper.GetString("serverVerifyToken")
	encToken := viper.GetString("serverVerifyEncToken")

	if token != "" {
		err = client.VerifyServer(ctx, token, encToken)
		if err != nil {
			return nil, fmt.Errorf("Verifing Server: %w", err)
		}
	}

	switch viper.GetString("mfaMode") {
	case "interactive-totp":
		client.MFACallback = func(ctx context.Context, c *api.Client, res *api.APIResponse) (http.Cookie, error) {
			challenge := api.MFAChallenge{}
			err := json.Unmarshal(res.Body, &challenge)
			if err != nil {
				return http.Cookie{}, fmt.Errorf("Parsing MFA Challenge")
			}
			if challenge.Provider.TOTP == "" {
				return http.Cookie{}, fmt.Errorf("Server Provided no TOTP Provider")
			}
			for i := 0; i < 3; i++ {
				var code string
				code, err := ReadPassword("Enter TOTP:")
				if err != nil {
					fmt.Printf("\n")
					return http.Cookie{}, fmt.Errorf("Reading TOTP: %w", err)
				}
				fmt.Printf("\n")
				req := api.MFAChallengeResponse{
					TOTP: code,
				}
				var raw *http.Response
				raw, _, err = c.DoCustomRequestAndReturnRawResponse(ctx, "POST", "mfa/verify/totp.json", "v2", req, nil)
				if err != nil {
					if errors.Unwrap(err) != api.ErrAPIResponseErrorStatusCode {
						return http.Cookie{}, fmt.Errorf("Doing MFA Challenge Response: %w", err)
					}
					fmt.Println("TOTP Verification Failed")
				} else {
					// MFA worked so lets find the cookie and return it
					for _, cookie := range raw.Cookies() {
						if cookie.Name == "passbolt_mfa" {
							return *cookie, nil
						}
					}
					return http.Cookie{}, fmt.Errorf("Unable to find Passbolt MFA Cookie")
				}
			}
			return http.Cookie{}, fmt.Errorf("Failed MFA Challenge 3 times: %w", err)
		}
	case "noninteractive-totp":
		helper.AddMFACallbackTOTP(client, viper.GetUint("mfaRetrys"), viper.GetDuration("mfaDelay"), viper.GetDuration("totpOffset"), viper.GetString("totpToken"))
	case "none":
	default:
	}

	err = client.Login(ctx)
	if err != nil {
		return nil, fmt.Errorf("Logging in: %w", err)
	}
	return client, nil
}
