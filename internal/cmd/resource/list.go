package resource

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// decryptedResource holds the result of decrypting a single resource
type decryptedResource struct {
	index    int
	resource api.Resource
	// typeSlug lets the collector tally skipped types without a second client lookup.
	typeSlug       string
	name           string
	username       string
	uri            string
	password       string
	description    string
	metadataFields map[string]any
	secretFields   map[string]any
	err            error
}

// ResourceListCmd Lists a Passbolt Resource
var ResourceListCmd = &cobra.Command{
	Use:     "resource",
	Short:   "Lists Passbolt Resources",
	Long:    `Lists Passbolt Resources`,
	Aliases: []string{"resources"},
	RunE:    ResourceList,
}

func init() {
	flags := ResourceListCmd.Flags()
	flags.Bool("favorite", false, "Resources that are marked as favorite")
	flags.Bool("own", false, "Resources that are owned by me")
	flags.StringP("group", "g", "", "Resources that are shared with group")
	flags.StringArrayP("folder", "f", []string{}, "Resources that are in folder")
	flags.StringArrayP("column", "c", resourceColumns.DefaultTableColumns(), "Columns to return (default list only for table format; JSON format includes all fields by default).\nPossible Columns: "+strings.Join(resourceColumns.Resolver().Canonical(), ", ")+"\nLegacy PascalCase column names (ID, FolderParentID, ...) remain accepted for backwards compatibility.")
}

type resourceListConfig struct {
	favorite       bool
	own            bool
	group          string
	folderParents  []string
	columns        []string
	columnsChanged bool
	jsonOutput     bool
	celFilter      string
}

func ResourceList(cmd *cobra.Command, args []string) error {
	config, err := parseResourceListFlags(cmd)
	if err != nil {
		return err
	}

	// Check if we need to fetch secrets (expensive server join + RSA decryption)
	// For v5 resources, metadata (name, username, uri) can be decrypted without secrets
	needSecrets := resourceColumns.RequiresSecrets(config.columns)

	// Check if CEL filter references any secret-bearing column (canonical or alias).
	if !needSecrets && config.celFilter != "" {
		refsSecrets, err := util.CELExpressionReferencesFields(config.celFilter, resourceColumns.SecretCelNames(), resourceColumns.CelEnvOptions()...)
		if err != nil {
			return fmt.Errorf("parsing filter: %w", err)
		}
		needSecrets = refsSecrets
	}

	return util.WithClient(cmd, func(ctx context.Context, client *api.Client) error {
		resources, err := client.GetResources(ctx, &api.GetResourcesOptions{
			FilterIsFavorite:        config.favorite,
			FilterIsOwnedByMe:       config.own,
			FilterIsSharedWithGroup: config.group,
			FilterHasParent:         config.folderParents,
			ContainSecret:           needSecrets,
		})
		if err != nil {
			return fmt.Errorf("listing Resource: %w", err)
		}

		// Decrypt all resources in parallel
		decrypted, skippedTypes, err := decryptResourcesParallel(ctx, client, resources, needSecrets)
		if err != nil {
			return err
		}

		// Apply CEL filter on already-decrypted data
		if config.celFilter != "" {
			decrypted, err = resourceColumns.Filter(ctx, decrypted, config.celFilter)
			if err != nil {
				return err
			}
		}

		var printErr error
		if config.jsonOutput {
			printErr = printJSONResources(decrypted, config.columnsChanged, config.columns)
		} else {
			printErr = printTableResources(decrypted, config.columns)
		}
		if printErr != nil {
			return printErr
		}

		// Last, and on stderr, so it survives a long table and leaves --json machine-readable.
		fmt.Fprint(os.Stderr, formatSkippedTypes(skippedTypes))
		return nil
	})
}

