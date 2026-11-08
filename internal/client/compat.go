package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-go"
)

// Options adapts the currently published SDK to the versioned, cookie-based API.
// It can be removed after the corrected SDK is published and adopted by the CLI.
func Options(base string, cfg config.Credentials) ([]qdivzero.Option, error) {
	return options(base, cfg, false)
}

func options(base string, cfg config.Credentials, persistSession bool) ([]qdivzero.Option, error) {
	if base == "" {
		base = qdivzero.DefaultBaseURL
	}
	origin, err := url.Parse(base)
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return nil, fmt.Errorf("invalid API base URL %q", base)
	}
	transport := &compatTransport{base: http.DefaultTransport, origin: origin, cfg: cfg, refreshToken: cfg.RefreshToken, persistSession: persistSession}
	opts := []qdivzero.Option{
		qdivzero.WithServerURL(base),
		qdivzero.WithCustomHTTPClient(&http.Client{Transport: transport}),
	}
	if cfg.APIKey != "" {
		opts = append(opts, qdivzero.WithAuthorization("Bearer "+cfg.APIKey))
	} else {
		if cfg.AccessToken != "" {
			opts = append(opts, qdivzero.WithAccessToken(cfg.AccessToken))
		}
		if cfg.RefreshToken != "" {
			opts = append(opts, qdivzero.WithRefreshToken(cfg.RefreshToken))
		}
		if cfg.Email != "" && cfg.Password != "" {
			opts = append(opts, qdivzero.WithCredentials(cfg.Email, cfg.Password))
		}
	}
	return opts, nil
}

type compatTransport struct {
	base           http.RoundTripper
	origin         *url.URL
	cfg            config.Credentials
	mu             sync.Mutex
	refreshToken   string
	persistSession bool
}

func (t *compatTransport) RoundTrip(original *http.Request) (*http.Response, error) {
	req := original.Clone(original.Context())
	if !sameOrigin(req.URL, t.origin) {
		for _, header := range []string{"Authorization", "Cookie", "X-Private-Beta-Token", "X-Account-Id"} {
			req.Header.Del(header)
		}
		return t.base.RoundTrip(req)
	}

	req.URL.Path = versionedPath(req.URL.Path, t.origin.Path)
	if req.URL.RawPath != "" {
		req.URL.RawPath = versionedPath(req.URL.RawPath, t.origin.EscapedPath())
	}
	if t.cfg.PrivateBetaToken != "" && req.Header.Get("X-Private-Beta-Token") == "" {
		req.Header.Set("X-Private-Beta-Token", t.cfg.PrivateBetaToken)
	}
	if t.cfg.AccountId != "" && req.Header.Get("X-Account-Id") == "" {
		req.Header.Set("X-Account-Id", t.cfg.AccountId)
	}
	if bearer := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer "); bearer != "" &&
		req.Header.Get("Authorization") == "Bearer "+bearer && bearer != t.cfg.APIKey &&
		!strings.HasPrefix(bearer, "tk_") && !strings.HasPrefix(bearer, "uk_") {
		req.Header.Del("Authorization")
		setCookie(req, "qdiv0_access", bearer)
	}

	isRefresh := strings.HasSuffix(req.URL.Path, "/auth/refresh")
	isLogin := strings.HasSuffix(req.URL.Path, "/auth/login")
	if isRefresh && req.Body != nil && req.Body != http.NoBody && req.ContentLength != 0 {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("decode refresh request: %w", err)
		}
		_ = req.Body.Close()
		if body.RefreshToken != "" {
			t.mu.Lock()
			refresh := t.refreshToken
			t.mu.Unlock()
			if refresh == "" {
				refresh = body.RefreshToken
			}
			setCookie(req, "qdiv0_refresh", refresh)
		}
		req.Body, req.GetBody, req.ContentLength = http.NoBody, nil, 0
		req.Header.Del("Content-Type")
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if (isLogin || isRefresh) && resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if target, err := resp.Location(); err == nil && !sameOrigin(target, t.origin) {
			resp.Body.Close()
			return nil, fmt.Errorf("authentication redirect to another origin")
		}
	}
	if (isLogin || isRefresh) && resp.StatusCode == http.StatusOK {
		if err := t.adaptSessionResponse(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}
	return resp, nil
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func versionedPath(path, basePath string) string {
	base := strings.TrimSuffix(basePath, "/")
	relative := strings.TrimPrefix(path, base)
	for _, prefix := range []string{"/accounts", "/auth", "/api-keys", "/instances", "/models", "/huggingface", "/serving-endpoints"} {
		if relative == prefix || strings.HasPrefix(relative, prefix+"/") {
			return base + "/v1" + relative
		}
	}
	return path
}

func setCookie(req *http.Request, name, value string) {
	cookies := req.Cookies()
	req.Header.Del("Cookie")
	for _, cookie := range cookies {
		if cookie.Name != name {
			req.AddCookie(cookie)
		}
	}
	req.AddCookie(&http.Cookie{Name: name, Value: value})
}

// Older SDKs read tokens from JSON; provide that local representation while
// retaining the actual Set-Cookie headers for newer SDKs and login commands.
func (t *compatTransport) adaptSessionResponse(resp *http.Response) error {
	var access, refresh string
	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case "qdiv0_access":
			access = cookie.Value
		case "qdiv0_refresh":
			refresh = cookie.Value
		}
	}
	if access == "" || refresh == "" {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read authentication response: %w", err)
	}
	resp.Body.Close()
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil || data == nil {
		return fmt.Errorf("decode authentication response: invalid JSON")
	}
	data["access_token"], data["refresh_token"] = access, refresh
	body, err = json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode authentication response: %w", err)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	previousRefresh := t.refreshToken
	t.refreshToken = refresh
	if t.persistSession {
		current, err := config.Read()
		if err != nil {
			return fmt.Errorf("read credentials for session rotation: %w", err)
		}
		if current.APIKey == "" && ((previousRefresh != "" && current.RefreshToken == previousRefresh) ||
			(previousRefresh == "" && t.cfg.Email != "" && current.Email == t.cfg.Email)) {
			current.AccessToken, current.RefreshToken = access, refresh
			if err := config.Write(current, true); err != nil {
				return fmt.Errorf("save rotated session: %w", err)
			}
		}
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", fmt.Sprint(len(body)))
	return nil
}
