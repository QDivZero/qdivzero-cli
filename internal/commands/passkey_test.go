package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QDivZero/qdivzero-go"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-cli/internal/output"
)

func TestLoginWithPasskey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	openBrowserFn = func(string) {}
	t.Cleanup(func() { openBrowserFn = openBrowser })

	var finishBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/passkeys/begin-login":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"challenge_id":"c1","public_key":{"publicKey":{"challenge":"AQID","rpId":"qdiv0.com"}}}`)
		case "/auth/passkeys/finish-login":
			raw, _ := io.ReadAll(r.Body)
			json.Unmarshal(raw, &finishBody)
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"access_token":"tok-1","refresh_token":"r-1"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	var out bytes.Buffer
	deps := &Deps{
		BaseURL: srv.URL,
		Client: func(ctx context.Context) (*qdivzero.API, error) {
			return qdivzero.NewAPI(qdivzero.WithServerURL(srv.URL))
		},
		Render: output.New(func() bool { return false }, &out),
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Stderr: &out,
	}
	cmd := newLoginCmd(deps)
	cmd.SetArgs([]string{"--passkey", "--email", "a@b.c"})
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	done := make(chan error, 1)
	go func() { done <- cmd.Execute() }()

	// Find the local ceremony URL in the output and post the credential.
	var url string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if u, ok := extractCeremonyURL(out.String()); ok {
			url = u
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if url == "" {
		t.Fatalf("ceremony URL not found in output:\n%s", out.String())
	}
	resp, err := http.Post(url+"/result", "application/json",
		strings.NewReader(`{"credential":{"id":"cred-1","response":{"signature":"sig"}}}`))
	if err != nil {
		t.Fatalf("post credential: %v", err)
	}
	resp.Body.Close()

	if err := <-done; err != nil {
		t.Fatalf("login --passkey: %v", err)
	}
	if finishBody["challenge_id"] != "c1" {
		t.Fatalf("finish body challenge_id = %v, want c1", finishBody["challenge_id"])
	}
	cred, _ := json.Marshal(finishBody["credential"])
	if !strings.Contains(string(cred), "cred-1") {
		t.Fatalf("finish body credential = %s, want cred-1", cred)
	}
	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccessToken != "tok-1" || cfg.RefreshToken != "r-1" {
		t.Fatalf("stored tokens = %+v", cfg)
	}
}

func extractCeremonyURL(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, "http://127.0.0.1:")
		if idx >= 0 {
			url := strings.TrimSpace(line[idx:])
			return url, true
		}
	}
	return "", false
}
