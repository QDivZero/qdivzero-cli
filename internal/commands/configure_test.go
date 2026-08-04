package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-cli/internal/output"
)

func executeConfigure(t *testing.T, args []string, stdin string) (string, error) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	var out bytes.Buffer
	deps := &Deps{
		Render: output.New(func() bool { return false }, &out),
		Stdin:  strings.NewReader(stdin),
		Stdout: &out,
		Stderr: &out,
	}
	cmd := newConfigureCmd(deps)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	err := cmd.Execute()
	return out.String(), err
}

func TestConfigureTokenFlagWritesCredentials(t *testing.T) {
	out, err := executeConfigure(t, []string{"--token", "t-123"}, "")
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if !strings.Contains(out, "configured") {
		t.Fatalf("missing confirmation: %q", out)
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccessToken != "t-123" {
		t.Fatalf("AccessToken = %q, want t-123", cfg.AccessToken)
	}
}

func TestConfigureInteractivePromptsForToken(t *testing.T) {
	out, err := executeConfigure(t, nil, "tok-abc\n")
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if !strings.Contains(out, "Access token") {
		t.Fatalf("expected prompt in output: %q", out)
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccessToken != "tok-abc" {
		t.Fatalf("AccessToken = %q, want tok-abc", cfg.AccessToken)
	}
}

func TestConfigureEmailPasswordFlags(t *testing.T) {
	_, err := executeConfigure(t, []string{"--email", "a@b.c", "--password", "pw"}, "")
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Email != "a@b.c" || cfg.Password != "pw" {
		t.Fatalf("unexpected credentials: %+v", cfg)
	}
}

func TestConfigureTokenAndEmailConflict(t *testing.T) {
	_, err := executeConfigure(t, []string{"--token", "t", "--email", "a@b.c"}, "")
	if err == nil {
		t.Fatal("configure: expected conflict error")
	}
}
