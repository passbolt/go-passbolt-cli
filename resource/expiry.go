package resource

import (
	"context"
	"fmt"

	"github.com/passbolt/go-passbolt/api"
)

// SetResourceExpiry updates only the expiry date of a resource.
func SetResourceExpiry(ctx context.Context, client *api.Client, id string, expired string) error {
	if expired == "" {
		return nil
	}
	_, _, err := client.DoCustomRequestAndReturnRawResponse(
		ctx,
		"PUT",
		fmt.Sprintf("resources/%s.json", id),
		"v2",
		map[string]string{"expired": expired},
		nil,
	)
	if err != nil {
		return fmt.Errorf("Setting expiry: %w", err)
	}
	return nil
}


