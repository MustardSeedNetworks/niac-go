package linklive

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type loginResponse struct {
	AccessToken string `json:"accessToken"`
}

const jwtPartCount = 3

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" {
		return c.token, nil
	}
	token, err := c.login(ctx)
	if err == nil {
		c.token = token
	}
	return token, err
}

func (c *Client) login(ctx context.Context) (string, error) {
	body, err := authBody(c.config.OrganizationID)
	if err != nil {
		return "", errors.New("encode Link-Live login request")
	}
	return c.sendLogin(ctx, body)
}

func (c *Client) refreshAccessToken(ctx context.Context, staleToken string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != staleToken {
		return c.token, nil
	}
	organizationID, err := c.refreshOrganization(staleToken)
	if err != nil {
		return "", err
	}
	body, err := authBody(organizationID)
	if err != nil {
		return "", errors.New("encode Link-Live refresh request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.config.IdentityURL+"/v2/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create Link-Live refresh request: %w", err)
	}
	req.Header.Set("Authorization", "Access "+staleToken)
	req.Header.Set("Content-Type", "application/json")
	data, err := c.do(req)
	if err != nil {
		return "", err
	}
	token, err := decodeToken(data)
	if err == nil {
		c.token = token
	}
	return token, err
}

func (c *Client) refreshOrganization(token string) (string, error) {
	if c.config.OrganizationID != "" {
		return c.config.OrganizationID, nil
	}
	parts := strings.Split(token, ".")
	if len(parts) != jwtPartCount {
		return "", errors.New("Link-Live organization ID is required to refresh access token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("Link-Live access token contains an invalid organization claim")
	}
	var claims struct {
		OrganizationID string `json:"org"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.OrganizationID == "" {
		return "", errors.New("Link-Live access token omitted organization claim")
	}
	return claims.OrganizationID, nil
}

func authBody(organizationID string) ([]byte, error) {
	claims := map[string]string{"app": linkLiveAppClaim}
	if organizationID != "" {
		claims["org"] = organizationID
	}
	return json.Marshal(map[string]any{"claims": claims})
}

func (c *Client) sendLogin(ctx context.Context, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.config.IdentityURL+"/v2/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create Link-Live login request: %w", err)
	}
	req.SetBasicAuth(c.config.Username, c.config.Password)
	req.Header.Set("Content-Type", "application/json")
	if c.config.MFACode != "" {
		req.Header.Set("X-Bedrock-Otpcode", c.config.MFACode)
	}
	data, err := c.do(req)
	if err != nil {
		return "", err
	}
	return decodeToken(data)
}

func decodeToken(data []byte) (string, error) {
	var response loginResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", errors.New("Link-Live login returned invalid JSON")
	}
	if response.AccessToken == "" {
		return "", errors.New("Link-Live login response omitted access token")
	}
	return response.AccessToken, nil
}
