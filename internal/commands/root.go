package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/client"
	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-cli/internal/output"
)

// Execute runs the root command with the given service constructors.
func Execute(version string, services ...func(*Deps) *cobra.Command) error {
	root := NewRootCmd(version, services...)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if hint := hintForError(err); hint != "" {
			fmt.Fprintln(os.Stderr, "hint:", hint)
		}
		return err
	}
	return nil
}

// hintForError returns a helpful hint for known error patterns, or "".
func hintForError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "status 401"):
		return "the stored token was rejected; get a fresh one and run 'qdivzero configure --token <token>' (add --force to overwrite)"
	case strings.Contains(msg, "not configured"):
		return "run 'qdivzero configure' first"
	}
	return ""
}

// NewRootCmd builds the CLI root command and wires the service registry.
func NewRootCmd(version string, services ...func(*Deps) *cobra.Command) *cobra.Command {
	jsonMode := false

	root := &cobra.Command{
		Use:           "qdivzero",
		Short:         "QDivZero API CLI",
		Long:          "Command-line client for the QDivZero API (https://api.qdiv0.com).",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.Version = version
	root.SetVersionTemplate("qdivzero {{.Version}}\n")
	root.PersistentFlags().BoolVar(&jsonMode, "json", false, "output as JSON")

	deps := &Deps{
		Version: version,
		Client:  client.Factory(),
		Render:  output.New(func() bool { return jsonMode }, os.Stdout),
		Config:  &config.Credentials{},
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}

	root.AddCommand(
		newConfigureCmd(deps),
		newConfigureBetaTokenCmd(deps),
		newLoginCmd(deps),
		newMeCmd(deps),
		newVersionCmd(deps),
	)
	for _, s := range services {
		root.AddCommand(s(deps))
	}
	return root
}
