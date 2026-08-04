package accounts

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QDivZero/qdivzero-go"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-cli/internal/output"
)

func testDeps(t *testing.T, handler http.HandlerFunc) (*commands.Deps, *bytes.Buffer) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	var buf bytes.Buffer
	deps := &commands.Deps{
		Client: func(ctx context.Context) (*qdivzero.API, error) {
			return qdivzero.NewAPI(qdivzero.WithServerURL(srv.URL), qdivzero.WithAccessToken("t"))
		},
		Render: output.New(func() bool { return false }, &buf),
		Config: &config.Credentials{},
		Stdin:  strings.NewReader(""),
		Stdout: &buf,
		Stderr: &buf,
	}
	return deps, &buf
}

func TestUseWithArgStoresAccount(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	deps, _ := testDeps(t, func(w http.ResponseWriter, r *http.Request) {})
	cmd := New(deps)
	cmd.SetArgs([]string{"use", "acc-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("accounts use: %v", err)
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccountId != "acc-1" {
		t.Fatalf("AccountId = %q, want acc-1", cfg.AccountId)
	}
}

func TestUseInteractiveSelectsFromMemberships(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"memberships":[{"account_id":"acc-1","role":"owner"},{"account_id":"acc-2","role":"member"}]}`))
	})
	deps.Stdin = strings.NewReader("2\n")
	cmd := New(deps)
	cmd.SetArgs([]string{"use"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("accounts use: %v", err)
	}
	if !strings.Contains(buf.String(), "acc-2") {
		t.Fatalf("expected selected account in output:\n%s", buf.String())
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccountId != "acc-2" {
		t.Fatalf("AccountId = %q, want acc-2", cfg.AccountId)
	}
}

func TestUseInteractiveInvalidSelection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	deps, _ := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"memberships":[{"account_id":"acc-1"}]}`))
	})
	deps.Stdin = strings.NewReader("9\n")
	cmd := New(deps)
	cmd.SetArgs([]string{"use"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("accounts use: expected error for invalid selection")
	}
}
