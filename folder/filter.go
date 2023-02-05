package folder

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/passbolt/go-passbolt/api"
)

// Environments for CEl
var celEnvOptions = []cel.EnvOption{
	cel.Variable("ID", cel.StringType),
	cel.Variable("FolderParentID", cel.StringType),
	cel.Variable("Name", cel.StringType),
	cel.Variable("CreatedTimestamp", cel.TimestampType),
	cel.Variable("ModifiedTimestamp", cel.TimestampType),
}

func filterFolders(folders *[]api.Folder, celCmd string, ctx context.Context) ([]api.Folder, error) {
	if celCmd == "" {
		return *folders, nil
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

	filteredFolders := []api.Folder{}
	for _, folder := range *folders {
		val, _, err := program.ContextEval(ctx, map[string]any{
			"ID":                folder.ID,
			"FolderParentID":    folder.FolderParentID,
			"Name":              folder.Name,
			"CreatedTimestamp":  folder.Created.Time,
			"ModifiedTimestamp": folder.Modified.Time,
		})

		if err != nil {
			return nil, err
		}

		if val.Value() == true {
			filteredFolders = append(filteredFolders, folder)
		}
	}

	if len(filteredFolders) == 0 {
		return nil, fmt.Errorf("No such folders found with filter %v!", celCmd)
	}

	return filteredFolders, nil
}