// decryptResourcesParallel decrypts resources with a worker pool, returning the successes plus a
// per-slug tally of resources skipped for want of a schema.
func decryptResourcesParallel(ctx context.Context, client *api.Client, resources []api.Resource, needSecrets bool) ([]decryptedResource, map[string]int, error) {
	// Use parallel decryption with worker pool
	numWorkers := min(
		// Limit Worker count to Resource count
		len(resources), int(viper.GetUint("workers")))

	// Filter resources - only require secrets if we're fetching them
	var validResources []api.Resource
	for i := range resources {
		if needSecrets && len(resources[i].Secrets) == 0 {
			continue
		}
		validResources = append(validResources, resources[i])
	}

	if len(validResources) == 0 {
		return []decryptedResource{}, nil, nil
	}

	// Channel for work items and results
	// Note: Session keys are pre-fetched during Login() when the server supports v5 metadata,
	// so no additional prefetching is needed here.
	jobs := make(chan int, len(validResources))
	results := make(chan decryptedResource, len(validResources))

	// Start workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			for idx := range jobs {
				resource := validResources[idx]

				// Lookup resource type from cache (single API call for all types). A type the
				// server no longer advertises has been disabled or deleted there; its resources
				// are skipped, tallied under the type ID since no slug is known.
				rType, err := client.GetResourceTypeCached(ctx, resource.ResourceTypeID)
				if err != nil {
					result := decryptedResource{index: idx, resource: resource, err: fmt.Errorf("get ResourceType: %w", err)}
					if errors.Is(err, api.ErrResourceTypeNotFound) {
						result.typeSlug = "type " + resource.ResourceTypeID + " (disabled on the server)"
						result.err = fmt.Errorf("%w: %w", helper.ErrUnsupportedResourceType, err)
					}
					results <- result
					continue
				}

				// For v4 resources without secret decryption, use plaintext fields directly
				// This avoids unnecessary function calls for 10k+ resources
				isV5 := strings.HasPrefix(rType.Slug, "v5-")

				// Skip a type this build has no schema for before decrypting anything. Only v5
				// metadata and JSON secrets need a schema; a v4 resource's plaintext columns are
				// readable without one, so those are still served by the fast path below.
				if (isV5 || needSecrets) && !api.HasResourceSchema(rType.Slug) {
					results <- decryptedResource{
						index:    idx,
						resource: resource,
						typeSlug: rType.Slug,
						err:      fmt.Errorf("%w: %s", helper.ErrUnsupportedResourceType, rType.Slug),
					}
					continue
				}

				if !needSecrets && !isV5 {
					// V4 resource - metadata is plaintext, no decryption needed
					results <- decryptedResource{
						index:       idx,
						resource:    resource,
						typeSlug:    rType.Slug,
						name:        resource.Name,
						username:    resource.Username,
						uri:         resource.URI,
						password:    "",
						description: resource.Description,
						metadataFields: map[string]any{
							"name":        resource.Name,
							"username":    resource.Username,
							"uri":         resource.URI,
							"description": resource.Description,
						},
					}
					continue
				}

				// Handle case where secrets weren't fetched
				var secret api.Secret
				if len(resource.Secrets) > 0 {
					secret = resource.Secrets[0]
				}

				_, metaFields, secFields, err := helper.GetResourceFieldMaps(
					client,
					resource,
					secret,
					*rType,
					needSecrets,
				)
				results <- decryptedResource{
					index:          idx,
					resource:       resource,
					typeSlug:       rType.Slug,
					name:           helper.GetStringField(metaFields, "name"),
					username:       helper.GetStringField(metaFields, "username"),
					uri:            helper.GetStringField(metaFields, "uri"),
					password:       helper.GetStringField(secFields, "password"),
					description:    helper.GetStringField(metaFields, "description"),
					metadataFields: metaFields,
					secretFields:   secFields,
					err:            err,
				}
			}
		})
	}

	// Send jobs
	for i := range validResources {
		jobs <- i
	}
	close(jobs)

	// Wait for workers and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results first
	allResults := make([]decryptedResource, len(validResources))
	for result := range results {
		allResults[result.index] = result
	}

	// Process results. Only a Resource type this build has no schema for, or that the server
	// has disabled, is skipped: those are whole categories the user can do nothing about from
	// here. Any other failure, a stored document the schema rejects included, names the resource
	// and aborts the listing, so bad data is never silently left out.
	decrypted := make([]decryptedResource, 0, len(validResources))
	skippedTypes := make(map[string]int)

	for _, result := range allResults {
		if result.err != nil {
			if errors.Is(result.err, helper.ErrUnsupportedResourceType) {
				skippedTypes[result.typeSlug]++
				continue
			}
			return nil, nil, util.ExplainReadError("get Resource "+result.resource.ID, result.typeSlug, result.err)
		}
		decrypted = append(decrypted, result)
	}

	// Returned rather than printed so the caller can emit it after the output. The count includes
	// resources a --filter would have excluded: a skipped resource was never decrypted, so no CEL
	// expression can be evaluated against it.
	return decrypted, skippedTypes, nil
}

