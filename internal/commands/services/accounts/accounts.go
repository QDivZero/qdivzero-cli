// Package accounts implements the "qdivzero accounts" service commands.
package accounts

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-go"
)

// New returns the accounts service command tree.
func New(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List and select accounts",
	}
	cmd.AddCommand(newListCmd(deps), newUseCmd(deps))
	return cmd
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// membershipRow renders one AccountMembershipResponse as a table row; the
// active account (X-Account-Id) is marked with "*".
func membershipRow(m *qdivzero.AccountMembershipResponse, active string) []string {
	id := str(m.AccountId)
	if id == active {
		id += " *"
	}
	return []string{
		id,
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
			active := deps.Config.AccountId
			var rows [][]string
			if resp.JSON200 != nil && resp.JSON200.Memberships != nil {
				for i := range *resp.JSON200.Memberships {
					rows = append(rows, membershipRow(&(*resp.JSON200.Memberships)[i], active))
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

// newUseCmd selects the active account (stored as the X-Account-Id header).
// With an argument it stores it directly; without one it lists the user's
// accounts interactively and lets them pick.
func newUseCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "use [account-id]",
		Short: "Select the active account (sent as X-Account-Id)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID := ""
			if len(args) == 1 {
				accountID = args[0]
			} else {
				ctx := cmd.Context()
				api, err := deps.Client(ctx)
				if err != nil {
					return err
				}
				resp, err := api.GetAccountsWithResponse(ctx)
				if err != nil {
					return fmt.Errorf("accounts use: %w", err)
				}
				if resp.StatusCode() != 200 {
					return fmt.Errorf("accounts use: status %d", resp.StatusCode())
				}
				var memberships []qdivzero.AccountMembershipResponse
				if resp.JSON200 != nil && resp.JSON200.Memberships != nil {
					memberships = *resp.JSON200.Memberships
				}
				if len(memberships) == 0 {
					return fmt.Errorf("accounts use: no accounts found")
				}
				fmt.Fprintln(deps.Stdout, "Select an account:")
				for i := range memberships {
					fmt.Fprintf(deps.Stdout, "  [%d] %s (role: %s)\n", i+1, str(memberships[i].AccountId), str(memberships[i].Role))
				}
				fmt.Fprint(deps.Stdout, "Account number: ")
				line, _ := bufio.NewReader(deps.Stdin).ReadString('\n')
				var n int
				if _, err := fmt.Sscanf(strings.TrimSpace(line), "%d", &n); err != nil || n < 1 || n > len(memberships) {
					return fmt.Errorf("accounts use: invalid selection")
				}
				accountID = str(memberships[n-1].AccountId)
			}
			if accountID == "" {
				return fmt.Errorf("accounts use: account id is required")
			}
			creds, err := config.Read()
			if err != nil {
				return err
			}
			creds.AccountId = accountID
			if err := config.Write(creds, true); err != nil {
				return err
			}
			fmt.Fprintf(deps.Stdout, "active account: %s\n", accountID)
			return nil
		},
	}
}
