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
	// Retrieve folder information with resources
	folders, err := client.GetFolders(ctx, &api.GetFoldersOptions{
		ContainChildrenResources: true,
		ContainChildrenFolders:   true,
	})
	if err != nil {
		return fmt.Errorf("Getting Folders: %w", err)
	}

	// Also get all resources with secrets to ensure we have complete data
	allResources, err := client.GetResources(ctx, &api.GetResourcesOptions{
		ContainSecret:       true,
		ContainResourceType: true,
	})
	if err != nil {
		return fmt.Errorf("Getting Resources: %w", err)
	}

	// Create a map of resources by ID for easy lookup
	resourceMap := make(map[string]api.Resource)
	for _, resource := range allResources {
		resourceMap[resource.ID] = resource
	}

	// Create root group
	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = "Passbolt"

	// Create maps to track folders and their relationships
	folderMap := make(map[string]*api.Folder)

	// First, store all folders in a map for easy lookup
	for i := range folders {
		folderMap[folders[i].ID] = &folders[i]
	}

	// Debug output
	fmt.Printf("\nFound %d folders and %d resources\n", len(folders), len(allResources))

	// Function to recursively build the folder structure
	var buildFolderStructure func(parentGroupPtr *gokeepasslib.Group, folderID string)
	buildFolderStructure = func(parentGroupPtr *gokeepasslib.Group, folderID string) {
		folder, exists := folderMap[folderID]
		if !exists {
			return
		}

		// Create a new group for this folder
		group := gokeepasslib.NewGroup()
		group.Name = folder.Name

		fmt.Printf("\nProcessing folder: %s (ID: %s) with %d child resources\n",
			folder.Name, folder.ID, len(folder.ChildrenResources))

		// Add resources to this folder's group
		for _, folderResource := range folder.ChildrenResources {
			// Look up the full resource with secrets
			resource, exists := resourceMap[folderResource.ID]
			if !exists {
				fmt.Printf("\nResource %s (%s) exists in folder but not in resource list\n",
					folderResource.ID, folderResource.Name)
				continue
			}

			if len(resource.Secrets) == 0 {
				fmt.Printf("\nSkipping Export of Resource %v %v Because of: no secrets available\n",
					resource.ID, resource.Name)
				continue
			}

			entry, err := getKeepassEntry(client, resource, resource.Secrets[0], resource.ResourceType)
			if err != nil {
				fmt.Printf("\nSkipping Export of Resource %v %v Because of: %v\n",
					resource.ID, resource.Name, err)
				continue
			}

			group.Entries = append(group.Entries, *entry)
			fmt.Printf("Added resource: %s to folder: %s\n", resource.Name, folder.Name)
		}

		// Process child folders
		for _, childFolder := range folders {
			if childFolder.FolderParentID == folderID {
				buildFolderStructure(&group, childFolder.ID)
			}
		}

		// Add this group to its parent
		parentGroupPtr.Groups = append(parentGroupPtr.Groups, group)
	}

	// Identify top-level folders (those without parents or with parents outside our folder list)
	for _, folder := range folders {
		if folder.FolderParentID == "" || folderMap[folder.FolderParentID] == nil {
			buildFolderStructure(&rootGroup, folder.ID)
		}
	}

	// Handle resources that are not in any folder (if any)
	resourcesWithoutFolder, err := client.GetResources(ctx, &api.GetResourcesOptions{
		ContainSecret:       true,
		ContainResourceType: true,
	})
	if err != nil {
		return fmt.Errorf("Getting Resources without folders: %w", err)
	}

	// Create a group for resources without folders
	noFolderGroup := gokeepasslib.NewGroup()
	noFolderGroup.Name = "Unfiled Resources"
	hasUnfiledResources := false

	for _, resource := range resourcesWithoutFolder {
		// Skip resources that are already in folders
		inFolder := false
		for _, folder := range folders {
			for _, folderResource := range folder.ChildrenResources {
				if folderResource.ID == resource.ID {
					inFolder = true
					break
				}
			}
			if inFolder {
				break
			}
		}

		if !inFolder {
			if len(resource.Secrets) == 0 {
				fmt.Printf("\nSkipping Export of Resource %v %v Because of: no secrets available\n", resource.ID, resource.Name)
				continue
			}

			entry, err := getKeepassEntry(client, resource, resource.Secrets[0], resource.ResourceType)
			if err != nil {
				fmt.Printf("\nSkipping Export of Resource %v %v Because of: %v\n", resource.ID, resource.Name, err)
				continue
			}

			noFolderGroup.Entries = append(noFolderGroup.Entries, *entry)
			hasUnfiledResources = true
		}
	}

	// Add the unfiled resources group if it has entries
	if hasUnfiledResources {
		rootGroup.Groups = append(rootGroup.Groups, noFolderGroup)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Creating File: %w", err)
	}
	defer file.Close()

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
	if len(resource.Secrets) == 0 {
		return nil, fmt.Errorf("no secrets available")
	}

	// Debug output for resource type
	fmt.Printf("Processing resource: %s, Type Slug: %s\n",
		resource.Name,
		resource.ResourceType.Slug)

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

	// Check if this is a TOTP resource
	hasTOTP := resource.ResourceType.Slug == "password-description-totp" ||
		resource.ResourceType.Slug == "totp"

	if hasTOTP {
		fmt.Printf("Found TOTP resource: %s\n", resource.Name)
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
			fmt.Printf("Parsed password-description-totp data for %s\n", resource.Name)
		} else {
			var secretData api.SecretDataTypeTOTP
			err = json.Unmarshal([]byte(rawSecretData), &secretData)
			if err != nil {
				return nil, fmt.Errorf("Parsing Decrypted Secret Data: %w", err)
			}
			totpData = secretData.TOTP
			fmt.Printf("Parsed totp data for %s\n", resource.Name)
		}

		// Verify TOTP data
		if totpData.SecretKey == "" {
			fmt.Printf("Warning: TOTP secret key is empty for %s\n", resource.Name)
		}

		v := url.Values{}
		v.Set("secret", totpData.SecretKey)
		v.Set("period", strconv.FormatUint(uint64(totpData.Period), 10))
		v.Set("algorithm", totpData.Algorithm)
		v.Set("digits", fmt.Sprint(totpData.Digits))

		issuer := resource.URI
		if issuer == "" {
			issuer = resource.Name
		}
		v.Set("issuer", issuer)

		accountName := resource.Username
		if accountName == "" {
			accountName = resource.Name
		}

		u := url.URL{
			Scheme:   "otpauth",
			Host:     "totp",
			Path:     "/" + issuer + ":" + accountName,
			RawQuery: encodeQuery(v),
		}

		otpURL := u.String()
		fmt.Printf("Generated OTP URL for %s: %s\n", resource.Name, otpURL)

		entry.Values = append(entry.Values,
			gokeepasslib.ValueData{
				Key: "otp",
				Value: gokeepasslib.V{
					Content:   otpURL,
					Protected: w.NewBoolWrapper(true),
				},
			},
		)
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
