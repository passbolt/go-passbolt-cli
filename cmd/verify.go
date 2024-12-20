package cmd

import (
	"fmt"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// verifyCMD represents the verify command
var verifyCMD = &cobra.Command{
	Use:   "verify",
	Short: "Verify Setup the Server Verification",
	Long:  `Verify Setup the Server Verification. You need to run this once after that the Server will always be verified if the same config is used`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := util.GetContext()

		viper.Set("serverVerifyToken", "")
		viper.Set("serverVerifyEncToken", "")

		serverAddress := viper.GetString("serverAddress")
		if serverAddress == "" {
			return fmt.Errorf("serverAddress is not defined")
		}

		userPrivateKey := viper.GetString("userPrivateKey")
		if userPrivateKey == "" {
			return fmt.Errorf("userPrivateKey is not defined")
		}

		userPassword := viper.GetString("userPassword")
		if userPassword == "" {
			pw, err := util.ReadPassword("Enter Password:")
			if err != nil {
				fmt.Println()
				return fmt.Errorf("Reading Password: %w", err)
			}
			userPassword = pw
			fmt.Println()
		}

		httpClient, err := util.GetHttpClient()
		if err != nil {
			return err
		}
		client, err := api.NewClient(httpClient, "", serverAddress, userPrivateKey, userPassword)
		if err != nil {
			return fmt.Errorf("Creating Client: %w", err)
		}

		client.Debug = viper.GetBool("debug")

		token, enctoken, err := client.SetupServerVerification(ctx)
		if err != nil {
			return fmt.Errorf("Setup Verification: %w", err)
		}
		viper.Set("serverVerifyToken", token)
		viper.Set("serverVerifyEncToken", enctoken)

		if viper.ConfigFileUsed() == "" {
			err := viper.SafeWriteConfig()
			if err != nil {
				return fmt.Errorf("Writing Config: %w", err)
			}
		} else {
			err := viper.WriteConfig()
			if err != nil {
				return fmt.Errorf("Writing Config: %w", err)
			}
		}
		fmt.Println("Verification Enabled")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCMD)
}
