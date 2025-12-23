package util

import "github.com/google/cel-go/cel"

// InitCELProgram - Initialize a CEL program with given CEL command and a set of environments
func InitCELProgram(celCmd string, options ...cel.EnvOption) (*cel.Program, error) {
	env, err := cel.NewEnv(options...)
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

	return &program, nil
}

// CELExpressionReferencesFields checks if a CEL expression references any of the given field names.
// Returns true if the expression references at least one of the specified fields.
func CELExpressionReferencesFields(celCmd string, fieldNames []string, options ...cel.EnvOption) (bool, error) {
	if celCmd == "" {
		return false, nil
	}

	env, err := cel.NewEnv(options...)
	if err != nil {
		return false, err
	}

	ast, issue := env.Compile(celCmd)
	if issue.Err() != nil {
		return false, issue.Err()
	}

	// Build a set of field names to check
	fieldSet := make(map[string]bool)
	for _, name := range fieldNames {
		fieldSet[name] = true
	}

	// Get the native AST representation which has ReferenceMap
	nativeAST := ast.NativeRep()
	refMap := nativeAST.ReferenceMap()
	for _, refInfo := range refMap {
		if fieldSet[refInfo.Name] {
			return true, nil
		}
	}
	return false, nil
}
