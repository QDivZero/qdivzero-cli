package commands

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func newVersionCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI and SDK versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(deps.Stdout, "qdivzero CLI %s (SDK %s)\n", deps.Version, sdkVersion())
			return nil
		},
	}
}

// sdkVersion resolves the qdivzero-go module version from build info.
func sdkVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, m := range info.Deps {
		if m.Path == "github.com/QDivZero/qdivzero-go" {
			return m.Version
		}
	}
	return "unknown"
}
