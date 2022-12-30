package keepass

import (
	"context"
	"fmt"
	"os"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// KeepassExportCmd Exports a Passbolt Keepass
var KeepassExportCmd = &cobra.Command{
	Use:     "keepass",
	Short:   "Exports Passbolt to a Keepass File",
	Long:    `Exports Passbolt to a Keepass File`,
	Aliases: []string{},
	RunE:    KeepassExport,
}

func init() {
	KeepassExportCmd.Flags().StringP("file", "f", "passbolt-export.kdbx", "File name of the Keepass File")
	KeepassExportCmd.Flags().StringP("password", "p", "", "Password for the Keypass File, if empty prompts interactively")
}

func KeepassExport(cmd *cobra.Command, args []string) error {
	filename, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}

	if filename == "" {
		return fmt.Errorf("the Filename cannot be empty")
	}

	keepassPassword, err := cmd.Flags().GetString("password")
	if err != nil {
		return err
	}

	ctx := util.GetContext()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer client.Logout(context.TODO())
	cmd.SilenceUsage = true

	if keepassPassword == "" {
		pw, err := util.ReadPassword("Enter Keepass Password:")
		if err != nil {
			fmt.Println()
			return fmt.Errorf("Reading Keepass Password: %w", err)
		}
		keepassPassword = pw
		fmt.Println()
	}

	fmt.Println("Getting Resources...")
	resources, err := client.GetResources(ctx, &api.GetResourcesOptions{
		ContainSecret:       true,
		ContainResourceType: true,
		ContainTags:         true,
	})
	if err != nil {
		return fmt.Errorf("Getting Resources: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Creating File: %w", err)
	}
	defer file.Close()

	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = "root"

	pterm.EnableStyling()
	pterm.DisableColor()
	progressbar, err := pterm.DefaultProgressbar.WithTitle("Decryping Resources").WithTotal(len(resources)).Start()
	if err != nil {
		return fmt.Errorf("Progress: %w", err)
	}

	for i, resource := range resources {
		_, _, _, _, pass, desc, err := helper.GetResourceFromData(client, resource, resource.Secrets[0], resource.ResourceType)
		if err != nil {
			return fmt.Errorf("Get Resource %v, %v %w", i, resource.ID, err)
		}

		entry := gokeepasslib.NewEntry()
		entry.Values = append(
			entry.Values,
			gokeepasslib.ValueData{Key: "Title", Value: gokeepasslib.V{Content: resource.Name}},
			gokeepasslib.ValueData{Key: "UserName", Value: gokeepasslib.V{Content: resource.Username}},
			gokeepasslib.ValueData{Key: "URL", Value: gokeepasslib.V{Content: resource.URI}},
			gokeepasslib.ValueData{Key: "Password", Value: gokeepasslib.V{Content: pass, Protected: w.NewBoolWrapper(true)}},
			gokeepasslib.ValueData{Key: "Notes", Value: gokeepasslib.V{Content: desc}},
		)
		rootGroup.Entries = append(rootGroup.Entries, entry)
		progressbar.Increment()
	}

	db := gokeepasslib.NewDatabase(
		gokeepasslib.WithDatabaseKDBXVersion4(),
	)
	db.Content.Meta.DatabaseName = "Passbolt Export"

	if keepassPassword != "" {
		db.Credentials = gokeepasslib.NewPasswordCredentials(keepassPassword)
	}

	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{rootGroup},
	}

	db.LockProtectedEntries()

	keepassEncoder := gokeepasslib.NewEncoder(file)
	if err := keepassEncoder.Encode(db); err != nil {
		return fmt.Errorf("Encodeing kdbx: %w", err)
	}
	fmt.Println("Done")

	return nil
}
