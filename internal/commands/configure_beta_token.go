package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/config"
)

func newConfigureBetaTokenCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure-beta-token <token>",
		Short: "Set the temporary X-Private-Beta-Token header (empty to clear)",
		Long: "Stores the private beta token in ~/.qdivzero/credentials; it is sent " +
			"as the X-Private-Beta-Token header on every request. Temporary until " +
			"the private beta gate is removed.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := config.Read()
			if err != nil {
				return err
			}
			creds.PrivateBetaToken = args[0]
			if err := config.Write(creds, true); err != nil {
				return err
			}
			if args[0] == "" {
				fmt.Fprintln(deps.Stdout, "cleared: X-Private-Beta-Token")
			} else {
				fmt.Fprintln(deps.Stdout, "configured: X-Private-Beta-Token")
			}
			return nil
		},
	}
	return cmd
}
