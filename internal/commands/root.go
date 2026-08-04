package commands

import (
	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute(version string) error {
	return NewRootCmd(version).Execute()
}

// NewRootCmd builds the CLI root command.
func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "qdivzero",
		Short:         "QDivZero API CLI",
		Long:          "Command-line client for the QDivZero API (https://api.qdiv0.com).",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.Version = version
	root.SetVersionTemplate("qdivzero {{.Version}}\n")
	return root
}
