package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gvm-tools/gvm/internal/platform"
	"github.com/gvm-tools/gvm/internal/ui"
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

// OAuthDeviceFlow performs GitHub's OAuth device authorization flow.
// Returns the access token on success.
func OAuthDeviceFlow(clientID string) (string, error) {
	if clientID == "" {
		return "", fmt.Errorf("GitHub OAuth client ID not configured — set GVM_GITHUB_CLIENT_ID or run 'gvm config set github-client-id <id>'")
	}

	// Step 1: Request device code
	deviceResp, err := requestDeviceCode(clientID)
	if err != nil {
		return "", err
	}

	// Step 2: Display code and open browser
	fmt.Println()
	fmt.Printf("  Go to:     %s\n", ui.Bold(deviceResp.VerificationURI))
	fmt.Printf("  Enter code: %s\n", ui.Bold(deviceResp.UserCode))
	fmt.Println()

	_ = platform.OpenBrowser(deviceResp.VerificationURI)

	// Step 3: Poll for token
	spinner := ui.NewSpinner("Waiting for authorization")

	interval := time.Duration(deviceResp.Interval+1) * time.Second
	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		token, pollErr := pollForToken(clientID, deviceResp.DeviceCode)
		if pollErr != nil {
			if pollErr.Error() == "slow_down" {
				interval += 5 * time.Second
				continue
			}
			if pollErr.Error() == "authorization_pending" {
				continue
			}
			spinner.Stop(false)
			return "", pollErr
		}

		spinner.StopWithMessage(true, "Authorized")
		return token, nil
	}

	spinner.Stop(false)
	return "", fmt.Errorf("authorization timed out — try again with 'gvm login <profile> http'")
}

func requestDeviceCode(clientID string) (*deviceCodeResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"client_id": clientID,
		"scope":     "repo,read:org,user:email",
	})

	req, err := http.NewRequest("POST", "https://github.com/login/device/code", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting device code (check your internet connection): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing device code response: %w", err)
	}

	return &result, nil
}

func pollForToken(clientID, deviceCode string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"client_id":   clientID,
		"device_code": deviceCode,
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
	})

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("polling for token: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("%s", result.Error)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return result.AccessToken, nil
}

// RevokeToken attempts to revoke a GitHub OAuth token using the token itself.
// This uses the token-bearer DELETE endpoint; if the OAuth App requires
// client_secret (which device-flow apps typically don't provide), revocation
// will fail silently. Users can always revoke tokens from GitHub Settings >
// Applications.
func RevokeToken(clientID, token string) {
	if clientID == "" || token == "" {
		return
	}

	body, _ := json.Marshal(map[string]string{
		"access_token": token,
	})

	req, err := http.NewRequest("DELETE",
		fmt.Sprintf("https://api.github.com/applications/%s/grant", clientID),
		bytes.NewReader(body),
	)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
