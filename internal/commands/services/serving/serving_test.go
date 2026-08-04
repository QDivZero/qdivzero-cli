package serving

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

func TestListServingEndpoints(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/serving-endpoints" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"endpoints": [
				{"id":"se-1","name":"llama","state":"running","enabled":true,"endpoint_type":"static"}
			]
		}`))
	})
	out := runCmd(t, deps, buf, "list")
	if !strings.Contains(out, "se-1") || !strings.Contains(out, "llama") {
		t.Fatalf("missing serving endpoint row:\n%s", out)
	}
}

func TestCreateServingEndpoint(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/serving-endpoints" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"se-2","name":"new-ep"}`))
	})
	out := runCmd(t, deps, buf, "create", "--name", "new-ep", "--workload-kind", "inference")
	if !strings.Contains(out, "se-2") {
		t.Fatalf("missing created endpoint id:\n%s", out)
	}
}

func TestCreateServingEndpointRequiresName(t *testing.T) {
	deps, _ := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	})
	cmd := New(deps)
	cmd.SetArgs([]string{"create"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("create without --name should fail")
	}
}
