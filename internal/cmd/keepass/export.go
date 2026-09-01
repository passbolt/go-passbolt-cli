package keepass

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// KeepassExportCmd Exports a Passbolt KeePass
var KeepassExportCmd = &cobra.Command{
	Use:   "keepass",
	Short: "Exports Passbolt to a KeePass File",
	Long: `Exports Passbolt Resources to a KeePass (kdbx) File.

The Passbolt Folder hierarchy is mirrored as nested KeePass Groups;
Resources without a Folder are placed in the top Group. Tags, creation and
modification timestamps, expiry dates and Custom Fields are carried over.

TOTP configurations are written in both conventions: the otpauth "otp"
field (read by KeePassXC and the KeeOtp2 plugin) and the native TimeOtp-*
fields used by the built-in generator of KeePass 2.x (Ctrl+T).

With --folder only the given Folder and its Subfolders are exported; with
--comments the Passbolt Comment threads are appended to the Notes field.`,
	Aliases: []string{},
	RunE:    KeepassExport,
}

func init() {
	KeepassExportCmd.Flags().StringP("file", "f", "passbolt-export.kdbx", "File name of the KeePass File")
	KeepassExportCmd.Flags().StringP("password", "p", "", "Password for the KeePass File, if empty prompts interactively")
	KeepassExportCmd.Flags().String("kdbx-version", "v3", "KDBX format version: v3 (AES-KDF, KDBX 3.1) or v4 (Argon2, KDBX 4)")
	KeepassExportCmd.Flags().String("folder", "", "ID of a Folder: export only this Folder and its Subfolders")
	KeepassExportCmd.Flags().Bool("comments", false, "Include Passbolt Comments in the KeePass Notes field (one extra API request per Resource)")
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

	folderID, err := cmd.Flags().GetString("folder")
	if err != nil {
		return err
	}

	includeComments, err := cmd.Flags().GetBool("comments")
	if err != nil {
		return err
	}

	var kdbxVersion gokeepasslib.DatabaseOption
	switch kdbxVersionFlag {
	case "v3":
		kdbxVersion = gokeepasslib.WithDatabaseKDBXVersion3()
	case "v4":
		kdbxVersion = gokeepasslib.WithDatabaseKDBXVersion40()
	default:
		return fmt.Errorf("invalid kdbx-version %q: must be v3 or v4", kdbxVersionFlag)
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		// A kdbx without credentials would be written without any protection,
		// so an empty password is rejected rather than silently accepted.
		if keepassPassword == "" {
			pw, err := util.ReadNewPassword("Enter KeePass Password:", "Confirm KeePass Password:")
			if err != nil {
				return fmt.Errorf("reading KeePass Password: %w", err)
			}
			keepassPassword = pw
		}

		fmt.Println("Getting Folders...")
		folders, err := client.GetFolders(ctx, &api.GetFoldersOptions{})
		if err != nil {
			return fmt.Errorf("getting Folders: %w", err)
		}

		var subtree map[string]bool
		if folderID != "" {
			subtree = subtreeIDs(folders, folderID)
			if subtree == nil {
				return fmt.Errorf("folder %v not found", folderID)
			}
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

		// With --folder, resources outside the subtree are dropped before the
		// loop so their secrets are never decrypted.
		if subtree != nil {
			inSubtree := make([]api.Resource, 0, len(resources))
			for _, resource := range resources {
				if subtree[resource.FolderParentID] {
					inSubtree = append(inSubtree, resource)
				}
			}
			resources = inSubtree
		}

		// Encode into a temp file and rename at the end, so a failed export
		// can't leave a partial kdbx behind that looks like a valid backup.
		tmpName := filename + ".tmp"
		file, err := os.Create(tmpName)
		if err != nil {
			return fmt.Errorf("creating File: %w", err)
		}
		committed := false
		defer func() {
			_ = file.Close()
			if !committed {
				_ = os.Remove(tmpName)
			}
		}()

		rootNode := &folderNode{name: "Passbolt Export"}
		folderNodes := buildFolderTree(rootNode, folders)
		if folderID != "" {
			rootNode.children = []*folderNode{folderNodes[folderID]}
		}

		pterm.EnableStyling()
		pterm.DisableColor()
		progressbar, err := pterm.DefaultProgressbar.WithTitle("Decrypting Resources").WithTotal(len(resources)).Start()
		if err != nil {
			return fmt.Errorf("progress: %w", err)
		}

		for _, resource := range resources {
			if len(resource.Secrets) == 0 {
				fmt.Printf("\nSkipping Export of Resource %v %v Because it has no Secret\n", resource.ID, resource.Name)
				progressbar.Increment()
				continue
			}

			entry, err := getKeepassEntry(client, resource, resource.Secrets[0], resource.ResourceType)
			if err != nil {
				fmt.Printf("\nSkipping Export of Resource %v %v Because of: %v\n", resource.ID, resource.Name, err)
				progressbar.Increment()
				continue
			}

			if includeComments {
				comments, err := client.GetComments(ctx, resource.ID, &api.GetCommentsOptions{})
				if err != nil {
					// Comments are an extra; a failed fetch shouldn't sink
					// the whole export.
					fmt.Printf("\nExporting Resource %v without Comments Because of: %v\n", resource.ID, err)
				} else {
					appendComments(entry, comments)
				}
			}

			// An empty or unknown FolderParentID (folder not shared with
			// this user) lands the entry in the root group.
			target := rootNode
			if node, ok := folderNodes[resource.FolderParentID]; ok {
				target = node
			}
			target.entries = append(target.entries, *entry)
			progressbar.Increment()
		}

		db := gokeepasslib.NewDatabase(kdbxVersion)
		db.Content.Meta.DatabaseName = "Passbolt Export"
		db.Credentials = gokeepasslib.NewPasswordCredentials(keepassPassword)

		// gokeepasslib's KDF defaults (6k AES-KDF rounds / 1 MiB Argon2
		// memory) are far below the KeePass 2.x defaults and trigger its
		// "weak key transformation settings" warning; match KeePass instead.
		switch kdbxVersionFlag {
		case "v3":
			db.Header.FileHeaders.TransformRounds = 600_000
		case "v4":
			db.Header.FileHeaders.KdfParameters.Memory = 64 * 1024 * 1024
		}

		db.Content.Root = &gokeepasslib.RootData{
			Groups: []gokeepasslib.Group{rootNode.toGroup()},
		}

		if err := db.LockProtectedEntries(); err != nil {
			return fmt.Errorf("locking protected entries: %w", err)
		}

		keepassEncoder := gokeepasslib.NewEncoder(file)
		if err := keepassEncoder.Encode(db); err != nil {
			return fmt.Errorf("encoding kdbx: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("writing kdbx: %w", err)
		}
		if err := os.Rename(tmpName, filename); err != nil {
			return fmt.Errorf("moving kdbx into place: %w", err)
		}
		committed = true
		fmt.Println("Done")

		return nil
	})
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

	addTotp(&entry, secretFields, name, username, uri)

	// Custom fields: each one becomes a protected KDBX field, matching what the
	// Passbolt browser extension does in resourcesKdbxExporter.setCustomFields.
	// helper.ParseCustomFields correlates the metadata and secret halves by id.
	addCustomFields(&entry, metadata, secretFields)

	addTags(&entry, resource)
	addTimes(&entry, resource)

	return &entry, nil
}

// addTotp exports the TOTP configuration in both conventions: an otpauth://
// URI in the "otp" field (read natively by KeePassXC and by the KeeOtp2
// plugin) and the TimeOtp-* fields KeePass 2.x's built-in generator reads
// (Ctrl+T), so the codes work out of the box in either application.
func addTotp(entry *gokeepasslib.Entry, secretFields map[string]any, name, username, uri string) {
	totpRaw, ok := secretFields["totp"].(map[string]any)
	if !ok {
		return
	}
	secretKey, _ := totpRaw["secret_key"].(string)
	// Skip TOTP entry if secret_key is missing — can't build a valid OTP URI
	if secretKey == "" {
		return
	}

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

	// TimeOtp-Secret-Base32 expects the plain Base32 secret; Passbolt stores
	// it that way already, just normalize spacing and case.
	base32Secret := strings.ToUpper(strings.ReplaceAll(secretKey, " ", ""))
	entry.Values = append(entry.Values, gokeepasslib.ValueData{Key: "TimeOtp-Secret-Base32", Value: gokeepasslib.V{Content: base32Secret, Protected: w.NewBoolWrapper(true)}})
	if digits > 0 {
		entry.Values = append(entry.Values, gokeepasslib.ValueData{Key: "TimeOtp-Length", Value: gokeepasslib.V{Content: strconv.Itoa(digits)}})
	}
	if period > 0 {
		entry.Values = append(entry.Values, gokeepasslib.ValueData{Key: "TimeOtp-Period", Value: gokeepasslib.V{Content: strconv.Itoa(period)}})
	}
	if hmac := timeOtpAlgorithm(algorithm); hmac != "" {
		entry.Values = append(entry.Values, gokeepasslib.ValueData{Key: "TimeOtp-Algorithm", Value: gokeepasslib.V{Content: hmac}})
	}
}

// timeOtpAlgorithm maps a Passbolt TOTP algorithm name to the value KeePass
// expects in TimeOtp-Algorithm. SHA1 (and anything unknown) returns "" — it
// is the KeePass default and needs no field.
func timeOtpAlgorithm(algorithm string) string {
	switch strings.ToUpper(algorithm) {
	case "SHA256":
		return "HMAC-SHA-256"
	case "SHA512":
		return "HMAC-SHA-512"
	default:
		return ""
	}
}

// appendComments appends the Passbolt comment thread to the entry's Notes
// field — KDBX has no first-class comment concept. Replies are indented
// below their parent.
func appendComments(entry *gokeepasslib.Entry, comments []api.Comment) {
	if len(comments) == 0 {
		return
	}
	var b strings.Builder
	b.WriteString("Comments:")
	var walk func(cs []api.Comment, indent string)
	walk = func(cs []api.Comment, indent string) {
		for _, c := range cs {
			b.WriteString("\n" + indent)
			if c.Created != nil {
				b.WriteString(c.Created.Format("2006-01-02 15:04") + " ")
			}
			b.WriteString(c.Content)
			walk(c.Children, indent+"  ")
		}
	}
	walk(comments, "")

	for i := range entry.Values {
		if entry.Values[i].Key == "Notes" {
			if entry.Values[i].Value.Content != "" {
				entry.Values[i].Value.Content += "\n\n"
			}
			entry.Values[i].Value.Content += b.String()
			return
		}
	}
	entry.Values = append(entry.Values, gokeepasslib.ValueData{Key: "Notes", Value: gokeepasslib.V{Content: b.String()}})
}

// addTags joins the Passbolt tag slugs (kept outside the encrypted metadata)
// into the KDBX Tags field; KeePass separates entry tags with ';'.
func addTags(entry *gokeepasslib.Entry, resource api.Resource) {
	if len(resource.Tags) == 0 {
		return
	}
	slugs := make([]string, len(resource.Tags))
	for i, tag := range resource.Tags {
		slugs[i] = tag.Slug
	}
	entry.Tags = strings.Join(slugs, ";")
}

// addTimes carries the Passbolt timestamps into the entry instead of stamping
// everything with "now". gokeepasslib's NewTimeData points every time field
// at ONE shared wrapper, so each field gets its own copy before the Time
// value is replaced — mutating in place would change all fields at once. The
// copies keep the KDBX-version-dependent formatting flag.
func addTimes(entry *gokeepasslib.Entry, resource api.Resource) {
	set := func(target **w.TimeWrapper, value *api.Time) {
		if value == nil {
			return
		}
		wrapper := **target
		wrapper.Time = value.Time
		*target = &wrapper
	}
	set(&entry.Times.CreationTime, resource.Created)
	set(&entry.Times.LastModificationTime, resource.Modified)
	if resource.Expired != nil {
		expiry := *entry.Times.CreationTime
		expiry.Time = resource.Expired.Time
		entry.Times.ExpiryTime = &expiry
		entry.Times.Expires = w.NewBoolWrapper(true)
	}
}

func addCustomFields(entry *gokeepasslib.Entry, metadata, secretFields map[string]any) {
	for _, cf := range helper.ParseCustomFields(metadata, secretFields) {
		if cf.Name == "" {
			continue
		}
		entry.Values = append(entry.Values, gokeepasslib.ValueData{
			Key:   cf.Name,
			Value: gokeepasslib.V{Content: cf.Value, Protected: w.NewBoolWrapper(true)},
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
