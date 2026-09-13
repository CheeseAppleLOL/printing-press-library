// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-written: cli-printing-press has no generator template for OAuth 2.1
// Authorization Code + PKCE with Dynamic Client Registration (confirmed via
// direct research against the generator's spec-format and phase files — see
// the research brief). Implemented per
// references/oauth2-pkce-cli-checklist.md's required behaviors: random state
// and PKCE verifier per attempt, S256 challenge, state validation before code
// exchange, loopback-only callback listener, separate timeout contexts for
// the browser wait vs the token exchange, and no silent dual-suppression of
// both the browser-open and machine-output paths.

package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/food-and-dining/swiggy/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/food-and-dining/swiggy/internal/config"
	"github.com/spf13/cobra"
)

const (
	swiggyAuthBaseURL      = "https://mcp.swiggy.com"
	swiggyAuthorizeURL     = swiggyAuthBaseURL + "/auth/authorize"
	swiggyTokenURL         = swiggyAuthBaseURL + "/auth/token"
	swiggyRegisterURL      = swiggyAuthBaseURL + "/auth/register"
	swiggyOAuthScope       = "mcp:tools mcp:resources mcp:prompts"
	swiggyLoopbackHost     = "127.0.0.1"
	swiggyCallbackWait     = 3 * time.Minute
	swiggyTokenExchangeTO  = 15 * time.Second
	swiggyDCRClientNameTag = "swiggy-pp-cli"
)

func newAuthLoginCmd(flags *rootFlags) *cobra.Command {
	var noOpen bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Complete the OAuth 2.1 PKCE browser login (phone + OTP) and store the access token",
		Long: "Swiggy Builders Club uses OAuth 2.1 Authorization Code + PKCE with Dynamic Client\n" +
			"Registration — there is no API key to paste. This opens your browser to Swiggy's own\n" +
			"consent screen (phone number + OTP), then stores the resulting access token.\n" +
			"Access tokens last 5 days; there is no refresh-token issuance in v1.0, so re-run this\n" +
			"command when 'swiggy-pp-cli status' or a 401 says you need to.",
		Example: "  swiggy-pp-cli auth login\n  swiggy-pp-cli auth login --no-open",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			// Machine-output + no-browser-launch would otherwise wait forever
			// for a callback that can never arrive. Fail fast with an
			// actionable usage error per the checklist rather than hanging.
			if flags.asJSON && noOpen {
				return usageErr(fmt.Errorf("auth login: --json and --no-open cannot be combined; this command needs either a browser launch or human-readable stdout to show the authorize URL"))
			}

			if cliutil.IsVerifyEnv() {
				// Never open a browser or dial the real OAuth server under
				// the verify harness.
				if flags.asJSON {
					return printJSONFiltered(w, map[string]any{"verify_mode": true, "would_open_browser": !noOpen}, flags)
				}
				fmt.Fprintln(w, "auth login: verify mode — skipping real browser launch and network calls.")
				return nil
			}

			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}

			clientID, err := swiggyRegisterClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("dynamic client registration failed: %w", err)
			}

			verifier, challenge, err := generatePKCEPair()
			if err != nil {
				return fmt.Errorf("generating PKCE pair: %w", err)
			}
			state, err := randomURLSafeString(24)
			if err != nil {
				return fmt.Errorf("generating state: %w", err)
			}

			listener, err := net.Listen("tcp", swiggyLoopbackHost+":0")
			if err != nil {
				return fmt.Errorf("starting loopback callback listener: %w", err)
			}
			redirectURI := fmt.Sprintf("http://%s/callback", listener.Addr().String())

			authorizeURL := buildAuthorizeURL(clientID, redirectURI, state, challenge)

			codeCh := make(chan string, 1)
			errCh := make(chan error, 1)
			server := &http.Server{
				Handler:           callbackHandler(state, codeCh, errCh),
				ReadHeaderTimeout: 5 * time.Second,
			}
			go func() { _ = server.Serve(listener) }()
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdownCtx)
			}()

			if flags.asJSON {
				if printErr := printJSONFiltered(w, map[string]any{
					"authorize_url": authorizeURL,
					"redirect_uri":  redirectURI,
					"state_id":      state,
					"expires_in":    swiggyCallbackWait.String(),
				}, flags); printErr != nil {
					return printErr
				}
			} else {
				fmt.Fprintln(w, "Opening your browser to sign in to Swiggy (phone number + OTP)...")
				fmt.Fprintln(w, "If it doesn't open automatically, visit:")
				fmt.Fprintln(w, "  "+authorizeURL)
			}
			if !noOpen {
				_ = openBrowser(authorizeURL)
			}

			// Browser-wait context: long enough for a human to complete
			// phone + OTP. Deliberately NOT reused for the token exchange
			// below (see checklist: a user who authorizes near the deadline
			// should not get an ambiguous token-exchange failure from
			// leftover callback time).
			waitCtx, waitCancel := context.WithTimeout(cmd.Context(), swiggyCallbackWait)
			defer waitCancel()

			var code string
			select {
			case code = <-codeCh:
			case err := <-errCh:
				return fmt.Errorf("oauth callback: %w", err)
			case <-waitCtx.Done():
				return fmt.Errorf("timed out after %s waiting for the browser login to complete", swiggyCallbackWait)
			}

			exchangeCtx, exchangeCancel := context.WithTimeout(cmd.Context(), swiggyTokenExchangeTO)
			defer exchangeCancel()
			accessToken, expiresIn, err := exchangeCodeForToken(exchangeCtx, clientID, code, verifier, redirectURI)
			if err != nil {
				return fmt.Errorf("exchanging authorization code: %w", err)
			}

			expiry := time.Now().Add(time.Duration(expiresIn) * time.Second)
			if err := cfg.SaveTokens(clientID, "", accessToken, "", expiry); err != nil {
				return configErr(fmt.Errorf("saving tokens: %w", err))
			}

			if flags.asJSON {
				return printJSONFiltered(w, map[string]any{"authenticated": true, "expires_at": expiry.UTC().Format(time.RFC3339)}, flags)
			}
			fmt.Fprintln(w, green("Logged in."))
			fmt.Fprintf(w, "  Token expires: %s\n", expiry.UTC().Format(time.RFC3339))
			return nil
		},
	}
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Print the authorize URL instead of opening a browser automatically")
	return cmd
}

