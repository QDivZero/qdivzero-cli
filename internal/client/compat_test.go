package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-go"
)

func TestCurrentAPIPathsAndSessionRefresh(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var refreshes atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		call := refreshes.Add(1)
		cookie, err := r.Cookie("qdiv0_refresh")
		want := "refresh-1"
		if call == 2 {
			want = "refresh-2"
		}
		if err != nil || cookie.Value != want {
			t.Errorf("refresh cookie = %v, error = %v", cookie, err)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("refresh request body = %q, want empty", body)
		}
		access, refresh := "new-access", "refresh-2"
		if call == 2 {
			access, refresh = "newer-access", "refresh-3"
		}
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_access", Value: access, Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_refresh", Value: refresh, Path: "/v1/auth/refresh"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})
	mux.HandleFunc("/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("session token was sent as an Authorization header")
		}
		cookie, err := r.Cookie("qdiv0_access")
		if err != nil || cookie.Value != "new-access" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	opts, err := Options(srv.URL, config.Credentials{})
	if err != nil {
		t.Fatal(err)
	}
	opts = append(opts, qdivzero.WithAccessToken("old-access"), qdivzero.WithRefreshToken("refresh-1"))
	api, err := qdivzero.NewAPI(opts...)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := api.GetAccountsWithResponse(context.Background())
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("accounts response = %v, error = %v", resp, err)
	}
	if refreshes.Load() != 1 {
		t.Fatalf("refresh calls = %d, want 1", refreshes.Load())
	}
	if err := api.Refresh(context.Background()); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if got := api.AccessToken(); got != "newer-access" {
		t.Fatalf("access after second refresh = %q", got)
	}
}

func TestAPIKeyRemainsBearerCredential(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/accounts" || r.Header.Get("Authorization") != "Bearer tk_example" {
			t.Errorf("path = %q, authorization = %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	opts, err := Options(srv.URL, config.Credentials{APIKey: "tk_example"})
	if err != nil {
		t.Fatal(err)
	}
	api, err := qdivzero.NewAPI(opts...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.GetAccountsWithResponse(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialsLoginOnDemandPreservesPrivateBetaHeader(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var logins atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		if r.Header.Get("X-Private-Beta-Token") != "beta" {
			t.Error("login request is missing the private beta header")
		}
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_access", Value: "session", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_refresh", Value: "refresh", Path: "/v1/auth/refresh"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})
	mux.HandleFunc("/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("qdiv0_access")
		if err != nil || cookie.Value != "session" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cfg := config.Credentials{Email: "user@example.com", Password: "secret", PrivateBetaToken: "beta"}
	if err := config.Write(cfg, true); err != nil {
		t.Fatal(err)
	}
	opts, err := options(srv.URL, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	api, err := qdivzero.NewAPI(opts...)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := api.GetAccountsWithResponse(context.Background())
	if err != nil || resp.StatusCode() != http.StatusOK || logins.Load() != 1 {
		t.Fatalf("accounts response = %v, logins = %d, error = %v", resp, logins.Load(), err)
	}
	stored, err := config.Read()
	if err != nil || stored.AccessToken != "session" || stored.RefreshToken != "refresh" {
		t.Fatalf("stored session = %#v, error = %v", stored, err)
	}
}

func TestAuthenticationDoesNotFollowCrossOriginRedirects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var received atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	opts, err := Options(source.URL, config.Credentials{RefreshToken: "refresh-secret"})
	if err != nil {
		t.Fatal(err)
	}
	api, err := qdivzero.NewAPI(opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := api.Refresh(context.Background()); err == nil {
		t.Fatal("expected cross-origin authentication redirect to fail")
	}
	if got := received.Load(); got != 0 {
		t.Fatalf("cross-origin target received %d requests", got)
	}
}

func TestRotatedSessionIsPersistedForTheNextCLIInvocation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := config.Credentials{AccessToken: "old-access", RefreshToken: "old-refresh", AccountId: "account-1"}
	if err := config.Write(cfg, true); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_access", Value: "new-access", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "qdiv0_refresh", Value: "new-refresh", Path: "/v1/auth/refresh"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	opts, err := options(srv.URL, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	api, err := qdivzero.NewAPI(opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := api.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	stored, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}
	if stored.AccessToken != "new-access" || stored.RefreshToken != "new-refresh" || stored.AccountId != "account-1" {
		t.Fatalf("stored credentials = %#v", stored)
	}
}
