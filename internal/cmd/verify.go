package cmd

import (
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// verifyCMD represents the verify command
var verifyCMD = &cobra.Command{
	Use:   "verify",
	Short: "Verify Setup the Server Verification",
	Long: `Verify Setup the Server Verification. You need to run this once after that the Server will always be verified if the same config is used

The verification tokens are written to the config file (mode 0600).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := util.GetContext()
		defer cancel()

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
				util.Println()
				return fmt.Errorf("reading Password: %w", err)
			}
			userPassword = pw
			util.Println()
		}

		httpClient, err := util.GetHTTPClient()
		if err != nil {
			return err
		}
		client, err := api.NewClient(httpClient, "", serverAddress, userPrivateKey, userPassword)
		if err != nil {
			return fmt.Errorf("creating Client: %w", err)
		}

		client.Debug = viper.GetBool("debug")

		token, enctoken, err := client.SetupServerVerification(ctx)
		if err != nil {
			return fmt.Errorf("setup Verification: %w", err)
		}
		viper.Set("serverVerifyToken", token)
		viper.Set("serverVerifyEncToken", enctoken)

		if err := util.WriteViperConfig(cfgPath); err != nil {
			return fmt.Errorf("writing Config: %w", err)
		}
		util.Println("Verification Enabled")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCMD)
}
