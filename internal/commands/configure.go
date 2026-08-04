package commands

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/config"
)

func newConfigureCmd(deps *Deps) *cobra.Command {
	var (
		token    string
		email    string
		password string
		apiKey   string
		force    bool
	)
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure authentication (writes ~/.qdivzero/credentials)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			provided := 0
			if token != "" {
				provided++
			}
			if email != "" || password != "" {
				provided++
			}
			if apiKey != "" {
				provided++
			}
			if provided > 1 {
				return fmt.Errorf("configure: use only one of --token, --email/--password or --api-key")
			}
			// Preserve the beta token and active account across reconfiguration.
			existing, err := config.Read()
			if err != nil {
				return err
			}
			creds := config.Credentials{
				PrivateBetaToken: existing.PrivateBetaToken,
				AccountId:        existing.AccountId,
			}
			if apiKey != "" {
				// API-key mode: clear the token/credentials so the API key is
				// always used (the SDK would otherwise prefer the access token).
				creds.APIKey = apiKey
				force = true
			} else if token != "" {
				creds.AccessToken = token
			} else if email != "" || password != "" {
				creds.Email = email
				creds.Password = password
			} else {
				prompted, err := promptCredentials(deps)
				if err != nil {
					return err
				}
				prompted.PrivateBetaToken = existing.PrivateBetaToken
				prompted.AccountId = existing.AccountId
				creds = prompted
			}
			if err := config.Write(creds, force); err != nil {
				return err
			}
			fmt.Fprintln(deps.Stdout, "configured: ~/.qdivzero/credentials")
			return nil
		},
	}
	cmd.Flags().StringVar(&token, "token", "", "access token (non-interactive)")
	cmd.Flags().StringVar(&email, "email", "", "account email (non-interactive)")
	cmd.Flags().StringVar(&password, "password", "", "account password (non-interactive)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "QDivZero API key (non-interactive; switches to API-key auth)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing credentials file")
	return cmd
}

// promptCredentials interactively asks for an access token, or email and
// password when the user selects that option. A bare answer that is not a
// menu choice is taken as the access token directly.
func promptCredentials(deps *Deps) (config.Credentials, error) {
	reader := bufio.NewReader(deps.Stdin)
	fmt.Fprintln(deps.Stdout, "QDivZero configuration")
	fmt.Fprintln(deps.Stdout, "  [1] Access token")
	fmt.Fprintln(deps.Stdout, "  [2] Email + password")
	fmt.Fprint(deps.Stdout, "Choose (1/2) [1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "2" {
		fmt.Fprint(deps.Stdout, "Email: ")
		email, _ := reader.ReadString('\n')
		fmt.Fprint(deps.Stdout, "Password: ")
		password, _ := reader.ReadString('\n')
		return config.Credentials{
			Email:    strings.TrimSpace(email),
			Password: strings.TrimSpace(password),
		}, nil
	}
	fmt.Fprint(deps.Stdout, "Access token: ")
	token := choice
	if choice == "" || choice == "1" {
		token, _ = reader.ReadString('\n')
		token = strings.TrimSpace(token)
	}
	return config.Credentials{AccessToken: token}, nil
}
