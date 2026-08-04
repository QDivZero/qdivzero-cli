package commands

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-go"
)

// loginClient builds an unauthenticated API client (the login endpoint does
// not require credentials, but the private beta header is applied when set).
func loginClient(deps *Deps) (*qdivzero.API, error) {
	base := "https://api.qdiv0.com"
	if deps.BaseURL != "" {
		base = deps.BaseURL
	}
	var opts []qdivzero.Option
	if cfg, err := config.Read(); err == nil && cfg.PrivateBetaToken != "" {
		opts = append(opts, qdivzero.WithHeader("X-Private-Beta-Token", cfg.PrivateBetaToken))
	}
	opts = append(opts, qdivzero.WithServerURL(base))
	return qdivzero.NewAPI(opts...)
}

func newLoginCmd(deps *Deps) *cobra.Command {
	var (
		email    string
		password string
		totpCode string
		passkey  bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in (email/password with 2FA, or passkey) and store the tokens",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := loginClient(deps)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			reader := bufio.NewReader(deps.Stdin)
			if email == "" {
				fmt.Fprint(deps.Stdout, "Email: ")
				line, _ := reader.ReadString('\n')
				email = strings.TrimSpace(line)
			}
			if email == "" {
				return fmt.Errorf("login: email is required")
			}
			if passkey {
				access, refresh, err := passkeyLogin(ctx, deps, api, email)
				if err != nil {
					return err
				}
				return storeLoginTokens(deps, email, "", access, refresh)
			}
			if password == "" {
				fmt.Fprint(deps.Stdout, "Password: ")
				line, _ := reader.ReadString('\n')
				password = strings.TrimSpace(line)
			}
			if password == "" {
				return fmt.Errorf("login: email and password are required")
			}

			// First attempt: email + password (may return a 2FA next_step).
			body := qdivzero.LoginRequest{Email: email, Password: password}
			if totpCode != "" {
				body.TotpCode = &totpCode
			}
			resp, err := api.PostAuthLoginWithResponse(ctx, body)
			if err != nil {
				return fmt.Errorf("login: %w", err)
			}
			if resp.JSON200 == nil || resp.JSON200.AccessToken == nil || *resp.JSON200.AccessToken == "" {
				next := ""
				if resp.JSON200 != nil && resp.JSON200.NextStep != nil {
					next = *resp.JSON200.NextStep
				}
				if totpCode == "" && (next == "totp" || next != "") {
					fmt.Fprint(deps.Stdout, "2FA required — TOTP code: ")
					line, _ := reader.ReadString('\n')
					code := strings.TrimSpace(line)
					body.TotpCode = &code
					resp, err = api.PostAuthLoginWithResponse(ctx, body)
					if err != nil {
						return fmt.Errorf("login: %w", err)
					}
				}
				if resp.JSON200 == nil || resp.JSON200.AccessToken == nil || *resp.JSON200.AccessToken == "" {
					return fmt.Errorf("login: failed — check your credentials and 2FA code (status %d)", resp.StatusCode())
				}
			}

			creds, err := config.Read()
			if err != nil {
				return err
			}
			creds.Email = email
			creds.Password = password
			creds.AccessToken = *resp.JSON200.AccessToken
			if resp.JSON200.RefreshToken != nil {
				creds.RefreshToken = *resp.JSON200.RefreshToken
			}
			if err := config.Write(creds, true); err != nil {
				return err
			}
			fmt.Fprintln(deps.Stdout, "logged in: tokens stored in ~/.qdivzero/credentials")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "account email (non-interactive)")
	cmd.Flags().StringVar(&password, "password", "", "account password (non-interactive)")
	cmd.Flags().StringVar(&totpCode, "totp-code", "", "2FA TOTP code (non-interactive)")
	cmd.Flags().BoolVar(&passkey, "passkey", false, "log in with a passkey (opens the browser for the ceremony)")
	return cmd
}

// storeLoginTokens persists the tokens from a successful login, preserving
// the beta token and active account.
func storeLoginTokens(deps *Deps, email, password, access, refresh string) error {
	creds, err := config.Read()
	if err != nil {
		return err
	}
	creds.Email = email
	creds.Password = password
	creds.AccessToken = access
	if refresh != "" {
		creds.RefreshToken = refresh
	}
	if err := config.Write(creds, true); err != nil {
		return err
	}
	fmt.Fprintln(deps.Stdout, "logged in: tokens stored in ~/.qdivzero/credentials")
	return nil
}
