package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	IDToken          string `json:"id_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func main() {
	domain := "dev-n75vbx4hgdk4krai.eu.auth0.com"
	clientID := "jS4Q3DCJlHlkm9djdkxb0GhJ3DD4wA6X"
	audience := "https://k8s-backend"
	scope := "openid profile email"

	deviceResp, err := requestDeviceCode(domain, clientID, audience, scope)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("user_code:", deviceResp.UserCode)
	fmt.Println("verification_uri:", deviceResp.VerificationURI)
	fmt.Println("verification_uri_complete:", deviceResp.VerificationURIComplete)
	fmt.Println()

	_ = openBrowser(deviceResp.VerificationURIComplete)

	tokenResp, err := pollForToken(domain, clientID, deviceResp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("LOGIN OK")
	fmt.Println("access_token:")
	fmt.Println(tokenResp.AccessToken)
	fmt.Println()
	fmt.Println("id_token:")
	fmt.Println(tokenResp.IDToken)
	fmt.Println()
	fmt.Println("expires_in:", tokenResp.ExpiresIn)
	fmt.Println("scope:", tokenResp.Scope)

	// пример вызова твоего backend:
	// callAPI("http://localhost:8080", tokenResp.AccessToken)
}

func requestDeviceCode(domain, clientID, audience, scope string) (*DeviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("audience", audience)
	form.Set("scope", scope)

	req, err := http.NewRequest(
		http.MethodPost,
		"https://"+domain+"/oauth/device/code",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed: %s: %s", res.Status, string(body))
	}

	var out DeviceCodeResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func pollForToken(domain, clientID string, device *DeviceCodeResponse) (*TokenResponse, error) {
	interval := time.Duration(device.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("device_code", device.DeviceCode)
		form.Set("client_id", clientID)

		req, err := http.NewRequest(
			http.MethodPost,
			"https://"+domain+"/oauth/token",
			strings.NewReader(form.Encode()),
		)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, err
		}

		var tok TokenResponse
		if err := json.Unmarshal(body, &tok); err != nil {
			return nil, fmt.Errorf("bad token response: %s", string(body))
		}

		if res.StatusCode == http.StatusOK {
			return &tok, nil
		}

		switch tok.Error {
		case "authorization_pending":
			fmt.Println("waiting for login in browser...")
			continue
		case "slow_down":
			interval += 2 * time.Second
			fmt.Println("slowing down polling to", interval)
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired")
		case "access_denied":
			return nil, fmt.Errorf("user denied access")
		default:
			return nil, fmt.Errorf("token request failed: %s: %s", tok.Error, tok.ErrorDescription)
		}
	}

	return nil, fmt.Errorf("device flow timed out")
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}

	return cmd.Start()
}

func callAPI(baseURL, accessToken string) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/protected", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	fmt.Println("api status:", res.Status)
	fmt.Println(string(body))
}

