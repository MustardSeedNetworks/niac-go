package linklive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type loginResponse struct {
	AccessToken string `json:"accessToken"`
}

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
	claims := map[string]string{"app": linkLiveAppClaim}
	if c.config.OrganizationID != "" {
		claims["org"] = c.config.OrganizationID
	}
	body, err := json.Marshal(map[string]any{"claims": claims})
	if err != nil {
		return "", errors.New("encode Link-Live login request")
	}
	return c.sendLogin(ctx, body)
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
