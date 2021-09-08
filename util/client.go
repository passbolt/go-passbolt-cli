package util

import (
	"context"
	"fmt"

	"github.com/speatzle/go-passbolt/api"
	"github.com/spf13/viper"
)

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
		return nil, fmt.Errorf("userPassword is not defined")
	}

	client, err := api.NewClient(nil, "", serverAddress, userPrivateKey, userPassword)
	if err != nil {
		return nil, fmt.Errorf("Creating Client: %w", err)
	}

	client.Debug = viper.GetBool("debug")

	err = client.Login(ctx)
	if err != nil {
		return nil, fmt.Errorf("Logging in: %w", err)
	}
	return client, nil
}
