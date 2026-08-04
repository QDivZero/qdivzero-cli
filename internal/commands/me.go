package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMeCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			resp, err := api.GetAuthMeWithResponse(ctx)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("me: status %d", resp.StatusCode())
			}
			user := resp.JSON200
			rows := [][]string{}
			if user != nil {
				rows = append(rows, []string{str(user.UserId), str(user.Email)})
			}
			return deps.Render.Render([]string{"ID", "EMAIL"}, rows, user)
		},
	}
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
