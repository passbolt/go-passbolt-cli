package resource

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
	"golang.org/x/exp/slices"
)

var ResourceToClipboardCommand = &cobra.Command{
	Use:     "resource",
	Short:   "Copies Resorces column entries to the clipboard.",
	Aliases: []string{"resources"},
	RunE:    ResourceToClipboard,
}

func init() {
	ResourceToClipboardCommand.Flags().StringArrayP("column", "c", []string{"Username", "Password"}, "Columns to return, possible Columns:\nID, FolderParentID, Name, Username, URI, Password, Description, CreatedTimestamp, ModifiedTimestamp")
}

func ResourceToClipboard(cmd *cobra.Command, args []string) error {
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("You need to specify atleast one column to return")
	}
	celFilter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return err
	}
	delay, err := cmd.Flags().GetInt("delay")
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

	resources, err := client.GetResources(ctx, &api.GetResourcesOptions{})
	if err != nil {
		return fmt.Errorf("Listing Resource: %w", err)
	}

	resources, err = filterResources(&resources, celFilter, ctx, client)
	if err != nil {
		return err
	}

	singleResource, err := choseResourceEntry(&resources)
	if err != nil {
		return err
	}
	if singleResource == nil {
		return nil
	}

	err = clipboard.Init()
	if err != nil {
		return err
	}
	util.SetDelay(delay)

	var password string
	var description string
	if slices.Contains(columns, "Password") ||
		slices.Contains(columns, "password") ||
		slices.Contains(columns, "description") ||
		slices.Contains(columns, "Description") {
		_, _, _, _, password, description, err = helper.GetResource(ctx, client, singleResource.ID)
		if err != nil {
			return fmt.Errorf("Get Resource %w", err)
		}
	}

	for _, c := range columns {
		switch strings.ToLower(c) {
		case "id":
			util.CopyValueToClipboard(c, singleResource.ID)
		case "folderparentid":
			util.CopyValueToClipboard(c, singleResource.FolderParentID)
		case "name":
			util.CopyValueToClipboard(c, shellescape.StripUnsafe(singleResource.Name))
		case "username":
			util.CopyValueToClipboard(c, shellescape.StripUnsafe(singleResource.Username))
		case "uri":
			util.CopyValueToClipboard(c, shellescape.StripUnsafe(singleResource.URI))
		case "password":
			util.CopyValueToClipboard(c, shellescape.StripUnsafe(password))
		case "description":
			util.CopyValueToClipboard(c, shellescape.StripUnsafe(description))
		case "createdtimestamp":
			util.CopyValueToClipboard(c, singleResource.Created.Format(time.RFC3339))
		case "modifiedtimestamp":
			util.CopyValueToClipboard(c, singleResource.Modified.Format(time.RFC3339))
		default:
			cmd.SilenceUsage = false
			return fmt.Errorf("Unknown Column: %v", c)
		}
	}
	return nil
}

// If more than one resource are selected, print an table an ask for selection of entry
func choseResourceEntry(resources *[]api.Resource) (*api.Resource, error) {
	if len(*resources) == 0 {
		return nil, fmt.Errorf("No resources to select!")
	}

	if len(*resources) == 1 {
		return &(*resources)[0], nil
	}

	data := pterm.TableData{[]string{"Index", "Name", "Username", "URI"}}
	for i, resource := range *resources {
		index := i + 1
		data = append(data, []string{strconv.Itoa(index), resource.Name, resource.Username, resource.URI})
	}

	var selectedResource *api.Resource
	for selectedResource == nil {
		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
		fmt.Print("Please chose an index of resource (c to abbort): ")
		var cliInput string
		fmt.Scanln(&cliInput)
		if cliInput == "c" {
			return nil, nil
		}

		selectedIndex, err := strconv.Atoi(cliInput)
		if err != nil || selectedIndex <= 0 || selectedIndex > len(*resources) {
			fmt.Printf("Input %s is not a valid index!\n", cliInput)
			fmt.Println()
			continue
		}

		selectedResource = &(*resources)[selectedIndex-1]
	}

	return selectedResource, nil
}
