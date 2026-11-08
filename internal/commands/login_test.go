package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QDivZero/qdivzero-cli/internal/config"
)

func TestLoginStoresSessionCookiesAndSupportsTOTP(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/login" {
			t.Errorf("login path = %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls++
		var body struct {
			TOTPCode string `json:"totp_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = io.WriteString(w, `{"next_step":"totp_required"}`)
			return
		}
		if body.TOTPCode != "123456" {
			t.Errorf("TOTP code = %q", body.TOTPCode)
		}
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_access", Value: "access-1", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_refresh", Value: "refresh-1", Path: "/v1/auth/refresh"})
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	var output bytes.Buffer
	deps := &Deps{BaseURL: srv.URL, Stdin: strings.NewReader("123456\n"), Stdout: &output, Stderr: &output}
	cmd := newLoginCmd(deps)
	cmd.SetArgs([]string{"--email", "user@example.com", "--password", "secret"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("login: %v", err)
	}
	got, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access-1" || got.RefreshToken != "refresh-1" || calls != 2 {
		t.Fatalf("access = %q, refresh = %q, calls = %d", got.AccessToken, got.RefreshToken, calls)
	}
}

func TestLoginReportsEmailVerificationWithoutPromptingForTOTP(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"next_step":"email_token"}`)
	}))
	defer srv.Close()
	var output bytes.Buffer
	deps := &Deps{BaseURL: srv.URL, Stdin: strings.NewReader(""), Stdout: &output, Stderr: &output}
	cmd := newLoginCmd(deps)
	cmd.SetArgs([]string{"--email", "user@example.com", "--password", "secret"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "email_token") {
		t.Fatalf("login error = %v, want email verification step", err)
	}
	if strings.Contains(output.String(), "TOTP") {
		t.Fatalf("unexpected TOTP prompt: %q", output.String())
	}
}
