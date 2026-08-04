package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-cli/internal/output"
)

func executeBetaToken(t *testing.T, args []string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	deps := &Deps{
		Render: output.New(func() bool { return false }, &out),
		Stdout: &out,
		Stderr: &out,
	}
	cmd := newConfigureBetaTokenCmd(deps)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	return out.String(), err
}

func TestConfigureBetaTokenWritesAndPreservesCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Write(config.Credentials{AccessToken: "tok"}, false); err != nil {
		t.Fatal(err)
	}
	out, err := executeBetaToken(t, []string{"beta-1"})
	if err != nil {
		t.Fatalf("configure-beta-token: %v", err)
	}
	if !strings.Contains(out, "configured") {
		t.Fatalf("missing confirmation: %q", out)
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PrivateBetaToken != "beta-1" {
		t.Fatalf("PrivateBetaToken = %q, want beta-1", cfg.PrivateBetaToken)
	}
	if cfg.AccessToken != "tok" {
		t.Fatalf("AccessToken was clobbered: %q", cfg.AccessToken)
	}
}

func TestConfigureBetaTokenEmptyClears(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Write(config.Credentials{AccessToken: "tok", PrivateBetaToken: "beta-1"}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := executeBetaToken(t, []string{""}); err != nil {
		t.Fatalf("configure-beta-token: %v", err)
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PrivateBetaToken != "" {
		t.Fatalf("PrivateBetaToken = %q, want cleared", cfg.PrivateBetaToken)
	}
	if cfg.AccessToken != "tok" {
		t.Fatalf("AccessToken was clobbered: %q", cfg.AccessToken)
	}
}
