package keepass

import (
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
	KeepassExportCmd.Flags().String("kdbx-version", "v3", "KDBX format version: v3 (AES-KDF, KDBX 3.1) or v4 (Argon2, KDBX 4)")
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

	kdbxVersionFlag, err := cmd.Flags().GetString("kdbx-version")
	if err != nil {
		return err
	}

	var kdbxVersion gokeepasslib.DatabaseOption
	switch kdbxVersionFlag {
	case "v3":
		kdbxVersion = gokeepasslib.WithDatabaseKDBXVersion3()
	case "v4":
		kdbxVersion = gokeepasslib.WithDatabaseKDBXVersion4()
	default:
		return fmt.Errorf("invalid kdbx-version %q: must be v3 or v4", kdbxVersionFlag)
	}

	ctx, cancel := util.GetContext()
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return err
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	cmd.SilenceUsage = true

	if keepassPassword == "" {
		pw, err := util.ReadPassword("Enter KeePass Password:")
		if err != nil {
			fmt.Println()
			return fmt.Errorf("reading KeePass Password: %w", err)
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
		return fmt.Errorf("getting Resources: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating File: %w", err)
	}
	defer file.Close()

	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = "root"

	pterm.EnableStyling()
	pterm.DisableColor()
	progressbar, err := pterm.DefaultProgressbar.WithTitle("Decryping Resources").WithTotal(len(resources)).Start()
	if err != nil {
		return fmt.Errorf("progress: %w", err)
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

	db := gokeepasslib.NewDatabase(kdbxVersion)
	db.Content.Meta.DatabaseName = "Passbolt Export"

	if keepassPassword != "" {
		db.Credentials = gokeepasslib.NewPasswordCredentials(keepassPassword)
	}

	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{rootGroup},
	}

	if err := db.LockProtectedEntries(); err != nil {
		return fmt.Errorf("locking protected entries: %w", err)
	}

	keepassEncoder := gokeepasslib.NewEncoder(file)
	if err := keepassEncoder.Encode(db); err != nil {
		return fmt.Errorf("encodeing kdbx: %w", err)
	}
	fmt.Println("Done")

	return nil
}

func getKeepassEntry(client *api.Client, resource api.Resource, secret api.Secret, rType api.ResourceType) (*gokeepasslib.Entry, error) {
	_, metadata, secretFields, err := helper.GetResourceFieldMaps(client, resource, secret, rType, true)
	if err != nil {
		return nil, fmt.Errorf("get Resource %v: %w", resource.ID, err)
	}

	name := helper.GetStringField(metadata, "name")
	username := helper.GetStringField(metadata, "username")
	uri := helper.GetStringField(metadata, "uri")
	password := helper.GetStringField(secretFields, "password")
	description := helper.GetStringField(metadata, "description")

	entry := gokeepasslib.NewEntry()
	entry.Values = append(
		entry.Values,
		gokeepasslib.ValueData{Key: "Title", Value: gokeepasslib.V{Content: name}},
		gokeepasslib.ValueData{Key: "UserName", Value: gokeepasslib.V{Content: username}},
		gokeepasslib.ValueData{Key: "URL", Value: gokeepasslib.V{Content: uri}},
		gokeepasslib.ValueData{Key: "Password", Value: gokeepasslib.V{Content: password, Protected: w.NewBoolWrapper(true)}},
		gokeepasslib.ValueData{Key: "Notes", Value: gokeepasslib.V{Content: description}},
	)

	if totpRaw, ok := secretFields["totp"].(map[string]any); ok {
		secretKey, _ := totpRaw["secret_key"].(string)
		// Skip TOTP entry if secret_key is missing — can't build a valid OTP URI
		if secretKey != "" {
			algorithm, _ := totpRaw["algorithm"].(string)
			var digits, period int
			if d, ok := totpRaw["digits"].(float64); ok {
				digits = int(d)
			}
			if p, ok := totpRaw["period"].(float64); ok {
				period = int(p)
			}

			v := url.Values{}
			v.Set("secret", secretKey)
			v.Set("period", strconv.FormatUint(uint64(period), 10))
			v.Set("algorithm", algorithm)
			v.Set("digits", fmt.Sprint(digits))

			issuer := uri
			if uri == "" {
				issuer = name
			}
			v.Set("issuer", issuer)

			accountName := username
			if username == "" {
				accountName = name
			}

			u := url.URL{
				Scheme:   "otpauth",
				Host:     "totp",
				Path:     "/" + issuer + ":" + accountName,
				RawQuery: encodeQuery(v),
			}

			entry.Values = append(entry.Values, gokeepasslib.ValueData{Key: "otp", Value: gokeepasslib.V{Content: u.String(), Protected: w.NewBoolWrapper(true)}})
		}
	}

	// Custom fields: each one becomes a protected KDBX field, matching what the
	// Passbolt browser extension does in resourcesKdbxExporter.setCustomFields.
	// Field name comes from metadata.custom_fields[].metadata_key, value from
	// secret.custom_fields[].secret_value, correlated by id.
	addCustomFields(&entry, metadata, secretFields)

	return &entry, nil
}

func addCustomFields(entry *gokeepasslib.Entry, metadata, secretFields map[string]any) {
	metaList, _ := metadata["custom_fields"].([]any)
	if len(metaList) == 0 {
		return
	}
	secretList, _ := secretFields["custom_fields"].([]any)
	valueByID := make(map[string]string, len(secretList))
	for _, item := range secretList {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		val, _ := m["secret_value"].(string)
		if id != "" {
			valueByID[id] = val
		}
	}
	for _, item := range metaList {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		key, _ := m["metadata_key"].(string)
		if key == "" {
			continue
		}
		entry.Values = append(entry.Values, gokeepasslib.ValueData{
			Key:   key,
			Value: gokeepasslib.V{Content: valueByID[id], Protected: w.NewBoolWrapper(true)},
		})
	}
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
