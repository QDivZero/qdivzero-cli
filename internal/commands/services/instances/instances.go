// Package instances implements the "qdivzero instances" service commands.
package instances

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-go"
)

// New returns the instances service command tree.
func New(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instances",
		Short: "Manage GPU instances",
	}
	cmd.AddCommand(
		newListCmd(deps),
		newShowCmd(deps),
		newStartCmd(deps),
		newStopCmd(deps),
		newDeleteCmd(deps),
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

func itoa(i *int) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%d", *i)
}

func boolStr(b *bool) string {
	if b == nil {
		return ""
	}
	return fmt.Sprintf("%t", *b)
}

func strPtr(s string) *string { return &s }

func intPtr(i int) *int { return &i }

func newListCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List instances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			resp, err := api.GetInstancesWithResponse(ctx, nil)
			if err != nil {
				return fmt.Errorf("instances list: %w", err)
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("instances list: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON200 != nil && resp.JSON200.Instances != nil {
				for _, i := range *resp.JSON200.Instances {
					rows = append(rows, instanceRow(&i))
				}
			}
			return deps.Render.Render(
				[]string{"ID", "NAME", "STATE", "GPU", "TIER"},
				rows,
				resp.JSON200,
			)
		},
	}
}

// instanceRow renders one InstanceResponse as a table row.
func instanceRow(i *qdivzero.InstanceResponse) []string {
	return []string{
		str(i.Id),
		str(i.Name),
		str(i.State),
		itoa(i.GpuCount),
		str(i.CapacityTier),
	}
}

func newShowCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			id := args[0]
			resp, err := api.GetInstanceByIDWithResponse(ctx, id)
			if err != nil {
				return fmt.Errorf("instances show: %w", err)
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("instances show: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON200 != nil {
				i := resp.JSON200
				rows = append(rows,
					[]string{"ID", str(i.Id)},
					[]string{"NAME", str(i.Name)},
					[]string{"STATE", str(i.State)},
					[]string{"GPU", itoa(i.GpuCount)},
					[]string{"TIER", str(i.CapacityTier)},
					[]string{"SERVERLESS", boolStr(i.ServerlessEnabled)},
					[]string{"CREATED", str(i.CreatedAt)},
					[]string{"DESCRIPTION", str(i.Description)},
				)
			}
			return deps.Render.Render([]string{"KEY", "VALUE"}, rows, resp.JSON200)
		},
	}
}

func newStartCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "start <id>",
		Short: "Start an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			id := args[0]
			resp, err := api.PostInstanceStartWithResponse(ctx, id)
			if err != nil {
				return fmt.Errorf("instances start: %w", err)
			}
			if resp.StatusCode() >= 400 {
				return fmt.Errorf("instances start: status %d", resp.StatusCode())
			}
			return deps.Render.Render(
				[]string{"ID", "STATUS"},
				[][]string{{id, "started"}},
				map[string]string{"id": id, "status": "started"},
			)
		},
	}
}

func newStopCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			id := args[0]
			resp, err := api.PostInstanceStopWithResponse(ctx, id)
			if err != nil {
				return fmt.Errorf("instances stop: %w", err)
			}
			if resp.StatusCode() >= 400 {
				return fmt.Errorf("instances stop: status %d", resp.StatusCode())
			}
			return deps.Render.Render(
				[]string{"ID", "STATUS"},
				[][]string{{id, "stopped"}},
				map[string]string{"id": id, "status": "stopped"},
			)
		},
	}
}

func newDeleteCmd(deps *commands.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			id := args[0]
			resp, err := api.DeleteInstanceWithResponse(ctx, id)
			if err != nil {
				return fmt.Errorf("instances delete: %w", err)
			}
			if resp.StatusCode() >= 400 {
				return fmt.Errorf("instances delete: status %d", resp.StatusCode())
			}
			return deps.Render.Render(
				[]string{"ID", "STATUS"},
				[][]string{{id, "deleted"}},
				map[string]string{"id": id, "status": "deleted"},
			)
		},
	}
}

func newCreateCmd(deps *commands.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an instance",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			repoID, _ := cmd.Flags().GetString("huggingface-repo-id")
			if repoID == "" {
				return fmt.Errorf("instances create: --huggingface-repo-id is required")
			}
			name, _ := cmd.Flags().GetString("name")
			serverless, _ := cmd.Flags().GetBool("serverless")
			runtimePreset, _ := cmd.Flags().GetString("runtime-preset")
			schedulingMode, _ := cmd.Flags().GetString("scheduling-mode")
			gpuPrefsRaw, _ := cmd.Flags().GetString("gpu-preferences")
			idleTimeout, _ := cmd.Flags().GetInt("idle-timeout-seconds")
			description, _ := cmd.Flags().GetString("description")

			body := qdivzero.CreateInstanceRequest{
				HuggingfaceRepoId: strPtr(repoID),
			}
			if cmd.Flags().Changed("name") {
				body.Name = strPtr(name)
			}
			if cmd.Flags().Changed("serverless") {
				body.ServerlessEnabled = &serverless
			}
			if cmd.Flags().Changed("runtime-preset") {
				body.RuntimePreset = strPtr(runtimePreset)
			}
			if cmd.Flags().Changed("scheduling-mode") {
				body.SchedulingMode = strPtr(schedulingMode)
			}
			if cmd.Flags().Changed("gpu-preferences") && gpuPrefsRaw != "" {
				var gpuPrefs []qdivzero.PublicGPUPreferenceJSON
				if err := json.Unmarshal([]byte(gpuPrefsRaw), &gpuPrefs); err != nil {
					return fmt.Errorf("instances create: --gpu-preferences: %w", err)
				}
				body.GpuPreferences = &gpuPrefs
			}
			if cmd.Flags().Changed("idle-timeout-seconds") {
				body.IdleTimeoutSeconds = intPtr(idleTimeout)
			}
			if cmd.Flags().Changed("description") {
				body.Description = strPtr(description)
			}

			resp, err := api.PostInstancesWithResponse(ctx, nil, body)
			if err != nil {
				return fmt.Errorf("instances create: %w", err)
			}
			if resp.StatusCode() != 201 {
				return fmt.Errorf("instances create: status %d", resp.StatusCode())
			}
			var rows [][]string
			if resp.JSON201 != nil {
				rows = append(rows, []string{str(resp.JSON201.Id)})
			}
			return deps.Render.Render([]string{"ID"}, rows, resp.JSON201)
		},
	}
	cmd.Flags().String("name", "", "instance name")
	cmd.Flags().String("huggingface-repo-id", "", "Hugging Face repo id (required)")
	cmd.Flags().Bool("serverless", false, "enable serverless mode")
	cmd.Flags().String("runtime-preset", "", "runtime preset")
	cmd.Flags().String("scheduling-mode", "", "scheduling mode")
	cmd.Flags().String("gpu-preferences", "", `GPU preferences as JSON, e.g. [{"public_gpu_id":"gpu-id","count":1}]`)
	cmd.Flags().Int("idle-timeout-seconds", 0, "idle timeout in seconds")
	cmd.Flags().String("description", "", "description")
	return cmd
}
