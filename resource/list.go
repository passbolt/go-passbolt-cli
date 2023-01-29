package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

// ResourceListCmd Lists a Passbolt Resource
var ResourceListCmd = &cobra.Command{
	Use:     "resource",
	Short:   "Lists Passbolt Resources",
	Long:    `Lists Passbolt Resources`,
	Aliases: []string{"resources"},
	RunE:    ResourceList,
}

var celEnvOptions = []cel.EnvOption{
	cel.Variable("ID", cel.StringType),
	cel.Variable("FolderParentID", cel.StringType),
	cel.Variable("Name", cel.StringType),
	cel.Variable("Username", cel.StringType),
	cel.Variable("URI", cel.StringType),
	cel.Variable("Password", cel.StringType),
	cel.Variable("Description", cel.StringType),
	cel.Variable("CreatedTimestamp", cel.TimestampType),
	cel.Variable("ModifiedTimestamp", cel.TimestampType),
	cel.Function("parseTimestamp",
		cel.Overload("parse_timestamp_string",
			[]*cel.Type{cel.StringType},
			cel.TimestampType,
			cel.UnaryBinding(func(timeStampInput ref.Val) ref.Val {
				timeStampString := fmt.Sprintf("%s", timeStampInput.Value())
				timeStamp, err := time.Parse(`2006-01-02+15:04:05`, timeStampString)
				if err != nil {
					fmt.Printf("Error while parsing timestamp: %v\n", err)
				}
				return types.Timestamp{Time: timeStamp}
			},
			),
		),
	),
}

func init() {
	ResourceListCmd.Flags().Bool("favorite", false, "Resources that are marked as favorite")
	ResourceListCmd.Flags().Bool("own", false, "Resources that are owned by me")
	ResourceListCmd.Flags().String("filter", "", "Define a CEl expression as filter for resources.")

	ResourceListCmd.Flags().StringP("group", "g", "", "Resources that are shared with group")
	ResourceListCmd.Flags().StringArrayP("folder", "f", []string{}, "Resources that are in folder")

	ResourceListCmd.Flags().StringArrayP("column", "c", []string{"ID", "FolderParentID", "Name", "Username", "URI"}, "Columns to return, possible Columns:\nID, FolderParentID, Name, Username, URI, Password, Description, CreatedTimestamp, ModifiedTimestamp")
}

func ResourceList(cmd *cobra.Command, args []string) error {
	favorite, err := cmd.Flags().GetBool("favorite")
	if err != nil {
		return err
	}
	own, err := cmd.Flags().GetBool("own")
	if err != nil {
		return err
	}
	group, err := cmd.Flags().GetString("group")
	if err != nil {
		return err
	}
	folderParents, err := cmd.Flags().GetStringArray("folder")
	if err != nil {
		return err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("You need to specify atleast one column to return")
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	celFilter, err := cmd.Flags().GetString("filter")
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

	resources, err := client.GetResources(ctx, &api.GetResourcesOptions{
		FilterIsFavorite:        favorite,
		FilterIsOwnedByMe:       own,
		FilterIsSharedWithGroup: group,
		FilterHasParent:         folderParents,
	})
	if err != nil {
		return fmt.Errorf("Listing Resource: %w", err)
	}

	resources, err = filterResources(&resources, celFilter, ctx, client)
	if err != nil {
		return err
	}

	if jsonOutput {
		outputResources := []ResourceJsonOutput{}
		for i := range resources {
			_, _, _, _, pass, desc, err := helper.GetResource(ctx, client, resources[i].ID)
			if err != nil {
				return fmt.Errorf("Get Resource %w", err)
			}
			outputResources = append(outputResources, ResourceJsonOutput{
				ID:                &resources[i].ID,
				FolderParentID:    &resources[i].FolderParentID,
				Name:              &resources[i].Name,
				Username:          &resources[i].Username,
				URI:               &resources[i].URI,
				Password:          &pass,
				Description:       &desc,
				CreatedTimestamp:  &resources[i].Created.Time,
				ModifiedTimestamp: &resources[i].Modified.Time,
			})
		}
		jsonResources, err := json.MarshalIndent(outputResources, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonResources))
	} else {
		data := pterm.TableData{columns}

		for _, resource := range resources {
			entry := make([]string, len(columns))
			for i := range columns {
				switch strings.ToLower(columns[i]) {
				case "id":
					entry[i] = resource.ID
				case "folderparentid":
					entry[i] = resource.FolderParentID
				case "name":
					entry[i] = shellescape.StripUnsafe(resource.Name)
				case "username":
					entry[i] = shellescape.StripUnsafe(resource.Username)
				case "uri":
					entry[i] = shellescape.StripUnsafe(resource.URI)
				case "password":
					_, _, _, _, pass, _, err := helper.GetResource(ctx, client, resource.ID)
					if err != nil {
						return fmt.Errorf("Get Resource %w", err)
					}
					entry[i] = shellescape.StripUnsafe(pass)
				case "description":
					_, _, _, _, _, desc, err := helper.GetResource(ctx, client, resource.ID)
					if err != nil {
						return fmt.Errorf("Get Resource %w", err)
					}
					entry[i] = shellescape.StripUnsafe(desc)
				case "createdtimestamp":
					entry[i] = resource.Created.Format(time.RFC3339)
				case "modifiedtimestamp":
					entry[i] = resource.Modified.Format(time.RFC3339)
				default:
					cmd.SilenceUsage = false
					return fmt.Errorf("Unknown Column: %v", columns[i])
				}
			}
			data = append(data, entry)
		}

		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	}
	return nil
}

func filterResources(resources *[]api.Resource, celCmd string, ctx context.Context, client *api.Client) ([]api.Resource, error) {
	if celCmd == "" {
		return *resources, nil
	}

	env, err := cel.NewEnv(celEnvOptions...)
	if err != nil {
		return nil, err
	}

	ast, issue := env.Compile(celCmd)
	if issue.Err() != nil {
		return nil, issue.Err()
	}

	program, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	filteredResources := []api.Resource{}
	for _, resource := range *resources {
		val, _, err := program.ContextEval(ctx, map[string]any{
			"Id":             resource.ID,
			"FolderParentID": resource.FolderParentID,
			"Name":           resource.Name,
			"Username":       resource.Username,
			"URI":            resource.URI,
			"Password": func() ref.Val {
				_, _, _, _, pass, _, err := helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					fmt.Printf("Get Resource %v", err)
					return types.String("")
				}
				return types.String(pass)
			},
			"Description": func() ref.Val {
				_, _, _, _, _, descr, err := helper.GetResource(ctx, client, resource.ID)
				if err != nil {
					fmt.Printf("Get Resource %v", err)
					return types.String("")
				}
				return types.String(descr)
			},
			"CreatedTimestamp":  resource.Created.Time,
			"ModifiedTimestamp": resource.Modified.Time,
		})

		if err != nil {
			return nil, err
		}

		if val.Value() == true {
			filteredResources = append(filteredResources, resource)
		}
	}

	if len(filteredResources) == 0 {
		return nil, fmt.Errorf("No such Resources found with filter %v!", celCmd)
	}
	return filteredResources, nil
}
