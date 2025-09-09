package keepass

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// KeepassExportCmd Exports a Passbolt KeePass
var KeepassExportCmd = &cobra.Command{
	Use:     "keepass",
	Short:   "Exports Passbolt to a KeePass File",
	Long:    `Exports Passbolt to a KeePass File`,
	Aliases: []string{},
	RunE:    KeepassExport,
}

func init() {
	KeepassExportCmd.Flags().StringP("file", "f", "passbolt-export.kdbx", "File name of the KeePass File")
	KeepassExportCmd.Flags().StringP("password", "p", "", "Password for the KeePass File, if empty prompts interactively")
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
		pw, err := util.ReadPassword("Enter KeePass Password:")
		if err != nil {
			fmt.Println()
			return fmt.Errorf("Reading KeePass Password: %w", err)
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

	for _, resource := range resources {
		entry, err := getKeepassEntry(client, resource, resource.Secrets[0], resource.ResourceType)
		if err != nil {
			fmt.Printf("\nSkipping Export of Resource %v %v Because of: %v\n", resource.ID, resource.Name, err)
			progressbar.Increment()
			continue
		}

		rootGroup.Entries = append(rootGroup.Entries, *entry)
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

func getKeepassEntry(client *api.Client, resource api.Resource, secret api.Secret, rType api.ResourceType) (*gokeepasslib.Entry, error) {
	_, _, _, _, pass, desc, err := helper.GetResourceFromData(client, resource, resource.Secrets[0], resource.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("Get Resource %v: %w", resource.ID, err)
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

	if resource.ResourceType.Slug == "password-description-totp" || resource.ResourceType.Slug == "totp" {
		var totpData api.SecretDataTOTP

		rawSecretData, err := client.DecryptMessage(resource.Secrets[0].Data)
		if err != nil {
			return nil, fmt.Errorf("Decrypting Secret Data: %w", err)
		}

		if resource.ResourceType.Slug == "password-description-totp" {
			var secretData api.SecretDataTypePasswordDescriptionTOTP
			err = json.Unmarshal([]byte(rawSecretData), &secretData)
			if err != nil {
				return nil, fmt.Errorf("Parsing Decrypted Secret Data: %w", err)
			}
			totpData = secretData.TOTP
		} else {
			var secretData api.SecretDataTypeTOTP
			err = json.Unmarshal([]byte(rawSecretData), &secretData)
			if err != nil {
				return nil, fmt.Errorf("Parsing Decrypted Secret Data: %w", err)
			}
			totpData = secretData.TOTP
		}

		v := url.Values{}
		v.Set("secret", totpData.SecretKey)
		v.Set("period", strconv.FormatUint(uint64(totpData.Period), 10))
		v.Set("algorithm", totpData.Algorithm)
		v.Set("digits", fmt.Sprint(totpData.Digits))

		issuer := resource.URI
		if resource.URI == "" {
			issuer = resource.Name

		}
		v.Set("issuer", issuer)

		accountName := resource.Username
		if resource.Username == "" {
			accountName = resource.Name
		}

		u := url.URL{
			Scheme:   "otpauth",
			Host:     "totp",
			Path:     "/" + issuer + ":" + accountName,
			RawQuery: encodeQuery(v),
		}

		entry.Values = append(entry.Values, gokeepasslib.ValueData{Key: "otp", Value: gokeepasslib.V{Content: u.String(), Protected: w.NewBoolWrapper(true)}})
	}

	return &entry, nil
}

// EncodeQuery is a copy-paste of url.Values.Encode, except it uses %20 instead
// of + to encode spaces. This is necessary to correctly render spaces in some
// authenticator apps, like Google Authenticator.
func encodeQuery(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		keyEscaped := url.PathEscape(k) // changed from url.QueryEscape
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.PathEscape(v)) // changed from url.QueryEscape
		}
	}
	return buf.String()
}