// generatePKCEPair returns a fresh (verifier, S256 challenge) pair. A new
// pair is generated per login attempt, never reused, per the checklist.
func generatePKCEPair() (verifier, challenge string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomURLSafeString(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func buildAuthorizeURL(clientID, redirectURI, state, challenge string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("scope", swiggyOAuthScope)
	return swiggyAuthorizeURL + "?" + q.Encode()
}

// callbackHandler validates state before ever handing the code back to the
// caller, per the checklist. It only ever serves loopback traffic (the
// listener itself is bound to 127.0.0.1).
func callbackHandler(expectedState string, codeCh chan<- string, errCh chan<- error) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/callback" {
			http.NotFound(rw, req)
			return
		}
		q := req.URL.Query()
		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			fmt.Fprintln(rw, "Login failed. You can close this tab.")
			select {
			case errCh <- fmt.Errorf("provider returned error=%s: %s", errParam, desc):
			default:
			}
			return
		}
		if q.Get("state") != expectedState {
			fmt.Fprintln(rw, "Login failed (state mismatch). You can close this tab.")
			select {
			case errCh <- errors.New("callback state mismatch — possible CSRF or stale login attempt"):
			default:
			}
			return
		}
		code := q.Get("code")
		if code == "" {
			fmt.Fprintln(rw, "Login failed (no code). You can close this tab.")
			select {
			case errCh <- errors.New("callback missing authorization code"):
			default:
			}
			return
		}
		fmt.Fprintln(rw, "Logged in. You can close this tab and return to the terminal.")
		select {
		case codeCh <- code:
		default:
		}
	})
}

// swiggyRegisterClient performs Dynamic Client Registration (RFC 7591)
// against POST /auth/register. Confirmed live via direct probe during
// research despite the platform changelog listing DCR as a v1.1 roadmap
// item — see the research brief's Reachability Risk section. Registers a
// fresh client on every login; Swiggy's endpoint returned the same
// client_id ("swiggy-mcp") across repeated registrations during probing, so
// this is idempotent in practice even though it is not cached locally.
func swiggyRegisterClient(ctx context.Context) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"client_name":                swiggyDCRClientNameTag,
		"redirect_uris":              []string{"http://" + swiggyLoopbackHost + ":0/callback"},
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, swiggyRegisterURL, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("parsing registration response: %w", err)
	}
	if parsed.ClientID == "" {
		return "", errors.New("registration response had no client_id")
	}
	return parsed.ClientID, nil
}

// exchangeCodeForToken performs the token-endpoint POST using a client that
// refuses cross-origin redirects, per the checklist — never reuse a shared
// API client's CheckRedirect, and never set CheckRedirect to nil.
func exchangeCodeForToken(ctx context.Context, clientID, code, verifier, redirectURI string) (accessToken string, expiresIn int, err error) {
	body, _ := json.Marshal(map[string]any{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"code":          code,
		"code_verifier": verifier,
		"redirect_uri":  redirectURI,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, swiggyTokenURL, strings.NewReader(string(body)))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{
		Timeout: swiggyTokenExchangeTO,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	if resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, fmt.Errorf("parsing token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return "", 0, errors.New("token response had no access_token")
	}
	if parsed.ExpiresIn <= 0 {
		parsed.ExpiresIn = 5 * 24 * 60 * 60 // Swiggy docs: 5-day access tokens.
	}
	return parsed.AccessToken, parsed.ExpiresIn, nil
}

// openBrowser is a minimal per-OS browser launcher. Best-effort: failures
// are surfaced by the caller printing the URL, never by hanging.
func openBrowser(rawURL string) error {
	var cmdName string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmdName, args = "open", []string{rawURL}
	case "windows":
		cmdName, args = "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		cmdName, args = "xdg-open", []string{rawURL}
	}
	// #nosec G204 -- cmdName is a fixed, hardcoded per-OS binary (not derived
	// from input); args is passed as argv (no shell interpretation), and
	// rawURL is this process's own constructed OAuth authorize URL, not
	// externally supplied data.
	return exec.Command(cmdName, args...).Start()
}
