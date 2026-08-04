// Package models implements the "qdivzero models" service commands.
package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-go"
)

// New returns the models service command tree.
func New(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List models",
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

func itoa(i *int) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%d", *i)
}

// modelRow renders one HuggingFaceModelResponse as a table row.
func modelRow(m *qdivzero.HuggingFaceModelResponse) []string {
	return []string{
		str(m.RepoId),
		str(m.PipelineTag),
		itoa(m.Likes),
		itoa(m.Downloads),
	}
}

func newListCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List fixed models",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			resp, err := api.GetModelsFixedWithResponse(ctx)
			if err != nil {
				return fmt.Errorf("models list: %w", err)
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("models list: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON200 != nil && resp.JSON200.Models != nil {
				for i := range *resp.JSON200.Models {
					rows = append(rows, modelRow(&(*resp.JSON200.Models)[i]))
				}
			}
			return deps.Render.Render(
				[]string{"REPO", "PIPELINE", "LIKES", "DOWNLOADS"},
				rows,
				resp.JSON200,
			)
		},
	}
}
