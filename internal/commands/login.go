package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with email/password (2FA TOTP supported) and store the tokens",
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
			if password == "" {
				fmt.Fprint(deps.Stdout, "Password: ")
				password = readSecret(reader)
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
					return fmt.Errorf("login: failed — %s (status %d)", loginErrorBody(resp.Body), resp.StatusCode())
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
	return cmd
}

// readSecret reads a line from the reader; when stdin is a terminal the echo
// is disabled so the secret is not displayed.
func readSecret(reader *bufio.Reader) string {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stdout)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(raw))
	}
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// loginErrorBody extracts the API error message from a login failure body.
func loginErrorBody(body []byte) string {
	if r, err := qdivzero.ParseErrorResponse(body); err == nil && r != nil && r.Error != nil {
		return *r.Error
	}
	return "unknown error"
}
