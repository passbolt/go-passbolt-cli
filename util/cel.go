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
