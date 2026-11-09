package cmd

import (
	"strings"
	"testing"
)

func TestOrganizationSettingsCommandTree(t *testing.T) {
	for _, operation := range []string{"get", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			command, remaining, err := rootCmd.Find([]string{operation, "organization-settings"})
			if err != nil {
				t.Fatal(err)
			}
			if command.Name() != "organization-settings" || len(remaining) != 0 {
				t.Fatalf("command=%s remaining=%v", command.CommandPath(), remaining)
			}
			if command.Flags().Lookup("plugin") == nil {
				t.Fatal("missing plugin flag")
			}
			err = command.ValidateRequiredFlags()
			if err == nil || !strings.Contains(err.Error(), "plugin") {
				t.Fatalf("missing plugin accepted: %v", err)
			}
			if operation == "update" && (command.Flags().Lookup("input-file") == nil || !strings.Contains(err.Error(), "input-file")) {
				t.Fatalf("missing required input-file: %v", err)
			}
			if operation == "delete" && command.Flags().Lookup("id") == nil {
				t.Fatal("missing settings ID flag")
			}
		})
	}
}
