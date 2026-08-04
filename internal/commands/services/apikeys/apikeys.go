// Package apikeys implements the "qdivzero api-keys" service commands.
package apikeys

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-go"
)

// New returns the api-keys service command tree.
func New(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-keys",
		Short: "Manage API keys",
	}
	cmd.AddCommand(
		newListCmd(deps),
		newCreateCmd(deps),
		newRevokeCmd(deps),
	)
	return cmd
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// apiKeyRow renders one ApiKeyResponse as a table row.
func apiKeyRow(k *qdivzero.ApiKeyResponse) []string {
	return []string{
		str(k.Id),
		str(k.KeyPrefix),
		str(k.CreatedAt),
		str(k.RevokedAt),
	}
}

func newListCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			resp, err := api.GetApiKeysWithResponse(ctx)
			if err != nil {
				return fmt.Errorf("api-keys list: %w", err)
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("api-keys list: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON200 != nil && resp.JSON200.ApiKeys != nil {
				for i := range *resp.JSON200.ApiKeys {
					rows = append(rows, apiKeyRow(&(*resp.JSON200.ApiKeys)[i]))
				}
			}
			return deps.Render.Render(
				[]string{"ID", "PREFIX", "CREATED", "REVOKED"},
				rows,
				resp.JSON200,
			)
		},
	}
}

func newCreateCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create an API key (the plaintext key is shown only once)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			resp, err := api.PostApiKeysWithResponse(ctx)
			if err != nil {
				return fmt.Errorf("api-keys create: %w", err)
			}
			if resp.StatusCode() != 201 {
				return fmt.Errorf("api-keys create: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON201 != nil {
				k := resp.JSON201
				rows = append(rows, []string{
					str(k.Id),
					str(k.KeyPrefix),
					str(k.PlaintextKey),
					str(k.CreatedAt),
				})
			}
			return deps.Render.Render(
				[]string{"ID", "PREFIX", "PLAINTEXT KEY", "CREATED"},
				rows,
				resp.JSON201,
			)
		},
	}
}

func newRevokeCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			id := args[0]
			resp, err := api.PostApiKeysKeyIDRevokeWithResponse(ctx, id)
			if err != nil {
				return fmt.Errorf("api-keys revoke: %w", err)
			}
			if resp.StatusCode() >= 400 {
				return fmt.Errorf("api-keys revoke: status %d", resp.StatusCode())
			}
			return deps.Render.Render(
				[]string{"ID", "STATUS"},
				[][]string{{id, "revoked"}},
				map[string]string{"id": id, "status": "revoked"},
			)
		},
	}
}
