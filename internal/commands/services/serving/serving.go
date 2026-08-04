// Package serving implements the "qdivzero serving-endpoints" service commands.
package serving

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-go"
)

// New returns the serving-endpoints service command tree.
func New(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serving-endpoints",
		Short: "Manage serving endpoints",
	}
	cmd.AddCommand(
		newListCmd(deps),
		newCreateCmd(deps),
	)
	return cmd
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func boolStr(b *bool) string {
	if b == nil {
		return ""
	}
	return fmt.Sprintf("%t", *b)
}

func strPtr(s string) *string { return &s }

// endpointRow renders one ServingEndpointResponse as a table row.
func endpointRow(e *qdivzero.ServingEndpointResponse) []string {
	return []string{
		str(e.Name),
		str(e.State),
		boolStr(e.Enabled),
		str(e.EndpointType),
		str(e.Id),
	}
}

func newListCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List serving endpoints",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			resp, err := api.GetServingEndpointsWithResponse(ctx)
			if err != nil {
				return fmt.Errorf("serving-endpoints list: %w", err)
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("serving-endpoints list: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON200 != nil {
				keys := make([]string, 0, len(*resp.JSON200))
				for k := range *resp.JSON200 {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					for i := range (*resp.JSON200)[k] {
						rows = append(rows, endpointRow(&(*resp.JSON200)[k][i]))
					}
				}
			}
			return deps.Render.Render(
				[]string{"NAME", "STATE", "ENABLED", "TYPE", "ID"},
				rows,
				resp.JSON200,
			)
		},
	}
}

func newCreateCmd(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a serving endpoint",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("serving-endpoints create: --name is required")
			}
			body := qdivzero.CreateServingEndpointRequest{
				Name: strPtr(name),
			}
			if cmd.Flags().Changed("display-name") {
				displayName, _ := cmd.Flags().GetString("display-name")
				body.DisplayName = strPtr(displayName)
			}
			if cmd.Flags().Changed("workload-kind") {
				workloadKind, _ := cmd.Flags().GetString("workload-kind")
				body.WorkloadKind = strPtr(workloadKind)
			}
			if cmd.Flags().Changed("instance-id") ||
				cmd.Flags().Changed("target-ref-id") ||
				cmd.Flags().Changed("target-type") {
				target := &qdivzero.ServingTargetRequest{}
				body.Target = target
				if cmd.Flags().Changed("instance-id") {
					v, _ := cmd.Flags().GetString("instance-id")
					target.InstanceId = strPtr(v)
				}
				if cmd.Flags().Changed("target-ref-id") {
					v, _ := cmd.Flags().GetString("target-ref-id")
					target.TargetRefId = strPtr(v)
				}
				if cmd.Flags().Changed("target-type") {
					v, _ := cmd.Flags().GetString("target-type")
					target.Type = strPtr(v)
				}
			}
			resp, err := api.PostServingEndpointWithResponse(ctx, body)
			if err != nil {
				return fmt.Errorf("serving-endpoints create: %w", err)
			}
			if resp.StatusCode() != 201 {
				return fmt.Errorf("serving-endpoints create: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON201 != nil {
				rows = append(rows, []string{str(resp.JSON201.Id)})
			}
			return deps.Render.Render([]string{"ID"}, rows, resp.JSON201)
		},
	}
	cmd.Flags().String("name", "", "endpoint name (required)")
	cmd.Flags().String("display-name", "", "display name")
	cmd.Flags().String("workload-kind", "", "workload kind")
	cmd.Flags().String("instance-id", "", "target instance id")
	cmd.Flags().String("target-ref-id", "", "target reference id")
	cmd.Flags().String("target-type", "", "target type")
	return cmd
}
