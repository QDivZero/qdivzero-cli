// Package accounts implements the "qdivzero accounts" service commands.
package accounts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-go"
)

// New returns the accounts service command tree.
func New(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List accounts",
	}
	cmd.AddCommand(newListCmd(deps))
	return cmd
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// membershipRow renders one AccountMembershipResponse as a table row.
func membershipRow(m *qdivzero.AccountMembershipResponse) []string {
	return []string{
		str(m.AccountId),
		str(m.Role),
		str(m.UserId),
	}
}

func newListCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List user accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			resp, err := api.GetAccountsWithResponse(ctx)
			if err != nil {
				return fmt.Errorf("accounts list: %w", err)
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("accounts list: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON200 != nil && resp.JSON200.Memberships != nil {
				for i := range *resp.JSON200.Memberships {
					rows = append(rows, membershipRow(&(*resp.JSON200.Memberships)[i]))
				}
			}
			return deps.Render.Render(
				[]string{"ACCOUNT", "ROLE", "USER"},
				rows,
				resp.JSON200,
			)
		},
	}
}
