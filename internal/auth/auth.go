// Package auth provides OAuth 2.0 device code authentication for Microsoft 365.
// It implements the device authorization grant flow (RFC 8628) using the OAuth v1.0
// endpoint style (resource-based) which provides broader M365 scopes than v2.0.
package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/loosehose/azonk/internal/config"
	"github.com/loosehose/azonk/internal/types"
	"github.com/loosehose/azonk/internal/ui"
	"github.com/fatih/color"
)

// Authenticator handles device code authentication and token management.
type Authenticator struct {
	clientID  string
	resource  string
	tokenFile string
	client    *http.Client
	tokens    *types.TokenResponse
}

// NewAuthenticator creates an Authenticator configured for Microsoft Graph.
func NewAuthenticator(outputDir string) *Authenticator {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		// Non-fatal: we'll fail later if we can't write
	}

	return &Authenticator{
		clientID:  config.MicrosoftOfficeClientID,
		resource:  config.GraphResource,
		tokenFile: outputDir + "/tokens.json",
		client: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// GetTokens returns valid tokens, authenticating or refreshing as needed.
func (a *Authenticator) GetTokens() (*types.TokenResponse, error) {
	// Try cached tokens first
	if err := a.loadTokens(); err == nil {
		if a.isTokenValid() {
			ui.Success("Using cached token (expires %s)", a.tokens.ExpiresAt.Format("15:04:05"))
			return a.tokens, nil
		}

		// Token expired - try refresh
		if a.tokens.RefreshToken != "" {
			ui.Info("Token expired, refreshing...")
			if err := a.refresh(); err == nil {
				return a.tokens, nil
			}
			ui.Error("Refresh failed: %v", err)
		}
	}

	// Need interactive authentication
	return a.deviceCodeAuth()
}

// GetAccessToken is a convenience method that returns just the access token string.
func (a *Authenticator) GetAccessToken() (string, error) {
	tokens, err := a.GetTokens()
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

// =============================================================================
// Device Code Flow
// =============================================================================

// deviceCodeAuth performs the complete device code authentication flow.
func (a *Authenticator) deviceCodeAuth() (*types.TokenResponse, error) {
	ui.Info("Starting device code authentication...")

	// Step 1: Request device code
	deviceCode, err := a.requestDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}

	// Step 2: Display user instructions
	a.displayAuthInstructions(deviceCode)

	// Step 3: Poll for token
	tokens, err := a.pollForToken(deviceCode)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Step 4: Parse expiration and cache
	a.tokens = tokens
	a.parseExpiration()

	if err := a.saveTokens(); err != nil {
		ui.Warning("Could not cache tokens: %v", err)
	}

	ui.Success("Authentication successful!")
	return a.tokens, nil
}

// requestDeviceCode initiates the device code flow by requesting a code from Azure AD.
func (a *Authenticator) requestDeviceCode() (*types.DeviceCodeResponse, error) {
	data := url.Values{
		"client_id": {a.clientID},
		"resource":  {a.resource},
	}

	resp, err := a.postForm(config.DeviceCodeEndpoint, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var deviceCode types.DeviceCodeResponse
	if err := json.Unmarshal(body, &deviceCode); err != nil {
		return nil, fmt.Errorf("parse response: %w (body: %s)", err, string(body))
	}

	return &deviceCode, nil
}

// displayAuthInstructions shows the user how to complete authentication.
func (a *Authenticator) displayAuthInstructions(dc *types.DeviceCodeResponse) {
	fmt.Println()
	color.HiBlack("┌" + strings.Repeat("─", 58) + "┐")
	fmt.Printf("│  To sign in, open a browser and go to:                   │\n")
	fmt.Printf("│  ")
	color.New(color.FgCyan).Printf("%-56s", dc.GetVerificationURL())
	fmt.Printf(" │\n")
	fmt.Printf("│                                                          │\n")
	fmt.Printf("│  Enter the code: ")
	color.New(color.FgGreen, color.Bold).Printf("%-39s", dc.UserCode)
	fmt.Printf(" │\n")
	color.HiBlack("└" + strings.Repeat("─", 58) + "┘")
	fmt.Println()
	ui.Info("Waiting for authentication...")
}

// pollForToken polls the token endpoint until authentication completes or times out.
func (a *Authenticator) pollForToken(dc *types.DeviceCodeResponse) (*types.TokenResponse, error) {
	interval := parseIntOrDefault(dc.Interval, 5)
	expiresIn := parseIntOrDefault(dc.ExpiresIn, 900)
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(interval) * time.Second)

		tokens, err := a.redeemDeviceCode(dc.DeviceCode)
		if err != nil {
			if strings.Contains(err.Error(), "authorization_pending") {
				continue
			}
			return nil, err
		}

		return tokens, nil
	}

	return nil, fmt.Errorf("authentication timed out after %d seconds", expiresIn)
}

// redeemDeviceCode exchanges a device code for access and refresh tokens.
func (a *Authenticator) redeemDeviceCode(deviceCode string) (*types.TokenResponse, error) {
	data := url.Values{
		"client_id":  {a.clientID},
		"code":       {deviceCode},
		"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	resp, err := a.postForm(config.TokenEndpoint, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var errResp struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if json.Unmarshal(body, &errResp); errResp.Error != "" {
		return nil, fmt.Errorf("%s: %s", errResp.Error, errResp.Description)
	}

	var tokens types.TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("parse tokens: %w", err)
	}

	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokens, nil
}

// =============================================================================
// Token Refresh
// =============================================================================

// refresh exchanges the refresh token for a new access token.
func (a *Authenticator) refresh() error {
	data := url.Values{
		"client_id":     {a.clientID},
		"refresh_token": {a.tokens.RefreshToken},
		"grant_type":    {"refresh_token"},
		"resource":      {a.resource},
	}

	resp, err := a.postForm(config.TokenEndpoint, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tokens types.TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return err
	}

	if tokens.AccessToken == "" {
		return fmt.Errorf("no access token in refresh response: %s", string(body))
	}

	a.tokens = &tokens
	a.parseExpiration()

	if err := a.saveTokens(); err != nil {
		ui.Warning("Could not cache refreshed tokens: %v", err)
	}

	ui.Success("Token refreshed")
	return nil
}

// =============================================================================
// Token Persistence
// =============================================================================

func (a *Authenticator) loadTokens() error {
	data, err := os.ReadFile(a.tokenFile)
	if err != nil {
		return err
	}

	var tokens types.TokenResponse
	if err := json.Unmarshal(data, &tokens); err != nil {
		return err
	}

	a.tokens = &tokens
	return nil
}

func (a *Authenticator) saveTokens() error {
	data, err := json.MarshalIndent(a.tokens, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.tokenFile, data, 0600)
}

// =============================================================================
// Helpers
// =============================================================================

func (a *Authenticator) isTokenValid() bool {
	if a.tokens == nil {
		return false
	}
	return time.Now().Before(a.tokens.ExpiresAt.Add(-5 * time.Minute))
}

func (a *Authenticator) parseExpiration() {
	if a.tokens.ExpiresOn != "" {
		var expiresOn int64
		fmt.Sscanf(a.tokens.ExpiresOn, "%d", &expiresOn)
		a.tokens.ExpiresAt = time.Unix(expiresOn, 0)
	} else {
		a.tokens.ExpiresAt = time.Now().Add(time.Hour)
	}
}

func (a *Authenticator) postForm(endpoint string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", config.UserAgent)

	return a.client.Do(req)
}

func parseIntOrDefault(s string, defaultVal int) int {
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil || val <= 0 {
		return defaultVal
	}
	return val
}
