package apikeys

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QDivZero/qdivzero-go"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
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
		Stdout: &buf,
		Stderr: &buf,
	}
	return deps, &buf
}

// runCmd executes the command and returns everything the renderer wrote to
// the deps buffer (the renderer, not cobra's out writer, is where commands
// emit their results).
func runCmd(t *testing.T, deps *commands.Deps, buf *bytes.Buffer, args ...string) string {
	t.Helper()
	cmd := New(deps)
	cmd.SetArgs(args)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return buf.String()
}

func TestListApiKeys(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api-keys" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"api_keys":[{"id":"k-1","key_prefix":"qd_abc","created_at":"2026-01-01"}]}`))
	})
	out := runCmd(t, deps, buf, "list")
	if !strings.Contains(out, "k-1") || !strings.Contains(out, "qd_abc") {
		t.Fatalf("missing api key row:\n%s", out)
	}
}

func TestCreateApiKeyShowsPlaintext(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api-keys" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"k-2","key_prefix":"qd_abc","plaintext_key":"qd_abc.secret"}`))
	})
	out := runCmd(t, deps, buf, "create")
	if !strings.Contains(out, "qd_abc.secret") {
		t.Fatalf("missing plaintext key in output:\n%s", out)
	}
}

func TestRevokeApiKey(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api-keys/k-3/revoke" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	out := runCmd(t, deps, buf, "revoke", "k-3")
	if !strings.Contains(out, "k-3") || !strings.Contains(out, "revoked") {
		t.Fatalf("missing revoke result:\n%s", out)
	}
}