// formatSkippedTypes renders the stderr notice for resources whose type this build cannot read,
// or "" when nothing was skipped. Slugs are sorted so the output is deterministic.
func formatSkippedTypes(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}

	total := 0
	for _, count := range counts {
		total += count
	}

	noun := "resources"
	if total == 1 {
		noun = "resource"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Warning: %d %s ignored: their Resource type is unknown to this version of "+
		"go-passbolt-cli or disabled on the server\n", total, noun)
	for _, slug := range slices.Sorted(maps.Keys(counts)) {
		fmt.Fprintf(&b, "  - %s: %d\n", slug, counts[slug])
	}
	b.WriteString("Update go-passbolt-cli to include the types it does not know yet\n")
	return b.String()
}

func printJSONResources(
	decrypted []decryptedResource,
	isColumnsChanged bool,
	columns []string,
) error {
	outputResources := make([]ResourceJSONOutput, len(decrypted))
	for i, d := range decrypted {
		name := d.name
		username := d.username
		uri := d.uri
		pass := d.password
		desc := d.description
		output := ResourceJSONOutput{
			ID:                &d.resource.ID,
			FolderParentID:    &d.resource.FolderParentID,
			Name:              &name,
			Username:          &username,
			URI:               &uri,
			Password:          &pass,
			Description:       &desc,
			CreatedTimestamp:  &d.resource.Created.Time,
			ModifiedTimestamp: &d.resource.Modified.Time,
			Deleted:           d.resource.Deleted,
			Expired:           d.resource.Expired != nil,
			ResourceTypeID:    &d.resource.ResourceTypeID,
		}
		if d.resource.Expired != nil {
			output.ExpiredAt = &d.resource.Expired.Time
		}
		if len(d.metadataFields) > 0 {
			output.Metadata = d.metadataFields
		}
		if len(d.secretFields) > 0 {
			output.Secret = d.secretFields
		}
		outputResources[i] = output
	}

	if isColumnsChanged {
		return util.PrintJSONColumnFiltered(outputResources, columns)
	}

	return util.PrintJSON(outputResources)
}

func printTableResources(
	decrypted []decryptedResource,
	columns []string,
) error {
	// Input is normalized by parseResourceListFlags; a miss in the resolver is a
	// defensive guard against a future caller that skips that step.
	return util.PrintTable(columns, decrypted, resourceColumns.TableValue)
}

func parseResourceListFlags(cmd *cobra.Command) (*resourceListConfig, error) {
	favorite, err := cmd.Flags().GetBool("favorite")
	if err != nil {
		return nil, err
	}
	own, err := cmd.Flags().GetBool("own")
	if err != nil {
		return nil, err
	}
	group, err := cmd.Flags().GetString("group")
	if err != nil {
		return nil, err
	}
	folderParents, err := cmd.Flags().GetStringArray("folder")
	if err != nil {
		return nil, err
	}
	columns, err := cmd.Flags().GetStringArray("column")
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, util.NoColumnsError(resourceColumns.Resolver().Canonical())
	}
	columns, err = resourceColumns.Resolver().NormalizeAll(columns)
	if err != nil {
		return nil, err
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return nil, err
	}
	celFilter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return nil, err
	}

	return &resourceListConfig{
		favorite:       favorite,
		own:            own,
		group:          group,
		folderParents:  folderParents,
		columns:        columns,
		columnsChanged: cmd.Flags().Changed("column"),
		jsonOutput:     jsonOutput,
		celFilter:      celFilter,
	}, nil
}
