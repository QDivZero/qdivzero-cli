package instances

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QDivZero/qdivzero-go"

	sdkclient "github.com/QDivZero/qdivzero-cli/internal/client"
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
			opts, err := sdkclient.Options(srv.URL, config.Credentials{AccessToken: "t"})
			if err != nil {
				return nil, err
			}
			return qdivzero.NewAPI(opts...)
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

func TestListInstances(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/instances" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"instances":[{"id":"i-1","name":"gpu-a","state":"running","gpu_count":1}]}`))
	})
	out := runCmd(t, deps, buf, "list")
	if !strings.Contains(out, "i-1") || !strings.Contains(out, "gpu-a") {
		t.Fatalf("missing instance row:\n%s", out)
	}
}

func TestStartInstance(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/instances/i-1/start" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	out := runCmd(t, deps, buf, "start", "i-1")
	if !strings.Contains(out, "i-1") {
		t.Fatalf("missing id in output:\n%s", out)
	}
}

func TestCreateInstanceRequiresModel(t *testing.T) {
	deps, _ := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	})
	cmd := New(deps)
	cmd.SetArgs([]string{"create", "--name", "x"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("create without --huggingface-repo-id should fail")
	}
}

func TestCreateInstancePreservesWorkloadKindAndOtherFields(t *testing.T) {
	deps, buf := testDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/instances" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["workload_kind"] != "decision" || body["name"] != "example" || body["huggingface_repo_id"] != "org/model" {
			t.Errorf("unexpected create request: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"i-1"}`))
	})
	runCmd(t, deps, buf, "create", "--name", "example", "--huggingface-repo-id", "org/model", "--workload-kind", "decision")
}
