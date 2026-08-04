package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/QDivZero/qdivzero-go"
)

// passkeyBegin holds the begin-login response fields we need. The API returns
// public_key as an object ({publicKey:{challenge, allowCredentials,...}}),
// which the generated SDK types as an array — so the response is parsed from
// the raw body.
type passkeyBegin struct {
	ChallengeID string `json:"challenge_id"`
	PublicKey   struct {
		PublicKey json.RawMessage `json:"publicKey"`
	} `json:"public_key"`
}

// passkeyLogin performs the browser-assisted passkey ceremony: begin-login
// against the API, a local page that runs navigator.credentials.get and posts
// the WebAuthn assertion (credential.toJSON()) back, then finish-login.
func passkeyLogin(ctx context.Context, deps *Deps, api *qdivzero.API, email string) (access, refresh string, err error) {
	beginBody, _ := json.Marshal(map[string]string{"email": email})
	beginResp, err := api.PostAuthPasskeysBeginLoginWithBody(ctx, "application/json", bytes.NewReader(beginBody))
	if err != nil {
		return "", "", fmt.Errorf("passkey begin: %w", err)
	}
	defer beginResp.Body.Close()
	if beginResp.StatusCode != 200 {
		raw, _ := io.ReadAll(beginResp.Body)
		return "", "", fmt.Errorf("passkey begin: status %d: %s", beginResp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var begin passkeyBegin
	if err := json.NewDecoder(beginResp.Body).Decode(&begin); err != nil {
		return "", "", fmt.Errorf("passkey begin: decode: %w", err)
	}
	if begin.ChallengeID == "" || len(begin.PublicKey.PublicKey) == 0 {
		return "", "", fmt.Errorf("passkey begin: missing challenge data")
	}

	credential, err := runCeremony(ctx, deps, begin)
	if err != nil {
		return "", "", err
	}

	finishBody, _ := json.Marshal(map[string]any{
		"email":        email,
		"challenge_id": begin.ChallengeID,
		"credential":   credential,
	})
	finishResp, err := api.PostAuthPasskeysFinishLoginWithBody(ctx, "application/json", bytes.NewReader(finishBody))
	if err != nil {
		return "", "", fmt.Errorf("passkey finish: %w", err)
	}
	defer finishResp.Body.Close()
	if finishResp.StatusCode != 200 {
		raw, _ := io.ReadAll(finishResp.Body)
		return "", "", fmt.Errorf("passkey finish: status %d: %s", finishResp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var tokens struct {
		AccessToken  *string `json:"access_token"`
		RefreshToken *string `json:"refresh_token"`
	}
	if err := json.NewDecoder(finishResp.Body).Decode(&tokens); err != nil {
		return "", "", fmt.Errorf("passkey finish: decode: %w", err)
	}
	if tokens.AccessToken == nil || *tokens.AccessToken == "" {
		return "", "", fmt.Errorf("passkey finish: empty access token")
	}
	refresh = ""
	if tokens.RefreshToken != nil {
		refresh = *tokens.RefreshToken
	}
	return *tokens.AccessToken, refresh, nil
}

// runCeremony starts a local HTTP server with the WebAuthn ceremony page,
// opens the browser, and waits for the credential posted back.
func runCeremony(ctx context.Context, deps *Deps, begin passkeyBegin) (json.RawMessage, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("passkey: local server: %w", err)
	}
	url := fmt.Sprintf("http://%s", ln.Addr())

	credCh := make(chan json.RawMessage, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, ceremonyPage(begin.PublicKey.PublicKey))
	})
	mux.HandleFunc("/result", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Credential json.RawMessage `json:"credential"`
			Error      string          `json:"error"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if payload.Error != "" || len(payload.Credential) == 0 || string(payload.Credential) == "null" {
			msg := payload.Error
			if msg == "" {
				msg = "no credential produced"
			}
			credCh <- json.RawMessage(`{"__error":` + jsonQuote(msg) + `}`)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, "Failed — check the terminal.")
			return
		}
		credCh <- payload.Credential
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "OK — you can close this tab and return to the terminal.")
	})
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(ln) }()
	defer server.Close()

	fmt.Fprintf(deps.Stdout, "Waiting for the passkey ceremony in the browser...\n")
	fmt.Fprintf(deps.Stdout, "If the browser did not open, visit %s\n", url)
	openBrowserFn(url)

	select {
	case cred := <-credCh:
		var errPayload struct {
			Error string `json:"__error"`
		}
		if err := json.Unmarshal(cred, &errPayload); err == nil && errPayload.Error != "" {
			return nil, fmt.Errorf("passkey: ceremony failed in the browser: %s", errPayload.Error)
		}
		return cred, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("passkey: ceremony timed out after 2 minutes")
	}
}

// jsonQuote quotes a string for embedding in JSON.
func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// openBrowserFn is indirections so tests can stub the browser launch.
var openBrowserFn = openBrowser

// openBrowser opens the URL in the default browser.
func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

// ceremonyPage renders the WebAuthn ceremony. It decodes the base64url
// challenge and allowCredentials ids, runs navigator.credentials.get, and
// posts credential.toJSON() back to /result.
func ceremonyPage(publicKey json.RawMessage) string {
	return `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>QDivZero sign in</title></head>
<body>
<p>Authenticate with your passkey…</p>
<script>
const payload = ` + string(publicKey) + `;
function b64urlToBytes(s) {
  const b = s.replace(/-/g, "+").replace(/_/g, "/");
  const pad = b.length % 4 ? 4 - (b.length % 4) : 0;
  const bin = atob(b + "=".repeat(pad));
  const u = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) u[i] = bin.charCodeAt(i);
  return u;
}
function bytesToB64url(buf) {
  const bytes = new Uint8Array(buf);
  let bin = "";
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}
// Serialize the assertion manually (credential.toJSON() is unavailable in
// some browsers).
function serializeCredential(credential) {
  const assertion = credential.response;
  const out = {
    id: credential.id,
    rawId: bytesToB64url(credential.rawId),
    type: credential.type,
    response: {
      authenticatorData: bytesToB64url(assertion.authenticatorData),
      clientDataJSON: bytesToB64url(assertion.clientDataJSON),
      signature: bytesToB64url(assertion.signature),
    },
    clientExtensionResults: credential.getClientExtensionResults ? credential.getClientExtensionResults() : {},
  };
  if (assertion.userHandle && assertion.userHandle.byteLength > 0) {
    out.response.userHandle = bytesToB64url(assertion.userHandle);
  }
  return out;
}
const options = {
  ...payload,
  challenge: b64urlToBytes(payload.challenge),
  allowCredentials: (payload.allowCredentials || []).map((d) => ({ ...d, id: b64urlToBytes(d.id) })),
};
(async () => {
  try {
    const credential = await navigator.credentials.get({ publicKey: options });
    if (!credential) throw new Error("no credential");
    const res = await fetch("/result", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ credential: serializeCredential(credential) }),
    });
    document.body.innerHTML = "<p>" + (res.ok ? "Success — close this tab." : "Failed — check the terminal.") + "</p>";
  } catch (e) {
    document.body.innerHTML = "<p>Error: " + e.message + "</p>";
    await fetch("/result", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ credential: null, error: String(e.message) }) });
  }
})();
</script>
</body>
</html>`
}
